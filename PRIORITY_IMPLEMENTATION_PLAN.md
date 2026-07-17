# GoSherpa Priority Implementation Plan

This plan replaces the completed relationship-reuse, possible-call, and
target-risk plan. Those tracks are now part of the product baseline. The next
priority is not to add more analysis facts for their own sake; it is to make
GoSherpa the tool an agent naturally runs every day in ordinary Go
repositories.

The must-use habit should feel like this:

```bash
gosherpa doctor --json
gosherpa agent context --base <base-ref> --use-snapshot --json
gosherpa context symbol ./internal/package.Target --use-snapshot --json
gosherpa tests affected --base <base-ref> --use-snapshot --json
```

The command name and first public scope are fixed by this plan:
`gosherpa agent context --base <base-ref>`. The first implementation is
diff-first. Symbol, file, and package drill-down stay with the existing
`gosherpa context` commands until a later plan explicitly adds target modes to
the agent workflow.

Primary source documents and required reconciliation targets:

- `docs/product/MUST_USE_READINESS.md`
- `docs/product/AGENT_PRIORITY_LIST.md`
- `docs/product/FEATURE_ROADMAP.md`
- `docs/product/AGENT_RECOMMENDATION_CRITERIA.md`
- `docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md`
- `docs/STATUS.md`
- `AGENT_NOTES.md`
- `llms.txt`

This file is the active implementation source of truth. Some product documents
still describe the previous priority framing or say command shape is no longer
the main gap. When those documents disagree with this plan, implement this plan
and reconcile the conflicting document in Phase 0 before feature work starts.

## Current Baseline

The previous priority plan completed the current daily-use foundation:

- typechecked loading and shared semantic context across the main workflows
- context exports for symbol, file, package, and diff targets
- relationship-capable snapshot format v2 and opt-in relationship reuse
- bounded possible-call signals for interface dispatch, goroutines, function
  literals, selected standard-library HTTP wiring, and imported receiver calls
- structured test planning with direct, related, contracts, caller-package, and
  fallback groups
- target-aware risk summaries in impact, context, explain, and PR outputs
- stable JSON envelope, golden fixtures, warnings, limitations, source ranges,
  and ambiguity diagnostics

The remaining must-use gap is product reliability in real repositories:

1. Agents still have to stitch together several commands to get the standard
   pre-edit or pre-PR answer.
2. Ordinary Go repositories often contain workspaces, nested modules, build
   tags, generated code, partial package failures, local replacements, and
   large monorepo boundaries.
3. Test and entrypoint recommendations need to understand more of how real Go
   programs are invoked and verified.

## Priority Order

1. **Zero-Friction Agent Workflow**
   Add one primary agent-facing workflow command that composes the existing
   intelligence into a bounded, stable, everyday answer.

2. **Real-World Repository Robustness**
   Make analysis behavior explicit and reliable across `go.work`, nested
   modules, build tags, generated files, local replacements, partial package
   failures, and large repositories.

3. **Tests And Entrypoint Intelligence**
   Promote tests and runtime entrypoints into stronger first-class signals so
   agents know what to read, what to change carefully, and what to run next.

Do not prioritize MCP, TUI, editor integration, graph export, hosted services,
automatic refactoring, or broad framework inference before these tracks are
done. The only new broad command in scope is the agent workflow entrypoint.

## Product Standard

This plan is complete when an agent can use GoSherpa in most Go repositories
without bespoke prompting:

- Start with one command for a diff-oriented context bundle, then use focused
  commands for symbol, file, or package drill-down.
- Receive a concise explanation of what matters and why.
- See exact source locations and bounded reading order.
- See direct and possible relationships without conflating certainty.
- See target risk, affected packages, and recommended tests.
- See repository-readiness warnings before trusting incomplete analysis.
- Fall back gracefully when packages do not typecheck or snapshots are stale.
- Consume stable JSON without losing readable human output.

## Primary Code Contracts To Preserve

- JSON responses use the shared envelope in `cmd/gosherpa/cli_json.go`.
- Stable JSON examples live in `cmd/gosherpa/testdata/golden-json/`.
- Schema guardrails live in `cmd/gosherpa/json_schema_test.go` and
  `cmd/gosherpa/json_golden_test.go`.
- Command and flag support is defined by `cmd/gosherpa/command_registry.go`,
  `cmd/gosherpa/cli_flags.go`, completion generation, and validation in
  `cmd/gosherpa/main.go`.
- Human output must remain readable and stable; successful output goes to
  stdout, diagnostics go to stderr.
- Confidence remains separate from risk. Risk describes likely blast radius;
  confidence describes trust in the analysis.
- Repository structural risk remains separate from `targetRisk`.
- Direct call edges and possible runtime edges must remain distinct.
- Existing `sherpa.CallScope` values describe callee locality, not certainty.
- Snapshot reuse must stay opt-in unless a slice explicitly changes that
  contract with tests, docs, schema notes, and fallback behavior.
