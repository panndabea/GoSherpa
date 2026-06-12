package sherpa

import "testing"

func TestFindSymbol(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "UserService",
			Kind: SymbolKindStruct,
		},
		{
			Name: "CreateUser",
			Kind: SymbolKindFunction,
		},
	}

	symbol := FindSymbol(symbols, "UserService")

	if symbol == nil {
		t.Fatal("expected symbol")
	}

	if symbol.Name != "UserService" {
		t.Fatalf("expected UserService, got %s", symbol.Name)
	}
}

func TestFindSymbolReturnsPosition(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "UserService",
			Kind: SymbolKindStruct,
			Position: Position{
				File: "internal/auth/service.go",
				Line: 12,
			},
		},
	}

	symbol := FindSymbol(symbols, "UserService")

	if symbol == nil {
		t.Fatal("expected symbol")
	}

	if symbol.Position.File != "internal/auth/service.go" {
		t.Fatalf(
			"expected file path, got %s",
			symbol.Position.File,
		)
	}

	if symbol.Position.Line != 12 {
		t.Fatalf(
			"expected line 12, got %d",
			symbol.Position.Line,
		)
	}
}