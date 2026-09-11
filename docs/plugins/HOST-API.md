# Host API Reference (v1)

Your frontend bundle's entry module must export an `activate` function. On
activation Draftline calls it with the `host` object documented here.
Returning a function (or exporting `deactivate`) registers your cleanup.

```js
export function activate(host) {
  // …register UI, wire commands…
  return () => { /* deactivate: stop timers, drop references */ }
}
```

| API version | First Draftline build |
|---|---|
| 1 (internal preview — may change until the marketplace ships) | 0.19.02588 |

Calls gated by a permission **throw synchronously** when the manifest
doesn't declare it. Gates are listed per method below.

## `host.apiVersion` / `host.plugin`

- `apiVersion: number` — always `1` for this reference.
- `plugin: { id, version }` — your own identity, as installed.

## `host.ui`

Plugins render into bare DOM elements the host owns — never Draftline's
React tree. Each mount target corresponds to a contribution declared in the
manifest; mounting an undeclared id throws.

### `host.ui.mountDockBar(id, { mount, unmount? })`
Fills a declared `editorDockBars` contribution. `mount(el)` receives a
container `<div>` at the editor's dock slot (rendered whenever a book is
open and the plugin is enabled); `unmount()` is called on deactivation and
before the container is emptied.

### `host.ui.mountSettingsSection(id, { mount, unmount? })`
Fills a declared `settingsSections` contribution. The section's nav item
(with the manifest's `title`) appears for enabled plugins; `mount(el)` runs
when the user opens it.

## `host.editor`

### `host.editor.getSelectionText(): string`
*Requires `editor.read`.* The current selection's plain text, `''` when
nothing is selected or no book is open.

### `host.editor.spotlight`
*All spotlight methods require `editor.decorate`.* The spotlight is
Draftline's single highlight-one-active-range-of-a-set primitive (it renders
Read Aloud's sentence highlight). It is deliberately **exclusive** — one
owner at a time; taking it replaces the previous owner's handlers.

- `setRanges(ranges: {from, to}[])` — begin a spotlight session over
  document positions; nothing is highlighted until `setActive`.
- `setActive(index: number)` — highlight the range at `index` (class
  `range-spotlight-current`); out-of-bounds clears the highlight.
- `clear()` — end the session.
- `onRangeClick(cb: (index) => void)` — a click inside a range reports its
  index (the caret still moves normally).
- `onDocEdited(cb: () => void)` — the user edited the document while the
  session was active. Ranges are remapped across the edit automatically so
  nothing flashes stale; use the callback to stop your session properly.

## `host.book`

### `host.book.getData(): BookData | null`
*Requires `editor.read`.* The open book's data (metadata, sections, chapter
HTML), or `null` on the launch screen. Treat it as read-only.

### `host.book.onOpened(cb: (book) => void): () => void`
*Requires `editor.read`.* Fires when a different book file becomes current.
Returns an unsubscribe function — call it in your `deactivate`.

*(A namespaced write path — `book.plugin-data` — is reserved and lands with
its first consumer; v1 has no book write API.)*

## `host.settings`

*Both require `settings`.* A per-user key/value bag namespaced to your
plugin, stored in the user's Draftline settings. Values must be
JSON-serializable. This is the **entire** per-user footprint a plugin has —
plugin files are machine-wide.

- `get(key): unknown`
- `set(key, value): Promise<void>`

## `host.commands`

### `host.commands.on(commandId, fn)`
Registers the handler for a command declared in the manifest (undeclared
ids throw). Declared shortcuts are bound by the host; a shortcut on a
not-yet-activated plugin with the `onCommand` activation event loads the
plugin first, then fires the handler.

## `host.backend`

### `host.backend.invoke(method, payload?): Promise<unknown>`
Sends one JSON-RPC request to your sidecar (starting it on first use) and
resolves with its result (`null` for empty results). Rejects with the
sidecar's error message, a timeout after 120 s, or a crash report. Long
work should return quickly and report progress via notifications
(`host.events`) instead of holding the call. See [SIDECAR.md](SIDECAR.md).

Throws when the plugin has no sidecar for the running platform
(`has_sidecar` is listing-visible).

## `host.events`

### `host.events.on(event, cb): () => void`
Subscribes to your sidecar's notifications: a sidecar notification with
method `progress` reaches `host.events.on('progress', cb)` with its
`params` as the payload. Returns an unsubscribe function.

## `host.log(...args)`

Console logging tagged with your plugin id. Prefer it over bare
`console.log` so your output is attributable.

---

## Worked example: dock bar + sidecar + progress events

```js
export function activate(host) {
  const offProgress = host.events.on('progress', p => render(p))

  host.ui.mountDockBar('goalbar', {
    mount(el) {
      el.innerHTML = '<button>Recount</button><span></span>'
      el.querySelector('button').onclick = async () => {
        const res = await host.backend.invoke('recount', {
          text: host.editor.getSelectionText(),
        })
        el.querySelector('span').textContent = `${res.words} words`
      }
    },
    unmount() { /* drop DOM references */ },
  })

  return () => offProgress()
}
```
