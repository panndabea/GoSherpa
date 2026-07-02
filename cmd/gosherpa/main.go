package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

const (
	exitSuccess       = 0
	exitFailure       = 1
	exitUsage         = 2
	jsonSchemaVersion = 1

	analysisModeAST  = agentcontext.AnalysisModeAST
	analysisModeDiff = agentcontext.AnalysisModeDiff
	confidenceMedium = agentcontext.ConfidenceMedium
	confidenceLow    = agentcontext.ConfidenceLow
)

type cliInvocation struct {
	Root               string
	Command            string
	CommandArgs        []string
	JSON               bool
	CallPathLimit      int
	CallPathMaxDepth   int
	HasCallPathOption  bool
	HasLimitOption     bool
	HasMaxDepthOption  bool
	BaseRef            string
	HasBaseOption      bool
	IncludeTests       bool
	HasTestsOption     bool
	ShowContext        bool
	HasContextOption   bool
	BuildTags          []string
	HasTagsOption      bool
	KindFilter         string
	SearchKind         sherpa.SymbolKind
	ReferenceKind      sherpa.ReferenceKind
	HasKindOption      bool
	TestScope          sherpa.TestScope
	HasTestScopeOption bool
	SearchPackage      string
	HasPackageOption   bool
	ContextLimits      agentcontext.LimitOptions
	HasContextLimit    bool
	HasSourceRadius    bool
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
	AnalysisMode string             `json:"analysisMode"`
	Confidence   string             `json:"confidence"`
	Limitations  []string           `json:"limitations"`
	References   []sherpa.Reference `json:"references"`
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
	AnalysisMode          string                     `json:"analysisMode"`
	Confidence            string                     `json:"confidence"`
	Limitations           []string                   `json:"limitations"`
	Kind                  sherpa.ImpactKind          `json:"kind"`
	References            []sherpa.Reference         `json:"references"`
	ReferenceAnalysisMode string                     `json:"referenceAnalysisMode,omitempty"`
	Callers               []sherpa.Caller            `json:"callers"`
	CallAnalysisMode      string                     `json:"callAnalysisMode,omitempty"`
	Dependencies          sherpa.PackageDependencies `json:"dependencies"`
	Packages              []string                   `json:"packages"`
	RelatedTests          []sherpa.RelatedTest       `json:"relatedTests"`
	TestCommands          []string                   `json:"testCommands"`
	TestPlan              sherpa.TestPlan            `json:"testPlan"`
}

type impactDiffJSONData struct {
	AnalysisMode            string                     `json:"analysisMode"`
	Confidence              string                     `json:"confidence"`
	Limitations             []string                   `json:"limitations"`
	ChangedFiles            []string                   `json:"changedFiles"`
	ChangedPackages         []string                   `json:"changedPackages"`
	AffectedPackages        []string                   `json:"affectedPackages"`
	AffectedSymbols         []string                   `json:"affectedSymbols"`
	ReferenceAnalysisMode   string                     `json:"referenceAnalysisMode,omitempty"`
	CallAnalysisMode        string                     `json:"callAnalysisMode,omitempty"`
	AffectedInterfaces      []string                   `json:"affectedInterfaces"`
	AffectedImplementations []string                   `json:"affectedImplementations"`
	InterfaceAnalysisMode   string                     `json:"interfaceAnalysisMode,omitempty"`
	AffectedTests           []impactengine.RelatedTest `json:"affectedTests"`
	TestCommands            []string                   `json:"testCommands"`
	TestPlan                sherpa.TestPlan            `json:"testPlan"`
}

type testsJSONData struct {
	AnalysisMode string                `json:"analysisMode"`
	Confidence   string                `json:"confidence"`
	Limitations  []string              `json:"limitations"`
	Kind         sherpa.TestTargetKind `json:"kind"`
	Scope        sherpa.TestScope      `json:"scope"`
	Tests        []sherpa.RelatedTest  `json:"tests"`
	Commands     []string              `json:"commands"`
	TestPlan     sherpa.TestPlan       `json:"testPlan"`
}

type testsAffectedJSONData struct {
	AnalysisMode  string                     `json:"analysisMode"`
	Confidence    string                     `json:"confidence"`
	Limitations   []string                   `json:"limitations"`
	AffectedTests []impactengine.RelatedTest `json:"affectedTests"`
	Commands      []string                   `json:"commands"`
	TestPlan      sherpa.TestPlan            `json:"testPlan"`
}

type dependenciesJSONData struct {
	Package string   `json:"package"`
	Imports []string `json:"imports"`
	UsedBy  []string `json:"usedBy"`
}

type implementersJSONData struct {
	AnalysisMode string                     `json:"analysisMode"`
	Confidence   string                     `json:"confidence"`
	Limitations  []string                   `json:"limitations"`
	Implementers []impactengine.Implementer `json:"implementers"`
}

