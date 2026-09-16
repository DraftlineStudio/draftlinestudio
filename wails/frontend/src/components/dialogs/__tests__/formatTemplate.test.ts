import { describe, expect, it } from 'vitest'
import type { Edition, EditionFormat, EditionIndex, EditionSnapshot } from '../../../types/draftline'
import {
  advancedFor, choiceGoesCustom, choicesFor, chooseTemplate, customPatch, isSpecification, kindOf,
  preferencePatch, setAdvanced, standardPatch, templateEditPatch, templateFor,
  templateSummary, typesettingOf,
} from '../formatTemplate'
import { defaultWizardOptions, exportSettingsBlock, wizardOptionsForFormat } from '../exportSource'
import {
  RELEASE_STEPS, draftDrift, editionSnapshot, editionSnapshotRows, formatsOnSnapshot,
  releaseConfirmed, releaseStepLabel,
} from '../snapshotModel'

const format = (patch: Partial<EditionFormat> = {}): EditionFormat =>
  ({ id: 'f1', kind: 'print', format: 'Paperback', ...patch })

const edition = (formats: EditionFormat[] = [format()]): Edition =>
  ({ id: 'ed1', label: 'First edition', year: '2026', status: 'Published', formats })

describe('a format’s typeset template', () => {
  it('is the export wizard’s own settings, read from the record', () => {
    const one = edition()
    expect(templateFor(one, one.formats[0])).toEqual(wizardOptionsForFormat(one, one.formats[0]))
  })

  it('writes back into the same block the export wizard reads', () => {
    const options = defaultWizardOptions()
    expect(templateEditPatch(options).export_settings).toEqual(exportSettingsBlock(options))
  })
})

// The old screen worked "custom" out by comparing the settings against the
// defaults. It could not: applying the standard read back as Custom, and an
// ebook whose standard equalled the defaults could never be moved off it.
describe('standard or custom is a choice, not a comparison', () => {
  it('treats a record that has never said as standard', () => {
    expect(typesettingOf(format())).toBe('standard')
    expect(typesettingOf(format({ typesetting: '' }))).toBe('standard')
    expect(typesettingOf(format({ typesetting: 'Custom' }))).toBe('custom')
  })

  it('moves to custom without changing a single setting', () => {
    const patch = customPatch()
    expect(patch.typesetting).toBe('custom')
    expect(patch.export_settings).toBeUndefined()
  })

  it('goes back to the standard by restoring the whole template', () => {
    const moved = defaultWizardOptions()
    moved.print = { ...moved.print, fontSize: 12, runningHeaders: false, gutterMargin: '0.2' }
    const patch = standardPatch(moved)
    expect(patch.typesetting).toBe('standard')
    const back = (patch.export_settings as Record<string, Record<string, unknown>>)['print-pdf']
    expect(back.fontSize).toBe(defaultWizardOptions().print.fontSize)
    expect(back.runningHeaders).toBe(true)
    expect(back.gutterMargin).toBe(defaultWizardOptions().print.gutterMargin)
  })

  // The trim is the size of the book, chosen on its own card. Turning a 6 × 9
  // paperback back into 5.5 × 8.5 because somebody clicked Industry standard
  // would be a worse surprise than any it prevents.
  it('keeps the print size when it puts everything else back', () => {
    const moved = defaultWizardOptions()
    moved.print = { ...moved.print, trimSize: '6x9', customWidth: '6', customHeight: '9', dropCap: false }
    const back = (standardPatch(moved).export_settings as Record<string, Record<string, unknown>>)['print-pdf']
    expect(back.trimSize).toBe('6x9')
    expect(back.customWidth).toBe('6')
    expect(back.dropCap).toBe(true)
  })

  it('records an edit to a setting as custom, with the edit', () => {
    const edited = defaultWizardOptions()
    edited.print = { ...edited.print, gutterMargin: '1.1' }
    const patch = templateEditPatch(edited)
    expect(patch.typesetting).toBe('custom')
    expect((patch.export_settings as Record<string, Record<string, unknown>>)['print-pdf'].gutterMargin).toBe('1.1')
  })

  it('knows which advanced rows are the published object rather than typesetting', () => {
    const groups = advancedFor(format({ page_count: '412' }), defaultWizardOptions())
    const block = groups.find(g => g.label === 'Book block')!
    expect(block.rows.filter(r => r.kind !== 'static').every(isSpecification)).toBe(true)
    const margins = groups.find(g => g.label === 'Margins')!
    expect(margins.rows.some(isSpecification)).toBe(false)
  })
})

