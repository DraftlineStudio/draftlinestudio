// Voice Cast section of the expanded Read Aloud panel: the cast-mode toggle,
// the narrator voice row, one row per detected chapter speaker with voice
// assignment and preview, the unknown-speaker bucket, and deterministic
// auto-casting from the book's Characters data.

import { useAppStore } from '../../../store/appStore'
import { useBookStore } from '../../../store/bookStore'
import { useReadAloudStore } from '../../../store/readAloudStore'
import { READ_ALOUD_VOICES } from '../../../services/readaloud/voices'
import { characterInitials } from '../../../utils/characterVisuals'
import type { ChapterSpeaker } from '../../../services/readaloud/cast'

function PreviewEq() {
  return (
    <span className="rap-preview-eq" aria-hidden="true">
      <span className="rap-eq-b rap-eq-b1" />
      <span className="rap-eq-b rap-eq-b2" />
      <span className="rap-eq-b rap-eq-b3" />
    </span>
  )
}

function PreviewButton({ voiceId }: { voiceId: string }) {
  const previewingVoice = useReadAloudStore(s => s.previewingVoice)
  const previewVoice = useReadAloudStore(s => s.previewVoice)
  const status = useReadAloudStore(s => s.status)
  const busy = status === 'playing' || status === 'starting'

  if (previewingVoice === voiceId) {
    return (
      <button className="rap-icon-btn rap-preview-btn" onClick={() => previewVoice(voiceId)} title="Stop preview" aria-label="Stop preview">
        <PreviewEq />
      </button>
    )
  }
  return (
    <button
      className="rap-icon-btn rap-preview-btn"
      onClick={() => previewVoice(voiceId)}
      disabled={busy}
      title={busy ? 'Preview is available while paused or stopped' : 'Preview voice'}
      aria-label="Preview voice"
    >
      <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="8 5 19 12 8 19" /></svg>
    </button>
  )
}

function CharacterRow({ speaker, dimmed }: { speaker: ChapterSpeaker; dimmed: boolean }) {
  const setSpeakerVoice = useReadAloudStore(s => s.setSpeakerVoice)
  const autoCastVoices = useReadAloudStore(s => s.autoCastVoices)

  return (
    <div className={`rap-cast-row${dimmed ? ' rap-cast-row--dim' : ''}`}>
      {speaker.voice ? (
        <span className="rap-cast-swatch" style={{ background: speaker.color }}>{characterInitials(speaker.name)}</span>
      ) : (
        <span className="rap-cast-swatch rap-cast-swatch--unknown">?</span>
      )}
      <span className="rap-cast-who">
        <span className="rap-cast-name">{speaker.name}</span>
        <span className="rap-cast-sub">
          {speaker.lineCount} {speaker.lineCount === 1 ? 'line' : 'lines'} in this chapter{speaker.voice ? '' : ' · not cast yet'}
        </span>
      </span>
      {speaker.voice ? (
        <>
          <select
            className="rap-select rap-cast-select"
            value={speaker.voice}
            onChange={e => setSpeakerVoice(speaker.key, e.target.value)}
            title={`${speaker.name} voice`}
            aria-label={`${speaker.name} voice`}
          >
            {READ_ALOUD_VOICES.map(v => <option key={v.id} value={v.id}>{v.label}</option>)}
          </select>
          <PreviewButton voiceId={speaker.voice} />
        </>
      ) : (
        // One click gives this speaker exactly what auto-cast would pick;
        // the select appears once a voice exists.
        <button className="rap-cast-assign" onClick={autoCastVoices}>Cast voice</button>
      )}
    </div>
  )
}

export default function VoiceCastSection() {
  const narratorVoice = useAppStore(s => s.settings.read_aloud_voice)
  const hasBook = useBookStore(s => !!s.book)
  const castMode = useReadAloudStore(s => s.castMode)
  const setCastMode = useReadAloudStore(s => s.setCastMode)
  const setSpeakerVoice = useReadAloudStore(s => s.setSpeakerVoice)
  const autoCastVoices = useReadAloudStore(s => s.autoCastVoices)
  const speakers = useReadAloudStore(s => s.speakers)
  const dialogueLineCount = useReadAloudStore(s => s.dialogueLineCount)

  const characterRows = speakers.filter(s => s.key !== 'narrator' && s.key !== 'unknown')
  const unknownRow = speakers.find(s => s.key === 'unknown')
  const uncast = characterRows.filter(s => !s.voice).length
  const rosterEmpty = characterRows.length === 0 && dialogueLineCount > 0

  const hint = dialogueLineCount === 0
    ? 'No dialogue detected in this chapter'
    : castMode ? `${characterRows.length + 1 + (unknownRow ? 1 : 0)} speakers detected` : 'Narrator reads everything'

  return (
    <div className="rap-cast-col">
      <div className="rap-section-head">
        <span className="rap-section-label">Voice Cast</span>
        <span className="rap-cast-hint">{hint}</span>
        <label className="settings-toggle" title={castMode ? 'Cast mode on — dialogue in character voices' : 'Cast mode off — single narrator'}>
          <input type="checkbox" checked={castMode} onChange={e => setCastMode(e.target.checked)} disabled={!hasBook} />
          <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
          <span className="settings-toggle-label">Cast</span>
        </label>
      </div>

      <div className="rap-cast-row">
        <span className="rap-cast-swatch rap-cast-swatch--narrator">N</span>
        <span className="rap-cast-who">
          <span className="rap-cast-name">Narration</span>
          <span className="rap-cast-sub">Everything outside quotes</span>
        </span>
        <select
          className="rap-select rap-cast-select"
          value={narratorVoice}
          onChange={e => setSpeakerVoice('narrator', e.target.value)}
          title="Narrator voice"
          aria-label="Narrator voice"
        >
          {READ_ALOUD_VOICES.map(v => <option key={v.id} value={v.id}>{v.label}</option>)}
        </select>
        <PreviewButton voiceId={narratorVoice} />
      </div>

      {characterRows.map(speaker => (
        <CharacterRow key={speaker.key} speaker={speaker} dimmed={!castMode} />
      ))}

      {unknownRow && (
        <div className={`rap-cast-row${castMode ? '' : ' rap-cast-row--dim'}`}>
          <span className="rap-cast-swatch rap-cast-swatch--unknown">?</span>
          <span className="rap-cast-who">
            <span className="rap-cast-name">Unknown speaker</span>
            <span className="rap-cast-sub">{unknownRow.lineCount} {unknownRow.lineCount === 1 ? 'line' : 'lines'} · reads in narrator voice</span>
          </span>
        </div>
      )}

      {rosterEmpty ? (
        <span className="rap-cast-note">Run character analysis to cast dialogue voices.</span>
      ) : characterRows.length > 0 && (
        <button
          className="rap-autocast"
          onClick={autoCastVoices}
          disabled={uncast === 0 || !castMode}
          title={uncast === 0 ? 'Every detected speaker already has a voice' : 'Assign voices to uncast speakers'}
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 12a9 9 0 1 1-2.64-6.36" /><polyline points="21 3 21 9 15 9" /></svg>
          Auto-cast from Characters
        </button>
      )}
    </div>
  )
}
