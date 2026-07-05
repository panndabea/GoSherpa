package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

const (
	exitSuccess       = 0
	exitFailure       = 1
	exitUsage         = 2
	jsonSchemaVersion = 1

	analysisModeAST             = agentcontext.AnalysisModeAST
	analysisModeDiff            = agentcontext.AnalysisModeDiff
	analysisModeDiffTypechecked = agentcontext.AnalysisModeDiffTypechecked
	confidenceMedium            = agentcontext.ConfidenceMedium
	confidenceLow               = agentcontext.ConfidenceLow
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
	All                bool
	HasAllOption       bool
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

func parseCLIArgs(args []string) (cliInvocation, error) {
	return parseCLIArgsWithFlagSpecs(args)
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
		fmt.Fprintln(stderr, "error: --tests is only supported by analyze, architecture, risk, symbols, search, packages, entrypoints, callers, explain, and context")
		return exitUsage
	}

	if invocation.HasAllOption && !supportsAllOption(invocation.Command) {
		fmt.Fprintln(stderr, "error: --all is only supported by deps")
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
		fmt.Fprintln(stderr, "error: --tags is only supported by analyze, refs, entrypoints, callers, callees, explain, context, impact, tests affected, implementers, interfaces, pr, doctor, and snapshot")
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

func supportsAllOption(command string) bool {
	spec, ok := commandSpecFor(command)
	return ok && spec.All
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

func printDepsUsage(writer io.Writer) {
	printUsageLines(writer, depsUsageLines)
}

func printTestsAffectedUsage(writer io.Writer) {
	printCommandUsage(writer, testsAffectedUsageLine)
}
