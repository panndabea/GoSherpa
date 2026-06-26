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
)

type Callee struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

type CalleesResult struct {
	Target  string   `json:"target"`
	Callees []Callee `json:"callees"`
}

type Caller struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

type CallersResult struct {
	Target  string   `json:"target"`
	Callers []Caller `json:"callers"`
}

type CallOptions struct {
	IncludeTests bool `json:"includeTests"`
}

type CallPathOptions struct {
	Limit    int `json:"limit"`
	MaxDepth int `json:"maxDepth"`
}

type CallPathStep struct {
	Caller   string   `json:"caller"`
	Callee   string   `json:"callee"`
	Position Position `json:"position"`
}

type CallPath struct {
	Steps []CallPathStep `json:"steps"`
}

type CallPathsResult struct {
	From  string     `json:"from"`
	To    string     `json:"to"`
	Paths []CallPath `json:"paths"`
}

type callTarget struct {
	Package  string
	Receiver string
	Name     string
}

type functionInfo struct {
	Package    string
	ImportPath string
	ModulePath string
	Receiver   string
	Name       string
	Target     string
	Decl       *ast.FuncDecl
	FileSet    *token.FileSet
	TypeInfo   *types.Info
	Position   Position
	Root       string
	Imports    map[string]string
}

type callReference struct {
	Name     string
	Expr     ast.Expr
	Position Position
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

func FindCallees(root string, target string) (CalleesResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return CalleesResult{}, err
	}

	normalizedTarget, err := normalizeCallTarget(rootPath, target)
	if err != nil {
		return CalleesResult{}, err
	}

	functions, err := collectFunctionInfos(rootPath)
	if err != nil {
		return CalleesResult{Target: normalizedTarget.String()}, err
	}

	function, err := findFunctionInfo(functions, normalizedTarget)
	if err != nil {
		return CalleesResult{Target: normalizedTarget.String()}, err
	}

	callees := collectCalleesFromFunction(function)

	return CalleesResult{
		Target:  normalizedTarget.String(),
		Callees: callees,
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

	functions, err := collectFunctionInfos(rootPath)
	if err != nil {
		return CallersResult{Target: normalizedTarget.String()}, err
	}

	function, err := findFunctionInfo(functions, normalizedTarget)
	if err != nil {
		return CallersResult{Target: normalizedTarget.String()}, err
	}

	callerFunctions := functions
	if options.IncludeTests {
		testFunctions, err := collectTestCallerFunctionInfos(rootPath)
		if err != nil {
			return CallersResult{Target: normalizedTarget.String()}, err
		}

		callerFunctions = append(callerFunctions, testFunctions...)
	}

	callers := collectCallersFromFunctions(callerFunctions, functionCallTarget(function))

	return CallersResult{
		Target:  normalizedTarget.String(),
		Callers: callers,
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

	functions, err := collectFunctionInfos(rootPath)
	if err != nil {
		return CallPathsResult{From: normalizedFrom.String(), To: normalizedTo.String()}, err
	}

	fromFunction, err := findFunctionInfo(functions, normalizedFrom)
	if err != nil {
		return CallPathsResult{From: normalizedFrom.String(), To: normalizedTo.String()}, err
	}

	toFunction, err := findFunctionInfo(functions, normalizedTo)
	if err != nil {
		return CallPathsResult{From: normalizedFrom.String(), To: normalizedTo.String()}, err
	}

	graph := buildCallGraph(functions)
	paths := findCallPathsInGraph(
		graph,
		functionNode(fromFunction),
		functionNode(toFunction),
		options,
	)

	return CallPathsResult{
		From:  normalizedFrom.String(),
		To:    normalizedTo.String(),
		Paths: paths,
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
					File: pos.Filename,
					Line: pos.Line,
				})

				functions = append(functions, functionInfo{
					Package:    packagePath,
					ImportPath: importPath,
					ModulePath: modulePath,
					Receiver:   receiver,
					Name:       name,
					Target:     functionTargetName(funcDecl),
					Decl:       funcDecl,
					FileSet:    fileSet,
					TypeInfo:   info,
					Position:   position,
					Root:       rootPath,
					Imports:    imports,
				})
			}
		}
	}

	return functions, nil
}

func collectTestCallerFunctionInfos(root string) ([]functionInfo, error) {
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
					File: pos.Filename,
					Line: pos.Line,
				})

				functions = append(functions, functionInfo{
					Package:    functionPackage,
					ImportPath: importPath,
					ModulePath: modulePath,
					Receiver:   receiver,
					Name:       name,
					Target:     functionTargetName(funcDecl),
					Decl:       funcDecl,
					FileSet:    fileSet,
					TypeInfo:   info,
					Position:   position,
					Root:       rootPath,
					Imports:    imports,
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
		})
	}

	sortCallees(callees)

	return callees
}

func collectCallReferencesFromFunction(function functionInfo) []callReference {
	if function.Decl.Body == nil {
		return nil
	}

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

		name, ok := callReferenceName(function, call.Fun)
		if !ok {
			return true
		}

		pos := function.FileSet.Position(call.Fun.Pos())
		position := positionRelativeToRoot(function.Root, Position{
			File: pos.Filename,
			Line: pos.Line,
		})

		references = append(references, callReference{
			Name:     name,
			Expr:     call.Fun,
			Position: position,
		})

		return true
	})

	sortCallReferences(references)

	return references
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
