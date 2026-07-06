package agentcontext

import (
	"fmt"
	"strings"

	"github.com/panndabea/GoSherpa/internal/sherpa"
)

func Format(report Report) string {
	var builder strings.Builder

	builder.WriteString("CONTEXT\n")
	builder.WriteString("\n")
	writeTarget(&builder, report)
	builder.WriteString("\n")
	writeDefinition(&builder, report.Symbol)
	builder.WriteString("\n")
	writeSource(&builder, report.SourceContext)
	builder.WriteString("\n")
	writePurpose(&builder, report.Purpose)
	builder.WriteString("\n")
	writeAnalysis(&builder, report)
	writeTruncation(&builder, report.Truncated)
	builder.WriteString("\n")
	writeCallers(&builder, report.Callers)
	builder.WriteString("\n")
	writeCallees(&builder, report.Callees)
	builder.WriteString("\n")
	writeReferences(&builder, report.References)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED INTERFACES", report.AffectedInterfaces)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED IMPLEMENTATIONS", report.AffectedImplementations)
	builder.WriteString("\n")
	writeRelatedTests(&builder, report.RelatedTests)
	builder.WriteString("\n")
	sherpa.WriteTestPlan(&builder, report.TestPlan, report.TestCommands)
	builder.WriteString("\n")
	writeValues(&builder, "LIMITATIONS", report.Limitations)
	writeWarnings(&builder, report.Warnings)

	return builder.String()
}

func FormatFile(report FileReport) string {
	var builder strings.Builder

	builder.WriteString("CONTEXT FILE\n")
	builder.WriteString("\n")
	writeFileTarget(&builder, report)
	builder.WriteString("\n")
	writePurpose(&builder, report.Purpose)
	builder.WriteString("\n")
	writeFileAnalysis(&builder, report)
	writeTruncation(&builder, report.Truncated)
	builder.WriteString("\n")
	writeFileSymbols(&builder, report.Symbols)
	builder.WriteString("\n")
	writeFileSourceContexts(&builder, report.Symbols, report.SourceContexts)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED INTERFACES", report.AffectedInterfaces)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED IMPLEMENTATIONS", report.AffectedImplementations)
	builder.WriteString("\n")
	writeAffectedTests(&builder, report.AffectedTests)
	builder.WriteString("\n")
	sherpa.WriteTestPlan(&builder, report.TestPlan, report.TestCommands)
	builder.WriteString("\n")
	writeReadingOrder(&builder, report.ReadingOrder)
	builder.WriteString("\n")
	writeValues(&builder, "LIMITATIONS", report.Limitations)
	writeWarnings(&builder, report.Warnings)

	return builder.String()
}

func FormatPackage(report PackageReport) string {
	var builder strings.Builder

	builder.WriteString("CONTEXT PACKAGE\n")
	builder.WriteString("\n")
	writePackageTarget(&builder, report)
	builder.WriteString("\n")
	writePurpose(&builder, report.Purpose)
	builder.WriteString("\n")
	writePackageAnalysis(&builder, report)
	writeTruncation(&builder, report.Truncated)
	builder.WriteString("\n")
	writeValues(&builder, "PACKAGE FILES", report.Files)
	builder.WriteString("\n")
	writePackageSymbols(&builder, report.Symbols)
	builder.WriteString("\n")
	writeFileSourceContexts(&builder, report.Symbols, report.SourceContexts)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED PACKAGES", report.AffectedPackages)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED INTERFACES", report.AffectedInterfaces)
	builder.WriteString("\n")
	writeValues(&builder, "AFFECTED IMPLEMENTATIONS", report.AffectedImplementations)
	builder.WriteString("\n")
	writeAffectedTests(&builder, report.AffectedTests)
	builder.WriteString("\n")
	sherpa.WriteTestPlan(&builder, report.TestPlan, report.TestCommands)
	builder.WriteString("\n")
	writeReadingOrder(&builder, report.ReadingOrder)
	builder.WriteString("\n")
	writeValues(&builder, "LIMITATIONS", report.Limitations)
	writeWarnings(&builder, report.Warnings)

	return builder.String()
}

func writeTarget(builder *strings.Builder, report Report) {
	builder.WriteString("TARGET\n")
	fmt.Fprintf(builder, "  %s (%s)\n", report.Identity.Target, report.Identity.Kind)
	if report.Identity.Package != "" {
		fmt.Fprintf(builder, "  Package: %s\n", report.Identity.Package)
	}
	if report.Identity.QualifiedName != "" {
		fmt.Fprintf(builder, "  Qualified: %s\n", report.Identity.QualifiedName)
	}
	if report.Identity.Signature != "" {
		fmt.Fprintf(builder, "  Signature: %s\n", report.Identity.Signature)
	}
}

