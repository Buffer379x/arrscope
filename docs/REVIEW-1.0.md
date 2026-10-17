# Pre-1.0 review and hardening

The review of 0.9.5 was repeated before implementation. Its five primary findings were reconfirmed, and additional failure paths and accumulated code were identified. Version 1.0.0 addresses those findings while retaining the existing interface, endpoints, media recognition semantics and feature set.

## Correctness and reliability

| Finding | Change |
| --- | --- |
| Configuration/tag-import lock inversion | Serialize settings writes separately; keep database/recognition work outside the configuration-reader mutex. Acquire the shared sync gate before the settings-write lock when both are required. |
| Read queries reserving a SQLite writer | Use explicit read-only transactions for library counts/results and dashboard statistics. |
| False logout success on deletion failure | Check session revocation before expiring the browser cookie or returning success. |
| Silent fallback on corrupt stored settings | Treat missing entries as initial setup; report read/decoding errors and retain existing data. Validate existing settings/rules before historical migrations. |
| Incomplete Sonarr episode validation | Share identity/file/series validation and file mapping between full and delta synchronization; reject conflicting snapshots and retain the previous cache. |
| Concurrent recognition-engine access during outbound reconciliation | Pass a protected immutable rules snapshot into transaction work. |
| Corrupt recognition JSON replaced during migration | Fail the migration instead of writing default rules over the existing entry. |
| Special characters changing the SQLite database location | Escape the database path as a file URI and establish private file permissions before SQLite opens it. |
| Legacy `mediaMatch` suppressing Emby filter validation | Remove the orphaned bypass; continue validating and applying the supported `embyMatch` filter. |

Snapshot writes now carry cancellation contexts through SQLite. Status readers and cancellation do not wait for the entire rewrite or memory reclamation. The final commit still checks the configuration generation under the status lock. Immutable recognition engines are published through an atomic pointer, so the authenticated status endpoint can read their committed revision without waiting for recognition work. Display-only settings changes do not wait for the global file/tag gate.

Password verification uses a small bounded capacity and short limiter-state locks. Existing per-address throttling and the password-hash check at session creation remain in place.

## Cleanup and performance

- Remove the unused parser expressions/helper, cleanup helper, unused TypeScript imports/error states/exports and the unreachable old recognition main-view branch. Test-only compatibility helpers live in test files; production endpoints remain available.
- Share custom-tag deletion cleanup for rules, colors, hidden tags and presets instead of maintaining two implementations.
- Split endpoint handlers, CSV handling, store opening/migrations, query filters/order/results, facets and statistics into focused files. Separate settings connections/appearance, library table rendering and rule editing into existing-markup components. Keep historical rules migrations explicitly identified.
- Remove 39 unused CSS rules (41 selector arms), retaining active arms of grouped selectors. Move the presentation rules into their own stylesheet with the original cascade order.
- Remove the unused duplicate `icon-source.png` from the public build: 1,156,447 embedded bytes saved. The displayed icon remains unchanged.
- Load the authenticated application, settings and recognition page on demand. Keep the complete local language registry and all translations.
- Cache the read-only tag catalog and maintain historical entries within data/rule writes. Pin matching rules and data together. Use separate aggregate-fill locks and generation checks so invalidated results cannot repopulate caches.
- Reuse returned settings/custom-tag snapshots, remove duplicate requests, discard outdated responses and tolerate blocked/full browser preference storage. Existing API notifications continue to report failures.
- Use HTTP status constants. Preserve existing English error identifiers because they are translation keys. Staticcheck's correctness, simplification and unused-code checks are part of both CI and release validation; general style diagnostics are excluded.

## Validation and visual compatibility

Targeted Go regression tests cover read availability during writes, corrupt configuration preservation, database path/permissions, failed and successful logout, lock ordering, full/delta episode identities, cache purity/isolation, cancellation, generation-checked rollback, concurrent reconciliation/custom-tag writes, display-only saves and password-work capacity.

The release gates include all frontend tests, strict TypeScript with unused-code checks, production builds, Go race/vet/static checks, dependency and runtime image scans, 50,000-file performance tests, both image architectures and the 100,000-file container test. The container test also verifies warm catalog responses and concurrent cold statistics/facets/catalog reads after rule reprocessing under the same 256 MiB/no-swap limit.

Separate synthetic 0.9.5 and 1.0.0 instances were compared at 1280 × 720 and 390 × 844. Dashboard, movie/TV libraries, light theme, connection/display settings and mobile library/display screenshots match pixel-for-pixel once loaded. Library/filter markup also matches. Dialogs and rule tables retain their appearance; the login/System version label reflects 1.0.0. Mobile settings and rule tables do not widen the page.

No schema version or credential format changes are introduced. Keep appdata and its local key; see [operations](OPERATIONS.md) and the [1.0.0 release notes](releases/1.0.0.md).
