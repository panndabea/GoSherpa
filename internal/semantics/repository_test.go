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

func TestLoadRepositoryLoadsGoWorkPackages(t *testing.T) {
	tmp := t.TempDir()

	writeSemanticTestFile(t, filepath.Join(tmp, "go.work"), `go 1.24

use (
	./app
	./service
)
`)
	writeSemanticTestFile(t, filepath.Join(tmp, "service", "go.mod"), "module example.com/service\n")
	writeSemanticTestFile(t, filepath.Join(tmp, "service", "service.go"), `package service

type Client struct{}

func (c *Client) Start() {}
`)
	writeSemanticTestFile(t, filepath.Join(tmp, "app", "go.mod"), `module example.com/app

require example.com/service v0.0.0
`)
	writeSemanticTestFile(t, filepath.Join(tmp, "app", "main.go"), `package app

import "example.com/service"

func Run(client *service.Client) {
	client.Start()
}
`)

	repo, err := LoadRepository(tmp, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	got := semanticTestPackagePaths(repo)
	assertSemanticTestContains(t, got, "./app")
	assertSemanticTestContains(t, got, "./service")
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

func TestLoadRepositoryHonorsBuildTags(t *testing.T) {
	tmp := t.TempDir()

	writeSemanticTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeSemanticTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Always() {}
`)
	writeSemanticTestFile(t, filepath.Join(tmp, "enterprise.go"), `//go:build enterprise

package service

func Enterprise() {}
`)

	withoutTags, err := LoadRepository(tmp, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if semanticTestCompiledFileExists(withoutTags, "enterprise.go") {
		t.Fatalf("expected enterprise.go to be excluded without tag, got %#v", withoutTags.Packages)
	}

	withTags, err := LoadRepository(tmp, LoadOptions{BuildTags: []string{"enterprise"}})
	if err != nil {
		t.Fatal(err)
	}
	if !semanticTestCompiledFileExists(withTags, "enterprise.go") {
		t.Fatalf("expected enterprise.go to be included with tag, got %#v", withTags.Packages)
	}
}

func TestLoadRepositoryRetriesEmptyLoadWithWritableCache(t *testing.T) {
	tmp := t.TempDir()

	goFile := filepath.Join(tmp, "service.go")
	writeSemanticTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeSemanticTestFile(t, goFile, "package app\n")

	original := packageLoader
	t.Cleanup(func() {
		packageLoader = original
	})

	calls := 0
	packageLoader = func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		calls++
		switch calls {
		case 1:
			if semanticTestEnvValue(cfg.Env, "GOCACHE") != "" {
				t.Fatalf("first package load unexpectedly set GOCACHE: %v", cfg.Env)
			}
			return nil, nil
		case 2:
			goCache := semanticTestEnvValue(cfg.Env, "GOCACHE")
			if !strings.Contains(filepath.ToSlash(goCache), "gosherpa-go-build-cache") {
				t.Fatalf("expected retry with gosherpa GOCACHE, got %q", goCache)
			}
			return []*packages.Package{
				{
					ID:              "example.com/app",
					Name:            "app",
					PkgPath:         "example.com/app",
					Dir:             tmp,
					GoFiles:         []string{goFile},
					CompiledGoFiles: []string{goFile},
				},
			}, nil
		default:
			t.Fatalf("unexpected package load call %d", calls)
			return nil, nil
		}
	}

	repo, err := LoadRepository(tmp, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("package load calls = %d, want 2", calls)
	}

	got := semanticTestPackagePaths(repo)
	assertSemanticTestContains(t, got, ".")
}

func TestNormalizeBuildTagsSplitsDeduplicatesAndSorts(t *testing.T) {
	got := NormalizeBuildTags([]string{"integration, enterprise", "enterprise", "debug"})
	want := []string{"debug", "enterprise", "integration"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("NormalizeBuildTags() = %#v, want %#v", got, want)
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

func TestPackageLoadHasCacheAccessErrorRecognizesSandboxPermissionErrors(t *testing.T) {
	pkgs := []*packages.Package{
		{
			PkgPath: "example.com/app",
			Errors: []packages.Error{
				{
					Msg: "-: open /Users/example/Library/Caches/go-build/cf/cache-d: operation not permitted",
				},
			},
		},
	}

	if !packageLoadHasCacheAccessError(pkgs) {
		t.Fatal("expected go-build cache permission error to be recognized")
	}

	ordinaryTypeError := []*packages.Package{
		{
			PkgPath: "example.com/app",
			Errors: []packages.Error{
				{Msg: "service.go:4:2: undefined: Missing"},
			},
		},
	}
	if packageLoadHasCacheAccessError(ordinaryTypeError) {
		t.Fatal("expected ordinary type error not to be treated as cache access")
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

func semanticTestCompiledFileExists(repo Repository, name string) bool {
	for _, pkg := range repo.Packages {
		for _, file := range pkg.CompiledGoFiles {
			if filepath.Base(file) == name {
				return true
			}
		}
	}

	return false
}

func semanticTestEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}

	return ""
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
