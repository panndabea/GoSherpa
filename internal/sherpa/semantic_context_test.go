package sherpa

import (
	"path/filepath"
	"testing"

	"github.com/panndabea/GoSherpa/internal/semantics"
)

func TestSemanticContextSharesRepositoryForReferencesAndCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {
	Target()
}

func Target() {
	Helper()
}

func Helper() {}
`)

	original := loadSemanticContextRepository
	loads := 0
	loadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		loads++
		return original(root, options)
	}
	defer func() {
		loadSemanticContextRepository = original
	}()

	context, err := NewSemanticContext(tmp, SemanticContextOptions{})
	if err != nil {
		t.Fatal(err)
	}

	references, err := FindReferenceReportWithContext(context, "Target", ReferenceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if references.AnalysisMode != ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s", references.AnalysisMode, ReferenceAnalysisModeTypechecked)
	}
	if len(references.References) != 2 {
		t.Fatalf("expected definition and caller reference, got %#v", references.References)
	}

	callers, err := FindCallersWithContext(context, "Target", CallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if callers.AnalysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("caller analysis mode = %q, want %s", callers.AnalysisMode, CallAnalysisModeTypechecked)
	}
	if len(callers.Callers) != 1 || callers.Callers[0].Name != "Entry" {
		t.Fatalf("expected Entry caller, got %#v", callers.Callers)
	}

	callees, err := FindCalleesWithContext(context, "Target", CallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if callees.AnalysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("callee analysis mode = %q, want %s", callees.AnalysisMode, CallAnalysisModeTypechecked)
	}
	if len(callees.Callees) != 1 || callees.Callees[0].Name != "Helper" {
		t.Fatalf("expected Helper callee, got %#v", callees.Callees)
	}

	if loads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", loads)
	}
}
