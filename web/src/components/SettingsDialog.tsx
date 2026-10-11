import { LanguageSelect, t, useI18n } from "../i18n";
import { SystemInfo } from "./SystemInfo";
import { Modal } from "./Modal";

import {
  AlertTriangle,
  OctagonAlert,
  SlidersHorizontal,
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
import { useEffect, useId, useRef, useState } from "preact/hooks";
import type { Connection, Service, Settings, Status } from "../types";

import { api } from "../api";
import {
  defaultWarningColors,
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
  status,
  connectionError,
  demo,
  close,
  saved,
  onRecognition,
  initialSection = "connections",
}: {
  settings: Settings;
  status: Status;
  connectionError?: string;
  demo: boolean;
  close: () => void;
  saved: () => Promise<void>;
  onRecognition: () => void;
  initialSection?: string;
}) {
  useI18n();
  const settingsFormId = useId();
  const contentRef = useRef<HTMLDivElement>(null);
  const [cfg, setCfg] = useState<Settings>(structuredClone(settings)),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [test, setTest] = useState<Record<string, string>>({}),
    [testing, setTesting] = useState<Record<string, boolean>>({}),
    [current, setCurrent] = useState(""),
    [password, setPassword] = useState(""),
    [section, setSection] = useState(initialSection),
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
    <Modal title={t("Settings")} close={close} wide className="settings-modal">
      <nav class="settings-tabs" aria-label={t("Settings sections")}>
        {[
          ["connections", t("Connections")],
          ["appearance", t("Appearance")],
          ["system", t("System")],
        ].map(([value, label]) => (
          <button
            type="button"
            class={section === value ? "active" : ""}
            aria-pressed={section === value}
            onClick={() => {
              setSection(value);
              contentRef.current?.scrollTo({ top: 0 });
            }}
          >
            {label}
          </button>
        ))}
      </nav>
      <div class="settings-content" ref={contentRef}>
        <form id={settingsFormId} onSubmit={submit}>
          <p class="quiet" hidden={section === "system"}>
            {t("Your services, your cache. No media directories required.")}
          </p>
          {demo && section !== "system" && (
            <div class="notice">
              {t(
                "Demo connections are locked. Try the TV view, default columns and tag colors.",
              )}
            </div>
          )}
          <div class="settings-section" hidden={section !== "connections"}>
            {(["radarr", "sonarr", "emby"] as ConnectionService[]).map(
              (service) => (
                <fieldset class="connection">
                  <legend>
                    {service === "radarr" ? (
                      <Film size={16} />
                    ) : (
                      <Tv size={16} />
                    )}{" "}
                    {service === "radarr"
                      ? "Radarr"
                      : service === "sonarr"
                        ? "Sonarr"
                        : t("Emby · optional")}
                  </legend>
                  {service === "emby" && (
                    <p class="quiet">
                      {t(
                        "Adds existing audio track metadata. No scans or media access. Matching requires a unique filename and identical file size.",
                      )}
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
                    {t("Enable service")}
                  </label>
                  <div class="connection-grid">
                    <label>
                      {t("Server address")}
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
                        {t(
                          "URL subpaths are supported, for example /{service}.",
                          { service },
                        )}
                      </small>
                    </label>
                    <label>
                      {t("API key")}
                      <input
                        type="password"
                        disabled={demo}
                        autoComplete="off"
                        placeholder={
                          cfg[service].hasKey
                            ? t("Saved · leave empty to keep")
                            : t("Enter API key")
                        }
                        value={cfg[service].apiKey || ""}
                        onInput={(e) =>
                          update(service, { apiKey: e.currentTarget.value })
                        }
                      />
                      <small>
                        {service === "emby"
                          ? t(
                              "Emby dashboard → API keys. Use a dedicated key for ArrScope.",
                            )
                          : t("In each service under Settings → General.")}
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
                      {t("Delete saved API key on save")}
                    </label>
                    {service === "emby" && (
                      <label class="path-mapping-field">
                        {t("Optional path mappings")}
                        <textarea
                          aria-label={t("Emby path mappings")}
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
                          {t(
                            "One line per media path: Arr path => Emby path. If configured, the full mapped path must also match. Different folder names inside the library prevent enrichment.",
                          )}
                        </small>
                      </label>
                    )}
                    {service !== "emby" ? (
                      <label>
                        {t("Synchronize")}
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
                            [0, t("Manual only")],
                            [15, t("Every 15 minutes")],
                            [30, t("Every 30 minutes")],
                            [60, t("Hourly")],
                            [360, t("Every 6 hours")],
                            [1440, t("Daily")],
                          ].map(([value, label]) => (
                            <option value={value}>{label}</option>
                          ))}
                        </select>
                      </label>
                    ) : (
                      <p class="quiet">
                        {t(
                          "Synced with Radarr and Sonarr. An error preserves the last complete cache.",
                        )}
                      </p>
                    )}
                    <div class="connection-test">
                      <button
                        type="button"
                        class="button"
                        disabled={demo || testing[service]}
                        onClick={async () => {
                          setTesting((current) => ({
                            ...current,
                            [service]: true,
                          }));
                          setTest((current) => ({
                            ...current,
                            [service]: t("Testing …"),
                          }));
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
                            setTest((current) => ({
                              ...current,
                              [service]: t("Connected · version {version}", {
                                version: r.version,
                              }),
                            }));
                          } catch (e) {
                            setTest((current) => ({
                              ...current,
                              [service]: (e as Error).message,
                            }));
                          } finally {
                            setTesting((current) => ({
                              ...current,
                              [service]: false,
                            }));
                          }
                        }}
                      >
                        {t("Test connection")}
                        <ArrowUpRight size={14} />
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
              <legend>{t("Interface language")}</legend>
              <LanguageSelect />
              <small>
                {t(
                  "Saved in this browser. Media names and filter values stay unchanged.",
                )}
              </small>
            </fieldset>
            <fieldset class="connection">
              <legend>
                <Layers size={16} />
                {t("TV view")}
              </legend>
              <label>
                {t("Results with active filters")}
                <select
                  value={cfg.seasonMode}
                  onChange={(e) =>
                    setCfg({
                      ...cfg,
                      seasonMode: e.currentTarget
                        .value as Settings["seasonMode"],
                    })
                  }
                >
                  <option value="matches">
                    {t("Show matching files only")}
                  </option>
                  <option value="season">
                    {t("Full matching season · highlight matches")}
                  </option>
                </select>
              </label>
              <small>
                {t(
                  "One matching file is enough to show its season in context.",
                )}
              </small>
            </fieldset>
            <fieldset class="connection">
              <legend>{t("Default columns")}</legend>
              <p class="quiet">
                {t(
                  "Choose visible columns and move them with the arrows. The order runs left to right, including hidden columns. Both are saved permanently. Filters and CSV still include all fields.",
                )}
              </p>
              <div class="column-settings">
                {(["radarr", "sonarr"] as Service[]).map((service) => (
                  <fieldset>
                    <legend>
                      {service === "radarr"
                        ? t("Movies · Radarr")
                        : t("TV series · Sonarr")}
                    </legend>
                    <div class="column-options">
                      {cfg.columnOrder[service].map((field, index) => (
                        <div class="column-option" key={field}>
                          <label class="check-field">
                            <input
                              type="checkbox"
                              aria-label={
                                (service === "radarr"
                                  ? t("Movies")
                                  : t("TV series")) +
                                ": " +
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
                                ? t("Title (always visible)")
                                : t("Episode (always visible)")
                              : columnLabels[field]}
                          </label>
                          <div class="column-move-tools">
                            {([-1, 1] as const).map((direction) => (
                              <button
                                type="button"
                                class="icon-button"
                                aria-label={
                                  (service === "radarr"
                                    ? t("Movies") + ": "
                                    : t("TV series") + ": ") +
                                  columnLabels[field] +
                                  " · " +
                                  (direction === -1
                                    ? t("Move up")
                                    : t("Move down"))
                                }
                                title={
                                  direction === -1
                                    ? t("Move column left")
                                    : t("Move column right")
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
                      {t("Restore defaults")}
                    </button>
                  </fieldset>
                ))}
              </div>
            </fieldset>
            <fieldset class="connection warning-color-settings">
              <legend>{t("Warning colors")}</legend>
              <p class="quiet">
                {t(
                  "Metadata disagreements and conflicting recognition rules use separate icons.",
                )}
              </p>
              {["metadata", "recognition"].map((key) => (
                <div class="tag-color-row">
                  <label style={{ color: cfg.warningColors[key] }}>
                    {key === "metadata" ? (
                      <AlertTriangle size={17} />
                    ) : (
                      <OctagonAlert size={17} />
                    )}{" "}
                    {t(
                      key === "metadata"
                        ? "Metadata disagreement"
                        : "Conflicting recognition rules",
                    )}
                  </label>
                  <input
                    type="color"
                    aria-label={t(
                      key === "metadata"
                        ? "Metadata warning color"
                        : "Recognition warning color",
                    )}
                    value={cfg.warningColors[key]}
                    onInput={(e) =>
                      setCfg({
                        ...cfg,
                        warningColors: {
                          ...cfg.warningColors,
                          [key]: e.currentTarget.value,
                        },
                      })
                    }
                  />
                  <button
                    type="button"
                    class="text-button"
                    onClick={() =>
                      setCfg({
                        ...cfg,
                        warningColors: {
                          ...cfg.warningColors,
                          [key]: defaultWarningColors[key],
                        },
                      })
                    }
                  >
                    {t("Default")}
                  </button>
                </div>
              ))}
            </fieldset>
            <fieldset class="connection tag-color-settings">
              <legend>{t("Tag colors")}</legend>
              <p class="quiet">
                {t(
                  "Custom colors apply to both libraries and themes. They remain saved until reset.",
                )}
              </p>
              <label>
                {t("Tag field")}
                <select
                  aria-label={t("Tag field")}
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
                  const label = value === "__unknown__" ? t("Unknown") : value;
                  return (
                    <div class="tag-color-row">
                      <Tag
                        kind={colorField}
                        value={value === "__unknown__" ? "" : value}
                        colors={cfg.tagColors || {}}
                      />
                      <input
                        type="color"
                        aria-label={t("Color for {value}", { value: label })}
                        value={
                          cfg.tagColors?.[key] ||
                          defaultColor(colorField, value)
                        }
                        onInput={(e) => setColor(key, e.currentTarget.value)}
                      />
                      <button
                        type="button"
                        class="text-button"
                        disabled={!cfg.tagColors?.[key]}
                        onClick={() => setColor(key)}
                        aria-label={t("Default color for {value}", {
                          value: label,
                        })}
                      >
                        {t("Default")}
                      </button>
                    </div>
                  );
                })}
              </div>
              {!colorValues.length && (
                <p class="quiet">
                  {t("No tags detected yet. Sync a library first.")}
                </p>
              )}
            </fieldset>
          </div>
          <div class="settings-section" hidden={section !== "system"}>
            <fieldset class="connection">
              <legend>{t("Filename recognition")}</legend>
              <p class="quiet">
                {t(
                  "Manage filename rules and preview results. Saving recalculates the cache without scanning media or contacting your services.",
                )}
              </p>
              <button
                type="button"
                class="button"
                onClick={() => {
                  if (
                    JSON.stringify(cfg) === JSON.stringify(settings) ||
                    confirm(t("Discard unsaved settings changes?"))
                  )
                    onRecognition();
                }}
              >
                <SlidersHorizontal size={15} />
                {t("Manage recognition rules")}
              </button>
            </fieldset>
            <SystemInfo
              status={status}
              settings={settings}
              connectionError={connectionError}
            />
          </div>
          {error && (
            <p class="error" role="alert">
              {error}
            </p>
          )}
          <div
            hidden={section !== "system"}
            class="settings-section maintenance-panel"
          >
            <p class="quiet">
              {t(
                "Back up cache and settings. A complete Appdata backup also requires the local key.",
              )}
            </p>{" "}
            <button
              type="button"
              class="button"
              onClick={async () => {
                try {
                  await api("backup", "POST");
                  setTest({
                    ...test,
                    backup: t(
                      "Consistent backup created in the Appdata directory.",
                    ),
                  });
                } catch (e) {
                  setError((e as Error).message);
                }
              }}
            >
              <Database size={15} />
              {t("Back up cache")}
            </button>
            {test.backup && <p role="status">{test.backup}</p>}
          </div>
        </form>
        <div hidden={section !== "system"}>
          <details class="password-settings">
            <summary>
              <LockKeyhole size={15} />
              {t("Change password")}
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
                {t("Current password")}
                <input
                  type="password"
                  autoComplete="current-password"
                  value={current}
                  required
                  onInput={(e) => setCurrent(e.currentTarget.value)}
                />
              </label>
              <label>
                {t("New password")}
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
                {t("Change password and sign out")}
              </button>
            </form>
          </details>
        </div>
      </div>
      <div class="modal-actions settings-save-bar">
        <button type="button" class="button" onClick={close}>
          {t("Close")}
        </button>
        <button
          type="submit"
          form={settingsFormId}
          class="primary"
          disabled={busy}
          hidden={section === "system"}
        >
          {busy ? <LoaderCircle class="spin" size={16} /> : <Check size={16} />}{" "}
          {t("Save settings")}
        </button>
      </div>
    </Modal>
  );
}
