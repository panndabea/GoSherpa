package sherpa

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/supertabaluga/gosherpa/internal/semantics"
)

type Reference struct {
	Name     string        `json:"name"`
	Kind     ReferenceKind `json:"kind,omitempty"`
	Position Position      `json:"position"`
}

const (
	ReferenceAnalysisModeTypechecked = "typechecked"
	ReferenceAnalysisModeASTFallback = "ast-fallback"
)

type ReferenceReport struct {
	Target       string      `json:"target"`
	References   []Reference `json:"references"`
	AnalysisMode string      `json:"analysisMode"`
	Warnings     []string    `json:"warnings"`
}

type ReferenceKind string

const (
	ReferenceKindDefinition  ReferenceKind = "definition"
	ReferenceKindCall        ReferenceKind = "call"
	ReferenceKindTypeUsage   ReferenceKind = "type_usage"
	ReferenceKindFieldAccess ReferenceKind = "field_access"
	ReferenceKindUsage       ReferenceKind = "usage"
)

type ReferenceOptions struct {
	Kind      ReferenceKind
	BuildTags []string
}

var loadSemanticReferenceRepository = semantics.LoadRepository

type referenceTarget struct {
	Package  string
	Receiver string
	Name     string
}

type referencePackage struct {
	Package    string
	ImportPath string
	Name       string
	FileSet    *token.FileSet
	Files      []*ast.File
	Info       types.Info
}

func FindReferences(root string, name string) ([]Reference, error) {
	return FindReferencesWithOptions(root, name, ReferenceOptions{})
}

func FindReferencesWithOptions(root string, name string, options ReferenceOptions) ([]Reference, error) {
	report, err := FindReferenceReportWithOptions(root, name, options)
	if err != nil {
		return nil, err
	}

	return report.References, nil
}

func FindReferenceReport(root string, name string) (ReferenceReport, error) {
	return FindReferenceReportWithOptions(root, name, ReferenceOptions{})
}

func FindReferenceReportWithOptions(root string, name string, options ReferenceOptions) (ReferenceReport, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return ReferenceReport{}, err
	}

	target, err := normalizeReferenceTarget(rootPath, name)
	if err != nil {
		return ReferenceReport{}, err
	}

	var report ReferenceReport
	if referenceShouldAttemptTypechecked(rootPath) {
		var ok bool
		report, ok = findTypecheckedReferenceReport(rootPath, target, options)
		if ok {
			return report, nil
		}
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return ReferenceReport{}, err
	}

	sort.Strings(files)

	packages, err := parseReferencePackages(rootPath, files)
	if err != nil {
		return ReferenceReport{}, err
	}

	targetPackages := referenceTargetPackageImports(packages, target)
	packageNames := referencePackageNames(packages)

	var refs []Reference
	for _, pkg := range packages {
		refs = append(refs, findReferencesInPackage(rootPath, pkg, target, targetPackages, packageNames)...)
	}

	refs = filterReferences(refs, options)
	sortReferences(refs)

	return ReferenceReport{
		Target:       target.String(),
		References:   nonNilReferences(refs),
		AnalysisMode: ReferenceAnalysisModeASTFallback,
		Warnings:     nonNilStrings(report.Warnings),
	}, nil
}

func referenceShouldAttemptTypechecked(root string) bool {
	info, err := os.Stat(filepath.Join(root, "go.mod"))
	return err == nil && !info.IsDir()
}

func normalizeReferenceTarget(root string, name string) (referenceTarget, error) {
	value := strings.TrimSpace(name)
	if value == "" {
		return referenceTarget{}, fmt.Errorf("reference target is empty")
	}

	packagePath, symbol, hasPackage, err := splitPackageQualifiedTarget(root, value)
	if err != nil {
		return referenceTarget{}, err
	}
	if !hasPackage && (strings.Contains(value, "/") || strings.Contains(value, "\\")) {
		return referenceTarget{}, fmt.Errorf("invalid reference target: %s", value)
	}

	segments := strings.Split(symbol, ".")

	if len(segments) != 1 && len(segments) != 2 {
		return referenceTarget{}, fmt.Errorf("invalid reference target: %s", value)
	}

	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(segment) {
			return referenceTarget{}, fmt.Errorf("invalid reference target: %s", value)
		}
	}

	target := referenceTarget{
		Package: packagePath,
		Name:    segments[len(segments)-1],
	}
	if len(segments) == 2 {
		target.Receiver = segments[0]
	}

	return target, nil
}

