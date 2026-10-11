# ArrScope 0.3.3 validation

- Full Go test suite, race suite and `go vet ./...`: passed locally. Backend filtering, preset storage and synchronization behavior are unchanged.
- Frontend: all 11 tests passed; TypeScript and production Vite build passed. English, German, Spanish and Italian catalogs retain identical keys and compatible placeholders.
- Browser inspection used isolated synthetic demo data, with dark and light themes at 1280px and 390px viewport widths. Color and warning tables contain their own headers, maintain left preview padding and show 40px bordered controls. No horizontal page overflow at 390px.
- Search input and preset dropdown both measure 42px. Native selects remain single-select; existing attribute checkbox multi-selects are unchanged.
- Text-color changes persisted across save and reload. Reset restored the default state. The existing confirmation and unsaved-change guards remain in place.
- Filename-preview results remain visible with nine empty attribute rows before input. `Movie.UHDBD.x265.mkv` correctly previews 2160p from the existing custom fallback rule and x265 from a default rule. The wider input sits above the result table, aligned left on desktop and stacked with its action on mobile.
- Settings footer has no Close button or divider; Save remains available. Unused close-prop wiring was removed; parent navigation still guards dirty settings.
- Local image builds use the existing read-only, non-root, 256MiB/no-swap configuration. Release gates additionally require vulnerability scans, both architectures, the 50,000-file performance check and the 100,000-file Emby/container stress check before publication.

SQLite schema remains 5. No production services or Unraid settings were changed. Production updates are initiated manually by the user.
