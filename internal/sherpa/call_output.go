package sherpa

import (
	"fmt"
	"strings"
)

func FormatCallees(result CalleesResult) string {
	return formatCallees(result, nil)
}

func FormatCalleesWithContext(result CalleesResult, contexts []SourceContext) string {
	return formatCallees(result, contexts)
}

func formatCallees(result CalleesResult, contexts []SourceContext) string {
	if len(result.Callees) == 0 {
		var builder strings.Builder
		fmt.Fprintf(&builder, "no callees found: %s\n", result.Target)
		writeCallAnalysis(&builder, result.AnalysisMode)
		writeCallLimitations(&builder, result.Limitations)
		writeCallWarnings(&builder, result.Warnings)
		return builder.String()
	}

	var builder strings.Builder

	builder.WriteString("CALLEES\n")
	builder.WriteString("\n")
	builder.WriteString(result.Target)
	builder.WriteString("\n")
	writeCallAnalysis(&builder, result.AnalysisMode)
	builder.WriteString("\n")

	for index, callee := range result.Callees {
		fmt.Fprintf(
			&builder,
			"  %-36s %s:%d\n",
			callee.Name,
			callee.Position.File,
			callee.Position.Line,
		)
		if index < len(contexts) {
			builder.WriteString(FormatSourceContext(contexts[index], "    "))
			if index < len(result.Callees)-1 {
				builder.WriteString("\n")
			}
		}
	}

	builder.WriteString("\n")
	writeCallWarnings(&builder, result.Warnings)
	writeCallLimitations(&builder, result.Limitations)
	fmt.Fprintf(&builder, "Found %d callees\n", len(result.Callees))

	return builder.String()
}

func PrintCallees(result CalleesResult) {
	fmt.Print(FormatCallees(result))
}

func FormatCallers(result CallersResult) string {
	return formatCallers(result, nil)
}

func FormatCallersWithContext(result CallersResult, contexts []SourceContext) string {
	return formatCallers(result, contexts)
}

func formatCallers(result CallersResult, contexts []SourceContext) string {
	if len(result.Callers) == 0 {
		var builder strings.Builder
		fmt.Fprintf(&builder, "no callers found: %s\n", result.Target)
		writeCallAnalysis(&builder, result.AnalysisMode)
		writeCallLimitations(&builder, result.Limitations)
		writeCallWarnings(&builder, result.Warnings)
		return builder.String()
	}

	var builder strings.Builder

	builder.WriteString("CALLERS\n")
	builder.WriteString("\n")
	builder.WriteString(result.Target)
	builder.WriteString("\n")
	writeCallAnalysis(&builder, result.AnalysisMode)
	builder.WriteString("\n")

	for index, caller := range result.Callers {
		fmt.Fprintf(
			&builder,
			"  %-36s %s:%d\n",
			caller.Name,
			caller.Position.File,
			caller.Position.Line,
		)
		if index < len(contexts) {
			builder.WriteString(FormatSourceContext(contexts[index], "    "))
			if index < len(result.Callers)-1 {
				builder.WriteString("\n")
			}
		}
	}

	builder.WriteString("\n")
	writeCallWarnings(&builder, result.Warnings)
	writeCallLimitations(&builder, result.Limitations)
	fmt.Fprintf(&builder, "Found %d callers\n", len(result.Callers))

	return builder.String()
}

func PrintCallers(result CallersResult) {
	fmt.Print(FormatCallers(result))
}

func writeCallAnalysis(builder *strings.Builder, analysisMode string) {
	analysisMode = strings.TrimSpace(analysisMode)
	if analysisMode == "" {
		return
	}

	fmt.Fprintf(builder, "Analysis: %s\n", analysisMode)
}

func writeCallWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	builder.WriteString("WARNINGS\n")
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}

		fmt.Fprintf(builder, "  %s\n", warning)
	}
	builder.WriteString("\n")
}

func writeCallLimitations(builder *strings.Builder, limitations []string) {
	if len(limitations) == 0 {
		return
	}

	builder.WriteString("LIMITATIONS\n")
	for _, limitation := range limitations {
		limitation = strings.TrimSpace(limitation)
		if limitation == "" {
			continue
		}

		fmt.Fprintf(builder, "  %s\n", limitation)
	}
	builder.WriteString("\n")
}

func FormatCallPaths(result CallPathsResult) string {
	if len(result.Paths) == 0 {
		var builder strings.Builder
		fmt.Fprintf(&builder, "no call path found: %s -> %s\n", result.From, result.To)
		writeCallAnalysis(&builder, result.AnalysisMode)
		writeCallWarnings(&builder, result.Warnings)
		writeCallLimitations(&builder, result.Limitations)
		return builder.String()
	}

	var builder strings.Builder

	if len(result.Paths) == 1 {
		builder.WriteString("CALL PATH\n")
		builder.WriteString("\n")
		writeCallAnalysis(&builder, result.AnalysisMode)
		if strings.TrimSpace(result.AnalysisMode) != "" {
			builder.WriteString("\n")
		}
		writeCallPath(&builder, result.From, result.Paths[0], "")
		builder.WriteString("\n")
		writeCallWarnings(&builder, result.Warnings)
		writeCallLimitations(&builder, result.Limitations)
		fmt.Fprintf(&builder, "Found %d path\n", len(result.Paths))

		return builder.String()
	}

	builder.WriteString("CALL PATHS\n")
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "%s -> %s\n", result.From, result.To)
	writeCallAnalysis(&builder, result.AnalysisMode)
	builder.WriteString("\n")

	for i, path := range result.Paths {
		fmt.Fprintf(&builder, "Path %d\n", i+1)
		writeCallPath(&builder, result.From, path, "  ")
		builder.WriteString("\n")
	}

	writeCallWarnings(&builder, result.Warnings)
	writeCallLimitations(&builder, result.Limitations)
	fmt.Fprintf(&builder, "Found %d paths\n", len(result.Paths))

	return builder.String()
}

func PrintCallPaths(result CallPathsResult) {
	fmt.Print(FormatCallPaths(result))
}

func writeCallPath(builder *strings.Builder, from string, path CallPath, indent string) {
	fmt.Fprintf(builder, "%s%s\n", indent, from)

	for _, step := range path.Steps {
		fmt.Fprintf(
			builder,
			"%s  -> %-36s %s:%d\n",
			indent,
			step.Callee,
			step.Position.File,
			step.Position.Line,
		)
	}
}
