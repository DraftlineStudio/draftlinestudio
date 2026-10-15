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

// Words a period can follow mid-sentence, so the lower-case word after them
// is not a sentence start. Single letters (initials, i.e., p.m.) are skipped
// separately.
const abbreviationBeforePeriod = new Set([
  'mr', 'mrs', 'ms', 'dr', 'st', 'vs', 'etc', 'jr', 'sr', 'prof', 'capt',
  'sgt', 'lt', 'col', 'gen', 'rev', 'hon', 'no', 'vol', 'ch', 'pp', 'approx',
  'dept', 'inc', 'ltd', 'co', 'ave', 'blvd', 'rd', 'mt',
])

// A lower-case letter opening the text or following a sentence-ending mark
// and whitespace. Dialogue tags ("Run!" she yelled) stay legal because the
// closing quote sits between the mark and the tag.
const sentenceStartPattern = /(^\s*|[.!?]\s+)(\p{Ll})/gu

function addSentenceCapitalization(issues: GrammarIssue[], text: string): void {
  sentenceStartPattern.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = sentenceStartPattern.exec(text)) !== null) {
    if (match[1][0] === '.') {
      if (text[match.index - 1] === '.') continue // ellipsis, a legal pause
      const wordBefore = text.slice(0, match.index).match(/(\p{Script=Latin}+)$/u)?.[1]
      if (wordBefore && (wordBefore.length === 1
        || abbreviationBeforePeriod.has(wordBefore.toLocaleLowerCase()))) continue
    }
    const from = match.index + match[0].length - 1
    const word = text.slice(from).match(/^[\p{Script=Latin}\p{N}'’‘ʼ＇]+/u)?.[0] ?? match[2]
    // Leave camel-case brands (iPhone) and alphanumerics (mp3) alone, and
    // leave the pronoun, bare or contracted (i, i'm), to the rule that names it.
    if (/[\p{Lu}\p{N}]/u.test(word.slice(1)) || word === 'i' || /^i['’‘ʼ＇]/.test(word)) continue
    issues.push({
      rule: 'sentence-capitalization',
      kind: 'grammar',
      from,
      to: from + word.length,
      message: 'Sentences begin with a capital letter.',
      replacements: [word[0].toLocaleUpperCase() + word.slice(1)],
    })
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

  addSentenceCapitalization(issues, text)

  addMatches(
    issues,
    text,
    // A standalone lower-case i, bare or opening a contraction (i'm, i'll,
    // i've, i'd). The period lookahead keeps i.e. and roman numerals in
    // references like i.x untouched; the hyphen guards keep i-beam whole.
    /(?<![\p{Script=Latin}\p{N}'’‘ʼ＇-])i(?=['’‘ʼ＇](?:m|ll|ve|d)\b|[\s,;:!?)\]}"”’…]|\.(?!\p{Script=Latin})|$)/gu,
    'pronoun-i',
    'The pronoun “I” is always capitalized.',
    'I',
  )

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
