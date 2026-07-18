# GoSherpa Status

GoSherpa is an early MVP. The Impact Engine v0.1 track from [PRD_V01.md](product/PRD_V01.md) is complete; current work continues the Symbol Intelligence track from [PRD_V02.md](product/PRD_V02.md).

Read the full product plan in [FEATURE_ROADMAP.md](product/FEATURE_ROADMAP.md), or start from the [docs index](README.md).

## Roadmap

| Next | Goal |
| --- | --- |
| Agent Workflow Robustness | Harden `gosherpa agent context --base <base-ref>` across real Go repository shapes, size limits, snapshot ergonomics, tests, and entrypoint signals |

## Repository Shape Support

| Repository shape | Status | Notes |
| --- | --- | --- |
| Single module with `go.mod` | Supported | File walking, package loading, symbols, references, context, impact, tests, snapshots, and `agent context` share the selected module boundary. |
| Workspace root with `go.work` | Supported | Workspace modules under `--root` are loaded as repository packages. Workspace modules outside `--root` are reported but not scanned as repository packages for that invocation. |
| Module root with parent or local `go.work` | Supported with boundary reporting | `doctor` and `agent context` report the visible workspace and whether the selected boundary is a module, workspace, or module inside a parent workspace. |
| Nested modules below the selected root | Intentionally separate | Nested modules are skipped by root-level analysis unless included by the selected `go.work`. Inspect them with `gosherpa --root <nested-module> ...`. |
| Local `replace` directives | Reported boundary evidence | Local replacements are listed with owner module, replacement path, and whether the path is inside `--root`. Replacement roots are not silently folded into the selected module unless the selected workspace includes them. |
| Build tags | Supported with explicit inputs | Use `--tags <list>` on supported commands. Build tags are normalized in readiness output and snapshot compatibility; tag changes make snapshots stale. |
| Generated Go files | Supported and visible | Compiler-visible files with the standard `// Code generated ... DO NOT EDIT.` marker are included in analysis and counted in readiness. Large generated reading-order entries may be summarized after hand-written files in `agent context`. |
| Partial package loading | Partially supported | Useful AST and available typechecked data are still returned. `load-error`, `parse-error`, and `type-error` diagnostics reduce confidence and list affected analysis sections. |
| Large repositories and large diffs | Supported with guardrails | Use snapshots and `--max-*` or `--max-bytes` limits. `doctor` and `agent context` expose cost counts so output size and stale inventory risk are visible. |

## Implemented

