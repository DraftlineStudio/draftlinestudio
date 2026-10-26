import type { Editor } from '@tiptap/core'
import { analyzeTypography } from './grammarCheck'
import { blockText } from '../extensions/GrammarCheck'

/**
 * Applies every typography fix in the chapter — straight quotes curled,
 * typed ellipses and double hyphens converted — as one transaction, so a
 * single undo reverts the whole sweep. Returns how many fixes were applied.
 *
 * Positions come from the same analysis the underlines use (blockText keeps
 * the offsets aligned), applied back to front so earlier fixes cannot shift
 * later ones.
 */
export function fixChapterTypography(editor: Editor): number {
  const { state } = editor
  const fixes: { from: number; to: number; text: string }[] = []

  state.doc.descendants((node, pos) => {
    if (!node.isTextblock) return
    for (const issue of analyzeTypography(blockText(node))) {
      if (!issue.replacements.length) continue
      fixes.push({
        from: pos + 1 + issue.from,
        to: pos + 1 + issue.to,
        text: issue.replacements[0],
      })
    }
    return false
  })

  if (fixes.length === 0) return 0
  const transaction = state.tr
  for (const fix of fixes.sort((left, right) => right.from - left.from)) {
    transaction.insertText(fix.text, fix.from, fix.to)
  }
  editor.view.dispatch(transaction)
  return fixes.length
}
