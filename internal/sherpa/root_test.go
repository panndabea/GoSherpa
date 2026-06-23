package sherpa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRepositoryRootAcceptsDirectoryWithGoMod(t *testing.T) {
	tmp := t.TempDir()
	writeRootTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")

	got, err := ResolveRepositoryRoot(tmp)
	if err != nil {
		t.Fatal(err)
	}

	want, err := filepath.Abs(tmp)
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Clean(want)

	if got.Path != want {
		t.Fatalf("expected %s, got %s", want, got.Path)
	}
}

func TestResolveRepositoryRootTrimsWhitespace(t *testing.T) {
	tmp := t.TempDir()
	writeRootTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")

	got, err := ResolveRepositoryRoot("  " + tmp + "  ")
	if err != nil {
		t.Fatal(err)
	}

	want, err := filepath.Abs(tmp)
	if err != nil {
		t.Fatal(err)
	}
	want = filepath.Clean(want)

	if got.Path != want {
		t.Fatalf("expected %s, got %s", want, got.Path)
	}
}

func TestResolveRepositoryRootRejectsEmptyRoot(t *testing.T) {
	_, err := ResolveRepositoryRoot("   ")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "repository root is empty") {
		t.Fatalf("expected empty root error, got %v", err)
	}
}

func TestResolveRepositoryRootRejectsMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	_, err := ResolveRepositoryRoot(missing)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "repository root does not exist") {
		t.Fatalf("expected missing root error, got %v", err)
	}
}

func TestResolveRepositoryRootRejectsFileRoot(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "go.mod")
	writeRootTestFile(t, path, "module example.com/app\n")

	_, err := ResolveRepositoryRoot(path)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "repository root is not a directory") {
		t.Fatalf("expected file root error, got %v", err)
	}
}

func TestResolveRepositoryRootRejectsDirectoryWithoutGoMod(t *testing.T) {
	tmp := t.TempDir()

	_, err := ResolveRepositoryRoot(tmp)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "repository root does not contain go.mod") {
		t.Fatalf("expected missing go.mod error, got %v", err)
	}
}

func TestResolveRepositoryRootRejectsGoModDirectory(t *testing.T) {
	tmp := t.TempDir()
	err := os.Mkdir(filepath.Join(tmp, "go.mod"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ResolveRepositoryRoot(tmp)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "repository root go.mod is not a file") {
		t.Fatalf("expected go.mod directory error, got %v", err)
	}
}

func TestPositionRelativeToRootReturnsRelativeSlashPath(t *testing.T) {
	root := t.TempDir()
	position := Position{
		File: filepath.Join(root, "internal", "sherpa", "parse.go"),
		Line: 12,
	}

	got := positionRelativeToRoot(root, position)

	if got.File != "internal/sherpa/parse.go" {
		t.Fatalf("expected internal/sherpa/parse.go, got %s", got.File)
	}

	if got.Line != 12 {
		t.Fatalf("expected line 12, got %d", got.Line)
	}
}

func TestPositionRelativeToRootCleansRelativeSegments(t *testing.T) {
	root := t.TempDir()
	position := Position{
		File: filepath.Join("internal", "..", "service.go"),
		Line: 7,
	}

	got := positionRelativeToRoot(root, position)

	if got.File != "service.go" {
		t.Fatalf("expected service.go, got %s", got.File)
	}

	if got.Line != 7 {
		t.Fatalf("expected line 7, got %d", got.Line)
	}
}

func TestPositionRelativeToRootLeavesEmptyFileUnchanged(t *testing.T) {
	position := Position{Line: 3}

	got := positionRelativeToRoot(t.TempDir(), position)

	if got != position {
		t.Fatalf("expected %#v, got %#v", position, got)
	}
}

func TestPositionRelativeToRootSlashNormalizesWhenRootIsEmpty(t *testing.T) {
	position := Position{
		File: filepath.Join("internal", "sherpa", "parse.go"),
		Line: 5,
	}

	got := positionRelativeToRoot("   ", position)

	if got.File != "internal/sherpa/parse.go" {
		t.Fatalf("expected slash-normalized path, got %s", got.File)
	}
}

func TestPositionRelativeToRootDoesNotInventRootRelativePathForOutsideFile(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.go")
	position := Position{
		File: outside,
		Line: 9,
	}

	got := positionRelativeToRoot(root, position)
	want := filepath.ToSlash(outside)

	if got.File != want {
		t.Fatalf("expected %s, got %s", want, got.File)
	}

	if got.Line != 9 {
		t.Fatalf("expected line 9, got %d", got.Line)
	}
}

func writeRootTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
}
