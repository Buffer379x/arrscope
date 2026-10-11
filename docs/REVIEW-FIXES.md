# ArrScope – umgesetzte Korrekturen

Stand: 11. Oktober 2026, Version 0.1.0-dev. Die Befunde der Abschlussprüfung sind korrigiert; die lokale Vorschau wurde mit dem bestehenden Datenvolume aktualisiert. Ihre 148 Beispieldateien, vier Presets, vier eigenen Tag-Farben sowie Spaltenauswahl und Reihenfolge blieben erhalten. GitHub-Veröffentlichung, Release und produktive Unraid-Installation folgen nach Abnahme.

## Änderungen

| Bereich | Ergebnis |
| --- | --- |
| Abhängigkeiten | Preact von 10.27.2 auf den korrigierten Patch 10.27.3 aktualisiert; Lockfile angepasst. |
| Dienstantworten | `null`, ungültiger JSON-Nachlauf, zu große Antworten, ungültige/doppelte Katalog- und Datei-IDs sowie unpassende Dateizuordnungen werden abgewiesen. Der letzte vollständige Cache bleibt erhalten. Gültige leere Listen bleiben zulässig. |
| Sync und Speicher | Globale Warteschlange statt zweier gleichzeitig vollständig aufgebauter Snapshots. Wartende Aufträge sind sichtbar und abbrechbar. Sonarr lädt Episoden und Dateien einer Serie mit höchstens zwei parallelen GET-Anfragen. Docker/Compose/Unraid-Template deaktivieren Swap ausdrücklich. |
| SQL und Aggregate | Anfragekontexte bis SQLite, gemeinsame Lesesnapshots, defensiv kopierte Facets-/Dashboard-Caches und Invalidierung nach erfolgreicher Übernahme. |
| Browseranfragen | Veraltete Ergebnis-, Filterwerte-, Dashboard- und Statusantworten überschreiben keinen neueren Zustand. Hintergrund-Polling pausiert; angemeldeter Leerlauf 30 Sekunden, Anmeldung 60 Sekunden, aktiver Sync 3 Sekunden. |
| Sprachen | Reihenfolge wird normalisiert; gleiche Mengen erzeugen keinen Konflikt. Zusätzliche ausdrücklich gemeldete Arr-Sprachen ergänzen erkannte Dateinamen-Sprachen. |
| HDR und Tonspuren | Emby-Video-HDR wird auch ohne Audio-Metadaten ergänzt. Teilweise unbekannte Audio-/Sprachmerkmale erscheinen im Filter „Unbekannt“ und im neuen Vollständigkeitsstatus. |
| Datenqualität | Filter „Metadaten-Vollständigkeit“, Hinweis bei unvollständigen Angaben und Dashboard-Widgets für Vollständigkeit und Emby-Zuordnung. Fehlendes HDR allein bedeutet keinen Fehler: Es kann SDR oder unbekannt sein. Negative Filter beziehen sich auf erkannte Angaben. |
| Emby-Identität | Optionale absolute Pfadzuordnungen als zusätzliche Prüfung neben Dateiname und Byte-Größe. Ohne passenden Pfad keine Ergänzung; mehrdeutige Kandidaten bleiben ausgeschlossen. Keine Medienzugriffe oder Scans. |
| Sitzungen und Keys | Passwortwechsel und Sitzungswiderruf sind atomar. Zusätzlich verhindert eine SQL-Prüfung des verifizierten Passwort-Hashs verspätete Sitzungen beziehungsweise Passwortänderungen nach einem parallelen Reset. Gespeicherte Dienst-Keys können ausdrücklich gelöscht werden; leere Eingaben behalten sie weiterhin bei. |
| CSV | Gleiche Filter und Sortierung wie die Liste. Der vollständige Export wird zunächst geschützt unter Appdata erstellt; SQL-, JSON- und Schreibfehler liefern einen Fehler statt eines scheinbar erfolgreichen Teil-Exports. |
| Wartbarkeit | Großes Frontend in Komponenten und Request-Hooks aufgeteilt, gemeinsame UI-Helfer, getrennte Styles und gemeinsame Dienst-Konfigurationshelfer. Überholte kleine mobile Metadatenregeln entfernt. |
| Design und Bedienung | Einstellungen in Verbindungen, Darstellung und Wartung mit erreichbarer Speichern-Leiste. Mobile Metadaten als Eigenschaftskarten mit 12-Pixel-Schrift. Einheitliche Operatorzeilen, erläutertes „enthält nicht alle“, Escape mit Fokus-Rückgabe und zugänglich benannte Dialoge. Eigene Tag-Farben wählen Schwarz oder Weiß nach tatsächlichem Kontrast. |
| Automatische Prüfungen | CI/Release um Frontend-Kontrasttests, pnpm audit, govulncheck, Laufzeitimage-Scan und synthetischen 100.000-Dateien-Test ohne Swap ergänzt. Die Workflows sind vorbereitet; auf GitHub werden sie erst nach Veröffentlichung ausgeführt. |

Beim zusätzlichen Container-Test fiel ein Fehler großer Dashboard-Abfragen auf. SQLite kann jetzt mit `SQLITE_TMPDIR=/data` temporäre Sortierdateien auf dem lokalen Appdata-Datenträger anlegen. Das Root-Dateisystem bleibt schreibgeschützt und `/tmp` auf 32 MiB begrenzt. Deshalb benötigt Appdata neben dem Cache auch freien Platz für Sortierungen und CSV-Exporte.

## Prüfung und Messwerte

