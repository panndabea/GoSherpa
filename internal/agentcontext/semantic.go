package agentcontext

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

type contextSemanticSnapshot struct {
	modulePath     string
	symbols        []sherpa.Symbol
	filesByPackage map[string][]string
	warnings       []string
}

type contextSymbolTarget struct {
	Package  string
	Receiver string
	Name     string
}

func loadContextSemanticSnapshot(root string, buildTags []string) (contextSemanticSnapshot, bool) {
	repo, err := semantics.LoadRepository(root, semantics.LoadOptions{
		BuildTags: buildTags,
	})
	if err != nil {
		return contextSemanticSnapshot{
			warnings: []string{fmt.Sprintf("typechecked context analysis unavailable: %v", err)},
		}, false
	}

	snapshot := contextSemanticSnapshot{
		modulePath:     contextModulePath(repo.Root),
		filesByPackage: make(map[string][]string),
		warnings:       append([]string{}, repo.Warnings...),
	}

	for _, pkg := range repo.Packages {
		files := contextSemanticPackageFiles(repo.Root, pkg)
		if len(files) > 0 {
			snapshot.filesByPackage[pkg.PackagePath] = uniqueSortedStrings(append(snapshot.filesByPackage[pkg.PackagePath], files...))
		}

		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}

			symbols, warnings := contextSymbolsFromFile(repo.Root, file, pkg.PackagePath, snapshot.modulePath)
			snapshot.warnings = append(snapshot.warnings, warnings...)
			snapshot.symbols = append(snapshot.symbols, symbols...)
		}
	}

	sortSymbols(snapshot.symbols)
	snapshot.warnings = uniqueStrings(snapshot.warnings)

	return snapshot, true
}

func (snapshot contextSemanticSnapshot) hasPackage(packagePath string) bool {
	_, ok := snapshot.filesByPackage[packagePath]
	return ok
}

func (snapshot contextSemanticSnapshot) packageFiles(root string, packagePath string) ([]string, []string) {
	files := append([]string{}, snapshot.filesByPackage[packagePath]...)
	testFiles, warnings := contextTestFilesForPackage(root, packagePath)
	files = append(files, testFiles...)

	return uniqueSortedStrings(files), warnings
}

func (snapshot contextSemanticSnapshot) symbolsInFile(root string, file string, packagePath string) ([]sherpa.Symbol, []string) {
	symbols := symbolsInFile(snapshot.symbols, file)
	if !strings.HasSuffix(file, "_test.go") {
		return symbols, nil
	}

	testSymbols, warnings := contextSymbolsFromFile(root, file, packagePath, snapshot.modulePath)
	symbols = append(symbols, testSymbols...)
	sortSymbols(symbols)

	return symbols, warnings
}

func (snapshot contextSemanticSnapshot) symbolsInPackage(root string, packagePath string) ([]sherpa.Symbol, []string) {
	symbols := symbolsInPackage(snapshot.symbols, packagePath)
	testFiles, warnings := contextTestFilesForPackage(root, packagePath)
	for _, file := range testFiles {
		fileSymbols, fileWarnings := contextSymbolsFromFile(root, file, packagePath, snapshot.modulePath)
		warnings = append(warnings, fileWarnings...)
		symbols = append(symbols, fileSymbols...)
	}

	sortSymbols(symbols)

	return symbols, uniqueStrings(warnings)
}

func (snapshot contextSemanticSnapshot) symbol(root string, target string) (sherpa.Symbol, bool, error) {
	parsed, err := parseContextSymbolTarget(root, target, snapshot.modulePath)
	if err != nil {
		return sherpa.Symbol{}, false, err
	}
	if parsed.Name == "" {
		return sherpa.Symbol{}, false, nil
	}

	var matches []sherpa.Symbol
	for _, symbol := range snapshot.symbols {
		if contextSymbolMatchesTarget(symbol, parsed) {
			matches = append(matches, symbol)
		}
	}

	if len(matches) == 0 {
		return sherpa.Symbol{}, false, nil
	}
	if len(matches) > 1 {
		return sherpa.Symbol{}, false, sherpa.NewAmbiguousTargetError("symbol", target, contextSymbolTargetCandidates(matches, snapshot.modulePath))
	}

	return matches[0], true, nil
}

func parseContextSymbolTarget(root string, target string, modulePath string) (contextSymbolTarget, error) {
	value := strings.TrimSpace(filepath.ToSlash(target))
	if value == "" {
		return contextSymbolTarget{}, fmt.Errorf("symbol target is empty")
	}

	packagePath, symbol, err := splitContextPackageQualifiedTarget(root, value, modulePath)
	if err != nil {
		return contextSymbolTarget{}, err
	}

	segments := strings.Split(symbol, ".")
	if len(segments) == 0 {
		return contextSymbolTarget{}, nil
	}

	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(segment) {
			return contextSymbolTarget{}, fmt.Errorf("invalid symbol target: %s", target)
		}
	}

	parsed := contextSymbolTarget{
		Package: packagePath,
		Name:    segments[len(segments)-1],
	}
	if len(segments) >= 2 {
		parsed.Receiver = segments[len(segments)-2]
	}

	return parsed, nil
}

func splitContextPackageQualifiedTarget(root string, target string, modulePath string) (string, string, error) {
	lastSlash := strings.LastIndex(target, "/")
	if lastSlash < 0 {
		return "", target, nil
	}

	firstDotAfterSlash := strings.Index(target[lastSlash+1:], ".")
	if firstDotAfterSlash < 0 {
		return "", target, nil
	}

	separator := lastSlash + 1 + firstDotAfterSlash
	packagePath, err := normalizeContextPackagePath(root, target[:separator], modulePath)
	if err != nil {
		return "", "", err
	}

	return packagePath, target[separator+1:], nil
}

func normalizeContextPackagePath(root string, packagePath string, modulePath string) (string, error) {
	value := strings.TrimSpace(filepath.ToSlash(packagePath))
	if value == "" {
		return "", fmt.Errorf("package path is empty")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("absolute package paths are not supported: %s", packagePath)
	}

	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		modulePath = contextModulePath(root)
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

func contextSymbolMatchesTarget(symbol sherpa.Symbol, target contextSymbolTarget) bool {
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

func contextSymbolTargetCandidates(symbols []sherpa.Symbol, modulePath string) []sherpa.TargetCandidate {
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

func contextSemanticPackageFiles(root string, pkg semantics.Package) []string {
	files := append([]string{}, pkg.GoFiles...)
	if len(files) == 0 {
		files = append(files, pkg.CompiledGoFiles...)
	}

	var result []string
	for _, file := range files {
		relative, ok := contextRelativeFile(root, file)
		if !ok || strings.HasSuffix(relative, "_test.go") || filepath.Ext(relative) != ".go" {
			continue
		}

		result = append(result, relative)
	}

	return uniqueSortedStrings(result)
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

func contextModulePath(root string) string {
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
