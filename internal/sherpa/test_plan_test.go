package sherpa

import (
	"reflect"
	"strings"
	"testing"
)

func TestPlanTestsGroupsContractPackages(t *testing.T) {
	plan := PlanTests([]RelatedTest{
		{
			Name:    "TestAuthenticatorContract",
			Package: "./internal/auth",
			Reasons: []string{
				RelatedTestReasonContract,
			},
			Targets: []string{"./internal/auth.Authenticator"},
		},
		{
			Name:            "TestDirectImplementation",
			Package:         "./internal/jwt",
			DirectReference: true,
			Reasons: []string{
				RelatedTestReasonDirectReference,
				RelatedTestReasonContract,
			},
			Targets: []string{"./internal/jwt.JWTAuthenticator"},
		},
	}, TestPlanOptions{
		Target:           "changed symbols",
		Kind:             TestTargetKindPackage,
		ContractPackages: []string{"./internal/auth", "./internal/jwt"},
		FallbackPackages: []string{"./internal/auth", "./internal/jwt"},
	})

	if len(plan.Direct) != 1 || plan.Direct[0].Package != "./internal/jwt" {
		t.Fatalf("expected direct implementation test to stay direct, got %#v", plan)
	}
	if len(plan.Contracts) != 1 || plan.Contracts[0].Package != "./internal/auth" {
		t.Fatalf("expected contract plan item for ./internal/auth, got %#v", plan)
	}
	if !reflect.DeepEqual(plan.Contracts[0].Tests, []string{"TestAuthenticatorContract"}) {
		t.Fatalf("expected contract tests, got %#v", plan.Contracts[0].Tests)
	}
	if !reflect.DeepEqual(TestPlanCommands(plan), []string{"go test ./internal/auth", "go test ./internal/jwt"}) {
		t.Fatalf("expected flat commands to include contracts, got %#v", TestPlanCommands(plan))
	}
}

func TestPlanTestsPreservesGroupShapeAndKnownFields(t *testing.T) {
	plan := PlanTests([]RelatedTest{
		{
			Name:            "TestDirect",
			Package:         "./internal/service",
			DirectReference: true,
			Reasons:         []string{RelatedTestReasonDirectReference},
			Targets:         []string{"./internal/service.Target"},
		},
		{
			Name:    "TestRelated",
			Package: "./internal/service",
			Reasons: []string{RelatedTestReasonSamePackage},
			Targets: []string{"./internal/service.Target"},
		},
		{
			Name:    "TestContract",
			Package: "./internal/contracts",
			Reasons: []string{RelatedTestReasonContract},
			Targets: []string{"./internal/service.Worker"},
		},
		{
			Name:    "TestCaller",
			Package: "./internal/caller",
			Reasons: []string{RelatedTestReasonCallerPackage},
			Targets: []string{"./internal/service.Target"},
		},
	}, TestPlanOptions{
		Target:           "./internal/service.Target",
		Kind:             TestTargetKindSymbol,
		TargetPackages:   []string{"./internal/service"},
		ContractPackages: []string{"./internal/contracts"},
		CallerPackages:   []string{"./internal/caller"},
		FallbackPackages: []string{"./internal/service", "./internal/contracts", "./internal/caller", "./internal/fallback"},
	})

	assertTestPlanContract(t, plan)
	assertTestPlanItem(t, plan.Direct, "./internal/service", "TestDirect", []string{"TestDirect"}, []string{"./internal/service.Target"})
	assertTestPlanItem(t, plan.Related, "./internal/service", "TestRelated", []string{"TestRelated"}, []string{"./internal/service.Target"})
	assertTestPlanItem(t, plan.Contracts, "./internal/contracts", "TestContract", []string{"TestContract"}, []string{"./internal/service.Worker"})
	assertTestPlanItem(t, plan.CallerPackages, "./internal/caller", "TestCaller", []string{"TestCaller"}, []string{"./internal/service.Target"})
	assertTestPlanItem(t, plan.Fallback, "./internal/fallback", "", nil, []string{"./internal/service.Target"})
	assertTestPlanItemCategory(t, plan.Direct[0], TestPlanCategoryFocused, TestPlanConfidenceMedium)
	assertTestPlanItemCategory(t, plan.Related[0], TestPlanCategoryFast, TestPlanConfidenceMedium)
	assertTestPlanItemCategory(t, plan.Contracts[0], TestPlanCategoryContract, TestPlanConfidenceMedium)
	assertTestPlanItemCategory(t, plan.CallerPackages[0], TestPlanCategoryCallerPackage, TestPlanConfidenceMedium)
	assertTestPlanItemCategory(t, plan.Fallback[0], TestPlanCategoryFast, TestPlanConfidenceLow)
}

