import { confirmDeletion } from "../confirmation";
import { useEffect, useState } from "preact/hooks";
import { Plus, Check, Trash2, LoaderCircle } from "lucide-preact";
import type { CustomTag } from "../types";
import { api } from "../api";
import { notify } from "../notifications";
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
  const [drafts, setDrafts] = useState<CustomTag[]>(structuredClone(tags));
  const [name, setName] = useState(""),
    [busy, setBusy] = useState(false);
  const dirty =
    !!name.trim() || JSON.stringify(tags) !== JSON.stringify(drafts);
  useEffect(() => onDirty(dirty), [dirty]);
  useEffect(() => () => onDirty(false), []);
  const add = () => {
    const value = name.trim();
    if (!value) return;
    if (
      drafts.some(
        (tag) => tag.name.toLocaleLowerCase() === value.toLocaleLowerCase(),
      )
    ) {
      notify(t("A custom tag with this name already exists"), "warning");
      return;
    }
    setDrafts([
      ...drafts,
      { id: Math.min(0, ...drafts.map((tag) => tag.id)) - 1, name: value },
    ]);
    setName("");
  };
  const save = async () => {
    if (name.trim()) {
      notify(t("Add the entered tag to the list before saving."), "warning");
      return;
    }
    setBusy(true);
    try {
      const saved = await api<CustomTag[]>("custom-tags/batch", "PUT", {
        original: tags,
        tags: drafts,
      });
      setDrafts(saved);
      await changed();
      notify(t("Settings saved."));
    } catch {
    } finally {
      setBusy(false);
    }
  };
  return (
    <section class="custom-tags-settings" aria-label={t("Custom tags")}>
      <header class="settings-section-heading">
        <h2>{t("Custom tags")}</h2>
        <button
          type="button"
          class="primary"
          disabled={busy || !dirty}
          onClick={() => void save()}
        >
          {busy ? <LoaderCircle class="spin" size={16} /> : <Check size={16} />}
          {t("Save settings")}
        </button>
        <p class="quiet">
          {t(
            "Create tags here, assign them in your library or use them in filename rules. Colors and filters show tags currently in use.",
          )}
        </p>
      </header>
      <section class="settings-card">
        <form
          class="custom-tag-create"
          onSubmit={(e) => {
            e.preventDefault();
            add();
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
          <button class="button" disabled={busy || !name.trim()}>
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
              {drafts.map((tag) => (
                <tr key={tag.id}>
                  <td>
                    <input
                      aria-label={t("Tag name: {name}", { name: tag.name })}
                      maxLength={80}
                      required
                      value={tag.name}
                      disabled={busy}
                      onInput={(e) =>
                        setDrafts(
                          drafts.map((current) =>
                            current.id === tag.id
                              ? { ...current, name: e.currentTarget.value }
                              : current,
                          ),
                        )
                      }
                    />
                  </td>
                  <td>
                    <div class="row-actions">
                      <button
                        type="button"
                        class="button danger-button"
                        disabled={busy}
                        aria-label={t("Delete tag: {name}", { name: tag.name })}
                        title={t("Delete tag: {name}", { name: tag.name })}
                        onClick={async () => {
                          if (
                            await confirmDeletion(
                              t(
                                "Delete {name}, its assignments and matching filename rules?",
                                { name: tag.name },
                              ),
                            )
                          )
                            setDrafts(
                              drafts.filter((current) => current.id !== tag.id),
                            );
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
          {!drafts.length && (
            <p class="quiet empty-rules">{t("No custom tags yet.")}</p>
          )}
        </div>
      </section>
    </section>
  );
}
