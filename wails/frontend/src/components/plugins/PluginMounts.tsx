// Render surfaces for plugin UI contributions. Plugins mount imperatively
// into a bare DOM element (never our React tree), so each contribution gets
// one stable container div whose lifecycle drives mount/unmount.

import { useEffect, useRef } from 'react'
import { usePluginStore, type MountedContribution } from '../../store/pluginStore'

function ContributionMount({ contribution }: { contribution: MountedContribution }) {
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    try {
      contribution.mount(el)
    } catch (e) {
      console.error(`Plugin ${contribution.pluginId} mount "${contribution.id}" failed:`, e)
    }
    return () => {
      try {
        contribution.unmount?.()
      } catch (e) {
        console.error(`Plugin ${contribution.pluginId} unmount "${contribution.id}" failed:`, e)
      }
      el.replaceChildren()
    }
  }, [contribution])
  return <div ref={ref} className="plugin-mount" data-plugin={contribution.pluginId} data-contribution={contribution.id} />
}

// Editor dock bars (e.g. a player bar) — rendered at the editor's dock slot.
export function PluginDockBars() {
  const dockBars = usePluginStore(s => s.dockBars)
  if (dockBars.length === 0) return null
  return <>{dockBars.map(c => <ContributionMount key={c.pluginId + ' ' + c.id} contribution={c} />)}</>
}

// One plugin-contributed settings section body.
export function PluginSettingsSection({ pluginId, sectionId }: { pluginId: string; sectionId: string }) {
  const section = usePluginStore(s => s.settingsSections.find(c => c.pluginId === pluginId && c.id === sectionId))
  if (!section) {
    return <div className="plugin-mount-pending">This plugin section hasn't loaded. Is the plugin enabled?</div>
  }
  return <ContributionMount contribution={section} />
}
