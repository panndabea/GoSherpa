package sherpa

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/panndabea/GoSherpa/internal/semantics"
)

type Callee struct {
	Name     string       `json:"name"`
	Position Position     `json:"position"`
	Range    *SourceRange `json:"range,omitempty"`
}

const (
	CallAnalysisModeTypechecked = "typechecked"
	CallAnalysisModeASTFallback = "ast-fallback"
)

type CalleesResult struct {
	Target       string   `json:"target"`
	AnalysisMode string   `json:"analysisMode"`
	Warnings     []string `json:"warnings"`
	Limitations  []string `json:"limitations"`
	Callees      []Callee `json:"callees"`
}

type Caller struct {
	Name     string       `json:"name"`
	Position Position     `json:"position"`
	Range    *SourceRange `json:"range,omitempty"`
}

type CallersResult struct {
	Target       string   `json:"target"`
	AnalysisMode string   `json:"analysisMode"`
	Warnings     []string `json:"warnings"`
	Limitations  []string `json:"limitations"`
	Callers      []Caller `json:"callers"`
}

type CallOptions struct {
	IncludeTests bool `json:"includeTests"`
	BuildTags    []string
}

type CallPathOptions struct {
	Limit    int `json:"limit"`
	MaxDepth int `json:"maxDepth"`
}

type CallPathStep struct {
	Caller   string       `json:"caller"`
	Callee   string       `json:"callee"`
	Position Position     `json:"position"`
	Range    *SourceRange `json:"range,omitempty"`
}

type CallPath struct {
	Steps []CallPathStep `json:"steps"`
}

type CallPathsResult struct {
	From         string     `json:"from"`
	To           string     `json:"to"`
	AnalysisMode string     `json:"analysisMode"`
	Warnings     []string   `json:"warnings"`
	Limitations  []string   `json:"limitations"`
	Paths        []CallPath `json:"paths"`
}

type callTarget struct {
	Package  string
	Receiver string
	Name     string
}

type functionInfo struct {
	Package     string
	PackageName string
	ImportPath  string
	ModulePath  string
	Receiver    string
	Name        string
	Target      string
	Decl        *ast.FuncDecl
	FileSet     *token.FileSet
	TypeInfo    *types.Info
	Position    Position
	Root        string
	Imports     map[string]string
	ImportPaths map[string]string
}

type callReference struct {
	Name     string
	Expr     ast.Expr
	Position Position
	Range    *SourceRange
}

type staticCallValueAssignments struct {
	values map[types.Object]ast.Expr
	counts map[types.Object]int
}

type callGraphNode struct {
	Key      string
	Target   string
	Position Position
}

type callGraphEdge struct {
	Caller   callGraphNode
	Callee   callGraphNode
	Position Position
	Range    *SourceRange
}

type testFunctionGroupKey struct {
	Dir     string
	Package string
}

type callPathSearchState struct {
	Node  string
	Nodes []string
	Steps []CallPathStep
}

type callUncertaintyKind string

const (
	callUncertaintyInterfaceDispatch callUncertaintyKind = "interface-dispatch"
	callUncertaintyFunctionValue     callUncertaintyKind = "function-value"
	callUncertaintyReflection        callUncertaintyKind = "reflection"
	callUncertaintyGoroutine         callUncertaintyKind = "goroutine"
	callUncertaintyFunctionLiteral   callUncertaintyKind = "function-literal"
)

type callUncertaintySignal struct {
	Kind     callUncertaintyKind
	Position Position
}

var loadSemanticCallRepository = semantics.LoadRepository

func FindCallees(root string, target string) (CalleesResult, error) {
	return FindCalleesWithOptions(root, target, CallOptions{})
}

