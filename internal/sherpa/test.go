package sherpa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
)

type TestTargetKind string

const (
	TestTargetKindSymbol  TestTargetKind = "symbol"
	TestTargetKindPackage TestTargetKind = "package"
)

type RelatedTest struct {
	Name            string   `json:"name"`
	Package         string   `json:"package"`
	PackageName     string   `json:"packageName"`
	Position        Position `json:"position"`
	DirectReference bool     `json:"directReference"`
	ExternalPackage bool     `json:"externalPackage"`
}

type TestsResult struct {
	Target   string         `json:"target"`
	Kind     TestTargetKind `json:"kind"`
	Tests    []RelatedTest  `json:"tests"`
	Commands []string       `json:"commands"`
	TestPlan TestPlan       `json:"testPlan"`
}

type testFileInfo struct {
	Package     string
	PackageName string
	FileSet     *token.FileSet
	File        *ast.File
}

type literalSubtest struct {
	Name string
	Pos  token.Pos
	Func *ast.FuncLit
}

func FindTests(root string, target string) (TestsResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return TestsResult{}, err
	}

	if isImpactPackageTarget(target) {
		return findPackageTests(rootPath, target)
	}

	return findSymbolTests(rootPath, target)
}

func findPackageTests(root string, target string) (TestsResult, error) {
	modPath, err := modulePath(root)
	if err != nil {
		return TestsResult{}, err
	}

	normalizedTarget, err := normalizeTargetPackage(target, modPath)
	if err != nil {
		return TestsResult{}, err
	}

	importsByPackage, err := collectPackageImports(root)
	if err != nil {
		return TestsResult{}, err
	}

	if _, ok := importsByPackage[normalizedTarget]; !ok {
		return TestsResult{}, fmt.Errorf("package not found: %s", normalizedTarget)
	}

	testFiles, err := collectTestFiles(root)
	if err != nil {
		return TestsResult{}, err
	}

	packages := map[string]struct{}{
		normalizedTarget: {},
	}
	tests := collectRelatedTests(root, testFiles, packages, referenceTarget{})
	plan := PlanTests(tests, TestPlanOptions{
		Target:           normalizedTarget,
		Kind:             TestTargetKindPackage,
		TargetPackages:   []string{normalizedTarget},
		FallbackPackages: []string{normalizedTarget},
	})

	return TestsResult{
		Target:   normalizedTarget,
		Kind:     TestTargetKindPackage,
		Tests:    tests,
		Commands: TestPlanCommands(plan),
		TestPlan: plan,
	}, nil
}

func findSymbolTests(root string, target string) (TestsResult, error) {
	normalizedTarget, err := normalizeReferenceTarget(root, target)
	if err != nil {
		return TestsResult{}, err
	}

	testFiles, err := collectTestFiles(root)
	if err != nil {
		return TestsResult{}, err
	}

	packages, err := referenceTargetPackages(root, normalizedTarget)
	if err != nil {
		return TestsResult{}, err
	}

	tests := collectRelatedTests(root, testFiles, packages, normalizedTarget)
	targetPackages := sortedMapKeys(packages)
	plan := PlanTests(tests, TestPlanOptions{
		Target:           normalizedTarget.String(),
		Kind:             TestTargetKindSymbol,
		TargetPackages:   targetPackages,
		FallbackPackages: targetPackages,
	})

	return TestsResult{
		Target:   normalizedTarget.String(),
		Kind:     TestTargetKindSymbol,
		Tests:    tests,
		Commands: TestPlanCommands(plan),
		TestPlan: plan,
	}, nil
}

func collectTestFiles(root string) ([]testFileInfo, error) {
	files, err := FindGoFiles(root)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	var testFiles []testFileInfo
	for _, path := range files {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		packagePath, err := packagePathForFile(root, path)
		if err != nil {
			return nil, err
		}

		testFiles = append(testFiles, testFileInfo{
			Package:     packagePath,
			PackageName: file.Name.Name,
			FileSet:     fileSet,
			File:        file,
		})
	}

	return testFiles, nil
}

