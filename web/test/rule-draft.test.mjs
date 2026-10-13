import test from "node:test";
import assert from "node:assert/strict";
import { mergeRuleDraft, sameRuleDraft } from "../src/rule-draft.ts";
const rule = (id, enabled = true, tag) => ({
  id,
  name: id,
  field: "source",
  match: "token",
  pattern: id,
  value: "UHD",
  mode: "override",
  priority: 10,
  enabled,
  ...(tag === undefined ? {} : { tag }),
});
test("confirmed rule creation/deletion preserves other unsaved checkbox changes", () => {
  const base = [rule("a"), rule("b")];
  const draft = [rule("a", false, false), rule("b")];
  const created = [rule("a"), rule("b"), rule("c")];
  assert.deepEqual(mergeRuleDraft(created, [...draft, rule("c")], created), [
    rule("a", false, false),
    rule("b"),
    rule("c"),
  ]);
  const deleted = [rule("a")];
  assert.deepEqual(mergeRuleDraft(deleted, [draft[0]], deleted), [draft[0]]);
  assert.deepEqual(base, [rule("a"), rule("b")]);
});
test("save all flags accepts the server snapshot and does not resurrect deleted rules", () => {
  const committed = [rule("a", false, false)];
  assert.deepEqual(mergeRuleDraft(committed, committed, committed), committed);
  assert.deepEqual(
    mergeRuleDraft(
      [rule("a"), rule("b")],
      [rule("a", false, false), rule("b")],
      [rule("a")],
    ),
    [rule("a", false, false)],
  );
});

test("returning a color checkbox to its default is a clean draft", () => {
  assert.equal(sameRuleDraft([rule("a")], [rule("a", true, true)]), true);
  assert.equal(sameRuleDraft([rule("a")], [rule("a", true, false)]), false);
  assert.equal(sameRuleDraft([rule("a")], [rule("a", false)]), false);
});
