package agentcontext

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
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
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if len(report.Limitations) == 0 {
		t.Fatal("expected limitations")
	}
	if !strings.Contains(report.Limitations[0], "typechecked") {
		t.Fatalf("expected typechecked call limitation, got %#v", report.Limitations)
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

func TestAnalyzeSymbolAppliesLimits(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{
		IncludeTests: true,
		Limits: LimitOptions{
			MaxReferences: 1,
			MaxTests:      1,
			SourceRadius:  NewSourceRadius(0),
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.Limits == nil || report.Limits.MaxReferences != 1 || report.Limits.MaxTests != 1 {
		t.Fatalf("expected limits to be recorded, got %#v", report.Limits)
	}
	if report.Limits.SourceRadius == nil || *report.Limits.SourceRadius != 0 {
		t.Fatalf("expected source radius 0 limit, got %#v", report.Limits)
	}
	if len(report.SourceContext.Lines) != 1 {
		t.Fatalf("expected target-only source context, got %#v", report.SourceContext.Lines)
	}
	if len(report.References) != 1 {
		t.Fatalf("expected one reference, got %#v", report.References)
	}
	if len(report.Callers) != 1 {
		t.Fatalf("expected one caller, got %#v", report.Callers)
	}
	if report.Truncated == nil || report.Truncated.References == 0 || report.Truncated.Callers == 0 {
		t.Fatalf("expected reference and caller truncation, got %#v", report.Truncated)
	}
}

func TestAnalyzeFileBuildsAgentContext(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if report.Target != "service.go" || report.File != "service.go" {
		t.Fatalf("target/file = %q/%q, want service.go/service.go", report.Target, report.File)
	}
	if report.Package != "." {
		t.Fatalf("package = %q, want .", report.Package)
	}
	if report.PackageName != "app" {
		t.Fatalf("package name = %q, want app", report.PackageName)
	}
	if len(report.Symbols) != 3 {
		t.Fatalf("expected 3 file symbols, got %#v", report.Symbols)
	}
	if report.Symbols[0].Name != "Entry" || report.Symbols[1].Name != "Target" || report.Symbols[2].Name != "Helper" {
		t.Fatalf("unexpected symbol order: %#v", report.Symbols)
	}
	if len(report.SourceContexts) != 3 {
		t.Fatalf("expected 3 source contexts, got %#v", report.SourceContexts)
	}
	if report.AnalysisMode != AnalysisModeAST {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeAST)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if report.Risk.Level != "medium" {
		t.Fatalf("risk level = %q, want medium", report.Risk.Level)
	}
	if len(report.AffectedPackages) != 1 || report.AffectedPackages[0] != "." {
		t.Fatalf("expected package impact ., got %#v", report.AffectedPackages)
	}
	if len(report.AffectedTests) != 1 || report.AffectedTests[0].Name != "TestTarget" {
		t.Fatalf("expected TestTarget affected test, got %#v", report.AffectedTests)
	}
	if len(report.TestCommands) != 1 || report.TestCommands[0] != "go test ." {
		t.Fatalf("expected go test . command, got %#v", report.TestCommands)
	}
	if len(report.ReadingOrder) != 5 {
		t.Fatalf("expected 5 reading order steps, got %#v", report.ReadingOrder)
	}
	if len(report.Limitations) != 5 {
		t.Fatalf("expected 5 limitations, got %#v", report.Limitations)
	}
}

func TestAnalyzeFileNotesTestsOptionInLimitations(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if len(report.Limitations) != 6 {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
	if !strings.Contains(report.Limitations[5], "--tests") {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
}

func TestAnalyzeFileAppliesLimits(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{
		Limits: LimitOptions{
			MaxSymbols: 1,
			MaxTests:   1,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if len(report.Symbols) != 1 || report.Symbols[0].Name != "Entry" {
		t.Fatalf("expected first symbol only, got %#v", report.Symbols)
	}
	if len(report.SourceContexts) != 1 {
		t.Fatalf("expected one source context, got %#v", report.SourceContexts)
	}
	if report.Truncated == nil || report.Truncated.Symbols != 2 || report.Truncated.SourceContexts != 2 {
		t.Fatalf("expected symbol/source truncation, got %#v", report.Truncated)
	}
}

func TestAnalyzePackageBuildsAgentContext(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if report.Target != "." || report.Package != "." {
		t.Fatalf("target/package = %q/%q, want . and .", report.Target, report.Package)
	}
	if report.PackageName != "app" {
		t.Fatalf("package name = %q, want app", report.PackageName)
	}
	if len(report.Files) != 2 || report.Files[0] != "service.go" || report.Files[1] != "service_test.go" {
		t.Fatalf("unexpected package files: %#v", report.Files)
	}
	if len(report.Symbols) != 4 {
		t.Fatalf("expected 4 package symbols, got %#v", report.Symbols)
	}
	if report.Symbols[0].Name != "Entry" || report.Symbols[1].Name != "Target" || report.Symbols[2].Name != "Helper" || report.Symbols[3].Name != "TestTarget" {
		t.Fatalf("unexpected symbol order: %#v", report.Symbols)
	}
	if len(report.SourceContexts) != 4 {
		t.Fatalf("expected 4 source contexts, got %#v", report.SourceContexts)
	}
	if report.AnalysisMode != AnalysisModeAST {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeAST)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if report.Risk.Level != "medium" {
		t.Fatalf("risk level = %q, want medium", report.Risk.Level)
	}
	if len(report.AffectedPackages) != 1 || report.AffectedPackages[0] != "." {
		t.Fatalf("expected package impact ., got %#v", report.AffectedPackages)
	}
	if len(report.AffectedTests) != 1 || report.AffectedTests[0].Name != "TestTarget" {
		t.Fatalf("expected TestTarget affected test, got %#v", report.AffectedTests)
	}
	if len(report.TestCommands) != 1 || report.TestCommands[0] != "go test ." {
		t.Fatalf("expected go test . command, got %#v", report.TestCommands)
	}
	if len(report.ReadingOrder) != 7 {
		t.Fatalf("expected 7 reading order steps, got %#v", report.ReadingOrder)
	}
	if len(report.Limitations) != 5 {
		t.Fatalf("expected 5 limitations, got %#v", report.Limitations)
	}
}

func TestAnalyzePackageNotesTestsOptionInLimitations(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if len(report.Limitations) != 6 {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
	if !strings.Contains(report.Limitations[5], "--tests") {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
}

func TestAnalyzePackageAppliesLimits(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{
		Limits: LimitOptions{
			MaxFiles:   1,
			MaxSymbols: 1,
			MaxTests:   1,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if len(report.Files) != 1 || report.Files[0] != "service.go" {
		t.Fatalf("expected service.go only, got %#v", report.Files)
	}
	if len(report.Symbols) != 1 || report.Symbols[0].Name != "Entry" {
		t.Fatalf("expected first symbol only, got %#v", report.Symbols)
	}
	if report.Truncated == nil || report.Truncated.Files != 1 || report.Truncated.Symbols != 3 {
		t.Fatalf("expected file/symbol truncation, got %#v", report.Truncated)
	}
}

func TestAnalyzeDiffBuildsAgentContext(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestSession(t *testing.T) {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "api", "handler.go"), `package api

import "example.com/app/internal/auth"

var _ = auth.Session{}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "api", "handler_test.go"), `package api

import "testing"

func TestHandler(t *testing.T) {}
`)
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")

	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	return Session{}
}
`)

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if report.Target != "HEAD" || report.Base != "HEAD" {
		t.Fatalf("target/base = %q/%q, want HEAD/HEAD", report.Target, report.Base)
	}
	if report.Purpose == "" {
		t.Fatal("expected purpose")
	}
	if report.AnalysisMode != AnalysisModeDiff {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeDiff)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if report.Risk.Level != "medium" {
		t.Fatalf("risk level = %q, want medium", report.Risk.Level)
	}
	if len(report.ChangedFiles) != 1 || report.ChangedFiles[0] != "internal/auth/session.go" {
		t.Fatalf("expected changed session.go, got %#v", report.ChangedFiles)
	}
	if len(report.AffectedSymbols) != 1 || report.AffectedSymbols[0] != "NewSession" {
		t.Fatalf("expected NewSession affected symbol, got %#v", report.AffectedSymbols)
	}
	if len(report.AffectedTests) != 2 {
		t.Fatalf("expected 2 affected tests, got %#v", report.AffectedTests)
	}
	if len(report.TestCommands) != 2 {
		t.Fatalf("expected 2 test commands, got %#v", report.TestCommands)
	}
	if len(report.ReadingOrder) != 3 {
		t.Fatalf("expected 3 reading order steps, got %#v", report.ReadingOrder)
	}
	if len(report.Limitations) != 4 {
		t.Fatalf("expected 4 limitations, got %#v", report.Limitations)
	}
}

func TestAnalyzeDiffNotesTestsOptionInLimitations(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n\nfunc Added() {}\n")

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if len(report.Limitations) != 5 {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
	if !strings.Contains(report.Limitations[4], "--tests") {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
}

func TestAnalyzeDiffAppliesLimits(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n\nfunc AddedFirst() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n\nfunc AddedSecond() {}\n")

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{
		Limits: LimitOptions{
			MaxFiles:   1,
			MaxSymbols: 1,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if len(report.ChangedFiles) != 1 {
		t.Fatalf("expected one changed file, got %#v", report.ChangedFiles)
	}
	if len(report.AffectedSymbols) != 1 {
		t.Fatalf("expected one affected symbol, got %#v", report.AffectedSymbols)
	}
	if report.Truncated == nil || report.Truncated.ChangedFiles != 1 || report.Truncated.AffectedSymbols != 1 {
		t.Fatalf("expected diff truncation, got %#v", report.Truncated)
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

func initAgentContextGitRepository(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	runAgentContextGit(t, root, "init")
	runAgentContextGit(t, root, "config", "user.email", "test@example.com")
	runAgentContextGit(t, root, "config", "user.name", "Test User")

	return root
}

func runAgentContextGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}

	return string(output)
}
