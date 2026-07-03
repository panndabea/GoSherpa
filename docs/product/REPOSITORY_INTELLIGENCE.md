# Repository Intelligence

## GoSherpa should not become just another Agent Framework

The AI ecosystem is rapidly filling with agent frameworks.

There are orchestration frameworks.
There are workflow frameworks.
There are multi-agent runtimes.
There are tool execution frameworks.

But almost all of them share the same weakness:

> **They do not truly understand Go repositories.**

Large Language Models can read code.

They cannot efficiently understand an entire Go codebase.

They struggle with:

* package relationships
* symbol graphs
* call graphs
* architectural boundaries
* ownership
* dependency impact
* deterministic repository navigation

As repositories grow, this problem becomes even worse.

---

# The Opportunity

Instead of becoming another generic agent framework, GoSherpa should create a completely new category:

> **Repository Intelligence**

GoSherpa should become the intelligence layer that sits between a Go repository and every AI agent.

```
Go Repository
        │
        ▼
   GoSherpa Engine
        │
        ▼
 Repository Intelligence
        │
        ▼
 Claude Code
 Cursor
 OpenAI Agents
 Aider
 Continue
 Goose
 MCP
 GitHub Actions
 Custom Agents
```

Every AI system needs repository understanding.

GoSherpa should provide it.

---

# Repository Intelligence

Repository Intelligence means transforming a Go repository into structured, deterministic knowledge.

Instead of asking an LLM to read thousands of files, GoSherpa provides a compact understanding of the repository.

Examples include:

* package graph
* symbol graph
* call graph
* interface relationships
* architecture layers
* dependency impact
* public API surface
* repository entry points
* test coverage map
* change impact
* semantic navigation
* agent context

This information is deterministic.

No hallucinations.

No guessing.

---

# What GoSherpa Should Do

A single command should provide an overview of any Go repository.

```
gosherpa analyze .
```

First slice implemented:

* repository summary
* package overview
* symbol counts
* important public symbols
* entry point candidates
* simple package hotspots
* testing overview
* readiness, limitations, and suggested next commands
* stable JSON output

Output could include:

* repository summary
* architecture overview
* package dependencies
* important symbols
* entry points
* hotspots
* dead code
* circular dependencies
* testing overview
* architectural smells
* suggested reading order

Understanding a repository should take seconds instead of hours.

---

# AI-First Context

The same intelligence should also be available as structured output.

```
gosherpa context .
```

Result:

```
context.json
```

Designed specifically for AI agents.

Instead of sending hundreds of files into an LLM context window, GoSherpa should provide:

* architecture
* symbols
* dependencies
* APIs
* packages
* risks
* relevant files
* semantic summaries

The LLM receives knowledge instead of raw source code.

---

# Repository Snapshots

GoSherpa should eventually support repository snapshots.

```
gosherpa snapshot
```

Produces:

```
.sherpa/

snapshot.bin
graph.bin
symbols.bin
types.bin
context.json
```

Future analyses become nearly instantaneous.

Snapshots also enable:

* incremental analysis
* CI acceleration
* PR intelligence
* agent caching
* historical comparisons

---

# Why This Matters

Frameworks evolve.

Editors change.

LLMs change.

Agent runtimes come and go.

But every AI system that works with Go code has the same fundamental requirement:

> It must understand the repository.

That requirement does not disappear.

It becomes more important every year.

---

# The North Star

GoSherpa should become:

> **The Repository Intelligence Layer for Go.**

Not just another framework.

Not just another CLI.

Not just another MCP server.

The foundation that every Go AI tool builds upon.

---

# Long-Term Vision

One day, developers should naturally assume that every serious AI tool working with Go internally uses GoSherpa.

Just as LLVM became the foundation for many compilers, GoSherpa can become the foundation for repository understanding in the Go ecosystem.

If GoSherpa owns Repository Intelligence, it won't simply compete inside the AI ecosystem.

It becomes infrastructure for the ecosystem itself.
