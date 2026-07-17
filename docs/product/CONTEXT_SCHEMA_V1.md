# GoSherpa Context JSON Schema v1

`gosherpa context` commands use the shared JSON response envelope:

```json
{
  "schemaVersion": 1,
  "command": "context symbol",
  "target": "Target",
  "root": "/repo",
  "modulePath": "example.com/app",
  "warnings": [],
  "data": {}
}
```

Warnings live on the envelope. Context `data` objects do not include a
`warnings` field.

## Shared Context Fields

Every context result includes:

- `target`: normalized target string.
- `purpose`: concise summary of what the context covers.
- `risk`: deterministic risk level and reasons.
- `targetRisk`: deterministic, target-specific impact breadth summary with
  `level`, `scope`, stable `reasons`, structured `signals`, and
  target-specific `limitations`. This is separate from `risk` and from
  `confidence`; it is not a defect prediction.
- `testCommands`: suggested `go test` commands.
- `testPlan`: grouped test plan with `direct`, `related`, `contracts`,
  `callerPackages`, and `fallback` items. Plan items may include `test`,
  `tests`, and `targets`. `targets` connects a recommendation to the symbol
  target, changed symbols, or interface/implementation contract signals when
  known.
- `readingOrder`: ordered source locations to inspect next. Entries include
  `position` and may include `range` when backed by a symbol, call, or test
  source range.
- `analysisMode`: deterministic analysis mode label.
- `referenceAnalysisMode`: optional trust label for reference subanalysis when
  the context kind includes changed-symbol or symbol-reference impact
  (`typechecked` or `ast-fallback`).
- `callAnalysisMode`: optional trust label for call graph subanalysis when the
  context kind includes caller/callee or changed-symbol impact (`typechecked`
  or `ast-fallback`).
- `interfaceAnalysisMode`: optional trust label for affected interface and
  implementation signals when interface subanalysis ran (`typechecked` or
  `ast-fallback`).
- `testAnalysisMode`: optional trust label for related-test and test-plan
  signals (`typechecked+ast` or `ast`).
- `confidence`: deterministic confidence label.
- `limitations`: known blind spots and conservative boundaries.

Current context analysis modes include `ast`, `typechecked+ast`,
`git-diff+ast`, and `git-diff+typechecked+ast`. Symbol context uses
`typechecked+ast` when symbol identity comes from typechecked package loading.
File and package context use `typechecked+ast` when package files and symbols
come from typechecked package loading while impact and test signals still use
syntax/local repository analysis. Diff context uses
`git-diff+typechecked+ast` when changed-symbol, reference, call, or interface
signals include typechecked package loading, and `git-diff+ast` otherwise.

Current context confidence values are `medium` and `low`. `low` is used when
warnings are present, AST fallback modes materially reduce trust, source
context is missing, a package target cannot be resolved, or snapshot reuse falls
back to live analysis. Envelope `warnings` hold the concrete diagnostics;
`data.limitations` explains stable blind spots such as AST fallback, package
load warnings, generated-file policy, hunk-based diff extraction, dynamic
dispatch, reflection, and skipped test-file callers unless `--tests` is used.

Location-bearing context entries reuse the shared position and range shapes from
`JSON_SCHEMA_V1.md`. References, callers, callees, affected tests, and related
tests keep `position` and may include `range` when Go parser positions identify
the exact source span. Related and affected tests may include `reasons` with
stable relationship labels such as `direct-reference`, `same-package`,
`target-package`, `external-package`, `changed-symbol`, `caller-package`, and
`contract`. Direct related or affected tests may also include
`targetReferences`, where each item names the direct `target`, its `position`,
and an optional exact occurrence `range`; the existing `targets` array remains
the compact planning surface.

When size controls are used, context results may also include:

- `limits`: active user-provided limits.
- `truncated`: counts of omitted entries by field.

Example:

```json
{
  "limits": {
    "maxReferences": 20,
    "maxTests": 10,
    "maxBytes": 12000,
    "sourceRadius": 1
  },
  "truncated": {
    "references": 7,
    "sourceLines": 4,
    "readingOrder": 2
  }
}
```

