# Architektur und HTTP-API

Go bedient die eingebettete Preact-Oberfläche und die JSON-API. SQLite über `modernc.org/sqlite` benötigt kein CGO. `/data` ist die einzige persistente Abbildung. Frontend und Backend laufen unter derselben Origin; keine externen Schriften, Telemetrie oder CDN-Abhängigkeiten.

## Datenmodell

`files` enthält Dienst-/Datei-ID, Medien-ID, Serien-/Staffelzuordnung, normalisierte Filterspalten und den vollständigen Dateieintrag als JSON. `years` ermöglicht mehrere Episodenjahre pro Datei. `search` ist ein FTS5-Index mit Titel, Serientitel und Dateiname. `kv` enthält Einstellungen, Konto und Sync-Status; `presets` und `sessions` sind eigenständige Tabellen. Schemaversionen werden über `PRAGMA user_version` geführt. Vor bestehenden Migrationen entsteht eine konsistente Sicherung, neuere unbekannte Versionen werden abgewiesen.

Für jedes angezeigte Merkmal enthält `evidence` unabhängig normalisierte Dateiname-, Arr-API- und optionale Emby-Werte, verwendete Herkunft (`filename`, `api`, `emby`, `combined`, `unknown`) und Konflikt. Der vollständige ursprüngliche Dateiname bleibt erhalten. Filmjahr und Titel stammen aus Radarr; Episodentitel, Staffel und Erstausstrahlungsjahre aus Sonarr. Nicht jede ursprüngliche API-Antwort wird vollständig archiviert.

