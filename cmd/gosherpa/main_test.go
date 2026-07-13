package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	"github.com/panndabea/GoSherpa/internal/sherpa"
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

func TestParseCLIArgsAcceptsBuildTags(t *testing.T) {
	got, err := parseCLIArgs([]string{"--tags", "enterprise,integration", "refs", "Target"})
	if err != nil {
		t.Fatal(err)
	}

	if !got.HasTagsOption {
		t.Fatal("expected tags option")
	}
	assertMainTestStrings(t, got.BuildTags, []string{"enterprise", "integration"})
}

func TestParseCLIArgsRejectsMissingBuildTagsValue(t *testing.T) {
	tests := [][]string{
		{"--tags"},
		{"--tags="},
		{"--tags", "   "},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			_, err := parseCLIArgs(test)
			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), "missing value for --tags") {
				t.Fatalf("expected missing tags value error, got %v", err)
			}
		})
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

func TestParseCLIArgsAcceptsUseSnapshotFlag(t *testing.T) {
	tests := [][]string{
		{"analyze", "--use-snapshot"},
		{"--use-snapshot", "symbols"},
		{"search", "target", "--use-snapshot"},
		{"context", "diff", "--base", "HEAD", "--use-snapshot"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			got, err := parseCLIArgs(test)
			if err != nil {
				t.Fatal(err)
			}

			if !got.UseSnapshot || !got.HasSnapshotOption {
				t.Fatalf("expected snapshot option, got %#v", got)
			}
		})
	}
}

func TestParseCLIArgsAcceptsAllFlag(t *testing.T) {
	got, err := parseCLIArgs([]string{"deps", "--all"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Command != "deps" {
		t.Fatalf("expected command deps, got %s", got.Command)
	}

	if !got.All || !got.HasAllOption {
		t.Fatal("expected all option")
	}

	if len(got.CommandArgs) != 0 {
		t.Fatalf("expected no command args, got %v", got.CommandArgs)
	}
}

func TestParseCLIArgsAcceptsContextFlag(t *testing.T) {
	tests := [][]string{
		{"--context", "symbol", "Run"},
		{"symbol", "Run", "--context"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			got, err := parseCLIArgs(test)
			if err != nil {
				t.Fatal(err)
			}

			if !got.ShowContext {
				t.Fatal("expected context flag")
			}

			if !got.HasContextOption {
				t.Fatal("expected context option marker")
			}

			if got.Command != "symbol" {
				t.Fatalf("expected command symbol, got %s", got.Command)
			}

			assertMainTestStrings(t, got.CommandArgs, []string{"Run"})
		})
	}
}

func TestParseCLIArgsAcceptsTestsFlag(t *testing.T) {
	tests := [][]string{
		{"entrypoints", "Target", "--tests"},
		{"--tests", "callers", "Target"},
		{"explain", "Target", "--tests"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			got, err := parseCLIArgs(test)
			if err != nil {
				t.Fatal(err)
			}

			if !got.IncludeTests {
				t.Fatal("expected tests flag")
			}

			if !got.HasTestsOption {
				t.Fatal("expected tests option marker")
			}
		})
	}
}

func TestParseCLIArgsAcceptsTestScope(t *testing.T) {
	tests := [][]string{
		{"tests", "Target", "--scope", "direct"},
		{"tests", "Target", "--scope=all"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			got, err := parseCLIArgs(test)
			if err != nil {
				t.Fatal(err)
			}

			if !got.HasTestScopeOption {
				t.Fatal("expected scope option marker")
			}
			if got.TestScope == "" {
				t.Fatal("expected test scope")
			}
		})
	}
}

