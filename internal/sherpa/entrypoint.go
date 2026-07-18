package sherpa

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
)

type EntryPointKind string

const (
	EntryPointKindMain           EntryPointKind = "main"
	EntryPointKindTest           EntryPointKind = "test"
	EntryPointKindStdlibHTTP     EntryPointKind = "stdlib-http-handler"
	EntryPointKindExported       EntryPointKind = "exported"
	EntryPointKindNoLocalCallers EntryPointKind = "no-local-callers"
)

type EntryPoint struct {
	Name            string         `json:"name"`
	Package         string         `json:"package,omitempty"`
	Kind            EntryPointKind `json:"kind"`
	Reason          string         `json:"reason,omitempty"`
	ReachableTarget string         `json:"reachableTarget,omitempty"`
	Certainty       CallCertainty  `json:"certainty,omitempty"`
	Position        Position       `json:"position"`
	Range           *SourceRange   `json:"range,omitempty"`
	Limitations     []string       `json:"limitations,omitempty"`
}

type EntryPointsResult struct {
	Target       string       `json:"target"`
	AnalysisMode string       `json:"analysisMode"`
	Warnings     []string     `json:"warnings"`
	EntryPoints  []EntryPoint `json:"entrypoints"`
}

type EntryPointSummaryOptions struct {
	Limit int
}

type EntryPointSummary struct {
	AnalysisMode string            `json:"analysisMode"`
	Confidence   string            `json:"confidence"`
	Counts       []EntryPointCount `json:"counts"`
	Examples     []EntryPoint      `json:"examples"`
	Limitations  []string          `json:"limitations"`
	Truncated    int               `json:"truncated,omitempty"`
}

type EntryPointCount struct {
	Kind      EntryPointKind `json:"kind"`
	Certainty CallCertainty  `json:"certainty"`
	Count     int            `json:"count"`
}

type entryPointGraphEvidence struct {
	reverseGraph    map[string][]callGraphEdge
	httpHandlerKeys map[string]struct{}
}

const (
	EntryPointSummaryConfidenceMedium = "medium"
	EntryPointSummaryConfidenceLow    = "low"

	defaultEntryPointSummaryLimit = 6
)

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

	functions, analysisMode, warnings, err := collectEntryPointFunctionInfos(rootPath, options)
	if err != nil {
		return EntryPointsResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	result, err := findEntryPointsInFunctionInfos(functions, normalizedTarget, analysisMode, warnings)
	if err != nil {
		return result, err
	}

	return result, nil
}

func SummarizeEntryPointsForTargets(root string, targets []string, options CallOptions, summaryOptions EntryPointSummaryOptions) (EntryPointSummary, []string, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return EntryPointSummary{}, nil, err
	}

	functions, analysisMode, warnings, err := collectEntryPointFunctionInfos(rootPath, options)
	if err != nil {
		summary := NormalizeEntryPointSummary(EntryPointSummary{
			AnalysisMode: analysisMode,
			Confidence:   EntryPointSummaryConfidenceLow,
			Limitations:  EntryPointSummaryLimitations(options.IncludeTests, analysisMode),
		})
		return summary, nonNilStrings(warnings), err
	}

	evidence := buildEntryPointGraphEvidence(functions)
	var records []EntryPoint
	for _, target := range uniqueSorted(nonNilStrings(targets)) {
		normalizedTarget, err := normalizeCallTarget(rootPath, target)
		if err != nil {
			warnings = append(warnings, "entrypoint summary unavailable for "+target+": "+err.Error())
			continue
		}

		result, err := findEntryPointsInFunctionInfosWithEvidence(functions, normalizedTarget, analysisMode, nil, evidence)
		if err != nil {
			if isEntryPointNonFunctionTargetError(err) {
				continue
			}
			warnings = append(warnings, "entrypoint summary unavailable for "+normalizedTarget.String()+": "+err.Error())
			continue
		}

		records = append(records, result.EntryPoints...)
		warnings = append(warnings, result.Warnings...)
	}

	summary := SummarizeEntryPointRecords(records, analysisMode, warnings, options.IncludeTests, summaryOptions)
	return summary, nonNilStrings(uniqueSorted(warnings)), nil
}

