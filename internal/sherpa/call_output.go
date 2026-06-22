package sherpa

import (
	"fmt"
	"strings"
)

func FormatCallees(result CalleesResult) string {
	if len(result.Callees) == 0 {
		return fmt.Sprintf("no callees found: %s\n", result.Target)
	}

	var builder strings.Builder

	builder.WriteString("CALLEES\n")
	builder.WriteString("\n")
	builder.WriteString(result.Target)
	builder.WriteString("\n")
	builder.WriteString("\n")

	for _, callee := range result.Callees {
		fmt.Fprintf(
			&builder,
			"  %-36s %s:%d\n",
			callee.Name,
			callee.Position.File,
			callee.Position.Line,
		)
	}

	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Found %d callees\n", len(result.Callees))

	return builder.String()
}

func PrintCallees(result CalleesResult) {
	fmt.Print(FormatCallees(result))
}

func FormatCallers(result CallersResult) string {
	if len(result.Callers) == 0 {
		return fmt.Sprintf("no callers found: %s\n", result.Target)
	}

	var builder strings.Builder

	builder.WriteString("CALLERS\n")
	builder.WriteString("\n")
	builder.WriteString(result.Target)
	builder.WriteString("\n")
	builder.WriteString("\n")

	for _, caller := range result.Callers {
		fmt.Fprintf(
			&builder,
			"  %-36s %s:%d\n",
			caller.Name,
			caller.Position.File,
			caller.Position.Line,
		)
	}

	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Found %d callers\n", len(result.Callers))

	return builder.String()
}

func PrintCallers(result CallersResult) {
	fmt.Print(FormatCallers(result))
}
