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
import type { CoverSourceReport, Edition, Metadata } from '../../types/draftline'
import {
  AttachCover, AttachCoverDialog, CheckCoverSource, RemoveCover, RevealInFileManager,
} from '../../../wailsjs/go/main/App'
import { droppedOn, subscribeFileDrop } from '../../services/fileDrop'
import {
  EDITION_COVER_NOTE, LARGE_COPY_HINT, coverFacts, coverNotices, coverPixels,
  coverThumbURL, firstArtworkPath, sourceStatusTone,
} from './coverModel'

interface Props {
  edition: Edition
  meta: Partial<Metadata>
  /**
   * Removing or replacing a cover goes through the same confirmation the wrap
   * uses: it can be the moment an author loses a file they cannot make again,
   * and the project may hold the only copy. The shell owns that dialog because
   * it owns the one that asks about the wrap.
   */
  onAsk: (action: 'remove' | 'replace', replace: () => void) => void
  /** Writes the stored cover back out to a file the author chooses. */
  onSaveCopy: () => void
}

export default function CoverCard({ edition, meta, onAsk, onSaveCopy }: Props) {
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

  // A drop anywhere on the card is the same act as the button.
  //
  // Claimed only when the drop landed on this card, so a project dropped
  // elsewhere still reaches the handler that opens it. This used to lean on
  // Wails' own drop-target filtering, which meant registering and removing
  // the one global handler as this component came and went -- and while it
  // was gone nothing intercepted a drop at all, so WebView2 downloaded the
  // file instead. See services/fileDrop.
  useEffect(() => subscribeFileDrop((x, y, paths) => {
    if (!droppedOn(x, y, '.bi-cover-drop')) return false
    const path = firstArtworkPath(paths)
    if (!path) {
      setError('Draftline reads cover artwork as JPEG, PNG, TIFF, WebP or BMP.')
      return true
    }
    void attach(path)
    return true
  }, 10), [attach])

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

  // The act itself, run only after the confirmation has been answered and any
  // backup has actually been written.
  const remove = async () => {
    await RemoveCover(editionID)
    updateEdition(editionID, { cover: undefined, cover_id: '' })
    setSource(null)
  }

  const thumb = cover ? coverThumbURL(edition) : ''

  // Showing the print-ready original where it lives. It is the author's own
  // file on their own disk — the project only remembers where — so the honest
  // answer to "where is it?" is the one the file browser already gives.
  const reveal = async () => {
    if (!sourcePath) return
    const result = await RevealInFileManager(sourcePath)
    if (!result.success) setError(result.error || 'That file could not be shown.')
    else if (result.note) setError(result.note)
  }

  return (
    <section className="bi-cover-card bi-cover-drop">
      <div className="bi-card-head">
        <span className="chapter-section-label">Cover</span>
        <small>This edition only</small>
      </div>

      {/* Everything is UNDER the picture. This card lives in a 300px column
          and a two-column layout put the facts off the side of the screen. */}
      {thumb ? (
        <img
          className="bi-cover-image" src={thumb}
          alt={`Cover artwork for ${edition.label || 'this edition'}`}
        />
      ) : (
        <div className="bi-ed-cover-plate">
          <span className="bi-ed-cover-kicker">{edition.label || 'Edition'}</span>
          <span className="bi-ed-cover-title">{meta.title || 'Untitled'}</span>
          <span className="bi-ed-cover-author">{meta.author || ''}</span>
        </div>
      )}

      {/* Under the picture: its size, and nothing else. */}
      <span className="bi-cover-pixels">
        {cover ? coverPixels(cover) : 'No cover attached'}
      </span>

      {cover && (
        <div className="bi-cover-facts">
          {coverFacts(cover).map(fact => (
            <div className="bi-cover-fact" key={fact.label}>
              <span>{fact.label}</span><em>{fact.value}</em>
            </div>
          ))}
          {sourcePath && (
            <div className="bi-fact-path">
              <span>Original file</span>
              <em title={sourcePath}>{sourcePath}</em>
            </div>
          )}
          {sourcePath && (
            <button type="button" className="dialog-btn sm bi-cover-reveal" onClick={() => void reveal()}>
              Show original in folder
            </button>
          )}
        </div>
      )}

      <div className="bi-cover-actions">
        <button
          className="dialog-btn sm" disabled={busy}
          onClick={() => (cover ? onAsk('replace', () => void attach('')) : void attach(''))}
        >
          {busy ? <><span className="bi-spinner" aria-hidden="true" />Working…</> : cover ? 'Replace…' : 'Attach cover…'}
        </button>
        {cover && (
          <button className="dialog-btn sm" disabled={busy} onClick={() => onAsk('remove', () => void remove())}>
            Remove
          </button>
        )}
        {cover && (
          <button className="dialog-btn sm" disabled={busy} onClick={onSaveCopy}>Save a copy…</button>
        )}
      </div>

      <label className="bi-cover-option" title={LARGE_COPY_HINT}>
        <input type="checkbox" checked={large} disabled={busy} onChange={e => setLarge(e.target.checked)} />
        <span>Keep a 2400 × 3840 copy</span>
      </label>

      {error && <p className="bi-cover-error">{error}</p>}

      {/* One line, and only when there is something to say about THIS
          artwork: what the conversion changed, or that the print-ready
          original has moved. */}
      {cover && coverNotices(cover).slice(0, 1).map(note => (
        <p className="bi-cover-notice" key={note}>{note}</p>
      ))}
      {source && source.status !== 'present' && (
        <p className={`bi-cover-source ${sourceStatusTone(source.status)}`}>{source.message}</p>
      )}

      <p className="bi-cover-note">{EDITION_COVER_NOTE}</p>
    </section>
  )
}
