// Expanded Read Aloud panel: a popover anchored above the player bar with
// the now-reading strip, the Voice Cast section, and chapter progress stats.
// Collapses on outside click; the bar's chevron toggles it.

import { useEffect, useRef } from 'react'
import { useReadAloudStore } from '../../../store/readAloudStore'
import { formatClock } from '../../../services/readaloud/estimates'
import SeekBar from './SeekBar'
import VoiceCastSection from './VoiceCastSection'

export default function ReadAloudExpandedPanel({ chapterTitle }: { chapterTitle: string }) {
  const sentences = useReadAloudStore(s => s.sentences)
  const currentIndex = useReadAloudStore(s => s.currentIndex)
  const progress = useReadAloudStore(s => s.progress)
  const status = useReadAloudStore(s => s.status)
  const castMode = useReadAloudStore(s => s.castMode)
  const speakers = useReadAloudStore(s => s.speakers)
  const ticks = useReadAloudStore(s => s.ticks)
  const currentSpeaker = useReadAloudStore(s => s.currentSpeaker)
  const dialogueLineCount = useReadAloudStore(s => s.dialogueLineCount)
  const seekToFraction = useReadAloudStore(s => s.seekToFraction)
  const setExpanded = useReadAloudStore(s => s.setExpanded)

  const panelRef = useRef<HTMLDivElement>(null)

  // Outside-click collapse: anywhere beyond the player dock (panel + bar).
  useEffect(() => {
    const onPointerDown = (e: PointerEvent) => {
      const dock = panelRef.current?.closest('.rap-dock')
      if (dock && e.target instanceof Node && !dock.contains(e.target)) setExpanded(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [setExpanded])

  const active = status !== 'idle'
  const speakerName = currentSpeaker?.name ?? 'Narration'
  const speakerColor = currentSpeaker?.color || 'var(--app-accent-text)'
  const currentText = (active && currentIndex >= 0 ? sentences[currentIndex]?.text : '')
    || (sentences.length ? 'Press play to start reading.' : 'Place the cursor in the chapter and press play.')
  const sentenceLabel = active && currentIndex >= 0 ? String(currentIndex + 1) : '–'
  const fraction = active && progress ? progress.fraction : 0
  const legend = castMode ? speakers.filter(s => s.lineCount > 0) : []

  return (
    <div className="rap-panel" ref={panelRef} role="dialog" aria-label="Read aloud details">
      <div className="rap-panel-header">
        <span className="rap-panel-title">Read Aloud</span>
        {chapterTitle && <span className="rap-panel-chapter">· {chapterTitle}</span>}
        <span className="rap-spacer" />
        <span className="rap-badge">Kokoro 82M · 24 kHz</span>
        <button className="rap-icon-btn" onClick={() => setExpanded(false)} title="Collapse" aria-label="Collapse panel">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9" /></svg>
        </button>
      </div>

      <div className="rap-nowreading">
        <span className="rap-nr-label" style={{ color: speakerColor }}>Now reading · {speakerName}</span>
        <span className="rap-nr-text" style={{ borderLeftColor: speakerColor }}>{currentText}</span>
      </div>

      <div className="rap-panel-body">
        <VoiceCastSection />

        <div className="rap-progress-col">
          <span className="rap-section-label">Chapter Progress</span>
          <SeekBar fraction={fraction} ticks={ticks} variant="panel" onSeek={seekToFraction} disabled={!sentences.length} />
          <div className="rap-progress-counts">
            <span>Sentence {sentenceLabel}</span>
            <span>{sentences.length} total</span>
          </div>
          {legend.length > 0 && (
            <div className="rap-legend">
              {legend.map(speaker => (
                <span key={speaker.key} className="rap-legend-item">
                  <span
                    className="rap-legend-dot"
                    style={{ background: speaker.color || (speaker.key === 'narrator' ? 'var(--app-accent-text)' : 'var(--text-muted)') }}
                  />
                  {speaker.key === 'narrator' ? 'Narration' : speaker.name}
                </span>
              ))}
            </div>
          )}
          <div className="rap-stat"><span>Time left</span><span>{active && progress ? formatClock(progress.remainingSec) : '–'}</span></div>
          <div className="rap-stat"><span>Elapsed</span><span>{active && progress ? formatClock(progress.elapsedSec) : '–'}</span></div>
          <div className="rap-stat"><span>Dialogue lines</span><span>{sentences.length ? `${dialogueLineCount} of ${sentences.length}` : '–'}</span></div>
          <div className="rap-hints">
            <span className="rap-kbd">Ctrl+Shift+L</span><span>Play / pause</span>
            <span className="rap-kbd">Ctrl+Shift+,</span><span className="rap-kbd">.</span><span>Sentence</span>
          </div>
        </div>
      </div>
    </div>
  )
}
