package symbolindex

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
)

type RelationshipKind string

const (
	RelationshipKindSymbolDefinition        RelationshipKind = "symbol-definition"
	RelationshipKindReference               RelationshipKind = "reference"
	RelationshipKindCall                    RelationshipKind = "call"
	RelationshipKindPossibleCall            RelationshipKind = "possible-call"
	RelationshipKindInterfaceImplementation RelationshipKind = "interface-implementation"
	RelationshipKindSatisfiedInterface      RelationshipKind = "satisfied-interface"
	RelationshipKindTestReference           RelationshipKind = "test-reference"
	RelationshipKindTestPlanSeed            RelationshipKind = "test-plan-seed"
	RelationshipKindPackageRelationship     RelationshipKind = "package-relationship"
)

type RelationshipCertainty string

const (
	RelationshipCertaintyDirect   RelationshipCertainty = "direct"
	RelationshipCertaintyPossible RelationshipCertainty = "possible"
	RelationshipCertaintyInferred RelationshipCertainty = "inferred"
)

type RelationshipIndex struct {
	References               []ReferenceRecord               `json:"references"`
	CallEdges                []CallEdgeRecord                `json:"callEdges"`
	PossibleCallEdges        []PossibleCallEdgeRecord        `json:"possibleCallEdges"`
	InterfaceImplementations []InterfaceImplementationRecord `json:"interfaceImplementations"`
	TestReferences           []TestReferenceRecord           `json:"testReferences"`
	PackageRelationships     []PackageRelationshipRecord     `json:"packageRelationships"`
}

type SymbolIdentity struct {
	Package       string              `json:"package,omitempty"`
	PackageName   string              `json:"packageName,omitempty"`
	Name          string              `json:"name,omitempty"`
	Receiver      string              `json:"receiver,omitempty"`
	QualifiedName string              `json:"qualifiedName,omitempty"`
	Kind          sherpa.SymbolKind   `json:"kind,omitempty"`
	Position      sherpa.Position     `json:"position,omitempty"`
	Range         *sherpa.SourceRange `json:"range,omitempty"`
}

type ReferenceRecord struct {
	Kind          RelationshipKind      `json:"kind"`
	Package       string                `json:"package,omitempty"`
	File          string                `json:"file,omitempty"`
	Source        SymbolIdentity        `json:"source,omitempty"`
	Target        SymbolIdentity        `json:"target"`
	ReferenceKind sherpa.ReferenceKind  `json:"referenceKind,omitempty"`
	Certainty     RelationshipCertainty `json:"certainty"`
	AnalysisMode  string                `json:"analysisMode,omitempty"`
	Position      sherpa.Position       `json:"position"`
	Range         *sherpa.SourceRange   `json:"range,omitempty"`
	Limitations   []string              `json:"limitations"`
}

type CallEdgeRecord struct {
	Kind         RelationshipKind      `json:"kind"`
	Package      string                `json:"package,omitempty"`
	File         string                `json:"file,omitempty"`
	Source       SymbolIdentity        `json:"source"`
	Target       SymbolIdentity        `json:"target"`
	CallScope    sherpa.CallScope      `json:"callScope,omitempty"`
	Certainty    RelationshipCertainty `json:"certainty"`
	AnalysisMode string                `json:"analysisMode,omitempty"`
	Position     sherpa.Position       `json:"position"`
	Range        *sherpa.SourceRange   `json:"range,omitempty"`
	Limitations  []string              `json:"limitations"`
}

type PossibleCallEdgeRecord struct {
	Kind         RelationshipKind      `json:"kind"`
	Package      string                `json:"package,omitempty"`
	File         string                `json:"file,omitempty"`
	Source       SymbolIdentity        `json:"source"`
	Target       SymbolIdentity        `json:"target"`
	CallScope    sherpa.CallScope      `json:"callScope,omitempty"`
	Certainty    RelationshipCertainty `json:"certainty"`
	Reason       string                `json:"reason,omitempty"`
	AnalysisMode string                `json:"analysisMode,omitempty"`
	Position     sherpa.Position       `json:"position"`
	Range        *sherpa.SourceRange   `json:"range,omitempty"`
	Limitations  []string              `json:"limitations"`
}

