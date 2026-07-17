# GoSherpa Priority Implementation Plan

This plan replaces the completed priority plan. It describes the next three
implementation tracks that should most improve GoSherpa's daily usefulness for
developers and coding agents.

The goal is still not to add many new commands. The goal is to make the current
GoSherpa workflow faster, more semantically complete, and easier to trust:

```bash
gosherpa doctor --json
gosherpa context diff --base <base-ref> --use-snapshot --json
gosherpa context symbol ./internal/package.Target --json
gosherpa impact symbol ./internal/package.Target --json
gosherpa tests affected --base <base-ref> --json
gosherpa pr --base <base-ref> --json
```

Primary source documents:

- `docs/product/MUST_USE_READINESS.md`
- `docs/product/AGENT_PRIORITY_LIST.md`
- `docs/product/FEATURE_ROADMAP.md`
- `docs/product/AGENT_RECOMMENDATION_CRITERIA.md`
- `docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md`
- `docs/STATUS.md`

`docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md` is currently a historical
baseline for the completed semantic-accuracy/context/test-planning plan. Treat
its command matrix and fixture notes as useful current-state evidence, but do
not treat its old phase mapping as active. Phase 0 below must append or replace
the audit with a new baseline for this plan before implementation starts.

Primary code contracts to preserve:

- JSON responses use the shared envelope in `cmd/gosherpa/cli_json.go`.
- Stable JSON examples live in `cmd/gosherpa/testdata/golden-json/`.
- Schema guardrails live in `cmd/gosherpa/json_schema_test.go` and
  `cmd/gosherpa/json_golden_test.go`.
- Command and flag support is defined by `cmd/gosherpa/command_registry.go`,
  `cmd/gosherpa/cli_flags.go`, and validation in `cmd/gosherpa/main.go`.
- Human output must remain readable and stable; successful output goes to
  stdout, diagnostics go to stderr.
- Current confidence values are `medium` and `low`. Do not add new confidence
  values unless schema docs, tests, and golden fixtures change in the same
  slice.
- Current repository risk levels are `low`, `medium`, and `high`; risk levels
  are not confidence values.
- Current `--use-snapshot` support is limited to `analyze`, `symbols`,
  `symbol`, `search`, `packages --tests`, `context diff`, `impact diff`,
  `tests affected`, and `pr`. Relationship commands and non-diff
  `context`/`impact` subcommands must continue to reject `--use-snapshot`
  until the slice that intentionally changes their CLI contract lands.
- Plain `tests <target>` and `tests affected --base <ref>` are separate CLI
  contracts. Do not move `--base`, `--tags`, `--scope`, or `--use-snapshot`
  between them without updating usage, validation, docs, and tests in the same
  slice.
- Existing `sherpa.CallScope` values (`local`, `external`, `builtin`,
  `dynamic`) describe callee locality. New relationship certainty values such
  as `direct` and `possible` must be represented as separate edge metadata, not
  by overloading the existing call scope field.
- Use `<base-ref>` in docs and verification commands. Pick a concrete existing
  ref such as `origin/main`, `main`, or a documented test ref at the start of a
  slice and record it in the verification note.

## Current State

The previous priority plan completed the current P0 product shape:

- typechecked loading and shared semantic context for the main agent-facing
  flows
- production-ready context export for symbol, file, package, and diff targets
- structured test planning with direct, related, contracts, caller-package, and
  fallback groups
- PR summaries, JSON schema coverage, golden fixtures, warnings, limitations,
  source ranges, and ambiguity diagnostics
- first-slice snapshot reuse for repository inventory and current
  changed-symbol inventory

The remaining must-use gap is deeper semantic completeness and repeated-query
speed. Ordinary Go repositories still need better behavior around:

1. persisted relationship reuse beyond inventory snapshots
2. possible runtime call relationships without overclaiming certainty
3. target-aware impact and risk summaries that turn many facts into a concise
   change judgment

These are the next three implementation tracks.

## Priority Order

1. **Relationship Index And Snapshot Reuse**
   Persist or rebuild reusable relationship data so repeated context, impact,
   reference, call, interface, and test queries do not redo the same expensive
   analysis each time.

2. **Runtime-Aware Possible Call Semantics**
   Improve caller, callee, path, entrypoint, context, impact, and PR workflows
   with conservative possible-call signals for interface dispatch, goroutines,
   selected standard-library runtime wiring, and imported receiver calls.

3. **Target-Aware Impact And Risk Summary**
   Add concise local/package/cross-package/exported/high-fan-in judgments to
   impact, context, and PR workflows while preserving the underlying evidence.

Do not prioritize MCP, TUI, editor integration, graph export, GitHub Actions, or
new broad standalone commands until these tracks are complete.

## Execution Rules For Agents

Before starting any slice:

- Read the source files listed for that slice.
- Re-read command support in `cmd/gosherpa/command_registry.go`,
  `cmd/gosherpa/cli_flags.go`, and validation in `cmd/gosherpa/main.go` before
  changing CLI behavior.
