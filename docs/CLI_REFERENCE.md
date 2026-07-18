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
go run ./cmd/gosherpa version
go run ./cmd/gosherpa architecture
go run ./cmd/gosherpa risk
go run ./cmd/gosherpa symbols --kind function
go run ./cmd/gosherpa context symbol ParseFile
go run ./cmd/gosherpa agent context --base HEAD
go run ./cmd/gosherpa context file internal/sherpa/impact.go
go run ./cmd/gosherpa doctor
go run ./cmd/gosherpa snapshot
go run ./cmd/gosherpa completion zsh
go run ./cmd/gosherpa explain ParseFile
go run ./cmd/gosherpa callers ParseFile
go run ./cmd/gosherpa impact ParseFile
go run ./cmd/gosherpa impact file internal/sherpa/impact.go
go run ./cmd/gosherpa impact diff --base HEAD
go run ./cmd/gosherpa pr --base HEAD
go run ./cmd/gosherpa tests ParseFile
go run ./cmd/gosherpa tests internal/sherpa/impact.go
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

Generated Go files are included when they are repository-local and visible to
the selected module or workspace. `gosherpa doctor` counts files with the
standard `// Code generated ... DO NOT EDIT.` header; relationship and context
commands treat those files like other compiler-visible Go files.

Use `gosherpa snapshot` to create a reusable inventory and relationship-capable
snapshot file. A valid snapshot can currently be reused by `analyze`,
`symbols`, `symbol`, `search`, `packages --tests`, `refs`, `callers`,
`callees`, `implementers`, `interface`, `interfaces`, `context symbol`,
`impact symbol`, and by diff-oriented current changed-symbol inventory plus
selected relationship subanalysis in `context diff`, `impact diff`,
`tests affected`, `pr`, and `agent context` with `--use-snapshot`. Missing,
stale, invalid, or relationship-incompatible snapshots fall back to live
repository analysis and report a warning.

```bash
./gosherpa snapshot
./gosherpa analyze --use-snapshot
./gosherpa symbols --use-snapshot
./gosherpa search parse --use-snapshot --json
./gosherpa packages --tests --use-snapshot
./gosherpa refs ParseFile --use-snapshot --json
./gosherpa callers ParseFile --use-snapshot --json
./gosherpa callees ParseFile --use-snapshot --json
./gosherpa context symbol ParseFile --use-snapshot --json
./gosherpa agent context --base HEAD --use-snapshot --json
./gosherpa impact symbol ParseFile --use-snapshot --json
./gosherpa context diff --base HEAD --use-snapshot --json
./gosherpa impact diff --base HEAD --use-snapshot --json
./gosherpa pr --base HEAD --use-snapshot --json
```

Enable shell completion by evaluating the script for your shell. The generated
scripts complete command names, subcommands, and supported flags; package and
symbol completion can come later.

```bash
./gosherpa completion zsh
./gosherpa completion bash
./gosherpa completion fish
```

## Command Overview

