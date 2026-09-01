import { useEffect, useMemo, useRef, useState } from 'react'
import { BuildStoryTimeline } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import type { BookData, Section, StoryTimelineResult } from '../../types/draftline'
import {
  buildStoryGraph,
  CHAPTER_WIDTHS,
  LANE_GAP,
  LANE_TOP,
  PAD_LEFT,
  ZOOM_LABELS,
  laneY,
  nodeX,
  typeColor,
  type GraphNode,
  type LaneMode,
} from './storyGraph'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
  /** Opens the character codex (cast view). */
  onOpenCodex: () => void
}

export default function StoryGraphPanel({ book, onNavigate, onOpenCodex }: Props) {
  const [result, setResult] = useState<StoryTimelineResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [mode, setMode] = useState<LaneMode>('threads')
  const [zoom, setZoom] = useState(1)
  const [selected, setSelected] = useState(0)
  const [focus, setFocus] = useState<string | null>(null)
  const [hover, setHover] = useState<number | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    void BuildStoryTimeline(book as types.BookData)
      .then(value => { if (!cancelled) setResult(value as unknown as StoryTimelineResult) })
      .catch(reason => { if (!cancelled) setError(String(reason)) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [book.file_path, book.analysis?.evidence?.content_hash])

  const graph = useMemo(
    () => (result ? buildStoryGraph(result, mode, zoom) : null),
    [result, mode, zoom],
  )

  // Selection is an index into a list that changes with mode and zoom; clamp it
  // rather than letting the inspector read past the end.
  useEffect(() => {
    if (graph && selected >= graph.nodes.length) setSelected(0)
  }, [graph, selected])

  useEffect(() => { setFocus(null) }, [mode])

  if (loading) return <div className="story-timeline-state"><span className="story-search-spinner" />Building the story graph…</div>
  if (error || (result && !result.success)) return <div className="story-timeline-state error">{error || result?.error}</div>
  if (!graph || graph.nodes.length === 0) {
    return (
      <div className="story-timeline-state">
        {result?.events.length
          ? 'No beats are plotted at this zoom level. Widen the zoom to include notable and minor beats.'
          : 'No timeline events are indexed yet. Let the on-open story analysis finish, then return here.'}
      </div>
    )
  }

  const chapterWidth = CHAPTER_WIDTHS[zoom]
  const chapterOrder = new Map(graph.chapters.map((chapter, position) => [chapter.index, position]))
  const svgWidth = PAD_LEFT * 2 + Math.max(1, graph.chapters.length) * chapterWidth
  const svgHeight = LANE_TOP + Math.max(1, graph.lanes.length) * LANE_GAP + 12
  const lanePosition = new Map(graph.lanes.map((lane, position) => [lane.id, position]))
  const laneById = new Map(graph.lanes.map(lane => [lane.id, lane]))
  const xOf = (node: GraphNode) => nodeX(node, chapterWidth, chapterOrder)
  const yOf = (node: GraphNode) => laneY(lanePosition.get(node.laneId) ?? 0)

  const visible = (node: GraphNode) => {
    if (!focus) return true
    if (mode === 'threads') return node.laneId === focus
    return (node.event.character_ids ?? []).includes(focus)
  }

  // Beats directly connected to the selection. Deliberately one hop, not the
  // whole reachable component: derived links chain most beats together, so
  // growing transitively would just highlight the entire graph.
  const chain = new Set<number>([selected])
  for (const link of graph.links) {
    if (link.from === selected) chain.add(link.to)
    if (link.to === selected) chain.add(link.from)
  }

  function select(index: number) {
    setSelected(index)
    const element = scrollRef.current
    const node = graph?.nodes[index]
    if (!element || !node) return
    const x = nodeX(node, chapterWidth, chapterOrder)
    element.scrollTo({ left: Math.max(0, x - element.clientWidth / 2), behavior: 'smooth' })
  }

  const current = graph.nodes[selected] ?? graph.nodes[0]
  const currentLinks = graph.links
    .filter(link => link.from === selected || link.to === selected)
    .map(link => {
      const otherIndex = link.from === selected ? link.to : link.from
      return { index: otherIndex, forward: link.from === selected, reason: link.reason, node: graph.nodes[otherIndex] }
    })
    .filter(item => !!item.node)

  const hovered = hover !== null ? graph.nodes[hover] : null

  return (
    <section className="story-graph" aria-label="Story graph">
      <div className="story-graph-rail">
        <div className="story-graph-mode" role="group" aria-label="Lane mode">
          <button type="button" className={mode === 'threads' ? 'active' : ''} onClick={() => setMode('threads')}>Threads</button>
          <button type="button" className={mode === 'characters' ? 'active' : ''} onClick={() => setMode('characters')}>Characters</button>
        </div>

        <span className="story-graph-rail-label">{mode === 'threads' ? 'Plot threads' : 'Characters'}</span>
        <div className="story-graph-legend">
          {graph.lanes.map(lane => (
            <button
              type="button"
              key={lane.id}
              className={focus === lane.id ? 'active' : ''}
              style={{ borderLeftColor: focus === lane.id ? lane.color : 'transparent', opacity: focus && focus !== lane.id ? 0.45 : 1 }}
              onClick={() => setFocus(focus === lane.id ? null : lane.id)}
              title={`${lane.name} · ${lane.eventCount} ${lane.eventCount === 1 ? 'beat' : 'beats'}`}
            >
              <i className={mode === 'threads' ? 'bar' : 'dot'} style={{ background: lane.color }} />
              <span>{lane.name}</span>
              <small>{chapterRange(graph.chapters, lane.from, lane.to)}</small>
            </button>
          ))}
        </div>

        {/* The codex button belongs to character lanes: once you are reading the
            graph by character, the cast view is the natural next stop. */}
        {mode === 'characters' && (
          <button type="button" className="story-graph-codex" onClick={onOpenCodex} title="Open the character codex">
            <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.25">
              <circle cx="4.5" cy="3.5" r="2" />
              <path d="M1.5 10.5v-1a3 3 0 0 1 3-3h.5a3 3 0 0 1 3 3v1" />
              <path d="M8 1.8a2 2 0 0 1 0 3.4" />
              <path d="M10.5 10.5v-1a3 3 0 0 0-2-2.8" />
            </svg>
            Open Character Codex
          </button>
        )}

        <div className="story-graph-rail-foot">
          <div className="story-graph-zoom">
            <button type="button" onClick={() => setZoom(value => Math.max(0, value - 1))} disabled={zoom === 0} aria-label="Show fewer beats">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M5 12h14" /></svg>
            </button>
            <button type="button" onClick={() => setZoom(value => Math.min(2, value + 1))} disabled={zoom === 2} aria-label="Show more beats">
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M5 12h14M12 5v14" /></svg>
            </button>
            <span>{ZOOM_LABELS[zoom]}</span>
          </div>
          <span className="story-graph-counts">
            {graph.nodes.length} of {graph.totalEvents} {graph.totalEvents === 1 ? 'beat' : 'beats'} · {graph.chapters.length} {graph.chapters.length === 1 ? 'chapter' : 'chapters'}
          </span>
        </div>
      </div>

      <div className="story-graph-canvas" ref={scrollRef}>
        <div className="story-graph-stage" style={{ width: svgWidth }}>
          {hovered && (
            <div
              className="story-graph-tip"
              style={{
                left: Math.max(6, Math.min(xOf(hovered) - 130, svgWidth - 266)),
                top: yOf(hovered) + 44,
              }}
            >
              <strong style={{ color: typeColor(hovered.event.primary_type) }}>
                {eventTypeLabel(hovered.event.primary_type)} · {hovered.event.chapter_title}
              </strong>
              <span>“{truncate(hovered.event.text, 110)}”</span>
            </div>
          )}

          <div className="story-graph-chapters">
            {graph.chapters.map(chapter => (
              <div key={chapter.index} style={{ width: chapterWidth }} title={`${chapter.title}: ${chapter.count} plotted`}>
                <span>{chapter.title}</span>
                {chapter.count > 0 && <small>{chapter.count}</small>}
              </div>
            ))}
          </div>

          <svg width={svgWidth} height={svgHeight} className="story-graph-svg" role="presentation">
            {graph.chapters.map((chapter, position) => (
              <line
                key={`grid-${chapter.index}`}
                x1={PAD_LEFT + position * chapterWidth}
                y1={0}
                x2={PAD_LEFT + position * chapterWidth}
                y2={svgHeight}
                className="story-graph-grid"
              />
            ))}
            <line x1={PAD_LEFT + graph.chapters.length * chapterWidth} y1={0} x2={PAD_LEFT + graph.chapters.length * chapterWidth} y2={svgHeight} className="story-graph-grid" />

            {graph.lanes.map((lane, position) => {
              const from = chapterOrder.get(lane.from) ?? 0
              const to = chapterOrder.get(lane.to) ?? from
              return (
                <line
                  key={`lane-${lane.id}`}
                  x1={PAD_LEFT + from * chapterWidth + 10}
                  y1={laneY(position)}
                  x2={PAD_LEFT + (to + 1) * chapterWidth - 10}
                  y2={laneY(position)}
                  stroke={lane.color}
                  strokeWidth={2}
                  strokeLinecap="round"
                  opacity={focus && focus !== lane.id ? 0.12 : 0.8}
                />
              )
            })}

            {graph.links.map((link, index) => {
              const from = graph.nodes[link.from]
              const to = graph.nodes[link.to]
              if (!from || !to) return null
              const x1 = xOf(from), y1 = yOf(from), x2 = xOf(to), y2 = yOf(to)
              const hot = chain.has(link.from) && chain.has(link.to)
              const dim = !visible(from) || !visible(to)
              return (
                <path
                  key={`link-${index}`}
                  d={`M ${x1} ${y1 - 8} Q ${(x1 + x2) / 2} ${Math.min(y1, y2) - 26} ${x2} ${y2 - 8}`}
                  className={`story-graph-link ${hot ? 'hot' : ''}`}
                  opacity={dim ? 0.08 : hot ? 0.95 : 0.35}
                />
              )
            })}

            {/* In character mode a beat's supporting cast is drawn as a short
                stub back to their own rail, so a shared scene reads as shared. */}
            {mode === 'characters' && graph.nodes.map(node => {
              const x = xOf(node), y = yOf(node)
              return (node.event.character_ids ?? []).slice(1).map(id => {
                const position = lanePosition.get(id)
                if (position === undefined) return null
                const otherY = laneY(position)
                const lane = laneById.get(id)
                return (
                  <path
                    key={`connector-${node.event.id}-${id}`}
                    d={`M ${x - 26} ${otherY} C ${x - 10} ${otherY}, ${x} ${otherY}, ${x} ${y}`}
                    fill="none"
                    stroke={lane?.color ?? '#868C96'}
                    strokeWidth={1.4}
                    opacity={visible(node) ? 0.6 : 0.08}
                  />
                )
              })
            })}

            {graph.nodes.map(node => {
              const x = xOf(node), y = yOf(node)
              const lane = laneById.get(node.laneId)
              const isSelected = node.index === selected
              const inChain = chain.has(node.index)
              return (
                <g
                  key={node.event.id}
                  className="story-graph-node"
                  onClick={() => select(node.index)}
                  onMouseEnter={() => setHover(node.index)}
                  onMouseLeave={() => setHover(null)}
                  role="button"
                  tabIndex={0}
                  aria-label={`${eventTypeLabel(node.event.primary_type)} in ${node.event.chapter_title}`}
                  onKeyDown={keyEvent => { if (keyEvent.key === 'Enter' || keyEvent.key === ' ') { keyEvent.preventDefault(); select(node.index) } }}
                >
                  <circle cx={x} cy={y} r={13} fill="transparent" />
                  {(isSelected || inChain) && (
                    <circle cx={x} cy={y} r={node.major ? 11 : 9.5} className="story-graph-halo" opacity={isSelected ? 1 : 0.4} />
                  )}
                  <circle
                    cx={x}
                    cy={y}
                    r={node.major ? 7 : 5.5}
                    fill={lane?.color ?? '#868C96'}
                    stroke={isSelected ? 'var(--app-accent-text)' : node.major ? typeColor(node.event.primary_type) : 'var(--bg-editor)'}
                    strokeWidth={2}
                    opacity={visible(node) ? 1 : 0.15}
                  />
                </g>
              )
            })}
          </svg>
        </div>
      </div>

      <div className="story-graph-inspector">
        <div className="story-graph-inspector-head">
          <span className="story-graph-type" style={{ borderColor: typeColor(current.event.primary_type), color: typeColor(current.event.primary_type) }}>
            {eventTypeLabel(current.event.primary_type)}
          </span>
          <small>Beat {selected + 1} of {graph.nodes.length}</small>
        </div>
        <div className="story-graph-inspector-loc">
          {current.event.chapter_title}
          {mode === 'threads' && laneById.get(current.laneId) ? ` · ${laneById.get(current.laneId)!.name}` : ''}
        </div>
        <blockquote className="story-graph-quote">“{current.event.text}”</blockquote>
        {current.event.time_kind !== 'manuscript' && (
          <div className="story-graph-time">{current.event.time_label}</div>
        )}
        {!!current.event.character_names?.length && (
          <div className="story-graph-chips">
            {current.event.character_names.map((name, position) => {
              const id = current.event.character_ids?.[position]
              const lane = id ? laneById.get(id) : undefined
              return (
                <span key={`${name}-${position}`}>
                  <i style={{ background: lane?.color ?? '#868C96' }} />
                  {name}
                </span>
              )
            })}
          </div>
        )}
        <div className="story-graph-links">
          {currentLinks.length > 0 && <span className="story-graph-rail-label">Connected beats</span>}
          {currentLinks.map(item => (
            <button type="button" key={item.index} onClick={() => select(item.index)} title={item.reason}>
              <b className={item.forward ? 'forward' : 'back'}>{item.forward ? '→' : '←'}</b>
              <span>{item.node.event.chapter_title} · {truncate(item.node.event.text, 70)}</span>
            </button>
          ))}
        </div>
        <div className="story-graph-inspector-foot">
          <button type="button" onClick={() => select((selected - 1 + graph.nodes.length) % graph.nodes.length)} aria-label="Previous beat">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M15 5l-7 7 7 7" /></svg>
          </button>
          <button type="button" onClick={() => select((selected + 1) % graph.nodes.length)} aria-label="Next beat">
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 5l7 7-7 7" /></svg>
          </button>
          <span />
          <button
            type="button"
            className="story-graph-open"
            onClick={() => onNavigate(current.event.section as Section, current.event.section_index, current.event.source_text)}
          >
            Open source →
          </button>
        </div>
      </div>
    </section>
  )
}

function chapterRange(chapters: { index: number; title: string }[], from: number, to: number): string {
  const short = (index: number) => {
    const match = chapters.find(chapter => chapter.index === index)
    const digits = match?.title.match(/\d+/)
    return digits ? digits[0] : String(index + 1)
  }
  return from === to ? `Ch ${short(from)}` : `Ch ${short(from)}–${short(to)}`
}

function eventTypeLabel(value: string): string {
  return value.replace(/_/g, ' ').replace(/^./, letter => letter.toUpperCase())
}

function truncate(value: string, limit: number): string {
  return value.length > limit ? `${value.slice(0, limit - 1)}…` : value
}
