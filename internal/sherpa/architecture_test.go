package sherpa

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeArchitectureFindsPackageSignals(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package app

import (
	"fmt"

	"example.com/app/internal/store"
)

type Service struct{}

func Run() {
	fmt.Println(store.Name())
}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package app

import "testing"

func TestRun(t *testing.T) {
	Run()
}
`)
	writeFile(t, filepath.Join(tmp, "internal", "store", "store.go"), `package store

func Name() string {
	return "store"
}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import (
	"example.com/app"
	"example.com/app/internal/store"
)

func main() {
	app.Run()
	_ = store.Name()
}
`)

	report, err := AnalyzeArchitecture(tmp, ArchitectureOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if report.PackageCount != 3 {
		t.Fatalf("expected 3 packages, got %d", report.PackageCount)
	}
	if len(report.Cycles) != 0 {
		t.Fatalf("expected no cycles, got %#v", report.Cycles)
	}

	storeFanIn := findArchitectureSignal(t, report.HighFanIn, "./internal/store")
	if storeFanIn.ImportedBy != 2 {
		t.Fatalf("expected store fan-in 2, got %#v", storeFanIn)
	}

	cmdFanOut := findArchitectureSignal(t, report.HighFanOut, "./cmd/app")
	if cmdFanOut.LocalImports != 2 {
		t.Fatalf("expected cmd fan-out 2, got %#v", cmdFanOut)
	}

	rootLargest := findArchitectureSignal(t, report.LargestPackages, ".")
	if rootLargest.Symbols != 2 {
		t.Fatalf("expected root symbols 2 without tests, got %#v", rootLargest)
	}

	leaf := findArchitectureSignal(t, report.LeafPackages, "./internal/store")
	if leaf.LocalImports != 0 {
		t.Fatalf("expected store to be a leaf package, got %#v", leaf)
	}

	for _, limitation := range report.Limitations {
		if strings.Contains(limitation, "excluded unless --tests") {
			return
		}
	}
	t.Fatalf("expected test exclusion limitation, got %#v", report.Limitations)
}

func TestAnalyzeArchitectureWithTestsIncludesTestSignals(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package app

func Run() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package app

import "testing"

func TestRun(t *testing.T) {
	Run()
}
`)

	report, err := AnalyzeArchitecture(tmp, ArchitectureOptions{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}

	rootLargest := findArchitectureSignal(t, report.LargestPackages, ".")
	if rootLargest.Symbols != 2 {
		t.Fatalf("expected test-inclusive symbol count 2, got %#v", rootLargest)
	}
	if rootLargest.ExternalImports != 1 {
		t.Fatalf("expected test import to be included, got %#v", rootLargest)
	}
}

func TestAnalyzeArchitectureReportsDependencyCycles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "one", "one.go"), `package one

import "example.com/app/internal/two"

func One() {
	two.Two()
}
`)
	writeFile(t, filepath.Join(tmp, "internal", "two", "two.go"), `package two

import "example.com/app/internal/one"

func Two() {
	one.One()
}
`)

	report, err := AnalyzeArchitecture(tmp, ArchitectureOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Cycles) != 1 {
		t.Fatalf("expected one cycle, got %#v", report.Cycles)
	}

	cycle := report.Cycles[0]
	if cycle.Size != 2 {
		t.Fatalf("expected cycle size 2, got %#v", cycle)
	}
	assertContainsString(t, cycle.Packages, "./internal/one")
	assertContainsString(t, cycle.Packages, "./internal/two")
}

func TestFormatArchitectureReport(t *testing.T) {
	report := ArchitectureReport{
		AnalysisMode: ArchitectureAnalysisModeAST,
		Confidence:   ArchitectureConfidence,
		Limitations:  []string{"Structural signals only."},
		PackageCount: 2,
		Cycles: []DependencyCycle{
			{Packages: []string{"./a", "./b"}, Size: 2},
		},
		MostCoupled: []PackageArchitectureSignal{
			{
				Package:      "./a",
				Reason:       "fan-in 1 + local fan-out 1",
				Score:        2,
				ImportedBy:   1,
				LocalImports: 1,
				Symbols:      3,
				GoFiles:      1,
			},
		},
	}

	got := FormatArchitectureReport(report)
	for _, want := range []string{
		"ARCHITECTURE",
		"Analysis: ast",
		"Packages: 2",
		"DEPENDENCY CYCLES",
		"./a, ./b",
		"MOST COUPLED PACKAGES",
		"score=2",
		"LIMITATIONS",
		"Structural signals only.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func findArchitectureSignal(t *testing.T, signals []PackageArchitectureSignal, packagePath string) PackageArchitectureSignal {
	t.Helper()

	for _, signal := range signals {
		if signal.Package == packagePath {
			return signal
		}
	}

	t.Fatalf("expected package %s in %#v", packagePath, signals)
	return PackageArchitectureSignal{}
}
