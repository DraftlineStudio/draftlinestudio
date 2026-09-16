// The cover on one edition: attach it, look at it, and be told what it is not.
//
// Artwork enters this component as a PATH and never as bytes. The button opens
// the native file dialog on the Go side, which returns a path; dropping a file
// on the card gives Wails the path too (EnableFileDrop in main.go). There is
// deliberately no <input type="file"> here: it would pull the whole image into
// the webview and then base64 it back across the bridge to reach the same
// place the path reaches directly.
//
// The picture itself is an ordinary <img> pointing at a URL this process
// serves, for the same reason.

import { useCallback, useEffect, useState } from 'react'
import { useBookStore } from '../../store/bookStore'
import type { CoverSourceReport, Edition, EditionFormat, Metadata } from '../../types/draftline'
import { AttachCover, AttachCoverDialog, CheckCoverSource, RemoveCover } from '../../../wailsjs/go/main/App'
import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime'
import {
  EDITION_COVER_CAVEAT, PRINT_COVER_CAVEAT, attachedLabel, coverFacts, coverNotices,
  coverThumbURL, firstArtworkPath, sourceStatusTone,
} from './coverModel'

interface Props {
  edition: Edition
  format: EditionFormat
  meta: Partial<Metadata>
}

export default function CoverCard({ edition, format, meta }: Props) {
  const updateEdition = useBookStore(s => s.updateEdition)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [large, setLarge] = useState(false)
  const [source, setSource] = useState<CoverSourceReport | null>(null)

  const cover = edition.cover
  const editionID = edition.id
  const sourcePath = cover?.source_path ?? ''
  const sourceChecksum = cover?.source_checksum ?? ''

  const attach = useCallback(async (path: string) => {
    setBusy(true)
    setError('')
    try {
      const result = path
        ? await AttachCover(editionID, path, large)
        : await AttachCoverDialog(editionID, large)
      if (result.cancelled) return
      if (!result.success || !result.cover) {
        setError(result.error || 'That cover could not be read.')
        return
      }
      updateEdition(editionID, { cover: result.cover, cover_id: result.cover.id })
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }, [editionID, large, updateEdition])

  // A drop anywhere on the card is the same act as the button. useDropTarget
  // is on, so this only fires over an element carrying the drop-target custom
  // property — see the COVER ART section of global.css.
  useEffect(() => {
    OnFileDrop((_x, _y, paths) => {
      const path = firstArtworkPath(paths ?? [])
      if (!path) {
        setError('Draftline reads cover artwork as JPEG, PNG, TIFF, WebP or BMP.')
        return
      }
      void attach(path)
    }, true)
    return () => OnFileDropOff()
  }, [attach])

  // Is the print-ready original still where it was? Asked whenever the record
  // changes, because the answer is about the disk and not about the record.
  useEffect(() => {
    let live = true
    if (!sourcePath) {
      setSource(null)
      return
    }
    CheckCoverSource(sourcePath, sourceChecksum)
      .then(report => { if (live) setSource(report) })
      .catch(() => { if (live) setSource(null) })
    return () => { live = false }
  }, [sourcePath, sourceChecksum])

  const remove = async () => {
    await RemoveCover(editionID)
    updateEdition(editionID, { cover: undefined, cover_id: '' })
    setSource(null)
  }

  const thumb = cover ? coverThumbURL(edition) : ''

  return (
    <section className="bi-ed-card bi-cover-drop">
      <div className="bi-ed-cover">
        {thumb ? (
          <img
            className="bi-cover-image" src={thumb}
            width={cover?.thumb_width} height={cover?.thumb_height}
            alt={`Cover artwork for ${edition.label || 'this edition'}`}
          />
        ) : (
          <div className="bi-ed-cover-plate">
            <span className="bi-ed-cover-kicker">{edition.label || 'Edition'}</span>
            <span className="bi-ed-cover-title">{meta.title || 'Untitled'}</span>
            <span className="bi-ed-cover-author">{meta.author || ''}</span>
          </div>
        )}
        <span className="bi-ed-cover-caption">
          {cover
            ? `${edition.label || 'Edition'} art${attachedLabel(cover) ? ` · ${attachedLabel(cover)}` : ''}`
            : 'No cover attached'}
        </span>
      </div>

      <div className="bi-ed-card-body">
        <div className="bi-section-head">
          <span className="chapter-section-label">Cover art — this edition only</span>
          <span className="bi-section-note">Drop artwork here, or choose a file.</span>
        </div>

        {cover ? (
          <div className="bi-ed-facts">
            {coverFacts(cover).map(fact => (
              <div className="bi-ed-fact" key={fact.label}>
                <span>{fact.label}</span><span>{fact.value}</span>
              </div>
            ))}
          </div>
        ) : (
          <p className="bi-ed-note">
            Nothing attached. Draftline makes the ebook cover from whatever you give it: a JPEG, a
            PNG, a TIFF, at any size from 625 × 1000 upwards. It is resized once, in linear light,
            and encoded at the quality this particular picture needs rather than at a fixed one.
          </p>
        )}

        <div className="bi-cover-actions">
          <button className="dialog-btn sm" disabled={busy} onClick={() => void attach('')}>
            {busy ? 'Working…' : cover ? 'Replace cover…' : 'Attach cover…'}
          </button>
          {cover && (
            <button className="dialog-btn sm" disabled={busy} onClick={() => void remove()}>
              Remove cover
            </button>
          )}
          <label className="bi-cover-option">
            <input type="checkbox" checked={large} onChange={e => setLarge(e.target.checked)} />
            <span>Also keep a 2400 × 3840 copy</span>
          </label>
        </div>
        <div className="bi-field-hint bi-cover-hint">
          The larger copy is for Kobo, which asks for 2400 on the short edge. It is off by default
          because Amazon charges the author a delivery fee per megabyte on every sale, so a bigger
          cover costs a little on every copy sold for as long as the book is on sale.
        </div>

        {error && <p className="bi-cover-error">{error}</p>}

        {cover && coverNotices(cover).map(note => (
          <p className="bi-cover-notice" key={note}>{note}</p>
        ))}

        {source && (
          <p className={`bi-cover-source ${sourceStatusTone(source.status)}`}>
            <strong>Print-ready original:</strong> {source.message}
          </p>
        )}

        <p className="bi-ed-note">{PRINT_COVER_CAVEAT}</p>
        <p className="bi-ed-note">{EDITION_COVER_CAVEAT}</p>

        <div className="bi-ed-facts">
          <div className="bi-ed-fact">
            <span>Text frozen at export</span><span>{format.snapshot_id ? format.snapshot_id : 'no snapshot'}</span>
          </div>
          <div className="bi-ed-fact">
            <span>Exports from</span><span>{format.snapshot_id ? 'the frozen text' : 'the current draft'}</span>
          </div>
        </div>
      </div>
    </section>
  )
}
