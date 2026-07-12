package impact

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	_ "unsafe"

	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

//go:linkname impactTestLoadSemanticContextRepository github.com/panndabea/GoSherpa/internal/sherpa.loadSemanticContextRepository
var impactTestLoadSemanticContextRepository func(string, semantics.LoadOptions) (semantics.Repository, error)

func TestAnalyzeSymbolPropagatesTestAnalysisWarnings(t *testing.T) {
	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "service.go"), `package service

func Target() {}
`)
	writeImpactTestFile(t, filepath.Join(root, "service_test.go"), `package service

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)

	original := impactTestLoadSemanticContextRepository
	impactTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		return semantics.Repository{}, errors.New("loader failed")
	}
	defer func() {
		impactTestLoadSemanticContextRepository = original
	}()

	report, err := AnalyzeSymbolWithOptions(root, "Target", AnalyzerOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if report.TestAnalysisMode != sherpa.TestAnalysisModeAST {
		t.Fatalf("expected ast test analysis fallback, got %q", report.TestAnalysisMode)
	}
	if len(report.Warnings) == 0 || !strings.Contains(strings.Join(report.Warnings, "\n"), "typechecked test reference analysis unavailable: loader failed") {
		t.Fatalf("expected propagated test analysis warning, got %#v", report.Warnings)
	}
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{".:TestTarget"})
}
