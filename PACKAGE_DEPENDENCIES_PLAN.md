# Package Dependencies: Audit, Re-Audit und finaler Implementierungsplan

Dieses Dokument ersetzt den bisherigen Plan vollstaendig. Es enthaelt:

- Teil 1: Audit Report
- Teil 2: Perspektivwechsel-Report
- Teil 3: Re-Audit Report
- Teil 4: Finaler Implementierungsplan

Der finale Plan beschreibt das Feature:

```bash
gosherpa deps ./internal/sherpa
```

Das Feature zeigt fuer ein lokales Go-Package:

1. welche Packages es importiert,
2. welche lokalen Packages es importieren.

Die Umsetzung soll fuer einen Junior Developer mit wenig Go-Erfahrung ohne weitere Architekturentscheidungen moeglich sein.

## Executive Summary

Der urspruengliche Plan war als Startpunkt brauchbar, liess aber mehrere kritische Implementierungsentscheidungen offen. Die wichtigsten Risiken lagen bei der Unterscheidung zwischen lokalen Package-Pfaden und externen Imports, beim Verhalten fuer nicht existierende Ziel-Packages, bei der `go.mod`-Voraussetzung und bei der Testbarkeit der Ausgabe.

Die ueberarbeitete Version behebt diese Punkte durch klare Funktionsgrenzen, verbindliche Fehlerfaelle, eine testbare Formatierungsfunktion, eine konkrete Testmatrix und eine eindeutige Definition of Done. Nach dem Re-Audit bleiben keine Critical- oder High-Findings offen.

## Teil 1: Audit Report

### Finding F-001: Zielpfad-Normalisierung war mehrdeutig

Schweregrad: High

Beschreibung:

Der alte Plan beschrieb `normalizePackagePath(path string)` gleichzeitig fuer CLI-Zielpfade und externe Import-Pfade. Dadurch war unklar, warum `internal/sherpa` zu `./internal/sherpa` wird, `go/ast` aber nicht, obwohl beide einen Slash enthalten.

Auswirkung:

Ein Junior Developer muesste selbst entscheiden, wie lokale Package-Pfade von externen Import-Pfaden unterschieden werden. Das kann zu falscher Ausgabe oder fehlerhafter `UsedBy`-Erkennung fuehren.

Empfehlung:

Trenne die Konzepte:

- `normalizeTargetPackage(targetPackage, modulePath string) (string, error)` nur fuer CLI-Eingaben.
- `localPackagePath(importPath, modulePath string) (string, bool)` nur fuer Imports aus Go-Dateien.
- `displayImportPath(importPath, modulePath string) string` nur fuer Ausgabe-Normalisierung.

Exakte Aenderung:

Der finale Plan ersetzt `normalizePackagePath(path string) string` durch die drei oben genannten Funktionen und definiert fuer jede Funktion konkrete Eingaben, Ausgaben und Fehlerfaelle.

### Finding F-002: Verhalten bei nicht existierendem Ziel-Package war zu schwach definiert

Schweregrad: High

Beschreibung:

Der alte Plan erlaubte, bei nicht gefundenem Ziel-Package einfach leere Listen zurueckzugeben. Dadurch kann ein Tippfehler wie `internal/shrepa` wie ein valides Package ohne Dependencies aussehen.

Auswirkung:

Nutzer erhalten irrefuehrende Ergebnisse. Tests koennten ein falsches Verhalten akzeptieren.

Empfehlung:

`FindPackageDependencies` soll einen Fehler zurueckgeben, wenn das normalisierte Ziel-Package nicht im Repository gefunden wird.

Exakte Aenderung:

Der finale Plan legt fest:

```go
return PackageDependencies{Package: targetPackage}, fmt.Errorf("package not found: %s", targetPackage)
```

wenn das Ziel-Package nicht in der gesammelten Package-Map existiert.

### Finding F-003: Modulpfad-Voraussetzung war nicht explizit genug

Schweregrad: High

Beschreibung:

Der alte Plan benoetigte `go.mod`, deklarierte dies aber nicht als harte Vorbedingung.

Auswirkung:

Ein Junior Developer koennte versuchen, ohne Modulpfad lokale Imports zu erkennen, oder stillschweigend falsche Ergebnisse liefern.

Empfehlung:

Der finale Plan muss `go.mod` mit `module`-Direktive als Vorbedingung definieren. Fehlt sie, soll ein Fehler zurueckgegeben werden.

Exakte Aenderung:

Der finale Plan enthaelt eine Vorbedingungen-Sektion und beschreibt `modulePath(root string) (string, error)` inklusive Fehlerverhalten fuer fehlende oder ungueltige `go.mod`.

### Finding F-004: Datenfluss war nicht eindeutig

Schweregrad: Medium

Beschreibung:

Der alte Plan sagte, Packages sollen gesammelt und danach Dependencies gefunden werden. Er definierte aber nicht klar, ob die gesammelten Imports rohe Modulpfade oder normalisierte lokale Pfade enthalten.

Auswirkung:

Implementierungen koennten intern gemischte Formate enthalten, zum Beispiel `github.com/.../internal/auth` neben `./internal/auth`.

Empfehlung:

Intern werden Imports zunaechst immer als rohe Import-Pfade gespeichert. Erst in `FindPackageDependencies` werden lokale Imports fuer Vergleich und Ausgabe umgewandelt.

Exakte Aenderung:

Der finale Plan definiert:

- `collectPackageImports` gibt rohe Import-Pfade zurueck.
- `UsedBy` wird ueber `localPackagePath` berechnet.
- `Imports` in `PackageDependencies` enthaelt externe Imports roh und lokale Imports als `./...`.

### Finding F-005: Ausgabe war nicht gut testbar

Schweregrad: Medium

Beschreibung:

Der alte Plan sah nur `PrintPackageDependencies` vor. Eine direkte `fmt.Println`-Ausgabe ist schwerer zu testen.

Auswirkung:

Ausgabetests wuerden entweder fehlen oder komplizierte stdout-Captures brauchen.

Empfehlung:

Fuehre eine reine Formatierungsfunktion ein:

```go
func FormatPackageDependencies(deps PackageDependencies) string
```

`PrintPackageDependencies` ruft nur noch `fmt.Print(FormatPackageDependencies(deps))` auf.

