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
