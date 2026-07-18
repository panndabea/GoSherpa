package sherpa

import (
	"fmt"
	"go/token"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type ImpactKind string

const (
	ImpactKindSymbol  ImpactKind = "symbol"
	ImpactKindPackage ImpactKind = "package"
)

type ImpactResult struct {
	Target                string              `json:"target"`
	Kind                  ImpactKind          `json:"kind"`
	TargetRisk            TargetRiskSummary   `json:"targetRisk"`
	References            []Reference         `json:"references"`
	ReferenceAnalysisMode string              `json:"referenceAnalysisMode,omitempty"`
	Callers               []Caller            `json:"callers"`
	CallAnalysisMode      string              `json:"callAnalysisMode,omitempty"`
	Dependencies          PackageDependencies `json:"dependencies"`
	Packages              []string            `json:"packages"`
	RelatedTests          []RelatedTest       `json:"relatedTests"`
	TestAnalysisMode      string              `json:"testAnalysisMode,omitempty"`
	TestCommands          []string            `json:"testCommands"`
	TestPlan              TestPlan            `json:"testPlan"`
	Warnings              []string            `json:"warnings"`
}

type ImpactBatchResult struct {
	Target string
	Result ImpactResult
	Err    error
}

type ImpactOptions struct {
	BuildTags []string
}

type impactAnalysisCache struct {
	References         *referenceAnalysisCache
	Context            *SemanticContext
	SkipTests          bool
	CallFunctions      []functionInfo
	CallAnalysisMode   string
	CallWarnings       []string
	CallErr            error
	CallFunctionsReady bool
}

func FindImpact(root string, target string) (ImpactResult, error) {
	return FindImpactWithOptions(root, target, ImpactOptions{})
}

func FindImpactWithOptions(root string, target string, options ImpactOptions) (ImpactResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return ImpactResult{}, err
	}

	if isImpactPackageTarget(target) {
		return findPackageImpact(rootPath, target)
	}

	return findSymbolImpact(rootPath, target, options)
}

func FindImpactWithContext(context *SemanticContext, target string, options ImpactOptions) (ImpactResult, error) {
	if context == nil {
		return ImpactResult{}, fmt.Errorf("semantic context is nil")
	}
	if !context.supportsBuildTags(options.BuildTags) {
		return ImpactResult{}, fmt.Errorf("semantic context build tags do not match impact options")
	}

	if isImpactPackageTarget(target) {
		return findPackageImpact(context.root, target)
	}

	return findSymbolImpactWithCache(context.root, target, options, newImpactAnalysisCacheWithContext(context, options))
}

func FindSymbolImpactSignalsWithOptions(root string, targets []string, options ImpactOptions) []ImpactBatchResult {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return impactBatchErrorResults(targets, err)
	}

	cache := newImpactAnalysisCache(rootPath, options)
	cache.SkipTests = true
	results := make([]ImpactBatchResult, 0, len(targets))
	for _, target := range targets {
		result, err := findSymbolImpactWithCache(rootPath, target, options, cache)
		results = append(results, ImpactBatchResult{
			Target: target,
			Result: result,
			Err:    err,
		})
	}

	return results
}

func FindSymbolImpactSignalsWithContext(context *SemanticContext, targets []string, options ImpactOptions) []ImpactBatchResult {
	if context == nil {
		return impactBatchErrorResults(targets, fmt.Errorf("semantic context is nil"))
	}
	if !context.supportsBuildTags(options.BuildTags) {
		return impactBatchErrorResults(targets, fmt.Errorf("semantic context build tags do not match impact options"))
	}

	cache := newImpactAnalysisCacheWithContext(context, options)
	cache.SkipTests = true
	results := make([]ImpactBatchResult, 0, len(targets))
	for _, target := range targets {
		result, err := findSymbolImpactWithCache(context.root, target, options, cache)
		results = append(results, ImpactBatchResult{
			Target: target,
			Result: result,
			Err:    err,
		})
	}

	return results
}

func impactBatchErrorResults(targets []string, err error) []ImpactBatchResult {
	results := make([]ImpactBatchResult, 0, len(targets))
	for _, target := range targets {
		results = append(results, ImpactBatchResult{
			Target: target,
			Err:    err,
		})
	}

	return results
}

