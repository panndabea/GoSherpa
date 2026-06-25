package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestChangedFilesBetweenRefs(t *testing.T) {
	root := initGitTestRepository(t)

	writeGitTestFile(t, filepath.Join(root, "service.go"), "package service\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	writeGitTestFile(t, filepath.Join(root, "service.go"), "package service\n\nfunc Run() {}\n")
	writeGitTestFile(t, filepath.Join(root, "internal", "worker.go"), "package internal\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "change service")
	head := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	got, err := ChangedFiles(root, base, head)
	if err != nil {
		t.Fatalf("ChangedFiles returned error: %v", err)
	}

	want := []string{"internal/worker.go", "service.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChangedFiles() = %#v, want %#v", got, want)
	}
}

func TestChangedFilesAgainstWorkingTree(t *testing.T) {
	root := initGitTestRepository(t)

	writeGitTestFile(t, filepath.Join(root, "service.go"), "package service\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	writeGitTestFile(t, filepath.Join(root, "service.go"), "package service\n\nfunc Run() {}\n")

	got, err := ChangedFiles(root, base, "")
	if err != nil {
		t.Fatalf("ChangedFiles returned error: %v", err)
	}

	want := []string{"service.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChangedFiles() = %#v, want %#v", got, want)
	}
}

func TestChangedLineRangesBetweenRefs(t *testing.T) {
	root := initGitTestRepository(t)

	writeGitTestFile(t, filepath.Join(root, "service.go"), `package service

func Run() string {
	return "old"
}
`)
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	writeGitTestFile(t, filepath.Join(root, "service.go"), `package service

func Run() string {
	return "new"
}

func Added() {}
`)
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "change service")
	head := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	got, err := ChangedLineRanges(root, base, head)
	if err != nil {
		t.Fatalf("ChangedLineRanges returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("ChangedLineRanges returned %#v, want one changed file", got)
	}
	if got[0].Path != "service.go" {
		t.Fatalf("ChangedLineRanges path = %q, want service.go", got[0].Path)
	}
	if !lineRangesContain(got[0].Ranges, 4) {
		t.Fatalf("ChangedLineRanges ranges = %#v, want line 4", got[0].Ranges)
	}
	if !lineRangesContain(got[0].Ranges, 7) {
		t.Fatalf("ChangedLineRanges ranges = %#v, want line 7", got[0].Ranges)
	}
}

func TestChangedLineRangesIncludesOldRangesForDeletedHunks(t *testing.T) {
	root := initGitTestRepository(t)

	writeGitTestFile(t, filepath.Join(root, "service.go"), `package service

func Kept() {}

func Removed() string {
	return "old"
}
`)
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	writeGitTestFile(t, filepath.Join(root, "service.go"), `package service

func Kept() {}
`)
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "remove function")
	head := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	got, err := ChangedLineRanges(root, base, head)
	if err != nil {
		t.Fatalf("ChangedLineRanges returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("ChangedLineRanges returned %#v, want one changed file", got)
	}
	if got[0].Path != "service.go" {
		t.Fatalf("ChangedLineRanges path = %q, want service.go", got[0].Path)
	}
	if got[0].OldPath != "service.go" {
		t.Fatalf("ChangedLineRanges old path = %q, want service.go", got[0].OldPath)
	}
	if len(got[0].Ranges) != 0 {
		t.Fatalf("ChangedLineRanges ranges = %#v, want no current-file ranges", got[0].Ranges)
	}
	if !lineRangesContain(got[0].OldRanges, 5) {
		t.Fatalf("ChangedLineRanges old ranges = %#v, want deleted function line 5", got[0].OldRanges)
	}
}

func TestFileAtRefReadsFileContents(t *testing.T) {
	root := initGitTestRepository(t)

	contents := "package service\n\nfunc Run() string { return \"old\" }\n"
	writeGitTestFile(t, filepath.Join(root, "service.go"), contents)
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))

	writeGitTestFile(t, filepath.Join(root, "service.go"), "package service\n\nfunc Run() string { return \"new\" }\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "change file")

	got, err := FileAtRef(root, base, "service.go")
	if err != nil {
		t.Fatalf("FileAtRef returned error: %v", err)
	}

	if string(got) != contents {
		t.Fatalf("FileAtRef() = %q, want %q", string(got), contents)
	}
}

func TestChangedFilesRejectsEmptyRoot(t *testing.T) {
	_, err := ChangedFiles(" \t ", "HEAD", "")
	if err == nil {
		t.Fatal("ChangedFiles returned nil error")
	}
	if !strings.Contains(err.Error(), "repository root is empty") {
		t.Fatalf("ChangedFiles error = %q, want repository root error", err)
	}
}

func TestChangedFilesRejectsEmptyBase(t *testing.T) {
	root := t.TempDir()

	_, err := ChangedFiles(root, " \t ", "")
	if err == nil {
		t.Fatal("ChangedFiles returned nil error")
	}
	if !strings.Contains(err.Error(), "base ref is empty") {
		t.Fatalf("ChangedFiles error = %q, want base ref error", err)
	}
}

func TestChangedFilesReturnsGitErrors(t *testing.T) {
	root := initGitTestRepository(t)

	_, err := ChangedFiles(root, "missing-ref", "")
	if err == nil {
		t.Fatal("ChangedFiles returned nil error")
	}
	if !strings.Contains(err.Error(), "git diff --name-only failed") {
		t.Fatalf("ChangedFiles error = %q, want git diff error", err)
	}
}

func TestChangedLineRangesReturnsGitErrors(t *testing.T) {
	root := initGitTestRepository(t)

	_, err := ChangedLineRanges(root, "missing-ref", "")
	if err == nil {
		t.Fatal("ChangedLineRanges returned nil error")
	}
	if !strings.Contains(err.Error(), "git diff --unified=0 failed") {
		t.Fatalf("ChangedLineRanges error = %q, want git diff error", err)
	}
}

func initGitTestRepository(t *testing.T) string {
	t.Helper()

	requireGit(t)

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")

	return root
}

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}
}

func runGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}

	return string(output)
}

func writeGitTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func lineRangesContain(ranges []ChangedLineRange, line int) bool {
	for _, lineRange := range ranges {
		if lineRange.Start <= line && line <= lineRange.End {
			return true
		}
	}

	return false
}
