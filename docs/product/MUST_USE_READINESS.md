# GoSherpa Must-Use Readiness

This document captures the current product judgment for what would make
GoSherpa move from "useful sometimes" to "a coding agent should reach for this
early in most Go repository tasks."

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
- context export for symbols, files, packages, and diffs, including target risk,
  confidence, analysis mode, and limitations fields
- PR review summaries through `gosherpa pr --base <ref>`, including changed
  files, packages, symbols, risk notes, target risk, affected tests, and
  verification commands
- stable JSON output for commands
- ambiguity diagnostics and package-qualified target support

The active product direction is agent-first repository intelligence. GoSherpa
now has the underlying context, impact, snapshot, possible-call, test-planning,
entrypoint, and target-risk signals; the next must-use threshold depends on
making the standard agent workflow obvious, bounded, and trustworthy across
workspaces, nested modules, build tags, generated files, partial package-load
failures, local replacements, and larger repositories.

Estimated readiness: roughly 86-88 percent of the must-use threshold.

## Must-Use Threshold

GoSherpa becomes a must-use tool when this habit feels obvious to coding
agents and still reads well for humans:

```bash
gosherpa doctor --json
gosherpa snapshot --json
gosherpa agent context --base <base-ref> --use-snapshot --json
gosherpa context symbol ./internal/auth.ValidateToken --use-snapshot --json
gosherpa tests affected --base <base-ref> --use-snapshot --json
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

These are the trust tracks that most directly decide whether GoSherpa becomes a
default tool. The previous P0 tracks are now baseline. The active P0 order is:
zero-friction agent workflow, real-world repository robustness, then stronger
tests and entrypoint intelligence.

### 1. Zero-Friction Agent Workflow

Status: first public slice implemented as
`gosherpa agent context --base <base-ref>`.

The command is diff-first and composes readiness, snapshot status, bounded diff
context, target risk, interface signals, affected packages, affected tests,
section modes, limitations, and suggested follow-up commands into one shared
JSON-envelope result. It supports `--base`, `--use-snapshot`, `--tags`,
`--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes`, and `--json`, plus global
`--root`. It intentionally rejects positional targets, `--tests`, `--scope`,
`--max-references`, and `--source-radius`.

Done for the first slice:

- one command gives a coherent pre-edit/pre-PR bundle for a diff
- JSON shape is additive to schema v1 and covered by golden/schema tests
- human output stays short
- composed byte budgets keep JSON valid and report aggregate plus per-section
  truncation metadata
- missing, stale, invalid, and valid snapshot states are covered for the agent
  workflow
- focused symbol/file/package drill-down remains with `gosherpa context`
- snapshot reuse remains opt-in and reports refresh guidance

Continuing focus areas:

- repository robustness fixtures for real-world Go layouts

Why this matters:

Agents should not have to stitch together `doctor`, `context diff`,
`impact diff`, `tests affected`, and `pr` before every ordinary edit.

### 2. Typechecked Loading Everywhere It Matters

Move the highest-risk analysis paths from AST-first heuristics toward Go
semantics using `go/packages` and the Go type checker.

Current status: priority implementation slice complete. `gosherpa refs`, call
graph commands, standalone interface navigation, context exports, impact
reports, affected-test reports, and PR summaries use typechecked paths where
available, with conservative AST fallback and structured warnings when package
loading is partial. A shared in-memory semantic context now reuses typechecked
repository loads across the main agent-facing flows, including `context diff`,
`impact diff`, `tests affected`, and `pr`, while snapshot reuse remains scoped
to inventory and current changed-symbol inventory.

Continuing focus areas:

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

### 3. Production-Ready Focused Context Export

The existing `gosherpa context symbol|file|package|diff` commands are the
focused drill-down layer beneath the agent-first daily driver,
`gosherpa agent context`. Keep them compact, bounded, and trustworthy for
symbol, file, package, and diff exploration.

Current status: priority implementation slice complete. Context schema docs,
golden fixtures, and schema guard tests are aligned for symbol, file, package,
and diff context. Context limit support is command-specific and documented;
unsupported limit combinations fail intentionally. Byte budgets keep output
valid and report truncation metadata. Confidence, analysis modes, warnings, and
limitations now consistently describe fallback modes, package-load warnings,
generated-file policy, nested-module boundaries, hunk-based diff limits, and
test-analysis limits.

Done when:

- context output is bounded, deterministic, and safe to paste into an agent
  workflow
- ambiguous targets return candidates instead of guessing
- every context result explains its confidence and known blind spots

Why this matters:

Agents need enough context to make a safe edit, not every fact in the
repository. A strong context command makes GoSherpa the first stop.

### 4. Structured Test Planning

Make test recommendations explain the tradeoff between fast confidence and broad
coverage.

Current status: priority implementation slice complete. Test plans expose
direct, related, interface/implementation contract, caller-package, and fallback
groups with reasons, concrete tests and targets when known, and non-nil arrays.
Diff-oriented reports attach changed-symbol targets when known, keep package
fallback for Go changes that cannot be narrowed to direct tests, and use
whole-repository fallback when changed files cannot be narrowed to
repository-local Go packages.

Recommended groups:

- direct tests that reference the changed symbol
- related tests in the same package
- caller-package tests
- interface or implementation contract tests
- fallback commands for broader safety

Current contract:

- JSON distinguishes `direct`, `related`, `contracts`, `callerPackages`, and
  `fallback` test commands
- each suggested command includes a reason
- diff-based recommendations include changed symbols when known
- empty direct results still provide a practical package or whole-repository
  fallback

Why this matters:

The tool should help users avoid both extremes: running too little and running
the whole suite by default.

### 5. One Pull-Request Intelligence Command

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
- target risk summary
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

### 6. Incremental Snapshot Or Index

Add an explicit persistent repository snapshot for repeated queries.

Status: implemented as explicit `gosherpa snapshot` creation with format v2
relationship-capability metadata. `gosherpa doctor` reports missing, valid,
stale, and invalid snapshots. Inventory commands, selected standalone
relationship commands, `context symbol`, `impact symbol`, diff-oriented
workflows, and the first `agent context` workflow can opt in to snapshot reuse;
unsupported portions fall back to live analysis with warnings.

Command sketch:

```bash
gosherpa snapshot
gosherpa agent context --base main --use-snapshot --json
```

Done when:

- snapshot creation is explicit
- invalidation is based on file content, mod time, or git state
- commands can reuse a valid snapshot
- stale snapshots produce clear diagnostics
- the cache format is versioned

### 7. Source Snippet Ranges

Expose enough position data for tools to open the exact source occurrence.

Current status: symbol definitions, references, call sites, call paths, related
tests, and direct test target references include columns and source ranges when
Go parser or typechecker positions identify the source span.

Done when:

- definitions include start and end positions
- references include exact occurrence locations where possible
- direct related and affected tests identify exact target occurrences where
  possible
- JSON keeps paths relative to the selected root

### 8. `gosherpa doctor`

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
