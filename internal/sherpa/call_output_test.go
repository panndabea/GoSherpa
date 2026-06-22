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
