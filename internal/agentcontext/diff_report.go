package agentcontext

import (
	"fmt"
	"strings"

	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

type DiffAnalyzeOptions struct {
	IncludeTests bool `json:"includeTests"`
	BuildTags    []string
	UseSnapshot  bool         `json:"useSnapshot"`
	Limits       LimitOptions `json:"limits"`
}

type DiffReport struct {
	Target                  string                       `json:"target"`
	Base                    string                       `json:"base"`
	Purpose                 string                       `json:"purpose"`
	Risk                    explainengine.RiskSummary    `json:"risk"`
	ChangedFiles            []string                     `json:"changedFiles"`
	ChangedPackages         []string                     `json:"changedPackages"`
	AffectedPackages        []string                     `json:"affectedPackages"`
	AffectedSymbols         []string                     `json:"affectedSymbols"`
	ChangedSymbolDetails    []impactengine.ChangedSymbol `json:"changedSymbolDetails"`
	ReferenceAnalysisMode   string                       `json:"referenceAnalysisMode,omitempty"`
	CallAnalysisMode        string                       `json:"callAnalysisMode,omitempty"`
	AffectedInterfaces      []string                     `json:"affectedInterfaces"`
	AffectedImplementations []string                     `json:"affectedImplementations"`
	InterfaceAnalysisMode   string                       `json:"interfaceAnalysisMode,omitempty"`
	AffectedTests           []impactengine.RelatedTest   `json:"affectedTests"`
	TestAnalysisMode        string                       `json:"testAnalysisMode,omitempty"`
	TestCommands            []string                     `json:"testCommands"`
	TestPlan                sherpa.TestPlan              `json:"testPlan"`
	ReadingOrder            []explainengine.ReadingStep  `json:"readingOrder"`
	AnalysisMode            string                       `json:"analysisMode"`
	Confidence              string                       `json:"confidence"`
	Limits                  *LimitOptions                `json:"limits,omitempty"`
	Truncated               *Truncation                  `json:"truncated,omitempty"`
	Limitations             []string                     `json:"limitations"`
	Warnings                []string                     `json:"-"`
}

func AnalyzeDiff(root string, base string, options DiffAnalyzeOptions) (DiffReport, error) {
	snapshotSymbols, snapshotUsed, snapshotWarnings := diffSnapshotSymbols(root, options)
	impactReport, err := impactengine.AnalyzeDiffWithOptions(root, base, "", impactengine.AnalyzerOptions{
		BuildTags:          options.BuildTags,
		UseSnapshotSymbols: snapshotUsed,
		SnapshotSymbols:    snapshotSymbols,
	})
	if err != nil {
		return DiffReport{}, err
	}

	limits := normalizeDiffLimits(options.Limits)

	report := DiffReport{
		Target:                  base,
		Base:                    base,
		ChangedFiles:            impactReport.ChangedFiles,
		ChangedPackages:         impactReport.ChangedPackages,
		AffectedPackages:        impactReport.AffectedPackages,
		AffectedSymbols:         impactReport.AffectedSymbols,
		ChangedSymbolDetails:    impactReport.ChangedSymbolDetails,
		ReferenceAnalysisMode:   impactReport.ReferenceAnalysisMode,
		CallAnalysisMode:        impactReport.CallAnalysisMode,
		AffectedInterfaces:      impactReport.AffectedInterfaces,
		AffectedImplementations: impactReport.AffectedImplementations,
		InterfaceAnalysisMode:   impactReport.InterfaceAnalysisMode,
		AffectedTests:           impactReport.AffectedTests,
		TestAnalysisMode:        impactReport.TestAnalysisMode,
		TestCommands:            impactReport.TestCommands,
		TestPlan:                impactReport.TestPlan,
		AnalysisMode:            diffAnalysisMode(impactReport, snapshotUsed),
		Limits:                  reportLimits(limits),
		Warnings:                append(snapshotWarnings, impactReport.Warnings...),
	}
	report.Purpose = diffPurpose(report)
	report.Risk = diffRiskSummary(report)
	report.ReadingOrder = diffReadingOrder(report)
	report.Limitations = diffLimitations(options.IncludeTests, report)
	report.Confidence = diffConfidence(report)
	report = applyDiffLimits(report, limits)

	return normalizeDiffReport(report), nil
}

func diffSnapshotSymbols(root string, options DiffAnalyzeOptions) ([]sherpa.Symbol, bool, []string) {
	if !options.UseSnapshot {
		return nil, false, nil
	}

	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{
		BuildTags: options.BuildTags,
	})
	if inspect.Status == snapshotstore.StatusValid {
		return append([]sherpa.Symbol{}, stored.Symbols...), true, nil
	}

	return nil, false, []string{diffSnapshotFallbackWarning(inspect)}
}