func (target referenceTarget) String() string {
	symbol := target.Symbol()
	if target.Package == "." {
		return symbol
	}
	if target.Package != "" {
		return target.Package + "." + symbol
	}

	return symbol
}

func (target referenceTarget) Symbol() string {
	if target.Receiver == "" {
		return target.Name
	}

	return target.Receiver + "." + target.Name
}

func ParseReferenceKind(value string) (ReferenceKind, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")

	switch normalized {
	case string(ReferenceKindDefinition), "def":
		return ReferenceKindDefinition, true
	case string(ReferenceKindCall):
		return ReferenceKindCall, true
	case string(ReferenceKindTypeUsage), "type":
		return ReferenceKindTypeUsage, true
	case string(ReferenceKindFieldAccess), "field":
		return ReferenceKindFieldAccess, true
	case string(ReferenceKindUsage), "reference":
		return ReferenceKindUsage, true
	default:
		return "", false
	}
}

func parseReferencePackages(root string, files []string) ([]referencePackage, error) {
	modulePath := readModulePath(root)
	groups := groupReferenceFiles(files)
	dirs := sortedReferencePackageDirs(groups)

	var packages []referencePackage
	for _, dir := range dirs {
		fileSet := token.NewFileSet()

		var parsedFiles []*ast.File
		for _, path := range groups[dir] {
			file, err := parser.ParseFile(fileSet, path, nil, 0)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}

			parsedFiles = append(parsedFiles, file)
		}

		if len(parsedFiles) == 0 {
			continue
		}

		info := types.Info{
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
		}

		importPath := referenceImportPath(root, modulePath, dir)
		config := types.Config{
			Importer: importer.Default(),
			Error:    func(error) {},
		}
		_, _ = config.Check(importPath, fileSet, parsedFiles, &info)

		packagePath, err := referencePackagePathForDir(root, dir)
		if err != nil {
			return nil, err
		}

		packages = append(packages, referencePackage{
			Package:    packagePath,
			ImportPath: importPath,
			Name:       parsedFiles[0].Name.Name,
			FileSet:    fileSet,
			Files:      parsedFiles,
			Info:       info,
		})
	}

	return packages, nil
}

func findTypecheckedReferenceReport(root string, target referenceTarget, options ReferenceOptions) (ReferenceReport, bool) {
	repo, err := loadSemanticReferenceRepository(root, semantics.LoadOptions{
		BuildTags: options.BuildTags,
	})
	if err != nil {
		return ReferenceReport{
			Target:       target.String(),
			AnalysisMode: ReferenceAnalysisModeASTFallback,
			Warnings:     []string{fmt.Sprintf("typechecked reference analysis unavailable: %v", err)},
		}, false
	}

	packages := semanticReferencePackages(repo)
	if len(packages) == 0 {
		return ReferenceReport{
			Target:       target.String(),
			AnalysisMode: ReferenceAnalysisModeASTFallback,
			Warnings:     append([]string{"typechecked reference analysis unavailable: no typechecked packages loaded"}, repo.Warnings...),
		}, false
	}

	targetObjects := semanticReferenceTargetObjects(packages, target)
	var refs []Reference
	for _, pkg := range packages {
		refs = append(refs, findTypecheckedReferencesInPackage(root, pkg, target, targetObjects)...)
	}

	refs = filterReferences(refs, options)
	sortReferences(refs)

	return ReferenceReport{
		Target:       target.String(),
		References:   nonNilReferences(refs),
		AnalysisMode: ReferenceAnalysisModeTypechecked,
		Warnings:     nonNilStrings(repo.Warnings),
	}, true
}

