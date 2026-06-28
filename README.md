<div align="center">
  <img src="pictures/gosherpa-readme-hero.png" alt="GoSherpa explorer mascot with a mountain map" width="900">

  <h1>GoSherpa</h1>

  <p>
    <strong>Structural code intelligence for Go projects.</strong><br>
    Explore symbols, references, packages, callers, and callees from a calm, deterministic CLI.
  </p>

  <p>
    <img alt="Go 1.24.4" src="https://img.shields.io/badge/Go-1.24.4-00ADD8?style=for-the-badge&amp;logo=go&amp;logoColor=white">
    <img alt="Status: Early MVP" src="https://img.shields.io/badge/status-early%20MVP-2F855A?style=for-the-badge">
    <img alt="Interface: CLI" src="https://img.shields.io/badge/interface-CLI-111827?style=for-the-badge">
    <img alt="Analysis: Go semantics + AST" src="https://img.shields.io/badge/analysis-Go%20semantics%20%2B%20AST-F6AD55?style=for-the-badge">
    <img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-2563EB?style=for-the-badge">
  </p>
</div>

---

> Ask a code-structure question. Get a small, trustworthy answer.

GoSherpa is a fast command-line companion for understanding Go repositories without opening half the project in your editor. It gives you just enough structure to navigate confidently: where a symbol lives, who references it, which packages depend on it, and how functions connect.

The Impact Engine described in [PRD_V01.md](docs/product/PRD_V01.md) is implemented as the v0.1 MVP: change-aware analysis for answering which packages, interfaces, implementations, and tests are affected by a file, package, symbol, or git diff. The active product track is Symbol Intelligence from [PRD_V02.md](docs/product/PRD_V02.md), expanding `gosherpa explain`, agent context exports, and semantic reference accuracy.

## Why It Exists

As Go projects grow, the important answer is often spread across files, packages, and call sites. GoSherpa turns common navigation questions into focused commands:

| When you ask... | GoSherpa helps you find... |
| --- | --- |
| "Where is this thing defined?" | Symbols, files, and line numbers |
| "Who uses this function?" | References and direct callers |
| "What does this function touch?" | Direct callees |
| "What depends on this package?" | Local package dependency relationships |
| "What might this change affect?" | Symbol, caller-chain, and diff-based package/test impact |

## Current Experience

| Capability | Command | Result |
| --- | --- | --- |
| Symbol atlas | `gosherpa symbols` | Lists discovered structs, interfaces, functions, and methods |
| Symbol lookup | `gosherpa symbol ParseFile` | Shows package, signature, docs, fields/methods, and source location |
| Symbol search | `gosherpa search parse file --kind function --limit 5` | Finds symbols by ranked, partial, case-insensitive matches with optional filters |
| Symbol explanation | `gosherpa explain ParseFile` | Combines purpose, risk, architecture role, reading order, callers/callees, impact signals, and tests |
| Agent context | `gosherpa context symbol ParseFile` | Exports a compact pre-edit context with identity, source excerpt, relationships, tests, confidence, and limitations |
| File context | `gosherpa context file internal/sherpa/impact.go` | Exports file symbols, source excerpts, affected packages/tests, reading order, confidence, and limitations |
| Package context | `gosherpa context package ./internal/sherpa` | Exports package files, symbols, source excerpts, affected packages/tests, reading order, confidence, and limitations |
| Diff context | `gosherpa context diff --base HEAD` | Exports changed files, changed symbols, affected packages/tests, reading order, confidence, and limitations |
| Test-aware explanation | `gosherpa explain ParseFile --tests` | Includes test-file callers in the symbol profile on demand |
| Reference search | `gosherpa refs ParseFile --kind call` | Finds Go-aware definitions and references, with optional kind filtering |
| Impact analysis | `gosherpa impact ParseFile` | Summarizes references, caller-chain impact, affected packages, and suggested tests |
| Structured impact | `gosherpa impact file service.go` | Reports file, package, symbol, and diff impact through a shared report model |
| Diff impact | `gosherpa impact diff --base HEAD` | Reports changed files, changed packages, affected packages, and affected tests |
| Test discovery | `gosherpa tests ParseFile` | Lists related tests and suggested `go test` commands |
| Affected tests | `gosherpa tests affected --base HEAD` | Prints suggested test commands for a git diff |
| Machine-readable output | `gosherpa symbols --json` | Emits JSON for all commands with a stable response envelope |
| Package dependencies | `gosherpa deps ./internal/sherpa` | Shows imports and local dependents |
| Interface implementers | `gosherpa implementers ./internal/auth.Authenticator` | Lists concrete local types satisfying an interface |
| Satisfied interfaces | `gosherpa interfaces ./internal/jwt.JWTAuthenticator` | Lists local interfaces satisfied by a type |
| Callees | `gosherpa callees ./internal/sherpa.ParseFile` | Lists direct calls made by a function or method |
| Callers | `gosherpa callers ./internal/sherpa.ParseFile` | Lists direct callers of a function or method |
| Test callers | `gosherpa callers ./internal/sherpa.ParseFile --tests` | Includes direct callers from `_test.go` files on demand |
| Call path | `gosherpa path Run ./internal/sherpa.ParseFile` | Shows the shortest repository-local call path between functions or methods |
| Call paths | `gosherpa paths Run ./internal/sherpa.ParseFile --limit 3` | Shows multiple repository-local call paths with optional limit and max depth |

