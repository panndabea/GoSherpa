package impact

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
	"github.com/panndabea/GoSherpa/internal/symbolindex"
)

const (
	AnalysisModeSnapshot            = "snapshot"
	AnalysisModeSnapshotTypechecked = "snapshot+typechecked"
	AnalysisModeSnapshotASTFallback = "snapshot+ast-fallback"
)

func cloneRelationshipIndex(index symbolindex.RelationshipIndex) symbolindex.RelationshipIndex {
	return symbolindex.RelationshipIndex{
		References:               append([]symbolindex.ReferenceRecord{}, index.References...),
		CallEdges:                append([]symbolindex.CallEdgeRecord{}, index.CallEdges...),
		PossibleCallEdges:        append([]symbolindex.PossibleCallEdgeRecord{}, index.PossibleCallEdges...),
		InterfaceImplementations: append([]symbolindex.InterfaceImplementationRecord{}, index.InterfaceImplementations...),
		TestReferences:           append([]symbolindex.TestReferenceRecord{}, index.TestReferences...),
		PackageRelationships:     append([]symbolindex.PackageRelationshipRecord{}, index.PackageRelationships...),
	}
}

func (a Analyzer) snapshotRelationshipsReady() bool {
	if !a.UseSnapshotRelationships {
		return false
	}
	return len(a.SnapshotRelationships.References) > 0 ||
		len(a.SnapshotRelationships.CallEdges) > 0 ||
		len(a.SnapshotRelationships.InterfaceImplementations) > 0 ||
		len(a.SnapshotRelationships.TestReferences) > 0
}

func (a Analyzer) analyzeChangedSymbolImpactsFromSnapshot(symbols []changedSymbol) (changedSymbolImpact, bool) {
	if !a.snapshotRelationshipsReady() || len(a.SnapshotSymbols) == 0 {
		return changedSymbolImpact{}, false
	}

	modulePath := impactModulePath(a.Root)
	seenTests := make(map[string]sherpa.RelatedTest)
	var impact changedSymbolImpact
	for _, changed := range normalizeChangedSymbols(symbols) {
		if changed.Deleted {
			return changedSymbolImpact{}, false
		}

		target := changedSymbolTestTarget(changed, modulePath)
		symbol, err := sherpa.FindSymbolTarget(a.Root, a.SnapshotSymbols, target)
		if err != nil {
			return changedSymbolImpact{}, false
		}

		references, referenceMode := a.snapshotReferencesForSymbol(symbol, "")
		callers, callMode := a.snapshotCallersForSymbol(symbol, false)
		impact.Packages = append(impact.Packages, packagesFromReferencePositions(references)...)
		impact.Packages = append(impact.Packages, packagesFromCallerPositions(callers)...)
		impact.ReferenceAnalysisMode = mergeSnapshotAnalysisMode(impact.ReferenceAnalysisMode, referenceMode)
		impact.CallAnalysisMode = mergeSnapshotAnalysisMode(impact.CallAnalysisMode, callMode)

		tests, testMode := a.snapshotDirectTestsForSymbol(symbol, target)
		for _, test := range tests {
			test = addRelatedTestReason(test, sherpa.RelatedTestReasonChangedSymbol)
			mergeRelatedTest(seenTests, test)
		}
		impact.TestAnalysisMode = mergeSnapshotAnalysisMode(impact.TestAnalysisMode, testMode)
	}

	impact.Packages = uniqueSortedStrings(impact.Packages)
	for _, test := range seenTests {
		impact.Tests = append(impact.Tests, test)
	}
	sortRelatedTests(impact.Tests)

	return impact, true
}

