# GoSherpa Priority Implementation Plan

This plan turns the current product priority into implementation slices that a
coding agent can execute without guessing the intended behavior.

The goal is not to add many new commands. The goal is to make the existing
GoSherpa workflow trustworthy enough that a developer or coding agent reaches
for it early in ordinary Go repository work.

Primary source documents:

- `docs/product/MUST_USE_READINESS.md`
- `docs/product/AGENT_PRIORITY_LIST.md`
- `docs/product/FEATURE_ROADMAP.md`
- `docs/product/AGENT_RECOMMENDATION_CRITERIA.md`
- `docs/STATUS.md`

Primary code contracts to preserve:

- JSON responses use the shared envelope in `cmd/gosherpa/cli_json.go`.
- Stable JSON examples live in `cmd/gosherpa/testdata/golden-json/`.
- Schema guardrails live in `cmd/gosherpa/json_schema_test.go` and
  `cmd/gosherpa/json_golden_test.go`.
- Command and flag support is defined by `cmd/gosherpa/command_registry.go`,
  `cmd/gosherpa/cli_flags.go`, and validation in `cmd/gosherpa/main.go`.

## Strict Review Corrections Incorporated

The following corrections are mandatory parts of this plan:

- Use `<base-ref>` in verification commands instead of assuming `main` exists.
  Pick an existing ref such as `origin/main`, `main`, or a documented test ref
  at the start of a slice.
- Do not broaden context limit flags blindly. Each context kind supports a
  specific flag set, listed in Phase 2.2.
- Do not describe method-specific type parameters as a Go fixture. Go supports
  generic functions, generic types, and methods on generic types, not methods
  with their own type parameter list.
- Treat `tests <target>` and `tests affected --base <ref>` as different CLI
  contracts. `--tags`, `--base`, and `--use-snapshot` apply to
  `tests affected`, not plain `tests <target>`.
- Do not imply snapshot reuse provides semantic relationship reuse. Current
  snapshot reuse covers inventory and current changed-symbol inventory only;
  relationship, impact, and call graph analysis still run live unless a slice
  explicitly implements and documents more.
- Do not introduce new confidence values casually. The current product surface
  uses `medium` and `low`; adding values such as `high` is a schema/docs/golden
  change.
- Keep generated-file behavior intentional. The current default should remain
  repository-native and conservative: include generated Go files unless an
  explicit documented policy says otherwise, and surface limitations when
  generated files may affect trust.

## Current Priority

GoSherpa already has useful MVP coverage: symbol lookup, search, references,
callers, callees, call paths, interface navigation, impact analysis, context
exports, structured test suggestions, PR summaries, JSON output, ambiguity
diagnostics, source ranges, `doctor`, and first-slice snapshot reuse.

The remaining gap is trust. The next implementation work should improve:

1. Typechecked loading everywhere it materially affects correctness.
2. Production-ready context export for agent workflows.
3. Structured test planning that clearly separates fast confidence from broad
   safety.

Everything below is ordered to support those three outcomes.

## Execution Rules For Agents

Before starting any slice:

- Read the relevant source files listed in the slice.
- Re-read command support in `cmd/gosherpa/command_registry.go` and flag
  validation in `cmd/gosherpa/main.go` before changing CLI behavior.
- Run or inspect the existing focused tests for that area.
- Pick a concrete `<base-ref>` for diff-oriented verification and record it.
- Keep human CLI output readable and stable.
- Keep JSON schemas, golden fixtures, and command output aligned.
- Preserve AST fallback behavior when typechecked loading is unavailable.
- Surface uncertainty as warnings, `analysisMode`, subanalysis modes,
  `confidence`, and `limitations`; do not silently drop failed analysis.
- Prefer narrow PR-sized slices over broad rewrites.

Every completed slice must include:

- Focused unit tests for the changed behavior.
- CLI or golden JSON coverage when command output changes.
- Documentation updates when JSON shape, user-visible behavior, limitations,
  readiness status, or flag support changes.
- A verification note listing exact commands run and the selected `<base-ref>`.

Recommended baseline verification:

```bash
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa context diff --base <base-ref> --json
```

Use narrower test commands while developing. Run `go test ./...` before a slice
is considered done unless the slice explicitly documents a pre-existing failure.

