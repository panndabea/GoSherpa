# GoSherpa CLI Reference

This page keeps the full command overview and machine-readable output notes outside the root README. Start with the [README](../README.md) if you want the shorter human introduction.

## Build And Run

```bash
git clone https://github.com/panndabea/GoSherpa.git
cd GoSherpa
go test ./...
go build -o gosherpa ./cmd/gosherpa
```

Prefer not to build a binary yet?

```bash
go run ./cmd/gosherpa analyze .
go run ./cmd/gosherpa architecture
go run ./cmd/gosherpa risk
go run ./cmd/gosherpa symbols --kind function
go run ./cmd/gosherpa context symbol ParseFile
go run ./cmd/gosherpa context file internal/sherpa/impact.go
go run ./cmd/gosherpa doctor
go run ./cmd/gosherpa explain ParseFile
go run ./cmd/gosherpa callers ParseFile
go run ./cmd/gosherpa impact ParseFile
go run ./cmd/gosherpa impact file internal/sherpa/impact.go
go run ./cmd/gosherpa impact diff --base HEAD
go run ./cmd/gosherpa pr --base HEAD
go run ./cmd/gosherpa tests ParseFile
go run ./cmd/gosherpa tests affected --base HEAD
go run ./cmd/gosherpa packages
go run ./cmd/gosherpa entrypoints ParseFile
go run ./cmd/gosherpa path main FindCallers
```

Use `--root` to run GoSherpa from another working directory. The path must point to a Go module root containing `go.mod` or a Go workspace root containing `go.work`.

```bash
./gosherpa --root /path/to/GoSherpa symbols
./gosherpa refs ParseFile --root /path/to/GoSherpa
```

## Command Overview

| Capability | Command | Result |
| --- | --- | --- |
| Repository overview | `gosherpa analyze .` | Summarizes packages, symbols, entrypoint candidates, structural risk, hotspots, tests, readiness, limitations, and suggested next commands |
| Architecture overview | `gosherpa architecture` | Reports dependency cycles, most coupled packages, high fan-in/fan-out packages, largest packages, and leaf packages |
| Risk overview | `gosherpa risk` | Summarizes structural repository risk from cycles, fan-in/fan-out, public API surface, interfaces, and tests |
| Symbol atlas | `gosherpa symbols --kind function --package ./internal/sherpa` | Lists discovered structs, interfaces, functions, and methods with optional kind, package, and test filters |
| Symbol lookup | `gosherpa symbol ParseFile` | Shows package, signature, docs, fields/methods, and source location |
| Symbol search | `gosherpa search parse file --kind function --limit 5` | Finds symbols by ranked, partial, case-insensitive matches with optional filters |
| Symbol explanation | `gosherpa explain ParseFile` | Combines purpose, risk, architecture role, reading order, callers/callees, impact signals, and tests |
| Agent context | `gosherpa context symbol ParseFile` | Exports a compact pre-edit context with identity, source excerpt, relationships, tests, confidence, and limitations |
| File context | `gosherpa context file internal/sherpa/impact.go` | Exports file symbols, source excerpts, affected packages/tests, reading order, confidence, and limitations |
| Package context | `gosherpa context package ./internal/sherpa` | Exports package files, symbols, source excerpts, affected packages/tests, reading order, confidence, and limitations |
| Diff context | `gosherpa context diff --base HEAD` | Exports changed files, changed symbols, typechecked changed-symbol impact when available, affected packages/tests, reading order, confidence, and limitations |
| Analysis readiness | `gosherpa doctor` | Reports module, Go environment, package loading, build tags, workspace, snapshot status, confidence, and warnings |
| Test-aware explanation | `gosherpa explain ParseFile --tests` | Includes test-file callers in the symbol profile on demand |
| Reference search | `gosherpa refs ParseFile --kind call` | Finds Go-aware definitions and references, with optional kind filtering |
| Impact analysis | `gosherpa impact ParseFile` | Summarizes references, caller-chain impact, affected packages, and suggested tests |
| Structured impact | `gosherpa impact file service.go` | Reports file, package, symbol, and diff impact through a shared report model |
| Diff impact | `gosherpa impact diff --base HEAD` | Reports changed files, changed packages, changed-symbol impact, affected packages, and affected tests |
| PR review | `gosherpa pr --base HEAD` | Summarizes changed files, packages, symbols, risk, affected tests, and verification commands |
| Test discovery | `gosherpa tests ParseFile --scope direct` | Lists related tests and suggested `go test` commands with optional direct/related/all scope |
| Affected tests | `gosherpa tests affected --base HEAD` | Prints suggested test commands for a git diff |
| Machine-readable output | `gosherpa symbols --json` | Emits JSON for all commands with a stable response envelope |
| Package inventory | `gosherpa packages --tests` | Lists local packages with file, symbol, import, reverse-dependency, and test indicators |
| Package dependencies | `gosherpa deps ./internal/sherpa` | Shows imports and local dependents |
| Dependency overview | `gosherpa deps --all` | Shows local and external imports plus reverse dependencies for every package |
| Interface implementers | `gosherpa implementers ./internal/auth.Authenticator` | Lists concrete local types satisfying an interface |
| Satisfied interfaces | `gosherpa interfaces ./internal/jwt.JWTAuthenticator` | Lists local interfaces satisfied by a type |
| Callees | `gosherpa callees ./internal/sherpa.ParseFile` | Lists direct calls made by a function or method |
| Callers | `gosherpa callers ./internal/sherpa.ParseFile` | Lists direct callers of a function or method |
| Test callers | `gosherpa callers ./internal/sherpa.ParseFile --tests` | Includes direct callers from `_test.go` files on demand |
| Entrypoints | `gosherpa entrypoints ./internal/sherpa.ParseFile` | Lists public, runtime, test, and no-local-caller functions that can reach a target |
| Call path | `gosherpa path Run ./internal/sherpa.ParseFile` | Shows the shortest repository-local call path between functions or methods |
| Call paths | `gosherpa paths Run ./internal/sherpa.ParseFile --limit 3` | Shows multiple repository-local call paths with optional limit and max depth |

