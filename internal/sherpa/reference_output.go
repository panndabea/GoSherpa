package sherpa

import (
	"fmt"
	"strings"
)

func FormatReferences(name string, refs []Reference) string {
	return formatReferences(name, refs, nil)
}

func FormatReferencesWithContext(name string, refs []Reference, contexts []SourceContext) string {
	return formatReferences(name, refs, contexts)
}

func formatReferences(name string, refs []Reference, contexts []SourceContext) string {
	var builder strings.Builder

	builder.WriteString("REFERENCES\n")
	builder.WriteString("\n")
	builder.WriteString(name)
	builder.WriteString("\n")
	builder.WriteString("\n")

	if len(refs) == 0 {
		builder.WriteString("  none\n")
		builder.WriteString("\n")
		builder.WriteString("Found 0 references\n")
		return builder.String()
	}

	for index, ref := range refs {
		fmt.Fprintf(
			&builder,
			"  %s:%d\n",
			ref.Position.File,
			ref.Position.Line,
		)
		if index < len(contexts) {
			builder.WriteString(FormatSourceContext(contexts[index], "    "))
			if index < len(refs)-1 {
				builder.WriteString("\n")
			}
		}
	}

	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Found %d references\n", len(refs))

	return builder.String()
}

func PrintReferences(name string, refs []Reference) {
	fmt.Print(FormatReferences(name, refs))
}
