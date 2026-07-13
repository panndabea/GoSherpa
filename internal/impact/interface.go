package impact

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

const (
	InterfaceAnalysisModeTypechecked = "typechecked"
	InterfaceAnalysisModeASTFallback = "ast-fallback"
)

type interfaceImpactSignals struct {
	Interfaces      []string
	Implementations []string
	AnalysisMode    string
	Warnings        []string
}

type Implementer struct {
	Name     string          `json:"name"`
	Position sherpa.Position `json:"position"`
}

type ImplementersResult struct {
	Target       string        `json:"target"`
	Implementers []Implementer `json:"implementers"`
	AnalysisMode string        `json:"-"`
	Warnings     []string      `json:"-"`
}

type SatisfiedInterface struct {
	Name     string          `json:"name"`
	Position sherpa.Position `json:"position"`
}

type InterfacesResult struct {
	Target       string               `json:"target"`
	Interfaces   []SatisfiedInterface `json:"interfaces"`
	AnalysisMode string               `json:"-"`
	Warnings     []string             `json:"-"`
}

type InterfaceMethodUsage struct {
	Kind     sherpa.ReferenceKind `json:"kind,omitempty"`
	Position sherpa.Position      `json:"position"`
	Range    *sherpa.SourceRange  `json:"range,omitempty"`
}

type InterfaceMethod struct {
	Name      string                 `json:"name"`
	Signature string                 `json:"signature,omitempty"`
	Usages    []InterfaceMethodUsage `json:"usages"`
}

type InterfaceResult struct {
	Target                  string             `json:"target"`
	Position                sherpa.Position    `json:"position"`
	Methods                 []InterfaceMethod  `json:"methods"`
	Implementers            []Implementer      `json:"implementers"`
	References              []sherpa.Reference `json:"references"`
	AnalysisMode            string             `json:"-"`
	ReferenceAnalysisMode   string             `json:"-"`
	MethodUsageAnalysisMode string             `json:"-"`
	Warnings                []string           `json:"-"`
	Limitations             []string           `json:"-"`
}

type interfaceGraph struct {
	Interfaces             []interfaceInfo
	Types                  []typeInfo
	ImplementationsByIface map[string][]string
	InterfacesByType       map[string][]string
	EmbeddingByIface       map[string][]string
	AnalysisMode           string
	Warnings               []string
}

type interfaceInfo struct {
	Name      string
	Package   string
	Qualified string
	Position  sherpa.Position
	Methods   methodSet
	Embedded  []interfaceRef
	Type      *types.Interface
}

type typeInfo struct {
	Name      string
	Package   string
	Qualified string
	Position  sherpa.Position
	Methods   methodSet
	Type      types.Type
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

type InterfaceOptions struct {
	BuildTags []string
}

var loadSemanticInterfaceRepository = semantics.LoadRepository

func FindImplementers(root string, target string) (ImplementersResult, error) {
	return FindImplementersWithOptions(root, target, InterfaceOptions{})
}

func FindImplementersWithOptions(root string, target string, options InterfaceOptions) (ImplementersResult, error) {
	return findImplementersWithContext(nil, root, target, options)
}

func FindImplementersWithContext(context *sherpa.SemanticContext, target string, options InterfaceOptions) (ImplementersResult, error) {
	if err := requireInterfaceContext(context, options); err != nil {
		return ImplementersResult{}, err
	}

	return findImplementersWithContext(context, context.Root(), target, options)
}

func findImplementersWithContext(context *sherpa.SemanticContext, root string, target string, options InterfaceOptions) (ImplementersResult, error) {
	graph, err := buildInterfaceGraphWithContext(context, root, options)
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
		AnalysisMode: graph.AnalysisMode,
		Warnings:     graph.Warnings,
	}, nil
}

func FindInterfaces(root string, target string) (InterfacesResult, error) {
	return FindInterfacesWithOptions(root, target, InterfaceOptions{})
}

func FindInterfacesWithOptions(root string, target string, options InterfaceOptions) (InterfacesResult, error) {
	return findInterfacesWithContext(nil, root, target, options)
}