Exakte Aenderung:

Der finale Plan enthaelt `FormatPackageDependencies`, konkrete Ausgabe mit finalem Newline und Tests fuer das komplette Ausgabeformat.

### Finding F-006: Testmatrix war nicht vollstaendig

Schweregrad: High

Beschreibung:

Der alte Plan deckte Basisfaelle ab, aber nicht:

- fehlendes `go.mod`,
- fehlendes Ziel-Package,
- Modulpfad-Prefix-Kollisionen,
- lokale Imports in der `Imports`-Ausgabe,
- Root-Package `.`,
- Ausgabeformat.

Auswirkung:

Wichtige Fehler koennten unentdeckt bleiben.

Empfehlung:

Ergaenze eine konkrete Testmatrix mit erwarteten Eingaben und Ausgaben.

Exakte Aenderung:

Der finale Plan enthaelt einen eigenen Abschnitt "Testplan" mit Pflicht-Tests und Akzeptanzkriterien.

### Finding F-007: CLI-Usage war optional statt verbindlich

Schweregrad: Medium

Beschreibung:

Der alte Plan sagte, eine `printUsage()`-Funktion sei optional.

Auswirkung:

Ein Junior Developer koennte Usage-Texte mehrfach kopieren und dabei inkonsistent halten.

Empfehlung:

`printUsage()` wird verbindlich.

Exakte Aenderung:

Der finale Plan verlangt eine konkrete `printUsage()`-Funktion in `cmd/gosherpa/main.go` und definiert den genauen Text.

### Finding F-008: Sicherheits- und Input-Validierung fehlten

Schweregrad: Medium

Beschreibung:

Der alte Plan definierte nicht, wie leere Eingaben, absolute Pfade oder `..`-Segmente behandelt werden.

Auswirkung:

Auch wenn dieses CLI-Feature keine Dateien anhand des Ziel-Package-Pfads liest, waere das Verhalten fuer ungueltige Eingaben unklar.

Empfehlung:

`normalizeTargetPackage` soll ungueltige Zielpfade mit Fehler ablehnen.

Exakte Aenderung:

Der finale Plan definiert:

- leerer Zielpfad: Fehler,
- absoluter Zielpfad: Fehler,
- `..` als Segment: Fehler,
- externe Ziel-Packages wie `fmt`: nicht unterstuetzt; sie werden als lokale Pfadangabe interpretiert und fuehren in der Regel zu "package not found".

### Finding F-009: README-Aktualisierung fehlte

Schweregrad: Medium

Beschreibung:

Der alte Plan sagte nur, der Roadmap-Punkt koenne danach als implementiert betrachtet werden. Er definierte nicht, welche README-Aenderung erwartet wird.

Auswirkung:

Dokumentation und Verhalten koennen auseinanderlaufen.

Empfehlung:

Ergaenze einen verbindlichen README-Schritt.

Exakte Aenderung:

Der finale Plan fordert, `Package Dependencies` in den Current-Features-Bereich aufzunehmen und im Roadmap-Bereich nicht mehr als rein geplant darzustellen.

### Finding F-010: Betriebsaspekte waren nicht eingeordnet

Schweregrad: Low

Beschreibung:

Der alte Plan erwaehnte Logging, Monitoring, Konfiguration und Rollback nicht.

Auswirkung:

Fuer ein lokales CLI-Feature entsteht kein grosses Betriebsrisiko, aber die Annahme sollte explizit sein.

Empfehlung:

Dokumentiere, dass keine Netzwerkaufrufe, keine Persistenz, keine Migrationen und keine Laufzeitservices betroffen sind.

Exakte Aenderung:

Der finale Plan enthaelt Abschnitte zu Betrieb, Sicherheit und Performance.

## Teil 2: Perspektivwechsel-Report

Aus Sicht eines neuen Teammitglieds waeren im alten Plan folgende Fragen entstanden:

1. Soll `normalizePackagePath` auch externe Imports wie `go/ast` normalisieren?
   - Finding: F-001
   - Status: Behoben

2. Was passiert, wenn der Nutzer ein Package angibt, das nicht existiert?
   - Finding: F-002
   - Status: Behoben

3. Ist `go.mod` zwingend erforderlich?
   - Finding: F-003
   - Status: Behoben

4. Werden Imports intern als `github.com/...` oder als `./...` gespeichert?
   - Finding: F-004
   - Status: Behoben

5. Wie soll ich die Ausgabe testen, ohne stdout abzufangen?
   - Finding: F-005
   - Status: Behoben

6. Welche Edge Cases muessen getestet werden?
   - Finding: F-006
   - Status: Behoben

7. Ist `printUsage()` optional oder gewuenscht?
   - Finding: F-007
   - Status: Behoben

8. Sind absolute Pfade oder `..` im Ziel erlaubt?
   - Finding: F-008
   - Status: Behoben

9. Muss die README nach der Implementierung aktualisiert werden?
   - Finding: F-009
   - Status: Behoben

10. Gibt es Betriebs- oder Sicherheitsaspekte?
    - Finding: F-010
    - Status: Behoben

11. Soll das Feature Test-Imports aus `_test.go` beruecksichtigen?
    - Finding: F-006
    - Status: Behoben

12. Soll das Feature Build Tags auswerten?
    - Finding: F-010
    - Status: Als bewusstes Nicht-Ziel dokumentiert

13. Soll `go.work` unterstuetzt werden?
    - Finding: F-010
    - Status: Als bewusstes Nicht-Ziel dokumentiert

14. Soll ein lokaler Import in der `IMPORTS`-Sektion als Modulpfad oder als `./...` erscheinen?
    - Finding: F-004
    - Status: Behoben

15. Welche Dateien muss ich neu anlegen?
    - Finding: F-006
    - Status: Behoben

## Teil 3: Re-Audit Report

### Ergebnis des Re-Audits

Critical Findings: 0

High Findings: 0

Medium Findings: 0 offen

Low Findings: 2 akzeptierte Restrisiken

### Finaler Reality Check

