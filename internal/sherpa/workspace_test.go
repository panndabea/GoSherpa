package sherpa

import (
	"path/filepath"
	"testing"
)

func TestWorkspacePackagePathForImportPathUsesRootModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")

	got, ok := WorkspacePackagePathForImportPath(root, "example.com/app/internal/auth")
	if !ok {
		t.Fatal("expected root module import path to resolve")
	}
	if got != "./internal/auth" {
		t.Fatalf("expected ./internal/auth, got %s", got)
	}

	got, ok = WorkspacePackagePathForImportPath(root, "example.com/app")
	if !ok {
		t.Fatal("expected root module package to resolve")
	}
	if got != "." {
		t.Fatalf("expected ., got %s", got)
	}
}

func TestWorkspacePackagePathForImportPathUsesGoWorkModules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.work"), `go 1.24.4

use (
	./app
	./service
)
`)
	writeFile(t, filepath.Join(root, "app", "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeFile(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n\ngo 1.24.4\n")

	got, ok := WorkspacePackagePathForImportPath(root, "example.com/app/processor")
	if !ok {
		t.Fatal("expected app workspace import path to resolve")
	}
	if got != "./app/processor" {
		t.Fatalf("expected ./app/processor, got %s", got)
	}

	got, ok = WorkspacePackagePathForImportPath(root, "example.com/service")
	if !ok {
		t.Fatal("expected service workspace import path to resolve")
	}
	if got != "./service" {
		t.Fatalf("expected ./service, got %s", got)
	}
}

func TestWorkspacePackagePathForImportPathRejectsExternalImportPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.work"), `go 1.24.4

use ./app
`)
	writeFile(t, filepath.Join(root, "app", "go.mod"), "module example.com/app\n\ngo 1.24.4\n")

	if got, ok := WorkspacePackagePathForImportPath(root, "example.com/other"); ok {
		t.Fatalf("expected external import path not to resolve, got %s", got)
	}
}
