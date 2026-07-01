package main

import (
	"strings"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/agentcontext"
	impactengine "github.com/supertabaluga/gosherpa/internal/impact"
	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func TestMainAgentJSONSchemaContracts(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	tests := []struct {
		name       string
		args       []string
		command    string
		target     string
		wantFields map[string]string
		wantArrays []string
	}{
		{
			name:    "callers",
			args:    []string{"callers", "Target", "--json"},
			command: "callers",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode": sherpa.CallAnalysisModeTypechecked,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"callers", "limitations"},
		},
		{
			name:    "callees",
			args:    []string{"callees", "Entry", "--json"},
			command: "callees",
			target:  "Entry",
			wantFields: map[string]string{
				"analysisMode": sherpa.CallAnalysisModeTypechecked,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"callees", "limitations"},
		},
		{
			name:    "explain",
			args:    []string{"explain", "Target", "--json"},
			command: "explain",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeAST,
				"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"references", "callers", "callees", "limitations", "readingOrder"},
		},
		{
			name:    "impact file",
			args:    []string{"impact", "file", "service.go", "--json"},
			command: "impact file",
			target:  "service.go",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"changedFiles", "changedPackages", "affectedInterfaces", "affectedImplementations", "limitations", "affectedTests"},
		},
		{
			name:    "impact package",
			args:    []string{"impact", "package", ".", "--json"},
			command: "impact package",
			target:  ".",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"changedPackages", "affectedInterfaces", "affectedImplementations", "limitations", "affectedTests"},
		},
		{
			name:    "impact symbol",
			args:    []string{"impact", "symbol", "Target", "--json"},
			command: "impact symbol",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"affectedSymbols", "affectedInterfaces", "affectedImplementations", "limitations", "affectedTests"},
		},
		{
			name:    "impact",
			args:    []string{"impact", "Target", "--json"},
			command: "impact",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode": agentcontext.AnalysisModeAST,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"references", "callers", "limitations", "relatedTests"},
		},
		{
			name:    "tests",
			args:    []string{"tests", "Target", "--json"},
			command: "tests",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode": agentcontext.AnalysisModeAST,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"tests", "commands", "limitations"},
		},
		{
			name:    "context symbol",
			args:    []string{"context", "symbol", "Target", "--json"},
			command: "context symbol",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeAST,
				"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"references", "callers", "callees", "limitations", "readingOrder"},
		},
		{
			name:    "context file",
			args:    []string{"context", "file", "service.go", "--json"},
			command: "context file",
			target:  "service.go",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"symbols", "sourceContexts", "affectedInterfaces", "affectedImplementations", "limitations", "affectedTests"},
		},
		{
			name:    "context package",
			args:    []string{"context", "package", ".", "--json"},
			command: "context package",
			target:  ".",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"files", "symbols", "sourceContexts", "affectedInterfaces", "affectedImplementations", "limitations", "affectedTests"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"gosherpa", "--root", tmp}, test.args...)
			result := runMainTest(t, args)

			if result.ExitCode != exitSuccess {
				t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
			}

			if result.Stderr != "" {
				t.Fatalf("expected empty stderr, got %q", result.Stderr)
			}

			payload := decodeMainTestJSON(t, result.Stdout)
			data := assertMainTestJSONEnvelope(t, payload, tmp, test.command, test.target, "example.com/app")

			for field, want := range test.wantFields {
				if data[field] != want {
					t.Fatalf("expected data.%s %q, got %v", field, want, data[field])
				}
			}

			for _, field := range test.wantArrays {
				if _, ok := data[field].([]any); !ok {
					t.Fatalf("expected data.%s to be a JSON array, got %T", field, data[field])
				}
			}

			if _, ok := data["warnings"]; ok {
				t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
			}

			if strings.Contains(result.Stdout, strings.ToUpper(test.command)) {
				t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
			}
		})
	}
}
