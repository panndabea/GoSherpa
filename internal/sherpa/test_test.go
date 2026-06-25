package sherpa

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestFindTestsForPackageIncludesSameAndExternalPackageTests(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), "package service\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service_test.go"), `package service

import "testing"

func TestSamePackage(t *testing.T) {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "service", "external_test.go"), `package service_test

import "testing"

func TestExternalPackage(t *testing.T) {}
`)

	result, err := FindTests(tmp, "./internal/service")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "./internal/service" {
		t.Fatalf("expected ./internal/service target, got %s", result.Target)
	}

	if result.Kind != TestTargetKindPackage {
		t.Fatalf("expected package target, got %s", result.Kind)
	}

	names := relatedTestNames(result.Tests)
	assertContainsString(t, names, "TestSamePackage")
	assertContainsString(t, names, "TestExternalPackage")

	external := findRelatedTest(result.Tests, "TestExternalPackage")
	if external == nil || !external.ExternalPackage {
		t.Fatalf("expected external package marker, got %v", external)
	}

	wantCommands := []string{"go test ./internal/service"}
	if !reflect.DeepEqual(result.Commands, wantCommands) {
		t.Fatalf("expected %v, got %v", wantCommands, result.Commands)
	}
}

func TestFindTestsForSymbolIncludesSamePackageAndDirectReferences(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
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

	result, err := FindTests(tmp, "ParseFile")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "ParseFile" {
		t.Fatalf("expected ParseFile target, got %s", result.Target)
	}

	if result.Kind != TestTargetKindSymbol {
		t.Fatalf("expected symbol target, got %s", result.Kind)
	}

	names := relatedTestNames(result.Tests)
	assertContainsString(t, names, "TestParserPackage")
	assertContainsString(t, names, "TestUsesParser")

	direct := findRelatedTest(result.Tests, "TestUsesParser")
	if direct == nil || !direct.DirectReference {
		t.Fatalf("expected direct reference marker, got %v", direct)
	}

	wantCommands := []string{"go test ./cmd/app", "go test ./internal/parser"}
	if !reflect.DeepEqual(result.Commands, wantCommands) {
		t.Fatalf("expected %v, got %v", wantCommands, result.Commands)
	}
}

func TestFindTestsMarksMethodReferencesAsDirect(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func (server Server) Start() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

import "testing"

func TestStart(t *testing.T) {
	var server Server
	server.Start()
}
`)

	result, err := FindTests(tmp, "Server.Start")
	if err != nil {
		t.Fatal(err)
	}

	test := findRelatedTest(result.Tests, "TestStart")
	if test == nil {
		t.Fatalf("expected TestStart, got %v", result.Tests)
	}

	if !test.DirectReference {
		t.Fatalf("expected direct reference marker, got %v", test)
	}
}

func TestFindTestsReturnsEmptyForMissingSymbol(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

import "testing"

func TestService(t *testing.T) {}
`)

	result, err := FindTests(tmp, "Missing")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Tests) != 0 {
		t.Fatalf("expected no tests, got %v", result.Tests)
	}

	if len(result.Commands) != 0 {
		t.Fatalf("expected no commands, got %v", result.Commands)
	}
}

func TestFindTestsReturnsErrorForMissingPackage(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), "package service\n")

	_, err := FindTests(tmp, "./internal/missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

func relatedTestNames(tests []RelatedTest) []string {
	var names []string
	for _, test := range tests {
		names = append(names, test.Name)
	}

	return names
}

func findRelatedTest(tests []RelatedTest, name string) *RelatedTest {
	for i := range tests {
		if tests[i].Name == name {
			return &tests[i]
		}
	}

	return nil
}
