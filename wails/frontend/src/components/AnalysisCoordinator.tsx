import { useEffect, useRef } from 'react'
import { useShallow } from 'zustand/react/shallow'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useBookStore } from '../store/bookStore'
import { useAppStore } from '../store/appStore'
import { useAnalysisStore, type AnalysisProgressEvent } from '../store/analysisStore'

const IDLE_ANALYSIS_DELAY = 15_000
const OPEN_ANALYSIS_DELAY = 750

// Non-visual coordinator for activity-aware local analysis. Editing only
// resets a timer; no manuscript-scale work occurs on the typing path.
export default function AnalysisCoordinator() {
  const { hasBook, identity, revision } = useBookStore(useShallow(state => ({
    hasBook: state.book !== null,
    identity: state.book ? `${state.book.file_path || ''}\u0000${state.book.metadata.created || ''}` : '',
    revision: state.analysisRevision,
  })))
  const enabled = useAppStore(state => state.settings.analysis_enabled)
  const { markStale, clear, run, receiveProgress } = useAnalysisStore(useShallow(state => ({
    markStale: state.markStale,
    clear: state.clear,
    run: state.run,
    receiveProgress: state.receiveProgress,
  })))
  const previousIdentity = useRef('')

  useEffect(() => EventsOn('analysis:progress', (event: AnalysisProgressEvent) => receiveProgress(event)), [receiveProgress])

  useEffect(() => {
    if (!hasBook || !enabled) {
      clear()
      previousIdentity.current = identity
      return
    }
    const newlyOpened = identity !== previousIdentity.current
    previousIdentity.current = identity
    markStale()
    const timer = window.setTimeout(() => void run(), newlyOpened ? OPEN_ANALYSIS_DELAY : IDLE_ANALYSIS_DELAY)
    return () => window.clearTimeout(timer)
  }, [hasBook, identity, revision, enabled, markStale, clear, run])

  return null
}
