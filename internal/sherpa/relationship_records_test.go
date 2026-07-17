package sherpa

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildReferenceRelationshipsWithOptionsIndexesDefinitionsAndCalls(t *testing.T) {
	root := t.TempDir()
	writeRelationshipTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeRelationshipTestFile(t, filepath.Join(root, "service.go"), `package service

func Entry() {
	Target()
}

func Target() {}
`)

	relationships, analysisMode, warnings, err := BuildReferenceRelationshipsWithOptions(root, ReferenceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if analysisMode != ReferenceAnalysisModeTypechecked {
		t.Fatalf("analysis mode = %q, want %q with warnings %#v", analysisMode, ReferenceAnalysisModeTypechecked, warnings)
	}

	var foundDefinition bool
	var foundCall bool
	for _, relationship := range relationships {
		if relationship.Target.Name != "Target" {
			continue
		}
		switch relationship.Kind {
		case ReferenceKindDefinition:
			foundDefinition = true
		case ReferenceKindCall:
			foundCall = true
			if relationship.Source.Name != "Entry" {
				t.Fatalf("call source = %#v, want Entry", relationship.Source)
			}
		}
	}

	if !foundDefinition || !foundCall {
		t.Fatalf("expected Target definition and call relationships, got %#v", relationships)
	}
}

func TestBuildCallRelationshipsWithOptionsIndexesDirectEdges(t *testing.T) {
	root := t.TempDir()
	writeRelationshipTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeRelationshipTestFile(t, filepath.Join(root, "service.go"), `package service

func Entry() {
	Target()
}

func Target() {}
`)

	relationships, analysisMode, warnings, err := BuildCallRelationshipsWithOptions(root, CallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if analysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("analysis mode = %q, want %q with warnings %#v", analysisMode, CallAnalysisModeTypechecked, warnings)
	}

	for _, relationship := range relationships {
		if relationship.Source.Name == "Entry" && relationship.Target.Name == "Target" {
			if relationship.Scope != CallScopeLocal {
				t.Fatalf("call scope = %q, want %q", relationship.Scope, CallScopeLocal)
			}
			return
		}
	}

	t.Fatalf("expected Entry -> Target call relationship, got %#v", relationships)
}

func TestBuildPossibleCallRelationshipsWithOptionsIndexesInterfaceDispatchEdges(t *testing.T) {
	root := t.TempDir()
	writeRelationshipTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeRelationshipTestFile(t, filepath.Join(root, "service.go"), `package service

type Processor interface { Process() }

type Worker struct{}

func (Worker) Process() {}

func Entry(processor Processor) {
	processor.Process()
}
`)

	relationships, analysisMode, warnings, err := BuildPossibleCallRelationshipsWithOptions(root, CallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if analysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("analysis mode = %q, want %q with warnings %#v", analysisMode, CallAnalysisModeTypechecked, warnings)
	}

	for _, relationship := range relationships {
		if relationship.Source.Name == "Entry" && relationship.Target.Receiver == "Worker" && relationship.Target.Name == "Process" {
			if relationship.Scope != CallScopeLocal {
				t.Fatalf("call scope = %q, want %q", relationship.Scope, CallScopeLocal)
			}
			if relationship.Reason != PossibleCallReasonInterfaceDispatch {
				t.Fatalf("reason = %q, want %q", relationship.Reason, PossibleCallReasonInterfaceDispatch)
			}
			if relationship.Position.File != "service.go" || relationship.Position.Line != 10 {
				t.Fatalf("expected call-site position service.go:10, got %#v", relationship.Position)
			}
			return
		}
	}

	t.Fatalf("expected Entry -> Worker.Process possible call relationship, got %#v", relationships)
}

func writeRelationshipTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
