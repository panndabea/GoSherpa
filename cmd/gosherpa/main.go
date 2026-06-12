package main

import (
	"fmt"
	"os"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: gosherpa symbol <name>")
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
	default:
		fmt.Println("unknown command:", command)
		fmt.Println("usage: gosherpa symbol <name>")
	}
}
