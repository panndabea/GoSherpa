package sherpa

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type PackageDependencies struct {
	Package string   `json:"package"`
	Imports []string `json:"imports"`
	UsedBy  []string `json:"usedBy"`
}

func FindPackageDependencies(root string, targetPackage string) (PackageDependencies, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return PackageDependencies{}, err
	}

	modPath, err := modulePath(rootPath)
	if err != nil {
		return PackageDependencies{}, err
	}

	target, err := normalizeTargetPackage(targetPackage, modPath)
	if err != nil {
		return PackageDependencies{}, err
	}

	importsByPackage, err := collectPackageImports(rootPath)
	if err != nil {
		return PackageDependencies{}, err
	}

	rawImports, ok := importsByPackage[target]
	if !ok {
		return PackageDependencies{Package: target}, fmt.Errorf("package not found: %s", target)
	}

	deps := PackageDependencies{Package: target}

	var displayImports []string
	for _, importPath := range rawImports {
		displayImports = append(displayImports, displayImportPath(importPath, modPath))
	}
	deps.Imports = uniqueSorted(displayImports)

	var usedBy []string
	for pkg, imports := range importsByPackage {
		if pkg == target {
			continue
		}

		for _, importPath := range imports {
			localPath, ok := localPackagePath(importPath, modPath)
			if ok && localPath == target {
				usedBy = append(usedBy, pkg)
				break
			}
		}
	}
	deps.UsedBy = uniqueSorted(usedBy)

	return deps, nil
}

func modulePath(root string) (string, error) {
	goModPath := filepath.Join(root, "go.mod")

	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "//") {
			continue
		}
		if !strings.HasPrefix(line, "module ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			return fields[1], nil
		}
	}

	return "", fmt.Errorf("go.mod does not contain a module directive")
}

func normalizeTargetPackage(targetPackage string, modulePath string) (string, error) {
	value := strings.TrimSpace(targetPackage)
	if value == "" {
		return "", fmt.Errorf("package path is empty")
	}

	if filepath.IsAbs(value) {
		return "", fmt.Errorf("absolute package paths are not supported: %s", value)
	}

	value = filepath.ToSlash(value)

	if value == modulePath {
		return ".", nil
	}

	if strings.HasPrefix(value, modulePath+"/") {
		value = strings.TrimPrefix(value, modulePath+"/")
	}

	value = strings.TrimPrefix(value, "./")

	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", fmt.Errorf("package path must not contain '..': %s", targetPackage)
		}
	}

	cleaned := path.Clean(value)
	if cleaned == "." {
		return ".", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("package path escapes repository: %s", targetPackage)
	}

	return "./" + cleaned, nil
}

func packagePathForFile(root string, file string) (string, error) {
	dir := filepath.Dir(file)
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

func parseImports(path string) ([]string, error) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	var imports []string
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("parse import path %s: %w", importSpec.Path.Value, err)
		}

		imports = append(imports, importPath)
	}

	return uniqueSorted(imports), nil
}

func collectPackageImports(root string) (map[string][]string, error) {
	files, err := FindGoFiles(root)
	if err != nil {
		return nil, err
	}

	importsByPackage := map[string][]string{}
	for _, file := range files {
		pkg, err := packagePathForFile(root, file)
		if err != nil {
			return nil, err
		}

		if _, ok := importsByPackage[pkg]; !ok {
			importsByPackage[pkg] = nil
		}

		imports, err := parseImports(file)
		if err != nil {
			return nil, err
		}

		importsByPackage[pkg] = append(importsByPackage[pkg], imports...)
	}

	for pkg, imports := range importsByPackage {
		importsByPackage[pkg] = uniqueSorted(imports)
	}

	return importsByPackage, nil
}

func localPackagePath(importPath string, modulePath string) (string, bool) {
	if importPath == modulePath {
		return ".", true
	}

	prefix := modulePath + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}

	return "./" + strings.TrimPrefix(importPath, prefix), true
}

func displayImportPath(importPath string, modulePath string) string {
	localPath, ok := localPackagePath(importPath, modulePath)
	if ok {
		return localPath
	}

	return importPath
}

func uniqueSorted(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
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
