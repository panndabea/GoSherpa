# Priority Implementation Audit

## Priority 1 First-Run Defaults Update

Date: 2026-07-19

Selected `<base-ref>` for verification: `HEAD`.
`git rev-parse --verify HEAD` resolved to
`e8e28ba002532e384f81420ba8ed78cfb1a12f1f`.

Priority 1 from `PRIORITY_IMPLEMENTATION_PLAN.md` is implemented as an
init-first workflow. `gosherpa init` now verifies or detects a local base ref,
writes `.gosherpa/config.json`, refreshes `.gosherpa/snapshot.json`, reports
warnings through the standard JSON envelope, and prints next-command guidance.

Behavior added:

- `.gosherpa/config.json` stores `schemaVersion`, `baseRef`, `useSnapshot`,
  normalized build tags, and `agentContext` limits.
- `agent context`, `pr`, and `tests affected` load saved defaults when explicit
  CLI flags are omitted.
- Explicit CLI values still win field-by-field, including per-limit agent
  context overrides.
- `--no-use-snapshot` disables saved snapshot reuse for config-aware commands.
- Missing-base errors now point users to `gosherpa init --base <ref>`.
- Existing unknown config fields are preserved when the prior config is valid;
  invalid config is replaced with normalized defaults and warnings.
- The composite GitHub Action now runs `gosherpa init --base <ref> --json`,
  uploads `init.json`, and then uses the shorter config-aware `agent context`,
  `pr`, and `tests affected` commands.

Verification commands:

```bash
go test ./internal/config ./internal/git
go test . -run 'TestGitHubActionDefinesPRIntelligenceWorkflow|TestDogfoodWorkflowUsesLocalActionWithFullHistory' -count=1 -v
go test ./cmd/gosherpa
go test ./cmd/gosherpa -run 'TestParseCLIArgsAcceptsNoUseSnapshotFlag|TestParseCLIArgsRejectsConflictingSnapshotFlags|TestMainRunsInitCommandAsJSON|TestMainRejectsNoUseSnapshotFlagOutsideConfigAwareCommands|TestMainAgentContextUsesConfigDefaultsAndPerFieldOverrides|TestMainPRAndTestsAffectedUseConfigBase|TestMainConfigWarningsReachJSONEnvelope' -count=1 -v
go test ./...
go build ./cmd/gosherpa
go run ./cmd/gosherpa --root /private/tmp/gosherpa-init-verify.xc71pu init --base HEAD --json
go run ./cmd/gosherpa --root /private/tmp/gosherpa-init-verify.xc71pu agent context --json
go run ./cmd/gosherpa --root /private/tmp/gosherpa-init-verify.xc71pu pr --json
go run ./cmd/gosherpa --root /private/tmp/gosherpa-init-verify.xc71pu tests affected --json
go run ./cmd/gosherpa --root /private/tmp/gosherpa-init-verify.xc71pu agent context --no-use-snapshot --json
```

Results:

- Focused config and git base-detection tests passed.
- Focused GitHub Action structure tests passed.
- Focused command parser and config-aware command tests passed.
- Full `go test ./...` passed.
- `go build ./cmd/gosherpa` passed.
- Manual smoke verification passed on a temporary Go module with an initialized
  config, a refreshed snapshot, a changed Go file, config-derived base refs, and
  explicit snapshot opt-out.

## Tests And Entrypoint Intelligence Slice 3.1 Update

Date: 2026-07-18

Selected `<base-ref>` for verification: `HEAD`.
`git rev-parse --verify HEAD` resolved to
`e3d161ed15c284617dcf80190b2f78a4104e582f`.

Slice 3.1 added an in-memory reusable test inventory in `internal/sherpa`.
`BuildTestInventory` records test packages, test files, top-level test
functions, statically visible literal subtests, conservative suite-like
`Test*` methods, source ranges, and limitations for dynamic table-driven
subtest names. Existing `FindTests` paths now collect parsed test files through
that inventory, preserving the public related-test and test-plan output shape.

Behavior preserved:

- Direct, related, contract, caller-package, and fallback test-plan groups stay
  unchanged.
- Public `RelatedTest` records continue to carry target-specific
  `targetReferences` with source ranges when parser or typechecker positions
  identify the reference.
- Snapshot format and compatibility metadata did not change. The inventory is
  intentionally in-memory for this slice.

Verification commands:

```bash
git rev-parse --verify HEAD
go test ./internal/sherpa ./internal/snapshot ./internal/agentworkflow ./cmd/gosherpa
go test ./...
```

Results:

- Focused tests passed.
- Full `go test ./...` passed.

## Real-World Repository Robustness Slice 2.5 Update

Date: 2026-07-18

Selected `<base-ref>` for verification: `HEAD`.
`git rev-parse --verify HEAD` resolved to
`e3d161ed15c284617dcf80190b2f78a4104e582f`.

Slice 2.5 aligned the documentation with the real-world repository robustness
behavior now implemented in slices 2.1 through 2.4. No command behavior or JSON
shape changed in this slice.

Documentation updates:

- `docs/STATUS.md` now has a repository-shape support matrix for single
  modules, workspaces, parent workspace visibility, nested modules, local
  replacements, build tags, generated Go files, partial package loading, and
  large repositories.
- `docs/CLI_REFERENCE.md` now includes concrete `--root`, nested-module
  inspection, `--tags`, and snapshot freshness examples.
- `AGENT_NOTES.md` and `llms.txt` now tell agents how to interpret skipped
  modules, external workspace modules, local replacements, tag-dependent
  snapshot staleness, and package-load diagnostics.
- `cmd/gosherpa/testdata/README.md` records why the permanent command fixtures
  are nested below the main module and which temporary fixture cases are
  generated by focused tests.

