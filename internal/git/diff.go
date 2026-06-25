package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func ChangedFiles(root string, base string, head string) ([]string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("repository root is empty")
	}

	base = strings.TrimSpace(base)
	if base == "" {
		return nil, fmt.Errorf("base ref is empty")
	}

	args := []string{"-C", root, "diff", "--name-only", base}
	if strings.TrimSpace(head) != "" {
		args = append(args, strings.TrimSpace(head))
	}

	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, fmt.Errorf("git diff --name-only failed: %w", err)
		}
		return nil, fmt.Errorf("git diff --name-only failed: %s: %w", message, err)
	}

	files := parseChangedFiles(output)
	sort.Strings(files)

	return files, nil
}

func parseChangedFiles(output []byte) []string {
	seen := make(map[string]struct{})
	var files []string

	for _, line := range strings.Split(string(output), "\n") {
		file := strings.TrimSpace(line)
		if file == "" {
			continue
		}

		file = filepath.ToSlash(file)
		if _, ok := seen[file]; ok {
			continue
		}

		seen[file] = struct{}{}
		files = append(files, file)
	}

	return files
}
