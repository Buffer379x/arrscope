# Review fixes from the initial release

Recorded on October 11, 2026 for version 0.1.0. These fixes are included in subsequent versions. This is historical validation, not a new performance benchmark.

| Area | Result |
| --- | --- |
| Dependencies | Preact patched to 10.27.3; matching lockfile. |
| Upstream data | Reject null, invalid trailing JSON, oversized responses, invalid/duplicate IDs and inconsistent file associations. Valid empty lists remain allowed; errors preserve the complete cache. |
| Sync/memory | One global queue; visible cancellable queued jobs. At most two concurrent Sonarr requests for a single series. Container RAM and requested swap limits. |
| Queries | Shared read snapshots, SQLite cancellation, defensively copied facet/dashboard caches with commit invalidation. |
| Browser requests | Stale response protection and paused hidden-tab polling; 3/30/60-second sync/idle/login polling. |
| Languages/HDR | Canonical sorted language sets; explicitly reported Arr languages supplement filenames. Emby video HDR works without audio streams. Partially unknown attributes remain filterable. |
| Data quality | Completeness filter, file warnings and dashboard counts. Missing HDR alone is not a failure; negative filters refer to reported metadata. |
| Emby identity | Optional full-path mappings in addition to unique basename and size. Ambiguity stays unmatched; no scans/media access. |
| Sessions/keys | Atomic password change/session revocation and password-hash checks against concurrent reset races. Explicit saved-key deletion; blank inputs preserve keys. |
| CSV | Same filter/sort as the list; protected complete temporary export before download; corrupt data/write failures return errors. |
| Maintenance/design | Components/hooks/style modules, three settings sections, reachable Save bar, readable mobile evidence cards, Escape/focus handling and accessible labels. |
| Colors | Actual luminance determines black/white text; 4,096 combinations tested, including the `#777777` contrast boundary. |
| Delivery checks | Race/vet, frontend tests/audit, govulncheck, runtime vulnerability scan and 100,000-file synthetic no-swap container test. |

Large dashboard sorts initially exhausted the small read-only container’s temporary space. `SQLITE_TMPDIR=/data` now provides local Appdata temporary storage while retaining the 32 MiB `/tmp` and read-only root. Keep free disk space for sorts and CSV.

## Historical measurements

Local Docker/OrbStack Linux/ARM64 on SSD, Go 1.27.2, `GOMAXPROCS=2`, `GOMEMLIMIT=192MiB`, RAM and RAM-plus-swap limits of 256 MiB. Synthetic mock services ran separately on the host. Native SQLite tests ran on macOS/ARM64 with 15 reported logical CPUs.

| Measurement | Result |
| --- | ---: |
| Native SQLite, 100,000 mixed files | p50 44.5 ms; p95 98.9 ms |
| Container, 50,000 movies + 50,000 episode files, two Emby tracks per file | Sync 10.04 s; no swap/OOM |
| HTTP filters after sync | p95 43.4 ms |
| Dashboard cold/cached | 1,576.6 ms / p95 0.85 ms |
| Facets cold/cached | 561.8 ms / p95 0.92 ms |
| Sampled app RSS peak during sync | 207.9 MiB |
| App RSS after queries | 34.8 MiB |

The cgroup peak reached 256 MiB and includes more than process RSS. The test passed with limited headroom; this does not guarantee arbitrary library sizes. In-memory snapshots remain a potential memory constraint for much larger or richer datasets. Batched SQLite staging is a future option. Reproduce with `python3 docs/scripts/containercheck.py --image scopearr:local` using a separately built image; it creates and cleans up its own synthetic container.

Race tests, vet, TypeScript, frontend tests and both architecture healthchecks passed. Local AMD64 was emulated. Synthetic regression tests cover malformed response cache preservation, queue cancellation, language sets, unknown tracks, cache invalidation, contexts, CSV sorting/corruption, key removal, path matching, migrations/backups, reset races and existing read-only/security/filter behavior. Preview data, four presets and custom colors survived the local container replacement.

At the recorded check, pnpm audit found no known vulnerability; govulncheck found no reachable known vulnerability (an unused OpenPGP module advisory remained); Trivy found no high/critical vulnerability in the checked runtime image. These are scan results, not a complete security certification.

Historical synthetic screenshots: [dashboard](screenshots/review-dashboard.png), [mobile metadata](screenshots/review-mobile-metadata.png), [dark mobile metadata](screenshots/review-mobile-metadata-dark.png), [mobile maintenance](screenshots/review-settings-mobile.png). See [current validation](VALIDATION.md) for later CI and production observations.
