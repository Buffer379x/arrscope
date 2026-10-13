import { interfaceThemes, themeLabels } from "../themes";
import { audioLanguageLabel } from "../languages";
import { LanguageFlag } from "./LanguageFlag";
import { TagReview, type TagPreview, type TagReviewRequest } from "./TagReview";
import { confirmDeletion } from "../confirmation";
import { TagPolicyDialog } from "./TagPolicyDialog";
import {
  tagTransition,
  type TagTransition,
  type TagDecision,
} from "../tag-policy";
import { notify } from "../notifications";
import { CustomTagsSettings } from "./CustomTagsSettings";
import type { CustomTag } from "../types";
import { LanguageSelect, t, useI18n, getLocale } from "../i18n";
import { SystemInfo } from "./SystemInfo";
import { RecognitionPage } from "./RecognitionPage";

import {
  RotateCcw,
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
  customTags,
  customChanged,
  settings,
  status,
  connectionError,
  demo,
  saved,
  onDirty,
  refresh,
  initialSection = "connections",
}: {
  customTags: CustomTag[];
  customChanged: () => Promise<void>;
  settings: Settings;
  status: Status;
  connectionError?: string;
  demo: boolean;
  saved: () => Promise<void>;
  onDirty: (dirty: boolean) => void;
  refresh: () => Promise<void>;
  initialSection?: string;
}) {
  const { locale, setLocale } = useI18n();
  const [draftLocale, setDraftLocale] = useState(locale);
  const [tagChanges, setTagChanges] = useState<TagTransition[] | null>(null);
  const settingsFormId = useId();
  const [customDirty, setCustomDirty] = useState(false);
  const [rulesDirty, setRulesDirty] = useState(false);
  const [pendingReview, setPendingReview] = useState<Service | null>(null);
  const [tagSaveReview, setTagSaveReview] = useState<{
    next: Settings;
    requests: TagReviewRequest[];
    previews: TagPreview[];
  } | null>(null);
  const [syncing, setSyncing] = useState<Record<string, boolean>>({});
  const contentRef = useRef<HTMLDivElement>(null);
  const [cfg, setCfg] = useState<Settings>(structuredClone(settings)),
    [busy, setBusy] = useState(false),
    [testing, setTesting] = useState<Record<string, boolean>>({}),
    [current, setCurrent] = useState(""),
    [password, setPassword] = useState(""),
    [section, setSection] = useState(initialSection),
    [colorField, setColorField] = useState("resolution"),
    [recognized, setRecognized] = useState<Record<string, string[]>>({});
  useEffect(() => {
    Promise.all([
      api<Record<string, string[]>>("display-tags"),
      api<Record<string, string[]>>("facets?service=radarr"),
      api<Record<string, string[]>>("facets?service=sonarr"),
    ])
      .then((all) => {
        const values: Record<string, string[]> = {};
        colorFields.forEach((field) => {
          values[field] = [
            ...new Set(
              all
                .slice(1)
                .flatMap((f) => f[field] || [])
                .filter(
                  (value) =>
                    (all[0][field] || []).includes(value) &&
                    (field !== "custom" || value !== "__unknown__"),
                ),
            ),
          ].sort((a, b) =>
            field === "custom"
              ? (
                  customTags.find((tag) => String(tag.id) === a)?.name || a
                ).localeCompare(
                  customTags.find((tag) => String(tag.id) === b)?.name || b,
                )
              : a.localeCompare(b),
          );
        });
        setRecognized(values);
      })
      .catch(() => {});
  }, [
    customTags,
    status.recognitionRevision,
    status.sync?.radarr?.lastSuccess,
    status.sync?.sonarr?.lastSuccess,
  ]);
  const dirty =
    JSON.stringify(cfg) !== JSON.stringify(settings) || draftLocale !== locale;
  useEffect(() => {
    onDirty(dirty || rulesDirty || customDirty);
  }, [dirty, rulesDirty, customDirty]);
  useEffect(() => () => onDirty(false), []);
  useEffect(() => {
    const warn = (e: BeforeUnloadEvent) => {
      if (dirty || rulesDirty || customDirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty, rulesDirty, customDirty]);
  const changeSection = (next: string) => {
    if (
      next !== section &&
      section === "custom" &&
      customDirty &&
      !confirm(t("Discard unsaved settings changes?"))
    )
      return;
    if (
      next !== section &&
      section === "recognition" &&
      rulesDirty &&
      !confirm(t("Discard unsaved rule changes?"))
    )
      return;
    if (
      dirty &&
      next !== section &&
      !confirm(t("Discard unsaved settings changes?"))
    )
      return;
    if (dirty && next !== section) {
      setCfg(structuredClone(settings));
      setDraftLocale(locale);
    }
    setSection(next);
  };
  const manualSync = async (service: ConnectionService) => {
    setSyncing((current) => ({ ...current, [service]: true }));
    try {
      await api("sync", "POST", { service });
      await refresh();
      notify(
        t("Sync queued. Services run one at a time; you can keep filtering."),
      );
    } catch {
    } finally {
      setSyncing((current) => ({ ...current, [service]: false }));
    }
  };
  // Stored overrides are retained for reactivation but never resurrect inactive values.
  const colorValues = [...new Set(recognized[colorField] || [])].sort();
  const setColor = (key: string, color?: string) => {
    setCfg((current) => {
      const colors = { ...current.tagColors };
      if (color) colors[key] = color;
      else delete colors[key];
      const textColors = { ...current.tagTextColors };
      if (!color) delete textColors[key];
      return { ...current, tagColors: colors, tagTextColors: textColors };
    });
  };
  const setTextColor = (key: string, color: string) => {
    setCfg((current) => {
      const colors = { ...current.tagTextColors };
      if (color) colors[key] = color;
      else delete colors[key];
      return { ...current, tagTextColors: colors };
    });
  };
  const update = (service: ConnectionService, patch: Partial<Connection>) =>
    setCfg((current) => ({
      ...current,
      [service]: { ...current[service], ...patch },
    }));
  const saveSettings = async (
    next: Settings,
    tagApprovals: Record<string, string> = {},
    propagate = false,
  ) => {
    setBusy(true);
    try {
      await api("settings", "PUT", { ...next, tagApprovals });
      setTagChanges(null);
      setTagSaveReview(null);
      setCfg(await api<Settings>("settings"));
      await saved();
      setLocale(draftLocale);
      notify(t("Settings saved."));
    } catch (e) {
      if (propagate) throw e;
    } finally {
      setBusy(false);
    }
  };
  const submit = async (e: Event) => {
    e.preventDefault();
    const changes = (["radarr", "sonarr"] as Service[])
      .map((service) => tagTransition(service, settings[service], cfg[service]))
      .filter((c) => c.needsMode || c.disabled.length);
    if (changes.length) {
      setTagChanges(changes);
      return;
    }
    await saveSettings(cfg);
  };
  const confirmTags = async (choices: Record<string, TagDecision>) => {
    const next = structuredClone(cfg);
    for (const service of ["radarr", "sonarr"] as Service[]) {
      const choice = choices[service];
      if (choice?.mode) next[service].tagSyncMode = choice.mode;
      if (choice?.disable) next[service].tagSyncDisable = choice.disable;
    }
    setBusy(true);
    try {
      const requests = (Object.keys(choices) as Service[]).map((service) => ({
        service,
        config: next[service],
      }));
      const previews: TagPreview[] = [];
      for (const request of requests)
        previews.push(await api<TagPreview>("tag-preview", "POST", request));
      setCfg(next);
      setTagChanges(null);
      if (
        previews.some((p) =>
          p.rows.some((row) => row.assignments > 0 || row.deleteDefinition),
        )
      )
        setTagSaveReview({ next, requests, previews });
      else
        await saveSettings(
          next,
          Object.fromEntries(previews.map((p) => [p.service, p.token])),
        );
    } catch {
    } finally {
      setBusy(false);
    }
  };

  return (
    <section class="settings-page" aria-label={t("Settings")}>
      {tagSaveReview && (
        <TagReview
          requests={tagSaveReview.requests}
          initial={tagSaveReview.previews}
          close={() => setTagSaveReview(null)}
          confirm={(tokens) => saveSettings(tagSaveReview.next, tokens, true)}
        />
      )}
      {pendingReview && (
        <TagReview
          requests={[
            { service: pendingReview, config: settings[pendingReview] },
          ]}
          close={() => setPendingReview(null)}
          confirm={async (tokens) => {
            await api("tag-approval", "POST", {
              service: pendingReview,
              token: tokens[pendingReview],
            });
            setPendingReview(null);
            await refresh();
            notify(t("Tag changes approved."));
          }}
        />
      )}
      {tagChanges && (
        <TagPolicyDialog
          changes={tagChanges}
          busy={busy}
          close={() => setTagChanges(null)}
          save={confirmTags}
        />
      )}
      <div class="library-intro">
        <h1>{t("Settings")}</h1>
      </div>
      <div class="settings-layout">
        <nav class="settings-tabs" aria-label={t("Settings sections")}>
          {[
            ["connections", t("Services")],
            ["appearance", t("Display")],
            ["custom", t("Custom tags")],
            ["recognition", t("Tag rules")],
            ["system", t("System")],
          ].map(([value, label]) => (
            <button
              type="button"
              class={section === value ? "active" : ""}
              aria-pressed={section === value}
              onClick={() => {
                changeSection(value);
                contentRef.current?.scrollTo({ top: 0 });
              }}
            >
              {label}
            </button>
          ))}
        </nav>
        <div class="settings-content" ref={contentRef}>
          {section === "recognition" && (
            <RecognitionPage onDirty={setRulesDirty} saved={refresh} />
          )}
          {section === "custom" && (
            <CustomTagsSettings
              tags={customTags}
              changed={async () => {
                await customChanged();
                const tags = await api<CustomTag[]>("custom-tags");
                const ids = new Set(tags.map((tag) => String(tag.id)));
                setCfg((current) => ({
                  ...current,
                  tagColors: Object.fromEntries(
                    Object.entries(current.tagColors || {}).filter(
                      ([key]) =>
                        !key.startsWith("custom:") || ids.has(key.slice(7)),
                    ),
                  ),
                  tagTextColors: Object.fromEntries(
                    Object.entries(current.tagTextColors || {}).filter(
                      ([key]) =>
                        !key.startsWith("custom:") || ids.has(key.slice(7)),
                    ),
                  ),
                }));
              }}
              onDirty={setCustomDirty}
            />
          )}
          <form
            id={settingsFormId}
            onSubmit={submit}
            hidden={["recognition", "custom"].includes(section)}
          >
            <header class="settings-section-heading">
              <h2>
                {t(
                  section === "connections"
                    ? "Services"
                    : section === "appearance"
                      ? "Display"
                      : section === "colors"
                        ? "Colors"
                        : "System",
                )}
              </h2>
              <button
                type="submit"
                form={settingsFormId}
                class="primary"
                disabled={busy || !dirty}
                hidden={["system", "recognition", "custom"].includes(section)}
              >
                {busy ? (
                  <LoaderCircle class="spin" size={16} />
                ) : (
                  <Check size={16} />
                )}{" "}
                {t("Save settings")}
              </button>
              <p class="quiet">
                {t(
                  section === "connections"
                    ? "Configure connections, synchronization intervals and optional outbound tags."
                    : section === "appearance"
                      ? "Choose your interface language, library layout and default columns."
                      : section === "colors"
                        ? "Customize tag backgrounds, text colors and warning indicators."
                        : "Check synchronization, manage backups and secure your account.",
                )}
              </p>
            </header>
            {demo && !["system", "recognition", "custom"].includes(section) && (
              <div class="notice">
                {t(
                  "Demo connections are locked. Try the TV view, default columns and tag colors.",
                )}
              </div>
            )}
            <div class="settings-section" hidden={section !== "connections"}>
              {(["radarr", "sonarr", "emby"] as ConnectionService[]).map(
                (service) => (
                  <section class="connection">
                    <h3 class="card-title">
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
                    </h3>
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
                                "Emby dashboard → API keys. Use a dedicated key for Scopearr.",
                              )
                            : t("In each service under Settings → General.")}
                        </small>
                      </label>
                      <label class="check-field">
                        <input
                          type="checkbox"
                          disabled={demo}
                          checked={!!cfg[service].clearKey}
                          onChange={async (e) => {
                            const input = e.currentTarget,
                              checked = input.checked;
                            if (
                              checked &&
                              !(await confirmDeletion(
                                t("Delete the saved API key when saving?"),
                              ))
                            ) {
                              input.checked = false;
                              return;
                            }
                            update(service, {
                              clearKey: checked,
                              ...(checked
                                ? { enabled: false, apiKey: "" }
                                : {}),
                            });
                          }}
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
                      {service !== "emby" && (
                        <section class="outbound-tag-settings">
                          <h4>
                            {t("Tag synchronization to {service}", {
                              service:
                                service === "radarr" ? "Radarr" : "Sonarr",
                            })}
                          </h4>
                          <p class="quiet">
                            {t(
                              "Disabled by default. When saving changes, choose whether existing tags should be kept or overwritten, and whether disabled groups should be kept or deleted.",
                            )}
                          </p>
                          <div class="tag-group-options">
                            {[
                              "resolution",
                              "codec",
                              "source",
                              "hdr",
                              "bitDepth",
                              "audio",
                              "channels",
                              "language",
                              "custom",
                            ].map((field) => (
                              <label class="check-field">
                                <input
                                  type="checkbox"
                                  disabled={demo}
                                  checked={
                                    !!cfg[service].tagSync?.[
                                      field as keyof import("../types").TagGroups
                                    ]
                                  }
                                  onChange={(e) =>
                                    update(service, {
                                      tagSync: {
                                        ...cfg[service].tagSync,
                                        [field]: e.currentTarget.checked,
                                      } as import("../types").TagGroups,
                                    })
                                  }
                                />
                                {labels[field]}
                              </label>
                            ))}
                          </div>
                          <small>
                            {t(
                              "Source includes processing tags such as Remux. Series collect the values of all episode files. Labels use lowercase letters, digits and hyphens.",
                            )}
                          </small>
                        </section>
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
                              notify(
                                t("Connected · version {version}", {
                                  version: r.version,
                                }),
                              );
                            } catch {
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
                      </div>
                    </div>
                  </section>
                ),
              )}
            </div>
            <div class="settings-section" hidden={section !== "appearance"}>
              <div class="appearance-preferences">
                <section class="connection">
                  <h3 class="card-title">{t("Interface language")}</h3>
                  <LanguageSelect
                    value={draftLocale}
                    onChange={setDraftLocale}
                  />
                  <small>
                    {t(
                      "Applied after saving in this browser. Media names and filter values stay unchanged.",
                    )}
                  </small>
                </section>
                <section class="connection">
                  <h3 class="card-title">{t("Interface theme")}</h3>
                  <select
                    aria-label={t("Interface theme")}
                    value={cfg.interfaceTheme || "default"}
                    onChange={(e) =>
                      setCfg({ ...cfg, interfaceTheme: e.currentTarget.value })
                    }
                  >
                    {interfaceThemes.map((theme) => (
                      <option value={theme}>{t(themeLabels[theme])}</option>
                    ))}
                  </select>
                  <small>
                    {t(
                      "Applied after saving. Every theme supports dark and light mode; layout and tag colors stay unchanged.",
                    )}
                  </small>
                </section>
              </div>
              <section class="connection">
                <h3 class="card-title">
                  <Layers size={16} />
                  {t("TV view")}
                </h3>
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
              </section>
              <section class="connection">
                <h3 class="card-title">{t("Default columns")}</h3>
                <p class="quiet">
                  {t(
                    "Choose visible columns and move them with the arrows. The order runs left to right, including hidden columns. Both are saved permanently. Filters and CSV still include all fields.",
                  )}
                </p>
                <div class="column-settings">
                  {(["radarr", "sonarr"] as Service[]).map((service) => (
                    <section class="columns-group">
                      <h3 class="card-title">
                        {service === "radarr"
                          ? t("Movies · Radarr")
                          : t("TV series · Sonarr")}
                      </h3>
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
                        class="icon-button"
                        aria-label={t("Restore defaults")}
                        title={t("Restore defaults")}
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
                        <RotateCcw size={16} />
                      </button>
                    </section>
                  ))}
                </div>
              </section>
            </div>
            <div class="settings-section" hidden={section !== "appearance"}>
              <section class="connection warning-color-settings">
                <h3 class="card-title">{t("Warning colors")}</h3>
                <p class="quiet">
                  {t(
                    "Metadata disagreements and conflicting recognition rules use separate icons.",
                  )}
                </p>
                <div class="table-wrap color-table-wrap">
                  <table class="color-settings-table warning-colors-table">
                    <thead>
                      <tr>
                        <th>{t("Tag preview")}</th>
                        <th>{t("Color")}</th>
                        <th>{t("Reset column")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {["metadata", "recognition"].map((key) => (
                        <tr>
                          <td>
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
                          </td>
                          <td>
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
                          </td>
                          <td>
                            <button
                              type="button"
                              class="icon-button color-reset"
                              aria-label={t("Restore defaults")}
                              title={t("Restore defaults")}
                              onClick={() => {
                                if (
                                  confirm(
                                    t(
                                      "Reset this warning color to its default?",
                                    ),
                                  )
                                )
                                  setCfg({
                                    ...cfg,
                                    warningColors: {
                                      ...cfg.warningColors,
                                      [key]: defaultWarningColors[key],
                                    },
                                  });
                              }}
                            >
                              <RotateCcw size={16} />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>
              <section class="connection tag-color-settings">
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
                <div class="table-wrap color-table-wrap">
                  <table class="color-settings-table">
                    <thead>
                      <tr>
                        <th>{t("Tag preview")}</th>
                        <th>{t("Color")}</th>
                        <th>{t("Text")}</th>
                        <th>{t("Reset column")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {colorValues
                        .filter(
                          (value) =>
                            colorField !== "language" ||
                            value !== "__unknown__",
                        )
                        .map((value) => {
                          const key = colorField + ":" + value;
                          const label =
                            value === "__unknown__"
                              ? t("Unknown")
                              : colorField === "language"
                                ? t(audioLanguageLabel(value, getLocale()))
                                : colorField === "custom"
                                  ? customTags.find(
                                      (tag) => String(tag.id) === value,
                                    )?.name || value
                                  : value;
                          return (
                            <tr>
                              <td>
                                {colorField === "language" ? (
                                  <LanguageFlag
                                    value={value}
                                    colors={cfg.tagColors || {}}
                                  />
                                ) : (
                                  <Tag
                                    kind={colorField}
                                    value={value === "__unknown__" ? "" : value}
                                    colors={cfg.tagColors || {}}
                                    textColors={cfg.tagTextColors || {}}
                                  ></Tag>
                                )}
                              </td>
                              <td>
                                <input
                                  type="color"
                                  aria-label={t("Color for {value}", {
                                    value: label,
                                  })}
                                  value={
                                    cfg.tagColors?.[key] ||
                                    defaultColor(colorField, value)
                                  }
                                  onInput={(e) =>
                                    setColor(key, e.currentTarget.value)
                                  }
                                />
                              </td>
                              <td>
                                <button
                                  type="button"
                                  class="icon-button tag-text-color color-action"
                                  disabled={colorField === "language"}
                                  aria-label={
                                    t("Text color for {value}", {
                                      value: label,
                                    }) +
                                    ": " +
                                    t(
                                      cfg.tagTextColors?.[key] === "#000000"
                                        ? "Black"
                                        : cfg.tagTextColors?.[key] === "#ffffff"
                                          ? "White"
                                          : "Automatic",
                                    )
                                  }
                                  title={t(
                                    cfg.tagTextColors?.[key] === "#000000"
                                      ? "Black"
                                      : cfg.tagTextColors?.[key] === "#ffffff"
                                        ? "White"
                                        : "Automatic",
                                  )}
                                  onClick={() =>
                                    setTextColor(
                                      key,
                                      cfg.tagTextColors?.[key] === "#000000"
                                        ? "#ffffff"
                                        : cfg.tagTextColors?.[key] === "#ffffff"
                                          ? ""
                                          : "#000000",
                                    )
                                  }
                                >
                                  <span>A</span>
                                  <span aria-hidden="true">
                                    {cfg.tagTextColors?.[key] === "#000000"
                                      ? "●"
                                      : cfg.tagTextColors?.[key] === "#ffffff"
                                        ? "○"
                                        : "◐"}
                                  </span>
                                </button>
                              </td>
                              <td>
                                <button
                                  type="button"
                                  class="icon-button color-action"
                                  disabled={
                                    !cfg.tagColors?.[key] &&
                                    !cfg.tagTextColors?.[key]
                                  }
                                  onClick={() => {
                                    if (
                                      confirm(
                                        t(
                                          "Reset background and text colors for {value}?",
                                          { value: label },
                                        ),
                                      )
                                    )
                                      setColor(key);
                                  }}
                                  aria-label={t("Default color for {value}", {
                                    value: label,
                                  })}
                                >
                                  <RotateCcw size={16} />
                                </button>
                              </td>
                            </tr>
                          );
                        })}
                    </tbody>
                  </table>
                </div>
                {!colorValues.length && (
                  <p class="quiet">
                    {t(
                      colorField === "custom"
                        ? "No custom tags assigned yet. Assign tags in your library or add a filename rule."
                        : "No tags detected yet. Sync a library first.",
                    )}
                  </p>
                )}
              </section>
            </div>
            <div class="settings-section" hidden={section !== "system"}>
              <SystemInfo
                status={status}
                settings={settings}
                connectionError={connectionError}
                demo={demo}
                syncing={syncing}
                manualSync={manualSync}
                reviewTags={(service) => setPendingReview(service)}
                backup={async () => {
                  await api("backup", "POST");
                  notify(
                    t("Consistent backup created in the Appdata directory."),
                  );
                }}
              />
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
                  } catch {}
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
      </div>
    </section>
  );
}
