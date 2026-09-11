// Plugin loader: fetches the installed-plugin listing, honors activation
// events, dynamically imports enabled plugins' frontend bundles same-origin
// from /plugins/<id>/…, and drives activate/deactivate. A plugin bundle's
// entry module must export `activate(host)` and may return (or export) a
// `deactivate` function.

import { useAppStore } from '../../store/appStore'
import { useBookStore } from '../../store/bookStore'
import { isPluginEnabled, usePluginStore, type PluginInfo } from '../../store/pluginStore'
import { buildHost, dropPluginCommands, firePluginCommand, HOST_API_VERSION } from './hostApi'

interface ActivePlugin {
  info: PluginInfo
  deactivate?: () => void
}

const active = new Map<string, ActivePlugin>()
let bootstrapped = false

function pluginById(id: string): PluginInfo | undefined {
  return usePluginStore.getState().plugins.find(p => p.id === id)
}

function activationEvents(info: PluginInfo): string[] {
  // No declared activation events means "activate at startup when enabled" —
  // the simple default for simple plugins.
  return info.activation?.length ? info.activation : ['onStartupIfEnabled']
}

function activatable(info: PluginInfo): boolean {
  return !info.load_error && info.has_frontend && info.supported && isPluginEnabled(info.id)
}

// activatePlugin loads and activates one plugin now (idempotent).
export async function activatePlugin(id: string): Promise<void> {
  if (active.has(id)) return
  const info = pluginById(id)
  if (!info || !activatable(info)) return
  if (info.api_version !== HOST_API_VERSION) return
  try {
    const mod = await import(/* @vite-ignore */ info.frontend_url)
    const activate = mod.activate ?? mod.default
    if (typeof activate !== 'function') {
      throw new Error('entry module exports no activate(host) function')
    }
    const host = buildHost(info)
    const result = await activate(host)
    const deactivate = typeof result === 'function' ? result : mod.deactivate
    active.set(id, { info, deactivate: typeof deactivate === 'function' ? deactivate : undefined })
    console.log(`[plugin ${id}] activated (v${info.version})`)
  } catch (e) {
    console.error(`Plugin ${id} failed to activate:`, e)
  }
}

export function deactivatePlugin(id: string): void {
  const entry = active.get(id)
  if (!entry) return
  active.delete(id)
  try {
    entry.deactivate?.()
  } catch (e) {
    console.error(`Plugin ${id} deactivate failed:`, e)
  }
  dropPluginCommands(id)
  usePluginStore.getState().removeContributions(id)
}

export function isPluginActive(id: string): boolean {
  return active.has(id)
}

// ensureActivatedFor activates every enabled plugin that declared the given
// activation event and isn't running yet.
async function ensureActivatedFor(event: string): Promise<void> {
  const plugins = usePluginStore.getState().plugins
  for (const info of plugins) {
    if (!active.has(info.id) && activatable(info) && activationEvents(info).includes(event)) {
      await activatePlugin(info.id)
    }
  }
}

// notifySettingsOpened is called by the settings dialog so plugins with the
// onSettingsOpen activation event load lazily when their section could show.
export function notifySettingsOpened(): void {
  void ensureActivatedFor('onSettingsOpen')
}

// ── Shortcuts ────────────────────────────────────────────────────────────────

interface ShortcutBinding {
  pluginId: string
  commandId: string
  key: string // normalized "mod+shift+l"
}

let shortcutBindings: ShortcutBinding[] = []

function normalizeShortcut(shortcut: string): string {
  return shortcut.trim().toLowerCase().split('+').map(s => s.trim()).sort().join('+')
}

function eventShortcut(e: KeyboardEvent): string {
  const parts: string[] = []
  if (e.ctrlKey || e.metaKey) parts.push('mod')
  if (e.shiftKey) parts.push('shift')
  if (e.altKey) parts.push('alt')
  const key = e.key.toLowerCase()
  if (!['control', 'meta', 'shift', 'alt'].includes(key)) parts.push(key)
  return parts.sort().join('+')
}

function rebuildShortcuts(): void {
  shortcutBindings = []
  for (const info of usePluginStore.getState().plugins) {
    if (info.load_error || !info.supported) continue
    for (const c of info.contributes?.commands ?? []) {
      if (c.shortcut) {
        shortcutBindings.push({ pluginId: info.id, commandId: c.id, key: normalizeShortcut(c.shortcut) })
      }
    }
  }
}

function onKeydown(e: KeyboardEvent): void {
  if (shortcutBindings.length === 0) return
  const pressed = eventShortcut(e)
  for (const b of shortcutBindings) {
    if (b.key !== pressed || !isPluginEnabled(b.pluginId)) continue
    e.preventDefault()
    void (async () => {
      // onCommand activation: load the plugin the first time its shortcut
      // fires.
      if (!active.has(b.pluginId)) await activatePlugin(b.pluginId)
      firePluginCommand(b.pluginId, b.commandId)
    })()
    return
  }
}

// ── Bootstrap ────────────────────────────────────────────────────────────────

// initPlugins runs once at app startup (after settings load): list installs,
// wire activation triggers, activate startup plugins.
export async function initPlugins(): Promise<void> {
  if (bootstrapped) return
  bootstrapped = true

  await usePluginStore.getState().refresh()
  rebuildShortcuts()
  window.addEventListener('keydown', onKeydown)

  // Keep shortcut bindings current when the listing refreshes (enable/disable
  // re-lists via pluginStore.setEnabled).
  usePluginStore.subscribe(() => rebuildShortcuts())

  // onBookOpen activation.
  let lastPath = useBookStore.getState().book?.file_path
  useBookStore.subscribe((state) => {
    const path = state.book?.file_path
    if (path && path !== lastPath) {
      lastPath = path
      void ensureActivatedFor('onBookOpen')
    }
  })

  await ensureActivatedFor('onStartupIfEnabled')
}
