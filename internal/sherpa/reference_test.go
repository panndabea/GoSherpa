package sherpa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
