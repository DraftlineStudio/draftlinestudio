# Releasing Draftline

Publishing a GitHub release builds installers for all three desktop platforms
and attaches them to that release. The automation lives in
`.github/workflows/release.yml`.

## What a release produces

| Asset | Platform | Notes |
|-------|----------|-------|
| `Draftline-<version>-windows-amd64-setup.exe` | Windows 10/11, x64 | NSIS installer. Installs to Program Files, adds Start Menu and desktop shortcuts, downloads the WebView2 runtime if missing, registers an uninstaller. |
| `Draftline-<version>-windows-amd64-portable.zip` | Windows 10/11, x64 | The bare `draftline.exe`. Runs from anywhere; file associations are registered per-user on first launch. |
| `Draftline-<version>-macos-universal.dmg` | macOS 10.15+, Intel and Apple Silicon | Drag-to-Applications disk image. Ad-hoc signed unless Apple credentials are configured (see below). |
| `Draftline-<version>-linux-x86_64.AppImage` | Linux x86_64 | Needs `libwebkit2gtk-4.1` and GTK 3 on the host (Ubuntu 22.04+, Fedora 36+, and equivalents). Mark executable and run. |
| `SHA256SUMS.txt` | all | Checksums of every asset. |

## Cutting a release

1. Bump the version in every location listed in `docs/VERSION.md` and add the
   `CHANGELOG.md` entry. Commit and push to `main`.
2. On GitHub, create a release whose tag is the new version, with or without a
   leading `v` (for example `v0.17.02562`). For a beta or release candidate,
   append a suffix (`v0.17.02562-beta`, `v0.17.02562-rc.1`) and tick
   **Set as a pre-release**; the suffix appears in the asset file names, while
   the binaries still report the plain version. The tag must otherwise match
   `const AppVersion` in `wails/app.go`; the workflow fails immediately if it
   does not, so a stale binary can never be published under a newer tag.
3. Publish the release. The workflow builds all packages (roughly 10 to 15
   minutes) and uploads them to the release. Draft releases do not trigger a
   build until they are published.

To test the packaging without creating a release, run the **Release** workflow
from the Actions tab (workflow_dispatch). The packages appear as run artifacts
and nothing is uploaded to a release.

The version is read from `wails/app.go` at build time and stamped into the
Windows exe metadata, the NSIS installer, and the macOS `Info.plist` by
`.github/scripts/stamp-version.js` and PlistBuddy. Those stamps are build-time
only; `wails/wails.json` and `build/darwin/Info.plist` are not edited in the
repository during a release.

## How each package is built

- **Windows** (`windows-latest`): `wails build -platform windows/amd64 -nsis`.
  The installer script is `wails/build/windows/installer/project.nsi`; Wails
  regenerates the companion `wails_tools.nsh` on every build, so only
  `project.nsi` is tracked; it shows the repository `LICENSE` (MIT) as the
  installer's license page. The exe icon and version resource come from
  `build/windows/icon.ico` and `build/windows/info.json`.
- **macOS** (`macos-latest`): `wails build -platform darwin/universal`
  produces the `.app` from `build/darwin/Info.plist` and an `.icns` generated
  from `build/appicon.png`. The bundle is code-signed, wrapped in a DMG with
  `hdiutil`, and notarized when credentials exist.
- **Linux** (`ubuntu-24.04`): `wails build -platform linux/amd64 -tags webkit2_41`,
  then an AppDir is assembled from `build/linux/draftline.desktop`,
  `build/linux/draftline-mime.xml`, and `build/appicon.png`, and packed with
  `appimagetool`.

## Optional signing secrets

Without these the workflow still succeeds; the notes describe what users see.

### macOS (Developer ID + notarization)

Add these repository secrets (Settings → Secrets and variables → Actions):

| Secret | Value |
|--------|-------|
| `APPLE_CERTIFICATE_P12` | Base64 of a `.p12` export of your **Developer ID Application** certificate and private key (`base64 -i cert.p12 \| pbcopy`). |
| `APPLE_CERTIFICATE_PASSWORD` | The password used when exporting the `.p12`. |
| `APPLE_ID` | Apple ID email of the developer account. |
| `APPLE_TEAM_ID` | 10-character Team ID. |
| `APPLE_APP_SPECIFIC_PASSWORD` | An app-specific password generated at appleid.apple.com for notarization. |

With the first two set, the app and DMG are signed with the hardened runtime
using `build/darwin/entitlements.plist`. With all five set, the DMG is also
notarized and stapled, so it opens without Gatekeeper warnings.

Without them the bundle is ad-hoc signed. Users must right-click → Open the
first time, or on macOS 15 and later approve the app under System Settings →
Privacy & Security after the first blocked launch. Downloaded copies may also
need `xattr -dr com.apple.quarantine /Applications/Draftline.app`.

### Windows (Authenticode)

Not wired up. Unsigned installers trigger a SmartScreen "unknown publisher"
warning that users dismiss with More info → Run anyway. Signing requires an
OV or EV code-signing certificate (a hardware token or a cloud signing
service such as Azure Trusted Signing); when one is available, add a
`signtool` step after `wails build` for `draftline.exe` and the installer.

## Assets and known gaps

- `wails/build/appicon.png` is 565×555. Wails' `.icns` generation and the
  AppImage icon both expect a square image; a 1024×1024 PNG is the
  recommended replacement.
- Linux and Windows are built for x86_64 only. Windows on ARM and Linux ARM64
  packages are not produced.
- Windows builds are also tested locally before every release; macOS and
  Linux packages are produced only in CI and have not been exercised on real
  hardware yet, so the first published release should be smoke-tested on each
  platform.
