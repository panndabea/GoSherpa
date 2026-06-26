package sherpa

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type SymbolSearchResult struct {
	Symbol       Symbol   `json:"symbol"`
	Score        int      `json:"score"`
	MatchedTerms []string `json:"matchedTerms"`
}

func FindSymbol(symbols []Symbol, name string) *Symbol {
	for i := range symbols {
		if symbols[i].Name == name {
			return &symbols[i]
		}
	}

	return nil
}

func SearchSymbols(symbols []Symbol, terms []string) []SymbolSearchResult {
	queryTerms := normalizeSearchTerms(terms)
	if len(queryTerms) == 0 {
		return nil
	}

	var results []SymbolSearchResult
	for _, symbol := range symbols {
		score, matchedTerms := scoreSymbolSearchResult(symbol, queryTerms)
		if len(matchedTerms) != len(queryTerms) {
			continue
		}

		results = append(results, SymbolSearchResult{
			Symbol:       symbol,
			Score:        score,
			MatchedTerms: matchedTerms,
		})
	}

	sort.SliceStable(results, func(i int, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}

		return symbolSearchSortKey(results[i].Symbol) < symbolSearchSortKey(results[j].Symbol)
	})

	return results
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

func normalizeSearchTerms(terms []string) []string {
	seen := map[string]bool{}
	var normalized []string

	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || seen[term] {
			continue
		}

		seen[term] = true
		normalized = append(normalized, term)
	}

	return normalized
}

func scoreSymbolSearchResult(symbol Symbol, terms []string) (int, []string) {
	var score int
	var matchedTerms []string

	name := strings.ToLower(symbol.Name)
	displayName := strings.ToLower(symbol.DisplayName())
	qualifiedName := strings.ToLower(symbol.QualifiedName)
	packagePath := strings.ToLower(symbol.Package)
	packageName := strings.ToLower(symbol.PackageName)
	segments := searchNameSegments(symbol.DisplayName())

	for _, term := range terms {
		termScore := scoreSearchTerm(term, name, displayName, qualifiedName, packagePath, packageName, segments)
		if termScore == 0 {
			return 0, matchedTerms
		}

		score += termScore
		matchedTerms = append(matchedTerms, term)
	}

	if displayName == strings.Join(terms, "") || displayName == strings.Join(terms, ".") {
		score += 100
	}

	return score, matchedTerms
}

func scoreSearchTerm(term string, name string, displayName string, qualifiedName string, packagePath string, packageName string, segments []string) int {
	switch {
	case name == term || displayName == term:
		return 100
	case strings.HasPrefix(name, term) || strings.HasPrefix(displayName, term):
		return 80
	case containsSearchSegment(segments, term):
		return 70
	case strings.Contains(name, term) || strings.Contains(displayName, term):
		return 60
	case strings.Contains(packageName, term):
		return 35
	case strings.Contains(packagePath, term):
		return 30
	case strings.Contains(qualifiedName, term):
		return 20
	default:
		return 0
	}
}

func containsSearchSegment(segments []string, term string) bool {
	for _, segment := range segments {
		if segment == term || strings.HasPrefix(segment, term) {
			return true
		}
	}

	return false
}

func searchNameSegments(name string) []string {
	var segments []string
	var current strings.Builder

	flush := func() {
		if current.Len() == 0 {
			return
		}

		segments = append(segments, strings.ToLower(current.String()))
		current.Reset()
	}

	var previous rune
	for _, char := range name {
		if char == '.' || char == '_' || char == '-' || char == '/' {
			flush()
			previous = 0
			continue
		}

		if current.Len() > 0 && unicode.IsUpper(char) && previous != 0 && (unicode.IsLower(previous) || unicode.IsDigit(previous)) {
			flush()
		}

		current.WriteRune(char)
		previous = char
	}

	flush()
	return segments
}

func symbolSearchSortKey(symbol Symbol) string {
	return strings.Join([]string{
		strings.ToLower(symbol.DisplayName()),
		strings.ToLower(symbol.Package),
		symbol.Position.File,
		fmt.Sprintf("%08d", symbol.Position.Line),
		fmt.Sprintf("%08d", symbol.Position.Column),
	}, "\x00")
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
