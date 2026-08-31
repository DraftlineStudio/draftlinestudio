import { useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import { useAppStore } from '../../store/appStore'
import { ImportEPUBDialog, ImportDOCXDialog } from '../../../wailsjs/go/main/App'
import { countWords } from '../../utils/textUtils'

type WizardStep = 'choose' | 'create' | 'import-preview'

interface ImportedBook {
  title: string
  author: string
  publisher: string
  chapterCount: number
  chapters: Array<{ title: string; wordCount: number }>
}

interface NewBookWizardProps {
  onCreated?: () => void
}

export default function NewBookWizard({ onCreated }: NewBookWizardProps) {
  const { confirmNewBook, cancelNewBookWizard, loadImportedBook } = useBookStore()
  const { settings } = useAppStore()

  const [step, setStep] = useState<WizardStep>('choose')
  const [title, setTitle] = useState('')
  const [author, setAuthor] = useState(settings.default_author)
  const [publisher, setPublisher] = useState(settings.default_publisher)

  // Import state
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState('')
  const [importedBook, setImportedBook] = useState<ImportedBook | null>(null)
  const [importedBookData, setImportedBookData] = useState<any>(null)

  async function handleCreate() {
    await confirmNewBook(title.trim() || 'Untitled', author.trim(), publisher.trim())
    onCreated?.()
  }

  async function handleImport(type: 'epub' | 'docx') {
    setImporting(true)
    setImportError('')
    try {
      const result = type === 'epub' ? await ImportEPUBDialog() : await ImportDOCXDialog()
      if (!result.success || !result.book) {
        if (result.error !== 'cancelled') {
          setImportError(result.error || 'Import failed')
        }
        setImporting(false)
        return
      }

      const book = result.book

      // Calculate word counts for preview
      const chapters = book.body.map((ch: any) => ({
        title: ch.title,
        wordCount: countWords(ch.content),
      }))

      setImportedBook({
        title: book.metadata.title,
        author: book.metadata.author || '',
        publisher: book.metadata.publisher || '',
        chapterCount: chapters.length,
        chapters,
      })
      setImportedBookData(book)
      setTitle(book.metadata.title)
      setAuthor(book.metadata.author || settings.default_author)
      setPublisher(book.metadata.publisher || settings.default_publisher)
      setStep('import-preview')
    } catch (e) {
      setImportError(String(e))
    }
    setImporting(false)
  }

  function handleConfirmImport() {
    if (!importedBookData) return

    // Update metadata with any user edits
    const book = {
      ...importedBookData,
      metadata: {
        ...importedBookData.metadata,
        title: title.trim() || 'Untitled',
        author: author.trim(),
        publisher: publisher.trim(),
      },
    }

    loadImportedBook(book)
    onCreated?.()
  }

  function handleKey(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && step === 'create') void handleCreate()
    if (e.key === 'Escape') cancelNewBookWizard()
  }

  // Step 1: Choose path
  if (step === 'choose') {
    return (
      <div className="dialog-overlay">
        <div className="dialog" style={{ maxWidth: 560 }}>
          <div className="dialog-title">New Book</div>
          <p className="dialog-subtitle">How would you like to start?</p>

          <div className="wizard-choice-grid wizard-choice-grid-3">
            <button className="wizard-choice-tile" onClick={() => setStep('create')}>
              <div className="wizard-choice-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M12 5v14M5 12h14" />
                </svg>
              </div>
              <div className="wizard-choice-label">Create from Scratch</div>
              <div className="wizard-choice-desc">Start with a blank book</div>
            </button>

            <button className="wizard-choice-tile" onClick={() => handleImport('epub')} disabled={importing}>
              <div className="wizard-choice-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M4 19.5A2.5 2.5 0 016.5 17H20" />
                  <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
                </svg>
              </div>
              <div className="wizard-choice-label">{importing ? 'Importing...' : 'Import EPUB'}</div>
              <div className="wizard-choice-desc">E-book format</div>
            </button>

            <button className="wizard-choice-tile" onClick={() => handleImport('docx')} disabled={importing}>
              <div className="wizard-choice-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                  <line x1="16" y1="13" x2="8" y2="13" />
                  <line x1="16" y1="17" x2="8" y2="17" />
                </svg>
              </div>
              <div className="wizard-choice-label">{importing ? 'Importing...' : 'Import DOCX'}</div>
              <div className="wizard-choice-desc">Word document</div>
            </button>
          </div>

          {importError && (
            <div className="wizard-error">{importError}</div>
          )}

          <div className="dialog-actions">
            <button className="dialog-btn" onClick={cancelNewBookWizard}>Cancel</button>
          </div>
        </div>
      </div>
    )
  }

  // Step 2a: Create from scratch
  if (step === 'create') {
    return (
      <div className="dialog-overlay">
        <div className="dialog">
          <div className="dialog-title">New Book</div>

          <div className="dialog-field">
            <label className="dialog-label">Title</label>
            <input
              className="dialog-input"
              value={title}
              onChange={e => setTitle(e.target.value)}
              placeholder="Untitled"
              autoFocus
              onKeyDown={handleKey}
            />
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Author</label>
            <input
              className="dialog-input"
              value={author}
              onChange={e => setAuthor(e.target.value)}
              placeholder="Author name"
              onKeyDown={handleKey}
            />
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Publisher</label>
            <input
              className="dialog-input"
              value={publisher}
              onChange={e => setPublisher(e.target.value)}
              placeholder="Publisher (optional)"
              onKeyDown={handleKey}
            />
          </div>

          <div className="dialog-actions">
            <button className="dialog-btn" onClick={() => setStep('choose')}>Back</button>
            <button className="dialog-btn primary" onClick={handleCreate}>Create</button>
          </div>
        </div>
      </div>
    )
  }

  // Step 2b: Import preview
  if (step === 'import-preview' && importedBook) {
    const totalWords = importedBook.chapters.reduce((sum, ch) => sum + ch.wordCount, 0)

    return (
      <div className="dialog-overlay">
        <div className="dialog" style={{ maxWidth: 520 }}>
          <div className="dialog-title">Import Preview</div>

          <div className="import-preview-stats">
            <div className="import-stat">
              <span className="import-stat-value">{importedBook.chapterCount}</span>
              <span className="import-stat-label">Chapters</span>
            </div>
            <div className="import-stat">
              <span className="import-stat-value">{totalWords.toLocaleString()}</span>
              <span className="import-stat-label">Words</span>
            </div>
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Title</label>
            <input
              className="dialog-input"
              value={title}
              onChange={e => setTitle(e.target.value)}
              placeholder="Untitled"
            />
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Author</label>
            <input
              className="dialog-input"
              value={author}
              onChange={e => setAuthor(e.target.value)}
              placeholder="Author name"
            />
          </div>

          <div className="dialog-field">
            <label className="dialog-label">Publisher</label>
            <input
              className="dialog-input"
              value={publisher}
              onChange={e => setPublisher(e.target.value)}
              placeholder="Publisher (optional)"
            />
          </div>

          <div className="import-preview-chapters">
            <label className="dialog-label">Chapters</label>
            <div className="import-chapter-list">
              {importedBook.chapters.slice(0, 10).map((ch, i) => (
                <div key={i} className="import-chapter-item">
                  <span className="import-chapter-title">{ch.title}</span>
                  <span className="import-chapter-words">{ch.wordCount.toLocaleString()} words</span>
                </div>
              ))}
              {importedBook.chapters.length > 10 && (
                <div className="import-chapter-more">
                  +{importedBook.chapters.length - 10} more chapters
                </div>
              )}
            </div>
          </div>

          <div className="dialog-actions">
            <button className="dialog-btn" onClick={() => setStep('choose')}>Back</button>
            <button className="dialog-btn primary" onClick={handleConfirmImport}>Import</button>
          </div>
        </div>
      </div>
    )
  }

  return null
}
