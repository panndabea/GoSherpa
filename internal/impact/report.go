package impact

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	gitdiff "github.com/panndabea/GoSherpa/internal/git"
	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
	"github.com/panndabea/GoSherpa/internal/symbolindex"
)

type Analyzer struct {
	Root               string
	BuildTags          []string
	UseSnapshotSymbols bool
	SnapshotSymbols    []sherpa.Symbol
}

type AnalyzerOptions struct {
	BuildTags          []string
	UseSnapshotSymbols bool
	SnapshotSymbols    []sherpa.Symbol
}

type RelatedTest = sherpa.RelatedTest
type TestPlan = sherpa.TestPlan

type ImpactReport struct {
	ChangedFiles            []string        `json:"changedFiles"`
	ChangedPackages         []string        `json:"changedPackages"`
	AffectedPackages        []string        `json:"affectedPackages"`
	AffectedSymbols         []string        `json:"affectedSymbols"`
	ChangedSymbolDetails    []ChangedSymbol `json:"changedSymbolDetails,omitempty"`
	ReferenceAnalysisMode   string          `json:"referenceAnalysisMode,omitempty"`
	CallAnalysisMode        string          `json:"callAnalysisMode,omitempty"`
	AffectedInterfaces      []string        `json:"affectedInterfaces"`
	AffectedImplementations []string        `json:"affectedImplementations"`
	InterfaceAnalysisMode   string          `json:"interfaceAnalysisMode,omitempty"`
	AffectedTests           []RelatedTest   `json:"affectedTests"`
	TestAnalysisMode        string          `json:"testAnalysisMode,omitempty"`
	TestCommands            []string        `json:"testCommands"`
	TestPlan                TestPlan        `json:"testPlan"`
	Warnings                []string        `json:"warnings"`
}

type changedSymbolImpact struct {
	Packages              []string
	Tests                 []sherpa.RelatedTest
	TestAnalysisMode      string
	ReferenceAnalysisMode string
	CallAnalysisMode      string
	Warnings              []string
}

func NewAnalyzer(root string) Analyzer {
	return Analyzer{Root: root}
}

func NewAnalyzerWithOptions(root string, options AnalyzerOptions) Analyzer {
	return Analyzer{
		Root:               root,
		BuildTags:          append([]string{}, options.BuildTags...),
		UseSnapshotSymbols: options.UseSnapshotSymbols,
		SnapshotSymbols:    append([]sherpa.Symbol{}, options.SnapshotSymbols...),
	}
}

func AnalyzeDiff(root string, base string, head string) (ImpactReport, error) {
	return NewAnalyzer(root).AnalyzeDiff(base, head)
}

func AnalyzeDiffWithOptions(root string, base string, head string, options AnalyzerOptions) (ImpactReport, error) {
	return NewAnalyzerWithOptions(root, options).AnalyzeDiff(base, head)
}

func AnalyzeDiffWithContext(context *sherpa.SemanticContext, base string, head string, options AnalyzerOptions) (ImpactReport, error) {
	if context == nil {
		return ImpactReport{}, fmt.Errorf("semantic context is nil")
	}

	var err error
	options, err = analyzerOptionsForContext(context, options)
	if err != nil {
		return ImpactReport{}, err
	}

	return NewAnalyzerWithOptions(context.Root(), options).AnalyzeDiffWithContext(context, base, head)
}

func AnalyzeFile(root string, file string) (ImpactReport, error) {
	return NewAnalyzer(root).AnalyzeFile(file)
}

func AnalyzeFileWithOptions(root string, file string, options AnalyzerOptions) (ImpactReport, error) {
	return NewAnalyzerWithOptions(root, options).AnalyzeFile(file)
}

func AnalyzeFileWithContext(context *sherpa.SemanticContext, file string, options AnalyzerOptions) (ImpactReport, error) {
	if context == nil {
		return ImpactReport{}, fmt.Errorf("semantic context is nil")
	}

	var err error
	options, err = analyzerOptionsForContext(context, options)
	if err != nil {
		return ImpactReport{}, err
	}

	return NewAnalyzerWithOptions(context.Root(), options).AnalyzeFileWithContext(context, file)
}

func AnalyzePackage(root string, targetPackage string) (ImpactReport, error) {
	return NewAnalyzer(root).AnalyzePackage(targetPackage)
}

func AnalyzePackageWithOptions(root string, targetPackage string, options AnalyzerOptions) (ImpactReport, error) {
	return NewAnalyzerWithOptions(root, options).AnalyzePackage(targetPackage)
}

func AnalyzePackageWithContext(context *sherpa.SemanticContext, targetPackage string, options AnalyzerOptions) (ImpactReport, error) {
	if context == nil {
		return ImpactReport{}, fmt.Errorf("semantic context is nil")
	}

	var err error
	options, err = analyzerOptionsForContext(context, options)
	if err != nil {
		return ImpactReport{}, err
	}

	return NewAnalyzerWithOptions(context.Root(), options).AnalyzePackageWithContext(context, targetPackage)
}

func AnalyzeSymbol(root string, target string) (ImpactReport, error) {
	return NewAnalyzer(root).AnalyzeSymbol(target)
}

