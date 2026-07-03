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

Source ranges are root-relative and use the same position shape:

```json
{
  "start": {
    "file": "service.go",
    "line": 12,
    "column": 3
  },
  "end": {
    "file": "service.go",
    "line": 12,
    "column": 9
  }
}
```

`range` is optional. It is emitted for symbols, references, callers, callees,
call path steps, and related tests when Go parser positions identify the exact
source span. `position` remains the primary compact location field.

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
    "line": 8,
    "column": 2
  },
  "range": {
    "start": {
      "file": "service.go",
      "line": 8,
      "column": 2
    },
    "end": {
      "file": "service.go",
      "line": 8,
      "column": 8
    }
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
    "line": 4,
    "column": 2
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
    "line": 5,
    "column": 1
  },
  "directReference": true,
  "externalPackage": false
}
```

Entrypoints:

```json
{
  "name": "Entry",
  "package": "./internal/service",
  "kind": "exported",
  "position": {
    "file": "internal/service/service.go",
    "line": 12,
    "column": 1
  },
  "range": {
    "start": {
      "file": "internal/service/service.go",
      "line": 12,
      "column": 1
    },
    "end": {
      "file": "internal/service/service.go",
      "line": 12,
      "column": 11
    }
  }
}
```

Entrypoint `kind` is one of `main`, `test`, `exported`, or
`no-local-callers`.

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
  "interfaceAnalysisMode": "typechecked",
  "confidence": "medium",
  "limitations": []
}
```

- `analysisMode`: deterministic label describing how the command assembled its
  result.
- `interfaceAnalysisMode`: optional label describing the interface-impact
  subanalysis when a bundle includes affected interface or implementation
  fields. It uses `typechecked` or `ast-fallback`.
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

Reference, standalone interface analysis, and interface subanalysis in context
or report-based impact bundles use the same `typechecked` and `ast-fallback`
labels. Broader explain, context, impact, test, and path commands currently use:

- `ast`: syntax plus local type information and repository-local heuristics.
- `typechecked+ast`: typechecked package loading for selected context fields,
  combined with syntax/local impact and test signals.
- `git-diff+ast`: git diff discovery plus AST/local repository analysis.

`callers.data.analysisMode` and `callees.data.analysisMode` use these values.
`explain.data.callAnalysisMode` and `context symbol.data.callAnalysisMode` use
the same values.

`context symbol.data.analysisMode` is different: it describes the broader
context bundle mode. It can be `typechecked+ast` when symbol identity comes from
typechecked package loading, or `ast` when that semantic path is unavailable or
does not include the target symbol. `context file.data.analysisMode` and
`context package.data.analysisMode` can be `typechecked+ast` when package files
and symbols come from typechecked package loading. `context diff.data.analysisMode`
is currently `git-diff+ast`.

`doctor.data.analysisMode` reports repository readiness rather than a code
relationship graph. It is `typechecked` when package loading completed and
`unavailable` when package loading failed.

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

## `entrypoints` Data

Envelope:

- `command`: `entrypoints`
- `target`: resolved call target

Data:

```json
{
  "analysisMode": "typechecked",
  "confidence": "medium",
  "limitations": [],
  "entrypoints": []
}
```

- `analysisMode`: call analysis trust mode, either `typechecked` or
  `ast-fallback`.
- `confidence`: deterministic trust label.
- `limitations`: call-graph and entrypoint-classification blind spots.
- `entrypoints`: array of entrypoint entries. The array is present even when
  empty.

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
  "analysisMode": "typechecked+ast",
  "confidence": "medium",
  "limitations": [],
  "symbol": {},
  "symbolAnalysisMode": "typechecked+ast",
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
  "interfaceAnalysisMode": "typechecked",
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
- `analysisMode`: broader explain-bundle mode, `typechecked+ast` when one or
  more composed subanalyses use typechecked package loading and `ast` otherwise.
- `confidence`: deterministic trust label.
- `limitations`: explain, call, and test-planning blind spots.
- `symbol`: symbol profile object.
- `symbolAnalysisMode`: symbol identity mode, currently `typechecked+ast`
  when the target was resolved from typechecked package loading and `ast` when
  the AST repository scan fallback was used.
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
- `interfaceAnalysisMode`: trust mode for `affectedInterfaces` and
  `affectedImplementations`, either `typechecked` or `ast-fallback`.
- `relatedTests`: tests related to the symbol.
- `testCommands`: suggested `go test` commands.
- `testPlan`: grouped test plan.
- `readingOrder`: ordered source locations to inspect next.

`data.warnings` is absent; use envelope `warnings`.

## `doctor` Data

Envelope:

- `command`: `doctor`
- `target`: `.`

Data:

