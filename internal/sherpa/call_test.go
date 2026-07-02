package sherpa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/supertabaluga/gosherpa/internal/semantics"
)

func TestNormalizeCallTarget(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "function", input: "ParseFile", want: "ParseFile"},
		{name: "trimmed function", input: " ParseFile ", want: "ParseFile"},
		{name: "method", input: "Server.Start", want: "Server.Start"},
		{name: "cache method", input: "Cache.Get", want: "Cache.Get"},
		{name: "package-like method", input: "sherpa.ParseFile", want: "sherpa.ParseFile"},
		{name: "package-qualified function", input: "./internal/sherpa.ParseFile", want: "./internal/sherpa.ParseFile"},
		{name: "package-qualified method", input: "./internal/sherpa.Server.Start", want: "./internal/sherpa.Server.Start"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeCallTarget("", test.input)
			if err != nil {
				t.Fatal(err)
			}

			if got.String() != test.want {
				t.Fatalf("expected %s, got %s", test.want, got.String())
			}
		})
	}
}

func TestNormalizeCallTargetDisplaysModuleRootPackageTarget(t *testing.T) {
	tmp := t.TempDir()
	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")

	got, err := normalizeCallTarget(tmp, "example.com/app.Run")
	if err != nil {
		t.Fatal(err)
	}

	if got.Package != "." {
		t.Fatalf("expected root package, got %s", got.Package)
	}
	if got.String() != "Run" {
		t.Fatalf("expected Run, got %s", got.String())
	}
}

