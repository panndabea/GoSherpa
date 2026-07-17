package sherpa

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
)

type RelationshipSymbolIdentity struct {
	Package       string
	PackageName   string
	Name          string
	Receiver      string
	QualifiedName string
	Kind          SymbolKind
	Position      Position
	Range         *SourceRange
}

type ReferenceRelationship struct {
	Package      string
	File         string
	Source       RelationshipSymbolIdentity
	Target       RelationshipSymbolIdentity
	Kind         ReferenceKind
	AnalysisMode string
	Position     Position
	Range        *SourceRange
	Limitations  []string
}

type CallRelationship struct {
	Package      string
	File         string
	Source       RelationshipSymbolIdentity
	Target       RelationshipSymbolIdentity
	Scope        CallScope
	AnalysisMode string
	Position     Position
	Range        *SourceRange
	Limitations  []string
}

type PossibleCallRelationship struct {
	Package      string
	File         string
	Source       RelationshipSymbolIdentity
	Target       RelationshipSymbolIdentity
	Scope        CallScope
	Reason       PossibleCallReason
	AnalysisMode string
	Position     Position
	Range        *SourceRange
	Limitations  []string
}

func BuildReferenceRelationshipsWithOptions(root string, options ReferenceOptions) ([]ReferenceRelationship, string, []string, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, "", nil, err
	}

	var packages []referencePackage
	var analysisMode string
	var warnings []string
	if referenceShouldAttemptTypechecked(rootPath) {
		cache := newReferenceAnalysisCache(rootPath, options)
		warnings = nonNilStrings(cache.Warnings)
		if len(cache.Packages) > 0 {
			packages = cache.Packages
			analysisMode = ReferenceAnalysisModeTypechecked
		}
	}

	if len(packages) == 0 {
		files, err := FindGoFiles(rootPath)
		if err != nil {
			return nil, "", warnings, err
		}

		sort.Strings(files)
		packages, err = parseReferencePackages(rootPath, files)
		if err != nil {
			return nil, "", warnings, err
		}
		analysisMode = ReferenceAnalysisModeASTFallback
	}

	relationships := buildReferenceRelationshipsFromPackages(rootPath, packages, options, analysisMode)
	return relationships, analysisMode, warnings, nil
}

func BuildCallRelationshipsWithOptions(root string, options CallOptions) ([]CallRelationship, string, []string, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, "", nil, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfos(rootPath, options)
	if err != nil {
		return nil, analysisMode, warnings, err
	}

	if options.IncludeTests {
		testFunctions, testWarnings, err := collectTestCallerFunctionInfos(rootPath, options)
		warnings = uniqueSorted(append(warnings, testWarnings...))
		if err != nil {
			return nil, analysisMode, warnings, err
		}
		functions = append(functions, testFunctions...)
		sortFunctionInfos(functions)
	}

	return buildCallRelationshipsFromFunctions(functions, analysisMode), analysisMode, warnings, nil
}

func BuildPossibleCallRelationshipsWithOptions(root string, options CallOptions) ([]PossibleCallRelationship, string, []string, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, "", nil, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfos(rootPath, options)
	if err != nil {
		return nil, analysisMode, warnings, err
	}

	if options.IncludeTests {
		testFunctions, testWarnings, err := collectTestCallerFunctionInfos(rootPath, options)
		warnings = uniqueSorted(append(warnings, testWarnings...))
		if err != nil {
			return nil, analysisMode, warnings, err
		}
		functions = append(functions, testFunctions...)
		sortFunctionInfos(functions)
	}

	return buildPossibleCallRelationshipsFromFunctions(functions, analysisMode), analysisMode, warnings, nil
}

