# Scopearr 0.4.0 validation

- Full local Go suite, race suite and vet passed. New tests cover atomic cross-page assignment, compound custom filters, manual/automatic tag precedence, recovery on the next sync, stable tag renames, catalog deletion, palette cleanup, preset-reference cleanup, unknown rules, zero-width regex matches, CSV selection/order and CSRF enforcement.
- A schema-5 migration fixture retains accounts, legacy sessions, encrypted API keys, presets, media and committed WAL pages. The original database and a pre-schema-6 backup remain available. Column layouts gain Custom without resetting existing choices.
- Frontend production builds and all 11 tests passed; all four locale catalogs have equal keys and valid parameters.
- Browser checks used isolated demo data only. Desktop inspection covered 1280px and mobile inspection covered 390px with no page overflow. Settings sections stay reachable; row actions sit side-by-side; long custom tags remain readable. Multi-valued tags are alphabetic, a partial bulk assignment has a mixed checkbox, mutations refresh list/facets immediately, and filename preview displays custom names rather than IDs.
- Custom colors appear only for used tags. Renames retain color identity; catalog deletion removes its saved background/text colors. Custom mutations preserve unrelated cached Dashboard aggregates.
- Local macOS/ARM64 custom-tag benchmark: 50,000-file combined custom filter p95 83.821 ms; atomic two-tag bulk assignment 243.166 ms. Release CI also executes this benchmark alongside the original 50,000-file gate.
- Docker builds use the existing non-root, read-only, 256MiB/no-swap configuration. Compose configuration validates; new variables retain legacy fallbacks. Default synchronizations skip custom-tag indexing when neither rules nor assignments require it.
- Public release requires protected-branch checks, both image architectures, dependency/runtime vulnerability scans and the 100,000-file Emby/no-swap gate. Historical images are copied unchanged; the previous stable channel is promoted only after approval gates.

No production connection, media scan or Unraid change was made. The user initiates production updates manually.
