package symbolindex

import (
	"fmt"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type LoadOptions struct {
	BuildTags []string
}

type Index struct {
	Root           string
	ModulePath     string
	Symbols        []sherpa.Symbol
	FilesByPackage map[string][]string
	Warnings       []string
}

type symbolTarget struct {
	Package  string
	Receiver string
	Name     string
}

func Load(root string, options LoadOptions) (Index, error) {
	repo, err := semantics.LoadRepository(root, semantics.LoadOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return Index{}, err
	}

	return FromRepository(repo)
}

func FromRepository(repo semantics.Repository) (Index, error) {
	index := Index{
		Root:           repo.Root,
		ModulePath:     modulePath(repo.Root),
		FilesByPackage: make(map[string][]string),
		Warnings:       append([]string{}, repo.Warnings...),
	}

	for _, pkg := range repo.Packages {
		files := semanticPackageFiles(repo.Root, pkg)
		if len(files) > 0 {
			index.FilesByPackage[pkg.PackagePath] = uniqueSortedStrings(append(index.FilesByPackage[pkg.PackagePath], files...))
		}

		for _, file := range files {
			symbols, warnings := symbolsFromFile(repo.Root, file, pkg.PackagePath, index.ModulePath)
			index.Warnings = append(index.Warnings, warnings...)
			index.Symbols = append(index.Symbols, symbols...)
		}
	}

	SortSymbols(index.Symbols)
	index.Warnings = uniqueSortedStrings(index.Warnings)

	return index, nil
}

func (index Index) HasPackage(packagePath string) bool {
	_, ok := index.FilesByPackage[packagePath]
	return ok
}

func (index Index) PackageFiles(packagePath string) []string {
	return append([]string{}, index.FilesByPackage[packagePath]...)
}

func (index Index) SymbolsInFile(file string) []sherpa.Symbol {
	var symbols []sherpa.Symbol
	for _, symbol := range index.Symbols {
		if filepath.ToSlash(symbol.Position.File) == filepath.ToSlash(file) {
			symbols = append(symbols, symbol)
		}
	}

	SortSymbols(symbols)
	return symbols
}

func (index Index) SymbolsInPackage(packagePath string) []sherpa.Symbol {
	var symbols []sherpa.Symbol
	for _, symbol := range index.Symbols {
		if symbol.Package == packagePath {
			symbols = append(symbols, symbol)
		}
	}

	SortSymbols(symbols)
	return symbols
}

func (index Index) FindSymbol(target string) (sherpa.Symbol, bool, error) {
	parsed, err := parseSymbolTarget(index.Root, target, index.ModulePath)
	if err != nil {
		return sherpa.Symbol{}, false, err
	}
	if parsed.Name == "" {
		return sherpa.Symbol{}, false, nil
	}

	var matches []sherpa.Symbol
	for _, symbol := range index.Symbols {
		if symbolMatchesTarget(symbol, parsed) {
			matches = append(matches, symbol)
		}
	}

	if len(matches) == 0 {
		return sherpa.Symbol{}, false, nil
	}
	if len(matches) > 1 {
		return sherpa.Symbol{}, false, sherpa.NewAmbiguousTargetError("symbol", target, symbolTargetCandidates(matches, index.ModulePath))
	}

	return matches[0], true, nil
}

func semanticPackageFiles(root string, pkg semantics.Package) []string {
	files := append([]string{}, pkg.GoFiles...)
	if len(files) == 0 {
		files = append(files, pkg.CompiledGoFiles...)
	}

	var result []string
	for _, file := range files {
		relative, ok := relativeFile(root, file)
		if !ok || strings.HasSuffix(relative, "_test.go") || filepath.Ext(relative) != ".go" {
			continue
		}

		result = append(result, relative)
	}

	return uniqueSortedStrings(result)
}

func symbolsFromFile(root string, file string, packagePath string, modulePath string) ([]sherpa.Symbol, []string) {
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
		symbols[i].Position = positionRelativeToRoot(rootPath, symbols[i].Position)
		if symbols[i].Range != nil {
			symbols[i].Range.Start = positionRelativeToRoot(rootPath, symbols[i].Range.Start)
			symbols[i].Range.End = positionRelativeToRoot(rootPath, symbols[i].Range.End)
		}
	}

	SortSymbols(symbols)
	return symbols, nil
}

