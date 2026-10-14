# Storage widgets and clock schedules

## Storage selection

1. Under Settings → Widgets, refresh each enabled service's paths. This sends a read-only GET to `/api/v3/diskspace`.
2. Select the storage areas that should be available in the dashboard. Unchecked paths are excluded. Saved selections survive temporary service failures.
3. For duplicate paths reporting the same filesystem capacity, enable Shared storage and use the same group name. For example, Radarr and Sonarr `/downloads` can share a `downloads` group; their `/documentaries` paths can share a separate `documentaries` group. Do not group distinct Movies and Shows capacities just because they reside on the same array.
4. Edit Name for each selected path, shared or independent. It defaults to the original path and does not change the path or shared-group key. Entire-library groups use the first selected service name (Radarr before Sonarr); each service scope uses that service’s name.
5. Save settings. Previous/next arrows below the widget title show one area at a time. The browser remembers the selected area per scope. Entire library includes both services; Movies and TV series include only their service's selected areas. Shared areas appear once. Used and Free colors apply to all scopes.

**Scopearr does not add storage areas together.** Unraid Movies, Shows and Documentaries shares can use overlapping subsets of the same array. Their filesystem capacities and available bytes cannot establish the physical union. Displaying separate areas avoids a misleading combined total without requiring an Unraid connection.

The Arr APIs report filesystem/mount capacities, **not media-directory size** or Unraid share contents. Used space is the selected filesystem's total minus free, including other data. A Documentaries share containing 889 GB can therefore report many TB used on its underlying disks. Scopearr cannot infer physical identity from paths: different container paths may refer to the same device, while identical paths can refer to different devices. Configure duplicate groups based on your deployment. Avoid `/` and `/config` unless you specifically want those filesystems; `/config` and `/downloads` may report the same NVME pool.

Values are shown in decimal GB/TB, matching common Unraid displays. For example, 34.1 TiB is approximately 37.5 TB, and 903.1 GiB is approximately 970 GB. This unit conversion does not resolve overlapping capacity or directory usage differences.

Readings refresh every 15 minutes for enabled services with selected paths; manual Refresh storage is also available. No media mounts, scans, Unraid connection or filesystem traversal are performed. A shared group uses the newest complete reading for both used and free. Conflicting total capacities prevent that area's chart and show a warning; unrelated areas remain available. Missing paths and failed reads are marked; valid last-successful readings remain visible. A changed service endpoint cannot reuse the previous endpoint's readings.

Official capacity implementations: [Radarr](https://github.com/Radarr/Radarr/blob/develop/src/NzbDrone.Core/DiskSpace/DiskSpaceService.cs), [Sonarr](https://github.com/Sonarr/Sonarr/blob/v4.0.20.3014/src/NzbDrone.Core/DiskSpace/DiskSpaceService.cs).

## Clock schedules

Services retain the existing intervals: manual, 15/30/60 minutes, 6 hours or daily. An optional Starting at field anchors slots to a wall clock. Enter an IANA timezone, for example `Europe/Berlin`; an empty timezone uses UTC. When a start time is first entered, the browser's timezone is proposed in the settings draft.

Examples: Daily + 01:00 runs at 01:00; every six hours + 12:00 runs at 00:00, 06:00, 12:00 and 18:00. Hourly + 12:15 runs at 15 minutes past each hour. With no start time, the original elapsed-time schedule is retained. Manual only ignores the start time. Initial setup still synchronizes immediately. System shows the next clock slot.

Scheduling checks run every 15 seconds. Jobs queue behind ongoing sync, rule-cache updates and outbound-tag work, so a busy server can delay execution. After downtime, the latest overdue slot triggers one sync; missed slots are not replayed individually. Nonexistent spring DST slots are skipped; repeated autumn slots run once. Webhook updates do not reset the full-sync clock.

## Display names and shared identities

Name labels a selected path in the widget; it does not identify a filesystem. Shared storage group is the deduplication key. Radarr `/downloads` and Sonarr `/downloads` can have different display names but the same group `nvme-downloads` so that the combined view offers one area. Movies and Shows on overlapping parts of an Unraid array should remain separate areas, not be grouped just because they belong to the same array.
