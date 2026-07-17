package agentcontext

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	_ "unsafe"

	explainengine "github.com/panndabea/GoSherpa/internal/explain"
	impactengine "github.com/panndabea/GoSherpa/internal/impact"
	"github.com/panndabea/GoSherpa/internal/semantics"
	"github.com/panndabea/GoSherpa/internal/sherpa"
	snapshotstore "github.com/panndabea/GoSherpa/internal/snapshot"
)

//go:linkname agentContextTestLoadSemanticContextRepository github.com/panndabea/GoSherpa/internal/sherpa.loadSemanticContextRepository
var agentContextTestLoadSemanticContextRepository func(string, semantics.LoadOptions) (semantics.Repository, error)

func TestAnalyzeSymbolBuildsAgentContext(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.Target != "Target" {
		t.Fatalf("target = %q, want Target", report.Target)
	}
	if report.Identity.Package != "." {
		t.Fatalf("identity package = %q, want .", report.Identity.Package)
	}
	if report.Identity.Symbol != "Target" {
		t.Fatalf("identity symbol = %q, want Target", report.Identity.Symbol)
	}
	if report.Identity.Signature != "func Target()" {
		t.Fatalf("identity signature = %q, want func Target()", report.Identity.Signature)
	}
	if report.SourceContext.Position.File != "service.go" {
		t.Fatalf("source context file = %q, want service.go", report.SourceContext.Position.File)
	}
	if len(report.SourceContext.Lines) == 0 {
		t.Fatal("expected source context lines")
	}
	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeTypecheckedAST)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if report.ReferenceAnalysisMode != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s", report.ReferenceAnalysisMode, sherpa.ReferenceAnalysisModeTypechecked)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if len(report.Limitations) == 0 {
		t.Fatal("expected limitations")
	}
	if !strings.Contains(report.Limitations[0], "typechecked") {
		t.Fatalf("expected typechecked symbol limitation, got %#v", report.Limitations)
	}
	if !strings.Contains(report.Limitations[1], "typechecked") {
		t.Fatalf("expected typechecked reference limitation, got %#v", report.Limitations)
	}
	if !strings.Contains(report.Limitations[2], "typechecked") {
		t.Fatalf("expected typechecked call limitation, got %#v", report.Limitations)
	}
	assertAgentContextLimitationsContainAll(t, report.Limitations,
		"Dynamic dispatch",
		"reflection",
		"Generated Go files",
		"Package load warnings lower confidence",
		"Test callers are included only when --tests",
	)
	if len(report.Callers) != 1 || report.Callers[0].Name != "Entry" {
		t.Fatalf("expected Entry caller, got %#v", report.Callers)
	}
	if len(report.Callees) != 1 || report.Callees[0].Name != "Helper" {
		t.Fatalf("expected Helper callee, got %#v", report.Callees)
	}
	if len(report.RelatedTests) != 1 || report.RelatedTests[0].Name != "TestTarget" {
		t.Fatalf("expected TestTarget related test, got %#v", report.RelatedTests)
	}
}

func TestAnalyzeSymbolUsesPackageQualifiedSemanticIdentity(t *testing.T) {
	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

func Target() {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "billing", "billing.go"), `package billing

func Target() {}
`)

	report, err := AnalyzeSymbol(root, "./internal/auth.Target", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeTypecheckedAST)
	}
	if report.Identity.Package != "./internal/auth" {
		t.Fatalf("identity package = %q, want ./internal/auth", report.Identity.Package)
	}
	if report.Identity.QualifiedName != "./internal/auth.Target" {
		t.Fatalf("identity qualified name = %q, want ./internal/auth.Target", report.Identity.QualifiedName)
	}
	if report.SourceContext.Position.File != "internal/auth/auth.go" {
		t.Fatalf("source context file = %q, want internal/auth/auth.go", report.SourceContext.Position.File)
	}
}

