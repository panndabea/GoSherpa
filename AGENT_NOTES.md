# GoSherpa for AI Agents

This file is an opt-in guide for humans and coding agents that want to use
GoSherpa as repository-grounded context before editing Go code.

It is intentionally named `AGENT_NOTES.md`, not `AGENTS.md`, so tools that
automatically load `AGENTS.md` do not treat this guide as mandatory project
instructions. Read it when GoSherpa itself is useful for the task.

## What GoSherpa Is

GoSherpa is a CLI tool for analyzing Go repositories and returning focused,
deterministic code-structure context. It helps answer questions about symbols,
references, callers, callees, packages, tests, interfaces, entrypoints, call
paths, and diff impact without relying only on broad text search.

Use GoSherpa output as grounded context that guides source reading. It should
not replace reading the relevant files before editing.

## When To Use It

Use GoSherpa when:

- The repository is written in Go.
- You need symbol, reference, caller/callee, dependency, test, interface,
  entrypoint, path, or diff-impact context.
- You want deterministic repository answers before changing code.
- You need JSON output for automation or agent workflows.
- You want a bounded reading order for a symbol, file, package, or diff.

Do not use GoSherpa when:

- The project is not a Go repository.
- You need runtime profiling, security scanning, linting, formatting, or
  automatic refactoring.
- The repository cannot run local Go tooling.
- You need production-grade semantic certainty beyond GoSherpa's conservative
  MVP analysis.

## Agent Workflow

For review, repair, or edit tasks in a Go repository, start with readiness and
bounded diff context:

```bash
gosherpa doctor --json
gosherpa context diff --base HEAD --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
gosherpa impact diff --base HEAD --json
gosherpa tests affected --base HEAD --json
```

For a PR-style summary, use:

```bash
gosherpa pr --base HEAD --json
```

When the target is unclear, start with a limited search:

```bash
gosherpa search parse file --limit 10 --json
gosherpa symbol ./internal/sherpa.ParseFile --json
```

For focused symbol work, use package-qualified targets when possible and cap
context output:

```bash
gosherpa context symbol ./internal/sherpa.ParseFile --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa explain ./internal/sherpa.ParseFile --json
gosherpa refs ./internal/sherpa.ParseFile --json
gosherpa callers ./internal/sherpa.ParseFile --json
gosherpa callees ./internal/sherpa.ParseFile --json
gosherpa tests ./internal/sherpa.ParseFile --json
```

If an unqualified target is ambiguous, use the candidates in GoSherpa's
diagnostic and retry with a package-qualified target, for example
`./internal/sherpa.ParseFile`.

Keep agent output bounded. Prefer `context ... --max-*` and `search --limit <n>`
before broad inventory commands like unfiltered `symbols`.

## Common Tasks

| Task | Start with |
| --- | --- |
| Check analysis readiness | `gosherpa doctor --json` |
| Get a repository overview | `gosherpa analyze . --json` |
| Inspect architecture and coupling | `gosherpa architecture --json` |
| Inspect structural risk | `gosherpa risk --json` |
| Find symbols by name or kind | `gosherpa search <terms> --limit <n> --json`; use `gosherpa symbols --json` only for broad inventory |
| Inspect one symbol | `gosherpa symbol <target> --json` |
| Get bounded pre-edit context | `gosherpa context symbol|file|package|diff ... --max-* ... --json` |
| Find references | `gosherpa refs <target> --json` |
| Find direct callers or callees | `gosherpa callers <target> --json` and `gosherpa callees <target> --json` |
| Explore call reachability | `gosherpa entrypoints <target> --json`, `gosherpa path <from> <to> --json`, or `gosherpa paths <from> <to> --json` |
| Inspect package relationships | `gosherpa packages --json`, `gosherpa deps <package> --json`, or `gosherpa deps --all --json` |
| Inspect interface relationships | `gosherpa implementers <interface> --json` or `gosherpa interfaces <type> --json` |
| Analyze changed files | `gosherpa context diff --base HEAD --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json` and `gosherpa impact diff --base HEAD --json` |
| Plan tests for a change | `gosherpa tests affected --base HEAD --json` |

## Context Commands

Use `gosherpa context` when an agent needs a compact reading bundle before
editing. Add size controls whenever the repository or target may be large:

```bash
gosherpa context symbol ParseFile --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa context file internal/sherpa/impact.go --max-symbols 20 --max-tests 10 --max-bytes 12000 --source-radius 1 --json
gosherpa context package ./internal/sherpa --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
gosherpa context diff --base HEAD --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
```

Context output can include source excerpts, symbols, references, callers,
callees, affected packages, affected tests, a test plan, reading order,
confidence, analysis modes, and limitations.

Use size controls for large repositories or limited context windows:

```bash
gosherpa context symbol ParseFile --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa context file internal/sherpa/impact.go --max-symbols 20 --source-radius 1 --json
gosherpa context diff --base HEAD --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
```

## JSON Output

Agents should prefer `--json` when available. Successful JSON output uses a
shared envelope:

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

Read these fields before acting:

- `warnings`: successful-output warnings from the shared envelope.
- `data.confidence`: a compact trust signal for the result.
- `data.analysisMode`, `data.referenceAnalysisMode`, `data.callAnalysisMode`,
  and `data.interfaceAnalysisMode`: how the result was assembled.
- `data.limitations`: known blind spots for this result.
- `data.truncated`: entries omitted because of explicit context limits or byte
  budgets.

When GoSherpa reports an ambiguous target in JSON mode, stdout stays empty and
stderr includes a structured diagnostic with candidate packages, files, lines,
and package-qualified examples.

## How To Use Results Safely

- Treat GoSherpa output as repository-grounded context, not as a proof of full
  runtime behavior.
- Inspect the source files named in `readingOrder`, references, callers,
  callees, tests, and impact output before editing.
- Limit broad outputs with `--max-*` or `--limit` flags before feeding them into
  an agent context window.
- Prefer package-qualified targets for symbols that may appear in multiple
  packages.
- Use reported confidence, warnings, analysis modes, and limitations in your
  reasoning.
- Do not claim complete call graph, runtime, security, or semantic certainty
  unless the output explicitly supports it.

## Known MVP Limitations

GoSherpa is an early MVP. APIs, commands, and JSON schemas may change.

Current analysis is intentionally conservative:

- It focuses on Go repositories that can run local Go tooling.
- Reference search uses `go/packages` typechecked loading when available and
  falls back to AST/per-package analysis when needed.
- Caller, callee, path, entrypoint, and interface impact analysis may miss
  dynamic dispatch, reflection, function values, goroutine starts, function
  literals, build-tag edge cases, aliases, and some generic cases.
- Test files are excluded from several analyses by default; use command flags
  such as `--tests` where available.
- Diff impact is hunk-based and does not infer every semantic consequence of
  changed statements.
- Test discovery is useful but incomplete for dynamic table-driven test names
  and framework-specific behavior.
- Snapshot creation and freshness diagnostics exist, but query commands still
  analyze repository data directly instead of reusing snapshots.
- GoSherpa is not a runtime profiler, security scanner, linter, formatter, or
  automatic refactoring engine.

## Links

- [CLI Reference](docs/CLI_REFERENCE.md)
- [Implementation Status](docs/STATUS.md)
- [Agent JSON Schema](docs/product/JSON_SCHEMA_V1.md)
- [Context JSON Schema](docs/product/CONTEXT_SCHEMA_V1.md)
- [Feature Roadmap](docs/product/FEATURE_ROADMAP.md)
- [Repository](https://github.com/panndabea/GoSherpa)
