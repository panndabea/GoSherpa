package explain

import (
	"fmt"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func Format(report Report) string {
	var builder strings.Builder

	builder.WriteString("EXPLAIN\n")
	builder.WriteString("\n")
	builder.WriteString("TARGET\n")
	fmt.Fprintf(&builder, "  %s (%s)\n", report.Target, report.Symbol.Kind)
	builder.WriteString("\n")

	builder.WriteString("DEFINITION\n")
	fmt.Fprintf(&builder, "  %s:%d\n", report.Symbol.Position.File, report.Symbol.Position.Line)
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
	writeWarnings(&builder, report.Warnings)

	return builder.String()
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

func writeValues(builder *strings.Builder, title string, values []string) {
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

func writeWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	builder.WriteString("\n")
	writeValues(builder, "WARNINGS", warnings)
}
