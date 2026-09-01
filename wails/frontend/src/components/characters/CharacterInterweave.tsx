import { useMemo, useState } from 'react'
import type { BookData, Character, RelationshipRecord } from '../../types/draftline'
import { characterColor } from '../../utils/characterVisuals'
import { chapterName } from './shared'

const ROW_HEIGHT = 38
const CHAPTER_WIDTH = 58
const PAD_X = 28
const PAD_Y = 20

interface Props {
  book: BookData
  characters: Character[]
  relationships: RelationshipRecord[]
  selectedId: string | null
  onSelect: (id: string) => void
}

/**
 * A manuscript-order character graph. Presence stays horizontal; each
 * source-backed relationship crossing bends between its two character rails
 * in the chapter where the interaction was recorded.
 */
export default function CharacterInterweave({ book, characters, relationships, selectedId, onSelect }: Props) {
  const [scrollTop, setScrollTop] = useState(0)
  const chapterCount = Math.max(1, [...(book.front_matter ?? []), ...(book.body ?? []), ...(book.back_matter ?? [])].length)
  const width = PAD_X * 2 + chapterCount * CHAPTER_WIDTH
  const height = PAD_Y * 2 + Math.max(1, characters.length) * ROW_HEIGHT
  const position = useMemo(() => new Map(characters.map((character, index) => [character.id, index])), [characters])
  const visibleRelationships = relationships.filter(rel => position.has(rel.character1_id) && position.has(rel.character2_id))
  const x = (chapter: number) => PAD_X + Math.max(0, Math.min(chapterCount - 1, chapter)) * CHAPTER_WIDTH + CHAPTER_WIDTH / 2
  const y = (id: string) => PAD_Y + (position.get(id) ?? 0) * ROW_HEIGHT + ROW_HEIGHT / 2

  return (
    <div className="chars-weave">
      <div className="chars-weave-names">
        <div style={{ paddingTop: PAD_Y + 24, transform: `translateY(${-scrollTop}px)` }}>
          {characters.map(character => (
          <button
            type="button"
            key={character.id}
            className={character.id === selectedId ? 'selected' : ''}
            style={{ height: ROW_HEIGHT }}
            onClick={() => onSelect(character.id)}
            title={`${character.name} · ${character.mention_count ?? 0} mentions`}
          >
            <i style={{ background: characterColor(character.name) }} />
            <span>{character.name}</span>
            <small>{character.mention_count ?? 0}</small>
          </button>
          ))}
        </div>
      </div>

      <div className="chars-weave-scroll" onScroll={event => setScrollTop(event.currentTarget.scrollTop)}>
        <div className="chars-weave-chapters" style={{ width }}>
          {Array.from({ length: chapterCount }, (_, chapter) => (
            <span key={chapter} style={{ width: CHAPTER_WIDTH }} title={chapterName(book, chapter)}>{chapter + 1}</span>
          ))}
        </div>
        <svg className="chars-weave-svg" width={width} height={height} aria-label="Character interaction timeline">
          {Array.from({ length: chapterCount + 1 }, (_, chapter) => (
            <line key={`grid-${chapter}`} x1={PAD_X + chapter * CHAPTER_WIDTH} y1={0} x2={PAD_X + chapter * CHAPTER_WIDTH} y2={height} className="chars-weave-grid" />
          ))}

          {characters.map(character => {
            const chapters = Object.entries(character.chapter_mentions ?? {})
              .filter(([, count]) => count > 0)
              .map(([chapter]) => Number(chapter))
              .sort((a, b) => a - b)
            if (chapters.length === 0) return null
            const color = characterColor(character.name)
            const active = !selectedId || selectedId === character.id
            return (
              <g key={`lane-${character.id}`} opacity={active ? 1 : 0.2}>
                <line
                  x1={x(chapters[0])}
                  y1={y(character.id)}
                  x2={x(chapters[chapters.length - 1])}
                  y2={y(character.id)}
                  stroke={color}
                  strokeWidth={selectedId === character.id ? 3 : 2}
                  strokeLinecap="round"
                />
                {chapters.map(chapter => (
                  <g key={`${character.id}-${chapter}`} className="chars-weave-node" onClick={() => onSelect(character.id)}>
                    <title>{`${character.name} · ${chapterName(book, chapter)} · ${character.chapter_mentions?.[chapter] ?? 0} mentions`}</title>
                    <circle cx={x(chapter)} cy={y(character.id)} r={9} fill="transparent" />
                    <circle cx={x(chapter)} cy={y(character.id)} r={4.2} fill={color} stroke="var(--bg-panel)" strokeWidth={1.5} />
                  </g>
                ))}
              </g>
            )
          })}

          {visibleRelationships.flatMap(relationship => {
            const color = characterColor(characters[position.get(relationship.character1_id) ?? 0]?.name ?? '')
            const chapters = [...new Set(relationship.chapter_history ?? [])]
            const active = !selectedId || selectedId === relationship.character1_id || selectedId === relationship.character2_id
            return chapters.map(chapter => {
              const cx = x(chapter)
              const y1 = y(relationship.character1_id)
              const y2 = y(relationship.character2_id)
              const bend = Math.min(24, Math.abs(y2 - y1) / 3)
              return (
                <path
                  key={`${relationship.id}-${chapter}`}
                  d={`M ${cx - bend} ${y1} C ${cx} ${y1}, ${cx} ${y2}, ${cx + bend} ${y2}`}
                  fill="none"
                  stroke={color}
                  strokeWidth={Math.max(1.25, Math.min(3, 1 + relationship.strength * 2))}
                  opacity={active ? 0.7 : 0.08}
                  className="chars-weave-link"
                >
                  <title>{`${chapterName(book, chapter)} · ${relationship.interaction_count} interactions across the manuscript`}</title>
                </path>
              )
            })
          })}
        </svg>
      </div>
      <div className="chars-weave-key">Dots show presence · curves show confirmed interaction overlap</div>
    </div>
  )
}
