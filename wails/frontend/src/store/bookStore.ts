// Book Store - Core book data, file operations, chapter management, UI state
// Delegates to specialized stores: editorStore, storyBibleStore

import { create } from 'zustand'
import type { BookData, ChapterItem, Character, EvidenceRecord, Metadata, ReadAloudCast, Section, WritingStyleOptions } from '../types/draftline'
import { DEFAULT_STYLE_OPTIONS } from '../types/draftline'
import type { ParagraphDiff, DiffChange } from '../utils/diff'
import { countBookWords } from '../utils/textUtils'

import { NewBook, PickBookPath, SaveBook, SaveBookAs, SaveBookSnapshots, OpenBookAsCopy, ShowInfoDialog, CloseBookFile } from '../../wailsjs/go/main/App'
import { types } from '../../wailsjs/go/models'
import { useAppStore } from './appStore'
import { useEditorStore, type DiffTarget, type EditorInstance, type EditorSelection } from './editorStore'
import { useStoryBibleStore } from './storyBibleStore'
import { createEditionActions, type EditionActions } from './editions'
import { createChapterActions, getSectionArray, newChapterID, type ChapterActions } from './chapters'
import { createCharacterIndexActions, type CharacterIndexActions } from './characterIndex'
import { createSavePipeline, type SaveOutcome } from './bookSave'
import { createBookAdoption } from './bookOpen'

export type { SaveOutcome }
import type { BookLockWarning } from './bookLock'
import { resetChapterHistorySession, saveAIChapterHistory, saveManualChapterSnapshot, scheduleChapterHistory as queueChapterHistory, type ChapterHistoryDependencies } from './chapterHistory'

// Status-bar text lives in appStore (app-level UI state); this is the funnel
// bookStore's save/index flows report through.
const setStatus = (msg: string) => useAppStore.getState().setStatusMessage(msg)

// The save pipeline, with the store wired into it. Destructured so the call
// sites below read the same as when these were plain functions.
const savePipeline = createSavePipeline({
  getBook: () => useBookStore.getState().book,
  getIsDirty: () => useBookStore.getState().isDirty,
  recordSaved: (filePath, clearDirty) => useBookStore.setState(s => ({
    ...(clearDirty && { isDirty: false }),
    book: s.book ? { ...s.book, file_path: filePath } : null,
  })),
  setAutoSaving: (isAutoSaving) => useBookStore.setState({ isAutoSaving }),
  autoSaveEnabled: () => useAppStore.getState().settings.activity_autosave_enabled,
  setStatus,
  getStatusMessage: () => useAppStore.getState().statusMessage,
  onSessionBegin: resetChapterHistorySession,
})
const { performSave, scheduleAutoSave, enqueueSaveTask, beginBookSession } = savePipeline

// Getting a book on screen, with the store wired into it.
const { loadBookFromPath, adoptBook, importExternalBook } = createBookAdoption({
  setLockWarning: (bookLockWarning) => useBookStore.setState(s => ({ dialogs: { ...s.dialogs, bookLockWarning } })),
  setOpening: (isOpening) => useBookStore.setState({ isOpening }),
  placeBook: (book, currentSection) => useBookStore.setState({
    book, currentSection, currentIndex: 0, isDirty: false, analysisRevision: 0,
  }),
  beginSession: () => beginBookSession(),
  setStatus,
  clearPendingDiff: () => useEditorStore.getState().clearPendingDiff(),
  rememberRecent: (book, path) => {
    // The word count comes from metadata (computed by Go on open/save);
    // counting in JS is only a fallback for books that predate the field.
    const wordCount = book.metadata.word_count || countBookWords(book)
    const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
    void useAppStore.getState().addRecentProject(types.RecentProject.createFrom({
      type: 'book', path: book.file_path || path, name: book.metadata.title || 'Untitled',
      lastOpened: new Date().toISOString(), stats: { chapters: chapterCount, words: wordCount }
    }))
  },
  loadImported: (book) => useBookStore.getState().loadImportedBook(book),
})

function ensureFrontendChapterIDs(book: BookData): BookData {
  const assign = (items: ChapterItem[]) => items.map(item => ({ ...item, id: item.id || newChapterID() }))
  return {
    ...book,
    front_matter: assign(book.front_matter),
    body: assign(book.body),
    back_matter: assign(book.back_matter),
  }
}