type InterfaceImplementationRecord struct {
	Kind           RelationshipKind      `json:"kind"`
	Package        string                `json:"package,omitempty"`
	File           string                `json:"file,omitempty"`
	Interface      SymbolIdentity        `json:"interface"`
	Implementation SymbolIdentity        `json:"implementation"`
	Certainty      RelationshipCertainty `json:"certainty"`
	AnalysisMode   string                `json:"analysisMode,omitempty"`
	Position       sherpa.Position       `json:"position"`
	Range          *sherpa.SourceRange   `json:"range,omitempty"`
	Limitations    []string              `json:"limitations"`
}

type TestReferenceRecord struct {
	Kind         RelationshipKind      `json:"kind"`
	Package      string                `json:"package,omitempty"`
	File         string                `json:"file,omitempty"`
	Test         SymbolIdentity        `json:"test"`
	Target       SymbolIdentity        `json:"target"`
	TestName     string                `json:"testName,omitempty"`
	Reasons      []string              `json:"reasons"`
	Certainty    RelationshipCertainty `json:"certainty"`
	AnalysisMode string                `json:"analysisMode,omitempty"`
	Position     sherpa.Position       `json:"position"`
	Range        *sherpa.SourceRange   `json:"range,omitempty"`
	Limitations  []string              `json:"limitations"`
}

type PackageRelationshipRecord struct {
	Kind           RelationshipKind      `json:"kind"`
	Package        string                `json:"package"`
	RelatedPackage string                `json:"relatedPackage"`
	Certainty      RelationshipCertainty `json:"certainty"`
	AnalysisMode   string                `json:"analysisMode,omitempty"`
	Reasons        []string              `json:"reasons"`
	Limitations    []string              `json:"limitations"`
}

func NewRelationshipIndex() RelationshipIndex {
	return RelationshipIndex{
		References:               []ReferenceRecord{},
		CallEdges:                []CallEdgeRecord{},
		PossibleCallEdges:        []PossibleCallEdgeRecord{},
		InterfaceImplementations: []InterfaceImplementationRecord{},
		TestReferences:           []TestReferenceRecord{},
		PackageRelationships:     []PackageRelationshipRecord{},
	}
}

func NormalizeRelationshipIndex(root string, relationships RelationshipIndex) RelationshipIndex {
	relationships.References = normalizeReferenceRecords(root, relationships.References)
	relationships.CallEdges = normalizeCallEdgeRecords(root, relationships.CallEdges)
	relationships.PossibleCallEdges = normalizePossibleCallEdgeRecords(root, relationships.PossibleCallEdges)
	relationships.InterfaceImplementations = normalizeInterfaceImplementationRecords(root, relationships.InterfaceImplementations)
	relationships.TestReferences = normalizeTestReferenceRecords(root, relationships.TestReferences)
	relationships.PackageRelationships = normalizePackageRelationshipRecords(relationships.PackageRelationships)

	return relationships
}

func normalizeReferenceRecords(root string, records []ReferenceRecord) []ReferenceRecord {
	normalized := make([]ReferenceRecord, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		record.Kind = defaultRelationshipKind(record.Kind, RelationshipKindReference)
		record.Certainty = defaultRelationshipCertainty(record.Certainty, RelationshipCertaintyDirect)
		record.Package = strings.TrimSpace(record.Package)
		record.File = normalizeRelationshipPath(root, record.File)
		record.Source = normalizeSymbolIdentity(root, record.Source)
		record.Target = normalizeSymbolIdentity(root, record.Target)
		record.AnalysisMode = strings.TrimSpace(record.AnalysisMode)
		record.Position = normalizeRelationshipPosition(root, record.Position)
		record.Range = normalizeRelationshipRange(root, record.Range)
		record.Limitations = uniqueSortedStrings(record.Limitations)
		key := referenceRecordKey(record)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, record)
	}
	sort.SliceStable(normalized, func(i int, j int) bool {
		return referenceRecordKey(normalized[i]) < referenceRecordKey(normalized[j])
	})
	return normalized
}

