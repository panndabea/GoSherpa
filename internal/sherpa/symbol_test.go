package sherpa

import "testing"

func TestSymbolRepresentsMethod(t *testing.T) {
	s := Symbol{
		Name: "Create",
		Kind: SymbolKindMethod,
		Position: Position{
			File: "internal/auth/service.go",
			Line: 25,
		},
		Receiver: "UserService",
	}

	if s.Name != "Create" {
		t.Fatalf("expected Create, got %s", s.Name)
	}

	if s.Kind != SymbolKindMethod {
		t.Fatalf("expected method kind")
	}

	if s.Receiver != "UserService" {
		t.Fatalf("expected UserService receiver")
	}
}