func TestParseCLIArgsRejectsInvalidTestScope(t *testing.T) {
	_, err := parseCLIArgs([]string{"tests", "Target", "--scope", "wide"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid value for --scope") {
		t.Fatalf("expected invalid scope error, got %v", err)
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
		{
			name:        "context diff",
			args:        []string{"context", "diff", "--base", "HEAD"},
			command:     "context",
			commandArgs: []string{"diff"},
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

	if !got.HasLimitOption {
		t.Fatal("expected limit option marker")
	}

	if !got.HasMaxDepthOption {
		t.Fatal("expected max-depth option marker")
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

func TestParseCLIArgsAcceptsSearchFilters(t *testing.T) {
	got, err := parseCLIArgs([]string{"search", "user", "--kind", "Function", "--package=./internal/service", "--tests", "--limit", "2"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Command != "search" {
		t.Fatalf("expected command search, got %s", got.Command)
	}

	assertMainTestStrings(t, got.CommandArgs, []string{"user"})

	if got.SearchKind != sherpa.SymbolKindFunction {
		t.Fatalf("expected function kind, got %s", got.SearchKind)
	}

	if !got.HasKindOption {
		t.Fatal("expected kind option marker")
	}

	if got.SearchPackage != "./internal/service" {
		t.Fatalf("expected package ./internal/service, got %s", got.SearchPackage)
	}

	if !got.HasPackageOption {
		t.Fatal("expected package option marker")
	}

	if !got.IncludeTests || !got.HasTestsOption {
		t.Fatal("expected tests option")
	}

	if got.CallPathLimit != 2 {
		t.Fatalf("expected limit 2, got %d", got.CallPathLimit)
	}

	if !got.HasLimitOption {
		t.Fatal("expected limit option marker")
	}
}

func TestParseCLIArgsAcceptsSymbolsFilters(t *testing.T) {
	got, err := parseCLIArgs([]string{"symbols", "--kind", "Function", "--package=./internal/service", "--tests"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Command != "symbols" {
		t.Fatalf("expected command symbols, got %s", got.Command)
	}

	if got.SearchKind != sherpa.SymbolKindFunction {
		t.Fatalf("expected function kind, got %s", got.SearchKind)
	}

	if !got.HasKindOption {
		t.Fatal("expected kind option marker")
	}

	if got.SearchPackage != "./internal/service" {
		t.Fatalf("expected package ./internal/service, got %s", got.SearchPackage)
	}

	if !got.HasPackageOption {
		t.Fatal("expected package option marker")
	}

	if !got.IncludeTests || !got.HasTestsOption {
		t.Fatal("expected tests option")
	}
}

func TestParseCLIArgsAcceptsAliasSymbolKind(t *testing.T) {
	got, err := parseCLIArgs([]string{"symbols", "--kind", "alias"})
	if err != nil {
		t.Fatal(err)
	}

	if got.SearchKind != sherpa.SymbolKindAlias {
		t.Fatalf("expected alias kind, got %s", got.SearchKind)
	}
}

func TestParseCLIArgsAcceptsRefsKindFilter(t *testing.T) {
	got, err := parseCLIArgs([]string{"refs", "ParseFile", "--kind", "call"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Command != "refs" {
		t.Fatalf("expected command refs, got %s", got.Command)
	}

	assertMainTestStrings(t, got.CommandArgs, []string{"ParseFile"})

	if got.ReferenceKind != sherpa.ReferenceKindCall {
		t.Fatalf("expected call reference kind, got %s", got.ReferenceKind)
	}

	if !got.HasKindOption {
		t.Fatal("expected kind option marker")
	}
}

func TestParseCLIArgsRejectsInvalidSearchFilters(t *testing.T) {
	tests := [][]string{
		{"search", "user", "--kind"},
		{"search", "user", "--kind="},
		{"search", "user", "--kind", "package"},
		{"symbols", "--kind", "package"},
		{"search", "user", "--package"},
		{"search", "user", "--package="},
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

func TestParseCLIArgsRejectsInvalidRefsKindFilters(t *testing.T) {
	tests := [][]string{
		{"refs", "ParseFile", "--kind"},
		{"refs", "ParseFile", "--kind="},
		{"refs", "ParseFile", "--kind", "function"},
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

func TestPrintUsageIncludesAnalyze(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "analyze [path] [--tests] [--use-snapshot]") {
		t.Fatalf("expected usage to contain analyze command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesArchitecture(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "architecture [--tests]") {
		t.Fatalf("expected usage to contain architecture command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesRisk(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "risk [--tests]") {
		t.Fatalf("expected usage to contain risk command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesPathCommands(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{
		"path <from> <to>",
		"paths <from> <to> [--limit <n>] [--max-depth <n>]",
		"entrypoints <function-or-method> [--tests]",
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
		"pr --base <ref>",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesContext(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{
		"context symbol <target> [--tests] [--max-references <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]",
		"context file <file> [--tests] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]",
		"context package <package> [--tests] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]",
		"context diff --base <ref> [--tests] [--use-snapshot] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>]",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesDoctor(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "doctor") {
		t.Fatalf("expected usage to contain doctor command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesSnapshot(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "snapshot") {
		t.Fatalf("expected usage to contain snapshot command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesVersion(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{"--version        print GoSherpa version information", "\n  version\n"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesTests(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{
		"tests <symbol-or-package-or-file> [--scope direct|related|all]",
		"tests affected --base <ref>",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesPackages(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "packages [--tests]") {
		t.Fatalf("expected usage to contain packages command, got:\n%s", output.String())
	}
}

func TestPrintUsageIncludesDeps(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{"deps <package>", "deps --all"} {
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

func TestPrintUsageIncludesInterfaceNavigation(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	for _, want := range []string{
		"implementers <interface>",
		"interface <interface>",
		"interfaces <type>",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected usage to contain %s, got:\n%s", want, output.String())
		}
	}
}

func TestPrintUsageIncludesSearch(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)

	if !strings.Contains(output.String(), "search <terms>") {
		t.Fatalf("expected usage to contain search command, got:\n%s", output.String())
	}
}

func TestMainPrintsTestsUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "tests"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	for _, want := range []string{
		"usage: gosherpa [--root <path>] tests <symbol-or-package-or-file> [--scope direct|related|all]",
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

func TestMainRunsVersionCommand(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "version"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"GoSherpa dev\n", "go: go"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected stdout to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRunsVersionFlag(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "--version"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	if !strings.Contains(result.Stdout, "GoSherpa dev\n") {
		t.Fatalf("expected stdout to contain version, got:\n%s", result.Stdout)
	}
}

func TestMainRunsVersionCommandJSON(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "version", "--json"})
	assertMainTestVersionJSON(t, result)
}

func TestMainRunsVersionFlagJSON(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "--json", "--version"})
	assertMainTestVersionJSON(t, result)
}

func assertMainTestVersionJSON(t *testing.T, result mainTestRunResult) {
	t.Helper()

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	if payload["schemaVersion"] != float64(jsonSchemaVersion) {
		t.Fatalf("expected schemaVersion %d, got %v", jsonSchemaVersion, payload["schemaVersion"])
	}
	if payload["command"] != "version" {
		t.Fatalf("expected command version, got %v", payload["command"])
	}
	if payload["target"] != "" {
		t.Fatalf("expected empty target, got %v", payload["target"])
	}
	if payload["root"] != "" {
		t.Fatalf("expected empty root, got %v", payload["root"])
	}
	if payload["modulePath"] != "" {
		t.Fatalf("expected empty modulePath, got %v", payload["modulePath"])
	}
	if warnings := assertMainTestJSONArray(t, payload, "warnings"); len(warnings) != 0 {
		t.Fatalf("expected warnings length 0, got %d: %#v", len(warnings), warnings)
	}

	data := assertMainTestJSONObject(t, payload, "data")
	if data["version"] != "dev" {
		t.Fatalf("expected version dev, got %v", data["version"])
	}
	goVersion, ok := data["goVersion"].(string)
	if !ok || !strings.HasPrefix(goVersion, "go") {
		t.Fatalf("expected goVersion to start with go, got %v", data["goVersion"])
	}
}

func TestMainRunsDoctorCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "doctor"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"DOCTOR",
		"Repository: " + filepath.Clean(tmp),
		"Module: example.com/app",
		"Analysis: typechecked",
		"Confidence: medium",
		"FILES",
		"PACKAGE LOAD",
		"Status: ok",
		"SNAPSHOT",
		"LIMITATIONS",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected stdout to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRunsDoctorCommandJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "doctor", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "doctor", ".", "example.com/app")

	if data["analysisMode"] != "typechecked" {
		t.Fatalf("expected typechecked analysis mode, got %v", data["analysisMode"])
	}
	if data["confidence"] != agentcontext.ConfidenceMedium {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}

	repository := assertMainTestJSONObject(t, data, "repository")
	if repository["modulePath"] != "example.com/app" {
		t.Fatalf("expected repository module path, got %v", repository["modulePath"])
	}
	assertMainTestJSONArrayHasLength(t, repository, "nestedModules", 0)
	goWork := assertMainTestJSONObject(t, repository, "goWork")
	if goWork["detected"] != false {
		t.Fatalf("expected no go.work, got %v", goWork["detected"])
	}

	packageLoad := assertMainTestJSONObject(t, data, "packageLoad")
	if packageLoad["status"] != "ok" {
		t.Fatalf("expected package load ok, got %v", packageLoad["status"])
	}
	if packageLoad["analysisMode"] != "typechecked" {
		t.Fatalf("expected typechecked package load, got %v", packageLoad["analysisMode"])
	}
	if packageLoad["packageCount"].(float64) <= 0 {
		t.Fatalf("expected loaded packages, got %v", packageLoad["packageCount"])
	}

	snapshot := assertMainTestJSONObject(t, data, "snapshot")
	if snapshot["status"] != "missing" {
		t.Fatalf("expected missing snapshot, got %v", snapshot["status"])
	}
	if snapshot["supported"] != true {
		t.Fatalf("expected snapshot support, got %v", snapshot["supported"])
	}

	assertMainTestJSONArrayHasLength(t, data, "buildTags", 0)
	assertMainTestJSONArrayHasLength(t, data, "limitations", 4)

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
	if strings.Contains(result.Stdout, "DOCTOR") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsDoctorCommandWithBuildTagsAndGoWork(t *testing.T) {
	tmp := writeMainImpactReportProject(t)
	writeMainTestFile(t, filepath.Join(tmp, "go.work"), `go 1.24

use .
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "--tags", "enterprise,integration", "doctor", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "doctor", ".", "example.com/app")

	tags := assertMainTestJSONArrayHasLength(t, data, "buildTags", 2)
	if tags[0] != "enterprise" || tags[1] != "integration" {
		t.Fatalf("expected sorted build tags, got %v", tags)
	}

	repository := assertMainTestJSONObject(t, data, "repository")
	goWork := assertMainTestJSONObject(t, repository, "goWork")
	if goWork["detected"] != true {
		t.Fatalf("expected go.work detection, got %v", goWork["detected"])
	}
	if goWork["path"] != "go.work" {
		t.Fatalf("expected root-relative go.work path, got %v", goWork["path"])
	}
	if goWork["scope"] != "root" {
		t.Fatalf("expected root go.work scope, got %v", goWork["scope"])
	}
}

func TestMainPrintsDoctorUsageWhenArgumentIsUnexpected(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "doctor", "extra"})

	want := "usage: gosherpa [--root <path>] doctor\n"
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

func TestMainRunsSnapshotCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"SNAPSHOT",
		"Status: valid",
		"Path: .gosherpa/snapshot.json",
		"Module: example.com/app",
		"CONTENTS",
		"Files:",
		"Packages:",
		"Symbols:",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected stdout to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if _, err := os.Stat(filepath.Join(tmp, ".gosherpa", "snapshot.json")); err != nil {
		t.Fatalf("expected snapshot file: %v", err)
	}
}

func TestMainRunsSnapshotCommandJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "--tags", "enterprise", "snapshot", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "snapshot", ".", "example.com/app")
	if data["status"] != "valid" {
		t.Fatalf("expected valid snapshot status, got %v", data["status"])
	}
	if data["path"] != ".gosherpa/snapshot.json" {
		t.Fatalf("expected snapshot path, got %v", data["path"])
	}

	snapshot := assertMainTestJSONObject(t, data, "snapshot")
	if snapshot["modulePath"] != "example.com/app" {
		t.Fatalf("expected module path, got %v", snapshot["modulePath"])
	}
	if snapshot["formatVersion"] != float64(1) {
		t.Fatalf("expected format version 1, got %v", snapshot["formatVersion"])
	}
	if snapshot["fingerprint"] == "" {
		t.Fatalf("expected fingerprint, got %v", snapshot["fingerprint"])
	}
	assertMainTestJSONArrayHasLength(t, snapshot, "buildTags", 1)
	assertMainTestJSONArrayHasLength(t, snapshot, "files", 4)
	assertMainTestJSONArrayHasLength(t, snapshot, "packages", 2)
	assertMainTestJSONArrayHasLength(t, snapshot, "symbols", 4)

	if strings.Contains(result.Stdout, "SNAPSHOT\n") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainDoctorReportsValidSnapshot(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "doctor", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "doctor", ".", "example.com/app")
	snapshot := assertMainTestJSONObject(t, data, "snapshot")
	if snapshot["status"] != "valid" {
		t.Fatalf("expected valid snapshot, got %v", snapshot["status"])
	}
	if snapshot["fileCount"].(float64) <= 0 {
		t.Fatalf("expected file count, got %v", snapshot["fileCount"])
	}
	assertMainTestJSONArrayHasLength(t, snapshot, "staleReasons", 0)
}

func TestMainDoctorReportsStaleSnapshot(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {
	Target()
}

func Target() {}

func NewChange() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "doctor", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "doctor", ".", "example.com/app")
	snapshot := assertMainTestJSONObject(t, data, "snapshot")
	if snapshot["status"] != "stale" {
		t.Fatalf("expected stale snapshot, got %v", snapshot["status"])
	}
	staleReasons := assertMainTestJSONArrayHasLength(t, snapshot, "staleReasons", 1)
	if staleReasons[0] != "repository files changed" {
		t.Fatalf("expected file-change stale reason, got %v", staleReasons)
	}
}

func TestMainPrintsSnapshotUsageWhenArgumentIsUnexpected(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "snapshot", "extra"})

	want := "usage: gosherpa [--root <path>] snapshot\n"
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

func TestMainRunsAnalyzeCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "analyze"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"ANALYZE",
		"Module: example.com/app",
		"Analysis: typechecked+ast",
		"Confidence: medium",
		"SUMMARY",
		"Packages: 2",
		"Go files: 3",
		"Test files: 1",
		"Symbols: 3",
		"SYMBOL SUMMARY",
		"Functions: 3",
		"Test symbols: 1",
		"PACKAGE OVERVIEW",
		"IMPORTANT SYMBOLS",
		"example.com/app.Entry",
		"ENTRY POINTS",
		"RISK",
		"Level: medium",
		"fan_in",
		"HOTSPOTS",
		"TESTING",
		"Suggested tests",
		"go test ./...",
		"READINESS",
		"SUGGESTED NEXT COMMANDS",
		"gosherpa context package .",
		"LIMITATIONS",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected stdout to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsAnalyzeCommandWithPathArgument(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "analyze", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "ANALYZE") {
		t.Fatalf("expected analyze output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsAnalyzeCommandWithRootRelativePathArgument(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "analyze", "."})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "Module: example.com/app") {
		t.Fatalf("expected analyze output for root-relative path, got:\n%s", result.Stdout)
	}
}

func TestMainRunsAnalyzeCommandAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "analyze", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "analyze", ".", "example.com/app")

	if data["analysisMode"] != agentcontext.AnalysisModeTypecheckedAST {
		t.Fatalf("expected typechecked+ast analysis mode, got %v", data["analysisMode"])
	}
	if data["confidence"] != agentcontext.ConfidenceMedium {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}

	repository := assertMainTestJSONObject(t, data, "repository")
	if repository["packageCount"] != float64(2) {
		t.Fatalf("expected 2 packages, got %#v", repository["packageCount"])
	}
	if repository["symbolCount"] != float64(3) {
		t.Fatalf("expected 3 production symbols, got %#v", repository["symbolCount"])
	}

	symbolSummary := assertMainTestJSONObject(t, data, "symbolSummary")
	if symbolSummary["tests"] != float64(1) {
		t.Fatalf("expected 1 test symbol, got %#v", symbolSummary["tests"])
	}

	assertMainTestJSONArrayHasLength(t, data, "packages", 2)
	assertMainTestJSONArrayHasLength(t, data, "importantSymbols", 3)
	assertMainTestJSONArrayHasLength(t, data, "entrypoints", 3)
	risk := assertMainTestJSONObject(t, data, "risk")
	if risk["level"] != sherpa.RiskLevelMedium {
		t.Fatalf("expected medium risk, got %#v", risk["level"])
	}
	if risk["score"] != float64(3) {
		t.Fatalf("expected risk score 3, got %#v", risk["score"])
	}
	assertMainTestJSONArrayHasLength(t, risk, "factors", 4)
	assertMainTestJSONArrayHasLength(t, risk, "packages", 2)
	assertMainTestJSONArrayHasLength(t, data, "hotspots", 2)
	assertMainTestJSONArrayHasLength(t, data, "limitations", 6)
	assertMainTestJSONArrayHasLength(t, data, "suggestions", 6)

	testingOverview := assertMainTestJSONObject(t, data, "testing")
	assertMainTestJSONArrayHasLength(t, testingOverview, "testPackages", 1)
	assertMainTestJSONArrayHasLength(t, testingOverview, "suggestedCommands", 2)

	readiness := assertMainTestJSONObject(t, data, "readiness")
	if readiness["packageLoad"] != "ok" {
		t.Fatalf("expected package load ok, got %#v", readiness["packageLoad"])
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
	if strings.Contains(result.Stdout, "ANALYZE") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsAnalyzeCommandAsJSONWithTests(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "analyze", "--tests", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "analyze", ".", "example.com/app")

	repository := assertMainTestJSONObject(t, data, "repository")
	if repository["symbolCount"] != float64(4) {
		t.Fatalf("expected test-inclusive symbol count 4, got %#v", repository["symbolCount"])
	}
	risk := assertMainTestJSONObject(t, data, "risk")
	if risk["symbolCount"] != float64(4) {
		t.Fatalf("expected test-inclusive risk symbol count 4, got %#v", risk["symbolCount"])
	}
	assertMainTestJSONArrayHasLength(t, data, "entrypoints", 4)
}

func TestMainRunsAnalyzeCommandFromSnapshotAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "analyze", "--tests", "--use-snapshot", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "analyze", ".", "example.com/app")
	if data["analysisMode"] != analyzeAnalysisModeSnapshotTypechecked {
		t.Fatalf("expected snapshot analysis mode, got %v", data["analysisMode"])
	}
	if data["confidence"] != agentcontext.ConfidenceMedium {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}
	assertMainTestJSONArrayHasLength(t, payload, "warnings", 0)

	repository := assertMainTestJSONObject(t, data, "repository")
	if repository["symbolCount"] != float64(4) {
		t.Fatalf("expected test-inclusive symbol count 4, got %#v", repository["symbolCount"])
	}
	assertMainTestJSONArrayHasLength(t, data, "packages", 2)
	if !mainTestJSONArrayContainsSubstring(data["limitations"].([]any), "reused a valid snapshot") {
		t.Fatalf("expected snapshot limitation, got %#v", data["limitations"])
	}
}

func TestMainPrintsAnalyzeUsageWhenArgumentIsUnexpected(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "analyze", ".", "extra"})

	want := "usage: gosherpa [--root <path>] analyze [path] [--tests] [--use-snapshot]\n"
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

func TestMainRunsArchitectureCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "architecture"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"ARCHITECTURE",
		"Analysis: ast",
		"Confidence: medium",
		"Packages: 2",
		"DEPENDENCY CYCLES",
		"MOST COUPLED PACKAGES",
		"HIGH FAN-IN PACKAGES",
		"HIGH FAN-OUT PACKAGES",
		"LARGEST PACKAGES",
		"LEAF PACKAGES",
		"LIMITATIONS",
		"Test-file imports and test-only symbols are excluded unless --tests is set.",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected stdout to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsArchitectureCommandAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "architecture", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "architecture", ".", "example.com/app")

	if data["analysisMode"] != sherpa.ArchitectureAnalysisModeAST {
		t.Fatalf("expected architecture analysis mode ast, got %#v", data["analysisMode"])
	}
	if data["confidence"] != sherpa.ArchitectureConfidence {
		t.Fatalf("expected architecture confidence medium, got %#v", data["confidence"])
	}
	if data["packageCount"] != float64(2) {
		t.Fatalf("expected package count 2, got %#v", data["packageCount"])
	}

	assertMainTestJSONArrayHasLength(t, data, "cycles", 0)
	assertMainTestJSONArrayHasLength(t, data, "limitations", 3)
	assertMainTestJSONArrayHasLength(t, data, "mostCoupled", 2)
	assertMainTestJSONArrayHasLength(t, data, "highFanIn", 1)
	assertMainTestJSONArrayHasLength(t, data, "highFanOut", 1)
	assertMainTestJSONArrayHasLength(t, data, "largestPackages", 2)
	assertMainTestJSONArrayHasLength(t, data, "leafPackages", 1)

	if strings.Contains(result.Stdout, "ARCHITECTURE") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsArchitectureCommandWithTests(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "architecture", "--tests", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "architecture", ".", "example.com/app")
	largestPackages := assertMainTestJSONArrayHasLength(t, data, "largestPackages", 2)

	rootPackage, ok := largestPackages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected package object, got %#v", largestPackages[0])
	}
	if rootPackage["package"] != "." {
		t.Fatalf("expected root package first, got %#v", rootPackage["package"])
	}
	if rootPackage["symbols"] != float64(3) {
		t.Fatalf("expected test-inclusive symbol count 3, got %#v", rootPackage["symbols"])
	}
	if rootPackage["externalImports"] != float64(1) {
		t.Fatalf("expected test import to be included, got %#v", rootPackage["externalImports"])
	}
}

func TestMainPrintsArchitectureUsageWhenArgumentIsUnexpected(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "architecture", "extra"})

	want := "usage: gosherpa [--root <path>] architecture [--tests]\n"
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

func TestMainRunsRiskCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "risk"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"RISK",
		"Level: medium",
		"Score: 3",
		"SUMMARY",
		"Packages: 2",
		"Exported symbols: 3",
		"FACTORS",
		"fan_in",
		"public_api",
		"PACKAGE SIGNALS",
		"DEPENDENCY CYCLES",
		"LIMITATIONS",
		"Risk is a deterministic structural summary, not a prediction of defects.",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected stdout to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsRiskCommandAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "risk", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "risk", ".", "example.com/app")

	if data["level"] != sherpa.RiskLevelMedium {
		t.Fatalf("expected medium risk, got %#v", data["level"])
	}
	if data["score"] != float64(3) {
		t.Fatalf("expected score 3, got %#v", data["score"])
	}
	if data["packageCount"] != float64(2) {
		t.Fatalf("expected package count 2, got %#v", data["packageCount"])
	}
	if data["exportedSymbols"] != float64(3) {
		t.Fatalf("expected exported symbols 3, got %#v", data["exportedSymbols"])
	}

	assertMainTestJSONArrayHasLength(t, data, "limitations", 3)
	assertMainTestJSONArrayHasLength(t, data, "factors", 4)
	assertMainTestJSONArrayHasLength(t, data, "packages", 2)
	assertMainTestJSONArrayHasLength(t, data, "cycles", 0)

	if strings.Contains(result.Stdout, "RISK") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsRiskCommandWithTests(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "risk", "--tests", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "risk", ".", "example.com/app")

	if data["symbolCount"] != float64(4) {
		t.Fatalf("expected test-inclusive symbol count 4, got %#v", data["symbolCount"])
	}
	if data["exportedSymbols"] != float64(4) {
		t.Fatalf("expected test-inclusive exported symbols 4, got %#v", data["exportedSymbols"])
	}

	packages := assertMainTestJSONArrayHasLength(t, data, "packages", 2)
	rootPackage, ok := packages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected package object, got %#v", packages[0])
	}
	if rootPackage["externalImports"] != float64(1) {
		t.Fatalf("expected test import to be included, got %#v", rootPackage["externalImports"])
	}
}

func TestMainPrintsRiskUsageWhenArgumentIsUnexpected(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "risk", "extra"})

	want := "usage: gosherpa [--root <path>] risk [--tests]\n"
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

	for _, want := range []string{"TESTS", "ParseFile (symbol)", "ANALYSIS", "Mode: typechecked+ast", "RELATED TESTS", "TestUsesParser", "TEST PLAN", "DIRECT", "FALLBACK", "go test ./cmd/app", "go test ./internal/parser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, "TestParserPackage") {
		t.Fatalf("expected focused default output to omit same-package tests when direct tests exist, got:\n%s", result.Stdout)
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsTestsCommandWithAllScope(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "ParseFile", "--scope", "all"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	for _, want := range []string{"TestParserPackage", "TestUsesParser", "RELATED", "go test ./internal/parser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRunsTestsCommandForFile(t *testing.T) {
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

func TestUsesParserFile(t *testing.T) {
	parser.ParseFile()
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "internal/parser/parser.go"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"TESTS", "internal/parser/parser.go (file)", "TestUsesParserFile", "go test ./cmd/app", "go test ./internal/parser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}
	if strings.Contains(result.Stdout, "TestParserPackage") {
		t.Fatalf("expected focused default output to omit same-package tests when direct file tests exist, got:\n%s", result.Stdout)
	}
}

func TestMainRunsTestsCommandForFileAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "cmd", "app", "main_test.go"), `package main

import (
	"testing"

	"example.com/app/internal/parser"
)

func TestUsesParserFile(t *testing.T) {
	parser.ParseFile()
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "tests", "internal/parser/parser.go", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "tests", "internal/parser/parser.go", "example.com/app")

	if data["kind"] != "file" {
		t.Fatalf("expected kind file, got %v", data["kind"])
	}
	if data["analysisMode"] != "typechecked+ast" {
		t.Fatalf("expected typechecked+ast analysis mode, got %v", data["analysisMode"])
	}
	assertMainTestJSONArrayHasLength(t, data, "tests", 1)
	assertMainTestJSONArrayHasLength(t, data, "commands", 2)
	testPlan := assertMainTestJSONObject(t, data, "testPlan")
	assertMainTestJSONArrayHasLength(t, testPlan, "direct", 1)
	assertMainTestJSONArrayHasLength(t, testPlan, "fallback", 1)
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

	if data["analysisMode"] != "typechecked+ast" {
		t.Fatalf("expected typechecked+ast analysis mode, got %v", data["analysisMode"])
	}

	if data["confidence"] != "medium" {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}

	if data["scope"] != "related" {
		t.Fatalf("expected related scope, got %v", data["scope"])
	}

	assertMainTestJSONArrayHasLength(t, data, "tests", 1)
	assertMainTestJSONArrayHasLength(t, data, "commands", 2)
	testPlan := assertMainTestJSONObject(t, data, "testPlan")
	assertMainTestJSONArrayHasLength(t, testPlan, "direct", 1)
	assertMainTestJSONArrayHasLength(t, testPlan, "related", 0)
	assertMainTestJSONArrayHasLength(t, testPlan, "fallback", 1)

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
		"TEST PLAN",
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

	if !strings.Contains(result.Stderr, "error: --base is only supported by context diff, impact diff, and tests affected") {
		t.Fatalf("expected base flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsTestsFlagForOtherCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "deps", ".", "--tests"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --tests is only supported by analyze, architecture, risk, symbols, search, packages, entrypoints, callers, explain, and context") {
		t.Fatalf("expected tests flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsUseSnapshotFlagForOtherCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "refs", "Target", "--use-snapshot"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --use-snapshot is only supported by analyze, symbols, symbol, search, packages, and context diff") {
		t.Fatalf("expected snapshot flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsUseSnapshotFlagForOtherContextCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "context", "symbol", "Target", "--use-snapshot"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --use-snapshot is only supported by analyze, symbols, symbol, search, packages, and context diff") {
		t.Fatalf("expected snapshot flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsAllFlagForOtherCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "symbols", "--all"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --all is only supported by deps") {
		t.Fatalf("expected all flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsScopeFlagForOtherCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "symbols", "--scope", "direct"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --scope is only supported by tests <symbol-or-package-or-file>") {
		t.Fatalf("expected scope flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsContextWithJSON(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "symbol", "Run", "--context", "--json"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --context is only supported for human output") {
		t.Fatalf("expected context JSON error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsSearchFilterFlagsForOtherCommands(t *testing.T) {
	tests := [][]string{
		{"gosherpa", "deps", ".", "--kind", "function"},
		{"gosherpa", "deps", ".", "--package", "."},
	}

	for _, test := range tests {
		t.Run(strings.Join(test, " "), func(t *testing.T) {
			result := runMainTest(t, test)

			if result.ExitCode != exitUsage {
				t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
			}

			if !strings.Contains(result.Stderr, "only supported") {
				t.Fatalf("expected search filter flag error, got:\n%s", result.Stderr)
			}

			if result.Stdout != "" {
				t.Fatalf("expected empty stdout, got %q", result.Stdout)
			}
		})
	}
}

func TestMainRejectsLimitFlagForUnsupportedCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "symbols", "--limit", "2"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --limit is only supported by search and path commands") {
		t.Fatalf("expected limit flag error, got:\n%s", result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainRejectsMaxDepthFlagForUnsupportedCommands(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "search", "user", "--max-depth", "2"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	if !strings.Contains(result.Stderr, "error: --max-depth is only supported by path commands") {
		t.Fatalf("expected max-depth flag error, got:\n%s", result.Stderr)
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

	for _, want := range []string{"IMPACT", "ParseFile (symbol)", "REFERENCES", "CALLERS", "AFFECTED PACKAGES", "SUGGESTED TESTS", "TEST PLAN", "DIRECT", "RELATED", "TestParserPackage", "TestUsesParser", "go test ./cmd/app", "go test ./internal/parser", "./cmd/app", "./internal/parser"} {
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
	testPlan := assertMainTestJSONObject(t, data, "testPlan")
	assertMainTestJSONArrayHasLength(t, testPlan, "direct", 1)
	assertMainTestJSONArrayHasLength(t, testPlan, "related", 1)

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

	for _, want := range []string{"IMPACT FILE", "CHANGED FILES", "service.go", "CHANGED PACKAGES", ".", "AFFECTED PACKAGES", "./cmd/app", "AFFECTED TESTS", "TestTarget", "TEST PLAN", "go test ."} {
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

	for _, want := range []string{"IMPACT PACKAGE", "CHANGED PACKAGES", ".", "AFFECTED PACKAGES", "./cmd/app", "AFFECTED TESTS", "TestTarget", "TEST PLAN", "go test ."} {
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

	for _, want := range []string{"IMPACT SYMBOL", "AFFECTED SYMBOLS", "Target", "AFFECTED PACKAGES", ".", "AFFECTED TESTS", "TestTarget", "TEST PLAN", "go test ."} {
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

	for _, want := range []string{"EXPLAIN", "TARGET", "Target (function)", "DEFINITION", "service.go", "PURPOSE", "RISK", "medium", "ARCHITECTURE ROLE", "leaf_dependency", "READING ORDER", "CALLED BY", "Entry", "REFERENCES", "AFFECTED PACKAGES", ".", "SUGGESTED TESTS", "TestTarget", "TEST PLAN", "go test ."} {
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
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 3)

	if data["purpose"] != "" {
		t.Fatalf("expected empty purpose, got %v", data["purpose"])
	}
	risk, ok := data["risk"].(map[string]any)
	if !ok {
		t.Fatalf("expected risk object, got %T", data["risk"])
	}
	if risk["level"] != "medium" {
		t.Fatalf("expected medium risk, got %v", risk["level"])
	}

	architectureRole, ok := data["architectureRole"].(map[string]any)
	if !ok {
		t.Fatalf("expected architectureRole object, got %T", data["architectureRole"])
	}
	if architectureRole["role"] != "leaf_dependency" {
		t.Fatalf("expected leaf_dependency role, got %v", architectureRole["role"])
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
}

func TestMainRunsExplainCommandAsJSONWithTests(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "explain", "Target", "--tests", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "explain", "Target", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "callers", 2)

	if strings.Contains(result.Stdout, "EXPLAIN") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsContextSymbolCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "symbol", "Target"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"CONTEXT", "TARGET", "Target (function)", "DEFINITION", "service.go", "SOURCE", "func Target() {}", "ANALYSIS", "Mode: typechecked+ast", "Reference analysis: typechecked", "Confidence: medium", "CALLED BY", "Entry", "REFERENCES", "AFFECTED PACKAGES", ".", "SUGGESTED TESTS", "TestTarget", "TEST PLAN", "go test .", "LIMITATIONS"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsContextSymbolCommandAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "symbol", "Target", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context symbol", "Target", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	if data["analysisMode"] != agentcontext.AnalysisModeTypecheckedAST {
		t.Fatalf("expected typechecked+ast analysis mode, got %v", data["analysisMode"])
	}
	if data["referenceAnalysisMode"] != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("expected typechecked reference analysis mode, got %v", data["referenceAnalysisMode"])
	}
	if data["confidence"] != "medium" {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}

	identity, ok := data["identity"].(map[string]any)
	if !ok {
		t.Fatalf("expected identity object, got %T", data["identity"])
	}
	if identity["package"] != "." {
		t.Fatalf("expected identity package ., got %v", identity["package"])
	}
	if identity["signature"] != "func Target()" {
		t.Fatalf("expected identity signature, got %v", identity["signature"])
	}

	sourceContext, ok := data["sourceContext"].(map[string]any)
	if !ok {
		t.Fatalf("expected sourceContext object, got %T", data["sourceContext"])
	}
	lines, ok := sourceContext["lines"].([]any)
	if !ok || len(lines) == 0 {
		t.Fatalf("expected source context lines, got %v", sourceContext["lines"])
	}

	assertMainTestJSONArrayHasLength(t, data, "references", 2)
	assertMainTestJSONArrayHasLength(t, data, "callers", 1)
	assertMainTestJSONArrayHasLength(t, data, "relatedTests", 1)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)
	limitations := assertMainTestJSONArray(t, data, "limitations")
	if !mainTestJSONArrayContainsSubstring(limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", limitations)
	}
	if !mainTestJSONArrayContainsSubstring(limitations, "Test analysis used typechecked") {
		t.Fatalf("expected test analysis limitation, got %#v", limitations)
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
}

func TestMainRunsContextSymbolCommandAsJSONWithTests(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "symbol", "Target", "--tests", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context symbol", "Target", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	assertMainTestJSONArrayHasLength(t, data, "callers", 2)
	limitations := assertMainTestJSONArray(t, data, "limitations")
	if !mainTestJSONArrayContainsSubstring(limitations, "Test analysis used typechecked") {
		t.Fatalf("expected test analysis limitation, got %#v", limitations)
	}
}

func TestMainRunsContextSymbolCommandWithLimits(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "symbol", "Target", "--tests", "--max-references", "1"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"TRUNCATED", "references:", "callers:", "reading order:"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRunsContextSymbolCommandWithLimitsAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{
		"gosherpa",
		"--root", tmp,
		"context", "symbol", "Target",
		"--tests",
		"--max-references", "1",
		"--max-tests", "1",
		"--source-radius", "0",
		"--json",
	})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context symbol", "Target", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	limits, ok := data["limits"].(map[string]any)
	if !ok {
		t.Fatalf("expected limits object, got %T", data["limits"])
	}
	if limits["maxReferences"] != float64(1) || limits["maxTests"] != float64(1) || limits["sourceRadius"] != float64(0) {
		t.Fatalf("unexpected limits: %#v", limits)
	}

	truncated, ok := data["truncated"].(map[string]any)
	if !ok {
		t.Fatalf("expected truncated object, got %T", data["truncated"])
	}
	if truncated["references"] == nil || truncated["callers"] == nil {
		t.Fatalf("expected reference and caller truncation, got %#v", truncated)
	}

	sourceContext, ok := data["sourceContext"].(map[string]any)
	if !ok {
		t.Fatalf("expected sourceContext object, got %T", data["sourceContext"])
	}
	lines := assertMainTestJSONArrayHasLength(t, sourceContext, "lines", 1)
	if lines[0].(map[string]any)["text"] != "func Target() {}" {
		t.Fatalf("expected target source line only, got %#v", lines)
	}
	assertMainTestJSONArrayHasLength(t, data, "references", 1)
	assertMainTestJSONArrayHasLength(t, data, "callers", 1)
}

func TestMainRunsContextSymbolCommandWithMaxBytesAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{
		"gosherpa",
		"--root", tmp,
		"context", "symbol", "Target",
		"--tests",
		"--max-bytes", "2400",
		"--json",
	})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context symbol", "Target", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	limits, ok := data["limits"].(map[string]any)
	if !ok {
		t.Fatalf("expected limits object, got %T", data["limits"])
	}
	if limits["maxBytes"] != float64(2400) {
		t.Fatalf("unexpected max bytes limit: %#v", limits)
	}

	truncated, ok := data["truncated"].(map[string]any)
	if !ok {
		t.Fatalf("expected truncated object, got %T", data["truncated"])
	}
	if truncated["sourceLines"] == nil && truncated["references"] == nil && truncated["callers"] == nil {
		t.Fatalf("expected byte-budget truncation, got %#v", truncated)
	}
}

func TestMainContextJSONByteBudgetContract(t *testing.T) {
	tests := []struct {
		name    string
		root    func(t *testing.T) string
		args    []string
		command string
		target  string
	}{
		{
			name: "symbol",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"context", "symbol", "Target", "--tests", "--max-bytes", "1200", "--json"},
			command: "context symbol",
			target:  "Target",
		},
		{
			name: "file",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"context", "file", "service.go", "--tests", "--max-bytes", "1200", "--json"},
			command: "context file",
			target:  "service.go",
		},
		{
			name: "package",
			root: func(t *testing.T) string {
				return writeMainImpactReportProject(t)
			},
			args:    []string{"context", "package", ".", "--tests", "--max-bytes", "1200", "--json"},
			command: "context package",
			target:  ".",
		},
		{
			name: "diff",
			root: func(t *testing.T) string {
				return writeMainPRDiffProject(t)
			},
			args:    []string{"context", "diff", "--base", "HEAD", "--tests", "--max-bytes", "1200", "--json"},
			command: "context diff",
			target:  "HEAD",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := test.root(t)
			args := append([]string{"gosherpa", "--root", root}, test.args...)
			result := runMainTest(t, args)

			if result.ExitCode != exitSuccess {
				t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
			}
			if result.Stderr != "" {
				t.Fatalf("expected empty stderr, got %q", result.Stderr)
			}

			payload := decodeMainTestJSON(t, result.Stdout)
			data := assertMainTestJSONEnvelope(t, payload, root, test.command, test.target, "example.com/app")
			assertMainTestContextJSONContract(t, payload, data)
			assertMainTestContextByteBudget(t, data, 1200)
		})
	}
}

func TestMainRunsContextSymbolCommandWithMaxBytes(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "symbol", "Target", "--max-bytes", "2400"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"TRUNCATED", "source lines:"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRejectsUnsupportedContextLimitOption(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "diff", "--base", "HEAD", "--max-references", "1"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}
	if !strings.Contains(result.Stderr, "unsupported context option for context diff: --max-references") {
		t.Fatalf("expected unsupported context option error, got %q", result.Stderr)
	}
	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainPrintsContextFileUsageWhenTargetIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "context", "file"})

	want := "usage: gosherpa [--root <path>] context file <file> [--tests] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]\n"
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

func TestMainRunsContextFileCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "file", "service.go"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"CONTEXT FILE",
		"FILE",
		"service.go",
		"Package: .",
		"Package name: service",
		"PURPOSE",
		"declares 2 supported symbols",
		"ANALYSIS",
		"Mode: typechecked+ast",
		"Confidence: medium",
		"Risk: medium",
		"FILE SYMBOLS",
		"Entry",
		"Target",
		"SOURCE",
		"func Entry()",
		"func Target() {}",
		"AFFECTED PACKAGES",
		"./cmd/app",
		"AFFECTED TESTS",
		"TestTarget",
		"TEST PLAN",
		"go test .",
		"READING ORDER",
		"File: service.go",
		"Symbol: Entry",
		"LIMITATIONS",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsContextFileCommandAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "file", "service.go", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context file", "service.go", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	if data["file"] != "service.go" {
		t.Fatalf("expected service.go file, got %v", data["file"])
	}
	if data["package"] != "." {
		t.Fatalf("expected package ., got %v", data["package"])
	}
	if data["packageName"] != "service" {
		t.Fatalf("expected package name service, got %v", data["packageName"])
	}
	if data["analysisMode"] != agentcontext.AnalysisModeTypecheckedAST {
		t.Fatalf("expected typechecked+ast analysis mode, got %v", data["analysisMode"])
	}
	if data["confidence"] != "medium" {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}

	assertMainTestJSONArrayHasLength(t, data, "symbols", 2)
	assertMainTestJSONArrayHasLength(t, data, "sourceContexts", 2)
	assertMainTestJSONArrayHasLength(t, data, "affectedPackages", 2)
	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 1)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 4)
	limitations := assertMainTestJSONArray(t, data, "limitations")
	if !mainTestJSONArrayContainsSubstring(limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", limitations)
	}
	if !mainTestJSONArrayContainsSubstring(limitations, "Test analysis used AST") {
		t.Fatalf("expected test analysis limitation, got %#v", limitations)
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "CONTEXT FILE") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsContextPackageUsageWhenTargetIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "context", "package"})

	want := "usage: gosherpa [--root <path>] context package <package> [--tests] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>] [--source-radius <n>]\n"
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

func TestMainRunsContextPackageCommand(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "package", "."})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"CONTEXT PACKAGE",
		"PACKAGE",
		".",
		"Package name: service",
		"PURPOSE",
		"declaring 3 supported symbols",
		"ANALYSIS",
		"Mode: typechecked+ast",
		"Confidence: medium",
		"Risk: medium",
		"PACKAGE FILES",
		"service.go",
		"service_test.go",
		"PACKAGE SYMBOLS",
		"Entry",
		"Target",
		"TestTarget",
		"SOURCE",
		"func Entry()",
		"func TestTarget(t *testing.T)",
		"AFFECTED PACKAGES",
		"./cmd/app",
		"AFFECTED TESTS",
		"TestTarget",
		"TEST PLAN",
		"go test .",
		"READING ORDER",
		"File: service.go",
		"Symbol: Entry",
		"LIMITATIONS",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsContextPackageCommandAsJSON(t *testing.T) {
	tmp := writeMainImpactReportProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "package", ".", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context package", ".", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	if data["package"] != "." {
		t.Fatalf("expected package ., got %v", data["package"])
	}
	if data["packageName"] != "service" {
		t.Fatalf("expected package name service, got %v", data["packageName"])
	}
	if data["analysisMode"] != agentcontext.AnalysisModeTypecheckedAST {
		t.Fatalf("expected typechecked+ast analysis mode, got %v", data["analysisMode"])
	}
	if data["confidence"] != "medium" {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}

	assertMainTestJSONArrayHasLength(t, data, "files", 2)
	assertMainTestJSONArrayHasLength(t, data, "symbols", 3)
	assertMainTestJSONArrayHasLength(t, data, "sourceContexts", 3)
	assertMainTestJSONArrayHasLength(t, data, "affectedPackages", 2)
	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 1)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 6)
	limitations := assertMainTestJSONArray(t, data, "limitations")
	if !mainTestJSONArrayContainsSubstring(limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", limitations)
	}
	if !mainTestJSONArrayContainsSubstring(limitations, "Test analysis used AST") {
		t.Fatalf("expected test analysis limitation, got %#v", limitations)
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "CONTEXT PACKAGE") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsContextDiffUsageWhenBaseIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "context", "diff"})

	want := "usage: gosherpa [--root <path>] context diff --base <ref> [--tests] [--use-snapshot] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>]\n"
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

func TestMainPrintsPRUsageWhenBaseIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "pr"})

	want := "usage: gosherpa [--root <path>] pr --base <ref>\n"
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

func TestMainRunsPRCommand(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "pr", "--base", "HEAD"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"PR REVIEW",
		"Base: HEAD",
		"Analysis: git-diff+typechecked+ast",
		"Confidence: medium",
		"Reference analysis: typechecked",
		"Call analysis: typechecked",
		"RISK",
		"Level: medium",
		"dependent packages",
		"REPOSITORY RISK",
		"Score: 3",
		"fan_in",
		"public_api",
		"CHANGED FILES",
		"internal/auth/session.go",
		"CHANGED PACKAGES",
		"./internal/auth",
		"CHANGED SYMBOLS",
		"NewSession",
		"READING ORDER",
		"Changed symbol: ./internal/auth.NewSession",
		"AFFECTED PACKAGES",
		"./internal/api",
		"AFFECTED TESTS",
		"TestSession",
		"TestHandler",
		"TEST PLAN",
		"go test ./internal/api",
		"go test ./internal/auth",
		"VERIFY",
		"go test ./...",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsPRCommandAsJSON(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "pr", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "pr", "HEAD", "example.com/app")

	if data["analysisMode"] != "git-diff+typechecked+ast" {
		t.Fatalf("expected diff analysis mode, got %v", data["analysisMode"])
	}
	if data["referenceAnalysisMode"] != "typechecked" {
		t.Fatalf("expected typechecked reference analysis mode, got %v", data["referenceAnalysisMode"])
	}
	if data["callAnalysisMode"] != "typechecked" {
		t.Fatalf("expected typechecked call analysis mode, got %v", data["callAnalysisMode"])
	}
	if data["confidence"] != "medium" {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}
	risk, ok := data["risk"].(map[string]any)
	if !ok {
		t.Fatalf("expected risk object, got %T", data["risk"])
	}
	if risk["level"] != "medium" {
		t.Fatalf("expected medium risk, got %v", risk["level"])
	}
	repositoryRisk := assertMainTestJSONObject(t, data, "repositoryRisk")
	if repositoryRisk["level"] != sherpa.RiskLevelMedium {
		t.Fatalf("expected medium repository risk, got %#v", repositoryRisk["level"])
	}
	if repositoryRisk["score"] != float64(3) {
		t.Fatalf("expected repository risk score 3, got %#v", repositoryRisk["score"])
	}

	assertMainTestJSONArrayHasLength(t, data, "changedFiles", 1)
	assertMainTestJSONArrayHasLength(t, data, "changedPackages", 1)
	assertMainTestJSONArrayHasLength(t, data, "changedSymbols", 1)
	assertMainTestJSONArrayHasLength(t, data, "changedSymbolDetails", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedPackages", 2)
	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 2)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 4)
	assertMainTestJSONArrayHasLength(t, data, "verificationCommands", 3)
	limitations := assertMainTestJSONArray(t, data, "limitations")
	if !mainTestJSONArrayContainsSubstring(limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", limitations)
	}
	if !mainTestJSONArrayContainsSubstring(limitations, "Test analysis used typechecked") {
		t.Fatalf("expected test analysis limitation, got %#v", limitations)
	}
	assertMainTestJSONArrayHasLength(t, repositoryRisk, "limitations", 3)
	assertMainTestJSONArrayHasLength(t, repositoryRisk, "factors", 4)
	assertMainTestJSONArrayHasLength(t, repositoryRisk, "packages", 2)
	assertMainTestJSONArrayHasLength(t, repositoryRisk, "cycles", 0)

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "PR REVIEW") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsContextDiffCommand(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "diff", "--base", "HEAD"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"CONTEXT DIFF",
		"BASE",
		"HEAD",
		"PURPOSE",
		"Diff changes 1 file across 1 Go package.",
		"ANALYSIS",
		"Mode: git-diff+typechecked+ast",
		"Reference analysis: typechecked",
		"Call analysis: typechecked",
		"Confidence: medium",
		"Risk: medium",
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
		"TEST PLAN",
		"go test ./internal/api",
		"go test ./internal/auth",
		"READING ORDER",
		"Changed symbol: ./internal/auth.NewSession",
		"Changed file: internal/auth/session.go",
		"LIMITATIONS",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsContextDiffCommandAsJSON(t *testing.T) {
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

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "diff", "--base", "HEAD", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context diff", "HEAD", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	if data["analysisMode"] != "git-diff+typechecked+ast" {
		t.Fatalf("expected diff analysis mode, got %v", data["analysisMode"])
	}
	if data["referenceAnalysisMode"] != "typechecked" {
		t.Fatalf("expected typechecked reference analysis mode, got %v", data["referenceAnalysisMode"])
	}
	if data["callAnalysisMode"] != "typechecked" {
		t.Fatalf("expected typechecked call analysis mode, got %v", data["callAnalysisMode"])
	}
	if data["confidence"] != "medium" {
		t.Fatalf("expected medium confidence, got %v", data["confidence"])
	}
	risk, ok := data["risk"].(map[string]any)
	if !ok {
		t.Fatalf("expected risk object, got %T", data["risk"])
	}
	if risk["level"] != "medium" {
		t.Fatalf("expected medium risk, got %v", risk["level"])
	}

	assertMainTestJSONArrayHasLength(t, data, "changedFiles", 1)
	assertMainTestJSONArrayHasLength(t, data, "changedPackages", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedPackages", 2)
	assertMainTestJSONArrayHasLength(t, data, "affectedSymbols", 1)
	assertMainTestJSONArrayHasLength(t, data, "changedSymbolDetails", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 2)
	assertMainTestJSONArrayHasLength(t, data, "testCommands", 2)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 4)
	limitations := assertMainTestJSONArray(t, data, "limitations")
	if !mainTestJSONArrayContainsSubstring(limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", limitations)
	}
	if !mainTestJSONArrayContainsSubstring(limitations, "Test analysis used typechecked") {
		t.Fatalf("expected test analysis limitation, got %#v", limitations)
	}

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}

	if strings.Contains(result.Stdout, "CONTEXT DIFF") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsContextDiffCommandFromSnapshotAsJSON(t *testing.T) {
	tmp := t.TempDir()
	initMainTestGitRepository(t, tmp)

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), "package app\n\nfunc Target() {}\n")
	runMainTestGit(t, tmp, "add", ".")
	runMainTestGit(t, tmp, "commit", "-m", "initial")

	writeMainTestFile(t, filepath.Join(tmp, "service.go"), "package app\n\nfunc Target() {}\n\nfunc Added() {}\n")
	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "context", "diff", "--base", "HEAD", "--use-snapshot", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context diff", "HEAD", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	if data["analysisMode"] != agentcontext.AnalysisModeSnapshotDiffTypechecked {
		t.Fatalf("expected snapshot diff analysis mode, got %v", data["analysisMode"])
	}
	assertMainTestJSONArrayHasLength(t, payload, "warnings", 0)
	assertMainTestJSONArrayHasLength(t, data, "changedSymbolDetails", 1)
	details := assertMainTestJSONArray(t, data, "changedSymbolDetails")
	firstDetail, ok := details[0].(map[string]any)
	if !ok || firstDetail["name"] != "Added" {
		t.Fatalf("expected Added changed symbol detail, got %#v", details)
	}
	if !mainTestJSONArrayContainsSubstring(data["limitations"].([]any), "reused a valid snapshot") {
		t.Fatalf("expected snapshot limitation, got %#v", data["limitations"])
	}
}

func TestMainRunsContextDiffCommandWithLimitsAsJSON(t *testing.T) {
	tmp := writeMainPRDiffProject(t)

	result := runMainTest(t, []string{
		"gosherpa",
		"--root", tmp,
		"context", "diff",
		"--base", "HEAD",
		"--max-files", "1",
		"--max-symbols", "1",
		"--max-tests", "1",
		"--json",
	})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "context diff", "HEAD", "example.com/app")
	assertMainTestContextJSONContract(t, payload, data)

	limits := assertMainTestJSONObject(t, data, "limits")
	if limits["maxFiles"] != float64(1) || limits["maxSymbols"] != float64(1) || limits["maxTests"] != float64(1) {
		t.Fatalf("unexpected context diff limits: %#v", limits)
	}

	truncated := assertMainTestJSONObject(t, data, "truncated")
	if truncated["affectedTests"] != float64(1) || truncated["readingOrder"] != float64(1) {
		t.Fatalf("expected affected test and reading order truncation, got %#v", truncated)
	}

	assertMainTestJSONArrayHasLength(t, data, "changedFiles", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedSymbols", 1)
	assertMainTestJSONArrayHasLength(t, data, "changedSymbolDetails", 1)
	assertMainTestJSONArrayHasLength(t, data, "affectedTests", 1)
	assertMainTestJSONArrayHasLength(t, data, "readingOrder", 3)
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
		"TEST PLAN",
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

	want := "usage: gosherpa [--root <path>] callers <function-or-method> [--tests] [--context]\n"
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

func TestMainPrintsImplementersUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "implementers"})

	want := "usage: gosherpa [--root <path>] implementers <interface>\n"
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

func TestMainRunsImplementersCommand(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "implementers", "./internal/auth.Authenticator"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"IMPLEMENTERS",
		"./internal/auth.Authenticator",
		"./internal/jwt.JWTAuthenticator",
		"internal/jwt/jwt.go",
		"Found 1 implementers",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsImplementersCommandAsJSON(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "implementers", "./internal/auth.Authenticator", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "implementers", "./internal/auth.Authenticator", "example.com/app")

	implementers := assertMainTestJSONArrayHasLength(t, data, "implementers", 1)
	implementer, ok := implementers[0].(map[string]any)
	if !ok {
		t.Fatalf("expected implementer object, got %T", implementers[0])
	}
	if implementer["name"] != "./internal/jwt.JWTAuthenticator" {
		t.Fatalf("expected JWTAuthenticator implementer, got %v", implementer["name"])
	}

	if strings.Contains(result.Stdout, "IMPLEMENTERS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsInterfaceUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "interface"})

	want := "usage: gosherpa [--root <path>] interface <interface>\n"
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

func TestMainRunsInterfaceCommand(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "interface", "./internal/auth.Authenticator"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"INTERFACE",
		"Target: ./internal/auth.Authenticator",
		"Authenticate func() error",
		"call       internal/session/session.go",
		"./internal/jwt.JWTAuthenticator",
		"type_usage internal/session/session.go",
		"Found 1 methods, 1 implementers",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsInterfaceCommandAsJSON(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "interface", "./internal/auth.Authenticator", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "interface", "./internal/auth.Authenticator", "example.com/app")

	methods := assertMainTestJSONArrayHasLength(t, data, "methods", 1)
	method, ok := methods[0].(map[string]any)
	if !ok {
		t.Fatalf("expected method object, got %T", methods[0])
	}
	if method["name"] != "Authenticate" {
		t.Fatalf("expected Authenticate method, got %v", method["name"])
	}
	usages := assertMainTestJSONArrayHasLength(t, method, "usages", 1)
	usage, ok := usages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected usage object, got %T", usages[0])
	}
	if usage["kind"] != string(sherpa.ReferenceKindCall) {
		t.Fatalf("expected call usage, got %v", usage["kind"])
	}
	assertMainTestJSONArrayHasLength(t, data, "implementers", 1)
	assertMainTestJSONArrayHasLength(t, data, "references", 2)
	assertMainTestJSONArrayHasLength(t, data, "limitations", 5)

	if strings.Contains(result.Stdout, "INTERFACE") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsInterfacesUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "interfaces"})

	want := "usage: gosherpa [--root <path>] interfaces <type>\n"
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

func TestMainRunsInterfacesCommand(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "interfaces", "./internal/jwt.JWTAuthenticator"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"INTERFACES",
		"./internal/jwt.JWTAuthenticator",
		"./internal/auth.Authenticator",
		"internal/auth/auth.go",
		"Found 1 interfaces",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsInterfacesCommandAsJSON(t *testing.T) {
	tmp := writeMainInterfaceProject(t)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "interfaces", "./internal/jwt.JWTAuthenticator", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "interfaces", "./internal/jwt.JWTAuthenticator", "example.com/app")

	interfaces := assertMainTestJSONArrayHasLength(t, data, "interfaces", 1)
	iface, ok := interfaces[0].(map[string]any)
	if !ok {
		t.Fatalf("expected interface object, got %T", interfaces[0])
	}
	if iface["name"] != "./internal/auth.Authenticator" {
		t.Fatalf("expected Authenticator interface, got %v", iface["name"])
	}

	if strings.Contains(result.Stdout, "INTERFACES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsEntryPointsUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "entrypoints"})

	want := "usage: gosherpa [--root <path>] entrypoints <function-or-method> [--tests]\n"
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

func TestMainRunsEntryPointsCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/service"

func main() {
	service.Entry()
}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Entry() {
	step()
}

func step() {
	Target()
}

func Target() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "entrypoints", "./internal/service.Target"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"ENTRYPOINTS",
		"Target: ./internal/service.Target",
		"Analysis: typechecked",
		"main",
		"exported",
		"Entry",
		"Target",
		"cmd/app/main.go",
		"internal/service/service.go",
		"Found 3 entrypoints",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsEntryPointsCommandWithTests(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "service_test.go"), `package service

func TestTarget() {
	Target()
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "entrypoints", "Target", "--tests"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"ENTRYPOINTS", "TestTarget", "test", "Target", "exported", "Found 2 entrypoints"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRunsEntryPointsCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	target()
}

func target() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "entrypoints", "target", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "entrypoints", "target", "example.com/app")

	entryPoints := assertMainTestJSONArrayHasLength(t, data, "entrypoints", 1)
	entryPoint, ok := entryPoints[0].(map[string]any)
	if !ok {
		t.Fatalf("expected entrypoint object, got %T", entryPoints[0])
	}
	if entryPoint["name"] != "Run" {
		t.Fatalf("expected Run entrypoint, got %v", entryPoint["name"])
	}
	if entryPoint["kind"] != string(sherpa.EntryPointKindExported) {
		t.Fatalf("expected exported entrypoint, got %v", entryPoint["kind"])
	}

	if strings.Contains(result.Stdout, "ENTRYPOINTS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
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

	for _, want := range []string{"CALLERS", "Step", "Analysis: typechecked", "Run", "Found 1 callers"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsCallersCommandFromGoWorkRootAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.work"), `go 1.24

use (
	./app
	./service
)
`)
	writeMainTestFile(t, filepath.Join(tmp, "service", "go.mod"), "module example.com/service\n")
	writeMainTestFile(t, filepath.Join(tmp, "service", "service.go"), `package service

type Client struct{}

func (c *Client) Start() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "app", "go.mod"), `module example.com/app

require example.com/service v0.0.0
`)
	writeMainTestFile(t, filepath.Join(tmp, "app", "main.go"), `package app

import "example.com/service"

func Run(client *service.Client) {
	client.Start()
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "callers", "./service.Client.Start", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "callers", "./service.Client.Start", "")

	if data["analysisMode"] != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %v", data["analysisMode"])
	}
	callers := assertMainTestJSONArrayHasLength(t, data, "callers", 1)
	caller, ok := callers[0].(map[string]any)
	if !ok {
		t.Fatalf("expected caller object, got %T", callers[0])
	}
	if caller["name"] != "Run" {
		t.Fatalf("expected Run caller, got %v", caller["name"])
	}
}

func TestMainRunsCallersCommandWithContext(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "callers", "Step", "--context"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"CALLERS",
		"Run",
		"service.go:4",
		"> 4 | \tStep()",
		"Found 1 callers",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsCallersCommandWithTests(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "service_test.go"), `package service

func TestStep() {
	Step()
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "callers", "Step", "--tests"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"CALLERS", "Step", "Run", "TestStep", "Found 2 callers"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsAmbiguousCallersCandidates(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "billing", "billing.go"), `package billing

func Target() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "callers", "Target"})

	if result.ExitCode != exitFailure {
		t.Fatalf("expected exit %d, got %d", exitFailure, result.ExitCode)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}

	for _, want := range []string{
		"error: ambiguous function target: Target",
		"package ./internal/auth, file internal/auth/auth.go:3, target ./internal/auth.Target",
		"package ./internal/billing, file internal/billing/billing.go:3, target ./internal/billing.Target",
		"use a package-qualified target",
		"./internal/auth.Target",
		"./internal/billing.Target",
	} {
		if !strings.Contains(result.Stderr, want) {
			t.Fatalf("expected stderr to contain %q, got:\n%s", want, result.Stderr)
		}
	}
}

func TestMainPrintsAmbiguousCallersCandidatesAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "billing", "billing.go"), `package billing

func Target() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "callers", "Target", "--json"})

	if result.ExitCode != exitFailure {
		t.Fatalf("expected exit %d, got %d", exitFailure, result.ExitCode)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}

	payload := decodeMainTestJSON(t, result.Stderr)
	if payload["schemaVersion"] != float64(jsonSchemaVersion) {
		t.Fatalf("expected schemaVersion %d, got %v", jsonSchemaVersion, payload["schemaVersion"])
	}
	if payload["command"] != "callers" {
		t.Fatalf("expected callers command, got %v", payload["command"])
	}
	if payload["target"] != "Target" {
		t.Fatalf("expected target Target, got %v", payload["target"])
	}
	if payload["root"] != filepath.Clean(tmp) {
		t.Fatalf("expected root %s, got %v", filepath.Clean(tmp), payload["root"])
	}
	if payload["modulePath"] != "example.com/app" {
		t.Fatalf("expected modulePath example.com/app, got %v", payload["modulePath"])
	}
	assertMainTestJSONArrayHasLength(t, payload, "warnings", 0)

	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %T", payload["error"])
	}
	if errorPayload["code"] != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target code, got %v", errorPayload["code"])
	}
	if errorPayload["kind"] != "function" {
		t.Fatalf("expected function kind, got %v", errorPayload["kind"])
	}
	if errorPayload["target"] != "Target" {
		t.Fatalf("expected error target Target, got %v", errorPayload["target"])
	}

	message, ok := errorPayload["message"].(string)
	if !ok || !strings.Contains(message, "ambiguous function target: Target") {
		t.Fatalf("expected ambiguous message, got %v", errorPayload["message"])
	}

	candidates := assertMainTestJSONArrayHasLength(t, errorPayload, "candidates", 2)
	assertMainTestAmbiguousCandidate(t, candidates[0], "./internal/auth", "Target", "internal/auth/auth.go", float64(3), "./internal/auth.Target")
	assertMainTestAmbiguousCandidate(t, candidates[1], "./internal/billing", "Target", "internal/billing/billing.go", float64(3), "./internal/billing.Target")
}

func TestMainAmbiguousTargetJSONContracts(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		command string
		target  string
		kind    string
	}{
		{
			name:    "symbol",
			args:    []string{"symbol", "Target", "--json"},
			command: "symbol",
			target:  "Target",
			kind:    "symbol",
		},
		{
			name:    "explain",
			args:    []string{"explain", "Target", "--json"},
			command: "explain",
			target:  "Target",
			kind:    "symbol",
		},
		{
			name:    "context symbol",
			args:    []string{"context", "symbol", "Target", "--json"},
			command: "context symbol",
			target:  "Target",
			kind:    "symbol",
		},
		{
			name:    "callers",
			args:    []string{"callers", "Target", "--json"},
			command: "callers",
			target:  "Target",
			kind:    "function",
		},
		{
			name:    "impact symbol",
			args:    []string{"impact", "symbol", "Target", "--json"},
			command: "impact symbol",
			target:  "Target",
			kind:    "symbol",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmp := writeMainAmbiguousTargetProject(t)
			args := append([]string{"gosherpa", "--root", tmp}, test.args...)
			result := runMainTest(t, args)

			assertMainTestAmbiguousTargetJSONError(t, result, tmp, test.command, test.target, test.kind)
		})
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

	if data["analysisMode"] != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %v", data["analysisMode"])
	}
	assertMainTestJSONArrayHasLength(t, data, "callers", 1)

	if strings.Contains(result.Stdout, "CALLERS") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsCalleesUsageWhenArgumentIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "callees"})

	want := "usage: gosherpa [--root <path>] callees <function-or-method> [--context]\n"
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

	for _, want := range []string{"CALLEES", "Run", "Analysis: typechecked", "Step", "Found 1 callees"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsCalleesCommandWithContext(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	result := runMainTest(t, []string{"gosherpa", "callees", "Run", "--context", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"CALLEES",
		"Step",
		"service.go:4",
		"> 4 | \tStep()",
		"Found 1 callees",
	} {
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

	if data["analysisMode"] != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %v", data["analysisMode"])
	}
	assertMainTestJSONArrayHasLength(t, data, "callees", 1)

	if strings.Contains(result.Stdout, "CALLEES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsCalleesCommandReportsDynamicLimitationsAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

type Processor interface { Process() }

func Run(processor Processor, callback func()) {
	processor.Process()
	callback()
}
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
	limitations := assertMainTestJSONArray(t, data, "limitations")

	assertMainTestStringArrayContains(t, limitations, "Interface dispatch may hide concrete call edges at service.go:6.")
	assertMainTestStringArrayContains(t, limitations, "Function value calls may hide concrete call edges at service.go:7.")
}

func TestMainPrintsRefsUsageWithoutValidatingRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	result := runMainTest(t, []string{"gosherpa", "--root", missingRoot, "refs"})

	want := "usage: gosherpa [--root <path>] refs <name> [--kind <kind>] [--context]\n"
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

func TestMainPrintsSearchUsageWithoutValidatingRoot(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	result := runMainTest(t, []string{"gosherpa", "--root", missingRoot, "search"})

	want := "usage: gosherpa [--root <path>] search <terms> [--kind <kind>] [--package <package>] [--tests] [--limit <n>] [--use-snapshot]\n"
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

func TestMainRunsSymbolsCommandFromSnapshotAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Run() {}
`)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbols", "--use-snapshot", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "symbols", "", "example.com/app")
	if data["analysisMode"] != analysisModeSnapshot {
		t.Fatalf("expected snapshot analysis mode, got %v", data["analysisMode"])
	}
	assertMainTestJSONArrayHasLength(t, data, "symbols", 1)
}

func TestMainRunsSymbolsWithFilters(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Worker struct{}

func Run() {}

func (Worker) Work() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service_test.go"), `package service

import "testing"

func TestRun(t *testing.T) {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "other", "other.go"), `package other

func TestableHelper() {}
`)

	result := runMainTest(t, []string{
		"gosherpa",
		"--root",
		tmp,
		"symbols",
		"--kind",
		"function",
		"--package",
		"./internal/service",
		"--tests",
	})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"TESTS", "TestRun", "internal/service/service_test.go"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	for _, unwanted := range []string{"\n  Run ", "\n  Worker", "TestableHelper", "internal/other"} {
		if strings.Contains(result.Stdout, unwanted) {
			t.Fatalf("expected output not to contain %s, got:\n%s", unwanted, result.Stdout)
		}
	}
}

func TestMainRunsSymbolsFiltersAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Worker struct{}

func Run() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "other", "other.go"), `package other

type Other struct{}
`)

	result := runMainTest(t, []string{
		"gosherpa",
		"--root",
		tmp,
		"symbols",
		"--kind",
		"struct",
		"--package",
		"./internal/service",
		"--json",
	})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "symbols", "", "example.com/app")
	symbols := assertMainTestJSONArrayHasLength(t, data, "symbols", 1)

	symbol, ok := symbols[0].(map[string]any)
	if !ok {
		t.Fatalf("expected symbol object, got %T", symbols[0])
	}
	if symbol["name"] != "Worker" {
		t.Fatalf("expected Worker symbol, got %v", symbol["name"])
	}
	if symbol["package"] != "./internal/service" {
		t.Fatalf("expected ./internal/service package, got %v", symbol["package"])
	}
}

func TestMainRunsSearchCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type UserRepository interface{}

func CreateUser() {}

func CreateTeam() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "search", "user", "create"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"SYMBOL SEARCH", "Query: user create", "Found 1 match", "CreateUser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	for _, unwanted := range []string{"CreateTeam", "UserRepository", tmp} {
		if strings.Contains(result.Stdout, unwanted) {
			t.Fatalf("expected output not to contain %s, got:\n%s", unwanted, result.Stdout)
		}
	}
}

