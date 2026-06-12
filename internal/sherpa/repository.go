package sherpa

func ParseRepository(root string) ([]Symbol, error) {
	files, err := FindGoFiles(root)
	if err != nil {
		return nil, err
	}

	var symbols []Symbol

	for _, file := range files {
		fileSymbols, err := ParseFile(file)
		if err != nil {
			return nil, err
		}

		symbols = append(symbols, fileSymbols...)
	}

	return symbols, nil
}