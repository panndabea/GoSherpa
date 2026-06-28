package agentcontext

import (
	explainengine "github.com/supertabaluga/gosherpa/internal/explain"
	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

const (
	AnalysisModeAST  = "ast"
	AnalysisModeDiff = "git-diff+ast"

	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

type AnalyzeOptions struct {
	IncludeTests bool `json:"includeTests"`
	SourceRadius int  `json:"sourceRadius"`
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
	AffectedPackages        []string                       `json:"affectedPackages"`
	AffectedInterfaces      []string                       `json:"affectedInterfaces"`
	AffectedImplementations []string                       `json:"affectedImplementations"`
	RelatedTests            []sherpa.RelatedTest           `json:"relatedTests"`
	TestCommands            []string                       `json:"testCommands"`
	TestPlan                sherpa.TestPlan                `json:"testPlan"`
	ReadingOrder            []explainengine.ReadingStep    `json:"readingOrder"`
	AnalysisMode            string                         `json:"analysisMode"`
	Confidence              string                         `json:"confidence"`
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
	})
	if err != nil {
		return Report{}, err
	}

	radius := options.SourceRadius
	if radius == 0 {
		radius = sherpa.DefaultSourceContextRadius
	}

	warnings := append([]string{}, explainReport.Warnings...)
	sourceContext, err := sherpa.ReadSourceContext(root, explainReport.Symbol.Position, radius)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	report := Report{
		Target:                  explainReport.Target,
		Identity:                identityFromSymbol(explainReport.Target, explainReport.Symbol),
		Symbol:                  explainReport.Symbol,
		SourceContext:           sourceContext,
		Purpose:                 explainReport.Purpose,
		Risk:                    explainReport.Risk,
		ArchitectureRole:        explainReport.ArchitectureRole,
		References:              explainReport.References,
		Callers:                 explainReport.Callers,
		Callees:                 explainReport.Callees,
		AffectedPackages:        explainReport.AffectedPackages,
		AffectedInterfaces:      explainReport.AffectedInterfaces,
		AffectedImplementations: explainReport.AffectedImplementations,
		RelatedTests:            explainReport.RelatedTests,
		TestCommands:            explainReport.TestCommands,
		TestPlan:                explainReport.TestPlan,
		ReadingOrder:            explainReport.ReadingOrder,
		AnalysisMode:            AnalysisModeAST,
		Warnings:                warnings,
	}
	report.Limitations = limitations(options.IncludeTests)
	report.Confidence = confidence(report)

	return normalizeReport(report), nil
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

func limitations(includeTestCallers bool) []string {
	values := []string{
		"Analysis uses syntax plus local type information, not full module loading.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
		"Call graph results are repository-local and may miss some imported-package receiver calls.",
		"Test discovery uses same-package tests and syntactic direct-reference matching.",
	}

	if !includeTestCallers {
		values = append(values, "Test callers are included only when --tests is used.")
	}

	return values
}

func confidence(report Report) string {
	if len(report.Warnings) > 0 || len(report.SourceContext.Lines) == 0 {
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
	report.RelatedTests = nonNilSlice(report.RelatedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.SourceContext.Lines = nonNilSlice(report.SourceContext.Lines)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}
