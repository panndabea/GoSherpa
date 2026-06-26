# PRD — GoSherpa Symbol Intelligence (v0.5)

## Implementation Status

Status: started.

Implemented first slice:

* `gosherpa explain <symbol>`
* human and JSON output
* purpose from Go doc comments, risk summary, architecture role, definition,
  reading order, references, callers, callees, affected packages, interface and
  implementation impact, related tests, and suggested test commands
* package-qualified `explain` caller/callee signals that avoid mixing same-name
  functions across local packages
* package-qualified standalone call graph commands for `callers`, `callees`,
  `path`, and `paths`
* receiver-variable method calls in standalone call graph commands, resolved
  with package-level type information
* standalone `implementers <interface>` and `interfaces <type>` commands
  backed by the local interface graph
* opt-in test callers for `gosherpa callers --tests` and
  `gosherpa explain --tests`
* ambiguity handling for duplicate unqualified symbol/function targets, with
  candidate package/file/line details, package-qualified examples, and
  structured JSON diagnostics when `--json` is used

---

## Hintergrund

Version 0.1 liefert Package-basierte Impact Analysis.

Das ist bewusst konservativ.

Version 0.5 erweitert GoSherpa auf **Symbol-Level Intelligence**.

GoSherpa soll nicht nur wissen, welche Packages betroffen sind, sondern welche konkreten:

* Funktionen
* Methoden
* Interfaces
* Structs
* Call Chains

durch eine Änderung beeinflusst werden.

Dadurch entsteht eine deutlich präzisere Change Analysis.

---

# Vision

Ein Entwickler möchte wissen:

> "Wenn ich diese Methode ändere – welche anderen Methoden, Interfaces, Services und Tests betrifft das?"

Nicht nur:

> "Package X ist betroffen."

Sondern:

> "Diese fünf Methoden hängen davon ab."

---

# Ziele

Neue Fähigkeiten:

* Caller Graph
* Callee Graph
* Interface Propagation
* Symbol Impact
* bessere Testauswahl
* Explain Mode

---

# Neue CLI

```bash
gosherpa callers auth.ValidateToken

gosherpa callees auth.ValidateToken

gosherpa impact symbol auth.ValidateToken

gosherpa interfaces auth.JWTAuthenticator

gosherpa implementers auth.Authenticator

gosherpa explain auth.ValidateToken

gosherpa tests affected --symbol auth.ValidateToken
```

---

# Beispiel

```bash
gosherpa impact symbol auth.ValidateToken
```

liefert

```text
Changed Symbol

auth.ValidateToken()

Direct callers

middleware.AuthMiddleware()

session.Verify()

api.Login()

Indirect callers

UserService.Login()

AdminService.Login()

Interfaces

Authenticator

Implementations

JWTAuthenticator

MockAuthenticator

Affected packages

internal/auth

internal/api

internal/session

Recommended tests

auth_test.go

middleware_test.go

integration/login_test.go
```

---

# Neue Engine

Neue Package

```text
internal/symbolgraph
```

Diese enthält

Definition Graph

Reference Graph

Call Graph

Interface Graph

Method Receiver Graph

---

# Symbol Index

Der bestehende Index wird erweitert.

Jedes Symbol erhält

```text
ID

Package

Receiver

Name

Kind

Location

Documentation
```

Kinds

```text
Function

Method

Struct

Interface

Const

Var

Type
```

---

# Call Graph

Für jede Funktion speichern

```text
calls →

called by ←
```

MVP

Direkte Aufrufe

Keine vollständige SSA nötig.

Normale AST-Analyse reicht zunächst.

---

# Interface Graph

Neue Beziehungen

```text
Interface

↓

Implementer

↓

Methods
```

Beispiel

```text
Authenticator

↓

JWTAuthenticator

↓

Validate()

Refresh()

Revoke()
```

Änderung an

```text
Validate()
```

kann damit propagiert werden.

---

# Explain Command

Neuer Command