func normalizeCallEdgeRecords(root string, records []CallEdgeRecord) []CallEdgeRecord {
	normalized := make([]CallEdgeRecord, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		record.Kind = defaultRelationshipKind(record.Kind, RelationshipKindCall)
		record.Certainty = defaultRelationshipCertainty(record.Certainty, RelationshipCertaintyDirect)
		record.Package = strings.TrimSpace(record.Package)
		record.File = normalizeRelationshipPath(root, record.File)
		record.Source = normalizeSymbolIdentity(root, record.Source)
		record.Target = normalizeSymbolIdentity(root, record.Target)
		record.AnalysisMode = strings.TrimSpace(record.AnalysisMode)
		record.Position = normalizeRelationshipPosition(root, record.Position)
		record.Range = normalizeRelationshipRange(root, record.Range)
		record.Limitations = uniqueSortedStrings(record.Limitations)
		key := callEdgeRecordKey(record)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, record)
	}
	sort.SliceStable(normalized, func(i int, j int) bool {
		return callEdgeRecordKey(normalized[i]) < callEdgeRecordKey(normalized[j])
	})
	return normalized
}

func normalizePossibleCallEdgeRecords(root string, records []PossibleCallEdgeRecord) []PossibleCallEdgeRecord {
	normalized := make([]PossibleCallEdgeRecord, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		record.Kind = defaultRelationshipKind(record.Kind, RelationshipKindPossibleCall)
		record.Certainty = defaultRelationshipCertainty(record.Certainty, RelationshipCertaintyPossible)
		record.Package = strings.TrimSpace(record.Package)
		record.File = normalizeRelationshipPath(root, record.File)
		record.Source = normalizeSymbolIdentity(root, record.Source)
		record.Target = normalizeSymbolIdentity(root, record.Target)
		record.Reason = strings.TrimSpace(record.Reason)
		record.AnalysisMode = strings.TrimSpace(record.AnalysisMode)
		record.Position = normalizeRelationshipPosition(root, record.Position)
		record.Range = normalizeRelationshipRange(root, record.Range)
		record.Limitations = uniqueSortedStrings(record.Limitations)
		key := possibleCallEdgeRecordKey(record)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, record)
	}
	sort.SliceStable(normalized, func(i int, j int) bool {
		return possibleCallEdgeRecordKey(normalized[i]) < possibleCallEdgeRecordKey(normalized[j])
	})
	return normalized
}

func normalizeInterfaceImplementationRecords(root string, records []InterfaceImplementationRecord) []InterfaceImplementationRecord {
	normalized := make([]InterfaceImplementationRecord, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		record.Kind = defaultRelationshipKind(record.Kind, RelationshipKindInterfaceImplementation)
		record.Certainty = defaultRelationshipCertainty(record.Certainty, RelationshipCertaintyDirect)
		record.Package = strings.TrimSpace(record.Package)
		record.File = normalizeRelationshipPath(root, record.File)
		record.Interface = normalizeSymbolIdentity(root, record.Interface)
		record.Implementation = normalizeSymbolIdentity(root, record.Implementation)
		record.AnalysisMode = strings.TrimSpace(record.AnalysisMode)
		record.Position = normalizeRelationshipPosition(root, record.Position)
		record.Range = normalizeRelationshipRange(root, record.Range)
		record.Limitations = uniqueSortedStrings(record.Limitations)
		key := interfaceImplementationRecordKey(record)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, record)
	}
	sort.SliceStable(normalized, func(i int, j int) bool {
		return interfaceImplementationRecordKey(normalized[i]) < interfaceImplementationRecordKey(normalized[j])
	})
	return normalized
}

