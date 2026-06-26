package impact

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/sherpa"
)

type interfaceImpactSignals struct {
	Interfaces      []string
	Implementations []string
}

type Implementer struct {
	Name     string          `json:"name"`
	Position sherpa.Position `json:"position"`
}

type ImplementersResult struct {
	Target       string        `json:"target"`
	Implementers []Implementer `json:"implementers"`
}

type SatisfiedInterface struct {
	Name     string          `json:"name"`
	Position sherpa.Position `json:"position"`
}

type InterfacesResult struct {
	Target     string               `json:"target"`
	Interfaces []SatisfiedInterface `json:"interfaces"`
}

type interfaceGraph struct {
	Interfaces             []interfaceInfo
	Types                  []typeInfo
	ImplementationsByIface map[string][]string
	InterfacesByType       map[string][]string
	EmbeddingByIface       map[string][]string
}

type interfaceInfo struct {
	Name      string
	Package   string
	Qualified string
	Position  sherpa.Position
	Methods   methodSet
	Embedded  []interfaceRef
}

type typeInfo struct {
	Name      string
	Package   string
	Qualified string
	Position  sherpa.Position
	Methods   methodSet
}

type methodSet map[string]string

type interfaceRef struct {
	Package string
	Name    string
}

type interfaceSymbolTarget struct {
	Package  string
	Receiver string
	Name     string
}

func FindImplementers(root string, target string) (ImplementersResult, error) {
	graph, err := buildInterfaceGraph(root)
	if err != nil {
		return ImplementersResult{}, err
	}

	iface, err := findInterfaceTarget(root, graph, target)
	if err != nil {
		return ImplementersResult{}, err
	}

	return ImplementersResult{
		Target:       iface.Qualified,
		Implementers: implementersForInterface(graph, iface.Qualified),
	}, nil
}

func FindInterfaces(root string, target string) (InterfacesResult, error) {
	graph, err := buildInterfaceGraph(root)
	if err != nil {
		return InterfacesResult{}, err
	}

	typ, err := findTypeTarget(root, graph, target)
	if err != nil {
		return InterfacesResult{}, err
	}

	return InterfacesResult{
		Target:     typ.Qualified,
		Interfaces: interfacesForType(graph, typ.Qualified),
	}, nil
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

		interfaces, implementations = appendInterfaceSignal(graph, interfaces, implementations, iface.Qualified)
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

			interfaces, implementations = appendInterfaceSignal(graph, interfaces, implementations, iface.Qualified)
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

		interfaces, implementations = appendInterfaceSignal(graph, interfaces, implementations, iface.Qualified)
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

	modulePath, _ := sherpa.ModulePath(root)
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
		imports := interfaceImportAliases(file, modulePath)

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
						methods, embedded := interfaceMethodSet(iface, packagePath, imports)
						if len(methods) == 0 && len(embedded) == 0 {
							continue
						}

						name := typeSpec.Name.Name
						interfaces = append(interfaces, interfaceInfo{
							Name:      name,
							Package:   packagePath,
							Qualified: qualifiedSignalName(packagePath, name),
							Position:  interfacePosition(root, fileSet, typeSpec.Pos()),
							Methods:   methods,
							Embedded:  embedded,
						})
						continue
					}

					ensureTypeInfo(typesByQualified, packagePath, typeSpec.Name.Name, interfacePosition(root, fileSet, typeSpec.Pos()))
				}
			case *ast.FuncDecl:
				if decl.Recv == nil {
					continue
				}

				receiver := interfaceReceiverName(decl)
				if receiver == "" {
					continue
				}

				typ := ensureTypeInfo(typesByQualified, packagePath, receiver, interfacePosition(root, fileSet, decl.Pos()))
				typ.Methods[decl.Name.Name] = methodSignature(decl.Type, packagePath, imports)
			}
		}
	}

	interfaces = resolveInterfaceMethodSets(interfaces)

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
		EmbeddingByIface:       interfaceEmbeddingMap(interfaces),
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

