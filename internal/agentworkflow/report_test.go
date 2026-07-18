package agentworkflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

func TestAnalyzeContextBuildsDiffFirstAgentWorkflow(t *testing.T) {
	root := writeAgentWorkflowProject(t)
	runAgentWorkflowGit(t, root, "init")
	runAgentWorkflowGit(t, root, "config", "user.email", "test@example.com")
	runAgentWorkflowGit(t, root, "config", "user.name", "Test User")
	runAgentWorkflowGit(t, root, "add", ".")
	runAgentWorkflowGit(t, root, "commit", "-m", "initial")

	servicePath := filepath.Join(root, "service.go")
	source, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	writeAgentWorkflowFile(t, servicePath, string(source)+"\nfunc Added() {}\n")

	report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{
		Limits: agentcontext.LimitOptions{
			MaxFiles:   10,
			MaxSymbols: 10,
			MaxTests:   10,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeContext returned error: %v", err)
	}

	if report.Target != "HEAD" || report.Base != "HEAD" {
		t.Fatalf("unexpected target/base: %#v", report)
	}
	if report.Readiness.PackageLoad.Status != "ok" {
		t.Fatalf("expected package load ok, got %#v", report.Readiness.PackageLoad)
	}
	if report.Readiness.RepositoryLayout.AnalysisBoundary != "module" {
		t.Fatalf("expected module repository layout, got %#v", report.Readiness.RepositoryLayout)
	}
	if report.Readiness.SkippedNestedModules == nil {
		t.Fatalf("expected non-nil skipped nested module array, got %#v", report.Readiness)
	}
	if report.Snapshot.Requested || report.Snapshot.Used {
		t.Fatalf("snapshot should not be requested or used, got %#v", report.Snapshot)
	}
	if report.Cost.PackageCount != 1 || report.Cost.GoFileCount != 2 || report.Cost.TestFileCount != 1 {
		t.Fatalf("expected repository cost counts, got %#v", report.Cost)
	}
	if report.Cost.SnapshotCountsAvailable {
		t.Fatalf("did not expect snapshot inventory counts without a snapshot, got %#v", report.Cost)
	}
	if len(report.ChangedFiles) != 1 || report.ChangedFiles[0] != "service.go" {
		t.Fatalf("expected changed service.go, got %#v", report.ChangedFiles)
	}
	if len(report.ChangedSymbolDetails) != 1 || report.ChangedSymbolDetails[0].Target != "example.com/app.Added" {
		t.Fatalf("expected Added changed symbol, got %#v", report.ChangedSymbolDetails)
	}
	if report.TargetRisk.Level == "" {
		t.Fatalf("expected target risk, got %#v", report.TargetRisk)
	}
	if report.PossibleRuntimeRelationships.Counts == nil || report.PossibleRuntimeRelationships.Examples == nil {
		t.Fatalf("expected non-nil possible relationship arrays, got %#v", report.PossibleRuntimeRelationships)
	}
	if report.InterfaceSummary.AffectedInterfaces == nil || report.InterfaceSummary.AffectedImplementations == nil {
		t.Fatalf("expected non-nil interface arrays, got %#v", report.InterfaceSummary)
	}
	if len(report.SectionModes) == 0 {
		t.Fatal("expected section modes")
	}
	if report.SectionTruncation == nil {
		t.Fatal("expected non-nil section truncation array")
	}
	assertAgentWorkflowSection(t, report.SectionModes, "readiness")
	assertAgentWorkflowSection(t, report.SectionModes, "context")
	assertAgentWorkflowSection(t, report.SectionModes, "impact")
	assertAgentWorkflowSection(t, report.SectionModes, "tests")
	assertAgentWorkflowSection(t, report.SectionModes, "snapshot")
	assertAgentWorkflowSection(t, report.SectionModes, "pr")
	if !agentWorkflowContains(report.SuggestedCommands, "gosherpa context diff --base HEAD --json") {
		t.Fatalf("expected context diff suggested command, got %#v", report.SuggestedCommands)
	}
	if !agentWorkflowContains(report.SuggestedCommands, "gosherpa context symbol example.com/app.Added --json") {
		t.Fatalf("expected changed-symbol drill-down command, got %#v", report.SuggestedCommands)
	}
	if strings.Contains(Format(report), "CONTEXT DIFF") {
		t.Fatalf("agent human output should not dump context diff output:\n%s", Format(report))
	}
}

func TestAnalyzeContextAppliesComposedByteLimit(t *testing.T) {
	root := writeAgentWorkflowProject(t)
	runAgentWorkflowGit(t, root, "init")
	runAgentWorkflowGit(t, root, "config", "user.email", "test@example.com")
	runAgentWorkflowGit(t, root, "config", "user.name", "Test User")
	runAgentWorkflowGit(t, root, "add", ".")
	runAgentWorkflowGit(t, root, "commit", "-m", "initial")

	servicePath := filepath.Join(root, "service.go")
	var builder strings.Builder
	for index := 0; index < 25; index++ {
		builder.WriteString("\nfunc Added")
		builder.WriteString(string(rune('A' + index)))
		builder.WriteString("() { Target() }\n")
	}
	source, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	writeAgentWorkflowFile(t, servicePath, string(source)+builder.String())

	const maxBytes = 4000
	report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{
		Limits: agentcontext.LimitOptions{
			MaxFiles:   40,
			MaxSymbols: 40,
			MaxTests:   40,
			MaxBytes:   maxBytes,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeContext returned error: %v", err)
	}

	if report.Limits == nil || report.Limits.MaxBytes != maxBytes {
		t.Fatalf("expected max byte limit in report, got %#v", report.Limits)
	}
	if report.Truncated == nil {
		t.Fatalf("expected truncation metadata")
	}
	if len(report.SectionTruncation) == 0 {
		t.Fatalf("expected section truncation metadata")
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > maxBytes && report.Truncated.ByteBudgetOverage == 0 {
		t.Fatalf("report exceeded byte budget without overage: got %d, want <= %d", len(data), maxBytes)
	}
	if report.ChangedFiles == nil || report.AffectedSymbols == nil || report.TestPlan.Direct == nil {
		t.Fatalf("expected normalized arrays after byte limiting, got %#v", report)
	}
}

func TestSectionTruncationFromTruncationIncludesEntrypoints(t *testing.T) {
	entries := sectionTruncationFromTruncation(&agentcontext.Truncation{
		EntryPointCounts:   2,
		EntryPointExamples: 3,
	})

	seen := map[string]int{}
	for _, entry := range entries {
		seen[entry.Section+"/"+entry.Field] = entry.Omitted
	}
	if seen["entrypoints/counts"] != 2 {
		t.Fatalf("expected entrypoint count truncation, got %#v", entries)
	}
	if seen["entrypoints/examples"] != 3 {
		t.Fatalf("expected entrypoint example truncation, got %#v", entries)
	}
}

func TestAnalyzeContextKeepsEmptyDiffArraysNonNil(t *testing.T) {
	root := writeAgentWorkflowProject(t)
	runAgentWorkflowGit(t, root, "init")
	runAgentWorkflowGit(t, root, "config", "user.email", "test@example.com")
	runAgentWorkflowGit(t, root, "config", "user.name", "Test User")
	runAgentWorkflowGit(t, root, "add", ".")
	runAgentWorkflowGit(t, root, "commit", "-m", "initial")

	report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeContext returned error: %v", err)
	}

	if len(report.ChangedFiles) != 0 || report.ChangedFiles == nil {
		t.Fatalf("expected empty non-nil changed files, got %#v", report.ChangedFiles)
	}
	if len(report.TestCommands) != 0 || report.TestCommands == nil {
		t.Fatalf("expected empty non-nil test commands, got %#v", report.TestCommands)
	}
	if report.TestPlan.Direct == nil || report.TestPlan.Related == nil || report.TestPlan.Fallback == nil {
		t.Fatalf("expected non-nil test plan arrays, got %#v", report.TestPlan)
	}
	if report.Purpose == "" {
		t.Fatal("expected purpose")
	}
}

func TestAnalyzeContextReportsSnapshotStates(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		root := writeCommittedAgentWorkflowProject(t)

		report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{UseSnapshot: true})
		if err != nil {
			t.Fatalf("AnalyzeContext returned error: %v", err)
		}

		if !report.Snapshot.Requested || report.Snapshot.Used {
			t.Fatalf("expected requested unused snapshot, got %#v", report.Snapshot)
		}
		if report.Snapshot.Status != snapshotstore.StatusMissing {
			t.Fatalf("expected missing snapshot, got %#v", report.Snapshot)
		}
		if report.Snapshot.RefreshCommand != "gosherpa snapshot --json" {
			t.Fatalf("expected refresh command, got %#v", report.Snapshot)
		}
		if !agentWorkflowContains(report.SuggestedCommands, "gosherpa snapshot --json") {
			t.Fatalf("expected snapshot refresh suggestion, got %#v", report.SuggestedCommands)
		}
	})

	t.Run("valid", func(t *testing.T) {
		root := writeCommittedAgentWorkflowProject(t)
		appendAgentWorkflowAddedFunction(t, root)
		writeAgentWorkflowSnapshot(t, root)

		report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{UseSnapshot: true})
		if err != nil {
			t.Fatalf("AnalyzeContext returned error: %v", err)
		}

		if !report.Snapshot.Requested || !report.Snapshot.Used {
			t.Fatalf("expected requested used snapshot, got %#v", report.Snapshot)
		}
		if report.Snapshot.Status != snapshotstore.StatusValid || !report.Snapshot.Fresh {
			t.Fatalf("expected valid fresh snapshot, got %#v", report.Snapshot)
		}
		if report.Snapshot.RefreshCommand != "" {
			t.Fatalf("did not expect refresh command for valid snapshot, got %#v", report.Snapshot)
		}
		if !report.Cost.SnapshotCountsAvailable || report.Cost.SymbolCount == 0 || report.Cost.RelationshipCount == 0 {
			t.Fatalf("expected snapshot-backed cost counts, got %#v", report.Cost)
		}
		if report.Cost.RelationshipCounts == nil {
			t.Fatalf("expected non-nil relationship counts")
		}
	})

	t.Run("stale", func(t *testing.T) {
		root := writeCommittedAgentWorkflowProject(t)
		writeAgentWorkflowSnapshot(t, root)
		appendAgentWorkflowAddedFunction(t, root)

		report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{UseSnapshot: true})
		if err != nil {
			t.Fatalf("AnalyzeContext returned error: %v", err)
		}

		if report.Snapshot.Used {
			t.Fatalf("expected stale snapshot fallback, got %#v", report.Snapshot)
		}
		if report.Snapshot.Status != snapshotstore.StatusStale {
			t.Fatalf("expected stale snapshot, got %#v", report.Snapshot)
		}
		if report.Snapshot.RefreshCommand != "gosherpa snapshot --json" {
			t.Fatalf("expected refresh command, got %#v", report.Snapshot)
		}
		if len(report.Warnings) == 0 {
			t.Fatalf("expected stale snapshot warning")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		root := writeCommittedAgentWorkflowProject(t)
		snapshotPath := snapshotstore.Path(root)
		writeAgentWorkflowFile(t, snapshotPath, "{not json")

		report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{UseSnapshot: true})
		if err != nil {
			t.Fatalf("AnalyzeContext returned error: %v", err)
		}

		if report.Snapshot.Used {
			t.Fatalf("expected invalid snapshot fallback, got %#v", report.Snapshot)
		}
		if report.Snapshot.Status != snapshotstore.StatusInvalid {
			t.Fatalf("expected invalid snapshot, got %#v", report.Snapshot)
		}
		if report.Snapshot.RefreshCommand != "gosherpa snapshot --json" {
			t.Fatalf("expected refresh command, got %#v", report.Snapshot)
		}
	})
}

