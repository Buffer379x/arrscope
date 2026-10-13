import type { RecognitionRule } from "./types";
// Preserve pending checkbox edits while confirmed CRUD commits its server snapshot.
export function mergeRuleDraft(
  base: RecognitionRule[],
  draft: RecognitionRule[],
  saved: RecognitionRule[],
): RecognitionRule[] {
  const originals = new Map(base.map((rule) => [rule.id, rule]));
  const drafts = new Map(draft.map((rule) => [rule.id, rule]));
  return saved.map((rule) => {
    const old = originals.get(rule.id),
      pending = drafts.get(rule.id);
    if (!old || !pending) return rule;
    return {
      ...rule,
      ...(old.enabled !== pending.enabled ? { enabled: pending.enabled } : {}),
      ...((old.tag !== false) !== (pending.tag !== false)
        ? { tag: pending.tag }
        : {}),
    };
  });
}

export function sameRuleDraft(
  a: RecognitionRule[],
  b: RecognitionRule[],
): boolean {
  const canonical = (rules: RecognitionRule[]) =>
    JSON.stringify(rules.map((rule) => ({ ...rule, tag: rule.tag !== false })));
  return canonical(a) === canonical(b);
}
