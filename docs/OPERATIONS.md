# Betrieb, Sicherung und Rollback

## Dateien und Rechte

`ARRSCOPE_DATA_DIR` ist standardmäßig `/data`. `arrscope.db` und `secret.key` werden mit Modus 0600 angelegt; Verzeichnisse mit 0700, soweit neu erstellt. Ein bestehendes Mount-Verzeichnis muss für die Container-UID/GID zugänglich sein. Niemals den Schlüssel ersetzen: Ohne passenden `secret.key` lassen sich gespeicherte API-Keys nicht entschlüsseln.

Die Container-Konfiguration setzt `GOMAXPROCS=2` für zwei parallele Go-Ausführungsthreads.

`GOMEMLIMIT=192MiB` ist ein weiches Go-Limit; Docker begrenzt separat auf 256 MiB und deaktiviert Swap durch ein gleich hohes RAM-plus-Swap-Limit. Große reale API-Antworten und besonders umfangreiche Metadaten können mehr Speicher benötigen. Speicherverbrauch auf dem Zielserver beobachten, bevor man Grenzen enger setzt.

Das flüchtige `/tmp`-tmpfs bleibt auf 32 MiB begrenzt. Das Image setzt `SQLITE_TMPDIR=/data`, damit größere SQLite-Sortierungen lokale temporäre Dateien auf dem Appdata-Datenträger anlegen können. Der CSV-Export nutzt ebenfalls Appdata für temporäre Dateien. Genügend freien lokalen Speicher vorhalten; keine Netzfreigabe. Das vermeidet Fehler großer Dashboard-Abfragen bei schreibgeschütztem Root-Dateisystem und kleinem tmpfs.

`ARRSCOPE_LISTEN` ist standardmäßig `:8080`; produktive Templates nutzen diesen Port. Der Container-Healthcheck prüft `127.0.0.1:8080/healthz`. `ARRSCOPE_HTTPS=true` verlangt HTTPS für Browser-Sitzungscookies. `ARRSCOPE_DEMO=true` ist ausschließlich für eigene Testdaten vorgesehen.

## Backup

Einstellungen → **Wartung → Cache sichern** erzeugt mittels SQLite `VACUUM INTO` eine konsistente Kopie in `/data/backups/`. Die letzten drei Kopien bleiben erhalten. Vor Schemaänderungen geschieht dies automatisch. Für eine vollständige Sicherung:

1. Cache sichern.
2. ArrScope stoppen.
3. Gesamten Appdata-Ordner einschließlich `secret.key`, Datenbank, WAL/SHM und Backups in das eigene Backup-System aufnehmen.
4. ArrScope starten.

Eine Datenbanksicherung ohne den Schlüssel reicht für eine vollständige Wiederherstellung nicht aus. API-Keys in der Datenbank sind verschlüsselt, weitere Bibliotheks- und Kontodaten liegen lokal in SQLite.

## Wiederherstellung / Rollback

1. ArrScope stoppen, aktuellen Appdata-Ordner vollständig separat sichern.
2. In Unraid Repository auf das gewünschte Versions-Image wie `ghcr.io/buffer379x/arrscope:0.1.0` setzen.
3. Einen vollständigen passenden Appdata-Stand wiederherstellen. Alternativ eine konsistente Datenbankkopie der passenden Schemaversion als `arrscope.db` kopieren und den dazugehörigen `secret.key` behalten. Veraltete WAL-/SHM-Dateien aus dem Zielverzeichnis entfernen, während ArrScope gestoppt ist, nachdem sie separat gesichert wurden.
4. UID/GID und Zugriffsrechte prüfen; Container starten. Anmeldung, Einstellungen und Cache prüfen.

Ältere App-Versionen verweigern eine neuere Datenbankversion. Ein bloßer Wechsel des Images kann nach Migrationen daher nicht genügen. Bei API-Änderungen der Dienste gelten weiterhin deren Kompatibilitätsgrenzen.

## Passwort vergessen

In der Container-Konsole das neue Passwort über stdin an `arrscope reset-password` übergeben. Es muss 12 bis 72 UTF-8-Bytes lang sein. Verwende die verdeckte Terminaleingabe, statt das Passwort als Befehlsargument oder in Chat-Nachrichten zu setzen:

```sh
stty -echo
IFS= read -r arrscope_password
stty echo
printf '\n'
printf '%s\n' "$arrscope_password" | arrscope reset-password
unset arrscope_password
```

Das Konto bleibt bestehen, alle bisherigen Sitzungen werden beendet. Ein neues Passwort wird ausschließlich vom Betreiber eingegeben.

## Fehler und Abbruch

Ein API-Key-Fehler, Zeitlimit, ungültige Antwort oder Datenbankfehler ersetzt den vorhandenen Cache nicht. Status und Zeitpunkt des letzten erfolgreichen Syncs stehen in der Oberfläche. Alle Dienst-Syncs werden global nacheinander verarbeitet; wartende Aufträge erscheinen als vorgemerkt. Geänderte Einstellungen verwerfen Ergebnisse bereits laufender alter Aufträge. Beenden per SIGTERM bricht HTTP-Anfragen ab und wartet auf deren Abschluss; ein laufender SQLite-Commit ist atomar.


## Verbindungen und optionale Pfadzuordnung

Ein leerer API-Key bedeutet weiterhin „gespeicherten Schlüssel beibehalten“. Zum Entfernen die Option **Gespeicherten API-Key beim Speichern löschen** wählen und speichern; die Verbindung wird dabei deaktiviert. Für Emby können absolute Containerpfade zeilenweise zugeordnet werden, z. B. `/movies => /library/movies` und `/tv => /library/tv`. Die Zuordnung ergänzt vorhandene API-Daten und liest keine Medien. Fehlt die passende Zuordnung oder unterscheiden sich weitere Unterordner, bleibt die Emby-Ergänzung aus. Bei Änderung wird für aktive Arr-Dienste ein neuer Sync vorgemerkt.

Schemaversion 4 ergänzt die Vollständigkeitsmerkmale und repariert Sprach-/HDR-Angaben aus dem bestehenden Cache. Vor dem Upgrade wird gesichert; Emby-Tonspuren, Einstellungen, Presets und Farben bleiben erhalten. Ein Rollback auf die vorherige App benötigt die Datenbanksicherung vor der Migration.
