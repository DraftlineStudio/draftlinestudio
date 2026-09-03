// Voice Cast section of the expanded Read Aloud panel: cast-mode toggle and
// the narrator voice row. Character rows, previews, and auto-cast land with
// the full cast UI.

import { useAppStore } from '../../../store/appStore'
import { useReadAloudStore } from '../../../store/readAloudStore'
import { READ_ALOUD_VOICES } from '../../../services/readaloud/voices'

export default function VoiceCastSection() {
  const narratorVoice = useAppStore(s => s.settings.read_aloud_voice)
  const castMode = useReadAloudStore(s => s.castMode)
  const setCastMode = useReadAloudStore(s => s.setCastMode)
  const setSpeakerVoice = useReadAloudStore(s => s.setSpeakerVoice)
  const speakers = useReadAloudStore(s => s.speakers)
  const dialogueLineCount = useReadAloudStore(s => s.dialogueLineCount)

  const detected = speakers.filter(s => s.key !== 'narrator' && s.lineCount > 0).length
  const hint = dialogueLineCount === 0
    ? 'No dialogue detected in this chapter'
    : castMode ? `${detected + 1} speakers detected` : 'Narrator reads everything'

  return (
    <div className="rap-cast-col">
      <div className="rap-section-head">
        <span className="rap-section-label">Voice Cast</span>
        <span className="rap-cast-hint">{hint}</span>
        <label className="settings-toggle" title={castMode ? 'Cast mode on — dialogue in character voices' : 'Cast mode off — single narrator'}>
          <input type="checkbox" checked={castMode} onChange={e => setCastMode(e.target.checked)} />
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
      </div>
    </div>
  )
}
