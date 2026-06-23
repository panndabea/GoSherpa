package sherpa

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUniqueSorted(t *testing.T) {
	got := uniqueSorted([]string{"strings", "", "fmt", "fmt"})
	want := []string{"fmt", "strings"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestModulePath(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")

	got, err := modulePath(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if got != "example.com/app" {
		t.Fatalf("expected example.com/app, got %s", got)
	}
}

func TestModulePathReturnsErrorWhenGoModIsMissing(t *testing.T) {
	tmp := t.TempDir()

	_, err := modulePath(tmp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestModulePathReturnsErrorWhenModuleDirectiveIsMissing(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "go 1.24.4\n")

	_, err := modulePath(tmp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeTargetPackage(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{name: "root", input: ".", output: "."},
		{name: "local with dot slash", input: "./internal/auth", output: "./internal/auth"},
		{name: "local without dot slash", input: "internal/auth", output: "./internal/auth"},
		{name: "trailing slash", input: "internal/auth/", output: "./internal/auth"},
		{name: "module root", input: "example.com/app", output: "."},
		{name: "module package", input: "example.com/app/internal/auth", output: "./internal/auth"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeTargetPackage(test.input, "example.com/app")
			if err != nil {
				t.Fatal(err)
			}

			if got != test.output {
				t.Fatalf("expected %s, got %s", test.output, got)
			}
		})
	}
}

func TestNormalizeTargetPackageRejectsInvalidInput(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"/tmp/project",
		"../auth",
		"internal/../auth",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := normalizeTargetPackage(test, "example.com/app")
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPackagePathForFile(t *testing.T) {
	tmp := t.TempDir()

	rootFile := filepath.Join(tmp, "main.go")
	packageFile := filepath.Join(tmp, "internal", "auth", "service.go")

	writeFile(t, rootFile, "package main\n")
	writeFile(t, packageFile, "package auth\n")

	rootPackage, err := packagePathForFile(tmp, rootFile)
	if err != nil {
		t.Fatal(err)
	}

	if rootPackage != "." {
		t.Fatalf("expected ., got %s", rootPackage)
	}

	authPackage, err := packagePathForFile(tmp, packageFile)
	if err != nil {
		t.Fatal(err)
	}

	if authPackage != "./internal/auth" {
		t.Fatalf("expected ./internal/auth, got %s", authPackage)
	}
}

func TestParseImports(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")

	writeFile(t, path, `package auth

import (
	. "errors"
	"fmt"
	_ "net/http"
	alias "strings"
)
`)

	got, err := parseImports(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"errors", "fmt", "net/http", "strings"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCollectPackageImports(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), `package auth

import "fmt"
`)
	writeFile(t, filepath.Join(tmp, "internal", "auth", "handler.go"), `package auth

import "strings"
`)

	importsByPackage, err := collectPackageImports(tmp)
	if err != nil {
		t.Fatal(err)
	}

	imports, ok := importsByPackage["./internal/auth"]
	if !ok {
		t.Fatal("expected ./internal/auth package")
	}

	assertContainsString(t, imports, "fmt")
	assertContainsString(t, imports, "strings")
}

func TestCollectPackageImportsIncludesPackagesWithoutImports(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "internal", "empty", "empty.go"), "package empty\n")

	importsByPackage, err := collectPackageImports(tmp)
	if err != nil {
		t.Fatal(err)
	}

	imports, ok := importsByPackage["./internal/empty"]
	if !ok {
		t.Fatal("expected ./internal/empty package")
	}

	if len(imports) != 0 {
		t.Fatalf("expected no imports, got %v", imports)
	}
}

func TestLocalPackagePath(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		wantPath   string
		wantOK     bool
	}{
		{name: "module root", importPath: "example.com/app", wantPath: ".", wantOK: true},
		{name: "module package", importPath: "example.com/app/internal/auth", wantPath: "./internal/auth", wantOK: true},
		{name: "std package", importPath: "fmt", wantPath: "", wantOK: false},
		{name: "std package with slash", importPath: "go/ast", wantPath: "", wantOK: false},
		{name: "prefix collision", importPath: "example.com/app2/internal/auth", wantPath: "", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotPath, gotOK := localPackagePath(test.importPath, "example.com/app")
			if gotPath != test.wantPath || gotOK != test.wantOK {
				t.Fatalf("expected %q, %v, got %q, %v", test.wantPath, test.wantOK, gotPath, gotOK)
			}
		})
	}
}

func TestFindPackageDependenciesFindsImports(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), `package auth

import (
	"fmt"
	"strings"
)
`)

	deps, err := FindPackageDependencies(tmp, "./internal/auth")
	if err != nil {
		t.Fatal(err)
	}

	if deps.Package != "./internal/auth" {
		t.Fatalf("expected ./internal/auth, got %s", deps.Package)
	}

	assertContainsString(t, deps.Imports, "fmt")
	assertContainsString(t, deps.Imports, "strings")

	if len(deps.UsedBy) != 0 {
		t.Fatalf("expected no used by packages, got %v", deps.UsedBy)
	}
}

