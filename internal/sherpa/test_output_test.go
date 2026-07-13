package sherpa

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatTests(t *testing.T) {
	result := TestsResult{
		Target:       "ParseFile",
		Kind:         TestTargetKindSymbol,
		AnalysisMode: TestAnalysisModeTypecheckedAST,
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
		TestPlan: TestPlan{
			Direct: []TestPlanItem{
				{
					Command: "go test ./cmd/app",
					Reason:  "Direct tests in ./cmd/app reference ParseFile: TestUsesParser.",
					Package: "./cmd/app",
					Test:    "TestUsesParser",
				},
			},
			Related: []TestPlanItem{
				{
					Command: "go test ./internal/parser",
					Reason:  "Same-package tests in ./internal/parser are related to ParseFile: TestParserPackage.",
					Package: "./internal/parser",
					Test:    "TestParserPackage",
				},
			},
		},
	}

	got := FormatTests(result)
	want := fmt.Sprintf(`TESTS

TARGET
  ParseFile (symbol)

ANALYSIS
  Mode: typechecked+ast

RELATED TESTS
  %-36s internal/parser/parser_test.go:5
  %-36s cmd/app/main_test.go:9 (direct, external)

TEST PLAN
  DIRECT
    go test ./cmd/app
      reason: Direct tests in ./cmd/app reference ParseFile: TestUsesParser.
  RELATED
    go test ./internal/parser
      reason: Same-package tests in ./internal/parser are related to ParseFile: TestParserPackage.
  CONTRACTS
    none
  CALLER PACKAGES
    none
  FALLBACK
    none
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

ANALYSIS
  Mode: ast

RELATED TESTS
  none

TEST PLAN
  DIRECT
    none
  RELATED
    none
  CONTRACTS
    none
  CALLER PACKAGES
    none
  FALLBACK
    none
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatTestsWithWarnings(t *testing.T) {
	result := TestsResult{
		Target:       "Target",
		Kind:         TestTargetKindSymbol,
		AnalysisMode: TestAnalysisModeAST,
		Warnings:     []string{"typechecked test reference analysis unavailable: loader failed"},
	}

	got := FormatTests(result)
	if !strings.Contains(got, "WARNINGS\n  - typechecked test reference analysis unavailable: loader failed\n") {
		t.Fatalf("expected warning output, got:\n%s", got)
	}
}