func (a Analyzer) analyzeSymbolFromSnapshot(target string, context *sherpa.SemanticContext) (ImpactReport, bool, error) {
	if !a.snapshotRelationshipsReady() || len(a.SnapshotSymbols) == 0 {
		return ImpactReport{}, false, nil
	}

	symbol, err := sherpa.FindSymbolTarget(a.Root, a.SnapshotSymbols, target)
	if err != nil {
		if strings.Contains(err.Error(), "ambiguous") {
			return ImpactReport{}, false, err
		}
		return ImpactReport{}, false, nil
	}

	report := ImpactReport{
		AffectedSymbols: []string{snapshotSymbolTarget(symbol, impactModulePath(a.Root))},
	}

	references, referenceMode := a.snapshotReferencesForSymbol(symbol, "")
	callers, callMode := a.snapshotCallersForSymbol(symbol, false)
	report.ReferenceAnalysisMode = referenceMode
	report.CallAnalysisMode = callMode
	report.AffectedPackages = uniqueSortedStrings(append(packagesFromReferencePositions(references), packagesFromCallerPositions(callers)...))

	signals, err := a.interfaceSignalsForSymbolWithSnapshot(context, snapshotSymbolTarget(symbol, impactModulePath(a.Root)), InterfaceOptions{
		BuildTags: a.BuildTags,
	})
	if err != nil {
		return ImpactReport{}, false, err
	}
	report.AffectedInterfaces = signals.Interfaces
	report.AffectedImplementations = signals.Implementations
	report.InterfaceAnalysisMode = signals.AnalysisMode
	report.Warnings = uniqueSortedStrings(append(report.Warnings, signals.Warnings...))

	contractPackages := contractPackagesForSignals(signals)
	report.AffectedPackages = uniqueSortedStrings(append(report.AffectedPackages, contractPackages...))

	targetPackages := nonEmptyStrings(symbol.Package)
	fallbackPackages := uniqueSortedStrings(append(append([]string{}, report.AffectedPackages...), targetPackages...))
	directTests, testMode := a.snapshotDirectTestsForSymbol(symbol, snapshotSymbolTarget(symbol, impactModulePath(a.Root)))
	seenTests := make(map[string]sherpa.RelatedTest)
	for _, test := range directTests {
		mergeRelatedTest(seenTests, test)
	}
	targetPackageSet := impactStringSet(targetPackages)
	contractPackageSet := impactStringSet(contractPackages)
	for _, pkg := range fallbackPackages {
		tests, err := findTestsWithContext(context, a.Root, pkg, sherpa.TestOptions{Scope: sherpa.TestScopeAll})
		if err != nil {
			report.Warnings = append(report.Warnings, err.Error())
			continue
		}
		testMode = mergeSnapshotAnalysisMode(testMode, tests.AnalysisMode)
		report.Warnings = append(report.Warnings, tests.Warnings...)
		for _, test := range tests.Tests {
			if _, ok := targetPackageSet[pkg]; !ok {
				test = removeRelatedTestReason(test, sherpa.RelatedTestReasonTargetPackage)
				test = addRelatedTestReason(test, sherpa.RelatedTestReasonCallerPackage)
			}
			if _, ok := contractPackageSet[pkg]; ok {
				test = addRelatedTestReason(test, sherpa.RelatedTestReasonContract)
			}
			mergeRelatedTest(seenTests, test)
		}
	}
	report.AffectedTests = make([]sherpa.RelatedTest, 0, len(seenTests))
	for _, test := range seenTests {
		report.AffectedTests = append(report.AffectedTests, test)
	}
	sortRelatedTests(report.AffectedTests)
	report.Warnings = uniqueSortedStrings(report.Warnings)
	report.TestAnalysisMode = normalizeSnapshotTestAnalysisMode(testMode)
	plan := sherpa.PlanTests(report.AffectedTests, sherpa.TestPlanOptions{
		Target:           snapshotSymbolTarget(symbol, impactModulePath(a.Root)),
		Kind:             sherpa.TestTargetKindSymbol,
		TargetPackages:   targetPackages,
		ContractPackages: contractPackages,
		CallerPackages:   packageDifference(report.AffectedPackages, targetPackages),
		FallbackPackages: fallbackPackages,
		Targets:          nonEmptyStrings(snapshotSymbolTarget(symbol, impactModulePath(a.Root))),
	})
	report.TestPlan = plan
	report.TestCommands = sherpa.TestPlanCommands(plan)

	return normalizeReport(report), true, nil
}

func (a Analyzer) snapshotReferencesForSymbol(symbol sherpa.Symbol, referenceKind sherpa.ReferenceKind) ([]sherpa.Reference, string) {
	var references []sherpa.Reference
	var modes []string
	for _, record := range a.SnapshotRelationships.References {
		if referenceKind != "" && record.ReferenceKind != referenceKind {
			continue
		}
		if !snapshotIdentityMatchesSymbol(record.Target, symbol) {
			continue
		}

		modes = append(modes, record.AnalysisMode)
		references = append(references, sherpa.Reference{
			Name:     snapshotSymbolTarget(symbol, impactModulePath(a.Root)),
			Kind:     record.ReferenceKind,
			Position: record.Position,
			Range:    record.Range,
		})
	}
	sort.Slice(references, func(i int, j int) bool {
		return positionSortKey(references[i].Position) < positionSortKey(references[j].Position)
	})

	return references, snapshotRelationshipAnalysisModeWithFallback(modes, len(a.SnapshotRelationships.References) > 0)
}

