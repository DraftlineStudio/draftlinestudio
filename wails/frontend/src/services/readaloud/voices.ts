// Read Aloud voice catalog. Must stay in sync with the Go-side manifest
// (wails/internal/readaloud/native.go): every id here maps to a speaker in the
// pinned voices.bin table.

export interface ReadAloudVoice {
  id: string
  label: string
}

export const READ_ALOUD_VOICES: ReadAloudVoice[] = [
  { id: 'af_heart', label: 'Heart — American female' },
  { id: 'af_bella', label: 'Bella — American female' },
  { id: 'af_nicole', label: 'Nicole — American female (soft)' },
  { id: 'am_michael', label: 'Michael — American male' },
  { id: 'am_fenrir', label: 'Fenrir — American male (deep)' },
  { id: 'am_puck', label: 'Puck — American male (bright)' },
  { id: 'bf_emma', label: 'Emma — British female' },
  { id: 'bm_george', label: 'George — British male' },
]

export const DEFAULT_READ_ALOUD_VOICE = 'af_heart'

export function isKnownVoice(id: string): boolean {
  return READ_ALOUD_VOICES.some(v => v.id === id)
}
