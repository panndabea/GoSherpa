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
| "What can reach this internal code?" | Runtime, test, exported, and no-local-caller entrypoints |
| "What depends on this package?" | Local package imports and dependents |
| "What might this change affect?" | Changed symbols, affected packages, and suggested tests |

## Try It

```bash
git clone https://github.com/panndabea/GoSherpa.git
cd GoSherpa
go test ./...
go run ./cmd/gosherpa analyze .
go run ./cmd/gosherpa doctor
go run ./cmd/gosherpa snapshot
go run ./cmd/gosherpa symbols --kind function
go run ./cmd/gosherpa explain ParseFile
```

Build a local binary when you are ready:

```bash
go build -o gosherpa ./cmd/gosherpa
./gosherpa search parse file
./gosherpa refs ParseFile --kind call
./gosherpa entrypoints ParseFile
./gosherpa impact diff --base HEAD
```

Use `--root` to run GoSherpa from another working directory:

```bash
./gosherpa --root /path/to/GoSherpa symbols
```

## Common Commands

| Command | What it shows |
| --- | --- |
| `gosherpa analyze .` | Repository overview with packages, symbols, hotspots, tests, readiness, and next commands |
| `gosherpa doctor` | Analysis readiness, Go environment, warnings, and confidence |
| `gosherpa snapshot` | Writes a versioned `.gosherpa/snapshot.json` repository inventory snapshot |
| `gosherpa symbols --kind function --package ./internal/sherpa` | Structs, interfaces, functions, and methods with optional kind, package, and test filters |
| `gosherpa symbol ParseFile` | One symbol's package, signature, docs, and source location |
| `gosherpa search parse file` | Ranked partial symbol matches |
| `gosherpa explain ParseFile` | Purpose, risk, relationships, reading order, and test signals |
| `gosherpa refs ParseFile --kind call` | Go-aware references filtered by kind |
| `gosherpa callers ParseFile` | Direct callers of a function or method |
| `gosherpa entrypoints ParseFile` | Public, runtime, test, and no-local-caller functions that can reach a target |
| `gosherpa impact diff --base HEAD` | Changed files, affected packages, and suggested tests |
| `gosherpa pr --base HEAD` | PR-style change summary with risk and verification commands |

For the full command list, examples, JSON output shape, and agent-oriented context commands, see the [CLI reference](docs/CLI_REFERENCE.md).

## For Agents And Automation

Every command supports machine-readable JSON through `--json` using a stable response envelope. GoSherpa also includes context commands for symbol, file, package, and diff targets:

```bash
gosherpa context symbol ParseFile --json
gosherpa context file internal/sherpa/impact.go --json
gosherpa context package ./internal/sherpa --json
gosherpa context diff --base HEAD --json
```

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
