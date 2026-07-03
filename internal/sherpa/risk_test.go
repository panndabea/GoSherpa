package sherpa

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeRiskFindsStructuralFactors(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package app

import (
	"fmt"

	"example.com/app/internal/store"
)

type Service struct{}

type Runner interface {
	Run()
}

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

	report, err := AnalyzeRisk(tmp, RiskOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if report.Level != RiskLevelMedium {
		t.Fatalf("expected medium risk, got %#v", report)
	}
	if report.Score != 4 {
		t.Fatalf("expected score 4, got %#v", report)
	}
	if report.PackageCount != 3 {
		t.Fatalf("expected 3 packages, got %d", report.PackageCount)
	}
	if report.ExportedSymbols != 4 {
		t.Fatalf("expected 4 exported symbols, got %d", report.ExportedSymbols)
	}
	if report.Interfaces != 1 {
		t.Fatalf("expected 1 interface, got %d", report.Interfaces)
	}
	if report.TestPackages != 1 {
		t.Fatalf("expected 1 test package, got %d", report.TestPackages)
	}

	rootPackage := findPackageRiskSignal(t, report.Packages, ".")
	if rootPackage.Level != RiskLevelMedium {
		t.Fatalf("expected root package medium risk, got %#v", rootPackage)
	}
	assertContainsString(t, rootPackage.Reasons, "Defines 1 interface(s).")

	storePackage := findPackageRiskSignal(t, report.Packages, "./internal/store")
	if storePackage.ImportedBy != 2 {
		t.Fatalf("expected store imported by 2, got %#v", storePackage)
	}

	assertRiskFactor(t, report.Factors, "fan_in")
	assertRiskFactor(t, report.Factors, "fan_out")
	assertRiskFactor(t, report.Factors, "public_api")
	assertRiskFactor(t, report.Factors, "interfaces")
}

func TestAnalyzeRiskReportsCyclesAsHighRisk(t *testing.T) {
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

	report, err := AnalyzeRisk(tmp, RiskOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if report.Level != RiskLevelHigh {
		t.Fatalf("expected high risk, got %#v", report)
	}
	if len(report.Cycles) != 1 {
		t.Fatalf("expected one cycle, got %#v", report.Cycles)
	}
	assertRiskFactor(t, report.Factors, "dependency_cycles")

	onePackage := findPackageRiskSignal(t, report.Packages, "./internal/one")
	if !onePackage.InCycle || onePackage.Level != RiskLevelHigh {
		t.Fatalf("expected cycle package high risk, got %#v", onePackage)
	}
}

func TestAnalyzeRiskWithTestsIncludesTestSymbols(t *testing.T) {
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

	report, err := AnalyzeRisk(tmp, RiskOptions{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}

	if report.SymbolCount != 2 {
		t.Fatalf("expected test-inclusive symbol count 2, got %#v", report)
	}

	rootPackage := findPackageRiskSignal(t, report.Packages, ".")
	if rootPackage.ExternalImports != 1 {
		t.Fatalf("expected test import to be included, got %#v", rootPackage)
	}
}

func TestFormatRiskReport(t *testing.T) {
	report := RiskReport{
		AnalysisMode:    RiskAnalysisModeAST,
		Confidence:      RiskConfidence,
		Level:           RiskLevelMedium,
		Score:           3,
		PackageCount:    2,
		SymbolCount:     4,
		ExportedSymbols: 2,
		Interfaces:      1,
		TestPackages:    1,
		Limitations:     []string{"Structural risk only."},
		Factors: []RiskFactor{
			{Level: RiskLevelMedium, Category: "public_api", Description: "Repository exposes 2 exported symbols.", Score: 1},
		},
		Packages: []PackageRiskSignal{
			{
				Package:         ".",
				Level:           RiskLevelMedium,
				Score:           3,
				Reasons:         []string{"Exposes 2 exported symbol(s)."},
				ImportedBy:      1,
				ExportedSymbols: 2,
				Interfaces:      1,
				Symbols:         4,
			},
		},
		Cycles: []DependencyCycle{
			{Packages: []string{"./a", "./b"}, Size: 2},
		},
	}

	got := FormatRiskReport(report)
	for _, want := range []string{
		"RISK",
		"Level: medium",
		"Score: 3",
		"SUMMARY",
		"Exported symbols: 2",
		"FACTORS",
		"public_api",
		"PACKAGE SIGNALS",
		"DEPENDENCY CYCLES",
		"./a, ./b",
		"LIMITATIONS",
		"Structural risk only.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func findPackageRiskSignal(t *testing.T, signals []PackageRiskSignal, packagePath string) PackageRiskSignal {
	t.Helper()

	for _, signal := range signals {
		if signal.Package == packagePath {
			return signal
		}
	}

	t.Fatalf("expected package %s in %#v", packagePath, signals)
	return PackageRiskSignal{}
}

func assertRiskFactor(t *testing.T, factors []RiskFactor, category string) {
	t.Helper()

	for _, factor := range factors {
		if factor.Category == category {
			return
		}
	}

	t.Fatalf("expected factor %s in %#v", category, factors)
}
