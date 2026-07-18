package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/panndabea/GoSherpa/internal/agentcontext"
	agentworkflow "github.com/panndabea/GoSherpa/internal/agentworkflow"
	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
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
			name:    "analyze",
			args:    []string{"analyze", "--json"},
			command: "analyze",
			target:  ".",
			wantFields: map[string]string{
				"analysisMode": agentcontext.AnalysisModeTypecheckedAST,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"buildTags", "packages", "importantSymbols", "entrypoints", "hotspots", "limitations", "suggestions"},
		},
		{
			name:    "callers",
			args:    []string{"callers", "Target", "--json"},
			command: "callers",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode": sherpa.CallAnalysisModeTypechecked,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"callers", "possibleCalls", "limitations"},
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
			wantArrays: []string{"callees", "possibleCalls", "limitations"},
		},
		{
			name:    "entrypoints",
			args:    []string{"entrypoints", "Target", "--json"},
			command: "entrypoints",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode": sherpa.CallAnalysisModeTypechecked,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"entrypoints", "limitations"},
		},
		{
			name:    "explain",
			args:    []string{"explain", "Target", "--json"},
			command: "explain",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"symbolAnalysisMode":    explainengine.SymbolAnalysisModeTypecheckedAST,
				"referenceAnalysisMode": sherpa.ReferenceAnalysisModeTypechecked,
				"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeTypecheckedAST,
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
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeAST,
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
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeAST,
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
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"referenceAnalysisMode": sherpa.ReferenceAnalysisModeTypechecked,
				"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeTypecheckedAST,
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
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"referenceAnalysisMode": sherpa.ReferenceAnalysisModeTypechecked,
				"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeTypecheckedAST,
				"confidence":            agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"references", "callers", "limitations", "relatedTests"},
		},
		{
			name:    "tests",
			args:    []string{"tests", "Target", "--json"},
			command: "tests",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode": agentcontext.AnalysisModeTypecheckedAST,
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"tests", "commands", "limitations"},
		},
		{
			name:    "doctor",
			args:    []string{"doctor", "--json"},
			command: "doctor",
			target:  ".",
			wantFields: map[string]string{
				"analysisMode": "typechecked",
				"confidence":   agentcontext.ConfidenceMedium,
			},
			wantArrays: []string{"buildTags", "limitations", "suggestions"},
		},
		{
			name:    "context symbol",
			args:    []string{"context", "symbol", "Target", "--json"},
			command: "context symbol",
			target:  "Target",
			wantFields: map[string]string{
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"referenceAnalysisMode": sherpa.ReferenceAnalysisModeTypechecked,
				"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeTypecheckedAST,
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
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeAST,
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
				"analysisMode":          agentcontext.AnalysisModeTypecheckedAST,
				"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
				"testAnalysisMode":      agentcontext.AnalysisModeAST,
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

func TestMainAdditionalAgentJSONMetadataContracts(t *testing.T) {
	tests := []struct {
		name    string
		root    func(t *testing.T) string
		args    []string
		command string
		target  string
	}{
		{
			name: "architecture",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"architecture", "--json"},
			command: "architecture",
			target:  ".",
		},
		{
			name: "risk",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"risk", "--json"},
			command: "risk",
			target:  ".",
		},
		{
			name: "refs",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"refs", "Target", "--json"},
			command: "refs",
			target:  "Target",
		},
		{
			name: "symbol",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"symbol", "Target", "--json"},
			command: "symbol",
			target:  "Target",
		},
		{
			name: "symbols",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"symbols", "--json"},
			command: "symbols",
			target:  "",
		},
		{
			name: "search",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"search", "Target", "--json"},
			command: "search",
			target:  "Target",
		},
		{
			name: "packages",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"packages", "--json"},
			command: "packages",
			target:  "",
		},
		{
			name: "deps",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"deps", ".", "--json"},
			command: "deps",
			target:  ".",
		},
		{
			name: "deps all",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"deps", "--all", "--json"},
			command: "deps",
			target:  "all",
		},
		{
			name: "path",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"path", "Entry", "Target", "--json"},
			command: "path",
			target:  "Entry -> Target",
		},
		{
			name: "paths",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"paths", "Entry", "Target", "--limit", "2", "--json"},
			command: "paths",
			target:  "Entry -> Target",
		},
		{
			name: "impact diff",
			root: func(t *testing.T) string {
				return writeMainPRDiffProject(t)
			},
			args:    []string{"impact", "diff", "--base", "HEAD", "--json"},
			command: "impact diff",
			target:  "HEAD",
		},
		{
			name: "implementers",
			root: func(t *testing.T) string {
				return writeMainInterfaceProject(t)
			},
			args:    []string{"implementers", "./internal/auth.Authenticator", "--json"},
			command: "implementers",
			target:  "./internal/auth.Authenticator",
		},
		{
			name: "interfaces",
			root: func(t *testing.T) string {
				return writeMainInterfaceProject(t)
			},
			args:    []string{"interfaces", "./internal/jwt.JWTAuthenticator", "--json"},
			command: "interfaces",
			target:  "./internal/jwt.JWTAuthenticator",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := test.root(t)
			args := append([]string{"gosherpa", "--root", root}, test.args...)
			result := runMainTest(t, args)

			if result.ExitCode != exitSuccess {
				t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
			}
			if result.Stderr != "" {
				t.Fatalf("expected empty stderr, got %q", result.Stderr)
			}

			payload := decodeMainTestJSON(t, result.Stdout)
			data := assertMainTestJSONEnvelope(t, payload, root, test.command, test.target, "example.com/app")
			assertMainTestAgentMetadataContract(t, payload, data)
		})
	}
}

