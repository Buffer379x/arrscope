import { confirmDeletion, confirmReset } from "../confirmation";
import { matchesRuleSearch } from "../rule-search";
import { newRuleID } from "../rule-id";
import { notify } from "../notifications";
import {
  ArrowDown,
  ArrowUp,
  ArrowDownUp,
  Check,
  LoaderCircle,
  Pencil,
  Plus,
  RotateCcw,
  Search,
  Trash2,
  OctagonAlert,
} from "lucide-preact";
import { useEffect, useState, useContext } from "preact/hooks";
import { api } from "../api";
import { t, useI18n } from "../i18n";
import type {
  RecognitionConfig,
  RecognitionRule,
  RecognitionResult,
} from "../types";
import { labels, CustomTagNames } from "../ui";
import { Modal } from "./Modal";
const fields = [
  "resolution",
  "codec",
  "source",
  "processing",
  "hdr",
  "bitDepth",
  "audio",
  "channels",
  "language",
  "custom",
];
const modes = {
  direct: "Direct recognition",
  fallback: "Only if missing",
  override: "Explicit override",
};
const columns: [keyof RecognitionRule, string][] = [
  ["priority", "Priority"],
  ["enabled", "Enabled"],
  ["tag", "Tag"],
  ["name", "Name"],
  ["field", "Attribute"],
  ["match", "Match type"],
  ["pattern", "Pattern"],
  ["value", "Result"],
  ["mode", "Mode"],
];
export function RecognitionPage({
  saved,
  onDirty,
}: {
  saved: () => Promise<void>;
  onDirty: (dirty: boolean) => void;
}) {
  useI18n();
  const customNames = useContext(CustomTagNames);
  const [config, setConfig] = useState<RecognitionConfig | null>(null),
    [defaults, setDefaults] = useState<RecognitionConfig | null>(null),
    [rules, setRules] = useState<RecognitionRule[]>([]),
    [sort, setSort] = useState<{ field: keyof RecognitionRule; desc: boolean }>(
      { field: "priority", desc: false },
    ),
    [editing, setEditing] = useState<RecognitionRule | null>(null),
    [ruleSearch, setRuleSearch] = useState(""),
    [filename, setFilename] = useState(""),
    [preview, setPreview] = useState<RecognitionResult | null>(null),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false),
    [previewBusy, setPreviewBusy] = useState(false);
  useEffect(() => {
    if (notice) {
      notify(notice);
      setNotice("");
    }
  }, [notice]);
  const dirty =
    !!config && JSON.stringify(rules) !== JSON.stringify(config.rules);
  useEffect(() => {
    onDirty(dirty);
    const warn = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  useEffect(() => () => onDirty(false), []);
  const load = async () => {
    try {
      const [current, base] = await Promise.all([
        api<RecognitionConfig>("recognition"),
        api<RecognitionConfig>("recognition/defaults"),
      ]);
      setConfig(current);
      setDefaults(base);
      setRules(current.rules);
      setError("");
      setPreview(null);
    } catch (e) {
      setError((e as Error).message);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  const change = async (next: RecognitionRule[]) => {
    if (!config || busy) return false;
    const previous = config.rules;
    setBusy(true);
    onDirty(true);
    setRules(next);
    setPreview(null);
    try {
      const current = await api<RecognitionConfig>("recognition", "PUT", {
        rules: next,
        revision: config.revision,
      });
      setConfig(current);
      setRules(current.rules);
      notify(t("Rules saved. Cached filenames have been reprocessed."));
      await saved();
      return true;
    } catch (e) {
      setRules(previous);
      setError((e as Error).message);
      return false;
    } finally {
      setBusy(false);
      onDirty(false);
    }
  };
  const filtered = rules.filter((rule) =>
    matchesRuleSearch(ruleSearch, [
      String(rule.priority),
      t(rule.enabled ? "Enabled" : "Disabled"),
      t(rule.tag !== false ? "Tag" : "Information only"),
      rule.name,
      labels[rule.field],
      t(rule.match === "token" ? "Token" : "Regular expression"),
      rule.pattern,
      rule.field === "custom"
        ? customNames[rule.value] || rule.value
        : rule.value,
      t(modes[rule.mode as keyof typeof modes]),
      rule.unless || "",
      rule.comparisonPattern || "",
    ]),
  );
  const sorted = [...filtered].sort((a, b) => {
    const x =
        sort.field === "tag"
          ? a.tag !== false
          : sort.field === "value" && a.field === "custom"
            ? customNames[a.value] || a.value
            : a[sort.field],
      y =
        sort.field === "tag"
          ? b.tag !== false
          : sort.field === "value" && b.field === "custom"
            ? customNames[b.value] || b.value
            : b[sort.field];
    const cmp =
      typeof x === "number" && typeof y === "number"
        ? x - y
        : String(x).localeCompare(String(y));
    return (sort.desc ? -cmp : cmp) || a.id.localeCompare(b.id);
  });
  return (
    <section class="recognition-page">
      <header class="settings-section-heading">
        <h2>{t("Filename recognition")}</h2>
        <p class="quiet">
          {t(
            "Rules only read filenames. API metadata fills remaining gaps; Emby audio tracks stay separate. No media scans are triggered.",
          )}
        </p>
      </header>
      <div class="recognition-guide">
        {Object.entries(modes).map(([mode, label]) => (
          <div>
            <b>{t(label)}</b>
            <p>
              {t(
                mode === "direct"
                  ? "Recognizes explicit filename values."
                  : mode === "fallback"
                    ? "Fills a field only when direct recognition found no value."
                    : "Intentionally replaces direct and fallback values.",
              )}
            </p>
          </div>
        ))}
      </div>
      <p class="quiet">
        {t(
          "Lower priority numbers run first; ties use the rule ID. Table sorting does not change priority. Different scalar values create a warning; HDR, audio codecs and languages are additive.",
        )}
      </p>
      {error && !config && (
        <button class="button" onClick={() => void load()}>
          {t("Reload rules")}
        </button>
      )}
      {!config && !error && (
        <p class="quiet">
          <LoaderCircle class="spin" size={16} />
          {t("Loading rules …")}
        </p>
      )}
      {config && (
        <>
          <p class="quiet">
            {t(
              "Custom rules add all matching tags; different tags are not conflicts.",
            )}
          </p>
          <p class="quiet">
            {t(
              "Tag controls presentation and colors. Recognition, filters and outbound categories remain independent.",
            )}
          </p>
          <div class="recognition-actions">
            <label class="search-field recognition-rule-search">
              <span>{t("Search recognition rules")}</span>
              <div class="input-wrap">
                <Search size={17} />
                <input
                  type="search"
                  aria-label={t("Search recognition rules")}
                  placeholder={t("Search all rule columns …")}
                  value={ruleSearch}
                  onInput={(e) => setRuleSearch(e.currentTarget.value)}
                />
              </div>
            </label>
            <div>
              <button
                class="button"
                disabled={busy || rules.length >= 256}
                onClick={() =>
                  setEditing({
                    id: newRuleID(),
                    name: "",
                    field: "resolution",
                    match: "token",
                    pattern: "",
                    value: "",
                    mode: "fallback",
                    priority: 100,
                    enabled: true,
                    consume: false,
                  })
                }
              >
                <Plus size={15} />
                {t("Add rule")}
              </button>
              <button
                class="icon-button"
                aria-label={t("Restore defaults")}
                title={t("Restore defaults")}
                disabled={busy || !defaults}
                onClick={async () => {
                  if (
                    await confirmDeletion(
                      t("Restore all default recognition rules?"),
                    )
                  )
                    change(structuredClone(defaults!.rules));
                }}
              >
                <RotateCcw size={15} />
              </button>
            </div>
          </div>
          <div class="table-wrap">
            <table class="recognition-table">
              <thead>
                <tr>
                  {columns.map(([field, label]) => (
                    <th
                      aria-sort={
                        sort.field === field
                          ? sort.desc
                            ? "descending"
                            : "ascending"
                          : "none"
                      }
                    >
                      <button
                        class="column-sort"
                        onClick={() =>
                          setSort({
                            field,
                            desc: sort.field === field ? !sort.desc : false,
                          })
                        }
                      >
                        {t(label)}
                        {sort.field === field ? (
                          sort.desc ? (
                            <ArrowDown size={13} />
                          ) : (
                            <ArrowUp size={13} />
                          )
                        ) : (
                          <ArrowDownUp size={13} />
                        )}
                      </button>
                    </th>
                  ))}
                  <th>{t("Actions")}</th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((rule) => (
                  <tr
                    key={rule.id}
                    class={!rule.enabled ? "rule-disabled" : ""}
                  >
                    <td>{rule.priority}</td>
                    <td>
                      <input
                        type="checkbox"
                        aria-label={t("Enable rule: {name}", {
                          name: rule.name,
                        })}
                        checked={rule.enabled}
                        disabled={busy}
                        onChange={(e) =>
                          change(
                            rules.map((r) =>
                              r.id === rule.id
                                ? { ...r, enabled: e.currentTarget.checked }
                                : r,
                            ),
                          )
                        }
                      />
                    </td>
                    <td>
                      <input
                        type="checkbox"
                        checked={rule.tag !== false}
                        disabled={busy}
                        aria-label={t("Show as tag: {name}", {
                          name: rule.name,
                        })}
                        onChange={(e) =>
                          void change(
                            rules.map((r) =>
                              r.id === rule.id
                                ? { ...r, tag: e.currentTarget.checked }
                                : r,
                            ),
                          )
                        }
                      />
                    </td>
                    <td>
                      <b>{rule.name}</b>
                      {rule.id.startsWith("default-") && (
                        <small class="quiet">{t("Default rule")}</small>
                      )}
                    </td>
                    <td>{labels[rule.field]}</td>
                    <td>
                      {t(
                        rule.match === "token" ? "Token" : "Regular expression",
                      )}
                    </td>
                    <td>
                      <code>{rule.pattern}</code>
                    </td>
                    <td>
                      {rule.field === "custom"
                        ? customNames[rule.value] || rule.value
                        : rule.value}
                    </td>
                    <td>{t(modes[rule.mode])}</td>
                    <td>
                      <div class="row-actions">
                        <button
                          class="icon-button"
                          disabled={busy}
                          aria-label={t("Edit rule: {name}", {
                            name: rule.name,
                          })}
                          onClick={() => setEditing({ ...rule })}
                        >
                          <Pencil size={15} />
                        </button>
                        <button
                          class="icon-button danger-button"
                          disabled={busy}
                          aria-label={t("Delete rule: {name}", {
                            name: rule.name,
                          })}
                          onClick={async () => {
                            if (
                              await confirmDeletion(
                                t("Delete rule {name}?", { name: rule.name }),
                              )
                            )
                              change(rules.filter((r) => r.id !== rule.id));
                          }}
                        >
                          <Trash2 size={15} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!!rules.length && !filtered.length && (
              <p class="quiet empty-rules">{t("No matching rules.")}</p>
            )}
            {!rules.length && (
              <p class="quiet empty-rules">
                {t(
                  "No filename rules. Only API and Emby metadata will be used.",
                )}
              </p>
            )}
          </div>
          <p class="quiet recognition-rule-count">
            {t("{count} rules", { count: rules.length })}
            {ruleSearch.trim() &&
              " · " + t("{count} matching rules", { count: filtered.length })}
            {busy && " · " + t("Reprocessing cache …")}
          </p>
          <section class="recognition-preview">
            <h2>{t("Preview filename")}</h2>
            <p class="quiet">
              {t(
                "Preview uses saved rules and the filename only. API and Emby enrichment is not included.",
              )}
            </p>
            <form
              onSubmit={async (e) => {
                e.preventDefault();
                setPreviewBusy(true);
                setError("");
                try {
                  setPreview(
                    await api<RecognitionResult>(
                      "recognition/preview",
                      "POST",
                      { rules, filename },
                    ),
                  );
                } catch (e) {
                  setError((e as Error).message);
                } finally {
                  setPreviewBusy(false);
                }
              }}
            >
              <label>
                {t("Filename")}
                <input
                  required
                  maxLength={2048}
                  value={filename}
                  placeholder="Movie.UHDBD.x265.mkv"
                  onInput={(e) => {
                    setFilename(e.currentTarget.value);
                    setPreview(null);
                  }}
                />
              </label>
              <button class="button" disabled={busy || previewBusy}>
                {previewBusy ? (
                  <LoaderCircle class="spin" size={15} />
                ) : (
                  <Search size={15} />
                )}{" "}
                {t("Preview")}
              </button>
            </form>
            <>
              <div class="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>{t("Attribute")}</th>
                      <th>{t("Result")}</th>
                      <th>{t("Matching rules")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {fields.map((field) => (
                      <tr>
                        <td>{labels[field]}</td>
                        <td>
                          {preview
                            ? (field === "custom"
                                ? (preview.values[field] || "")
                                    .split(" + ")
                                    .map((id) => customNames[id] || id)
                                    .filter(Boolean)
                                    .join(" · ")
                                : preview.values[field]) || t("Unknown")
                            : "—"}
                        </td>
                        <td>
                          {(preview?.matches[field] || [])
                            .map(
                              (id) =>
                                rules.find((r) => r.id === id)?.name || id,
                            )
                            .join(" · ") || "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {!!preview?.conflicts.length && (
                <div class="notice recognition-conflict-details">
                  <OctagonAlert size={18} />
                  <div>
                    <b>{t("Conflicting recognition rules")}</b>
                    {preview.conflicts.map((c) => (
                      <p>
                        {labels[c.field]}: {c.values.join(" / ")} ·{" "}
                        {c.ruleNames.join(" · ")}
                      </p>
                    ))}
                  </div>
                </div>
              )}
            </>
          </section>
        </>
      )}
      {editing && (
        <RuleEditor
          rule={editing}
          busy={busy}
          close={() => {
            if (!busy) setEditing(null);
          }}
          save={async (rule) => {
            const ok = await change(
              rules.some((r) => r.id === rule.id)
                ? rules.map((r) => (r.id === rule.id ? rule : r))
                : [...rules, rule],
            );
            if (ok) setEditing(null);
          }}
        />
      )}
    </section>
  );
}
function RuleEditor({
  rule,
  busy,
  close,
  save,
}: {
  rule: RecognitionRule;
  busy: boolean;
  close: () => void;
  save: (rule: RecognitionRule) => Promise<void>;
}) {
  useI18n();
  const customNames = useContext(CustomTagNames);
  const [draft, setDraft] = useState(rule);
  const patch = (value: Partial<RecognitionRule>) =>
    setDraft({ ...draft, ...value });
  return (
    <Modal title={t("Edit recognition rule")} close={close} wide>
      <form
        class="rule-editor"
        onSubmit={async (e) => {
          e.preventDefault();
          if (!busy) await save(draft);
        }}
      >
        <label>
          {t("Name")}
          <input
            disabled={busy}
            required
            maxLength={80}
            value={draft.name}
            onInput={(e) => patch({ name: e.currentTarget.value })}
          />
        </label>
        <label>
          {t("Attribute")}
          <select
            disabled={busy}
            value={draft.field}
            onChange={(e) =>
              patch({
                field: e.currentTarget.value,
                comparisonPattern:
                  e.currentTarget.value === "audio"
                    ? draft.comparisonPattern
                    : "",
                mode:
                  e.currentTarget.value === "custom" ? "direct" : draft.mode,
                value:
                  e.currentTarget.value === "custom"
                    ? Object.keys(customNames)[0] || ""
                    : draft.value,
              })
            }
          >
            {fields.map((field) => (
              <option value={field}>{labels[field]}</option>
            ))}
          </select>
        </label>
        <label>
          {t("Match type")}
          <select
            disabled={busy}
            value={draft.match}
            onChange={(e) =>
              patch({
                match: e.currentTarget.value as RecognitionRule["match"],
              })
            }
          >
            <option value="token">{t("Token")}</option>
            <option value="regex">{t("Regular expression")}</option>
          </select>
        </label>
        <label>
          {t("Pattern")}
          <input
            disabled={busy}
            required
            maxLength={500}
            value={draft.pattern}
            onInput={(e) => patch({ pattern: e.currentTarget.value })}
          />
        </label>
        <label>
          {t("Result")}
          {draft.field === "custom" ? (
            <select
              disabled={busy}
              required
              value={draft.value}
              onChange={(e) => patch({ value: e.currentTarget.value })}
            >
              <option value="">{t("Select a custom tag")}</option>
              {Object.entries(customNames).map(([id, name]) => (
                <option value={id}>{name}</option>
              ))}
            </select>
          ) : (
            <input
              disabled={busy}
              required
              maxLength={80}
              value={draft.value}
              onInput={(e) => patch({ value: e.currentTarget.value })}
            />
          )}
        </label>
        <label>
          {t("Mode")}
          <select
            value={draft.mode}
            disabled={busy || draft.field === "custom"}
            onChange={(e) =>
              patch({ mode: e.currentTarget.value as RecognitionRule["mode"] })
            }
          >
            {Object.entries(modes).map(([mode, label]) => (
              <option value={mode}>{t(label)}</option>
            ))}
          </select>
        </label>
        <label>
          {t("Priority")}
          <input
            disabled={busy}
            required
            type="number"
            min={0}
            max={10000}
            step={1}
            value={draft.priority}
            onInput={(e) => patch({ priority: Number(e.currentTarget.value) })}
          />
        </label>
        <label class="check-field">
          <input
            disabled={busy}
            type="checkbox"
            checked={draft.enabled}
            onChange={(e) => patch({ enabled: e.currentTarget.checked })}
          />
          {t("Enabled")}
        </label>
        <p class="quiet rule-editor-help">
          {t(
            "Matching is case-insensitive. Tokens require separators. Regular expressions use Go RE2 syntax; capture groups can be referenced with $1 in the result.",
          )}
        </p>
        <details class="rule-advanced">
          <summary>{t("Advanced matching")}</summary>
          {draft.field === "audio" && (
            <label>
              {t("Equivalent metadata codecs (regular expression)")}
              <input
                disabled={busy}
                maxLength={500}
                value={draft.comparisonPattern || ""}
                placeholder="^DTS(?:-HD(?: MA)?)?$"
                onInput={(e) =>
                  patch({ comparisonPattern: e.currentTarget.value })
                }
              />
              <small>
                {t(
                  "Matching codec names are equivalent for warnings only. Clear this expression to compare strictly; displayed audio tracks stay unchanged.",
                )}
              </small>
            </label>
          )}
          <label>
            {t("Skip when filename matches this expression")}
            <input
              disabled={busy}
              maxLength={500}
              value={draft.unless || ""}
              onInput={(e) => patch({ unless: e.currentTarget.value })}
            />
          </label>
          <label class="check-field">
            <input
              disabled={busy}
              type="checkbox"
              checked={draft.consume}
              onChange={(e) => patch({ consume: e.currentTarget.checked })}
            />
            {t("Consume matched text before subsequent rules")}
          </label>
          <p class="quiet">
            {t(
              "Useful for audio variants such as DTS-HD before DTS. Applies within the same attribute and mode.",
            )}
          </p>
        </details>
        <div class="modal-actions">
          <button type="button" class="button" onClick={close}>
            {t("Cancel")}
          </button>
          <button class="primary">
            <Check size={15} />
            {t("Apply to draft")}
          </button>
        </div>
      </form>
    </Modal>
  );
}
