package agentcontext

import "fmt"

type LimitOptions struct {
	MaxFiles      int  `json:"maxFiles,omitempty"`
	MaxReferences int  `json:"maxReferences,omitempty"`
	MaxSymbols    int  `json:"maxSymbols,omitempty"`
	MaxTests      int  `json:"maxTests,omitempty"`
	SourceRadius  *int `json:"sourceRadius,omitempty"`
}

type Truncation struct {
	Files                   int `json:"files,omitempty"`
	Symbols                 int `json:"symbols,omitempty"`
	SourceContexts          int `json:"sourceContexts,omitempty"`
	References              int `json:"references,omitempty"`
	Callers                 int `json:"callers,omitempty"`
	Callees                 int `json:"callees,omitempty"`
	RelatedTests            int `json:"relatedTests,omitempty"`
	AffectedTests           int `json:"affectedTests,omitempty"`
	ChangedFiles            int `json:"changedFiles,omitempty"`
	ChangedPackages         int `json:"changedPackages,omitempty"`
	AffectedPackages        int `json:"affectedPackages,omitempty"`
	AffectedSymbols         int `json:"affectedSymbols,omitempty"`
	AffectedInterfaces      int `json:"affectedInterfaces,omitempty"`
	AffectedImplementations int `json:"affectedImplementations,omitempty"`
	ReadingOrder            int `json:"readingOrder,omitempty"`
}

func NewSourceRadius(value int) *int {
	return &value
}

func normalizeLimits(sourceRadius int, limits LimitOptions) LimitOptions {
	if limits.SourceRadius == nil && sourceRadius > 0 {
		limits.SourceRadius = NewSourceRadius(sourceRadius)
	}

	return limits
}

func sourceRadiusOrDefault(limits LimitOptions, fallback int) int {
	if limits.SourceRadius != nil {
		return *limits.SourceRadius
	}

	return fallback
}

func limitOptionsActive(limits LimitOptions) bool {
	return limits.MaxFiles > 0 ||
		limits.MaxReferences > 0 ||
		limits.MaxSymbols > 0 ||
		limits.MaxTests > 0 ||
		limits.SourceRadius != nil
}

func truncationActive(truncation Truncation) bool {
	return truncation.Files > 0 ||
		truncation.Symbols > 0 ||
		truncation.SourceContexts > 0 ||
		truncation.References > 0 ||
		truncation.Callers > 0 ||
		truncation.Callees > 0 ||
		truncation.RelatedTests > 0 ||
		truncation.AffectedTests > 0 ||
		truncation.ChangedFiles > 0 ||
		truncation.ChangedPackages > 0 ||
		truncation.AffectedPackages > 0 ||
		truncation.AffectedSymbols > 0 ||
		truncation.AffectedInterfaces > 0 ||
		truncation.AffectedImplementations > 0 ||
		truncation.ReadingOrder > 0
}

func limitSlice[T any](values []T, max int) ([]T, int) {
	if max <= 0 || len(values) <= max {
		return values, 0
	}

	return append([]T{}, values[:max]...), len(values) - max
}

func reportLimits(limits LimitOptions) *LimitOptions {
	if !limitOptionsActive(limits) {
		return nil
	}

	return &limits
}

func reportTruncation(truncation Truncation) *Truncation {
	if !truncationActive(truncation) {
		return nil
	}

	return &truncation
}

func truncationMessages(truncation *Truncation) []string {
	if truncation == nil {
		return nil
	}

	var messages []string
	appendMessage := func(label string, count int) {
		if count <= 0 {
			return
		}

		messages = append(messages, fmt.Sprintf("%s: %d omitted", label, count))
	}

	appendMessage("files", truncation.Files)
	appendMessage("symbols", truncation.Symbols)
	appendMessage("source contexts", truncation.SourceContexts)
	appendMessage("references", truncation.References)
	appendMessage("callers", truncation.Callers)
	appendMessage("callees", truncation.Callees)
	appendMessage("related tests", truncation.RelatedTests)
	appendMessage("affected tests", truncation.AffectedTests)
	appendMessage("changed files", truncation.ChangedFiles)
	appendMessage("changed packages", truncation.ChangedPackages)
	appendMessage("affected packages", truncation.AffectedPackages)
	appendMessage("affected symbols", truncation.AffectedSymbols)
	appendMessage("affected interfaces", truncation.AffectedInterfaces)
	appendMessage("affected implementations", truncation.AffectedImplementations)
	appendMessage("reading order", truncation.ReadingOrder)

	return messages
}
