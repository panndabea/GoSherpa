package impact

import (
	"fmt"
	"strings"
)

func FormatImplementers(result ImplementersResult) string {
	if len(result.Implementers) == 0 {
		return fmt.Sprintf("no implementers found: %s\n", result.Target)
	}

	var builder strings.Builder

	builder.WriteString("IMPLEMENTERS\n")
	builder.WriteString("\n")
	builder.WriteString(result.Target)
	builder.WriteString("\n")
	builder.WriteString("\n")

	for _, implementer := range result.Implementers {
		fmt.Fprintf(
			&builder,
			"  %-36s %s:%d\n",
			implementer.Name,
			implementer.Position.File,
			implementer.Position.Line,
		)
	}

	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Found %d implementers\n", len(result.Implementers))

	return builder.String()
}

func FormatInterfaces(result InterfacesResult) string {
	if len(result.Interfaces) == 0 {
		return fmt.Sprintf("no interfaces found: %s\n", result.Target)
	}

	var builder strings.Builder

	builder.WriteString("INTERFACES\n")
	builder.WriteString("\n")
	builder.WriteString(result.Target)
	builder.WriteString("\n")
	builder.WriteString("\n")

	for _, iface := range result.Interfaces {
		fmt.Fprintf(
			&builder,
			"  %-36s %s:%d\n",
			iface.Name,
			iface.Position.File,
			iface.Position.Line,
		)
	}

	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Found %d interfaces\n", len(result.Interfaces))

	return builder.String()
}
