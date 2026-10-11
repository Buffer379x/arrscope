# ArrScope 0.3.0 validation

Recorded on October 11, 2026. All service fixtures are synthetic. No production services, media mounts or Unraid installation were changed during validation.

## Automated checks

- Go tests, race detector and vet pass.
- TypeScript and Vite build pass. Nine frontend tests cover catalogs, placeholders, locale selection, tag contrast, dashboard filters, flags and legacy language-selection aliases.
- Frontend dependency audit reports no known vulnerabilities. Go's reachable-code vulnerability scan reports no affected application symbols or imported packages; its module-only advisory is not used by the application. CI also scans runtime images for HIGH/CRITICAL vulnerabilities.
- Recognition tests cover token boundaries, regex captures and validation, zero-width matches, defaults, fallback/override priority, codec aliases, compound profiles, independent conflict types, preview/authentication/CSRF, stale revision protection, rollback after partial processing, cancelled saves, persisted custom rules and sync commits using current rules.
- Schema-4 migration tests retain presets, settings and cached Emby tracks while creating a backup. Saved column layouts migrate to one Audio column. Warning colors persist and invalid values are rejected.
- Filters, CSV and language statistics share cached results. Unique movie/episode counts avoid duplicate files or audio tracks. Multi-episode files, Specials, overlapping languages and unknown values remain explicit. Unknown Emby matching is distinct from failed matching.
- Unattributed audio values remain separate from verified pairs in the unified Audio column and CSV. No language/codec pairing is invented.

## Local performance

Paged queries sort compact row IDs before fetching full evidence and audio-track JSON for the current page. A covering trait index avoids loading media JSON for compound filters. Season context reuses matched IDs and groups once, counts via the group index and pages via the series-order index. Regression tests compare every allowed movie sort in both directions across three pages with the complete result order and check empty/out-of-range season pages with multi-year/Specials context.

Local prerelease candidate on macOS/arm64; Go runtime reports 15 CPUs and uses `GOMAXPROCS=2` for the native benchmark. Dataset: 50,000 files per service; query matrix includes search, numeric ranges, sorting, exclusions, compound all/not-all, both libraries and season context.

| Measurement | Result |
| --- | --- |
| 50k-file filter p50 | 18.99 ms |
| 50k-file filter p95 | 69.78 ms |
| Initial statistics across 100k files | 1.52 s |

Isolated arm64 Docker test of the final memory/pagination candidate: 100,000 synthetic files, two matched Emby audio tracks per file, `GOMAXPROCS=2`, Go memory target at most 128 MiB under the detected 256 MiB container limit, equal memory and memory-plus-swap limits, non-root, read-only root filesystem and all capabilities dropped. The runtime clamps existing higher Unraid targets too; smaller user targets remain respected.

| Measurement | Result |
| --- | --- |
| Successful complete sync | 23.33 s |
| Sampled application RSS peak during sync | 231.44 MiB |
| Application RSS after queries and reprocessing | 45.92 MiB |
| HTTP filter p95 | 20.49 ms |
| Cold statistics request | 2.40 s |
| Cached statistics p95 | 0.80 ms |
| Cold facets request | 0.69 s |
| Cached facets p95 | 0.62 ms |
| Reprocess 100k cached filenames | 16.46 s |
| Upstream requests caused by rule save | 0 |

The cgroup peak reached its 256 MiB limit, including filesystem/database page cache; the process remained healthy and was not OOM-killed. Sampled RSS can miss transient peaks. Cold aggregate requests are slower than indexed library filters and are cached until sync or rule changes. These measurements are environment-specific, not universal latency guarantees. The final CI and release workflow rerun the synthetic container checks on the published candidate.

## Browser checks

- Desktop at 1280×900 and mobile at 390×844, with explicit viewport overrides reset after testing.
- Dashboard and library headings both use 18px desktop / 16px mobile; the Dashboard subtitle is removed.
- Rule add, edit draft, preview, save, sort and delete flow. `UHDBD` fallback recognizes 2160p only when no explicit resolution exists.
- A temporary contradictory test rule produces a recognition warning and a separate metadata warning. Both have distinct icons/colors; the conflict filter finds only matching files and the detail dialog identifies the contributing rules. The contradictory rule was removed after testing.
- Language flags open temporary filters and preserve independent library filters. The unified Audio column shows verified German/DTS and English/AC-3 pairs.
- Filename tooltip is accessible by keyboard focus on the icon before the title; clicking opens file details.
- All three settings tabs fit on mobile and maintain identical dialog geometry. The rule editor and horizontally scrollable rules table do not overflow the page.
- New strings are translated in English, German, Spanish and Italian. Settings tabs and warning labels were checked in the browser in Spanish and Italian.