func TestMainInterfaceJSONSchemaContract(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "interface", "./internal/auth.Authenticator", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "interface", "./internal/auth.Authenticator", "example.com/app")

	for field, want := range map[string]string{
		"analysisMode":            impactengine.InterfaceAnalysisModeTypechecked,
		"referenceAnalysisMode":   sherpa.ReferenceAnalysisModeTypechecked,
		"methodUsageAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
		"confidence":              agentcontext.ConfidenceMedium,
	} {
		if data[field] != want {
			t.Fatalf("expected %s %q, got %v", field, want, data[field])
		}
	}

	for _, key := range []string{"methods", "implementers", "references", "limitations"} {
		value, ok := data[key].([]any)
		if !ok {
			t.Fatalf("expected %s array, got %T", key, data[key])
		}
		if len(value) == 0 {
			t.Fatalf("expected non-empty %s array", key)
		}
	}
}

func TestMainContextDiffJSONSchemaContract(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "diff", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context diff", "HEAD", "example.com/app")

	wantFields := map[string]string{
		"analysisMode":          agentcontext.AnalysisModeDiffTypechecked,
		"referenceAnalysisMode": sherpa.ReferenceAnalysisModeTypechecked,
		"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
		"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
		"testAnalysisMode":      sherpa.TestAnalysisModeTypecheckedAST,
		"confidence":            agentcontext.ConfidenceMedium,
	}
	for field, want := range wantFields {
		if data[field] != want {
			t.Fatalf("expected data.%s %q, got %v", field, want, data[field])
		}
	}

	for _, field := range []string{
		"changedFiles",
		"changedPackages",
		"affectedPackages",
		"affectedSymbols",
		"affectedInterfaces",
		"affectedImplementations",
		"affectedTests",
		"testCommands",
		"limitations",
		"readingOrder",
	} {
		if _, ok := data[field].([]any); !ok {
			t.Fatalf("expected data.%s to be a JSON array, got %T", field, data[field])
		}
	}
	assertMainTestTestPlanContract(t, data, "testPlan")

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "CONTEXT DIFF") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainAgentContextJSONSchemaContract(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "agent", "context", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "agent context", "HEAD", "example.com/app")

	wantFields := map[string]string{
		"analysisMode": agentcontext.AnalysisModeDiffTypechecked,
		"confidence":   agentcontext.ConfidenceMedium,
	}
	for field, want := range wantFields {
		if data[field] != want {
			t.Fatalf("expected data.%s %q, got %v", field, want, data[field])
		}
	}

	for _, field := range []string{
		"buildTags",
		"changedFiles",
		"changedPackages",
		"affectedPackages",
		"affectedSymbols",
		"changedSymbolDetails",
		"readingOrder",
		"testCommands",
		"suggestedCommands",
		"sectionModes",
		"sectionTruncation",
		"limitations",
	} {
		if _, ok := data[field].([]any); !ok {
			t.Fatalf("expected data.%s to be a JSON array, got %T", field, data[field])
		}
	}
	for _, field := range []string{"readiness", "snapshot", "cost", "targetRisk", "possibleRuntimeRelationships", "interfaceSummary", "testPlan"} {
		if _, ok := data[field].(map[string]any); !ok {
			t.Fatalf("expected data.%s to be a JSON object, got %T", field, data[field])
		}
	}
	assertMainTestTestPlanContract(t, data, "testPlan")

	readiness := assertMainTestJSONObject(t, data, "readiness")
	packageLoad := assertMainTestJSONObject(t, readiness, "packageLoad")
	for _, field := range []string{"buildTags", "affectedSections", "warnings", "diagnostics"} {
		if _, ok := packageLoad[field].([]any); !ok {
			t.Fatalf("expected data.readiness.packageLoad.%s to be an array, got %T", field, packageLoad[field])
		}
	}

	snapshot := assertMainTestJSONObject(t, data, "snapshot")
	if snapshot["requested"] != false || snapshot["used"] != false {
		t.Fatalf("expected snapshot to be unrequested and unused, got %#v", snapshot)
	}
	cost := assertMainTestJSONObject(t, data, "cost")
	for _, field := range []string{"relationshipCounts", "limitations"} {
		if _, ok := cost[field].([]any); !ok {
			t.Fatalf("expected data.cost.%s to be an array, got %T", field, cost[field])
		}
	}
	if cost["packageCount"].(float64) <= 0 || cost["goFileCount"].(float64) <= 0 {
		t.Fatalf("expected positive cost package/file counts, got %#v", cost)
	}
	possibleRuntime := assertMainTestJSONObject(t, data, "possibleRuntimeRelationships")
	for _, field := range []string{"counts", "examples", "limitations"} {
		if _, ok := possibleRuntime[field].([]any); !ok {
			t.Fatalf("expected data.possibleRuntimeRelationships.%s to be an array, got %T", field, possibleRuntime[field])
		}
	}
	sectionModes := assertMainTestJSONArray(t, data, "sectionModes")
	if len(sectionModes) == 0 {
		t.Fatal("expected section modes")
	}
	var sawSnapshot bool
	for _, value := range sectionModes {
		mode, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("expected section mode object, got %#v", value)
		}
		if mode["section"] == "snapshot" {
			sawSnapshot = true
			if mode["analysisMode"] != agentworkflow.AnalysisModeLive {
				t.Fatalf("expected live snapshot section mode, got %#v", mode)
			}
		}
	}
	if !sawSnapshot {
		t.Fatalf("expected snapshot section mode, got %#v", sectionModes)
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
	if strings.Contains(result.Stdout, "AGENT CONTEXT") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainContextDiffEmptyJSONSchemaContract(t *testing.T) {
	tmp := t.TempDir()
	initMainTestGitRepository(t, tmp)

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), "package app\n\nfunc Target() {}\n")
	runMainTestGit(t, tmp, "add", ".")
	runMainTestGit(t, tmp, "commit", "-m", "initial")

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "diff", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context diff", "HEAD", "example.com/app")

	if confidence, ok := data["confidence"].(string); !ok || (confidence != agentcontext.ConfidenceMedium && confidence != agentcontext.ConfidenceLow) {
		t.Fatalf("expected context confidence, got %#v", data["confidence"])
	}
	if limitations := assertMainTestJSONArray(t, data, "limitations"); len(limitations) == 0 {
		t.Fatal("expected non-empty limitations")
	}
	for _, field := range []string{
		"changedFiles",
		"changedPackages",
		"affectedPackages",
		"affectedSymbols",
		"changedSymbolDetails",
		"affectedInterfaces",
		"affectedImplementations",
		"affectedTests",
		"testCommands",
		"readingOrder",
	} {
		assertMainTestJSONArrayHasLength(t, data, field, 0)
	}
	assertMainTestTestPlanContract(t, data, "testPlan")
	assertMainTestJSONFieldsAbsent(t, data, "warnings")
}

