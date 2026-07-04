package impact

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func TestFindImplementersReturnsImplementations(t *testing.T) {
	root := writeInterfaceImpactProject(t)

	result, err := FindImplementers(root, "./internal/auth.Authenticator")
	if err != nil {
		t.Fatalf("FindImplementers returned error: %v", err)
	}

	if result.Target != "./internal/auth.Authenticator" {
		t.Fatalf("expected target ./internal/auth.Authenticator, got %s", result.Target)
	}
	if len(result.Implementers) != 1 {
		t.Fatalf("expected 1 implementer, got %#v", result.Implementers)
	}
	if result.Implementers[0].Name != "./internal/jwt.JWTAuthenticator" {
		t.Fatalf("expected JWTAuthenticator implementer, got %#v", result.Implementers[0])
	}
	if result.Implementers[0].Position.File != "internal/jwt/jwt.go" {
		t.Fatalf("expected root-relative position, got %#v", result.Implementers[0].Position)
	}
	if result.AnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %q", result.AnalysisMode)
	}
}

func TestFindInterfacesReturnsSatisfiedInterfaces(t *testing.T) {
	root := writeInterfaceImpactProject(t)

	result, err := FindInterfaces(root, "./internal/jwt.JWTAuthenticator")
	if err != nil {
		t.Fatalf("FindInterfaces returned error: %v", err)
	}

	if result.Target != "./internal/jwt.JWTAuthenticator" {
		t.Fatalf("expected target ./internal/jwt.JWTAuthenticator, got %s", result.Target)
	}
	if len(result.Interfaces) != 1 {
		t.Fatalf("expected 1 interface, got %#v", result.Interfaces)
	}
	if result.Interfaces[0].Name != "./internal/auth.Authenticator" {
		t.Fatalf("expected Authenticator interface, got %#v", result.Interfaces[0])
	}
	if result.Interfaces[0].Position.File != "internal/auth/auth.go" {
		t.Fatalf("expected root-relative position, got %#v", result.Interfaces[0].Position)
	}
	if result.AnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %q", result.AnalysisMode)
	}
}

func TestFindImplementersUsesTypecheckedAliases(t *testing.T) {
	root := writeInterfaceAliasProject(t)

	result, err := FindImplementers(root, "./internal/auth.Authenticator")
	if err != nil {
		t.Fatalf("FindImplementers returned error: %v", err)
	}

	if result.AnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %q", result.AnalysisMode)
	}
	if len(result.Implementers) != 1 {
		t.Fatalf("expected 1 implementer, got %#v", result.Implementers)
	}
	if result.Implementers[0].Name != "./internal/jwt.JWTAuthenticator" {
		t.Fatalf("expected JWTAuthenticator implementer, got %#v", result.Implementers[0])
	}
}

func TestFindImplementersUsesGenericPointerReceiver(t *testing.T) {
	root := writeGenericPointerReceiverProject(t)

	result, err := FindImplementers(root, "./internal/cache.Flusher")
	if err != nil {
		t.Fatalf("FindImplementers returned error: %v", err)
	}

	if result.AnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %q", result.AnalysisMode)
	}
	if len(result.Implementers) != 1 {
		t.Fatalf("expected 1 implementer, got %#v", result.Implementers)
	}
	if result.Implementers[0].Name != "./internal/cache.Cache" {
		t.Fatalf("expected Cache implementer, got %#v", result.Implementers[0])
	}
}

func TestFindImplementersWithOptionsHonorsBuildTags(t *testing.T) {
	root := t.TempDir()

	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeImpactTestFile(t, filepath.Join(root, "service.go"), `package service

type Runner interface {
	Run() error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "enterprise.go"), `//go:build enterprise

package service

type EnterpriseRunner struct{}

func (EnterpriseRunner) Run() error {
	return nil
}
`)

	withoutTags, err := FindImplementersWithOptions(root, "Runner", InterfaceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutTags.Implementers) != 0 {
		t.Fatalf("expected no tagged implementers without tag, got %#v", withoutTags.Implementers)
	}

	withTags, err := FindImplementersWithOptions(root, "Runner", InterfaceOptions{
		BuildTags: []string{"enterprise"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if withTags.AnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %q", withTags.AnalysisMode)
	}
	if len(withTags.Implementers) != 1 || withTags.Implementers[0].Name != "EnterpriseRunner" {
		t.Fatalf("expected tagged implementer, got %#v", withTags.Implementers)
	}
}

func TestFindImplementersUsesTypecheckedLoaderAcrossGoWorkModules(t *testing.T) {
	root := t.TempDir()

	writeImpactTestFile(t, filepath.Join(root, "go.work"), `go 1.24.4

use (
	./app
	./service
)
`)
	writeImpactTestFile(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "service", "service.go"), `package service

type Payload struct{}

type Processor interface {
	Process(Payload) error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "app", "go.mod"), `module example.com/app

go 1.24.4

require example.com/service v0.0.0
`)
	writeImpactTestFile(t, filepath.Join(root, "app", "processor", "processor.go"), `package processor

import "example.com/service"

type LocalProcessor struct{}

func (LocalProcessor) Process(service.Payload) error {
	return nil
}
`)

	result, err := FindImplementers(root, "./service.Processor")
	if err != nil {
		t.Fatalf("FindImplementers returned error: %v", err)
	}

	if result.AnalysisMode != InterfaceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %q with warnings %#v", result.AnalysisMode, result.Warnings)
	}
	if len(result.Implementers) != 1 {
		t.Fatalf("expected 1 implementer, got %#v", result.Implementers)
	}
	if result.Implementers[0].Name != "./app/processor.LocalProcessor" {
		t.Fatalf("expected LocalProcessor implementer, got %#v", result.Implementers[0])
	}
}

