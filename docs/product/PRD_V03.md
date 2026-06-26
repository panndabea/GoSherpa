# PRD — GoSherpa Intelligence Platform (v1.0)

## Vision

GoSherpa ist keine Sammlung einzelner CLI-Kommandos mehr.

GoSherpa wird die Intelligence-Schicht einer Go-Codebasis.

Menschen

↓

CLI

↓

AI Agents

↓

CI/CD

↓

GitHub Actions

↓

IDEs

↓

alle greifen auf dieselbe Repository-Intelligenz zu.

GoSherpa beantwortet nicht mehr nur:

> "Wo ist diese Funktion?"

sondern

> "Was bedeutet diese Änderung für das gesamte System?"

---

# Ziele

Version 1.0 führt eine einheitliche Intelligence API ein.

Alle Features greifen auf dieselbe Analyse-Engine zurück.

Neue Features:

* AI Context Export
* Pull Request Intelligence
* GitHub Action
* MCP Server
* Repository Snapshots
* Risk Analysis
* Architecture Reports

---

# Neues Konzept

GoSherpa besitzt einen vollständigen Repository Graph.

Dieser besteht aus

```text
Packages

↓

Symbols

↓

References

↓

Calls

↓

Interfaces

↓

Implementations

↓

Tests

↓

Git History
```

Darauf bauen alle Features auf.

---

# Neue CLI

```bash
gosherpa analyze

gosherpa snapshot

gosherpa pr

gosherpa prompt

gosherpa risk

gosherpa architecture

gosherpa serve

gosherpa graph
```

---

# Repository Snapshot

```bash
gosherpa snapshot
```

erzeugt

```text
.gosherpa/

index.db

graph.json

metadata.json
```

Damit muss nicht jedes Tool das Repository erneut analysieren.

---

# Prompt Export

```bash
gosherpa prompt auth.ValidateToken
```

liefert

eine kompakte Markdown-Datei

```text
# Symbol

ValidateToken()

Package

internal/auth

Purpose

...

Called By

...

Calls

...

Interfaces

...

Tests

...

Related Types

...
```

Gedacht für

Cursor

Claude Code

Codex

Aider

OpenAI Agents

Gemini CLI

usw.

GoSherpa erzeugt den Kontext.

Das LLM schreibt den Code.

---

# Pull Request Intelligence

```bash
gosherpa pr
```

Ausgabe

```text
Summary

4 Packages changed

12 Symbols changed

Risk

Medium

Affected APIs

...

Affected Interfaces

...

Recommended Reviewers

...

Recommended Tests

...

Breaking Change Risk

Low
```

Optional

```bash
gosherpa pr --markdown
```

für GitHub.

---

# Risk Analysis

```bash
gosherpa risk
```

Beispiel

```text
High Risk

authentication/

public interfaces

database adapters

Low Risk

internal helpers

private utilities
```

Berechnung

* öffentliche Interfaces
* viele Callers
* viele Packages
* zentrale Dependencies

---

# Architecture Report

```bash
gosherpa architecture
```

liefert

```text
Largest Packages

Most Coupled Packages

Dependency Cycles

Unused Interfaces

Large Structs

God Packages

Highly Connected Services

Package Heatmap
```

Keine Lint-Regeln.

Reine Strukturinformationen.

---

# GitHub Action

Neue Action

```yaml
- uses: gosherpa/action
```

Kommentiert automatisch

PR Summary

Affected Tests

Risk Score

Architecture Impact

---

# MCP Server

```bash
gosherpa serve
```

startet einen MCP Server.

Tools

```text
find_symbol

explain_symbol

impact_analysis

package_graph

callers

callees

tests

risk
```

Dadurch können AI-Agents gezielt Informationen abrufen, statt das gesamte Repository zu durchsuchen.

---

# Explain Engine v2

```bash
gosherpa explain auth.ValidateToken
```

liefert

Purpose

Dependencies

Call Graph

Interface Graph

Tests

Risk

Architecture Role

Related Files

Related Types

Suggested Reading Order

---

# Graph Export

```bash
gosherpa graph --format json
```

oder

```bash
gosherpa graph --format graphviz
```

oder

```bash
gosherpa graph --format mermaid
```

Dadurch können externe Tools Visualisierungen erzeugen.

---

# JSON API

Alle Commands unterstützen

```bash
--json
```

GoSherpa wird dadurch problemlos in

CI

Editoren

Dashboards

AI

integrierbar.

---

# Performance

Snapshot

<10 Sekunden

Explain

<50 ms

Impact

<200 ms

Prompt

<100 ms

PR

<500 ms

---

# Nicht Teil von v1.0

Kein Chatbot

Keine automatische Codegenerierung

Keine Refactorings

Keine IDE

Keine Cloud

Keine SaaS

Keine Telemetrie

GoSherpa bleibt ein lokales Open-Source-CLI.

---

# README Positionierung

## GoSherpa

Code Intelligence for Go.

GoSherpa understands your Go codebase and answers questions like

* What changed?
* What is affected?
* Which tests should I run?
* Who calls this function?
* Which interfaces are involved?
* What is the architectural impact?
* What context should I give my AI coding assistant?

All from one fast CLI.

---

# Erfolgsdefinition

Ein Entwickler kann innerhalb weniger Sekunden verstehen

* eine Funktion
* ein Package
* eine Änderung
* eine Pull Request
* eine Architektur

ohne IDE-Suche, manuelle Call-Graph-Analyse oder vollständiges Einlesen der Codebasis.

GoSherpa wird damit zur Intelligence-Schicht für Go-Repositories.

# Notiz
Ich würde eine Sache gegenüber diesem PRD noch ändern
Nachdem ich deine Vision und GoSherpa gesehen habe, glaube ich, dass der eigentliche North Star nicht "Change Intelligence" ist.
Er lautet:
"Make Go repositories understandable for humans, CI and AI."
Das ist deutlich breiter und zukunftssicherer.
Dann ordnen sich alle Features logisch darunter ein:
🔎 Explore – find, refs, callers, callees, explain
🧠 Understand – impact, risk, architecture
🤖 AI – prompt, serve (MCP), strukturierte JSON-Ausgaben
🚀 Automate – pr, snapshot, GitHub Action
Das macht GoSherpa nicht zu einem "Tool mit vielen Befehlen", sondern zu einer Plattform für Repository Intelligence. Gerade im Zeitalter von AI-Coding-Assistenten ist das eine deutlich stärkere Positionierung, weil du dich auf den Teil konzentrierst, den LLMs allein nicht zuverlässig leisten: eine schnelle, präzise und strukturierte Analyse einer Go-Codebasis.