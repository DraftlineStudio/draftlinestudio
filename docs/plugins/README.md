# Draftline Plugin Development

Draftline supports installable plugins: self-contained folders that add UI
and functionality without being compiled into the app. This directory is the
developer documentation for building them.

> **Status: host API v1 is an internal preview.** Until the marketplace
> ships, the API may change between Draftline releases without deprecation.
> A plugin pins the API it targets via `api_version` in its manifest; a
> mismatched plugin is listed in Settings → Plugins but never activated.

| Document | Contents |
|---|---|
| [MANIFEST.md](MANIFEST.md) | The manifest.json schema: every field, permission, contribution point, and activation event |
| [HOST-API.md](HOST-API.md) | The `host` object your frontend bundle receives: full API reference |
| [SIDECAR.md](SIDECAR.md) | The sidecar process contract: JSON-RPC over stdio, lifecycle, supervision |

## What a plugin is

A plugin is a **folder** with up to three kinds of artifact:

```
<plugin root>/acme.word-goals/
├── manifest.json        required — identity, contributions, permissions
├── frontend/
│   └── index.js         optional — one bundled JS module (the UI half)
└── bin/
    ├── acme-goals.exe   optional — a sidecar executable per platform
    └── acme-goals       (the backend half; any language that speaks
                          JSON-RPC over stdio)
```

- **manifest.json** is pure data. Draftline reads it to render your plugin's
  slots (settings section, dock bar, keyboard shortcuts) *without executing
  anything*. See [MANIFEST.md](MANIFEST.md).
- **The frontend bundle** is one compiled/bundled ES module, dynamically
  imported into the app's webview and handed the [`host`](HOST-API.md) API.
  Author it in anything that compiles to a single JS file.
- **The sidecar** is a normal standalone executable Draftline launches and
  supervises, talking JSON-RPC 2.0 over stdin/stdout. It runs in its own
  process — a plugin crash can never take the app down. See
  [SIDECAR.md](SIDECAR.md).

A plugin needs at least one of the two code artifacts (frontend or sidecar).

## Where plugins live

Plugins are installed **machine-wide** — one copy shared by every OS user:

| Platform | Shared plugin root |
|---|---|
| Windows | `C:\ProgramData\Draftline\plugins\` |
| macOS | `/Library/Application Support/Draftline/plugins/` |
| Linux | `/var/lib/draftline/plugins/` |

When the shared root isn't available (portable installs, locked-down
machines), the per-user fallback is `<user config dir>/draftline/plugins/`
(`%APPDATA%\draftline\plugins\` on Windows). If both roots contain the same
plugin id, the shared copy wins.

Per-user state never includes plugin files: whether a user has a plugin
**enabled**, and the plugin's own settings, are a few lines in that user's
`settings.json`. Plugins are disabled by default after installation; each
user switches them on in **Settings → Plugins**.

## Development workflow

Point Draftline at a development directory; plugins there shadow installed
copies of the same id:

```
Draftline.exe --plugin-dev C:\dev\my-plugins
# or
set DRAFTLINE_PLUGIN_DEV=C:\dev\my-plugins
```

Each subfolder of that directory is treated as a plugin. Edit, rebuild your
bundle, and relaunch (or disable/enable the plugin in Settings → Plugins to
re-activate without a restart).

## Lifecycle

1. **Discovery** — at startup Draftline scans the plugin roots and reads
   every `manifest.json`. A broken manifest shows up in Settings → Plugins
   with its error; it is never partially loaded.
2. **Slot rendering** — declared contributions (settings sections, dock
   bars, shortcuts) appear in the UI for enabled plugins without loading any
   plugin code.
3. **Activation** — the frontend bundle is imported and its
   `activate(host)` called when one of the manifest's `activation` events
   fires (`onStartupIfEnabled`, `onCommand`, `onSettingsOpen`,
   `onBookOpen`). No declared events means activate-at-startup-when-enabled.
4. **Sidecar start** — lazily, on the first `host.backend.invoke(...)`.
5. **Deactivation** — disabling the plugin calls your `deactivate` (if you
   returned or exported one), unmounts your contributions, and stops your
   sidecar.

## Hello world (10 minutes, UI-only)

`C:\dev\my-plugins\acme.hello\manifest.json`:

```json
{
  "schema": 1,
  "id": "acme.hello",
  "name": "Hello Draftline",
  "version": "0.1.0",
  "publisher": "Acme",
  "api_version": 1,
  "description": "Proof-of-life settings section.",
  "permissions": ["editor.read"],
  "frontend": { "entry": "frontend/index.js" },
  "contributes": {
    "settingsSections": [{ "id": "hello", "title": "Hello" }]
  }
}
```

`C:\dev\my-plugins\acme.hello\frontend\index.js`:

```js
export function activate(host) {
  host.ui.mountSettingsSection('hello', {
    mount(el) {
      const btn = document.createElement('button')
      btn.textContent = 'What am I looking at?'
      const out = document.createElement('p')
      btn.onclick = () => {
        out.textContent = 'Selection: ' + (host.editor.getSelectionText() || '(nothing selected)')
      }
      el.append(btn, out)
    },
  })
  host.log('activated')
  return () => host.log('deactivated')
}
```

Launch with `--plugin-dev C:\dev\my-plugins`, open **Settings → Plugins**,
enable *Hello Draftline*, and a **Hello** section appears in the settings
nav.

## Rules of the road

- Your bundle mounts into bare DOM elements the host provides — never
  Draftline's React tree. Ship your own framework if you want one.
- Ship your own styles with your bundle; don't rely on Draftline's
  stylesheet internals beyond the mounted element's box.
- Everything you can touch is gated by the permissions your manifest
  declares; an undeclared capability throws.
- Bulk data (model downloads, audio, large payloads) must not go through the
  JSON-RPC pipe — serve it from your sidecar over a loopback socket and pass
  the URL.
