import { useEffect, useMemo, useRef, useState } from 'react'
import { UpdateStoryAuthorModel } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import { useBookStore } from '../../store/bookStore'
import type { BookData, EvidenceRecord, Section } from '../../types/draftline'
import {
  appendStoryDayCorrection,
  buildStoryMapLayout,
  describeEvent,
  type EraKind,
  type MapNode,
} from './storyMapModel'

interface Props {
  book: BookData
  onNavigate: (section: Section, index: number, query: string) => void
}

/** Node fill by era kind — section tokens, matching the era chip colours. */
function nodeFill(kind: EraKind): string {
  if (kind === 'past' || kind === 'memory' || kind === 'dream' || kind === 'simulation') {
    return 'var(--section-back)'
  }
  return 'var(--section-front)'
}

export default function StoryMapPanel({ book, onNavigate }: Props) {
  const updateBook = useBookStore(state => state.updateBook)
  const scrollRef = useRef<HTMLDivElement>(null)

  // The generated Go types carry the fingerprint; the local AnalysisData
  // interface has not caught up yet, so narrow structurally.
  const fingerprint = (book.analysis as { fingerprint?: types.StoryFingerprint } | undefined)?.fingerprint
  const records = book.analysis?.evidence?.records

  const layout = useMemo(
    () => (fingerprint ? buildStoryMapLayout(fingerprint) : null),
    [fingerprint],
  )

  const evidenceById = useMemo(() => {
    const map = new Map<string, EvidenceRecord>()
    for (const record of records ?? []) map.set(record.id, record)
    return map
  }, [records])

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [dismissedAnchors, setDismissedAnchors] = useState<Set<string>>(new Set())
  const [anchorBusy, setAnchorBusy] = useState(false)
  const [anchorError, setAnchorError] = useState('')

  // Keep a valid selection when the fingerprint rebuilds under us.
  const nodes = layout?.nodes ?? []
  const selectedIndex = selectedId ? nodes.findIndex(node => node.id === selectedId) : -1
  const selected = selectedIndex >= 0 ? nodes[selectedIndex] : nodes[0] ?? null
  const selectedPos = selected ? nodes.indexOf(selected) : -1

  const detail = useMemo(
    () => (fingerprint && selected ? describeEvent(fingerprint, evidenceById, selected.id) : null),
    [fingerprint, evidenceById, selected],
  )

  useEffect(() => { setAnchorError('') }, [selected?.id])

  const select = (node: MapNode) => {
    setSelectedId(node.id)
    const el = scrollRef.current
    if (el) el.scrollTo({ left: Math.max(0, node.x - el.clientWidth / 2), behavior: 'smooth' })
  }

  const step = (delta: number) => {
    if (nodes.length === 0 || selectedPos < 0) return
    select(nodes[(selectedPos + delta + nodes.length) % nodes.length])
  }

  const scrollToEra = (x: number) => {
    scrollRef.current?.scrollTo({ left: Math.max(0, x - 60), behavior: 'smooth' })
  }

  const openInChapter = () => {
    if (!detail?.navigation) return
    onNavigate(detail.navigation.section as Section, detail.navigation.sectionIndex, detail.navigation.query)
  }

  const confirmPlacement = async () => {
    if (!fingerprint || !detail || anchorBusy) return
    const draft = appendStoryDayCorrection(fingerprint, detail.id)
    if (!draft) return
    setAnchorBusy(true)
    setAnchorError('')
    try {
      const result = await UpdateStoryAuthorModel(
        book as types.BookData,
        types.StoryAuthorModel.createFrom(draft.model),
      )
      if (result.success && result.book) {
        updateBook(result.book as unknown as BookData)
      } else {
        setAnchorError(result.error || 'Could not save the placement.')
      }
    } catch (error) {
      setAnchorError(String(error))
    } finally {
      setAnchorBusy(false)
    }
  }

  if (!fingerprint) {
    return (
      <div className="smap-empty">
        <strong>No story fingerprint yet.</strong>
        <span>Draftline builds the story fingerprint after 15 seconds of writing inactivity.</span>
      </div>
    )
  }

  if (!layout || layout.nodes.length === 0) {
    return (
      <div className="smap-empty">
        <strong>Nothing to map yet.</strong>
        <span>The fingerprint has no placeable events for this manuscript.</span>
      </div>
    )
  }

  const showAnchor = !!detail?.needsAnchor && !dismissedAnchors.has(detail.id)
  const tickBottom = layout.height - 40

  return (
    <div className="smap-root">
      <div className="smap-scroll" ref={scrollRef}>
        <div className="smap-inner" style={{ width: layout.width }}>
          <div className="smap-eras">
            {layout.eras.map(era => (
              <button
                type="button"
                key={era.id}
                className={`smap-era smap-era--${era.kind}${era.anomaly ? ' smap-era--anomaly' : ''}`}
                style={{ width: era.width - ERA_CHIP_TRIM }}
                onClick={() => scrollToEra(era.x)}
                title={`${era.label} · ${era.sub}`}
              >
                <span className="smap-era-name">{era.label}</span>
                <span className="smap-era-sub">{era.sub}</span>
              </button>
            ))}
          </div>

          <svg width={layout.width - 16} height={layout.height} className="smap-svg">
            {/* Simulation frames: dashed outline spanning the era band. */}
            {layout.eras.filter(era => era.kind === 'simulation').map(era => (
              <g key={`sim-${era.id}`}>
                <rect
                  x={era.x - 4}
                  y={20}
                  width={era.width + 8}
                  height={layout.height - 72}
                  rx={5}
                  className="smap-sim-frame"
                />
                <text x={era.x + 4} y={32} className="smap-sim-label">{era.label}</text>
              </g>
            ))}

            {layout.ticks.map(tick => (
              <g key={`tick-${tick.x}`}>
                <line x1={tick.x} y1={16} x2={tick.x} y2={tickBottom} className="smap-tick" />
                <text x={tick.x + 6} y={14} className="smap-tick-label">{tick.label}</text>
              </g>
            ))}

            {layout.checkpoints.map(checkpoint => (
              <g key={checkpoint.id}>
                <line x1={checkpoint.x} y1={22} x2={checkpoint.x} y2={layout.height - 44} className="smap-cp-line" />
                <path
                  d={`M ${checkpoint.x} 22 L ${checkpoint.x} 40 M ${checkpoint.x} 23 L ${checkpoint.x + 8} 26.5 L ${checkpoint.x} 30 Z`}
                  className="smap-cp-flag"
                />
                <text x={checkpoint.x + 8} y={30} className="smap-cp-label">
                  {`CP · ${checkpoint.title}${checkpoint.ratio ? ` · ${checkpoint.ratio}` : ''}`}
                </text>
              </g>
            ))}

            {layout.loop && (
              <g>
                <path d={layout.loop.d} className="smap-loop" />
                <text x={layout.loop.labelX} y={layout.loop.labelY} className="smap-loop-label">
                  {layout.loop.label}
                </text>
              </g>
            )}

            <path d={layout.pathD} className="smap-path" />

            {layout.dangles.map(dangle => (
              <g key={dangle.threadId}>
                <path d={dangle.d} className="smap-dangle" />
                <text x={dangle.tx} y={dangle.ty} className="smap-dangle-label">{dangle.label}</text>
              </g>
            ))}

            {layout.nodes.map(node => {
              const isSelected = selected?.id === node.id
              return (
                <g key={node.id} className="smap-node" onClick={() => select(node)}>
                  <circle cx={node.x} cy={node.y} r={13} fill="transparent" />
                  {isSelected && (
                    <circle cx={node.x} cy={node.y} r={node.r + 4} className="smap-node-halo" />
                  )}
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={node.r}
                    fill={nodeFill(node.kind)}
                    className={`smap-node-dot${node.dashed ? ' smap-node-dot--uncertain' : ''}${node.major ? ' smap-node-dot--major' : ''}`}
                  />
                  <text x={node.x} y={node.y + 3} textAnchor="middle" className="smap-node-ord">{node.ord}</text>
                  <text
                    x={node.x}
                    y={node.y + node.r + 13}
                    textAnchor="middle"
                    className={`smap-node-label${isSelected ? ' smap-node-label--selected' : ''}`}
                  >
                    {node.label}
                  </text>
                </g>
              )
            })}
          </svg>

          {layout.hiddenCount > 0 && (
            <div className="smap-overflow-note">
              +{layout.hiddenCount} more events · showing the {layout.nodes.length} most important
            </div>
          )}
        </div>
      </div>

      <div className="smap-detail">
        {detail ? (
          <>
            <div className="smap-detail-head">
              <span className={`smap-chip smap-chip--${detail.kind}`}>{detail.kindChip}</span>
              <span className="smap-detail-spacer" />
              <span className="smap-when">{detail.when}</span>
            </div>
            <div className="smap-title">{detail.title}</div>
            <div className="smap-loc">{detail.location}</div>
            {detail.quote && <div className="smap-quote">&ldquo;{detail.quote}&rdquo;</div>}

            {showAnchor && (
              <div className="smap-anchor">
                <div className="smap-anchor-head">
                  Inferred placement · {detail.anchorPercent}%
                  {detail.anchorLabel ? ` · “${detail.anchorLabel}”` : ''}
                </div>
                <div className="smap-anchor-actions">
                  <button
                    type="button"
                    className="smap-btn-confirm"
                    disabled={anchorBusy}
                    onClick={() => { void confirmPlacement() }}
                  >
                    {anchorBusy ? 'Saving…' : `Confirm Day ${detail.anchorDay}`}
                  </button>
                  <button
                    type="button"
                    className="smap-btn-ghost"
                    disabled={anchorBusy}
                    onClick={() => setDismissedAnchors(prev => new Set(prev).add(detail.id))}
                  >
                    Leave floating
                  </button>
                </div>
                {anchorError && <div className="smap-anchor-error">{anchorError}</div>}
              </div>
            )}

            <div className="smap-detail-body">
              {detail.stateChanges.length > 0 && (
                <>
                  <span className="smap-label">State changes</span>
                  {detail.stateChanges.map(line => (
                    <div className="smap-state-line" key={line}><i>•</i><span>{line}</span></div>
                  ))}
                </>
              )}
              {detail.obligations.length > 0 && (
                <>
                  <span className="smap-label">Obligations</span>
                  {detail.obligations.map(obligation => (
                    <div className={`smap-obl-line smap-obl-line--${obligation.tone}`} key={`${obligation.symbol}-${obligation.text}`}>
                      <i>{obligation.symbol}</i><span>{obligation.text}</span>
                    </div>
                  ))}
                </>
              )}
            </div>

            {detail.chips.length > 0 && (
              <div className="smap-chips">
                {detail.chips.map(chip => <span className="smap-chip-tag" key={chip}>{chip}</span>)}
              </div>
            )}

            <div className="smap-detail-foot">
              <button type="button" className="smap-nav-btn" onClick={() => step(-1)} aria-label="Previous event">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M15 5l-7 7 7 7" /></svg>
              </button>
              <button type="button" className="smap-nav-btn" onClick={() => step(1)} aria-label="Next event">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 5l7 7-7 7" /></svg>
              </button>
              <span className="smap-detail-spacer" />
              <button
                type="button"
                className="smap-open-link"
                onClick={openInChapter}
                disabled={!detail.navigation}
              >
                Open in {detail.chapterLabel} →
              </button>
            </div>
          </>
        ) : (
          <div className="smap-detail-empty">Select an event on the map.</div>
        )}
      </div>
    </div>
  )
}

/** Era chips render 3px narrower than their band, like the mock. */
const ERA_CHIP_TRIM = 3
