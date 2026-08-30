export type GrammarIssueKind = 'grammar' | 'style'

export interface GrammarIssue {
  rule: string
  kind: GrammarIssueKind
  from: number
  to: number
  message: string
  replacements: string[]
}

const changeListeners = new Set<() => void>()
let enabled = true

export function setGrammarCheckEnabled(next: boolean): void {
  if (enabled === next) return
  enabled = next
  changeListeners.forEach(listener => listener())
}

export function onGrammarSettingsChanged(listener: () => void): () => void {
  changeListeners.add(listener)
  return () => changeListeners.delete(listener)
}

function preserveInitialCase(replacement: string, source: string): string {
  if (source[0] === source[0]?.toLocaleUpperCase()) {
    return replacement[0].toLocaleUpperCase() + replacement.slice(1)
  }
  return replacement
}

function addMatches(
  issues: GrammarIssue[],
  text: string,
  pattern: RegExp,
  rule: string,
  message: string | ((match: RegExpExecArray) => string),
  replacement: string | ((match: RegExpExecArray) => string),
  kind: GrammarIssueKind = 'grammar',
): void {
  pattern.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = pattern.exec(text)) !== null) {
    const next = typeof replacement === 'function' ? replacement(match) : replacement
    issues.push({
      rule,
      kind,
      from: match.index,
      to: match.index + match[0].length,
      message: typeof message === 'function' ? message(match) : message,
      replacements: next ? [preserveInitialCase(next, match[0])] : [],
    })
    if (match[0].length === 0) pattern.lastIndex++
  }
}

/**
 * Fast, deterministic checks intended for continuous editor feedback. These
 * rules deliberately target high-confidence mistakes instead of trying to
 * judge an author's voice or sentence style.
 */
export function analyzeGrammar(text: string): GrammarIssue[] {
  if (!enabled || !text) return []

  const issues: GrammarIssue[] = []

  const repeatedWord = /\b([A-Za-z]+(?:['\u2019][A-Za-z]+)?)\s+\1\b/gi
  let repeated: RegExpExecArray | null
  while ((repeated = repeatedWord.exec(text)) !== null) {
    const secondStart = repeated.index + repeated[0].length - repeated[1].length
    issues.push({
      rule: 'repeated-word',
      kind: 'grammar',
      from: secondStart,
      to: secondStart + repeated[1].length,
      message: `Repeated word “${repeated[1]}”.`,
      replacements: [''],
    })
  }

  addMatches(
    issues,
    text,
    /\b(could|should|would|might|must)\s+of\b/gi,
    'modal-of',
    'Use “have” after a modal verb.',
    match => `${match[1]} have`,
  )
  addMatches(issues, text, /\b(I)\s+is\b/g, 'subject-verb-i', '“I” takes “am,” not “is.”', 'I am')
  addMatches(
    issues,
    text,
    /\b(you|we|they)\s+is\b/gi,
    'subject-verb-plural',
    'This subject takes “are,” not “is.”',
    match => `${match[1]} are`,
  )
  addMatches(
    issues,
    text,
    /\b(he|she|it)\s+are\b/gi,
    'subject-verb-singular',
    'This subject takes “is,” not “are.”',
    match => `${match[1]} is`,
  )
  addMatches(
    issues,
    text,
    /\b(more\s+better|more\s+worse|most\s+best|most\s+worst)\b/gi,
    'double-comparative',
    'Avoid a double comparative or superlative.',
    match => match[0].replace(/^(more|most)\s+/i, ''),
  )
  addMatches(issues, text, /\birregardless\b/gi, 'irregardless', 'Use “regardless” in standard prose.', 'regardless', 'style')

  const articlePattern = /\b(a|an)\s+([A-Za-z][A-Za-z'-]{2,})\b/gi
  let article: RegExpExecArray | null
  while ((article = articlePattern.exec(text)) !== null) {
    const current = article[1].toLocaleLowerCase()
    const noun = article[2].toLocaleLowerCase()
    const vowelSound = /^[aeiou]/.test(noun)
      && !/^(uni([^n]|$)|use|user|usual|euro|one|once)/.test(noun)
    const silentH = /^(honest|honor|hour|heir)/.test(noun)
    const expected = vowelSound || silentH ? 'an' : 'a'
    if (current === expected) continue
    issues.push({
      rule: 'article-agreement',
      kind: 'grammar',
      from: article.index,
      to: article.index + current.length,
      message: `Use “${expected}” before “${article[2]}.”`,
      replacements: [preserveInitialCase(expected, article[1])],
    })
  }

  addMatches(
    issues,
    text,
    /([!?])\1{1,}/g,
    'repeated-punctuation',
    'Repeated punctuation can be reduced.',
    match => match[1],
    'style',
  )

  return issues.sort((left, right) => left.from - right.from || left.to - right.to)
}
