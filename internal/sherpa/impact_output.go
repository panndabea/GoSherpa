package sherpa

import (
	"fmt"
	"strings"
)

func FormatImpact(result ImpactResult) string {
	var builder strings.Builder

	builder.WriteString("IMPACT\n")
	builder.WriteString("\n")
	builder.WriteString("TARGET\n")
	fmt.Fprintf(&builder, "  %s (%s)\n", result.Target, result.Kind)
	builder.WriteString("\n")

	if result.Kind == ImpactKindPackage {
		writeImpactValues(&builder, "DIRECT DEPENDENTS", result.Dependencies.UsedBy)
		builder.WriteString("\n")
		writeImpactValues(&builder, "AFFECTED PACKAGES", result.Packages)
		builder.WriteString("\n")
		writeImpactRelatedTests(&builder, result.RelatedTests)
		builder.WriteString("\n")
		WriteTestPlan(&builder, result.TestPlan, result.TestCommands)
		writeImpactWarnings(&builder, result.Warnings)
		return builder.String()
	}

	writeImpactReferences(&builder, result.References)
	builder.WriteString("\n")
	writeImpactCallers(&builder, result.Callers)
	builder.WriteString("\n")
	writeImpactValues(&builder, "AFFECTED PACKAGES", result.Packages)
	builder.WriteString("\n")
	writeImpactRelatedTests(&builder, result.RelatedTests)
	builder.WriteString("\n")
	WriteTestPlan(&builder, result.TestPlan, result.TestCommands)
	writeImpactWarnings(&builder, result.Warnings)

	return builder.String()
}

func PrintImpact(result ImpactResult) {
	fmt.Print(FormatImpact(result))
}

func writeImpactReferences(builder *strings.Builder, refs []Reference) {
	builder.WriteString("REFERENCES\n")
	if len(refs) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, ref := range refs {
		fmt.Fprintf(builder, "  %s:%d\n", ref.Position.File, ref.Position.Line)
	}
}

func writeImpactCallers(builder *strings.Builder, callers []Caller) {
	builder.WriteString("CALLERS\n")
	if len(callers) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, caller := range callers {
		fmt.Fprintf(
			builder,
			"  %-36s %s:%d\n",
			caller.Name,
			caller.Position.File,
			caller.Position.Line,
		)
	}
}

func writeImpactValues(builder *strings.Builder, title string, values []string) {
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

func writeImpactRelatedTests(builder *strings.Builder, tests []RelatedTest) {
	builder.WriteString("SUGGESTED TESTS\n")
	if len(tests) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, test := range tests {
		tags := relatedTestTags(test)
		if tags == "" {
			fmt.Fprintf(
				builder,
				"  %-36s %s:%d\n",
				test.Name,
				test.Position.File,
				test.Position.Line,
			)
			continue
		}

		fmt.Fprintf(
			builder,
			"  %-36s %s:%d (%s)\n",
			test.Name,
			test.Position.File,
			test.Position.Line,
			tags,
		)
	}
}

func writeImpactWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	builder.WriteString("\n")
	writeImpactValues(builder, "WARNINGS", warnings)
}
