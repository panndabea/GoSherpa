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

func calleeNames(callees []sherpa.Callee) []string {
	names := make([]string, 0, len(callees))
	for _, callee := range callees {
		names = append(names, callee.Name)
	}

	return names
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
