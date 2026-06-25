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
- Direct symbol and package impact analysis.
- Related test discovery with suggested `go test` commands.
- Package dependency analysis.
- Direct syntactic callee analysis.
- Direct syntactic caller analysis.
- Shortest and limited repository-local call path analysis.
- Repository root selection with `--root`.

Current limitations:

- References are type-aware inside packages and recognize local package selector
  calls, but do not yet use full module/package loading.
- Impact analysis is direct-only and does not yet include transitive callers.
- Test discovery uses same-package tests and syntactic direct-reference
  matching; table-test names are not extracted yet.
- Callers and callees are AST-based and can miss receiver-variable method calls.
- Call paths inherit the current AST-based caller and callee limitations.
- Function names can be ambiguous across packages.
- Output is optimized for reading, but not yet consistently structured.
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

Human question:

```text
Did the command succeed, find nothing, or fail?
```

Requirements:

- Exit `0` for successful command execution.
- Exit non-zero for invalid usage, parse errors, missing repository roots, and
  ambiguous targets.
- Decide whether "not found" should be exit `0` with an empty state or a
  distinct non-zero status. For human-first UX, prefer exit `0` when the command
  ran successfully and found no matches.

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

Status: impact MVP and test discovery MVP implemented for symbols and packages.

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
- Show direct references and direct callers for symbols.
- Show direct dependent packages for packages.
- Show affected local packages.
- Show related tests and suggested `go test` commands.

Later behavior:

- Accept file targets.
- Show transitive callers up to a default depth.
- Show exported API impact when target is exported.

Useful output groups:

- Target summary.
- Direct code impact.
- Package impact.
- Test impact.
- Confidence and limitations.

Done when:

- A developer can use the output to decide what to inspect and test next.

### 5.3 Change Risk Summary

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

- Add `--json` to all commands.
- Define stable schemas per command.
- Include schema version.
- Include command metadata:
  - root
  - module path
  - target
  - include tests
  - analysis mode
  - warnings
- Keep stdout pure JSON on success.
- Print non-JSON diagnostics to stderr.

Done when:

- JSON output can be tested with golden files.
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

- `--json`
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
| JSON | High for tools | Low after result model | Later |
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