func FindInterfacesWithContext(context *sherpa.SemanticContext, target string, options InterfaceOptions) (InterfacesResult, error) {
	if err := requireInterfaceContext(context, options); err != nil {
		return InterfacesResult{}, err
	}

	return findInterfacesWithContext(context, context.Root(), target, options)
}

func findInterfacesWithContext(context *sherpa.SemanticContext, root string, target string, options InterfaceOptions) (InterfacesResult, error) {
	graph, err := buildInterfaceGraphWithContext(context, root, options)
	if err != nil {
		return InterfacesResult{}, err
	}

	typ, err := findTypeTarget(root, graph, target)
	if err != nil {
		return InterfacesResult{}, err
	}

	return InterfacesResult{
		Target:       typ.Qualified,
		Interfaces:   interfacesForType(graph, typ.Qualified),
		AnalysisMode: graph.AnalysisMode,
		Warnings:     graph.Warnings,
	}, nil
}

func InspectInterface(root string, target string) (InterfaceResult, error) {
	return InspectInterfaceWithOptions(root, target, InterfaceOptions{})
}

func InspectInterfaceWithOptions(root string, target string, options InterfaceOptions) (InterfaceResult, error) {
	return inspectInterfaceWithContext(nil, root, target, options)
}

func InspectInterfaceWithContext(context *sherpa.SemanticContext, target string, options InterfaceOptions) (InterfaceResult, error) {
	if err := requireInterfaceContext(context, options); err != nil {
		return InterfaceResult{}, err
	}

	return inspectInterfaceWithContext(context, context.Root(), target, options)
}

func inspectInterfaceWithContext(context *sherpa.SemanticContext, root string, target string, options InterfaceOptions) (InterfaceResult, error) {
	graph, err := buildInterfaceGraphWithContext(context, root, options)
	if err != nil {
		return InterfaceResult{}, err
	}

	iface, err := findInterfaceTarget(root, graph, target)
	if err != nil {
		return InterfaceResult{}, err
	}

	methods := interfaceMethodsForResult(graph, iface)
	methodUsageAnalysisMode := InterfaceAnalysisModeASTFallback
	methodUsages, methodWarnings, methodUsageTypechecked := findInterfaceMethodUsagesWithContext(context, root, iface, options)
	if methodUsageTypechecked {
		methodUsageAnalysisMode = InterfaceAnalysisModeTypechecked
		methods = attachInterfaceMethodUsages(methods, methodUsages)
	}

	references, referenceAnalysisMode, referenceWarnings := interfaceReferencesWithContext(context, root, iface.Qualified, options)
	limitations := interfaceResultLimitations(graph.AnalysisMode, methodUsageAnalysisMode)

	return InterfaceResult{
		Target:                  iface.Qualified,
		Position:                iface.Position,
		Methods:                 methods,
		Implementers:            implementersForInterface(graph, iface.Qualified),
		References:              references,
		AnalysisMode:            graph.AnalysisMode,
		ReferenceAnalysisMode:   referenceAnalysisMode,
		MethodUsageAnalysisMode: methodUsageAnalysisMode,
		Warnings:                uniqueSortedStrings(append(append([]string{}, graph.Warnings...), append(referenceWarnings, methodWarnings...)...)),
		Limitations:             limitations,
	}, nil
}

func interfaceSignalsForPackages(root string, packages []string, options InterfaceOptions) (interfaceImpactSignals, error) {
	return interfaceSignalsForPackagesWithContext(nil, root, packages, options)
}

func interfaceSignalsForPackagesWithContext(context *sherpa.SemanticContext, root string, packages []string, options InterfaceOptions) (interfaceImpactSignals, error) {
	packages = uniqueSortedStrings(packages)
	if len(packages) == 0 {
		return interfaceImpactSignals{}, nil
	}

	graph, err := buildInterfaceGraphWithContext(context, root, options)
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
		AnalysisMode:    graph.AnalysisMode,
		Warnings:        graph.Warnings,
	}, nil
}

