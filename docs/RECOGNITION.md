# Filename recognition

Open **Settings → System → Manage recognition rules**. The page edits a draft. Preview uses the draft and the filename only; Radarr, Sonarr and Emby enrichment is not included in preview.

## Rule modes and priority

- **Direct recognition** extracts explicit filename attributes.
- **Only if missing** applies only when direct rules found no value for that attribute.
- **Explicit override** intentionally replaces direct and fallback results for that attribute.

Lower priority numbers run first. Equal priorities use the stable rule ID. Sorting table headers changes the display order only. For scalar attributes, the first matching rule supplies the displayed value. Different values in the winning mode produce a recognition warning with the contributing rules and values. Codec aliases from the same family do not conflict. HDR profiles, audio codecs and languages are additive.

An explicit override resolves lower-mode conflicts deliberately. Two contradictory overrides still warn. Filename/API disagreements use a separate metadata warning. Both colors can be changed in **Settings → Appearance → Warning colors**.

## Example: missing UHD resolution

Add a rule with:

| Setting | Value |
| --- | --- |
| Name | UHD disc resolution |
| Attribute | Resolution |
| Match type | Token |
| Pattern | UHDBD |
| Result | 2160p |
| Mode | Only if missing |
| Priority | 10 |

`Movie.UHDBD.x265.mkv` recognizes 2160p. `Movie.1080p.UHDBD.x265.mkv` retains 1080p. This fallback does not create a conflict with the explicit resolution. Add a separate Source rule if you also want `UHDBD` recognized as a source.

## Patterns

Tokens match case-insensitively at separators such as dots, spaces, hyphens and brackets. Regular expressions use Go's RE2 syntax, with bounded input and configuration size and no catastrophic backtracking. Capture references such as `${1}p` can be used in regex results. Invalid expressions are rejected, including lookbehind and backreferences. Zero-width matches cannot infer values.

Advanced controls include an exclusion expression and consuming matched text before subsequent rules in the same attribute and mode. Default audio rules use consumption to distinguish DTS-HD from DTS. Preserve this setting when editing overlapping audio patterns.

## Saving and data safety

**Apply to draft** changes the draft only. **Restore defaults** also changes the draft only. **Save and reprocess cache** validates all rules, creates a backup and commits rules and the recomputed cache in one transaction. Failed or cancelled saves preserve the previous state. A revision check prevents a stale browser from overwriting another session's changes.

No upstream requests or media scans are made. API traits stay separate from filename evidence. Previously matched Emby tracks and HDR data are replayed from the cache. A sync that finishes after an edit uses the current rules at commit time. Library filters, CSV, facets and statistics all use the updated cache.

Rules are stored in SQLite, not in the application image. New installations receive the built-in defaults; an update does not replace your saved rules. Keep a complete Appdata backup, including the local key. Rule edits and migrations share the last-three-snapshot retention policy.