func parseSymbolTarget(root string, target string, modulePath string) (symbolTarget, error) {
	value := strings.TrimSpace(filepath.ToSlash(target))
	if value == "" {
		return symbolTarget{}, fmt.Errorf("symbol target is empty")
	}

	packagePath, symbol, err := splitPackageQualifiedTarget(root, value, modulePath)
	if err != nil {
		return symbolTarget{}, err
	}

	segments := strings.Split(symbol, ".")
	if len(segments) != 1 && len(segments) != 2 {
		return symbolTarget{}, fmt.Errorf("invalid symbol target: %s", target)
	}
	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(segment) {
			return symbolTarget{}, fmt.Errorf("invalid symbol target: %s", target)
		}
	}

	parsed := symbolTarget{
		Package: packagePath,
		Name:    segments[len(segments)-1],
	}
	if len(segments) == 2 {
		parsed.Receiver = segments[0]
	}

	return parsed, nil
}

func splitPackageQualifiedTarget(root string, target string, modulePath string) (string, string, error) {
	lastSlash := strings.LastIndex(target, "/")
	if lastSlash < 0 {
		return "", target, nil
	}

	firstDotAfterSlash := strings.Index(target[lastSlash+1:], ".")
	if firstDotAfterSlash < 0 {
		return "", target, nil
	}

	separator := lastSlash + 1 + firstDotAfterSlash
	packagePath, err := normalizePackagePath(root, target[:separator], modulePath)
	if err != nil {
		return "", "", err
	}

	return packagePath, target[separator+1:], nil
}

func normalizePackagePath(root string, packagePath string, modulePath string) (string, error) {
	value := strings.TrimSpace(filepath.ToSlash(packagePath))
	if value == "" {
		return "", fmt.Errorf("package path is empty")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("absolute package paths are not supported: %s", packagePath)
	}

	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		modulePath = modulePathFromRoot(root)
	}
	if modulePath != "" {
		if value == modulePath {
			return ".", nil
		}
		if strings.HasPrefix(value, modulePath+"/") {
			value = strings.TrimPrefix(value, modulePath+"/")
		} else if !strings.HasPrefix(value, "./") && strings.Contains(value, ".") {
			return "", fmt.Errorf("non-local package-qualified symbol targets are not supported: %s", packagePath)
		}
	} else if !strings.HasPrefix(value, "./") && strings.Contains(value, ".") {
		return "", fmt.Errorf("module path is required for package-qualified symbol target: %s", packagePath)
	}

	value = strings.TrimPrefix(value, "./")
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", fmt.Errorf("package path must not contain '..': %s", packagePath)
		}
	}

	cleaned := path.Clean(value)
	if cleaned == "." {
		return ".", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("package path escapes repository: %s", packagePath)
	}

	return "./" + cleaned, nil
}

func symbolMatchesTarget(symbol sherpa.Symbol, target symbolTarget) bool {
	if target.Package != "" && symbol.Package != target.Package {
		return false
	}

	if target.Receiver != "" {
		return symbol.Receiver == target.Receiver && symbol.Name == target.Name
	}
	if target.Package != "" {
		return symbol.Receiver == "" && symbol.Name == target.Name
	}

	return symbol.Name == target.Name
}

func symbolTargetCandidates(symbols []sherpa.Symbol, modulePath string) []sherpa.TargetCandidate {
	candidates := make([]sherpa.TargetCandidate, 0, len(symbols))
	for _, symbol := range symbols {
		target := symbol.DisplayName()
		candidates = append(candidates, sherpa.TargetCandidate{
			Package:  symbol.Package,
			Symbol:   target,
			Position: symbol.Position,
			Example:  sherpa.FormatPackageQualifiedTarget(symbol.Package, target, modulePath),
		})
	}

	return candidates
}

func modulePath(root string) string {
	value, err := sherpa.ModulePath(root)
	if err == nil {
		return value
	}

	return modulePathFromRoot(root)
}

func modulePathFromRoot(root string) string {
	contents, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1]
		}
	}

	return ""
}

func relativeFile(root string, file string) (string, bool) {
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

func positionRelativeToRoot(root string, position sherpa.Position) sherpa.Position {
	relative, ok := relativeFile(root, position.File)
	if !ok {
		position.File = filepath.ToSlash(position.File)
		return position
	}

	position.File = relative
	return position
}

func SortSymbols(symbols []sherpa.Symbol) {
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
