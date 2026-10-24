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
    // leave the pronoun, bare or contracted (i, i'm), to the rule that names
    // it. An apostrophe-less contraction (im, dont) defers too: the missing
    // apostrophe is the real error, and "Im" is no fix at all.
    if (/[\p{Lu}\p{N}]/u.test(word.slice(1)) || word === 'i' || /^i['’‘ʼ＇]/.test(word)
      || apostropheLessContractions[word.toLocaleLowerCase()] !== undefined) continue
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

// Contractions typed without their apostrophe. Only forms that are not
// themselves words belong here — cant, wont, ill, hell, shell, wed, and were
// all mean something bare — except that "im" and "ive" are in the SCOWL
// dictionaries, so this rule is the only thing standing between them and a
// clean page. "its" and sentence-opening "were" get their own rules below.
const apostropheLessContractions: Record<string, string> = {
  im: "I'm", ive: "I've",
  youre: "you're", youd: "you'd", youll: "you'll", youve: "you've",
  theyre: "they're", theyd: "they'd", theyll: "they'll", theyve: "they've",
  weve: "we've", hes: "he's", shes: "she's",
  dont: "don't", doesnt: "doesn't", didnt: "didn't",
  isnt: "isn't", wasnt: "wasn't", werent: "weren't", arent: "aren't",
  aint: "ain't", hasnt: "hasn't", havent: "haven't", hadnt: "hadn't",
  couldnt: "couldn't", wouldnt: "wouldn't", shouldnt: "shouldn't",
  mustnt: "mustn't", neednt: "needn't",
  whos: "who's", whats: "what's", wheres: "where's",
  theres: "there's", thats: "that's", heres: "here's",
}
const apostropheLessPattern = new RegExp(
  `\\b(?:${Object.keys(apostropheLessContractions).join('|')})\\b`, 'gi')

function addApostropheLessContractions(issues: GrammarIssue[], text: string): void {
  apostropheLessPattern.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = apostropheLessPattern.exec(text)) !== null) {
    // All caps is an acronym or shouting (IM, DONT WALK), not a contraction.
    if (match[0].length > 1 && match[0] === match[0].toLocaleUpperCase()) continue
    const fix = apostropheLessContractions[match[0].toLocaleLowerCase()]
    issues.push({
      rule: 'missing-apostrophe',
      kind: 'grammar',
      from: match.index,
      to: match.index + match[0].length,
      message: `Missing apostrophe: “${fix}.”`,
      replacements: [preserveInitialCase(fix, match[0])],
    })
  }
}

// -ing words that are commonly nouns after a possessive (its opening, its
// sting, its thing), so "its" before them proves nothing. Plurals (workings,
// surroundings) never match the pattern's ing\b and need no entry.
const ingNouns = new Set([
  'thing', 'king', 'ring', 'wing', 'string', 'spring', 'sting', 'sibling',
  'darling', 'duckling', 'dumpling', 'ceiling', 'evening', 'morning',
  'inning', 'lightning', 'opening', 'ending', 'beginning', 'meaning',
  'making', 'timing', 'bidding', 'calling', 'footing', 'standing',
  'undoing', 'coming', 'passing', 'landing', 'heading', 'housing',
  'casing', 'lining', 'setting', 'warning', 'building', 'painting',
  'drawing', 'writing', 'wedding', 'clothing', 'feeling', 'offering',
  'blessing', 'crossing', 'dressing', 'railing', 'handwriting',
])

function addItsIngConfusion(issues: GrammarIssue[], text: string): void {
  // "its" before what reads as a verb (its raining there) is very
  // likely "it's." A gerund can also be a noun, so this is the softer style
  // tier; the determiner cases below are certain and stay grammar.
  const pattern = /\bits\s+(\p{Script=Latin}+ing)\b(?!-)/giu
  let match: RegExpExecArray | null
  while ((match = pattern.exec(text)) !== null) {
    if (ingNouns.has(match[1].toLocaleLowerCase())) continue
    const to = match.index + 3
    issues.push({
      rule: 'its-contraction',
      kind: 'style',
      from: match.index,
      to,
      message: '“Its” is the possessive. For “it is,” write “it’s.”',
      replacements: [preserveInitialCase("it's", match[0])],
    })
  }
}

// Speech verbs that mark a dialogue tag. Kept to verbs that rarely open an
// unrelated sentence directly after a quote.
const speechVerbs = 'said|asked|replied|answered|whispered|shouted|yelled|called|muttered|murmured|snapped|quipped|added|continued|repeated|cried|screamed|croaked|groaned|growled|hissed|sighed|offered|agreed|admitted|insisted|warned|promised|teased|joked|laughed|chuckled|demanded|ordered|begged|pleaded|stammered|stuttered|mumbled|grumbled|barked|bellowed|hollered|drawled|countered|retorted|interjected|observed|noted|remarked|explained|suggested|mused|echoed|finished'

