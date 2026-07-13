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
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type changedSymbolRange struct {
	Name     string
	Position sherpa.Position
	Range    *sherpa.SourceRange
	Start    int
	End      int
}

type ChangedSymbol struct {
	Package  string              `json:"package"`
	Name     string              `json:"name"`
	Target   string              `json:"target,omitempty"`
	Position sherpa.Position     `json:"position"`
	Range    *sherpa.SourceRange `json:"range,omitempty"`
	Deleted  bool                `json:"deleted,omitempty"`
}

type changedSymbol = ChangedSymbol

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
	return symbolsForChangedLineRangesWithCurrentSymbols(root, base, head, changedLines, nil, false)
}

func symbolsForChangedLineRangesWithCurrentSymbols(root string, base string, head string, changedLines []gitdiff.ChangedFileLineRanges, snapshotSymbols []sherpa.Symbol, useSnapshotSymbols bool) ([]changedSymbol, error) {
	var symbols []changedSymbol
	useSnapshotForCurrentFiles := useSnapshotSymbols && strings.TrimSpace(head) == ""

	for _, changedFile := range changedLines {
		if !changedFileHasGoPath(changedFile) {
			continue
		}

		var currentSymbols []changedSymbol
		var err error
		if useSnapshotForCurrentFiles {
			currentSymbols, err = changedSymbolsForCurrentFileFromSnapshot(root, changedFile, snapshotSymbols)
		} else {
			currentSymbols, err = changedSymbolsForCurrentFile(root, head, changedFile)
		}
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

func changedSymbolsForCurrentFileFromSnapshot(root string, changedFile gitdiff.ChangedFileLineRanges, snapshotSymbols []sherpa.Symbol) ([]changedSymbol, error) {
	if len(changedFile.Ranges) == 0 {
		return nil, nil
	}
	packagePath, ok := packageForChangedFile(changedFile.Path)
	if !ok {
		return nil, nil
	}

	symbols := changedSymbolRangesFromSnapshot(changedFile.Path, snapshotSymbols)
	return changedSymbolRecords(root, packagePath, changedSymbolsForRanges(symbols, changedFile.Ranges), false), nil
}

func changedSymbolRangesFromSnapshot(file string, symbols []sherpa.Symbol) []changedSymbolRange {
	var ranges []changedSymbolRange
	for _, symbol := range symbols {
		if !snapshotSymbolHasChangedSymbolKind(symbol) {
			continue
		}
		if filepath.ToSlash(symbol.Position.File) != filepath.ToSlash(file) {
			continue
		}

		symbolRange, ok := changedSymbolRangeFromSnapshotSymbol(symbol)
		if !ok {
			continue
		}
		ranges = append(ranges, symbolRange)
	}

	return ranges
}

func snapshotSymbolHasChangedSymbolKind(symbol sherpa.Symbol) bool {
	switch symbol.Kind {
	case sherpa.SymbolKindFunction, sherpa.SymbolKindMethod, sherpa.SymbolKindStruct, sherpa.SymbolKindInterface:
		return true
	default:
		return false
	}
}

func changedSymbolRangeFromSnapshotSymbol(symbol sherpa.Symbol) (changedSymbolRange, bool) {
	name := strings.TrimSpace(symbol.DisplayName())
	if name == "" || symbol.Position.Line <= 0 {
		return changedSymbolRange{}, false
	}

	sourceRange := symbol.Range
	if sourceRange == nil {
		sourceRange = &sherpa.SourceRange{
			Start: symbol.Position,
			End:   symbol.Position,
		}
	}

	start := sourceRange.Start.Line
	end := sourceRange.End.Line
	if start <= 0 {
		start = symbol.Position.Line
	}
	if end <= 0 {
		end = start
	}

	return changedSymbolRange{
		Name:     name,
		Position: symbol.Position,
		Range:    sourceRange,
		Start:    start,
		End:      end,
	}, true
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

	return changedSymbolRecords(root, packagePath, changedSymbolsForRanges(symbols, changedFile.Ranges), false), nil
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

	return changedSymbolRecords(root, packagePath, changedSymbolsForRanges(symbols, changedFile.OldRanges), true), nil
}

func changedSymbolsForRanges(symbols []changedSymbolRange, ranges []gitdiff.ChangedLineRange) []changedSymbolRange {
	var changedSymbols []changedSymbolRange
	for _, symbol := range symbols {
		for _, lineRange := range ranges {
			if !lineRangesOverlap(symbol.Start, symbol.End, lineRange.Start, lineRange.End) {
				continue
			}

			changedSymbols = append(changedSymbols, symbol)
			break
		}
	}

	return changedSymbols
}

func changedSymbolRecords(root string, packagePath string, symbols []changedSymbolRange, deleted bool) []changedSymbol {
	records := make([]changedSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		name := strings.TrimSpace(symbol.Name)
		if name == "" {
			continue
		}

		records = append(records, changedSymbol{
			Package:  packagePath,
			Name:     name,
			Position: changedSymbolPositionRelativeToRoot(root, symbol.Position),
			Range:    changedSymbolRangeRelativeToRoot(root, symbol.Range),
			Deleted:  deleted,
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
		symbol.Target = strings.TrimSpace(symbol.Target)
		if symbol.Package == "" || symbol.Name == "" {
			continue
		}

		key := symbol.Package + "\x00" + symbol.Name
		existing, ok := seen[key]
		if !ok || preferChangedSymbol(symbol, existing) {
			seen[key] = symbol
			continue
		}

		seen[key] = mergeChangedSymbol(existing, symbol)
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

func preferChangedSymbol(candidate changedSymbol, existing changedSymbol) bool {
	if existing.Deleted && !candidate.Deleted {
		return true
	}
	if existing.Position.File == "" && candidate.Position.File != "" {
		return true
	}
	if existing.Range == nil && candidate.Range != nil {
		return true
	}

	return false
}

func mergeChangedSymbol(existing changedSymbol, candidate changedSymbol) changedSymbol {
	if existing.Target == "" {
		existing.Target = candidate.Target
	}
	if existing.Position.File == "" {
		existing.Position = candidate.Position
	}
	if existing.Range == nil {
		existing.Range = candidate.Range
	}
	existing.Deleted = existing.Deleted && candidate.Deleted

	return existing
}

func changedSymbolsWithTargets(symbols []changedSymbol, modulePath string) []ChangedSymbol {
	normalized := normalizeChangedSymbols(symbols)
	result := make([]ChangedSymbol, 0, len(normalized))
	for _, symbol := range normalized {
		symbol.Target = changedSymbolTestTarget(symbol, modulePath)
		result = append(result, symbol)
	}

	return result
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

			symbols = append(symbols, newChangedSymbolRange(fileSet, decl.Pos(), decl.End(), name))
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

				symbols = append(symbols, newChangedSymbolRange(fileSet, typeSpec.Pos(), typeSpec.End(), typeSpec.Name.Name))
			}
		}
	}

	return symbols, nil
}

func newChangedSymbolRange(fileSet *token.FileSet, start token.Pos, end token.Pos, name string) changedSymbolRange {
	startPosition := fileSet.Position(start)
	endPosition := fileSet.Position(end)

	return changedSymbolRange{
		Name: strings.TrimSpace(name),
		Position: sherpa.Position{
			File:   startPosition.Filename,
			Line:   startPosition.Line,
			Column: startPosition.Column,
		},
		Range: &sherpa.SourceRange{
			Start: sherpa.Position{
				File:   startPosition.Filename,
				Line:   startPosition.Line,
				Column: startPosition.Column,
			},
			End: sherpa.Position{
				File:   endPosition.Filename,
				Line:   endPosition.Line,
				Column: endPosition.Column,
			},
		},
		Start: startPosition.Line,
		End:   endPosition.Line,
	}
}

func changedSymbolRangeRelativeToRoot(root string, sourceRange *sherpa.SourceRange) *sherpa.SourceRange {
	if sourceRange == nil {
		return nil
	}

	return &sherpa.SourceRange{
		Start: changedSymbolPositionRelativeToRoot(root, sourceRange.Start),
		End:   changedSymbolPositionRelativeToRoot(root, sourceRange.End),
	}
}

func changedSymbolPositionRelativeToRoot(root string, position sherpa.Position) sherpa.Position {
	if position.File == "" {
		return position
	}

	position.File = filepath.ToSlash(position.File)
	root = strings.TrimSpace(root)
	if root == "" {
		return position
	}

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return position
	}
	rootPath = filepath.Clean(rootPath)

	filePath := filepath.FromSlash(position.File)
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(rootPath, filePath)
	}
	filePath = filepath.Clean(filePath)

	relative, err := filepath.Rel(rootPath, filePath)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		position.File = filepath.ToSlash(filePath)
		return position
	}

	position.File = filepath.ToSlash(relative)
	return position
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
