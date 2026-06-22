package sherpa

import "testing"

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

func TestFormatCalleesWithEmptyList(t *testing.T) {
	result := CalleesResult{Target: "Empty"}

	got := FormatCallees(result)
	want := "no callees found: Empty\n"

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

func TestFormatCallersWithEmptyList(t *testing.T) {
	result := CallersResult{Target: "Empty"}

	got := FormatCallers(result)
	want := "no callers found: Empty\n"

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
