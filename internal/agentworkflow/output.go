package agentworkflow

import (
	"fmt"
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func Format(report Report) string {
	report = normalizeReport(report)

	var builder strings.Builder
	builder.WriteString("AGENT CONTEXT\n\n")
	fmt.Fprintf(&builder, "Base: %s\n", report.Base)
	fmt.Fprintf(&builder, "Readiness: %s", report.Readiness.Status)
	if report.Readiness.PackageLoad.Status != "" {
		fmt.Fprintf(&builder, " (packages: %s)", report.Readiness.PackageLoad.Status)
	}
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Snapshot: %s", report.Snapshot.Status)
	if report.Snapshot.Requested {
		if report.Snapshot.Used {
			builder.WriteString(" (used)")
		} else {
			builder.WriteString(" (requested, not used)")
		}
	}
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Analysis: %s\n", valueOrUnknown(report.AnalysisMode))
	fmt.Fprintf(&builder, "Confidence: %s\n", valueOrUnknown(report.Confidence))
	fmt.Fprintf(&builder, "Target risk: %s (%s)\n", report.TargetRisk.Level, report.TargetRisk.Scope)
	builder.WriteString("\n")

	writeValues(&builder, "CHANGED FILES", report.ChangedFiles)
	builder.WriteString("\n")
	writeValues(&builder, "CHANGED PACKAGES", report.ChangedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED SYMBOLS", report.AffectedSymbols)
	builder.WriteString("\n")

	builder.WriteString("TESTS\n")
	if len(report.TestCommands) == 0 {
		builder.WriteString("  none\n")
	} else {
		for _, command := range report.TestCommands {
			fmt.Fprintf(&builder, "  %s\n", command)
		}
	}
	builder.WriteString("\n")
	sherpa.WriteTestPlan(&builder, report.TestPlan, report.TestCommands)
	builder.WriteString("\n")

	writeValues(&builder, "WARNINGS", report.Warnings)
	builder.WriteString("\n")
	writeValues(&builder, "NEXT COMMANDS", report.SuggestedCommands)

	return builder.String()
}

func writeValues(builder *strings.Builder, title string, values []string) {
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(values) == 0 {
		builder.WriteString("  none\n")
		return
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		fmt.Fprintf(builder, "  %s\n", value)
	}
}
