# Version Number Locations

The project version (`MAJOR.MINOR.BUILD`, e.g. `0.15.02374`) is not generated from a single source — it is hand-maintained in the locations below. **When bumping the version, update every one of them.** (See `CHANGELOG.md` for the versioning scheme: BUILD increments by 1 per change; one fix = one number.)

Current version as of this writing: **0.18.02577**

## Locations to update

| # | File | What to change |
|---|------|----------------|
| 1 | `wails/app.go` | `const AppVersion = "..."` (~line 47). This is what `GetAppVersion` returns — shown on the welcome screen and in the settings dialog footer. |
| 2 | `wails/frontend/package.json` | `"version"` field (line 4). |
| 3 | `wails/frontend/package-lock.json` | `"version"` appears **twice** — top-level (~line 3) and under `packages[""]` (~line 9). Running any `npm install` after editing package.json also syncs these. |
| 4 | `CHANGELOG.md` (repo root) | Add a new `## [x.y.0zzzz] - YYYY-MM-DD` entry at the top describing the change. |
| 5 | This file | The "current version" line above, so it stays a reliable cross-check. |

## Notes

- `wails/wails.json` has no version field; there is nothing to update there. The release workflow stamps `AppVersion` into it (and into `wails/build/darwin/Info.plist`) at build time — see `docs/RELEASING.md`.
- A GitHub release tag must equal `AppVersion`, optionally prefixed with `v` and optionally followed by a pre-release suffix such as `-beta`; the release workflow refuses to build otherwise.
- To catch stragglers after a bump, search the repo for the **old** build number (excluding `node_modules/`). The only remaining hits should be historical `CHANGELOG.md` entries.
