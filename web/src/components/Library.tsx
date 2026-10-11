import {
  AlertTriangle,
  ArrowDown,
  ArrowDownUp,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  ArrowUpRight,
  BarChart3,
  Bookmark,
  Check,
  Columns3,
  Database,
  Download,
  FileVideo2,
  Film,
  Info,
  Layers,
  LoaderCircle,
  LogOut,
  Moon,
  RefreshCw,
  Search,
  Settings as SettingsIcon,
  ShieldCheck,
  Sun,
  Tv,
  X,
} from "lucide-preact";
import type { JSX } from "preact";
import { useEffect, useState } from "preact/hooks";
import type {
  FilmSort,
  MediaFile,
  Preset,
  Results,
  Service,
  Settings,
  Status,
} from "../types";

import { api } from "../api";
import { useFacets, useLibraryQuery } from "../hooks/useLibraryQuery";
import {
  bytes,
  columnLabels,
  count,
  date,
  defaultColumnOrder,
  defaultColumns,
  emptyResults,
  FilterUpdate,
  Tag,
  TagColors,
} from "../ui";
import { FileDialog } from "./FileDialog";
import { FilterPanel } from "./FilterPanel";
import { Modal } from "./Modal";
import { SettingsDialog } from "./SettingsDialog";
import { StatisticsPage } from "./StatisticsPage";
const movieSortFields = [
  "title",
  "year",
  "resolution",
  "codec",
  "source",
  "size",
  "hdr",
  "bitDepth",
  "audio",
  "channels",
  "language",
];
function initialMovieSort(): FilmSort {
  try {
    const saved = JSON.parse(
      localStorage.getItem("arrscope-film-sort") || "null",
    );
    if (
      saved &&
      movieSortFields.includes(saved.field) &&
      (saved.direction === "asc" || saved.direction === "desc")
    )
      return saved;
  } catch {}
  return { field: "title", direction: "asc" };
}
export function Library({
  status,
  refresh,
}: {
  status: Status;
  refresh: () => Promise<void>;
}) {
  const [service, setService] = useState<Service>("radarr"),
    [view, setView] = useState<"library" | "statistics">("statistics"),
    [movieSort, setMovieSort] = useState<FilmSort>(initialMovieSort),
    [filters, setFilters] = useState<Record<Service, string>>({
      radarr: "",
      sonarr: "",
    }),
    [page, setPage] = useState(1),
    [results, setResults] = useState<Results>(emptyResults),
    [facets, setFacets] = useState<Record<string, string[]>>({}),
    [settings, setSettings] = useState<Settings | null>(null),
    [presets, setPresets] = useState<Preset[]>([]),
    [presetIds, setPresetIds] = useState<Record<Service, number | undefined>>({
      radarr: undefined,
      sonarr: undefined,
    }),
    [showAllColumns, setShowAllColumns] = useState<Record<Service, boolean>>({
      radarr: false,
      sonarr: false,
    }),
    [settingsOpen, setSettingsOpen] = useState(false),
    [collapsed, setCollapsed] = useState(
      localStorage.getItem("arrscope-filter-collapsed") === "true",
    ),
    [filename, setFilename] = useState<MediaFile | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [notice, setNotice] = useState(""),
    [theme, setTheme] = useState(
      localStorage.getItem("arrscope-theme") || "dark",
    ),
    [savePreset, setSavePreset] = useState(false),
    [presetName, setPresetName] = useState("");
  const q = new URLSearchParams(filters[service]);
  const sync = status.sync?.[service];
  const configured = settings?.[service].enabled;
  const visibleColumns = settings?.visibleColumns[service] || defaultColumns;
  const columnOrder = settings?.columnOrder[service] || defaultColumnOrder;
  const displayColumns = columnOrder.filter(
    (field) => showAllColumns[service] || visibleColumns.includes(field),
  );
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("arrscope-theme", theme);
  }, [theme]);
  useEffect(() => {
    localStorage.setItem("arrscope-filter-collapsed", String(collapsed));
  }, [collapsed]);
  useEffect(() => {
    localStorage.setItem("arrscope-film-sort", JSON.stringify(movieSort));
  }, [movieSort]);
  const loadSettings = async () => setSettings(await api<Settings>("settings"));
  const loadPresets = async () => setPresets(await api<Preset[]>("presets"));
  useEffect(() => {
    loadSettings().catch((e) => setError(e.message));
    loadPresets().catch(() => {});
  }, []);
  useFacets(
    view === "library",
    service,
    sync?.lastSuccess,
    setFacets,
    setError,
  );
  useLibraryQuery({
    view,
    service,
    filters,
    page,
    movieSort,
    lastSuccess: sync?.lastSuccess,
    seasonMode: settings?.seasonMode,
    setResults,
    setBusy,
    setError,
  });
  const change = (key: string, values: FilterUpdate) => {
    setFilters((current) => {
      const n = new URLSearchParams(current[service]);
      const next =
        typeof values === "function" ? values(n.getAll(key)) : values;
      n.delete(key);
      next.forEach((v) => n.append(key, v));
      return { ...current, [service]: n.toString() };
    });
    setPage(1);
  };
  const applyPreset = (p: Preset) => {
    setPresetIds({ ...presetIds, [p.service]: p.id });
    setService(p.service);
    const n = new URLSearchParams(p.query);
    n.delete("service");
    n.delete("sort");
    n.delete("dir");
    setView("library");
    setFilters({ ...filters, [p.service]: n.toString() });
    setPage(1);
  };
  const reset = () => {
    setFilters({ ...filters, [service]: "" });
    setPage(1);
  };
  const runSync = async () => {
    try {
      await api("sync", "POST", { service });
      await refresh();
      setNotice(
        "Synchronisierung vorgemerkt. Dienste werden nacheinander verarbeitet; du kannst weiter filtern.",
      );
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const doSavePreset = async (e: Event) => {
    e.preventDefault();
    try {
      const created = await api<Preset>("presets", "POST", {
        name: presetName,
        service,
        query: filters[service],
      });
      await loadPresets();
      setPresetIds({ ...presetIds, [service]: created.id });
      setSavePreset(false);
      setPresetName("");
      setNotice("Filteransicht gespeichert.");
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const exportQuery = new URLSearchParams(filters[service]);
  exportQuery.set("service", service);
  if (service === "radarr") {
    exportQuery.set("sort", movieSort.field);
    exportQuery.set("dir", movieSort.direction);
  }
  const active = [...q.entries()].filter(
    ([k]) => k !== "sort" && k !== "dir" && !k.endsWith("Op"),
  );
  const canonicalQuery = (query: string) => {
    const params = new URLSearchParams(query);
    params.delete("service");
    params.delete("page");
    params.delete("sort");
    params.delete("dir");
    return JSON.stringify(
      [...params.entries()].sort(
        ([a, b], [c, d]) => a.localeCompare(c) || b.localeCompare(d),
      ),
    );
  };
  const matchingPresets = presets.filter(
    (p) =>
      p.service === service &&
      canonicalQuery(p.query) === canonicalQuery(filters[service]),
  );
  const selectedPreset =
    matchingPresets.find((p) => p.id === presetIds[service]) ||
    matchingPresets[0];
  const deletePreset = async () => {
    if (!selectedPreset) return;
    try {
      await api("presets?id=" + selectedPreset.id, "DELETE");
      await loadPresets();
      setNotice("Preset gelöscht.");
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const sortColumn = (field: string) => {
    setMovieSort({
      field,
      direction:
        movieSort.field === field && movieSort.direction === "asc"
          ? "desc"
          : "asc",
    });
    setPage(1);
  };
  const columnHeader = (field: string, label: string, titleCell = false) => {
    const current = movieSort.field === field;
    return (
      <th
        key={field}
        scope="col"
        class={titleCell ? "title-cell" : undefined}
        aria-sort={
          service === "radarr"
            ? current
              ? movieSort.direction === "asc"
                ? "ascending"
                : "descending"
              : "none"
            : undefined
        }
      >
        {service === "radarr" ? (
          <button
            class={"column-sort" + (current ? " current" : "")}
            onClick={() => sortColumn(field)}
            aria-label={label + " sortieren"}
            title={
              "Nach " +
              label +
              " " +
              (current && movieSort.direction === "asc"
                ? "absteigend"
                : "aufsteigend") +
              " sortieren"
            }
          >
            {label}
            {current ? (
              movieSort.direction === "asc" ? (
                <ArrowUp size={13} />
              ) : (
                <ArrowDown size={13} />
              )
            ) : (
              <ArrowDownUp size={13} />
            )}
          </button>
        ) : (
          label
        )}
      </th>
    );
  };
  const pages = Math.max(1, Math.ceil(results.displayTotal / 100));
  useEffect(() => {
    if (!busy && results.page === page && page > pages) setPage(pages);
  }, [busy, results.page, page, pages]);
  return (
    <TagColors.Provider value={settings?.tagColors || {}}>
      <div class="shell">
        <header class="header">
          <a href="/" class="brand" aria-label="ArrScope Startseite">
            <img
              src="/icon-64.png"
              srcSet="/icon-64.png 1x, /icon-256.png 2x"
              width="34"
              height="34"
              alt=""
            />
            <b>
              ArrScope<span> / </span>
            </b>
            <span class="brand-caption">LIBRARY EXPLORER</span>
          </a>
          <nav class="tabs" aria-label="Hauptmenü">
            <button
              class={view === "statistics" ? "active" : ""}
              onClick={() => setView("statistics")}
            >
              <BarChart3 size={17} /> Dashboard
            </button>

            <button
              class={view === "library" && service === "radarr" ? "active" : ""}
              onClick={() => {
                setView("library");
                if (service === "radarr") return;
                setService("radarr");
                setPage(1);
                setResults(emptyResults);
              }}
            >
              <Film size={17} /> Radarr <span>Filme</span>
            </button>
            <button
              class={view === "library" && service === "sonarr" ? "active" : ""}
              onClick={() => {
                setView("library");
                if (service === "sonarr") return;
                setService("sonarr");
                setPage(1);
                setResults(emptyResults);
              }}
            >
              <Tv size={17} /> Sonarr <span>Serien</span>
            </button>
          </nav>
          <div class="header-actions">
            <span class="local-status">
              <i /> Lokal verbunden
            </span>
            <button
              class="icon-button"
              aria-label={theme === "dark" ? "Helles Design" : "Dunkles Design"}
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            >
              {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </button>
            <button
              class="icon-button"
              aria-label="Einstellungen"
              onClick={() => setSettingsOpen(true)}
            >
              <SettingsIcon size={18} />
            </button>
            <button
              class="icon-button"
              aria-label="Abmelden"
              onClick={async () => {
                await api("logout", "POST");
                await refresh();
              }}
            >
              <LogOut size={18} />
            </button>
          </div>
        </header>
        {status.demo && (
          <div class="demo-bar">
            <span>
              <Info size={14} /> Demo · Beispieldaten, keine Verbindung zu
              deinen Diensten
            </span>
            <span>ArrScope {status.version}</span>
          </div>
        )}
        <main class="main">
          {view === "statistics" ? (
            <StatisticsPage status={status} />
          ) : (
            <>
              <section class="library-intro">
                <h1>
                  {service === "radarr" ? "Filmbibliothek" : "Serienbibliothek"}
                </h1>
                <span>{count(sync?.count || 0)} Dateien im Cache</span>
              </section>
              {error && (
                <div class="notice error" role="alert">
                  <AlertTriangle size={16} />
                  {error}
                  <button
                    class="icon-button"
                    aria-label="Fehlermeldung schließen"
                    onClick={() => setError("")}
                  >
                    <X size={16} />
                  </button>
                </div>
              )}
              {notice && (
                <div class="notice success" role="status">
                  <Check size={16} />
                  {notice}
                  <button
                    class="icon-button"
                    aria-label="Meldung schließen"
                    onClick={() => setNotice("")}
                  >
                    <X size={16} />
                  </button>
                </div>
              )}
              {!status.demo && settings && !configured && (
                <div class="connect-card">
                  <Database size={24} />
                  <div>
                    <b>
                      Verbinde deine{" "}
                      {service === "radarr" ? "Film-" : "Serien-"}
                      Bibliothek.
                    </b>
                    <p>
                      Adresse und API-Key genügen. ArrScope liest ausschließlich
                      vorhandene Daten.
                    </p>
                  </div>
                  <button class="primary" onClick={() => setSettingsOpen(true)}>
                    Dienst einrichten <ArrowUpRight size={16} />
                  </button>
                </div>
              )}
              <FilterPanel
                q={q}
                collapsed={collapsed}
                setCollapsed={setCollapsed}
                active={active}
                service={service}
                facets={facets}
                presets={presets}
                selectedPreset={selectedPreset}
                applyPreset={applyPreset}
                reset={reset}
                setSavePreset={setSavePreset}
                deletePreset={deletePreset}
                change={change}
              />
              <section class="library-section">
                <div class="results-heading">
                  <div>
                    <h2>
                      Deine {service === "radarr" ? "Filme" : "Serien"}{" "}
                      <span>{count(results.total)}</span>
                      {busy && <LoaderCircle size={16} class="spin" />}
                    </h2>
                    <p>
                      {active.length
                        ? "Passende Dateien für deine Filter"
                        : "Der vollständige Bestand deiner Bibliothek"}
                      {results.displayTotal > results.total &&
                        ` · ${count(results.displayTotal - results.total)} weitere Dateien als Staffelkontext`}
                    </p>
                  </div>
                  <div class="results-tools">
                    <button
                      class="button"
                      aria-pressed={showAllColumns[service]}
                      disabled={
                        visibleColumns.length === columnOrder.length &&
                        !showAllColumns[service]
                      }
                      title={
                        showAllColumns[service]
                          ? "Zur gespeicherten Spaltenauswahl zurückkehren"
                          : "Ausgeblendete Spalten in der gespeicherten Reihenfolge anzeigen"
                      }
                      onClick={() =>
                        setShowAllColumns({
                          ...showAllColumns,
                          [service]: !showAllColumns[service],
                        })
                      }
                    >
                      <Columns3 size={15} />{" "}
                      {showAllColumns[service]
                        ? "Standardansicht"
                        : "Alle Spalten"}
                    </button>
                    <a class="button" href={"/api/export?" + exportQuery}>
                      <Download size={15} /> CSV exportieren
                    </a>
                  </div>
                </div>
                <div
                  class={"table-wrap" + (busy ? " busy" : "")}
                  aria-busy={busy}
                >
                  <table>
                    <thead>
                      <tr>
                        {displayColumns.map((field) =>
                          columnHeader(
                            field,
                            field === "title"
                              ? service === "radarr"
                                ? "Titel"
                                : "Episode"
                              : columnLabels[field],
                            field === "title",
                          ),
                        )}
                      </tr>
                    </thead>
                    <tbody>
                      {results.files.map((f, i) => {
                        const prev = results.files[i - 1];
                        const group =
                          service === "sonarr" &&
                          (!prev ||
                            prev.mediaId !== f.mediaId ||
                            prev.season !== f.season);
                        const cells: Record<string, JSX.Element> = {
                          title: (
                            <td key="title" class="title-cell">
                              <div class="title-line">
                                <span class="row-icon">
                                  {service === "radarr" ? (
                                    <Film size={15} />
                                  ) : (
                                    <FileVideo2 size={15} />
                                  )}
                                </span>
                                <b>{f.title}</b>
                                <button
                                  class="file-button"
                                  aria-label={
                                    "Dateiname und Metadaten: " + f.title
                                  }
                                  aria-describedby={"filename-" + f.id}
                                  onClick={() => setFilename(f)}
                                >
                                  <FileVideo2 size={14} />
                                  <span
                                    class="filename-tooltip"
                                    role="tooltip"
                                    id={"filename-" + f.id}
                                  >
                                    {f.filename}
                                  </span>
                                </button>
                                {f.conflict && (
                                  <button
                                    class="conflict-button"
                                    title="Dateiname und API widersprechen sich"
                                    aria-label="Widersprüchliche Metadaten anzeigen"
                                    onClick={() => setFilename(f)}
                                  >
                                    <AlertTriangle size={14} />
                                  </button>
                                )}
                              </div>
                              {service === "sonarr" && (
                                <small class="episode-filename">
                                  {f.filename}
                                </small>
                              )}
                              {!f.matched && (
                                <small class="context-label">
                                  Staffelkontext · kein Filtertreffer
                                </small>
                              )}
                            </td>
                          ),
                          year: (
                            <td key="year" class="year-cell">
                              {f.years.length ? f.years.join(" / ") : "—"}
                            </td>
                          ),
                          resolution: (
                            <td key="resolution">
                              <Tag
                                value={f.resolution}
                                kind={"resolution r" + f.resolution}
                              />
                            </td>
                          ),
                          codec: (
                            <td key="codec">
                              <Tag
                                value={f.codec}
                                kind="codec"
                                title={f.family}
                              />
                            </td>
                          ),
                          source: (
                            <td key="source">
                              <div class="source-tags">
                                <Tag value={f.source} />
                                {f.processing && (
                                  <Tag value={f.processing} kind="processing" />
                                )}
                              </div>
                            </td>
                          ),
                          size: (
                            <td key="size" class="nowrap">
                              {bytes(f.size)}
                            </td>
                          ),
                          hdr: (
                            <td key="hdr">
                              <div class="source-tags">
                                {(f.hdr ? f.hdr.split(" + ") : [""]).map(
                                  (value) => (
                                    <Tag value={value} kind="hdr" />
                                  ),
                                )}
                              </div>
                            </td>
                          ),
                          bitDepth: <td key="bitDepth">{f.bitDepth || "—"}</td>,
                          audio: (
                            <td key="audio">
                              <div class="source-tags">
                                {(f.audio ? f.audio.split(" + ") : [""]).map(
                                  (value) => (
                                    <Tag
                                      key={value}
                                      value={value}
                                      kind="audio"
                                    />
                                  ),
                                )}
                              </div>
                            </td>
                          ),
                          channels: <td key="channels">{f.channels || "—"}</td>,
                          language: (
                            <td key="language">
                              <div class="language-details">
                                {f.audioTracks?.length ? (
                                  <div class="track-language-list">
                                    {f.audioTracks.map((track) => (
                                      <div key={track.index}>
                                        <span>
                                          {track.language || "Unbekannt"}
                                        </span>
                                        <Tag value={track.codec} kind="audio" />
                                      </div>
                                    ))}
                                  </div>
                                ) : (
                                  <span>{f.language || "—"}</span>
                                )}
                                {(f.language || f.audio) && (
                                  <button
                                    class="icon-button"
                                    aria-label={
                                      "Tonspurinformationen: " + f.title
                                    }
                                    title={
                                      f.audioTracks?.length
                                        ? "Tonspuren aus Emby. Dateidetails öffnen."
                                        : "Zuordnung Sprache → Codec nicht verfügbar. Dateidetails öffnen."
                                    }
                                    onClick={() => setFilename(f)}
                                  >
                                    <Info size={14} />
                                  </button>
                                )}
                              </div>
                            </td>
                          ),
                        };
                        return (
                          <>
                            {group && (
                              <tr class="season-row">
                                <td colSpan={displayColumns.length}>
                                  <Layers size={15} />
                                  <strong>
                                    {f.series} <span>—</span>{" "}
                                    {f.season === 0
                                      ? "Specials"
                                      : "Season " +
                                        String(f.season).padStart(2, "0")}
                                  </strong>
                                  <span class="season-note">
                                    {settings?.seasonMode === "season"
                                      ? "Staffelkontext aktiviert"
                                      : "Passende Dateien"}
                                  </span>
                                </td>
                              </tr>
                            )}
                            <tr
                              key={f.service + f.id}
                              class={!f.matched ? "context-row" : ""}
                            >
                              {displayColumns.map((field) => cells[field])}
                            </tr>
                          </>
                        );
                      })}
                    </tbody>
                  </table>
                  {!results.files.length && !busy && (
                    <div class="empty-state">
                      <Search size={32} />
                      <h3>
                        {active.length
                          ? "Keine passenden Dateien."
                          : "Hier entsteht dein Überblick."}
                      </h3>
                      <p>
                        {active.length
                          ? "Passe deine Filter an oder setze sie zurück."
                          : "Verbinde einen Dienst und starte den ersten Sync."}
                      </p>
                      <button
                        class="button"
                        onClick={
                          active.length ? reset : () => setSettingsOpen(true)
                        }
                      >
                        {active.length
                          ? "Filter zurücksetzen"
                          : "Einstellungen öffnen"}
                      </button>
                    </div>
                  )}
                </div>
                <div class="pagination">
                  <span>
                    {results.displayTotal
                      ? `${count((page - 1) * 100 + 1)}–${count(Math.min(page * 100, results.displayTotal))} von ${count(results.displayTotal)} Dateien`
                      : "0 Dateien"}
                    {service === "sonarr" && " · nach Serie und Staffel"}
                  </span>
                  <div>
                    <button
                      class="icon-button"
                      aria-label="Vorherige Seite"
                      disabled={page <= 1}
                      onClick={() => setPage(page - 1)}
                    >
                      <ArrowLeft size={16} />
                    </button>
                    <span>
                      Seite {page} / {pages}
                    </span>
                    <button
                      class="icon-button"
                      aria-label="Nächste Seite"
                      disabled={page >= pages}
                      onClick={() => setPage(page + 1)}
                    >
                      <ArrowRight size={16} />
                    </button>
                  </div>
                </div>
              </section>
              <footer class="footer">
                <div>
                  <span
                    class={
                      "sync-dot " +
                      (sync?.error ? "failed" : sync?.running ? "running" : "")
                    }
                  />
                  <span>
                    {sync?.running
                      ? sync.queued
                        ? "Sync wartet auf freien Platz …"
                        : "Synchronisierung läuft …"
                      : sync?.error
                        ? "Sync fehlgeschlagen · " + sync.error
                        : "Letzter Sync: " + date(sync?.lastSuccess)}
                  </span>
                  {sync?.error && sync.lastSuccess && (
                    <span>Cache: {date(sync.lastSuccess)}</span>
                  )}
                  <button
                    class="text-button"
                    disabled={!!sync?.running || status.demo || !configured}
                    onClick={runSync}
                  >
                    <RefreshCw size={13} class={sync?.running ? "spin" : ""} />{" "}
                    Jetzt synchronisieren
                  </button>
                </div>
                <span>
                  <ShieldCheck size={13} /> Nur vorhandene Daten <i /> ArrScope{" "}
                  {status.version}
                </span>
              </footer>
            </>
          )}
        </main>
        {filename && (
          <FileDialog file={filename} close={() => setFilename(null)} />
        )}{" "}
        {settingsOpen && settings && (
          <SettingsDialog
            settings={settings}
            demo={status.demo}
            close={() => setSettingsOpen(false)}
            saved={async () => {
              await loadSettings();
              await refresh();
              setNotice("Einstellungen gespeichert.");
            }}
          />
        )}
        {savePreset && (
          <Modal
            title="Filteransicht speichern"
            close={() => setSavePreset(false)}
          >
            <form onSubmit={doSavePreset}>
              <p class="quiet">
                Speichert die aktuelle Filterkombination für{" "}
                {service === "radarr" ? "Filme" : "Serien"}.
              </p>
              <label>
                Name
                <input
                  autoFocus
                  maxLength={80}
                  required
                  value={presetName}
                  placeholder="Zum Beispiel: 1080p · x264"
                  onInput={(e) => setPresetName(e.currentTarget.value)}
                />
              </label>
              <button class="primary">
                <Bookmark size={16} /> Ansicht speichern
              </button>
            </form>
          </Modal>
        )}
      </div>
    </TagColors.Provider>
  );
}
