interface Candidate {
  word: string
  distance: number
  prefix: number
  lengthDelta: number
}

export type WordBuckets = Map<number, string[]>

export function buildWordBuckets(entries: string[]): WordBuckets {
  const buckets: WordBuckets = new Map()
  const seen = new Set<string>()
  entries.forEach(entry => {
    if (!/^[A-Za-z][A-Za-z'-]*$/.test(entry)) return
    const word = entry.toLocaleLowerCase()
    if (word.length < 2 || seen.has(word)) return
    seen.add(word)
    const bucket = buckets.get(word.length)
    if (bucket) bucket.push(word)
    else buckets.set(word.length, [word])
  })
  return buckets
}

function boundedDamerauLevenshtein(source: string, target: string, maxDistance: number): number {
  if (Math.abs(source.length - target.length) > maxDistance) return maxDistance + 1
  if (source === target) return 0

  let previousPrevious: number[] | null = null
  let previous = Array.from({ length: target.length + 1 }, (_, index) => index)

  for (let sourceIndex = 1; sourceIndex <= source.length; sourceIndex++) {
    const current = new Array<number>(target.length + 1)
    current[0] = sourceIndex
    let rowMinimum = current[0]

    for (let targetIndex = 1; targetIndex <= target.length; targetIndex++) {
      const substitutionCost = source[sourceIndex - 1] === target[targetIndex - 1] ? 0 : 1
      let distance = Math.min(
        previous[targetIndex] + 1,
        current[targetIndex - 1] + 1,
        previous[targetIndex - 1] + substitutionCost,
      )

      if (
        previousPrevious
        && sourceIndex > 1
        && targetIndex > 1
        && source[sourceIndex - 1] === target[targetIndex - 2]
        && source[sourceIndex - 2] === target[targetIndex - 1]
      ) {
        distance = Math.min(distance, previousPrevious[targetIndex - 2] + 1)
      }

      current[targetIndex] = distance
      rowMinimum = Math.min(rowMinimum, distance)
    }

    if (rowMinimum > maxDistance) return maxDistance + 1
    previousPrevious = previous
    previous = current
  }

  return previous[target.length]
}

function commonPrefixLength(left: string, right: string): number {
  const length = Math.min(left.length, right.length)
  let index = 0
  while (index < length && left[index] === right[index]) index++
  return index
}

export function findIndexedSuggestions(
  buckets: WordBuckets,
  word: string,
  limit: number,
): string[] {
  const normalized = word.toLocaleLowerCase()
  const maxDistance = normalized.length <= 4 ? 1 : 2
  const candidates: Candidate[] = []

  for (
    let length = Math.max(2, normalized.length - maxDistance);
    length <= normalized.length + maxDistance;
    length++
  ) {
    for (const candidate of buckets.get(length) ?? []) {
      if (candidate === normalized) continue
      const distance = boundedDamerauLevenshtein(normalized, candidate, maxDistance)
      if (distance > maxDistance) continue
      candidates.push({
        word: candidate,
        distance,
        prefix: commonPrefixLength(normalized, candidate),
        lengthDelta: Math.abs(normalized.length - candidate.length),
      })
    }
  }

  candidates.sort((left, right) =>
    left.distance - right.distance
    || right.prefix - left.prefix
    || left.lengthDelta - right.lengthDelta
    || left.word.localeCompare(right.word),
  )
  return candidates.slice(0, limit).map(candidate => candidate.word)
}
