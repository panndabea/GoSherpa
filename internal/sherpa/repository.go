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

	for _, file := range files {
		fileSymbols, err := ParseFile(file)
		if err != nil {
			return nil, err
		}

		for i := range fileSymbols {
			fileSymbols[i].Position = positionRelativeToRoot(rootPath, fileSymbols[i].Position)
		}

		symbols = append(symbols, fileSymbols...)
	}

	return symbols, nil
}
