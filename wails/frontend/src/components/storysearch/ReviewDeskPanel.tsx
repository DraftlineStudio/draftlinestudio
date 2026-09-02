import { useEffect, useMemo, useState } from 'react'
import { UpdateStoryAuthorModel } from '../../../wailsjs/go/main/App'
import { types } from '../../../wailsjs/go/models'
import { useBookStore } from '../../store/bookStore'
import type { BookData, EvidenceRecord, Section } from '../../types/draftline'
import {
  buildReviewCards,
  cardActions,
  layoutReviewCards,
  withLoopContext,
  withoutCorrection,
  type DecisionStatus,
  type ReviewActionKind,
  type ReviewCard,
} from './reviewDeskModel'

interface Props {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
  /** Reports the number of undecided detections so the shell can badge the tab. */
  onCount?: (undecided: number) => void
  /** Wired by the shell to switch to the Threads tab ("View thread"). */
  onOpenThreads?: () => void
}

interface CardRef {
  key: string
  label: string
  section: Section
  index: number
  query: string
  quote?: string
}

export default function ReviewDeskPanel({ book, onNavigate, onCount, onOpenThreads }: Props) {
  const setDecision = useBookStore(state => state.setContinuityDecision)
  const updateBook = useBookStore(state => state.updateBook)
  const [busyCard, setBusyCard] = useState<string | null>(null)
  const [error, setError] = useState('')

  const fingerprint = book.analysis?.fingerprint
  const decisions = book.analysis?.continuity?.decisions

  const evidenceById = useMemo(() => {
    const map = new Map<string, EvidenceRecord>()
    for (const record of book.analysis?.evidence?.records ?? []) map.set(record.id, record)
    return map
  }, [book.analysis?.evidence])

  const cards = useMemo(
    () => buildReviewCards(fingerprint, decisions, evidenceById),
    [fingerprint, decisions, evidenceById],
  )

  const refsByCard = useMemo(() => {
    const map = new Map<string, CardRef[]>()
    for (const card of cards) map.set(card.id, cardRefs(card, book, evidenceById))
    return map
  }, [cards, book, evidenceById])

  const undecided = useMemo(() => cards.filter(card => !card.decided).length, [cards])
  useEffect(() => {
    onCount?.(undecided)
    return () => onCount?.(0)
  }, [undecided, onCount])

  if (!fingerprint) {
    return (
      <div className="bbreview-state">
        Draftline builds the story fingerprint after 15 seconds of writing inactivity.
        Detections that need an author call will gather here.
      </div>
    )
  }
  if (cards.length === 0) {
    return <div className="bbreview-state">No detections — the fingerprint has nothing that needs a call.</div>
  }

  const decide = (card: ReviewCard, status: DecisionStatus | null) => {
    for (const id of card.decisionIds) setDecision(id, status)
  }

  const applyAuthorModel = async (card: ReviewCard, draft: ReturnType<typeof withLoopContext>['model']): Promise<boolean> => {
    setBusyCard(card.id)
    setError('')
    try {
      const result = await UpdateStoryAuthorModel(book as types.BookData, types.StoryAuthorModel.createFrom(draft))
      if (!result.success || !result.book) {
        setError(result.error || 'Could not update the story author model.')
        return false
      }
      updateBook(result.book as unknown as BookData)
      return true
    } catch (reason) {
      setError(String(reason))
      return false
    } finally {
      setBusyCard(null)
    }
  }

  const markReset = async (card: ReviewCard) => {
    const draft = withLoopContext(fingerprint.author_model, card.id)
    if (draft.added && !(await applyAuthorModel(card, draft.model))) return
    decide(card, 'reviewed')
  }

  const discardCorrection = async (card: ReviewCard) => {
    if (!card.correctionId) {
      decide(card, 'dismissed')
      return
    }
    const draft = withoutCorrection(fingerprint.author_model, card.correctionId)
    if (!draft.removed) {
      decide(card, 'dismissed')
      return
    }
    // Rebuild drops both the correction and its diagnostic, so no decision is needed.
    await applyAuthorModel(card, draft.model)
  }

  const runAction = (card: ReviewCard, action: ReviewActionKind) => {
    switch (action) {
      case 'goto': {
        const ref = refsByCard.get(card.id)?.[0]
        if (ref) onNavigate(ref.section, ref.index, ref.query)
        break
      }
      case 'review': decide(card, 'reviewed'); break
      case 'dismiss':
      case 'flag_duplicate':
      case 'intentional': decide(card, 'dismissed'); break
      case 'reopen': decide(card, null); break
      case 'reset': void markReset(card); break
      case 'discard_correction': void discardCorrection(card); break
      case 'open_threads': onOpenThreads?.(); break
      case 'reattach': break
    }
  }

  const renderCard = (card: ReviewCard, hero: boolean) => {
    const refs = refsByCard.get(card.id) ?? []
    return (
      <article
        key={card.id}
        className={`bbreview-card tone-${card.kindColor}${card.decided ? ' decided' : ''}${hero ? ' hero' : ''}`}
      >
        <div className={`bbreview-kind tone-${card.kindColor}`}>
          {card.kindLabel}
          {card.confidencePct !== undefined && ` · ${card.confidencePct}%`}
          {card.decided && ' · RESOLVED ✓'}
        </div>
        <div className="bbreview-title">{card.title}</div>
        {card.excerpt && <div className="bbreview-excerpt">{card.excerpt}</div>}
        <div className="bbreview-detail">{card.detail}</div>
        {card.kind === 'near_duplicate_chapter' && !card.decided && (
          <div className="bbreview-interp">
            Interpretations: accidental duplicate · <b>reset / loop</b> · repeated event · alternate account
          </div>
        )}
        {refs.length > 0 && (
          <div className="bbreview-refs">
            {refs.map(ref => (
              <button
                key={ref.key}
                type="button"
                className="bbreview-ref"
                title={ref.quote ? `“${ref.quote}”` : `Open ${ref.label}`}
                onClick={() => onNavigate(ref.section, ref.index, ref.query)}
              >
                {ref.label}
              </button>
            ))}
          </div>
        )}
        <div className="bbreview-actions">
          {cardActions(card).map(action => (
            <button
              key={action.action}
              type="button"
              className={`bbreview-btn ${action.tone}`}
              disabled={action.disabled || busyCard === card.id}
              title={action.title}
              onClick={() => runAction(card, action.action)}
            >
              {action.label}
            </button>
          ))}
        </div>
      </article>
    )
  }

  const layout = layoutReviewCards(cards)
  return (
    <section className="bbreview-panel" aria-label="Review detections">
      {error && <div className="bbreview-error">{error}</div>}
      <div className="bbreview-columns">
        <div className="bbreview-hero">
          {layout.hero
            ? renderCard(layout.hero, true)
            : <div className="bbreview-allclear">Every detection has a call — nothing waiting on you.</div>}
        </div>
        {layout.columns.map((column, index) => (
          <div className="bbreview-col" key={index}>
            {column.map(card => renderCard(card, false))}
          </div>
        ))}
      </div>
    </section>
  )
}

