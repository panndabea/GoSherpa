package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	explainengine "github.com/supertabaluga/gosherpa/internal/explain"
	impactengine "github.com/supertabaluga/gosherpa/internal/impact"
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
	HasLimitOption    bool
	HasMaxDepthOption bool
	BaseRef           string
	HasBaseOption     bool
	IncludeTests      bool
	HasTestsOption    bool
	SearchKind        sherpa.SymbolKind
	HasKindOption     bool
	SearchPackage     string
	HasPackageOption  bool
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

type jsonErrorResponse struct {
	SchemaVersion int           `json:"schemaVersion"`
	Command       string        `json:"command"`
	Target        string        `json:"target"`
	Root          string        `json:"root"`
	ModulePath    string        `json:"modulePath"`
	Warnings      []string      `json:"warnings"`
	Error         jsonErrorData `json:"error"`
}

type jsonErrorData struct {
	Code       string                   `json:"code"`
	Message    string                   `json:"message"`
	Kind       string                   `json:"kind,omitempty"`
	Target     string                   `json:"target,omitempty"`
	Candidates []sherpa.TargetCandidate `json:"candidates,omitempty"`
}

type referencesJSONData struct {
	References []sherpa.Reference `json:"references"`
}

type symbolJSONData struct {
	Symbol sherpa.Symbol `json:"symbol"`
}

type symbolsJSONData struct {
	Symbols []sherpa.Symbol `json:"symbols"`
}

