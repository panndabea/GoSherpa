package main

import "io"

type commandHandler func(cliInvocation, io.Writer, io.Writer) int

type commandSpec struct {
	Name          string
	Usage         []string
	Handler       commandHandler
	JSON          bool
	Limit         bool
	MaxDepth      bool
	Package       bool
	Kind          bool
	Tests         bool
	All           bool
	Context       bool
	ContextLimits bool
	Snapshot      bool
	Tags          bool
	TagsWhen      func(cliInvocation) bool
	BaseWhen      func(cliInvocation) bool
}

var commandSpecs = []commandSpec{
	{
		Name:     "analyze",
		Usage:    []string{analyzeUsageLine},
		Handler:  runAnalyzeCommand,
		JSON:     true,
		Tests:    true,
		Snapshot: true,
		Tags:     true,
	},
	{
		Name:    "architecture",
		Usage:   []string{architectureUsageLine},
		Handler: runArchitectureCommand,
		JSON:    true,
		Tests:   true,
	},
	{
		Name:    "risk",
		Usage:   []string{riskUsageLine},
		Handler: runRiskCommand,
		JSON:    true,
		Tests:   true,
	},
	{
		Name:          "context",
		Usage:         contextUsageLines,
		Handler:       runContextCommand,
		JSON:          true,
		Tests:         true,
		ContextLimits: true,
		Tags:          true,
		BaseWhen:      isContextDiffInvocation,
	},
	{
		Name:    "doctor",
		Usage:   []string{doctorUsageLine},
		Handler: runDoctorCommand,
		JSON:    true,
		Tags:    true,
	},
	{
		Name:    "snapshot",
		Usage:   []string{snapshotUsageLine},
		Handler: runSnapshotCommand,
		JSON:    true,
		Tags:    true,
	},
	{
		Name:  "completion",
		Usage: []string{completionUsageLine},
	},
	{
		Name:    "version",
		Usage:   []string{versionUsageLine},
		Handler: runVersionCommand,
		JSON:    true,
	},
	{
		Name:    "explain",
		Usage:   []string{explainUsageLine},
		Handler: runExplainCommand,
		JSON:    true,
		Tests:   true,
		Tags:    true,
	},
	{
		Name:     "symbols",
		Usage:    []string{symbolsUsageLine},
		Handler:  runSymbolsCommand,
		JSON:     true,
		Package:  true,
		Kind:     true,
		Tests:    true,
		Snapshot: true,
	},
	{
		Name:     "symbol",
		Usage:    []string{symbolUsageLine},
		Handler:  runSymbolCommand,
		JSON:     true,
		Context:  true,
		Snapshot: true,
	},
	{
		Name:     "search",
		Usage:    []string{searchUsageLine},
		Handler:  runSearchCommand,
		JSON:     true,
		Limit:    true,
		Package:  true,
		Kind:     true,
		Tests:    true,
		Snapshot: true,
	},
	{
		Name:    "refs",
		Usage:   []string{refsUsageLine},
		Handler: runRefsCommand,
		JSON:    true,
		Kind:    true,
		Context: true,
		Tags:    true,
	},
	{
		Name:     "impact",
		Usage:    impactUsageLines,
		Handler:  runImpactCommand,
		JSON:     true,
		Tags:     true,
		BaseWhen: isImpactDiffInvocation,
	},
	{
		Name:     "pr",
		Usage:    []string{prUsageLine},
		Handler:  runPRCommand,
		JSON:     true,
		Tags:     true,
		BaseWhen: isPRInvocation,
	},
	{
		Name:     "tests",
		Usage:    testsUsageLines,
		Handler:  runTestsCommand,
		JSON:     true,
		TagsWhen: isTestsAffectedInvocation,
		BaseWhen: isTestsAffectedInvocation,
	},
	{
		Name:    "deps",
		Usage:   depsUsageLines,
		Handler: runDepsCommand,
		JSON:    true,
		All:     true,
	},
	{
		Name:     "packages",
		Usage:    []string{packagesUsageLine},
		Handler:  runPackagesCommand,
		JSON:     true,
		Tests:    true,
		Snapshot: true,
	},
	{
		Name:    "implementers",
		Usage:   []string{implementersUsageLine},
		Handler: runImplementersCommand,
		JSON:    true,
		Tags:    true,
	},
	{
		Name:    "interface",
		Usage:   []string{interfaceUsageLine},
		Handler: runInterfaceCommand,
		JSON:    true,
		Tags:    true,
	},
	{
		Name:    "interfaces",
		Usage:   []string{interfacesUsageLine},
		Handler: runInterfacesCommand,
		JSON:    true,
		Tags:    true,
	},
	{
		Name:     "path",
		Usage:    []string{pathUsageLine},
		Handler:  runPathCommand,
		JSON:     true,
		Limit:    true,
		MaxDepth: true,
	},
	{
		Name:     "paths",
		Usage:    []string{pathsUsageLine},
		Handler:  runPathCommand,
		JSON:     true,
		Limit:    true,
		MaxDepth: true,
	},
	{
		Name:    "entrypoints",
		Usage:   []string{entrypointsUsageLine},
		Handler: runEntrypointsCommand,
		JSON:    true,
		Tests:   true,
		Tags:    true,
	},
	{
		Name:    "callers",
		Usage:   []string{callersUsageLine},
		Handler: runCallersCommand,
		JSON:    true,
		Tests:   true,
		Context: true,
		Tags:    true,
	},
	{
		Name:    "callees",
		Usage:   []string{calleesUsageLine},
		Handler: runCalleesCommand,
		JSON:    true,
		Context: true,
		Tags:    true,
	},
}

