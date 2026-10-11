# Prüfung der Version 0.1.0-dev

Stand: 11. Oktober 2026. Lokal geprüft; Veröffentlichung und Zielserver-Installation folgen erst nach Abnahme.

## Automatische und manuelle Prüfungen

- Go-Tests mit Race-Detector: Parser/Prioritäten, Codec-Familien, Token-Grenzen, unbekannte Merkmale, kombinierte Sprachen/HDR, AND/OR, FTS-Suche, SQL-Parametrisierung, Staffelkontext, Mehrfachepisoden, Specials, Pagination, CSV/Preset-Filter, Login/CSRF, API-Key-Maskierung/Verschlüsselung, begrenzte Loginversuche, Neustart, Backup-Retention, Migration und Wiederherstellung.
- Mock-APIs prüfen ausschließlich zugelassene GET-Aufrufe und den API-Key-Header. Abgelehnter Key, 5xx, Abbruch und Redirect erhalten den alten Cache beziehungsweise geben keinen Key an andere Ziele weiter.
- Frontend-Typecheck und Vite-Produktionsbuild, Go vet, Linux-Crossbuilds für amd64 und arm64.
- Compose-Konfiguration validiert; XML syntaktisch geprüft.
- Tabellenüberschriften: Titel, Jahr, Auflösung, Codec, Source und optionale Details auf-/absteigend; numerische Auflösung, unbekannte Werte am Ende, ungültige Sortierspalte/-richtung abgewiesen. Statistiktests prüfen eindeutige Film-/Serien-IDs, Mehrfachepisoden, doppelte Zuordnungen, Specials, unbekannte Kategorien und Summen je Dienst. Statistik-Endpunkt verlangt Anmeldung.
- Browser: `1080p + x264` liefert neun ausschließlich passende Filmzeilen im Demo-Bestand. Serien: 81 passende 1080p-Dateien, 40 zusätzliche als Kontext gekennzeichnete Dateien. Einstellungen, Preset-Dropdown/Speichern/Laden, vollständig einklappbare Filter und Metadaten-Dialog geprüft. Individuelle x264-Farbe in beiden Designs und nach App-Neustart bestätigt; Preset bleibt im Dropdown verfügbar. Tastatur-Escape schließt den Dialog. Neue Statistikseite: Demo zählt 26 Filme, 5 Serien, 123 eindeutige Episoden und 148 Dateien. Diagramme für Gesamtbestand/Filme/Serien geprüft; Sortierung bleibt nach Neuladen, Preset-Wahl und Statistikwechsel erhalten. Tastatur-Enter wechselt die Sortierrichtung. Beide Themes und 390-Pixel-Mobile-Layout geprüft; horizontales Scrollen beschränkt sich auf die Tabelle.

Screenshots: [Desktop](screenshots/desktop.jpg), [Serien](screenshots/series.jpg), [Mobil](screenshots/mobile.jpg), [Hell](screenshots/light.jpg). [Statistiken](screenshots/statistics.jpg), [Statistiken mobil](screenshots/statistics-mobile.jpg), [Spaltensortierung](screenshots/column-sorting.jpg). Beispieldaten sind synthetisch und keine reale Bibliothek des Betreibers.

## Last und Speicher

Native macOS/arm64-Umgebung, 15 vom Host gemeldete logische CPUs, `GOMAXPROCS=2`, lokale SQLite-Dateien. Go 1.27.2. Die Produktions-App verwendet `GOMEMLIMIT=192MiB`. Messungen gelten für diese Umgebung und Datenverteilung.

| Messung | Ergebnis |
| --- | ---: |
| SQLite-Test, je 50.000 Dateien pro Dienst, gemischte Auflösung/Codec-, Text- und Jahresfilter, zusätzliche Spalten-/Richtungssortierung, Ausschlüsse sowie alle/nicht-alle Audio-/HDR-/Sprachfilter, beide Serienmodi, 60 Anfragen | p50 49,0 ms; p95 108,7 ms |
| Statistikaggregation, 100.000 Dateien in einem Cache-Snapshot | 310,6 ms |
| Vollständiger Radarr-Sync über lokale Mock-API, 50.000 Dateien | 2,22 s |
| HTTP-Filter `1080p + x264`, 50.000 Treffer, erste 100 als Seite, 60 Anfragen | p95 28,8 ms |
| App-Prozess, Spitze während Sync und HTTP-Abfragen | 207,0 MiB RSS |
| Derselbe 50k-Cache nach Neustart und einer Trefferseite | 26,8 MiB RSS-Spitze |

