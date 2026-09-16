// One registered format: what it is, what artwork it ships with, and the
// template an export of it starts from.
//
// The template is the export wizard's own settings, stored on this record.
// Editing it here is the same act as editing it in the wizard and saving it
// back — there is one template per format and one place it is written, which
// is what stops the two screens disagreeing about how a book is set.

import type { Edition, EditionFormat, Metadata } from '../../types/draftline'
import { coverFormat, coverThumbURL } from './coverModel'
import { formatBadge, isbnLocked, kindDot } from './editionModel'
import {
  STANDARD_HINT, advancedFor, choiceGoesCustom, choicesFor, chooseTemplate, isSpecification,
  kindOf, setAdvanced, typesettingOf, type AdvancedRow,
} from './formatTemplate'
import type { WizardOptions } from './exportSource'
import { RevealInFileManager } from '../../../wailsjs/go/main/App'
import { fileSizeLabel, wrapPreviewURL, wrapSizeLabel } from './wrapModel'

interface Props {
  meta: Partial<Metadata>
  edition: Edition
  format: EditionFormat
  options: WizardOptions
  advancedOpen: boolean
  onAdvanced: (open: boolean) => void
  onFormat: (patch: Partial<EditionFormat>) => void
  /** A card above the Advanced line: configures the book, stays standard. */
  onPreference: (options: WizardOptions) => void
  /** An Advanced edit: this is what going Custom means. */
  onTemplate: (options: WizardOptions) => void
  /** Back to the standard: the whole template as it ships. */
  onStandard: () => void
  /** To custom: it changes no setting, it unlocks them. */
  onCustom: () => void
  onRemove: () => void
  onExport: () => void
  onAddWrap: () => void
  onReplaceWrap: () => void
  onRemoveWrap: () => void
  onSaveWrapCopy: () => void
  onKeepWrap: (keep: boolean) => void
  /** True while a picker is open or a copy is being made for this format. */
  wrapBusy: boolean
  wrapError: string
}

const REGISTRATIONS = ['Not yet assigned', 'Registered — Bowker', 'Registered — Nielsen', 'Free retailer ISBN']

