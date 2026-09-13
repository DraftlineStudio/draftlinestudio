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
   * Lower-case tag of the top-level block this row represents: `p` for prose,
   * or a structural block such as `hr` (scene break), `blockquote`, `pre`,
   * `h1`–`h6`, `ul`, `ol`. Structural rows keep their wrapper markup through
   * the review so an AI pass can never flatten them into paragraphs.
   */
  tag: string
  /**
   * Verbatim original block HTML for this row, including any inline
   * formatting (`<em>`, `<strong>`, `<a>`, attributes). Empty for blocks
   * that exist only on the revised side (pure insertions).
   */
  originalHtml: string
  /**
   * Verbatim revised block HTML for this row. Empty for blocks that exist
   * only on the original side (pure deletions).
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

/** One extracted top-level block: its tag, plain text (for diffing), and verbatim HTML. */
interface ParaBlock {
  /** Lower-case tag of the top-level element (`p`, `hr`, `blockquote`, `pre`, `h2`, `ul`, …). */
  tag: string
  /** Tag-stripped, entity-decoded, trimmed text used for word-level diffing. */
  text: string
  /** Verbatim block HTML exactly as it appeared in the source. */
  html: string
}

/** Elements with no closing tag. */
const VOID_TAGS = new Set(['hr', 'br', 'img', 'input', 'wbr', 'source', 'col', 'embed', 'area', 'base', 'link', 'meta', 'param', 'track'])
/** Inline markup that, when it appears at the top level, is prose rather than a block boundary. */
const INLINE_TAGS = new Set(['em', 'strong', 'b', 'i', 'u', 's', 'strike', 'span', 'a', 'code', 'sub', 'sup', 'mark', 'small', 'del', 'ins', 'br'])

/**
 * Split chapter HTML into its top-level blocks. Nested markup (a `<p>` inside
 * a `<blockquote>`, `<li>` inside `<ul>`, `<code>` inside `<pre>`) stays inside
 * its parent block, so structure survives a diff round trip. Bare top-level
 * text and inline markup become a paragraph block.
 */
export function splitTopLevelBlocks(html: string): { tag: string; html: string }[] {
  const blocks: { tag: string; html: string }[] = []
  const tokenRe = /<!--[\s\S]*?-->|<(\/?)([a-zA-Z][a-zA-Z0-9-]*)(?:\s[^>]*)?(\/?)>/g
  let depth = 0
  let blockStart = -1
  let blockTag = ''
  let cursor = 0 // end of the last consumed block; bare text lives between cursor and the next block
  const pushBareText = (to: number) => {
    const raw = html.slice(cursor, to).trim()
    if (raw) blocks.push({ tag: 'p', html: raw })
  }
  let m: RegExpExecArray | null
  while ((m = tokenRe.exec(html)) !== null) {
    if (m[0].startsWith('<!--')) continue
    const isClose = m[1] === '/'
    const name = m[2].toLowerCase()
    const selfClosing = m[3] === '/' || VOID_TAGS.has(name)
    const end = m.index + m[0].length
    if (depth === 0) {
      if (isClose || INLINE_TAGS.has(name)) continue // stray close tag or inline prose at the top level
      pushBareText(m.index)
      if (selfClosing) {
        blocks.push({ tag: name, html: m[0] })
        cursor = end
        continue
      }
      blockStart = m.index
      blockTag = name
      depth = 1
    } else if (isClose) {
      depth--
      if (depth === 0) {
        blocks.push({ tag: blockTag, html: html.slice(blockStart, end) })
        cursor = end
      }
    } else if (!selfClosing) {
      depth++
    }
  }
  if (depth > 0) {
    // Unterminated block (malformed model output): keep everything from its start.
    blocks.push({ tag: blockTag, html: html.slice(blockStart).trim() })
    cursor = html.length
  }
  pushBareText(html.length)
  return blocks
}

/** Extract each top-level block from an HTML string, keeping tag, text, and verbatim HTML. */
function extractBlocks(html: string): ParaBlock[] {
  return splitTopLevelBlocks(html).map(block => ({
    tag: block.tag,
    text: stripTags(block.html).trim(),
    html: block.html,
  }))
}

