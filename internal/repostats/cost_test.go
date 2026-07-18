package repostats

import (
	"testing"

	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

func TestSummarizeCostCombinesRepositoryAndSnapshotCounts(t *testing.T) {
	summary := SummarizeCost(CostInput{
		Layout: sherpa.RepositoryLayout{
			GoFiles:                 12,
			TestFiles:               3,
			GeneratedFiles:          2,
			SkippedNestedModules:    []string{"nested"},
			SkippedWorkspaceModules: []string{"../external"},
			LocalReplacements:       []sherpa.LocalReplacement{{ModulePath: "example.com/lib"}},
		},
		PackageCount:            4,
		PackageLoadWarningCount: 1,
		Snapshot: snapshotstore.InspectResult{
			Status:       snapshotstore.StatusValid,
			FileCount:    15,
			PackageCount: 4,
			SymbolCount:  27,
			RelationshipMetadata: snapshotstore.RelationshipMetadata{
				Present:    true,
				Capable:    true,
				TotalCount: 31,
				CountsByKind: []snapshotstore.RelationshipKindCount{
					{Kind: "call", Count: 5},
				},
			},
		},
		ChangedFileCount:     2,
		ChangedPackageCount:  1,
		AffectedPackageCount: 3,
		AffectedSymbolCount:  8,
		TestCommandCount:     2,
	})

	if summary.PackageCount != 4 || summary.GoFileCount != 12 || summary.GeneratedFileCount != 2 {
		t.Fatalf("unexpected repository counts: %#v", summary)
	}
	if summary.SymbolCount != 27 || summary.RelationshipCount != 31 || !summary.SnapshotCountsAvailable {
		t.Fatalf("unexpected snapshot counts: %#v", summary)
	}
	if summary.SkippedModuleCount != 2 || summary.SkippedNestedModuleCount != 1 || summary.SkippedWorkspaceCount != 1 {
		t.Fatalf("unexpected skipped module counts: %#v", summary)
	}
	if len(summary.RelationshipCounts) != 1 || summary.RelationshipCounts[0].Kind != "call" {
		t.Fatalf("expected relationship counts, got %#v", summary.RelationshipCounts)
	}
	if summary.Limitations == nil {
		t.Fatalf("expected non-nil limitations")
	}
}

func TestSummarizeCostReportsMissingSnapshotInventoryLimit(t *testing.T) {
	summary := SummarizeCost(CostInput{
		Layout:       sherpa.RepositoryLayout{GoFiles: 1},
		PackageCount: 1,
		Snapshot:     snapshotstore.InspectResult{Status: snapshotstore.StatusMissing},
	})

	if summary.RelationshipCounts == nil || summary.Limitations == nil {
		t.Fatalf("expected non-nil arrays, got %#v", summary)
	}
	if summary.SnapshotCountsAvailable {
		t.Fatalf("missing snapshot should not expose inventory counts: %#v", summary)
	}
	if summary.SymbolCount != 0 || summary.RelationshipCount != 0 {
		t.Fatalf("expected zero unavailable inventory counts, got %#v", summary)
	}
	if !costTestContains(summary.Limitations, "Full symbol and relationship inventory counts require an existing readable snapshot.") {
		t.Fatalf("expected missing snapshot inventory limitation, got %#v", summary.Limitations)
	}
}

func TestSummarizeCostWarnsAboutStaleSnapshotCounts(t *testing.T) {
	summary := SummarizeCost(CostInput{
		Layout:       sherpa.RepositoryLayout{GoFiles: 1},
		PackageCount: 1,
		Snapshot: snapshotstore.InspectResult{
			Status:      snapshotstore.StatusStale,
			SymbolCount: 3,
		},
	})

	if !summary.SnapshotCountsAvailable {
		t.Fatalf("stale snapshots can still expose stored inventory counts: %#v", summary)
	}
	if !costTestContains(summary.Limitations, "Snapshot inventory counts come from stale snapshot metadata; refresh before relying on them for current repository size.") {
		t.Fatalf("expected stale snapshot limitation, got %#v", summary.Limitations)
	}
}

func costTestContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
