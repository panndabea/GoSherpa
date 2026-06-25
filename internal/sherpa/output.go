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

		name := symbol.Name
		if symbol.Kind == SymbolKindMethod && symbol.Receiver != "" {
			name = symbol.Receiver + "." + symbol.Name
		}

		fmt.Fprintf(
			builder,
			"  %-36s %s:%d\n",
			name,
			symbol.Position.File,
			symbol.Position.Line,
		)
	}

	builder.WriteString("\n")
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
