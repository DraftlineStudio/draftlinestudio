/**
 * Heuristic-based AI Detection
 *
 * Analyzes text for patterns commonly associated with AI-generated content.
 * This is NOT a replacement for ML-based detectors like Pangram or GPTZero,
 * but provides a quick, free, offline "canary" warning.
 *
 * Metrics analyzed:
 * - Burstiness: AI text tends to have uniform sentence complexity
 * - Vocabulary richness: AI often uses predictable, "safe" word choices
 * - Repetition: AI tends to repeat phrases and structures
 * - Sentence variety: AI has less variation in sentence patterns
 * - Transition patterns: AI overuses certain transitional phrases
 */

export interface AIDetectionResult {
  score: number  // 0-100, where 0 = definitely human, 100 = likely AI
  confidence: 'low' | 'medium' | 'high'
  breakdown: {
    burstiness: number      // 0-100
    vocabularyRichness: number  // 0-100
    repetition: number      // 0-100
    sentenceVariety: number // 0-100
    transitionPatterns: number // 0-100
  }
  flags: string[]  // Specific concerns detected
}

// Common AI transitional phrases (overused by LLMs)
const AI_TRANSITION_PHRASES = [
  'however', 'moreover', 'furthermore', 'additionally', 'consequently',
  'nevertheless', 'nonetheless', 'in conclusion', 'in summary',
  'it is worth noting', 'it is important to note', 'it should be noted',
  'in this context', 'in light of', 'with that being said',
  'that said', 'having said that', 'on the other hand',
  'as a result', 'as such', 'thus', 'hence', 'therefore',
  'in essence', 'essentially', 'fundamentally', 'ultimately',
  'interestingly', 'notably', 'significantly', 'importantly',
  'delve', 'delves', 'delving', 'realm', 'tapestry', 'multifaceted',
  'nuanced', 'intricate', 'comprehensive', 'robust', 'leverage',
  'paradigm', 'synergy', 'holistic', 'seamless', 'cutting-edge',
]

// Phrases that are red flags for AI fiction specifically
const AI_FICTION_PHRASES = [
  'a sense of', 'a wave of', 'a surge of', 'a pang of',
  'couldn\'t help but', 'found himself', 'found herself',
  'let out a breath', 'released a breath', 'exhaled a breath',
  'the weight of', 'a mix of', 'a mixture of',
  'something akin to', 'nothing short of',
  'in that moment', 'at that moment', 'in this moment',
  'eyes widened', 'heart raced', 'heart pounded',
  'a small smile', 'a faint smile', 'a slight smile',
  'nodded slowly', 'shook his head', 'shook her head',
]

/**
 * Extract plain text from HTML
 */
function htmlToText(html: string): string {
  const div = document.createElement('div')
  div.innerHTML = html
  return div.textContent || div.innerText || ''
}

/**
 * Split text into sentences
 */
function getSentences(text: string): string[] {
  return text
    .replace(/([.!?])\s+/g, '$1|')
    .split('|')
    .map(s => s.trim())
    .filter(s => s.length > 0)
}

/**
 * Split text into words
 */
