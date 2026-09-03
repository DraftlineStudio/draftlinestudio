import { describe, it, expect } from 'vitest'
import { autoCast, buildChapterCast, buildRoster, castVoiceKey, type ChapterSpeaker } from './cast'
import { speakerKeyFor, type AttributionResult, type GenderEvidence, type SpeakerKey } from './attribution'
import { DurationEstimator } from './estimates'
import { nextReadAloudSpeed, speedLabel } from './speeds'
import type { BookData } from '../../types/draftline'

function attrWith(lineCounts: Array<[SpeakerKey, number]>): AttributionResult {
  return { sentences: [], lineCounts: new Map(lineCounts), genderEvidence: new Map() }
}

function book(): BookData {
  return {
    version: '2.0',
    metadata: { title: 't', author: 'a' },
    copyright: '',
    front_matter: [],
    body: [{ title: 'One', type: 'chapter', content: '' }],
    back_matter: [],
    story_bible: {
      characters: [
        { id: 'c1', name: 'Renee Alvarez', role: 'protagonist', description: '', appearance: '', personality: '', motivation: '', notes: '', aliases: ['Renee'], chapter_mentions: { 0: 12 } },
        { id: 'c2', name: 'Marcus Webb', role: 'antagonist', description: '', appearance: '', personality: '', motivation: '', notes: '', aliases: [], chapter_mentions: { 1: 4 } },
        { id: 'c3', name: 'The Precinct', role: 'other', description: '', appearance: '', personality: '', motivation: '', notes: '', entity_kind: 'place', chapter_mentions: { 0: 9 } },
      ],
      plot_notes: '',
      timeline: '',
    },
  } as unknown as BookData
}

describe('buildRoster', () => {
  it('includes chapter-mentioned people, text-detected stragglers, and excludes places', () => {
    const roster = buildRoster(book(), 0, 'Marcus Webb waited by the door of the precinct.')
    const keys = roster.map(r => r.key)
    expect(keys).toContain(speakerKeyFor('Renee Alvarez')) // indexed in chapter 0
    expect(keys).toContain(speakerKeyFor('Marcus Webb')) // not indexed here, but in the text
    expect(keys).not.toContain(speakerKeyFor('The Precinct')) // places don't speak
  })
})

describe('buildChapterCast', () => {
  it('orders narrator first, characters by line count, unknown last', () => {
    const roster = buildRoster(book(), 0, 'Marcus Webb spoke to Renee.')
    const attribution = attrWith([
      ['narrator', 40],
      [speakerKeyFor('Renee Alvarez'), 5],
      [speakerKeyFor('Marcus Webb'), 9],
      ['unknown', 2],
    ])
    const cast = buildChapterCast(attribution, roster, { cast_mode: true, voices: { 'marcus webb': 'am_puck' } }, 'af_heart')
    expect(cast.map(s => s.name)).toEqual(['Narration', 'Marcus Webb', 'Renee Alvarez', 'Unknown speaker'])
    expect(cast[0].voice).toBe('af_heart')
    expect(cast[1].voice).toBe('am_puck') // saved assignment resolved
    expect(cast[2].voice).toBeNull() // uncast
    expect(cast[2].color).not.toBe('')
  })

  it('drops speakers with no lines and unknown voice ids', () => {
    const roster = buildRoster(book(), 0, '')
    const attribution = attrWith([['narrator', 10], [speakerKeyFor('Renee Alvarez'), 3]])
    const cast = buildChapterCast(attribution, roster, { cast_mode: true, voices: { 'renee alvarez': 'not_a_voice' } }, 'af_heart')
    expect(cast.map(s => s.name)).toEqual(['Narration', 'Renee Alvarez'])
    expect(cast[1].voice).toBeNull()
  })
})

