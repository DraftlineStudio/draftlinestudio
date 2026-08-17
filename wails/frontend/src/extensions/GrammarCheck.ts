import { Extension } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { analyzeGrammar, onGrammarSettingsChanged } from '../services/grammarCheck'

export const GrammarCheckPluginKey = new PluginKey<DecorationSet>('grammarCheck')

export const GrammarCheck = Extension.create({
  name: 'grammarCheck',

  addProseMirrorPlugins() {
    return [
      new Plugin<DecorationSet>({
        key: GrammarCheckPluginKey,
        state: {
          init: (_, { doc }) => buildDecorations(doc),
          apply: (transaction, decorations, _oldState, newState) => {
            if (transaction.getMeta(GrammarCheckPluginKey)) return buildDecorations(newState.doc)
            return transaction.docChanged
              ? decorations.map(transaction.mapping, transaction.doc)
              : decorations
          },
        },
        props: {
          decorations(state) {
            return GrammarCheckPluginKey.getState(state)
          },
        },
        view(editorView) {
          let refreshTimer: ReturnType<typeof setTimeout> | null = null
          const scheduleRefresh = () => {
            if (refreshTimer) clearTimeout(refreshTimer)
            refreshTimer = setTimeout(() => {
              refreshTimer = null
              if (!editorView.isDestroyed) {
                editorView.dispatch(editorView.state.tr.setMeta(GrammarCheckPluginKey, true))
              }
            }, 180)
          }
          const unsubscribe = onGrammarSettingsChanged(scheduleRefresh)
          return {
            update(view, previousState) {
              if (view.state.doc !== previousState.doc) scheduleRefresh()
            },
            destroy() {
              if (refreshTimer) clearTimeout(refreshTimer)
              unsubscribe()
            },
          }
        },
      }),
    ]
  },
})

function buildDecorations(doc: ProseMirrorNode): DecorationSet {
  const decorations: Decoration[] = []
  doc.descendants((node, pos) => {
    if (!node.isText) return
    for (const issue of analyzeGrammar(node.text || '')) {
      decorations.push(Decoration.inline(pos + issue.from, pos + issue.to, {
        class: issue.kind === 'style' ? 'style-error' : 'grammar-error',
        title: issue.message,
      }))
    }
  })
  return DecorationSet.create(doc, decorations)
}
