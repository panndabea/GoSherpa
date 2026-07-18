package sherpa

import (
	"sort"
	"strings"
)

type TestPlan struct {
	Direct         []TestPlanItem `json:"direct"`
	Related        []TestPlanItem `json:"related"`
	Contracts      []TestPlanItem `json:"contracts"`
	CallerPackages []TestPlanItem `json:"callerPackages"`
	Fallback       []TestPlanItem `json:"fallback"`
	Confidence     string         `json:"confidence"`
	Limitations    []string       `json:"limitations"`
}

const TestPlanWholeRepositoryPackage = "./..."

const (
	TestPlanCategoryFast            = "fast"
	TestPlanCategoryFocused         = "focused"
	TestPlanCategoryContract        = "contract"
	TestPlanCategoryCallerPackage   = "caller-package"
	TestPlanCategoryIntegrationLike = "integration-like"
	TestPlanCategoryBroadFallback   = "broad-fallback"
)

const (
	TestPlanConfidenceMedium = "medium"
	TestPlanConfidenceLow    = "low"
)

type TestPlanItem struct {
	Command    string   `json:"command"`
	Reason     string   `json:"reason"`
	Category   string   `json:"category,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Package    string   `json:"package,omitempty"`
	Test       string   `json:"test,omitempty"`
	Tests      []string `json:"tests,omitempty"`
	Targets    []string `json:"targets,omitempty"`
}

type TestPlanRepositorySignals struct {
	GeneratedPackages []string
	SkippedPackages   []string
	Warnings          []string
}

func AnalyzeTestPlanRepositorySignals(root string, packages []string, warnings []string) TestPlanRepositorySignals {
	signals := TestPlanRepositorySignals{
		Warnings: uniqueSorted(warnings),
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return normalizeTestPlanRepositorySignals(signals)
	}

	layout, layoutWarnings := AnalyzeRepositoryLayout(root)
	packageSet := normalizedPlanPackageSet(packages)
	for _, generatedPackage := range layout.GeneratedPackages {
		if planPackageMatchesSet(generatedPackage.Package, packageSet) {
			signals.GeneratedPackages = append(signals.GeneratedPackages, planPackageDisplay(generatedPackage.Package))
		}
	}
	for _, skippedPackage := range layout.SkippedNestedModules {
		if planPackageMatchesSet(skippedPackage, packageSet) {
			signals.SkippedPackages = append(signals.SkippedPackages, planPackageDisplay(skippedPackage))
		}
	}
	if len(signals.SkippedPackages) > 0 {
		signals.Warnings = append(signals.Warnings, layoutWarnings...)
	}

	return normalizeTestPlanRepositorySignals(signals)
}

type TestPlanOptions struct {
	Target            string
	Kind              TestTargetKind
	TargetPackages    []string
	ContractPackages  []string
	CallerPackages    []string
	FallbackPackages  []string
	Targets           []string
	RepositorySignals TestPlanRepositorySignals
}

func PlanTests(tests []RelatedTest, options TestPlanOptions) TestPlan {
	target := strings.TrimSpace(options.Target)
	if target == "" {
		target = "target"
	}

	repositorySignals := normalizeTestPlanRepositorySignals(options.RepositorySignals)
	targetPackages := uniqueSorted(options.TargetPackages)
	contractPackages := uniqueSorted(options.ContractPackages)
	callerPackages := uniqueSorted(options.CallerPackages)
	fallbackPackages := uniqueSorted(options.FallbackPackages)
	defaultTargets := defaultTestPlanTargets(options, target)
	targetPackageSet := stringSet(targetPackages)
	contractPackageSet := stringSet(contractPackages)
	callerPackageSet := stringSet(callerPackages)

	evidence := make(testPlanEvidenceByPackage)
	directGroups := make(map[string][]string)
	relatedGroups := make(map[string][]string)
	contractGroups := make(map[string][]string)
	callerGroups := make(map[string][]string)
	directTargets := make(map[string][]string)
	relatedTargets := make(map[string][]string)
	contractTargets := make(map[string][]string)
	callerTargets := make(map[string][]string)

	for _, test := range tests {
		pkg := strings.TrimSpace(test.Package)
		if pkg == "" {
			continue
		}
		testTargets := uniqueSorted(test.Targets)
		reasonSet := stringSet(test.Reasons)
		evidence.Add(test)

		switch {
		case test.DirectReference:
			directGroups[pkg] = append(directGroups[pkg], test.Name)
			directTargets[pkg] = append(directTargets[pkg], testTargets...)
		case hasString(contractPackageSet, pkg) || hasString(reasonSet, RelatedTestReasonContract):
			contractGroups[pkg] = append(contractGroups[pkg], test.Name)
			contractTargets[pkg] = append(contractTargets[pkg], testTargets...)
		case hasString(callerPackageSet, pkg):
			callerGroups[pkg] = append(callerGroups[pkg], test.Name)
			callerTargets[pkg] = append(callerTargets[pkg], testTargets...)
		case len(targetPackageSet) == 0 || hasString(targetPackageSet, pkg):
			relatedGroups[pkg] = append(relatedGroups[pkg], test.Name)
			relatedTargets[pkg] = append(relatedTargets[pkg], testTargets...)
		default:
			callerGroups[pkg] = append(callerGroups[pkg], test.Name)
			callerTargets[pkg] = append(callerTargets[pkg], testTargets...)
		}
	}
	if len(directGroups) == 0 {
		fallbackPackages = uniqueSorted(append(fallbackPackages, targetPackages...))
	}

	plan := TestPlan{
		Direct: testPlanItemsFromGroups(directGroups, directTargets, defaultTargets, func(pkg string, names []string, targets []string) testPlanItemDetails {
			return testPlanItemDetails{
				Reason:     directTestPlanReason(options.Kind, target, pkg, names, targets),
				Category:   categoryForDirectTests(evidence[pkg]),
				Confidence: confidenceForPlanItem(pkg, false, repositorySignals),
			}
		}),
		Related: testPlanItemsFromGroups(relatedGroups, relatedTargets, defaultTargets, func(pkg string, names []string, targets []string) testPlanItemDetails {
			return testPlanItemDetails{
				Reason:     relatedTestPlanReason(options.Kind, target, pkg, names, targets),
				Category:   categoryForRelatedTests(evidence[pkg]),
				Confidence: confidenceForPlanItem(pkg, false, repositorySignals),
			}
		}),
		Contracts: testPlanItemsFromGroups(contractGroups, contractTargets, defaultTargets, func(pkg string, names []string, targets []string) testPlanItemDetails {
			return testPlanItemDetails{
				Reason:     contractTestPlanReason(pkg, names, targets),
				Category:   TestPlanCategoryContract,
				Confidence: confidenceForPlanItem(pkg, false, repositorySignals),
			}
		}),
		CallerPackages: testPlanItemsFromGroups(callerGroups, callerTargets, defaultTargets, func(pkg string, names []string, targets []string) testPlanItemDetails {
			return testPlanItemDetails{
				Reason:     callerPackageTestPlanReason(pkg, names, targets),
				Category:   TestPlanCategoryCallerPackage,
				Confidence: confidenceForPlanItem(pkg, false, repositorySignals),
			}
		}),
	}

	existingCommands := testPlanCommandSet(plan)
	for _, pkg := range fallbackPackages {
		command := testPackageCommand(pkg)
		if _, ok := existingCommands[command]; ok {
			continue
		}

		plan.Fallback = append(plan.Fallback, TestPlanItem{
			Command:    command,
			Reason:     fallbackTestPlanReason(options.Kind, target, pkg, defaultTargets),
			Category:   fallbackTestPlanCategory(pkg),
			Confidence: confidenceForPlanItem(pkg, true, repositorySignals),
			Package:    pkg,
			Targets:    defaultTargets,
		})
		existingCommands[command] = struct{}{}
	}
	plan.Confidence = testPlanConfidence(plan, repositorySignals)
	plan.Limitations = testPlanLimitations(plan, repositorySignals)

	return NormalizeTestPlan(plan)
}

func defaultTestPlanTargets(options TestPlanOptions, target string) []string {
	targets := uniqueSorted(options.Targets)
	if len(targets) > 0 {
		return targets
	}

	if options.Kind != TestTargetKindSymbol {
		return nil
	}

	target = strings.TrimSpace(target)
	if target == "" || target == "target" {
		return nil
	}

	return []string{target}
}

func NormalizeTestPlan(plan TestPlan) TestPlan {
	plan.Direct = nonNilTestPlanItems(plan.Direct)
	plan.Related = nonNilTestPlanItems(plan.Related)
	plan.Contracts = nonNilTestPlanItems(plan.Contracts)
	plan.CallerPackages = nonNilTestPlanItems(plan.CallerPackages)
	plan.Fallback = nonNilTestPlanItems(plan.Fallback)
	if strings.TrimSpace(plan.Confidence) == "" {
		if TestPlanEmpty(plan) {
			plan.Confidence = TestPlanConfidenceLow
		} else {
			plan.Confidence = TestPlanConfidenceMedium
		}
	}
	plan.Limitations = nonNilStrings(uniqueSorted(plan.Limitations))

	return plan
}

func TestPlanCommands(plan TestPlan) []string {
	var commands []string
	for _, item := range plan.Direct {
		commands = append(commands, item.Command)
	}
	for _, item := range plan.Related {
		commands = append(commands, item.Command)
	}
	for _, item := range plan.Contracts {
		commands = append(commands, item.Command)
	}
	for _, item := range plan.CallerPackages {
		commands = append(commands, item.Command)
	}
	for _, item := range plan.Fallback {
		commands = append(commands, item.Command)
	}

	return uniqueSorted(commands)
}

func TestPlanEmpty(plan TestPlan) bool {
	return len(plan.Direct) == 0 &&
		len(plan.Related) == 0 &&
		len(plan.Contracts) == 0 &&
		len(plan.CallerPackages) == 0 &&
		len(plan.Fallback) == 0
}

func FallbackTestPlan(commands []string) TestPlan {
	commands = uniqueSorted(commands)
	plan := TestPlan{
		Fallback: make([]TestPlanItem, 0, len(commands)),
	}

	for _, command := range commands {
		plan.Fallback = append(plan.Fallback, TestPlanItem{
			Command:    command,
			Reason:     "Run the existing suggested test command.",
			Category:   fallbackTestPlanCategory(commandPackage(command)),
			Confidence: TestPlanConfidenceLow,
		})
	}
	plan.Confidence = TestPlanConfidenceLow
	plan.Limitations = []string{"Fallback commands were provided without related-test evidence."}

	return NormalizeTestPlan(plan)
}

type testPlanItemDetails struct {
	Reason     string
	Category   string
	Confidence string
}

func testPlanItemsFromGroups(groups map[string][]string, targetsByPackage map[string][]string, defaultTargets []string, details func(string, []string, []string) testPlanItemDetails) []TestPlanItem {
	packages := make([]string, 0, len(groups))
	for pkg := range groups {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	items := make([]TestPlanItem, 0, len(packages))
	for _, pkg := range packages {
		names := uniqueSorted(groups[pkg])
		targets := uniqueSorted(targetsByPackage[pkg])
		if len(targets) == 0 {
			targets = defaultTargets
		}
		itemDetails := details(pkg, names, targets)
		item := TestPlanItem{
			Command:    testPackageCommand(pkg),
			Reason:     itemDetails.Reason,
			Category:   itemDetails.Category,
			Confidence: itemDetails.Confidence,
			Package:    pkg,
			Targets:    targets,
		}
		if len(names) == 1 {
			item.Test = names[0]
		}
		if len(names) > 0 {
			item.Tests = names
		}

		items = append(items, item)
	}

	return items
}

type testPlanEvidence struct {
	ExternalPackage bool
	DirectReference bool
	Reasons         []string
}

type testPlanEvidenceByPackage map[string]testPlanEvidence

func (evidence testPlanEvidenceByPackage) Add(test RelatedTest) {
	pkg := strings.TrimSpace(test.Package)
	if pkg == "" {
		return
	}
	current := evidence[pkg]
	current.ExternalPackage = current.ExternalPackage || test.ExternalPackage
	current.DirectReference = current.DirectReference || test.DirectReference
	current.Reasons = uniqueSorted(append(current.Reasons, test.Reasons...))
	evidence[pkg] = current
}

func categoryForDirectTests(evidence testPlanEvidence) string {
	if evidence.ExternalPackage {
		return TestPlanCategoryIntegrationLike
	}
	return TestPlanCategoryFocused
}

func categoryForRelatedTests(evidence testPlanEvidence) string {
	if evidence.ExternalPackage {
		return TestPlanCategoryIntegrationLike
	}
	return TestPlanCategoryFast
}

func fallbackTestPlanCategory(pkg string) string {
	if strings.TrimSpace(pkg) == TestPlanWholeRepositoryPackage {
		return TestPlanCategoryBroadFallback
	}
	return TestPlanCategoryFast
}

func confidenceForPlanItem(pkg string, fallback bool, signals TestPlanRepositorySignals) string {
	if fallback || packageHasRepositorySignal(pkg, signals) {
		return TestPlanConfidenceLow
	}
	return TestPlanConfidenceMedium
}

func testPlanConfidence(plan TestPlan, signals TestPlanRepositorySignals) string {
	if len(signals.GeneratedPackages) > 0 || len(signals.SkippedPackages) > 0 || len(signals.Warnings) > 0 {
		return TestPlanConfidenceLow
	}
	if testPlanHasBroadFallback(plan) {
		return TestPlanConfidenceLow
	}
	if len(plan.Direct) == 0 && len(plan.Related) == 0 && len(plan.Contracts) == 0 && len(plan.CallerPackages) == 0 {
		return TestPlanConfidenceLow
	}
	if len(plan.Fallback) > 0 && len(plan.Direct) == 0 {
		return TestPlanConfidenceLow
	}
	return TestPlanConfidenceMedium
}

func testPlanHasBroadFallback(plan TestPlan) bool {
	for _, item := range plan.Fallback {
		if item.Package == TestPlanWholeRepositoryPackage {
			return true
		}
	}
	return false
}

func testPlanLimitations(plan TestPlan, signals TestPlanRepositorySignals) []string {
	limitations := []string{TestInventoryLimitationDynamicSubtests}
	if len(plan.Direct) == 0 && len(plan.Fallback) > 0 {
		limitations = append(limitations, "No direct test references were found; fallback commands compile impacted packages but may miss behavior-specific coverage.")
	}
	for _, item := range plan.Fallback {
		if item.Package == TestPlanWholeRepositoryPackage {
			limitations = append(limitations, "Whole-repository fallback is broad because the change could not be narrowed to repository-local Go packages.")
			break
		}
	}
	if len(signals.GeneratedPackages) > 0 {
		limitations = append(limitations, "Generated packages involved in test planning: "+strings.Join(uniqueSorted(signals.GeneratedPackages), ", ")+". Generated code is compiler-visible, but direct test references may be sparse.")
	}
	if len(signals.SkippedPackages) > 0 {
		limitations = append(limitations, "Skipped module or package boundaries may hide tests outside the selected root: "+strings.Join(uniqueSorted(signals.SkippedPackages), ", ")+".")
	}
	if len(signals.Warnings) > 0 {
		limitations = append(limitations, "Package-load or repository-shape warnings lower test-plan confidence; inspect command warnings before trusting completeness.")
	}
	return uniqueSorted(limitations)
}

func packageHasRepositorySignal(pkg string, signals TestPlanRepositorySignals) bool {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return false
	}
	for _, generatedPackage := range signals.GeneratedPackages {
		if pkg == strings.TrimSpace(generatedPackage) {
			return true
		}
	}
	for _, skippedPackage := range signals.SkippedPackages {
		if pkg == strings.TrimSpace(skippedPackage) {
			return true
		}
	}
	return false
}

func normalizeTestPlanRepositorySignals(signals TestPlanRepositorySignals) TestPlanRepositorySignals {
	signals.GeneratedPackages = nonNilStrings(uniqueSorted(planPackageDisplays(signals.GeneratedPackages)))
	signals.SkippedPackages = nonNilStrings(uniqueSorted(planPackageDisplays(signals.SkippedPackages)))
	signals.Warnings = nonNilStrings(uniqueSorted(signals.Warnings))
	return signals
}

func planPackageDisplays(packages []string) []string {
	values := make([]string, 0, len(packages))
	for _, pkg := range packages {
		display := planPackageDisplay(pkg)
		if strings.TrimSpace(display) == "" {
			continue
		}
		values = append(values, display)
	}
	return values
}

func normalizedPlanPackageSet(packages []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, pkg := range packages {
		normalized := normalizePlanPackage(pkg)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func planPackageMatchesSet(pkg string, set map[string]struct{}) bool {
	if len(set) == 0 {
		return false
	}
	normalized := normalizePlanPackage(pkg)
	if normalized == "" {
		return false
	}
	if _, ok := set[normalized]; ok {
		return true
	}
	for candidate := range set {
		if candidate == "." || candidate == TestPlanWholeRepositoryPackage {
			return true
		}
		if strings.HasPrefix(candidate, normalized+"/") || strings.HasPrefix(normalized, candidate+"/") {
			return true
		}
	}
	return false
}

func planPackageDisplay(pkg string) string {
	normalized := normalizePlanPackage(pkg)
	if normalized == "" {
		return strings.TrimSpace(pkg)
	}
	if normalized == "." || normalized == TestPlanWholeRepositoryPackage {
		return normalized
	}
	return "./" + normalized
}

func normalizePlanPackage(pkg string) string {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return ""
	}
	if pkg == TestPlanWholeRepositoryPackage {
		return pkg
	}
	pkg = strings.TrimPrefix(pkg, "./")
	if pkg == "" {
		return "."
	}
	return pkg
}

func commandPackage(command string) string {
	command = strings.TrimSpace(command)
	if !strings.HasPrefix(command, "go test ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(command, "go test "))
}

func directTestPlanReason(kind TestTargetKind, target string, pkg string, names []string, targets []string) string {
	if kind == TestTargetKindFile && len(targets) > 0 {
		return "Direct tests in " + pkg + " reference symbols from " + target + ": " + strings.Join(targets, ", ") + testPlanNamesSuffix(names)
	}

	return "Direct tests in " + pkg + " reference " + testPlanTargetPhrase(target, targets) + testPlanNamesSuffix(names)
}

func relatedTestPlanReason(kind TestTargetKind, target string, pkg string, names []string, targets []string) string {
	if kind == TestTargetKindPackage {
		if len(targets) > 0 {
			return "Tests in target package " + pkg + " cover " + testPlanTargetPhrase(target, targets) + testPlanNamesSuffix(names)
		}
		return "Tests in target package " + pkg + testPlanNamesSuffix(names)
	}
	if kind == TestTargetKindFile {
		if len(targets) > 0 {
			return "Same-package tests in " + pkg + " cover symbols from " + target + ": " + strings.Join(targets, ", ") + testPlanNamesSuffix(names)
		}

		return "Same-package tests in " + pkg + " are related to " + target + testPlanNamesSuffix(names)
	}
	if len(targets) > 0 {
		return "Same-package tests in " + pkg + " cover " + testPlanTargetPhrase(target, targets) + testPlanNamesSuffix(names)
	}

	return "Same-package tests in " + pkg + " are related to " + target + testPlanNamesSuffix(names)
}

func callerPackageTestPlanReason(pkg string, names []string, targets []string) string {
	if len(targets) > 0 {
		return "Tests in caller package " + pkg + " cover impacted code paths from " + testPlanTargetPhrase("changed symbols", targets) + testPlanNamesSuffix(names)
	}

	return "Tests in caller package " + pkg + " cover impacted code paths" + testPlanNamesSuffix(names)
}

func contractTestPlanReason(pkg string, names []string, targets []string) string {
	if len(targets) > 0 {
		return "Contract tests in " + pkg + " cover affected interfaces or implementations from " + testPlanTargetPhrase("contract signals", targets) + testPlanNamesSuffix(names)
	}

	return "Contract tests in " + pkg + " cover affected interfaces or implementations" + testPlanNamesSuffix(names)
}

func fallbackTestPlanReason(kind TestTargetKind, target string, pkg string, targets []string) string {
	if pkg == TestPlanWholeRepositoryPackage {
		return "Run the full test suite because the change could not be narrowed to repository-local Go packages."
	}
	if len(targets) > 0 {
		target = strings.TrimSpace(target)
		if target == "changed symbols" {
			return "Run package tests for " + pkg + " to compile impacted code from " + testPlanTargetPhrase(target, targets) + " and cover tests not matched directly."
		}
		if kind == TestTargetKindFile {
			return "Run package tests for " + pkg + " to compile code related to symbols from " + target + " and cover tests not matched directly."
		}

		return "Run package tests for " + pkg + " to compile code related to " + testPlanTargetPhrase(target, targets) + " and cover tests not matched directly."
	}

	return "Run package tests for " + pkg + " to compile impacted code and cover tests not matched directly."
}

func testPlanTargetPhrase(target string, targets []string) string {
	target = strings.TrimSpace(target)
	targets = uniqueSorted(targets)
	if len(targets) == 0 {
		return target
	}
	if len(targets) == 1 && targets[0] == target {
		return target
	}
	if target == "" || target == "target" {
		return strings.Join(targets, ", ")
	}

	return target + " " + strings.Join(targets, ", ")
}

func testPlanNamesSuffix(names []string) string {
	names = uniqueSorted(names)
	if len(names) == 0 {
		return "."
	}

	return ": " + strings.Join(names, ", ") + "."
}

func testPlanCommandSet(plan TestPlan) map[string]struct{} {
	commands := make(map[string]struct{})
	for _, item := range plan.Direct {
		commands[item.Command] = struct{}{}
	}
	for _, item := range plan.Related {
		commands[item.Command] = struct{}{}
	}
	for _, item := range plan.Contracts {
		commands[item.Command] = struct{}{}
	}
	for _, item := range plan.CallerPackages {
		commands[item.Command] = struct{}{}
	}
	for _, item := range plan.Fallback {
		commands[item.Command] = struct{}{}
	}

	return commands
}

func testPackageCommand(pkg string) string {
	return "go test " + pkg
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		set[value] = struct{}{}
	}

	return set
}

func hasString(values map[string]struct{}, value string) bool {
	_, ok := values[value]
	return ok
}

func packageDifference(values []string, excluded []string) []string {
	excludedSet := stringSet(excluded)
	var result []string
	for _, value := range values {
		if _, ok := excludedSet[value]; ok {
			continue
		}
		result = append(result, value)
	}

	return uniqueSorted(result)
}

func sortedTestPackages(tests []RelatedTest) []string {
	var packages []string
	for _, test := range tests {
		packages = append(packages, test.Package)
	}

	return uniqueSorted(packages)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

func sortedMapKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)

	return result
}

func nonNilTestPlanItems(values []TestPlanItem) []TestPlanItem {
	if values == nil {
		return []TestPlanItem{}
	}

	return values
}
