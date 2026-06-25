package impact

import (
	"fmt"
	"strings"
)

func FormatDiffReport(report ImpactReport) string {
	return formatImpactReport("IMPACT DIFF", report, true, true, false)
}

func FormatFileReport(report ImpactReport) string {
	return formatImpactReport("IMPACT FILE", report, true, true, false)
}

func FormatPackageReport(report ImpactReport) string {
	return formatImpactReport("IMPACT PACKAGE", report, false, true, false)
}

func FormatSymbolReport(report ImpactReport) string {
	return formatImpactReport("IMPACT SYMBOL", report, false, false, true)
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

func formatImpactReport(title string, report ImpactReport, includeChangedFiles bool, includeChangedPackages bool, includeAffectedSymbols bool) string {
	var builder strings.Builder

	builder.WriteString(title)
	builder.WriteString("\n")
	builder.WriteString("\n")

	if includeChangedFiles {
		writeReportValues(&builder, "CHANGED FILES", report.ChangedFiles)
		builder.WriteString("\n")
	}
	if includeChangedPackages {
		writeReportValues(&builder, "CHANGED PACKAGES", report.ChangedPackages)
		builder.WriteString("\n")
	}
	if includeAffectedSymbols {
		writeReportValues(&builder, "AFFECTED SYMBOLS", report.AffectedSymbols)
		builder.WriteString("\n")
	}

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
