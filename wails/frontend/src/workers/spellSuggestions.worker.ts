import Typo from 'typo-js'
import {
  buildWordBuckets,
  findIndexedSuggestions,
  type WordBuckets,
} from '../services/spellSuggestions'

interface SuggestionRequest {
  type?: 'suggest'
  id: number
  word: string
  limit: number
}

interface InitRequest {
  type: 'init'
}

let indexPromise: Promise<WordBuckets | null> | null = null
const cache = new Map<string, string[]>()

function loadIndex(): Promise<WordBuckets | null> {
  if (indexPromise) return indexPromise
  indexPromise = (async () => {
    try {
      const [affResponse, dicResponse] = await Promise.all([
        fetch('/dictionaries/en_US.aff'),
        fetch('/dictionaries/en_US.dic'),
      ])
      if (!affResponse.ok || !dicResponse.ok) return null

      const dictionary = new Typo(
        'en_US',
        await affResponse.text(),
        await dicResponse.text(),
      ) as Typo & { dictionaryTable: Record<string, unknown> }

      return buildWordBuckets(Object.keys(dictionary.dictionaryTable))
    } catch {
      return null
    }
  })()
  return indexPromise
}

async function findSuggestions(word: string, limit: number): Promise<string[]> {
  const normalized = word.toLocaleLowerCase()
  const cached = cache.get(normalized)
  if (cached) return cached.slice(0, limit)

  const buckets = await loadIndex()
  if (!buckets) return []

  const suggestions = findIndexedSuggestions(buckets, normalized, Math.max(limit, 8))
  cache.set(normalized, suggestions)
  return suggestions.slice(0, limit)
}

self.onmessage = async (event: MessageEvent<SuggestionRequest | InitRequest>) => {
  if (event.data.type === 'init') {
    await loadIndex()
    self.postMessage({ type: 'ready' })
    return
  }

  const { id, word, limit } = event.data
  self.postMessage({
    type: 'suggestions',
    id,
    suggestions: await findSuggestions(word, limit),
  })
}

// Build the compact lookup index during application startup instead of making
// the first context-menu request pay dictionary parsing and indexing costs.
void loadIndex().then(() => self.postMessage({ type: 'ready' }))
