package sherpa

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestFindImpactForSymbol(t *testing.T) {
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

	result, err := FindImpact(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "ParseFile" {
		t.Fatalf("expected ParseFile target, got %s", result.Target)
	}

	if result.Kind != ImpactKindSymbol {
		t.Fatalf("expected symbol impact, got %s", result.Kind)
	}

	if len(result.References) != 2 {
		t.Fatalf("expected 2 references, got %v", result.References)
	}

	callers := callTestCallerNames(result.Callers)
	assertContainsString(t, callers, "Run")

	assertContainsString(t, result.Packages, "./cmd/app")
	assertContainsString(t, result.Packages, "./internal/parser")
}

func TestFindImpactForTypeSymbolDoesNotWarnWhenCallersDoNotApply(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func Run(server Server) {}
`)

	result, err := FindImpact(tmp, "Server")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.References) != 2 {
		t.Fatalf("expected 2 references, got %v", result.References)
	}

	if len(result.Callers) != 0 {
		t.Fatalf("expected no callers, got %v", result.Callers)
	}

	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
}

func TestFindImpactForPackage(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), "package auth\n")
	writeFile(t, filepath.Join(tmp, "cmd", "api", "main.go"), `package main

import "example.com/app/internal/auth"
`)

	result, err := FindImpact(tmp, "./internal/auth")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "./internal/auth" {
		t.Fatalf("expected ./internal/auth target, got %s", result.Target)
	}

	if result.Kind != ImpactKindPackage {
		t.Fatalf("expected package impact, got %s", result.Kind)
	}

	assertContainsString(t, result.Dependencies.UsedBy, "./cmd/api")

	got := result.Packages
	want := []string{"./cmd/api", "./internal/auth"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestIsImpactPackageTarget(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{target: ".", want: true},
		{target: "./internal/auth", want: true},
		{target: "internal/auth", want: true},
		{target: "example.com/app/internal/auth", want: true},
		{target: "ParseFile", want: false},
		{target: "Server.Start", want: false},
	}

	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			got := isImpactPackageTarget(test.target)
			if got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}
