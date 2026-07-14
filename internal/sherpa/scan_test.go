package sherpa

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGoFiles(t *testing.T) {
	tmp := t.TempDir()

	err := os.WriteFile(
		filepath.Join(tmp, "main.go"),
		[]byte("package main"),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(
		filepath.Join(tmp, "README.md"),
		[]byte("# hello"),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	files, err := FindGoFiles(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 go file, got %d", len(files))
	}
}

func TestFindGoFilesSkipsNestedModules(t *testing.T) {
	tmp := t.TempDir()
	writeScanTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/root\n")
	writeScanTestFile(t, filepath.Join(tmp, "main.go"), "package main")
	writeScanTestFile(t, filepath.Join(tmp, "pkg", "pkg.go"), "package pkg")
	writeScanTestFile(t, filepath.Join(tmp, "nested", "go.mod"), "module example.com/nested\n")
	writeScanTestFile(t, filepath.Join(tmp, "nested", "nested.go"), "package nested")
	writeScanTestFile(t, filepath.Join(tmp, "nested", "child", "child.go"), "package child")

	files, err := FindGoFiles(tmp)
	if err != nil {
		t.Fatal(err)
	}

	got := scanTestFileSet(files)
	assertScanTestHasFile(t, got, filepath.Join(tmp, "main.go"))
	assertScanTestHasFile(t, got, filepath.Join(tmp, "pkg", "pkg.go"))
	assertScanTestNoFile(t, got, filepath.Join(tmp, "nested", "nested.go"))
	assertScanTestNoFile(t, got, filepath.Join(tmp, "nested", "child", "child.go"))
}

func TestFindGoFilesIncludesGoWorkModulesWhenRootHasGoMod(t *testing.T) {
	tmp := t.TempDir()
	writeScanTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/root\n")
	writeScanTestFile(t, filepath.Join(tmp, "go.work"), `go 1.24

use (
	.
	./service
)
`)
	writeScanTestFile(t, filepath.Join(tmp, "main.go"), "package root")
	writeScanTestFile(t, filepath.Join(tmp, "service", "go.mod"), "module example.com/service\n")
	writeScanTestFile(t, filepath.Join(tmp, "service", "service.go"), "package service")
	writeScanTestFile(t, filepath.Join(tmp, "unlisted", "go.mod"), "module example.com/unlisted\n")
	writeScanTestFile(t, filepath.Join(tmp, "unlisted", "unlisted.go"), "package unlisted")

	files, err := FindGoFiles(tmp)
	if err != nil {
		t.Fatal(err)
	}

	got := scanTestFileSet(files)
	assertScanTestHasFile(t, got, filepath.Join(tmp, "main.go"))
	assertScanTestHasFile(t, got, filepath.Join(tmp, "service", "service.go"))
	assertScanTestNoFile(t, got, filepath.Join(tmp, "unlisted", "unlisted.go"))
}

func TestFindGoFilesKeepsNestedGoModWhenRootIsNotModule(t *testing.T) {
	tmp := t.TempDir()
	writeScanTestFile(t, filepath.Join(tmp, "main.go"), "package main")
	writeScanTestFile(t, filepath.Join(tmp, "nested", "go.mod"), "module example.com/nested\n")
	writeScanTestFile(t, filepath.Join(tmp, "nested", "nested.go"), "package nested")

	files, err := FindGoFiles(tmp)
	if err != nil {
		t.Fatal(err)
	}

	got := scanTestFileSet(files)
	assertScanTestHasFile(t, got, filepath.Join(tmp, "main.go"))
	assertScanTestHasFile(t, got, filepath.Join(tmp, "nested", "nested.go"))
}

func writeScanTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func scanTestFileSet(files []string) map[string]struct{} {
	values := make(map[string]struct{}, len(files))
	for _, file := range files {
		values[filepath.Clean(file)] = struct{}{}
	}

	return values
}

func assertScanTestHasFile(t *testing.T, files map[string]struct{}, file string) {
	t.Helper()

	if _, ok := files[filepath.Clean(file)]; !ok {
		t.Fatalf("expected files to include %s, got %#v", file, files)
	}
}

func assertScanTestNoFile(t *testing.T, files map[string]struct{}, file string) {
	t.Helper()

	if _, ok := files[filepath.Clean(file)]; ok {
		t.Fatalf("expected files to omit %s, got %#v", file, files)
	}
}
