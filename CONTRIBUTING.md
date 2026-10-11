# Contributing to Scopearr

Use English for public descriptions, documentation, changelogs and release notes. Interface translations are welcome: see [the translation guide](docs/TRANSLATING.md).

Keep all upstream operations read-only, preserve the last complete cache on failure, and do not add media mounts or scans. Never commit credentials, Appdata, personal templates or private library screenshots.

For changes after a release, bump `VERSION`, matching frontend package/Docker defaults, and add an English changelog/release entry. Run the checks in [README](README.md) and describe meaningful validation in your pull request. New releases are manually approved; normal pushes do not update stable.
