package sherpa

import (
	"fmt"
	"strings"
)

func FindSymbol(symbols []Symbol, name string) *Symbol {
	for i := range symbols {
		if symbols[i].Name == name {
			return &symbols[i]
		}
	}

	return nil
}

func FindSymbolTarget(root string, symbols []Symbol, target string) (Symbol, error) {
	parsed, err := normalizeReferenceTarget(root, target)
	if err != nil {
		return Symbol{}, err
	}

	var matches []Symbol
	for _, symbol := range symbols {
		if symbolMatchesReferenceTarget(symbol, parsed) {
			matches = append(matches, symbol)
		}
	}

	if len(matches) == 0 {
		return Symbol{}, fmt.Errorf("symbol not found: %s", strings.TrimSpace(target))
	}
	if len(matches) > 1 {
		return Symbol{}, NewAmbiguousTargetError("symbol", target, symbolTargetCandidates(root, matches))
	}

	return matches[0], nil
}

func symbolMatchesReferenceTarget(symbol Symbol, target referenceTarget) bool {
	if target.Package != "" && symbol.Package != target.Package {
		return false
	}

	if target.Receiver != "" {
		return symbol.Receiver == target.Receiver && symbol.Name == target.Name
	}
	if target.Package != "" {
		return symbol.Receiver == "" && symbol.Name == target.Name
	}

	return symbol.Name == target.Name
}

func symbolTargetCandidates(root string, symbols []Symbol) []TargetCandidate {
	modulePath := readModulePath(root)
	candidates := make([]TargetCandidate, 0, len(symbols))

	for _, symbol := range symbols {
		candidates = append(candidates, TargetCandidate{
			Package:  symbol.Package,
			Symbol:   symbol.DisplayName(),
			Position: symbol.Position,
			Example:  FormatPackageQualifiedTarget(symbol.Package, symbol.DisplayName(), modulePath),
		})
	}

	return candidates
}