func FindCalleesWithOptions(root string, target string, options CallOptions) (CalleesResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return CalleesResult{}, err
	}

	normalizedTarget, err := normalizeCallTarget(rootPath, target)
	if err != nil {
		return CalleesResult{}, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfos(rootPath, options)
	if err != nil {
		return CalleesResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	return findCalleesInFunctions(functions, normalizedTarget, analysisMode, warnings)
}

func FindCalleesWithContext(context *SemanticContext, target string, options CallOptions) (CalleesResult, error) {
	if context == nil {
		return CalleesResult{}, fmt.Errorf("semantic context is nil")
	}
	if !context.supportsBuildTags(options.BuildTags) {
		return CalleesResult{}, fmt.Errorf("semantic context build tags do not match call options")
	}

	normalizedTarget, err := normalizeCallTarget(context.root, target)
	if err != nil {
		return CalleesResult{}, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfosWithContext(context, options)
	if err != nil {
		return CalleesResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	return findCalleesInFunctions(functions, normalizedTarget, analysisMode, warnings)
}

func findCalleesInFunctions(functions []functionInfo, target callTarget, analysisMode string, warnings []string) (CalleesResult, error) {
	function, err := findFunctionInfo(functions, target)
	if err != nil {
		return CalleesResult{
			Target:       target.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	callees := collectCalleesFromFunction(function)
	limitations := collectDynamicCallLimitations([]functionInfo{function})

	return CalleesResult{
		Target:       target.String(),
		AnalysisMode: analysisMode,
		Warnings:     nonNilStrings(warnings),
		Limitations:  nonNilStrings(limitations),
		Callees:      callees,
	}, nil
}

func FindCallers(root string, target string) (CallersResult, error) {
	return FindCallersWithOptions(root, target, CallOptions{})
}

func FindCallersWithOptions(root string, target string, options CallOptions) (CallersResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return CallersResult{}, err
	}

	normalizedTarget, err := normalizeCallTarget(rootPath, target)
	if err != nil {
		return CallersResult{}, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfos(rootPath, options)
	if err != nil {
		return CallersResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	return findCallersInFunctionsWithOptions(rootPath, functions, normalizedTarget, options, analysisMode, warnings)
}

func FindCallersWithContext(context *SemanticContext, target string, options CallOptions) (CallersResult, error) {
	if context == nil {
		return CallersResult{}, fmt.Errorf("semantic context is nil")
	}
	if !context.supportsBuildTags(options.BuildTags) {
		return CallersResult{}, fmt.Errorf("semantic context build tags do not match call options")
	}

	normalizedTarget, err := normalizeCallTarget(context.root, target)
	if err != nil {
		return CallersResult{}, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfosWithContext(context, options)
	if err != nil {
		return CallersResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	return findCallersInFunctionsWithContext(context, context.root, functions, normalizedTarget, options, analysisMode, warnings)
}

func findCallersInFunctions(functions []functionInfo, target callTarget) (CallersResult, error) {
	return findCallersInFunctionsWithOptions("", functions, target, CallOptions{}, "", nil)
}

func findCallersInFunctionsWithOptions(root string, functions []functionInfo, target callTarget, options CallOptions, analysisMode string, warnings []string) (CallersResult, error) {
	return findCallersInFunctionsWithContext(nil, root, functions, target, options, analysisMode, warnings)
}

func findCallersInFunctionsWithContext(context *SemanticContext, root string, functions []functionInfo, target callTarget, options CallOptions, analysisMode string, warnings []string) (CallersResult, error) {
	function, err := findFunctionInfo(functions, target)
	if err != nil {
		return CallersResult{
			Target:       target.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	callerFunctions := functions
	if options.IncludeTests {
		testFunctions, testWarnings, err := collectTestCallerFunctionInfosWithContext(context, root, options)
		warnings = uniqueSorted(append(warnings, testWarnings...))
		if err != nil {
			return CallersResult{
				Target:       target.String(),
				AnalysisMode: analysisMode,
				Warnings:     nonNilStrings(warnings),
			}, err
		}

		callerFunctions = append(callerFunctions, testFunctions...)
	}

	callers := collectCallersFromFunctions(callerFunctions, functionCallTarget(function))
	limitations := collectDynamicCallLimitations(callerFunctions)

	return CallersResult{
		Target:       target.String(),
		AnalysisMode: analysisMode,
		Warnings:     nonNilStrings(warnings),
		Limitations:  nonNilStrings(limitations),
		Callers:      callers,
	}, nil
}

func FindCallPaths(root string, from string, to string, options CallPathOptions) (CallPathsResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return CallPathsResult{}, err
	}

	normalizedFrom, err := normalizeCallTarget(rootPath, from)
	if err != nil {
		return CallPathsResult{}, err
	}

	normalizedTo, err := normalizeCallTarget(rootPath, to)
	if err != nil {
		return CallPathsResult{From: normalizedFrom.String()}, err
	}

	options, err = normalizeCallPathOptions(options)
	if err != nil {
		return CallPathsResult{From: normalizedFrom.String(), To: normalizedTo.String()}, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfos(rootPath, CallOptions{})
	if err != nil {
		return CallPathsResult{
			From:         normalizedFrom.String(),
			To:           normalizedTo.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	fromFunction, err := findFunctionInfo(functions, normalizedFrom)
	if err != nil {
		return CallPathsResult{
			From:         normalizedFrom.String(),
			To:           normalizedTo.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	toFunction, err := findFunctionInfo(functions, normalizedTo)
	if err != nil {
		return CallPathsResult{
			From:         normalizedFrom.String(),
			To:           normalizedTo.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	graph := buildCallGraph(functions)
	paths := findCallPathsInGraph(
		graph,
		functionNode(fromFunction),
		functionNode(toFunction),
		options,
	)

	return CallPathsResult{
		From:         normalizedFrom.String(),
		To:           normalizedTo.String(),
		AnalysisMode: analysisMode,
		Warnings:     nonNilStrings(warnings),
		Limitations:  collectDynamicCallLimitations(functions),
		Paths:        paths,
	}, nil
}

func normalizeCallPathOptions(options CallPathOptions) (CallPathOptions, error) {
	if options.Limit < 0 {
		return CallPathOptions{}, fmt.Errorf("limit must be greater than zero")
	}

	if options.Limit == 0 {
		options.Limit = 1
	}

	if options.MaxDepth < 0 {
		return CallPathOptions{}, fmt.Errorf("max depth must be zero or greater")
	}

	return options, nil
}

func normalizeCallTarget(root string, target string) (callTarget, error) {
	value := strings.TrimSpace(target)
	if value == "" {
		return callTarget{}, fmt.Errorf("target is empty")
	}

	packagePath, symbol, hasPackage, err := splitPackageQualifiedTarget(root, value)
	if err != nil {
		return callTarget{}, err
	}
	if !hasPackage && (strings.Contains(value, "/") || strings.Contains(value, "\\")) {
		return callTarget{}, fmt.Errorf("invalid call target: %s", value)
	}

	segments := strings.Split(symbol, ".")
	if len(segments) != 1 && len(segments) != 2 {
		return callTarget{}, fmt.Errorf("invalid call target: %s", value)
	}

	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(segment) {
			return callTarget{}, fmt.Errorf("invalid call target: %s", value)
		}
	}

	targetInfo := callTarget{
		Package: packagePath,
		Name:    segments[len(segments)-1],
	}
	if len(segments) == 2 {
		targetInfo.Receiver = segments[0]
	}

	return targetInfo, nil
}

func (target callTarget) String() string {
	symbol := target.Symbol()
	if target.Package == "." {
		return symbol
	}
	if target.Package != "" {
		return target.Package + "." + symbol
	}

	return symbol
}

func (target callTarget) Symbol() string {
	if target.Receiver == "" {
		return target.Name
	}

	return target.Receiver + "." + target.Name
}

func functionTargetName(funcDecl *ast.FuncDecl) string {
	name := funcDecl.Name.Name
	receiver := receiverTypeName(funcDecl)
	if receiver == "" {
		return name
	}

	return receiver + "." + name
}

func functionCallTarget(function functionInfo) callTarget {
	return callTarget{
		Package:  function.Package,
		Receiver: function.Receiver,
		Name:     function.Name,
	}
}

func receiverTypeName(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return ""
	}

	return receiverBaseName(funcDecl.Recv.List[0].Type)
}

func receiverBaseName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.StarExpr:
		return receiverBaseName(node.X)
	case *ast.IndexExpr:
		return receiverBaseName(node.X)
	case *ast.IndexListExpr:
		return receiverBaseName(node.X)
	case *ast.ParenExpr:
		return receiverBaseName(node.X)
	default:
		return ""
	}
}

func collectFunctionInfos(root string) ([]functionInfo, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	modulePath := readModulePath(rootPath)
	sort.Strings(files)

	groups := groupReferenceFiles(files)
	dirs := sortedReferencePackageDirs(groups)

	var functions []functionInfo
	for _, dir := range dirs {
		fileSet := token.NewFileSet()
		packagePath, err := referencePackagePathForDir(rootPath, dir)
		if err != nil {
			return nil, err
		}
		importPath := referenceImportPath(rootPath, modulePath, dir)
		fileImports := make(map[*ast.File]map[string]string)

		var parsedFiles []*ast.File
		for _, file := range groups[dir] {
			parsedFile, err := parser.ParseFile(fileSet, file, nil, 0)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", file, err)
			}

			parsedFiles = append(parsedFiles, parsedFile)
			fileImports[parsedFile] = callImportAliases(parsedFile, modulePath)
		}

		info := &types.Info{
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		}
		config := types.Config{
			Importer: importer.Default(),
			Error:    func(error) {},
		}
		_, _ = config.Check(importPath, fileSet, parsedFiles, info)

		for _, parsedFile := range parsedFiles {
			imports := fileImports[parsedFile]
			for _, decl := range parsedFile.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}

				receiver := receiverTypeName(funcDecl)
				name := funcDecl.Name.Name
				pos := fileSet.Position(funcDecl.Pos())
				position := positionRelativeToRoot(rootPath, Position{
					File:   pos.Filename,
					Line:   pos.Line,
					Column: pos.Column,
				})

				functions = append(functions, functionInfo{
					Package:     packagePath,
					PackageName: parsedFile.Name.Name,
					ImportPath:  importPath,
					ModulePath:  modulePath,
					Receiver:    receiver,
					Name:        name,
					Target:      functionTargetName(funcDecl),
					Decl:        funcDecl,
					FileSet:     fileSet,
					TypeInfo:    info,
					Position:    position,
					Root:        rootPath,
					Imports:     imports,
				})
			}
		}
	}

	return functions, nil
}

func collectCallFunctionInfos(root string, options CallOptions) ([]functionInfo, string, []string, error) {
	functions, warnings, ok := collectTypecheckedCallFunctionInfos(root, options)
	if ok {
		return functions, CallAnalysisModeTypechecked, warnings, nil
	}

	functions, err := collectFunctionInfos(root)
	if err != nil {
		return nil, CallAnalysisModeASTFallback, warnings, err
	}

	return functions, CallAnalysisModeASTFallback, warnings, nil
}

func collectCallFunctionInfosWithContext(context *SemanticContext, options CallOptions) ([]functionInfo, string, []string, error) {
	if context == nil {
		return nil, "", nil, fmt.Errorf("semantic context is nil")
	}
	if !context.supportsBuildTags(options.BuildTags) {
		return collectCallFunctionInfos(context.root, options)
	}

	functions, warnings, ok := context.typecheckedCallFunctionInfos(options)
	if ok {
		return functions, CallAnalysisModeTypechecked, warnings, nil
	}

	functions, err := collectFunctionInfos(context.root)
	if err != nil {
		return nil, CallAnalysisModeASTFallback, warnings, err
	}

	return functions, CallAnalysisModeASTFallback, warnings, nil
}

func collectTypecheckedCallFunctionInfos(root string, options CallOptions) ([]functionInfo, []string, bool) {
	if !referenceShouldAttemptTypechecked(root) {
		return nil, nil, false
	}

	repo, err := loadSemanticCallRepository(root, semantics.LoadOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return nil, []string{fmt.Sprintf("typechecked call analysis unavailable: %v", err)}, false
	}

	functions := semanticCallFunctionInfos(repo)
	if len(functions) == 0 {
		return nil, append([]string{"typechecked call analysis unavailable: no typechecked packages loaded"}, repo.Warnings...), false
	}

	return functions, nonNilStrings(repo.Warnings), true
}

func semanticCallFunctionInfos(repo semantics.Repository) []functionInfo {
	modulePath := readModulePath(repo.Root)
	importPaths := semanticCallImportPaths(repo)

	var functions []functionInfo
	for _, pkg := range repo.Packages {
		if !semanticCallPackageUsable(pkg) {
			continue
		}

		for _, parsedFile := range pkg.Files {
			imports := callImportAliases(parsedFile, modulePath)
			for _, decl := range parsedFile.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}

				receiver := receiverTypeName(funcDecl)
				name := funcDecl.Name.Name
				pos := pkg.FileSet.Position(funcDecl.Pos())
				position := positionRelativeToRoot(repo.Root, Position{
					File:   pos.Filename,
					Line:   pos.Line,
					Column: pos.Column,
				})

				functions = append(functions, functionInfo{
					Package:     pkg.PackagePath,
					PackageName: pkg.Name,
					ImportPath:  pkg.ImportPath,
					ModulePath:  modulePath,
					Receiver:    receiver,
					Name:        name,
					Target:      functionTargetName(funcDecl),
					Decl:        funcDecl,
					FileSet:     pkg.FileSet,
					TypeInfo:    pkg.TypesInfo,
					Position:    position,
					Root:        repo.Root,
					Imports:     imports,
					ImportPaths: importPaths,
				})
			}
		}
	}

	sortFunctionInfos(functions)

	return functions
}

func semanticCallPackageUsable(pkg semantics.Package) bool {
	return pkg.FileSet != nil && pkg.TypesInfo != nil && len(pkg.Files) > 0
}

func collectTestCallerFunctionInfos(root string, options CallOptions) ([]functionInfo, []string, error) {
	functions, warnings, ok := collectTypecheckedTestCallerFunctionInfos(root, options)
	if ok {
		return functions, warnings, nil
	}

	functions, err := collectASTTestCallerFunctionInfos(root)
	return functions, warnings, err
}

func collectTestCallerFunctionInfosWithContext(context *SemanticContext, root string, options CallOptions) ([]functionInfo, []string, error) {
	if context == nil {
		return collectTestCallerFunctionInfos(root, options)
	}
	if !context.supportsBuildTags(options.BuildTags) {
		return collectTestCallerFunctionInfos(context.root, options)
	}

	functions, warnings, ok := context.typecheckedTestCallFunctionInfos(options)
	if ok {
		return functions, warnings, nil
	}

	functions, err := collectASTTestCallerFunctionInfos(context.root)
	return functions, warnings, err
}

func collectTypecheckedTestCallerFunctionInfos(root string, options CallOptions) ([]functionInfo, []string, bool) {
	if !referenceShouldAttemptTypechecked(root) {
		return nil, nil, false
	}

	repo, err := loadSemanticCallRepository(root, semantics.LoadOptions{
		IncludeTests: true,
		BuildTags:    options.BuildTags,
	})
	if err != nil {
		return nil, []string{fmt.Sprintf("typechecked test caller analysis unavailable: %v", err)}, false
	}

	return semanticTestCallFunctionInfos(repo), nonNilStrings(repo.Warnings), true
}

func semanticTestCallFunctionInfos(repo semantics.Repository) []functionInfo {
	modulePath := readModulePath(repo.Root)
	importPaths := semanticCallImportPaths(repo)
	seen := make(map[string]struct{})

	var functions []functionInfo
	for _, pkg := range repo.Packages {
		if !semanticCallPackageUsable(pkg) {
			continue
		}

		functionPackage := testFunctionPackagePath(pkg.PackagePath, pkg.Name)
		for _, parsedFile := range pkg.Files {
			imports := callImportAliases(parsedFile, modulePath)
			for _, decl := range parsedFile.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}

				pos := pkg.FileSet.Position(funcDecl.Pos())
				if !strings.HasSuffix(pos.Filename, "_test.go") {
					continue
				}

				receiver := receiverTypeName(funcDecl)
				name := funcDecl.Name.Name
				position := positionRelativeToRoot(repo.Root, Position{
					File:   pos.Filename,
					Line:   pos.Line,
					Column: pos.Column,
				})
				key := testFunctionInfoKey(position, functionTargetName(funcDecl))
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}

				functions = append(functions, functionInfo{
					Package:     functionPackage,
					PackageName: pkg.Name,
					ImportPath:  pkg.ImportPath,
					ModulePath:  modulePath,
					Receiver:    receiver,
					Name:        name,
					Target:      functionTargetName(funcDecl),
					Decl:        funcDecl,
					FileSet:     pkg.FileSet,
					TypeInfo:    pkg.TypesInfo,
					Position:    position,
					Root:        repo.Root,
					Imports:     imports,
					ImportPaths: importPaths,
				})
			}
		}
	}

	sortFunctionInfos(functions)

	return functions
}

func semanticCallImportPaths(repo semantics.Repository) map[string]string {
	paths := make(map[string]string)
	for _, pkg := range repo.Packages {
		if strings.TrimSpace(pkg.ImportPath) == "" || strings.TrimSpace(pkg.PackagePath) == "" {
			continue
		}

		paths[pkg.ImportPath] = pkg.PackagePath
	}

	return paths
}

func testFunctionPackagePath(packagePath string, packageName string) string {
	if isExternalTestPackage(packageName) {
		return packagePath + "_test"
	}

	return packagePath
}

func testFunctionInfoKey(position Position, target string) string {
	return position.File + ":" + strconv.Itoa(position.Line) + ":" + strconv.Itoa(position.Column) + ":" + target
}

func collectASTTestCallerFunctionInfos(root string) ([]functionInfo, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	modulePath := readModulePath(rootPath)
	sort.Strings(files)

	regularFilesByDir := make(map[string][]string)
	regularPackagesByDir := make(map[string]string)
	testFilesByGroup := make(map[testFunctionGroupKey][]string)

	for _, file := range files {
		dir := filepath.Dir(file)
		packageName, err := parseCallFilePackageName(file)
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(file, "_test.go") {
			key := testFunctionGroupKey{
				Dir:     dir,
				Package: packageName,
			}
			testFilesByGroup[key] = append(testFilesByGroup[key], file)
			continue
		}

		regularFilesByDir[dir] = append(regularFilesByDir[dir], file)
		if regularPackagesByDir[dir] == "" {
			regularPackagesByDir[dir] = packageName
		}
	}

	for _, files := range regularFilesByDir {
		sort.Strings(files)
	}
	for _, files := range testFilesByGroup {
		sort.Strings(files)
	}

	keys := sortedTestFunctionGroupKeys(testFilesByGroup)
	var functions []functionInfo
	for _, key := range keys {
		packagePath, err := referencePackagePathForDir(rootPath, key.Dir)
		if err != nil {
			return nil, err
		}

		regularPackage := regularPackagesByDir[key.Dir]
		isExternalTestPackage := regularPackage != "" && key.Package != regularPackage
		functionPackage := packagePath
		importPath := referenceImportPath(rootPath, modulePath, key.Dir)
		filesToParse := append([]string{}, testFilesByGroup[key]...)
		if isExternalTestPackage {
			functionPackage = packagePath + "_test"
			importPath += "_test"
		} else {
			filesToParse = append(append([]string{}, regularFilesByDir[key.Dir]...), filesToParse...)
		}

		fileSet := token.NewFileSet()
		fileImports := make(map[*ast.File]map[string]string)
		testFiles := make(map[*ast.File]struct{})

		var parsedFiles []*ast.File
		for _, file := range filesToParse {
			parsedFile, err := parser.ParseFile(fileSet, file, nil, 0)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", file, err)
			}

			parsedFiles = append(parsedFiles, parsedFile)
			fileImports[parsedFile] = callImportAliases(parsedFile, modulePath)
			if strings.HasSuffix(file, "_test.go") {
				testFiles[parsedFile] = struct{}{}
			}
		}

		info := &types.Info{
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		}
		config := types.Config{
			Importer: importer.Default(),
			Error:    func(error) {},
		}
		_, _ = config.Check(importPath, fileSet, parsedFiles, info)

		for _, parsedFile := range parsedFiles {
			if _, ok := testFiles[parsedFile]; !ok {
				continue
			}

			imports := fileImports[parsedFile]
			for _, decl := range parsedFile.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}

				receiver := receiverTypeName(funcDecl)
				name := funcDecl.Name.Name
				pos := fileSet.Position(funcDecl.Pos())
				position := positionRelativeToRoot(rootPath, Position{
					File:   pos.Filename,
					Line:   pos.Line,
					Column: pos.Column,
				})

				functions = append(functions, functionInfo{
					Package:     functionPackage,
					PackageName: key.Package,
					ImportPath:  importPath,
					ModulePath:  modulePath,
					Receiver:    receiver,
					Name:        name,
					Target:      functionTargetName(funcDecl),
					Decl:        funcDecl,
					FileSet:     fileSet,
					TypeInfo:    info,
					Position:    position,
					Root:        rootPath,
					Imports:     imports,
				})
			}
		}
	}

	return functions, nil
}

func parseCallFilePackageName(file string) (string, error) {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, file, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", file, err)
	}

	return parsedFile.Name.Name, nil
}

