# GoSherpa Feature Roadmap

GoSherpa should help people understand Go codebases quickly and confidently.

The primary product is a human-friendly code navigation tool. It should answer
the questions developers ask before, during, and after a change:

- What is this symbol?
- Where is it defined?
- Where is it used?
- Who calls it?
- What does it call?
- Which packages depend on this package?
- What might break if I change this?
- Which tests should I run?

Agent-friendly usage should be designed in from the beginning, but it should not
lead the product. The default experience should remain clear terminal output for
humans. Machine-readable output can be added as a stable alternate surface once
the underlying concepts are reliable.

## Product Positioning

GoSherpa is structural code intelligence for Go projects.

It is not an IDE replacement, a static analyzer suite, or a code generation
framework. It is a fast command-line companion for answering navigation and
impact questions inside a repository.

The Impact Engine direction from [PRD_V01.md](PRD_V01.md) is implemented as the
v0.1 MVP: a conservative Change Intelligence CLI with diff-based impact
analysis, package-level affected tests, and interface impact signals. The next
product direction is Symbol Intelligence from [PRD_V02.md](PRD_V02.md), centered
on richer symbol profiles and `gosherpa explain`.

Core promise:

```text
Ask a code-structure question. Get a small, trustworthy answer.
```

Design principles:

- Human-readable by default.
- Deterministic output.
- Small commands that compose well.
- Prefer Go semantics over text matching where correctness matters.
- Show enough context to support decisions, not so much that output becomes
  noisy.
- Keep the implementation explainable.
- Do not require annotations, generated metadata, or project-specific config for
  normal use.

## Current Baseline

Implemented:

- Repository scanning.
- Go file discovery.
- Struct discovery.
- Interface discovery.
- Function discovery.
- Method discovery.
- Symbol lookup.
- Go-aware reference lookup.
- Initial `gosherpa explain <symbol>` profile with purpose, definition, reading
  order, references, callers, callees, impact signals, tests, and JSON output.
- Direct symbol and package impact analysis.
- Related test discovery with suggested `go test` commands.
- Package dependency analysis.
- Direct syntactic callee analysis.
- Direct syntactic caller analysis.
- Shortest and limited repository-local call path analysis.
- Machine-readable JSON output for all commands with a stable response envelope.
- Golden JSON fixtures for all commands.
- Repository root selection with `--root`.
- Stable CLI exit codes with diagnostics on stderr.
- Git changed-file discovery foundation via `internal/git.ChangedFiles`.
- Changed-package mapping for git diffs via `internal/impact.ChangedPackages`.
- Diff impact report foundation via `internal/impact.AnalyzeDiff`.
- `gosherpa impact diff --base <ref>` with human and JSON output.
- `gosherpa tests affected --base <ref>` with human and JSON output.
- `gosherpa impact file|package|symbol` with human and JSON output.
- Interface and implementer impact signals based on local method sets with
  import-aware signature matching and embedded-interface expansion.
- Changed-symbol extraction from git diff hunks, including deleted symbols from
  base files.
- Package-qualified symbol impact for references and affected tests.
- Transitive caller impact for symbol changes.
- Affected-test planning for transitive caller packages.

Current limitations:

- References are type-aware inside packages and recognize local package selector
  calls, but do not yet use full module/package loading.
- Symbol impact includes transitive callers and package tests for affected
  caller packages.
- Diff impact is hunk-based; it reports directly changed or deleted top-level
  Go functions and struct/interface types, but it does not infer every semantic
  consequence of changed statements.
- Package-qualified symbol impact disambiguates references and affected tests;
  unqualified symbol targets can still be ambiguous across packages.
- Interface implementer impact canonicalizes local/external import paths in
  method signatures and resolves local embedded interfaces, but does not yet run
  the full Go type checker for aliases, build tags, or generic edge cases.
- Test discovery uses same-package tests and syntactic direct-reference
  matching; table-test names are not extracted yet.
- Callers and callees are AST-based and can miss receiver-variable method calls.
- Call paths inherit the current AST-based caller and callee limitations.
- Function names can be ambiguous across packages.
- Positions only expose file and line, not columns or ranges.
- Tests are skipped by some analysis paths and are not yet first-class.

## Roadmap Overview

The roadmap is organized around the order in which a developer naturally moves
through a change:

