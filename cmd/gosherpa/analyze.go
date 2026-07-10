package main

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"sort"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

const (
	analyzeMaxImportantSymbols = 12
	analyzeMaxEntryPoints      = 12
	analyzeMaxHotspots         = 5
	analyzeMaxTestCommands     = 6

	analyzeAnalysisModeSnapshotAST         = "snapshot+ast"
	analyzeAnalysisModeSnapshotTypechecked = "snapshot+typechecked+ast"
)

type analyzeOptions struct {
	IncludeTests bool
	BuildTags    []string
	UseSnapshot  bool
}

type analyzeReport struct {
	Target           string                   `json:"target"`
	AnalysisMode     string                   `json:"analysisMode"`
	Confidence       string                   `json:"confidence"`
	Limitations      []string                 `json:"limitations"`
	Repository       analyzeRepositorySummary `json:"repository"`
	BuildTags        []string                 `json:"buildTags"`
	SymbolSummary    analyzeSymbolSummary     `json:"symbolSummary"`
	Packages         []sherpa.PackageSummary  `json:"packages"`
	ImportantSymbols []analyzeSymbolProfile   `json:"importantSymbols"`
	EntryPoints      []sherpa.EntryPoint      `json:"entrypoints"`
	Risk             sherpa.RiskReport        `json:"risk"`
	Hotspots         []analyzeHotspot         `json:"hotspots"`
	Testing          analyzeTestingOverview   `json:"testing"`
	Readiness        analyzeReadinessSummary  `json:"readiness"`
	Suggestions      []string                 `json:"suggestions"`
	Warnings         []string                 `json:"-"`
}

type analyzeRepositorySummary struct {
	ModulePath       string `json:"modulePath"`
	GoFiles          int    `json:"goFiles"`
	TestFiles        int    `json:"testFiles"`
	GeneratedFiles   int    `json:"generatedFiles"`
	PackageCount     int    `json:"packageCount"`
	TestPackageCount int    `json:"testPackageCount"`
	SymbolCount      int    `json:"symbolCount"`
}

type analyzeSymbolSummary struct {
	Total      int `json:"total"`
	Structs    int `json:"structs"`
	Interfaces int `json:"interfaces"`
	Functions  int `json:"functions"`
	Methods    int `json:"methods"`
	Exported   int `json:"exported"`
	Tests      int `json:"tests"`
}

type analyzeSymbolProfile struct {
	Name          string              `json:"name"`
	Kind          sherpa.SymbolKind   `json:"kind"`
	Package       string              `json:"package,omitempty"`
	QualifiedName string              `json:"qualifiedName,omitempty"`
	Signature     string              `json:"signature,omitempty"`
	Position      sherpa.Position     `json:"position"`
	Range         *sherpa.SourceRange `json:"range,omitempty"`
}

type analyzeHotspot struct {
	Package         string `json:"package"`
	Reason          string `json:"reason"`
	ImportedBy      int    `json:"importedBy"`
	Imports         int    `json:"imports"`
	LocalImports    int    `json:"localImports"`
	ExternalImports int    `json:"externalImports"`
	Symbols         int    `json:"symbols"`
	GoFiles         int    `json:"goFiles"`
	TestFiles       int    `json:"testFiles"`
}

type analyzeTestingOverview struct {
	TestFiles         int      `json:"testFiles"`
	TestPackageCount  int      `json:"testPackageCount"`
	TestPackages      []string `json:"testPackages"`
	SuggestedCommands []string `json:"suggestedCommands"`
}

type analyzeReadinessSummary struct {
	AnalysisMode     string   `json:"analysisMode"`
	Confidence       string   `json:"confidence"`
	PackageLoad      string   `json:"packageLoad"`
	PackageLoadCount int      `json:"packageLoadCount"`
	SnapshotStatus   string   `json:"snapshotStatus"`
	Suggestions      []string `json:"suggestions"`
}

