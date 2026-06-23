package sherpa

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseRepositoryReturnsRootRelativeFilePositions(t *testing.T) {
	tmp := t.TempDir()

	err := os.MkdirAll(filepath.Join(tmp, "internal", "service"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(
		filepath.Join(tmp, "internal", "service", "service.go"),
		[]byte(`
package service

func Run() {}
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

	want := "internal/service/service.go"
	if symbols[0].Position.File != want {
		t.Fatalf("expected %s, got %s", want, symbols[0].Position.File)
	}

	if strings.Contains(symbols[0].Position.File, tmp) {
		t.Fatalf("expected root-relative path, got %s", symbols[0].Position.File)
	}
}
