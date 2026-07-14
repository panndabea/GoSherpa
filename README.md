<div align="center">
  <img src="pictures/gosherpa-readme-hero.png" alt="GoSherpa explorer mascot with a mountain map" width="900">

  <h1>GoSherpa</h1>

  <p>
    <strong>Structural code intelligence for Go projects.</strong><br>
    Explore symbols, references, packages, callers, and callees from a calm, deterministic CLI.
  </p>

  <p>
    <a href="go.mod"><img alt="Go 1.24.4" src="https://img.shields.io/badge/Go-1.24.4-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white&amp;labelColor=10242C"></a>
    <a href="https://github.com/panndabea/GoSherpa/actions/workflows/ci.yml"><img alt="CI tests" src="https://img.shields.io/github/actions/workflow/status/panndabea/GoSherpa/ci.yml?branch=main&amp;label=tests&amp;style=flat-square&amp;logo=githubactions&amp;logoColor=white&amp;labelColor=10242C"></a>
    <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/badge/license-MIT-F3B64B?style=flat-square&amp;labelColor=10242C"></a>
  </p>
</div>

---

> Ask a code-structure question. Get a small, trustworthy answer.

GoSherpa is a fast command-line companion for understanding Go repositories. It helps you find where a symbol lives, who uses it, what a function calls, which packages depend on each other, and what a change might affect.

It is built for developers and coding agents who want focused, repeatable answers without opening half the project in an editor.

## What You Can Ask

| When you ask... | GoSherpa helps you find... |
| --- | --- |
| "Where is this thing defined?" | Symbols, files, signatures, and line numbers |
| "Who uses this function?" | References and direct callers |
| "What does this function touch?" | Direct callees and local call paths |
| "Which parts of this interface are used?" | Methods, implementers, type references, and visible interface method calls |
| "What can reach this internal code?" | Runtime, test, exported, and no-local-caller entrypoints |
| "What depends on this package?" | Local package imports and dependents |
| "Which tests cover this file?" | Direct file-symbol references, same-package tests, and suggested commands |
| "What might this change affect?" | Changed symbols, affected packages, and suggested tests |

## Try It

```bash
git clone https://github.com/panndabea/GoSherpa.git
cd GoSherpa
go run ./cmd/gosherpa version
go run ./cmd/gosherpa search parse file
go run ./cmd/gosherpa symbol ParseFile
go run ./cmd/gosherpa refs ParseFile --kind call
go run ./cmd/gosherpa tests internal/sherpa/parse.go
```

For a broader repository overview, run:

```bash
go run ./cmd/gosherpa analyze .
go run ./cmd/gosherpa doctor
```

Build a local binary when you are ready for repeated use:

```bash
go build -o gosherpa ./cmd/gosherpa
./gosherpa snapshot
./gosherpa analyze --use-snapshot
./gosherpa symbols --use-snapshot
./gosherpa context diff --base HEAD --use-snapshot --json
./gosherpa completion zsh
./gosherpa entrypoints ParseFile
./gosherpa impact diff --base HEAD --use-snapshot
```

When contributing to GoSherpa itself, run the full test suite:

```bash
go test ./...
```

Use `--root` to run GoSherpa from another working directory:

```bash
./gosherpa --root /path/to/GoSherpa symbols
```

## Common Commands

