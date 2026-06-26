# GoSherpa Impact Engine v0.1

Status: MVP completed on 2026-06-26.

GoSherpa v0.1 turns the existing code explorer into a conservative Change
Intelligence CLI for Go repositories. The release answers the core
[PRD_V01](../product/PRD_V01.md) questions: what changed, which packages are
affected, which interfaces or implementations may be involved, and which tests
are worth running.

## Implemented

- `gosherpa impact file <file>`
- `gosherpa impact package <package>`
- `gosherpa impact symbol <package.Symbol>`
- `gosherpa impact diff --base <ref>`
- `gosherpa tests affected --base <ref>`
- Shared `internal/impact.ImpactReport`
- Git changed-file and hunk-range reading in `internal/git`
- Changed-package mapping for Go files
- Changed-symbol extraction from diff hunks
- Deletion-aware changed-symbol extraction from base refs
- Affected package reporting
- Affected interface and implementation reporting
- Import-aware interface method signature matching
- Embedded-interface expansion for local interfaces
- Package-qualified symbol impact for references and tests
- Transitive caller packages for symbol impact
- Affected-test planning and suggested `go test` commands
- Human and JSON output for impact/test commands
- Focused unit tests plus golden JSON coverage

## Conservative Boundaries

- Analysis is local and AST-based.
- Interface impact does not run the full Go type checker.
- Diff symbols are hunk-based and focused on top-level Go functions and
  struct/interface types.
- Test planning remains package-oriented, with existing direct-reference
  signals where available.
- No SSA, pointer analysis, build-tag matrix, GitHub Action, MCP, IDE
  integration, AI integration, SQLite snapshot, or watcher is included in v0.1.

## Verification

- `go test ./internal/git ./internal/impact`
- `go test ./...`
- `git diff --check`

## Next

[PRD_V02](../product/PRD_V02.md) moves GoSherpa from impact reports toward
Symbol Intelligence. The highest-leverage next command is `gosherpa explain`, a
compact profile for one symbol that can serve humans and AI coding tools without
making GoSherpa a code generator.