Verification commands:

```bash
git rev-parse --verify HEAD
go test ./cmd/gosherpa
go test ./...
```

Results:

- `go test ./cmd/gosherpa` passed.
- `go test ./...` passed.
- The slice intentionally did not refresh snapshots or alter runtime behavior.

## Real-World Repository Robustness Slice 2.3 Update

Date: 2026-07-18

Selected `<base-ref>` for diff-oriented verification: `HEAD`.
`git rev-parse --verify HEAD` resolved to
`48479227159562076eb001e9280a9900646ad610`.

Slice 2.3 centralized generated-file detection in `internal/sherpa` using the
standard `// Code generated ... DO NOT EDIT.` marker. Repository layout now
reports deterministic generated-file counts and bounded major generated package
summaries with root-relative package directory, package name, file count, total
bytes, and largest generated file evidence.

Behavior added:

- `doctor.data.repository.generatedPackages` and
  `agent context.data.readiness.repositoryLayout.generatedPackages` expose the
  shared layout summary.
- `agent context.data.readiness.generatedPackages` repeats the bounded summary
  directly in readiness for agent triage.
- Large generated file steps in `agent context.data.readingOrder` are grouped
  after hand-written file steps when they would otherwise dominate the first
  reading pass.
- Compiler-visible generated files remain included in semantic analysis; no
  exclusion flag was added.

Verification commands run during implementation:

```bash
git rev-parse --verify HEAD
go test ./internal/sherpa ./internal/agentworkflow ./cmd/gosherpa
go test ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa agent context --base HEAD --max-files 5 --max-symbols 5 --max-tests 5 --max-bytes 12000 --json
```

Results:

- `internal/sherpa`, `internal/agentworkflow`, and `cmd/gosherpa` passed after
  the additive JSON golden update.
- Full `go test ./...` passed.
- `doctor --json` passed with `generatedFiles: 0`,
  `generatedPackages: []`, and the existing skipped nested fixture-module
  warning.
- `agent context --base HEAD ... --json` passed with matching readiness
  generated-file fields, bounded changed-file output, and expected
  `sectionTruncation` from the requested limits.
- Generated definitions, references, interface usage, and tests remain covered
  by `TestMainIncludesGeneratedGoFilesConsistently`.
- New focused tests cover generated header detection, deterministic major
  package summaries, and large generated reading-order summarization in the
  agent workflow.

## Real-World Repository Robustness Slice 2.1 Update

Date: 2026-07-18

Selected `<base-ref>` for the diff-oriented verification command: `HEAD`.

`git rev-parse --verify HEAD` resolved to
`35747a3083af66b45d80abeae02e377c7b534b72`.

Slice 2.1 implemented a shared repository layout summary in
`internal/sherpa`. `doctor` now exposes that layout as `data.repository`, and
`agent context` exposes the same bounded shape at
`data.readiness.repositoryLayout` while retaining its previous short readiness
fields. The layout records the selected analysis boundary, root or parent
`go.work`, workspace modules, skipped workspace modules outside `--root`,
nested modules, skipped nested modules, local `replace` directives, and file
counts.

Behavior recorded for this repository after the slice:

- `doctor --json` reports `analysisBoundary: "module"`.
- The root file walk covers 127 Go files, including 50 test files and no
  generated files.
- The fixture modules under `cmd/gosherpa/testdata/interface_project`,
  `cmd/gosherpa/testdata/json_project`, and
  `cmd/gosherpa/testdata/possible_calls_project` are now reported as skipped
  nested modules with guidance to inspect them using separate `--root` values.
- Package loading still succeeds with 11 repository-local packages and
  `analysisMode: "typechecked"`.
- The existing snapshot was stale because the working tree changed; this slice
  did not refresh it because snapshot reuse was not the verification target.

Verification commands run:

```bash
git rev-parse --verify HEAD
go test ./internal/sherpa ./internal/semantics ./internal/agentworkflow ./cmd/gosherpa
go test ./internal/agentworkflow
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa agent context --base HEAD --max-files 5 --max-symbols 5 --max-tests 5 --max-bytes 12000 --json
```

Results:

- Focused tests passed for `internal/sherpa`, `internal/semantics`,
  `internal/agentworkflow`, and `cmd/gosherpa`.
- Full `go test ./...` passed.
- `doctor --json` passed with one expected envelope warning for skipped nested
  fixture modules and a stale-snapshot suggestion.
- `agent context --base HEAD ... --json` passed with one repository-layout
  warning, `readiness.confidence: "low"` due to that warning, and explicit
  `sectionTruncation` from the requested byte and item limits.

## Agent Workflow Phase 0 And Slice 1.1/1.2 Update

Date: 2026-07-17

This section records the active plan from `PRIORITY_IMPLEMENTATION_PLAN.md`:
zero-friction agent workflow first, then real-world repository robustness, then
tests and entrypoint intelligence. The older relationship-reuse,
possible-call, and target-risk entries below are preserved as completed
baseline evidence, not as the active phase map.

### Selected Base Ref

Selected `<base-ref>`: `origin/main`

`git rev-parse --verify origin/main` resolved to
`ab46bbc3abfa372d5ca2e7fdc5f8b78d1383ac06`. Local `main` resolved to the same
commit at audit time.

### Baseline Verification

Commands run before feature edits:

```bash
git rev-parse --verify origin/main
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa context diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa impact diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa tests affected --base origin/main --use-snapshot --json
go run ./cmd/gosherpa pr --base origin/main --use-snapshot --json
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
```

Results:

