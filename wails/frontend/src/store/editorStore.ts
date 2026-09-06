// Editor Store - Editor reference, selection, inline prompt, diff/review state

import { create } from 'zustand'
import { getHTMLFromFragment } from '@tiptap/core'
import type { ParagraphDiff, DiffChange } from '../utils/diff'

// Invalidates async diff operations when navigation or a newer AI result
// replaces/clears the review while the lazily loaded diff module is in flight.
let pendingDiffGeneration = 0

// Editor instance type (minimal interface for selection access)
export interface EditorInstance {
  state: {
    selection: {
      from: number
      to: number
      empty: boolean
      $from: { parentOffset: number; parent: { content: { size: number } } }
      $to: { parentOffset: number; parent: { content: { size: number } } }
      content: () => { content: Parameters<typeof getHTMLFromFragment>[0] }
    }
    schema: Parameters<typeof getHTMLFromFragment>[1]
    doc: { textBetween: (from: number, to: number, separator: string) => string }
  }
  getHTML: () => string
  view: { state: { selection: { from: number; to: number } } }
  commands: {
    setTextSelection: (range: { from: number; to: number }) => boolean
    insertContentAt: (range: { from: number; to: number }, content: string) => boolean
  }
}

export interface EditorSelection {
  html: string
  text: string
  from: number
  to: number
  documentHtml: string
  sameTextBlock: boolean
  startsAtTextBlockBoundary: boolean
  endsAtTextBlockBoundary: boolean
}

export type DiffTarget =
  | { kind: 'chapter' }
  | {
      kind: 'selection'
      from: number
      to: number
      sourceDocumentHtml: string
      sameTextBlock?: boolean
      startsAtTextBlockBoundary?: boolean
      endsAtTextBlockBoundary?: boolean
    }

interface SelectionReplacement {
  range: { from: number; to: number }
  content: string
}

/** Adapt block-wrapped AI HTML to the shape of the original TipTap selection. */
export function prepareSelectionReplacement(target: Extract<DiffTarget, { kind: 'selection' }>, revisedHtml: string): SelectionReplacement {
  // TipTap serializes even an inline selection as <p>...</p>. Reinserting that
  // block inside its original paragraph splits the paragraph and can leave an
  // empty paragraph at either boundary. Keep a one-paragraph inline edit inline.
  const trimmed = revisedHtml.trim()
  const paragraphBlocks = trimmed.match(/<p(?:\s[^>]*)?>[\s\S]*?<\/p>/gi)
  const singleParagraph = paragraphBlocks?.length === 1
    ? trimmed.match(/^<p(?:\s[^>]*)?>([\s\S]*)<\/p>$/i)
    : null
  if (target.sameTextBlock && singleParagraph) {
    return { range: { from: target.from, to: target.to }, content: singleParagraph[1] }
  }

  // For a multi-block edit, consume wrappers that the original selection
  // covered completely so they cannot survive as empty paragraphs.
  return {
    range: {
      from: target.startsAtTextBlockBoundary ? Math.max(0, target.from - 1) : target.from,
      to: target.endsAtTextBlockBoundary ? target.to + 1 : target.to,
    },
    content: revisedHtml,
  }
}

export interface AppliedDiff {
  beforeHtml: string
  afterHtml: string
  historyReason?: string
}

interface EditorStore {
  // Editor reference
  editorRef: EditorInstance | null
  setEditorRef: (editor: EditorInstance | null) => void
  getEditorSelection: () => EditorSelection | null

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
    target: DiffTarget
    historyReason?: string
    applyError?: string
  } | null
  setPendingDiff: (payload: { diffs: ParagraphDiff[]; originalHtml: string; target?: DiffTarget; historyReason?: string }) => Promise<void>
  acceptChange: (idx: number) => void
  rejectChange: (idx: number) => void
  setFocusedChange: (idx: number) => void
  prevChange: () => void
  nextChange: () => void
  acceptAllDiff: () => void
  rejectAllDiff: () => void
  applyPendingDiff: (updateContent: (html: string) => void) => Promise<AppliedDiff | null>
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
    const html = getHTMLFromFragment(selection.content().content, editorRef.state.schema)
    return {
      html,
      text,
      from,
      to,
      documentHtml: editorRef.getHTML(),
      sameTextBlock: selection.$from.parent === selection.$to.parent,
      startsAtTextBlockBoundary: selection.$from.parentOffset === 0,
      endsAtTextBlockBoundary: selection.$to.parentOffset === selection.$to.parent.content.size,
    }
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

  setPendingDiff: async ({ diffs, originalHtml, target = { kind: 'chapter' }, historyReason }) => {
    const generation = ++pendingDiffGeneration
    // Lazy-load the diff engine so it stays out of the main bundle
    // (AIStudio already imports it dynamically; a static import here
    // would defeat Vite's code-splitting).
    const { extractChanges } = await import('../utils/diff')
    if (pendingDiffGeneration !== generation) return
    const changes = extractChanges(diffs)
    set({ pendingDiff: { diffs, changes, focusedChangeIdx: 0, originalHtml, target, historyReason } })
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
    if (!get().pendingDiff) return null
    const generation = pendingDiffGeneration
    const { assembleFromChanges } = await import('../utils/diff')
    if (pendingDiffGeneration !== generation) return null
    const { pendingDiff } = get()
    if (!pendingDiff) return null

    const accepted = pendingDiff.changes.some(change => change.accepted)
    if (!accepted) {
      pendingDiffGeneration++
      set({ pendingDiff: null })
      return null
    }

    const revisedHtml = assembleFromChanges(pendingDiff.diffs, pendingDiff.changes)
    let beforeHtml: string
    let afterHtml: string

    if (pendingDiff.target.kind === 'selection') {
      const editor = get().editorRef
      if (!editor) {
        set({ pendingDiff: { ...pendingDiff, applyError: 'The editor is unavailable. No text was changed.' } })
        return null
      }
      beforeHtml = editor.getHTML()
      if (beforeHtml !== pendingDiff.target.sourceDocumentHtml) {
        set({ pendingDiff: { ...pendingDiff, applyError: 'The chapter changed after this AI pass started. No text was changed.' } })
        return null
      }
      const replacement = prepareSelectionReplacement(pendingDiff.target, revisedHtml)
      const applied = editor.commands.insertContentAt(replacement.range, replacement.content)
      if (!applied) {
        set({ pendingDiff: { ...pendingDiff, applyError: 'Draftline could not replace the selected range. No text was changed.' } })
        return null
      }
      afterHtml = editor.getHTML()
      // Keep the book model synchronized immediately. The same replacement is
      // one TipTap transaction, so Ctrl+Z can still undo it in the live editor.
      updateContent(afterHtml)
    } else {
      beforeHtml = pendingDiff.originalHtml
      afterHtml = revisedHtml
      updateContent(afterHtml)
    }
    pendingDiffGeneration++
    set({ pendingDiff: null })
    return { beforeHtml, afterHtml, historyReason: pendingDiff.historyReason }
  },

  clearPendingDiff: () => {
    pendingDiffGeneration++
    set({ pendingDiff: null })
  },
}))
