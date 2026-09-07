import { Extension } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import { checkWord, isLoaded, onDictionaryChanged } from '../services/spellCheck'

export const SpellCheckPluginKey = new PluginKey<DecorationSet>('spellCheck')

export const SpellCheck = Extension.create({
  name: 'spellCheck',

  addProseMirrorPlugins() {
    return [
      new Plugin<DecorationSet>({
        key: SpellCheckPluginKey,
        state: {
          init: (_, { doc }) => buildDecorations(doc),
          apply: (transaction, decorations, _oldState, newState) => {
            if (transaction.getMeta(SpellCheckPluginKey)) {
              return buildDecorations(newState.doc)
            }
            return transaction.docChanged
              ? decorations.map(transaction.mapping, transaction.doc)
              : decorations
          },
        },
        props: {
          decorations(state) {
            return SpellCheckPluginKey.getState(state)
          },
        },
        view(editorView) {
          let refreshTimer: ReturnType<typeof setTimeout> | null = null
          const scheduleRefresh = () => {
            if (refreshTimer) clearTimeout(refreshTimer)
            refreshTimer = setTimeout(() => {
              refreshTimer = null
              if (!editorView.isDestroyed) {
                editorView.dispatch(editorView.state.tr.setMeta(SpellCheckPluginKey, true))
              }
            }, 150)
          }
          const unsubscribe = onDictionaryChanged(scheduleRefresh)
          if (isLoaded()) scheduleRefresh()
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
  // Keep in sync with spellWordPattern in services/spellCheck.ts: Latin
  // script so accented words stay whole.
  const wordPattern = /\p{Script=Latin}+(?:['’‘ʼ＇]\p{Script=Latin}+)*/gu

  doc.descendants((node, pos) => {
    if (!node.isText) return

    const text = node.text || ''
    wordPattern.lastIndex = 0

    let match: RegExpExecArray | null
    while ((match = wordPattern.exec(text)) !== null) {
      const before = text[match.index - 1]
      const after = text[match.index + match[0].length]
      if (/\d/.test(before) || /\d/.test(after)) continue

      if (!checkWord(match[0])) {
        decorations.push(
          Decoration.inline(pos + match.index, pos + match.index + match[0].length, {
            class: 'spelling-error',
          }),
        )
      }
    }
  })

  return DecorationSet.create(doc, decorations)
}