func collectEntryPointFunctionInfos(rootPath string, options CallOptions) ([]functionInfo, string, []string, error) {
	functions, analysisMode, warnings, err := collectCallFunctionInfos(rootPath, options)
	if err != nil {
		return functions, analysisMode, warnings, err
	}

	if !options.IncludeTests {
		return functions, analysisMode, warnings, nil
	}

	testFunctions, testWarnings, err := collectTestCallerFunctionInfos(rootPath, options)
	warnings = uniqueSorted(append(warnings, testWarnings...))
	if err != nil {
		return functions, analysisMode, warnings, err
	}

	functions = append(functions, testFunctions...)
	sortFunctionInfos(functions)

	return functions, analysisMode, warnings, nil
}

func findEntryPointsInFunctionInfos(functions []functionInfo, normalizedTarget callTarget, analysisMode string, warnings []string) (EntryPointsResult, error) {
	return findEntryPointsInFunctionInfosWithEvidence(functions, normalizedTarget, analysisMode, warnings, buildEntryPointGraphEvidence(functions))
}

func findEntryPointsInFunctionInfosWithEvidence(functions []functionInfo, normalizedTarget callTarget, analysisMode string, warnings []string, evidence entryPointGraphEvidence) (EntryPointsResult, error) {
	targetFunction, err := findFunctionInfo(functions, normalizedTarget)
	if err != nil {
		return EntryPointsResult{
			Target:       normalizedTarget.String(),
			AnalysisMode: analysisMode,
			Warnings:     nonNilStrings(warnings),
		}, err
	}

	targetKey := functionNode(targetFunction).Key
	reachable := reachableCallerKeys(evidence.reverseGraph, targetKey)
	entryPoints := collectEntryPointsFromReachableFunctions(functions, evidence.reverseGraph, reachable, evidence.httpHandlerKeys, targetKey, normalizedTarget.String())

	return EntryPointsResult{
		Target:       normalizedTarget.String(),
		AnalysisMode: analysisMode,
		Warnings:     nonNilStrings(warnings),
		EntryPoints:  NormalizeEntryPoints(entryPoints),
	}, nil
}

func buildEntryPointGraphEvidence(functions []functionInfo) entryPointGraphEvidence {
	graph := buildEntryPointCallGraph(functions)
	return entryPointGraphEvidence{
		reverseGraph:    reverseCallGraph(graph),
		httpHandlerKeys: stdlibHTTPHandlerEntryPointKeys(functions),
	}
}

func buildEntryPointCallGraph(functions []functionInfo) map[string][]callGraphEdge {
	graph := buildCallGraph(functions)
	catalog := newInterfaceDispatchCatalog(functions)

	for _, function := range functions {
		caller := functionNode(function)
		for _, possibleCall := range collectPossibleCallsFromFunctionWithFunctions(function, functions, catalog) {
			if possibleCall.Scope != CallScopeLocal {
				continue
			}
			if possibleCall.Reason != PossibleCallReasonGoroutine &&
				possibleCall.Reason != PossibleCallReasonFunctionLiteral &&
				possibleCall.Reason != PossibleCallReasonStdlibHTTPHandler {
				continue
			}

			target := callTarget{
				Package: possibleCall.calleePackage,
			}
			target.Receiver, target.Name = relationshipSplitDisplayName(possibleCall.Callee)
			if target.Name == "" {
				continue
			}

			match, ok := findMatchingFunctionInfo(functions, target)
			if !ok {
				continue
			}

			graph[caller.Key] = appendCallGraphEdgeIfMissing(graph[caller.Key], callGraphEdge{
				Caller:    caller,
				Callee:    functionNode(match),
				Certainty: CallCertaintyPossible,
				Reason:    possibleCall.Reason,
				Position:  possibleCall.Position,
				Range:     possibleCall.Range,
			})
		}
		sortCallGraphEdges(graph[caller.Key])
	}

	return graph
}

func stdlibHTTPHandlerEntryPointKeys(functions []functionInfo) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, function := range functions {
		for _, possibleCall := range collectStdlibHTTPHandlerPossibleCalls(function, functions) {
			if possibleCall.Scope != CallScopeLocal || possibleCall.calleePackage == "" || possibleCall.Callee == "" {
				continue
			}

			target := callTarget{Package: possibleCall.calleePackage}
			target.Receiver, target.Name = relationshipSplitDisplayName(possibleCall.Callee)
			if target.Name == "" {
				continue
			}

			match, ok := findMatchingFunctionInfo(functions, target)
			if !ok {
				continue
			}
			keys[functionNode(match).Key] = struct{}{}
		}
	}

	return keys
}