func interfaceSignalsForSymbol(root string, target string, options InterfaceOptions) (interfaceImpactSignals, error) {
	return interfaceSignalsForSymbolWithContext(nil, root, target, options)
}

func interfaceSignalsForSymbolWithContext(context *sherpa.SemanticContext, root string, target string, options InterfaceOptions) (interfaceImpactSignals, error) {
	targetParts := parseInterfaceSymbolTarget(root, target)
	if targetParts.Name == "" {
		return interfaceImpactSignals{}, nil
	}

	graph, err := buildInterfaceGraphWithContext(context, root, options)
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
			AnalysisMode:    graph.AnalysisMode,
			Warnings:        graph.Warnings,
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
		AnalysisMode:    graph.AnalysisMode,
		Warnings:        graph.Warnings,
	}, nil
}

func buildInterfaceGraph(root string, options InterfaceOptions) (interfaceGraph, error) {
	graph, warnings, ok := buildTypecheckedInterfaceGraph(root, options)
	if ok {
		return graph, nil
	}

	graph, err := buildASTInterfaceGraph(root)
	if err != nil {
		return interfaceGraph{}, err
	}

	graph.AnalysisMode = InterfaceAnalysisModeASTFallback
	graph.Warnings = nonNilStrings(warnings)

	return graph, nil
}

func buildInterfaceGraphWithContext(context *sherpa.SemanticContext, root string, options InterfaceOptions) (interfaceGraph, error) {
	if context == nil || !interfaceContextSupportsBuildTags(context, options) {
		return buildInterfaceGraph(root, options)
	}

	graph, warnings, ok := buildTypecheckedInterfaceGraphWithContext(context)
	if ok {
		return graph, nil
	}

	graph, err := buildASTInterfaceGraph(root)
	if err != nil {
		return interfaceGraph{}, err
	}

	graph.AnalysisMode = InterfaceAnalysisModeASTFallback
	graph.Warnings = nonNilStrings(warnings)

	return graph, nil
}

func buildTypecheckedInterfaceGraph(root string, options InterfaceOptions) (interfaceGraph, []string, bool) {
	if !interfaceShouldAttemptTypechecked(root) {
		return interfaceGraph{}, nil, false
	}

	repo, err := loadSemanticInterfaceRepository(root, semantics.LoadOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return interfaceGraph{}, []string{fmt.Sprintf("typechecked interface analysis unavailable: %v", err)}, false
	}

	graph := semanticInterfaceGraph(repo)
	if graph.AnalysisMode == "" {
		return interfaceGraph{}, append([]string{"typechecked interface analysis unavailable: no typechecked packages loaded"}, repo.Warnings...), false
	}

	return graph, nonNilStrings(repo.Warnings), true
}

func buildTypecheckedInterfaceGraphWithContext(context *sherpa.SemanticContext) (interfaceGraph, []string, bool) {
	if context == nil {
		return interfaceGraph{}, nil, false
	}

	repo, attempted, err := context.TypecheckedRepository()
	if !attempted {
		return interfaceGraph{}, nil, false
	}
	if err != nil {
		return interfaceGraph{}, []string{fmt.Sprintf("typechecked interface analysis unavailable: %v", err)}, false
	}

	graph := semanticInterfaceGraph(repo)
	if graph.AnalysisMode == "" {
		return interfaceGraph{}, append([]string{"typechecked interface analysis unavailable: no typechecked packages loaded"}, repo.Warnings...), false
	}

	return graph, nonNilStrings(repo.Warnings), true
}

func interfaceContextSupportsBuildTags(context *sherpa.SemanticContext, options InterfaceOptions) bool {
	if context == nil {
		return false
	}

	return strings.Join(context.BuildTags(), "\x00") == strings.Join(semantics.NormalizeBuildTags(options.BuildTags), "\x00")
}

