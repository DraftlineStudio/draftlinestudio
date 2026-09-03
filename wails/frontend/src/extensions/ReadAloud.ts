// Read Aloud sentence highlighting. Follows the ChapterSearch pattern: a
// meta-driven ProseMirror plugin holding sentence ranges and the active
// index, rendered purely as an inline decoration — document content and the
// undo history are never touched. Click and doc-edit callbacks are injected
// by the playback store so this module stays free of store imports.

import { Extension } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet, type EditorView } from '@tiptap/pm/view'

export interface ReadAloudRange {
  from: number
  to: number
}

export interface ReadAloudUpdate {
  active?: boolean
  sentences?: ReadAloudRange[]
  activeIndex?: number
}

interface ReadAloudPluginState {
  active: boolean
  sentences: ReadAloudRange[]
  activeIndex: number
  decorations: DecorationSet
}

export const ReadAloudPluginKey = new PluginKey<ReadAloudPluginState>('readAloud')

interface ReadAloudHandlers {
  onSentenceClick?: (index: number) => void
  onDocEdited?: () => void
}

let handlers: ReadAloudHandlers = {}

export function setReadAloudHandlers(next: ReadAloudHandlers): void {
  handlers = next
}

const inactiveState: Omit<ReadAloudPluginState, 'decorations'> = {
  active: false,
  sentences: [],
  activeIndex: -1,
}

export const ReadAloud = Extension.create({
  name: 'readAloud',

  addProseMirrorPlugins() {
    return [
      new Plugin<ReadAloudPluginState>({
        key: ReadAloudPluginKey,
        state: {
          init: () => ({ ...inactiveState, decorations: DecorationSet.empty }),
          apply(transaction, previous, _oldState, newState) {
            const update = transaction.getMeta(ReadAloudPluginKey) as ReadAloudUpdate | undefined
            if (update) {
              const next = { ...previous, ...update }
              return { ...next, decorations: buildDecorations(newState.doc, next) }
            }
            if (!transaction.docChanged) return previous
            if (previous.active) {
              // Editing during playback stops it. Never dispatch inside
              // apply; the store's stop() clears this state properly.
              queueMicrotask(() => handlers.onDocEdited?.())
            }
            // Map ranges across the edit so nothing flashes at a stale
            // position before the stop lands.
            return {
              ...previous,
              sentences: previous.sentences.map(r => ({
                from: transaction.mapping.map(r.from),
                to: transaction.mapping.map(r.to),
              })),
              decorations: previous.decorations.map(transaction.mapping, transaction.doc),
            }
          },
        },
        props: {
          decorations(state) {
            return ReadAloudPluginKey.getState(state)?.decorations ?? null
          },
          handleClick(view, pos) {
            const state = ReadAloudPluginKey.getState(view.state)
            if (!state?.active || !handlers.onSentenceClick) return false
            const index = state.sentences.findIndex(s => pos >= s.from && pos < s.to)
            if (index !== -1) handlers.onSentenceClick(index)
            // Let the cursor move normally either way.
            return false
          },
        },
      }),
    ]
  },
})

function buildDecorations(doc: ProseMirrorNode, state: Omit<ReadAloudPluginState, 'decorations'>): DecorationSet {
  if (!state.active || state.activeIndex < 0 || state.activeIndex >= state.sentences.length) {
    return DecorationSet.empty
  }
  const sentence = state.sentences[state.activeIndex]
  const from = Math.max(0, Math.min(sentence.from, doc.content.size))
  const to = Math.max(from, Math.min(sentence.to, doc.content.size))
  if (from === to) return DecorationSet.empty
  return DecorationSet.create(doc, [
    Decoration.inline(from, to, { class: 'read-aloud-current' }),
  ])
}

export function updateReadAloud(view: EditorView, update: ReadAloudUpdate): void {
  view.dispatch(view.state.tr.setMeta(ReadAloudPluginKey, update))
}

export function clearReadAloud(view: EditorView): void {
  updateReadAloud(view, { ...inactiveState })
}