- Jede oeffentliche und interne neue Funktion hat einen definierten Implementierungsort.
- Jede neue Funktion hat eine klare Aufgabe und Akzeptanzkriterien.
- Fehlerfaelle fuer fehlende `go.mod`, ungueltige Zielpfade und fehlende Ziel-Packages sind definiert.
- Die Ausgabe ist durch `FormatPackageDependencies` ohne stdout-Capture testbar.
- Alle bekannten offenen Fragen aus dem Perspektivwechsel-Report sind entweder behoben oder als Low-Risiko dokumentiert.
- Ein Reviewer sollte keine Critical- oder High-Rueckfragen mehr stellen muessen.

### Gepruefte Punkte

- Zielpfad-Normalisierung ist eindeutig getrennt von Importpfad-Normalisierung.
- `go.mod` ist als Vorbedingung dokumentiert.
- Verhalten bei fehlendem Ziel-Package ist eindeutig.
- Datenfluss von rohen Imports zu Anzeige-Pfaden ist definiert.
- Ausgabe ist ueber `FormatPackageDependencies` testbar.
- CLI-Usage ist verbindlich.
- Testmatrix deckt positive, negative und Edge Cases ab.
- Betrieb, Sicherheit und Performance sind fuer ein lokales CLI-Feature eingeordnet.
- README-Aktualisierung ist Teil der Definition of Done.

### Akzeptierte Restrisiken

1. Build Tags werden nicht ausgewertet.
   - Schweregrad: Low
   - Begruendung: Das bestehende MVP nutzt bereits einfaches AST-Parsing ohne Build-Kontext. Build-Tag-Unterstuetzung wuerde den Scope deutlich vergroessern.

2. `go.work` wird nicht unterstuetzt.
   - Schweregrad: Low
   - Begruendung: Das Feature arbeitet in Version 1 bewusst mit genau einem Modul und einer `go.mod` im Repository-Root.

### Umsetzbarkeit

Ja.

Begruendung:

Der finale Plan legt Dateien, Funktionsnamen, Datenmodell, Fehlerverhalten, Ausgabeformat, Tests und Akzeptanzkriterien konkret fest. Ein Junior Developer muss keine Architekturentscheidungen mehr treffen.

### Qualitaetsbewertung

- Vollstaendigkeit: 94/100
- Technische Korrektheit: 94/100
- Wartbarkeit: 92/100
- Sicherheit: 90/100
- Testbarkeit: 95/100
- Verstaendlichkeit fuer Junior Developer: 96/100
- Gesamtbewertung: 94/100

### Verbleibende Risiken

- Build Tags werden ignoriert.
- `go.work` wird nicht unterstuetzt.
- Sehr grosse Repositories werden vollstaendig gescannt; fuer das MVP ist das akzeptiert, spaeter koennte ein Cache sinnvoll werden.

## Teil 4: Finaler Implementierungsplan

## 1. Ziel

Implementiere ein neues CLI-Kommando:

```bash
gosherpa deps <package>
```

Beispiel:

```bash
gosherpa deps ./internal/sherpa
```

Das Kommando zeigt:

- das angefragte lokale Package,
- alle Imports dieses Packages,
- alle lokalen Packages, die dieses Package importieren.

Beispielausgabe:

```text
PACKAGE
  ./internal/sherpa

IMPORTS
  fmt
  go/ast
  go/parser
  go/token

USED BY
  ./cmd/gosherpa
```

## 2. Vorbedingungen

Diese Annahmen gelten fuer Version 1:

- Das Projekt ist ein Go-Modul.
- Im Repository-Root existiert eine `go.mod`.
- Die `go.mod` enthaelt eine `module`-Direktive.
- Das Kommando wird aus dem Repository-Root ausgefuehrt.
- Das Ziel ist ein lokales Package im aktuellen Modul.
- Externe Ziel-Packages wie `fmt` oder `go/ast` werden nicht unterstuetzt.
- Es werden keine neuen externen Go-Dependencies hinzugefuegt.

Wenn `go.mod` fehlt oder keine `module`-Direktive enthaelt, soll `FindPackageDependencies` einen Fehler zurueckgeben.

## 3. Nicht-Ziele

Diese Dinge werden in dieser Version nicht implementiert:

- Kein Call Graph.
- Keine Analyse einzelner Funktionen.
- Keine Build-Tag-Auswertung.
- Keine `go.work`-Unterstuetzung.
- Keine Unterscheidung zwischen normalen Imports und Test-Only-Imports.
- Keine Package-Analyse ueber mehrere Module hinweg.
- Keine grafische Ausgabe.
- Kein Cache.
- Kein Logging-, Monitoring- oder Tracing-System.

Wichtig:

`_test.go`-Dateien werden in dieser Version genauso wie andere `.go`-Dateien gescannt. Dadurch koennen Test-Only-Imports in `IMPORTS` erscheinen. Das ist fuer Version 1 akzeptiert und bewusst dokumentiert.

## 4. Betroffene Dateien

Neue Dateien:

- `internal/sherpa/dependency.go`
- `internal/sherpa/dependency_test.go`
- `internal/sherpa/dependency_output.go`
- `internal/sherpa/dependency_output_test.go`

Zu aendernde Dateien:

- `cmd/gosherpa/main.go`
- `README.md`

Nicht aendern:

- Bestehende Symbol- und Referenzfunktionen sollen fachlich unveraendert bleiben.
- Keine Umbenennung bestehender Public Functions.
- Keine neuen Third-Party-Packages.

## 5. Datenmodell

Datei: `internal/sherpa/dependency.go`

Fuege folgenden Typ hinzu:

```go
type PackageDependencies struct {
	Package string
	Imports []string
	UsedBy  []string
}
```

Bedeutung:

- `Package`: normalisierter lokaler Package-Pfad, zum Beispiel `./internal/sherpa` oder `.`.
- `Imports`: sortierte Liste der Imports des Ziel-Packages.
- `UsedBy`: sortierte Liste lokaler Packages, die das Ziel-Package importieren.

Formatregeln fuer `Imports`:

- Externe Imports bleiben roh, zum Beispiel `fmt`, `go/ast`, `github.com/other/lib`.
- Imports in das aktuelle Modul werden als lokale Package-Pfade angezeigt, zum Beispiel `./internal/auth`.

Formatregeln fuer `UsedBy`:

- Alle Eintraege sind lokale Package-Pfade.
- Beispiel: `./cmd/gosherpa`

## 6. Funktionsuebersicht

Implementiere in `internal/sherpa/dependency.go` genau diese Funktionen:

```go
func FindPackageDependencies(root string, targetPackage string) (PackageDependencies, error)
func modulePath(root string) (string, error)
func normalizeTargetPackage(targetPackage string, modulePath string) (string, error)
func packagePathForFile(root string, file string) (string, error)
func parseImports(path string) ([]string, error)
func collectPackageImports(root string) (map[string][]string, error)
func localPackagePath(importPath string, modulePath string) (string, bool)
func displayImportPath(importPath string, modulePath string) string
func uniqueSorted(values []string) []string
```

Nur `FindPackageDependencies` wird exportiert. Alle anderen Funktionen bleiben klein geschrieben und sind nur innerhalb des Package `sherpa` sichtbar.

## 7. Implementierungsschritte

### Schritt 1: `modulePath` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func modulePath(root string) (string, error)
```

Aufgabe:

Lies den Modulpfad aus `go.mod`.

Vorgehen:

1. Baue den Pfad mit `filepath.Join(root, "go.mod")`.
2. Lies die Datei mit `os.ReadFile`.
3. Teile den Inhalt mit `strings.Split(string(data), "\n")` in Zeilen.
4. Iteriere ueber alle Zeilen.
5. Entferne Whitespace mit `strings.TrimSpace`.
6. Ignoriere leere Zeilen.
7. Ignoriere Zeilen, die mit `//` beginnen.
8. Suche eine Zeile, die mit `module ` beginnt.
9. Nutze `strings.Fields(line)`.
10. Wenn mindestens zwei Felder vorhanden sind, gib Feld 2 zurueck.
11. Wenn keine gueltige `module`-Direktive gefunden wird, gib einen Fehler zurueck.

Erwartete Fehler:

- `go.mod` kann nicht gelesen werden.
- `go.mod` enthaelt keine gueltige `module`-Direktive.

Beispiel:

```text
module github.com/supertabaluga/gosherpa
```

Rueckgabe:

```text
github.com/supertabaluga/gosherpa
```

Akzeptanz fuer diesen Schritt:

- Modulpfad wird korrekt gelesen.
- Fehlende `go.mod` fuehrt zu einem Fehler.
- `go.mod` ohne `module` fuehrt zu einem Fehler.

### Schritt 2: `normalizeTargetPackage` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func normalizeTargetPackage(targetPackage string, modulePath string) (string, error)
```

Aufgabe:

Normalisiere die CLI-Eingabe auf einen lokalen Package-Pfad.

Diese Funktion wird nur fuer das Ziel aus der CLI verwendet. Sie wird nicht fuer Imports aus Go-Dateien verwendet.

Erlaubte Eingaben:

```text
.
./internal/sherpa
internal/sherpa
internal/sherpa/
github.com/supertabaluga/gosherpa/internal/sherpa
```

Erwartete Ausgaben:

```text
.                                                   -> .
./internal/sherpa                                  -> ./internal/sherpa
internal/sherpa                                    -> ./internal/sherpa
internal/sherpa/                                   -> ./internal/sherpa
github.com/supertabaluga/gosherpa/internal/sherpa  -> ./internal/sherpa
```

Vorgehen:

1. Entferne Whitespace mit `strings.TrimSpace`.
2. Wenn der Wert leer ist, gib einen Fehler zurueck.
3. Wenn `filepath.IsAbs(value)` true ist, gib einen Fehler zurueck.
4. Wandle Trennzeichen mit `filepath.ToSlash(value)` in Slash-Form um.
5. Wenn der Wert exakt dem Modulpfad entspricht, gib `"."` zurueck.
6. Wenn der Wert mit `modulePath + "/"` beginnt, entferne diesen Prefix.
7. Entferne einen fuehrenden `"./"`-Prefix.
8. Pruefe vor `path.Clean`, ob ein Pfadsegment exakt `".."` ist. Nutze dafuer `strings.Split(value, "/")`. Wenn ein Segment `".."` ist, gib einen Fehler zurueck.
9. Bereinige den Pfad mit `path.Clean`.
10. Wenn das Ergebnis `"."` ist, gib `"."` zurueck.
11. Wenn das Ergebnis `".."` ist oder mit `"../"` beginnt, gib einen Fehler zurueck.
12. Gib `"./" + cleaned` zurueck.

Hinweis:

Nutze fuer Schritt 9 das Package `path`, nicht `filepath`, weil der Wert nach `filepath.ToSlash` bereits Slash-basiert ist.

Akzeptanz fuer diesen Schritt:

- Relative lokale Packages werden zu `./...`.
- Modulpfad-Ziele werden zu `./...`.
- Leere, absolute und ausbrechende Pfade werden abgelehnt.

### Schritt 3: `packagePathForFile` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func packagePathForFile(root string, file string) (string, error)
```

Aufgabe:

Leite aus einem Dateipfad den lokalen Package-Pfad ab.

Beispiele:

```text
root=/repo, file=/repo/main.go                         -> .
root=/repo, file=/repo/internal/sherpa/parse.go        -> ./internal/sherpa
root=., file=internal/sherpa/parse.go                  -> ./internal/sherpa
root=., file=cmd/gosherpa/main.go                      -> ./cmd/gosherpa
```

Vorgehen:

1. Bestimme den Ordner mit `filepath.Dir(file)`.
2. Berechne den relativen Pfad mit `filepath.Rel(root, dir)`.
3. Wenn `filepath.Rel` einen Fehler liefert, gib diesen Fehler zurueck.
4. Wandle den relativen Pfad mit `filepath.ToSlash` in Slash-Form um.
5. Wenn der relative Pfad `"."` ist, gib `"."` zurueck.
6. Sonst gib `"./" + relativePath` zurueck.

Akzeptanz fuer diesen Schritt:

- Root-Dateien werden als Package `"."` erkannt.
- Unterordner werden als `./...` erkannt.
- Die Funktion funktioniert mit absoluten und relativen Testpfaden.