func (a Analyzer) snapshotCallersForSymbol(symbol sherpa.Symbol, includeTests bool) ([]sherpa.Caller, string) {
	var callers []sherpa.Caller
	var modes []string
	for _, record := range a.SnapshotRelationships.CallEdges {
		if !includeTests && snapshotRelationshipFromTestFile(record.Source, record.File) {
			continue
		}
		if !snapshotIdentityMatchesSymbol(record.Target, symbol) {
			continue
		}

		modes = append(modes, record.AnalysisMode)
		callers = append(callers, sherpa.Caller{
			Name:     snapshotIdentityDisplayName(record.Source),
			Position: record.Position,
			Range:    record.Range,
		})
	}
	sort.Slice(callers, func(i int, j int) bool {
		if callers[i].Name != callers[j].Name {
			return callers[i].Name < callers[j].Name
		}
		return positionSortKey(callers[i].Position) < positionSortKey(callers[j].Position)
	})

	return callers, snapshotRelationshipAnalysisModeWithFallback(modes, len(a.SnapshotRelationships.CallEdges) > 0)
}

func (a Analyzer) snapshotDirectTestsForSymbol(symbol sherpa.Symbol, target string) ([]sherpa.RelatedTest, string) {
	seen := make(map[string]sherpa.RelatedTest)
	var modes []string
	for _, record := range a.SnapshotRelationships.References {
		if !snapshotIdentityMatchesSymbol(record.Target, symbol) || !snapshotRelationshipFromTestFile(record.Source, record.File) {
			continue
		}
		if test, ok := snapshotRelatedTestFromReference(record.Source, record.Position, record.Range, symbol.Package, target); ok {
			modes = append(modes, record.AnalysisMode)
			mergeRelatedTest(seen, test)
		}
	}
	for _, record := range a.SnapshotRelationships.CallEdges {
		if !snapshotIdentityMatchesSymbol(record.Target, symbol) || !snapshotRelationshipFromTestFile(record.Source, record.File) {
			continue
		}
		if test, ok := snapshotRelatedTestFromReference(record.Source, record.Position, record.Range, symbol.Package, target); ok {
			modes = append(modes, record.AnalysisMode)
			mergeRelatedTest(seen, test)
		}
	}
	for _, record := range a.SnapshotRelationships.TestReferences {
		if !snapshotIdentityMatchesSymbol(record.Target, symbol) {
			continue
		}
		if test, ok := snapshotRelatedTestFromReference(record.Test, record.Position, record.Range, symbol.Package, target); ok {
			modes = append(modes, record.AnalysisMode)
			mergeRelatedTest(seen, test)
		}
	}

	tests := make([]sherpa.RelatedTest, 0, len(seen))
	for _, test := range seen {
		tests = append(tests, test)
	}
	sortRelatedTests(tests)

	hasTestRecords := len(a.SnapshotRelationships.References) > 0 ||
		len(a.SnapshotRelationships.CallEdges) > 0 ||
		len(a.SnapshotRelationships.TestReferences) > 0
	return tests, snapshotRelationshipAnalysisModeWithFallback(modes, hasTestRecords)
}

func snapshotRelatedTestFromReference(source symbolindex.SymbolIdentity, position sherpa.Position, sourceRange *sherpa.SourceRange, targetPackage string, target string) (sherpa.RelatedTest, bool) {
	name := snapshotIdentityDisplayName(source)
	if !strings.HasPrefix(name, "Test") || source.Position.File == "" {
		return sherpa.RelatedTest{}, false
	}

	packageMatches := source.Package == targetPackage
	externalPackage := strings.HasSuffix(source.PackageName, "_test")
	reasons := []string{sherpa.RelatedTestReasonDirectReference}
	if packageMatches {
		reasons = append(reasons, sherpa.RelatedTestReasonSamePackage)
	}
	if externalPackage {
		reasons = append(reasons, sherpa.RelatedTestReasonExternalPackage)
	}

	return sherpa.RelatedTest{
		Name:            name,
		Package:         source.Package,
		PackageName:     firstNonEmptyImpactString(source.PackageName, path.Base(strings.TrimPrefix(source.Package, "./"))),
		Position:        source.Position,
		Range:           source.Range,
		DirectReference: true,
		ExternalPackage: externalPackage,
		Reasons:         uniqueSortedStrings(reasons),
		Targets:         nonEmptyStrings(target),
		TargetReferences: []sherpa.RelatedTestTargetReference{
			{
				Target:   target,
				Position: position,
				Range:    sourceRange,
			},
		},
	}, true
}

