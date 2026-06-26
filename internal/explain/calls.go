package explain

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type callSignalTarget struct {
	Package  string
	Receiver string
	Name     string
}

type explainFunctionInfo struct {
	Package  string
	Receiver string
	Name     string
	Target   string
	Decl     *ast.FuncDecl
	FileSet  *token.FileSet
	Position sherpa.Position
	Root     string
	Imports  map[string]string
}

func callSignalsForSymbol(root string, target string, symbol sherpa.Symbol) ([]sherpa.Caller, []sherpa.Callee, []string) {
	callTarget := callSignalTargetForSymbol(root, target, symbol)
	if callTarget.Name == "" {
		return nil, nil, nil
	}

	callers, callees, err := findCallSignals(root, callTarget)
	if err != nil {
		return nil, nil, []string{"calls: " + err.Error()}
	}

	return callers, callees, nil
}

func callSignalTargetForSymbol(root string, target string, symbol sherpa.Symbol) callSignalTarget {
	switch symbol.Kind {
	case sherpa.SymbolKindFunction:
	case sherpa.SymbolKindMethod:
	default:
		return callSignalTarget{}
	}

	parsed := parseSymbolTarget(root, target)
	packagePath := parsed.Package
	if packagePath == "" {
		packagePath = symbolPackage(root, symbol)
	}

	return callSignalTarget{
		Package:  packagePath,
		Receiver: symbol.Receiver,
		Name:     symbol.Name,
	}
}

func findCallSignals(root string, target callSignalTarget) ([]sherpa.Caller, []sherpa.Callee, error) {
	functions, err := collectExplainFunctionInfos(root)
	if err != nil {
		return nil, nil, err
	}

	function, err := findExplainFunctionInfo(functions, target)
	if err != nil {
		return nil, nil, err
	}

	return collectExplainCallers(functions, target), collectExplainCallees(function), nil
}

func collectExplainFunctionInfos(root string) ([]explainFunctionInfo, error) {
	rootPath, err := explainAbsoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	files, err := sherpa.FindGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	modulePath, _ := sherpa.ModulePath(rootPath)
	sort.Strings(files)

	var functions []explainFunctionInfo
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		fileSet := token.NewFileSet()
		parsedFile, err := parser.ParseFile(fileSet, file, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}

		packagePath := explainPackagePathForFile(rootPath, file)
		imports := explainImportAliases(parsedFile, modulePath)
		for _, decl := range parsedFile.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			receiver := receiverName(funcDecl)
			name := funcDecl.Name.Name
			position := explainPosition(rootPath, fileSet, funcDecl.Pos())

			functions = append(functions, explainFunctionInfo{
				Package:  packagePath,
				Receiver: receiver,
				Name:     name,
				Target:   explainFunctionTargetName(receiver, name),
				Decl:     funcDecl,
				FileSet:  fileSet,
				Position: position,
				Root:     rootPath,
				Imports:  imports,
			})
		}
	}

	return functions, nil
}

func findExplainFunctionInfo(functions []explainFunctionInfo, target callSignalTarget) (explainFunctionInfo, error) {
	var matches []explainFunctionInfo
	for _, function := range functions {
		if !explainFunctionMatchesTarget(function, target) {
			continue
		}

		matches = append(matches, function)
	}

	if len(matches) == 0 {
		return explainFunctionInfo{}, fmt.Errorf("function not found: %s", target.Display())
	}
	if len(matches) > 1 {
		return explainFunctionInfo{}, fmt.Errorf("ambiguous function target: %s", target.Display())
	}

	return matches[0], nil
}

func explainFunctionMatchesTarget(function explainFunctionInfo, target callSignalTarget) bool {
	if target.Package != "" && function.Package != target.Package {
		return false
	}

	return function.Receiver == target.Receiver && function.Name == target.Name
}

func collectExplainCallers(functions []explainFunctionInfo, target callSignalTarget) []sherpa.Caller {
	var callers []sherpa.Caller
	for _, function := range functions {
		if function.Decl.Body == nil {
			continue
		}

		ast.Inspect(function.Decl.Body, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			if _, ok := node.(*ast.FuncLit); ok {
				return false
			}

			call, ok := node.(*ast.CallExpr)
			if !ok || !callExprReferencesTarget(function, target, call.Fun) {
				return true
			}

			callers = append(callers, sherpa.Caller{
				Name:     function.Target,
				Position: explainPosition(function.Root, function.FileSet, call.Fun.Pos()),
			})

			return true
		})
	}

	sortExplainCallers(callers)

	return callers
}

