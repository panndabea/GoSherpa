package impact

import (
	"fmt"
	"strings"
)

func FormatDiffReport(report ImpactReport) string {
	var builder strings.Builder

	builder.WriteString("IMPACT DIFF\n")
	builder.WriteString("\n")

	writeReportValues(&builder, "CHANGED FILES", report.ChangedFiles)
	builder.WriteString("\n")
	writeReportValues(&builder, "CHANGED PACKAGES", report.ChangedPackages)
	builder.WriteString("\n")
	writeReportValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writeReportRelatedTests(&builder, report.AffectedTests)
	builder.WriteString("\n")
	writeReportValues(&builder, "SUGGESTED COMMANDS", report.TestCommands)

	if len(report.Warnings) > 0 {
		builder.WriteString("\n")
		writeReportValues(&builder, "WARNINGS", report.Warnings)
	}

	return builder.String()
}

func FormatAffectedTestsReport(report ImpactReport) string {
	var builder strings.Builder

	writeReportRelatedTests(&builder, report.AffectedTests)
	builder.WriteString("\n")
	writeReportValues(&builder, "SUGGESTED COMMANDS", report.TestCommands)

	if len(report.Warnings) > 0 {
		builder.WriteString("\n")
		writeReportValues(&builder, "WARNINGS", report.Warnings)
	}

	return builder.String()
}

func writeReportValues(builder *strings.Builder, title string, values []string) {
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

func writeReportRelatedTests(builder *strings.Builder, tests []RelatedTest) {
	builder.WriteString("AFFECTED TESTS\n")

	if len(tests) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, test := range tests {
		tags := reportRelatedTestTags(test)
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

func reportRelatedTestTags(test RelatedTest) string {
	var tags []string
	if test.DirectReference {
		tags = append(tags, "direct")
	}
	if test.ExternalPackage {
		tags = append(tags, "external")
	}

	return strings.Join(tags, ", ")
}
