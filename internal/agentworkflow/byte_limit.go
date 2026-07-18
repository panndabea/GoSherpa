package agentworkflow

import (
	"encoding/json"
	"sort"
	"strings"

	agentcontext "github.com/panndabea/GoSherpa/internal/agentcontext"
	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func applyAgentByteLimit(report Report, maxBytes int) Report {
	if maxBytes <= 0 {
		return report
	}

	report = normalizeReport(report)
	truncation := agentTruncationValue(report.Truncated)

	enforceAgentByteLimit(maxBytes, func(next agentcontext.Truncation) int {
		report.Truncated = agentTruncationPtr(next)
		report.SectionTruncation = normalizeSectionTruncation(report.SectionTruncation)
		return encodedAgentJSONLen(normalizeReport(report))
	}, &truncation,
		func() bool {
			if !trimLastAgent(&report.PossibleRuntimeRelationships.Examples) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "possibleRuntimeRelationships", "examples", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.ReadingOrder) {
				return false
			}
			truncation.ReadingOrder++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "context", "readingOrder", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.ChangedSymbolDetails, 1) {
				return false
			}
			truncation.ChangedSymbolDetails++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "context", "changedSymbolDetails", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.AffectedSymbols) {
				return false
			}
			truncation.AffectedSymbols++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "impact", "affectedSymbols", 1)
			return true
		},
		func() bool {
			if !trimTestPlanItemAgent(&report.TestPlan) {
				return false
			}
			truncation.TestPlanItems++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "tests", "testPlan", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.TestCommands, 1) {
				return false
			}
			truncation.TestCommands++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "tests", "testCommands", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.SuggestedCommands, 4) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "pr", "suggestedCommands", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.PossibleRuntimeRelationships.Counts) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "possibleRuntimeRelationships", "counts", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.Cost.RelationshipCounts) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "cost", "relationshipCounts", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.Cost.Limitations, 1) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "cost", "limitations", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.InterfaceSummary.AffectedImplementations) {
				return false
			}
			truncation.AffectedImplementations++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "interfaces", "affectedImplementations", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.InterfaceSummary.AffectedInterfaces) {
				return false
			}
			truncation.AffectedInterfaces++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "interfaces", "affectedInterfaces", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.AffectedPackages, 1) {
				return false
			}
			truncation.AffectedPackages++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "impact", "affectedPackages", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.ChangedFiles, 1) {
				return false
			}
			truncation.ChangedFiles++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "context", "changedFiles", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.ChangedPackages, 1) {
				return false
			}
			truncation.ChangedPackages++
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "context", "changedPackages", 1)
			return true
		},
		func() bool {
			if !trimSectionModeLimitation(&report.SectionModes) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "sectionModes", "limitations", 1)
			return true
		},
		func() bool {
			if !trimLastPreservingAgent(&report.Limitations, 4) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "workflow", "limitations", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.Readiness.RepoShapeWarnings) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "readiness", "repoShapeWarnings", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.Readiness.NestedModules) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "readiness", "nestedModules", 1)
			return true
		},
		func() bool {
			if !trimLastAgent(&report.Snapshot.StaleReasons) {
				return false
			}
			report.SectionTruncation = addSectionTruncation(report.SectionTruncation, "snapshot", "staleReasons", 1)
			return true
		},
	)

	report.Truncated = agentTruncationPtr(truncation)
	return normalizeReport(report)
}

