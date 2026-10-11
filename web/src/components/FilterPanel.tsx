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
  return (
    <section
      class={"filter-panel" + (collapsed ? " collapsed" : "")}
      aria-label="Filter"
    >
      <div class="filter-heading">
        <span>
          <SlidersHorizontal size={16} /> FILTER
        </span>
        <div class="preset-toolbar">
          <select
            aria-label="Filter-Preset"
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
            <option value="">Eigene Filter</option>
            <option value="all">Alle Dateien</option>
            {presets
              .filter((p) => p.service === service)
              .map((p) => (
                <option value={String(p.id)}>{p.name}</option>
              ))}
          </select>
          <button
            class="icon-button"
            aria-label="Preset speichern"
            title="Filter dauerhaft als Preset speichern"
            onClick={() => setSavePreset(true)}
          >
            <Bookmark size={16} />
          </button>
          <button
            class="icon-button"
            aria-label="Ausgewähltes Preset löschen"
            title="Ausgewähltes Preset löschen"
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
            {collapsed ? "Filter ausklappen" : "Filter einklappen"}
            <ChevronDown size={15} class={!collapsed ? "rotated" : ""} />
          </button>
        </div>
      </div>
      {!collapsed && (
        <div id="filter-controls">
          <div class="filter-grid">
            <label class="search-field">
              <span>Titel oder Dateiname</span>
              <div class="input-wrap">
                <Search size={17} />
                <input
                  placeholder={
                    service === "radarr"
                      ? "Film oder Datei suchen …"
                      : "Serie, Episode oder Datei …"
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
              <span>Jahr</span>
              <div class="year-inputs">
                <input
                  type="number"
                  min="1800"
                  max="3000"
                  placeholder="Von"
                  aria-label="Jahr von"
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
                  placeholder="Bis"
                  aria-label="Jahr bis"
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
                <span>Dateigröße {k === "sizeMin" ? "von" : "bis"} (GiB)</span>
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
              <span>Datenqualität</span>
              <select
                value={q.get("conflict") || ""}
                onChange={(e) =>
                  change(
                    "conflict",
                    e.currentTarget.value ? [e.currentTarget.value] : [],
                  )
                }
              >
                <option value="">Alle Angaben</option>
                <option value="true">Widersprüchlich</option>
                <option value="false">Ohne Widerspruch</option>
              </select>
            </label>
            <label>
              <span>Metadaten-Vollständigkeit</span>
              <select
                aria-label="Metadaten-Vollständigkeit"
                value={q.get("metadata") || ""}
                onChange={(e) =>
                  change(
                    "metadata",
                    e.currentTarget.value ? [e.currentTarget.value] : [],
                  )
                }
              >
                <option value="">Alle Dateien</option>
                <option value="incomplete">Metadaten unvollständig</option>
                <option value="complete">Grundmerkmale vollständig</option>
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
                  ? "Jahr unbekannt"
                  : "Dateigröße unbekannt"}
              </label>
            ))}
          </div>
        </div>
      )}
      <p class="filter-help">
        {!collapsed && (
          <>
            Mehrere Eigenschaften gelten mit UND. Bei Audio, HDR und Sprache
            bestimmt der Operator, ob einzelne oder alle gewählten Werte
            vorhanden sein müssen. „Enthält nicht alle“ bedeutet: Mindestens
            eines fehlt in den erkannten Angaben. Das beweist bei
            unvollständigen Metadaten nicht, dass es in der Datei fehlt.
            „Unbekannt“ findet auch teilweise unbekannte Tonspuren.
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
                        q: "Suche",
                        yearMin: "Ab",
                        yearMax: "Bis",
                        conflict: "Konflikt",
                        metadata: "Metadatenstatus",
                        sizeMin: "Bytes ab",
                        sizeMax: "Bytes bis",
                        unknownYear: "Jahr unbekannt",
                        unknownSize: "Größe unbekannt",
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
                        ? "Unbekannt"
                        : k === "metadata"
                          ? value === "incomplete"
                            ? "unvollständig"
                            : "vollständig"
                          : value,
                    )
                    .join(" + ")}
                  <X size={12} />
                </button>
              ))
          ) : (
            <span class="quiet">Keine aktiven Filter</span>
          )}
        </div>
        {active.length > 0 && (
          <button class="text-button" onClick={reset}>
            Zurücksetzen
          </button>
        )}
      </div>
    </section>
  );
}
