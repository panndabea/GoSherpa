package sherpa

import (
	"path"
	"path/filepath"
	"strings"
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
		if localPath, ok := localPackagePath(value, modulePath); ok {
			return localPath, true
		}
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
	records, _ := workspaceModuleRecords(root, filepath.Join(root, "go.work"))
	dirs := make([]string, 0, len(records))
	for _, record := range records {
		if !record.Summary.InsideRoot {
			continue
		}
		dirs = append(dirs, record.Dir)
	}
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
