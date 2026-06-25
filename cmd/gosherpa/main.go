package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

const (
	exitSuccess       = 0
	exitFailure       = 1
	exitUsage         = 2
	jsonSchemaVersion = 1
)

type cliInvocation struct {
	Root              string
	Command           string
	CommandArgs       []string
	JSON              bool
	CallPathLimit     int
	CallPathMaxDepth  int
	HasCallPathOption bool
}

type jsonResponse[T any] struct {
	SchemaVersion int      `json:"schemaVersion"`
	Command       string   `json:"command"`
	Target        string   `json:"target"`
	Root          string   `json:"root"`
	ModulePath    string   `json:"modulePath"`
	Warnings      []string `json:"warnings"`
	Data          T        `json:"data"`
}

type referencesJSONData struct {
	References []sherpa.Reference `json:"references"`
}

type impactJSONData struct {
	Kind         sherpa.ImpactKind          `json:"kind"`
	References   []sherpa.Reference         `json:"references"`
	Callers      []sherpa.Caller            `json:"callers"`
	Dependencies sherpa.PackageDependencies `json:"dependencies"`
	Packages     []string                   `json:"packages"`
	RelatedTests []sherpa.RelatedTest       `json:"relatedTests"`
	TestCommands []string                   `json:"testCommands"`
}

type testsJSONData struct {
	Kind     sherpa.TestTargetKind `json:"kind"`
	Tests    []sherpa.RelatedTest  `json:"tests"`
	Commands []string              `json:"commands"`
}

type dependenciesJSONData struct {
	Package string   `json:"package"`
	Imports []string `json:"imports"`
	UsedBy  []string `json:"usedBy"`
}

type callersJSONData struct {
	Callers []sherpa.Caller `json:"callers"`
}

type calleesJSONData struct {
	Callees []sherpa.Callee `json:"callees"`
}

type callPathsJSONData struct {
	From  string            `json:"from"`
	To    string            `json:"to"`
	Paths []sherpa.CallPath `json:"paths"`
}