func referenceTargetPackages(root string, target referenceTarget) (map[string]struct{}, error) {
	files, err := FindGoFiles(root)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	packages := make(map[string]struct{})
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		if !fileDefinesReferenceTarget(file, target) {
			continue
		}

		packagePath, err := packagePathForFile(root, path)
		if err != nil {
			return nil, err
		}
		if target.Package != "" && packagePath != target.Package {
			continue
		}

		packages[packagePath] = struct{}{}
	}

	return packages, nil
}

func fileDefinesReferenceTarget(file *ast.File, target referenceTarget) bool {
	for _, decl := range file.Decls {
		if declDefinesReferenceTarget(decl, target) {
			return true
		}
	}

	return false
}

func collectRelatedTests(root string, testFiles []testFileInfo, packages map[string]struct{}, target referenceTarget) []RelatedTest {
	var tests []RelatedTest
	modulePath := readModulePath(root)

	for _, testFile := range testFiles {
		_, packageMatches := packages[testFile.Package]
		imports := testFileLocalImports(testFile.File, modulePath)

		for _, decl := range testFile.File.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || !isGoTestFunction(funcDecl) {
				continue
			}

			directReference := false
			if target.Name != "" {
				directReference = functionReferencesTarget(funcDecl, target, packageMatches, imports)
			}

			if !packageMatches && !directReference {
				continue
			}

			pos := testFile.FileSet.Position(funcDecl.Pos())
			tests = append(tests, RelatedTest{
				Name:            funcDecl.Name.Name,
				Package:         testFile.Package,
				PackageName:     testFile.PackageName,
				Position:        positionRelativeToRoot(root, Position{File: pos.Filename, Line: pos.Line}),
				DirectReference: directReference,
				ExternalPackage: isExternalTestPackage(testFile.PackageName),
			})

			for _, subtest := range literalSubtests(funcDecl) {
				subtestDirectReference := false
				if target.Name != "" {
					subtestDirectReference = nodeReferencesTarget(subtest.Func.Body, target, packageMatches, imports)
				}

				if !packageMatches && !subtestDirectReference {
					continue
				}

				pos := testFile.FileSet.Position(subtest.Pos)
				tests = append(tests, RelatedTest{
					Name:            funcDecl.Name.Name + "/" + subtest.Name,
					Package:         testFile.Package,
					PackageName:     testFile.PackageName,
					Position:        positionRelativeToRoot(root, Position{File: pos.Filename, Line: pos.Line}),
					DirectReference: subtestDirectReference,
					ExternalPackage: isExternalTestPackage(testFile.PackageName),
				})
			}
		}
	}

	sortRelatedTests(tests)

	return tests
}

func testFileLocalImports(file *ast.File, modulePath string) map[string]string {
	imports := make(map[string]string)
	if modulePath == "" {
		return imports
	}

	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}

		localPath, ok := localPackagePath(importPath, modulePath)
		if !ok {
			continue
		}

		name := path.Base(importPath)
		if importSpec.Name != nil {
			name = importSpec.Name.Name
		}
		if name == "" || name == "." || name == "_" {
			continue
		}

		imports[name] = localPath
	}

	return imports
}

func isGoTestFunction(funcDecl *ast.FuncDecl) bool {
	return funcDecl.Recv == nil && strings.HasPrefix(funcDecl.Name.Name, "Test")
}

func functionReferencesTarget(funcDecl *ast.FuncDecl, target referenceTarget, samePackage bool, imports map[string]string) bool {
	if funcDecl.Body == nil {
		return false
	}

	return nodeReferencesTarget(funcDecl.Body, target, samePackage, imports)
}

