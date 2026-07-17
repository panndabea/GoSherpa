package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
	"github.com/panndabea/GoSherpa/internal/symbolindex"
)

const (
	analysisModeSnapshot            = "snapshot"
	analysisModeSnapshotTypechecked = "snapshot+typechecked"
	analysisModeSnapshotASTFallback = "snapshot+ast-fallback"
)

func loadSymbolsForInventoryCommand(root string, invocation cliInvocation) ([]sherpa.Symbol, []string, string, error) {
	var warnings []string
	if invocation.UseSnapshot {
		stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{
			BuildTags: invocation.BuildTags,
		})
		if inspect.Status == snapshotstore.StatusValid {
			return cloneSlice(stored.Symbols), nil, analysisModeSnapshot, nil
		}

		warnings = append(warnings, snapshotFallbackWarning(inspect))
	}

	symbols, err := sherpa.ParseRepository(root)
	if err != nil {
		return nil, warnings, "", err
	}

	return symbols, warnings, fallbackAnalysisMode(invocation), nil
}

func loadPackagesForInventoryCommand(root string, invocation cliInvocation) ([]sherpa.PackageSummary, []string, string, error) {
	var warnings []string
	if invocation.UseSnapshot && invocation.IncludeTests {
		stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{
			BuildTags: invocation.BuildTags,
		})
		if inspect.Status == snapshotstore.StatusValid {
			return cloneSlice(stored.Packages), nil, analysisModeSnapshot, nil
		}

		warnings = append(warnings, snapshotFallbackWarning(inspect))
	} else if invocation.UseSnapshot {
		warnings = append(warnings, "snapshot not used: packages without --tests need production-only package counts; using live repository analysis")
	}

	packages, err := sherpa.FindPackageSummaries(root, sherpa.PackageInventoryOptions{
		IncludeTests: invocation.IncludeTests,
	})
	if err != nil {
		return nil, warnings, "", err
	}

	return packages, warnings, fallbackAnalysisMode(invocation), nil
}

func loadDiffAnalyzerOptions(root string, buildTags []string, useSnapshot bool) (impactengine.AnalyzerOptions, bool, []string) {
	options := impactengine.AnalyzerOptions{
		BuildTags: buildTags,
	}
	if !useSnapshot {
		return options, false, nil
	}

	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{
		BuildTags: buildTags,
	})
	if inspect.Status == snapshotstore.StatusValid {
		options.UseSnapshotSymbols = true
		options.SnapshotSymbols = cloneSlice(stored.Symbols)
		if relationshipSnapshotHasReusableData(stored.Relationships) {
			options.UseSnapshotRelationships = true
			options.SnapshotRelationships = stored.Relationships
		}
		return options, true, nil
	}

	return options, false, []string{snapshotFallbackWarning(inspect)}
}

func relationshipSnapshotHasReusableData(relationships symbolindex.RelationshipIndex) bool {
	return len(relationships.References) > 0 ||
		len(relationships.CallEdges) > 0 ||
		len(relationships.InterfaceImplementations) > 0 ||
		len(relationships.TestReferences) > 0
}

func loadReferenceReportForCommand(root string, invocation cliInvocation, target string) (sherpa.ReferenceReport, error) {
	if invocation.UseSnapshot {
		report, used, warnings, err := referenceReportFromSnapshot(root, invocation, target)
		if err != nil {
			return sherpa.ReferenceReport{}, err
		}
		if used {
			return report, nil
		}

		live, err := sherpa.FindReferenceReportWithOptions(root, target, sherpa.ReferenceOptions{
			Kind:      invocation.ReferenceKind,
			BuildTags: invocation.BuildTags,
		})
		live.Warnings = uniqueStringsInOrder(append(warnings, live.Warnings...))
		return live, err
	}

	return sherpa.FindReferenceReportWithOptions(root, target, sherpa.ReferenceOptions{
		Kind:      invocation.ReferenceKind,
		BuildTags: invocation.BuildTags,
	})
}

