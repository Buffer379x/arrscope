import { RowInfo } from "./RowInfo";
import { applyInterfaceTheme } from "../themes";
import { TagInformation, TagVisibility } from "../ui";
import { confirmDeletion } from "../confirmation";
import { notify } from "../notifications";
import { CustomBulk } from "./CustomBulk";
import type { CustomTag } from "../types";
import { t, useI18n } from "../i18n";
import {
  Tag as TagIcon,
  RotateCcw,
  ArrowDown,
  ArrowDownUp,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  ArrowUpRight,
  BarChart3,
  Bookmark,
  Columns3,
  Database,
  Download,
  Film,
  Info,
  Layers,
  LoaderCircle,
  LogOut,
  Moon,
  Search,
  Settings as SettingsIcon,
  Sun,
  Tv,
} from "lucide-preact";
import type { JSX } from "preact";
import { useEffect, useRef, useState } from "preact/hooks";
import type {
  FilmSort,
  MediaFile,
  Preset,
  Results,
  Service,
  Settings,
  Status,
} from "../types";

import { canonicalPresetQuery } from "../preset-query";
import { api, ApiError } from "../api";
import { dashboardService, makeDashboardFilter } from "../dashboard-filter";
import { LanguageFlag, LanguageFlags } from "./LanguageFlag";
import { getLocale } from "../i18n";
import { useFacets, useLibraryQuery } from "../hooks/useLibraryQuery";
import {
  bytes,
  columnLabels,
  count,
  defaultColumnOrder,
  defaultColumns,
  emptyResults,
  FilterUpdate,
  Tag,
  CustomTagNames,
  TagColors,
  TagTextColors,
} from "../ui";
import { RecognitionPage } from "./RecognitionPage";
import { AudioCell } from "./AudioCell";
import { defaultWarningColors } from "../ui";
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
  "custom",
];
function initialMovieSort(): FilmSort {
  try {
    const saved = JSON.parse(
      localStorage.getItem("scopearr-film-sort") || "null",
    );
    if (saved?.field === "language") saved.field = "audio";
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
  connectionError,
}: {
  status: Status;
  connectionError?: string;
  refresh: () => Promise<void>;
}) {
  useI18n();
  const [customTags, setCustomTags] = useState<CustomTag[]>([]);
  const [customRevision, setCustomRevision] = useState(0);
  const reviewNotified = useRef<Record<string, boolean>>({});
  useEffect(() => {
    for (const service of ["radarr", "sonarr"] as Service[]) {
      const required = !!status.tagSync?.[service]?.approvalRequired;
      if (required && !reviewNotified.current[service])
        notify(t("Tag removals need review in Settings → System."), "warning");
      reviewNotified.current[service] = required;
    }
  }, [status.tagSync]);
  const [tagVisibility, setTagVisibility] = useState<Record<
    string,
    string[]
  > | null>(null);
  useEffect(() => {
    api<Record<string, string[]>>("display-tags")
      .then(setTagVisibility)
      .catch(() => {});
  }, [
    status.recognitionRevision,
    status.sync?.radarr?.lastSuccess,
    status.sync?.sonarr?.lastSuccess,
    customRevision,
  ]);
  const [customOpen, setCustomOpen] = useState(false);
  const [singleCustom, setSingleCustom] = useState<number | null>(null);
  const loadCustom = async () => {
    const tags = await api<CustomTag[]>("custom-tags");
    setCustomTags(tags);
    setCustomRevision((value) => value + 1);
    const ids = new Set(tags.map((tag) => String(tag.id)));
    setFilters(
      (current) =>
        Object.fromEntries(
          Object.entries(current).map(([service, query]) => {
            const q = new URLSearchParams(query);
            const selected = q.getAll("custom");
            q.delete("custom");
            selected
              .filter((id) => id === "__unknown__" || ids.has(id))
              .forEach((id) => q.append("custom", id));
            if (!q.has("custom")) q.delete("customOp");
            return [service, q.toString()];
          }),
        ) as Record<Service, string>,
    );
  };
  const [service, setService] = useState<Service>("radarr"),
    [view, setView] = useState<"library" | "statistics" | "recognition">(
      "statistics",
    ),
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
    [rulesDirty, setRulesDirty] = useState(false),
    [settingsDirty, setSettingsDirty] = useState(false),
    [settingsOpen, setSettingsOpen] = useState(false),
    [settingsSection, setSettingsSection] = useState("connections"),
    [collapsed, setCollapsed] = useState(
      localStorage.getItem("scopearr-filter-collapsed") === "true",
    ),
    [filename, setFilename] = useState<MediaFile | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [notice, setNotice] = useState(""),
    [theme, setTheme] = useState(
      localStorage.getItem("scopearr-theme") || "dark",
    ),
    [savePreset, setSavePreset] = useState(false),
    [presetName, setPresetName] = useState(""),
    [presetError, setPresetError] = useState("");
  useEffect(() => {
    if (notice) {
      notify(notice);
      setNotice("");
    }
  }, [notice]);
  const libraryScroll = useRef(0);
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
    localStorage.setItem("scopearr-theme", theme);
  }, [theme]);
  useEffect(() => {
    applyInterfaceTheme(
      document.documentElement,
      settings?.interfaceTheme,
      theme,
    );
  }, [settings?.interfaceTheme, theme]);
  useEffect(() => {
    localStorage.setItem("scopearr-filter-collapsed", String(collapsed));
  }, [collapsed]);
  useEffect(() => {
    localStorage.setItem("scopearr-film-sort", JSON.stringify(movieSort));
  }, [movieSort]);
  const loadSettings = async () => setSettings(await api<Settings>("settings"));
  const loadPresets = async () => setPresets(await api<Preset[]>("presets"));
  useEffect(() => {
    loadCustom().catch((e) => setError(e.message));
    loadSettings().catch((e) => setError(e.message));
    loadPresets().catch(() => {});
  }, []);
  useFacets(
    view === "library" && !settingsOpen,
    service,
    `${sync?.lastSuccess}:${status.recognitionRevision}:${customRevision}`,
    setFacets,
    setError,
  );
  useLibraryQuery({
    view: settingsOpen ? "settings" : view,
    service,
    filters,
    page,
    movieSort,
    lastSuccess: `${sync?.lastSuccess}:${status.recognitionRevision}:${customRevision}`,
    seasonMode: settings?.seasonMode,
    setResults,
    setBusy,
    setError,
  });
  const openSettings = (section = "connections") => {
    if (!settingsOpen) libraryScroll.current = window.scrollY;
    setSettingsSection(section);
    setSettingsOpen(true);
  };
  const canLeaveRules = () => {
    if (
      settingsOpen &&
      settingsDirty &&
      !confirm(t("Discard unsaved settings changes?"))
    )
      return false;
    if (
      view === "recognition" &&
      rulesDirty &&
      !confirm(t("Discard unsaved rule changes?"))
    )
      return false;
    setSettingsOpen(false);
    return true;
  };
  const change = (key: string, values: FilterUpdate) => {
    const updateFilters = (current: Record<Service, string>) => {
      const n = new URLSearchParams(current[service]);
      const next =
        typeof values === "function" ? values(n.getAll(key)) : values;
      n.delete(key);
      next.forEach((v) => n.append(key, v));
      return { ...current, [service]: n.toString() };
    };
    setFilters(updateFilters);
    setPage(1);
  };
  const openDashboardFilter = (field: string, value: string, scope: string) => {
    const target = dashboardService(scope, service);
    setFilters((current) => ({
      ...current,
      [target]: makeDashboardFilter(field, value, scope).query,
    }));
    setPresetIds((current) => ({ ...current, [target]: undefined }));
    setService(target);
    setCollapsed(false);
    setPage(1);
    setResults(emptyResults);
    setView("library");
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
  const doSavePreset = async (e: Event) => {
    e.preventDefault();
    try {
      const duplicate = presets.find(
        (p) =>
          p.service === service &&
          canonicalPresetQuery(p.query) ===
            canonicalPresetQuery(filters[service]),
      );
      if (duplicate) {
        notify(
          t("These filters are already saved as {name}.", {
            name: duplicate.name,
          }),
          "warning",
        );
        setPresetError(
          t("These filters are already saved as {name}.", {
            name: duplicate.name,
          }),
        );
        return;
      }
      const created = await api<Preset>("presets", "POST", {
        name: presetName,
        service,
        query: filters[service],
      });
      await loadPresets();
      setPresetIds({ ...presetIds, [service]: created.id });
      setSavePreset(false);
      setPresetName("");
      setNotice(t("Filter view saved."));
    } catch (e) {
      setPresetError(
        e instanceof ApiError && e.data.preset
          ? t("These filters are already saved as {name}.", {
              name: e.data.preset.name,
            })
          : (e as Error).message,
      );
    }
  };
  const exportQuery = new URLSearchParams(filters[service]);
  exportQuery.set("service", service);
  exportQuery.set("lang", getLocale());
  if (service === "radarr") {
    exportQuery.set("sort", movieSort.field);
    exportQuery.set("dir", movieSort.direction);
  }
  const active = [...q.entries()].filter(
    ([k]) => k !== "sort" && k !== "dir" && !k.endsWith("Op"),
  );
  const matchingPresets = presets.filter(
    (p) =>
      p.service === service &&
      canonicalPresetQuery(p.query) === canonicalPresetQuery(filters[service]),
  );
  const selectedPreset =
    matchingPresets.find((p) => p.id === presetIds[service]) ||
    matchingPresets[0];
  const deletePreset = async () => {
    if (
      !selectedPreset ||
      !(await confirmDeletion(
        t("Delete preset {name}?", { name: selectedPreset.name }),
      ))
    )
      return;
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
          service === "radarr" && field !== "info"
            ? current
              ? movieSort.direction === "asc"
                ? "ascending"
                : "descending"
              : "none"
            : undefined
        }
      >
        {service === "radarr" && field !== "info" ? (
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
    <CustomTagNames.Provider
      value={Object.fromEntries(
        customTags.map((tag) => [String(tag.id), tag.name]),
      )}
    >
      <TagVisibility.Provider value={tagVisibility}>
        <TagColors.Provider value={settings?.tagColors || {}}>
          <TagTextColors.Provider value={settings?.tagTextColors || {}}>
            <div class="shell">
              <header class="header">
                <a href="/" class="brand" aria-label={t("Scopearr home")}>
                  <img
                    src="/icon-64.png"
                    srcSet="/icon-64.png 1x, /icon-256.png 2x"
                    width="34"
                    height="34"
                    alt=""
                  />
                  <b>
                    Scopearr<span> / </span>
                  </b>
                  <span class="brand-caption">{t("LIBRARY EXPLORER")}</span>
                </a>
                <nav class="tabs" aria-label={t("Main navigation")}>
                  <button
                    class={view === "statistics" ? "active" : ""}
                    onClick={() => {
                      if (!canLeaveRules()) return;
                      setView("statistics");
                    }}
                  >
                    <BarChart3 size={17} />
                    {t("Dashboard")}
                  </button>

                  <button
                    class={
                      view === "library" && service === "radarr" ? "active" : ""
                    }
                    onClick={() => {
                      if (!canLeaveRules()) return;
                      setView("library");
                      if (service === "radarr") return;
                      setService("radarr");
                      setPage(1);
                      setResults(emptyResults);
                    }}
                  >
                    <Film size={17} /> {t("Movies")}
                  </button>
                  <button
                    class={
                      view === "library" && service === "sonarr" ? "active" : ""
                    }
                    onClick={() => {
                      if (!canLeaveRules()) return;
                      setView("library");
                      if (service === "sonarr") return;
                      setService("sonarr");
                      setPage(1);
                      setResults(emptyResults);
                    }}
                  >
                    <Tv size={17} /> {t("TV series")}
                  </button>
                </nav>
                <div class="header-actions">
                  <button
                    class="icon-button"
                    aria-label={
                      theme === "dark" ? t("Light theme") : t("Dark theme")
                    }
                    onClick={() =>
                      setTheme(theme === "dark" ? "light" : "dark")
                    }
                  >
                    {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
                  </button>
                  <button
                    class="icon-button"
                    aria-label={t("Settings")}
                    onClick={() => {
                      if (canLeaveRules()) openSettings();
                    }}
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
                </div>
              )}
              <main class="main">
                {settingsOpen && settings ? (
                  <SettingsDialog
                    customTags={customTags}
                    customChanged={async () => {
                      await loadCustom();
                      await loadSettings();
                      await loadPresets();
                      await refresh();
                    }}
                    settings={settings}
                    initialSection={settingsSection}
                    onDirty={setSettingsDirty}
                    refresh={refresh}
                    status={status}
                    connectionError={connectionError}
                    demo={status.demo}
                    saved={async () => {
                      await loadSettings();
                      await refresh();
                    }}
                  />
                ) : view === "recognition" ? (
                  <RecognitionPage onDirty={setRulesDirty} saved={refresh} />
                ) : view === "statistics" ? (
                  <StatisticsPage
                    status={status}
                    onFilter={openDashboardFilter}
                  />
                ) : (
                  <>
                    <section class="library-intro">
                      <h1>
                        {service === "radarr"
                          ? t("Movie library")
                          : t("TV library")}
                      </h1>
                      <span>
                        {count(sync?.count || 0)} {t("files in cache")}
                      </span>
                    </section>
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
                              "An address and API key are enough. Scopearr only reads existing data.",
                            )}
                          </p>
                        </div>
                        <button
                          class="primary"
                          onClick={() => {
                            if (canLeaveRules()) openSettings();
                          }}
                        >
                          {t("Set up service")}
                          <ArrowUpRight size={16} />
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
                      setSavePreset={(value) => {
                        setPresetError("");
                        setSavePreset(value);
                      }}
                      deletePreset={deletePreset}
                      change={change}
                    />
                    <section class="library-section">
                      <div class="results-heading">
                        <div>
                          <h2>
                            {t(
                              service === "radarr"
                                ? "Your movies"
                                : "Your TV series",
                            )}{" "}
                            <span>{count(results.total)}</span>
                            {busy && <LoaderCircle size={16} class="spin" />}
                          </h2>
                          <p>
                            {active.length
                              ? t("Files matching your filters")
                              : t("Your complete library")}
                            {results.displayTotal > results.total &&
                              t(
                                " · {count} additional files as season context",
                                {
                                  count: count(
                                    results.displayTotal - results.total,
                                  ),
                                },
                              )}
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
                          <button
                            type="button"
                            class="button"
                            disabled={!results.total}
                            onClick={() => setCustomOpen(true)}
                          >
                            {t("Custom tags")}
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
                        <table
                          class={
                            displayColumns.includes("custom")
                              ? "with-custom"
                              : ""
                          }
                        >
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
                                      <b>{f.title}</b>
                                    </div>
                                    {service === "sonarr" && (
                                      <small class="episode-filename">
                                        {f.filename}
                                      </small>
                                    )}
                                    {!f.matched && (
                                      <small class="context-label">
                                        {t(
                                          "Season context · not a filter match",
                                        )}
                                      </small>
                                    )}
                                  </td>
                                ),
                                info: (
                                  <td key="info" class="info-cell">
                                    <RowInfo
                                      file={f}
                                      settings={settings}
                                      details={() => setFilename(f)}
                                    />
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
                                        <Tag
                                          value={f.processing}
                                          kind="processing"
                                        />
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
                                bitDepth: (
                                  <td key="bitDepth">
                                    <Tag kind="bitDepth" value={f.bitDepth} />
                                  </td>
                                ),
                                audio: (
                                  <td key="audio">
                                    <AudioCell file={f} />
                                  </td>
                                ),
                                custom: (
                                  <td key="custom" class="custom-cell">
                                    <div class="source-tags">
                                      <button
                                        class="icon-button single-tag-button"
                                        aria-label={t(
                                          "Edit custom tags: {title}",
                                          { title: f.title },
                                        )}
                                        title={t("Edit custom tags: {title}", {
                                          title: f.title,
                                        })}
                                        onClick={() => setSingleCustom(f.id)}
                                      >
                                        <TagIcon size={15} />
                                      </button>
                                      {(f.custom || []).map((tag) => (
                                        <Tag
                                          kind="custom"
                                          value={String(tag.id)}
                                          forceTag={tag.manual}
                                          label={tag.name}
                                          title={tag.name}
                                        />
                                      ))}
                                    </div>
                                  </td>
                                ),
                                channels: (
                                  <td key="channels">
                                    <Tag kind="channels" value={f.channels} />
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
                                                number: String(
                                                  f.season,
                                                ).padStart(2, "0"),
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
                                  <TagInformation.Provider
                                    value={f.informationOnly || {}}
                                  >
                                    <tr
                                      key={f.service + f.id}
                                      class={!f.matched ? "context-row" : ""}
                                    >
                                      {displayColumns.map(
                                        (field) => cells[field],
                                      )}
                                    </tr>
                                  </TagInformation.Provider>
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
                                : t(
                                    "Connect a service and start the first sync.",
                                  )}
                            </p>
                            <button
                              class={active.length ? "icon-button" : "button"}
                              title={
                                active.length ? t("Reset filters") : undefined
                              }
                              aria-label={
                                active.length
                                  ? t("Reset filters")
                                  : t("Open settings")
                              }
                              onClick={
                                active.length ? reset : () => openSettings()
                              }
                            >
                              {active.length ? (
                                <RotateCcw size={16} />
                              ) : (
                                t("Open settings")
                              )}
                            </button>
                          </div>
                        )}
                      </div>
                      <div class="pagination">
                        <span>
                          {results.displayTotal
                            ? t("{from}–{to} of {total} files", {
                                from: count((page - 1) * 100 + 1),
                                to: count(
                                  Math.min(page * 100, results.displayTotal),
                                ),
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
                          <span>
                            {t("Page {page} / {pages}", { page, pages })}
                          </span>
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
                  </>
                )}
              </main>
              {(customOpen || singleCustom !== null) && (
                <CustomBulk
                  query={
                    singleCustom !== null
                      ? new URLSearchParams({
                          service,
                          fileId: String(singleCustom),
                        }).toString()
                      : exportQuery.toString()
                  }
                  single={singleCustom !== null}
                  close={() => {
                    setCustomOpen(false);
                    setSingleCustom(null);
                  }}
                  saved={async (total) => {
                    setNotice(
                      t("Custom tags updated for {count} matching files.", {
                        count: count(total),
                      }),
                    );
                    await loadCustom();
                    await refresh();
                    if (singleCustom === null) setPage(1);
                    setCustomOpen(false);
                    setSingleCustom(null);
                  }}
                />
              )}
              {filename && (
                <TagInformation.Provider value={filename.informationOnly || {}}>
                  <FileDialog
                    warningColors={
                      settings?.warningColors || defaultWarningColors
                    }
                    file={filename}
                    close={() => setFilename(null)}
                  />
                </TagInformation.Provider>
              )}{" "}
              {savePreset && (
                <Modal
                  title={t("Save filter view")}
                  close={() => setSavePreset(false)}
                >
                  <form onSubmit={doSavePreset}>
                    {presetError && (
                      <p class="error" role="alert">
                        {presetError}
                      </p>
                    )}
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
          </TagTextColors.Provider>
        </TagColors.Provider>
      </TagVisibility.Provider>
    </CustomTagNames.Provider>
  );
}
