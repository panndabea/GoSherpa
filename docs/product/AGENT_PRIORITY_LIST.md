# GoSherpa Agent Priority List

This list describes the features that would make GoSherpa a default tool for a
coding agent working in real Go repositories.

It is intentionally narrower than the general product roadmap. The question is
not "what would be nice to have?", but:

```text
What would make an agent reach for GoSherpa before opening a dozen files?
```

## Decision Rule

Prioritize work that helps an agent:

- find the right code faster
- avoid changing the wrong symbol
- understand impact before editing
- choose the smallest useful test set
- explain uncertainty instead of pretending the graph is complete
- consume results through stable JSON without losing human-readable output

Human CLI output should remain first-class. Agent surfaces should be explicit,
stable, and boring in the best possible way.

## P0 - Default-Agent-Use Blockers

These are the features that would move GoSherpa from "useful sometimes" to
"run this early in most Go tasks".

### 1. Zero-Friction Agent Workflow

Add one primary command that returns the compact diff-oriented bundle an agent
needs before editing or preparing a PR.

Implemented first public slice:

```bash
gosherpa agent context --base <base-ref>
gosherpa agent context --base <base-ref> --use-snapshot --json
```

The command is diff-first only. It composes readiness, snapshot status,
changed files/packages/symbols, bounded reading order, target risk, affected
packages, possible runtime relationship summaries when snapshot-backed,
interface summaries, test plan, suggested commands, section modes, confidence,
and limitations. It does not accept symbol, file, package, or free-form
positional targets; focused drill-down remains with the existing context
commands.

Implemented supporting context slices:

```bash
gosherpa context symbol <target>
gosherpa context symbol <target> --json
gosherpa context file <file>
gosherpa context file <file> --json
gosherpa context package <package>
gosherpa context package <package> --json
gosherpa context diff --base <ref>
gosherpa context diff --base <ref> --json
```

This supports symbol targets with source excerpt, symbol identity, references,
callers, callees, impact signals, related tests, suggested commands,
confidence, limitations, and opt-in test callers through `--tests`.
File context exports file symbols, source excerpts, package-level impact,
affected tests, suggested commands, reading order, confidence, and limitations.
Package context exports package files, symbols, source excerpts, package-level
impact, affected tests, suggested commands, reading order, confidence, and
limitations.
Diff context exports changed files, changed packages, changed symbols, affected
packages, affected tests, suggested commands, reading order, confidence, and
limitations. It is now a lower-level focused command used by
`agent context`, not the primary daily-driver command.

The current context JSON already includes `analysisMode`, `confidence`, and
`limitations`, with schema docs, entry-count size controls, and `--max-bytes`
byte-budget control for context exports. The remaining hardening work is
broader consistency across agent-facing commands.

Current daily-driver shape:

```bash
gosherpa doctor --json
gosherpa snapshot --json
gosherpa agent context --base <base-ref> --use-snapshot --json
gosherpa context symbol ./internal/auth.ValidateToken --use-snapshot --json
gosherpa tests affected --base <base-ref> --use-snapshot --json
```

The command name is locked for this plan. Later agent workflow targets may be
added only after the diff-first workflow is robust.

The output should include:

- target identity and disambiguated package
- definition location
- signature
- documentation
- concise source excerpt
- references
- direct callers
- direct callees
- interface relationships
- implementers or satisfied interfaces when relevant
- affected packages
- related tests
- suggested `go test` commands
- known limitations
- confidence level

Why this helps:

An agent usually needs enough context to make a safe edit, not every possible
fact about the repository. A context command would reduce random file opening
and make GoSherpa the first stop for symbol-level work.

Done for the first slice:

- `--json` output has a documented schema.
- Human output is readable in a terminal.
- Output has size controls: `--max-files`, `--max-symbols`, `--max-tests`, and
  `--max-bytes` for the composed payload.
- `--base` is required.
- Symbol, file, package, and free-form targets are rejected for this command.
- Unsupported inherited flags are rejected with explicit validation messages.

Still needed:

- More real-world repository fixture coverage.

### 2. Real-World Repository Robustness

Make analysis behavior explicit and reliable across the Go repository shapes
agents encounter every day.

Focus areas:

- `go.work`
- nested modules
- build tags
- generated files policy
- local replacements
- partial package-load failures
- large monorepo boundaries

Why this helps:

The daily-driver command is only habit-forming if repository readiness warnings
make incomplete analysis obvious before an agent trusts the answer.

Done when:

- `agent context`, focused context commands, impact, tests, and doctor agree on
  readiness and fallback wording.
- Fixtures cover the repository-shape cases above.
- Partial package failure keeps useful AST fallback while surfacing confidence
  and limitations.

### 3. Typechecked Repository Loading

Move the highest-risk analysis paths from AST-only heuristics toward Go
semantics using `go/packages` and the Go type checker.

Current status: `gosherpa refs`, call graph commands, and standalone interface
navigation have `go/packages`-backed typechecked paths with AST fallback. A
first in-memory semantic context shares typechecked repository loads for symbol
identity, references, calls, file/package context symbol inventory, and context
interface-impact signals. Context and report-based impact bundles now expose
`interfaceAnalysisMode` for the interface subanalysis, but still need broader
shared semantic loading across every analysis field.

Focus areas:

- `go.work`
- nested modules
- build tags
- generated files policy
- generic types and methods
- type aliases
- embedded interfaces
- imported package selectors
- method values
- package-qualified duplicate names
- external dependencies where useful for resolution

Why this helps:

If GoSherpa misses a caller or reports the wrong implementer, an agent has to
fall back to manual search. Accuracy is the main trust gate.

Done when:

- Typechecked loading is available for symbols, references, call graph,
  interfaces, and impact.
- Failed package loads are reported as warnings, not silently ignored.
- Existing AST behavior remains available as a fallback when full loading is not
  possible.
- Fixtures cover build tags, `go.work`, aliases, generics, embedded
  interfaces, and cross-package method calls.

### 4. Confidence And Limitations In Every Agent-Facing Result

Expose what GoSherpa knows, what it inferred, and what it could not prove.

Suggested JSON fields:

```json
{
  "confidence": "high",
  "limitations": [
    "dynamic dispatch not resolved",
    "2 packages failed to typecheck"
  ],
  "analysisMode": "typechecked"
}
```

Why this helps:

Agents need to decide whether to trust the answer, broaden the search, or run
extra checks. A conservative answer with visible limitations is much more useful
than an overconfident incomplete one.

Done when:

- JSON output includes `confidence`, `analysisMode`, and `limitations` for
  context, impact, callers, callees, interfaces, and tests. Context has the
  first slice.
- Bundles that include interface or implementation impact expose
  `interfaceAnalysisMode` separately from the broader bundle `analysisMode`.
- Human output shows warnings without becoming noisy.
- Confidence is rule-based and deterministic.

### 5. Test Planning And Entrypoint Intelligence

Make test recommendations and runtime entrypoint hints stronger first-class
signals in agent workflows.

Suggested output groups:

- direct tests that reference the changed symbol
- same-package tests
- caller-package tests
- interface or implementation contract tests
- broader fallback commands

Command sketches:

```bash
gosherpa tests affected --base main --json
gosherpa context diff --base main --tests
```

Why this helps:

An agent often has to choose between running too few tests and running the whole
suite. Grouped recommendations make the tradeoff visible.

Done when:

- JSON distinguishes `direct`, `related`, `contracts`, and `fallback` test
  commands.
- Each suggested command includes a reason.
- Diff-based recommendations include changed symbols when known.
- Empty test results include the package fallback command when appropriate.

## P1 - Make It Fast Enough To Use Repeatedly

These features are not as important as correctness, but they determine whether
GoSherpa stays in the workflow during a long task.

### 6. Incremental Snapshot Or Index

Add a persistent repository snapshot for repeated queries.

Status: implemented with explicit `gosherpa snapshot` creation, versioned
`.gosherpa/snapshot.json` output, freshness metadata, package and symbol
inventory, build tags, git state, relationship-capability metadata, and
`doctor` diagnostics for missing, valid, stale, and invalid snapshots. Opt-in
reuse is available for inventory commands, selected standalone relationship
commands, `context symbol`, `impact symbol`, diff-oriented workflows, and
`agent context`.

Command sketches:

```bash
gosherpa analyze
gosherpa snapshot
gosherpa context diff --base main --use-snapshot
```

The snapshot could live under `.gosherpa/` or a cache directory.

Why this helps:

Agents ask many small questions. Re-scanning a large repository for every query
discourages use.

Done when:

- Snapshot creation is explicit.
- Snapshot invalidation is based on file content, mod time, or git state.
- Query commands can use the snapshot automatically when valid.
- Stale snapshots produce clear diagnostics.
- The cache format has a version.

### 7. One Pull-Request Intelligence Command

Status: first slice implemented as one high-level command for change review.

Command sketch:

```bash
gosherpa pr --base main
gosherpa pr --base main --json
```

The first slice summarizes:

- changed files
- changed packages
- changed symbols
- affected symbols
- affected interfaces and implementers
- risk notes
- recommended tests
- commands to verify the change

Why this helps:

For review and repair tasks, the agent's first question is usually "what did
this change touch and what should I run?" One command should answer that.

Done for the first slice:

- The command is a thin composition over existing impact, symbol, interface, and
  test engines.
- Human output is short enough for a PR comment.
- JSON output is stable enough for CI and agents.

### 8. Source Snippet Ranges

Status: implemented for symbols, references, callers, callees, call path steps,
related tests, and range-backed reading-order entries in JSON output.

Expose file, line, column, and optional range information.

Why this helps:

Agents can open exactly the right code instead of reading whole files. This also
improves editor, MCP, and future IDE integrations.

Done for the first slice:

- Symbol definitions include start and end positions.
- References include enough position data to locate the exact occurrence.
- Direct related and affected tests include exact target occurrence ranges when
  available.
- JSON keeps relative paths stable from the selected root.

## P2 - Make GoSherpa Easy To Integrate Everywhere

These features turn the CLI into infrastructure other tools can rely on.

### 9. MCP Or Long-Running Server Mode

Expose GoSherpa through a local protocol that avoids repeated process startup
and makes queries discoverable.

Command sketch:

```bash
gosherpa serve
```

Useful operations:

- resolve symbol
- explain symbol
- find references
- find callers
- find callees
- analyze diff
- suggest tests
- export context

Why this helps:

Agents can call structured tools directly instead of formatting shell commands
and parsing text.

Done when:

- Server responses reuse the same schema as CLI JSON.
- Server startup validates the repository root.
- Long-running mode reuses the snapshot or in-memory index.

### 10. Schema Documentation And Compatibility Policy

Document the JSON contracts as product surface.

Why this helps:

Agents, CI jobs, and editor integrations need to know when output changes are
compatible.

Done when:

- Each JSON command has an example fixture.
- Schema versioning rules are written down.
- Breaking fields require a schema version bump.
- Golden tests cover representative real outputs.

### 11. `gosherpa doctor`

Add a command that explains analysis readiness.

Command sketch:

```bash
gosherpa doctor
```

It should report:

- module root
- Go version
- `go.work` detection
- package load failures
- build tags in use
- generated-file policy
- ignored directories
- snapshot status
- unsupported analysis cases found

Why this helps:

When results look incomplete, an agent needs to know whether the tool, the repo,
or the current environment is the reason.

Done when:

- `doctor` exits non-zero only for real failures.
- Warnings are structured in JSON.
- The output suggests specific next commands or flags.

## Suggested Build Order

1. Harden `gosherpa agent context --base <base-ref>` for size, snapshot
   ergonomics, and real repository shapes.
2. Broaden real-world repository robustness fixtures.
3. Promote tests and entrypoints as stronger first-class workflow signals.
4. Continue semantic accuracy work where it improves those workflows.
5. Defer MCP, TUI, graph export, and hosted/server surfaces until the daily
   workflow is reliable.

## The Real Adoption Threshold

I would start using GoSherpa by default when this workflow is reliable:

```bash
gosherpa agent context --base <base-ref> --use-snapshot --json
gosherpa context symbol <package-qualified-symbol> --json
gosherpa tests affected --base <base-ref> --use-snapshot --json
```

Those commands need to be fast, deterministic, explicit about uncertainty, and
accurate enough that they reduce manual search instead of creating verification
work.
