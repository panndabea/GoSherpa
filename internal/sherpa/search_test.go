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

func TestSearchSymbolsMatchesCaseInsensitivePartialName(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "CreateUser",
			Kind: SymbolKindFunction,
		},
		{
			Name: "DeleteTeam",
			Kind: SymbolKindFunction,
		},
	}

	results := SearchSymbols(symbols, []string{"USER"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Symbol.Name != "CreateUser" {
		t.Fatalf("expected CreateUser, got %s", results[0].Symbol.Name)
	}

	assertSearchTestStrings(t, results[0].MatchedTerms, []string{"user"})
}

func TestSearchSymbolsRequiresAllTerms(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "CreateUser",
			Kind: SymbolKindFunction,
		},
		{
			Name: "CreateTeam",
			Kind: SymbolKindFunction,
		},
	}

	results := SearchSymbols(symbols, []string{"create", "user"})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Symbol.Name != "CreateUser" {
		t.Fatalf("expected CreateUser, got %s", results[0].Symbol.Name)
	}
}

func TestSearchSymbolsRanksExactAndPrefixMatchesFirst(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "CreateUserCommand",
			Kind: SymbolKindStruct,
		},
		{
			Name: "CreateUser",
			Kind: SymbolKindFunction,
		},
		{
			Name: "EnsureCreateUser",
			Kind: SymbolKindFunction,
		},
	}

	results := SearchSymbols(symbols, []string{"CreateUser"})

	assertSearchTestResultNames(t, results, []string{"CreateUser", "CreateUserCommand", "EnsureCreateUser"})

	if results[0].Score <= results[1].Score {
		t.Fatalf("expected exact match score to beat prefix match: %#v", results)
	}
}

func TestSearchSymbolsMatchesMethodDisplayName(t *testing.T) {
	symbols := []Symbol{
		{
			Name:     "Create",
			Kind:     SymbolKindMethod,
			Receiver: "UserService",
		},
		{
			Name: "CreateUserService",
			Kind: SymbolKindFunction,
		},
	}

	results := SearchSymbols(symbols, []string{"service", "create"})

	assertSearchTestResultNames(t, results, []string{"UserService.Create", "CreateUserService"})
}

func TestSearchSymbolsUsesStableSortForTies(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "UpdateUser",
			Kind: SymbolKindFunction,
		},
		{
			Name: "ArchiveUser",
			Kind: SymbolKindFunction,
		},
	}

	results := SearchSymbols(symbols, []string{"user"})

	assertSearchTestResultNames(t, results, []string{"ArchiveUser", "UpdateUser"})
}

func TestSearchSymbolsFiltersByKind(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "UserService",
			Kind: SymbolKindStruct,
		},
		{
			Name: "UserRepository",
			Kind: SymbolKindInterface,
		},
	}

	results := SearchSymbolsWithOptions(symbols, []string{"user"}, SymbolSearchOptions{
		Kind: SymbolKindInterface,
	})

	assertSearchTestResultNames(t, results, []string{"UserRepository"})
}

func TestSearchSymbolsFiltersByPackage(t *testing.T) {
	symbols := []Symbol{
		{
			Name:    "CreateUser",
			Kind:    SymbolKindFunction,
			Package: "./internal/service",
		},
		{
			Name:    "CreateUserHandler",
			Kind:    SymbolKindFunction,
			Package: "./internal/http",
		},
	}

	results := SearchSymbolsWithOptions(symbols, []string{"user"}, SymbolSearchOptions{
		Package: "./internal/service",
	})

	assertSearchTestResultNames(t, results, []string{"CreateUser"})
}

func TestSearchSymbolsFiltersToTests(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "CreateUser",
			Kind: SymbolKindFunction,
		},
		{
			Name: "TestCreateUser",
			Kind: SymbolKindFunction,
		},
	}

	results := SearchSymbolsWithOptions(symbols, []string{"user"}, SymbolSearchOptions{
		TestsOnly: true,
	})

	assertSearchTestResultNames(t, results, []string{"TestCreateUser"})
}

func TestFilterSymbolsUsesKindPackageAndTestOptions(t *testing.T) {
	symbols := []Symbol{
		{
			Name:    "CreateUser",
			Kind:    SymbolKindFunction,
			Package: "./internal/service",
		},
		{
			Name:    "TestCreateUser",
			Kind:    SymbolKindFunction,
			Package: "./internal/service",
		},
		{
			Name:    "TestCreateUserHandler",
			Kind:    SymbolKindFunction,
			Package: "./internal/http",
		},
		{
			Name:    "UserRepository",
			Kind:    SymbolKindInterface,
			Package: "./internal/service",
		},
	}

	filtered := FilterSymbols(symbols, SymbolFilterOptions{
		Kind:      SymbolKindFunction,
		Package:   "./internal/service",
		TestsOnly: true,
	})

	if len(filtered) != 1 {
		t.Fatalf("expected 1 symbol, got %d: %#v", len(filtered), filtered)
	}
	if filtered[0].Name != "TestCreateUser" {
		t.Fatalf("expected TestCreateUser, got %s", filtered[0].Name)
	}
}

func TestSearchSymbolsFiltersToTestFiles(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "CreateUserHelper",
			Kind: SymbolKindFunction,
			Position: Position{
				File: "internal/service/service.go",
			},
		},
		{
			Name: "CreateUserHelper",
			Kind: SymbolKindFunction,
			Position: Position{
				File: "internal/service/service_test.go",
			},
		},
	}

	results := SearchSymbolsWithOptions(symbols, []string{"helper"}, SymbolSearchOptions{
		TestsOnly: true,
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %#v", len(results), results)
	}

	if results[0].Symbol.Position.File != "internal/service/service_test.go" {
		t.Fatalf("expected test-file helper, got %s", results[0].Symbol.Position.File)
	}
}

func TestSearchSymbolsLimitsResultsAfterRanking(t *testing.T) {
	symbols := []Symbol{
		{
			Name: "CreateUserCommand",
			Kind: SymbolKindStruct,
		},
		{
			Name: "CreateUser",
			Kind: SymbolKindFunction,
		},
		{
			Name: "EnsureCreateUser",
			Kind: SymbolKindFunction,
		},
	}

	results := SearchSymbolsWithOptions(symbols, []string{"CreateUser"}, SymbolSearchOptions{
		Limit: 2,
	})

	assertSearchTestResultNames(t, results, []string{"CreateUser", "CreateUserCommand"})
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

func assertSearchTestResultNames(t *testing.T, results []SymbolSearchResult, want []string) {
	t.Helper()

	if len(results) != len(want) {
		t.Fatalf("expected %d results, got %d: %#v", len(want), len(results), results)
	}

	var got []string
	for _, result := range results {
		got = append(got, result.Symbol.DisplayName())
	}

	assertSearchTestStrings(t, got, want)
}

func assertSearchTestStrings(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
