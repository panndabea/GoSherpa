package explain

import (
	"strings"
	"testing"

	"github.com/panndabea/GoSherpa/internal/sherpa"
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
		Purpose: "Target handles the main service step.",
		Risk: RiskSummary{
			Level: "medium",
			Reasons: []string{
				"Symbol is exported.",
				"Called by 1 function or method.",
			},
		},
		ArchitectureRole: ArchitectureRole{
			Role: "connector",
			Reasons: []string{
				"Symbol is called by upstream code and calls downstream code.",
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
		ReadingOrder: []ReadingStep{
			{
				Title:    "Definition",
				Reason:   "Start with the symbol declaration and nearby implementation.",
				Position: sherpa.Position{File: "service.go", Line: 7},
			},
			{
				Title:    "Test: TestTarget",
				Reason:   "Check expected behavior and regression coverage.",
				Position: sherpa.Position{File: "service_test.go", Line: 5},
			},
		},
	}

	output := Format(report)
	for _, want := range []string{
		"EXPLAIN",
		"TARGET",
		"Target (function)",
		"DEFINITION",
		"service.go:7",
		"PURPOSE",
		"Target handles the main service step.",
		"RISK",
		"medium",
		"ARCHITECTURE ROLE",
		"connector",
		"READING ORDER",
		"Definition - service.go:7",
		"Test: TestTarget",
		"CALLED BY",
		"Entry",
		"CALLS",
		"Helper",
		"REFERENCES",
		"SUGGESTED TESTS",
		"TestTarget",
		"TEST PLAN",
		"go test .",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}
