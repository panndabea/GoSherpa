# Priority Implementation Audit

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