## Phase 0: Baseline Audit And Guardrails

Goal: make implementation safer by confirming the current output contract and
turning unknowns into concrete slice tasks before behavior changes.

Entry criteria:

- Working tree is understood.
- Existing tests pass, or current failures are documented before new edits.
- `<base-ref>` is selected and recorded for diff-oriented checks.

Tasks:

- [x] Run `go test ./...` and record any pre-existing failures.
- [x] Run `go run ./cmd/gosherpa doctor --json` and record readiness warnings.
- [x] Create or update `docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md` with:
      - selected `<base-ref>`
      - test status
      - command contract gaps
      - fixture gaps
      - schema/golden drift
      - open questions that block implementation
- [x] Compare current docs against golden JSON fixtures for these top
      agent-facing commands:
      - `context symbol`
      - `context file`
      - `context package`
      - `context diff`
      - `impact symbol`
      - `impact diff`
      - `tests`
      - `tests affected`
      - `pr`
- [x] List which commands already expose trust fields:
      `analysisMode`, `confidence`, `limitations`, envelope `warnings`, and
      subanalysis modes such as `referenceAnalysisMode`, `callAnalysisMode`,
      `interfaceAnalysisMode`, and `testAnalysisMode`.
- [x] List commands where warnings are present internally but missing, noisy,
      duplicated, or misplaced in human output.
- [x] Confirm command-specific flag support and unsupported-combination errors:
      - `--tags`
      - `--use-snapshot`
      - `--base`
      - context limit flags
      - `--scope`
- [x] Identify missing edge-case fixtures for:
      - `go.work`
      - nested modules
      - build tags
      - generated files
      - generic functions, generic types, and methods on generic types
      - type aliases
      - embedded interfaces
      - imported package selectors
      - method values
      - escaping or reassigned function values
      - duplicate package-qualified names

Primary files to inspect:

- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`
- `cmd/gosherpa/testdata/golden-json/`
- `docs/product/JSON_SCHEMA_V1.md`
- `docs/product/CONTEXT_SCHEMA_V1.md`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`

Exit criteria:

- The implementation audit exists and is concrete enough to drive PRs.
- Current test failures, if any, are documented before implementation begins.
- Missing fixtures are assigned to Phase 1, 2, or 3 tasks.
- No slice proceeds with an unknown command or flag contract.

## Phase 1: Typechecked Loading Everywhere It Matters

Goal: move high-risk analysis paths from AST-first heuristics toward shared
typechecked repository semantics while keeping conservative AST fallback.

Why this comes first:

Accuracy is the main trust gate. If GoSherpa misses a caller, reports the wrong
implementer, loses build-tag coverage, or confuses duplicate symbols, users and
agents fall back to manual search.

### Slice 1.1: Semantic Loader Diagnostics

Goal: make package-loading success, partial success, and failure visible and
reusable.

Tasks:

- [x] Review `semantics.LoadRepository` warning behavior for package load
      errors, type errors, partial packages, empty loads, workspace patterns,
      and cache fallback.
- [x] Keep hard errors for cases where no useful fallback result can be
      produced.
- [x] Ensure failed or partial package loads become structured warnings where
      fallback output can still be produced.
- [x] Ensure `SemanticContext.TypecheckedRepository` and
      `SemanticContext.TypecheckedTestRepository` expose consistent warnings
      for references, calls, interface analysis, test analysis, and context
      reports.
- [x] Ensure warning text is stable, relative to the selected root where
      possible, and deduplicated.
- [x] Add tests that simulate package load warnings and verify downstream
      result warnings and confidence behavior.

Primary files:

- `internal/semantics/repository.go`
- `internal/semantics/repository_cache.go`
- `internal/semantics/repository_test.go`
- `internal/sherpa/semantic_context.go`
- `internal/sherpa/semantic_context_test.go`
- `internal/sherpa/reference.go`
- `internal/sherpa/call.go`
- `internal/sherpa/test.go`
- `internal/impact/interface.go`
- `internal/agentcontext/report.go`

Exit criteria:

- Typechecked load problems are visible in result warnings.
- Fallback paths keep returning useful AST-backed results when possible.
- Confidence drops deterministically to the existing supported value when
  warnings, fallback modes, or partial semantic data materially reduce trust.
