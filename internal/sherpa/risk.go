package sherpa

import (
	"fmt"
	"go/ast"
	"sort"
	"strings"
)

const (
	RiskAnalysisModeAST = "ast"
	RiskConfidence      = "medium"

	RiskLevelLow    = "low"
	RiskLevelMedium = "medium"
	RiskLevelHigh   = "high"

	riskMaxPackageSignals = 10
)

type RiskOptions struct {
	IncludeTests bool
}

type RiskReport struct {
	AnalysisMode    string              `json:"analysisMode"`
	Confidence      string              `json:"confidence"`
	Level           string              `json:"level"`
	Score           int                 `json:"score"`
	Limitations     []string            `json:"limitations"`
	PackageCount    int                 `json:"packageCount"`
	SymbolCount     int                 `json:"symbolCount"`
	ExportedSymbols int                 `json:"exportedSymbols"`
	Interfaces      int                 `json:"interfaces"`
	TestPackages    int                 `json:"testPackages"`
	Factors         []RiskFactor        `json:"factors"`
	Packages        []PackageRiskSignal `json:"packages"`
	Cycles          []DependencyCycle   `json:"cycles"`
}

type RiskFactor struct {
	Level       string `json:"level"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Score       int    `json:"score"`
}

type PackageRiskSignal struct {
	Package         string   `json:"package"`
	Level           string   `json:"level"`
	Score           int      `json:"score"`
	Reasons         []string `json:"reasons"`
	ImportedBy      int      `json:"importedBy"`
	LocalImports    int      `json:"localImports"`
	ExternalImports int      `json:"externalImports"`
	Symbols         int      `json:"symbols"`
	ExportedSymbols int      `json:"exportedSymbols"`
	Interfaces      int      `json:"interfaces"`
	GoFiles         int      `json:"goFiles"`
	TestFiles       int      `json:"testFiles"`
	HasTests        bool     `json:"hasTests"`
	InCycle         bool     `json:"inCycle"`
}

type packageSymbolRiskStats struct {
	exportedSymbols int
	interfaces      int
}

func AnalyzeRisk(root string, options RiskOptions) (RiskReport, error) {
	packages, err := FindPackageSummaries(root, PackageInventoryOptions{
		IncludeTests: options.IncludeTests,
	})
	if err != nil {
		return RiskReport{}, err
	}

	architecture, err := AnalyzeArchitecture(root, ArchitectureOptions{
		IncludeTests: options.IncludeTests,
	})
	if err != nil {
		return RiskReport{}, err
	}

	symbols, err := ParseRepository(root)
	if err != nil {
		return RiskReport{}, err
	}
	symbols = riskSelectedSymbols(symbols, options.IncludeTests)
	statsByPackage := packageSymbolRiskStatsByPackage(symbols)
	cyclePackages := packagesInDependencyCycles(architecture.Cycles)

	report := RiskReport{
		AnalysisMode:    RiskAnalysisModeAST,
		Confidence:      RiskConfidence,
		Limitations:     riskLimitations(options.IncludeTests),
		PackageCount:    len(packages),
		SymbolCount:     len(symbols),
		ExportedSymbols: countExportedRiskSymbols(symbols),
		Interfaces:      countRiskInterfaces(symbols),
		TestPackages:    countRiskTestPackages(packages),
		Cycles:          architecture.Cycles,
	}
	report.Packages = packageRiskSignals(packages, statsByPackage, cyclePackages)
	report.Factors = repositoryRiskFactors(report, packages)
	report.Score = riskFactorScore(report.Factors)
	report.Level = riskLevel(report.Score)

	return normalizeRiskReport(report), nil
}

func riskSelectedSymbols(symbols []Symbol, includeTests bool) []Symbol {
	if includeTests {
		return symbols
	}

	var selected []Symbol
	for _, symbol := range symbols {
		if strings.HasSuffix(symbol.Position.File, "_test.go") {
			continue
		}

		selected = append(selected, symbol)
	}

	return selected
}

func packageSymbolRiskStatsByPackage(symbols []Symbol) map[string]packageSymbolRiskStats {
	statsByPackage := map[string]packageSymbolRiskStats{}
	for _, symbol := range symbols {
		stats := statsByPackage[symbol.Package]
		if isRiskExportedSymbol(symbol) {
			stats.exportedSymbols++
		}
		if symbol.Kind == SymbolKindInterface {
			stats.interfaces++
		}
		statsByPackage[symbol.Package] = stats
	}

	return statsByPackage
}

func packageRiskSignals(packages []PackageSummary, statsByPackage map[string]packageSymbolRiskStats, cyclePackages map[string]bool) []PackageRiskSignal {
	var signals []PackageRiskSignal
	for _, pkg := range packages {
		stats := statsByPackage[pkg.Package]
		score, reasons := packageRiskScore(pkg, stats, cyclePackages[pkg.Package])
		if score == 0 {
			continue
		}

		signals = append(signals, PackageRiskSignal{
			Package:         pkg.Package,
			Level:           riskLevel(score),
			Score:           score,
			Reasons:         riskUniqueStringsInOrder(reasons),
			ImportedBy:      pkg.ImportedBy,
			LocalImports:    pkg.LocalImports,
			ExternalImports: pkg.ExternalImports,
			Symbols:         pkg.Symbols,
			ExportedSymbols: stats.exportedSymbols,
			Interfaces:      stats.interfaces,
			GoFiles:         pkg.GoFiles,
			TestFiles:       pkg.TestFiles,
			HasTests:        pkg.HasTests,
			InCycle:         cyclePackages[pkg.Package],
		})
	}

	sortPackageRiskSignals(signals)
	if len(signals) > riskMaxPackageSignals {
		signals = signals[:riskMaxPackageSignals]
	}

	return signals
}

func packageRiskScore(pkg PackageSummary, stats packageSymbolRiskStats, inCycle bool) (int, []string) {
	score := 0
	var reasons []string

	if inCycle {
		score += 4
		reasons = append(reasons, "Participates in a local dependency cycle.")
	}
	if pkg.ImportedBy >= 5 {
		score += 3
		reasons = append(reasons, fmt.Sprintf("Imported by %d local packages.", pkg.ImportedBy))
	} else if pkg.ImportedBy >= 2 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Imported by %d local packages.", pkg.ImportedBy))
	} else if pkg.ImportedBy == 1 {
		score++
		reasons = append(reasons, "Imported by 1 local package.")
	}
	if pkg.LocalImports >= 5 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Imports %d local packages.", pkg.LocalImports))
	} else if pkg.LocalImports > 0 {
		score++
		reasons = append(reasons, fmt.Sprintf("Imports %d local package(s).", pkg.LocalImports))
	}
	if pkg.Symbols >= 20 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Declares %d supported symbols.", pkg.Symbols))
	} else if pkg.Symbols >= 8 {
		score++
		reasons = append(reasons, fmt.Sprintf("Declares %d supported symbols.", pkg.Symbols))
	}
	if stats.exportedSymbols >= 10 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Exposes %d exported symbols.", stats.exportedSymbols))
	} else if stats.exportedSymbols > 0 {
		score++
		reasons = append(reasons, fmt.Sprintf("Exposes %d exported symbol(s).", stats.exportedSymbols))
	}
	if stats.interfaces >= 3 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Defines %d interfaces.", stats.interfaces))
	} else if stats.interfaces > 0 {
		score++
		reasons = append(reasons, fmt.Sprintf("Defines %d interface(s).", stats.interfaces))
	}

	return score, reasons
}

func repositoryRiskFactors(report RiskReport, packages []PackageSummary) []RiskFactor {
	var factors []RiskFactor

	if len(report.Cycles) > 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelHigh,
			Category:    "dependency_cycles",
			Description: fmt.Sprintf("%d local dependency cycle(s) detected.", len(report.Cycles)),
			Score:       4,
		})
	}

	topFanIn := maxPackageImportedBy(packages)
	if topFanIn >= 5 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelHigh,
			Category:    "fan_in",
			Description: fmt.Sprintf("A package is imported by %d local packages.", topFanIn),
			Score:       3,
		})
	} else if topFanIn > 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "fan_in",
			Description: fmt.Sprintf("A package is imported by %d local package(s).", topFanIn),
			Score:       1,
		})
	}

	topFanOut := maxPackageLocalImports(packages)
	if topFanOut >= 5 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "fan_out",
			Description: fmt.Sprintf("A package imports %d local packages.", topFanOut),
			Score:       2,
		})
	} else if topFanOut > 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "fan_out",
			Description: fmt.Sprintf("A package imports %d local package(s).", topFanOut),
			Score:       1,
		})
	}

	if report.ExportedSymbols >= 10 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "public_api",
			Description: fmt.Sprintf("Repository exposes %d exported symbols.", report.ExportedSymbols),
			Score:       3,
		})
	} else if report.ExportedSymbols > 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "public_api",
			Description: fmt.Sprintf("Repository exposes %d exported symbol(s).", report.ExportedSymbols),
			Score:       1,
		})
	}

	if report.Interfaces >= 3 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "interfaces",
			Description: fmt.Sprintf("Repository defines %d interfaces.", report.Interfaces),
			Score:       2,
		})
	} else if report.Interfaces > 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "interfaces",
			Description: fmt.Sprintf("Repository defines %d interface(s).", report.Interfaces),
			Score:       1,
		})
	}

	if report.TestPackages == 0 && report.PackageCount > 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelMedium,
			Category:    "tests",
			Description: "No package tests were found.",
			Score:       1,
		})
	} else if report.TestPackages > 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelLow,
			Category:    "tests",
			Description: fmt.Sprintf("Tests found in %d package(s).", report.TestPackages),
			Score:       0,
		})
	}

	if len(factors) == 0 {
		factors = append(factors, RiskFactor{
			Level:       RiskLevelLow,
			Category:    "baseline",
			Description: "No broad structural risk signals found.",
			Score:       0,
		})
	}

	return factors
}

func riskFactorScore(factors []RiskFactor) int {
	score := 0
	for _, factor := range factors {
		score += factor.Score
	}

	return score
}

func riskLevel(score int) string {
	if score >= 7 {
		return RiskLevelHigh
	}
	if score >= 3 {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

func riskLimitations(includeTests bool) []string {
	limitations := []string{
		"Risk is a deterministic structural summary, not a prediction of defects.",
		"Runtime behavior, reflection, generated behavior, and framework wiring are not inferred.",
	}
	if includeTests {
		return append(limitations, "Test-file imports and test symbols are included because --tests was set.")
	}

	return append(limitations, "Test-file imports and test-only symbols are excluded unless --tests is set.")
}

func normalizeRiskReport(report RiskReport) RiskReport {
	if report.AnalysisMode == "" {
		report.AnalysisMode = RiskAnalysisModeAST
	}
	if report.Confidence == "" {
		report.Confidence = RiskConfidence
	}
	if report.Level == "" {
		report.Level = riskLevel(report.Score)
	}
	report.Limitations = nonNilStrings(report.Limitations)
	report.Factors = nonNilRiskFactors(report.Factors)
	report.Packages = nonNilPackageRiskSignals(report.Packages)
	report.Cycles = nonNilCycles(report.Cycles)

	return report
}

func countExportedRiskSymbols(symbols []Symbol) int {
	count := 0
	for _, symbol := range symbols {
		if isRiskExportedSymbol(symbol) {
			count++
		}
	}

	return count
}

func countRiskInterfaces(symbols []Symbol) int {
	count := 0
	for _, symbol := range symbols {
		if symbol.Kind == SymbolKindInterface {
			count++
		}
	}

	return count
}

func countRiskTestPackages(packages []PackageSummary) int {
	count := 0
	for _, pkg := range packages {
		if pkg.HasTests {
			count++
		}
	}

	return count
}

func isRiskExportedSymbol(symbol Symbol) bool {
	return ast.IsExported(symbol.Name)
}

func packagesInDependencyCycles(cycles []DependencyCycle) map[string]bool {
	packages := map[string]bool{}
	for _, cycle := range cycles {
		for _, packagePath := range cycle.Packages {
			packages[packagePath] = true
		}
	}

	return packages
}

func maxPackageImportedBy(packages []PackageSummary) int {
	maximum := 0
	for _, pkg := range packages {
		if pkg.ImportedBy > maximum {
			maximum = pkg.ImportedBy
		}
	}

	return maximum
}

func maxPackageLocalImports(packages []PackageSummary) int {
	maximum := 0
	for _, pkg := range packages {
		if pkg.LocalImports > maximum {
			maximum = pkg.LocalImports
		}
	}

	return maximum
}

func sortPackageRiskSignals(signals []PackageRiskSignal) {
	sort.SliceStable(signals, func(i int, j int) bool {
		if signals[i].Score != signals[j].Score {
			return signals[i].Score > signals[j].Score
		}
		if signals[i].ImportedBy != signals[j].ImportedBy {
			return signals[i].ImportedBy > signals[j].ImportedBy
		}
		if signals[i].LocalImports != signals[j].LocalImports {
			return signals[i].LocalImports > signals[j].LocalImports
		}
		if signals[i].ExportedSymbols != signals[j].ExportedSymbols {
			return signals[i].ExportedSymbols > signals[j].ExportedSymbols
		}
		if signals[i].Interfaces != signals[j].Interfaces {
			return signals[i].Interfaces > signals[j].Interfaces
		}

		return signals[i].Package < signals[j].Package
	})
}

func riskUniqueStringsInOrder(values []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func nonNilRiskFactors(values []RiskFactor) []RiskFactor {
	if values == nil {
		return []RiskFactor{}
	}

	return values
}

func nonNilPackageRiskSignals(values []PackageRiskSignal) []PackageRiskSignal {
	if values == nil {
		return []PackageRiskSignal{}
	}
	for i, signal := range values {
		signal.Reasons = nonNilStrings(signal.Reasons)
		values[i] = signal
	}

	return values
}
