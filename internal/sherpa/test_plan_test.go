package sherpa

import (
	"reflect"
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
