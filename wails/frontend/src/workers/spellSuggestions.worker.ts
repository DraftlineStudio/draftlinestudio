import Typo from 'typo-js'

interface SuggestionRequest {
  id: number
  word: string
  limit: number
}

let dictionaryPromise: Promise<Typo | null> | null = null
const cache = new Map<string, string[]>()

function loadDictionary(): Promise<Typo | null> {
  if (dictionaryPromise) return dictionaryPromise
  dictionaryPromise = (async () => {
    try {
      const [affResponse, dicResponse] = await Promise.all([
        fetch('/dictionaries/en_US.aff'),
        fetch('/dictionaries/en_US.dic'),
      ])
      if (!affResponse.ok || !dicResponse.ok) return null
      return new Typo('en_US', await affResponse.text(), await dicResponse.text())
    } catch {
      return null
    }
  })()
  return dictionaryPromise
}

self.onmessage = async (event: MessageEvent<SuggestionRequest>) => {
  const { id, word, limit } = event.data
  const dictionary = await loadDictionary()
  let suggestions = cache.get(word)
  if (!suggestions) {
    suggestions = dictionary?.suggest(word) ?? []
    cache.set(word, suggestions)
  }
  self.postMessage({ id, suggestions: suggestions.slice(0, limit) })
}
