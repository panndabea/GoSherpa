package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/panndabea/GoSherpa/internal/sherpa"
	"github.com/panndabea/GoSherpa/internal/symbolindex"
)

func TestLoadReusableLoadsValidRelationshipSnapshot(t *testing.T) {
	root := writeSnapshotTestProject(t)
	serviceFile := filepath.Join(root, "service.go")

	built, err := Build(root, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	built.Relationships = symbolindex.RelationshipIndex{
		References: []symbolindex.ReferenceRecord{
			{
				Package: ".",
				File:    serviceFile,
				Target: symbolindex.SymbolIdentity{
					Package: ".",
					Name:    "Target",
				},
				ReferenceKind: sherpa.ReferenceKindCall,
				Position: sherpa.Position{
					File:   serviceFile,
					Line:   4,
					Column: 2,
				},
			},
		},
	}

	if _, err := Write(root, built); err != nil {
		t.Fatal(err)
	}

	loaded, inspect := LoadReusable(root, BuildOptions{})
	if inspect.Status != StatusValid {
		t.Fatalf("expected valid snapshot, got %#v", inspect)
	}
	if loaded.FormatVersion != FormatVersion {
		t.Fatalf("format version = %d, want %d", loaded.FormatVersion, FormatVersion)
	}
	if !loaded.RelationshipMetadata.Present || !loaded.RelationshipMetadata.Capable {
		t.Fatalf("expected relationship-capable metadata, got %#v", loaded.RelationshipMetadata)
	}
	if got := relationshipCount(t, loaded.RelationshipMetadata, string(symbolindex.RelationshipKindReference)); got != 1 {
		t.Fatalf("reference count = %d, want 1", got)
	}
	if got := relationshipCount(t, loaded.RelationshipMetadata, string(symbolindex.RelationshipKindSymbolDefinition)); got != 2 {
		t.Fatalf("symbol-definition count = %d, want 2", got)
	}
	if len(loaded.Relationships.References) != 1 {
		t.Fatalf("expected one reference, got %#v", loaded.Relationships.References)
	}
	reference := loaded.Relationships.References[0]
	if reference.File != "service.go" || reference.Position.File != "service.go" {
		t.Fatalf("expected relationship paths to be root-relative, got %#v", reference)
	}
	if !inspect.RelationshipMetadata.Capable {
		t.Fatalf("inspect should expose relationship capability, got %#v", inspect.RelationshipMetadata)
	}
}

func TestBuildPersistsPossibleCallRelationshipMetadata(t *testing.T) {
	root := t.TempDir()
	writeSnapshotTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeSnapshotTestFile(t, filepath.Join(root, "service.go"), `package app

type Processor interface { Process() }

type Worker struct{}

func (Worker) Process() {}

func Entry(processor Processor) {
	processor.Process()
}
`)

	built, err := Build(root, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(built.Relationships.PossibleCallEdges) != 1 {
		t.Fatalf("expected one possible call edge, got %#v", built.Relationships.PossibleCallEdges)
	}
	edge := built.Relationships.PossibleCallEdges[0]
	if edge.Target.Receiver != "Worker" || edge.Target.Name != "Process" {
		t.Fatalf("expected Worker.Process possible call edge, got %#v", edge)
	}
	if edge.Reason != string(sherpa.PossibleCallReasonInterfaceDispatch) {
		t.Fatalf("reason = %q, want %q", edge.Reason, sherpa.PossibleCallReasonInterfaceDispatch)
	}
	if got := relationshipCount(t, built.RelationshipMetadata, string(symbolindex.RelationshipKindPossibleCall)); got != 1 {
		t.Fatalf("possible-call count = %d, want 1", got)
	}
}

func TestLoadReusableReportsStaleRelationshipSnapshot(t *testing.T) {
	root := writeSnapshotTestProject(t)
	built, err := Build(root, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, built); err != nil {
		t.Fatal(err)
	}

	writeSnapshotTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() {
	Target()
}

func Target() {}

func Added() {}
`)

	_, inspect := LoadReusable(root, BuildOptions{})
	if inspect.Status != StatusStale {
		t.Fatalf("expected stale snapshot, got %#v", inspect)
	}
	if !containsString(inspect.StaleReasons, "repository files changed") {
		t.Fatalf("expected repository file stale reason, got %#v", inspect.StaleReasons)
	}
	if !inspect.RelationshipMetadata.Capable {
		t.Fatalf("stale diagnostics should retain stored relationship metadata, got %#v", inspect.RelationshipMetadata)
	}
}

func TestLoadReusableReportsBuildTagMismatchForRelationshipSnapshot(t *testing.T) {
	root := writeSnapshotTestProject(t)
	built, err := Build(root, BuildOptions{BuildTags: []string{"enterprise"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write(root, built); err != nil {
		t.Fatal(err)
	}

	_, inspect := LoadReusable(root, BuildOptions{})
	if inspect.Status != StatusStale {
		t.Fatalf("expected stale snapshot, got %#v", inspect)
	}
	if !containsString(inspect.StaleReasons, "build tags changed") {
		t.Fatalf("expected build tag stale reason, got %#v", inspect.StaleReasons)
	}
}

func TestLoadReusableReportsMalformedRelationshipSnapshot(t *testing.T) {
	root := writeSnapshotTestProject(t)
	if err := os.MkdirAll(filepath.Dir(Path(root)), 0755); err != nil {
		t.Fatal(err)
	}
	writeSnapshotTestFile(t, Path(root), `{
  "formatVersion": 2,
  "relationshipMetadata": {
    "present": true,
    "capable": true
  },
  "relationships": "not-an-object"
}
`)

	_, inspect := LoadReusable(root, BuildOptions{})
	if inspect.Status != StatusInvalid {
		t.Fatalf("expected invalid snapshot, got %#v", inspect)
	}
	if !strings.Contains(inspect.Message, "Snapshot could not be read") {
		t.Fatalf("expected read failure message, got %q", inspect.Message)
	}
	if !containsString(inspect.StaleReasons, "snapshot could not be read") {
		t.Fatalf("expected read stale reason, got %#v", inspect.StaleReasons)
	}
}

func TestLoadReusableReportsLegacyInventorySnapshotAsStale(t *testing.T) {
	root := writeSnapshotTestProject(t)
	built, err := Build(root, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	built.FormatVersion = legacyFormatVersion
	if _, err := Write(root, built); err != nil {
		t.Fatal(err)
	}

	loaded, inspect := LoadReusable(root, BuildOptions{})
	if loaded.FormatVersion != legacyFormatVersion {
		t.Fatalf("loaded format version = %d, want %d", loaded.FormatVersion, legacyFormatVersion)
	}
	if inspect.Status != StatusStale {
		t.Fatalf("expected stale legacy snapshot, got %#v", inspect)
	}
	if !containsString(inspect.StaleReasons, "snapshot format version changed") {
		t.Fatalf("expected format stale reason, got %#v", inspect.StaleReasons)
	}
	if inspect.RelationshipMetadata.Capable {
		t.Fatalf("legacy snapshot must not be relationship-capable, got %#v", inspect.RelationshipMetadata)
	}
}

func relationshipCount(t *testing.T, metadata RelationshipMetadata, kind string) int {
	t.Helper()
	for _, count := range metadata.CountsByKind {
		if count.Kind == kind {
			return count.Count
		}
	}
	t.Fatalf("missing relationship count for %s in %#v", kind, metadata.CountsByKind)
	return 0
}

func writeSnapshotTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeSnapshotTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeSnapshotTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() {
	Target()
}

func Target() {}
`)
	return root
}

func writeSnapshotTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