func parseCLIArgs(args []string) (cliInvocation, error) {
	invocation := cliInvocation{Root: "."}
	var positionals []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--json" {
			invocation.JSON = true
			continue
		}

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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	invocation, err := parseCLIArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}

	if invocation.Command == "" {
		printUsage(stderr)
		return exitUsage
	}

	if invocation.HasCallPathOption && invocation.Command != "path" && invocation.Command != "paths" {
		fmt.Fprintln(stderr, "error: --limit and --max-depth are only supported by path commands")
		return exitUsage
	}

	if invocation.JSON && knownCommand(invocation.Command) && !supportsJSON(invocation.Command) {
		fmt.Fprintln(stderr, "error: --json is only supported by refs, impact, tests, deps, path, paths, callers, and callees")
		return exitUsage
	}

	switch invocation.Command {
	case "symbol":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] symbol <name>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		name := invocation.CommandArgs[0]

		symbols, err := sherpa.ParseRepository(root)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		symbol := sherpa.FindSymbol(symbols, name)
		if symbol == nil {
			fmt.Fprintln(stderr, "symbol not found:", name)
			return exitFailure
		}

		fmt.Fprintln(stdout, "Name:", symbol.Name)
		fmt.Fprintln(stdout, "Kind:", symbol.Kind)
		fmt.Fprintln(stdout, "File:", symbol.Position.File)
		fmt.Fprintln(stdout, "Line:", symbol.Position.Line)
		return exitSuccess

	case "symbols":
		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		symbols, err := sherpa.ParseRepository(root)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		fmt.Fprint(stdout, sherpa.FormatSymbols(symbols))
		return exitSuccess

	case "refs":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] refs <name>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		name := invocation.CommandArgs[0]

		refs, err := sherpa.FindReferences(root, name)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		if invocation.JSON {
			return writeJSON(stdout, stderr, newJSONResponse(root, "refs", name, nil, referencesJSONData{
				References: nonNilSlice(refs),
			}))
		}

		fmt.Fprint(stdout, sherpa.FormatReferences(name, refs))
		return exitSuccess

	case "impact":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] impact <symbol-or-package>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindImpact(root, target)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		if invocation.JSON {
			normalizedResult := impactJSONResult(result)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"impact",
				normalizedResult.Target,
				normalizedResult.Warnings,
				impactJSONDataFromResult(normalizedResult),
			))
		}

		fmt.Fprint(stdout, sherpa.FormatImpact(result))
		return exitSuccess

	case "tests":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] tests <symbol-or-package>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindTests(root, target)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		if invocation.JSON {
			normalizedResult := testsJSONResult(result)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"tests",
				normalizedResult.Target,
				nil,
				testsJSONDataFromResult(normalizedResult),
			))
		}

		fmt.Fprint(stdout, sherpa.FormatTests(result))
		return exitSuccess

	case "deps":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] deps <package>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		targetPackage := invocation.CommandArgs[0]

		deps, err := sherpa.FindPackageDependencies(root, targetPackage)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		if invocation.JSON {
			normalizedDeps := dependenciesJSONResult(deps)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"deps",
				normalizedDeps.Package,
				nil,
				dependenciesJSONDataFromResult(normalizedDeps),
			))
		}

		fmt.Fprint(stdout, sherpa.FormatPackageDependencies(deps))
		return exitSuccess
	case "path", "paths":
		if len(invocation.CommandArgs) < 2 {
			fmt.Fprintf(stderr, "usage: gosherpa [--root <path>] %s <from> <to> [--limit <n>] [--max-depth <n>]\n", invocation.Command)
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		options := sherpa.CallPathOptions{
			Limit:    invocation.CallPathLimit,
			MaxDepth: invocation.CallPathMaxDepth,
		}

		result, err := sherpa.FindCallPaths(root, invocation.CommandArgs[0], invocation.CommandArgs[1], options)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		if invocation.JSON {
			normalizedResult := callPathsJSONResult(result)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				invocation.Command,
				callPathJSONTarget(normalizedResult),
				nil,
				callPathsJSONDataFromResult(normalizedResult),
			))
		}

		fmt.Fprint(stdout, sherpa.FormatCallPaths(result))
		return exitSuccess
	case "callers":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] callers <function-or-method>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindCallers(root, target)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		if invocation.JSON {
			normalizedResult := callersJSONResult(result)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"callers",
				normalizedResult.Target,
				nil,
				callersJSONDataFromResult(normalizedResult),
			))
		}

		fmt.Fprint(stdout, sherpa.FormatCallers(result))
		return exitSuccess
	case "callees":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] callees <function-or-method>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindCallees(root, target)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}

		if invocation.JSON {
			normalizedResult := calleesJSONResult(result)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"callees",
				normalizedResult.Target,
				nil,
				calleesJSONDataFromResult(normalizedResult),
			))
		}

		fmt.Fprint(stdout, sherpa.FormatCallees(result))
		return exitSuccess
	default:
		fmt.Fprintln(stderr, "unknown command:", invocation.Command)
		printUsage(stderr)
		return exitUsage
	}
}

func knownCommand(command string) bool {
	switch command {
	case "symbol", "symbols", "refs", "impact", "tests", "deps", "path", "paths", "callers", "callees":
		return true
	default:
		return false
	}
}

func supportsJSON(command string) bool {
	switch command {
	case "refs", "impact", "tests", "deps", "path", "paths", "callers", "callees":
		return true
	default:
		return false
	}
}

func writeJSON(stdout io.Writer, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitFailure
	}

	return exitSuccess
}

func newJSONResponse[T any](root string, command string, target string, warnings []string, data T) jsonResponse[T] {
	modulePath, err := sherpa.ModulePath(root)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	return jsonResponse[T]{
		SchemaVersion: jsonSchemaVersion,
		Command:       command,
		Target:        target,
		Root:          root,
		ModulePath:    modulePath,
		Warnings:      nonNilSlice(warnings),
		Data:          data,
	}
}