- Inspect existing focused tests for the area.
- Select and record a concrete `<base-ref>` for diff-oriented checks.
- Preserve AST fallback behavior when typechecked loading is unavailable.
- Surface uncertainty as warnings, `analysisMode`, subanalysis modes,
  `confidence`, `limitations`, and explicit relationship scopes or certainty
  labels. Do not silently drop failed or partial analysis.
- Keep JSON schemas, golden fixtures, human output, and docs aligned.
- Keep changes PR-sized. If a slice reveals a deeper issue, record it as a
  follow-up instead of expanding the slice without bound.

Every completed slice must include:

- focused unit tests for changed behavior
- CLI or golden JSON coverage when command output changes
- documentation updates when JSON shape, user-visible behavior, limitations,
  readiness status, or flag support changes
- a verification note listing exact commands run and the selected `<base-ref>`

Recommended baseline verification for the current repository state:

```bash
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --json
go run ./cmd/gosherpa context diff --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --json
go run ./cmd/gosherpa impact diff --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa tests affected --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa pr --base <base-ref> --use-snapshot --json
```

## Phase 0: Baseline Audit And Design Lock

Goal: turn the next three tracks into concrete implementation boundaries before
changing behavior.

Entry criteria:

- Working tree is understood.
- Existing tests pass, or current failures are documented before edits.
- A concrete `<base-ref>` is selected and recorded.

Tasks:

- [x] Run `go test ./...`.
- [x] Run `go run ./cmd/gosherpa doctor --json`.
- [x] Run the current must-use command set with and without `--use-snapshot`
      where it is currently supported:
      - `context symbol ./internal/sherpa.PlanTests --json`
      - `context diff --base <base-ref> --json`
      - `context diff --base <base-ref> --use-snapshot --json`
      - `impact symbol ./internal/sherpa.PlanTests --json`
      - `impact diff --base <base-ref> --json`
      - `impact diff --base <base-ref> --use-snapshot --json`
      - `tests affected --base <base-ref> --json`
      - `tests affected --base <base-ref> --use-snapshot --json`
      - `pr --base <base-ref> --json`
      - `pr --base <base-ref> --use-snapshot --json`
- [x] Record the current CLI snapshot flag matrix from
      `command_registry.go`, `cli_flags.go`, and `main.go`. This must include
      which subcommands reject `--use-snapshot` today.
- [x] Record current runtime and warnings for the above commands in a short
      audit update under `docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md`.
- [x] Confirm whether `.gosherpa/snapshot.json` is missing, valid, stale, or
      invalid before changing snapshot behavior.
- [x] Decide whether snapshot format changes remain format version `1` with
      compatible additive fields or require a version bump to `2`.
- [x] If a version bump is needed, decide how `normalizeSnapshot` treats missing
      or zero `formatVersion`; old snapshots must not be silently upgraded into
      a relationship-capable format.
- [x] Define the first accepted relationship scopes or certainty labels:
      - symbol definitions and inventory
      - references
      - direct call edges
      - possible call edges
      - interface implementers and satisfied interfaces
      - test references and test plan seeds
- [x] Define the first accepted target risk levels and reason categories for
      impact summaries.

