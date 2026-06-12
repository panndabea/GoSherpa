package sherpa

import (
	"fmt"
	"strings"
)

func PrintSymbols(symbols []Symbol) {
	printSymbolGroup("📦 STRUCTS", symbols, SymbolKindStruct)
	printSymbolGroup("🔌 INTERFACES", symbols, SymbolKindInterface)
	printSymbolGroup("⚙️ FUNCTIONS", symbols, SymbolKindFunction)
	printSymbolGroup("🔧 METHODS", symbols, SymbolKindMethod)
	printTests(symbols)
}

func printSymbolGroup(title string, symbols []Symbol, kind SymbolKind) {
	if !hasSymbolsOfKind(symbols, kind) {
		return
	}

	fmt.Println(title)

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

		fmt.Printf(
			"  %-36s %s:%d\n",
			name,
			symbol.Position.File,
			symbol.Position.Line,
		)
	}

	fmt.Println()
}

func printTests(symbols []Symbol) {
	if !hasTests(symbols) {
		return
	}

	fmt.Println("🧪 TESTS")

	for _, symbol := range symbols {
		if symbol.Kind != SymbolKindFunction {
			continue
		}

		if !strings.HasPrefix(symbol.Name, "Test") {
			continue
		}

		fmt.Printf(
			"  %-36s %s:%d\n",
			symbol.Name,
			symbol.Position.File,
			symbol.Position.Line,
		)
	}

	fmt.Println()
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