func nodeReferencesTarget(node ast.Node, target referenceTarget, samePackage bool, imports map[string]string) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if found {
			return false
		}

		switch node := node.(type) {
		case *ast.Ident:
			if target.Receiver == "" && node.Name == target.Name && (target.Package == "" || samePackage) {
				found = true
				return false
			}
		case *ast.SelectorExpr:
			if selectorReferencesTarget(node, target, samePackage, imports) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

func literalSubtests(funcDecl *ast.FuncDecl) []literalSubtest {
	if funcDecl.Body == nil {
		return nil
	}

	testParamNames := testingTParamNames(funcDecl.Type)
	if len(testParamNames) == 0 {
		return nil
	}

	return collectLiteralSubtests(funcDecl.Body, testParamNames)
}

func collectLiteralSubtests(body *ast.BlockStmt, testParamNames map[string]struct{}) []literalSubtest {
	if body == nil {
		return nil
	}

	var subtests []literalSubtest
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case nil:
			return true
		case *ast.FuncLit:
			return false
		case *ast.CallExpr:
			name, funcLit, ok := literalSubtestCall(node, testParamNames)
			if !ok {
				return true
			}

			subtests = append(subtests, literalSubtest{
				Name: name,
				Pos:  node.Pos(),
				Func: funcLit,
			})

			nestedParamNames := mergeStringSets(testParamNames, testingTParamNames(funcLit.Type))
			nestedSubtests := collectLiteralSubtests(funcLit.Body, nestedParamNames)
			for _, nested := range nestedSubtests {
				nested.Name = name + "/" + nested.Name
				subtests = append(subtests, nested)
			}

			return false
		default:
			return true
		}
	})

	return subtests
}

func literalSubtestCall(call *ast.CallExpr, testParamNames map[string]struct{}) (string, *ast.FuncLit, bool) {
	if len(call.Args) < 2 {
		return "", nil, false
	}

	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Run" {
		return "", nil, false
	}

	receiver, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", nil, false
	}
	if _, ok := testParamNames[receiver.Name]; !ok {
		return "", nil, false
	}

	name, ok := stringLiteralValue(call.Args[0])
	if !ok {
		return "", nil, false
	}

	funcLit, ok := call.Args[1].(*ast.FuncLit)
	if !ok {
		return "", nil, false
	}

	return name, funcLit, true
}

func stringLiteralValue(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}

	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}

	return value, true
}

func testingTParamNames(funcType *ast.FuncType) map[string]struct{} {
	names := make(map[string]struct{})
	if funcType == nil || funcType.Params == nil {
		return names
	}

	for _, field := range funcType.Params.List {
		if !isTestingTType(field.Type) {
			continue
		}
		for _, name := range field.Names {
			if name.Name != "" {
				names[name.Name] = struct{}{}
			}
		}
	}

	return names
}

func isTestingTType(expr ast.Expr) bool {
	starExpr, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}

	selector, ok := starExpr.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "T" {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == "testing"
}

func mergeStringSets(first map[string]struct{}, second map[string]struct{}) map[string]struct{} {
	merged := make(map[string]struct{}, len(first)+len(second))
	for value := range first {
		merged[value] = struct{}{}
	}
	for value := range second {
		merged[value] = struct{}{}
	}

	return merged
}

func selectorReferencesTarget(selector *ast.SelectorExpr, target referenceTarget, samePackage bool, imports map[string]string) bool {
	if target.Package != "" {
		if samePackage {
			name, ok := selectorName(selector)
			if ok && name == target.Symbol() {
				return true
			}
		}

		packageName, ok := selector.X.(*ast.Ident)
		if !ok || target.Receiver != "" || selector.Sel.Name != target.Name {
			return false
		}

		return imports[packageName.Name] == target.Package
	}

	name, ok := selectorName(selector)
	if ok && name == target.Symbol() {
		return true
	}

	return selector.Sel.Name == target.Name
}

func isExternalTestPackage(packageName string) bool {
	return strings.HasSuffix(packageName, "_test")
}

func testCommands(tests []RelatedTest) []string {
	var commands []string
	for _, test := range tests {
		commands = append(commands, "go test "+test.Package)
	}

	return uniqueSorted(commands)
}

func sortRelatedTests(tests []RelatedTest) {
	sort.Slice(tests, func(i int, j int) bool {
		if tests[i].Package != tests[j].Package {
			return tests[i].Package < tests[j].Package
		}

		if tests[i].Position.File != tests[j].Position.File {
			return tests[i].Position.File < tests[j].Position.File
		}

		if tests[i].Position.Line != tests[j].Position.Line {
			return tests[i].Position.Line < tests[j].Position.Line
		}

		return tests[i].Name < tests[j].Name
	})
}
