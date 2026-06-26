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

func TestFindSymbolTargetUsesPackageQualifiedTarget(t *testing.T) {
	symbols := []Symbol{
		{
			Name:    "Run",
			Kind:    SymbolKindFunction,
			Package: "./internal/worker",
		},
		{
			Name:    "Run",
			Kind:    SymbolKindFunction,
			Package: "./internal/service",
		},
	}

	symbol, err := FindSymbolTarget("", symbols, "./internal/service.Run")
	if err != nil {
		t.Fatal(err)
	}

	if symbol.Package != "./internal/service" {
		t.Fatalf("expected service package, got %s", symbol.Package)
	}
}

func TestFindSymbolTargetPackageQualifiedNameDoesNotMatchMethods(t *testing.T) {
	symbols := []Symbol{
		{
			Name:    "Symbol",
			Kind:    SymbolKindStruct,
			Package: "./internal/sherpa",
		},
		{
			Name:     "Symbol",
			Kind:     SymbolKindMethod,
			Package:  "./internal/sherpa",
			Receiver: "callTarget",
		},
	}

	symbol, err := FindSymbolTarget("", symbols, "./internal/sherpa.Symbol")
	if err != nil {
		t.Fatal(err)
	}

	if symbol.Kind != SymbolKindStruct {
		t.Fatalf("expected struct symbol, got %s", symbol.Kind)
	}
}

func TestFindSymbolTargetReportsAmbiguousTargets(t *testing.T) {
	symbols := []Symbol{
		{
			Name:    "Run",
			Kind:    SymbolKindFunction,
			Package: "./internal/worker",
			Position: Position{
				File: "internal/worker/worker.go",
				Line: 5,
			},
		},
		{
			Name:    "Run",
			Kind:    SymbolKindFunction,
			Package: "./internal/service",
			Position: Position{
				File: "internal/service/service.go",
				Line: 7,
			},
		},
	}

	_, err := FindSymbolTarget("", symbols, "Run")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}

	ambiguity, ok := err.(*AmbiguousTargetError)
	if !ok {
		t.Fatalf("expected ambiguity error, got %T", err)
	}

	if len(ambiguity.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(ambiguity.Candidates))
	}
}
