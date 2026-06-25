package sherpa

import (
	"go/token"
	"path/filepath"
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
	Warnings     []string            `json:"warnings"`
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
	normalizedTarget, err := normalizeReferenceTarget(root, target)
	if err != nil {
		return ImpactResult{}, err
	}

	refs, err := FindReferences(root, target)
	if err != nil {
		return ImpactResult{}, err
	}

	result := ImpactResult{
		Target:     normalizedTarget.String(),
		Kind:       ImpactKindSymbol,
		References: refs,
	}

	if normalizedTarget.Package == "" {
		callers, err := impactSymbolCallers(root, target)
		if err == nil {
			result.Callers = callers
		} else if !isImpactNonFunctionTargetError(err) {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}

	result.Packages = impactedPackages(root, refs, result.Callers)

	tests, warnings := impactTests(root, target)
	result.RelatedTests = tests.Tests
	result.TestCommands = tests.Commands
	result.Warnings = append(result.Warnings, warnings...)

	return result, nil
}

func impactSymbolCallers(root string, target string) ([]Caller, error) {
	normalizedTarget, err := normalizeCallTarget(target)
	if err != nil {
		return nil, err
	}

	functions, err := collectFunctionInfos(root)
	if err != nil {
		return nil, err
	}

	return collectTransitiveCallersFromFunctions(functions, normalizedTarget)
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
