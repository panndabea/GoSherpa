package sherpa

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/semantics"
)

func TestFindReferences(t *testing.T) {
	tmp := t.TempDir()

	path := filepath.Join(tmp, "service.go")

	err := os.WriteFile(path, []byte(`
package auth

func ParseFile() {
}

func Run() {
	ParseFile()
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	refs, err := FindReferences(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d", len(refs))
	}

	assertReferenceKinds(t, refs, []ReferenceKind{ReferenceKindDefinition, ReferenceKindCall})
}

func TestFindReferencesReturnsRootRelativeFilePositions(t *testing.T) {
	tmp := t.TempDir()

	path := filepath.Join(tmp, "internal", "service", "service.go")

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(path, []byte(`
package service

func ParseFile() {
}

func Run() {
	ParseFile()
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	refs, err := FindReferences(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d", len(refs))
	}

	for _, ref := range refs {
		if ref.Position.File != "internal/service/service.go" {
			t.Fatalf("expected internal/service/service.go, got %s", ref.Position.File)
		}

		if strings.Contains(ref.Position.File, tmp) {
			t.Fatalf("expected root-relative path, got %s", ref.Position.File)
		}
	}
}

func TestFindReferencesIgnoresShadowedIdentifiers(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func ParseFile() {}

func Run() {
	ParseFile()

	ParseFile := func() {}
	ParseFile()
}
`)

	refs, err := FindReferences(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d: %v", len(refs), refs)
	}
}

func TestFindReferencesFindsTypeReferences(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func Run(server Server) Server {
	return server
}
`)

	refs, err := FindReferences(tmp, "Server")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 3 {
		t.Fatalf("expected 3 references, got %d: %v", len(refs), refs)
	}

	assertReferenceKinds(t, refs, []ReferenceKind{
		ReferenceKindDefinition,
		ReferenceKindTypeUsage,
		ReferenceKindTypeUsage,
	})
}

func TestFindReferencesFindsMethodReferences(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func (server Server) Start() {}

func Run(server Server) {
	server.Start()
}
`)

	refs, err := FindReferences(tmp, "Server.Start")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d: %v", len(refs), refs)
	}

	assertReferenceKinds(t, refs, []ReferenceKind{ReferenceKindDefinition, ReferenceKindCall})
}

func TestFindReferencesFindsFieldAccessReferences(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct {
	Name string
}

func Run(server Server) string {
	return server.Name
}
`)

	refs, err := FindReferences(tmp, "Server.Name")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d: %v", len(refs), refs)
	}

	assertReferenceKinds(t, refs, []ReferenceKind{ReferenceKindDefinition, ReferenceKindFieldAccess})
}

func TestFindReferencesFiltersByKind(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func ParseFile() {}

func Run() {
	ParseFile()
}
`)

	refs, err := FindReferencesWithOptions(tmp, "ParseFile", ReferenceOptions{Kind: ReferenceKindCall})
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d: %v", len(refs), refs)
	}
	if refs[0].Kind != ReferenceKindCall {
		t.Fatalf("expected call reference, got %s", refs[0].Kind)
	}
}

func TestFindReferencesFindsLocalPackageSelectorReferences(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/parser"

func Run() {
	parser.ParseFile()
}
`)

	refs, err := FindReferences(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d: %v", len(refs), refs)
	}

	files := referenceTestFiles(refs)
	assertContainsString(t, files, "internal/parser/parser.go")
	assertContainsString(t, files, "cmd/app/main.go")
	assertReferenceKinds(t, refs, []ReferenceKind{ReferenceKindCall, ReferenceKindDefinition})
}

func TestFindReferenceReportUsesGoPackagesForLocalPackages(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/parser"

func Run() {
	parser.ParseFile()
}
`)

	report, err := FindReferenceReport(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if report.AnalysisMode != ReferenceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %s", report.AnalysisMode)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", report.Warnings)
	}
	if len(report.References) != 2 {
		t.Fatalf("expected 2 references, got %d: %v", len(report.References), report.References)
	}
}

func TestFindReferencesHonorsPackageQualifiedTargets(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}

func New(session Session) Session {
	return session
}
`)
	writeFile(t, filepath.Join(tmp, "internal", "billing", "session.go"), `package billing

type Session struct{}

func New(session Session) Session {
	return session
}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import (
	auth "example.com/app/internal/auth"
	"example.com/app/internal/billing"
)

func Run() {
	_ = auth.Session{}
	_ = billing.Session{}
}
`)

	refs, err := FindReferences(tmp, "./internal/auth.Session")
	if err != nil {
		t.Fatal(err)
	}

	files := referenceTestFiles(refs)
	assertContainsString(t, files, "internal/auth/session.go")
	assertContainsString(t, files, "cmd/app/main.go")
	if containsString(files, "internal/billing/session.go") {
		t.Fatalf("expected auth Session refs only, got %v", refs)
	}
}