- Current snapshot reuse accepts `--use-snapshot` on `analyze`, `symbols`,
  `symbol`, `search`, `packages`, `refs`, `callers`, `callees`,
  `implementers`, `interface`, `interfaces`, `context symbol`, `context diff`,
  `impact symbol`, `impact diff`, `tests affected`, and `pr`. Unsupported
  portions must continue to report live fallback honestly. `packages` accepts
  the flag, but current reuse is meaningful only for the test-inclusive
  inventory shape; preserve or explicitly change that nuance with tests.
- Any new snapshot-backed data must have explicit compatibility inputs,
  relationship-capability metadata when applicable, stale/invalid diagnostics,
  and tests that prove old snapshots are not silently trusted as richer data.
- Plain `tests <target>` and `tests affected --base <ref>` remain separate CLI
  contracts.
- The new agent workflow is a top-level `agent` command with a required
  `context` subcommand. CLI changes must update registry specs, parsing,
  validation helpers, usage text, help output, completion output, validation
  tests, CLI docs, schema docs, and golden JSON together.
- Global `--root` must work for every new agent workflow check. Do not add a
  second root-selection mechanism.
- JSON schema version remains `1` unless a slice intentionally makes a
  breaking contract change. New command-specific fields should be additive,
  non-nil where arrays are expected, deterministic, and documented before or in
  the same slice that exposes them.
- Use `<base-ref>` in docs and verification commands. Pick a concrete existing
  ref such as `origin/main`, `main`, or a documented test ref at the start of a
  slice and record it in the verification note.

## Execution Rules For Agents

Before starting any slice:

- Read the source files listed for that slice.
- Re-read command support in `cmd/gosherpa/command_registry.go`,
  `cmd/gosherpa/cli_flags.go`, completion generation, and validation in
  `cmd/gosherpa/main.go` before changing CLI behavior.
- Reconcile or explicitly annotate source-document conflicts found in
  `docs/product/MUST_USE_READINESS.md`, `docs/product/AGENT_PRIORITY_LIST.md`,
  `docs/product/FEATURE_ROADMAP.md`, `docs/STATUS.md`, `AGENT_NOTES.md`, or
  `llms.txt` before relying on those documents for implementation details.
- Inspect existing focused tests for the area.
- Select and record a concrete `<base-ref>` for diff-oriented checks.
- Run `git rev-parse --verify <base-ref>` or record why the selected ref is
  unavailable in the current fixture.
- Create or refresh a snapshot before any verification command that is meant to
  prove snapshot reuse. If the slice intentionally verifies stale/missing
  fallback, say that explicitly in the verification note.
- Preserve AST fallback behavior when typechecked loading is unavailable.
- Surface uncertainty as warnings, `analysisMode`, subanalysis modes,
  `confidence`, `limitations`, `targetRisk.limitations`, and explicit
  relationship certainty labels.
- Keep JSON schemas, golden fixtures, human output, and docs aligned.
- Keep changes PR-sized. If a slice reveals a deeper issue, record it as a
  follow-up instead of expanding the slice without bound.

Every completed slice must include:

- focused tests for changed behavior
- CLI or golden JSON coverage when command output changes
- documentation updates when JSON shape, user-visible behavior, limitations,
  readiness status, or flag support changes
- a verification note listing exact commands run and the selected `<base-ref>`

Recommended baseline verification:

```bash
git rev-parse --verify <base-ref>
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa snapshot --json
go run ./cmd/gosherpa context diff --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa impact diff --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa tests affected --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa pr --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
```

## Phase 0: Baseline Audit And Command Design Lock

Goal: turn the three new tracks into concrete implementation boundaries before
changing behavior.

Tasks:

- [ ] Run the baseline verification commands above.
- [ ] Append a new dated section to
      `docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md`; preserve its historical
      relationship-reuse/possible-call/target-risk entries as completed
      evidence, not as the active phase map.
- [ ] Record current runtime, warnings, analysis modes, target risk summaries,
      and snapshot status in `docs/product/PRIORITY_IMPLEMENTATION_AUDIT.md`.
- [ ] Record the current CLI flag matrix from `command_registry.go`,
      `cli_flags.go`, `main.go`, and `completion.go`, including the exact
      validation messages that must change for `agent context`.
- [ ] Confirm the locked command shape:
      `gosherpa agent context --base <base-ref>`.
- [ ] Confirm that the first public command is diff-first only. It must not
      accept symbol, file, package, or free-form task targets in this plan.
- [ ] Document the first JSON shape for the composed agent workflow output from
      the locked contract below.
- [ ] Record which fields are embedded from existing report models and which
      are summarized to keep output bounded.