Primary files to inspect:

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
- `docs/product/JSON_SCHEMA_V1.md`
- `docs/product/CONTEXT_SCHEMA_V1.md`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`

Exit criteria:

- Audit records current behavior and selected `<base-ref>`.
- Snapshot format decision is documented before implementation.
- The first relationship and risk scopes are explicit.
- No implementation slice starts with unknown schema or CLI contract impact.

Verification:

```bash
go test ./...
go run ./cmd/gosherpa doctor --json
```

Phase 0 verification completed on 2026-07-17 with selected `<base-ref>`
`origin/main`. Full runtime, warning, snapshot, relationship-label, and
target-risk decisions are recorded in
`docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md`.

## Phase 1: Relationship Index And Snapshot Reuse

Goal: make repeated GoSherpa queries faster and more consistent by reusing
relationship data, not only inventory.

Why this comes first:

Agents ask several related questions in sequence. The first snapshot slices can
reuse inventory, selected standalone relationship records, and current
changed-symbol inventory, but must-use context, impact, affected-test, and PR
relationship subanalysis still need broader relationship reuse. Persisted or
reusable relationship data is the largest speed and consistency lever before
adding broader integration surfaces.

### Slice 1.1: Relationship Index Contract

Goal: define the in-memory relationship index before persistence.

Tasks:

- [x] Extend or wrap `symbolindex.RepositoryIndex` with relationship-oriented
      structures without forcing every command to adopt them immediately.
- [x] Model stable source locations with existing `Position` and
      `SourceRange` shapes where possible.
- [x] Model these records with deterministic ordering:
      - `ReferenceRecord`
      - `CallEdgeRecord`
      - `PossibleCallEdgeRecord` if it can be represented without enabling
        Phase 2 behavior yet
      - `InterfaceImplementationRecord`
      - `TestReferenceRecord`
      - `PackageRelationshipRecord` if needed for package impact
- [x] Include relationship metadata:
      - package path
      - source file
      - source symbol identity where known
      - target symbol identity or package-qualified target
      - analysis mode
      - edge certainty or relationship kind: `direct`, `possible`,
        `interface`, `test`, or another documented stable value
      - callee locality separately from edge certainty when call data uses
        `local`, `external`, `builtin`, or `dynamic`
      - limitations or warning references where needed
- [x] Keep AST fallback data representable. Do not require type information for
      every relationship record.
- [x] Add focused tests for deterministic ordering, non-nil slices, duplicate
      removal, and relative paths.

Primary files:

- `internal/symbolindex/index.go`
- `internal/symbolindex/index_test.go`
- `internal/semantics/repository.go`
- `internal/sherpa/reference.go`
- `internal/sherpa/call.go`
- `internal/sherpa/test.go`
- `internal/sherpa/test_plan.go`
- `internal/impact/interface.go`

Exit criteria:

- A relationship index can represent current references, direct calls,
  interface relationships, and test references.
- The model has no command-specific JSON leakage.
- The index can be built from a typechecked repository when available.
- Tests cover sorting, deduplication, and empty-array behavior.

Verification:

```bash
go test ./internal/symbolindex ./internal/semantics ./internal/sherpa ./internal/impact
```

Slice 1.1 verification completed on 2026-07-17. Added the in-memory
`symbolindex.RelationshipIndex` contract with deterministic normalization,
relationship certainty separate from call scope, root-relative source
locations, non-nil slices, and focused `internal/symbolindex` tests. No
snapshot persistence or CLI behavior was enabled in this slice.

### Slice 1.2: Snapshot Format For Relationships

Goal: persist the first relationship index slice safely.

Tasks:

- [x] Add compatible snapshot fields or bump `FormatVersion` intentionally.
- [x] Separate the persisted snapshot file shape from public `snapshot --json`
      output if needed. Adding relationship arrays directly to
      `snapshotstore.Snapshot` will otherwise expose unbounded relationship
      detail through the command JSON.
- [x] Persist relationship records only when they can be invalidated by the
      existing snapshot fingerprint inputs, or extend the fingerprint inputs in
      the same slice.
- [x] Ensure build tags are part of snapshot compatibility, as they are today.
- [x] Keep snapshot writes deterministic.
- [x] Ensure stale, invalid, and missing snapshot diagnostics remain clear in
      `doctor`.
- [x] Add bounded relationship metadata to inspect/status output, for example
      presence flags, format capability, and counts by relationship kind. Do
      not dump full relationship records from `doctor`.
- [x] Add tests for:
      - valid relationship snapshot load
      - stale relationship snapshot fallback
      - build-tag mismatch fallback
      - malformed relationship data fallback
      - backwards compatibility with old inventory-only snapshots if format
        version is unchanged
      - intentional stale or invalid diagnostics for old snapshots if format
        version is bumped
- [x] Update schema docs for `snapshot` output.

Primary files:

- `internal/snapshot/snapshot.go`
- `cmd/gosherpa/snapshot.go`
- `cmd/gosherpa/doctor.go`
- `cmd/gosherpa/snapshot_reuse.go`
- `cmd/gosherpa/main_test.go`
- `cmd/gosherpa/json_schema_test.go`
- `docs/product/JSON_SCHEMA_V1.md`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`

Exit criteria:

- `gosherpa snapshot --json` exposes relationship snapshot metadata without
  dumping unbounded relationship details.
- `doctor --json` reports whether relationship data is present and reusable.
- Persisted relationship details can be loaded internally without becoming part
  of the public command envelope by accident.
- Old snapshots either load safely or produce an intentional invalid/stale
  diagnostic.
- Snapshot limitations clearly distinguish inventory reuse from relationship
  reuse.

Verification:

```bash
go test ./internal/snapshot ./cmd/gosherpa
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa doctor --json
```

Slice 1.2 verification completed on 2026-07-17. Selected `<base-ref>` remains
`origin/main`, though no diff-oriented verification was required for this
slice. Implemented snapshot format v2 with persisted internal relationship
arrays, explicit relationship-capability metadata, bounded public
`snapshot --json` summaries, `doctor` relationship metadata, stale diagnostics
for legacy v1 snapshots, build-tag mismatch checks, malformed relationship data
fallback, schema/docs updates, and focused snapshot/CLI tests.

Commands run:

```bash
go test ./internal/snapshot
go test ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa doctor --json
```

### Slice 1.3: Reuse For References, Calls, Interfaces, And Tests

Goal: make high-value query paths consume valid relationship snapshots.

Tasks:

- [x] Add internal loaders that can choose between:
      - valid relationship snapshot data
      - shared in-memory semantic context
      - live AST/typechecked fallback
- [x] Start with opt-in `--use-snapshot`; do not make automatic snapshot reuse
      the default in this slice.
- [x] Reuse relationship snapshot data for these commands when safe:
      - `refs`
      - `callers`
      - `callees`
      - `implementers`
      - `interfaces`
      - `interface` with persisted methods, references, and implementers; method
        usage records remain live-only follow-up data
- [x] Leave `path`, `paths`, and plain `tests <target>` unsupported until
      snapshot compatibility inputs and test-reference semantics are covered by
      a dedicated slice.
