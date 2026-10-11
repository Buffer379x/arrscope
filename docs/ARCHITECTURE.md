# Architecture and HTTP API

Go serves embedded Preact/TypeScript assets and JSON endpoints from one origin. SQLite via `modernc.org/sqlite` requires no CGO. `/data` is the only persistent mount. No external fonts, telemetry, CDN assets, media mounts or separate database server are required.

## Storage and metadata

`files` stores service/file IDs, media ID, series/season associations, indexed normalized fields and the full file record as JSON. `years` supports multiple episode years. FTS5 `search` indexes title, series and filename. `file_languages` indexes normalized language membership, including partially unknown tracks. `kv` holds settings, filename-recognition rules, account and sync status; `presets` and `sessions` are separate tables. `PRAGMA user_version` tracks schema changes; backups precede existing-database migrations and unknown future schemas are rejected.

Each attribute’s `evidence` retains normalized filename, Arr API and optional Emby values, selected origin (`filename`, `rule`, `api`, `emby`, `combined`, `unknown`) and conflict. Original filenames are preserved; not every raw upstream response is archived. Titles/years come from Radarr; episode names, seasons and first-air years from Sonarr. Existing quality and MediaInfo attributes supplement filename tokens; ArrScope never invokes media analysis. Pixel dimensions alone do not establish a quality class.

Audio, language and HDR attributes are component sets. Codec aliases share a family without collapsing explicit encoder distinctions. Additional explicitly reported languages enrich filename languages without inventing a per-track codec. Unknown track attributes set `unknownTraits`; `metadataIncomplete` considers resolution, codec, source, audio, language and failed configured Emby matching. Missing HDR alone may mean SDR or unknown and does not make metadata incomplete. Negative filters describe missing **reported** attributes, not proven absence.

## Sync

Radarr loads the movie catalog, then movie files in batches of 100 movie IDs. Sonarr loads the series catalog, then episodes/files per series with at most two concurrent GET requests; series are processed sequentially. One global queue builds one service snapshot at a time. Each HTTP request has a 20-second timeout and up to three attempts for network/5xx errors. Redirects are rejected to prevent key forwarding.

Data is gathered in memory, validated and committed only after a complete successful fetch. A transaction replaces the service files, search index and year mappings atomically. WAL readers observe a complete state; pagination counts and rows share one read transaction. Settings/dispatch are locked together; changed configuration cancels old generations and prevents stale commits. A multi-request upstream catalog is not a transactional snapshot, so detected identity inconsistencies fail and retain the previous cache.

Facet and dashboard caches are defensively copied immutable snapshots, invalidated after successful commits. Queries carry cancellation contexts to SQLite. Frontend request generations and abort controllers reject stale results. Polling pauses in hidden tabs; intervals are 3 seconds during sync, 30 seconds when idle/authenticated and 60 seconds on login.

## Endpoints

| Method/path | Purpose |
| --- | --- |
| GET `/healthz` | Public process health and version |
| GET `/api/status` | Setup/auth state; authenticated CSRF and sync status |
| POST `/api/setup` | One-time admin setup with token |
| POST `/api/login`, `/api/logout` | Create/revoke session |
| GET/PUT `/api/settings` | Masked settings / encrypted persistence |
| POST `/api/test` | Read-only upstream system-status test |
| POST `/api/sync` | Queue asynchronous service sync |
| GET `/api/files` | Paginated matches/season context |
| GET `/api/statistics` | Counts/distributions from one cached snapshot |
| GET `/api/facets` | Available values in the entire service cache |
| GET/POST/DELETE `/api/presets` | Read/create/delete named presets |
| GET `/api/export` | Complete CSV of actual matches |
| PUT `/api/password` | Change password and revoke sessions |
| POST `/api/backup` | Consistent local database snapshot |

Except public status, health, setup and login, endpoints require a session. Mutations require `X-CSRF-Token`; foreign Origin headers are rejected. Cookies are HttpOnly, SameSite Strict, valid for 24 hours, and Secure when TLS/HTTPS is configured. API-key response fields stay empty; only `hasKey` indicates presence. Upstream bodies and keys are never exposed in errors. Password writes and session invalidation are transactional; session creation verifies the authenticated password hash again to prevent delayed old-password sessions after a concurrent reset.

## Queries and CSV

Files/export share parameterized filtering: `service=radarr|sonarr`, `q`, `yearMin`, `yearMax`, `resolution`, `codec`, `family`, `source`, `processing`, `hdr`, `bitDepth`, `audio`, `channels`, `language`, `sizeMin`, `sizeMax`, `unknownYear=true`, `unknownSize=true`, `conflict=true|false`, `metadata=complete|incomplete`.

Multiple values of an ordinary field use OR; different fields use AND. `<field>Op=eq|not` includes or negates the entire selected set. Audio/HDR/language also allow `all|notAll`, combining component conditions with AND or negating that complete combination. Invalid, repeated or inappropriate operators are rejected; an operator without values has no effect. `__unknown__` selects empty or partially unknown audio/language attributes. Sizes are bytes, displayed as GiB. FTS search uses sanitized AND-prefix terms, not arbitrary substrings or user FTS syntax.