func applyDiffLimits(report DiffReport, limits LimitOptions) DiffReport {
	var truncation Truncation
	originalReadingOrderCount := len(report.ReadingOrder)

	report.AffectedTests = prioritizeContextTests(report.AffectedTests)
	report.ChangedFiles, truncation.ChangedFiles = limitSlice(report.ChangedFiles, limits.MaxFiles)
	report.AffectedSymbols, truncation.AffectedSymbols = limitSlice(report.AffectedSymbols, limits.MaxSymbols)
	report.ChangedSymbolDetails, truncation.ChangedSymbolDetails = limitSlice(report.ChangedSymbolDetails, limits.MaxSymbols)
	report.AffectedTests, truncation.AffectedTests = limitSlice(report.AffectedTests, limits.MaxTests)
	report.ReadingOrder = diffReadingOrder(report)
	if originalReadingOrderCount > len(report.ReadingOrder) {
		truncation.ReadingOrder = originalReadingOrderCount - len(report.ReadingOrder)
	}

	report.Truncated = reportTruncation(truncation)
	report = applyDiffByteLimit(report, limits.MaxBytes)

	return report
}

func diffPurpose(report DiffReport) string {
	if len(report.ChangedFiles) == 0 {
		return "No changed files were found relative to the base ref."
	}
	if len(report.ChangedPackages) == 0 {
		return fmt.Sprintf(
			"Diff changes %s, but no repository-local Go packages were changed.",
			countNoun(len(report.ChangedFiles), "file"),
		)
	}

	return fmt.Sprintf(
		"Diff changes %s across %s. Impact analysis reaches %s, %s, and %s.",
		countNoun(len(report.ChangedFiles), "file"),
		countNoun(len(report.ChangedPackages), "Go package"),
		countNoun(len(report.AffectedPackages), "package"),
		countNoun(len(report.AffectedSymbols), "symbol"),
		countNoun(len(report.AffectedTests), "affected test"),
	)
}

func diffRiskSummary(report DiffReport) explainengine.RiskSummary {
	score := 0
	var reasons []string

	if len(report.ChangedPackages) == 0 {
		reasons = append(reasons, "No changed Go packages found.")
	} else {
		reasons = append(reasons, fmt.Sprintf("Changed Go packages: %d.", len(report.ChangedPackages)))
	}

	if len(report.AffectedPackages) > 1 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Impact reaches %d packages.", len(report.AffectedPackages)))
	} else if len(report.AffectedPackages) == 1 {
		score++
		reasons = append(reasons, "Impact stays within 1 package.")
	}

	if len(report.AffectedSymbols) > 0 {
		reasons = append(reasons, fmt.Sprintf("Affected symbols found: %d.", len(report.AffectedSymbols)))
	}

	interfaceSignals := len(report.AffectedInterfaces) + len(report.AffectedImplementations)
	if interfaceSignals > 0 {
		score += 2
		reasons = append(reasons, fmt.Sprintf("Touches %d interface or implementation signals.", interfaceSignals))
	}

	if len(report.AffectedTests) == 0 && len(report.ChangedPackages) > 0 {
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

func diffReadingOrder(report DiffReport) []explainengine.ReadingStep {
	return BuildDiffReadingOrder(report.ChangedFiles, report.ChangedSymbolDetails, report.AffectedTests)
}

func BuildDiffReadingOrder(changedFiles []string, changedSymbols []impactengine.ChangedSymbol, affectedTests []sherpa.RelatedTest) []explainengine.ReadingStep {
	steps := make([]explainengine.ReadingStep, 0, len(changedSymbols)+len(changedFiles)+len(affectedTests))
	for _, symbol := range changedSymbols {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Changed symbol: " + changedSymbolReadingTitle(symbol),
			Reason:   "Inspect the changed top-level symbol before broader package impact.",
			Position: symbol.Position,
			Range:    symbol.Range,
		})
	}

	for _, file := range changedFiles {
		steps = append(steps, explainengine.ReadingStep{
			Title:  "Changed file: " + file,
			Reason: "Start with the files changed by the diff.",
			Position: sherpa.Position{
				File: file,
				Line: 1,
			},
		})
	}

	for _, test := range affectedTests {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Test: " + test.Name,
			Reason:   "Check expected behavior and regression coverage.",
			Position: test.Position,
			Range:    test.Range,
		})
	}

	return steps
}