func sortedTestFunctionGroupKeys(groups map[testFunctionGroupKey][]string) []testFunctionGroupKey {
	keys := make([]testFunctionGroupKey, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i int, j int) bool {
		if keys[i].Dir != keys[j].Dir {
			return keys[i].Dir < keys[j].Dir
		}

		return keys[i].Package < keys[j].Package
	})

	return keys
}

func findFunctionInfo(functions []functionInfo, target callTarget) (functionInfo, error) {
	var matches []functionInfo
	for _, function := range functions {
		if functionMatchesCallTarget(function, target) {
			matches = append(matches, function)
		}
	}

	if len(matches) == 0 {
		return functionInfo{}, fmt.Errorf("function not found: %s", target.String())
	}

	if len(matches) > 1 {
		return functionInfo{}, NewAmbiguousTargetError("function", target.String(), functionTargetCandidates(matches))
	}

	return matches[0], nil
}

func functionTargetCandidates(functions []functionInfo) []TargetCandidate {
	candidates := make([]TargetCandidate, 0, len(functions))
	for _, function := range functions {
		candidates = append(candidates, TargetCandidate{
			Package:  function.Package,
			Symbol:   function.Target,
			Position: function.Position,
			Example:  FormatPackageQualifiedTarget(function.Package, function.Target, function.ModulePath),
		})
	}

	return candidates
}

