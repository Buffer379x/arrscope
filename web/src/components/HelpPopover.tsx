import { CircleHelp } from "lucide-preact";
import { useId, useState } from "preact/hooks";
import { t } from "../i18n";
export function HelpPopover({ label, text }: { label: string; text: string }) {
  const [open, setOpen] = useState(false),
    id = useId();
  return (
    <span
      class={"filter-logic-help inline-help" + (open ? " open" : "")}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type="button"
        class="icon-button"
        aria-label={t(label)}
        aria-describedby={id}
        aria-expanded={open}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onClick={() => setOpen(true)}
        onKeyDown={(e) => {
          if (e.key === "Escape") setOpen(false);
        }}
      >
        <CircleHelp size={16} />
      </button>
      <span id={id} role="tooltip" class="filter-logic-tooltip">
        {t(text)}
      </span>
    </span>
  );
}