func TestAnalyzeSymbolUsesWorkspaceSemanticContext(t *testing.T) {
	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.work"), `go 1.24.4

use (
	./app
	./service
)
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service", "service.go"), `package service

type Payload struct{}

type Processor interface {
	Process(Payload) error
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "app", "go.mod"), `module example.com/app

go 1.24.4

require example.com/service v0.0.0
`)
	writeAgentContextTestFile(t, filepath.Join(root, "app", "processor", "processor.go"), `package processor

import "example.com/service"

type LocalProcessor struct{}

func (LocalProcessor) Process(service.Payload) error {
	return nil
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "app", "main.go"), `package app

import (
	"example.com/app/processor"
	"example.com/service"
)

func Run() {
	local := processor.LocalProcessor{}
	_ = local.Process(service.Payload{})
}
`)

	report, err := AnalyzeSymbol(root, "./app/processor.LocalProcessor.Process", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.Identity.Package != "./app/processor" {
		t.Fatalf("identity package = %q, want ./app/processor", report.Identity.Package)
	}
	if report.SourceContext.Position.File != "app/processor/processor.go" {
		t.Fatalf("source context file = %q, want app/processor/processor.go", report.SourceContext.Position.File)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if len(report.Callers) != 1 || report.Callers[0].Name != "Run" || report.Callers[0].Position.File != "app/main.go" {
		t.Fatalf("expected workspace Run caller, got %#v", report.Callers)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if !agentContextStringSliceContains(report.AffectedInterfaces, "./service.Processor") {
		t.Fatalf("expected service Processor interface, got %#v", report.AffectedInterfaces)
	}
	if !agentContextStringSliceContains(report.AffectedImplementations, "./app/processor.LocalProcessor") {
		t.Fatalf("expected app LocalProcessor implementation, got %#v", report.AffectedImplementations)
	}

	importPathReport, err := AnalyzeSymbol(root, "example.com/app/processor.LocalProcessor.Process", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol with import path target returned error: %v", err)
	}
	if importPathReport.Identity.Package != "./app/processor" {
		t.Fatalf("import path identity package = %q, want ./app/processor", importPathReport.Identity.Package)
	}
	if importPathReport.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("import path analysis mode = %q, want %s with warnings %#v", importPathReport.AnalysisMode, AnalysisModeTypecheckedAST, importPathReport.Warnings)
	}
	if !agentContextStringSliceContains(importPathReport.AffectedInterfaces, "./service.Processor") {
		t.Fatalf("expected import path target to report service Processor interface, got %#v", importPathReport.AffectedInterfaces)
	}
	if !agentContextStringSliceContains(importPathReport.AffectedImplementations, "./app/processor.LocalProcessor") {
		t.Fatalf("expected import path target to report app LocalProcessor implementation, got %#v", importPathReport.AffectedImplementations)
	}
}

func TestAnalyzeSymbolUsesGoWorkWhenRootAlsoHasGoMod(t *testing.T) {
	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), `module example.com/root

go 1.24.4

require example.com/service v0.0.0
`)
	writeAgentContextTestFile(t, filepath.Join(root, "go.work"), `go 1.24.4

use (
	.
	./service
)
`)
	writeAgentContextTestFile(t, filepath.Join(root, "root.go"), `package root

import "example.com/service"

func Run(svc service.Service) error {
	process := svc.Process
	return process(service.Payload{})
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service", "service.go"), `package service

type Payload struct{}

type Processor interface {
	Process(Payload) error
}

type Service struct{}

func (Service) Process(Payload) error {
	return nil
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service", "service_test.go"), `package service

import "testing"

func TestServiceProcess(t *testing.T) {
	if (Service{}).Process(Payload{}) != nil {
		t.Fatal("unexpected error")
	}
}
`)

	report, err := AnalyzeSymbol(root, "./service.Service.Process", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.Identity.Package != "./service" {
		t.Fatalf("identity package = %q, want ./service", report.Identity.Package)
	}
	if report.SourceContext.Position.File != "service/service.go" {
		t.Fatalf("source context file = %q, want service/service.go", report.SourceContext.Position.File)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if len(report.Callers) != 1 || report.Callers[0].Name != "Run" || report.Callers[0].Position.File != "root.go" {
		t.Fatalf("expected root Run caller through method value, got %#v", report.Callers)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if !agentContextStringSliceContains(report.AffectedInterfaces, "./service.Processor") {
		t.Fatalf("expected service Processor interface, got %#v", report.AffectedInterfaces)
	}
	if !agentContextStringSliceContains(report.AffectedImplementations, "./service.Service") {
		t.Fatalf("expected service implementation signal, got %#v", report.AffectedImplementations)
	}
	if len(report.RelatedTests) != 1 || report.RelatedTests[0].Name != "TestServiceProcess" {
		t.Fatalf("expected service related test, got %#v", report.RelatedTests)
	}

	importPathReport, err := AnalyzeSymbol(root, "example.com/service.Service.Process", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol with workspace import path target returned error: %v", err)
	}
	if importPathReport.Identity.Package != "./service" {
		t.Fatalf("import path identity package = %q, want ./service", importPathReport.Identity.Package)
	}
	if importPathReport.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("import path analysis mode = %q, want %s with warnings %#v", importPathReport.AnalysisMode, AnalysisModeTypecheckedAST, importPathReport.Warnings)
	}
}

func TestAnalyzeSymbolSkipsNestedModuleTargets(t *testing.T) {
	root := writeAgentContextNestedModuleProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.Identity.Package != "." {
		t.Fatalf("identity package = %q, want .", report.Identity.Package)
	}
	if report.Identity.Definition.File != "service.go" {
		t.Fatalf("expected root service.go definition, got %#v", report.Identity.Definition)
	}
	if report.SourceContext.Position.File != "service.go" {
		t.Fatalf("expected root source context, got %#v", report.SourceContext.Position)
	}
	if agentContextRelatedTestsContain(report.RelatedTests, "TestNestedTarget") {
		t.Fatalf("did not expect nested-module test, got %#v", report.RelatedTests)
	}
	if !agentContextRelatedTestsContain(report.RelatedTests, "TestTarget") {
		t.Fatalf("expected root TestTarget, got %#v", report.RelatedTests)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s with warnings %#v", report.Confidence, ConfidenceMedium, report.Warnings)
	}
}

func TestAnalyzeSymbolBuildsContextForGenericReceiverMethod(t *testing.T) {
	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "cache", "cache.go"), `package cache

type Cache[T any] struct {
	value T
}

func (c Cache[T]) Get() T {
	return c.value
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "cmd", "app", "main.go"), `package main

import "example.com/app/internal/cache"

func RunDirect(values cache.Cache[string]) string {
	return values.Get()
}

func RunValue(values cache.Cache[string]) string {
	get := values.Get
	return get()
}
`)

	report, err := AnalyzeSymbol(root, "./internal/cache.Cache.Get", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.Identity.Package != "./internal/cache" {
		t.Fatalf("identity package = %q, want ./internal/cache", report.Identity.Package)
	}
	if report.Identity.Symbol != "Cache.Get" {
		t.Fatalf("identity symbol = %q, want Cache.Get", report.Identity.Symbol)
	}
	if report.Identity.Signature != "func (c Cache[T]) Get() T" {
		t.Fatalf("identity signature = %q, want generic receiver signature", report.Identity.Signature)
	}
	if report.SourceContext.Position.File != "internal/cache/cache.go" {
		t.Fatalf("source context file = %q, want internal/cache/cache.go", report.SourceContext.Position.File)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if len(report.Callers) != 2 {
		t.Fatalf("expected direct and method-value callers, got %#v", report.Callers)
	}
	if !agentContextCallersContain(report.Callers, "RunDirect", "cmd/app/main.go") {
		t.Fatalf("expected RunDirect caller, got %#v", report.Callers)
	}
	if !agentContextCallersContain(report.Callers, "RunValue", "cmd/app/main.go") {
		t.Fatalf("expected RunValue method-value caller, got %#v", report.Callers)
	}
}

func TestAnalyzeSymbolUsesBuildTagsForSemanticIdentity(t *testing.T) {
	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package app

func Always() {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "enterprise.go"), `//go:build enterprise

package app

func Target() {}
`)

	withoutTags, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol without tags returned error: %v", err)
	}
	if withoutTags.AnalysisMode != AnalysisModeAST {
		t.Fatalf("analysis mode without tags = %q, want %s", withoutTags.AnalysisMode, AnalysisModeAST)
	}

	withTags, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{BuildTags: []string{"enterprise"}})
	if err != nil {
		t.Fatalf("AnalyzeSymbol with tags returned error: %v", err)
	}
	if withTags.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode with tags = %q, want %s", withTags.AnalysisMode, AnalysisModeTypecheckedAST)
	}
	if withTags.SourceContext.Position.File != "enterprise.go" {
		t.Fatalf("source context file with tags = %q, want enterprise.go", withTags.SourceContext.Position.File)
	}
}

func TestAnalyzeSymbolBuildsTypecheckedContextForTypeAlias(t *testing.T) {
	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "model", "user.go"), `package model

type UserID string
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "auth.go"), `package auth

import "example.com/app/internal/model"

// UserID is the auth-facing user identifier.
type UserID = model.UserID

func Normalize(id UserID) UserID {
	return id
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "auth_test.go"), `package auth

import "testing"

func TestNormalizeUserID(t *testing.T) {
	var id UserID
	_ = Normalize(id)
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "session", "session.go"), `package session

import "example.com/app/internal/auth"