func functionMatchesCallTarget(function functionInfo, target callTarget) bool {
	if target.Package != "" && function.Package != target.Package {
		return false
	}

	return function.Receiver == target.Receiver && function.Name == target.Name
}

func functionNode(function functionInfo) callGraphNode {
	key := fmt.Sprintf(
		"%s\x00%s:%d",
		function.Target,
		function.Position.File,
		function.Position.Line,
	)

	return callGraphNode{
		Key:      key,
		Target:   function.Target,
		Position: function.Position,
	}
}

func buildCallGraph(functions []functionInfo) map[string][]callGraphEdge {
	graph := make(map[string][]callGraphEdge)

	for _, function := range functions {
		caller := functionNode(function)
		graph[caller.Key] = nil

		references := collectCallReferencesFromFunction(function)
		for _, reference := range references {
			matches := matchingFunctionInfosForCall(functions, function, reference.Expr)
			for _, match := range matches {
				graph[caller.Key] = append(graph[caller.Key], callGraphEdge{
					Caller:   caller,
					Callee:   functionNode(match),
					Position: reference.Position,
					Range:    reference.Range,
				})
			}
		}

		sortCallGraphEdges(graph[caller.Key])
	}

	return graph
}

func matchingFunctionInfosForCall(functions []functionInfo, caller functionInfo, expr ast.Expr) []functionInfo {
	var matches []functionInfo
	for _, function := range functions {
		if callExprReferencesTarget(caller, functionCallTarget(function), expr) {
			matches = append(matches, function)
		}
	}

	sortFunctionInfos(matches)

	return matches
}