/**
 * Evidence-backed chapter references for a card; near-duplicate diagnostics
 * carry no evidence, so their chapter_indices (indexes into the concatenated
 * front matter + body + back matter list — see duplicateChapterDiagnostics in
 * wails/internal/fingerprint/diagnostics.go) are resolved against the book.
 */
function cardRefs(card: ReviewCard, book: BookData, evidenceById: ReadonlyMap<string, EvidenceRecord>): CardRef[] {
  const refs: CardRef[] = []
  const seen = new Set<string>()
  for (const evidenceId of card.evidenceIds) {
    const record = evidenceById.get(evidenceId)
    if (!record) continue
    const section = record.section as Section
    const positionKey = `${section}:${record.section_index}`
    if (seen.has(positionKey)) continue
    seen.add(positionKey)
    refs.push({
      key: evidenceId,
      label: chapterLabel(book, section, record.section_index),
      section,
      index: record.section_index,
      query: (record.text ?? '').slice(0, 80),
      quote: record.text,
    })
  }
  if (refs.length === 0) {
    for (const globalIndex of card.chapterIndices) {
      const resolved = resolveGlobalChapter(book, globalIndex)
      if (!resolved) continue
      const positionKey = `${resolved.section}:${resolved.index}`
      if (seen.has(positionKey)) continue
      seen.add(positionKey)
      refs.push({
        key: `chapter-${globalIndex}`,
        label: chapterLabel(book, resolved.section, resolved.index),
        section: resolved.section,
        index: resolved.index,
        query: '',
      })
    }
  }
  return refs
}

function resolveGlobalChapter(book: BookData, globalIndex: number): { section: Section; index: number } | null {
  const front = book.front_matter?.length ?? 0
  const body = book.body?.length ?? 0
  const back = book.back_matter?.length ?? 0
  if (globalIndex < 0) return null
  if (globalIndex < front) return { section: 'front_matter', index: globalIndex }
  if (globalIndex < front + body) return { section: 'body', index: globalIndex - front }
  if (globalIndex < front + body + back) return { section: 'back_matter', index: globalIndex - front - body }
  return null
}

function chapterLabel(book: BookData, section: Section, index: number): string {
  const list =
    section === 'front_matter' ? book.front_matter :
    section === 'body' ? book.body :
    section === 'back_matter' ? book.back_matter : undefined
  const title = list?.[index]?.title?.trim()
  if (title) return title
  if (section === 'body') return `Chapter ${index + 1}`
  return section === 'front_matter' ? `Front matter ${index + 1}` : `Back matter ${index + 1}`
}