func analyzeRepository(root string, options analyzeOptions) (analyzeReport, error) {
	doctor := analyzeDoctor(root, options.BuildTags)

	packages, symbols, inventoryMode, inventoryWarnings, err := analyzeInventory(root, options)
	if err != nil {
		return analyzeReport{}, err
	}

	selectedSymbols := analyzeSelectedSymbols(symbols, options.IncludeTests)
	testPackages := analyzeTestPackages(packages)
	analysisMode := analyzeAnalysisMode(doctor)
	if inventoryMode == analysisModeSnapshot {
		analysisMode = analyzeSnapshotAnalysisMode(analysisMode)
	}
	warnings := append(nonNilSlice(doctor.Warnings), inventoryWarnings...)

	risk, err := sherpa.AnalyzeRisk(root, sherpa.RiskOptions{
		IncludeTests: options.IncludeTests,
	})
	if err != nil {
		return analyzeReport{}, err
	}

	report := analyzeReport{
		Target:       ".",
		AnalysisMode: analysisMode,
		Confidence:   jsonConfidence(warnings, analysisMode),
		Repository: analyzeRepositorySummary{
			ModulePath:       doctor.Repository.ModulePath,
			GoFiles:          doctor.Repository.GoFiles,
			TestFiles:        doctor.Repository.TestFiles,
			GeneratedFiles:   doctor.Repository.GeneratedFiles,
			PackageCount:     len(packages),
			TestPackageCount: len(testPackages),
			SymbolCount:      len(selectedSymbols),
		},
		BuildTags:        semantics.NormalizeBuildTags(options.BuildTags),
		SymbolSummary:    analyzeSymbolSummaryFromSymbols(selectedSymbols, symbols),
		Packages:         packages,
		ImportantSymbols: analyzeImportantSymbols(selectedSymbols),
		EntryPoints:      analyzeEntryPoints(selectedSymbols, options.IncludeTests),
		Risk:             risk,
		Hotspots:         analyzeHotspots(packages),
		Testing: analyzeTestingOverview{
			TestFiles:         doctor.Repository.TestFiles,
			TestPackageCount:  len(testPackages),
			TestPackages:      testPackages,
			SuggestedCommands: analyzeSuggestedTestCommands(testPackages),
		},
		Readiness: analyzeReadinessSummary{
			AnalysisMode:     doctor.AnalysisMode,
			Confidence:       doctor.Confidence,
			PackageLoad:      doctor.PackageLoad.Status,
			PackageLoadCount: doctor.PackageLoad.PackageCount,
			SnapshotStatus:   doctor.Snapshot.Status,
			Suggestions:      doctor.Suggestions,
		},
		Warnings: warnings,
	}
	report.Limitations = analyzeLimitations(options.IncludeTests, inventoryMode == analysisModeSnapshot)
	report.Suggestions = analyzeSuggestions(report)

	return normalizeAnalyzeReport(report), nil
}

func analyzeInventory(root string, options analyzeOptions) ([]sherpa.PackageSummary, []sherpa.Symbol, string, []string, error) {
	var warnings []string
	if options.UseSnapshot {
		stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{
			BuildTags: options.BuildTags,
		})
		if inspect.Status == snapshotstore.StatusValid {
			packages, err := analyzePackagesFromSnapshotOrLive(root, stored, options.IncludeTests)
			if err != nil {
				return nil, nil, "", warnings, err
			}

			return packages, cloneSlice(stored.Symbols), analysisModeSnapshot, warnings, nil
		}

		warnings = append(warnings, snapshotFallbackWarning(inspect))
	}

	packages, err := sherpa.FindPackageSummaries(root, sherpa.PackageInventoryOptions{
		IncludeTests: options.IncludeTests,
	})
	if err != nil {
		return nil, nil, "", warnings, err
	}

	symbols, err := sherpa.ParseRepository(root)
	if err != nil {
		return nil, nil, "", warnings, err
	}

	return packages, symbols, "", warnings, nil
}

func analyzePackagesFromSnapshotOrLive(root string, stored snapshotstore.Snapshot, includeTests bool) ([]sherpa.PackageSummary, error) {
	if includeTests {
		return cloneSlice(stored.Packages), nil
	}

	return sherpa.FindPackageSummaries(root, sherpa.PackageInventoryOptions{})
}

