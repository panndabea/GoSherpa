package semantics

import (
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestLoadRepositoryLoadsLocalPackages(t *testing.T) {
	tmp := t.TempDir()

	writeSemanticTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeSemanticTestFile(t, filepath.Join(tmp, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)
	writeSemanticTestFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/auth"

func Run() {
	auth.Target()
}
`)

	repo, err := LoadRepository(tmp, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(repo.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", repo.Warnings)
	}

	got := semanticTestPackagePaths(repo)
	assertSemanticTestContains(t, got, "./cmd/app")
	assertSemanticTestContains(t, got, "./internal/auth")
}

func TestLoadRepositoryReportsPackageWarnings(t *testing.T) {
	tmp := t.TempDir()

	writeSemanticTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeSemanticTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}

func Broken() {
	Missing()
}
`)

	repo, err := LoadRepository(tmp, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(repo.Warnings) == 0 {
		t.Fatal("expected package load warnings")
	}
	if !strings.Contains(strings.Join(repo.Warnings, "\n"), "Missing") {
		t.Fatalf("expected warning to mention Missing, got %v", repo.Warnings)
	}
}

func TestPackageWarningsIgnoresTransientGoBuildCacheMissWithUsableData(t *testing.T) {
	pkg := &packages.Package{
		PkgPath:   "example.com/app",
		Syntax:    []*ast.File{{}},
		Types:     types.NewPackage("example.com/app", "app"),
		TypesInfo: &types.Info{},
		Errors: []packages.Error{
			{
				Msg: "-: loading compiled Go files from cache: reading srcfiles list: cache entry not found: open /Users/example/Library/Caches/go-build/07/cache-a: no such file or directory",
			},
		},
	}

	warnings := packageWarnings(t.TempDir(), pkg)
	if len(warnings) != 0 {
		t.Fatalf("expected transient cache miss to be ignored, got %v", warnings)
	}
}

func TestPackageWarningsKeepsCacheMissWithoutUsableData(t *testing.T) {
	pkg := &packages.Package{
		PkgPath: "example.com/app",
		Errors: []packages.Error{
			{
				Msg: "-: loading compiled Go files from cache: reading srcfiles list: cache entry not found: open /Users/example/Library/Caches/go-build/07/cache-a: no such file or directory",
			},
		},
	}

	warnings := packageWarnings(t.TempDir(), pkg)
	if len(warnings) == 0 {
		t.Fatal("expected cache miss to remain visible when semantic data is unavailable")
	}
	if !strings.Contains(warnings[0], "cache entry not found") {
		t.Fatalf("expected cache warning, got %v", warnings)
	}
}

func semanticTestPackagePaths(repo Repository) []string {
	var paths []string
	for _, pkg := range repo.Packages {
		paths = append(paths, pkg.PackagePath)
	}

	return paths
}

func assertSemanticTestContains(t *testing.T, values []string, want string) {
	t.Helper()

	for _, value := range values {
		if value == want {
			return
		}
	}

	t.Fatalf("expected %q in %v", want, values)
}

func writeSemanticTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
