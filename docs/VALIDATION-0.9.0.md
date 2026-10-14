# Scopearr 0.9.0 validation

- Go tests, including race checks, and go vet passed locally.
- All 28 frontend tests and TypeScript checks passed.
- Regression tests cover legacy defaults, explicit empty widget visibility, reordered-widget persistence after restart, invalid settings, paginated matches, full-season context, and complete exports/bulk selections.
- HTTP tests confirm that list requests use the saved page size, cannot override it through query parameters, and do not truncate CSV exports.
- Browser checks use local synthetic data: widget visibility/reordering, saved page size in both libraries, next page, reset confirmation, and responsive settings.
- The non-root ARM64 container completed a 100,000-file synthetic Radarr/Sonarr sync with Emby and Jellyfin enabled under a 256 MiB memory limit without swap. Peak sampled application RSS: 249.12 MiB; idle after queries: 47.82 MiB; HTTP filter p95: 24.159 ms. No OOM occurred.
- Real external services were not modified. GitHub CI and the release workflow repeat security, performance and container gates before publication.
