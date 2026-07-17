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
	if !strings.Contains(plan.Fallback[0].Reason, "full test suite") {
		t.Fatalf("expected whole-repository fallback reason, got %#v", plan.Fallback[0])
	}
	if !reflect.DeepEqual(TestPlanCommands(plan), []string{"go test ./..."}) {
		t.Fatalf("expected whole-repository command, got %#v", TestPlanCommands(plan))
	}
}

func assertTestPlanContract(t *testing.T, plan TestPlan) {
	t.Helper()

	if plan.Direct == nil || plan.Related == nil || plan.Contracts == nil || plan.CallerPackages == nil || plan.Fallback == nil {
		t.Fatalf("expected non-nil test plan groups, got %#v", plan)
	}
	for _, items := range [][]TestPlanItem{plan.Direct, plan.Related, plan.Contracts, plan.CallerPackages, plan.Fallback} {
		for _, item := range items {
			if item.Command == "" {
				t.Fatalf("expected plan item command, got %#v", item)
			}
			if item.Reason == "" {
				t.Fatalf("expected plan item reason, got %#v", item)
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
