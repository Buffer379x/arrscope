import { StorageWidget } from "./StorageWidget";
import { getLocale, t, useI18n } from "../i18n";
import {
  BarChart3,
  Clapperboard,
  Database,
  FileVideo2,
  Film,
  LoaderCircle,
  Tv,
} from "lucide-preact";
import { useContext, useEffect, useState } from "preact/hooks";
import type { Statistics, Status, Settings } from "../types";

import { LanguageFlag } from "./LanguageFlag";
import { audioLanguageLabel } from "../languages";
import { api } from "../api";
import { count, labels, Tag, TagColors } from "../ui";
const chartPalette = [
  "#62b69d",
  "#729acb",
  "#b59ad9",
  "#d4b56f",
  "#ce8da8",
  "#78b5c2",
  "#aaa96e",
  "#c9946c",
];
export function StatisticsPage({
  status,
  settings,
  onFilter,
}: {
  status: Status;
  settings: Settings | null;
  onFilter: (field: string, value: string, scope: string) => void;
}) {
  useI18n();
  const [data, setData] = useState<Statistics | null>(null),
    [scope, setScope] = useState("all"),
    [retry, setRetry] = useState(0),
    [error, setError] = useState("");
  useEffect(() => {
    const controller = new AbortController();
    let current = true;
    setError("");
    api<Statistics>("statistics", "GET", undefined, controller.signal)
      .then((value) => {
        if (current) setData(value);
      })
      .catch((e) => {
        if (current && e.name !== "AbortError") setError(e.message);
      });
    return () => {
      current = false;
      controller.abort();
    };
  }, [
    status.sync?.radarr.lastSuccess,
    status.sync?.sonarr.lastSuccess,
    status.recognitionRevision,
    status.cacheRevision,
    retry,
  ]);
  return (
    <section class="statistics-page" aria-label={t("Dashboard")}>
      <div class="statistics-heading">
        <div>
          <h1>{t("Dashboard")}</h1>
        </div>
        <span class="quiet">
          <Database size={14} />
          {t("From the local cache")}
        </span>
      </div>

      {error && (
        <button class="button" onClick={() => setRetry((value) => value + 1)}>
          {t("Retry")}
        </button>
      )}
      {!data && !error && (
        <p class="quiet">
          <LoaderCircle class="spin" size={16} />
          {t("Loading dashboard …")}
        </p>
      )}
      {data && (
        <>
          <div class="stat-widgets">
            {[
              {
                label: t("Movies"),
                value: data.movies,
                icon: Film,
                detail: t("with existing files"),
              },
              {
                label: t("TV series"),
                value: data.series,
                icon: Tv,
                detail: t("with existing episodes"),
              },
              {
                label: t("Episodes"),
                value: data.episodes,
                icon: Clapperboard,
                detail: t("including specials and multi-episode files"),
              },
              {
                label: t("Media files"),
                value: data.files,
                icon: FileVideo2,
                detail: t("movie and episode files"),
              },
            ].map((widget) => (
              <article class="stat-widget">
                <div>
                  <span>{widget.label}</span>
                  <widget.icon size={19} />
                </div>
                <strong>{count(widget.value)}</strong>
                <small>{widget.detail}</small>
              </article>
            ))}
          </div>
          <section class="quality-widgets" aria-label={t("Metadata quality")}>
            <article class="stat-widget">
              <span>{t("Incomplete metadata")}</span>
              <strong>{count(data.incomplete)}</strong>
              <small>
                {t(
                  "Resolution, codec, source, audio and language; HDR may be SDR or unknown.",
                )}
              </small>
            </article>
            <article class="stat-widget">
              <span>{t("Emby matching")}</span>
              <dl>
                {[
                  ["matched", t("Matched")],
                  ["missing", t("Not found")],
                  ["ambiguous", t("Ambiguous")],
                  ["missing_size", t("Size missing")],
                  ["no_tracks", t("Audio tracks missing")],
                  ["unconfigured", t("Without Emby enrichment")],
                  ["pending", t("Awaiting Emby enrichment")],
                ]
                  .sort(
                    (a, b) => (data.emby[b[0]] || 0) - (data.emby[a[0]] || 0),
                  )
                  .map(([key, label]) => (
                    <div>
                      <dt>{label}</dt>
                      <dd>{count(data.emby[key] || 0)}</dd>
                    </div>
                  ))}
              </dl>
            </article>
          </section>
          <div class="distribution-heading">
            <div>
              <h2>{t("Distribution")}</h2>
            </div>
            <div
              class="statistics-scopes"
              role="group"
              aria-label={t("Statistics scope")}
            >
              {[
                { value: "all", label: t("Entire library") },
                { value: "radarr", label: t("Movies") },
                { value: "sonarr", label: t("TV series") },
              ].map((item) => (
                <button
                  class={scope === item.value ? "active" : ""}
                  aria-pressed={scope === item.value}
                  onClick={() => setScope(item.value)}
                >
                  {item.label}
                </button>
              ))}
            </div>
          </div>
          <div class="distribution-grid">
            {["resolution", "codec", "source"].map((field) => (
              <DistributionChart
                field={field}
                onFilter={(value) => onFilter(field, value, scope)}
                buckets={data.distributions[scope]?.[field] || []}
              />
            ))}
          </div>
          <div class="dashboard-detail-grid">
            <LanguageDistribution
              buckets={data.distributions[scope]?.language || []}
              total={data.languageTotals[scope] || 0}
              onFilter={(value) => onFilter("language", value, scope)}
            />
            {settings && <StorageWidget settings={settings} scope={scope} />}
          </div>
        </>
      )}
    </section>
  );
}
function DistributionChart({
  field,
  buckets,
  onFilter,
}: {
  field: string;
  onFilter: (value: string) => void;
  buckets: { value: string; count: number }[];
}) {
  useI18n();
  const [expanded, setExpanded] = useState(false);
  const colors = useContext(TagColors);
  const total = buckets.reduce((sum, b) => sum + b.count, 0);
  const colored = [...buckets]
    .sort((a, b) => b.count - a.count || a.value.localeCompare(b.value))
    .map((bucket, i) => ({
      ...bucket,
      color:
        colors[field + ":" + (bucket.value || "__unknown__")] ||
        (!bucket.value ? "#87949c" : chartPalette[i % chartPalette.length]),
    }));
  return (
    <article class="distribution-card">
      <div class="distribution-title">
        <h3>{labels[field]}</h3>
        <span>
          {count(total)} {t("files")}
        </span>
      </div>
      {total ? (
        <>
          <div
            class="distribution-stack"
            role="img"
            aria-label={
              labels[field] +
              ": " +
              colored
                .map((b) => (b.value || t("Unknown")) + " " + count(b.count))
                .join(", ")
            }
          >
            {colored.map((b) => (
              <span
                style={{
                  width: (b.count / total) * 100 + "%",
                  backgroundColor: b.color,
                }}
                title={(b.value || t("Unknown")) + ": " + count(b.count)}
              />
            ))}
          </div>
          <ul class="distribution-list">
            {(expanded ? colored : colored.slice(0, 5)).map((b) => (
              <li>
                <button
                  type="button"
                  class="distribution-label chart-filter"
                  onClick={() => onFilter(b.value)}
                  aria-label={t("Filter {field}: {value}", {
                    field: labels[field],
                    value: b.value || t("Unknown"),
                  })}
                >
                  <span
                    class="chart-dot"
                    style={{ backgroundColor: b.color }}
                  />
                  <Tag
                    value={b.value}
                    kind={
                      field === "resolution" ? "resolution r" + b.value : field
                    }
                  />
                  <b>{count(b.count)}</b>
                  <span>
                    {new Intl.NumberFormat(getLocale(), {
                      maximumFractionDigits: 1,
                    }).format((b.count / total) * 100)}{" "}
                    %
                  </span>
                </button>
                <div class="distribution-track">
                  <span
                    style={{
                      width: (b.count / total) * 100 + "%",
                      backgroundColor: b.color,
                    }}
                  />
                </div>
              </li>
            ))}
          </ul>
          {colored.length > 5 && (
            <button
              class="button widget-expand"
              aria-expanded={expanded}
              onClick={() => setExpanded(!expanded)}
            >
              {t(expanded ? "Less" : "More")}
            </button>
          )}
        </>
      ) : (
        <div class="statistics-empty">
          <BarChart3 size={28} />
          <p>{t("No files in cache yet")}</p>
        </div>
      )}
    </article>
  );
}