func loadCallersForCommand(root string, invocation cliInvocation, target string) (sherpa.CallersResult, error) {
	if invocation.UseSnapshot {
		result, used, warnings, err := callersFromSnapshot(root, invocation, target)
		if err != nil {
			return sherpa.CallersResult{}, err
		}
		if used {
			return result, nil
		}

		live, err := sherpa.FindCallersWithOptions(root, target, sherpa.CallOptions{
			IncludeTests: invocation.IncludeTests,
			BuildTags:    invocation.BuildTags,
		})
		live.Warnings = uniqueStringsInOrder(append(warnings, live.Warnings...))
		return live, err
	}

	return sherpa.FindCallersWithOptions(root, target, sherpa.CallOptions{
		IncludeTests: invocation.IncludeTests,
		BuildTags:    invocation.BuildTags,
	})
}

func loadCalleesForCommand(root string, invocation cliInvocation, target string) (sherpa.CalleesResult, error) {
	if invocation.UseSnapshot {
		result, used, warnings, err := calleesFromSnapshot(root, invocation, target)
		if err != nil {
			return sherpa.CalleesResult{}, err
		}
		if used {
			return result, nil
		}

		live, err := sherpa.FindCalleesWithOptions(root, target, sherpa.CallOptions{
			BuildTags: invocation.BuildTags,
		})
		live.Warnings = uniqueStringsInOrder(append(warnings, live.Warnings...))
		return live, err
	}

	return sherpa.FindCalleesWithOptions(root, target, sherpa.CallOptions{
		BuildTags: invocation.BuildTags,
	})
}

func loadImplementersForCommand(root string, invocation cliInvocation, target string) (impactengine.ImplementersResult, error) {
	options := impactengine.InterfaceOptions{BuildTags: invocation.BuildTags}
	if invocation.UseSnapshot {
		result, used, warnings, err := implementersFromSnapshot(root, invocation, target)
		if err != nil {
			return impactengine.ImplementersResult{}, err
		}
		if used {
			return result, nil
		}

		semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
			BuildTags: invocation.BuildTags,
		})
		if err != nil {
			return impactengine.ImplementersResult{}, err
		}
		live, err := impactengine.FindImplementersWithContext(semanticContext, target, options)
		live.Warnings = uniqueStringsInOrder(append(warnings, live.Warnings...))
		return live, err
	}

	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return impactengine.ImplementersResult{}, err
	}

	return impactengine.FindImplementersWithContext(semanticContext, target, options)
}

func loadInterfacesForCommand(root string, invocation cliInvocation, target string) (impactengine.InterfacesResult, error) {
	options := impactengine.InterfaceOptions{BuildTags: invocation.BuildTags}
	if invocation.UseSnapshot {
		result, used, warnings, err := interfacesFromSnapshot(root, invocation, target)
		if err != nil {
			return impactengine.InterfacesResult{}, err
		}
		if used {
			return result, nil
		}

		semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
			BuildTags: invocation.BuildTags,
		})
		if err != nil {
			return impactengine.InterfacesResult{}, err
		}
		live, err := impactengine.FindInterfacesWithContext(semanticContext, target, options)
		live.Warnings = uniqueStringsInOrder(append(warnings, live.Warnings...))
		return live, err
	}

	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return impactengine.InterfacesResult{}, err
	}

	return impactengine.FindInterfacesWithContext(semanticContext, target, options)
}

func loadInterfaceForCommand(root string, invocation cliInvocation, target string) (impactengine.InterfaceResult, error) {
	options := impactengine.InterfaceOptions{BuildTags: invocation.BuildTags}
	if invocation.UseSnapshot {
		result, used, warnings, err := interfaceFromSnapshot(root, invocation, target)
		if err != nil {
			return impactengine.InterfaceResult{}, err
		}
		if used {
			return result, nil
		}

		semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
			BuildTags: invocation.BuildTags,
		})
		if err != nil {
			return impactengine.InterfaceResult{}, err
		}
		live, err := impactengine.InspectInterfaceWithContext(semanticContext, target, options)
		live.Warnings = uniqueStringsInOrder(append(warnings, live.Warnings...))
		return live, err
	}

	semanticContext, err := sherpa.NewSemanticContext(root, sherpa.SemanticContextOptions{
		BuildTags: invocation.BuildTags,
	})
	if err != nil {
		return impactengine.InterfaceResult{}, err
	}

	return impactengine.InspectInterfaceWithContext(semanticContext, target, options)
}

