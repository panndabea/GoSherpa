package sherpa

import (
	"go/ast"
	"go/parser"
	"go/token"
)

func ParseFile(path string) ([]Symbol, error) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok {
			pos := fileSet.Position(funcDecl.Pos())

			if funcDecl.Recv == nil {
				symbols = append(symbols, Symbol{
					Name: funcDecl.Name.Name,
					Kind: SymbolKindFunction,
					Position: Position{
						File: pos.Filename,
						Line: pos.Line,
					},
				})

				continue
			}

			receiver := receiverName(funcDecl)

			symbols = append(symbols, Symbol{
				Name:     funcDecl.Name.Name,
				Kind:     SymbolKindMethod,
				Receiver: receiver,
				Position: Position{
					File: pos.Filename,
					Line: pos.Line,
				},
			})

			continue
		}

		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			kind := SymbolKind("")

			_, ok = typeSpec.Type.(*ast.StructType)
			if ok {
				kind = SymbolKindStruct
			}

			_, ok = typeSpec.Type.(*ast.InterfaceType)
			if ok {
				kind = SymbolKindInterface
			}

			if kind == "" {
				continue
			}

			pos := fileSet.Position(typeSpec.Pos())

			symbols = append(symbols, Symbol{
				Name: typeSpec.Name.Name,
				Kind: kind,
				Position: Position{
					File: pos.Filename,
					Line: pos.Line,
				},
			})
		}
	}

	return symbols, nil
}


func receiverName(funcDecl *ast.FuncDecl) string {
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