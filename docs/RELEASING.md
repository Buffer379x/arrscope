# Veröffentlichung

Geplant ist das öffentliche Repository `Buffer379x/arrscope` mit MIT-Lizenz und das öffentliche Paket `ghcr.io/buffer379x/arrscope`. Dieses Dokument beschreibt den vorbereiteten Ablauf; die lokale Quellcode-Auslieferung allein veröffentlicht noch nichts.

## Erstmalig

1. Nach Abnahme Repository mit Standardbranch `main` anlegen und den geprüften Quellcode pushen. Keine lokalen Appdata-/Testverzeichnisse, Tokens oder persönlichen Installationsprofile hochladen.
2. GitHub Actions aktivieren und den Workflow **Checks** abwarten. Er baut die Oberfläche, prüft Go mit Race-Detector und vet, misst Filter und startet einen Container einschließlich Neustartprüfung. Zusätzlich laufen Frontend-Kontrasttests, pnpm audit, govulncheck, ein Scan des Laufzeitimages auf hohe/kritische CVEs und der 100.000-Dateien-Container-Test ohne Swap.
3. Bei Fehlern korrigieren und Checks erneut abwarten. Kein Release bei roten Checks.
4. **Approved release** auf `main` manuell mit Version `0.1.0` starten. Der Workflow prüft Paket-Abhängigkeiten und Tests erneut, baut und pusht die Versions-Images, testet beide Architekturen, scannt die Images und prüft den großen synthetischen Bestand ohne Swap. Anschließend erstellt er Tag/Release; erst danach wird `stable` auf denselben Digest gesetzt.
5. GHCR → Paket `arrscope` → Package settings → Visibility → **Public**. Die Repository-Sichtbarkeit alleine macht das Paket nicht öffentlich. Dies einmalig mit dem Betreiber durchführen.
6. Einen anonymen Pull überprüfen und das Image in Unraid installieren. PNG und XML werden aus dem öffentlichen Repository geladen.

Keine PATs sind im Repository erforderlich: der Workflow verwendet das kurzlebige `GITHUB_TOKEN` mit `contents:write` und `packages:write`. Drittanbieter-Actions sind auf Commit-SHAs fixiert, Dependabot schlägt monatliche Updates vor. [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

## Weitere Versionen

Änderungen und Tests zunächst auf einem Branch/PR, danach geprüfter Stand auf `main`. Erst nach Abnahme neuen manuellen Release starten, z. B. `0.1.1`. Bestehende veröffentlichte Git-Tags werden abgewiesen. `stable` bewegt sich nur im Release-Workflow nach bestandenen Prüfungen. Normale Pushes veröffentlichen kein Update.

Release-Notizen werden aus GitHub-Änderungen generiert. Der erste Release beschreibt eine neue Anwendung; bei späteren Migrationen ergänzend Änderungen und Rollback-Hinweise in der Release-Beschreibung angeben.

Unraid zieht `stable`, sobald der Betreiber ein Update anstößt. Compose: `docker compose pull && docker compose up -d`. Kein Hintergrund-Pull durch ArrScope. Falls der Workflow nach Veröffentlichung der Versions-Images vor Promotion scheitert, bleibt `stable` unverändert; Ursache prüfen, bevor ein erneuter Lauf gestartet wird.