- Tests cover both hard-error and warning-with-fallback cases.

Verification:

```bash
go test ./internal/semantics ./internal/sherpa ./internal/impact ./internal/agentcontext
go test ./cmd/gosherpa
```

### Slice 1.2: Shared Semantic Context Adoption

Goal: prevent each analysis bundle from re-loading or re-computing semantic
facts inconsistently.

Tasks:

- [x] Trace each agent-facing command and record where it creates, reuses, or
      bypasses `sherpa.NewSemanticContext`.
- [x] Ensure these flows use shared semantic data where available:
      - `explain`
      - `context symbol`
      - `context file`
      - `context package`
      - `context diff`
      - `impact <target>`
      - `impact symbol`
      - `impact diff`
      - `tests affected`
      - `pr`
- [x] Keep command-level `analysisMode` separate from subanalysis modes:
      `referenceAnalysisMode`, `callAnalysisMode`, `interfaceAnalysisMode`, and
      `testAnalysisMode`.
- [x] Preserve build-tag option matching. If a semantic context was created
      with different tags than a subanalysis requests, return a clear error or
      create a correctly tagged context; do not reuse mismatched data.
- [x] Do not treat snapshot symbols as semantic relationship data. Snapshot
      reuse may seed inventory or current changed-symbol inventory only unless
      this slice explicitly adds a documented relationship snapshot contract.
- [x] Avoid changing JSON field names unless schema docs and golden fixtures
      are updated in the same slice.

Primary files:

- `cmd/gosherpa/command_handlers.go`
- `cmd/gosherpa/pr.go`
- `cmd/gosherpa/snapshot_reuse.go`
- `internal/sherpa/semantic_context.go`
- `internal/explain/report.go`
- `internal/agentcontext/semantic.go`
- `internal/agentcontext/report.go`
- `internal/agentcontext/file_report.go`
- `internal/agentcontext/package_report.go`
- `internal/agentcontext/diff_report.go`
- `internal/impact/report.go`
- `cmd/gosherpa/cli_json.go`

Exit criteria:

- Main context, impact, affected-test, and PR flows share semantic loading where
  their option sets match.
- Reused semantic data does not change ordering, determinism, or fallback
  behavior.
- Subanalysis trust fields describe only their own analysis.
- Snapshot and semantic-context responsibilities remain clearly separated.

Verification:

```bash
go test ./internal/sherpa ./internal/explain ./internal/agentcontext ./internal/impact
go test ./cmd/gosherpa
```

### Slice 1.3: Go Workspace And Nested Module Fixtures

Goal: make workspace and nested-module behavior explicit and tested.

Tasks:

- [x] Add fixtures for a repository with `go.work` and multiple modules.
- [x] Add fixtures for nested modules that must not be folded into the wrong
      package scope.
- [x] Verify symbol lookup, references, callers, context export, impact, and
      tests behave predictably across workspace package patterns.
- [x] Verify roots with direct `go.mod` and roots with direct `go.work`.
- [x] Document intentionally unsupported cross-module behavior as a limitation.

Primary files:

- `internal/semantics/repository.go`
- `internal/semantics/repository_test.go`
- `internal/sherpa/root.go`
- `internal/sherpa/workspace.go`
- `internal/sherpa/workspace_test.go`
- `internal/sherpa/reference_test.go`
- `internal/sherpa/call_test.go`
- `internal/agentcontext/report_test.go`
- `cmd/gosherpa/main_test.go`

Exit criteria:

- `go.work` package patterns are covered by tests.
- Nested module behavior is deterministic.
- Warnings explain unsupported or partially loaded workspaces.
- Docs state what cross-module behavior is supported today.

Verification:

```bash
go test ./internal/semantics ./internal/sherpa ./internal/agentcontext
go test ./cmd/gosherpa
```

### Slice 1.4: Build Tags And Generated Files

Goal: make build-tag and generated-file behavior reliable and visible.

Current CLI contract to preserve unless deliberately changed:

- `--tags` is supported by `analyze`, `refs`, `entrypoints`, `callers`,
  `callees`, `explain`, `context`, `impact`, `tests affected`, `implementers`,
  `interface`, `interfaces`, `pr`, `doctor`, and `snapshot`.
