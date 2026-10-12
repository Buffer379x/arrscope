import { t, useI18n } from "../i18n";
import { audioLanguageKey } from "../languages";
import {
  RotateCcw,
  Bookmark,
  ChevronDown,
  Search,
  SlidersHorizontal,
  Trash2,
  X,
} from "lucide-preact";
import type { Preset, Service } from "../types";

import {
  embyMatchLabels,
  FilterOperator,
  FilterUpdate,
  labels,
  operatorLabels,
} from "../ui";
import { StateFilter } from "./StateFilter";
import { MultiFilter } from "./MultiFilter";
type Props = {
  q: URLSearchParams;
  collapsed: boolean;
  setCollapsed: (value: boolean) => void;
  active: [string, string][];
  service: Service;
  facets: Record<string, string[]>;
  presets: Preset[];
  selectedPreset: Preset | undefined;
  applyPreset: (preset: Preset) => void;
  reset: () => void;
  setSavePreset: (value: boolean) => void;
  deletePreset: () => Promise<void>;
  change: (key: string, values: FilterUpdate) => void;
};
export function FilterPanel({
  q,
  collapsed,
  setCollapsed,
  active,
  service,
  facets,
  presets,
  selectedPreset,
  applyPreset,
  reset,
  setSavePreset,
  deletePreset,
  change,
}: Props) {
  useI18n();
  return (
    <section
      class={"filter-panel filter-panel-v2" + (collapsed ? " collapsed" : "")}
      aria-label={t("Filters")}
    >
      <div class="filter-always-visible">
        <label class="search-field">
          <span>{t("Title or filename")}</span>
          <div class="input-wrap">
            <Search size={17} />
            <input
              placeholder={
                service === "radarr"
                  ? t("Search movie or file …")
                  : t("Series, episode or file …")
              }
              value={q.get("q") || ""}
              onInput={(e) =>
                change(
                  "q",
                  e.currentTarget.value ? [e.currentTarget.value] : [],
                )
              }
            />
          </div>
        </label>
        <div class="preset-toolbar">
          <select
            aria-label={t("Filter preset")}
            value={selectedPreset ? String(selectedPreset.id) : "all"}
            disabled={!presets.some((p) => p.service === service)}
            onChange={(e) => {
              const preset = presets.find(
                (p) => String(p.id) === e.currentTarget.value,
              );
              if (preset) applyPreset(preset);
              else if (e.currentTarget.value === "all") reset();
            }}
          >
            <option value="all">{t("All files")}</option>
            {presets
              .filter((p) => p.service === service)
              .map((p) => (
                <option value={String(p.id)}>{p.name}</option>
              ))}
          </select>
        </div>
        <div class="filter-icon-tools">
          <button
            class="icon-button"
            aria-label={t("Save preset")}
            title={t("Save filters as a permanent preset")}
            onClick={() => setSavePreset(true)}
          >
            <Bookmark size={16} />
          </button>
          <button
            class="icon-button danger-button"
            aria-label={t("Delete selected preset")}
            title={t("Delete selected preset")}
            disabled={!selectedPreset}
            onClick={deletePreset}
          >
            <Trash2 size={16} />
          </button>
          <button
            class="icon-button"
            aria-label={collapsed ? t("Expand filters") : t("Collapse filters")}
            title={collapsed ? t("Expand filters") : t("Collapse filters")}
            aria-expanded={!collapsed}
            aria-controls="filter-controls"
            onClick={() => setCollapsed(!collapsed)}
          >
            <ChevronDown size={17} class={!collapsed ? "rotated" : ""} />
          </button>
        </div>
      </div>
      {!collapsed && (
        <div id="filter-controls" class="filter-groups">
          <section class="attribute-filters">
            <h2>{t("Media attributes")}</h2>
            <div class="attribute-grid">
              {[
                "resolution",
                "codec",
                "source",
                "processing",
                "family",
                "bitDepth",
                "hdr",
                "audio",
                "language",
                "channels",
                "custom",
              ].map((k) => (
                <MultiFilter
                  key={k}
                  name={k}
                  values={facets[k] || []}
                  selected={q.getAll(k)}
                  operator={(q.get(k + "Op") || "eq") as FilterOperator}
                  operatorChange={(op) =>
                    change(k + "Op", op === "eq" ? [] : [op])
                  }
                  change={(v) => change(k, v)}
                />
              ))}
              <label class="range-field">
                <span>{t("Year")}</span>
                <div class="year-inputs">
                  {["yearMin", "yearMax"].map((k, index) => (
                    <>
                      <input
                        type="number"
                        min="1800"
                        max="3000"
                        placeholder={index ? t("To") : t("From")}
                        aria-label={index ? t("Year to") : t("Year from")}
                        value={q.get(k) || ""}
                        onInput={(e) =>
                          change(
                            k,
                            e.currentTarget.value
                              ? [e.currentTarget.value]
                              : [],
                          )
                        }
                      />
                      {!index && <span>–</span>}
                    </>
                  ))}
                </div>
              </label>
              <label class="range-field">
                <span>{t("File size (GiB)")}</span>
                <div class="year-inputs">
                  {["sizeMin", "sizeMax"].map((k, index) => (
                    <>
                      <input
                        type="number"
                        min="0"
                        step="0.1"
                        placeholder={index ? t("To") : t("From")}
                        aria-label={
                          index
                            ? t("File size to (GiB)")
                            : t("File size from (GiB)")
                        }
                        value={q.has(k) ? Number(q.get(k)) / 1024 ** 3 : ""}
                        onInput={(e) =>
                          change(
                            k,
                            e.currentTarget.value
                              ? [
                                  String(
                                    Math.round(
                                      Number(e.currentTarget.value) * 1024 ** 3,
                                    ),
                                  ),
                                ]
                              : [],
                          )
                        }
                      />
                      {!index && <span>–</span>}
                    </>
                  ))}
                </div>
              </label>
            </div>
          </section>
          <section class="state-filters">
            <h2>{t("Status and metadata quality")}</h2>
            <StateFilter
              name="conflict"
              label="Metadata disagreement"
              value={q.get("conflict") || ""}
              positive="true"
              negative="false"
              change={change}
            />
            <StateFilter
              name="ruleConflict"
              label="Conflicting recognition rules"
              value={q.get("ruleConflict") || ""}
              positive="true"
              negative="false"
              change={change}
            />
            <StateFilter
              name="metadata"
              label="Incomplete metadata"
              value={q.get("metadata") || ""}
              positive="incomplete"
              negative="complete"
              change={change}
            />
            <StateFilter
              name="embyMatch"
              label="All unsuccessful matches"
              value={q.get("embyMatch") || ""}
              positive="unmatched"
              negative="matched"
              change={change}
            />
            {["unknownYear", "unknownSize"].map((k) => (
              <label class="check-field">
                <input
                  type="checkbox"
                  checked={q.get(k) === "true"}
                  onChange={(e) =>
                    change(k, e.currentTarget.checked ? ["true"] : [])
                  }
                />
                {k === "unknownYear"
                  ? t("Unknown year")
                  : t("Unknown file size")}
              </label>
            ))}
            <details class="filter-detail">
              <summary>{t("Detailed Emby status")}</summary>
              <label>
                <span>{t("Emby matching")}</span>
                <select
                  aria-label={t("Emby matching")}
                  value={q.get("embyMatch") || ""}
                  onChange={(e) =>
                    change(
                      "embyMatch",
                      e.currentTarget.value ? [e.currentTarget.value] : [],
                    )
                  }
                >
                  <option value="">{t("All files")}</option>
                  {[
                    "unmatched",
                    "matched",
                    "missing",
                    "ambiguous",
                    "missing_size",
                    "no_tracks",
                    "unconfigured",
                  ].map((k) => (
                    <option value={k}>{embyMatchLabels[k]}</option>
                  ))}
                </select>
              </label>
            </details>
            <details class="filter-detail">
              <summary>{t("Filter logic")}</summary>
              <p class="filter-help">
                {t(
                  "Different fields use AND. For audio, HDR and language, the operator decides whether some or all selected values are required. “Does not contain all” means at least one is missing from the known metadata, which does not prove its absence in the file. “Unknown” also finds partially unknown audio tracks.",
                )}
              </p>
            </details>
          </section>
        </div>
      )}
      <div class="filter-footer">
        <div class="active-filters">
          {active.length ? (
            active
              .filter(
                ([k], i) =>
                  !["all", "notAll"].includes(q.get(k + "Op") || "") ||
                  active.findIndex(([key]) => key === k) === i,
              )
              .map(([k, v]) => (
                <button
                  class={
                    "filter-chip" +
                    (["not", "notAll"].includes(q.get(k + "Op") || "")
                      ? " excluded"
                      : "")
                  }
                  onClick={() =>
                    change(k, (current) =>
                      ["all", "notAll"].includes(q.get(k + "Op") || "")
                        ? []
                        : current.filter((x) => x !== v),
                    )
                  }
                >
                  {labels[k] ||
                    (
                      {
                        q: t("Search"),
                        yearMin: t("From"),
                        yearMax: t("To"),
                        conflict: t("Metadata disagreement"),
                        ruleConflict: t("Conflicting recognition rules"),
                        embyMatch: t("Emby matching"),
                        metadata: t("Metadata status"),
                        sizeMin: t("Bytes from"),
                        sizeMax: t("Bytes to"),
                        unknownYear: t("Unknown year"),
                        unknownSize: t("Unknown size"),
                      } as Record<string, string>
                    )[k] ||
                    k}
                  {["all", "notAll"].includes(q.get(k + "Op") || "")
                    ? " " +
                      operatorLabels[q.get(k + "Op") as FilterOperator] +
                      ": "
                    : q.get(k + "Op") === "not"
                      ? " ≠ "
                      : ": "}
                  {(["all", "notAll"].includes(q.get(k + "Op") || "")
                    ? q.getAll(k)
                    : [v]
                  )
                    .map((value) =>
                      k === "embyMatch"
                        ? embyMatchLabels[value] || value
                        : ["conflict", "ruleConflict"].includes(k)
                          ? t(value === "true" ? "Conflicting" : "No conflicts")
                          : value === "__unknown__"
                            ? t("Unknown")
                            : k === "metadata"
                              ? value === "incomplete"
                                ? t("incomplete")
                                : t("complete")
                              : k === "language"
                                ? t(audioLanguageKey(value))
                                : value,
                    )
                    .join(" + ")}
                  <X size={12} />
                </button>
              ))
          ) : (
            <span class="quiet">{t("No active filters")}</span>
          )}
        </div>
        {active.length > 0 && (
          <button
            class="icon-button"
            onClick={reset}
            aria-label={t("Reset filters")}
            title={t("Reset filters")}
          >
            <RotateCcw size={16} />
          </button>
        )}
      </div>
    </section>
  );
}