func newImpactAnalysisCache(root string, options ImpactOptions) *impactAnalysisCache {
	return &impactAnalysisCache{
		References: newReferenceAnalysisCache(root, ReferenceOptions{
			BuildTags: options.BuildTags,
		}),
	}
}

func newImpactAnalysisCacheWithContext(context *SemanticContext, options ImpactOptions) *impactAnalysisCache {
	if context == nil {
		return &impactAnalysisCache{}
	}

	return &impactAnalysisCache{
		Context: context,
		References: context.referenceAnalysisCache(ReferenceOptions{
			BuildTags: options.BuildTags,
		}),
	}
}

func isImpactPackageTarget(target string) bool {
	value := strings.TrimSpace(target)
	if hasPackageQualifiedSymbolShape(value) {
		return false
	}
	if value == "." || strings.HasPrefix(value, "./") {
		return true
	}

	return strings.Contains(value, "/") || strings.Contains(value, "\\")
}

func hasPackageQualifiedSymbolShape(target string) bool {
	value := strings.TrimSpace(filepath.ToSlash(target))

	lastSlash := strings.LastIndex(value, "/")
	if lastSlash < 0 {
		return false
	}

	firstDotAfterSlash := strings.Index(value[lastSlash+1:], ".")
	if firstDotAfterSlash < 0 {
		return false
	}

	symbol := value[lastSlash+1+firstDotAfterSlash+1:]
	segments := strings.Split(symbol, ".")
	if len(segments) != 1 && len(segments) != 2 {
		return false
	}

	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(segment) {
			return false
		}
	}

	return true
}

func findPackageImpact(root string, target string) (ImpactResult, error) {
	deps, err := FindPackageDependencies(root, target)
	if err != nil {
		return ImpactResult{}, err
	}

	packages := uniqueSorted(append([]string{deps.Package}, deps.UsedBy...))
	tests, warnings := impactTestsForPackages(root, packages)
	tests = annotateCallerPackageTestReasons(tests, []string{deps.Package})
	plan := PlanTests(tests, TestPlanOptions{
		Target:            deps.Package,
		Kind:              TestTargetKindPackage,
		TargetPackages:    []string{deps.Package},
		CallerPackages:    deps.UsedBy,
		FallbackPackages:  packages,
		RepositorySignals: AnalyzeTestPlanRepositorySignals(root, packages, warnings),
	})

	result := ImpactResult{
		Target:           deps.Package,
		Kind:             ImpactKindPackage,
		Dependencies:     deps,
		Packages:         packages,
		RelatedTests:     tests,
		TestAnalysisMode: TestAnalysisModeAST,
		TestCommands:     TestPlanCommands(plan),
		TestPlan:         plan,
		Warnings:         warnings,
	}
	result.TargetRisk = packageTargetRisk(result)

	return result, nil
}

func findSymbolImpact(root string, target string, options ImpactOptions) (ImpactResult, error) {
	return findSymbolImpactWithCache(root, target, options, nil)
}

func findSymbolImpactWithCache(root string, target string, options ImpactOptions, cache *impactAnalysisCache) (ImpactResult, error) {
	normalizedTarget, err := normalizeReferenceTarget(root, target)
	if err != nil {
		return ImpactResult{}, err
	}

	var referenceCache *referenceAnalysisCache
	if cache != nil {
		referenceCache = cache.References
	}
	referenceReport, err := findReferenceReportForTarget(root, normalizedTarget, ReferenceOptions{
		BuildTags: options.BuildTags,
	}, referenceCache)
	if err != nil {
		return ImpactResult{}, err
	}

	result := ImpactResult{
		Target:                normalizedTarget.String(),
		Kind:                  ImpactKindSymbol,
		References:            referenceReport.References,
		ReferenceAnalysisMode: referenceReport.AnalysisMode,
		Warnings:              referenceReport.Warnings,
	}

	callers, analysisMode, warnings, err := impactSymbolCallersWithCache(root, target, options, cache)
	result.Warnings = append(result.Warnings, warnings...)
	if err == nil {
		result.CallAnalysisMode = analysisMode
		result.Callers = callers
	} else if !isImpactNonFunctionTargetError(err) {
		result.CallAnalysisMode = analysisMode
		result.Warnings = append(result.Warnings, err.Error())
	}

	result.Packages = impactedPackages(root, result.References, result.Callers)

	targetPackageSet, err := referenceTargetPackages(root, normalizedTarget)
	var targetPackages []string
	if err == nil {
		targetPackages = sortedMapKeys(targetPackageSet)
	} else {
		result.Warnings = append(result.Warnings, err.Error())
	}

	if cache == nil || !cache.SkipTests {
		tests, warnings := impactSymbolTestsWithCache(root, target, result.Packages, targetPackages, cache)
		result.RelatedTests = tests.Tests
		result.TestAnalysisMode = tests.AnalysisMode
		result.TestCommands = tests.Commands
		result.TestPlan = tests.TestPlan
		result.Warnings = append(result.Warnings, tests.Warnings...)
		result.Warnings = append(result.Warnings, warnings...)
	}
	result.TargetRisk = symbolTargetRisk(root, result, targetPackages, options, cache)

	return result, nil
}

