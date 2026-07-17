package agentcontext

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type FileAnalyzeOptions struct {
	IncludeTests bool `json:"includeTests"`
	BuildTags    []string
	SourceRadius int          `json:"sourceRadius"`
	Limits       LimitOptions `json:"limits"`
}

type FileReport struct {
	Target                  string                      `json:"target"`
	File                    string                      `json:"file"`
	Package                 string                      `json:"package"`
	PackageName             string                      `json:"packageName,omitempty"`
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

func AnalyzeFile(root string, target string, options FileAnalyzeOptions) (FileReport, error) {
	file, err := normalizeFileTarget(root, target)
	if err != nil {
		return FileReport{}, err
	}

	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return FileReport{}, err
	}

	impactReport, err := impactengine.AnalyzeFileWithContext(semanticContext, file, impactengine.AnalyzerOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return FileReport{}, err
	}

	packageName, err := filePackageName(root, file)
	if err != nil {
		return FileReport{}, err
	}

	semanticSnapshot, semanticOK := loadContextSemanticSnapshotWithContext(semanticContext)
	warnings := append([]string{}, impactReport.Warnings...)
	warnings = append(warnings, semanticSnapshot.warnings...)

	analysisMode := AnalysisModeAST
	var symbols []sherpa.Symbol
	if semanticOK && semanticSnapshot.hasPackage(firstString(impactReport.ChangedPackages)) {
		analysisMode = AnalysisModeTypecheckedAST
		var symbolWarnings []string
		symbols, symbolWarnings = semanticSnapshot.symbolsInFile(root, file, firstString(impactReport.ChangedPackages))
		warnings = append(warnings, symbolWarnings...)
	} else {
		if semanticOK {
			warnings = append(warnings, fmt.Sprintf("typechecked context package not loaded: %s", firstString(impactReport.ChangedPackages)))
		}
		allSymbols, err := sherpa.ParseRepository(root)
		if err != nil {
			return FileReport{}, err
		}
		symbols = symbolsInFile(allSymbols, file)
	}

	limits := normalizeFileLimits(options.SourceRadius, options.Limits)
	radius := sourceRadiusOrDefault(limits, sherpa.DefaultSourceContextRadius)

	sourceContexts, err := sourceContextsForSymbols(root, symbols, radius)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	report := FileReport{
		Target:                  file,
		File:                    file,
		Package:                 firstString(impactReport.ChangedPackages),
		PackageName:             packageName,
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
	report.Purpose = filePurpose(report)
	report.Risk = fileRiskSummary(report)
	report.ReadingOrder = fileReadingOrder(report)
	report.Limitations = fileLimitations(options.IncludeTests, report.AnalysisMode, report.InterfaceAnalysisMode, report.TestAnalysisMode)
	report.Confidence = fileConfidence(report)
	report = applyFileLimits(report, limits)

	return normalizeFileReport(report), nil
}

func applyFileLimits(report FileReport, limits LimitOptions) FileReport {
	var truncation Truncation
	originalReadingOrderCount := len(report.ReadingOrder)

	report.AffectedTests = prioritizeContextTests(report.AffectedTests)
	report.Symbols, truncation.Symbols = limitSlice(report.Symbols, limits.MaxSymbols)
	report.SourceContexts, truncation.SourceContexts = limitSlice(report.SourceContexts, limits.MaxSymbols)
	report.AffectedTests, truncation.AffectedTests = limitSlice(report.AffectedTests, limits.MaxTests)
	report.ReadingOrder = fileReadingOrder(report)
	if originalReadingOrderCount > len(report.ReadingOrder) {
		truncation.ReadingOrder = originalReadingOrderCount - len(report.ReadingOrder)
	}

	report.Truncated = reportTruncation(truncation)
	report = applyFileByteLimit(report, limits.MaxBytes)

	return report
}

func normalizeFileTarget(root string, target string) (string, error) {
	value := strings.TrimSpace(target)
	if value == "" {
		return "", fmt.Errorf("file path is empty")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("absolute file paths are not supported: %s", target)
	}

	value = path.Clean(filepath.ToSlash(value))
	if value == "." || path.IsAbs(value) || value == ".." || strings.HasPrefix(value, "../") {
		return "", fmt.Errorf("context file target must be a repository-local Go file: %s", target)
	}
	if path.Ext(value) != ".go" {
		return "", fmt.Errorf("context file target must be a repository-local Go file: %s", target)
	}

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root %s: %w", root, err)
	}

	filePath := filepath.Join(rootPath, filepath.FromSlash(value))
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("context file target not found: %s", value)
		}
		return "", fmt.Errorf("stat context file %s: %w", value, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("context file target is a directory: %s", value)
	}

	return value, nil
}