func requireInterfaceContext(context *sherpa.SemanticContext, options InterfaceOptions) error {
	if context == nil {
		return fmt.Errorf("semantic context is nil")
	}
	if !interfaceContextSupportsBuildTags(context, options) {
		return fmt.Errorf("semantic context build tags do not match interface options")
	}

	return nil
}

func interfaceShouldAttemptTypechecked(root string) bool {
	for _, name := range []string{"go.mod", "go.work"} {
		info, err := os.Stat(filepath.Join(root, name))
		if err == nil && !info.IsDir() {
			return true
		}
	}

	return false
}

func semanticInterfaceGraph(repo semantics.Repository) interfaceGraph {
	modulePath, _ := sherpa.ModulePath(repo.Root)

	var interfaces []interfaceInfo
	var typeInfos []typeInfo
	usablePackages := 0

	for _, pkg := range repo.Packages {
		if !semanticInterfacePackageUsable(pkg) {
			continue
		}

		usablePackages++
		names := pkg.Types.Scope().Names()
		for _, name := range names {
			typeName, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}

			named, ok := typeName.Type().(*types.Named)
			if !ok {
				continue
			}

			iface, ok := named.Underlying().(*types.Interface)
			if ok {
				iface.Complete()
				interfaces = append(interfaces, interfaceInfo{
					Name:      typeName.Name(),
					Package:   pkg.PackagePath,
					Qualified: qualifiedSignalName(pkg.PackagePath, typeName.Name()),
					Position:  interfacePosition(repo.Root, pkg.FileSet, typeName.Pos()),
					Methods:   typecheckedInterfaceMethodSet(iface),
					Embedded:  typecheckedEmbeddedInterfaces(modulePath, iface),
					Type:      iface,
				})
				continue
			}

			typeInfos = append(typeInfos, typeInfo{
				Name:      typeName.Name(),
				Package:   pkg.PackagePath,
				Qualified: qualifiedSignalName(pkg.PackagePath, typeName.Name()),
				Position:  interfacePosition(repo.Root, pkg.FileSet, typeName.Pos()),
				Methods:   typecheckedTypeMethodSet(named),
				Type:      named,
			})
		}
	}

	if usablePackages == 0 {
		return interfaceGraph{}
	}

	sortInterfaceInfos(interfaces)
	sortTypeInfos(typeInfos)

	graph := interfaceGraph{
		Interfaces:   interfaces,
		Types:        typeInfos,
		AnalysisMode: InterfaceAnalysisModeTypechecked,
		Warnings:     nonNilStrings(repo.Warnings),
	}

	return completeInterfaceGraph(graph)
}

func semanticInterfacePackageUsable(pkg semantics.Package) bool {
	return pkg.FileSet != nil && pkg.Types != nil
}

func typecheckedInterfaceMethodSet(iface *types.Interface) methodSet {
	methods := make(methodSet)
	if iface == nil {
		return methods
	}

	iface.Complete()
	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		methods[method.Name()] = method.Type().String()
	}

	return methods
}

func typecheckedTypeMethodSet(typ types.Type) methodSet {
	methods := methodSetFromSelectionSet(types.NewMethodSet(typ))
	if named, ok := typ.(*types.Named); ok {
		mergeMethodSets(methods, methodSetFromSelectionSet(types.NewMethodSet(types.NewPointer(named))))
	}

	return methods
}

func methodSetFromSelectionSet(methodSetValue *types.MethodSet) methodSet {
	methods := make(methodSet)
	if methodSetValue == nil {
		return methods
	}

	for i := 0; i < methodSetValue.Len(); i++ {
		selection := methodSetValue.At(i)
		method := selection.Obj()
		methods[method.Name()] = method.Type().String()
	}

	return methods
}

func typecheckedEmbeddedInterfaces(modulePath string, iface *types.Interface) []interfaceRef {
	if iface == nil {
		return nil
	}

	var embedded []interfaceRef
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		ref, ok := typecheckedInterfaceRef(modulePath, iface.EmbeddedType(i))
		if ok {
			embedded = append(embedded, ref)
		}
	}

	return embedded
}