func normalizeAnalyzeReport(report analyzeReport) analyzeReport {
	if strings.TrimSpace(report.Target) == "" {
		report.Target = "."
	}
	if strings.TrimSpace(report.AnalysisMode) == "" {
		report.AnalysisMode = analysisModeAST
	}
	report.Confidence = strings.TrimSpace(report.Confidence)
	if report.Confidence == "" {
		report.Confidence = jsonConfidence(report.Warnings, report.AnalysisMode)
	}
	report.BuildTags = nonNilSlice(semantics.NormalizeBuildTags(report.BuildTags))
	report.Limitations = nonNilSlice(report.Limitations)
	report.Packages = nonNilSlice(report.Packages)
	report.ImportantSymbols = nonNilSlice(report.ImportantSymbols)
	report.EntryPoints = nonNilSlice(report.EntryPoints)
	report.Risk = normalizeAnalyzeRiskReport(report.Risk)
	report.Hotspots = nonNilSlice(report.Hotspots)
	report.Testing.TestPackages = nonNilSlice(report.Testing.TestPackages)
	report.Testing.SuggestedCommands = nonNilSlice(report.Testing.SuggestedCommands)
	report.Readiness.Suggestions = nonNilSlice(report.Readiness.Suggestions)
	report.Suggestions = nonNilSlice(report.Suggestions)
	report.Warnings = nonNilSlice(uniqueStringsInOrder(report.Warnings))

	return report
}

func normalizeAnalyzeRiskReport(report sherpa.RiskReport) sherpa.RiskReport {
	report.Limitations = nonNilSlice(report.Limitations)
	report.Factors = nonNilSlice(report.Factors)
	report.Packages = nonNilSlice(report.Packages)
	for i, pkg := range report.Packages {
		pkg.Reasons = nonNilSlice(pkg.Reasons)
		report.Packages[i] = pkg
	}
	report.Cycles = nonNilSlice(report.Cycles)
	for i, cycle := range report.Cycles {
		cycle.Packages = nonNilSlice(cycle.Packages)
		report.Cycles[i] = cycle
	}
	if strings.TrimSpace(report.AnalysisMode) == "" {
		report.AnalysisMode = sherpa.RiskAnalysisModeAST
	}
	if strings.TrimSpace(report.Confidence) == "" {
		report.Confidence = sherpa.RiskConfidence
	}
	if strings.TrimSpace(report.Level) == "" {
		report.Level = sherpa.RiskLevelLow
	}

	return report
}

func analyzeAnalysisMode(doctor doctorReport) string {
	if doctor.AnalysisMode == doctorAnalysisModeTypechecked {
		return agentcontext.AnalysisModeTypecheckedAST
	}

	return analysisModeAST
}

func analyzeSelectedSymbols(symbols []sherpa.Symbol, includeTests bool) []sherpa.Symbol {
	var selected []sherpa.Symbol
	for _, symbol := range symbols {
		if !includeTests && analyzeSymbolIsTest(symbol) {
			continue
		}

		selected = append(selected, symbol)
	}
	sortAnalyzeSymbols(selected)

	return selected
}

func analyzeSymbolSummaryFromSymbols(selected []sherpa.Symbol, all []sherpa.Symbol) analyzeSymbolSummary {
	summary := analyzeSymbolSummary{Total: len(selected)}
	for _, symbol := range selected {
		switch symbol.Kind {
		case sherpa.SymbolKindStruct:
			summary.Structs++
		case sherpa.SymbolKindInterface:
			summary.Interfaces++
		case sherpa.SymbolKindFunction:
			summary.Functions++
		case sherpa.SymbolKindMethod:
			summary.Methods++
		}
		if analyzeSymbolIsExported(symbol) {
			summary.Exported++
		}
	}
	for _, symbol := range all {
		if analyzeSymbolIsTest(symbol) {
			summary.Tests++
		}
	}

	return summary
}

func analyzeImportantSymbols(symbols []sherpa.Symbol) []analyzeSymbolProfile {
	var important []sherpa.Symbol
	for _, symbol := range symbols {
		if !analyzeSymbolIsExported(symbol) && !analyzeSymbolIsMain(symbol) {
			continue
		}

		important = append(important, symbol)
	}

	sortAnalyzeSymbolsByImportance(important)
	if len(important) > analyzeMaxImportantSymbols {
		important = important[:analyzeMaxImportantSymbols]
	}

	profiles := make([]analyzeSymbolProfile, 0, len(important))
	for _, symbol := range important {
		profiles = append(profiles, analyzeSymbolProfileFromSymbol(symbol))
	}

	return profiles
}