func packageTargetRisk(result ImpactResult) TargetRiskSummary {
	signals := TargetRiskSignals{
		AffectedPackages:   len(result.Packages),
		PackageFanIn:       len(result.Dependencies.UsedBy),
		MissingDirectTests: !hasDirectRelatedTest(result.RelatedTests),
		FallbackTests:      len(result.TestPlan.Fallback) > 0,
		Warnings:           len(result.Warnings),
	}

	score := 1
	reasons := []string{TargetRiskReasonAffectedPackages}
	scope := TargetRiskScopePackage
	if signals.AffectedPackages > 1 {
		score += 2
		scope = TargetRiskScopeCrossPackage
	}
	if signals.PackageFanIn > 0 {
		score += 1
		reasons = append(reasons, TargetRiskReasonPackageFanIn)
	}
	if signals.PackageFanIn >= 5 {
		score += 2
	}
	if signals.MissingDirectTests {
		score++
		reasons = append(reasons, TargetRiskReasonMissingDirectTests)
	}
	if signals.FallbackTests {
		score++
		reasons = append(reasons, TargetRiskReasonFallbackTests)
	}
	if signals.Warnings > 0 {
		score += 2
		reasons = append(reasons, TargetRiskReasonAnalysisWarning)
	}

	return NormalizeTargetRiskSummary(TargetRiskSummary{
		Level:   TargetRiskLevelForScore(score),
		Scope:   scope,
		Reasons: reasons,
		Signals: signals,
		Limitations: []string{
			"Target risk summarizes deterministic impact breadth; it is not a defect prediction.",
			"Package risk uses local import fan-in and affected-package/test signals only.",
		},
	})
}

