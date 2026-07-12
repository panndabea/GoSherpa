# GoSherpa Must-Use Readiness

This document captures the current product judgment for what would make
GoSherpa move from "useful sometimes" to "I should reach for this early in most
Go repository tasks."

Use this as a decision lens for future planning and implementation steps. The
feature roadmap and PRDs remain the broader source of truth; this file ranks the
work by what most increases daily trust and default usage.

## Current Judgment

GoSherpa already has real MVP substance:

- symbol discovery, lookup, search, and explanation
- references, callers, callees, and repository-local call paths
- interface implementer and satisfied-interface navigation
- file, package, symbol, and diff impact analysis
- affected test suggestions
- context export for symbols, files, packages, and diffs, including confidence,
  analysis mode, and limitations fields
- PR review summaries through `gosherpa pr --base <ref>`, including changed
  files, packages, symbols, risk notes, affected tests, and verification
  commands
- stable JSON output for commands
- ambiguity diagnostics and package-qualified target support

The remaining gap is not mostly feature quantity. The remaining gap is trust:
ordinary Go repositories need to be analyzed accurately enough that a developer
or coding agent will prefer GoSherpa before opening many files manually.

Estimated readiness: roughly 70-75 percent of the must-use threshold.

## Must-Use Threshold

GoSherpa becomes a must-use tool when this habit feels obvious:

```bash
gosherpa context diff --base main
gosherpa context symbol ./internal/auth.ValidateToken --json
gosherpa impact symbol ./internal/auth.ValidateToken
gosherpa tests affected --base main
```

The output should give a compact, trustworthy editing map:

- what changed or what is being inspected
- where the relevant code lives
- who calls it and what it calls
- which interfaces and implementers are involved
- what packages and tests may be affected
- which uncertainty remains
- which command should be run next

## P0 Work

These are the next items that most directly decide whether GoSherpa becomes a
default tool.

### 1. Typechecked Loading Everywhere It Matters

Move the highest-risk analysis paths from AST-first heuristics toward Go
semantics using `go/packages` and the Go type checker.

Current status: `gosherpa refs`, call graph commands, and standalone interface
navigation have `go/packages`-backed typechecked paths with AST fallback. A
first in-memory semantic context now shares one typechecked repository load
across symbol identity, references, call signals, file/package context symbol
inventory, and context interface-impact signals for `explain`, `context symbol`,
`context file`, `context package`, symbol impact, and changed-symbol impact
paths. Context and report-based impact bundles expose `interfaceAnalysisMode`
for the interface subanalysis, but still need broader shared semantic loading
across every analysis field.

Focus areas:

- `go.work`
- nested modules
- build tags
- generated-file policy
- generic types and methods
- type aliases
- embedded interfaces
- imported package selectors
- method values
- package-qualified duplicate names
- external dependencies where useful for resolution

Done when:

- symbols, references, callers, callees, interfaces, context, and impact can use
  typechecked loading
- failed package loads are reported as warnings instead of silently ignored
- AST fallback remains available when full loading is unavailable
- fixtures cover build tags, `go.work`, aliases, generics, embedded interfaces,
  and cross-package method calls

Why this matters:

Accuracy is the main trust gate. If GoSherpa misses a caller or reports the
wrong implementer, users and agents fall back to manual search.

### 2. Production-Ready Agent Context Export

The existing `gosherpa context symbol|file|package|diff` commands are the right
center of gravity. Make them the first command a coding agent wants to run
before editing.

Needed next:

- keep documented JSON schemas aligned with golden fixtures and command output
- keep byte-budget and entry-count controls reliable across context outputs:
  `--max-bytes`, `--max-files`, `--max-references`, `--max-symbols`,
  `--max-tests`, and `--source-radius`
- broaden and document `confidence`, `analysisMode`, and `limitations`
  behavior consistently beyond the existing context reports
- concise human warnings without noisy terminal output
- consistent behavior for symbol, file, package, and diff targets

Done when:

- context output is bounded, deterministic, and safe to paste into an agent
  workflow
- ambiguous targets return candidates instead of guessing
- every context result explains its confidence and known blind spots

Why this matters:

Agents need enough context to make a safe edit, not every fact in the
repository. A strong context command makes GoSherpa the first stop.

### 3. Structured Test Planning

Make test recommendations explain the tradeoff between fast confidence and broad
coverage.

