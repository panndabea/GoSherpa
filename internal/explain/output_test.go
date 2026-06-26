package explain

import (
	"strings"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func TestFormat(t *testing.T) {
	report := Report{
		Target: "Target",
		Symbol: sherpa.Symbol{
			Name: "Target",
			Kind: sherpa.SymbolKindFunction,
			Position: sherpa.Position{
				File: "service.go",
				Line: 7,
			},
		},
		Callers: []sherpa.Caller{
			{Name: "Entry", Position: sherpa.Position{File: "service.go", Line: 3}},
		},
		Callees: []sherpa.Callee{
			{Name: "Helper", Position: sherpa.Position{File: "service.go", Line: 8}},
		},
		References: []sherpa.Reference{
			{Position: sherpa.Position{File: "service_test.go", Line: 6}},
		},
		AffectedPackages: []string{"."},
		RelatedTests: []sherpa.RelatedTest{
			{Name: "TestTarget", Position: sherpa.Position{File: "service_test.go", Line: 5}, DirectReference: true},
		},
		TestCommands: []string{"go test ."},
	}

	output := Format(report)
	for _, want := range []string{
		"EXPLAIN",
		"TARGET",
		"Target (function)",
		"DEFINITION",
		"service.go:7",
		"CALLED BY",
		"Entry",
		"CALLS",
		"Helper",
		"REFERENCES",
		"SUGGESTED TESTS",
		"TestTarget",
		"SUGGESTED COMMANDS",
		"go test .",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}
