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
the diff-first agent workflow:

```bash
gosherpa doctor --json
gosherpa snapshot --json
gosherpa agent context --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
```

Use focused commands when the agent workflow points at a symbol or package:

```bash
gosherpa context symbol ./internal/sherpa.ParseFile --use-snapshot --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa tests affected --base HEAD --use-snapshot --json
gosherpa pr --base HEAD --use-snapshot --json
```

When the target is unclear, start with a limited search:

```bash
gosherpa search parse file --limit 10 --json
gosherpa symbol ./internal/sherpa.ParseFile --json
```

For focused symbol work, use package-qualified targets when possible and cap
context output:

```bash
gosherpa context symbol ./internal/sherpa.ParseFile --use-snapshot --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa explain ./internal/sherpa.ParseFile --json
gosherpa refs ./internal/sherpa.ParseFile --use-snapshot --json
gosherpa callers ./internal/sherpa.ParseFile --use-snapshot --json
gosherpa callees ./internal/sherpa.ParseFile --use-snapshot --json
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
| Create or refresh reusable snapshot data | `gosherpa snapshot --json` |
| Get a repository overview | `gosherpa analyze . --json` |
| Inspect architecture and coupling | `gosherpa architecture --json` |
| Inspect structural risk | `gosherpa risk --json` |
| Find symbols by name or kind | `gosherpa search <terms> --limit <n> --json`; use `gosherpa symbols --json` only for broad inventory |
| Inspect one symbol | `gosherpa symbol <target> --json` |
| Get daily diff context | `gosherpa agent context --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json` |
| Get focused pre-edit context | `gosherpa context symbol <target> --use-snapshot --max-* --json` when a fresh snapshot exists; otherwise `gosherpa context symbol|file|package|diff ... --max-* ... --json` |
| Find references | `gosherpa refs <target> --json` |
| Find direct callers or callees | `gosherpa callers <target> --json` and `gosherpa callees <target> --json` |
| Explore call reachability | `gosherpa entrypoints <target> --json`, `gosherpa path <from> <to> --json`, or `gosherpa paths <from> <to> --json` |
| Inspect package relationships | `gosherpa packages --json`, `gosherpa deps <package> --json`, or `gosherpa deps --all --json` |
| Inspect interface relationships | `gosherpa interface <interface> --json`; add `--use-snapshot` when a fresh snapshot exists. Use `gosherpa implementers <interface> --json` or `gosherpa interfaces <type> --json` for focused lists |
| Analyze changed files | `gosherpa agent context --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json`; use focused `context diff`, `impact diff`, and `pr` commands for drill-down |
| Plan tests for a symbol, package, or file | `gosherpa tests <target> --json`; for files use `gosherpa tests internal/service/service.go --json` |
| Plan tests for a change | `gosherpa tests affected --base HEAD --json` |

## Context Commands

Use `gosherpa context` when an agent needs a compact reading bundle before
editing a known symbol, file, package, or diff. Use `gosherpa agent context`
first when the task is diff-oriented. Add size controls whenever the repository
or target may be large:

```bash
gosherpa agent context --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
gosherpa context symbol ParseFile --use-snapshot --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa context file internal/sherpa/impact.go --max-symbols 20 --max-tests 10 --max-bytes 12000 --source-radius 1 --json
gosherpa context package ./internal/sherpa --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
gosherpa context diff --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
```

Context output can include source excerpts, symbols, references, callers,
callees, affected packages, affected tests, a test plan, reading order,
confidence, analysis modes, and limitations.

Use size controls for large repositories or limited context windows:

```bash
gosherpa context symbol ParseFile --use-snapshot --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa context file internal/sherpa/impact.go --max-symbols 20 --source-radius 1 --json
gosherpa context diff --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
```

## JSON Output

Agents should prefer `--json` when available. Successful JSON output uses a
shared envelope:

```json
{
  "schemaVersion": 1,
  "command": "agent context",
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
- `data.readiness.repositoryLayout` on `agent context`, and
  `data.repository` on `doctor`: selected module/workspace boundary, skipped
  nested modules, external workspace modules, local replacements, generated
  file counts, and major generated packages.
- `data.readiness.packageLoad.diagnostics` on `agent context`, and
  `data.packageLoad.diagnostics` on `doctor`: package-load diagnostics with
  package, file/position when known, `load-error` vs `type-error`, reason, and
  affected analysis sections.
- `data.targetRisk`: deterministic impact-breadth evidence for the current
  target or diff; inspect its reasons, signals, and limitations before treating
  it as a planning input.
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
- For `agent context --max-bytes`, inspect both `data.truncated` and
  `data.sectionTruncation`; the former is aggregate metadata, while the latter
  identifies the workflow sections shortened by item or byte budgets.
- Prefer package-qualified targets for symbols that may appear in multiple
  packages.
- Use reported confidence, target-risk reasons/signals, warnings, analysis
  modes, and limitations in your reasoning.
- When `doctor` or `agent context` reports skipped nested modules, external
  workspace modules, or local replacements, inspect those roots separately with
  `--root` if the task touches that boundary.
- When package-load diagnostics are present, treat `load-error` as an analysis
  boundary problem and `type-error` as partial typechecked data with lower
  confidence; inspect the affected sections before trusting references, calls,
  interfaces, or test recommendations.
- When generated packages are reported, remember that compiler-visible
  generated files are still analyzed. In `agent context`, large generated file
  reading-order entries may be summarized after hand-written files so the first
  pass stays focused.
- Do not claim complete call graph, runtime, security, or semantic certainty
  unless the output explicitly supports it.

## Known MVP Limitations

GoSherpa is an early MVP. APIs, commands, and JSON schemas may change.

Current analysis is intentionally conservative:

- It focuses on Go repositories that can run local Go tooling.
- Reference search uses `go/packages` typechecked loading when available and
  falls back to AST/per-package analysis when needed.
- Caller, callee, path, entrypoint, and interface impact analysis may miss
  dynamic dispatch, reflection, reassigned or escaping function values,
  goroutine starts, function literals, build-tag edge cases, aliases, and some
  generic cases.
- Test files are excluded from several analyses by default; use command flags
  such as `--tests` where available.
- Diff impact is hunk-based and does not infer every semantic consequence of
  changed statements.
- Test discovery is useful but incomplete for dynamic table-driven test names
  and framework-specific behavior.
- Snapshot creation and freshness diagnostics exist. Inventory commands,
  selected standalone relationship commands (`refs`, `callers`, `callees`,
  `implementers`, `interface`, and `interfaces`), and current changed-symbol
  inventory plus selected relationship subanalysis in diff-oriented workflows,
  `context symbol`, `impact symbol`, and `agent context` can reuse valid
  snapshots. Unsupported portions fall back to live analysis with warnings.
- Nested modules and workspace modules outside the selected `--root` are
  reported as repository layout boundaries, not silently folded into the
  current analysis. Local `replace` directives are visible as layout evidence;
  replacement roots may still need separate inspection.
- GoSherpa is not a runtime profiler, security scanner, linter, formatter, or
  automatic refactoring engine.

## Links

- [CLI Reference](docs/CLI_REFERENCE.md)
- [Implementation Status](docs/STATUS.md)
- [Agent JSON Schema](docs/product/JSON_SCHEMA_V1.md)
- [Context JSON Schema](docs/product/CONTEXT_SCHEMA_V1.md)
- [Feature Roadmap](docs/product/FEATURE_ROADMAP.md)
- [Repository](https://github.com/panndabea/GoSherpa)
