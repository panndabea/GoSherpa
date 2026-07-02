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
- `testCommands`: suggested `go test` commands.
- `testPlan`: grouped test plan with `direct`, `related`, `callerPackages`,
  and `fallback` items.
- `readingOrder`: ordered source locations to inspect next.
- `analysisMode`: deterministic analysis mode label.
- `interfaceAnalysisMode`: optional trust label for affected interface and
  implementation signals when interface subanalysis ran (`typechecked` or
  `ast-fallback`).
- `confidence`: deterministic confidence label.
- `limitations`: known blind spots and conservative boundaries.

Current context analysis modes include `ast`, `typechecked+ast`, and
`git-diff+ast`. Symbol context uses `typechecked+ast` when symbol identity comes
from typechecked package loading. File and package context use `typechecked+ast`
when package files and symbols come from typechecked package loading while
impact and test signals still use syntax/local repository analysis.

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

Supported context flags:

- `--max-files <n>` limits file lists where a context kind has files.
- `--max-references <n>` limits symbol references, callers, and callees.
- `--max-symbols <n>` limits symbol lists and matching source excerpts.
- `--max-tests <n>` limits related or affected test lists.
- `--max-bytes <n>` applies a best-effort byte budget to the context data
  payload by omitting large fields in a deterministic order while keeping JSON
  valid.
- `--source-radius <n>` limits source excerpt radius for symbol, file, and
  package context. `0` means target line only.

`context diff` supports `--max-files`, `--max-symbols`, `--max-tests`, and
`--max-bytes`.

Purpose and risk are computed before output truncation. The `truncated` object
explains what was omitted from the bounded context bundle.
If the requested byte budget is smaller than the minimum report shell, the
`truncated.byteBudgetOverage` field reports the remaining byte overage.

## Context Kinds

`context symbol` adds symbol identity, definition, source context, references,
callers, callees, `callAnalysisMode`, affected packages, interface signals,
`interfaceAnalysisMode`, related tests, and test planning.

`context file` adds file identity, package identity, file symbols, source
contexts, package-level impact, `interfaceAnalysisMode`, affected tests, and
test planning.

`context package` adds package identity, package files, package symbols, source
contexts, package-level impact, `interfaceAnalysisMode`, affected tests, and
test planning.

`context diff` adds base ref, changed files, changed packages, affected
symbols, affected packages, interface signals, `interfaceAnalysisMode`,
affected tests, and test planning.
