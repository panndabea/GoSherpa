package sherpa

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/panndabea/GoSherpa/internal/semantics"
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
	if len(result.TestPlan.Direct) != 1 || result.TestPlan.Direct[0].Package != "./cmd/app" {
		t.Fatalf("expected direct test plan item for ./cmd/app, got %#v", result.TestPlan)
	}
	if len(result.TestPlan.Related) != 1 || result.TestPlan.Related[0].Package != "./internal/parser" {
		t.Fatalf("expected related test plan item for ./internal/parser, got %#v", result.TestPlan)
	}
}

func TestFindTestsWithRelatedScopeFocusesDirectReferences(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "parser", "parser_test.go"), `package parser

import "testing"

func TestParserPackage(t *testing.T) {}
func TestParserOther(t *testing.T) {}
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

	result, err := FindTestsWithOptions(tmp, "ParseFile", TestOptions{Scope: TestScopeRelated})
	if err != nil {
		t.Fatal(err)
	}

	names := relatedTestNames(result.Tests)
	assertContainsString(t, names, "TestUsesParser")
	if containsString(names, "TestParserPackage") || containsString(names, "TestParserOther") {
		t.Fatalf("expected focused direct tests, got %v", names)
	}

	if result.Scope != TestScopeRelated {
		t.Fatalf("expected related scope, got %s", result.Scope)
	}
	if len(result.TestPlan.Direct) != 1 || result.TestPlan.Direct[0].Package != "./cmd/app" {
		t.Fatalf("expected direct plan item for ./cmd/app, got %#v", result.TestPlan)
	}
	if len(result.TestPlan.Related) != 0 {
		t.Fatalf("expected no related plan items, got %#v", result.TestPlan)
	}
	if len(result.TestPlan.Fallback) != 1 || result.TestPlan.Fallback[0].Package != "./internal/parser" {
		t.Fatalf("expected fallback package test for ./internal/parser, got %#v", result.TestPlan)
	}
}

func TestFindTestsWithAllScopePreservesSamePackageRelatedTests(t *testing.T) {
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

	result, err := FindTestsWithOptions(tmp, "ParseFile", TestOptions{Scope: TestScopeAll})
	if err != nil {
		t.Fatal(err)
	}

	names := relatedTestNames(result.Tests)
	assertContainsString(t, names, "TestParserPackage")
	assertContainsString(t, names, "TestUsesParser")
	if len(result.TestPlan.Related) != 1 || result.TestPlan.Related[0].Package != "./internal/parser" {
		t.Fatalf("expected related plan item for ./internal/parser, got %#v", result.TestPlan)
	}
}

func TestFindTestsHonorsPackageQualifiedSymbolTargets(t *testing.T) {
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
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main_test.go"), `package main

import (
	"testing"

	auth "example.com/app/internal/auth"
	"example.com/app/internal/billing"
)

func TestUsesAuthSession(t *testing.T) {
	_ = auth.Session{}
}

func TestUsesBillingSession(t *testing.T) {
	_ = billing.Session{}
}
`)

	result, err := FindTests(tmp, "./internal/auth.Session")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "./internal/auth.Session" {
		t.Fatalf("expected ./internal/auth.Session target, got %s", result.Target)
	}

	names := relatedTestNames(result.Tests)
	assertContainsString(t, names, "TestAuthSession")
	assertContainsString(t, names, "TestUsesAuthSession")
	if containsString(names, "TestBillingSession") || containsString(names, "TestUsesBillingSession") {
		t.Fatalf("expected auth tests only, got %v", names)
	}

	wantCommands := []string{"go test ./cmd/app", "go test ./internal/auth"}
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

func TestFindTestsUsesTypeInfoForExternalTestReceiverMethodReferences(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Client struct{}

func (client *Client) Start() {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "service", "service_test.go"), `package service_test

import (
	"testing"

	"example.com/app/internal/service"
)

