import { ArrowUpRight, Film, Tv } from "lucide-preact";
import { useState, type Dispatch, type StateUpdater } from "preact/hooks";
import { api } from "../api";
import { confirmDeletion } from "../confirmation";
import { t, useI18n } from "../i18n";
import { notify } from "../notifications";
import type { Connection, Settings } from "../types";
import { labels, type ConnectionService } from "../ui";
import { WebhookSettings } from "./WebhookSettings";

export function ConnectionSettings({
  cfg,
  setCfg,
  settings,
  demo,
  hidden,
}: {
  cfg: Settings;
  setCfg: Dispatch<StateUpdater<Settings>>;
  settings: Settings;
  demo: boolean;
  hidden: boolean;
}) {
  useI18n();
  const [testing, setTesting] = useState<Record<string, boolean>>({});
  const [mediaProvider, setMediaProvider] = useState<"emby" | "jellyfin">(
    settings.emby.enabled || !settings.jellyfin.enabled ? "emby" : "jellyfin",
  );
  const update = (service: ConnectionService, patch: Partial<Connection>) =>
    setCfg((current) => ({
      ...current,
      [service]: { ...current[service], ...patch },
    }));

  return (
    <div class="settings-section" hidden={hidden}>
      <section class="connection">
        <label>
          {t("Sync timezone")}
          <input
            value={cfg.syncTimezone || ""}
            placeholder={Intl.DateTimeFormat().resolvedOptions().timeZone}
            onInput={(e) =>
              setCfg({ ...cfg, syncTimezone: e.currentTarget.value })
            }
          />
        </label>
        <small>
          {t(
            "Clock schedules use this IANA timezone. Empty uses UTC; no start time keeps the existing interval schedule.",
          )}
        </small>
      </section>
      {(["radarr", "sonarr", mediaProvider] as ConnectionService[]).map(
        (service) => (
          <section class="connection" key={service}>
            <header class="connection-heading">
              <h3 class="card-title">
                {service === "radarr" ? <Film size={16} /> : <Tv size={16} />}{" "}
                {service === "radarr"
                  ? "Radarr"
                  : service === "sonarr"
                    ? "Sonarr"
                    : t("Media server")}
              </h3>
              <div class="connection-heading-test">
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
                      const r = await api<{ version: string }>("test", "POST", {
                        service,
                        url: cfg[service].url,
                        apiKey: cfg[service].apiKey || "",
                      });
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
            </header>
            {(service === "emby" || service === "jellyfin") && (
              <p class="quiet">
                {t(
                  "Adds existing video and audio stream metadata. No scans or media access. Matching requires a unique filename and identical file size.",
                )}
              </p>
            )}
            {(service === "emby" || service === "jellyfin") && (
              <label class="media-provider-select">
                {t("Media server")}
                <select
                  aria-label={t("Media server")}
                  value={mediaProvider}
                  onChange={(e) =>
                    setMediaProvider(
                      e.currentTarget.value as "emby" | "jellyfin",
                    )
                  }
                >
                  <option value="emby">Emby</option>
                  <option value="jellyfin">Jellyfin</option>
                </select>
              </label>
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
                  {t("URL subpaths are supported, for example /{service}.", {
                    service,
                  })}
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
                  {service === "emby" || service === "jellyfin"
                    ? t(
                        "Create a dedicated API key in the selected media server dashboard.",
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
                      ...(checked ? { enabled: false, apiKey: "" } : {}),
                    });
                  }}
                />
                {t("Delete saved API key on save")}
              </label>
              {(service === "emby" || service === "jellyfin") && (
                <label class="path-mapping-field">
                  {t("Optional path mappings")}
                  <textarea
                    aria-label={t("Media server path mappings")}
                    disabled={demo}
                    rows={3}
                    value={cfg[service].pathMappings || ""}
                    placeholder="/movies => /media/movies"
                    onInput={(e) =>
                      update(service, {
                        pathMappings: e.currentTarget.value,
                      })
                    }
                  />
                  <small>
                    {t(
                      "One line per media path: Arr path => media server path. If configured, the full mapped path must also match. Different folder names inside the library prevent enrichment.",
                    )}
                  </small>
                </label>
              )}
              {service === "radarr" || service === "sonarr" ? (
                <div class="sync-clock-grid">
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
                  <label>
                    {t("Starting at")}
                    <input
                      type="text"
                      inputMode="text"
                      placeholder="HH:MM"
                      pattern="([01][0-9]|2[0-3]):[0-5][0-9]"
                      maxLength={5}
                      autoComplete="off"
                      disabled={demo || cfg[service].interval === 0}
                      value={cfg[service].syncStart || ""}
                      onInput={(e) => {
                        update(service, {
                          syncStart: e.currentTarget.value,
                        });
                        if (!cfg.syncTimezone)
                          setCfg((current) => ({
                            ...current,
                            syncTimezone:
                              Intl.DateTimeFormat().resolvedOptions().timeZone,
                          }));
                      }}
                    />
                  </label>
                </div>
              ) : (
                <p class="quiet">
                  {t(
                    "Synced with Radarr and Sonarr. An error preserves the last complete cache.",
                  )}
                </p>
              )}
              {(service === "radarr" || service === "sonarr") && (
                <div class="tag-integration-settings">
                  <section class="external-tag-settings">
                    <h4>{t("External tags")}</h4>
                    <label class="check-field">
                      <input
                        type="checkbox"
                        checked={!!cfg[service].tagImport}
                        onChange={(e) =>
                          update(service, {
                            tagImport: e.currentTarget.checked,
                            ...(e.currentTarget.checked &&
                            !cfg[service].tagImport
                              ? { tagImportInterval: 5 }
                              : {}),
                          })
                        }
                      />
                      {t("Import external tags as custom tags")}
                    </label>
                    <label>
                      {t("Tag check interval")}
                      <select
                        value={cfg[service].tagImportInterval ?? 5}
                        disabled={!cfg[service].tagImport}
                        onChange={(e) =>
                          update(service, {
                            tagImportInterval: Number(e.currentTarget.value),
                          })
                        }
                      >
                        {[0, 5, 15, 30, -1].map((v) => (
                          <option value={v}>
                            {t(
                              v === 0
                                ? "Manual only"
                                : v === -1
                                  ? "With library sync"
                                  : v === 5
                                    ? "Every 5 minutes"
                                    : v === 15
                                      ? "Every 15 minutes"
                                      : "Every 30 minutes",
                            )}
                          </option>
                        ))}
                      </select>
                    </label>
                    <p class="quiet">
                      {t(
                        "Local edits are checked immediately. Enable outbound Custom tags below to send them back. Auto-tag rule references protect edits; no file or media server requests are needed for tag checks.",
                      )}
                    </p>
                  </section>
                  <section class="outbound-tag-settings">
                    <h4>
                      {t("Tag synchronization to {service}", {
                        service: service === "radarr" ? "Radarr" : "Sonarr",
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
                </div>
              )}
              {(service === "radarr" || service === "sonarr") && (
                <WebhookSettings
                  service={service}
                  config={cfg[service]}
                  changed={(patch) => update(service, patch)}
                  savedEnabled={!!settings[service].webhookEnabled}
                  demo={demo}
                />
              )}
            </div>
          </section>
        ),
      )}
    </div>
  );
}