func TestNormalizeCallTargetRejectsInvalidInput(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"Server.",
		".Start",
		"A.B.C",
		"github.com/example/app.ParseFile",
		`C:\repo\Run`,
		"not valid",
		"(*Server).Start",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := normalizeCallTarget("", test)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestFunctionTargetName(t *testing.T) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, "targets.go", `package sample

func ParseFile() {}

type Server struct{}

func (s Server) Start() {}

func (s *Server) Stop() {}

type Cache[T any] struct{}

func (c Cache[T]) Get() {}
`, 0)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, decl := range parsedFile.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok {
			got = append(got, functionTargetName(funcDecl))
		}
	}

	want := []string{"ParseFile", "Server.Start", "Server.Stop", "Cache.Get"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCallName(t *testing.T) {
	calls := parseCallTestCalls(t, `package sample

func Calls() {
	ParseFile()
	parser.ParseFile()
	fileSet.Position()
	NewCache[int]()
	pkg.NewCache[int]()
}
`)

	got := callTestNames(calls)
	want := []string{
		"ParseFile",
		"parser.ParseFile",
		"fileSet.Position",
		"NewCache",
		"pkg.NewCache",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCallMatchesTarget(t *testing.T) {
	tests := []struct {
		name       string
		calleeName string
		target     string
		want       bool
	}{
		{name: "exact function", calleeName: "Step", target: "Step", want: true},
		{name: "selector function", calleeName: "sherpa.ParseFile", target: "ParseFile", want: true},
		{name: "different function", calleeName: "Start", target: "Stop", want: false},
		{name: "exact method expression", calleeName: "Server.Start", target: "Server.Start", want: true},
		{name: "receiver variable does not match method target", calleeName: "server.Start", target: "Server.Start", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := callMatchesTarget(test.calleeName, test.target)
			if got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestCallNameHandlesParenthesizedAndMethodExpressionCalls(t *testing.T) {
	calls := parseCallTestCalls(t, `package sample

type Server struct{}

func Calls(fn func()) {
	(fn)()
	(*Server).Start()
}
`)

	got := callTestNames(calls)
	want := []string{"fn", "Server.Start"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCollectCalleesFromFunction(t *testing.T) {
	function := parseCallTestFunction(t, `package sample

func Run() {
	Start()
	defer Cleanup()
	go Worker()
	fmt.Println("ready")
}
`, "Run")

	got := collectCalleesFromFunction(function)
	names := callTestCalleeNames(got)

	assertContainsString(t, names, "Start")
	assertContainsString(t, names, "Cleanup")
	assertContainsString(t, names, "Worker")
	assertContainsString(t, names, "fmt.Println")
}

func TestCollectCalleesFromFunctionIgnoresFunctionLiterals(t *testing.T) {
	function := parseCallTestFunction(t, `package sample

func Run() {
	Start()
	fn := func() {
		Hidden()
	}
	fn()
}
`, "Run")

	got := collectCalleesFromFunction(function)
	names := callTestCalleeNames(got)

	assertContainsString(t, names, "Start")
	assertContainsString(t, names, "fn")

	if containsString(names, "Hidden") {
		t.Fatal("expected Hidden to be ignored")
	}
}

func TestCollectCallersFromFunctions(t *testing.T) {
	function := parseCallTestFunction(t, `package sample

func Run() {
	Step()
}
`, "Run")

	got := collectCallersFromFunctions([]functionInfo{function}, callTarget{Name: "Step"})

	if len(got) != 1 {
		t.Fatalf("expected 1 caller, got %v", got)
	}

	if got[0].Name != "Run" {
		t.Fatalf("expected Run, got %s", got[0].Name)
	}
}

func TestCollectCallersFromFunctionsMatchesSelectorFunctionTarget(t *testing.T) {
	function := parseCallTestFunction(t, `package sample

func Run() {
	sherpa.ParseFile()
}
`, "Run")

	got := collectCallersFromFunctions([]functionInfo{function}, callTarget{Name: "ParseFile"})

	if len(got) != 1 {
		t.Fatalf("expected 1 caller, got %v", got)
	}

	if got[0].Name != "Run" {
		t.Fatalf("expected Run, got %s", got[0].Name)
	}
}

func TestCollectCallersFromFunctionsIgnoresFunctionLiterals(t *testing.T) {
	function := parseCallTestFunction(t, `package sample

func Run() {
	fn := func() {
		Step()
	}
	fn()
}
`, "Run")

	got := collectCallersFromFunctions([]functionInfo{function}, callTarget{Name: "Step"})

	if len(got) != 0 {
		t.Fatalf("expected no callers, got %v", got)
	}
}

func TestCollectTransitiveCallersFromFunctions(t *testing.T) {
	functions := parseCallTestFunctions(t, `package sample

func Entry() {
	Mid()
}

func Mid() {
	Step()
}

func Step() {}
`)

	got, err := collectTransitiveCallersFromFunctions(functions, callTarget{Name: "Step"})
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(got)
	want := []string{"Entry", "Mid"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected %v, got %v", want, names)
	}
}

func TestFindEntryPointsFindsMainExportedAndTargetEntrypoints(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/service"

func main() {
	service.Entry()
}
`)
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Entry() {
	step()
}

func step() {
	Target()
}

func Target() {}
`)

	result, err := FindEntryPoints(tmp, "./internal/service.Target")
	if err != nil {
		t.Fatal(err)
	}

	got := callTestEntryPointLabels(result.EntryPoints)
	want := []string{
		"main:./cmd/app:main",
		"exported:./internal/service:Entry",
		"exported:./internal/service:Target",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	if result.AnalysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %s", result.AnalysisMode)
	}
	if result.EntryPoints[0].Position.File != "cmd/app/main.go" {
		t.Fatalf("expected root-relative main position, got %s", result.EntryPoints[0].Position.File)
	}
	assertSourceRange(t, result.EntryPoints[0].Range, "cmd/app/main.go", 5, 1, 5, 10)
}

func TestFindEntryPointsFindsNoLocalCaller(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func run() {
	target()
}

func target() {}
`)

	result, err := FindEntryPoints(tmp, "target")
	if err != nil {
		t.Fatal(err)
	}

	got := callTestEntryPointLabels(result.EntryPoints)
	want := []string{"no-local-callers:.:run"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindEntryPointsWithOptionsIncludesTestEntryPoints(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func target() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

func TestTarget() {
	target()
}
`)

	withoutTests, err := FindEntryPoints(tmp, "target")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(callTestEntryPointLabels(withoutTests.EntryPoints), []string{"no-local-callers:.:target"}) {
		t.Fatalf("expected target fallback without tests, got %#v", withoutTests.EntryPoints)
	}

	withTests, err := FindEntryPointsWithOptions(tmp, "target", CallOptions{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}

	got := callTestEntryPointLabels(withTests.EntryPoints)
	want := []string{"test:.:TestTarget"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestCollectCalleesFromFunctionReturnsEmptyForNilBody(t *testing.T) {
	got := collectCalleesFromFunction(functionInfo{
		Decl: &ast.FuncDecl{},
	})

	if len(got) != 0 {
		t.Fatalf("expected no callees, got %v", got)
	}
}

func TestSortCallees(t *testing.T) {
	callees := []Callee{
		{Name: "Beta", Position: Position{File: "b.go", Line: 1}},
		{Name: "Beta", Position: Position{File: "a.go", Line: 2}},
		{Name: "Gamma", Position: Position{File: "a.go", Line: 1}},
		{Name: "Alpha", Position: Position{File: "a.go", Line: 1}},
	}

	sortCallees(callees)

	got := []string{
		callees[0].Name,
		callees[1].Name,
		callees[2].Name,
		callees[3].Name,
	}
	want := []string{"Alpha", "Gamma", "Beta", "Beta"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestSortCallers(t *testing.T) {
	callers := []Caller{
		{Name: "Beta", Position: Position{File: "b.go", Line: 1}},
		{Name: "Beta", Position: Position{File: "a.go", Line: 2}},
		{Name: "Gamma", Position: Position{File: "a.go", Line: 1}},
		{Name: "Alpha", Position: Position{File: "a.go", Line: 1}},
	}

	sortCallers(callers)

	got := []string{
		callers[0].Name,
		callers[1].Name,
		callers[2].Name,
		callers[3].Name,
	}
	want := []string{"Alpha", "Gamma", "Beta", "Beta"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindCalleesFindsTopLevelFunctionCallees(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Start()
	Stop()
}

func Start() {}

func Stop() {}
`)

	result, err := FindCallees(tmp, "Run")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCalleeNames(result.Callees)
	assertContainsString(t, names, "Start")
	assertContainsString(t, names, "Stop")
}

func TestFindCalleesReturnsRootRelativeFilePositions(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	result, err := FindCallees(tmp, "Run")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Callees) != 1 {
		t.Fatalf("expected 1 callee, got %v", result.Callees)
	}

	got := result.Callees[0].Position.File
	if got != "internal/service/service.go" {
		t.Fatalf("expected internal/service/service.go, got %s", got)
	}

	assertSourceRange(t, result.Callees[0].Range, "internal/service/service.go", 4, 2, 4, 6)

	if strings.Contains(got, tmp) {
		t.Fatalf("expected root-relative path, got %s", got)
	}
}

func TestFindCalleesFindsMethodCallees(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func (s *Server) Start() {
	Listen()
}

func Listen() {}
`)

	result, err := FindCallees(tmp, "Server.Start")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCalleeNames(result.Callees)
	assertContainsString(t, names, "Listen")
}

func TestFindCalleesReportsReceiverVariableMethodCallees(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func Run(server *Server) {
	server.Start()
}

func (s *Server) Start() {}
`)

	result, err := FindCallees(tmp, "Run")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCalleeNames(result.Callees)
	if !reflect.DeepEqual(names, []string{"Server.Start"}) {
		t.Fatalf("expected receiver method callee, got %v", names)
	}
}

func TestFindCalleesReturnsErrorForMissingFunction(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {}
`)

	_, err := FindCallees(tmp, "Missing")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "function not found: Missing") {
		t.Fatalf("expected missing function error, got %v", err)
	}
}

func TestFindCalleesReturnsErrorForAmbiguousTarget(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "one", "service.go"), `package one

func Run() {}
`)
	writeFile(t, filepath.Join(tmp, "two", "service.go"), `package two

func Run() {}
`)

	_, err := FindCallees(tmp, "Run")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "ambiguous function target: Run") {
		t.Fatalf("expected ambiguous function error, got %v", err)
	}
	for _, want := range []string{
		"package ./one, file one/service.go:3, target ./one.Run",
		"package ./two, file two/service.go:3, target ./two.Run",
		"use a package-qualified target",
		"./one.Run",
		"./two.Run",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected ambiguous function error to contain %q, got:\n%v", want, err)
		}
	}
}

func TestFindCalleesUsesPackageQualifiedTarget(t *testing.T) {
	tmp := writePackageQualifiedCallProject(t)

	result, err := FindCallees(tmp, "./internal/auth.Target")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "./internal/auth.Target" {
		t.Fatalf("expected normalized target, got %s", result.Target)
	}

	names := callTestCalleeNames(result.Callees)
	if !reflect.DeepEqual(names, []string{"Helper"}) {
		t.Fatalf("expected auth Helper only, got %v", names)
	}

	files := callTestCalleeFiles(result.Callees)
	if !reflect.DeepEqual(files, []string{"internal/auth/auth.go"}) {
		t.Fatalf("expected auth callee file only, got %v", files)
	}
}

func TestFindCalleesUsesSemanticLoaderForCrossPackageReceiverMethodCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Client struct{}

func (c *Client) Start() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/service"

func Run(client *service.Client) {
	client.Start()
}
`)

	result, err := FindCallees(tmp, "./cmd/app.Run")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCalleeNames(result.Callees)
	if !reflect.DeepEqual(names, []string{"Client.Start"}) {
		t.Fatalf("expected semantic receiver method callee, got %v", names)
	}
}

func TestFindCallsReportTypecheckedAnalysisMode(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Target()
}

func Target() {}
`)

	callees, err := FindCallees(tmp, "Run")
	if err != nil {
		t.Fatal(err)
	}
	if callees.AnalysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked callee analysis mode, got %q", callees.AnalysisMode)
	}
	if len(callees.Warnings) != 0 {
		t.Fatalf("expected no callee warnings, got %v", callees.Warnings)
	}

	callers, err := FindCallers(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}
	if callers.AnalysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked caller analysis mode, got %q", callers.AnalysisMode)
	}
	if len(callers.Warnings) != 0 {
		t.Fatalf("expected no caller warnings, got %v", callers.Warnings)
	}
}

func TestFindCallersWithOptionsHonorsBuildTags(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "enterprise.go"), `//go:build enterprise

package service

func Run() {
	Target()
}
`)

	withoutTags, err := FindCallersWithOptions(tmp, "Target", CallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutTags.Callers) != 0 {
		t.Fatalf("expected no callers without tag, got %#v", withoutTags.Callers)
	}

	withTags, err := FindCallersWithOptions(tmp, "Target", CallOptions{
		BuildTags: []string{"enterprise"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if withTags.AnalysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %s", withTags.AnalysisMode)
	}
	names := callTestCallerNames(withTags.Callers)
	if !reflect.DeepEqual(names, []string{"Run"}) {
		t.Fatalf("expected tagged caller, got %#v", withTags.Callers)
	}
}

func TestFindCalleesIgnoresTestFiles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

func TestRun() {}
`)

	_, err := FindCallees(tmp, "TestRun")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "function not found: TestRun") {
		t.Fatalf("expected missing function error, got %v", err)
	}
}

func TestFindCalleesWrapsParseErrorsWithFilePath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "broken.go")

	writeFile(t, path, `package service

func Run(
`)

	_, err := FindCallees(tmp, "Run")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "parse ") {
		t.Fatalf("expected parse prefix, got %v", err)
	}

	if !strings.Contains(err.Error(), path) {
		t.Fatalf("expected error to contain %s, got %v", path, err)
	}
}

func TestFindCallersFindsTopLevelFunctionCallers(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Stop() {
	Step()
}

func Step() {}
`)

	result, err := FindCallers(tmp, "Step")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	assertContainsString(t, names, "Run")
	assertContainsString(t, names, "Stop")
}

