// Voice-cast model for Read Aloud: builds the chapter's speaker roster from
// the book's character data, resolves saved voice assignments, and provides
// deterministic auto-casting. Persistence lives on BookData.read_aloud_cast
// (an optional .draftline archive member); keys are lowercased canonical
// character names so they survive re-indexing.

import type { BookData, Character, ReadAloudCast } from '../../types/draftline'
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

// TTS attribution is recall-first, unlike the precision-first analysis
// sidebars: a name the roster can't see doesn't just miss its own lines — it
// hands them to whoever the fallback rules guess ("Alvarez said" attributed
// to Hanlon because Alvarez was still in review). So review-status
// auto-detections stay in; only explicit rejections are excluded.
function speakableCharacter(character: Character): boolean {
  if (character.is_auto_detected && character.detection_status === 'rejected') return false
  const kind = character.entity_kind
  return kind === undefined || kind === 'person' || kind === 'unknown'
}

// Honorifics and connectives never identify a character on their own.
const NON_IDENTIFYING_PARTS = new Set([
  'mr', 'mrs', 'ms', 'miss', 'dr', 'sir', 'lady', 'lord', 'madam', 'dame',
  'detective', 'officer', 'sergeant', 'captain', 'lieutenant', 'colonel',
  'general', 'major', 'agent', 'professor', 'doctor', 'judge', 'father',
  'mother', 'sister', 'brother', 'aunt', 'uncle', 'the', 'old', 'young',
  'saint', 'st', 'van', 'von', 'del', 'de', 'la', 'le',
])

// Individual capitalized words of a name ("Renee Alvarez" → Renee, Alvarez)
// — dialogue tags usually use just the surname or given name, and entity
// resolution doesn't always record each part as an alias.
function nameParts(alias: string): string[] {
  return alias
    .split(/[\s-]+/)
    .map(part => part.replace(/[^\p{L}'’]/gu, ''))
    .filter(part => part.length >= 3 && /^\p{Lu}/u.test(part) && !NON_IDENTIFYING_PARTS.has(part.toLowerCase()))
}

// The characters that can speak in this chapter: people mentioned here per
// the analysis index, plus any whose name literally appears in the chapter
// text (covers stale analysis and manual characters). Aliases are expanded
// with their individual name parts; a part claimed by more than one
// character ("Webb" of Marcus Webb and Sarah Webb) identifies nobody and is
// dropped from everyone — except a character's own full name, which always
// stands.
export function buildRoster(book: BookData, globalChapterIdx: number, chapterText: string): RosterEntry[] {
  const characters = (book.story_bible?.characters ?? []).filter(c => c.name?.trim() && speakableCharacter(c))

  // Expand every character's aliases with derived name parts.
  const expanded = characters.map(character => {
    const name = character.name.trim()
    const explicit = [name, ...(character.aliases ?? [])].map(a => a.trim()).filter(a => a.length > 1)
    const derived = explicit.flatMap(nameParts)
    return { character, name, aliases: [...new Set([...explicit, ...derived])] }
  })

  // First determine who can actually belong to this chapter. Globally unique
  // aliases may pull in a character when analysis is stale, but an ambiguous
  // surname must not pull every same-named character from the whole novel into
  // this chapter and then erase the useful alias from the real participant.
  const globalClaims = new Map<string, number>()
  for (const entry of expanded) {
    for (const alias of new Set(entry.aliases.map(a => a.toLowerCase()))) {
      globalClaims.set(alias, (globalClaims.get(alias) ?? 0) + 1)
    }
  }
  const candidates = expanded.filter(entry => {
    const mentionedHere = (entry.character.chapter_mentions?.[globalChapterIdx] ?? 0) > 0
    if (mentionedHere || chapterText.includes(entry.name)) return true
    return entry.aliases.some(alias =>
      (globalClaims.get(alias.toLowerCase()) ?? 0) === 1 && chapterText.includes(alias))
  })

  // An alias is ambiguous only among characters who can occur in this
  // chapter. The same surname used by unrelated characters elsewhere in the
  // novel should not disable ordinary surname dialogue tags here.
  const claims = new Map<string, number>()
  for (const entry of candidates) {
    for (const alias of new Set(entry.aliases.map(a => a.toLowerCase()))) {
      claims.set(alias, (claims.get(alias) ?? 0) + 1)
    }
  }

  const roster: RosterEntry[] = []
  const seen = new Set<string>()
  for (const entry of candidates) {
    const key = speakerKeyFor(entry.name)
    if (seen.has(key)) continue
    const aliases = entry.aliases.filter(alias =>
      alias === entry.name || (claims.get(alias.toLowerCase()) ?? 0) <= 1)
    seen.add(key)
    roster.push({ key, name: entry.name, aliases })
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