- [ ] Record real-world repository fixture needs for workspaces, nested
      modules, build tags, generated files, local replacements, and partial
      package failures.
- [ ] Record the first test and entrypoint fixture matrix.
- [ ] Update or annotate `docs/product/MUST_USE_READINESS.md` and
      `docs/product/AGENT_PRIORITY_LIST.md` so they no longer contradict this
      plan's active priority order.

Primary files to inspect:

- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/command_handlers.go`
- `cmd/gosherpa/completion.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/pr.go`
- `cmd/gosherpa/snapshot_reuse.go`
- `internal/agentcontext/diff_report.go`
- `internal/agentcontext/report.go`
- `internal/impact/report.go`
- `internal/sherpa/test.go`
- `internal/sherpa/entrypoint.go`
- `internal/semantics/repository.go`
- `internal/sherpa/workspace.go`
- `internal/snapshot/snapshot.go`
- `docs/product/MUST_USE_READINESS.md`
- `docs/product/AGENT_PRIORITY_LIST.md`
- `docs/product/FEATURE_ROADMAP.md`
- `docs/product/JSON_SCHEMA_V1.md`
- `docs/product/CONTEXT_SCHEMA_V1.md`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`
- `AGENT_NOTES.md`
- `llms.txt`

Exit criteria:

- The new command contract is explicit.
- The first JSON shape is documented before implementation.
- Current limitations are recorded against the new plan.
- Fixture scope is known before feature work starts.

Verification:

```bash
go test ./...
go run ./cmd/gosherpa doctor --json
```

## Locked Agent Workflow Contract

The first agent workflow command is:

```bash
gosherpa agent context --base <base-ref> [--use-snapshot] [--tags <list>] [--max-files <n>] [--max-symbols <n>] [--max-tests <n>] [--json]
```

Global `--root <path>` must work. Slice 1.3 adds `--max-bytes <n>` once the
byte-budget behavior is implemented. The first public command must reject
symbol/file/package positional targets, `--max-references`, `--source-radius`,
`--scope`, `--tests`, and any other inherited flag whose semantics are not
explicitly defined here. Affected-test planning is part of the agent workflow by
default; `--tests` is not required to receive recommended tests.

Behavior:

- It is diff-first and requires `--base <base-ref>`.
- It never shells out to `gosherpa`; it composes Go APIs and existing command
  helpers directly.
- It does not create snapshots automatically. With `--use-snapshot`, valid
  snapshots may be reused and missing, stale, invalid, or relationship-limited
  snapshots must fall back with envelope warnings and a suggested refresh
  command.
- It uses the shared JSON envelope with `command: "agent context"` and
  `target` set to the selected base ref.
- Human output is intentionally short: readiness, target risk, changed targets,
  recommended tests, warnings, and next commands.

Initial JSON `data` fields:

- `target`, `base`, and `purpose`
- `readiness`: bounded repository readiness summary, package-load status, and
  repo-shape warnings
- `snapshot`: requested/used status, freshness, relationship capability, and
  refresh guidance without embedding full snapshot records
- `changedFiles`, `changedPackages`, `affectedPackages`, `affectedSymbols`,
  and bounded `changedSymbolDetails`
- `readingOrder`
- `targetRisk`
- `possibleRuntimeRelationships`: counts by reason/scope/certainty plus
  bounded examples only when certainty labels and limitations are present
- `interfaceSummary`: bounded affected interfaces and implementations
- `testPlan` and `testCommands`
- `suggestedCommands`
- `sectionModes`: deterministic array of `{section, analysisMode, confidence,
  limitations}` entries for readiness, context, impact, tests, snapshot, and PR
  subanalysis where applicable
- `analysisMode`, `confidence`, `limitations`, `limits`, and `truncated`

Do not embed full `context diff`, `impact diff`, `tests affected`, and `pr`
reports wholesale. The agent workflow is a composed summary with links back to
focused commands, not a JSON dump of every existing report.

## Phase 1: Zero-Friction Agent Workflow

Goal: make one command provide the everyday pre-edit or pre-PR bundle an agent
needs.

Why this comes first:

GoSherpa already has the underlying facts. Daily use depends on reducing the
coordination cost. An agent should not have to independently run `doctor`,
`context diff`, `impact diff`, `tests affected`, and `pr`, then reconcile their
warnings and risk summaries by hand.

### Slice 1.1: Agent Workflow Data Model

Goal: define the internal and JSON model for the composed workflow before
adding a public command.

Tasks:

- [ ] Add `internal/agentworkflow` with the report model, normalizers, ordering
      helpers, and focused unit tests. Keep `cmd/gosherpa` as a thin CLI layer.