func filePackageName(root string, file string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root %s: %w", root, err)
	}

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, filepath.Join(rootPath, filepath.FromSlash(file)), nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("parse package clause %s: %w", file, err)
	}
	if parsed.Name == nil {
		return "", fmt.Errorf("parse package clause %s: missing package name", file)
	}

	return parsed.Name.Name, nil
}

func symbolsInFile(symbols []sherpa.Symbol, file string) []sherpa.Symbol {
	var result []sherpa.Symbol
	for _, symbol := range symbols {
		if symbol.Position.File == file {
			result = append(result, symbol)
		}
	}

	sort.SliceStable(result, func(i int, j int) bool {
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

func sourceContextsForSymbols(root string, symbols []sherpa.Symbol, radius int) ([]sherpa.SourceContext, error) {
	if len(symbols) == 0 {
		return []sherpa.SourceContext{}, nil
	}

	positions := make([]sherpa.Position, 0, len(symbols))
	for _, symbol := range symbols {
		positions = append(positions, symbol.Position)
	}

	return sherpa.ReadSourceContexts(root, positions, radius)
}

func filePurpose(report FileReport) string {
	if report.Package == "" {
		return fmt.Sprintf("File %s could not be mapped to a repository-local Go package.", report.File)
	}
	if len(report.Symbols) == 0 {
		return fmt.Sprintf(
			"File %s belongs to package %s; no supported top-level symbols were found.",
			report.File,
			report.Package,
		)
	}

	return fmt.Sprintf(
		"File %s declares %s in package %s; impact analysis reaches %s and %s.",
		report.File,
		countNoun(len(report.Symbols), "supported symbol"),
		report.Package,
		countNoun(len(report.AffectedPackages), "package"),
		countNoun(len(report.AffectedTests), "affected test"),
	)
}

func fileRiskSummary(report FileReport) explainengine.RiskSummary {
	score := 0
	var reasons []string

	if len(report.Symbols) == 0 {
		reasons = append(reasons, "No supported top-level symbols found in the file.")
	} else {
		reasons = append(reasons, fmt.Sprintf("File declares %d supported symbols.", len(report.Symbols)))
		if len(report.Symbols) > 5 {
			score += 2
		} else if len(report.Symbols) > 2 {
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

func fileReadingOrder(report FileReport) []explainengine.ReadingStep {
	steps := []explainengine.ReadingStep{
		{
			Title:  "File: " + report.File,
			Reason: "Start with the target file and its package-level shape.",
			Position: sherpa.Position{
				File: report.File,
				Line: 1,
			},
		},
	}

	for _, symbol := range report.Symbols {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Symbol: " + symbol.DisplayName(),
			Reason:   "Inspect the symbols declared in the target file.",
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

func fileLimitations(includeTests bool, analysisMode string, interfaceAnalysisMode string, testAnalysisMode string) []string {
	values := []string{
		"File context uses package-level impact for affected packages and tests.",
		"Source excerpts are limited to supported top-level Go symbols: functions, methods, structs, interfaces, and type aliases.",
		fileContextAnalysisLimitation(analysisMode),
		interfaceAnalysisLimitation(interfaceAnalysisMode),
		testAnalysisLimitation(testAnalysisMode),
		"Dynamic dispatch, reflection, and complex function-value flows are not fully resolved.",
		"Generated Go files are included when package loading includes them; generated code emitted outside the parsed repository is not inferred.",
		"Package load warnings lower confidence; inspect envelope warnings before relying on semantic interfaces or tests.",
		"Test discovery uses direct references, same-package tests, file-contained symbols, and literal t.Run subtest names.",
	}

	if includeTests {
		values = append(values, "--tests is accepted for workflow symmetry; file context always includes affected tests from impact analysis.")
	}

	return values
}

func fileContextAnalysisLimitation(analysisMode string) string {
	switch analysisMode {
	case AnalysisModeTypecheckedAST:
		return "File context used shared typechecked package loading for package files, symbols, and interface impact where available."
	default:
		return "File context used AST fallback for package files and symbols because typechecked loading was unavailable."
	}
}

func fileConfidence(report FileReport) string {
	if len(report.Warnings) > 0 || report.File == "" || report.Package == "" {
		return ConfidenceLow
	}
	if report.InterfaceAnalysisMode == impactengine.InterfaceAnalysisModeASTFallback {
		return ConfidenceLow
	}

	return ConfidenceMedium
}

func normalizeFileReport(report FileReport) FileReport {
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
	report.Warnings = nonNilSlice(uniqueStrings(report.Warnings))

	return report
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
