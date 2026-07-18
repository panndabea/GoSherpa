package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

const (
	doctorAnalysisModeTypechecked = "typechecked"
	doctorAnalysisModeUnavailable = "unavailable"
)

type doctorReport struct {
	Target       string               `json:"target"`
	Environment  doctorEnvironment    `json:"environment"`
	Repository   doctorRepository     `json:"repository"`
	BuildTags    []string             `json:"buildTags"`
	PackageLoad  doctorPackageLoad    `json:"packageLoad"`
	Snapshot     doctorSnapshotStatus `json:"snapshot"`
	AnalysisMode string               `json:"analysisMode"`
	Confidence   string               `json:"confidence"`
	Limitations  []string             `json:"limitations"`
	Suggestions  []string             `json:"suggestions"`
	Warnings     []string             `json:"-"`
}

type doctorEnvironment struct {
	GoVersion string `json:"goVersion"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

type doctorRepository struct {
	Root           string       `json:"root"`
	ModulePath     string       `json:"modulePath"`
	GoModPath      string       `json:"goModPath"`
	GoWork         doctorGoWork `json:"goWork"`
	GoFiles        int          `json:"goFiles"`
	TestFiles      int          `json:"testFiles"`
	GeneratedFiles int          `json:"generatedFiles"`
	NestedModules  []string     `json:"nestedModules"`
}

type doctorGoWork struct {
	Detected bool   `json:"detected"`
	Path     string `json:"path,omitempty"`
	Scope    string `json:"scope,omitempty"`
}

type doctorPackageLoad struct {
	Status       string                 `json:"status"`
	AnalysisMode string                 `json:"analysisMode"`
	PackageCount int                    `json:"packageCount"`
	Packages     []doctorPackageSummary `json:"packages"`
	WarningCount int                    `json:"warningCount"`
	Message      string                 `json:"message,omitempty"`
}

type doctorPackageSummary struct {
	Package       string `json:"package"`
	Name          string `json:"name"`
	Files         int    `json:"files"`
	CompiledFiles int    `json:"compiledFiles"`
}

type doctorSnapshotStatus struct {
	Supported            bool                               `json:"supported"`
	Status               string                             `json:"status"`
	Path                 string                             `json:"path,omitempty"`
	Message              string                             `json:"message"`
	FormatVersion        int                                `json:"formatVersion,omitempty"`
	CreatedAt            string                             `json:"createdAt,omitempty"`
	Fingerprint          string                             `json:"fingerprint,omitempty"`
	CurrentFingerprint   string                             `json:"currentFingerprint,omitempty"`
	FileCount            int                                `json:"fileCount,omitempty"`
	PackageCount         int                                `json:"packageCount,omitempty"`
	SymbolCount          int                                `json:"symbolCount,omitempty"`
	RelationshipMetadata snapshotstore.RelationshipMetadata `json:"relationshipMetadata"`
	StaleReasons         []string                           `json:"staleReasons"`
}

func analyzeDoctor(root string, buildTags []string) doctorReport {
	normalizedTags := semantics.NormalizeBuildTags(buildTags)
	report := doctorReport{
		Target: ".",
		Environment: doctorEnvironment{
			GoVersion: runtime.Version(),
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
		},
		Repository: doctorRepository{
			Root:      filepath.Clean(root),
			GoModPath: "go.mod",
		},
		BuildTags: normalizedTags,
		Snapshot:  inspectDoctorSnapshot(root, normalizedTags),
	}

	modulePath, err := sherpa.ModulePath(root)
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("module path unavailable: %v", err))
	}
	report.Repository.ModulePath = modulePath

	goFiles, err := sherpa.FindGoFiles(root)
	if err != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("go file scan failed: %v", err))
	} else {
		report.Repository.GoFiles = len(goFiles)
		report.Repository.TestFiles = countTestFiles(goFiles)
		report.Repository.GeneratedFiles = countGeneratedFiles(goFiles)
	}

	report.Repository.GoWork = detectGoWork(root)
	nestedModules, nestedWarnings := findNestedModules(root)
	report.Repository.NestedModules = nestedModules
	report.Warnings = append(report.Warnings, nestedWarnings...)

	repo, err := semantics.LoadRepository(root, semantics.LoadOptions{
		BuildTags: normalizedTags,
	})
	if err != nil {
		message := fmt.Sprintf("typechecked package loading failed: %v", err)
		report.PackageLoad = doctorPackageLoad{
			Status:       "failed",
			AnalysisMode: doctorAnalysisModeUnavailable,
			WarningCount: 1,
			Message:      message,
		}
		report.AnalysisMode = doctorAnalysisModeUnavailable
		report.Warnings = append(report.Warnings, message)
	} else {
		report.PackageLoad = doctorPackageLoad{
			Status:       doctorPackageLoadStatus(repo.Warnings),
			AnalysisMode: doctorAnalysisModeTypechecked,
			PackageCount: len(repo.Packages),
			Packages:     doctorPackageSummaries(repo.Packages),
			WarningCount: len(repo.Warnings),
		}
		report.AnalysisMode = doctorAnalysisModeTypechecked
		report.Warnings = append(report.Warnings, repo.Warnings...)
	}

	report.Warnings = uniqueStringsInOrder(report.Warnings)
	report.Confidence = jsonConfidence(report.Warnings, report.AnalysisMode)
	report.Limitations = doctorLimitations()
	report.Suggestions = doctorSuggestions(report)

	return normalizeDoctorReport(report)
}

func normalizeDoctorReport(report doctorReport) doctorReport {
	if strings.TrimSpace(report.Target) == "" {
		report.Target = "."
	}
	report.Repository.Root = filepath.Clean(report.Repository.Root)
	report.Repository.NestedModules = nonNilSlice(report.Repository.NestedModules)
	report.BuildTags = nonNilSlice(semantics.NormalizeBuildTags(report.BuildTags))
	report.Snapshot.StaleReasons = nonNilSlice(report.Snapshot.StaleReasons)
	report.PackageLoad.Packages = nonNilSlice(report.PackageLoad.Packages)
	report.Limitations = nonNilSlice(report.Limitations)
	report.Suggestions = nonNilSlice(report.Suggestions)
	report.Warnings = nonNilSlice(uniqueStringsInOrder(report.Warnings))
	if strings.TrimSpace(report.AnalysisMode) == "" {
		report.AnalysisMode = doctorAnalysisModeUnavailable
	}
	if strings.TrimSpace(report.Confidence) == "" {
		report.Confidence = jsonConfidence(report.Warnings, report.AnalysisMode)
	}

	return report
}

func doctorPackageLoadStatus(warnings []string) string {
	if len(warnings) > 0 {
		return "warnings"
	}

	return "ok"
}

func doctorPackageSummaries(packages []semantics.Package) []doctorPackageSummary {
	summaries := make([]doctorPackageSummary, 0, len(packages))
	for _, pkg := range packages {
		summaries = append(summaries, doctorPackageSummary{
			Package:       pkg.PackagePath,
			Name:          pkg.Name,
			Files:         len(pkg.GoFiles),
			CompiledFiles: len(pkg.CompiledGoFiles),
		})
	}

	sort.Slice(summaries, func(i int, j int) bool {
		if summaries[i].Package != summaries[j].Package {
			return summaries[i].Package < summaries[j].Package
		}
		return summaries[i].Name < summaries[j].Name
	})

	return summaries
}

func doctorLimitations() []string {
	return []string{
		"Doctor checks repository readiness; it does not prove every downstream analysis is complete.",
		"Package loading follows the current Go environment and any provided --tags values.",
		"Generated files are included when Go package loading includes them; generated-file policy is currently informational.",
		"Snapshots store repository inventory, freshness metadata, bounded relationship metadata, and selected reusable relationship records; reuse remains opt-in per command.",
	}
}

func doctorSuggestions(report doctorReport) []string {
	var suggestions []string
	switch report.PackageLoad.Status {
	case "failed":
		suggestions = append(suggestions, "Fix package load errors, install missing dependencies, or rerun with the build tags used by this repository.")
	case "warnings":
		suggestions = append(suggestions, "Review package load warnings before relying on semantic references, calls, or interface results.")
	default:
		suggestions = append(suggestions, "Run gosherpa agent context --base <ref> or gosherpa context symbol <target> for focused code intelligence.")
	}

	if len(report.BuildTags) == 0 {
		suggestions = append(suggestions, "Use --tags <list> when inspecting build-tagged code.")
	}
	if report.Repository.GoWork.Detected {
		suggestions = append(suggestions, "A go.work file is visible to this module; package loading may follow workspace module resolution.")
	}
	if len(report.Repository.NestedModules) > 0 {
		suggestions = append(suggestions, "Nested modules were found; inspect them with separate --root values when needed.")
	}
	switch report.Snapshot.Status {
	case snapshotstore.StatusMissing:
		suggestions = append(suggestions, "Run gosherpa snapshot to create a reusable repository inventory snapshot.")
	case snapshotstore.StatusStale:
		suggestions = append(suggestions, "Run gosherpa snapshot to refresh the stale repository snapshot.")
	case snapshotstore.StatusInvalid:
		suggestions = append(suggestions, "Recreate the repository snapshot with gosherpa snapshot.")
	}

	return uniqueStringsInOrder(suggestions)
}

func formatDoctorReport(report doctorReport) string {
	report = normalizeDoctorReport(report)

	var builder strings.Builder
	builder.WriteString("DOCTOR\n\n")
	fmt.Fprintf(&builder, "Repository: %s\n", report.Repository.Root)
	fmt.Fprintf(&builder, "Module: %s\n", valueOrNone(report.Repository.ModulePath))
	fmt.Fprintf(&builder, "Go: %s %s/%s\n", report.Environment.GoVersion, report.Environment.GOOS, report.Environment.GOARCH)
	fmt.Fprintf(&builder, "Analysis: %s\n", report.AnalysisMode)
	fmt.Fprintf(&builder, "Confidence: %s\n", report.Confidence)
	builder.WriteString("\n")

	builder.WriteString("FILES\n")
	fmt.Fprintf(&builder, "  go.mod: %s\n", valueOrNone(report.Repository.GoModPath))
	fmt.Fprintf(&builder, "  go.work: %s\n", formatGoWork(report.Repository.GoWork))
	fmt.Fprintf(&builder, "  Go files: %d\n", report.Repository.GoFiles)
	fmt.Fprintf(&builder, "  Test files: %d\n", report.Repository.TestFiles)
	fmt.Fprintf(&builder, "  Generated files: %d\n", report.Repository.GeneratedFiles)
	writeDoctorValues(&builder, "  Nested modules", report.Repository.NestedModules)
	builder.WriteString("\n")

	builder.WriteString("BUILD TAGS\n")
	writeDoctorIndentedValues(&builder, report.BuildTags)
	builder.WriteString("\n")

	builder.WriteString("PACKAGE LOAD\n")
	fmt.Fprintf(&builder, "  Status: %s\n", report.PackageLoad.Status)
	fmt.Fprintf(&builder, "  Analysis: %s\n", report.PackageLoad.AnalysisMode)
	fmt.Fprintf(&builder, "  Packages: %d\n", report.PackageLoad.PackageCount)
	fmt.Fprintf(&builder, "  Warnings: %d\n", report.PackageLoad.WarningCount)
	if strings.TrimSpace(report.PackageLoad.Message) != "" {
		fmt.Fprintf(&builder, "  Message: %s\n", report.PackageLoad.Message)
	}
	builder.WriteString("\n")

	builder.WriteString("SNAPSHOT\n")
	fmt.Fprintf(&builder, "  Status: %s\n", report.Snapshot.Status)
	if strings.TrimSpace(report.Snapshot.Path) != "" {
		fmt.Fprintf(&builder, "  Path: %s\n", report.Snapshot.Path)
	}
	if report.Snapshot.FormatVersion > 0 {
		fmt.Fprintf(&builder, "  Format: %d\n", report.Snapshot.FormatVersion)
	}
	if strings.TrimSpace(report.Snapshot.CreatedAt) != "" {
		fmt.Fprintf(&builder, "  Created: %s\n", report.Snapshot.CreatedAt)
	}
	if report.Snapshot.FileCount > 0 || report.Snapshot.PackageCount > 0 || report.Snapshot.SymbolCount > 0 {
		fmt.Fprintf(&builder, "  Files: %d\n", report.Snapshot.FileCount)
		fmt.Fprintf(&builder, "  Packages: %d\n", report.Snapshot.PackageCount)
		fmt.Fprintf(&builder, "  Symbols: %d\n", report.Snapshot.SymbolCount)
	}
	writeSnapshotRelationshipMetadata(&builder, report.Snapshot.RelationshipMetadata)
	fmt.Fprintf(&builder, "  %s\n", report.Snapshot.Message)
	if len(report.Snapshot.StaleReasons) > 0 {
		writeDoctorValues(&builder, "  Stale reasons", report.Snapshot.StaleReasons)
	}
	builder.WriteString("\n")

	writeDoctorSection(&builder, "SUGGESTIONS", report.Suggestions)
	builder.WriteString("\n")
	writeDoctorSection(&builder, "LIMITATIONS", report.Limitations)

	if len(report.Warnings) > 0 {
		builder.WriteString("\n")
		writeDoctorSection(&builder, "WARNINGS", report.Warnings)
	}

	return builder.String()
}

func writeDoctorValues(builder *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		fmt.Fprintf(builder, "%s: none\n", label)
		return
	}

	fmt.Fprintf(builder, "%s:\n", label)
	writeDoctorIndentedValuesWithIndent(builder, values, "    ")
}

func writeDoctorSection(builder *strings.Builder, title string, values []string) {
	builder.WriteString(title)
	builder.WriteString("\n")
	writeDoctorIndentedValues(builder, values)
}

func writeDoctorIndentedValues(builder *strings.Builder, values []string) {
	writeDoctorIndentedValuesWithIndent(builder, values, "  ")
}

func writeDoctorIndentedValuesWithIndent(builder *strings.Builder, values []string, indent string) {
	if len(values) == 0 {
		fmt.Fprintf(builder, "%snone\n", indent)
		return
	}

	for _, value := range values {
		fmt.Fprintf(builder, "%s%s\n", indent, value)
	}
}

func valueOrNone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "none"
	}

	return value
}

func formatGoWork(goWork doctorGoWork) string {
	if !goWork.Detected {
		return "none"
	}
	if goWork.Scope == "" {
		return goWork.Path
	}

	return fmt.Sprintf("%s (%s)", goWork.Path, goWork.Scope)
}

func countTestFiles(files []string) int {
	count := 0
	for _, file := range files {
		if strings.HasSuffix(filepath.ToSlash(file), "_test.go") {
			count++
		}
	}

	return count
}

func countGeneratedFiles(files []string) int {
	count := 0
	for _, file := range files {
		if generatedGoFile(file) {
			count++
		}
	}

	return count
}

func generatedGoFile(file string) bool {
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}

	lines := strings.SplitN(string(data), "\n", 12)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "// Code generated ") && strings.HasSuffix(trimmed, " DO NOT EDIT.") {
			return true
		}
	}

	return false
}

func detectGoWork(root string) doctorGoWork {
	root = filepath.Clean(root)
	for dir := root; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, "go.work")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			scope := "parent"
			if dir == root {
				scope = "root"
			}
			return doctorGoWork{
				Detected: true,
				Path:     doctorDisplayPath(root, path),
				Scope:    scope,
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}

	return doctorGoWork{}
}

func findNestedModules(root string) ([]string, []string) {
	var modules []string
	var warnings []string
	root = filepath.Clean(root)

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, fmt.Sprintf("nested module scan skipped %s: %v", doctorDisplayPath(root, path), walkErr))
			return nil
		}
		if path == root {
			return nil
		}
		if entry.IsDir() && doctorShouldSkipDir(entry.Name()) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != "go.mod" {
			return nil
		}
		if filepath.Clean(path) == filepath.Join(root, "go.mod") {
			return nil
		}

		modules = append(modules, filepath.ToSlash(filepath.Dir(doctorDisplayPath(root, path))))
		return nil
	})
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("nested module scan failed: %v", err))
	}

	sort.Strings(modules)
	return modules, uniqueStringsInOrder(warnings)
}

func doctorShouldSkipDir(name string) bool {
	switch name {
	case ".git", ".gosherpa", "vendor":
		return true
	default:
		return false
	}
}

func inspectDoctorSnapshot(root string, buildTags []string) doctorSnapshotStatus {
	result := snapshotstore.Inspect(root, snapshotstore.BuildOptions{
		BuildTags: buildTags,
	})

	return doctorSnapshotStatus{
		Supported:            result.Supported,
		Status:               result.Status,
		Path:                 result.Path,
		Message:              result.Message,
		FormatVersion:        result.FormatVersion,
		CreatedAt:            result.CreatedAt,
		Fingerprint:          result.Fingerprint,
		CurrentFingerprint:   result.CurrentFingerprint,
		FileCount:            result.FileCount,
		PackageCount:         result.PackageCount,
		SymbolCount:          result.SymbolCount,
		RelationshipMetadata: result.RelationshipMetadata,
		StaleReasons:         result.StaleReasons,
	}
}

func doctorDisplayPath(root string, path string) string {
	relative, err := filepath.Rel(root, path)
	if err == nil && relative != "." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != ".." {
		return filepath.ToSlash(relative)
	}

	return filepath.Clean(path)
}
