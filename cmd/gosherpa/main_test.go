package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
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
	_, err := parseCLIArgs([]string{"--xml", "symbols"})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "unknown flag: --xml") {
		t.Fatalf("expected unknown flag error, got %v", err)
	}
}

func TestParseCLIArgsAcceptsJSONFlag(t *testing.T) {
	tests := [][]string{
		{"--json", "refs", "ParseFile"},
		{"refs", "ParseFile", "--json"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			got, err := parseCLIArgs(test)
			if err != nil {
				t.Fatal(err)
			}

			if !got.JSON {
				t.Fatal("expected JSON flag")
			}

			if got.Command != "refs" {
				t.Fatalf("expected command refs, got %s", got.Command)
			}

			assertMainTestStrings(t, got.CommandArgs, []string{"ParseFile"})
		})
	}
}

func TestParseCLIArgsAcceptsBaseFlagForDiffCommands(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		command     string
		commandArgs []string
	}{
		{
			name:        "impact diff spaced",
			args:        []string{"impact", "diff", "--base", "HEAD"},
			command:     "impact",
			commandArgs: []string{"diff"},
		},
		{
			name:        "impact diff equals",
			args:        []string{"--base=HEAD", "impact", "diff"},
			command:     "impact",
			commandArgs: []string{"diff"},
		},
		{
			name:        "tests affected",
			args:        []string{"tests", "affected", "--base", "HEAD"},
			command:     "tests",
			commandArgs: []string{"affected"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseCLIArgs(test.args)
			if err != nil {
				t.Fatal(err)
			}

			if got.Command != test.command {
				t.Fatalf("expected command %s, got %s", test.command, got.Command)
			}

			assertMainTestStrings(t, got.CommandArgs, test.commandArgs)

			if got.BaseRef != "HEAD" {
				t.Fatalf("expected base HEAD, got %s", got.BaseRef)
			}

			if !got.HasBaseOption {
				t.Fatal("expected base option marker")
			}
		})
	}
}

