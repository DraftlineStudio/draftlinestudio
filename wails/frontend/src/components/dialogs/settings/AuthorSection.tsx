// Author Settings Section - the writer's identity and copyright template,
// pre-filled into every new book.

import type { AuthorSectionProps } from './types'

export default function AuthorSection({
  author, setAuthor,
  publisher, setPublisher,
  copyright, setCopyright,
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

      <div className="settings-section-label">Copyright Template</div>
      <div className="dialog-field">
        <label className="dialog-label">Default Copyright Text</label>
        <textarea
          className="dialog-input settings-copyright-textarea"
          value={copyright}
          onChange={e => setCopyright(e.target.value)}
          placeholder={"Copyright © [YEAR] [AUTHOR]. All rights reserved.\n\nNo part of this publication may be reproduced..."}
          rows={5}
        />
        <div className="settings-hint">Inserted into the Copyright page of every new book. Use [YEAR] and [AUTHOR] as placeholders.</div>
      </div>
    </>
  )
}