func Load(id auth.UserID) auth.UserID {
	return id
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "session", "session_test.go"), `package session

import (
	"testing"

	"example.com/app/internal/auth"
)

func TestLoadUserID(t *testing.T) {
	var id auth.UserID
	_ = Load(id)
}
`)

	report, err := AnalyzeSymbol(root, "./internal/auth.UserID", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.Identity.Kind != sherpa.SymbolKindAlias {
		t.Fatalf("identity kind = %q, want %s", report.Identity.Kind, sherpa.SymbolKindAlias)
	}
	if report.Identity.Signature != "type UserID = model.UserID" {
		t.Fatalf("identity signature = %q, want alias signature", report.Identity.Signature)
	}
	if report.SourceContext.Position.File != "internal/auth/auth.go" {
		t.Fatalf("source context file = %q, want internal/auth/auth.go", report.SourceContext.Position.File)
	}
	if report.ReferenceAnalysisMode != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s", report.ReferenceAnalysisMode, sherpa.ReferenceAnalysisModeTypechecked)
	}
	if !agentContextStringSliceContains(report.AffectedPackages, "./internal/auth") || !agentContextStringSliceContains(report.AffectedPackages, "./internal/session") {
		t.Fatalf("expected auth and session affected packages, got %#v", report.AffectedPackages)
	}
	if len(report.RelatedTests) < 2 {
		t.Fatalf("expected alias-related tests, got %#v", report.RelatedTests)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s with warnings %#v", report.Confidence, ConfidenceMedium, report.Warnings)
	}
}

func TestAnalyzeSymbolIncludesTestCallersWithOption(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if len(report.Callers) != 2 {
		t.Fatalf("expected 2 callers, got %#v", report.Callers)
	}
	if report.Callers[1].Name != "TestTarget" {
		t.Fatalf("expected test caller, got %#v", report.Callers)
	}
}

func TestNormalizeContextLimitsAppliesDefaultsAndPreservesExplicitValues(t *testing.T) {
	symbol := normalizeSymbolLimits(0, LimitOptions{MaxTests: 3})
	if symbol.MaxReferences != DefaultSymbolMaxReferences || symbol.MaxTests != 3 || symbol.MaxBytes != DefaultMaxBytes {
		t.Fatalf("unexpected symbol limits: %#v", symbol)
	}

	file := normalizeFileLimits(0, LimitOptions{MaxSymbols: 2})
	if file.MaxSymbols != 2 || file.MaxTests != DefaultMaxTests || file.MaxBytes != DefaultMaxBytes {
		t.Fatalf("unexpected file limits: %#v", file)
	}

	pkg := normalizePackageLimits(0, LimitOptions{MaxFiles: 4})
	if pkg.MaxFiles != 4 || pkg.MaxSymbols != DefaultPackageMaxSymbols || pkg.MaxTests != DefaultMaxTests {
		t.Fatalf("unexpected package limits: %#v", pkg)
	}

	diff := normalizeDiffLimits(LimitOptions{MaxBytes: 1024})
	if diff.MaxFiles != DefaultDiffMaxFiles || diff.MaxSymbols != DefaultDiffMaxSymbols || diff.MaxTests != DefaultMaxTests || diff.MaxBytes != 1024 {
		t.Fatalf("unexpected diff limits: %#v", diff)
	}
}

func TestAnalyzeSymbolRecordsDefaultLimits(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.Limits == nil {
		t.Fatal("expected default context limits to be recorded")
	}
	if report.Limits.MaxReferences != DefaultSymbolMaxReferences || report.Limits.MaxTests != DefaultMaxTests || report.Limits.MaxBytes != DefaultMaxBytes {
		t.Fatalf("unexpected default limits: %#v", report.Limits)
	}
}

func TestApplySymbolLimitsPrioritizesDirectTests(t *testing.T) {
	report := Report{
		Symbol: sherpa.Symbol{
			Name:     "Target",
			Kind:     sherpa.SymbolKindFunction,
			Position: sherpa.Position{File: "service.go", Line: 7},
		},
		RelatedTests: []sherpa.RelatedTest{
			{
				Name:     "TestPackageOnly",
				Package:  ".",
				Position: sherpa.Position{File: "service_test.go", Line: 20},
			},
			{
				Name:            "TestDirect",
				Package:         ".",
				Position:        sherpa.Position{File: "service_test.go", Line: 30},
				DirectReference: true,
			},
			{
				Name:     "TestOther",
				Package:  ".",
				Position: sherpa.Position{File: "service_test.go", Line: 40},
			},
		},
	}

	limited := applySymbolLimits(report, LimitOptions{MaxTests: 1})
	if len(limited.RelatedTests) != 1 || limited.RelatedTests[0].Name != "TestDirect" {
		t.Fatalf("expected direct test to be retained, got %#v", limited.RelatedTests)
	}
	if limited.Truncated == nil || limited.Truncated.RelatedTests != 2 {
		t.Fatalf("expected related test truncation, got %#v", limited.Truncated)
	}
	if len(limited.ReadingOrder) != 2 || limited.ReadingOrder[1].Title != "Test: TestDirect" {
		t.Fatalf("expected reading order to include direct test, got %#v", limited.ReadingOrder)
	}
}

func TestApplySymbolByteLimitPreservesCoreSignalsBeforeVerboseTestPlan(t *testing.T) {
	report := Report{
		Symbol: sherpa.Symbol{
			Name:     "Target",
			Kind:     sherpa.SymbolKindFunction,
			Position: sherpa.Position{File: "service.go", Line: 7},
		},
		SourceContext: sherpa.SourceContext{
			Position: sherpa.Position{File: "service.go", Line: 7},
			Lines: []sherpa.SourceContextLine{
				{Number: 7, Text: "func Target() {}", Target: true},
			},
		},
		References: []sherpa.Reference{
			{Position: sherpa.Position{File: "service.go", Line: 7}},
		},
		Callers: []sherpa.Caller{
			{Name: "Entry", Position: sherpa.Position{File: "service.go", Line: 3}},
		},
		Callees: []sherpa.Callee{
			{Name: "Helper", Position: sherpa.Position{File: "service.go", Line: 8}},
		},
		TestPlan: sherpa.TestPlan{
			CallerPackages: []sherpa.TestPlanItem{
				{
					Command: "go test ./internal/service",
					Reason:  strings.Repeat("verbose test plan ", 500),
					Package: "./internal/service",
				},
			},
		},
		ReadingOrder: []explainengine.ReadingStep{
			{Title: "Definition", Position: sherpa.Position{File: "service.go", Line: 7}},
			{Title: "Caller: Entry", Position: sherpa.Position{File: "service.go", Line: 3}},
			{Title: "Callee: Helper", Position: sherpa.Position{File: "service.go", Line: 8}},
		},
	}

	limited := applySymbolByteLimit(report, 1800)
	if len(limited.SourceContext.Lines) != 1 || !limited.SourceContext.Lines[0].Target {
		t.Fatalf("expected target source line to be retained, got %#v", limited.SourceContext.Lines)
	}
	if len(limited.References) != 1 || len(limited.Callers) != 1 || len(limited.Callees) != 1 {
		t.Fatalf("expected core signals to be retained, refs=%#v callers=%#v callees=%#v", limited.References, limited.Callers, limited.Callees)
	}
	if limited.Truncated == nil || limited.Truncated.TestPlanItems == 0 {
		t.Fatalf("expected verbose test plan truncation, got %#v", limited.Truncated)
	}
}

