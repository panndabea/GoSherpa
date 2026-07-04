package sherpa

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

func WorkspacePackagePathForImportPath(root string, importPath string) (string, bool) {
	value := strings.TrimSpace(filepath.ToSlash(importPath))
	if value == "" || strings.HasPrefix(value, "./") || filepath.IsAbs(value) {
		return "", false
	}

	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return "", false
	}

	if modulePath := readModulePath(rootPath); modulePath != "" {
		return localPackagePath(value, modulePath)
	}

	for _, moduleDir := range workspaceModuleDirs(rootPath) {
		modulePath := readModulePath(moduleDir)
		if modulePath == "" {
			continue
		}

		localPath, ok := localPackagePath(value, modulePath)
		if !ok {
			continue
		}

		packagePath, ok := workspacePackagePath(rootPath, moduleDir, localPath)
		if ok {
			return packagePath, true
		}
	}

	return "", false
}

func workspaceModuleDirs(root string) []string {
	workPath := filepath.Join(root, "go.work")
	contents, err := os.ReadFile(workPath)
	if err != nil {
		return nil
	}

	workFile, err := modfile.ParseWork(workPath, contents, nil)
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	for _, use := range workFile.Use {
		if use == nil {
			continue
		}

		value := strings.TrimSpace(use.Path)
		if value == "" {
			continue
		}

		moduleDir := value
		if !filepath.IsAbs(moduleDir) {
			moduleDir = filepath.Join(root, filepath.FromSlash(moduleDir))
		}
		moduleDir = filepath.Clean(moduleDir)
		if !workspacePathInsideRoot(root, moduleDir) {
			continue
		}

		seen[moduleDir] = struct{}{}
	}

	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	return dirs
}

func workspacePackagePath(root string, moduleDir string, localPath string) (string, bool) {
	relativeModule, err := filepath.Rel(root, moduleDir)
	if err != nil {
		return "", false
	}

	relativeModule = filepath.ToSlash(relativeModule)
	if relativeModule == "." {
		relativeModule = ""
	}

	localPath = strings.TrimPrefix(filepath.ToSlash(localPath), "./")
	if localPath == "." {
		localPath = ""
	}

	value := relativeModule
	if localPath != "" {
		if value == "" {
			value = localPath
		} else {
			value = path.Join(value, localPath)
		}
	}

	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == "" {
		return ".", true
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}

	return "./" + cleaned, true
}

func workspacePathInsideRoot(root string, target string) bool {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}

	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
