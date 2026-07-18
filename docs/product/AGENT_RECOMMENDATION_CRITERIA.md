# Agent Recommendation Criteria

This document captures the product judgment for when GoSherpa is good enough to
recommend to coding agents, and which failures should block that recommendation.

The short answer is: recommend GoSherpa for Go repositories as the first
structural orientation step before reading or editing code. Do not treat it as
the only source of truth. Its best role is repository intelligence: compact,
deterministic signals about packages, symbols, impact, tests, and uncertainty.

## Current Recommendation

GoSherpa is worth recommending to agents when the task involves a Go module and
the agent needs to answer questions like:

- what does this repository look like?
- where is this symbol defined?
- who calls this code, and what does it call?
- what changed in this diff?
- which packages and tests are probably affected?
- what should I read before editing?
- where is the analysis uncertain?

The recommended agent habit is:

```bash
gosherpa doctor --json
gosherpa snapshot --json
gosherpa agent context --base main --use-snapshot --json --max-files 10 --max-symbols 20 --max-tests 10 --max-bytes 12000
gosherpa context symbol <target> --json --max-references 20 --max-tests 10
gosherpa tests affected --base main --use-snapshot --json
```

Agents should prefer entry-count limits first. Use `--max-bytes` only as a hard
budget guard, because byte trimming can remove otherwise useful source,
reference, or test detail.

## Knockout Criteria

Any item in this section should block a "default recommended for agents" claim
until it is fixed or explicitly documented as out of scope.

### 1. No Clean Build Or Test Baseline

Do not recommend a release or implementation slice if the project cannot pass:

```bash
go test ./...
go build ./cmd/gosherpa
```

Environment-only failures should be documented separately. Code failures block
recommendation.

### 2. Agent-Facing Output Is Not Stable JSON

Agent-facing commands must support `--json` with the shared envelope:

- `schemaVersion`
- `command`
- `target`
- `root`
- `modulePath`
- `warnings`
- `data`

Schema changes must be intentional, documented, and covered by tests or golden
fixtures.

### 3. Context Output Is Unbounded Or Noisy

Any command meant for agent context must have practical size controls and
truncation metadata. Relevant controls include:

- `--max-files`
- `--max-references`
- `--max-symbols`
- `--max-tests`
- `--source-radius`
- `--max-bytes`

If output can explode on normal repositories without a predictable way to bound
it, it is not agent-ready.

### 4. Ambiguous Targets Are Guessed

GoSherpa must not silently pick one symbol when several symbols match. Ambiguous
targets should return structured candidates with package, file, and line
information so an agent can retry with a qualified target.

### 5. Confidence And Limitations Are Missing

Every agent-facing result that influences editing or review should expose:

- `analysisMode`
- `confidence`
- `limitations`
- successful-analysis warnings in the shared `warnings` field

Agents need to know whether an answer came from typechecked loading, AST
analysis, git diff heuristics, or a fallback path.

### 6. Typechecked Analysis Fails Silently

High-risk navigation should use Go semantics where available, especially for:

- references
- callers and callees
- interface implementers
- satisfied interfaces
- package-qualified duplicate names
- build tags
- generic types and methods
- type aliases
- embedded interfaces
- imported package selectors

If typechecked loading fails, the fallback must be visible. Silent fallback is a
trust failure.

### 7. Diff Context Is Missing Or Review-Weak

For review and repair tasks, agents need a compact diff map. A recommendation is
blocked if `agent context` cannot explain:

- changed files
- changed packages
- changed symbols when known
- affected packages
- affected tests
- suggested test commands
- risk and limitations
- reading order

### 8. Test Planning Is Too Vague

Suggested tests must be useful enough to change agent behavior. Affected-test
output should separate, when possible:

- direct tests
- related same-package tests
- caller-package tests
- fallback package commands

Each command should include a reason. A bare list of packages is helpful, but it
is not enough for a strong agent recommendation.

### 9. Results Hide Known Blind Spots

Known blind spots must stay visible in output and docs. Examples include:

- dynamic dispatch
- reflection
- function values
- generated-file policy
- incomplete package loading
- nested modules and `go.work`
- build-tag-sensitive code
- hunk-based changed-symbol detection

An incomplete answer is acceptable. An overconfident incomplete answer is not.

### 10. Commands Have Unexpected Write Side Effects

Repository intelligence commands should be read-only by default. Any command
that writes snapshots, cache files, reports, or generated artifacts must make
that behavior explicit and predictable.

Agents should be able to run `doctor`, `analyze`, `context`, `impact`, `refs`,
`callers`, `callees`, `tests`, and `pr` without modifying the working tree.

## Implementation Checklist

When adding or changing an agent-facing feature, check this list before calling
it ready:

- command works from a module root and with `--root`
- command has human output and stable `--json` output when appropriate
- JSON uses the shared envelope
- warnings are explicit and machine-readable
- output includes `analysisMode`, `confidence`, and `limitations` when it can
  guide editing or review
- output is bounded by entry-count limits, byte limits, or both
- truncation is reported in `data.truncated`
- ambiguous targets return candidates instead of guessing
- build tags are supported or the limitation is visible
- typechecked loading is used where correctness depends on Go semantics
- AST or diff fallback is visible when used
- suggested test commands include reasons
- golden JSON fixtures or schema tests cover the result shape
- docs mention the command, its limits, and its blind spots
- `go test ./...` passes

## What Agents Should Do With GoSherpa Output

Agents should use GoSherpa output as a map, not as a replacement for reading the
code. A good workflow is:

1. Run `doctor` if repository readiness is unknown.
2. Run `snapshot` when repeated queries or snapshot-backed relationships matter.
3. Run `agent context` for review, repair, or edit tasks.
4. Run `context symbol`, `context file`, or `context package` before focused editing.
5. Open the source locations from `readingOrder`.
6. Run the suggested focused tests.
7. Broaden to `go test ./...` when confidence, risk, or changed surface area
   warrants it.

## Current Product Judgment

GoSherpa is already useful enough to recommend as an early agent tool for Go
repositories because it provides:

- repository overview through `analyze`
- readiness checks through `doctor`
- symbol, file, package, and diff context
- references, callers, callees, paths, interfaces, and implementers
- impact and PR summaries
- affected-test suggestions
- stable JSON envelopes
- visible confidence and limitations in the strongest agent-facing commands

The main remaining recommendation risk is not feature count. It is trust:
accuracy, bounded output, visible fallback behavior, and test recommendations
that are specific enough to guide real edits.