func impactJSONResult(result sherpa.ImpactResult) sherpa.ImpactResult {
	result.References = nonNilSlice(result.References)
	result.Callers = nonNilSlice(result.Callers)
	result.Dependencies.Imports = nonNilSlice(result.Dependencies.Imports)
	result.Dependencies.UsedBy = nonNilSlice(result.Dependencies.UsedBy)
	result.Packages = nonNilSlice(result.Packages)
	result.RelatedTests = nonNilSlice(result.RelatedTests)
	result.TestCommands = nonNilSlice(result.TestCommands)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func impactJSONDataFromResult(result sherpa.ImpactResult) impactJSONData {
	return impactJSONData{
		Kind:         result.Kind,
		References:   result.References,
		Callers:      result.Callers,
		Dependencies: result.Dependencies,
		Packages:     result.Packages,
		RelatedTests: result.RelatedTests,
		TestCommands: result.TestCommands,
	}
}

func testsJSONResult(result sherpa.TestsResult) sherpa.TestsResult {
	result.Tests = nonNilSlice(result.Tests)
	result.Commands = nonNilSlice(result.Commands)

	return result
}

func testsJSONDataFromResult(result sherpa.TestsResult) testsJSONData {
	return testsJSONData{
		Kind:     result.Kind,
		Tests:    result.Tests,
		Commands: result.Commands,
	}
}

func dependenciesJSONResult(result sherpa.PackageDependencies) sherpa.PackageDependencies {
	result.Imports = nonNilSlice(result.Imports)
	result.UsedBy = nonNilSlice(result.UsedBy)

	return result
}

func dependenciesJSONDataFromResult(result sherpa.PackageDependencies) dependenciesJSONData {
	return dependenciesJSONData{
		Package: result.Package,
		Imports: result.Imports,
		UsedBy:  result.UsedBy,
	}
}

func callersJSONResult(result sherpa.CallersResult) sherpa.CallersResult {
	result.Callers = nonNilSlice(result.Callers)

	return result
}

func callersJSONDataFromResult(result sherpa.CallersResult) callersJSONData {
	return callersJSONData{
		Callers: result.Callers,
	}
}

func calleesJSONResult(result sherpa.CalleesResult) sherpa.CalleesResult {
	result.Callees = nonNilSlice(result.Callees)

	return result
}

func calleesJSONDataFromResult(result sherpa.CalleesResult) calleesJSONData {
	return calleesJSONData{
		Callees: result.Callees,
	}
}

func callPathsJSONResult(result sherpa.CallPathsResult) sherpa.CallPathsResult {
	result.Paths = nonNilSlice(result.Paths)
	for i := range result.Paths {
		result.Paths[i].Steps = nonNilSlice(result.Paths[i].Steps)
	}

	return result
}

func callPathJSONTarget(result sherpa.CallPathsResult) string {
	return result.From + " -> " + result.To
}

func callPathsJSONDataFromResult(result sherpa.CallPathsResult) callPathsJSONData {
	return callPathsJSONData{
		From:  result.From,
		To:    result.To,
		Paths: result.Paths,
	}
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}

func resolveRootPath(root string, stderr io.Writer) (string, bool) {
	repositoryRoot, err := sherpa.ResolveRepositoryRoot(root)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return "", false
	}

	return repositoryRoot.Path, true
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gosherpa [--root <path>] <command> [args]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "global options:")
	fmt.Fprintln(writer, "  --root <path>    repository root, defaults to .")
	fmt.Fprintln(writer, "  --json           machine-readable output for analysis commands")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "commands:")
	fmt.Fprintln(writer, "  symbols")
	fmt.Fprintln(writer, "  symbol <name>")
	fmt.Fprintln(writer, "  refs <name>")
	fmt.Fprintln(writer, "  impact <symbol-or-package>")
	fmt.Fprintln(writer, "  tests <symbol-or-package>")
	fmt.Fprintln(writer, "  deps <package>")
	fmt.Fprintln(writer, "  path <from> <to>")
	fmt.Fprintln(writer, "  paths <from> <to> [--limit <n>] [--max-depth <n>]")
	fmt.Fprintln(writer, "  callers <function-or-method>")
	fmt.Fprintln(writer, "  callees <function-or-method>")
}