func TestMainRunsSearchCommandWithFilters(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type UserRepository interface{}

func CreateUser() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service_test.go"), `package service

func TestCreateUser() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "http", "handler.go"), `package http

func CreateUserHandler() {}
`)

	result := runMainTest(t, []string{
		"gosherpa",
		"--root",
		tmp,
		"search",
		"user",
		"--kind",
		"function",
		"--package",
		"./internal/service",
		"--tests",
		"--limit",
		"1",
	})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"SYMBOL SEARCH", "Found 1 match", "TestCreateUser"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	for _, unwanted := range []string{"CreateUserHandler", "UserRepository"} {
		if strings.Contains(result.Stdout, unwanted) {
			t.Fatalf("expected output not to contain %s, got:\n%s", unwanted, result.Stdout)
		}
	}
}

func TestMainRunsSearchCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Target() {}

func TestTarget() {}
`)

	result := runMainTest(t, []string{"gosherpa", "search", "target", "--json", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "search", "target", "example.com/app")

	assertMainTestJSONArrayHasLength(t, data, "terms", 1)
	results := assertMainTestJSONArrayHasLength(t, data, "results", 2)

	first, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first result to be a JSON object, got %T", results[0])
	}

	symbol, ok := first["symbol"].(map[string]any)
	if !ok {
		t.Fatalf("expected first symbol to be a JSON object, got %T", first["symbol"])
	}

	if symbol["name"] != "Target" {
		t.Fatalf("expected first search result Target, got %v", symbol["name"])
	}

	if strings.Contains(result.Stdout, "SYMBOL SEARCH") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainSearchFallsBackWhenSnapshotIsStale(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Existing() {}
`)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Existing() {}

