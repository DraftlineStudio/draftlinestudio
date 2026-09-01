# File Associations (cross-platform)

Draftline claims `.draftline` as its own document type and appears in
"Open with" for `.epub`/`.docx` (routed to the importers). `.storiverse` is registered as its own type (`Draftline.Universe`, slate-gray icon) for visual identity only — double-clicking one shows a native "support coming in a later version — update to open" dialog until Storiverse ships. `.pdf` is deliberately unregistered: there is no PDF importer.

## Runtime plumbing (shared, 0.16.02477)

All OS-initiated opens funnel into one path:

- **Launch argument** — `main.go` scans `os.Args` via `launchFilePath()`
  (`fileopen.go`) and stashes the file; the frontend collects it once via the
  `TakePendingOpenPath` binding after init.
- **Second instance** — `SingleInstanceLock` in `main.go`: double-clicking a
  document while Draftline runs focuses the existing window and emits a
  `file:open` event with the path (`onSecondInstanceLaunch`).
- **macOS open-file events** — `mac.Options.OnFileOpen` → same pending slot /
  event (`onMacFileOpen`).

The frontend (`App.tsx` effect → `bookStore.openExternalFile`) routes by
extension: `.draftline` → `openRecentBook` (the normal open flow),
`.epub`/`.docx` → `importExternalBook` (the wizard's importers, minus the
picker). Both honor the unsaved-changes dialog via path-carrying
`pendingAction` values (`{ openPath }` / `{ importPath }`), so an OS open can
never silently discard unsaved work.

## Windows (0.16.02478)

`fileassoc_windows.go` self-registers on every launch — best-effort, off the
startup path, under **HKCU\Software\Classes** (per-user, no admin, works for
the portable exe):

- `Draftline.Project` ProgID (name, icon from the exe, open command) and
  `.draftline` → that ProgID.
- `Applications\draftline.exe` with `SupportedTypes` `.draftline`/`.epub`/
  `.docx` — this adds the "Open with → Draftline" entry **without** touching
  the user's default epub/docx handlers.

Guards: registration is skipped unless the executable is literally
`draftline.exe`, so `wails dev` (draftline-dev.exe) can never point
associations at a dev build. A moved exe heals itself on next launch.
To remove: delete `HKCU\Software\Classes\Draftline.Project`, `.draftline`,
and `Applications\draftline.exe`. Explorer may need a restart to refresh
icons.

## macOS (packaging asset, untested — no mac hardware here)

`build/darwin/Info.plist` declares `CFBundleDocumentTypes`: `.draftline` as
an exported UTI (`com.draftline.project`, `LSHandlerRank Owner`), epub/docx
as `Alternate` (Open-with only). Static plist — no wails template
placeholders. Launch Services picks it up when the `.app` is first moved
into place; the `OnFileOpen` runtime hook is already wired.

## Linux (packaging assets)

`build/linux/draftline.desktop` (with `MimeType=` for all three formats) and
`draftline-mime.xml` (defines `application/x-draftline`). Install:

```sh
cp build/linux/draftline-mime.xml ~/.local/share/mime/packages/
update-mime-database ~/.local/share/mime
cp build/linux/draftline.desktop ~/.local/share/applications/
update-desktop-database ~/.local/share/applications
```

## Tests

`fileopen_test.go` — launch-arg filtering (extension allowlist, flags,
relative paths, missing files) and the one-shot pending-path handoff.
`bookStore.test.ts` "external file opens" — dirty-book deferral preserving
the exact path, importer routing, unsupported-extension no-op.