func normalizeTestReferenceRecords(root string, records []TestReferenceRecord) []TestReferenceRecord {
	normalized := make([]TestReferenceRecord, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		record.Kind = defaultRelationshipKind(record.Kind, RelationshipKindTestReference)
		record.Certainty = defaultRelationshipCertainty(record.Certainty, RelationshipCertaintyDirect)
		record.Package = strings.TrimSpace(record.Package)
		record.File = normalizeRelationshipPath(root, record.File)
		record.Test = normalizeSymbolIdentity(root, record.Test)
		record.Target = normalizeSymbolIdentity(root, record.Target)
		record.TestName = strings.TrimSpace(record.TestName)
		record.Reasons = uniqueSortedStrings(record.Reasons)
		record.AnalysisMode = strings.TrimSpace(record.AnalysisMode)
		record.Position = normalizeRelationshipPosition(root, record.Position)
		record.Range = normalizeRelationshipRange(root, record.Range)
		record.Limitations = uniqueSortedStrings(record.Limitations)
		key := testReferenceRecordKey(record)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, record)
	}
	sort.SliceStable(normalized, func(i int, j int) bool {
		return testReferenceRecordKey(normalized[i]) < testReferenceRecordKey(normalized[j])
	})
	return normalized
}

func normalizePackageRelationshipRecords(records []PackageRelationshipRecord) []PackageRelationshipRecord {
	normalized := make([]PackageRelationshipRecord, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		record.Kind = defaultRelationshipKind(record.Kind, RelationshipKindPackageRelationship)
		record.Certainty = defaultRelationshipCertainty(record.Certainty, RelationshipCertaintyDirect)
		record.Package = strings.TrimSpace(record.Package)
		record.RelatedPackage = strings.TrimSpace(record.RelatedPackage)
		record.AnalysisMode = strings.TrimSpace(record.AnalysisMode)
		record.Reasons = uniqueSortedStrings(record.Reasons)
		record.Limitations = uniqueSortedStrings(record.Limitations)
		key := packageRelationshipRecordKey(record)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, record)
	}
	sort.SliceStable(normalized, func(i int, j int) bool {
		return packageRelationshipRecordKey(normalized[i]) < packageRelationshipRecordKey(normalized[j])
	})
	return normalized
}

func normalizeSymbolIdentity(root string, identity SymbolIdentity) SymbolIdentity {
	identity.Package = strings.TrimSpace(identity.Package)
	identity.PackageName = strings.TrimSpace(identity.PackageName)
	identity.Name = strings.TrimSpace(identity.Name)
	identity.Receiver = strings.TrimSpace(identity.Receiver)
	identity.QualifiedName = strings.TrimSpace(identity.QualifiedName)
	identity.Position = normalizeRelationshipPosition(root, identity.Position)
	identity.Range = normalizeRelationshipRange(root, identity.Range)

	return identity
}

func normalizeRelationshipPosition(root string, position sherpa.Position) sherpa.Position {
	position.File = normalizeRelationshipPath(root, position.File)
	return position
}

func normalizeRelationshipRange(root string, sourceRange *sherpa.SourceRange) *sherpa.SourceRange {
	if sourceRange == nil {
		return nil
	}

	normalized := *sourceRange
	normalized.Start = normalizeRelationshipPosition(root, normalized.Start)
	normalized.End = normalizeRelationshipPosition(root, normalized.End)
	return &normalized
}

func normalizeRelationshipPath(root string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	cleaned := filepath.Clean(value)
	root = strings.TrimSpace(root)
	if filepath.IsAbs(cleaned) && root != "" {
		if rootPath, err := filepath.Abs(root); err == nil {
			if relative, ok := relativePath(rootPath, cleaned); ok {
				return relative
			}
		}
	}

	return filepath.ToSlash(cleaned)
}

func relativePath(root string, value string) (string, bool) {
	relative, err := filepath.Rel(root, value)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(relative), true
}