1. Find the relevant code.
2. Understand what it is.
3. Understand who uses it.
4. Understand what it depends on.
5. Estimate impact.
6. Run the right tests.
7. Navigate interactively when the answer is bigger than one command.

Recommended phases:

| Phase | Theme | Outcome |
| --- | --- | --- |
| 0 | CLI and data foundations | Commands are consistent, deterministic, and easy to trust. |
| 1 | Find and inspect | Developers can find symbols and understand definitions quickly. |
| 2 | Semantic references | References and relationships become Go-aware. |
| 3 | Interfaces and implementations | GoSherpa answers core Go design questions. |
| 4 | Call graph and paths | Developers can move through execution relationships. |
| 5 | Tests and impact | GoSherpa helps plan safe changes. |
| 6 | Interactive navigation | Bigger explorations become comfortable. |
| 7 | Machine-readable surfaces | Agents and scripts can consume the same intelligence reliably. |

## Phase 0: CLI and Data Foundations

Goal: make the existing tool feel reliable before adding bigger features.

### 0.1 Repository Root Selection

Status: implemented.

Human question:

```text
Can I run GoSherpa from anywhere and point it at the project?
```

Command sketch:

```bash
gosherpa symbols --root /path/to/repo
gosherpa refs ParseFile --root .
```

Implemented behavior:

- Global `--root` flag.
- Default to `.` when omitted.
- File paths are normalized relative to the chosen root in output.
- CLI roots are rejected when they do not contain a direct `go.mod`.
- Internal analysis remains testable against loose
  Go files.

Done:

- Every command accepts the same root selection behavior.
- Tests cover root normalization.
- Output stays stable regardless of the shell's current working directory.

### 0.2 Output Consistency

Human question:

```text
Can I scan GoSherpa output quickly without relearning every command?
```

Requirements:

- Use consistent headings.
- Sort all result lists deterministically.
- Use consistent empty states.
- Print errors to stderr.
- Print successful command output to stdout.
- Avoid surprising icons in contexts where plain text is clearer.

Suggested output shape:

```text
CALLERS

Target: UserService.Create

  Handler.CreateUser    internal/http/handler.go:42
  TestCreateUser        internal/user/service_test.go:18

Found 2 callers
```

Done when:

- All commands follow a shared formatting style.
- Snapshot-style tests cover important output formats.
- Empty results are explicit and helpful.

### 0.3 Exit Codes

Status: implemented.

Human question:

```text
Did the command succeed, find nothing, or fail?
```

Requirements:

- Exit `0` for successful command execution.
- Exit `1` for runtime and analysis errors.
- Exit `2` for invalid usage.
- Print diagnostics and usage errors to stderr.
- Keep successful command output on stdout.

Done when:

- Exit behavior is documented.
- Tests cover usage errors, repository errors, and empty results.

### 0.4 Shared Result Model

Human question:

```text
Can every command name code locations the same way?
```

Requirements:

- Introduce shared internal types for:
  - repository root
  - package path
  - symbol identity
  - source position
  - source range
  - command result metadata
- Add columns and end positions where Go parser data supports them.
- Keep display formatting separate from analysis.

Done when:

- Analysis functions return structured results.
- Formatting functions do not perform repository analysis.
- Future JSON output can be added without rewriting every command.

## Phase 1: Find and Inspect

Goal: make GoSherpa useful at the very beginning of a coding session.

### 1.1 Search

Human question:

```text
I remember part of the name. Where is the thing?
```

Command sketch:

```bash
gosherpa search user create
gosherpa search repository --kind interface
gosherpa search handler --package ./internal/http
```

MVP behavior:

- Search symbols by partial, case-insensitive name matching.
- Match multiple query terms.
- Search functions, methods, structs, interfaces, constants, variables, and
  tests once those symbols are indexed.
- Support filters:
  - `--kind`
  - `--package`
  - `--tests`
  - `--limit`

Nice follow-ups:

- Fuzzy ranking.
- Exact-match boost.
- Package-name boost.
- Recently touched file boost is out of scope for now because it requires VCS
  integration and can make output less deterministic.

Done when:

- A developer can find common symbols without knowing exact names.
- Ambiguous names are shown as a ranked list instead of becoming an error.

### 1.2 Rich Symbol Details

Human question:

```text
What exactly is this symbol?
```

Command sketch:

```bash
gosherpa symbol UserService.Create
gosherpa symbol ./internal/user.UserService.Create
gosherpa symbol UserRepository --context
```

MVP behavior:

- Show symbol kind.
- Show package.
- Show file, line, and column.
- Show receiver for methods.
- Show signature for functions and methods.
- Show fields for structs.
- Show method set for interfaces.
- Show doc comment when present.
- Show short source context with `--context`.

Done when:

- Looking up a symbol usually removes the need to open the file immediately.
- Ambiguous lookups show all candidates and explain how to disambiguate.

### 1.3 Symbol Filters

Human question:

```text
Can I list just the things I care about?
```

Command sketch:

```bash
gosherpa symbols --kind struct
gosherpa symbols --kind method --package ./internal/sherpa
gosherpa symbols --tests
```

Requirements:

- Filter by kind.
- Filter by package.
- Include or exclude tests explicitly.
- Support stable sorting by package, kind, name, and file.

Done when:

- `symbols` remains useful in medium-size repositories instead of becoming a
  wall of output.

### 1.4 Source Context

Human question:

```text
Can I see enough code around the result to know whether it matters?
```

Command sketch:

```bash
gosherpa refs ParseFile --context
gosherpa callers UserService.Create --context 3
```

Requirements:

- Add optional context lines around locations.
- Highlight or mark the relevant line in plain text.
- Keep context disabled by default for compact output.
- Allow a numeric value for number of surrounding lines.

Done when:

- Context output is readable in a terminal and covered by tests.
- Missing files or unreadable files produce clear errors.

## Phase 2: Semantic References

Status: MVP implemented with per-package `go/types` object matching and local
module import selector matching.

Goal: replace fragile text-like references with Go-aware relationships.

### 2.1 Type-Checked Loading

Human question:

```text
Can I trust that this reference is really the same symbol?
```

Implementation direction:

- Use `golang.org/x/tools/go/packages`.
- Load packages with syntax, type info, compiled files, imports, and module
  metadata.
- Keep an AST-only fallback only where semantic loading is unavailable or too
  expensive.

Requirements:

- Resolve identifiers to `types.Object`.
- Track definitions and references through Go type information.
- Handle import aliases.
- Handle package-qualified function calls.
- Handle receiver methods.
- Include tests when requested.

Done when:

- `refs` no longer reports unrelated identifiers with the same spelling.
- Method references work through receiver variables.
- Tests cover same-name symbols in different packages.

### 2.2 Reference Kinds

Human question:

```text
Is this a read, a write, a call, a type usage, or something else?
```

Command sketch:

```bash
gosherpa refs UserService.Create --kind call
gosherpa refs User --kind type
```

Reference kinds:

- definition
- call
- read
- write
- type usage
- field access
- method expression
- method value
- interface satisfaction
- import

MVP:

- Start with definition, call, type usage, field access, and import.
- Add read/write classification later.

Done when:

- References are more informative than a list of coordinates.
- Output remains compact by grouping by kind or package.

### 2.3 Ambiguity Handling

Human question:

```text
There are three New functions. Which one did I mean?
```

Requirements:

- Detect ambiguous symbol names.
- Show candidate list.
- Provide exact disambiguation examples.

Example:

```text
Ambiguous symbol: New

Candidates:

  ./internal/server.New       internal/server/server.go:18
  ./internal/config.New       internal/config/config.go:12
  ./internal/store.New        internal/store/store.go:27

Try:

  gosherpa refs ./internal/server.New
```

Done when:

- Ambiguity becomes a helpful path forward instead of a dead end.

## Phase 3: Interfaces and Implementations

Goal: answer the most Go-specific architecture questions.

### 3.1 Implementers of an Interface

Human question:

```text
Who implements this interface?
```

Command sketch:

```bash
gosherpa implements UserRepository
gosherpa implements ./internal/user.UserRepository
```

MVP behavior:

- Resolve interface type.
- Find named types whose method sets satisfy it.
- Show direct implementers.
- Include pointer/value receiver information.
- Indicate whether implementation is explicit only by usage or structurally
  valid in the package graph.

Done when:

- Developers can understand interface boundaries without manual method matching.

### 3.2 Interfaces Satisfied by a Type

Human question:

```text
Which interfaces does this type satisfy?
```

Command sketch:

