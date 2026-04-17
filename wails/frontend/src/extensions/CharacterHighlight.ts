import { Extension } from '@tiptap/core'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'

export interface CharacterHighlightOptions {
  /** Function that returns array of names to highlight */
  getNames: () => string[]
  /** CSS class to apply to highlights */
  highlightClass: string
}

export const CharacterHighlightPluginKey = new PluginKey('characterHighlight')

export const CharacterHighlight = Extension.create<CharacterHighlightOptions>({
  name: 'characterHighlight',

  addOptions() {
    return {
      getNames: () => [],
      highlightClass: 'character-highlight',
    }
  },

  addProseMirrorPlugins() {
    const { getNames, highlightClass } = this.options

    return [
      new Plugin({
        key: CharacterHighlightPluginKey,
        state: {
          init(_, { doc }) {
            return buildDecorations(doc, getNames(), highlightClass)
          },
          apply(tr, oldDecorations, oldState, newState) {
            // Rebuild decorations if document changed or names changed
            // For simplicity, we rebuild on every transaction
            // This could be optimized to only rebuild when names change
            return buildDecorations(newState.doc, getNames(), highlightClass)
          },
        },
        props: {
          decorations(state) {
            return this.getState(state)
          },
        },
      }),
    ]
  },
})

function buildDecorations(doc: any, names: string[], className: string): DecorationSet {
  const decorations: Decoration[] = []

  // Pattern for @ai prompts (always active)
  const aiPromptPattern = /@ai\s+[^\n]+/gi

  // Build regex pattern for character names (if any)
  let namesPattern: RegExp | null = null
  if (names.length > 0) {
    const escapedNames = names.map(n => n.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
    namesPattern = new RegExp(`\\b(${escapedNames.join('|')})\\b`, 'gi')
  }

  doc.descendants((node: any, pos: number) => {
    if (!node.isText) return

    const text = node.text || ''

    // Highlight @ai prompts
    let match
    aiPromptPattern.lastIndex = 0
    while ((match = aiPromptPattern.exec(text)) !== null) {
      const from = pos + match.index
      const to = from + match[0].length

      decorations.push(
        Decoration.inline(from, to, { class: 'ai-prompt-highlight' })
      )
    }

    // Highlight character names
    if (namesPattern) {
      namesPattern.lastIndex = 0
      while ((match = namesPattern.exec(text)) !== null) {
        const from = pos + match.index
        const to = from + match[0].length

        decorations.push(
          Decoration.inline(from, to, { class: className })
        )
      }
    }
  })

  return DecorationSet.create(doc, decorations)
}
