package sherpa

import (
	"sort"
	"strings"
)

type TestPlan struct {
	Direct         []TestPlanItem `json:"direct"`
	Related        []TestPlanItem `json:"related"`
	CallerPackages []TestPlanItem `json:"callerPackages"`
	Fallback       []TestPlanItem `json:"fallback"`
}

type TestPlanItem struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
	Package string `json:"package,omitempty"`
	Test    string `json:"test,omitempty"`
}

type TestPlanOptions struct {
	Target           string
	Kind             TestTargetKind
	TargetPackages   []string
	CallerPackages   []string
	FallbackPackages []string
}

func PlanTests(tests []RelatedTest, options TestPlanOptions) TestPlan {
	target := strings.TrimSpace(options.Target)
	if target == "" {
		target = "target"
	}

	targetPackages := uniqueSorted(options.TargetPackages)
	callerPackages := uniqueSorted(options.CallerPackages)
	fallbackPackages := uniqueSorted(options.FallbackPackages)
	targetPackageSet := stringSet(targetPackages)
	callerPackageSet := stringSet(callerPackages)

	directGroups := make(map[string][]string)
	relatedGroups := make(map[string][]string)
	callerGroups := make(map[string][]string)

	for _, test := range tests {
		pkg := strings.TrimSpace(test.Package)
		if pkg == "" {
			continue
		}

		switch {
		case test.DirectReference:
			directGroups[pkg] = append(directGroups[pkg], test.Name)
		case hasString(callerPackageSet, pkg):
			callerGroups[pkg] = append(callerGroups[pkg], test.Name)
		case len(targetPackageSet) == 0 || hasString(targetPackageSet, pkg):
			relatedGroups[pkg] = append(relatedGroups[pkg], test.Name)
		default:
			callerGroups[pkg] = append(callerGroups[pkg], test.Name)
		}
	}

	plan := TestPlan{
		Direct: testPlanItemsFromGroups(directGroups, func(pkg string, names []string) string {
			return directTestPlanReason(target, pkg, names)
		}),
		Related: testPlanItemsFromGroups(relatedGroups, func(pkg string, names []string) string {
			return relatedTestPlanReason(options.Kind, target, pkg, names)
		}),
		CallerPackages: testPlanItemsFromGroups(callerGroups, func(pkg string, names []string) string {
			return callerPackageTestPlanReason(pkg, names)
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
			Reason:  fallbackTestPlanReason(pkg),
			Package: pkg,
		})
		existingCommands[command] = struct{}{}
	}

	return NormalizeTestPlan(plan)
}

func NormalizeTestPlan(plan TestPlan) TestPlan {
	plan.Direct = nonNilTestPlanItems(plan.Direct)
	plan.Related = nonNilTestPlanItems(plan.Related)
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

func testPlanItemsFromGroups(groups map[string][]string, reason func(string, []string) string) []TestPlanItem {
	packages := make([]string, 0, len(groups))
	for pkg := range groups {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	items := make([]TestPlanItem, 0, len(packages))
	for _, pkg := range packages {
		names := uniqueSorted(groups[pkg])
		item := TestPlanItem{
			Command: testPackageCommand(pkg),
			Reason:  reason(pkg, names),
			Package: pkg,
		}
		if len(names) == 1 {
			item.Test = names[0]
		}

		items = append(items, item)
	}

	return items
}

func directTestPlanReason(target string, pkg string, names []string) string {
	return "Direct tests in " + pkg + " reference " + target + testPlanNamesSuffix(names)
}

func relatedTestPlanReason(kind TestTargetKind, target string, pkg string, names []string) string {
	if kind == TestTargetKindPackage {
		return "Tests in target package " + pkg + testPlanNamesSuffix(names)
	}

	return "Same-package tests in " + pkg + " are related to " + target + testPlanNamesSuffix(names)
}

func callerPackageTestPlanReason(pkg string, names []string) string {
	return "Tests in caller package " + pkg + " cover impacted code paths" + testPlanNamesSuffix(names)
}

func fallbackTestPlanReason(pkg string) string {
	return "Run package tests for " + pkg + " to compile impacted code and cover tests not matched directly."
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
