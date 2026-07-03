package sherpa

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFindPackageSummariesCountsProductionPackageMap(t *testing.T) {
	tmp := writePackageInventoryProject(t)

	packages, err := FindPackageSummaries(tmp, PackageInventoryOptions{})
	if err != nil {
		t.Fatal(err)
	}

	rootPackage := findPackageSummary(t, packages, ".")
	if rootPackage.PackageName != "app" {
		t.Fatalf("expected package name app, got %s", rootPackage.PackageName)
	}
	if rootPackage.GoFiles != 1 {
		t.Fatalf("expected 1 go file, got %d", rootPackage.GoFiles)
	}
	if rootPackage.TestFiles != 1 {
		t.Fatalf("expected 1 test file, got %d", rootPackage.TestFiles)
	}
	if rootPackage.Symbols != 2 {
		t.Fatalf("expected 2 production symbols, got %d", rootPackage.Symbols)
	}
	if rootPackage.Imports != 2 || rootPackage.LocalImports != 1 || rootPackage.ExternalImports != 1 {
		t.Fatalf("expected 2 imports with 1 local and 1 external, got %#v", rootPackage)
	}
	if rootPackage.ImportedBy != 1 {
		t.Fatalf("expected root package to be imported by 1 package, got %d", rootPackage.ImportedBy)
	}
	if !rootPackage.HasTests {
		t.Fatal("expected root package to be marked with tests")
	}

	storePackage := findPackageSummary(t, packages, "./internal/store")
	if storePackage.ImportedBy != 2 {
		t.Fatalf("expected store package to be imported by 2 packages, got %d", storePackage.ImportedBy)
	}
}

func TestFindPackageSummariesIncludesTestSignalsWhenRequested(t *testing.T) {
	tmp := writePackageInventoryProject(t)

	packages, err := FindPackageSummaries(tmp, PackageInventoryOptions{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}

	rootPackage := findPackageSummary(t, packages, ".")
	if rootPackage.Symbols != 3 {
		t.Fatalf("expected test symbol to be included, got %d symbols", rootPackage.Symbols)
	}
	if rootPackage.Imports != 3 || rootPackage.LocalImports != 1 || rootPackage.ExternalImports != 2 {
		t.Fatalf("expected test imports to be included, got %#v", rootPackage)
	}
}

func TestFormatPackageSummaries(t *testing.T) {
	output := FormatPackageSummaries([]PackageSummary{
		{
			Package:         ".",
			PackageName:     "app",
			GoFiles:         1,
			TestFiles:       1,
			Symbols:         2,
			Imports:         2,
			LocalImports:    1,
			ExternalImports: 1,
			ImportedBy:      1,
			HasTests:        true,
		},
	})

	for _, want := range []string{"PACKAGES", "PACKAGE", "app", "SYMBOLS", "yes"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func writePackageInventoryProject(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package app

import (
	"fmt"

	"example.com/app/internal/store"
)

type Service struct{}

func Run() {
	fmt.Println(store.Name())
}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package app

import "testing"

func TestRun(t *testing.T) {
	Run()
}
`)
	writeFile(t, filepath.Join(tmp, "internal", "store", "store.go"), `package store

func Name() string {
	return "store"
}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import (
	"example.com/app"
	"example.com/app/internal/store"
)

func main() {
	app.Run()
	_ = store.Name()
}
`)

	return tmp
}

func findPackageSummary(t *testing.T, packages []PackageSummary, packagePath string) PackageSummary {
	t.Helper()

	for _, pkg := range packages {
		if pkg.Package == packagePath {
			return pkg
		}
	}

	t.Fatalf("expected package %s in %#v", packagePath, packages)
	return PackageSummary{}
}
