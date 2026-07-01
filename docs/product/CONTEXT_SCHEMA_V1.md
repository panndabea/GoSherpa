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
- `confidence`: deterministic confidence label.
- `limitations`: known blind spots and conservative boundaries.

When size controls are used, context results may also include:

- `limits`: active user-provided limits.
- `truncated`: counts of omitted entries by field.

Example:

```json
{
  "limits": {
    "maxReferences": 20,
    "maxTests": 10,
    "sourceRadius": 1
  },
  "truncated": {
    "references": 7,
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
- `--source-radius <n>` limits source excerpt radius for symbol, file, and
  package context. `0` means target line only.

`context diff` supports `--max-files`, `--max-symbols`, and `--max-tests`.

Purpose and risk are computed before output truncation. The `truncated` object
explains what was omitted from the bounded context bundle.

## Context Kinds

`context symbol` adds symbol identity, definition, source context, references,
callers, callees, affected packages, interface signals, related tests, and test
planning.

`context file` adds file identity, package identity, file symbols, source
contexts, package-level impact, affected tests, and test planning.

`context package` adds package identity, package files, package symbols, source
contexts, package-level impact, affected tests, and test planning.

`context diff` adds base ref, changed files, changed packages, affected
symbols, affected packages, interface signals, affected tests, and test
planning.