func (a Analyzer) interfaceSignalsForPackagesWithSnapshot(context *sherpa.SemanticContext, packages []string, options InterfaceOptions) (interfaceImpactSignals, error) {
	if a.snapshotRelationshipsReady() && len(a.SnapshotRelationships.InterfaceImplementations) > 0 {
		return snapshotInterfaceSignalsForPackages(a.SnapshotRelationships, packages), nil
	}

	return interfaceSignalsForPackagesWithContext(context, a.Root, packages, options)
}

func (a Analyzer) interfaceSignalsForSymbolWithSnapshot(context *sherpa.SemanticContext, target string, options InterfaceOptions) (interfaceImpactSignals, error) {
	if a.snapshotRelationshipsReady() && len(a.SnapshotRelationships.InterfaceImplementations) > 0 && len(a.SnapshotSymbols) > 0 {
		symbol, err := sherpa.FindSymbolTarget(a.Root, a.SnapshotSymbols, target)
		if err == nil && symbol.Receiver == "" {
			return snapshotInterfaceSignalsForSymbol(a.SnapshotRelationships, symbol), nil
		}
	}

	return interfaceSignalsForSymbolWithContext(context, a.Root, target, options)
}

func snapshotInterfaceSignalsForPackages(relationships symbolindex.RelationshipIndex, packages []string) interfaceImpactSignals {
	packageSet := impactStringSet(packages)
	implementersByInterface := snapshotImplementersByInterface(relationships)

	var interfaces []string
	var implementations []string
	var modes []string
	for _, record := range relationships.InterfaceImplementations {
		interfaceName := snapshotIdentityQualifiedName(record.Interface)
		implementationName := snapshotIdentityQualifiedName(record.Implementation)
		if _, ok := packageSet[record.Interface.Package]; ok {
			interfaces = append(interfaces, interfaceName)
			implementations = append(implementations, implementersByInterface[interfaceName]...)
			modes = append(modes, record.AnalysisMode)
		}
		if _, ok := packageSet[record.Implementation.Package]; ok {
			interfaces = append(interfaces, interfaceName)
			implementations = append(implementations, implementationName)
			modes = append(modes, record.AnalysisMode)
		}
	}

	return interfaceImpactSignals{
		Interfaces:      uniqueSortedStrings(interfaces),
		Implementations: uniqueSortedStrings(implementations),
		AnalysisMode:    snapshotRelationshipAnalysisMode(modes),
		Warnings:        []string{},
	}
}

func snapshotInterfaceSignalsForSymbol(relationships symbolindex.RelationshipIndex, symbol sherpa.Symbol) interfaceImpactSignals {
	var interfaces []string
	var implementations []string
	var modes []string
	for _, record := range relationships.InterfaceImplementations {
		if snapshotIdentityMatchesSymbol(record.Interface, symbol) {
			interfaces = append(interfaces, snapshotIdentityQualifiedName(record.Interface))
			implementations = append(implementations, snapshotIdentityQualifiedName(record.Implementation))
			modes = append(modes, record.AnalysisMode)
			continue
		}
		if snapshotIdentityMatchesSymbol(record.Implementation, symbol) {
			interfaces = append(interfaces, snapshotIdentityQualifiedName(record.Interface))
			implementations = append(implementations, snapshotIdentityQualifiedName(record.Implementation))
			modes = append(modes, record.AnalysisMode)
		}
	}

	return interfaceImpactSignals{
		Interfaces:      uniqueSortedStrings(interfaces),
		Implementations: uniqueSortedStrings(implementations),
		AnalysisMode:    snapshotRelationshipAnalysisMode(modes),
		Warnings:        []string{},
	}
}

func snapshotImplementersByInterface(relationships symbolindex.RelationshipIndex) map[string][]string {
	result := make(map[string][]string)
	for _, record := range relationships.InterfaceImplementations {
		if record.Kind != symbolindex.RelationshipKindInterfaceImplementation {
			continue
		}
		interfaceName := snapshotIdentityQualifiedName(record.Interface)
		implementationName := snapshotIdentityQualifiedName(record.Implementation)
		result[interfaceName] = append(result[interfaceName], implementationName)
	}
	for iface, implementations := range result {
		result[iface] = uniqueSortedStrings(implementations)
	}

	return result
}