func TestPlanTestsAddsFallbackWhenDirectEvidenceIsEmpty(t *testing.T) {
	plan := PlanTests(nil, TestPlanOptions{
		Target:           "./internal/service.Target",
		Kind:             TestTargetKindSymbol,
		TargetPackages:   []string{"./internal/service"},
		FallbackPackages: []string{"./internal/service"},
	})

	assertTestPlanContract(t, plan)
	if len(plan.Direct) != 0 || len(plan.Related) != 0 || len(plan.Contracts) != 0 || len(plan.CallerPackages) != 0 {
		t.Fatalf("expected only fallback recommendations, got %#v", plan)
	}
	assertTestPlanItem(t, plan.Fallback, "./internal/service", "", nil, []string{"./internal/service.Target"})
	assertTestPlanItemCategory(t, plan.Fallback[0], TestPlanCategoryFast, TestPlanConfidenceLow)
	if plan.Confidence != TestPlanConfidenceLow {
		t.Fatalf("expected fallback-only plan confidence low, got %#v", plan)
	}
	if !testPlanLimitationsContain(plan, "No direct test references were found") {
		t.Fatalf("expected fallback limitation, got %#v", plan.Limitations)
	}
	if !reflect.DeepEqual(TestPlanCommands(plan), []string{"go test ./internal/service"}) {
		t.Fatalf("expected fallback command, got %#v", TestPlanCommands(plan))
	}
}

func TestPlanTestsDistinguishesWholeRepositoryFallback(t *testing.T) {
	plan := PlanTests(nil, TestPlanOptions{
		Target:           "changed files",
		FallbackPackages: []string{TestPlanWholeRepositoryPackage},
	})

	assertTestPlanContract(t, plan)
	assertTestPlanItem(t, plan.Fallback, TestPlanWholeRepositoryPackage, "", nil, nil)
	assertTestPlanItemCategory(t, plan.Fallback[0], TestPlanCategoryBroadFallback, TestPlanConfidenceLow)
	if !strings.Contains(plan.Fallback[0].Reason, "full test suite") {
		t.Fatalf("expected whole-repository fallback reason, got %#v", plan.Fallback[0])
	}
	if !testPlanLimitationsContain(plan, "Whole-repository fallback is broad") {
		t.Fatalf("expected broad fallback limitation, got %#v", plan.Limitations)
	}
	if !reflect.DeepEqual(TestPlanCommands(plan), []string{"go test ./..."}) {
		t.Fatalf("expected whole-repository command, got %#v", TestPlanCommands(plan))
	}
}

func TestPlanTestsMarksExternalPackageTestsIntegrationLike(t *testing.T) {
	plan := PlanTests([]RelatedTest{
		{
			Name:            "TestExternalDirect",
			Package:         "./internal/service",
			ExternalPackage: true,
			DirectReference: true,
			Reasons:         []string{RelatedTestReasonDirectReference, RelatedTestReasonExternalPackage},
			Targets:         []string{"./internal/service.Target"},
		},
	}, TestPlanOptions{
		Target:           "./internal/service.Target",
		Kind:             TestTargetKindSymbol,
		TargetPackages:   []string{"./internal/service"},
		FallbackPackages: []string{"./internal/service"},
	})

	assertTestPlanContract(t, plan)
	assertTestPlanItemCategory(t, plan.Direct[0], TestPlanCategoryIntegrationLike, TestPlanConfidenceMedium)
}

