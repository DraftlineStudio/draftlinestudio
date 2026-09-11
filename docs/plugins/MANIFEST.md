# Plugin Manifest Reference

Every plugin folder contains a `manifest.json`. Draftline validates it
strictly: **unknown fields are rejected**, so typos fail loudly instead of
silently doing nothing. A plugin whose manifest fails validation appears in
Settings → Plugins with the error and is never activated.

## Complete example

```json
{
  "schema": 1,
  "id": "acme.word-goals",
  "name": "Word Goals",
  "version": "1.2.0",
  "publisher": "Acme",
  "api_version": 1,
  "description": "Daily word-count goals with streaks.",
  "permissions": ["editor.read", "settings"],
  "activation": ["onStartupIfEnabled"],
  "frontend": { "entry": "frontend/index.js" },
  "sidecar": {
    "windows-amd64": "bin/word-goals.exe",
    "darwin-arm64": "bin/word-goals",
    "linux-amd64": "bin/word-goals"
  },
  "contributes": {
    "editorDockBars": [{ "id": "goalbar" }],
    "settingsSections": [{ "id": "goals", "title": "Word Goals" }],
    "commands": [
      { "id": "goals.toggle", "shortcut": "Mod+Shift+G" }
    ]
  }
}
```

## Fields

| Field | Required | Meaning |
|---|---|---|
| `schema` | yes | Manifest schema version. Currently `1`. |
| `id` | yes | Globally unique, `publisher.name` in lowercase kebab: `^[a-z0-9][a-z0-9-]*(\.[a-z0-9][a-z0-9-]*)+$`. Also the folder's conventional name and the namespace for events, settings, and commands. |
| `name` | yes | Display name. |
| `version` | yes | Your version string (semver recommended). |
| `publisher` | yes | Display publisher. |
| `api_version` | yes | The host API version the plugin targets. This build of Draftline provides **v1**. A mismatch lists the plugin but blocks activation with a clear message. |
| `description` | no | One or two sentences for the plugins list. |
| `permissions` | no | What the plugin may do — see below. Anything not declared throws at call time. |
| `activation` | no | When the frontend bundle loads — see below. Empty/omitted means `onStartupIfEnabled`. |
| `frontend` | see note | `{ "entry": "<relative path>" }` to the bundled ES module. |
| `sidecar` | see note | Map of `"<goos>-<goarch>"` → relative executable path. Keys: `windows-amd64`, `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`. A plugin without a sidecar entry for the running platform simply has no backend there. |
| `contributes` | no | Declarative UI slots — see below. |

**Note:** a plugin must declare `frontend`, `sidecar`, or both.

**Path rules** (for `frontend.entry` and `sidecar` values): relative, forward
slashes, no `.`/`..` segments, no drive letters. Violations fail validation.

## Permissions

| Permission | Grants |
|---|---|
| `editor.read` | Read the manuscript text, the selection, and the open book's data; watch book opens. |
| `editor.decorate` | Drive the editor spotlight (highlight ranges). |
| `propose-edits` | Propose changes through Draftline's review panel. *(Declared for forward compatibility; the v1 API surface for it has not shipped.)* |
| `book.plugin-data` | Store plugin data inside the book file, namespaced to the plugin. *(Reserved; the write API lands with the first consumer.)* |
| `settings` | Keep per-user plugin settings (`host.settings`). |
| `secrets` | Store secrets in the OS keyring, namespaced. *(Reserved for the AI plugin work; no v1 API yet.)* |
| `network` | Network access from the plugin's own sidecar process. |
| `network:model-download` | Download pinned, hash-verified artifacts. |

Unknown permission strings fail validation. Note honestly what enforcement
means: `host.*` capabilities are enforced by the host at call time, while a
sidecar is its own OS process — its `network*` declarations are consent
surface (shown to the user) rather than an OS-level sandbox.

## Activation events

| Event | The frontend bundle loads when… |
|---|---|
| `onStartupIfEnabled` | …the app starts and the plugin is enabled. |
| `onCommand` | …one of the plugin's shortcuts fires the first time. |
| `onSettingsOpen` | …the settings dialog opens. |
| `onBookOpen` | …a book is opened. |

Sidecars are always lazy: they start on the first `host.backend.invoke`.

## Contributions

Contributions are rendered from the manifest for **enabled** plugins before
any plugin code runs; your bundle then fills them at activation.

- `editorDockBars: [{ id }]` — a docked bar at the editor's dock slot (where
  the Read Aloud player lives). Fill it with
  `host.ui.mountDockBar(id, …)`.
- `settingsSections: [{ id, title }]` — a nav item in the settings dialog.
  Fill it with `host.ui.mountSettingsSection(id, …)`.
- `commands: [{ id, shortcut? }]` — invocable commands. `shortcut` uses
  `Mod+Shift+X` notation (`Mod` = Ctrl on Windows/Linux, Cmd on macOS).
  Handle firings with `host.commands.on(id, fn)`.

Mounting a slot you didn't declare throws — the manifest is the single
source of truth for what a plugin touches.
