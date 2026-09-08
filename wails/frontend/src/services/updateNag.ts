// Background update check driving the subtle title-bar indicator. Checks
// once shortly after launch and then hourly, only while the setting allows
// it; failures are silent — the indicator is a courtesy, never a nag dialog.

import { CheckForUpdates } from '../../wailsjs/go/main/App'
import { useAppStore } from '../store/appStore'

const HOUR = 60 * 60 * 1000
const LAUNCH_DELAY = 10 * 1000

let started = false

async function runCheck(): Promise<void> {
  const { settings, setUpdateAvailable } = useAppStore.getState()
  if (!settings.update_check_enabled) {
    setUpdateAvailable(null)
    return
  }
  try {
    const result = await CheckForUpdates()
    if (result.update_available && result.latest_label && !result.error) {
      setUpdateAvailable({ label: result.latest_label })
    } else {
      setUpdateAvailable(null)
    }
  } catch {
    // Offline or rate-limited — keep whatever we knew before.
  }
}

// startUpdateNag arms the launch + hourly checks. Safe to call more than
// once; only the first call does anything.
export function startUpdateNag(): void {
  if (started) return
  started = true
  setTimeout(() => void runCheck(), LAUNCH_DELAY)
  setInterval(() => void runCheck(), HOUR)
}

// checkNow runs an immediate check — used when the setting is switched on.
export function checkNow(): void {
  void runCheck()
}