Die Qualität des Dienstes ergänzt Auflösung und Quelle. `mediaInfo` ergänzt bestehende Video-/Audioangaben. Pixelabmessungen alleine ersetzen keine Qualitätsklasse; ohne passenden Qualitätswert oder ausdrücklichen Auflösungstoken bleibt die Klasse unbekannt. Die API liefert ihre vorhandenen MediaInfo-Daten; ArrScope ruft keine MediaInfo-Analyse auf. [Radarr-Ressource](https://github.com/Radarr/Radarr/blob/develop/src/Radarr.Api.V3/MovieFiles/MediaInfoResource.cs), [Sonarr-Ressource](https://github.com/Sonarr/Sonarr/blob/develop/src/Sonarr.Api.V3/EpisodeFiles/MediaInfoResource.cs).

Explizite Sprachen werden als getrennte Filterwerte angeboten. Eine Datei mit Deutsch und Englisch passt auf beide Sprachfilter. Kombinierte HDR-Angaben können über ihre Komponenten wie Dolby Vision gefiltert werden. `x264` und `AVC` gehören zur selben Familie, ein Unterschied zwischen diesen beiden allein wird nicht als Widerspruch angezeigt.

## Synchronisation

Radarr-Katalogabfrage, danach Dateiabfragen in 100er-Blöcken über Film-IDs. Sonarr-Katalog, anschließend Episoden und Episodendateien je Serie. Eine Mehrfachepisodendatei wird über ihre Datei-ID einmal dargestellt. Eine globale Warteschlange verarbeitet höchstens einen Dienst-Snapshot gleichzeitig. Sonarr lädt Episoden und Dateien einer Serie mit höchstens zwei parallelen GET-Anfragen; Serien werden nacheinander verarbeitet. Der Status unterscheidet `running` und `queued`; wartende und laufende Aufträge sind abbrechbar. Zeitlimit je Anfrage: 20 Sekunden, höchstens drei Versuche bei Netzwerk-/5xx-Fehlern. Redirects werden abgewiesen, damit Keys nicht an andere Ziele weitergegeben werden.

Neue Dienst-Daten werden im Arbeitsspeicher gesammelt. Erst nach vollständig erfolgreichem Abruf ersetzt eine SQLite-Transaktion den Dienst-Cache, Suchindex und Jahreszuordnungen. Andere Leser sehen über WAL einen vollständigen Stand. Die paginierte Abfrage hält Zähler und Daten in einer gemeinsamen Lesetransaktion. Einstellungen und Sync-Dispatch sind zusammen gesperrt, damit veraltete Konfigurationen keine neuen Aufträge starten. Alte Aufträge werden bei Änderungen abgebrochen und durch Generationen vom Commit ausgeschlossen.

Es gibt keine Garantie eines gemeinsamen Snapshots über alle HTTP-Aufrufe des upstream Dienstes. Aus der API erkannte Zuordnungsfehler erhalten den bisherigen Stand; Dateien ohne bekannte Episodenzuordnung zeigen ihren Dateinamen und unbekanntes Jahr.

## Interne Endpunkte

| Methode / Pfad | Funktion |
| --- | --- |
| GET `/healthz` | Prozesszustand und Version, ohne Anmeldung |
| GET `/api/status` | Einrichtung / Anmeldung; angemeldet zusätzlich CSRF und Sync-Status |
| POST `/api/setup` | Einmaliges Konto mit Einrichtungscode |
| POST `/api/login`, `/api/logout` | Sitzung erstellen / beenden |
| GET / PUT `/api/settings` | Maskierte Einstellungen / verschlüsselt speichern |
| POST `/api/test` | Verbindungstest über lesenden Systemstatus |
| POST `/api/sync` | Asynchroner Auftrag für `service` |
| GET `/api/files` | Trefferseite / Staffelkontext |
| GET `/api/statistics` | Bestandszahlen und Merkmalsverteilungen aus einem Cache-Snapshot |
| GET `/api/facets` | Verfügbare Merkmalswerte je Dienst |
| GET / POST / DELETE `/api/presets` | Presets lesen / speichern / per ID löschen |
| GET `/api/export` | Alle tatsächlichen Treffer als CSV |
| PUT `/api/password` | Passwort ändern, Sitzungen beenden |
| POST `/api/backup` | Konsistente lokale Datenbanksicherung |

Außer Einrichtung, Login und öffentlichem Status benötigen APIs eine Sitzung. Mutationen benötigen `X-CSRF-Token`; fremde Origin-Header werden abgewiesen. Cookies: HttpOnly, SameSite Strict, 24 Stunden, bei TLS/HTTPS-Konfiguration Secure. API-Key-Antworten sind immer leer, `hasKey` zeigt nur das Vorhandensein an. Fehler geben keine upstream Antworttexte oder Keys aus.

`files` und `export` teilen dieselbe Filterfunktion: `service=radarr|sonarr`, `q`, `yearMin`, `yearMax`, `resolution`, `codec`, `family`, `source`, `processing`, `hdr`, `bitDepth`, `audio`, `channels`, `language`, `sizeMin`, `sizeMax`, `unknownYear=true`, `unknownSize=true`, `conflict=true|false`, `metadata=complete|incomplete`. Mehrfache Parameter eines Merkmals sind ODER, andere Merkmale UND. Für jedes Merkmal ist `<field>Op=eq|not` erlaubt, beispielsweise `sourceOp=not&source=BluRay&source=WEB-DL`. Ohne Operator gilt `eq`. `not` negiert die gesamte Wertemenge, sodass keiner der ausgewählten Werte vorkommen darf. Unbekannte Werte sind intern leere Zeichenfolgen und bleiben erhalten, solange `__unknown__` nicht Teil des Ausschlusses ist. Sprache/HDR/Audio verwenden Komponentenbedingungen. Zusätzlich ist für diese drei Felder `all|notAll` erlaubt: `all` verbindet gewählte Komponenten mit AND, `notAll` negiert diese vollständige AND-Bedingung. Die Bedingungen enthalten ausschließlich parametrische Werte. Für ein einzelwertiges Feld werden diese zwei Operatoren abgewiesen. Ungültige oder mehrfach angegebene Operatoren werden abgewiesen. Ein Operator ohne Werte hat keine Filterwirkung. Leere Werte sind über `__unknown__` wählbar. Größe wird in Bytes übergeben, die Oberfläche verwendet GiB.

Für `files`: `page` ab 1, feste Seitengröße 100; `mode=matches|season` überschreibt die gespeicherte Ansicht. `total` zählt echte Treffer, `displayTotal` zusätzlich sichtbaren Staffelkontext, `matched` markiert echte Treffer. Sonarr sortiert nach Serie, Staffel und Episodennummer. Radarr erlaubt `sort=title|year|resolution|codec|source|size|hdr|bitDepth|audio|channels|language` und `dir=asc|desc`. Standard: Titel aufsteigend; unbekannte Werte bleiben in beiden Richtungen am Ende. Auflösungen, Bit-Tiefe, Kanäle und Größe werden numerisch geordnet. Die Oberfläche speichert Spalte und Richtung im lokalen Browser unabhängig von Filter-Presets. CSV nutzt UTF-8 BOM und Semikolon, verhindert Formelausführung über Präfixe und enthält keine zusätzlichen Kontextdateien.

Textsuche nutzt bereinigte UND-Prefixbegriffe, keine beliebige Teilzeichenfolgensuche und keine Nutzer-FTS-Syntax. Die SQLite-Filter sind parametrisiert; Sortierspalten stammen aus einer festen Liste. Facets beziehen sich auf den gesamten Dienst-Cache, nicht auf den aktuellen Filter.

## Quellen

Lesende Datei-Endpunkte: [Radarr MovieFileController](https://github.com/Radarr/Radarr/blob/develop/src/Radarr.Api.V3/MovieFiles/MovieFileController.cs), [Sonarr EpisodeFileController](https://github.com/Sonarr/Sonarr/blob/develop/src/Sonarr.Api.V3/EpisodeFiles/EpisodeFileController.cs). [FTS5](https://www.sqlite.org/fts5.html) und [WAL / lokale Dateisysteme](https://www.sqlite.org/wal.html).

## Statistiken

`GET /api/statistics` zählt eindeutige Film- und Serien-IDs mit vorhandenen Cache-Dateien. Episoden werden über Serie, Staffel und Episodennummer dedupliziert, einschließlich Specials. Jede Episode einer Mehrfachepisodendatei zählt einzeln. Die Verteilungen zählen Dateien nach Auflösung, Codec und Source (Verarbeitungsart wie Remux bleibt davon getrennt), jeweils für Gesamtbestand, Radarr und Sonarr. Unbekannte Werte sind enthalten. Die Aggregation läuft vollständig in SQLite innerhalb einer Lesetransaktion; Listenfilter werden nicht übernommen und keine Dienste abgefragt. Die Statistik aktualisiert sich nach erfolgreichem Sync.

## Darstellungseinstellungen

`settings.visibleColumns` speichert je Dienst (`radarr`, `sonarr`) eine Liste sichtbarer Spalten in `kv`. Fehlende Einstellungen älterer Installationen erhalten die fünf ursprünglichen Spalten. Titel/Episode ist verpflichtend; unbekannte Dienste, doppelte oder unbekannte Spalten werden beim Speichern abgewiesen. `settings.columnOrder` speichert unabhängig von der Sichtbarkeit die vollständige Reihenfolge aller elf Spalten je Dienst. Fehlende Einstellungen erhalten die bisherige Reihenfolge; unvollständige Listen, Duplikate und unbekannte Spalten oder Dienste werden abgewiesen. Tabellenkopf und Zellen werden aus derselben geordneten Liste aufgebaut. Der temporäre Schalter „Alle Spalten“ filtert diese Liste nicht nach Sichtbarkeit und verändert die gespeicherten Defaults nicht. Seriengruppen verwenden die tatsächliche sichtbare Spaltenzahl. Filter, Sortierung und CSV bleiben unabhängig von der Sichtbarkeit.

Die Oberfläche öffnet nach dem Login das Dashboard. Der Filterblock besitzt nur die Zustände vollständig sichtbar und minimiert; im minimierten Zustand bleiben Presets und aktive Filter zugänglich.

## Optionale Emby-Ergänzung

`settings.emby` speichert Aktivierung, URL und verschlüsselten API-Key; `interval` ist stets 0, weil Emby zum Dienst-Sync gehört. Aktivierung, URL- oder Key-Änderung invalidiert laufende Snapshots und synchronisiert aktivierte Arr-Dienste neu. Anzeigeeinstellungen invalidieren keine Verbindungen. Demoverbindungen bleiben gesperrt.

Der GET-Client teilt Timeout (20 Sekunden), höchstens drei Versuche, Redirect-Verbot und bereinigte Fehler mit dem Arr-Client, verwendet jedoch `X-Emby-Token` und eine eigene Whitelist für `System/Info` und `Items`. Eine URL mit Unterpfad wird unterstützt; `/emby` wird nur bei Bedarf angehängt. Film- und Episodenmetadaten werden getrennt in Seiten mit höchstens 250 Einträgen gelesen. Seitenanzahl, Totals und doppelte IDs werden geprüft. Nur benötigte Dateikandidaten verbleiben im Speicher.

Identität: genauer Basisdateiname + positive Byte-Größe, eindeutig auf beiden Seiten. Mehrfachversionen verwenden die passenden MediaSources. Item-Streams werden nur für die exakt gleiche Source-Path eingesetzt, nie für eine andere Version. Fehlende/mehrdeutige Zuordnung bleibt `embyMatch`; `audioTracks` enthält einzelne interne Tonspuren mit Index, Sprache, Codec, Kanälen, Layout, Titel und Standardflag. Alle API-Daten werden als Daten behandelt. Keine Sprache wird aus einem Tonspurtitel geschätzt. Subtitle- und externe Audio-Streams werden ausgeschlossen.

Eindeutige, vorhandene Emby-Tonspurangaben bestimmen die Audio-/Sprachmengen für Filter und Anzeige; wenn alle Spuren eines Merkmals unbekannt sind, bleiben vorhandene Dateiname-/Arr-Angaben erhalten, ohne sie einzelnen Spuren zuzuordnen; Herkunft bleibt in Evidence getrennt. Die Hauptspalte Kanäle behält die Dateiname-/Arr-Angabe, während der Dialog pro Tonspur die echte Kanalanzahl beziehungsweise explizites Layout anzeigt. Explizite Emby-HDR-Typen/-Subtypen ergänzen die vorhandenen HDR-Tags; DV-Profilnummern erzeugen keinen angenommenen Fallback. Ein Emby-Fehler bricht den ergänzten Dienst-Snapshot ab, sodass keine partiellen Daten übernommen werden. Es gibt keinen separaten Medienkatalog, keinen Playback-Aufruf und keine Medienanalyse.

Migration auf Schemaversion 4 repariert zwischengespeicherte Audio-/HDR-/Sprachmerkmale aus Dateinamen und bereits gespeicherter API-Evidence, atomar in 500er-Batches mit vorheriger Sicherung. Einstellungen, Presets und Dateizugehörigkeit bleiben erhalten. Die optionalen Emby-Felder leben im bestehenden JSON, ohne weitere SQL-Spalten. Bei einem erfolgreichen neuen Sync stammen sie aus der aktuellen Emby-Antwort.


## Korrekturen der Abschlussprüfung

Die JSON-Decodierung verlangt ein nicht-null Objekt beziehungsweise eine Liste, prüft die vollständige Antwort bis EOF und begrenzt die Größe auf 64 MiB. Katalog-/Datei-IDs, Titel, Zuordnungen und Duplikate werden vor der Cache-Übernahme geprüft. Leere gültige Listen bleiben zulässig; ungültige Antworten erhalten den letzten vollständigen Stand.

Sprachen werden als normalisierte, sortierte Menge behandelt. Zusätzliche ausdrücklich gemeldete Arr-Sprachen ergänzen Dateinamen-Sprachen, ohne einer Tonspur einen geratenen Codec zuzuweisen. `unknownTraits` kennzeichnet Audio-/Sprachmerkmale, wenn mindestens eine ergänzte Tonspur dafür keine Angabe besitzt. `__unknown__` findet dadurch sowohl vollständig als auch teilweise unbekannte Audio-/Sprachangaben. `metadataIncomplete` bewertet Auflösung, Codec, Source, Audio, Sprache sowie erfolglose konfigurierte Emby-Zuordnungen. Fehlendes HDR kann SDR oder unbekannt bedeuten und setzt diesen Status alleine nicht. Ein negativer Filter beschreibt fehlende **erkannte Angaben**, keine bestätigte Abwesenheit in der Datei.

`settings.emby.pathMappings` enthält optional Zeilen `Arr-Pfad => Emby-Pfad`. Ist eine Zuordnung eingerichtet, verlangt Emby zusätzlich zum Dateinamen und der Byte-Größe den exakt passenden vollständigen Pfad; die längste passende Arr-Präfixzuordnung gewinnt. Ohne passende Zuordnung wird nichts ergänzt. Pfade kommen ausschließlich aus API-Metadaten; keine Dateisystemzugriffe. Der Arr-Quellpfad bleibt nur während des Syncs im Speicher und wird nicht an den Browser ausgegeben.

Facets und Dashboard-Aggregate sind unveränderliche, defensiv kopierte In-Memory-Snapshots und werden nach einem erfolgreichen Dienst-Commit invalidiert. Ihre SQL-Abfragen laufen in Lesetransaktionen. HTTP-Kontexte werden bis zu SQLite durchgereicht. Browser-Anfragen erhalten Abbruchcontroller und Schutz gegen verspätete Antworten; Filterwerte werden nur in einer Bibliotheksansicht geladen. Das Status-Polling pausiert bei ausgeblendeten Browser-Tabs; Intervalle: aktiver Sync 3 Sekunden, angemeldeter Leerlauf 30 Sekunden, Anmeldung 60 Sekunden.

CSV verwendet dieselbe Sortierfunktion und Filterlogik wie die Liste. Der gesamte Export wird zunächst in einer geschützten temporären Datei unter Appdata erstellt. SQL-, JSON- und Schreibfehler liefern vor dem Download einen Fehler. Erfolgreiche/abgebrochene Exporte entfernen die Datei. `clearKey` ist eine ausdrückliche Löschoption für deaktivierte Verbindungen; ein leeres Eingabefeld alleine behält weiterhin den gespeicherten Key.

Passwortänderung und Sitzungswiderruf erfolgen in einer Datenbanktransaktion. Sitzungsanlage und bestätigte Passwortänderungen prüfen zusätzlich den zuvor verifizierten Passwort-Hash in ihrem SQL-Schreibvorgang. Eine verspätete Anmeldung mit alten Zugangsdaten erzeugt dadurch auch bei einem parallel ausgeführten CLI-Passwortreset keine neue gültige Sitzung.

Das Frontend ist in Bibliothek, Filter, Dashboard, Einstellungen, Metadaten-Dialog, gemeinsame Darstellung und Request-Hooks aufgeteilt. Styles liegen in getrennten Modulen. Einstellungen besitzen die Bereiche Verbindungen, Darstellung und Wartung mit dauerhaft erreichbarer Speichern-Leiste. Auf schmalen Bildschirmen zeigt der Metadatenvergleich Eigenschaftskarten statt kleiner Tabellenbeschriftungen.
