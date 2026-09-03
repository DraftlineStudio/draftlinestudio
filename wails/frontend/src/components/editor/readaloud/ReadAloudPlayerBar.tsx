// Read Aloud player bar — docked at the bottom of the editor column (the
// bottom counterpart of the chapter find bar), redesigned after the design
// mock in reference-assets/"TTS implementation feedback". All state lives in
// readAloudStore; the bar renders only while the plugin is enabled and the
// player is open. When the voice model is missing it offers setup instead of
// transport controls (mirroring the AI Studio "set up" flow).

import { useEffect, useRef, useState } from 'react'
import { useAppStore } from '../../../store/appStore'
import { useBookStore } from '../../../store/bookStore'
import { useReadAloudStore } from '../../../store/readAloudStore'
import { READ_ALOUD_VOICES } from '../../../services/readaloud/voices'
import { speedLabel } from '../../../services/readaloud/speeds'
import { formatClock } from '../../../services/readaloud/estimates'
import SeekBar from './SeekBar'

function Eq({ playing, color }: { playing: boolean; color?: string }) {
  const style = {
    animationPlayState: playing ? 'running' : 'paused',
    background: color || 'var(--rap-speaker, var(--app-accent-text))',
  } as const
  return (
    <span className="rap-eq" aria-hidden="true">
      <span className="rap-eq-b rap-eq-b1" style={style} />
      <span className="rap-eq-b rap-eq-b2" style={style} />
      <span className="rap-eq-b rap-eq-b3" style={style} />
    </span>
  )
}

function VolumeSlider() {
  const applyVolume = useReadAloudStore(s => s.applyVolume)
  const setVolume = useReadAloudStore(s => s.setVolume)
  const muted = useReadAloudStore(s => s.muted)
  const saved = useAppStore(s => s.settings.read_aloud_volume)
  const [dragging, setDragging] = useState<number | null>(null)
  const trackRef = useRef<HTMLDivElement>(null)

  const shown = dragging ?? saved
  const pct = `${Math.round((muted ? 0 : shown) * 100)}%`

  const fractionAt = (clientX: number): number => {
    const rect = trackRef.current?.getBoundingClientRect()
    if (!rect || rect.width <= 0) return shown
    return Math.max(0, Math.min(1, (clientX - rect.left) / rect.width))
  }

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    e.preventDefault()
    const start = fractionAt(e.clientX)
    setDragging(start)
    applyVolume(start)
    const move = (ev: PointerEvent) => {
      const f = fractionAt(ev.clientX)
      setDragging(f)
      applyVolume(f)
    }
    const up = (ev: PointerEvent) => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
      setDragging(null)
      setVolume(fractionAt(ev.clientX)) // persists
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
  }

  return (
    <div
      ref={trackRef}
      className="rap-volume"
      onPointerDown={handlePointerDown}
      title={`Volume ${Math.round((muted ? 0 : shown) * 100)}%`}
      role="slider"
      aria-label="Volume"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round((muted ? 0 : shown) * 100)}
    >
      <span className="rap-volume-track"><span className="rap-volume-fill" style={{ width: pct }} /></span>
      <span className="rap-volume-thumb" style={{ left: pct }} />
    </div>
  )
}