- Repository scanning and Go file discovery
- Struct, interface, type alias, function, and method discovery
- Initial shared repository index v0 for typechecked package, file, and symbol inventory used by symbol identity and context exports
- First in-memory semantic context slice reuses typechecked repository loads across symbol identity, references, call signals, direct test-reference analysis, file/package context symbol inventory, and context interface-impact signals for `explain`, `context symbol`, `context file`, `context package`, symbol impact, and changed-symbol impact paths
- Symbol listing filters by kind, package, and test symbols
- Rich symbol lookup with package paths, signatures, doc comments, struct fields, interface methods, and Go-aware reference lookup
- Typechecked reference analysis via `go/packages`, with an AST/per-package fallback when semantic loading is unavailable
- Ranked `gosherpa search <terms>` for partial, case-insensitive symbol discovery with kind, package, test, and limit filters
- Initial `gosherpa analyze [path]` repository overview with package summaries, symbol counts, important symbols, entrypoint candidates, structural risk, hotspots, testing overview, readiness, limitations, suggested next commands, and JSON output
- Initial `gosherpa architecture` overview with dependency cycles, most coupled packages, high fan-in/fan-out packages, largest packages, leaf packages, limitations, and JSON output
- Initial `gosherpa risk` overview with structural risk level, score, factors, package signals, dependency cycles, limitations, and JSON output
- Initial `gosherpa explain <symbol>` profile with purpose, risk, target risk, architecture role, reading order, human output, and JSON output
- Initial `gosherpa context symbol <target>` export with source excerpt, symbol identity, references, callers, callees, impact signals, tests, target risk, confidence, limitations, and JSON output
- Initial `gosherpa context file <file>` export with file symbols, source excerpts, affected packages, affected tests, target risk, reading order, confidence, limitations, and JSON output
- Initial `gosherpa context package <package>` export with package files, symbols, source excerpts, affected packages, affected tests, target risk, reading order, confidence, limitations, and JSON output
- Initial `gosherpa context diff --base <ref>` export with changed files, changed packages, changed symbols, affected packages, affected tests, target risk, reading order, confidence, limitations, and JSON output
- Initial `gosherpa agent context --base <ref>` workflow with readiness, snapshot status, cost counts, changed targets, target risk, possible runtime relationship summary, interface summary, affected tests, section modes, suggested commands, short human output, JSON output, and composed `--max-bytes` budgeting with per-section truncation metadata
- Context export size controls with entry-count limits, source radius limits, and `--max-bytes` byte-budget truncation
- Initial `gosherpa doctor` readiness report with module, Go environment, workspace, build tag, package loading, structured package-load diagnostics, snapshot status, bounded relationship snapshot metadata, confidence, limitations, warnings, and JSON output
- Shared repository layout summary for `doctor` and `agent context` that reports the selected analysis boundary, root or parent `go.work`, workspace modules, skipped nested modules, skipped workspace modules outside `--root`, local replace directives, file counts, generated-file counts, and major generated packages
- `doctor` and `agent context` expose normalized build tags plus structured package-load diagnostics that distinguish load, parse, and type errors with package, file/position when known, reason, message, and affected analysis sections; existing envelope warnings remain the compatibility surface for warning text
- `doctor` and `agent context` expose a compact `cost` summary with package, Go file, test file, generated-file, skipped-module, local-replacement, package-load-warning, diff target, affected target, symbol, and relationship counts. Full symbol and relationship inventory counts come from readable snapshot metadata, with explicit limitations when snapshots are missing, stale, or invalid.
- Initial `gosherpa snapshot` command that writes a versioned `.gosherpa/snapshot.json` repository inventory snapshot with file freshness metadata, package summaries, symbols, build tags, git state, and relationship-capability metadata
- Static shell completion script generation with `gosherpa completion zsh|bash|fish`
- Package-aware caller/callee signals for package-qualified `gosherpa explain` targets
- Direct symbol and package impact analysis
- Related test discovery for symbols, packages, and files with suggested `go test` commands
- Package dependency analysis
- Direct caller and callee analysis with package-aware targets and receiver-variable method calls
- Shortest and limited repository-local call path analysis
- Dynamic call uncertainty limitations for caller, callee, and call-path outputs, including interface dispatch, function values, reflection, goroutine starts, function literal calls, and imported receiver boundaries; `callers --json` and `callees --json` also expose separate `possibleCalls` arrays for bounded possible runtime call signals without changing direct call counts
- Typechecked interface-dispatch possible calls resolve selector calls on interface-typed values to known local implementer methods when the candidate set is bounded; unknown, broad, or unsupported dispatch remains a limitation instead of a direct call edge
- Runtime possible calls now name visible local targets for direct goroutine starts, goroutine function literals, immediately invoked function literals, and function literals passed to simple local call sites; reassigned, struct-field, or escaping function values remain conservative limitations or dynamic possible calls
- Initial `gosherpa entrypoints <target>` analysis for `main.main`, test functions with `--tests`, statically visible stdlib `net/http` handlers, exported functions, and functions with no local callers
- Package-aware standalone call graph commands for package-qualified targets
- Type alias symbol discovery, including alias signatures and typechecked `context symbol` / `impact symbol` coverage
- Receiver-variable method calls in standalone call graph commands, resolved with package-level type information
- Simple locally assigned function values and concrete method values resolved as caller, callee, and call-path edges when type information proves a single static target
- Standalone interface navigation with `implementers`, `interfaces`, and `interface` profiles for methods, implementers, type references, and visible method usage
- Opt-in test callers for `gosherpa callers --tests` and `gosherpa explain --tests`
- Ambiguity errors for duplicate unqualified symbols include candidate packages, files, lines, and package-qualified examples
- Machine-readable JSON output for all commands with a stable response envelope
- Golden JSON fixtures for all commands
- Repository root selection with `--root`
- Stable CLI exit codes with diagnostics on stderr
- Git changed-file discovery foundation via `internal/git.ChangedFiles`
- Changed-package mapping for git diffs via `internal/impact.ChangedPackages`
- Diff impact report foundation via `internal/impact.AnalyzeDiff`
- `gosherpa impact diff --base <ref>` with human and JSON output
- `gosherpa tests affected --base <ref>` with human and JSON output
- `gosherpa pr --base <ref>` with human and JSON output for PR-style changed files, packages, symbols, diff risk notes, target risk, structural repository risk, affected tests, and verification commands
- Diff-oriented reports enrich changed top-level symbols with typechecked reference and call impact when package loading is available, exposing `git-diff+typechecked+ast`, `referenceAnalysisMode`, and `callAnalysisMode`
- `gosherpa impact file|package|symbol` with human and JSON output
- Interface and implementer impact signals based on local method sets with import-aware signature matching and embedded-interface expansion
- Changed-symbol extraction from git diff hunks, including deleted symbols from base files
- Package-qualified symbol impact for references and affected tests
- Transitive caller impact for symbol changes
- Affected-test planning for transitive caller packages
- Related test entries expose stable relationship `reasons` and concrete
  `targets` where the symbol or changed-symbol relationship is known; structured
  test plan items also expose concrete `tests` arrays and `targets`.
