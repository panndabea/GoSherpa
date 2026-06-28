package semantics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
