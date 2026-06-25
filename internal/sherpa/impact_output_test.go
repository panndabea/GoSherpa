package sherpa

import (
	"fmt"
	"testing"
)

func TestFormatImpactForSymbol(t *testing.T) {
	result := ImpactResult{
		Target: "ParseFile",
		Kind:   ImpactKindSymbol,
		References: []Reference{
			{
				Name: "ParseFile",
				Position: Position{
					File: "internal/parser/parser.go",
					Line: 3,
				},
			},
			{
				Name: "ParseFile",
				Position: Position{
					File: "cmd/app/main.go",
					Line: 6,
				},
			},
		},
		Callers: []Caller{
			{
				Name: "Run",
				Position: Position{
					File: "cmd/app/main.go",
					Line: 6,
				},
			},
		},
		Packages: []string{"./cmd/app", "./internal/parser"},
		RelatedTests: []RelatedTest{
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
				Position: Position{
					File: "cmd/app/main_test.go",
					Line: 9,
				},
			},
		},
		TestCommands: []string{"go test ./cmd/app", "go test ./internal/parser"},
	}

	got := FormatImpact(result)
	want := fmt.Sprintf(`IMPACT

TARGET
  ParseFile (symbol)

REFERENCES
  internal/parser/parser.go:3
  cmd/app/main.go:6

DIRECT CALLERS
  %-36s cmd/app/main.go:6

AFFECTED PACKAGES
  ./cmd/app
  ./internal/parser

SUGGESTED TESTS
  %-36s internal/parser/parser_test.go:5
  %-36s cmd/app/main_test.go:9 (direct)

SUGGESTED COMMANDS
  go test ./cmd/app
  go test ./internal/parser
`, "Run", "TestParserPackage", "TestUsesParser")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatImpactForPackage(t *testing.T) {
	result := ImpactResult{
		Target: "./internal/auth",
		Kind:   ImpactKindPackage,
		Dependencies: PackageDependencies{
			Package: "./internal/auth",
			UsedBy:  []string{"./cmd/api"},
		},
		Packages: []string{"./cmd/api", "./internal/auth"},
		RelatedTests: []RelatedTest{
			{
				Name:    "TestAuth",
				Package: "./internal/auth",
				Position: Position{
					File: "internal/auth/service_test.go",
					Line: 5,
				},
			},
		},
		TestCommands: []string{"go test ./internal/auth"},
	}

	got := FormatImpact(result)
	want := fmt.Sprintf(`IMPACT

TARGET
  ./internal/auth (package)

DIRECT DEPENDENTS
  ./cmd/api

AFFECTED PACKAGES
  ./cmd/api
  ./internal/auth

SUGGESTED TESTS
  %-36s internal/auth/service_test.go:5

SUGGESTED COMMANDS
  go test ./internal/auth
`, "TestAuth")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatImpactWithEmptySections(t *testing.T) {
	result := ImpactResult{
		Target: "Missing",
		Kind:   ImpactKindSymbol,
	}

	got := FormatImpact(result)
	want := `IMPACT

TARGET
  Missing (symbol)

REFERENCES
  none

DIRECT CALLERS
  none

AFFECTED PACKAGES
  none

SUGGESTED TESTS
  none

SUGGESTED COMMANDS
  none
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatImpactWithWarnings(t *testing.T) {
	result := ImpactResult{
		Target:   "Run",
		Kind:     ImpactKindSymbol,
		Warnings: []string{"ambiguous function target: Run"},
	}

	got := FormatImpact(result)
	want := `IMPACT

TARGET
  Run (symbol)

REFERENCES
  none

DIRECT CALLERS
  none

AFFECTED PACKAGES
  none

SUGGESTED TESTS
  none

SUGGESTED COMMANDS
  none

WARNINGS
  ambiguous function target: Run
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
