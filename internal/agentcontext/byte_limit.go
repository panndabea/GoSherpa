package agentcontext

import (
	"encoding/json"

	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func applySymbolByteLimit(report Report, maxBytes int) Report {
	truncation := truncationValue(report.Truncated)
	enforceByteLimit(maxBytes, func(truncation Truncation) int {
		report.Truncated = reportTruncation(truncation)
		return encodedJSONLen(normalizeReport(report))
	}, &truncation,
		func() bool {
			return trimTestPlanItem(&report.TestPlan, &truncation.TestPlanItems)
		},
		func() bool {
			return trimSourceContextLine(&report.SourceContext, &truncation.SourceLines)
		},
		func() bool {
			if !trimLast(&report.RelatedTests, &truncation.RelatedTests) {
				return false
			}
			report.ReadingOrder = symbolReadingOrder(report)
			return true
		},
		func() bool {
			return trimLast(&report.AffectedImplementations, &truncation.AffectedImplementations)
		},
		func() bool {
			return trimLast(&report.AffectedInterfaces, &truncation.AffectedInterfaces)
		},
		func() bool {
			return trimEntryPointExample(report.EntryPointSummary, &truncation.EntryPointExamples)
		},
		func() bool {
			return trimEntryPointCount(report.EntryPointSummary, &truncation.EntryPointCounts)
		},
		func() bool {
			return trimLast(&report.References, &truncation.References)
		},
		func() bool {
			if !trimLast(&report.Callers, &truncation.Callers) {
				return false
			}
			report.ReadingOrder = symbolReadingOrder(report)
			return true
		},
		func() bool {
			if !trimLast(&report.Callees, &truncation.Callees) {
				return false
			}
			report.ReadingOrder = symbolReadingOrder(report)
			return true
		},
		func() bool {
			return trimLastPreserving(&report.TestCommands, 1, &truncation.TestCommands)
		},
		func() bool {
			return trimLastPreserving(&report.AffectedPackages, 1, &truncation.AffectedPackages)
		},
		func() bool {
			return trimLast(&report.ReadingOrder, &truncation.ReadingOrder)
		},
	)

	report.Truncated = reportTruncation(truncation)
	return report
}

func applyFileByteLimit(report FileReport, maxBytes int) FileReport {
	truncation := truncationValue(report.Truncated)
	enforceByteLimit(maxBytes, func(truncation Truncation) int {
		report.Truncated = reportTruncation(truncation)
		return encodedJSONLen(normalizeFileReport(report))
	}, &truncation,
		func() bool {
			return trimTestPlanItem(&report.TestPlan, &truncation.TestPlanItems)
		},
		func() bool {
			return trimLast(&report.SourceContexts, &truncation.SourceContexts)
		},
		func() bool {
			if !trimLast(&report.AffectedTests, &truncation.AffectedTests) {
				return false
			}
			report.ReadingOrder = fileReadingOrder(report)
			return true
		},
		func() bool {
			return trimLast(&report.AffectedImplementations, &truncation.AffectedImplementations)
		},
		func() bool {
			return trimLast(&report.AffectedInterfaces, &truncation.AffectedInterfaces)
		},
		func() bool {
			return trimLastPreserving(&report.TestCommands, 1, &truncation.TestCommands)
		},
		func() bool {
			return trimLastPreserving(&report.AffectedPackages, 1, &truncation.AffectedPackages)
		},
		func() bool {
			if !trimLastPreserving(&report.Symbols, 1, &truncation.Symbols) {
				return false
			}
			report.ReadingOrder = fileReadingOrder(report)
			return true
		},
		func() bool {
			return trimLast(&report.ReadingOrder, &truncation.ReadingOrder)
		},
	)

	report.Truncated = reportTruncation(truncation)
	return report
}

func trimEntryPointExample(summary *sherpa.EntryPointSummary, counter *int) bool {
	if summary == nil || len(summary.Examples) == 0 {
		return false
	}
	summary.Examples = append([]sherpa.EntryPoint{}, summary.Examples[:len(summary.Examples)-1]...)
	summary.Truncated++
	(*counter)++
	return true
}

func trimEntryPointCount(summary *sherpa.EntryPointSummary, counter *int) bool {
	if summary == nil || len(summary.Counts) == 0 {
		return false
	}
	summary.Counts = append([]sherpa.EntryPointCount{}, summary.Counts[:len(summary.Counts)-1]...)
	(*counter)++
	return true
}

func applyPackageByteLimit(report PackageReport, maxBytes int) PackageReport {
	truncation := truncationValue(report.Truncated)
	enforceByteLimit(maxBytes, func(truncation Truncation) int {
		report.Truncated = reportTruncation(truncation)
		return encodedJSONLen(normalizePackageReport(report))
	}, &truncation,
		func() bool {
			return trimTestPlanItem(&report.TestPlan, &truncation.TestPlanItems)
		},
		func() bool {
			return trimLast(&report.SourceContexts, &truncation.SourceContexts)
		},
		func() bool {
			if !trimLast(&report.AffectedTests, &truncation.AffectedTests) {
				return false
			}
			report.ReadingOrder = packageReadingOrder(report)
			return true
		},
		func() bool {
			return trimLast(&report.AffectedImplementations, &truncation.AffectedImplementations)
		},
		func() bool {
			return trimLast(&report.AffectedInterfaces, &truncation.AffectedInterfaces)
		},
		func() bool {
			return trimLastPreserving(&report.TestCommands, 1, &truncation.TestCommands)
		},
		func() bool {
			return trimLastPreserving(&report.AffectedPackages, 1, &truncation.AffectedPackages)
		},
		func() bool {
			if !trimLastPreserving(&report.Symbols, 1, &truncation.Symbols) {
				return false
			}
			report.ReadingOrder = packageReadingOrder(report)
			return true
		},
		func() bool {
			if !trimLastPreserving(&report.Files, 1, &truncation.Files) {
				return false
			}
			report.ReadingOrder = packageReadingOrder(report)
			return true
		},
		func() bool {
			return trimLast(&report.ReadingOrder, &truncation.ReadingOrder)
		},
	)

	report.Truncated = reportTruncation(truncation)
	return report
}

func applyDiffByteLimit(report DiffReport, maxBytes int) DiffReport {
	truncation := truncationValue(report.Truncated)
	enforceByteLimit(maxBytes, func(truncation Truncation) int {
		report.Truncated = reportTruncation(truncation)
		return encodedJSONLen(normalizeDiffReport(report))
	}, &truncation,
		func() bool {
			return trimTestPlanItem(&report.TestPlan, &truncation.TestPlanItems)
		},
		func() bool {
			if !trimLast(&report.AffectedTests, &truncation.AffectedTests) {
				return false
			}
			report.ReadingOrder = diffReadingOrder(report)
			return true
		},
		func() bool {
			return trimLast(&report.AffectedSymbols, &truncation.AffectedSymbols)
		},
		func() bool {
			return trimLast(&report.AffectedImplementations, &truncation.AffectedImplementations)
		},
		func() bool {
			return trimLast(&report.AffectedInterfaces, &truncation.AffectedInterfaces)
		},
		func() bool {
			return trimEntryPointExample(report.EntryPointSummary, &truncation.EntryPointExamples)
		},
		func() bool {
			return trimEntryPointCount(report.EntryPointSummary, &truncation.EntryPointCounts)
		},
		func() bool {
			return trimLastPreserving(&report.TestCommands, 1, &truncation.TestCommands)
		},
		func() bool {
			return trimLastPreserving(&report.AffectedPackages, 1, &truncation.AffectedPackages)
		},
		func() bool {
			if !trimLastPreserving(&report.ChangedSymbolDetails, 1, &truncation.ChangedSymbolDetails) {
				return false
			}
			report.ReadingOrder = diffReadingOrder(report)
			return true
		},
		func() bool {
			if !trimLastPreserving(&report.ChangedFiles, 1, &truncation.ChangedFiles) {
				return false
			}
			report.ReadingOrder = diffReadingOrder(report)
			return true
		},
		func() bool {
			return trimLastPreserving(&report.ChangedPackages, 1, &truncation.ChangedPackages)
		},
		func() bool {
			return trimLast(&report.ReadingOrder, &truncation.ReadingOrder)
		},
	)

	report.Truncated = reportTruncation(truncation)
	return report
}

func enforceByteLimit(maxBytes int, size func(Truncation) int, truncation *Truncation, reducers ...func() bool) {
	if maxBytes <= 0 {
		return
	}

	for size(*truncation) > maxBytes {
		reduced := false
		for _, reduce := range reducers {
			if reduce() {
				reduced = true
				break
			}
		}
		if !reduced {
			break
		}
	}

	if overage := size(*truncation) - maxBytes; overage > 0 {
		truncation.ByteBudgetOverage = overage
		for {
			next := size(*truncation) - maxBytes
			if next <= 0 {
				truncation.ByteBudgetOverage = 0
				return
			}
			if next == truncation.ByteBudgetOverage {
				return
			}
			truncation.ByteBudgetOverage = next
		}
	}
}

func truncationValue(truncation *Truncation) Truncation {
	if truncation == nil {
		return Truncation{}
	}

	return *truncation
}

func encodedJSONLen(value any) int {
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}

	return len(data)
}

