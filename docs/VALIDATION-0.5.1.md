# Validation — 0.5.1

Local validation used macOS/Apple Silicon with an ARM64 Docker container. All external requests used isolated synthetic Arr/Emby fixtures; no production library or credentials were used.

- Go race suite and vet passed. Outbound tests cover authoritative replacement of pre-existing assignments, reuse of definitions, Sonarr episode-value unions, custom-tag removal, group opt-out cleanup, endpoint changes, durable recovery after an ambiguous update, and separation of tag errors from completed file snapshots. The write fixture rejects unexpected payload fields or endpoints.
- DTS comparison and subtitle-language tests passed, including strict customized rules and schema-6-to-7 migration preserving edited defaults.
- All 13 frontend tests and the TypeScript/Vite build passed. Translation catalogs have matching keys and placeholders. Rule search combines terms across table columns without changing the stored rule set.
- Browser checks confirmed the search/Add/restore toolbar at desktop and 390-pixel width, counts below the table, full Emby matching status display and opt-in tag controls. Demo connections remain locked.
- The unchanged isolated container check completed with 50,000 movie files and 50,000 episode files, two Emby audio tracks per file, a read-only filesystem, unprivileged UID/GID and 256 MiB memory/no swap. No OOM occurred.

The first release attempt was stopped by the unchanged ten-second cold Dashboard request timeout on the hosted AMD64 runner, after both file syncs completed without OOM. Compact expression indexes now serve episode identities, completeness and JSON validity without reading the large evidence/audio payload. Corruption and rollback tests remain passing; no timeout or memory gates were relaxed.

Final local results with the index: sampled application peak RSS 212.48 MiB, post-query RSS 35.56 MiB, HTTP filter p95 21.403 ms, cold Dashboard 1144 ms, cached Dashboard p95 1.209 ms, recognition reprocessing 14735 ms. The initial local cold Dashboard measurement was 2305 ms. Cached Emby tracks survived reprocessing; reprocessing issued zero upstream requests with tag synchronization disabled. Cgroup peak reached the 256 MiB limit (including reclaimable filesystem cache); RSS is reported separately. These are fixture measurements, not guarantees for every host or library.

Pull-request and release workflows additionally repeat version consistency, dependency/security checks, race tests, query benchmarks, container validation and architecture-specific image checks before promoting stable. Production Unraid updates remain user-triggered.