func TestAnalyzeSymbolAppliesLimits(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{
		IncludeTests: true,
		Limits: LimitOptions{
			MaxReferences: 1,
			MaxTests:      1,
			SourceRadius:  NewSourceRadius(0),
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.Limits == nil || report.Limits.MaxReferences != 1 || report.Limits.MaxTests != 1 {
		t.Fatalf("expected limits to be recorded, got %#v", report.Limits)
	}
	if report.Limits.SourceRadius == nil || *report.Limits.SourceRadius != 0 {
		t.Fatalf("expected source radius 0 limit, got %#v", report.Limits)
	}
	if len(report.SourceContext.Lines) != 1 {
		t.Fatalf("expected target-only source context, got %#v", report.SourceContext.Lines)
	}
	if len(report.References) != 1 {
		t.Fatalf("expected one reference, got %#v", report.References)
	}
	if len(report.Callers) != 1 {
		t.Fatalf("expected one caller, got %#v", report.Callers)
	}
	if report.Truncated == nil || report.Truncated.References == 0 || report.Truncated.Callers == 0 {
		t.Fatalf("expected reference and caller truncation, got %#v", report.Truncated)
	}
}

func TestAnalyzeSymbolAppliesByteLimit(t *testing.T) {
	root := writeAgentContextProject(t)
	maxBytes := 2400

	report, err := AnalyzeSymbol(root, "Target", AnalyzeOptions{
		IncludeTests: true,
		Limits: LimitOptions{
			MaxBytes: maxBytes,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeSymbol returned error: %v", err)
	}

	if report.Limits == nil || report.Limits.MaxBytes != maxBytes {
		t.Fatalf("expected max bytes limit to be recorded, got %#v", report.Limits)
	}
	if report.Truncated == nil {
		t.Fatal("expected byte budget truncation")
	}
	if report.Truncated.SourceLines == 0 && report.Truncated.References == 0 && report.Truncated.Callers == 0 && report.Truncated.Callees == 0 {
		t.Fatalf("expected context details to be truncated, got %#v", report.Truncated)
	}
	if size := encodedJSONLen(normalizeReport(report)); size > maxBytes && report.Truncated.ByteBudgetOverage == 0 {
		t.Fatalf("expected report to fit budget or report overage, size=%d budget=%d truncation=%#v", size, maxBytes, report.Truncated)
	}
}

func TestAnalyzeFileBuildsAgentContext(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if report.Target != "service.go" || report.File != "service.go" {
		t.Fatalf("target/file = %q/%q, want service.go/service.go", report.Target, report.File)
	}
	if report.Package != "." {
		t.Fatalf("package = %q, want .", report.Package)
	}
	if report.PackageName != "app" {
		t.Fatalf("package name = %q, want app", report.PackageName)
	}
	if len(report.Symbols) != 3 {
		t.Fatalf("expected 3 file symbols, got %#v", report.Symbols)
	}
	if report.Symbols[0].Name != "Entry" || report.Symbols[1].Name != "Target" || report.Symbols[2].Name != "Helper" {
		t.Fatalf("unexpected symbol order: %#v", report.Symbols)
	}
	if len(report.SourceContexts) != 3 {
		t.Fatalf("expected 3 source contexts, got %#v", report.SourceContexts)
	}
	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeTypecheckedAST)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if report.Risk.Level != "medium" {
		t.Fatalf("risk level = %q, want medium", report.Risk.Level)
	}
	if len(report.AffectedPackages) != 1 || report.AffectedPackages[0] != "." {
		t.Fatalf("expected package impact ., got %#v", report.AffectedPackages)
	}
	if len(report.AffectedTests) != 1 || report.AffectedTests[0].Name != "TestTarget" {
		t.Fatalf("expected TestTarget affected test, got %#v", report.AffectedTests)
	}
	if len(report.TestCommands) != 1 || report.TestCommands[0] != "go test ." {
		t.Fatalf("expected go test . command, got %#v", report.TestCommands)
	}
	if len(report.ReadingOrder) != 5 {
		t.Fatalf("expected 5 reading order steps, got %#v", report.ReadingOrder)
	}
	assertAgentContextReadingStepRange(t, report.ReadingOrder[1], "service.go", 3, 1, 5, 2)
	assertAgentContextReadingStepRange(t, report.ReadingOrder[4], "service_test.go", 5, 1, 7, 2)
	if !agentContextLimitationsContain(report.Limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", report.Limitations)
	}
	if !agentContextLimitationsContain(report.Limitations, "Test analysis used AST") {
		t.Fatalf("expected test analysis limitation, got %#v", report.Limitations)
	}
	assertAgentContextLimitationsContainAll(t, report.Limitations,
		"Dynamic dispatch",
		"reflection",
		"Generated Go files",
		"Package load warnings lower confidence",
	)
}

func TestAnalyzeFileSharesSemanticContextForImpactAndSymbols(t *testing.T) {
	root := writeAgentContextProject(t)

	original := agentContextTestLoadSemanticContextRepository
	repositoryLoads := 0
	agentContextTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if !options.IncludeTests {
			repositoryLoads++
		}
		return original(root, options)
	}
	defer func() {
		agentContextTestLoadSemanticContextRepository = original
	}()

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
}

func TestAnalyzeFileAndPackageLowerConfidenceForPackageLoadWarnings(t *testing.T) {
	root := writeAgentContextProjectWithPackageLoadWarning(t)

	fileReport, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}
	if fileReport.Confidence != ConfidenceLow {
		t.Fatalf("file confidence = %q, want %s with warnings %#v", fileReport.Confidence, ConfidenceLow, fileReport.Warnings)
	}
	if !agentContextWarningsContain(fileReport.Warnings, "Missing") {
		t.Fatalf("expected file context package load warning, got %#v", fileReport.Warnings)
	}
	if !agentContextLimitationsContain(fileReport.Limitations, "Package load warnings lower confidence") {
		t.Fatalf("expected file context package load limitation, got %#v", fileReport.Limitations)
	}

	packageReport, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}
	if packageReport.Confidence != ConfidenceLow {
		t.Fatalf("package confidence = %q, want %s with warnings %#v", packageReport.Confidence, ConfidenceLow, packageReport.Warnings)
	}
	if !agentContextWarningsContain(packageReport.Warnings, "Missing") {
		t.Fatalf("expected package context package load warning, got %#v", packageReport.Warnings)
	}
	if !agentContextLimitationsContain(packageReport.Limitations, "Package load warnings lower confidence") {
		t.Fatalf("expected package context package load limitation, got %#v", packageReport.Limitations)
	}
}

