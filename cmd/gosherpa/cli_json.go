package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type jsonResponse[T any] struct {
	SchemaVersion int      `json:"schemaVersion"`
	Command       string   `json:"command"`
	Target        string   `json:"target"`
	Root          string   `json:"root"`
	ModulePath    string   `json:"modulePath"`
	Warnings      []string `json:"warnings"`
	Data          T        `json:"data"`
}

type jsonErrorResponse struct {
	SchemaVersion int           `json:"schemaVersion"`
	Command       string        `json:"command"`
	Target        string        `json:"target"`
	Root          string        `json:"root"`
	ModulePath    string        `json:"modulePath"`
	Warnings      []string      `json:"warnings"`
	Error         jsonErrorData `json:"error"`
}

type jsonErrorData struct {
	Code       string                   `json:"code"`
	Message    string                   `json:"message"`
	Kind       string                   `json:"kind,omitempty"`
	Target     string                   `json:"target,omitempty"`
	Candidates []sherpa.TargetCandidate `json:"candidates,omitempty"`
}

type referencesJSONData struct {
	AnalysisMode string             `json:"analysisMode"`
	Confidence   string             `json:"confidence"`
	Limitations  []string           `json:"limitations"`
	References   []sherpa.Reference `json:"references"`
}

type symbolJSONData struct {
	Symbol sherpa.Symbol `json:"symbol"`
}

type symbolsJSONData struct {
	Symbols []sherpa.Symbol `json:"symbols"`
}

type searchJSONData struct {
	Terms   []string                    `json:"terms"`
	Results []sherpa.SymbolSearchResult `json:"results"`
}

type impactJSONData struct {
	AnalysisMode          string                     `json:"analysisMode"`
	Confidence            string                     `json:"confidence"`
	Limitations           []string                   `json:"limitations"`
	Kind                  sherpa.ImpactKind          `json:"kind"`
	References            []sherpa.Reference         `json:"references"`
	ReferenceAnalysisMode string                     `json:"referenceAnalysisMode,omitempty"`
	Callers               []sherpa.Caller            `json:"callers"`
	CallAnalysisMode      string                     `json:"callAnalysisMode,omitempty"`
	Dependencies          sherpa.PackageDependencies `json:"dependencies"`
	Packages              []string                   `json:"packages"`
	RelatedTests          []sherpa.RelatedTest       `json:"relatedTests"`
	TestCommands          []string                   `json:"testCommands"`
	TestPlan              sherpa.TestPlan            `json:"testPlan"`
}

type impactDiffJSONData struct {
	AnalysisMode            string                     `json:"analysisMode"`
	Confidence              string                     `json:"confidence"`
	Limitations             []string                   `json:"limitations"`
	ChangedFiles            []string                   `json:"changedFiles"`
	ChangedPackages         []string                   `json:"changedPackages"`
	AffectedPackages        []string                   `json:"affectedPackages"`
	AffectedSymbols         []string                   `json:"affectedSymbols"`
	ReferenceAnalysisMode   string                     `json:"referenceAnalysisMode,omitempty"`
	CallAnalysisMode        string                     `json:"callAnalysisMode,omitempty"`
	AffectedInterfaces      []string                   `json:"affectedInterfaces"`
	AffectedImplementations []string                   `json:"affectedImplementations"`
	InterfaceAnalysisMode   string                     `json:"interfaceAnalysisMode,omitempty"`
	AffectedTests           []impactengine.RelatedTest `json:"affectedTests"`
	TestCommands            []string                   `json:"testCommands"`
	TestPlan                sherpa.TestPlan            `json:"testPlan"`
}

type testsJSONData struct {
	AnalysisMode string                `json:"analysisMode"`
	Confidence   string                `json:"confidence"`
	Limitations  []string              `json:"limitations"`
	Kind         sherpa.TestTargetKind `json:"kind"`
	Scope        sherpa.TestScope      `json:"scope"`
	Tests        []sherpa.RelatedTest  `json:"tests"`
	Commands     []string              `json:"commands"`
	TestPlan     sherpa.TestPlan       `json:"testPlan"`
}

