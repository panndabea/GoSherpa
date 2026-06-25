package sherpa

import (
	"fmt"
	"testing"
)

func TestFormatTests(t *testing.T) {
	result := TestsResult{
		Target: "ParseFile",
		Kind:   TestTargetKindSymbol,
		Tests: []RelatedTest{
			{
				Name:    "TestParserPackage",
				Package: "./internal/parser",
				Position: Position{
					File: "internal/parser/parser_test.go",
					Line: 5,
				},
			},
			{
				Name:            "TestUsesParser",
				Package:         "./cmd/app",
				DirectReference: true,
				ExternalPackage: true,
				Position: Position{
					File: "cmd/app/main_test.go",
					Line: 9,
				},
			},
		},
		Commands: []string{"go test ./cmd/app", "go test ./internal/parser"},
	}

	got := FormatTests(result)
	want := fmt.Sprintf(`TESTS

TARGET
  ParseFile (symbol)

RELATED TESTS
  %-36s internal/parser/parser_test.go:5
  %-36s cmd/app/main_test.go:9 (direct, external)

SUGGESTED COMMANDS
  go test ./cmd/app
  go test ./internal/parser
`, "TestParserPackage", "TestUsesParser")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatTestsWithEmptyLists(t *testing.T) {
	result := TestsResult{
		Target: "Missing",
		Kind:   TestTargetKindSymbol,
	}

	got := FormatTests(result)
	want := `TESTS

TARGET
  Missing (symbol)

RELATED TESTS
  none

SUGGESTED COMMANDS
  none
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
