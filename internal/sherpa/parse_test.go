package sherpa

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile(t *testing.T) {
	_, err := ParseFile("does-not-exist.go")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFileFindsStruct(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

type UserService struct {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}

	if symbols[0].Name != "UserService" {
		t.Fatalf("expected UserService, got %s", symbols[0].Name)
	}

	if symbols[0].Kind != SymbolKindStruct {
		t.Fatalf("expected struct kind, got %s", symbols[0].Kind)
	}
}

func TestParseFileFindsFunction(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

func CreateUser() {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(symbols))
	}

	if symbols[0].Name != "CreateUser" {
		t.Fatalf("expected CreateUser, got %s", symbols[0].Name)
	}

	if symbols[0].Kind != SymbolKindFunction {
		t.Fatalf("expected function kind, got %s", symbols[0].Kind)
	}
}

func TestParseFileFindsMethod(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

type UserService struct{}

func (s *UserService) Create() {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(symbols))
	}

	method := symbols[1]

	if method.Name != "Create" {
		t.Fatalf("expected Create, got %s", method.Name)
	}

	if method.Kind != SymbolKindMethod {
		t.Fatalf("expected method kind")
	}

	if method.Receiver != "UserService" {
		t.Fatalf("expected UserService receiver, got %s", method.Receiver)
	}
}

func TestParseFileFindsValueMethod(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

type UserService struct{}

func (s UserService) Create() {
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	symbols, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	method := symbols[1]

	if method.Receiver != "UserService" {
		t.Fatalf("expected UserService receiver, got %s", method.Receiver)
	}
}