- Plain `tests <target>` does not support `--tags`.
- Unsupported `--tags` use must fail with a clear usage error.

Tasks:

- [x] Add or expand fixtures with symbols, callers, implementers, and tests
      behind build tags.
- [x] Verify `--tags` reaches every semantic path that claims support,
      including shared semantic contexts and downstream subanalyses.
- [x] Verify unsupported `--tags` combinations remain rejected intentionally.
- [x] Define generated-file policy in docs. Default target policy:
      repository-local generated Go files are included like compiler-visible
      Go files unless explicitly ignored by existing discovery rules; any
      generated-file trust caveat is surfaced as a limitation.
- [x] If generated-file detection is added, detect standard
      `// Code generated ... DO NOT EDIT.` headers without changing schema
      fields unless docs, tests, and goldens are updated together.
- [x] Make generated-file behavior consistent in symbol inventory, references,
      callers, interface impact, context reports, and tests.

Primary files:

- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/completion.go`
- `internal/semantics/repository.go`
- `internal/sherpa/scan.go`
- `internal/sherpa/parse.go`
- `internal/sherpa/semantic_context.go`
- `internal/sherpa/reference_test.go`
- `internal/sherpa/call_test.go`
- `internal/impact/interface_test.go`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`

Exit criteria:

- Build-tag behavior is covered for references, callers/callees, interfaces,
  context, impact, affected tests, PR output, doctor, and snapshot where
  applicable.
- Unsupported tag combinations are tested.
- Generated-file behavior is intentional, consistent, and documented.

Verification:

```bash
go test ./internal/semantics ./internal/sherpa ./internal/impact ./internal/agentcontext
go test ./cmd/gosherpa
```

### Slice 1.5: Go Semantic Edge Cases

Goal: improve correctness for ordinary modern Go code.

Tasks:

- [x] Add fixtures for generic functions, generic types, and methods on generic
      types.
- [x] Add fixtures for type aliases and package-qualified duplicate names.
- [x] Add fixtures for embedded interfaces, imported package selectors, and
      concrete method values.
- [x] Add fixtures for simple locally assigned function values that should be
      resolved and escaping or reassigned function values that should remain
      limited.
- [x] Verify references, callers, callees, implementers, satisfied interfaces,
      impact, and context reading order for those fixtures.
- [x] Preserve ambiguity diagnostics for unqualified duplicate targets.
- [x] Add or update limitations for dynamic dispatch, reflection, runtime
      wiring, escaping function values, and external dependency relationships
      that remain intentionally unresolved.

Primary files:

- `internal/sherpa/reference.go`
- `internal/sherpa/call.go`
- `internal/impact/interface.go`
- `internal/explain/report.go`
- `internal/agentcontext/report.go`
- `internal/sherpa/reference_test.go`
- `internal/sherpa/call_test.go`
- `internal/impact/interface_test.go`
- `internal/explain/report_test.go`
- `internal/agentcontext/report_test.go`
- `cmd/gosherpa/main_test.go`

Exit criteria:

- Modern Go edge cases are either correctly resolved or explicitly surfaced as
  limitations.
- Ambiguous unqualified names return candidates instead of guessing.
- Tests distinguish supported static cases from intentionally unsupported
  dynamic cases.

Verification:

```bash
go test ./internal/sherpa ./internal/impact ./internal/explain ./internal/agentcontext
go test ./cmd/gosherpa
```

Phase 1 is done when:

- Symbols, references, callers, callees, interfaces, context, impact,
  affected-test, and PR flows can use typechecked loading where it materially
  affects correctness.
- Package-load failures are warnings when fallback output is possible.
- AST fallback remains available and tested.
- Fixtures cover workspace, build, generated-file, and language edge cases.

## Phase 2: Production-Ready Agent Context Export

Goal: make `gosherpa context symbol|file|package|diff` the first command an
agent wants before editing.

Why this comes second:

Once semantic accuracy is stronger, context export becomes the stable product
surface. It must be bounded, deterministic, schema-aligned, and honest about
uncertainty.

### Slice 2.1: Context Contract Audit

Goal: define one consistent contract for all context variants without forcing
fields onto variants where they do not belong.

