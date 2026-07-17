<div align="center">
  <img src="pictures/gosherpa-readme-hero.png" alt="GoSherpa explorer mascot with a mountain map" width="900">

  <h1>GoSherpa</h1>

  <p>
    <strong>Structural code intelligence for Go projects.</strong><br>
    Explore symbols, references, packages, callers, callees, tests, and diff impact from a calm, deterministic CLI.
  </p>

  <p>
    <a href="go.mod"><img alt="Go 1.24.4" src="https://img.shields.io/badge/Go-1.24.4-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white&amp;labelColor=10242C"></a>
    <a href="https://github.com/panndabea/GoSherpa/actions/workflows/ci.yml"><img alt="CI tests" src="https://img.shields.io/github/actions/workflow/status/panndabea/GoSherpa/ci.yml?branch=main&amp;label=tests&amp;style=flat-square&amp;logo=githubactions&amp;logoColor=white&amp;labelColor=10242C"></a>
    <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/badge/license-MIT-F3B64B?style=flat-square&amp;labelColor=10242C"></a>
  </p>
</div>

---

> Ask a code-structure question. Get a small, trustworthy answer.

GoSherpa is a command-line companion for understanding Go repositories before
you edit them. It helps developers and coding agents answer focused questions
about where code lives, who uses it, what it calls, which packages and tests are
nearby, and what a change might affect.

It is intentionally boring in the good way: local analysis, deterministic
output, human-readable tables by default, and JSON when you want automation.

## What You Can Ask

| Question | Start with |
| --- | --- |
| What shape is this repository in? | `gosherpa doctor`, `gosherpa analyze .` |
| Where is this symbol defined? | `gosherpa search <terms>`, `gosherpa symbol <target>` |
| Who uses this function or type? | `gosherpa refs <target>`, `gosherpa callers <target>` |
| What does this function call? | `gosherpa callees <target>`, `gosherpa path <from> <to>` |
| Which interfaces and implementers are involved? | `gosherpa interface <interface>`, `gosherpa implementers <interface>` |
| Which tests are related? | `gosherpa tests <target>`, `gosherpa tests affected --base <ref>` |
| What might this change affect? | `gosherpa context diff --base <ref>`, `gosherpa impact diff --base <ref>` |
| What should an agent read before editing? | `gosherpa context symbol|file|package|diff ... --json` |

## Quickstart

```bash
git clone https://github.com/panndabea/GoSherpa.git
cd GoSherpa
go run ./cmd/gosherpa version
go run ./cmd/gosherpa doctor
go run ./cmd/gosherpa analyze .
```

Try a few focused questions:

```bash
go run ./cmd/gosherpa search parse file
go run ./cmd/gosherpa symbol ParseFile
go run ./cmd/gosherpa refs ParseFile --kind call
go run ./cmd/gosherpa callers ParseFile
go run ./cmd/gosherpa tests internal/sherpa/parse.go
```

Build a local binary when you want repeated use:

```bash
go build -o gosherpa ./cmd/gosherpa
./gosherpa snapshot
./gosherpa analyze --use-snapshot
./gosherpa context symbol ParseFile --use-snapshot --json
./gosherpa impact diff --base HEAD --use-snapshot
./gosherpa pr --base HEAD --use-snapshot --json
```

When contributing to GoSherpa itself, run:

```bash
go test ./...
```

Use `--root` to inspect another Go module or workspace from your current
directory:

```bash
./gosherpa --root /path/to/project analyze .
./gosherpa --root /path/to/project context diff --base HEAD --json
```

## Common Workflows

For repository orientation:

```bash
gosherpa doctor
gosherpa analyze .
gosherpa architecture
gosherpa risk
```

For focused symbol work:

```bash
gosherpa search parse file --limit 10
gosherpa context symbol ./internal/sherpa.ParseFile --max-references 20 --max-tests 10 --json
gosherpa refs ./internal/sherpa.ParseFile --json
gosherpa callers ./internal/sherpa.ParseFile --json
gosherpa callees ./internal/sherpa.ParseFile --json
```