describe('the choices on the card', () => {
  it('asks a print format about its size and its running heads, and an ebook neither', () => {
    const options = defaultWizardOptions()
    expect(choicesFor(format(), options).map(c => c.id))
      .toEqual(['trim', 'dropcap', 'scene', 'chapter', 'heads', 'folio'])
    expect(choicesFor(format({ kind: 'ebook', format: 'eBook' }), options).map(c => c.id))
      .toEqual(['dropcap', 'scene', 'chapter'])
  })

  it('marks a control no exporter reads yet rather than hiding it', () => {
    const options = defaultWizardOptions()
    // A print scene break and an ebook scene break both reach a renderer now.
    expect(choicesFor(format(), options).find(c => c.id === 'scene')?.held).toBe(false)
    expect(choicesFor(format({ kind: 'ebook' }), options).find(c => c.id === 'scene')?.held).toBe(false)
    // A narration script's layout is not written yet, and says so.
    expect(choicesFor(format({ kind: 'audio' }), options).find(c => c.id === 'scene')?.held).toBe(true)
  })

  // These are preferences. A paperback at 6 x 9 with a short rule between
  // scenes is still set the industry-standard way, and saying otherwise is
  // what made the cards feel like traps.
  it('changes a scene break and a chapter opening on a printed page', () => {
    const options = defaultWizardOptions()
    expect(chooseTemplate(options, format(), 'scene', 'rule').print.sceneBreakStyle).toBe('rule')
    expect(chooseTemplate(options, format(), 'chapter', 'compact').print.chapterStyle).toBe('compact')
  })

  // Every sub-label on a row of cards is a short phrase saying what the option
  // does. One of them used to render a mock page spread with the open book's
  // own title in it, which matched nothing else on the screen.
  it('describes each running-head option the way the other cards do', () => {
    const heads = choicesFor(format(), defaultWizardOptions()).find(c => c.id === 'heads')!
    expect(heads.options.map(o => o.sub)).toEqual([
      'Author on left pages, title on right',
      'Title on left pages, chapter on right',
      'Chapter on both pages',
      'Page numbers only',
    ])
  })

  it('sets what the running head says, and turns it off without forgetting it', () => {
    const options = defaultWizardOptions()
    const titled = chooseTemplate(options, format(), 'heads', 'title-chapter')
    expect(titled.print.headerContent).toBe('title-chapter')
    expect(titled.print.runningHeaders).toBe(true)

    const off = chooseTemplate(titled, format(), 'heads', 'none')
    expect(off.print.runningHeaders).toBe(false)
    expect(off.print.headerContent).toBe('title-chapter')
    expect(choicesFor(format(), off).find(c => c.id === 'heads')?.value).toBe('none')
  })

  // The one card that is a deviation. A custom trim is a page size nobody
  // offers as standard, and its two measurements are set in Advanced.
  it('treats a custom print size as a deviation and everything else as a preference', () => {
    expect(choiceGoesCustom('trim', 'custom')).toBe(true)
    expect(choiceGoesCustom('trim', '6x9')).toBe(false)
    expect(choiceGoesCustom('scene', 'rule')).toBe(false)
    expect(choiceGoesCustom('heads', 'none')).toBe(false)
  })

  // Where the folio sits is a preference, and it is the one that decides
  // whether the running head has to indent inside it.
  it('offers the two folio positions worth a card and keeps the third in Advanced', () => {
    const options = defaultWizardOptions()
    const card = choicesFor(format(), options).find(c => c.id === 'folio')!
    expect(card.options.map(o => o.id)).toEqual(['top-outside', 'bottom-center'])

    const moved = chooseTemplate(options, format(), 'folio', 'bottom-center')
    expect(moved.print.pageNumberPosition).toBe('bottom-center')
    expect(choiceGoesCustom('folio', 'bottom-center')).toBe(false)

    const advanced = advancedFor(format(), options)
      .find(g => g.label === 'Chapters & furniture')!.rows.find(r => r.id === 'folios')!
    expect(advanced.kind === 'select' && advanced.options).toContain('Bottom outside')
  })

  it('saves a preference without moving the format off the standard', () => {
    const patch = preferencePatch(chooseTemplate(defaultWizardOptions(), format(), 'scene', 'rule'))
    expect(patch.typesetting).toBeUndefined()
    expect(patch.export_settings).toBeDefined()
  })

  it('takes a trim and carries its measurements with it', () => {
    const next = chooseTemplate(defaultWizardOptions(), format(), 'trim', '6x9')
    expect(next.print.trimSize).toBe('6x9')
    expect(next.print.customWidth).toBe('6')
    expect(next.print.customHeight).toBe('9')
  })
})

