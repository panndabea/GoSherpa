package agentcontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeSymbolBuildsAgentContext(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.Target != "Target" {
		t.Fatalf("target = %q, want Target", report.Target)
	}
	if report.Identity.Package != "." {
		t.Fatalf("identity package = %q, want .", report.Identity.Package)
	}
	if report.Identity.Symbol != "Target" {
		t.Fatalf("identity symbol = %q, want Target", report.Identity.Symbol)
	}
	if report.Identity.Signature != "func Target()" {
		t.Fatalf("identity signature = %q, want func Target()", report.Identity.Signature)
	}
	if report.SourceContext.Position.File != "service.go" {
		t.Fatalf("source context file = %q, want service.go", report.SourceContext.Position.File)
	}
	if len(report.SourceContext.Lines) == 0 {
		t.Fatal("expected source context lines")
	}
	if report.AnalysisMode != AnalysisModeAST {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeAST)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if len(report.Limitations) == 0 {
		t.Fatal("expected limitations")
	}
	if len(report.Callers) != 1 || report.Callers[0].Name != "Entry" {
		t.Fatalf("expected Entry caller, got %#v", report.Callers)
	}
	if len(report.Callees) != 1 || report.Callees[0].Name != "Helper" {
		t.Fatalf("expected Helper callee, got %#v", report.Callees)
	}
	if len(report.RelatedTests) != 1 || report.RelatedTests[0].Name != "TestTarget" {
		t.Fatalf("expected TestTarget related test, got %#v", report.RelatedTests)
	}
}

func TestAnalyzeSymbolIncludesTestCallersWithOption(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if len(report.Callers) != 2 {
		t.Fatalf("expected 2 callers, got %#v", report.Callers)
	}
	if report.Callers[1].Name != "TestTarget" {
		t.Fatalf("expected test caller, got %#v", report.Callers)
	}
}

func writeAgentContextProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() {
	Target()
}

// Target handles the main service step.
func Target() {
	Helper()
}

func Helper() {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service_test.go"), `package app

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)

	return root
}

func writeAgentContextTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