// When the verb takes an object, the words after the quote are their own
// sentence (He said nothing. She asked the guide.), not a tag.
const tagObjectFollowers = /^\s+(?:that|a|an|the|his|her|its|my|our|your|their|some|something|nothing|anything|everything|this|these|those|it|him|them|us|me|you|goodbye|hello|yes|no)\b/i

const lowercaseTagPronouns = new Set(['He', 'She', 'They', 'It', 'We', 'You'])

function addDialogueTagPunctuation(issues: GrammarIssue[], text: string): void {
  // A period closing a quote that a tag follows: "Leave it." Marcus said.
  // The period becomes a comma, and a bare pronoun loses its capital.
  const periodTag = new RegExp(`\\.(["”])\\s+([\\p{Script=Latin}]+)\\s+(${speechVerbs})\\b`, 'gu')
  let match: RegExpExecArray | null
  while ((match = periodTag.exec(text)) !== null) {
    if (tagObjectFollowers.test(text.slice(match.index + match[0].length))) continue
    const speaker = lowercaseTagPronouns.has(match[2])
      ? match[2].toLocaleLowerCase()
      : match[2]
    issues.push({
      rule: 'dialogue-tag-punctuation',
      kind: 'grammar',
      from: match.index,
      to: match.index + match[0].length,
      message: 'A dialogue tag follows, so the quote ends with a comma.',
      replacements: [`,${match[1]} ${speaker} ${match[3]}`],
    })
  }

  // The same tag after ! or ? keeps its mark but not the capital: "Run!" He
  // said. Names stay capitalized, so only bare pronouns are certain.
  const shoutTag = new RegExp(`([!?]["”])\\s+(He|She|They|It|We|You)\\s+(${speechVerbs})\\b`, 'gu')
  while ((match = shoutTag.exec(text)) !== null) {
    if (tagObjectFollowers.test(text.slice(match.index + match[0].length))) continue
    issues.push({
      rule: 'dialogue-tag-punctuation',
      kind: 'grammar',
      from: match.index,
      to: match.index + match[0].length,
      message: 'A dialogue tag continues the sentence, so the pronoun is lowercase.',
      replacements: [`${match[1]} ${match[2].toLocaleLowerCase()} ${match[3]}`],
    })
  }
}

// Adverbs that are never proper nouns, so a capital after a comma is a typo:
// "Relax, Just breathe."
const neverProperAfterComma = new Set([
  'Just', 'Very', 'Really', 'Quite', 'Suddenly', 'Maybe', 'Perhaps',
  'Probably', 'Actually', 'Basically', 'Honestly', 'Seriously', 'Literally',
  'Definitely', 'Almost', 'Nearly', 'Barely', 'Hardly', 'Mostly', 'Usually',
  'Finally', 'Eventually', 'Certainly', 'Obviously', 'Apparently',
  'Especially', 'Gonna', 'Gotta',
])
const commaCapitalPattern = /,\s+(\p{Lu}\p{Script=Latin}+)\b/gu

function addCommaCapitalization(issues: GrammarIssue[], text: string): void {
  commaCapitalPattern.lastIndex = 0
  let match: RegExpExecArray | null
  while ((match = commaCapitalPattern.exec(text)) !== null) {
    if (!neverProperAfterComma.has(match[1])) continue
    const from = match.index + match[0].length - match[1].length
    issues.push({
      rule: 'comma-capitalization',
      kind: 'grammar',
      from,
      to: from + match[1].length,
      message: 'A sentence does not restart after a comma.',
      replacements: [match[1].toLocaleLowerCase()],
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
  addApostropheLessContractions(issues, text)
  addItsIngConfusion(issues, text)
  addDialogueTagPunctuation(issues, text)
  addCommaCapitalization(issues, text)

  addMatches(
    issues,
    text,
    // A possessive can never precede a determiner or these verb forms, so
    // this tier is certain; the gerund case lives in the style tier above.
    /\bits(?=\s+(?:a|an|the|been|not|gonna|gotta)\b(?!-))/gi,
    'its-contraction',
    '“Its” is the possessive. For “it is,” write “it’s.”',
    "it's",
  )
  addMatches(
    issues,
    text,
    // "Were not" or "Were gonna" opening a sentence is "We're" missing its
    // apostrophe; inside a sentence, bare "were" is an ordinary past tense.
    /(?<=^["“‘'\s]*|[.!?]["”’']?\s+|["“])Were(?=\s+(?:not|gonna)\b)/gu,
    'were-contraction',
    'This looks like “We’re” (we are).',
    "We're",
  )
  addMatches(
    issues,
    text,
    // An exclamation or question mark followed by a period (enough!.) —
    // the stronger mark already ended the sentence. Ellipses stay legal.
    /([!?])\.(?!\.)/g,
    'conflicting-end-marks',
    'One end mark is enough.',
    match => match[1],
  )
  addMatches(
    issues,
    text,
    /\.{3,}/g,
    'ellipsis',
    'Use an ellipsis character (…).',
    '…',
    'style',
  )

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
    const silentH = /^(honest|honou?r|hour|heir)/.test(noun)
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
