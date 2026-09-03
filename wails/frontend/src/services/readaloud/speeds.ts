// Shared Read Aloud speed constants. Speed is a synthesis-engine parameter
// (Kokoro re-times the speech; no playbackRate pitch shift), validated
// 0.5–2.0 by the Go side — the UI exposes 0.8–2.0.

export const READ_ALOUD_MIN_SPEED = 0.8
export const READ_ALOUD_MAX_SPEED = 2.0

// The player bar's cycle stops. Includes 1.1 — the app default — so cycling
// always visits the value fresh installs start at.
export const READ_ALOUD_SPEED_CYCLE = [0.8, 1.0, 1.1, 1.2, 1.5, 2.0]

export function clampReadAloudSpeed(speed: number): number {
  const n = Number(speed)
  if (!Number.isFinite(n)) return 1.1
  return Math.min(READ_ALOUD_MAX_SPEED, Math.max(READ_ALOUD_MIN_SPEED, n))
}

// The next cycle stop after the current speed (nearest stop, then +1).
export function nextReadAloudSpeed(current: number): number {
  let nearest = 0
  for (let i = 1; i < READ_ALOUD_SPEED_CYCLE.length; i++) {
    if (Math.abs(READ_ALOUD_SPEED_CYCLE[i] - current) < Math.abs(READ_ALOUD_SPEED_CYCLE[nearest] - current)) nearest = i
  }
  return READ_ALOUD_SPEED_CYCLE[(nearest + 1) % READ_ALOUD_SPEED_CYCLE.length]
}

// "1.1×"-style label with no trailing noise for whole numbers ("2×").
export function speedLabel(speed: number): string {
  return `${Number.isInteger(speed) ? speed : speed.toFixed(1)}×`
}
