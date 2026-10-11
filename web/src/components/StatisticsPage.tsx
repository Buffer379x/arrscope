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
import type { Statistics, Status } from "../types";

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
export function StatisticsPage({ status }: { status: Status }) {
  const [data, setData] = useState<Statistics | null>(null),
    [scope, setScope] = useState("all"),
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
  }, [status.sync?.radarr.lastSuccess, status.sync?.sonarr.lastSuccess]);
  return (
    <section class="statistics-page" aria-label="Dashboard">
      <div class="statistics-heading">
        <div>
          <h1>Dashboard</h1>
          <p>Deine gespeicherte Bibliothek auf einen Blick.</p>
        </div>
        <span class="quiet">
          <Database size={14} /> Aus dem lokalen Cache
        </span>
      </div>
      {error && (
        <p class="notice error" role="alert">
          {error}
        </p>
      )}
      {!data && !error && (
        <p class="quiet">
          <LoaderCircle class="spin" size={16} /> Dashboard wird geladen …
        </p>
      )}
      {data && (
        <>
          <div class="stat-widgets">
            {[
              {
                label: "Filme",
                value: data.movies,
                icon: Film,
                detail: "mit vorhandenen Dateien",
              },
              {
                label: "Serien",
                value: data.series,
                icon: Tv,
                detail: "mit vorhandenen Episoden",
              },
              {
                label: "Episoden",
                value: data.episodes,
                icon: Clapperboard,
                detail: "inklusive Specials und Mehrfachepisoden",
              },
              {
                label: "Mediendateien",
                value: data.files,
                icon: FileVideo2,
                detail: "Filme und Episodendateien",
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
          <div class="distribution-heading">
            <div>
              <h2>Medienmerkmale</h2>
              <p>Verteilung nach Dateien · unabhängig von Listenfiltern</p>
            </div>
            <div
              class="statistics-scopes"
              role="group"
              aria-label="Statistikbestand"
            >
              {[
                { value: "all", label: "Gesamtbestand" },
                { value: "radarr", label: "Filme" },
                { value: "sonarr", label: "Serien" },
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
          <section class="quality-widgets" aria-label="Metadatenqualität">
            <article class="stat-widget">
              <span>Metadaten unvollständig</span>
              <strong>{count(data.incomplete)}</strong>
              <small>
                Auflösung, Codec, Source, Audio und Sprache; HDR kann SDR oder
                unbekannt sein.
              </small>
            </article>
            <article class="stat-widget">
              <span>Emby-Zuordnung</span>
              <dl>
                {[
                  ["matched", "Zugeordnet"],
                  ["missing", "Nicht gefunden"],
                  ["ambiguous", "Mehrdeutig"],
                  ["missing_size", "Größe fehlt"],
                  ["no_tracks", "Tonspuren fehlen"],
                  ["unconfigured", "Ohne Emby-Ergänzung"],
                ].map(([key, label]) => (
                  <div>
                    <dt>{label}</dt>
                    <dd>{count(data.emby[key] || 0)}</dd>
                  </div>
                ))}
              </dl>
              <small>
                Eine Zuordnung kann trotzdem einzelne unbekannte Tonspurmerkmale
                enthalten.
              </small>
            </article>
          </section>
          <div class="distribution-grid">
            {["resolution", "codec", "source"].map((field) => (
              <DistributionChart
                field={field}
                buckets={data.distributions[scope]?.[field] || []}
              />
            ))}
          </div>
          <p class="statistics-footnote">
            Gezählt werden Filme und Serien mit Dateien im Cache. Jede
            eindeutige Episode zählt einmal; eine Datei kann mehrere Episoden
            enthalten. „Unbekannt“ bleibt in den Diagrammen sichtbar.
          </p>
        </>
      )}
    </section>
  );
}
function DistributionChart({
  field,
  buckets,
}: {
  field: string;
  buckets: { value: string; count: number }[];
}) {
  const colors = useContext(TagColors);
  const total = buckets.reduce((sum, b) => sum + b.count, 0);
  const colored = buckets.map((bucket, i) => ({
    ...bucket,
    color:
      colors[field + ":" + (bucket.value || "__unknown__")] ||
      (!bucket.value ? "#87949c" : chartPalette[i % chartPalette.length]),
  }));
  return (
    <article class="distribution-card">
      <div class="distribution-title">
        <h3>{labels[field]}</h3>
        <span>{count(total)} Dateien</span>
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
                .map((b) => (b.value || "Unbekannt") + " " + count(b.count))
                .join(", ")
            }
          >
            {colored.map((b) => (
              <span
                style={{
                  width: (b.count / total) * 100 + "%",
                  backgroundColor: b.color,
                }}
                title={(b.value || "Unbekannt") + ": " + count(b.count)}
              />
            ))}
          </div>
          <ul class="distribution-list">
            {colored.map((b) => (
              <li>
                <div class="distribution-label">
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
                    {new Intl.NumberFormat("de-DE", {
                      maximumFractionDigits: 1,
                    }).format((b.count / total) * 100)}{" "}
                    %
                  </span>
                </div>
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
        </>
      ) : (
        <div class="statistics-empty">
          <BarChart3 size={28} />
          <p>Noch keine Dateien im Cache</p>
        </div>
      )}
    </article>
  );
}
