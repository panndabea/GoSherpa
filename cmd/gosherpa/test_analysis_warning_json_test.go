package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	_ "unsafe"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
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

func TestAnalyzePRSharesSemanticDiffSession(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	original := mainTestLoadSemanticContextRepository
	repositoryLoads := 0
	testRepositoryLoads := 0
	mainTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if options.IncludeTests {
			testRepositoryLoads++
		} else {
			repositoryLoads++
		}

		return original(root, options)
	}
	defer func() {
		mainTestLoadSemanticContextRepository = original
	}()

	report, err := analyzePR(tmp, "HEAD", prAnalyzeOptions{})
	if err != nil {
		t.Fatalf("analyzePR returned error: %v", err)
	}

	if report.AnalysisMode != agentcontext.AnalysisModeDiffTypechecked {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, agentcontext.AnalysisModeDiffTypechecked, report.Warnings)
	}
	if report.TestAnalysisMode != sherpa.TestAnalysisModeTypecheckedAST {
		t.Fatalf("test analysis mode = %q, want %s", report.TestAnalysisMode, sherpa.TestAnalysisModeTypecheckedAST)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
	if testRepositoryLoads != 1 {
		t.Fatalf("expected one shared semantic test repository load, got %d", testRepositoryLoads)
	}
}

func TestAnalyzeImpactSubcommandSharesSemanticSession(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	original := mainTestLoadSemanticContextRepository
	repositoryLoads := 0
	testRepositoryLoads := 0
	mainTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if options.IncludeTests {
			testRepositoryLoads++
		} else {
			repositoryLoads++
		}

		return original(root, options)
	}
	defer func() {
		mainTestLoadSemanticContextRepository = original
	}()

	report, err := analyzeImpactSubcommand(tmp, "symbol", "Target", nil)
	if err != nil {
		t.Fatalf("analyzeImpactSubcommand returned error: %v", err)
	}

	if report.ReferenceAnalysisMode != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s with warnings %#v", report.ReferenceAnalysisMode, sherpa.ReferenceAnalysisModeTypechecked, report.Warnings)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if report.TestAnalysisMode != sherpa.TestAnalysisModeTypecheckedAST {
		t.Fatalf("test analysis mode = %q, want %s", report.TestAnalysisMode, sherpa.TestAnalysisModeTypecheckedAST)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
	if testRepositoryLoads != 1 {
		t.Fatalf("expected one shared semantic test repository load, got %d", testRepositoryLoads)
	}
}

func TestMainImpactCommandSharesSemanticSession(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	original := mainTestLoadSemanticContextRepository
	repositoryLoads := 0
	testRepositoryLoads := 0
	mainTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if options.IncludeTests {
			testRepositoryLoads++
		} else {
			repositoryLoads++
		}

		return original(root, options)
	}
	defer func() {
		mainTestLoadSemanticContextRepository = original
	}()

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "Target", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "impact", "Target", "example.com/app")
	if data["referenceAnalysisMode"] != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %v, want %s", data["referenceAnalysisMode"], sherpa.ReferenceAnalysisModeTypechecked)
	}
	if data["callAnalysisMode"] != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %v, want %s", data["callAnalysisMode"], sherpa.CallAnalysisModeTypechecked)
	}
	if data["testAnalysisMode"] != sherpa.TestAnalysisModeTypecheckedAST {
		t.Fatalf("test analysis mode = %v, want %s", data["testAnalysisMode"], sherpa.TestAnalysisModeTypecheckedAST)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
	if testRepositoryLoads != 1 {
		t.Fatalf("expected one shared semantic test repository load, got %d", testRepositoryLoads)
	}
}

func TestMainImpactDiffCommandSharesSemanticSession(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	original := mainTestLoadSemanticContextRepository
	repositoryLoads := 0
	testRepositoryLoads := 0
	mainTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if options.IncludeTests {
			testRepositoryLoads++
		} else {
			repositoryLoads++
		}

		return original(root, options)
	}
	defer func() {
		mainTestLoadSemanticContextRepository = original
	}()

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "diff", "--base", "HEAD", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "impact diff", "HEAD", "example.com/app")
	if data["analysisMode"] != agentcontext.AnalysisModeDiffTypechecked {
		t.Fatalf("analysis mode = %v, want %s", data["analysisMode"], agentcontext.AnalysisModeDiffTypechecked)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
	if testRepositoryLoads != 1 {
		t.Fatalf("expected one shared semantic test repository load, got %d", testRepositoryLoads)
	}
}

func TestMainTestsAffectedCommandSharesSemanticSession(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	original := mainTestLoadSemanticContextRepository
	repositoryLoads := 0
	testRepositoryLoads := 0
	mainTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if options.IncludeTests {
			testRepositoryLoads++
		} else {
			repositoryLoads++
		}

		return original(root, options)
	}
	defer func() {
		mainTestLoadSemanticContextRepository = original
	}()

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "affected", "--base", "HEAD", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "tests affected", "HEAD", "example.com/app")
	if data["analysisMode"] != agentcontext.AnalysisModeDiffTypechecked {
		t.Fatalf("analysis mode = %v, want %s", data["analysisMode"], agentcontext.AnalysisModeDiffTypechecked)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
	if testRepositoryLoads != 1 {
		t.Fatalf("expected one shared semantic test repository load, got %d", testRepositoryLoads)
	}
}

func TestMainInterfaceCommandSharesSemanticSession(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	original := mainTestLoadSemanticContextRepository
	repositoryLoads := 0
	testRepositoryLoads := 0
	mainTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if options.IncludeTests {
			testRepositoryLoads++
		} else {
			repositoryLoads++
		}

		return original(root, options)
	}
	defer func() {
		mainTestLoadSemanticContextRepository = original
	}()

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "interface", "./internal/auth.Authenticator", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "interface", "./internal/auth.Authenticator", "example.com/app")
	if data["analysisMode"] != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("analysis mode = %v, want %s", data["analysisMode"], impactengine.InterfaceAnalysisModeTypechecked)
	}
	if data["referenceAnalysisMode"] != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %v, want %s", data["referenceAnalysisMode"], sherpa.ReferenceAnalysisModeTypechecked)
	}
	if data["methodUsageAnalysisMode"] != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("method usage analysis mode = %v, want %s", data["methodUsageAnalysisMode"], impactengine.InterfaceAnalysisModeTypechecked)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
	if testRepositoryLoads != 0 {
		t.Fatalf("expected no shared semantic test repository loads, got %d", testRepositoryLoads)
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
