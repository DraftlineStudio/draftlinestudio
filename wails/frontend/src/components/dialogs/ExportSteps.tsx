// The four panes of the export wizard: which formats, their settings, the
// artwork that travels with them, and the last question before a file exists.
//
// Each pane is a function of what the flow has been told so far — it holds no
// state of its own — which is why they live here rather than inside
// ExportWizard: the wizard owns the answers, and this owns how they look.

import type { Edition } from '../../types/draftline'
import type {
  ArtRow, FlowItem, FlowMode, FlowStep, OptionField, ReviewRow, SettingRow,
} from './exportFlow'
import { previewFor, readField, settingGroups } from './exportFlow'
import type { WizardOptions } from './exportSource'
import { FormatGlyph, TickGlyph } from './exportGlyphs'

export interface StepsProps {
  step: FlowStep
  mode: FlowMode
  edition?: Edition
  title: string
  offered: FlowItem[]
  selected: string[]
  onToggle: (id: string) => void
  chosen: FlowItem[]
  tab: string
  onTab: (id: string) => void
  touched: string[]
  optionsFor: (item: FlowItem) => WizardOptions
  onField: (item: FlowItem, field: OptionField, value: string | number | boolean) => void
  looseValue: (item: FlowItem, row: SettingRow) => string | boolean
  onLoose: (item: FlowItem, row: SettingRow, value: string | boolean) => void
  includeArt: boolean
  onIncludeArt: (value: boolean) => void
  art: ArtRow[]
  onChooseArt: (id: string) => void
  artError: string
  lock: boolean | null
  onLock: (value: boolean) => void
  review: ReviewRow[]
  words: number
}

export default function ExportSteps(props: StepsProps) {
  if (props.step === 'formats') return <FormatsPane {...props} />
  if (props.step === 'settings') return <SettingsPane {...props} />
  if (props.step === 'artwork') return <ArtworkPane {...props} />
  if (props.step === 'finalize') return <FinalizePane {...props} />
  return <ReviewPane {...props} />
}

function PaneHead({ title, note }: { title: string; note: string }) {
  return <div className="export-pane-head"><h2>{title}</h2><p>{note}</p></div>
}

// ── Formats ────────────────────────────────────────────────────────────────

function FormatsPane({ mode, edition, offered, selected, onToggle }: StepsProps) {
  const intro = mode === 'edition' && edition
    ? `Registered formats on ${edition.label || 'this edition'}. Each one exports with its own saved template.`
    : 'Pick any mix. Each gets its own settings next.'
  return <>
    <PaneHead title="Which formats?" note={intro} />
    <div className="export-format-list">
      {offered.map(item => {
        const on = selected.includes(item.id)
        return (
          <button type="button" key={item.id} className={`export-format-row${on ? ' selected' : ''}`}
            onClick={() => onToggle(item.id)} aria-pressed={on}>
            <span className={`export-check${on ? ' on' : ''}`} aria-hidden="true">{on && <TickGlyph />}</span>
            <span className="export-format-icon" aria-hidden="true"><FormatGlyph output={item.output} /></span>
            <span className="export-format-text">
              <span className="export-format-title">
                <strong>{item.label}</strong>
                {item.isbn && <em>{item.isbn}</em>}
              </span>
              <small>{item.desc}</small>
            </span>
            <span className="export-format-meta">{item.meta}</span>
            {item.badge && <span className="export-badge neutral">{item.badge}</span>}
          </button>
        )
      })}
    </div>
    <p className="export-pane-foot">Each file is written separately. You’ll pick where to save at the end.</p>
  </>
}

// ── Settings ───────────────────────────────────────────────────────────────