func analyzeEntryPoints(symbols []sherpa.Symbol, includeTests bool) []sherpa.EntryPoint {
	var entryPoints []sherpa.EntryPoint
	for _, symbol := range symbols {
		kind, ok := analyzeEntryPointKind(symbol, includeTests)
		if !ok {
			continue
		}

		entryPoints = append(entryPoints, sherpa.EntryPoint{
			Name:     symbol.DisplayName(),
			Package:  symbol.Package,
			Kind:     kind,
			Position: symbol.Position,
			Range:    symbol.Range,
		})
	}

	sort.Slice(entryPoints, func(i int, j int) bool {
		leftPriority := analyzeEntryPointKindPriority(entryPoints[i].Kind)
		rightPriority := analyzeEntryPointKindPriority(entryPoints[j].Kind)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if entryPoints[i].Package != entryPoints[j].Package {
			return entryPoints[i].Package < entryPoints[j].Package
		}
		if entryPoints[i].Name != entryPoints[j].Name {
			return entryPoints[i].Name < entryPoints[j].Name
		}
		return entryPoints[i].Position.File < entryPoints[j].Position.File
	})
	if len(entryPoints) > analyzeMaxEntryPoints {
		entryPoints = entryPoints[:analyzeMaxEntryPoints]
	}

	return entryPoints
}

func analyzeEntryPointKind(symbol sherpa.Symbol, includeTests bool) (sherpa.EntryPointKind, bool) {
	if analyzeSymbolIsMain(symbol) {
		return sherpa.EntryPointKindMain, true
	}
	if includeTests && analyzeSymbolIsGoTestEntryPoint(symbol) {
		return sherpa.EntryPointKindTest, true
	}
	if symbol.Kind == sherpa.SymbolKindFunction && analyzeSymbolIsExported(symbol) {
		return sherpa.EntryPointKindExported, true
	}

	return "", false
}

func analyzeHotspots(packages []sherpa.PackageSummary) []analyzeHotspot {
	hotspots := make([]analyzeHotspot, 0, len(packages))
	for _, pkg := range packages {
		score := analyzeHotspotScore(pkg)
		if score == 0 {
			continue
		}

		hotspots = append(hotspots, analyzeHotspotFromPackage(pkg))
	}

	sort.Slice(hotspots, func(i int, j int) bool {
		leftScore := analyzeHotspotScoreFromHotspot(hotspots[i])
		rightScore := analyzeHotspotScoreFromHotspot(hotspots[j])
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		if hotspots[i].ImportedBy != hotspots[j].ImportedBy {
			return hotspots[i].ImportedBy > hotspots[j].ImportedBy
		}
		if hotspots[i].LocalImports != hotspots[j].LocalImports {
			return hotspots[i].LocalImports > hotspots[j].LocalImports
		}
		if hotspots[i].Symbols != hotspots[j].Symbols {
			return hotspots[i].Symbols > hotspots[j].Symbols
		}
		return hotspots[i].Package < hotspots[j].Package
	})
	if len(hotspots) > analyzeMaxHotspots {
		hotspots = hotspots[:analyzeMaxHotspots]
	}

	return hotspots
}

func analyzeHotspotFromPackage(pkg sherpa.PackageSummary) analyzeHotspot {
	return analyzeHotspot{
		Package:         pkg.Package,
		Reason:          analyzeHotspotReason(pkg),
		ImportedBy:      pkg.ImportedBy,
		Imports:         pkg.Imports,
		LocalImports:    pkg.LocalImports,
		ExternalImports: pkg.ExternalImports,
		Symbols:         pkg.Symbols,
		GoFiles:         pkg.GoFiles,
		TestFiles:       pkg.TestFiles,
	}
}

func analyzeTestingCommandsFromPackages(packages []string) []string {
	var commands []string
	if len(packages) > 0 {
		commands = append(commands, "go test ./...")
	}
	for _, pkg := range packages {
		commands = append(commands, "go test "+pkg)
		if len(commands) >= analyzeMaxTestCommands {
			break
		}
	}

	return uniqueStringsInOrder(commands)
}

func analyzeSuggestedTestCommands(testPackages []string) []string {
	return analyzeTestingCommandsFromPackages(testPackages)
}

func analyzeTestPackages(packages []sherpa.PackageSummary) []string {
	var result []string
	for _, pkg := range packages {
		if !pkg.HasTests {
			continue
		}

		result = append(result, pkg.Package)
	}
	sort.Strings(result)

	return result
}