Files use `page>=1`, 100 rows per page and optional `mode=matches|season`. `total` counts actual matches, `displayTotal` includes context and `matched` identifies real matches. Series sort by title, season and episode. Movie sort allowlist: title/year/resolution/codec/source/size/hdr/bitDepth/audio/channels/language, with `dir=asc|desc`. Default: title ascending. Numeric attributes sort numerically; unknowns stay last.

CSV applies the same filters/sort, UTF-8 BOM, semicolon separator and spreadsheet-formula safeguards. A protected temporary Appdata file completes before download; SQL/JSON/write errors fail rather than deliver a partial success. Cancellation/success removes temporary files. Context-only rows are excluded. Headers use `Accept-Language` or `lang`; media values stay canonical.

## Emby

Emby config has enable/URL/encrypted key and `interval=0`; it joins each Arr service sync. The client permits only GET `System/Info` and paginated `Items` using `X-Emby-Token`, 250 items per page. Subpaths are supported. Totals, page completeness and duplicate IDs are validated; only relevant candidates stay in memory.

Identity requires a unique exact basename and positive byte size on both sides. Multiple versions use the matching MediaSource. Item-level streams belong only to the exact matching source path. Optional absolute prefix mappings require the full mapped path too; longest Arr prefix wins. Unmatched/ambiguous identities remain explicit. Metadata paths are never opened and Arr source paths are not returned to browsers.

Only internal audio streams count: index, explicitly reported language, codec/profile, channels/layout, title and default flag. Languages are never inferred from track titles. Unique Emby tracks provide audio/language sets; if all values of an attribute are unknown, existing filename/Arr data remains without assigning it to tracks. The main Channels column retains filename/Arr data; details show each track. Explicit Emby video HDR supplements reported tags; profile numbers never imply an invented HDR10 fallback. Emby failures abort the enriched service snapshot.

## Presentation, translations and versioning

Settings store visible columns and the complete eleven-column order independently per service. Title/episode remains visible. Unknown/duplicate/missing columns and invalid services are rejected. Header/cells share one ordered list; temporary “All columns” does not change defaults. Tag backgrounds are validated hex values with automatic text contrast. Optional text overrides accept only black or white; both maps persist in the existing settings JSON without a schema migration. Filter presets remain canonical SQLite data independent of display language.

The dashboard opens after login. Distribution-tag buttons replace the target library’s current in-memory query and select the corresponding value in the existing controls. The panel expands, with no separate drilldown banner. The other library’s filters remain independent. The combined statistics scope uses the currently selected library; movie/TV scopes choose explicitly. Reload clears unsaved filters. Presets change only when explicitly saved or selected.

JSON catalogs are discovered at build time, including future contributed languages. English keys are stable message identifiers with named placeholders and plain-text values. Go embeds the same catalogs for API errors/CSV. `Accept-Language` negotiation supports regional fallback and quality weights. UI preferences use browser storage; canonical media/filter values are never translated in storage. See [translation guide](TRANSLATING.md).

Root `VERSION` is embedded by Go, checked against package/Docker defaults and validated by CI against released tags. Release inputs must match source. Checked-in English notes feed GitHub releases before stable promotion. See [releasing](RELEASING.md).

## Recognition rules (0.3.0)

Schema 5 stores original API traits, a recognition revision, rule conflicts, indexed Emby matching states and indexed language membership. Built-in parser defaults seed a persisted ruleset only when none exists. Go RE2 rules are bounded to 256 entries, compiled once and grouped by attribute and mode. Direct, fallback and override phases are deterministic; scalar conflicts are independent of metadata disagreements. Compound audio/HDR/language results are additive.

Authenticated GET `/api/recognition` and `/api/recognition/defaults` expose rule configuration. CSRF-protected POST `/api/recognition/preview` compiles a draft and recognizes one filename without saving. PUT `/api/recognition` requires `{rules, revision}`. A stale revision returns 409. A serialized writer commits rule configuration, normalized file JSON, indexed fields and language membership together, then invalidates aggregate caches. Reprocessing uses batches of 250 rows and a consistent backup, without upstream requests. Sync commits acquire the same writer lock and reparse with the current rules, preserving cached Emby tracks and HDR.

Files/CSV/presets share `conflict=true|false`, `ruleConflict=true|false` and `embyMatch=unmatched|matched|missing|ambiguous|missing_size|no_tracks|unconfigured`. `unmatched` includes attempted matching failures and excludes unconfigured enrichment. The unified Audio column does not alter the separate `audio` and `language` filter parameters. Legacy language aliases and column layouts remain compatible.

Dashboard language distributions count distinct movies or episodes per language, independently of file and track multiplicity. Statistics include `languageTotals` by scope; overlapping multilingual groups must not be interpreted as a partition.

Paged lists materialize sorted row IDs and fetch full evidence/track JSON only for the selected page. Matching is reevaluated only for Sonarr season-context rows. A deterministic outer sort preserves all allowed column orders and tie-breakers while keeping large JSON out of the full-catalog temporary sort. CSV still streams all actual matches with the same sort semantics.