func findMatchingFunctionInfo(functions []functionInfo, target callTarget) (functionInfo, bool) {
	for _, function := range functions {
		if functionMatchesCallTarget(function, target) {
			return function, true
		}
	}

	return functionInfo{}, false
}

func appendCallGraphEdgeIfMissing(edges []callGraphEdge, edge callGraphEdge) []callGraphEdge {
	key := callGraphEdgeKey(edge)
	for _, existing := range edges {
		if callGraphEdgeKey(existing) == key {
			return edges
		}
	}

	return append(edges, edge)
}

func callGraphEdgeKey(edge callGraphEdge) string {
	return strings.Join([]string{
		edge.Caller.Key,
		edge.Callee.Key,
		edge.Position.File,
		strconv.Itoa(edge.Position.Line),
		strconv.Itoa(edge.Position.Column),
	}, "\x00")
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

func collectEntryPointsFromReachableFunctions(functions []functionInfo, reverseGraph map[string][]callGraphEdge, reachable map[string]struct{}, httpHandlerKeys map[string]struct{}, targetKey string, reachableTarget string) []EntryPoint {
	var entryPoints []EntryPoint
	for _, function := range functions {
		node := functionNode(function)
		if _, ok := reachable[node.Key]; !ok {
			continue
		}

		kind, ok := functionEntryPointKind(function, reverseGraph, node.Key, httpHandlerKeys)
		if !ok {
			continue
		}

		certainty, pathReason := entryPointPathEvidence(reverseGraph, targetKey, node.Key)
		certainty = entryPointCertainty(kind, certainty)
		reason := entryPointReason(kind, certainty, pathReason)
		entryPoints = append(entryPoints, EntryPoint{
			Name:            function.Target,
			Package:         function.Package,
			Kind:            kind,
			Reason:          reason,
			ReachableTarget: reachableTarget,
			Certainty:       certainty,
			Position:        function.Position,
			Range:           functionDeclarationRange(function),
			Limitations:     entryPointRecordLimitations(kind, certainty),
		})
	}

	sortEntryPoints(entryPoints)

	return entryPoints
}

func entryPointPathEvidence(reverseGraph map[string][]callGraphEdge, targetKey string, entryKey string) (CallCertainty, PossibleCallReason) {
	if entryKey == targetKey {
		return CallCertaintyDirect, ""
	}

	if entryPointPathExists(reverseGraph, targetKey, entryKey, true) {
		return CallCertaintyDirect, ""
	}

	reason, ok := entryPointPossiblePathReason(reverseGraph, targetKey, entryKey)
	if ok {
		return CallCertaintyPossible, reason
	}

	return CallCertaintyPossible, ""
}

func entryPointPathExists(reverseGraph map[string][]callGraphEdge, targetKey string, entryKey string, directOnly bool) bool {
	seen := map[string]struct{}{targetKey: {}}
	queue := []string{targetKey}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range reverseGraph[current] {
			if directOnly && edge.Certainty == CallCertaintyPossible {
				continue
			}
			if edge.Caller.Key == entryKey {
				return true
			}
			if _, ok := seen[edge.Caller.Key]; ok {
				continue
			}
			seen[edge.Caller.Key] = struct{}{}
			queue = append(queue, edge.Caller.Key)
		}
	}

	return false
}

func entryPointPossiblePathReason(reverseGraph map[string][]callGraphEdge, targetKey string, entryKey string) (PossibleCallReason, bool) {
	type state struct {
		key    string
		reason PossibleCallReason
	}
	seen := map[string]struct{}{targetKey + "\x00": {}}
	queue := []state{{key: targetKey}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, edge := range reverseGraph[current.key] {
			reason := current.reason
			if edge.Certainty == CallCertaintyPossible && reason == "" {
				reason = edge.Reason
			}
			if edge.Caller.Key == entryKey && reason != "" {
				return reason, true
			}
			stateKey := edge.Caller.Key + "\x00" + string(reason)
			if _, ok := seen[stateKey]; ok {
				continue
			}
			seen[stateKey] = struct{}{}
			queue = append(queue, state{key: edge.Caller.Key, reason: reason})
		}
	}

	return "", false
}

func entryPointCertainty(kind EntryPointKind, pathCertainty CallCertainty) CallCertainty {
	switch kind {
	case EntryPointKindStdlibHTTP, EntryPointKindExported, EntryPointKindNoLocalCallers:
		return CallCertaintyPossible
	}
	if pathCertainty == "" {
		return CallCertaintyDirect
	}

	return pathCertainty
}