function SettingsPane(props: StepsProps) {
  const { mode, edition, chosen, tab, onTab, touched, optionsFor } = props
  const active = chosen.find(item => item.id === tab) ?? chosen[0]
  if (!active) return <PaneHead title="Check each format’s settings" note="Choose a format first." />
  const options = optionsFor(active)
  const groups = settingGroups(active.output, active.record, options)
  const preview = previewFor(active.output, options)
  const intro = mode === 'edition' && edition
    ? `Prefilled from the templates saved on ${edition.label || 'this edition'}.`
    : 'Draftline defaults. Change anything you like.'

  return <>
    <PaneHead title="Check each format’s settings" note={intro} />
    <div className="export-tabs" role="tablist">
      {chosen.map(item => (
        <button type="button" key={item.id} role="tab" aria-selected={item.id === active.id}
          className={`export-tab${item.id === active.id ? ' active' : ''}`} onClick={() => onTab(item.id)}>
          {item.label}
          {touched.includes(item.id) && <span className="export-tab-dot" aria-label="edited" />}
        </button>
      ))}
    </div>
    <div className="export-settings">
      <div className="export-setting-groups" key={active.id}>
        {groups.map(group => (
          <section className="export-setting-group" key={group.label}>
            <div className="export-group-head">
              <span className="chapter-section-label">{group.label}</span>
              {group.note && <small>{group.note}</small>}
            </div>
            <div className="export-setting-grid">
              {group.rows.map(row => (
                <div className="export-setting" key={row.id}>
                  <span className="export-setting-label">{row.label}</span>
                  <SettingControl row={row} item={active} options={options} {...props} />
                </div>
              ))}
            </div>
          </section>
        ))}
      </div>
      <aside className="export-preview">
        <div className="export-preview-page" style={{ aspectRatio: preview.ratio }} aria-hidden="true">
          <span className="export-preview-kicker">Chapter One</span>
          <i style={{ width: '90%' }} /><i /><i style={{ width: '95%' }} /><i style={{ width: '70%' }} />
          <span className="export-preview-break">⁂</span>
          <i style={{ width: '85%' }} /><i /><i style={{ width: '60%' }} />
        </div>
        <small className="export-preview-caption">{preview.caption}</small>
      </aside>
    </div>
    {mode === 'edition' && (
      <p className="export-template-note">
        These came from the saved template on the edition. Change anything and Draftline will ask
        whether to update the template before continuing.
      </p>
    )}
  </>
}

interface ControlProps extends StepsProps {
  row: SettingRow
  item: FlowItem
  options: WizardOptions
}

function SettingControl({ row, item, options, onField, looseValue, onLoose }: ControlProps) {
  if (row.kind === 'static') return <div className="export-static">{row.value}</div>

  if (row.kind === 'toggle') {
    // An inverted toggle shows the opposite of the field it writes: "Title
    // page" is ticked when omitTitlePage is false.
    const stored = row.field ? readField(options, row.field) === true : looseValue(item, row) === true
    const on = row.invert ? !stored : stored
    return (
      <label className="export-toggle">
        <input type="checkbox" checked={on} onChange={event => {
          const next = row.invert ? !event.target.checked : event.target.checked
          if (row.field) onField(item, row.field, next)
          else onLoose(item, row, next)
        }} />
        <span className="export-switch" aria-hidden="true" />
      </label>
    )
  }

  if (row.kind === 'text') {
    const value = row.field ? String(readField(options, row.field)) : String(looseValue(item, row) ?? '')
    return (
      <input className="export-input" value={value} inputMode="decimal" onChange={event => {
        if (row.field) onField(item, row.field, event.target.value)
        else onLoose(item, row, event.target.value)
      }} />
    )
  }

  // A select stores whatever the option's own type is — a number for a type
  // size, a string for a typeface — so the chosen entry is matched by its
  // printed value rather than by casting the element's string back.
  const current = row.field ? readField(options, row.field) : looseValue(item, row)
  const index = row.choices.findIndex(choice => String(choice.value) === String(current))
  return (
    <select className="export-select" value={index < 0 ? 0 : index} onChange={event => {
      const choice = row.choices[Number(event.target.value)]
      if (!choice) return
      if (row.field) onField(item, row.field, choice.value)
      else onLoose(item, row, String(choice.value))
    }}>
      {row.choices.map((choice, i) => <option key={choice.label} value={i}>{choice.label}</option>)}
    </select>
  )
}

// ── Artwork ────────────────────────────────────────────────────────────────