func TestAnalyzeContextReportsRepositoryLayoutWarnings(t *testing.T) {
	root := writeCommittedAgentWorkflowProject(t)
	writeAgentWorkflowFile(t, filepath.Join(root, "nested", "go.mod"), "module example.com/nested\n\ngo 1.24.4\n")
	writeAgentWorkflowFile(t, filepath.Join(root, "nested", "nested.go"), "package nested\n")
	appendAgentWorkflowAddedFunction(t, root)

	report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeContext returned error: %v", err)
	}

	if len(report.Readiness.RepositoryLayout.SkippedNestedModules) != 1 || report.Readiness.RepositoryLayout.SkippedNestedModules[0] != "nested" {
		t.Fatalf("expected skipped nested module in repository layout, got %#v", report.Readiness.RepositoryLayout)
	}
	if !agentWorkflowContains(report.Readiness.RepoShapeWarnings, "nested modules skipped by the selected analysis boundary: nested; inspect them with --root <path> when needed") {
		t.Fatalf("expected skipped nested module readiness warning, got %#v", report.Readiness.RepoShapeWarnings)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected repository layout warning in envelope warnings")
	}
}

func TestAnalyzeContextReportsPackageLoadDiagnostics(t *testing.T) {
	root := writeCommittedAgentWorkflowProject(t)
	servicePath := filepath.Join(root, "service.go")
	writeAgentWorkflowFile(t, servicePath, `package app

func Entry() {
	Target()
}

func Target() {}

func Broken() {
	Missing()
}
`)

	report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeContext returned error: %v", err)
	}

	if report.Confidence != agentcontext.ConfidenceLow {
		t.Fatalf("expected low confidence from package load diagnostics, got %q", report.Confidence)
	}
	if report.Readiness.PackageLoad.Status != "warnings" {
		t.Fatalf("expected package load warnings, got %#v", report.Readiness.PackageLoad)
	}
	if len(report.Readiness.PackageLoad.Diagnostics) == 0 {
		t.Fatalf("expected package load diagnostics, got %#v", report.Readiness.PackageLoad)
	}
	diagnostic := report.Readiness.PackageLoad.Diagnostics[0]
	if diagnostic.Kind != "type-error" {
		t.Fatalf("expected type-error diagnostic, got %#v", diagnostic)
	}
	if diagnostic.File != "service.go" {
		t.Fatalf("expected root-relative diagnostic file, got %#v", diagnostic)
	}
	if !strings.Contains(diagnostic.Reason, "Missing") {
		t.Fatalf("expected diagnostic reason to mention Missing, got %#v", diagnostic)
	}
	if !agentWorkflowContains(diagnostic.AffectedSections, "context") || !agentWorkflowContains(diagnostic.AffectedSections, "tests") {
		t.Fatalf("expected agent affected sections, got %#v", diagnostic.AffectedSections)
	}
}

