package impact

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type interfaceImpactSignals struct {
	Interfaces      []string
	Implementations []string
}

type interfaceGraph struct {
	Interfaces             []interfaceInfo
	Types                  []typeInfo
	ImplementationsByIface map[string][]string
	InterfacesByType       map[string][]string
}

type interfaceInfo struct {
	Name      string
	Package   string
	Qualified string
	Methods   methodSet
}

type typeInfo struct {
	Name      string
	Package   string
	Qualified string
	Methods   methodSet
}

type methodSet map[string]string

type interfaceSymbolTarget struct {
	Package  string
	Receiver string
	Name     string
}

func interfaceSignalsForPackages(root string, packages []string) (interfaceImpactSignals, error) {
	packages = uniqueSortedStrings(packages)
	if len(packages) == 0 {
		return interfaceImpactSignals{}, nil
	}

	graph, err := buildInterfaceGraph(root)
	if err != nil {
		return interfaceImpactSignals{}, err
	}

	packageSet := make(map[string]struct{})
	for _, pkg := range packages {
		packageSet[pkg] = struct{}{}
	}

	var interfaces []string
	var implementations []string

	for _, iface := range graph.Interfaces {
		if _, ok := packageSet[iface.Package]; !ok {
			continue
		}

		interfaces = append(interfaces, iface.Qualified)
		implementations = append(implementations, graph.ImplementationsByIface[iface.Qualified]...)
	}

	for _, typ := range graph.Types {
		if _, ok := packageSet[typ.Package]; !ok {
			continue
		}

		implementedInterfaces := graph.InterfacesByType[typ.Qualified]
		if len(implementedInterfaces) == 0 {
			continue
		}

		interfaces = append(interfaces, implementedInterfaces...)
		implementations = append(implementations, typ.Qualified)
	}

	return interfaceImpactSignals{
		Interfaces:      uniqueSortedStrings(interfaces),
		Implementations: uniqueSortedStrings(implementations),
	}, nil
}

func interfaceSignalsForSymbol(root string, target string) (interfaceImpactSignals, error) {
	targetParts := parseInterfaceSymbolTarget(root, target)
	if targetParts.Name == "" {
		return interfaceImpactSignals{}, nil
	}

	graph, err := buildInterfaceGraph(root)
	if err != nil {
		return interfaceImpactSignals{}, err
	}

	var interfaces []string
	var implementations []string

	if targetParts.Receiver != "" {
		for _, iface := range graph.Interfaces {
			if !targetMatchesNamedSignal(targetParts, iface.Package, iface.Name) {
				continue
			}
			if _, ok := iface.Methods[targetParts.Name]; !ok {
				continue
			}

			interfaces = append(interfaces, iface.Qualified)
			implementations = append(implementations, graph.ImplementationsByIface[iface.Qualified]...)
		}

		for _, typ := range graph.Types {
			if !targetMatchesNamedSignal(targetParts, typ.Package, typ.Name) {
				continue
			}
			if _, ok := typ.Methods[targetParts.Name]; !ok {
				continue
			}

			interfaces = append(interfaces, graph.InterfacesByType[typ.Qualified]...)
			implementations = append(implementations, typ.Qualified)
		}

		return interfaceImpactSignals{
			Interfaces:      uniqueSortedStrings(interfaces),
			Implementations: uniqueSortedStrings(implementations),
		}, nil
	}

	for _, iface := range graph.Interfaces {
		if !targetMatchesNamedSignal(targetParts, iface.Package, iface.Name) {
			continue
		}

		interfaces = append(interfaces, iface.Qualified)
		implementations = append(implementations, graph.ImplementationsByIface[iface.Qualified]...)
	}

	for _, typ := range graph.Types {
		if !targetMatchesNamedSignal(targetParts, typ.Package, typ.Name) {
			continue
		}

		interfaces = append(interfaces, graph.InterfacesByType[typ.Qualified]...)
		implementations = append(implementations, typ.Qualified)
	}

	return interfaceImpactSignals{
		Interfaces:      uniqueSortedStrings(interfaces),
		Implementations: uniqueSortedStrings(implementations),
	}, nil
}

