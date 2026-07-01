# GoSherpa Agent JSON Schema v1

This document describes the stable JSON contracts for agent-facing `gosherpa
--json` output, with special attention to call analysis and explain output.
Command-specific context bundle fields are also described in
`docs/product/CONTEXT_SCHEMA_V1.md`.

Agents should tolerate additional fields in v1 objects. Documented fields should
remain stable until `schemaVersion` changes.

## Shared Envelope

Successful JSON output uses this envelope:

```json
{
  "schemaVersion": 1,
  "command": "callers",
  "target": "Target",
  "root": "/repo",
  "modulePath": "example.com/app",
  "warnings": [],
  "data": {}
}
```

- `schemaVersion`: integer schema version. Current value is `1`.
- `command`: canonical command name, such as `callers`, `callees`, `explain`,
  or `context symbol`.
- `target`: normalized command target. Call path commands use
  `<from> -> <to>`.
- `root`: resolved repository root used for analysis.
- `modulePath`: Go module path read from `go.mod`.
- `warnings`: array of warning strings. The field is always present; it is an
  empty array when there are no warnings.
- `data`: command-specific result object.

Warnings live on the shared envelope for successful command output. For the
contracts documented here, `data.warnings` is absent. In particular, call
analysis fallback warnings from `callers`, `callees`, `explain`, and
`context symbol` are emitted in envelope `warnings`.

## Common Objects

Positions are root-relative and 1-based:

```json
{
  "file": "service.go",
  "line": 12,
  "column": 3
}
```

`column` may be omitted when it is not known.

Symbols use this profile shape. Some fields are omitted when empty:

- `name`
- `kind`: `struct`, `interface`, `function`, or `method`
- `package`
- `packageName`
- `qualifiedName`
- `signature`
- `documentation`
- `position`
- `range`: `{ "start": position, "end": position }`
- `receiver`
- `receiverType`
- `fields`: array of `{ "name", "type", "tag", "embedded" }`
- `methods`: array of `{ "name", "signature", "embedded" }`

References:

```json
{
  "name": "Target",
  "kind": "call",
  "position": {
    "file": "service.go",
    "line": 8
  }
}
```

Reference `kind` values include `definition`, `call`, `type_usage`,
`field_access`, and `usage`.

Callers and callees share the same entry shape:

```json
{
  "name": "Run",
  "position": {
    "file": "service.go",
    "line": 4
  }
}
```

Related tests:

```json
{
  "name": "TestTarget",
  "package": ".",
  "packageName": "service",
  "position": {
    "file": "service_test.go",
    "line": 5
  },
  "directReference": true,
  "externalPackage": false
}
```

Test plans:

```json
{
  "direct": [],
  "related": [],
  "callerPackages": [],
  "fallback": []
}
```

Each test plan item has `command` and `reason`, with optional `package` and
`test`.

Risk summaries use `{ "level": string, "reasons": [] }`. Architecture roles use
`{ "role": string, "reasons": [] }`. Reading-order entries use
`{ "title": string, "reason": string, "position": position }`.

## Common Analysis Metadata

Agent-facing analysis data objects include:

```json
{
  "analysisMode": "ast",
  "confidence": "medium",
  "limitations": []
}
```

- `analysisMode`: deterministic label describing how the command assembled its
  result.
- `confidence`: deterministic trust label, currently `medium` or `low`.
- `limitations`: known blind spots and conservative boundaries for the command.

`confidence` is `low` when warnings were emitted or a typechecked path fell
back to AST analysis. Otherwise, current agent-facing commands report
`medium`. `limitations` is always present on the documented agent-facing data
objects below.

## Analysis Modes

Call analysis mode values:

- `typechecked`: call graph was resolved with typechecked package loading where
  available.
- `ast-fallback`: GoSherpa fell back to AST-only call analysis because
  typechecked loading was unavailable.

Reference and standalone interface analysis use the same `typechecked` and
`ast-fallback` labels. Broader context, impact, test, and path commands
currently use:

- `ast`: syntax plus local type information and repository-local heuristics.
- `git-diff+ast`: git diff discovery plus AST/local repository analysis.

`callers.data.analysisMode` and `callees.data.analysisMode` use these values.
`explain.data.callAnalysisMode` and `context symbol.data.callAnalysisMode` use
the same values.

`context symbol.data.analysisMode` is different: it describes the broader
context bundle mode, currently `ast`. `context diff.data.analysisMode` is
currently `git-diff+ast`.

## `callers` Data

Envelope:

- `command`: `callers`
- `target`: resolved call target

Data:

```json
{
  "analysisMode": "typechecked",
  "confidence": "medium",
  "limitations": [],
  "callers": []
}
```

- `analysisMode`: call analysis trust mode, either `typechecked` or
  `ast-fallback`.
- `confidence`: deterministic trust label.
- `limitations`: call-graph blind spots and scope boundaries.
- `callers`: array of caller entries. The array is present even when empty.