- Structured test plans expose plan-level `confidence` and `limitations` plus
  per-command `category` and item `confidence`. Categories distinguish
  focused, fast, contract, caller-package, integration-like, and broad-fallback
  recommendations.
- Reusable test inventory records test packages, `_test.go` files, top-level
  test functions, statically visible literal subtests, conservative
  suite-like `Test*` methods, source ranges, and inventory limitations; current
  public test workflows use this inventory without changing snapshot format.
- Direct related and affected tests expose `targetReferences` with target names,
  positions, and exact occurrence ranges when parser or typechecker positions
  identify the referenced symbol.
- Structured test plans include a `contracts` group for interface or
  implementation contract packages, keeping those recommendations separate from
  direct, related, caller-package, and fallback test commands.
- Diff test planning distinguishes package-level fallback from
  whole-repository fallback, using `go test ./...` when changed files cannot be
  narrowed to repository-local Go packages.
- Generated packages, skipped module/package boundaries, and package-load or
  repository-shape warnings lower test-plan confidence and are surfaced as
  limitations in tests, impact, context, PR, and agent workflow JSON.
- Agent-facing context, impact, PR, and affected-test JSON limitations include
  subanalysis notes for reference, call, interface, and test analysis modes
  where those modes are present.
- Impact, context, explain, and PR JSON expose additive `targetRisk` summaries
  with deterministic levels, scopes, reason categories, structured evidence
  signals, and limitations; repository structural risk remains separate.
- Package inventory with `gosherpa packages`, including file, symbol, import, reverse-dependency, and test indicators
- All-package dependency overview with `gosherpa deps --all`, including local imports, external imports, and reverse dependencies
- Reference kind classification and `gosherpa refs --kind <kind>` filtering,
  including read/write classification for value and field references
- Source ranges with columns for symbols, references, callers, callees, call paths, related tests, direct test target references, and range-backed reading-order entries in JSON output
- Opt-in snapshot reuse for inventory commands through `--use-snapshot` on `analyze`, `symbols`, `symbol`, `search`, and test-inclusive `packages --tests`; standalone relationship commands `refs`, `callers`, `callees`, `implementers`, `interface`, and `interfaces`, plus `context symbol` and `impact symbol`, can reuse valid relationship records; diff-oriented commands `context diff`, `impact diff`, `tests affected`, and `pr` can reuse valid snapshot symbols for current changed-symbol inventory and selected relationship subanalyses. Missing, stale, invalid, or relationship-incompatible snapshots fall back to live analysis with warnings.
- `agent context --base <ref> --use-snapshot` can reuse valid snapshot-backed diff context and relationship summaries while keeping focused symbol, file, and package drill-down in the existing `context` commands.

## Known MVP Limitations

