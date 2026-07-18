# GoSherpa v0.8.0

Status: prepared for release on 2026-07-18.

GoSherpa v0.8.0 is the first broad pre-1.0 agent-first CLI release. It moves
the project beyond the v0.1 Impact Engine MVP into a practical repository
intelligence tool for Go codebases: one local CLI for orientation, focused
symbol context, relationships, impact, test planning, PR summaries, and stable
JSON output.

## Highlights

- `gosherpa agent context --base <ref>` as the primary diff-first workflow for
  coding agents.
- Focused context exports for symbols, files, packages, and diffs, with source
  excerpts, reading order, confidence, limitations, and truncation metadata.
- Typechecked reference, caller, callee, and interface navigation where package
  loading succeeds, with visible AST fallback when it does not.
- Relationship-capable format v2 snapshots with opt-in reuse for inventory,
  selected relationship commands, focused symbol impact/context, diff-oriented
  workflows, and `agent context`.
- Structured test planning with direct, related, contract, caller-package, and
  fallback recommendations.
- Target-risk summaries in impact, context, explain, PR, and agent workflow
  JSON.
- Repository-shape readiness for modules, workspaces, nested modules, build
  tags, generated files, local replacements, package-load diagnostics, and
  large output guardrails.
- Stable JSON envelope for all commands, plus schema and golden fixture
  coverage for agent-facing output.
- Static shell completion generation for zsh, bash, and fish.

## Command Surface

The v0.8.0 CLI includes:

- Readiness and inventory: `doctor`, `analyze`, `snapshot`, `symbols`,
  `symbol`, `search`, `packages`
- Repository structure: `architecture`, `risk`, `deps`
- Relationships: `refs`, `callers`, `callees`, `path`, `paths`, `entrypoints`
- Interfaces: `implementers`, `interface`, `interfaces`
- Context exports: `context symbol`, `context file`, `context package`,
  `context diff`, `agent context`
- Impact and tests: `impact`, `impact file`, `impact package`,
  `impact symbol`, `impact diff`, `tests`, `tests affected`, `pr`
- Shell integration: `completion zsh`, `completion bash`, `completion fish`

## Conservative Boundaries

- GoSherpa remains a local static-analysis CLI, not an IDE, hosted service,
  linter, security scanner, runtime profiler, or code generator.
- Diff impact is hunk-based and does not infer every statement-level semantic
  consequence.
- Caller, callee, path, entrypoint, and interface analysis remain conservative
  around dynamic dispatch, reflection, custom routers, framework wiring,
  dependency internals, reassigned function values, and escaping function
  values.
- Test discovery can miss dynamic table-driven subtest names and
  framework-specific behavior.
- Snapshot reuse is explicit. Missing, stale, invalid, or
  relationship-limited snapshots fall back to live analysis with warnings.
- `agent context` is intentionally diff-first in this release; focused symbol,
  file, and package drill-down stays in `gosherpa context`.

## Verification

Release-prep verification should pass:

```bash
go test ./...
go build -o /private/tmp/gosherpa-v0.8.0 ./cmd/gosherpa
go run ./cmd/gosherpa version
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa agent context --base HEAD --use-snapshot --max-files 5 --max-symbols 5 --max-tests 5 --max-bytes 12000 --json
git diff --check
```

Before publishing a tag, refresh the repository snapshot if it is part of the
release artifact set or if release validation depends on snapshot-backed
counts:

```bash
go run ./cmd/gosherpa snapshot --json
```

## Next

The remaining path to v1.0 is release hardening rather than a major feature
count problem: broader real-world fixtures, tighter release automation,
published installation artifacts, and final compatibility language for the
agent-facing JSON contracts.