func TestFindCallersReturnsRootRelativeFilePositions(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)

	result, err := FindCallers(tmp, "Step")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Callers) != 1 {
		t.Fatalf("expected 1 caller, got %v", result.Callers)
	}

	got := result.Callers[0].Position.File
	if got != "internal/service/service.go" {
		t.Fatalf("expected internal/service/service.go, got %s", got)
	}

	assertSourceRange(t, result.Callers[0].Range, "internal/service/service.go", 4, 2, 4, 6)

	if strings.Contains(got, tmp) {
		t.Fatalf("expected root-relative path, got %s", got)
	}
}

func TestFindCallersFindsMethodExpressionCallers(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func Run(server *Server) {
	(*Server).Start(server)
}

func (s *Server) Start() {}
`)

	result, err := FindCallers(tmp, "Server.Start")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	assertContainsString(t, names, "Run")
}

func TestFindCallersMatchesReceiverVariableCallsToMethodTargets(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func Run(server *Server) {
	server.Start()
}

func (s *Server) Start() {}
`)

	result, err := FindCallers(tmp, "Server.Start")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	if !reflect.DeepEqual(names, []string{"Run"}) {
		t.Fatalf("expected receiver method caller, got %v", names)
	}
}

func TestFindCallersReturnsEmptyWhenTargetHasNoCallers(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {}
`)

	result, err := FindCallers(tmp, "Run")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Callers) != 0 {
		t.Fatalf("expected no callers, got %v", result.Callers)
	}
}

func TestFindCallersReturnsErrorForMissingFunction(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {}
`)

	_, err := FindCallers(tmp, "Missing")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "function not found: Missing") {
		t.Fatalf("expected missing function error, got %v", err)
	}
}