func trimLast[T any](values *[]T, omitted *int) bool {
	if len(*values) == 0 {
		return false
	}

	*values = append([]T{}, (*values)[:len(*values)-1]...)
	*omitted++
	return true
}

func trimLastPreserving[T any](values *[]T, min int, omitted *int) bool {
	if min < 0 {
		min = 0
	}
	if len(*values) <= min {
		return false
	}

	*values = append([]T{}, (*values)[:len(*values)-1]...)
	*omitted++
	return true
}

func trimSourceContextLine(context *sherpa.SourceContext, omitted *int) bool {
	if len(context.Lines) == 0 {
		return false
	}

	index := sourceContextLineTrimIndex(context.Lines)
	if index < 0 {
		return false
	}
	context.Lines = append(context.Lines[:index], context.Lines[index+1:]...)
	*omitted++
	return true
}

func sourceContextLineTrimIndex(lines []sherpa.SourceContextLine) int {
	targetIndex := -1
	for i, line := range lines {
		if line.Target {
			targetIndex = i
			break
		}
	}
	if targetIndex < 0 {
		return len(lines) - 1
	}

	bestIndex := -1
	bestDistance := -1
	for i, line := range lines {
		if line.Target {
			continue
		}

		distance := i - targetIndex
		if distance < 0 {
			distance = -distance
		}
		if distance > bestDistance || distance == bestDistance && i > bestIndex {
			bestDistance = distance
			bestIndex = i
		}
	}
	if bestIndex >= 0 {
		return bestIndex
	}

	return -1
}

func trimTestPlanItem(plan *sherpa.TestPlan, omitted *int) bool {
	if trimLast(&plan.Fallback, omitted) {
		return true
	}
	if trimLast(&plan.CallerPackages, omitted) {
		return true
	}
	if trimLast(&plan.Contracts, omitted) {
		return true
	}
	if trimLast(&plan.Related, omitted) {
		return true
	}
	if trimLast(&plan.Direct, omitted) {
		return true
	}

	return false
}