func referenceReportFromSnapshot(root string, invocation cliInvocation, target string) (sherpa.ReferenceReport, bool, []string, error) {
	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{BuildTags: invocation.BuildTags})
	if inspect.Status != snapshotstore.StatusValid {
		return sherpa.ReferenceReport{}, false, []string{snapshotFallbackWarning(inspect)}, nil
	}
	if len(stored.Relationships.References) == 0 {
		return sherpa.ReferenceReport{}, false, []string{snapshotRelationshipFallbackWarning("reference")}, nil
	}

	symbol, ok, warning, err := snapshotTargetSymbol(root, stored, target)
	if err != nil || !ok {
		return sherpa.ReferenceReport{}, false, optionalWarning(warning), err
	}

	var references []sherpa.Reference
	var modes []string
	for _, record := range stored.Relationships.References {
		if invocation.ReferenceKind != "" && record.ReferenceKind != invocation.ReferenceKind {
			continue
		}
		if !snapshotIdentityMatchesSymbol(record.Target, symbol) {
			continue
		}

		modes = append(modes, record.AnalysisMode)
		references = append(references, sherpa.Reference{
			Name:     target,
			Kind:     record.ReferenceKind,
			Position: record.Position,
			Range:    record.Range,
		})
	}
	sortReferencesForSnapshot(references)

	return sherpa.ReferenceReport{
		Target:       target,
		References:   nonNilSlice(references),
		AnalysisMode: snapshotRelationshipAnalysisMode(modes),
		Warnings:     []string{},
	}, true, nil, nil
}

func callersFromSnapshot(root string, invocation cliInvocation, target string) (sherpa.CallersResult, bool, []string, error) {
	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{BuildTags: invocation.BuildTags})
	if inspect.Status != snapshotstore.StatusValid {
		return sherpa.CallersResult{}, false, []string{snapshotFallbackWarning(inspect)}, nil
	}
	if len(stored.Relationships.CallEdges) == 0 {
		return sherpa.CallersResult{}, false, []string{snapshotRelationshipFallbackWarning("call")}, nil
	}

	symbol, ok, warning, err := snapshotTargetSymbol(root, stored, target)
	if err != nil || !ok {
		return sherpa.CallersResult{}, false, optionalWarning(warning), err
	}

	var callers []sherpa.Caller
	var limitations []string
	var modes []string
	for _, record := range stored.Relationships.CallEdges {
		if !invocation.IncludeTests && snapshotRecordFromTestFile(record.Source, record.File) {
			continue
		}
		if !snapshotIdentityMatchesSymbol(record.Target, symbol) {
			continue
		}

		modes = append(modes, record.AnalysisMode)
		limitations = append(limitations, record.Limitations...)
		callers = append(callers, sherpa.Caller{
			Name:     snapshotIdentityDisplayName(record.Source),
			Position: record.Position,
			Range:    record.Range,
		})
	}
	sortCallersForSnapshot(callers)

	return sherpa.CallersResult{
		Target:       target,
		AnalysisMode: snapshotRelationshipAnalysisMode(modes),
		Warnings:     []string{},
		Limitations:  uniqueStringsInOrder(limitations),
		Callers:      nonNilSlice(callers),
	}, true, nil, nil
}