## Size Controls

Supported context flags are command-specific:

| Context command | Supported limits |
| --- | --- |
| `context symbol` | `--max-references`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context file` | `--max-symbols`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context package` | `--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context diff` | `--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes` |

Unsupported combinations fail as usage errors. `--max-files <n>` limits file
lists, `--max-references <n>` limits symbol references, callers, and callees,
`--max-symbols <n>` limits symbol lists and matching source excerpts, and
`--max-tests <n>` limits related or affected test lists. `--source-radius <n>`
limits source excerpt radius for symbol, file, and package context; `0` means
target line only.

`--max-bytes <n>` applies a best-effort byte budget to the context `data`
payload, not the shared JSON envelope. Large fields are omitted in a
deterministic order while keeping JSON valid and leaving envelope warnings
outside `data`.

`context symbol` and `context diff` also support `--use-snapshot`. For
`context symbol`, a valid relationship snapshot can provide reference, caller,
callee, and selected interface signals while source context, purpose, and test
planning remain live-analysis fields. For `context diff`, a valid snapshot can
provide current changed-symbol inventory and selected relationship subanalysis
signals. Missing, stale, or invalid snapshots fall back to live context analysis
with a warning.

Purpose and risk are computed before output truncation. The `truncated` object
explains what was omitted from the bounded context `data` bundle.
If the requested byte budget is smaller than the minimum `data` report shell,
the `truncated.byteBudgetOverage` field reports the remaining byte overage.

## Context Kinds

The context variants intentionally expose different relationship scopes:

| Context kind | Identity fields | Relationship fields |
| --- | --- | --- |
| `context symbol` | `target`, `identity`, `symbol`, `sourceContext` | `references`, `callers`, `callees`, `affectedPackages`, interface signals, related tests |
| `context file` | `target`, `file`, `package`, `packageName`, `symbols`, `sourceContexts` | `affectedPackages`, interface signals, affected tests |
| `context package` | `target`, `package`, `packageName`, `files`, `symbols`, `sourceContexts` | `affectedPackages`, interface signals, affected tests |
| `context diff` | `target`, `base`, `changedFiles`, `changedPackages`, `affectedSymbols`, `changedSymbolDetails` | `affectedPackages`, changed-symbol subanalysis modes, interface signals, affected tests |

`references`, `callers`, and `callees` are symbol-context fields. File,
package, and diff context omit those arrays until a future contract defines
their scope, limits, and ordering for broader bundles. File and package context
also omit `referenceAnalysisMode` and `callAnalysisMode` because they do not
run reference or call subanalysis directly. Diff context may include those mode
fields when changed-symbol impact runs reference or call subanalysis, but still
does not include full reference, caller, or callee arrays.

`context symbol` adds symbol identity, definition, source context, references,
`referenceAnalysisMode`, callers, callees, `callAnalysisMode`, affected
packages, interface signals, `interfaceAnalysisMode`, related tests,
`testAnalysisMode`, and test planning.

`context file` adds file identity, package identity, file symbols, source
contexts, package-level impact, `interfaceAnalysisMode`, affected tests,
`testAnalysisMode`, and test planning.

`context package` adds package identity, package files, package symbols, source
contexts, package-level impact, `interfaceAnalysisMode`, affected tests,
`testAnalysisMode`, and test planning.

`context diff` adds base ref, changed files, changed packages, affected
symbols, `changedSymbolDetails` with package, target, position, optional range,
and optional deleted status, affected packages, `referenceAnalysisMode`,
`callAnalysisMode`, interface signals, `interfaceAnalysisMode`, affected tests,
`testAnalysisMode`, and test planning. Its `readingOrder` starts with changed
symbol locations when they are known, then changed files, then affected tests.
Related or affected tests and test-plan items may include `targets` naming the
symbol target or changed symbols that explain why the test or package command is
recommended; related or affected tests may include `reasons` explaining direct
symbol, changed-symbol, target-package, caller-package, same-package,
interface/implementation contract, or external-package relationships. Test-plan
items may include structured `tests` when concrete test names are known.