```text
REFERENCES

ParseFile

  internal/sherpa/repository.go:12
  internal/sherpa/reference.go:41

Found 4 references
```

## Quick Start

```bash
git clone https://github.com/panndabea/GoSherpa.git
cd GoSherpa
go test ./...
go build -o gosherpa ./cmd/gosherpa
```

Then explore the repository:

```bash
./gosherpa symbols
./gosherpa symbols --json
./gosherpa symbol ParseFile
./gosherpa symbol ParseFile --json
./gosherpa search parse file
./gosherpa search parse file --kind function --limit 5
./gosherpa search ParseFile --package ./internal/sherpa
./gosherpa search ParseFile --tests
./gosherpa search parse file --json
./gosherpa context symbol ParseFile
./gosherpa context symbol ParseFile --tests
./gosherpa context symbol ParseFile --json
./gosherpa context file internal/sherpa/impact.go
./gosherpa context file internal/sherpa/impact.go --json
./gosherpa context package ./internal/sherpa
./gosherpa context package ./internal/sherpa --json
./gosherpa context diff --base HEAD
./gosherpa context diff --base HEAD --json
./gosherpa explain ParseFile
./gosherpa explain ParseFile --tests
./gosherpa explain ParseFile --json
./gosherpa refs ParseFile
./gosherpa refs ParseFile --kind call
./gosherpa refs ParseFile --json
./gosherpa impact ParseFile
./gosherpa impact ParseFile --json
./gosherpa impact ./internal/sherpa
./gosherpa impact file internal/sherpa/impact.go
./gosherpa impact package ./internal/sherpa
./gosherpa impact symbol ParseFile
./gosherpa impact diff --base HEAD
./gosherpa impact diff --base HEAD --json
./gosherpa tests ParseFile
./gosherpa tests ParseFile --json
./gosherpa tests ./internal/sherpa
./gosherpa tests affected --base HEAD
./gosherpa tests affected --base HEAD --json
./gosherpa deps ./internal/sherpa
./gosherpa deps ./internal/sherpa --json
./gosherpa implementers ./internal/auth.Authenticator
./gosherpa implementers ./internal/auth.Authenticator --json
./gosherpa interfaces ./internal/jwt.JWTAuthenticator
./gosherpa interfaces ./internal/jwt.JWTAuthenticator --json
./gosherpa callees ParseFile
./gosherpa callees ./internal/sherpa.ParseFile
./gosherpa callees ParseFile --json
./gosherpa callers ParseFile
./gosherpa callers ParseFile --tests
./gosherpa callers ./internal/sherpa.ParseFile
./gosherpa callers ParseFile --json
./gosherpa path main FindCallers
./gosherpa path main ./internal/sherpa.FindCallers
./gosherpa path main FindCallers --json
./gosherpa paths main collectCalleesFromFunction --limit 3 --max-depth 6
./gosherpa paths main ./internal/sherpa.collectCalleesFromFunction --limit 3 --max-depth 6
./gosherpa paths main collectCalleesFromFunction --limit 3 --max-depth 6 --json
```

Use `--root` to run GoSherpa from another working directory. The path must point to a Go module root containing `go.mod`.

```bash
./gosherpa --root /path/to/GoSherpa symbols
./gosherpa refs ParseFile --root /path/to/GoSherpa
```

JSON output for all commands uses a stable envelope:

```json
{
  "schemaVersion": 1,
  "command": "impact",
  "target": "ParseFile",
  "root": "/path/to/GoSherpa",
  "modulePath": "github.com/supertabaluga/gosherpa",
  "warnings": [],
  "data": {}
}
```

When `--json` is used with an ambiguous target, GoSherpa keeps stdout empty and
emits a structured diagnostic to stderr with `error.code: "ambiguous_target"`
and candidate package/file/line details.

Prefer not to build a binary yet?

```bash
go run ./cmd/gosherpa symbols
go run ./cmd/gosherpa context symbol ParseFile
go run ./cmd/gosherpa context file internal/sherpa/impact.go
go run ./cmd/gosherpa explain ParseFile
go run ./cmd/gosherpa callers ParseFile
go run ./cmd/gosherpa impact ParseFile
go run ./cmd/gosherpa impact file internal/sherpa/impact.go
go run ./cmd/gosherpa impact diff --base HEAD
go run ./cmd/gosherpa tests ParseFile
go run ./cmd/gosherpa tests affected --base HEAD
go run ./cmd/gosherpa path main FindCallers
```