function ArtworkPane({ includeArt, onIncludeArt, art, onChooseArt, artError }: StepsProps) {
  return <>
    <PaneHead
      title="Include print-ready artwork?"
      note="Cover files are written next to each format. Print formats get the full wrap sized to the spine; ebooks get the storefront cover."
    />
    <div className="export-choice-pair">
      <button type="button" className={`export-choice${includeArt ? ' selected' : ''}`} onClick={() => onIncludeArt(true)} aria-pressed={includeArt}>
        <strong>Yes, include artwork</strong>
        <small>Confirm the file for each format below.</small>
      </button>
      <button type="button" className={`export-choice${includeArt ? '' : ' selected'}`} onClick={() => onIncludeArt(false)} aria-pressed={!includeArt}>
        <strong>Interior only</strong>
        <small>Manuscript files only. Upload covers separately.</small>
      </button>
    </div>
    {includeArt && (
      <div className="export-group">
        <span className="chapter-section-label">Confirm artwork per format</span>
        <div className="export-art-list">
          {art.map(row => (
            <div className="export-art-row" key={row.id}>
              <div className={`export-art-thumb${row.wide ? ' wide' : ''}`}>
                {row.thumbURL
                  ? <img src={row.thumbURL} alt="" />
                  : <span>{row.thumbText}</span>}
              </div>
              <div className="export-art-text">
                <span className="export-art-title">
                  <strong>{row.label}</strong>
                  <span className={`export-badge ${row.badgeKind}`}>{row.badge}</span>
                </span>
                <span className="export-art-file">{row.file}</span>
                <span className="export-art-spec">{row.spec}</span>
              </div>
              <div className="export-art-actions">
                <button type="button" className="dialog-btn sm" onClick={() => onChooseArt(row.id)}>Choose file…</button>
              </div>
            </div>
          ))}
          {!art.length && <p className="export-pane-foot">None of the chosen formats carries artwork.</p>}
        </div>
        {artError && <p className="export-art-error" role="alert">{artError}</p>}
      </div>
    )}
  </>
}

// ── Finalize and review ────────────────────────────────────────────────────

function FinalizePane({ edition, lock, onLock, review, words }: StepsProps) {
  const name = edition?.label || 'this edition'
  return <>
    <PaneHead
      title={`Is this the final text for ${name}?`}
      note="Locking freezes a snapshot of today’s manuscript on the edition. Every future export of this edition uses that snapshot, even after you rewrite for a later edition. You have to choose before exporting."
    />
    <div className="export-lock-pair">
      <button type="button" className={`export-lock${lock === true ? ' selected' : ''}`} onClick={() => onLock(true)} aria-pressed={lock === true}>
        <span className={`export-radio${lock === true ? ' on' : ''}`} aria-hidden="true" />
        <span className="export-lock-text">
          <strong>Yes, lock this text to the edition</strong>
          <small>
            Snapshot of {words.toLocaleString()} words saved inside the project. Later edits go to the
            working draft, not this edition. Unlock any time from Book Info.
          </small>
        </span>
      </button>
      <button type="button" className={`export-lock${lock === false ? ' selected' : ''}`} onClick={() => onLock(false)} aria-pressed={lock === false}>
        <span className={`export-radio${lock === false ? ' on' : ''}`} aria-hidden="true" />
        <span className="export-lock-text">
          <strong>No, export the current draft</strong>
          <small>Files come from the text as it is right now. Nothing is frozen; the next export picks up any changes.</small>
        </span>
      </button>
    </div>
    {lock !== null && (
      <div className="export-group">
        <span className="chapter-section-label">Ready to export</span>
        <ReviewList rows={review} />
      </div>
    )}
  </>
}

function ReviewPane({ review }: StepsProps) {
  return <>
    <PaneHead title="Ready to export" note="Files come from the current draft. Nothing is locked or saved to an edition." />
    <ReviewList rows={review} />
  </>
}

function ReviewList({ rows }: { rows: ReviewRow[] }) {
  return (
    <div className="export-review-list">
      {rows.map(row => (
        <div className="export-review-row" key={row.k}>
          <span className="export-review-k">{row.k}</span>
          <span className="export-review-v">{row.v}</span>
        </div>
      ))}
    </div>
  )
}
