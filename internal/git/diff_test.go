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
