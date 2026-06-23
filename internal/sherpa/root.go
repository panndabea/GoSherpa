package sherpa

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type RepositoryRoot struct {
	Path string
}

func ResolveRepositoryRoot(input string) (RepositoryRoot, error) {
	root, err := absoluteRootPath(input)
	if err != nil {
		return RepositoryRoot{}, err
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return RepositoryRoot{}, fmt.Errorf("repository root does not exist: %s", root)
		}
		return RepositoryRoot{}, fmt.Errorf("stat repository root %s: %w", root, err)
	}

	if !info.IsDir() {
		return RepositoryRoot{}, fmt.Errorf("repository root is not a directory: %s", root)
	}

	goModPath := filepath.Join(root, "go.mod")
	goModInfo, err := os.Stat(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return RepositoryRoot{}, fmt.Errorf("repository root does not contain go.mod: %s", root)
		}
		return RepositoryRoot{}, fmt.Errorf("stat go.mod in repository root %s: %w", root, err)
	}

	if goModInfo.IsDir() {
		return RepositoryRoot{}, fmt.Errorf("repository root go.mod is not a file: %s", goModPath)
	}

	return RepositoryRoot{Path: root}, nil
}

func absoluteRootPath(root string) (string, error) {
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

func positionRelativeToRoot(root string, position Position) Position {
	if position.File == "" {
		return position
	}

	if strings.TrimSpace(root) == "" {
		position.File = filepath.ToSlash(position.File)
		return position
	}

	rootPath, err := absoluteRootPath(root)
	if err != nil {
		position.File = filepath.ToSlash(position.File)
		return position
	}

	filePath := position.File
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(rootPath, filePath)
	}
	filePath = filepath.Clean(filePath)

	relativePath, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		position.File = filepath.ToSlash(position.File)
		return position
	}

	if relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		position.File = filepath.ToSlash(position.File)
		return position
	}

	position.File = filepath.ToSlash(relativePath)
	return position
}
