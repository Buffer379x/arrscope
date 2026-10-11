import { useEffect, useState } from "preact/hooks";
import { Plus, Save, Trash2 } from "lucide-preact";
import type { CustomTag } from "../types";
import { api } from "../api";
import { t, useI18n } from "../i18n";
export function CustomTagsSettings({
  tags,
  changed,
  onDirty,
}: {
  tags: CustomTag[];
  changed: () => Promise<void>;
  onDirty: (value: boolean) => void;
}) {
  useI18n();
  const [drafts, setDrafts] = useState<Record<string, string>>({}),
    [name, setName] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const dirty =
    !!name.trim() ||
    tags.some(
      (tag) =>
        Object.hasOwn(drafts, String(tag.id)) && drafts[tag.id] !== tag.name,
    );
  useEffect(() => onDirty(dirty), [dirty]);
  useEffect(() => () => onDirty(false), []);
  const save = async (tag: CustomTag) => {
    setBusy(true);
    setError("");
    try {
      await api("custom-tags", tag.id ? "PUT" : "POST", tag);
      setDrafts((current) => {
        const next = { ...current };
        delete next[tag.id];
        return next;
      });
      if (!tag.id) setName("");
      await changed();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <section class="custom-tags-settings" aria-label={t("Custom tags")}>
      <h2>{t("Custom tags")}</h2>
      <p class="quiet">
        {t(
          "Create tags here, assign them in your library or use them in filename rules. Colors and filters show tags currently in use.",
        )}
      </p>
      {error && (
        <div class="notice error" role="alert">
          {error}
        </div>
      )}
      <form
        class="custom-tag-create"
        onSubmit={(e) => {
          e.preventDefault();
          void save({ id: 0, name });
        }}
      >
        <label>
          {t("Tag name")}
          <input
            required
            maxLength={80}
            value={name}
            disabled={busy}
            onInput={(e) => setName(e.currentTarget.value)}
          />
        </label>
        <button class="primary" disabled={busy || !name.trim()}>
          <Plus size={16} />
          {t("Add tag")}
        </button>
      </form>
      <div class="table-wrap custom-tags-table">
        <table>
          <thead>
            <tr>
              <th>{t("Name")}</th>
              <th>{t("Actions")}</th>
            </tr>
          </thead>
          <tbody>
            {tags.map((tag) => (
              <tr>
                <td>
                  <input
                    aria-label={t("Tag name: {name}", { name: tag.name })}
                    maxLength={80}
                    value={drafts[tag.id] ?? tag.name}
                    disabled={busy}
                    onInput={(e) =>
                      setDrafts({ ...drafts, [tag.id]: e.currentTarget.value })
                    }
                  />
                </td>
                <td>
                  <div class="row-actions">
                    <button
                      type="button"
                      class="button"
                      disabled={
                        busy ||
                        !drafts[tag.id]?.trim() ||
                        drafts[tag.id] === tag.name
                      }
                      onClick={() =>
                        void save({ id: tag.id, name: drafts[tag.id] })
                      }
                      aria-label={t("Save tag: {name}", { name: tag.name })}
                      title={t("Save tag: {name}", { name: tag.name })}
                    >
                      <Save size={16} />
                    </button>
                    <button
                      type="button"
                      class="button"
                      disabled={busy}
                      aria-label={t("Delete tag: {name}", { name: tag.name })}
                      title={t("Delete tag: {name}", { name: tag.name })}
                      onClick={async () => {
                        if (
                          !confirm(
                            t(
                              "Delete {name}, its assignments and matching filename rules?",
                              { name: tag.name },
                            ),
                          )
                        )
                          return;
                        setBusy(true);
                        setError("");
                        try {
                          await api("custom-tags", "DELETE", { id: tag.id });
                          await changed();
                        } catch (e) {
                          setError((e as Error).message);
                        } finally {
                          setBusy(false);
                        }
                      }}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!tags.length && (
          <p class="quiet empty-rules">{t("No custom tags yet.")}</p>
        )}
      </div>
    </section>
  );
}