type interfacesJSONData struct {
	AnalysisMode string                            `json:"analysisMode"`
	Confidence   string                            `json:"confidence"`
	Limitations  []string                          `json:"limitations"`
	Interfaces   []impactengine.SatisfiedInterface `json:"interfaces"`
}

type callersJSONData struct {
	AnalysisMode string          `json:"analysisMode"`
	Confidence   string          `json:"confidence"`
	Limitations  []string        `json:"limitations"`
	Callers      []sherpa.Caller `json:"callers"`
}

type calleesJSONData struct {
	AnalysisMode string          `json:"analysisMode"`
	Confidence   string          `json:"confidence"`
	Limitations  []string        `json:"limitations"`
	Callees      []sherpa.Callee `json:"callees"`
}

type callPathsJSONData struct {
	AnalysisMode string            `json:"analysisMode"`
	Confidence   string            `json:"confidence"`
	Limitations  []string          `json:"limitations"`
	From         string            `json:"from"`
	To           string            `json:"to"`
	Paths        []sherpa.CallPath `json:"paths"`
}

type entrypointsJSONData struct {
	AnalysisMode string              `json:"analysisMode"`
	Confidence   string              `json:"confidence"`
	Limitations  []string            `json:"limitations"`
	EntryPoints  []sherpa.EntryPoint `json:"entrypoints"`
}

