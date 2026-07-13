package impact

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
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
	if report.ReferenceAnalysisMode != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s", report.ReferenceAnalysisMode, sherpa.ReferenceAnalysisModeTypechecked)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/api:TestHandler", "./internal/auth:TestSession"})
	assertStrings(t, relatedTestReasons(report.AffectedTests, "./internal/auth", "TestSession"), []string{
		sherpa.RelatedTestReasonTargetPackage,
		sherpa.RelatedTestReasonChangedSymbol,
	})
	assertStrings(t, relatedTestReasons(report.AffectedTests, "./internal/api", "TestHandler"), []string{
		sherpa.RelatedTestReasonCallerPackage,
	})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/api", "go test ./internal/auth"})
	assertStrings(t, report.Warnings, []string{})
}

func TestAnalyzeDiffUsesChangedSymbolsForDirectTestPlan(t *testing.T) {
	root := initImpactGitTestRepository(t)

	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	return Session{}
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestAuthNewSession(t *testing.T) {
	_ = NewSession()
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler.go"), `package api

import "example.com/app/internal/auth"

func Build() auth.Session {
	return auth.NewSession()
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler_test.go"), `package api

import (
	"testing"

	"example.com/app/internal/auth"
)

func TestAPINewSession(t *testing.T) {
	_ = auth.NewSession()
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "billing", "session.go"), `package billing

type Session struct{}

func NewSession() Session {
	return Session{}
}

func Touch() string {
	return "old"
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "billing", "session_test.go"), `package billing

import "testing"

func TestBillingNewSession(t *testing.T) {
	_ = NewSession()
}
`)
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	session := Session{}
	return session
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "billing", "session.go"), `package billing

type Session struct{}

func NewSession() Session {
	return Session{}
}

func Touch() string {
	return "new"
}
`)

	report, err := AnalyzeDiff(root, base, "")
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	assertStrings(t, report.AffectedSymbols, []string{"NewSession", "Touch"})
	assertStrings(t, directRelatedTestNames(report.AffectedTests), []string{
		"./internal/api:TestAPINewSession",
		"./internal/auth:TestAuthNewSession",
	})
	assertStrings(t, testPlanItemPackages(report.TestPlan.Direct), []string{"./internal/api", "./internal/auth"})
	assertStrings(t, testPlanItemPackages(report.TestPlan.Related), []string{"./internal/billing"})
	assertStrings(t, testPlanItemTargets(report.TestPlan.Direct, "./internal/api"), []string{"./internal/auth.NewSession"})
	assertStrings(t, testPlanItemTargets(report.TestPlan.Direct, "./internal/auth"), []string{"./internal/auth.NewSession"})
	assertStrings(t, testPlanItemTargets(report.TestPlan.Related, "./internal/billing"), []string{"./internal/billing.Touch"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/api", "go test ./internal/auth", "go test ./internal/billing"})
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
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/api:TestHandler", "./internal/auth:TestSession"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/api", "go test ./internal/auth"})
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
	assertStrings(t, report.TestCommands, []string{"go test ./cmd/app", "go test ./internal/auth"})
}

func TestAnalyzeSymbolReportsTypeAliasImpact(t *testing.T) {
	root := writeTypeAliasImpactProject(t)

	report, err := AnalyzeSymbol(root, "./internal/auth.UserID")
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	assertStrings(t, report.AffectedSymbols, []string{"./internal/auth.UserID"})
	assertStrings(t, report.AffectedPackages, []string{"./internal/auth", "./internal/session"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/auth:TestNormalizeUserID", "./internal/session:TestLoadUserID"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/auth", "go test ./internal/session"})
	if report.ReferenceAnalysisMode != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s", report.ReferenceAnalysisMode, sherpa.ReferenceAnalysisModeTypechecked)
	}
	if report.InterfaceAnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, InterfaceAnalysisModeTypechecked)
	}
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
	assertStrings(t, report.AffectedPackages, []string{"./internal/auth", "./internal/jwt", "./internal/session"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/auth:TestAuthenticatorContract", "./internal/jwt:TestJWTAuthenticatorContract"})
	assertStrings(t, relatedTestReasons(report.AffectedTests, "./internal/auth", "TestAuthenticatorContract"), []string{
		sherpa.RelatedTestReasonTargetPackage,
		sherpa.RelatedTestReasonContract,
	})
	assertStrings(t, relatedTestReasons(report.AffectedTests, "./internal/jwt", "TestJWTAuthenticatorContract"), []string{
		sherpa.RelatedTestReasonCallerPackage,
		sherpa.RelatedTestReasonContract,
	})
	assertStrings(t, testPlanItemPackages(report.TestPlan.Contracts), []string{"./internal/auth", "./internal/jwt"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/auth", "go test ./internal/jwt", "go test ./internal/session"})
	if report.InterfaceAnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked interface analysis mode, got %q", report.InterfaceAnalysisMode)
	}
}

func TestAnalyzePackageReportsInterfacesAndImplementationsForChangedImplementationPackage(t *testing.T) {
	root := writeInterfaceImpactProject(t)

	report, err := AnalyzePackage(root, "./internal/jwt")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.AffectedInterfaces, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/jwt.JWTAuthenticator"})
	if report.InterfaceAnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked interface analysis mode, got %q", report.InterfaceAnalysisMode)
	}
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

func TestAnalyzePackageUsesImportIdentityForInterfaceMethodSignatures(t *testing.T) {
	root := writeInterfaceImportIdentityProject(t)

	report, err := AnalyzePackage(root, "./internal/auth")
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	assertStrings(t, report.AffectedInterfaces, []string{"./internal/auth.Authenticator"})
	assertStrings(t, report.AffectedImplementations, []string{"./internal/good.GoodAuthenticator"})
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
	assertStrings(t, report.AffectedPackages, []string{"./internal/auth", "./internal/jwt", "./internal/session"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/auth:TestAuthenticatorContract", "./internal/jwt:TestJWTAuthenticatorContract"})
	assertStrings(t, testPlanItemPackages(report.TestPlan.Contracts), []string{"./internal/auth", "./internal/jwt"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/auth", "go test ./internal/jwt", "go test ./internal/session"})
	if report.InterfaceAnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked interface analysis mode, got %q", report.InterfaceAnalysisMode)
	}
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
	if report.InterfaceAnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked interface analysis mode, got %q", report.InterfaceAnalysisMode)
	}
}

func TestAnalyzeSymbolReportsFallbackInterfaceAnalysisMode(t *testing.T) {
	oldLoader := loadSemanticInterfaceRepository
	loadSemanticInterfaceRepository = func(string, semantics.LoadOptions) (semantics.Repository, error) {
		return semantics.Repository{}, errors.New("loader failed")
	}
	defer func() {
		loadSemanticInterfaceRepository = oldLoader
	}()

	root := writeInterfaceImpactProject(t)

	report, err := AnalyzeSymbol(root, "./internal/jwt.JWTAuthenticator")
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.InterfaceAnalysisMode != InterfaceAnalysisModeASTFallback {
		t.Fatalf("expected AST fallback interface analysis mode, got %q", report.InterfaceAnalysisMode)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], "typechecked interface analysis unavailable: loader failed") {
		t.Fatalf("expected typechecked fallback warning, got %#v", report.Warnings)
	}
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
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth_test.go"), `package auth

import "testing"

func TestAuthenticatorContract(t *testing.T) {}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "jwt", "jwt.go"), `package jwt

type JWTAuthenticator struct{}

func (JWTAuthenticator) Authenticate() error {
	return nil
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "jwt", "jwt_test.go"), `package jwt

import "testing"

func TestJWTAuthenticatorContract(t *testing.T) {}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "session", "session.go"), `package session

import "example.com/app/internal/auth"

type SessionStore struct{}

func Run(authenticator auth.Authenticator) error {
	return authenticator.Authenticate()
}
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

func writeInterfaceImportIdentityProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "authmodel", "user.go"), `package model

type User struct{}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "sessionmodel", "user.go"), `package model

type User struct{}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

import (
	ctx "context"
	model "example.com/app/internal/authmodel"
)

type Authenticator interface {
	Authenticate(ctx.Context, model.User) error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "good", "good.go"), `package good

