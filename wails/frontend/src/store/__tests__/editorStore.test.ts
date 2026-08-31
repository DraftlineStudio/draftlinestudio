import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ParagraphDiff } from '../../utils/diff'

let editorStoreMod: typeof import('../editorStore')

const changedParagraph: ParagraphDiff = {
  chunks: [
    { type: 'delete', text: 'old' },
    { type: 'insert', text: 'new' },
  ],
  hasChanges: true,
  originalText: 'old',
  revisedText: 'new',
  originalHtml: '<p>old</p>',
  revisedHtml: '<p>new</p>',
}

beforeEach(async () => {
  vi.resetModules()
  editorStoreMod = await import('../editorStore')
})

describe('lazy diff-engine coordination', () => {
  it('does not resurrect a diff cleared while the module is loading', async () => {
    const pending = editorStoreMod.useEditorStore.getState().setPendingDiff({
      diffs: [changedParagraph],
      originalHtml: '<p>old</p>',
    })

    editorStoreMod.useEditorStore.getState().clearPendingDiff()
    await pending

    expect(editorStoreMod.useEditorStore.getState().pendingDiff).toBeNull()
  })

  it('applies the latest review decision made while assembly is loading', async () => {
    await editorStoreMod.useEditorStore.getState().setPendingDiff({
      diffs: [changedParagraph],
      originalHtml: '<p>old</p>',
    })
    const updateContent = vi.fn()

    const applying = editorStoreMod.useEditorStore.getState().applyPendingDiff(updateContent)
    editorStoreMod.useEditorStore.getState().acceptChange(0)
    await applying

    expect(updateContent).toHaveBeenCalledWith('<p>new</p>')
    expect(editorStoreMod.useEditorStore.getState().pendingDiff).toBeNull()
  })
})
