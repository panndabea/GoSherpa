package sherpa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/panndabea/GoSherpa/internal/semantics"
)

type TestTargetKind string

const (
	TestTargetKindSymbol  TestTargetKind = "symbol"
	TestTargetKindPackage TestTargetKind = "package"
)

type TestScope string

const (
	TestScopeRelated TestScope = "related"
	TestScopeDirect  TestScope = "direct"
	TestScopeAll     TestScope = "all"
)

type TestOptions struct {
	Scope TestScope
}

const (
	TestAnalysisModeAST            = "ast"
	TestAnalysisModeTypecheckedAST = "typechecked+ast"
)

const (
	RelatedTestReasonDirectReference = "direct-reference"
	RelatedTestReasonSamePackage     = "same-package"
	RelatedTestReasonTargetPackage   = "target-package"
	RelatedTestReasonExternalPackage = "external-package"
	RelatedTestReasonChangedSymbol   = "changed-symbol"
	RelatedTestReasonCallerPackage   = "caller-package"
)

type RelatedTest struct {
	Name            string       `json:"name"`
	Package         string       `json:"package"`
	PackageName     string       `json:"packageName"`
	Position        Position     `json:"position"`
	Range           *SourceRange `json:"range,omitempty"`
	DirectReference bool         `json:"directReference"`
	ExternalPackage bool         `json:"externalPackage"`
	Reasons         []string     `json:"reasons,omitempty"`
	Targets         []string     `json:"targets,omitempty"`
}

type TestsResult struct {
	Target       string         `json:"target"`
	Kind         TestTargetKind `json:"kind"`
	Scope        TestScope      `json:"scope,omitempty"`
	AnalysisMode string         `json:"analysisMode"`
	Warnings     []string       `json:"warnings"`
	Tests        []RelatedTest  `json:"tests"`
	Commands     []string       `json:"commands"`
	TestPlan     TestPlan       `json:"testPlan"`
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
	End  token.Pos
	Func *ast.FuncLit
}

type testReferenceKeys map[string]struct{}

type testReferenceAnalysis struct {
	Keys         testReferenceKeys
	AnalysisMode string
	Warnings     []string
}

var loadSemanticTestRepository = semantics.LoadRepository

func FindTests(root string, target string) (TestsResult, error) {
	return FindTestsWithOptions(root, target, TestOptions{Scope: TestScopeAll})
}

func FindTestsWithOptions(root string, target string, options TestOptions) (TestsResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return TestsResult{}, err
	}

	options = normalizeTestOptions(options)
	if isImpactPackageTarget(target) {
		return findPackageTestsWithOptions(rootPath, target, options)
	}

	return findSymbolTestsWithOptions(rootPath, target, options)
}

func ParseTestScope(value string) (TestScope, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(TestScopeRelated):
		return TestScopeRelated, true
	case string(TestScopeDirect):
		return TestScopeDirect, true
	case string(TestScopeAll):
		return TestScopeAll, true
	default:
		return "", false
	}
}

func normalizeTestOptions(options TestOptions) TestOptions {
	if options.Scope == "" {
		options.Scope = TestScopeRelated
		return options
	}

	if _, ok := ParseTestScope(string(options.Scope)); !ok {
		options.Scope = TestScopeRelated
	}

	return options
}

func findPackageTests(root string, target string) (TestsResult, error) {
	return findPackageTestsWithOptions(root, target, TestOptions{Scope: TestScopeAll})
}

func findPackageTestsWithOptions(root string, target string, options TestOptions) (TestsResult, error) {
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
	relatedTests, _, _ := collectRelatedTests(root, testFiles, packages, referenceTarget{})
	tests := filterTestsForScope(relatedTests, TestTargetKindPackage, options.Scope)
	plan := PlanTests(tests, TestPlanOptions{
		Target:           normalizedTarget,
		Kind:             TestTargetKindPackage,
		TargetPackages:   []string{normalizedTarget},
		FallbackPackages: []string{normalizedTarget},
	})

	return TestsResult{
		Target:       normalizedTarget,
		Kind:         TestTargetKindPackage,
		Scope:        options.Scope,
		AnalysisMode: TestAnalysisModeAST,
		Tests:        tests,
		Commands:     TestPlanCommands(plan),
		TestPlan:     plan,
	}, nil
}

func findSymbolTests(root string, target string) (TestsResult, error) {
	return findSymbolTestsWithOptions(root, target, TestOptions{Scope: TestScopeAll})
}

