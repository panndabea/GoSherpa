package sherpa

import (
	"go/token"
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
	Target       string              `json:"target"`
	Kind         ImpactKind          `json:"kind"`
	References   []Reference         `json:"references"`
	Callers      []Caller            `json:"callers"`
	Dependencies PackageDependencies `json:"dependencies"`
	Packages     []string            `json:"packages"`
	RelatedTests []RelatedTest       `json:"relatedTests"`
	TestCommands []string            `json:"testCommands"`
	TestPlan     TestPlan            `json:"testPlan"`
	Warnings     []string            `json:"warnings"`
}

type ImpactOptions struct {
	BuildTags []string
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
	plan := PlanTests(tests, TestPlanOptions{
		Target:           deps.Package,
		Kind:             TestTargetKindPackage,
		TargetPackages:   []string{deps.Package},
		CallerPackages:   deps.UsedBy,
		FallbackPackages: packages,
	})

	return ImpactResult{
		Target:       deps.Package,
		Kind:         ImpactKindPackage,
		Dependencies: deps,
		Packages:     packages,
		RelatedTests: tests,
		TestCommands: TestPlanCommands(plan),
		TestPlan:     plan,
		Warnings:     warnings,
	}, nil
}

func findSymbolImpact(root string, target string, options ImpactOptions) (ImpactResult, error) {
	normalizedTarget, err := normalizeReferenceTarget(root, target)
	if err != nil {
		return ImpactResult{}, err
	}

	referenceReport, err := FindReferenceReportWithOptions(root, target, ReferenceOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return ImpactResult{}, err
	}

	result := ImpactResult{
		Target:     normalizedTarget.String(),
		Kind:       ImpactKindSymbol,
		References: referenceReport.References,
		Warnings:   referenceReport.Warnings,
	}

	if normalizedTarget.Package == "" {
		callers, warnings, err := impactSymbolCallers(root, target, options)
		result.Warnings = append(result.Warnings, warnings...)
		if err == nil {
			result.Callers = callers
		} else if !isImpactNonFunctionTargetError(err) {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}

	result.Packages = impactedPackages(root, result.References, result.Callers)

	targetPackageSet, err := referenceTargetPackages(root, normalizedTarget)
	var targetPackages []string
	if err == nil {
		targetPackages = sortedMapKeys(targetPackageSet)
	} else {
		result.Warnings = append(result.Warnings, err.Error())
	}

	tests, warnings := impactSymbolTests(root, target, result.Packages, targetPackages)
	result.RelatedTests = tests.Tests
	result.TestCommands = tests.Commands
	result.TestPlan = tests.TestPlan
	result.Warnings = append(result.Warnings, warnings...)

	return result, nil
}

func impactSymbolCallers(root string, target string, options ImpactOptions) ([]Caller, []string, error) {
	normalizedTarget, err := normalizeCallTarget(root, target)
	if err != nil {
		return nil, nil, err
	}

	functions, _, warnings, err := collectCallFunctionInfos(root, CallOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return nil, warnings, err
	}

	callers, err := collectTransitiveCallersFromFunctions(functions, normalizedTarget)
	return callers, warnings, err
}

func impactSymbolTests(root string, target string, packages []string, targetPackages []string) (TestsResult, []string) {
	symbolTests, warnings := impactTests(root, target)
	mergedTests := symbolTests.Tests

	packageTests, packageWarnings := impactTestsForPackages(root, packages)
	warnings = append(warnings, packageWarnings...)
	mergedTests = mergeRelatedTests(mergedTests, packageTests)

	sortRelatedTests(mergedTests)
	if len(targetPackages) == 0 {
		targetPackages = sortedTestPackages(symbolTests.Tests)
	}
	fallbackPackages := uniqueSorted(append(append([]string{}, targetPackages...), packages...))
	plan := PlanTests(mergedTests, TestPlanOptions{
		Target:           firstNonEmptyString(symbolTests.Target, target),
		Kind:             TestTargetKindSymbol,
		TargetPackages:   targetPackages,
		CallerPackages:   packageDifference(packages, targetPackages),
		FallbackPackages: fallbackPackages,
	})

	return TestsResult{
		Target:   symbolTests.Target,
		Kind:     TestTargetKindSymbol,
		Tests:    mergedTests,
		Commands: TestPlanCommands(plan),
		TestPlan: plan,
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

	return collectRelatedTests(root, testFiles, packageSet, referenceTarget{}), nil
}

func impactTests(root string, target string) (TestsResult, []string) {
	tests, err := FindTests(root, target)
	if err != nil {
		return TestsResult{}, []string{err.Error()}
	}

	return tests, nil
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
