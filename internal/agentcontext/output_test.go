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

func TestFormatDiff(t *testing.T) {
	report := DiffReport{
		Target:  "HEAD",
		Base:    "HEAD",
		Purpose: "Diff changes 1 file across 1 Go package.",
		Risk: explainengine.RiskSummary{
			Level: "medium",
			Reasons: []string{
				"Impact reaches 2 packages.",
			},
		},
		ChangedFiles:     []string{"internal/auth/session.go"},
		ChangedPackages:  []string{"./internal/auth"},
		AffectedPackages: []string{"./internal/api", "./internal/auth"},
		AffectedSymbols:  []string{"NewSession"},
		AffectedTests: []sherpa.RelatedTest{
			{Name: "TestSession", Position: sherpa.Position{File: "internal/auth/session_test.go", Line: 5}},
		},
		TestCommands: []string{"go test ./internal/auth"},
		ReadingOrder: []explainengine.ReadingStep{
			{
				Title:  "Changed file: internal/auth/session.go",
				Reason: "Start with the files changed by the diff.",
				Position: sherpa.Position{
					File: "internal/auth/session.go",
					Line: 1,
				},
			},
		},
		AnalysisMode: AnalysisModeDiff,
		Confidence:   ConfidenceMedium,
		Limitations: []string{
			"Diff context uses git diff plus syntax-level repository analysis, not full module loading.",
		},
	}

	output := FormatDiff(report)
	for _, want := range []string{
		"CONTEXT DIFF",
		"BASE",
		"HEAD",
		"PURPOSE",
		"Diff changes 1 file",
		"ANALYSIS",
		"Mode: git-diff+ast",
		"Confidence: medium",
		"Risk: medium",
		"Impact reaches 2 packages.",
		"CHANGED FILES",
		"internal/auth/session.go",
		"CHANGED PACKAGES",
		"./internal/auth",
		"AFFECTED SYMBOLS",
		"NewSession",
		"AFFECTED PACKAGES",
		"./internal/api",
		"AFFECTED TESTS",
		"TestSession",
		"SUGGESTED COMMANDS",
		"go test ./internal/auth",
		"READING ORDER",
		"Changed file: internal/auth/session.go",
		"LIMITATIONS",
		"Diff context uses git diff",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}
