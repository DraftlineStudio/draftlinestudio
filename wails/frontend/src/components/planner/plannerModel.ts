// Planner model: pure functions over the Planner's persisted data
// (planner.json in the .draftline archive) and the manuscript. No React, no
// store access, so every rule here is unit-testable.
//
// The Planner is a story-line timeline: lanes (main plot, subplots, character
// arcs) crossed with chapters, with plot cards where a line meets a chapter.
// Cards are filled in by hand or proposed from an outline by the
// deterministic rules in parseOutline; nothing is committed until accepted.

import type {
  BeatTemplateId, BookData, Character, PlannerCard, PlannerData, PlannerLane, PlannerNote,
} from '../../types/draftline'

export const MAIN_LANE_ID = 'main'
export const MAIN_LANE_COLOR = '#5aafe0'
// The Later column: a card that belongs to no chapter of the book's body.
export const LATER_COLUMN_ID = ''
export const LANE_PALETTE = ['#98C379', '#61AFEF', '#FABF73', '#2DD4BF', '#F87171', '#A3E635', '#818CF8', '#FB923C']
export const DEAD_NOTE_ID = 'dead-ideas'

export type CardStatus = 'planned' | 'drafted'
export const STATUS_COLORS: Record<CardStatus, string> = {
  planned: '#868C96', drafted: '#5aafe0',
}

export interface Beat { name: string; pct: number }
export const BEATS: Record<Exclude<BeatTemplateId, 'none'>, Beat[]> = {
  'three-act': [
    { name: 'Inciting incident', pct: 0.12 }, { name: 'First plot point', pct: 0.25 }, { name: 'Midpoint', pct: 0.5 },
    { name: 'Second plot point', pct: 0.75 }, { name: 'Climax', pct: 0.9 },
  ],
  'save-the-cat': [
    { name: 'Opening image', pct: 0.01 }, { name: 'Catalyst', pct: 0.1 }, { name: 'Break into two', pct: 0.2 }, { name: 'Midpoint', pct: 0.5 },
    { name: 'All is lost', pct: 0.75 }, { name: 'Break into three', pct: 0.8 }, { name: 'Final image', pct: 0.99 },
  ],
}
export const BEAT_NAMES: Record<BeatTemplateId, string> = { none: 'None', 'three-act': 'Three-Act', 'save-the-cat': 'Save the Cat' }
// Every beat name the importer recognises, including the Save the Cat beats
// that are not drawn as marks.
export const BEATS_ALL: Beat[] = [
  ...BEATS['three-act'], ...BEATS['save-the-cat'],
  { name: 'Theme stated', pct: 0.05 }, { name: 'Debate', pct: 0.15 }, { name: 'B story', pct: 0.22 }, { name: 'Fun and games', pct: 0.3 },
  { name: 'Bad guys close in', pct: 0.6 }, { name: 'Dark night of the soul', pct: 0.78 }, { name: 'Finale', pct: 0.9 },
]

export function newId(prefix: string): string {
  const rand = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID().replace(/-/g, '').slice(0, 12)
    : Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
  return `${prefix}-${rand}`
}

export function nowStamp(): string {
  return new Date().toISOString()
}

