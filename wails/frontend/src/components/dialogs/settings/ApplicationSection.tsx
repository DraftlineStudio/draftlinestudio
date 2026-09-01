// Application Settings Section - Identity, Theme, Save Location

import type { ApplicationSectionProps } from './types'

export default function ApplicationSection({
  author, setAuthor,
  publisher, setPublisher,
  copyright, setCopyright,
  saveDir, setSaveDir,
  themeMode, setThemeMode,
  autoThemeUseManual, setAutoThemeUseManual,
  autoThemeDawn, setAutoThemeDawn,
  autoThemeDusk, setAutoThemeDusk,
  activityAutoSaveEnabled, setActivityAutoSaveEnabled,
  analysisCPUProfile, setAnalysisCPUProfile,
  onBrowse,
}: ApplicationSectionProps) {
  return (
    <>
      <div className="settings-section-label" style={{ marginTop: 0 }}>Identity</div>
      <div className="dialog-field">
        <label className="dialog-label">Default Author Name</label>
        <input className="dialog-input" value={author} onChange={e => setAuthor(e.target.value)} placeholder="Your name" autoFocus />
        <div className="settings-hint">Pre-filled when creating a new book.</div>
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Default Publisher</label>
        <input className="dialog-input" value={publisher} onChange={e => setPublisher(e.target.value)} placeholder="Publisher or imprint name" />
      </div>

      <div className="settings-section-label">Copyright Template</div>
      <div className="dialog-field">
        <label className="dialog-label">Default Copyright Text</label>
        <textarea
          className="dialog-input settings-copyright-textarea"
          value={copyright}
          onChange={e => setCopyright(e.target.value)}
          placeholder={"Copyright © [YEAR] [AUTHOR]. All rights reserved.\n\nNo part of this publication may be reproduced..."}
          rows={5}
        />
        <div className="settings-hint">Inserted into the Copyright page of every new book. Use [YEAR] and [AUTHOR] as placeholders.</div>
      </div>

      <div className="settings-section-label">Interface</div>
      <div className="dialog-field">
        <label className="dialog-label">Theme</label>
        <div className="settings-theme-row">
          <button className={`settings-theme-btn${themeMode === 'light' ? ' active' : ''}`} onClick={() => setThemeMode('light')}>
            <svg width="11" height="11" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.5">
              <circle cx="7" cy="7" r="2.8"/>
              <line x1="7" y1="1" x2="7" y2="2.4"/><line x1="7" y1="11.6" x2="7" y2="13"/>
              <line x1="1" y1="7" x2="2.4" y2="7"/><line x1="11.6" y1="7" x2="13" y2="7"/>
              <line x1="2.9" y1="2.9" x2="3.9" y2="3.9"/><line x1="10.1" y1="10.1" x2="11.1" y2="11.1"/>
              <line x1="11.1" y1="2.9" x2="10.1" y2="3.9"/><line x1="3.9" y1="10.1" x2="2.9" y2="11.1"/>
            </svg>
            Light
          </button>
          <button className={`settings-theme-btn${themeMode === 'dark' ? ' active' : ''}`} onClick={() => setThemeMode('dark')}>
            <svg width="11" height="11" viewBox="0 0 14 14" fill="currentColor">
              <path d="M7 1a6 6 0 1 0 0 12A6 6 0 0 0 7 1zm0 1.5A4.5 4.5 0 1 1 7 11.5V2.5z"/>
            </svg>
            Dark
          </button>
          <button className={`settings-theme-btn${themeMode === 'auto' ? ' active' : ''}`} onClick={() => setThemeMode('auto')}>
            <svg width="11" height="11" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
              <circle cx="7" cy="7" r="5.5"/>
              <path d="M7 1.5V7l3.5 2"/>
            </svg>
            Auto
          </button>
        </div>
      </div>
      {themeMode === 'auto' && (
        <div className="settings-auto-theme">
          <div className="settings-hint" style={{ marginBottom: 12 }}>
            Automatically switches between light and dark themes based on time of day.
          </div>
          <div className="dialog-field">
            <label className="dialog-label" style={{ marginBottom: 8 }}>Time Source</label>
            <label className="settings-toggle">
              <input type="checkbox" checked={autoThemeUseManual} onChange={e => setAutoThemeUseManual(e.target.checked)} />
              <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
              <span className="settings-toggle-label">{autoThemeUseManual ? 'Manual times' : 'Automatic (based on location)'}</span>
            </label>
          </div>
          {autoThemeUseManual && (
            <div className="settings-two-col" style={{ marginTop: 12 }}>
              <div className="dialog-field">
                <label className="dialog-label">Dawn (light mode starts)</label>
                <input
                  className="dialog-input"
                  type="time"
                  value={autoThemeDawn}
                  onChange={e => setAutoThemeDawn(e.target.value)}
                />
              </div>
              <div className="dialog-field">
                <label className="dialog-label">Dusk (dark mode starts)</label>
                <input
                  className="dialog-input"
                  type="time"
                  value={autoThemeDusk}
                  onChange={e => setAutoThemeDusk(e.target.value)}
                />
              </div>
            </div>
          )}
          {!autoThemeUseManual && (
            <div className="settings-hint" style={{ marginTop: 8 }}>
              Sunrise and sunset times are fetched automatically when the app starts.
              This sends your device's IP address to ipwho.is to estimate your location,
              and the derived coordinates to sunrise-sunset.org.
            </div>
          )}
        </div>
      )}
      <div className="dialog-field">
        <label className="dialog-label">Default Save Location</label>
        <div className="settings-path-row">
          <input className="dialog-input" value={saveDir} onChange={e => setSaveDir(e.target.value)} placeholder="Leave blank to use system default" />
          <button className="dialog-btn settings-browse-btn" onClick={onBrowse}>Browse…</button>
        </div>
      </div>
      <div className="settings-section-label">Saving &amp; Recovery</div>
      <div className="dialog-field">
        <label className="settings-toggle">
          <input
            type="checkbox"
            checked={activityAutoSaveEnabled}
            onChange={e => setActivityAutoSaveEnabled(e.target.checked)}
          />
          <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
          <span className="settings-toggle-label">Activity-based saving</span>
        </label>
        <div className="settings-hint">
          Saves paused edits and records changed chapter versions during active writing. When disabled, Draftline saves only when you ask it to.
        </div>
      </div>
      <div className="settings-section-label">Background Analysis</div>
      <div className="dialog-field">
        <label className="dialog-label">CPU usage</label>
        <select
          className="dialog-select"
          value={analysisCPUProfile}
          onChange={event => setAnalysisCPUProfile(event.target.value as typeof analysisCPUProfile)}
        >
          <option value="adaptive">Adaptive (recommended)</option>
          <option value="gentle">Gentle — 1–2 analysis cores</option>
          <option value="balanced">Balanced — up to 4 analysis cores</option>
          <option value="fast">Fast — use available CPU</option>
        </select>
        <div className="settings-hint">
          Adaptive automatically switches large manuscripts to Gentle. These are concurrency budgets rather than exact CPU percentages; Draftline leaves capacity for the editor in Adaptive, Gentle, and Balanced modes.
        </div>
      </div>
    </>
  )
}
