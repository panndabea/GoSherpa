package agentworkflow

import (
	"fmt"
	"sort"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/repostats"
	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
	"github.com/panndabea/GoSherpa/internal/symbolindex"
)

const (
	AnalysisModeLive         = "live"
	AnalysisModeLiveFallback = "live-fallback"
)

type AnalyzeOptions struct {
	BuildTags   []string
	UseSnapshot bool
	Limits      agentcontext.LimitOptions
}

type Report struct {
	Target                       string                       `json:"target"`
	Base                         string                       `json:"base"`
	Purpose                      string                       `json:"purpose"`
	BuildTags                    []string                     `json:"buildTags"`
	Readiness                    ReadinessSummary             `json:"readiness"`
	Snapshot                     SnapshotSummary              `json:"snapshot"`
	Cost                         repostats.CostSummary        `json:"cost"`
	ChangedFiles                 []string                     `json:"changedFiles"`
	ChangedPackages              []string                     `json:"changedPackages"`
	AffectedPackages             []string                     `json:"affectedPackages"`
	AffectedSymbols              []string                     `json:"affectedSymbols"`
	ChangedSymbolDetails         []impactengine.ChangedSymbol `json:"changedSymbolDetails"`
	ReadingOrder                 []explainengine.ReadingStep  `json:"readingOrder"`
	TargetRisk                   sherpa.TargetRiskSummary     `json:"targetRisk"`
	PossibleRuntimeRelationships PossibleRuntimeSummary       `json:"possibleRuntimeRelationships"`
	InterfaceSummary             InterfaceSummary             `json:"interfaceSummary"`
	EntryPointSummary            sherpa.EntryPointSummary     `json:"entrypointSummary"`
	TestPlan                     sherpa.TestPlan              `json:"testPlan"`
	TestCommands                 []string                     `json:"testCommands"`
	SuggestedCommands            []string                     `json:"suggestedCommands"`
	SectionModes                 []SectionMode                `json:"sectionModes"`
	SectionTruncation            []SectionTruncation          `json:"sectionTruncation"`
	AnalysisMode                 string                       `json:"analysisMode"`
	Confidence                   string                       `json:"confidence"`
	Limitations                  []string                     `json:"limitations"`
	Limits                       *agentcontext.LimitOptions   `json:"limits,omitempty"`
	Truncated                    *agentcontext.Truncation     `json:"truncated,omitempty"`
	Warnings                     []string                     `json:"-"`
}

type ReadinessSummary struct {
	Status               string                           `json:"status"`
	AnalysisMode         string                           `json:"analysisMode"`
	Confidence           string                           `json:"confidence"`
	RepositoryLayout     sherpa.RepositoryLayout          `json:"repositoryLayout"`
	PackageLoad          PackageLoadSummary               `json:"packageLoad"`
	GoWork               GoWorkSummary                    `json:"goWork"`
	GeneratedFiles       int                              `json:"generatedFiles"`
	GeneratedPackages    []sherpa.GeneratedPackageSummary `json:"generatedPackages"`
	NestedModules        []string                         `json:"nestedModules"`
	SkippedNestedModules []string                         `json:"skippedNestedModules"`
	RepoShapeWarnings    []string                         `json:"repoShapeWarnings"`
	Limitations          []string                         `json:"limitations"`
}

type PackageLoadSummary struct {
	Status           string                            `json:"status"`
	AnalysisMode     string                            `json:"analysisMode"`
	BuildTags        []string                          `json:"buildTags"`
	AffectedSections []string                          `json:"affectedSections"`
	PackageCount     int                               `json:"packageCount"`
	WarningCount     int                               `json:"warningCount"`
	Warnings         []string                          `json:"warnings"`
	Diagnostics      []semantics.PackageLoadDiagnostic `json:"diagnostics"`
	Message          string                            `json:"message,omitempty"`
}

