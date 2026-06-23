package sherpa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeCallTarget(test.input)
			if err != nil {
				t.Fatal(err)
			}

			if got != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}

func TestNormalizeCallTargetRejectsInvalidInput(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"Server.",
		".Start",
		"A.B.C",
		"./internal/sherpa.ParseFile",
		"github.com/example/app.ParseFile",
		`C:\repo\Run`,
		"not valid",
		"(*Server).Start",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := normalizeCallTarget(test)
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

	got := collectCallersFromFunctions([]functionInfo{function}, "Step")

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

	got := collectCallersFromFunctions([]functionInfo{function}, "ParseFile")

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

	got := collectCallersFromFunctions([]functionInfo{function}, "Step")

	if len(got) != 0 {
		t.Fatalf("expected no callers, got %v", got)
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

func TestFindCallersDoesNotMatchReceiverVariableCallsToMethodTargets(t *testing.T) {
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

	if len(result.Callers) != 0 {
		t.Fatalf("expected no callers, got %v", result.Callers)
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
			return functionInfo{
				Target:  functionTargetName(funcDecl),
				Decl:    funcDecl,
				FileSet: fileSet,
			}
		}
	}

	t.Fatalf("expected function %s", target)
	return functionInfo{}
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

func callTestCallerNames(callers []Caller) []string {
	var names []string
	for _, caller := range callers {
		names = append(names, caller.Name)
	}

	return names
}
