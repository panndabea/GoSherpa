package explain

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	impactengine "github.com/supertabaluga/gosherpa/internal/impact"
	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type Report struct {
	Target                  string               `json:"target"`
	Symbol                  sherpa.Symbol        `json:"symbol"`
	References              []sherpa.Reference   `json:"references"`
	Callers                 []sherpa.Caller      `json:"callers"`
	Callees                 []sherpa.Callee      `json:"callees"`
	AffectedPackages        []string             `json:"affectedPackages"`
	AffectedInterfaces      []string             `json:"affectedInterfaces"`
	AffectedImplementations []string             `json:"affectedImplementations"`
	RelatedTests            []sherpa.RelatedTest `json:"relatedTests"`
	TestCommands            []string             `json:"testCommands"`
	Warnings                []string             `json:"warnings"`
}

type symbolTarget struct {
	Package  string
	Receiver string
	Name     string
}

func Analyze(root string, target string) (Report, error) {
	impactResult, err := sherpa.FindImpact(root, target)
	if err != nil {
		return Report{}, err
	}
	if impactResult.Kind != sherpa.ImpactKindSymbol {
		return Report{}, fmt.Errorf("explain target must be a symbol: %s", target)
	}

	symbols, err := sherpa.ParseRepository(root)
	if err != nil {
		return Report{}, err
	}

	symbol, err := findSymbol(root, symbols, impactResult.Target)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		Target:           impactResult.Target,
		Symbol:           symbol,
		References:       impactResult.References,
		AffectedPackages: impactResult.Packages,
		RelatedTests:     impactResult.RelatedTests,
		TestCommands:     impactResult.TestCommands,
		Warnings:         impactResult.Warnings,
	}

	impactReport, err := impactengine.AnalyzeSymbol(root, target)
	if err != nil {
		report.Warnings = append(report.Warnings, err.Error())
	} else {
		report.AffectedInterfaces = impactReport.AffectedInterfaces
		report.AffectedImplementations = impactReport.AffectedImplementations
		report.Warnings = append(report.Warnings, impactReport.Warnings...)
	}

	callTarget := callTargetForSymbol(symbol)
	if callTarget != "" {
		callers, err := sherpa.FindCallers(root, callTarget)
		if err != nil {
			report.Warnings = append(report.Warnings, "callers: "+err.Error())
		} else {
			report.Callers = callers.Callers
		}

		callees, err := sherpa.FindCallees(root, callTarget)
		if err != nil {
			report.Warnings = append(report.Warnings, "callees: "+err.Error())
		} else {
			report.Callees = callees.Callees
		}
	}

	return normalizeReport(report), nil
}

func findSymbol(root string, symbols []sherpa.Symbol, target string) (sherpa.Symbol, error) {
	parsed := parseSymbolTarget(root, target)
	if parsed.Name == "" {
		return sherpa.Symbol{}, fmt.Errorf("symbol not found: %s", target)
	}

	var matches []sherpa.Symbol
	for _, symbol := range symbols {
		if !symbolMatchesTarget(root, symbol, parsed) {
			continue
		}

		matches = append(matches, symbol)
	}

	if len(matches) == 0 {
		return sherpa.Symbol{}, fmt.Errorf("symbol not found: %s", target)
	}
	if len(matches) > 1 {
		return sherpa.Symbol{}, fmt.Errorf("ambiguous symbol target: %s", target)
	}

	return matches[0], nil
}

func symbolMatchesTarget(root string, symbol sherpa.Symbol, target symbolTarget) bool {
	if target.Package != "" && symbolPackage(root, symbol) != target.Package {
		return false
	}

	if target.Receiver != "" {
		return symbol.Receiver == target.Receiver && symbol.Name == target.Name
	}

	return symbol.Name == target.Name
}

func parseSymbolTarget(root string, target string) symbolTarget {
	value := strings.TrimSpace(filepath.ToSlash(target))
	if value == "" {
		return symbolTarget{}
	}

	packagePath, symbol := splitPackageQualifiedTarget(root, value)
	parts := strings.Split(symbol, ".")
	if len(parts) >= 2 {
		return symbolTarget{
			Package:  packagePath,
			Receiver: parts[len(parts)-2],
			Name:     parts[len(parts)-1],
		}
	}

	return symbolTarget{
		Package: packagePath,
		Name:    symbol,
	}
}

func splitPackageQualifiedTarget(root string, target string) (string, string) {
	lastSlash := strings.LastIndex(target, "/")
	if lastSlash < 0 {
		return "", target
	}

	firstDotAfterSlash := strings.Index(target[lastSlash+1:], ".")
	if firstDotAfterSlash < 0 {
		return "", target
	}

	separator := lastSlash + 1 + firstDotAfterSlash
	packagePath := normalizePackagePath(root, target[:separator])
	symbol := target[separator+1:]

	return packagePath, symbol
}

func normalizePackagePath(root string, packagePath string) string {
	value := strings.TrimSpace(filepath.ToSlash(packagePath))

	modulePath, err := sherpa.ModulePath(root)
	if err == nil && modulePath != "" {
		if value == modulePath {
			return "."
		}
		if strings.HasPrefix(value, modulePath+"/") {
			value = "./" + strings.TrimPrefix(value, modulePath+"/")
		}
	}

	value = strings.TrimPrefix(value, "./")
	cleaned := path.Clean(value)
	if cleaned == "." {
		return "."
	}

	return "./" + cleaned
}

func symbolPackage(root string, symbol sherpa.Symbol) string {
	file := strings.TrimSpace(filepath.ToSlash(symbol.Position.File))
	if file == "" {
		return ""
	}

	if filepath.IsAbs(file) {
		relative, err := filepath.Rel(root, file)
		if err == nil {
			file = filepath.ToSlash(relative)
		}
	}

	dir := path.Dir(file)
	if dir == "." {
		return "."
	}

	return "./" + dir
}

func callTargetForSymbol(symbol sherpa.Symbol) string {
	switch symbol.Kind {
	case sherpa.SymbolKindFunction:
		return symbol.Name
	case sherpa.SymbolKindMethod:
		if symbol.Receiver == "" {
			return symbol.Name
		}

		return symbol.Receiver + "." + symbol.Name
	default:
		return ""
	}
}

func normalizeReport(report Report) Report {
	report.References = nonNil(report.References)
	report.Callers = nonNil(report.Callers)
	report.Callees = nonNil(report.Callees)
	report.AffectedPackages = nonNil(report.AffectedPackages)
	report.AffectedInterfaces = nonNil(report.AffectedInterfaces)
	report.AffectedImplementations = nonNil(report.AffectedImplementations)
	report.RelatedTests = nonNil(report.RelatedTests)
	report.TestCommands = nonNil(report.TestCommands)
	report.Warnings = uniqueSorted(report.Warnings)

	return report
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	sort.Strings(result)

	return result
}