func AnalyzeSymbolWithOptions(root string, target string, options AnalyzerOptions) (ImpactReport, error) {
	return NewAnalyzerWithOptions(root, options).AnalyzeSymbol(target)
}

func AnalyzeSymbolWithContext(context *sherpa.SemanticContext, target string, options AnalyzerOptions) (ImpactReport, error) {
	if context == nil {
		return ImpactReport{}, fmt.Errorf("semantic context is nil")
	}

	var err error
	options, err = analyzerOptionsForContext(context, options)
	if err != nil {
		return ImpactReport{}, err
	}

	return NewAnalyzerWithOptions(context.Root(), options).AnalyzeSymbolWithContext(context, target)
}

func (a Analyzer) AnalyzeDiff(base string, head string) (ImpactReport, error) {
	semanticContext, err := sherpa.NewSemanticContext(a.Root, sherpa.SemanticContextOptions{
		BuildTags: a.BuildTags,
	})
	if err != nil {
		return ImpactReport{}, err
	}

	return a.AnalyzeDiffWithContext(semanticContext, base, head)
}

func (a Analyzer) AnalyzeDiffWithContext(context *sherpa.SemanticContext, base string, head string) (ImpactReport, error) {
	if context == nil {
		return ImpactReport{}, fmt.Errorf("semantic context is nil")
	}

	buildTags, err := analyzerBuildTagsForContext(context, a.BuildTags)
	if err != nil {
		return ImpactReport{}, err
	}
	a.BuildTags = buildTags

	return a.analyzeDiffWithContext(context, base, head)
}

func (a Analyzer) analyzeDiffWithContext(semanticContext *sherpa.SemanticContext, base string, head string) (ImpactReport, error) {
	changedFiles, err := gitdiff.ChangedFiles(a.Root, base, head)
	if err != nil {
		return ImpactReport{}, err
	}

	report := ImpactReport{
		ChangedFiles:            changedFiles,
		ChangedPackages:         PackagesForFiles(changedFiles),
		AffectedSymbols:         []string{},
		AffectedInterfaces:      []string{},
		AffectedImplementations: []string{},
	}

	changedSymbols, err := changedSymbolsForDiffWithCurrentSymbols(a.Root, base, head, a.SnapshotSymbols, a.UseSnapshotSymbols)
	if err != nil {
		return ImpactReport{}, err
	}
	modulePath := impactModulePath(a.Root)
	report.AffectedSymbols = changedSymbolNames(changedSymbols)
	report.ChangedSymbolDetails = changedSymbolsWithTargets(changedSymbols, modulePath)
	symbolImpact := a.analyzeChangedSymbolImpactsWithContext(changedSymbols, semanticContext)
	report.AffectedPackages, report.Warnings = affectedPackagesForChangedPackages(a.Root, report.ChangedPackages)
	report.AffectedPackages = uniqueSortedStrings(append(report.AffectedPackages, symbolImpact.Packages...))
	report.ReferenceAnalysisMode = symbolImpact.ReferenceAnalysisMode
	report.CallAnalysisMode = symbolImpact.CallAnalysisMode
	report.TestAnalysisMode = symbolImpact.TestAnalysisMode
	report.Warnings = uniqueSortedStrings(append(report.Warnings, symbolImpact.Warnings...))
	signals, err := interfaceSignalsForPackagesWithContext(semanticContext, a.Root, report.ChangedPackages, InterfaceOptions{
		BuildTags: a.BuildTags,
	})
	if err != nil {
		return ImpactReport{}, err
	}
	report.AffectedInterfaces = signals.Interfaces
	report.AffectedImplementations = signals.Implementations
	report.InterfaceAnalysisMode = signals.AnalysisMode
	report.Warnings = uniqueSortedStrings(append(report.Warnings, signals.Warnings...))
	contractPackages := contractPackagesForSignals(signals)
	report.AffectedPackages = uniqueSortedStrings(append(report.AffectedPackages, contractPackages...))
	report.AffectedTests, report.TestPlan, report.TestCommands, report.TestAnalysisMode, report.Warnings = affectedTestsForPackagesWithContext(semanticContext, a.Root, report.ChangedPackages, report.AffectedPackages, changedSymbols, symbolImpact.Tests, contractPackages, report.Warnings)

	return normalizeReport(report), nil
}

func analyzerOptionsForContext(context *sherpa.SemanticContext, options AnalyzerOptions) (AnalyzerOptions, error) {
	buildTags, err := analyzerBuildTagsForContext(context, options.BuildTags)
	if err != nil {
		return AnalyzerOptions{}, err
	}
	options.BuildTags = buildTags

	return options, nil
}

func analyzerBuildTagsForContext(context *sherpa.SemanticContext, buildTags []string) ([]string, error) {
	if context == nil {
		return semantics.NormalizeBuildTags(buildTags), nil
	}

	contextTags := context.BuildTags()
	optionTags := semantics.NormalizeBuildTags(buildTags)
	if len(optionTags) == 0 {
		return contextTags, nil
	}
	if strings.Join(optionTags, "\x00") != strings.Join(contextTags, "\x00") {
		return nil, fmt.Errorf("semantic context build tags do not match analyzer options")
	}

	return optionTags, nil
}