func symbolTargetRisk(root string, result ImpactResult, targetPackages []string, options ImpactOptions, cache *impactAnalysisCache) TargetRiskSummary {
	symbol, symbolFound := impactTargetSymbol(root, result.Target)
	signals := TargetRiskSignals{
		AffectedPackages:     len(result.Packages),
		DirectReferences:     directReferenceCount(result.References),
		TransitiveCallers:    len(result.Callers),
		CallerPackages:       callerPackageCount(result.Callers),
		ExportedSymbol:       symbolFound && impactSymbolExported(symbol),
		ExportedTypeMethod:   symbolFound && impactSymbolExportedTypeMethod(symbol),
		PackageFanIn:         targetPackageFanIn(root, targetPackages),
		MissingDirectTests:   !hasDirectRelatedTest(result.RelatedTests),
		FallbackTests:        len(result.TestPlan.Fallback) > 0,
		PossibleRuntimeCalls: possibleRuntimeCallCount(root, result.Target, options, cache),
		Warnings:             len(result.Warnings),
	}

	score := 0
	var reasons []string
	scope := TargetRiskScopeLocal
	if signals.AffectedPackages > 1 {
		score += 2
		reasons = append(reasons, TargetRiskReasonAffectedPackages)
		scope = TargetRiskScopeCrossPackage
	} else if signals.AffectedPackages == 1 {
		score++
		scope = TargetRiskScopePackage
	}
	if signals.DirectReferences > 0 {
		score++
		reasons = append(reasons, TargetRiskReasonDirectReferences)
	}
	if signals.DirectReferences >= 5 {
		score++
	}
	if signals.TransitiveCallers > 0 {
		score++
		reasons = append(reasons, TargetRiskReasonTransitiveCallers)
	}
	if signals.CallerPackages > 1 {
		score++
		scope = TargetRiskScopeCrossPackage
	}
	if signals.ExportedTypeMethod {
		score += 3
		reasons = append(reasons, TargetRiskReasonExportedTypeMethod)
		scope = TargetRiskScopeExportedAPI
	} else if signals.ExportedSymbol {
		score += 2
		reasons = append(reasons, TargetRiskReasonExportedSymbol)
		scope = TargetRiskScopeExportedAPI
	}
	if signals.PackageFanIn > 0 {
		score++
		reasons = append(reasons, TargetRiskReasonPackageFanIn)
	}
	if signals.PackageFanIn >= 5 {
		score += 2
	}
	if signals.MissingDirectTests {
		score++
		reasons = append(reasons, TargetRiskReasonMissingDirectTests)
	}
	if signals.FallbackTests {
		score++
		reasons = append(reasons, TargetRiskReasonFallbackTests)
	}
	if signals.PossibleRuntimeCalls > 0 {
		score++
		reasons = append(reasons, TargetRiskReasonPossibleRuntimeCalls)
	}
	if signals.Warnings > 0 {
		score += 2
		reasons = append(reasons, TargetRiskReasonAnalysisWarning)
	}
	if len(reasons) == 0 {
		reasons = append(reasons, TargetRiskReasonAffectedPackages)
	}

	return NormalizeTargetRiskSummary(TargetRiskSummary{
		Level:   TargetRiskLevelForScore(score),
		Scope:   scope,
		Reasons: reasons,
		Signals: signals,
		Limitations: []string{
			"Target risk summarizes deterministic impact breadth; it is not a defect prediction.",
			"Possible runtime calls count only bounded static signals and do not include full pointer or reflection analysis.",
		},
	})
}

func impactTargetSymbol(root string, target string) (Symbol, bool) {
	symbols, err := ParseRepository(root)
	if err != nil {
		return Symbol{}, false
	}

	symbol, err := FindSymbolTarget(root, symbols, target)
	if err != nil {
		return Symbol{}, false
	}

	return symbol, true
}

func impactSymbolExported(symbol Symbol) bool {
	return token.IsExported(symbol.Name)
}

func impactSymbolExportedTypeMethod(symbol Symbol) bool {
	if symbol.Kind != SymbolKindMethod {
		return false
	}

	return token.IsExported(symbol.Name) || token.IsExported(symbol.Receiver)
}

func directReferenceCount(references []Reference) int {
	count := 0
	for _, reference := range references {
		if reference.Kind == ReferenceKindDefinition {
			continue
		}
		count++
	}

	return count
}

func callerPackageCount(callers []Caller) int {
	packages := make(map[string]struct{})
	for _, caller := range callers {
		pkg, ok := impactPackageForFile(caller.Position.File)
		if !ok {
			continue
		}
		packages[pkg] = struct{}{}
	}

	return len(packages)
}

func impactPackageForFile(file string) (string, bool) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", false
	}

	file = filepath.ToSlash(file)
	file = path.Clean(file)
	if file == "." || path.IsAbs(file) || file == ".." || strings.HasPrefix(file, "../") {
		return "", false
	}
	if path.Ext(file) != ".go" {
		return "", false
	}

	dir := path.Dir(file)
	if dir == "." {
		return ".", true
	}

	return "./" + dir, true
}

func targetPackageFanIn(root string, packages []string) int {
	maxFanIn := 0
	for _, pkg := range uniqueSorted(packages) {
		deps, err := FindPackageDependencies(root, pkg)
		if err != nil {
			continue
		}
		if len(deps.UsedBy) > maxFanIn {
			maxFanIn = len(deps.UsedBy)
		}
	}

	return maxFanIn
}

func hasDirectRelatedTest(tests []RelatedTest) bool {
	for _, test := range tests {
		if test.DirectReference {
			return true
		}
	}

	return false
}