func findInterfaceTarget(root string, graph interfaceGraph, target string) (interfaceInfo, error) {
	targetParts := parseInterfaceSymbolTarget(root, target)
	if targetParts.Name == "" || targetParts.Receiver != "" {
		return interfaceInfo{}, fmt.Errorf("interface not found: %s", strings.TrimSpace(target))
	}

	var matches []interfaceInfo
	for _, iface := range graph.Interfaces {
		if !targetMatchesNamedSignal(targetParts, iface.Package, iface.Name) {
			continue
		}

		matches = append(matches, iface)
	}

	if len(matches) == 0 {
		return interfaceInfo{}, fmt.Errorf("interface not found: %s", strings.TrimSpace(target))
	}
	if len(matches) > 1 {
		return interfaceInfo{}, sherpa.NewAmbiguousTargetError("interface", target, interfaceTargetCandidates(root, matches))
	}

	return matches[0], nil
}

func findTypeTarget(root string, graph interfaceGraph, target string) (typeInfo, error) {
	targetParts := parseInterfaceSymbolTarget(root, target)
	if targetParts.Name == "" || targetParts.Receiver != "" {
		return typeInfo{}, fmt.Errorf("type not found: %s", strings.TrimSpace(target))
	}

	var matches []typeInfo
	for _, typ := range graph.Types {
		if !targetMatchesNamedSignal(targetParts, typ.Package, typ.Name) {
			continue
		}

		matches = append(matches, typ)
	}

	if len(matches) == 0 {
		return typeInfo{}, fmt.Errorf("type not found: %s", strings.TrimSpace(target))
	}
	if len(matches) > 1 {
		return typeInfo{}, sherpa.NewAmbiguousTargetError("type", target, typeTargetCandidates(root, matches))
	}

	return matches[0], nil
}

func interfaceTargetCandidates(root string, interfaces []interfaceInfo) []sherpa.TargetCandidate {
	modulePath, _ := sherpa.ModulePath(root)

	candidates := make([]sherpa.TargetCandidate, 0, len(interfaces))
	for _, iface := range interfaces {
		candidates = append(candidates, sherpa.TargetCandidate{
			Package:  iface.Package,
			Symbol:   iface.Name,
			Position: iface.Position,
			Example:  sherpa.FormatPackageQualifiedTarget(iface.Package, iface.Name, modulePath),
		})
	}

	return candidates
}

func typeTargetCandidates(root string, types []typeInfo) []sherpa.TargetCandidate {
	modulePath, _ := sherpa.ModulePath(root)

	candidates := make([]sherpa.TargetCandidate, 0, len(types))
	for _, typ := range types {
		candidates = append(candidates, sherpa.TargetCandidate{
			Package:  typ.Package,
			Symbol:   typ.Name,
			Position: typ.Position,
			Example:  sherpa.FormatPackageQualifiedTarget(typ.Package, typ.Name, modulePath),
		})
	}

	return candidates
}

func implementersForInterface(graph interfaceGraph, qualified string) []Implementer {
	typesByQualified := typesByQualifiedName(graph.Types)

	var implementers []Implementer
	for _, implementation := range graph.ImplementationsByIface[qualified] {
		typ, ok := typesByQualified[implementation]
		if !ok {
			continue
		}

		implementers = append(implementers, Implementer{
			Name:     typ.Qualified,
			Position: typ.Position,
		})
	}

	sortImplementers(implementers)

	return implementers
}

func interfacesForType(graph interfaceGraph, qualified string) []SatisfiedInterface {
	interfacesByQualified := interfacesByQualifiedName(graph.Interfaces)

	var interfaces []SatisfiedInterface
	for _, interfaceName := range graph.InterfacesByType[qualified] {
		iface, ok := interfacesByQualified[interfaceName]
		if !ok {
			continue
		}

		interfaces = append(interfaces, SatisfiedInterface{
			Name:     iface.Qualified,
			Position: iface.Position,
		})
	}

	sortSatisfiedInterfaces(interfaces)

	return interfaces
}

