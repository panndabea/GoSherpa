package agentcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
	"github.com/panndabea/GoSherpa/internal/symbolindex"
)

type contextSemanticSnapshot struct {
	index    symbolindex.RepositoryIndex
	warnings []string
}

func loadContextSemanticSnapshotWithContext(context *sherpa.SemanticContext) (contextSemanticSnapshot, bool) {
	if context == nil {
		return contextSemanticSnapshot{
			warnings: []string{"typechecked context analysis unavailable: semantic context is nil"},
		}, false
	}

	repo, attempted, err := context.TypecheckedRepository()
	if !attempted {
		return contextSemanticSnapshot{}, false
	}
	if err != nil {
		return contextSemanticSnapshot{
			warnings: []string{fmt.Sprintf("typechecked context analysis unavailable: %v", err)},
		}, false
	}

	index, err := symbolindex.FromRepository(repo)
	if err != nil {
		return contextSemanticSnapshot{
			warnings: []string{fmt.Sprintf("typechecked context analysis unavailable: %v", err)},
		}, false
	}

	snapshot := contextSemanticSnapshot{
		index:    index,
		warnings: append([]string{}, index.Warnings...),
	}
	snapshot.warnings = uniqueStrings(snapshot.warnings)

	return snapshot, true
}

func (snapshot contextSemanticSnapshot) hasPackage(packagePath string) bool {
	return snapshot.index.HasPackage(packagePath)
}

func (snapshot contextSemanticSnapshot) packageFiles(root string, packagePath string) ([]string, []string) {
	files := snapshot.index.PackageFiles(packagePath)
	testFiles, warnings := contextTestFilesForPackage(root, packagePath)
	files = append(files, testFiles...)

	return uniqueSortedStrings(files), warnings
}

func (snapshot contextSemanticSnapshot) symbolsInFile(root string, file string, packagePath string) ([]sherpa.Symbol, []string) {
	symbols := snapshot.index.SymbolsInFile(file)
	if !strings.HasSuffix(file, "_test.go") {
		return symbols, nil
	}

	testSymbols, warnings := contextSymbolsFromFile(root, file, packagePath, snapshot.index.ModulePath)
	symbols = append(symbols, testSymbols...)
	sortSymbols(symbols)

	return symbols, warnings
}

func (snapshot contextSemanticSnapshot) symbolsInPackage(root string, packagePath string) ([]sherpa.Symbol, []string) {
	symbols := snapshot.index.SymbolsInPackage(packagePath)
	testFiles, warnings := contextTestFilesForPackage(root, packagePath)
	for _, file := range testFiles {
		fileSymbols, fileWarnings := contextSymbolsFromFile(root, file, packagePath, snapshot.index.ModulePath)
		warnings = append(warnings, fileWarnings...)
		symbols = append(symbols, fileSymbols...)
	}

	sortSymbols(symbols)

	return symbols, uniqueStrings(warnings)
}

func contextTestFilesForPackage(root string, packagePath string) ([]string, []string) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return nil, []string{fmt.Sprintf("resolve repository root %s: %v", root, err)}
	}

	dir := packageDirectory(packagePath)
	dirPath := rootPath
	if dir != "." {
		dirPath = filepath.Join(rootPath, filepath.FromSlash(dir))
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, []string{fmt.Sprintf("read package directory %s: %v", packagePath, err)}
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		if dir == "." {
			files = append(files, entry.Name())
			continue
		}

		files = append(files, filepath.ToSlash(filepath.Join(dir, entry.Name())))
	}

	sort.Strings(files)

	return files, nil
}

func contextSymbolsFromFile(root string, file string, packagePath string, modulePath string) ([]sherpa.Symbol, []string) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return nil, []string{fmt.Sprintf("resolve repository root %s: %v", root, err)}
	}

	absoluteFile := filepath.Join(rootPath, filepath.FromSlash(file))
	symbols, err := sherpa.ParseFile(absoluteFile)
	if err != nil {
		return nil, []string{fmt.Sprintf("parse %s: %v", file, err)}
	}

	for i := range symbols {
		symbols[i].Package = packagePath
		symbols[i].QualifiedName = sherpa.FormatPackageQualifiedTarget(packagePath, symbols[i].DisplayName(), modulePath)
		symbols[i].Position = contextPositionRelativeToRoot(rootPath, symbols[i].Position)
		if symbols[i].Range != nil {
			symbols[i].Range.Start = contextPositionRelativeToRoot(rootPath, symbols[i].Range.Start)
			symbols[i].Range.End = contextPositionRelativeToRoot(rootPath, symbols[i].Range.End)
		}
	}

	sortSymbols(symbols)

	return symbols, nil
}

func contextRelativeFile(root string, file string) (string, bool) {
	if strings.TrimSpace(file) == "" {
		return "", false
	}

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}

	filePath := file
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(rootPath, filePath)
	}
	filePath = filepath.Clean(filePath)

	relative, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		return "", false
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(relative), true
}

func contextPositionRelativeToRoot(root string, position sherpa.Position) sherpa.Position {
	relative, ok := contextRelativeFile(root, position.File)
	if !ok {
		position.File = filepath.ToSlash(position.File)
		return position
	}

	position.File = relative
	return position
}

func sortSymbols(symbols []sherpa.Symbol) {
	sort.SliceStable(symbols, func(i int, j int) bool {
		if symbols[i].Package != symbols[j].Package {
			return symbols[i].Package < symbols[j].Package
		}
		if symbols[i].Position.File != symbols[j].Position.File {
			return symbols[i].Position.File < symbols[j].Position.File
		}
		if symbols[i].Position.Line != symbols[j].Position.Line {
			return symbols[i].Position.Line < symbols[j].Position.Line
		}
		if symbols[i].Position.Column != symbols[j].Position.Column {
			return symbols[i].Position.Column < symbols[j].Position.Column
		}

		return symbols[i].DisplayName() < symbols[j].DisplayName()
	})
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		seen[value] = struct{}{}
	}

	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}