func buildReferenceRelationshipsFromPackages(root string, packages []referencePackage, options ReferenceOptions, analysisMode string) []ReferenceRelationship {
	definitions := referenceRelationshipDefinitions(root, packages)
	seen := make(map[string]struct{})
	var records []ReferenceRelationship

	addRecord := func(pkg referencePackage, source RelationshipSymbolIdentity, target RelationshipSymbolIdentity, kind ReferenceKind, start token.Pos, end token.Pos) {
		if options.Kind != "" && kind != options.Kind {
			return
		}
		if !start.IsValid() {
			return
		}

		position := relationshipPosition(root, pkg.FileSet, start)
		record := ReferenceRelationship{
			Package:      pkg.Package,
			File:         position.File,
			Source:       source,
			Target:       target,
			Kind:         kind,
			AnalysisMode: analysisMode,
			Position:     position,
			Range:        sourceRangeRelativeToRoot(root, pkg.FileSet, start, end),
		}
		key := referenceRelationshipKey(record)
		if _, ok := seen[key]; ok {
			return
		}

		seen[key] = struct{}{}
		records = append(records, record)
	}

	for _, pkg := range packages {
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
				source := referenceRelationshipSource(definitions, pkg, stack)

				switch node := node.(type) {
				case *ast.Ident:
					object, definition := referenceIdentObject(pkg.Info, node)
					target, ok := referenceRelationshipIdentityForObject(root, pkg, definitions, object)
					if !ok {
						return true
					}
					addRecord(pkg, source, target, referenceKindForIdent(object, definition, parent, node), node.Pos(), node.End())
				case *ast.SelectorExpr:
					object := pkg.Info.Uses[node.Sel]
					selection := pkg.Info.Selections[node]
					targetObject := object
					if selection != nil && selection.Obj() != nil {
						targetObject = selection.Obj()
					}
					target, ok := referenceRelationshipIdentityForObject(root, pkg, definitions, targetObject)
					if !ok {
						return true
					}
					addRecord(pkg, source, target, referenceKindForTypecheckedSelector(object, selection, parent, node), node.Sel.Pos(), node.Sel.End())
				}

				return true
			})
		}
	}

	sort.Slice(records, func(i int, j int) bool {
		return referenceRelationshipKey(records[i]) < referenceRelationshipKey(records[j])
	})

	return records
}

func referenceRelationshipDefinitions(root string, packages []referencePackage) map[types.Object]RelationshipSymbolIdentity {
	definitions := make(map[types.Object]RelationshipSymbolIdentity)
	modulePath := readModulePath(root)
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					object := pkg.Info.Defs[decl.Name]
					if object == nil {
						continue
					}
					receiver := receiverTypeName(decl)
					kind := SymbolKindFunction
					if receiver != "" {
						kind = SymbolKindMethod
					}
					definitions[object] = RelationshipSymbolIdentity{
						Package:       pkg.Package,
						PackageName:   pkg.Name,
						Name:          decl.Name.Name,
						Receiver:      receiver,
						QualifiedName: FormatPackageQualifiedTarget(pkg.Package, relationshipDisplayName(receiver, decl.Name.Name), modulePath),
						Kind:          kind,
						Position:      relationshipPosition(root, pkg.FileSet, decl.Pos()),
						Range:         sourceRangeRelativeToRoot(root, pkg.FileSet, decl.Pos(), decl.End()),
					}
				case *ast.GenDecl:
					for _, spec := range decl.Specs {
						switch spec := spec.(type) {
						case *ast.TypeSpec:
							object := pkg.Info.Defs[spec.Name]
							if object == nil {
								continue
							}
							definitions[object] = RelationshipSymbolIdentity{
								Package:       pkg.Package,
								PackageName:   pkg.Name,
								Name:          spec.Name.Name,
								QualifiedName: FormatPackageQualifiedTarget(pkg.Package, spec.Name.Name, modulePath),
								Kind:          relationshipTypeSpecKind(spec),
								Position:      relationshipPosition(root, pkg.FileSet, spec.Pos()),
								Range:         sourceRangeRelativeToRoot(root, pkg.FileSet, spec.Pos(), spec.End()),
							}
							addRelationshipFieldDefinitions(root, modulePath, pkg, definitions, spec)
						case *ast.ValueSpec:
							for _, name := range spec.Names {
								object := pkg.Info.Defs[name]
								if object == nil {
									continue
								}
								definitions[object] = RelationshipSymbolIdentity{
									Package:       pkg.Package,
									PackageName:   pkg.Name,
									Name:          name.Name,
									QualifiedName: FormatPackageQualifiedTarget(pkg.Package, name.Name, modulePath),
									Position:      relationshipPosition(root, pkg.FileSet, name.Pos()),
									Range:         sourceRangeRelativeToRoot(root, pkg.FileSet, name.Pos(), name.End()),
								}
							}
						}
					}
				}
			}
		}
	}

	return definitions
}

