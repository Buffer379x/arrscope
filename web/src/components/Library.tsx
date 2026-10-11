import { t, useI18n } from "../i18n";
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
import {
  dashboardService,
  makeDashboardFilter,
  type DashboardFilter,
} from "../dashboard-filter";
import { LanguageFlag, LanguageFlags } from "./LanguageFlag";
import { getLocale } from "../i18n";
import { labels } from "../ui";
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
  useI18n();
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
  const [dashboardFilter, setDashboardFilter] =
    useState<DashboardFilter | null>(null);
  const effectiveFilters = dashboardFilter
    ? { radarr: dashboardFilter.query, sonarr: dashboardFilter.query }
    : filters;
  const q = new URLSearchParams(effectiveFilters[service]);
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
    filters: effectiveFilters,
    page,
    movieSort,
    lastSuccess: sync?.lastSuccess,
    seasonMode: settings?.seasonMode,
    setResults,
    setBusy,
    setError,
  });
  const change = (key: string, values: FilterUpdate) => {
    const updateFilters = (current: Record<Service, string>) => {
      const n = new URLSearchParams(current[service]);
      const next =
        typeof values === "function" ? values(n.getAll(key)) : values;
      n.delete(key);
      next.forEach((v) => n.append(key, v));
      return { ...current, [service]: n.toString() };
    };
    if (dashboardFilter)
      setDashboardFilter((current) =>
        current
          ? {
              ...current,
              query: updateFilters({
                radarr: current.query,
                sonarr: current.query,
              })[service],
            }
          : current,
      );
    else setFilters(updateFilters);
    setPage(1);
  };
  const openDashboardFilter = (field: string, value: string, scope: string) => {
    setDashboardFilter(makeDashboardFilter(field, value, scope));
    setService(dashboardService(scope, service));
    setPage(1);
    setResults(emptyResults);
    setView("library");
  };
  const closeDashboardFilter = () => {
    if (!dashboardFilter) return;
    setDashboardFilter(null);
    setPage(1);
    setResults(emptyResults);
  };
  const applyPreset = (p: Preset) => {
    setDashboardFilter(null);
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
    if (dashboardFilter) {
      closeDashboardFilter();
      return;
    }
    setFilters({ ...filters, [service]: "" });
    setPage(1);
  };
  const runSync = async () => {
    try {
      await api("sync", "POST", { service });
      await refresh();
      setNotice(
        t("Sync queued. Services run one at a time; you can keep filtering."),
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
        query: effectiveFilters[service],
      });
      await loadPresets();
      setPresetIds({ ...presetIds, [service]: created.id });
      setSavePreset(false);
      setPresetName("");
      setNotice(t("Filter view saved."));
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const exportQuery = new URLSearchParams(effectiveFilters[service]);
  exportQuery.set("service", service);
  exportQuery.set("lang", getLocale());
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
      canonicalQuery(p.query) === canonicalQuery(effectiveFilters[service]),
  );
  const selectedPreset =
    matchingPresets.find((p) => p.id === presetIds[service]) ||
    matchingPresets[0];
  const deletePreset = async () => {
    if (!selectedPreset) return;
    try {
      await api("presets?id=" + selectedPreset.id, "DELETE");
      await loadPresets();
      setNotice(t("Preset deleted."));
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
            aria-label={t("Sort by {field}", { field: label })}
            title={t(
              current && movieSort.direction === "asc"
                ? "Sort {field} descending"
                : "Sort {field} ascending",
              { field: label },
            )}
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
          <a href="/" class="brand" aria-label={t("ArrScope home")}>
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
            <span class="brand-caption">{t("LIBRARY EXPLORER")}</span>
          </a>
          <nav class="tabs" aria-label={t("Main navigation")}>
            <button
              class={view === "statistics" ? "active" : ""}
              onClick={() => {
                closeDashboardFilter();
                setView("statistics");
              }}
            >
              <BarChart3 size={17} />
              {t("Dashboard")}
            </button>

            <button
              class={view === "library" && service === "radarr" ? "active" : ""}
              onClick={() => {
                closeDashboardFilter();
                setView("library");
                if (service === "radarr") return;
                setService("radarr");
                setPage(1);
                setResults(emptyResults);
              }}
            >
              <Film size={17} /> Radarr <span>{t("Movies")}</span>
            </button>
            <button
              class={view === "library" && service === "sonarr" ? "active" : ""}
              onClick={() => {
                closeDashboardFilter();
                setView("library");
                if (service === "sonarr") return;
                setService("sonarr");
                setPage(1);
                setResults(emptyResults);
              }}
            >
              <Tv size={17} /> Sonarr <span>{t("TV series")}</span>
            </button>
          </nav>
          <div class="header-actions">
            <span class="app-version">v{status.version}</span>
            <span class="local-status">
              <i />
              {t("Connected locally")}
            </span>
            <button
              class="icon-button"
              aria-label={theme === "dark" ? t("Light theme") : t("Dark theme")}
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            >
              {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </button>
            <button
              class="icon-button"
              aria-label={t("Settings")}
              onClick={() => setSettingsOpen(true)}
            >
              <SettingsIcon size={18} />
            </button>
            <button
              class="icon-button"
              aria-label={t("Sign out")}
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
              <Info size={14} />
              {t("Demo · sample data, no connection to your services")}
            </span>
            <span>ArrScope {status.version}</span>
          </div>
        )}
        <main class="main">
          {view === "statistics" ? (
            <StatisticsPage status={status} onFilter={openDashboardFilter} />
          ) : (
            <>
              <section class="library-intro">
                <h1>
                  {service === "radarr" ? t("Movie library") : t("TV library")}
                </h1>
                <span>
                  {count(sync?.count || 0)} {t("files in cache")}
                </span>
              </section>
              {error && (
                <div class="notice error" role="alert">
                  <AlertTriangle size={16} />
                  {error}
                  <button
                    class="icon-button"
                    aria-label={t("Dismiss error")}
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
                    aria-label={t("Dismiss notification")}
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
                      {t(
                        service === "radarr"
                          ? "Connect your movie library."
                          : "Connect your TV library.",
                      )}
                    </b>
                    <p>
                      {t(
                        "An address and API key are enough. ArrScope only reads existing data.",
                      )}
                    </p>
                  </div>
                  <button class="primary" onClick={() => setSettingsOpen(true)}>
                    {t("Set up service")}
                    <ArrowUpRight size={16} />
                  </button>
                </div>
              )}
              {dashboardFilter && (
                <div class="dashboard-filter-notice" role="status">
                  <span>
                    {t("Temporary dashboard filter")}:{" "}
                    {labels[dashboardFilter.field]} ·{" "}
                    {dashboardFilter.value || t("Unknown")}
                  </span>
                  {dashboardFilter.scope === "all" && (
                    <select
                      aria-label={t("Library for dashboard filter")}
                      value={service}
                      onChange={(event) => {
                        setService(event.currentTarget.value as Service);
                        setPage(1);
                        setResults(emptyResults);
                      }}
                    >
                      <option value="radarr">{t("Movies")}</option>
                      <option value="sonarr">{t("TV series")}</option>
                    </select>
                  )}
                  <button class="text-button" onClick={closeDashboardFilter}>
                    {t("Return to previous filters")}
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
                      {t(
                        service === "radarr" ? "Your movies" : "Your TV series",
                      )}{" "}
                      <span>{count(results.total)}</span>
                      {busy && <LoaderCircle size={16} class="spin" />}
                    </h2>
                    <p>
                      {active.length
                        ? t("Files matching your filters")
                        : t("Your complete library")}
                      {results.displayTotal > results.total &&
                        t(" · {count} additional files as season context", {
                          count: count(results.displayTotal - results.total),
                        })}
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
                          ? t("Return to saved columns")
                          : t("Show hidden columns in the saved order")
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
                        ? t("Default view")
                        : t("All columns")}
                    </button>
                    <a class="button" href={"/api/export?" + exportQuery}>
                      <Download size={15} />
                      {t("Export CSV")}
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
                                ? t("Title")
                                : t("Episode")
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
                                  aria-label={t(
                                    "Filename and metadata: {title}",
                                    { title: f.title },
                                  )}
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
                                    title={t("Filename and API disagree")}
                                    aria-label={t("Show conflicting metadata")}
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
                                  {t("Season context · not a filter match")}
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
                                        <LanguageFlag value={track.language} />
                                        <Tag value={track.codec} kind="audio" />
                                      </div>
                                    ))}
                                  </div>
                                ) : (
                                  <LanguageFlags value={f.language} />
                                )}
                                {(f.language || f.audio) && (
                                  <button
                                    class="icon-button"
                                    aria-label={t(
                                      "Audio track information: {title}",
                                      { title: f.title },
                                    )}
                                    title={
                                      f.audioTracks?.length
                                        ? t(
                                            "Audio tracks from Emby. Open file details.",
                                          )
                                        : t(
                                            "Language → codec mapping unavailable. Open file details.",
                                          )
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
                                      ? t("Specials")
                                      : t("Season {number}", {
                                          number: String(f.season).padStart(
                                            2,
                                            "0",
                                          ),
                                        })}
                                  </strong>
                                  <span class="season-note">
                                    {settings?.seasonMode === "season"
                                      ? t("Season context enabled")
                                      : t("Matching files")}
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
                          ? t("No matching files.")
                          : t("Your overview starts here.")}
                      </h3>
                      <p>
                        {active.length
                          ? t("Adjust or reset your filters.")
                          : t("Connect a service and start the first sync.")}
                      </p>
                      <button
                        class="button"
                        onClick={
                          active.length ? reset : () => setSettingsOpen(true)
                        }
                      >
                        {active.length
                          ? t("Reset filters")
                          : t("Open settings")}
                      </button>
                    </div>
                  )}
                </div>
                <div class="pagination">
                  <span>
                    {results.displayTotal
                      ? t("{from}–{to} of {total} files", {
                          from: count((page - 1) * 100 + 1),
                          to: count(Math.min(page * 100, results.displayTotal)),
                          total: count(results.displayTotal),
                        })
                      : t("0 files")}
                    {service === "sonarr" && t(" · by series and season")}
                  </span>
                  <div>
                    <button
                      class="icon-button"
                      aria-label={t("Previous page")}
                      disabled={page <= 1}
                      onClick={() => setPage(page - 1)}
                    >
                      <ArrowLeft size={16} />
                    </button>
                    <span>{t("Page {page} / {pages}", { page, pages })}</span>
                    <button
                      class="icon-button"
                      aria-label={t("Next page")}
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
                        ? t("Sync waiting for a slot …")
                        : t("Sync in progress …")
                      : sync?.error
                        ? t("Sync failed · {error}", { error: t(sync.error) })
                        : t("Last sync: {date}", {
                            date: date(sync?.lastSuccess),
                          })}
                  </span>
                  {sync?.error && sync.lastSuccess && (
                    <span>
                      {t("Cache: {date}", { date: date(sync.lastSuccess) })}
                    </span>
                  )}
                  <button
                    class="text-button"
                    disabled={!!sync?.running || status.demo || !configured}
                    onClick={runSync}
                  >
                    <RefreshCw size={13} class={sync?.running ? "spin" : ""} />{" "}
                    {t("Sync now")}
                  </button>
                </div>
                <span>
                  <ShieldCheck size={13} />
                  {t("Existing data only")}
                  <i /> ArrScope {status.version}
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
              setNotice(t("Settings saved."));
            }}
          />
        )}
        {savePreset && (
          <Modal
            title={t("Save filter view")}
            close={() => setSavePreset(false)}
          >
            <form onSubmit={doSavePreset}>
              <p class="quiet">
                {t(
                  service === "radarr"
                    ? "Save the current filter combination for movies."
                    : "Save the current filter combination for TV series.",
                )}
              </p>
              <label>
                {t("Name")}
                <input
                  autoFocus
                  maxLength={80}
                  required
                  value={presetName}
                  placeholder={t("For example: 1080p · x264")}
                  onInput={(e) => setPresetName(e.currentTarget.value)}
                />
              </label>
              <button class="primary">
                <Bookmark size={16} />
                {t("Save view")}
              </button>
            </form>
          </Modal>
        )}
      </div>
    </TagColors.Provider>
  );
}