describe('autoCast', () => {
  const speakersFixture = (voices: Record<string, string | null> = {}): ChapterSpeaker[] => [
    { key: 'narrator', name: 'Narration', color: '', lineCount: 40, voice: 'af_heart' },
    { key: speakerKeyFor('Renee Alvarez'), name: 'Renee Alvarez', color: '#f00', lineCount: 9, voice: voices['renee alvarez'] ?? null },
    { key: speakerKeyFor('Marcus Webb'), name: 'Marcus Webb', color: '#0f0', lineCount: 5, voice: voices['marcus webb'] ?? null },
  ]

  const evidence = new Map<SpeakerKey, GenderEvidence>([
    [speakerKeyFor('Renee Alvarez'), { he: 0, she: 3, they: 0 }],
    [speakerKeyFor('Marcus Webb'), { he: 2, she: 0, they: 0 }],
  ])

  it('is deterministic, gender-aware, and avoids the narrator voice', () => {
    const first = autoCast(speakersFixture(), evidence, 'af_heart')
    const second = autoCast(speakersFixture(), evidence, 'af_heart')
    expect(second).toEqual(first)
    expect(first['renee alvarez']?.startsWith('af_') || first['renee alvarez']?.startsWith('bf_')).toBe(true)
    expect(first['marcus webb']?.startsWith('am_') || first['marcus webb']?.startsWith('bm_')).toBe(true)
    expect(first['renee alvarez']).not.toBe('af_heart')
    expect(first['renee alvarez']).not.toBe(first['marcus webb'])
  })

  it('never overwrites a manual assignment', () => {
    const assignments = autoCast(speakersFixture({ 'marcus webb': 'bm_george' }), evidence, 'af_heart')
    expect(assignments['marcus webb']).toBeUndefined()
    expect(assignments['renee alvarez']).toBeDefined()
    expect(assignments['renee alvarez']).not.toBe('bm_george')
  })

  it('cycles with repeats when speakers outnumber voices', () => {
    const many: ChapterSpeaker[] = Array.from({ length: 10 }, (_, i) => ({
      key: speakerKeyFor(`Person ${String.fromCharCode(65 + i)}`),
      name: `Person ${String.fromCharCode(65 + i)}`,
      color: '#123',
      lineCount: 10 - i,
      voice: null,
    }))
    const assignments = autoCast(many, new Map(), 'af_heart')
    expect(Object.keys(assignments)).toHaveLength(10)
    for (const voice of Object.values(assignments)) expect(voice).toBeTruthy()
  })
})

describe('castVoiceKey', () => {
  it('maps speaker keys to persisted map keys', () => {
    expect(castVoiceKey(speakerKeyFor('Renee Alvarez'))).toBe('renee alvarez')
    expect(castVoiceKey('narrator')).toBeNull()
    expect(castVoiceKey('unknown')).toBeNull()
  })
})

describe('DurationEstimator', () => {
  it('starts at the seed rate and converges to measurements', () => {
    const estimator = new DurationEstimator()
    const seeded = estimator.secondsFor(100, 1)
    expect(seeded).toBeCloseTo(5.8, 1)

    // Feed a consistently slower narrator: 0.1 s/char at 1.0×.
    for (let i = 0; i < 20; i++) estimator.record(100, 10, 1)
    const calibrated = estimator.secondsFor(100, 1)
    expect(calibrated).toBeGreaterThan(9)
    // Speed scales inversely.
    expect(estimator.secondsFor(100, 2)).toBeCloseTo(calibrated / 2, 5)
  })

  it('normalizes recorded durations by speed', () => {
    const a = new DurationEstimator()
    const b = new DurationEstimator()
    for (let i = 0; i < 20; i++) {
      a.record(100, 10, 1) // 10s at 1.0×
      b.record(100, 5, 2) // 5s at 2.0× — same underlying rate
    }
    expect(a.secondsFor(100, 1)).toBeCloseTo(b.secondsFor(100, 1), 5)
  })
})

describe('speeds', () => {
  it('cycles through the stops and wraps', () => {
    expect(nextReadAloudSpeed(1.1)).toBe(1.2)
    expect(nextReadAloudSpeed(2.0)).toBe(0.8)
    expect(nextReadAloudSpeed(1.08)).toBe(1.2) // nearest stop is 1.1
  })

  it('labels speeds tersely', () => {
    expect(speedLabel(2)).toBe('2×')
    expect(speedLabel(1.1)).toBe('1.1×')
  })
})
