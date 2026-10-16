# Scopearr 0.9.2 validation

Live read-only comparison of the reported movie confirmed that Scopearr 0.9.1 uniquely matches the file and reads its MP3 audio track, but discards Emby's video values. Emby's detail page reports 576×244 pixels, MPEG4 with XVID FourCC, 8-bit video and stereo MP3; the filename and Arr evidence contain no resolution or codec. The regression fixture reconstructs these observed fields; it is not a captured raw API response.

Regression coverage includes filename/Arr priority, MPEG-4 normalization, cropped/interlaced dimensions, video-only matches, alternate-source boundaries, provider agreement/conflict, explicit audio-layout fallback, cache persistence and recognition reprocessing. The widget test ensures all eight detailed states belong to exactly one of four groups and totals remain equal.

Local frontend tests, TypeScript, production build and non-network Go tests are run before publication. Full HTTP/race tests, security scans, multi-architecture container checks and the 100,000-file dual-provider resource check run in GitHub CI/release workflows. No production update or media scan is performed.

The initial 100,000-file dual-provider run exposed an OOM at the unchanged 256 MiB/no-swap limit. Candidate indexes now retain only identity fingerprints; normalized track/video results are stored once, with equal provider inventories shared immediately and bounded per-snapshot pools for identical immutable video/track values. Redundant persisted channel-layout copies were removed, and released provider-index pages are returned before the next provider. The same resource gate is rerun before release.
