import test from "node:test";
import assert from "node:assert/strict";
import { newRuleID } from "../src/rule-id.ts";
test("adding filename rules works without secure-context randomUUID", () => {
  const httpCrypto = { getRandomValues: crypto.getRandomValues.bind(crypto) };
  assert.equal(httpCrypto.randomUUID, undefined);
  const ids = Array.from({ length: 1000 }, () => newRuleID(httpCrypto));
  assert.equal(new Set(ids).size, ids.length);
  for (const id of ids) assert.match(id, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
});