- [x] When enabling `--use-snapshot` for a command that does not support it
      today, update usage, validation, completion, CLI tests, and docs in the
      same slice. Unsupported combinations must continue to fail intentionally
      until the slice lands.
- [x] Do not enable `--use-snapshot` for a command whose current flags cannot
      express the snapshot compatibility inputs it needs. Either add and test
      those flags in the same slice, or leave the command unsupported.
- [x] Preserve package-qualified ambiguity diagnostics. Snapshot data must not
      guess on ambiguous unqualified targets.
- [x] Ensure analysis modes clearly show snapshot involvement, for example
      `snapshot+typechecked` or another documented stable value. Add shared
      constants and schema docs instead of ad hoc strings.
- [x] Ensure human warnings do not become noisy when snapshot fallback occurs.
- [x] Add CLI tests proving stale snapshots fall back to live analysis with
      warnings.
- [x] Update JSON schemas and golden fixtures only for intentionally changed
      output fields.

Primary files:

- `cmd/gosherpa/snapshot_reuse.go`
- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/completion.go`
- `cmd/gosherpa/command_handlers.go`
- `cmd/gosherpa/cli_json.go`
- `internal/sherpa/reference.go`
- `internal/sherpa/call.go`
- `internal/sherpa/test.go`
- `internal/impact/interface.go`
- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`

Exit criteria:

- Snapshot-backed relationship commands produce the same semantic answers as
  live analysis for covered fixture cases.
- Fallback behavior remains deterministic and visible.
- Analysis modes and limitations do not overclaim the scope of snapshot reuse.

Verification:

```bash
go test ./internal/sherpa ./internal/impact ./cmd/gosherpa
go run ./cmd/gosherpa snapshot
go run ./cmd/gosherpa refs ./internal/sherpa.ParseFile --use-snapshot --json
go run ./cmd/gosherpa callers ./internal/sherpa.ParseFile --use-snapshot --json
```

Slice 1.3 verification completed on 2026-07-17. Selected `<base-ref>` remains
`origin/main`, though no diff-oriented verification was required for this
slice. Snapshot creation now persists first-slice reference, direct-call, and
interface relationship records; `refs`, `callers`, `callees`, `implementers`,
`interface`, and `interfaces` accept opt-in `--use-snapshot` and fall back to
live analysis with warnings when snapshots are missing, stale, invalid, or do
not contain the needed relationship data. `path`, `paths`, and plain
`tests <target>` remain intentionally unsupported for `--use-snapshot`.

Commands run:

```bash
go test ./internal/sherpa ./internal/impact ./internal/snapshot ./cmd/gosherpa
```

### Slice 1.4: Reuse In Context, Impact, Affected Tests, And PR

Goal: make the main must-use workflow benefit from relationship reuse.

Tasks:

- [x] Extend `context symbol`, `context diff`, `impact symbol`, `impact diff`,
      `tests affected`, and `pr` to use relationship snapshots where safe.
- [x] Decide the exact `--use-snapshot` subcommand matrix before editing. If
      `context file`, `context package`, `impact file`, or `impact package`
      remain unsupported, keep explicit validation tests and docs that say so.
- [x] When extending `--use-snapshot` beyond currently supported diff-oriented
      commands, update `supportsSnapshotOption`, usage lines, completion, CLI
      validation tests, docs, and examples in the same slice.
- [x] Keep current changed-symbol inventory reuse intact.
- [x] Ensure `referenceAnalysisMode`, `callAnalysisMode`,
      `interfaceAnalysisMode`, and `testAnalysisMode` reflect only their own
      subanalysis.
- [x] Add limitations that explain which parts used snapshot data and which
      still ran live.
- [x] Keep byte budgets and truncation behavior valid after snapshot-backed
      data is introduced.
- [x] Add performance-oriented tests or benchmark-style regression checks where
      practical, but do not make wall-clock timing brittle in unit tests.
- [x] Update `AGENT_NOTES.md`, `llms.txt`, `docs/CLI_REFERENCE.md`,
      `docs/STATUS.md`, and schema docs.

Primary files:

- `internal/agentcontext/report.go`
- `internal/agentcontext/diff_report.go`
- `internal/impact/report.go`
- `internal/sherpa/impact.go`
- `internal/sherpa/test.go`
- `cmd/gosherpa/pr.go`
- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/completion.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/testdata/golden-json/`

Exit criteria:

- The must-use command set can benefit from valid relationship snapshots.
- Snapshot fallback warnings stay on the envelope or stderr as appropriate.
- Human output remains concise.
- Docs no longer say relationship queries always analyze live when the new
  behavior is implemented.

Verification:

```bash
go test ./internal/agentcontext ./internal/impact ./internal/sherpa ./cmd/gosherpa
go test ./...
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
go run ./cmd/gosherpa context diff --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --use-snapshot --json
go run ./cmd/gosherpa impact diff --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa tests affected --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa pr --base <base-ref> --use-snapshot --json
```

Slice 1.4 implementation completed on 2026-07-17. Selected `<base-ref>`
remains `origin/main`. `context symbol` and `impact symbol` now accept
opt-in `--use-snapshot`; `context file`, `context package`, `impact file`, and
`impact package` remain intentionally unsupported. Valid relationship snapshots
can now supply selected reference, call, interface, direct-test, and changed
symbol impact subanalysis for `context symbol`, `context diff`, `impact symbol`,
`impact diff`, `tests affected`, and `pr`, while unsupported report portions
continue to run live and remain documented in limitations.

Final verification on 2026-07-17 used `origin/main` for `<base-ref>`:

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

The `context symbol` smoke reported snapshot-backed reference and call
subanalysis, and `impact symbol` reported `snapshot+typechecked`.

Phase 1 is done when:

- Snapshot reuse covers selected relationship data, not only inventory.
- Valid snapshots improve repeated-query consistency for the must-use workflow.
- Stale or invalid snapshots fall back clearly.
- JSON docs, golden fixtures, `doctor`, and status docs agree.

## Phase 2: Runtime-Aware Possible Call Semantics

Goal: improve ordinary Go call and entrypoint analysis without pretending to
perform full whole-program runtime inference.

Why this comes second:

After relationship reuse is explicit, the biggest correctness gap is call graph
completeness. GoSherpa already surfaces uncertainty for dynamic dispatch,
reflection, function values, goroutines, and function literals. The next step is
to turn some visible uncertainty into conservative possible-call relationships.

Rules:

- Use `direct` for statically proven call edges.
- Use `possible` for conservative edges that are plausible but not proven as the
  exact runtime target.
- Never merge possible edges into direct caller/callee counts without marking
  them.
- Keep limitations visible when possible calls are present.
- Do not implement full SSA, whole-program pointer analysis, reflection
  analysis, or broad framework inference in this phase. The stdlib HTTP first
  slice below is the only runtime-wiring inference explicitly in scope.

### Slice 2.1: Possible Call Data Model

Goal: create a stable representation for possible call edges.

Tasks:

- [x] Extend call result models with possible call edges in a backwards
      compatible way, or add a clearly documented adjacent field.
- [x] Keep possible edges separate from existing `callees[].scope` values.
      `scope` should continue to mean locality; possible/direct should mean
      certainty or relationship kind.
- [x] Include reason categories such as:
      - `interface-dispatch`
      - `goroutine`
      - `function-literal`
      - `function-value`
      - `stdlib-http-handler`
      - `imported-receiver`
- [x] Include source positions and ranges for the call site when available.
- [x] Keep direct `callers`, `callees`, `path`, and `paths` output stable unless
      a command explicitly includes possible edges.
- [x] Decide whether possible edges are always emitted in JSON or gated by a
      flag. Prefer additive JSON fields over new flags if output remains
      bounded.
- [x] Add schema tests and golden JSON fixtures for one representative command.

Primary files:

- `internal/sherpa/call.go`
- `internal/sherpa/call_output.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`
- `docs/product/JSON_SCHEMA_V1.md`

Exit criteria:

- Possible calls have a stable JSON shape and source ranges where available.
- Existing direct call behavior remains unchanged for current fixtures.
- Limitations explain the difference between direct and possible edges.

Verification:

```bash
go test ./internal/sherpa ./cmd/gosherpa
```

Slice 2.1 verification completed on 2026-07-17. Selected `<base-ref>` remains
`origin/main`, though no diff-oriented verification was required for this
slice. Possible calls are emitted as additive `possibleCalls` arrays for
`callers --json` and `callees --json`; direct caller/callee arrays and call path
output remain unchanged. Possible edges carry `certainty: possible`, a stable
reason category, callee locality in `scope`, and source position/range when
available. Initial emitted reasons are derived from existing bounded
uncertainty signals for interface dispatch, goroutine starts, function
literals, and function values; reflection remains a limitation only.

Commands run:

```bash
go test ./internal/sherpa ./cmd/gosherpa
go run ./cmd/gosherpa --root cmd/gosherpa/testdata/possible_calls_project callees Entry --json
go test ./...
```

### Slice 2.2: Interface Dispatch Possible Edges

Goal: connect visible interface-typed calls to known local implementers as
possible callees.

Tasks:

- [ ] Detect selector calls on interface-typed values when typechecked data is
      available.
- [ ] Use the existing interface graph to find local implementers with matching
      method signatures.
- [ ] Emit possible call edges from the caller to matching implementer methods.
- [ ] Preserve current direct call behavior for concrete receiver calls.
- [ ] Avoid possible edges when the implementer set is too broad or unknown;
      add a limitation instead.
- [ ] Add fixtures for:
      - interface and implementation in the same package
      - interface and implementation across packages
      - embedded interfaces
      - duplicate method names with different signatures
      - no known local implementer
- [ ] Feed possible interface edges into context and impact limitations or
      relationship sections without making them direct impact evidence unless
      explicitly documented.

Primary files:

- `internal/sherpa/call.go`
- `internal/sherpa/call_test.go`
- `internal/impact/interface.go`
- `internal/impact/interface_test.go`
- `internal/agentcontext/report.go`
- `internal/impact/report.go`

Exit criteria:

- Interface dispatch produces bounded possible edges where local implementers
  are known.
- Unknown or broad dynamic dispatch remains a limitation.
- Context, impact, and PR workflows do not overclaim possible edges.

Verification:

```bash
go test ./internal/sherpa ./internal/impact ./internal/agentcontext ./cmd/gosherpa
```

### Slice 2.3: Goroutines, Function Literals, And Function Values

Goal: handle common visible runtime call patterns conservatively.

Tasks:

- [ ] Treat `go Target(...)` as a possible runtime entry/call edge with reason
      `goroutine` while preserving the direct callee where already known.
- [ ] Detect immediately invoked function literals and function literals passed
      to simple local call sites where source ranges can be named clearly.
- [ ] Keep simple locally assigned function values that resolve to one static
      target as direct or currently supported behavior.
- [ ] Keep reassigned or escaping function values as limitations or possible
      edges only when a single bounded candidate set is visible.
- [ ] Add fixtures for:
      - direct goroutine start
      - goroutine function literal calling a target
      - local function value assigned once
      - reassigned function value
      - function value stored in struct field
      - escaping function value passed to unknown function
- [ ] Ensure entrypoint analysis can report goroutine-originating reachability
      where supported, with clear limitations.

Primary files:

- `internal/sherpa/call.go`
- `internal/sherpa/entrypoint.go`
- `internal/sherpa/call_test.go`
- `internal/sherpa/call_output_test.go`
- `cmd/gosherpa/main_test.go`
- `cmd/gosherpa/testdata/golden-json/entrypoints.json`

Exit criteria:

- Common visible goroutine and function-literal cases are no longer only vague
  limitations.
- Reassigned and escaping function values remain conservative.
- Entrypoint output stays readable and bounded.

Verification:

```bash
go test ./internal/sherpa ./cmd/gosherpa
```

### Slice 2.4: Standard-Library Runtime Wiring First Slice

Goal: infer a small set of common standard-library entrypoints without becoming
framework-specific.

Tasks:

- [ ] Add stdlib HTTP handler detection for common patterns:
      - functions or methods passed to `http.HandleFunc`
      - values passed to `http.Handle` when they have `ServeHTTP`
      - `http.Server{Handler: ...}` when statically visible
- [ ] Emit entrypoint kind values carefully. If new enum values are added,
      update schema docs, tests, and goldens in the same slice.
- [ ] Add limitations that framework routers and custom runtime wiring remain
      outside this first slice.
- [ ] Add fixtures for:
      - function handler
      - method handler
      - handler interface value
      - custom router not inferred
      - ambiguous handler value
- [ ] Feed detected entrypoints into `entrypoints`, `context symbol`, `impact
      symbol`, and `pr` only where output stays bounded.

Primary files:

- `internal/sherpa/entrypoint.go`
- `internal/sherpa/call.go`
- `internal/sherpa/call_test.go`
- `internal/sherpa/entrypoint_output.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/main_test.go`
- `docs/product/JSON_SCHEMA_V1.md`
- `docs/STATUS.md`

Exit criteria:

- GoSherpa can name obvious stdlib HTTP entrypoints.
- Unsupported framework routing remains documented as a limitation.
- JSON and human output agree.

Verification:

```bash
go test ./internal/sherpa ./internal/agentcontext ./internal/impact ./cmd/gosherpa
go run ./cmd/gosherpa entrypoints <target> --json
```

### Slice 2.5: Imported Receiver Boundary Signals

Goal: make imported receiver calls visible as bounded external relationship
signals without pretending GoSherpa can inspect arbitrary dependency behavior.

Tasks:

- [ ] Use typechecked selector information to name imported package receiver
      method calls when the method object and import path are visible.
- [ ] Emit these as external boundary edges or possible call records with call
      site position and range. Do not require a repository-local definition
      range for imported methods.
- [ ] Preserve existing direct local caller/callee behavior and existing
      `CallScopeExternal` semantics.
- [ ] Do not parse the module cache, vendor trees, or remote dependencies in
      this slice unless a separate design note explicitly approves that scope.
- [ ] Add limitations that external dependency internals and runtime dispatch
      remain outside local impact evidence.
- [ ] Add fixtures for:
      - standard-library receiver method
      - imported module or replace-module receiver method
      - alias import selector
      - dot import or unsupported selector shape
      - local receiver call remains direct/local
- [ ] Feed imported receiver boundary signals into context, impact, and PR only
      as bounded relationship or limitation evidence, not as local affected
      package evidence.

Primary files:

- `internal/sherpa/call.go`
- `internal/sherpa/call_test.go`
- `internal/agentcontext/report.go`
- `internal/impact/report.go`
- `cmd/gosherpa/testdata/golden-json/callees.json`
- `cmd/gosherpa/testdata/golden-json/context-symbol.json`

Exit criteria:

- Imported receiver calls are no longer only invisible or vague limitations
  when typechecked selector data can name them.
- External calls do not become local impact evidence.
- JSON and human output make the external boundary clear.

Verification:

```bash
go test ./internal/sherpa ./internal/agentcontext ./internal/impact ./cmd/gosherpa
```

Phase 2 is done when:

- Direct and possible call edges are clearly separated.
- Interface dispatch, selected runtime wiring, and imported receiver boundaries
  improve impact and entrypoint context without false certainty.
- Limitations remain visible and specific.
- Tests cover supported static cases and intentionally unsupported dynamic
  cases.

## Phase 3: Target-Aware Impact And Risk Summary

Goal: turn existing impact evidence into a concise, target-specific judgment:
local, package-level, cross-package, exported API, contract-level, or high
fan-in.

Why this comes third:

GoSherpa already reports changed files, packages, symbols, callers,
interfaces, tests, and PR risk notes. The next usability step is helping a user
or agent quickly decide how wide a change looks and what evidence supports that
judgment.

### Slice 3.1: Target Risk Model

Goal: define a stable target risk summary separate from repository structural
risk.

Tasks:

- [ ] Add a target risk summary type with:
      - `level`: use existing risk levels where appropriate, not confidence
      - `scope`: `local`, `package`, `cross-package`, `exported-api`,
        `interface-contract`, or another documented stable value
      - `reasons`: short evidence-backed strings
      - `signals`: structured counts or facts where useful
      - `limitations`: target-specific uncertainty
- [ ] Choose JSON field names before implementation. Prefer additive
      `targetRisk` fields for impact, context, and PR outputs unless a slice
      intentionally evolves an existing `risk` field. Do not create two
      ambiguous meanings for `risk` in the same output.
- [ ] Keep repository risk (`gosherpa risk`) separate from target impact risk.
- [ ] Define deterministic scoring rules based on current evidence:
      - number of affected packages
      - number of direct references
      - transitive caller packages
      - exported symbol or exported type method
      - interface implementer or contract impact
      - package fan-in
      - high fan-in as a signal or reason, not a scope value unless documented
      - missing or fallback test evidence
      - possible call edges from Phase 2 when available
- [ ] Add tests for local, package-level, cross-package, exported API, and
      interface contract cases.

Primary files:

- `internal/impact/report.go`
- `internal/sherpa/impact.go`
- `internal/sherpa/risk.go`
- `internal/explain/report.go`
- `internal/agentcontext/report.go`
- `internal/impact/report_test.go`
- `internal/sherpa/impact_test.go`

Exit criteria:

- Target risk is deterministic and evidence-backed.
- Risk wording does not hide raw impact data.
- Confidence remains a separate trust signal.

Verification:

```bash
go test ./internal/impact ./internal/sherpa ./internal/explain ./internal/agentcontext
```

### Slice 3.2: Impact Command Integration

Goal: expose target risk summaries in `impact file|package|symbol|diff`.

Tasks:

- [ ] Add target risk summaries to impact reports.
- [ ] Preserve existing impact fields and ordering.
- [ ] Ensure `impact diff` summarizes both aggregate diff risk and individual
      changed-symbol risk where bounded.
- [ ] Add human output that is short enough to scan:
      - one headline line
      - a few reasons
      - raw evidence sections unchanged
- [ ] Add JSON schema coverage and update golden fixtures.
- [ ] Ensure empty or non-Go-only diffs still produce useful fallback risk
      wording without pretending symbol impact exists.

Primary files:

- `internal/impact/report.go`
- `internal/impact/report_output.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`
- `cmd/gosherpa/testdata/golden-json/impact-symbol.json`
- `cmd/gosherpa/testdata/golden-json/impact-diff.json`

Exit criteria:

- `impact` answers "how wide does this change look?" directly.
- JSON consumers receive structured reasons and signals.
- Human output remains concise.

Verification:

```bash
go test ./internal/impact ./cmd/gosherpa
go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --json
go run ./cmd/gosherpa impact diff --base <base-ref> --json
```

### Slice 3.3: Context And PR Integration

Goal: make risk summaries visible where users actually start work.

Tasks:

- [ ] Add target risk summary to `context symbol`, `context file`,
      `context package`, and `context diff` only where it helps and remains
      bounded.
- [ ] Align `pr --base <ref>` risk summary with the new target risk model while
      preserving existing repository risk.
- [ ] Ensure `pr` distinguishes:
      - diff target risk
      - repository structural risk
      - test plan confidence and fallback breadth
- [ ] Add risk summary reasons to reading order or suggested next commands only
      if they point to real evidence.
- [ ] Update golden JSON fixtures for context and PR outputs.
- [ ] Update `AGENT_NOTES.md` and `llms.txt` so agents read risk summaries
      without treating them as proof.

Primary files:

- `internal/agentcontext/report.go`
- `internal/agentcontext/file_report.go`
- `internal/agentcontext/package_report.go`
- `internal/agentcontext/diff_report.go`
- `internal/agentcontext/output.go`
- `cmd/gosherpa/pr.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/testdata/golden-json/context-symbol.json`
- `cmd/gosherpa/testdata/golden-json/context-diff.json`
- `cmd/gosherpa/testdata/golden-json/pr.json`

Exit criteria:

- Context and PR outputs show a concise target risk judgment.
- Repository risk and target risk are not conflated.
- Agent docs tell consumers to inspect reasons, evidence, limitations, and
  tests before acting.

Verification:

```bash
go test ./internal/agentcontext ./cmd/gosherpa
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --json
go run ./cmd/gosherpa context diff --base <base-ref> --json
go run ./cmd/gosherpa pr --base <base-ref> --json
```

### Slice 3.4: Documentation And Compatibility Pass

Goal: keep the public contract clear after risk summaries become first-class.

Tasks:

- [ ] Update:
      - `docs/CLI_REFERENCE.md`
      - `docs/STATUS.md`
      - `docs/product/MUST_USE_READINESS.md`
      - `docs/product/JSON_SCHEMA_V1.md`
      - `docs/product/CONTEXT_SCHEMA_V1.md`
      - `README.md` if examples materially change
- [ ] Ensure examples use `<base-ref>` unless a real verified ref is needed.
- [ ] Document risk summary as deterministic evidence, not a defect prediction.
- [ ] Document how possible call edges affect risk, if Phase 2 has landed.
- [ ] Document the compatibility rule for existing `risk` fields versus new
      `targetRisk` fields.
- [ ] Run schema and golden tests.

Primary files:

- docs listed above
- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`

