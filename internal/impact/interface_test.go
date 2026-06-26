package impact

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
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
