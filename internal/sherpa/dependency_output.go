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

func FormatRepositoryDependencies(report RepositoryDependencies) string {
	var builder strings.Builder

	builder.WriteString("DEPENDENCIES\n\n")
	if len(report.Packages) == 0 {
		builder.WriteString("  none\n")
		return builder.String()
	}

	for i, pkg := range report.Packages {
		if i > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString(pkg.Package)
		builder.WriteString("\n")

		writeDependencyList(&builder, "LOCAL IMPORTS", pkg.LocalImports)
		writeDependencyList(&builder, "EXTERNAL IMPORTS", pkg.ExternalImports)
		writeDependencyList(&builder, "USED BY", pkg.UsedBy)
	}

	fmt.Fprintf(&builder, "\nFound %d packages\n", len(report.Packages))

	return builder.String()
}

func writeDependencyList(builder *strings.Builder, title string, values []string) {
	builder.WriteString("  ")
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(values) == 0 {
		builder.WriteString("    none\n")
		return
	}

	for _, value := range values {
		builder.WriteString("    ")
		builder.WriteString(value)
		builder.WriteString("\n")
	}
}

func PrintPackageDependencies(deps PackageDependencies) {
	fmt.Print(FormatPackageDependencies(deps))
}
