package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

const analysisModeSnapshot = "snapshot"

func loadSymbolsForInventoryCommand(root string, invocation cliInvocation) ([]sherpa.Symbol, []string, string, error) {
	var warnings []string
	if invocation.UseSnapshot {
		stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{
			BuildTags: invocation.BuildTags,
		})
		if inspect.Status == snapshotstore.StatusValid {
			return cloneSlice(stored.Symbols), nil, analysisModeSnapshot, nil
		}

		warnings = append(warnings, snapshotFallbackWarning(inspect))
	}

	symbols, err := sherpa.ParseRepository(root)
	if err != nil {
		return nil, warnings, "", err
	}

	return symbols, warnings, fallbackAnalysisMode(invocation), nil
}

func loadPackagesForInventoryCommand(root string, invocation cliInvocation) ([]sherpa.PackageSummary, []string, string, error) {
	var warnings []string
	if invocation.UseSnapshot && invocation.IncludeTests {
		stored, inspect := snapshotstore.LoadReusable(root, snapshotstore.BuildOptions{
			BuildTags: invocation.BuildTags,
		})
		if inspect.Status == snapshotstore.StatusValid {
			return cloneSlice(stored.Packages), nil, analysisModeSnapshot, nil
		}

		warnings = append(warnings, snapshotFallbackWarning(inspect))
	} else if invocation.UseSnapshot {
		warnings = append(warnings, "snapshot not used: packages without --tests need production-only package counts; using live repository analysis")
	}

	packages, err := sherpa.FindPackageSummaries(root, sherpa.PackageInventoryOptions{
		IncludeTests: invocation.IncludeTests,
	})
	if err != nil {
		return nil, warnings, "", err
	}

	return packages, warnings, fallbackAnalysisMode(invocation), nil
}

func snapshotFallbackWarning(inspect snapshotstore.InspectResult) string {
	message := strings.TrimSpace(inspect.Message)
	if message == "" {
		message = "snapshot could not be used"
	}

	if len(inspect.StaleReasons) > 0 {
		return fmt.Sprintf("snapshot not used: %s (%s); using live repository analysis", message, strings.Join(inspect.StaleReasons, ", "))
	}

	return fmt.Sprintf("snapshot not used: %s; using live repository analysis", message)
}

func fallbackAnalysisMode(invocation cliInvocation) string {
	if invocation.UseSnapshot {
		return analysisModeAST
	}

	return ""
}

func writeHumanWarnings(stderr io.Writer, warnings []string) {
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}

		fmt.Fprintln(stderr, "warning:", warning)
	}
}

func cloneSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return append([]T{}, values...)
}
