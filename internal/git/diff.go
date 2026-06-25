package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type ChangedLineRange struct {
	Start int
	End   int
}

type ChangedFileLineRanges struct {
	Path   string
	Ranges []ChangedLineRange
}

var diffHunkHeaderPattern = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

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

func ChangedLineRanges(root string, base string, head string) ([]ChangedFileLineRanges, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("repository root is empty")
	}

	base = strings.TrimSpace(base)
	if base == "" {
		return nil, fmt.Errorf("base ref is empty")
	}

	args := []string{"-C", root, "diff", "--unified=0", base}
	if strings.TrimSpace(head) != "" {
		args = append(args, strings.TrimSpace(head))
	}

	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, fmt.Errorf("git diff --unified=0 failed: %w", err)
		}
		return nil, fmt.Errorf("git diff --unified=0 failed: %s: %w", message, err)
	}

	return parseChangedLineRanges(output), nil
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

func parseChangedLineRanges(output []byte) []ChangedFileLineRanges {
	byPath := make(map[string][]ChangedLineRange)
	var currentPath string

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "+++ ") {
			currentPath = diffNewPath(line)
			continue
		}
		if currentPath == "" {
			continue
		}

		lineRange, ok := diffHunkLineRange(line)
		if !ok {
			continue
		}

		byPath[currentPath] = append(byPath[currentPath], lineRange)
	}

	files := make([]ChangedFileLineRanges, 0, len(byPath))
	for path, ranges := range byPath {
		files = append(files, ChangedFileLineRanges{
			Path:   path,
			Ranges: mergeChangedLineRanges(ranges),
		})
	}

	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files
}

func diffNewPath(line string) string {
	value := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
	if value == "/dev/null" || value == "" {
		return ""
	}

	if strings.HasPrefix(value, `"`) {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			value = unquoted
		}
	}

	value = strings.TrimPrefix(value, "b/")

	return filepath.ToSlash(value)
}

func diffHunkLineRange(line string) (ChangedLineRange, bool) {
	matches := diffHunkHeaderPattern.FindStringSubmatch(line)
	if matches == nil {
		return ChangedLineRange{}, false
	}

	start, err := strconv.Atoi(matches[1])
	if err != nil {
		return ChangedLineRange{}, false
	}
	if start < 1 {
		start = 1
	}

	count := 1
	if matches[2] != "" {
		count, err = strconv.Atoi(matches[2])
		if err != nil {
			return ChangedLineRange{}, false
		}
	}

	end := start + count - 1
	if count == 0 {
		end = start
	}

	return ChangedLineRange{Start: start, End: end}, true
}

func mergeChangedLineRanges(ranges []ChangedLineRange) []ChangedLineRange {
	sort.Slice(ranges, func(i int, j int) bool {
		if ranges[i].Start != ranges[j].Start {
			return ranges[i].Start < ranges[j].Start
		}

		return ranges[i].End < ranges[j].End
	})

	var merged []ChangedLineRange
	for _, lineRange := range ranges {
		if lineRange.End < lineRange.Start {
			lineRange.End = lineRange.Start
		}

		if len(merged) == 0 {
			merged = append(merged, lineRange)
			continue
		}

		last := &merged[len(merged)-1]
		if lineRange.Start <= last.End+1 {
			if lineRange.End > last.End {
				last.End = lineRange.End
			}
			continue
		}

		merged = append(merged, lineRange)
	}

	return merged
}