func sortFunctionInfos(functions []functionInfo) {
	sort.Slice(functions, func(i int, j int) bool {
		if functions[i].Position.File != functions[j].Position.File {
			return functions[i].Position.File < functions[j].Position.File
		}

		if functions[i].Position.Line != functions[j].Position.Line {
			return functions[i].Position.Line < functions[j].Position.Line
		}

		return functions[i].Target < functions[j].Target
	})
}

func sortCallGraphEdges(edges []callGraphEdge) {
	sort.Slice(edges, func(i int, j int) bool {
		if edges[i].Position.File != edges[j].Position.File {
			return edges[i].Position.File < edges[j].Position.File
		}

		if edges[i].Position.Line != edges[j].Position.Line {
			return edges[i].Position.Line < edges[j].Position.Line
		}

		if edges[i].Callee.Target != edges[j].Callee.Target {
			return edges[i].Callee.Target < edges[j].Callee.Target
		}

		return edges[i].Callee.Key < edges[j].Callee.Key
	})
}

func findCallPathsInGraph(graph map[string][]callGraphEdge, from callGraphNode, to callGraphNode, options CallPathOptions) []CallPath {
	if from.Key == to.Key {
		return []CallPath{{}}
	}

	maxDepth := options.MaxDepth
	if maxDepth == 0 {
		maxDepth = len(graph)
	}

	queue := []callPathSearchState{
		{
			Node:  from.Key,
			Nodes: []string{from.Key},
		},
	}

	var paths []CallPath
	for len(queue) > 0 && len(paths) < options.Limit {
		state := queue[0]
		queue = queue[1:]

		if len(state.Steps) >= maxDepth {
			continue
		}

		for _, edge := range graph[state.Node] {
			if containsCallPathNode(state.Nodes, edge.Callee.Key) {
				continue
			}

			steps := append([]CallPathStep{}, state.Steps...)
			steps = append(steps, CallPathStep{
				Caller:   edge.Caller.Target,
				Callee:   edge.Callee.Target,
				Position: edge.Position,
				Range:    edge.Range,
			})

			if edge.Callee.Key == to.Key {
				paths = append(paths, CallPath{Steps: steps})
				if len(paths) >= options.Limit {
					break
				}

				continue
			}

			nodes := append([]string{}, state.Nodes...)
			nodes = append(nodes, edge.Callee.Key)

			queue = append(queue, callPathSearchState{
				Node:  edge.Callee.Key,
				Nodes: nodes,
				Steps: steps,
			})
		}
	}

	return paths
}

func containsCallPathNode(nodes []string, target string) bool {
	for _, node := range nodes {
		if node == target {
			return true
		}
	}

	return false
}

func callName(expr ast.Expr) (string, bool) {
	if name, ok := callIdentName(expr); ok {
		return name, true
	}

	parts, ok := selectorPath(expr)
	if !ok || len(parts) == 0 {
		return "", false
	}

	return strings.Join(parts, "."), true
}

func callMatchesTarget(calleeName string, target string) bool {
	if calleeName == target {
		return true
	}

	if strings.Contains(target, ".") {
		return false
	}

	return strings.HasSuffix(calleeName, "."+target)
}

func callIdentName(expr ast.Expr) (string, bool) {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name, true
	case *ast.IndexExpr:
		return callIdentName(node.X)
	case *ast.IndexListExpr:
		return callIdentName(node.X)
	case *ast.ParenExpr:
		return callIdentName(node.X)
	default:
		return "", false
	}
}

func selectorPath(expr ast.Expr) ([]string, bool) {
	switch node := expr.(type) {
	case *ast.Ident:
		return []string{node.Name}, true
	case *ast.SelectorExpr:
		prefix, ok := selectorPath(node.X)
		if !ok {
			return nil, false
		}

		return append(prefix, node.Sel.Name), true
	case *ast.IndexExpr:
		return selectorPath(node.X)
	case *ast.IndexListExpr:
		return selectorPath(node.X)
	case *ast.ParenExpr:
		return selectorPath(node.X)
	case *ast.StarExpr:
		return selectorPath(node.X)
	default:
		return nil, false
	}
}

func selectorName(expr ast.Expr) (string, bool) {
	parts, ok := selectorPath(expr)
	if !ok || len(parts) == 0 {
		return "", false
	}

	return strings.Join(parts, "."), true
}

func callExprReferencesTarget(function functionInfo, target callTarget, expr ast.Expr) bool {
	if callObjectReferencesTarget(function, target, expr) {
		return true
	}

	if callSelectionReferencesTarget(function, target, expr) {
		return true
	}

	if target.Package == "" {
		name, ok := callName(expr)
		return ok && callMatchesTarget(name, target.Symbol())
	}

	if target.Receiver == "" {
		if name, ok := callIdentName(expr); ok && name == target.Name {
			return function.Package == target.Package || function.Imports["."] == target.Package
		}
	}

	parts, ok := selectorPath(expr)
	if !ok {
		return false
	}

	if target.Receiver == "" && len(parts) == 2 {
		return function.Imports[parts[0]] == target.Package && parts[1] == target.Name
	}

	if target.Receiver != "" && len(parts) == 2 {
		if parts[0] != target.Receiver || parts[1] != target.Name {
			return false
		}

		return function.Package == target.Package || function.Imports["."] == target.Package
	}

	if target.Receiver != "" && len(parts) == 3 {
		return function.Imports[parts[0]] == target.Package &&
			parts[1] == target.Receiver &&
			parts[2] == target.Name
	}

	return false
}

func callObjectReferencesTarget(function functionInfo, target callTarget, expr ast.Expr) bool {
	object := callObject(function, expr)
	return callFunctionObjectReferencesTarget(function, object, target)
}

