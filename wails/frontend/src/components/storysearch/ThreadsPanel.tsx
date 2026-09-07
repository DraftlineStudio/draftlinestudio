// Threads panel — intentionally disconnected while the manuscript-memory
// engine (v5) is rebuilt. Thread inference returns as a projection of
// narrative developments, not raw fingerprints.

import type { BookData, Section } from '../../types/draftline'

interface ThreadsPanelProps {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
}

export default function ThreadsPanel(_props: ThreadsPanelProps) {
  return (
    <div className="smap-empty">
      <div className="smap-empty-title">Story threads are being rebuilt</div>
      <div className="smap-empty-hint">
        Thread tracking returns on the new narrative engine, built from typed
        obligations and goals rather than keyword overlap.
      </div>
    </div>
  )
}
