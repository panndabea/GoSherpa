package sherpa

import "testing"

func TestFormatReferences(t *testing.T) {
	refs := []Reference{
		{
			Name: "ParseFile",
			Kind: ReferenceKindDefinition,
			Position: Position{
				File: "internal/sherpa/parse.go",
				Line: 11,
			},
		},
		{
			Name: "ParseFile",
			Kind: ReferenceKindCall,
			Position: Position{
				File: "internal/sherpa/repository.go",
				Line: 8,
			},
		},
	}

	got := FormatReferences("ParseFile", refs)
	want := `REFERENCES

ParseFile

  definition   internal/sherpa/parse.go:11
  call         internal/sherpa/repository.go:8

Found 2 references
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatReferencesWithContext(t *testing.T) {
	refs := []Reference{
		{
			Name: "Run",
			Kind: ReferenceKindDefinition,
			Position: Position{
				File: "service.go",
				Line: 3,
			},
		},
		{
			Name: "Run",
			Kind: ReferenceKindCall,
			Position: Position{
				File: "service.go",
				Line: 7,
			},
		},
	}
	contexts := []SourceContext{
		{
			Lines: []SourceContextLine{
				{Number: 2, Text: "func helper() {}"},
				{Number: 3, Text: "func Run() {", Target: true},
				{Number: 4, Text: "}"},
			},
		},
		{
			Lines: []SourceContextLine{
				{Number: 6, Text: "func caller() {"},
				{Number: 7, Text: "\tRun()", Target: true},
				{Number: 8, Text: "}"},
			},
		},
	}

	got := FormatReferencesWithContext("Run", refs, contexts)
	want := `REFERENCES

Run

  definition   service.go:3
      2 | func helper() {}
    > 3 | func Run() {
      4 | }

  call         service.go:7
      6 | func caller() {
    > 7 | 	Run()
      8 | }

Found 2 references
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatReferenceReportShowsAnalysisModeAndWarnings(t *testing.T) {
	report := ReferenceReport{
		Target:       "Run",
		AnalysisMode: ReferenceAnalysisModeASTFallback,
		Warnings:     []string{"typechecked reference analysis unavailable: loader failed"},
		References: []Reference{
			{
				Name: "Run",
				Kind: ReferenceKindDefinition,
				Position: Position{
					File: "service.go",
					Line: 3,
				},
			},
		},
	}

	got := FormatReferenceReport(report)
	want := `REFERENCES

Run
analysisMode: "ast-fallback"

  definition   service.go:3

WARNINGS
  typechecked reference analysis unavailable: loader failed

Found 1 references
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatReferencesWithEmptyList(t *testing.T) {
	got := FormatReferences("Missing", nil)
	want := `REFERENCES

Missing

  none

Found 0 references
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