type testsAffectedJSONData struct {
	AnalysisMode          string                     `json:"analysisMode"`
	Confidence            string                     `json:"confidence"`
	Limitations           []string                   `json:"limitations"`
	ReferenceAnalysisMode string                     `json:"referenceAnalysisMode,omitempty"`
	CallAnalysisMode      string                     `json:"callAnalysisMode,omitempty"`
	InterfaceAnalysisMode string                     `json:"interfaceAnalysisMode,omitempty"`
	AffectedTests         []impactengine.RelatedTest `json:"affectedTests"`
	Commands              []string                   `json:"commands"`
	TestPlan              sherpa.TestPlan            `json:"testPlan"`
}

type dependenciesJSONData struct {
	Package string   `json:"package"`
	Imports []string `json:"imports"`
	UsedBy  []string `json:"usedBy"`
}

type repositoryDependenciesJSONData struct {
	Packages []sherpa.PackageDependencySummary `json:"packages"`
}

type packagesJSONData struct {
	Packages []sherpa.PackageSummary `json:"packages"`
}

type implementersJSONData struct {
	AnalysisMode string                     `json:"analysisMode"`
	Confidence   string                     `json:"confidence"`
	Limitations  []string                   `json:"limitations"`
	Implementers []impactengine.Implementer `json:"implementers"`
}

type interfacesJSONData struct {
	AnalysisMode string                            `json:"analysisMode"`
	Confidence   string                            `json:"confidence"`
	Limitations  []string                          `json:"limitations"`
	Interfaces   []impactengine.SatisfiedInterface `json:"interfaces"`
}

type callersJSONData struct {
	AnalysisMode string          `json:"analysisMode"`
	Confidence   string          `json:"confidence"`
	Limitations  []string        `json:"limitations"`
	Callers      []sherpa.Caller `json:"callers"`
}

type calleesJSONData struct {
	AnalysisMode string          `json:"analysisMode"`
	Confidence   string          `json:"confidence"`
	Limitations  []string        `json:"limitations"`
	Callees      []sherpa.Callee `json:"callees"`
}

type callPathsJSONData struct {
	AnalysisMode string            `json:"analysisMode"`
	Confidence   string            `json:"confidence"`
	Limitations  []string          `json:"limitations"`
	From         string            `json:"from"`
	To           string            `json:"to"`
	Paths        []sherpa.CallPath `json:"paths"`
}

type entrypointsJSONData struct {
	AnalysisMode string              `json:"analysisMode"`
	Confidence   string              `json:"confidence"`
	Limitations  []string            `json:"limitations"`
	EntryPoints  []sherpa.EntryPoint `json:"entrypoints"`
}

type explainJSONData struct {
	Target                  string                         `json:"target"`
	AnalysisMode            string                         `json:"analysisMode"`
	Confidence              string                         `json:"confidence"`
	Limitations             []string                       `json:"limitations"`
	Symbol                  sherpa.Symbol                  `json:"symbol"`
	SymbolAnalysisMode      string                         `json:"symbolAnalysisMode,omitempty"`
	Purpose                 string                         `json:"purpose"`
	Risk                    explainengine.RiskSummary      `json:"risk"`
	ArchitectureRole        explainengine.ArchitectureRole `json:"architectureRole"`
	References              []sherpa.Reference             `json:"references"`
	ReferenceAnalysisMode   string                         `json:"referenceAnalysisMode,omitempty"`
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
}

func writeJSON(stdout io.Writer, stderr io.Writer, value any) int {
	if err := encodeJSON(stdout, value); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitFailure
	}

	return exitSuccess
}

func encodeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(value)
}

func newJSONResponse[T any](root string, command string, target string, warnings []string, data T) jsonResponse[T] {
	modulePath, err := sherpa.ModulePath(root)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	return jsonResponse[T]{
		SchemaVersion: jsonSchemaVersion,
		Command:       command,
		Target:        target,
		Root:          root,
		ModulePath:    modulePath,
		Warnings:      nonNilSlice(warnings),
		Data:          data,
	}
}

func writeCommandError(jsonOutput bool, root string, command string, target string, stderr io.Writer, err error) int {
	if jsonOutput && writeTypedJSONError(stderr, root, command, target, err) {
		return exitFailure
	}

	fmt.Fprintln(stderr, "error:", err)
	return exitFailure
}

func writeTypedJSONError(stderr io.Writer, root string, command string, target string, err error) bool {
	var ambiguous *sherpa.AmbiguousTargetError
	if !errors.As(err, &ambiguous) {
		return false
	}

	response := newJSONErrorResponse(root, command, target, jsonErrorData{
		Code:       "ambiguous_target",
		Message:    err.Error(),
		Kind:       ambiguous.Kind,
		Target:     ambiguous.Target,
		Candidates: nonNilSlice(ambiguous.Candidates),
	})

	if encodeErr := encodeJSON(stderr, response); encodeErr != nil {
		fmt.Fprintln(stderr, "error:", encodeErr)
	}

	return true
}

