package main

import (
	"fmt"
	"strings"

	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type prReport struct {
	Base                    string                     `json:"base"`
	AnalysisMode            string                     `json:"analysisMode"`
	Confidence              string                     `json:"confidence"`
	Limitations             []string                   `json:"limitations"`
	ChangedFiles            []string                   `json:"changedFiles"`
	ChangedPackages         []string                   `json:"changedPackages"`
	ChangedSymbols          []string                   `json:"changedSymbols"`
	AffectedPackages        []string                   `json:"affectedPackages"`
	AffectedInterfaces      []string                   `json:"affectedInterfaces"`
	AffectedImplementations []string                   `json:"affectedImplementations"`
	InterfaceAnalysisMode   string                     `json:"interfaceAnalysisMode,omitempty"`
	Risk                    explainengine.RiskSummary  `json:"risk"`
	AffectedTests           []impactengine.RelatedTest `json:"affectedTests"`
	TestCommands            []string                   `json:"testCommands"`
	TestPlan                sherpa.TestPlan            `json:"testPlan"`
	VerificationCommands    []string                   `json:"verificationCommands"`
	Warnings                []string                   `json:"-"`
}

func analyzePR(root string, base string, buildTags []string) (prReport, error) {
	impactReport, err := impactengine.AnalyzeDiffWithOptions(root, base, "", impactengine.AnalyzerOptions{
		BuildTags: buildTags,
	})
	if err != nil {
		return prReport{}, err
	}

	report := prReport{
		Base:                    base,
		AnalysisMode:            analysisModeDiff,
		ChangedFiles:            impactReport.ChangedFiles,
		ChangedPackages:         impactReport.ChangedPackages,
		ChangedSymbols:          impactReport.AffectedSymbols,
		AffectedPackages:        impactReport.AffectedPackages,
		AffectedInterfaces:      impactReport.AffectedInterfaces,
		AffectedImplementations: impactReport.AffectedImplementations,
		InterfaceAnalysisMode:   strings.TrimSpace(impactReport.InterfaceAnalysisMode),
		AffectedTests:           impactReport.AffectedTests,
		TestCommands:            impactReport.TestCommands,
		TestPlan:                impactReport.TestPlan,
		Warnings:                impactReport.Warnings,
	}
	report.Confidence = jsonConfidence(report.Warnings, report.AnalysisMode, report.InterfaceAnalysisMode)
	report.Limitations = impactLimitations(report.AnalysisMode)
	report.Risk = prRiskSummary(report)
	report.VerificationCommands = prVerificationCommands(report.TestCommands)

	return normalizePRReport(report), nil
}

func normalizePRReport(report prReport) prReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.ChangedSymbols = nonNilSlice(report.ChangedSymbols)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.VerificationCommands = nonNilSlice(report.VerificationCommands)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func prRiskSummary(report prReport) explainengine.RiskSummary {
	level := "low"
	var reasons []string

	bump := func(next string, reason string) {
		if prRiskRank(next) > prRiskRank(level) {
			level = next
		}
		reasons = append(reasons, reason)
	}

	if len(report.Warnings) > 0 {
		bump("high", "Analysis emitted warnings; verify the reported impact before relying on it.")
	}
	if len(report.AffectedInterfaces) > 0 || len(report.AffectedImplementations) > 0 {
		bump("high", "Interface contracts or implementations may be affected.")
	}
	if prHasDependentPackages(report.ChangedPackages, report.AffectedPackages) {
		bump("medium", "Changes reach dependent packages outside the directly changed package set.")
	}
	if len(report.ChangedPackages) > 1 {
		bump("medium", "The diff spans multiple Go packages.")
	}
	if len(report.ChangedSymbols) == 0 && len(report.ChangedFiles) > 0 {
		bump("medium", "No changed top-level symbols were identified, so review file-level impact carefully.")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "No dependent packages or interface relationships were reported.")
	}

	return explainengine.RiskSummary{
		Level:   level,
		Reasons: uniqueStringsInOrder(reasons),
	}
}

func prRiskRank(level string) int {
	switch level {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func prHasDependentPackages(changedPackages []string, affectedPackages []string) bool {
	changed := make(map[string]struct{}, len(changedPackages))
	for _, pkg := range changedPackages {
		changed[pkg] = struct{}{}
	}

	for _, pkg := range affectedPackages {
		if _, ok := changed[pkg]; !ok {
			return true
		}
	}

	return false
}

func prVerificationCommands(testCommands []string) []string {
	commands := append([]string{}, testCommands...)
	commands = append(commands, "go test ./...")

	return uniqueStringsInOrder(commands)
}

func formatPRReport(report prReport) string {
	report = normalizePRReport(report)

	var builder strings.Builder
	builder.WriteString("PR REVIEW\n\n")
	fmt.Fprintf(&builder, "Base: %s\n", report.Base)
	fmt.Fprintf(&builder, "Analysis: %s\n", report.AnalysisMode)
	fmt.Fprintf(&builder, "Confidence: %s\n", report.Confidence)
	if report.InterfaceAnalysisMode != "" {
		fmt.Fprintf(&builder, "Interface analysis: %s\n", report.InterfaceAnalysisMode)
	}
	builder.WriteString("\n")

	writePRRisk(&builder, report.Risk)
	builder.WriteString("\n")
	writePRValues(&builder, "CHANGED FILES", report.ChangedFiles)
	builder.WriteString("\n")
	writePRValues(&builder, "CHANGED PACKAGES", report.ChangedPackages)
	builder.WriteString("\n")
	writePRValues(&builder, "CHANGED SYMBOLS", report.ChangedSymbols)
	builder.WriteString("\n")
	writePRValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writePRValues(&builder, "AFFECTED INTERFACES", report.AffectedInterfaces)
	builder.WriteString("\n")
	writePRValues(&builder, "AFFECTED IMPLEMENTATIONS", report.AffectedImplementations)
	builder.WriteString("\n")
	writePRAffectedTests(&builder, report.AffectedTests)
	builder.WriteString("\n")
	sherpa.WriteTestPlan(&builder, report.TestPlan, report.TestCommands)
	builder.WriteString("\n")
	writePRValues(&builder, "VERIFY", report.VerificationCommands)

	if len(report.Warnings) > 0 {
		builder.WriteString("\n")
		writePRValues(&builder, "WARNINGS", report.Warnings)
	}

	return builder.String()
}

func writePRRisk(builder *strings.Builder, risk explainengine.RiskSummary) {
	builder.WriteString("RISK\n")
	level := strings.TrimSpace(risk.Level)
	if level == "" {
		level = "unknown"
	}
	fmt.Fprintf(builder, "  Level: %s\n", level)
	for _, reason := range risk.Reasons {
		fmt.Fprintf(builder, "  %s\n", reason)
	}
}

func writePRValues(builder *strings.Builder, title string, values []string) {
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(values) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, value := range values {
		builder.WriteString("  ")
		builder.WriteString(value)
		builder.WriteString("\n")
	}
}

func writePRAffectedTests(builder *strings.Builder, tests []impactengine.RelatedTest) {
	builder.WriteString("AFFECTED TESTS\n")
	if len(tests) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, test := range tests {
		fmt.Fprintf(builder, "  %-36s %s:%d\n", test.Name, test.Position.File, test.Position.Line)
	}
}

func uniqueStringsInOrder(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
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
