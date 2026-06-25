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
	Path      string
	OldPath   string
	Ranges    []ChangedLineRange
	OldRanges []ChangedLineRange
}

var diffHunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

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

func FileAtRef(root string, ref string, file string) ([]byte, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("repository root is empty")
	}

	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("ref is empty")
	}

	file = filepath.ToSlash(strings.TrimSpace(file))
	if file == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	output, err := exec.Command("git", "-C", root, "show", ref+":"+file).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, fmt.Errorf("git show failed: %w", err)
		}
		return nil, fmt.Errorf("git show failed: %s: %w", message, err)
	}

	return output, nil
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
	byPath := make(map[string]*ChangedFileLineRanges)
	var oldPath string
	var newPath string

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			oldPath = ""
			newPath = ""
			continue
		}
		if strings.HasPrefix(line, "--- ") {
			oldPath = diffPath(line, "--- ", "a/")
			continue
		}
		if strings.HasPrefix(line, "+++ ") {
			newPath = diffPath(line, "+++ ", "b/")
			continue
		}

		oldRange, oldCount, newRange, newCount, ok := diffHunkLineRanges(line)
		if !ok {
			continue
		}

		path := newPath
		if path == "" {
			path = oldPath
		}
		if path == "" {
			continue
		}

		changedFile := byPath[path]
		if changedFile == nil {
			changedFile = &ChangedFileLineRanges{
				Path:    path,
				OldPath: oldPath,
			}
			byPath[path] = changedFile
		}
		if changedFile.OldPath == "" {
			changedFile.OldPath = oldPath
		}

		if newCount > 0 {
			changedFile.Ranges = append(changedFile.Ranges, newRange)
		}
		if oldCount > 0 {
			changedFile.OldRanges = append(changedFile.OldRanges, oldRange)
		}
	}

	files := make([]ChangedFileLineRanges, 0, len(byPath))
	for _, changedFile := range byPath {
		files = append(files, ChangedFileLineRanges{
			Path:      changedFile.Path,
			OldPath:   changedFile.OldPath,
			Ranges:    mergeChangedLineRanges(changedFile.Ranges),
			OldRanges: mergeChangedLineRanges(changedFile.OldRanges),
		})
	}

	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files
}

func diffPath(line string, prefix string, gitPrefix string) string {
	value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if value == "/dev/null" || value == "" {
		return ""
	}

	if strings.HasPrefix(value, `"`) {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			value = unquoted
		}
	}

	value = strings.TrimPrefix(value, gitPrefix)

	return filepath.ToSlash(value)
}

func diffHunkLineRanges(line string) (ChangedLineRange, int, ChangedLineRange, int, bool) {
	matches := diffHunkHeaderPattern.FindStringSubmatch(line)
	if matches == nil {
		return ChangedLineRange{}, 0, ChangedLineRange{}, 0, false
	}

	oldRange, oldCount, err := diffHunkSideLineRange(matches[1], matches[2])
	if err != nil {
		return ChangedLineRange{}, 0, ChangedLineRange{}, 0, false
	}

	newRange, newCount, err := diffHunkSideLineRange(matches[3], matches[4])
	if err != nil {
		return ChangedLineRange{}, 0, ChangedLineRange{}, 0, false
	}

	return oldRange, oldCount, newRange, newCount, true
}

func diffHunkSideLineRange(startValue string, countValue string) (ChangedLineRange, int, error) {
	start, err := strconv.Atoi(startValue)
	if err != nil {
		return ChangedLineRange{}, 0, err
	}
	if start < 1 {
		start = 1
	}

	count := 1
	if countValue != "" {
		count, err = strconv.Atoi(countValue)
		if err != nil {
			return ChangedLineRange{}, 0, err
		}
	}

	end := start + count - 1
	if count == 0 {
		end = start
	}

	return ChangedLineRange{Start: start, End: end}, count, nil
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