func entryPointReason(kind EntryPointKind, certainty CallCertainty, pathReason PossibleCallReason) string {
	pathLabel := string(pathReason)
	switch kind {
	case EntryPointKindMain:
		if certainty == CallCertaintyPossible && pathLabel != "" {
			return "main.main reaches the target through possible " + pathLabel + " wiring."
		}
		return "main.main reaches the target through repository-local calls."
	case EntryPointKindTest:
		if certainty == CallCertaintyPossible && pathLabel != "" {
			return "A Go test entrypoint reaches the target through possible " + pathLabel + " wiring."
		}
		return "A Go test entrypoint reaches the target when --tests is used."
	case EntryPointKindStdlibHTTP:
		return "The target is registered as a statically visible net/http handler."
	case EntryPointKindExported:
		if pathLabel != "" {
			return "An exported function reaches the target through possible " + pathLabel + " wiring."
		}
		return "An exported function is treated as a public entrypoint."
	case EntryPointKindNoLocalCallers:
		if pathLabel != "" {
			return "A function with no repository-local callers reaches the target through possible " + pathLabel + " wiring."
		}
		return "A function with no repository-local callers may be invoked externally."
	default:
		if pathLabel != "" {
			return "Entrypoint reachability uses possible " + pathLabel + " wiring."
		}
		return "Entrypoint reaches the target through repository-local calls."
	}
}

func entryPointRecordLimitations(kind EntryPointKind, certainty CallCertainty) []string {
	limitations := []string{
		"Entrypoint reachability is repository-local and static; reflection, custom routers, dynamic function values, and dependency internals can be missed.",
	}
	if certainty == CallCertaintyPossible {
		limitations = append(limitations, "Possible entrypoint evidence is a planning signal, not proof that runtime traffic reaches the target.")
	}
	switch kind {
	case EntryPointKindStdlibHTTP:
		limitations = append(limitations, "Stdlib net/http handler detection covers statically visible Handle and HandleFunc registrations only.")
	case EntryPointKindExported, EntryPointKindNoLocalCallers:
		limitations = append(limitations, "Public and no-local-caller entrypoint kinds are external-invocation heuristics.")
	case EntryPointKindTest:
		limitations = append(limitations, "Test entrypoints are included only when --tests is requested.")
	}

	return limitations
}