func findSymbolTestsWithOptions(root string, target string, options TestOptions) (TestsResult, error) {
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

	relatedTests, analysisMode, warnings := collectRelatedTests(root, testFiles, packages, normalizedTarget)
	tests := filterTestsForScope(relatedTests, TestTargetKindSymbol, options.Scope)
	tests = annotateRelatedTestTargets(tests, normalizedTarget.String())
	targetPackages := sortedMapKeys(packages)
	plan := PlanTests(tests, TestPlanOptions{
		Target:           normalizedTarget.String(),
		Kind:             TestTargetKindSymbol,
		TargetPackages:   targetPackages,
		FallbackPackages: targetPackages,
	})

	return TestsResult{
		Target:       normalizedTarget.String(),
		Kind:         TestTargetKindSymbol,
		Scope:        options.Scope,
		AnalysisMode: analysisMode,
		Warnings:     warnings,
		Tests:        tests,
		Commands:     TestPlanCommands(plan),
		TestPlan:     plan,
	}, nil
}

func annotateRelatedTestTargets(tests []RelatedTest, target string) []RelatedTest {
	target = strings.TrimSpace(target)
	if target == "" {
		return tests
	}

	result := make([]RelatedTest, 0, len(tests))
	for _, test := range tests {
		test.Targets = uniqueSorted(append(test.Targets, target))
		result = append(result, test)
	}

	return result
}

func annotateRelatedTestReason(test RelatedTest, reason string) RelatedTest {
	test.Reasons = appendRelatedTestReason(test.Reasons, reason)
	return test
}

func appendRelatedTestReason(reasons []string, reason string) []string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return reasons
	}
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}

	return append(reasons, reason)
}

func relatedTestReasons(packageMatches bool, directReference bool, externalPackage bool, target referenceTarget) []string {
	var reasons []string
	if directReference {
		reasons = appendRelatedTestReason(reasons, RelatedTestReasonDirectReference)
	}
	if packageMatches {
		if target.Name == "" {
			reasons = appendRelatedTestReason(reasons, RelatedTestReasonTargetPackage)
		} else {
			reasons = appendRelatedTestReason(reasons, RelatedTestReasonSamePackage)
		}
	}
	if externalPackage {
		reasons = appendRelatedTestReason(reasons, RelatedTestReasonExternalPackage)
	}

	return reasons
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

func collectRelatedTests(root string, testFiles []testFileInfo, packages map[string]struct{}, target referenceTarget) ([]RelatedTest, string, []string) {
	var tests []RelatedTest
	modulePath := readModulePath(root)
	typecheckedDirectReferences := analyzeTypecheckedDirectTestReferences(root, target)

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
				directReference = typecheckedDirectReferences.Keys.Contains(root, testFile.FileSet, funcDecl.Pos(), funcDecl.Name.Name) ||
					functionReferencesTarget(funcDecl, target, packageMatches, imports)
			}

			if !packageMatches && !directReference {
				continue
			}

			pos := testFile.FileSet.Position(funcDecl.Pos())
			externalPackage := isExternalTestPackage(testFile.PackageName)
			tests = append(tests, RelatedTest{
				Name:            funcDecl.Name.Name,
				Package:         testFile.Package,
				PackageName:     testFile.PackageName,
				Position:        positionRelativeToRoot(root, Position{File: pos.Filename, Line: pos.Line, Column: pos.Column}),
				Range:           sourceRangeRelativeToRoot(root, testFile.FileSet, funcDecl.Pos(), funcDecl.End()),
				DirectReference: directReference,
				ExternalPackage: externalPackage,
				Reasons:         relatedTestReasons(packageMatches, directReference, externalPackage, target),
			})

			for _, subtest := range literalSubtests(funcDecl) {
				subtestDirectReference := false
				if target.Name != "" {
					subtestName := funcDecl.Name.Name + "/" + subtest.Name
					subtestDirectReference = typecheckedDirectReferences.Keys.Contains(root, testFile.FileSet, subtest.Pos, subtestName) ||
						nodeReferencesTarget(subtest.Func.Body, target, packageMatches, imports)
				}

				if !packageMatches && !subtestDirectReference {
					continue
				}

				pos := testFile.FileSet.Position(subtest.Pos)
				externalPackage := isExternalTestPackage(testFile.PackageName)
				tests = append(tests, RelatedTest{
					Name:            funcDecl.Name.Name + "/" + subtest.Name,
					Package:         testFile.Package,
					PackageName:     testFile.PackageName,
					Position:        positionRelativeToRoot(root, Position{File: pos.Filename, Line: pos.Line, Column: pos.Column}),
					Range:           sourceRangeRelativeToRoot(root, testFile.FileSet, subtest.Pos, subtest.End),
					DirectReference: subtestDirectReference,
					ExternalPackage: externalPackage,
					Reasons:         relatedTestReasons(packageMatches, subtestDirectReference, externalPackage, target),
				})
			}
		}
	}

	sortRelatedTests(tests)

	return tests, typecheckedDirectReferences.AnalysisMode, nonNilStrings(typecheckedDirectReferences.Warnings)
}

