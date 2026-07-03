package impact

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitdiff "github.com/panndabea/GoSherpa/internal/git"
)

type changedSymbolRange struct {
	Name  string
	Start int
	End   int
}

type changedSymbol struct {
	Package string
	Name    string
}

func ChangedSymbols(root string, base string, head string) ([]string, error) {
	changedLines, err := gitdiff.ChangedLineRanges(root, base, head)
	if err != nil {
		return nil, err
	}

	symbols, err := symbolsForChangedLineRanges(root, base, head, changedLines)
	if err != nil {
		return nil, err
	}

	return changedSymbolNames(symbols), nil
}

func symbolsForChangedLineRanges(root string, base string, head string, changedLines []gitdiff.ChangedFileLineRanges) ([]changedSymbol, error) {
	var symbols []changedSymbol

	for _, changedFile := range changedLines {
		if !changedFileHasGoPath(changedFile) {
			continue
		}

		currentSymbols, err := changedSymbolsForCurrentFile(root, head, changedFile)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, currentSymbols...)

		baseSymbols, err := changedSymbolsForBaseFile(root, base, changedFile)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, baseSymbols...)
	}

	return normalizeChangedSymbols(symbols), nil
}

func changedFileHasGoPath(changedFile gitdiff.ChangedFileLineRanges) bool {
	if _, ok := packageForChangedFile(changedFile.Path); ok {
		return true
	}

	if _, ok := packageForChangedFile(changedFile.OldPath); ok {
		return true
	}

	return false
}

func changedSymbolsForCurrentFile(root string, head string, changedFile gitdiff.ChangedFileLineRanges) ([]changedSymbol, error) {
	if len(changedFile.Ranges) == 0 {
		return nil, nil
	}
	packagePath, ok := packageForChangedFile(changedFile.Path)
	if !ok {
		return nil, nil
	}

	filePath := filepath.Join(root, filepath.FromSlash(changedFile.Path))
	var symbols []changedSymbolRange
	var err error
	if strings.TrimSpace(head) == "" {
		symbols, err = parseChangedSymbolRanges(filePath)
	} else {
		var source []byte
		source, err = gitdiff.FileAtRef(root, head, changedFile.Path)
		if err == nil {
			symbols, err = parseChangedSymbolRangesFromSource(filePath, source)
		}
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	return changedSymbolRecords(packagePath, changedSymbolsForRanges(symbols, changedFile.Ranges)), nil
}

func changedSymbolsForBaseFile(root string, base string, changedFile gitdiff.ChangedFileLineRanges) ([]changedSymbol, error) {
	if len(changedFile.OldRanges) == 0 {
		return nil, nil
	}

	oldPath := changedFile.OldPath
	if oldPath == "" {
		oldPath = changedFile.Path
	}
	packagePath, ok := packageForChangedFile(oldPath)
	if !ok {
		return nil, nil
	}

	source, err := gitdiff.FileAtRef(root, base, oldPath)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(root, filepath.FromSlash(oldPath))
	symbols, err := parseChangedSymbolRangesFromSource(filePath, source)
	if err != nil {
		return nil, err
	}

	return changedSymbolRecords(packagePath, changedSymbolsForRanges(symbols, changedFile.OldRanges)), nil
}

func changedSymbolsForRanges(symbols []changedSymbolRange, ranges []gitdiff.ChangedLineRange) []string {
	var changedSymbols []string
	for _, symbol := range symbols {
		for _, lineRange := range ranges {
			if !lineRangesOverlap(symbol.Start, symbol.End, lineRange.Start, lineRange.End) {
				continue
			}

			changedSymbols = append(changedSymbols, symbol.Name)
			break
		}
	}

	return changedSymbols
}

func changedSymbolRecords(packagePath string, names []string) []changedSymbol {
	records := make([]changedSymbol, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		records = append(records, changedSymbol{
			Package: packagePath,
			Name:    name,
		})
	}

	return records
}

func changedSymbolNames(symbols []changedSymbol) []string {
	names := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		names = append(names, symbol.Name)
	}

	return uniqueSortedStrings(names)
}

func normalizeChangedSymbols(symbols []changedSymbol) []changedSymbol {
	seen := make(map[string]changedSymbol)
	for _, symbol := range symbols {
		symbol.Package = strings.TrimSpace(symbol.Package)
		symbol.Name = strings.TrimSpace(symbol.Name)
		if symbol.Package == "" || symbol.Name == "" {
			continue
		}

		seen[symbol.Package+"\x00"+symbol.Name] = symbol
	}

	normalized := make([]changedSymbol, 0, len(seen))
	for _, symbol := range seen {
		normalized = append(normalized, symbol)
	}

	sort.Slice(normalized, func(i int, j int) bool {
		if normalized[i].Package != normalized[j].Package {
			return normalized[i].Package < normalized[j].Package
		}

		return normalized[i].Name < normalized[j].Name
	})

	return normalized
}

func parseChangedSymbolRanges(filePath string) ([]changedSymbolRange, error) {
	return parseChangedSymbolRangesFromSource(filePath, nil)
}

func parseChangedSymbolRangesFromSource(filePath string, source any) ([]changedSymbolRange, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filePath, source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse changed symbols in %s: %w", filePath, err)
	}

	var symbols []changedSymbolRange
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			name := decl.Name.Name
			if decl.Recv != nil {
				receiver := changedSymbolReceiverName(decl)
				if receiver != "" {
					name = receiver + "." + name
				}
			}

			symbols = append(symbols, changedSymbolRange{
				Name:  name,
				Start: fileSet.Position(decl.Pos()).Line,
				End:   fileSet.Position(decl.End()).Line,
			})
		case *ast.GenDecl:
			if decl.Tok != token.TYPE {
				continue
			}

			for _, spec := range decl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if !isChangedSymbolType(typeSpec.Type) {
					continue
				}

				symbols = append(symbols, changedSymbolRange{
					Name:  typeSpec.Name.Name,
					Start: fileSet.Position(typeSpec.Pos()).Line,
					End:   fileSet.Position(typeSpec.End()).Line,
				})
			}
		}
	}

	return symbols, nil
}

func isChangedSymbolType(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.StructType, *ast.InterfaceType:
		return true
	default:
		return false
	}
}

func changedSymbolReceiverName(funcDecl *ast.FuncDecl) string {
	field := funcDecl.Recv.List[0]

	ident, ok := field.Type.(*ast.Ident)
	if ok {
		return ident.Name
	}

	starExpr, ok := field.Type.(*ast.StarExpr)
	if ok {
		ident, ok := starExpr.X.(*ast.Ident)
		if ok {
			return ident.Name
		}
	}

	return ""
}

func lineRangesOverlap(leftStart int, leftEnd int, rightStart int, rightEnd int) bool {
	if leftEnd < leftStart {
		leftEnd = leftStart
	}
	if rightEnd < rightStart {
		rightEnd = rightStart
	}

	return leftStart <= rightEnd && rightStart <= leftEnd
}