| Command | What it shows |
| --- | --- |
| `gosherpa version` | GoSherpa and Go runtime version information |
| `gosherpa analyze .` | Repository overview with packages, symbols, hotspots, tests, readiness, and next commands |
| `gosherpa doctor` | Analysis readiness, Go environment, warnings, and confidence |
| `gosherpa snapshot` | Writes a versioned `.gosherpa/snapshot.json` repository inventory snapshot |
| `gosherpa completion zsh` | Prints shell completion scripts for zsh, bash, or fish |
| `gosherpa analyze --use-snapshot` | Reuses a valid snapshot for repository overview inventory when available |
| `gosherpa symbols --use-snapshot` | Reuses a valid snapshot for symbol inventory queries, with live-analysis fallback warnings |
| `gosherpa symbols --kind function --package ./internal/sherpa` | Structs, interfaces, type aliases, functions, and methods with optional kind, package, and test filters |
| `gosherpa symbol ParseFile` | One symbol's package, signature, docs, and source location |
| `gosherpa search parse file` | Ranked partial symbol matches |
| `gosherpa explain ParseFile` | Purpose, risk, relationships, reading order, and test signals |
| `gosherpa refs ParseFile --kind call` | Go-aware references filtered by kind |
| `gosherpa callers ParseFile` | Direct callers of a function or method |
| `gosherpa interface ./internal/auth.Authenticator` | Interface methods, implementers, type references, and visible interface method usage |
| `gosherpa entrypoints ParseFile` | Public, runtime, test, and no-local-caller functions that can reach a target |
| `gosherpa tests internal/sherpa/parse.go` | Related tests and commands for a changed file |
| `gosherpa impact diff --base HEAD --use-snapshot` | Changed files, affected packages, and suggested tests, with valid snapshot reuse for current changed-symbol inventory |
| `gosherpa pr --base HEAD --use-snapshot` | PR-style change summary with risk and verification commands, with valid snapshot reuse where available |

For the full command list, examples, JSON output shape, and agent-oriented context commands, see the [CLI reference](docs/CLI_REFERENCE.md).

## For Agents And Automation

Every command supports machine-readable JSON through `--json` using a stable response envelope. GoSherpa also includes context commands for symbol, file, package, and diff targets:

```bash
gosherpa context symbol ParseFile --json
gosherpa context file internal/sherpa/impact.go --json
gosherpa context package ./internal/sherpa --json
gosherpa context diff --base HEAD --use-snapshot --json
```

### Agent Quickstart

Agents should start with bounded, task-specific context instead of broad inventory dumps.

```bash
# 1. Check whether analysis is trustworthy
gosherpa doctor --json

# 2. For a change, start with bounded diff context
gosherpa context diff --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json

# 3. For a symbol task, fetch focused context
gosherpa context symbol ParseFile --max-references 20 --max-tests 10 --max-bytes 12000 --json

# 4. For impact and test planning
gosherpa impact diff --base HEAD --use-snapshot --json
gosherpa tests affected --base HEAD --use-snapshot --json
```

Agent workflow rules:

- Use `--json` for automation and agent workflows.
- Prefer `context ...` commands before editing files.
- Always bound large context commands with `--max-*` flags.
- Avoid broad inventory dumps like unfiltered `symbols` unless you really need them.
- Use `search --limit <n>` to discover targets, then switch to `symbol`, `context symbol`, `refs`, `callers`, or `callees`.
- If a target is ambiguous, rerun with the package-qualified target GoSherpa suggests.
- Add `--tests` only when test code matters.

Start with [GoSherpa for AI Agents](AGENT_NOTES.md) when deciding whether the tool is useful for a repository or task.

See the [Agent JSON Schema](docs/product/JSON_SCHEMA_V1.md) and [Context JSON Schema](docs/product/CONTEXT_SCHEMA_V1.md) for the detailed contracts.

## Status

GoSherpa is an early MVP. The Impact Engine v0.1 track is implemented, including file, package, symbol, and diff impact analysis. Current work continues the Symbol Intelligence track: deeper explanations, better context exports, and more precise semantic references.

Read the [implementation status](docs/STATUS.md), [release notes](docs/releases/RELEASE_NOTES_V01.md), or [feature roadmap](docs/product/FEATURE_ROADMAP.md) for the full details.

## Design Principles

| Principle | What it means in practice |
| --- | --- |
| Human-readable first | Output is meant to be scanned in a terminal, not decoded from a dump |
| Deterministic by default | Stable answers make it easier to trust diffs and repeated runs |
| Small commands | Each command answers one navigation question clearly |
| Repository-native | GoSherpa uses information already present in the codebase |
| No hidden ceremony | No annotations, no generated metadata, no project-specific setup for normal use |

## Philosophy

GoSherpa focuses on visibility, not magic.

It does not try to become your IDE, rewrite your project, or infer more certainty than it has. It reads the code you already have and returns compact answers that help you keep moving.

## License

GoSherpa is available under the [MIT License](LICENSE).
