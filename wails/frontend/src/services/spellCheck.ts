import Typo from 'typo-js'

let dictionary: Typo | null = null
let loadingPromise: Promise<void> | null = null
let enabled = true
const changeListeners = new Set<() => void>()
const customWords = new Set<string>()
const ignoredWords = new Set<string>()
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

const apostrophePattern = /[’‘ʼ＇]/g
// Latin script (not bare ASCII) so accented prose — café, naïve, fiancée —
// is checked as whole words instead of being split at the diacritic.
const spellWordPattern = /\p{Script=Latin}+(?:['’‘ʼ＇]\p{Script=Latin}+)*/gu

export function normalizeSpellWord(word: string): string {
  return word.replace(apostrophePattern, "'")
}

function cleanWord(word: string): string {
  return normalizeSpellWord(word).replace(/^[^\p{Script=Latin}0-9]+|[^\p{Script=Latin}0-9]+$/gu, '')
}

export function getDictionaryRoot(word: string): string {
  return cleanWord(word).replace(/'s$/i, '')
}

export function normalizeCustomDictionary(words: string[]): string[] {
  const normalized = new Map<string, string>()
  words.forEach(word => {
    const root = getDictionaryRoot(word.trim())
    if (root) normalized.set(root.toLocaleLowerCase(), root)
  })
  return [...normalized.values()].sort((left, right) => left.localeCompare(right))
}

export function normalizeIgnoredWords(names: string[]): string[] {
  const normalized = new Map<string, string>()
  names.forEach(name => {
    for (const token of name.match(spellWordPattern) ?? []) {
      const root = getDictionaryRoot(token)
      if (root) normalized.set(root.toLocaleLowerCase(), root)
    }
  })
  return [...normalized.values()].sort((left, right) => left.localeCompare(right))
}

function notifyChanged(): void {
  changeListeners.forEach(listener => listener())
}

function matchCase(candidate: string, source: string): string {
  let matched = candidate
  if (source === source.toLocaleUpperCase()) matched = candidate.toLocaleUpperCase()
  else if (source[0] === source[0]?.toLocaleUpperCase()) {
    matched = candidate[0].toLocaleUpperCase() + candidate.slice(1)
  }
  const sourceApostrophe = source.match(/[’‘ʼ＇]/)?.[0]
  return sourceApostrophe ? matched.replace(/'/g, sourceApostrophe) : matched
}

export async function loadDictionary(): Promise<void> {
  if (dictionary) return
  if (loadingPromise) return loadingPromise

  loadingPromise = (async () => {
    try {
      // Parse and index the worker dictionary in parallel with the main checker
      // so the first context menu never pays the cold-start cost.
      prewarmSpellSuggestions()
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

// Project words are transient: confirmed character names and aliases should
// not be persisted into the writer's personal dictionary.
export function setIgnoredWords(words: string[]): void {
  const next = new Set(normalizeIgnoredWords(words).map(word => word.toLocaleLowerCase()))
  if (next.size === ignoredWords.size && [...next].every(word => ignoredWords.has(word))) return

  ignoredWords.clear()
  next.forEach(word => ignoredWords.add(word))
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
  if (customWords.has(key) || customWords.has(rootKey) || ignoredWords.has(key) || ignoredWords.has(rootKey)) return true
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

  return [...ranked].slice(0, limit).map(suggestion => matchCase(suggestion, word))
}

function getSuggestionWorker(): Worker | null {
  if (suggestionWorker) return suggestionWorker
  if (typeof Worker === 'undefined') return null

  try {
    suggestionWorker = new Worker(
      new URL('../workers/spellSuggestions.worker.ts', import.meta.url),
      { type: 'module' },
    )
    suggestionWorker.onmessage = (event: MessageEvent<
      | { type: 'ready' }
      | { type: 'suggestions'; id: number; suggestions: string[] }
    >) => {
      if (event.data.type === 'ready') return
      const resolve = pendingSuggestionRequests.get(event.data.id)
      if (!resolve) return
      pendingSuggestionRequests.delete(event.data.id)
      resolve(event.data.suggestions)
    }
    suggestionWorker.onerror = () => {
      pendingSuggestionRequests.forEach(resolve => resolve([]))
      pendingSuggestionRequests.clear()
      suggestionCache.clear()
      suggestionWorker?.terminate()
      suggestionWorker = null
    }
    return suggestionWorker
  } catch {
    suggestionWorker = null
    return null
  }
}

export function prewarmSpellSuggestions(): void {
  getSuggestionWorker()?.postMessage({ type: 'init' })
}

function requestWorkerSuggestions(word: string, limit: number): Promise<string[]> {
  const worker = getSuggestionWorker()
  if (!worker) return Promise.resolve([])

  return new Promise(resolve => {
    const id = nextSuggestionRequestId++
    pendingSuggestionRequests.set(id, resolve)
    worker.postMessage({ type: 'suggest', id, word, limit })
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
  const ranked = new Map<string, string>()
  const add = (suggestion: string) => {
    const normalized = cleanWord(suggestion).toLocaleLowerCase()
    if (normalized && normalized !== key && !ranked.has(normalized)) {
      ranked.set(normalized, matchCase(normalized, word))
    }
  }
  immediate.forEach(add)
  broad.forEach(add)
  return [...ranked.values()].slice(0, limit)
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
  while (start > 0 && /[\p{Script=Latin}0-9'’‘ʼ＇]/u.test(text[start - 1])) {
    start--
  }

  // Move end forward to end of word
  while (end < text.length && /[\p{Script=Latin}0-9'’‘ʼ＇]/u.test(text[end])) {
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