func buildInterfaceGraph(root string) (interfaceGraph, error) {
	files, err := sherpa.FindGoFiles(root)
	if err != nil {
		return interfaceGraph{}, err
	}

	sort.Strings(files)

	var interfaces []interfaceInfo
	typesByQualified := make(map[string]*typeInfo)

	for _, filePath := range files {
		if strings.HasSuffix(filePath, "_test.go") {
			continue
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, filePath, nil, 0)
		if err != nil {
			return interfaceGraph{}, err
		}

		packagePath, err := interfacePackagePathForFile(root, filePath)
		if err != nil {
			return interfaceGraph{}, err
		}

		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				if decl.Tok != token.TYPE {
					continue
				}

				for _, spec := range decl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					iface, ok := typeSpec.Type.(*ast.InterfaceType)
					if ok {
						methods := interfaceMethodSet(iface)
						if len(methods) == 0 {
							continue
						}

						name := typeSpec.Name.Name
						interfaces = append(interfaces, interfaceInfo{
							Name:      name,
							Package:   packagePath,
							Qualified: qualifiedSignalName(packagePath, name),
							Methods:   methods,
						})
						continue
					}

					ensureTypeInfo(typesByQualified, packagePath, typeSpec.Name.Name)
				}
			case *ast.FuncDecl:
				if decl.Recv == nil {
					continue
				}

				receiver := interfaceReceiverName(decl)
				if receiver == "" {
					continue
				}

				typ := ensureTypeInfo(typesByQualified, packagePath, receiver)
				typ.Methods[decl.Name.Name] = methodSignature(decl.Type)
			}
		}
	}

	types := make([]typeInfo, 0, len(typesByQualified))
	for _, typ := range typesByQualified {
		types = append(types, *typ)
	}
	sort.Slice(types, func(i int, j int) bool {
		return types[i].Qualified < types[j].Qualified
	})

	graph := interfaceGraph{
		Interfaces:             interfaces,
		Types:                  types,
		ImplementationsByIface: make(map[string][]string),
		InterfacesByType:       make(map[string][]string),
	}

	for _, iface := range interfaces {
		for _, typ := range types {
			if !typeImplementsInterface(typ, iface) {
				continue
			}

			graph.ImplementationsByIface[iface.Qualified] = append(graph.ImplementationsByIface[iface.Qualified], typ.Qualified)
			graph.InterfacesByType[typ.Qualified] = append(graph.InterfacesByType[typ.Qualified], iface.Qualified)
		}
	}

	for iface, implementations := range graph.ImplementationsByIface {
		graph.ImplementationsByIface[iface] = uniqueSortedStrings(implementations)
	}
	for typ, interfaces := range graph.InterfacesByType {
		graph.InterfacesByType[typ] = uniqueSortedStrings(interfaces)
	}

	return graph, nil
}

func interfaceMethodSet(iface *ast.InterfaceType) methodSet {
	methods := make(methodSet)
	for _, method := range iface.Methods.List {
		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		signature := methodSignature(funcType)
		for _, name := range method.Names {
			methods[name.Name] = signature
		}
	}

	return methods
}

func methodSignature(funcType *ast.FuncType) string {
	if funcType == nil {
		return ""
	}

	return fieldListSignature(funcType.Params) + "->" + fieldListSignature(funcType.Results)
}

func fieldListSignature(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}

	var parts []string
	for _, field := range fields.List {
		fieldType := exprSignature(field.Type)
		count := len(field.Names)
		if count == 0 {
			count = 1
		}

		for range count {
			parts = append(parts, fieldType)
		}
	}

	return strings.Join(parts, ",")
}