func TestFindReferenceReportTypecheckedPackageQualifiedSelectors(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "auth.go"), `package auth

type Session struct{}

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "billing", "billing.go"), `package billing

type Session struct{}

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import (
	identity "example.com/app/internal/auth"
	money "example.com/app/internal/billing"
)

func Run() {
	identity.Target()
	_ = identity.Session{}
	money.Target()
	_ = money.Session{}
}
`)

	tests := []struct {
		name       string
		target     string
		wantKind   ReferenceKind
		wantLine   int
		blockFiles []string
	}{
		{
			name:       "function",
			target:     "./internal/auth.Target",
			wantKind:   ReferenceKindCall,
			wantLine:   9,
			blockFiles: []string{"internal/billing/billing.go"},
		},
		{
			name:       "type",
			target:     "./internal/auth.Session",
			wantKind:   ReferenceKindTypeUsage,
			wantLine:   10,
			blockFiles: []string{"internal/billing/billing.go"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, err := FindReferenceReport(tmp, test.target)
			if err != nil {
				t.Fatal(err)
			}

			if report.AnalysisMode != ReferenceAnalysisModeTypechecked {
				t.Fatalf("expected typechecked analysis mode, got %s", report.AnalysisMode)
			}
			if len(report.Warnings) != 0 {
				t.Fatalf("expected no warnings, got %v", report.Warnings)
			}

			files := referenceTestFiles(report.References)
			assertContainsString(t, files, "internal/auth/auth.go")
			assertContainsString(t, files, "cmd/app/main.go")
			for _, blocked := range test.blockFiles {
				if containsString(files, blocked) {
					t.Fatalf("expected refs to exclude %s, got %v", blocked, report.References)
				}
			}
			assertReferenceAt(t, report.References, "cmd/app/main.go", test.wantLine, test.wantKind)
		})
	}
}

func TestFindReferenceReportReturnsPackageLoadWarnings(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}

func Broken() {
	Missing()
}
`)

	report, err := FindReferenceReport(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	if report.AnalysisMode != ReferenceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %s", report.AnalysisMode)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected package load warnings")
	}
	if !strings.Contains(strings.Join(report.Warnings, "\n"), "Missing") {
		t.Fatalf("expected warning to mention Missing, got %v", report.Warnings)
	}
}

func TestFindReferenceReportFallsBackToASTWhenLoaderFails(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}

func Run() {
	Target()
}
`)

	oldLoader := loadSemanticReferenceRepository
	loadSemanticReferenceRepository = func(string, semantics.LoadOptions) (semantics.Repository, error) {
		return semantics.Repository{}, fmt.Errorf("loader failed")
	}
	t.Cleanup(func() {
		loadSemanticReferenceRepository = oldLoader
	})

	report, err := FindReferenceReport(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	if report.AnalysisMode != ReferenceAnalysisModeASTFallback {
		t.Fatalf("expected ast-fallback analysis mode, got %s", report.AnalysisMode)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected loader warning")
	}
	if !strings.Contains(report.Warnings[0], "loader failed") {
		t.Fatalf("expected loader warning, got %v", report.Warnings)
	}
	if len(report.References) != 2 {
		t.Fatalf("expected 2 references from AST fallback, got %d: %v", len(report.References), report.References)
	}
}

func TestNormalizeReferenceTargetRejectsInvalidInput(t *testing.T) {
	tmp := t.TempDir()

	tests := []string{
		"",
		"   ",
		"Server.",
		".Start",
		"A.B.C",
		"github.com/example/app.ParseFile",
		`C:\repo\Run`,
		"not valid",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := normalizeReferenceTarget(tmp, test)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeReferenceTargetDisplaysModuleRootPackageTarget(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")

	got, err := normalizeReferenceTarget(tmp, "example.com/app.Run")
	if err != nil {
		t.Fatal(err)
	}

	if got.Package != "." {
		t.Fatalf("expected root package, got %s", got.Package)
	}
	if got.String() != "Run" {
		t.Fatalf("expected Run, got %s", got.String())
	}
}

func referenceTestFiles(refs []Reference) []string {
	var files []string
	for _, ref := range refs {
		files = append(files, ref.Position.File)
	}

	return files
}

func assertReferenceKinds(t *testing.T, refs []Reference, want []ReferenceKind) {
	t.Helper()

	var got []ReferenceKind
	for _, ref := range refs {
		got = append(got, ref.Kind)
	}

	if len(got) != len(want) {
		t.Fatalf("expected kinds %v, got %v", want, got)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("expected kinds %v, got %v", want, got)
		}
	}
}

func assertReferenceAt(t *testing.T, refs []Reference, file string, line int, kind ReferenceKind) {
	t.Helper()

	for _, ref := range refs {
		if ref.Position.File == file && ref.Position.Line == line && ref.Kind == kind {
			return
		}
	}

	t.Fatalf("expected %s reference at %s:%d, got %v", kind, file, line, refs)
}
