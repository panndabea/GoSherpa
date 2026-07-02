package sherpa

import (
	"fmt"
	"strings"
)

func FormatEntryPoints(result EntryPointsResult) string {
	if len(result.EntryPoints) == 0 {
		var builder strings.Builder
		fmt.Fprintf(&builder, "no entrypoints found: %s\n", result.Target)
		writeCallAnalysis(&builder, result.AnalysisMode)
		writeCallWarnings(&builder, result.Warnings)
		return builder.String()
	}

	var builder strings.Builder

	builder.WriteString("ENTRYPOINTS\n")
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Target: %s\n", result.Target)
	writeCallAnalysis(&builder, result.AnalysisMode)
	builder.WriteString("\n")

	for _, entryPoint := range result.EntryPoints {
		fmt.Fprintf(
			&builder,
			"  %-17s %-36s %-20s %s:%d\n",
			entryPoint.Kind,
			entryPoint.Name,
			entryPoint.Package,
			entryPoint.Position.File,
			entryPoint.Position.Line,
		)
	}

	builder.WriteString("\n")
	writeCallWarnings(&builder, result.Warnings)
	fmt.Fprintf(&builder, "Found %d entrypoints\n", len(result.EntryPoints))

	return builder.String()
}

func PrintEntryPoints(result EntryPointsResult) {
	fmt.Print(FormatEntryPoints(result))
}
