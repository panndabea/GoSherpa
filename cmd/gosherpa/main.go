package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type cliInvocation struct {
	Root              string
	Command           string
	CommandArgs       []string
	CallPathLimit     int
	CallPathMaxDepth  int
	HasCallPathOption bool
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

		if arg == "--limit" {
			value, err := parsePositiveFlagValue("--limit", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.CallPathLimit = value
			invocation.HasCallPathOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--limit=") {
			value, err := parsePositiveInteger("--limit", strings.TrimPrefix(arg, "--limit="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.CallPathLimit = value
			invocation.HasCallPathOption = true
			continue
		}

		if arg == "--max-depth" {
			value, err := parsePositiveFlagValue("--max-depth", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.CallPathMaxDepth = value
			invocation.HasCallPathOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--max-depth=") {
			value, err := parsePositiveInteger("--max-depth", strings.TrimPrefix(arg, "--max-depth="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.CallPathMaxDepth = value
			invocation.HasCallPathOption = true
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

func parsePositiveFlagValue(flag string, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return 0, fmt.Errorf("missing value for %s", flag)
	}

	return parsePositiveInteger(flag, args[index+1])
}

func parsePositiveInteger(flag string, value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("missing value for %s", flag)
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid value for %s: %s", flag, trimmed)
	}

	return parsed, nil
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

	if invocation.HasCallPathOption && invocation.Command != "path" && invocation.Command != "paths" {
		fmt.Println("error: --limit and --max-depth are only supported by path commands")
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

	case "impact":
		if len(invocation.CommandArgs) < 1 {
			fmt.Println("usage: gosherpa [--root <path>] impact <symbol-or-package>")
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindImpact(root, target)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintImpact(result)

	case "tests":
		if len(invocation.CommandArgs) < 1 {
			fmt.Println("usage: gosherpa [--root <path>] tests <symbol-or-package>")
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindTests(root, target)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintTests(result)

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
	case "path", "paths":
		if len(invocation.CommandArgs) < 2 {
			fmt.Printf("usage: gosherpa [--root <path>] %s <from> <to> [--limit <n>] [--max-depth <n>]\n", invocation.Command)
			return
		}

		root, ok := resolveRootPath(invocation.Root)
		if !ok {
			return
		}

		options := sherpa.CallPathOptions{
			Limit:    invocation.CallPathLimit,
			MaxDepth: invocation.CallPathMaxDepth,
		}

		result, err := sherpa.FindCallPaths(root, invocation.CommandArgs[0], invocation.CommandArgs[1], options)
		if err != nil {
			fmt.Println("error:", err)
			return
		}

		sherpa.PrintCallPaths(result)
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
	fmt.Println("  impact <symbol-or-package>")
	fmt.Println("  tests <symbol-or-package>")
	fmt.Println("  deps <package>")
	fmt.Println("  path <from> <to>")
	fmt.Println("  paths <from> <to> [--limit <n>] [--max-depth <n>]")
	fmt.Println("  callers <function-or-method>")
	fmt.Println("  callees <function-or-method>")
}