Der SQLite-Test umfasst auch Trefferzählung und JSON-Dekodierung einer Seite. Der HTTP-Test umfasst Authentifizierung, Abfrage und JSON-Antwort. Der Fixture-Server läuft separat; sein Speicher wird nicht zur App gerechnet. Die Idle-Messung ist eine kurze Messung nach Neustart, keine Langzeitbeobachtung. Messungen liefen teilweise gleichzeitig und sind kein isolierter Hardware-Benchmark.

Reproduzierbar: `GOMAXPROCS=2 ARRSCOPE_PERF=1 go test -run TestPerformance50K -count=1 -v ./internal/app`. Ein separater Fixture-Test ist unter `docs/scripts/loadcheck.py` verfügbar: `python3 docs/scripts/loadcheck.py /pfad/zur/arrscope-binary`.

## Containerprüfung und offene Punkte

Docker 29.4.0 mit OrbStack auf macOS/arm64: beide Linux-Images lokal gebaut, passende statische ELF-Architektur geprüft, Healthchecks für arm64 und amd64 bestanden (amd64 emuliert). ARM-Container mit UID/GID 99:100, schreibgeschütztem Root-Dateisystem, entfernten Capabilities, no-new-privileges, 32 MiB /tmp und 256 MiB Speicherlimit gestartet. WAL-Dateien und Schlüssel gehören 99:100. Nach Container-Neustart blieben Anmeldung, Bibliothek, Preset und eigene Tag-Farbe erhalten. Demo-Container: 10,9 MiB Speicher zum Messzeitpunkt. Der 50k-Speichertest oben bleibt eine native Messung; das Container-Limit allein belegt noch keinen 50k-Container-Sync. CI-/Release-Workflows sind vorbereitet, aber erst nach Veröffentlichung ausführbar.

Echte Radarr-/Sonarr-Daten, Erreichbarkeit aus dem Docker-VLAN, langlaufender Speicherverbrauch, TLS/Proxy-Betrieb und der Update-/Rollback-Ablauf auf Unraid müssen bei der Einrichtung geprüft werden. Die Memory-Ziele sind in den synthetischen lokalen Tests erreicht, keine feste Zusicherung für beliebig große API-Antworten oder Bibliotheken. Die automatische Migration/Wiederherstellung wurde auf SQLite-Ebene geprüft, noch nicht als echter Unraid-Containerwechsel.

## Ergänzungen vom 11.10.2026

- Build/TypeScript, `go test -race ./...` und `go vet ./...` erfolgreich. Spaltenauswahl: Validierung, ältere Einstellungen, getrennte Dienstdefaults, authentifiziertes Speichern/Lesen, Ablehnung ungültiger Auswahl ohne Änderung und Datenbank-Neustart getestet.
- Browser: Filmspalten Jahr aus/Größe und HDR an; Serien nur Episode und Codec. Überschriften und Zellen stimmen überein; Staffelgruppen verwenden `colspan=2`. Auswahl nach Neuladen und echtem Docker-Neustart bestätigt; anschließend beide Dienste auf ihre ursprünglichen fünf Standardspalten zurückgesetzt. Drei vorhandene Presets und vier Tag-Farben erhalten.
- Ausgeklappt stehen sämtliche Filter sofort zur Verfügung, minimiert fehlen alle Eingabefelder. Kein Zwischenzustand mehr.
- Erneute Anmeldung öffnet Dashboard; Menüfolge Dashboard, Radarr, Sonarr. Keine Browserfehler oder Warnungen.
- Screenshots: [Dashboard](screenshots/dashboard.jpg), [Standardspalten](screenshots/default-columns.jpg), [Alle Filter](screenshots/all-filters.jpg).

- Standardspalten bei 390 Pixeln: beide Auswahlgruppen stehen untereinander. Das mobile Menü zeigt Dashboard/Radarr/Sonarr ohne Zusatzlabel, damit alle drei Einträge innerhalb der Seitenbreite bleiben.

## Spaltenreihenfolge und vollständige Ansicht

