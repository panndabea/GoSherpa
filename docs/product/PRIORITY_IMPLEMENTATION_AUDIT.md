# Priority Implementation Audit

Date: 2026-07-16

This audit records Phase 0 of `PRIORITY_IMPLEMENTATION_PLAN.md`. It is the
baseline for the semantic-accuracy, context-export, and test-planning slices
that follow.

## Selected Base Ref

Selected `<base-ref>`: `origin/main`

Reason: local refs include `main`, `origin/HEAD`, and `origin/main`; the remote
tracking branch is available and is the least ambiguous diff baseline for this
repository.

## Baseline Verification

Commands run:

```bash
go test ./...
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa context diff --base origin/main --json
```

Results:

- `go test ./...` passed for all packages.
- `doctor --json` emitted no envelope warnings.
- `doctor --json` reported package loading as `ok`, `analysisMode` as
  `typechecked`, and confidence as `medium`.
- `doctor --json` reported the repository snapshot as `stale` with stale
  reasons `git state changed` and `repository files changed`.
- `context diff --base origin/main --json` emitted no envelope warnings and
  reported no changed files relative to `origin/main`.

## Command Contract Snapshot

Current command and flag support is driven by:

- `cmd/gosherpa/command_registry.go`
- `cmd/gosherpa/cli_flags.go`
- `cmd/gosherpa/main.go`

Confirmed contracts:

- `--tags` is accepted for `analyze`, `refs`, `entrypoints`, `callers`,
  `callees`, `explain`, `context`, `impact`, `tests affected`,
  `implementers`, `interface`, `interfaces`, `pr`, `doctor`, and `snapshot`.
- Plain `tests <target>` does not accept `--tags`.
- `--base` is accepted for `context diff`, `impact diff`, `tests affected`,
  and `pr`.
- `--use-snapshot` is accepted for `analyze`, `symbols`, `symbol`, `search`,
  `packages`, `context diff`, `impact diff`, `tests affected`, and `pr`.
- `--scope` is accepted for plain `tests <symbol-or-package-or-file>` and is
  intentionally not accepted for `tests affected`.
- Context limits are command-specific:
  - `context symbol`: `--max-references`, `--max-tests`, `--max-bytes`,
    `--source-radius`
  - `context file`: `--max-symbols`, `--max-tests`, `--max-bytes`,
    `--source-radius`
  - `context package`: `--max-files`, `--max-symbols`, `--max-tests`,
    `--max-bytes`, `--source-radius`
  - `context diff`: `--max-files`, `--max-symbols`, `--max-tests`,
    `--max-bytes`

Contract notes after Phase 0:

- Phase 2.2 added focused CLI tests for unsupported context-limit
  combinations, including `context diff --source-radius`,
  `context diff --max-references`, `context file --max-references`,
  `context package --max-references`, `context symbol --max-files`, and
  `context symbol --max-symbols`.
- Add explicit unsupported-combination tests for `tests <target> --tags` and
  `tests affected --scope`.
- The global usage text lists `--tags` as a global option, while validation
  intentionally limits it by command. This is acceptable today but should be
  clarified in CLI docs during the Phase 1 build-tag slice.

## Golden JSON And Schema Status

Stable golden JSON fixtures exist for these top agent-facing commands:

- `context symbol`
- `context file`
- `context package`
- `context diff`
- `impact symbol`
- `impact diff`
- `tests`
- `tests affected`
- `pr`

Schema and golden guardrails exist in:

- `cmd/gosherpa/json_schema_test.go`
- `cmd/gosherpa/json_golden_test.go`
- `cmd/gosherpa/testdata/golden-json/`
- `docs/product/JSON_SCHEMA_V1.md`
- `docs/product/CONTEXT_SCHEMA_V1.md`

Observed schema/golden drift risks:

- `doctor.json` exists as a golden fixture file but is not part of
  `json_golden_test.go`; it is covered by schema-style tests in
  `main_test.go` and `json_schema_test.go`.
- `context diff`, `impact diff`, `tests affected`, and `pr` have focused
  golden tests with synthetic git diffs, separate from the shared golden loop.
- Product docs still contain examples with concrete `main` or `HEAD` refs in
  several places. Future docs updates should prefer `<base-ref>` unless a
  slice intentionally uses a real verified ref such as `origin/main`.

## Trust Fields By Command

Envelope-level `warnings` are present in the shared JSON response for all JSON
commands.

Top agent-facing command data fields:

| Command | Trust fields present |
| --- | --- |
| `context symbol` | `analysisMode`, `confidence`, `limitations`, `referenceAnalysisMode`, `callAnalysisMode`, `interfaceAnalysisMode`, `testAnalysisMode` |
| `context file` | `analysisMode`, `confidence`, `limitations`, `interfaceAnalysisMode`, `testAnalysisMode` |
| `context package` | `analysisMode`, `confidence`, `limitations`, `interfaceAnalysisMode`, `testAnalysisMode` |
| `context diff` | `analysisMode`, `confidence`, `limitations`, `referenceAnalysisMode`, `callAnalysisMode`, `interfaceAnalysisMode`, `testAnalysisMode` |
| `impact symbol` | `analysisMode`, `confidence`, `limitations`, `referenceAnalysisMode`, `callAnalysisMode`, `interfaceAnalysisMode`, `testAnalysisMode` |
| `impact diff` | `analysisMode`, `confidence`, `limitations`, `referenceAnalysisMode`, `callAnalysisMode`, `interfaceAnalysisMode`, `testAnalysisMode` |
| `tests` | `analysisMode`, `confidence`, `limitations` |
| `tests affected` | `analysisMode`, `confidence`, `limitations`, `referenceAnalysisMode`, `callAnalysisMode`, `interfaceAnalysisMode`, `testAnalysisMode` |
| `pr` | `analysisMode`, `confidence`, `limitations`, `referenceAnalysisMode`, `callAnalysisMode`, `interfaceAnalysisMode`, `testAnalysisMode` |

Warnings and limitations placement:

- JSON envelope warnings are consistently outside `data`.
- Data-level `limitations` are consistently present on the listed
  agent-facing commands.
- Existing tests assert that `data.warnings` is absent for several key command
  shapes.
- Phase 2.3 added context limitations for generated-file policy and package
  load warnings, kept confidence values to `medium` and `low`, and covered
  stale-snapshot human warnings without stderr noise.
- Phase 2.4 confirmed context schema coverage for trust fields, kept schema
  version `1`, covered ambiguous target JSON candidates for `context symbol`,
  and added an empty-but-valid `context diff` JSON contract.

Follow-up needed:

- Phase 1 should verify package-load warnings are deduplicated and propagated
  through all semantic subanalyses.
- Phase 2 should keep schema and golden fixtures aligned with any future
  context trust-field changes.

## Fixture Coverage Snapshot

Already covered by existing unit or CLI tests:

- Direct `go.mod` roots.
- `go.work` detection and workspace package loading in semantic, workspace,
  context, diff, and CLI tests.
- Nested module detection in `doctor`.
- Build-tag parsing, normalization, and selected command behavior.
- Type aliases and generic alias signatures.
- Generic receiver methods and methods on generic types.
- Embedded local interfaces in interface impact.
- Imported package selectors for reference/call analysis.
- Concrete method values and simple function-value calls in call analysis.
- Reassigned or escaping function values as limited/unresolved cases.
- Duplicate unqualified symbols returning ambiguity candidates.

Fixture gaps to assign:

- Phase 1.3: add a dedicated nested-module behavior fixture that verifies
  symbol lookup, references, callers, context export, impact, and tests do not
  fold nested modules into the wrong package scope.
- Phase 1.3: document intentionally unsupported cross-module behavior for
  workspace and nested-module cases.
- Phase 1.4: expand build-tag fixtures across every command that claims
  `--tags` support, not only representative semantic paths.
- Phase 1.4: document generated-file policy and decide whether standard
  generated headers should produce limitations outside `doctor`.
- Phase 1.5: keep modern Go semantic edge-case coverage connected to
  agent-facing CLI/golden outputs, not only package-level unit tests.
- Phase 2.2: tiny byte-budget and entry-count limit tests now cover every
  context variant.
- Phase 3.2: add diff fixtures for deleted symbols, hunk-only changes,
  non-Go-only changes, interface contract changes, multiple packages, and no
  directly referenced tests.
- Phase 3.3: add caller-package-only and cross-package interface contract test
  recommendation fixtures.

## Open Questions

- Should `doctor.json` become an active golden fixture, or should `doctor`
  remain covered only by schema-style tests because environment fields vary?
- Should generated-file detection outside `doctor` add limitations to context,
  impact, and test-planning outputs, or should documentation alone describe the
  current repository-native policy?
- Should product docs normalize all example base refs to `<base-ref>` now, or
  should that happen only when each affected slice updates its command docs?
- Should nested modules under `cmd/gosherpa/testdata` be treated as fixtures
  only, or should `doctor`/context documentation call out that they are
  intentionally separate analysis roots?

## Phase Mapping

- Phase 1 starts with semantic loader diagnostics and warning propagation.
- Phase 1.3 owns workspace and nested-module behavioral fixtures.
- Phase 1.4 owns build-tag breadth and generated-file policy.
- Phase 1.5 owns modern Go semantic edge-case fixtures and limitations.
- Phase 2 owns context field contract, limits, confidence, warning placement,
  and schema/golden alignment.
- Phase 3 owns test plan shape, diff-based recommendations, contract and
  caller-package groups, and output polish. Phase 3.1 hardened the shared
  `sherpa.TestPlan` contract, non-nil groups, known test/target fields,
  fallback commands, and CLI differences between `tests <target>` and
  `tests affected`. Phase 3.2 added whole-repository fallback for non-Go-only
  diffs, kept package fallback for hunk-only and deleted-symbol Go changes, and
  verified changed-symbol targets for function and method diffs. Phase 3.3
  verified contract recommendations across interface/implementation packages
  and caller-package recommendations when caller tests are the only narrow
  signal. Phase 3.4 aligned CLI/status/schema docs with grouped human and JSON
  test planning, including package-level and whole-repository fallbacks.
