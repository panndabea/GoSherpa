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
`, "Run")

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
	}

	got := FormatImpact(result)
	want := `IMPACT

TARGET
  ./internal/auth (package)

DIRECT DEPENDENTS
  ./cmd/api

AFFECTED PACKAGES
  ./cmd/api
  ./internal/auth
`

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

WARNINGS
  ambiguous function target: Run
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
