package sherpa

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReadSourceContextReturnsRootRelativeSnippet(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "internal", "service", "service.go")
	writeSourceContextTestFile(t, path, "package service\n\nfunc helper() {}\nfunc Run() {}\nfunc done() {}\n")

	context, err := ReadSourceContext(tmp, Position{
		File: "internal/service/service.go",
		Line: 4,
	}, 1)
	if err != nil {
		t.Fatal(err)
	}

	if context.Position.File != "internal/service/service.go" {
		t.Fatalf("expected root-relative file, got %s", context.Position.File)
	}

	want := []SourceContextLine{
		{Number: 3, Text: "func helper() {}"},
		{Number: 4, Text: "func Run() {}", Target: true},
		{Number: 5, Text: "func done() {}"},
	}
	if !reflect.DeepEqual(context.Lines, want) {
		t.Fatalf("expected %#v, got %#v", want, context.Lines)
	}
}

func TestReadSourceContextRejectsLineOutsideFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "service.go")
	writeSourceContextTestFile(t, path, "package service\n")

	_, err := ReadSourceContext(tmp, Position{
		File: "service.go",
		Line: 3,
	}, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatSourceContextMarksTargetLine(t *testing.T) {
	context := SourceContext{
		Lines: []SourceContextLine{
			{Number: 9, Text: "func helper() {}"},
			{Number: 10, Text: "func Run() {}", Target: true},
			{Number: 11, Text: "func done() {}"},
		},
	}

	got := FormatSourceContext(context, "  ")
	want := `     9 | func helper() {}
  > 10 | func Run() {}
    11 | func done() {}
`

	if got != want {
		t.Fatalf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func writeSourceContextTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