Example reference output:

```text
REFERENCES

ParseFile

  internal/sherpa/repository.go:12
  internal/sherpa/reference.go:41

Found 4 references
```

## Example Commands

```bash
./gosherpa analyze .
./gosherpa analyze . --tests
./gosherpa analyze . --json
./gosherpa architecture
./gosherpa architecture --tests
./gosherpa architecture --json
./gosherpa risk
./gosherpa risk --tests
./gosherpa risk --json
./gosherpa symbols
./gosherpa symbols --kind struct
./gosherpa symbols --kind method --package ./internal/sherpa
./gosherpa symbols --tests
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
./gosherpa context symbol ParseFile --max-bytes 12000 --json
./gosherpa context file internal/sherpa/impact.go
./gosherpa context file internal/sherpa/impact.go --json
./gosherpa context package ./internal/sherpa
./gosherpa context package ./internal/sherpa --json
./gosherpa context diff --base HEAD
./gosherpa context diff --base HEAD --json
./gosherpa doctor
./gosherpa doctor --json
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
./gosherpa pr --base HEAD
./gosherpa pr --base HEAD --json
./gosherpa tests ParseFile
./gosherpa tests ParseFile --scope direct
./gosherpa tests ParseFile --scope all
./gosherpa tests ParseFile --json
./gosherpa tests ./internal/sherpa
./gosherpa tests affected --base HEAD
./gosherpa tests affected --base HEAD --json
./gosherpa packages
./gosherpa packages --tests
./gosherpa packages --json
./gosherpa deps ./internal/sherpa
./gosherpa deps --all
./gosherpa deps ./internal/sherpa --json
./gosherpa deps --all --json
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
./gosherpa entrypoints ParseFile
./gosherpa entrypoints ParseFile --tests
./gosherpa entrypoints ./internal/sherpa.ParseFile --json
./gosherpa path main FindCallers
./gosherpa path main ./internal/sherpa.FindCallers
./gosherpa path main FindCallers --json
./gosherpa paths main collectCalleesFromFunction --limit 3 --max-depth 6
./gosherpa paths main ./internal/sherpa.collectCalleesFromFunction --limit 3 --max-depth 6
./gosherpa paths main collectCalleesFromFunction --limit 3 --max-depth 6 --json
```

## JSON Output

JSON output for all commands uses a stable envelope:

```json
{
  "schemaVersion": 1,
  "command": "impact",
  "target": "ParseFile",
  "root": "/path/to/GoSherpa",
  "modulePath": "github.com/panndabea/GoSherpa",
  "warnings": [],
  "data": {}
}
```

When `--json` is used with an ambiguous target, GoSherpa keeps stdout empty and emits a structured diagnostic to stderr with `error.code: "ambiguous_target"` and candidate package/file/line details.

Context commands support size controls for agent workflows: `--max-files`,
`--max-references`, `--max-symbols`, `--max-tests`, `--source-radius`, and
`--max-bytes`. The byte budget omits large context fields deterministically and
keeps JSON valid; any omissions are reported in `data.truncated`.

Diff-oriented JSON such as `impact diff`, `tests affected`, `pr`, and
`context diff` can report `git-diff+typechecked+ast` plus
`referenceAnalysisMode` and `callAnalysisMode` when changed-symbol impact uses
typechecked package loading.

`analyze --json` provides the repository-level entry point for agents and
scripts: package summaries, symbol counts, important public symbols,
entrypoint candidates, structural risk, simple hotspots, test overview,
readiness, limitations, and suggested next commands.

See the [Agent JSON Schema](product/JSON_SCHEMA_V1.md) and [Context JSON Schema](product/CONTEXT_SCHEMA_V1.md) for the full machine-readable contracts.
