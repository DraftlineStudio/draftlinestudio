import { Extension } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { Plugin, PluginKey, type Transaction } from '@tiptap/pm/state'
import { Decoration, DecorationSet, type EditorView } from '@tiptap/pm/view'

export interface ChapterSearchOptions {
  query: string
  caseSensitive: boolean
  wholeWord: boolean
  activeIndex: number
}

export interface ChapterSearchMatch {
  from: number
  to: number
}

interface ChapterSearchState extends ChapterSearchOptions {
  matches: ChapterSearchMatch[]
  decorations: DecorationSet
}

const emptyOptions: ChapterSearchOptions = {
  query: '',
  caseSensitive: false,
  wholeWord: false,
  activeIndex: 0,
}

export const ChapterSearchPluginKey = new PluginKey<ChapterSearchState>('chapterSearch')

export const ChapterSearch = Extension.create({
  name: 'chapterSearch',

  addProseMirrorPlugins() {
    return [
      new Plugin<ChapterSearchState>({
        key: ChapterSearchPluginKey,
        state: {
          init: (_, { doc }) => buildState(doc, emptyOptions),
          apply(transaction, previous, _oldState, newState) {
            const update = transaction.getMeta(ChapterSearchPluginKey) as Partial<ChapterSearchOptions> | undefined
            if (!transaction.docChanged && !update) return previous
            return buildState(newState.doc, { ...previous, ...update })
          },
        },
        props: {
          decorations(state) {
            return ChapterSearchPluginKey.getState(state)?.decorations ?? null
          },
        },
      }),
    ]
  },
})

export function updateChapterSearch(view: EditorView, update: Partial<ChapterSearchOptions>) {
  view.dispatch(view.state.tr.setMeta(ChapterSearchPluginKey, update))
}

export function getChapterSearchState(view: EditorView): ChapterSearchState | undefined {
  return ChapterSearchPluginKey.getState(view.state)
}

export function replaceCurrentChapterMatch(view: EditorView, replacement: string): boolean {
  const state = getChapterSearchState(view)
  const match = state?.matches[state.activeIndex]
  if (!state || !match) return false

  const transaction = view.state.tr.insertText(replacement, match.from, match.to)
  transaction.setMeta(ChapterSearchPluginKey, { activeIndex: state.activeIndex })
  view.dispatch(transaction)

  // Advance beyond the inserted text. This prevents replacements that still
  // contain the query (for example, Mara -> Mara Ionescu) from repeatedly
  // targeting the text that was just inserted.
  const updated = getChapterSearchState(view)
  if (updated?.matches.length) {
    const afterReplacement = match.from + replacement.length
    const nextIndex = updated.matches.findIndex(candidate => candidate.from >= afterReplacement)
    updateChapterSearch(view, { activeIndex: nextIndex === -1 ? 0 : nextIndex })
  }
  return true
}

export function replaceAllChapterMatches(view: EditorView, replacement: string): number {
  const state = getChapterSearchState(view)
  if (!state?.matches.length) return 0

  // Work backwards so earlier document positions remain valid. A single
  // transaction also makes Replace All a single undo operation.
  const transaction: Transaction = view.state.tr
  for (let index = state.matches.length - 1; index >= 0; index--) {
    const match = state.matches[index]
    transaction.insertText(replacement, match.from, match.to)
  }
  transaction.setMeta(ChapterSearchPluginKey, { activeIndex: 0 })
  view.dispatch(transaction)
  return state.matches.length
}

function buildState(doc: ProseMirrorNode, options: ChapterSearchOptions): ChapterSearchState {
  const matches = findChapterMatches(doc, options)
  const activeIndex = matches.length
    ? Math.min(Math.max(options.activeIndex, 0), matches.length - 1)
    : 0
  const decorations = matches.map((match, index) => Decoration.inline(match.from, match.to, {
    class: index === activeIndex ? 'chapter-search-match chapter-search-match-current' : 'chapter-search-match',
  }))

  return {
    query: options.query,
    caseSensitive: options.caseSensitive,
    wholeWord: options.wholeWord,
    activeIndex,
    matches,
    decorations: DecorationSet.create(doc, decorations),
  }
}

function findChapterMatches(doc: ProseMirrorNode, options: ChapterSearchOptions): ChapterSearchMatch[] {
  if (!options.query) return []

  const needle = options.caseSensitive ? options.query : options.query.toLocaleLowerCase()
  const matches: ChapterSearchMatch[] = []

  doc.descendants((node, nodePosition) => {
    if (!node.isTextblock) return

    // Adjacent text nodes can carry different formatting marks. Treat them as
    // one searchable run so a phrase remains findable across bold/italic text,
    // while hard breaks and block boundaries still stop a match.
    const runs: Array<{ text: string; from: number }> = []
    node.descendants((child, relativePosition) => {
      if (!child.isText || !child.text) return
      const from = nodePosition + 1 + relativePosition
      const previous = runs[runs.length - 1]
      if (previous && previous.from + previous.text.length === from) {
        previous.text += child.text
      } else {
        runs.push({ text: child.text, from })
      }
    })

    for (const run of runs) {
      const haystack = options.caseSensitive ? run.text : run.text.toLocaleLowerCase()
      let offset = 0
      while (offset <= haystack.length - needle.length) {
        const found = haystack.indexOf(needle, offset)
        if (found === -1) break

        const before = run.text[found - 1]
        const after = run.text[found + options.query.length]
        const isWholeWord = !options.wholeWord || (!isWordCharacter(before) && !isWordCharacter(after))
        if (isWholeWord) {
          matches.push({ from: run.from + found, to: run.from + found + options.query.length })
        }
        offset = found + Math.max(needle.length, 1)
      }
    }
  })

  return matches
}

function isWordCharacter(character: string | undefined): boolean {
  return character != null && /[\p{L}\p{N}_]/u.test(character)
}