### Schritt 4: `parseImports` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func parseImports(path string) ([]string, error)
```

Aufgabe:

Lies alle Import-Pfade aus einer Go-Datei.

Vorgehen:

1. Erstelle ein `token.FileSet`.
2. Parse die Datei mit:

```go
parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
```

3. Iteriere ueber `file.Imports`.
4. Lies `importSpec.Path.Value`.
5. Entferne die Anfuehrungszeichen mit `strconv.Unquote`.
6. Wenn `strconv.Unquote` einen Fehler liefert, gib einen Fehler mit Kontext zurueck.
7. Sammle alle Import-Pfade.
8. Gib `uniqueSorted(imports)` zurueck.

Beispiele:

Go-Code:

```go
import (
	"fmt"
	"strings"
)
```

Rueckgabe:

```text
fmt
strings
```

Akzeptanz fuer diesen Schritt:

- Single Imports werden erkannt.
- Import-Bloecke werden erkannt.
- Alias-, Dot- und Blank-Imports liefern trotzdem nur den Pfad.
- Rueckgabe ist sortiert und dedupliziert.

### Schritt 5: `uniqueSorted` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func uniqueSorted(values []string) []string
```

Aufgabe:

Dedupliziere Strings und sortiere sie alphabetisch.

Vorgehen:

1. Erstelle eine Map `map[string]struct{}`.
2. Fuege jeden nicht-leeren Wert in die Map ein.
3. Erstelle eine Ergebnisliste aus den Keys.
4. Sortiere die Liste mit `sort.Strings`.
5. Gib die Liste zurueck.

Akzeptanz fuer diesen Schritt:

- Doppelte Werte erscheinen nur einmal.
- Leere Strings werden ignoriert.
- Rueckgabe ist alphabetisch sortiert.

### Schritt 6: `collectPackageImports` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func collectPackageImports(root string) (map[string][]string, error)
```

Aufgabe:

Scanne das Repository und sammle pro lokalem Package alle rohen Import-Pfade.

Wichtig:

Diese Funktion speichert Imports genau so, wie sie in Go-Dateien stehen. Sie wandelt lokale Imports noch nicht in `./...` um.

Vorgehen:

1. Rufe `FindGoFiles(root)` auf.
2. Erstelle `importsByPackage := map[string][]string{}`.
3. Iteriere ueber alle Dateien.
4. Bestimme das Package mit `packagePathForFile(root, file)`.
5. Stelle sicher, dass `importsByPackage[pkg]` existiert, auch wenn die Datei keine Imports hat.
6. Rufe `parseImports(file)` auf.
7. Fuege die Imports zur Liste fuer dieses Package hinzu.
8. Nach der Schleife: Dedupliziere und sortiere alle Listen mit `uniqueSorted`.
9. Gib die Map zurueck.

Akzeptanz fuer diesen Schritt:

- Packages ohne Imports erscheinen trotzdem in der Map.
- Imports aus mehreren Dateien desselben Packages werden zusammengefuehrt.
- Doppelte Imports pro Package werden entfernt.
- Die Importlisten sind sortiert.

### Schritt 7: `localPackagePath` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func localPackagePath(importPath string, modulePath string) (string, bool)
```

Aufgabe:

Erkenne, ob ein Import zum aktuellen Modul gehoert, und wandle ihn in einen lokalen Package-Pfad um.

Regeln:

```text
importPath == modulePath       -> ".", true
modulePath + "/internal/auth"  -> "./internal/auth", true
fmt                            -> "", false
go/ast                         -> "", false
github.com/other/lib           -> "", false
github.com/supertabaluga/gosherpa2/internal/auth -> "", false
```

Wichtig:

Nutze fuer Prefix-Pruefung nur `modulePath + "/"`, nicht einfach `strings.HasPrefix(importPath, modulePath)`. Sonst wuerde `github.com/app2` faelschlich zu `github.com/app` passen.

Akzeptanz fuer diesen Schritt:

- Exakter Modulpfad wird als `"."` erkannt.
- Unterpackages des Moduls werden als `./...` erkannt.
- Prefix-Kollisionen werden nicht als lokal erkannt.

### Schritt 8: `displayImportPath` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func displayImportPath(importPath string, modulePath string) string
```

Aufgabe:

Bereite einen Import-Pfad fuer die Ausgabe vor.

Vorgehen:

1. Rufe `localPackagePath(importPath, modulePath)` auf.
2. Wenn `ok == true`, gib den lokalen Package-Pfad zurueck.
3. Sonst gib `importPath` unveraendert zurueck.

Beispiele:

```text
fmt                                                     -> fmt
go/ast                                                  -> go/ast
github.com/supertabaluga/gosherpa/internal/sherpa       -> ./internal/sherpa
```

Akzeptanz fuer diesen Schritt:

- Externe Imports bleiben unveraendert.
- Lokale Imports werden als `./...` angezeigt.

### Schritt 9: `FindPackageDependencies` implementieren

Datei: `internal/sherpa/dependency.go`

Signatur:

```go
func FindPackageDependencies(root string, targetPackage string) (PackageDependencies, error)
```

Aufgabe:

Orchestriere die komplette Dependency-Analyse.

Vorgehen:

1. Lies den Modulpfad mit `modulePath(root)` und speichere ihn in einer lokalen Variable namens `modPath`.
2. Normalisiere das Ziel mit `normalizeTargetPackage(targetPackage, modPath)`.
3. Sammle alle Package-Imports mit `collectPackageImports(root)`.
4. Pruefe, ob das Ziel-Package in der Map existiert.
5. Wenn nicht, gib einen Fehler `package not found: <target>` zurueck.
6. Erstelle `deps := PackageDependencies{Package: target}`.
7. Nimm die rohen Imports des Ziel-Packages.
8. Wandle jeden rohen Import mit `displayImportPath` fuer die Ausgabe um.
9. Setze `deps.Imports = uniqueSorted(displayImports)`.
10. Iteriere ueber alle Packages in der Map.
11. Ueberspringe das Ziel-Package selbst.
12. Fuer jeden rohen Import eines Packages:
    - Rufe `localPackagePath(importPath, modPath)` auf.
    - Wenn `ok == true` und der lokale Pfad dem Ziel entspricht, fuege das Package zu `UsedBy` hinzu.
    - Brich die innere Schleife ab, damit dasselbe Package nur einmal hinzugefuegt wird.
13. Setze `deps.UsedBy = uniqueSorted(usedBy)`.
14. Gib `deps, nil` zurueck.

Akzeptanz fuer diesen Schritt:

- Ziel-Package wird normalisiert.
- Fehlendes Ziel-Package fuehrt zu Fehler.
- Imports sind sortiert und dedupliziert.
- UsedBy ist sortiert und dedupliziert.
- Das Ziel-Package taucht nicht in `UsedBy` auf.
- Lokale Imports werden fuer die Ausgabe als `./...` angezeigt.

## 8. Ausgabe implementieren

Datei: `internal/sherpa/dependency_output.go`

Implementiere:

```go
func FormatPackageDependencies(deps PackageDependencies) string
func PrintPackageDependencies(deps PackageDependencies)
```

`FormatPackageDependencies` baut den kompletten Ausgabetext und gibt ihn als String zurueck.

`PrintPackageDependencies` gibt den String aus:

```go
func PrintPackageDependencies(deps PackageDependencies) {
	fmt.Print(FormatPackageDependencies(deps))
}
```

Exaktes Ausgabeformat:

```text
PACKAGE
  ./internal/sherpa

