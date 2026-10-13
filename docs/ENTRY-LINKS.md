# Info column and service entry links

Display → List display stores four independently selectable actions and their left-to-right order for both libraries: service link, file details, metadata disagreement and recognition conflict. Warning colors remain under Display. Warnings only appear on affected files; hiding them does not change detection or filters.

Info is a normal configurable column in each library’s column selection/order controls. It contains actions rather than sortable media attributes, and is omitted from CSV. The initial order is Link → Details → Metadata warning → Recognition warning. The additive settings upgrade inserts Info before the existing columns, preserves other choices and writes a consistent pre-upgrade backup once. Older images need that pre-upgrade backup to restore compatible settings; SQLite remains at schema 11.

The service interfaces route by the API's **titleSlug**, not the internal media ID:

- Radarr: configured base URL + `/movie/<titleSlug>`. The API normally supplies a numeric TMDB slug; TMDB ID is used only when titleSlug is missing. See the official [route](https://github.com/Radarr/Radarr/blob/af7814f7a89ea46f21522f0a1b8bcdcf7644b8bf/frontend/src/App/AppRoutes.tsx) and [movie page lookup](https://github.com/Radarr/Radarr/blob/af7814f7a89ea46f21522f0a1b8bcdcf7644b8bf/frontend/src/Movie/Details/MovieDetailsPage.tsx).
- Sonarr: configured base URL + `/series/<titleSlug>`. Episode files open their parent series. The official [Sonarr 4.0.20.3014 route](https://github.com/Sonarr/Sonarr/blob/v4.0.20.3014/frontend/src/App/AppRoutes.tsx) uses titleSlug.

Successful file syncs persist these slugs with cached files. Older cache rows use `/api/open-entry?service=…&mediaId=…`: Scopearr requires a logged-in session and an existing cached media ID, then resolves the slug through one GET-only resource read if needed and redirects to the configured service UI. It neither scans nor changes the remote library. URL subpaths are retained. Empty/unusable service addresses show a disabled link icon; failed legacy resolution reports an error without changing the cache.

Links use a new tab, noopener/noreferrer and no-referrer. API keys are sent only in authenticated backend request headers and never included in links. Arbitrary target URLs, credentials in base URLs, query strings, fragments and invalid slugs are rejected. Normal list queries do not make upstream requests for links.
