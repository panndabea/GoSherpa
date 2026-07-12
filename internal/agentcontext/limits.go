package agentcontext

import "fmt"

type LimitOptions struct {
	MaxFiles      int  `json:"maxFiles,omitempty"`
	MaxReferences int  `json:"maxReferences,omitempty"`
	MaxSymbols    int  `json:"maxSymbols,omitempty"`
	MaxTests      int  `json:"maxTests,omitempty"`
	MaxBytes      int  `json:"maxBytes,omitempty"`
	SourceRadius  *int `json:"sourceRadius,omitempty"`
}

const (
	DefaultSymbolMaxReferences = 25
	DefaultFileMaxSymbols      = 40
	DefaultPackageMaxFiles     = 40
	DefaultPackageMaxSymbols   = 80
	DefaultDiffMaxFiles        = 60
	DefaultDiffMaxSymbols      = 80
	DefaultMaxTests            = 25
	DefaultMaxBytes            = 24576
)

type Truncation struct {
	Files                   int `json:"files,omitempty"`
	Symbols                 int `json:"symbols,omitempty"`
	SourceContexts          int `json:"sourceContexts,omitempty"`
	SourceLines             int `json:"sourceLines,omitempty"`
	References              int `json:"references,omitempty"`
	Callers                 int `json:"callers,omitempty"`
	Callees                 int `json:"callees,omitempty"`
	RelatedTests            int `json:"relatedTests,omitempty"`
	AffectedTests           int `json:"affectedTests,omitempty"`
	TestCommands            int `json:"testCommands,omitempty"`
	TestPlanItems           int `json:"testPlanItems,omitempty"`
	ChangedFiles            int `json:"changedFiles,omitempty"`
	ChangedPackages         int `json:"changedPackages,omitempty"`
	AffectedPackages        int `json:"affectedPackages,omitempty"`
	AffectedSymbols         int `json:"affectedSymbols,omitempty"`
	ChangedSymbolDetails    int `json:"changedSymbolDetails,omitempty"`
	AffectedInterfaces      int `json:"affectedInterfaces,omitempty"`
	AffectedImplementations int `json:"affectedImplementations,omitempty"`
	ReadingOrder            int `json:"readingOrder,omitempty"`
	ByteBudgetOverage       int `json:"byteBudgetOverage,omitempty"`
}

func NewSourceRadius(value int) *int {
	return &value
}

func normalizeSymbolLimits(sourceRadius int, limits LimitOptions) LimitOptions {
	limits = normalizeSourceRadius(sourceRadius, limits)
	limits.MaxReferences = defaultPositiveLimit(limits.MaxReferences, DefaultSymbolMaxReferences)
	limits.MaxTests = defaultPositiveLimit(limits.MaxTests, DefaultMaxTests)
	limits.MaxBytes = defaultPositiveLimit(limits.MaxBytes, DefaultMaxBytes)

	return limits
}

func normalizeFileLimits(sourceRadius int, limits LimitOptions) LimitOptions {
	limits = normalizeSourceRadius(sourceRadius, limits)
	limits.MaxSymbols = defaultPositiveLimit(limits.MaxSymbols, DefaultFileMaxSymbols)
	limits.MaxTests = defaultPositiveLimit(limits.MaxTests, DefaultMaxTests)
	limits.MaxBytes = defaultPositiveLimit(limits.MaxBytes, DefaultMaxBytes)

	return limits
}

func normalizePackageLimits(sourceRadius int, limits LimitOptions) LimitOptions {
	limits = normalizeSourceRadius(sourceRadius, limits)
	limits.MaxFiles = defaultPositiveLimit(limits.MaxFiles, DefaultPackageMaxFiles)
	limits.MaxSymbols = defaultPositiveLimit(limits.MaxSymbols, DefaultPackageMaxSymbols)
	limits.MaxTests = defaultPositiveLimit(limits.MaxTests, DefaultMaxTests)
	limits.MaxBytes = defaultPositiveLimit(limits.MaxBytes, DefaultMaxBytes)

	return limits
}

func normalizeDiffLimits(limits LimitOptions) LimitOptions {
	limits.MaxFiles = defaultPositiveLimit(limits.MaxFiles, DefaultDiffMaxFiles)
	limits.MaxSymbols = defaultPositiveLimit(limits.MaxSymbols, DefaultDiffMaxSymbols)
	limits.MaxTests = defaultPositiveLimit(limits.MaxTests, DefaultMaxTests)
	limits.MaxBytes = defaultPositiveLimit(limits.MaxBytes, DefaultMaxBytes)

	return limits
}

func normalizeSourceRadius(sourceRadius int, limits LimitOptions) LimitOptions {
	if limits.SourceRadius == nil && sourceRadius > 0 {
		limits.SourceRadius = NewSourceRadius(sourceRadius)
	}

	return limits
}

func defaultPositiveLimit(value int, fallback int) int {
	if value > 0 {
		return value
	}

	return fallback
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
		limits.MaxBytes > 0 ||
		limits.SourceRadius != nil
}

func truncationActive(truncation Truncation) bool {
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
	appendMessage("source lines", truncation.SourceLines)
	appendMessage("references", truncation.References)
	appendMessage("callers", truncation.Callers)
	appendMessage("callees", truncation.Callees)
	appendMessage("related tests", truncation.RelatedTests)
	appendMessage("affected tests", truncation.AffectedTests)
	appendMessage("test commands", truncation.TestCommands)
	appendMessage("test plan items", truncation.TestPlanItems)
	appendMessage("changed files", truncation.ChangedFiles)
	appendMessage("changed packages", truncation.ChangedPackages)
	appendMessage("affected packages", truncation.AffectedPackages)
	appendMessage("affected symbols", truncation.AffectedSymbols)
	appendMessage("changed symbol details", truncation.ChangedSymbolDetails)
	appendMessage("affected interfaces", truncation.AffectedInterfaces)
	appendMessage("affected implementations", truncation.AffectedImplementations)
	appendMessage("reading order", truncation.ReadingOrder)
	if truncation.ByteBudgetOverage > 0 {
		messages = append(messages, fmt.Sprintf("byte budget overage: %d bytes", truncation.ByteBudgetOverage))
	}

	return messages
}