func typesByQualifiedName(types []typeInfo) map[string]typeInfo {
	result := make(map[string]typeInfo, len(types))
	for _, typ := range types {
		result[typ.Qualified] = typ
	}

	return result
}

func interfacesByQualifiedName(interfaces []interfaceInfo) map[string]interfaceInfo {
	result := make(map[string]interfaceInfo, len(interfaces))
	for _, iface := range interfaces {
		result[iface.Qualified] = iface
	}

	return result
}

func sortImplementers(implementers []Implementer) {
	sort.Slice(implementers, func(i int, j int) bool {
		if implementers[i].Name != implementers[j].Name {
			return implementers[i].Name < implementers[j].Name
		}
		if implementers[i].Position.File != implementers[j].Position.File {
			return implementers[i].Position.File < implementers[j].Position.File
		}

		return implementers[i].Position.Line < implementers[j].Position.Line
	})
}

func sortSatisfiedInterfaces(interfaces []SatisfiedInterface) {
	sort.Slice(interfaces, func(i int, j int) bool {
		if interfaces[i].Name != interfaces[j].Name {
			return interfaces[i].Name < interfaces[j].Name
		}
		if interfaces[i].Position.File != interfaces[j].Position.File {
			return interfaces[i].Position.File < interfaces[j].Position.File
		}

		return interfaces[i].Position.Line < interfaces[j].Position.Line
	})
}

func appendInterfaceSignal(graph interfaceGraph, interfaces []string, implementations []string, qualified string) ([]string, []string) {
	interfaces = append(interfaces, qualified)
	implementations = append(implementations, graph.ImplementationsByIface[qualified]...)

	for _, embedding := range graph.EmbeddingByIface[qualified] {
		interfaces = append(interfaces, embedding)
		implementations = append(implementations, graph.ImplementationsByIface[embedding]...)
	}

	return interfaces, implementations
}

func interfaceMethodSet(iface *ast.InterfaceType, packagePath string, imports map[string]string) (methodSet, []interfaceRef) {
	methods := make(methodSet)
	var embedded []interfaceRef

	for _, method := range iface.Methods.List {
		funcType, ok := method.Type.(*ast.FuncType)
		if ok {
			signature := methodSignature(funcType, packagePath, imports)
			for _, name := range method.Names {
				methods[name.Name] = signature
			}
			continue
		}

		ref, ok := embeddedInterfaceRef(method.Type, packagePath, imports)
		if ok {
			embedded = append(embedded, ref)
		}
	}

	return methods, embedded
}

func embeddedInterfaceRef(expr ast.Expr, packagePath string, imports map[string]string) (interfaceRef, bool) {
	switch expr := expr.(type) {
	case *ast.Ident:
		return interfaceRef{Package: packagePath, Name: expr.Name}, true
	case *ast.SelectorExpr:
		ident, ok := expr.X.(*ast.Ident)
		if !ok {
			return interfaceRef{}, false
		}

		importPackage, ok := imports[ident.Name]
		if !ok {
			return interfaceRef{}, false
		}

		return interfaceRef{Package: importPackage, Name: expr.Sel.Name}, true
	case *ast.ParenExpr:
		return embeddedInterfaceRef(expr.X, packagePath, imports)
	default:
		return interfaceRef{}, false
	}
}

func interfaceImportAliases(file *ast.File, modulePath string) map[string]string {
	imports := make(map[string]string)
	if modulePath == "" {
		return imports
	}

	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}

		alias := interfaceImportAlias(importSpec, importPath)
		if alias == "" {
			continue
		}

		importIdentity := importPath
		if packagePath, ok := localInterfaceImportPackage(importPath, modulePath); ok {
			importIdentity = packagePath
		}

		imports[alias] = importIdentity
	}

	return imports
}

func interfaceImportAlias(importSpec *ast.ImportSpec, importPath string) string {
	if importSpec.Name != nil {
		switch importSpec.Name.Name {
		case ".", "_":
			return ""
		default:
			return importSpec.Name.Name
		}
	}

	return path.Base(importPath)
}