- [ ] Include bounded sections for:
      - readiness summary
      - snapshot status and reuse status
      - changed files, packages, and symbols
      - reading order
      - target risk
      - affected packages
      - possible runtime relationships summary
      - interface and implementer summary
      - structured test plan
      - suggested next commands
      - warnings, limitations, confidence, and truncation metadata
- [ ] Reuse existing report types where stable, but avoid dumping multiple full
      reports into one oversized JSON object.
- [ ] Define stable summary adapters for any reused `context diff`,
      `impact diff`, `tests affected`, `pr`, and `doctor` data.
- [ ] Ensure non-nil arrays and deterministic ordering.
- [ ] Define how subanalysis modes are represented when different sections use
      snapshot, typechecked, AST, or live fallback data.
- [ ] Keep the model additive to JSON schema v1; do not introduce a breaking
      schema-version change for the first command.
- [ ] Add focused model tests before CLI exposure.

Primary files:

- `internal/agentworkflow/`
- `internal/agentcontext/report.go`
- `internal/agentcontext/diff_report.go`
- `internal/impact/report.go`
- `internal/sherpa/test_plan.go`
- `internal/sherpa/target_risk.go`
- `cmd/gosherpa/cli_json.go`

Exit criteria:

- The composed bundle can be built internally from existing analyzers.
- It remains bounded and deterministic.
- It preserves uncertainty from each contributing analysis.

Verification:

```bash
go test ./internal/agentworkflow ./internal/agentcontext ./internal/impact ./internal/sherpa
```

### Slice 1.2: Public Agent Command

Goal: expose the first daily-driver command.

Locked CLI shape:

```bash
gosherpa agent context --base <base-ref> --use-snapshot --json
```

Tasks:

- [ ] Add the selected command and subcommand to the command registry.
- [ ] Implement `agent` as a top-level command with a required `context`
      subcommand and no other `agent` subcommands in this plan.
- [ ] Add usage, validation, completion, and help output for `agent context`.
- [ ] Support exactly `--base`, `--use-snapshot`, `--max-files`,
      `--max-symbols`, `--max-tests`, `--tags`, and `--json` in this slice,
      plus global `--root`.
- [ ] Reject `agent context` without `--base` and reject symbol/file/package
      positional targets, `--max-references`, `--source-radius`, `--scope`, and
      `--tests`. Also reject `--max-bytes` until Slice 1.3 implements
      byte-budget behavior.
- [ ] Update validation error text for `--base`, `--use-snapshot`, `--tags`,
      and allowed context-limit flags so `agent context` is listed accurately.
- [ ] Compose existing readiness, context, impact, affected-test, and PR
      logic through Go APIs instead of shelling out. Extract reusable helpers
      only where needed; do not move unrelated CLI behavior in this slice.
- [ ] Keep human output short: a readiness line, a risk line, key changed
      targets, recommended tests, warnings, and next commands.
- [ ] Ensure JSON output uses the shared envelope.
- [ ] Add golden JSON coverage and CLI validation tests.
- [ ] Update `docs/CLI_REFERENCE.md`, `AGENT_NOTES.md`, `llms.txt`, and
      schema docs.

Primary files:

- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/command_handlers.go`
- `cmd/gosherpa/completion.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/agent.go`
- `cmd/gosherpa/main_test.go`
- `cmd/gosherpa/testdata/golden-json/`
- `internal/agentworkflow/`
- `internal/agentcontext/`
- `internal/impact/`
- `internal/sherpa/`

Exit criteria:

- One command gives a coherent pre-edit/pre-PR answer for a diff.
- Snapshot fallback warnings are visible but not noisy.
- Existing commands remain stable.

Verification:

```bash
go test ./cmd/gosherpa ./internal/agentworkflow ./internal/agentcontext ./internal/impact ./internal/sherpa
go run ./cmd/gosherpa help agent
go run ./cmd/gosherpa help agent context
go run ./cmd/gosherpa agent context --base <base-ref> --use-snapshot --json
```

### Slice 1.3: Agent Workflow Size And Trust Controls

Goal: make the command safe for large repositories and bounded agent context
windows.

Tasks:

- [ ] Enable `--max-bytes` for `agent context` and apply it to the composed
      JSON `data` payload, not the shared envelope, without producing invalid
      JSON.
- [ ] Add per-section truncation metadata.
- [ ] Make the reading order budget-aware.
- [ ] Add summary-first output for large diffs.
- [ ] Ensure huge generated files, broad reference sets, and large test plans
      degrade into summaries with limitations.
- [ ] Keep a minimum valid report shell under tight budgets and report any
      remaining `byteBudgetOverage`.
- [ ] Add fixtures or tests for truncation and large-section behavior.

Primary files:

- `internal/agentcontext/byte_limit.go`
- `internal/agentcontext/limits.go`
- `internal/agentcontext/test_order.go`
- `internal/agentworkflow/`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`
- `cmd/gosherpa/json_golden_test.go`
- `cmd/gosherpa/json_schema_test.go`

