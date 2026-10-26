import { Extension } from '@tiptap/core'
import type { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import type { StyleFinding } from '../types/draftline'
import { blockText } from './GrammarCheck'

// Renders the analysis pipeline's style findings (book.analysis.style) as
// amber underlines. Findings carry plaintext offsets the editor cannot use
// directly, so each is placed by searching for its prefix+snippet+suffix;
// a finding that matches nowhere — or in more than one place — is stale or
// ambiguous and is silently skipped rather than underlined wrongly.

export interface PlacedStyleFinding {
  finding: StyleFinding
  from: number
  to: number
}

export const StyleFindingsPluginKey = new PluginKey<DecorationSet>('styleFindings')

export function placeStyleFindings(doc: ProseMirrorNode, findings: StyleFinding[]): PlacedStyleFinding[] {
  if (!findings.length) return []
  const candidates = new Map<string, PlacedStyleFinding[]>()

  doc.descendants((node, pos) => {
    if (!node.isTextblock) return
    const text = blockText(node)
    for (const finding of findings) {
      const prefix = finding.prefix ?? ''
      const needle = prefix + finding.snippet + (finding.suffix ?? '')
      if (!needle) continue
      let index = text.indexOf(needle)
      while (index !== -1) {
        const from = pos + 1 + index + prefix.length
        const list = candidates.get(finding.id) ?? []
        list.push({ finding, from, to: from + finding.snippet.length })
        candidates.set(finding.id, list)
        index = text.indexOf(needle, index + 1)
      }
    }
    return false
  })

  const placed: PlacedStyleFinding[] = []
  for (const list of candidates.values()) {
    if (list.length === 1) placed.push(list[0])
  }
  return placed.sort((left, right) => left.from - right.from)
}

export interface StyleFindingsOptions {
  getFindings: () => StyleFinding[]
}

export const StyleFindings = Extension.create<StyleFindingsOptions>({
  name: 'styleFindings',

  addOptions() {
    return { getFindings: () => [] }
  },

  addProseMirrorPlugins() {
    const getFindings = this.options.getFindings
    const buildDecorations = (doc: ProseMirrorNode): DecorationSet => {
      const decorations = placeStyleFindings(doc, getFindings()).map(placed =>
        Decoration.inline(placed.from, placed.to, {
          class: 'style-error',
          title: placed.finding.message,
        }))
      return DecorationSet.create(doc, decorations)
    }

    return [
      new Plugin<DecorationSet>({
        key: StyleFindingsPluginKey,
        state: {
          init: (_, { doc }) => buildDecorations(doc),
          apply: (transaction, decorations, _oldState, newState) => {
            if (transaction.getMeta(StyleFindingsPluginKey)) return buildDecorations(newState.doc)
            return transaction.docChanged
              ? decorations.map(transaction.mapping, transaction.doc)
              : decorations
          },
        },
        props: {
          decorations(state) {
            return StyleFindingsPluginKey.getState(state)
          },
        },
        view(editorView) {
          let refreshTimer: ReturnType<typeof setTimeout> | null = null
          const scheduleRefresh = () => {
            if (refreshTimer) clearTimeout(refreshTimer)
            refreshTimer = setTimeout(() => {
              refreshTimer = null
              if (!editorView.isDestroyed) {
                editorView.dispatch(editorView.state.tr.setMeta(StyleFindingsPluginKey, true))
              }
            }, 250)
          }
          return {
            update(view, previousState) {
              if (view.state.doc !== previousState.doc) scheduleRefresh()
            },
            destroy() {
              if (refreshTimer) clearTimeout(refreshTimer)
            },
          }
        },
      }),
    ]
  },
})