func (a Analyzer) AnalyzeFile(file string) (ImpactReport, error) {
	return a.analyzeFile(file, nil)
}

func (a Analyzer) AnalyzeFileWithContext(context *sherpa.SemanticContext, file string) (ImpactReport, error) {
	buildTags, err := analyzerBuildTagsForContext(context, a.BuildTags)
	if err != nil {
		return ImpactReport{}, err
	}
	a.BuildTags = buildTags

	return a.analyzeFile(file, context)
}

func (a Analyzer) analyzeFile(file string, context *sherpa.SemanticContext) (ImpactReport, error) {
	changedFile, changedPackage, err := fileTarget(file)
	if err != nil {
		return ImpactReport{}, err
	}

	var report ImpactReport
	if context != nil {
		report, err = a.AnalyzePackageWithContext(context, changedPackage)
	} else {
		report, err = a.AnalyzePackage(changedPackage)
	}
	if err != nil {
		return ImpactReport{}, err
	}

	report.ChangedFiles = []string{changedFile}
	report.ChangedPackages = []string{changedPackage}

	return normalizeReport(report), nil
}

func (a Analyzer) AnalyzePackage(targetPackage string) (ImpactReport, error) {
	return a.analyzePackage(targetPackage, nil)
}

func (a Analyzer) AnalyzePackageWithContext(context *sherpa.SemanticContext, targetPackage string) (ImpactReport, error) {
	buildTags, err := analyzerBuildTagsForContext(context, a.BuildTags)
	if err != nil {
		return ImpactReport{}, err
	}
	a.BuildTags = buildTags

	return a.analyzePackage(targetPackage, context)
}

func (a Analyzer) analyzePackage(targetPackage string, context *sherpa.SemanticContext) (ImpactReport, error) {
	impactOptions := sherpa.ImpactOptions{
		BuildTags: a.BuildTags,
	}
	var result sherpa.ImpactResult
	var err error
	if context != nil {
		result, err = sherpa.FindImpactWithContext(context, targetPackage, impactOptions)
	} else {
		result, err = sherpa.FindImpactWithOptions(a.Root, targetPackage, impactOptions)
	}
	if err != nil {
		return ImpactReport{}, err
	}
	if result.Kind != sherpa.ImpactKindPackage {
		return ImpactReport{}, fmt.Errorf("package impact target must be a package path: %s", targetPackage)
	}

	report := reportFromImpactResult(result)
	report.ChangedPackages = []string{result.Target}
	signals, err := interfaceSignalsForPackagesWithContext(context, a.Root, report.ChangedPackages, InterfaceOptions{
		BuildTags: a.BuildTags,
	})
	if err != nil {
		return ImpactReport{}, err
	}
	report.AffectedInterfaces = signals.Interfaces
	report.AffectedImplementations = signals.Implementations
	report.InterfaceAnalysisMode = signals.AnalysisMode
	report.Warnings = uniqueSortedStrings(append(report.Warnings, signals.Warnings...))
	contractPackages := contractPackagesForSignals(signals)
	report.AffectedPackages = uniqueSortedStrings(append(report.AffectedPackages, contractPackages...))
	report.AffectedTests, report.TestPlan, report.TestCommands, report.TestAnalysisMode, report.Warnings = affectedTestsForPackagesWithContext(context, a.Root, report.ChangedPackages, report.AffectedPackages, nil, nil, contractPackages, report.Warnings)

	return normalizeReport(report), nil
}

func (a Analyzer) AnalyzeSymbol(target string) (ImpactReport, error) {
	semanticContext, err := sherpa.NewSemanticContext(a.Root, sherpa.SemanticContextOptions{
		BuildTags: a.BuildTags,
	})
	if err != nil {
		return ImpactReport{}, err
	}

	return a.analyzeSymbol(target, semanticContext)
}

func (a Analyzer) AnalyzeSymbolWithContext(context *sherpa.SemanticContext, target string) (ImpactReport, error) {
	buildTags, err := analyzerBuildTagsForContext(context, a.BuildTags)
	if err != nil {
		return ImpactReport{}, err
	}
	a.BuildTags = buildTags

	return a.analyzeSymbol(target, context)
}

func (a Analyzer) analyzeSymbol(target string, context *sherpa.SemanticContext) (ImpactReport, error) {
	if err := a.requireUniqueSymbolTarget(target, context); err != nil {
		return ImpactReport{}, err
	}

	impactOptions := sherpa.ImpactOptions{
		BuildTags: a.BuildTags,
	}
	var result sherpa.ImpactResult
	var err error
	if context != nil {
		result, err = sherpa.FindImpactWithContext(context, target, impactOptions)
	} else {
		result, err = sherpa.FindImpactWithOptions(a.Root, target, impactOptions)
	}
	if err != nil {
		return ImpactReport{}, err
	}
	if result.Kind != sherpa.ImpactKindSymbol {
		return ImpactReport{}, fmt.Errorf("symbol impact target must be a symbol: %s", target)
	}

	report := reportFromImpactResult(result)
	report.AffectedSymbols = []string{result.Target}
	signals, err := interfaceSignalsForSymbolWithContext(context, a.Root, target, InterfaceOptions{
		BuildTags: a.BuildTags,
	})
	if err != nil {
		return ImpactReport{}, err
	}
	report.AffectedInterfaces = signals.Interfaces
	report.AffectedImplementations = signals.Implementations
	report.InterfaceAnalysisMode = signals.AnalysisMode
	report.Warnings = uniqueSortedStrings(append(report.Warnings, signals.Warnings...))
	contractPackages := contractPackagesForSignals(signals)
	report.AffectedPackages = uniqueSortedStrings(append(report.AffectedPackages, contractPackages...))
	report = a.enrichSymbolContractTestsWithContext(context, report, result, contractPackages, contractTargetsByPackage(signals))

	return normalizeReport(report), nil
}

