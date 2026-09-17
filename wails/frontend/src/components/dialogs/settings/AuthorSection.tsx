// Author Settings Section - the writer's identity, pre-filled into every new
// book.
//
// There is no copyright template here. A copyright page is written from the
// book's own fields, on Book & Editions, where the year, the holder, the
// edition and the ISBN of the format live.

import type { AuthorSectionProps } from './types'

export default function AuthorSection({
  author, setAuthor,
  publisher, setPublisher,
}: AuthorSectionProps) {
  return (
    <>
      <div className="settings-section-label" style={{ marginTop: 0 }}>Identity</div>
      <div className="dialog-field">
        <label className="dialog-label">Default Author Name</label>
        <input className="dialog-input" value={author} onChange={e => setAuthor(e.target.value)} placeholder="Your name" autoFocus />
        <div className="settings-hint">Pre-filled when creating a new book.</div>
      </div>
      <div className="dialog-field">
        <label className="dialog-label">Default Publisher</label>
        <input className="dialog-input" value={publisher} onChange={e => setPublisher(e.target.value)} placeholder="Publisher or imprint name" />
      </div>

      <div className="settings-section-label">Copyright</div>
      <div className="settings-hint">
        A book&apos;s copyright page is the Copyright Page under Front Pages, and that is what an
        export prints. Book &amp; Editions shows the page a book&apos;s own record would produce — the
        copyright holder, the edition, the publication date and the ISBN of the format being
        published — so you can see what yours should say.
      </div>
    </>
  )
}