```bash
gosherpa interfaces UserStore
gosherpa interfaces ./internal/postgres.UserStore
```

MVP behavior:

- Resolve named type.
- Find local interfaces satisfied by that type.
- Show interface package and method count.
- Distinguish value receiver and pointer receiver satisfaction.

Done when:

- A developer can discover architectural contracts around a concrete type.

### 3.3 Interface Method Usage

Human question:

```text
Which methods of this interface are actually used?
```

Command sketch:

```bash
gosherpa interface UserRepository
```

MVP behavior:

- Show interface methods.
- Show implementers.
- Show references to the interface type.
- Show call sites for each interface method when type information can resolve
  them.

Done when:

- GoSherpa can help identify overly broad or unused interfaces.

## Phase 4: Call Graph and Paths

Goal: help developers follow execution flow.

### 4.1 Improve Callers and Callees

Human question:

```text
Who calls this, and what does it call?
```

Command sketch:

```bash
gosherpa callers UserService.Create
gosherpa callees Handler.CreateUser
```

Improvements over current MVP:

- Resolve package-qualified function calls.
- Resolve receiver-variable method calls.
- Include test callers with `--tests`.
- Include or exclude external package calls.
- Group output by package.
- Show call site line and enclosing function.

Done when:

- Callers and callees are type-aware for ordinary Go code.
- Known limitations are documented for dynamic dispatch, reflection, and
  function values.

### 4.2 Call Paths

Status: MVP implemented with a syntactic repository-local call graph.

Human question:

```text
How can execution get from A to B?
```

Command sketch:

```bash
gosherpa path Handler.ServeHTTP UserService.Create
gosherpa paths Handler.ServeHTTP UserService.Create --limit 5
gosherpa paths UserService.Create --from-entrypoints
```

MVP behavior:

- Build a repository-local call graph.
- Find shortest path between two symbols.
- Support max depth.
- Support result limit.
- Show each step as caller -> callee with file locations.

Example:

```text
CALL PATH

Handler.ServeHTTP
  -> Handler.CreateUser          internal/http/handler.go:42
  -> UserService.Create          internal/user/service.go:31
```

Done when:

- A developer can understand how a function is reached without opening many
  files manually.

### 4.3 Entrypoints

Human question:

```text
What public or runtime entrypoints can lead here?
```

Command sketch:

```bash
gosherpa entrypoints UserService.Create
```

Potential entrypoint kinds:

- `main.main`
- HTTP handlers.
- CLI command handlers.
- test functions.
- exported functions.
- goroutine starts.

MVP:

- Start with `main.main`, tests, exported functions, and functions with no local
  callers.
- Avoid framework-specific inference until there is a clear need.

Done when:

- Developers can work backward from an internal function to likely starting
  points.

### 4.4 Dynamic Call Limitations

Human question:

```text
Can GoSherpa tell me when the graph may be incomplete?
```

Requirements:

- Detect and report common uncertainty sources:
  - interface dispatch
  - function values
  - reflection
  - generated code
  - build tags
  - cgo
- Prefer honest partial answers over false certainty.

Done when:

- Output includes a short limitations note when relevant.

## Phase 5: Tests and Impact

Status: direct impact MVP, test discovery MVP, and PRD v0.1 Impact Engine MVP
implemented for symbols, packages, files, and diffs.

Goal: help developers make changes with confidence.

### 5.1 Test Discovery

Status: MVP implemented with same-package tests, external `_test` packages,
direct symbol references, and suggested `go test` commands.

Human question:

```text
Which tests are related to this symbol or package?
```

Command sketch:

```bash
gosherpa tests UserService.Create
gosherpa tests ./internal/user
gosherpa tests internal/user/service.go
```

MVP behavior:

- Find tests in the same package.
- Find tests in external `_test` packages.
- Find tests that reference the target symbol.
- Suggest exact `go test` commands.

Later behavior:

- Accept file targets.
- Find table-test names when possible.
- Use type-aware reference matching inside test files.

Example:

```text
TESTS

Target: UserService.Create

Direct references:

  TestUserService_Create          internal/user/service_test.go:18
  TestCreateRejectsEmptyEmail     internal/user/service_test.go:64

Suggested command:

  go test ./internal/user
```

Done when:

- Developers get a practical test command, not just a list of files.

### 5.2 Impact Analysis

Human question:

```text
What might be affected if I change this?
```

Command sketch:

```bash
gosherpa impact UserService.Create
gosherpa impact ./internal/user
gosherpa impact internal/user/service.go
```

MVP behavior:

- Accept symbol and package targets.
- Show direct references and caller-chain impact for symbols.
- Show direct dependent packages for packages.
- Show affected local packages.
- Show related tests and suggested `go test` commands.

Later behavior:

- Accept file targets.
- Expose configurable caller-depth controls.
- Show exported API impact when target is exported.

Useful output groups:

- Target summary.
- Direct code impact.
- Package impact.
- Test impact.
- Confidence and limitations.

Done when:

- A developer can use the output to decide what to inspect and test next.

### 5.3 Impact Engine v0.1

Status: implemented from [PRD_V01.md](PRD_V01.md). `internal/git.ChangedFiles`,
`internal/impact.ChangedPackages`, and `internal/impact.AnalyzeDiff` are
implemented. `internal/impact.ChangedSymbols` extracts changed and deleted
symbols from git diff hunks. `gosherpa impact diff --base <ref>` and `gosherpa
tests affected --base <ref>` are implemented for human and JSON output.
`gosherpa impact file|package|symbol` is implemented for human and JSON output.
Interface and implementer impact signals are implemented as a conservative
local method-set scan with import-aware signature matching and embedded-interface
expansion. Package-qualified symbol impact disambiguates references and affected
tests. Symbol impact includes transitive callers in affected packages.
Affected-test planning includes package tests for affected caller packages.

Human question:

```text
If I change this file, package, symbol, or diff, what is affected?
```

Command sketch:

```bash
gosherpa impact file internal/auth/session.go
gosherpa impact diff --base origin/main
gosherpa impact symbol ./internal/auth.Session
gosherpa impact package ./internal/auth
gosherpa tests affected --base origin/main
```

MVP behavior:

- Read changed files from git diffs. Implemented.
- Map changed files to packages. Implemented for Go files.
- Report changed packages and affected dependent packages. Implemented
  for diff reports.
- Report changed and deleted symbols from git diff hunks. Implemented
  for top-level Go functions and struct/interface types.
- Report affected tests at package granularity. Implemented for affected
  packages.
- Report affected interfaces and implementations. Implemented with
  import-aware method signature matching and embedded-interface expansion.
- Disambiguate package-qualified symbol impact for references and affected
  tests. Implemented.
- Include transitive caller packages for symbol impact. Implemented.
- Include package tests for affected caller packages. Implemented.
- Preserve the existing JSON response discipline for new commands. Implemented
  for `impact diff`, `tests affected`, and `impact file|package|symbol`.

Architecture:

- `internal/git` reads diffs, changed files, changed hunk line ranges, and file
  contents at refs; it knows no Go semantics. `ChangedFiles`, hunk range
  parsing, and `FileAtRef` are implemented.
- `internal/index` builds repository graphs; it knows no Git semantics.
- `internal/impact` consumes index data and produces `ImpactReport`.
  `ChangedPackages` maps changed Go files to local package paths first, and
  `ChangedSymbols` maps hunk ranges to current-file Go symbols plus deleted
  symbols from base-file ranges. `AnalyzeDiff`, `AnalyzeFile`,
  `AnalyzePackage`, and `AnalyzeSymbol` produce the first
  package/test/symbol/interface/implementation impact reports. Package-qualified
  symbol targets are normalized before reference and test impact collection, and
  symbol impact walks caller chains for affected-package impact.
- Test discovery remains separate and package-oriented for v0.1.

Done when:

- `gosherpa impact diff --base <ref>` reports changed packages, affected
  packages, and affected tests. Implemented.
- `gosherpa tests affected --base <ref>` prints suggested `go test` commands.
  Implemented.
- `gosherpa impact file|package|symbol` reports package/test impact through
  `ImpactReport`. Implemented.
- `gosherpa impact diff --base <ref>` reports affected symbols from changed
  hunks, including deleted top-level symbols read from the base ref.
  Implemented.
- Affected interfaces and implementations are populated for changed packages
  and symbol targets. Implemented with import-aware signature
  matching and embedded-interface expansion.
- Package-qualified symbol targets avoid same-name reference/test bleed from
  other packages. Implemented.
