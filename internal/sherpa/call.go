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
	Name     string
	Position Position
}

type CalleesResult struct {
	Target  string
	Callees []Callee
}

type Caller struct {
	Name     string
	Position Position
}

type CallersResult struct {
	Target  string
	Callers []Caller
}

type functionInfo struct {
	Target   string
	Decl     *ast.FuncDecl
	FileSet  *token.FileSet
	Position Position
	Root     string
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
