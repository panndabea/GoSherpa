package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type cliInvocation struct {
	Root        string
	Command     string
	CommandArgs []string
}

func parseCLIArgs(args []string) (cliInvocation, error) {
	invocation := cliInvocation{Root: "."}
	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--root" {
			if i+1 >= len(args) {
				return cliInvocation{}, fmt.Errorf("missing value for --root")
			}

			value := strings.TrimSpace(args[i+1])
			if value == "" {
				return cliInvocation{}, fmt.Errorf("missing value for --root")
			}

			invocation.Root = value
			i++
			continue
		}

		if strings.HasPrefix(arg, "--root=") {
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--root="))
			if value == "" {
				return cliInvocation{}, fmt.Errorf("missing value for --root")
			}

			invocation.Root = value
			continue
		}

		if strings.HasPrefix(arg, "-") {
			return cliInvocation{}, fmt.Errorf("unknown flag: %s", arg)
		}

		positionals = append(positionals, arg)
	}

	if len(positionals) > 0 {
		invocation.Command = positionals[0]
	}

	if len(positionals) > 1 {
		invocation.CommandArgs = positionals[1:]
	}

	return invocation, nil
}

func main() {
	invocation, err := parseCLIArgs(os.Args[1:])
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	if invocation.Command == "" {
		printUsage()
		return
	}

	switch invocation.Command {
	case "symbol":
		if len(invocation.CommandArgs) < 1 {
			fmt.Println("usage: gosherpa [--root <path>] symbol <name>")
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		name := invocation.CommandArgs[0]

		symbols, err := sherpa.ParseRepository(root)
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
		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		symbols, err := sherpa.ParseRepository(root)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintSymbols(symbols)

	case "refs":
		if len(invocation.CommandArgs) < 1 {
			fmt.Println("usage: gosherpa [--root <path>] refs <name>")
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		name := invocation.CommandArgs[0]

		refs, err := sherpa.FindReferences(root, name)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintReferences(name, refs)

	case "deps":
		if len(invocation.CommandArgs) < 1 {
			fmt.Println("usage: gosherpa [--root <path>] deps <package>")
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		targetPackage := invocation.CommandArgs[0]

		deps, err := sherpa.FindPackageDependencies(root, targetPackage)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintPackageDependencies(deps)
	case "callers":
		if len(invocation.CommandArgs) < 1 {
			fmt.Println("usage: gosherpa [--root <path>] callers <function-or-method>")
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindCallers(root, target)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintCallers(result)
	case "callees":
		if len(invocation.CommandArgs) < 1 {
			fmt.Println("usage: gosherpa [--root <path>] callees <function-or-method>")
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindCallees(root, target)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintCallees(result)
	default:
		fmt.Println("unknown command:", invocation.Command)
		printUsage()
	}
}

func resolveRootPath(root string) (string, bool) {
	repositoryRoot, err := sherpa.ResolveRepositoryRoot(root)
	if err != nil {
		fmt.Println("error:", err)
		return "", false
	}

	return repositoryRoot.Path, true
}

func printUsage() {
	fmt.Println("usage: gosherpa [--root <path>] <command> [args]")
	fmt.Println()
	fmt.Println("global options:")
	fmt.Println("  --root <path>    repository root, defaults to .")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  symbols")
	fmt.Println("  symbol <name>")
	fmt.Println("  refs <name>")
	fmt.Println("  deps <package>")
	fmt.Println("  callers <function-or-method>")
	fmt.Println("  callees <function-or-method>")
}