Shared context requirements:

- Envelope warnings live in `warnings`.
- Result limitations live in `data.limitations`.
- Every context variant exposes `analysisMode`, `confidence`, `testPlan`,
  `readingOrder`, and bounded-output metadata when limits are active.
- Subanalysis modes are present only when the corresponding subanalysis is
  included.
- Empty collections serialize as stable empty arrays where the schema expects
  arrays.

Context field matrix:

| Context kind | Required identity fields | Relationship fields |
| --- | --- | --- |
| `context symbol` | `target`, `identity`, `symbol`, `sourceContext` | references, callers, callees, affected packages, interface signals, related tests |
| `context file` | `target`, `file`, `package`, `symbols`, `sourceContexts` | affected packages, interface signals, affected tests |
| `context package` | `target`, `package`, `files`, `symbols`, `sourceContexts` | affected packages, interface signals, affected tests |
| `context diff` | `target`, `base`, changed files, changed packages, changed symbols | affected packages, affected symbols, interface signals, affected tests |

Tasks:

- [x] Compare `Report`, `FileReport`, `PackageReport`, and `DiffReport`.
- [x] Verify each context kind follows the matrix above.
- [x] Decide which fields are intentionally omitted for each context kind and
      document why in schema docs.
- [x] Do not add references/callers/callees to file, package, or diff context
      unless a slice explicitly defines the scope, limits, output order, and
      schema contract.
- [x] Ensure `readingOrder` never points to omitted or inconsistent data after
      limits are applied.

Primary files:

- `internal/agentcontext/report.go`
- `internal/agentcontext/file_report.go`
- `internal/agentcontext/package_report.go`
- `internal/agentcontext/diff_report.go`
- `internal/agentcontext/output.go`
- `internal/agentcontext/diff_output.go`
- `docs/product/CONTEXT_SCHEMA_V1.md`
- `docs/product/JSON_SCHEMA_V1.md`

Exit criteria:

- Context variants have a documented shared field contract.
- Intentional differences between symbol, file, package, and diff context are
  explicit.
- No context kind overpromises relationship data it does not compute.

Verification:

```bash
go test ./internal/agentcontext
go test ./cmd/gosherpa
```

### Slice 2.2: Limits And Byte Budgets

Goal: make context output safe to paste into agent workflows.

Supported context limit matrix:

| Context command | Supported limits |
| --- | --- |
| `context symbol` | `--max-references`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context file` | `--max-symbols`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context package` | `--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context diff` | `--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes` |

Unsupported combinations must remain clear usage errors, for example
`context diff --source-radius`, `context diff --max-references`,
`context file --max-references`, and `context symbol --max-files`.

Tasks:

- [x] Verify every supported limit in the matrix above.
- [x] Add tests for unsupported combinations and exact error messages where
      stable.
- [x] Ensure truncation metadata is deterministic and reports what was removed.
- [x] Add tests for tiny byte budgets and entry-count limits.
- [x] Ensure limits do not produce invalid JSON, nil slices where empty arrays
      are expected, or broken reading order.
- [x] Keep source excerpts useful under tight budgets for symbol, file, and
      package context.

Primary files:

- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/command_registry.go`
- `internal/agentcontext/limits.go`
- `internal/agentcontext/byte_limit.go`
- `internal/agentcontext/report_test.go`
- `internal/agentcontext/output_test.go`
- `cmd/gosherpa/main_test.go`
- `cmd/gosherpa/json_golden_test.go`

Exit criteria:

- Every documented context limit works consistently for its context kind.
- Unsupported context limit combinations fail intentionally.
- Truncated output remains valid, deterministic, and useful.

Verification:

```bash
go test ./internal/agentcontext
go test ./cmd/gosherpa
```

### Slice 2.3: Confidence, Limitations, And Warnings

Goal: make every context result explain how much to trust it.

Tasks:

- [x] Align confidence calculation for context results without changing the set
      of supported confidence values unless docs, schema tests, and goldens are
      updated in the same slice.
- [x] Ensure confidence drops to `low` when warnings, AST fallback modes,
      missing source context, or partial loads materially reduce trust.
- [x] Ensure limitations mention relevant blind spots:
      - AST fallback
      - dynamic dispatch
      - reflection
      - generated-file policy
      - skipped test-file callers unless `--tests` is enabled
      - package load failures
      - hunk-based diff limitations
      - snapshot reuse scope
- [x] Ensure human output shows concise warnings without terminal noise.
- [x] Ensure JSON warnings stay on the envelope and data limitations stay in
      the data object.
- [x] Ensure warnings are deduplicated, root-relative where possible, and do not
      leak temporary cache paths unless necessary to diagnose the issue.

Primary files:

- `internal/agentcontext/report.go`
- `internal/agentcontext/file_report.go`
- `internal/agentcontext/package_report.go`
- `internal/agentcontext/diff_report.go`
- `internal/agentcontext/output.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/main_test.go`
- `docs/product/CONTEXT_SCHEMA_V1.md`
- `docs/product/JSON_SCHEMA_V1.md`

Exit criteria:

- Context results never look overconfident when analysis is partial.
- Human and JSON output both expose uncertainty in the right place.
- Confidence semantics are documented and tested.

Verification:

```bash
go test ./internal/agentcontext ./cmd/gosherpa
```

### Slice 2.4: Golden JSON And Schema Alignment

Goal: prevent drift between implementation, fixtures, and docs.

Tasks:

- [x] Update golden JSON only after behavior is intentionally changed.
- [x] Ensure `json_schema_test.go` covers context fields and required trust
      fields.
- [x] Ensure `CONTEXT_SCHEMA_V1.md` and `JSON_SCHEMA_V1.md` match actual output.
- [x] Add regression coverage for ambiguous targets returning candidates
      instead of guessed output.
- [x] Add regression coverage for empty-but-valid context results where
      applicable.
- [x] Keep schema version at `1` for compatible additive changes; bump only for
      breaking JSON contract changes.

Primary files:

- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`
- `cmd/gosherpa/testdata/golden-json/`
- `docs/product/CONTEXT_SCHEMA_V1.md`
- `docs/product/JSON_SCHEMA_V1.md`

Exit criteria:

- Golden fixtures, schema tests, and docs agree.
- A future agent can detect contract drift by running tests.

Verification:

```bash
go test ./cmd/gosherpa
go test ./...
```

Phase 2 is done when:

- Context output is bounded, deterministic, and safe for agent prompts.
- Ambiguous targets return candidates.
- Every context result explains confidence and blind spots.
- JSON schemas and golden fixtures are aligned.

## Phase 3: Structured Test Planning

Goal: make test recommendations explain the tradeoff between quick confidence
and broader safety.

Why this comes third:

After GoSherpa can identify code and impact more reliably, the next highest
value is telling users exactly which tests to run first and why.

### Slice 3.1: Test Plan Contract Hardening

Goal: lock down the existing test plan shape and semantics.

Required JSON groups:

- `direct`: tests with direct references to changed or inspected symbols.
- `related`: tests in the same target package or otherwise close to the target.
- `contracts`: tests covering affected interfaces or implementations.
- `callerPackages`: tests in packages that call affected symbols.
- `fallback`: broader commands when direct evidence is empty or incomplete.

Current CLI contract to preserve unless deliberately changed:

- `tests <target>` supports `--scope direct|related|all`.
- `tests <target>` does not support `--base`, `--tags`, or `--use-snapshot`.
- `tests affected --base <ref>` supports `--tags` and `--use-snapshot`.
- `tests affected --base <ref>` does not support `--scope`.

Tasks:

- [x] Confirm all command outputs use the same `sherpa.TestPlan` shape.
- [x] Ensure every `TestPlanItem` includes a command and reason.
- [x] Ensure package, `test`, `tests`, and `targets` are included when known and
      omitted only when genuinely unknown.
- [x] Ensure empty direct results still produce practical fallback commands.
- [x] Preserve non-nil arrays for all test plan groups.
- [x] Document group semantics and CLI differences in schema docs and CLI docs.

Primary files:

- `internal/sherpa/test_plan.go`
- `internal/sherpa/test_plan_test.go`
- `internal/sherpa/test.go`
- `internal/sherpa/test_output.go`
- `internal/impact/report.go`
- `internal/agentcontext/report.go`
- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_json.go`
- `docs/CLI_REFERENCE.md`
- `docs/product/JSON_SCHEMA_V1.md`

Exit criteria:

- Test plan groups have stable, documented semantics.
- Every suggested command has a reason.
- Empty direct evidence does not leave the user with no practical next command.
- Plain `tests` and `tests affected` remain distinct and tested.

Verification:

```bash
go test ./internal/sherpa ./internal/impact ./internal/agentcontext ./cmd/gosherpa
```

### Slice 3.2: Diff-Based Test Planning

Goal: improve `tests affected --base`, `context diff`, `impact diff`, and `pr`
for changed-symbol workflows.

Tasks:

- [x] Ensure changed symbols are attached to test plan targets when known.
- [x] Ensure deleted symbols, hunk-only changes, and non-Go file changes still
      produce useful fallback recommendations.
- [x] Distinguish package-level fallback from whole-repo fallback.
- [x] Add tests for diffs that affect:
      - one function
      - one type with methods
      - an interface contract
      - multiple packages
      - no directly referenced tests
      - only non-Go files
- [x] Ensure PR human output remains short enough for review while JSON keeps
      the full structure.
- [x] Verify snapshot fallback warnings for diff-oriented commands using
      `--use-snapshot`.

Primary files:

- `internal/impact/diff.go`
- `internal/impact/report.go`
- `internal/impact/report_output.go`
- `internal/agentcontext/diff_report.go`
- `internal/agentcontext/diff_output.go`
- `cmd/gosherpa/pr.go`
- `cmd/gosherpa/command_handlers.go`
- `cmd/gosherpa/main_test.go`
- `cmd/gosherpa/json_golden_test.go`

Exit criteria:

- Diff-oriented commands include changed-symbol targets in test plan items when
  known.
- Fallbacks are useful when direct tests are empty.
- Human PR output remains concise.
- Snapshot fallback scope is visible and not overclaimed.

Verification:

```bash
go test ./internal/impact ./internal/agentcontext ./cmd/gosherpa
```

### Slice 3.3: Contract And Caller-Package Test Coverage

Goal: make interface and caller-package test recommendations more precise.

Tasks:

- [x] Verify affected interfaces and implementations feed `contracts`.
- [x] Verify transitive callers feed `callerPackages` without duplicating
      `direct`, `related`, or `contracts`.
- [x] Add fixtures where interface implementers live in different packages from
      interface definitions.
- [x] Add fixtures where caller-package tests are the only useful narrow test
      recommendation.
- [x] Ensure reasons clearly explain why a package appears in `contracts` or
      `callerPackages`.
- [x] Ensure deduplication preserves deterministic group order.

Primary files:

- `internal/impact/interface.go`
- `internal/impact/report.go`
- `internal/sherpa/test.go`
- `internal/sherpa/test_plan.go`
- `internal/sherpa/test_test.go`
- `internal/sherpa/test_plan_test.go`
- `internal/agentcontext/report_test.go`

Exit criteria:

- Interface and implementation signals produce contract recommendations when
  suitable tests exist.
- Caller-package recommendations are deduplicated and explained.
- Group ordering is deterministic.

Verification:

```bash
go test ./internal/impact ./internal/sherpa ./internal/agentcontext
go test ./cmd/gosherpa
```

### Slice 3.4: Output Polish And Documentation

Goal: make test planning easy to consume in both terminal and JSON workflows.

Tasks:

- [x] Review human output for `tests`, `tests affected`, `context diff`,
      `impact diff`, and `pr`.
- [x] Ensure human output shows grouped recommendations without becoming noisy.
- [x] Ensure JSON output keeps full structure and stable arrays.
- [x] Update `docs/CLI_REFERENCE.md`, `docs/STATUS.md`, and schema docs.
- [x] Update golden JSON fixtures only when behavior changed intentionally.
- [x] Verify examples use an existing `<base-ref>` placeholder or documented
      real ref consistently.

Primary files:

- `internal/sherpa/test_output.go`
- `internal/sherpa/impact_output.go`
- `internal/impact/report_output.go`
- `internal/agentcontext/output.go`
- `internal/agentcontext/diff_output.go`
- `cmd/gosherpa/pr.go`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`
- `docs/product/JSON_SCHEMA_V1.md`