func packagesFromReferencePositions(references []sherpa.Reference) []string {
	files := make([]string, 0, len(references))
	for _, reference := range references {
		files = append(files, reference.Position.File)
	}

	return packagesFromPositionFiles(files)
}

func packagesFromCallerPositions(callers []sherpa.Caller) []string {
	files := make([]string, 0, len(callers))
	for _, caller := range callers {
		files = append(files, caller.Position.File)
	}

	return packagesFromPositionFiles(files)
}

func packagesFromPositionFiles(files []string) []string {
	var packages []string
	for _, file := range files {
		file = strings.TrimSpace(filepath.ToSlash(file))
		if file == "" || filepath.IsAbs(file) {
			continue
		}
		if pkg, ok := packageForChangedFile(file); ok {
			packages = append(packages, pkg)
		}
	}

	return uniqueSortedStrings(packages)
}

func snapshotIdentityMatchesSymbol(identity symbolindex.SymbolIdentity, symbol sherpa.Symbol) bool {
	if strings.TrimSpace(identity.Package) != "" && identity.Package != symbol.Package {
		return false
	}
	if identity.Name != symbol.Name {
		return false
	}

	return identity.Receiver == symbol.Receiver
}

func snapshotIdentityDisplayName(identity symbolindex.SymbolIdentity) string {
	if strings.TrimSpace(identity.Package) == "" && strings.TrimSpace(identity.QualifiedName) != "" {
		return identity.QualifiedName
	}
	if strings.TrimSpace(identity.Receiver) != "" {
		return identity.Receiver + "." + identity.Name
	}
	if strings.TrimSpace(identity.Name) != "" {
		return identity.Name
	}

	return identity.QualifiedName
}

func snapshotIdentityQualifiedName(identity symbolindex.SymbolIdentity) string {
	if strings.TrimSpace(identity.QualifiedName) != "" {
		return identity.QualifiedName
	}

	return snapshotIdentityDisplayName(identity)
}

func snapshotSymbolTarget(symbol sherpa.Symbol, modulePath string) string {
	if strings.TrimSpace(symbol.QualifiedName) != "" {
		return symbol.QualifiedName
	}

	return sherpa.FormatPackageQualifiedTarget(symbol.Package, symbol.DisplayName(), modulePath)
}

func snapshotRelationshipFromTestFile(identity symbolindex.SymbolIdentity, file string) bool {
	return strings.HasSuffix(identity.Position.File, "_test.go") || strings.HasSuffix(file, "_test.go")
}

func snapshotRelationshipAnalysisMode(modes []string) string {
	return snapshotRelationshipAnalysisModeWithFallback(modes, len(modes) > 0)
}

func snapshotRelationshipAnalysisModeWithFallback(modes []string, used bool) string {
	hasTypechecked := false
	hasASTFallback := false
	for _, mode := range modes {
		switch strings.TrimSpace(mode) {
		case "typechecked":
			hasTypechecked = true
		case "ast-fallback":
			hasASTFallback = true
		case AnalysisModeSnapshotTypechecked:
			hasTypechecked = true
		case AnalysisModeSnapshotASTFallback:
			hasASTFallback = true
		}
	}
	if hasTypechecked {
		return AnalysisModeSnapshotTypechecked
	}
	if hasASTFallback {
		return AnalysisModeSnapshotASTFallback
	}
	if used {
		return AnalysisModeSnapshot
	}

	return ""
}

func mergeSnapshotAnalysisMode(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if next == "" {
		return current
	}
	if current == "" {
		return next
	}
	if current == AnalysisModeSnapshotTypechecked || next == AnalysisModeSnapshotTypechecked {
		return AnalysisModeSnapshotTypechecked
	}
	if current == AnalysisModeSnapshotASTFallback || next == AnalysisModeSnapshotASTFallback {
		return AnalysisModeSnapshotASTFallback
	}
	if current == AnalysisModeSnapshot || next == AnalysisModeSnapshot {
		return AnalysisModeSnapshot
	}

	return mergeAnalysisMode(current, next, sherpa.ReferenceAnalysisModeTypechecked, sherpa.ReferenceAnalysisModeASTFallback)
}

func normalizeSnapshotTestAnalysisMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode != "" {
		return mode
	}

	return sherpa.TestAnalysisModeAST
}

func positionSortKey(position sherpa.Position) string {
	return strings.Join([]string{
		position.File,
		intKey(position.Line),
		intKey(position.Column),
	}, "\x00")
}
