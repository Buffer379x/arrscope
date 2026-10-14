import { ExternalTagStatus } from "./ExternalTagStatus";
import { Database, RefreshCw } from "lucide-preact";
import type { ConnectionService } from "../ui";
import { t, useI18n } from "../i18n";
import type { Settings, Status } from "../types";
import { count, date } from "../ui";

export function SystemInfo({
  status,
  settings,
  connectionError,
  demo,
  syncing,
  manualSync,
  backup,
  reviewTags,
}: {
  status: Status;
  settings: Settings;
  connectionError?: string;
  demo: boolean;
  syncing: Record<string, boolean>;
  manualSync: (service: ConnectionService) => Promise<void>;
  backup: () => Promise<void>;
  reviewTags: (service: "radarr" | "sonarr") => void;
}) {
  useI18n();
  return (
    <section class="system-info" aria-label={t("System")}>
      <dl class="system-details">
        <div>
          <dt>{t("Version")}</dt>
          <dd>
            {status.version}
            <p class="release-check-status" role="status">
              {status.releaseCheck?.successAt
                ? status.releaseCheck.updateAvailable
                  ? t("New version {version} available", {
                      version: status.releaseCheck.latest || "",
                    })
                  : t("Version is up to date")
                : t("Checking for a new version …")}
            </p>
            {status.releaseCheck?.error && (
              <small class="quiet">{t(status.releaseCheck.error)}</small>
            )}
          </dd>
        </div>
        <div>
          <dt>{t("Back up cache")}</dt>
          <dd>
            <p class="quiet">
              {t(
                "Back up cache and settings. A complete Appdata backup also requires the local key.",
              )}
            </p>
            <button
              type="button"
              class="button"
              onClick={() => void backup().catch(() => {})}
            >
              <Database size={15} />
              {t("Back up cache")}
            </button>
          </dd>
        </div>
      </dl>
      <section class="settings-card system-service-card">
        <h3>{t("Service sync status")}</h3>
        <p class="quiet">
          {t(
            "Service status reflects the last sync, not a live connection test.",
          )}
        </p>
        <div class="system-services">
          {(["radarr", "sonarr"] as const).map((service) => {
            const sync = status.sync?.[service];
            const tags = status.tagSync?.[service];
            return (
              <article key={service}>
                <div class="system-service-heading">
                  <h4>{service === "radarr" ? "Radarr" : "Sonarr"}</h4>{" "}
                  <button
                    type="button"
                    class="button system-sync-button"
                    disabled={
                      demo ||
                      !settings[service].enabled ||
                      syncing[service] ||
                      sync?.running
                    }
                    onClick={() => void manualSync(service)}
                  >
                    <RefreshCw size={15} class={sync?.running ? "spin" : ""} />
                    {t("Sync now")}
                  </button>
                  {sync?.running && (
                    <progress aria-label={t("Sync in progress …")} />
                  )}
                </div>
                <p class={sync?.error ? "error" : ""}>
                  {!settings[service].enabled
                    ? t("Disabled")
                    : sync?.running
                      ? sync.queued
                        ? t("Sync waiting for a slot …")
                        : t("Sync in progress …")
                      : sync?.error
                        ? t("Sync failed · {error}", { error: t(sync.error) })
                        : t("Last sync: {date}", {
                            date: date(sync?.lastSuccess),
                          })}
                </p>
                {status.nextSync?.[service] && (
                  <small class="next-sync">
                    {t("Next sync: {date}", {
                      date: date(status.nextSync[service]),
                    })}
                  </small>
                )}
                {settings[service].webhookEnabled && (
                  <div class="webhook-status">
                    <b>{t("Realtime updates")}</b>
                    <p class="quiet">
                      {status.webhook?.[service]?.error
                        ? t(status.webhook[service].error)
                        : t("Last webhook update: {date}", {
                            date: date(status.webhook?.[service]?.lastSuccess),
                          })}
                    </p>
                    <small>
                      {t("{count} pending", {
                        count: status.webhook?.[service]?.pending || 0,
                      })}
                    </small>
                  </div>
                )}
                {settings[service].tagImport && (
                  <ExternalTagStatus
                    service={service}
                    status={status}
                    disabled={demo || !settings[service].enabled}
                  />
                )}
                <small>
                  {count(sync?.count || 0)} {t("files in cache")}
                </small>
                {(Object.values(settings[service].tagSync || {}).some(
                  Boolean,
                ) ||
                  tags?.running ||
                  tags?.pending ||
                  tags?.error ||
                  tags?.approvalRequired ||
                  !!tags?.retainedDefinitions) && (
                  <div class="tag-sync-status">
                    <b>{t("Tag synchronization")}</b>
                    <p class={tags?.error ? "error" : "quiet"}>
                      {tags?.error
                        ? t(tags.error)
                        : tags?.running
                          ? t("Tag synchronization in progress …")
                          : tags?.pending
                            ? t("Tag synchronization pending …")
                            : t("Last sync: {date}", {
                                date: date(tags?.lastSuccess),
                              })}
                    </p>
                    {!!tags?.retainedDefinitions && (
                      <p class="quiet">
                        {t(
                          "{count} tag definitions retained because the service still uses them.",
                          { count: tags.retainedDefinitions },
                        )}
                      </p>
                    )}
                    {tags?.approvalRequired && (
                      <button
                        type="button"
                        class="button"
                        onClick={() => reviewTags(service)}
                      >
                        {t("Review tag removals")}
                      </button>
                    )}
                    {tags?.running && (
                      <progress
                        aria-label={t("Tag synchronization in progress …")}
                      />
                    )}
                  </div>
                )}
                {sync?.error && sync.lastSuccess && (
                  <p class="quiet">
                    {t("Last successful sync")}: {date(sync.lastSuccess)}
                  </p>
                )}
              </article>
            );
          })}
          <article>
            <h4>Emby</h4>
            <p>{settings.emby.enabled ? t("Enabled") : t("Disabled")}</p>
            <small>{t("Emby enriches the Radarr and Sonarr syncs.")}</small>
            <button
              type="button"
              class="button system-sync-button"
              disabled={
                demo ||
                !settings.emby.enabled ||
                syncing.emby ||
                (!settings.radarr.enabled && !settings.sonarr.enabled) ||
                Object.values(status.sync || {}).some((s) => s.running)
              }
              onClick={() => void manualSync("emby")}
            >
              <RefreshCw size={15} />
              {t("Match now")}
            </button>
            {settings.emby.enabled &&
              Object.values(status.sync || {}).some((s) => s.running) && (
                <progress aria-label={t("Sync in progress …")} />
              )}
          </article>
        </div>
      </section>
    </section>
  );
}
