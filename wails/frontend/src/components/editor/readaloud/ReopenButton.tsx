// Floating reopen affordance: appears bottom-right of the editor column when
// the Read Aloud plugin is enabled but the player bar is closed.

import { useAppStore } from '../../../store/appStore'
import { useReadAloudStore } from '../../../store/readAloudStore'

export default function ReopenButton() {
  const enabled = useAppStore(s => s.settings.read_aloud_enabled)
  const playerVisible = useReadAloudStore(s => s.playerVisible)
  const setPlayerVisible = useReadAloudStore(s => s.setPlayerVisible)

  if (!enabled || playerVisible) return null

  return (
    <button
      className="rap-reopen"
      onClick={() => setPlayerVisible(true)}
      title="Read Aloud"
      aria-label="Open Read Aloud player"
    >
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M11 5 6 9H3v6h3l5 4z" />
        <path d="M15.5 9a4.5 4.5 0 0 1 0 6" />
      </svg>
    </button>
  )
}