func NewLiveSymbol() {}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "search", "newlive", "--use-snapshot", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr for JSON warnings, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	warnings := assertMainTestJSONArrayHasLength(t, payload, "warnings", 1)
	if !strings.Contains(warnings[0].(string), "Snapshot is stale") {
		t.Fatalf("expected stale snapshot warning, got %#v", warnings)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", payload["data"])
	}
	if data["analysisMode"] != analysisModeAST {
		t.Fatalf("expected AST fallback analysis mode, got %v", data["analysisMode"])
	}

	results := assertMainTestJSONArrayHasLength(t, data, "results", 1)
	first, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected search result object, got %T", results[0])
	}
	symbol := assertMainTestJSONObject(t, first, "symbol")
	if symbol["name"] != "NewLiveSymbol" {
		t.Fatalf("expected live fallback symbol, got %v", symbol["name"])
	}
}

func TestMainRunsSymbolCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

// Run starts the service.
func Run(name string) error {
	return nil
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbol", "./internal/service.Run"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"SYMBOL",
		"Name: Run",
		"Package: ./internal/service",
		"Qualified: ./internal/service.Run",
		"Signature: func Run(name string) error",
		"DOCUMENTATION",
		"Run starts the service.",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}
}

func TestMainRunsSymbolCommandWithContext(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func helper() {}

func Run() {
	helper()
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbol", "./internal/service.Run", "--context"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"SYMBOL",
		"File: internal/service/service.go",
		"CONTEXT",
		"> 5 | func Run() {",
		"  6 | \thelper()",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
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

func TestMainRunsSymbolCommandFromSnapshotAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Run() {}
`)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "symbol", "Run", "--use-snapshot", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "symbol", "Run", "example.com/app")
	if data["analysisMode"] != analysisModeSnapshot {
		t.Fatalf("expected snapshot analysis mode, got %v", data["analysisMode"])
	}
	symbol := assertMainTestJSONObject(t, data, "symbol")
	if symbol["name"] != "Run" {
		t.Fatalf("expected Run symbol, got %v", symbol["name"])
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

	for _, want := range []string{"REFERENCES", "ParseFile", "definition", "call", "internal/service/service.go", "Found 2 references"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsRefsCommandWithKindFilter(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func ParseFile() {
}

func Run() {
	ParseFile()
}
`)

	result := runMainTest(t, []string{"gosherpa", "refs", "ParseFile", "--kind", "call", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{"REFERENCES", "ParseFile", "call", "internal/service/service.go:7", "Found 1 references"} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected output to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, "definition") {
		t.Fatalf("expected filtered output to omit definition, got:\n%s", result.Stdout)
	}
}

func TestMainRunsRefsCommandWithContext(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func ParseFile() {
}

func Run() {
	ParseFile()
}
`)

	result := runMainTest(t, []string{"gosherpa", "refs", "ParseFile", "--context", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d", exitSuccess, result.ExitCode)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"REFERENCES",
		"definition   internal/service/service.go:3",
		"> 3 | func ParseFile() {",
		"call         internal/service/service.go:7",
		"> 7 | \tParseFile()",
		"Found 2 references",
	} {
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

	references := assertMainTestJSONArrayHasLength(t, data, "references", 2)
	firstReference, ok := references[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first reference object, got %#v", references[0])
	}
	if firstReference["kind"] != string(sherpa.ReferenceKindDefinition) {
		t.Fatalf("expected first reference kind definition, got %#v", firstReference["kind"])
	}
	secondReference, ok := references[1].(map[string]any)
	if !ok {
		t.Fatalf("expected second reference object, got %#v", references[1])
	}
	if secondReference["kind"] != string(sherpa.ReferenceKindCall) {
		t.Fatalf("expected second reference kind call, got %#v", secondReference["kind"])
	}

	if strings.Contains(result.Stdout, "REFERENCES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsRefsCommandWithBuildTags(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "enterprise.go"), `//go:build enterprise

package service

func Run() {
	Target()
}
`)

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "--tags", "enterprise", "refs", "Target", "--json"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "refs", "Target", "example.com/app")

	references := assertMainTestJSONArrayHasLength(t, data, "references", 2)
	foundTaggedReference := false
	for _, reference := range references {
		item, ok := reference.(map[string]any)
		if !ok {
			t.Fatalf("expected reference object, got %#v", reference)
		}
		position, ok := item["position"].(map[string]any)
		if !ok {
			t.Fatalf("expected reference position object, got %#v", item["position"])
		}
		if position["file"] == "enterprise.go" {
			foundTaggedReference = true
		}
	}
	if !foundTaggedReference {
		t.Fatalf("expected tagged reference in enterprise.go, got %#v", references)
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

func TestMainRunsDepsAllCommand(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package app

import (
	"fmt"

	"example.com/app/internal/parser"
)
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app"

func Run() {}
`)

	result := runMainTest(t, []string{"gosherpa", "deps", "--all", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	for _, want := range []string{
		"DEPENDENCIES",
		".",
		"./cmd/app",
		"./internal/parser",
		"LOCAL IMPORTS",
		"./internal/parser",
		"EXTERNAL IMPORTS",
		"fmt",
		"USED BY",
		"Found 3 packages",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected stdout to contain %s, got:\n%s", want, result.Stdout)
		}
	}

	if strings.Contains(result.Stdout, tmp) {
		t.Fatalf("expected root-relative output, got:\n%s", result.Stdout)
	}
}

func TestMainRunsDepsAllCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package app

import "example.com/app/internal/parser"
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app"
`)

	result := runMainTest(t, []string{"gosherpa", "deps", "--all", "--json", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "deps", "all", "example.com/app")
	packages := assertMainTestJSONArrayHasLength(t, data, "packages", 3)

	rootPackage, ok := packages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected package object, got %#v", packages[0])
	}
	if rootPackage["package"] != "." {
		t.Fatalf("expected root package first, got %#v", rootPackage["package"])
	}
	assertMainTestJSONArrayHasLength(t, rootPackage, "localImports", 1)
	assertMainTestJSONArrayHasLength(t, rootPackage, "externalImports", 0)
	assertMainTestJSONArrayHasLength(t, rootPackage, "usedBy", 1)

	if strings.Contains(result.Stdout, "DEPENDENCIES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainPrintsDepsUsageWhenAllHasArguments(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "deps", "--all", "."})

	want := "usage: gosherpa [--root <path>] deps <package>\n" +
		"       gosherpa [--root <path>] deps --all\n"
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

func TestMainRunsPackagesCommandAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package app

import "example.com/app/internal/parser"

type Service struct{}

func Run() {
	parser.ParseFile()
}
`)
	writeMainTestFile(t, filepath.Join(tmp, "service_test.go"), `package app

import "testing"

func TestRun(t *testing.T) {
	Run()
}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "parser", "parser.go"), `package parser

func ParseFile() {}
`)

	result := runMainTest(t, []string{"gosherpa", "packages", "--tests", "--json", "--root", tmp})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "packages", "", "example.com/app")
	packages := assertMainTestJSONArrayHasLength(t, data, "packages", 2)

	rootPackage, ok := packages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected package object, got %#v", packages[0])
	}
	if rootPackage["package"] != "." {
		t.Fatalf("expected root package first, got %#v", rootPackage["package"])
	}
	if rootPackage["symbols"] != float64(3) {
		t.Fatalf("expected test-inclusive symbol count 3, got %#v", rootPackage["symbols"])
	}
	if rootPackage["hasTests"] != true {
		t.Fatalf("expected hasTests true, got %#v", rootPackage["hasTests"])
	}

	if strings.Contains(result.Stdout, "PACKAGES") {
		t.Fatalf("expected JSON-only stdout, got:\n%s", result.Stdout)
	}
}

func TestMainRunsPackagesCommandFromSnapshotAsJSON(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package app

func Run() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "service_test.go"), `package app

import "testing"

func TestRun(t *testing.T) {
	Run()
}
`)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "packages", "--tests", "--use-snapshot", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	data := assertMainTestJSONEnvelope(t, payload, tmp, "packages", "", "example.com/app")
	if data["analysisMode"] != analysisModeSnapshot {
		t.Fatalf("expected snapshot analysis mode, got %v", data["analysisMode"])
	}

	packages := assertMainTestJSONArrayHasLength(t, data, "packages", 1)
	rootPackage, ok := packages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected package object, got %#v", packages[0])
	}
	if rootPackage["symbols"] != float64(2) {
		t.Fatalf("expected test-inclusive snapshot symbol count 2, got %#v", rootPackage["symbols"])
	}
}

func TestMainPackagesWithoutTestsFallsBackFromSnapshot(t *testing.T) {
	tmp := t.TempDir()

	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "service.go"), `package app

func Run() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "service_test.go"), `package app

import "testing"

func TestRun(t *testing.T) {
	Run()
}
`)

	snapshotResult := runMainTest(t, []string{"gosherpa", "--root", tmp, "snapshot"})
	if snapshotResult.ExitCode != exitSuccess {
		t.Fatalf("expected snapshot success, got %d\nstderr:\n%s", snapshotResult.ExitCode, snapshotResult.Stderr)
	}

	result := runMainTest(t, []string{"gosherpa", "--root", tmp, "packages", "--use-snapshot", "--json"})
	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("expected empty stderr for JSON warnings, got %q", result.Stderr)
	}

	payload := decodeMainTestJSON(t, result.Stdout)
	warnings := assertMainTestJSONArrayHasLength(t, payload, "warnings", 1)
	if !strings.Contains(warnings[0].(string), "production-only package counts") {
		t.Fatalf("expected production-count fallback warning, got %#v", warnings)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", payload["data"])
	}
	if data["analysisMode"] != analysisModeAST {
		t.Fatalf("expected AST fallback analysis mode, got %v", data["analysisMode"])
	}

	packages := assertMainTestJSONArrayHasLength(t, data, "packages", 1)
	rootPackage, ok := packages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected package object, got %#v", packages[0])
	}
	if rootPackage["symbols"] != float64(1) {
		t.Fatalf("expected production-only symbol count 1, got %#v", rootPackage["symbols"])
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

func TestMainPrintsZshCompletion(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "completion", "zsh"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	for _, want := range []string{
		"#compdef gosherpa",
		"'completion:completion zsh|bash|fish'",
		"_describe -t context-subcommands 'context target' context_subcommands",
		"'--use-snapshot[reuse a valid repository snapshot]'",
		"'--scope[filter test scope]:scope:(direct related all)'",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected zsh completion to contain %q, got:\n%s", want, result.Stdout)
		}
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}
}

