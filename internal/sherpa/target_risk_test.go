package sherpa

import "testing"

func TestNormalizeTargetRiskSummaryDefaultsToLocalLow(t *testing.T) {
	got := NormalizeTargetRiskSummary(TargetRiskSummary{})

	if got.Level != TargetRiskLevelLow {
		t.Fatalf("expected low level, got %#v", got)
	}
	if got.Scope != TargetRiskScopeLocal {
		t.Fatalf("expected local scope, got %#v", got)
	}
	if got.Reasons == nil {
		t.Fatalf("expected non-nil reasons, got %#v", got)
	}
	if got.Limitations == nil {
		t.Fatalf("expected non-nil limitations, got %#v", got)
	}
}
