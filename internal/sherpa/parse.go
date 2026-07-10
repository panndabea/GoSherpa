package sherpa

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strconv"
	"strings"
)

func ParseFile(path string) ([]Symbol, error) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok {
			position := symbolPosition(fileSet, funcDecl.Pos())
			receiver := receiverName(funcDecl)

			if funcDecl.Recv == nil {
				symbols = append(symbols, Symbol{
					Name:          funcDecl.Name.Name,
					Kind:          SymbolKindFunction,
					PackageName:   file.Name.Name,
					Signature:     functionSignature(funcDecl),
					Documentation: commentText(funcDecl.Doc),
					Position:      position,
					Range:         symbolRange(fileSet, funcDecl.Pos(), funcDecl.End()),
				})

				continue
			}

			symbols = append(symbols, Symbol{
				Name:          funcDecl.Name.Name,
				Kind:          SymbolKindMethod,
				PackageName:   file.Name.Name,
				Signature:     functionSignature(funcDecl),
				Documentation: commentText(funcDecl.Doc),
				Position:      position,
				Range:         symbolRange(fileSet, funcDecl.Pos(), funcDecl.End()),
				Receiver:      receiver,
				ReceiverType:  receiverType(funcDecl),
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

			if kind == "" && typeSpec.Assign.IsValid() {
				kind = SymbolKindAlias
			}

			if kind == "" {
				continue
			}

			position := symbolPosition(fileSet, typeSpec.Pos())

			symbol := Symbol{
				Name:          typeSpec.Name.Name,
				Kind:          kind,
				PackageName:   file.Name.Name,
				Documentation: typeDocumentation(genDecl, typeSpec),
				Position:      position,
				Range:         symbolRange(fileSet, typeSpec.Pos(), typeSpec.End()),
			}
			if kind == SymbolKindAlias {
				symbol.Signature = typeAliasSignature(typeSpec)
			}
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				symbol.Fields = structFields(structType)
			}
			if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				symbol.Methods = interfaceMethods(interfaceType)
			}

			symbols = append(symbols, symbol)
		}
	}

	return symbols, nil
}

func typeAliasSignature(typeSpec *ast.TypeSpec) string {
	if typeSpec == nil {
		return ""
	}

	return "type " + typeSpec.Name.Name + " = " + nodeString(typeSpec.Type)
}

func receiverName(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return ""
	}

	field := funcDecl.Recv.List[0]

	return receiverBaseName(field.Type)
}

func receiverType(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return ""
	}

	return nodeString(funcDecl.Recv.List[0].Type)
}

func functionSignature(funcDecl *ast.FuncDecl) string {
	functionType := nodeString(funcDecl.Type)
	if strings.HasPrefix(functionType, "func") {
		functionType = strings.TrimPrefix(functionType, "func")
	}

	if funcDecl.Recv == nil {
		return "func " + funcDecl.Name.Name + functionType
	}

	return "func (" + fieldListSignature(funcDecl.Recv) + ") " + funcDecl.Name.Name + functionType
}

func fieldListSignature(fields *ast.FieldList) string {
	if fields == nil {
		return ""
	}

	var parts []string
	for _, field := range fields.List {
		fieldType := nodeString(field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, fieldType)
			continue
		}

		var names []string
		for _, name := range field.Names {
			names = append(names, name.Name)
		}

		parts = append(parts, strings.Join(names, ", ")+" "+fieldType)
	}

	return strings.Join(parts, ", ")
}

func structFields(structType *ast.StructType) []SymbolField {
	if structType == nil || structType.Fields == nil {
		return nil
	}

	var fields []SymbolField
	for _, field := range structType.Fields.List {
		fieldType := nodeString(field.Type)
		tag := fieldTag(field)
		if len(field.Names) == 0 {
			fields = append(fields, SymbolField{
				Name:     embeddedFieldName(field.Type),
				Type:     fieldType,
				Tag:      tag,
				Embedded: true,
			})
			continue
		}

		for _, name := range field.Names {
			fields = append(fields, SymbolField{
				Name: name.Name,
				Type: fieldType,
				Tag:  tag,
			})
		}
	}

	return fields
}

func interfaceMethods(interfaceType *ast.InterfaceType) []SymbolMethod {
	if interfaceType == nil || interfaceType.Methods == nil {
		return nil
	}

	var methods []SymbolMethod
	for _, field := range interfaceType.Methods.List {
		if len(field.Names) == 0 {
			name := nodeString(field.Type)
			methods = append(methods, SymbolMethod{
				Name:      name,
				Signature: name,
				Embedded:  true,
			})
			continue
		}

		signature := interfaceMethodSignature(field.Type)
		for _, name := range field.Names {
			methods = append(methods, SymbolMethod{
				Name:      name.Name,
				Signature: name.Name + signature,
			})
		}
	}

	return methods
}

func interfaceMethodSignature(expr ast.Expr) string {
	funcType, ok := expr.(*ast.FuncType)
	if !ok {
		return " " + nodeString(expr)
	}

	signature := nodeString(funcType)
	if strings.HasPrefix(signature, "func") {
		return strings.TrimPrefix(signature, "func")
	}

	return " " + signature
}

func embeddedFieldName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(expr.X)
	case *ast.IndexExpr:
		return embeddedFieldName(expr.X)
	case *ast.IndexListExpr:
		return embeddedFieldName(expr.X)
	case *ast.ParenExpr:
		return embeddedFieldName(expr.X)
	default:
		return nodeString(expr)
	}
}

func fieldTag(field *ast.Field) string {
	if field == nil || field.Tag == nil {
		return ""
	}

	tag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return field.Tag.Value
	}

	return tag
}

func typeDocumentation(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) string {
	if typeSpec.Doc != nil {
		return commentText(typeSpec.Doc)
	}

	return commentText(genDecl.Doc)
}

func commentText(commentGroup *ast.CommentGroup) string {
	if commentGroup == nil {
		return ""
	}

	return strings.TrimSpace(commentGroup.Text())
}

func symbolPosition(fileSet *token.FileSet, pos token.Pos) Position {
	position := fileSet.Position(pos)
	return Position{
		File:   position.Filename,
		Line:   position.Line,
		Column: position.Column,
	}
}

func symbolRange(fileSet *token.FileSet, start token.Pos, end token.Pos) *SourceRange {
	return sourceRange(fileSet, start, end)
}

func sourceRange(fileSet *token.FileSet, start token.Pos, end token.Pos) *SourceRange {
	if fileSet == nil || !start.IsValid() || !end.IsValid() {
		return nil
	}

	return &SourceRange{
		Start: symbolPosition(fileSet, start),
		End:   symbolPosition(fileSet, end),
	}
}

func sourceRangeRelativeToRoot(root string, fileSet *token.FileSet, start token.Pos, end token.Pos) *SourceRange {
	rng := sourceRange(fileSet, start, end)
	if rng == nil {
		return nil
	}

	rng.Start = positionRelativeToRoot(root, rng.Start)
	rng.End = positionRelativeToRoot(root, rng.End)

	return rng
}

func nodeString(node any) string {
	var builder bytes.Buffer
	if err := printer.Fprint(&builder, token.NewFileSet(), node); err != nil {
		return ""
	}

	return builder.String()
}
