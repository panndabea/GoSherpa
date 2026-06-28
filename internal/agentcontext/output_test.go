package agentcontext

import (
	"strings"
	"testing"

	explainengine "github.com/supertabaluga/gosherpa/internal/explain"
	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func TestFormat(t *testing.T) {
	report := Report{
		Target: "Target",
		Identity: Identity{
			Target:        "Target",
			Package:       ".",
			Symbol:        "Target",
			Kind:          sherpa.SymbolKindFunction,
			QualifiedName: "example.com/app.Target",
			Signature:     "func Target()",
			Definition:    sherpa.Position{File: "service.go", Line: 7},
		},
		Symbol: sherpa.Symbol{
			Name:     "Target",
			Kind:     sherpa.SymbolKindFunction,
			Package:  ".",
			Position: sherpa.Position{File: "service.go", Line: 7},
		},
		SourceContext: sherpa.SourceContext{
			Position: sherpa.Position{File: "service.go", Line: 7},
			Lines: []sherpa.SourceContextLine{
				{Number: 6, Text: "// Target handles the main service step."},
				{Number: 7, Text: "func Target() {", Target: true},
				{Number: 8, Text: "\tHelper()"},
			},
		},
		Purpose: "Target handles the main service step.",
		Risk: explainengine.RiskSummary{
			Level: "medium",
		},
		ArchitectureRole: explainengine.ArchitectureRole{
			Role: "connector",
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
		AnalysisMode: AnalysisModeAST,
		Confidence:   ConfidenceMedium,
		Limitations: []string{
			"Dynamic dispatch, reflection, and function values are not resolved.",
		},
	}

	output := Format(report)
	for _, want := range []string{
		"CONTEXT",
		"TARGET",
		"Target (function)",
		"Package: .",
		"Qualified: example.com/app.Target",
		"Signature: func Target()",
		"DEFINITION",
		"service.go:7",
		"SOURCE",
		"> 7 | func Target() {",
		"PURPOSE",
		"Target handles the main service step.",
		"ANALYSIS",
		"Mode: ast",
		"Confidence: medium",
		"Risk: medium",
		"Architecture role: connector",
		"CALLED BY",
		"Entry",
		"CALLS",
		"Helper",
		"REFERENCES",
		"SUGGESTED TESTS",
		"TestTarget",
		"SUGGESTED COMMANDS",
		"go test .",
		"LIMITATIONS",
		"Dynamic dispatch",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}
