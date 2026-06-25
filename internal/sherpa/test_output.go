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

	builder.WriteString("SUGGESTED COMMANDS\n")
	if len(result.Commands) == 0 {
		builder.WriteString("  none\n")
	} else {
		for _, command := range result.Commands {
			builder.WriteString("  ")
			builder.WriteString(command)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func PrintTests(result TestsResult) {
	fmt.Print(FormatTests(result))
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
