package sherpa

import (
	"fmt"
	"strings"
)

func FormatSymbols(symbols []Symbol) string {
	var builder strings.Builder

	writeSymbolGroup(&builder, "📦 STRUCTS", symbols, SymbolKindStruct)
	writeSymbolGroup(&builder, "🔌 INTERFACES", symbols, SymbolKindInterface)
	writeSymbolGroup(&builder, "⚙️ FUNCTIONS", symbols, SymbolKindFunction)
	writeSymbolGroup(&builder, "🔧 METHODS", symbols, SymbolKindMethod)
	writeSymbolTests(&builder, symbols)

	return builder.String()
}

func FormatSymbol(symbol Symbol) string {
	var builder strings.Builder

	builder.WriteString("SYMBOL\n\n")
	fmt.Fprintf(&builder, "Name: %s\n", symbol.DisplayName())
	fmt.Fprintf(&builder, "Kind: %s\n", symbol.Kind)
	if symbol.Package != "" {
		fmt.Fprintf(&builder, "Package: %s\n", symbol.Package)
	}
	if symbol.PackageName != "" {
		fmt.Fprintf(&builder, "Package name: %s\n", symbol.PackageName)
	}
	if symbol.QualifiedName != "" {
		fmt.Fprintf(&builder, "Qualified: %s\n", symbol.QualifiedName)
	}
	if symbol.Signature != "" {
		fmt.Fprintf(&builder, "Signature: %s\n", symbol.Signature)
	}
	if symbol.ReceiverType != "" {
		fmt.Fprintf(&builder, "Receiver: %s\n", symbol.ReceiverType)
	}

	fmt.Fprintf(&builder, "File: %s\n", symbol.Position.File)
	fmt.Fprintf(&builder, "Line: %d\n", symbol.Position.Line)
	if symbol.Position.Column > 0 {
		fmt.Fprintf(&builder, "Column: %d\n", symbol.Position.Column)
	}
	if symbol.Range != nil && symbol.Range.End.Line > 0 {
		fmt.Fprintf(&builder, "End: %s:%d", symbol.Range.End.File, symbol.Range.End.Line)
		if symbol.Range.End.Column > 0 {
			fmt.Fprintf(&builder, ":%d", symbol.Range.End.Column)
		}
		builder.WriteString("\n")
	}

	writeSymbolDocumentation(&builder, symbol)
	writeSymbolFields(&builder, symbol.Fields)
	writeSymbolMethods(&builder, symbol.Methods)

	return builder.String()
}

func PrintSymbols(symbols []Symbol) {
	fmt.Print(FormatSymbols(symbols))
}

func writeSymbolGroup(builder *strings.Builder, title string, symbols []Symbol, kind SymbolKind) {
	if !hasSymbolsOfKind(symbols, kind) {
		return
	}

	builder.WriteString(title)
	builder.WriteString("\n")

	for _, symbol := range symbols {
		if symbol.Kind != kind {
			continue
		}

		if kind == SymbolKindFunction && strings.HasPrefix(symbol.Name, "Test") {
			continue
		}

		fmt.Fprintf(
			builder,
			"  %-36s %s:%d\n",
			symbol.DisplayName(),
			symbol.Position.File,
			symbol.Position.Line,
		)
	}

	builder.WriteString("\n")
}

func writeSymbolDocumentation(builder *strings.Builder, symbol Symbol) {
	documentation := strings.TrimSpace(symbol.Documentation)
	if documentation == "" {
		return
	}

	builder.WriteString("\nDOCUMENTATION\n")
	for _, line := range strings.Split(documentation, "\n") {
		fmt.Fprintf(builder, "  %s\n", strings.TrimRight(line, " "))
	}
}

func writeSymbolFields(builder *strings.Builder, fields []SymbolField) {
	if len(fields) == 0 {
		return
	}

	builder.WriteString("\nFIELDS\n")
	for _, field := range fields {
		name := field.Name
		if field.Embedded {
			name = "(embedded) " + name
		}
		fmt.Fprintf(builder, "  %-24s %s", name, field.Type)
		if field.Tag != "" {
			fmt.Fprintf(builder, " `%s`", field.Tag)
		}
		builder.WriteString("\n")
	}
}

func writeSymbolMethods(builder *strings.Builder, methods []SymbolMethod) {
	if len(methods) == 0 {
		return
	}

	builder.WriteString("\nMETHODS\n")
	for _, method := range methods {
		name := method.Name
		if method.Embedded {
			name = "(embedded) " + name
		}
		fmt.Fprintf(builder, "  %-24s %s\n", name, method.Signature)
	}
}

func writeSymbolTests(builder *strings.Builder, symbols []Symbol) {
	if !hasTests(symbols) {
		return
	}

	builder.WriteString("🧪 TESTS\n")

	for _, symbol := range symbols {
		if symbol.Kind != SymbolKindFunction {
			continue
		}

		if !strings.HasPrefix(symbol.Name, "Test") {
			continue
		}

		fmt.Fprintf(
			builder,
			"  %-36s %s:%d\n",
			symbol.Name,
			symbol.Position.File,
			symbol.Position.Line,
		)
	}

	builder.WriteString("\n")
}

func hasSymbolsOfKind(symbols []Symbol, kind SymbolKind) bool {
	for _, symbol := range symbols {
		if symbol.Kind != kind {
			continue
		}

		if kind == SymbolKindFunction && strings.HasPrefix(symbol.Name, "Test") {
			continue
		}

		return true
	}

	return false
}

func hasTests(symbols []Symbol) bool {
	for _, symbol := range symbols {
		if symbol.Kind == SymbolKindFunction && strings.HasPrefix(symbol.Name, "Test") {
			return true
		}
	}

	return false
}