func writeFileTarget(builder *strings.Builder, report FileReport) {
	builder.WriteString("FILE\n")
	if strings.TrimSpace(report.File) == "" {
		builder.WriteString("  unknown\n")
	} else {
		fmt.Fprintf(builder, "  %s\n", report.File)
	}
	if report.Package != "" {
		fmt.Fprintf(builder, "  Package: %s\n", report.Package)
	}
	if report.PackageName != "" {
		fmt.Fprintf(builder, "  Package name: %s\n", report.PackageName)
	}
}

func writePackageTarget(builder *strings.Builder, report PackageReport) {
	builder.WriteString("PACKAGE\n")
	if strings.TrimSpace(report.Package) == "" {
		builder.WriteString("  unknown\n")
	} else {
		fmt.Fprintf(builder, "  %s\n", report.Package)
	}
	if report.PackageName != "" {
		fmt.Fprintf(builder, "  Package name: %s\n", report.PackageName)
	}
}

func writeDefinition(builder *strings.Builder, symbol sherpa.Symbol) {
	builder.WriteString("DEFINITION\n")
	fmt.Fprintf(builder, "  %s:%d\n", symbol.Position.File, symbol.Position.Line)
}

func writeSource(builder *strings.Builder, context sherpa.SourceContext) {
	builder.WriteString("SOURCE\n")
	if len(context.Lines) == 0 {
		builder.WriteString("  none\n")
		return
	}

	builder.WriteString(sherpa.FormatSourceContext(context, "  "))
}

func writePurpose(builder *strings.Builder, purpose string) {
	builder.WriteString("PURPOSE\n")
	purpose = strings.TrimSpace(purpose)
	if purpose == "" {
		builder.WriteString("  none\n")
		return
	}

	for _, line := range strings.Split(purpose, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			builder.WriteString("\n")
			continue
		}

		builder.WriteString("  ")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
}

func writeAnalysis(builder *strings.Builder, report Report) {
	builder.WriteString("ANALYSIS\n")
	mode := strings.TrimSpace(report.AnalysisMode)
	if mode == "" {
		mode = "unknown"
	}
	confidence := strings.TrimSpace(report.Confidence)
	if confidence == "" {
		confidence = "unknown"
	}

	fmt.Fprintf(builder, "  Mode: %s\n", mode)
	if strings.TrimSpace(report.ReferenceAnalysisMode) != "" {
		fmt.Fprintf(builder, "  Reference analysis: %s\n", report.ReferenceAnalysisMode)
	}
	if strings.TrimSpace(report.CallAnalysisMode) != "" {
		fmt.Fprintf(builder, "  Call analysis: %s\n", report.CallAnalysisMode)
	}
	writeInterfaceAnalysis(builder, report.InterfaceAnalysisMode)
	writeTestAnalysis(builder, report.TestAnalysisMode)
	fmt.Fprintf(builder, "  Confidence: %s\n", confidence)
	if report.Risk.Level != "" {
		fmt.Fprintf(builder, "  Risk: %s\n", report.Risk.Level)
	}
	if report.ArchitectureRole.Role != "" {
		fmt.Fprintf(builder, "  Architecture role: %s\n", report.ArchitectureRole.Role)
	}
}

func writeFileAnalysis(builder *strings.Builder, report FileReport) {
	builder.WriteString("ANALYSIS\n")
	mode := strings.TrimSpace(report.AnalysisMode)
	if mode == "" {
		mode = "unknown"
	}
	confidence := strings.TrimSpace(report.Confidence)
	if confidence == "" {
		confidence = "unknown"
	}

	fmt.Fprintf(builder, "  Mode: %s\n", mode)
	writeInterfaceAnalysis(builder, report.InterfaceAnalysisMode)
	writeTestAnalysis(builder, report.TestAnalysisMode)
	fmt.Fprintf(builder, "  Confidence: %s\n", confidence)
	if report.Risk.Level != "" {
		fmt.Fprintf(builder, "  Risk: %s\n", report.Risk.Level)
	}
	for _, reason := range report.Risk.Reasons {
		fmt.Fprintf(builder, "  - %s\n", reason)
	}
}

func writePackageAnalysis(builder *strings.Builder, report PackageReport) {
	builder.WriteString("ANALYSIS\n")
	mode := strings.TrimSpace(report.AnalysisMode)
	if mode == "" {
		mode = "unknown"
	}
	confidence := strings.TrimSpace(report.Confidence)
	if confidence == "" {
		confidence = "unknown"
	}

	fmt.Fprintf(builder, "  Mode: %s\n", mode)
	writeInterfaceAnalysis(builder, report.InterfaceAnalysisMode)
	writeTestAnalysis(builder, report.TestAnalysisMode)
	fmt.Fprintf(builder, "  Confidence: %s\n", confidence)
	if report.Risk.Level != "" {
		fmt.Fprintf(builder, "  Risk: %s\n", report.Risk.Level)
	}
	for _, reason := range report.Risk.Reasons {
		fmt.Fprintf(builder, "  - %s\n", reason)
	}
}