func changedSymbolReadingTitle(symbol impactengine.ChangedSymbol) string {
	title := strings.TrimSpace(symbol.Target)
	if title == "" {
		title = strings.TrimSpace(symbol.Name)
	}
	if title == "" {
		title = "(unknown)"
	}
	if symbol.Deleted {
		title += " (deleted)"
	}

	return title
}

func diffAnalysisMode(report impactengine.ImpactReport, snapshotUsed bool) string {
	if report.ReferenceAnalysisMode == sherpa.ReferenceAnalysisModeTypechecked ||
		report.CallAnalysisMode == sherpa.CallAnalysisModeTypechecked ||
		report.InterfaceAnalysisMode == impactengine.InterfaceAnalysisModeTypechecked {
		if snapshotUsed {
			return AnalysisModeSnapshotDiffTypechecked
		}
		return AnalysisModeDiffTypechecked
	}

	if snapshotUsed {
		return AnalysisModeSnapshotDiff
	}
	return AnalysisModeDiff
}

func diffLimitations(includeTests bool, report DiffReport) []string {
	values := []string{
		"Changed symbols are hunk-based and limited to top-level functions, methods, structs, and interfaces.",
		"Statement-level semantic impact, dynamic dispatch, reflection, and function values are not resolved.",
		"Test discovery uses direct references, same-package tests, file-contained symbols, and literal t.Run subtest names.",
	}
	switch report.AnalysisMode {
	case AnalysisModeSnapshotDiffTypechecked:
		values = append([]string{
			"Diff context reused a valid snapshot for current changed-symbol inventory and uses git diff plus typechecked symbol, reference, call, or interface signals where available.",
		}, values...)
	case AnalysisModeSnapshotDiff:
		values = append([]string{
			"Diff context reused a valid snapshot for current changed-symbol inventory and uses git diff plus syntax-level repository analysis.",
		}, values...)
	case AnalysisModeDiffTypechecked:
		values = append([]string{"Diff context uses git diff plus typechecked symbol, reference, call, or interface signals where available."}, values...)
	default:
		values = append([]string{"Diff context uses git diff plus syntax-level repository analysis, not full module loading."}, values...)
	}
	if strings.TrimSpace(report.ReferenceAnalysisMode) != "" {
		values = append(values, "Reference analysis mode: "+report.ReferenceAnalysisMode+".")
	}
	if strings.TrimSpace(report.CallAnalysisMode) != "" {
		values = append(values, "Call analysis mode: "+report.CallAnalysisMode+".")
	}
	if strings.TrimSpace(report.InterfaceAnalysisMode) != "" {
		values = append(values, interfaceAnalysisLimitation(report.InterfaceAnalysisMode))
	}
	if strings.TrimSpace(report.TestAnalysisMode) != "" {
		values = append(values, testAnalysisLimitation(report.TestAnalysisMode))
	}

	if includeTests {
		values = append(values, "--tests is accepted for workflow symmetry; diff context always includes affected tests from impact analysis.")
	}

	return values
}

func diffSnapshotFallbackWarning(inspect snapshotstore.InspectResult) string {
	message := strings.TrimSpace(inspect.Message)
	if message == "" {
		message = "snapshot could not be used"
	}
	if len(inspect.StaleReasons) > 0 {
		return fmt.Sprintf("snapshot not used: %s (%s); using live diff context analysis", message, strings.Join(inspect.StaleReasons, ", "))
	}

	return fmt.Sprintf("snapshot not used: %s; using live diff context analysis", message)
}

func diffConfidence(report DiffReport) string {
	if len(report.Warnings) > 0 {
		return ConfidenceLow
	}
	if report.ReferenceAnalysisMode == sherpa.ReferenceAnalysisModeASTFallback {
		return ConfidenceLow
	}
	if report.CallAnalysisMode == sherpa.CallAnalysisModeASTFallback {
		return ConfidenceLow
	}
	if report.InterfaceAnalysisMode == impactengine.InterfaceAnalysisModeASTFallback {
		return ConfidenceLow
	}

	return ConfidenceMedium
}

func normalizeDiffReport(report DiffReport) DiffReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.ChangedSymbolDetails = nonNilSlice(report.ChangedSymbolDetails)
	report.ReferenceAnalysisMode = strings.TrimSpace(report.ReferenceAnalysisMode)
	report.CallAnalysisMode = strings.TrimSpace(report.CallAnalysisMode)
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

func countNoun(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", noun)
	}

	return fmt.Sprintf("%d %ss", count, noun)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