## Design Principles

| Principle | What it means in practice |
| --- | --- |
| Human-readable first | Output is meant to be scanned in a terminal, not decoded from a dump |
| Deterministic by default | Stable answers make it easier to trust diffs and repeated runs |
| Small commands | Each command answers one navigation question clearly |
| Repository-native | GoSherpa uses information already present in the codebase |
| No hidden ceremony | No annotations, no generated metadata, no project-specific setup for normal use |

## Roadmap

GoSherpa is an early MVP. The Impact Engine v0.1 track from [PRD_V01.md](docs/product/PRD_V01.md) is complete; current work continues the Symbol Intelligence track from [PRD_V02.md](docs/product/PRD_V02.md).

| Next | Goal |
| --- | --- |
| Symbol Intelligence | Deepen typechecked relationships, context output, and structured test planning around `gosherpa explain` |

Read the full product plan in [FEATURE_ROADMAP.md](docs/product/FEATURE_ROADMAP.md), or start from the [docs index](docs/README.md).

## Status

Implemented:

- Repository scanning and Go file discovery
- Struct, interface, function, and method discovery
- Rich symbol lookup with package paths, signatures, doc comments, struct fields, interface methods, and Go-aware reference lookup
- Typechecked reference analysis via `go/packages`, with an AST/per-package fallback when semantic loading is unavailable
- Ranked `gosherpa search <terms>` for partial, case-insensitive symbol discovery with kind, package, test, and limit filters
- Initial `gosherpa explain <symbol>` profile with purpose, risk, architecture role, reading order, human output, and JSON output
- Initial `gosherpa context symbol <target>` export with source excerpt, symbol identity, references, callers, callees, impact signals, tests, confidence, limitations, and JSON output
- Initial `gosherpa context file <file>` export with file symbols, source excerpts, affected packages, affected tests, reading order, confidence, limitations, and JSON output
- Initial `gosherpa context package <package>` export with package files, symbols, source excerpts, affected packages, affected tests, reading order, confidence, limitations, and JSON output
- Initial `gosherpa context diff --base <ref>` export with changed files, changed packages, changed symbols, affected packages, affected tests, reading order, confidence, limitations, and JSON output
- Package-aware caller/callee signals for package-qualified `gosherpa explain` targets
- Direct symbol and package impact analysis
- Related test discovery with suggested `go test` commands
- Package dependency analysis
- Direct caller and callee analysis with package-aware targets and receiver-variable method calls
- Shortest and limited repository-local call path analysis
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
- `gosherpa impact file|package|symbol` with human and JSON output
- Interface and implementer impact signals based on local method sets with import-aware signature matching and embedded-interface expansion
- Changed-symbol extraction from git diff hunks, including deleted symbols from base files
- Package-qualified symbol impact for references and affected tests
- Transitive caller impact for symbol changes
- Affected-test planning for transitive caller packages
- Reference kind classification and `gosherpa refs --kind <kind>` filtering

Release notes:

- [GoSherpa Impact Engine v0.1](docs/releases/RELEASE_NOTES_V01.md)

Known MVP limitations:

- Reference search uses `go/packages`-backed typechecked loading when available and falls back to AST/per-package type information; caller, callee, path, and interface impact analysis remain conservative and mostly local.
- Test files are skipped by reference, callee, path, and default caller analysis; `callers --tests` and `explain --tests` include test-file callers on demand.
- Symbol impact includes transitive callers and package tests for affected caller packages.
- Diff impact is hunk-based; it reports directly changed or deleted top-level Go functions and struct/interface types, but it does not infer every semantic consequence of changed statements.
- Package-qualified symbol impact disambiguates references and affected tests; unqualified symbol targets may require disambiguation across packages.
- Interface implementer impact canonicalizes local/external import paths in method signatures and resolves local embedded interfaces, but it does not yet use full module-level typechecked analysis for aliases, build tags, or generic edge cases.
- Test discovery uses same-package tests and syntactic direct-reference matching; table-test names are not extracted yet.
- Caller, callee, and path analysis still do not resolve dynamic dispatch, reflection, function values, or every imported-package receiver call.
- Context export currently supports symbol, file, package, and diff targets.
- Unqualified standalone call targets can be ambiguous across packages; GoSherpa reports candidates and suggests package-qualified targets such as `./internal/auth.Target`.

## Philosophy

GoSherpa focuses on visibility, not magic.

It does not try to become your IDE, rewrite your project, or infer more certainty than it has. It reads the code you already have and returns compact answers that help you keep moving.

## License

GoSherpa is available under the [MIT License](LICENSE).
