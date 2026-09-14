// Application Settings Section - Updates, Theme, Saving & Recovery, Analysis

import { useEffect, useState } from 'react'
import { CheckForUpdates, DownloadUpdate } from '../../../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../../../wailsjs/runtime/runtime'
import type { main } from '../../../../wailsjs/go/models'
import { useAppStore } from '../../../store/appStore'
import { checkNow } from '../../../services/updateNag'
import type { ApplicationSectionProps } from './types'

export default function ApplicationSection({
  saveDir, setSaveDir,
  themeMode, setThemeMode,
  autoThemeUseManual, setAutoThemeUseManual,
  autoThemeDawn, setAutoThemeDawn,
  autoThemeDusk, setAutoThemeDusk,
  activityAutoSaveEnabled, setActivityAutoSaveEnabled,
  analysisCPUProfile, setAnalysisCPUProfile,
  onBrowse,
}: ApplicationSectionProps) {
  const updateCheckEnabled = useAppStore(s => s.settings.update_check_enabled)
  const saveAppSettings = useAppStore(s => s.saveSettings)
  const knownUpdate = useAppStore(s => s.updateAvailable)
  const setUpdateAvailable = useAppStore(s => s.setUpdateAvailable)
  // The background check already knows whether a newer version exists. Seed
  // from it so the title-bar chip leads straight to a ready Download button
  // instead of a second manual check.
  const [updateState, setUpdateState] = useState<'idle' | 'checking' | 'checked' | 'downloading' | 'downloaded'>(knownUpdate ? 'checked' : 'idle')
  const [updateResult, setUpdateResult] = useState<main.UpdateCheckResult | null>(knownUpdate?.result ?? null)
  const [updateError, setUpdateError] = useState('')
  const [downloadPath, setDownloadPath] = useState('')

  const runUpdateCheck = async () => {
    setUpdateState('checking')
    setUpdateError('')
    setDownloadPath('')
    try {
      const result = await CheckForUpdates()
      setUpdateResult(result)
      setUpdateError(result.error || '')
      setUpdateState('checked')
      if (result.update_available && result.latest_label && !result.error) {
        setUpdateAvailable({ label: result.latest_label, result })
      } else if (!result.error) {
        setUpdateAvailable(null)
      }
    } catch {
      setUpdateError('Could not check for updates.')
      setUpdateState('checked')
    }
  }

  // With automatic checks on, opening this section is itself the ask: run a
  // check when nothing is known yet so the answer is waiting, not a button.
  useEffect(() => {
    if (!knownUpdate && updateCheckEnabled) void runUpdateCheck()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const runUpdateDownload = async () => {
    setUpdateState('downloading')
    setUpdateError('')
    try {
      const result = await DownloadUpdate()
      if (result.error) {
        setUpdateError(result.error)
        setUpdateState('checked')
        return
      }
      setDownloadPath(result.path || '')
      setUpdateState('downloaded')
    } catch {
      setUpdateError('The download failed.')
      setUpdateState('checked')
    }
  }

  const updateReady = !updateError && updateResult?.update_available && (updateState === 'checked' || updateState === 'downloading')

  return (
    <>
      <div className="settings-section-label" style={{ marginTop: 0 }}>Updates</div>
      <div className="dialog-field">
        <div className="settings-path-row">
          {updateReady && (
            <button
              className="dialog-btn primary settings-browse-btn"
              onClick={() => void runUpdateDownload()}
              disabled={updateState === 'downloading'}
            >
              {updateState === 'downloading' ? 'Downloading…' : `Download ${updateResult?.latest_label ?? ''}`}
            </button>
          )}
          <button
            className="dialog-btn settings-browse-btn"
            onClick={() => void runUpdateCheck()}
            disabled={updateState === 'checking' || updateState === 'downloading'}
          >
            {updateState === 'checking' ? 'Checking…' : updateReady ? 'Check again' : 'Check for Updates'}
          </button>
        </div>
        {updateError && <div className="settings-hint">{updateError}</div>}
        {!updateError && updateState === 'checked' && updateResult && !updateResult.update_available && (
          <div className="settings-hint">You're up to date ({updateResult.current_version}).</div>
        )}
        {updateReady && updateResult && (
          <div className="settings-hint">
            Version {updateResult.latest_label} is ready to download (you have {updateResult.current_version}).{' '}
            {updateResult.release_url && (
              <a
                href="#"
                onClick={event => { event.preventDefault(); BrowserOpenURL(updateResult.release_url!) }}
              >
                Release notes
              </a>
            )}
            {!updateResult.asset_name && ' No package is published for this platform yet — use the release page.'}
          </div>
        )}
        {updateState === 'downloaded' && (
          <div className="settings-hint">
            Downloaded and verified{downloadPath ? `: ${downloadPath}` : '.'} Draftline will close in a moment so the update can be applied, then reopen when it finishes — you'll be prompted first if you have unsaved changes. On Linux the package manager runs in a terminal window and may ask for your password.
          </div>
        )}
        {updateState === 'idle' && (
          <div className="settings-hint">
            Checks the Draftline releases page on GitHub. Nothing is downloaded unless you ask.
          </div>
        )}
      </div>
      <div className="dialog-field">
        <label className="settings-toggle">
          <input
            type="checkbox"
            checked={updateCheckEnabled}
            onChange={e => {
              const enabled = e.target.checked
              void saveAppSettings({ update_check_enabled: enabled })
              if (enabled) checkNow()
              else setUpdateAvailable(null)
            }}
          />
          <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
          <span className="settings-toggle-label">Check for updates automatically</span>
        </label>
        <div className="settings-hint">
          Checks the GitHub releases page on launch and once an hour, and shows a small arrow in the title bar when a newer version exists. Nothing downloads without you asking.
        </div>
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
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
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
      <div className="dialog-field">
        <label className="dialog-label">Default Save Location</label>
        <div className="settings-path-row">
          <input className="dialog-input" value={saveDir} onChange={e => setSaveDir(e.target.value)} placeholder="Leave blank to use system default" />
          <button className="dialog-btn settings-browse-btn" onClick={onBrowse}>Browse…</button>
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
          <option value="gentle">Gentle — 1–2 analysis workers</option>
          <option value="balanced">Balanced — up to 4 analysis workers</option>
          <option value="fast">Fast — one worker per logical CPU</option>
        </select>
        <div className="settings-hint">
          These limits apply only to linguistic and evidence-analysis workers. They never constrain Draftline's scheduler, bindings, files, asset server, or Read Aloud. Adaptive uses Balanced workers and adds a bounded-memory queue for large manuscripts.
        </div>
      </div>
    </>
  )
}