func TestFindInterfacesFallsBackWhenTypecheckedLoadingFails(t *testing.T) {
	oldLoader := loadSemanticInterfaceRepository
	loadSemanticInterfaceRepository = func(string, semantics.LoadOptions) (semantics.Repository, error) {
		return semantics.Repository{}, errors.New("loader failed")
	}
	defer func() {
		loadSemanticInterfaceRepository = oldLoader
	}()

	root := writeInterfaceImpactProject(t)

	result, err := FindInterfaces(root, "./internal/jwt.JWTAuthenticator")
	if err != nil {
		t.Fatalf("FindInterfaces returned error: %v", err)
	}

	if result.AnalysisMode != InterfaceAnalysisModeASTFallback {
		t.Fatalf("expected AST fallback analysis mode, got %q", result.AnalysisMode)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "typechecked interface analysis unavailable: loader failed") {
		t.Fatalf("expected typechecked fallback warning, got %#v", result.Warnings)
	}
	if len(result.Interfaces) != 1 || result.Interfaces[0].Name != "./internal/auth.Authenticator" {
		t.Fatalf("expected fallback interface result, got %#v", result.Interfaces)
	}
}

func TestFindImplementersReportsAmbiguousInterfaceTargets(t *testing.T) {
	root := writeAmbiguousInterfaceProject(t)

	_, err := FindImplementers(root, "Store")
	if err == nil {
		t.Fatal("expected ambiguous target error")
	}

	ambiguity, ok := err.(*sherpa.AmbiguousTargetError)
	if !ok {
		t.Fatalf("expected AmbiguousTargetError, got %T: %v", err, err)
	}
	if ambiguity.Kind != "interface" {
		t.Fatalf("expected interface ambiguity, got %s", ambiguity.Kind)
	}
	if len(ambiguity.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", ambiguity.Candidates)
	}
	if !strings.Contains(err.Error(), "./internal/auth.Store") {
		t.Fatalf("expected package-qualified example, got %v", err)
	}
}

func TestFindInterfacesReportsAmbiguousTypeTargets(t *testing.T) {
	root := writeAmbiguousInterfaceProject(t)

	_, err := FindInterfaces(root, "FileStore")
	if err == nil {
		t.Fatal("expected ambiguous target error")
	}

	ambiguity, ok := err.(*sherpa.AmbiguousTargetError)
	if !ok {
		t.Fatalf("expected AmbiguousTargetError, got %T: %v", err, err)
	}
	if ambiguity.Kind != "type" {
		t.Fatalf("expected type ambiguity, got %s", ambiguity.Kind)
	}
	if len(ambiguity.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", ambiguity.Candidates)
	}
	if !strings.Contains(err.Error(), "./internal/local.FileStore") {
		t.Fatalf("expected package-qualified example, got %v", err)
	}
}

func writeAmbiguousInterfaceProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "store.go"), `package auth

type Store interface {
	Save() error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "billing", "store.go"), `package billing

type Store interface {
	Save() error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "local", "store.go"), `package local

type FileStore struct{}

func (FileStore) Save() error {
	return nil
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "remote", "store.go"), `package remote

type FileStore struct{}

func (FileStore) Save() error {
	return nil
}
`)

	return root
}

func writeInterfaceAliasProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "model", "user.go"), `package model

type UserID string
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

import "example.com/app/internal/model"

type UserID = model.UserID

type Authenticator interface {
	Authenticate(UserID) error
}
`)
	writeImpactTestFile(t, filepath.Join(root, "internal", "jwt", "jwt.go"), `package jwt

import "example.com/app/internal/model"

type JWTAuthenticator struct{}

func (JWTAuthenticator) Authenticate(model.UserID) error {
	return nil
}
`)

	return root
}

func writeGenericPointerReceiverProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "cache", "cache.go"), `package cache

type Flusher interface {
	Flush() error
}

type Cache[T any] struct{}

func (*Cache[T]) Flush() error {
	return nil
}
`)

	return root
}
