import { useCallback, useMemo, useState } from 'react'
import { types } from '../../../wailsjs/go/models'
import type { AnalysisData, BookData, EvidenceRecord, Section } from '../../types/draftline'
import {
  buildThreadRows,
  type SegmentKind,
  type ThreadCategory,
  type ThreadRow,
} from './threadsModel'

type ChipFilter = 'all' | ThreadCategory

const SVG_WIDTH = 1260
const ROW_HEIGHT = 30
const ROW_TOP = 24
const PLOT_LEFT = 310
const PLOT_RIGHT = 1160
const NAME_MAX = 26

const SEGMENT_DASH: Record<SegmentKind, string | undefined> = {
  solid: undefined,
  gap: '2 5',
  open: '4 3',
  intentional: '1 5',
}

const SEGMENT_OPACITY: Record<SegmentKind, number> = {
  solid: 1,
  gap: 0.5,
  open: 1,
  intentional: 0.8,
}

function plotX(x: number): number {
  return PLOT_LEFT + Math.max(0, Math.min(1, x)) * (PLOT_RIGHT - PLOT_LEFT)
}

function shortName(name: string): string {
  return name.length > NAME_MAX ? `${name.slice(0, NAME_MAX - 1).trimEnd()}…` : name
}

export default function ThreadsPanel({ book, onNavigate }: {
  book: BookData
  onNavigate: (section: Section, index: number, query: string) => void
}) {
  const [filter, setFilter] = useState<ChipFilter>('all')

  // The generated Go bindings carry the fingerprint; the frontend AnalysisData
  // interface has not been widened yet, so narrow locally instead of `any`.
  const fingerprint = (book.analysis as (AnalysisData & { fingerprint?: types.StoryFingerprint }) | undefined)?.fingerprint

  const model = useMemo(
    () => (fingerprint ? buildThreadRows(fingerprint) : null),
    [fingerprint],
  )

  const eventById = useMemo(() => {
    const map = new Map<string, types.FingerprintEvent>()
    for (const event of fingerprint?.events ?? []) map.set(event.id, event)
    return map
  }, [fingerprint])

  const evidenceById = useMemo(() => {
    const map = new Map<string, EvidenceRecord>()
    for (const record of book.analysis?.evidence?.records ?? []) map.set(record.id, record)
    return map
  }, [book.analysis?.evidence])

  const openRow = useCallback((row: ThreadRow) => {
    const event = row.openingEventId ? eventById.get(row.openingEventId) : undefined
    if (!event) return
    for (const evidenceId of event.evidence_ids ?? []) {
      const record = evidenceById.get(evidenceId)
      if (record) {
        onNavigate(record.section as Section, record.section_index, (record.text ?? '').slice(0, 80))
        return
      }
    }
  }, [eventById, evidenceById, onNavigate])

  if (!fingerprint || !model) {
    return (
      <div className="bbthreads-panel">
        <div className="bbthreads-empty">
          Draftline builds the story fingerprint after 15 seconds of writing inactivity — thread tracking appears once it has run.
        </div>
      </div>
    )
  }

  if (model.rows.length === 0) {
    return (
      <div className="bbthreads-panel">
        <div className="bbthreads-empty">
          No story threads yet — open questions, threats, and promises will plot here as the fingerprint finds them.
        </div>
      </div>
    )
  }

  const chips: { id: ChipFilter; label: string; count: number }[] = [
    { id: 'all', label: 'All', count: model.counts.all },
    { id: 'open', label: 'Open', count: model.counts.open },
    { id: 'resolved', label: 'Resolved', count: model.counts.resolved },
    { id: 'dormant', label: 'Dormant', count: model.counts.dormant },
  ]

  const converge = model.converge
  const svgHeight = ROW_TOP + model.rows.length * ROW_HEIGHT

  return (
    <div className="bbthreads-panel">
      <div className="bbthreads-head">
        {chips.map(chip => (
          <button
            key={chip.id}
            type="button"
            className={`bbthreads-chip${filter === chip.id ? ' active' : ''}`}
            onClick={() => setFilter(chip.id)}
          >
            {chip.label} · {chip.count}
          </button>
        ))}
        <span className="bbthreads-caption">
          {model.basis === 'story_day' ? 'columns = story time' : 'columns = manuscript order (few time anchors yet)'}
          {' · click a row to jump to its opening scene'}
        </span>
      </div>
      <div className="bbthreads-scroll">
        <svg className="bbthreads-svg" width={SVG_WIDTH} height={svgHeight}>
          {model.rows.map((row, index) => {
            const y = ROW_TOP + index * ROW_HEIGHT
            const dimmed = filter !== 'all' && row.category !== filter
            return (
              <g
                key={row.id}
                className={`bbthreads-row${dimmed ? ' bbthreads-dim' : ''}`}
                role="button"
                tabIndex={0}
                aria-label={`${row.name} — ${row.stateLabel}`}
                onClick={() => openRow(row)}
                onKeyDown={event => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault()
                    openRow(row)
                  }
                }}
              >
                <title>{`${row.name} — ${row.stateLabel}`}</title>
                <rect className="bbthreads-rowbg" x={0} y={y - 14} width={SVG_WIDTH} height={28} rx={3} />
                <text className="bbthreads-name" x={12} y={y + 3.5}>{shortName(row.name)}</text>
                <text className="bbthreads-state" x={170} y={y + 3.5} fill={row.stateColor}>{row.stateLabel}</text>
                {row.segments.map((segment, segmentIndex) => (
                  <line
                    key={segmentIndex}
                    x1={plotX(segment.x1)}
                    y1={y}
                    x2={plotX(segment.x2)}
                    y2={y}
                    stroke={row.color}
                    strokeWidth={2}
                    strokeLinecap="round"
                    strokeDasharray={SEGMENT_DASH[segment.kind]}
                    opacity={SEGMENT_OPACITY[segment.kind]}
                  />
                ))}
                {row.plotted && <circle cx={plotX(row.x0)} cy={y} r={3} fill={row.color} />}
                {row.ring && <circle cx={plotX(row.ringX)} cy={y} r={4} fill="none" stroke={row.color} strokeWidth={1.5} />}
                {row.openEnd && <text className="bbthreads-openend" x={PLOT_RIGHT + 10} y={y + 3.5}>{'OPEN →'}</text>}
                {row.note && (
                  <text
                    className={`bbthreads-note${row.note.tone === 'error' ? ' error' : ''}`}
                    x={plotX(row.note.x)}
                    y={y + 14}
                  >
                    {row.note.text}
                  </text>
                )}
              </g>
            )
          })}
          {converge && (() => {
            const cx = plotX(converge.x)
            const yA = ROW_TOP + converge.fromRow * ROW_HEIGHT
            const yB = ROW_TOP + converge.toRow * ROW_HEIGHT
            return (
              <g className="bbthreads-converge">
                <path
                  className="bbthreads-converge-path"
                  d={`M ${cx} ${yA + 8} C ${cx + 11} ${yA + 16}, ${cx + 11} ${yB - 16}, ${cx} ${yB - 8}`}
                />
                <text className="bbthreads-converge-label" x={cx + 16} y={(yA + yB) / 2 + 3}>{converge.label}</text>
              </g>
            )
          })()}
        </svg>
      </div>
      {model.moreCount > 0 && (
        <div className="bbthreads-more">+{model.moreCount} more threads</div>
      )}
    </div>
  )
}
