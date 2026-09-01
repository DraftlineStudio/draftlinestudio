// Book Store - Core book data, file operations, chapter management, UI state
// Delegates to specialized stores: editorStore, storyBibleStore

import { create } from 'zustand'
import type { BookData, ChapterItem, Character, EvidenceRecord, Metadata, Section, WritingStyleOptions } from '../types/draftline'
import { DEFAULT_STYLE_OPTIONS } from '../types/draftline'
import type { ParagraphDiff, DiffChange } from '../utils/diff'
import { countBookWords } from '../utils/textUtils'

import { NewBook, OpenBookDialog, SaveBook, SaveBookAs, SaveBookSnapshots, OpenRecentProject, AddRecentProject, IndexBook, MergeEntities, SplitEntity, ImportEPUB, ImportDOCX, ShowInfoDialog } from '../../wailsjs/go/main/App'
import { types } from '../../wailsjs/go/models'
import { useAppStore } from './appStore'
import { useEditorStore, type EditorInstance } from './editorStore'
import { useStoryBibleStore } from './storyBibleStore'

// Status-bar text lives in appStore (app-level UI state); this is the funnel
// bookStore's save/index flows report through.
const setStatus = (msg: string) => useAppStore.getState().setStatusMessage(msg)

// Auto-save debounce timer
let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
const AUTO_SAVE_DELAY = 5000
const HISTORY_SNAPSHOT_DELAY = 10 * 60 * 1000
let historySnapshotTimer: ReturnType<typeof setTimeout> | null = null
const changedChapterIDs = new Set<string>()

// Save serialization + staleness tracking (module-scoped, not reactive state).
// saveRevision is bumped by scheduleAutoSave() — the funnel every dirty-marking
// mutation already calls — so a save that completes after further edits knows
// its snapshot is stale and must not clear isDirty.
let saveRevision = 0
// Identifies the currently open project. A save may finish after the user has
// discarded that project and opened another one; its result must never mutate
// the replacement project's path or dirty state.
let bookSession = 0
// saveChain serializes all saves (manual, autosave, close, save-and-proceed)
// so two SaveBook calls can never interleave on the Go side.
let saveChain: Promise<unknown> = Promise.resolve()

export type SaveOutcome =
  | { status: 'saved'; filePath: string }
  | { status: 'stale'; filePath: string }
  | { status: 'superseded' }
  | { status: 'cancelled' }
  | { status: 'error'; message: string }

function beginBookSession() {
  bookSession++
  if (autoSaveTimer) {
    clearTimeout(autoSaveTimer)
    autoSaveTimer = null
  }
  if (historySnapshotTimer) {
    clearTimeout(historySnapshotTimer)
    historySnapshotTimer = null
  }
  changedChapterIDs.clear()
}