function chapterHistoryDependencies(): ChapterHistoryDependencies {
  return {
    getBook: () => useBookStore.getState().book,
    getSession: savePipeline.currentSession,
    getRevision: savePipeline.currentRevision,
    enqueue: enqueueSaveTask,
    setStatus,
    onSaved: (filePath, revision) => useBookStore.setState(current => ({
      isDirty: savePipeline.currentRevision() === revision ? false : current.isDirty,
      book: current.book ? { ...current.book, file_path: filePath } : null,
    })),
  }
}

function scheduleChapterHistory(chapterID?: string) {
  queueChapterHistory(chapterID, chapterHistoryDependencies())
}

// Only the dialogs entangled with the save pipeline live here; self-contained
// dialogs (metadata, new chapter, export, chapter history) are in appStore.
interface DialogState {
  showUnsavedWarning: boolean
  // 'open' = show the file picker; the object forms carry the specific file
  // (recent projects, OS file associations) through the unsaved-changes flow.
  pendingAction: 'new' | 'open' | { openPath: string } | { importPath: string } | null
  showNewBookWizard: boolean
  // Set when a book about to be opened carries a claim from another device.
  // Nothing is blocked; the author is told and chooses. See internal/booklock.
  bookLockWarning: BookLockWarning | null
}

interface BookStore extends EditionActions, ChapterActions, CharacterIndexActions {
  // Core state
  book: BookData | null
  currentSection: Section
  currentIndex: number
  isDirty: boolean
  isAutoSaving: boolean
  // True while a book archive is being opened and parsed; drives the
  // full-screen opening overlay and guards against concurrent opens.
  isOpening: boolean
  isIndexing: boolean
  analysisRevision: number

  // UI state
  dialogs: DialogState

  // Workspace view: the editor, or the full-screen Cast view
  viewMode: 'editor' | 'cast' | 'planner'
  setViewMode: (mode: 'editor' | 'cast' | 'planner') => void

  // File operations
  newBook: () => Promise<void>
  openBook: () => Promise<void>
  openRecentBook: (path: string) => Promise<void>
  // OS file associations / "Open with": routes .draftline to the open flow
  // and .epub/.docx to the importers, honoring the unsaved-changes dialog.
  openExternalFile: (path: string) => Promise<void>
  saveBook: () => Promise<void>
  saveBookAs: () => Promise<void>
  restoreChapterHistory: (content: string) => Promise<boolean>
  snapshotCurrentChapter: (reason?: string) => Promise<boolean> // writer-initiated checkpoint; true once stored
  closeProject: () => Promise<void>

  // Direct book update (for analysis results, etc.)
  updateBook: (book: BookData) => void
  updateEvidenceRecord: (recordID: string, changes: Partial<EvidenceRecord>) => void
  setContinuityDecision: (signalID: string, status: 'reviewed' | 'dismissed' | null) => void

  // Navigation
  setCurrentChapter: (section: Section, index: number) => void

  // Chapter and record mutations come from ChapterActions (see chapters.ts).

  // Story Bible (delegates to storyBibleStore)
  addCharacter: (char: Character) => void
  updateCharacter: (char: Character) => void
  deleteCharacter: (id: string) => void
  highlightedCharacterId: string | null
  setHighlightedCharacter: (id: string | null) => void
  getHighlightedCharacterNames: () => string[]

  // Writing goals & style
  updateWritingGoals: (goals: Partial<{ target_word_count: number; daily_word_goal: number; words_today: number; last_writing_date: string }>) => void
  updateStyleOptions: (options: Partial<WritingStyleOptions>) => void
  getStyleOptions: () => WritingStyleOptions

  // Indexing and entity correction come from CharacterIndexActions.

  // UI actions
  closeUnsavedWarning: () => void
  // Answers to "this book may be open on another device". Opening anyway is
  // allowed on purpose: the warning is a guess, and the author knows things
  // the sidecar does not — that the laptop is shut, that it was them.
  openBookAnyway: () => Promise<void>
  openBookAsCopy: () => Promise<void>
  cancelBookLockWarning: () => void
  saveAndProceed: () => Promise<void>
  discardAndProceed: () => Promise<void>
  initBook: () => Promise<void>
  confirmNewBook: (title: string, author: string, publisher: string) => Promise<void>
  loadImportedBook: (book: BookData) => void
  cancelNewBookWizard: () => void