func addRelationshipFieldDefinitions(root string, modulePath string, pkg referencePackage, definitions map[types.Object]RelationshipSymbolIdentity, spec *ast.TypeSpec) {
	structType, ok := spec.Type.(*ast.StructType)
	if !ok {
		return
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			object := pkg.Info.Defs[name]
			if object == nil {
				continue
			}
			displayName := relationshipDisplayName(spec.Name.Name, name.Name)
			definitions[object] = RelationshipSymbolIdentity{
				Package:       pkg.Package,
				PackageName:   pkg.Name,
				Name:          name.Name,
				Receiver:      spec.Name.Name,
				QualifiedName: FormatPackageQualifiedTarget(pkg.Package, displayName, modulePath),
				Position:      relationshipPosition(root, pkg.FileSet, name.Pos()),
				Range:         sourceRangeRelativeToRoot(root, pkg.FileSet, name.Pos(), name.End()),
			}
		}
	}
}

func referenceRelationshipIdentityForObject(root string, pkg referencePackage, definitions map[types.Object]RelationshipSymbolIdentity, object types.Object) (RelationshipSymbolIdentity, bool) {
	if object == nil {
		return RelationshipSymbolIdentity{}, false
	}
	if identity, ok := definitions[object]; ok {
		return identity, true
	}

	modulePath := readModulePath(root)
	packagePath, ok := relationshipObjectPackage(root, modulePath, pkg, object)
	if !ok {
		return RelationshipSymbolIdentity{}, false
	}

	receiver := ""
	if function, ok := object.(*types.Func); ok {
		receiver = callFuncReceiverName(function)
	}
	name := object.Name()
	if name == "" {
		return RelationshipSymbolIdentity{}, false
	}

	return RelationshipSymbolIdentity{
		Package:       packagePath,
		Name:          name,
		Receiver:      receiver,
		QualifiedName: FormatPackageQualifiedTarget(packagePath, relationshipDisplayName(receiver, name), modulePath),
		Kind:          relationshipSymbolKindForObject(object),
		Position:      relationshipPosition(root, pkg.FileSet, object.Pos()),
	}, true
}

func relationshipObjectPackage(root string, modulePath string, pkg referencePackage, object types.Object) (string, bool) {
	objectPackage := object.Pkg()
	if objectPackage == nil {
		return pkg.Package, true
	}
	if objectPackage.Path() == pkg.ImportPath {
		return pkg.Package, true
	}
	if localPath, ok := WorkspacePackagePathForImportPath(root, objectPackage.Path()); ok {
		return localPath, true
	}
	if localPath, ok := callLocalImportPackage(objectPackage.Path(), modulePath); ok {
		return localPath, true
	}

	return "", false
}

func referenceRelationshipSource(definitions map[types.Object]RelationshipSymbolIdentity, pkg referencePackage, stack []ast.Node) RelationshipSymbolIdentity {
	for i := len(stack) - 1; i >= 0; i-- {
		switch node := stack[i].(type) {
		case *ast.FuncDecl:
			if identity, ok := definitions[pkg.Info.Defs[node.Name]]; ok {
				return identity
			}
		case *ast.TypeSpec:
			if identity, ok := definitions[pkg.Info.Defs[node.Name]]; ok {
				return identity
			}
		}
	}

	return RelationshipSymbolIdentity{}
}