| Command | Status | Runtime | Envelope warnings | Analysis mode | Confidence |
| --- | --- | ---: | --- | --- | --- |
| `go test ./...` | passed | 2.05s, cached | n/a | n/a | n/a |
| `doctor --json` | passed | 4.99s | none | `typechecked` | `medium` |
| `snapshot --json` | passed | 23.30s | none | n/a | n/a |
| `context diff --base origin/main --use-snapshot --json` | passed | 2.31s | none | `snapshot+git-diff+ast` | `medium` |
| `impact diff --base origin/main --use-snapshot --json` | passed | 2.28s | none | `snapshot+git-diff+ast` | `medium` |
| `tests affected --base origin/main --use-snapshot --json` | passed | 2.29s | none | `snapshot+git-diff+ast` | `medium` |
| `pr --base origin/main --use-snapshot --json` | passed | 2.66s | none | `snapshot+git-diff+ast` | `medium` |
| `context symbol ./internal/sherpa.PlanTests --use-snapshot --json` | passed | 10.54s | none | `typechecked+ast` | `medium` |

`doctor --json` initially reported a stale snapshot with stale reason
`git state changed`. `snapshot --json` refreshed `.gosherpa/snapshot.json` to a
valid format v2 snapshot with 122 files, 10 packages, 2581 symbols, and
relationship metadata marked `present: true` and `capable: true`.

Diff-oriented commands used a valid snapshot after refresh. Because
`origin/main` equaled `main` at the baseline point, changed files, changed
packages, affected packages, affected symbols, affected tests, test commands,
and reading order were empty. The diff target risk was `low` / `local` with
reason `missing-direct-tests`.

The focused symbol drill-down for `./internal/sherpa.PlanTests` reported:

- `analysisMode: typechecked+ast`
- `referenceAnalysisMode: snapshot+typechecked`
- `callAnalysisMode: snapshot+typechecked`
- `interfaceAnalysisMode: typechecked`
- `testAnalysisMode: typechecked+ast`
- `targetRisk.level: high`
- `targetRisk.scope: exported-api`
- truncation for source lines, callees, related tests, and test-plan items

### Locked Command Contract

The first public agent workflow command is locked as:

```bash
gosherpa agent context --base <base-ref> [--use-snapshot] [--tags <list>] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--max-bytes <n>] [--json]
```

Global `--root <path>` applies through the existing root-selection mechanism.
The command is diff-first only. It rejects symbol, file, package, or free-form
positional targets, and it rejects inherited flags whose semantics are not part
of the first contract: `--tests`, `--scope`, `--max-references`,
and `--source-radius`.

### CLI Flag Matrix Changes

Before this slice, no top-level `agent` command existed. The implementation
adds `agent` to `command_registry.go`, `main.go` help/validation,
`cli_flags.go` parsing compatibility, and `completion.go`.

Validation messages that changed:

```text
--base is only supported by context diff, impact diff, tests affected, pr, and agent context
--use-snapshot is only supported by analyze, symbols, symbol, search, packages, refs, callers, callees, implementers, interface, interfaces, context symbol, context diff, impact symbol, impact diff, tests affected, pr, and agent context
--tags is only supported by analyze, refs, entrypoints, callers, callees, explain, context, impact, tests affected, implementers, interface, interfaces, pr, doctor, snapshot, and agent context
--max-files, --max-references, --max-symbols, --max-tests, --max-bytes, and --source-radius are only supported by context; --max-files, --max-symbols, --max-tests, and --max-bytes are also supported by agent context
unsupported agent context option: --max-references
unsupported agent context option: --source-radius
```

### First JSON Shape

`agent context --json` uses the shared envelope with
`command: "agent context"` and `target` set to the selected base ref.

Initial `data` fields:

- `target`, `base`, and `purpose`
- `readiness`: bounded package-load and repo-shape summary
- `snapshot`: requested/used status, freshness, relationship metadata, and
  refresh guidance
- `changedFiles`, `changedPackages`, `affectedPackages`, `affectedSymbols`,
  and bounded `changedSymbolDetails`
- `readingOrder`
- `targetRisk`
- `possibleRuntimeRelationships`: counts and bounded examples when a valid
  relationship snapshot is reused, otherwise explicit limitations
- `interfaceSummary`
- `testPlan` and `testCommands`
- `suggestedCommands`
- `sectionModes` for readiness, snapshot, context, impact, interfaces, tests,
  and PR-oriented follow-up
- `analysisMode`, `confidence`, `limitations`, `limits`, and `truncated`
- `sectionTruncation`: per-section `{section, field, omitted}` entries when
  item limits or the composed byte budget reduce the workflow payload

Embedded or directly reused report fields:

- `agentcontext.DiffReport` supplies changed-file, changed-package,
  changed-symbol, affected-package, target-risk, interface, test, reading-order,
  analysis-mode, confidence, limit, truncation, limitation, and warning fields.
- `sherpa.TestPlan` is reused as the grouped test-plan contract.
- `snapshot.RelationshipMetadata` is reused for bounded snapshot capability
  reporting.
- `sherpa.TargetRiskSummary` is reused unchanged.

Summarized fields:

- Doctor-style readiness is summarized to package-load status, repo-shape
  warnings, generated-file count, nested modules, and go.work detection.
- PR work is summarized as section mode plus suggested follow-up commands,
  rather than embedding the full `pr` report and repository structural risk.
- Possible runtime relationships are summarized by reason/scope/certainty and
  bounded examples only when a valid snapshot provides certainty labels.
- Interface data is summarized to affected interface/implementation names, not
  full interface profiles.

### Fixture Matrix

Real-world repository robustness fixtures needed next:

- `go.work` with multiple modules and root module participation
- nested modules excluded from root analysis unless inspected with `--root`
- build-tagged files that appear only with `--tags`
- generated Go files with standard `Code generated ... DO NOT EDIT` headers
- local `replace` dependencies that affect imported receiver boundary signals
- partial package-load failures that keep AST fallback and warnings visible
- large repository/diff fixtures for Slice 1.3 byte and section truncation

Test and entrypoint fixtures needed next:

- direct changed-symbol tests
- same-package related tests
- caller-package tests
- interface/implementation contract tests
- package fallback and whole-repository fallback
- stdlib `net/http` handler entrypoints
- CLI command-handler style entrypoints
- goroutine/function-literal possible runtime reachability

### Implementation Verification

Additional commands run after Slice 1.1/1.2 implementation:

```bash
go test ./internal/agentworkflow
go test ./cmd/gosherpa
```

The public command, validation, help, completion, schema guard, and golden JSON
coverage are implemented for the first diff-first `agent context` slice.

Additional commands run after Slice 1.3 implementation:

```bash
go test ./internal/agentworkflow
go test ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa help agent
go run ./cmd/gosherpa help agent context
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa agent context --base origin/main --use-snapshot --max-files 5 --max-symbols 5 --max-tests 5 --max-bytes 12000 --json
```

Slice 1.3 enables `agent context --max-bytes <n>` for the composed JSON `data`
payload, keeps the shared envelope outside the byte budget, records aggregate
truncation in `data.truncated`, and records per-section omissions in
`data.sectionTruncation`. Tight budgets keep a minimum valid report shell and
report `truncated.byteBudgetOverage` if that shell still cannot fit.
The refreshed verification snapshot was valid format v2 with 127 files,
11 packages, 2654 symbols, and relationship metadata marked present and
capable. The final agent-context verification used the snapshot with no
envelope warnings.

Slice 1.4 keeps snapshot creation explicit, surfaces `gosherpa snapshot --json`
as the refresh command for missing, stale, and invalid snapshots, reuses the
same snapshot freshness wording exposed by snapshot inspection, and adds
agent-workflow tests for missing, valid, stale, and invalid snapshot states.

Date: 2026-07-17

This audit records Phase 0 of `PRIORITY_IMPLEMENTATION_PLAN.md` for the next
priority tracks: relationship snapshot reuse, runtime-aware possible calls, and
target-aware impact risk. The previous audit described the completed semantic
accuracy, context export, and test-planning plan and is no longer the active
phase map.

## Selected Base Ref

Selected `<base-ref>`: `origin/main`

Reason: local refs include `main`, `remotes/origin/HEAD -> origin/main`, and
`remotes/origin/main`. `git rev-parse --verify origin/main` resolved to
`22dcd2974802f460d13013207e659c20e9bed593`, making it a concrete local diff
baseline.

## Baseline Verification

Commands run:

```bash
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --json
go run ./cmd/gosherpa context diff --base origin/main --json
go run ./cmd/gosherpa context diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --json
go run ./cmd/gosherpa impact diff --base origin/main --json
go run ./cmd/gosherpa impact diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa tests affected --base origin/main --json
go run ./cmd/gosherpa tests affected --base origin/main --use-snapshot --json
go run ./cmd/gosherpa pr --base origin/main --json
go run ./cmd/gosherpa pr --base origin/main --use-snapshot --json
```

Results:

| Command | Status | Runtime | Envelope warnings | Analysis mode | Confidence |
| --- | --- | ---: | --- | --- | --- |
| `go test ./...` | passed | about 51s wall | n/a | n/a | n/a |
| `doctor --json` | passed | 18.36s | none | `typechecked` | `medium` |
| `context symbol ./internal/sherpa.PlanTests --json` | passed | 8.79s | none | `typechecked+ast` | `medium` |
| `context diff --base origin/main --json` | passed | 0.39s | none | `git-diff+ast` | `medium` |
| `context diff --base origin/main --use-snapshot --json` | passed | 0.48s | stale snapshot fallback | `git-diff+ast` | `low` |
| `impact symbol ./internal/sherpa.PlanTests --json` | passed | 7.40s | none | `typechecked+ast` | `medium` |
| `impact diff --base origin/main --json` | passed | 0.38s | none | `git-diff+ast` | `medium` |
| `impact diff --base origin/main --use-snapshot --json` | passed | 0.45s | stale snapshot fallback | `git-diff+ast` | `low` |
| `tests affected --base origin/main --json` | passed | 0.35s | none | `git-diff+ast` | `medium` |
| `tests affected --base origin/main --use-snapshot --json` | passed | 0.43s | stale snapshot fallback | `git-diff+ast` | `low` |
| `pr --base origin/main --json` | passed | 0.67s | none | `git-diff+ast` | `medium` |
| `pr --base origin/main --use-snapshot --json` | passed | 0.72s | stale snapshot fallback | `git-diff+ast` | `low` |

The snapshot fallback warning text for snapshot-enabled diff-oriented commands:

```text
snapshot not used: Snapshot is stale. Run gosherpa snapshot to refresh it. (git state changed, repository files changed); using live repository analysis
```

`context diff --use-snapshot` uses the same stale reasons but says
`using live diff context analysis`.

## Snapshot Baseline

`.gosherpa/snapshot.json` exists and is stale.

`doctor --json` reported:

- status: `stale`
- path: `.gosherpa/snapshot.json`
- format version: `1`
- created at: `2026-07-05T21:52:17Z`
- file count: `106`
- package count: `14`
- symbol count: `1740`
- stale reasons: `git state changed`, `repository files changed`

Current snapshot format version `1` stores repository inventory and freshness
metadata: files, package summaries, symbols, build tags, git state, and a
fingerprint over repository inputs. It does not persist references, calls,
interface relationships, test references, or target risk summaries.