- `go test -race ./...`, `go vet ./...` und Frontend-/Container-Build erfolgreich. Reihenfolge je Dienst: Legacy-Defaults, unvollständige/ungültige/duplizierte Listen, authentifizierter Settings-GET/PUT und Erhalt nach Datenbank-Neustart getestet.
- Browser: Codec vor Titel verschoben, ausgeblendetes HDR hinter Titel; sechs sichtbare Filmspalten verwenden diese Reihenfolge, vollständige Ansicht zeigt elf einschließlich HDR am konfigurierten Platz. Tabellenkopf und Alien-Zellen stimmen überein. Serien mit Jahr vor Episode und Sprache vor Kanälen geprüft; Staffelzeilen verwenden fünf beziehungsweise elf Spalten. Ein-/Ausblenden verändert die Defaults nicht. Enter bedient die Pfeile.
- Reale Docker-Neustartprüfung bestätigt beide gespeicherten Reihenfolgen. Bei 390 Pixeln bleiben Spaltenauswahl und Pfeile innerhalb der Seitenbreite. Schnelle Änderungen verwenden den jeweils aktuellen Zustand, damit Auswahl und Reihenfolge beider Dienste erhalten bleiben.
- Vorhandene drei Presets und vier Tag-Farben bleiben erhalten. Nach QA ursprüngliche Auswahl wiederhergestellt: Filme sechs Spalten einschließlich Größe, Serien fünf.
- Screenshots: [Reihenfolge](screenshots/column-order-settings.png), [Alle Spalten](screenshots/all-columns.png), [Mobil](screenshots/column-order-mobile.png).

## Ausschlussfilter

- Backend: `go test -race ./...` und `go vet ./...` erfolgreich. Alle zehn Merkmalsfilter prüfen positive/negative Mehrfachwerte, unbekannte Angaben, Komponenten in Sprache/HDR, Grenzen von Tokens und SQL-Injection-Werte. Ungültige und doppelte Operatoren werden abgewiesen. Kombiniertes 1080p/x265 ohne BluRay/WEB-DL, Pagination, Staffelkontext und Ausschluss zusätzlicher Kontextdateien beim Export getestet.
- Authentifizierte HTTP-Tests: Preset speichert/liest den negativen Operator; Liste und CSV liefern dieselbe Datei. Ungültige Operatoren liefern HTTP 400 bei Liste, CSV und Preset. Alte Presets ohne Operator behalten ihre Bedeutung.
- Browser: Praxisbeispiel liefert vier Filmdateien, alle 1080p/x265/WEBRip. Zwei schnell gewählte Ausschlüsse bleiben gleichzeitig erhalten. Wechsel auf „ist“ liefert null Treffer; Wechsel zurück wieder vier. Gespeichertes Beispielpreset lässt sich zurücksetzen und erneut korrekt laden. Minimierter Filter zeigt beide Source-Ausschlüsse.
- Sonarr: „Source ist nicht BluRay“ liefert 92 tatsächliche Treffer und 30 zusätzliche Dateien als gekennzeichneten Staffelkontext. Auf Seite 1 enthalten alle 76 tatsächlichen Treffer ausschließlich WEB-DL/WEBRip; 24 weitere Zeilen sind Kontext.
- Docker-Neustart: Beispielpreset und Operator erhalten; reale lokale API-Liste und CSV enthalten identische vier Dateinamen. Vorhandene drei Presets und vier Tag-Farben bleiben erhalten, zusätzlich ist das Beispielpreset vorhanden.
- Helles/dunkles Design und mobile Breite 390 Pixel geprüft; Operatoren passen in den Filterblock, Seitenbreite bleibt 390 Pixel. Keine Browserwarnungen oder Fehler.
- Screenshots: [Ausschlussfilter](screenshots/exclusion-filters.png), [Treffer](screenshots/exclusion-results.png), [Mobil](screenshots/exclusion-mobile.png), [Dunkles Design](screenshots/exclusion-dark.png).

## Mehrfachmerkmale und Emby-Ergänzung · 11. Oktober

