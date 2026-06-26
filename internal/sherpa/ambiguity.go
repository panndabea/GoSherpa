package sherpa

import (
	"fmt"
	"sort"
	"strings"
)

type TargetCandidate struct {
	Package  string
	Symbol   string
	Position Position
	Example  string
}

type AmbiguousTargetError struct {
	Kind       string
	Target     string
	Candidates []TargetCandidate
}

func NewAmbiguousTargetError(kind string, target string, candidates []TargetCandidate) *AmbiguousTargetError {
	copied := append([]TargetCandidate(nil), candidates...)
	sortTargetCandidates(copied)

	return &AmbiguousTargetError{
		Kind:       strings.TrimSpace(kind),
		Target:     strings.TrimSpace(target),
		Candidates: copied,
	}
}

func (err *AmbiguousTargetError) Error() string {
	if err == nil {
		return ""
	}

	kind := err.Kind
	if kind == "" {
		kind = "symbol"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "ambiguous %s target: %s", kind, err.Target)

	if len(err.Candidates) == 0 {
		return builder.String()
	}

	builder.WriteString("\n")
	builder.WriteString("candidates:\n")
	for _, candidate := range err.Candidates {
		fmt.Fprintf(
			&builder,
			"  - package %s, file %s, target %s\n",
			targetCandidatePackage(candidate.Package),
			targetCandidatePosition(candidate.Position),
			targetCandidateExample(candidate),
		)
	}

	examples := targetCandidateExamples(err.Candidates)
	if len(examples) > 0 {
		builder.WriteString("use a package-qualified target, for example:\n")
		for _, example := range examples {
			fmt.Fprintf(&builder, "  %s\n", example)
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

func FormatPackageQualifiedTarget(packagePath string, symbol string, modulePath string) string {
	packagePath = strings.TrimSpace(packagePath)
	symbol = strings.TrimSpace(symbol)
	modulePath = strings.TrimSpace(modulePath)
	if symbol == "" {
		return ""
	}

	if packagePath == "" {
		return symbol
	}
	if packagePath == "." {
		if modulePath != "" {
			return modulePath + "." + symbol
		}

		return symbol
	}

	return packagePath + "." + symbol
}

func sortTargetCandidates(candidates []TargetCandidate) {
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].Package != candidates[j].Package {
			return candidates[i].Package < candidates[j].Package
		}
		if candidates[i].Symbol != candidates[j].Symbol {
			return candidates[i].Symbol < candidates[j].Symbol
		}
		if candidates[i].Position.File != candidates[j].Position.File {
			return candidates[i].Position.File < candidates[j].Position.File
		}
		if candidates[i].Position.Line != candidates[j].Position.Line {
			return candidates[i].Position.Line < candidates[j].Position.Line
		}

		return candidates[i].Example < candidates[j].Example
	})
}

func targetCandidatePackage(packagePath string) string {
	packagePath = strings.TrimSpace(packagePath)
	if packagePath == "" {
		return "(unknown)"
	}

	return packagePath
}

func targetCandidatePosition(position Position) string {
	file := strings.TrimSpace(position.File)
	if file == "" {
		return "(unknown)"
	}
	if position.Line <= 0 {
		return file
	}

	return fmt.Sprintf("%s:%d", file, position.Line)
}

func targetCandidateExample(candidate TargetCandidate) string {
	example := strings.TrimSpace(candidate.Example)
	if example != "" {
		return example
	}

	return FormatPackageQualifiedTarget(candidate.Package, candidate.Symbol, "")
}

func targetCandidateExamples(candidates []TargetCandidate) []string {
	seen := make(map[string]struct{})
	var examples []string

	for _, candidate := range candidates {
		example := targetCandidateExample(candidate)
		if example == "" || example == candidate.Symbol {
			continue
		}
		if _, ok := seen[example]; ok {
			continue
		}

		seen[example] = struct{}{}
		examples = append(examples, example)
	}

	return examples
}
