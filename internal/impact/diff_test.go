package impact

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPackagesForFiles(t *testing.T) {
	got := PackagesForFiles([]string{
		"main.go",
		"internal/auth/session.go",
		"internal/auth/session_test.go",
		"internal//api/handler.go",
		"README.md",
		"go.mod",
		"internal/auth/notes.txt",
		"",
		"../outside.go",
		"/tmp/outside.go",
	})

	want := []string{".", "./internal/api", "./internal/auth"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PackagesForFiles() = %#v, want %#v", got, want)
	}
}

func TestChangedPackagesBetweenRefs(t *testing.T) {
	root := initImpactGitTestRepository(t)

	writeImpactTestFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# test\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	writeImpactTestFile(t, filepath.Join(root, "main.go"), "package main\n\nfunc main() {}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler.go"), "package api\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# changed\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "change packages")
	head := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	got, err := ChangedPackages(root, base, head)
	if err != nil {
		t.Fatalf("ChangedPackages returned error: %v", err)
	}

	want := []string{".", "./internal/api"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChangedPackages() = %#v, want %#v", got, want)
	}
}

func TestChangedPackagesAgainstWorkingTree(t *testing.T) {
	root := initImpactGitTestRepository(t)

	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# test\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n\nfunc Load() {}\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# changed\n")

	got, err := ChangedPackages(root, base, "")
	if err != nil {
		t.Fatalf("ChangedPackages returned error: %v", err)
	}

	want := []string{"./internal/auth"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChangedPackages() = %#v, want %#v", got, want)
	}
}

func TestChangedPackagesPropagatesGitErrors(t *testing.T) {
	root := initImpactGitTestRepository(t)

	_, err := ChangedPackages(root, "missing-ref", "")
	if err == nil {
		t.Fatal("ChangedPackages returned nil error")
	}
	if !strings.Contains(err.Error(), "git diff --name-only failed") {
		t.Fatalf("ChangedPackages error = %q, want git diff error", err)
	}
}

func initImpactGitTestRepository(t *testing.T) string {
	t.Helper()

	requireImpactGit(t)

	root := t.TempDir()
	runImpactGit(t, root, "init")
	runImpactGit(t, root, "config", "user.email", "test@example.com")
	runImpactGit(t, root, "config", "user.name", "Test User")

	return root
}

func requireImpactGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}
}

func runImpactGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}

	return string(output)
}

func writeImpactTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
