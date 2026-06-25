package impact

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyzeDiffReportsChangedAndAffectedPackages(t *testing.T) {
	root := initImpactGitTestRepository(t)

	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# test\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n\ntype Session struct{}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session_test.go"), "package auth\n\nimport \"testing\"\n\nfunc TestSession(t *testing.T) {}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler.go"), "package api\n\nimport \"example.com/app/internal/auth\"\n\nvar _ = auth.Session{}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "api", "handler_test.go"), "package api\n\nimport \"testing\"\n\nfunc TestHandler(t *testing.T) {}\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "other", "other.go"), "package other\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "other", "other_test.go"), "package other\n\nimport \"testing\"\n\nfunc TestOther(t *testing.T) {}\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# changed\n")
	writeImpactTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), "package auth\n\ntype Session struct{}\n\nfunc NewSession() Session { return Session{} }\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "change auth")
	head := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	report, err := AnalyzeDiff(root, base, head)
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	assertStrings(t, report.ChangedFiles, []string{"README.md", "internal/auth/session.go"})
	assertStrings(t, report.ChangedPackages, []string{"./internal/auth"})
	assertStrings(t, report.AffectedPackages, []string{"./internal/api", "./internal/auth"})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{"./internal/api:TestHandler", "./internal/auth:TestSession"})
	assertStrings(t, report.TestCommands, []string{"go test ./internal/api", "go test ./internal/auth"})
	assertStrings(t, report.Warnings, []string{})
}

func TestAnalyzeDiffReturnsEmptyImpactForNonGoChanges(t *testing.T) {
	root := initImpactGitTestRepository(t)

	writeImpactTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# test\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "initial")
	base := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	writeImpactTestFile(t, filepath.Join(root, "README.md"), "# changed\n")
	runImpactGit(t, root, "add", ".")
	runImpactGit(t, root, "commit", "-m", "docs")
	head := strings.TrimSpace(runImpactGit(t, root, "rev-parse", "HEAD"))

	report, err := NewAnalyzer(root).AnalyzeDiff(base, head)
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	assertStrings(t, report.ChangedFiles, []string{"README.md"})
	assertStrings(t, report.ChangedPackages, []string{})
	assertStrings(t, report.AffectedPackages, []string{})
	assertStrings(t, relatedTestNames(report.AffectedTests), []string{})
	assertStrings(t, report.TestCommands, []string{})
	assertStrings(t, report.Warnings, []string{})
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func relatedTestNames(tests []RelatedTest) []string {
	names := make([]string, 0, len(tests))
	for _, test := range tests {
		names = append(names, test.Package+":"+test.Name)
	}

	return names
}
