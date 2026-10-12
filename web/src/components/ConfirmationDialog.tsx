import { useEffect, useState } from "preact/hooks";
import { Trash2 } from "lucide-preact";
import { t, useI18n } from "../i18n";
import {
  currentConfirmation,
  subscribeConfirmation,
  answerConfirmation,
} from "../confirmation";
import { Modal } from "./Modal";
export function ConfirmationDialog() {
  useI18n();
  const [, refresh] = useState(0);
  useEffect(() => subscribeConfirmation(() => refresh((n) => n + 1)), []);
  const pending = currentConfirmation();
  if (!pending) return null;
  return (
    <Modal
      title={t("Confirm deletion")}
      close={() => answerConfirmation(false)}
      className="confirmation-dialog"
    >
      <p>{pending.message}</p>
      <div class="modal-actions">
        <button
          type="button"
          class="button"
          onClick={() => answerConfirmation(false)}
        >
          {t("Cancel")}
        </button>
        <button
          type="button"
          class="button danger-button"
          onClick={() => answerConfirmation(true)}
        >
          <Trash2 size={16} />
          {t("Delete")}
        </button>
      </div>
    </Modal>
  );
}
