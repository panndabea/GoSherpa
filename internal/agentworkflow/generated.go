package agentworkflow

import (
	"fmt"
	"path/filepath"
	"strings"

	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

const largeGeneratedReadingOrderFileBytes int64 = 32 * 1024

func applyGeneratedFilePolicy(root string, report Report) Report {
	generatedByPath := generatedReadingFiles(root, report.ReadingOrder)
	if len(generatedByPath) == 0 {
		return report
	}

	readingOrder, summarized := summarizeLargeGeneratedFileSteps(report.ReadingOrder, generatedByPath)
	if summarized == 0 {
		return report
	}

	report.ReadingOrder = readingOrder
	limitation := "Large generated Go files are compiler-visible and included in analysis; agent reading order summarizes generated file steps after hand-written steps when they would otherwise dominate."
	report.Readiness.Limitations = append(report.Readiness.Limitations, limitation)
	report.Limitations = append(report.Limitations, limitation)
	return report
}

func generatedReadingFiles(root string, steps []explainengine.ReadingStep) map[string]sherpa.GeneratedFileSummary {
	files := make([]string, 0, len(steps))
	seen := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		file := readingStepFileKey(step.Position.File)
		if file == "" {
			continue
		}
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		files = append(files, file)
	}

	summaries := sherpa.GeneratedGoFileSummaries(root, files)
	result := make(map[string]sherpa.GeneratedFileSummary, len(summaries))
	for _, summary := range summaries {
		result[readingStepFileKey(summary.Path)] = summary
	}
	return result
}

func summarizeLargeGeneratedFileSteps(steps []explainengine.ReadingStep, generatedByPath map[string]sherpa.GeneratedFileSummary) ([]explainengine.ReadingStep, int) {
	if len(steps) == 0 || len(generatedByPath) == 0 {
		return steps, 0
	}

	var retained []explainengine.ReadingStep
	var generated []explainengine.ReadingStep
	for _, step := range steps {
		summary, ok := generatedByPath[readingStepFileKey(step.Position.File)]
		if ok && isChangedFileReadingStep(step) && summary.SizeBytes >= largeGeneratedReadingOrderFileBytes {
			generated = append(generated, step)
			continue
		}
		retained = append(retained, step)
	}
	if !largeGeneratedStepsWouldDominate(steps, generated) {
		return steps, 0
	}

	retained = append(retained, generatedSummaryStep(generated))
	return retained, len(generated)
}

func largeGeneratedStepsWouldDominate(steps []explainengine.ReadingStep, generated []explainengine.ReadingStep) bool {
	if len(generated) == 0 {
		return false
	}
	if len(generated) > 1 {
		return true
	}

	changedFileSteps := 0
	for _, step := range steps {
		if isChangedFileReadingStep(step) {
			changedFileSteps++
		}
	}
	return changedFileSteps > 0 && len(generated) >= changedFileSteps
}

func generatedSummaryStep(steps []explainengine.ReadingStep) explainengine.ReadingStep {
	first := steps[0]
	title := fmt.Sprintf("Generated files summary: %d large changed files", len(steps))
	if len(steps) == 1 {
		title = "Generated file summary: " + readingStepFileKey(first.Position.File)
	}
	return explainengine.ReadingStep{
		Title:    title,
		Reason:   "Generated Go files are included for compiler-visible facts; read hand-written files first unless the generated declaration is the direct target.",
		Position: first.Position,
		Range:    first.Range,
	}
}

func isChangedFileReadingStep(step explainengine.ReadingStep) bool {
	return strings.HasPrefix(strings.TrimSpace(step.Title), "Changed file:")
}

func readingStepFileKey(file string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(file)))
}
