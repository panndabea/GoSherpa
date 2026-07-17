package main

import (
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func loadContextSymbolReportForCommand(root string, invocation cliInvocation, target string) (agentcontext.Report, error) {
	report, err := agentcontext.AnalyzeSymbol(root, target, agentcontext.AnalyzeOptions{
		IncludeTests: invocation.IncludeTests,
		BuildTags:    invocation.BuildTags,
		Limits:       invocation.ContextLimits,
	})
	if err != nil {
		return agentcontext.Report{}, err
	}
	if !invocation.UseSnapshot {
		return report, nil
	}

	report, err = applyContextSymbolSnapshotRelationships(root, invocation, target, report)
	if err != nil {
		return agentcontext.Report{}, err
	}

	return report, nil
}

func applyContextSymbolSnapshotRelationships(root string, invocation cliInvocation, target string, report agentcontext.Report) (agentcontext.Report, error) {
	var warnings []string
	relationshipUsed := false
	packageInputsChanged := false

	references, used, fallbackWarnings, err := referenceReportFromSnapshot(root, invocation, target)
	if err != nil {
		return agentcontext.Report{}, err
	}
	if used {
		report.References = references.References
		report.ReferenceAnalysisMode = references.AnalysisMode
		relationshipUsed = true
		packageInputsChanged = true
	} else {
		warnings = append(warnings, fallbackWarnings...)
	}

	callers, used, fallbackWarnings, err := callersFromSnapshot(root, invocation, target)
	if err != nil {
		return agentcontext.Report{}, err
	}
	if used {
		report.Callers = callers.Callers
		report.CallAnalysisMode = callers.AnalysisMode
		relationshipUsed = true
		packageInputsChanged = true
	} else {
		warnings = append(warnings, fallbackWarnings...)
	}

	callees, used, fallbackWarnings, err := calleesFromSnapshot(root, invocation, target)
	if err != nil {
		return agentcontext.Report{}, err
	}
	if used {
		report.Callees = callees.Callees
		report.CallAnalysisMode = mergeContextSnapshotMode(report.CallAnalysisMode, callees.AnalysisMode)
		relationshipUsed = true
	} else {
		warnings = append(warnings, fallbackWarnings...)
	}

	report, interfaceUsed, interfaceWarnings, err := applyContextSymbolInterfaceSnapshot(root, invocation, target, report)
	if err != nil {
		return agentcontext.Report{}, err
	}
	relationshipUsed = relationshipUsed || interfaceUsed
	warnings = append(warnings, interfaceWarnings...)

	if packageInputsChanged {
		report.AffectedPackages = contextPackagesFromRelationshipPositions(report.References, report.Callers)
	}
	if relationshipUsed {
		report.Limitations = appendLimitations(report.Limitations, []string{
			"Context symbol reused a valid relationship snapshot for references, call graph, or interface signals where available; source context, purpose, and test planning may still use live analysis.",
		})
	}

	report.Warnings = uniqueStringsInOrder(append(report.Warnings, warnings...))
	report.Confidence = contextSnapshotConfidence(report)

	return report, nil
}

func applyContextSymbolInterfaceSnapshot(root string, invocation cliInvocation, target string, report agentcontext.Report) (agentcontext.Report, bool, []string, error) {
	switch report.Symbol.Kind {
	case sherpa.SymbolKindInterface:
		implementers, used, warnings, err := implementersFromSnapshot(root, invocation, target)
		if err != nil || !used {
			return report, false, warnings, err
		}
		report.AffectedInterfaces = nonEmptyStrings(snapshotSymbolTarget(report.Symbol))
		report.AffectedImplementations = implementerNames(implementers.Implementers)
		report.InterfaceAnalysisMode = implementers.AnalysisMode
		return report, true, nil, nil
	case sherpa.SymbolKindStruct:
		interfaces, used, warnings, err := interfacesFromSnapshot(root, invocation, target)
		if err != nil || !used {
			return report, false, warnings, err
		}
		report.AffectedInterfaces = interfaceNames(interfaces.Interfaces)
		report.AffectedImplementations = nonEmptyStrings(snapshotSymbolTarget(report.Symbol))
		report.InterfaceAnalysisMode = interfaces.AnalysisMode
		return report, true, nil, nil
	default:
		return report, false, nil, nil
	}
}

func contextPackagesFromRelationshipPositions(references []sherpa.Reference, callers []sherpa.Caller) []string {
	var files []string
	for _, reference := range references {
		files = append(files, reference.Position.File)
	}
	for _, caller := range callers {
		files = append(files, caller.Position.File)
	}

	return impactengine.PackagesForFiles(files)
}

func implementerNames(implementers []impactengine.Implementer) []string {
	names := make([]string, 0, len(implementers))
	for _, implementer := range implementers {
		names = append(names, implementer.Name)
	}

	return uniqueStringsInOrder(names)
}

func interfaceNames(interfaces []impactengine.SatisfiedInterface) []string {
	names := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		names = append(names, iface.Name)
	}

	return uniqueStringsInOrder(names)
}

func nonEmptyStrings(values ...string) []string {
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}

	return result
}

func mergeContextSnapshotMode(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if next == "" {
		return current
	}
	if current == "" {
		return next
	}
	if current == analysisModeSnapshotTypechecked || next == analysisModeSnapshotTypechecked {
		return analysisModeSnapshotTypechecked
	}
	if current == analysisModeSnapshotASTFallback || next == analysisModeSnapshotASTFallback {
		return analysisModeSnapshotASTFallback
	}
	if strings.HasPrefix(current, "snapshot") || strings.HasPrefix(next, "snapshot") {
		return analysisModeSnapshot
	}
	if current == sherpa.CallAnalysisModeTypechecked || next == sherpa.CallAnalysisModeTypechecked {
		return sherpa.CallAnalysisModeTypechecked
	}
	if current == sherpa.CallAnalysisModeASTFallback || next == sherpa.CallAnalysisModeASTFallback {
		return sherpa.CallAnalysisModeASTFallback
	}

	return current
}

func contextSnapshotConfidence(report agentcontext.Report) string {
	if len(report.Warnings) > 0 || len(report.SourceContext.Lines) == 0 {
		return confidenceLow
	}
	for _, mode := range []string{report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode, report.TestAnalysisMode} {
		if strings.Contains(mode, "ast-fallback") {
			return confidenceLow
		}
	}

	return confidenceMedium
}
