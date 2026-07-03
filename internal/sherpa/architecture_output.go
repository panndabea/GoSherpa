package sherpa

import (
	"fmt"
	"strings"
)

func FormatArchitectureReport(report ArchitectureReport) string {
	report = normalizeArchitectureReport(report)

	var builder strings.Builder
	builder.WriteString("ARCHITECTURE\n\n")
	fmt.Fprintf(&builder, "Analysis: %s\n", report.AnalysisMode)
	fmt.Fprintf(&builder, "Confidence: %s\n", report.Confidence)
	fmt.Fprintf(&builder, "Packages: %d\n\n", report.PackageCount)

	writeDependencyCycles(&builder, report.Cycles)
	writeArchitectureSignalSection(&builder, "MOST COUPLED PACKAGES", report.MostCoupled)
	writeArchitectureSignalSection(&builder, "HIGH FAN-IN PACKAGES", report.HighFanIn)
	writeArchitectureSignalSection(&builder, "HIGH FAN-OUT PACKAGES", report.HighFanOut)
	writeArchitectureSignalSection(&builder, "LARGEST PACKAGES", report.LargestPackages)
	writeArchitectureSignalSection(&builder, "LEAF PACKAGES", report.LeafPackages)
	writeArchitectureLimitations(&builder, report.Limitations)

	return builder.String()
}

func writeDependencyCycles(builder *strings.Builder, cycles []DependencyCycle) {
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

func writeArchitectureSignalSection(builder *strings.Builder, title string, signals []PackageArchitectureSignal) {
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(signals) == 0 {
		builder.WriteString("  none\n\n")
		return
	}

	for _, signal := range signals {
		fmt.Fprintf(
			builder,
			"  %-28s score=%d fan-in=%d fan-out=%d external=%d symbols=%d files=%d  %s\n",
			signal.Package,
			signal.Score,
			signal.ImportedBy,
			signal.LocalImports,
			signal.ExternalImports,
			signal.Symbols,
			signal.GoFiles,
			signal.Reason,
		)
	}
	builder.WriteString("\n")
}

func writeArchitectureLimitations(builder *strings.Builder, limitations []string) {
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
