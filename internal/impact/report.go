package impact

import (
	"sort"
	"strconv"
	"strings"

	gitdiff "github.com/supertabaluga/gosherpa/internal/git"
	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type Analyzer struct {
	Root string
}

type RelatedTest = sherpa.RelatedTest

type ImpactReport struct {
	ChangedFiles            []string      `json:"changedFiles"`
	ChangedPackages         []string      `json:"changedPackages"`
	AffectedPackages        []string      `json:"affectedPackages"`
	AffectedSymbols         []string      `json:"affectedSymbols"`
	AffectedInterfaces      []string      `json:"affectedInterfaces"`
	AffectedImplementations []string      `json:"affectedImplementations"`
	AffectedTests           []RelatedTest `json:"affectedTests"`
	TestCommands            []string      `json:"testCommands"`
	Warnings                []string      `json:"warnings"`
}

func NewAnalyzer(root string) Analyzer {
	return Analyzer{Root: root}
}

func AnalyzeDiff(root string, base string, head string) (ImpactReport, error) {
	return NewAnalyzer(root).AnalyzeDiff(base, head)
}

func (a Analyzer) AnalyzeDiff(base string, head string) (ImpactReport, error) {
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

	report.AffectedPackages, report.Warnings = affectedPackagesForChangedPackages(a.Root, report.ChangedPackages)
	report.AffectedTests, report.TestCommands, report.Warnings = affectedTestsForPackages(a.Root, report.AffectedPackages, report.Warnings)

	return normalizeReport(report), nil
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

func affectedTestsForPackages(root string, packages []string, warnings []string) ([]sherpa.RelatedTest, []string, []string) {
	seen := make(map[string]sherpa.RelatedTest)
	var commands []string

	for _, pkg := range packages {
		tests, err := sherpa.FindTests(root, pkg)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}

		for _, test := range tests.Tests {
			seen[relatedTestKey(test)] = test
		}
		commands = append(commands, tests.Commands...)
	}

	result := make([]sherpa.RelatedTest, 0, len(seen))
	for _, test := range seen {
		result = append(result, test)
	}

	sortRelatedTests(result)

	return result, uniqueSortedStrings(commands), uniqueSortedStrings(warnings)
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
	report.AffectedInterfaces = nonNilStrings(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilStrings(report.AffectedImplementations)
	report.TestCommands = nonNilStrings(report.TestCommands)
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

func intKey(value int) string {
	return strconv.Itoa(value)
}
