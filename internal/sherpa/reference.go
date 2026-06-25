package sherpa

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Reference struct {
	Name     string   `json:"name"`
	Position Position `json:"position"`
}

type referenceTarget struct {
	Receiver string
	Name     string
}

type referencePackage struct {
	ImportPath string
	Name       string
	FileSet    *token.FileSet
	Files      []*ast.File
	Info       types.Info
}

func FindReferences(root string, name string) ([]Reference, error) {
	target, err := normalizeReferenceTarget(name)
	if err != nil {
		return nil, err
	}

	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	packages, err := parseReferencePackages(rootPath, files)
	if err != nil {
		return nil, err
	}

	targetPackages := referenceTargetPackageImports(packages, target)
	packageNames := referencePackageNames(packages)

	var refs []Reference
	for _, pkg := range packages {
		refs = append(refs, findReferencesInPackage(rootPath, pkg, target, targetPackages, packageNames)...)
	}

	sortReferences(refs)

	return refs, nil
}

func normalizeReferenceTarget(name string) (referenceTarget, error) {
	value := strings.TrimSpace(name)
	if value == "" {
		return referenceTarget{}, fmt.Errorf("reference target is empty")
	}

	if strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return referenceTarget{}, fmt.Errorf("package-qualified reference targets are not supported: %s", value)
	}

	segments := strings.Split(value, ".")
	if len(segments) != 1 && len(segments) != 2 {
		return referenceTarget{}, fmt.Errorf("invalid reference target: %s", value)
	}

	for _, segment := range segments {
		if segment == "" || !token.IsIdentifier(segment) {
			return referenceTarget{}, fmt.Errorf("invalid reference target: %s", value)
		}
	}

	target := referenceTarget{Name: segments[len(segments)-1]}
	if len(segments) == 2 {
		target.Receiver = segments[0]
	}

	return target, nil
}

func (target referenceTarget) String() string {
	if target.Receiver == "" {
		return target.Name
	}

	return target.Receiver + "." + target.Name
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

		packages = append(packages, referencePackage{
			ImportPath: importPath,
			Name:       parsedFiles[0].Name.Name,
			FileSet:    fileSet,
			Files:      parsedFiles,
			Info:       info,
		})
	}

	return packages, nil
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
		if packageDefinesReferenceTarget(pkg, target) {
			imports[pkg.ImportPath] = struct{}{}
		}
	}

	return imports
}

func packageDefinesReferenceTarget(pkg referencePackage, target referenceTarget) bool {
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
	addReference := func(pos token.Pos) {
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
			Position: positionRelativeToRoot(root, Position{
				File: position.Filename,
				Line: position.Line,
			}),
		})
	}

	for _, file := range pkg.Files {
		imports := referenceFileImports(file, packageNames)

		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.Ident:
				if referenceObjectMatchesTarget(referenceIdentObject(pkg.Info, node), targetObjects) {
					addReference(node.Pos())
				}
			case *ast.SelectorExpr:
				if referenceSelectionMatchesTarget(pkg.Info.Selections[node], targetObjects) {
					addReference(node.Sel.Pos())
				}

				if referenceSelectorMatchesImportedTarget(pkg.Info, node, target, targetPackages, imports) {
					addReference(node.Sel.Pos())
				}
			}

			return true
		})
	}

	return refs
}

func referenceTargetObjects(pkg referencePackage, target referenceTarget) map[types.Object]struct{} {
	objects := make(map[types.Object]struct{})
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

func referenceIdentObject(info types.Info, ident *ast.Ident) types.Object {
	if object := info.Defs[ident]; object != nil {
		return object
	}

	return info.Uses[ident]
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

func sortReferences(refs []Reference) {
	sort.Slice(refs, func(i int, j int) bool {
		if refs[i].Position.File != refs[j].Position.File {
			return refs[i].Position.File < refs[j].Position.File
		}

		if refs[i].Position.Line != refs[j].Position.Line {
			return refs[i].Position.Line < refs[j].Position.Line
		}

		return refs[i].Name < refs[j].Name
	})
}
