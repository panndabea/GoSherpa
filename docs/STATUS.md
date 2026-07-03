# GoSherpa Status

GoSherpa is an early MVP. The Impact Engine v0.1 track from [PRD_V01.md](product/PRD_V01.md) is complete; current work continues the Symbol Intelligence track from [PRD_V02.md](product/PRD_V02.md).

Read the full product plan in [FEATURE_ROADMAP.md](product/FEATURE_ROADMAP.md), or start from the [docs index](README.md).

## Roadmap

| Next | Goal |
| --- | --- |
| Symbol Intelligence | Deepen typechecked relationships, context output, and structured test planning around `gosherpa explain` |

## Implemented

- Repository scanning and Go file discovery
- Struct, interface, function, and method discovery
- Symbol listing filters by kind, package, and test symbols
- Rich symbol lookup with package paths, signatures, doc comments, struct fields, interface methods, and Go-aware reference lookup
- Typechecked reference analysis via `go/packages`, with an AST/per-package fallback when semantic loading is unavailable
- Ranked `gosherpa search <terms>` for partial, case-insensitive symbol discovery with kind, package, test, and limit filters
- Initial `gosherpa analyze [path]` repository overview with package summaries, symbol counts, important symbols, entrypoint candidates, hotspots, testing overview, readiness, limitations, suggested next commands, and JSON output
- Initial `gosherpa explain <symbol>` profile with purpose, risk, architecture role, reading order, human output, and JSON output
- Initial `gosherpa context symbol <target>` export with source excerpt, symbol identity, references, callers, callees, impact signals, tests, confidence, limitations, and JSON output
- Initial `gosherpa context file <file>` export with file symbols, source excerpts, affected packages, affected tests, reading order, confidence, limitations, and JSON output
- Initial `gosherpa context package <package>` export with package files, symbols, source excerpts, affected packages, affected tests, reading order, confidence, limitations, and JSON output
- Initial `gosherpa context diff --base <ref>` export with changed files, changed packages, changed symbols, affected packages, affected tests, reading order, confidence, limitations, and JSON output
- Context export size controls with entry-count limits, source radius limits, and `--max-bytes` byte-budget truncation
- Initial `gosherpa doctor` readiness report with module, Go environment, workspace, build tag, package loading, snapshot, confidence, limitations, warnings, and JSON output
- Package-aware caller/callee signals for package-qualified `gosherpa explain` targets
- Direct symbol and package impact analysis
- Related test discovery with suggested `go test` commands
- Package dependency analysis
- Direct caller and callee analysis with package-aware targets and receiver-variable method calls
- Shortest and limited repository-local call path analysis
- Initial `gosherpa entrypoints <target>` analysis for `main.main`, test functions with `--tests`, exported functions, and functions with no local callers
- Package-aware standalone call graph commands for package-qualified targets
- Receiver-variable method calls in standalone call graph commands, resolved with package-level type information
- Standalone interface navigation with `implementers` and `interfaces`
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
- `gosherpa pr --base <ref>` with human and JSON output for PR-style changed files, packages, symbols, risk notes, affected tests, and verification commands
- Diff-oriented reports enrich changed top-level symbols with typechecked reference and call impact when package loading is available, exposing `git-diff+typechecked+ast`, `referenceAnalysisMode`, and `callAnalysisMode`
- `gosherpa impact file|package|symbol` with human and JSON output
- Interface and implementer impact signals based on local method sets with import-aware signature matching and embedded-interface expansion
- Changed-symbol extraction from git diff hunks, including deleted symbols from base files
- Package-qualified symbol impact for references and affected tests
- Transitive caller impact for symbol changes
- Affected-test planning for transitive caller packages
- Package inventory with `gosherpa packages`, including file, symbol, import, reverse-dependency, and test indicators
- All-package dependency overview with `gosherpa deps --all`, including local imports, external imports, and reverse dependencies
- Reference kind classification and `gosherpa refs --kind <kind>` filtering
- Source ranges with columns for symbols, references, callers, callees, call paths, and related tests in JSON output

## Known MVP Limitations

- Reference search uses `go/packages`-backed typechecked loading when available and falls back to AST/per-package type information; caller, callee, path, and interface impact analysis remain conservative and mostly local.
- Test files are skipped by reference, callee, path, default caller, and default entrypoint analysis; `callers --tests`, `entrypoints --tests`, and `explain --tests` include test-file callers on demand.
- Symbol impact includes transitive callers and package tests for affected caller packages.
- Diff impact is hunk-based; it reports directly changed or deleted top-level Go functions and struct/interface types, enriches current changed symbols with typechecked reference/call impact when available, but it does not infer every semantic consequence of changed statements.
- Package-qualified symbol impact disambiguates references and affected tests; unqualified symbol targets may require disambiguation across packages.
- Interface implementer impact canonicalizes local/external import paths in method signatures and resolves local embedded interfaces, but it does not yet use full module-level typechecked analysis for aliases, build tags, or generic edge cases.
- Test discovery uses direct references, same-package tests, and literal `t.Run` subtest names; dynamic table-driven names may be incomplete.
- Caller, callee, path, and entrypoint analysis still do not resolve dynamic dispatch, reflection, function values, or every imported-package receiver call.
- Entrypoint analysis is heuristic; framework-specific entrypoints such as HTTP routers and CLI command handlers are not inferred yet.
- Context export currently supports symbol, file, package, and diff targets.
- `gosherpa analyze` hotspots and entrypoint candidates are inventory-based; use focused `context`, `entrypoints`, `impact`, and `tests` commands for deeper relationship analysis.
- Unqualified standalone call targets can be ambiguous across packages; GoSherpa reports candidates and suggests package-qualified targets such as `./internal/auth.Target`.

## Release Notes

- [GoSherpa Impact Engine v0.1](releases/RELEASE_NOTES_V01.md)
