export interface SunTimes {
  dawn: Date
  dusk: Date
  source: 'api' | 'manual' | 'fallback'
}

/**
 * Get approximate location from IP address (free API, no key required)
 */
async function getLocationByIP(): Promise<{ lat: number; lng: number }> {
  const res = await fetch('http://ip-api.com/json/?fields=lat,lon')
  if (!res.ok) throw new Error('Failed to fetch location')
  const data = await res.json()
  return { lat: data.lat, lng: data.lon }
}

/**
 * Fetch sunrise/sunset times from API based on IP geolocation
 */
export async function fetchSunTimes(): Promise<SunTimes> {
  try {
    const { lat, lng } = await getLocationByIP()
    const res = await fetch(
      `https://api.sunrise-sunset.org/json?lat=${lat}&lng=${lng}&formatted=0`
    )
    if (!res.ok) throw new Error('Failed to fetch sun times')
    const data = await res.json()

    if (data.status !== 'OK') throw new Error('Invalid sun times response')

    return {
      dawn: new Date(data.results.sunrise),
      dusk: new Date(data.results.sunset),
      source: 'api'
    }
  } catch (e) {
    console.warn('Failed to fetch sun times, using defaults:', e)
    return getDefaultSunTimes()
  }
}

/**
 * Fallback sun times (6:30 AM dawn, 7:00 PM dusk)
 */
export function getDefaultSunTimes(): SunTimes {
  const today = new Date()
  const dawn = new Date(today)
  dawn.setHours(6, 30, 0, 0)
  const dusk = new Date(today)
  dusk.setHours(19, 0, 0, 0)
  return { dawn, dusk, source: 'fallback' }
}

/**
 * Parse manual time strings (HH:MM format) into Date objects for today
 */
export function parseManualTimes(dawnStr: string, duskStr: string): SunTimes {
  const today = new Date()

  const [dawnH, dawnM] = dawnStr.split(':').map(Number)
  const dawn = new Date(today)
  dawn.setHours(dawnH || 6, dawnM || 30, 0, 0)

  const [duskH, duskM] = duskStr.split(':').map(Number)
  const dusk = new Date(today)
  dusk.setHours(duskH || 19, duskM || 0, 0, 0)

  return { dawn, dusk, source: 'manual' }
}

/**
 * Check if current time is during night (before dawn or after dusk)
 */
export function isNightTime(dawn: Date, dusk: Date): boolean {
  const now = new Date()
  const nowMinutes = now.getHours() * 60 + now.getMinutes()
  const dawnMinutes = dawn.getHours() * 60 + dawn.getMinutes()
  const duskMinutes = dusk.getHours() * 60 + dusk.getMinutes()
  return nowMinutes < dawnMinutes || nowMinutes >= duskMinutes
}

/**
 * Get the next transition time (either dawn or dusk, whichever comes next)
 */
export function getNextTransitionTime(dawn: Date, dusk: Date): Date {
  const now = new Date()
  const nowMinutes = now.getHours() * 60 + now.getMinutes()
  const dawnMinutes = dawn.getHours() * 60 + dawn.getMinutes()
  const duskMinutes = dusk.getHours() * 60 + dusk.getMinutes()

  // Create today's dawn and dusk
  const todayDawn = new Date(now)
  todayDawn.setHours(dawn.getHours(), dawn.getMinutes(), 0, 0)

  const todayDusk = new Date(now)
  todayDusk.setHours(dusk.getHours(), dusk.getMinutes(), 0, 0)

  // If before dawn, next transition is dawn
  if (nowMinutes < dawnMinutes) {
    return todayDawn
  }
  // If before dusk, next transition is dusk
  if (nowMinutes < duskMinutes) {
    return todayDusk
  }
  // After dusk, next transition is tomorrow's dawn
  const tomorrowDawn = new Date(todayDawn)
  tomorrowDawn.setDate(tomorrowDawn.getDate() + 1)
  return tomorrowDawn
}

/**
 * Format time for display (e.g., "6:30 AM")
 */
export function formatTime(date: Date): string {
  return date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
}