func newJSONErrorResponse(root string, command string, target string, errorData jsonErrorData) jsonErrorResponse {
	var warnings []string
	modulePath, err := sherpa.ModulePath(root)
	if err != nil {
		warnings = append(warnings, err.Error())
	}

	return jsonErrorResponse{
		SchemaVersion: jsonSchemaVersion,
		Command:       command,
		Target:        target,
		Root:          root,
		ModulePath:    modulePath,
		Warnings:      nonNilSlice(warnings),
		Error:         errorData,
	}
}

func analyzeJSONResult(report analyzeReport) analyzeReport {
	report = normalizeAnalyzeReport(report)
	report.Limitations = nonNilSlice(report.Limitations)
	report.BuildTags = nonNilSlice(report.BuildTags)
	report.Packages = nonNilSlice(report.Packages)
	report.ImportantSymbols = nonNilSlice(report.ImportantSymbols)
	report.EntryPoints = nonNilSlice(report.EntryPoints)
	report.Hotspots = nonNilSlice(report.Hotspots)
	report.Testing.TestPackages = nonNilSlice(report.Testing.TestPackages)
	report.Testing.SuggestedCommands = nonNilSlice(report.Testing.SuggestedCommands)
	report.Readiness.Suggestions = nonNilSlice(report.Readiness.Suggestions)
	report.Suggestions = nonNilSlice(report.Suggestions)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func impactJSONResult(result sherpa.ImpactResult) sherpa.ImpactResult {
	result.References = nonNilSlice(result.References)
	result.ReferenceAnalysisMode = strings.TrimSpace(result.ReferenceAnalysisMode)
	result.Callers = nonNilSlice(result.Callers)
	result.CallAnalysisMode = strings.TrimSpace(result.CallAnalysisMode)
	result.Dependencies.Imports = nonNilSlice(result.Dependencies.Imports)
	result.Dependencies.UsedBy = nonNilSlice(result.Dependencies.UsedBy)
	result.Packages = nonNilSlice(result.Packages)
	result.RelatedTests = nonNilSlice(result.RelatedTests)
	result.TestCommands = nonNilSlice(result.TestCommands)
	result.TestPlan = sherpa.NormalizeTestPlan(result.TestPlan)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func impactJSONDataFromResult(result sherpa.ImpactResult) impactJSONData {
	analysisMode := impactResultAnalysisMode(result)

	return impactJSONData{
		AnalysisMode:          analysisMode,
		Confidence:            jsonConfidence(result.Warnings, analysisMode, result.ReferenceAnalysisMode, result.CallAnalysisMode),
		Limitations:           impactBundleLimitations(analysisMode, result.ReferenceAnalysisMode, result.CallAnalysisMode),
		Kind:                  result.Kind,
		References:            result.References,
		ReferenceAnalysisMode: result.ReferenceAnalysisMode,
		Callers:               result.Callers,
		CallAnalysisMode:      result.CallAnalysisMode,
		Dependencies:          result.Dependencies,
		Packages:              result.Packages,
		RelatedTests:          result.RelatedTests,
		TestCommands:          result.TestCommands,
		TestPlan:              result.TestPlan,
	}
}

func impactResultAnalysisMode(result sherpa.ImpactResult) string {
	return bundleAnalysisMode(result.ReferenceAnalysisMode, result.CallAnalysisMode)
}

func impactDiffJSONResult(report impactengine.ImpactReport) impactengine.ImpactReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.ReferenceAnalysisMode = strings.TrimSpace(report.ReferenceAnalysisMode)
	report.CallAnalysisMode = strings.TrimSpace(report.CallAnalysisMode)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func impactDiffJSONDataFromReport(report impactengine.ImpactReport, analysisMode string) impactDiffJSONData {
	analysisMode = impactReportAnalysisMode(report, analysisMode)

	return impactDiffJSONData{
		AnalysisMode:            analysisMode,
		Confidence:              jsonConfidence(report.Warnings, analysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode),
		Limitations:             impactBundleLimitations(analysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode),
		ChangedFiles:            report.ChangedFiles,
		ChangedPackages:         report.ChangedPackages,
		AffectedPackages:        report.AffectedPackages,
		AffectedSymbols:         report.AffectedSymbols,
		ReferenceAnalysisMode:   report.ReferenceAnalysisMode,
		CallAnalysisMode:        report.CallAnalysisMode,
		AffectedInterfaces:      report.AffectedInterfaces,
		AffectedImplementations: report.AffectedImplementations,
		InterfaceAnalysisMode:   report.InterfaceAnalysisMode,
		AffectedTests:           report.AffectedTests,
		TestCommands:            report.TestCommands,
		TestPlan:                report.TestPlan,
	}
}

func impactReportAnalysisMode(report impactengine.ImpactReport, fallback string) string {
	if fallback == analysisModeDiff {
		if report.ReferenceAnalysisMode == sherpa.ReferenceAnalysisModeTypechecked ||
			report.CallAnalysisMode == sherpa.CallAnalysisModeTypechecked ||
			report.InterfaceAnalysisMode == impactengine.InterfaceAnalysisModeTypechecked {
			return analysisModeDiffTypechecked
		}

		return analysisModeDiff
	}

	return bundleAnalysisMode(report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode)
}

func testsJSONResult(result sherpa.TestsResult) sherpa.TestsResult {
	result.Tests = nonNilSlice(result.Tests)
	result.Commands = nonNilSlice(result.Commands)
	result.TestPlan = sherpa.NormalizeTestPlan(result.TestPlan)

	return result
}

func testsJSONDataFromResult(result sherpa.TestsResult) testsJSONData {
	return testsJSONData{
		AnalysisMode: analysisModeAST,
		Confidence:   jsonConfidence(nil, analysisModeAST),
		Limitations:  testLimitations(analysisModeAST),
		Kind:         result.Kind,
		Scope:        result.Scope,
		Tests:        result.Tests,
		Commands:     result.Commands,
		TestPlan:     result.TestPlan,
	}
}

func testsAffectedJSONDataFromReport(report impactengine.ImpactReport) testsAffectedJSONData {
	analysisMode := impactReportAnalysisMode(report, analysisModeDiff)

	return testsAffectedJSONData{
		AnalysisMode:          analysisMode,
		Confidence:            jsonConfidence(report.Warnings, analysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode),
		Limitations:           testLimitations(analysisMode),
		ReferenceAnalysisMode: report.ReferenceAnalysisMode,
		CallAnalysisMode:      report.CallAnalysisMode,
		InterfaceAnalysisMode: report.InterfaceAnalysisMode,
		AffectedTests:         report.AffectedTests,
		Commands:              report.TestCommands,
		TestPlan:              report.TestPlan,
	}
}

func dependenciesJSONResult(result sherpa.PackageDependencies) sherpa.PackageDependencies {
	result.Imports = nonNilSlice(result.Imports)
	result.UsedBy = nonNilSlice(result.UsedBy)

	return result
}

func dependenciesJSONDataFromResult(result sherpa.PackageDependencies) dependenciesJSONData {
	return dependenciesJSONData{
		Package: result.Package,
		Imports: result.Imports,
		UsedBy:  result.UsedBy,
	}
}

func repositoryDependenciesJSONResult(result sherpa.RepositoryDependencies) sherpa.RepositoryDependencies {
	result.Packages = nonNilSlice(result.Packages)
	for i, pkg := range result.Packages {
		pkg.Imports = nonNilSlice(pkg.Imports)
		pkg.LocalImports = nonNilSlice(pkg.LocalImports)
		pkg.ExternalImports = nonNilSlice(pkg.ExternalImports)
		pkg.UsedBy = nonNilSlice(pkg.UsedBy)
		result.Packages[i] = pkg
	}

	return result
}

func repositoryDependenciesJSONDataFromResult(result sherpa.RepositoryDependencies) repositoryDependenciesJSONData {
	return repositoryDependenciesJSONData{
		Packages: result.Packages,
	}
}

func packagesJSONResult(result []sherpa.PackageSummary) []sherpa.PackageSummary {
	return nonNilSlice(result)
}

func packagesJSONDataFromResult(result []sherpa.PackageSummary) packagesJSONData {
	return packagesJSONData{
		Packages: result,
	}
}

func implementersJSONResult(result impactengine.ImplementersResult) impactengine.ImplementersResult {
	result.Implementers = nonNilSlice(result.Implementers)
	result.Warnings = nonNilSlice(result.Warnings)
	if strings.TrimSpace(result.AnalysisMode) == "" {
		result.AnalysisMode = impactengine.InterfaceAnalysisModeASTFallback
	}

	return result
}

func implementersJSONDataFromResult(result impactengine.ImplementersResult) implementersJSONData {
	return implementersJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  interfaceLimitations(result.AnalysisMode),
		Implementers: result.Implementers,
	}
}

func interfacesJSONResult(result impactengine.InterfacesResult) impactengine.InterfacesResult {
	result.Interfaces = nonNilSlice(result.Interfaces)
	result.Warnings = nonNilSlice(result.Warnings)
	if strings.TrimSpace(result.AnalysisMode) == "" {
		result.AnalysisMode = impactengine.InterfaceAnalysisModeASTFallback
	}

	return result
}

func interfacesJSONDataFromResult(result impactengine.InterfacesResult) interfacesJSONData {
	return interfacesJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  interfaceLimitations(result.AnalysisMode),
		Interfaces:   result.Interfaces,
	}
}