func semanticReferencePackages(repo semantics.Repository) []referencePackage {
	var packages []referencePackage
	for _, pkg := range repo.Packages {
		if pkg.FileSet == nil || pkg.TypesInfo == nil || len(pkg.Files) == 0 {
			continue
		}

		packages = append(packages, referencePackage{
			Package:    pkg.PackagePath,
			ImportPath: pkg.ImportPath,
			Name:       pkg.Name,
			FileSet:    pkg.FileSet,
			Files:      pkg.Files,
			Info:       *pkg.TypesInfo,
		})
	}

	return packages
}

func semanticReferenceTargetObjects(packages []referencePackage, target referenceTarget) map[types.Object]struct{} {
	objects := make(map[types.Object]struct{})
	for _, pkg := range packages {
		if target.Package != "" && pkg.Package != target.Package {
			continue
		}

		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				addReferenceTargetObjects(objects, pkg.Info, decl, target)
			}
		}
	}

	return objects
}

func findTypecheckedReferencesInPackage(root string, pkg referencePackage, target referenceTarget, targetObjects map[types.Object]struct{}) []Reference {
	seen := make(map[token.Pos]struct{})

	var refs []Reference
	addReference := func(pos token.Pos, kind ReferenceKind) {
		if !pos.IsValid() {
			return
		}

		if _, ok := seen[pos]; ok {
			return
		}

		seen[pos] = struct{}{}
		position := pkg.FileSet.Position(pos)
		refs = append(refs, Reference{
			Name: target.String(),
			Kind: kind,
			Position: positionRelativeToRoot(root, Position{
				File: position.Filename,
				Line: position.Line,
			}),
		})
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

			switch node := node.(type) {
			case *ast.Ident:
				object, definition := referenceIdentObject(pkg.Info, node)
				if referenceObjectMatchesTarget(object, targetObjects) {
					addReference(node.Pos(), referenceKindForIdent(object, definition, parent, node))
				}
			case *ast.SelectorExpr:
				object := pkg.Info.Uses[node.Sel]
				selection := pkg.Info.Selections[node]
				if referenceObjectMatchesTarget(object, targetObjects) || referenceSelectionMatchesTarget(selection, targetObjects) {
					addReference(node.Sel.Pos(), referenceKindForTypecheckedSelector(object, selection, parent, node))
				}
			}

			return true
		})
	}

	return refs
}

func referenceKindForTypecheckedSelector(object types.Object, selection *types.Selection, parent ast.Node, selector *ast.SelectorExpr) ReferenceKind {
	if selection != nil && selection.Kind() == types.FieldVal {
		return ReferenceKindFieldAccess
	}

	if _, ok := object.(*types.TypeName); ok {
		return ReferenceKindTypeUsage
	}

	return referenceKindForSelector(selection, parent, selector)
}

func splitPackageQualifiedTarget(root string, value string) (string, string, bool, error) {
	value = strings.TrimSpace(filepath.ToSlash(value))

	lastSlash := strings.LastIndex(value, "/")
	if lastSlash < 0 {
		return "", value, false, nil
	}

	firstDotAfterSlash := strings.Index(value[lastSlash+1:], ".")
	if firstDotAfterSlash < 0 {
		return "", value, false, nil
	}

	separator := lastSlash + 1 + firstDotAfterSlash
	packagePath, err := normalizeReferencePackagePath(root, value[:separator])
	if err != nil {
		return "", "", false, err
	}

	return packagePath, value[separator+1:], true, nil
}

