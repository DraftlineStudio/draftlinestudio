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
  /**
   * Verbatim original `<p>…</p>` HTML for this paragraph, including any inline
   * formatting (`<em>`, `<strong>`, `<a>`, attributes). Empty for paragraphs
   * that exist only on the revised side (pure insertions).
   */
  originalHtml: string
  /**
   * Verbatim revised `<p>…</p>` HTML for this paragraph. Empty for paragraphs
   * that exist only on the original side (pure deletions).
   */
  revisedHtml: string
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

/** One extracted paragraph: its plain text (for diffing) and verbatim HTML. */
interface ParaBlock {
  /** Tag-stripped, entity-decoded, trimmed text used for word-level diffing. */
  text: string
  /** Verbatim `<p …>…</p>` block exactly as it appeared in the source HTML. */
  html: string
}

/** Extract each <p> element from an HTML string, keeping both text and verbatim HTML. */
function extractBlocks(html: string): ParaBlock[] {
  const blocks: ParaBlock[] = []
  const re = /<p[^>]*>([\s\S]*?)<\/p>/gi
  let m: RegExpExecArray | null
  while ((m = re.exec(html)) !== null) {
    blocks.push({ text: stripTags(m[1]).trim(), html: m[0] })
  }
  // Fallback for plain text without <p> tags
  if (blocks.length === 0 && html.trim()) {
    blocks.push({ text: stripTags(html).trim(), html: html.trim() })
  }
  return blocks
}

/** Extract plain-text content of each <p> element from an HTML string. */
export function extractParagraphs(html: string): string[] {
  return extractBlocks(html).map(b => b.text)
}

/** One step in a line-level edit script between two paragraph arrays. */
interface SeqOp {
  type: 'equal' | 'delete' | 'insert'
  ai?: number // index into the original array (equal, delete)
  bi?: number // index into the revised array (equal, insert)
}

/**
 * LCS-based edit script over two arrays of paragraph texts.
 * Yields ops in document order (ascending original / revised indices).
 */
function diffSeq(a: string[], b: string[]): SeqOp[] {
  const m = a.length
  const n = b.length
  const dp: Uint32Array[] = Array.from({ length: m + 1 }, () => new Uint32Array(n + 1))
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] = a[i - 1] === b[j - 1]
        ? dp[i - 1][j - 1] + 1
        : Math.max(dp[i - 1][j], dp[i][j - 1])
    }
  }
  const ops: SeqOp[] = []
  let i = m, j = n
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && a[i - 1] === b[j - 1]) {
      ops.unshift({ type: 'equal', ai: i - 1, bi: j - 1 }); i--; j--
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      ops.unshift({ type: 'insert', bi: j - 1 }); j--
    } else {
      ops.unshift({ type: 'delete', ai: i - 1 }); i--
    }
  }
  return ops
}

/** Build a single paragraph diff row from an optional original/revised block. */
function makeRow(o: ParaBlock | null, r: ParaBlock | null): ParagraphDiff {
  const oText = o?.text ?? ''
  const rText = r?.text ?? ''
  const chunks = wordDiff(oText, rText)
  return {
    chunks,
    hasChanges: chunks.some(c => c.type !== 'equal'),
    originalText: oText,
    revisedText: rText,
    originalHtml: o?.html ?? '',
    revisedHtml: r?.html ?? '',
  }
}

/**
 * Diff two HTML chapter strings paragraph by paragraph.
 *
 * Paragraphs are aligned by an LCS edit script over their plain text (not by
 * positional index), so an inserted or deleted paragraph does not cascade into
 * spurious full-paragraph diffs on every following paragraph. Each row keeps the
 * verbatim original and revised `<p>` HTML so formatting can be preserved when the
 * diff is reassembled.
 */
