package sherpa

import (
	"fmt"
	"testing"
)

func TestFormatCallees(t *testing.T) {
	result := CalleesResult{
		Target: "ParseFile",
		Callees: []Callee{
			{
				Name: "parser.ParseFile",
				Position: Position{
					File: "internal/sherpa/parse.go",
					Line: 11,
				},
			},
			{
				Name: "fileSet.Position",
				Position: Position{
					File: "internal/sherpa/parse.go",
					Line: 18,
				},
			},
		},
	}

	got := FormatCallees(result)
	want := `CALLEES

ParseFile

  parser.ParseFile                     internal/sherpa/parse.go:11
  fileSet.Position                     internal/sherpa/parse.go:18

Found 2 callees
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCalleesWithContext(t *testing.T) {
	result := CalleesResult{
		Target: "Run",
		Callees: []Callee{
			{
				Name: "Step",
				Position: Position{
					File: "service.go",
					Line: 4,
				},
			},
		},
	}
	contexts := []SourceContext{
		{
			Lines: []SourceContextLine{
				{Number: 3, Text: "func Run() {"},
				{Number: 4, Text: "\tStep()", Target: true},
				{Number: 5, Text: "}"},
			},
		},
	}

	got := FormatCalleesWithContext(result, contexts)
	want := fmt.Sprintf(`CALLEES

Run

  %-36s service.go:4
      3 | func Run() {
    > 4 | 	Step()
      5 | }

Found 1 callees
`, "Step")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCalleesGroupsLocalBeforeExternalBuiltinAndDynamic(t *testing.T) {
	result := CalleesResult{
		Target: "Run",
		Callees: []Callee{
			{
				Name:  "fmt.Println",
				Scope: CallScopeExternal,
				Position: Position{
					File: "service.go",
					Line: 5,
				},
			},
			{
				Name:  "Step",
				Scope: CallScopeLocal,
				Position: Position{
					File: "service.go",
					Line: 4,
				},
			},
			{
				Name:  "append",
				Scope: CallScopeBuiltin,
				Position: Position{
					File: "service.go",
					Line: 6,
				},
			},
			{
				Name:  "callback",
				Scope: CallScopeDynamic,
				Position: Position{
					File: "service.go",
					Line: 7,
				},
			},
		},
	}
	contexts := []SourceContext{
		{
			Lines: []SourceContextLine{
				{Number: 5, Text: "\tfmt.Println(\"ready\")", Target: true},
			},
		},
		{
			Lines: []SourceContextLine{
				{Number: 4, Text: "\tStep()", Target: true},
			},
		},
		{
			Lines: []SourceContextLine{
				{Number: 6, Text: "\t_ = append([]int{}, 1)", Target: true},
			},
		},
		{
			Lines: []SourceContextLine{
				{Number: 7, Text: "\tcallback()", Target: true},
			},
		},
	}

	got := FormatCalleesWithContext(result, contexts)
	want := fmt.Sprintf(`CALLEES

Run

LOCAL
  %-36s service.go:4
    > 4 | 	Step()

EXTERNAL / BUILTIN / DYNAMIC
  %-36s service.go:5
    > 5 | 	fmt.Println("ready")

  %-36s service.go:6
    > 6 | 	_ = append([]int{}, 1)

  %-36s service.go:7
    > 7 | 	callback()

Found 4 callees
`, "Step", "fmt.Println", "append", "callback")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCalleesWithEmptyList(t *testing.T) {
	result := CalleesResult{Target: "Empty"}

	got := FormatCallees(result)
	want := "no callees found: Empty\n"

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCalleesShowsAnalysisModeAndWarnings(t *testing.T) {
	result := CalleesResult{
		Target:       "Run",
		AnalysisMode: CallAnalysisModeASTFallback,
		Warnings:     []string{"typechecked call analysis unavailable: loader failed"},
		Callees: []Callee{
			{
				Name: "Step",
				Position: Position{
					File: "service.go",
					Line: 4,
				},
			},
		},
	}

	got := FormatCallees(result)
	want := fmt.Sprintf(`CALLEES

Run
Analysis: ast-fallback

  %-36s service.go:4

WARNINGS
  typechecked call analysis unavailable: loader failed

Found 1 callees
`, "Step")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCalleesShowsLimitations(t *testing.T) {
	result := CalleesResult{
		Target:       "Run",
		AnalysisMode: CallAnalysisModeTypechecked,
		Limitations: []string{
			"Function value calls may hide concrete call edges at service.go:9.",
		},
		Callees: []Callee{
			{
				Name: "callback",
				Position: Position{
					File: "service.go",
					Line: 9,
				},
			},
		},
	}

	got := FormatCallees(result)
	want := fmt.Sprintf(`CALLEES

Run
Analysis: typechecked

  %-36s service.go:9

LIMITATIONS
  Function value calls may hide concrete call edges at service.go:9.

Found 1 callees
`, "callback")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCallers(t *testing.T) {
	result := CallersResult{
		Target: "ParseFile",
		Callers: []Caller{
			{
				Name: "Load",
				Position: Position{
					File: "internal/sherpa/repository.go",
					Line: 12,
				},
			},
			{
				Name: "Run",
				Position: Position{
					File: "cmd/gosherpa/main.go",
					Line: 70,
				},
			},
		},
	}

	got := FormatCallers(result)
	want := `CALLERS

ParseFile

  Load                                 internal/sherpa/repository.go:12
  Run                                  cmd/gosherpa/main.go:70

Found 2 callers
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCallersWithContext(t *testing.T) {
	result := CallersResult{
		Target: "Step",
		Callers: []Caller{
			{
				Name: "Run",
				Position: Position{
					File: "service.go",
					Line: 4,
				},
			},
		},
	}
	contexts := []SourceContext{
		{
			Lines: []SourceContextLine{
				{Number: 3, Text: "func Run() {"},
				{Number: 4, Text: "\tStep()", Target: true},
				{Number: 5, Text: "}"},
			},
		},
	}

	got := FormatCallersWithContext(result, contexts)
	want := fmt.Sprintf(`CALLERS

Step

  %-36s service.go:4
      3 | func Run() {
    > 4 | 	Step()
      5 | }

Found 1 callers
`, "Run")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCallersWithEmptyList(t *testing.T) {
	result := CallersResult{Target: "Empty"}

	got := FormatCallers(result)
	want := "no callers found: Empty\n"

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCallPaths(t *testing.T) {
	result := CallPathsResult{
		From: "Entry",
		To:   "Target",
		Paths: []CallPath{
			{
				Steps: []CallPathStep{
					{
						Caller: "Entry",
						Callee: "Middle",
						Position: Position{
							File: "service.go",
							Line: 4,
						},
					},
					{
						Caller: "Middle",
						Callee: "Target",
						Position: Position{
							File: "service.go",
							Line: 8,
						},
					},
				},
			},
		},
	}

	got := FormatCallPaths(result)
	want := fmt.Sprintf(`CALL PATH

Entry
  -> %-36s service.go:4
  -> %-36s service.go:8

Found 1 path
`, "Middle", "Target")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCallPathsWithEmptyList(t *testing.T) {
	result := CallPathsResult{
		From: "Entry",
		To:   "Target",
	}

	got := FormatCallPaths(result)
	want := "no call path found: Entry -> Target\n"

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatCallPathsWithMultiplePaths(t *testing.T) {
	result := CallPathsResult{
		From: "Entry",
		To:   "Target",
		Paths: []CallPath{
			{
				Steps: []CallPathStep{
					{
						Caller: "Entry",
						Callee: "First",
						Position: Position{
							File: "service.go",
							Line: 4,
						},
					},
					{
						Caller: "First",
						Callee: "Target",
						Position: Position{
							File: "service.go",
							Line: 8,
						},
					},
				},
			},
			{
				Steps: []CallPathStep{
					{
						Caller: "Entry",
						Callee: "Second",
						Position: Position{
							File: "service.go",
							Line: 5,
						},
					},
					{
						Caller: "Second",
						Callee: "Target",
						Position: Position{
							File: "service.go",
							Line: 12,
						},
					},
				},
			},
		},
	}

	got := FormatCallPaths(result)
	want := fmt.Sprintf(`CALL PATHS

Entry -> Target

Path 1
  Entry
    -> %-36s service.go:4
    -> %-36s service.go:8

Path 2
  Entry
    -> %-36s service.go:5
    -> %-36s service.go:12

Found 2 paths
`, "First", "Target", "Second", "Target")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatEntryPoints(t *testing.T) {
	result := EntryPointsResult{
		Target:       "Target",
		AnalysisMode: CallAnalysisModeTypechecked,
		EntryPoints: []EntryPoint{
			{
				Name:    "main",
				Package: "./cmd/app",
				Kind:    EntryPointKindMain,
				Position: Position{
					File: "cmd/app/main.go",
					Line: 5,
				},
			},
			{
				Name:    "Entry",
				Package: "./internal/service",
				Kind:    EntryPointKindExported,
				Position: Position{
					File: "internal/service/service.go",
					Line: 3,
				},
			},
		},
	}

	got := FormatEntryPoints(result)
	want := fmt.Sprintf(`ENTRYPOINTS

Target: Target
Analysis: typechecked

  %-17s %-9s %-36s %-20s cmd/app/main.go:5
  %-17s %-9s %-36s %-20s internal/service/service.go:3

Found 2 entrypoints
`, EntryPointKindMain, CallCertaintyDirect, "main", "./cmd/app", EntryPointKindExported, CallCertaintyDirect, "Entry", "./internal/service")

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatEntryPointsWithEmptyList(t *testing.T) {
	result := EntryPointsResult{Target: "Target"}

	got := FormatEntryPoints(result)
	want := "no entrypoints found: Target\n"

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
