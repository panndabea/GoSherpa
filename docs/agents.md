# GoSherpa for AI Agents

GoSherpa is a CLI tool for analyzing Go codebases and returning repository-grounded context that humans and AI coding agents can inspect before changing code.

It is useful when an agent needs deterministic answers about Go symbols, references, callers, callees, package dependencies, tests, or diff impact without relying only on broad text search.

## Should an agent use GoSherpa?

Use GoSherpa when:

- The repository is written in Go.
- The agent needs symbol, reference, caller/callee, dependency, or diff-impact context.
- Deterministic CLI output is preferred over broad text search.
- JSON output is required for automation.
- The agent needs repository-grounded context before editing code.

Do not use GoSherpa when:

- The project is not written in Go.
- The task is runtime profiling, security scanning, linting, or automatic refactoring.
- The agent needs a mature, versioned production tool with stable releases.
- The repository cannot run local Go tooling.

## What GoSherpa solves

GoSherpa answers structure questions that often slow down code edits in unfamiliar Go repositories:

- Where is this symbol defined?
- What package owns this file or symbol?
- Who references or calls this function?
- What functions does this function call?
- Which packages import or depend on this package?
- Which tests are likely relevant to a symbol, package, file, or diff?
- What changed in a diff, and what might be affected?
- What compact context should an agent read before editing?

Use the output as grounded repository context. It should guide source reading; it should not replace reading the relevant files.

## Agent workflow

For review, repair, or edit tasks in a Go repository, start with readiness and diff context:

```bash
gosherpa doctor
gosherpa impact diff --base HEAD --json
gosherpa context diff --base HEAD --json
gosherpa explain <symbol> --json
```

For a focused symbol or package task, useful follow-up commands include:

```bash
gosherpa symbol <symbol> --json
gosherpa refs <symbol> --json
gosherpa callers <symbol> --json
gosherpa callees <symbol> --json
gosherpa deps <package> --json
gosherpa tests <symbol-or-package> --json
```

Use package-qualified targets when unqualified names are ambiguous, for example `./internal/sherpa.ParseFile`.

## Most useful commands for agents

| Command | Use it for |
| --- | --- |
| `gosherpa doctor` | Confirm the repository, Go environment, build tags, package loading, and confidence before deeper analysis. |
| `gosherpa analyze . --json` | Get a repository overview with packages, symbols, risk, hotspots, tests, readiness, and suggested next commands. |
| `gosherpa context diff --base HEAD --json` | Get bounded pre-edit context for changed files, symbols, packages, tests, reading order, confidence, and limitations. |
| `gosherpa impact diff --base HEAD --json` | Understand changed files, changed packages, changed symbols, affected packages, and affected tests. |
| `gosherpa explain <symbol> --json` | Get a symbol profile with purpose, risk, relationships, reading order, tests, and limitations. |
| `gosherpa refs <symbol> --json` | Find Go-aware definitions and references, optionally filtered by kind. |
| `gosherpa callers <symbol> --json` | Find direct callers of a function or method. |
| `gosherpa callees <symbol> --json` | Find direct calls made by a function or method. |
| `gosherpa deps <package> --json` | Inspect package imports and local dependents. |
| `gosherpa tests affected --base HEAD --json` | Ask for suggested test commands for a diff. |

## Consuming JSON output

Agents should prefer JSON output when available:

```bash
gosherpa context symbol ParseFile --json
gosherpa context file internal/sherpa/impact.go --json
gosherpa context package ./internal/sherpa --json
gosherpa context diff --base HEAD --json
```

JSON output uses a common envelope:

```json
{
  "schemaVersion": 1,
  "command": "context diff",
  "target": "HEAD",
  "root": "/path/to/repo",
  "modulePath": "github.com/example/project",
  "warnings": [],
  "data": {}
}
```

Read `warnings`, `data.confidence`, `data.limitations`, and any truncation metadata before acting. Context commands support size limits such as `--max-files`, `--max-symbols`, `--max-tests`, `--source-radius`, and `--max-bytes`; omitted fields are reported in the JSON result.

When GoSherpa reports an ambiguous target, use the candidates in the diagnostic and retry with a package-qualified symbol. Do not infer relationships that GoSherpa does not report.

## How agents should use the output

- Treat GoSherpa output as repository-grounded context.
- Prefer JSON for automation and human output for quick terminal inspection.
- Inspect the source files named by GoSherpa before editing.
- Use reported confidence, warnings, analysis modes, and limitations in the agent's reasoning.
- Avoid claiming full call graph, runtime, security, or semantic certainty unless the output explicitly supports it.

## Known limitations and maturity

GoSherpa is currently an early MVP. APIs, commands, and output schemas may change.

Current analysis is intentionally conservative:

- It focuses on Go repositories that can run local Go tooling.
- Reference analysis uses typechecked package loading when available and AST/per-package fallback when needed.
- Caller, callee, path, entrypoint, and interface impact analysis may miss dynamic dispatch, reflection, function values, build-tag edge cases, and some generic or alias cases.
- Diff impact is hunk-based and does not infer every semantic consequence of changed statements.
- Test discovery is helpful but incomplete for dynamic table-driven test names and framework-specific behavior.
- GoSherpa is not a runtime profiler, security scanner, linter, formatter, or automatic refactoring engine.

## Links

- [CLI Reference](CLI_REFERENCE.md)
- [Implementation Status](STATUS.md)
- [Agent JSON Schema](product/JSON_SCHEMA_V1.md)
- [Context JSON Schema](product/CONTEXT_SCHEMA_V1.md)
- [Repository](https://github.com/panndabea/GoSherpa)