func (a Analyzer) requireUniqueSymbolTarget(target string, context *sherpa.SemanticContext) error {
	root := a.Root
	if context != nil {
		root = context.Root()
		repo, attempted, err := context.TypecheckedRepository()
		if attempted && err == nil {
			index, err := symbolindex.FromRepository(repo)
			if err == nil {
				if _, found, err := index.FindSymbol(target); err != nil {
					return err
				} else if found {
					return nil
				}
			}
		}
	}

	symbols, err := sherpa.ParseRepository(root)
	if err != nil {
		return err
	}

	_, err = sherpa.FindSymbolTarget(root, symbols, target)
	return err
}

func reportFromImpactResult(result sherpa.ImpactResult) ImpactReport {
	return ImpactReport{
		AffectedPackages:      result.Packages,
		ReferenceAnalysisMode: result.ReferenceAnalysisMode,
		CallAnalysisMode:      result.CallAnalysisMode,
		AffectedTests:         result.RelatedTests,
		TestAnalysisMode:      result.TestAnalysisMode,
		TestCommands:          result.TestCommands,
		TestPlan:              result.TestPlan,
		Warnings:              result.Warnings,
	}
}

func (a Analyzer) enrichSymbolContractTestsWithContext(context *sherpa.SemanticContext, report ImpactReport, result sherpa.ImpactResult, contractPackages []string, targetsByPackage map[string][]string) ImpactReport {
	contractPackages = uniqueSortedStrings(contractPackages)
	if len(contractPackages) == 0 {
		return report
	}

	seen := make(map[string]sherpa.RelatedTest)
	for _, test := range report.AffectedTests {
		mergeRelatedTest(seen, test)
	}

	testAnalysisMode := report.TestAnalysisMode
	warnings := append([]string{}, report.Warnings...)
	for _, pkg := range contractPackages {
		tests, err := findTestsWithContext(context, a.Root, pkg, sherpa.TestOptions{Scope: sherpa.TestScopeAll})
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		testAnalysisMode = mergeTestAnalysisMode(testAnalysisMode, tests.AnalysisMode)
		warnings = append(warnings, tests.Warnings...)

		for _, test := range tests.Tests {
			if targets := targetsByPackage[pkg]; len(targets) > 0 {
				test.Targets = uniqueSortedStrings(append(test.Targets, targets...))
			}
			test = addRelatedTestReason(test, sherpa.RelatedTestReasonContract)
			mergeRelatedTest(seen, test)
		}
	}

	report.AffectedTests = make([]sherpa.RelatedTest, 0, len(seen))
	for _, test := range seen {
		report.AffectedTests = append(report.AffectedTests, test)
	}
	sortRelatedTests(report.AffectedTests)

	targetPackages := symbolTestPlanTargetPackages(result.Target, result.TestPlan)
	callerPackages := packagesFromTestPlanItems(result.TestPlan.CallerPackages)
	fallbackPackages := uniqueSortedStrings(append(append([]string{}, report.AffectedPackages...), targetPackages...))
	targets := nonEmptyStrings(result.Target)
	report.TestPlan = sherpa.PlanTests(report.AffectedTests, sherpa.TestPlanOptions{
		Target:           firstNonEmptyImpactString(result.Target, "target"),
		Kind:             sherpa.TestTargetKindSymbol,
		TargetPackages:   targetPackages,
		ContractPackages: contractPackages,
		CallerPackages:   callerPackages,
		FallbackPackages: fallbackPackages,
		Targets:          targets,
	})
	report.TestCommands = sherpa.TestPlanCommands(report.TestPlan)
	report.TestAnalysisMode = normalizeTestAnalysisMode(testAnalysisMode)
	report.Warnings = uniqueSortedStrings(warnings)

	return report
}

func fileTarget(file string) (string, string, error) {
	value := strings.TrimSpace(file)
	if value == "" {
		return "", "", fmt.Errorf("file path is empty")
	}
	if filepath.IsAbs(value) {
		return "", "", fmt.Errorf("absolute file paths are not supported: %s", file)
	}

	value = path.Clean(filepath.ToSlash(value))
	pkg, ok := packageForChangedFile(value)
	if !ok {
		return "", "", fmt.Errorf("impact file target must be a repository-local Go file: %s", file)
	}

	return value, pkg, nil
}

