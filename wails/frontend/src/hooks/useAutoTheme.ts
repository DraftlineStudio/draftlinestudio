import { useState, useEffect, useRef, useCallback } from 'react'
import {
  fetchSunTimes,
  parseManualTimes,
  isNightTime,
  getNextTransitionTime,
  type SunTimes
} from '../utils/sunTimes'

export type ThemeMode = 'light' | 'dark' | 'auto'

interface UseAutoThemeOptions {
  themeMode: ThemeMode
  manualDawn?: string
  manualDusk?: string
  useManualTimes?: boolean
  onTransition?: (theme: 'light' | 'dark') => void
}

interface UseAutoThemeResult {
  computedTheme: 'light' | 'dark'
  sunTimes: SunTimes | null
  isLoading: boolean
  refetchSunTimes: () => void
}

export function useAutoTheme({
  themeMode,
  manualDawn,
  manualDusk,
  useManualTimes = false,
  onTransition
}: UseAutoThemeOptions): UseAutoThemeResult {
  const [sunTimes, setSunTimes] = useState<SunTimes | null>(null)
  const [computedTheme, setComputedTheme] = useState<'light' | 'dark'>('dark')
  const [isLoading, setIsLoading] = useState(false)
  const prevThemeRef = useRef<'light' | 'dark' | null>(null)
  const timeoutRef = useRef<number | null>(null)

  const fetchTimes = useCallback(async () => {
    if (themeMode !== 'auto') return

    setIsLoading(true)
    try {
      if (useManualTimes && manualDawn && manualDusk) {
        setSunTimes(parseManualTimes(manualDawn, manualDusk))
      } else {
        const times = await fetchSunTimes()
        setSunTimes(times)
      }
    } catch (e) {
      console.error('Failed to get sun times:', e)
    }
    setIsLoading(false)
  }, [themeMode, useManualTimes, manualDawn, manualDusk])

  // Fetch sun times when auto mode is enabled or settings change
  useEffect(() => {
    fetchTimes()
  }, [fetchTimes])

  // Check time and schedule transitions
  useEffect(() => {
    if (themeMode !== 'auto' || !sunTimes) return

    const checkAndSchedule = () => {
      const shouldBeDark = isNightTime(sunTimes.dawn, sunTimes.dusk)
      const newTheme = shouldBeDark ? 'dark' : 'light'

      // Only trigger transition callback when theme actually changes
      if (prevThemeRef.current !== null && prevThemeRef.current !== newTheme) {
        onTransition?.(newTheme)
      }

      setComputedTheme(newTheme)
      prevThemeRef.current = newTheme

      // Schedule next check at transition time
      const nextTransition = getNextTransitionTime(sunTimes.dawn, sunTimes.dusk)
      const msUntilTransition = nextTransition.getTime() - Date.now()

      // Check every minute as a fallback, or at the exact transition time
      const checkInterval = Math.min(msUntilTransition + 1000, 60000)
      timeoutRef.current = window.setTimeout(checkAndSchedule, checkInterval)
    }

    checkAndSchedule()

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [themeMode, sunTimes, onTransition])

  // Reset prevThemeRef when switching away from auto mode
  useEffect(() => {
    if (themeMode !== 'auto') {
      prevThemeRef.current = null
    }
  }, [themeMode])

  // For non-auto modes, just return the explicit theme
  const finalTheme = themeMode === 'auto' ? computedTheme : themeMode

  return {
    computedTheme: finalTheme,
    sunTimes: themeMode === 'auto' ? sunTimes : null,
    isLoading,
    refetchSunTimes: fetchTimes
  }
}