func defaultRelationshipKind(value RelationshipKind, fallback RelationshipKind) RelationshipKind {
	if strings.TrimSpace(string(value)) == "" {
		return fallback
	}

	return value
}

func defaultRelationshipCertainty(value RelationshipCertainty, fallback RelationshipCertainty) RelationshipCertainty {
	if strings.TrimSpace(string(value)) == "" {
		return fallback
	}

	return value
}

func referenceRecordKey(record ReferenceRecord) string {
	return relationshipKey(
		string(record.Kind),
		record.Package,
		record.File,
		symbolIdentityKey(record.Source),
		symbolIdentityKey(record.Target),
		string(record.ReferenceKind),
		string(record.Certainty),
		record.AnalysisMode,
		positionKey(record.Position),
		rangeKey(record.Range),
		strings.Join(record.Limitations, "\x00"),
	)
}

func callEdgeRecordKey(record CallEdgeRecord) string {
	return relationshipKey(
		string(record.Kind),
		record.Package,
		record.File,
		symbolIdentityKey(record.Source),
		symbolIdentityKey(record.Target),
		string(record.CallScope),
		string(record.Certainty),
		record.AnalysisMode,
		positionKey(record.Position),
		rangeKey(record.Range),
		strings.Join(record.Limitations, "\x00"),
	)
}

func possibleCallEdgeRecordKey(record PossibleCallEdgeRecord) string {
	return relationshipKey(
		string(record.Kind),
		record.Package,
		record.File,
		symbolIdentityKey(record.Source),
		symbolIdentityKey(record.Target),
		string(record.CallScope),
		string(record.Certainty),
		record.Reason,
		record.AnalysisMode,
		positionKey(record.Position),
		rangeKey(record.Range),
		strings.Join(record.Limitations, "\x00"),
	)
}

func interfaceImplementationRecordKey(record InterfaceImplementationRecord) string {
	return relationshipKey(
		string(record.Kind),
		record.Package,
		record.File,
		symbolIdentityKey(record.Interface),
		symbolIdentityKey(record.Implementation),
		string(record.Certainty),
		record.AnalysisMode,
		positionKey(record.Position),
		rangeKey(record.Range),
		strings.Join(record.Limitations, "\x00"),
	)
}

func testReferenceRecordKey(record TestReferenceRecord) string {
	return relationshipKey(
		string(record.Kind),
		record.Package,
		record.File,
		symbolIdentityKey(record.Test),
		symbolIdentityKey(record.Target),
		record.TestName,
		strings.Join(record.Reasons, "\x00"),
		string(record.Certainty),
		record.AnalysisMode,
		positionKey(record.Position),
		rangeKey(record.Range),
		strings.Join(record.Limitations, "\x00"),
	)
}

func packageRelationshipRecordKey(record PackageRelationshipRecord) string {
	return relationshipKey(
		string(record.Kind),
		record.Package,
		record.RelatedPackage,
		string(record.Certainty),
		record.AnalysisMode,
		strings.Join(record.Reasons, "\x00"),
		strings.Join(record.Limitations, "\x00"),
	)
}

func symbolIdentityKey(identity SymbolIdentity) string {
	return relationshipKey(
		identity.Package,
		identity.PackageName,
		identity.Name,
		identity.Receiver,
		identity.QualifiedName,
		string(identity.Kind),
		positionKey(identity.Position),
		rangeKey(identity.Range),
	)
}

func positionKey(position sherpa.Position) string {
	return relationshipKey(position.File, fmt.Sprintf("%08d", position.Line), fmt.Sprintf("%08d", position.Column))
}

func rangeKey(sourceRange *sherpa.SourceRange) string {
	if sourceRange == nil {
		return ""
	}

	return relationshipKey(positionKey(sourceRange.Start), positionKey(sourceRange.End))
}

func relationshipKey(parts ...string) string {
	return strings.Join(parts, "\x00")
}