func typecheckedInterfaceRef(modulePath string, typ types.Type) (interfaceRef, bool) {
	named, ok := typ.(*types.Named)
	if !ok {
		return interfaceRef{}, false
	}
	if named.Obj() == nil || named.Obj().Pkg() == nil {
		return interfaceRef{}, false
	}

	packagePath, ok := localInterfaceImportPackage(named.Obj().Pkg().Path(), modulePath)
	if !ok {
		return interfaceRef{}, false
	}

	return interfaceRef{Package: packagePath, Name: named.Obj().Name()}, true
}

func buildASTInterfaceGraph(root string) (interfaceGraph, error) {
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
		Interfaces:   interfaces,
		Types:        types,
		AnalysisMode: InterfaceAnalysisModeASTFallback,
	}

	return completeInterfaceGraph(graph), nil
}

func completeInterfaceGraph(graph interfaceGraph) interfaceGraph {
	graph.ImplementationsByIface = make(map[string][]string)
	graph.InterfacesByType = make(map[string][]string)
	graph.EmbeddingByIface = interfaceEmbeddingMap(graph.Interfaces)
	graph.Warnings = nonNilStrings(graph.Warnings)

	for _, iface := range graph.Interfaces {
		for _, typ := range graph.Types {
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

	return graph
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

func interfaceMethodsForResult(graph interfaceGraph, iface interfaceInfo) []InterfaceMethod {
	methods := resolvedInterfaceMethodsForResult(graph, iface.Qualified, make(map[string]struct{}))
	names := make([]string, 0, len(methods))
	for name := range methods {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]InterfaceMethod, 0, len(names))
	for _, name := range names {
		result = append(result, InterfaceMethod{
			Name:      name,
			Signature: formatInterfaceMethodSignature(methods[name]),
			Usages:    []InterfaceMethodUsage{},
		})
	}

	return result
}

func resolvedInterfaceMethodsForResult(graph interfaceGraph, qualified string, seen map[string]struct{}) methodSet {
	if _, ok := seen[qualified]; ok {
		return methodSet{}
	}
	seen[qualified] = struct{}{}

	interfacesByQualified := interfacesByQualifiedName(graph.Interfaces)
	iface, ok := interfacesByQualified[qualified]
	if !ok {
		return methodSet{}
	}

	methods := cloneMethodSet(iface.Methods)
	for _, embedded := range iface.Embedded {
		embeddedQualified := qualifiedSignalName(embedded.Package, embedded.Name)
		mergeMethodSets(methods, resolvedInterfaceMethodsForResult(graph, embeddedQualified, seen))
	}

	return methods
}

func formatInterfaceMethodSignature(signature string) string {
	signature = strings.TrimSpace(signature)
	if signature == "" || strings.HasPrefix(signature, "func(") {
		return signature
	}

	params, results, ok := strings.Cut(signature, "->")
	if !ok {
		return signature
	}

	params = strings.TrimSpace(params)
	results = strings.TrimSpace(results)
	if results == "" {
		return "func(" + params + ")"
	}
	if strings.Contains(results, ",") {
		return "func(" + params + ") (" + results + ")"
	}

	return "func(" + params + ") " + results
}

func attachInterfaceMethodUsages(methods []InterfaceMethod, usages map[string][]InterfaceMethodUsage) []InterfaceMethod {
	for i := range methods {
		methods[i].Usages = nonNilInterfaceMethodUsages(usages[methods[i].Name])
	}

	return methods
}

func interfaceReferences(root string, target string, options InterfaceOptions) ([]sherpa.Reference, string, []string) {
	return interfaceReferencesWithContext(nil, root, target, options)
}

func interfaceReferencesWithContext(context *sherpa.SemanticContext, root string, target string, options InterfaceOptions) ([]sherpa.Reference, string, []string) {
	referenceOptions := sherpa.ReferenceOptions{
		BuildTags: options.BuildTags,
	}

	var report sherpa.ReferenceReport
	var err error
	if context != nil && interfaceContextSupportsBuildTags(context, options) {
		report, err = sherpa.FindReferenceReportWithContext(context, target, referenceOptions)
	} else {
		report, err = sherpa.FindReferenceReportWithOptions(root, target, referenceOptions)
	}
	if err != nil {
		return []sherpa.Reference{}, sherpa.ReferenceAnalysisModeASTFallback, []string{fmt.Sprintf("interface references unavailable: %v", err)}
	}

	analysisMode := strings.TrimSpace(report.AnalysisMode)
	if analysisMode == "" {
		analysisMode = sherpa.ReferenceAnalysisModeASTFallback
	}

	return nonNilInterfaceReferences(report.References), analysisMode, nonNilStrings(report.Warnings)
}

func findInterfaceMethodUsages(root string, iface interfaceInfo, options InterfaceOptions) (map[string][]InterfaceMethodUsage, []string, bool) {
	return findInterfaceMethodUsagesWithContext(nil, root, iface, options)
}

func findInterfaceMethodUsagesWithContext(context *sherpa.SemanticContext, root string, iface interfaceInfo, options InterfaceOptions) (map[string][]InterfaceMethodUsage, []string, bool) {
	usages := make(map[string][]InterfaceMethodUsage)
	if !interfaceShouldAttemptTypechecked(root) {
		return usages, nil, false
	}

	var repo semantics.Repository
	var err error
	if context != nil && interfaceContextSupportsBuildTags(context, options) {
		var attempted bool
		repo, attempted, err = context.TypecheckedRepository()
		if !attempted {
			return usages, nil, false
		}
	} else {
		repo, err = loadSemanticInterfaceRepository(root, semantics.LoadOptions{
			BuildTags: options.BuildTags,
		})
	}
	if err != nil {
		return usages, []string{fmt.Sprintf("typechecked interface method usage unavailable: %v", err)}, false
	}

	methodObjects, ok := semanticInterfaceMethodObjects(repo, iface)
	if !ok {
		warnings := append([]string{"typechecked interface method usage unavailable: target interface was not loaded"}, repo.Warnings...)
		return usages, nonNilStrings(warnings), false
	}
	if len(methodObjects) == 0 {
		return usages, nonNilStrings(repo.Warnings), true
	}

	for _, pkg := range repo.Packages {
		if pkg.FileSet == nil || pkg.TypesInfo == nil {
			continue
		}

		for _, file := range pkg.Files {
			var stack []ast.Node
			ast.Inspect(file, func(node ast.Node) bool {
				if node == nil {
					stack = stack[:len(stack)-1]
					return false
				}

				parent := ast.Node(nil)
				if len(stack) > 0 {
					parent = stack[len(stack)-1]
				}
				stack = append(stack, node)

				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				selection := pkg.TypesInfo.Selections[selector]
				if selection == nil {
					return true
				}

				methodName, ok := methodObjects[selection.Obj()]
				if !ok {
					return true
				}

				usages[methodName] = append(usages[methodName], InterfaceMethodUsage{
					Kind:     interfaceMethodUsageKind(parent, selector),
					Position: interfacePosition(repo.Root, pkg.FileSet, selector.Sel.Pos()),
					Range:    interfaceSourceRange(repo.Root, pkg.FileSet, selector.Sel.Pos(), selector.Sel.End()),
				})

				return true
			})
		}
	}

	for method := range usages {
		sortInterfaceMethodUsages(usages[method])
	}

	return usages, nonNilStrings(repo.Warnings), true
}

func semanticInterfaceMethodObjects(repo semantics.Repository, iface interfaceInfo) (map[types.Object]string, bool) {
	for _, pkg := range repo.Packages {
		if pkg.PackagePath != iface.Package || pkg.Types == nil {
			continue
		}

		object, ok := pkg.Types.Scope().Lookup(iface.Name).(*types.TypeName)
		if !ok {
			continue
		}

		named, ok := object.Type().(*types.Named)
		if !ok {
			continue
		}

		interfaceType, ok := named.Underlying().(*types.Interface)
		if !ok {
			continue
		}

		interfaceType.Complete()
		methods := make(map[types.Object]string)
		for i := 0; i < interfaceType.NumMethods(); i++ {
			method := interfaceType.Method(i)
			methods[method] = method.Name()
		}

		return methods, true
	}

	return nil, false
}

func interfaceMethodUsageKind(parent ast.Node, selector *ast.SelectorExpr) sherpa.ReferenceKind {
	call, ok := parent.(*ast.CallExpr)
	if ok && call.Fun == selector {
		return sherpa.ReferenceKindCall
	}

	return sherpa.ReferenceKindUsage
}

func interfaceSourceRange(root string, fileSet *token.FileSet, start token.Pos, end token.Pos) *sherpa.SourceRange {
	if !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &sherpa.SourceRange{
		Start: interfacePosition(root, fileSet, start),
		End:   interfacePosition(root, fileSet, end),
	}
}

func sortInterfaceMethodUsages(usages []InterfaceMethodUsage) {
	sort.Slice(usages, func(i int, j int) bool {
		if usages[i].Position.File != usages[j].Position.File {
			return usages[i].Position.File < usages[j].Position.File
		}
		if usages[i].Position.Line != usages[j].Position.Line {
			return usages[i].Position.Line < usages[j].Position.Line
		}
		if usages[i].Position.Column != usages[j].Position.Column {
			return usages[i].Position.Column < usages[j].Position.Column
		}

		return usages[i].Kind < usages[j].Kind
	})
}

func interfaceResultLimitations(analysisMode string, methodUsageAnalysisMode string) []string {
	limitations := []string{
		"Interface method usage reports statically visible selector usages only.",
		"Dynamic dispatch, reflection, and generated runtime wiring may hide additional usage.",
	}
	if analysisMode != InterfaceAnalysisModeTypechecked || methodUsageAnalysisMode != InterfaceAnalysisModeTypechecked {
		limitations = append(limitations, "Interface method usage requires typechecked package loading; AST fallback keeps methods, implementers, and type references conservative.")
	}

	return limitations
}

func nonNilInterfaceReferences(references []sherpa.Reference) []sherpa.Reference {
	if references == nil {
		return []sherpa.Reference{}
	}

	return references
}

func nonNilInterfaceMethodUsages(usages []InterfaceMethodUsage) []InterfaceMethodUsage {
	if usages == nil {
		return []InterfaceMethodUsage{}
	}

	return usages
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

func sortInterfaceInfos(interfaces []interfaceInfo) {
	sort.Slice(interfaces, func(i int, j int) bool {
		if interfaces[i].Qualified != interfaces[j].Qualified {
			return interfaces[i].Qualified < interfaces[j].Qualified
		}
		if interfaces[i].Position.File != interfaces[j].Position.File {
			return interfaces[i].Position.File < interfaces[j].Position.File
		}

		return interfaces[i].Position.Line < interfaces[j].Position.Line
	})
}

func sortTypeInfos(types []typeInfo) {
	sort.Slice(types, func(i int, j int) bool {
		if types[i].Qualified != types[j].Qualified {
			return types[i].Qualified < types[j].Qualified
		}
		if types[i].Position.File != types[j].Position.File {
			return types[i].Position.File < types[j].Position.File
		}

		return types[i].Position.Line < types[j].Position.Line
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
	if typ.Type != nil && iface.Type != nil {
		return typecheckedTypeImplementsInterface(typ.Type, iface.Type)
	}

	for method, ifaceSignature := range iface.Methods {
		typeSignature, ok := typ.Methods[method]
		if !ok || typeSignature != ifaceSignature {
			return false
		}
	}

	return true
}

func typecheckedTypeImplementsInterface(typ types.Type, iface *types.Interface) bool {
	if typ == nil || iface == nil {
		return false
	}

	iface.Complete()
	if types.Implements(typ, iface) {
		return true
	}

	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}

	return types.Implements(types.NewPointer(named), iface)
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
	if localPath, ok := sherpa.WorkspacePackagePathForImportPath(root, value); ok {
		return localPath
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