func TestAnalyzeContextSummarizesLargeGeneratedReadingOrder(t *testing.T) {
	root := writeAgentWorkflowProject(t)
	writeAgentWorkflowFile(t, filepath.Join(root, "generated_a.go"), smallGeneratedAgentWorkflowSource("A"))
	writeAgentWorkflowFile(t, filepath.Join(root, "generated_b.go"), smallGeneratedAgentWorkflowSource("B"))
	runAgentWorkflowGit(t, root, "init")
	runAgentWorkflowGit(t, root, "config", "user.email", "test@example.com")
	runAgentWorkflowGit(t, root, "config", "user.name", "Test User")
	runAgentWorkflowGit(t, root, "add", ".")
	runAgentWorkflowGit(t, root, "commit", "-m", "initial")

	appendAgentWorkflowAddedFunction(t, root)
	writeAgentWorkflowFile(t, filepath.Join(root, "generated_a.go"), largeGeneratedAgentWorkflowSource("A"))
	writeAgentWorkflowFile(t, filepath.Join(root, "generated_b.go"), largeGeneratedAgentWorkflowSource("B"))

	report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{
		Limits: agentcontext.LimitOptions{
			MaxFiles:   10,
			MaxSymbols: 10,
			MaxTests:   10,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeContext returned error: %v", err)
	}

	if report.Readiness.GeneratedFiles != 2 {
		t.Fatalf("expected generated file count, got %#v", report.Readiness)
	}
	if len(report.Readiness.GeneratedPackages) != 1 || report.Readiness.GeneratedPackages[0].Package != "." || report.Readiness.GeneratedPackages[0].Files != 2 {
		t.Fatalf("expected generated package summary, got %#v", report.Readiness.GeneratedPackages)
	}
	manualIndex := agentWorkflowReadingStepIndex(report.ReadingOrder, "Changed file: service.go")
	summaryIndex := agentWorkflowReadingStepIndex(report.ReadingOrder, "Generated files summary: 2 large changed files")
	if manualIndex < 0 || summaryIndex < 0 {
		t.Fatalf("expected manual and generated summary reading steps, got %#v", report.ReadingOrder)
	}
	if summaryIndex <= manualIndex {
		t.Fatalf("expected generated summary after hand-written file step, got %#v", report.ReadingOrder)
	}
	for _, step := range report.ReadingOrder {
		if step.Title == "Changed file: generated_a.go" || step.Title == "Changed file: generated_b.go" {
			t.Fatalf("expected large generated file steps to be summarized, got %#v", report.ReadingOrder)
		}
	}
	if !agentWorkflowContains(report.Limitations, "Large generated Go files are compiler-visible and included in analysis; agent reading order summarizes generated file steps after hand-written steps when they would otherwise dominate.") {
		t.Fatalf("expected generated reading-order limitation, got %#v", report.Limitations)
	}
}

func TestAnalyzeContextReportsBuildTagSnapshotMismatch(t *testing.T) {
	root := writeCommittedAgentWorkflowProject(t)
	builtSnapshot, err := snapshotstore.Build(root, snapshotstore.BuildOptions{BuildTags: []string{"enterprise"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotstore.Write(root, builtSnapshot); err != nil {
		t.Fatal(err)
	}

	report, err := AnalyzeContext(root, "HEAD", AnalyzeOptions{UseSnapshot: true})
	if err != nil {
		t.Fatalf("AnalyzeContext returned error: %v", err)
	}

	if report.Snapshot.Status != snapshotstore.StatusStale {
		t.Fatalf("expected stale snapshot, got %#v", report.Snapshot)
	}
	if !agentWorkflowContains(report.Snapshot.StaleReasons, "build tags changed") {
		t.Fatalf("expected build tag stale reason, got %#v", report.Snapshot.StaleReasons)
	}
	if report.Snapshot.Used {
		t.Fatalf("expected tag-mismatched snapshot fallback, got %#v", report.Snapshot)
	}
	if report.BuildTags == nil || len(report.BuildTags) != 0 {
		t.Fatalf("expected non-nil empty workflow build tags, got %#v", report.BuildTags)
	}
	if report.Readiness.PackageLoad.BuildTags == nil || len(report.Readiness.PackageLoad.BuildTags) != 0 {
		t.Fatalf("expected non-nil empty package load build tags, got %#v", report.Readiness.PackageLoad.BuildTags)
	}
}

func smallGeneratedAgentWorkflowSource(suffix string) string {
	return `// Code generated by gosherpa test. DO NOT EDIT.

package app

func Generated` + suffix + `() string { return "` + suffix + `" }
`
}

func largeGeneratedAgentWorkflowSource(suffix string) string {
	return `// Code generated by gosherpa test. DO NOT EDIT.

package app

func Generated` + suffix + `() string { return "` + strings.Repeat(suffix, 40*1024) + `" }
`
}

func agentWorkflowReadingStepIndex(steps []explainengine.ReadingStep, title string) int {
	for index, step := range steps {
		if step.Title == title {
			return index
		}
	}
	return -1
}

func writeAgentWorkflowProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeAgentWorkflowFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentWorkflowFile(t, filepath.Join(root, "service.go"), `package app

func Entry() {
	Target()
}

func Target() {}
`)
	writeAgentWorkflowFile(t, filepath.Join(root, "service_test.go"), `package app

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)
	return root
}

func writeCommittedAgentWorkflowProject(t *testing.T) string {
	t.Helper()
	root := writeAgentWorkflowProject(t)
	runAgentWorkflowGit(t, root, "init")
	runAgentWorkflowGit(t, root, "config", "user.email", "test@example.com")
	runAgentWorkflowGit(t, root, "config", "user.name", "Test User")
	runAgentWorkflowGit(t, root, "add", ".")
	runAgentWorkflowGit(t, root, "commit", "-m", "initial")
	return root
}

func appendAgentWorkflowAddedFunction(t *testing.T, root string) {
	t.Helper()
	servicePath := filepath.Join(root, "service.go")
	source, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	writeAgentWorkflowFile(t, servicePath, string(source)+"\nfunc Added() {}\n")
}

func writeAgentWorkflowSnapshot(t *testing.T, root string) {
	t.Helper()
	snapshot, err := snapshotstore.Build(root, snapshotstore.BuildOptions{})
	if err != nil {
		t.Fatalf("Build snapshot returned error: %v", err)
	}
	if _, err := snapshotstore.Write(root, snapshot); err != nil {
		t.Fatalf("Write snapshot returned error: %v", err)
	}
}

func writeAgentWorkflowFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func runAgentWorkflowGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func assertAgentWorkflowSection(t *testing.T, modes []SectionMode, section string) {
	t.Helper()
	for _, mode := range modes {
		if mode.Section == section {
			if mode.AnalysisMode == "" || mode.Confidence == "" || mode.Limitations == nil {
				t.Fatalf("section %s is incomplete: %#v", section, mode)
			}
			return
		}
	}
	t.Fatalf("missing section %s in %#v", section, modes)
}

func agentWorkflowContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