func TestFindCallersReturnsErrorForAmbiguousTarget(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "one", "service.go"), `package one

func Run() {}
`)
	writeFile(t, filepath.Join(tmp, "two", "service.go"), `package two

func Run() {}
`)

	_, err := FindCallers(tmp, "Run")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "ambiguous function target: Run") {
		t.Fatalf("expected ambiguous function error, got %v", err)
	}
	for _, want := range []string{
		"package ./one, file one/service.go:3, target ./one.Run",
		"package ./two, file two/service.go:3, target ./two.Run",
		"use a package-qualified target",
		"./one.Run",
		"./two.Run",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected ambiguous function error to contain %q, got:\n%v", want, err)
		}
	}
}

func TestFindCallersUsesPackageQualifiedTarget(t *testing.T) {
	tmp := writePackageQualifiedCallProject(t)

	result, err := FindCallers(tmp, "./internal/auth.Target")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "./internal/auth.Target" {
		t.Fatalf("expected normalized target, got %s", result.Target)
	}

	names := callTestCallerNames(result.Callers)
	wantNames := []string{"Run", "Entry"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("expected %v, got %v", wantNames, names)
	}

	files := callTestCallerFiles(result.Callers)
	wantFiles := []string{"cmd/app/main.go", "internal/auth/auth.go"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("expected %v, got %v", wantFiles, files)
	}
}