func localInterfaceImportPackage(importPath string, modulePath string) (string, bool) {
	if importPath == modulePath {
		return ".", true
	}

	prefix := modulePath + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}

	return "./" + strings.TrimPrefix(importPath, prefix), true
}

func resolveInterfaceMethodSets(interfaces []interfaceInfo) []interfaceInfo {
	indexes := make(map[string]int, len(interfaces))
	for i, iface := range interfaces {
		indexes[iface.Qualified] = i
	}

	resolved := make(map[string]methodSet, len(interfaces))
	for i := range interfaces {
		interfaces[i].Methods = resolveInterfaceMethodSet(i, interfaces, indexes, resolved, make(map[string]struct{}))
	}

	return interfaces
}

func resolveInterfaceMethodSet(index int, interfaces []interfaceInfo, indexes map[string]int, resolved map[string]methodSet, resolving map[string]struct{}) methodSet {
	iface := interfaces[index]
	if methods, ok := resolved[iface.Qualified]; ok {
		return cloneMethodSet(methods)
	}

	if _, ok := resolving[iface.Qualified]; ok {
		return cloneMethodSet(iface.Methods)
	}
	resolving[iface.Qualified] = struct{}{}

	methods := cloneMethodSet(iface.Methods)
	for _, embedded := range iface.Embedded {
		embeddedQualified := qualifiedSignalName(embedded.Package, embedded.Name)
		embeddedIndex, ok := indexes[embeddedQualified]
		if !ok {
			continue
		}

		embeddedMethods := resolveInterfaceMethodSet(embeddedIndex, interfaces, indexes, resolved, resolving)
		mergeMethodSets(methods, embeddedMethods)
	}

	delete(resolving, iface.Qualified)
	resolved[iface.Qualified] = cloneMethodSet(methods)

	return cloneMethodSet(methods)
}

func cloneMethodSet(methods methodSet) methodSet {
	cloned := make(methodSet, len(methods))
	for name, signature := range methods {
		cloned[name] = signature
	}

	return cloned
}

func mergeMethodSets(target methodSet, source methodSet) {
	const conflictingMethodSignature = "\x00conflicting-interface-method"

	for name, signature := range source {
		existing, ok := target[name]
		if ok && existing != signature {
			target[name] = conflictingMethodSignature
			continue
		}

		target[name] = signature
	}
}

func interfaceEmbeddingMap(interfaces []interfaceInfo) map[string][]string {
	known := make(map[string]struct{}, len(interfaces))
	direct := make(map[string][]string)
	for _, iface := range interfaces {
		known[iface.Qualified] = struct{}{}
	}

	for _, iface := range interfaces {
		for _, embedded := range iface.Embedded {
			embeddedQualified := qualifiedSignalName(embedded.Package, embedded.Name)
			if _, ok := known[embeddedQualified]; !ok {
				continue
			}

			direct[embeddedQualified] = append(direct[embeddedQualified], iface.Qualified)
		}
	}

	result := make(map[string][]string)
	for qualified := range known {
		result[qualified] = uniqueSortedStrings(collectEmbeddingInterfaces(qualified, direct, make(map[string]struct{})))
	}

	return result
}

func collectEmbeddingInterfaces(qualified string, direct map[string][]string, seen map[string]struct{}) []string {
	var result []string
	for _, embedding := range direct[qualified] {
		if _, ok := seen[embedding]; ok {
			continue
		}

		seen[embedding] = struct{}{}
		result = append(result, embedding)
		result = append(result, collectEmbeddingInterfaces(embedding, direct, seen)...)
	}

	return result
}

type interfaceTypeContext struct {
	PackagePath string
	Imports     map[string]string
}