func normalizeReferencePackagePath(root string, packagePath string) (string, error) {
	value := strings.TrimSpace(filepath.ToSlash(packagePath))
	if value == "" {
		return "", fmt.Errorf("package path is empty")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("absolute package paths are not supported: %s", packagePath)
	}

	modulePath := readModulePath(root)
	if modulePath != "" {
		if value == modulePath {
			return ".", nil
		}
		if strings.HasPrefix(value, modulePath+"/") {
			value = strings.TrimPrefix(value, modulePath+"/")
		} else if !strings.HasPrefix(value, "./") && strings.Contains(value, ".") {
			return "", fmt.Errorf("non-local package-qualified reference targets are not supported: %s", packagePath)
		}
	} else if !strings.HasPrefix(value, "./") && strings.Contains(value, ".") {
		return "", fmt.Errorf("module path is required for package-qualified reference target: %s", packagePath)
	}

	value = strings.TrimPrefix(value, "./")
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", fmt.Errorf("package path must not contain '..': %s", packagePath)
		}
	}

	cleaned := path.Clean(value)
	if cleaned == "." {
		return ".", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("package path escapes repository: %s", packagePath)
	}

	return "./" + cleaned, nil
}

func referencePackagePathForDir(root string, dir string) (string, error) {
	relativePath, err := filepath.Rel(root, dir)
	if err != nil {
		return "", err
	}

	relativePath = filepath.ToSlash(relativePath)
	if relativePath == "." {
		return ".", nil
	}

	return "./" + relativePath, nil
}

func readModulePath(root string) string {
	contents, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1]
		}
	}

	return ""
}

func groupReferenceFiles(files []string) map[string][]string {
	groups := make(map[string][]string)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		dir := filepath.Dir(file)
		groups[dir] = append(groups[dir], file)
	}

	for _, group := range groups {
		sort.Strings(group)
	}

	return groups
}

func sortedReferencePackageDirs(groups map[string][]string) []string {
	var dirs []string
	for dir := range groups {
		dirs = append(dirs, dir)
	}

	sort.Strings(dirs)

	return dirs
}

func referenceImportPath(root string, modulePath string, dir string) string {
	relativePath, err := filepath.Rel(root, dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}

	if modulePath == "" {
		return filepath.ToSlash(relativePath)
	}

	if relativePath == "." {
		return modulePath
	}

	return modulePath + "/" + filepath.ToSlash(relativePath)
}

func referenceTargetPackageImports(packages []referencePackage, target referenceTarget) map[string]struct{} {
	imports := make(map[string]struct{})
	if target.Receiver != "" {
		return imports
	}

	for _, pkg := range packages {
		if target.Package != "" && pkg.Package != target.Package {
			continue
		}
		if packageDefinesReferenceTarget(pkg, target) {
			imports[pkg.ImportPath] = struct{}{}
		}
	}

	return imports
}

func packageDefinesReferenceTarget(pkg referencePackage, target referenceTarget) bool {
	if target.Package != "" && pkg.Package != target.Package {
		return false
	}

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if declDefinesReferenceTarget(decl, target) {
				return true
			}
		}
	}

	return false
}

func referencePackageNames(packages []referencePackage) map[string]string {
	names := make(map[string]string)
	for _, pkg := range packages {
		names[pkg.ImportPath] = pkg.Name
	}

	return names
}

func findReferencesInPackage(
	root string,
	pkg referencePackage,
	target referenceTarget,
	targetPackages map[string]struct{},
	packageNames map[string]string,
) []Reference {
	targetObjects := referenceTargetObjects(pkg, target)
	seen := make(map[token.Pos]struct{})

	var refs []Reference
	addReference := func(pos token.Pos, kind ReferenceKind) {
		if !pos.IsValid() {
			return
		}

		if _, ok := seen[pos]; ok {
			return
		}

		seen[pos] = struct{}{}
		position := pkg.FileSet.Position(pos)
		refs = append(refs, Reference{
			Name: target.String(),
			Kind: kind,
			Position: positionRelativeToRoot(root, Position{
				File: position.Filename,
				Line: position.Line,
			}),
		})
	}

	for _, file := range pkg.Files {
		imports := referenceFileImports(file, packageNames)
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

			switch node := node.(type) {
			case *ast.Ident:
				object, definition := referenceIdentObject(pkg.Info, node)
				if referenceObjectMatchesTarget(object, targetObjects) {
					addReference(node.Pos(), referenceKindForIdent(object, definition, parent, node))
				}
			case *ast.SelectorExpr:
				selection := pkg.Info.Selections[node]
				if referenceSelectionMatchesTarget(selection, targetObjects) {
					addReference(node.Sel.Pos(), referenceKindForSelector(selection, parent, node))
				}

				if referenceSelectorMatchesImportedTarget(pkg.Info, node, target, targetPackages, imports) {
					addReference(node.Sel.Pos(), referenceKindForImportedSelector(parent, node))
				}
			}

			return true
		})
	}

	return refs
}

