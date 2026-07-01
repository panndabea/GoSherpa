package semantics

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type LoadOptions struct {
	IncludeTests bool
	BuildFlags   []string
	Patterns     []string
}

type Repository struct {
	Root     string
	Packages []Package
	Warnings []string
}

type Package struct {
	ID              string
	Name            string
	PackagePath     string
	ImportPath      string
	Dir             string
	GoFiles         []string
	CompiledGoFiles []string
	FileSet         *token.FileSet
	Files           []*ast.File
	Types           *types.Package
	TypesInfo       *types.Info
}

func LoadRepository(root string, options LoadOptions) (Repository, error) {
	rootPath, err := absoluteRoot(root)
	if err != nil {
		return Repository{}, err
	}

	patterns := options.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Dir:        rootPath,
		Tests:      options.IncludeTests,
		BuildFlags: append([]string{}, options.BuildFlags...),
	}

	loaded, err := packages.Load(cfg, patterns...)
	if err != nil {
		return Repository{}, fmt.Errorf("load packages: %w", err)
	}
	if len(loaded) == 0 {
		return Repository{}, fmt.Errorf("load packages: no packages matched %s", strings.Join(patterns, ", "))
	}

	repo := Repository{Root: rootPath}
	for _, pkg := range loaded {
		semanticPackage, ok, err := packageFromLoaded(rootPath, pkg)
		if err != nil {
			return Repository{}, err
		}
		if !ok {
			continue
		}

		repo.Packages = append(repo.Packages, semanticPackage)
		repo.Warnings = append(repo.Warnings, packageWarnings(rootPath, pkg)...)
	}

	if len(repo.Packages) == 0 {
		return Repository{}, fmt.Errorf("load packages: no repository-local packages matched %s", strings.Join(patterns, ", "))
	}

	sort.Slice(repo.Packages, func(i int, j int) bool {
		if repo.Packages[i].PackagePath != repo.Packages[j].PackagePath {
			return repo.Packages[i].PackagePath < repo.Packages[j].PackagePath
		}
		return repo.Packages[i].ID < repo.Packages[j].ID
	})
	repo.Warnings = uniqueSorted(repo.Warnings)

	return repo, nil
}

func packageFromLoaded(root string, pkg *packages.Package) (Package, bool, error) {
	if pkg == nil {
		return Package{}, false, nil
	}

	dir := strings.TrimSpace(pkg.Dir)
	if dir == "" {
		dir = packageDir(pkg)
	}
	if dir == "" {
		return Package{}, false, nil
	}

	dir = filepath.Clean(dir)
	if !pathInsideRoot(root, dir) {
		return Package{}, false, nil
	}

	packagePath, err := packagePathForDir(root, dir)
	if err != nil {
		return Package{}, false, err
	}

	return Package{
		ID:              pkg.ID,
		Name:            pkg.Name,
		PackagePath:     packagePath,
		ImportPath:      pkg.PkgPath,
		Dir:             dir,
		GoFiles:         normalizePaths(pkg.GoFiles),
		CompiledGoFiles: normalizePaths(pkg.CompiledGoFiles),
		FileSet:         pkg.Fset,
		Files:           pkg.Syntax,
		Types:           pkg.Types,
		TypesInfo:       pkg.TypesInfo,
	}, true, nil
}

func packageDir(pkg *packages.Package) string {
	for _, file := range pkg.CompiledGoFiles {
		if strings.TrimSpace(file) != "" {
			return filepath.Dir(file)
		}
	}
	for _, file := range pkg.GoFiles {
		if strings.TrimSpace(file) != "" {
			return filepath.Dir(file)
		}
	}

	return ""
}

func packageWarnings(root string, pkg *packages.Package) []string {
	if pkg == nil {
		return nil
	}

	label := packageLabel(pkg)
	var warnings []string
	for _, packageErr := range pkg.Errors {
		message := packageErr.Error()
		if !packageLoadWarningIsActionable(pkg, message) {
			continue
		}

		warnings = append(warnings, fmt.Sprintf("package load warning: %s: %s", label, relativePackageError(root, message)))
	}
	for _, typeErr := range pkg.TypeErrors {
		warnings = append(warnings, fmt.Sprintf("package load warning: %s: %s", label, relativePackageError(root, typeErr.Error())))
	}

	return warnings
}

func packageLoadWarningIsActionable(pkg *packages.Package, message string) bool {
	if packageLoadWarningIsTransientCacheMiss(message) && packageHasUsableSemanticData(pkg) {
		return false
	}

	return true
}

func packageLoadWarningIsTransientCacheMiss(message string) bool {
	value := strings.ToLower(filepath.ToSlash(message))

	return strings.Contains(value, "loading compiled go files from cache") &&
		strings.Contains(value, "reading srcfiles list") &&
		strings.Contains(value, "cache entry not found")
}

func packageHasUsableSemanticData(pkg *packages.Package) bool {
	return pkg != nil &&
		len(pkg.Syntax) > 0 &&
		pkg.Types != nil &&
		pkg.TypesInfo != nil
}

func packageLabel(pkg *packages.Package) string {
	if pkg.PkgPath != "" {
		return pkg.PkgPath
	}
	if pkg.ID != "" {
		return pkg.ID
	}

	return "unknown"
}

func relativePackageError(root string, value string) string {
	root = filepath.ToSlash(filepath.Clean(root))
	if root == "." || root == "" {
		return filepath.ToSlash(value)
	}

	return strings.ReplaceAll(filepath.ToSlash(value), root+"/", "")
}

func absoluteRoot(root string) (string, error) {
	value := strings.TrimSpace(root)
	if value == "" {
		return "", fmt.Errorf("repository root is empty")
	}

	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve repository root %s: %w", value, err)
	}

	return filepath.Clean(absolute), nil
}

func packagePathForDir(root string, dir string) (string, error) {
	relativePath, err := filepath.Rel(root, dir)
	if err != nil {
		return "", err
	}

	relativePath = filepath.ToSlash(relativePath)
	if relativePath == "." {
		return ".", nil
	}

	return "./" + relativePath, nil
}

func pathInsideRoot(root string, path string) bool {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	return relativePath == "." || (relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator)))
}

func normalizePaths(paths []string) []string {
	values := append([]string{}, paths...)
	for i := range values {
		values[i] = filepath.Clean(values[i])
	}
	sort.Strings(values)

	return values
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	sort.Strings(values)
	result := values[:0]
	var last string
	for index, value := range values {
		if index > 0 && value == last {
			continue
		}
		result = append(result, value)
		last = value
	}

	return result
}
