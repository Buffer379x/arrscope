# Scopearr 0.8.0 validation

Validated on 2026-10-14 with ARM64 macOS/OrbStack and isolated synthetic Arr, Emby and Jellyfin APIs. Production Unraid and media services were not changed. Required Linux AMD64 CI and the release workflow repeat tests, scans, both-architecture smoke tests and the dual-provider memory check.

## Automated checks

Full Go race suite, vet, TypeScript, the production frontend build and all 28 frontend tests pass. New regression coverage checks Jellyfin base subpaths, authenticated GET-only endpoints, explicit HDR profiles, provider agreement/conflict/missing behavior, independent encrypted credentials, reprocessing retained track snapshots, rejection of a failed second-provider snapshot, strict season majorities, unknown values, ties, multi-episode weighting, negation, OR/exclusion matching states, presets, context counts and scoped webhook replacement. Earlier migration checks now expect schema 14. CSV still rejects malformed cached records instead of returning a partial successful file.

The strict 50,000-file season-outlier performance test is included in the existing performance CI command. Match-only p95: 20.860 ms; full-season context p95: 26.944 ms. The existing general library checks also pass below the 200 ms target; custom-tag query p95 was 87.615 ms on the same environment.

## Container check

100,000 files (50,000 per Arr service), simultaneous Emby and Jellyfin with two matching audio tracks each, read-only root, capabilities dropped, non-root execution, GOMAXPROCS=2, 256 MiB including swap disabled:

| Measurement | ARM64 result |
|---|---:|
| Sync sampled RSS peak | 237.44 MiB |
| HTTP filter p95 | 24.009 ms |
| Cold dashboard | 1257.614 ms |
| Cached dashboard p95 | 0.808 ms |
| Cold facets | 983.395 ms |
| Presentation catalog/counts | 2631.420 ms |
| RSS after queries | 44.79 MiB |
| Recognition reprocessing | 44.612 seconds |

No OOM occurred. The first implementation failed the unchanged memory gate; compact per-provider records, elimination of an intermediate result map and reuse of normalized track slices corrected it. No memory limit or deadline was relaxed. Recognition reprocessing preserves verified tracks and makes zero upstream calls. The public `containercheck.py --jellyfin` option and CI/release invocations cover both providers.

## Browser checks

Desktop: external/outbound sections have equal widths and sit side by side; narrow screens stack them. The media-provider selector preserves separate URLs and enabled states. Synthetic ten-episode season filters return only E04 for both outlier types and retain matching CSV query parameters. The question-mark popup opens on click/focus, closes on blur/Escape, and explains the complete-season majority. A discovered focus/click toggle bug was corrected. File details retain individual provider states and majority attributes.

390×844: settings document width stays 390 px, tag sections stack, both help popups fit within the viewport, and filter labels/operators remain usable. Temporary viewport overrides were reset.

Emby-only and Jellyfin-only settings were saved separately; System and file details show only the enabled provider. German final-preview Dashboard, column headers and filter descriptions use Quelle. The persistent local preview runs version 0.8.0; it retains its previous data.

## Migration and practical limits

Actual schema 13→14 migration retained 26 movie and 122 episode files. Settings, recognition configuration and the account record stayed byte-identical. A consistent backup matched the original database, and 0.7.3 started healthy with that backup and its original encryption key.

Jellyfin was validated against first-party API definitions and controlled API fixtures, not a user-provided live Jellyfin installation. Missing/stale data in a media server cannot establish the physical absence of a language. Majority-based outliers intentionally leave ties and unknowns unclassified. Selected storage names are captions; shared-group identities still prevent duplicate capacity readings.