func TestMainPrintsBashCompletion(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "completion", "bash"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	for _, want := range []string{
		"_gosherpa()",
		"complete -F _gosherpa gosherpa",
		"completion)",
		"COMPREPLY=( $(compgen -W \"zsh bash fish\" -- \"$cur\") )",
		"printf '%s\\n' '--root --json --tests --use-snapshot --tags'",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected bash completion to contain %q, got:\n%s", want, result.Stdout)
		}
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}
}

func TestMainPrintsFishCompletion(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "completion", "fish"})

	if result.ExitCode != exitSuccess {
		t.Fatalf("expected exit %d, got %d\nstderr:\n%s", exitSuccess, result.ExitCode, result.Stderr)
	}

	for _, want := range []string{
		"function __fish_gosherpa_seen_command",
		"complete -c gosherpa -f -n 'not __fish_gosherpa_seen_command' -a 'completion' -d 'completion zsh|bash|fish'",
		"complete -c gosherpa -f -n '__fish_seen_subcommand_from completion' -a 'zsh' -d 'zsh completion script'",
		"complete -c gosherpa -n '__fish_seen_subcommand_from symbols' -l kind -r -a 'struct interface alias function method definition call type_usage field_access usage'",
	} {
		if !strings.Contains(result.Stdout, want) {
			t.Fatalf("expected fish completion to contain %q, got:\n%s", want, result.Stdout)
		}
	}

	if result.Stderr != "" {
		t.Fatalf("expected empty stderr, got %q", result.Stderr)
	}
}

