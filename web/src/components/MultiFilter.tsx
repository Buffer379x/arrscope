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
          aria-label={labels[name] + ": Operator"}
          value={operator}
          onChange={(e) =>
            operatorChange(e.currentTarget.value as FilterOperator)
          }
        >
          <option value="eq">
            {compound ? "enthält mindestens eines" : "ist"}
          </option>
          <option value="not">
            {compound ? "enthält keines" : "ist nicht"}
          </option>
          {compound && (
            <>
              <option value="all">enthält alle</option>
              <option value="notAll">enthält nicht alle</option>
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
                ? "Nicht alle: "
                : operator === "all"
                  ? "Alle: "
                  : negative
                    ? "Nicht: "
                    : "") +
              selected
                .map((v) => (v === "__unknown__" ? "Unbekannt" : v))
                .join(", ")
            : negative
              ? "Keine Ausschlüsse"
              : "Alle"}
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
            Auswahl zurücksetzen
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
                {v === "__unknown__" ? "Unbekannt" : v}
              </label>
            ))
          ) : (
            <span class="quiet">Noch keine Werte im Cache</span>
          )}
        </div>
      )}
    </div>
  );
}