export default function ReadAloudPlayerBar() {
  const settings = useAppStore(s => s.settings)
  const openSettings = useAppStore(s => s.openSettings)
  const chapterTitle = useBookStore(s => {
    if (!s.book) return ''
    const list = s.currentSection === 'front_matter'
      ? s.book.front_matter
      : s.currentSection === 'back_matter' ? s.book.back_matter : s.book.body
    return list?.[s.currentIndex]?.title ?? ''
  })
  const status = useReadAloudStore(s => s.status)
  const sentences = useReadAloudStore(s => s.sentences)
  const currentIndex = useReadAloudStore(s => s.currentIndex)
  const progress = useReadAloudStore(s => s.progress)
  const playbackError = useReadAloudStore(s => s.playbackError)
  const modelState = useReadAloudStore(s => s.modelState)
  const modelLoading = useReadAloudStore(s => s.modelLoading)
  const refreshModelStatus = useReadAloudStore(s => s.refreshModelStatus)
  const playFromCursor = useReadAloudStore(s => s.playFromCursor)
  const playChapter = useReadAloudStore(s => s.playChapter)
  const togglePause = useReadAloudStore(s => s.togglePause)
  const stop = useReadAloudStore(s => s.stop)
  const skip = useReadAloudStore(s => s.skip)
  const seekToFraction = useReadAloudStore(s => s.seekToFraction)
  const setVoice = useReadAloudStore(s => s.setVoice)
  const cycleSpeed = useReadAloudStore(s => s.cycleSpeed)
  const muted = useReadAloudStore(s => s.muted)
  const toggleMute = useReadAloudStore(s => s.toggleMute)
  const expanded = useReadAloudStore(s => s.expanded)
  const setExpanded = useReadAloudStore(s => s.setExpanded)
  const setPlayerVisible = useReadAloudStore(s => s.setPlayerVisible)
  const currentSpeaker = useReadAloudStore(s => s.currentSpeaker)
  const ticks = useReadAloudStore(s => s.ticks)

  useEffect(() => {
    void refreshModelStatus()
  }, [refreshModelStatus])

  const verified = useReadAloudStore(s => s.verified)
  const prepareFromCursor = useReadAloudStore(s => s.prepareFromCursor)
  // Pre-fill the synthesis buffer while the bar sits open and idle, so the
  // first press of play starts speaking immediately. The store dedupes
  // repeated calls for an unchanged cursor/queue.
  useEffect(() => {
    if (status === 'idle' && modelState === 'ready' && verified) prepareFromCursor()
  }, [status, modelState, verified, prepareFromCursor])

  const active = status !== 'idle'
  const playing = status === 'playing' || status === 'starting'
  const needsSetup = modelState === 'missing' || modelState === 'error' || modelState === 'downloading' || modelState === 'corrupt'
  const glow = settings.read_aloud_glow
  const fraction = active && progress ? progress.fraction : 0
  const sentenceLabel = active && currentIndex >= 0 ? String(currentIndex + 1) : '–'
  const timeLeft = active && progress ? `−${formatClock(progress.remainingSec)}` : ''
  const statusText = playbackError
    ?? (status === 'starting' ? (modelLoading ? 'Loading voices…' : 'Preparing…') : '')

  if (needsSetup) {
    return (
      <div className="rap-dock">
        <div className="rap-bar" role="region" aria-label="Read aloud player">
          <span className="rap-wordmark">READ ALOUD</span>
          <span className="rap-status">
            {modelState === 'downloading'
              ? 'Voice model downloading…'
              : modelState === 'corrupt'
                ? 'Voice model failed verification.'
                : 'Voice model not installed.'}
          </span>
          <button className="rap-setup" onClick={() => openSettings('readaloud')}>
            {modelState === 'downloading' ? 'View progress' : modelState === 'corrupt' ? 'Repair' : 'Set up'}
          </button>
          <button className="rap-icon-btn rap-close" onClick={() => setPlayerVisible(false)} title="Close" aria-label="Close read aloud">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><line x1="6" y1="6" x2="18" y2="18" /><line x1="18" y1="6" x2="6" y2="18" /></svg>
          </button>
        </div>
      </div>
    )
  }

  const speakerName = currentSpeaker?.name ?? 'Narration'
  const speakerColor = currentSpeaker?.color || ''

  return (
    <div className="rap-dock" style={speakerColor ? ({ '--rap-speaker': speakerColor } as React.CSSProperties) : undefined}>
      <div className="rap-bar" role="region" aria-label="Read aloud player">
        <span className="rap-topline" aria-hidden="true">
          <span className="rap-topline-fill" style={{ width: `${(fraction * 100).toFixed(2)}%` }} />
          {glow && active && (
            <span className="rap-topline-glow" style={{ left: `${(fraction * 100).toFixed(2)}%` }} />
          )}
        </span>

        <div className="rap-ident">
          <Eq playing={status === 'playing'} />
          <span className="rap-ident-text">
            <span className="rap-wordmark">READ ALOUD</span>
            <span className="rap-speaker">{speakerName}</span>
          </span>
        </div>
        <span className="rap-divider" />

        <div className="rap-transport">
          <button className="rap-icon-btn" onClick={() => skip(-1)} disabled={!active} title="Previous sentence (Ctrl+Shift+,)" aria-label="Previous sentence">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polygon points="18 5 9.5 12 18 19" /><line x1="6" y1="5" x2="6" y2="19" /></svg>
          </button>
          <button
            className={`rap-play${glow && status === 'playing' ? ' rap-play--glow' : ''}`}
            onClick={active ? togglePause : playFromCursor}
            title={active ? (playing ? 'Pause (Ctrl+Shift+L)' : 'Resume (Ctrl+Shift+L)') : 'Read from cursor (Ctrl+Shift+L)'}
            aria-label={active ? (playing ? 'Pause' : 'Resume') : 'Read from cursor'}
          >
            {playing
              ? <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><line x1="9" y1="5.5" x2="9" y2="18.5" /><line x1="15" y1="5.5" x2="15" y2="18.5" /></svg>
              : <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" style={{ marginLeft: 1 }}><polygon points="7 4.5 19.5 12 7 19.5" /></svg>}
          </button>
          <button className="rap-icon-btn" onClick={() => skip(1)} disabled={!active} title="Next sentence (Ctrl+Shift+.)" aria-label="Next sentence">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polygon points="6 5 14.5 12 6 19" /><line x1="18" y1="5" x2="18" y2="19" /></svg>
          </button>
          <button className="rap-icon-btn" onClick={stop} disabled={!active} title="Stop" aria-label="Stop">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><rect x="8" y="8" width="8" height="8" rx="1" /></svg>
          </button>
          {!active && (
            <button className="rap-text-btn" onClick={playChapter}>Read chapter</button>
          )}
        </div>
        <span className="rap-divider" />

        <div className="rap-center">
          <div className="rap-label-row">
            <span className="rap-chapter">{chapterTitle}</span>
            <span className="rap-position">
              {statusText || (active ? `Sentence ${sentenceLabel} of ${sentences.length}` : sentences.length ? `${sentences.length} sentences queued` : '')}
            </span>
            <span className="rap-spacer" />
            <span className="rap-time">{timeLeft}</span>
          </div>
          <SeekBar
            fraction={fraction}
            ticks={ticks}
            variant="bar"
            onSeek={seekToFraction}
            disabled={!sentences.length}
          />
        </div>
        <span className="rap-compact-label">
          {active ? `${sentenceLabel}/${sentences.length}` : ''}{timeLeft ? ` · ${timeLeft}` : ''}
        </span>
        <span className="rap-divider" />

        <div className="rap-options">
          <select
            className="rap-select"
            value={settings.read_aloud_voice}
            onChange={e => setVoice(e.target.value)}
            title="Voice"
            aria-label="Voice"
          >
            {READ_ALOUD_VOICES.map(v => <option key={v.id} value={v.id}>{v.label}</option>)}
          </select>
          <button className="rap-chip-btn rap-speed" onClick={cycleSpeed} title="Playback speed" aria-label="Playback speed">
            {speedLabel(settings.read_aloud_speed)}
          </button>
        </div>
        <span className="rap-divider" />

        <div className="rap-audio">
          <button className="rap-icon-btn" onClick={toggleMute} title={muted ? 'Unmute' : 'Mute'} aria-label={muted ? 'Unmute' : 'Mute'}>
            {muted
              ? <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M11 5 6 9H3v6h3l5 4z" /><line x1="16" y1="9" x2="22" y2="15" /><line x1="22" y1="9" x2="16" y2="15" /></svg>
              : <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M11 5 6 9H3v6h3l5 4z" /><path d="M15.5 9a4.5 4.5 0 0 1 0 6" /><path d="M18.5 6.5a8.5 8.5 0 0 1 0 11" /></svg>}
          </button>
          <VolumeSlider />
          <button className="rap-icon-btn" onClick={() => setExpanded(!expanded)} title={expanded ? 'Collapse' : 'Expand'} aria-label={expanded ? 'Collapse panel' : 'Expand panel'}>
            <span className="rap-chevron" style={{ transform: `rotate(${expanded ? 180 : 0}deg)` }}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 15 12 9 18 15" /></svg>
            </span>
          </button>
          <button className="rap-icon-btn rap-close" onClick={() => setPlayerVisible(false)} title="Close (stops playback)" aria-label="Close read aloud">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><line x1="6" y1="6" x2="18" y2="18" /><line x1="18" y1="6" x2="6" y2="18" /></svg>
          </button>
        </div>
      </div>
    </div>
  )
}