import (
	"context"
	authuser "example.com/app/internal/authmodel"
)

type GoodAuthenticator struct{}

func (GoodAuthenticator) Authenticate(context.Context, authuser.User) error {
	return nil
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "bad", "bad.go"), `package bad

import (
	ctx "context"
	model "example.com/app/internal/sessionmodel"
)

type BadAuthenticator struct{}

func (BadAuthenticator) Authenticate(ctx.Context, model.User) error {
	return nil
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

func writeTypeAliasImpactProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "model", "user.go"), `package model

type UserID string
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

import "example.com/app/internal/model"

type UserID = model.UserID

func Normalize(id UserID) UserID {
	return id
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth_test.go"), `package auth

import "testing"

func TestNormalizeUserID(t *testing.T) {
	var id UserID
	_ = Normalize(id)
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "session", "session.go"), `package session

import "example.com/app/internal/auth"

func Load(id auth.UserID) auth.UserID {
	return id
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "session", "session_test.go"), `package session

import (
	"testing"

	"example.com/app/internal/auth"
)

func TestLoadUserID(t *testing.T) {
	var id auth.UserID
	_ = Load(id)
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

func directRelatedTestNames(tests []RelatedTest) []string {
	names := make([]string, 0, len(tests))
	for _, test := range tests {
		if !test.DirectReference {
			continue
		}

		names = append(names, test.Package+":"+test.Name)
	}

	return names
}

func relatedTestReasons(tests []RelatedTest, pkg string, name string) []string {
	for _, test := range tests {
		if test.Package == pkg && test.Name == name {
			return test.Reasons
		}
	}

	return nil
}

func testPlanItemPackages(items []sherpa.TestPlanItem) []string {
	packages := make([]string, 0, len(items))
	for _, item := range items {
		packages = append(packages, item.Package)
	}

	return packages
}

func testPlanItemTargets(items []sherpa.TestPlanItem, pkg string) []string {
	for _, item := range items {
		if item.Package == pkg {
			return item.Targets
		}
	}

	return nil
}
