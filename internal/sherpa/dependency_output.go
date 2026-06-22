package sherpa

import (
	"fmt"
	"strings"
)

func FormatPackageDependencies(deps PackageDependencies) string {
	var builder strings.Builder

	builder.WriteString("PACKAGE\n")
	builder.WriteString("  ")
	builder.WriteString(deps.Package)
	builder.WriteString("\n\n")

	builder.WriteString("IMPORTS\n")
	if len(deps.Imports) == 0 {
		builder.WriteString("  none\n")
	} else {
		for _, importPath := range deps.Imports {
			builder.WriteString("  ")
			builder.WriteString(importPath)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\n")

	builder.WriteString("USED BY\n")
	if len(deps.UsedBy) == 0 {
		builder.WriteString("  none\n")
	} else {
		for _, packagePath := range deps.UsedBy {
			builder.WriteString("  ")
			builder.WriteString(packagePath)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func PrintPackageDependencies(deps PackageDependencies) {
	fmt.Print(FormatPackageDependencies(deps))
}