Current status: first structured slice implemented. Test plans expose direct,
related, caller-package, and fallback groups with reasons, and diff-oriented
reports attach changed-symbol targets when known.

Recommended groups:

- direct tests that reference the changed symbol
- related tests in the same package
- caller-package tests
- interface or implementation contract tests
- fallback commands for broader safety

Done when:

- JSON distinguishes `direct`, `related`, and `fallback` test commands
- each suggested command includes a reason
- diff-based recommendations include changed symbols when known
- empty direct results still provide a practical package fallback

Why this matters:

The tool should help users avoid both extremes: running too little and running
the whole suite by default.

### 4. One Pull-Request Intelligence Command

Status: first slice implemented as `gosherpa pr --base <ref>` with human and
JSON output.

Keep improving the high-level review command that composes existing impact,
symbol, interface, and test engines.

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
- verification commands

Done for the first slice:

- human output is short enough for a PR comment
- JSON output is stable enough for CI and agents
- the command is a thin composition over existing analysis APIs

Why this matters:

For review, repair, and pre-merge tasks, the first question is usually: what did
this change touch, how risky is it, and what should I run?

## P1 Work

These items make GoSherpa easier to keep in the workflow after the P0 trust
gates are handled.

### 5. Incremental Snapshot Or Index

Add an explicit persistent repository snapshot for repeated queries.

Status: first slice implemented as `gosherpa snapshot`, writing a versioned
`.gosherpa/snapshot.json` inventory with file freshness metadata, package
summaries, symbols, build tags, and git state. `gosherpa doctor` reports
missing, valid, stale, and invalid snapshots. `analyze`, `symbols`, `symbol`,
`search`, and test-inclusive `packages --tests` can opt in to snapshot reuse;
broader command reuse remains future work.

Command sketch:

```bash
gosherpa snapshot
gosherpa context diff --base main --use-snapshot
```

Done when:

- snapshot creation is explicit
- invalidation is based on file content, mod time, or git state
- commands can reuse a valid snapshot
- stale snapshots produce clear diagnostics
- the cache format is versioned

### 6. Source Snippet Ranges

Expose enough position data for tools to open the exact source occurrence.

Current status: symbol definitions include columns and source ranges. References,
call sites, and tests still need exact occurrence ranges.

Done when:

- definitions include start and end positions
- references include exact occurrence locations where possible
- JSON keeps paths relative to the selected root

### 7. `gosherpa doctor`

Status: first slice implemented as `gosherpa doctor` with human and JSON output.

Keep improving the command that explains analysis readiness.

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

Why this matters:

When results look incomplete, users need to know whether the limitation comes
from the repository, the environment, or GoSherpa itself.

## P2 Work

These features turn GoSherpa into infrastructure for other tools, but they
should follow the trust work above.

- MCP or long-running server mode
- editor integration hooks
- optional TUI
- graph export
- architecture reports
- GitHub Action

These can be valuable, but they should not outrank semantic accuracy, context
quality, test planning, and PR intelligence.

## Future Step Checklist

Before choosing the next implementation task, ask:

- Does this increase trust in ordinary Go projects?
- Does this reduce the need to open many files manually?
- Does this make impact or test selection more precise?
- Does this expose uncertainty clearly instead of hiding it?
- Does this improve the `context` or `pr` workflow?
- Does this preserve stable human output and JSON output?

Prefer work that answers "yes" to several of these questions.

## Not The Immediate Threshold

The following are useful, but they are not what would make GoSherpa must-use by
themselves:

- more standalone commands without improving the main workflow
- a TUI before semantic trust is stronger
- an MCP server before schemas and confidence fields are stable
- broad architecture reports before PR and context workflows are excellent
- full SSA, pointer analysis, reflection analysis, or runtime inference

GoSherpa should stay conservative: small, trustworthy answers are better than
large, overconfident ones.

## Definition Of Must-Use

GoSherpa reaches must-use quality when a developer or coding agent can enter an
unfamiliar Go repository and, within a few commands, answer:

- What is this symbol and where is it defined?
- Who calls it and what does it call?
- Which interfaces or implementations are involved?
- What might break if I change it?
- Which tests should I run first?
- What uncertainty remains?
- What should I inspect next?

At that point, GoSherpa is not just a CLI with many commands. It is a dependable
repository intelligence layer for Go.
