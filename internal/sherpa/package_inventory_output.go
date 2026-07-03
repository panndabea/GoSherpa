package sherpa

import (
	"fmt"
	"strings"
)

func FormatPackageSummaries(packages []PackageSummary) string {
	var builder strings.Builder

	builder.WriteString("PACKAGES\n\n")
	if len(packages) == 0 {
		builder.WriteString("  none\n")
		return builder.String()
	}

	fmt.Fprintf(
		&builder,
		"  %-28s %-12s %5s %5s %7s %7s %5s %8s %7s %5s\n",
		"PACKAGE",
		"NAME",
		"GO",
		"TEST",
		"SYMBOLS",
		"IMPORTS",
		"LOCAL",
		"EXTERNAL",
		"USED BY",
		"TESTS",
	)
	for _, pkg := range packages {
		fmt.Fprintf(
			&builder,
			"  %-28s %-12s %5d %5d %7d %7d %5d %8d %7d %5s\n",
			pkg.Package,
			pkg.PackageName,
			pkg.GoFiles,
			pkg.TestFiles,
			pkg.Symbols,
			pkg.Imports,
			pkg.LocalImports,
			pkg.ExternalImports,
			pkg.ImportedBy,
			yesNo(pkg.HasTests),
		)
	}

	return builder.String()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}
