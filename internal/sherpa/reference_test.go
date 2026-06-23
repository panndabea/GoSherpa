package sherpa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindReferences(t *testing.T) {
	tmp := t.TempDir()

	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

func ParseFile() {
}

func Run() {
	ParseFile()
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	refs, err := FindReferences(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d", len(refs))
	}
}

func TestFindReferencesReturnsRootRelativeFilePositions(t *testing.T) {
	tmp := t.TempDir()

	path := filepath.Join(tmp, "internal", "service", "service.go")

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(path, []byte(`
package service

func ParseFile() {
}

func Run() {
	ParseFile()
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	refs, err := FindReferences(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d", len(refs))
	}

	for _, ref := range refs {
		if ref.Position.File != "internal/service/service.go" {
			t.Fatalf("expected internal/service/service.go, got %s", ref.Position.File)
		}

		if strings.Contains(ref.Position.File, tmp) {
			t.Fatalf("expected root-relative path, got %s", ref.Position.File)
		}
	}
}