## CLI Snapshot Flag Matrix

Current support is defined by `cmd/gosherpa/command_registry.go`,
`cmd/gosherpa/cli_flags.go`, and validation in `cmd/gosherpa/main.go`.

Commands that accept `--use-snapshot` today:

- `analyze`
- `symbols`
- `symbol`
- `search`
- `packages`
- `context diff`
- `impact diff`
- `tests affected`
- `pr`

Important nuance: `packages --use-snapshot` is accepted by validation, but
snapshot data is reused only when `--tests` is also present. Without `--tests`,
the command emits a warning and runs live analysis because production-only
package counts are not represented by the snapshot.

Commands or subcommands that reject `--use-snapshot` today:

- `architecture`
- `risk`
- `doctor`
- `snapshot`
- `explain`
- `refs`
- `impact file`
- `impact package`
- `impact symbol`
- `tests <target>`
- `deps`
- `implementers`
- `interface`
- `interfaces`
- `path`
- `paths`
- `entrypoints`
- `callers`
- `callees`
- `context symbol`
- `context file`
- `context package`

Current validation error:

```text
--use-snapshot is only supported by analyze, symbols, symbol, search, packages, context diff, impact diff, tests affected, and pr
```

Any slice that enables relationship snapshot reuse for a currently rejected
command must update command specs, usage, completion, validation tests, docs,
schema notes, and golden fixtures together.

## Snapshot Format Decision

Relationship-capable persisted snapshots should use snapshot format version
`2`.

Reasons:

- Format version `1` is inventory-only and is already documented that way.
- Relationship reuse will change internal trust semantics even if public command
  JSON remains mostly additive.
- `normalizeSnapshot` currently maps missing or zero `formatVersion` to the
  current `FormatVersion`. That behavior is safe only for inventory-compatible
  v1 data; old or zero-version snapshots must not be silently treated as
  relationship-capable.

Compatibility rule for implementation:

- v1 snapshots may continue to decode for diagnostics, but after the v2 slice
  they are intentionally stale for reuse until refreshed.
- Relationship loaders must require explicit v2 relationship capability
  metadata before using persisted relationship data.
- Missing, zero, v1, or malformed relationship metadata must produce a visible
  fallback to live analysis for relationship queries.
- `normalizeSnapshot` behavior must preserve old file diagnostics without
  upgrading old files into relationship-capable snapshots.

## Slice 1.2 Implementation Update

Date: 2026-07-17

Slice 1.2 implemented snapshot format version `2` for relationship-capable
snapshot files. The persisted `.gosherpa/snapshot.json` shape now includes
internal relationship arrays and explicit `relationshipMetadata`; public
`gosherpa snapshot --json` output is intentionally separated into a bounded
summary with counts, not full relationship records.

Compatibility behavior after the slice:

- New snapshots write `formatVersion: 2`.
- Missing or zero `formatVersion` normalizes to legacy version `1`, not to a
  relationship-capable snapshot.
- Legacy v1 snapshots load as inventory files but are reported stale against
  the current v2 format with `snapshot format version changed`.
- A v2 snapshot without explicit relationship-capable metadata is stale with
  `relationship metadata missing`.
- Malformed relationship data makes the snapshot invalid and falls back to live
  analysis.
- Build tags remain part of compatibility checks; mismatched build tags make
  the snapshot stale.

Bounded relationship metadata now appears in `doctor --json` and
`snapshot --json`:

- `present`
- `capable`
- `snapshotFormatVersion`
- `totalCount`
- `countsByKind`

The first v2 snapshot created during verification reported relationship data as
present and capable, with `symbol-definition` count populated from the symbol
inventory and other relationship record counts currently `0`. Command-level
relationship reuse remains a Slice 1.3/1.4 follow-up; no additional
`--use-snapshot` CLI support was enabled in Slice 1.2.

Verification commands:

```bash
go test ./internal/snapshot
go test ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa doctor --json
```

## Slice 1.3 Implementation Update

Date: 2026-07-17

Slice 1.3 implemented opt-in snapshot reuse for selected standalone
relationship commands.

New snapshot behavior:

- `snapshot` format v2 now persists first-slice reference, direct call, and
  interface relationship records instead of only bounded relationship metadata.
- Relationship metadata counts now include persisted `reference`, `call`,
  `interface-implementation`, and `satisfied-interface` records when those
  relationships are present.
- Snapshot relationship records are still not dumped by public
  `snapshot --json`; public output remains bounded metadata.

New CLI support:

- `refs <target> --use-snapshot`
- `callers <target> --use-snapshot`
- `callees <target> --use-snapshot`
- `implementers <interface> --use-snapshot`
- `interface <interface> --use-snapshot`
- `interfaces <type> --use-snapshot`

Relationship snapshot outputs use `snapshot+typechecked`,
`snapshot+ast-fallback`, or `snapshot` as their analysis mode, depending on the
persisted relationship data. Missing, stale, invalid, or relationship-empty
snapshots fall back to live analysis with envelope or human-output warnings.
Ambiguous unqualified targets continue to return typed ambiguity diagnostics
instead of guessing from snapshot records.

Still intentionally unsupported in this slice:

- `path --use-snapshot`
- `paths --use-snapshot`
- plain `tests <target> --use-snapshot`
- non-diff `context` and `impact` subcommands

Verification commands:

```bash
go test ./internal/sherpa ./internal/impact ./internal/snapshot ./cmd/gosherpa
```

## Accepted Relationship Labels

The first relationship index should separate relationship kind, edge certainty,
and callee locality. Do not overload `sherpa.CallScope`.

Accepted relationship kinds:

| Kind | Scope |
| --- | --- |
| `symbol-definition` | symbol inventory and definition locations |
| `reference` | references to symbols, with existing `ReferenceKind` values |
| `call` | direct static call edges |
| `possible-call` | conservative runtime-aware possible call edges |
| `interface-implementation` | concrete type implements interface |
| `satisfied-interface` | type satisfies interface |
| `test-reference` | test directly references a target |
| `test-plan-seed` | bounded seed used to build direct, related, contract, caller-package, or fallback plans |
| `package-relationship` | package-level relationship for package impact or dependency-derived summaries |

Accepted edge certainty labels:

- `direct`: parser or typechecker evidence identifies a concrete relationship.
- `possible`: conservative possible runtime relationship; never presented as a
  proven direct edge.
- `inferred`: bounded derived relationship such as a test-plan seed or package
  fallback. Use only where the source fields make the derivation clear.

Call edges may also carry existing callee locality labels separately:

- `local`
- `external`
- `builtin`
- `dynamic`

The relationship index must preserve AST fallback records. Type information can
improve identity, but it must not be required for every record.

## Accepted Target Risk Model

Target-aware impact risk should be separate from repository structural risk.
The additive JSON field should be `targetRisk` unless a later slice deliberately
evolves an existing field with docs, schema tests, and golden fixtures.

Accepted risk levels:

- `low`
- `medium`
- `high`

Accepted risk scopes:

- `local`
- `package`
- `cross-package`
- `exported-api`
- `interface-contract`

Accepted reason categories:

- `affected-packages`
- `direct-references`
- `transitive-callers`
- `exported-symbol`
- `exported-type-method`
- `interface-contract`
- `package-fan-in`
- `missing-direct-tests`
- `fallback-tests`
- `possible-runtime-calls`
- `non-go-or-hunk-only-diff`
- `snapshot-fallback`
- `analysis-warning`

Scoring rules should be deterministic and evidence-backed. Confidence remains a
separate trust signal: risk describes expected blast radius, while confidence
describes how much to trust the analysis.

## Source Files Inspected

Phase 0 source inspection covered:

- `internal/snapshot/snapshot.go`
- `internal/symbolindex/index.go`
- `internal/semantics/repository.go`
- `internal/sherpa/semantic_context.go`
- `internal/sherpa/reference.go`
- `internal/sherpa/call.go`
- `internal/sherpa/impact.go`
- `internal/sherpa/test.go`
- `internal/impact/report.go`
- `internal/agentcontext/report.go`
- `cmd/gosherpa/snapshot_reuse.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/snapshot.go`
- `cmd/gosherpa/doctor.go`
- `docs/product/JSON_SCHEMA_V1.md`
- `docs/product/CONTEXT_SCHEMA_V1.md`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`

## Implementation Boundary

Phase 1 should start with an in-memory relationship index contract. Persistence
and CLI support should remain follow-up slices until the in-memory model is
deterministic, deduplicated, and covered by tests.

No command should claim relationship snapshot reuse until the corresponding
relationship kind is persisted, checked for compatibility, and surfaced through
analysis modes or limitations.

## Slice 1.4 Implementation Update

Date: 2026-07-17

Selected `<base-ref>` remains `origin/main`.

Slice 1.4 extended relationship snapshot reuse into the main must-use workflow:

- `context symbol <target> --use-snapshot`
- `context diff --base <ref> --use-snapshot`
- `impact symbol <target> --use-snapshot`
- `impact diff --base <ref> --use-snapshot`
- `tests affected --base <ref> --use-snapshot`
- `pr --base <ref> --use-snapshot`

CLI contract changes:

- `context symbol` and `impact symbol` now accept `--use-snapshot`.
- `context file`, `context package`, `impact file`, `impact package`, `path`,
  `paths`, and plain `tests <target>` still reject `--use-snapshot`.
- `context diff`, `impact diff`, `tests affected`, and `pr` keep existing
  snapshot support and now pass persisted relationship records to impact
  subanalysis in addition to current changed-symbol inventory.

Analysis-mode behavior:

- Snapshot-backed relationship subanalysis reports `snapshot+typechecked`,
  `snapshot+ast-fallback`, or `snapshot`.
- `context symbol.data.analysisMode` remains the broader symbol/source-context
  mode, while `referenceAnalysisMode`, `callAnalysisMode`,
  `interfaceAnalysisMode`, and `testAnalysisMode` report their own snapshot or
  live subanalysis modes.
- Diff-oriented bundles continue to report
  `snapshot+git-diff+typechecked+ast` or `snapshot+git-diff+ast` when a valid
  snapshot is used.

Implementation notes:

- Snapshot relationship records are passed through `impact.AnalyzerOptions`
  without making `internal/impact` depend on `internal/snapshot`.
- `context symbol` uses a CLI-level overlay for references, callers, callees,
  and selected interface signals; source context, purpose, and test planning
  remain live where needed.
- Relationship snapshots can seed direct affected tests from persisted
  reference, call, or test-reference records when available. Package-level and
  unsupported test planning still uses live analysis.
- Limitations and docs now distinguish snapshot-backed subanalysis from live
  report fields.

Verification commands:

```bash
go test ./internal/impact ./internal/agentcontext ./internal/snapshot ./cmd/gosherpa
go test ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
go run ./cmd/gosherpa context diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --use-snapshot --json
go run ./cmd/gosherpa impact diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa tests affected --base origin/main --use-snapshot --json
go run ./cmd/gosherpa pr --base origin/main --use-snapshot --json
```

Observed JSON modes:

- `context symbol` reported `referenceAnalysisMode: snapshot+typechecked` and
  `callAnalysisMode: snapshot+typechecked`; broader symbol/source context
  remained `typechecked+ast`.
- `impact symbol` reported `analysisMode: snapshot+typechecked`.
- `context diff`, `impact diff`, `tests affected`, and `pr` reported
  `snapshot+git-diff+typechecked+ast` for the bundle-level mode with
  `origin/main`.

## Slice 2.1 Implementation Update

Date: 2026-07-17

Selected `<base-ref>` remains `origin/main`, though no diff-oriented
verification was required for this slice.

Slice 2.1 added a public possible-call model without changing direct call
semantics:

- `callers --json` and `callees --json` now include an additive
  `possibleCalls` array.
- Possible-call entries include `caller`, optional `callee`, `certainty:
  possible`, `reason`, `scope`, `position`, and `range` when available.
- The first stable reason categories are `interface-dispatch`, `goroutine`,
  `function-literal`, `function-value`, `stdlib-http-handler`, and
  `imported-receiver`.
- Current emitted possible calls come from existing bounded uncertainty signals
  for interface dispatch, goroutine starts, function literals, and function
  values. Reflection remains a limitation only.
- Direct `callers`, `callees`, `path`, and `paths` output remain separate from
  possible edges; `scope` still describes callee locality, not certainty.

Verification commands:

```bash
go test ./internal/sherpa ./cmd/gosherpa
go run ./cmd/gosherpa --root cmd/gosherpa/testdata/possible_calls_project callees Entry --json
go test ./...
```

## Slice 2.2 Implementation Update

Date: 2026-07-17

Selected `<base-ref>` remains `origin/main`, though no diff-oriented
verification was required for this slice.

Slice 2.2 connected visible interface dispatch to bounded local implementer
methods:

- Interface-typed selector calls now use typechecked receiver and method data
  to find repository-local implementer methods with matching method signatures.
- Emitted possible calls keep `certainty: possible`, reason
  `interface-dispatch`, call-site position/range, and `scope: local`.
- Unknown, broad, or unsupported interface dispatch remains a limitation
  instead of producing an abstract guessed edge.
- Direct caller/callee arrays remain unchanged; concrete receiver calls are
  still direct evidence, while interface-dispatch implementers are only
  possible evidence.
- Fixture coverage now includes same-package implementers, cross-package
  implementers, embedded interfaces, duplicate method names with incompatible
  signatures, and no-known-implementer cases.
- Snapshot format v2 now persists `possible-call` relationship records, and
  snapshot-backed `callers`/`callees` include additive `possibleCalls` from
  valid relationship snapshots without dumping relationship details from
  public `snapshot --json`.

Implementation note: the call layer builds its own typechecked local method
catalog for dispatch matching to avoid creating an import cycle with the
interface impact package. The matching semantics remain aligned with the
interface graph: local method sets, signature compatibility, embedded
interfaces, and package-local implementer boundaries.

Verification commands:

```bash
go test ./...
go test ./internal/sherpa ./internal/snapshot ./cmd/gosherpa
go test ./internal/sherpa ./internal/impact ./internal/agentcontext ./cmd/gosherpa
go run ./cmd/gosherpa --root cmd/gosherpa/testdata/possible_calls_project callees Entry --json
go run ./cmd/gosherpa --root cmd/gosherpa/testdata/possible_calls_project snapshot --json
```

## Slice 2.3 Implementation Update

Date: 2026-07-17

Selected `<base-ref>` remains `origin/main`
(`080ab07749c3eb742bbf387da3d0ac807aed49a1` at verification time), though no
diff-oriented verification was required for this slice.

Slice 2.3 made visible goroutine and function-literal runtime flows more
concrete without changing direct caller/callee counts:

- `go Target()` still appears as a direct callee where statically known and now
  emits a `possibleCalls` entry with reason `goroutine` and `scope: local` for
  local targets.
- Goroutine function literals such as `go func() { Target() }()` emit bounded
  possible local target calls instead of only abstract function-literal
  uncertainty.
- Immediately invoked function literals and function literals passed to simple
  local call sites emit possible target calls when their bodies contain visible
  call sites with source ranges.
- Single-assignment local function values continue to resolve as direct calls
  when type information proves one static target.
- Reassigned function values, struct-field function values, and escaping
  function literals remain conservative: GoSherpa reports limitations or
  dynamic possible calls without guessing hidden concrete targets.
- Entrypoint reachability now uses bounded local `goroutine` and
  `function-literal` possible edges so a target reached from a visible
  goroutine function literal can report the originating entrypoint.

Verification commands:

```bash
go test ./internal/sherpa
go test ./cmd/gosherpa
go test ./internal/snapshot
go test ./internal/impact ./internal/agentcontext
go test ./internal/sherpa ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa --root cmd/gosherpa/testdata/possible_calls_project callees Entry --json
go run ./cmd/gosherpa --root cmd/gosherpa/testdata/json_project entrypoints Target --json
```

## Slice 2.5 Implementation Update

Date: 2026-07-17

Selected `<base-ref>` remains `origin/main`, though no diff-oriented
verification was required for this slice.

Slice 2.5 added imported receiver boundary signals without expanding analysis
into dependency internals:

- Typechecked selector calls to imported receiver methods now emit additive
  `possibleCalls` with reason `imported-receiver`, `certainty: possible`,
  `scope: external`, call-site position/range, and an import-path-qualified
  callee such as `strings.Builder.WriteString` or `example.com/dep.Client.Do`.
- Direct caller/callee arrays and existing `CallScopeExternal` semantics are
  preserved; local receiver calls remain direct/local and do not produce
  imported-boundary possible calls.
- Standard-library receivers, local replace-module dependencies, alias imports,
  dot-import package-function non-cases, and local receiver calls are covered by
  focused fixtures.
- `callers` target matching avoids treating external possible-call method names
  as unqualified local target matches.
- Context, impact, and PR workflows continue to keep external dependencies out
  of local affected-package evidence and describe imported receivers only as
  external boundary/limitation context.

Verification commands:

```bash
go test ./internal/sherpa
go test ./cmd/gosherpa
go test ./internal/sherpa ./internal/agentcontext ./internal/impact ./cmd/gosherpa
```

## Phase 3 Implementation Update

Date: 2026-07-17

Selected `<base-ref>` remains `origin/main`.

Phase 3 promoted the accepted target-aware risk model into the public workflow
without replacing raw impact evidence:

- Impact, explain, context, and PR reports now include additive `targetRisk`
  summaries with `level`, `scope`, `reasons`, `signals`, and `limitations`.
- The implementation keeps target risk separate from confidence and from the
  existing repository structural risk surfaced by `gosherpa risk` and `pr`.
- Deterministic scoring uses current evidence such as affected packages, direct
  references, transitive callers, exported symbols, exported type methods,
  interface-contract involvement, package fan-in, missing/fallback test
  evidence, possible runtime calls, analysis warnings, hunk-only or non-Go
  diffs, and snapshot fallback.
- `impact file|package|symbol|diff` JSON and human output expose concise
  target-risk summaries while preserving the detailed affected package,
  reference, caller, callee, test, confidence, and limitation sections.
- `impact diff` and downstream `context diff`/`pr` outputs include aggregate
  diff target risk and per-changed-symbol target risk where bounded symbol
  analysis is available.
- `context symbol|file|package|diff`, `explain`, and `pr` carry target risk so
  agents see impact breadth before editing, while docs instruct consumers to
  treat it as deterministic evidence rather than proof or defect prediction.
- JSON schema docs, context schema docs, CLI docs, agent guidance, and golden
  fixtures were updated for the additive `targetRisk` contract.

Verification commands:

```bash
go test ./internal/sherpa ./internal/impact ./internal/explain ./internal/agentcontext ./cmd/gosherpa
go test ./cmd/gosherpa
```

## Final Integration Update

Date: 2026-07-17

Selected `<base-ref>` remained `origin/main` for the final workflow pass.
`context symbol --use-snapshot` and `impact symbol --use-snapshot` are both
supported, so the non-snapshot symbol fallback pair was not needed.

Final verification covered a fresh relationship-capable v2 snapshot, doctor,
the must-use diff-oriented command set, the snapshot-backed symbol pair, and
the full Go test suite:

```bash
go test ./internal/sherpa ./internal/impact ./internal/explain ./internal/agentcontext ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa context diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa impact diff --base origin/main --use-snapshot --json
go run ./cmd/gosherpa tests affected --base origin/main --use-snapshot --json
go run ./cmd/gosherpa pr --base origin/main --use-snapshot --json
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --use-snapshot --json
```

Observed results:

- `doctor --json` reported the snapshot as valid with relationship metadata
  present and capable.
- `context diff`, `impact diff`, `tests affected`, and `pr` reported
  `snapshot+git-diff+typechecked+ast` with no warnings.
- `context diff`, `impact diff`, and `pr` reported
  `targetRisk: medium/cross-package`; `tests affected` remains focused on test
  planning and does not expose target risk.
- `context symbol ./internal/sherpa.PlanTests --use-snapshot --json` reported
  snapshot-backed reference/call subanalysis, broader `typechecked+ast`
  context, and `targetRisk: high/exported-api`.
- `impact symbol ./internal/sherpa.PlanTests --use-snapshot --json` reported
  `snapshot+typechecked` and `targetRisk: high/exported-api`.
- No final smoke command emitted envelope warnings.

## Current Plan Slice 2.4 Update

Date: 2026-07-18

Selected `<base-ref>`: `HEAD`, resolved to
`92798c4c4ab0b1c55648e7f7d066b7db123e3365`.

Slice 2.4 added large-repository performance guardrails without wall-clock
assertions:

- Added shared `internal/repostats` cost summaries for `doctor` and
  `agent context`.
- `doctor.data.cost` and `agent context.data.cost` now report package, Go file,
  test file, generated-file, skipped-module, local-replacement,
  package-load-warning, changed-target, affected-target, symbol, and
  relationship counts.
- Symbol and relationship inventory counts come from readable snapshot
  metadata; missing, stale, and invalid snapshot states are reflected in
  limitations and refresh guidance instead of forcing extra live inventory
  work.
- `agentcontext.DiffReport` carries internal `json:"-"` snapshot reuse
  metadata so `agent context` can summarize possible runtime relationships from
  the already loaded diff snapshot data instead of loading the snapshot again
  for that section.
- Added deterministic tests for `repostats` normalization and agent-context
  snapshot reuse metadata; existing semantics cache tests remain the
  non-timing guardrail for repeated package-load paths.
- Updated CLI docs, schema docs, status, agent notes, and `llms.txt` to explain
  `data.cost` and when to refresh snapshots.

Verification commands:

```bash
git rev-parse --verify HEAD
go test ./internal/semantics ./internal/snapshot ./internal/repostats ./internal/agentworkflow ./internal/agentcontext ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa agent context --base HEAD --use-snapshot --max-bytes 12000 --json
```

Observed results:

- `doctor --json` reported a stale local snapshot, retained stored symbol and
  relationship inventory counts in `data.cost`, and suggested
  `gosherpa snapshot`.
- `agent context --base HEAD --use-snapshot --max-bytes 12000 --json`
  intentionally verified stale-snapshot fallback and refresh guidance without
  refreshing the local snapshot during this slice.
