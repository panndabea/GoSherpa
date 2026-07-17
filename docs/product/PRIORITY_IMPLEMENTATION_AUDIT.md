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

- v1 snapshots may continue to load for inventory reuse.
- Relationship loaders must require explicit v2 relationship capability
  metadata before using persisted relationship data.
- Missing, zero, v1, or malformed relationship metadata must produce a visible
  fallback to live analysis for relationship queries.
- If `normalizeSnapshot` changes during the v2 slice, it must preserve old
  inventory behavior without upgrading old files into relationship-capable
  snapshots.

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