- Symbol impact reports transitive callers in affected packages. Implemented.
- Symbol impact suggests package tests for transitive caller packages.
  Implemented.
- JSON and human output are covered by focused tests and golden fixtures for
  diff impact, affected tests, and file/package/symbol impact.

### 5.4 Change Risk Summary

Human question:

```text
Is this a local change or a wide change?
```

Command sketch:

```bash
gosherpa impact UserService.Create --summary
```

MVP behavior:

- Provide a simple qualitative summary:
  - local
  - package-level
  - cross-package
  - exported API
  - high fan-in
- Include the facts behind the summary.

Done when:

- Risk wording is useful but not overconfident.
- The summary never hides the raw evidence.

## Phase 6: Package and Architecture Navigation

Goal: help developers understand repository structure beyond individual symbols.

### 6.1 Package List

Human question:

```text
What packages are in this repository?
```

Command sketch:

```bash
gosherpa packages
gosherpa packages --tests
```

MVP behavior:

- List local packages.
- Show file count.
- Show symbol count.
- Show import count.
- Show imported-by count.
- Mark packages with tests.

Done when:

- New contributors can get a compact map of the project.

### 6.2 Dependency Graph

Human question:

```text
How do packages depend on each other?
```

Command sketch:

```bash
gosherpa deps ./internal/user
gosherpa deps --all
gosherpa deps --graph
```

MVP behavior:

- Improve current `deps` output.
- Support all-package dependency overview.
- Distinguish local imports from external imports.
- Show reverse dependencies.
- Support depth-limited traversal.

Done when:

- Package-level architecture can be inspected without reading imports manually.

### 6.3 Cycles and Layering Signals

Human question:

```text
Is the package structure healthy?
```

Command sketch:

```bash
gosherpa cycles
gosherpa packages --fan-in
gosherpa packages --fan-out
```

MVP behavior:

- Report local dependency cycles if any.
- Show high fan-in packages.
- Show high fan-out packages.
- Show leaf packages.

Done when:

- GoSherpa can surface package structure smells without becoming a linter.

## Phase 7: Interactive Navigation

Goal: make larger explorations comfortable for humans.

### 7.1 Terminal UI

Human question:

```text
Can I explore this repo without repeatedly typing commands?
```

Command sketch:

```bash
gosherpa tui
```

MVP behavior:

- Search symbols.
- View symbol details.
- View references.
- View callers and callees.
- View package dependencies.
- Open selected file in editor command if configured.

Design constraints:

- Keep the TUI optional.
- Keep command-line behavior first-class.
- Reuse the same analysis engine.

Done when:

- The TUI is useful for browsing but does not become required for core usage.

### 7.2 Shell Completion

Human question:

```text
Can the CLI help me type valid commands?
```

Command sketch:

```bash
gosherpa completion zsh
gosherpa completion bash
gosherpa completion fish
```

MVP behavior:

- Complete command names and flags.
- Optionally complete package paths.
- Symbol completion can come later if indexing is fast enough.

Done when:

- Common command usage becomes easier without adding conceptual complexity.

### 7.3 Editor Integration Hooks

Human question:

```text
Can GoSherpa open the result where I work?
```

Command sketch:

```bash
gosherpa symbol UserService.Create --open
gosherpa refs UserService.Create --editor code
```

MVP behavior:

- Print editor-friendly `file:line:column` locations first.
- Add optional `--open` later.
- Support editor command configuration through environment variables.

Done when:

- GoSherpa remains editor-agnostic but easy to connect to editors.

## Phase 8: Machine-Readable and Agent-Friendly Surfaces

Goal: expose the same reliable code intelligence to scripts and agents.

This phase should come after the human concepts are stable enough that schemas
will not churn constantly.

### 8.1 JSON Output

Status: JSON MVP implemented for all commands with a versioned response
envelope and golden fixtures.

Human and agent question:

```text
Can another tool consume this answer safely?
```

Command sketch:

```bash
gosherpa callers UserService.Create --json
gosherpa impact ./internal/user --json
```

Requirements:

- Extend `--json` to all commands. Implemented.
- Define stable schemas per command.
- Include schema version. Implemented for the current JSON commands.
- Include command metadata. Implemented for the current JSON commands:
  - root
  - module path
  - target
  - include tests
  - analysis mode
  - warnings
- Keep stdout pure JSON on success.
- Print non-JSON diagnostics to stderr.

