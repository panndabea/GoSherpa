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
    <img alt="Analysis: AST based" src="https://img.shields.io/badge/analysis-AST%20based-F6AD55?style=for-the-badge">
  </p>
</div>

---

> Ask a code-structure question. Get a small, trustworthy answer.

GoSherpa is a fast command-line companion for understanding Go repositories without opening half the project in your editor. It gives you just enough structure to navigate confidently: where a symbol lives, who references it, which packages depend on it, and how functions connect.

The Impact Engine described in [PRD_V01.md](PRD_V01.md) is implemented as the v0.1 MVP: change-aware analysis for answering which packages, interfaces, implementations, and tests are affected by a file, package, symbol, or git diff. The next product track is Symbol Intelligence from [PRD_V02.md](PRD_V02.md), with `gosherpa explain` as the likely center of gravity.

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
| Symbol lookup | `gosherpa symbol ParseFile` | Shows a definition with kind, file, and line |
| Symbol explanation | `gosherpa explain ParseFile` | Combines purpose, risk, architecture role, reading order, callers/callees, impact signals, and tests |
| Reference search | `gosherpa refs ParseFile` | Finds Go-aware definitions and references across the repository |
| Impact analysis | `gosherpa impact ParseFile` | Summarizes references, caller-chain impact, affected packages, and suggested tests |
| Structured impact | `gosherpa impact file service.go` | Reports file, package, symbol, and diff impact through a shared report model |
| Diff impact | `gosherpa impact diff --base HEAD` | Reports changed files, changed packages, affected packages, and affected tests |
| Test discovery | `gosherpa tests ParseFile` | Lists related tests and suggested `go test` commands |
| Affected tests | `gosherpa tests affected --base HEAD` | Prints suggested test commands for a git diff |
| Machine-readable output | `gosherpa symbols --json` | Emits JSON for all commands with a stable response envelope |
| Package dependencies | `gosherpa deps ./internal/sherpa` | Shows imports and local dependents |
| Callees | `gosherpa callees ./internal/sherpa.ParseFile` | Lists direct calls made by a function or method |
| Callers | `gosherpa callers ./internal/sherpa.ParseFile` | Lists direct syntactic callers of a function or method |
| Call path | `gosherpa path Run ./internal/sherpa.ParseFile` | Shows the shortest repository-local call path between functions or methods |
| Call paths | `gosherpa paths Run ./internal/sherpa.ParseFile --limit 3` | Shows multiple call paths with optional limit and max depth |

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
./gosherpa explain ParseFile
./gosherpa explain ParseFile --json
./gosherpa refs ParseFile
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
./gosherpa callees ParseFile
./gosherpa callees ./internal/sherpa.ParseFile
./gosherpa callees ParseFile --json
./gosherpa callers ParseFile
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

Prefer not to build a binary yet?

```bash
go run ./cmd/gosherpa symbols
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

GoSherpa is an early MVP. The Impact Engine v0.1 track from [PRD_V01.md](PRD_V01.md) is complete; the next work is Symbol Intelligence from [PRD_V02.md](PRD_V02.md).

| Next | Goal |
| --- | --- |
| Symbol Intelligence | Start `gosherpa explain` as the focused entry point for PRD v0.5 |

Read the full product plan in [FEATURE_ROADMAP.md](FEATURE_ROADMAP.md).

## Status

Implemented:

- Repository scanning and Go file discovery
- Struct, interface, function, and method discovery
- Symbol lookup and Go-aware reference lookup
- Initial `gosherpa explain <symbol>` profile with purpose, risk, architecture role, reading order, human output, and JSON output
- Package-aware caller/callee signals for package-qualified `gosherpa explain` targets
- Direct symbol and package impact analysis
- Related test discovery with suggested `go test` commands
- Package dependency analysis
- Direct syntactic caller and callee analysis
- Shortest and limited repository-local call path analysis
- Package-aware standalone call graph commands for package-qualified targets
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

Release notes:

- [GoSherpa Impact Engine v0.1](RELEASE_NOTES_V01.md)

Planned next:

- resolve receiver-variable method calls with type information

Known MVP limitations:

- Reference search is type-aware inside packages and recognizes local package selector calls, but it does not yet use full module/package loading.
- Test files are skipped by reference, caller, and callee analysis.
- Symbol impact includes transitive callers and package tests for affected caller packages.
- Diff impact is hunk-based; it reports directly changed or deleted top-level Go functions and struct/interface types, but it does not infer every semantic consequence of changed statements.
- Package-qualified symbol impact disambiguates references and affected tests; unqualified symbol targets can still be ambiguous across packages.
- Interface implementer impact canonicalizes local/external import paths in method signatures and resolves local embedded interfaces, but it does not yet run the full Go type checker for aliases, build tags, or generic edge cases.
- Test discovery uses same-package tests and syntactic direct-reference matching; table-test names are not extracted yet.
- Caller and callee analysis is AST-based and can miss receiver-variable method calls.
- Call path analysis inherits the current AST-based caller and callee limitations.
- Unqualified standalone call targets can still be ambiguous across packages; use package-qualified targets such as `./internal/auth.Target` to disambiguate.

## Philosophy

GoSherpa focuses on visibility, not magic.

It does not try to become your IDE, rewrite your project, or infer more certainty than it has. It reads the code you already have and returns compact answers that help you keep moving.
