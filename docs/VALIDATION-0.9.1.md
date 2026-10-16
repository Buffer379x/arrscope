# Scopearr 0.9.1 validation

The local frontend tests, TypeScript check and production frontend build pass.
Regression tests cover the additive Processing column upgrade, preservation of hidden/reordered columns, idempotent normalization and independent Source/Processing sorting with unknown values last.
Existing layout regression expectations include the new column.

Local HTTP listeners and the Docker socket are unavailable under this session's network permissions. The full Go/race tests, container builds, security scans and 100,000-file container checks therefore run in GitHub CI and the release workflow. No real media-service settings or the Unraid installation are changed.

The sync-time control uses a text input with a 24-hour HH:MM pattern and bounded grid sizing. Browser/device checks on physical iPad Safari are not available in this environment.