Exit criteria:

- The agent workflow remains useful under strict byte limits.
- Truncation is explicit, deterministic, and schema-covered.

Verification:

```bash
go test ./internal/agentworkflow ./internal/agentcontext ./cmd/gosherpa
go run ./cmd/gosherpa agent context --base <base-ref> --max-bytes 12000 --json
```

### Slice 1.4: Snapshot Ergonomics For Daily Use

Goal: make valid snapshot reuse easy without hiding freshness problems.

Tasks:

- [ ] Do not auto-create snapshots in this plan. The agent workflow may only
      suggest `gosherpa snapshot --json` or report that a valid snapshot was
      used.
- [ ] Add a concise stale/missing snapshot recommendation to the agent command.
- [ ] Ensure `doctor`, `snapshot`, and the agent workflow use consistent
      freshness wording.
- [ ] Add tests for missing, stale, invalid, and valid snapshot behavior.
- [ ] Document the recommended daily loop:
      `snapshot`, `agent context`, focused `context symbol`, `tests affected`.

Primary files:

- `cmd/gosherpa/snapshot.go`
- `cmd/gosherpa/doctor.go`
- `cmd/gosherpa/snapshot_reuse.go`
- `internal/snapshot/snapshot.go`
- `internal/agentworkflow/`
- `docs/CLI_REFERENCE.md`
- `AGENT_NOTES.md`
- `llms.txt`

Exit criteria:

- Agents know when snapshot data was used.
- Agents know exactly how to refresh stale data.
- Snapshot status is visible without overwhelming the core answer.

Verification:

```bash
go test ./internal/snapshot ./internal/agentworkflow ./cmd/gosherpa
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa agent context --base <base-ref> --use-snapshot --json
```

Phase 1 is done when:

- A single command answers the standard agent pre-edit/pre-PR question.
- The command is bounded, deterministic, schema-covered, and documented.
- Existing focused commands remain useful for follow-up drilling.

## Phase 2: Real-World Repository Robustness

Goal: make GoSherpa trustworthy across the repository shapes agents actually
encounter.

Why this comes second:

Once there is one daily command, repo-shape failures become the next trust
gate. The command must explain what it could analyze, what it skipped, and how
that affects the answer.

### Slice 2.1: Workspace And Module Boundary Audit

Goal: make `go.work`, nested modules, and module boundaries explicit.

Tasks:

- [ ] Audit current behavior for:
      - root with `go.mod`
      - root with `go.work`
      - nested modules below a module root
      - workspace modules outside the selected root
      - local `replace` modules
- [ ] Add a repository layout summary used by `doctor` and the agent workflow.
- [ ] Report skipped nested modules and explain how to inspect them with
      `--root`.
- [ ] Ensure file walking, symbol lookup, references, context, impact, and
      tests agree on module boundaries.
- [ ] Add fixtures for single module, workspace, nested module, and local
      replace-module cases.

Primary files:

- `internal/sherpa/workspace.go`
- `internal/sherpa/root.go`
- `internal/sherpa/scan.go`
- `internal/semantics/repository.go`
- `cmd/gosherpa/doctor.go`
- `internal/agentworkflow/`
- `internal/agentcontext/`

Exit criteria:

- GoSherpa can explain the selected analysis boundary.
- Nested or external workspace modules are not silently ignored.
- Existing root-selection behavior remains stable.

Verification:

```bash
go test ./internal/sherpa ./internal/semantics ./internal/agentworkflow ./cmd/gosherpa
go run ./cmd/gosherpa doctor --json
```

### Slice 2.2: Build Tags And Package Loading Diagnostics

Goal: make build constraints and partial typechecking failures actionable.

Tasks:

- [ ] Ensure `--tags` compatibility is consistently represented in snapshot
      freshness, `doctor`, context, impact, tests, and the agent workflow.
- [ ] Report package loading failures with package path, file when known,
      concise reason, and affected analysis sections.
- [ ] Distinguish "package failed to load" from "package loaded with type
      errors" where `go/packages` exposes that information.
- [ ] Add build-tag fixtures with included and excluded files.
- [ ] Add tests for stale snapshot behavior when tags differ.
- [ ] Ensure confidence and limitations reflect partial loading.

Primary files:

- `internal/semantics/repository.go`
- `internal/semantics/repository_cache.go`
- `internal/snapshot/snapshot.go`
- `cmd/gosherpa/doctor.go`
- `cmd/gosherpa/cli_flags.go`
- `internal/agentworkflow/`
- `internal/agentcontext/`
- `internal/impact/`

Exit criteria:

- Build-tag choices are visible and reproducible.
- Partial package failures reduce confidence without hiding useful results.
- Snapshot compatibility remains correct.

Verification:

```bash
go test ./internal/semantics ./internal/snapshot ./internal/agentworkflow ./internal/agentcontext ./cmd/gosherpa
```

### Slice 2.3: Generated File Policy

Goal: make generated code behavior explicit and useful.

Tasks:

- [ ] Centralize generated-file detection using the standard
      `// Code generated ... DO NOT EDIT.` convention.
- [ ] Add generated-file counts and major generated packages to `doctor` and
      the agent workflow.
- [ ] Include compiler-visible generated files in semantic analysis, but
      summarize and deprioritize large generated files in reading order when
      they would dominate the agent workflow output.
- [ ] Do not exclude compiler-visible generated files from semantic analysis
      unless a slice intentionally adds and documents a flag.
- [ ] Add fixtures for generated definitions, references, and tests.
- [ ] Ensure limitations explain when generated code dominates a result.

Primary files:

- `internal/sherpa/scan.go`
- `internal/sherpa/source_context.go`
- `internal/agentworkflow/`
- `internal/agentcontext/report.go`
- `cmd/gosherpa/doctor.go`
- `docs/CLI_REFERENCE.md`
- `docs/STATUS.md`

Exit criteria:

- Generated code is visible as a repository property.
- Reading order remains useful when generated files are large.
- Analysis does not silently disagree with the Go compiler's visible files.

Verification:

```bash
go test ./internal/sherpa ./internal/agentworkflow ./internal/agentcontext ./cmd/gosherpa
```

### Slice 2.4: Large Repository Performance Guardrails

Goal: keep daily commands fast enough and predictable in larger repositories.

Tasks:

- [ ] Add benchmark-style tests or regression checks for repeated query paths
      where wall-clock assertions are not brittle.
- [ ] Track counts that explain cost: packages, files, symbols, relationships,
      generated files, skipped modules, and package-load warnings.
- [ ] Ensure snapshot-backed agent workflow avoids repeated expensive loads.
- [ ] Add deterministic ordering and deduplication tests for any newly shared
      repo-shape summaries.
- [ ] Document when users should refresh snapshots.

Primary files:

- `internal/semantics/repository_cache.go`
- `internal/snapshot/snapshot.go`
- `cmd/gosherpa/snapshot_reuse.go`
- `internal/agentworkflow/`
- `internal/agentcontext/`
- `cmd/gosherpa/doctor.go`

Exit criteria:

- Repeated agent workflows benefit from snapshot and shared context reuse.
- Large-repo cost is visible in diagnostics.
- Tests avoid brittle timing assumptions.

Verification:

```bash
go test ./internal/semantics ./internal/snapshot ./internal/agentworkflow ./internal/agentcontext ./cmd/gosherpa
go test ./...
```

### Slice 2.5: Robustness Documentation And Fixtures Pass

Goal: make the supported repository matrix clear.

Tasks:

- [ ] Update `docs/STATUS.md` with current support and limitations for
      workspaces, nested modules, build tags, generated files, local replaces,
      partial package loads, and large repositories.
- [ ] Update `AGENT_NOTES.md` and `llms.txt` with guidance for interpreting
      repo-shape warnings.
- [ ] Add or update CLI reference examples for `--root`, `--tags`, snapshot
      freshness, and nested-module inspection.
- [ ] Ensure fixture names and tests make unsupported cases intentional.

Primary files:

- `docs/STATUS.md`
- `docs/CLI_REFERENCE.md`
- `AGENT_NOTES.md`
- `llms.txt`
- `cmd/gosherpa/testdata/`

Exit criteria:

- Users and agents can tell whether a repository shape is supported,
  partially supported, or intentionally out of scope.
- The docs match command behavior.

Verification:

```bash
go test ./cmd/gosherpa
go test ./...
```

Phase 2 is done when:

- GoSherpa explains repository boundaries and partial analysis clearly.
- The daily agent workflow remains useful in imperfect Go repositories.
- Workspaces, nested modules, build tags, generated code, and local
  replacements are covered by fixtures or explicit limitations.

## Phase 3: Tests And Entrypoint Intelligence

Goal: make "what should I run next?" and "how is this code reached?" stronger
first-class answers.

Why this comes third:

Agents trust a repository tool when it helps them act safely. Context and
impact are useful; concrete test and entrypoint guidance turns them into a
workflow.

### Slice 3.1: Test Inventory Model

Goal: centralize test discovery into a reusable model.

Tasks:

- [ ] Build or extend a test inventory that records packages, test files, test
      functions, subtests when statically visible, suite-like patterns when
      conservative, and target references.
- [ ] Preserve existing direct, related, contracts, caller-package, and fallback
      groups.
- [ ] Include source ranges for test functions and target references when
      available.
- [ ] Keep dynamic table-driven test names as limitations unless statically
      visible.