type explainJSONData struct {
	Target                  string                         `json:"target"`
	AnalysisMode            string                         `json:"analysisMode"`
	Confidence              string                         `json:"confidence"`
	Limitations             []string                       `json:"limitations"`
	Symbol                  sherpa.Symbol                  `json:"symbol"`
	Purpose                 string                         `json:"purpose"`
	Risk                    explainengine.RiskSummary      `json:"risk"`
	ArchitectureRole        explainengine.ArchitectureRole `json:"architectureRole"`
	References              []sherpa.Reference             `json:"references"`
	ReferenceAnalysisMode   string                         `json:"referenceAnalysisMode,omitempty"`
	Callers                 []sherpa.Caller                `json:"callers"`
	Callees                 []sherpa.Callee                `json:"callees"`
	CallAnalysisMode        string                         `json:"callAnalysisMode"`
	AffectedPackages        []string                       `json:"affectedPackages"`
	AffectedInterfaces      []string                       `json:"affectedInterfaces"`
	AffectedImplementations []string                       `json:"affectedImplementations"`
	InterfaceAnalysisMode   string                         `json:"interfaceAnalysisMode,omitempty"`
	RelatedTests            []sherpa.RelatedTest           `json:"relatedTests"`
	TestCommands            []string                       `json:"testCommands"`
	TestPlan                sherpa.TestPlan                `json:"testPlan"`
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

		if arg == "--context" {
			invocation.ShowContext = true
			invocation.HasContextOption = true
			continue
		}

		if arg == "--tags" {
			value, err := parseStringFlagValue("--tags", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			tags, err := parseBuildTagsFlag("--tags", value)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.BuildTags = tags
			invocation.HasTagsOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--tags=") {
			tags, err := parseBuildTagsFlag("--tags", strings.TrimPrefix(arg, "--tags="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.BuildTags = tags
			invocation.HasTagsOption = true
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
			value, err := parseStringFlagValue("--kind", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.KindFilter = value
			invocation.HasKindOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--kind=") {
			value, err := parseStringFlag("--kind", strings.TrimPrefix(arg, "--kind="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.KindFilter = value
			invocation.HasKindOption = true
			continue
		}

		if arg == "--scope" {
			value, err := parseStringFlagValue("--scope", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			scope, err := parseTestScopeFlag("--scope", value)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.TestScope = scope
			invocation.HasTestScopeOption = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--scope=") {
			scope, err := parseTestScopeFlag("--scope", strings.TrimPrefix(arg, "--scope="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.TestScope = scope
			invocation.HasTestScopeOption = true
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

		if arg == "--max-files" {
			value, err := parsePositiveFlagValue("--max-files", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxFiles = value
			invocation.HasContextLimit = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--max-files=") {
			value, err := parsePositiveInteger("--max-files", strings.TrimPrefix(arg, "--max-files="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxFiles = value
			invocation.HasContextLimit = true
			continue
		}

		if arg == "--max-references" {
			value, err := parsePositiveFlagValue("--max-references", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxReferences = value
			invocation.HasContextLimit = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--max-references=") {
			value, err := parsePositiveInteger("--max-references", strings.TrimPrefix(arg, "--max-references="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxReferences = value
			invocation.HasContextLimit = true
			continue
		}

		if arg == "--max-symbols" {
			value, err := parsePositiveFlagValue("--max-symbols", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxSymbols = value
			invocation.HasContextLimit = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--max-symbols=") {
			value, err := parsePositiveInteger("--max-symbols", strings.TrimPrefix(arg, "--max-symbols="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxSymbols = value
			invocation.HasContextLimit = true
			continue
		}

		if arg == "--max-tests" {
			value, err := parsePositiveFlagValue("--max-tests", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxTests = value
			invocation.HasContextLimit = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--max-tests=") {
			value, err := parsePositiveInteger("--max-tests", strings.TrimPrefix(arg, "--max-tests="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxTests = value
			invocation.HasContextLimit = true
			continue
		}

		if arg == "--max-bytes" {
			value, err := parsePositiveFlagValue("--max-bytes", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxBytes = value
			invocation.HasContextLimit = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--max-bytes=") {
			value, err := parsePositiveInteger("--max-bytes", strings.TrimPrefix(arg, "--max-bytes="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.MaxBytes = value
			invocation.HasContextLimit = true
			continue
		}

		if arg == "--source-radius" {
			value, err := parseNonNegativeFlagValue("--source-radius", args, i)
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.SourceRadius = agentcontext.NewSourceRadius(value)
			invocation.HasContextLimit = true
			invocation.HasSourceRadius = true
			i++
			continue
		}

		if strings.HasPrefix(arg, "--source-radius=") {
			value, err := parseNonNegativeInteger("--source-radius", strings.TrimPrefix(arg, "--source-radius="))
			if err != nil {
				return cliInvocation{}, err
			}

			invocation.ContextLimits.SourceRadius = agentcontext.NewSourceRadius(value)
			invocation.HasContextLimit = true
			invocation.HasSourceRadius = true
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

	if invocation.HasKindOption {
		if err := parseKindFilter(&invocation); err != nil {
			return cliInvocation{}, err
		}
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

func parseBuildTagsFlag(flag string, value string) ([]string, error) {
	tags := semantics.NormalizeBuildTags([]string{value})
	if len(tags) == 0 {
		return nil, fmt.Errorf("missing value for %s", flag)
	}

	return tags, nil
}

func parsePositiveFlagValue(flag string, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return 0, fmt.Errorf("missing value for %s", flag)
	}

	return parsePositiveInteger(flag, args[index+1])
}

func parseNonNegativeFlagValue(flag string, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return 0, fmt.Errorf("missing value for %s", flag)
	}

	return parseNonNegativeInteger(flag, args[index+1])
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

func parseNonNegativeInteger(flag string, value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("missing value for %s", flag)
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid value for %s: %s", flag, trimmed)
	}

	return parsed, nil
}

func parseKindFilter(invocation *cliInvocation) error {
	switch invocation.Command {
	case "search", "symbols":
		kind, err := parseSymbolKindFlag("--kind", invocation.KindFilter)
		if err != nil {
			return err
		}
		invocation.SearchKind = kind
	case "refs":
		kind, err := parseReferenceKindFlag("--kind", invocation.KindFilter)
		if err != nil {
			return err
		}
		invocation.ReferenceKind = kind
	}

	return nil
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

func parseReferenceKindFlag(flag string, value string) (sherpa.ReferenceKind, error) {
	trimmed, err := parseStringFlag(flag, value)
	if err != nil {
		return "", err
	}

	if kind, ok := sherpa.ParseReferenceKind(trimmed); ok {
		return kind, nil
	}

	return "", fmt.Errorf("invalid value for %s: %s", flag, trimmed)
}

func parseTestScopeFlag(flag string, value string) (sherpa.TestScope, error) {
	trimmed, err := parseStringFlag(flag, value)
	if err != nil {
		return "", err
	}

	if scope, ok := sherpa.ParseTestScope(trimmed); ok {
		return scope, nil
	}

	return "", fmt.Errorf("invalid value for %s: %s", flag, trimmed)
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

	if invocation.HasMaxDepthOption && !supportsMaxDepthOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --max-depth is only supported by path commands")
		return exitUsage
	}

	if invocation.HasPackageOption && !supportsPackageOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --package is only supported by search and symbols")
		return exitUsage
	}

	if invocation.HasContextLimit && !supportsContextLimitOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --max-files, --max-references, --max-symbols, --max-tests, --max-bytes, and --source-radius are only supported by context")
		return exitUsage
	}

	if err := validateContextLimitOptions(invocation); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}

	if invocation.HasKindOption && !supportsKindOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --kind is only supported by search, symbols, and refs")
		return exitUsage
	}

	if invocation.HasTestScopeOption && !isScopedTestsInvocation(invocation) {
		fmt.Fprintln(stderr, "error: --scope is only supported by tests <symbol-or-package>")
		return exitUsage
	}

	if invocation.HasBaseOption && !isBaseAwareInvocation(invocation) {
		fmt.Fprintln(stderr, "error: --base is only supported by context diff, impact diff, and tests affected")
		return exitUsage
	}

	if invocation.HasTestsOption && !supportsTestsOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --tests is only supported by symbols, search, entrypoints, callers, explain, and context")
		return exitUsage
	}

	if invocation.HasContextOption && invocation.JSON {
		fmt.Fprintln(stderr, "error: --context is only supported for human output")
		return exitUsage
	}

	if invocation.HasContextOption && knownCommand(invocation.Command) && !supportsContextOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --context is only supported by symbol, refs, callers, and callees")
		return exitUsage
	}

	if invocation.HasTagsOption && knownCommand(invocation.Command) && !supportsTagsOption(invocation) {
		fmt.Fprintln(stderr, "error: --tags is only supported by refs, entrypoints, callers, callees, explain, context, impact, tests affected, implementers, interfaces, pr, and doctor")
		return exitUsage
	}

	if invocation.JSON && knownCommand(invocation.Command) && !supportsJSON(invocation.Command) {
		fmt.Fprintln(stderr, "error: --json is only supported by known commands")
		return exitUsage
	}

	spec, ok := commandSpecFor(invocation.Command)
	if !ok {
		fmt.Fprintln(stderr, "unknown command:", invocation.Command)
		printUsage(stderr)
		return exitUsage
	}

	return spec.Handler(invocation, stdout, stderr)
}

func knownCommand(command string) bool {
	_, ok := commandSpecFor(command)
	return ok
}

func supportsJSON(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.JSON
}

func isBaseAwareInvocation(invocation cliInvocation) bool {
	spec, ok := commandSpecFor(invocation.Command)
	if !ok || spec.BaseWhen == nil {
		return false
	}

	return spec.BaseWhen(invocation)
}

func supportsLimitOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.Limit
}

func supportsMaxDepthOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.MaxDepth
}

func supportsPackageOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.Package
}

func supportsKindOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.Kind
}

func supportsContextLimitOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.ContextLimits
}

func supportsTestsOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.Tests
}

func supportsContextOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.Context
}

func supportsTagsOption(invocation cliInvocation) bool {
	spec, ok := commandSpecFor(invocation.Command)
	if !ok {
		return false
	}
	if spec.Tags {
		return true
	}
	if spec.TagsWhen != nil {
		return spec.TagsWhen(invocation)
	}

	return false
}

func validateContextLimitOptions(invocation cliInvocation) error {
	if !invocation.HasContextLimit || invocation.Command != "context" || len(invocation.CommandArgs) == 0 {
		return nil
	}

	var unsupported []string
	switch invocation.CommandArgs[0] {
	case "symbol":
		if invocation.ContextLimits.MaxFiles > 0 {
			unsupported = append(unsupported, "--max-files")
		}
		if invocation.ContextLimits.MaxSymbols > 0 {
			unsupported = append(unsupported, "--max-symbols")
		}
	case "file":
		if invocation.ContextLimits.MaxFiles > 0 {
			unsupported = append(unsupported, "--max-files")
		}
		if invocation.ContextLimits.MaxReferences > 0 {
			unsupported = append(unsupported, "--max-references")
		}
	case "package":
		if invocation.ContextLimits.MaxReferences > 0 {
			unsupported = append(unsupported, "--max-references")
		}
	case "diff":
		if invocation.ContextLimits.MaxReferences > 0 {
			unsupported = append(unsupported, "--max-references")
		}
		if invocation.HasSourceRadius {
			unsupported = append(unsupported, "--source-radius")
		}
	}

	if len(unsupported) == 0 {
		return nil
	}

	return fmt.Errorf("unsupported context option for context %s: %s", invocation.CommandArgs[0], strings.Join(unsupported, ", "))
}

func isImpactDiffInvocation(invocation cliInvocation) bool {
	return invocation.Command == "impact" && len(invocation.CommandArgs) > 0 && invocation.CommandArgs[0] == "diff"
}

func isContextDiffInvocation(invocation cliInvocation) bool {
	return invocation.Command == "context" && len(invocation.CommandArgs) > 0 && invocation.CommandArgs[0] == "diff"
}

func isTestsAffectedInvocation(invocation cliInvocation) bool {
	return invocation.Command == "tests" && len(invocation.CommandArgs) > 0 && invocation.CommandArgs[0] == "affected"
}

func isScopedTestsInvocation(invocation cliInvocation) bool {
	return invocation.Command == "tests" && !isTestsAffectedInvocation(invocation)
}

func isPRInvocation(invocation cliInvocation) bool {
	return invocation.Command == "pr"
}

func isImpactReportSubcommand(command string) bool {
	switch command {
	case "file", "package", "symbol":
		return true
	default:
		return false
	}
}

func analyzeImpactSubcommand(root string, kind string, target string, buildTags []string) (impactengine.ImpactReport, error) {
	options := impactengine.AnalyzerOptions{BuildTags: buildTags}
	switch kind {
	case "file":
		return impactengine.AnalyzeFileWithOptions(root, target, options)
	case "package":
		return impactengine.AnalyzePackageWithOptions(root, target, options)
	case "symbol":
		return impactengine.AnalyzeSymbolWithOptions(root, target, options)
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
	result.ReferenceAnalysisMode = strings.TrimSpace(result.ReferenceAnalysisMode)
	result.Callers = nonNilSlice(result.Callers)
	result.CallAnalysisMode = strings.TrimSpace(result.CallAnalysisMode)
	result.Dependencies.Imports = nonNilSlice(result.Dependencies.Imports)
	result.Dependencies.UsedBy = nonNilSlice(result.Dependencies.UsedBy)
	result.Packages = nonNilSlice(result.Packages)
	result.RelatedTests = nonNilSlice(result.RelatedTests)
	result.TestCommands = nonNilSlice(result.TestCommands)
	result.TestPlan = sherpa.NormalizeTestPlan(result.TestPlan)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func impactJSONDataFromResult(result sherpa.ImpactResult) impactJSONData {
	analysisMode := impactResultAnalysisMode(result)

	return impactJSONData{
		AnalysisMode:          analysisMode,
		Confidence:            jsonConfidence(result.Warnings, analysisMode, result.ReferenceAnalysisMode, result.CallAnalysisMode),
		Limitations:           impactBundleLimitations(analysisMode, result.ReferenceAnalysisMode, result.CallAnalysisMode),
		Kind:                  result.Kind,
		References:            result.References,
		ReferenceAnalysisMode: result.ReferenceAnalysisMode,
		Callers:               result.Callers,
		CallAnalysisMode:      result.CallAnalysisMode,
		Dependencies:          result.Dependencies,
		Packages:              result.Packages,
		RelatedTests:          result.RelatedTests,
		TestCommands:          result.TestCommands,
		TestPlan:              result.TestPlan,
	}
}

func impactResultAnalysisMode(result sherpa.ImpactResult) string {
	return bundleAnalysisMode(result.ReferenceAnalysisMode, result.CallAnalysisMode)
}

func impactDiffJSONResult(report impactengine.ImpactReport) impactengine.ImpactReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.ReferenceAnalysisMode = strings.TrimSpace(report.ReferenceAnalysisMode)
	report.CallAnalysisMode = strings.TrimSpace(report.CallAnalysisMode)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func impactDiffJSONDataFromReport(report impactengine.ImpactReport, analysisMode string) impactDiffJSONData {
	analysisMode = impactReportAnalysisMode(report, analysisMode)

	return impactDiffJSONData{
		AnalysisMode:            analysisMode,
		Confidence:              jsonConfidence(report.Warnings, analysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode),
		Limitations:             impactBundleLimitations(analysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode),
		ChangedFiles:            report.ChangedFiles,
		ChangedPackages:         report.ChangedPackages,
		AffectedPackages:        report.AffectedPackages,
		AffectedSymbols:         report.AffectedSymbols,
		ReferenceAnalysisMode:   report.ReferenceAnalysisMode,
		CallAnalysisMode:        report.CallAnalysisMode,
		AffectedInterfaces:      report.AffectedInterfaces,
		AffectedImplementations: report.AffectedImplementations,
		InterfaceAnalysisMode:   report.InterfaceAnalysisMode,
		AffectedTests:           report.AffectedTests,
		TestCommands:            report.TestCommands,
		TestPlan:                report.TestPlan,
	}
}

func impactReportAnalysisMode(report impactengine.ImpactReport, fallback string) string {
	if fallback == analysisModeDiff {
		return fallback
	}

	return bundleAnalysisMode(report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode)
}

func testsJSONResult(result sherpa.TestsResult) sherpa.TestsResult {
	result.Tests = nonNilSlice(result.Tests)
	result.Commands = nonNilSlice(result.Commands)
	result.TestPlan = sherpa.NormalizeTestPlan(result.TestPlan)

	return result
}

func testsJSONDataFromResult(result sherpa.TestsResult) testsJSONData {
	return testsJSONData{
		AnalysisMode: analysisModeAST,
		Confidence:   jsonConfidence(nil, analysisModeAST),
		Limitations:  testLimitations(analysisModeAST),
		Kind:         result.Kind,
		Scope:        result.Scope,
		Tests:        result.Tests,
		Commands:     result.Commands,
		TestPlan:     result.TestPlan,
	}
}

func testsAffectedJSONDataFromReport(report impactengine.ImpactReport) testsAffectedJSONData {
	return testsAffectedJSONData{
		AnalysisMode:  analysisModeDiff,
		Confidence:    jsonConfidence(report.Warnings, analysisModeDiff),
		Limitations:   testLimitations(analysisModeDiff),
		AffectedTests: report.AffectedTests,
		Commands:      report.TestCommands,
		TestPlan:      report.TestPlan,
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

func implementersJSONResult(result impactengine.ImplementersResult) impactengine.ImplementersResult {
	result.Implementers = nonNilSlice(result.Implementers)
	result.Warnings = nonNilSlice(result.Warnings)
	if strings.TrimSpace(result.AnalysisMode) == "" {
		result.AnalysisMode = impactengine.InterfaceAnalysisModeASTFallback
	}

	return result
}

func implementersJSONDataFromResult(result impactengine.ImplementersResult) implementersJSONData {
	return implementersJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  interfaceLimitations(result.AnalysisMode),
		Implementers: result.Implementers,
	}
}

func interfacesJSONResult(result impactengine.InterfacesResult) impactengine.InterfacesResult {
	result.Interfaces = nonNilSlice(result.Interfaces)
	result.Warnings = nonNilSlice(result.Warnings)
	if strings.TrimSpace(result.AnalysisMode) == "" {
		result.AnalysisMode = impactengine.InterfaceAnalysisModeASTFallback
	}

	return result
}

func interfacesJSONDataFromResult(result impactengine.InterfacesResult) interfacesJSONData {
	return interfacesJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  interfaceLimitations(result.AnalysisMode),
		Interfaces:   result.Interfaces,
	}
}

func callersJSONResult(result sherpa.CallersResult) sherpa.CallersResult {
	result.Callers = nonNilSlice(result.Callers)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func callersJSONDataFromResult(result sherpa.CallersResult) callersJSONData {
	return callersJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  callLimitations(result.AnalysisMode),
		Callers:      result.Callers,
	}
}

func calleesJSONResult(result sherpa.CalleesResult) sherpa.CalleesResult {
	result.Callees = nonNilSlice(result.Callees)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func calleesJSONDataFromResult(result sherpa.CalleesResult) calleesJSONData {
	return calleesJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  callLimitations(result.AnalysisMode),
		Callees:      result.Callees,
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
		AnalysisMode: analysisModeAST,
		Confidence:   jsonConfidence(nil, analysisModeAST),
		Limitations:  callPathLimitations(),
		From:         result.From,
		To:           result.To,
		Paths:        result.Paths,
	}
}

func entrypointsJSONResult(result sherpa.EntryPointsResult) sherpa.EntryPointsResult {
	result.EntryPoints = nonNilSlice(result.EntryPoints)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func entrypointsJSONDataFromResult(result sherpa.EntryPointsResult) entrypointsJSONData {
	return entrypointsJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  entrypointLimitations(result.AnalysisMode),
		EntryPoints:  result.EntryPoints,
	}
}

func contextSymbolJSONResult(report agentcontext.Report) agentcontext.Report {
	report.References = nonNilSlice(report.References)
	report.Callers = nonNilSlice(report.Callers)
	report.Callees = nonNilSlice(report.Callees)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.RelatedTests = nonNilSlice(report.RelatedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.SourceContext.Lines = nonNilSlice(report.SourceContext.Lines)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func contextFileJSONResult(report agentcontext.FileReport) agentcontext.FileReport {
	report.Symbols = nonNilSlice(report.Symbols)
	report.SourceContexts = nonNilSlice(report.SourceContexts)
	for i := range report.SourceContexts {
		report.SourceContexts[i].Lines = nonNilSlice(report.SourceContexts[i].Lines)
	}
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func contextPackageJSONResult(report agentcontext.PackageReport) agentcontext.PackageReport {
	report.Files = nonNilSlice(report.Files)
	report.Symbols = nonNilSlice(report.Symbols)
	report.SourceContexts = nonNilSlice(report.SourceContexts)
	for i := range report.SourceContexts {
		report.SourceContexts[i].Lines = nonNilSlice(report.SourceContexts[i].Lines)
	}
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func contextDiffJSONResult(report agentcontext.DiffReport) agentcontext.DiffReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func explainJSONResult(report explainengine.Report) explainengine.Report {
	report.References = nonNilSlice(report.References)
	report.ReferenceAnalysisMode = strings.TrimSpace(report.ReferenceAnalysisMode)
	report.Callers = nonNilSlice(report.Callers)
	report.Callees = nonNilSlice(report.Callees)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.RelatedTests = nonNilSlice(report.RelatedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func explainJSONDataFromReport(report explainengine.Report) explainJSONData {
	analysisMode := explainAnalysisMode(report)

	return explainJSONData{
		Target:                  report.Target,
		AnalysisMode:            analysisMode,
		Confidence:              jsonConfidence(report.Warnings, analysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode),
		Limitations:             explainLimitations(report.ReferenceAnalysisMode, report.CallAnalysisMode),
		Symbol:                  report.Symbol,
		Purpose:                 report.Purpose,
		Risk:                    report.Risk,
		ArchitectureRole:        report.ArchitectureRole,
		References:              report.References,
		ReferenceAnalysisMode:   report.ReferenceAnalysisMode,
		Callers:                 report.Callers,
		Callees:                 report.Callees,
		CallAnalysisMode:        report.CallAnalysisMode,
		AffectedPackages:        report.AffectedPackages,
		AffectedInterfaces:      report.AffectedInterfaces,
		AffectedImplementations: report.AffectedImplementations,
		InterfaceAnalysisMode:   report.InterfaceAnalysisMode,
		RelatedTests:            report.RelatedTests,
		TestCommands:            report.TestCommands,
		TestPlan:                report.TestPlan,
		ReadingOrder:            report.ReadingOrder,
	}
}

func explainAnalysisMode(report explainengine.Report) string {
	return bundleAnalysisMode(report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode)
}

func bundleAnalysisMode(analysisModes ...string) string {
	for _, mode := range analysisModes {
		if mode == sherpa.ReferenceAnalysisModeTypechecked ||
			mode == sherpa.CallAnalysisModeTypechecked ||
			mode == impactengine.InterfaceAnalysisModeTypechecked {
			return agentcontext.AnalysisModeTypecheckedAST
		}
	}

	return analysisModeAST
}

func jsonConfidence(warnings []string, analysisModes ...string) string {
	if len(warnings) > 0 {
		return confidenceLow
	}

	for _, mode := range analysisModes {
		if mode == sherpa.CallAnalysisModeASTFallback ||
			mode == sherpa.ReferenceAnalysisModeASTFallback ||
			mode == impactengine.InterfaceAnalysisModeASTFallback {
			return confidenceLow
		}
	}

	return confidenceMedium
}

func referenceLimitations(analysisMode string) []string {
	return []string{
		referenceAnalysisLimitation(analysisMode),
		"References are repository-local and may not include generated or build-tagged code outside the loaded package set.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
	}
}

func referenceAnalysisLimitation(analysisMode string) string {
	switch analysisMode {
	case sherpa.ReferenceAnalysisModeTypechecked:
		return "Reference analysis used typechecked package loading where available."
	case sherpa.ReferenceAnalysisModeASTFallback:
		return "Reference analysis used AST fallback because typechecked loading was unavailable."
	default:
		return "Reference analysis used syntax plus local type information."
	}
}

func callLimitations(analysisMode string) []string {
	return []string{
		callAnalysisLimitation(analysisMode),
		"Call graph results are repository-local.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
		"Imported-package receiver calls may be incomplete.",
	}
}

func callAnalysisLimitation(analysisMode string) string {
	switch analysisMode {
	case sherpa.CallAnalysisModeTypechecked:
		return "Call analysis used typechecked package loading where available."
	case sherpa.CallAnalysisModeASTFallback:
		return "Call analysis used AST fallback because typechecked loading was unavailable."
	default:
		return "Call analysis used syntax plus local type information."
	}
}

func callPathLimitations() []string {
	return []string{
		"Path analysis uses repository-local call graph edges from syntax plus local type information.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
		"Only bounded shortest paths requested by the command are returned.",
	}
}

func entrypointLimitations(analysisMode string) []string {
	limitations := callLimitations(analysisMode)
	limitations = append(limitations,
		"Entry point classification is heuristic: main functions, test functions, exported functions, and functions with no local callers.",
		"Framework-specific entrypoints such as HTTP routers and CLI command handlers are not inferred yet.",
	)

	return limitations
}

func explainLimitations(referenceAnalysisMode string, callAnalysisMode string) []string {
	limitations := []string{
		"Explain analysis combines symbol, reference, impact, test, and call signals from local repository analysis.",
		"Purpose, risk, architecture role, and reading order are deterministic heuristics.",
	}
	if strings.TrimSpace(referenceAnalysisMode) != "" {
		limitations = append(limitations, referenceAnalysisLimitation(referenceAnalysisMode))
	}
	limitations = append(limitations, callLimitations(callAnalysisMode)...)
	limitations = append(limitations, testLimitations(analysisModeAST)...)

	return limitations
}

func impactBundleLimitations(analysisMode string, referenceAnalysisMode string, callAnalysisMode string) []string {
	limitations := impactLimitations(analysisMode)
	if strings.TrimSpace(referenceAnalysisMode) != "" {
		limitations = append(limitations, referenceAnalysisLimitation(referenceAnalysisMode))
	}
	if strings.TrimSpace(callAnalysisMode) != "" {
		limitations = append(limitations, callAnalysisLimitation(callAnalysisMode))
	}

	return limitations
}

func impactLimitations(analysisMode string) []string {
	if analysisMode == analysisModeDiff {
		return []string{
			"Diff impact is based on git changed files and hunk-level changed symbol extraction.",
			"Impact analysis uses syntax plus local package dependency and interface signals.",
			"Statement-level semantic consequences are not fully inferred.",
			"Dynamic dispatch, reflection, and function values are not resolved.",
		}
	}

	return []string{
		"Impact analysis uses syntax plus local package dependency and interface signals.",
		"Symbol impact includes references and conservative caller-package propagation.",
		"Interface impact uses typechecked method sets when package loading succeeds and AST fallback otherwise.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
	}
}

func testLimitations(analysisMode string) []string {
	if analysisMode == analysisModeDiff {
		return []string{
			"Affected test planning is based on changed packages, affected packages, and syntactic test references.",
			"Literal t.Run subtest names are extracted; dynamic table-test names may be incomplete.",
			"Fallback commands are package-level when direct test functions are not known.",
		}
	}

	return []string{
		"Test discovery uses direct references, same-package tests, and literal t.Run subtest names.",
		"Dynamic table-test names may be incomplete.",
		"Fallback commands are package-level when direct test functions are not known.",
	}
}

func interfaceLimitations(analysisMode string) []string {
	switch analysisMode {
	case impactengine.InterfaceAnalysisModeTypechecked:
		return []string{
			"Interface analysis used typechecked method sets from repository-local packages.",
			"External implementations outside the repository are not reported.",
			"Build tags follow the default Go package loading environment.",
		}
	case impactengine.InterfaceAnalysisModeASTFallback:
		return []string{
			"Interface analysis used AST fallback because typechecked loading was unavailable.",
			"Embedded local interfaces are expanded, but alias, build-tag, and generic edge cases may be incomplete.",
			"External implementations outside the repository are not reported.",
		}
	default:
		return []string{
			"Interface analysis uses local method-set matching.",
			"External implementations outside the repository are not reported.",
		}
	}
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}

func referencePositions(refs []sherpa.Reference) []sherpa.Position {
	positions := make([]sherpa.Position, 0, len(refs))
	for _, ref := range refs {
		positions = append(positions, ref.Position)
	}

	return positions
}

func callerPositions(callers []sherpa.Caller) []sherpa.Position {
	positions := make([]sherpa.Position, 0, len(callers))
	for _, caller := range callers {
		positions = append(positions, caller.Position)
	}

	return positions
}

func calleePositions(callees []sherpa.Callee) []sherpa.Position {
	positions := make([]sherpa.Position, 0, len(callees))
	for _, callee := range callees {
		positions = append(positions, callee.Position)
	}

	return positions
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
	fmt.Fprintln(writer, "  --tags <list>    build tags for semantic package loading")
	fmt.Fprintln(writer, "  --json           machine-readable output for all commands")
	fmt.Fprintln(writer, "  --context        show source context for supported human output")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "commands:")
	for _, spec := range commandSpecs {
		for _, usage := range spec.Usage {
			fmt.Fprintln(writer, "  "+usage)
		}
	}
}

func printUsageLines(writer io.Writer, usageLines []string) {
	for i, usage := range usageLines {
		prefix := "usage: "
		if i > 0 {
			prefix = "       "
		}

		fmt.Fprintln(writer, prefix+"gosherpa [--root <path>] "+usage)
	}
}

func printCommandUsage(writer io.Writer, usage string) {
	printUsageLines(writer, []string{usage})
}

func printContextUsage(writer io.Writer) {
	printUsageLines(writer, contextUsageLines)
}

func printContextSymbolUsage(writer io.Writer) {
	printCommandUsage(writer, contextSymbolUsageLine)
}

func printContextFileUsage(writer io.Writer) {
	printCommandUsage(writer, contextFileUsageLine)
}

func printContextPackageUsage(writer io.Writer) {
	printCommandUsage(writer, contextPackageUsageLine)
}

func printContextDiffUsage(writer io.Writer) {
	printCommandUsage(writer, contextDiffUsageLine)
}

func printImpactUsage(writer io.Writer) {
	printUsageLines(writer, impactUsageLines)
}

func printImpactDiffUsage(writer io.Writer) {
	printCommandUsage(writer, impactDiffUsageLine)
}

func printPRUsage(writer io.Writer) {
	printCommandUsage(writer, prUsageLine)
}

func printDoctorUsage(writer io.Writer) {
	printCommandUsage(writer, doctorUsageLine)
}

func printImpactSubcommandUsage(writer io.Writer, kind string) {
	switch kind {
	case "file":
		printCommandUsage(writer, impactFileUsageLine)
	case "package":
		printCommandUsage(writer, impactPackageUsageLine)
	case "symbol":
		printCommandUsage(writer, impactSymbolUsageLine)
	default:
		printImpactUsage(writer)
	}
}

func printTestsUsage(writer io.Writer) {
	printUsageLines(writer, testsUsageLines)
}

func printTestsAffectedUsage(writer io.Writer) {
	printCommandUsage(writer, testsAffectedUsageLine)
}
