// Editor Store - Editor reference, selection, inline prompt, diff/review state

import { create } from 'zustand'
import type { ParagraphDiff, DiffChange } from '../utils/diff'

// Editor instance type (minimal interface for selection access)
export interface EditorInstance {
  state: {
    selection: { from: number; to: number; empty: boolean }
    doc: { textBetween: (from: number, to: number, separator: string) => string }
  }
  getHTML: () => string
  view: { state: { selection: { from: number; to: number } } }
  commands: { setTextSelection: (range: { from: number; to: number }) => boolean }
}

interface EditorStore {
  // Editor reference
  editorRef: EditorInstance | null
  setEditorRef: (editor: EditorInstance | null) => void
  getEditorSelection: () => { html: string; text: string; from: number; to: number } | null

  // Inline AI prompt (Ctrl+L)
  inlinePrompt: {
    active: boolean
    cursorPos: number
  } | null
  openInlinePrompt: (cursorPos: number) => void
  closeInlinePrompt: () => void

  // Diff/review state
  pendingDiff: {
    diffs: ParagraphDiff[]
    changes: DiffChange[]
    focusedChangeIdx: number
    originalHtml: string
  } | null
  setPendingDiff: (payload: { diffs: ParagraphDiff[]; originalHtml: string }) => Promise<void>
  acceptChange: (idx: number) => void
  rejectChange: (idx: number) => void
  setFocusedChange: (idx: number) => void
  prevChange: () => void
  nextChange: () => void
  acceptAllDiff: () => void
  rejectAllDiff: () => void
  applyPendingDiff: (updateContent: (html: string) => void) => Promise<void>
  clearPendingDiff: () => void
}

export const useEditorStore = create<EditorStore>((set, get) => ({
  // Editor reference
  editorRef: null,
  setEditorRef: (editor) => set({ editorRef: editor }),

  getEditorSelection: () => {
    const { editorRef } = get()
    if (!editorRef) return null
    const { selection } = editorRef.state
    if (selection.empty) return null
    const { from, to } = selection
    const text = editorRef.state.doc.textBetween(from, to, '\n')
    return { html: text, text, from, to }
  },

  // Inline AI prompt
  inlinePrompt: null,

  openInlinePrompt: (cursorPos) => {
    set({ inlinePrompt: { active: true, cursorPos } })
  },

  closeInlinePrompt: () => {
    set({ inlinePrompt: null })
  },

  // Diff/review state
  pendingDiff: null,

  setPendingDiff: async ({ diffs, originalHtml }) => {
    // Lazy-load the diff engine so it stays out of the main bundle
    // (AIStudio already imports it dynamically; a static import here
    // would defeat Vite's code-splitting).
    const { extractChanges } = await import('../utils/diff')
    const changes = extractChanges(diffs)
    set({ pendingDiff: { diffs, changes, focusedChangeIdx: 0, originalHtml } })
  },

  acceptChange: (idx) => set(s => {
    if (!s.pendingDiff) return {}
    const changes = s.pendingDiff.changes.map((c, i) => i === idx ? { ...c, accepted: true, decided: true } : c)
    let nextIdx = idx
    const undecidedAfter = changes.findIndex((c, i) => i > idx && !c.decided)
    if (undecidedAfter !== -1) {
      nextIdx = undecidedAfter
    } else {
      const undecidedBefore = changes.findIndex(c => !c.decided)
      if (undecidedBefore !== -1) {
        nextIdx = undecidedBefore
      } else {
        nextIdx = Math.min(idx + 1, changes.length - 1)
      }
    }
    return { pendingDiff: { ...s.pendingDiff, changes, focusedChangeIdx: nextIdx } }
  }),

  rejectChange: (idx) => set(s => {
    if (!s.pendingDiff) return {}
    const changes = s.pendingDiff.changes.map((c, i) => i === idx ? { ...c, accepted: false, decided: true } : c)
    let nextIdx = idx
    const undecidedAfter = changes.findIndex((c, i) => i > idx && !c.decided)
    if (undecidedAfter !== -1) {
      nextIdx = undecidedAfter
    } else {
      const undecidedBefore = changes.findIndex(c => !c.decided)
      if (undecidedBefore !== -1) {
        nextIdx = undecidedBefore
      } else {
        nextIdx = Math.min(idx + 1, changes.length - 1)
      }
    }
    return { pendingDiff: { ...s.pendingDiff, changes, focusedChangeIdx: nextIdx } }
  }),

  setFocusedChange: (idx) => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, focusedChangeIdx: idx } }
    : {}),

  prevChange: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, focusedChangeIdx: Math.max(0, s.pendingDiff.focusedChangeIdx - 1) } }
    : {}),

  nextChange: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, focusedChangeIdx: Math.min(s.pendingDiff.changes.length - 1, s.pendingDiff.focusedChangeIdx + 1) } }
    : {}),

  acceptAllDiff: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, changes: s.pendingDiff.changes.map(c =>
        c.decided ? c : { ...c, accepted: true, decided: true }
      ) } }
    : {}),

  rejectAllDiff: () => set(s => s.pendingDiff
    ? { pendingDiff: { ...s.pendingDiff, changes: s.pendingDiff.changes.map(c =>
        c.decided ? c : { ...c, accepted: false, decided: true }
      ) } }
    : {}),

  applyPendingDiff: async (updateContent) => {
    const { pendingDiff } = get()
    if (!pendingDiff) return
    const { assembleFromChanges } = await import('../utils/diff')
    updateContent(assembleFromChanges(pendingDiff.diffs, pendingDiff.changes))
    set({ pendingDiff: null })
  },

  clearPendingDiff: () => set({ pendingDiff: null }),
}))