| Capability | Command | Result |
| --- | --- | --- |
| Version | `gosherpa version` | Prints GoSherpa and Go runtime version information |
| Help | `gosherpa help callers` | Prints global or command-specific usage without running analysis |
| Repository overview | `gosherpa analyze .` | Summarizes packages, symbols, entrypoint candidates, structural risk, hotspots, tests, readiness, limitations, and suggested next commands |
| Architecture overview | `gosherpa architecture` | Reports dependency cycles, most coupled packages, high fan-in/fan-out packages, largest packages, and leaf packages |
| Risk overview | `gosherpa risk` | Summarizes structural repository risk from cycles, fan-in/fan-out, public API surface, interfaces, and tests |
| Symbol atlas | `gosherpa symbols --kind function --package ./internal/sherpa` | Lists discovered structs, interfaces, type aliases, functions, and methods with optional kind, package, and test filters |
| Symbol lookup | `gosherpa symbol ParseFile` | Shows package, signature, docs, fields/methods, and source location |
| Symbol search | `gosherpa search parse file --kind function --limit 5` | Finds symbols by ranked, partial, case-insensitive matches with optional filters |
| Symbol explanation | `gosherpa explain ParseFile` | Combines purpose, risk, target risk, architecture role, reading order, callers/callees, impact signals, and tests |
| Agent workflow | `gosherpa agent context --base HEAD` | Composes readiness, snapshot status, diff context, target risk, affected tests, section modes, and next commands into one bounded daily-driver answer |
| Agent context | `gosherpa context symbol ParseFile` | Exports a compact pre-edit context with identity, source excerpt, relationships, tests, target risk, confidence, and limitations |
| File context | `gosherpa context file internal/sherpa/impact.go` | Exports file symbols, source excerpts, affected packages/tests, target risk, reading order, confidence, and limitations |
| Package context | `gosherpa context package ./internal/sherpa` | Exports package files, symbols, source excerpts, affected packages/tests, target risk, reading order, confidence, and limitations |
| Diff context | `gosherpa context diff --base HEAD` | Exports changed files, changed symbols, typechecked changed-symbol impact when available, affected packages/tests, target risk, reading order, confidence, and limitations |
| Analysis readiness | `gosherpa doctor` | Reports module, Go environment, package loading, build tags, workspace, snapshot status, confidence, and warnings |
| Repository snapshot | `gosherpa snapshot` | Writes `.gosherpa/snapshot.json` with versioned file, package, symbol, build-tag, git-state, freshness, and relationship metadata |
| Shell completion | `gosherpa completion zsh` | Prints completion scripts for zsh, bash, or fish |
| Snapshot-backed analysis | `gosherpa analyze --use-snapshot` | Reuses a valid snapshot for repository overview inventory, selected relationship commands, and diff changed-symbol inventory where available, with live-analysis fallback warnings |
| Test-aware explanation | `gosherpa explain ParseFile --tests` | Includes test-file callers in the symbol profile on demand |
| Reference search | `gosherpa refs ParseFile --kind call` | Finds Go-aware definitions and references, with optional kind filtering |
| Impact analysis | `gosherpa impact ParseFile` | Summarizes references, caller-chain impact, affected packages, and suggested tests |
| Structured impact | `gosherpa impact file service.go` | Reports file, package, symbol, and diff impact through a shared report model |
| Diff impact | `gosherpa impact diff --base HEAD` | Reports changed files, changed packages, changed-symbol impact, affected packages, and affected tests |
| PR review | `gosherpa pr --base HEAD` | Summarizes changed files, packages, symbols, diff risk, target risk, repository risk, affected tests, and verification commands |
| Test discovery | `gosherpa tests ParseFile --scope direct` | Lists related tests for a symbol, package, or file and suggested `go test` commands with optional direct/related/all scope |
| Affected tests | `gosherpa tests affected --base HEAD` | Prints suggested test commands for a git diff |
| Machine-readable output | `gosherpa symbols --json` | Emits JSON for all commands with a stable response envelope |
| Package inventory | `gosherpa packages --tests` | Lists local packages with file, symbol, import, reverse-dependency, and test indicators |
| Package dependencies | `gosherpa deps ./internal/sherpa` | Shows imports and local dependents |
| Dependency overview | `gosherpa deps --all` | Shows local and external imports plus reverse dependencies for every package |
| Interface implementers | `gosherpa implementers ./internal/auth.Authenticator` | Lists concrete local types satisfying an interface |
| Interface profile | `gosherpa interface ./internal/auth.Authenticator` | Shows methods, implementers, type references, and visible interface method usage |
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
./gosherpa version
./gosherpa analyze . --tests
./gosherpa analyze --use-snapshot
./gosherpa analyze . --json
./gosherpa architecture
./gosherpa architecture --tests
./gosherpa architecture --json
./gosherpa risk
./gosherpa risk --tests
./gosherpa risk --json
./gosherpa symbols
./gosherpa symbols --kind struct
./gosherpa symbols --kind alias
./gosherpa symbols --kind method --package ./internal/sherpa
./gosherpa symbols --tests
./gosherpa symbols --use-snapshot
./gosherpa symbols --json
./gosherpa symbol ParseFile
./gosherpa symbol ParseFile --use-snapshot
./gosherpa symbol ParseFile --json
./gosherpa search parse file
./gosherpa search parse file --kind function --limit 5
./gosherpa search ParseFile --package ./internal/sherpa
./gosherpa search ParseFile --tests
./gosherpa search ParseFile --use-snapshot
./gosherpa search parse file --json
./gosherpa context symbol ParseFile
./gosherpa context symbol ParseFile --tests
./gosherpa context symbol ParseFile --use-snapshot --json
./gosherpa context symbol ParseFile --json
./gosherpa context symbol ParseFile --max-bytes 12000 --json
./gosherpa agent context --base HEAD
./gosherpa agent context --base HEAD --use-snapshot --json
./gosherpa agent context --base HEAD --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
./gosherpa context file internal/sherpa/impact.go
./gosherpa context file internal/sherpa/impact.go --json
./gosherpa context package ./internal/sherpa
./gosherpa context package ./internal/sherpa --json
./gosherpa context diff --base HEAD
./gosherpa context diff --base HEAD --use-snapshot --json
./gosherpa context diff --base HEAD --json
./gosherpa doctor
./gosherpa doctor --json
./gosherpa snapshot
./gosherpa snapshot --json
./gosherpa completion zsh
./gosherpa completion bash
./gosherpa completion fish
./gosherpa explain ParseFile
./gosherpa explain ParseFile --tests
./gosherpa explain ParseFile --json
./gosherpa refs ParseFile
./gosherpa refs ParseFile --kind call
./gosherpa refs ParseFile --use-snapshot
./gosherpa refs ParseFile --json
./gosherpa impact ParseFile
./gosherpa impact ParseFile --json
./gosherpa impact ./internal/sherpa
./gosherpa impact file internal/sherpa/impact.go
./gosherpa impact package ./internal/sherpa
./gosherpa impact symbol ParseFile
./gosherpa impact symbol ParseFile --use-snapshot
./gosherpa impact diff --base HEAD
./gosherpa impact diff --base HEAD --use-snapshot
./gosherpa impact diff --base HEAD --json
./gosherpa pr --base HEAD
./gosherpa pr --base HEAD --use-snapshot
./gosherpa pr --base HEAD --json
./gosherpa tests ParseFile
./gosherpa tests ParseFile --scope direct
./gosherpa tests ParseFile --scope all
./gosherpa tests ParseFile --json
./gosherpa tests ./internal/sherpa
./gosherpa tests internal/sherpa/impact.go
./gosherpa tests affected --base HEAD
./gosherpa tests affected --base HEAD --use-snapshot
./gosherpa tests affected --base HEAD --json
./gosherpa packages
./gosherpa packages --tests
./gosherpa packages --tests --use-snapshot
./gosherpa packages --json
./gosherpa deps ./internal/sherpa
./gosherpa deps --all
./gosherpa deps ./internal/sherpa --json
./gosherpa deps --all --json
./gosherpa implementers ./internal/auth.Authenticator
./gosherpa implementers ./internal/auth.Authenticator --use-snapshot
./gosherpa implementers ./internal/auth.Authenticator --json
./gosherpa interface ./internal/auth.Authenticator
./gosherpa interface ./internal/auth.Authenticator --use-snapshot
./gosherpa interface ./internal/auth.Authenticator --json
./gosherpa interfaces ./internal/jwt.JWTAuthenticator
./gosherpa interfaces ./internal/jwt.JWTAuthenticator --use-snapshot
./gosherpa interfaces ./internal/jwt.JWTAuthenticator --json
./gosherpa callees ParseFile
./gosherpa callees ./internal/sherpa.ParseFile
./gosherpa callees ParseFile --use-snapshot
./gosherpa callees ParseFile --json
./gosherpa callers ParseFile
./gosherpa callers ParseFile --tests
./gosherpa callers ./internal/sherpa.ParseFile
./gosherpa callers ParseFile --use-snapshot
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

