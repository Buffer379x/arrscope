# Unraid

ArrScope benötigt ein öffentliches Release-Image. Es wird über das mitgelieferte Template verwaltet, anschließend nutzt Unraid seine normalen Update-Funktionen.

1. `deploy/arrscope.xml` als `my-arrscope.xml` in `/boot/config/plugins/dockerMan/templates-user/` ablegen. Falls du ein persönliches Template erhalten hast, dieses verwenden.
2. Docker → Add Container → Template `arrscope` auswählen.
3. Repository `ghcr.io/buffer379x/arrscope:stable`, Privileged aus, Shell `sh`. Für Bridge einen freien Host-Port angeben. Für ein bereits eingerichtetes Docker-VLAN dessen Custom-Netz und eine reservierte feste IP wählen; Port 8080 ist dann direkt auf dieser IP erreichbar.
4. `/data` auf einen lokalen SSD-/NVMe-Pool-Pfad wie `/mnt/cache/appdata/arrscope` abbilden. Passe den tatsächlichen Poolnamen an. Kein Medienverzeichnis hinzufügen. Vor dem ersten Start das neue ArrScope-Verzeichnis mit `mkdir -p /mnt/cache/appdata/arrscope`, `chown 99:100 /mnt/cache/appdata/arrscope` und `chmod 700 /mnt/cache/appdata/arrscope` vorbereiten (Pool-Pfad und UID/GID anpassen).
5. Extra Parameters enthalten `--user=99:100 --read-only --tmpfs=/tmp:rw,noexec,nosuid,nodev,size=32m --cap-drop=ALL --security-opt=no-new-privileges --memory=256m`. Bei anderer UID/GID diese ändern und den Appdata-Besitz entsprechend setzen. `PUID`/`PGID`-Variablen werden nicht verwendet, da kein Root-Entrypoint startet.
6. Apply. Nach erfolgreichem Start WebUI öffnen. In der Container-Konsole `cat /data/setup-token` ausführen; Code und neues Passwort selbst in der Einrichtung eingeben.
7. Radarr/Sonarr-Adressen und API-Keys in den App-Einstellungen eingeben, testen, aktivieren und speichern. API-Keys stehen im jeweiligen Dienst unter Einstellungen → Allgemein. Die erste vollständige Synchronisation abwarten.

Aktiviere Autostart für ArrScope. Updates: Docker → Check for Updates → ArrScope → Update. Unraid zieht den freigegebenen `stable`-Stand und verwendet das bestehende `/data`. Die App erstellt bei notwendigen Migrationen eine Datenbankkopie.

Bei Nutzung einer eigenen IP ist keine Host-Portfreigabe nötig. ArrScope muss beide Dienste erreichen können; VLAN-/Firewallregeln werden vom Betreiber festgelegt. Ohne HTTPS bleibt `ARRSCOPE_HTTPS=false`; bei einem bestehenden HTTPS-Proxy auf `true` umstellen. Kein öffentlicher Internetzugriff wird durch das Template eingerichtet.

Falls das Image mit einer Zugriffsmeldung nicht geladen werden kann: Die GHCR-Paket-Sichtbarkeit muss **Public** sein. Ein öffentliches GitHub-Repository allein reicht dafür nicht.
