import { useMemo, useState } from 'react'
import { FEATURE_REGISTRY, type FeatureSettingKey } from '../../../features/registry'

interface PluginsSectionProps {
  enabled: Record<FeatureSettingKey, boolean>
  onToggle: (key: FeatureSettingKey, enabled: boolean) => void
}

export default function PluginsSection({ enabled, onToggle }: PluginsSectionProps) {
  const [query, setQuery] = useState('')
  const normalizedQuery = query.trim().toLocaleLowerCase()
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
        <input
          value={query}
          onChange={event => setQuery(event.target.value)}
          placeholder="Search installed plugins"
          aria-label="Search installed plugins"
        />
        {query && <button onClick={() => setQuery('')} aria-label="Clear search">×</button>}
      </div>

      <div className="plugins-tabs" aria-label="Plugin source">
        <button className="active">Installed</button>
        <button disabled title="Community plugins are planned for a future release">Marketplace <span>Soon</span></button>
      </div>

      <div className="plugins-list">
        {features.map(feature => (
          <article className={`plugin-card${enabled[feature.settingKey] ? '' : ' disabled'}`} key={feature.id}>
            <div className="plugin-icon" style={{ '--plugin-accent': feature.accent } as React.CSSProperties}>
              {feature.monogram}
            </div>
            <div className="plugin-card-body">
              <div className="plugin-card-title-row">
                <div>
                  <h3>{feature.name}</h3>
                  <span>{feature.vendor} · {feature.version}</span>
                </div>
                <label className="settings-toggle" title={`${enabled[feature.settingKey] ? 'Disable' : 'Enable'} ${feature.name}`}>
                  <input
                    type="checkbox"
                    checked={enabled[feature.settingKey]}
                    onChange={event => onToggle(feature.settingKey, event.target.checked)}
                  />
                  <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
                </label>
              </div>
              <p>{feature.description}</p>
              <div className="plugin-card-meta">
                <span>{feature.category}</span>
                <span>Bundled</span>
                <span className={enabled[feature.settingKey] ? 'enabled' : ''}>
                  {enabled[feature.settingKey] ? 'Enabled' : 'Disabled'}
                </span>
              </div>
            </div>
          </article>
        ))}
        {features.length === 0 && (
          <div className="plugins-empty">No installed plugins match “{query}”.</div>
        )}
      </div>
    </div>
  )
}
