package sherpa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type Reference struct {
	Name     string
	Position Position
}

func FindReferences(root string, name string) ([]Reference, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	var refs []Reference

	for _, file := range files {
		fileRefs, err := findReferencesInFile(file, name)
		if err != nil {
			return nil, err
		}

		for i := range fileRefs {
			fileRefs[i].Position = positionRelativeToRoot(rootPath, fileRefs[i].Position)
		}

		refs = append(refs, fileRefs...)
	}

	return refs, nil
}

func findReferencesInFile(path string, name string) ([]Reference, error) {
	if strings.HasSuffix(path, "_test.go") {
		return nil, nil
	}

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var refs []Reference

	ast.Inspect(file, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name != name {
			return true
		}

		pos := fileSet.Position(ident.Pos())

		refs = append(refs, Reference{
			Name: name,
			Position: Position{
				File: pos.Filename,
				Line: pos.Line,
			},
		})

		return true
	})

	return refs, nil
}
