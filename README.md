# GoSherpa

GoSherpa helps you explore and understand Go codebases.

It provides fast access to symbols, references, package relationships, and eventually call paths, dependency analysis, and impact discovery across a repository.

## Why?

As Go projects grow, answering simple questions becomes harder:

- Where is this function used?
- Where is this type defined?
- Which packages depend on this one?
- What implements this interface?
- Which tests are affected by a change?
- What code paths lead here?

GoSherpa helps answer these questions without manually jumping through dozens of files.

## Current Features

### List Symbols

Explore all discovered symbols in a repository.

```bash
gosherpa symbols
```

Example output:

```text
📦 STRUCTS
  Symbol                               internal/sherpa/symbol.go:17
  Position                             internal/sherpa/symbol.go:12

⚙️ FUNCTIONS
  ParseFile                            internal/sherpa/parse.go:9
  FindGoFiles                          internal/sherpa/scan.go:8
```

### Symbol Lookup

Find a symbol and show where it is defined.

```bash
gosherpa symbol ParseFile
```

Example:

```text
Name: ParseFile
Kind: function
File: internal/sherpa/parse.go
Line: 9
```

### Reference Search

Find references to a symbol across the repository.

```bash
gosherpa refs ParseFile
```

Example:

```text
🔍 REFERENCES

ParseFile

  internal/sherpa/repository.go:12
  internal/sherpa/reference.go:41

Found 4 references
```

### Package Dependencies

Show imports and local packages that depend on a package.

```bash
gosherpa deps ./internal/sherpa
```

## Roadmap

Planned features:

### Call Paths

Inspect incoming and outgoing function calls.

```bash
gosherpa callers UserService.Create
gosherpa callees UserService.Create
```

### Impact Analysis

Estimate the scope of a change before making it.

```bash
gosherpa impact internal/auth/session.go
```

### Test Discovery

Find tests related to packages, types, and functions.

```bash
gosherpa tests UserService.Create
```

## Example

```bash
gosherpa symbols
gosherpa symbol ParseFile
gosherpa refs ParseFile
gosherpa deps ./internal/sherpa
```

## Status

Early MVP.

Implemented:

- Repository scanning
- Go file discovery
- Struct discovery
- Interface discovery
- Function discovery
- Method discovery
- Symbol lookup
- Reference lookup
- Package dependency analysis

In progress:

- Better reference analysis
- Call graph analysis

## Philosophy

GoSherpa focuses on information already present in the codebase.

No annotations.
No code generation.
No magic.

Just better visibility into how a Go project is put together.
