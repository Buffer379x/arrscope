# ArrScope 0.3.2 validation

- Full Go race suite and `go vet ./...`: passed. Additional manual Emby synchronization test passed with CSRF enforcement, both enabled library jobs, overlap rejection and GET-only requests.
- Frontend: 11 tests passed, TypeScript check and production Vite build passed. Four locale dictionaries have identical keys and compatible placeholders.
- Preset normalization tests cover value ordering, duplicate values, language aliases, numeric ranges, unused/default operators and display-only parameters. Sixteen concurrent saves create exactly one preset; legacy records remain unchanged.
- Browser inspection against isolated synthetic demo data: unchanged 42px search height when collapsed, checkbox multi-selection for 1080p/2160p, inactive status checkbox after operator changes, duplicate-preset warning, 40px compact color controls, persisted text colors and reset confirmation.
- Settings are a full page; the header gear remains between theme and sign out. At 390px width all settings sections, including System, remain reachable without page overflow. Audio rows retain tracks and file details with the mapping information button removed.
- Local read-only, non-root container build completed. No production connections or Unraid changes were used during validation.
- Release gates also require Go/frontend tests, both container architectures, vulnerability scans and the established 100,000-file Emby memory/HTTP stress check before stable promotion.

Database schema remains 5. Production upgrades are initiated manually in Unraid.

## Additional release memory finding

The first release candidate passed sync and query checks but triggered an OOM event during the subsequent full-cache filename-rule reprocessing. Stable promotion and GitHub release publication were blocked. The corrected candidate reserves three quarters of a detected container limit for SQLite/process allocations and page cache (64 MiB Go soft target under 256 MiB), including legacy higher GOMEMLIMIT settings. Reclaimable Go heap pages are explicitly returned before backup and reprocessing, outside ordinary query paths. The same complete required checks and release gates must pass on this corrected source.

Full-cache recognition updates reuse a prepared update statement and retain unchanged indexed language associations. Their authenticated request budget is bounded at 180 seconds to accommodate large caches on slower hosts; ordinary HTTP query limits and the 256 MiB/no-swap stress limit remain unchanged. Tests verify language index changes, preservation and removal, cancellation and atomic rollback.
