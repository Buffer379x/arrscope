import { RefreshCw } from "lucide-preact";
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
}: {
  status: Status;
  settings: Settings;
  connectionError?: string;
  demo: boolean;
  syncing: Record<string, boolean>;
  manualSync: (service: ConnectionService) => Promise<void>;
}) {
  useI18n();
  return (
    <section class="system-info" aria-label={t("System")}>
      <dl class="system-details">
        <div>
          <dt>{t("Version")}</dt>
          <dd>{status.version}</dd>
        </div>
        <div>
          <dt>{t("App connection")}</dt>
          <dd role="status" class={connectionError ? "error" : ""}>
            {connectionError
              ? t("Connection unavailable")
              : t("Connected to Scopearr")}
          </dd>
        </div>
      </dl>
      <h3>{t("Service sync status")}</h3>
      <p class="quiet">
        {t(
          "Service status reflects the last sync, not a live connection test.",
        )}
      </p>
      <div class="system-services">
        {(["radarr", "sonarr"] as const).map((service) => {
          const sync = status.sync?.[service];
          return (
            <article key={service}>
              <h4>{service === "radarr" ? "Radarr" : "Sonarr"}</h4>
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
              <small>
                {count(sync?.count || 0)} {t("files in cache")}
              </small>
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
  );
}