func referenceTargetObjects(pkg referencePackage, target referenceTarget) map[types.Object]struct{} {
	objects := make(map[types.Object]struct{})
	if target.Package != "" && pkg.Package != target.Package {
		return objects
	}

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			addReferenceTargetObjects(objects, pkg.Info, decl, target)
		}
	}

	return objects
}

func addReferenceTargetObjects(objects map[types.Object]struct{}, info types.Info, decl ast.Decl, target referenceTarget) {
	funcDecl, ok := decl.(*ast.FuncDecl)
	if ok {
		if funcDeclDefinesReferenceTarget(funcDecl, target) {
			addReferenceTargetObject(objects, info.Defs[funcDecl.Name])
		}

		return
	}

	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	for _, spec := range genDecl.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			if target.Receiver == "" && spec.Name.Name == target.Name {
				addReferenceTargetObject(objects, info.Defs[spec.Name])
			}
			addStructFieldReferenceTargetObjects(objects, info, spec, target)
		case *ast.ValueSpec:
			if target.Receiver != "" {
				continue
			}

			for _, name := range spec.Names {
				if name.Name == target.Name {
					addReferenceTargetObject(objects, info.Defs[name])
				}
			}
		}
	}
}

func addStructFieldReferenceTargetObjects(objects map[types.Object]struct{}, info types.Info, spec *ast.TypeSpec, target referenceTarget) {
	if target.Receiver == "" || spec.Name.Name != target.Receiver {
		return
	}

	structType, ok := spec.Type.(*ast.StructType)
	if !ok {
		return
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == target.Name {
				addReferenceTargetObject(objects, info.Defs[name])
			}
		}
	}
}

func addReferenceTargetObject(objects map[types.Object]struct{}, object types.Object) {
	if object == nil {
		return
	}

	objects[object] = struct{}{}
}

func declDefinesReferenceTarget(decl ast.Decl, target referenceTarget) bool {
	funcDecl, ok := decl.(*ast.FuncDecl)
	if ok {
		if funcDecl.Recv != nil {
			return false
		}

		return funcDeclDefinesReferenceTarget(funcDecl, target)
	}

	genDecl, ok := decl.(*ast.GenDecl)
	if !ok || target.Receiver != "" {
		return false
	}

	for _, spec := range genDecl.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			if spec.Name.Name == target.Name {
				return true
			}
		case *ast.ValueSpec:
			for _, name := range spec.Names {
				if name.Name == target.Name {
					return true
				}
			}
		}
	}

	return false
}

func funcDeclDefinesReferenceTarget(funcDecl *ast.FuncDecl, target referenceTarget) bool {
	if funcDecl.Name.Name != target.Name {
		return false
	}

	receiver := receiverTypeName(funcDecl)
	if target.Receiver == "" {
		return true
	}

	return receiver == target.Receiver
}

func referenceIdentObject(info types.Info, ident *ast.Ident) (types.Object, bool) {
	if object := info.Defs[ident]; object != nil {
		return object, true
	}

	return info.Uses[ident], false
}

func referenceObjectMatchesTarget(object types.Object, targetObjects map[types.Object]struct{}) bool {
	if object == nil {
		return false
	}

	_, ok := targetObjects[object]
	return ok
}

func referenceSelectionMatchesTarget(selection *types.Selection, targetObjects map[types.Object]struct{}) bool {
	if selection == nil {
		return false
	}

	return referenceObjectMatchesTarget(selection.Obj(), targetObjects)
}

