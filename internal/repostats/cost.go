package repostats

import (
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

type CostInput struct {
	Layout                  sherpa.RepositoryLayout
	PackageCount            int
	PackageLoadWarningCount int
	Snapshot                snapshotstore.InspectResult
	ChangedFileCount        int
	ChangedPackageCount     int
	AffectedPackageCount    int
	AffectedSymbolCount     int
	TestCommandCount        int
}

type CostSummary struct {
	PackageCount             int                                   `json:"packageCount"`
	GoFileCount              int                                   `json:"goFileCount"`
	TestFileCount            int                                   `json:"testFileCount"`
	GeneratedFileCount       int                                   `json:"generatedFileCount"`
	SymbolCount              int                                   `json:"symbolCount"`
	RelationshipCount        int                                   `json:"relationshipCount"`
	RelationshipCounts       []snapshotstore.RelationshipKindCount `json:"relationshipCounts"`
	SkippedModuleCount       int                                   `json:"skippedModuleCount"`
	SkippedNestedModuleCount int                                   `json:"skippedNestedModuleCount"`
	SkippedWorkspaceCount    int                                   `json:"skippedWorkspaceCount"`
	LocalReplacementCount    int                                   `json:"localReplacementCount"`
	PackageLoadWarningCount  int                                   `json:"packageLoadWarningCount"`
	ChangedFileCount         int                                   `json:"changedFileCount"`
	ChangedPackageCount      int                                   `json:"changedPackageCount"`
	AffectedPackageCount     int                                   `json:"affectedPackageCount"`
	AffectedSymbolCount      int                                   `json:"affectedSymbolCount"`
	TestCommandCount         int                                   `json:"testCommandCount"`
	SnapshotStatus           string                                `json:"snapshotStatus"`
	SnapshotCountsAvailable  bool                                  `json:"snapshotCountsAvailable"`
	Limitations              []string                              `json:"limitations"`
}

func SummarizeCost(input CostInput) CostSummary {
	layout := sherpa.NormalizeRepositoryLayout(input.Layout)
	snapshot := normalizeInspect(input.Snapshot)

	summary := CostSummary{
		PackageCount:             positive(input.PackageCount),
		GoFileCount:              positive(layout.GoFiles),
		TestFileCount:            positive(layout.TestFiles),
		GeneratedFileCount:       positive(layout.GeneratedFiles),
		SymbolCount:              positive(snapshot.SymbolCount),
		RelationshipCount:        positive(snapshot.RelationshipMetadata.TotalCount),
		RelationshipCounts:       nonNilRelationshipCounts(snapshot.RelationshipMetadata.CountsByKind),
		SkippedNestedModuleCount: len(layout.SkippedNestedModules),
		SkippedWorkspaceCount:    len(layout.SkippedWorkspaceModules),
		LocalReplacementCount:    len(layout.LocalReplacements),
		PackageLoadWarningCount:  positive(input.PackageLoadWarningCount),
		ChangedFileCount:         positive(input.ChangedFileCount),
		ChangedPackageCount:      positive(input.ChangedPackageCount),
		AffectedPackageCount:     positive(input.AffectedPackageCount),
		AffectedSymbolCount:      positive(input.AffectedSymbolCount),
		TestCommandCount:         positive(input.TestCommandCount),
		SnapshotStatus:           snapshot.Status,
		SnapshotCountsAvailable:  snapshotHasInventoryCounts(snapshot),
	}
	summary.SkippedModuleCount = summary.SkippedNestedModuleCount + summary.SkippedWorkspaceCount
	summary.Limitations = costLimitations(summary, snapshot)

	return NormalizeCostSummary(summary)
}

func NormalizeCostSummary(summary CostSummary) CostSummary {
	summary.PackageCount = positive(summary.PackageCount)
	summary.GoFileCount = positive(summary.GoFileCount)
	summary.TestFileCount = positive(summary.TestFileCount)
	summary.GeneratedFileCount = positive(summary.GeneratedFileCount)
	summary.SymbolCount = positive(summary.SymbolCount)
	summary.RelationshipCount = positive(summary.RelationshipCount)
	summary.RelationshipCounts = nonNilRelationshipCounts(summary.RelationshipCounts)
	summary.SkippedModuleCount = positive(summary.SkippedModuleCount)
	summary.SkippedNestedModuleCount = positive(summary.SkippedNestedModuleCount)
	summary.SkippedWorkspaceCount = positive(summary.SkippedWorkspaceCount)
	summary.LocalReplacementCount = positive(summary.LocalReplacementCount)
	summary.PackageLoadWarningCount = positive(summary.PackageLoadWarningCount)
	summary.ChangedFileCount = positive(summary.ChangedFileCount)
	summary.ChangedPackageCount = positive(summary.ChangedPackageCount)
	summary.AffectedPackageCount = positive(summary.AffectedPackageCount)
	summary.AffectedSymbolCount = positive(summary.AffectedSymbolCount)
	summary.TestCommandCount = positive(summary.TestCommandCount)
	summary.SnapshotStatus = strings.TrimSpace(summary.SnapshotStatus)
	if summary.SnapshotStatus == "" {
		summary.SnapshotStatus = snapshotstore.StatusMissing
	}
	summary.Limitations = nonNilStrings(summary.Limitations)

	return summary
}

func normalizeInspect(inspect snapshotstore.InspectResult) snapshotstore.InspectResult {
	if strings.TrimSpace(inspect.Status) == "" {
		inspect.Status = snapshotstore.StatusMissing
	}
	return inspect
}

func snapshotHasInventoryCounts(snapshot snapshotstore.InspectResult) bool {
	return snapshot.FileCount > 0 || snapshot.PackageCount > 0 || snapshot.SymbolCount > 0 || snapshot.RelationshipMetadata.Present
}

func costLimitations(summary CostSummary, snapshot snapshotstore.InspectResult) []string {
	limitations := []string{
		"Cost counts describe analysis size and reuse hints; they are not wall-clock timings.",
		"Package and file counts follow the selected --root, go.work, nested-module, and build-tag boundaries.",
	}
	if !summary.SnapshotCountsAvailable {
		limitations = append(limitations, "Full symbol and relationship inventory counts require an existing readable snapshot.")
	}
	if snapshot.Status == snapshotstore.StatusStale {
		limitations = append(limitations, "Snapshot inventory counts come from stale snapshot metadata; refresh before relying on them for current repository size.")
	}
	if snapshot.Status == snapshotstore.StatusInvalid {
		limitations = append(limitations, "Snapshot inventory counts are unavailable because the snapshot is invalid.")
	}
	return limitations
}

func nonNilRelationshipCounts(values []snapshotstore.RelationshipKindCount) []snapshotstore.RelationshipKindCount {
	if values == nil {
		return []snapshotstore.RelationshipKindCount{}
	}
	return append([]snapshotstore.RelationshipKindCount{}, values...)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string{}, values...)
}

func positive(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