func methodSignature(funcType *ast.FuncType, packagePath string, imports map[string]string) string {
	if funcType == nil {
		return ""
	}

	context := interfaceTypeContext{
		PackagePath: packagePath,
		Imports:     imports,
	}

	return fieldListSignature(funcType.Params, context) + "->" + fieldListSignature(funcType.Results, context)
}

func fieldListSignature(fields *ast.FieldList, context interfaceTypeContext) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}

	var parts []string
	for _, field := range fields.List {
		fieldType := exprSignature(field.Type, context)
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

func exprSignature(expr ast.Expr, context interfaceTypeContext) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		if isPredeclaredTypeIdentifier(expr.Name) {
			return expr.Name
		}

		return qualifiedTypeIdentity(context.PackagePath, expr.Name)
	case *ast.SelectorExpr:
		ident, ok := expr.X.(*ast.Ident)
		if ok {
			if importPath, ok := context.Imports[ident.Name]; ok {
				return qualifiedTypeIdentity(importPath, expr.Sel.Name)
			}
		}

		return exprSignature(expr.X, context) + "." + expr.Sel.Name
	case *ast.StarExpr:
		return "*" + exprSignature(expr.X, context)
	case *ast.ArrayType:
		if expr.Len == nil {
			return "[]" + exprSignature(expr.Elt, context)
		}

		return "[" + nodeSignature(expr.Len) + "]" + exprSignature(expr.Elt, context)
	case *ast.Ellipsis:
		return "..." + exprSignature(expr.Elt, context)
	case *ast.MapType:
		return "map[" + exprSignature(expr.Key, context) + "]" + exprSignature(expr.Value, context)
	case *ast.ChanType:
		switch expr.Dir {
		case ast.RECV:
			return "<-chan " + exprSignature(expr.Value, context)
		case ast.SEND:
			return "chan<- " + exprSignature(expr.Value, context)
		default:
			return "chan " + exprSignature(expr.Value, context)
		}
	case *ast.FuncType:
		return "func(" + fieldListSignature(expr.Params, context) + ")->" + fieldListSignature(expr.Results, context)
	case *ast.IndexExpr:
		return exprSignature(expr.X, context) + "[" + exprSignature(expr.Index, context) + "]"
	case *ast.IndexListExpr:
		var indexes []string
		for _, index := range expr.Indices {
			indexes = append(indexes, exprSignature(index, context))
		}

		return exprSignature(expr.X, context) + "[" + strings.Join(indexes, ",") + "]"
	case *ast.ParenExpr:
		return exprSignature(expr.X, context)
	}

	return nodeSignature(expr)
}

func nodeSignature(node any) string {
	var buffer bytes.Buffer
	_ = printer.Fprint(&buffer, token.NewFileSet(), node)
	return buffer.String()
}

func isPredeclaredTypeIdentifier(name string) bool {
	switch name {
	case "any", "bool", "byte", "comparable", "complex64", "complex128", "error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	default:
		return false
	}
}

func qualifiedTypeIdentity(packagePath string, name string) string {
	if packagePath == "" {
		return name
	}

	return packagePath + "." + name
}

func ensureTypeInfo(types map[string]*typeInfo, packagePath string, name string, position sherpa.Position) *typeInfo {
	qualified := qualifiedSignalName(packagePath, name)
	if typ, ok := types[qualified]; ok {
		if typ.Position.File == "" && position.File != "" {
			typ.Position = position
		}

		return typ
	}

	typ := &typeInfo{
		Name:      name,
		Package:   packagePath,
		Qualified: qualified,
		Position:  position,
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

func interfacePosition(root string, fileSet *token.FileSet, pos token.Pos) sherpa.Position {
	position := fileSet.Position(pos)
	file := filepath.ToSlash(position.Filename)

	relativeFile, err := filepath.Rel(root, position.Filename)
	if err == nil && relativeFile != "." && !strings.HasPrefix(relativeFile, "..") && !filepath.IsAbs(relativeFile) {
		file = filepath.ToSlash(relativeFile)
	}

	return sherpa.Position{
		File:   file,
		Line:   position.Line,
		Column: position.Column,
	}
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