export default function FormatPane(props: Props) {
  const { meta, edition, format, options } = props
  const kind = kindOf(format)
  const badge = formatBadge(format)
  const locked = isbnLocked(format)
  const isbn = (format.isbn13 ?? '').trim()
  const choices = choicesFor(format, options)
  const typesetting = typesettingOf(format)
  const custom = typesetting === 'custom'
  const thumb = coverThumbURL(edition)
  const cover = edition.cover
  const wrap = format.wrap
  const isPrint = kind === 'print'
  const preview = wrap ? wrapPreviewURL(edition.id, wrap) : ''

  const subtitle = [
    edition.year ? `© ${edition.year}` : '',
    kind === 'print' ? (format.trim ?? '').trim() : '',
    isbn,
  ].filter(Boolean).join(' · ')

  // The cards above the Advanced line are preferences. Choosing a trim, a
  // scene-break mark or what the running head says configures the book and
  // leaves it on the industry standard; only Advanced moves it off.
  function choose(choiceID: string, value: string) {
    const next = chooseTemplate(options, format, choiceID, value)
    if (choiceGoesCustom(choiceID, value)) {
      // A custom trim is finished in Advanced — it is two measurements, not a
      // choice — so it moves the format to Custom and opens the drawer.
      props.onTemplate(next)
      props.onAdvanced(true)
      return
    }
    props.onPreference(next)
  }

  // The print-ready original lives on the author's disk, not in the project.
  // The useful thing to do with a recorded path is open it where it is.
  async function reveal(path: string) {
    if (!path) return
    await RevealInFileManager(path)
  }

  function applyAdvanced(row: AdvancedRow, value: string | boolean) {
    if ('record' in row && row.record) {
      props.onFormat({ [row.record]: value } as Partial<EditionFormat>)
      return
    }
    props.onTemplate(setAdvanced(options, kind, row, value))
  }

  return <>
    <header className="bi-pane-head">
      <div className="bi-pane-head-text">
        <div className="bi-pane-title">
          <span className="bi-ed-dot" style={{ background: kindDot(format.kind) }} />
          <strong>{(format.format ?? '').trim() || 'Format'}</strong>
          <em>— {edition.label || 'Edition'}</em>
          <span className={`bi-badge ${badge.kind}`}>{badge.label}</span>
        </div>
        <span className="bi-pane-sub">{subtitle}</span>
      </div>
      <div className="bi-pane-actions">
        <button type="button" className="dialog-btn sm" onClick={props.onRemove}>Remove format</button>
        <button type="button" className="dialog-btn sm primary" onClick={props.onExport}>Export this format…</button>
      </div>
    </header>

    {/* ── Artwork ──
        On a printed format the wrap IS the artwork: it takes the place the
        edition's cover used to occupy here, because the cover is set on the
        edition and repeating it told nobody anything about the file that
        actually goes to the printer. An ebook and a script still show the
        edition's cover, which is theirs. */}
    <section className="bi-card bi-art">
      <div className="bi-art-cover">
        {isPrint
          ? (wrap
            ? (preview
              ? <img
                className="bi-wrap-preview" src={preview} alt=""
                width={wrap.preview_width} height={wrap.preview_height}
              />
              : <span className="bi-wrap-noimage">{(wrap.file_name.split('.').pop() || '').toUpperCase()}</span>)
            : <button type="button" className="bi-wrap-empty" disabled={props.wrapBusy} onClick={props.onAddWrap}>
              {props.wrapBusy
                ? <><span className="bi-spinner big" aria-hidden="true" /><span>Reading the artwork…</span></>
                : <>
                  <span aria-hidden="true">+</span>
                  <span>Add full wraparound…</span>
                  <small>{format.page_count ? 'sized from trim and page count' : 'set a page count for the spine'}</small>
                </>}
            </button>)
          : <>
            <div className="bi-cover-plate">
              {thumb
                ? <img src={thumb} alt={`Cover of ${edition.label || 'this edition'}`} />
                : <span className="bi-cover-drawn"><em>{meta.title || 'Untitled'}</em><i>{meta.author || ''}</i></span>}
            </div>
            <small>{edition.label || 'Edition'} cover</small>
          </>}
      </div>

      <div className="bi-art-body">
        <div className="bi-card-head">
          <span className="chapter-section-label">Artwork</span>
          <small>{
            isPrint ? 'Back · spine · front in one file for the printer.'
              : kind === 'audio' ? 'The edition’s cover, placed on the script’s opening page.'
                : 'The edition’s cover. Change it on the edition.'
          }</small>
        </div>

        <div className="bi-facts">
          {isPrint && wrap ? <>
            <div className="bi-fact"><span>Wrap</span><em>{wrap.file_name}</em></div>
            <div className="bi-fact">
              <span>Size</span>
              <em>{[wrapSizeLabel(wrap), fileSizeLabel(wrap.bytes ?? 0)].filter(Boolean).join(' · ') || '—'}</em>
            </div>

          </> : isPrint ? (
            <div className="bi-fact"><span>Wrap</span><em>None attached</em></div>
          ) : <>
            <div className="bi-fact"><span>Format</span><em>{cover ? coverFormat(cover) : 'No cover attached'}</em></div>
            <div className="bi-fact">
              <span>Pixels</span>
              <em>{cover ? `${cover.width} × ${cover.height}` : '—'}</em>
            </div>
          </>}
        </div>

        {isPrint && wrap?.source_path && (
          <div className="bi-fact-path">
            <span>Original file</span>
            <em title={wrap.source_path}>{wrap.source_path}</em>
          </div>
        )}

        {isPrint && wrap && <>
          {/* The toggle lives with the artwork it is about, not in a column
              beside it. */}
          <label className={`bi-toggle-row${props.wrapBusy ? ' off' : ''}`}>
            <span className="bi-toggle">
              <input
                type="checkbox" checked={!!wrap.stored} disabled={props.wrapBusy}
                onChange={event => props.onKeepWrap(event.target.checked)}
              />
              <span className="bi-switch" aria-hidden="true" />
            </span>
            <span>Keep a copy inside this Draftline file</span>
          </label>

          <div className="bi-wrap-actions">
            <button type="button" className="dialog-btn sm" disabled={props.wrapBusy} onClick={props.onReplaceWrap}>
              {props.wrapBusy ? <><span className="bi-spinner" aria-hidden="true" />Working…</> : 'Replace…'}
            </button>
            <button type="button" className="dialog-btn sm" disabled={props.wrapBusy} onClick={props.onRemoveWrap}>Remove</button>
            {/* A stored copy can be written back out; a linked one is already
                a file, so the useful thing is to be shown where it is. */}
            {wrap.stored
              ? <button type="button" className="dialog-btn sm" disabled={props.wrapBusy} onClick={props.onSaveWrapCopy}>Save a copy…</button>
              : wrap.source_path && (
                <button type="button" className="dialog-btn sm" onClick={() => void reveal(wrap.source_path ?? '')}>
                  Show in folder
                </button>
              )}
          </div>
        </>}

        {props.wrapError && <p className="bi-error" role="alert">{props.wrapError}</p>}
      </div>
    </section>

    {/* ── Identity ── */}
    <section className="bi-card">
      <div className="bi-card-head">
        <span className="chapter-section-label">Identity</span>
        <small>{locked ? 'This format is published, so its ISBN is fixed.' : 'Fixed once the ISBN is registered.'}</small>
      </div>
      <div className="bi-grid">
        <div className="bi-field">
          <label className="bi-field-label">ISBN-13</label>
          <input
            className="dialog-input mono" value={format.isbn13 ?? ''} placeholder="978-…"
            readOnly={locked} disabled={locked}
            onChange={event => props.onFormat({ isbn13: event.target.value })}
          />
          <div className="bi-field-hint">Optional. A format without one exports perfectly well.</div>
        </div>
        <div className="bi-field">
          <label className="bi-field-label">Registration</label>
          <select
            className="dialog-select" value={format.registration ?? REGISTRATIONS[0]}
            onChange={event => props.onFormat({ registration: event.target.value })}
          >
            {REGISTRATIONS.map(one => <option key={one} value={one}>{one}</option>)}
          </select>
        </div>
        <div className="bi-field">
          <label className="bi-field-label">Imprint of record</label>
          <input
            className="dialog-input" value={format.imprint_of_record ?? ''}
            placeholder={meta.imprint || meta.publisher || ''}
            onChange={event => props.onFormat({ imprint_of_record: event.target.value })}
          />
        </div>
      </div>
    </section>

    {/* ── Typeset settings ── */}
    <section className="bi-card bi-typeset">
      <div className="bi-card-head">
        <span className="chapter-section-label">Typesetting options</span>
      </div>

      {/* One control, not two. The old screen had a badge that was worked out
          from the settings and a card that set them, and they disagreed:
          applying the standard read back as Custom, and an ebook could never
          be moved off it at all. This is the single choice, stored on the
          record. */}
      <div className="bi-choice-block">
        <div className="bi-choice-row">
          <button
            type="button" className={`bi-choice${custom ? '' : ' selected'}`}
            onClick={props.onStandard} aria-pressed={!custom}
          >
            <span className="bi-choice-label">Industry standard</span>
            <small>Everything below set the usual way</small>
          </button>
          <button
            type="button" className={`bi-choice${custom ? ' selected' : ''}`}
            onClick={props.onCustom} aria-pressed={custom}
          >
            <span className="bi-choice-label">Custom</span>
            <small>Unlock Advanced and set your own</small>
          </button>
        </div>
        <small className="bi-choice-note">
          {custom
            ? 'Your own settings. Choosing Industry standard puts every one of them back, and keeps your print size.'
            : STANDARD_HINT[kind]}
        </small>
      </div>

      {choices.map(choice => (
        <div className="bi-choice-block" key={choice.id}>
          <div className="bi-choice-head">
            <strong>{choice.label}</strong>
            {choice.hint && <small>{choice.hint}</small>}
            {choice.held && <small className="bi-held">Recorded here; no exporter reads it yet.</small>}
          </div>
          <div className="bi-choice-row">
            {choice.options.map(option => (
              <button
                type="button" key={option.id}
                className={`bi-choice${option.id === choice.value ? ' selected' : ''}`}
                onClick={() => choose(choice.id, option.id)}
                aria-pressed={option.id === choice.value}
              >
                <span className="bi-choice-label">
                  {option.glyph && <i>{option.glyph}</i>}
                  {option.label}
                </span>
                {option.sub && <small>{option.sub}</small>}
              </button>
            ))}
          </div>
        </div>
      ))}

      <div className="bi-advanced">
        <button
          type="button" className="bi-disclosure"
          onClick={() => props.onAdvanced(!props.advancedOpen)} aria-expanded={props.advancedOpen}
        >
          <span className={`bi-caret${props.advancedOpen ? ' open' : ''}`} aria-hidden="true">›</span>
          <strong>Advanced</strong>
          {!props.advancedOpen && <small>{advancedHint(kind)}</small>}
        </button>

        {props.advancedOpen && <div className="bi-advanced-body">
          {advancedFor(format, options).map(group => (
            <div className="bi-advanced-group" key={group.label}>
              <span className="bi-advanced-label">{group.label}</span>
              <div className="bi-grid">
                {group.rows.map(row => (
                  <div className="bi-field" key={row.id}>
                    <label className="bi-field-label">{row.label}</label>
                    <AdvancedField
                      row={row} locked={!custom && !isSpecification(row)}
                      onChange={value => applyAdvanced(row, value)}
                    />
                    {row.hint && <div className="bi-field-hint">{row.hint}</div>}
                  </div>
                ))}
              </div>
            </div>
          ))}
          <small className="bi-advanced-foot">
            {custom
              ? 'An export of this format starts from these and asks before saving changes back.'
              : 'These are the industry standard. Choose Custom above to unlock them; the page count and the book block stay editable either way, because those describe the object rather than how it is set.'}
          </small>
        </div>}
      </div>
    </section>

    <p className="bi-pane-foot">
      The copyright page and release details are written from the shared book details and this
      edition. Locking the manuscript text happens when you export.
    </p>
  </>
}

