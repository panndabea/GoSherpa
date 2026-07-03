package main

import (
	"fmt"
	"io"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func runDoctorCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 {
		printDoctorUsage(stderr)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	report := analyzeDoctor(root, invocation.BuildTags)
	if invocation.JSON {
		normalizedReport := normalizeDoctorReport(report)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"doctor",
			normalizedReport.Target,
			normalizedReport.Warnings,
			normalizedReport,
		))
	}

	fmt.Fprint(stdout, formatDoctorReport(report))
	return exitSuccess
}

func runContextCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printContextUsage(stderr)
		return exitUsage
	}

	switch invocation.CommandArgs[0] {
	case "symbol":
		if len(invocation.CommandArgs) != 2 {
			printContextSymbolUsage(stderr)
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[1]
		report, err := agentcontext.AnalyzeSymbol(root, target, agentcontext.AnalyzeOptions{
			IncludeTests: invocation.IncludeTests,
			BuildTags:    invocation.BuildTags,
			Limits:       invocation.ContextLimits,
		})
		if err != nil {
			return writeCommandError(invocation.JSON, root, "context symbol", target, stderr, err)
		}

		if invocation.JSON {
			normalizedReport := contextSymbolJSONResult(report)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"context symbol",
				normalizedReport.Target,
				normalizedReport.Warnings,
				normalizedReport,
			))
		}

		fmt.Fprint(stdout, agentcontext.Format(report))
		return exitSuccess
	case "file":
		if len(invocation.CommandArgs) != 2 {
			printContextFileUsage(stderr)
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[1]
		report, err := agentcontext.AnalyzeFile(root, target, agentcontext.FileAnalyzeOptions{
			IncludeTests: invocation.IncludeTests,
			BuildTags:    invocation.BuildTags,
			Limits:       invocation.ContextLimits,
		})
		if err != nil {
			return writeCommandError(invocation.JSON, root, "context file", target, stderr, err)
		}

		if invocation.JSON {
			normalizedReport := contextFileJSONResult(report)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"context file",
				normalizedReport.Target,
				normalizedReport.Warnings,
				normalizedReport,
			))
		}

		fmt.Fprint(stdout, agentcontext.FormatFile(report))
		return exitSuccess
	case "package":
		if len(invocation.CommandArgs) != 2 {
			printContextPackageUsage(stderr)
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		target := invocation.CommandArgs[1]
		report, err := agentcontext.AnalyzePackage(root, target, agentcontext.PackageAnalyzeOptions{
			IncludeTests: invocation.IncludeTests,
			BuildTags:    invocation.BuildTags,
			Limits:       invocation.ContextLimits,
		})
		if err != nil {
			return writeCommandError(invocation.JSON, root, "context package", target, stderr, err)
		}

		if invocation.JSON {
			normalizedReport := contextPackageJSONResult(report)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"context package",
				normalizedReport.Target,
				normalizedReport.Warnings,
				normalizedReport,
			))
		}

		fmt.Fprint(stdout, agentcontext.FormatPackage(report))
		return exitSuccess
	case "diff":
		if len(invocation.CommandArgs) != 1 || !invocation.HasBaseOption {
			printContextDiffUsage(stderr)
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		report, err := agentcontext.AnalyzeDiff(root, invocation.BaseRef, agentcontext.DiffAnalyzeOptions{
			IncludeTests: invocation.IncludeTests,
			BuildTags:    invocation.BuildTags,
			Limits:       invocation.ContextLimits,
		})
		if err != nil {
			return writeCommandError(invocation.JSON, root, "context diff", invocation.BaseRef, stderr, err)
		}

		if invocation.JSON {
			normalizedReport := contextDiffJSONResult(report)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"context diff",
				normalizedReport.Target,
				normalizedReport.Warnings,
				normalizedReport,
			))
		}

		fmt.Fprint(stdout, agentcontext.FormatDiff(report))
		return exitSuccess
	default:
		printContextUsage(stderr)
		return exitUsage
	}
}

func runPRCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 || !invocation.HasBaseOption {
		printPRUsage(stderr)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	report, err := analyzePR(root, invocation.BaseRef, invocation.BuildTags)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "pr", invocation.BaseRef, stderr, err)
	}

	if invocation.JSON {
		normalizedReport := normalizePRReport(report)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"pr",
			normalizedReport.Base,
			normalizedReport.Warnings,
			normalizedReport,
		))
	}

	fmt.Fprint(stdout, formatPRReport(report))
	return exitSuccess
}

func runExplainCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, explainUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	target := invocation.CommandArgs[0]

	report, err := explainengine.AnalyzeWithOptions(root, target, explainengine.AnalyzeOptions{
		IncludeTests: invocation.IncludeTests,
		BuildTags:    invocation.BuildTags,
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
}

func runSymbolCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, symbolUsageLine)
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

	if invocation.ShowContext {
		context, err := sherpa.ReadSourceContext(root, symbol.Position, sherpa.DefaultSourceContextRadius)
		if err != nil {
			return writeCommandError(false, root, "symbol", target, stderr, err)
		}

		fmt.Fprint(stdout, sherpa.FormatSymbolWithContext(symbol, context))
		return exitSuccess
	}

	fmt.Fprint(stdout, sherpa.FormatSymbol(symbol))
	return exitSuccess
}

func runSearchCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, searchUsageLine)
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
}

func runSymbolsCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	symbols, err := sherpa.ParseRepository(root)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "symbols", "", stderr, err)
	}
	symbols = sherpa.FilterSymbols(symbols, sherpa.SymbolFilterOptions{
		Kind:      invocation.SearchKind,
		Package:   invocation.SearchPackage,
		TestsOnly: invocation.IncludeTests,
	})

	if invocation.JSON {
		return writeJSON(stdout, stderr, newJSONResponse(root, "symbols", "", nil, symbolsJSONData{
			Symbols: nonNilSlice(symbols),
		}))
	}

	fmt.Fprint(stdout, sherpa.FormatSymbols(symbols))
	return exitSuccess
}

func runRefsCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, refsUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	name := invocation.CommandArgs[0]

	report, err := sherpa.FindReferenceReportWithOptions(root, name, sherpa.ReferenceOptions{
		Kind:      invocation.ReferenceKind,
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "refs", name, stderr, err)
	}

	if invocation.JSON {
		return writeJSON(stdout, stderr, newJSONResponse(root, "refs", name, report.Warnings, referencesJSONData{
			AnalysisMode: report.AnalysisMode,
			Confidence:   jsonConfidence(report.Warnings, report.AnalysisMode),
			Limitations:  referenceLimitations(report.AnalysisMode),
			References:   nonNilSlice(report.References),
		}))
	}

	if invocation.ShowContext {
		contexts, err := sherpa.ReadSourceContexts(root, referencePositions(report.References), sherpa.DefaultSourceContextRadius)
		if err != nil {
			return writeCommandError(false, root, "refs", name, stderr, err)
		}

		fmt.Fprint(stdout, sherpa.FormatReferenceReportWithContext(report, contexts))
		return exitSuccess
	}

	fmt.Fprint(stdout, sherpa.FormatReferenceReport(report))
	return exitSuccess
}

func runImpactCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
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

		report, err := impactengine.AnalyzeDiffWithOptions(root, invocation.BaseRef, "", impactengine.AnalyzerOptions{
			BuildTags: invocation.BuildTags,
		})
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
				impactDiffJSONDataFromReport(normalizedReport, analysisModeDiff),
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
		report, err := analyzeImpactSubcommand(root, kind, target, invocation.BuildTags)
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
				impactDiffJSONDataFromReport(normalizedReport, analysisModeAST),
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

	result, err := sherpa.FindImpactWithOptions(root, target, sherpa.ImpactOptions{
		BuildTags: invocation.BuildTags,
	})
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
}

func runTestsCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
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

		report, err := impactengine.AnalyzeDiffWithOptions(root, invocation.BaseRef, "", impactengine.AnalyzerOptions{
			BuildTags: invocation.BuildTags,
		})
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

	result, err := sherpa.FindTestsWithOptions(root, target, sherpa.TestOptions{
		Scope: invocation.TestScope,
	})
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
}

func runDepsCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, depsUsageLine)
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
}

func runPackagesCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 {
		printCommandUsage(stderr, packagesUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	packages, err := sherpa.FindPackageSummaries(root, sherpa.PackageInventoryOptions{
		IncludeTests: invocation.IncludeTests,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "packages", "", stderr, err)
	}

	if invocation.JSON {
		normalizedPackages := packagesJSONResult(packages)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"packages",
			"",
			nil,
			packagesJSONDataFromResult(normalizedPackages),
		))
	}

	fmt.Fprint(stdout, sherpa.FormatPackageSummaries(packages))
	return exitSuccess
}

func runImplementersCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, implementersUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	target := invocation.CommandArgs[0]

	result, err := impactengine.FindImplementersWithOptions(root, target, impactengine.InterfaceOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "implementers", target, stderr, err)
	}

	if invocation.JSON {
		normalizedResult := implementersJSONResult(result)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"implementers",
			normalizedResult.Target,
			normalizedResult.Warnings,
			implementersJSONDataFromResult(normalizedResult),
		))
	}

	fmt.Fprint(stdout, impactengine.FormatImplementers(result))
	return exitSuccess
}

func runInterfacesCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, interfacesUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	target := invocation.CommandArgs[0]

	result, err := impactengine.FindInterfacesWithOptions(root, target, impactengine.InterfaceOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "interfaces", target, stderr, err)
	}

	if invocation.JSON {
		normalizedResult := interfacesJSONResult(result)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"interfaces",
			normalizedResult.Target,
			normalizedResult.Warnings,
			interfacesJSONDataFromResult(normalizedResult),
		))
	}

	fmt.Fprint(stdout, impactengine.FormatInterfaces(result))
	return exitSuccess
}

func runPathCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 2 {
		usage := pathDetailedUsageLine
		if invocation.Command == "paths" {
			usage = pathsUsageLine
		}
		printCommandUsage(stderr, usage)
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
}

func runEntrypointsCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, entrypointsUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	target := invocation.CommandArgs[0]

	result, err := sherpa.FindEntryPointsWithOptions(root, target, sherpa.CallOptions{
		IncludeTests: invocation.IncludeTests,
		BuildTags:    invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "entrypoints", target, stderr, err)
	}

	if invocation.JSON {
		normalizedResult := entrypointsJSONResult(result)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"entrypoints",
			normalizedResult.Target,
			normalizedResult.Warnings,
			entrypointsJSONDataFromResult(normalizedResult),
		))
	}

	fmt.Fprint(stdout, sherpa.FormatEntryPoints(result))
	return exitSuccess
}

func runCallersCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, callersUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	target := invocation.CommandArgs[0]

	result, err := sherpa.FindCallersWithOptions(root, target, sherpa.CallOptions{
		IncludeTests: invocation.IncludeTests,
		BuildTags:    invocation.BuildTags,
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
			normalizedResult.Warnings,
			callersJSONDataFromResult(normalizedResult),
		))
	}

	if invocation.ShowContext {
		contexts, err := sherpa.ReadSourceContexts(root, callerPositions(result.Callers), sherpa.DefaultSourceContextRadius)
		if err != nil {
			return writeCommandError(false, root, "callers", target, stderr, err)
		}

		fmt.Fprint(stdout, sherpa.FormatCallersWithContext(result, contexts))
		return exitSuccess
	}

	fmt.Fprint(stdout, sherpa.FormatCallers(result))
	return exitSuccess
}

func runCalleesCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, calleesUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	target := invocation.CommandArgs[0]

	result, err := sherpa.FindCalleesWithOptions(root, target, sherpa.CallOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "callees", target, stderr, err)
	}

	if invocation.JSON {
		normalizedResult := calleesJSONResult(result)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"callees",
			normalizedResult.Target,
			normalizedResult.Warnings,
			calleesJSONDataFromResult(normalizedResult),
		))
	}

	if invocation.ShowContext {
		contexts, err := sherpa.ReadSourceContexts(root, calleePositions(result.Callees), sherpa.DefaultSourceContextRadius)
		if err != nil {
			return writeCommandError(false, root, "callees", target, stderr, err)
		}

		fmt.Fprint(stdout, sherpa.FormatCalleesWithContext(result, contexts))
		return exitSuccess
	}

	fmt.Fprint(stdout, sherpa.FormatCallees(result))
	return exitSuccess
}