Done when:

- Current JSON commands are covered by CLI tests.
- JSON responses use one envelope shape with `schemaVersion`, `command`,
  `target`, `root`, `modulePath`, `warnings`, and `data`.
- JSON output is covered by golden fixture tests.
- Human text output and JSON output share the same analysis result.

### 8.2 Batch Queries

Human and agent question:

```text
Can I ask several related questions without rescanning the repository each time?
```

Command sketch:

```bash
gosherpa query --json < queries.json
```

MVP behavior:

- Accept a list of simple operations.
- Build or load the repository index once.
- Return ordered results.

Done when:

- Repeated analysis becomes faster for scripts and future integrations.

### 8.3 MCP or Tool Server

Agent question:

```text
Can an agent call GoSherpa as a structured code-intelligence tool?
```

Command sketch:

```bash
gosherpa serve --mcp
```

Potential tools:

- `symbols`
- `symbol`
- `search`
- `refs`
- `callers`
- `callees`
- `path`
- `deps`
- `implements`
- `tests`
- `impact`

Done when:

- The server is a thin wrapper around the same stable analysis APIs.
- Tool responses are bounded, deterministic, and include warnings.

## Cross-Cutting Technical Work

### Semantic Index

The roadmap becomes much easier once GoSherpa has a shared repository index.

Candidate model:

```text
RepositoryIndex
  Module
  Packages
  Files
  Symbols
  References
  Calls
  Implementations
  Tests
```

Initial goals:

- Build once per command.
- Keep package loading deterministic.
- Represent all source locations with file, line, column, and optional end
  position.
- Preserve enough type information to resolve references and method calls.

Later goals:

- Cache between commands.
- Incremental rebuilds.
- Watch mode for TUI or daemon usage.

### Symbol Identity

Name-only targets are convenient but become ambiguous quickly.

Support levels:

1. Short names:

   ```text
   ParseFile
   UserService.Create
   ```

2. Package-qualified names:

   ```text
   ./internal/sherpa.ParseFile
   ./internal/user.UserService.Create
   ```

3. Fully stable internal IDs:

   ```text
   github.com/supertabaluga/gosherpa/internal/sherpa.ParseFile
   ```

Recommendation:

- Keep short names for convenience.
- Show clear ambiguity errors.
- Prefer package-qualified examples in documentation once ambiguity appears.

### Build Tags and Package Loading

Go repositories often depend on build tags, generated files, tests, and cgo.

Requirements:

- Add `--tags`.
- Add `--tests`.
- Report package load errors clearly.
- Avoid silently ignoring packages that failed to load.

Command sketch:

```bash
gosherpa refs FeatureFlag --tags integration
gosherpa callers Handler.ServeHTTP --tests
```

### Performance

Human CLI tools need to feel instant for small repositories and acceptable for
medium ones.

Targets:

- Small repo: most commands complete in under 300 ms.
- Medium repo: common commands complete in under 2 seconds.
- Large repo: provide progress or caching before commands feel broken.

Implementation options:

- Parse only what a command needs in early phases.
- Move to shared indexing when semantic features expand.
- Add optional cache after correctness is solid.

### Testing Strategy

Use tests to protect the trustworthiness of code intelligence.

Test layers:

- Unit tests for normalization, formatting, and symbol identity.
- Fixture-based tests for references, calls, interfaces, and packages.
- Golden output tests for human-readable command output.
- JSON schema tests once JSON exists.
- Integration tests for CLI behavior and exit codes.

Important fixtures:

- Same symbol name in multiple packages.
- Methods with pointer and value receivers.
- Interface satisfaction across packages.
- Import aliases.
- Tests in same package and external `_test` package.
- Build tags.
- Generated files.
- Function values.
- Interface dispatch.

### Documentation

Docs should remain example-driven.

Needed docs:

- README quick start.
- Command reference.
- Roadmap.
- Known limitations.
- Examples using a small fixture project.
- Explanation of syntactic vs semantic analysis.

## Suggested Release Milestones

### v0.1: Current MVP

Theme: basic repository visibility.

Included:

- symbols
- symbol
- refs
- deps
- callers
- callees

### v0.2: CLI Polish and Search

Theme: make the current tool pleasant.

Included:

- global `--root`
- consistent output
- stable exit behavior
- `search`
- symbol filters
- context output
- richer symbol details

### v0.3: Semantic References

Theme: make references trustworthy.

Included:

- `go/packages` loader
- type-aware `refs`
- package-qualified symbol targets
- ambiguity handling
- test inclusion flag

### v0.4: Interfaces

Theme: answer core Go architecture questions.

Included:

- `implements`
- `interfaces`
- richer interface details
- interface method usage where practical

### v0.5: Call Graph

Theme: follow execution flow.

Included:

- type-aware callers
- type-aware callees
- `path`
- `entrypoints`
- documented uncertainty for dynamic dispatch

### v0.6: Tests and Impact

Theme: plan safer changes.

Included:

- `tests`
- `impact`
- suggested `go test` commands
- change risk summary

### v0.7: Architecture Navigation

Theme: understand the repository as a system.

Included:

- `packages`
- improved `deps`
- `cycles`
- fan-in and fan-out summaries

### v0.8: Interactive UX

Theme: make exploration comfortable.

Included:

- optional TUI
- shell completion
- editor integration hooks

### v1.0: Stable Human Product

Theme: dependable daily-use tool.

Criteria:

- Core commands are stable.
- Output is documented.
- Known limitations are honest and visible.
- Semantic analysis handles ordinary Go projects well.
- The tool is useful without configuration.
- Tests cover realistic Go patterns.

### v1.1: Machine-Readable Surfaces

Theme: make GoSherpa excellent for scripts and agents.

Included:

- stable schemas
- batch query mode
- possible MCP server

## Feature Priority Matrix

| Feature | Human value | Implementation risk | Recommended priority |
| --- | --- | --- | --- |
| `--root` | High | Low | Immediate |
| Consistent output | High | Low | Immediate |
| `search` | High | Low | Immediate |
| Rich `symbol` details | High | Medium | Near-term |
| Type-aware `refs` | Very high | High | Near-term |
| `implements` | Very high | Medium | Near-term |
| Type-aware callers/callees | Very high | High | Mid-term |
| `path` | High | High | Mid-term |
| `tests` | Very high | Medium | Mid-term |
| `impact` | Very high | High | Mid-term |
| `packages` | Medium | Low | Mid-term |
| `cycles` | Medium | Low | Later |
| TUI | Medium | Medium | Later |
| JSON | High for tools | Low after result model | In progress |
| MCP server | High for agents | Medium | Later |

## Near-Term Implementation Plan

If the next goal is to make GoSherpa noticeably better for humans, the best
sequence is:

1. Add global command plumbing.
   - Introduce shared options.
   - Add `--root`.
   - Add usage tests.

2. Normalize output.
   - Add formatting helpers.
   - Move direct printing out of analysis.
   - Add golden tests for current commands.

3. Add `search`.
   - Reuse current symbol parsing.
   - Add kind and package filters.
   - Add clear ambiguity-friendly output.

4. Improve `symbol`.
   - Add signatures.
   - Add package names.
   - Add receiver information.
   - Add context output.

5. Introduce a semantic package loader.
   - Start behind internal APIs.
   - Keep existing behavior until semantic references are ready.
   - Add fixtures for confusing reference cases.

6. Replace `refs` with semantic references.
   - Preserve a simple human output shape.
   - Add reference kinds incrementally.

7. Add `implements`.
   - Reuse type information from the semantic loader.
   - Keep output compact and concrete.

This path improves the daily human experience quickly while preparing the
deeper features.

## Non-Goals for Now

Avoid these until the core navigation experience is strong:

- Full IDE replacement.
- Whole-program pointer analysis.
- Framework-specific route discovery.
- Automatic refactoring.
- Code generation.
- Lint rule collections.
- Mandatory daemon mode.
- Project-specific configuration as a requirement for basic use.

## Definition of a Great Human-First GoSherpa

GoSherpa is doing its job when a developer can enter an unfamiliar Go repo and,
within a few commands, answer:

- What are the main packages?
- Where is this concept implemented?
- Who calls this function?
- What does this function call?
- Which concrete types implement this interface?
- Which tests give me confidence before I change it?
- Is this change likely local, package-wide, or cross-cutting?

The best version of GoSherpa feels like a calm senior engineer sitting next to
the terminal: concise, accurate, and useful exactly when the codebase gets too
large to hold in your head.