function newChapterID(): string {
  return typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? `ch-${crypto.randomUUID().replace(/-/g, '')}`
    : `ch-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

function ensureFrontendChapterIDs(book: BookData): BookData {
  const assign = (items: ChapterItem[]) => items.map(item => ({ ...item, id: item.id || newChapterID() }))
  return {
    ...book,
    front_matter: assign(book.front_matter),
    body: assign(book.body),
    back_matter: assign(book.back_matter),
  }
}

// Shared save primitive. Chains on saveChain, snapshots the book AFTER the
// prior save completes (so a queued save always writes the newest state),
// and clears isDirty only if no edit happened while the save was in flight.
// Always writes result.file_path back into the book (autosave previously lost it).
function performSave(kind: 'save' | 'saveAs'): Promise<SaveOutcome> {
  const run = saveChain.then(async (): Promise<SaveOutcome> => {
    const book = useBookStore.getState().book
    if (!book) return { status: 'error', message: 'no book open' }
    const rev = saveRevision
    const session = bookSession
    try {
      const result = kind === 'saveAs' ? await SaveBookAs(book as any) : await SaveBook(book as any)
      if (bookSession !== session) return { status: 'superseded' }
      if (result.success) {
        if (saveRevision === rev) {
          useBookStore.setState(s => ({ isDirty: false, book: s.book ? { ...s.book, file_path: result.file_path } : null }))
          return { status: 'saved', filePath: result.file_path }
        } else {
          // Edited while saving: keep isDirty so the re-armed autosave persists the newer state.
          useBookStore.setState(s => ({ book: s.book ? { ...s.book, file_path: result.file_path } : null }))
          return { status: 'stale', filePath: result.file_path }
        }
      }
      if (result.error === 'cancelled') return { status: 'cancelled' }
      return { status: 'error', message: result.error || 'unknown error' }
    } catch (e) {
      return { status: 'error', message: String(e) }
    }
  })
  saveChain = run.catch(() => {})
  return run
}

function scheduleAutoSave() {
  saveRevision++
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  if (!useAppStore.getState().settings.activity_autosave_enabled) {
    autoSaveTimer = null
    return
  }
  autoSaveTimer = setTimeout(async () => {
    if (!useAppStore.getState().settings.activity_autosave_enabled) return
    const state = useBookStore.getState()
    if (state.book && state.isDirty && state.book.file_path) {
      const session = bookSession
      useBookStore.setState({ isAutoSaving: true })
      const outcome = await performSave('save')
      if (bookSession !== session) return
      if (outcome.status === 'saved') {
        useBookStore.setState({ isAutoSaving: false })
        setStatus('Auto-saved')
        setTimeout(() => {
          if (useAppStore.getState().statusMessage === 'Auto-saved') {
            setStatus('')
          }
        }, 2000)
      } else {
        useBookStore.setState({ isAutoSaving: false })
      }
    }
  }, AUTO_SAVE_DELAY)
}

function scheduleChapterHistory(chapterID?: string) {
  if (!chapterID || !useAppStore.getState().settings.activity_autosave_enabled) return
  changedChapterIDs.add(chapterID)
  if (historySnapshotTimer) return
  historySnapshotTimer = setTimeout(() => {
    historySnapshotTimer = null
    if (!useAppStore.getState().settings.activity_autosave_enabled) {
      changedChapterIDs.clear()
      return
    }
    const ids = new Set(changedChapterIDs)
    changedChapterIDs.clear()
    const state = useBookStore.getState()
    const book = state.book
    if (!book?.file_path || ids.size === 0) return
    const requests = (['front_matter', 'body', 'back_matter'] as const).flatMap(section =>
      book[section]
        .filter(chapter => chapter.id && ids.has(chapter.id))
        .map(chapter => ({
          chapter_id: chapter.id!,
          section,
          chapter_title: chapter.title,
          content: chapter.content,
          reason: 'Writing session',
        })),
    )
    if (requests.length === 0) return
    const session = bookSession
    const run = saveChain.then(async () => {
      if (bookSession !== session || !useAppStore.getState().settings.activity_autosave_enabled) return
      const latestBook = useBookStore.getState().book
      if (!latestBook) return
      try {
        const result = await SaveBookSnapshots(latestBook as any, requests as any)
        if (bookSession !== session) return
        if (result.success) {
          setStatus('Chapter history updated')
        } else {
          ids.forEach(id => changedChapterIDs.add(id))
          setStatus(`Chapter history failed: ${result.error || 'unknown error'}`)
        }
      } catch (e) {
        if (bookSession === session) {
          ids.forEach(id => changedChapterIDs.add(id))
          setStatus(`Chapter history failed: ${String(e)}`)
        }
      }
    })
    saveChain = run.catch(() => {})
  }, HISTORY_SNAPSHOT_DELAY)
}

// Only the dialogs entangled with the save pipeline live here; self-contained
// dialogs (metadata, new chapter, export, chapter history) are in appStore.
interface DialogState {
  showUnsavedWarning: boolean
  // 'open' = show the file picker; the object forms carry the specific file
  // (recent projects, OS file associations) through the unsaved-changes flow.
  pendingAction: 'new' | 'open' | { openPath: string } | { importPath: string } | null
  showNewBookWizard: boolean
}

interface BookStore {
  // Core state
  book: BookData | null
  currentSection: Section
  currentIndex: number
  isDirty: boolean
  isAutoSaving: boolean
  isIndexing: boolean
  analysisRevision: number

  // UI state
  dialogs: DialogState

  // Workspace view: the editor, or the full-screen Cast view
  viewMode: 'editor' | 'cast'
  setViewMode: (mode: 'editor' | 'cast') => void

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
  closeProject: () => Promise<void>

  // Direct book update (for analysis results, etc.)
  updateBook: (book: BookData) => void
  updateEvidenceRecord: (recordID: string, changes: Partial<EvidenceRecord>) => void

  // Navigation
  setCurrentChapter: (section: Section, index: number) => void

  // Chapter operations
  updateCurrentContent: (html: string) => void
  updateChapterTitle: (section: Section, index: number, title: string) => void
  updateChapterSubtitle: (section: Section, index: number, subtitle: string) => void
  addChapter: (section: Section, item: ChapterItem) => void
  deleteChapter: (section: Section, index: number) => void
  moveChapter: (section: Section, from: number, to: number) => void
  updateMetadata: (metadata: Partial<Metadata>) => void
  updateCopyright: (html: string) => void

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

  // Indexing and entity correction
  indexBook: () => Promise<void>
  mergeEntities: (entityIds: string[], canonical: string) => Promise<boolean>
  splitEntity: (entityId: string, mentionIds: string[], newCanonical: string) => Promise<boolean>
  clearAllCharacters: () => void

  // UI actions
  closeUnsavedWarning: () => void
  saveAndProceed: () => Promise<void>
  discardAndProceed: () => Promise<void>
  initBook: () => Promise<void>
  confirmNewBook: (title: string, author: string, publisher: string) => Promise<void>
  loadImportedBook: (book: BookData) => void
  cancelNewBookWizard: () => void

  // Editor store bridge (for backwards compatibility)
  editorRef: EditorInstance | null
  setEditorRef: (editor: EditorInstance | null) => void
  getEditorSelection: () => { html: string; text: string; from: number; to: number } | null
  inlinePrompt: { active: boolean; cursorPos: number } | null
  openInlinePrompt: (cursorPos: number) => void
  closeInlinePrompt: () => void
  pendingDiff: {
    diffs: ParagraphDiff[]
    changes: DiffChange[]
    focusedChangeIdx: number
    originalHtml: string
  } | null
  setPendingDiff: (payload: { diffs: ParagraphDiff[]; originalHtml: string }) => void
  acceptChange: (idx: number) => void
  rejectChange: (idx: number) => void
  setFocusedChange: (idx: number) => void
  prevChange: () => void
  nextChange: () => void
  acceptAllDiff: () => void
  rejectAllDiff: () => void
  applyPendingDiff: () => void
  clearPendingDiff: () => void
}

// Direct import for OS "Open with" on .epub/.docx — same importers the
// New Book wizard uses, minus the file picker and preview step.
async function importExternalBook(path: string) {
  const ext = (path.split('.').pop() || '').toLowerCase()
  try {
    const result = ext === 'epub' ? await ImportEPUB(path) : await ImportDOCX(path)
    if (!result.success || !result.book) {
      if (result.error !== 'cancelled') setStatus(`Import failed: ${result.error || 'unknown error'}`)
      return
    }
    useBookStore.getState().loadImportedBook(result.book as unknown as BookData)
    const warnings = result.warnings?.length || 0
    if (warnings > 0) {
      setStatus(`Imported: ${result.book.metadata?.title || path} (${warnings} import warning${warnings === 1 ? '' : 's'})`)
    }
  } catch (e) {
    setStatus(`Import failed: ${e}`)
  }
}

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
      const opened: BookData = await OpenBookDialog()
      if (!opened?.version) return
      const section: Section = opened.body.length > 0 ? 'body' : 'front_matter'
      beginBookSession()
      useBookStore.setState({ book: opened, currentSection: section, currentIndex: 0, isDirty: false, analysisRevision: 0 })
      setStatus(`Opened: ${opened.metadata.title}`)
    } catch (e) {
      setStatus(`Error opening file: ${e}`)
    }
  }
}

function getSectionArray(book: BookData, section: Section): ChapterItem[] {
  switch (section) {
    case 'front_matter': return book.front_matter
    case 'body': return book.body
    case 'back_matter': return book.back_matter
    default: return []
  }
}

function setSectionArray(book: BookData, section: Section, items: ChapterItem[]): BookData {
  switch (section) {
    case 'front_matter': return { ...book, front_matter: items }
    case 'body': return { ...book, body: items }
    case 'back_matter': return { ...book, back_matter: items }
    default: return book
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
  isIndexing: false,
  analysisRevision: 0,

  // UI state
  dialogs: { showUnsavedWarning: false, pendingAction: null, showNewBookWizard: false },

  viewMode: 'editor',
  setViewMode: (mode) => set({ viewMode: mode }),

  // Editor store bridge - delegate to editorStore
  get editorRef() { return useEditorStore.getState().editorRef },
  setEditorRef: (editor) => useEditorStore.getState().setEditorRef(editor),
  getEditorSelection: () => useEditorStore.getState().getEditorSelection(),
  get inlinePrompt() { return useEditorStore.getState().inlinePrompt },
  openInlinePrompt: (cursorPos) => useEditorStore.getState().openInlinePrompt(cursorPos),
  closeInlinePrompt: () => useEditorStore.getState().closeInlinePrompt(),
  get pendingDiff() { return useEditorStore.getState().pendingDiff },
  setPendingDiff: (payload) => useEditorStore.getState().setPendingDiff(payload),
  acceptChange: (idx) => useEditorStore.getState().acceptChange(idx),
  rejectChange: (idx) => useEditorStore.getState().rejectChange(idx),
  setFocusedChange: (idx) => useEditorStore.getState().setFocusedChange(idx),
  prevChange: () => useEditorStore.getState().prevChange(),
  nextChange: () => useEditorStore.getState().nextChange(),
  acceptAllDiff: () => useEditorStore.getState().acceptAllDiff(),
  rejectAllDiff: () => useEditorStore.getState().rejectAllDiff(),
  applyPendingDiff: () => useEditorStore.getState().applyPendingDiff(get().updateCurrentContent),
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
    const { isDirty } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: 'open' } }))
      return
    }
    try {
      const book: BookData = await OpenBookDialog()
      if (!book?.version) return
      const section: Section = book.body.length > 0 ? 'body' : 'front_matter'
      beginBookSession()
      set({ book, currentSection: section, currentIndex: 0, isDirty: false, analysisRevision: 0 })
      setStatus(`Opened: ${book.metadata.title}`)
      useEditorStore.getState().clearPendingDiff()
      if (book.file_path) {
        const wordCount = countBookWords(book)
        const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
        await AddRecentProject(types.RecentProject.createFrom({
          type: 'book', path: book.file_path, name: book.metadata.title || 'Untitled',
          lastOpened: new Date().toISOString(), stats: { chapters: chapterCount, words: wordCount }
        }))
      }
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
    const { isDirty } = get()
    if (isDirty) {
      set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: true, pendingAction: { openPath: path } } }))
      return
    }
    try {
      const book: BookData = await OpenRecentProject(path)
      if (!book?.version) return
      const section: Section = book.body.length > 0 ? 'body' : 'front_matter'
      beginBookSession()
      set({ book, currentSection: section, currentIndex: 0, isDirty: false, analysisRevision: 0 })
      setStatus(`Opened: ${book.metadata.title}`)
      useEditorStore.getState().clearPendingDiff()
      const wordCount = countBookWords(book)
      const chapterCount = book.front_matter.length + book.body.length + book.back_matter.length
      await AddRecentProject(types.RecentProject.createFrom({
        type: 'book', path: book.file_path || path, name: book.metadata.title || 'Untitled',
        lastOpened: new Date().toISOString(), stats: { chapters: chapterCount, words: wordCount }
      }))
    } catch (e) {
      setStatus(`Error opening file: ${e}`)
    }
  },

  saveBook: async () => {
    if (!get().book) return
    const outcome = await performSave('save')
    if (outcome.status === 'saved') {
      setStatus(`Saved: ${outcome.filePath}`)
    } else if (outcome.status === 'stale') {
      setStatus('Newer edits were made while saving — save again')
    } else if (outcome.status === 'error') {
      setStatus(`Save failed: ${outcome.message}`)
    }
  },

  saveBookAs: async () => {
    if (!get().book) return
    const outcome = await performSave('saveAs')
    if (outcome.status === 'saved') {
      setStatus(`Saved: ${outcome.filePath}`)
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
      const session = bookSession
      const request = [{ chapter_id: chapter.id, section: currentSection, chapter_title: chapter.title, content: chapter.content, reason: 'Before history restore' }]
      const run = saveChain.then(() => {
        const latest = useBookStore.getState().book
        if (bookSession !== session || !latest) return { success: false, file_path: '', error: 'project changed' }
        return SaveBookSnapshots(latest as any, request as any)
      })
      saveChain = run.catch(() => {})
      const result = await run
      if (bookSession !== session || !result.success) {
        setStatus(`Could not create restore checkpoint: ${result.error || 'unknown error'}`)
        return false
      }
    }
    get().updateCurrentContent(content)
    setStatus('Previous chapter version restored — save to keep it')
    return true
  },

  closeProject: async () => {
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

  // Navigation
  setCurrentChapter: (section, index) => {
    set({ currentSection: section, currentIndex: index })
    useEditorStore.getState().clearPendingDiff()
  },

  // Chapter operations
  updateCurrentContent: (html) => {
    const { book, currentSection, currentIndex } = get()
    if (!book) return
    if (currentSection === 'copyright') {
      set(state => ({ book: { ...book, copyright: html }, isDirty: true, analysisRevision: state.analysisRevision + 1 }))
      scheduleAutoSave()
      return
    }
    const items = getSectionArray(book, currentSection)
    if (!items[currentIndex]) return
    const updated = items.map((item, i) => i === currentIndex ? { ...item, content: html } : item)
    set(state => ({ book: setSectionArray(book, currentSection, updated), isDirty: true, analysisRevision: state.analysisRevision + 1 }))
    scheduleChapterHistory(items[currentIndex].id)
    scheduleAutoSave()
  },

  updateChapterTitle: (section, index, title) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    const updated = items.map((item, i) => i === index ? { ...item, title } : item)
    set(state => ({ book: setSectionArray(book, section, updated), isDirty: true, analysisRevision: state.analysisRevision + 1 }))
    scheduleChapterHistory(items[index]?.id)
    scheduleAutoSave()
  },

  updateChapterSubtitle: (section, index, subtitle) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    const updated = items.map((item, i) => i === index ? { ...item, subtitle } : item)
    set(state => ({ book: setSectionArray(book, section, updated), isDirty: true, analysisRevision: state.analysisRevision + 1 }))
    scheduleChapterHistory(items[index]?.id)
    scheduleAutoSave()
  },

  addChapter: (section, item) => {
    const { book } = get()
    if (!book || section === 'copyright') return
    const items = [...getSectionArray(book, section), { ...item, id: item.id || newChapterID() }]
    const newBook = setSectionArray(book, section, items)
    set(state => ({ book: newBook, currentSection: section, currentIndex: items.length - 1, isDirty: true, analysisRevision: state.analysisRevision + 1 }))
    scheduleAutoSave()
  },

  deleteChapter: (section, index) => {
    const { book, currentSection, currentIndex } = get()
    if (!book || section === 'copyright') return
    const items = getSectionArray(book, section)
    if (items.length <= 1 && section === 'body') return
    const updated = items.filter((_, i) => i !== index)
    const newBook = setSectionArray(book, section, updated)
    let newIndex = currentIndex
    if (section === currentSection && index === currentIndex) {
      newIndex = Math.max(0, index - 1)
    } else if (section === currentSection && index < currentIndex) {
      newIndex = currentIndex - 1
    }
    set(state => ({ book: newBook, currentIndex: Math.min(newIndex, updated.length - 1), isDirty: true, analysisRevision: state.analysisRevision + 1 }))
    scheduleAutoSave()
  },

  moveChapter: (section, from, to) => {
    const { book, currentSection, currentIndex } = get()
    if (!book || section === 'copyright') return
    const items = [...getSectionArray(book, section)]
    const [moved] = items.splice(from, 1)
    items.splice(to, 0, moved)
    let newIndex = currentIndex
    if (section === currentSection) {
      if (currentIndex === from) newIndex = to
      else if (from < currentIndex && to >= currentIndex) newIndex = currentIndex - 1
      else if (from > currentIndex && to <= currentIndex) newIndex = currentIndex + 1
    }
    set(state => ({ book: setSectionArray(book, section, items), currentIndex: newIndex, isDirty: true, analysisRevision: state.analysisRevision + 1 }))
    scheduleAutoSave()
  },

  updateMetadata: (metadata) => {
    const { book } = get()
    if (!book) return
    set({ book: { ...book, metadata: { ...book.metadata, ...metadata } }, isDirty: true })
    scheduleAutoSave()
  },

  updateCopyright: (html) => {
    const { book } = get()
    if (!book) return
    set(state => ({ book: { ...book, copyright: html }, isDirty: true, analysisRevision: state.analysisRevision + 1 }))
    scheduleAutoSave()
  },

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
  indexBook: async () => {
    const { book } = get()
    if (!book) return
    set({ isIndexing: true })
    setStatus('Indexing characters...')
    try {
      const result = await IndexBook(book as any)
      if (result.success && result.book) {
        set(state => ({
          book: result.book as unknown as BookData,
          isIndexing: false,
          isDirty: true,
          analysisRevision: state.analysisRevision + 1,
        }))
        setStatus(`Found ${result.characters_found} characters (${result.new_characters} new)`)
        scheduleAutoSave()
      } else {
        set({ isIndexing: false })
        setStatus(result.error || 'Indexing failed')
      }
    } catch (e) {
      set({ isIndexing: false })
      setStatus(`Indexing error: ${e}`)
    }
  },

  // Purge every character (including fossils from older versions of the
  // detector) plus all entity/relationship data, for a clean re-index.
  clearAllCharacters: () => {
    const { book } = get()
    if (!book) return
    set({
      book: {
        ...book,
        story_bible: { ...book.story_bible, characters: [], plot_notes: book.story_bible?.plot_notes || '', timeline: book.story_bible?.timeline || '' },
        analysis: { ...book.analysis, entity_resolution: undefined, relationships: undefined, evidence: undefined },
      },
      isDirty: true,
    })
    setStatus('All characters cleared — re-index to detect them fresh')
    scheduleAutoSave()
  },

  mergeEntities: async (entityIds, canonical) => {
    const { book } = get()
    if (!book) return false
    try {
      const result = await MergeEntities(book as any, entityIds, canonical)
      if (result.success && result.book) {
        set(state => ({ book: result.book as unknown as BookData, isDirty: true, analysisRevision: state.analysisRevision + 1 }))
        setStatus('Characters merged')
        scheduleAutoSave()
        return true
      }
      setStatus(result.error || 'Merge failed')
      return false
    } catch (e) {
      setStatus(`Merge error: ${e}`)
      return false
    }
  },

  splitEntity: async (entityId, mentionIds, newCanonical) => {
    const { book } = get()
    if (!book) return false
    try {
      const result = await SplitEntity(book as any, entityId, mentionIds, newCanonical)
      if (result.success && result.book) {
        set(state => ({ book: result.book as unknown as BookData, isDirty: true, analysisRevision: state.analysisRevision + 1 }))
        setStatus('Character split')
        scheduleAutoSave()
        return true
      }
      setStatus(result.error || 'Split failed')
      return false
    } catch (e) {
      setStatus(`Split error: ${e}`)
      return false
    }
  },

  // UI actions

  closeUnsavedWarning: () => set(s => ({ dialogs: { ...s.dialogs, showUnsavedWarning: false, pendingAction: null } })),

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
