// The character pipeline's store actions, lifted out of bookStore so the store
// keeps its size.
//
// All four hand a book to Go and take a rewritten one back. They share a shape
// because they share a hazard: the backend returns a whole book, so a result
// that arrives after the writer has moved on must not be written over what is
// on screen now. The host owns that decision, along with dirtying and the
// autosave, exactly as it does for chapter mutations.

import { IndexBook, MergeEntities, SplitEntity } from '../../wailsjs/go/main/App'
import type { BookData } from '../types/draftline'

export interface CharacterIndexActions {
  indexBook: () => Promise<void>
  mergeEntities: (entityIds: string[], canonical: string) => Promise<boolean>
  splitEntity: (entityId: string, mentionIds: string[], newCanonical: string) => Promise<boolean>
  clearAllCharacters: () => void
}

export interface CharacterIndexHost {
  getBook: () => BookData | null
  /** Replace the book and mark it dirty; `prose` bumps the analysis revision. */
  commit: (book: BookData, options?: { prose?: boolean }) => void
  setIndexing: (indexing: boolean) => void
  setStatus: (message: string) => void
  autosave: () => void
}

export function createCharacterIndexActions(host: CharacterIndexHost): CharacterIndexActions {
  // Merge and split differ only in the call they make and the words they use.
  const reconcile = async (
    run: (book: BookData) => Promise<{ success: boolean; book?: unknown; error?: string }>,
    done: string,
    failed: string,
    errored: string,
  ): Promise<boolean> => {
    const book = host.getBook()
    if (!book) return false
    try {
      const result = await run(book)
      if (result.success && result.book) {
        host.commit(result.book as BookData, { prose: true })
        host.setStatus(done)
        host.autosave()
        return true
      }
      host.setStatus(result.error || failed)
      return false
    } catch (e) {
      host.setStatus(`${errored}: ${e}`)
      return false
    }
  }

  return {
    indexBook: async () => {
      const book = host.getBook()
      if (!book) return
      host.setIndexing(true)
      host.setStatus('Indexing characters...')
      try {
        const result = await IndexBook(book as any)
        if (result.success && result.book) {
          host.commit(result.book as unknown as BookData, { prose: true })
          host.setIndexing(false)
          host.setStatus(`Found ${result.characters_found} characters (${result.new_characters} new)`)
          host.autosave()
        } else {
          host.setIndexing(false)
          host.setStatus(result.error || 'Indexing failed')
        }
      } catch (e) {
        host.setIndexing(false)
        host.setStatus(`Indexing error: ${e}`)
      }
    },

    mergeEntities: (entityIds, canonical) =>
      reconcile(book => MergeEntities(book as any, entityIds, canonical), 'Characters merged', 'Merge failed', 'Merge error'),

    splitEntity: (entityId, mentionIds, newCanonical) =>
      reconcile(book => SplitEntity(book as any, entityId, mentionIds, newCanonical), 'Character split', 'Split failed', 'Split error'),

    // Purge every character (including fossils from older versions of the
    // detector) plus all entity/relationship data, for a clean re-index.
    clearAllCharacters: () => {
      const book = host.getBook()
      if (!book) return
      host.commit({
        ...book,
        story_bible: {
          ...book.story_bible,
          characters: [],
          plot_notes: book.story_bible?.plot_notes || '',
          timeline: book.story_bible?.timeline || '',
        },
        analysis: { ...book.analysis, entity_resolution: undefined, relationships: undefined, evidence: undefined },
      })
      host.setStatus('All characters cleared — re-index to detect them fresh')
      host.autosave()
    },
  }
}