func TestStart(t *testing.T) {
	client := &service.Client{}
	client.Start()
}
`)

	result, err := FindTests(tmp, "./internal/service.Client.Start")
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
	if result.AnalysisMode != TestAnalysisModeTypecheckedAST {
		t.Fatalf("expected typechecked+ast analysis mode, got %q with warnings %#v", result.AnalysisMode, result.Warnings)
	}
	if len(result.TestPlan.Direct) != 1 || result.TestPlan.Direct[0].Package != "./internal/service" {
		t.Fatalf("expected direct test plan item for ./internal/service, got %#v", result.TestPlan)
	}
}

func TestFindTestsReportsTypecheckedReferenceWarnings(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)

	original := loadSemanticTestRepository
	loadSemanticTestRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		return semantics.Repository{}, errors.New("loader failed")
	}
	defer func() {
		loadSemanticTestRepository = original
	}()

	result, err := FindTests(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	if result.AnalysisMode != TestAnalysisModeAST {
		t.Fatalf("expected ast fallback mode, got %q", result.AnalysisMode)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "loader failed") {
		t.Fatalf("expected loader warning, got %#v", result.Warnings)
	}

	test := findRelatedTest(result.Tests, "TestTarget")
	if test == nil || !test.DirectReference {
		t.Fatalf("expected AST fallback to preserve direct test reference, got %#v", result.Tests)
	}
}

func TestFindTestsIncludesLiteralSubtestsForDirectReferences(t *testing.T) {
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
	t.Run("unrelated", func(t *testing.T) {})
	t.Run(testName(), func(t *testing.T) {
		service.Target()
	})
}

func testName() string {
	return "dynamic"
}
`)

	result, err := FindTests(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	names := relatedTestNames(result.Tests)
	assertContainsString(t, names, "TestTargetCases")
	assertContainsString(t, names, "TestTargetCases/uses target")
	if containsString(names, "TestTargetCases/unrelated") || containsString(names, "TestTargetCases/dynamic") {
		t.Fatalf("expected only directly related literal subtests, got %v", names)
	}

	subtest := findRelatedTest(result.Tests, "TestTargetCases/uses target")
	if subtest == nil || !subtest.DirectReference {
		t.Fatalf("expected direct literal subtest marker, got %v", subtest)
	}
}

func TestFindTestsIncludesNestedLiteralSubtests(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

import "testing"

func TestTargetCases(t *testing.T) {
	t.Run("group", func(t *testing.T) {
		t.Run("leaf", func(t *testing.T) {
			Target()
		})
	})
}
`)

	result, err := FindTests(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	names := relatedTestNames(result.Tests)
	assertContainsString(t, names, "TestTargetCases/group")
	assertContainsString(t, names, "TestTargetCases/group/leaf")
}

func TestFindTestsReturnsSourceRanges(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

import "testing"

func TestTarget(t *testing.T) {
	Target()
}

func TestSubtests(t *testing.T) {
	t.Run("case", func(t *testing.T) { Target() })
}
`)

	result, err := FindTests(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	topLevel := findRelatedTest(result.Tests, "TestTarget")
	if topLevel == nil {
		t.Fatalf("expected TestTarget, got %v", result.Tests)
	}
	assertSourceRange(t, topLevel.Range, "service_test.go", 5, 1, 7, 2)

	subtest := findRelatedTest(result.Tests, "TestSubtests/case")
	if subtest == nil {
		t.Fatalf("expected TestSubtests/case, got %v", result.Tests)
	}
	assertSourceRange(t, subtest.Range, "service_test.go", 10, 2, 10, 48)
}

func TestFindTestsAddsFallbackForSymbolPackageWithoutTests(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}
`)

	result, err := FindTests(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Tests) != 0 {
		t.Fatalf("expected no related tests, got %#v", result.Tests)
	}

	wantCommands := []string{"go test ."}
	if !reflect.DeepEqual(result.Commands, wantCommands) {
		t.Fatalf("expected %v, got %v", wantCommands, result.Commands)
	}

	if len(result.TestPlan.Fallback) != 1 {
		t.Fatalf("expected fallback test plan item, got %#v", result.TestPlan)
	}
	item := result.TestPlan.Fallback[0]
	if item.Command != "go test ." || item.Package != "." {
		t.Fatalf("unexpected fallback item: %#v", item)
	}
	if item.Reason == "" {
		t.Fatalf("expected fallback reason, got %#v", item)
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
