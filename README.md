<div align="center">
  <img src="pictures/gosherpa-readme-hero.png" alt="GoSherpa explorer mascot with a mountain map" width="900">

  <h1>GoSherpa</h1>

  <p>
    <strong>Agent-first structural intelligence for Go repositories.</strong><br>
    Give coding agents and developers bounded, deterministic context before changing Go code.
  </p>

  <p>
    <a href="go.mod"><img alt="Go 1.24.4" src="https://img.shields.io/badge/Go-1.24.4-00ADD8?style=flat-square&amp;logo=go&amp;logoColor=white&amp;labelColor=10242C"></a>
    <a href="https://github.com/panndabea/GoSherpa/actions/workflows/ci.yml"><img alt="CI tests" src="https://img.shields.io/github/actions/workflow/status/panndabea/GoSherpa/ci.yml?branch=main&amp;label=tests&amp;style=flat-square&amp;logo=githubactions&amp;logoColor=white&amp;labelColor=10242C"></a>
    <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/badge/license-MIT-F3B64B?style=flat-square&amp;labelColor=10242C"></a>
  </p>
</div>

---

> Give an agent a repository question. Get a small, trustworthy map.

GoSherpa is an agent-first command-line intelligence layer for Go repositories.
It helps coding agents and developers answer focused questions before editing:
where code lives, who uses it, what it calls, which entrypoints may reach it,
which packages and tests are nearby, and what a change might affect.

Agent-first keeps the human CLI first-class. GoSherpa is intentionally boring
in the good way: local analysis, deterministic output, readable terminal
tables, and stable JSON envelopes for automation.

## What You Can Ask

| Question | Start with |
| --- | --- |
| What should an agent read before editing? | `gosherpa init --base <ref>`, then `gosherpa agent context --json` |
| What shape is this repository in? | `gosherpa doctor`, `gosherpa analyze .` |
| Where is this symbol defined? | `gosherpa search <terms>`, `gosherpa symbol <target>` |
| Who uses this function or type? | `gosherpa refs <target>`, `gosherpa callers <target>` |
| What does this function call? | `gosherpa callees <target>`, `gosherpa path <from> <to>` |
| Which interfaces and implementers are involved? | `gosherpa interface <interface>`, `gosherpa implementers <interface>` |
| Which tests are related? | `gosherpa tests <target>`, `gosherpa tests affected --base <ref>` |
| What might this change affect? | `gosherpa agent context` after init, `gosherpa impact diff --base <ref>` |

## Quickstart

```bash
git clone https://github.com/panndabea/GoSherpa.git
cd GoSherpa
go run ./cmd/gosherpa version
go run ./cmd/gosherpa init --base HEAD --json
go run ./cmd/gosherpa doctor --json
go run ./cmd/gosherpa agent context --json
go run ./cmd/gosherpa pr --json
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
./gosherpa init --base HEAD
./gosherpa analyze --use-snapshot
./gosherpa context symbol ParseFile --use-snapshot --json
./gosherpa agent context --json
./gosherpa impact diff --base HEAD --use-snapshot
./gosherpa pr --json
```

When contributing to GoSherpa itself, run:

```bash
go test ./...
```

Use `--root` to inspect another Go module or workspace from your current
directory:

```bash
./gosherpa --root /path/to/project analyze .
./gosherpa --root /path/to/project init --base HEAD
./gosherpa --root /path/to/project agent context --json
```

## Common Workflows

For the default agent-first pre-edit flow:

```bash
gosherpa init --base HEAD
gosherpa doctor --json
gosherpa agent context --json
gosherpa pr --json
```

`init` writes `.gosherpa/config.json` and refreshes
`.gosherpa/snapshot.json`. Initialized repositories can omit `--base`,
snapshot reuse, build tags, and standard agent-context limits for
`agent context`, `pr`, and `tests affected`. Use `--no-use-snapshot` on those
commands when you want live analysis for a single run.

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
gosherpa agent context --json
gosherpa impact diff --base HEAD --json
gosherpa tests affected --json
gosherpa pr --json
```

For GitHub pull requests, GoSherpa can also run as a composite Action. The
Action runs `gosherpa init`, writes a step summary, and uploads JSON artifacts
for `init`, `doctor`, `agent context`, `pr`, and `tests affected`:

```yaml
- uses: panndabea/GoSherpa@main
  with:
    base-ref: ${{ github.event.pull_request.base.sha }}
```

Use `actions/checkout` with `fetch-depth: 0` before running the Action so
GoSherpa can compare the pull request with its base.

For faster repeated queries, create a snapshot and opt into reuse:

```bash
gosherpa init --base HEAD
gosherpa symbols --use-snapshot
gosherpa refs ParseFile --use-snapshot --json
gosherpa callers ParseFile --use-snapshot --json
gosherpa interface ./internal/auth.Authenticator --use-snapshot --json
gosherpa agent context --json
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
- `agent context`, `context diff`, `impact diff`, `tests affected`, and `pr` for current
  changed-symbol inventory and selected relationship subanalysis

If a snapshot is missing, stale, invalid, or does not contain the relationship
records a command needs, GoSherpa falls back to live analysis and reports a
warning.

## For Agents

Agents should prefer bounded, task-specific context over broad inventory dumps:

```bash
gosherpa init --base HEAD
gosherpa doctor --json
gosherpa agent context --json
gosherpa context symbol ./internal/sherpa.ParseFile --use-snapshot --max-references 20 --max-tests 10 --max-bytes 12000 --json
gosherpa tests affected --json
```

Start with [GoSherpa for AI Agents](AGENT_NOTES.md) for workflow guidance, and
use the [Agent JSON Schema](docs/product/JSON_SCHEMA_V1.md) and
[Context JSON Schema](docs/product/CONTEXT_SCHEMA_V1.md) for machine-readable
contracts.

## Status

GoSherpa v0.8.0 is a pre-1.0 agent-first CLI release. The original Impact
Engine v0.1 track is implemented, Symbol Intelligence is part of the baseline,
and the current product surface centers on bounded repository context, impact,
test planning, PR summaries, snapshots, and honest uncertainty for ordinary Go
repositories.

Read the [implementation status](docs/STATUS.md), [v0.8.0 release notes](docs/releases/RELEASE_NOTES_V08.md),
or [feature roadmap](docs/product/FEATURE_ROADMAP.md) for the current product
state.

## Design Principles

| Principle | What it means |
| --- | --- |
| Agent-first workflow | The daily driver should give coding agents bounded context before edits |
| Human-readable too | Terminal output should be scannable before it is exhaustive |
| Deterministic by default | Repeated runs should produce stable, trustworthy answers |
| Focused commands | Each command should answer one navigation question clearly |
| Repository-native | GoSherpa uses information already present in the codebase |
| Honest limits | Warnings, confidence, and limitations are part of the output contract |

## License

GoSherpa is available under the [MIT License](LICENSE).