  // Editor store bridge (for backwards compatibility)
  editorRef: EditorInstance | null
  setEditorRef: (editor: EditorInstance | null) => void
  getEditorSelection: () => EditorSelection | null
  pendingDiff: {
    diffs: ParagraphDiff[]
    changes: DiffChange[]
    focusedChangeIdx: number
    originalHtml: string
    target: DiffTarget
    historyReason?: string
    applyError?: string
  } | null
  setPendingDiff: (payload: { diffs: ParagraphDiff[]; originalHtml: string; target?: DiffTarget; historyReason?: string }) => void
  acceptChange: (idx: number) => void
  rejectChange: (idx: number) => void
  setFocusedChange: (idx: number) => void
  prevChange: () => void
  nextChange: () => void
  acceptAllDiff: () => void
  rejectAllDiff: () => void
  applyPendingDiff: () => Promise<void>
  clearPendingDiff: () => void
}

// Direct import for OS "Open with" on .epub/.docx — same importers the
// New Book wizard uses, minus the file picker and preview step.
// Shared tail of the unsaved-changes dialog: run whatever the user was
// trying to do before the dialog interrupted.
async function proceedWithAction(action: DialogState['pendingAction']) {
  if (action === 'new') {
    useBookStore.setState(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: true } }))
    return
  }
  if (action && typeof action === 'object') {
    if ('openPath' in action) {
      await useBookStore.getState().openRecentBook(action.openPath)
    } else {
      await importExternalBook(action.importPath)
    }
    return
  }
  if (action === 'open') {
    try {
      const path = await PickBookPath()
      if (!path) return
      await loadBookFromPath(path)
    } catch (e) {
      setStatus(`Error opening file: ${e}`)
    }
  }
}

export function getGlobalChapterIndex(book: BookData | null, section: Section, index: number): number {
  if (!book || section === 'copyright') return -1
  if (section === 'front_matter') return index
  if (section === 'body') return book.front_matter.length + index
  if (section === 'back_matter') return book.front_matter.length + book.body.length + index
  return -1
}

