package sherpa

import (
	"fmt"
	"strings"
)

func FormatRiskReport(report RiskReport) string {
	report = normalizeRiskReport(report)

	var builder strings.Builder
	builder.WriteString("RISK\n\n")
	fmt.Fprintf(&builder, "Level: %s\n", report.Level)
	fmt.Fprintf(&builder, "Score: %d\n", report.Score)
	fmt.Fprintf(&builder, "Analysis: %s\n", report.AnalysisMode)
	fmt.Fprintf(&builder, "Confidence: %s\n\n", report.Confidence)

	writeRiskSummary(&builder, report)
	writeRiskFactors(&builder, report.Factors)
	writePackageRiskSignals(&builder, report.Packages)
	writeRiskCycles(&builder, report.Cycles)
	writeRiskLimitations(&builder, report.Limitations)

	return builder.String()
}

func writeRiskSummary(builder *strings.Builder, report RiskReport) {
	builder.WriteString("SUMMARY\n")
	fmt.Fprintf(builder, "  Packages: %d\n", report.PackageCount)
	fmt.Fprintf(builder, "  Symbols: %d\n", report.SymbolCount)
	fmt.Fprintf(builder, "  Exported symbols: %d\n", report.ExportedSymbols)
	fmt.Fprintf(builder, "  Interfaces: %d\n", report.Interfaces)
	fmt.Fprintf(builder, "  Test packages: %d\n\n", report.TestPackages)
}

func writeRiskFactors(builder *strings.Builder, factors []RiskFactor) {
	builder.WriteString("FACTORS\n")
	if len(factors) == 0 {
		builder.WriteString("  none\n\n")
		return
	}

	for _, factor := range factors {
		fmt.Fprintf(
			builder,
			"  %-7s score=%d %-18s %s\n",
			factor.Level,
			factor.Score,
			factor.Category,
			factor.Description,
		)
	}
	builder.WriteString("\n")
}

func writePackageRiskSignals(builder *strings.Builder, packages []PackageRiskSignal) {
	builder.WriteString("PACKAGE SIGNALS\n")
	if len(packages) == 0 {
		builder.WriteString("  none\n\n")
		return
	}

	for _, pkg := range packages {
		fmt.Fprintf(
			builder,
			"  %-28s level=%s score=%d fan-in=%d fan-out=%d exported=%d interfaces=%d symbols=%d\n",
			pkg.Package,
			pkg.Level,
			pkg.Score,
			pkg.ImportedBy,
			pkg.LocalImports,
			pkg.ExportedSymbols,
			pkg.Interfaces,
			pkg.Symbols,
		)
		for _, reason := range pkg.Reasons {
			fmt.Fprintf(builder, "    - %s\n", reason)
		}
	}
	builder.WriteString("\n")
}

func writeRiskCycles(builder *strings.Builder, cycles []DependencyCycle) {
	builder.WriteString("DEPENDENCY CYCLES\n")
	if len(cycles) == 0 {
		builder.WriteString("  none\n\n")
		return
	}

	for _, cycle := range cycles {
		fmt.Fprintf(builder, "  %s\n", strings.Join(cycle.Packages, ", "))
	}
	builder.WriteString("\n")
}

func writeRiskLimitations(builder *strings.Builder, limitations []string) {
	builder.WriteString("LIMITATIONS\n")
	if len(limitations) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, limitation := range limitations {
		builder.WriteString("  - ")
		builder.WriteString(limitation)
		builder.WriteString("\n")
	}
}