func functionEntryPointKind(function functionInfo, reverseGraph map[string][]callGraphEdge, key string, httpHandlerKeys map[string]struct{}) (EntryPointKind, bool) {
	switch {
	case functionIsMain(function):
		return EntryPointKindMain, true
	case functionIsTestEntryPoint(function):
		return EntryPointKindTest, true
	case functionIsStdlibHTTPHandler(httpHandlerKeys, key):
		return EntryPointKindStdlibHTTP, true
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

func functionIsStdlibHTTPHandler(httpHandlerKeys map[string]struct{}, key string) bool {
	if len(httpHandlerKeys) == 0 {
		return false
	}

	_, ok := httpHandlerKeys[key]
	return ok
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
	case EntryPointKindStdlibHTTP:
		return 2
	case EntryPointKindExported:
		return 3
	case EntryPointKindNoLocalCallers:
		return 4
	default:
		return 5
	}
}

func SummarizeEntryPointRecords(entryPoints []EntryPoint, analysisMode string, warnings []string, includeTests bool, options EntryPointSummaryOptions) EntryPointSummary {
	entryPoints = NormalizeEntryPoints(entryPoints)
	limit := options.Limit
	if limit <= 0 {
		limit = defaultEntryPointSummaryLimit
	}

	countsByKey := make(map[string]EntryPointCount)
	for _, entryPoint := range entryPoints {
		certainty := entryPoint.Certainty
		if certainty == "" {
			certainty = CallCertaintyPossible
		}
		key := string(entryPoint.Kind) + "\x00" + string(certainty)
		count := countsByKey[key]
		count.Kind = entryPoint.Kind
		count.Certainty = certainty
		count.Count++
		countsByKey[key] = count
	}

	counts := make([]EntryPointCount, 0, len(countsByKey))
	for _, count := range countsByKey {
		counts = append(counts, count)
	}
	sortEntryPointCounts(counts)

	examples := append([]EntryPoint{}, entryPoints...)
	truncated := 0
	if len(examples) > limit {
		truncated = len(examples) - limit
		examples = examples[:limit]
	}

	confidence := EntryPointSummaryConfidenceMedium
	if len(warnings) > 0 || analysisMode == CallAnalysisModeASTFallback {
		confidence = EntryPointSummaryConfidenceLow
	}

	return NormalizeEntryPointSummary(EntryPointSummary{
		AnalysisMode: analysisMode,
		Confidence:   confidence,
		Counts:       counts,
		Examples:     examples,
		Limitations:  EntryPointSummaryLimitations(includeTests, analysisMode),
		Truncated:    truncated,
	})
}

func NormalizeEntryPointSummary(summary EntryPointSummary) EntryPointSummary {
	summary.AnalysisMode = strings.TrimSpace(summary.AnalysisMode)
	if summary.AnalysisMode == "" {
		summary.AnalysisMode = CallAnalysisModeASTFallback
	}
	summary.Confidence = strings.TrimSpace(summary.Confidence)
	if summary.Confidence == "" {
		summary.Confidence = EntryPointSummaryConfidenceMedium
	}
	summary.Counts = nonNilEntryPointCounts(summary.Counts)
	sortEntryPointCounts(summary.Counts)
	summary.Examples = NormalizeEntryPoints(summary.Examples)
	summary.Limitations = nonNilStrings(uniqueSorted(summary.Limitations))
	if summary.Truncated < 0 {
		summary.Truncated = 0
	}

	return summary
}

func NormalizeEntryPoints(entryPoints []EntryPoint) []EntryPoint {
	if entryPoints == nil {
		return []EntryPoint{}
	}

	result := append([]EntryPoint{}, entryPoints...)
	for i := range result {
		if result[i].Certainty == "" {
			result[i].Certainty = CallCertaintyDirect
		}
		result[i].Reason = strings.TrimSpace(result[i].Reason)
		result[i].ReachableTarget = strings.TrimSpace(result[i].ReachableTarget)
		result[i].Limitations = nonNilStrings(uniqueSorted(result[i].Limitations))
	}
	sortEntryPoints(result)

	return result
}

func isEntryPointNonFunctionTargetError(err error) bool {
	if err == nil {
		return false
	}

	return strings.HasPrefix(err.Error(), "function not found:")
}

func EntryPointSummaryLimitations(includeTests bool, analysisMode string) []string {
	limitations := []string{
		entryPointAnalysisLimitation(analysisMode),
		"Entrypoint summaries are bounded reachability evidence; direct caller evidence remains separate.",
		"Possible runtime wiring keeps certainty labels explicit and is not merged with direct calls.",
		"Framework-specific routers, custom routers, custom runtime wiring, reflection, reassigned or escaping function values, and dependency internals are not inferred.",
	}
	if !includeTests {
		limitations = append(limitations, "Test entrypoints are excluded unless --tests is requested.")
	}

	return limitations
}

func entryPointAnalysisLimitation(analysisMode string) string {
	switch strings.TrimSpace(analysisMode) {
	case CallAnalysisModeTypechecked:
		return "Entrypoint analysis used typechecked package loading where available, with static repository-local call graph evidence."
	case CallAnalysisModeASTFallback:
		return "Entrypoint analysis used AST fallback because typechecked package loading was unavailable."
	default:
		return "Entrypoint analysis used static repository-local call graph evidence."
	}
}

func nonNilEntryPointCounts(counts []EntryPointCount) []EntryPointCount {
	if counts == nil {
		return []EntryPointCount{}
	}

	return counts
}

func sortEntryPointCounts(counts []EntryPointCount) {
	sort.Slice(counts, func(i int, j int) bool {
		leftPriority := entryPointKindPriority(counts[i].Kind)
		rightPriority := entryPointKindPriority(counts[j].Kind)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if counts[i].Kind != counts[j].Kind {
			return counts[i].Kind < counts[j].Kind
		}
		if counts[i].Certainty != counts[j].Certainty {
			return entryPointCertaintyPriority(counts[i].Certainty) < entryPointCertaintyPriority(counts[j].Certainty)
		}

		return counts[i].Count > counts[j].Count
	})
}

func entryPointCertaintyPriority(certainty CallCertainty) int {
	switch certainty {
	case CallCertaintyDirect:
		return 0
	case CallCertaintyPossible:
		return 1
	default:
		return 2
	}
}
