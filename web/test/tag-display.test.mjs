import test from "node:test";
import assert from "node:assert/strict";
import { tagDisplayKey, tagIsDisplayed } from "../src/tag-display.ts";

test("tag display is category-specific, defaults to visible and handles unknowns", () => {
  const hidden = {
    "source:__unknown__": true,
    "codec:vp9": true,
    "audio:DTS": false,
  };
  assert.equal(tagDisplayKey("resolution r2160p", "2160p"), "resolution:2160p");
  assert.equal(tagDisplayKey("source", ""), "source:__unknown__");
  assert.equal(tagIsDisplayed(hidden, "source", ""), false);
  assert.equal(tagIsDisplayed(hidden, "source", "__unknown__"), false);
  assert.equal(tagIsDisplayed(hidden, "codec", "vp9"), false);
  assert.equal(tagIsDisplayed(hidden, "resolution", ""), true);
  assert.equal(tagIsDisplayed(hidden, "audio", "DTS"), true);
  assert.equal(tagIsDisplayed({}, "custom", "42"), true);
  const facets = ["BluRay", "__unknown__", "UHD"];
  assert.deepEqual(
    facets.filter((v) => tagIsDisplayed(hidden, "source", v)),
    ["BluRay", "UHD"],
  );
  assert.deepEqual(facets, ["BluRay", "__unknown__", "UHD"]);
});
