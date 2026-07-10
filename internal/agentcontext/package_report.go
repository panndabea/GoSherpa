package agentcontext

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type PackageAnalyzeOptions struct {
	IncludeTests bool `json:"includeTests"`
	BuildTags    []string
	SourceRadius int          `json:"sourceRadius"`
	Limits       LimitOptions `json:"limits"`
}

type PackageReport struct {
	Target                  string                      `json:"target"`
	Package                 string                      `json:"package"`
	PackageName             string                      `json:"packageName,omitempty"`
	Files                   []string                    `json:"files"`
	Symbols                 []sherpa.Symbol             `json:"symbols"`
	SourceContexts          []sherpa.SourceContext      `json:"sourceContexts"`
	Purpose                 string                      `json:"purpose"`
	Risk                    explainengine.RiskSummary   `json:"risk"`
	AffectedPackages        []string                    `json:"affectedPackages"`
	AffectedInterfaces      []string                    `json:"affectedInterfaces"`
	AffectedImplementations []string                    `json:"affectedImplementations"`
	InterfaceAnalysisMode   string                      `json:"interfaceAnalysisMode,omitempty"`
	AffectedTests           []impactengine.RelatedTest  `json:"affectedTests"`
	TestAnalysisMode        string                      `json:"testAnalysisMode,omitempty"`
	TestCommands            []string                    `json:"testCommands"`
	TestPlan                sherpa.TestPlan             `json:"testPlan"`
	ReadingOrder            []explainengine.ReadingStep `json:"readingOrder"`
	AnalysisMode            string                      `json:"analysisMode"`
	Confidence              string                      `json:"confidence"`
	Limits                  *LimitOptions               `json:"limits,omitempty"`
	Truncated               *Truncation                 `json:"truncated,omitempty"`
	Limitations             []string                    `json:"limitations"`
	Warnings                []string                    `json:"-"`
}

func AnalyzePackage(root string, targetPackage string, options PackageAnalyzeOptions) (PackageReport, error) {
	impactReport, err := impactengine.AnalyzePackageWithOptions(root, targetPackage, impactengine.AnalyzerOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return PackageReport{}, err
	}

	packagePath := firstString(impactReport.ChangedPackages)
	semanticSnapshot, semanticOK := loadContextSemanticSnapshot(root, options.BuildTags)
	warnings := append([]string{}, impactReport.Warnings...)
	warnings = append(warnings, semanticSnapshot.warnings...)

	analysisMode := AnalysisModeAST
	var files []string
	if semanticOK && semanticSnapshot.hasPackage(packagePath) {
		analysisMode = AnalysisModeTypecheckedAST
		var fileWarnings []string
		files, fileWarnings = semanticSnapshot.packageFiles(root, packagePath)
		warnings = append(warnings, fileWarnings...)
	} else {
		if semanticOK {
			warnings = append(warnings, fmt.Sprintf("typechecked context package not loaded: %s", packagePath))
		}
		var err error
		files, err = packageFiles(root, packagePath)
		if err != nil {
			return PackageReport{}, err
		}
	}

	packageName, err := packageNameForFiles(root, files)
	if err != nil {
		return PackageReport{}, err
	}

	var symbols []sherpa.Symbol
	if analysisMode == AnalysisModeTypecheckedAST {
		var symbolWarnings []string
		symbols, symbolWarnings = semanticSnapshot.symbolsInPackage(root, packagePath)
		warnings = append(warnings, symbolWarnings...)
	} else {
		allSymbols, err := sherpa.ParseRepository(root)
		if err != nil {
			return PackageReport{}, err
		}
		symbols = symbolsInPackage(allSymbols, packagePath)
	}

	limits := normalizePackageLimits(options.SourceRadius, options.Limits)
	radius := sourceRadiusOrDefault(limits, sherpa.DefaultSourceContextRadius)

	sourceContexts, err := sourceContextsForSymbols(root, symbols, radius)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	report := PackageReport{
		Target:                  packagePath,
		Package:                 packagePath,
		PackageName:             packageName,
		Files:                   files,
		Symbols:                 symbols,
		SourceContexts:          sourceContexts,
		AffectedPackages:        impactReport.AffectedPackages,
		AffectedInterfaces:      impactReport.AffectedInterfaces,
		AffectedImplementations: impactReport.AffectedImplementations,
		InterfaceAnalysisMode:   impactReport.InterfaceAnalysisMode,
		AffectedTests:           impactReport.AffectedTests,
		TestAnalysisMode:        impactReport.TestAnalysisMode,
		TestCommands:            impactReport.TestCommands,
		TestPlan:                impactReport.TestPlan,
		AnalysisMode:            analysisMode,
		Limits:                  reportLimits(limits),
		Warnings:                warnings,
	}
	report.Purpose = packagePurpose(report)
	report.Risk = packageRiskSummary(report)
	report.ReadingOrder = packageReadingOrder(report)
	report.Limitations = packageLimitations(options.IncludeTests, report.AnalysisMode)
	report.Confidence = packageConfidence(report)
	report = applyPackageLimits(report, limits)

	return normalizePackageReport(report), nil
}