func callObject(function functionInfo, expr ast.Expr) types.Object {
	if function.TypeInfo == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.Ident:
		return function.TypeInfo.Uses[node]
	case *ast.SelectorExpr:
		return function.TypeInfo.Uses[node.Sel]
	case *ast.IndexExpr:
		return callObject(function, node.X)
	case *ast.IndexListExpr:
		return callObject(function, node.X)
	case *ast.ParenExpr:
		return callObject(function, node.X)
	default:
		return nil
	}
}

func callFunctionObjectReferencesTarget(function functionInfo, object types.Object, target callTarget) bool {
	called, ok := object.(*types.Func)
	if !ok {
		return false
	}

	if called.Name() != target.Name {
		return false
	}

	if callFuncReceiverName(called) != target.Receiver {
		return false
	}

	return callObjectPackageMatchesTarget(function, called, target)
}

func callSelectionReferencesTarget(function functionInfo, target callTarget, expr ast.Expr) bool {
	if target.Receiver == "" || function.TypeInfo == nil {
		return false
	}

	selection := callSelection(function, expr)
	if selection == nil {
		return false
	}

	object := selection.Obj()
	if object == nil || object.Name() != target.Name {
		return false
	}

	if callSelectionReceiverName(selection) != target.Receiver {
		return false
	}

	return callObjectPackageMatchesTarget(function, object, target)
}

func callSelection(function functionInfo, expr ast.Expr) *types.Selection {
	if function.TypeInfo == nil {
		return nil
	}

	selector, ok := callSelectorExpr(expr)
	if !ok {
		return nil
	}

	return function.TypeInfo.Selections[selector]
}

func callSelectorExpr(expr ast.Expr) (*ast.SelectorExpr, bool) {
	switch node := expr.(type) {
	case *ast.SelectorExpr:
		return node, true
	case *ast.IndexExpr:
		return callSelectorExpr(node.X)
	case *ast.IndexListExpr:
		return callSelectorExpr(node.X)
	case *ast.ParenExpr:
		return callSelectorExpr(node.X)
	default:
		return nil, false
	}
}

func callSelectionReceiverName(selection *types.Selection) string {
	function, ok := selection.Obj().(*types.Func)
	if !ok {
		return ""
	}

	return callFuncReceiverName(function)
}

func callFuncReceiverName(function *types.Func) string {
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return ""
	}

	return callTypeReceiverName(signature.Recv().Type())
}

func callTypeReceiverName(typ types.Type) string {
	switch typ := typ.(type) {
	case *types.Named:
		return typ.Obj().Name()
	case *types.Pointer:
		return callTypeReceiverName(typ.Elem())
	default:
		return ""
	}
}

func callObjectPackageMatchesTarget(function functionInfo, object types.Object, target callTarget) bool {
	if target.Package == "" {
		return true
	}

	objectPackage := object.Pkg()
	if objectPackage == nil {
		return function.Package == target.Package
	}

	if packagePath, ok := function.ImportPaths[objectPackage.Path()]; ok {
		return packagePath == target.Package
	}

	if objectPackage.Path() == function.ImportPath {
		return function.Package == target.Package
	}

	localPackage, ok := callLocalImportPackage(objectPackage.Path(), function.ModulePath)
	if !ok {
		return false
	}

	return localPackage == target.Package
}

func callImportAliases(file *ast.File, modulePath string) map[string]string {
	aliases := make(map[string]string)
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}

		packagePath, ok := callLocalImportPackage(importPath, modulePath)
		if !ok {
			continue
		}

		alias := path.Base(importPath)
		if importSpec.Name != nil {
			alias = importSpec.Name.Name
		}
		if alias == "_" {
			continue
		}

		aliases[alias] = packagePath
	}

	return aliases
}

func callLocalImportPackage(importPath string, modulePath string) (string, bool) {
	if modulePath == "" {
		return "", false
	}
	if importPath == modulePath {
		return ".", true
	}
	if !strings.HasPrefix(importPath, modulePath+"/") {
		return "", false
	}

	localPath := strings.TrimPrefix(importPath, modulePath+"/")
	cleaned := path.Clean(localPath)
	if cleaned == "." {
		return ".", true
	}

	return "./" + cleaned, true
}

func collectCalleesFromFunction(function functionInfo) []Callee {
	references := collectCallReferencesFromFunction(function)
	callees := make([]Callee, 0, len(references))
	for _, reference := range references {
		callees = append(callees, Callee{
			Name:     reference.Name,
			Position: reference.Position,
			Range:    reference.Range,
		})
	}

	sortCallees(callees)

	return callees
}

func collectCallReferencesFromFunction(function functionInfo) []callReference {
	if function.Decl.Body == nil {
		return nil
	}

	staticValues := collectStaticCallValueAssignments(function)
	var references []callReference
	ast.Inspect(function.Decl.Body, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}

		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		expr := resolveStaticCallValue(function, staticValues, call.Fun)
		name, ok := callReferenceName(function, expr)
		if !ok {
			return true
		}

		pos := function.FileSet.Position(call.Fun.Pos())
		position := positionRelativeToRoot(function.Root, Position{
			File:   pos.Filename,
			Line:   pos.Line,
			Column: pos.Column,
		})

		references = append(references, callReference{
			Name:     name,
			Expr:     expr,
			Position: position,
			Range:    sourceRangeRelativeToRoot(function.Root, function.FileSet, call.Fun.Pos(), call.Fun.End()),
		})

		return true
	})

	sortCallReferences(references)

	return references
}

func collectStaticCallValueAssignments(function functionInfo) staticCallValueAssignments {
	assignments := staticCallValueAssignments{
		values: make(map[types.Object]ast.Expr),
		counts: make(map[types.Object]int),
	}

	if function.TypeInfo == nil || function.Decl.Body == nil {
		return assignments
	}

	ast.Inspect(function.Decl.Body, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}

		switch typed := node.(type) {
		case *ast.AssignStmt:
			collectAssignStmtStaticCallValues(function, assignments, typed)
		case *ast.ValueSpec:
			collectValueSpecStaticCallValues(function, assignments, typed)
		}

		return true
	})

	return assignments
}

func collectAssignStmtStaticCallValues(function functionInfo, assignments staticCallValueAssignments, stmt *ast.AssignStmt) {
	if stmt.Tok != token.DEFINE {
		invalidateAssignedCallValues(function, assignments, stmt.Lhs)
		return
	}

	if len(stmt.Lhs) != len(stmt.Rhs) {
		invalidateAssignedCallValues(function, assignments, stmt.Lhs)
		return
	}

	for i, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		if object := function.TypeInfo.Uses[ident]; object != nil {
			invalidateStaticCallValue(assignments, object)
			continue
		}

		object := function.TypeInfo.Defs[ident]
		if object == nil {
			continue
		}

		recordStaticCallValue(function, assignments, object, stmt.Rhs[i])
	}
}