func possibleRuntimeCallCount(root string, target string, options ImpactOptions, cache *impactAnalysisCache) int {
	normalizedTarget, err := normalizeCallTarget(root, target)
	if err != nil {
		return 0
	}

	functions, _, _, err := collectImpactCallFunctionInfos(root, options, cache)
	if err != nil {
		return 0
	}
	function, err := findFunctionInfo(functions, normalizedTarget)
	if err != nil {
		return 0
	}

	possibleCalls := collectPossibleCallsFromFunctionWithFunctions(function, functions, newInterfaceDispatchCatalog(functions))
	return len(possibleCalls)
}

func impactSymbolCallers(root string, target string, options ImpactOptions) ([]Caller, string, []string, error) {
	return impactSymbolCallersWithCache(root, target, options, nil)
}

func impactSymbolCallersWithCache(root string, target string, options ImpactOptions, cache *impactAnalysisCache) ([]Caller, string, []string, error) {
	normalizedTarget, err := normalizeCallTarget(root, target)
	if err != nil {
		return nil, "", nil, err
	}

	functions, analysisMode, warnings, err := collectImpactCallFunctionInfos(root, options, cache)
	if err != nil {
		return nil, analysisMode, warnings, err
	}

	callers, err := collectTransitiveCallersFromFunctions(functions, normalizedTarget)
	return callers, analysisMode, warnings, err
}

func collectImpactCallFunctionInfos(root string, options ImpactOptions, cache *impactAnalysisCache) ([]functionInfo, string, []string, error) {
	if cache == nil {
		return collectCallFunctionInfos(root, CallOptions{
			BuildTags: options.BuildTags,
		})
	}

	if !cache.CallFunctionsReady {
		cache.CallFunctionsReady = true
		callOptions := CallOptions{
			BuildTags: options.BuildTags,
		}
		if cache.Context != nil {
			cache.CallFunctions, cache.CallAnalysisMode, cache.CallWarnings, cache.CallErr = collectCallFunctionInfosWithContext(cache.Context, callOptions)
		} else {
			cache.CallFunctions, cache.CallAnalysisMode, cache.CallWarnings, cache.CallErr = collectCallFunctionInfos(root, callOptions)
		}
	}

	return cache.CallFunctions, cache.CallAnalysisMode, cache.CallWarnings, cache.CallErr
}

func impactSymbolTests(root string, target string, packages []string, targetPackages []string) (TestsResult, []string) {
	return impactSymbolTestsWithCache(root, target, packages, targetPackages, nil)
}

func impactSymbolTestsWithCache(root string, target string, packages []string, targetPackages []string, cache *impactAnalysisCache) (TestsResult, []string) {
	symbolTests, warnings := impactTestsWithCache(root, target, cache)
	mergedTests := symbolTests.Tests

	packageTests, packageWarnings := impactTestsForPackages(root, packages)
	warnings = append(warnings, packageWarnings...)
	if len(targetPackages) == 0 {
		targetPackages = sortedTestPackages(symbolTests.Tests)
	}
	packageTests = annotateCallerPackageTestReasons(packageTests, targetPackages)
	mergedTests = mergeRelatedTests(mergedTests, packageTests)

	sortRelatedTests(mergedTests)
	fallbackPackages := uniqueSorted(append(append([]string{}, targetPackages...), packages...))
	plan := PlanTests(mergedTests, TestPlanOptions{
		Target:            firstNonEmptyString(symbolTests.Target, target),
		Kind:              TestTargetKindSymbol,
		TargetPackages:    targetPackages,
		CallerPackages:    packageDifference(packages, targetPackages),
		FallbackPackages:  fallbackPackages,
		RepositorySignals: AnalyzeTestPlanRepositorySignals(root, fallbackPackages, warnings),
	})

	return TestsResult{
		Target:       symbolTests.Target,
		Kind:         TestTargetKindSymbol,
		AnalysisMode: symbolTests.AnalysisMode,
		Warnings:     nonNilStrings(symbolTests.Warnings),
		Tests:        mergedTests,
		Commands:     TestPlanCommands(plan),
		TestPlan:     plan,
	}, uniqueSorted(warnings)
}

