package impact

import (
	"fmt"
	"strings"
)

func FormatInterface(result InterfaceResult) string {
	var builder strings.Builder

	builder.WriteString("INTERFACE\n")
	builder.WriteString("\n")
	fmt.Fprintf(&builder, "Target: %s\n", result.Target)
	if result.Position.File != "" {
		fmt.Fprintf(&builder, "Definition: %s:%d\n", result.Position.File, result.Position.Line)
	}
	writeInterfaceAnalysis(&builder, "Analysis", result.AnalysisMode)
	writeInterfaceAnalysis(&builder, "References", result.ReferenceAnalysisMode)
	writeInterfaceAnalysis(&builder, "Method usage", result.MethodUsageAnalysisMode)
	builder.WriteString("\n")

	builder.WriteString("METHODS\n")
	if len(result.Methods) == 0 {
		builder.WriteString("  no methods found\n")
	} else {
		for _, method := range result.Methods {
			if strings.TrimSpace(method.Signature) != "" {
				fmt.Fprintf(&builder, "  %s %s\n", method.Name, method.Signature)
			} else {
				fmt.Fprintf(&builder, "  %s\n", method.Name)
			}

			if len(method.Usages) == 0 {
				builder.WriteString("    no visible interface method usage found\n")
				continue
			}

			for _, usage := range method.Usages {
				fmt.Fprintf(
					&builder,
					"    %-10s %s:%d\n",
					formatInterfaceReferenceKind(usage.Kind),
					usage.Position.File,
					usage.Position.Line,
				)
			}
		}
	}
	builder.WriteString("\n")

	builder.WriteString("IMPLEMENTERS\n")
	if len(result.Implementers) == 0 {
		builder.WriteString("  no implementers found\n")
	} else {
		for _, implementer := range result.Implementers {
			fmt.Fprintf(
				&builder,
				"  %-36s %s:%d\n",
				implementer.Name,
				implementer.Position.File,
				implementer.Position.Line,
			)
		}
	}
	builder.WriteString("\n")

	builder.WriteString("REFERENCES\n")
	if len(result.References) == 0 {
		builder.WriteString("  no references found\n")
	} else {
		for _, reference := range result.References {
			fmt.Fprintf(
				&builder,
				"  %-10s %s:%d\n",
				formatInterfaceReferenceKind(reference.Kind),
				reference.Position.File,
				reference.Position.Line,
			)
		}
	}
	builder.WriteString("\n")

	writeInterfaceWarnings(&builder, result.Warnings)
	writeInterfaceLimitations(&builder, result.Limitations)
	fmt.Fprintf(&builder, "Found %d methods, %d implementers, %d references\n", len(result.Methods), len(result.Implementers), len(result.References))

	return builder.String()
}

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

func writeInterfaceAnalysis(builder *strings.Builder, label string, analysisMode string) {
	analysisMode = strings.TrimSpace(analysisMode)
	if analysisMode == "" {
		return
	}

	fmt.Fprintf(builder, "%s: %s\n", label, analysisMode)
}

func writeInterfaceWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	builder.WriteString("WARNINGS\n")
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}

		fmt.Fprintf(builder, "  %s\n", warning)
	}
	builder.WriteString("\n")
}

func writeInterfaceLimitations(builder *strings.Builder, limitations []string) {
	if len(limitations) == 0 {
		return
	}

	builder.WriteString("LIMITATIONS\n")
	for _, limitation := range limitations {
		limitation = strings.TrimSpace(limitation)
		if limitation == "" {
			continue
		}

		fmt.Fprintf(builder, "  %s\n", limitation)
	}
	builder.WriteString("\n")
}

func formatInterfaceReferenceKind(kind any) string {
	value := strings.TrimSpace(fmt.Sprint(kind))
	if value == "" {
		return "usage"
	}

	return value
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