export const useBookStore = create<BookStore>((set, get) => ({
  // Core state
  book: null,
  currentSection: 'body',
  currentIndex: 0,
  isDirty: false,
  isAutoSaving: false,
  isOpening: false,
  isIndexing: false,
  analysisRevision: 0,

  // UI state
  dialogs: { showUnsavedWarning: false, pendingAction: null, showNewBookWizard: false, bookLockWarning: null },

  viewMode: 'editor',
  setViewMode: (mode) => set({ viewMode: mode }),

  // Editor store bridge - delegate to editorStore
  get editorRef() { return useEditorStore.getState().editorRef },
  setEditorRef: (editor) => useEditorStore.getState().setEditorRef(editor),
  getEditorSelection: () => useEditorStore.getState().getEditorSelection(),
  get pendingDiff() { return useEditorStore.getState().pendingDiff },
  setPendingDiff: (payload) => useEditorStore.getState().setPendingDiff(payload),
  acceptChange: (idx) => useEditorStore.getState().acceptChange(idx),
  rejectChange: (idx) => useEditorStore.getState().rejectChange(idx),
  setFocusedChange: (idx) => useEditorStore.getState().setFocusedChange(idx),
  prevChange: () => useEditorStore.getState().prevChange(),
  nextChange: () => useEditorStore.getState().nextChange(),
  acceptAllDiff: () => useEditorStore.getState().acceptAllDiff(),
  rejectAllDiff: () => useEditorStore.getState().rejectAllDiff(),
  applyPendingDiff: async () => {
    const before = get()
    const section = before.currentSection
    const index = before.currentIndex
    const chapter = section === 'copyright' || !before.book
      ? null
      : getSectionArray(before.book, section)[index]
    const applied = await useEditorStore.getState().applyPendingDiff(get().updateCurrentContent)
    if (!applied || !chapter || section === 'copyright') return
    await saveAIChapterHistory({
      chapter,
      section,
      beforeHtml: applied.beforeHtml,
      afterHtml: applied.afterHtml,
      reason: applied.historyReason || 'AI edit',
    }, chapterHistoryDependencies())
  },
  clearPendingDiff: () => useEditorStore.getState().clearPendingDiff(),

  // File operations
  newBook: async () => {
    const { isDirty } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'new' } }))
      return
    }
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: true } }))
  },

  openBook: async () => {
    const { isDirty, isOpening } = get()
    if (isOpening) return
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'open' } }))
      return
    }
    try {
      const path = await PickBookPath()
      if (!path) return
      await loadBookFromPath(path)
    } catch (e) {
      setStatus(`Error opening file: ${e}`)
    }
  },

  openExternalFile: async (path: string) => {
    const ext = (path.split('.').pop() || '').toLowerCase()
    if (ext === 'draftline') {
      await get().openRecentBook(path)
      return
    }
    if (ext === 'storiverse') {
      setStatus('Storiverse universes are not supported in this version')
      void ShowInfoDialog(
        'Storiverse universe',
        'This is a Storiverse universe file. Universe support is coming in a later version of Draftline — update to a version that supports Storiverse to open it.',
      )
      return
    }
    if (ext !== 'epub' && ext !== 'docx') return
    if (get().isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: { importPath: path } } }))
      return
    }
    await importExternalBook(path)
  },

  openRecentBook: async (path: string) => {
    const { isDirty, isOpening } = get()
    if (isOpening) return
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: { openPath: path } } }))
      return
    }
    await loadBookFromPath(path)
  },

  saveBook: async () => {
    // Whatever is still in the editor belongs in this save.
    useEditorStore.getState().drainPendingEdit()
    if (!get().book) return
    const outcome = await performSave('save')
    if (outcome.status === 'saved') {
      setStatus(outcome.warning || `Saved: ${outcome.filePath}`)
    } else if (outcome.status === 'stale') {
      setStatus('Newer edits were made while saving — save again')
    } else if (outcome.status === 'error') {
      setStatus(`Save failed: ${outcome.message}`)
    }
  },

  saveBookAs: async () => {
    useEditorStore.getState().drainPendingEdit()
    if (!get().book) return
    const outcome = await performSave('saveAs')
    if (outcome.status === 'saved') {
      setStatus(outcome.warning || `Saved: ${outcome.filePath}`)
    } else if (outcome.status === 'stale') {
      setStatus('Newer edits were made while saving — save again')
    } else if (outcome.status === 'error') {
      setStatus(`Save failed: ${outcome.message}`)
    }
    // 'cancelled' (user dismissed the picker) stays silent.
  },

  restoreChapterHistory: async (content) => {
    const { book, currentSection, currentIndex } = get()
    if (!book || currentSection === 'copyright') return false
    const chapter = getSectionArray(book, currentSection)[currentIndex]
    if (!chapter?.id) return false
    if (book.file_path) {
      const session = savePipeline.currentSession()
      const request = [{ chapter_id: chapter.id, section: currentSection, chapter_title: chapter.title, content: chapter.content, reason: 'Before history restore' }]
      const result = await enqueueSaveTask(async () => {
        const latest = useBookStore.getState().book
        if (savePipeline.currentSession() !== session || !latest) return { success: false, file_path: '', error: 'project changed' }
        return SaveBookSnapshots(latest as any, request as any)
      })
      if (savePipeline.currentSession() !== session || !result.success) {
        setStatus(`Could not create restore checkpoint: ${result.error || 'unknown error'}`)
        return false
      }
    }
    get().updateCurrentContent(content)
    setStatus('Previous chapter version restored — save to keep it')
    return true
  },

  snapshotCurrentChapter: (reason) =>
    saveManualChapterSnapshot(reason, get, useEditorStore.getState().editorRef?.getHTML(), chapterHistoryDependencies()),

  closeProject: async () => {
    // A close reads isDirty, so the editor has to have handed over first.
    useEditorStore.getState().drainPendingEdit()
    const { book, isDirty } = get()
    if (book && isDirty) {
      const outcome = await performSave('save')
      if (outcome.status === 'error') {
        setStatus(`Save failed: ${outcome.message} — project not closed`)
        return
      }
      if (outcome.status === 'cancelled') {
        setStatus('Save cancelled — project not closed')
        return
      }
      if (outcome.status !== 'saved') {
        setStatus('Newer edits are still unsaved — project not closed')
        return
      }
    }
    beginBookSession()
    set({ book: null, currentSection: 'body', currentIndex: 0, isDirty: false, analysisRevision: 0 })
    setStatus('')
    useEditorStore.getState().clearPendingDiff()
    // Free this book's instance lock so another Draftline window can open
    // it. Best-effort fire-and-forget: closing must never block on it.
    void Promise.resolve(CloseBookFile()).catch(() => {})
    useAppStore.getState().setShowWelcome(true)
  },

  // Direct book update (for analysis results, etc.)
  updateBook: (book) => {
    set({ book, isDirty: true })
    scheduleAutoSave()
  },

  updateEvidenceRecord: (recordID, changes) => {
    const { book } = get()
    const evidence = book?.analysis?.evidence
    if (!book || !evidence) return
    const records = evidence.records.map(record => record.id === recordID ? { ...record, ...changes } : record)
    set({
      book: {
        ...book,
        analysis: { ...book.analysis, evidence: { ...evidence, records } },
      },
      isDirty: true,
    })
    scheduleAutoSave()
  },

  // Continuity questions are rebuilt from the manuscript every time, so only
  // the author's decision is stored. Passing null clears a decision, which
  // returns the question to the outstanding queue.
  setContinuityDecision: (signalID, status) => {
    const { book } = get()
    if (!book) return
    const existing = book.analysis?.continuity?.decisions ?? []
    const without = existing.filter(decision => decision.signal_id !== signalID)
    const decisions = status
      ? [...without, { signal_id: signalID, status, decided_at: new Date().toISOString() }]
      : without
    set({
      book: {
        ...book,
        analysis: { ...book.analysis, continuity: { ...book.analysis?.continuity, decisions } },
      },
      isDirty: true,
    })
    scheduleAutoSave()
  },

  // Navigation
  setCurrentChapter: (section, index) => {
    set({ currentSection: section, currentIndex: index })
    useEditorStore.getState().clearPendingDiff()
  },

  ...createChapterActions({
    read: () => {
      const { book, currentSection, currentIndex } = get()
      return { book, currentSection, currentIndex }
    },
    apply: (patch) => set(state => ({
      book: patch.book,
      isDirty: true,
      ...(patch.currentSection !== undefined && { currentSection: patch.currentSection }),
      ...(patch.currentIndex !== undefined && { currentIndex: patch.currentIndex }),
      ...(patch.prose && { analysisRevision: state.analysisRevision + 1 }),
    })),
    noteHistory: scheduleChapterHistory,
    autosave: scheduleAutoSave,
  }),
  ...createEditionActions(() => get().book, book => { set({ book, isDirty: true }); scheduleAutoSave() }),

  // Story Bible - delegate to storyBibleStore
  addCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().addCharacter(book, char)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  updateCharacter: (char) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().updateCharacter(book, char)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  deleteCharacter: (id) => {
    const { book } = get()
    if (!book) return
    const updated = useStoryBibleStore.getState().deleteCharacter(book, id)
    set({ book: updated, isDirty: true })
    scheduleAutoSave()
  },

  get highlightedCharacterId() { return useStoryBibleStore.getState().highlightedCharacterId },
  setHighlightedCharacter: (id) => useStoryBibleStore.getState().setHighlightedCharacter(id),
  getHighlightedCharacterNames: () => useStoryBibleStore.getState().getHighlightedCharacterNames(get().book),

  // Writing goals & style
  updateWritingGoals: (goals) => {
    const { book } = get()
    if (!book) return
    const existing = book.writing_goals ?? { target_word_count: 0, daily_word_goal: 0, words_today: 0, last_writing_date: '' }
    set({ book: { ...book, writing_goals: { ...existing, ...goals } }, isDirty: true })
    scheduleAutoSave()
  },

  updateStyleOptions: (options) => {
    const { book } = get()
    if (!book) return
    const existing = book.style_options ?? DEFAULT_STYLE_OPTIONS
    set({ book: { ...book, style_options: { ...existing, ...options } }, isDirty: true })
    scheduleAutoSave()
  },

  getStyleOptions: () => {
    const { book } = get()
    return book?.style_options ?? DEFAULT_STYLE_OPTIONS
  },

  // Indexing - runs the two-phase character pipeline (mention extraction +
  // entity resolution). The backend returns the full updated book, including
  // analysis.entity_resolution which relationship analysis depends on.
  ...createCharacterIndexActions({
    getBook: () => get().book,
    commit: (book, options) => set(state => ({
      book,
      isDirty: true,
      ...(options?.prose && { analysisRevision: state.analysisRevision + 1 }),
    })),
    setIndexing: (isIndexing) => set({ isIndexing }),
    setStatus,
    autosave: scheduleAutoSave,
  }),
  // UI actions

  closeUnsavedWarning: () => set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null } })),

  openBookAnyway: async () => {
    const warning = get().dialogs.bookLockWarning
    if (!warning) return
    set(s => ({ dialogs: { ...s.dialogs, bookLockWarning: null } }))
    await loadBookFromPath(warning.path, true)
  },

  openBookAsCopy: async () => {
    const warning = get().dialogs.bookLockWarning
    if (!warning) return
    set(s => ({ dialogs: { ...s.dialogs, bookLockWarning: null } }))
    await adoptBook(() => OpenBookAsCopy(warning.path), warning.path)
  },

  cancelBookLockWarning: () => set(s => ({ dialogs: { ...s.dialogs, bookLockWarning: null } })),

  initBook: async () => {
    try {
      const book: BookData = await NewBook()
      beginBookSession()
      set({ book, currentSection: 'body', currentIndex: 0, isDirty: false, analysisRevision: 0 })
      setStatus('Ready')
    } catch (e) {
      console.error('initBook failed:', e)
    }
  },

  confirmNewBook: async (title, author, publisher) => {
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: false } }))
    try {
      const book: BookData = await NewBook()
      const merged: BookData = { ...book, metadata: { ...book.metadata, title: title || 'Untitled', author, publisher } }
      beginBookSession()
      set({ book: merged, currentSection: 'body', currentIndex: 0, isDirty: false, analysisRevision: 0 })
      setStatus('New project created')
    } catch (e) {
      setStatus(`Error: ${e}`)
    }
  },

  loadImportedBook: (book: BookData) => {
    set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: false } }))
    const section: Section = book.body.length > 0 ? 'body' : book.front_matter.length > 0 ? 'front_matter' : 'back_matter'
    beginBookSession()
    const identifiedBook = ensureFrontendChapterIDs(book)
    set({ book: identifiedBook, currentSection: section, currentIndex: 0, isDirty: true, analysisRevision: 0 })
    setStatus(`Imported: ${book.metadata.title}`)
    scheduleAutoSave()
    useAppStore.getState().setShowWelcome(false)
  },

  cancelNewBookWizard: () => set(s => ({ dialogs: { ...s.dialogs, showNewBookWizard: false } })),


  saveAndProceed: async () => {
    useEditorStore.getState().drainPendingEdit()
    // The dialog stays open until the save actually succeeds: a failed or
    // cancelled save must not let the pending new/open action discard the book.
    const { book, dialogs } = get()
    const action = dialogs.pendingAction
    if (book) {
      const outcome = await performSave('save')
      if (outcome.status === 'error') {
        setStatus(`Save failed: ${outcome.message}`)
        return // dialog stays open, pendingAction retained
      }
      if (outcome.status === 'cancelled') {
        return // dismissing the SaveAs picker is not consent to discard
      }
      if (outcome.status !== 'saved') {
        setStatus('Newer edits are still unsaved — action cancelled')
        return
      }
      setStatus(`Saved: ${outcome.filePath}`)
    }
    set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null } }))
    await proceedWithAction(action)
  },

  discardAndProceed: async () => {
    const { dialogs } = get()
    const action = dialogs.pendingAction
    beginBookSession()
    set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null }, isDirty: false }))
    await proceedWithAction(action)
  },
}))
