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

func TestFormatRepositoryDependencies(t *testing.T) {
	report := RepositoryDependencies{
		Packages: []PackageDependencySummary{
			{
				Package:         ".",
				LocalImports:    []string{"./internal/store"},
				ExternalImports: []string{"fmt"},
				UsedBy:          []string{"./cmd/app"},
			},
			{
				Package: "./internal/store",
				UsedBy:  []string{".", "./cmd/app"},
			},
		},
	}

	got := FormatRepositoryDependencies(report)
	want := `DEPENDENCIES

.
  LOCAL IMPORTS
    ./internal/store
  EXTERNAL IMPORTS
    fmt
  USED BY
    ./cmd/app

./internal/store
  LOCAL IMPORTS
    none
  EXTERNAL IMPORTS
    none
  USED BY
    .
    ./cmd/app

Found 2 packages
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatRepositoryDependenciesWithEmptyReport(t *testing.T) {
	got := FormatRepositoryDependencies(RepositoryDependencies{})
	want := `DEPENDENCIES

  none
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
