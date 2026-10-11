import { t, useI18n } from "../i18n";
import { audioLanguageKey } from "../languages";
import {
  Bookmark,
  ChevronDown,
  Search,
  SlidersHorizontal,
  Trash2,
  X,
} from "lucide-preact";
import type { Preset, Service } from "../types";

import { FilterOperator, FilterUpdate, labels, operatorLabels } from "../ui";
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
      class={"filter-panel" + (collapsed ? " collapsed" : "")}
      aria-label={t("Filters")}
    >
      <div class="filter-heading">
        <span>
          <SlidersHorizontal size={16} />
          {t("FILTERS")}
        </span>
        <div class="preset-toolbar">
          <select
            aria-label={t("Filter preset")}
            value={
              selectedPreset
                ? String(selectedPreset.id)
                : active.length
                  ? ""
                  : "all"
            }
            onChange={(e) => {
              const p = presets.find(
                (p) => String(p.id) === e.currentTarget.value,
              );
              if (p) applyPreset(p);
              else if (e.currentTarget.value === "all") reset();
            }}
          >
            <option value="">{t("Custom filters")}</option>
            <option value="all">{t("All files")}</option>
            {presets
              .filter((p) => p.service === service)
              .map((p) => (
                <option value={String(p.id)}>{p.name}</option>
              ))}
          </select>
          <button
            class="icon-button"
            aria-label={t("Save preset")}
            title={t("Save filters as a permanent preset")}
            onClick={() => setSavePreset(true)}
          >
            <Bookmark size={16} />
          </button>
          <button
            class="icon-button"
            aria-label={t("Delete selected preset")}
            title={t("Delete selected preset")}
            disabled={!selectedPreset}
            onClick={deletePreset}
          >
            <Trash2 size={16} />
          </button>
        </div>
        <div class="filter-view-tools">
          <button
            class="text-button"
            aria-expanded={!collapsed}
            aria-controls="filter-controls"
            onClick={() => setCollapsed(!collapsed)}
          >
            {collapsed ? t("Expand filters") : t("Collapse filters")}
            <ChevronDown size={15} class={!collapsed ? "rotated" : ""} />
          </button>
        </div>
      </div>
      {!collapsed && (
        <div id="filter-controls">
          <div class="filter-grid">
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
            {["resolution", "codec", "source"].map((k) => (
              <MultiFilter
                key={k}
                name={k}
                values={facets[k] || []}
                selected={q.getAll(k)}
                operator={(q.get(k + "Op") || "eq") as FilterOperator}
                operatorChange={(operator) =>
                  change(k + "Op", operator === "eq" ? [] : [operator])
                }
                change={(v) => change(k, v)}
              />
            ))}
            <label class="year-field">
              <span>{t("Year")}</span>
              <div class="year-inputs">
                <input
                  type="number"
                  min="1800"
                  max="3000"
                  placeholder={t("From")}
                  aria-label={t("Year from")}
                  value={q.get("yearMin") || ""}
                  onInput={(e) =>
                    change(
                      "yearMin",
                      e.currentTarget.value ? [e.currentTarget.value] : [],
                    )
                  }
                />
                <span>–</span>
                <input
                  type="number"
                  min="1800"
                  max="3000"
                  placeholder={t("To")}
                  aria-label={t("Year to")}
                  value={q.get("yearMax") || ""}
                  onInput={(e) =>
                    change(
                      "yearMax",
                      e.currentTarget.value ? [e.currentTarget.value] : [],
                    )
                  }
                />
              </div>
            </label>
          </div>
          <div class="advanced-grid">
            {[
              "family",
              "processing",
              "hdr",
              "bitDepth",
              "audio",
              "channels",
              "language",
            ].map((k) => (
              <MultiFilter
                key={k}
                name={k}
                values={facets[k] || []}
                selected={q.getAll(k)}
                operator={(q.get(k + "Op") || "eq") as FilterOperator}
                operatorChange={(operator) =>
                  change(k + "Op", operator === "eq" ? [] : [operator])
                }
                change={(v) => change(k, v)}
              />
            ))}
            {["sizeMin", "sizeMax"].map((k) => (
              <label>
                <span>
                  {k === "sizeMin"
                    ? t("File size from (GiB)")
                    : t("File size to (GiB)")}
                </span>
                <input
                  type="number"
                  min="0"
                  step="0.1"
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
              </label>
            ))}
            <label>
              <span>{t("Data quality")}</span>
              <select
                value={q.get("conflict") || ""}
                onChange={(e) =>
                  change(
                    "conflict",
                    e.currentTarget.value ? [e.currentTarget.value] : [],
                  )
                }
              >
                <option value="">{t("All metadata")}</option>
                <option value="true">{t("Conflicting")}</option>
                <option value="false">{t("No conflicts")}</option>
              </select>
            </label>
            <label>
              <span>{t("Metadata completeness")}</span>
              <select
                aria-label={t("Metadata completeness")}
                value={q.get("metadata") || ""}
                onChange={(e) =>
                  change(
                    "metadata",
                    e.currentTarget.value ? [e.currentTarget.value] : [],
                  )
                }
              >
                <option value="">{t("All files")}</option>
                <option value="incomplete">{t("Incomplete metadata")}</option>
                <option value="complete">
                  {t("Core attributes complete")}
                </option>
              </select>
            </label>
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
          </div>
        </div>
      )}
      <p class="filter-help">
        {!collapsed && (
          <>
            {t(
              "Different fields use AND. For audio, HDR and language, the operator decides whether some or all selected values are required. “Does not contain all” means at least one is missing from the known metadata, which does not prove its absence in the file. “Unknown” also finds partially unknown audio tracks.",
            )}
          </>
        )}
      </p>
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
                        conflict: t("Conflict"),
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
                      value === "__unknown__"
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
          <button class="text-button" onClick={reset}>
            {t("Reset")}
          </button>
        )}
      </div>
    </section>
  );
}