For a change or pull request:

```bash
gosherpa context diff --base HEAD --max-files 20 --max-symbols 40 --max-tests 20 --json
gosherpa impact diff --base HEAD --json
gosherpa tests affected --base HEAD --json
gosherpa pr --base HEAD --json
```

For faster repeated queries, create a snapshot and opt into reuse:

```bash
gosherpa snapshot
gosherpa symbols --use-snapshot
gosherpa refs ParseFile --use-snapshot --json
gosherpa callers ParseFile --use-snapshot --json
gosherpa interface ./internal/auth.Authenticator --use-snapshot --json
gosherpa context diff --base HEAD --use-snapshot --json
```

## Command Map

| Area | Commands |
| --- | --- |
| Readiness and inventory | `doctor`, `analyze`, `snapshot`, `symbols`, `symbol`, `search`, `packages` |
| Repository structure | `architecture`, `risk`, `deps` |
| Relationships | `refs`, `callers`, `callees`, `path`, `paths`, `entrypoints` |
| Interfaces | `implementers`, `interface`, `interfaces` |
| Context exports | `context symbol`, `context file`, `context package`, `context diff` |
| Impact and tests | `impact`, `impact file`, `impact package`, `impact symbol`, `impact diff`, `tests`, `tests affected`, `pr` |
| Shell integration | `completion zsh`, `completion bash`, `completion fish` |

For every command, flag, example, and JSON note, see the
[CLI reference](docs/CLI_REFERENCE.md).

## JSON And Snapshots

Analysis commands support `--json` through a stable response envelope with the
command, target, root, module path, warnings, and command-specific `data`.
Context and diff workflows also expose confidence, limitations, reading order,
target-risk evidence, and truncation metadata where relevant.

`gosherpa snapshot` writes `.gosherpa/snapshot.json`, a reusable repository
inventory and bounded relationship snapshot. A valid snapshot can currently be
reused by:

- `analyze`, `symbols`, `symbol`, `search`, and `packages --tests`
- `refs`, `callers`, `callees`, `implementers`, `interface`, and `interfaces`
- `context symbol` and `impact symbol`
- `context diff`, `impact diff`, `tests affected`, and `pr` for current
  changed-symbol inventory and selected relationship subanalysis

If a snapshot is missing, stale, invalid, or does not contain the relationship
records a command needs, GoSherpa falls back to live analysis and reports a
warning.

## For Agents

Agents should prefer bounded, task-specific context over broad inventory dumps:

```bash
gosherpa doctor --json
gosherpa context diff --base HEAD --use-snapshot --max-files 20 --max-symbols 40 --max-tests 20 --max-bytes 12000 --json
gosherpa context symbol ./internal/sherpa.ParseFile --use-snapshot --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa tests affected --base HEAD --use-snapshot --json
```

Start with [GoSherpa for AI Agents](AGENT_NOTES.md) for workflow guidance, and
use the [Agent JSON Schema](docs/product/JSON_SCHEMA_V1.md) and
[Context JSON Schema](docs/product/CONTEXT_SCHEMA_V1.md) for machine-readable
contracts.

## Status

GoSherpa is an early MVP. The Impact Engine v0.1 track is implemented, and
current work focuses on deeper symbol intelligence, reusable relationship data,
runtime-aware possible call signals, and target-aware impact summaries.

Read the [implementation status](docs/STATUS.md), [release notes](docs/releases/RELEASE_NOTES_V01.md),
or [feature roadmap](docs/product/FEATURE_ROADMAP.md) for the current product
state.

## Design Principles

| Principle | What it means |
| --- | --- |
| Human-readable first | Terminal output should be scannable before it is exhaustive |
| Deterministic by default | Repeated runs should produce stable, trustworthy answers |
| Focused commands | Each command should answer one navigation question clearly |
| Repository-native | GoSherpa uses information already present in the codebase |
| Honest limits | Warnings, confidence, and limitations are part of the output contract |

## License

GoSherpa is available under the [MIT License](LICENSE).
