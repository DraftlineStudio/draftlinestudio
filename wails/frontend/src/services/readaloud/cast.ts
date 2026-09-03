// Voice-cast model for Read Aloud: builds the chapter's speaker roster from
// the book's character data, resolves saved voice assignments, and provides
// deterministic auto-casting. Persistence lives on BookData.read_aloud_cast
// (an optional .draftline archive member); keys are lowercased canonical
// character names so they survive re-indexing.

import type { BookData, Character, ReadAloudCast } from '../../types/draftline'
import { isConfirmedCharacter } from '../../utils/characterStatus'
import { characterColor } from '../../utils/characterVisuals'
import { READ_ALOUD_VOICES } from './voices'
import { speakerKeyFor, type AttributionResult, type GenderEvidence, type RosterEntry, type SpeakerKey } from './attribution'

export interface ChapterSpeaker {
  key: SpeakerKey
  name: string // 'Narration' | canonical character name | 'Unknown speaker'
  // Character rows carry their app-wide characterColor; narrator/unknown use
  // '' and the UI falls back to accent / muted styling.
  color: string
  lineCount: number // dialogue+mixed sentences this chapter (narration count for the narrator)
  voice: string | null // null = uncast (falls back to the narrator voice)
  characterId?: string
}

// The persisted map key for a character speaker ('char:renee alvarez' →
// 'renee alvarez').
export function castVoiceKey(key: SpeakerKey): string | null {
  return key.startsWith('char:') ? key.slice(5) : null
}

function speakableCharacter(character: Character): boolean {
  if (!isConfirmedCharacter(character)) return false
  const kind = character.entity_kind
  return kind === undefined || kind === 'person' || kind === 'unknown'
}

// The characters that can speak in this chapter: confirmed people mentioned
// here per the analysis index, plus any whose name literally appears in the
// chapter text (covers stale analysis and manual characters).
export function buildRoster(book: BookData, globalChapterIdx: number, chapterText: string): RosterEntry[] {
  const characters = book.story_bible?.characters ?? []
  const roster: RosterEntry[] = []
  const seen = new Set<string>()
  for (const character of characters) {
    const name = character.name?.trim()
    if (!name || !speakableCharacter(character)) continue
    const key = speakerKeyFor(name)
    if (seen.has(key)) continue
    const mentionedHere = (character.chapter_mentions?.[globalChapterIdx] ?? 0) > 0
    const aliases = [name, ...(character.aliases ?? [])].filter(a => a.trim().length > 1)
    const appearsInText = !mentionedHere && aliases.some(alias => chapterText.includes(alias))
    if (!mentionedHere && !appearsInText) continue
    seen.add(key)
    roster.push({ key, name, aliases })
  }
  return roster
}

// Resolves the chapter's speaker rows: Narration first, then detected
// character speakers by line count, then the unknown bucket when it has
// lines. Saved assignments come from the book's cast; unsaved rows are null.
export function buildChapterCast(
  attribution: AttributionResult,
  roster: RosterEntry[],
  savedCast: ReadAloudCast | null | undefined,
  narratorVoice: string,
): ChapterSpeaker[] {
  const voices = savedCast?.voices ?? {}
  const knownVoice = (id: string | undefined): string | null =>
    id && READ_ALOUD_VOICES.some(v => v.id === id) ? id : null

  const speakers: ChapterSpeaker[] = [{
    key: 'narrator',
    name: 'Narration',
    color: '',
    lineCount: attribution.lineCounts.get('narrator') ?? 0,
    voice: narratorVoice,
  }]

  const chars = roster
    .map(entry => ({
      key: entry.key,
      name: entry.name,
      color: characterColor(entry.name),
      lineCount: attribution.lineCounts.get(entry.key) ?? 0,
      voice: knownVoice(voices[castVoiceKey(entry.key) ?? '']),
    }))
    .filter(speaker => speaker.lineCount > 0)
    .sort((a, b) => b.lineCount - a.lineCount || a.name.localeCompare(b.name))
  speakers.push(...chars)

  const unknownLines = attribution.lineCounts.get('unknown') ?? 0
  if (unknownLines > 0) {
    speakers.push({ key: 'unknown', name: 'Unknown speaker', color: '', lineCount: unknownLines, voice: null })
  }
  return speakers
}

// Voice pools by gender, derived from the id prefix convention
// (af_/bf_ female, am_/bm_ male).
const FEMALE_POOL = READ_ALOUD_VOICES.filter(v => v.id.startsWith('af_') || v.id.startsWith('bf_')).map(v => v.id)
const MALE_POOL = READ_ALOUD_VOICES.filter(v => v.id.startsWith('am_') || v.id.startsWith('bm_')).map(v => v.id)
// No/ambiguous evidence: alternate female/male so neighbors stay distinct.
const COMBINED_POOL = FEMALE_POOL.flatMap((id, i) => (MALE_POOL[i] ? [id, MALE_POOL[i]] : [id]))
  .concat(MALE_POOL.slice(FEMALE_POOL.length))

function poolFor(evidence: GenderEvidence | undefined): string[] {
  if (!evidence) return COMBINED_POOL
  const { he, she, they } = evidence
  if (she > he && she >= they) return FEMALE_POOL
  if (he > she && he >= they) return MALE_POOL
  return COMBINED_POOL
}

// Deterministic auto-cast: rank uncast character speakers by line count
// (name breaks ties), pick from the gender-matched pool, skip voices already
// in use (narrator's and manual assignments) until the pool forces repeats.
// Never overwrites an existing assignment. Returns ONLY the new assignments,
// keyed for ReadAloudCast.voices.
export function autoCast(
  speakers: ChapterSpeaker[],
  genderEvidence: Map<SpeakerKey, GenderEvidence>,
  narratorVoice: string,
): Record<string, string> {
  const used = new Set<string>([narratorVoice])
  for (const speaker of speakers) {
    if (speaker.key !== 'narrator' && speaker.voice) used.add(speaker.voice)
  }

  const assignments: Record<string, string> = {}
  const ranked = speakers.filter(s => s.key.startsWith('char:') && !s.voice)
  for (const speaker of ranked) {
    const pool = poolFor(genderEvidence.get(speaker.key))
    const pick = pool.find(id => !used.has(id))
      // Pool exhausted: reuse, preferring voices other than the narrator's.
      ?? pool.find(id => id !== narratorVoice)
      ?? pool[0]
    if (!pick) continue
    used.add(pick)
    const key = castVoiceKey(speaker.key)
    if (key) assignments[key] = pick
  }
  return assignments
}
