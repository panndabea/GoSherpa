package impact

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyzeDiffReportsChangedAndAffectedPackages(t *testing.T) {
	root := initImpactGitTestRepository(t)

	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# test\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n\ntype Session struct{}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session_test.go"), "package auth\n\nimport \"testing\"\n\nfunc TestSession(t *testing.T) {}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler.go"), "package api\n\nimport \"example.com/app/internal/auth\"\n\nvar _ = auth.Session{}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler_test.go"), "package api\n\nimport \"testing\"\n\nfunc TestHandler(t *testing.T) {}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "other", "other.go"), "package other\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "other", "other_test.go"), "package other\n\nimport \"testing\"\n\nfunc TestOther(t *testing.T) {}\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# changed\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n\ntype Session struct{}\n\nfunc NewSession() Session { return Session{} }\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "change auth")
	head := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	report, err := AnalyzeDiff(root, base, head)
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	assertStrings(t, report.ChangedFiles, []string{"README.md", "internal/auth/session.go"})
	assertStrings(t, report.ChangedPackages, []string{"./internal/auth"})
	assertStrings(t, report.AffectedPackages, []string{"./internal/api", "./internal/auth"})
	assertStrings(t, report.AffectedSymbols, []string{"NewSession"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/api:TestHandler", "./internal/auth:TestSession"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/api", "go test ./internal/auth"})
	assertStrings(t, report.Warnings, []string{})
}

func TestAnalyzeDiffReturnsEmptyImpactForNonGoChanges(t *testing.T) {
	root := initImpactGitTestRepository(t)

	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# test\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# changed\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "docs")
	head := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	report, err := NewAnalyzer(root).AnalyzeDiff(base, head)
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	assertStrings(t, report.ChangedFiles, []string{"README.md"})
	assertStrings(t, report.ChangedPackages, []string{})
	assertStrings(t, report.AffectedPackages, []string{})
	assertStrings(t, report.AffectedSymbols, []string{})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{})
	assertStrings(t, report.TestCommands, []string{})
	assertStrings(t, report.Warnings, []string{})
}

func TestAnalyzeFileReportsPackageImpact(t *testing.T) {
	root := writeImpactAnalysisProject(t)

	report, err := AnalyzeFile(root, "internal/auth/session.go")
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	assertStrings(t, report.ChangedFiles, []string{"internal/auth/session.go"})
	assertStrings(t, report.ChangedPackages, []string{"./internal/auth"})
	assertStrings(t, report.AffectedPackages, []string{"./internal/api", "./internal/auth"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/api:TestHandler", "./internal/auth:TestSession"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/api", "go test ./internal/auth"})
}

func TestAnalyzePackageReportsPackageImpact(t *testing.T) {
	root := writeImpactAnalysisProject(t)

	report, err := NewAnalyzer(root).AnalyzePackage("./internal/auth")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.ChangedFiles, []string{})
	assertStrings(t, report.ChangedPackages, []string{"./internal/auth"})
	assertStrings(t, report.AffectedPackages, []string{"./internal/api", "./internal/auth"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/api:TestHandler", "./internal/auth:TestSession"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/api", "go test ./internal/auth"})
}

func TestAnalyzeSymbolReportsSymbolImpact(t *testing.T) {
	root := writeImpactAnalysisProject(t)

	report, err := AnalyzeSymbol(root, "./internal/auth.Session")
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	assertStrings(t, report.ChangedFiles, []string{})
	assertStrings(t, report.ChangedPackages, []string{})
	assertStrings(t, report.AffectedSymbols, []string{"./internal/auth.Session"})
	assertStrings(t, report.AffectedPackages, []string{"./internal/api", "./internal/auth"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/auth:TestSession"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/auth"})
}

func TestAnalyzeSymbolHonorsPackageQualifiedTargets(t *testing.T) {
	root := writePackageQualifiedSymbolImpactProject(t)

	report, err := AnalyzeSymbol(root, "./internal/auth.Session")
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	assertStrings(t, report.AffectedSymbols, []string{"./internal/auth.Session"})
	assertStrings(t, report.AffectedPackages, []string{"./cmd/app", "./internal/auth"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/auth:TestAuthSession"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/auth"})
}

func TestAnalyzeFileRejectsNonGoFiles(t *testing.T) {
	root := writeImpactAnalysisProject(t)

	_, err := AnalyzeFile(root, "README.md")
	if err == nil {
		t.Fatal("AnalyzeFile returned nil error")
	}
	if !strings.Contains(err.Error(), "impact file target must be a repository-local Go file") {
		t.Fatalf("AnalyzeFile error = %q, want Go file error", err)
	}
}

func TestAnalyzePackageReportsInterfacesAndImplementationsForChangedInterfacePackage(t *testing.T) {
	root := writeInterfaceImpactProject(t)

	report, err := AnalyzePackage(root, "./internal/auth")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.AffectedInterfaces, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/jwt.JWTAuthenticator"})
}

