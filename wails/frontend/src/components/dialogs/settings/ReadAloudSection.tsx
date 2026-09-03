// Read Aloud Settings Section - voice model setup, voice, performance.
// Only rendered while the plugin is enabled (the nav entry is gated).
// Mirrors the AI Studio setup-card pattern: a prerequisite card with status
// badge and guided download, then the configuration fields.

import { useEffect, useState } from 'react'
import type { ReadAloudSectionProps } from './types'
import { READ_ALOUD_VOICES } from '../../../services/readaloud/voices'
import { useReadAloudStore } from '../../../store/readAloudStore'

const SPEED_STEPS = [0.8, 0.9, 1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6]

function formatMB(bytes: number): string {
  return `${Math.round(bytes / (1024 * 1024))} MB`
}

export default function ReadAloudSection({
  voice, setVoice,
  speed, setSpeed,
  device, setDevice,
  threads, setThreads,
}: ReadAloudSectionProps) {
  const modelState = useReadAloudStore(s => s.modelState)
  const bytesTotal = useReadAloudStore(s => s.bytesTotal)
  const download = useReadAloudStore(s => s.download)
  const modelError = useReadAloudStore(s => s.modelError)
  const diagnostics = useReadAloudStore(s => s.diagnostics)
  const gpuInstalled = useReadAloudStore(s => s.gpuInstalled)
  const gpuBytesTotal = useReadAloudStore(s => s.gpuBytesTotal)
  const benchmarking = useReadAloudStore(s => s.benchmarking)
  const refreshModelStatus = useReadAloudStore(s => s.refreshModelStatus)
  const downloadModel = useReadAloudStore(s => s.downloadModel)
  const downloadGPUModel = useReadAloudStore(s => s.downloadGPUModel)
  const cancelDownload = useReadAloudStore(s => s.cancelDownload)
  const removeModel = useReadAloudStore(s => s.removeModel)
  const runBenchmark = useReadAloudStore(s => s.runBenchmark)
  const [showDiagnostics, setShowDiagnostics] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    void refreshModelStatus()
  }, [refreshModelStatus])

  const pct = download && download.overall_total > 0
    ? Math.min(100, Math.round((download.overall_received / download.overall_total) * 100))
    : 0
  const sizeLabel = bytesTotal > 0 ? formatMB(bytesTotal) : '~130 MB'

  return (
    <>
      <div className="settings-section-label" style={{ marginTop: 0 }}>Voice Model</div>

      <div className="settings-cc-card">
        <div className="settings-cc-header">
          <span className="settings-cc-title">Kokoro voices</span>
          {modelState === 'unknown' && <span className="settings-cc-badge checking">Checking…</span>}
          {modelState === 'ready' && <span className="settings-cc-badge ok">✓ Installed — {sizeLabel}</span>}
          {modelState === 'missing' && <span className="settings-cc-badge error">Not installed</span>}
          {modelState === 'error' && <span className="settings-cc-badge error">Error</span>}
          {modelState === 'downloading' && <span className="settings-cc-badge checking">Downloading… {pct}%</span>}
          <button
            className="settings-cc-recheck"
            onClick={() => void refreshModelStatus()}
            title="Re-check"
            disabled={modelState === 'downloading'}
          >
            <svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
              <path d="M10.5 2A5 5 0 1 0 11 6.5"/><polyline points="10.5 1 10.5 3.5 8 3.5"/>
            </svg>
          </button>
        </div>
        <p className="settings-cc-desc">
          Reads your manuscript aloud with the Kokoro-82M voice model (Apache 2.0), entirely on
          this machine. The only network use is this one-time {sizeLabel} download from pinned,
          checksum-verified sources; at playback nothing leaves your computer.
        </p>

        {(modelState === 'missing' || modelState === 'error') && (
          <div className="settings-cc-setup">
            <button className="dialog-btn primary" style={{ marginTop: 4 }} onClick={downloadModel}>
              Download voice model ({sizeLabel})
            </button>
            <p className="settings-hint" style={{ marginTop: 8, marginBottom: 0 }}>
              The download can be cancelled and resumes where it left off.
            </p>
          </div>
        )}

        {modelState === 'downloading' && (
          <div className="settings-cc-setup">
            <div className="read-aloud-progress" style={{ marginTop: 4 }}>
              <div className="read-aloud-progress-fill" style={{ width: `${pct}%` }} />
            </div>
            <p className="settings-hint" style={{ marginTop: 6, marginBottom: 8 }}>
              {download
                ? `${formatMB(download.overall_received)} of ${formatMB(download.overall_total)} — ${download.file}`
                : 'Starting download…'}
            </p>
            <button className="dialog-btn" onClick={cancelDownload}>Cancel download</button>
          </div>
        )}

        {modelState === 'ready' && (
          <button className="dialog-btn" style={{ marginTop: 4 }} onClick={() => void removeModel()}>
            Remove downloaded model
          </button>
        )}

        {modelError && <p className="settings-hint read-aloud-error" style={{ marginTop: 8 }}>{modelError}</p>}
      </div>

      <div className="settings-section-label">Voice</div>
      <div className="dialog-field">
        <label className="dialog-label">Voice</label>
        <select className="dialog-select" value={voice} onChange={e => setVoice(e.target.value)}>
          {READ_ALOUD_VOICES.map(v => <option key={v.id} value={v.id}>{v.label}</option>)}
        </select>
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Speed</label>
        <select className="dialog-select" value={String(Math.round(speed * 10) / 10)} onChange={e => setSpeed(Number(e.target.value))}>
          {SPEED_STEPS.map(s => <option key={s} value={String(s)}>{s.toFixed(1)}× {s === 1.2 ? '(default)' : s === 1.0 ? '(natural)' : ''}</option>)}
        </select>
      </div>

      <div className="settings-section-label">Performance</div>
      <div className="dialog-field">
        <label className="dialog-label">Synthesis Device</label>
        <div className="settings-theme-row">
          <button className={`settings-theme-btn${device === 'wasm' ? ' active' : ''}`} onClick={() => setDevice('wasm')}>
            CPU — wasm q8
          </button>
          <button className={`settings-theme-btn${device === 'webgpu' ? ' active' : ''}`} onClick={() => setDevice('webgpu')}>
            GPU — fp32
          </button>
        </div>
        <div className="settings-hint">
          CPU works reliably everywhere. GPU uses the full-precision model (the quantized one
          produces distorted audio on WebGPU) and is faster where it initializes cleanly; if it
          fails on this machine playback falls back to CPU automatically.
        </div>
        {device === 'webgpu' && !gpuInstalled && modelState !== 'downloading' && (
          <button className="dialog-btn primary" style={{ marginTop: 8 }} onClick={downloadGPUModel}>
            Download GPU voice model ({gpuBytesTotal > 0 ? formatMB(gpuBytesTotal) : '~311 MB'})
          </button>
        )}
        {device === 'webgpu' && gpuInstalled && (
          <div className="settings-hint" style={{ marginTop: 6 }}>GPU model installed.</div>
        )}
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Threads</label>
        <div className="settings-theme-row">
          <button className={`settings-theme-btn${threads === 'auto' ? ' active' : ''}`} onClick={() => setThreads('auto')}>
            Auto — recommended
          </button>
          <button className={`settings-theme-btn${threads === 'single' ? ' active' : ''}`} onClick={() => setThreads('single')}>
            Single
          </button>
        </div>
        <div className="settings-hint">
          Auto uses all but one CPU core (needs the cross-origin-isolated runtime — the
          diagnostics show whether it is active). Changes apply the next time playback starts.
          If Read Aloud misbehaves, CPU + Single is the safe fallback.
        </div>
      </div>
      <div className="dialog-field">
        <button className="dialog-btn" onClick={() => void runBenchmark()} disabled={benchmarking || modelState !== 'ready'}>
          {benchmarking ? 'Running performance check…' : 'Run performance check'}
        </button>
        <div className="settings-hint">
          Times a sentence on each installed backend and keeps the faster one. Results appear in
          the diagnostics below.
        </div>
      </div>

      <div className="settings-section-label">Diagnostics</div>
      <div className="dialog-field">
        <div className="settings-path-row">
          <button className="dialog-btn" onClick={() => setShowDiagnostics(v => !v)}>
            {showDiagnostics ? 'Hide diagnostics' : 'Show diagnostics'}
          </button>
          <button
            className="dialog-btn"
            disabled={diagnostics.length === 0}
            onClick={() => {
              void navigator.clipboard.writeText(diagnostics.join('\n')).then(() => {
                setCopied(true)
                setTimeout(() => setCopied(false), 1500)
              })
            }}
          >
            {copied ? 'Copied ✓' : 'Copy'}
          </button>
        </div>
        {showDiagnostics && (
          <div className="settings-cc-log read-aloud-diag-log" style={{ marginTop: 8 }}>
            {diagnostics.length === 0
              ? <div>No diagnostics yet — they are recorded when the voice model loads for playback.</div>
              : diagnostics.map((line, i) => <div key={i}>{line}</div>)}
          </div>
        )}
      </div>
    </>
  )
}
