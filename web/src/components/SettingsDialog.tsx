import { Modal } from "./Modal";

import {
  ArrowDown,
  ArrowUp,
  ArrowUpRight,
  Check,
  Database,
  Film,
  Layers,
  LoaderCircle,
  LockKeyhole,
  Tv,
} from "lucide-preact";
import { useEffect, useState } from "preact/hooks";
import type { Connection, Service, Settings } from "../types";

import { api } from "../api";
import {
  colorFields,
  columnLabels,
  ConnectionService,
  defaultColor,
  defaultColumnOrder,
  defaultColumns,
  labels,
  tableColumns,
  Tag,
} from "../ui";
export function SettingsDialog({
  settings,
  demo,
  close,
  saved,
}: {
  settings: Settings;
  demo: boolean;
  close: () => void;
  saved: () => Promise<void>;
}) {
  const [cfg, setCfg] = useState<Settings>(structuredClone(settings)),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [test, setTest] = useState<Record<string, string>>({}),
    [current, setCurrent] = useState(""),
    [password, setPassword] = useState(""),
    [section, setSection] = useState("connections"),
    [colorField, setColorField] = useState("resolution"),
    [recognized, setRecognized] = useState<Record<string, string[]>>({});
  useEffect(() => {
    Promise.all([
      api<Record<string, string[]>>("facets?service=radarr"),
      api<Record<string, string[]>>("facets?service=sonarr"),
    ])
      .then((all) => {
        const values: Record<string, string[]> = {};
        colorFields.forEach((field) => {
          values[field] = [
            ...new Set(all.flatMap((f) => f[field] || [])),
          ].sort();
        });
        setRecognized(values);
      })
      .catch((e) => setError(e.message));
  }, []);
  const colorValues = [
    ...new Set([
      ...(recognized[colorField] || []),
      ...Object.keys(cfg.tagColors || {})
        .filter((k) => k.startsWith(colorField + ":"))
        .map((k) => k.slice(colorField.length + 1)),
    ]),
  ].sort();
  const setColor = (key: string, color?: string) => {
    setCfg((current) => {
      const colors = { ...current.tagColors };
      if (color) colors[key] = color;
      else delete colors[key];
      return { ...current, tagColors: colors };
    });
  };
  const update = (service: ConnectionService, patch: Partial<Connection>) =>
    setCfg((current) => ({
      ...current,
      [service]: { ...current[service], ...patch },
    }));
  const submit = async (e: Event) => {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      await api("settings", "PUT", cfg);
      await saved();
      close();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="Einstellungen" close={close} wide>
      <nav class="settings-tabs" aria-label="Einstellungsbereiche">
        {[
          ["connections", "Verbindungen"],
          ["appearance", "Darstellung"],
          ["maintenance", "Wartung"],
        ].map(([value, label]) => (
          <button
            type="button"
            class={section === value ? "active" : ""}
            aria-pressed={section === value}
            onClick={() => setSection(value)}
          >
            {label}
          </button>
        ))}
      </nav>
      <form onSubmit={submit}>
        <p class="quiet">
          Deine Dienste, dein Cache. Medienverzeichnisse werden nicht benötigt.
        </p>
        {demo && (
          <div class="notice">
            Demo-Verbindungen sind gesperrt. Serienansicht, Standardspalten und
            Tag-Farben kannst du ausprobieren.
          </div>
        )}
        <div class="settings-section" hidden={section !== "connections"}>
          {(["radarr", "sonarr", "emby"] as ConnectionService[]).map(
            (service) => (
              <fieldset class="connection">
                <legend>
                  {service === "radarr" ? <Film size={16} /> : <Tv size={16} />}{" "}
                  {service === "radarr"
                    ? "Radarr"
                    : service === "sonarr"
                      ? "Sonarr"
                      : "Emby · optional"}
                </legend>
                {service === "emby" && (
                  <p class="quiet">
                    Ergänzt bereits vorhandene Tonspurmetadaten. Keine Scans
                    oder Medienzugriffe. Zuordnung nur bei eindeutigem
                    Dateinamen und identischer Dateigröße.
                  </p>
                )}
                <label class="check-field">
                  <input
                    type="checkbox"
                    disabled={demo}
                    checked={cfg[service].enabled}
                    onChange={(e) =>
                      update(service, { enabled: e.currentTarget.checked })
                    }
                  />{" "}
                  Dienst aktivieren
                </label>
                <div class="connection-grid">
                  <label>
                    Serveradresse
                    <input
                      type="url"
                      disabled={demo}
                      placeholder={
                        service === "radarr"
                          ? "http://192.168.1.10:7878"
                          : service === "sonarr"
                            ? "http://192.168.1.10:8989"
                            : "http://192.168.1.10:8096"
                      }
                      value={cfg[service].url}
                      onInput={(e) =>
                        update(service, { url: e.currentTarget.value })
                      }
                    />
                    <small>
                      Auch mit URL-Unterpfad, zum Beispiel /
                      {service === "emby" ? "emby" : service}.
                    </small>
                  </label>
                  <label>
                    API-Key
                    <input
                      type="password"
                      disabled={demo}
                      autoComplete="off"
                      placeholder={
                        cfg[service].hasKey
                          ? "Gespeichert · leer lassen zum Beibehalten"
                          : "API-Key eingeben"
                      }
                      value={cfg[service].apiKey || ""}
                      onInput={(e) =>
                        update(service, { apiKey: e.currentTarget.value })
                      }
                    />
                    <small>
                      {service === "emby"
                        ? "Im Emby-Dashboard unter API-Schlüssel. Einen eigenen Schlüssel für ArrScope verwenden."
                        : "Im jeweiligen Dienst unter Einstellungen → Allgemein."}
                    </small>
                  </label>
                  <label class="check-field">
                    <input
                      type="checkbox"
                      disabled={demo}
                      checked={!!cfg[service].clearKey}
                      onChange={(e) =>
                        update(service, {
                          clearKey: e.currentTarget.checked,
                          ...(e.currentTarget.checked
                            ? { enabled: false, apiKey: "" }
                            : {}),
                        })
                      }
                    />
                    Gespeicherten API-Key beim Speichern löschen
                  </label>
                  {service === "emby" && (
                    <label class="path-mapping-field">
                      Optionale Pfadzuordnungen
                      <textarea
                        aria-label="Emby-Pfadzuordnungen"
                        disabled={demo}
                        rows={3}
                        value={cfg.emby.pathMappings || ""}
                        placeholder="/movies => /media/movies"
                        onInput={(e) =>
                          update("emby", {
                            pathMappings: e.currentTarget.value,
                          })
                        }
                      />
                      <small>
                        Eine Zeile je Medienpfad: Arr-Pfad =&gt; Emby-Pfad. Wenn
                        gesetzt, muss zusätzlich der zugeordnete vollständige
                        Pfad übereinstimmen. Unterschiedliche Ordnernamen
                        innerhalb der Bibliothek verhindern dann die Ergänzung.
                      </small>
                    </label>
                  )}
                  {service !== "emby" ? (
                    <label>
                      Synchronisieren
                      <select
                        disabled={demo}
                        value={cfg[service].interval}
                        onChange={(e) =>
                          update(service, {
                            interval: Number(e.currentTarget.value),
                          })
                        }
                      >
                        {[
                          [0, "Nur manuell"],
                          [15, "Alle 15 Minuten"],
                          [30, "Alle 30 Minuten"],
                          [60, "Stündlich"],
                          [360, "Alle 6 Stunden"],
                          [1440, "Täglich"],
                        ].map(([value, label]) => (
                          <option value={value}>{label}</option>
                        ))}
                      </select>
                    </label>
                  ) : (
                    <p class="quiet">
                      Wird zusammen mit Radarr und Sonarr synchronisiert. Ein
                      Fehler behält den letzten vollständigen Cache-Stand.
                    </p>
                  )}
                  <div class="connection-test">
                    <button
                      type="button"
                      class="button"
                      disabled={demo || test[service] === "Prüfe …"}
                      onClick={async () => {
                        setTest({ ...test, [service]: "Prüfe …" });
                        try {
                          const r = await api<{ version: string }>(
                            "test",
                            "POST",
                            {
                              service,
                              url: cfg[service].url,
                              apiKey: cfg[service].apiKey || "",
                            },
                          );
                          setTest({
                            ...test,
                            [service]: "Verbunden · Version " + r.version,
                          });
                        } catch (e) {
                          setTest({ ...test, [service]: (e as Error).message });
                        }
                      }}
                    >
                      Verbindung testen <ArrowUpRight size={14} />
                    </button>
                    <small role="status">{test[service]}</small>
                  </div>
                </div>
              </fieldset>
            ),
          )}
        </div>
        <div class="settings-section" hidden={section !== "appearance"}>
          <fieldset class="connection">
            <legend>
              <Layers size={16} /> Serienansicht
            </legend>
            <label>
              Ergebnisse bei aktiven Filtern
              <select
                value={cfg.seasonMode}
                onChange={(e) =>
                  setCfg({
                    ...cfg,
                    seasonMode: e.currentTarget.value as Settings["seasonMode"],
                  })
                }
              >
                <option value="matches">Nur passende Dateien anzeigen</option>
                <option value="season">
                  Ganze passende Staffel · Treffer markieren
                </option>
              </select>
            </label>
            <small>
              Eine passende Datei genügt, um ihre Staffel im Kontext anzuzeigen.
            </small>
          </fieldset>
          <fieldset class="connection">
            <legend>Standardspalten</legend>
            <p class="quiet">
              Wähle die sichtbaren Spalten und verschiebe sie mit den Pfeilen.
              Die Reihenfolge gilt von links nach rechts, auch für ausgeblendete
              Spalten. Beides wird dauerhaft gespeichert. Filter und CSV
              enthalten weiterhin alle Merkmale.
            </p>
            <div class="column-settings">
              {(["radarr", "sonarr"] as Service[]).map((service) => (
                <fieldset>
                  <legend>
                    {service === "radarr"
                      ? "Filme · Radarr"
                      : "Serien · Sonarr"}
                  </legend>
                  <div class="column-options">
                    {cfg.columnOrder[service].map((field, index) => (
                      <div class="column-option" key={field}>
                        <label class="check-field">
                          <input
                            type="checkbox"
                            aria-label={
                              (service === "radarr" ? "Filme: " : "Serien: ") +
                              columnLabels[field]
                            }
                            checked={cfg.visibleColumns[service].includes(
                              field,
                            )}
                            disabled={field === "title"}
                            onChange={(e) => {
                              const checked = e.currentTarget.checked;
                              setCfg((current) => {
                                const selected = new Set(
                                  current.visibleColumns[service],
                                );
                                if (checked) selected.add(field);
                                else selected.delete(field);
                                return {
                                  ...current,
                                  visibleColumns: {
                                    ...current.visibleColumns,
                                    [service]: tableColumns
                                      .map(([key]) => key)
                                      .filter((key) => selected.has(key)),
                                  },
                                };
                              });
                            }}
                          />
                          {field === "title"
                            ? service === "radarr"
                              ? "Titel (immer sichtbar)"
                              : "Episode (immer sichtbar)"
                            : columnLabels[field]}
                        </label>
                        <div class="column-move-tools">
                          {([-1, 1] as const).map((direction) => (
                            <button
                              type="button"
                              class="icon-button"
                              aria-label={
                                (service === "radarr"
                                  ? "Filme: "
                                  : "Serien: ") +
                                columnLabels[field] +
                                (direction === -1
                                  ? " nach oben"
                                  : " nach unten")
                              }
                              title={
                                direction === -1
                                  ? "Spalte nach links verschieben"
                                  : "Spalte nach rechts verschieben"
                              }
                              disabled={
                                direction === -1
                                  ? index === 0
                                  : index ===
                                    cfg.columnOrder[service].length - 1
                              }
                              onClick={() => {
                                setCfg((current) => {
                                  const order = [
                                    ...current.columnOrder[service],
                                  ];
                                  const position = order.indexOf(field);
                                  const next = position + direction;
                                  if (next < 0 || next >= order.length)
                                    return current;
                                  [order[position], order[next]] = [
                                    order[next],
                                    order[position],
                                  ];
                                  return {
                                    ...current,
                                    columnOrder: {
                                      ...current.columnOrder,
                                      [service]: order,
                                    },
                                  };
                                });
                              }}
                            >
                              {direction === -1 ? (
                                <ArrowUp size={14} />
                              ) : (
                                <ArrowDown size={14} />
                              )}
                            </button>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                  <button
                    type="button"
                    class="text-button"
                    onClick={() =>
                      setCfg((current) => ({
                        ...current,
                        visibleColumns: {
                          ...current.visibleColumns,
                          [service]: [...defaultColumns],
                        },
                        columnOrder: {
                          ...current.columnOrder,
                          [service]: [...defaultColumnOrder],
                        },
                      }))
                    }
                  >
                    Standard wiederherstellen
                  </button>
                </fieldset>
              ))}
            </div>
          </fieldset>
          <fieldset class="connection tag-color-settings">
            <legend>Tag-Farben</legend>
            <p class="quiet">
              Eigene Farben gelten für beide Bibliotheken und beide Designs. Sie
              bleiben bis zum Zurücksetzen gespeichert.
            </p>
            <label>
              Tag-Eigenschaft
              <select
                aria-label="Tag-Eigenschaft"
                value={colorField}
                onChange={(e) => setColorField(e.currentTarget.value)}
              >
                {colorFields.map((field) => (
                  <option value={field}>{labels[field]}</option>
                ))}
              </select>
            </label>
            <div class="tag-color-grid">
              {colorValues.map((value) => {
                const key = colorField + ":" + value;
                const label = value === "__unknown__" ? "Unbekannt" : value;
                return (
                  <div class="tag-color-row">
                    <Tag
                      kind={colorField}
                      value={value === "__unknown__" ? "" : value}
                      colors={cfg.tagColors || {}}
                    />
                    <input
                      type="color"
                      aria-label={"Farbe für " + label}
                      value={
                        cfg.tagColors?.[key] || defaultColor(colorField, value)
                      }
                      onInput={(e) => setColor(key, e.currentTarget.value)}
                    />
                    <button
                      type="button"
                      class="text-button"
                      disabled={!cfg.tagColors?.[key]}
                      onClick={() => setColor(key)}
                      aria-label={"Standardfarbe für " + label}
                    >
                      Standard
                    </button>
                  </div>
                );
              })}
            </div>
            {!colorValues.length && (
              <p class="quiet">
                Noch keine Tags erkannt. Synchronisiere zuerst eine Bibliothek.
              </p>
            )}
          </fieldset>
        </div>
        {error && (
          <p class="error" role="alert">
            {error}
          </p>
        )}
        <div
          hidden={section !== "maintenance"}
          class="settings-section maintenance-panel"
        >
          <p class="quiet">
            Sichere Cache und Einstellungen. Eine vollständige Appdata-Sicherung
            benötigt zusätzlich den lokalen Schlüssel.
          </p>{" "}
          <button
            type="button"
            class="button"
            onClick={async () => {
              try {
                await api("backup", "POST");
                setTest({
                  ...test,
                  backup:
                    "Konsistente Sicherung im Appdata-Verzeichnis erstellt.",
                });
              } catch (e) {
                setError((e as Error).message);
              }
            }}
          >
            <Database size={15} /> Cache sichern
          </button>
          {test.backup && <p role="status">{test.backup}</p>}
        </div>
        <div class="modal-actions settings-save-bar">
          <button type="button" class="button" onClick={close}>
            Schließen
          </button>
          <button class="primary" disabled={busy}>
            {busy ? (
              <LoaderCircle class="spin" size={16} />
            ) : (
              <Check size={16} />
            )}{" "}
            Einstellungen speichern
          </button>
        </div>
      </form>
      <div hidden={section !== "maintenance"}>
        <details class="password-settings">
          <summary>
            <LockKeyhole size={15} /> Passwort ändern
          </summary>
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              try {
                await api("password", "PUT", { current, password });
                location.reload();
              } catch (e) {
                setError((e as Error).message);
              }
            }}
          >
            <label>
              Aktuelles Passwort
              <input
                type="password"
                autoComplete="current-password"
                value={current}
                required
                onInput={(e) => setCurrent(e.currentTarget.value)}
              />
            </label>
            <label>
              Neues Passwort
              <input
                type="password"
                autoComplete="new-password"
                value={password}
                required
                minLength={12}
                onInput={(e) => setPassword(e.currentTarget.value)}
              />
            </label>
            <button class="button" disabled={demo}>
              Passwort ändern und abmelden
            </button>
          </form>
        </details>
      </div>
    </Modal>
  );
}
