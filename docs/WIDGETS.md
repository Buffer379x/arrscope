# Storage widgets and clock schedules

## Storage selection

1. Under Settings → Widgets, refresh each enabled service's paths. This sends a read-only GET to `/api/v3/diskspace`.
2. Select the filesystems that should contribute to the dashboard. Unchecked paths are excluded. Saved selections are retained when a service temporarily fails.
3. For multiple paths on the same storage, enable Shared storage and use the same group name. The Entire library view counts that group once. The Movies and TV series views include only their respective selected service paths. Distinct groups and ungrouped paths are summed.
4. Save settings. Used and Free colors apply to all scopes. Total capacity appears inside the chart.

The Arr APIs report filesystem/mount capacities, **not media-directory size** and not Unraid share disk allocation policy. Scopearr cannot infer physical identity from path strings: different container paths may refer to the same device, while identically named paths may refer to separate devices. Select/group based on your actual deployment. Multiple paths on a single pool must share a group even within one service. Independent Film/TV Unraid pools remain separate; a shared Downloads pool uses a common group.

Readings refresh every 15 minutes for enabled services with selected paths; manual Refresh storage is also available. No media mounts, scans or filesystem traversal are performed. For a shared group with matching total capacity, the newest complete reading supplies both used and free space. Different total capacities show a warning rather than presenting an unreliable combined chart. Missing selected paths or failed reads are marked; the last successful values remain available. Disabling a service excludes it from totals.

Official capacity implementations: [Radarr](https://github.com/Radarr/Radarr/blob/develop/src/NzbDrone.Core/DiskSpace/DiskSpaceService.cs), [Sonarr](https://github.com/Sonarr/Sonarr/blob/v4.0.20.3014/src/NzbDrone.Core/DiskSpace/DiskSpaceService.cs).

## Clock schedules

Services retain the existing intervals: manual, 15/30/60 minutes, 6 hours or daily. An optional Starting at field anchors slots to a wall clock. Enter an IANA timezone, for example `Europe/Berlin`; an empty timezone uses UTC. When a start time is first entered, the browser's timezone is proposed in the settings draft.

Examples: Daily + 01:00 runs at 01:00; every six hours + 12:00 runs at 00:00, 06:00, 12:00 and 18:00. Hourly + 12:15 runs at 15 minutes past each hour. With no start time, the original elapsed-time schedule is retained. Manual only ignores the start time. Initial setup still synchronizes immediately. System shows the next clock slot.

Scheduling checks run every 15 seconds. Jobs queue behind ongoing sync, rule-cache updates and outbound-tag work, so a busy server can delay execution. After downtime, the latest overdue slot triggers one sync; missed slots are not replayed individually. Nonexistent spring DST slots are skipped; repeated autumn slots run once. Webhook updates do not reset the full-sync clock.