function LanguageDistribution({
  buckets,
  total,
  onFilter,
}: {
  buckets: { value: string; count: number }[];
  total: number;
  onFilter: (value: string) => void;
}) {
  useI18n();
  const [expanded, setExpanded] = useState(false);
  const ranked = [...buckets].sort(
    (a, b) => b.count - a.count || a.value.localeCompare(b.value),
  );
  return (
    <article class="distribution-card language-distribution">
      <div class="distribution-title">
        <h3>{t("Language distribution")}</h3>
        <span>
          {count(total)} {t("titles")}
        </span>
      </div>
      {total ? (
        <ul class="distribution-list">
          {(expanded ? ranked : ranked.slice(0, 5)).map((bucket, i) => {
            const label = t(audioLanguageLabel(bucket.value, getLocale())),
              percentage = (bucket.count / total) * 100;
            return (
              <li>
                <button
                  class="distribution-label chart-filter language-chart-filter"
                  onClick={() => onFilter(bucket.value)}
                  aria-label={t("Filter language: {value}", { value: label })}
                >
                  <LanguageFlag value={bucket.value} focusable={false} />
                  <span>{label}</span>
                  <b>{count(bucket.count)}</b>
                  <span>
                    {new Intl.NumberFormat(getLocale(), {
                      maximumFractionDigits: 1,
                    }).format(percentage)}{" "}
                    %
                  </span>
                </button>
                <div class="distribution-track">
                  <span
                    style={{
                      width: percentage + "%",
                      backgroundColor: chartPalette[i % chartPalette.length],
                    }}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      ) : (
        <p class="quiet">{t("No files in cache yet")}</p>
      )}
      {ranked.length > 5 && (
        <button
          class="button widget-expand"
          aria-expanded={expanded}
          onClick={() => setExpanded(!expanded)}
        >
          {t(expanded ? "Less" : "More")}
        </button>
      )}
    </article>
  );
}