IMPORTS
  fmt
  go/ast

USED BY
  ./cmd/gosherpa
```

Regeln:

- Zwischen den drei Abschnitten steht jeweils eine Leerzeile.
- Jede Liste ist mit zwei Leerzeichen eingerueckt.
- Wenn `Imports` leer ist, steht unter `IMPORTS` der Wert `none`.
- Wenn `UsedBy` leer ist, steht unter `USED BY` der Wert `none`.
- Der String endet mit genau einem Newline.
- Keine Unicode-Symbole in dieser neuen Ausgabe.
- Keine Farben.

Beispiel fuer leere Listen:

```text
PACKAGE
  ./internal/empty

IMPORTS
  none

USED BY
  none
```

Akzeptanz fuer diesen Schritt:

- Ausgabe ist deterministisch.
- Ausgabe kann ohne stdout-Capture getestet werden.
- Format entspricht exakt den Beispielen.

## 9. CLI anschliessen

Datei: `cmd/gosherpa/main.go`

### Schritt 9.1: `printUsage` einfuehren

Fuege in `main.go` diese Funktion hinzu:

```go
func printUsage() {
	fmt.Println("usage: gosherpa <command> [args]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  symbols")
	fmt.Println("  symbol <name>")
	fmt.Println("  refs <name>")
	fmt.Println("  deps <package>")
}
```

Nutze `printUsage()`:

- wenn kein Kommando angegeben wurde,
- wenn ein unbekanntes Kommando angegeben wurde.

### Schritt 9.2: `deps`-Case einfuegen

Fuege im `switch command` hinzu:

```go
case "deps":
	if len(os.Args) < 3 {
		fmt.Println("usage: gosherpa deps <package>")
		return
	}

	targetPackage := os.Args[2]

	deps, err := sherpa.FindPackageDependencies(".", targetPackage)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	sherpa.PrintPackageDependencies(deps)
```

Akzeptanz fuer diesen Schritt:

- `gosherpa deps ./internal/sherpa` ruft die neue Fachlogik auf.
- Fehler werden im Stil des bestehenden Codes als `error: ...` ausgegeben.
- Bestehende Kommandos funktionieren weiter.
- `gofmt` formatiert `main.go` ohne manuelle Nacharbeit.

## 10. README aktualisieren

Datei: `README.md`

Fuege unter "Current Features" einen Abschnitt fuer Package Dependencies hinzu.

Beispieltext fuer die README:

```text
### Package Dependencies

Show imports and local packages that depend on a package.

    gosherpa deps ./internal/sherpa
```

Passe den Roadmap-Abschnitt an:

- Entferne `Package Dependencies` aus den rein geplanten Features oder kennzeichne es als implementiert.
- Lasse `Impact Analysis` als zukuenftiges Feature stehen, weil es auf Dependencies aufbauen kann.

Akzeptanz fuer diesen Schritt:

- README nennt das neue Kommando.
- README widerspricht dem implementierten Feature nicht.
- Bestehende Beispiele bleiben gueltig.

## 11. Testplan

Alle neuen Tests koennen im Package `sherpa` geschrieben werden, nicht in `sherpa_test`. Dadurch duerfen Tests auch unexportierte Hilfsfunktionen pruefen.

### Datei: `internal/sherpa/dependency_test.go`

#### Test 1: Modulpfad wird gelesen

Name:

```go
func TestModulePath(t *testing.T)
```

Setup:

- Temp-Dir erstellen.
- `go.mod` mit `module example.com/app` schreiben.
- `modulePath(tmp)` aufrufen.

Erwartung:

- Rueckgabe ist `example.com/app`.
- Fehler ist `nil`.

#### Test 2: Fehlende go.mod liefert Fehler

Name:

```go
func TestModulePathReturnsErrorWhenGoModIsMissing(t *testing.T)
```

Setup:

- Temp-Dir ohne `go.mod`.
- `modulePath(tmp)` aufrufen.

Erwartung:

- Fehler ist nicht `nil`.

#### Test 3: go.mod ohne module-Direktive liefert Fehler

Name:

```go
func TestModulePathReturnsErrorWhenModuleDirectiveIsMissing(t *testing.T)
```

Setup:

- Temp-Dir erstellen.
- `go.mod` mit Inhalt `go 1.24.4` schreiben.
- `modulePath(tmp)` aufrufen.

Erwartung:

- Fehler ist nicht `nil`.

#### Test 4: Ziel-Package wird normalisiert

Name:

```go
func TestNormalizeTargetPackage(t *testing.T)
```

Testfaelle:

```text
.                         -> .
./internal/auth           -> ./internal/auth
internal/auth             -> ./internal/auth
internal/auth/            -> ./internal/auth
example.com/app           -> .
example.com/app/internal/auth -> ./internal/auth
```

Erwartung:

- Alle Faelle liefern den erwarteten Wert.
- Fehler ist `nil`.

#### Test 5: Ungueltige Ziel-Pfade werden abgelehnt

Name:

```go
func TestNormalizeTargetPackageRejectsInvalidInput(t *testing.T)
```

Testfaelle:

```text
""
"   "
"/tmp/project"
"../auth"
"internal/../auth"
```

Erwartung:

- Jeder Fall liefert einen Fehler.

#### Test 6: Package-Pfad wird aus Datei abgeleitet

Name:

```go
func TestPackagePathForFile(t *testing.T)
```

Setup:

- Temp-Dir erstellen.
- Dateien `main.go` und `internal/auth/service.go` schreiben.
- `packagePathForFile` fuer beide Dateien aufrufen.

Erwartung:

- Root-Datei ergibt `"."`.
- Datei in `internal/auth` ergibt `"./internal/auth"`.

#### Test 7: Imports werden aus Datei gelesen

Name:

```go
func TestParseImports(t *testing.T)
```

Setup:

Go-Datei mit:

```go
package auth