`callers --json` and `callees --json` keep direct caller/callee arrays separate
from additive `possibleCalls` entries. Possible calls are conservative runtime
signals such as interface dispatch, goroutine starts, function literals, and
function values; they do not change direct call counts. Interface-dispatch
entries are emitted only when typechecked analysis can name a bounded set of
known local implementer methods. Unknown or broad interface dispatch remains a
limitation rather than a guessed edge. Visible local targets in direct
goroutine starts, goroutine function literals, immediately invoked function
literals, and function literals passed to simple local call sites are reported
as possible calls with source ranges; reassigned or escaping function values
remain conservative.

`tests <target>` supports `--scope direct|related|all` for symbol, package, or
file targets. It intentionally does not accept `--base`, `--tags`, or
`--use-snapshot`. `tests affected --base <ref>` is diff-oriented, supports
`--tags` and `--use-snapshot`, and intentionally does not accept `--scope`.
Both commands expose grouped `testPlan` recommendations in JSON and grouped
human output with `direct`, `related`, `contracts`, `callerPackages`, and
`fallback` sections. Diff fallbacks are package-level when Go packages are
known and whole-repository (`go test ./...`) when a change cannot be narrowed
to repository-local Go packages.

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

Context commands support command-specific size controls for agent workflows:

| Command | Supported limits |
| --- | --- |
| `agent context` | `--use-snapshot`, `--tags`, `--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes` |
| `context symbol` | `--use-snapshot`, `--max-references`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context file` | `--max-symbols`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context package` | `--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes`, `--source-radius` |
| `context diff` | `--max-files`, `--max-symbols`, `--max-tests`, `--max-bytes` |

