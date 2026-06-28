# PRD — GoSherpa Impact Engine (v0.1)

## Implementation Status

Status: MVP completed on 2026-06-26.

Implemented:

* `gosherpa impact file <file>`
* `gosherpa impact diff --base <ref>`
* `gosherpa tests affected --base <ref>`
* `gosherpa impact symbol <package.Symbol>`
* `gosherpa impact package <package>`
* shared `ImpactReport`
* git changed-file and changed-symbol extraction
* deletion-aware diff symbol extraction from base refs
* changed package and affected package reporting
* affected interface and implementation reporting
* affected test and `go test` command planning
* human and JSON output with golden coverage

Conservative MVP boundaries:

* analysis remains local and AST-based
* interface impact uses import-aware signature matching, not the full Go type checker
* test selection is package-oriented plus existing direct-reference signals
* no SSA, pointer analysis, GitHub Action, MCP, IDE, or AI integration

Release notes: [RELEASE_NOTES_V01.md](../releases/RELEASE_NOTES_V01.md)

---

## Hintergrund

GoSherpa ist ein CLI-Tool zur strukturellen Analyse von Go-Codebasen.

Aktuell unterstützt GoSherpa Navigation durch:

* Packages
* Dateien
* Symbole
* Referenzen
* Interfaces
* Abhängigkeiten

Das nächste große Feature ist eine **Impact Analysis Engine**, die Entwicklern beantwortet:

> "Wenn ich diese Änderung mache – was ist davon betroffen?"

Das Ziel ist NICHT ein weiterer Linter oder ein weiterer Code Explorer.

GoSherpa soll zur strukturellen Intelligenzschicht einer Go-Codebasis werden.

---

# Vision

GoSherpa soll die folgenden Fragen beantworten können:

* Welche Packages sind betroffen?
* Welche Interfaces sind betroffen?
* Welche Implementierungen könnten brechen?
* Welche Funktionen rufen den geänderten Code auf?
* Welche Tests sollte ich ausführen?
* Welche Services könnten betroffen sein?

Langfristig soll GoSherpa für Go werden, was Nx "affected" für JavaScript ist.

---

# Ziele v0.1

Implementiere einen ersten konservativen Impact Analyzer.

Der Fokus liegt auf:

* hohe Genauigkeit
* einfache Architektur
* schnelle Ausführung
* keine False Negatives wenn möglich

Lieber zu viele Treffer als zu wenige.

---

# CLI

Neue Commands:

```bash
gosherpa impact file <file>

gosherpa impact diff --base origin/main

gosherpa tests affected --base origin/main

gosherpa impact symbol <package.Symbol>

gosherpa impact package <package>
```

---

# Beispiel

Änderung:

```
internal/auth/session.go
```

Ausgabe:

```
Changed package

internal/auth

Affected packages

internal/api
internal/user
internal/session

Affected interfaces

Authenticator
SessionProvider

Affected implementations

JWTAuthenticator
RedisSessionStore

Affected tests

internal/auth/auth_test.go

internal/user/user_test.go

integration/auth_test.go
```

---

# MVP Scope

## 1. Repository Index

GoSherpa besitzt bereits einen Index.

Erweitere diesen um:

Package Graph

```
Package

Imports

Imported by
```

Symbol Graph

```
Function

Method

Struct

Interface
```

Reference Graph

```
Definition

References
```

Interface Graph

```
Interface

Implementers
```

Test Graph

```
Package

Test files

Test functions
```

---

## 2. Git Diff Reader

Neue Komponente:

```
internal/git
```

Funktionen:

```
ChangedFiles(base, head)

ChangedPackages()

ChangedSymbols()
```

Verwendet:

```
git diff --name-only

git diff
```

---

## 3. Impact Engine

Neue Package:

```
internal/impact
```

API

```
Analyzer

AnalyzeFile()

AnalyzePackage()

AnalyzeDiff()
```

Output

```
ImpactReport
```

mit

```
ChangedFiles

ChangedPackages

AffectedPackages

AffectedSymbols

AffectedInterfaces

AffectedImplementations

AffectedTests
```

---

## 4. Test Discovery

Für jedes Package:

Finde

```
*_test.go
```

und ordne Tests ihrem Package zu.

Regeln MVP:

Wenn Package betroffen

→ alle Tests dieses Packages

Wenn abhängiges Package betroffen

→ ebenfalls dessen Tests

Keine Function-Level Analyse in v0.1.

---

## 5. Output

Standard:

Pretty CLI

Optional:

```
--json
```

liefert

```
{
  "schemaVersion": 1,
  "command": "impact diff",
  "target": "origin/main",
  "root": "/path/to/repo",
  "modulePath": "example.com/project",
  "warnings": [],
  "data": {
    "changedFiles": [],
    "changedPackages": [],
    "affectedPackages": [],
    "affectedSymbols": [],
    "affectedInterfaces": [],
    "affectedImplementations": [],
    "affectedTests": [],
    "testCommands": [],
    "testPlan": {}
  }
}
```

---

# Architektur

Neue Packages

```
internal/git

internal/impact

internal/sherpa
```

Keine zyklischen Abhängigkeiten.

Impact Engine komponiert Git-Diff-Daten mit den bestehenden
Repository-, Reference-, Test- und Package-Helfern aus `internal/sherpa`.

Git kennt keine Go-Logik.

Die Repository-Analyse kennt kein Git.

---

# Nicht Teil von v0.1

Keine SSA

Keine Pointer Analysis

Keine vollständige Call Graph Analyse

Keine IDE Integration

Keine GitHub Action

Keine AI Integration

Keine MCP

Keine SQLite

Keine Watcher

---

# Performance

100k LOC

Index <5 Sekunden

Impact <300ms

Diff Analyse <500ms

---

# Tests

Unit Tests

für

Package Graph

Git Diff

Impact Analyzer

JSON Output

Golden Tests für CLI Output.

---

# Dokumentation

README erweitern um

## Impact Analysis

Beispiele

```bash
gosherpa impact diff --base main

gosherpa tests affected --base main

gosherpa impact file internal/auth/session.go
```

inklusive Screenshots.

---

# Erfolgsdefinition

Ein Go-Entwickler kann nach einer Änderung sofort beantworten:

* Welche Packages sind betroffen?
* Welche Tests sollte ich laufen lassen?
* Welche Interfaces könnten brechen?

ohne IDE und ohne manuelle Analyse.

GoSherpa entwickelt sich damit von einem Code Explorer zu einer Change Intelligence CLI für Go.
