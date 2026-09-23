import type { Section } from '../types/draftline'

// Taking the writer to a piece of text, rather than to the top of the chapter
// that contains it.
//
// Switching chapters alone leaves someone looking at a page and hunting for
// whatever they asked to see. The chapter find bar already knows how to locate
// a phrase and scroll it into view, so anything that wants to show a passage
// asks it to, and there is one implementation of "take me there" instead of one
// per caller.
//
// The event is deferred a tick because the find bar belongs to the chapter
// being opened: it has to exist, and be showing that chapter, before it can be
// asked to search it.
export const FIND_IN_MANUSCRIPT_EVENT = 'draftline:find-story-evidence'

export function findInManuscript(query: string): void {
  const text = query.trim()
  if (!text) return
  window.setTimeout(() => {
    window.dispatchEvent(new CustomEvent(FIND_IN_MANUSCRIPT_EVENT, { detail: { query: text } }))
  }, 0)
}

export interface ManuscriptDestination {
  section: Section
  index: number
  /** The text to find once the chapter is open. Optional: with none, the jump is to the chapter. */
  find?: string
}

/**
 * Show a passage: leave whatever workspace is open, go to its chapter, and
 * find it there.
 */
export function goToManuscript(
  destination: ManuscriptDestination,
  deps: {
    setViewMode: (mode: 'editor' | 'cast' | 'planner') => void
    setCurrentChapter: (section: Section, index: number) => void
  },
): void {
  deps.setViewMode('editor')
  deps.setCurrentChapter(destination.section, destination.index)
  if (destination.find) findInManuscript(destination.find)
}
