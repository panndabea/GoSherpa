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
