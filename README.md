# GoSherpa

GoSherpa helps you explore and understand Go codebases.

It provides fast access to symbol relationships, call paths, package dependencies and change impact across a repository.

## Why?

As Go projects grow, answering simple questions becomes harder:

* Where is this function used?
* Which packages depend on this one?
* What implements this interface?
* Which tests are affected by a change?
* What code paths lead here?

GoSherpa helps answer these questions without manually jumping through files.

## Features

### Symbol Lookup

Find definitions, references and implementations.

```bash
gosherpa symbol UserService
```

### Call Paths

Inspect incoming and outgoing function calls.

```bash
gosherpa callers UserService.Create
gosherpa callees UserService.Create
```

### Package Dependencies

Understand how packages relate to each other.

```bash
gosherpa deps ./internal/auth
```

### Impact Analysis

Estimate the scope of a change before making it.

```bash
gosherpa impact internal/auth/session.go
```

### Test Discovery

Find tests related to packages, types and functions.

```bash
gosherpa tests UserService.Create
```

## Example

```bash
gosherpa index .
gosherpa symbol UserService
gosherpa callers UserService.Create
gosherpa impact ./internal/auth
```

## Status

Early development.

Current focus:

* Repository indexing
* Symbol relationships
* Call graph analysis
* Dependency analysis
* Test discovery

## Philosophy

GoSherpa focuses on information already present in the codebase.

No annotations.
No code generation.
No magic.

Just better visibility into how a Go project is put together.