func TestFindCallersUsesTypeInfoForPackageQualifiedReceiverVariableCalls(t *testing.T) {
	tmp := writePackageQualifiedReceiverCallProject(t)

	result, err := FindCallers(tmp, "./internal/auth.Server.Start")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	if !reflect.DeepEqual(names, []string{"Entry"}) {
		t.Fatalf("expected auth Entry only, got %v", names)
	}

	files := callTestCallerFiles(result.Callers)
	if !reflect.DeepEqual(files, []string{"internal/auth/auth.go"}) {
		t.Fatalf("expected auth caller file only, got %v", files)
	}
}

func TestFindCallersNormalizesModulePathTargets(t *testing.T) {
	tmp := writePackageQualifiedCallProject(t)

	result, err := FindCallers(tmp, "example.com/app/internal/auth.Target")
	if err != nil {
		t.Fatal(err)
	}

	if result.Target != "./internal/auth.Target" {
		t.Fatalf("expected module path target to normalize, got %s", result.Target)
	}

	files := callTestCallerFiles(result.Callers)
	if containsString(files, "internal/billing/billing.go") {
		t.Fatalf("expected auth callers only, got %v", files)
	}
}

func TestFindCallersUsesSemanticLoaderForPackageNameSelectorCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "adapter", "target.go"), `package service

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/adapter"

func Run() {
	service.Target()
}
`)

	result, err := FindCallers(tmp, "./internal/adapter.Target")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	if !reflect.DeepEqual(names, []string{"Run"}) {
		t.Fatalf("expected semantic package selector caller, got %v", names)
	}

	files := callTestCallerFiles(result.Callers)
	if !reflect.DeepEqual(files, []string{"cmd/app/main.go"}) {
		t.Fatalf("expected caller file only, got %v", files)
	}
}

func TestFindCallersUsesSemanticLoaderForCrossPackageReceiverMethodCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Client struct{}

func (c *Client) Start() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/service"

func Run(client *service.Client) {
	client.Start()
}
`)

	result, err := FindCallers(tmp, "./internal/service.Client.Start")
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	if !reflect.DeepEqual(names, []string{"Run"}) {
		t.Fatalf("expected semantic receiver method caller, got %v", names)
	}

	files := callTestCallerFiles(result.Callers)
	if !reflect.DeepEqual(files, []string{"cmd/app/main.go"}) {
		t.Fatalf("expected caller file only, got %v", files)
	}
}

func TestFindCallsFallBackToASTWhenSemanticLoaderFails(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Target()
}

func Target() {}
`)

	oldLoader := loadSemanticCallRepository
	loadSemanticCallRepository = func(string, semantics.LoadOptions) (semantics.Repository, error) {
		return semantics.Repository{}, fmt.Errorf("loader failed")
	}
	t.Cleanup(func() {
		loadSemanticCallRepository = oldLoader
	})

	callees, err := FindCallees(tmp, "Run")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(callTestCalleeNames(callees.Callees), []string{"Target"}) {
		t.Fatalf("expected AST fallback callee, got %v", callees.Callees)
	}
	if callees.AnalysisMode != CallAnalysisModeASTFallback {
		t.Fatalf("expected AST fallback callee analysis mode, got %q", callees.AnalysisMode)
	}
	if !reflect.DeepEqual(callees.Warnings, []string{"typechecked call analysis unavailable: loader failed"}) {
		t.Fatalf("expected fallback callee warning, got %v", callees.Warnings)
	}

	callers, err := FindCallers(tmp, "Target")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(callTestCallerNames(callers.Callers), []string{"Run"}) {
		t.Fatalf("expected AST fallback caller, got %v", callers.Callers)
	}
	if callers.AnalysisMode != CallAnalysisModeASTFallback {
		t.Fatalf("expected AST fallback caller analysis mode, got %q", callers.AnalysisMode)
	}
	if !reflect.DeepEqual(callers.Warnings, []string{"typechecked call analysis unavailable: loader failed"}) {
		t.Fatalf("expected fallback caller warning, got %v", callers.Warnings)
	}
}