func buildCallRelationshipsFromFunctions(functions []functionInfo, analysisMode string) []CallRelationship {
	seen := make(map[string]struct{})
	var records []CallRelationship

	for _, function := range functions {
		source := relationshipIdentityFromFunction(function)
		limitations := collectDynamicCallLimitations([]functionInfo{function})
		for _, reference := range collectCallReferencesFromFunction(function) {
			matches := matchingFunctionInfosForCall(functions, function, reference.Expr)
			if len(matches) == 0 {
				record := CallRelationship{
					Package:      function.Package,
					File:         reference.Position.File,
					Source:       source,
					Target:       relationshipIdentityFromCallReference(function, reference),
					Scope:        callReferenceScope(function, reference),
					AnalysisMode: analysisMode,
					Position:     reference.Position,
					Range:        reference.Range,
					Limitations:  limitations,
				}
				key := callRelationshipKey(record)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				records = append(records, record)
				continue
			}

			for _, match := range matches {
				record := CallRelationship{
					Package:      function.Package,
					File:         reference.Position.File,
					Source:       source,
					Target:       relationshipIdentityFromFunction(match),
					Scope:        callReferenceScope(function, reference),
					AnalysisMode: analysisMode,
					Position:     reference.Position,
					Range:        reference.Range,
					Limitations:  limitations,
				}
				key := callRelationshipKey(record)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				records = append(records, record)
			}
		}
	}

	sort.Slice(records, func(i int, j int) bool {
		return callRelationshipKey(records[i]) < callRelationshipKey(records[j])
	})

	return records
}

func buildPossibleCallRelationshipsFromFunctions(functions []functionInfo, analysisMode string) []PossibleCallRelationship {
	seen := make(map[string]struct{})
	var records []PossibleCallRelationship
	catalog := newInterfaceDispatchCatalog(functions)

	for _, function := range functions {
		source := relationshipIdentityFromFunction(function)
		limitations := collectDynamicCallLimitations([]functionInfo{function})
		for _, possibleCall := range collectPossibleCallsFromFunctionWithCatalog(function, catalog) {
			record := PossibleCallRelationship{
				Package:      function.Package,
				File:         possibleCall.Position.File,
				Source:       source,
				Target:       relationshipIdentityFromPossibleCall(functions, function, possibleCall),
				Scope:        possibleCall.Scope,
				Reason:       possibleCall.Reason,
				AnalysisMode: analysisMode,
				Position:     possibleCall.Position,
				Range:        possibleCall.Range,
				Limitations:  limitations,
			}
			key := possibleCallRelationshipKey(record)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			records = append(records, record)
		}
	}

	sort.Slice(records, func(i int, j int) bool {
		return possibleCallRelationshipKey(records[i]) < possibleCallRelationshipKey(records[j])
	})

	return records
}

func relationshipIdentityFromFunction(function functionInfo) RelationshipSymbolIdentity {
	displayName := relationshipDisplayName(function.Receiver, function.Name)
	kind := SymbolKindFunction
	if function.Receiver != "" {
		kind = SymbolKindMethod
	}

	return RelationshipSymbolIdentity{
		Package:       function.Package,
		PackageName:   function.PackageName,
		Name:          function.Name,
		Receiver:      function.Receiver,
		QualifiedName: FormatPackageQualifiedTarget(function.Package, displayName, function.ModulePath),
		Kind:          kind,
		Position:      function.Position,
		Range:         sourceRangeRelativeToRoot(function.Root, function.FileSet, function.Decl.Pos(), function.Decl.End()),
	}
}

func relationshipIdentityFromPossibleCall(functions []functionInfo, source functionInfo, possibleCall PossibleCall) RelationshipSymbolIdentity {
	receiver, name := relationshipSplitDisplayName(possibleCall.Callee)
	target := callTarget{
		Package:  possibleCall.calleePackage,
		Receiver: receiver,
		Name:     name,
	}
	for _, function := range functions {
		if functionMatchesCallTarget(function, target) {
			return relationshipIdentityFromFunction(function)
		}
	}

	qualifiedName := strings.TrimSpace(possibleCall.Callee)
	if possibleCall.calleePackage != "" {
		qualifiedName = FormatPackageQualifiedTarget(possibleCall.calleePackage, relationshipDisplayName(receiver, name), source.ModulePath)
	}

	kind := SymbolKindFunction
	if receiver != "" {
		kind = SymbolKindMethod
	}

	return RelationshipSymbolIdentity{
		Package:       possibleCall.calleePackage,
		Name:          name,
		Receiver:      receiver,
		QualifiedName: qualifiedName,
		Kind:          kind,
		Position:      possibleCall.Position,
		Range:         possibleCall.Range,
	}
}