func TestAnalyzePackageReportsInterfacesAndImplementationsForChangedImplementationPackage(t *testing.T) {
	root := writeInterfaceImpactProject(t)

	report, err := AnalyzePackage(root, "./internal/jwt")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.AffectedInterfaces, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/jwt.JWTAuthenticator"})
}

func TestAnalyzePackageRequiresMatchingInterfaceMethodSignatures(t *testing.T) {
	root := writeInterfaceSignatureProject(t)

	report, err := AnalyzePackage(root, "./internal/auth")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.AffectedInterfaces, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/jwt.JWTAuthenticator"})
}

func TestAnalyzePackageResolvesEmbeddedInterfaceMethodSets(t *testing.T) {
	root := writeEmbeddedInterfaceImpactProject(t)

	report, err := AnalyzePackage(root, "./internal/store")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.AffectedInterfaces, []string{
		"./internal/auth.ReadWriter",
		"./internal/auth.Writer",
		"./internal/base.Reader",
	})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/store.FileStore"})
}

func TestAnalyzePackageReportsInterfacesEmbeddingChangedInterface(t *testing.T) {
	root := writeEmbeddedInterfaceImpactProject(t)

	report, err := AnalyzePackage(root, "./internal/base")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.AffectedInterfaces, []string{
		"./internal/auth.ReadWriter",
		"./internal/base.Reader",
	})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/store.FileStore"})
}

func TestAnalyzeSymbolReportsInterfaceImplementations(t *testing.T) {
	root := writeInterfaceImpactProject(t)

	report, err := AnalyzeSymbol(root, "./internal/auth.Authenticator")
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	assertStrings(t, report.AffectedSymbols, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedInterfaces, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/jwt.JWTAuthenticator"})
}

func TestAnalyzeSymbolReportsImplementedInterfaces(t *testing.T) {
	root := writeInterfaceImpactProject(t)

	report, err := AnalyzeSymbol(root, "./internal/jwt.JWTAuthenticator")
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	assertStrings(t, report.AffectedSymbols, []string{"./internal/jwt.JWTAuthenticator"})
	assertStrings(t, report.AffectedInterfaces, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/jwt.JWTAuthenticator"})
}

func writeImpactAnalysisProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n\ntype Session struct{}\n\nfunc NewSession() Session { return Session{} }\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session_test.go"), "package auth\n\nimport \"testing\"\n\nfunc TestSession(t *testing.T) { _ = Session{} }\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler.go"), "package api\n\nimport \"example.com/app/internal/auth\"\n\nvar _ = auth.Session{}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler_test.go"), "package api\n\nimport \"testing\"\n\nfunc TestHandler(t *testing.T) {}\n")

	return root
}

func writeInterfaceImpactProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

type Authenticator interface {
	Authenticate() error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "jwt", "jwt.go"), `package jwt

type JWTAuthenticator struct{}

func (JWTAuthenticator) Authenticate() error {
	return nil
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "session", "session.go"), `package session

type SessionStore struct{}
`)

	return root
}

func writeInterfaceSignatureProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

type Authenticator interface {
	Authenticate(user string) error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "jwt", "jwt.go"), `package jwt

type JWTAuthenticator struct{}

func (JWTAuthenticator) Authenticate(name string) error {
	return nil
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "bad", "bad.go"), `package bad

type BadAuthenticator struct{}

func (BadAuthenticator) Authenticate() error {
	return nil
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "count", "count.go"), `package count

type CountAuthenticator struct{}

func (CountAuthenticator) Authenticate(user string) (bool, error) {
	return false, nil
}
`)

	return root
}

func writePackageQualifiedSymbolImpactProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestAuthSession(t *testing.T) {}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "billing", "session.go"), `package billing

type Session struct{}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "billing", "session_test.go"), `package billing

import "testing"

func TestBillingSession(t *testing.T) {}
`)
	writeImpactTestFile(t, filepath.Join(root, "cmd", "app", "main.go"), `package main

import (
	auth "example.com/app/internal/auth"
	"example.com/app/internal/billing"
)

func Run() {
	_ = auth.Session{}
	_ = billing.Session{}
}
`)

	return root
}

func writeEmbeddedInterfaceImpactProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "base", "reader.go"), `package base

type Reader interface {
	Read() error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

import ports "example.com/app/internal/base"

type Writer interface {
	Write() error
}

type ReadWriter interface {
	ports.Reader
	Writer
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "store", "store.go"), `package store

type FileStore struct{}

func (FileStore) Read() error {
	return nil
}

func (FileStore) Write() error {
	return nil
}
`)

	return root
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func relatedTestNames(tests []RelatedTest) []string {
	names := make([]string, 0, len(tests))
	for _, test := range tests {
		names = append(names, test.Package+":"+test.Name)
	}

	return names
}