func callersJSONResult(result sherpa.CallersResult) sherpa.CallersResult {
	result.Callers = nonNilSlice(result.Callers)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func callersJSONDataFromResult(result sherpa.CallersResult) callersJSONData {
	return callersJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  callLimitations(result.AnalysisMode),
		Callers:      result.Callers,
	}
}

func calleesJSONResult(result sherpa.CalleesResult) sherpa.CalleesResult {
	result.Callees = nonNilSlice(result.Callees)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func calleesJSONDataFromResult(result sherpa.CalleesResult) calleesJSONData {
	return calleesJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  callLimitations(result.AnalysisMode),
		Callees:      result.Callees,
	}
}

func callPathsJSONResult(result sherpa.CallPathsResult) sherpa.CallPathsResult {
	result.Paths = nonNilSlice(result.Paths)
	for i := range result.Paths {
		result.Paths[i].Steps = nonNilSlice(result.Paths[i].Steps)
	}

	return result
}

func callPathJSONTarget(result sherpa.CallPathsResult) string {
	return result.From + " -> " + result.To
}

func callPathsJSONDataFromResult(result sherpa.CallPathsResult) callPathsJSONData {
	return callPathsJSONData{
		AnalysisMode: analysisModeAST,
		Confidence:   jsonConfidence(nil, analysisModeAST),
		Limitations:  callPathLimitations(),
		From:         result.From,
		To:           result.To,
		Paths:        result.Paths,
	}
}