import (
	"fmt"
	alias "strings"
	_ "net/http"
	. "errors"
)
```

Erwartung:

- Rueckgabe enthaelt `errors`, `fmt`, `net/http`, `strings`.
- Rueckgabe ist sortiert.

#### Test 8: Imports werden pro Package gesammelt

Name:

```go
func TestCollectPackageImports(t *testing.T)
```

Setup:

- `go.mod` ist fuer diese Funktion nicht noetig.
- Zwei Dateien in `internal/auth`.
- Eine Datei importiert `fmt`.
- Die andere importiert `strings`.

Erwartung:

- Map enthaelt `./internal/auth`.
- Imports fuer `./internal/auth` enthalten `fmt` und `strings`.

#### Test 9: Packages ohne Imports bleiben sichtbar

Name:

```go
func TestCollectPackageImportsIncludesPackagesWithoutImports(t *testing.T)
```

Setup:

- Datei `internal/empty/empty.go` ohne Imports.

Erwartung:

- Map enthaelt `./internal/empty`.
- Importliste ist leer.

#### Test 10: Lokale Imports werden erkannt

Name:

```go
func TestLocalPackagePath(t *testing.T)
```

Testfaelle:

```text
example.com/app                       -> ".", true
example.com/app/internal/auth         -> "./internal/auth", true
fmt                                   -> "", false
go/ast                                -> "", false
example.com/app2/internal/auth        -> "", false
```

Erwartung:

- Alle Faelle liefern exakt den erwarteten Pfad und Boolean.

#### Test 11: Dependencies finden Imports

Name:

```go
func TestFindPackageDependenciesFindsImports(t *testing.T)
```

Setup:

- `go.mod` mit `module example.com/app`.
- Datei `internal/auth/service.go` importiert `fmt` und `strings`.

Aufruf:

```go
deps, err := FindPackageDependencies(tmp, "./internal/auth")
```

Erwartung:

- `err == nil`.
- `deps.Package == "./internal/auth"`.
- `deps.Imports` enthaelt `fmt` und `strings`.
- `deps.UsedBy` ist leer.

#### Test 12: Dependencies finden UsedBy

Name:

```go
func TestFindPackageDependenciesFindsUsedBy(t *testing.T)
```

Setup:

- `go.mod` mit `module example.com/app`.
- `internal/auth/service.go` existiert.
- `cmd/api/main.go` importiert `example.com/app/internal/auth`.

Aufruf:

```go
deps, err := FindPackageDependencies(tmp, "internal/auth")
```

Erwartung:

- `err == nil`.
- `deps.Package == "./internal/auth"`.
- `deps.UsedBy` enthaelt `./cmd/api`.

#### Test 13: Lokale Imports werden in Imports als lokale Pfade angezeigt

Name:

```go
func TestFindPackageDependenciesDisplaysLocalImportsAsLocalPaths(t *testing.T)
```

Setup:

- `go.mod` mit `module example.com/app`.
- `internal/store/store.go` existiert.
- `internal/auth/service.go` importiert `example.com/app/internal/store`.

Erwartung:

- `deps.Imports` fuer `./internal/auth` enthaelt `./internal/store`.
- `deps.Imports` enthaelt nicht `example.com/app/internal/store`.

#### Test 14: Fehlendes Ziel-Package liefert Fehler

Name:

```go
func TestFindPackageDependenciesReturnsErrorForMissingPackage(t *testing.T)
```

Setup:

- `go.mod` mit `module example.com/app`.
- Eine beliebige Go-Datei in einem anderen Package.

Aufruf:

```go
_, err := FindPackageDependencies(tmp, "./internal/missing")
```

Erwartung:

- Fehler ist nicht `nil`.

#### Test 15: Deduplizierung

Name:

```go
func TestFindPackageDependenciesDeduplicatesImportsAndUsedBy(t *testing.T)
```

Setup:

- Zwei Dateien im selben Package importieren `fmt`.
- Ein nutzendes Package importiert das Ziel in mehreren Dateien.

Erwartung:

- `fmt` steht nur einmal in `Imports`.
- Das nutzende Package steht nur einmal in `UsedBy`.

### Datei: `internal/sherpa/dependency_output_test.go`

#### Test 16: Ausgabeformat mit Werten

Name:

```go
func TestFormatPackageDependencies(t *testing.T)
```

Input:

```go
deps := PackageDependencies{
	Package: "./internal/sherpa",
	Imports: []string{"fmt", "go/ast"},
	UsedBy:  []string{"./cmd/gosherpa"},
}
```

Erwarteter String:

```text
PACKAGE
  ./internal/sherpa

IMPORTS
  fmt
  go/ast

USED BY
  ./cmd/gosherpa