func exprSignature(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.FuncType:
		return "func(" + fieldListSignature(expr.Params) + ")->" + fieldListSignature(expr.Results)
	}

	var buffer bytes.Buffer
	_ = printer.Fprint(&buffer, token.NewFileSet(), expr)
	return buffer.String()
}

func ensureTypeInfo(types map[string]*typeInfo, packagePath string, name string) *typeInfo {
	qualified := qualifiedSignalName(packagePath, name)
	if typ, ok := types[qualified]; ok {
		return typ
	}

	typ := &typeInfo{
		Name:      name,
		Package:   packagePath,
		Qualified: qualified,
		Methods:   make(methodSet),
	}
	types[qualified] = typ

	return typ
}

func interfaceReceiverName(funcDecl *ast.FuncDecl) string {
	field := funcDecl.Recv.List[0]

	ident, ok := field.Type.(*ast.Ident)
	if ok {
		return ident.Name
	}

	starExpr, ok := field.Type.(*ast.StarExpr)
	if ok {
		ident, ok := starExpr.X.(*ast.Ident)
		if ok {
			return ident.Name
		}
	}

	return ""
}

func typeImplementsInterface(typ typeInfo, iface interfaceInfo) bool {
	for method, ifaceSignature := range iface.Methods {
		typeSignature, ok := typ.Methods[method]
		if !ok || typeSignature != ifaceSignature {
			return false
		}
	}

	return true
}

func interfacePackagePathForFile(root string, filePath string) (string, error) {
	relativeDir, err := filepath.Rel(root, filepath.Dir(filePath))
	if err != nil {
		return "", err
	}

	relativeDir = filepath.ToSlash(relativeDir)
	if relativeDir == "." {
		return ".", nil
	}

	return "./" + relativeDir, nil
}

func qualifiedSignalName(packagePath string, name string) string {
	if packagePath == "." {
		return name
	}

	return packagePath + "." + name
}

func parseInterfaceSymbolTarget(root string, target string) interfaceSymbolTarget {
	value := strings.TrimSpace(filepath.ToSlash(target))
	if value == "" {
		return interfaceSymbolTarget{}
	}

	packagePath, symbol := splitPackageQualifiedSignalTarget(root, value)
	parts := strings.Split(symbol, ".")
	if len(parts) >= 2 {
		return interfaceSymbolTarget{
			Package:  packagePath,
			Receiver: parts[len(parts)-2],
			Name:     parts[len(parts)-1],
		}
	}

	return interfaceSymbolTarget{
		Package: packagePath,
		Name:    symbol,
	}
}

func splitPackageQualifiedSignalTarget(root string, target string) (string, string) {
	lastSlash := strings.LastIndex(target, "/")
	if lastSlash < 0 {
		return "", target
	}

	firstDotAfterSlash := strings.Index(target[lastSlash+1:], ".")
	if firstDotAfterSlash < 0 {
		return "", target
	}

	separator := lastSlash + 1 + firstDotAfterSlash
	packagePath := normalizeSignalPackagePath(root, target[:separator])
	symbol := target[separator+1:]

	return packagePath, symbol
}

func normalizeSignalPackagePath(root string, packagePath string) string {
	value := strings.TrimSpace(filepath.ToSlash(packagePath))

	modulePath, err := sherpa.ModulePath(root)
	if err == nil && modulePath != "" {
		if value == modulePath {
			return "."
		}
		if strings.HasPrefix(value, modulePath+"/") {
			value = "./" + strings.TrimPrefix(value, modulePath+"/")
		}
	}

	value = strings.TrimPrefix(value, "./")
	cleaned := path.Clean(value)
	if cleaned == "." {
		return "."
	}

	return "./" + cleaned
}

func targetMatchesNamedSignal(target interfaceSymbolTarget, packagePath string, name string) bool {
	if target.Package != "" && target.Package != packagePath {
		return false
	}

	if target.Receiver != "" {
		return target.Receiver == name
	}

	return target.Name == name
}