Exit criteria:

- Docs, schema tests, and golden fixtures agree.
- Users can understand the difference between confidence, limitations,
  repository risk, and target risk.

Verification:

```bash
go test ./cmd/gosherpa
go test ./...
```

Phase 3 is done when:

- Impact, context, and PR outputs include concise target-aware risk summaries.
- The summaries are deterministic, bounded, and evidence-backed.
- Risk never replaces raw impact facts, test recommendations, or limitations.

## Final Integration Pass

Goal: prove the three tracks work together as one product workflow.

Tasks:

- [ ] Select and record `<base-ref>`.
- [ ] Confirm that `context symbol --use-snapshot` and
      `impact symbol --use-snapshot` are supported by the CLI after Phase 1.4.
      If either remains intentionally unsupported, run the non-snapshot form
      and record the reason.
- [ ] Refresh or create a snapshot:

      ```bash
      go run ./cmd/gosherpa snapshot --json
      ```

- [ ] Run the must-use diff-oriented command set:

      ```bash
      go run ./cmd/gosherpa doctor --json
      go run ./cmd/gosherpa context diff --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa impact diff --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa tests affected --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa pr --base <base-ref> --use-snapshot --json
      ```

- [ ] Run the symbol command pair. Use this pair if Phase 1.4 intentionally
      enabled snapshot reuse for symbol commands:

      ```bash
      go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
      go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --use-snapshot --json
      ```

