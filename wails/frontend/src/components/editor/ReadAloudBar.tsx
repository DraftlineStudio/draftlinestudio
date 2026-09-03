// Read Aloud player bar - docked at the bottom of the editor column, the
// bottom counterpart of the chapter find bar. All state lives in
// readAloudStore; the bar renders only while the plugin is enabled and the
// player is open. When the voice model is missing it offers setup instead of
// transport controls (mirroring the AI Studio "set up" flow).

import { useEffect } from 'react'
import { useAppStore } from '../../store/appStore'
import { useReadAloudStore } from '../../store/readAloudStore'
import { READ_ALOUD_VOICES } from '../../services/readaloud/voices'

const SPEED_STEPS = [0.8, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6]

export default function ReadAloudBar() {
  const settings = useAppStore(s => s.settings)
  const openSettings = useAppStore(s => s.openSettings)
  const status = useReadAloudStore(s => s.status)
  const sentences = useReadAloudStore(s => s.sentences)
  const currentIndex = useReadAloudStore(s => s.currentIndex)
  const playbackError = useReadAloudStore(s => s.playbackError)
  const modelState = useReadAloudStore(s => s.modelState)
  const modelLoading = useReadAloudStore(s => s.modelLoading)
  const refreshModelStatus = useReadAloudStore(s => s.refreshModelStatus)
  const playFromCursor = useReadAloudStore(s => s.playFromCursor)
  const playChapter = useReadAloudStore(s => s.playChapter)
  const togglePause = useReadAloudStore(s => s.togglePause)
  const stop = useReadAloudStore(s => s.stop)
  const skip = useReadAloudStore(s => s.skip)
  const setVoice = useReadAloudStore(s => s.setVoice)
  const setSpeed = useReadAloudStore(s => s.setSpeed)
  const setPlayerVisible = useReadAloudStore(s => s.setPlayerVisible)

  useEffect(() => {
    void refreshModelStatus()
  }, [refreshModelStatus])

  const active = status !== 'idle'
  const needsSetup = modelState === 'missing' || modelState === 'error' || modelState === 'downloading'

  return (
    <div className="read-aloud-bar" role="region" aria-label="Read aloud player">
      <span className="read-aloud-bar-scope">Read aloud</span>

      {needsSetup ? (
        <>
          <span className="read-aloud-bar-status">
            {modelState === 'downloading'
              ? 'Voice model downloading…'
              : 'Voice model not installed.'}
          </span>
          <button className="read-aloud-bar-setup" onClick={() => openSettings('readaloud')}>
            {modelState === 'downloading' ? 'View progress' : 'Set up'}
          </button>
        </>
      ) : (
        <>
          <button
            className="read-aloud-bar-btn"
            onClick={() => skip(-1)}
            disabled={!active}
            title="Previous sentence (Ctrl+Shift+,)"
            aria-label="Previous sentence"
          >
            <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2 1h1.6v10H2zM11 1v10L4.4 6z"/></svg>
          </button>
          {active ? (
            <button
              className="read-aloud-bar-btn"
              onClick={togglePause}
              title={status === 'paused' ? 'Resume (Ctrl+Shift+L)' : 'Pause (Ctrl+Shift+L)'}
              aria-label={status === 'paused' ? 'Resume' : 'Pause'}
            >
              {status === 'paused'
                ? <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 1.5l8 4.5-8 4.5z"/></svg>
                : <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 1.5h2.6v9H2.5zM6.9 1.5h2.6v9H6.9z"/></svg>}
            </button>
          ) : (
            <button
              className="read-aloud-bar-btn"
              onClick={playFromCursor}
              title="Read from cursor (Ctrl+Shift+L)"
              aria-label="Read from cursor"
            >
              <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 1.5l8 4.5-8 4.5z"/></svg>
            </button>
          )}
          <button
            className="read-aloud-bar-btn"
            onClick={stop}
            disabled={!active}
            title="Stop"
            aria-label="Stop"
          >
            <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><rect x="2" y="2" width="8" height="8" rx="1"/></svg>
          </button>
          <button
            className="read-aloud-bar-btn"
            onClick={() => skip(1)}
            disabled={!active}
            title="Next sentence (Ctrl+Shift+.)"
            aria-label="Next sentence"
          >
            <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M8.4 1H10v10H8.4zM1 1l6.6 5L1 11z"/></svg>
          </button>

          <span className="read-aloud-bar-position">
            {active && currentIndex >= 0 ? `${currentIndex + 1} / ${sentences.length}` : ''}
          </span>

          <span className="read-aloud-bar-status">
            {status === 'starting' ? (modelLoading ? 'Loading voices…' : 'Preparing…') : ''}
            {playbackError ?? ''}
          </span>

          {!active && (
            <button className="read-aloud-bar-text" onClick={playChapter}>Read chapter</button>
          )}

          <select
            className="read-aloud-bar-select"
            value={settings.read_aloud_voice}
            onChange={e => setVoice(e.target.value)}
            title="Voice"
            aria-label="Voice"
          >
            {READ_ALOUD_VOICES.map(v => <option key={v.id} value={v.id}>{v.label}</option>)}
          </select>
          <select
            className="read-aloud-bar-select read-aloud-bar-select--speed"
            value={String(Math.round(settings.read_aloud_speed * 10) / 10)}
            onChange={e => setSpeed(Number(e.target.value))}
            title="Speed"
            aria-label="Speed"
          >
            {SPEED_STEPS.map(s => <option key={s} value={String(s)}>{s.toFixed(1)}×</option>)}
          </select>
        </>
      )}

      <button
        className="read-aloud-bar-btn read-aloud-bar-close"
        onClick={() => setPlayerVisible(false)}
        title="Close (stops playback)"
        aria-label="Close read aloud"
      >
        ×
      </button>
    </div>
  )
}