func entrypointsJSONResult(result sherpa.EntryPointsResult) sherpa.EntryPointsResult {
	result.EntryPoints = nonNilSlice(result.EntryPoints)
	result.Warnings = nonNilSlice(result.Warnings)

	return result
}

func entrypointsJSONDataFromResult(result sherpa.EntryPointsResult) entrypointsJSONData {
	return entrypointsJSONData{
		AnalysisMode: result.AnalysisMode,
		Confidence:   jsonConfidence(result.Warnings, result.AnalysisMode),
		Limitations:  entrypointLimitations(result.AnalysisMode),
		EntryPoints:  result.EntryPoints,
	}
}

func contextSymbolJSONResult(report agentcontext.Report) agentcontext.Report {
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
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func contextFileJSONResult(report agentcontext.FileReport) agentcontext.FileReport {
	report.Symbols = nonNilSlice(report.Symbols)
	report.SourceContexts = nonNilSlice(report.SourceContexts)
	for i := range report.SourceContexts {
		report.SourceContexts[i].Lines = nonNilSlice(report.SourceContexts[i].Lines)
	}
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func contextPackageJSONResult(report agentcontext.PackageReport) agentcontext.PackageReport {
	report.Files = nonNilSlice(report.Files)
	report.Symbols = nonNilSlice(report.Symbols)
	report.SourceContexts = nonNilSlice(report.SourceContexts)
	for i := range report.SourceContexts {
		report.SourceContexts[i].Lines = nonNilSlice(report.SourceContexts[i].Lines)
	}
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func contextDiffJSONResult(report agentcontext.DiffReport) agentcontext.DiffReport {
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.ReferenceAnalysisMode = strings.TrimSpace(report.ReferenceAnalysisMode)
	report.CallAnalysisMode = strings.TrimSpace(report.CallAnalysisMode)
	report.AffectedInterfaces = nonNilSlice(report.AffectedInterfaces)
	report.AffectedImplementations = nonNilSlice(report.AffectedImplementations)
	report.InterfaceAnalysisMode = strings.TrimSpace(report.InterfaceAnalysisMode)
	report.AffectedTests = nonNilSlice(report.AffectedTests)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.Risk.Reasons = nonNilSlice(report.Risk.Reasons)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func explainJSONResult(report explainengine.Report) explainengine.Report {
	report.SymbolAnalysisMode = strings.TrimSpace(report.SymbolAnalysisMode)
	report.References = nonNilSlice(report.References)
	report.ReferenceAnalysisMode = strings.TrimSpace(report.ReferenceAnalysisMode)
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
	report.Warnings = nonNilSlice(report.Warnings)

	return report
}

func explainJSONDataFromReport(report explainengine.Report) explainJSONData {
	analysisMode := explainAnalysisMode(report)

	return explainJSONData{
		Target:                  report.Target,
		AnalysisMode:            analysisMode,
		Confidence:              jsonConfidence(report.Warnings, analysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode),
		Limitations:             explainLimitations(report.ReferenceAnalysisMode, report.CallAnalysisMode),
		Symbol:                  report.Symbol,
		SymbolAnalysisMode:      report.SymbolAnalysisMode,
		Purpose:                 report.Purpose,
		Risk:                    report.Risk,
		ArchitectureRole:        report.ArchitectureRole,
		References:              report.References,
		ReferenceAnalysisMode:   report.ReferenceAnalysisMode,
		Callers:                 report.Callers,
		Callees:                 report.Callees,
		CallAnalysisMode:        report.CallAnalysisMode,
		AffectedPackages:        report.AffectedPackages,
		AffectedInterfaces:      report.AffectedInterfaces,
		AffectedImplementations: report.AffectedImplementations,
		InterfaceAnalysisMode:   report.InterfaceAnalysisMode,
		RelatedTests:            report.RelatedTests,
		TestCommands:            report.TestCommands,
		TestPlan:                report.TestPlan,
		ReadingOrder:            report.ReadingOrder,
	}
}

func explainAnalysisMode(report explainengine.Report) string {
	return bundleAnalysisMode(report.SymbolAnalysisMode, report.ReferenceAnalysisMode, report.CallAnalysisMode, report.InterfaceAnalysisMode)
}

func bundleAnalysisMode(analysisModes ...string) string {
	for _, mode := range analysisModes {
		if mode == agentcontext.AnalysisModeTypecheckedAST ||
			mode == agentcontext.AnalysisModeDiffTypechecked ||
			mode == explainengine.SymbolAnalysisModeTypecheckedAST ||
			mode == sherpa.ReferenceAnalysisModeTypechecked ||
			mode == sherpa.CallAnalysisModeTypechecked ||
			mode == impactengine.InterfaceAnalysisModeTypechecked {
			return agentcontext.AnalysisModeTypecheckedAST
		}
	}

	return analysisModeAST
}

func jsonConfidence(warnings []string, analysisModes ...string) string {
	if len(warnings) > 0 {
		return confidenceLow
	}

	for _, mode := range analysisModes {
		if mode == sherpa.CallAnalysisModeASTFallback ||
			mode == sherpa.ReferenceAnalysisModeASTFallback ||
			mode == impactengine.InterfaceAnalysisModeASTFallback {
			return confidenceLow
		}
	}

	return confidenceMedium
}

func referenceLimitations(analysisMode string) []string {
	return []string{
		referenceAnalysisLimitation(analysisMode),
		"References are repository-local and may not include generated or build-tagged code outside the loaded package set.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
	}
}

func referenceAnalysisLimitation(analysisMode string) string {
	switch analysisMode {
	case sherpa.ReferenceAnalysisModeTypechecked:
		return "Reference analysis used typechecked package loading where available."
	case sherpa.ReferenceAnalysisModeASTFallback:
		return "Reference analysis used AST fallback because typechecked loading was unavailable."
	default:
		return "Reference analysis used syntax plus local type information."
	}
}

func callLimitations(analysisMode string) []string {
	return []string{
		callAnalysisLimitation(analysisMode),
		"Call graph results are repository-local.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
		"Imported-package receiver calls may be incomplete.",
	}
}

func callAnalysisLimitation(analysisMode string) string {
	switch analysisMode {
	case sherpa.CallAnalysisModeTypechecked:
		return "Call analysis used typechecked package loading where available."
	case sherpa.CallAnalysisModeASTFallback:
		return "Call analysis used AST fallback because typechecked loading was unavailable."
	default:
		return "Call analysis used syntax plus local type information."
	}
}

func callPathLimitations() []string {
	return []string{
		"Path analysis uses repository-local call graph edges from syntax plus local type information.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
		"Only bounded shortest paths requested by the command are returned.",
	}
}

func entrypointLimitations(analysisMode string) []string {
	limitations := callLimitations(analysisMode)
	limitations = append(limitations,
		"Entry point classification is heuristic: main functions, test functions, exported functions, and functions with no local callers.",
		"Framework-specific entrypoints such as HTTP routers and CLI command handlers are not inferred yet.",
	)

	return limitations
}

func explainLimitations(referenceAnalysisMode string, callAnalysisMode string) []string {
	limitations := []string{
		"Explain analysis combines symbol, reference, impact, test, and call signals from local repository analysis.",
		"Purpose, risk, architecture role, and reading order are deterministic heuristics.",
	}
	if strings.TrimSpace(referenceAnalysisMode) != "" {
		limitations = append(limitations, referenceAnalysisLimitation(referenceAnalysisMode))
	}
	limitations = append(limitations, callLimitations(callAnalysisMode)...)
	limitations = append(limitations, testLimitations(analysisModeAST)...)

	return limitations
}

func impactBundleLimitations(analysisMode string, referenceAnalysisMode string, callAnalysisMode string) []string {
	limitations := impactLimitations(analysisMode)
	if strings.TrimSpace(referenceAnalysisMode) != "" {
		limitations = append(limitations, referenceAnalysisLimitation(referenceAnalysisMode))
	}
	if strings.TrimSpace(callAnalysisMode) != "" {
		limitations = append(limitations, callAnalysisLimitation(callAnalysisMode))
	}

	return limitations
}

func impactLimitations(analysisMode string) []string {
	if analysisMode == analysisModeDiff || analysisMode == analysisModeDiffTypechecked {
		semanticLine := "Impact analysis uses syntax plus local package dependency and interface signals."
		if analysisMode == analysisModeDiffTypechecked {
			semanticLine = "Impact analysis uses git diff plus typechecked symbol, reference, call, or interface signals where available."
		}

		return []string{
			"Diff impact is based on git changed files and hunk-level changed symbol extraction.",
			semanticLine,
			"Statement-level semantic consequences are not fully inferred.",
			"Dynamic dispatch, reflection, and function values are not resolved.",
		}
	}

	return []string{
		"Impact analysis uses syntax plus local package dependency and interface signals.",
		"Symbol impact includes references and conservative caller-package propagation.",
		"Interface impact uses typechecked method sets when package loading succeeds and AST fallback otherwise.",
		"Dynamic dispatch, reflection, and function values are not resolved.",
	}
}

func testLimitations(analysisMode string) []string {
	if analysisMode == analysisModeDiff || analysisMode == analysisModeDiffTypechecked {
		semanticLine := "Affected test planning is based on changed packages, affected packages, and syntactic test references."
		if analysisMode == analysisModeDiffTypechecked {
			semanticLine = "Affected test planning includes typechecked changed-symbol impact where available, then falls back to package-level commands."
		}

		return []string{
			semanticLine,
			"Literal t.Run subtest names are extracted; dynamic table-test names may be incomplete.",
			"Fallback commands are package-level when direct test functions are not known.",
		}
	}

	return []string{
		"Test discovery uses direct references, same-package tests, and literal t.Run subtest names.",
		"Dynamic table-test names may be incomplete.",
		"Fallback commands are package-level when direct test functions are not known.",
	}
}

func interfaceLimitations(analysisMode string) []string {
	switch analysisMode {
	case impactengine.InterfaceAnalysisModeTypechecked:
		return []string{
			"Interface analysis used typechecked method sets from repository-local packages.",
			"External implementations outside the repository are not reported.",
			"Build tags follow the default Go package loading environment.",
		}
	case impactengine.InterfaceAnalysisModeASTFallback:
		return []string{
			"Interface analysis used AST fallback because typechecked loading was unavailable.",
			"Embedded local interfaces are expanded, but alias, build-tag, and generic edge cases may be incomplete.",
			"External implementations outside the repository are not reported.",
		}
	default:
		return []string{
			"Interface analysis uses local method-set matching.",
			"External implementations outside the repository are not reported.",
		}
	}
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}