func calleesFromSnapshot(root string, invocation cliInvocation, target string) (sherpa.CalleesResult, bool, []string, error) {
	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{BuildTags: invocation.BuildTags})
	if inspect.Status != snapshotstore.StatusValid {
		return sherpa.CalleesResult{}, false, []string{snapshotFallbackWarning(inspect)}, nil
	}
	if len(stored.Relationships.CallEdges) == 0 {
		return sherpa.CalleesResult{}, false, []string{snapshotRelationshipFallbackWarning("call")}, nil
	}

	symbol, ok, warning, err := snapshotTargetSymbol(root, stored, target)
	if err != nil || !ok {
		return sherpa.CalleesResult{}, false, optionalWarning(warning), err
	}
	if strings.HasSuffix(symbol.Position.File, "_test.go") {
		return sherpa.CalleesResult{}, false, []string{"snapshot not used: callee analysis for test-file targets is not part of the current CLI contract; using live repository analysis"}, nil
	}

	var callees []sherpa.Callee
	var limitations []string
	var modes []string
	for _, record := range stored.Relationships.CallEdges {
		if !snapshotIdentityMatchesSymbol(record.Source, symbol) {
			continue
		}

		modes = append(modes, record.AnalysisMode)
		limitations = append(limitations, record.Limitations...)
		callees = append(callees, sherpa.Callee{
			Name:     snapshotIdentityDisplayName(record.Target),
			Scope:    record.CallScope,
			Position: record.Position,
			Range:    record.Range,
		})
	}
	sortCalleesForSnapshot(callees)

	return sherpa.CalleesResult{
		Target:       target,
		AnalysisMode: snapshotRelationshipAnalysisMode(modes),
		Warnings:     []string{},
		Limitations:  uniqueStringsInOrder(limitations),
		Callees:      nonNilSlice(callees),
	}, true, nil, nil
}

func implementersFromSnapshot(root string, invocation cliInvocation, target string) (impactengine.ImplementersResult, bool, []string, error) {
	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{BuildTags: invocation.BuildTags})
	if inspect.Status != snapshotstore.StatusValid {
		return impactengine.ImplementersResult{}, false, []string{snapshotFallbackWarning(inspect)}, nil
	}
	if len(stored.Relationships.InterfaceImplementations) == 0 {
		return impactengine.ImplementersResult{}, false, []string{snapshotRelationshipFallbackWarning("interface")}, nil
	}

	symbol, ok, warning, err := snapshotTargetSymbol(root, stored, target)
	if err != nil || !ok {
		return impactengine.ImplementersResult{}, false, optionalWarning(warning), err
	}

	var implementers []impactengine.Implementer
	var modes []string
	for _, record := range stored.Relationships.InterfaceImplementations {
		if record.Kind != symbolindex.RelationshipKindInterfaceImplementation {
			continue
		}
		if !snapshotIdentityMatchesSymbol(record.Interface, symbol) {
			continue
		}

		modes = append(modes, record.AnalysisMode)
		implementers = append(implementers, impactengine.Implementer{
			Name:     snapshotIdentityQualifiedName(record.Implementation),
			Position: record.Implementation.Position,
		})
	}
	sortImplementersForSnapshot(implementers)

	return impactengine.ImplementersResult{
		Target:       snapshotSymbolTarget(symbol),
		Implementers: nonNilSlice(implementers),
		AnalysisMode: snapshotRelationshipAnalysisMode(modes),
		Warnings:     []string{},
	}, true, nil, nil
}

func interfacesFromSnapshot(root string, invocation cliInvocation, target string) (impactengine.InterfacesResult, bool, []string, error) {
	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{BuildTags: invocation.BuildTags})
	if inspect.Status != snapshotstore.StatusValid {
		return impactengine.InterfacesResult{}, false, []string{snapshotFallbackWarning(inspect)}, nil
	}
	if len(stored.Relationships.InterfaceImplementations) == 0 {
		return impactengine.InterfacesResult{}, false, []string{snapshotRelationshipFallbackWarning("interface")}, nil
	}

	symbol, ok, warning, err := snapshotTargetSymbol(root, stored, target)
	if err != nil || !ok {
		return impactengine.InterfacesResult{}, false, optionalWarning(warning), err
	}

	var interfaces []impactengine.SatisfiedInterface
	var modes []string
	for _, record := range stored.Relationships.InterfaceImplementations {
		if record.Kind != symbolindex.RelationshipKindSatisfiedInterface {
			continue
		}
		if !snapshotIdentityMatchesSymbol(record.Implementation, symbol) {
			continue
		}

		modes = append(modes, record.AnalysisMode)
		interfaces = append(interfaces, impactengine.SatisfiedInterface{
			Name:     snapshotIdentityQualifiedName(record.Interface),
			Position: record.Interface.Position,
		})
	}
	sortInterfacesForSnapshot(interfaces)

	return impactengine.InterfacesResult{
		Target:       snapshotSymbolTarget(symbol),
		Interfaces:   nonNilSlice(interfaces),
		AnalysisMode: snapshotRelationshipAnalysisMode(modes),
		Warnings:     []string{},
	}, true, nil, nil
}