func writeInterfaceAnalysis(builder *strings.Builder, analysisMode string) {
	analysisMode = strings.TrimSpace(analysisMode)
	if analysisMode == "" {
		return
	}

	fmt.Fprintf(builder, "  Interface analysis: %s\n", analysisMode)
}

func writeTestAnalysis(builder *strings.Builder, analysisMode string) {
	analysisMode = strings.TrimSpace(analysisMode)
	if analysisMode == "" {
		return
	}

	fmt.Fprintf(builder, "  Test analysis: %s\n", analysisMode)
}

func writeTruncation(builder *strings.Builder, truncation *Truncation) {
	messages := truncationMessages(truncation)
	if len(messages) == 0 {
		return
	}

	builder.WriteString("\n")
	builder.WriteString("TRUNCATED\n")
	for _, message := range messages {
		fmt.Fprintf(builder, "  %s\n", message)
	}
}

func writeCallers(builder *strings.Builder, callers []sherpa.Caller) {
	builder.WriteString("CALLED BY\n")
	if len(callers) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, caller := range callers {
		fmt.Fprintf(builder, "  %-36s %s:%d\n", caller.Name, caller.Position.File, caller.Position.Line)
	}
}

func writeCallees(builder *strings.Builder, callees []sherpa.Callee) {
	builder.WriteString("CALLS\n")
	if len(callees) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, callee := range callees {
		fmt.Fprintf(builder, "  %-36s %s:%d\n", callee.Name, callee.Position.File, callee.Position.Line)
	}
}

func writeFileSymbols(builder *strings.Builder, symbols []sherpa.Symbol) {
	builder.WriteString("FILE SYMBOLS\n")
	if len(symbols) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, symbol := range symbols {
		fmt.Fprintf(
			builder,
			"  %-10s %-36s %s:%d\n",
			symbol.Kind,
			symbol.DisplayName(),
			symbol.Position.File,
			symbol.Position.Line,
		)
	}
}

func writePackageSymbols(builder *strings.Builder, symbols []sherpa.Symbol) {
	builder.WriteString("PACKAGE SYMBOLS\n")
	if len(symbols) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, symbol := range symbols {
		fmt.Fprintf(
			builder,
			"  %-10s %-36s %s:%d\n",
			symbol.Kind,
			symbol.DisplayName(),
			symbol.Position.File,
			symbol.Position.Line,
		)
	}
}

func writeFileSourceContexts(builder *strings.Builder, symbols []sherpa.Symbol, contexts []sherpa.SourceContext) {
	builder.WriteString("SOURCE\n")
	if len(contexts) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for index, context := range contexts {
		title := context.Position.File
		if context.Position.Line > 0 {
			title = fmt.Sprintf("%s:%d", context.Position.File, context.Position.Line)
		}
		if index < len(symbols) {
			title = fmt.Sprintf("%s %s", symbols[index].DisplayName(), title)
		}

		fmt.Fprintf(builder, "  %s\n", title)
		builder.WriteString(sherpa.FormatSourceContext(context, "    "))
	}
}

func writeReferences(builder *strings.Builder, refs []sherpa.Reference) {
	builder.WriteString("REFERENCES\n")
	if len(refs) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, ref := range refs {
		fmt.Fprintf(builder, "  %s:%d\n", ref.Position.File, ref.Position.Line)
	}
}

func writeRelatedTests(builder *strings.Builder, tests []sherpa.RelatedTest) {
	builder.WriteString("SUGGESTED TESTS\n")
	if len(tests) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, test := range tests {
		tags := relatedTestTags(test)
		if tags == "" {
			fmt.Fprintf(builder, "  %-36s %s:%d\n", test.Name, test.Position.File, test.Position.Line)
			continue
		}

		fmt.Fprintf(builder, "  %-36s %s:%d (%s)\n", test.Name, test.Position.File, test.Position.Line, tags)
	}
}

func relatedTestTags(test sherpa.RelatedTest) string {
	var tags []string
	if test.DirectReference {
		tags = append(tags, "direct")
	}

	if test.ExternalPackage {
		tags = append(tags, "external")
	}

	return strings.Join(tags, ", ")
}

func writeValues(builder *strings.Builder, title string, values []string) {
	builder.WriteString(title)
	builder.WriteString("\n")
	if len(values) == 0 {
		builder.WriteString("  none\n")
		return
	}

	for _, value := range values {
		for _, line := range strings.Split(value, "\n") {
			builder.WriteString("  ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
}

func writeWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	builder.WriteString("\n")
	writeValues(builder, "WARNINGS", warnings)
}