func TestFindCallersIgnoresTestFileCallers(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Step() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

func TestStep() {
	Step()
}
`)

	result, err := FindCallers(tmp, "Step")
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Callers) != 0 {
		t.Fatalf("expected no callers from test files, got %v", result.Callers)
	}
}

func TestFindCallersWithOptionsIncludesTestFileCallers(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Run() {
	Step()
}

func Step() {}
`)
	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

func TestStep() {
	Step()
}
`)

	result, err := FindCallersWithOptions(tmp, "Step", CallOptions{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	want := []string{"Run", "TestStep"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected %v, got %v", want, names)
	}

	files := callTestCallerFiles(result.Callers)
	wantFiles := []string{"service.go", "service_test.go"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("expected %v, got %v", wantFiles, files)
	}
}

func TestFindCallersWithOptionsIncludesExternalTestPackageCallers(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Step() {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "service", "service_test.go"), `package service_test

import "example.com/app/internal/service"

func TestStep() {
	service.Step()
}
`)

	result, err := FindCallersWithOptions(tmp, "./internal/service.Step", CallOptions{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}

	names := callTestCallerNames(result.Callers)
	want := []string{"TestStep"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected %v, got %v", want, names)
	}

	files := callTestCallerFiles(result.Callers)
	wantFiles := []string{"internal/service/service_test.go"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("expected %v, got %v", wantFiles, files)
	}
}

func TestFindCallersWithOptionsUsesTypeInfoForExternalTestReceiverCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Client struct{}

func (c *Client) Start() {}
`)
	writeFile(t, filepath.Join(tmp, "internal", "service", "service_test.go"), `package service_test

import "example.com/app/internal/service"

func TestStart() {
	client := &service.Client{}
	client.Start()
}
`)

	result, err := FindCallersWithOptions(tmp, "./internal/service.Client.Start", CallOptions{IncludeTests: true})
	if err != nil {
		t.Fatal(err)
	}

	if result.AnalysisMode != CallAnalysisModeTypechecked {
		t.Fatalf("expected typechecked analysis mode, got %q", result.AnalysisMode)
	}

	names := callTestCallerNames(result.Callers)
	want := []string{"TestStart"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("expected %v, got %v", want, names)
	}

	files := callTestCallerFiles(result.Callers)
	wantFiles := []string{"internal/service/service_test.go"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("expected %v, got %v", wantFiles, files)
	}
}

