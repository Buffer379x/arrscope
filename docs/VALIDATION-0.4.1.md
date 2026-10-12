# Scopearr 0.4.1 validation

## Local checks

- Frontend: TypeScript/Vite build and 12 tests pass, including all four translation catalogs and a rule-ID test with no randomUUID API.
- Backend: complete Go tests with race detection pass. New draft tests cover duplicate-name rollback, valid name swaps, stale snapshots, invalid IDs, dependent deletion cleanup, authentication and CSRF enforcement.
- Browser: English/German language draft remains unapplied until Save; saved language switches the interface. Custom tags are added to a draft and saved together. Duplicate renames show an error toast and do not alter the stored list. Filename Add rule opens the editor; Apply to draft, preview and Save complete correctly. Rule preview maps UHDBD to 2160p using the draft fallback rule.
- Browser: success toasts disappear automatically; errors are displayed bottom-right. Services contain no manual sync buttons; System contains Radarr, Sonarr and Emby actions. Library footer and recognition back button are absent.
- Responsive: verified actual CSS widths 1280 and 390, without document overflow. Settings navigation keeps all six sections visible on mobile. Dark and light card themes use the existing shared color variables.
- Container: built locally for ARM64; preview runs as UID/GID 99:100 with a read-only root filesystem, dropped capabilities, no-new-privileges and a 256 MiB memory limit. Existing synthetic data retained through restart.

## Large-library memory reserve

The first Main and release checks exposed an existing memory-margin issue: the 100,000-file release fixture was OOM-killed at 256 MiB, and the Main run timed out on a cold aggregate. Discarded snapshot/index buffers are now reclaimed between large sync phases; consumed file slices are released before starting the next service. SQLite page caches are bounded per connection and temporary tables use disk. The original memory, timeout and performance checks remain unchanged. The corrected build is verified through the same complete release gates.

## Automated release gates

The protected-branch `tests` workflow and approved-release workflow enforce frontend tests, Go race/vet checks, dependency audits, vulnerability scans, 50,000-file filtering, both container architectures and the 100,000-file Emby/cache-reprocessing check. CI results and image digests are recorded in the delivery report after publication.

No schema migration or production/Unraid update is performed by this release task. Only the local synthetic preview is used for browser validation.