func enforceAgentByteLimit(maxBytes int, size func(agentcontext.Truncation) int, truncation *agentcontext.Truncation, reducers ...func() bool) {
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

func encodedAgentJSONLen(value any) int {
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return len(data)
}

func trimLastAgent[T any](values *[]T) bool {
	if len(*values) == 0 {
		return false
	}
	*values = append([]T{}, (*values)[:len(*values)-1]...)
	return true
}

func trimLastPreservingAgent[T any](values *[]T, min int) bool {
	if min < 0 {
		min = 0
	}
	if len(*values) <= min {
		return false
	}
	*values = append([]T{}, (*values)[:len(*values)-1]...)
	return true
}

func trimTestPlanItemAgent(plan *sherpa.TestPlan) bool {
	if trimLastAgent(&plan.Fallback) {
		return true
	}
	if trimLastAgent(&plan.CallerPackages) {
		return true
	}
	if trimLastAgent(&plan.Contracts) {
		return true
	}
	if trimLastAgent(&plan.Related) {
		return true
	}
	return trimLastAgent(&plan.Direct)
}

func trimSectionModeLimitation(modes *[]SectionMode) bool {
	for index := len(*modes) - 1; index >= 0; index-- {
		if trimLastAgent(&(*modes)[index].Limitations) {
			return true
		}
	}
	return false
}

func sectionTruncationFromTruncation(truncation *agentcontext.Truncation) []SectionTruncation {
	if truncation == nil {
		return []SectionTruncation{}
	}

	var entries []SectionTruncation
	add := func(section string, field string, count int) {
		if count > 0 {
			entries = addSectionTruncation(entries, section, field, count)
		}
	}

	add("context", "changedFiles", truncation.ChangedFiles)
	add("context", "changedPackages", truncation.ChangedPackages)
	add("context", "changedSymbolDetails", truncation.ChangedSymbolDetails)
	add("context", "readingOrder", truncation.ReadingOrder)
	add("impact", "affectedPackages", truncation.AffectedPackages)
	add("impact", "affectedSymbols", truncation.AffectedSymbols)
	add("interfaces", "affectedInterfaces", truncation.AffectedInterfaces)
	add("interfaces", "affectedImplementations", truncation.AffectedImplementations)
	add("tests", "affectedTests", truncation.AffectedTests)
	add("tests", "testCommands", truncation.TestCommands)
	add("tests", "testPlan", truncation.TestPlanItems)

	return normalizeSectionTruncation(entries)
}

func addSectionTruncation(entries []SectionTruncation, section string, field string, omitted int) []SectionTruncation {
	section = strings.TrimSpace(section)
	field = strings.TrimSpace(field)
	if section == "" || field == "" || omitted <= 0 {
		return entries
	}
	for index := range entries {
		if entries[index].Section == section && entries[index].Field == field {
			entries[index].Omitted += omitted
			return entries
		}
	}
	return append(entries, SectionTruncation{Section: section, Field: field, Omitted: omitted})
}

func normalizeSectionTruncation(entries []SectionTruncation) []SectionTruncation {
	if entries == nil {
		return []SectionTruncation{}
	}

	var filtered []SectionTruncation
	for _, entry := range entries {
		entry.Section = strings.TrimSpace(entry.Section)
		entry.Field = strings.TrimSpace(entry.Field)
		if entry.Section == "" || entry.Field == "" || entry.Omitted <= 0 {
			continue
		}
		filtered = addSectionTruncation(filtered, entry.Section, entry.Field, entry.Omitted)
	}
	sort.Slice(filtered, func(i int, j int) bool {
		if filtered[i].Section != filtered[j].Section {
			return filtered[i].Section < filtered[j].Section
		}
		return filtered[i].Field < filtered[j].Field
	})
	if filtered == nil {
		return []SectionTruncation{}
	}
	return filtered
}

func agentTruncationValue(truncation *agentcontext.Truncation) agentcontext.Truncation {
	if truncation == nil {
		return agentcontext.Truncation{}
	}
	return *truncation
}

func agentTruncationPtr(truncation agentcontext.Truncation) *agentcontext.Truncation {
	if !agentTruncationActive(truncation) {
		return nil
	}
	return &truncation
}

func agentTruncationActive(truncation agentcontext.Truncation) bool {
	return truncation.Files > 0 ||
		truncation.Symbols > 0 ||
		truncation.SourceContexts > 0 ||
		truncation.SourceLines > 0 ||
		truncation.References > 0 ||
		truncation.Callers > 0 ||
		truncation.Callees > 0 ||
		truncation.RelatedTests > 0 ||
		truncation.AffectedTests > 0 ||
		truncation.TestCommands > 0 ||
		truncation.TestPlanItems > 0 ||
		truncation.ChangedFiles > 0 ||
		truncation.ChangedPackages > 0 ||
		truncation.AffectedPackages > 0 ||
		truncation.AffectedSymbols > 0 ||
		truncation.ChangedSymbolDetails > 0 ||
		truncation.AffectedInterfaces > 0 ||
		truncation.AffectedImplementations > 0 ||
		truncation.ReadingOrder > 0 ||
		truncation.ByteBudgetOverage > 0
}
