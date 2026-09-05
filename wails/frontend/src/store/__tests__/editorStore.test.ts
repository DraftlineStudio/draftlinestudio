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

  it('applies a selected-text review through the editor range without replacing the chapter', async () => {
    let chapterHtml = '<p>Before old after.</p>'
    const insertContentAt = vi.fn((_range: { from: number; to: number }, content: string) => {
      expect(content).toBe('<p>new</p>')
      chapterHtml = '<p>Before new after.</p>'
      return true
    })
    editorStoreMod.useEditorStore.setState({
      editorRef: {
        getHTML: () => chapterHtml,
        commands: { insertContentAt },
      } as any,
    })
    await editorStoreMod.useEditorStore.getState().setPendingDiff({
      diffs: [changedParagraph],
      originalHtml: '<p>old</p>',
      target: {
        kind: 'selection',
        from: 8,
        to: 11,
        sourceDocumentHtml: '<p>Before old after.</p>',
      },
      historyReason: 'AI Line Edit',
    })
    editorStoreMod.useEditorStore.getState().acceptAllDiff()
    const updateContent = vi.fn()

    const applied = await editorStoreMod.useEditorStore.getState().applyPendingDiff(updateContent)

    expect(insertContentAt).toHaveBeenCalledWith({ from: 8, to: 11 }, '<p>new</p>')
    expect(updateContent).toHaveBeenCalledWith('<p>Before new after.</p>')
    expect(updateContent).not.toHaveBeenCalledWith('<p>new</p>')
    expect(applied).toEqual({
      beforeHtml: '<p>Before old after.</p>',
      afterHtml: '<p>Before new after.</p>',
      historyReason: 'AI Line Edit',
    })
  })

  it('refuses a stale selected-text review without changing any content', async () => {
    const insertContentAt = vi.fn(() => true)
    editorStoreMod.useEditorStore.setState({
      editorRef: {
        getHTML: () => '<p>The author edited this chapter meanwhile.</p>',
        commands: { insertContentAt },
      } as any,
    })
    await editorStoreMod.useEditorStore.getState().setPendingDiff({
      diffs: [changedParagraph],
      originalHtml: '<p>old</p>',
      target: { kind: 'selection', from: 1, to: 4, sourceDocumentHtml: '<p>old chapter</p>' },
    })
    editorStoreMod.useEditorStore.getState().acceptAllDiff()
    const updateContent = vi.fn()

    expect(await editorStoreMod.useEditorStore.getState().applyPendingDiff(updateContent)).toBeNull()
    expect(insertContentAt).not.toHaveBeenCalled()
    expect(updateContent).not.toHaveBeenCalled()
    expect(editorStoreMod.useEditorStore.getState().pendingDiff?.applyError).toContain('chapter changed')
  })
})
