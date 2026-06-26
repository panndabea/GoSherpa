package sherpa

import (
	"fmt"
	"testing"
)

func TestFormatSymbolSearch(t *testing.T) {
	results := []SymbolSearchResult{
		{
			Symbol: Symbol{
				Name:    "CreateUser",
				Kind:    SymbolKindFunction,
				Package: "./internal/service",
				Position: Position{
					File: "internal/service/service.go",
					Line: 12,
				},
			},
			Score:        140,
			MatchedTerms: []string{"user", "create"},
		},
	}

	got := FormatSymbolSearch([]string{"user", "create"}, results)
	want := fmt.Sprintf(`SYMBOL SEARCH

Query: user create
Found 1 match

  %-5d %-10s %-36s %-20s internal/service/service.go:12
`, 140, SymbolKindFunction, "CreateUser", "./internal/service")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatSymbolSearchWithEmptyList(t *testing.T) {
	got := FormatSymbolSearch([]string{"missing"}, nil)
	want := `SYMBOL SEARCH

Query: missing
Found 0 matches

`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