func TestNormalizeContextReportsDeduplicateWarnings(t *testing.T) {
	warnings := []string{"package load warning: ./service: Missing", "package load warning: ./service: Missing"}

	fileReport := normalizeFileReport(FileReport{Warnings: warnings})
	if len(fileReport.Warnings) != 1 {
		t.Fatalf("expected deduplicated file warnings, got %#v", fileReport.Warnings)
	}

	packageReport := normalizePackageReport(PackageReport{Warnings: warnings})
	if len(packageReport.Warnings) != 1 {
		t.Fatalf("expected deduplicated package warnings, got %#v", packageReport.Warnings)
	}

	diffReport := normalizeDiffReport(DiffReport{Warnings: warnings})
	if len(diffReport.Warnings) != 1 {
		t.Fatalf("expected deduplicated diff warnings, got %#v", diffReport.Warnings)
	}
}

func TestAnalyzeFileNotesTestsOptionInLimitations(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if !agentContextLimitationsContain(report.Limitations, "--tests") {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
}

func TestAnalyzeFileAppliesLimits(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{
		Limits: LimitOptions{
			MaxSymbols: 1,
			MaxTests:   1,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if len(report.Symbols) != 1 || report.Symbols[0].Name != "Entry" {
		t.Fatalf("expected first symbol only, got %#v", report.Symbols)
	}
	if len(report.SourceContexts) != 1 {
		t.Fatalf("expected one source context, got %#v", report.SourceContexts)
	}
	if report.Truncated == nil || report.Truncated.Symbols != 2 || report.Truncated.SourceContexts != 2 {
		t.Fatalf("expected symbol/source truncation, got %#v", report.Truncated)
	}
}

func TestAnalyzeFileAppliesByteLimit(t *testing.T) {
	root := writeAgentContextProject(t)
	maxBytes := 2600

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{
		Limits: LimitOptions{
			MaxBytes: maxBytes,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if report.Limits == nil || report.Limits.MaxBytes != maxBytes {
		t.Fatalf("expected max bytes limit to be recorded, got %#v", report.Limits)
	}
	if report.Truncated == nil || report.Truncated.SourceContexts == 0 {
		t.Fatalf("expected source context byte truncation, got %#v", report.Truncated)
	}
	if size := encodedJSONLen(normalizeFileReport(report)); size > maxBytes && report.Truncated.ByteBudgetOverage == 0 {
		t.Fatalf("expected report to fit budget or report overage, size=%d budget=%d truncation=%#v", size, maxBytes, report.Truncated)
	}
}

func TestAnalyzeFileByteLimitPreservesCoreSignals(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzeFile(root, "service.go", FileAnalyzeOptions{
		Limits: LimitOptions{
			MaxBytes: 1200,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeFile returned error: %v", err)
	}

	if len(report.Symbols) == 0 {
		t.Fatalf("expected byte-limited file context to keep at least one symbol, got %#v", report.Truncated)
	}
	if len(report.AffectedPackages) == 0 {
		t.Fatalf("expected byte-limited file context to keep affected package signal, got %#v", report.Truncated)
	}
	if len(report.TestCommands) == 0 {
		t.Fatalf("expected byte-limited file context to keep a compact test command, got %#v", report.Truncated)
	}
}

func TestAnalyzePackageBuildsAgentContext(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if report.Target != "." || report.Package != "." {
		t.Fatalf("target/package = %q/%q, want . and .", report.Target, report.Package)
	}
	if report.PackageName != "app" {
		t.Fatalf("package name = %q, want app", report.PackageName)
	}
	if len(report.Files) != 2 || report.Files[0] != "service.go" || report.Files[1] != "service_test.go" {
		t.Fatalf("unexpected package files: %#v", report.Files)
	}
	if len(report.Symbols) != 4 {
		t.Fatalf("expected 4 package symbols, got %#v", report.Symbols)
	}
	if report.Symbols[0].Name != "Entry" || report.Symbols[1].Name != "Target" || report.Symbols[2].Name != "Helper" || report.Symbols[3].Name != "TestTarget" {
		t.Fatalf("unexpected symbol order: %#v", report.Symbols)
	}
	if len(report.SourceContexts) != 4 {
		t.Fatalf("expected 4 source contexts, got %#v", report.SourceContexts)
	}
	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeTypecheckedAST)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if report.Risk.Level != "medium" {
		t.Fatalf("risk level = %q, want medium", report.Risk.Level)
	}
	if len(report.AffectedPackages) != 1 || report.AffectedPackages[0] != "." {
		t.Fatalf("expected package impact ., got %#v", report.AffectedPackages)
	}
	if len(report.AffectedTests) != 1 || report.AffectedTests[0].Name != "TestTarget" {
		t.Fatalf("expected TestTarget affected test, got %#v", report.AffectedTests)
	}
	if len(report.TestCommands) != 1 || report.TestCommands[0] != "go test ." {
		t.Fatalf("expected go test . command, got %#v", report.TestCommands)
	}
	if len(report.ReadingOrder) != 7 {
		t.Fatalf("expected 7 reading order steps, got %#v", report.ReadingOrder)
	}
	if !agentContextLimitationsContain(report.Limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", report.Limitations)
	}
	if !agentContextLimitationsContain(report.Limitations, "Test analysis used AST") {
		t.Fatalf("expected test analysis limitation, got %#v", report.Limitations)
	}
	assertAgentContextLimitationsContainAll(t, report.Limitations,
		"Dynamic dispatch",
		"reflection",
		"Generated Go files",
		"Package load warnings lower confidence",
	)
}

func TestAnalyzePackageSharesSemanticContextForImpactAndSymbols(t *testing.T) {
	root := writeAgentContextProject(t)

	original := agentContextTestLoadSemanticContextRepository
	repositoryLoads := 0
	agentContextTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if !options.IncludeTests {
			repositoryLoads++
		}
		return original(root, options)
	}
	defer func() {
		agentContextTestLoadSemanticContextRepository = original
	}()

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
}

func TestAnalyzePackageNotesTestsOptionInLimitations(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if !agentContextLimitationsContain(report.Limitations, "--tests") {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
}

func TestAnalyzePackageUsesBuildTagsForContextSymbols(t *testing.T) {
	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package app

func Always() {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "enterprise.go"), `//go:build enterprise

package app

func Enterprise() {}
`)

	withoutTags, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzePackage without tags returned error: %v", err)
	}
	if withoutTags.AnalysisMode != AnalysisModeTypecheckedAST {
		t.Fatalf("analysis mode = %q, want %s", withoutTags.AnalysisMode, AnalysisModeTypecheckedAST)
	}
	if agentContextReportHasSymbol(withoutTags.Symbols, "Enterprise") {
		t.Fatalf("did not expect Enterprise without build tag, got %#v", withoutTags.Symbols)
	}

	withTags, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{BuildTags: []string{"enterprise"}})
	if err != nil {
		t.Fatalf("AnalyzePackage with tags returned error: %v", err)
	}
	if !agentContextReportHasSymbol(withTags.Symbols, "Enterprise") {
		t.Fatalf("expected Enterprise with build tag, got %#v", withTags.Symbols)
	}
}

func TestAnalyzePackageAppliesLimits(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{
		Limits: LimitOptions{
			MaxFiles:   1,
			MaxSymbols: 1,
			MaxTests:   1,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if len(report.Files) != 1 || report.Files[0] != "service.go" {
		t.Fatalf("expected service.go only, got %#v", report.Files)
	}
	if len(report.Symbols) != 1 || report.Symbols[0].Name != "Entry" {
		t.Fatalf("expected first symbol only, got %#v", report.Symbols)
	}
	if report.Truncated == nil || report.Truncated.Files != 1 || report.Truncated.Symbols != 3 {
		t.Fatalf("expected file/symbol truncation, got %#v", report.Truncated)
	}
}

func TestAnalyzePackageAppliesByteLimit(t *testing.T) {
	root := writeAgentContextProject(t)
	maxBytes := 2800

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{
		Limits: LimitOptions{
			MaxBytes: maxBytes,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if report.Limits == nil || report.Limits.MaxBytes != maxBytes {
		t.Fatalf("expected max bytes limit to be recorded, got %#v", report.Limits)
	}
	if report.Truncated == nil || report.Truncated.SourceContexts == 0 {
		t.Fatalf("expected source context byte truncation, got %#v", report.Truncated)
	}
	if size := encodedJSONLen(normalizePackageReport(report)); size > maxBytes && report.Truncated.ByteBudgetOverage == 0 {
		t.Fatalf("expected report to fit budget or report overage, size=%d budget=%d truncation=%#v", size, maxBytes, report.Truncated)
	}
}

func TestAnalyzePackageByteLimitPreservesCoreSignals(t *testing.T) {
	root := writeAgentContextProject(t)

	report, err := AnalyzePackage(root, ".", PackageAnalyzeOptions{
		Limits: LimitOptions{
			MaxBytes: 1400,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzePackage returned error: %v", err)
	}

	if len(report.Files) == 0 {
		t.Fatalf("expected byte-limited package context to keep at least one file, got %#v", report.Truncated)
	}
	if len(report.Symbols) == 0 {
		t.Fatalf("expected byte-limited package context to keep at least one symbol, got %#v", report.Truncated)
	}
	if len(report.AffectedPackages) == 0 {
		t.Fatalf("expected byte-limited package context to keep affected package signal, got %#v", report.Truncated)
	}
	if len(report.TestCommands) == 0 {
		t.Fatalf("expected byte-limited package context to keep a compact test command, got %#v", report.Truncated)
	}
}

func TestAnalyzeDiffBuildsAgentContext(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), `package auth

type Session struct{}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "session_test.go"), `package auth

import "testing"

func TestSession(t *testing.T) {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "api", "handler.go"), `package api

import "example.com/app/internal/auth"

var _ = auth.Session{}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "internal", "api", "handler_test.go"), `package api

import "testing"

func TestHandler(t *testing.T) {}
`)
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")

	writeAgentContextTestFile(t, filepath.Join(root, "internal", "auth", "session.go"), `package auth

type Session struct{}

func NewSession() Session {
	return Session{}
}
`)

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if report.Target != "HEAD" || report.Base != "HEAD" {
		t.Fatalf("target/base = %q/%q, want HEAD/HEAD", report.Target, report.Base)
	}
	if report.Purpose == "" {
		t.Fatal("expected purpose")
	}
	if report.AnalysisMode != AnalysisModeDiffTypechecked {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeDiffTypechecked)
	}
	if report.ReferenceAnalysisMode != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s", report.ReferenceAnalysisMode, sherpa.ReferenceAnalysisModeTypechecked)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if report.Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceMedium)
	}
	if report.Risk.Level != "medium" {
		t.Fatalf("risk level = %q, want medium", report.Risk.Level)
	}
	if len(report.ChangedFiles) != 1 || report.ChangedFiles[0] != "internal/auth/session.go" {
		t.Fatalf("expected changed session.go, got %#v", report.ChangedFiles)
	}
	if len(report.AffectedSymbols) != 1 || report.AffectedSymbols[0] != "NewSession" {
		t.Fatalf("expected NewSession affected symbol, got %#v", report.AffectedSymbols)
	}
	if len(report.ChangedSymbolDetails) != 1 {
		t.Fatalf("expected one changed symbol detail, got %#v", report.ChangedSymbolDetails)
	}
	if detail := report.ChangedSymbolDetails[0]; detail.Target != "./internal/auth.NewSession" || detail.Position.File != "internal/auth/session.go" || detail.Position.Line != 5 {
		t.Fatalf("unexpected changed symbol detail: %#v", detail)
	}
	if len(report.AffectedTests) != 2 {
		t.Fatalf("expected 2 affected tests, got %#v", report.AffectedTests)
	}
	if len(report.TestPlan.CallerPackages) != 1 || report.TestPlan.CallerPackages[0].Package != "./internal/api" {
		t.Fatalf("expected caller-package test plan item for ./internal/api, got %#v", report.TestPlan.CallerPackages)
	}
	if len(report.TestCommands) != 2 {
		t.Fatalf("expected 2 test commands, got %#v", report.TestCommands)
	}
	if len(report.ReadingOrder) != 4 {
		t.Fatalf("expected 4 reading order steps, got %#v", report.ReadingOrder)
	}
	if report.ReadingOrder[0].Title != "Changed symbol: ./internal/auth.NewSession" {
		t.Fatalf("expected changed symbol to lead reading order, got %#v", report.ReadingOrder)
	}
	if !agentContextLimitationsContain(report.Limitations, "Interface analysis used typechecked") {
		t.Fatalf("expected interface analysis limitation, got %#v", report.Limitations)
	}
	if !agentContextLimitationsContain(report.Limitations, "Test analysis used typechecked") {
		t.Fatalf("expected test analysis limitation, got %#v", report.Limitations)
	}
	assertAgentContextLimitationsContainAll(t, report.Limitations,
		"hunk-based",
		"dynamic dispatch",
		"reflection",
		"Generated Go files",
		"Package load or snapshot warnings lower confidence",
	)
}

func TestAnalyzeDiffUsesGoWorkWhenRootAlsoHasGoMod(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), `module example.com/root

go 1.24.4

require example.com/service v0.0.0
`)
	writeAgentContextTestFile(t, filepath.Join(root, "go.work"), `go 1.24.4

use (
	.
	./service
)
`)
	writeAgentContextTestFile(t, filepath.Join(root, "root.go"), `package root

import "example.com/service"

func Run(svc service.Service) error {
	process := svc.Process
	return process(service.Payload{})
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service", "go.mod"), "module example.com/service\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service", "service.go"), `package service

type Payload struct{}

type Processor interface {
	Process(Payload) error
}

type Service struct{}

func (Service) Process(Payload) error {
	return nil
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service", "service_test.go"), `package service

import "testing"

func TestServiceProcess(t *testing.T) {
	if (Service{}).Process(Payload{}) != nil {
		t.Fatal("unexpected error")
	}
}
`)
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")

	writeAgentContextTestFile(t, filepath.Join(root, "service", "service.go"), `package service

type Payload struct{}

type Processor interface {
	Process(Payload) error
}

type Service struct{}

func (Service) Process(Payload) error {
	_ = Payload{}
	return nil
}
`)

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeDiffTypechecked {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeDiffTypechecked, report.Warnings)
	}
	if report.ReferenceAnalysisMode != sherpa.ReferenceAnalysisModeTypechecked {
		t.Fatalf("reference analysis mode = %q, want %s", report.ReferenceAnalysisMode, sherpa.ReferenceAnalysisModeTypechecked)
	}
	if report.CallAnalysisMode != sherpa.CallAnalysisModeTypechecked {
		t.Fatalf("call analysis mode = %q, want %s", report.CallAnalysisMode, sherpa.CallAnalysisModeTypechecked)
	}
	if report.TestAnalysisMode != sherpa.TestAnalysisModeTypecheckedAST {
		t.Fatalf("test analysis mode = %q, want %s with warnings %#v", report.TestAnalysisMode, sherpa.TestAnalysisModeTypecheckedAST, report.Warnings)
	}
	if report.InterfaceAnalysisMode != impactengine.InterfaceAnalysisModeTypechecked {
		t.Fatalf("interface analysis mode = %q, want %s", report.InterfaceAnalysisMode, impactengine.InterfaceAnalysisModeTypechecked)
	}
	if len(report.ChangedFiles) != 1 || report.ChangedFiles[0] != "service/service.go" {
		t.Fatalf("expected changed service file, got %#v", report.ChangedFiles)
	}
	if len(report.ChangedPackages) != 1 || report.ChangedPackages[0] != "./service" {
		t.Fatalf("expected changed service package, got %#v", report.ChangedPackages)
	}
	if len(report.ChangedSymbolDetails) != 1 {
		t.Fatalf("expected one changed symbol detail, got %#v", report.ChangedSymbolDetails)
	}
	if detail := report.ChangedSymbolDetails[0]; detail.Target != "./service.Service.Process" || detail.Position.File != "service/service.go" {
		t.Fatalf("unexpected changed symbol detail: %#v", detail)
	}
	if !agentContextStringSliceContains(report.AffectedPackages, ".") || !agentContextStringSliceContains(report.AffectedPackages, "./service") {
		t.Fatalf("expected root and service affected packages, got %#v", report.AffectedPackages)
	}
	if !agentContextStringSliceContains(report.AffectedInterfaces, "./service.Processor") {
		t.Fatalf("expected service Processor interface, got %#v", report.AffectedInterfaces)
	}
	if !agentContextStringSliceContains(report.AffectedImplementations, "./service.Service") {
		t.Fatalf("expected service implementation signal, got %#v", report.AffectedImplementations)
	}
	if len(report.AffectedTests) != 1 || report.AffectedTests[0].Name != "TestServiceProcess" {
		t.Fatalf("expected service affected test, got %#v", report.AffectedTests)
	}
	if len(report.ReadingOrder) == 0 || report.ReadingOrder[0].Title != "Changed symbol: ./service.Service.Process" {
		t.Fatalf("expected changed workspace symbol to lead reading order, got %#v", report.ReadingOrder)
	}
}

func TestAnalyzeDiffSharesSemanticSessionForImpactAndTests(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() string {
	return Target()
}

func Target() string {
	return "old"
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service_test.go"), `package app

import "testing"

func TestTarget(t *testing.T) {
	if Target() == "" {
		t.Fatal("empty")
	}
}
`)
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")

	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() string {
	return Target()
}

func Target() string {
	return "new"
}
`)

	original := agentContextTestLoadSemanticContextRepository
	repositoryLoads := 0
	testRepositoryLoads := 0
	agentContextTestLoadSemanticContextRepository = func(root string, options semantics.LoadOptions) (semantics.Repository, error) {
		if options.IncludeTests {
			testRepositoryLoads++
		} else {
			repositoryLoads++
		}

		return original(root, options)
	}
	defer func() {
		agentContextTestLoadSemanticContextRepository = original
	}()

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeDiffTypechecked {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeDiffTypechecked, report.Warnings)
	}
	if report.TestAnalysisMode != sherpa.TestAnalysisModeTypecheckedAST {
		t.Fatalf("test analysis mode = %q, want %s", report.TestAnalysisMode, sherpa.TestAnalysisModeTypecheckedAST)
	}
	if repositoryLoads != 1 {
		t.Fatalf("expected one shared semantic repository load, got %d", repositoryLoads)
	}
	if testRepositoryLoads != 1 {
		t.Fatalf("expected one shared semantic test repository load, got %d", testRepositoryLoads)
	}
}

func TestAnalyzeDiffUsesValidSnapshotForCurrentChangedSymbols(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")

	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n\nfunc Added() {}\n")
	snapshot, err := snapshotstore.Build(root, snapshotstore.BuildOptions{})
	if err != nil {
		t.Fatalf("Build snapshot returned error: %v", err)
	}
	if _, err := snapshotstore.Write(root, snapshot); err != nil {
		t.Fatalf("Write snapshot returned error: %v", err)
	}

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{UseSnapshot: true})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeSnapshotDiffTypechecked {
		t.Fatalf("analysis mode = %q, want %s with warnings %#v", report.AnalysisMode, AnalysisModeSnapshotDiffTypechecked, report.Warnings)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("expected no snapshot warnings, got %#v", report.Warnings)
	}
	if len(report.ChangedSymbolDetails) != 1 || report.ChangedSymbolDetails[0].Name != "Added" {
		t.Fatalf("expected Added changed symbol from snapshot, got %#v", report.ChangedSymbolDetails)
	}
	if !agentContextLimitationsContain(report.Limitations, "reused a valid snapshot") {
		t.Fatalf("expected snapshot limitation, got %#v", report.Limitations)
	}
	assertAgentContextLimitationsContainAll(t, report.Limitations,
		"Generated Go files",
		"Package load or snapshot warnings lower confidence",
	)
}

func TestAnalyzeDiffFallsBackWhenSnapshotIsStale(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")

	snapshot, err := snapshotstore.Build(root, snapshotstore.BuildOptions{})
	if err != nil {
		t.Fatalf("Build snapshot returned error: %v", err)
	}
	if _, err := snapshotstore.Write(root, snapshot); err != nil {
		t.Fatalf("Write snapshot returned error: %v", err)
	}
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n\nfunc Added() {}\n")

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{UseSnapshot: true})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if report.AnalysisMode != AnalysisModeDiffTypechecked {
		t.Fatalf("analysis mode = %q, want %s", report.AnalysisMode, AnalysisModeDiffTypechecked)
	}
	if len(report.Warnings) == 0 || !strings.Contains(strings.Join(report.Warnings, "\n"), "snapshot not used: Snapshot is stale") {
		t.Fatalf("expected stale snapshot warning, got %#v", report.Warnings)
	}
	if report.Confidence != ConfidenceLow {
		t.Fatalf("confidence = %q, want %s", report.Confidence, ConfidenceLow)
	}
}

func TestAnalyzeDiffNotesTestsOptionInLimitations(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), "package app\n\nfunc Target() {}\n\nfunc Added() {}\n")

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{IncludeTests: true})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if !agentContextLimitationsContain(report.Limitations, "--tests") {
		t.Fatalf("expected --tests limitation note, got %#v", report.Limitations)
	}
}

func TestAnalyzeDiffAppliesLimits(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n\nfunc AddedFirst() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n\nfunc AddedSecond() {}\n")

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{
		Limits: LimitOptions{
			MaxFiles:   1,
			MaxSymbols: 1,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if len(report.ChangedFiles) != 1 {
		t.Fatalf("expected one changed file, got %#v", report.ChangedFiles)
	}
	if len(report.AffectedSymbols) != 1 {
		t.Fatalf("expected one affected symbol, got %#v", report.AffectedSymbols)
	}
	if report.Truncated == nil || report.Truncated.ChangedFiles != 1 || report.Truncated.AffectedSymbols != 1 {
		t.Fatalf("expected diff truncation, got %#v", report.Truncated)
	}
}

func TestAnalyzeDiffAppliesByteLimit(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n\nfunc AddedFirst() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n\nfunc AddedSecond() {}\n")

	maxBytes := 1200
	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{
		Limits: LimitOptions{
			MaxBytes: maxBytes,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if report.Limits == nil || report.Limits.MaxBytes != maxBytes {
		t.Fatalf("expected max bytes limit to be recorded, got %#v", report.Limits)
	}
	if report.Truncated == nil || report.Truncated.AffectedSymbols == 0 && report.Truncated.ChangedFiles == 0 {
		t.Fatalf("expected diff byte truncation, got %#v", report.Truncated)
	}
	if size := encodedJSONLen(normalizeDiffReport(report)); size > maxBytes && report.Truncated.ByteBudgetOverage == 0 {
		t.Fatalf("expected report to fit budget or report overage, size=%d budget=%d truncation=%#v", size, maxBytes, report.Truncated)
	}
}

func TestAnalyzeDiffByteLimitPreservesCoreSignals(t *testing.T) {
	root := initAgentContextGitRepository(t)

	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n")
	runAgentContextGit(t, root, "add", ".")
	runAgentContextGit(t, root, "commit", "-m", "initial")
	writeAgentContextTestFile(t, filepath.Join(root, "first.go"), "package app\n\nfunc First() {}\n\nfunc AddedFirst() {}\n")
	writeAgentContextTestFile(t, filepath.Join(root, "second.go"), "package app\n\nfunc Second() {}\n\nfunc AddedSecond() {}\n")

	report, err := AnalyzeDiff(root, "HEAD", DiffAnalyzeOptions{
		Limits: LimitOptions{
			MaxBytes: 1000,
		},
	})
	if err != nil {
		t.Fatalf("AnalyzeDiff returned error: %v", err)
	}

	if len(report.ChangedFiles) == 0 {
		t.Fatalf("expected byte-limited diff context to keep at least one changed file, got %#v", report.Truncated)
	}
	if len(report.ChangedPackages) == 0 {
		t.Fatalf("expected byte-limited diff context to keep changed package signal, got %#v", report.Truncated)
	}
	if len(report.ChangedSymbolDetails) == 0 {
		t.Fatalf("expected byte-limited diff context to keep at least one changed symbol detail, got %#v", report.Truncated)
	}
}

func writeAgentContextProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() {
	Target()
}

// Target handles the main service step.
func Target() {
	Helper()
}

func Helper() {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service_test.go"), `package app

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)

	return root
}

func writeAgentContextProjectWithPackageLoadWarning(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package app

func Entry() {
	Target()
}

func Target() {}

func Broken() {
	Missing()
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service_test.go"), `package app

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)

	return root
}

func writeAgentContextNestedModuleProject(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeAgentContextTestFile(t, filepath.Join(root, "go.mod"), "module example.com/root\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "service.go"), `package root

func Entry() {
	Target()
}

func Target() {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "service_test.go"), `package root

import "testing"

func TestTarget(t *testing.T) {
	Target()
}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "nested", "go.mod"), "module example.com/nested\n\ngo 1.24.4\n")
	writeAgentContextTestFile(t, filepath.Join(root, "nested", "service.go"), `package nested

func Entry() {
	Target()
}

func Target() {}
`)
	writeAgentContextTestFile(t, filepath.Join(root, "nested", "service_test.go"), `package nested

import "testing"

func TestNestedTarget(t *testing.T) {
	Target()
}
`)

	return root
}

func agentContextReportHasSymbol(symbols []sherpa.Symbol, name string) bool {
	for _, symbol := range symbols {
		if symbol.Name == name {
			return true
		}
	}

	return false
}

func agentContextRelatedTestsContain(tests []sherpa.RelatedTest, name string) bool {
	for _, test := range tests {
		if test.Name == name {
			return true
		}
	}

	return false
}

func agentContextWarningsContain(warnings []string, want string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, want) {
			return true
		}
	}

	return false
}

func agentContextStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}

	return false
}

func agentContextCallersContain(callers []sherpa.Caller, name string, file string) bool {
	for _, caller := range callers {
		if caller.Name == name && caller.Position.File == file {
			return true
		}
	}

	return false
}

func agentContextLimitationsContain(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}

	return false
}

func assertAgentContextLimitationsContainAll(t *testing.T, values []string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !agentContextLimitationsContain(values, want) {
			t.Fatalf("expected limitations to contain %q, got %#v", want, values)
		}
	}
}

func assertAgentContextReadingStepRange(t *testing.T, step explainengine.ReadingStep, file string, startLine int, startColumn int, endLine int, endColumn int) {
	t.Helper()

	if step.Range == nil {
		t.Fatalf("expected reading step %q to include a range", step.Title)
	}
	if step.Range.Start.File != file || step.Range.Start.Line != startLine || step.Range.Start.Column != startColumn {
		t.Fatalf("unexpected range start for %q: %#v", step.Title, step.Range.Start)
	}
	if step.Range.End.File != file || step.Range.End.Line != endLine || step.Range.End.Column != endColumn {
		t.Fatalf("unexpected range end for %q: %#v", step.Title, step.Range.End)
	}
}

func writeAgentContextTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func initAgentContextGitRepository(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	runAgentContextGit(t, root, "init")
	runAgentContextGit(t, root, "config", "user.email", "test@example.com")
	runAgentContextGit(t, root, "config", "user.name", "Test User")

	return root
}

func runAgentContextGit(t *testing.T, root string, args ...string) string {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}

	return string(output)
}
