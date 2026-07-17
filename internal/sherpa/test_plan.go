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
}

const TestPlanWholeRepositoryPackage = "./..."

type TestPlanItem struct {
	Command string   `json:"command"`
	Reason  string   `json:"reason"`
	Package string   `json:"package,omitempty"`
	Test    string   `json:"test,omitempty"`
	Tests   []string `json:"tests,omitempty"`
	Targets []string `json:"targets,omitempty"`
}

type TestPlanOptions struct {
	Target           string
	Kind             TestTargetKind
	TargetPackages   []string
	ContractPackages []string
	CallerPackages   []string
	FallbackPackages []string
	Targets          []string
}

func PlanTests(tests []RelatedTest, options TestPlanOptions) TestPlan {
	target := strings.TrimSpace(options.Target)
	if target == "" {
		target = "target"
	}

	targetPackages := uniqueSorted(options.TargetPackages)
	contractPackages := uniqueSorted(options.ContractPackages)
	callerPackages := uniqueSorted(options.CallerPackages)
	fallbackPackages := uniqueSorted(options.FallbackPackages)
	defaultTargets := defaultTestPlanTargets(options, target)
	targetPackageSet := stringSet(targetPackages)
	contractPackageSet := stringSet(contractPackages)
	callerPackageSet := stringSet(callerPackages)

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

	plan := TestPlan{
		Direct: testPlanItemsFromGroups(directGroups, directTargets, defaultTargets, func(pkg string, names []string, targets []string) string {
			return directTestPlanReason(options.Kind, target, pkg, names, targets)
		}),
		Related: testPlanItemsFromGroups(relatedGroups, relatedTargets, defaultTargets, func(pkg string, names []string, targets []string) string {
			return relatedTestPlanReason(options.Kind, target, pkg, names, targets)
		}),
		Contracts: testPlanItemsFromGroups(contractGroups, contractTargets, defaultTargets, func(pkg string, names []string, targets []string) string {
			return contractTestPlanReason(pkg, names, targets)
		}),
		CallerPackages: testPlanItemsFromGroups(callerGroups, callerTargets, defaultTargets, func(pkg string, names []string, targets []string) string {
			return callerPackageTestPlanReason(pkg, names, targets)
		}),
	}

	existingCommands := testPlanCommandSet(plan)
	for _, pkg := range fallbackPackages {
		command := testPackageCommand(pkg)
		if _, ok := existingCommands[command]; ok {
			continue
		}

		plan.Fallback = append(plan.Fallback, TestPlanItem{
			Command: command,
			Reason:  fallbackTestPlanReason(options.Kind, target, pkg, defaultTargets),
			Package: pkg,
			Targets: defaultTargets,
		})
		existingCommands[command] = struct{}{}
	}

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
			Command: command,
			Reason:  "Run the existing suggested test command.",
		})
	}

	return NormalizeTestPlan(plan)
}

func testPlanItemsFromGroups(groups map[string][]string, targetsByPackage map[string][]string, defaultTargets []string, reason func(string, []string, []string) string) []TestPlanItem {
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
		item := TestPlanItem{
			Command: testPackageCommand(pkg),
			Reason:  reason(pkg, names, targets),
			Package: pkg,
			Targets: targets,
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