- Reference search uses `go/packages`-backed typechecked loading when available and falls back to AST/per-package type information; caller, callee, path, and interface impact analysis remain conservative and mostly local.
- Test files are skipped by reference, callee, path, default caller, and default entrypoint analysis; `callers --tests`, `entrypoints --tests`, and `explain --tests` include test-file callers on demand.
- Symbol impact includes transitive callers and package tests for affected caller packages.
- Diff impact is hunk-based; it reports directly changed or deleted top-level Go functions and struct/interface types, enriches current changed symbols with typechecked reference/call impact when available, but it does not infer every semantic consequence of changed statements.
- Package-qualified symbol impact disambiguates references and affected tests; unqualified symbol targets may require disambiguation across packages.
- Interface implementer impact canonicalizes local/external import paths in method signatures and resolves local embedded interfaces, but generated-file, build-tag, and generic edge cases may remain incomplete.
- Interface method usage reports statically visible selector usage for interface-typed values when typechecked package loading succeeds; dynamic dispatch, reflection, and runtime wiring can hide additional usage.
- Test discovery uses direct references, same-package tests, file-contained symbols, and literal `t.Run` subtest names; dynamic table-driven names may be incomplete.
- Caller, callee, path, and entrypoint analysis still do not resolve dynamic dispatch, reflection, reassigned or escaping function values, or dependency internals; caller and callee JSON outputs surface bounded imported receiver calls as external `possibleCalls` when typechecked selector data can name the receiver method.
- Entrypoint analysis is heuristic; statically visible stdlib `net/http` handler registrations are inferred, but framework-specific routers, custom runtime wiring, and CLI command handlers are not inferred yet.
- Context export currently supports symbol, file, package, and diff targets. The top-level `agent context` workflow is diff-first only and rejects symbol/file/package positional targets until later plans define those modes.
- Nested modules below a module root are treated as separate analysis roots and
  are skipped by root-level file walking, symbol lookup, references, callers,
  context, impact, and test discovery unless they are included through an
  explicit `go.work` workspace or inspected with their own `--root`. `doctor`
  and `agent context` report skipped nested modules and suggest separate
  `--root` inspection.
- `go.work` modules outside the selected `--root` can affect workspace module
  resolution, but they are not scanned as repository packages for that
  invocation. Use a separate `--root` for external workspace modules that are
  part of the change.
- Local `replace` directives are reported as repository layout evidence.
  Replacement modules under the root but outside the selected module/workspace
  boundary remain separate analysis roots; external replacements may affect
  typechecking but are not walked as repository packages.
- Repository-local generated Go files are included when they are visible to the
  selected module or workspace. `doctor` and `agent context` detect the
  standard `// Code generated ... DO NOT EDIT.` header for reporting generated
  counts and major generated packages; relationship and context commands
  analyze generated files like other compiler-visible Go files. The agent
  workflow summarizes large generated file reading-order entries after
  hand-written files when generated files would otherwise dominate the first
  reading pass.
- Snapshot creation and stale/missing/valid diagnostics are implemented with format v2 relationship-capability metadata, bounded `doctor`/`snapshot --json` relationship counts, inventory reuse for `analyze`, `symbols`, `symbol`, `search`, `packages --tests`, relationship reuse for `refs`, `callers`, `callees`, `implementers`, `interface`, `interfaces`, `context symbol`, `impact symbol`, selected diff-oriented relationship subanalysis in `context diff`, `impact diff`, `tests affected`, `pr`, and the first `agent context` workflow; unsupported portions fall back to live analysis with warnings.
- Repeated large-repo workflows should refresh snapshots with `gosherpa snapshot --json` before relying on snapshot inventory counts, after build-tag changes, and whenever `doctor`, `agent context`, or envelope warnings report missing, stale, invalid, or relationship-limited snapshots.
- The shared repository index v0 currently covers package, file, symbol inventory, and an in-memory relationship-index contract. A first in-memory semantic context shares typechecked loads for symbol identity, references, calls, direct test-reference analysis, file/package context inventory, and context interface-impact signals; persisted relationship records now back selected standalone and must-use workflow relationship queries while unsupported relationship shapes remain live-only.
- Shell completion covers commands, subcommands, and flags; package and symbol completion are not dynamic yet.
- `gosherpa analyze` hotspots and entrypoint candidates are inventory-based; use focused `context`, `entrypoints`, `impact`, and `tests` commands for deeper relationship analysis.
- Unqualified standalone call targets can be ambiguous across packages; GoSherpa reports candidates and suggests package-qualified targets such as `./internal/auth.Target`.

## Release Notes

- [GoSherpa Impact Engine v0.1](releases/RELEASE_NOTES_V01.md)
