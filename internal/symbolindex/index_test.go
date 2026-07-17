package symbolindex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/panndabea/GoSherpa/internal/sherpa"
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
	if index.Relationships.References == nil ||
		index.Relationships.CallEdges == nil ||
		index.Relationships.PossibleCallEdges == nil ||
		index.Relationships.InterfaceImplementations == nil ||
		index.Relationships.TestReferences == nil ||
		index.Relationships.PackageRelationships == nil {
		t.Fatalf("relationship slices should be non-nil, got %#v", index.Relationships)
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

func TestNormalizeRelationshipIndexSortsDedupesAndRelativizes(t *testing.T) {
	root := t.TempDir()
	serviceFile := filepath.Join(root, "internal", "service", "service.go")
	otherFile := filepath.Join(root, "internal", "service", "other.go")
	target := SymbolIdentity{
		Package: ".",
		Name:    "Target",
		Position: sherpa.Position{
			File:   serviceFile,
			Line:   3,
			Column: 1,
		},
	}
	source := SymbolIdentity{
		Package: ".",
		Name:    "Caller",
		Position: sherpa.Position{
			File:   serviceFile,
			Line:   7,
			Column: 1,
		},
	}

	relationships := NormalizeRelationshipIndex(root, RelationshipIndex{
		References: []ReferenceRecord{
			{
				Package:       ".",
				File:          serviceFile,
				Source:        source,
				Target:        target,
				ReferenceKind: sherpa.ReferenceKindCall,
				Position: sherpa.Position{
					File:   serviceFile,
					Line:   8,
					Column: 2,
				},
				Range: &sherpa.SourceRange{
					Start: sherpa.Position{File: serviceFile, Line: 8, Column: 2},
					End:   sherpa.Position{File: serviceFile, Line: 8, Column: 8},
				},
				Limitations: []string{"beta", "alpha", "alpha"},
			},
			{
				Package:       ".",
				File:          serviceFile,
				Source:        source,
				Target:        target,
				ReferenceKind: sherpa.ReferenceKindCall,
				Position: sherpa.Position{
					File:   serviceFile,
					Line:   8,
					Column: 2,
				},
				Range: &sherpa.SourceRange{
					Start: sherpa.Position{File: serviceFile, Line: 8, Column: 2},
					End:   sherpa.Position{File: serviceFile, Line: 8, Column: 8},
				},
				Limitations: []string{"alpha", "beta"},
			},
			{
				Package: ".",
				File:    otherFile,
				Target:  SymbolIdentity{Package: ".", Name: "Other"},
				Position: sherpa.Position{
					File: otherFile,
					Line: 4,
				},
			},
		},
		CallEdges: []CallEdgeRecord{
			{
				Package:   ".",
				File:      serviceFile,
				Source:    source,
				Target:    target,
				CallScope: sherpa.CallScopeLocal,
				Position:  sherpa.Position{File: serviceFile, Line: 8, Column: 2},
			},
		},
		PossibleCallEdges: []PossibleCallEdgeRecord{
			{
				Package:   ".",
				File:      serviceFile,
				Source:    source,
				Target:    target,
				CallScope: sherpa.CallScopeDynamic,
				Reason:    "interface dispatch",
				Position:  sherpa.Position{File: serviceFile, Line: 9, Column: 2},
			},
		},
		InterfaceImplementations: []InterfaceImplementationRecord{
			{
				Package:        ".",
				File:           serviceFile,
				Interface:      SymbolIdentity{Package: ".", Name: "Runner"},
				Implementation: SymbolIdentity{Package: ".", Name: "Service"},
				Position:       sherpa.Position{File: serviceFile, Line: 12, Column: 1},
			},
		},
		TestReferences: []TestReferenceRecord{
			{
				Package:  ".",
				File:     filepath.Join(root, "service_test.go"),
				Test:     SymbolIdentity{Package: ".", Name: "TestTarget"},
				Target:   target,
				TestName: "TestTarget",
				Reasons:  []string{"same-package", "direct-reference", "direct-reference"},
				Position: sherpa.Position{File: filepath.Join(root, "service_test.go"), Line: 6, Column: 2},
			},
		},
		PackageRelationships: []PackageRelationshipRecord{
			{
				Package:        "./internal/api",
				RelatedPackage: "./internal/service",
				Reasons:        []string{"import", "import"},
			},
		},
	})

	if len(relationships.References) != 2 {
		t.Fatalf("expected duplicate references to be removed, got %#v", relationships.References)
	}
	if relationships.References[0].File != "internal/service/other.go" {
		t.Fatalf("references should be sorted by normalized key, got %#v", relationships.References)
	}
	if relationships.References[1].File != "internal/service/service.go" ||
		relationships.References[1].Position.File != "internal/service/service.go" ||
		relationships.References[1].Range.Start.File != "internal/service/service.go" ||
		relationships.References[1].Source.Position.File != "internal/service/service.go" {
		t.Fatalf("reference paths should be root-relative, got %#v", relationships.References[1])
	}
	if got := relationships.References[1].Limitations; len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("limitations should be sorted and deduped, got %#v", got)
	}
	if relationships.References[1].Kind != RelationshipKindReference ||
		relationships.References[1].Certainty != RelationshipCertaintyDirect {
		t.Fatalf("reference defaults not applied: %#v", relationships.References[1])
	}
	if relationships.CallEdges[0].Kind != RelationshipKindCall ||
		relationships.CallEdges[0].Certainty != RelationshipCertaintyDirect ||
		relationships.CallEdges[0].CallScope != sherpa.CallScopeLocal {
		t.Fatalf("call edge should keep scope separate from certainty: %#v", relationships.CallEdges[0])
	}
	if relationships.PossibleCallEdges[0].Kind != RelationshipKindPossibleCall ||
		relationships.PossibleCallEdges[0].Certainty != RelationshipCertaintyPossible ||
		relationships.PossibleCallEdges[0].CallScope != sherpa.CallScopeDynamic {
		t.Fatalf("possible call edge defaults not applied: %#v", relationships.PossibleCallEdges[0])
	}
	if relationships.InterfaceImplementations[0].Kind != RelationshipKindInterfaceImplementation ||
		relationships.InterfaceImplementations[0].Certainty != RelationshipCertaintyDirect {
		t.Fatalf("interface defaults not applied: %#v", relationships.InterfaceImplementations[0])
	}
	if got := relationships.TestReferences[0].Reasons; len(got) != 2 || got[0] != "direct-reference" || got[1] != "same-package" {
		t.Fatalf("test reasons should be sorted and deduped, got %#v", got)
	}
	if relationships.TestReferences[0].File != "service_test.go" ||
		relationships.TestReferences[0].Position.File != "service_test.go" {
		t.Fatalf("test reference paths should be root-relative, got %#v", relationships.TestReferences[0])
	}
	if relationships.PackageRelationships[0].Kind != RelationshipKindPackageRelationship ||
		relationships.PackageRelationships[0].Certainty != RelationshipCertaintyDirect {
		t.Fatalf("package relationship defaults not applied: %#v", relationships.PackageRelationships[0])
	}
}

func TestNormalizeRelationshipIndexReturnsNonNilEmptySlices(t *testing.T) {
	relationships := NormalizeRelationshipIndex("", RelationshipIndex{})

	if relationships.References == nil ||
		relationships.CallEdges == nil ||
		relationships.PossibleCallEdges == nil ||
		relationships.InterfaceImplementations == nil ||
		relationships.TestReferences == nil ||
		relationships.PackageRelationships == nil {
		t.Fatalf("expected non-nil empty slices, got %#v", relationships)
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
