// Story Graph panel — intentionally disconnected while the manuscript-memory
// engine (v5) is rebuilt. The graph will be driven by narrative developments,
// not raw fingerprints, once that layer is trustworthy.

import type { BookData, Section } from '../../types/draftline'

interface StoryMapPanelProps {
  book: BookData
  onNavigate: (section: Section, sectionIndex: number, evidenceQuery: string) => void
}

export default function StoryMapPanel(_props: StoryMapPanelProps) {
  return (
    <div className="smap-empty">
      <div className="smap-empty-title">The Story Graph is being rebuilt</div>
      <div className="smap-empty-hint">
        Draftline's narrative engine is moving to typed, source-grounded manuscript
        memory. The graph returns when it can be built from narrative developments
        instead of raw sentence fingerprints.
      </div>
    </div>
  )
}
