// Word-level diff utility for comparing prose paragraphs.

export interface DiffChunk {
  type: 'equal' | 'insert' | 'delete'
  text: string
}

export interface ParagraphDiff {
  chunks: DiffChunk[]
  hasChanges: boolean
  originalText: string
  revisedText: string
}

/** Decode common HTML entities for comparison. */
function decodeEntities(text: string): string {
  return text
    .replace(/&nbsp;/gi, ' ')
    .replace(/&amp;/gi, '&')
    .replace(/&lt;/gi, '<')
    .replace(/&gt;/gi, '>')
    .replace(/&quot;/gi, '"')
    .replace(/&#39;/gi, "'")
    .replace(/&rsquo;/gi, "'")
    .replace(/&lsquo;/gi, "'")
    .replace(/&rdquo;/gi, '"')
    .replace(/&ldquo;/gi, '"')
    .replace(/&mdash;/gi, '—')
    .replace(/&ndash;/gi, '–')
    .replace(/&hellip;/gi, '…')
}

/** Split text into word+whitespace tokens. */
function tokenize(text: string): string[] {
  // Decode entities first for consistent comparison
  const decoded = decodeEntities(text)
  return decoded.match(/\S+|\s+/g) ?? []
}

/** Merge consecutive same-type chunks for cleaner display. */
function mergeChunks(chunks: DiffChunk[]): DiffChunk[] {
  const out: DiffChunk[] = []
  for (const c of chunks) {
    const last = out[out.length - 1]
    if (last && last.type === c.type) {
      last.text += c.text
    } else {
      out.push({ ...c })
    }
  }
  return out
}

/**
 * Compute a word-level diff between two plain-text strings.
 * Uses LCS-based dynamic programming, capped at 1500 tokens per side
 * to prevent UI freezes on very long paragraphs.
 */
export function wordDiff(original: string, revised: string): DiffChunk[] {
  const ot = tokenize(original).slice(0, 1500)
  const rt = tokenize(revised).slice(0, 1500)
  const m = ot.length
  const n = rt.length

  // DP table using typed arrays for efficiency
  const dp: Uint16Array[] = Array.from({ length: m + 1 }, () => new Uint16Array(n + 1))
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] = ot[i - 1] === rt[j - 1]
        ? dp[i - 1][j - 1] + 1
        : Math.max(dp[i - 1][j], dp[i][j - 1])
    }
  }

  // Backtrack to build chunk list
  const chunks: DiffChunk[] = []
  let i = m, j = n
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && ot[i - 1] === rt[j - 1]) {
      chunks.unshift({ type: 'equal', text: ot[i - 1] })
      i--; j--
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      chunks.unshift({ type: 'insert', text: rt[j - 1] })
      j--
    } else {
      chunks.unshift({ type: 'delete', text: ot[i - 1] })
      i--
    }
  }

  return mergeChunks(chunks)
}

function stripTags(html: string): string {
  // Remove HTML tags and decode common entities
  return decodeEntities(html.replace(/<[^>]*>/g, ''))
}

/** Extract plain-text content of each <p> element from an HTML string. */
export function extractParagraphs(html: string): string[] {
  const paras: string[] = []
  const re = /<p[^>]*>([\s\S]*?)<\/p>/gi
  let m: RegExpExecArray | null
  while ((m = re.exec(html)) !== null) {
    paras.push(stripTags(m[1]).trim())
  }
  // Fallback for plain text without <p> tags
  if (paras.length === 0 && html.trim()) {
    paras.push(stripTags(html).trim())
  }
  return paras
}

/**
 * Diff two HTML chapter strings paragraph by paragraph.
 * Returns one ParagraphDiff per paragraph (matched by position).
 */
export function diffContent(originalHtml: string, revisedHtml: string): ParagraphDiff[] {
  const origParas = extractParagraphs(originalHtml)
  const revParas = extractParagraphs(revisedHtml)
  const count = Math.max(origParas.length, revParas.length)
  return Array.from({ length: count }, (_, idx) => {
    const orig = origParas[idx] ?? ''
    const rev = revParas[idx] ?? ''
    const chunks = wordDiff(orig, rev)
    return {
      chunks,
      hasChanges: chunks.some(c => c.type !== 'equal'),
      originalText: orig,
      revisedText: rev,
    }
  })
}

/**
 * Reassemble chapter HTML from diffs.
 * `accepted[i]` = true means use the revised version of paragraph i.
 */
export function assembleParagraphs(diffs: ParagraphDiff[], accepted: boolean[]): string {
  return diffs
    .map((d, i) => `<p>${accepted[i] ? d.revisedText : d.originalText}</p>`)
    .join('\n')
}

// ── Per-change (word-level) tracking ─────────────────────────────────────────

/** A single contiguous run of delete/insert chunks within one paragraph. */
export interface DiffChange {
  paraIdx: number
  startIdx: number  // inclusive chunk index
  endIdx: number    // exclusive chunk index
  accepted: boolean // true = take AI version; false = keep original
  decided: boolean  // true = user explicitly made a choice
}

/** Extract all individual change groups from a set of paragraph diffs. */
export function extractChanges(diffs: ParagraphDiff[]): DiffChange[] {
  const changes: DiffChange[] = []
  diffs.forEach((diff, paraIdx) => {
    if (!diff.hasChanges) return
    let i = 0
    while (i < diff.chunks.length) {
      if (diff.chunks[i].type !== 'equal') {
        const startIdx = i
        while (i < diff.chunks.length && diff.chunks[i].type !== 'equal') i++
        // Default to NOT accepted - user must explicitly accept each change
        // This prevents accidental data loss if AI returns empty/broken content
        changes.push({ paraIdx, startIdx, endIdx: i, accepted: false, decided: false })
      } else {
        i++
      }
    }
  })
  return changes
}

/**
 * Assemble final chapter HTML using per-change acceptance.
 * accepted change → take inserts, drop deletes
 * rejected change → keep deletes (original text), drop inserts
 */
export function assembleFromChanges(diffs: ParagraphDiff[], changes: DiffChange[]): string {
  const changeMap = new Map<string, boolean>()
  changes.forEach(ch => {
    for (let ci = ch.startIdx; ci < ch.endIdx; ci++) {
      changeMap.set(`${ch.paraIdx}-${ci}`, ch.accepted)
    }
  })
  return diffs.map((diff, paraIdx) => {
    const text = diff.chunks.map((chunk, ci) => {
      if (chunk.type === 'equal') return chunk.text
      const accepted = changeMap.get(`${paraIdx}-${ci}`) ?? true
      if (chunk.type === 'insert') return accepted ? chunk.text : ''
      if (chunk.type === 'delete') return accepted ? '' : chunk.text
      return ''
    }).join('')
    return `<p>${text}</p>`
  }).join('\n')
}
