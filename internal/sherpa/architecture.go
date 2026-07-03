package sherpa

import (
	"fmt"
	"sort"
)

const (
	ArchitectureAnalysisModeAST = "ast"
	ArchitectureConfidence      = "medium"

	architectureMaxRankedSignals = 5
	architectureMaxLeafPackages  = 10
)

type ArchitectureOptions struct {
	IncludeTests bool
}

type ArchitectureReport struct {
	AnalysisMode    string                      `json:"analysisMode"`
	Confidence      string                      `json:"confidence"`
	Limitations     []string                    `json:"limitations"`
	PackageCount    int                         `json:"packageCount"`
	Cycles          []DependencyCycle           `json:"cycles"`
	MostCoupled     []PackageArchitectureSignal `json:"mostCoupled"`
	HighFanIn       []PackageArchitectureSignal `json:"highFanIn"`
	HighFanOut      []PackageArchitectureSignal `json:"highFanOut"`
	LargestPackages []PackageArchitectureSignal `json:"largestPackages"`
	LeafPackages    []PackageArchitectureSignal `json:"leafPackages"`
}

type DependencyCycle struct {
	Packages []string `json:"packages"`
	Size     int      `json:"size"`
}

type PackageArchitectureSignal struct {
	Package         string `json:"package"`
	Reason          string `json:"reason"`
	Score           int    `json:"score"`
	ImportedBy      int    `json:"importedBy"`
	LocalImports    int    `json:"localImports"`
	ExternalImports int    `json:"externalImports"`
	Symbols         int    `json:"symbols"`
	GoFiles         int    `json:"goFiles"`
	TestFiles       int    `json:"testFiles"`
	HasTests        bool   `json:"hasTests"`
}

func AnalyzeArchitecture(root string, options ArchitectureOptions) (ArchitectureReport, error) {
	packages, err := FindPackageSummaries(root, PackageInventoryOptions{
		IncludeTests: options.IncludeTests,
	})
	if err != nil {
		return ArchitectureReport{}, err
	}

	dependencies, err := FindRepositoryDependenciesWithOptions(root, DependencyOptions{
		IncludeTests: options.IncludeTests,
	})
	if err != nil {
		return ArchitectureReport{}, err
	}

	report := ArchitectureReport{
		AnalysisMode:    ArchitectureAnalysisModeAST,
		Confidence:      ArchitectureConfidence,
		Limitations:     architectureLimitations(options.IncludeTests),
		PackageCount:    len(packages),
		Cycles:          findDependencyCycles(architectureAdjacency(packages, dependencies)),
		MostCoupled:     architectureMostCoupled(packages),
		HighFanIn:       architectureHighFanIn(packages),
		HighFanOut:      architectureHighFanOut(packages),
		LargestPackages: architectureLargestPackages(packages),
		LeafPackages:    architectureLeafPackages(packages),
	}

	return normalizeArchitectureReport(report), nil
}

func architectureLimitations(includeTests bool) []string {
	limitations := []string{
		"Architecture signals are structural and do not judge code quality.",
		"Dynamic runtime relationships, reflection, generated behavior, and framework wiring are not inferred.",
	}
	if includeTests {
		return append(limitations, "Test-file imports and test symbols are included because --tests was set.")
	}

	return append(limitations, "Test-file imports and test-only symbols are excluded unless --tests is set.")
}

func normalizeArchitectureReport(report ArchitectureReport) ArchitectureReport {
	if report.AnalysisMode == "" {
		report.AnalysisMode = ArchitectureAnalysisModeAST
	}
	if report.Confidence == "" {
		report.Confidence = ArchitectureConfidence
	}
	report.Limitations = nonNilStrings(report.Limitations)
	report.Cycles = nonNilCycles(report.Cycles)
	report.MostCoupled = nonNilArchitectureSignals(report.MostCoupled)
	report.HighFanIn = nonNilArchitectureSignals(report.HighFanIn)
	report.HighFanOut = nonNilArchitectureSignals(report.HighFanOut)
	report.LargestPackages = nonNilArchitectureSignals(report.LargestPackages)
	report.LeafPackages = nonNilArchitectureSignals(report.LeafPackages)

	return report
}

func architectureMostCoupled(packages []PackageSummary) []PackageArchitectureSignal {
	var signals []PackageArchitectureSignal
	for _, pkg := range packages {
		score := pkg.ImportedBy + pkg.LocalImports
		if score == 0 {
			continue
		}

		signals = append(signals, architectureSignal(pkg, score, fmt.Sprintf("fan-in %d + local fan-out %d", pkg.ImportedBy, pkg.LocalImports)))
	}

	sortArchitectureSignals(signals)
	return limitArchitectureSignals(signals, architectureMaxRankedSignals)
}

func architectureHighFanIn(packages []PackageSummary) []PackageArchitectureSignal {
	var signals []PackageArchitectureSignal
	for _, pkg := range packages {
		if pkg.ImportedBy == 0 {
			continue
		}

		signals = append(signals, architectureSignal(pkg, pkg.ImportedBy, fmt.Sprintf("imported by %d local package(s)", pkg.ImportedBy)))
	}

	sortArchitectureSignals(signals)
	return limitArchitectureSignals(signals, architectureMaxRankedSignals)
}

func architectureHighFanOut(packages []PackageSummary) []PackageArchitectureSignal {
	var signals []PackageArchitectureSignal
	for _, pkg := range packages {
		if pkg.LocalImports == 0 {
			continue
		}

		signals = append(signals, architectureSignal(pkg, pkg.LocalImports, fmt.Sprintf("imports %d local package(s)", pkg.LocalImports)))
	}

	sortArchitectureSignals(signals)
	return limitArchitectureSignals(signals, architectureMaxRankedSignals)
}

