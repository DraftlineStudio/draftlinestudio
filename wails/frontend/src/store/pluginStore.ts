// Plugin Store — the frontend half of the plugin platform host. Holds the
// installed-plugin listing from the Go side and the UI contributions that
// activated plugin bundles have mounted (dock bars, settings sections).
// Activation itself lives in services/plugins/loader.ts; this store is the
// rendering surface components subscribe to.

import { create } from 'zustand'
import { PluginList, PluginSetEnabled } from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'
import { useAppStore } from './appStore'

export type PluginInfo = main.PluginInfo

// A mounted UI contribution: the plugin supplies imperative mount/unmount
// against a bare DOM element (never the React tree — no framework coupling).
export interface MountedContribution {
  pluginId: string
  id: string
  title?: string
  mount: (el: HTMLElement) => void
  unmount?: () => void
}

interface PluginStore {
  plugins: PluginInfo[]
  loaded: boolean
  dockBars: MountedContribution[]
  settingsSections: MountedContribution[]

  refresh: () => Promise<PluginInfo[]>
  setEnabled: (pluginId: string, enabled: boolean) => Promise<void>

  registerDockBar: (c: MountedContribution) => void
  registerSettingsSection: (c: MountedContribution) => void
  // Removes every contribution a plugin has mounted (deactivation).
  removeContributions: (pluginId: string) => void
}

export function isPluginEnabled(pluginId: string): boolean {
  return useAppStore.getState().settings.plugins_enabled?.[pluginId] ?? false
}

export const usePluginStore = create<PluginStore>((set, get) => ({
  plugins: [],
  loaded: false,
  dockBars: [],
  settingsSections: [],

  refresh: async () => {
    try {
      const plugins = await PluginList()
      set({ plugins, loaded: true })
      return plugins
    } catch (e) {
      console.error('Plugin listing failed:', e)
      set({ loaded: true })
      return get().plugins
    }
  },

  setEnabled: async (pluginId, enabled) => {
    // Go persists the flag (authoritative for sidecar gating); mirror it into
    // the in-memory settings so gates react without a settings round-trip.
    await PluginSetEnabled(pluginId, enabled)
    const app = useAppStore.getState()
    app.saveSettings({
      plugins_enabled: { ...(app.settings.plugins_enabled ?? {}), [pluginId]: enabled },
    })
    const { activatePlugin, deactivatePlugin } = await import('../services/plugins/loader')
    if (enabled) await activatePlugin(pluginId)
    else deactivatePlugin(pluginId)
    await get().refresh()
  },

  registerDockBar: (c) => set(s => ({ dockBars: [...s.dockBars.filter(x => !(x.pluginId === c.pluginId && x.id === c.id)), c] })),
  registerSettingsSection: (c) => set(s => ({ settingsSections: [...s.settingsSections.filter(x => !(x.pluginId === c.pluginId && x.id === c.id)), c] })),
  removeContributions: (pluginId) => {
    const s = get()
    for (const c of [...s.dockBars, ...s.settingsSections]) {
      if (c.pluginId === pluginId) {
        try { c.unmount?.() } catch (e) { console.error(`Plugin ${pluginId} unmount failed:`, e) }
      }
    }
    set({
      dockBars: s.dockBars.filter(c => c.pluginId !== pluginId),
      settingsSections: s.settingsSections.filter(c => c.pluginId !== pluginId),
    })
  },
}))