type GoWorkSummary struct {
	Detected bool   `json:"detected"`
	Path     string `json:"path,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

type SnapshotSummary struct {
	Requested            bool                               `json:"requested"`
	Used                 bool                               `json:"used"`
	Status               string                             `json:"status"`
	Path                 string                             `json:"path,omitempty"`
	Message              string                             `json:"message"`
	Fresh                bool                               `json:"fresh"`
	FormatVersion        int                                `json:"formatVersion,omitempty"`
	CreatedAt            string                             `json:"createdAt,omitempty"`
	RelationshipMetadata snapshotstore.RelationshipMetadata `json:"relationshipMetadata"`
	StaleReasons         []string                           `json:"staleReasons"`
	RefreshCommand       string                             `json:"refreshCommand,omitempty"`
}

type PossibleRuntimeSummary struct {
	Counts      []PossibleRuntimeCount   `json:"counts"`
	Examples    []PossibleRuntimeExample `json:"examples"`
	Limitations []string                 `json:"limitations"`
}

type PossibleRuntimeCount struct {
	Reason    string `json:"reason"`
	Scope     string `json:"scope"`
	Certainty string `json:"certainty"`
	Count     int    `json:"count"`
}

type PossibleRuntimeExample struct {
	Caller      string              `json:"caller"`
	Callee      string              `json:"callee,omitempty"`
	Reason      string              `json:"reason"`
	Scope       string              `json:"scope"`
	Certainty   string              `json:"certainty"`
	Position    sherpa.Position     `json:"position"`
	Range       *sherpa.SourceRange `json:"range,omitempty"`
	Limitations []string            `json:"limitations"`
}

type InterfaceSummary struct {
	AnalysisMode            string   `json:"analysisMode,omitempty"`
	AffectedInterfaces      []string `json:"affectedInterfaces"`
	AffectedImplementations []string `json:"affectedImplementations"`
	Limitations             []string `json:"limitations"`
}

type SectionMode struct {
	Section      string   `json:"section"`
	AnalysisMode string   `json:"analysisMode"`
	Confidence   string   `json:"confidence"`
	Limitations  []string `json:"limitations"`
}

type SectionTruncation struct {
	Section string `json:"section"`
	Field   string `json:"field"`
	Omitted int    `json:"omitted"`
}

func AnalyzeContext(root string, base string, options AnalyzeOptions) (Report, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return Report{}, fmt.Errorf("base ref is required")
	}

	buildTags := semantics.NormalizeBuildTags(options.BuildTags)
	readiness := analyzeReadiness(root, buildTags)

	contextReport, err := agentcontext.AnalyzeDiff(root, base, agentcontext.DiffAnalyzeOptions{
		BuildTags:   buildTags,
		UseSnapshot: options.UseSnapshot,
		Limits:      options.Limits,
	})
	if err != nil {
		return Report{}, err
	}

	snapshotUsed := contextReport.SnapshotUsed || strings.HasPrefix(contextReport.AnalysisMode, "snapshot+")
	snapshotInspect := contextReport.SnapshotInspect
	if strings.TrimSpace(snapshotInspect.Status) == "" {
		snapshotInspect = snapshotstore.Inspect(root, snapshotstore.BuildOptions{BuildTags: buildTags})
	}
	snapshotSummary := summarizeSnapshot(snapshotInspect, options.UseSnapshot, snapshotUsed)
	possibleRuntime := possibleRuntimeSummary(contextReport.SnapshotRelationships, options.UseSnapshot && snapshotUsed, snapshotInspect.Status)
	cost := repostats.SummarizeCost(repostats.CostInput{
		Layout:                  readiness.RepositoryLayout,
		PackageCount:            readiness.PackageLoad.PackageCount,
		PackageLoadWarningCount: readiness.PackageLoad.WarningCount,
		Snapshot:                snapshotInspect,
		ChangedFileCount:        len(contextReport.ChangedFiles),
		ChangedPackageCount:     len(contextReport.ChangedPackages),
		AffectedPackageCount:    len(contextReport.AffectedPackages),
		AffectedSymbolCount:     len(contextReport.AffectedSymbols),
		TestCommandCount:        len(contextReport.TestCommands),
	})

	report := Report{
		Target:                       base,
		Base:                         base,
		Purpose:                      workflowPurpose(contextReport),
		BuildTags:                    buildTags,
		Readiness:                    readiness,
		Snapshot:                     snapshotSummary,
		Cost:                         cost,
		ChangedFiles:                 contextReport.ChangedFiles,
		ChangedPackages:              contextReport.ChangedPackages,
		AffectedPackages:             contextReport.AffectedPackages,
		AffectedSymbols:              contextReport.AffectedSymbols,
		ChangedSymbolDetails:         contextReport.ChangedSymbolDetails,
		ReadingOrder:                 contextReport.ReadingOrder,
		TargetRisk:                   contextReport.TargetRisk,
		PossibleRuntimeRelationships: possibleRuntime,
		InterfaceSummary: InterfaceSummary{
			AnalysisMode:            strings.TrimSpace(contextReport.InterfaceAnalysisMode),
			AffectedInterfaces:      contextReport.AffectedInterfaces,
			AffectedImplementations: contextReport.AffectedImplementations,
			Limitations:             interfaceSummaryLimitations(contextReport.InterfaceAnalysisMode),
		},
		EntryPointSummary: entryPointSummaryFromContext(contextReport),
		TestPlan:          contextReport.TestPlan,
		TestCommands:      contextReport.TestCommands,
		AnalysisMode:      contextReport.AnalysisMode,
		Limits:            workflowLimits(contextReport.Limits, options.Limits.MaxBytes),
		Truncated:         contextReport.Truncated,
		Warnings:          uniqueStringsInOrder(append(append(readiness.PackageLoad.Warnings, readiness.RepoShapeWarnings...), contextReport.Warnings...)),
		SectionTruncation: sectionTruncationFromTruncation(contextReport.Truncated),
		SuggestedCommands: suggestedCommands(base, snapshotSummary, contextReport.ChangedSymbolDetails),
	}
	report = applyGeneratedFilePolicy(root, report)
	report.SectionModes = sectionModes(report.Readiness, snapshotSummary, contextReport)
	report.Limitations = workflowLimitations(report, contextReport)
	report.Confidence = workflowConfidence(report, contextReport)
	report = applyAgentByteLimit(report, options.Limits.MaxBytes)

	return normalizeReport(report), nil
}

func analyzeReadiness(root string, buildTags []string) ReadinessSummary {
	layout, layoutWarnings := sherpa.AnalyzeRepositoryLayout(root)
	readiness := ReadinessSummary{
		Status:           "ready",
		AnalysisMode:     "typechecked",
		Confidence:       agentcontext.ConfidenceMedium,
		RepositoryLayout: layout,
		GoWork: GoWorkSummary{
			Detected: layout.GoWork.Detected,
			Path:     layout.GoWork.Path,
			Scope:    layout.GoWork.Scope,
		},
		GeneratedFiles:       layout.GeneratedFiles,
		GeneratedPackages:    layout.GeneratedPackages,
		NestedModules:        layout.NestedModules,
		SkippedNestedModules: layout.SkippedNestedModules,
		Limitations: []string{
			"Readiness summarizes repository shape and package loading; it does not prove every downstream relationship is complete.",
			"Package loading follows the current Go environment and provided --tags values.",
			"Generated files are analyzed when visible to package loading; major generated packages are summarized for repository-shape triage.",
		},
	}

	readiness.RepoShapeWarnings = repoShapeWarnings(readiness, layoutWarnings)

	repo, err := semantics.LoadRepository(root, semantics.LoadOptions{BuildTags: buildTags})
	if err != nil {
		message := fmt.Sprintf("typechecked package loading failed: %v", err)
		readiness.Status = "limited"
		readiness.AnalysisMode = "unavailable"
		readiness.PackageLoad = PackageLoadSummary{
			Status:           "failed",
			AnalysisMode:     "unavailable",
			BuildTags:        buildTags,
			AffectedSections: agentPackageLoadSections(),
			WarningCount:     1,
			Warnings:         []string{message},
			Diagnostics:      packageLoadFailureDiagnostics(message, agentPackageLoadSections()),
			Message:          message,
		}
	} else {
		status := "ok"
		if len(repo.Warnings) > 0 {
			status = "warnings"
			readiness.Status = "warnings"
		}
		readiness.PackageLoad = PackageLoadSummary{
			Status:           status,
			AnalysisMode:     "typechecked",
			BuildTags:        buildTags,
			AffectedSections: agentPackageLoadSections(),
			PackageCount:     len(repo.Packages),
			WarningCount:     len(repo.Warnings),
			Warnings:         uniqueStringsInOrder(repo.Warnings),
			Diagnostics:      semantics.PackageLoadDiagnosticsWithSections(repo.Diagnostics, agentPackageLoadSections()),
		}
	}

	if len(readiness.RepoShapeWarnings) > 0 && readiness.Status == "ready" {
		readiness.Status = "warnings"
	}
	if readiness.Status != "ready" {
		readiness.Confidence = agentcontext.ConfidenceLow
	}

	return normalizeReadiness(readiness)
}

func agentPackageLoadSections() []string {
	return []string{"readiness", "context", "impact", "interfaces", "tests"}
}

func packageLoadFailureDiagnostics(message string, affectedSections []string) []semantics.PackageLoadDiagnostic {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}

	return semantics.PackageLoadDiagnosticsWithSections([]semantics.PackageLoadDiagnostic{
		{
			Package: "repository",
			Kind:    semantics.PackageLoadDiagnosticKindLoadError,
			Reason:  message,
			Message: message,
		},
	}, affectedSections)
}

func repoShapeWarnings(readiness ReadinessSummary, layoutWarnings []string) []string {
	warnings := append([]string{}, layoutWarnings...)
	if readiness.GoWork.Detected {
		warnings = append(warnings, "go.work detected; package loading follows workspace module resolution.")
	}
	if len(readiness.SkippedNestedModules) > 0 && !hasWarningPrefix(warnings, "nested modules skipped by the selected analysis boundary:") {
		warnings = append(warnings, fmt.Sprintf("nested modules skipped by the selected analysis boundary: %s", strings.Join(readiness.SkippedNestedModules, ", ")))
	}
	if readiness.GeneratedFiles > 0 {
		warnings = append(warnings, fmt.Sprintf("generated Go files detected: %d", readiness.GeneratedFiles))
	}
	return uniqueStringsInOrder(warnings)
}

func hasWarningPrefix(warnings []string, prefix string) bool {
	for _, warning := range warnings {
		if strings.HasPrefix(warning, prefix) {
			return true
		}
	}
	return false
}

func summarizeSnapshot(inspect snapshotstore.InspectResult, requested bool, used bool) SnapshotSummary {
	summary := SnapshotSummary{
		Requested:            requested,
		Used:                 used,
		Status:               inspect.Status,
		Path:                 inspect.Path,
		Message:              inspect.Message,
		Fresh:                inspect.Status == snapshotstore.StatusValid,
		FormatVersion:        inspect.FormatVersion,
		CreatedAt:            inspect.CreatedAt,
		RelationshipMetadata: inspect.RelationshipMetadata,
		StaleReasons:         inspect.StaleReasons,
	}
	if !requested {
		summary.Message = "Snapshot reuse was not requested for this agent workflow."
	}
	if requested && !used {
		summary.RefreshCommand = "gosherpa snapshot --json"
	}
	if inspect.Status != snapshotstore.StatusValid {
		summary.RefreshCommand = "gosherpa snapshot --json"
	}
	return normalizeSnapshotSummary(summary)
}

func possibleRuntimeSummary(relationships symbolindex.RelationshipIndex, useSnapshot bool, snapshotStatus string) PossibleRuntimeSummary {
	summary := PossibleRuntimeSummary{
		Limitations: []string{
			"Possible runtime relationships are separate from direct call edges and keep certainty labels explicit.",
			"Possible runtime relationships are conservative static signals; reflection and hidden framework wiring can still be missed.",
		},
	}
	if !useSnapshot {
		summary.Limitations = append(summary.Limitations, "Examples require --use-snapshot with a valid relationship-capable snapshot in this first agent workflow slice.")
		return normalizePossibleRuntimeSummary(summary)
	}

	if snapshotStatus != snapshotstore.StatusValid {
		summary.Limitations = append(summary.Limitations, "The snapshot was not valid, so possible runtime examples were not reused.")
		return normalizePossibleRuntimeSummary(summary)
	}

	counts := make(map[string]PossibleRuntimeCount)
	for _, record := range relationships.PossibleCallEdges {
		reason := valueOrUnknown(record.Reason)
		scope := valueOrUnknown(string(record.CallScope))
		certainty := valueOrUnknown(string(record.Certainty))
		key := reason + "\x00" + scope + "\x00" + certainty
		count := counts[key]
		count.Reason = reason
		count.Scope = scope
		count.Certainty = certainty
		count.Count++
		counts[key] = count

		if len(summary.Examples) < 5 {
			summary.Examples = append(summary.Examples, possibleRuntimeExample(record))
		}
	}
	for _, count := range counts {
		summary.Counts = append(summary.Counts, count)
	}
	if len(summary.Counts) == 0 {
		summary.Limitations = append(summary.Limitations, "The valid snapshot did not contain possible runtime relationship records.")
	}
	return normalizePossibleRuntimeSummary(summary)
}

func possibleRuntimeExample(record symbolindex.PossibleCallEdgeRecord) PossibleRuntimeExample {
	return PossibleRuntimeExample{
		Caller:      symbolIdentityName(record.Source),
		Callee:      symbolIdentityName(record.Target),
		Reason:      valueOrUnknown(record.Reason),
		Scope:       valueOrUnknown(string(record.CallScope)),
		Certainty:   valueOrUnknown(string(record.Certainty)),
		Position:    record.Position,
		Range:       record.Range,
		Limitations: nonNilSlice(record.Limitations),
	}
}

func symbolIdentityName(identity symbolindex.SymbolIdentity) string {
	if strings.TrimSpace(identity.QualifiedName) != "" {
		return identity.QualifiedName
	}
	if strings.TrimSpace(identity.Receiver) != "" && strings.TrimSpace(identity.Name) != "" {
		return identity.Receiver + "." + identity.Name
	}
	return strings.TrimSpace(identity.Name)
}

func workflowPurpose(contextReport agentcontext.DiffReport) string {
	if strings.TrimSpace(contextReport.Purpose) != "" {
		return "Agent diff context for " + contextReport.Base + ": " + contextReport.Purpose
	}
	return "Agent diff context for " + contextReport.Base + "."
}

func workflowLimits(limits *agentcontext.LimitOptions, maxBytes int) *agentcontext.LimitOptions {
	if limits == nil {
		return nil
	}
	copied := *limits
	if maxBytes > 0 {
		copied.MaxBytes = maxBytes
	} else {
		copied.MaxBytes = 0
	}
	copied.MaxReferences = 0
	copied.SourceRadius = nil
	return &copied
}

func interfaceSummaryLimitations(mode string) []string {
	if strings.TrimSpace(mode) == "" {
		return []string{"Interface summary is empty because no interface subanalysis mode was reported for this diff."}
	}
	return []string{"Interface summary is bounded to affected interface and implementation names, not full interface profiles."}
}

func sectionModes(readiness ReadinessSummary, snapshot SnapshotSummary, contextReport agentcontext.DiffReport) []SectionMode {
	snapshotMode := AnalysisModeLive
	snapshotConfidence := agentcontext.ConfidenceMedium
	if snapshot.Used {
		snapshotMode = "snapshot"
	} else if snapshot.Requested {
		snapshotMode = AnalysisModeLiveFallback
		snapshotConfidence = agentcontext.ConfidenceLow
	}

	testMode := strings.TrimSpace(contextReport.TestAnalysisMode)
	if testMode == "" {
		testMode = contextReport.AnalysisMode
	}
	interfaceMode := strings.TrimSpace(contextReport.InterfaceAnalysisMode)
	if interfaceMode == "" {
		interfaceMode = contextReport.AnalysisMode
	}

	return []SectionMode{
		{Section: "readiness", AnalysisMode: readiness.AnalysisMode, Confidence: readiness.Confidence, Limitations: readiness.Limitations},
		{Section: "snapshot", AnalysisMode: snapshotMode, Confidence: snapshotConfidence, Limitations: snapshotSectionLimitations(snapshot)},
		{Section: "context", AnalysisMode: contextReport.AnalysisMode, Confidence: contextReport.Confidence, Limitations: contextReport.Limitations},
		{Section: "impact", AnalysisMode: contextReport.AnalysisMode, Confidence: contextReport.Confidence, Limitations: impactSectionLimitations(contextReport)},
		{Section: "interfaces", AnalysisMode: interfaceMode, Confidence: contextReport.Confidence, Limitations: interfaceSummaryLimitations(contextReport.InterfaceAnalysisMode)},
		{Section: "entrypoints", AnalysisMode: entryPointSectionMode(contextReport), Confidence: entryPointSectionConfidence(contextReport), Limitations: entryPointSectionLimitations(contextReport)},
		{Section: "tests", AnalysisMode: testMode, Confidence: contextReport.Confidence, Limitations: testSectionLimitations(contextReport)},
		{Section: "pr", AnalysisMode: contextReport.AnalysisMode, Confidence: contextReport.Confidence, Limitations: []string{"PR summary is represented as bounded suggested commands and verification-oriented test recommendations, not the full pr report."}},
	}
}

func entryPointSummaryFromContext(contextReport agentcontext.DiffReport) sherpa.EntryPointSummary {
	if contextReport.EntryPointSummary == nil {
		return sherpa.NormalizeEntryPointSummary(sherpa.EntryPointSummary{
			AnalysisMode: contextReport.CallAnalysisMode,
			Confidence:   sherpa.EntryPointSummaryConfidenceLow,
			Limitations:  sherpa.EntryPointSummaryLimitations(false, contextReport.CallAnalysisMode),
		})
	}

	return sherpa.NormalizeEntryPointSummary(*contextReport.EntryPointSummary)
}

func entryPointSectionMode(contextReport agentcontext.DiffReport) string {
	if contextReport.EntryPointSummary != nil {
		return sherpa.NormalizeEntryPointSummary(*contextReport.EntryPointSummary).AnalysisMode
	}
	if strings.TrimSpace(contextReport.CallAnalysisMode) != "" {
		return contextReport.CallAnalysisMode
	}
	return contextReport.AnalysisMode
}

func entryPointSectionConfidence(contextReport agentcontext.DiffReport) string {
	if contextReport.EntryPointSummary != nil {
		return sherpa.NormalizeEntryPointSummary(*contextReport.EntryPointSummary).Confidence
	}
	return agentcontext.ConfidenceLow
}

func entryPointSectionLimitations(contextReport agentcontext.DiffReport) []string {
	return entryPointSummaryFromContext(contextReport).Limitations
}

func snapshotSectionLimitations(snapshot SnapshotSummary) []string {
	if snapshot.Used {
		return []string{"A valid snapshot was reused where supported; unsupported subanalysis still uses live repository analysis."}
	}
	if snapshot.Requested {
		return []string{"Snapshot reuse was requested but was not used for at least one supported section; inspect envelope warnings and refresh guidance."}
	}
	return []string{"Snapshot reuse was not requested; this workflow used live repository analysis."}
}

func impactSectionLimitations(contextReport agentcontext.DiffReport) []string {
	limitations := []string{
		"Impact is summarized from diff context signals and does not embed the full impact diff report.",
		"Target risk remains separate from confidence and repository structural risk.",
	}
	if strings.TrimSpace(contextReport.ReferenceAnalysisMode) != "" {
		limitations = append(limitations, "Reference analysis mode: "+contextReport.ReferenceAnalysisMode+".")
	}
	if strings.TrimSpace(contextReport.CallAnalysisMode) != "" {
		limitations = append(limitations, "Call analysis mode: "+contextReport.CallAnalysisMode+".")
	}
	return limitations
}

func testSectionLimitations(contextReport agentcontext.DiffReport) []string {
	limitations := []string{"Affected-test planning is included by default for agent context; --tests is not required."}
	if strings.TrimSpace(contextReport.TestAnalysisMode) != "" {
		limitations = append(limitations, "Test analysis mode: "+contextReport.TestAnalysisMode+".")
	}
	return limitations
}

func suggestedCommands(base string, snapshot SnapshotSummary, changedSymbols []impactengine.ChangedSymbol) []string {
	var commands []string
	commands = append(commands, "gosherpa doctor --json")
	if strings.TrimSpace(snapshot.RefreshCommand) != "" {
		commands = append(commands, snapshot.RefreshCommand)
	}
	snapshotFlag := ""
	if snapshot.Requested {
		snapshotFlag = " --use-snapshot"
	}
	commands = append(commands,
		fmt.Sprintf("gosherpa context diff --base %s%s --json", base, snapshotFlag),
		fmt.Sprintf("gosherpa impact diff --base %s%s --json", base, snapshotFlag),
		fmt.Sprintf("gosherpa tests affected --base %s%s --json", base, snapshotFlag),
		fmt.Sprintf("gosherpa pr --base %s%s --json", base, snapshotFlag),
	)
	for _, symbol := range changedSymbols {
		target := strings.TrimSpace(symbol.Target)
		if target == "" {
			continue
		}
		commands = append(commands, fmt.Sprintf("gosherpa context symbol %s%s --json", target, snapshotFlag))
		if len(commands) >= 9 {
			break
		}
	}
	return uniqueStringsInOrder(commands)
}

func workflowLimitations(report Report, contextReport agentcontext.DiffReport) []string {
	limitations := []string{
		"Agent context is diff-first and requires a base ref; symbol, file, and package drill-down remain separate context commands.",
		"The workflow composes bounded summaries and suggested follow-up commands instead of embedding full context, impact, tests, and pr reports.",
		"Affected-test planning is included by default; --tests is not part of this command contract.",
		"Snapshot reuse is opt-in and never creates snapshots automatically.",
	}
	limitations = append(limitations, report.Limitations...)
	limitations = append(limitations, contextReport.Limitations...)
	limitations = append(limitations, report.PossibleRuntimeRelationships.Limitations...)
	limitations = append(limitations, report.EntryPointSummary.Limitations...)
	return uniqueStringsInOrder(limitations)
}

func workflowConfidence(report Report, contextReport agentcontext.DiffReport) string {
	if len(report.Warnings) > 0 {
		return agentcontext.ConfidenceLow
	}
	if report.Readiness.Confidence == agentcontext.ConfidenceLow || contextReport.Confidence == agentcontext.ConfidenceLow {
		return agentcontext.ConfidenceLow
	}
	if report.Snapshot.Requested && !report.Snapshot.Used {
		return agentcontext.ConfidenceLow
	}
	return agentcontext.ConfidenceMedium
}

func normalizeReport(report Report) Report {
	report.BuildTags = nonNilSlice(semantics.NormalizeBuildTags(report.BuildTags))
	report.ChangedFiles = nonNilSlice(report.ChangedFiles)
	report.ChangedPackages = nonNilSlice(report.ChangedPackages)
	report.AffectedPackages = nonNilSlice(report.AffectedPackages)
	report.AffectedSymbols = nonNilSlice(report.AffectedSymbols)
	report.ChangedSymbolDetails = nonNilSlice(report.ChangedSymbolDetails)
	report.ReadingOrder = nonNilSlice(report.ReadingOrder)
	report.TargetRisk = sherpa.NormalizeTargetRiskSummary(report.TargetRisk)
	report.PossibleRuntimeRelationships = normalizePossibleRuntimeSummary(report.PossibleRuntimeRelationships)
	report.Cost = repostats.NormalizeCostSummary(report.Cost)
	report.InterfaceSummary.AffectedInterfaces = nonNilSlice(report.InterfaceSummary.AffectedInterfaces)
	report.InterfaceSummary.AffectedImplementations = nonNilSlice(report.InterfaceSummary.AffectedImplementations)
	report.InterfaceSummary.Limitations = nonNilSlice(report.InterfaceSummary.Limitations)
	report.EntryPointSummary = sherpa.NormalizeEntryPointSummary(report.EntryPointSummary)
	report.TestPlan = sherpa.NormalizeTestPlan(report.TestPlan)
	report.TestCommands = nonNilSlice(report.TestCommands)
	report.SuggestedCommands = nonNilSlice(report.SuggestedCommands)
	report.SectionModes = nonNilSlice(report.SectionModes)
	report.SectionTruncation = normalizeSectionTruncation(report.SectionTruncation)
	report.Limitations = nonNilSlice(uniqueStringsInOrder(report.Limitations))
	report.Warnings = nonNilSlice(uniqueStringsInOrder(report.Warnings))
	report.Readiness = normalizeReadiness(report.Readiness)
	report.Snapshot = normalizeSnapshotSummary(report.Snapshot)
	if strings.TrimSpace(report.Target) == "" {
		report.Target = report.Base
	}
	if strings.TrimSpace(report.AnalysisMode) == "" {
		report.AnalysisMode = agentcontext.AnalysisModeDiff
	}
	if strings.TrimSpace(report.Confidence) == "" {
		report.Confidence = agentcontext.ConfidenceMedium
	}
	return report
}

func normalizeReadiness(readiness ReadinessSummary) ReadinessSummary {
	readiness.RepositoryLayout = sherpa.NormalizeRepositoryLayout(readiness.RepositoryLayout)
	readiness.PackageLoad.BuildTags = nonNilSlice(semantics.NormalizeBuildTags(readiness.PackageLoad.BuildTags))
	readiness.PackageLoad.AffectedSections = nonNilSlice(readiness.PackageLoad.AffectedSections)
	readiness.PackageLoad.Warnings = nonNilSlice(uniqueStringsInOrder(readiness.PackageLoad.Warnings))
	readiness.PackageLoad.Diagnostics = nonNilSlice(semantics.PackageLoadDiagnosticsWithSections(readiness.PackageLoad.Diagnostics, readiness.PackageLoad.AffectedSections))
	readiness.GeneratedPackages = nonNilSlice(sherpa.NormalizeGeneratedPackageSummaries(readiness.GeneratedPackages))
	readiness.NestedModules = nonNilSlice(readiness.NestedModules)
	readiness.SkippedNestedModules = nonNilSlice(readiness.SkippedNestedModules)
	readiness.RepoShapeWarnings = nonNilSlice(uniqueStringsInOrder(readiness.RepoShapeWarnings))
	readiness.Limitations = nonNilSlice(readiness.Limitations)
	if strings.TrimSpace(readiness.Status) == "" {
		readiness.Status = "ready"
	}
	if strings.TrimSpace(readiness.AnalysisMode) == "" {
		readiness.AnalysisMode = "typechecked"
	}
	if strings.TrimSpace(readiness.Confidence) == "" {
		readiness.Confidence = agentcontext.ConfidenceMedium
	}
	if strings.TrimSpace(readiness.PackageLoad.Status) == "" {
		readiness.PackageLoad.Status = "unknown"
	}
	if strings.TrimSpace(readiness.PackageLoad.AnalysisMode) == "" {
		readiness.PackageLoad.AnalysisMode = readiness.AnalysisMode
	}
	return readiness
}

func normalizeSnapshotSummary(summary SnapshotSummary) SnapshotSummary {
	summary.StaleReasons = nonNilSlice(summary.StaleReasons)
	if strings.TrimSpace(summary.Status) == "" {
		summary.Status = snapshotstore.StatusMissing
	}
	if strings.TrimSpace(summary.Message) == "" {
		summary.Message = "Snapshot status is unavailable."
	}
	return summary
}

func normalizePossibleRuntimeSummary(summary PossibleRuntimeSummary) PossibleRuntimeSummary {
	summary.Counts = nonNilSlice(summary.Counts)
	summary.Examples = nonNilSlice(summary.Examples)
	summary.Limitations = nonNilSlice(uniqueStringsInOrder(summary.Limitations))
	sort.Slice(summary.Counts, func(i int, j int) bool {
		if summary.Counts[i].Reason != summary.Counts[j].Reason {
			return summary.Counts[i].Reason < summary.Counts[j].Reason
		}
		if summary.Counts[i].Scope != summary.Counts[j].Scope {
			return summary.Counts[i].Scope < summary.Counts[j].Scope
		}
		return summary.Counts[i].Certainty < summary.Counts[j].Certainty
	})
	return summary
}

func valueOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

func uniqueStringsInOrder(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
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
	return result
}
