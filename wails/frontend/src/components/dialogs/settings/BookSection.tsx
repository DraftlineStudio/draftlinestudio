// Book Defaults Settings Section - Editor Display, Export Settings

import type { BookSectionProps } from './types'
import { BOOK_FONTS, TRIM_SIZES } from './constants'

export default function BookSection({
  bookFont, setBookFont,
  editorFontSize, setEditorFontSize,
  bookFontSize, setBookFontSize,
  bookLineSpacing, setBookLineSpacing,
  bookDropCaps, setBookDropCaps,
  bookTrimSize, setBookTrimSize,
}: BookSectionProps) {
  return (
    <>
      <div className="settings-section-label" style={{ marginTop: 0 }}>Editor Display</div>
      <div className="dialog-field">
        <label className="dialog-label">Font</label>
        <select className="dialog-select" value={bookFont} onChange={e => setBookFont(e.target.value)}>
          {BOOK_FONTS.map(f => <option key={f} value={f}>{f}</option>)}
        </select>
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Text Size</label>
        <div className="settings-theme-row">
          <button className={`settings-theme-btn${editorFontSize === 'small' ? ' active' : ''}`} onClick={() => setEditorFontSize('small')}>
            Small (12pt)
          </button>
          <button className={`settings-theme-btn${editorFontSize === 'normal' ? ' active' : ''}`} onClick={() => setEditorFontSize('normal')}>
            Normal (14pt)
          </button>
          <button className={`settings-theme-btn${editorFontSize === 'large' ? ' active' : ''}`} onClick={() => setEditorFontSize('large')}>
            Large (16pt)
          </button>
        </div>
        <div className="settings-hint">Controls how text appears in the editor. Does not affect exported files.</div>
      </div>

      <div className="settings-section-label">Export Settings</div>
      <div className="settings-two-col">
        <div className="dialog-field">
          <label className="dialog-label">Font Size (pt)</label>
          <input className="dialog-input" type="number" min={9} max={16} step={0.5}
            value={bookFontSize} onChange={e => setBookFontSize(Number(e.target.value))} />
        </div>
        <div className="dialog-field">
          <label className="dialog-label">Line Spacing</label>
          <select className="dialog-select" value={bookLineSpacing} onChange={e => setBookLineSpacing(e.target.value)}>
            <option value="1.0">Single (1.0)</option>
            <option value="1.25">Comfortable (1.25)</option>
            <option value="1.5">Relaxed (1.5)</option>
            <option value="2.0">Double (2.0)</option>
          </select>
        </div>
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Trim Size</label>
        <select className="dialog-select" value={bookTrimSize} onChange={e => setBookTrimSize(e.target.value)}>
          {TRIM_SIZES.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}
        </select>
        <div className="settings-hint">Used when exporting to PDF for print-ready typesetting.</div>
      </div>
      <div className="dialog-field">
        <label className="dialog-label" style={{ marginBottom: 8 }}>Drop Caps</label>
        <label className="settings-toggle">
          <input type="checkbox" checked={bookDropCaps} onChange={e => setBookDropCaps(e.target.checked)} />
          <span className="settings-toggle-track"><span className="settings-toggle-thumb" /></span>
          <span className="settings-toggle-label">{bookDropCaps ? 'Enabled — first letter of each chapter is enlarged' : 'Disabled'}</span>
        </label>
      </div>
    </>
  )
}
