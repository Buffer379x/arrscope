import test from "node:test";
import assert from "node:assert/strict";
import { canonicalPresetQuery } from "../src/preset-query.ts";
test("preset identity normalizes selections, defaults, ranges, search and legacy aliases", () => {
  for (const [a,b] of [
    ["resolution=1080p&resolution=2160p&codec=x265", "codec=x265&resolution=2160p&resolution=1080p&resolution=1080p&resolutionOp=eq&page=4&sort=year&dir=desc"],
    ["language=de&language=eng&languageOp=all", "language=Englisch&language=Deutsch&languageOp=all&sourceOp=not"],
    ["yearMin=02010&sizeMin=0001024", "yearMin=2010&sizeMin=1024"],
    ["hdr=HDR10&hdrOp=notAll", "hdr=HDR10&hdrOp=not"],
    ["q=FOO++bar+foo", "q=bar+foo"],
  ]) assert.equal(canonicalPresetQuery(a),canonicalPresetQuery(b));
  assert.notEqual(canonicalPresetQuery("codec=x265"),canonicalPresetQuery("codec=x265&codecOp=not"));
  assert.notEqual(canonicalPresetQuery("hdr=HDR10&hdr=Dolby+Vision&hdrOp=all"),canonicalPresetQuery("hdr=HDR10&hdr=Dolby+Vision&hdrOp=notAll"));
  assert.notEqual(canonicalPresetQuery("conflict=true"),canonicalPresetQuery("conflict=false"));
});
test("media-server and season status filters remain part of saved preset identity", () => {
  assert.equal(canonicalPresetQuery("mediaMissing=true&mediaMatched=true&page=7"), canonicalPresetQuery("mediaMatched=true&mediaMissing=true"));
  for (const key of ["mediaMatched", "mediaMissing", "mediaAmbiguous", "mediaConflicting", "mediaNoTracks", "mediaPending", "mediaUnconfigured", "seasonOutlier", "languageOutlier"]) {
    assert.notEqual(canonicalPresetQuery(`${key}=true`), canonicalPresetQuery(""));
    assert.notEqual(canonicalPresetQuery(`${key}=true`), canonicalPresetQuery(`${key}=false`));
  }
});