```

Der erwartete String muss mit einem Newline enden.

#### Test 17: Ausgabeformat mit leeren Listen

Name:

```go
func TestFormatPackageDependenciesWithEmptyLists(t *testing.T)
```

Erwartung:

- Unter `IMPORTS` steht `none`.
- Unter `USED BY` steht `none`.
- Der String endet mit genau einem Newline.

## 12. Test-Helfer

Lege in `internal/sherpa/dependency_test.go` diese Helper an.

```go
func writeFile(t *testing.T, path string, contents string)
```

Verhalten:

1. `t.Helper()` aufrufen.
2. Parent-Ordner mit `os.MkdirAll(filepath.Dir(path), 0755)` erstellen.
3. Datei mit `os.WriteFile(path, []byte(contents), 0644)` schreiben.
4. Bei Fehlern `t.Fatal(err)`.

```go
func containsString(values []string, want string) bool
```

Verhalten:

- Gibt `true` zurueck, wenn `want` in `values` enthalten ist.

```go
func countString(values []string, want string) int
```

Verhalten:

- Zaehlt, wie oft `want` in `values` vorkommt.

Optional, aber hilfreich:

```go
func assertContainsString(t *testing.T, values []string, want string)
```

Verhalten:

- Ruft `t.Helper()` auf.
- Schlaegt fehl, wenn `want` nicht enthalten ist.

## 13. Formatierung und Verifikation

Nach der Implementierung ausfuehren:

```bash
gofmt -w cmd/gosherpa/main.go internal/sherpa/dependency.go internal/sherpa/dependency_output.go internal/sherpa/dependency_test.go internal/sherpa/dependency_output_test.go
go test ./...
```

Falls `go test ./...` wegen eines nicht beschreibbaren Go-Build-Caches fehlschlaegt, lokal im Repository testen:

```bash
GOCACHE=.cache/go-build go test ./...
```

Manuelle Verifikation:

```bash
go run ./cmd/gosherpa deps ./internal/sherpa
go run ./cmd/gosherpa deps internal/sherpa
go run ./cmd/gosherpa deps github.com/supertabaluga/gosherpa/internal/sherpa
go run ./cmd/gosherpa deps ./does/not/exist
```

Erwartung:

- Die ersten drei Kommandos zeigen dasselbe Ziel-Package.
- Das vierte Kommando gibt einen Fehler aus.

## 14. Sicherheit

Dieses Feature verarbeitet lokale Dateien im Repository.

Sicherheitsregeln:

- Keine Netzwerkaufrufe.
- Keine Ausfuehrung von fremdem Code.
- Kein Schreiben von Analyseergebnissen auf die Platte.
- Keine Secrets oder Credentials lesen, ausser sie stehen versehentlich in Go-Dateien; GoSherpa gibt nur Import-Pfade aus.
- Ziel-Package-Eingaben mit leerem Wert, absolutem Pfad oder `..` werden abgelehnt.

Akzeptanz:

- Analyse bleibt read-only.
- Ungueltige Zielpfade fuehren zu Fehlern.

## 15. Performance

Performance-Regeln fuer Version 1:

- Verwende `parser.ImportsOnly`, nicht komplettes AST-Parsing.
- Scanne das Repository einmal pro Kommando.
- Keine Goroutines.
- Kein Cache.

Begruendung:

Das aktuelle Projekt ist ein fruehes MVP. Ein einfacher linearer Scan ist leichter zu verstehen, zu testen und zu warten.

Akzeptanz:

- Laufzeit ist ungefaehr proportional zur Anzahl der `.go`-Dateien.
- Keine unnoetigen mehrfachen Scans innerhalb eines `deps`-Aufrufs.

## 16. Betrieb und Wartung

Betriebsannahmen:

- GoSherpa ist ein lokales CLI-Tool.
- Es gibt keinen Serverprozess.
- Es gibt keine Datenbank.
- Es gibt keine Migrationen.
- Es gibt kein Deployment-Risiko durch Runtime-Konfiguration.

Wartungsregeln:

- Neue Logik bleibt in `internal/sherpa/dependency.go`.
- Ausgabe bleibt in `internal/sherpa/dependency_output.go`.
- CLI bleibt in `cmd/gosherpa/main.go`.
- Tests liegen neben der Fachlogik.
- Keine Vermischung mit Symbol- oder Referenzsuche.

## 17. Definition of Done

Das Feature ist fertig, wenn alle Punkte erfuellt sind:

- `internal/sherpa/dependency.go` existiert.
- `internal/sherpa/dependency_output.go` existiert.
- `internal/sherpa/dependency_test.go` existiert.
- `internal/sherpa/dependency_output_test.go` existiert.
- `cmd/gosherpa/main.go` enthaelt den `deps`-Command.
- `cmd/gosherpa/main.go` enthaelt `printUsage()`.
- `README.md` dokumentiert `gosherpa deps`.
- `go test ./...` laeuft erfolgreich.
- `go run ./cmd/gosherpa deps ./internal/sherpa` laeuft erfolgreich.
- Fehlende `go.mod` wird getestet.
- `go.mod` ohne `module`-Direktive wird getestet.
- Fehlendes Ziel-Package wird getestet.
- Lokale Imports werden in `Imports` als `./...` angezeigt.
- `UsedBy` enthaelt nur lokale Package-Pfade.
- `Imports` und `UsedBy` sind sortiert.
- Doppelte Eintraege werden entfernt.
- Bestehende Kommandos `symbols`, `symbol` und `refs` bleiben funktionsfaehig.

## 18. Empfohlene Reihenfolge fuer die Umsetzung

1. `internal/sherpa/dependency.go` erstellen.
2. `PackageDependencies` definieren.
3. `uniqueSorted` implementieren und testen.
4. `modulePath` implementieren und testen.
5. `normalizeTargetPackage` implementieren und testen.
6. `packagePathForFile` implementieren und testen.
7. `parseImports` implementieren und testen.
8. `collectPackageImports` implementieren und testen.
9. `localPackagePath` implementieren und testen.
10. `displayImportPath` implementieren und testen, falls nicht indirekt abgedeckt.
11. `FindPackageDependencies` implementieren und testen.
12. `internal/sherpa/dependency_output.go` erstellen.
13. `FormatPackageDependencies` implementieren und testen.
14. `PrintPackageDependencies` implementieren.
15. `cmd/gosherpa/main.go` anpassen.
16. `README.md` aktualisieren.
17. `gofmt` ausfuehren.
18. `go test ./...` ausfuehren.
19. Manuelle Verifikationskommandos ausfuehren.

## 19. Typische Fehlerquellen

- `ImportSpec.Path.Value` enthaelt Anfuehrungszeichen. Deshalb `strconv.Unquote` verwenden.
- `strings.HasPrefix(importPath, modulePath)` ist falsch. Immer `modulePath + "/"` verwenden.
- Ein Package kann mehrere Dateien haben. Imports muessen zusammengefuehrt werden.
- Packages ohne Imports muessen trotzdem in der Map auftauchen.
- `filepath.Rel` kann OS-spezifische Separatoren liefern. Danach `filepath.ToSlash` verwenden.
- `normalizeTargetPackage` ist nicht fuer externe Imports gedacht.
- `go/ast` ist ein externer Import und darf nicht zu `./go/ast` werden.
- Das Ziel-Package darf nicht in `UsedBy` erscheinen.
- Ausgabe-Strings sollen mit genau einem Newline enden.
- Nach CLI-Aenderungen immer `gofmt` laufen lassen.
