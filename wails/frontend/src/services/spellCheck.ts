import Typo from 'typo-js'

let dictionary: Typo | null = null
let loadingPromise: Promise<void> | null = null
let enabled = true
const changeListeners = new Set<() => void>()
const customWords = new Set<string>()
const checkCache = new Map<string, boolean>()
const suggestionCache = new Map<string, Promise<string[]>>()
const pendingSuggestionRequests = new Map<number, (suggestions: string[]) => void>()
let suggestionWorker: Worker | null = null
let nextSuggestionRequestId = 1
const commonCorrections: Record<string, string[]> = {
  hte: ['the'],
  teh: ['the'],
  th: ['the'],
  ot: ['to'],
}

function cleanWord(word: string): string {
  return word.replace(/^[^A-Za-z0-9]+|[^A-Za-z0-9]+$/g, '')
}

export function getDictionaryRoot(word: string): string {
  return cleanWord(word).replace(/['’]s$/i, '')
}

export function normalizeCustomDictionary(words: string[]): string[] {
  const normalized = new Map<string, string>()
  words.forEach(word => {
    const root = getDictionaryRoot(word.trim())
    if (root) normalized.set(root.toLocaleLowerCase(), root)
  })
  return [...normalized.values()].sort((left, right) => left.localeCompare(right))
}

function notifyChanged(): void {
  changeListeners.forEach(listener => listener())
}

function matchCase(candidate: string, source: string): string {
  if (source === source.toLocaleUpperCase()) return candidate.toLocaleUpperCase()
  if (source[0] === source[0]?.toLocaleUpperCase()) {
    return candidate[0].toLocaleUpperCase() + candidate.slice(1)
  }
  return candidate
}

export async function loadDictionary(): Promise<void> {
  if (dictionary) return
  if (loadingPromise) return loadingPromise

  loadingPromise = (async () => {
    try {
      const [affResponse, dicResponse] = await Promise.all([
        fetch('/dictionaries/en_US.aff'),
        fetch('/dictionaries/en_US.dic'),
      ])

      if (!affResponse.ok || !dicResponse.ok) {
        console.warn('Failed to load dictionary files')
        return
      }

      const affData = await affResponse.text()
      const dicData = await dicResponse.text()

      dictionary = new Typo('en_US', affData, dicData)
      checkCache.clear()
      suggestionCache.clear()
      notifyChanged()
      console.log('Spell check dictionary loaded')
    } catch (err) {
      console.warn('Failed to initialize spell checker:', err)
    }
  })()

  return loadingPromise
}

export function isLoaded(): boolean {
  return dictionary !== null
}

export function setSpellCheckEnabled(next: boolean): void {
  if (enabled === next) return
  enabled = next
  checkCache.clear()
  notifyChanged()
}

export function onDictionaryChanged(listener: () => void): () => void {
  changeListeners.add(listener)
  return () => changeListeners.delete(listener)
}

export function setCustomWords(words: string[]): void {
  const next = new Set(normalizeCustomDictionary(words).map(word => word.toLocaleLowerCase()))
  if (next.size === customWords.size && [...next].every(word => customWords.has(word))) return

  customWords.clear()
  next.forEach(word => customWords.add(word))
  checkCache.clear()
  suggestionCache.clear()
  notifyChanged()
}

export function checkWord(word: string): boolean {
  if (!enabled || !dictionary) return true // Assume correct if disabled or no dictionary
  const cleaned = cleanWord(word)
  if (!cleaned || cleaned.length < 2) return true
  if (/^\d+(?:st|nd|rd|th)$/i.test(cleaned)) return true

  const key = cleaned.toLocaleLowerCase()
  const root = getDictionaryRoot(cleaned)
  const rootKey = root.toLocaleLowerCase()
  if (customWords.has(key) || customWords.has(rootKey)) return true
  const cached = checkCache.get(key)
  if (cached !== undefined) return cached

  const correct = dictionary.check(cleaned)
    || (root !== cleaned && dictionary.check(root))
  checkCache.set(key, correct)
  return correct
}

export function getImmediateSuggestions(word: string, limit = 5): string[] {
  if (!dictionary) return []
  const cleaned = cleanWord(word)
  if (!cleaned) return []
  const key = cleaned.toLocaleLowerCase()
  const ranked = new Set<string>()
  const addIfCorrect = (candidate: string) => {
    if (candidate && candidate !== key && dictionary?.check(candidate)) ranked.add(candidate)
  }

  commonCorrections[key]?.forEach(addIfCorrect)
  if (key.startsWith('e') && key.length > 2) addIfCorrect(key.slice(1))

  for (let index = 0; index < key.length - 1; index++) {
    const candidate = key.slice(0, index)
      + key[index + 1]
      + key[index]
      + key.slice(index + 2)
    addIfCorrect(candidate)
  }

  return [...ranked].slice(0, limit).map(suggestion => matchCase(suggestion, cleaned))
}

function getSuggestionWorker(): Worker | null {
  if (suggestionWorker) return suggestionWorker
  if (typeof Worker === 'undefined') return null

  try {
    suggestionWorker = new Worker(
      new URL('../workers/spellSuggestions.worker.ts', import.meta.url),
      { type: 'module' },
    )
    suggestionWorker.onmessage = (event: MessageEvent<{ id: number; suggestions: string[] }>) => {
      const resolve = pendingSuggestionRequests.get(event.data.id)
      if (!resolve) return
      pendingSuggestionRequests.delete(event.data.id)
      resolve(event.data.suggestions)
    }
    suggestionWorker.onerror = () => {
      pendingSuggestionRequests.forEach(resolve => resolve([]))
      pendingSuggestionRequests.clear()
      suggestionWorker?.terminate()
      suggestionWorker = null
    }
    return suggestionWorker
  } catch {
    suggestionWorker = null
    return null
  }
}

function requestWorkerSuggestions(word: string, limit: number): Promise<string[]> {
  const worker = getSuggestionWorker()
  if (!worker) return Promise.resolve([])

  return new Promise(resolve => {
    const id = nextSuggestionRequestId++
    pendingSuggestionRequests.set(id, resolve)
    worker.postMessage({ id, word, limit })
  })
}

export async function getSuggestions(word: string, limit = 5): Promise<string[]> {
  const cleaned = cleanWord(word)
  if (!cleaned) return []
  const key = cleaned.toLocaleLowerCase()
  let broadSuggestions = suggestionCache.get(key)
  if (!broadSuggestions) {
    broadSuggestions = requestWorkerSuggestions(key, limit)
    suggestionCache.set(key, broadSuggestions)
  }

  const immediate = getImmediateSuggestions(cleaned, limit)
  const broad = await broadSuggestions
  const ranked = new Set([...immediate, ...broad.map(suggestion => matchCase(suggestion, cleaned))])
  return [...ranked].slice(0, limit)
}

// Get the word at the current cursor position in a contenteditable
export function getWordAtCursor(): { word: string; range: Range } | null {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return null

  const range = selection.getRangeAt(0)
  const node = range.startContainer

  if (node.nodeType !== Node.TEXT_NODE) return null

  const text = node.textContent || ''
  const offset = range.startOffset

  // Find word boundaries
  let start = offset
  let end = offset

  // Move start back to beginning of word
  while (start > 0 && /\w/.test(text[start - 1])) {
    start--
  }

  // Move end forward to end of word
  while (end < text.length && /\w/.test(text[end])) {
    end++
  }

  if (start === end) return null

  const word = text.substring(start, end)

  // Create a range for the word
  const wordRange = document.createRange()
  wordRange.setStart(node, start)
  wordRange.setEnd(node, end)

  return { word, range: wordRange }
}

// Replace a word at a given range with a new word
export function replaceWord(range: Range, newWord: string): void {
  const selection = window.getSelection()
  if (!selection) return

  selection.removeAllRanges()
  selection.addRange(range)

  // Use execCommand for contenteditable compatibility with undo
  document.execCommand('insertText', false, newWord)
}