func filterTestsForScope(tests []RelatedTest, kind TestTargetKind, scope TestScope) []RelatedTest {
	switch scope {
	case TestScopeAll:
		return tests
	case TestScopeDirect:
		if kind == TestTargetKindPackage {
			return tests
		}
		return directRelatedTests(tests)
	case TestScopeRelated:
		if kind == TestTargetKindPackage {
			return tests
		}
		direct := directRelatedTests(tests)
		if len(direct) > 0 {
			return direct
		}
		return tests
	default:
		return tests
	}
}

func directRelatedTests(tests []RelatedTest) []RelatedTest {
	var direct []RelatedTest
	for _, test := range tests {
		if test.DirectReference {
			direct = append(direct, test)
		}
	}

	return direct
}

func analyzeTypecheckedDirectTestReferences(root string, target referenceTarget) testReferenceAnalysis {
	if target.Name == "" || !referenceShouldAttemptTypechecked(root) {
		return testReferenceAnalysis{AnalysisMode: TestAnalysisModeAST}
	}

	repo, err := loadSemanticTestRepository(root, semantics.LoadOptions{IncludeTests: true})
	if err != nil {
		return testReferenceAnalysis{
			AnalysisMode: TestAnalysisModeAST,
			Warnings:     []string{fmt.Sprintf("typechecked test reference analysis unavailable: %v", err)},
		}
	}

	packages := semanticReferencePackages(repo)
	if len(packages) == 0 {
		return testReferenceAnalysis{
			AnalysisMode: TestAnalysisModeAST,
			Warnings:     append([]string{"typechecked test reference analysis unavailable: no typechecked packages loaded"}, repo.Warnings...),
		}
	}

	targetObjects := semanticReferenceTargetObjects(packages, target)
	if len(targetObjects) == 0 {
		return testReferenceAnalysis{
			AnalysisMode: TestAnalysisModeTypecheckedAST,
			Warnings:     nonNilStrings(repo.Warnings),
		}
	}

	keys := make(testReferenceKeys)
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok || !isGoTestFunction(funcDecl) {
					continue
				}

				if typecheckedNodeReferencesTarget(pkg.Info, funcDecl.Body, targetObjects) {
					keys.Add(root, pkg.FileSet, funcDecl.Pos(), funcDecl.Name.Name)
				}

				for _, subtest := range literalSubtests(funcDecl) {
					if !typecheckedNodeReferencesTarget(pkg.Info, subtest.Func.Body, targetObjects) {
						continue
					}

					keys.Add(root, pkg.FileSet, subtest.Pos, funcDecl.Name.Name+"/"+subtest.Name)
				}
			}
		}
	}

	return testReferenceAnalysis{
		Keys:         keys,
		AnalysisMode: TestAnalysisModeTypecheckedAST,
		Warnings:     nonNilStrings(repo.Warnings),
	}
}

func typecheckedNodeReferencesTarget(info types.Info, node ast.Node, targetObjects map[types.Object]struct{}) bool {
	if node == nil {
		return false
	}

	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if found {
			return false
		}

		switch node := node.(type) {
		case *ast.Ident:
			object, _ := referenceIdentObject(info, node)
			if referenceObjectMatchesTarget(object, targetObjects) {
				found = true
				return false
			}
		case *ast.SelectorExpr:
			object := info.Uses[node.Sel]
			selection := info.Selections[node]
			if referenceObjectMatchesTarget(object, targetObjects) || referenceSelectionMatchesTarget(selection, targetObjects) {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

func (keys testReferenceKeys) Add(root string, fileSet *token.FileSet, pos token.Pos, name string) {
	if keys == nil {
		return
	}

	key := testReferenceKey(root, fileSet, pos, name)
	if key == "" {
		return
	}

	keys[key] = struct{}{}
}

func (keys testReferenceKeys) Contains(root string, fileSet *token.FileSet, pos token.Pos, name string) bool {
	if keys == nil {
		return false
	}

	_, ok := keys[testReferenceKey(root, fileSet, pos, name)]
	return ok
}

func testReferenceKey(root string, fileSet *token.FileSet, pos token.Pos, name string) string {
	if fileSet == nil || !pos.IsValid() || name == "" {
		return ""
	}

	position := fileSet.Position(pos)
	relative := positionRelativeToRoot(root, Position{
		File:   position.Filename,
		Line:   position.Line,
		Column: position.Column,
	})

	return relative.File + ":" + strconv.Itoa(relative.Line) + ":" + strconv.Itoa(relative.Column) + ":" + name
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
				End:  node.End(),
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