function getWords(text: string): string[] {
  return text
    .toLowerCase()
    .replace(/[^\w\s'-]/g, ' ')
    .split(/\s+/)
    .filter(w => w.length > 0)
}

/**
 * Calculate standard deviation
 */
function standardDeviation(values: number[]): number {
  if (values.length === 0) return 0
  const mean = values.reduce((a, b) => a + b, 0) / values.length
  const squaredDiffs = values.map(v => Math.pow(v - mean, 2))
  return Math.sqrt(squaredDiffs.reduce((a, b) => a + b, 0) / values.length)
}

/**
 * Calculate burstiness score
 * AI text tends to have uniform sentence lengths; human writing is "bursty"
 * Lower variation = more AI-like
 */
function calculateBurstiness(sentences: string[]): number {
  if (sentences.length < 3) return 50 // Not enough data

  const lengths = sentences.map(s => s.split(/\s+/).length)
  const stdDev = standardDeviation(lengths)
  const mean = lengths.reduce((a, b) => a + b, 0) / lengths.length

  // Coefficient of variation (normalized standard deviation)
  const cv = mean > 0 ? stdDev / mean : 0

  // Human writing typically has CV > 0.5, AI tends to be < 0.3
  // Map to 0-100 where lower CV = higher AI score
  if (cv > 0.6) return 10  // Very bursty = human-like
  if (cv > 0.45) return 25
  if (cv > 0.35) return 40
  if (cv > 0.25) return 60
  if (cv > 0.15) return 80
  return 95  // Very uniform = AI-like
}

/**
 * Calculate vocabulary richness
 * AI tends to use a narrower, more "safe" vocabulary
 */
function calculateVocabularyRichness(words: string[]): number {
  if (words.length < 50) return 50 // Not enough data

  const uniqueWords = new Set(words)
  const typeTokenRatio = uniqueWords.size / words.length

  // Also check for hapax legomena (words appearing only once)
  const wordCounts = new Map<string, number>()
  words.forEach(w => wordCounts.set(w, (wordCounts.get(w) || 0) + 1))
  const hapaxCount = Array.from(wordCounts.values()).filter(c => c === 1).length
  const hapaxRatio = hapaxCount / uniqueWords.size

  // Combine metrics
  // Higher TTR and hapax ratio = more human-like
  const combinedScore = (typeTokenRatio * 0.6 + hapaxRatio * 0.4)

  // Map to AI score (inverse - higher richness = lower AI score)
  if (combinedScore > 0.7) return 10
  if (combinedScore > 0.55) return 25
  if (combinedScore > 0.45) return 40
  if (combinedScore > 0.35) return 60
  if (combinedScore > 0.25) return 80
  return 95
}

/**
 * Calculate repetition score
 * AI often repeats phrases and sentence structures
 */
function calculateRepetition(text: string, sentences: string[]): number {
  const flags: string[] = []
  let repetitionScore = 0

  // Check for repeated phrases (3+ words)
  const phrases = new Map<string, number>()
  const words = text.toLowerCase().split(/\s+/)

  for (let i = 0; i < words.length - 2; i++) {
    const phrase = words.slice(i, i + 3).join(' ')
    if (phrase.length > 8) { // Skip very short phrases
      phrases.set(phrase, (phrases.get(phrase) || 0) + 1)
    }
  }

  const repeatedPhrases = Array.from(phrases.entries()).filter(([_, count]) => count > 2)
  if (repeatedPhrases.length > 3) repetitionScore += 30
  else if (repeatedPhrases.length > 1) repetitionScore += 15

  // Check for repeated sentence starters
  const starters = sentences.map(s => {
    const firstWords = s.split(/\s+/).slice(0, 2).join(' ').toLowerCase()
    return firstWords
  })
  const starterCounts = new Map<string, number>()
  starters.forEach(s => starterCounts.set(s, (starterCounts.get(s) || 0) + 1))

  const repeatedStarters = Array.from(starterCounts.values()).filter(c => c > 2).length
  if (repeatedStarters > 2) repetitionScore += 30
  else if (repeatedStarters > 0) repetitionScore += 15

  // Check for paragraph structure repetition
  const paragraphs = text.split(/\n\s*\n/)
  if (paragraphs.length >= 3) {
    const paraSentenceCounts = paragraphs.map(p => getSentences(p).length)
    const paraStdDev = standardDeviation(paraSentenceCounts)
    if (paraStdDev < 1) repetitionScore += 20 // Very uniform paragraph lengths
  }

  return Math.min(100, repetitionScore)
}

/**
 * Calculate sentence variety score
 */
function calculateSentenceVariety(sentences: string[]): number {
  if (sentences.length < 5) return 50

  let varietyScore = 0

  // Check sentence length variety
  const lengths = sentences.map(s => s.split(/\s+/).length)
  const minLen = Math.min(...lengths)
  const maxLen = Math.max(...lengths)
  const range = maxLen - minLen

  if (range < 5) varietyScore += 40  // Very little variety
  else if (range < 10) varietyScore += 20

  // Check for variety in sentence starters (first word)
  const firstWords = sentences.map(s => s.split(/\s+/)[0]?.toLowerCase() || '')
  const uniqueFirstWords = new Set(firstWords)
  const starterVariety = uniqueFirstWords.size / sentences.length

  if (starterVariety < 0.4) varietyScore += 40
  else if (starterVariety < 0.6) varietyScore += 20

  // Check punctuation variety
  const hasQuestion = sentences.some(s => s.endsWith('?'))
  const hasExclamation = sentences.some(s => s.endsWith('!'))
  const hasSemicolon = sentences.some(s => s.includes(';'))
  const hasDash = sentences.some(s => s.includes('—') || s.includes('--'))

  const punctVariety = [hasQuestion, hasExclamation, hasSemicolon, hasDash].filter(Boolean).length
  if (punctVariety === 0) varietyScore += 20

  return Math.min(100, varietyScore)
}

/**
 * Check for AI transition patterns
 */
function calculateTransitionPatterns(text: string): { score: number; flags: string[] } {
  const lowerText = text.toLowerCase()
  const wordCount = getWords(text).length
  const flags: string[] = []

  let matchCount = 0

  // Check for overused AI transitions
  for (const phrase of AI_TRANSITION_PHRASES) {
    const regex = new RegExp(`\\b${phrase}\\b`, 'gi')
    const matches = (text.match(regex) || []).length
    if (matches > 0) {
      matchCount += matches
      if (matches > 2) {
        flags.push(`Overused: "${phrase}" (${matches}x)`)
      }
    }
  }

  // Check for AI fiction clichés
  for (const phrase of AI_FICTION_PHRASES) {
    const regex = new RegExp(phrase.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi')
    const matches = (text.match(regex) || []).length
    if (matches > 0) {
      matchCount += matches * 1.5 // Weight fiction clichés higher
      if (matches > 1) {
        flags.push(`AI cliché: "${phrase}" (${matches}x)`)
      }
    }
  }

  // Normalize by word count
  const density = (matchCount / wordCount) * 100

  let score = 0
  if (density > 3) score = 90
  else if (density > 2) score = 70
  else if (density > 1) score = 50
  else if (density > 0.5) score = 30
  else score = 10

  return { score, flags }
}

/**
 * Main analysis function
 */
export function analyzeText(html: string): AIDetectionResult {
  const text = htmlToText(html)

  if (text.length < 100) {
    return {
      score: 50,
      confidence: 'low',
      breakdown: {
        burstiness: 50,
        vocabularyRichness: 50,
        repetition: 50,
        sentenceVariety: 50,
        transitionPatterns: 50,
      },
      flags: ['Text too short for reliable analysis'],
    }
  }

  const sentences = getSentences(text)
  const words = getWords(text)

  // Calculate individual metrics
  const burstiness = calculateBurstiness(sentences)
  const vocabularyRichness = calculateVocabularyRichness(words)
  const repetition = calculateRepetition(text, sentences)
  const sentenceVariety = calculateSentenceVariety(sentences)
  const { score: transitionPatterns, flags: transitionFlags } = calculateTransitionPatterns(text)

  // Weighted average for final score
  const weights = {
    burstiness: 0.20,
    vocabularyRichness: 0.15,
    repetition: 0.20,
    sentenceVariety: 0.15,
    transitionPatterns: 0.30, // Transition patterns are strong indicators
  }

  const score = Math.round(
    burstiness * weights.burstiness +
    vocabularyRichness * weights.vocabularyRichness +
    repetition * weights.repetition +
    sentenceVariety * weights.sentenceVariety +
    transitionPatterns * weights.transitionPatterns
  )

  // Determine confidence based on text length and score extremity
  let confidence: 'low' | 'medium' | 'high'
  if (words.length < 200) {
    confidence = 'low'
  } else if (words.length < 500) {
    confidence = 'medium'
  } else {
    confidence = 'high'
  }

  // Collect all flags
  const flags: string[] = [...transitionFlags]

  if (burstiness > 70) flags.push('Low sentence length variation')
  if (vocabularyRichness > 70) flags.push('Limited vocabulary diversity')
  if (repetition > 60) flags.push('Repetitive patterns detected')
  if (sentenceVariety < 30 && sentenceVariety > 70) flags.push('Uniform sentence structure')

  return {
    score,
    confidence,
    breakdown: {
      burstiness,
      vocabularyRichness,
      repetition,
      sentenceVariety,
      transitionPatterns,
    },
    flags,
  }
}

/**
 * Get color for score display
 */
export function getScoreColor(score: number): string {
  if (score <= 25) return '#4ade80' // Green
  if (score <= 50) return '#a3e635' // Lime
  if (score <= 70) return '#facc15' // Yellow
  if (score <= 85) return '#fb923c' // Orange
  return '#ef4444' // Red
}

/**
 * Get label for score
 */
export function getScoreLabel(score: number): string {
  if (score <= 20) return 'Very Human'
  if (score <= 40) return 'Likely Human'
  if (score <= 60) return 'Mixed Signals'
  if (score <= 80) return 'AI Patterns'
  return 'Likely AI'
}
