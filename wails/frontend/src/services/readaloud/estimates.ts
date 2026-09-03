// Duration estimation for the Read Aloud progress display. Synthesized units
// have exact durations (samples / sampleRate, sentence pauses already baked
// in); unsynthesized text is estimated from a speed-normalized
// seconds-per-character rate that calibrates itself as chunks arrive.

// Prior rate at 1.0× speed, blended away as real measurements accumulate.
const SEED_SECONDS_PER_CHAR = 0.058
// Weight of the prior, in pseudo-characters: after ~a few sentences the
// measured rate dominates and the display stops drifting.
const SEED_WEIGHT_CHARS = 150

export class DurationEstimator {
  private normalizedSeconds = 0 // Σ seconds·speed (i.e. seconds at 1.0×)
  private chars = 0

  record(charCount: number, seconds: number, speed: number): void {
    if (charCount <= 0 || seconds <= 0 || speed <= 0) return
    this.normalizedSeconds += seconds * speed
    this.chars += charCount
  }

  // Estimated audible seconds for charCount characters at the given speed.
  secondsFor(charCount: number, speed: number): number {
    if (charCount <= 0 || speed <= 0) return 0
    const rate = (this.normalizedSeconds + SEED_SECONDS_PER_CHAR * SEED_WEIGHT_CHARS)
      / (this.chars + SEED_WEIGHT_CHARS)
    return charCount * rate / speed
  }

  reset(): void {
    this.normalizedSeconds = 0
    this.chars = 0
  }
}

// "M:SS" clock text (no hours — chapters don't run that long at 24 kHz).
export function formatClock(seconds: number): string {
  const s = Math.max(0, Math.round(seconds))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}