export function diffContent(originalHtml: string, revisedHtml: string): ParagraphDiff[] {
  const orig = extractBlocks(originalHtml)
  const rev = extractBlocks(revisedHtml)
  const ops = diffSeq(orig.map(b => b.text), rev.map(b => b.text))

  const rows: ParagraphDiff[] = []
  let k = 0
  while (k < ops.length) {
    if (ops[k].type === 'equal') {
      rows.push(makeRow(orig[ops[k].ai!], rev[ops[k].bi!]))
      k++
      continue
    }
    // Gather a maximal run of non-equal ops between two equal anchors and pair
    // deletes with inserts positionally so they render as word-level edits
    // rather than whole-paragraph delete+insert churn.
    const dels: number[] = []
    const inss: number[] = []
    while (k < ops.length && ops[k].type !== 'equal') {
      if (ops[k].type === 'delete') dels.push(ops[k].ai!)
      else inss.push(ops[k].bi!)
      k++
    }
    const pairCount = Math.min(dels.length, inss.length)
    for (let p = 0; p < pairCount; p++) rows.push(makeRow(orig[dels[p]], rev[inss[p]]))
    for (let p = pairCount; p < dels.length; p++) rows.push(makeRow(orig[dels[p]], null))
    for (let p = pairCount; p < inss.length; p++) rows.push(makeRow(null, rev[inss[p]]))
  }
  return rows
}

/**
 * Reassemble chapter HTML from diffs.
 * `accepted[i]` = true means use the revised version of paragraph i.
 * Verbatim per-paragraph HTML is preserved so formatting is never flattened.
 */
export function assembleParagraphs(diffs: ParagraphDiff[], accepted: boolean[]): string {
  return diffs
    .map((d, i) => (accepted[i] ? d.revisedHtml : d.originalHtml))
    .filter(html => html.length > 0)
    .join('')
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
 *
 * Formatting-preserving rules, per paragraph:
 *  - No changes            → emit the verbatim ORIGINAL `<p>` HTML.
 *  - Every change rejected → emit the verbatim ORIGINAL `<p>` HTML
 *                            (empty for a pure insertion → nothing).
 *  - Every change accepted → emit the verbatim REVISED `<p>` HTML
 *                            (empty for a pure deletion → nothing).
 *  - Mixed (some accepted) → reconstruct from word chunks. Only in this case is
 *                            inline formatting within the paragraph flattened,
 *                            and only for that one paragraph.
 *
 * This guarantees that accepting a change in one paragraph can never strip
 * formatting from any paragraph the user did not change, and that rejecting
 * everything reproduces the original HTML.
 */
export function assembleFromChanges(diffs: ParagraphDiff[], changes: DiffChange[]): string {
  const changeMap = new Map<string, boolean>()
  const paraCounts = new Map<number, { acc: number; total: number }>()
  changes.forEach(ch => {
    for (let ci = ch.startIdx; ci < ch.endIdx; ci++) {
      changeMap.set(`${ch.paraIdx}-${ci}`, ch.accepted)
    }
    const pc = paraCounts.get(ch.paraIdx) ?? { acc: 0, total: 0 }
    pc.total++
    if (ch.accepted) pc.acc++
    paraCounts.set(ch.paraIdx, pc)
  })

  const out: string[] = []
  diffs.forEach((diff, paraIdx) => {
    if (!diff.hasChanges) {
      if (diff.originalHtml) out.push(diff.originalHtml)
      return
    }
    const pc = paraCounts.get(paraIdx) ?? { acc: 0, total: 0 }
    if (pc.acc === 0) {
      // All changes rejected → keep the original paragraph verbatim.
      if (diff.originalHtml) out.push(diff.originalHtml)
      return
    }
    if (pc.acc === pc.total) {
      // All changes accepted → take the revised paragraph verbatim.
      if (diff.revisedHtml) out.push(diff.revisedHtml)
      return
    }
    // Mixed acceptance within a single paragraph: reconstruct from word chunks.
    const text = diff.chunks.map((chunk, ci) => {
      if (chunk.type === 'equal') return chunk.text
      const accepted = changeMap.get(`${paraIdx}-${ci}`) ?? true
      if (chunk.type === 'insert') return accepted ? chunk.text : ''
      if (chunk.type === 'delete') return accepted ? '' : chunk.text
      return ''
    }).join('')
    out.push(`<p>${text}</p>`)
  })
  return out.join('')
}
