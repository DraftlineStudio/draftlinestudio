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
  sourceKey: string
  title: string
  synopsis: string
  groupId: string
  groupName: string
  laneId: string
  who: string[]
  accepted: boolean
}

export interface ParsedOutline {
  proposals: Proposal[]
  lanes: ProposalLane[]
  characters: ProposalCharacter[]
}

export interface ProposalLane { id: string; name: string }
export interface ProposalCharacter {
  id: string
  name: string
  aliases: string[]
  description: string
  accepted: boolean
  createLane: boolean
}

function sourceToken(value: string): string {
  return cleanOutlineLine(value).normalize('NFKC').toLocaleLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-+|-+$/g, '') || 'untitled'
}

// A Scratchpad outline can be proposed repeatedly while it grows. Accepted
// movements carry their source key on the card, so only genuinely new
// movements return to the review screen. Title matching covers cards imported
// before source keys existed; it is intentionally limited to the same note.
export function onlyNewOutlineProposals(parsed: ParsedOutline, cards: PlannerCard[], sourceId: string): ParsedOutline {
  const previous = cards.filter(card => card.origin === 'outline' && card.source_id === sourceId)
  const keys = new Set(previous.map(card => card.source_key).filter((key): key is string => !!key))
  const legacyTitles = new Set(previous.filter(card => !card.source_key).map(card => sourceToken(card.title)))
  const proposals = parsed.proposals.filter(proposal => !keys.has(proposal.sourceKey) && !legacyTitles.has(sourceToken(proposal.title)))
  const usedLanes = new Set(proposals.map(proposal => proposal.laneId))
  return { ...parsed, proposals, lanes: parsed.lanes.filter(lane => usedLanes.has(lane.id)) }
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

const MAX_OUTLINE_CARDS = 48
const MAX_OUTLINE_SYNOPSIS = 280
const OUTLINE_GROUP = 'Outline'

interface OutlineHeading { level: number; title: string }

function cleanOutlineLine(value: string): string {
  return value
    .trim()
    .replace(/^#{1,6}\s+/, '')
    .replace(/^(?:[-*•]|\d+[.)])\s+/, '')
    .replace(/^\*\*([^*]+)\*\*\s*[:—–-]?\s*/, '$1 — ')
    .replace(/\*\*|__|`/g, '')
    .replace(/\s+/g, ' ')
    .replace(/\s+([,.;:!?])/g, '$1')
    .trim()
}

function outlineHeading(raw: string): OutlineHeading | null {
  const line = raw.trim()
  const markdown = line.match(/^(#{1,6})\s+(.+)/)
  if (markdown) return { level: markdown[1].length, title: cleanOutlineLine(markdown[2]) }
  if (line.length < 100 && /^(?:part|act)\s+[\p{L}\p{N}]+\b/iu.test(line)) return { level: 1, title: cleanOutlineLine(line) }
  if (line.length < 100 && /^chapter\s+[\p{L}\p{N}]+\b/iu.test(line)) return { level: 2, title: cleanOutlineLine(line) }
  return null
}

function characterSection(title: string): boolean {
  return /^(?:(?:principal|main|supporting|minor)\s+)?characters?$|^cast(?:\s+of\s+characters)?$|^dramatis\s+personae$/iu.test(title.trim())
}

function characterNames(title: string): string[] {
  return title.split(/\s+(?:\/|aka|a\.k\.a\.)\s+/iu).map(cleanOutlineLine).filter(name => {
    const words = name.split(/\s+/)
    return name.length >= 2 && name.length <= 60 && words.length <= 5 && words.every(word => /^[\p{L}\p{N}'’.-]+$/u.test(word))
  })
}

function outlineCharacters(text: string, codex: CodexPerson[]): { text: string; characters: ProposalCharacter[] } {
  const kept: string[] = []
  const characters: ProposalCharacter[] = []
  const known = new Set(codex.flatMap(person => [person.name, ...person.aliases]).map(name => name.trim().toLocaleLowerCase()))
  let sectionLevel: number | null = null
  let current: { names: string[]; details: string[] } | null = null
  let seq = 0
  const flush = () => {
    if (!current?.names.length) { current = null; return }
    const names = current.names.some(name => known.has(name.toLocaleLowerCase())) ? [] : current.names
    if (names.length) {
      const canonical = names[0]
      const first = canonical.split(/\s+/)[0]
      const aliases = [...new Set([...names.slice(1), ...(canonical.includes(' ') ? [first] : [])])]
      characters.push({
        id: `pc:${++seq}`, name: canonical, aliases,
        description: shortenedCharacterDescription(current.details.map(cleanOutlineLine).filter(Boolean).join(' ')),
        accepted: true, createLane: true,
      })
      for (const name of [canonical, ...aliases]) known.add(name.toLocaleLowerCase())
    }
    current = null
  }

  for (const raw of text.split(/\r?\n/)) {
    const heading = outlineHeading(raw)
    if (sectionLevel !== null) {
      if (heading && heading.level <= sectionLevel) {
        flush()
        sectionLevel = null
      } else {
        if (heading) {
          flush()
          current = { names: characterNames(heading.title), details: [] }
        } else if (current && raw.trim()) current.details.push(raw)
        continue
      }
    }
    if (heading && characterSection(heading.title)) {
      sectionLevel = heading.level
      continue
    }
    kept.push(raw)
  }
  flush()
  return { text: kept.join('\n'), characters }
}

function shortenedCharacterDescription(value: string): string {
  const text = value.replace(/\s+/g, ' ').trim()
  if (Array.from(text).length <= 500) return text
  return Array.from(text).slice(0, 497).join('').replace(/\s+\S*$/, '') + '…'
}

function shortenedSynopsis(value: string): string {
  const text = value.replace(/\s+/g, ' ').trim()
  if (Array.from(text).length <= MAX_OUTLINE_SYNOPSIS) return text
  const prefix = Array.from(text).slice(0, MAX_OUTLINE_SYNOPSIS - 3).join('')
  return prefix.replace(/\s+\S*$/, '') + '…'
}

function proposalText(title: string, detailLines: string[]): { title: string; synopsis: string; source: string } {
  const details = detailLines.map(cleanOutlineLine).filter(Boolean)
  if (title) {
    return { title: shortenedTitle(cleanOutlineLine(title)), synopsis: shortenedSynopsis(details.join(' ')), source: [title, ...details].join(' ') }
  }
  const source = details.join(' ')
  const sentences = splitSentences(source)
  const first = sentences[0]?.trim() || 'Untitled card'
  return { title: shortenedTitle(first), synopsis: shortenedSynopsis(sentences.slice(1).join(' ').trim()), source }
}

function flatOutlineUnits(text: string): string[] {
  const units: string[] = []
  let currentList: { indent: number; text: string[] } | null = null
  let paragraph: string[] = []
  const flushParagraph = () => {
    if (paragraph.length) units.push(paragraph.join(' '))
    paragraph = []
  }
  const flushList = () => {
    if (currentList) units.push(currentList.text.join(' '))
    currentList = null
  }
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line || line === '⁂') {
      flushParagraph()
      flushList()
      continue
    }
    const item = raw.match(/^(\s*)(?:[-*•]|\d+[.)])\s+(.*)/)
    if (item) {
      flushParagraph()
      const indent = item[1].replace(/\t/g, '    ').length
      const value = cleanOutlineLine(item[2])
      if (currentList && indent > currentList.indent) currentList.text.push(value)
      else {
        flushList()
        currentList = { indent, text: [value] }
      }
      continue
    }
    flushList()
    paragraph.push(cleanOutlineLine(line))
  }
  flushParagraph()
  flushList()
  return units.filter(Boolean)
}

function compactUnits(units: string[]): string[][] {
  if (units.length <= 12) return units.map(unit => [unit])
  const total = units.reduce((sum, unit) => sum + unit.length, 0)
  const target = Math.max(600, Math.ceil(total / MAX_OUTLINE_CARDS))
  const chunks: string[][] = []
  let chunk: string[] = []
  let size = 0
  for (const unit of units) {
    if (chunk.length && size >= target) {
      chunks.push(chunk)
      chunk = []
      size = 0
    }
    chunk.push(unit)
    size += unit.length
  }
  if (chunk.length) chunks.push(chunk)
  return chunks
}

// Freeform, deterministic outline import. Headings establish hierarchy, not
// manuscript chapters: the second heading level becomes cards and the level
// above it becomes an optional story-line group. Supporting headings, bullets,
// and paragraphs are folded into each card's synopsis. Text without a useful
// heading hierarchy is compacted into substantial adjacent blocks. Every card
// is unpinned (Later) until the writer attaches it to a manuscript chapter.
export function parseOutline(text: string, codex: CodexPerson[], lanes: PlannerLane[]): ParsedOutline {
  let seq = 0
  let groupSeq = 0
  const proposals: Proposal[] = []
  const sourceKeys = new Map<string, number>()
  const prepared = outlineCharacters(text, codex)
  const outlineText = prepared.text
  const headings = outlineText.split(/\r?\n/).map(outlineHeading).filter((h): h is OutlineHeading => !!h)
  const levels = [...new Set(headings.map(h => h.level))].sort((a, b) => a - b)
  const cardLevel = levels.length > 1 ? levels[1] : levels[0]
  const groups = new Map<string, string>()
  const proposedLanes: ProposalLane[] = []
  const groupID = (name: string) => {
    const key = name.trim().toLocaleLowerCase() || OUTLINE_GROUP.toLocaleLowerCase()
    if (!groups.has(key)) {
      const id = `g${++groupSeq}`
      groups.set(key, id)
      if (name !== OUTLINE_GROUP) proposedLanes.push({ id: `new:${id}`, name })
    }
    return groups.get(key)!
  }
  const make = (title: string, details: string[], groupName: string) => {
    const shaped = proposalText(title, details)
    if (!shaped.source.trim()) return
    const who = [
      ...codex.filter(c => c.aliases.some(a => hasAlias(shaped.source, a))).map(c => c.id),
      ...prepared.characters.filter(c => [c.name, ...c.aliases].some(a => hasAlias(shaped.source, a))).map(c => c.id),
    ]
    const laneChar = who.map(w => lanes.find(l => l.character_id === w)).find(Boolean)
    const groupId = groupID(groupName)
    const sourceBase = `${sourceToken(groupName)}::${sourceToken(shaped.title)}`
    const sourceOccurrence = (sourceKeys.get(sourceBase) ?? 0) + 1
    sourceKeys.set(sourceBase, sourceOccurrence)
    proposals.push({
      id: `p${++seq}`, sourceKey: sourceOccurrence === 1 ? sourceBase : `${sourceBase}::${sourceOccurrence}`,
      title: shaped.title, synopsis: shaped.synopsis,
      groupId, groupName,
      laneId: groupName === OUTLINE_GROUP ? (laneChar?.id ?? MAIN_LANE_ID) : `new:${groupId}`,
      who, accepted: true,
    })
  }

  if (cardLevel !== undefined) {
    let groupName = OUTLINE_GROUP
    let current: { title: string; details: string[]; groupName: string } | null = null
    let groupPrelude: string[] = []
    const flush = () => {
      if (current) make(current.title, current.details, current.groupName)
      current = null
    }
    const flushPrelude = () => {
      if (groupPrelude.length) make(groupName, groupPrelude, groupName)
      groupPrelude = []
    }
    for (const raw of outlineText.split(/\r?\n/)) {
      const heading = outlineHeading(raw)
      if (heading) {
        if (heading.level < cardLevel) {
          flush()
          flushPrelude()
          groupName = heading.title || OUTLINE_GROUP
        } else if (heading.level === cardLevel) {
          flush()
          current = { title: heading.title, details: groupPrelude, groupName }
          groupPrelude = []
        } else if (current) current.details.push(heading.title)
        else groupPrelude.push(heading.title)
        continue
      }
      if (!raw.trim() || raw.trim() === '⁂') continue
      if (current) current.details.push(raw)
      else groupPrelude.push(raw)
    }
    flush()
    flushPrelude()
  } else {
    for (const chunk of compactUnits(flatOutlineUnits(outlineText))) make('', chunk, OUTLINE_GROUP)
  }

  if (proposals.length <= MAX_OUTLINE_CARDS) return { proposals, lanes: proposedLanes, characters: prepared.characters }
  const stride = Math.ceil(proposals.length / MAX_OUTLINE_CARDS)
  const compacted: Proposal[] = []
  for (let i = 0; i < proposals.length; i += stride) {
    const chunk = proposals.slice(i, i + stride)
    const first = chunk[0]
    compacted.push({
      ...first,
      synopsis: shortenedSynopsis([first.synopsis, ...chunk.slice(1).map(other => `${other.title}. ${other.synopsis}`)].filter(Boolean).join(' ')),
      who: [...new Set(chunk.flatMap(p => p.who))],
    })
  }
  return { proposals: compacted, lanes: proposedLanes, characters: prepared.characters }
}
