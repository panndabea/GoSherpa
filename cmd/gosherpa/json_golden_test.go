package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMainJSONGoldenFiles(t *testing.T) {
	fixtureRoot := filepath.Join("testdata", "json_project")
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "analyze",
			args: []string{"analyze", "--json"},
		},
		{
			name: "symbol",
			args: []string{"symbol", "Entry", "--json"},
		},
		{
			name: "symbols",
			args: []string{"symbols", "--json"},
		},
		{
			name: "search",
			args: []string{"search", "Target", "--json"},
		},
		{
			name: "search-filtered",
			args: []string{"search", "Target", "--kind", "function", "--package", ".", "--tests", "--limit", "1", "--json"},
		},
		{
			name: "refs",
			args: []string{"refs", "Target", "--json"},
		},
		{
			name: "impact",
			args: []string{"impact", "Target", "--json"},
		},
		{
			name: "impact-file",
			args: []string{"impact", "file", "service.go", "--json"},
		},
		{
			name: "impact-package",
			args: []string{"impact", "package", ".", "--json"},
		},
		{
			name: "impact-symbol",
			args: []string{"impact", "symbol", "Target", "--json"},
		},
		{
			name: "explain",
			args: []string{"explain", "Target", "--json"},
		},
		{
			name: "context-symbol",
			args: []string{"context", "symbol", "Target", "--json"},
		},
		{
			name: "context-file",
			args: []string{"context", "file", "service.go", "--json"},
		},
		{
			name: "context-package",
			args: []string{"context", "package", ".", "--json"},
		},
		{
			name: "tests",
			args: []string{"tests", "Target", "--json"},
		},
		{
			name: "deps",
			args: []string{"deps", ".", "--json"},
		},
		{
			name: "deps-all",
			args: []string{"deps", "--all", "--json"},
		},
		{
			name: "packages",
			args: []string{"packages", "--json"},
		},
		{
			name: "callers",
			args: []string{"callers", "Target", "--json"},
		},
		{
			name: "callees",
			args: []string{"callees", "Entry", "--json"},
		},
		{
			name: "path",
			args: []string{"path", "Entry", "Target", "--json"},
		},
		{
			name: "paths",
			args: []string{"paths", "Entry", "Target", "--limit", "2", "--json"},
		},
		{
			name: "entrypoints",
			args: []string{"entrypoints", "Target", "--json"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"gosherpa", "--root", fixtureRoot}, test.args...)
			result := runMainTest(t, args)

			if result.ExitCode != exitSuccess {
				t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
			}

			if result.Stderr != "" {
				t.Fatalf("expected empty stderr, got %q", result.Stderr)
			}

			actual := canonicalGoldenActualJSON(t, result.Stdout, fixtureRoot)
			expectedPath := filepath.Join("testdata", "golden-json", test.name+".json")
			expectedBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
			expected := canonicalGoldenJSON(t, expectedBytes)

			if actual != expected {
				t.Fatalf("golden JSON mismatch for %s\nexpected:\n%s\nactual:\n%s", test.name, expected, actual)
			}
		})
	}
}

func TestMainInterfaceJSONGoldenFiles(t *testing.T) {
	fixtureRoot := filepath.Join("testdata", "interface_project")
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "implementers",
			args: []string{"implementers", "./internal/auth.Authenticator", "--json"},
		},
		{
			name: "interfaces",
			args: []string{"interfaces", "./internal/jwt.JWTAuthenticator", "--json"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"gosherpa", "--root", fixtureRoot}, test.args...)
			result := runMainTest(t, args)

			if result.ExitCode != exitSuccess {
				t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
			}

			if result.Stderr != "" {
				t.Fatalf("expected empty stderr, got %q", result.Stderr)
			}

			actual := canonicalGoldenActualJSON(t, result.Stdout, fixtureRoot)
			expectedPath := filepath.Join("testdata", "golden-json", test.name+".json")
			expectedBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
			expected := canonicalGoldenJSON(t, expectedBytes)

			if actual != expected {
				t.Fatalf("golden JSON mismatch for %s\nexpected:\n%s\nactual:\n%s", test.name, expected, actual)
			}
		})
	}
}