func collectValueSpecStaticCallValues(function functionInfo, assignments staticCallValueAssignments, spec *ast.ValueSpec) {
	if len(spec.Names) != len(spec.Values) {
		for _, name := range spec.Names {
			if name == nil || name.Name == "_" {
				continue
			}
			invalidateStaticCallValue(assignments, function.TypeInfo.Defs[name])
		}
		return
	}

	for i, name := range spec.Names {
		if name == nil || name.Name == "_" {
			continue
		}

		recordStaticCallValue(function, assignments, function.TypeInfo.Defs[name], spec.Values[i])
	}
}

func invalidateAssignedCallValues(function functionInfo, assignments staticCallValueAssignments, lhs []ast.Expr) {
	for _, expr := range lhs {
		ident, ok := expr.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		invalidateStaticCallValue(assignments, function.TypeInfo.Uses[ident])
	}
}

func recordStaticCallValue(function functionInfo, assignments staticCallValueAssignments, object types.Object, expr ast.Expr) {
	if object == nil {
		return
	}

	if !callStaticFunctionValueExpr(function, expr) {
		invalidateStaticCallValue(assignments, object)
		return
	}

	assignments.counts[object]++
	if assignments.counts[object] != 1 {
		delete(assignments.values, object)
		return
	}

	assignments.values[object] = expr
}

func invalidateStaticCallValue(assignments staticCallValueAssignments, object types.Object) {
	if object == nil {
		return
	}

	assignments.counts[object]++
	delete(assignments.values, object)
}

func resolveStaticCallValue(function functionInfo, assignments staticCallValueAssignments, expr ast.Expr) ast.Expr {
	object := callValueObject(function, expr)
	if object == nil {
		return expr
	}

	resolved, ok := assignments.values[object]
	if !ok {
		return expr
	}

	return resolved
}

func collectDynamicCallLimitations(functions []functionInfo) []string {
	signalsByKind := make(map[callUncertaintyKind][]Position)
	for _, function := range functions {
		for _, signal := range collectCallUncertaintySignals(function) {
			signalsByKind[signal.Kind] = append(signalsByKind[signal.Kind], signal.Position)
		}
	}

	kinds := []callUncertaintyKind{
		callUncertaintyInterfaceDispatch,
		callUncertaintyFunctionValue,
		callUncertaintyReflection,
		callUncertaintyGoroutine,
		callUncertaintyFunctionLiteral,
	}

	var limitations []string
	for _, kind := range kinds {
		positions := uniqueCallLimitationPositions(signalsByKind[kind])
		if len(positions) == 0 {
			continue
		}

		limitations = append(limitations, formatCallUncertaintyLimitation(kind, positions))
	}

	return limitations
}

func collectCallUncertaintySignals(function functionInfo) []callUncertaintySignal {
	if function.Decl.Body == nil {
		return nil
	}

	staticValues := collectStaticCallValueAssignments(function)
	var signals []callUncertaintySignal
	ast.Inspect(function.Decl.Body, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		switch typed := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.GoStmt:
			if typed.Call != nil {
				signals = append(signals, callUncertaintySignal{
					Kind:     callUncertaintyGoroutine,
					Position: callExpressionPosition(function, typed.Call.Fun),
				})
			}
			return true
		case *ast.CallExpr:
			if _, ok := typed.Fun.(*ast.FuncLit); ok {
				signals = append(signals, callUncertaintySignal{
					Kind:     callUncertaintyFunctionLiteral,
					Position: callExpressionPosition(function, typed.Fun),
				})
			}
			expr := resolveStaticCallValue(function, staticValues, typed.Fun)
			if callUsesInterfaceDispatch(function, expr) {
				signals = append(signals, callUncertaintySignal{
					Kind:     callUncertaintyInterfaceDispatch,
					Position: callExpressionPosition(function, typed.Fun),
				})
			}
			if callUsesFunctionValue(function, expr) {
				signals = append(signals, callUncertaintySignal{
					Kind:     callUncertaintyFunctionValue,
					Position: callExpressionPosition(function, typed.Fun),
				})
			}
			if callUsesReflection(function, expr) {
				signals = append(signals, callUncertaintySignal{
					Kind:     callUncertaintyReflection,
					Position: callExpressionPosition(function, typed.Fun),
				})
			}
		}

		return true
	})

	return signals
}

func callUsesInterfaceDispatch(function functionInfo, expr ast.Expr) bool {
	selection := callSelection(function, expr)
	if selection == nil {
		return false
	}

	if _, ok := selection.Obj().(*types.Func); !ok {
		return false
	}

	return callTypeIsInterface(selection.Recv())
}

func callUsesFunctionValue(function functionInfo, expr ast.Expr) bool {
	if function.TypeInfo == nil {
		return false
	}
	if _, ok := expr.(*ast.FuncLit); ok {
		return false
	}

	if functionValueCallObjectIsStaticFunction(function, expr) {
		return false
	}

	typ := function.TypeInfo.TypeOf(expr)
	return callTypeIsSignature(typ)
}

func functionValueCallObjectIsStaticFunction(function functionInfo, expr ast.Expr) bool {
	if function.TypeInfo == nil {
		return false
	}

	selection := callSelection(function, expr)
	if selection != nil {
		if _, ok := selection.Obj().(*types.Func); ok {
			return true
		}
	}

	object := callObject(function, expr)
	if object == nil {
		return false
	}

	if _, ok := object.(*types.Func); ok {
		return true
	}
	if _, ok := object.(*types.Builtin); ok {
		return true
	}
	if _, ok := object.(*types.TypeName); ok {
		return true
	}

	return false
}

func callStaticFunctionValueExpr(function functionInfo, expr ast.Expr) bool {
	if function.TypeInfo == nil {
		return false
	}

	selection := callSelection(function, expr)
	if selection != nil {
		if _, ok := selection.Obj().(*types.Func); ok {
			return !callTypeIsInterface(selection.Recv())
		}
	}

	object := callObject(function, expr)
	if object == nil {
		return false
	}

	_, ok := object.(*types.Func)
	return ok
}

func callValueObject(function functionInfo, expr ast.Expr) types.Object {
	if function.TypeInfo == nil {
		return nil
	}

	switch node := expr.(type) {
	case *ast.Ident:
		return function.TypeInfo.Uses[node]
	case *ast.IndexExpr:
		return callValueObject(function, node.X)
	case *ast.IndexListExpr:
		return callValueObject(function, node.X)
	case *ast.ParenExpr:
		return callValueObject(function, node.X)
	default:
		return nil
	}
}

func callUsesReflection(function functionInfo, expr ast.Expr) bool {
	if object := callObject(function, expr); callObjectFromPackage(object, "reflect") {
		return true
	}

	selection := callSelection(function, expr)
	return selection != nil && callObjectFromPackage(selection.Obj(), "reflect")
}

