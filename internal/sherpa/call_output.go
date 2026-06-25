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

func FormatCallPaths(result CallPathsResult) string {
	if len(result.Paths) == 0 {
		return fmt.Sprintf("no call path found: %s -> %s\n", result.From, result.To)
	}

	var builder strings.Builder

	if len(result.Paths) == 1 {
		builder.WriteString("CALL PATH\n")
		builder.WriteString("\n")
		writeCallPath(&builder, result.From, result.Paths[0], "")
		builder.WriteString("\n")
		fmt.Fprintf(&builder, "Found %d path\n", len(result.Paths))

		return builder.String()
	}

	builder.WriteString("CALL PATHS\n")
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "%s -> %s\n", result.From, result.To)
	builder.WriteString("\n")

	for i, path := range result.Paths {
		fmt.Fprintf(&builder, "Path %d\n", i+1)
		writeCallPath(&builder, result.From, path, "  ")
		builder.WriteString("\n")
	}

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