- [ ] If symbol command snapshot reuse remains intentionally unsupported, run
      this pair instead and record the reason:

      ```bash
      go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --json
      go run ./cmd/gosherpa impact symbol ./internal/sherpa.PlanTests --json
      ```

- [ ] Verify the outputs answer:
      - what changed or what is inspected
      - where relevant code lives
      - who calls it and what it calls
      - which possible runtime relationships are visible
      - which interfaces or implementers are involved
      - which packages and tests may be affected
      - how wide the change appears
      - what uncertainty remains
      - which command should run next
- [ ] Run `go test ./...`.
- [ ] Update `docs/STATUS.md` with completed readiness improvements.
- [ ] Update `docs/product/MUST_USE_READINESS.md` if readiness estimate or
      priority order changes.
- [ ] Update `README.md`, `AGENT_NOTES.md`, and `llms.txt` if workflows or
      agent guidance materially change.

Exit criteria:

- Valid relationship snapshots improve the main workflow without hiding
  fallback.
- Runtime-aware possible calls improve context while preserving conservative
  claims.
- Target risk summaries are useful and evidence-backed.
- Trust fields remain visible and deterministic.
- Human output and JSON output remain stable enough for daily use.

## Out Of Scope Until These Phases Are Done

Do not prioritize these ahead of the three tracks above:

- MCP or long-running server mode
- batch query mode
- TUI
- editor integration hooks
- graph export
- GitHub Action
- broad architecture reports beyond current risk and architecture commands
- project-specific configuration as a requirement for basic use
- full SSA, whole-program pointer analysis, reflection analysis, or runtime
  framework inference
- automatic refactoring or code modification

These can be valuable later, but they should follow relationship reuse,
runtime-aware possible calls, and target-aware impact risk.

## Suggested PR Slice Order

1. Baseline audit and design lock.
2. Relationship index in-memory contract.
3. Snapshot format and `doctor` metadata for relationship data.
4. Snapshot reuse for `refs`, `callers`, `callees`, and interface commands.
5. Snapshot reuse for `context`, `impact`, `tests affected`, and `pr`.
6. Possible call JSON model and limitations.
7. Interface-dispatch possible edges.
8. Goroutine, function literal, and bounded function-value possible edges.
9. Standard-library HTTP entrypoint first slice.
10. Imported receiver boundary signals.
11. Target risk model.
12. Impact command risk summary integration.
13. Context and PR risk summary integration.
14. Documentation, schema, golden fixture, and final integration pass.

Each PR should leave `go test ./...` passing unless it explicitly documents a
pre-existing failure before edits begin.