func impactTestsForPackages(root string, packages []string) ([]RelatedTest, []string) {
	if len(packages) == 0 {
		return nil, nil
	}

	testFiles, err := collectTestFiles(root)
	if err != nil {
		return nil, []string{err.Error()}
	}

	packageSet := make(map[string]struct{})
	for _, pkg := range packages {
		packageSet[pkg] = struct{}{}
	}

	tests, _, _ := collectRelatedTests(root, testFiles, packageSet, referenceTarget{}, TestTargetKindPackage)
	return tests, nil
}

func impactTests(root string, target string) (TestsResult, []string) {
	return impactTestsWithCache(root, target, nil)
}

func impactTestsWithCache(root string, target string, cache *impactAnalysisCache) (TestsResult, []string) {
	if cache != nil && cache.Context != nil {
		tests, err := FindTestsWithContext(cache.Context, target, TestOptions{Scope: TestScopeAll})
		if err != nil {
			return TestsResult{}, []string{err.Error()}
		}

		return tests, nil
	}

	tests, err := FindTests(root, target)
	if err != nil {
		return TestsResult{}, []string{err.Error()}
	}

	return tests, nil
}

func annotateCallerPackageTestReasons(tests []RelatedTest, targetPackages []string) []RelatedTest {
	targetPackageSet := stringSet(targetPackages)
	if len(targetPackageSet) == 0 {
		return tests
	}

	result := make([]RelatedTest, 0, len(tests))
	for _, test := range tests {
		if _, ok := targetPackageSet[test.Package]; !ok {
			test.Reasons = removeRelatedTestReason(test.Reasons, RelatedTestReasonTargetPackage)
			test = annotateRelatedTestReason(test, RelatedTestReasonCallerPackage)
		}
		result = append(result, test)
	}

	return result
}

func mergeRelatedTests(existing []RelatedTest, incoming []RelatedTest) []RelatedTest {
	merged := make(map[string]RelatedTest)

	for _, test := range existing {
		merged[relatedTestImpactKey(test)] = test
	}

	for _, test := range incoming {
		key := relatedTestImpactKey(test)
		current, ok := merged[key]
		if ok {
			current.DirectReference = current.DirectReference || test.DirectReference
			current.ExternalPackage = current.ExternalPackage || test.ExternalPackage
			current.Targets = uniqueSorted(append(current.Targets, test.Targets...))
			current.TargetReferences = uniqueSortedRelatedTestTargetReferences(append(current.TargetReferences, test.TargetReferences...))
			current.Reasons = mergeRelatedTestReasons(current.Reasons, test.Reasons)
			merged[key] = current
			continue
		}

		merged[key] = test
	}

	tests := make([]RelatedTest, 0, len(merged))
	for _, test := range merged {
		tests = append(tests, test)
	}

	sortRelatedTests(tests)

	return tests
}

func relatedTestImpactKey(test RelatedTest) string {
	parts := []string{
		test.Package,
		test.PackageName,
		test.Name,
		test.Position.File,
		strconv.Itoa(test.Position.Line),
	}

	return strings.Join(parts, "\x00")
}

func mergeRelatedTestReasons(first []string, second []string) []string {
	result := append([]string{}, first...)
	for _, reason := range second {
		result = appendRelatedTestReason(result, reason)
	}

	return result
}

func removeRelatedTestReason(reasons []string, reason string) []string {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reasons) == 0 {
		return reasons
	}

	result := reasons[:0]
	for _, existing := range reasons {
		if existing == reason {
			continue
		}
		result = append(result, existing)
	}

	return result
}

func isImpactNonFunctionTargetError(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasPrefix(err.Error(), "function not found:")
}

func impactedPackages(root string, refs []Reference, callers []Caller) []string {
	var packages []string

	for _, ref := range refs {
		packages = appendImpactPackage(packages, root, ref.Position)
	}

	for _, caller := range callers {
		packages = appendImpactPackage(packages, root, caller.Position)
	}

	return uniqueSorted(packages)
}

func appendImpactPackage(packages []string, root string, position Position) []string {
	if position.File == "" {
		return packages
	}

	filePath := position.File
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(root, position.File)
	}

	packagePath, err := packagePathForFile(root, filePath)
	if err != nil {
		return packages
	}

	return append(packages, packagePath)
}