Unsupported context-limit combinations fail as usage errors. `agent context`
does not accept `--max-references`, `--source-radius`, `--scope`, or `--tests`.
The context byte budget omits large context fields deterministically and keeps
JSON valid; workflow-level omissions are reported in `data.truncated` and
per-section omissions in `data.sectionTruncation`.

Diff-oriented JSON such as `impact diff`, `tests affected`, `pr`, and
`context diff` can report `git-diff+typechecked+ast` plus
`referenceAnalysisMode` and `callAnalysisMode` when changed-symbol impact uses
typechecked package loading. `context diff` and `pr` include
`changedSymbolDetails` and put changed-symbol locations first in
`readingOrder` when positions are known. With `--use-snapshot` on
`agent context`, `context diff`, `impact diff`, `tests affected`, or `pr`, a
valid snapshot is reused for current changed-symbol inventory and selected
persisted relationship records, reporting `snapshot+git-diff+typechecked+ast`
or `snapshot+git-diff+ast`. With
`--use-snapshot` on `refs`, `callers`, `callees`, `implementers`, `interface`,
`interfaces`, `context symbol`, or `impact symbol`, valid relationship records
report `snapshot+typechecked`, `snapshot+ast-fallback`, or `snapshot`
depending on how the snapshot was built.

`snapshot --json` returns a bounded summary of the persisted snapshot:
`formatVersion`, environment and freshness inputs, `fileCount`,
`packageCount`, `symbolCount`, and `relationshipMetadata` with stable
`countsByKind`. It intentionally does not dump the persisted file, package,
symbol, or relationship records.

`pr --json` keeps the diff-oriented `risk` summary, adds `targetRisk` for the
specific diff blast-radius judgment, and also includes `repositoryRisk`, the
full structural `RiskReport` from `gosherpa risk`.

`agent context --json` uses `command: "agent context"` and `target` set to the
base ref. Its `data` object is a composed summary with readiness, snapshot
status, changed targets, reading order, target risk, possible runtime
relationship summaries, interface summary, test plan, suggested commands,
section modes, confidence, limitations, limits, and truncation metadata. It is
not a dump of the full `context diff`, `impact diff`, `tests affected`, and
`pr` reports.

`analyze --json` provides the repository-level entry point for agents and
scripts: package summaries, symbol counts, important public symbols,
entrypoint candidates, structural risk, simple hotspots, test overview,
readiness, limitations, and suggested next commands.

See the [Agent JSON Schema](product/JSON_SCHEMA_V1.md) and [Context JSON Schema](product/CONTEXT_SCHEMA_V1.md) for the full machine-readable contracts.
