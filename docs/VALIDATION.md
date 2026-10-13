# Validation and measurement boundaries

The initial release was tested locally and in GitHub Actions, then installed on Unraid. Documentation records synthetic test conditions separately from actual production observations. User library data, personal installation profiles and credentials are not published.

## Automated coverage

Go race tests and vet cover filename parsing/provenance, codec families and token boundaries, unknown metadata, compound audio/HDR/language sets, include/exclude/all/not-all operators, FTS/parameterization, grouped seasons, specials, multi-episode files, pagination, CSV/preset equivalence, sorting, statistics/deduplication, login/CSRF, key encryption/masking, rate limiting, sessions/reset races, restart, backup retention, migrations/restore and future-schema rejection.

Catalog/Emby mock tests enforce allowlisted GET-only endpoints and correct key headers. Separate tag-worker fixtures enforce the permitted tag/editor mutations, exact payload boundaries, persistent policy choices, safe unused-definition deletion and reference protection. Rejected keys, 5xx, redirects, timeout/cancellation and invalid responses preserve complete cache state. Emby tests cover pagination/totals, MediaSources, exact byte sizes, duplicate/ambiguous sources, per-track metadata, external audio/subtitle exclusion, unknown tracks, path matching and persistence.

Frontend checks include TypeScript/Vite builds, tag contrast and invalid styles. Version 0.2.0 adds complete catalog/placeholder checks, translation key validation, locale fallback/interpolation, display-only language aliases and encoded dashboard drilldowns. Browser checks complement these tests; see the version-specific validation record when present.

## Initial local browser verification

Synthetic demo: 26 movies, 5 series, 123 episodes and 148 files. Checks covered positive/negative/compound filters, both season modes, CSV/presets, column order/visibility, tag colors in both themes, dashboard scope, sorting persistence, mobile width 390 px, keyboard Enter/Escape and focus return. Existing presets/colors/settings survived container restarts. Multi-codec, multi-HDR and Emby language-to-codec examples were synthetic.

Historical screenshots include [desktop](screenshots/desktop.jpg), [series](screenshots/series.jpg), [mobile](screenshots/mobile.jpg), [light theme](screenshots/light.jpg), [compound audio](screenshots/compound-audio-tags.png) and [Emby tracks](screenshots/emby-audio-tracks.png). Older images show the original German interface.

## Historical local performance

macOS/ARM64, 15 reported logical CPUs, local SQLite, Go 1.27.2, `GOMAXPROCS=2`, `GOMEMLIMIT=192MiB`. Earlier native 100,000-file mixed-query test: p50 49.0 ms, p95 108.7 ms. A separate 50,000-file mock sync took 2.22 seconds; HTTP `1080p + x264` p95 28.8 ms, sampled process peak 207.0 MiB. After restart and one query, peak 26.8 MiB. A 50,000-movie test with two Emby tracks per file took 3.244 seconds, HTTP p95 40.8 ms, peak 219.9 MiB and 27.0 MiB after restart.

The later local no-swap container test and its memory/headroom limits are documented in [review fixes](REVIEW-FIXES.md). These measurements used synthetic mock APIs; fixture processes are excluded from application RSS. Short post-restart observations are not long-term idle measurements. Some workloads overlapped, so they are not isolated hardware benchmarks.

## GitHub release 0.1.0

Checks and release jobs passed, including both image architectures and runtime high/critical vulnerability scans. On a shared Linux/AMD64 runner with four available CPUs and `GOMAXPROCS=2`, the native 50,000-file test achieved p50 84 ms and p95 168 ms, meeting the 200 ms target for that test.

The separate 100,000-file container workload (50,000 per service, two Emby tracks each) completed without OOM or swap. HTTP filter p95 was 303 ms in Checks and 525 ms in the release run; cold dashboard aggregation took approximately 5.5 and 7 seconds. Release sync: 33.437 seconds; sampled RSS peak 210.07 MiB. RSS remained 202.1 MiB after release-run queries, which is not a measured permanent idle state. Runner load, disk and architecture affect results; the 100,000-file HTTP test is not claimed to meet 200 ms.

Reproduce native queries with `GOMAXPROCS=2 SCOPEARR_PERF=1 go test -run TestPerformance50K -count=1 -v ./internal/app`, host mock workloads using `docs/scripts/loadcheck.py`, and isolated containers using `docs/scripts/containercheck.py`. The scripts create their own synthetic state; they must not be pointed at production Appdata.

## Production smoke checks

The actual Unraid installation connected successfully to Radarr 6.4.4.10685, Sonarr 4.0.20.3014 and Emby 4.10.1.0. Initial sync, real combined-filter samples, season grouping, dual HDR tags and per-track language/codec display were checked. A targeted Scopearr restart preserved authentication, cache and saved connections; all three connection tests still passed. Container health and autostart were confirmed.

This smoke test is not a production performance benchmark. Some real files retain incomplete upstream metadata or unmatched Emby identity; no scans were triggered to repair these. Real update/rollback, long-term memory, arbitrary library growth and different HTTPS/proxy configurations remain separate operational checks. A host swap-limit capability warning was observed; RAM limiting remained active, but swap enforcement was not confirmed on that host.

## Version 0.2.0 local review

Go race tests and vet, seven frontend tests, TypeScript/Vite production build and version consistency checks passed. An additional HTTP regression test verifies translated JSON errors and Italian CSV headers without changing canonical language values. Browser review covered German, English, Spanish and Italian live switching, language persistence after reload, translated login, desktop and 390 px mobile layouts, dashboard drilldowns (including unknowns), combined-scope movie/TV switching, restoration of prior filters, text-based language filtering and keyboard flag tooltips. Existing demo presets and custom columns/colors survived the container replacement.

The local ARM64 Docker test used 100,000 synthetic files (50,000 per service), two Emby tracks per file, a 256 MiB memory limit with no swap, GOMAXPROCS=2 and GOMEMLIMIT=192MiB. It completed in 9.552 seconds without OOM. HTTP filter p95: 55.167 ms; cold statistics: 1,526.764 ms; cached statistics p95: 0.847 ms. Sampled app RSS peak: 213.31 MiB, after queries: 35.11 MiB. Cgroup memory including page cache reached 256 MiB, so this does not establish spare memory headroom for larger workloads. Tests and fixture servers shared the local machine; these are synthetic measurements, not production guarantees.

Frontend dependency audit reported no known vulnerabilities. Go symbol analysis found no reachable vulnerabilities (one advisory in an unused module path); the final ARM64 runtime image scan found no high or critical vulnerabilities. These are scan results at review time, not a guarantee against all flaws. Production remains on 0.1.0 until approval and release; the new version has not yet been validated against the user's live installation. Spanish and Italian translations have structural/browser coverage but have not received independent native-speaker review.

[English 0.2.0 dashboard screenshot](screenshots/dashboard-0.2.0-en.png) uses synthetic demo data only.

Version 0.3.0 adds configurable recognition, atomic cache reprocessing, separate conflict states, Emby matching filters and unique language statistics. See the [0.3.0 validation record](VALIDATION-0.3.0.md). The synthetic container check also verifies rule reprocessing without upstream requests and retention of per-track Emby data.

See [0.5.3 validation](VALIDATION-0.5.3.md) for removal previews, rule presentation, schema 9 and the final container fixture.
