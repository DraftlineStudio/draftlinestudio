// Reading a failure from whatever a rejected promise carried.
//
// Wails rejects a bound call with an Error whose message is the Go error or
// the recovered panic. Nothing guarantees that, though: a rejection can carry
// a bare string, a plain object, or nothing worth reading at all. This picks
// the useful half and always returns something a writer can act on, because
// "undefined" on screen is the same dead end as no message at all.

/** The useful half of whatever a promise was rejected with. */
export function failureMessage(reason: unknown): string {
  if (reason instanceof Error && reason.message.trim()) {
    return reason.message.trim()
  }
  if (typeof reason === 'string' && reason.trim()) {
    return reason.trim()
  }
  if (reason && typeof reason === 'object') {
    const shaped = reason as { message?: unknown; error?: unknown }
    for (const value of [shaped.message, shaped.error]) {
      if (typeof value === 'string' && value.trim()) {
        return value.trim()
      }
    }
  }
  return 'The operation failed and gave no reason.'
}