func TestMainContextVariantRelationshipFieldBoundaries(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	tests := []struct {
		name       string
		root       string
		args       []string
		command    string
		target     string
		absentKeys []string
	}{
		{
			name:       "context file",
			root:       tmp,
			args:       []string{"context", "file", "service.go", "--json"},
			command:    "context file",
			target:     "service.go",
			absentKeys: []string{"references", "referenceAnalysisMode", "callers", "callees", "callAnalysisMode"},
		},
		{
			name:       "context package",
			root:       tmp,
			args:       []string{"context", "package", ".", "--json"},
			command:    "context package",
			target:     ".",
			absentKeys: []string{"references", "referenceAnalysisMode", "callers", "callees", "callAnalysisMode"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"gosherpa", "--root", test.root}, test.args...)
			result := runMainTest(t, args)
			if result.ExitCode != exitSuccess {
				t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
			}
			if result.Stderr != "" {
				t.Fatalf("expected empty stderr, got %q", result.Stderr)
			}

			payload := decodeMainTestJSON(t, result.Stdout)
			data := assertMainTestJSONEnvelope(t, payload, test.root, test.command, test.target, "example.com/app")
			assertMainTestJSONFieldsAbsent(t, data, test.absentKeys...)
		})
	}

	diffRoot := writeMainPRDiffProject(t)
	result := runMainTest(t, []string{"gosherpa", "--root", diffRoot, "context", "diff", "--base", "HEAD", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, diffRoot, "context diff", "HEAD", "example.com/app")
	assertMainTestJSONFieldsAbsent(t, data, "references", "callers", "callees")
}

