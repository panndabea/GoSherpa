package agentcontext

import (
	"fmt"

	explainengine "github.com/supertabaluga/gosherpa/internal/explain"
	impactengine "github.com/supertabaluga/gosherpa/internal/impact"
	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type DiffAnalyzeOptions struct {
	IncludeTests bool `json:"includeTests"`
}

type DiffReport struct {
	Target                  string                      `json:"target"`
	Base                    string                      `json:"base"`
	Purpose                 string                      `json:"purpose"`
	Risk                    explainengine.RiskSummary   `json:"risk"`
	ChangedFiles            []string                    `json:"changedFiles"`
	ChangedPackages         []string                    `json:"changedPackages"`
	AffectedPackages        []string                    `json:"affectedPackages"`
	AffectedSymbols         []string                    `json:"affectedSymbols"`
	AffectedInterfaces      []string                    `json:"affectedInterfaces"`
	AffectedImplementations []string                    `json:"affectedImplementations"`
	AffectedTests           []impactengine.RelatedTest  `json:"affectedTests"`
	TestCommands            []string                    `json:"testCommands"`
	ReadingOrder            []explainengine.ReadingStep `json:"readingOrder"`
	AnalysisMode            string                      `json:"analysisMode"`
	Confidence              string                      `json:"confidence"`
	Limitations             []string                    `json:"limitations"`
	Warnings                []string                    `json:"-"`
}

func AnalyzeDiff(root string, base string, options DiffAnalyzeOptions) (DiffReport, error) {
	impactReport, err := impactengine.AnalyzeDiff(root, base, "")
	if err != nil {
		return DiffReport{}, err
	}

	report := DiffReport{
		Target:                  base,
		Base:                    base,
		ChangedFiles:            impactReport.ChangedFiles,
		ChangedPackages:         impactReport.ChangedPackages,
		AffectedPackages:        impactReport.AffectedPackages,
		AffectedSymbols:         impactReport.AffectedSymbols,
		AffectedInterfaces:      impactReport.AffectedInterfaces,
		AffectedImplementations: impactReport.AffectedImplementations,
		AffectedTests:           impactReport.AffectedTests,
		TestCommands:            impactReport.TestCommands,
		AnalysisMode:            AnalysisModeDiff,
		Warnings:                impactReport.Warnings,
	}
	report.Purpose = diffPurpose(report)
	report.Risk = diffRiskSummary(report)
	report.ReadingOrder = diffReadingOrder(report)
	report.Limitations = diffLimitations(options.IncludeTests)
	report.Confidence = diffConfidence(report)

	return normalizeDiffReport(report), nil
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
	steps := make([]explainengine.ReadingStep, 0, len(report.ChangedFiles)+len(report.AffectedTests))
	for _, file := range report.ChangedFiles {
		steps = append(steps, explainengine.ReadingStep{
			Title:  "Changed file: " + file,
			Reason: "Start with the files changed by the diff.",
			Position: sherpa.Position{
				File: file,
				Line: 1,
			},
		})
	}

	for _, test := range report.AffectedTests {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Test: " + test.Name,
			Reason:   "Check expected behavior and regression coverage.",
			Position: test.Position,
		})
	}

	return steps
}

func diffLimitations(includeTests bool) []string {
	values := []string{
		"Diff context uses git diff plus syntax-level repository analysis, not full module loading.",
		"Changed symbols are hunk-based and limited to top-level functions, methods, structs, and interfaces.",
		"Statement-level semantic impact, dynamic dispatch, reflection, and function values are not resolved.",
		"Test discovery uses same-package tests and syntactic direct-reference matching.",
	}

	if includeTests {
		values = append(values, "--tests is accepted for workflow symmetry; diff context always includes affected tests from impact analysis.")
	}

	return values
}

func diffConfidence(report DiffReport) string {
	if len(report.Warnings) > 0 {
		return ConfidenceLow
	}

	return ConfidenceMedium
}

func normalizeDiffReport(report DiffReport) DiffReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
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