func TestFindPackageDependenciesWorksWithAbsoluteRoot(t *testing.T) {
	tmp := t.TempDir()
	absoluteRoot, err := filepath.Abs(tmp)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), `package auth

import "fmt"
`)

	deps, err := FindPackageDependencies(absoluteRoot, "./internal/auth")
	if err != nil {
		t.Fatal(err)
	}

	if deps.Package != "./internal/auth" {
		t.Fatalf("expected ./internal/auth, got %s", deps.Package)
	}

	assertContainsString(t, deps.Imports, "fmt")
}

func TestFindPackageDependenciesFindsUsedBy(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), "package auth\n")
	writeFile(t, filepath.Join(tmp, "cmd", "api", "main.go"), `package main

import "example.com/app/internal/auth"
`)

	deps, err := FindPackageDependencies(tmp, "internal/auth")
	if err != nil {
		t.Fatal(err)
	}

	if deps.Package != "./internal/auth" {
		t.Fatalf("expected ./internal/auth, got %s", deps.Package)
	}

	assertContainsString(t, deps.UsedBy, "./cmd/api")
}

func TestFindPackageDependenciesDisplaysLocalImportsAsLocalPaths(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "store", "store.go"), "package store\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), `package auth

import "example.com/app/internal/store"
`)

	deps, err := FindPackageDependencies(tmp, "./internal/auth")
	if err != nil {
		t.Fatal(err)
	}

	assertContainsString(t, deps.Imports, "./internal/store")

	if containsString(deps.Imports, "example.com/app/internal/store") {
		t.Fatal("expected raw module import to be displayed as local path")
	}
}

func TestFindPackageDependenciesReturnsErrorForMissingPackage(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), "package auth\n")

	_, err := FindPackageDependencies(tmp, "./internal/missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindPackageDependenciesDeduplicatesImportsAndUsedBy(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), `package auth

import "fmt"
`)
	writeFile(t, filepath.Join(tmp, "internal", "auth", "handler.go"), `package auth

import "fmt"
`)
	writeFile(t, filepath.Join(tmp, "cmd", "api", "main.go"), `package main

import "example.com/app/internal/auth"
`)
	writeFile(t, filepath.Join(tmp, "cmd", "api", "worker.go"), `package main

import "example.com/app/internal/auth"
`)

	deps, err := FindPackageDependencies(tmp, "./internal/auth")
	if err != nil {
		t.Fatal(err)
	}

	if countString(deps.Imports, "fmt") != 1 {
		t.Fatalf("expected fmt once, got %v", deps.Imports)
	}

	if countString(deps.UsedBy, "./cmd/api") != 1 {
		t.Fatalf("expected ./cmd/api once, got %v", deps.UsedBy)
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}

func countString(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}

	return count
}

func assertContainsString(t *testing.T, values []string, want string) {
	t.Helper()

	if !containsString(values, want) {
		t.Fatalf("expected %v to contain %s", values, want)
	}
}