func symbolLookupTarget(target string) string {
	value := strings.TrimSpace(target)
	lastSlash := strings.LastIndexAny(value, `/\`)
	if lastSlash < 0 {
		return value
	}

	tail := value[lastSlash+1:]
	firstDot := strings.Index(tail, ".")
	if firstDot < 0 || firstDot+1 >= len(tail) {
		return value
	}

	return tail[firstDot+1:]
}

func affectedPackagesForChangedPackages(root string, changedPackages []string) ([]string, []string) {
	affected := append([]string{}, changedPackages...)
	var warnings []string

	for _, pkg := range changedPackages {
		deps, err := sherpa.FindPackageDependencies(root, pkg)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}

		affected = append(affected, deps.Package)
		affected = append(affected, deps.UsedBy...)
	}

	return uniqueSortedStrings(affected), uniqueSortedStrings(warnings)
}

func changedSymbolsForDiff(root string, base string, head string) ([]changedSymbol, error) {
	return changedSymbolsForDiffWithCurrentSymbols(root, base, head, nil, false)
}

func changedSymbolsForDiffWithCurrentSymbols(root string, base string, head string, currentSymbols []sherpa.Symbol, useCurrentSymbols bool) ([]changedSymbol, error) {
	changedLines, err := gitdiff.ChangedLineRanges(root, base, head)
	if err != nil {
		return nil, err
	}

	return symbolsForChangedLineRangesWithCurrentSymbols(root, base, head, changedLines, currentSymbols, useCurrentSymbols)
}

func (a Analyzer) analyzeChangedSymbolImpacts(symbols []changedSymbol) changedSymbolImpact {
	return a.analyzeChangedSymbolImpactsWithContext(symbols, nil)
}

func (a Analyzer) analyzeChangedSymbolImpactsWithContext(symbols []changedSymbol, context *sherpa.SemanticContext) changedSymbolImpact {
	symbols = normalizeChangedSymbols(symbols)
	if len(symbols) == 0 {
		return changedSymbolImpact{}
	}

	modulePath := impactModulePath(a.Root)
	seenTests := make(map[string]sherpa.RelatedTest)
	targets := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		target := changedSymbolTestTarget(symbol, modulePath)
		if strings.TrimSpace(target) == "" {
			continue
		}
		targets = append(targets, target)
	}

	var impact changedSymbolImpact
	impactOptions := sherpa.ImpactOptions{BuildTags: a.BuildTags}
	var batches []sherpa.ImpactBatchResult
	if context != nil {
		batches = sherpa.FindSymbolImpactSignalsWithContext(context, targets, impactOptions)
	} else {
		batches = sherpa.FindSymbolImpactSignalsWithOptions(a.Root, targets, impactOptions)
	}
	for _, batch := range batches {
		if batch.Err != nil {
			impact.Warnings = append(impact.Warnings, fmt.Sprintf("symbol impact unavailable for %s: %v", batch.Target, batch.Err))
			continue
		}

		result := batch.Result
		impact.Packages = append(impact.Packages, result.Packages...)
		for _, test := range result.RelatedTests {
			mergeRelatedTest(seenTests, test)
		}
		impact.ReferenceAnalysisMode = mergeReferenceAnalysisMode(impact.ReferenceAnalysisMode, result.ReferenceAnalysisMode)
		impact.CallAnalysisMode = mergeCallAnalysisMode(impact.CallAnalysisMode, result.CallAnalysisMode)
		impact.TestAnalysisMode = mergeTestAnalysisMode(impact.TestAnalysisMode, result.TestAnalysisMode)
		impact.Warnings = append(impact.Warnings, result.Warnings...)
	}

	impact.Packages = uniqueSortedStrings(impact.Packages)
	impact.Warnings = uniqueSortedStrings(impact.Warnings)
	for _, test := range seenTests {
		impact.Tests = append(impact.Tests, test)
	}
	sortRelatedTests(impact.Tests)

	return impact
}

func affectedTestsForPackages(root string, changedPackages []string, packages []string, changedSymbols []changedSymbol, extraTests []sherpa.RelatedTest, contractPackages []string, warnings []string) ([]sherpa.RelatedTest, sherpa.TestPlan, []string, string, []string) {
	return affectedTestsForPackagesWithContext(nil, root, changedPackages, packages, changedSymbols, extraTests, contractPackages, warnings)
}

func affectedTestsForPackagesWithContext(context *sherpa.SemanticContext, root string, changedPackages []string, packages []string, changedSymbols []changedSymbol, extraTests []sherpa.RelatedTest, contractPackages []string, warnings []string) ([]sherpa.RelatedTest, sherpa.TestPlan, []string, string, []string) {
	seen := make(map[string]sherpa.RelatedTest)
	modulePath := impactModulePath(root)
	changedTargets := changedSymbolPlanTargets(changedSymbols, modulePath)
	changedTargetsByPackage := changedSymbolPlanTargetsByPackage(changedSymbols, modulePath)
	changedPackageSet := impactStringSet(changedPackages)
	contractPackageSet := impactStringSet(contractPackages)
	testAnalysisMode := ""

	directTests, directTestAnalysisMode := directTestsForChangedSymbolsWithContext(context, root, changedSymbols, &warnings)
	testAnalysisMode = mergeTestAnalysisMode(testAnalysisMode, directTestAnalysisMode)
	for _, test := range directTests {
		mergeRelatedTest(seen, test)
	}
	for _, test := range extraTests {
		mergeRelatedTest(seen, test)
	}

	for _, pkg := range packages {
		tests, err := findTestsWithContext(context, root, pkg, sherpa.TestOptions{Scope: sherpa.TestScopeAll})
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		testAnalysisMode = mergeTestAnalysisMode(testAnalysisMode, tests.AnalysisMode)
		warnings = append(warnings, tests.Warnings...)

		for _, test := range tests.Tests {
			if targets := changedTargetsByPackage[pkg]; len(targets) > 0 {
				test.Targets = uniqueSortedStrings(append(test.Targets, targets...))
				test = addRelatedTestReason(test, sherpa.RelatedTestReasonChangedSymbol)
			}
			if _, ok := changedPackageSet[pkg]; !ok {
				test = removeRelatedTestReason(test, sherpa.RelatedTestReasonTargetPackage)
				test = addRelatedTestReason(test, sherpa.RelatedTestReasonCallerPackage)
			}
			if _, ok := contractPackageSet[pkg]; ok {
				test = addRelatedTestReason(test, sherpa.RelatedTestReasonContract)
			}
			mergeRelatedTest(seen, test)
		}
	}

	result := make([]sherpa.RelatedTest, 0, len(seen))
	for _, test := range seen {
		result = append(result, test)
	}

	sortRelatedTests(result)
	target := "affected packages"
	if len(changedSymbols) > 0 {
		target = "changed symbols"
	}
	plan := sherpa.PlanTests(result, sherpa.TestPlanOptions{
		Target:           target,
		Kind:             sherpa.TestTargetKindPackage,
		TargetPackages:   changedPackages,
		ContractPackages: contractPackages,
		CallerPackages:   packageDifference(packages, changedPackages),
		FallbackPackages: packages,
		Targets:          changedTargets,
	})

	return result, plan, sherpa.TestPlanCommands(plan), normalizeTestAnalysisMode(testAnalysisMode), uniqueSortedStrings(warnings)
}

func mergeReferenceAnalysisMode(current string, next string) string {
	return mergeAnalysisMode(current, next, sherpa.ReferenceAnalysisModeTypechecked, sherpa.ReferenceAnalysisModeASTFallback)
}

func mergeCallAnalysisMode(current string, next string) string {
	return mergeAnalysisMode(current, next, sherpa.CallAnalysisModeTypechecked, sherpa.CallAnalysisModeASTFallback)
}

func mergeTestAnalysisMode(current string, next string) string {
	return mergeAnalysisMode(current, next, sherpa.TestAnalysisModeTypecheckedAST, sherpa.TestAnalysisModeAST)
}

func normalizeTestAnalysisMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return sherpa.TestAnalysisModeAST
	}

	return mode
}

func mergeAnalysisMode(current string, next string, typechecked string, fallback string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if next == "" {
		return current
	}
	if current == "" {
		return next
	}
	if current == typechecked || next == typechecked {
		return typechecked
	}
	if current == fallback || next == fallback {
		return fallback
	}

	return current
}

func directTestsForChangedSymbols(root string, symbols []changedSymbol, warnings *[]string) ([]sherpa.RelatedTest, string) {
	return directTestsForChangedSymbolsWithContext(nil, root, symbols, warnings)
}

func directTestsForChangedSymbolsWithContext(context *sherpa.SemanticContext, root string, symbols []changedSymbol, warnings *[]string) ([]sherpa.RelatedTest, string) {
	if len(symbols) == 0 {
		return nil, ""
	}

	modulePath := impactModulePath(root)
	seen := make(map[string]sherpa.RelatedTest)
	testAnalysisMode := ""
	for _, symbol := range normalizeChangedSymbols(symbols) {
		target := changedSymbolTestTarget(symbol, modulePath)
		tests, err := findTestsWithContext(context, root, target, sherpa.TestOptions{
			Scope: sherpa.TestScopeDirect,
		})
		if err != nil {
			if warnings != nil {
				*warnings = append(*warnings, err.Error())
			}
			continue
		}
		testAnalysisMode = mergeTestAnalysisMode(testAnalysisMode, tests.AnalysisMode)
		if warnings != nil {
			*warnings = append(*warnings, tests.Warnings...)
		}

		for _, test := range tests.Tests {
			test.Targets = uniqueSortedStrings(append(test.Targets, target))
			test = addRelatedTestReason(test, sherpa.RelatedTestReasonChangedSymbol)
			mergeRelatedTest(seen, test)
		}
	}

	result := make([]sherpa.RelatedTest, 0, len(seen))
	for _, test := range seen {
		result = append(result, test)
	}
	sortRelatedTests(result)

	return result, testAnalysisMode
}

func findTestsWithContext(context *sherpa.SemanticContext, root string, target string, options sherpa.TestOptions) (sherpa.TestsResult, error) {
	if context != nil {
		return sherpa.FindTestsWithContext(context, target, options)
	}

	return sherpa.FindTestsWithOptions(root, target, options)
}

func changedSymbolPlanTargets(symbols []changedSymbol, modulePath string) []string {
	var targets []string
	for _, symbol := range normalizeChangedSymbols(symbols) {
		target := changedSymbolTestTarget(symbol, modulePath)
		if strings.TrimSpace(target) == "" {
			continue
		}

		targets = append(targets, target)
	}

	return uniqueSortedStrings(targets)
}

func changedSymbolPlanTargetsByPackage(symbols []changedSymbol, modulePath string) map[string][]string {
	targetsByPackage := make(map[string][]string)
	for _, symbol := range normalizeChangedSymbols(symbols) {
		target := changedSymbolTestTarget(symbol, modulePath)
		if strings.TrimSpace(target) == "" {
			continue
		}

		targetsByPackage[symbol.Package] = append(targetsByPackage[symbol.Package], target)
	}

	for pkg, targets := range targetsByPackage {
		targetsByPackage[pkg] = uniqueSortedStrings(targets)
	}

	return targetsByPackage
}

func changedSymbolTestTarget(symbol changedSymbol, modulePath string) string {
	if symbol.Package == "." && !strings.Contains(modulePath, "/") {
		return symbol.Name
	}

	return sherpa.FormatPackageQualifiedTarget(symbol.Package, symbol.Name, modulePath)
}

func mergeRelatedTest(seen map[string]sherpa.RelatedTest, test sherpa.RelatedTest) {
	key := relatedTestKey(test)
	existing, ok := seen[key]
	if !ok {
		seen[key] = test
		return
	}

	test.DirectReference = test.DirectReference || existing.DirectReference
	test.ExternalPackage = test.ExternalPackage || existing.ExternalPackage
	test.Targets = uniqueSortedStrings(append(test.Targets, existing.Targets...))
	test.TargetReferences = mergeRelatedTestTargetReferences(test.TargetReferences, existing.TargetReferences)
	test.Reasons = mergeRelatedTestReasons(test.Reasons, existing.Reasons)
	if test.Range == nil {
		test.Range = existing.Range
	}

	seen[key] = test
}

func mergeRelatedTestTargetReferences(first []sherpa.RelatedTestTargetReference, second []sherpa.RelatedTestTargetReference) []sherpa.RelatedTestTargetReference {
	byKey := make(map[string]sherpa.RelatedTestTargetReference)
	for _, ref := range append(first, second...) {
		ref.Target = strings.TrimSpace(ref.Target)
		if ref.Target == "" {
			continue
		}

		key := relatedTestTargetReferenceKey(ref)
		if key == "" {
			continue
		}
		if existing, ok := byKey[key]; ok && existing.Range != nil {
			continue
		}
		byKey[key] = ref
	}

	refs := make([]sherpa.RelatedTestTargetReference, 0, len(byKey))
	for _, ref := range byKey {
		refs = append(refs, ref)
	}

	sort.Slice(refs, func(i int, j int) bool {
		if refs[i].Target != refs[j].Target {
			return refs[i].Target < refs[j].Target
		}
		if refs[i].Position.File != refs[j].Position.File {
			return refs[i].Position.File < refs[j].Position.File
		}
		if refs[i].Position.Line != refs[j].Position.Line {
			return refs[i].Position.Line < refs[j].Position.Line
		}
		if refs[i].Position.Column != refs[j].Position.Column {
			return refs[i].Position.Column < refs[j].Position.Column
		}
		if refs[i].Range == nil || refs[j].Range == nil {
			return refs[i].Range != nil
		}
		if refs[i].Range.End.Line != refs[j].Range.End.Line {
			return refs[i].Range.End.Line < refs[j].Range.End.Line
		}

		return refs[i].Range.End.Column < refs[j].Range.End.Column
	})

	return refs
}

func relatedTestTargetReferenceKey(ref sherpa.RelatedTestTargetReference) string {
	if ref.Target == "" {
		return ""
	}
	if ref.Position.File == "" || ref.Position.Line <= 0 {
		return ref.Target
	}

	key := ref.Target + ":" + ref.Position.File + ":" + strconv.Itoa(ref.Position.Line) + ":" + strconv.Itoa(ref.Position.Column)
	if ref.Range != nil {
		key += ":" + strconv.Itoa(ref.Range.End.Line) + ":" + strconv.Itoa(ref.Range.End.Column)
	}

	return key
}

func impactModulePath(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1]
		}
	}

	return ""
}

func relatedTestKey(test sherpa.RelatedTest) string {
	parts := []string{
		test.Package,
		test.PackageName,
		test.Name,
		test.Position.File,
		intKey(test.Position.Line),
	}

	return strings.Join(parts, "\x00")
}

func addRelatedTestReason(test sherpa.RelatedTest, reason string) sherpa.RelatedTest {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return test
	}
	for _, existing := range test.Reasons {
		if existing == reason {
			return test
		}
	}

	test.Reasons = append(test.Reasons, reason)
	return test
}

func removeRelatedTestReason(test sherpa.RelatedTest, reason string) sherpa.RelatedTest {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(test.Reasons) == 0 {
		return test
	}

	reasons := test.Reasons[:0]
	for _, existing := range test.Reasons {
		if existing == reason {
			continue
		}
		reasons = append(reasons, existing)
	}
	test.Reasons = reasons

	return test
}

func mergeRelatedTestReasons(first []string, second []string) []string {
	result := append([]string{}, first...)
	for _, reason := range second {
		reason = strings.TrimSpace(reason)
		if reason == "" {
			continue
		}
		seen := false
		for _, existing := range result {
			if existing == reason {
				seen = true
				break
			}
		}
		if !seen {
			result = append(result, reason)
		}
	}

	return result
}

func contractPackagesForSignals(signals interfaceImpactSignals) []string {
	return uniqueSortedStrings(append(signalPackages(signals.Interfaces), signalPackages(signals.Implementations)...))
}

func contractTargetsByPackage(signals interfaceImpactSignals) map[string][]string {
	result := make(map[string][]string)
	for _, target := range append(append([]string{}, signals.Interfaces...), signals.Implementations...) {
		pkg := packageForQualifiedSymbol(target)
		if pkg == "" {
			continue
		}
		result[pkg] = append(result[pkg], target)
	}
	for pkg, targets := range result {
		result[pkg] = uniqueSortedStrings(targets)
	}

	return result
}

func signalPackages(targets []string) []string {
	var packages []string
	for _, target := range targets {
		if pkg := packageForQualifiedSymbol(target); pkg != "" {
			packages = append(packages, pkg)
		}
	}

	return packages
}

func symbolTestPlanTargetPackages(target string, plan sherpa.TestPlan) []string {
	packages := packagesFromTestPlanItems(plan.Related)
	if len(packages) > 0 {
		return packages
	}

	return nonEmptyStrings(packageForQualifiedSymbol(target))
}

func packagesFromTestPlanItems(items []sherpa.TestPlanItem) []string {
	var packages []string
	for _, item := range items {
		packages = append(packages, item.Package)
	}

	return uniqueSortedStrings(packages)
}

func packageForQualifiedSymbol(target string) string {
	value := strings.TrimSpace(filepath.ToSlash(target))
	if value == "" {
		return ""
	}

	lastSlash := strings.LastIndex(value, "/")
	if lastSlash >= 0 {
		firstDotAfterSlash := strings.Index(value[lastSlash+1:], ".")
		if firstDotAfterSlash >= 0 {
			return value[:lastSlash+1+firstDotAfterSlash]
		}
	}

	if strings.HasPrefix(value, "./") {
		lastDot := strings.LastIndex(value, ".")
		if lastDot > 0 {
			return value[:lastDot]
		}
	}

	if !strings.Contains(value, "/") {
		return "."
	}

	return ""
}

func firstNonEmptyImpactString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

func nonEmptyStrings(values ...string) []string {
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}

	return result
}

func impactStringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}

	return set
}

func sortRelatedTests(tests []sherpa.RelatedTest) {
	sort.Slice(tests, func(i int, j int) bool {
		if tests[i].Package != tests[j].Package {
			return tests[i].Package < tests[j].Package
		}
		if tests[i].Position.File != tests[j].Position.File {
			return tests[i].Position.File < tests[j].Position.File
		}
		if tests[i].Position.Line != tests[j].Position.Line {
			return tests[i].Position.Line < tests[j].Position.Line
		}

		return tests[i].Name < tests[j].Name
	})
}

func normalizeReport(report ImpactReport) ImpactReport {
	report.ChangedFiles = nonNilStrings(report.ChangedFiles)
	report.ChangedPackages = nonNilStrings(report.ChangedPackages)
	report.AffectedPackages = nonNilStrings(report.AffectedPackages)
	report.AffectedSymbols = nonNilStrings(report.AffectedSymbols)
	report.ChangedSymbolDetails = nonNilChangedSymbols(report.ChangedSymbolDetails)
	report.ReferenceAnalysisMode = strings.TrimSpace(report.ReferenceAnalysisMode)
	report.CallAnalysisMode = strings.TrimSpace(report.CallAnalysisMode)
	report.AffectedInterfaces = nonNilStrings(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilStrings(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.TestCommands = nonNilStrings(report.TestCommands)
	report.TestAnalysisMode = strings.TrimSpace(report.TestAnalysisMode)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Warnings = nonNilStrings(report.Warnings)

	if report.AffectedTests == nil {
		report.AffectedTests = []sherpa.RelatedTest{}
	}

	return report
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		seen[value] = struct{}{}
	}

	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}

func nonNilChangedSymbols(values []ChangedSymbol) []ChangedSymbol {
	if values == nil {
		return []ChangedSymbol{}
	}

	return values
}

func packageDifference(values []string, excluded []string) []string {
	excludedSet := make(map[string]struct{}, len(excluded))
	for _, value := range excluded {
		excludedSet[value] = struct{}{}
	}

	var result []string
	for _, value := range values {
		if _, ok := excludedSet[value]; ok {
			continue
		}

		result = append(result, value)
	}

	return uniqueSortedStrings(result)
}

func intKey(value int) string {
	return strconv.Itoa(value)
}
