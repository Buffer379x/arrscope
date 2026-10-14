import { useEffect, useState } from "preact/hooks";
import { HardDrive } from "lucide-preact";
import type { Settings, StorageData } from "../types";
import { t, useI18n, getLocale } from "../i18n";
import { api } from "../api";
export function storageBytes(n: number) {
  return (
    new Intl.NumberFormat(getLocale(), { maximumFractionDigits: 1 }).format(
      n / 1024 ** 4,
    ) + " TiB"
  );
}
export function StorageWidget({
  settings,
  scope,
}: {
  settings: Settings;
  scope: string;
}) {
  useI18n();
  const [data, setData] = useState<StorageData | null>(null);
  const [error, setError] = useState(false);
  useEffect(() => {
    let current = true;
    const abort = new AbortController();
    const load = () =>
      api<StorageData>("storage", "GET", undefined, abort.signal)
        .then((d) => {
          if (current) {
            setData(d);
            setError(false);
          }
        })
        .catch((e) => {
          if (current && e.name !== "AbortError") setError(true);
        });
    load();
    const timer = setInterval(load, 60000);
    return () => {
      current = false;
      clearInterval(timer);
      abort.abort();
    };
  }, [JSON.stringify(settings.widgets.storage)]);
  const total = data?.totals[scope];
  const used = total?.total ? (total.used / total.total) * 100 : 0;
  return (
    <article class="distribution-card storage-widget">
      <div class="distribution-title">
        <h3>{t("Storage distribution")}</h3>
        <HardDrive size={18} />
      </div>
      {total && total.total > 0 ? (
        <>
          <div
            class="storage-chart"
            role="img"
            aria-label={t("{used} used · {free} free", {
              used: storageBytes(total.used),
              free: storageBytes(total.free),
            })}
            style={{
              background: `conic-gradient(${settings.widgets.colors.used} 0% ${used}%, ${settings.widgets.colors.free} ${used}% 100%)`,
            }}
          >
            <div>
              <strong>{storageBytes(total.total)}</strong>
              <span>{t("Total space")}</span>
            </div>
          </div>
          <dl class="storage-legend">
            {["used", "free"].map((key) => (
              <div>
                <dt>
                  <span
                    class="chart-dot"
                    style={{ backgroundColor: settings.widgets.colors[key] }}
                  />
                  {t(key === "used" ? "Used space" : "Free space")}
                </dt>
                <dd>
                  {storageBytes(key === "used" ? total.used : total.free)}
                </dd>
              </div>
            ))}
          </dl>
        </>
      ) : (
        <p class="quiet">
          {t(
            error
              ? "Storage readings are stale or unavailable"
              : "Select storage paths in Settings → Widgets.",
          )}
        </p>
      )}
      {total?.error && <p class="quiet">{t(total.error)}</p>}
    </article>
  );
}
