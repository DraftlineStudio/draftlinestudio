import { useEffect, useMemo, useState } from 'react'
import { BuildStoryTimeline } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import type { BookData, Character, RelationshipRecord, Section, StoryTimelineEvent, StoryTimelineResult } from '../../types/draftline'
import { characterColor } from '../../utils/characterVisuals'
import { allChapters, chapterLocation, chapterName } from './shared'

const ROW_HEIGHT = 44
const PAD_Y = 22
const PAD_X = 16
const NAME_WIDTH = 208
/** Chapter column widths, narrow → wide. */
const CHAPTER_WIDTHS = [46, 78, 128, 210] as const

const KEY_BEAT_TYPES = new Set(['turning_point', 'conflict', 'resolution', 'discovery'])
const NOTABLE_BEAT_TYPES = new Set(['introduction', 'first_interaction', 'time_reference'])
const BEAT_TYPE_COLORS: Record<string, string> = {
  introduction: '#5aafe0',
  first_interaction: '#5aafe0',
  interaction: '#5aafe0',
  discovery: '#6699FF',
  time_reference: '#E5C07B',
  conflict: '#E5C07B',
  turning_point: '#E06C75',
  transition: '#868C96',
  state: '#57A874',
  resolution: '#4ade80',
}

interface Props {
  book: BookData
  characters: Character[]
  relationships: RelationshipRecord[]
  selectedId: string | null
  onSelect: (id: string) => void
  onOpenChapter?: (section: Section, index: number) => void
}

type Hover =
  | { kind: 'beat'; x: number; y: number; event: StoryTimelineEvent }
  | { kind: 'cross'; x: number; y: number; a: string; b: string; chapter: number; interactions: number }
  | null

/**
 * The Character Center's large-format weave: a manuscript-order view of
 * character presence, interactions, and source-backed story beats.
 *
 * Presence is drawn as real segments so absences are visible rather than
 * smoothed over, source-backed story beats sit at their true position inside a
 * chapter (shared placement rule with the Story Graph), and each confirmed
 * relationship crossing is offset within its chapter so two pairs meeting in
 * the same chapter stay distinguishable.
 */
