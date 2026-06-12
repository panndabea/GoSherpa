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

