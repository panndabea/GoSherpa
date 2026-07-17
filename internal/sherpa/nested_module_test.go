package sherpa

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNestedModulesStayOutOfRootAgentFacingAnalysis(t *testing.T) {
	root := writeNestedModuleAnalysisProject(t)

	symbols, err := ParseRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	symbol, err := FindSymbolTarget(root, symbols, "Target")
	if err != nil {
		t.Fatal(err)
	}
	if symbol.Position.File != "service.go" {
		t.Fatalf("expected root Target symbol, got %#v", symbol)
	}

	references, err := FindReferenceReport(root, "Target")
	if err != nil {
		t.Fatal(err)
	}
	assertNestedModuleNoPositions(t, referenceTestFiles(references.References), "nested/")

	callers, err := FindCallers(root, "Target")
	if err != nil {
		t.Fatal(err)
	}
	callerNames := callTestCallerNames(callers.Callers)
	assertContainsString(t, callerNames, "Entry")
	assertNestedModuleNoCallerFiles(t, callers.Callers, "nested/")

	tests, err := FindTests(root, "Target")
	if err != nil {
		t.Fatal(err)
	}
	testNames := relatedTestNames(tests.Tests)
	assertContainsString(t, testNames, "TestTarget")
	assertNestedModuleNoStrings(t, testNames, "TestNestedTarget")

	impact, err := FindImpact(root, "Target")
	if err != nil {
		t.Fatal(err)
	}
	if impact.Target != "Target" {
		t.Fatalf("expected impact target Target, got %s", impact.Target)
	}
	assertNestedModuleNoPositions(t, referenceTestFiles(impact.References), "nested/")
	assertNestedModuleNoCallerFiles(t, impact.Callers, "nested/")
	impactTests := relatedTestNames(impact.RelatedTests)
	assertContainsString(t, impactTests, "TestTarget")
	assertNestedModuleNoStrings(t, impactTests, "TestNestedTarget")
}

func writeNestedModuleAnalysisProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/root\n\ngo 1.24.4\n")
	writeFile(t, filepath.Join(root, "service.go"), `package root

func Entry() {
	Target()
}

func Target() {}
`)
	writeFile(t, filepath.Join(root, "service_test.go"), `package root

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)
	writeFile(t, filepath.Join(root, "nested", "go.mod"), "module example.com/nested\n\ngo 1.24.4\n")
	writeFile(t, filepath.Join(root, "nested", "service.go"), `package nested

func Entry() {
	Target()
}

func Target() {}
`)
	writeFile(t, filepath.Join(root, "nested", "service_test.go"), `package nested

import "testing"

func TestNestedTarget(t *testing.T) {
	Target()
}
`)

	return root
}

func assertNestedModuleNoCallerFiles(t *testing.T, callers []Caller, blockedPrefix string) {
	t.Helper()

	for _, caller := range callers {
		if strings.HasPrefix(caller.Position.File, blockedPrefix) {
			t.Fatalf("expected callers to exclude %s, got %#v", blockedPrefix, callers)
		}
	}
}

func assertNestedModuleNoPositions(t *testing.T, files []string, blockedPrefix string) {
	t.Helper()

	for _, file := range files {
		if strings.HasPrefix(file, blockedPrefix) {
			t.Fatalf("expected positions to exclude %s, got %#v", blockedPrefix, files)
		}
	}
}

func assertNestedModuleNoStrings(t *testing.T, values []string, blocked string) {
	t.Helper()

	for _, value := range values {
		if value == blocked {
			t.Fatalf("expected values to exclude %s, got %#v", blocked, values)
		}
	}
}
