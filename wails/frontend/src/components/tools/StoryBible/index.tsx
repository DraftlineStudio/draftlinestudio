// StoryBible - Plot Notes and Timeline (characters live in the Cast view)

import { useBookStore } from '../../../store/bookStore'

// ── Plot Notes ───────────────────────────────────────────────────────────────

export function PlotSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Plot Bible</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Outline acts, subplots, themes, and unresolved threads.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.plot_notes ?? ''}
        onChange={e => updateStoryBibleText('plot_notes', e.target.value)}
        placeholder={"Act 1:\n  Opening image...\n  Inciting incident...\n\nAct 2:\n  ...\n\nThemes:\n  ..."}
      />
    </div>
  )
}

// ── Timeline ─────────────────────────────────────────────────────────────────

export function TimelineSection() {
  const { book, updateStoryBibleText } = useBookStore()
  if (!book) return <div className="tool-empty-state">Open a project first.</div>
  return (
    <div>
      <div className="tool-label" style={{ marginBottom: 4 }}>Story Timeline</div>
      <p className="settings-hint" style={{ marginBottom: 8 }}>Chronological events in story time. One event per line.</p>
      <textarea
        className="bible-textarea"
        value={book.story_bible?.timeline ?? ''}
        onChange={e => updateStoryBibleText('timeline', e.target.value)}
        placeholder={"Day 1 — Marcus arrives in the city (Ch. 1)\nDay 1, evening — He meets Clara (Ch. 2)\nDay 3 — The letter arrives (Ch. 4)"}
      />
    </div>
  )
}
