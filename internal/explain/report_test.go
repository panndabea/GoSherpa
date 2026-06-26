package explain

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func TestAnalyzeBuildsSymbolProfile(t *testing.T) {
	root := writeExplainProject(t)

	report, err := Analyze(root, "Target")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if report.Target != "Target" {
		t.Fatalf("target = %q, want Target", report.Target)
	}
	if report.Symbol.Name != "Target" {
		t.Fatalf("symbol name = %q, want Target", report.Symbol.Name)
	}
	if report.Symbol.Position.File != "service.go" {
		t.Fatalf("symbol file = %q, want service.go", report.Symbol.Position.File)
	}
	if report.Purpose != "Target handles the main service step." {
		t.Fatalf("purpose = %q, want doc comment", report.Purpose)
	}
	if report.Risk.Level != "medium" {
		t.Fatalf("risk level = %q, want medium", report.Risk.Level)
	}
	if report.ArchitectureRole.Role != "connector" {
		t.Fatalf("architecture role = %q, want connector", report.ArchitectureRole.Role)
	}

	assertNames(t, callerNames(report.Callers), []string{"Entry"})
	assertNames(t, calleeNames(report.Callees), []string{"Helper"})
	assertNames(t, report.AffectedPackages, []string{"."})
	assertNames(t, testNames(report.RelatedTests), []string{"TestTarget"})
	assertNames(t, report.TestCommands, []string{"go test ."})
	assertNames(t, readingStepTitles(report.ReadingOrder), []string{
		"Definition",
		"Callee: Helper",
		"Caller: Entry",
		"Test: TestTarget",
	})
}

func TestAnalyzeWithOptionsIncludesTestCallers(t *testing.T) {
	root := writeExplainProject(t)

	report, err := AnalyzeWithOptions(root, "Target", AnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzeWithOptions returned error: %v", err)
	}

	assertNames(t, callerNames(report.Callers), []string{"Entry", "TestTarget"})
	assertNames(t, callerFiles(report.Callers), []string{"service.go", "service_test.go"})
}

func TestAnalyzeUsesPackageQualifiedCallSignals(t *testing.T) {
	root := writePackageQualifiedExplainProject(t)

	report, err := Analyze(root, "./internal/auth.Target")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if report.Symbol.Position.File != "internal/auth/auth.go" {
		t.Fatalf("symbol file = %q, want internal/auth/auth.go", report.Symbol.Position.File)
	}

	assertNames(t, callerNames(report.Callers), []string{"Run", "Entry"})
	assertNames(t, callerFiles(report.Callers), []string{
		"cmd/app/main.go",
		"internal/auth/auth.go",
	})
	assertNames(t, calleeNames(report.Callees), []string{"Helper"})
	assertNames(t, calleeFiles(report.Callees), []string{"internal/auth/auth.go"})
}

func TestAnalyzeRejectsPackageTargets(t *testing.T) {
	root := writeExplainProject(t)

	_, err := Analyze(root, ".")
	if err == nil {
		t.Fatal("Analyze returned nil error")
	}
}

func writeExplainProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeExplainTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeExplainTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() {
	Target()
}

// Target handles the main service step.
func Target() {
	Helper()
}

func Helper() {}
`)
	writeExplainTestFile(t, filepath.Join(root, "service_test.go"), `package app

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)

	return root
}

func writePackageQualifiedExplainProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeExplainTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeExplainTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

func Entry() {
	Target()
}

func Target() {
	Helper()
}

func Helper() {}
`)
	writeExplainTestFile(t, filepath.Join(root, "internal", "billing", "billing.go"), `package billing

func Entry() {
	Target()
}

func Target() {
	Helper()
}

func Helper() {}
`)
	writeExplainTestFile(t, filepath.Join(root, "cmd", "app", "main.go"), `package main

import (
	authpkg "example.com/app/internal/auth"
	"example.com/app/internal/billing"
)

func Run() {
	authpkg.Target()
	billing.Target()
}
`)

	return root
}

func writeExplainTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func callerNames(callers []sherpa.Caller) []string {
	names := make([]string, 0, len(callers))
	for _, caller := range callers {
		names = append(names, caller.Name)
	}

	return names
}

func callerFiles(callers []sherpa.Caller) []string {
	files := make([]string, 0, len(callers))
	for _, caller := range callers {
		files = append(files, caller.Position.File)
	}

	return files
}

func calleeNames(callees []sherpa.Callee) []string {
	names := make([]string, 0, len(callees))
	for _, callee := range callees {
		names = append(names, callee.Name)
	}

	return names
}

func calleeFiles(callees []sherpa.Callee) []string {
	files := make([]string, 0, len(callees))
	for _, callee := range callees {
		files = append(files, callee.Position.File)
	}

	return files
}

func testNames(tests []sherpa.RelatedTest) []string {
	names := make([]string, 0, len(tests))
	for _, test := range tests {
		names = append(names, test.Name)
	}

	return names
}

func readingStepTitles(steps []ReadingStep) []string {
	titles := make([]string, 0, len(steps))
	for _, step := range steps {
		titles = append(titles, step.Title)
	}

	return titles
}

func assertNames(t *testing.T, got []string, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
