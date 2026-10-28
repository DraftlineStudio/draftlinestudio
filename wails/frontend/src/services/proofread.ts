import type { Editor } from '@tiptap/core'
import { analyzeGrammar } from './grammarCheck'
import { checkWord, isLoaded as isSpellCheckLoaded } from './spellCheck'
import { blockText } from '../extensions/GrammarCheck'
import { placeStyleFindings } from '../extensions/StyleFindings'
import type { Section, StyleData, StyleFinding } from '../types/draftline'

// The proofread walk: every detection in the open chapter, in reading
// order, with enough context to act on from the sidebar. Spelling and
// grammar come from the same analyses that draw the underlines, and the
// style layer's findings join them, so the walk and the squiggles always
// agree.

export interface Detection {
  kind: 'spelling' | 'grammar' | 'style'
  rule: string
  from: number
  to: number
  text: string
  message: string
  replacements: string[]
  // Text around the detection inside its paragraph, for the card.
  before: string
  after: string
  // Unique list key: the position-independent detectionKey, with an ordinal
  // appended when identical contexts repeat in the chapter.
  key: string
  // Set for findings that came from the analysis pipeline (echoes), which
  // carry their own dismissal actions.
  finding?: StyleFinding
}

// Keep in sync with spellWordPattern in services/spellCheck.ts.
const wordPattern = /\p{Script=Latin}+(?:['’‘ʼ＇]\p{Script=Latin}+)*/gu

const contextChars = 48

/** The chapter index the analysis pipeline would give the open chapter:
 * front matter, body, then back matter. */
export function globalChapterIndex(section: Section, index: number, frontCount: number, bodyCount: number): number {
  if (section === 'front_matter') return index
  if (section === 'body') return frontCount + index
  if (section === 'back_matter') return frontCount + bodyCount + index
  return -1
}

/** The style layer's findings for one chapter, minus everything the author
 * has silenced. */
export function styleFindingsFor(style: StyleData | undefined, globalIndex: number): StyleFinding[] {
  if (!style?.findings?.length || globalIndex < 0) return []
  const dismissed = new Set((style.decisions ?? []).map(decision => decision.finding_id))
  const ignoredWords = new Set((style.ignored_words ?? []).map(word => word.toLowerCase()))
  return style.findings.filter(finding =>
    finding.chapter_index === globalIndex && !dismissed.has(finding.id)
    && !ignoredWords.has(finding.snippet.toLowerCase()))
}

/** A position-independent key for a detection, so a session skip survives
 * the position shifts that applying earlier fixes causes. */
export function detectionKey(detection: Detection): string {
  return `${detection.rule}|${detection.text}|${detection.before.slice(-12)}|${detection.after.slice(0, 12)}`
}

export function collectDetections(editor: Editor, backendFindings: StyleFinding[]): Detection[] {
  const detections: Detection[] = []
  const doc = editor.state.doc

  doc.descendants((node, pos) => {
    if (!node.isTextblock) return

    const text = blockText(node)
    const slice = (from: number, to: number) => text.slice(Math.max(0, from), Math.min(text.length, to))

    for (const issue of analyzeGrammar(text)) {
      detections.push({
        kind: issue.kind === 'style' ? 'style' : 'grammar',
        rule: issue.rule,
        from: pos + 1 + issue.from,
        to: pos + 1 + issue.to,
        text: text.slice(issue.from, issue.to),
        message: issue.message,
        replacements: issue.replacements,
        key: '',
        before: slice(issue.from - contextChars, issue.from),
        after: slice(issue.to, issue.to + contextChars),
      })
    }

    if (isSpellCheckLoaded()) {
      // Per text node like the SpellCheck extension, so offsets agree with
      // its underlines; numbers glued to words stay unflagged.
      node.descendants((child, childPos) => {
        if (!child.isText) return
        const childText = child.text || ''
        const base = pos + 1 + childPos
        wordPattern.lastIndex = 0
        let match: RegExpExecArray | null
        while ((match = wordPattern.exec(childText)) !== null) {
          const beforeChar = childText[match.index - 1]
          const afterChar = childText[match.index + match[0].length]
          if (/\d/.test(beforeChar) || /\d/.test(afterChar)) continue
          if (checkWord(match[0])) continue
          const inBlock = base - (pos + 1) + match.index
          detections.push({
            kind: 'spelling',
            rule: 'spelling',
            from: base + match.index,
            to: base + match.index + match[0].length,
            text: match[0],
            message: `“${match[0]}” is not in the dictionary.`,
            replacements: [],
            key: '',
            before: slice(inBlock - contextChars, inBlock),
            after: slice(inBlock + match[0].length, inBlock + match[0].length + contextChars),
          })
        }
      })
    }
    return false
  })

  for (const placed of placeStyleFindings(doc, backendFindings)) {
    const $from = doc.resolve(placed.from)
    const blockStart = $from.start()
    const text = blockText($from.parent)
    const inBlock = placed.from - blockStart
    detections.push({
      kind: 'style',
      rule: placed.finding.rule,
      from: placed.from,
      to: placed.to,
      text: placed.finding.snippet,
      message: placed.finding.message,
      replacements: placed.finding.replacements ?? [],
      key: '',
      before: text.slice(Math.max(0, inBlock - contextChars), inBlock),
      after: text.slice(inBlock + placed.finding.snippet.length, inBlock + placed.finding.snippet.length + contextChars),
      finding: placed.finding,
    })
  }

  detections.sort((left, right) => left.from - right.from || left.to - right.to)
  // Repeated contexts ("word" "word") produce identical detectionKeys; an
  // ordinal keeps every list key unique and stable in document order.
  const seen = new Map<string, number>()
  for (const detection of detections) {
    const base = detectionKey(detection)
    const ordinal = seen.get(base) ?? 0
    seen.set(base, ordinal + 1)
    detection.key = ordinal === 0 ? base : `${base}#${ordinal}`
  }
  return detections
}

/** The rules the one-click typography fix resolves. */
export function isTypographyRule(rule: string): boolean {
  return rule === 'straight-quote' || rule === 'ellipsis' || rule === 'double-hyphen'
}

/** How many of the chapter's detections the one-click typography fix would
 * resolve. */
export function typographyFixableCount(detections: Detection[]): number {
  return detections.filter(detection => isTypographyRule(detection.rule)).length
}

/** Applies one replacement and returns where the walk should resume. */
export function applyDetection(editor: Editor, detection: Detection, replacement: string): number {
  const { state } = editor
  editor.view.dispatch(state.tr.insertText(replacement, detection.from, detection.to))
  return detection.from
}