func TestMainPrintsCompletionUsageWhenShellIsMissing(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "completion"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	want := "usage: gosherpa [--root <path>] completion zsh|bash|fish\n"
	if result.Stderr != want {
		t.Fatalf("expected completion usage %q, got %q", want, result.Stderr)
	}

	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}
}

func TestMainPrintsCompletionUsageWhenShellIsUnsupported(t *testing.T) {
	result := runMainTest(t, []string{"gosherpa", "completion", "powershell"})

	if result.ExitCode != exitUsage {
		t.Fatalf("expected exit %d, got %d", exitUsage, result.ExitCode)
	}

	for _, want := range []string{
		"error: unsupported completion shell: powershell",
		"usage: gosherpa [--root <path>] completion zsh|bash|fish",
	} {
		if !strings.Contains(result.Stderr, want) {
			t.Fatalf("expected completion error to contain %q, got:\n%s", want, result.Stderr)
		}
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

func writeMainAmbiguousTargetProject(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "billing", "billing.go"), `package billing

func Target() {}
`)

	return tmp
}

func writeMainPRDiffProject(t *testing.T) string {
	t.Helper()

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

	return tmp
}

func writeMainInterfaceProject(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	writeMainTestFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeMainTestFile(t, filepath.Join(tmp, "internal", "auth", "auth.go"), `package auth

type Authenticator interface {
	Authenticate() error
}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "jwt", "jwt.go"), `package jwt

type JWTAuthenticator struct{}

func (JWTAuthenticator) Authenticate() error {
	return nil
}
`)
	writeMainTestFile(t, filepath.Join(tmp, "internal", "session", "session.go"), `package session

import "example.com/app/internal/auth"

func Run(authenticator auth.Authenticator) error {
	return authenticator.Authenticate()
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

	if warnings := assertMainTestJSONArray(t, payload, "warnings"); len(warnings) != 0 {
		t.Fatalf("expected warnings length 0, got %d: %#v", len(warnings), warnings)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a JSON object, got %T", payload["data"])
	}

	return data
}

func assertMainTestContextJSONContract(t *testing.T, payload map[string]any, data map[string]any) {
	t.Helper()

	if _, ok := data["warnings"]; ok {
		t.Fatalf("expected context warnings to live on the JSON envelope, got data warnings: %v", data["warnings"])
	}
	assertMainTestJSONArray(t, payload, "warnings")

	analysisMode, ok := data["analysisMode"].(string)
	if !ok || strings.TrimSpace(analysisMode) == "" {
		t.Fatalf("expected non-empty context analysisMode string, got %#v", data["analysisMode"])
	}

	confidence, ok := data["confidence"].(string)
	if !ok || (confidence != agentcontext.ConfidenceMedium && confidence != agentcontext.ConfidenceLow) {
		t.Fatalf("expected context confidence %q or %q, got %#v", agentcontext.ConfidenceMedium, agentcontext.ConfidenceLow, data["confidence"])
	}

	risk := assertMainTestJSONObject(t, data, "risk")
	if level, ok := risk["level"].(string); !ok || strings.TrimSpace(level) == "" {
		t.Fatalf("expected non-empty context risk level, got %#v", risk["level"])
	}
	assertMainTestJSONArray(t, risk, "reasons")

	assertMainTestJSONArray(t, data, "testCommands")
	assertMainTestJSONArray(t, data, "limitations")
	testPlan := assertMainTestJSONObject(t, data, "testPlan")
	assertMainTestJSONArray(t, testPlan, "direct")
	assertMainTestJSONArray(t, testPlan, "related")
	assertMainTestJSONArray(t, testPlan, "contracts")
	assertMainTestJSONArray(t, testPlan, "callerPackages")
	assertMainTestJSONArray(t, testPlan, "fallback")
	assertMainTestJSONArray(t, data, "readingOrder")
}

func assertMainTestContextByteBudget(t *testing.T, data map[string]any, maxBytes int) {
	t.Helper()

	limits := assertMainTestJSONObject(t, data, "limits")
	if limits["maxBytes"] != float64(maxBytes) {
		t.Fatalf("expected maxBytes %d, got %#v", maxBytes, limits)
	}

	truncated := assertMainTestJSONObject(t, data, "truncated")
	if len(truncated) == 0 {
		t.Fatal("expected non-empty truncated object for byte-budgeted context")
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("expected context data to encode as JSON: %v", err)
	}
	if len(encoded) > maxBytes {
		overage, ok := truncated["byteBudgetOverage"].(float64)
		if !ok || overage <= 0 {
			t.Fatalf("context data is %d bytes with budget %d but byteBudgetOverage is missing: %#v", len(encoded), maxBytes, truncated)
		}
	}
}

func assertMainTestJSONArray(t *testing.T, payload map[string]any, key string) []any {
	t.Helper()

	values, ok := payload[key].([]any)
	if !ok {
		t.Fatalf("expected %s to be a JSON array, got %T", key, payload[key])
	}

	return values
}

func assertMainTestJSONArrayHasLength(t *testing.T, payload map[string]any, key string, length int) []any {
	t.Helper()

	values := assertMainTestJSONArray(t, payload, key)

	if len(values) != length {
		t.Fatalf("expected %s length %d, got %d", key, length, len(values))
	}

	return values
}

func assertMainTestStringArrayContains(t *testing.T, values []any, want string) {
	t.Helper()

	for _, value := range values {
		if value == want {
			return
		}
	}

	t.Fatalf("expected JSON array to contain %q, got %#v", want, values)
}

func assertMainTestJSONObject(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := payload[key].(map[string]any)
	if !ok {
		t.Fatalf("expected %s to be a JSON object, got %T", key, payload[key])
	}

	return value
}

func assertMainTestAmbiguousCandidate(t *testing.T, value any, packagePath string, symbol string, file string, line float64, example string) {
	t.Helper()

	candidate, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected candidate object, got %T", value)
	}
	if candidate["package"] != packagePath {
		t.Fatalf("expected package %s, got %v", packagePath, candidate["package"])
	}
	if candidate["symbol"] != symbol {
		t.Fatalf("expected symbol %s, got %v", symbol, candidate["symbol"])
	}
	if candidate["example"] != example {
		t.Fatalf("expected example %s, got %v", example, candidate["example"])
	}

	position, ok := candidate["position"].(map[string]any)
	if !ok {
		t.Fatalf("expected position object, got %T", candidate["position"])
	}
	if position["file"] != file {
		t.Fatalf("expected file %s, got %v", file, position["file"])
	}
	if position["line"] != line {
		t.Fatalf("expected line %v, got %v", line, position["line"])
	}
}

func assertMainTestAmbiguousTargetJSONError(t *testing.T, result mainTestRunResult, root string, command string, target string, kind string) {
	t.Helper()

	if result.ExitCode != exitFailure {
		t.Fatalf("expected exit %d, got %d\nstdout:\n%s\nstderr:\n%s", exitFailure, result.ExitCode, result.Stdout, result.Stderr)
	}
	if result.Stdout != "" {
		t.Fatalf("expected empty stdout, got %q", result.Stdout)
	}

	payload := decodeMainTestJSON(t, result.Stderr)
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
	if payload["modulePath"] != "example.com/app" {
		t.Fatalf("expected modulePath example.com/app, got %v", payload["modulePath"])
	}
	assertMainTestJSONArrayHasLength(t, payload, "warnings", 0)
	if _, ok := payload["data"]; ok {
		t.Fatalf("expected ambiguous error response not to include data, got %v", payload["data"])
	}

	errorPayload := assertMainTestJSONObject(t, payload, "error")
	if errorPayload["code"] != "ambiguous_target" {
		t.Fatalf("expected ambiguous_target code, got %v", errorPayload["code"])
	}
	if errorPayload["kind"] != kind {
		t.Fatalf("expected %s kind, got %v", kind, errorPayload["kind"])
	}
	if errorPayload["target"] != target {
		t.Fatalf("expected error target %s, got %v", target, errorPayload["target"])
	}

	message, ok := errorPayload["message"].(string)
	if !ok || !strings.Contains(message, "ambiguous "+kind+" target: "+target) {
		t.Fatalf("expected ambiguous message for %s %s, got %v", kind, target, errorPayload["message"])
	}

	candidates := assertMainTestJSONArrayHasLength(t, errorPayload, "candidates", 2)
	assertMainTestAmbiguousCandidate(t, candidates[0], "./internal/auth", "Target", "internal/auth/auth.go", float64(3), "./internal/auth.Target")
	assertMainTestAmbiguousCandidate(t, candidates[1], "./internal/billing", "Target", "internal/billing/billing.go", float64(3), "./internal/billing.Target")
}
