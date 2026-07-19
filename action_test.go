package gosherpa_test

import (
	"os"
	"strings"
	"testing"
)

func TestGitHubActionDefinesPRIntelligenceWorkflow(t *testing.T) {
	contents := readTextFile(t, "action.yml")

	required := []string{
		"name: GoSherpa PR Intelligence",
		"runs:",
		"using: composite",
		"base-ref:",
		"gosherpa --root",
		"init \\",
		"doctor --json",
		"agent context",
		"pr \\",
		"tests affected",
		"init.json",
		"GITHUB_STEP_SUMMARY",
		"actions/upload-artifact",
	}

	for _, text := range required {
		if !strings.Contains(contents, text) {
			t.Fatalf("action.yml missing %q", text)
		}
	}

	forbidden := []string{
		"snapshot --json",
		"pr --base",
		"tests affected --base",
	}

	for _, text := range forbidden {
		if strings.Contains(contents, text) {
			t.Fatalf("action.yml still contains old workflow fragment %q", text)
		}
	}
}

func TestDogfoodWorkflowUsesLocalActionWithFullHistory(t *testing.T) {
	contents := readTextFile(t, ".github/workflows/gosherpa-pr.yml")

	required := []string{
		"name: GoSherpa PR Intelligence",
		"pull_request:",
		"uses: actions/checkout",
		"fetch-depth: 0",
		"uses: actions/setup-go",
		"uses: ./",
		"version: local",
		"base-ref:",
	}

	for _, text := range required {
		if !strings.Contains(contents, text) {
			t.Fatalf("gosherpa-pr workflow missing %q", text)
		}
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(contents)
}