- [ ] Persist safe test-reference or test-inventory data in snapshots only when
      compatibility inputs are sufficient; if persisted shape changes, update
      snapshot format/capability metadata, stale diagnostics, docs, and tests in
      the same slice.
- [ ] Add fixtures for internal tests, external `_test` packages, subtests,
      table-driven literals, and suite-style helpers.

Primary files:

- `internal/sherpa/test.go`
- `internal/sherpa/test_plan.go`
- `internal/sherpa/test_output.go`
- `internal/symbolindex/relationship.go`
- `internal/snapshot/snapshot.go`
- `internal/agentworkflow/`
- `cmd/gosherpa/testdata/`

Exit criteria:

- Test discovery has one reusable source of truth.
- Existing test-plan JSON remains compatible or changes are additive and
  schema-covered.
- Dynamic test behavior is explained, not guessed.

Verification:

```bash
go test ./internal/sherpa ./internal/snapshot ./internal/agentworkflow ./cmd/gosherpa
```

### Slice 3.2: Stronger Affected-Test Planning

Goal: improve recommended tests for symbol, file, package, diff, and agent
workflow outputs.

Tasks:

- [ ] Feed the centralized test inventory into `tests`, `tests affected`,
      `context`, `impact`, `pr`, and the agent workflow.
- [ ] Improve package fallback selection when direct references are absent.
- [ ] Distinguish fast, focused, contract, caller-package, integration-like,
      and broad fallback commands where evidence supports it.
- [ ] Add reasons that explain why each command is recommended.
- [ ] Ensure generated files and skipped packages affect test confidence and
      limitations.
- [ ] Add golden JSON coverage for representative affected-test and agent
      workflow outputs.

Primary files:

- `internal/sherpa/test.go`
- `internal/sherpa/test_plan.go`
- `internal/impact/report.go`
- `internal/agentworkflow/`
- `internal/agentcontext/`
- `cmd/gosherpa/pr.go`
- `cmd/gosherpa/testdata/golden-json/`

Exit criteria:

- The smallest useful test set is easier to identify.
- Broader fallback commands remain available and clearly justified.
- Test-plan confidence is visible in agent-facing outputs.

Verification:

```bash
go test ./internal/sherpa ./internal/impact ./internal/agentworkflow ./internal/agentcontext ./cmd/gosherpa
go run ./cmd/gosherpa tests affected --base <base-ref> --use-snapshot --json
go run ./cmd/gosherpa agent context --base <base-ref> --use-snapshot --json
```

### Slice 3.3: Entrypoint Inventory Model

Goal: represent program entrypoints and runtime wiring as reusable evidence.

Tasks:

- [ ] Centralize entrypoint records for:
      - `main.main`
      - tests with `--tests`
      - exported functions
      - no-local-caller functions
      - stdlib `net/http` handlers already supported
      - visible goroutine origins already supported
- [ ] Include kind, reason, source range, reachable target when known,
      certainty, and limitations.
- [ ] Keep framework-specific routers out of scope unless a bounded pattern is
      explicitly accepted.
- [ ] Add fixtures for command packages, HTTP handlers, tests, workers, and
      unsupported custom routing.
- [ ] Define and implement the bounded entrypoint summary used by context,
      impact, PR, and the agent workflow: counts by kind, top reachable
      examples, certainty labels, source locations, and limitations.

Primary files:

- `internal/sherpa/entrypoint.go`
- `internal/sherpa/entrypoint_output.go`
- `internal/sherpa/call.go`
- `internal/agentworkflow/`
- `internal/agentcontext/report.go`
- `internal/impact/report.go`
- `cmd/gosherpa/testdata/golden-json/entrypoints.json`

Exit criteria:

- Entrypoints become reusable relationship evidence, not only a standalone
  command result.
- Unsupported runtime wiring remains visible as a limitation.

Verification:

```bash
go test ./internal/sherpa ./internal/agentworkflow ./internal/agentcontext ./internal/impact ./cmd/gosherpa
go run ./cmd/gosherpa entrypoints ./internal/sherpa.PlanTests --json
```

### Slice 3.4: Entrypoints In Context, Impact, PR, And Agent Workflow

Goal: show how inspected or changed code is reached.

Tasks:

- [ ] Add bounded entrypoint summaries to `context symbol`, `impact symbol`,
      `context diff`, `impact diff`, `pr`, and the agent workflow where
      relevant.
- [ ] Keep entrypoint evidence separate from direct caller evidence.
- [ ] Include possible runtime paths only when certainty labels and limitations
      are clear.
- [ ] Update target risk scoring only if entrypoint evidence materially changes
      blast-radius judgment; document the rule.
- [ ] Add human output that is concise enough not to crowd out tests and risk.
- [ ] Update golden JSON fixtures and schema docs.

Primary files:

- `internal/agentcontext/report.go`
- `internal/agentcontext/output.go`
- `internal/agentworkflow/`
- `internal/impact/report.go`
- `internal/impact/report_output.go`
- `cmd/gosherpa/pr.go`
- `cmd/gosherpa/cli_json.go`
- `cmd/gosherpa/testdata/golden-json/`

Exit criteria:

- Agents can see likely runtime reachability for changed or inspected targets.
- Entrypoint summaries improve planning without claiming full runtime coverage.

Verification:

```bash
go test ./internal/agentworkflow ./internal/agentcontext ./internal/impact ./internal/sherpa ./cmd/gosherpa
go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
go run ./cmd/gosherpa agent context --base <base-ref> --use-snapshot --json
```

### Slice 3.5: Test And Entrypoint Documentation Pass

Goal: keep users and agents calibrated.

Tasks:

- [ ] Update `docs/CLI_REFERENCE.md`, `docs/STATUS.md`,
      `docs/product/JSON_SCHEMA_V1.md`, `docs/product/CONTEXT_SCHEMA_V1.md`,
      `AGENT_NOTES.md`, and `llms.txt`.
- [ ] Document the difference between direct tests, related tests, contract
      tests, caller-package tests, integration-like tests, and fallbacks.
- [ ] Document entrypoint certainty and unsupported runtime wiring.
- [ ] Ensure examples use `<base-ref>` unless recording a verified local ref.
- [ ] Run schema and golden tests.

Primary files:

- docs listed above
- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`

Exit criteria:

- Test and entrypoint claims are clear, bounded, and documented.
- Agents know how to interpret missing direct tests and unsupported runtime
  wiring.

Verification:

```bash
go test ./cmd/gosherpa
go test ./...
```

Phase 3 is done when:

- Test recommendations are stronger, grouped, and evidence-backed.
- Entrypoint reachability appears in the workflows where it changes planning.
- Runtime and test uncertainty stay visible.

## Final Integration Pass

Goal: prove the three tracks work together as one product workflow.

Tasks:

- [ ] Select and record `<base-ref>`.
- [ ] Run `git rev-parse --verify <base-ref>` and record the resolved commit.
- [ ] Refresh or create a snapshot:

      ```bash
      go run ./cmd/gosherpa snapshot --json
      ```

- [ ] Run:

      ```bash
      git rev-parse --verify <base-ref>
      go test ./...
      go run ./cmd/gosherpa doctor --json
      go run ./cmd/gosherpa agent context --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa context diff --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa impact diff --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa tests affected --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa pr --base <base-ref> --use-snapshot --json
      go run ./cmd/gosherpa context symbol ./internal/sherpa.PlanTests --use-snapshot --json
      go run ./cmd/gosherpa entrypoints ./internal/sherpa.PlanTests --json
      ```

- [ ] Verify the outputs answer:
      - what changed or what is inspected
      - where relevant code lives
      - who calls it and what it calls
      - which possible runtime relationships are visible
      - which entrypoints may reach it
      - which interfaces or implementers are involved
      - which packages and tests may be affected
      - how wide the change appears
      - what uncertainty remains
      - which command or test should run next
- [ ] Update `docs/STATUS.md` with completed readiness improvements.
- [ ] Update `docs/product/MUST_USE_READINESS.md` if readiness estimate or
      priority order changes.
- [ ] Update `README.md`, `AGENT_NOTES.md`, and `llms.txt` if workflows or
      agent guidance materially change.

Exit criteria:

- The daily agent workflow works as a coherent product surface.
- Real-world repo-shape warnings are useful and specific.
- Test and entrypoint evidence improves next-action planning.
- Human output and JSON output remain stable enough for daily use.

## Out Of Scope Until This Plan Is Done

- MCP or long-running server mode
- hosted service behavior
- TUI
- editor integration hooks
- graph export
- GitHub Action integration
- project-specific configuration as a requirement for basic use
- full SSA, whole-program pointer analysis, reflection analysis, or broad
  framework inference
- automatic refactoring or code modification

These can be valuable later, but they should follow the daily agent workflow,
real-world repository robustness, and stronger test/entrypoint intelligence.

## Suggested PR Slice Order

1. Baseline audit and command design lock.
2. Agent workflow data model.
3. Public agent workflow command.
4. Agent workflow size and trust controls.
5. Snapshot ergonomics for daily use.
6. Workspace and module boundary audit.
7. Build-tag and package-loading diagnostics.
8. Generated-file policy.
9. Large-repository performance guardrails.
10. Robustness documentation and fixtures pass.
11. Test inventory model.
12. Stronger affected-test planning.
13. Entrypoint inventory model.
14. Entrypoints in context, impact, PR, and agent workflow.
15. Test and entrypoint documentation pass.
16. Final integration pass.

Each PR should leave `go test ./...` passing unless it explicitly documents a
pre-existing failure before edits begin.