describe('advanced', () => {
  it('keeps the printed specification on the record and the typesetting in the template', () => {
    const groups = advancedFor(format({ page_count: '412' }), defaultWizardOptions())
    const block = groups.find(g => g.label === 'Book block')!
    const paper = block.rows.find(r => r.id === 'paper')!
    expect('record' in paper && paper.record).toBe('paper_stock')

    const margins = groups.find(g => g.label === 'Margins')!
    const gutter = margins.rows.find(r => r.id === 'gutter')!
    expect('field' in gutter && gutter.field).toBe('gutterMargin')
    expect('record' in gutter).toBe(false)
  })

  it('reads the screen’s words back as the values the exporter holds', () => {
    const options = defaultWizardOptions()
    const rows = advancedFor(format(), options).find(g => g.label === 'Type')!.rows
    const size = rows.find(r => r.id === 'size')!
    const align = rows.find(r => r.id === 'align')!
    const leading = rows.find(r => r.id === 'leading')!

    expect(setAdvanced(options, 'print', size, '11 pt').print.fontSize).toBe(11)
    expect(setAdvanced(options, 'print', align, 'Justified').print.textAlign).toBe('justify')
    expect(setAdvanced(options, 'print', leading, 'Open · 1.6').print.lineHeight).toBe(1.6)
  })

  it('asks every format what goes in the file, against its own answers', () => {
    for (const [kind, word] of [['print', 'Paperback'], ['ebook', 'eBook'], ['audio', 'Audiobook']] as const) {
      const groups = advancedFor(format({ kind, format: word }), defaultWizardOptions())
      expect(groups[groups.length - 1].label).toBe('Contents')
    }
  })

  it('tells an ebook with no ISBN what its package will carry instead', () => {
    const groups = advancedFor(format({ kind: 'ebook', format: 'eBook' }), defaultWizardOptions())
    const identifier = groups[0].rows.find(r => r.id === 'identifier')!
    expect(identifier.kind === 'static' && identifier.value).toBe('A generated identifier')
  })
})

describe('the line under a format', () => {
  it('says what that format will actually be', () => {
    expect(kindOf(format())).toBe('print')
    expect(templateSummary(format({ trim: '6 × 9 in' }), defaultWizardOptions()))
      .toContain('6 × 9 in · Merriweather 9 pt')
    expect(templateSummary(format({ kind: 'ebook', format: 'eBook' }), defaultWizardOptions()))
      .toContain('reflowable')
  })
})

// ── The release confirmation ───────────────────────────────────────────────

const snapshot = (patch: Partial<EditionSnapshot> = {}): EditionSnapshot => ({
  id: 'a91f3c7ede', frozen: '2026-04-02T11:42:00Z', word_count: 73873,
  sections: 25, members: 26, bytes: 412000, title: 'Wide Water', ...patch,
} as EditionSnapshot)

const index = (formats: EditionFormat[], snapshots: EditionSnapshot[]): EditionIndex =>
  ({ version: 1, editions: [edition(formats)], snapshots })

describe('releasing a locked manuscript', () => {
  it('takes three steps and only the last one can act', () => {
    expect(RELEASE_STEPS).toBe(3)
    expect(releaseStepLabel(1)).toBe('Yes, continue')
    expect(releaseStepLabel(2)).toBe('I understand, continue')
    expect(releaseStepLabel(3)).toBe('Release snapshot')
  })

  it('accepts only the edition’s own name, and never a blank one', () => {
    const one = edition()
    expect(releaseConfirmed(one, 'First edition')).toBe(true)
    expect(releaseConfirmed(one, '  first EDITION ')).toBe(true)
    expect(releaseConfirmed(one, 'yes')).toBe(false)
    expect(releaseConfirmed(one, '')).toBe(false)
    expect(releaseConfirmed(one, 'First editions')).toBe(false)
    // An edition with no name cannot be confirmed by typing nothing.
    expect(releaseConfirmed({ ...one, label: '' }, '')).toBe(false)
  })

  it('finds the edition’s snapshot through whichever format carries it', () => {
    const formats = [format({ id: 'a' }), format({ id: 'b', snapshot_id: 'a91f3c7ede' })]
    const record = snapshot()
    expect(editionSnapshot(index(formats, [record]), edition(formats))?.id).toBe('a91f3c7ede')
  })

  it('names the formats that stand on those words', () => {
    const formats = [
      format({ id: 'a', format: 'Paperback', snapshot_id: 'x' }),
      format({ id: 'b', format: 'eBook', kind: 'ebook', snapshot_id: 'x' }),
      format({ id: 'c', format: 'Hardcover' }),
    ]
    expect(formatsOnSnapshot(edition(formats), 'x')).toEqual(['Paperback', 'eBook'])
  })

  it('says how far the working draft has moved, in either direction', () => {
    expect(draftDrift(snapshot(), 76013)).toBe('+2,140 words since')
    expect(draftDrift(snapshot(), 73873)).toBe('the same length')
    expect(draftDrift(snapshot(), 70000)).toBe('3,873 words shorter since')
  })

  it('counts the formats standing on the text rather than claiming a tally of exports', () => {
    const formats = [format({ id: 'a', snapshot_id: 'a91f3c7ede' }), format({ id: 'b', snapshot_id: 'a91f3c7ede' })]
    const rows = editionSnapshotRows(index(formats, [snapshot()]), snapshot(), 76013)
    expect(rows.find(r => r.label === 'Formats using it')?.value).toBe('2 formats')
    expect(rows.find(r => r.label === 'Working draft')?.value).toBe('+2,140 words since')
    expect(rows.find(r => r.label === 'Words')?.value).toBe('73,873')
  })
})