func analyzeSuggestions(report analyzeReport) []string {
	var suggestions []string
	if report.Risk.Level == sherpa.RiskLevelMedium || report.Risk.Level == sherpa.RiskLevelHigh {
		suggestions = append(suggestions, "Inspect structural risk with gosherpa risk")
	}
	if len(report.Hotspots) > 0 {
		suggestions = append(suggestions, "Inspect package "+report.Hotspots[0].Package+" with gosherpa context package "+report.Hotspots[0].Package)
	}
	if len(report.ImportantSymbols) > 0 {
		suggestions = append(suggestions, "Inspect symbol "+report.ImportantSymbols[0].QualifiedName+" with gosherpa context symbol "+report.ImportantSymbols[0].QualifiedName)
	}
	if report.Testing.TestPackageCount > 0 {
		suggestions = append(suggestions, "Run gosherpa tests affected --base <ref> when reviewing a change.")
	}
	suggestions = append(suggestions,
		"Run gosherpa context diff --base <ref> for focused change context.",
		"Run gosherpa doctor when analysis confidence looks low.",
	)

	return uniqueStringsInOrder(suggestions)
}

func analyzeSnapshotAnalysisMode(base string) string {
	if base == agentcontext.AnalysisModeTypecheckedAST {
		return analyzeAnalysisModeSnapshotTypechecked
	}

	return analyzeAnalysisModeSnapshotAST
}

func analyzeLimitations(includeTests bool, usingSnapshot bool) []string {
	limitations := []string{
		"Analyze composes repository inventory, package summaries, symbol discovery, and doctor readiness checks.",
		"Hotspots are simple inventory signals, not runtime profiling or full dependency impact.",
		"Entry point overview is based on main functions, exported functions, and optional Go test entrypoints.",
		"Build tags are applied to semantic readiness checks; syntax inventory still follows discovered Go files.",
	}
	if usingSnapshot {
		limitations = append(limitations, "Analyze reused a valid snapshot for symbol inventory; risk and readiness still use live repository analysis.")
		if !includeTests {
			limitations = append(limitations, "Package summaries still use live analysis without --tests because snapshots store test-inclusive package inventory.")
		}
	} else {
		limitations = append(limitations, "Analyze reads repository inventory directly unless --use-snapshot is provided with a valid snapshot.")
	}
	if !includeTests {
		limitations = append(limitations, "Test symbols are counted but omitted from entry point and important symbol lists unless --tests is used.")
	}

	return limitations
}

func analyzeSymbolProfileFromSymbol(symbol sherpa.Symbol) analyzeSymbolProfile {
	return analyzeSymbolProfile{
		Name:          symbol.DisplayName(),
		Kind:          symbol.Kind,
		Package:       symbol.Package,
		QualifiedName: symbol.QualifiedName,
		Signature:     symbol.Signature,
		Position:      symbol.Position,
		Range:         symbol.Range,
	}
}

func analyzeHotspotScore(pkg sherpa.PackageSummary) int {
	return pkg.ImportedBy*100 + pkg.LocalImports*25 + pkg.Symbols
}

func analyzeHotspotScoreFromHotspot(hotspot analyzeHotspot) int {
	return hotspot.ImportedBy*100 + hotspot.LocalImports*25 + hotspot.Symbols
}

func analyzeHotspotReason(pkg sherpa.PackageSummary) string {
	switch {
	case pkg.ImportedBy > 0:
		return fmt.Sprintf("imported by %d local %s", pkg.ImportedBy, analyzePlural("package", pkg.ImportedBy))
	case pkg.LocalImports > 0:
		return fmt.Sprintf("imports %d local %s", pkg.LocalImports, analyzePlural("package", pkg.LocalImports))
	default:
		return fmt.Sprintf("declares %d supported %s", pkg.Symbols, analyzePlural("symbol", pkg.Symbols))
	}
}

func analyzePlural(word string, count int) string {
	if count == 1 {
		return word
	}

	return word + "s"
}

func analyzeSymbolIsExported(symbol sherpa.Symbol) bool {
	if symbol.Name == "" {
		return false
	}

	return ast.IsExported(symbol.Name)
}

func analyzeSymbolIsMain(symbol sherpa.Symbol) bool {
	return symbol.Kind == sherpa.SymbolKindFunction &&
		symbol.PackageName == "main" &&
		symbol.Name == "main"
}