func TestFindCallersIgnoresTargetsDefinedOnlyInTestFiles(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service_test.go"), `package service

func TestOnlyTarget() {}
`)

	_, err := FindCallers(tmp, "TestOnlyTarget")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "function not found: TestOnlyTarget") {
		t.Fatalf("expected missing function error, got %v", err)
	}
}

func TestFindCallersWrapsParseErrorsWithFilePath(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "broken.go")

	writeFile(t, path, `package service

func Run(
`)

	_, err := FindCallers(tmp, "Run")
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "parse ") {
		t.Fatalf("expected parse prefix, got %v", err)
	}

	if !strings.Contains(err.Error(), path) {
		t.Fatalf("expected error to contain %s, got %v", path, err)
	}
}

func TestFindCallPathsFindsShortestPath(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {
	Slow()
	Fast()
}

func Slow() {
	Hop()
}

func Hop() {
	Target()
}

func Fast() {
	Target()
}

func Target() {}
`)

	result, err := FindCallPaths(tmp, "Entry", "Target", CallPathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 1 {
		t.Fatalf("expected 1 path, got %v", result.Paths)
	}

	got := callTestPathCallees(result.Paths[0])
	want := []string{"Fast", "Target"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindCallPathsReturnsMultiplePathsWithLimit(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

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

	result, err := FindCallPaths(tmp, "Entry", "Target", CallPathOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 2 {
		t.Fatalf("expected 2 paths, got %v", result.Paths)
	}

	got := [][]string{
		callTestPathCallees(result.Paths[0]),
		callTestPathCallees(result.Paths[1]),
	}
	want := [][]string{
		{"First", "Target"},
		{"Second", "Target"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindCallPathsHonorsMaxDepth(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {
	Middle()
}

func Middle() {
	Target()
}

func Target() {}
`)

	result, err := FindCallPaths(tmp, "Entry", "Target", CallPathOptions{MaxDepth: 1})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 0 {
		t.Fatalf("expected no paths, got %v", result.Paths)
	}
}

func TestFindCallPathsReturnsRootRelativeCallPositions(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Entry() {
	Target()
}

func Target() {}
`)

	result, err := FindCallPaths(tmp, "Entry", "Target", CallPathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 1 || len(result.Paths[0].Steps) != 1 {
		t.Fatalf("expected 1 direct path, got %v", result.Paths)
	}

	got := result.Paths[0].Steps[0].Position.File
	if got != "internal/service/service.go" {
		t.Fatalf("expected internal/service/service.go, got %s", got)
	}

	assertSourceRange(t, result.Paths[0].Steps[0].Range, "internal/service/service.go", 4, 2, 4, 8)

	if strings.Contains(got, tmp) {
		t.Fatalf("expected root-relative path, got %s", got)
	}
}

func TestFindCallPathsMatchesSelectorFunctionCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

func Target() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/service"

func Entry() {
	service.Target()
}
`)

	result, err := FindCallPaths(tmp, "Entry", "Target", CallPathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 1 {
		t.Fatalf("expected 1 path, got %v", result.Paths)
	}

	got := callTestPathCallees(result.Paths[0])
	want := []string{"Target"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindCallPathsMatchesReceiverVariableMethodCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

type Server struct{}

func Entry(server *Server) {
	server.Start()
}

func (s *Server) Start() {}
`)

	result, err := FindCallPaths(tmp, "Entry", "Server.Start", CallPathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 1 {
		t.Fatalf("expected 1 path, got %v", result.Paths)
	}

	got := callTestPathCallees(result.Paths[0])
	want := []string{"Server.Start"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFindCallPathsUsesSemanticLoaderForCrossPackageReceiverMethodCalls(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(tmp, "internal", "service", "service.go"), `package service

type Client struct{}

func (c *Client) Start() {}
`)
	writeFile(t, filepath.Join(tmp, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/service"

func Entry(client *service.Client) {
	client.Start()
}
`)

	result, err := FindCallPaths(tmp, "./cmd/app.Entry", "./internal/service.Client.Start", CallPathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 1 {
		t.Fatalf("expected 1 path, got %v", result.Paths)
	}

	got := callTestPathCallees(result.Paths[0])
	want := []string{"Client.Start"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	files := callTestPathFiles(result.Paths[0])
	wantFiles := []string{"cmd/app/main.go"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("expected %v, got %v", wantFiles, files)
	}
}

func TestFindCallPathsUsesPackageQualifiedTargets(t *testing.T) {
	tmp := writePackageQualifiedCallProject(t)

	result, err := FindCallPaths(tmp, "Run", "./internal/auth.Helper", CallPathOptions{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}

	if result.To != "./internal/auth.Helper" {
		t.Fatalf("expected normalized target, got %s", result.To)
	}

	if len(result.Paths) != 1 {
		t.Fatalf("expected 1 auth path, got %v", result.Paths)
	}

	got := callTestPathCallees(result.Paths[0])
	want := []string{"Target", "Helper"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	files := callTestPathFiles(result.Paths[0])
	wantFiles := []string{"cmd/app/main.go", "internal/auth/auth.go"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("expected %v, got %v", wantFiles, files)
	}
}

func TestFindCallPathsReturnsZeroStepPathForSameTarget(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {}
`)

	result, err := FindCallPaths(tmp, "Entry", "Entry", CallPathOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Paths) != 1 {
		t.Fatalf("expected 1 path, got %v", result.Paths)
	}

	if len(result.Paths[0].Steps) != 0 {
		t.Fatalf("expected zero steps, got %v", result.Paths[0].Steps)
	}
}

func TestFindCallPathsReturnsErrorForMissingFunction(t *testing.T) {
	tmp := t.TempDir()

	writeFile(t, filepath.Join(tmp, "service.go"), `package service

func Entry() {}
`)

	_, err := FindCallPaths(tmp, "Entry", "Missing", CallPathOptions{})
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "function not found: Missing") {
		t.Fatalf("expected missing function error, got %v", err)
	}
}

func TestNormalizeCallPathOptionsRejectsNegativeValues(t *testing.T) {
	_, err := normalizeCallPathOptions(CallPathOptions{Limit: -1})
	if err == nil {
		t.Fatal("expected limit error")
	}

	_, err = normalizeCallPathOptions(CallPathOptions{MaxDepth: -1})
	if err == nil {
		t.Fatal("expected max depth error")
	}
}

func parseCallTestCalls(t *testing.T, source string) []*ast.CallExpr {
	t.Helper()

	function := parseCallTestFunction(t, source, "Calls")

	var calls []*ast.CallExpr
	ast.Inspect(function.Decl.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok {
			calls = append(calls, call)
		}

		return true
	})

	return calls
}

func parseCallTestFunctions(t *testing.T, source string) []functionInfo {
	t.Helper()

	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, "calls.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	var functions []functionInfo
	for _, decl := range parsedFile.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		receiver := receiverTypeName(funcDecl)
		name := funcDecl.Name.Name
		functions = append(functions, functionInfo{
			Receiver: receiver,
			Name:     name,
			Target:   functionTargetName(funcDecl),
			Decl:     funcDecl,
			FileSet:  fileSet,
		})
	}

	return functions
}

func parseCallTestFunction(t *testing.T, source string, target string) functionInfo {
	t.Helper()

	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, "calls.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, decl := range parsedFile.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Name.Name == target {
			receiver := receiverTypeName(funcDecl)
			name := funcDecl.Name.Name
			return functionInfo{
				Receiver: receiver,
				Name:     name,
				Target:   functionTargetName(funcDecl),
				Decl:     funcDecl,
				FileSet:  fileSet,
			}
		}
	}

	t.Fatalf("expected function %s", target)
	return functionInfo{}
}

func writePackageQualifiedCallProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

func Entry() {
	Target()
}

func Target() {
	Helper()
}

func Helper() {}
`)
	writeFile(t, filepath.Join(root, "internal", "billing", "billing.go"), `package billing

func Entry() {
	Target()
}

func Target() {
	Helper()
}

func Helper() {}
`)
	writeFile(t, filepath.Join(root, "cmd", "app", "main.go"), `package main

import (
	authpkg "example.com/app/internal/auth"
	"example.com/app/internal/billing"
)

func Run() {
	authpkg.Target()
	billing.Target()
}
`)

	return root
}

func writePackageQualifiedReceiverCallProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n")
	writeFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

type Server struct{}

func Entry(server *Server) {
	server.Start()
}

func (s *Server) Start() {}
`)
	writeFile(t, filepath.Join(root, "internal", "billing", "billing.go"), `package billing

type Server struct{}

func Entry(server *Server) {
	server.Start()
}

func (s *Server) Start() {}
`)

	return root
}

func callTestNames(calls []*ast.CallExpr) []string {
	var names []string
	for _, call := range calls {
		name, ok := callName(call.Fun)
		if ok {
			names = append(names, name)
		}
	}

	return names
}

func callTestCalleeNames(callees []Callee) []string {
	var names []string
	for _, callee := range callees {
		names = append(names, callee.Name)
	}

	return names
}

func callTestCalleeFiles(callees []Callee) []string {
	var files []string
	for _, callee := range callees {
		files = append(files, callee.Position.File)
	}

	return files
}

func callTestCallerNames(callers []Caller) []string {
	var names []string
	for _, caller := range callers {
		names = append(names, caller.Name)
	}

	return names
}

func callTestCallerFiles(callers []Caller) []string {
	var files []string
	for _, caller := range callers {
		files = append(files, caller.Position.File)
	}

	return files
}

func callTestEntryPointLabels(entryPoints []EntryPoint) []string {
	var labels []string
	for _, entryPoint := range entryPoints {
		labels = append(labels, string(entryPoint.Kind)+":"+entryPoint.Package+":"+entryPoint.Name)
	}

	return labels
}

func callTestPathCallees(path CallPath) []string {
	var names []string
	for _, step := range path.Steps {
		names = append(names, step.Callee)
	}

	return names
}

func callTestPathFiles(path CallPath) []string {
	var files []string
	for _, step := range path.Steps {
		files = append(files, step.Position.File)
	}

	return files
}
