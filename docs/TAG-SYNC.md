# Outbound tag synchronization

Under **Settings → Services**, select outbound tag groups independently for Radarr and Sonarr, then save. Every group is off by default. Emby remains a read-only metadata source.

The selected groups define the **complete desired tag set** for each Radarr movie or Sonarr series. Existing assignments, including tags previously added manually, are removed when they are not part of this desired set. Global tag definitions are preserved because they may be referenced elsewhere in Radarr/Sonarr.

Sonarr assigns tags to series rather than episode files. Scopearr combines values from all cached episode files of a series; mixed 1080p/x264 and 2160p/x265 episodes produce all four tags. Source includes processing values such as Remux. Missing/unknown traits do not produce tags.

Labels are normalized to lowercase ASCII letters, digits and hyphens. Punctuation and whitespace become separators; non-ASCII characters receive deterministic code-point encodings. Equivalent normalized labels share one remote definition. Existing definitions are reused case-insensitively. Interface translations do not change outgoing values.

## Reconciliation and recovery

A tag job waits for a completed file snapshot for the configured service URL. Changing URLs never applies the previous server's cached IDs to the new endpoint. Jobs share the file-sync gate, preventing overlapping reads and writes. Updates use small batches and verify the returned assignments.

Successful file syncs, relevant settings changes, saved recognition rules and custom-tag edits queue reconciliation. Failed jobs retain durable intent and retry in the background. A tag failure does not invalidate the successful file snapshot. System settings display running/pending state, last success and errors. Disabled services pause pending jobs.

Removing a custom tag removes its assignment during the next successful reconciliation. Disabling a group removes values that are no longer provided by another selected group. Disabling every group clears the assignments of titles already managed by Scopearr on that URL; new, unmanaged titles are left alone. Keep the service enabled and reachable until cleanup completes.

Scopearr does not delete global tag definitions. Remote automation or another client may reapply tags after reconciliation; Scopearr reports unverified update responses and reconciles again on the next applicable job.

## API boundaries

Catalog reads remain allowlisted GET requests. The outbound worker additionally permits only GET/POST `/api/v3/tag` and PUT `/api/v3/movie/editor` or `/api/v3/series/editor`. Editor payloads contain resource IDs, tag IDs, `applyTags: "replace"` and `moveFiles: false`; no scan, download, rename, profile or path-change requests are issued. API keys stay server-side and redirects are rejected.

See the official [Radarr API](https://radarr.video/docs/api/), [Sonarr API](https://sonarr.tv/docs/api/), [Radarr movie editor](https://github.com/Radarr/Radarr/blob/develop/src/Radarr.Api.V3/Movies/MovieEditorController.cs) and [Sonarr series editor](https://github.com/Sonarr/Sonarr/blob/develop/src/Sonarr.Api.V3/Series/SeriesEditorController.cs).

## Rollback

Database rollback does not undo changes already sent to another service. If remote cleanup is desired, disable all outbound groups and wait for successful reconciliation before rolling back. Restore the prior image together with its matching database backup and retain the encryption key, following [operations](OPERATIONS.md).