func TestPlanTestsRepositorySignalsLowerConfidence(t *testing.T) {
	plan := PlanTests([]RelatedTest{
		{
			Name:            "TestGenerated",
			Package:         "./internal/generated",
			DirectReference: true,
			Reasons:         []string{RelatedTestReasonDirectReference},
			Targets:         []string{"./internal/generated.Target"},
		},
	}, TestPlanOptions{
		Target:           "./internal/generated.Target",
		Kind:             TestTargetKindSymbol,
		TargetPackages:   []string{"./internal/generated"},
		FallbackPackages: []string{"./internal/generated"},
		RepositorySignals: TestPlanRepositorySignals{
			GeneratedPackages: []string{"./internal/generated"},
			SkippedPackages:   []string{"./nested/module"},
			Warnings:          []string{"package load warning"},
		},
	})

	assertTestPlanContract(t, plan)
	assertTestPlanItemCategory(t, plan.Direct[0], TestPlanCategoryFocused, TestPlanConfidenceLow)
	if plan.Confidence != TestPlanConfidenceLow {
		t.Fatalf("expected repository signals to lower confidence, got %#v", plan)
	}
	for _, fragment := range []string{"Generated packages involved", "Skipped module or package boundaries", "Package-load or repository-shape warnings"} {
		if !testPlanLimitationsContain(plan, fragment) {
			t.Fatalf("expected limitation containing %q, got %#v", fragment, plan.Limitations)
		}
	}
}

func assertTestPlanContract(t *testing.T, plan TestPlan) {
	t.Helper()

	if plan.Direct == nil || plan.Related == nil || plan.Contracts == nil || plan.CallerPackages == nil || plan.Fallback == nil {
		t.Fatalf("expected non-nil test plan groups, got %#v", plan)
	}
	if strings.TrimSpace(plan.Confidence) == "" {
		t.Fatalf("expected test plan confidence, got %#v", plan)
	}
	if plan.Limitations == nil {
		t.Fatalf("expected non-nil test plan limitations, got %#v", plan)
	}
	for _, items := range [][]TestPlanItem{plan.Direct, plan.Related, plan.Contracts, plan.CallerPackages, plan.Fallback} {
		for _, item := range items {
			if item.Command == "" {
				t.Fatalf("expected plan item command, got %#v", item)
			}
			if item.Reason == "" {
				t.Fatalf("expected plan item reason, got %#v", item)
			}
			if item.Category == "" {
				t.Fatalf("expected plan item category, got %#v", item)
			}
			if item.Confidence == "" {
				t.Fatalf("expected plan item confidence, got %#v", item)
			}
		}
	}
}

func assertTestPlanItem(t *testing.T, items []TestPlanItem, pkg string, test string, tests []string, targets []string) {
	t.Helper()

	if len(items) != 1 {
		t.Fatalf("expected one plan item for %s, got %#v", pkg, items)
	}
	item := items[0]
	if item.Package != pkg {
		t.Fatalf("expected package %s, got %#v", pkg, item)
	}
	if item.Command != "go test "+pkg {
		t.Fatalf("expected package test command for %s, got %#v", pkg, item)
	}
	if item.Test != test {
		t.Fatalf("expected test %q, got %#v", test, item)
	}
	if !reflect.DeepEqual(item.Tests, tests) {
		t.Fatalf("expected tests %#v, got %#v", tests, item.Tests)
	}
	if !reflect.DeepEqual(item.Targets, targets) {
		t.Fatalf("expected targets %#v, got %#v", targets, item.Targets)
	}
}

func assertTestPlanItemCategory(t *testing.T, item TestPlanItem, category string, confidence string) {
	t.Helper()

	if item.Category != category {
		t.Fatalf("expected category %q, got %#v", category, item)
	}
	if item.Confidence != confidence {
		t.Fatalf("expected confidence %q, got %#v", confidence, item)
	}
}

func testPlanLimitationsContain(plan TestPlan, fragment string) bool {
	for _, limitation := range plan.Limitations {
		if strings.Contains(limitation, fragment) {
			return true
		}
	}
	return false
}
