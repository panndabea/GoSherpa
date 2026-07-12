package symbolindex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHonorsBuildTagsForSymbols(t *testing.T) {
	root := t.TempDir()
	writeSymbolIndexTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeSymbolIndexTestFile(t, filepath.Join(root, "default.go"), `//go:build !enterprise

package app

func Target() {}
`)
	writeSymbolIndexTestFile(t, filepath.Join(root, "enterprise.go"), `//go:build enterprise

package app

func Target() {}
`)

	withoutTags, err := Load(root, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	symbol, found, err := withoutTags.FindSymbol("Target")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected Target without tags")
	}
	if symbol.Position.File != "default.go" {
		t.Fatalf("symbol file without tags = %q, want default.go", symbol.Position.File)
	}

	withTags, err := Load(root, LoadOptions{BuildTags: []string{"enterprise"}})
	if err != nil {
		t.Fatal(err)
	}
	symbol, found, err = withTags.FindSymbol("Target")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected Target with enterprise tag")
	}
	if symbol.Position.File != "enterprise.go" {
		t.Fatalf("symbol file with tags = %q, want enterprise.go", symbol.Position.File)
	}
}

func TestLoadBuildsRepositoryIndexRecords(t *testing.T) {
	root := t.TempDir()
	writeSymbolIndexTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeSymbolIndexTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)
	writeSymbolIndexTestFile(t, filepath.Join(root, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/auth"

func Run() {
	auth.Target()
}
`)

	index, err := LoadRepositoryIndex(root, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if index.Root != root {
		t.Fatalf("root = %q, want %q", index.Root, root)
	}
	if index.ModulePath != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", index.ModulePath)
	}
	if len(index.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %#v", index.Packages)
	}
	if len(index.Files) != 2 {
		t.Fatalf("expected 2 files, got %#v", index.Files)
	}

	pkg, ok := index.FindPackage("./cmd/app")
	if !ok {
		t.Fatalf("expected ./cmd/app package, got %#v", index.Packages)
	}
	if pkg.Name != "main" || pkg.ImportPath != "example.com/app/cmd/app" || pkg.Dir != "cmd/app" {
		t.Fatalf("unexpected package record: %#v", pkg)
	}
	if len(pkg.Files) != 1 || pkg.Files[0] != "cmd/app/main.go" {
		t.Fatalf("unexpected package files: %#v", pkg.Files)
	}

	file, ok := index.FindFile("cmd/app/main.go")
	if !ok {
		t.Fatalf("expected cmd/app/main.go file, got %#v", index.Files)
	}
	if file.Package != "./cmd/app" || file.PackageName != "main" {
		t.Fatalf("unexpected file record: %#v", file)
	}

	symbols := index.SymbolsInPackage("./cmd/app")
	if len(symbols) != 1 || symbols[0].Name != "Run" {
		t.Fatalf("unexpected package symbols: %#v", symbols)
	}
}

func TestIndexAccessorsReturnCopies(t *testing.T) {
	root := t.TempDir()
	writeSymbolIndexTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeSymbolIndexTestFile(t, filepath.Join(root, "service.go"), `package app

func Target() {}
`)

	index, err := Load(root, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	files := index.PackageFiles(".")
	files[0] = "mutated.go"
	if got := index.PackageFiles("."); len(got) != 1 || got[0] != "service.go" {
		t.Fatalf("package files should be immutable copy, got %#v", got)
	}

	pkg, ok := index.FindPackage(".")
	if !ok {
		t.Fatal("expected root package")
	}
	pkg.Files[0] = "mutated.go"
	pkg, ok = index.FindPackage(".")
	if !ok || len(pkg.Files) != 1 || pkg.Files[0] != "service.go" {
		t.Fatalf("package record should be immutable copy, got %#v", pkg)
	}
}

func TestFindSymbolReportsAmbiguousCandidates(t *testing.T) {
	root := t.TempDir()
	writeSymbolIndexTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeSymbolIndexTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)
	writeSymbolIndexTestFile(t, filepath.Join(root, "internal", "billing", "billing.go"), `package billing

func Target() {}
`)

	index, err := Load(root, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = index.FindSymbol("Target")
	if err == nil {
		t.Fatal("expected ambiguous symbol error")
	}

	for _, want := range []string{
		"ambiguous symbol target: Target",
		"package ./internal/auth, file internal/auth/auth.go:3, target ./internal/auth.Target",
		"package ./internal/billing, file internal/billing/billing.go:3, target ./internal/billing.Target",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got:\n%v", want, err)
		}
	}
}

func TestFindSymbolAcceptsModuleQualifiedTarget(t *testing.T) {
	root := t.TempDir()
	writeSymbolIndexTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeSymbolIndexTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)

	index, err := Load(root, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}

	symbol, found, err := index.FindSymbol("example.com/app/internal/auth.Target")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected module-qualified Target")
	}
	if symbol.Package != "./internal/auth" {
		t.Fatalf("symbol package = %q, want ./internal/auth", symbol.Package)
	}
}

func writeSymbolIndexTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