func referenceKindForIdent(object types.Object, definition bool, parent ast.Node, ident *ast.Ident) ReferenceKind {
	if definition {
		return ReferenceKindDefinition
	}

	if _, ok := object.(*types.TypeName); ok {
		return ReferenceKindTypeUsage
	}

	if referenceNodeIsCall(parent, ident) {
		return ReferenceKindCall
	}

	return ReferenceKindUsage
}

func referenceKindForSelector(selection *types.Selection, parent ast.Node, selector *ast.SelectorExpr) ReferenceKind {
	if selection != nil && selection.Kind() == types.FieldVal {
		return ReferenceKindFieldAccess
	}

	if referenceNodeIsTypeUsage(parent, selector) {
		return ReferenceKindTypeUsage
	}

	if referenceNodeIsCall(parent, selector) {
		return ReferenceKindCall
	}

	return ReferenceKindUsage
}

func referenceKindForImportedSelector(parent ast.Node, selector *ast.SelectorExpr) ReferenceKind {
	if referenceNodeIsTypeUsage(parent, selector) {
		return ReferenceKindTypeUsage
	}

	if referenceNodeIsCall(parent, selector) {
		return ReferenceKindCall
	}

	return ReferenceKindUsage
}

func referenceNodeIsCall(parent ast.Node, expr ast.Expr) bool {
	call, ok := parent.(*ast.CallExpr)
	return ok && call.Fun == expr
}

func referenceNodeIsTypeUsage(parent ast.Node, expr ast.Expr) bool {
	switch parent := parent.(type) {
	case *ast.ArrayType:
		return parent.Elt == expr
	case *ast.ChanType:
		return parent.Value == expr
	case *ast.CompositeLit:
		return parent.Type == expr
	case *ast.Field:
		return parent.Type == expr
	case *ast.MapType:
		return parent.Key == expr || parent.Value == expr
	case *ast.TypeAssertExpr:
		return parent.Type == expr
	case *ast.TypeSpec:
		return parent.Type == expr
	case *ast.ValueSpec:
		return parent.Type == expr
	default:
		return false
	}
}

func referenceFileImports(file *ast.File, packageNames map[string]string) map[string]string {
	imports := make(map[string]string)
	for _, importSpec := range file.Imports {
		path, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}

		name := packageNames[path]
		if importSpec.Name != nil {
			name = importSpec.Name.Name
		}

		if name == "" || name == "." || name == "_" {
			continue
		}

		imports[name] = path
	}

	return imports
}

func referenceSelectorMatchesImportedTarget(
	info types.Info,
	selector *ast.SelectorExpr,
	target referenceTarget,
	targetPackages map[string]struct{},
	imports map[string]string,
) bool {
	if target.Receiver != "" || selector.Sel.Name != target.Name {
		return false
	}

	packageName, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	importPath, ok := imports[packageName.Name]
	if !ok {
		return false
	}

	if _, ok := targetPackages[importPath]; !ok {
		return false
	}

	object := info.Uses[packageName]
	if object == nil {
		return true
	}

	importedPackage, ok := object.(*types.PkgName)
	if !ok {
		return false
	}

	return importedPackage.Imported().Path() == importPath
}

func filterReferences(refs []Reference, options ReferenceOptions) []Reference {
	if options.Kind == "" {
		return refs
	}

	filtered := refs[:0]
	for _, ref := range refs {
		if ref.Kind == options.Kind {
			filtered = append(filtered, ref)
		}
	}

	return filtered
}

func sortReferences(refs []Reference) {
	sort.Slice(refs, func(i int, j int) bool {
		if refs[i].Position.File != refs[j].Position.File {
			return refs[i].Position.File < refs[j].Position.File
		}

		if refs[i].Position.Line != refs[j].Position.Line {
			return refs[i].Position.Line < refs[j].Position.Line
		}

		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}

		return refs[i].Name < refs[j].Name
	})
}

func nonNilReferences(refs []Reference) []Reference {
	if refs == nil {
		return []Reference{}
	}

	return refs
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}
