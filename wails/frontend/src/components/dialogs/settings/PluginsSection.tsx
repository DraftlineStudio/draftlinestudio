import { useMemo, useState } from 'react'
import { FEATURE_REGISTRY, type FeatureSettingKey } from '../../../features/registry'
import { usePluginStore, isPluginEnabled } from '../../../store/pluginStore'
import { useAppStore } from '../../../store/appStore'

interface PluginsSectionProps {
  enabled: Record<FeatureSettingKey, boolean>
  onToggle: (key: FeatureSettingKey, enabled: boolean) => void
}

export default function PluginsSection({ enabled, onToggle }: PluginsSectionProps) {
  const [query, setQuery] = useState('')
  const [tab, setTab] = useState<'installed' | 'marketplace'>('installed')
  const normalizedQuery = query.trim().toLocaleLowerCase()
  // Installed platform plugins (real plugin folders, distinct from the
  // bundled core features below). Subscribing to settings keeps the toggles
  // live after setEnabled round-trips.
  useAppStore(s => s.settings.plugins_enabled)
  const installedPlugins = usePluginStore(s => s.plugins)
  const setPluginEnabled = usePluginStore(s => s.setEnabled)
  const matchedPlugins = useMemo(() => installedPlugins.filter(p => {
    if (!normalizedQuery) return true
    return `${p.name} ${p.publisher} ${p.description}`.toLocaleLowerCase().includes(normalizedQuery)
  }), [installedPlugins, normalizedQuery])
  const features = useMemo(() => FEATURE_REGISTRY.filter(feature => {
    if (!normalizedQuery) return true
    return `${feature.name} ${feature.vendor} ${feature.category} ${feature.description}`
      .toLocaleLowerCase()
      .includes(normalizedQuery)
  }), [normalizedQuery])

  return (
    <div className="plugins-settings">
      <div className="plugins-settings-heading">
        <div>
          <h2>Plugins</h2>
          <p>Enable only the tools you want in your writing workspace.</p>
        </div>
        <span className="plugins-count">{FEATURE_REGISTRY.length} bundled</span>
      </div>

      <div className="plugins-search-wrap">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>
        </svg>
        <input value={query} onChange={event => setQuery(event.target.value)} placeholder="Search plugins" aria-label="Search plugins" />
        {query && <button onClick={() => setQuery('')} aria-label="Clear search">×</button>}
      </div>

      <div className="plugins-tabs" aria-label="Plugin source">
        <button className={tab === 'installed' ? 'active' : ''} onClick={() => setTab('installed')}>Installed</button>
        <button className={tab === 'marketplace' ? 'active' : ''} onClick={() => setTab('marketplace')}>Marketplace <span>Preview</span></button>
      </div>

      <div className="plugins-list">
        {tab === 'installed' && features.map(feature => (
          <article className={`plugin-card${enabled[feature.settingKey] ? '' : ' disabled'}`} key={feature.id}>
            <div className="plugin-icon" style={{ '--plugin-accent': feature.accent } as React.CSSProperties}>{feature.monogram}</div>
            <div className="plugin-card-body">
              <div className="plugin-card-title-row">
                <div><h3>{feature.name}</h3><span>{feature.vendor} · {feature.version}</span></div>
                <label className="settings-toggle" title={`${enabled[feature.settingKey] ? 'Disable' : 'Enable'} ${feature.name}`}>
                  <input type="checkbox" checked={enabled[feature.settingKey]} onChange={event => onToggle(feature.settingKey, event.target.checked)} />
                  <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
                </label>
              </div>
              <p>{feature.description}</p>
              <div className="plugin-card-meta">
                <span>{feature.category}</span><span>Bundled</span>
                <span>{feature.resourceProfile === 'tiny' ? 'Tiny' : feature.resourceProfile === 'light' ? 'Lightweight' : 'Optional runtime'}</span>
                <span className={enabled[feature.settingKey] ? 'enabled' : ''}>{enabled[feature.settingKey] ? 'Enabled' : 'Disabled'}</span>
              </div>
            </div>
          </article>
        ))}
        {tab === 'installed' && matchedPlugins.map(plugin => (
          <article className={`plugin-card${isPluginEnabled(plugin.id) ? '' : ' disabled'}`} key={plugin.id}>
            <div className="plugin-icon" style={{ '--plugin-accent': '#7c8697' } as React.CSSProperties}>
              {plugin.name.slice(0, 2)}
            </div>
            <div className="plugin-card-body">
              <div className="plugin-card-title-row">
                <div><h3>{plugin.name}</h3><span>{plugin.publisher} · {plugin.version}</span></div>
                {!plugin.load_error && plugin.supported && (
                  <label className="settings-toggle" title={`${isPluginEnabled(plugin.id) ? 'Disable' : 'Enable'} ${plugin.name}`}>
                    <input type="checkbox" checked={isPluginEnabled(plugin.id)} onChange={event => void setPluginEnabled(plugin.id, event.target.checked)} />
                    <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
                  </label>
                )}
              </div>
              <p>{plugin.load_error ? `This plugin failed to load: ${plugin.load_error}` : plugin.description || 'No description provided.'}</p>
              <div className="plugin-card-meta">
                <span>Plugin</span>
                <span>{plugin.root === 'dev' ? 'Development' : plugin.root === 'shared' ? 'This machine' : 'This user'}</span>
                {!plugin.supported && !plugin.load_error && <span>Needs a newer Draftline (API v{plugin.api_version})</span>}
                {plugin.has_sidecar && <span>{plugin.running ? 'Running' : 'Stopped'}</span>}
                <span className={isPluginEnabled(plugin.id) ? 'enabled' : ''}>{isPluginEnabled(plugin.id) ? 'Enabled' : 'Disabled'}</span>
              </div>
            </div>
          </article>
        ))}
        {tab === 'installed' && features.length === 0 && matchedPlugins.length === 0 && <div className="plugins-empty">No installed plugins match “{query}”.</div>}
        {tab === 'marketplace' && (
          <div className="plugins-marketplace-preview">
            <div className="plugin-icon" style={{ '--plugin-accent': '#4ba39a' } as React.CSSProperties}>Mx</div>
            <div>
              <h3>Community catalog not connected</h3>
              <p>Draftline now identifies analyzer capabilities and resource profiles independently of the UI. The signed catalog, package installer, permissions, updates, and rollback are the next marketplace layer.</p>
              <span>Model packs will show download size, license, memory estimate, and manuscript access before installation.</span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