func applyPackageLimits(report PackageReport, limits LimitOptions) PackageReport {
	var truncation Truncation
	originalReadingOrderCount := len(report.ReadingOrder)

	report.AffectedTests = prioritizeContextTests(report.AffectedTests)
	report.Files, truncation.Files = limitSlice(report.Files, limits.MaxFiles)
	report.Symbols, truncation.Symbols = limitSlice(report.Symbols, limits.MaxSymbols)
	report.SourceContexts, truncation.SourceContexts = limitSlice(report.SourceContexts, limits.MaxSymbols)
	report.AffectedTests, truncation.AffectedTests = limitSlice(report.AffectedTests, limits.MaxTests)
	report.ReadingOrder = packageReadingOrder(report)
	if originalReadingOrderCount > len(report.ReadingOrder) {
		truncation.ReadingOrder = originalReadingOrderCount - len(report.ReadingOrder)
	}

	report.Truncated = reportTruncation(truncation)
	report = applyPackageByteLimit(report, limits.MaxBytes)

	return report
}

func packageFiles(root string, packagePath string) ([]string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root %s: %w", root, err)
	}

	dir := packageDirectory(packagePath)
	dirPath := rootPath
	if dir != "." {
		dirPath = filepath.Join(rootPath, filepath.FromSlash(dir))
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read package directory %s: %w", packagePath, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}

		if dir == "." {
			files = append(files, entry.Name())
			continue
		}

		files = append(files, path.Join(dir, entry.Name()))
	}

	sort.Strings(files)
	return files, nil
}

func packageDirectory(packagePath string) string {
	if strings.TrimSpace(packagePath) == "" || packagePath == "." {
		return "."
	}

	return strings.TrimPrefix(filepath.ToSlash(packagePath), "./")
}

func packageNameForFiles(root string, files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	firstName := ""
	for _, file := range files {
		name, err := filePackageName(root, file)
		if err != nil {
			return "", err
		}
		if firstName == "" {
			firstName = name
		}
		if !strings.HasSuffix(name, "_test") {
			return name, nil
		}
	}

	return firstName, nil
}

func symbolsInPackage(symbols []sherpa.Symbol, packagePath string) []sherpa.Symbol {
	var result []sherpa.Symbol
	for _, symbol := range symbols {
		if symbol.Package == packagePath {
			result = append(result, symbol)
		}
	}

	sort.SliceStable(result, func(i int, j int) bool {
		if result[i].Position.File != result[j].Position.File {
			return result[i].Position.File < result[j].Position.File
		}
		if result[i].Position.Line != result[j].Position.Line {
			return result[i].Position.Line < result[j].Position.Line
		}
		if result[i].Position.Column != result[j].Position.Column {
			return result[i].Position.Column < result[j].Position.Column
		}

		return result[i].DisplayName() < result[j].DisplayName()
	})

	return result
}

func packagePurpose(report PackageReport) string {
	if report.Package == "" {
		return "Package target could not be mapped to a repository-local Go package."
	}
	if len(report.Files) == 0 {
		return fmt.Sprintf("Package %s has no repository-local Go files.", report.Package)
	}
	if len(report.Symbols) == 0 {
		return fmt.Sprintf(
			"Package %s contains %s; no supported top-level symbols were found.",
			report.Package,
			countNoun(len(report.Files), "Go file"),
		)
	}

	return fmt.Sprintf(
		"Package %s contains %s declaring %s; impact analysis reaches %s and %s.",
		report.Package,
		countNoun(len(report.Files), "Go file"),
		countNoun(len(report.Symbols), "supported symbol"),
		countNoun(len(report.AffectedPackages), "package"),
		countNoun(len(report.AffectedTests), "affected test"),
	)
}