func interfaceFromSnapshot(root string, invocation cliInvocation, target string) (impactengine.InterfaceResult, bool, []string, error) {
	stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{BuildTags: invocation.BuildTags})
	if inspect.Status != snapshotstore.StatusValid {
		return impactengine.InterfaceResult{}, false, []string{snapshotFallbackWarning(inspect)}, nil
	}
	if len(stored.Relationships.InterfaceImplementations) == 0 {
		return impactengine.InterfaceResult{}, false, []string{snapshotRelationshipFallbackWarning("interface")}, nil
	}

	symbol, ok, warning, err := snapshotTargetSymbol(root, stored, target)
	if err != nil || !ok {
		return impactengine.InterfaceResult{}, false, optionalWarning(warning), err
	}

	implementersResult, _, _, err := implementersFromSnapshot(root, invocation, target)
	if err != nil {
		return impactengine.InterfaceResult{}, false, nil, err
	}
	referenceReport, _, _, err := referenceReportFromSnapshot(root, invocation, target)
	if err != nil {
		return impactengine.InterfaceResult{}, false, nil, err
	}

	methods := make([]impactengine.InterfaceMethod, 0, len(symbol.Methods))
	for _, method := range symbol.Methods {
		methods = append(methods, impactengine.InterfaceMethod{
			Name:      method.Name,
			Signature: method.Signature,
			Usages:    []impactengine.InterfaceMethodUsage{},
		})
	}

	return impactengine.InterfaceResult{
		Target:                  snapshotSymbolTarget(symbol),
		Position:                symbol.Position,
		Methods:                 methods,
		Implementers:            implementersResult.Implementers,
		References:              referenceReport.References,
		AnalysisMode:            implementersResult.AnalysisMode,
		ReferenceAnalysisMode:   referenceReport.AnalysisMode,
		MethodUsageAnalysisMode: analysisModeSnapshot,
		Warnings:                []string{},
		Limitations:             []string{"Snapshot interface profiles reuse persisted interface and reference records; interface method usage records are not persisted in this slice."},
	}, true, nil, nil
}

func snapshotTargetSymbol(root string, stored snapshotstore.Snapshot, target string) (sherpa.Symbol, bool, string, error) {
	symbol, err := sherpa.FindSymbolTarget(root, stored.Symbols, target)
	if err == nil {
		return symbol, true, "", nil
	}

	var ambiguous *sherpa.AmbiguousTargetError
	if errors.As(err, &ambiguous) {
		return sherpa.Symbol{}, false, "", err
	}

	return sherpa.Symbol{}, false, "snapshot not used: target is not represented in snapshot symbol definitions; using live repository analysis", nil
}

func snapshotRelationshipFallbackWarning(kind string) string {
	return fmt.Sprintf("snapshot not used: valid snapshot has no %s relationship records; using live repository analysis", kind)
}