func relationshipIdentityFromCallReference(function functionInfo, reference callReference) RelationshipSymbolIdentity {
	receiver, name := relationshipSplitDisplayName(reference.Name)
	return RelationshipSymbolIdentity{
		Name:          name,
		Receiver:      receiver,
		QualifiedName: reference.Name,
		Position:      reference.Position,
		Range:         reference.Range,
	}
}

func relationshipPosition(root string, fileSet *token.FileSet, pos token.Pos) Position {
	if fileSet == nil || !pos.IsValid() {
		return Position{}
	}

	position := fileSet.Position(pos)
	return positionRelativeToRoot(root, Position{
		File:   position.Filename,
		Line:   position.Line,
		Column: position.Column,
	})
}

func relationshipDisplayName(receiver string, name string) string {
	if strings.TrimSpace(receiver) == "" {
		return strings.TrimSpace(name)
	}

	return strings.TrimSpace(receiver) + "." + strings.TrimSpace(name)
}

func relationshipSplitDisplayName(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	lastDot := strings.LastIndex(value, ".")
	if lastDot <= 0 || lastDot == len(value)-1 {
		return "", value
	}

	return value[:lastDot], value[lastDot+1:]
}

func relationshipTypeSpecKind(spec *ast.TypeSpec) SymbolKind {
	if spec == nil {
		return ""
	}
	if _, ok := spec.Type.(*ast.StructType); ok {
		return SymbolKindStruct
	}
	if _, ok := spec.Type.(*ast.InterfaceType); ok {
		return SymbolKindInterface
	}
	if spec.Assign.IsValid() {
		return SymbolKindAlias
	}

	return ""
}

func relationshipSymbolKindForObject(object types.Object) SymbolKind {
	switch object := object.(type) {
	case *types.Func:
		if callFuncReceiverName(object) != "" {
			return SymbolKindMethod
		}
		return SymbolKindFunction
	case *types.TypeName:
		underlying := types.Unalias(object.Type()).Underlying()
		switch underlying.(type) {
		case *types.Struct:
			return SymbolKindStruct
		case *types.Interface:
			return SymbolKindInterface
		default:
			return SymbolKindAlias
		}
	default:
		return ""
	}
}

func referenceRelationshipKey(record ReferenceRelationship) string {
	return strings.Join([]string{
		record.Package,
		record.File,
		relationshipIdentityKey(record.Source),
		relationshipIdentityKey(record.Target),
		string(record.Kind),
		record.AnalysisMode,
		positionRelationshipKey(record.Position),
		rangeRelationshipKey(record.Range),
	}, "\x00")
}

func callRelationshipKey(record CallRelationship) string {
	return strings.Join([]string{
		record.Package,
		record.File,
		relationshipIdentityKey(record.Source),
		relationshipIdentityKey(record.Target),
		string(record.Scope),
		record.AnalysisMode,
		positionRelationshipKey(record.Position),
		rangeRelationshipKey(record.Range),
		strings.Join(record.Limitations, "\x00"),
	}, "\x00")
}

func possibleCallRelationshipKey(record PossibleCallRelationship) string {
	return strings.Join([]string{
		record.Package,
		record.File,
		relationshipIdentityKey(record.Source),
		relationshipIdentityKey(record.Target),
		string(record.Scope),
		string(record.Reason),
		record.AnalysisMode,
		positionRelationshipKey(record.Position),
		rangeRelationshipKey(record.Range),
		strings.Join(record.Limitations, "\x00"),
	}, "\x00")
}

func relationshipIdentityKey(identity RelationshipSymbolIdentity) string {
	return strings.Join([]string{
		identity.Package,
		identity.PackageName,
		identity.Name,
		identity.Receiver,
		identity.QualifiedName,
		string(identity.Kind),
		positionRelationshipKey(identity.Position),
		rangeRelationshipKey(identity.Range),
	}, "\x00")
}

func positionRelationshipKey(position Position) string {
	return fmt.Sprintf("%s:%08d:%08d", filepath.ToSlash(position.File), position.Line, position.Column)
}

func rangeRelationshipKey(sourceRange *SourceRange) string {
	if sourceRange == nil {
		return ""
	}

	return positionRelationshipKey(sourceRange.Start) + "\x00" + positionRelationshipKey(sourceRange.End)
}
