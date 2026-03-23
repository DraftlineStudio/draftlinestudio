import Typo from 'typo-js'

let dictionary: Typo | null = null
let loadingPromise: Promise<void> | null = null

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

export function checkWord(word: string): boolean {
  if (!dictionary) return true // Assume correct if no dictionary
  // Strip punctuation from edges
  const cleanWord = word.replace(/^[^\w]+|[^\w]+$/g, '')
  if (!cleanWord || cleanWord.length < 2) return true
  return dictionary.check(cleanWord)
}

export function getSuggestions(word: string, limit = 5): string[] {
  if (!dictionary) return []
  const cleanWord = word.replace(/^[^\w]+|[^\w]+$/g, '')
  if (!cleanWord) return []
  const suggestions = dictionary.suggest(cleanWord)
  return suggestions.slice(0, limit)
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
