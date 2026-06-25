package impact

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

	gitdiff "github.com/supertabaluga/gosherpa/internal/git"
)

type changedSymbolRange struct {
	Name  string
	Start int
	End   int
}

func ChangedSymbols(root string, base string, head string) ([]string, error) {
	changedLines, err := gitdiff.ChangedLineRanges(root, base, head)
	if err != nil {
		return nil, err
	}

	return symbolsForChangedLineRanges(root, changedLines)
}

func symbolsForChangedLineRanges(root string, changedLines []gitdiff.ChangedFileLineRanges) ([]string, error) {
	var symbols []string

	for _, changedFile := range changedLines {
		if _, ok := packageForChangedFile(changedFile.Path); !ok {
			continue
		}

		fileSymbols, err := changedSymbolsForFile(root, changedFile)
		if err != nil {
			return nil, err
		}

		symbols = append(symbols, fileSymbols...)
	}

	return uniqueSortedStrings(symbols), nil
}

func changedSymbolsForFile(root string, changedFile gitdiff.ChangedFileLineRanges) ([]string, error) {
	filePath := filepath.Join(root, filepath.FromSlash(changedFile.Path))
	symbols, err := parseChangedSymbolRanges(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var changedSymbols []string
	for _, symbol := range symbols {
		for _, lineRange := range changedFile.Ranges {
			if !lineRangesOverlap(symbol.Start, symbol.End, lineRange.Start, lineRange.End) {
				continue
			}

			changedSymbols = append(changedSymbols, symbol.Name)
			break
		}
	}

	return changedSymbols, nil
}

func parseChangedSymbolRanges(filePath string) ([]changedSymbolRange, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filePath, nil, 0)
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
