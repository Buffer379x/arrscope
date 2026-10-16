import test from "node:test";
import assert from "node:assert/strict";
import { channelRows } from "../src/audio-channels.ts";
import { mediaEntryLink, mediaEntryChoices } from "../src/media-link.ts";

test("channels follow track order, keep duplicates and never borrow another track layout", () => {
  const tracks = [
    { channelLayout: "5.1", channels: 6 },
    { channelLayout: "stereo", channels: 2 },
    { channelLayout: "5.1", channels: 6 },
    { channels: 0 },
    { channels: 6 },
  ];
  assert.deepEqual(channelRows({ audioTracks: tracks, channels: "7.1" }), [
    "5.1",
    "2.0",
    "5.1",
    "",
    "6 ch",
  ]);
  assert.deepEqual(channelRows({ channels: "2.0 + 5.1" }), ["2.0", "5.1"]);
  assert.deepEqual(channelRows({ channels: "" }), [""]);
  assert.deepEqual(
    channelRows({
      audioTracks: [{ channelLayout: "mono", channels: 1 }],
      channels: "",
    }),
    ["1.0"],
  );
});
test("provider links preserve subpaths and require safe matched route identities", () => {
  const c = { enabled: true, url: "https://media.example/subpath/" };
  const match = {
    status: "matched",
    link: { itemId: "series-123", serverId: "server_456" },
  };
  assert.equal(
    mediaEntryLink(c, "emby", match),
    "https://media.example/subpath/web/index.html#!/item?id=series-123&serverId=server_456",
  );
  assert.equal(
    mediaEntryLink(c, "jellyfin", match),
    "https://media.example/subpath/web/#/details?id=series-123&serverId=server_456",
  );
  for (const url of [
    "javascript:alert(1)",
    "https://user:secret@media.test",
    "https://media.test/?apikey=secret",
    "https://media.test/#fragment",
    "",
  ])
    assert.equal(mediaEntryLink({ ...c, url }, "emby", match), undefined);
  assert.equal(
    mediaEntryLink({ ...c, enabled: false }, "emby", match),
    undefined,
  );
  assert.equal(mediaEntryLink(c, "unknown", match), undefined);
  assert.equal(mediaEntryLink(c, "emby", { status: "missing" }), undefined);
  for (const itemId of [
    "../settings",
    "id&api_key=secret",
    "x".repeat(129),
    "",
  ])
    assert.equal(
      mediaEntryLink(c, "emby", { ...match, link: { ...match.link, itemId } }),
      undefined,
    );
  const file = { mediaServers: { emby: match, jellyfin: match } };
  assert.deepEqual(
    mediaEntryChoices(
      { emby: c, jellyfin: { ...c, enabled: false } },
      file,
    ).map((x) => x.provider),
    ["emby"],
  );
  assert.equal(mediaEntryChoices({ emby: c, jellyfin: c }, file).length, 2);
  assert.equal(
    mediaEntryChoices(
      { emby: c, jellyfin: c },
      { mediaServers: { emby: match } },
    )[1].href,
    undefined,
  );
  assert.equal(mediaEntryChoices(null, file).length, 0);
});