- Automatische Tests: vier Operatoren `eq`, `not`, `all`, `notAll` für Audio/HDR/Sprache, unterschiedliche Kombinationen einschließlich unbekannter Felder, komplette/teilweise/fehlende Treffer, Staffelkontext und Export. HTTP-Presets erhalten die neuen Operatoren; Liste und CSV liefern dieselben Dateien. Einzelwertige Felder akzeptieren weiterhin nur ist/ist-nicht.
- Audio-Parser erkennt mehrere Tokens und verhindert Doppelzählung von DTS-HD MA als DTS-HD/DTS sowie E-AC-3 als AC-3. Wiederholte und benachbarte Tokens, Slash-Listen und Wortgrenzen getestet. HDR-Tags ergänzen explizite Profile aus bestehenden API-Daten; DV erzeugt keinen angenommenen HDR10-Fallback. ISO-Sprachcodes ger/deu/eng/fra/fre/jpn werden normalisiert; und bleibt unbekannt.
- Migration v2→v3: gespeicherte Merkmale und JSON konsistent repariert; Einstellungen, Presets und API-Evidence erhalten. Sicherung vor Migration bestätigt. Ältere Migration v1 und Wiederherstellung weiterhin getestet.
- Emby-Mocks: Header-Key, ausschließlich GET für System/Info und Items, 250er-Pagination, Mehrfachversionen, genaue Größen, abweichende Containerpfade, doppelte Quellen/Dateien, Mehrfachepisoden mit identischen Source-Daten, fehlende Größen/Streams, unbekannte Tonspuren, externe Audio- und Subtitle-Streams, DTS-Profile, Cache-Persistenz und per-track CSV. Vollständigkeit der Seiten und Totals geprüft.
- Emby-Fehler: 401, 5xx, Redirect, Timeout und Abbruch; keine Key-Weitergabe und keine sensiblen Antworttexte. Der letzte vollständige Dienst-Cache bleibt bei Fehler erhalten. Emby-Key in Einstellungen maskiert, verschlüsselt gespeichert, nach Neustart entschlüsselt und bei leerem Eingabefeld beibehalten. Verbindungstest nutzt gespeicherten Key, ohne Einstellungen zu ändern.
- Browser: Audio enthält alle AC-3/DTS liefert Arrival und Blade Runner 2049; HDR enthält alle HDR10/Dolby Vision liefert nur Blade Runner, enthält nicht alle liefert die übrigen 25 Filme. Audio und HDR erscheinen als separate Tags. Emby-Tonspuren zeigen Deutsch/DTS und Englisch/AC-3 samt 5.1 und Standardspur. In der Sprachspalte und im Dateidialog sichtbar. Vier bestehende Presets, vier eigene Farben und die aktuelle Spaltenauswahl erhalten.
- Mobil: Filter bei 390 Pixel ohne Seitenüberlauf; alle Operatoren innerhalb der Breite. Tonspurdialog 362 Pixel, Inhalt 360 Pixel, Seitenbreite 390 Pixel. Desktop-/Mobilprüfung mit Beispieldaten; die Emby-Daten der Demo sind synthetisch.
- Reale Erreichbarkeit: öffentlicher Emby-Systemstatus erreichbar, vom Nutzer genannte Version bestätigt. Authentifizierter Bibliothekszugriff ist noch nicht geprüft; dafür fehlt die direkte Key-Eingabe bei der echten Einrichtung. Keine realen Keys ausgelesen.

Screenshots: [Mehrere Tags](screenshots/compound-audio-tags.png), [HDR mit UND](screenshots/compound-hdr-all.png), [Tonspurzuordnung](screenshots/emby-audio-tracks.png), [Tonspuren mobil](screenshots/emby-tracks-mobile.png), [Filter mobil](screenshots/compound-mobile.png).

Zusätzlicher Lasttest, dieselbe native Testumgebung, **50.000 Filme mit jeweils zwei Tonspuren** über paginierte Mock-Emby-API: Sync 3,244 s, HTTP-Filter p95 40,8 ms, Prozessspitze 219,9 MiB RSS und 27,0 MiB nach Neustart. Zwei korrekte Sprach-/Codec-Zuordnungen pro Datei und Kombinationsfilter geprüft. Dieser Test ist kein Messwert des echten Emby-Servers und kein 50k-Test unter dem Container-Speicherlimit. Reproduzierbar mit `python3 docs/scripts/loadcheck.py /pfad/zur/arrscope-binary --emby`.


## Abschließende Korrekturrunde

Die Befunde der Abschlussprüfung wurden korrigiert und erneut geprüft. Aktuelle Änderungen, Regressionstests, Browsernachweise und Messwerte: [Korrekturbericht](REVIEW-FIXES.md). 100.000 Dateien mit je zwei Emby-Tonspuren laufen jetzt im Container ohne Swap erfolgreich durch (10,04 s, Filter p95 43,4 ms). Zwischengespeicherte Dashboard-/Filterwerte-Abfragen liegen unter 1 ms; Erstberechnungen benötigen bei diesem Bestand länger. Der cgroup-Peak erreicht das harte 256-MiB-Limit, die stichprobenweise App-RSS-Spitze beträgt 207,9 MiB. Der große Bestand ist damit konkret geprüft; keine Zusicherung für beliebig große Bibliotheken. Vorhandene vier Presets, vier Farben und Darstellungseinstellungen bleiben beim realen lokalen Containerwechsel erhalten.
