package sherpa

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
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

type functionInfo struct {
	Target   string
	Decl     *ast.FuncDecl
	FileSet  *token.FileSet
	Position Position
	Root     string
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

type callPathSearchState struct {
	Node  string
	Nodes []string
	Steps []CallPathStep
}

func FindCallees(root string, target string) (CalleesResult, error) {
	normalizedTarget, err := normalizeCallTarget(target)
	if err != nil {
		return CalleesResult{}, err
	}

	functions, err := collectFunctionInfos(root)
	if err != nil {
		return CalleesResult{Target: normalizedTarget}, err
	}

	function, err := findFunctionInfo(functions, normalizedTarget)
	if err != nil {
		return CalleesResult{Target: normalizedTarget}, err
	}

	callees := collectCalleesFromFunction(function)

	return CalleesResult{
		Target:  normalizedTarget,
		Callees: callees,
	}, nil
}

func FindCallers(root string, target string) (CallersResult, error) {
	normalizedTarget, err := normalizeCallTarget(target)
	if err != nil {
		return CallersResult{}, err
	}

	functions, err := collectFunctionInfos(root)
	if err != nil {
		return CallersResult{Target: normalizedTarget}, err
	}

	_, err = findFunctionInfo(functions, normalizedTarget)
	if err != nil {
		return CallersResult{Target: normalizedTarget}, err
	}

	callers := collectCallersFromFunctions(functions, normalizedTarget)

	return CallersResult{
		Target:  normalizedTarget,
		Callers: callers,
	}, nil
}

func FindCallPaths(root string, from string, to string, options CallPathOptions) (CallPathsResult, error) {
	normalizedFrom, err := normalizeCallTarget(from)
	if err != nil {
		return CallPathsResult{}, err
	}

	normalizedTo, err := normalizeCallTarget(to)
	if err != nil {
		return CallPathsResult{From: normalizedFrom}, err
	}

	options, err = normalizeCallPathOptions(options)
	if err != nil {
		return CallPathsResult{From: normalizedFrom, To: normalizedTo}, err
	}

	functions, err := collectFunctionInfos(root)
	if err != nil {
		return CallPathsResult{From: normalizedFrom, To: normalizedTo}, err
	}

	fromFunction, err := findFunctionInfo(functions, normalizedFrom)
	if err != nil {
		return CallPathsResult{From: normalizedFrom, To: normalizedTo}, err
	}

	toFunction, err := findFunctionInfo(functions, normalizedTo)
	if err != nil {
		return CallPathsResult{From: normalizedFrom, To: normalizedTo}, err
	}

	graph := buildCallGraph(functions)
	paths := findCallPathsInGraph(
		graph,
		functionNode(fromFunction),
		functionNode(toFunction),
		options,
	)

	return CallPathsResult{
		From:  normalizedFrom,
		To:    normalizedTo,
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

func normalizeCallTarget(target string) (string, error) {
	value := strings.TrimSpace(target)
	if value == "" {
		return "", fmt.Errorf("target is empty")
	}

	if strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return "", fmt.Errorf("package-qualified call targets are not supported: %s", value)
	}

	segments := strings.Split(value, ".")
	if len(segments) != 1 && len(segments) != 2 {
		return "", fmt.Errorf("invalid call target: %s", value)
	}

	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(segment) {
			return "", fmt.Errorf("invalid call target: %s", value)
		}
	}

	return value, nil
}

func functionTargetName(funcDecl *ast.FuncDecl) string {
	name := funcDecl.Name.Name
	receiver := receiverTypeName(funcDecl)
	if receiver == "" {
		return name
	}

	return receiver + "." + name
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

	sort.Strings(files)

	var functions []functionInfo
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		fileSet := token.NewFileSet()
		parsedFile, err := parser.ParseFile(fileSet, file, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}

		for _, decl := range parsedFile.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			pos := fileSet.Position(funcDecl.Pos())
			position := positionRelativeToRoot(rootPath, Position{
				File: pos.Filename,
				Line: pos.Line,
			})

			functions = append(functions, functionInfo{
				Target:   functionTargetName(funcDecl),
				Decl:     funcDecl,
				FileSet:  fileSet,
				Position: position,
				Root:     rootPath,
			})
		}
	}

	return functions, nil
}

func findFunctionInfo(functions []functionInfo, target string) (functionInfo, error) {
	var matches []functionInfo
	for _, function := range functions {
		if function.Target == target {
			matches = append(matches, function)
		}
	}

	if len(matches) == 0 {
		return functionInfo{}, fmt.Errorf("function not found: %s", target)
	}

	if len(matches) > 1 {
		return functionInfo{}, fmt.Errorf("ambiguous function target: %s", target)
	}

	return matches[0], nil
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

		callees := collectCalleesFromFunction(function)
		for _, callee := range callees {
			matches := matchingFunctionInfos(functions, callee.Name)
			for _, match := range matches {
				graph[caller.Key] = append(graph[caller.Key], callGraphEdge{
					Caller:   caller,
					Callee:   functionNode(match),
					Position: callee.Position,
				})
			}
		}

		sortCallGraphEdges(graph[caller.Key])
	}

	return graph
}

func matchingFunctionInfos(functions []functionInfo, calleeName string) []functionInfo {
	var matches []functionInfo
	for _, function := range functions {
		if callMatchesTarget(calleeName, function.Target) {
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
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name, true
	case *ast.SelectorExpr:
		return selectorName(node)
	case *ast.IndexExpr:
		return callName(node.X)
	case *ast.IndexListExpr:
		return callName(node.X)
	case *ast.ParenExpr:
		return callName(node.X)
	default:
		return "", false
	}
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

func selectorName(expr ast.Expr) (string, bool) {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name, true
	case *ast.SelectorExpr:
		prefix, ok := selectorName(node.X)
		if !ok || prefix == "" {
			return node.Sel.Name, true
		}

		return prefix + "." + node.Sel.Name, true
	case *ast.IndexExpr:
		return selectorName(node.X)
	case *ast.IndexListExpr:
		return selectorName(node.X)
	case *ast.ParenExpr:
		return selectorName(node.X)
	case *ast.StarExpr:
		return selectorName(node.X)
	default:
		return "", false
	}
}

func collectCalleesFromFunction(function functionInfo) []Callee {
	if function.Decl.Body == nil {
		return nil
	}

	var callees []Callee
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

		name, ok := callName(call.Fun)
		if !ok {
			return true
		}

		pos := function.FileSet.Position(call.Fun.Pos())
		position := positionRelativeToRoot(function.Root, Position{
			File: pos.Filename,
			Line: pos.Line,
		})

		callees = append(callees, Callee{
			Name:     name,
			Position: position,
		})

		return true
	})

	sortCallees(callees)

	return callees
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

func collectCallersFromFunctions(functions []functionInfo, target string) []Caller {
	var callers []Caller

	for _, function := range functions {
		callees := collectCalleesFromFunction(function)
		for _, callee := range callees {
			if !callMatchesTarget(callee.Name, target) {
				continue
			}

			callers = append(callers, Caller{
				Name:     function.Target,
				Position: callee.Position,
			})
		}
	}

	sortCallers(callers)

	return callers
}

func collectTransitiveCallersFromFunctions(functions []functionInfo, target string) ([]Caller, error) {
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
