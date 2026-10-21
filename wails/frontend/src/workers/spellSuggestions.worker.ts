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
  language?: string
}

interface InitRequest {
  type: 'init'
  language?: string
}

let indexLanguage = ''
let indexPromise: Promise<WordBuckets | null> | null = null
const cache = new Map<string, string[]>()

function loadIndex(language: string): Promise<WordBuckets | null> {
  if (indexPromise && language === indexLanguage) return indexPromise
  indexLanguage = language
  cache.clear()
  indexPromise = (async () => {
    try {
      const [affResponse, dicResponse] = await Promise.all([
        fetch(`/dictionaries/${language}.aff`),
        fetch(`/dictionaries/${language}.dic`),
      ])
      if (!affResponse.ok || !dicResponse.ok) return null

      const dictionary = new Typo(
        language,
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

async function findSuggestions(word: string, limit: number, language: string): Promise<string[]> {
  const normalized = word.toLocaleLowerCase()
  const buckets = await loadIndex(language)
  // The cache is per language: loadIndex clears it on a switch, and a request
  // that was overtaken by a switch mid-await must not write stale suggestions
  // into the new language's cache.
  if (language !== indexLanguage) return []
  const cached = cache.get(normalized)
  if (cached) return cached.slice(0, limit)
  if (!buckets) return []

  const suggestions = findIndexedSuggestions(buckets, normalized, Math.max(limit, 8))
  cache.set(normalized, suggestions)
  return suggestions.slice(0, limit)
}

self.onmessage = async (event: MessageEvent<SuggestionRequest | InitRequest>) => {
  if (event.data.type === 'init') {
    // The creating thread sends init immediately with the writer's language,
    // so the index is built during application startup instead of making the
    // first context-menu request pay dictionary parsing and indexing costs.
    await loadIndex(event.data.language ?? 'en_US')
    self.postMessage({ type: 'ready' })
    return
  }

  const { id, word, limit, language } = event.data
  self.postMessage({
    type: 'suggestions',
    id,
    suggestions: await findSuggestions(word, limit, language ?? 'en_US'),
  })
}
