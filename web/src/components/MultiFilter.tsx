import { t, useI18n } from "../i18n";
import { audioLanguageKey } from "../languages";
import { ChevronDown } from "lucide-preact";
import { useEffect, useRef, useState } from "preact/hooks";

import { compoundFields, FilterOperator, FilterUpdate, labels } from "../ui";
export function MultiFilter({
  name,
  values,
  selected,
  operator,
  operatorChange,
  change,
}: {
  name: string;
  values: string[];
  selected: string[];
  operator: FilterOperator;
  operatorChange: (operator: FilterOperator) => void;
  change: (v: FilterUpdate) => void;
}) {
  useI18n();
  const negative = operator === "not" || operator === "notAll";
  const compound = compoundFields.includes(name);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const dismiss = () => {
    setOpen(false);
    trigger.current?.focus();
  };
  useEffect(() => {
    if (!open) return;
    const outside = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("click", outside);
    return () => document.removeEventListener("click", outside);
  }, [open]);
  return (
    <div
      class="multi-filter"
      ref={ref}
      onKeyDown={(e) => {
        if (open && e.key === "Escape") {
          e.stopPropagation();
          dismiss();
        }
      }}
      onFocusOut={(e) => {
        if (!ref.current?.contains(e.relatedTarget as Node)) setOpen(false);
      }}
    >
      <div class="field-label-row">
        <span class="field-label" id={"label-" + name}>
          {labels[name]}
        </span>
        <select
          class="filter-operator"
          aria-label={t("{field}: operator", { field: labels[name] })}
          value={operator}
          onChange={(e) =>
            operatorChange(e.currentTarget.value as FilterOperator)
          }
        >
          <option value="eq">
            {compound ? t("contains at least one") : t("is")}
          </option>
          <option value="not">
            {compound ? t("contains none") : t("is not")}
          </option>
          {compound && (
            <>
              <option value="all">{t("contains all")}</option>
              <option value="notAll">{t("does not contain all")}</option>
            </>
          )}
        </select>
      </div>
      <button
        ref={trigger}
        class={
          "select-button" +
          (selected.length
            ? negative
              ? " selected excluded"
              : " selected"
            : "")
        }
        aria-labelledby={"label-" + name}
        aria-expanded={open}
        aria-controls={"options-" + name}
        onClick={() => setOpen(!open)}
      >
        <span class="select-values">
          {selected.length
            ? (operator === "notAll"
                ? t("Not all: ")
                : operator === "all"
                  ? t("All: ")
                  : negative
                    ? t("Not: ")
                    : "") +
              selected
                .map((v) =>
                  v === "__unknown__"
                    ? t("Unknown")
                    : name === "language"
                      ? t(audioLanguageKey(v))
                      : v,
                )
                .join(", ")
            : negative
              ? t("No exclusions")
              : t("All")}
        </span>
        <ChevronDown size={15} />
      </button>
      {open && (
        <div
          class="select-popover"
          id={"options-" + name}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              e.stopPropagation();
              dismiss();
            }
          }}
        >
          <button class="text-button" onClick={() => change([])}>
            {t("Reset selection")}
          </button>
          {values.length ? (
            values.map((v) => (
              <label>
                <input
                  type="checkbox"
                  checked={selected.includes(v)}
                  onChange={(e) => {
                    const checked = e.currentTarget.checked;
                    change((current) =>
                      checked
                        ? [...new Set([...current, v])]
                        : current.filter((x) => x !== v),
                    );
                  }}
                />
                {v === "__unknown__"
                  ? t("Unknown")
                  : name === "language"
                    ? t(audioLanguageKey(v))
                    : v}
              </label>
            ))
          ) : (
            <span class="quiet">{t("No values in cache yet")}</span>
          )}
        </div>
      )}
    </div>
  );
}
