package main

import (
	"bytes"
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
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{
		"usage: gosherpa [--root <path>] <command> [args]",
		"--root <path>    repository root, defaults to .",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesPathCommands(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{
		"path <from> <to>",
		"paths <from> <to> [--limit <n>] [--max-depth <n>]",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesImpact(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "impact <symbol-or-package>") {
		t.Fatalf("expected usage to contain impact command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesTests(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "tests <symbol-or-package>") {
		t.Fatalf("expected usage to contain tests command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesCallees(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "callees <function-or-method>") {
		t.Fatalf("expected usage to contain callees command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesCallers(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "callers <function-or-method>") {
		t.Fatalf("expected usage to contain callers command, got:\n%s", output.String())
	}
}

func TestMainPrintsTestsUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "tests"})

	want := "usage: gosherpa [--root <path>] tests <symbol-or-package>\n"
	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if result.Stderr != want {
		t.Fatalf("expected %q, got %q", want, result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestRunReturnsUsageExitWhenCommandIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "usage: gosherpa [--root <path>] <command> [args]") {
		t.Fatalf("expected usage in stderr, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "ParseFile"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"TESTS", "ParseFile (symbol)", "RELATED TESTS", "TestParserPackage", "TestUsesParser", "SUGGESTED COMMANDS", "go test ./cmd/app", "go test ./internal/parser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsImpactUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "impact"})

	want := "usage: gosherpa [--root <path>] impact <symbol-or-package>\n"
	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if result.Stderr != want {
		t.Fatalf("expected %q, got %q", want, result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "ParseFile"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"IMPACT", "ParseFile (symbol)", "REFERENCES", "DIRECT CALLERS", "AFFECTED PACKAGES", "SUGGESTED TESTS", "SUGGESTED COMMANDS", "TestParserPackage", "TestUsesParser", "go test ./cmd/app", "go test ./internal/parser", "./cmd/app", "./internal/parser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsPathUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "path", "Entry"})

	want := "usage: gosherpa [--root <path>] path <from> <to> [--limit <n>] [--max-depth <n>]\n"
	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if result.Stderr != want {
		t.Fatalf("expected %q, got %q", want, result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "path", "Entry", "Target"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"CALL PATH", "Entry", "Middle", "Target", "Found 1 path"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
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

	result := runMainTest(t, []string{"gosherpa", "paths", "Entry", "Target", "--limit", "2", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"CALL PATHS", "Path 1", "Path 2", "First", "Second", "Found 2 paths"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsCallersUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "callers"})

	want := "usage: gosherpa [--root <path>] callers <function-or-method>\n"
	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if result.Stderr != want {
		t.Fatalf("expected %q, got %q", want, result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "callers", "Step"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"CALLERS", "Step", "Run", "Found 1 callers"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsCalleesUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "callees"})

	want := "usage: gosherpa [--root <path>] callees <function-or-method>\n"
	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if result.Stderr != want {
		t.Fatalf("expected %q, got %q", want, result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
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

	result := runMainTest(t, []string{"gosherpa", "callees", "Run", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"CALLEES", "Run", "Step", "Found 1 callees"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsRefsUsageWithoutValidatingRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	result := runMainTest(t, []string{"gosherpa", "--root", missingRoot, "refs"})

	want := "usage: gosherpa [--root <path>] refs <name>\n"
	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if result.Stderr != want {
		t.Fatalf("expected %q, got %q", want, result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainPrintsUnknownCommandWithoutValidatingRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	result := runMainTest(t, []string{"gosherpa", "--root", missingRoot, "unknown"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	for _, want := range []string{
		"unknown command: unknown",
		"usage: gosherpa [--root <path>] <command> [args]",
	} {
		if !strings.Contains(result.Stderr, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stderr)
		}
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}

	if strings.Contains(result.Stderr, "error: repository root") {
		t.Fatalf("expected no root validation, got:\n%s", result.Stderr)
	}
}

func TestMainRunsSymbolsWithRootBeforeCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Run() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbols"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"Run", "internal/service/service.go"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
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

	result := runMainTest(t, []string{"gosherpa", "refs", "ParseFile", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"REFERENCES", "ParseFile", "internal/service/service.go", "Found 2 references"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsErrorWhenRootIsMissing(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	result := runMainTest(t, []string{"gosherpa", "--root", missingRoot, "symbols"})

	if result.ExitCode != exitFailure {
		t.Fatalf("expected exit %d, got %d", exitFailure, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: repository root does not exist") {
		t.Fatalf("expected missing root error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainPrintsErrorWhenRootHasNoGoMod(t *testing.T) {
	tmp := t.TempDir()
	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbols"})

	if result.ExitCode != exitFailure {
		t.Fatalf("expected exit %d, got %d", exitFailure, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: repository root does not contain go.mod") {
		t.Fatalf("expected missing go.mod error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestRunReturnsFailureExitWhenSymbolIsMissing(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), "package service\n")

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbol", "Missing"})

	if result.ExitCode != exitFailure {
		t.Fatalf("expected exit %d, got %d", exitFailure, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "symbol not found: Missing") {
		t.Fatalf("expected missing symbol error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainPrintsErrorForUnknownFlag(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "--json", "symbols"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: unknown flag: --json") {
		t.Fatalf("expected unknown flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

type mainTestRunResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func runMainTest(t *testing.T, args []string) mainTestRunResult {
	t.Helper()

	commandArgs := args
	if len(commandArgs) > 0 && filepath.Base(commandArgs[0]) == "gosherpa" {
		commandArgs = commandArgs[1:]
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(commandArgs, &stdout, &stderr)

	return mainTestRunResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
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