```json
{
  "target": ".",
  "environment": {
    "goVersion": "go1.24.4",
    "goos": "darwin",
    "goarch": "arm64"
  },
  "repository": {
    "root": "/repo",
    "modulePath": "example.com/app",
    "goModPath": "go.mod",
    "goWork": {
      "detected": false
    },
    "goFiles": 12,
    "testFiles": 4,
    "generatedFiles": 0,
    "nestedModules": []
  },
  "buildTags": [],
  "packageLoad": {
    "status": "ok",
    "analysisMode": "typechecked",
    "packageCount": 3,
    "packages": [],
    "warningCount": 0
  },
  "snapshot": {
    "supported": false,
    "status": "not_implemented",
    "message": "Persistent snapshots are not implemented yet; commands analyze the repository on demand."
  },
  "analysisMode": "typechecked",
  "confidence": "medium",
  "limitations": [],
  "suggestions": []
}
```

- `environment`: Go runtime and platform used by the GoSherpa binary.
- `repository`: resolved module root, module path, file counts, `go.work`
  detection, and nested-module hints.
- `buildTags`: normalized build tags supplied through `--tags`.
- `packageLoad`: status of typechecked package loading. `status` is `ok`,
  `warnings`, or `failed`.
- `snapshot`: current snapshot support and status.
- `analysisMode`: readiness mode, currently `typechecked` or `unavailable`.
- `confidence`: `low` when warnings are emitted, otherwise `medium`.
- `limitations`: boundaries of the readiness check.
- `suggestions`: deterministic next steps.

Package load warnings live on the shared envelope. `data.warnings` is absent;
use envelope `warnings`.

## Impact And Test Data

`impact`, `impact file`, `impact package`, `impact symbol`, `impact diff`,
`tests`, and `tests affected` data objects include the common metadata fields:

- `analysisMode`: `ast`, `typechecked+ast`, or `git-diff+ast`. Direct impact
  queries use `typechecked+ast` when one or more composed subanalyses used
  typechecked package loading; diff-based queries use `git-diff+ast`.
- `confidence`: deterministic trust label.
- `limitations`: command-specific impact or test-planning blind spots.

Impact data keeps its existing arrays such as `references`, `callers`,
`affectedPackages`, `affectedTests`, `testCommands`, and `testPlan`. Test data
keeps `tests` or `affectedTests`, `commands`, and `testPlan`. `tests` also
includes `scope` with one of `direct`, `related`, or `all`; the default
`related` scope focuses direct references when they exist.

Symbol impact data includes `referenceAnalysisMode` and `callAnalysisMode`
when those subanalyses ran. Report-based impact data (`impact file`,
`impact package`, `impact symbol`, and `impact diff`) also includes
`affectedInterfaces`, `affectedImplementations`, and `interfaceAnalysisMode`
when interface subanalysis ran.

## Interface And Path Data

`implementers`, `interfaces`, `path`, and `paths` data objects include the
common metadata fields:

- `analysisMode`: `typechecked` or `ast-fallback` for `implementers` and
  `interfaces`; currently `ast` for path commands.
- `confidence`: deterministic trust label.
- `limitations`: command-specific interface or path-analysis blind spots.

Interface data keeps `implementers` or `interfaces`. Path data keeps `from`,
`to`, and `paths`.

## Package Inventory Data

Envelope:

- `command`: `packages`
- `target`: empty string

Data:

```json
{
  "packages": [
    {
      "package": "./internal/sherpa",
      "packageName": "sherpa",
      "goFiles": 24,
      "testFiles": 18,
      "symbols": 120,
      "imports": 8,
      "localImports": 1,
      "externalImports": 7,
      "importedBy": 3,
      "hasTests": true
    }
  ]
}
```

- `packages`: local package summaries sorted by package path.
- `goFiles`: non-test `.go` file count.
- `testFiles`: `_test.go` file count.
- `symbols`: discovered symbol count. Test symbols are included only when
  `--tests` is used.
- `imports`: unique import count. Test-file imports are included only when
  `--tests` is used.
- `localImports`: unique imports that resolve to another package in the same
  module.
- `externalImports`: unique imports outside the local module.
- `importedBy`: number of local packages importing this package.
- `hasTests`: true when the package directory contains at least one `_test.go`
  file.

## `context` Call Metadata

`context` data is documented in `docs/product/CONTEXT_SCHEMA_V1.md`.
`context symbol` additionally includes explicit call analysis trust metadata:

- `analysisMode`: broader context analysis mode, `typechecked+ast` when symbol
  identity comes from typechecked package loading and `ast` otherwise.
- `confidence`: deterministic trust label.
- `limitations`: context, call, and test-planning blind spots.
- `callAnalysisMode`: call graph trust mode, either `typechecked` or
  `ast-fallback`.
- `interfaceAnalysisMode`: trust mode for interface and implementation impact
  fields when present, either `typechecked` or `ast-fallback`.
- `callers`: repository-local callers.
- `callees`: repository-local callees.

Agents should not treat `analysisMode` and `callAnalysisMode` as interchangeable.
`analysisMode` describes how the context bundle was assembled; `callAnalysisMode`
describes the trust level of the call graph fields.
The same distinction applies to `referenceAnalysisMode` and
`interfaceAnalysisMode`: each describes only its corresponding subanalysis
inside a broader bundle.

`data.warnings` is absent; use envelope `warnings`.
