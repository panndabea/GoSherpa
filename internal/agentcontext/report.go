package agentcontext

import (
	"strings"

	explainengine "github.com/supertabaluga/gosherpa/internal/explain"
	impactengine "github.com/supertabaluga/gosherpa/internal/impact"
	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

const (
	AnalysisModeAST            = "ast"
	AnalysisModeTypecheckedAST = "typechecked+ast"
	AnalysisModeDiff           = "git-diff+ast"

	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

type AnalyzeOptions struct {
	IncludeTests bool `json:"includeTests"`
	BuildTags    []string
	SourceRadius int          `json:"sourceRadius"`
	Limits       LimitOptions `json:"limits"`
}

type Report struct {
	Target                  string                         `json:"target"`
	Identity                Identity                       `json:"identity"`
	Symbol                  sherpa.Symbol                  `json:"symbol"`
	SourceContext           sherpa.SourceContext           `json:"sourceContext"`
	Purpose                 string                         `json:"purpose"`
	Risk                    explainengine.RiskSummary      `json:"risk"`
	ArchitectureRole        explainengine.ArchitectureRole `json:"architectureRole"`
	References              []sherpa.Reference             `json:"references"`
	Callers                 []sherpa.Caller                `json:"callers"`
	Callees                 []sherpa.Callee                `json:"callees"`
	CallAnalysisMode        string                         `json:"callAnalysisMode"`
	AffectedPackages        []string                       `json:"affectedPackages"`
	AffectedInterfaces      []string                       `json:"affectedInterfaces"`
	AffectedImplementations []string                       `json:"affectedImplementations"`
	InterfaceAnalysisMode   string                         `json:"interfaceAnalysisMode,omitempty"`
	RelatedTests            []sherpa.RelatedTest           `json:"relatedTests"`
	TestCommands            []string                       `json:"testCommands"`
	TestPlan                sherpa.TestPlan                `json:"testPlan"`
	ReadingOrder            []explainengine.ReadingStep    `json:"readingOrder"`
	AnalysisMode            string                         `json:"analysisMode"`
	Confidence              string                         `json:"confidence"`
	Limits                  *LimitOptions                  `json:"limits,omitempty"`
	Truncated               *Truncation                    `json:"truncated,omitempty"`
	Limitations             []string                       `json:"limitations"`
	Warnings                []string                       `json:"-"`
}

type Identity struct {
	Target        string            `json:"target"`
	Package       string            `json:"package"`
	PackageName   string            `json:"packageName,omitempty"`
	Symbol        string            `json:"symbol"`
	Kind          sherpa.SymbolKind `json:"kind"`
	QualifiedName string            `json:"qualifiedName,omitempty"`
	Signature     string            `json:"signature,omitempty"`
	Definition    sherpa.Position   `json:"definition"`
}

func AnalyzeSymbol(root string, target string, options AnalyzeOptions) (Report, error) {
	explainReport, err := explainengine.AnalyzeWithOptions(root, target, explainengine.AnalyzeOptions{
		IncludeTests: options.IncludeTests,
		BuildTags:    options.BuildTags,
	})
	if err != nil {
		return Report{}, err
	}

	limits := normalizeLimits(options.SourceRadius, options.Limits)
	radius := sourceRadiusOrDefault(limits, sherpa.DefaultSourceContextRadius)

	warnings := append([]string{}, explainReport.Warnings...)
	analysisMode := AnalysisModeAST
	symbol := explainReport.Symbol
	semanticSnapshot, semanticOK := loadContextSemanticSnapshot(root, options.BuildTags)
	warnings = append(warnings, semanticSnapshot.warnings...)
	if semanticOK {
		semanticSymbol, found, err := semanticSnapshot.symbol(root, target)
		if err != nil {
			return Report{}, err
		}
		if found {
			symbol = semanticSymbol
			analysisMode = AnalysisModeTypecheckedAST
		}
	}

	sourceContext, err := sherpa.ReadSourceContext(root, symbol.Position, radius)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	report := Report{
		Target:                  explainReport.Target,
		Identity:                identityFromSymbol(explainReport.Target, symbol),
		Symbol:                  symbol,
		SourceContext:           sourceContext,
		Purpose:                 explainReport.Purpose,
		Risk:                    explainReport.Risk,
		ArchitectureRole:        explainReport.ArchitectureRole,
		References:              explainReport.References,
		Callers:                 explainReport.Callers,
		Callees:                 explainReport.Callees,
		CallAnalysisMode:        explainReport.CallAnalysisMode,
		AffectedPackages:        explainReport.AffectedPackages,
		AffectedInterfaces:      explainReport.AffectedInterfaces,
		AffectedImplementations: explainReport.AffectedImplementations,
		InterfaceAnalysisMode:   explainReport.InterfaceAnalysisMode,
		RelatedTests:            explainReport.RelatedTests,
		TestCommands:            explainReport.TestCommands,
		TestPlan:                explainReport.TestPlan,
		ReadingOrder:            explainReport.ReadingOrder,
		AnalysisMode:            analysisMode,
		Limits:                  reportLimits(limits),
		Warnings:                warnings,
	}
	report.Limitations = limitations(options.IncludeTests, report.AnalysisMode, report.CallAnalysisMode)
	report.Confidence = confidence(report)
	report = applySymbolLimits(report, limits)

	return normalizeReport(report), nil
}

func applySymbolLimits(report Report, limits LimitOptions) Report {
	var truncation Truncation
	originalReadingOrderCount := len(report.ReadingOrder)

	report.References, truncation.References = limitSlice(report.References, limits.MaxReferences)
	report.Callers, truncation.Callers = limitSlice(report.Callers, limits.MaxReferences)
	report.Callees, truncation.Callees = limitSlice(report.Callees, limits.MaxReferences)
	report.RelatedTests, truncation.RelatedTests = limitSlice(report.RelatedTests, limits.MaxTests)
	report.ReadingOrder = symbolReadingOrder(report)
	if originalReadingOrderCount > len(report.ReadingOrder) {
		truncation.ReadingOrder = originalReadingOrderCount - len(report.ReadingOrder)
	}

	report.Truncated = reportTruncation(truncation)
	report = applySymbolByteLimit(report, limits.MaxBytes)

	return report
}

func symbolReadingOrder(report Report) []explainengine.ReadingStep {
	steps := []explainengine.ReadingStep{
		{
			Title:    "Definition",
			Reason:   "Start with the symbol declaration and nearby implementation.",
			Position: report.Symbol.Position,
		},
	}

	for _, callee := range firstCallees(report.Callees, 3) {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Callee: " + callee.Name,
			Reason:   "Understand direct work delegated by this symbol.",
			Position: callee.Position,
		})
	}

	for _, caller := range firstCallers(report.Callers, 3) {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Caller: " + caller.Name,
			Reason:   "See how callers depend on this symbol.",
			Position: caller.Position,
		})
	}

	for _, test := range firstRelatedTests(report.RelatedTests, 3) {
		steps = append(steps, explainengine.ReadingStep{
			Title:    "Test: " + test.Name,
			Reason:   "Check expected behavior and regression coverage.",
			Position: test.Position,
		})
	}

	return steps
}

