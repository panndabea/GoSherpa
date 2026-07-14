package semantics

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
)

type LoadOptions struct {
	IncludeTests bool
	BuildTags    []string
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

var packageLoader = packages.Load

var repositoryLoadCache = newRepositoryCache(16)

func LoadRepository(root string, options LoadOptions) (Repository, error) {
	rootPath, err := absoluteRoot(root)
	if err != nil {
		return Repository{}, err
	}

	patterns := packageLoadPatterns(rootPath, options)
	cacheKey := repositoryCacheKey(rootPath, options, patterns)
	fingerprint, fingerprintErr := repositoryInputFingerprint(rootPath, options)
	if fingerprintErr == nil {
		if repo, ok := repositoryLoadCache.Get(cacheKey, fingerprint); ok {
			return repo, nil
		}
	}

	repo, err := loadRepositoryUncached(rootPath, options, patterns)
	if err == nil && fingerprintErr == nil {
		repositoryLoadCache.Put(cacheKey, fingerprint, repo)
	}

	return repo, err
}

func loadRepositoryUncached(rootPath string, options LoadOptions, patterns []string) (Repository, error) {
	loaded, err := loadPackages(rootPath, options, patterns, "")
	if err != nil {
		if packageLoadErrorIsCacheAccess(err.Error()) {
			if retried, retryErr, ok := retryLoadRepositoryWithWritableCache(rootPath, options, patterns); ok {
				return retried, retryErr
			}
		}

		return Repository{}, fmt.Errorf("load packages: %w", err)
	}
	if len(loaded) == 0 {
		if retried, retryErr, ok := retryLoadRepositoryWithWritableCache(rootPath, options, patterns); ok {
			return retried, retryErr
		}
	}

	repo, err := repositoryFromLoaded(rootPath, patterns, loaded)
	if packageLoadHasCacheAccessError(loaded) {
		if retried, retryErr, ok := retryLoadRepositoryWithWritableCache(rootPath, options, patterns); ok {
			return retried, retryErr
		}
	}

	return repo, err
}

func packageLoadPatterns(root string, options LoadOptions) []string {
	patterns := options.Patterns
	if len(patterns) == 0 {
		if workspacePatterns, ok := workspacePackageLoadPatterns(root); ok {
			return workspacePatterns
		}

		patterns = []string{"./..."}
	}

	return patterns
}

func workspacePackageLoadPatterns(root string) ([]string, bool) {
	goWorkPath := filepath.Join(root, "go.work")
	contents, err := os.ReadFile(goWorkPath)
	if err != nil {
		return nil, false
	}

	workFile, err := modfile.ParseWork(goWorkPath, contents, nil)
	if err != nil {
		return nil, false
	}

	var patterns []string
	for _, use := range workFile.Use {
		if use == nil {
			continue
		}

		path := strings.TrimSpace(filepath.ToSlash(use.Path))
		if path == "" {
			continue
		}
		if filepath.IsAbs(path) {
			relativePath, err := filepath.Rel(root, filepath.Clean(path))
			if err != nil || relativePath == "." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || relativePath == ".." {
				continue
			}
			path = filepath.ToSlash(relativePath)
		}

		path = strings.TrimPrefix(path, "./")
		cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		if cleaned == "." {
			patterns = append(patterns, "./...")
			continue
		}
		if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
			continue
		}

		patterns = append(patterns, "./"+cleaned+"/...")
	}

	patterns = uniqueSorted(patterns)
	if len(patterns) == 0 {
		return nil, false
	}

	return patterns, true
}

func loadPackages(root string, options LoadOptions, patterns []string, goCache string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Dir:        root,
		Tests:      options.IncludeTests,
		BuildFlags: packageLoadBuildFlags(options),
	}
	if strings.TrimSpace(goCache) != "" {
		cfg.Env = envWith("GOCACHE", goCache)
	}

	return packageLoader(cfg, patterns...)
}

func retryLoadRepositoryWithWritableCache(root string, options LoadOptions, patterns []string) (Repository, error, bool) {
	cache, err := writableGoBuildCache()
	if err != nil {
		return Repository{}, nil, false
	}

	loaded, err := loadPackages(root, options, patterns, cache)
	if err != nil {
		return Repository{}, fmt.Errorf("load packages: %w", err), true
	}

	repo, err := repositoryFromLoaded(root, patterns, loaded)
	return repo, err, true
}

func writableGoBuildCache() (string, error) {
	path := filepath.Join(os.TempDir(), "gosherpa-go-build-cache")
	if err := os.MkdirAll(path, 0700); err != nil {
		return "", err
	}

	return path, nil
}

func envWith(key string, value string) []string {
	prefix := key + "="
	env := os.Environ()
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}

	return append(env, prefix+value)
}

func repositoryFromLoaded(root string, patterns []string, loaded []*packages.Package) (Repository, error) {
	if len(loaded) == 0 {
		return Repository{}, fmt.Errorf("load packages: no packages matched %s", strings.Join(patterns, ", "))
	}

	repo := Repository{Root: root}
	for _, pkg := range loaded {
		semanticPackage, ok, err := packageFromLoaded(root, pkg)
		if err != nil {
			return Repository{}, err
		}
		if !ok {
			continue
		}

		repo.Packages = append(repo.Packages, semanticPackage)
		repo.Warnings = append(repo.Warnings, packageWarnings(root, pkg)...)
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

func packageLoadBuildFlags(options LoadOptions) []string {
	flags := append([]string{}, options.BuildFlags...)
	tags := NormalizeBuildTags(options.BuildTags)
	if len(tags) > 0 {
		flags = append(flags, "-tags="+strings.Join(tags, ","))
	}

	return flags
}

func NormalizeBuildTags(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	for _, value := range values {
		fields := strings.FieldsFunc(value, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
		for _, field := range fields {
			tag := strings.TrimSpace(field)
			if tag == "" {
				continue
			}

			seen[tag] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}

	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	return tags
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

func packageLoadHasCacheAccessError(packages []*packages.Package) bool {
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		for _, packageErr := range pkg.Errors {
			if packageLoadErrorIsCacheAccess(packageErr.Error()) {
				return true
			}
		}
		for _, typeErr := range pkg.TypeErrors {
			if packageLoadErrorIsCacheAccess(typeErr.Error()) {
				return true
			}
		}
	}

	return false
}

func packageLoadErrorIsCacheAccess(message string) bool {
	value := strings.ToLower(filepath.ToSlash(message))
	if !strings.Contains(value, "go-build") && !strings.Contains(value, "loading compiled go files from cache") {
		return false
	}

	return strings.Contains(value, "operation not permitted") ||
		strings.Contains(value, "permission denied") ||
		strings.Contains(value, "cache entry not found")
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
