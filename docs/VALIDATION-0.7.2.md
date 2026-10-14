# Scopearr 0.7.2 validation

Validated on 2026-10-14 using macOS/OrbStack ARM64; CI repeats checks on Linux AMD64 before merge and release. Test services and credentials are synthetic. Production Unraid and Arr settings are unchanged.

## Automated checks

- Full Go race suite and `go vet` pass. External-tag tests cover ordinary imports, durable removal intents, re-additions, series-wide external assignments, preserved local episode granularity, external removals, global deletion, concurrent rename/deletion review, retained local additions after remote definition deletion without stale-ID replay, auto-rule protection, rule-reference removal, source-scoped protection, server URL changes and undo during an in-flight write.
- Failed and null catalog reads and unreadable referenced rules retain the prior snapshot and block writes. Backend protection rejects editing/removing protected tags even when UI restrictions are bypassed.
- All 27 frontend tests, TypeScript checks and production build pass, including four-catalog key/parameter parity. Version, package, Docker defaults and English release notes agree.
- Local `govulncheck` reports no reachable vulnerabilities; one required-module advisory is not called by this application. Required CI includes dependency audit and a high/critical runtime-image scan.

## Container load test

100,000 synthetic files (50,000 per service), two Emby tracks per file, `GOMAXPROCS=2`, a 256 MiB container cap with no swap, a read-only root and dropped capabilities:

| Measurement | ARM64 local result |
|---|---:|
| Filter HTTP p95 | 25.112 ms |
| Cold dashboard | 1327.638 ms |
| Cached dashboard p95 | 0.666 ms |
| Sampled sync RSS peak | 217.91 MiB |
| RSS after queries | 48.68 MiB |
| Rule-cache rebuild | 33.56 s |

No OOM or container restart occurred. Recognition preserved Emby metadata and made zero upstream requests. Cold aggregate requests retain their existing ten-second deadline; no memory/deadline gates were relaxed. CI repeats this resource test on the final source.

## UI and upgrade checks

- Audio, Language and Storage collapse to the same 468 px card height. Expanding Language increased only that card to 754.5 px. The storage donut, arrows and used/free values remain inside their card. Area selection survives browser reload.
- Auto-tag gear opens actual service/rule names in settings and the library. Protected names/deletion/assignment controls are disabled, ordinary assignments remain editable, and auto-tag text-color changes save explicitly.
- System shows tag-check status and queues manual checks. Mobile review at 390 × 844 keeps navigation available and custom-tag controls in a horizontally scrollable table. Light/dark palettes retain the existing controls and color contrast logic.
- Existing local preview migrated schema 11 → 12, retaining 26 movie and 122 series files and exact stored settings, recognition and account values. The pre-migration backup matches the prior database. Version 0.7.1 started healthy with that backup and the same local encryption key.

## Practical limits

Incoming tag checks cannot be truly event-driven because ordinary Arr tag edits do not emit Connect webhooks. An auto-rule reference cannot establish the historic creator of an individual assignment. Sonarr assignments are series-level. Imported custom catalogs share the 128-tag limit. Failed imports are atomic; Arr writes across requests cannot be atomic, and concurrent changes are checked again on subsequent jobs. Storage areas remain separate because overlapping Unraid shares cannot be summed from Arr mount readings.
