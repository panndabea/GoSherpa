package sherpa

func FindSymbol(symbols []Symbol, name string) *Symbol {
	for i := range symbols {
		if symbols[i].Name == name {
			return &symbols[i]
		}
	}

	return nil
}