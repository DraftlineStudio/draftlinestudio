// RangeSpotlight: the core editor primitive for highlighting one active range
// out of an owned set — Read Aloud's sentence highlight today, and the
// plugin host API's `host.editor.spotlight` surface (plugins cannot register
// ProseMirror plugins at runtime, so the host owns this one on their behalf).
// Follows the ChapterSearch pattern: a meta-driven ProseMirror plugin holding
// ranges and the active index, rendered purely as an inline decoration —
// document content and the undo history are never touched. Click and doc-edit
// callbacks are injected through a module slot so this module imports no
// store. The spotlight is deliberately exclusive: one owner at a time.

import { Extension } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet, type EditorView } from '@tiptap/pm/view'

export interface SpotlightRange {
  from: number
  to: number
}

export interface SpotlightUpdate {
  active?: boolean
  ranges?: SpotlightRange[]
  activeIndex?: number
}

interface SpotlightPluginState {
  active: boolean
  ranges: SpotlightRange[]
  activeIndex: number
  decorations: DecorationSet
}

export const RangeSpotlightKey = new PluginKey<SpotlightPluginState>('rangeSpotlight')

export interface SpotlightHandlers {
  onRangeClick?: (index: number) => void
  onDocEdited?: () => void
}

let handlers: SpotlightHandlers = {}

export function setSpotlightHandlers(next: SpotlightHandlers): void {
  handlers = next
}

const inactiveState: Omit<SpotlightPluginState, 'decorations'> = {
  active: false,
  ranges: [],
  activeIndex: -1,
}

// findRangeIndex maps a document position to the range containing it, -1 when
// none does. Exported for the host API's click routing and for tests.
export function findRangeIndex(ranges: SpotlightRange[], pos: number): number {
  return ranges.findIndex(r => pos >= r.from && pos < r.to)
}

// createSpotlightPlugin builds the ProseMirror plugin. Factored out of the
// TipTap extension so state transitions are testable against a bare
// EditorState (the vitest environment has no DOM).
export function createSpotlightPlugin(): Plugin<SpotlightPluginState> {
  return new Plugin<SpotlightPluginState>({
    key: RangeSpotlightKey,
    state: {
      init: () => ({ ...inactiveState, decorations: DecorationSet.empty }),
      apply(transaction, previous, _oldState, newState) {
        const update = transaction.getMeta(RangeSpotlightKey) as SpotlightUpdate | undefined
        if (update) {
          const next = { ...previous, ...update }
          return { ...next, decorations: buildDecorations(newState.doc, next) }
        }
        if (!transaction.docChanged) return previous
        if (previous.active) {
          // Editing while the spotlight is live stops its owner. Never
          // dispatch inside apply; the owner's stop path clears this state.
          queueMicrotask(() => handlers.onDocEdited?.())
        }
        // Map ranges across the edit so nothing flashes at a stale position
        // before the stop lands.
        return {
          ...previous,
          ranges: previous.ranges.map(r => ({
            from: transaction.mapping.map(r.from),
            to: transaction.mapping.map(r.to),
          })),
          decorations: previous.decorations.map(transaction.mapping, transaction.doc),
        }
      },
    },
    props: {
      decorations(state) {
        return RangeSpotlightKey.getState(state)?.decorations ?? null
      },
      handleClick(view, pos) {
        const state = RangeSpotlightKey.getState(view.state)
        if (!state?.active || !handlers.onRangeClick) return false
        const index = findRangeIndex(state.ranges, pos)
        if (index !== -1) handlers.onRangeClick(index)
        // Let the cursor move normally either way.
        return false
      },
    },
  })
}

export const RangeSpotlight = Extension.create({
  name: 'rangeSpotlight',

  addProseMirrorPlugins() {
    return [createSpotlightPlugin()]
  },
})

function buildDecorations(doc: ProseMirrorNode, state: Omit<SpotlightPluginState, 'decorations'>): DecorationSet {
  if (!state.active || state.activeIndex < 0 || state.activeIndex >= state.ranges.length) {
    return DecorationSet.empty
  }
  const range = state.ranges[state.activeIndex]
  const from = Math.max(0, Math.min(range.from, doc.content.size))
  const to = Math.max(from, Math.min(range.to, doc.content.size))
  if (from === to) return DecorationSet.empty
  return DecorationSet.create(doc, [
    Decoration.inline(from, to, { class: 'range-spotlight-current' }),
  ])
}

export function updateSpotlight(view: EditorView, update: SpotlightUpdate): void {
  view.dispatch(view.state.tr.setMeta(RangeSpotlightKey, update))
}

export function clearSpotlight(view: EditorView): void {
  updateSpotlight(view, { ...inactiveState })
}