Exit criteria:

- Terminal output helps a human choose a test strategy quickly.
- JSON output gives agents complete structured commands and reasons.
- Docs, goldens, and tests agree.

Verification:

```bash
go test ./internal/sherpa ./internal/impact ./internal/agentcontext ./cmd/gosherpa
go test ./...
```

Phase 3 is done when:

- JSON distinguishes direct, related, contract, caller-package, and fallback
  commands.
- Each command includes a reason.
- Diff-based recommendations include changed symbols when known.
- Empty direct results still provide practical fallback commands.
- CLI docs clearly distinguish plain `tests` from `tests affected`.

## Final Integration Pass

Goal: ensure the three priority tracks work together as one product workflow.

Tasks:

- [x] Select and record `<base-ref>`.
- [x] Run the must-use command set against GoSherpa itself:

      ```bash
      go run ./cmd/gosherpa context diff --base <base-ref>
      go run ./cmd/gosherpa context diff --base <base-ref> --json
      go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --json
      go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests
      go run ./cmd/gosherpa tests affected --base <base-ref> --json
      go run ./cmd/gosherpa pr --base <base-ref> --json
      ```

- [x] Verify the outputs answer:
      - what changed or what is inspected
      - where relevant code lives
      - who calls it and what it calls
      - which interfaces or implementers are involved
      - which packages and tests may be affected
      - what uncertainty remains
      - which command should run next
- [x] Run `go test ./...`.
- [x] Update `docs/STATUS.md` with completed readiness improvements.
- [x] Update `docs/product/MUST_USE_READINESS.md` if the readiness estimate or
      priority order changes.
- [x] Update `README.md` only if user-facing workflows or examples materially
      change. Reviewed; no README update was needed because existing examples
      remain valid.

Final integration verification:

- Selected `<base-ref>`: `origin/main`.
- `go run ./cmd/gosherpa context diff --base origin/main` passed.
- `go run ./cmd/gosherpa context diff --base origin/main --json` passed with
  schema version 1 and no envelope warnings.
- `go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --json`
  passed with symbol identity, relationships, test plan, trust fields, and no
  envelope warnings.
- `go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests` passed and
  reported affected packages, tests, grouped recommendations, and limitations.
- `go run ./cmd/gosherpa tests affected --base origin/main --json` passed with
  schema version 1, grouped test plan output, fallback recommendations, trust
  fields, and no envelope warnings.
- `go run ./cmd/gosherpa pr --base origin/main --json` passed with schema
  version 1, changed files/packages/symbols, affected tests, verification
  commands, trust fields, and no envelope warnings.
- `go test ./...` passed.

Exit criteria:

- The top context, impact, tests, and PR commands feel coherent together.
- Trust fields are visible and deterministic.
- The tool remains conservative instead of overconfident.
- Final docs and examples use valid, documented command contracts.

## Out Of Scope Until These Phases Are Done

Do not prioritize these ahead of the three tracks above:

- MCP or long-running server mode.
- Editor integration hooks.
- TUI.
- Graph export.
- Broad architecture reports beyond current slices.
- GitHub Action.
- More standalone commands that do not improve context, impact, test planning,
  or PR review trust.
- Full SSA, pointer analysis, reflection analysis, or runtime inference.

These can be valuable later, but they should follow semantic accuracy, context
quality, and structured test planning.

## Suggested PR Slice Order

1. Baseline audit and fixture gap list.
2. Semantic loader diagnostics.
3. Shared semantic context adoption for one command family.
4. Shared semantic context adoption for remaining agent-facing flows.
5. Workspace and nested-module fixtures.
6. Build-tag and generated-file fixtures.
7. Generic, alias, embedded-interface, selector, and method-value fixtures.
8. Context contract alignment.
9. Context limit and byte-budget hardening.
10. Confidence, limitations, and warning consistency.
11. Golden JSON and schema alignment.
12. Test plan contract hardening.
13. Diff-based test planning.
14. Contract and caller-package test coverage.
15. Human output and docs polish.
16. Final integration pass.

Each PR should keep behavior changes, docs, and tests together. If a slice
uncovers a deeper issue, record it as a follow-up rather than expanding the PR
without a clear exit criterion.
