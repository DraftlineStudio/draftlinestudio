// Read Aloud Settings Section - voice, speed, and voice model management

import { useEffect } from 'react'
import type { ReadAloudSectionProps } from './types'
import { READ_ALOUD_VOICES } from '../../../services/readaloud/voices'
import { useReadAloudStore } from '../../../store/readAloudStore'

function formatMB(bytes: number): string {
  return `${Math.round(bytes / (1024 * 1024))} MB`
}

export default function ReadAloudSection({
  readAloudEnabled,
  voice, setVoice,
  speed, setSpeed,
}: ReadAloudSectionProps) {
  const modelState = useReadAloudStore(s => s.modelState)
  const bytesTotal = useReadAloudStore(s => s.bytesTotal)
  const download = useReadAloudStore(s => s.download)
  const modelError = useReadAloudStore(s => s.modelError)
  const refreshModelStatus = useReadAloudStore(s => s.refreshModelStatus)
  const downloadModel = useReadAloudStore(s => s.downloadModel)
  const cancelDownload = useReadAloudStore(s => s.cancelDownload)
  const removeModel = useReadAloudStore(s => s.removeModel)

  useEffect(() => {
    void refreshModelStatus()
  }, [refreshModelStatus])

  const pct = download && download.overall_total > 0
    ? Math.min(100, Math.round((download.overall_received / download.overall_total) * 100))
    : 0

  return (
    <>
      <div className="settings-section-label" style={{ marginTop: 0 }}>Read Aloud</div>
      <div className="settings-hint" style={{ marginBottom: 12 }}>
        Reads your manuscript aloud with the Kokoro voice model (Apache 2.0), entirely on this
        machine. The only network use is the one-time model download below; at playback nothing
        leaves your computer.
      </div>
      {!readAloudEnabled && (
        <div className="settings-hint" style={{ marginBottom: 12 }}>
          The Read Aloud plugin is disabled. Enable it under Plugins to use these settings.
        </div>
      )}

      <div className="dialog-field">
        <label className="dialog-label">Voice</label>
        <select className="dialog-input" value={voice} onChange={e => setVoice(e.target.value)} disabled={!readAloudEnabled}>
          {READ_ALOUD_VOICES.map(v => (
            <option key={v.id} value={v.id}>{v.label}</option>
          ))}
        </select>
      </div>

      <div className="dialog-field">
        <label className="dialog-label">Speed: {speed.toFixed(1)}×</label>
        <input
          type="range" className="read-aloud-speed-slider"
          min={0.8} max={1.6} step={0.1} value={speed}
          onChange={e => setSpeed(Number(e.target.value))}
          disabled={!readAloudEnabled}
        />
        <div className="settings-hint">0.8× – 1.6×. The default of 1.2× suits most prose.</div>
      </div>

      <div className="settings-section-label">Voice Model</div>
      <div className="dialog-field">
        {modelState === 'ready' && (
          <>
            <div className="settings-hint" style={{ marginBottom: 8 }}>
              Voice model installed ({formatMB(bytesTotal)}).
            </div>
            <button className="dialog-btn" onClick={() => void removeModel()}>Remove downloaded model</button>
          </>
        )}
        {(modelState === 'missing' || modelState === 'error') && (
          <>
            <div className="settings-hint" style={{ marginBottom: 8 }}>
              Not downloaded. One-time download of about {bytesTotal > 0 ? formatMB(bytesTotal) : '130 MB'} from
              pinned, checksum-verified sources; it can be cancelled and resumed.
            </div>
            <button className="dialog-btn primary" onClick={downloadModel} disabled={!readAloudEnabled}>
              Download voice model
            </button>
          </>
        )}
        {modelState === 'downloading' && (
          <>
            <div className="settings-hint" style={{ marginBottom: 8 }}>
              Downloading… {download ? `${formatMB(download.overall_received)} of ${formatMB(download.overall_total)} (${pct}%)` : 'starting'}
            </div>
            <div className="read-aloud-progress"><div className="read-aloud-progress-fill" style={{ width: `${pct}%` }} /></div>
            <button className="dialog-btn" style={{ marginTop: 8 }} onClick={cancelDownload}>Cancel</button>
          </>
        )}
        {modelState === 'unknown' && <div className="settings-hint">Checking model status…</div>}
        {modelError && <div className="settings-hint read-aloud-error">{modelError}</div>}
      </div>
    </>
  )
}
