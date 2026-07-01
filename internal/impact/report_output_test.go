package impact

import (
	"strings"
	"testing"
)

func TestFormatPackageReportIncludesInterfaceAnalysisMode(t *testing.T) {
	report := ImpactReport{
		ChangedPackages:       []string{"./internal/auth"},
		AffectedPackages:      []string{"./internal/auth"},
		InterfaceAnalysisMode: InterfaceAnalysisModeTypechecked,
	}

	output := FormatPackageReport(report)

	for _, want := range []string{
		"IMPACT PACKAGE",
		"ANALYSIS",
		"Interface analysis: typechecked",
		"CHANGED PACKAGES",
		"./internal/auth",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}