func callObjectFromPackage(object types.Object, packagePath string) bool {
	if object == nil || object.Pkg() == nil {
		return false
	}

	return object.Pkg().Path() == packagePath
}

func callTypeIsSignature(typ types.Type) bool {
	if typ == nil {
		return false
	}

	_, ok := typ.Underlying().(*types.Signature)
	return ok
}

func callTypeIsInterface(typ types.Type) bool {
	if typ == nil {
		return false
	}

	switch typed := typ.(type) {
	case *types.Interface:
		return true
	case *types.Named:
		return callTypeIsInterface(typed.Underlying())
	case *types.Pointer:
		return callTypeIsInterface(typed.Elem())
	case *types.TypeParam:
		return callTypeIsInterface(typed.Constraint())
	default:
		return false
	}
}

func callExpressionPosition(function functionInfo, expr ast.Expr) Position {
	pos := function.FileSet.Position(expr.Pos())
	return positionRelativeToRoot(function.Root, Position{
		File:   pos.Filename,
		Line:   pos.Line,
		Column: pos.Column,
	})
}

func uniqueCallLimitationPositions(positions []Position) []Position {
	seen := make(map[string]struct{})
	var result []Position
	for _, position := range positions {
		key := callLimitationPositionString(position)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, position)
	}

	sort.Slice(result, func(i int, j int) bool {
		if result[i].File != result[j].File {
			return result[i].File < result[j].File
		}
		if result[i].Line != result[j].Line {
			return result[i].Line < result[j].Line
		}

		return result[i].Column < result[j].Column
	})

	return result
}

func formatCallUncertaintyLimitation(kind callUncertaintyKind, positions []Position) string {
	location := formatCallLimitationLocations(positions)
	switch kind {
	case callUncertaintyInterfaceDispatch:
		return "Interface dispatch may hide concrete call edges at " + location + "."
	case callUncertaintyFunctionValue:
		return "Function value calls may hide concrete call edges at " + location + "."
	case callUncertaintyReflection:
		return "Reflection may hide dynamic call relationships at " + location + "."
	case callUncertaintyGoroutine:
		return "Goroutine starts may make execution paths incomplete at " + location + "."
	case callUncertaintyFunctionLiteral:
		return "Function literal calls are not expanded into call graph edges at " + location + "."
	default:
		return "Dynamic call behavior may be incomplete at " + location + "."
	}
}

func formatCallLimitationLocations(positions []Position) string {
	const maxPositions = 3

	limit := len(positions)
	if limit > maxPositions {
		limit = maxPositions
	}

	locations := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		locations = append(locations, callLimitationPositionString(positions[i]))
	}

	if len(positions) > maxPositions {
		locations = append(locations, fmt.Sprintf("%d more", len(positions)-maxPositions))
	}

	return strings.Join(locations, ", ")
}

func callLimitationPositionString(position Position) string {
	if position.File == "" || position.Line <= 0 {
		return ""
	}

	return fmt.Sprintf("%s:%d", position.File, position.Line)
}

func callReferenceName(function functionInfo, expr ast.Expr) (string, bool) {
	if name, ok := callSelectionName(function, expr); ok {
		return name, true
	}

	return callName(expr)
}

func callSelectionName(function functionInfo, expr ast.Expr) (string, bool) {
	selection := callSelection(function, expr)
	if selection == nil {
		return "", false
	}

	if _, ok := selection.Obj().(*types.Func); !ok {
		return "", false
	}

	receiver := callSelectionReceiverName(selection)
	if receiver == "" {
		return "", false
	}

	return receiver + "." + selection.Obj().Name(), true
}

func sortCallees(callees []Callee) {
	sort.Slice(callees, func(i int, j int) bool {
		if callees[i].Position.File != callees[j].Position.File {
			return callees[i].Position.File < callees[j].Position.File
		}

		if callees[i].Position.Line != callees[j].Position.Line {
			return callees[i].Position.Line < callees[j].Position.Line
		}

		return callees[i].Name < callees[j].Name
	})
}

func sortCallReferences(references []callReference) {
	sort.Slice(references, func(i int, j int) bool {
		if references[i].Position.File != references[j].Position.File {
			return references[i].Position.File < references[j].Position.File
		}

		if references[i].Position.Line != references[j].Position.Line {
			return references[i].Position.Line < references[j].Position.Line
		}

		return references[i].Name < references[j].Name
	})
}

func sortCallers(callers []Caller) {
	sort.Slice(callers, func(i int, j int) bool {
		if callers[i].Position.File != callers[j].Position.File {
			return callers[i].Position.File < callers[j].Position.File
		}

		if callers[i].Position.Line != callers[j].Position.Line {
			return callers[i].Position.Line < callers[j].Position.Line
		}

		return callers[i].Name < callers[j].Name
	})
}

func collectCallersFromFunctions(functions []functionInfo, target callTarget) []Caller {
	var callers []Caller

	for _, function := range functions {
		references := collectCallReferencesFromFunction(function)
		for _, reference := range references {
			if !callExprReferencesTarget(function, target, reference.Expr) {
				continue
			}

			callers = append(callers, Caller{
				Name:     function.Target,
				Position: reference.Position,
				Range:    reference.Range,
			})
		}
	}

	sortCallers(callers)

	return callers
}

func collectTransitiveCallersFromFunctions(functions []functionInfo, target callTarget) ([]Caller, error) {
	targetFunction, err := findFunctionInfo(functions, target)
	if err != nil {
		return nil, err
	}

	graph := buildCallGraph(functions)
	reverseGraph := reverseCallGraph(graph)
	targetNode := functionNode(targetFunction)
	queue := []string{targetNode.Key}
	seenQueue := map[string]struct{}{
		targetNode.Key: {},
	}
	seenCallers := make(map[string]struct{})
	var callers []Caller

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range reverseGraph[current] {
			if _, ok := seenCallers[edge.Caller.Key]; !ok {
				seenCallers[edge.Caller.Key] = struct{}{}
				callers = append(callers, Caller{
					Name:     edge.Caller.Target,
					Position: edge.Position,
					Range:    edge.Range,
				})
			}

			if _, ok := seenQueue[edge.Caller.Key]; ok {
				continue
			}

			seenQueue[edge.Caller.Key] = struct{}{}
			queue = append(queue, edge.Caller.Key)
		}
	}

	sortCallers(callers)

	return callers, nil
}

func reverseCallGraph(graph map[string][]callGraphEdge) map[string][]callGraphEdge {
	reversed := make(map[string][]callGraphEdge)
	for _, edges := range graph {
		for _, edge := range edges {
			reversed[edge.Callee.Key] = append(reversed[edge.Callee.Key], edge)
		}
	}

	for key := range reversed {
		sortCallGraphEdges(reversed[key])
	}

	return reversed
}
