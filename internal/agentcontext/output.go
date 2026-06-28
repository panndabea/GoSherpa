package agentcontext

import (
	"fmt"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func Format(report Report) string {
	var builder strings.Builder

	builder.WriteString("CONTEXT\n")
	builder.WriteString("\n")
	writeTarget(&builder, report)
	builder.WriteString("\n")
	writeDefinition(&builder, report.Symbol)
	builder.WriteString("\n")
	writeSource(&builder, report.SourceContext)
	builder.WriteString("\n")
	writePurpose(&builder, report.Purpose)
	builder.WriteString("\n")
	writeAnalysis(&builder, report)
	builder.WriteString("\n")
	writeCallers(&builder, report.Callers)
	builder.WriteString("\n")
	writeCallees(&builder, report.Callees)
	builder.WriteString("\n")
	writeReferences(&builder, report.References)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED INTERFACES", report.AffectedInterfaces)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED IMPLEMENTATIONS", report.AffectedImplementations)
	builder.WriteString("\n")
	writeRelatedTests(&builder, report.RelatedTests)
	builder.WriteString("\n")
	writeValues(&builder, "SUGGESTED COMMANDS", report.TestCommands)
	builder.WriteString("\n")
	writeValues(&builder, "LIMITATIONS", report.Limitations)
	writeWarnings(&builder, report.Warnings)

	return builder.String()
}

func writeTarget(builder *strings.Builder, report Report) {
	builder.WriteString("TARGET\n")
	fmt.Fprintf(builder, "  %s (%s)\n", report.Identity.Target, report.Identity.Kind)
	if report.Identity.Package != "" {
		fmt.Fprintf(builder, "  Package: %s\n", report.Identity.Package)
	}
	if report.Identity.QualifiedName != "" {
		fmt.Fprintf(builder, "  Qualified: %s\n", report.Identity.QualifiedName)
	}
	if report.Identity.Signature != "" {
		fmt.Fprintf(builder, "  Signature: %s\n", report.Identity.Signature)
	}
}

func writeDefinition(builder *strings.Builder, symbol sherpa.Symbol) {
	builder.WriteString("DEFINITION\n")
	fmt.Fprintf(builder, "  %s:%d\n", symbol.Position.File, symbol.Position.Line)
}

func writeSource(builder *strings.Builder, context sherpa.SourceContext) {
	builder.WriteString("SOURCE\n")
	if len(context.Lines) == 0 {
		builder.WriteString("  none\n")
		return
	}

	builder.WriteString(sherpa.FormatSourceContext(context, "  "))
}

func writePurpose(builder *strings.Builder, purpose string) {
	builder.WriteString("PURPOSE\n")
	purpose = strings.TrimSpace(purpose)
	if purpose == "" {
		builder.WriteString("  none\n")
		return
	}

	for _, line := range strings.Split(purpose, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			builder.WriteString("\n")
			continue
		}

		builder.WriteString("  ")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
}

func writeAnalysis(builder *strings.Builder, report Report) {
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
	fmt.Fprintf(builder, "  Confidence: %s\n", confidence)
	if report.Risk.Level != "" {
		fmt.Fprintf(builder, "  Risk: %s\n", report.Risk.Level)
	}
	if report.ArchitectureRole.Role != "" {
		fmt.Fprintf(builder, "  Architecture role: %s\n", report.ArchitectureRole.Role)
	}
}

func writeCallers(builder *strings.Builder, callers []sherpa.Caller) {
	builder.WriteString("CALLED BY\n")
	if len(callers) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, caller := range callers {
		fmt.Fprintf(builder, "  %-36s %s:%d\n", caller.Name, caller.Position.File, caller.Position.Line)
	}
}

func writeCallees(builder *strings.Builder, callees []sherpa.Callee) {
	builder.WriteString("CALLS\n")
	if len(callees) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, callee := range callees {
		fmt.Fprintf(builder, "  %-36s %s:%d\n", callee.Name, callee.Position.File, callee.Position.Line)
	}
}

func writeReferences(builder *strings.Builder, refs []sherpa.Reference) {
	builder.WriteString("REFERENCES\n")
	if len(refs) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, ref := range refs {
		fmt.Fprintf(builder, "  %s:%d\n", ref.Position.File, ref.Position.Line)
	}
}

func writeRelatedTests(builder *strings.Builder, tests []sherpa.RelatedTest) {
	builder.WriteString("SUGGESTED TESTS\n")
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

func relatedTestTags(test sherpa.RelatedTest) string {
	var tags []string
	if test.DirectReference {
		tags = append(tags, "direct")
	}

	if test.ExternalPackage {
		tags = append(tags, "external")
	}

	return strings.Join(tags, ", ")
}

func writeValues(builder *strings.Builder, title string, values []string) {
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(values) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, value := range values {
		for _, line := range strings.Split(value, "\n") {
			builder.WriteString("  ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
}

func writeWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	builder.WriteString("\n")
	writeValues(builder, "WARNINGS", warnings)
}