function AdvancedField({ row, locked, onChange }: {
  row: AdvancedRow
  locked: boolean
  onChange: (value: string | boolean) => void
}) {
  if (row.kind === 'static') return <div className="bi-static mono">{row.value}</div>
  if (row.kind === 'toggle') {
    return (
      <label className={`bi-toggle${locked ? ' locked' : ''}`}>
        <input
          type="checkbox" checked={row.value} disabled={locked}
          onChange={event => onChange(event.target.checked)}
        />
        <span className="bi-switch" aria-hidden="true" />
      </label>
    )
  }
  if (row.kind === 'text') {
    return (
      <input
        className={`dialog-input${row.mono ? ' mono' : ''}`} value={row.value}
        readOnly={locked} disabled={locked}
        onChange={event => onChange(event.target.value)}
      />
    )
  }
  // A record written by another build may hold a word this list does not
  // offer. It is kept rather than silently rewritten.
  const options = row.value && !row.options.includes(row.value) ? [row.value, ...row.options] : row.options
  return (
    <select
      className="dialog-select" value={row.value} disabled={locked}
      onChange={event => onChange(event.target.value)}
    >
      {!row.value && <option value="">Not set</option>}
      {options.map(one => <option key={one} value={one}>{one}</option>)}
    </select>
  )
}

function advancedHint(kind: string): string {
  if (kind === 'print') return 'Book block, margins, type, chapter furniture, contents'
  if (kind === 'ebook') return 'Package, reader defaults, contents'
  return 'Script type, contents'
}