var commandSpecIndex = indexCommandSpecs(commandSpecs)

const (
	analyzeUsageLine        = "analyze [path] [--tests] [--use-snapshot]"
	architectureUsageLine   = "architecture [--tests]"
	riskUsageLine           = "risk [--tests]"
	contextSymbolUsageLine  = "context symbol <target> [--tests] [--max-references <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]"
	contextFileUsageLine    = "context file <file> [--tests] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]"
	contextPackageUsageLine = "context package <package> [--tests] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]"
	contextDiffUsageLine    = "context diff --base <ref> [--tests] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>]"
	doctorUsageLine         = "doctor"
	snapshotUsageLine       = "snapshot"
	completionUsageLine     = "completion zsh|bash|fish"
	versionUsageLine        = "version"
	explainUsageLine        = "explain <symbol> [--tests]"
	symbolsUsageLine        = "symbols [--kind <kind>] [--package <package>] [--tests] [--use-snapshot]"
	symbolUsageLine         = "symbol <target> [--context] [--use-snapshot]"
	searchUsageLine         = "search <terms> [--kind <kind>] [--package <package>] [--tests] [--limit <n>] [--use-snapshot]"
	refsUsageLine           = "refs <name> [--kind <kind>] [--context]"
	impactDefaultUsageLine  = "impact <symbol-or-package>"
	impactFileUsageLine     = "impact file <file>"
	impactPackageUsageLine  = "impact package <package>"
	impactSymbolUsageLine   = "impact symbol <symbol>"
	impactDiffUsageLine     = "impact diff --base <ref>"
	prUsageLine             = "pr --base <ref>"
	testsDefaultUsageLine   = "tests <symbol-or-package> [--scope direct|related|all]"
	testsAffectedUsageLine  = "tests affected --base <ref>"
	depsPackageUsageLine    = "deps <package>"
	depsAllUsageLine        = "deps --all"
	packagesUsageLine       = "packages [--tests] [--use-snapshot]"
	implementersUsageLine   = "implementers <interface>"
	interfaceUsageLine      = "interface <interface>"
	interfacesUsageLine     = "interfaces <type>"
	pathUsageLine           = "path <from> <to>"
	pathDetailedUsageLine   = "path <from> <to> [--limit <n>] [--max-depth <n>]"
	pathsUsageLine          = "paths <from> <to> [--limit <n>] [--max-depth <n>]"
	entrypointsUsageLine    = "entrypoints <function-or-method> [--tests]"
	callersUsageLine        = "callers <function-or-method> [--tests] [--context]"
	calleesUsageLine        = "callees <function-or-method> [--context]"
)

var (
	contextUsageLines = []string{
		contextSymbolUsageLine,
		contextFileUsageLine,
		contextPackageUsageLine,
		contextDiffUsageLine,
	}
	impactUsageLines = []string{
		impactDefaultUsageLine,
		impactFileUsageLine,
		impactPackageUsageLine,
		impactSymbolUsageLine,
		impactDiffUsageLine,
	}
	testsUsageLines = []string{
		testsDefaultUsageLine,
		testsAffectedUsageLine,
	}
	depsUsageLines = []string{
		depsPackageUsageLine,
		depsAllUsageLine,
	}
)

func indexCommandSpecs(specs []commandSpec) map[string]commandSpec {
	index := make(map[string]commandSpec, len(specs))
	for _, spec := range specs {
		index[spec.Name] = spec
	}

	return index
}

func commandSpecFor(command string) (commandSpec, bool) {
	spec, ok := commandSpecIndex[command]
	return spec, ok
}