func TestMainTestsAffectedJSONSchemaContract(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "affected", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "tests affected", "HEAD", "example.com/app")

	wantFields := map[string]string{
		"analysisMode":          agentcontext.AnalysisModeDiffTypechecked,
		"referenceAnalysisMode": sherpa.ReferenceAnalysisModeTypechecked,
		"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
		"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
		"testAnalysisMode":      sherpa.TestAnalysisModeTypecheckedAST,
		"confidence":            agentcontext.ConfidenceMedium,
	}
	for field, want := range wantFields {
		if data[field] != want {
			t.Fatalf("expected data.%s %q, got %v", field, want, data[field])
		}
	}

	for _, field := range []string{"affectedTests", "commands", "limitations"} {
		if _, ok := data[field].([]any); !ok {
			t.Fatalf("expected data.%s to be a JSON array, got %T", field, data[field])
		}
	}

	assertMainTestTestPlanContract(t, data, "testPlan")

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "AFFECTED TESTS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPRJSONSchemaContract(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "pr", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "pr", "HEAD", "example.com/app")

	wantFields := map[string]string{
		"analysisMode":          agentcontext.AnalysisModeDiffTypechecked,
		"referenceAnalysisMode": sherpa.ReferenceAnalysisModeTypechecked,
		"callAnalysisMode":      sherpa.CallAnalysisModeTypechecked,
		"interfaceAnalysisMode": impactengine.InterfaceAnalysisModeTypechecked,
		"testAnalysisMode":      sherpa.TestAnalysisModeTypecheckedAST,
		"confidence":            agentcontext.ConfidenceMedium,
	}
	for field, want := range wantFields {
		if data[field] != want {
			t.Fatalf("expected data.%s %q, got %v", field, want, data[field])
		}
	}

	for _, field := range []string{
		"limitations",
		"changedFiles",
		"changedPackages",
		"changedSymbols",
		"changedSymbolDetails",
		"affectedPackages",
		"affectedInterfaces",
		"affectedImplementations",
		"affectedTests",
		"testCommands",
		"readingOrder",
		"verificationCommands",
	} {
		if _, ok := data[field].([]any); !ok {
			t.Fatalf("expected data.%s to be a JSON array, got %T", field, data[field])
		}
	}

	for _, field := range []string{"risk", "repositoryRisk", "testPlan"} {
		if _, ok := data[field].(map[string]any); !ok {
			t.Fatalf("expected data.%s to be a JSON object, got %T", field, data[field])
		}
	}
	assertMainTestTestPlanContract(t, data, "testPlan")

	repositoryRisk := assertMainTestJSONObject(t, data, "repositoryRisk")
	for _, field := range []string{"limitations", "factors", "packages", "cycles"} {
		if _, ok := repositoryRisk[field].([]any); !ok {
			t.Fatalf("expected data.repositoryRisk.%s to be a JSON array, got %T", field, repositoryRisk[field])
		}
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "PR REVIEW") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainContextDiffJSONLimitContract(t *testing.T) {
	tmp := writeMainContextDiffLimitProject(t)

	result := runMainTest(t, []string{
		"gosherpa",
		"--root", tmp,
		"context", "diff",
		"--base", "HEAD",
		"--max-files", "1",
		"--max-symbols", "1",
		"--max-tests", "1",
		"--json",
	})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context diff", "HEAD", "example.com/app")

	limits, ok := data["limits"].(map[string]any)
	if !ok {
		t.Fatalf("expected limits object, got %T", data["limits"])
	}
	if limits["maxFiles"] != float64(1) || limits["maxSymbols"] != float64(1) || limits["maxTests"] != float64(1) {
		t.Fatalf("unexpected limits: %#v", limits)
	}

	truncated, ok := data["truncated"].(map[string]any)
	if !ok {
		t.Fatalf("expected truncated object, got %T", data["truncated"])
	}
	for _, field := range []string{"changedFiles", "affectedSymbols", "changedSymbolDetails", "affectedTests", "readingOrder"} {
		if truncated[field] == nil {
			t.Fatalf("expected %s truncation, got %#v", field, truncated)
		}
	}

	assertMainTestJSONArrayHasLength(t, data, "changedFiles", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedSymbols", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 1)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 3)

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
}

func writeMainContextDiffLimitProject(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	initMainTestGitRepository(t, tmp)

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "first.go"), `package app

func First() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "first_test.go"), `package app

import "testing"

func TestFirst(t *testing.T) {
	First()
}
`)
	writeMainTestFile(t, filepath.Join(tmp, "second.go"), `package app

func Second() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "second_test.go"), `package app

import "testing"

func TestSecond(t *testing.T) {
	Second()
}
`)
	runMainTestGit(t, tmp, "add", ".")
	runMainTestGit(t, tmp, "commit", "-m", "initial")

	writeMainTestFile(t, filepath.Join(tmp, "first.go"), `package app

func First() {}

func AddedFirst() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "second.go"), `package app

func Second() {}

func AddedSecond() {}
`)

	return tmp
}

func assertMainTestAgentMetadataContract(t *testing.T, payload map[string]any, data map[string]any) {
	t.Helper()

	assertMainTestJSONArray(t, payload, "warnings")
	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	analysisMode, ok := data["analysisMode"].(string)
	if !ok || strings.TrimSpace(analysisMode) == "" {
		t.Fatalf("expected non-empty analysisMode string, got %#v", data["analysisMode"])
	}

	confidence, ok := data["confidence"].(string)
	if !ok || (confidence != agentcontext.ConfidenceMedium && confidence != agentcontext.ConfidenceLow) {
		t.Fatalf("expected confidence %q or %q, got %#v", agentcontext.ConfidenceMedium, agentcontext.ConfidenceLow, data["confidence"])
	}

	limitations := assertMainTestJSONArray(t, data, "limitations")
	if len(limitations) == 0 {
		t.Fatal("expected non-empty limitations array")
	}
}

func assertMainTestTestPlanContract(t *testing.T, data map[string]any, key string) {
	t.Helper()

	testPlan := assertMainTestJSONObject(t, data, key)
	for _, field := range []string{"direct", "related", "contracts", "callerPackages", "fallback"} {
		if _, ok := testPlan[field].([]any); !ok {
			t.Fatalf("expected data.%s.%s to be a JSON array, got %T", key, field, testPlan[field])
		}
	}
}

func assertMainTestJSONFieldsAbsent(t *testing.T, data map[string]any, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := data[key]; ok {
			t.Fatalf("expected data.%s to be absent, got %#v", key, data[key])
		}
	}
}
