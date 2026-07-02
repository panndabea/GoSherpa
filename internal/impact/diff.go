package impact

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	gitdiff "github.com/panndabea/GoSherpa/internal/git"
)

func ChangedPackages(root string, base string, head string) ([]string, error) {
	files, err := gitdiff.ChangedFiles(root, base, head)
	if err != nil {
		return nil, err
	}

	return PackagesForFiles(files), nil
}

func PackagesForFiles(files []string) []string {
	seen := make(map[string]struct{})

	for _, file := range files {
		pkg, ok := packageForChangedFile(file)
		if !ok {
			continue
		}

		seen[pkg] = struct{}{}
	}

	packages := make([]string, 0, len(seen))
	for pkg := range seen {
		packages = append(packages, pkg)
	}

	sort.Strings(packages)

	return packages
}

func packageForChangedFile(file string) (string, bool) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", false
	}

	file = filepath.ToSlash(file)
	file = path.Clean(file)

	if file == "." || path.IsAbs(file) || file == ".." || strings.HasPrefix(file, "../") {
		return "", false
	}
	if path.Ext(file) != ".go" {
		return "", false
	}

	dir := path.Dir(file)
	if dir == "." {
		return ".", true
	}

	return "./" + dir, true
}
