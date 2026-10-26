// The author's rulings on analysis findings, lifted out of bookStore so the
// store keeps its size. Both actions follow the same shape as chapters.ts:
// read the book, build the next one, hand it back through the host, which
// owns marking the store dirty and arming the autosave. Decisions are keyed
// by hashes of the surrounding prose, so they lapse when the text changes.

import type { BookData } from '../types/draftline'

export interface DecisionActions {
  setContinuityDecision: (signalID: string, status: 'reviewed' | 'dismissed' | null) => void
  setStyleDecision: (findingID: string, status: 'dismissed' | null) => void
}

export interface DecisionHost {
  read: () => BookData | null
  apply: (book: BookData) => void
}

export function createDecisionActions(host: DecisionHost): DecisionActions {
  return {
    setContinuityDecision: (signalID, status) => {
      const book = host.read()
      if (!book) return
      const existing = book.analysis?.continuity?.decisions ?? []
      const without = existing.filter(decision => decision.signal_id !== signalID)
      const decisions = status
        ? [...without, { signal_id: signalID, status, decided_at: new Date().toISOString() }]
        : without
      host.apply({
        ...book,
        analysis: { ...book.analysis, continuity: { ...book.analysis?.continuity, decisions } },
      })
    },

    setStyleDecision: (findingID, status) => {
      const book = host.read()
      if (!book) return
      const existing = book.analysis?.style?.decisions ?? []
      const without = existing.filter(decision => decision.finding_id !== findingID)
      const decisions = status
        ? [...without, { finding_id: findingID, status, decided_at: new Date().toISOString() }]
        : without
      host.apply({
        ...book,
        analysis: { ...book.analysis, style: { ...book.analysis?.style, decisions } },
      })
    },
  }
}