// "Today at 3:41 PM", "Yesterday", "5 days ago" — the app's relative-date voice.
export function relativeStamp(iso?: string): string {
  if (!iso) return '—'
  const then = new Date(iso)
  if (Number.isNaN(then.getTime())) return iso
  const now = new Date()
  const days = Math.floor((new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime() - new Date(then.getFullYear(), then.getMonth(), then.getDate()).getTime()) / 86_400_000)
  const time = then.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
  if (days <= 0) return `Today at ${time}`
  if (days === 1) return 'Yesterday'
  if (days < 7) return `${days} days ago`
  return then.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function countWords(text: string): number {
  return (text.trim().match(/\S+/g) ?? []).length
}

function stripHtml(html: string): string {
  return html.replace(/<[^>]+>/g, ' ').replace(/&nbsp;/g, ' ').replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>')
}

// ── Persisted data ──────────────────────────────────────────────────────────

export function deadNote(): PlannerNote {
  return {
    id: DEAD_NOTE_ID, system: 'dead', excluded: true, title: 'Dead ideas',
    body: 'Cards thrown away from the timeline and board land here. Nothing is lost.\n',
  }
}

export function emptyPlanner(): PlannerData {
  return {
    version: 1,
    lanes: [{ id: MAIN_LANE_ID, name: 'Main plot', kind: 'main', color: MAIN_LANE_COLOR }],
    cards: [],
    notes: [deadNote()],
    synopsis: {},
    beat_template: 'none',
    hidden_lanes: [],
    compact: true,
  }
}

// The book's Planner, with the invariants every view relies on: a main lane
// first and the Dead Ideas note present. Does not mutate the book.
export function ensurePlanner(book: BookData | null): PlannerData {
  const base = book?.planner ?? emptyPlanner()
  const baseLanes = base.lanes ?? []
  const baseNotes = base.notes ?? []
  const lanes = baseLanes.some(l => l.id === MAIN_LANE_ID)
    ? baseLanes
    : [{ id: MAIN_LANE_ID, name: 'Main plot', kind: 'main' as const, color: MAIN_LANE_COLOR }, ...baseLanes]
  const notes = baseNotes.some(n => n.system === 'dead') ? baseNotes : [...baseNotes, deadNote()]
  return { ...base, lanes, cards: base.cards ?? [], notes, synopsis: base.synopsis ?? {}, hidden_lanes: base.hidden_lanes ?? [] }
}

// ── The manuscript, as the Planner sees it ─────────────────────────────────

export interface PlannerChapter {
  id: string
  num: number
  title: string
  words: number
  scenes: number
  drafted: boolean
}

// Body chapters in manuscript order. Scene breaks are counted here with the
// same rule the backend applies, so a chapter with words always has at least
// one scene to link a card to.
export function bookChapters(book: BookData | null): PlannerChapter[] {
  if (!book) return []
  return book.body.map((ch, i) => {
    const words = countWords(stripHtml(ch.content || ''))
    const id = ch.id ?? `body/${i}`
    const scenes = words > 0 ? sceneBreaks(ch.content || '') + 1 : 0
    return { id, num: i + 1, title: ch.title || '', words, scenes: words > 0 ? scenes : 0, drafted: words > 0 }
  })
}

// Scene breaks in a chapter's HTML: a horizontal rule, or a paragraph whose
// whole text is a centred break marker. The marker list is the one the
// backend applies (internal/indexing/scenes.go sceneBreakMarker), form for
// form, so the count before the first analysis is the count after it.
const BREAK_MARKERS = /^(?:\*\s*\*\s*\*|\*{3,}|⁂|#\s*#\s*#|#{3,}|-\s*-\s*-|-{3,}|~\s*~\s*~|\.\s*\.\s*\.)$/
function sceneBreaks(html: string): number {
  const rules = (html.match(/<hr\b[^>]*>/gi) ?? []).length
  const paragraphs = (html.match(/<p\b[^>]*>[\s\S]*?<\/p>/gi) ?? [])
    .filter(p => BREAK_MARKERS.test(stripHtml(p).replace(/\s+/g, ' ').trim()))
    .length
  return rules + paragraphs
}

export interface CodexPerson {
  id: string
  name: string
  aliases: string[]
}

// Accepted people from the codex: the names Import Outline recognises and the
// candidates for character lanes and a card's "who".
export function codexPeople(book: BookData | null): CodexPerson[] {
  const chars: Character[] = book?.story_bible?.characters ?? []
  return chars
    .filter(c => c.id && c.name && (!c.entity_kind || c.entity_kind === 'person') && c.detection_status !== 'rejected'
      && !(c.is_auto_detected && c.detection_status !== 'accepted'))
    .map(c => ({ id: c.id, name: c.name, aliases: [c.name, ...(c.aliases ?? [])].filter(Boolean) }))
}

export function firstName(name: string): string {
  return name.split(' ')[0]
}

// ── Cards on screen ────────────────────────────────────────────────────────

// A card as the views draw it: the persisted card plus its derived status.
export interface DisplayCard extends PlannerCard {
  st: CardStatus
}

// The status the Planner shows for a card: drafted once it is linked to a
// scene, planned until then.
export function statusOf(card: PlannerCard): CardStatus {
  return card.link ? 'drafted' : 'planned'
}

// Every card the canvas draws, with its derived status.
export function displayCards(planner: PlannerData): DisplayCard[] {
  return planner.cards.map(c => ({ ...c, st: statusOf(c) }))
}


export function laneChips(card: PlannerCard, lanes: PlannerLane[]): PlannerLane[] {
  return card.lines.map(id => lanes.find(l => l.id === id)).filter((l): l is PlannerLane => !!l)
}

const sameName = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase()

// The codex person a card's `who` entry means now. Codex IDs are positional
// and reassigned by re-indexing, so the name stored beside the ID is the
// durable key: the person under the ID counts only while that name is still
// theirs (their name or an alias); otherwise the person now carrying the
// stored name, if any. With no stored name the ID alone decides.
export function whoPerson(id: string, stored: string | undefined, codex: CodexPerson[]): CodexPerson | undefined {
  const byId = codex.find(c => c.id === id)
  if (!stored) return byId
  if (byId && byId.aliases.some(a => sameName(a, stored))) return byId
  return codex.find(c => c.aliases.some(a => sameName(a, stored)))
}

// The name previously stored beside an ID on a card, if any.
const storedName = (id: string, previous?: Pick<PlannerCard, 'who' | 'who_names'>): string | undefined => {
  const at = previous?.who.indexOf(id) ?? -1
  return at >= 0 ? previous?.who_names?.[at] : undefined
}

// The names to store beside a card's `who`: the current codex name of the
// person each ID means (see whoPerson), else the name the card already
// recorded for that ID, else the ID itself.
export function whoNames(who: string[], codex: CodexPerson[], previous?: Pick<PlannerCard, 'who' | 'who_names'>): string[] {
  return who.map(id => {
    const stored = storedName(id, previous)
    return whoPerson(id, stored, codex)?.name ?? stored ?? id
  })
}

// A card's `who` and `who_names` as they should be written now: every ID
// re-bound to the codex person its stored name means (an ID reassigned to
// someone else is replaced by the right person's current ID), names aligned
// beside them, each person once.
export function rebindWho(who: string[], codex: CodexPerson[], previous?: Pick<PlannerCard, 'who' | 'who_names'>): Pick<PlannerCard, 'who' | 'who_names'> {
  const bound: string[] = []
  const names: string[] = []
  for (const id of who) {
    const stored = storedName(id, previous)
    const person = whoPerson(id, stored, codex)
    const next = person?.id ?? id
    if (bound.includes(next)) continue
    bound.push(next)
    names.push(person?.name ?? stored ?? id)
  }
  return { who: bound, who_names: names }
}

// A card's people by first name: the current codex name of the person each
// ID means, else the name recorded on the card when the ID was set, else the
// raw ID.
export function whoText(card: PlannerCard, codex: CodexPerson[]): string {
  return card.who.map((id, i) => {
    const stored = card.who_names?.[i]
    return firstName(whoPerson(id, stored, codex)?.name ?? stored ?? id)
  }).join(', ')
}

// Whether a card's `who` includes a codex person, by what its entries mean
// now rather than by raw ID.
export function whoIncludes(card: Pick<PlannerCard, 'who' | 'who_names'>, person: CodexPerson, codex: CodexPerson[]): boolean {
  return card.who.some((id, i) => whoPerson(id, card.who_names?.[i], codex)?.id === person.id)
}

export function chapterLabel(ch: PlannerChapter | undefined, fallback = 'Later'): string {
  if (!ch) return fallback
  return `Chapter ${ch.num}${ch.title ? ` · ${ch.title}` : ''}`
}

// ── Timeline layout ────────────────────────────────────────────────────────

export interface LayoutMetrics { cardH: number; colW: number; gap: number; pad: number; headerH: number; laneW: number }

export function layoutMetrics(compact: boolean, hasBeats: boolean): LayoutMetrics {
  return { cardH: compact ? 66 : 92, colW: compact ? 176 : 236, gap: 6, pad: 8, headerH: hasBeats ? 78 : 56, laneW: 156 }
}

export interface LaneRow { lane: PlannerLane; height: number; top: number }

// Row heights follow the busiest cell on each lane so cards never overlap.
export function laneRows(lanes: PlannerLane[], cards: DisplayCard[], columnIds: string[], m: LayoutMetrics): LaneRow[] {
  let top = m.headerH
  return lanes.map(lane => {
    let max = 1
    for (const col of columnIds) {
      const n = cards.filter(c => c.chapter_id === col && c.lines[0] === lane.id).length
      if (n > max) max = n
    }
    const height = m.pad * 2 + max * m.cardH + (max - 1) * m.gap
    const row = { lane, height, top }
    top += height
    return row
  })
}

export interface Connector { left: number; top: number; height: number; width: number; color: string; topCap: boolean; bottomCap: boolean }
export interface Tie { laneId: string; left: number; top: number; color: string }

// A card on several lines is drawn once, on its first line, with a bracket
// down (or up) to a small diamond on each crossing lane.
export function columnConnectors(cardsInColumn: DisplayCard[], rows: LaneRow[], lanes: PlannerLane[], m: LayoutMetrics): { connectors: Connector[]; ties: Tie[] } {
  const rowOf = Object.fromEntries(rows.map(r => [r.lane.id, r]))
  const connectors: Connector[] = []
  const ties: Tie[] = []
  let k = 0
  for (const card of cardsInColumn) {
    const primary = rowOf[card.lines[0]]
    if (!primary) continue
    const others = card.lines.slice(1).filter(id => rowOf[id])
    if (!others.length) continue
    const x = 6 + k * 5
    k++
    const color = lanes.find(l => l.id === card.lines[0])?.color ?? MAIN_LANE_COLOR
    const index = cardsInColumn.filter(o => o.lines[0] === card.lines[0]).indexOf(card)
    const cy = primary.top + m.pad + index * (m.cardH + m.gap) + m.cardH / 2
    const ys = others.map(id => {
      const ty = rowOf[id].top + m.pad + m.cardH / 2
      ties.push({ laneId: id, left: x - 3, top: m.pad + m.cardH / 2 - 3, color })
      return ty
    })
    const top = Math.min(cy, ...ys)
    const bottom = Math.max(cy, ...ys)
    connectors.push({ left: x, top, height: bottom - top, width: 20 - x, color, topCap: cy === top, bottomCap: cy === bottom })
  }
  return { connectors, ties }
}

export function beatMarks(template: string | undefined, gridWidth: number): { name: string; left: number }[] {
  if (!template || !(template in BEATS)) return []
  return BEATS[template as Exclude<BeatTemplateId, 'none'>].map(b => ({ name: b.name, left: Math.round(b.pct * gridWidth) }))
}

// ── Dead ideas ─────────────────────────────────────────────────────────────

// A deleted or dismissed card is written into the Dead Ideas note as an
// outline entry, so Propose Cards can bring it back.
export function deadIdeaBlock(card: PlannerCard, why: 'Deleted' | 'Dismissed', chapters: PlannerChapter[], lanes: PlannerLane[], codex: CodexPerson[]): string {
  const lane = lanes.find(l => l.id === card.lines[0])?.name ?? 'Main plot'
  const ch = chapters.find(c => c.id === card.chapter_id)
  const where = ch ? `Chapter ${ch.num}${ch.title ? ` · ${ch.title}` : ''}` : 'Later'
  const who = card.who.map(id => codex.find(c => c.id === id)?.name ?? id).join(', ')
  return `\n## ${card.title}\n${card.synopsis || ''}${card.changes ? `\nWhat changes: ${card.changes}` : ''}\n— ${why} from ${where} · ${lane}${who ? ` · ${who}` : ''}\n`
}

// ── Import Outline ─────────────────────────────────────────────────────────

export interface Proposal {
  id: string
  title: string
  synopsis: string
  chapterNum: number
  laneId: string
  who: string[]
  accepted: boolean
}

export interface ParsedOutline {
  proposals: Proposal[]
  // Chapter titles the outline supplied ("# Chapter 3 — The Alarm"), by number.
  titles: Record<number, string>
}

const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

function hasAlias(text: string, alias: string): boolean {
  return new RegExp(`(^|[^\\p{L}\\p{N}_])${escapeRe(alias)}($|[^\\p{L}\\p{N}_])`, 'iu').test(text)
}

function shortenedTitle(text: string): string {
  const chars = Array.from(text)
  if (chars.length <= 72) return text
  const prefix = chars.slice(0, 69).join('')
  return prefix.replace(/\s+\S*$/, '') + '…'
}

function splitSentences(text: string): string[] {
  return text.match(/[^.!?]+[.!?]+["'”’]?|[^.!?]+$/g) ?? [text]
}

// Deterministic rules, no engine and no AI:
//  - '#' headings and "Chapter N" / "Part N" / "Act N" lines set the chapter
//    position for everything under them and may name the chapter.
//  - List items and paragraphs become cards; the first sentence is the title,
//    the rest the synopsis.
//  - A recognised beat name lands on the main line at its manuscript percent.
//  - Names the codex knows pick the card's "who"; the first who that has a
//    character lane picks the lane, otherwise the main line.
//  - Unstructured text with many paragraphs is grouped into scene-sized
//    proposals; headings, beats, and lists always retain their explicit shape.
export function parseOutline(text: string, chapters: PlannerChapter[], codex: CodexPerson[], lanes: PlannerLane[]): ParsedOutline {
  const out: Proposal[] = []
  const titles: Record<number, string> = {}
  let current: number | null = null
  let index = 0
  let para: string[] = []
  let listIndent: number | null = null
  let lastList: Proposal | null = null
  let sawStructure = false
  let seq = 0

  const make = (t: string): Proposal => {
    const sentences = splitSentences(t)
    let title = t.trim()
    let synopsis = ''
    if (sentences.length > 1) {
      title = sentences[0].trim()
      synopsis = sentences.slice(1).join(' ').trim()
    }
    title = shortenedTitle(title)
    const who = codex.filter(c => c.aliases.some(a => hasAlias(t, a))).map(c => c.id)
    const laneChar = who.map(w => lanes.find(l => l.character_id === w)).find(Boolean)
    return { id: `p${++seq}`, title, synopsis, chapterNum: current ?? 1, laneId: laneChar ? laneChar.id : MAIN_LANE_ID, who, accepted: true }
  }
  const flush = () => {
    if (!para.length) return
    const t = para.join(' ').trim()
    para = []
    if (t) out.push(make(t))
  }

  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    const isHeading = /^#{1,6}\s+/.test(line) || (line.length < 60 && /^(chapter|part|act)\s+\w+/i.test(line))
    if (isHeading) {
      flush()
      sawStructure = true
      listIndent = null
      lastList = null
      const m = line.match(/(?:chapter|part|act)\s+(\d+)/i)
      current = m ? Math.max(1, +m[1]) : index + 1
      index = current
      const rest = line.replace(/^#{1,6}\s+/, '').replace(/^(chapter|part|act)\s+\w+\s*[—:–-]?\s*/i, '').trim()
      if (rest && rest.length < 60) titles[current] = rest
      continue
    }
    const beat = BEATS_ALL.find(b => new RegExp(`^${escapeRe(b.name)}\\b`, 'i').test(line))
    if (beat) {
      flush()
      sawStructure = true
      listIndent = null
      lastList = null
      const p = make(line.replace(/^[^:—-]+[:—-]\s*/, '') || line)
      p.title = beat.name + (p.title && p.title !== line ? ` — ${p.title}` : '')
      p.chapterNum = Math.max(1, Math.round(beat.pct * (chapters.length || 12)))
      out.push(p)
      continue
    }
    const item = line.match(/^(?:[-*•]|\d+[.)])\s+(.*)/)
    if (item) {
      flush()
      sawStructure = true
      const indent = (raw.match(/^\s*/) ?? [''])[0].replace(/\t/g, '    ').length
      if (listIndent !== null && indent > listIndent && lastList) {
        lastList.synopsis = [lastList.synopsis, item[1].trim()].filter(Boolean).join(' ')
        continue
      }
      listIndent = indent
      lastList = make(item[1])
      out.push(lastList)
      continue
    }
    if (!line || line === '⁂') {
      flush()
      listIndent = null
      lastList = null
      continue
    }
    if (lastList) {
      lastList.synopsis = [lastList.synopsis, line].filter(Boolean).join(' ')
      continue
    }
    para.push(line)
  }
  flush()

  if (!sawStructure && out.length > 12) {
    const drafted = chapters.filter(c => c.drafted).length || chapters.length || 8
    const per = Math.ceil(out.length / Math.min(12, drafted))
    const grouped: Proposal[] = []
    for (let i = 0; i < out.length; i += per) {
      const chunk = out.slice(i, i + per)
      const who = [...new Set(chunk.flatMap(c => c.who))]
      const laneChar = who.map(w => lanes.find(l => l.character_id === w)).find(Boolean)
      grouped.push({
        id: `p${++seq}`, title: chunk[0].title, synopsis: chunk.slice(1).map(c => c.title).join(' '),
        chapterNum: Math.floor(i / per) + 1, laneId: laneChar ? laneChar.id : MAIN_LANE_ID, who, accepted: true,
      })
    }
    return { proposals: grouped, titles }
  }
  return { proposals: out, titles }
}
