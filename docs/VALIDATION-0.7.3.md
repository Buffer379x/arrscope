# Scopearr 0.7.3 validation

Validated on 2026-10-14 using ARM64 macOS/OrbStack and isolated synthetic services. Required CI repeats checks on Linux AMD64 before merge and release. Production Unraid and Arr settings remain unchanged.

## Automated checks

Full Go race suite and vet pass. Regression coverage verifies retained presentation values/colors after rule removal, linked-tag deletion rejection, composite/partial-unknown counts, stable deletion markers and reappearance only after a recognized match, immutable external identities during display-name edits, manual/external management transitions, visible-name sorting, hourly slots before the configured anchor, shared-storage names without changed grouping/capacity, and schema 12→13 backup and persistence. Earlier deletion tests now explicitly remove assignments before deleting definitions.

All 27 frontend tests, TypeScript and production build pass, including catalog key/parameter parity. Version/package/Docker/English notes agree. Local govulncheck reports no reachable vulnerabilities; one required-module advisory is not called by the application. CI additionally runs dependency and runtime-image security scans.

## Load check

100,000 files (50,000 per service), two Emby tracks per file, 256 MiB with no swap, read-only root, dropped capabilities and GOMAXPROCS=2:

| Measurement | ARM64 local result |
|---|---:|
| HTTP filter p95 | 25.236 ms |
| Cold dashboard | 1293.593 ms |
| Cached dashboard p95 | 0.713 ms |
| Full presentation catalog/counts | 1555.457 ms |
| Sampled sync RSS peak | 220.60 MiB |
| RSS after queries | 48.90 MiB |

No OOM occurred. Count aggregation uses one grouped trait query, not repeated library scans for every retained value. The strict existing memory and request deadlines remain unchanged. Recognition cache reprocessing preserves Emby metadata and makes zero upstream requests. CI repeats the catalog-count assertion under the same limit.

## UI checks

Desktop and 390×844 mobile review: service sync actions are beside service headings and outside the tag-check block; incoming/outgoing sections share equal full-width layout. Counts precede color controls, linked/protected trash icons are disabled, and tables scroll on narrow screens. Auto-tag display-name edits persist in the library and CSV while the synthetic upstream name remains lowercase and no upstream writes occur. Storage names save immediately after the explicit settings save and appear in the dashboard without capacity/group changes. Library navigation collapses both filters and preserves the visible search value.

The existing local preview also migrated schema 12→13 with all 26 movie and 122 series files retained. Stored settings, recognition and account values were byte-identical; the pre-migration backup matched the original database. Version 0.7.2 started healthy with that backup and the same encryption key.

## Semantics

Counts refer to Scopearr file entries across both services, not external profiles/rule references or series counts. Active external auto rules still protect deletion at zero file count. Display names are local captions; canonical identities remain unchanged. Presentation deletion removes its saved colors, not recognition rules or outbound categories; a later recognized match restores a default entry. Shared-area labels use Radarr before Sonarr in the combined scope, and their own service name in each service scope.
