package agentcontext

import (
	"fmt"
	"strings"

	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func FormatDiff(report DiffReport) string {
	var builder strings.Builder

	builder.WriteString("CONTEXT DIFF\n")
	builder.WriteString("\n")
	writeDiffBase(&builder, report)
	builder.WriteString("\n")
	writePurpose(&builder, report.Purpose)
	builder.WriteString("\n")
	writeDiffAnalysis(&builder, report)
	writeTruncation(&builder, report.Truncated)
	builder.WriteString("\n")
	writeValues(&builder, "CHANGED FILES", report.ChangedFiles)
	builder.WriteString("\n")
	writeValues(&builder, "CHANGED PACKAGES", report.ChangedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED SYMBOLS", report.AffectedSymbols)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED INTERFACES", report.AffectedInterfaces)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED IMPLEMENTATIONS", report.AffectedImplementations)
	builder.WriteString("\n")
	writeAffectedTests(&builder, report.AffectedTests)
	builder.WriteString("\n")
	sherpa.WriteTestPlan(&builder, report.TestPlan, report.TestCommands)
	builder.WriteString("\n")
	writeReadingOrder(&builder, report.ReadingOrder)
	builder.WriteString("\n")
	writeValues(&builder, "LIMITATIONS", report.Limitations)
	writeWarnings(&builder, report.Warnings)

	return builder.String()
}

func writeDiffBase(builder *strings.Builder, report DiffReport) {
	builder.WriteString("BASE\n")
	if strings.TrimSpace(report.Base) == "" {
		builder.WriteString("  unknown\n")
		return
	}

	fmt.Fprintf(builder, "  %s\n", report.Base)
}

func writeDiffAnalysis(builder *strings.Builder, report DiffReport) {
	builder.WriteString("ANALYSIS\n")
	mode := strings.TrimSpace(report.AnalysisMode)
	if mode == "" {
		mode = "unknown"
	}
	confidence := strings.TrimSpace(report.Confidence)
	if confidence == "" {
		confidence = "unknown"
	}

	fmt.Fprintf(builder, "  Mode: %s\n", mode)
	if strings.TrimSpace(report.ReferenceAnalysisMode) != "" {
		fmt.Fprintf(builder, "  Reference analysis: %s\n", report.ReferenceAnalysisMode)
	}
	if strings.TrimSpace(report.CallAnalysisMode) != "" {
		fmt.Fprintf(builder, "  Call analysis: %s\n", report.CallAnalysisMode)
	}
	writeInterfaceAnalysis(builder, report.InterfaceAnalysisMode)
	writeTestAnalysis(builder, report.TestAnalysisMode)
	fmt.Fprintf(builder, "  Confidence: %s\n", confidence)
	if report.Risk.Level != "" {
		fmt.Fprintf(builder, "  Risk: %s\n", report.Risk.Level)
	}
	for _, reason := range report.Risk.Reasons {
		fmt.Fprintf(builder, "  - %s\n", reason)
	}
}

func writeAffectedTests(builder *strings.Builder, tests []sherpa.RelatedTest) {
	builder.WriteString("AFFECTED TESTS\n")
	if len(tests) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, test := range tests {
		tags := relatedTestTags(test)
		if tags == "" {
			fmt.Fprintf(builder, "  %-36s %s:%d\n", test.Name, test.Position.File, test.Position.Line)
			continue
		}

		fmt.Fprintf(builder, "  %-36s %s:%d (%s)\n", test.Name, test.Position.File, test.Position.Line, tags)
	}
}

func writeReadingOrder(builder *strings.Builder, steps []explainengine.ReadingStep) {
	builder.WriteString("READING ORDER\n")
	if len(steps) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, step := range steps {
		if step.Position.File == "" {
			fmt.Fprintf(builder, "  %s\n", step.Title)
			continue
		}

		fmt.Fprintf(builder, "  %-36s %s:%d\n", step.Title, step.Position.File, step.Position.Line)
	}
}
