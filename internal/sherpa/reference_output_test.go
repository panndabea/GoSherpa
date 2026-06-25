package sherpa

import "testing"

func TestFormatReferences(t *testing.T) {
	refs := []Reference{
		{
			Name: "ParseFile",
			Position: Position{
				File: "internal/sherpa/parse.go",
				Line: 11,
			},
		},
		{
			Name: "ParseFile",
			Position: Position{
				File: "internal/sherpa/repository.go",
				Line: 8,
			},
		},
	}

	got := FormatReferences("ParseFile", refs)
	want := `REFERENCES

ParseFile

  internal/sherpa/parse.go:11
  internal/sherpa/repository.go:8

Found 2 references
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
