# Media servers and season outliers

## Independent Emby and Jellyfin connections

Settings → Services has one Media server card. Its selector switches between independent Emby and Jellyfin configurations: Enabled, URL, encrypted API key and optional path mappings. Saving retains both drafts. One instance of each provider can be enabled simultaneously; both are disabled on a fresh installation. Disabled providers are omitted from System and file details, including while a new snapshot is being built.

Both providers read only `System/Info` and paginated `Items` with an `X-Emby-Token` header. Jellyfin preserves the configured base URL, including subpaths. Emby adds `/emby` when that prefix is not already present. API keys never appear in browser responses or logs. No scan, probe, playback, media mount or refresh command is used. Jellyfin explicitly reported VideoRangeType profiles are normalized to HDR10, HDR10+, HLG and Dolby Vision; SDR and unspecified profiles do not imply HDR.

A full Radarr/Sonarr snapshot reads all enabled media servers in sequence. The provider buttons in System request a fresh library snapshot, including all enabled providers; they do not scan or refresh the media servers. An enabled provider failure rejects the new snapshot and keeps the last complete cache. Webhooks keep supplemental data only for an unchanged file identity, otherwise matching stays pending until the next full sync.

Matching uses a unique basename and positive identical byte size. Optional path mappings add a full-path requirement; alternate versions are matched to their own MediaSources. External audio and subtitles do not become audio tracks. Existing audio language codes are resolved through the bundled ISO registry.

Provider results are retained separately in `mediaServers`. The combined result is matched if at least one provider supplies a unique track inventory and other uniquely matched providers agree. Different reported language/codec/channel inventories produce Conflicting media server metadata and fall back to filename/Arr traits, instead of creating nonexistent combinations. Differing stream order, indices, titles and default flags alone do not conflict. Matching providers can contribute explicit HDR profiles. Details list only active providers. Legacy `embyMatch` and evidence fields remain available internally and in the API for cache/preset compatibility; their combined meaning is media-server matching, not Emby-only matching.

## Matching filters

Media server matching uses checkboxes with the same is/is-not operator as other status filters. Included states use OR with each other; excluded states are omitted, and other filter groups use AND. Available states: Matched, Not found (including missing byte size), Ambiguous, Conflicting media server metadata, Audio tracks missing, Pending and Without enrichment. Choose Matched → is not for the complement; a redundant All unsuccessful matches option is unnecessary. The former Unknown year control is removed; older saved queries remain valid.

## Season outliers

TV series has two checkbox/operator filters with hover, focus and tap-accessible question-mark explanations:

- **Technical outliers** compare resolution, codec family (H.264/H.265 rather than encoder labels), the set of audio codecs and the existing Channels value.
- **Language outliers** compare the complete set of known audio languages, including missing or additional languages relative to the season majority.

Each comparison uses the entire cached series/season, never the currently filtered results. At least three episodes are required. A baseline must have at least two agreeing episodes and represent strictly more than half of all cached episodes in that season. Unknown values do not support a baseline and are not flagged; partial unknown audio/language sets are skipped. Ties and evenly mixed seasons have no inferred baseline. A multi-episode file counts by its distinct associated episode IDs and is shown once. Specials use their separate season-0 group.

Nine 1080p HEVC E-AC-3 5.1 episodes and one 720p H.264 AAC 2.0 episode flag the latter as a technical outlier. Nine German/English episodes and one English-only episode flag the latter as a language outlier. Uniform seasons, or two 1080p episodes and two 720p episodes, are not flagged. Is not means not flagged, and includes unknowns; it does not prove that every physical episode is identical. Only files known to Sonarr are compared, not unavailable future/missing episodes.

Affected file rows and visible season headings are highlighted; file details show the differing attributes and majority values. Indexed `season_analysis` rows are refreshed atomically with full syncs, scoped webhook updates and recognition-rule reprocessing. Match-only lists, full-season context, CSV, presets and custom-tag bulk selections use the same predicates. CSV excludes context-only rows and adds both outlier flags. Schema 14 migrates cached seasons with a consistent pre-upgrade backup.

The first-party implementation references are [Jellyfin ItemsController](https://github.com/jellyfin/jellyfin/blob/master/Jellyfin.Api/Controllers/ItemsController.cs) and [Jellyfin MediaStream](https://typescript-sdk.jellyfin.org/interfaces/generated-client.MediaStream.html).