func analyzeSymbolIsGoTestEntryPoint(symbol sherpa.Symbol) bool {
	if symbol.Kind != sherpa.SymbolKindFunction {
		return false
	}
	if !strings.HasSuffix(filepath.ToSlash(symbol.Position.File), "_test.go") {
		return false
	}

	return strings.HasPrefix(symbol.Name, "Test") ||
		strings.HasPrefix(symbol.Name, "Benchmark") ||
		strings.HasPrefix(symbol.Name, "Fuzz") ||
		strings.HasPrefix(symbol.Name, "Example")
}

func analyzeSymbolIsTest(symbol sherpa.Symbol) bool {
	return strings.HasSuffix(filepath.ToSlash(symbol.Position.File), "_test.go") ||
		analyzeSymbolIsGoTestEntryPoint(symbol)
}

func sortAnalyzeSymbols(symbols []sherpa.Symbol) {
	sort.Slice(symbols, func(i int, j int) bool {
		if symbols[i].Package != symbols[j].Package {
			return symbols[i].Package < symbols[j].Package
		}
		if symbols[i].Position.File != symbols[j].Position.File {
			return symbols[i].Position.File < symbols[j].Position.File
		}
		if symbols[i].Position.Line != symbols[j].Position.Line {
			return symbols[i].Position.Line < symbols[j].Position.Line
		}
		return symbols[i].DisplayName() < symbols[j].DisplayName()
	})
}

func sortAnalyzeSymbolsByImportance(symbols []sherpa.Symbol) {
	sort.Slice(symbols, func(i int, j int) bool {
		leftPriority := analyzeSymbolKindPriority(symbols[i].Kind)
		rightPriority := analyzeSymbolKindPriority(symbols[j].Kind)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if symbols[i].Package != symbols[j].Package {
			return symbols[i].Package < symbols[j].Package
		}
		if symbols[i].DisplayName() != symbols[j].DisplayName() {
			return symbols[i].DisplayName() < symbols[j].DisplayName()
		}
		return symbols[i].Position.File < symbols[j].Position.File
	})
}

func analyzeSymbolKindPriority(kind sherpa.SymbolKind) int {
	switch kind {
	case sherpa.SymbolKindInterface:
		return 0
	case sherpa.SymbolKindStruct:
		return 1
	case sherpa.SymbolKindAlias:
		return 2
	case sherpa.SymbolKindFunction:
		return 3
	case sherpa.SymbolKindMethod:
		return 4
	default:
		return 5
	}
}

func analyzeEntryPointKindPriority(kind sherpa.EntryPointKind) int {
	switch kind {
	case sherpa.EntryPointKindMain:
		return 0
	case sherpa.EntryPointKindTest:
		return 1
	case sherpa.EntryPointKindExported:
		return 2
	default:
		return 3
	}
}

func formatAnalyzeReport(report analyzeReport) string {
	report = normalizeAnalyzeReport(report)

	var builder strings.Builder
	builder.WriteString("ANALYZE\n\n")
	fmt.Fprintf(&builder, "Module: %s\n", valueOrNone(report.Repository.ModulePath))
	fmt.Fprintf(&builder, "Analysis: %s\n", report.AnalysisMode)
	fmt.Fprintf(&builder, "Confidence: %s\n", report.Confidence)
	builder.WriteString("\n")

	builder.WriteString("SUMMARY\n")
	fmt.Fprintf(&builder, "  Packages: %d\n", report.Repository.PackageCount)
	fmt.Fprintf(&builder, "  Go files: %d\n", report.Repository.GoFiles)
	fmt.Fprintf(&builder, "  Test files: %d\n", report.Repository.TestFiles)
	fmt.Fprintf(&builder, "  Symbols: %d\n", report.Repository.SymbolCount)
	fmt.Fprintf(&builder, "  Test packages: %d\n", report.Repository.TestPackageCount)
	builder.WriteString("\n")

	builder.WriteString("SYMBOL SUMMARY\n")
	fmt.Fprintf(&builder, "  Structs: %d\n", report.SymbolSummary.Structs)
	fmt.Fprintf(&builder, "  Interfaces: %d\n", report.SymbolSummary.Interfaces)
	fmt.Fprintf(&builder, "  Functions: %d\n", report.SymbolSummary.Functions)
	fmt.Fprintf(&builder, "  Methods: %d\n", report.SymbolSummary.Methods)
	fmt.Fprintf(&builder, "  Exported: %d\n", report.SymbolSummary.Exported)
	fmt.Fprintf(&builder, "  Test symbols: %d\n", report.SymbolSummary.Tests)
	builder.WriteString("\n")

	writeAnalyzePackageOverview(&builder, report.Packages)
	builder.WriteString("\n")
	writeAnalyzeSymbolProfiles(&builder, "IMPORTANT SYMBOLS", report.ImportantSymbols)
	builder.WriteString("\n")
	writeAnalyzeEntryPoints(&builder, report.EntryPoints)
	builder.WriteString("\n")
	writeAnalyzeRisk(&builder, report.Risk)
	builder.WriteString("\n")
	writeAnalyzeHotspots(&builder, report.Hotspots)
	builder.WriteString("\n")
	writeAnalyzeTesting(&builder, report.Testing)
	builder.WriteString("\n")

	builder.WriteString("READINESS\n")
	fmt.Fprintf(&builder, "  Package load: %s\n", report.Readiness.PackageLoad)
	fmt.Fprintf(&builder, "  Loaded packages: %d\n", report.Readiness.PackageLoadCount)
	fmt.Fprintf(&builder, "  Snapshot: %s\n", report.Readiness.SnapshotStatus)
	builder.WriteString("\n")

	writeDoctorSection(&builder, "SUGGESTED NEXT COMMANDS", report.Suggestions)
	builder.WriteString("\n")
	writeDoctorSection(&builder, "LIMITATIONS", report.Limitations)
	if len(report.Warnings) > 0 {
		builder.WriteString("\n")
		writeDoctorSection(&builder, "WARNINGS", report.Warnings)
	}

	return builder.String()
}