func TestMainImpactDiffJSONGoldenFile(t *testing.T) {
	sourceFixtureRoot := filepath.Join("testdata", "json_project")
	fixtureRoot := t.TempDir()
	copyMainTestTree(t, sourceFixtureRoot, fixtureRoot)
	initMainTestGitRepository(t, fixtureRoot)
	runMainTestGit(t, fixtureRoot, "add", ".")
	runMainTestGit(t, fixtureRoot, "commit", "-m", "initial")

	servicePath := filepath.Join(fixtureRoot, "service.go")
	serviceSource, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	writeMainTestFile(t, servicePath, string(serviceSource)+"\nfunc Added() {}\n")

	result := runMainTest(t, []string{"gosherpa", "--root", fixtureRoot, "impact", "diff", "--base", "HEAD", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	actual := canonicalGoldenActualJSON(t, result.Stdout, fixtureRoot)
	expectedPath := filepath.Join("testdata", "golden-json", "impact-diff.json")
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	expected := canonicalGoldenJSON(t, expectedBytes)

	if actual != expected {
		t.Fatalf("golden JSON mismatch for impact-diff\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestMainContextDiffJSONGoldenFile(t *testing.T) {
	sourceFixtureRoot := filepath.Join("testdata", "json_project")
	fixtureRoot := t.TempDir()
	copyMainTestTree(t, sourceFixtureRoot, fixtureRoot)
	initMainTestGitRepository(t, fixtureRoot)
	runMainTestGit(t, fixtureRoot, "add", ".")
	runMainTestGit(t, fixtureRoot, "commit", "-m", "initial")

	servicePath := filepath.Join(fixtureRoot, "service.go")
	serviceSource, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	writeMainTestFile(t, servicePath, string(serviceSource)+"\nfunc Added() {}\n")

	result := runMainTest(t, []string{"gosherpa", "--root", fixtureRoot, "context", "diff", "--base", "HEAD", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	actual := canonicalGoldenActualJSON(t, result.Stdout, fixtureRoot)
	expectedPath := filepath.Join("testdata", "golden-json", "context-diff.json")
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	expected := canonicalGoldenJSON(t, expectedBytes)

	if actual != expected {
		t.Fatalf("golden JSON mismatch for context-diff\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestMainTestsAffectedJSONGoldenFile(t *testing.T) {
	sourceFixtureRoot := filepath.Join("testdata", "json_project")
	fixtureRoot := t.TempDir()
	copyMainTestTree(t, sourceFixtureRoot, fixtureRoot)
	initMainTestGitRepository(t, fixtureRoot)
	runMainTestGit(t, fixtureRoot, "add", ".")
	runMainTestGit(t, fixtureRoot, "commit", "-m", "initial")

	servicePath := filepath.Join(fixtureRoot, "service.go")
	serviceSource, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	writeMainTestFile(t, servicePath, string(serviceSource)+"\nfunc Added() {}\n")

	result := runMainTest(t, []string{"gosherpa", "--root", fixtureRoot, "tests", "affected", "--base", "HEAD", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	actual := canonicalGoldenActualJSON(t, result.Stdout, fixtureRoot)
	expectedPath := filepath.Join("testdata", "golden-json", "tests-affected.json")
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	expected := canonicalGoldenJSON(t, expectedBytes)

	if actual != expected {
		t.Fatalf("golden JSON mismatch for tests-affected\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func TestMainPRJSONGoldenFile(t *testing.T) {
	sourceFixtureRoot := filepath.Join("testdata", "json_project")
	fixtureRoot := t.TempDir()
	copyMainTestTree(t, sourceFixtureRoot, fixtureRoot)
	initMainTestGitRepository(t, fixtureRoot)
	runMainTestGit(t, fixtureRoot, "add", ".")
	runMainTestGit(t, fixtureRoot, "commit", "-m", "initial")

	servicePath := filepath.Join(fixtureRoot, "service.go")
	serviceSource, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	writeMainTestFile(t, servicePath, string(serviceSource)+"\nfunc Added() {}\n")

	result := runMainTest(t, []string{"gosherpa", "--root", fixtureRoot, "pr", "--base", "HEAD", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	actual := canonicalGoldenActualJSON(t, result.Stdout, fixtureRoot)
	expectedPath := filepath.Join("testdata", "golden-json", "pr.json")
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatal(err)
	}
	expected := canonicalGoldenJSON(t, expectedBytes)

	if actual != expected {
		t.Fatalf("golden JSON mismatch for pr\nexpected:\n%s\nactual:\n%s", expected, actual)
	}
}

func canonicalGoldenActualJSON(t *testing.T, output string, fixtureRoot string) string {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected valid JSON, got %v:\n%s", err, output)
	}

	absoluteRoot, err := filepath.Abs(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	absoluteRoot = filepath.Clean(absoluteRoot)

	if payload["root"] != absoluteRoot {
		t.Fatalf("expected root %s, got %v", absoluteRoot, payload["root"])
	}

	payload["root"] = "<ROOT>"

	return canonicalGoldenJSONValue(t, payload)
}

func canonicalGoldenJSON(t *testing.T, data []byte) string {
	t.Helper()

	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("expected valid golden JSON, got %v:\n%s", err, string(data))
	}

	return canonicalGoldenJSONValue(t, payload)
}

func canonicalGoldenJSONValue(t *testing.T, payload any) string {
	t.Helper()

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(payload); err != nil {
		t.Fatal(err)
	}

	return output.String()
}
