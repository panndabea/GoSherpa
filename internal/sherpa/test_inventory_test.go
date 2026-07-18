package sherpa

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildTestInventoryRecordsPackagesFunctionsSubtestsAndSuiteMethods(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), "package service\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service_test.go"), `package service

import "testing"

func TestService(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		t.Run("nested", func(t *testing.T) {})
	})
	for _, tt := range []struct{name string}{{"dynamic"}} {
		t.Run(tt.name, func(t *testing.T) {})
	}
}

type ServiceSuite struct{}

func (suite *ServiceSuite) TestWorkflow(t *testing.T) {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "service", "external_test.go"), `package service_test

import "testing"

func TestExternal(t *testing.T) {}
`)

	inventory, err := BuildTestInventory(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(inventory.Files) != 2 {
		t.Fatalf("expected two test files, got %#v", inventory.Files)
	}
	if len(inventory.Functions) != 2 {
		t.Fatalf("expected two top-level test functions, got %#v", inventory.Functions)
	}
	if len(inventory.Packages) != 2 {
		t.Fatalf("expected internal and external test package records, got %#v", inventory.Packages)
	}

	internalPackage := findTestInventoryPackage(inventory.Packages, "./internal/service", "service")
	if internalPackage == nil {
		t.Fatalf("expected internal service package record, got %#v", inventory.Packages)
	}
	if internalPackage.TestFunctions != 1 || internalPackage.Subtests != 2 || internalPackage.SuiteLikeMethods != 1 {
		t.Fatalf("unexpected internal package counts: %#v", internalPackage)
	}
	if !reflect.DeepEqual(internalPackage.Files, []string{"internal/service/service_test.go"}) {
		t.Fatalf("unexpected internal test files: %#v", internalPackage.Files)
	}

	externalPackage := findTestInventoryPackage(inventory.Packages, "./internal/service", "service_test")
	if externalPackage == nil || !externalPackage.ExternalPackage {
		t.Fatalf("expected external service_test package record, got %#v", inventory.Packages)
	}

	function := findTestInventoryFunction(inventory.Functions, "TestService")
	if function == nil {
		t.Fatalf("expected TestService function, got %#v", inventory.Functions)
	}
	if function.Range == nil || function.Position.File != "internal/service/service_test.go" {
		t.Fatalf("expected source range and root-relative position, got %#v", function)
	}
	if got := testInventorySubtestNames(function.Subtests); !reflect.DeepEqual(got, []string{"happy", "happy/nested"}) {
		t.Fatalf("expected literal nested subtests only, got %#v", got)
	}

	file := findTestInventoryFile(inventory.Files, "internal/service/service_test.go")
	if file == nil || len(file.SuiteLikeMethods) != 1 || !file.SuiteLikeMethods[0].SuiteLike {
		t.Fatalf("expected conservative suite-like method record, got %#v", file)
	}
	if !reflect.DeepEqual(inventory.Limitations, []string{TestInventoryLimitationDynamicSubtests}) {
		t.Fatalf("unexpected inventory limitations: %#v", inventory.Limitations)
	}
}

func TestFindTestsStillReturnsTargetReferenceRangesFromInventoryBackedFiles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package app

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package app

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)

	result, err := FindTests(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}

	test := findRelatedTest(result.Tests, "TestTarget")
	if test == nil {
		t.Fatalf("expected TestTarget, got %#v", result.Tests)
	}
	if test.Range == nil {
		t.Fatalf("expected test function range, got %#v", test)
	}
	if len(test.TargetReferences) != 1 || test.TargetReferences[0].Range == nil {
		t.Fatalf("expected target reference source range, got %#v", test.TargetReferences)
	}
}

func findTestInventoryPackage(records []TestPackageRecord, pkg string, packageName string) *TestPackageRecord {
	for i := range records {
		if records[i].Package == pkg && records[i].PackageName == packageName {
			return &records[i]
		}
	}
	return nil
}

func findTestInventoryFile(records []TestFileRecord, file string) *TestFileRecord {
	for i := range records {
		if records[i].File == file {
			return &records[i]
		}
	}
	return nil
}

func findTestInventoryFunction(records []TestFunctionRecord, name string) *TestFunctionRecord {
	for i := range records {
		if records[i].Name == name {
			return &records[i]
		}
	}
	return nil
}

func testInventorySubtestNames(records []TestSubtestRecord) []string {
	names := make([]string, 0, len(records))
	for _, record := range records {
		names = append(names, record.Name)
	}
	return names
}