type searchJSONData struct {
	Terms   []string                    `json:"terms"`
	Results []sherpa.SymbolSearchResult `json:"results"`
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

type impactDiffJSONData struct {
	ChangedFiles            []string                   `json:"changedFiles"`
	ChangedPackages         []string                   `json:"changedPackages"`
	AffectedPackages        []string                   `json:"affectedPackages"`
	AffectedSymbols         []string                   `json:"affectedSymbols"`
	AffectedInterfaces      []string                   `json:"affectedInterfaces"`
	AffectedImplementations []string                   `json:"affectedImplementations"`
	AffectedTests           []impactengine.RelatedTest `json:"affectedTests"`
	TestCommands            []string                   `json:"testCommands"`
}

type testsJSONData struct {
	Kind     sherpa.TestTargetKind `json:"kind"`
	Tests    []sherpa.RelatedTest  `json:"tests"`
	Commands []string              `json:"commands"`
}

type testsAffectedJSONData struct {
	AffectedTests []impactengine.RelatedTest `json:"affectedTests"`
	Commands      []string                   `json:"commands"`
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

type explainJSONData struct {
	Target                  string                         `json:"target"`
	Symbol                  sherpa.Symbol                  `json:"symbol"`
	Purpose                 string                         `json:"purpose"`
	Risk                    explainengine.RiskSummary      `json:"risk"`
	ArchitectureRole        explainengine.ArchitectureRole `json:"architectureRole"`
	References              []sherpa.Reference             `json:"references"`
	Callers                 []sherpa.Caller                `json:"callers"`
	Callees                 []sherpa.Callee                `json:"callees"`
	AffectedPackages        []string                       `json:"affectedPackages"`
	AffectedInterfaces      []string                       `json:"affectedInterfaces"`
	AffectedImplementations []string                       `json:"affectedImplementations"`
	RelatedTests            []sherpa.RelatedTest           `json:"relatedTests"`
	TestCommands            []string                       `json:"testCommands"`
	ReadingOrder            []explainengine.ReadingStep    `json:"readingOrder"`
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

		if arg == "--tests" {
			invocation.IncludeTests = true
			invocation.HasTestsOption = true
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

		if arg == "--base" {
			value, err := parseStringFlagValue("--base", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.BaseRef = value
			invocation.HasBaseOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--base=") {
			value, err := parseStringFlag("--base", strings.TrimPrefix(arg, "--base="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.BaseRef = value
			invocation.HasBaseOption = true
			continue
		}

		if arg == "--kind" {
			value, err := parseSymbolKindFlagValue("--kind", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.SearchKind = value
			invocation.HasKindOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--kind=") {
			value, err := parseSymbolKindFlag("--kind", strings.TrimPrefix(arg, "--kind="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.SearchKind = value
			invocation.HasKindOption = true
			continue
		}

		if arg == "--package" {
			value, err := parseStringFlagValue("--package", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.SearchPackage = value
			invocation.HasPackageOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--package=") {
			value, err := parseStringFlag("--package", strings.TrimPrefix(arg, "--package="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.SearchPackage = value
			invocation.HasPackageOption = true
			continue
		}

		if arg == "--limit" {
			value, err := parsePositiveFlagValue("--limit", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.CallPathLimit = value
			invocation.HasLimitOption = true
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
			invocation.HasLimitOption = true
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
			invocation.HasMaxDepthOption = true
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
			invocation.HasMaxDepthOption = true
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

func parseStringFlagValue(flag string, args []string, index int) (string, error) {
	if index+1 >= len(args) {
		return "", fmt.Errorf("missing value for %s", flag)
	}

	return parseStringFlag(flag, args[index+1])
}

func parseStringFlag(flag string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return "", fmt.Errorf("missing value for %s", flag)
	}

	return trimmed, nil
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

func parseSymbolKindFlagValue(flag string, args []string, index int) (sherpa.SymbolKind, error) {
	if index+1 >= len(args) {
		return "", fmt.Errorf("missing value for %s", flag)
	}

	return parseSymbolKindFlag(flag, args[index+1])
}

func parseSymbolKindFlag(flag string, value string) (sherpa.SymbolKind, error) {
	trimmed, err := parseStringFlag(flag, value)
	if err != nil {
		return "", err
	}

	kind := sherpa.SymbolKind(strings.ToLower(trimmed))
	if isSupportedSearchKind(kind) {
		return kind, nil
	}

	return "", fmt.Errorf("invalid value for %s: %s", flag, trimmed)
}

func isSupportedSearchKind(kind sherpa.SymbolKind) bool {
	switch kind {
	case sherpa.SymbolKindStruct, sherpa.SymbolKindInterface, sherpa.SymbolKindFunction, sherpa.SymbolKindMethod:
		return true
	default:
		return false
	}
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

	if invocation.HasLimitOption && !supportsLimitOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --limit is only supported by search and path commands")
		return exitUsage
	}

	if invocation.HasMaxDepthOption && !isPathCommand(invocation.Command) {
		fmt.Fprintln(stderr, "error: --max-depth is only supported by path commands")
		return exitUsage
	}

	if (invocation.HasKindOption || invocation.HasPackageOption) && invocation.Command != "search" {
		fmt.Fprintln(stderr, "error: --kind and --package are only supported by search")
		return exitUsage
	}

	if invocation.HasBaseOption && !isBaseAwareInvocation(invocation) {
		fmt.Fprintln(stderr, "error: --base is only supported by impact diff and tests affected")
		return exitUsage
	}

	if invocation.HasTestsOption && !supportsTestsOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --tests is only supported by search, callers, and explain")
		return exitUsage
	}

	if invocation.JSON && knownCommand(invocation.Command) && !supportsJSON(invocation.Command) {
		fmt.Fprintln(stderr, "error: --json is only supported by known commands")
		return exitUsage
	}

	switch invocation.Command {
	case "explain":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] explain <symbol> [--tests]")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		report, err := explainengine.AnalyzeWithOptions(root, target, explainengine.AnalyzeOptions{
			IncludeTests: invocation.IncludeTests,
		})
		if err != nil {
			return writeCommandError(invocation.JSON, root, "explain", target, stderr, err)
		}

		if invocation.JSON {
			normalizedReport := explainJSONResult(report)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"explain",
				normalizedReport.Target,
				normalizedReport.Warnings,
				explainJSONDataFromReport(normalizedReport),
			))
		}

		fmt.Fprint(stdout, explainengine.Format(report))
		return exitSuccess

	case "symbol":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] symbol <target>")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		symbols, err := sherpa.ParseRepository(root)
		if err != nil {
			return writeCommandError(invocation.JSON, root, "symbol", target, stderr, err)
		}

		symbol, err := sherpa.FindSymbolTarget(root, symbols, target)
		if err != nil {
			return writeCommandError(invocation.JSON, root, "symbol", target, stderr, err)
		}

		if invocation.JSON {
			return writeJSON(stdout, stderr, newJSONResponse(root, "symbol", target, nil, symbolJSONData{
				Symbol: symbol,
			}))
		}

		fmt.Fprint(stdout, sherpa.FormatSymbol(symbol))
		return exitSuccess

	case "search":
		if len(invocation.CommandArgs) < 1 {
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] search <terms> [--kind <kind>] [--package <package>] [--tests] [--limit <n>]")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		terms := invocation.CommandArgs
		target := strings.Join(terms, " ")

		symbols, err := sherpa.ParseRepository(root)
		if err != nil {
			return writeCommandError(invocation.JSON, root, "search", target, stderr, err)
		}

		results := sherpa.SearchSymbolsWithOptions(symbols, terms, sherpa.SymbolSearchOptions{
			Kind:      invocation.SearchKind,
			Package:   invocation.SearchPackage,
			TestsOnly: invocation.IncludeTests,
			Limit:     invocation.CallPathLimit,
		})
		if invocation.JSON {
			return writeJSON(stdout, stderr, newJSONResponse(root, "search", target, nil, searchJSONData{
				Terms:   nonNilSlice(terms),
				Results: nonNilSlice(results),
			}))
		}

		fmt.Fprint(stdout, sherpa.FormatSymbolSearch(terms, results))
		return exitSuccess

	case "symbols":
		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		symbols, err := sherpa.ParseRepository(root)
		if err != nil {
			return writeCommandError(invocation.JSON, root, "symbols", "", stderr, err)
		}

		if invocation.JSON {
			return writeJSON(stdout, stderr, newJSONResponse(root, "symbols", "", nil, symbolsJSONData{
				Symbols: nonNilSlice(symbols),
			}))
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
			return writeCommandError(invocation.JSON, root, "refs", name, stderr, err)
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
			printImpactUsage(stderr)
			return exitUsage
		}

		if invocation.CommandArgs[0] == "diff" {
			if len(invocation.CommandArgs) != 1 || !invocation.HasBaseOption {
				printImpactDiffUsage(stderr)
				return exitUsage
			}

			root, ok := resolveRootPath(invocation.Root, stderr)
			if !ok {
				return exitFailure
			}

			report, err := impactengine.AnalyzeDiff(root, invocation.BaseRef, "")
			if err != nil {
				return writeCommandError(invocation.JSON, root, "impact diff", invocation.BaseRef, stderr, err)
			}

			if invocation.JSON {
				normalizedReport := impactDiffJSONResult(report)
				return writeJSON(stdout, stderr, newJSONResponse(
					root,
					"impact diff",
					invocation.BaseRef,
					normalizedReport.Warnings,
					impactDiffJSONDataFromReport(normalizedReport),
				))
			}

			fmt.Fprint(stdout, impactengine.FormatDiffReport(report))
			return exitSuccess
		}

		if isImpactReportSubcommand(invocation.CommandArgs[0]) {
			if len(invocation.CommandArgs) != 2 {
				printImpactSubcommandUsage(stderr, invocation.CommandArgs[0])
				return exitUsage
			}

			root, ok := resolveRootPath(invocation.Root, stderr)
			if !ok {
				return exitFailure
			}

			kind := invocation.CommandArgs[0]
			target := invocation.CommandArgs[1]
			report, err := analyzeImpactSubcommand(root, kind, target)
			if err != nil {
				return writeCommandError(invocation.JSON, root, "impact "+kind, target, stderr, err)
			}

			if invocation.JSON {
				normalizedReport := impactDiffJSONResult(report)
				return writeJSON(stdout, stderr, newJSONResponse(
					root,
					"impact "+kind,
					target,
					normalizedReport.Warnings,
					impactDiffJSONDataFromReport(normalizedReport),
				))
			}

			fmt.Fprint(stdout, formatImpactSubcommandReport(kind, report))
			return exitSuccess
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindImpact(root, target)
		if err != nil {
			return writeCommandError(invocation.JSON, root, "impact", target, stderr, err)
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
			printTestsUsage(stderr)
			return exitUsage
		}

		if invocation.CommandArgs[0] == "affected" {
			if len(invocation.CommandArgs) != 1 || !invocation.HasBaseOption {
				printTestsAffectedUsage(stderr)
				return exitUsage
			}

			root, ok := resolveRootPath(invocation.Root, stderr)
			if !ok {
				return exitFailure
			}

			report, err := impactengine.AnalyzeDiff(root, invocation.BaseRef, "")
			if err != nil {
				return writeCommandError(invocation.JSON, root, "tests affected", invocation.BaseRef, stderr, err)
			}

			if invocation.JSON {
				normalizedReport := impactDiffJSONResult(report)
				return writeJSON(stdout, stderr, newJSONResponse(
					root,
					"tests affected",
					invocation.BaseRef,
					normalizedReport.Warnings,
					testsAffectedJSONDataFromReport(normalizedReport),
				))
			}

			fmt.Fprint(stdout, impactengine.FormatAffectedTestsReport(report))
			return exitSuccess
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindTests(root, target)
		if err != nil {
			return writeCommandError(invocation.JSON, root, "tests", target, stderr, err)
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
			return writeCommandError(invocation.JSON, root, "deps", targetPackage, stderr, err)
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
			return writeCommandError(
				invocation.JSON,
				root,
				invocation.Command,
				invocation.CommandArgs[0]+" -> "+invocation.CommandArgs[1],
				stderr,
				err,
			)
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
			fmt.Fprintln(stderr, "usage: gosherpa [--root <path>] callers <function-or-method> [--tests]")
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[0]

		result, err := sherpa.FindCallersWithOptions(root, target, sherpa.CallOptions{
			IncludeTests: invocation.IncludeTests,
		})
		if err != nil {
			return writeCommandError(invocation.JSON, root, "callers", target, stderr, err)
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
			return writeCommandError(invocation.JSON, root, "callees", target, stderr, err)
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
	case "symbol", "symbols", "search", "refs", "impact", "tests", "deps", "path", "paths", "callers", "callees", "explain":
		return true
	default:
		return false
	}
}

func supportsJSON(command string) bool {
	switch command {
	case "symbol", "symbols", "search", "refs", "impact", "tests", "deps", "path", "paths", "callers", "callees", "explain":
		return true
	default:
		return false
	}
}

func isBaseAwareInvocation(invocation cliInvocation) bool {
	return isImpactDiffInvocation(invocation) || isTestsAffectedInvocation(invocation)
}

func supportsLimitOption(command string) bool {
	return command == "search" || isPathCommand(command)
}

func isPathCommand(command string) bool {
	return command == "path" || command == "paths"
}

func supportsTestsOption(command string) bool {
	switch command {
	case "search", "callers", "explain":
		return true
	default:
		return false
	}
}

func isImpactDiffInvocation(invocation cliInvocation) bool {
	return invocation.Command == "impact" && len(invocation.CommandArgs) > 0 && invocation.CommandArgs[0] == "diff"
}

func isTestsAffectedInvocation(invocation cliInvocation) bool {
	return invocation.Command == "tests" && len(invocation.CommandArgs) > 0 && invocation.CommandArgs[0] == "affected"
}

func isImpactReportSubcommand(command string) bool {
	switch command {
	case "file", "package", "symbol":
		return true
	default:
		return false
	}
}

func analyzeImpactSubcommand(root string, kind string, target string) (impactengine.ImpactReport, error) {
	switch kind {
	case "file":
		return impactengine.AnalyzeFile(root, target)
	case "package":
		return impactengine.AnalyzePackage(root, target)
	case "symbol":
		return impactengine.AnalyzeSymbol(root, target)
	default:
		return impactengine.ImpactReport{}, fmt.Errorf("unknown impact subcommand: %s", kind)
	}
}

func formatImpactSubcommandReport(kind string, report impactengine.ImpactReport) string {
	switch kind {
	case "file":
		return impactengine.FormatFileReport(report)
	case "package":
		return impactengine.FormatPackageReport(report)
	case "symbol":
		return impactengine.FormatSymbolReport(report)
	default:
		return impactengine.FormatDiffReport(report)
	}
}

func writeJSON(stdout io.Writer, stderr io.Writer, value any) int {
	if err := encodeJSON(stdout, value); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitFailure
	}

	return exitSuccess
}

func encodeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(value)
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

func writeCommandError(jsonOutput bool, root string, command string, target string, stderr io.Writer, err error) int {
	if jsonOutput && writeTypedJSONError(stderr, root, command, target, err) {
		return exitFailure
	}

	fmt.Fprintln(stderr, "error:", err)
	return exitFailure
}

func writeTypedJSONError(stderr io.Writer, root string, command string, target string, err error) bool {
	var ambiguous *sherpa.AmbiguousTargetError
	if !errors.As(err, &ambiguous) {
		return false
	}

	response := newJSONErrorResponse(root, command, target, jsonErrorData{
		Code:       "ambiguous_target",
		Message:    err.Error(),
		Kind:       ambiguous.Kind,
		Target:     ambiguous.Target,
		Candidates: nonNilSlice(ambiguous.Candidates),
	})

	if encodeErr := encodeJSON(stderr, response); encodeErr != nil {
		fmt.Fprintln(stderr, "error:", encodeErr)
	}

	return true
}

func newJSONErrorResponse(root string, command string, target string, errorData jsonErrorData) jsonErrorResponse {
	var warnings []string
	modulePath, err := sherpa.ModulePath(root)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	return jsonErrorResponse{
		SchemaVersion: jsonSchemaVersion,
		Command:       command,
		Target:        target,
		Root:          root,
		ModulePath:    modulePath,
		Warnings:      nonNilSlice(warnings),
		Error:         errorData,
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

func impactDiffJSONResult(report impactengine.ImpactReport) impactengine.ImpactReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func impactDiffJSONDataFromReport(report impactengine.ImpactReport) impactDiffJSONData {
	return impactDiffJSONData{
		ChangedFiles:            report.ChangedFiles,
		ChangedPackages:         report.ChangedPackages,
		AffectedPackages:        report.AffectedPackages,
		AffectedSymbols:         report.AffectedSymbols,
		AffectedInterfaces:      report.AffectedInterfaces,
		AffectedImplementations: report.AffectedImplementations,
		AffectedTests:           report.AffectedTests,
		TestCommands:            report.TestCommands,
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

func testsAffectedJSONDataFromReport(report impactengine.ImpactReport) testsAffectedJSONData {
	return testsAffectedJSONData{
		AffectedTests: report.AffectedTests,
		Commands:      report.TestCommands,
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

func explainJSONResult(report explainengine.Report) explainengine.Report {
	report.References = nonNilSlice(report.References)
	report.Callers = nonNilSlice(report.Callers)
	report.Callees = nonNilSlice(report.Callees)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.RelatedTests = nonNilSlice(report.RelatedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func explainJSONDataFromReport(report explainengine.Report) explainJSONData {
	return explainJSONData{
		Target:                  report.Target,
		Symbol:                  report.Symbol,
		Purpose:                 report.Purpose,
		Risk:                    report.Risk,
		ArchitectureRole:        report.ArchitectureRole,
		References:              report.References,
		Callers:                 report.Callers,
		Callees:                 report.Callees,
		AffectedPackages:        report.AffectedPackages,
		AffectedInterfaces:      report.AffectedInterfaces,
		AffectedImplementations: report.AffectedImplementations,
		RelatedTests:            report.RelatedTests,
		TestCommands:            report.TestCommands,
		ReadingOrder:            report.ReadingOrder,
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
	fmt.Fprintln(writer, "  --json           machine-readable output for all commands")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "commands:")
	fmt.Fprintln(writer, "  explain <symbol> [--tests]")
	fmt.Fprintln(writer, "  symbols")
	fmt.Fprintln(writer, "  symbol <target>")
	fmt.Fprintln(writer, "  search <terms> [--kind <kind>] [--package <package>] [--tests] [--limit <n>]")
	fmt.Fprintln(writer, "  refs <name>")
	fmt.Fprintln(writer, "  impact <symbol-or-package>")
	fmt.Fprintln(writer, "  impact file <file>")
	fmt.Fprintln(writer, "  impact package <package>")
	fmt.Fprintln(writer, "  impact symbol <symbol>")
	fmt.Fprintln(writer, "  impact diff --base <ref>")
	fmt.Fprintln(writer, "  tests <symbol-or-package>")
	fmt.Fprintln(writer, "  tests affected --base <ref>")
	fmt.Fprintln(writer, "  deps <package>")
	fmt.Fprintln(writer, "  path <from> <to>")
	fmt.Fprintln(writer, "  paths <from> <to> [--limit <n>] [--max-depth <n>]")
	fmt.Fprintln(writer, "  callers <function-or-method> [--tests]")
	fmt.Fprintln(writer, "  callees <function-or-method>")
}

func printImpactUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gosherpa [--root <path>] impact <symbol-or-package>")
	fmt.Fprintln(writer, "       gosherpa [--root <path>] impact file <file>")
	fmt.Fprintln(writer, "       gosherpa [--root <path>] impact package <package>")
	fmt.Fprintln(writer, "       gosherpa [--root <path>] impact symbol <symbol>")
	fmt.Fprintln(writer, "       gosherpa [--root <path>] impact diff --base <ref>")
}

func printImpactDiffUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gosherpa [--root <path>] impact diff --base <ref>")
}

func printImpactSubcommandUsage(writer io.Writer, kind string) {
	switch kind {
	case "file":
		fmt.Fprintln(writer, "usage: gosherpa [--root <path>] impact file <file>")
	case "package":
		fmt.Fprintln(writer, "usage: gosherpa [--root <path>] impact package <package>")
	case "symbol":
		fmt.Fprintln(writer, "usage: gosherpa [--root <path>] impact symbol <symbol>")
	default:
		printImpactUsage(writer)
	}
}

func printTestsUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gosherpa [--root <path>] tests <symbol-or-package>")
	fmt.Fprintln(writer, "       gosherpa [--root <path>] tests affected --base <ref>")
}

func printTestsAffectedUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gosherpa [--root <path>] tests affected --base <ref>")
}
