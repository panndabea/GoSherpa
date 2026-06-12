package sherpa

import (
	"os"
	"path/filepath"
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