/** Alignment key: blocks only align with blocks of the same kind, so a scene break never pairs with an empty paragraph. */
function blockKey(block: ParaBlock): string {
  return `${block.tag}\u0000${block.text}`
}

/** Extract plain-text content of each top-level block from an HTML string. */
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
    tag: o?.tag ?? r?.tag ?? 'p',
    originalHtml: o?.html ?? '',
    revisedHtml: r?.html ?? '',
  }
}

/**
 * Diff two HTML chapter strings block by block.
 *
 * Blocks are aligned by an LCS edit script over their kind and plain text (not
 * by positional index), so an inserted or deleted paragraph does not cascade
 * into spurious full-paragraph diffs on every following paragraph. Each row
 * keeps the verbatim original and revised block HTML so formatting can be
 * preserved when the diff is reassembled.
 *
 * Structural blocks (scene-break `<hr>`, `<blockquote>`, `<pre>`, headings,
 * lists) are rows of their own kind. A scene break the model dropped has no
 * text on either side, so it produces no change and is always kept; a dropped
 * block quote or heading shows as a deletion the writer must accept.
 */
export function diffContent(originalHtml: string, revisedHtml: string): ParagraphDiff[] {
  const orig = extractBlocks(originalHtml)
  const rev = extractBlocks(revisedHtml)
  const ops = diffSeq(orig.map(blockKey), rev.map(blockKey))

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
    // rather than whole-paragraph delete+insert churn. Only blocks of the same
    // kind pair; a structural block never merges into a paragraph rewrite.
    const dels: number[] = []
    const inss: number[] = []
    while (k < ops.length && ops[k].type !== 'equal') {
      if (ops[k].type === 'delete') dels.push(ops[k].ai!)
      else inss.push(ops[k].bi!)
      k++
    }
    let i = 0
    let j = 0
    while (i < dels.length || j < inss.length) {
      const o = i < dels.length ? orig[dels[i]] : null
      const r = j < inss.length ? rev[inss[j]] : null
      if (o && r && o.tag === r.tag) {
        rows.push(makeRow(o, r)); i++; j++
      } else if (o && (!r || o.tag !== 'p')) {
        rows.push(makeRow(o, null)); i++
      } else {
        rows.push(makeRow(null, r)); j++
      }
    }
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
 *  - Mixed (some accepted) → reconstruct from word chunks inside the block's
 *                            own wrapper (`<p>`, heading, `<blockquote><p>`,
 *                            `<pre><code>`). Only in this case is inline
 *                            formatting within the block flattened, and only
 *                            for that one block. Kinds with no sensible
 *                            flattening (lists) take the revised HTML whole.
 *
 * This guarantees that accepting a change in one paragraph can never strip
 * formatting from any paragraph the user did not change, that rejecting
 * everything reproduces the original HTML, and that scene breaks and other
 * structural blocks survive the pass in place.
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
    // Mixed acceptance within a single block: reconstruct from word chunks.
    const text = diff.chunks.map((chunk, ci) => {
      if (chunk.type === 'equal') return chunk.text
      const accepted = changeMap.get(`${paraIdx}-${ci}`) ?? true
      if (chunk.type === 'insert') return accepted ? chunk.text : ''
      if (chunk.type === 'delete') return accepted ? '' : chunk.text
      return ''
    }).join('')
    out.push(wrapMixedBlock(diff.tag, text) ?? diff.revisedHtml)
  })
  return out.join('')
}

function escapeHtml(text: string): string {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

/**
 * Wrap flattened text in the block's own markup. Returns null for kinds that
 * cannot be rebuilt from flat text (lists), where the caller falls back to the
 * revised HTML as a whole.
 */
function wrapMixedBlock(tag: string, text: string): string | null {
  const safe = escapeHtml(text)
  switch (tag) {
    case 'p': case 'h1': case 'h2': case 'h3': case 'h4': case 'h5': case 'h6':
      return `<${tag}>${safe}</${tag}>`
    case 'blockquote':
      return `<blockquote><p>${safe}</p></blockquote>`
    case 'pre':
      return `<pre><code>${safe}</code></pre>`
    default:
      return null
  }
}
