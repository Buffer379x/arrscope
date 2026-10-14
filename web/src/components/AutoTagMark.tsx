import { Settings as Gear } from "lucide-preact";
import { useState } from "preact/hooks";
import type { CustomTag } from "../types";
import { t } from "../i18n";
import { Modal } from "./Modal";
export function autoTagHint(tag: CustomTag) {
  return [
    t("Used by external auto-tag rules"),
    ...(tag.rules || []).map((r) =>
      r.replace(/^radarr/, "Radarr").replace(/^sonarr/, "Sonarr"),
    ),
    t("Change the rule in the service to change this tag."),
  ].join("\n");
}
export function AutoTagMark({ tag }: { tag: CustomTag }) {
  const [open, setOpen] = useState(false);
  if (!tag.protected) return null;
  const hint = autoTagHint(tag);
  return (
    <>
      <button
        type="button"
        class="auto-tag-mark"
        title={hint}
        aria-label={hint}
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          setOpen(true);
        }}
      >
        <Gear size={13} />
      </button>
      {open && (
        <Modal title={t("External auto-tag")} close={() => setOpen(false)}>
          <p class="auto-tag-explanation">{hint}</p>
        </Modal>
      )}
    </>
  );
}