export default function CharacterInterweave({ book, characters, relationships, selectedId, onSelect, onOpenChapter }: Props) {
  const [timeline, setTimeline] = useState<StoryTimelineResult | null>(null)
  const [zoom, setZoom] = useState(1)
  const [showBeats, setShowBeats] = useState(true)
  const [showCrossings, setShowCrossings] = useState(true)
  const [hover, setHover] = useState<Hover>(null)

  useEffect(() => {
    let cancelled = false
    void BuildStoryTimeline(book as types.BookData)
      .then(value => { if (!cancelled) setTimeline(value as unknown as StoryTimelineResult) })
      .catch(() => { if (!cancelled) setTimeline(null) })
    return () => { cancelled = true }
  }, [book.file_path, book.analysis?.evidence?.content_hash])

  const chapters = useMemo(() => allChapters(book), [book])
  const chapterCount = Math.max(1, chapters.length)
  const chapterWidth = CHAPTER_WIDTHS[zoom]
  const width = PAD_X * 2 + chapterCount * chapterWidth
  const height = PAD_Y * 2 + Math.max(1, characters.length) * ROW_HEIGHT

  const position = useMemo(() => new Map(characters.map((character, index) => [character.id, index])), [characters])
  const rowY = (id: string) => PAD_Y + (position.get(id) ?? 0) * ROW_HEIGHT + ROW_HEIGHT / 2
  const columnX = (chapter: number) => PAD_X + clampChapter(chapter, chapterCount) * chapterWidth
  const atChapter = (chapter: number, fraction: number) => columnX(chapter) + fraction * chapterWidth

  /** Contiguous runs of chapters a character is present in, so gaps show. */
  const presence = useMemo(() => characters.map(character => ({
    id: character.id,
    runs: contiguousRuns(
      Object.entries(character.chapter_mentions ?? {})
        .filter(([, count]) => count > 0)
        .map(([chapter]) => Number(chapter))
        .filter(chapter => Number.isFinite(chapter))
        .sort((a, b) => a - b),
    ),
  })), [characters])

  /** Beats belonging to a character that has a rail, at their real position. */
  const beats = useMemo(() => {
    if (!showBeats) return []
    return (timeline?.events ?? []).flatMap(event =>
      (event.character_ids ?? [])
        .filter(id => position.has(id))
        .map(id => ({
          id: `${event.id}-${id}`,
          characterID: id,
          event,
          tier: eventTier(event),
          x: atChapter(event.chapter_index, positionOfParagraph(event.paragraph_index)),
          y: rowY(id),
        })),
    )
  }, [timeline, position, showBeats, chapterWidth, chapterCount, characters])

  /**
   * One crossing per (pair, chapter). Multiple crossings in the same chapter
   * are spread across the column instead of stacking on its centre, which is
   * what made a busy chapter unreadable.
   */
  const crossings = useMemo(() => {
    if (!showCrossings) return []
    const byChapter = new Map<number, { rel: RelationshipRecord; chapter: number }[]>()
    for (const rel of relationships) {
      if (!position.has(rel.character1_id) || !position.has(rel.character2_id)) continue
      for (const chapter of new Set(rel.chapter_history ?? [])) {
        const bucket = byChapter.get(chapter) ?? []
        bucket.push({ rel, chapter })
        byChapter.set(chapter, bucket)
      }
    }
    return [...byChapter.entries()].flatMap(([chapter, list]) =>
      list.map((item, index) => {
        // Spread evenly across the column, biased to the middle two thirds.
        const fraction = crossingFraction(index, list.length)
        return {
          key: `${item.rel.id}-${chapter}`,
          rel: item.rel,
          chapter,
          x: atChapter(chapter, fraction),
          y1: rowY(item.rel.character1_id),
          y2: rowY(item.rel.character2_id),
        }
      }),
    )
  }, [relationships, position, showCrossings, chapterWidth, chapterCount, characters])

  const nameOf = (id: string) => characters[position.get(id) ?? -1]?.name ?? 'Unknown'
  const relatedToSelection = useMemo(() => {
    if (!selectedId) return null
    const ids = new Set<string>([selectedId])
    for (const rel of relationships) {
      if (rel.character1_id === selectedId) ids.add(rel.character2_id)
      if (rel.character2_id === selectedId) ids.add(rel.character1_id)
    }
    return ids
  }, [selectedId, relationships])

  const beatCount = beats.length
  const shownChapters = timeline?.chapters?.length ?? 0

  return (
    <div className="chars-weave">
      <div className="chars-weave-toolbar">
        <div className="chars-weave-zoom">
          <button type="button" onClick={() => setZoom(v => Math.max(0, v - 1))} disabled={zoom === 0} aria-label="Narrower chapters">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M5 12h14" /></svg>
          </button>
          <button type="button" onClick={() => setZoom(v => Math.min(CHAPTER_WIDTHS.length - 1, v + 1))} disabled={zoom === CHAPTER_WIDTHS.length - 1} aria-label="Wider chapters">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M5 12h14M12 5v14" /></svg>
          </button>
        </div>
        <label className={showBeats ? 'active' : ''}>
          <input type="checkbox" checked={showBeats} onChange={e => setShowBeats(e.target.checked)} />
          Story beats
        </label>
        <label className={showCrossings ? 'active' : ''}>
          <input type="checkbox" checked={showCrossings} onChange={e => setShowCrossings(e.target.checked)} />
          Interactions
        </label>
        <span className="chars-weave-counts">
          {characters.length} {characters.length === 1 ? 'rail' : 'rails'}
          {showBeats && timeline ? ` · ${beatCount} ${beatCount === 1 ? 'beat' : 'beats'} across ${shownChapters} ${shownChapters === 1 ? 'chapter' : 'chapters'}` : ''}
          {showCrossings ? ` · ${crossings.length} ${crossings.length === 1 ? 'crossing' : 'crossings'}` : ''}
        </span>
      </div>

      <div className="chars-weave-body">
        <div className="chars-weave-scroll">
          <div className="chars-weave-stage" style={{ width: NAME_WIDTH + width }}>
            <div className="chars-weave-header" style={{ paddingLeft: NAME_WIDTH + PAD_X }}>
              {chapters.map((chapter, index) => (
                <span
                  key={index}
                  style={{ width: chapterWidth }}
                  title={chapter.title || `Chapter ${index + 1}`}
                  onClick={() => onOpenChapter?.(...locate(book, index))}
                >
                  {zoom >= 2 ? (chapter.title || `Chapter ${index + 1}`) : index + 1}
                </span>
              ))}
            </div>

            <div className="chars-weave-rows" style={{ height }}>
              <div className="chars-weave-names" style={{ width: NAME_WIDTH, paddingTop: PAD_Y }}>
                {characters.map(character => {
                  const dim = relatedToSelection ? !relatedToSelection.has(character.id) : false
                  return (
                    <button
                      type="button"
                      key={character.id}
                      className={`${character.id === selectedId ? 'selected' : ''} ${dim ? 'dim' : ''}`}
                      style={{ height: ROW_HEIGHT }}
                      onClick={() => onSelect(character.id)}
                      title={`${character.name} · ${character.mention_count ?? 0} mentions`}
                    >
                      <i style={{ background: characterColor(character.name) }} />
                      <span>{character.name}</span>
                      <small>{character.mention_count ?? 0}</small>
                    </button>
                  )
                })}
              </div>

              <svg
                className="chars-weave-svg"
                width={width}
                height={height}
                aria-label="Character interaction timeline"
              >
                {Array.from({ length: chapterCount + 1 }, (_, chapter) => (
                  <line key={`grid-${chapter}`} x1={PAD_X + chapter * chapterWidth} y1={0} x2={PAD_X + chapter * chapterWidth} y2={height} className="chars-weave-grid" />
                ))}

                {/* Row bands make it possible to track a rail across a wide canvas. */}
                {characters.map((character, index) => (
                  index % 2 === 1 ? (
                    <rect key={`band-${character.id}`} x={0} y={PAD_Y + index * ROW_HEIGHT} width={width} height={ROW_HEIGHT} className="chars-weave-band" />
                  ) : null
                ))}

                {presence.map(({ id, runs }) => {
                  const character = characters[position.get(id) ?? -1]
                  if (!character) return null
                  const color = characterColor(character.name)
                  const dim = relatedToSelection ? !relatedToSelection.has(id) : false
                  const isSelected = id === selectedId
                  return (
                    <g key={`rail-${id}`} opacity={dim ? 0.18 : 1}>
                      {runs.map(([from, to]) => (
                        <line
                          key={`${id}-${from}-${to}`}
                          x1={atChapter(from, 0.12)}
                          y1={rowY(id)}
                          x2={atChapter(to, 0.88)}
                          y2={rowY(id)}
                          stroke={color}
                          strokeWidth={isSelected ? 3 : 2}
                          strokeLinecap="round"
                          opacity={isSelected ? 1 : 0.75}
                        />
                      ))}
                    </g>
                  )
                })}

                {crossings.map(crossing => {
                  const involved = !selectedId || crossing.rel.character1_id === selectedId || crossing.rel.character2_id === selectedId
                  const color = characterColor(nameOf(crossing.rel.character1_id))
                  const span = Math.abs(crossing.y2 - crossing.y1)
                  const bend = Math.max(10, Math.min(30, span / 2.5))
                  return (
                    <path
                      key={crossing.key}
                      className="chars-weave-link"
                      d={`M ${crossing.x - bend} ${crossing.y1} C ${crossing.x} ${crossing.y1}, ${crossing.x} ${crossing.y2}, ${crossing.x + bend} ${crossing.y2}`}
                      fill="none"
                      stroke={color}
                      strokeWidth={Math.max(1.25, Math.min(3.2, 1 + crossing.rel.strength * 2))}
                      opacity={involved ? 0.72 : 0.07}
                      onMouseEnter={() => setHover({
                        kind: 'cross',
                        x: crossing.x,
                        y: Math.min(crossing.y1, crossing.y2),
                        a: nameOf(crossing.rel.character1_id),
                        b: nameOf(crossing.rel.character2_id),
                        chapter: crossing.chapter,
                        interactions: crossing.rel.interaction_count,
                      })}
                      onMouseLeave={() => setHover(null)}
                    />
                  )
                })}

                {beats.map(beat => {
                  const dim = relatedToSelection ? !relatedToSelection.has(beat.characterID) : false
                  const radius = beat.tier === 0 ? 5.5 : beat.tier === 1 ? 4 : 3
                  return (
                    <g
                      key={beat.id}
                      className="chars-weave-node"
                      onMouseEnter={() => setHover({ kind: 'beat', x: beat.x, y: beat.y, event: beat.event })}
                      onMouseLeave={() => setHover(null)}
                      onClick={() => {
                        onSelect(beat.characterID)
                        onOpenChapter?.(beat.event.section as Section, beat.event.section_index)
                      }}
                    >
                      <circle cx={beat.x} cy={beat.y} r={10} fill="transparent" />
                      <circle
                        cx={beat.x}
                        cy={beat.y}
                        r={radius}
                        fill={characterColor(nameOf(beat.characterID))}
                        stroke={beat.tier === 0 ? typeColor(beat.event.primary_type) : 'var(--bg-panel)'}
                        strokeWidth={beat.tier === 0 ? 2 : 1.5}
                        opacity={dim ? 0.12 : 1}
                      />
                    </g>
                  )
                })}
              </svg>

              {hover && (
                <div
                  className="chars-weave-tip"
                  style={{ left: NAME_WIDTH + hover.x + 14, top: hover.y + 12 }}
                >
                  {hover.kind === 'beat' ? (
                    <>
                      <strong style={{ color: typeColor(hover.event.primary_type) }}>
                        {label(hover.event.primary_type)} · {hover.event.chapter_title}
                      </strong>
                      <em>“{truncate(hover.event.text, 150)}”</em>
                      {!!hover.event.character_names?.length && <span>{hover.event.character_names.join(' · ')}</span>}
                    </>
                  ) : (
                    <>
                      <strong>{hover.a} &amp; {hover.b}</strong>
                      <span>{chapterName(book, hover.chapter)} · {hover.interactions} recorded {hover.interactions === 1 ? 'interaction' : 'interactions'} across the manuscript</span>
                    </>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="chars-weave-key">
        Rails show presence, with gaps where a character is absent
        {showBeats ? ' · dots are source-backed beats, ringed when decisive' : ''}
        {showCrossings ? ' · curves are confirmed interactions' : ''}
      </div>
    </div>
  )
}

/**
 * Horizontal position of one crossing inside its chapter column. A single
 * crossing sits centred; several are spread across the middle of the column so
 * two pairs meeting in the same chapter never draw on top of each other.
 */
export function crossingFraction(index: number, total: number): number {
  if (total <= 1) return 0.5
  return 0.2 + (index / (total - 1)) * 0.6
}

/** Groups sorted chapter indices into [start, end] runs of consecutive values. */
export function contiguousRuns(sorted: number[]): [number, number][] {
  const runs: [number, number][] = []
  for (const chapter of sorted) {
    const last = runs[runs.length - 1]
    if (last && chapter === last[1] + 1) last[1] = chapter
    else runs.push([chapter, chapter])
  }
  return runs
}

export function clampChapter(chapter: number, count: number): number {
  if (!Number.isFinite(chapter)) return 0
  return Math.max(0, Math.min(count - 1, chapter))
}

/** Importance tier used only to vary beat markers in the character weave. */
export function eventTier(event: StoryTimelineEvent): number {
  const cast = event.character_ids?.length ?? 0
  if (event.pinned || event.status === 'confirmed') return 0
  if (KEY_BEAT_TYPES.has(event.primary_type)) return 0
  if (cast >= 2 && event.time_kind === 'anchored') return 0
  if (NOTABLE_BEAT_TYPES.has(event.primary_type) || cast >= 2) return 1
  return 2
}

export function typeColor(type: string): string {
  return BEAT_TYPE_COLORS[type] ?? '#868C96'
}

/**
 * Fractional placement of a beat inside a chapter from its paragraph index.
 * The logarithmic curve keeps both early and late beats distinguishable in
 * chapters whose paragraph counts vary widely.
 */
export function positionOfParagraph(paragraph: number): number {
  if (!Number.isFinite(paragraph) || paragraph <= 0) return 0.18
  const eased = Math.log10(1 + Math.min(paragraph, 400)) / Math.log10(401)
  return 0.12 + eased * 0.74
}

function locate(book: BookData, index: number): [Section, number] {
  const spot = chapterLocation(book, index)
  return [spot.section, spot.index]
}

function label(value: string): string {
  return value.replace(/_/g, ' ').replace(/^./, letter => letter.toUpperCase())
}

function truncate(value: string, limit: number): string {
  return value.length > limit ? `${value.slice(0, limit - 1)}…` : value
}
