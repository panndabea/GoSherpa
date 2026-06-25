package sherpa

import (
	"path/filepath"
	"strings"
)

type ImpactKind string

const (
	ImpactKindSymbol  ImpactKind = "symbol"
	ImpactKindPackage ImpactKind = "package"
)

type ImpactResult struct {
	Target       string
	Kind         ImpactKind
	References   []Reference
	Callers      []Caller
	Dependencies PackageDependencies
	Packages     []string
	RelatedTests []RelatedTest
	TestCommands []string
	Warnings     []string
}

func FindImpact(root string, target string) (ImpactResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return ImpactResult{}, err
	}

	if isImpactPackageTarget(target) {
		return findPackageImpact(rootPath, target)
	}

	return findSymbolImpact(rootPath, target)
}

func isImpactPackageTarget(target string) bool {
	value := strings.TrimSpace(target)
	if value == "." || strings.HasPrefix(value, "./") {
		return true
	}

	return strings.Contains(value, "/") || strings.Contains(value, "\\")
}

func findPackageImpact(root string, target string) (ImpactResult, error) {
	deps, err := FindPackageDependencies(root, target)
	if err != nil {
		return ImpactResult{}, err
	}

	packages := uniqueSorted(append([]string{deps.Package}, deps.UsedBy...))
	tests, warnings := impactTests(root, target)

	return ImpactResult{
		Target:       deps.Package,
		Kind:         ImpactKindPackage,
		Dependencies: deps,
		Packages:     packages,
		RelatedTests: tests.Tests,
		TestCommands: tests.Commands,
		Warnings:     warnings,
	}, nil
}

func findSymbolImpact(root string, target string) (ImpactResult, error) {
	refs, err := FindReferences(root, target)
	if err != nil {
		return ImpactResult{}, err
	}

	result := ImpactResult{
		Target:     strings.TrimSpace(target),
		Kind:       ImpactKindSymbol,
		References: refs,
	}

	callers, err := FindCallers(root, target)
	if err == nil {
		result.Callers = callers.Callers
	} else if !isImpactNonFunctionTargetError(err) {
		result.Warnings = append(result.Warnings, err.Error())
	}

	result.Packages = impactedPackages(root, refs, result.Callers)

	tests, warnings := impactTests(root, target)
	result.RelatedTests = tests.Tests
	result.TestCommands = tests.Commands
	result.Warnings = append(result.Warnings, warnings...)

	return result, nil
}

func impactTests(root string, target string) (TestsResult, []string) {
	tests, err := FindTests(root, target)
	if err != nil {
		return TestsResult{}, []string{err.Error()}
	}

	return tests, nil
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
