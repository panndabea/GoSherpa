package main

import (
	"fmt"
	"os"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "symbol":
		if len(os.Args) < 3 {
			fmt.Println("usage: gosherpa symbol <name>")
			return
		}

		name := os.Args[2]

		symbols, err := sherpa.ParseRepository(".")
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		symbol := sherpa.FindSymbol(symbols, name)
		if symbol == nil {
			fmt.Println("symbol not found:", name)
			return
		}

		fmt.Println("Name:", symbol.Name)
		fmt.Println("Kind:", symbol.Kind)
		fmt.Println("File:", symbol.Position.File)
		fmt.Println("Line:", symbol.Position.Line)

	case "symbols":
		symbols, err := sherpa.ParseRepository(".")
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintSymbols(symbols)

	case "refs":
		if len(os.Args) < 3 {
			fmt.Println("usage: gosherpa refs <name>")
			return
		}

		name := os.Args[2]

		refs, err := sherpa.FindReferences(".", name)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintReferences(name, refs)

	case "deps":
		if len(os.Args) < 3 {
			fmt.Println("usage: gosherpa deps <package>")
			return
		}

		targetPackage := os.Args[2]

		deps, err := sherpa.FindPackageDependencies(".", targetPackage)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintPackageDependencies(deps)
	case "callees":
		if len(os.Args) < 3 {
			fmt.Println("usage: gosherpa callees <function-or-method>")
			return
		}

		target := os.Args[2]

		result, err := sherpa.FindCallees(".", target)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintCallees(result)
	default:
		fmt.Println("unknown command:", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("usage: gosherpa <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  symbols")
	fmt.Println("  symbol <name>")
	fmt.Println("  refs <name>")
	fmt.Println("  deps <package>")
	fmt.Println("  callees <function-or-method>")
}
