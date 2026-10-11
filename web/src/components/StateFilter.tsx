import { useEffect, useState } from "preact/hooks";
import { t, useI18n } from "../i18n";
import { OperatorButton } from "./OperatorButton";
import type { FilterUpdate } from "../ui";

export function StateFilter({
  name,
  label,
  value,
  positive,
  negative,
  change,
}: {
  name: string;
  label: string;
  value: string;
  positive: string;
  negative: string;
  change: (key: string, value: FilterUpdate) => void;
}) {
  useI18n();
  const active = value === positive || value === negative;
  const [exclude, setExclude] = useState(value === negative);
  useEffect(() => {
    if (active) setExclude(value === negative);
  }, [value, active, negative]);
  return (
    <div class="state-filter">
      <label class="check-field">
        <input
          type="checkbox"
          checked={active}
          onChange={(e) =>
            change(
              name,
              e.currentTarget.checked ? [exclude ? negative : positive] : [],
            )
          }
        />
        {t(label)}
      </label>
      <OperatorButton
        field={t(label)}
        value={exclude ? "not" : "eq"}
        change={(mode) => {
          setExclude(mode === "not");
          // Preparing an operator never activates an unchecked filter.
          if (active) change(name, [mode === "not" ? negative : positive]);
        }}
      />
    </div>
  );
}
