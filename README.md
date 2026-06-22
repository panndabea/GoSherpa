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

## Why It Exists

As Go projects grow, the important answer is often spread across files, packages, and call sites. GoSherpa turns common navigation questions into focused commands:

| When you ask... | GoSherpa helps you find... |
| --- | --- |
| "Where is this thing defined?" | Symbols, files, and line numbers |
| "Who uses this function?" | References and direct callers |
| "What does this function touch?" | Direct callees |
| "What depends on this package?" | Local package dependency relationships |
| "What might this change affect?" | The future impact-analysis workflow |

## Current Experience

| Capability | Command | Result |
| --- | --- | --- |
| Symbol atlas | `gosherpa symbols` | Lists discovered structs, interfaces, functions, and methods |
| Symbol lookup | `gosherpa symbol ParseFile` | Shows a definition with kind, file, and line |
| Reference search | `gosherpa refs ParseFile` | Finds syntactic references across the repository |
| Package dependencies | `gosherpa deps ./internal/sherpa` | Shows imports and local dependents |
| Callees | `gosherpa callees ParseFile` | Lists direct calls made by a function or method |
| Callers | `gosherpa callers ParseFile` | Lists direct syntactic callers of a function or method |

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

Then explore the repository from the project root:

```bash
./gosherpa symbols
./gosherpa symbol ParseFile
./gosherpa refs ParseFile
./gosherpa deps ./internal/sherpa
./gosherpa callees ParseFile
./gosherpa callers ParseFile
```

Prefer not to build a binary yet?

```bash
go run ./cmd/gosherpa symbols
go run ./cmd/gosherpa callers ParseFile
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

GoSherpa is an early MVP, with the next work focused on deeper navigation and safer changes.

| Next | Goal |
| --- | --- |
| Call paths | Find paths between two functions or methods |
| Better references | Move from syntactic matches toward more Go-aware relationships |
| Impact analysis | Estimate what a change might affect before making it |
| Test discovery | Find tests related to packages, types, and functions |
| Machine-readable output | Add a stable surface for scripts and agents once the concepts settle |

Read the full product plan in [FEATURE_ROADMAP.md](FEATURE_ROADMAP.md).

## Status

Implemented today:

- Repository scanning and Go file discovery
- Struct, interface, function, and method discovery
- Symbol lookup and reference lookup
- Package dependency analysis
- Direct syntactic caller and callee analysis

Known MVP limitations:

- Reference search is syntactic, not type-checked.
- Caller and callee analysis is AST-based and can miss receiver-variable method calls.
- One-segment function targets can produce selector-call false positives.
- Function names can be ambiguous across packages.

## Philosophy

GoSherpa focuses on visibility, not magic.

It does not try to become your IDE, rewrite your project, or infer more certainty than it has. It reads the code you already have and returns compact answers that help you keep moving.
