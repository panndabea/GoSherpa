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
	writeFile(t, filepath.Join(tmp, "internal", "parser", "parser_test.go"), `package parser

import "testing"

func TestParserPackage(t *testing.T) {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main_test.go"), `package main

import (
	"testing"

	"example.com/app/internal/parser"
)

func TestUsesParser(t *testing.T) {
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

	tests := relatedTestNames(result.RelatedTests)
	assertContainsString(t, tests, "TestParserPackage")
	assertContainsString(t, tests, "TestUsesParser")

	wantCommands := []string{"go test ./cmd/app", "go test ./internal/parser"}
	if !reflect.DeepEqual(result.TestCommands, wantCommands) {
		t.Fatalf("expected %v, got %v", wantCommands, result.TestCommands)
	}
}

func TestFindImpactIncludesTransitiveCallerPackages(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "core", "core.go"), `package core

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "core", "core_test.go"), `package core

import "testing"

func TestCore(t *testing.T) {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "worker", "worker.go"), `package worker

import "example.com/app/internal/core"

func Mid() {
	core.Target()
}
`)
	writeFile(t, filepath.Join(tmp, "internal", "worker", "worker_test.go"), `package worker

import "testing"

func TestWorker(t *testing.T) {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/worker"

func Entry() {
	worker.Mid()
}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main_test.go"), `package main

import "testing"

func TestApp(t *testing.T) {}
`)

	result, err := FindImpact(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	callers := callTestCallerNames(result.Callers)
	assertContainsString(t, callers, "Mid")
	assertContainsString(t, callers, "Entry")

	assertContainsString(t, result.Packages, "./internal/core")
	assertContainsString(t, result.Packages, "./internal/worker")
	assertContainsString(t, result.Packages, "./cmd/app")

	tests := relatedTestNames(result.RelatedTests)
	assertContainsString(t, tests, "TestCore")
	assertContainsString(t, tests, "TestWorker")
	assertContainsString(t, tests, "TestApp")

	wantCommands := []string{"go test ./cmd/app", "go test ./internal/core", "go test ./internal/worker"}
	if !reflect.DeepEqual(result.TestCommands, wantCommands) {
		t.Fatalf("expected %v, got %v", wantCommands, result.TestCommands)
	}
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

func TestFindImpactIncludesLiteralSubtestsInSuggestedTests(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main_test.go"), `package main

import (
	"testing"

	"example.com/app/internal/service"
)

func TestTargetCases(t *testing.T) {
	t.Run("uses target", func(t *testing.T) {
		service.Target()
	})
}
`)

	result, err := FindImpact(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	tests := relatedTestNames(result.RelatedTests)
	assertContainsString(t, tests, "TestTargetCases")
	assertContainsString(t, tests, "TestTargetCases/uses target")
}

func TestFindImpactHonorsPackageQualifiedSymbolTargets(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeFile(t, filepath.Join(tmp, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestAuthSession(t *testing.T) {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "billing", "session.go"), `package billing

type Session struct{}
`)
	writeFile(t, filepath.Join(tmp, "internal", "billing", "session_test.go"), `package billing

import "testing"

func TestBillingSession(t *testing.T) {}
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

	result, err := FindImpact(tmp, "./internal/auth.Session")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "./internal/auth.Session" {
		t.Fatalf("expected ./internal/auth.Session target, got %s", result.Target)
	}

	if result.Kind != ImpactKindSymbol {
		t.Fatalf("expected symbol impact, got %s", result.Kind)
	}

	assertContainsString(t, result.Packages, "./cmd/app")
	assertContainsString(t, result.Packages, "./internal/auth")
	if containsString(result.Packages, "./internal/billing") {
		t.Fatalf("expected auth packages only, got %v", result.Packages)
	}

	tests := relatedTestNames(result.RelatedTests)
	assertContainsString(t, tests, "TestAuthSession")
	if containsString(tests, "TestBillingSession") {
		t.Fatalf("expected auth tests only, got %v", tests)
	}
}

func TestFindImpactForPackage(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service.go"), "package auth\n")
	writeFile(t, filepath.Join(tmp, "cmd", "api", "main.go"), `package main

import "example.com/app/internal/auth"
`)
	writeFile(t, filepath.Join(tmp, "internal", "auth", "service_test.go"), `package auth

import "testing"

func TestAuth(t *testing.T) {}
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

	tests := relatedTestNames(result.RelatedTests)
	assertContainsString(t, tests, "TestAuth")

	wantCommands := []string{"go test ./internal/auth"}
	if !reflect.DeepEqual(result.TestCommands, wantCommands) {
		t.Fatalf("expected %v, got %v", wantCommands, result.TestCommands)
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
		{target: "./internal/auth.Session", want: false},
		{target: "example.com/app/internal/auth.Session", want: false},
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
