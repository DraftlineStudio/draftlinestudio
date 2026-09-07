// Review desk — intentionally disconnected while the manuscript-memory
// engine (v5) is rebuilt. Reviewable detections return as continuity
// inspections with scope awareness.

import { useEffect } from 'react'
import type { BookData, Section } from '../../types/draftline'

interface ReviewDeskPanelProps {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
  onCount: (undecided: number) => void
  onOpenThreads: () => void
}

export default function ReviewDeskPanel({ onCount }: ReviewDeskPanelProps) {
  useEffect(() => {
    onCount(0)
  }, [onCount])
  return (
    <div className="smap-empty">
      <div className="smap-empty-title">Worth Reviewing is being rebuilt</div>
      <div className="smap-empty-hint">
        Detections return as scope-aware continuity inspections on the new
        narrative engine. The Continuity tab remains available meanwhile.
      </div>
    </div>
  )
}
