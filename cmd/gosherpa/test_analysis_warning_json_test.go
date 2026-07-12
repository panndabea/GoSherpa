package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	_ "unsafe"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

//go:linkname mainTestLoadSemanticContextRepository github.com/panndabea/GoSherpa/internal/sherpa.loadSemanticContextRepository
var mainTestLoadSemanticContextRepository func(string, semantics.LoadOptions) (semantics.Repository, error)

func TestMainContextSymbolJSONPropagatesTestAnalysisWarnings(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	original := mainTestLoadSemanticContextRepository
	mainTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		return semantics.Repository{}, errors.New("loader failed")
	}
	defer func() {
		mainTestLoadSemanticContextRepository = original
	}()

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "symbol", "Target", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	if payload["schemaVersion"] != float64(jsonSchemaVersion) {
		t.Fatalf("expected schemaVersion %d, got %v", jsonSchemaVersion, payload["schemaVersion"])
	}
	if payload["command"] != "context symbol" {
		t.Fatalf("expected context symbol command, got %v", payload["command"])
	}
	if payload["target"] != "Target" {
		t.Fatalf("expected Target target, got %v", payload["target"])
	}
	if payload["root"] != filepath.Clean(tmp) {
		t.Fatalf("expected root %s, got %v", filepath.Clean(tmp), payload["root"])
	}
	if payload["modulePath"] != "example.com/app" {
		t.Fatalf("expected module path example.com/app, got %v", payload["modulePath"])
	}
	warnings := assertMainTestJSONArray(t, payload, "warnings")
	if len(warnings) == 0 || !mainTestJSONArrayContainsSubstring(warnings, "typechecked test reference analysis unavailable: loader failed") {
		t.Fatalf("expected test analysis warning in JSON envelope, got %#v", warnings)
	}

	data := assertMainTestJSONObject(t, payload, "data")
	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
	if data["testAnalysisMode"] != sherpa.TestAnalysisModeAST {
		t.Fatalf("expected ast test analysis fallback, got %v", data["testAnalysisMode"])
	}
	if data["confidence"] != agentcontext.ConfidenceLow {
		t.Fatalf("expected low confidence, got %v", data["confidence"])
	}
}

func mainTestJSONArrayContainsSubstring(values []any, substring string) bool {
	for _, value := range values {
		text, ok := value.(string)
		if ok && strings.Contains(text, substring) {
			return true
		}
	}

	return false
}
