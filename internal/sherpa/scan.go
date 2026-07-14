package sherpa

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func FindGoFiles(root string) ([]string, error) {
	rootPath := filepath.Clean(root)
	files := make(map[string]struct{})

	if moduleDirs := workspaceModuleDirs(rootPath); len(moduleDirs) > 0 {
		for _, moduleDir := range moduleDirs {
			if err := collectGoFiles(moduleDir, true, files); err != nil {
				return nil, err
			}
		}

		return sortedGoFiles(files), nil
	}

	if err := collectGoFiles(rootPath, regularFileExists(filepath.Join(rootPath, "go.mod")), files); err != nil {
		return nil, err
	}

	return sortedGoFiles(files), nil
}

func collectGoFiles(walkRoot string, skipNestedModules bool, files map[string]struct{}) error {
	return filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skipNestedModules && filepath.Clean(path) != filepath.Clean(walkRoot) && regularFileExists(filepath.Join(path, "go.mod")) {
				return fs.SkipDir
			}

			return nil
		}

		if filepath.Ext(path) == ".go" {
			files[filepath.Clean(path)] = struct{}{}
		}

		return nil
	})
}

func sortedGoFiles(files map[string]struct{}) []string {
	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}
	sort.Strings(result)

	return result
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
