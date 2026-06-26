package sherpa

import "testing"

func TestFormatSymbolWithContext(t *testing.T) {
	symbol := Symbol{
		Name: "Run",
		Kind: SymbolKindFunction,
		Position: Position{
			File: "service.go",
			Line: 3,
		},
	}
	context := SourceContext{
		Lines: []SourceContextLine{
			{Number: 2, Text: "func helper() {}"},
			{Number: 3, Text: "func Run() {", Target: true},
			{Number: 4, Text: "}"},
		},
	}

	got := FormatSymbolWithContext(symbol, context)
	want := `SYMBOL

Name: Run
Kind: function
File: service.go
Line: 3

CONTEXT
    2 | func helper() {}
  > 3 | func Run() {
    4 | }
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}