Erfolgreich: `go test -race -count=1 ./...` (46,52 Sekunden), `go vet ./...`, TypeScript-Prüfung, Produktionsbuild, Frontend-Tests, Builds und Healthchecks für Linux/arm64 und Linux/amd64. amd64 wurde lokal emuliert. Compose-Konfiguration, XML und Workflow-YAML wurden auf Syntax geprüft.

Regressionstests prüfen ungültige Dienstantworten mit Cache-Erhalt, Warteschlange und Abbruch, Sprachmengen, HDR ohne Audio, teilweise unbekannte Tonspuren, Aggregate-Invalidierung und defensives Kopieren, abgebrochene Kontexte, CSV-Sortierung und beschädigte Daten, ausdrückliche Key-Löschung samt maskierter Antwort, Pfadzuordnung sowie Migration v3→v4 mit Emby-Tonspuren, Presets und Darstellung. Ein absichtlich fehlschlagender Sitzungswiderruf lässt das alte Passwort unverändert. Ein deterministischer Test blockiert veraltete Anmeldung und Passwortänderung nach Reset. Die vorhandenen Tests für lesende API-Endpunkte, CSRF, Key-Schutz, negative/Kombinationsfilter, Specials und Mehrfachepisoden bestehen weiterhin.

Frontend-Kontrasttests prüfen 4.096 Farbkombinationen und ungültige Farbwerte; der reproduzierte Grenzfall `#777777` erreicht mit Schwarz mindestens 4,5:1. Browserprüfung: drei Einstellungsbereiche, Speichern-Leiste, Escape und Fokus-Rückgabe, neuer Vollständigkeitsfilter mit zwei Film-Treffern sowie helles/dunkles Design bei 390 Pixeln. Metadaten-Dialog: 362 Pixel breit, Seite 390 Pixel ohne Überlauf, Eigenschaftstexte 12 Pixel. Keine Browserwarnungen oder Fehler bei der abschließenden Kontrolle.

| Messung | Ergebnis |
| --- | ---: |
| SQLite: 100.000 Dateien, verschiedene Filter/Sortierungen/Serienmodi | p50 44,5 ms; p95 98,9 ms |
| Container: 50.000 Filme + 50.000 Serien-Dateien, je zwei Emby-Tonspuren, beide Aufträge zusammen vorgemerkt | Sync 10,04 s; erfolgreich ohne Swap/OOM |
| HTTP-Filter nach diesem Sync | p95 43,4 ms |
| Dashboard: erste / zwischengespeicherte HTTP-Abfragen | 1.576,6 ms / p95 0,85 ms |
| Filterwerte: erste / zwischengespeicherte HTTP-Abfragen | 561,8 ms / p95 0,92 ms |
| Stichprobenweise App-RSS-Spitze während des Syncs | 207,9 MiB |
| App-RSS nach Abfragen | 34,8 MiB |

Containerumgebung: Docker/OrbStack, Linux/arm64, lokales SSD-Dateisystem, Go 1.27.2, `GOMAXPROCS=2`, `GOMEMLIMIT=192MiB`, hartes RAM- und RAM-plus-Swap-Limit jeweils 256 MiB. Mock-Dienste liefen auf dem Host; alle Daten waren synthetisch. Native SQLite-Messung: macOS/arm64, 15 gemeldete logische CPUs, dieselben Go-Limits. Messungen sind keine Prognose für den echten Server.

Der cgroup-Peak lag bei 256 MiB, einschließlich weiterer dem Container zugerechneter Speicherarten. Er ist vom stichprobenweise gemessenen Prozess-RSS zu unterscheiden. Der Test lief ohne OOM und ohne erlaubten Swap, besitzt aber wenig zusätzliche Reserve. Die globale Warteschlange behebt den geprüften Überlappungsfall. Snapshots werden weiterhin im Arbeitsspeicher aufgebaut; für deutlich größere beziehungsweise metadatenreichere Bibliotheken bleibt SQLite-Staging in Batches eine mögliche weitere Ausbaustufe. Keine unbegrenzte Speicherzusage.

Reproduzierbar nach lokalem Build: `python3 docs/scripts/containercheck.py --image arrscope:local`. Das Skript legt ausschließlich eigene temporäre Mock-Dienste, einen Testcontainer und einen synthetischen Datenbestand an und entfernt den Testcontainer anschließend.

## Sicherheitsprüfung und Auslieferung

`pnpm audit`: keine bekannte Schwachstelle. `govulncheck`: keine erreichbare bekannte Schwachstelle und keine in importierten Paketen; ein Modulhinweis betrifft weiterhin das nicht verwendete OpenPGP-Paket. Trivy 0.75.0 meldet im geprüften Laufzeitimage keine hohen oder kritischen bekannten CVEs für Alpine 3.23.6 oder die Go-Binary. Das ist ein Scan-Ergebnis, keine vollständige Sicherheitszertifizierung.

Die Vorschau unter `http://127.0.0.1:8080/` wurde aktualisiert und ist gesund. Schema 4 ist mit vorheriger Sicherung eingespielt; bestehende Einstellungen und Presets sind erhalten. Der Quellcode und das ZIP wurden aktualisiert. Echte authentifizierte Radarr-/Sonarr-/Emby-Daten, VLAN-Erreichbarkeit, HTTPS-Proxy, Langzeitbetrieb und Update/Restore auf Unraid bleiben Teil der anschließenden Einrichtung.

Screenshots: [Dashboard](screenshots/review-dashboard.png), [Metadaten mobil](screenshots/review-mobile-metadata.png), [Dunkle Metadaten mobil](screenshots/review-mobile-metadata-dark.png), [Wartung mobil](screenshots/review-settings-mobile.png).