func collectExplainCallees(function explainFunctionInfo) []sherpa.Callee {
	if function.Decl.Body == nil {
		return nil
	}

	var callees []sherpa.Callee
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

		name, ok := explainCallName(call.Fun)
		if !ok {
			return true
		}

		callees = append(callees, sherpa.Callee{
			Name:     name,
			Position: explainPosition(function.Root, function.FileSet, call.Fun.Pos()),
		})

		return true
	})

	sortExplainCallees(callees)

	return callees
}

func callExprReferencesTarget(function explainFunctionInfo, target callSignalTarget, expr ast.Expr) bool {
	if target.Receiver == "" {
		if name, ok := explainIdentName(expr); ok && name == target.Name {
			return function.Package == target.Package || function.Imports["."] == target.Package
		}
	}

	parts, ok := explainSelectorPath(expr)
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

func explainCallName(expr ast.Expr) (string, bool) {
	if name, ok := explainIdentName(expr); ok {
		return name, true
	}

	parts, ok := explainSelectorPath(expr)
	if !ok || len(parts) == 0 {
		return "", false
	}

	return strings.Join(parts, "."), true
}

func explainIdentName(expr ast.Expr) (string, bool) {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name, true
	case *ast.IndexExpr:
		return explainIdentName(node.X)
	case *ast.IndexListExpr:
		return explainIdentName(node.X)
	case *ast.ParenExpr:
		return explainIdentName(node.X)
	default:
		return "", false
	}
}

func explainSelectorPath(expr ast.Expr) ([]string, bool) {
	switch node := expr.(type) {
	case *ast.Ident:
		return []string{node.Name}, true
	case *ast.SelectorExpr:
		prefix, ok := explainSelectorPath(node.X)
		if !ok {
			return nil, false
		}

		return append(prefix, node.Sel.Name), true
	case *ast.IndexExpr:
		return explainSelectorPath(node.X)
	case *ast.IndexListExpr:
		return explainSelectorPath(node.X)
	case *ast.ParenExpr:
		return explainSelectorPath(node.X)
	case *ast.StarExpr:
		return explainSelectorPath(node.X)
	default:
		return nil, false
	}
}

func explainImportAliases(file *ast.File, modulePath string) map[string]string {
	aliases := make(map[string]string)
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}

		packagePath, ok := explainLocalImportPackage(importPath, modulePath)
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

func explainLocalImportPackage(importPath string, modulePath string) (string, bool) {
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

func explainPackagePathForFile(root string, file string) string {
	relative, err := filepath.Rel(root, file)
	if err != nil {
		return ""
	}

	dir := path.Dir(filepath.ToSlash(relative))
	if dir == "." {
		return "."
	}

	return "./" + dir
}

func explainFunctionTargetName(receiver string, name string) string {
	if receiver == "" {
		return name
	}

	return receiver + "." + name
}

func explainAbsoluteRootPath(root string) (string, error) {
	value := strings.TrimSpace(root)
	if value == "" {
		return "", fmt.Errorf("repository root is empty")
	}

	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve repository root %s: %w", value, err)
	}

	return filepath.Clean(absolute), nil
}

func explainPosition(root string, fileSet *token.FileSet, pos token.Pos) sherpa.Position {
	position := fileSet.Position(pos)
	return explainPositionRelativeToRoot(root, sherpa.Position{
		File: position.Filename,
		Line: position.Line,
	})
}

func explainPositionRelativeToRoot(root string, position sherpa.Position) sherpa.Position {
	if position.File == "" {
		return position
	}

	filePath := position.File
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(root, filePath)
	}

	relative, err := filepath.Rel(root, filepath.Clean(filePath))
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		position.File = filepath.ToSlash(position.File)
		return position
	}

	position.File = filepath.ToSlash(relative)

	return position
}

func sortExplainCallers(callers []sherpa.Caller) {
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

func sortExplainCallees(callees []sherpa.Callee) {
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

func (target callSignalTarget) Display() string {
	name := explainFunctionTargetName(target.Receiver, target.Name)
	if target.Package == "" || target.Package == "." {
		return name
	}

	return target.Package + "." + name
}