func packageRiskSummary(report PackageReport) explainengine.RiskSummary {
	score := 0
	var reasons []string

	if len(report.Files) == 0 {
		reasons = append(reasons, "No Go files found in the package.")
	} else {
		reasons = append(reasons, fmt.Sprintf("Package contains %d Go files.", len(report.Files)))
		if len(report.Files) > 5 {
			score += 2
		} else if len(report.Files) > 1 {
			score++
		}
	}

	if len(report.Symbols) == 0 {
		reasons = append(reasons, "No supported top-level symbols found in the package.")
	} else {
		reasons = append(reasons, fmt.Sprintf("Package declares %d supported symbols.", len(report.Symbols)))
		if len(report.Symbols) > 10 {
			score += 2
		} else if len(report.Symbols) > 4 {
			score++
		}
	}

	if len(report.AffectedPackages) > 1 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Impact reaches %d packages.", len(report.AffectedPackages)))
	} else if len(report.AffectedPackages) == 1 {
		score++
		reasons = append(reasons, "Impact stays within 1 package.")
	}

	interfaceSignals := len(report.AffectedInterfaces) + len(report.AffectedImplementations)
	if interfaceSignals > 0 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Touches %d interface or implementation signals.", interfaceSignals))
	}

	if len(report.AffectedTests) == 0 && report.Package != "" {
		score++
		reasons = append(reasons, "No affected tests found.")
	} else if len(report.AffectedTests) > 0 {
		reasons = append(reasons, fmt.Sprintf("Affected tests found: %d.", len(report.AffectedTests)))
	}

	level := "low"
	if score >= 5 {
		level = "high"
	} else if score >= 2 {
		level = "medium"
	}

	return explainengine.RiskSummary{
		Level:   level,
		Reasons: uniqueStrings(reasons),
	}
}

func packageReadingOrder(report PackageReport) []explainengine.ReadingStep {
	steps := make([]explainengine.ReadingStep, 0, len(report.Files)+len(report.Symbols)+len(report.AffectedTests))

	for _, file := range report.Files {
		steps = append(steps, explainengine.ReadingStep{
			Title:  "File: " + file,
			Reason: "Start with the files in the target package.",
			Position: sherpa.Position{
				File: file,
				Line: 1,
			},
		})
	}

	for _, symbol := range report.Symbols {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Symbol: " + symbol.DisplayName(),
			Reason:   "Inspect the symbols declared in the target package.",
			Position: symbol.Position,
			Range:    symbol.Range,
		})
	}

	for _, test := range report.AffectedTests {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Test: " + test.Name,
			Reason:   "Check expected behavior and regression coverage.",
			Position: test.Position,
			Range:    test.Range,
		})
	}

	return steps
}

func packageLimitations(includeTests bool, analysisMode string) []string {
	values := []string{
		"Package context uses package-level impact for affected packages and tests.",
		"Source excerpts are limited to supported top-level Go symbols: functions, methods, structs, interfaces, and type aliases.",
		packageContextAnalysisLimitation(analysisMode),
		"Dynamic dispatch, reflection, and function values are not resolved.",
		"Test discovery uses direct references, same-package tests, and literal t.Run subtest names.",
	}

	if includeTests {
		values = append(values, "--tests is accepted for workflow symmetry; package context always includes affected tests from impact analysis.")
	}

	return values
}

func packageContextAnalysisLimitation(analysisMode string) string {
	switch analysisMode {
	case AnalysisModeTypecheckedAST:
		return "Package context used typechecked package loading for package files and symbols, with syntax/local impact signals."
	default:
		return "Package context used AST fallback for package files and symbols because typechecked loading was unavailable."
	}
}

func packageConfidence(report PackageReport) string {
	if len(report.Warnings) > 0 || report.Package == "" {
		return ConfidenceLow
	}
	if report.InterfaceAnalysisMode == impactengine.InterfaceAnalysisModeASTFallback {
		return ConfidenceLow
	}

	return ConfidenceMedium
}

func normalizePackageReport(report PackageReport) PackageReport {
	report.Files = nonNilSlice(report.Files)
	report.Symbols = nonNilSlice(report.Symbols)
	report.SourceContexts = nonNilSlice(report.SourceContexts)
	for i := range report.SourceContexts {
		report.SourceContexts[i].Lines = nonNilSlice(report.SourceContexts[i].Lines)
	}
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestAnalysisMode = strings.TrimSpace(report.TestAnalysisMode)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}
