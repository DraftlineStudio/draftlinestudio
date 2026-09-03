// Read Aloud floating player - transport controls for local TTS playback.
// Mounted only while the plugin is enabled and the player is visible; all
// state lives in readAloudStore so the toolbar, shortcuts, and editor
// highlight stay in sync with it.

import { useEffect } from 'react'
import { useAppStore } from '../../store/appStore'
import { useReadAloudStore } from '../../store/readAloudStore'
import { READ_ALOUD_VOICES } from '../../services/readaloud/voices'

const SPEED_STEPS = [0.8, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6]

export default function ReadAloudPlayer() {
  const settings = useAppStore(s => s.settings)
  const status = useReadAloudStore(s => s.status)
  const sentences = useReadAloudStore(s => s.sentences)
  const currentIndex = useReadAloudStore(s => s.currentIndex)
  const playbackError = useReadAloudStore(s => s.playbackError)
  const modelState = useReadAloudStore(s => s.modelState)
  const modelLoading = useReadAloudStore(s => s.modelLoading)
  const download = useReadAloudStore(s => s.download)
  const refreshModelStatus = useReadAloudStore(s => s.refreshModelStatus)
  const downloadModel = useReadAloudStore(s => s.downloadModel)
  const cancelDownload = useReadAloudStore(s => s.cancelDownload)
  const togglePause = useReadAloudStore(s => s.togglePause)
  const stop = useReadAloudStore(s => s.stop)
  const skip = useReadAloudStore(s => s.skip)
  const setVoice = useReadAloudStore(s => s.setVoice)
  const setSpeed = useReadAloudStore(s => s.setSpeed)
  const setPlayerVisible = useReadAloudStore(s => s.setPlayerVisible)
  const playChapter = useReadAloudStore(s => s.playChapter)

  useEffect(() => {
    void refreshModelStatus()
  }, [refreshModelStatus])

  const downloadPct = download && download.overall_total > 0
    ? Math.min(100, Math.round((download.overall_received / download.overall_total) * 100))
    : 0

  return (
    <div className="read-aloud-player">
      <div className="read-aloud-player-header">
        <span className="read-aloud-player-title">Read Aloud</span>
        <button className="read-aloud-close" onClick={() => setPlayerVisible(false)} title="Close (stops playback)">×</button>
      </div>

      {modelState === 'missing' && (
        <div className="read-aloud-player-body">
          <p className="read-aloud-note">One-time voice model download (~130 MB). Fully local afterwards.</p>
          <button className="dialog-btn primary" onClick={downloadModel}>Download voice model</button>
        </div>
      )}

      {modelState === 'downloading' && (
        <div className="read-aloud-player-body">
          <p className="read-aloud-note">Downloading voice model… {downloadPct}%</p>
          <div className="read-aloud-progress"><div className="read-aloud-progress-fill" style={{ width: `${downloadPct}%` }} /></div>
          <button className="dialog-btn" onClick={cancelDownload}>Cancel</button>
        </div>
      )}

      {(modelState === 'ready' || modelState === 'unknown') && (
        <div className="read-aloud-player-body">
          {status === 'starting' && (
            <p className="read-aloud-note">
              {modelLoading ? 'Loading voices…' : 'Preparing…'}
            </p>
          )}
          {playbackError && <p className="read-aloud-note read-aloud-error">{playbackError}</p>}

          <div className="read-aloud-transport">
            <button className="toolbar-btn" onClick={() => skip(-1)} disabled={status === 'idle'} title="Previous sentence (Ctrl+Shift+,)">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2 1h1.6v10H2zM11 1v10L4.4 6z"/></svg>
            </button>
            {status === 'idle' ? (
              <button className="toolbar-btn" onClick={playChapter} title="Read chapter">
                <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 1.5l8 4.5-8 4.5z"/></svg>
              </button>
            ) : (
              <button className="toolbar-btn" onClick={togglePause} title={status === 'paused' ? 'Resume (Ctrl+Shift+L)' : 'Pause (Ctrl+Shift+L)'}>
                {status === 'paused'
                  ? <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 1.5l8 4.5-8 4.5z"/></svg>
                  : <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M2.5 1.5h2.6v9H2.5zM6.9 1.5h2.6v9H6.9z"/></svg>}
              </button>
            )}
            <button className="toolbar-btn" onClick={stop} disabled={status === 'idle'} title="Stop">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><rect x="2" y="2" width="8" height="8" rx="1"/></svg>
            </button>
            <button className="toolbar-btn" onClick={() => skip(1)} disabled={status === 'idle'} title="Next sentence (Ctrl+Shift+.)">
              <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor"><path d="M8.4 1H10v10H8.4zM1 1l6.6 5L1 11z"/></svg>
            </button>
            <span className="read-aloud-position">
              {status !== 'idle' && currentIndex >= 0 ? `${currentIndex + 1} / ${sentences.length}` : ''}
            </span>
          </div>

          <div className="read-aloud-options">
            <select
              className="toolbar-select read-aloud-voice-select"
              value={settings.read_aloud_voice}
              onChange={e => setVoice(e.target.value)}
              title="Voice"
            >
              {READ_ALOUD_VOICES.map(v => <option key={v.id} value={v.id}>{v.label}</option>)}
            </select>
            <select
              className="toolbar-select read-aloud-speed-select"
              value={String(Math.round(settings.read_aloud_speed * 10) / 10)}
              onChange={e => setSpeed(Number(e.target.value))}
              title="Speed"
            >
              {SPEED_STEPS.map(s => <option key={s} value={String(s)}>{s.toFixed(1)}×</option>)}
            </select>
          </div>
        </div>
      )}
    </div>
  )
}
