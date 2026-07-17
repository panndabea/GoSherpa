package sherpa

const (
	TargetRiskLevelLow    = RiskLevelLow
	TargetRiskLevelMedium = RiskLevelMedium
	TargetRiskLevelHigh   = RiskLevelHigh

	TargetRiskScopeLocal             = "local"
	TargetRiskScopePackage           = "package"
	TargetRiskScopeCrossPackage      = "cross-package"
	TargetRiskScopeExportedAPI       = "exported-api"
	TargetRiskScopeInterfaceContract = "interface-contract"

	TargetRiskReasonAffectedPackages     = "affected-packages"
	TargetRiskReasonDirectReferences     = "direct-references"
	TargetRiskReasonTransitiveCallers    = "transitive-callers"
	TargetRiskReasonExportedSymbol       = "exported-symbol"
	TargetRiskReasonExportedTypeMethod   = "exported-type-method"
	TargetRiskReasonInterfaceContract    = "interface-contract"
	TargetRiskReasonPackageFanIn         = "package-fan-in"
	TargetRiskReasonMissingDirectTests   = "missing-direct-tests"
	TargetRiskReasonFallbackTests        = "fallback-tests"
	TargetRiskReasonPossibleRuntimeCalls = "possible-runtime-calls"
	TargetRiskReasonNonGoOrHunkOnlyDiff  = "non-go-or-hunk-only-diff"
	TargetRiskReasonSnapshotFallback     = "snapshot-fallback"
	TargetRiskReasonAnalysisWarning      = "analysis-warning"
)

type TargetRiskSummary struct {
	Level       string            `json:"level"`
	Scope       string            `json:"scope"`
	Reasons     []string          `json:"reasons"`
	Signals     TargetRiskSignals `json:"signals"`
	Limitations []string          `json:"limitations"`
}

type TargetRiskSignals struct {
	AffectedPackages     int  `json:"affectedPackages"`
	DirectReferences     int  `json:"directReferences"`
	TransitiveCallers    int  `json:"transitiveCallers"`
	CallerPackages       int  `json:"callerPackages"`
	ExportedSymbol       bool `json:"exportedSymbol"`
	ExportedTypeMethod   bool `json:"exportedTypeMethod"`
	InterfaceContracts   int  `json:"interfaceContracts"`
	PackageFanIn         int  `json:"packageFanIn"`
	MissingDirectTests   bool `json:"missingDirectTests"`
	FallbackTests        bool `json:"fallbackTests"`
	PossibleRuntimeCalls int  `json:"possibleRuntimeCalls"`
	Warnings             int  `json:"warnings"`
	NonGoOrHunkOnlyDiff  bool `json:"nonGoOrHunkOnlyDiff"`
	SnapshotFallback     bool `json:"snapshotFallback"`
}

func NormalizeTargetRiskSummary(summary TargetRiskSummary) TargetRiskSummary {
	if summary.Level == "" {
		summary.Level = TargetRiskLevelLow
	}
	if summary.Scope == "" {
		summary.Scope = TargetRiskScopeLocal
	}
	summary.Reasons = uniqueSorted(summary.Reasons)
	if summary.Reasons == nil {
		summary.Reasons = []string{}
	}
	summary.Limitations = uniqueSorted(summary.Limitations)
	if summary.Limitations == nil {
		summary.Limitations = []string{}
	}

	return summary
}

func TargetRiskLevelForScore(score int) string {
	switch {
	case score >= 6:
		return TargetRiskLevelHigh
	case score >= 3:
		return TargetRiskLevelMedium
	default:
		return TargetRiskLevelLow
	}
}
