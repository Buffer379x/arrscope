# ArrScope 0.3.1 validation

## Scope

Heading/card alignment, existing-control Dashboard filtering, footer removal and persistent tag text colors. Schema 5 and all upstream clients remain unchanged. Tests use local synthetic fixtures; no production services or Unraid installation are modified.

## Local checks

- All Go tests pass with the race detector; `go vet ./...` passes.
- All ten frontend tests pass, including four-catalog consistency, canonical Dashboard queries and independently validated black/white text colors.
- TypeScript, Vite production build and version consistency checks pass.
- Settings API tests check valid text-color round trips, invalid values returning 400 without changing configuration, and persistence across store/server restart. Legacy settings with no text-color map retain automatic rendering.
- The non-root, read-only Docker image builds and runs against the existing synthetic preview volume with a 256 MiB limit and no additional swap.

## Browser checks

At a 1280 × 900 desktop viewport, both Dashboard and Movies have heading top 133 px, line height 24 px and first card top 171 px in the demo. At a 390 × 844 mobile viewport, Dashboard and TV series have heading top 155 px and first card top 193 px, with no document-level horizontal overflow. Demo-banner height is included in these coordinates.

A click on Dashboard 1080p expands the existing Resolution filter, selects 1080p, returns matching movie rows and exports the same query to CSV. A TV-scoped x265 click selects the existing Codec filter, whose x265 checkbox is checked. The separate temporary-filter banner is absent. Saved presets are unchanged; selections reset on page reload.

Black and white text choices appear immediately in settings previews and survive save/reload. Setting only a foreground keeps the default background and border. Reset removes both overrides. Existing darker border and hover styling remain independent of text color. The Dashboard footer paragraph is absent.

## Release gates

The required GitHub `tests` check and release workflow repeat frontend, race/vet, dependency, 50k-file performance, image security, restart and 100k-file/Emby container checks. Publication waits for successful required checks. Public versioned and `stable` image manifests are checked for matching digests and both supported architectures after publication.