func architectureLargestPackages(packages []PackageSummary) []PackageArchitectureSignal {
	var signals []PackageArchitectureSignal
	for _, pkg := range packages {
		if pkg.Symbols == 0 {
			continue
		}

		signals = append(signals, architectureSignal(pkg, pkg.Symbols, fmt.Sprintf("contains %d symbol(s)", pkg.Symbols)))
	}

	sortArchitectureSignals(signals)
	return limitArchitectureSignals(signals, architectureMaxRankedSignals)
}

func architectureLeafPackages(packages []PackageSummary) []PackageArchitectureSignal {
	var signals []PackageArchitectureSignal
	for _, pkg := range packages {
		if pkg.LocalImports != 0 {
			continue
		}

		signals = append(signals, architectureSignal(pkg, pkg.ImportedBy, "imports no local packages"))
	}

	sortArchitectureSignals(signals)
	return limitArchitectureSignals(signals, architectureMaxLeafPackages)
}

func architectureSignal(pkg PackageSummary, score int, reason string) PackageArchitectureSignal {
	return PackageArchitectureSignal{
		Package:         pkg.Package,
		Reason:          reason,
		Score:           score,
		ImportedBy:      pkg.ImportedBy,
		LocalImports:    pkg.LocalImports,
		ExternalImports: pkg.ExternalImports,
		Symbols:         pkg.Symbols,
		GoFiles:         pkg.GoFiles,
		TestFiles:       pkg.TestFiles,
		HasTests:        pkg.HasTests,
	}
}

func sortArchitectureSignals(signals []PackageArchitectureSignal) {
	sort.SliceStable(signals, func(i int, j int) bool {
		if signals[i].Score != signals[j].Score {
			return signals[i].Score > signals[j].Score
		}
		if signals[i].ImportedBy != signals[j].ImportedBy {
			return signals[i].ImportedBy > signals[j].ImportedBy
		}
		if signals[i].LocalImports != signals[j].LocalImports {
			return signals[i].LocalImports > signals[j].LocalImports
		}
		if signals[i].ExternalImports != signals[j].ExternalImports {
			return signals[i].ExternalImports > signals[j].ExternalImports
		}
		if signals[i].Symbols != signals[j].Symbols {
			return signals[i].Symbols > signals[j].Symbols
		}

		return signals[i].Package < signals[j].Package
	})
}

func limitArchitectureSignals(signals []PackageArchitectureSignal, limit int) []PackageArchitectureSignal {
	if len(signals) > limit {
		signals = signals[:limit]
	}

	return signals
}

func architectureAdjacency(packages []PackageSummary, dependencies RepositoryDependencies) map[string][]string {
	adjacency := map[string][]string{}
	for _, pkg := range packages {
		adjacency[pkg.Package] = nil
	}
	for _, pkg := range dependencies.Packages {
		if _, ok := adjacency[pkg.Package]; !ok {
			adjacency[pkg.Package] = nil
		}
	}

	for _, pkg := range dependencies.Packages {
		for _, localImport := range pkg.LocalImports {
			if _, ok := adjacency[localImport]; !ok || localImport == pkg.Package {
				continue
			}

			adjacency[pkg.Package] = append(adjacency[pkg.Package], localImport)
		}
	}
	for pkg, imports := range adjacency {
		adjacency[pkg] = uniqueSorted(imports)
	}

	return adjacency
}

func findDependencyCycles(adjacency map[string][]string) []DependencyCycle {
	var index int
	var stack []string
	indices := map[string]int{}
	lowlinks := map[string]int{}
	onStack := map[string]bool{}
	var cycles []DependencyCycle

	var visit func(string)
	visit = func(node string) {
		indices[node] = index
		lowlinks[node] = index
		index++
		stack = append(stack, node)
		onStack[node] = true

		for _, next := range adjacency[node] {
			if _, ok := indices[next]; !ok {
				visit(next)
				if lowlinks[next] < lowlinks[node] {
					lowlinks[node] = lowlinks[next]
				}
				continue
			}
			if onStack[next] && indices[next] < lowlinks[node] {
				lowlinks[node] = indices[next]
			}
		}

		if lowlinks[node] != indices[node] {
			return
		}

		var component []string
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == node {
				break
			}
		}
		if len(component) > 1 {
			sort.Strings(component)
			cycles = append(cycles, DependencyCycle{
				Packages: component,
				Size:     len(component),
			})
		}
	}

	nodes := make([]string, 0, len(adjacency))
	for node := range adjacency {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if _, ok := indices[node]; ok {
			continue
		}

		visit(node)
	}

	sort.SliceStable(cycles, func(i int, j int) bool {
		if cycles[i].Size != cycles[j].Size {
			return cycles[i].Size > cycles[j].Size
		}
		if len(cycles[i].Packages) == 0 || len(cycles[j].Packages) == 0 {
			return len(cycles[i].Packages) > len(cycles[j].Packages)
		}

		return cycles[i].Packages[0] < cycles[j].Packages[0]
	})

	return cycles
}

func nonNilCycles(values []DependencyCycle) []DependencyCycle {
	if values == nil {
		return []DependencyCycle{}
	}
	for i, cycle := range values {
		cycle.Packages = nonNilStrings(cycle.Packages)
		values[i] = cycle
	}

	return values
}

func nonNilArchitectureSignals(values []PackageArchitectureSignal) []PackageArchitectureSignal {
	if values == nil {
		return []PackageArchitectureSignal{}
	}

	return values
}