func firstCallees(values []sherpa.Callee, limit int) []sherpa.Callee {
	if len(values) <= limit {
		return values
	}

	return values[:limit]
}

func firstCallers(values []sherpa.Caller, limit int) []sherpa.Caller {
	if len(values) <= limit {
		return values
	}

	return values[:limit]
}

func firstRelatedTests(values []sherpa.RelatedTest, limit int) []sherpa.RelatedTest {
	if len(values) <= limit {
		return values
	}

	return values[:limit]
}

func identityFromSymbol(target string, symbol sherpa.Symbol) Identity {
	return Identity{
		Target:        target,
		Package:       symbol.Package,
		PackageName:   symbol.PackageName,
		Symbol:        symbol.DisplayName(),
		Kind:          symbol.Kind,
		QualifiedName: symbol.QualifiedName,
		Signature:     symbol.Signature,
		Definition:    symbol.Position,
	}
}

func limitations(includeTestCallers bool, analysisMode string, callAnalysisMode string) []string {
	values := []string{
		symbolContextAnalysisLimitation(analysisMode),
		callAnalysisLimitation(callAnalysisMode),
		"Dynamic dispatch, reflection, and function values are not resolved.",
		"Call graph results are repository-local and may miss some imported-package receiver calls.",
		"Test discovery uses same-package tests and syntactic direct-reference matching.",
	}

	if !includeTestCallers {
		values = append(values, "Test callers are included only when --tests is used.")
	}

	return values
}

func symbolContextAnalysisLimitation(analysisMode string) string {
	switch analysisMode {
	case AnalysisModeTypecheckedAST:
		return "Symbol context used typechecked package loading for symbol identity, with syntax/local impact and test signals."
	default:
		return "Symbol context used AST fallback for symbol identity because typechecked loading was unavailable or did not include the target symbol."
	}
}

func callAnalysisLimitation(callAnalysisMode string) string {
	switch callAnalysisMode {
	case sherpa.CallAnalysisModeTypechecked:
		return "Call analysis used typechecked package loading where available."
	case sherpa.CallAnalysisModeASTFallback:
		return "Call analysis used AST fallback because typechecked loading was unavailable."
	default:
		return "Analysis uses syntax plus local type information, not full module loading."
	}
}

func confidence(report Report) string {
	if len(report.Warnings) > 0 || len(report.SourceContext.Lines) == 0 {
		return ConfidenceLow
	}
	if report.InterfaceAnalysisMode == impactengine.InterfaceAnalysisModeASTFallback {
		return ConfidenceLow
	}

	return ConfidenceMedium
}

func normalizeReport(report Report) Report {
	report.References = nonNilSlice(report.References)
	report.Callers = nonNilSlice(report.Callers)
	report.Callees = nonNilSlice(report.Callees)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.RelatedTests = nonNilSlice(report.RelatedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.SourceContext.Lines = nonNilSlice(report.SourceContext.Lines)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = uniqueStrings(report.Warnings)

	return report
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}