func snapshotRelationshipAnalysisMode(modes []string) string {
	hasTypechecked := false
	hasASTFallback := false
	for _, mode := range modes {
		switch strings.TrimSpace(mode) {
		case "typechecked":
			hasTypechecked = true
		case "ast-fallback":
			hasASTFallback = true
		}
	}
	if hasTypechecked {
		return analysisModeSnapshotTypechecked
	}
	if hasASTFallback {
		return analysisModeSnapshotASTFallback
	}

	return analysisModeSnapshot
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

func snapshotSymbolTarget(symbol sherpa.Symbol) string {
	if strings.TrimSpace(symbol.QualifiedName) != "" {
		return symbol.QualifiedName
	}

	return symbol.DisplayName()
}

func snapshotRecordFromTestFile(identity symbolindex.SymbolIdentity, file string) bool {
	return strings.HasSuffix(identity.Position.File, "_test.go") || strings.HasSuffix(file, "_test.go")
}

func optionalWarning(warning string) []string {
	if strings.TrimSpace(warning) == "" {
		return nil
	}

	return []string{warning}
}

func snapshotFallbackWarning(inspect snapshotstore.InspectResult) string {
	message := strings.TrimSpace(inspect.Message)
	if message == "" {
		message = "snapshot could not be used"
	}

	if len(inspect.StaleReasons) > 0 {
		return fmt.Sprintf("snapshot not used: %s (%s); using live repository analysis", message, strings.Join(inspect.StaleReasons, ", "))
	}

	return fmt.Sprintf("snapshot not used: %s; using live repository analysis", message)
}

func sortReferencesForSnapshot(references []sherpa.Reference) {
	sort.Slice(references, func(i int, j int) bool {
		if references[i].Position.File != references[j].Position.File {
			return references[i].Position.File < references[j].Position.File
		}
		if references[i].Position.Line != references[j].Position.Line {
			return references[i].Position.Line < references[j].Position.Line
		}
		if references[i].Position.Column != references[j].Position.Column {
			return references[i].Position.Column < references[j].Position.Column
		}
		if references[i].Kind != references[j].Kind {
			return references[i].Kind < references[j].Kind
		}

		return references[i].Name < references[j].Name
	})
}

func sortCallersForSnapshot(callers []sherpa.Caller) {
	sort.Slice(callers, func(i int, j int) bool {
		if callers[i].Position.File != callers[j].Position.File {
			return callers[i].Position.File < callers[j].Position.File
		}
		if callers[i].Position.Line != callers[j].Position.Line {
			return callers[i].Position.Line < callers[j].Position.Line
		}

		return callers[i].Name < callers[j].Name
	})
}

func sortCalleesForSnapshot(callees []sherpa.Callee) {
	sort.Slice(callees, func(i int, j int) bool {
		if callees[i].Position.File != callees[j].Position.File {
			return callees[i].Position.File < callees[j].Position.File
		}
		if callees[i].Position.Line != callees[j].Position.Line {
			return callees[i].Position.Line < callees[j].Position.Line
		}

		return callees[i].Name < callees[j].Name
	})
}

func sortImplementersForSnapshot(implementers []impactengine.Implementer) {
	sort.Slice(implementers, func(i int, j int) bool {
		if implementers[i].Name != implementers[j].Name {
			return implementers[i].Name < implementers[j].Name
		}
		if implementers[i].Position.File != implementers[j].Position.File {
			return implementers[i].Position.File < implementers[j].Position.File
		}

		return implementers[i].Position.Line < implementers[j].Position.Line
	})
}

func sortInterfacesForSnapshot(interfaces []impactengine.SatisfiedInterface) {
	sort.Slice(interfaces, func(i int, j int) bool {
		if interfaces[i].Name != interfaces[j].Name {
			return interfaces[i].Name < interfaces[j].Name
		}
		if interfaces[i].Position.File != interfaces[j].Position.File {
			return interfaces[i].Position.File < interfaces[j].Position.File
		}

		return interfaces[i].Position.Line < interfaces[j].Position.Line
	})
}

func fallbackAnalysisMode(_ cliInvocation) string {
	return analysisModeAST
}

func writeHumanWarnings(stderr io.Writer, warnings []string) {
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}

		fmt.Fprintln(stderr, "warning:", warning)
	}
}

func cloneSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return append([]T{}, values...)
}
