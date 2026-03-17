import { useBookStore } from '../store/bookStore'

function countWords(html: string): number {
  const div = document.createElement('div')
  div.innerHTML = html
  const text = div.textContent || div.innerText || ''
  return text.trim().split(/\s+/).filter((w) => w.length > 0).length
}

function totalWords(book: ReturnType<typeof useBookStore.getState>['book']): number {
  if (!book) return 0
  const allContent = [
    book.copyright,
    ...book.front_matter.map((c) => c.content),
    ...book.body.map((c) => c.content),
    ...book.back_matter.map((c) => c.content),
  ]
  return allContent.reduce((sum, html) => sum + countWords(html || ''), 0)
}

export default function StatusBar() {
  const { book, isDirty, statusMessage } = useBookStore()
  const words = totalWords(book)
  const filePath = book?.file_path || null
  const fileName = filePath ? filePath.split(/[\\/]/).pop() : null

  return (
    <div className="statusbar">
      <div className="statusbar-left">
        <span className={`statusbar-dot ${isDirty ? 'dirty' : ''}`} title={isDirty ? 'Unsaved changes' : 'Saved'} />
        <span className="statusbar-file">{fileName ? fileName : 'Unsaved'}</span>
        {statusMessage && <span>{statusMessage}</span>}
      </div>
      <div className="statusbar-right">
        <span className="statusbar-words">{words.toLocaleString()} words</span>
      </div>
    </div>
  )
}
