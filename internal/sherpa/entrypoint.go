package sherpa

import (
	"go/ast"
	"sort"
	"strings"
)

type EntryPointKind string

const (
	EntryPointKindMain           EntryPointKind = "main"
	EntryPointKindTest           EntryPointKind = "test"
	EntryPointKindExported       EntryPointKind = "exported"
	EntryPointKindNoLocalCallers EntryPointKind = "no-local-callers"
)

type EntryPoint struct {
	Name     string         `json:"name"`
	Package  string         `json:"package,omitempty"`
	Kind     EntryPointKind `json:"kind"`
	Position Position       `json:"position"`
	Range    *SourceRange   `json:"range,omitempty"`
}

type EntryPointsResult struct {
	Target       string       `json:"target"`
	AnalysisMode string       `json:"analysisMode"`
	Warnings     []string     `json:"warnings"`
	EntryPoints  []EntryPoint `json:"entrypoints"`
}

func FindEntryPoints(root string, target string) (EntryPointsResult, error) {
	return FindEntryPointsWithOptions(root, target, CallOptions{})
}

func FindEntryPointsWithOptions(root string, target string, options CallOptions) (EntryPointsResult, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return EntryPointsResult{}, err
	}

	normalizedTarget, err := normalizeCallTarget(rootPath, target)
	if err != nil {
		return EntryPointsResult{}, err
	}

	functions, analysisMode, warnings, err := collectCallFunctionInfos(rootPath, options)
	if err != nil {
		return EntryPointsResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	targetFunction, err := findFunctionInfo(functions, normalizedTarget)
	if err != nil {
		return EntryPointsResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	graphFunctions := functions
	if options.IncludeTests {
		testFunctions, testWarnings, err := collectTestCallerFunctionInfos(rootPath, options)
		warnings = uniqueSorted(append(warnings, testWarnings...))
		if err != nil {
			return EntryPointsResult{
				Target:       normalizedTarget.String(),
				AnalysisMode: analysisMode,
				Warnings:     nonNilStrings(warnings),
			}, err
		}

		graphFunctions = append(graphFunctions, testFunctions...)
		sortFunctionInfos(graphFunctions)
	}

	graph := buildCallGraph(graphFunctions)
	reverseGraph := reverseCallGraph(graph)
	reachable := reachableCallerKeys(reverseGraph, functionNode(targetFunction).Key)
	entryPoints := collectEntryPointsFromReachableFunctions(graphFunctions, reverseGraph, reachable)

	return EntryPointsResult{
		Target:       normalizedTarget.String(),
		AnalysisMode: analysisMode,
		Warnings:     nonNilStrings(warnings),
		EntryPoints:  entryPoints,
	}, nil
}

func reachableCallerKeys(reverseGraph map[string][]callGraphEdge, targetKey string) map[string]struct{} {
	reachable := map[string]struct{}{
		targetKey: {},
	}
	queue := []string{targetKey}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range reverseGraph[current] {
			if _, ok := reachable[edge.Caller.Key]; ok {
				continue
			}

			reachable[edge.Caller.Key] = struct{}{}
			queue = append(queue, edge.Caller.Key)
		}
	}

	return reachable
}

func collectEntryPointsFromReachableFunctions(functions []functionInfo, reverseGraph map[string][]callGraphEdge, reachable map[string]struct{}) []EntryPoint {
	var entryPoints []EntryPoint
	for _, function := range functions {
		node := functionNode(function)
		if _, ok := reachable[node.Key]; !ok {
			continue
		}

		kind, ok := functionEntryPointKind(function, reverseGraph, node.Key)
		if !ok {
			continue
		}

		entryPoints = append(entryPoints, EntryPoint{
			Name:     function.Target,
			Package:  function.Package,
			Kind:     kind,
			Position: function.Position,
			Range:    functionDeclarationRange(function),
		})
	}

	sortEntryPoints(entryPoints)

	return entryPoints
}

func functionEntryPointKind(function functionInfo, reverseGraph map[string][]callGraphEdge, key string) (EntryPointKind, bool) {
	switch {
	case functionIsMain(function):
		return EntryPointKindMain, true
	case functionIsTestEntryPoint(function):
		return EntryPointKindTest, true
	case functionIsExported(function):
		return EntryPointKindExported, true
	case !functionHasLocalCallers(reverseGraph, key):
		return EntryPointKindNoLocalCallers, true
	default:
		return "", false
	}
}

func functionIsMain(function functionInfo) bool {
	return function.PackageName == "main" && function.Receiver == "" && function.Name == "main"
}

func functionIsTestEntryPoint(function functionInfo) bool {
	return strings.HasSuffix(function.Position.File, "_test.go") &&
		function.Receiver == "" &&
		isGoTestEntryPointName(function.Name)
}

func isGoTestEntryPointName(name string) bool {
	return strings.HasPrefix(name, "Test") ||
		strings.HasPrefix(name, "Benchmark") ||
		strings.HasPrefix(name, "Fuzz") ||
		strings.HasPrefix(name, "Example")
}

func functionIsExported(function functionInfo) bool {
	return function.Name != "" && ast.IsExported(function.Name)
}

func functionHasLocalCallers(reverseGraph map[string][]callGraphEdge, key string) bool {
	for _, edge := range reverseGraph[key] {
		if edge.Caller.Key != key {
			return true
		}
	}

	return false
}

func functionDeclarationRange(function functionInfo) *SourceRange {
	if function.Decl == nil || function.Decl.Name == nil {
		return nil
	}

	return sourceRangeRelativeToRoot(function.Root, function.FileSet, function.Decl.Pos(), function.Decl.Name.End())
}

func sortEntryPoints(entryPoints []EntryPoint) {
	sort.Slice(entryPoints, func(i int, j int) bool {
		leftPriority := entryPointKindPriority(entryPoints[i].Kind)
		rightPriority := entryPointKindPriority(entryPoints[j].Kind)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}

		if entryPoints[i].Package != entryPoints[j].Package {
			return entryPoints[i].Package < entryPoints[j].Package
		}

		if entryPoints[i].Name != entryPoints[j].Name {
			return entryPoints[i].Name < entryPoints[j].Name
		}

		if entryPoints[i].Position.File != entryPoints[j].Position.File {
			return entryPoints[i].Position.File < entryPoints[j].Position.File
		}

		return entryPoints[i].Position.Line < entryPoints[j].Position.Line
	})
}

func entryPointKindPriority(kind EntryPointKind) int {
	switch kind {
	case EntryPointKindMain:
		return 0
	case EntryPointKindTest:
		return 1
	case EntryPointKindExported:
		return 2
	case EntryPointKindNoLocalCallers:
		return 3
	default:
		return 4
	}
}
