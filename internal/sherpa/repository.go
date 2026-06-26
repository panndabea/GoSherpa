package sherpa

func ParseRepository(root string) ([]Symbol, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	files, err := FindGoFiles(rootPath)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol
	modulePath := readModulePath(rootPath)

	for _, file := range files {
		fileSymbols, err := ParseFile(file)
		if err != nil {
			return nil, err
		}

		packagePath, err := packagePathForFile(rootPath, file)
		if err != nil {
			return nil, err
		}

		for i := range fileSymbols {
			fileSymbols[i] = symbolRelativeToRoot(rootPath, packagePath, modulePath, fileSymbols[i])
		}

		symbols = append(symbols, fileSymbols...)
	}

	return symbols, nil
}

func symbolRelativeToRoot(root string, packagePath string, modulePath string, symbol Symbol) Symbol {
	symbol.Package = packagePath
	symbol.QualifiedName = FormatPackageQualifiedTarget(packagePath, symbol.DisplayName(), modulePath)
	symbol.Position = positionRelativeToRoot(root, symbol.Position)
	if symbol.Range != nil {
		symbol.Range.Start = positionRelativeToRoot(root, symbol.Range.Start)
		symbol.Range.End = positionRelativeToRoot(root, symbol.Range.End)
	}

	return symbol
}
