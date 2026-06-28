package sherpa

import (
	"fmt"
	"strings"
)

func FormatReferences(name string, refs []Reference) string {
	return formatReferences(name, refs, nil)
}

func FormatReferencesWithContext(name string, refs []Reference, contexts []SourceContext) string {
	return formatReferences(name, refs, contexts)
}

func FormatReferenceReport(report ReferenceReport) string {
	return formatReferenceReport(report, nil)
}

func FormatReferenceReportWithContext(report ReferenceReport, contexts []SourceContext) string {
	return formatReferenceReport(report, contexts)
}

func formatReferences(name string, refs []Reference, contexts []SourceContext) string {
	return formatReferenceBody(&strings.Builder{}, name, refs, contexts, "", nil)
}

func formatReferenceReport(report ReferenceReport, contexts []SourceContext) string {
	return formatReferenceBody(
		&strings.Builder{},
		report.Target,
		report.References,
		contexts,
		report.AnalysisMode,
		report.Warnings,
	)
}

func formatReferenceBody(
	builder *strings.Builder,
	name string,
	refs []Reference,
	contexts []SourceContext,
	analysisMode string,
	warnings []string,
) string {
	builder.WriteString("REFERENCES\n")
	builder.WriteString("\n")
	builder.WriteString(name)
	builder.WriteString("\n")
	if strings.TrimSpace(analysisMode) != "" {
		fmt.Fprintf(builder, "analysisMode: %q\n", analysisMode)
	}
	builder.WriteString("\n")

	if len(refs) == 0 {
		builder.WriteString("  none\n")
		builder.WriteString("\n")
		writeReferenceWarnings(builder, warnings)
		builder.WriteString("Found 0 references\n")
		return builder.String()
	}

	for index, ref := range refs {
		fmt.Fprintf(
			builder,
			"  %-12s %s:%d\n",
			formatReferenceKind(ref.Kind),
			ref.Position.File,
			ref.Position.Line,
		)
		if index < len(contexts) {
			builder.WriteString(FormatSourceContext(contexts[index], "    "))
			if index < len(refs)-1 {
				builder.WriteString("\n")
			}
		}
	}

	builder.WriteString("\n")
	writeReferenceWarnings(builder, warnings)
	fmt.Fprintf(builder, "Found %d references\n", len(refs))

	return builder.String()
}

func writeReferenceWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	builder.WriteString("WARNINGS\n")
	for _, warning := range warnings {
		fmt.Fprintf(builder, "  %s\n", warning)
	}
	builder.WriteString("\n")
}

func formatReferenceKind(kind ReferenceKind) string {
	if kind == "" {
		return string(ReferenceKindUsage)
	}

	return string(kind)
}

func PrintReferences(name string, refs []Reference) {
	fmt.Print(FormatReferences(name, refs))
}