`data.warnings` is absent; use envelope `warnings`.

## `callees` Data

Envelope:

- `command`: `callees`
- `target`: resolved call target

Data:

```json
{
  "analysisMode": "typechecked",
  "confidence": "medium",
  "limitations": [],
  "callees": []
}
```

- `analysisMode`: call analysis trust mode, either `typechecked` or
  `ast-fallback`.
- `confidence`: deterministic trust label.
- `limitations`: call-graph blind spots and scope boundaries.
- `callees`: array of callee entries. The array is present even when empty.

`data.warnings` is absent; use envelope `warnings`.

## `refs` Data

Envelope:

- `command`: `refs`
- `target`: resolved reference target

Data:

```json
{
  "analysisMode": "typechecked",
  "confidence": "medium",
  "limitations": [],
  "references": []
}
```

- `analysisMode`: reference analysis trust mode, either `typechecked` or
  `ast-fallback`.
- `confidence`: deterministic trust label.
- `limitations`: reference-analysis blind spots and scope boundaries.
- `references`: array of reference entries. The array is present even when
  empty.

`data.warnings` is absent; use envelope `warnings`.

## `explain` Data

Envelope:

- `command`: `explain`
- `target`: resolved symbol target

Data:

```json
{
  "target": "Target",
  "analysisMode": "ast",
  "confidence": "medium",
  "limitations": [],
  "symbol": {},
  "purpose": "",
  "risk": {
    "level": "medium",
    "reasons": []
  },
  "architectureRole": {
    "role": "leaf_dependency",
    "reasons": []
  },
  "references": [],
  "callers": [],
  "callees": [],
  "callAnalysisMode": "typechecked",
  "affectedPackages": [],
  "affectedInterfaces": [],
  "affectedImplementations": [],
  "relatedTests": [],
  "testCommands": [],
  "testPlan": {
    "direct": [],
    "related": [],
    "callerPackages": [],
    "fallback": []
  },
  "readingOrder": []
}
```

- `target`: normalized symbol target.
- `analysisMode`: broader explain-bundle mode, currently `ast`.
- `confidence`: deterministic trust label.
- `limitations`: explain, call, and test-planning blind spots.
- `symbol`: symbol profile object.
- `purpose`: extracted symbol purpose, or an empty string when none is found.
- `risk`: deterministic risk summary.
- `architectureRole`: deterministic architectural role summary.
- `references`: repository-local references.
- `callers`: repository-local callers.
- `callees`: repository-local callees.
- `callAnalysisMode`: call analysis trust mode, either `typechecked` or
  `ast-fallback`.
- `affectedPackages`: package paths affected by changes to the symbol.
- `affectedInterfaces`: interface contracts affected by changes to the symbol.
- `affectedImplementations`: implementation symbols affected by changes to the
  symbol.
- `relatedTests`: tests related to the symbol.
- `testCommands`: suggested `go test` commands.
- `testPlan`: grouped test plan.
- `readingOrder`: ordered source locations to inspect next.

`data.warnings` is absent; use envelope `warnings`.

## Impact And Test Data

`impact`, `impact file`, `impact package`, `impact symbol`, `impact diff`,
`tests`, and `tests affected` data objects include the common metadata fields:

- `analysisMode`: `ast` for direct impact/test queries, `git-diff+ast` for
  diff-based queries.
- `confidence`: deterministic trust label.
- `limitations`: command-specific impact or test-planning blind spots.

Impact data keeps its existing arrays such as `references`, `callers`,
`affectedPackages`, `affectedTests`, `testCommands`, and `testPlan`. Test data
keeps `tests` or `affectedTests`, `commands`, and `testPlan`.

## Interface And Path Data

`implementers`, `interfaces`, `path`, and `paths` data objects include the
common metadata fields:

- `analysisMode`: `typechecked` or `ast-fallback` for `implementers` and
  `interfaces`; currently `ast` for path commands.
- `confidence`: deterministic trust label.
- `limitations`: command-specific interface or path-analysis blind spots.

Interface data keeps `implementers` or `interfaces`. Path data keeps `from`,
`to`, and `paths`.

## `context symbol` Call Metadata

`context symbol` data is the symbol context bundle documented in
`docs/product/CONTEXT_SCHEMA_V1.md`, plus explicit call analysis trust metadata:

- `analysisMode`: broader context analysis mode, currently `ast`.
- `confidence`: deterministic trust label.
- `limitations`: context, call, and test-planning blind spots.
- `callAnalysisMode`: call graph trust mode, either `typechecked` or
  `ast-fallback`.
- `callers`: repository-local callers.
- `callees`: repository-local callees.

Agents should not treat `analysisMode` and `callAnalysisMode` as interchangeable.
`analysisMode` describes how the context bundle was assembled; `callAnalysisMode`
describes the trust level of the call graph fields.

`data.warnings` is absent; use envelope `warnings`.
