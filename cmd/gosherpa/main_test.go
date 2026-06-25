package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCLIArgsDefaultsRootToCurrentDirectory(t *testing.T) {
	got, err := parseCLIArgs([]string{"symbols"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Root != "." {
		t.Fatalf("expected root ., got %s", got.Root)
	}

	if got.Command != "symbols" {
		t.Fatalf("expected command symbols, got %s", got.Command)
	}

	if len(got.CommandArgs) != 0 {
		t.Fatalf("expected no command args, got %v", got.CommandArgs)
	}
}

func TestParseCLIArgsAcceptsRootBeforeCommand(t *testing.T) {
	got, err := parseCLIArgs([]string{"--root", "/repo", "symbols"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Root != "/repo" {
		t.Fatalf("expected root /repo, got %s", got.Root)
	}

	if got.Command != "symbols" {
		t.Fatalf("expected command symbols, got %s", got.Command)
	}
}

func TestParseCLIArgsAcceptsRootAfterCommand(t *testing.T) {
	got, err := parseCLIArgs([]string{"symbols", "--root", "/repo"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Root != "/repo" {
		t.Fatalf("expected root /repo, got %s", got.Root)
	}

	if got.Command != "symbols" {
		t.Fatalf("expected command symbols, got %s", got.Command)
	}
}

func TestParseCLIArgsAcceptsRootAfterCommandArgument(t *testing.T) {
	got, err := parseCLIArgs([]string{"refs", "ParseFile", "--root", "/repo"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Root != "/repo" {
		t.Fatalf("expected root /repo, got %s", got.Root)
	}

	if got.Command != "refs" {
		t.Fatalf("expected command refs, got %s", got.Command)
	}

	assertMainTestStrings(t, got.CommandArgs, []string{"ParseFile"})
}

func TestParseCLIArgsAcceptsRootEqualsForm(t *testing.T) {
	got, err := parseCLIArgs([]string{"--root=/repo", "symbols"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Root != "/repo" {
		t.Fatalf("expected root /repo, got %s", got.Root)
	}

	if got.Command != "symbols" {
		t.Fatalf("expected command symbols, got %s", got.Command)
	}
}

func TestParseCLIArgsRejectsMissingRootValue(t *testing.T) {
	tests := [][]string{
		{"--root"},
		{"--root="},
		{"--root", "   "},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			_, err := parseCLIArgs(test)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), "missing value for --root") {
				t.Fatalf("expected missing root value error, got %v", err)
			}
		})
	}
}

func TestParseCLIArgsRejectsUnknownFlag(t *testing.T) {
	_, err := parseCLIArgs([]string{"--json", "symbols"})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "unknown flag: --json") {
		t.Fatalf("expected unknown flag error, got %v", err)
	}
}

func TestParseCLIArgsLastRootWins(t *testing.T) {
	got, err := parseCLIArgs([]string{"--root", "/one", "symbols", "--root=/two"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Root != "/two" {
		t.Fatalf("expected root /two, got %s", got.Root)
	}
}

func TestParseCLIArgsAcceptsCallPathOptions(t *testing.T) {
	got, err := parseCLIArgs([]string{"paths", "Entry", "Target", "--limit", "3", "--max-depth=4"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Command != "paths" {
		t.Fatalf("expected command paths, got %s", got.Command)
	}

	assertMainTestStrings(t, got.CommandArgs, []string{"Entry", "Target"})

	if got.CallPathLimit != 3 {
		t.Fatalf("expected limit 3, got %d", got.CallPathLimit)
	}

	if got.CallPathMaxDepth != 4 {
		t.Fatalf("expected max depth 4, got %d", got.CallPathMaxDepth)
	}

	if !got.HasCallPathOption {
		t.Fatal("expected call path option marker")
	}
}

func TestParseCLIArgsRejectsInvalidCallPathOptions(t *testing.T) {
	tests := [][]string{
		{"paths", "Entry", "Target", "--limit"},
		{"paths", "Entry", "Target", "--limit="},
		{"paths", "Entry", "Target", "--limit", "0"},
		{"paths", "Entry", "Target", "--max-depth", "nope"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			_, err := parseCLIArgs(test)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPrintUsageIncludesRoot(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	for _, want := range []string{
		"usage: gosherpa [--root <path>] <command> [args]",
		"--root <path>    repository root, defaults to .",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output)
		}
	}
}

func TestPrintUsageIncludesPathCommands(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	for _, want := range []string{
		"path <from> <to>",
		"paths <from> <to> [--limit <n>] [--max-depth <n>]",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output)
		}
	}
}

func TestPrintUsageIncludesImpact(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	if !strings.Contains(output, "impact <symbol-or-package>") {
		t.Fatalf("expected usage to contain impact command, got:\n%s", output)
	}
}

func TestPrintUsageIncludesTests(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	if !strings.Contains(output, "tests <symbol-or-package>") {
		t.Fatalf("expected usage to contain tests command, got:\n%s", output)
	}
}

func TestPrintUsageIncludesCallees(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	if !strings.Contains(output, "callees <function-or-method>") {
		t.Fatalf("expected usage to contain callees command, got:\n%s", output)
	}
}

func TestPrintUsageIncludesCallers(t *testing.T) {
	output := captureMainTestStdout(t, func() {
		printUsage()
	})

	if !strings.Contains(output, "callers <function-or-method>") {
		t.Fatalf("expected usage to contain callers command, got:\n%s", output)
	}
}

func TestMainPrintsTestsUsageWhenArgumentIsMissing(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "tests"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa [--root <path>] tests <symbol-or-package>\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainRunsTestsCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "parser", "parser_test.go"), `package parser

import "testing"

func TestParserPackage(t *testing.T) {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "cmd", "app", "main_test.go"), `package main

import (
	"testing"

	"example.com/app/internal/parser"
)

func TestUsesParser(t *testing.T) {
	parser.ParseFile()
}
`)

	setMainTestArgs(t, []string{"gosherpa", "--root", tmp, "tests", "ParseFile"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"TESTS", "ParseFile (symbol)", "RELATED TESTS", "TestParserPackage", "TestUsesParser", "SUGGESTED COMMANDS", "go test ./cmd/app", "go test ./internal/parser"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainPrintsImpactUsageWhenArgumentIsMissing(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "impact"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa [--root <path>] impact <symbol-or-package>\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainRunsImpactCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/parser"

func Run() {
	parser.ParseFile()
}
`)

	setMainTestArgs(t, []string{"gosherpa", "--root", tmp, "impact", "ParseFile"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"IMPACT", "ParseFile (symbol)", "REFERENCES", "DIRECT CALLERS", "AFFECTED PACKAGES", "./cmd/app", "./internal/parser"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainPrintsPathUsageWhenArgumentIsMissing(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "path", "Entry"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa [--root <path>] path <from> <to> [--limit <n>] [--max-depth <n>]\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainRunsPathCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {
	Middle()
}

func Middle() {
	Target()
}

func Target() {}
`)

	setMainTestArgs(t, []string{"gosherpa", "--root", tmp, "path", "Entry", "Target"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"CALL PATH", "Entry", "Middle", "Target", "Found 1 path"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainRunsPathsCommandWithLimit(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {
	First()
	Second()
}

func First() {
	Target()
}

func Second() {
	Target()
}

func Target() {}
`)

	setMainTestArgs(t, []string{"gosherpa", "paths", "Entry", "Target", "--limit", "2", "--root", tmp})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"CALL PATHS", "Path 1", "Path 2", "First", "Second", "Found 2 paths"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainPrintsCallersUsageWhenArgumentIsMissing(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "callers"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa [--root <path>] callers <function-or-method>\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainRunsCallersCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	setMainTestArgs(t, []string{"gosherpa", "--root", tmp, "callers", "Step"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"CALLERS", "Step", "Run", "Found 1 callers"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainPrintsCalleesUsageWhenArgumentIsMissing(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "callees"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa [--root <path>] callees <function-or-method>\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainRunsCalleesCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	setMainTestArgs(t, []string{"gosherpa", "callees", "Run", "--root", tmp})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"CALLEES", "Run", "Step", "Found 1 callees"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainPrintsRefsUsageWithoutValidatingRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	setMainTestArgs(t, []string{"gosherpa", "--root", missingRoot, "refs"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	want := "usage: gosherpa [--root <path>] refs <name>\n"
	if output != want {
		t.Fatalf("expected %q, got %q", want, output)
	}
}

func TestMainPrintsUnknownCommandWithoutValidatingRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	setMainTestArgs(t, []string{"gosherpa", "--root", missingRoot, "unknown"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{
		"unknown command: unknown",
		"usage: gosherpa [--root <path>] <command> [args]",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, "error: repository root") {
		t.Fatalf("expected no root validation, got:\n%s", output)
	}
}

func TestMainRunsSymbolsWithRootBeforeCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Run() {}
`)

	setMainTestArgs(t, []string{"gosherpa", "--root", tmp, "symbols"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"Run", "internal/service/service.go"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainRunsRefsWithRootAfterCommandArgument(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func ParseFile() {
}

func Run() {
	ParseFile()
}
`)

	setMainTestArgs(t, []string{"gosherpa", "refs", "ParseFile", "--root", tmp})

	output := captureMainTestStdout(t, func() {
		main()
	})

	for _, want := range []string{"ParseFile", "internal/service/service.go", "Found 2 references"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, output)
		}
	}

	if strings.Contains(output, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", output)
	}
}

func TestMainPrintsErrorWhenRootIsMissing(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	setMainTestArgs(t, []string{"gosherpa", "--root", missingRoot, "symbols"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	if !strings.Contains(output, "error: repository root does not exist") {
		t.Fatalf("expected missing root error, got:\n%s", output)
	}
}

func TestMainPrintsErrorWhenRootHasNoGoMod(t *testing.T) {
	tmp := t.TempDir()
	setMainTestArgs(t, []string{"gosherpa", "--root", tmp, "symbols"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	if !strings.Contains(output, "error: repository root does not contain go.mod") {
		t.Fatalf("expected missing go.mod error, got:\n%s", output)
	}
}

func TestMainPrintsErrorForUnknownFlag(t *testing.T) {
	setMainTestArgs(t, []string{"gosherpa", "--json", "symbols"})

	output := captureMainTestStdout(t, func() {
		main()
	})

	if !strings.Contains(output, "error: unknown flag: --json") {
		t.Fatalf("expected unknown flag error, got:\n%s", output)
	}
}

func captureMainTestStdout(t *testing.T, run func()) string {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Stdout = oldStdout
	})

	os.Stdout = writePipe

	run()

	err = writePipe.Close()
	if err != nil {
		t.Fatal(err)
	}

	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatal(err)
	}

	err = readPipe.Close()
	if err != nil {
		t.Fatal(err)
	}

	return string(output)
}

func setMainTestArgs(t *testing.T, args []string) {
	t.Helper()

	oldArgs := os.Args
	os.Args = args
	t.Cleanup(func() {
		os.Args = oldArgs
	})
}

func writeMainTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func assertMainTestStrings(t *testing.T, got []string, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