func TestParseCLIArgsRejectsMissingBaseValue(t *testing.T) {
	tests := [][]string{
		{"impact", "diff", "--base"},
		{"impact", "diff", "--base="},
		{"impact", "diff", "--base", "--json"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			_, err := parseCLIArgs(test)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), "missing value for --base") {
				t.Fatalf("expected missing base value error, got %v", err)
			}
		})
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
		"--json           machine-readable output for all commands",
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

	for _, want := range []string{
		"impact <symbol-or-package>",
		"impact file <file>",
		"impact package <package>",
		"impact symbol <symbol>",
		"impact diff --base <ref>",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesTests(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{
		"tests <symbol-or-package>",
		"tests affected --base <ref>",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
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

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	for _, want := range []string{
		"usage: gosherpa [--root <path>] tests <symbol-or-package>",
		"gosherpa [--root <path>] tests affected --base <ref>",
	} {
		if !strings.Contains(result.Stderr, want) {
			t.Fatalf("expected stderr to contain %s, got %q", want, result.Stderr)
		}
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainPrintsTestsAffectedUsageWhenBaseIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "tests", "affected"})

	want := "usage: gosherpa [--root <path>] tests affected --base <ref>\n"
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

func TestMainRunsTestsCommandAsJSON(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "ParseFile", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "tests", "ParseFile", "example.com/app")

	if data["kind"] != "symbol" {
		t.Fatalf("expected kind symbol, got %v", data["kind"])
	}

	assertMainTestJSONArrayHasLength(t, data, "tests", 2)
	assertMainTestJSONArrayHasLength(t, data, "commands", 2)

	if strings.Contains(result.Stdout, "TESTS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsTestsAffectedCommand(t *testing.T) {
	tmp := t.TempDir()
	initMainTestGitRepository(t, tmp)

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestSession(t *testing.T) {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler.go"), `package api

import "example.com/app/internal/auth"

var _ = auth.Session{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler_test.go"), `package api

import "testing"

func TestHandler(t *testing.T) {}
`)
	runMainTestGit(t, tmp, "add", ".")
	runMainTestGit(t, tmp, "commit", "-m", "initial")

	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	return Session{}
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "affected", "--base", "HEAD"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"AFFECTED TESTS",
		"TestSession",
		"TestHandler",
		"SUGGESTED COMMANDS",
		"go test ./internal/api",
		"go test ./internal/auth",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	for _, notWant := range []string{"IMPACT DIFF", "CHANGED FILES", "CHANGED PACKAGES"} {
		if strings.Contains(result.Stdout, notWant) {
			t.Fatalf("expected output not to contain %s, got:\n%s", notWant, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsTestsAffectedCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()
	initMainTestGitRepository(t, tmp)

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestSession(t *testing.T) {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler.go"), `package api

import "example.com/app/internal/auth"

var _ = auth.Session{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler_test.go"), `package api

import "testing"

func TestHandler(t *testing.T) {}
`)
	runMainTestGit(t, tmp, "add", ".")
	runMainTestGit(t, tmp, "commit", "-m", "initial")

	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	return Session{}
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "affected", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "tests affected", "HEAD", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 2)
	assertMainTestJSONArrayHasLength(t, data, "commands", 2)

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "AFFECTED TESTS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsImpactUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "impact"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	for _, want := range []string{
		"usage: gosherpa [--root <path>] impact <symbol-or-package>",
		"gosherpa [--root <path>] impact diff --base <ref>",
	} {
		if !strings.Contains(result.Stderr, want) {
			t.Fatalf("expected stderr to contain %s, got %q", want, result.Stderr)
		}
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainPrintsImpactDiffUsageWhenBaseIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "impact", "diff"})

	want := "usage: gosherpa [--root <path>] impact diff --base <ref>\n"
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

func TestMainPrintsImpactSubcommandUsageWhenTargetIsMissing(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "file",
			args: []string{"gosherpa", "impact", "file"},
			want: "usage: gosherpa [--root <path>] impact file <file>\n",
		},
		{
			name: "package",
			args: []string{"gosherpa", "impact", "package"},
			want: "usage: gosherpa [--root <path>] impact package <package>\n",
		},
		{
			name: "symbol",
			args: []string{"gosherpa", "impact", "symbol"},
			want: "usage: gosherpa [--root <path>] impact symbol <symbol>\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := runMainTest(t, test.args)

			if result.ExitCode != exitUsage {
				t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
			}

			if result.Stderr != test.want {
				t.Fatalf("expected %q, got %q", test.want, result.Stderr)
			}

			if result.Stdout != "" {
				t.Fatalf("expected empty stdout, got %q", result.Stdout)
			}
		})
	}
}

func TestMainRejectsBaseFlagForOtherCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "symbols", "--base", "HEAD"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --base is only supported by impact diff and tests affected") {
		t.Fatalf("expected base flag error, got:\n%s", result.Stderr)
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

	for _, want := range []string{"IMPACT", "ParseFile (symbol)", "REFERENCES", "CALLERS", "AFFECTED PACKAGES", "SUGGESTED TESTS", "SUGGESTED COMMANDS", "TestParserPackage", "TestUsesParser", "go test ./cmd/app", "go test ./internal/parser", "./cmd/app", "./internal/parser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsImpactCommandAsJSON(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "--json", "--root", tmp, "impact", "ParseFile"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "impact", "ParseFile", "example.com/app")

	if data["kind"] != "symbol" {
		t.Fatalf("expected kind symbol, got %v", data["kind"])
	}

	assertMainTestJSONArrayHasLength(t, data, "references", 2)
	assertMainTestJSONArrayHasLength(t, data, "callers", 1)
	assertMainTestJSONArrayHasLength(t, data, "relatedTests", 2)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	dependencies, ok := data["dependencies"].(map[string]any)
	if !ok {
		t.Fatalf("expected dependencies to be a JSON object, got %T", data["dependencies"])
	}

	assertMainTestJSONArrayHasLength(t, dependencies, "imports", 0)
	assertMainTestJSONArrayHasLength(t, dependencies, "usedBy", 0)

	if strings.Contains(result.Stdout, "IMPACT") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsImpactFileCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "file", "service.go"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"IMPACT FILE", "CHANGED FILES", "service.go", "CHANGED PACKAGES", ".", "AFFECTED PACKAGES", "./cmd/app", "AFFECTED TESTS", "TestTarget", "SUGGESTED COMMANDS", "go test ."} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsImpactPackageCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "package", "."})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"IMPACT PACKAGE", "CHANGED PACKAGES", ".", "AFFECTED PACKAGES", "./cmd/app", "AFFECTED TESTS", "TestTarget", "SUGGESTED COMMANDS", "go test ."} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, "CHANGED FILES") {
		t.Fatalf("expected package output not to include changed files, got:\n%s", result.Stdout)
	}
}

func TestMainRunsImpactSymbolCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "symbol", "Target"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"IMPACT SYMBOL", "AFFECTED SYMBOLS", "Target", "AFFECTED PACKAGES", ".", "AFFECTED TESTS", "TestTarget", "SUGGESTED COMMANDS", "go test ."} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRunsExplainCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "explain", "Target"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"EXPLAIN", "TARGET", "Target (function)", "DEFINITION", "service.go", "PURPOSE", "READING ORDER", "CALLED BY", "Entry", "REFERENCES", "AFFECTED PACKAGES", ".", "SUGGESTED TESTS", "TestTarget", "SUGGESTED COMMANDS", "go test ."} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsExplainCommandAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "explain", "Target", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "explain", "Target", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "references", 2)
	assertMainTestJSONArrayHasLength(t, data, "callers", 1)
	assertMainTestJSONArrayHasLength(t, data, "callees", 0)
	assertMainTestJSONArrayHasLength(t, data, "relatedTests", 1)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 1)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 3)

	if data["purpose"] != "" {
		t.Fatalf("expected empty purpose, got %v", data["purpose"])
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
}

func TestMainRunsImpactDiffCommand(t *testing.T) {
	tmp := t.TempDir()
	initMainTestGitRepository(t, tmp)

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestSession(t *testing.T) {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler.go"), `package api

import "example.com/app/internal/auth"

var _ = auth.Session{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler_test.go"), `package api

import "testing"

func TestHandler(t *testing.T) {}
`)
	runMainTestGit(t, tmp, "add", ".")
	runMainTestGit(t, tmp, "commit", "-m", "initial")

	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	return Session{}
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "diff", "--base", "HEAD"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"IMPACT DIFF",
		"CHANGED FILES",
		"internal/auth/session.go",
		"CHANGED PACKAGES",
		"./internal/auth",
		"AFFECTED PACKAGES",
		"./internal/api",
		"AFFECTED SYMBOLS",
		"NewSession",
		"AFFECTED TESTS",
		"TestSession",
		"TestHandler",
		"SUGGESTED COMMANDS",
		"go test ./internal/api",
		"go test ./internal/auth",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsImpactDiffCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()
	initMainTestGitRepository(t, tmp)

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestSession(t *testing.T) {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler.go"), `package api

import "example.com/app/internal/auth"

var _ = auth.Session{}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "api", "handler_test.go"), `package api

import "testing"

func TestHandler(t *testing.T) {}
`)
	runMainTestGit(t, tmp, "add", ".")
	runMainTestGit(t, tmp, "commit", "-m", "initial")

	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	return Session{}
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "impact", "diff", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "impact diff", "HEAD", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "changedFiles", 1)
	assertMainTestJSONArrayHasLength(t, data, "changedPackages", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedPackages", 2)
	assertMainTestJSONArrayHasLength(t, data, "affectedSymbols", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 2)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "IMPACT DIFF") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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

func TestMainRunsPathCommandAsJSON(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "path", "Entry", "Target", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "path", "Entry -> Target", "example.com/app")

	if data["from"] != "Entry" {
		t.Fatalf("expected from Entry, got %v", data["from"])
	}

	if data["to"] != "Target" {
		t.Fatalf("expected to Target, got %v", data["to"])
	}

	paths := assertMainTestJSONArrayHasLength(t, data, "paths", 1)
	path, ok := paths[0].(map[string]any)
	if !ok {
		t.Fatalf("expected path object, got %T", paths[0])
	}

	assertMainTestJSONArrayHasLength(t, path, "steps", 2)

	if strings.Contains(result.Stdout, "CALL PATH") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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

func TestMainRunsPathsCommandAsJSON(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "paths", "Entry", "Target", "--limit", "2", "--json", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "paths", "Entry -> Target", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "paths", 2)

	if strings.Contains(result.Stdout, "CALL PATHS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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

func TestMainRunsCallersCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "callers", "Step", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "callers", "Step", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "callers", 1)

	if strings.Contains(result.Stdout, "CALLERS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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

func TestMainRunsCalleesCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	result := runMainTest(t, []string{"gosherpa", "callees", "Run", "--json", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "callees", "Run", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "callees", 1)

	if strings.Contains(result.Stdout, "CALLEES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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

func TestMainRunsSymbolsCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Worker struct{}

func Run() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbols", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "symbols", "", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "symbols", 2)

	if strings.Contains(result.Stdout, "FUNCTIONS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsSymbolCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Run() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbol", "Run", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "symbol", "Run", "example.com/app")

	symbol, ok := data["symbol"].(map[string]any)
	if !ok {
		t.Fatalf("expected symbol to be a JSON object, got %T", data["symbol"])
	}

	if symbol["name"] != "Run" {
		t.Fatalf("expected symbol name Run, got %v", symbol["name"])
	}

	if symbol["kind"] != "function" {
		t.Fatalf("expected symbol kind function, got %v", symbol["kind"])
	}

	if strings.Contains(result.Stdout, "Name:") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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

func TestMainRunsRefsCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func ParseFile() {
}

func Run() {
	ParseFile()
}
`)

	result := runMainTest(t, []string{"gosherpa", "refs", "ParseFile", "--json", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "refs", "ParseFile", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "references", 2)

	if strings.Contains(result.Stdout, "REFERENCES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsDepsCommandAsJSON(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "deps", "./internal/parser", "--json", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "deps", "./internal/parser", "example.com/app")

	if data["package"] != "./internal/parser" {
		t.Fatalf("expected package ./internal/parser, got %v", data["package"])
	}

	assertMainTestJSONArrayHasLength(t, data, "imports", 0)
	assertMainTestJSONArrayHasLength(t, data, "usedBy", 1)

	if strings.Contains(result.Stdout, "PACKAGE DEPENDENCIES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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
	result := runMainTest(t, []string{"gosherpa", "--xml", "symbols"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: unknown flag: --xml") {
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

func writeMainImpactReportProject(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {
	Target()
}

func Target() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "service_test.go"), `package service

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)
	writeMainTestFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app"

func Run() {
	service.Entry()
}
`)

	return tmp
}

func copyMainTestTree(t *testing.T, source string, destination string) {
	t.Helper()

	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		if relativePath == "." {
			return nil
		}

		targetPath := filepath.Join(destination, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func initMainTestGitRepository(t *testing.T, root string) {
	t.Helper()

	requireMainTestGit(t)

	runMainTestGit(t, root, "init")
	runMainTestGit(t, root, "config", "user.email", "test@example.com")
	runMainTestGit(t, root, "config", "user.name", "Test User")
}

func requireMainTestGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}
}

func runMainTestGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	commandArgs := append([]string{"-C", root}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}

	return string(output)
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

func decodeMainTestJSON(t *testing.T, output string) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected valid JSON, got %v:\n%s", err, output)
	}

	return payload
}

func assertMainTestJSONEnvelope(t *testing.T, payload map[string]any, root string, command string, target string, modulePath string) map[string]any {
	t.Helper()

	if payload["schemaVersion"] != float64(jsonSchemaVersion) {
		t.Fatalf("expected schemaVersion %d, got %v", jsonSchemaVersion, payload["schemaVersion"])
	}

	if payload["command"] != command {
		t.Fatalf("expected command %s, got %v", command, payload["command"])
	}

	if payload["target"] != target {
		t.Fatalf("expected target %s, got %v", target, payload["target"])
	}

	if payload["root"] != filepath.Clean(root) {
		t.Fatalf("expected root %s, got %v", filepath.Clean(root), payload["root"])
	}

	if payload["modulePath"] != modulePath {
		t.Fatalf("expected modulePath %s, got %v", modulePath, payload["modulePath"])
	}

	assertMainTestJSONArrayHasLength(t, payload, "warnings", 0)

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a JSON object, got %T", payload["data"])
	}

	return data
}

func assertMainTestJSONArrayHasLength(t *testing.T, payload map[string]any, key string, length int) []any {
	t.Helper()

	values, ok := payload[key].([]any)
	if !ok {
		t.Fatalf("expected %s to be a JSON array, got %T", key, payload[key])
	}

	if len(values) != length {
		t.Fatalf("expected %s length %d, got %d", key, length, len(values))
	}

	return values
}
