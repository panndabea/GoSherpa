package sherpa

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRepository(t *testing.T) {
	tmp := t.TempDir()

	err := os.WriteFile(
		filepath.Join(tmp, "service.go"),
		[]byte(`
package auth

type UserService struct{}
`),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseRepository(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}
}