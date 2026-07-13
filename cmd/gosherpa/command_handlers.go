package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func runAnalyzeCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) > 1 {
		printCommandUsage(stderr, analyzeUsageLine)
		return exitUsage
	}

	rootInput := invocation.Root
	if len(invocation.CommandArgs) == 1 {
		rootInput = invocation.CommandArgs[0]
		if !filepath.IsAbs(rootInput) && invocation.Root != "" && invocation.Root != "." {
			rootInput = filepath.Join(invocation.Root, rootInput)
		}
	}

	root, ok := resolveRootPath(rootInput, stderr)
	if !ok {
		return exitFailure
	}

	report, err := analyzeRepository(root, analyzeOptions{
		IncludeTests: invocation.IncludeTests,
		BuildTags:    invocation.BuildTags,
		UseSnapshot:  invocation.UseSnapshot,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "analyze", ".", stderr, err)
	}

	if invocation.JSON {
		normalizedReport := analyzeJSONResult(report)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"analyze",
			normalizedReport.Target,
			normalizedReport.Warnings,
			normalizedReport,
		))
	}

	fmt.Fprint(stdout, formatAnalyzeReport(report))
	return exitSuccess
}

func runArchitectureCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 {
		printCommandUsage(stderr, architectureUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	report, err := sherpa.AnalyzeArchitecture(root, sherpa.ArchitectureOptions{
		IncludeTests: invocation.IncludeTests,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "architecture", ".", stderr, err)
	}

	if invocation.JSON {
		normalizedReport := architectureJSONResult(report)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"architecture",
			".",
			nil,
			normalizedReport,
		))
	}

	fmt.Fprint(stdout, sherpa.FormatArchitectureReport(report))
	return exitSuccess
}

func runRiskCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) != 0 {
		printCommandUsage(stderr, riskUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	report, err := sherpa.AnalyzeRisk(root, sherpa.RiskOptions{
		IncludeTests: invocation.IncludeTests,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "risk", ".", stderr, err)
	}

	if invocation.JSON {
		normalizedReport := riskJSONResult(report)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"risk",
			".",
			nil,
			normalizedReport,
		))
	}

	fmt.Fprint(stdout, sherpa.FormatRiskReport(report))
	return exitSuccess
}

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
			UseSnapshot:  invocation.UseSnapshot,
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

	symbols, warnings, analysisMode, err := loadSymbolsForInventoryCommand(root, invocation)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "symbol", target, stderr, err)
	}

	symbol, err := sherpa.FindSymbolTarget(root, symbols, target)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "symbol", target, stderr, err)
	}

	if invocation.JSON {
		return writeJSON(stdout, stderr, newJSONResponse(root, "symbol", target, warnings, symbolJSONData{
			AnalysisMode: analysisMode,
			Symbol:       symbol,
		}))
	}

	writeHumanWarnings(stderr, warnings)

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

	symbols, warnings, analysisMode, err := loadSymbolsForInventoryCommand(root, invocation)
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
		return writeJSON(stdout, stderr, newJSONResponse(root, "search", target, warnings, searchJSONData{
			AnalysisMode: analysisMode,
			Terms:        nonNilSlice(terms),
			Results:      nonNilSlice(results),
		}))
	}

	writeHumanWarnings(stderr, warnings)
	fmt.Fprint(stdout, sherpa.FormatSymbolSearch(terms, results))
	return exitSuccess
}

func runSymbolsCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	symbols, warnings, analysisMode, err := loadSymbolsForInventoryCommand(root, invocation)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "symbols", "", stderr, err)
	}
	symbols = sherpa.FilterSymbols(symbols, sherpa.SymbolFilterOptions{
		Kind:      invocation.SearchKind,
		Package:   invocation.SearchPackage,
		TestsOnly: invocation.IncludeTests,
	})

	if invocation.JSON {
		return writeJSON(stdout, stderr, newJSONResponse(root, "symbols", "", warnings, symbolsJSONData{
			AnalysisMode: analysisMode,
			Symbols:      nonNilSlice(symbols),
		}))
	}

	writeHumanWarnings(stderr, warnings)
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

	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "impact", target, stderr, err)
	}

	result, err := sherpa.FindImpactWithContext(semanticContext, target, sherpa.ImpactOptions{
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
			normalizedResult.Warnings,
			testsJSONDataFromResult(normalizedResult),
		))
	}

	fmt.Fprint(stdout, sherpa.FormatTests(result))
	return exitSuccess
}

func runDepsCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if invocation.All {
		if len(invocation.CommandArgs) != 0 {
			printDepsUsage(stderr)
			return exitUsage
		}

		root, ok := resolveRootPath(invocation.Root, stderr)
		if !ok {
			return exitFailure
		}

		report, err := sherpa.FindRepositoryDependencies(root)
		if err != nil {
			return writeCommandError(invocation.JSON, root, "deps", "all", stderr, err)
		}

		if invocation.JSON {
			normalizedReport := repositoryDependenciesJSONResult(report)
			return writeJSON(stdout, stderr, newJSONResponse(
				root,
				"deps",
				"all",
				nil,
				repositoryDependenciesJSONDataFromResult(normalizedReport),
			))
		}

		fmt.Fprint(stdout, sherpa.FormatRepositoryDependencies(report))
		return exitSuccess
	}

	if len(invocation.CommandArgs) != 1 {
		printDepsUsage(stderr)
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

	packages, warnings, analysisMode, err := loadPackagesForInventoryCommand(root, invocation)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "packages", "", stderr, err)
	}

	if invocation.JSON {
		normalizedPackages := packagesJSONResult(packages)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"packages",
			"",
			warnings,
			packagesJSONDataFromResult(normalizedPackages, analysisMode),
		))
	}

	writeHumanWarnings(stderr, warnings)
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

	options := impactengine.InterfaceOptions{
		BuildTags: invocation.BuildTags,
	}
	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "implementers", target, stderr, err)
	}

	result, err := impactengine.FindImplementersWithContext(semanticContext, target, options)
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

func runInterfaceCommand(invocation cliInvocation, stdout io.Writer, stderr io.Writer) int {
	if len(invocation.CommandArgs) < 1 {
		printCommandUsage(stderr, interfaceUsageLine)
		return exitUsage
	}

	root, ok := resolveRootPath(invocation.Root, stderr)
	if !ok {
		return exitFailure
	}

	target := invocation.CommandArgs[0]

	options := impactengine.InterfaceOptions{
		BuildTags: invocation.BuildTags,
	}
	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "interface", target, stderr, err)
	}

	result, err := impactengine.InspectInterfaceWithContext(semanticContext, target, options)
	if err != nil {
		return writeCommandError(invocation.JSON, root, "interface", target, stderr, err)
	}

	if invocation.JSON {
		normalizedResult := interfaceJSONResult(result)
		return writeJSON(stdout, stderr, newJSONResponse(
			root,
			"interface",
			normalizedResult.Target,
			normalizedResult.Warnings,
			interfaceJSONDataFromResult(normalizedResult),
		))
	}

	fmt.Fprint(stdout, impactengine.FormatInterface(result))
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

	options := impactengine.InterfaceOptions{
		BuildTags: invocation.BuildTags,
	}
	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return writeCommandError(invocation.JSON, root, "interfaces", target, stderr, err)
	}

	result, err := impactengine.FindInterfacesWithContext(semanticContext, target, options)
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
			normalizedResult.Warnings,
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
