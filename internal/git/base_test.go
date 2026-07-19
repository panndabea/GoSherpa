package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectBaseVerifiesExplicitRef(t *testing.T) {
	root := initBaseGitTestRepository(t)
	writeBaseGitTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	runBaseGit(t, root, "add", ".")
	runBaseGit(t, root, "commit", "-m", "initial")

	got, err := DetectBase(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	if got.Selected != "HEAD" || got.Source != BaseSourceExplicit {
		t.Fatalf("unexpected detection: %#v", got)
	}
	if len(got.Candidates) != 1 || !got.Candidates[0].Resolved || got.Candidates[0].Commit == "" {
		t.Fatalf("expected resolved explicit candidate, got %#v", got.Candidates)
	}
}

func TestDetectBaseRejectsMissingExplicitRef(t *testing.T) {
	root := initBaseGitTestRepository(t)

	got, err := DetectBase(root, "missing")
	if err == nil {
		t.Fatal("expected explicit ref error")
	}
	if got.Source != BaseSourceExplicit {
		t.Fatalf("expected explicit source, got %#v", got)
	}
	if len(got.Candidates) != 1 || got.Candidates[0].Resolved {
		t.Fatalf("expected unresolved candidate, got %#v", got.Candidates)
	}
}

func TestDetectBaseUsesCandidateOrder(t *testing.T) {
	root := initBaseGitTestRepository(t)
	writeBaseGitTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	runBaseGit(t, root, "add", ".")
	runBaseGit(t, root, "commit", "-m", "initial")
	runBaseGit(t, root, "branch", "-M", "main")
	runBaseGit(t, root, "update-ref", "refs/remotes/origin/main", "HEAD")

	got, err := DetectBase(root, "")
	if err != nil {
		t.Fatal(err)
	}

	if got.Selected != "origin/main" || got.Source != BaseSourceDetected {
		t.Fatalf("expected origin/main detection, got %#v", got)
	}
	if len(got.Candidates) == 0 || got.Candidates[0].Ref != "origin/main" || !got.Candidates[0].Resolved {
		t.Fatalf("expected first candidate to resolve origin/main, got %#v", got.Candidates)
	}
}

func TestDetectBaseFallsBackOutsideGit(t *testing.T) {
	root := t.TempDir()

	got, err := DetectBase(root, "")
	if err != nil {
		t.Fatal(err)
	}

	if got.Selected != "HEAD" || got.Source != BaseSourceFallback {
		t.Fatalf("expected HEAD fallback, got %#v", got)
	}
	if len(got.Warnings) == 0 || !strings.Contains(got.Warnings[0], "not a git work tree") {
		t.Fatalf("expected non-git warning, got %#v", got.Warnings)
	}
}

func initBaseGitTestRepository(t *testing.T) string {
	t.Helper()

	requireBaseGit(t)
	root := t.TempDir()
	runBaseGit(t, root, "init")
	runBaseGit(t, root, "config", "user.email", "test@example.com")
	runBaseGit(t, root, "config", "user.name", "Test User")
	return root
}

func requireBaseGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}
}

func runBaseGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	commandArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output)
}

func writeBaseGitTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
