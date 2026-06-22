package sherpa

import "testing"

func TestFormatPackageDependencies(t *testing.T) {
	deps := PackageDependencies{
		Package: "./internal/sherpa",
		Imports: []string{"fmt", "go/ast"},
		UsedBy:  []string{"./cmd/gosherpa"},
	}

	got := FormatPackageDependencies(deps)
	want := `PACKAGE
  ./internal/sherpa

IMPORTS
  fmt
  go/ast

USED BY
  ./cmd/gosherpa
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatPackageDependenciesWithEmptyLists(t *testing.T) {
	deps := PackageDependencies{
		Package: "./internal/empty",
	}

	got := FormatPackageDependencies(deps)
	want := `PACKAGE
  ./internal/empty

IMPORTS
  none

USED BY
  none
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
