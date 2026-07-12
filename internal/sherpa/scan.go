package sherpa

import (
	"io/fs"
	"os"
	"path/filepath"
)

func FindGoFiles(root string) ([]string, error) {
	var files []string
	rootPath := filepath.Clean(root)
	skipNestedModules := regularFileExists(filepath.Join(rootPath, "go.mod"))

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skipNestedModules && filepath.Clean(path) != rootPath && regularFileExists(filepath.Join(path, "go.mod")) {
				return fs.SkipDir
			}

			return nil
		}

		if filepath.Ext(path) == ".go" {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
