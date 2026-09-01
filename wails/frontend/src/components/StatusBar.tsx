import { useShallow } from 'zustand/react/shallow'
import { useBookStore } from '../store/bookStore'
import { useAppStore } from '../store/appStore'
import { useAnalysisStore } from '../store/analysisStore'

export default function StatusBar() {
  const { book, isDirty, isAutoSaving, setViewMode } = useBookStore(useShallow(s => ({
    book: s.book,
    isDirty: s.isDirty,
    isAutoSaving: s.isAutoSaving,
    setViewMode: s.setViewMode,
  })))
  const statusMessage = useAppStore(s => s.statusMessage)
  const openStorySearch = useAppStore(s => s.openStorySearch)
  const filePath = book?.file_path || null
  const fileName = filePath ? filePath.split(/[\\/]/).pop() : null
  const analysis = useAnalysisStore(useShallow(s => ({
    state: s.state,
    progress: s.progress,
    message: s.message,
    modules: s.modules,
    error: s.error,
    run: s.run,
  })))

  return (
    <div className="statusbar">
      <div className="statusbar-left">
        {isAutoSaving ? (
          <span className="statusbar-autosave" title="Auto-saving...">
            <svg className="autosave-spinner" width="12" height="12" viewBox="0 0 12 12">
              <circle cx="6" cy="6" r="5" fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray="20" strokeLinecap="round">
                <animateTransform attributeName="transform" type="rotate" from="0 6 6" to="360 6 6" dur="0.8s" repeatCount="indefinite" />
              </circle>
            </svg>
          </span>
        ) : (
          <span className={`statusbar-dot ${isDirty ? 'dirty' : ''}`} title={isDirty ? 'Unsaved changes' : 'Saved'} />
        )}
        <span className="statusbar-file">{fileName ? fileName : 'Unsaved'}</span>
        {statusMessage && <span className="statusbar-message">{statusMessage}</span>}
      </div>
      {book && analysis.state === 'running' && (
        <div className="statusbar-analysis-progress" title={analysis.message}>
          <span>{analysis.message}</span>
          <div className="statusbar-analysis-track" aria-label={`${analysis.progress}% complete`}>
            <div style={{ width: `${analysis.progress}%` }} />
          </div>
          <span className="statusbar-analysis-percent">{Math.round(analysis.progress)}%</span>
        </div>
      )}
      <div className="statusbar-right">
        {book && (
          <button className="statusbar-story-search" onClick={() => { setViewMode('editor'); openStorySearch() }} title="Ask Draftline or open the story timeline (Ctrl+Shift+F)">
            <svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.25">
              <circle cx="5" cy="5" r="3.4" /><path d="M7.5 7.5 11 11" />
            </svg>
            Ask Draftline
          </button>
        )}
        {book && (
          <button className="statusbar-story-search" onClick={() => setViewMode('cast')} title="Open the character codex">
            <svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.25">
              <circle cx="4.5" cy="3.5" r="2" />
              <path d="M1.5 10.5v-1a3 3 0 0 1 3-3h.5a3 3 0 0 1 3 3v1" />
              <path d="M8 1.8a2 2 0 0 1 0 3.4" />
              <path d="M10.5 10.5v-1a3 3 0 0 0-2-2.8" />
            </svg>
            Character Map
          </button>
        )}
        {book && analysis.state !== 'idle' && (
          <button
            className="statusbar-analysis-modules"
            onClick={() => void analysis.run()}
            disabled={analysis.state === 'running'}
            title={analysis.state === 'error' ? analysis.error : analysis.state === 'stale' ? 'Analysis is out of date; click to run now' : 'Story analysis is current'}
          >
            {(['characters', 'plot', 'prose'] as const).map(module => (
              <span className="statusbar-analysis-module" key={module}>
                <i className={`analysis-state-dot ${analysis.modules[module]}`} />
                {module === 'characters' ? 'Characters' : module === 'plot' ? 'Plot' : 'Prose'}
              </span>
            ))}
          </button>
        )}
      </div>
    </div>
  )
}