```bash
gosherpa explain auth.ValidateToken
```

Ausgabe

```text
ValidateToken

Package

internal/auth

Calls

parseJWT()

verifySignature()

loadKeys()

Called by

Login()

Refresh()

Middleware()

Implements

Authenticator.Validate()

Referenced in

18 files

Test coverage

auth_test.go

integration/login_test.go
```

Dies wird das wichtigste Diagnose-Tool.

---

# Symbol Impact

Neue Analyse

```text
AnalyzeSymbol()
```

liefert

```text
Changed Symbol

↓

Caller

↓

Caller

↓

Package

↓

Tests
```

statt

```text
Package

↓

Package

↓

Tests
```

Dadurch deutlich weniger False Positives.

---

# Test Selection v2

Nicht mehr

Package

↓

Tests

sondern

Symbol

↓

Caller

↓

Package

↓

Tests

Beispiel

```text
Eine Utility-Funktion wird geändert.

Nur drei Packages verwenden sie.

Nicht mehr zwanzig.
```

---

# JSON API

```bash
gosherpa impact symbol auth.ValidateToken --json
```

liefert

```json
{
  "symbol": "...",
  "callers": [],
  "callees": [],
  "interfaces": [],
  "implementations": [],
  "affected_packages": [],
  "affected_tests": []
}
```

---

# Architektur

Neue Module

```text
internal/symbolgraph

internal/callgraph

internal/interfacegraph
```

Impact Engine verwendet jetzt

Package Graph

*

Symbol Graph

*

Call Graph

*

Interface Graph

---

# Explain Engine

Neue Package

```text
internal/explain
```

liefert strukturierte Reports.

Alle Informationen kommen aus bestehenden Graphen.

Keine Sonderlogik.

---

# Performance

Indexing

<10 Sekunden

Caller Query

<50 ms

Explain

<100 ms

Impact Symbol

<200 ms

---

# Nicht Teil von v0.5

Keine SSA

Keine Reflection Analysis

Keine Dynamic Dispatch

Keine Runtime Analysis

Keine Build Tag Matrix

Keine CGO Analyse

---

# Tests

Golden Tests

für

Caller Graph

Callee Graph

Explain

Impact Symbol

Interface Analysis

---

# README

Neue Sektion

## Symbol Intelligence

mit Beispielen

```bash
gosherpa callers auth.ValidateToken

gosherpa explain auth.ValidateToken

gosherpa impact symbol auth.ValidateToken
```

---

# Erfolgsdefinition

Ein Entwickler kann innerhalb weniger Sekunden beantworten:

* Wer ruft diese Methode auf?
* Was ruft diese Methode auf?
* Welche Interfaces hängen daran?
* Welche Implementierungen sind betroffen?
* Welche Tests sollte ich ausführen?
* Welche Services könnten brechen?

GoSherpa entwickelt sich damit von einer Repository-Navigation zu einer echten strukturellen Analyseplattform für Go-Codebasen.

# Notiz
Eine Ergänzung, die ich noch wichtiger finde
Wenn ich GoSherpa heute neu bauen würde, wäre gosherpa explain das eigentliche Killer-Feature – sogar wichtiger als impact.
Beispielsweise:
gosherpa explain internal/auth.ValidateToken
würde nicht nur Listen ausgeben, sondern eine komplette Zusammenfassung:
Zweck der Funktion
Wer sie verwendet
Welche Interfaces betroffen sind
Welche öffentlichen APIs daran hängen
Welche Tests relevant sind
Welches Risiko eine Änderung hat (Low/Medium/High)
Das wird nicht nur für Menschen extrem nützlich sein, sondern auch für AI-Agents wie Codex, Claude Code oder Cursor. Diese könnten gosherpa explain als zuverlässige Kontextquelle nutzen, statt das gesamte Repository selbst analysieren zu müssen. Ich würde diesen Explain-Workflow deshalb als zentrales Leitprinzip für die weitere Entwicklung etablieren.
