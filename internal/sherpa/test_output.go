package sherpa

import (
	"fmt"
	"strings"
)

func FormatTests(result TestsResult) string {
	var builder strings.Builder

	builder.WriteString("TESTS\n")
	builder.WriteString("\n")
	builder.WriteString("TARGET\n")
	fmt.Fprintf(&builder, "  %s (%s)\n", result.Target, result.Kind)
	builder.WriteString("\n")

	builder.WriteString("ANALYSIS\n")
	fmt.Fprintf(&builder, "  Mode: %s\n", normalizeTestAnalysisMode(result.AnalysisMode))
	if len(result.Warnings) > 0 {
		builder.WriteString("\n")
		builder.WriteString("WARNINGS\n")
		for _, warning := range result.Warnings {
			fmt.Fprintf(&builder, "  - %s\n", warning)
		}
	}
	builder.WriteString("\n")

	builder.WriteString("RELATED TESTS\n")
	if len(result.Tests) == 0 {
		builder.WriteString("  none\n")
	} else {
		for _, test := range result.Tests {
			tags := relatedTestTags(test)
			if tags == "" {
				fmt.Fprintf(
					&builder,
					"  %-36s %s:%d\n",
					test.Name,
					test.Position.File,
					test.Position.Line,
				)
				continue
			}

			fmt.Fprintf(
				&builder,
				"  %-36s %s:%d (%s)\n",
				test.Name,
				test.Position.File,
				test.Position.Line,
				tags,
			)
		}
	}
	builder.WriteString("\n")

	WriteTestPlan(&builder, result.TestPlan, result.Commands)

	return builder.String()
}

func PrintTests(result TestsResult) {
	fmt.Print(FormatTests(result))
}

func normalizeTestAnalysisMode(mode string) string {
	if mode == "" {
		return TestAnalysisModeAST
	}

	return mode
}

func relatedTestTags(test RelatedTest) string {
	var tags []string
	if test.DirectReference {
		tags = append(tags, "direct")
	}

	if test.ExternalPackage {
		tags = append(tags, "external")
	}

	return strings.Join(tags, ", ")
}

func WriteTestPlan(builder *strings.Builder, plan TestPlan, fallbackCommands []string) {
	plan = NormalizeTestPlan(plan)
	if TestPlanEmpty(plan) && len(fallbackCommands) > 0 {
		plan = FallbackTestPlan(fallbackCommands)
	}

	builder.WriteString("TEST PLAN\n")
	writeTestPlanSection(builder, "DIRECT", plan.Direct)
	writeTestPlanSection(builder, "RELATED", plan.Related)
	writeTestPlanSection(builder, "CALLER PACKAGES", plan.CallerPackages)
	writeTestPlanSection(builder, "FALLBACK", plan.Fallback)
}

func writeTestPlanSection(builder *strings.Builder, title string, items []TestPlanItem) {
	builder.WriteString("  ")
	builder.WriteString(title)
	builder.WriteString("\n")

	if len(items) == 0 {
		builder.WriteString("    none\n")
		return
	}

	for _, item := range items {
		builder.WriteString("    ")
		builder.WriteString(item.Command)
		builder.WriteString("\n")
		if item.Reason != "" {
			builder.WriteString("      reason: ")
			builder.WriteString(item.Reason)
			builder.WriteString("\n")
		}
		if len(item.Targets) > 0 {
			builder.WriteString("      targets: ")
			builder.WriteString(strings.Join(item.Targets, ", "))
			builder.WriteString("\n")
		}
	}
}