func writeAnalyzePackageOverview(builder *strings.Builder, packages []sherpa.PackageSummary) {
	builder.WriteString("PACKAGE OVERVIEW\n")
	if len(packages) == 0 {
		builder.WriteString("  none\n")
		return
	}

	fmt.Fprintf(builder, "  %-28s %5s %5s %7s %7s %7s\n", "PACKAGE", "GO", "TEST", "SYMBOLS", "IMPORTS", "USED BY")
	for _, pkg := range packages {
		fmt.Fprintf(
			builder,
			"  %-28s %5d %5d %7d %7d %7d\n",
			pkg.Package,
			pkg.GoFiles,
			pkg.TestFiles,
			pkg.Symbols,
			pkg.Imports,
			pkg.ImportedBy,
		)
	}
}

func writeAnalyzeSymbolProfiles(builder *strings.Builder, title string, symbols []analyzeSymbolProfile) {
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(symbols) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, symbol := range symbols {
		fmt.Fprintf(builder, "  %-10s %-34s %s:%d\n", symbol.Kind, symbol.QualifiedName, symbol.Position.File, symbol.Position.Line)
	}
}

func writeAnalyzeEntryPoints(builder *strings.Builder, entryPoints []sherpa.EntryPoint) {
	builder.WriteString("ENTRY POINTS\n")
	if len(entryPoints) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, entryPoint := range entryPoints {
		fmt.Fprintf(builder, "  %-10s %-34s %s:%d\n", entryPoint.Kind, entryPoint.Name, entryPoint.Position.File, entryPoint.Position.Line)
	}
}

func writeAnalyzeHotspots(builder *strings.Builder, hotspots []analyzeHotspot) {
	builder.WriteString("HOTSPOTS\n")
	if len(hotspots) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, hotspot := range hotspots {
		fmt.Fprintf(builder, "  %-28s %s\n", hotspot.Package, hotspot.Reason)
	}
}

func writeAnalyzeRisk(builder *strings.Builder, risk sherpa.RiskReport) {
	risk = normalizeAnalyzeRiskReport(risk)

	builder.WriteString("RISK\n")
	fmt.Fprintf(builder, "  Level: %s\n", risk.Level)
	fmt.Fprintf(builder, "  Score: %d\n", risk.Score)
	if len(risk.Factors) == 0 {
		builder.WriteString("  Factors: none\n")
		return
	}

	builder.WriteString("  Factors:\n")
	for _, factor := range risk.Factors {
		fmt.Fprintf(builder, "    - %s: %s\n", factor.Category, factor.Description)
	}
}

func writeAnalyzeTesting(builder *strings.Builder, testing analyzeTestingOverview) {
	builder.WriteString("TESTING\n")
	fmt.Fprintf(builder, "  Test files: %d\n", testing.TestFiles)
	writeDoctorValues(builder, "  Test packages", testing.TestPackages)
	writeDoctorValues(builder, "  Suggested tests", testing.SuggestedCommands)
}
