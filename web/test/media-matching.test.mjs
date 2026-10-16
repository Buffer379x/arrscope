import test from "node:test";
import assert from "node:assert/strict";
import { matchingGroups, matchingCount } from "../src/media-matching.ts";
test("matching widget groups every detailed state exactly once", () => {
  const states = [
    "matched",
    "no_tracks",
    "missing",
    "missing_size",
    "ambiguous",
    "conflicting",
    "pending",
    "unconfigured",
  ];
  const grouped = matchingGroups.flatMap((g) => [...g.states]);
  assert.equal(new Set(grouped).size, states.length);
  assert.deepEqual([...grouped].sort(), states.sort());
  const counts = Object.fromEntries(
    states.map((state, index) => [state, index + 1]),
  );
  assert.equal(
    matchingGroups.reduce((sum, g) => sum + matchingCount(counts, g.states), 0),
    Object.values(counts).reduce((a, b) => a + b, 0),
  );
  assert.equal(
    matchingCount({ matched: 3, no_tracks: 2 }, matchingGroups[0].states),
    5,
  );
  assert.equal(matchingCount({}, matchingGroups[3].states), 0);
});
