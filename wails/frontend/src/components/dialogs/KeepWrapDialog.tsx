// Keeping a copy of the print-ready wrap inside the project.
//
// It is one checkbox and it is not a small decision, so it asks — with the
// real numbers rather than a vague caution. A wrap is tens of megabytes; the
// project file is rewritten whole on every save, five seconds after a
// keystroke, and a sync client re-uploads it each time. Linking instead costs
// nothing and needs the original to still be where it was at export.

import type { EditionWrap } from '../../types/wrap'
import { fileSizeLabel } from './wrapModel'

interface Props {
  wrap: EditionWrap
  /** The project's name and current size, so the growth is a real number. */
  projectName: string
  projectBytes: number
  onKeep: () => void
  onLink: () => void
}

export default function KeepWrapDialog({ wrap, projectName, projectBytes, onKeep, onLink }: Props) {
  const added = wrap.bytes ?? 0
  const grown = projectBytes > 0 ? fileSizeLabel(projectBytes + added) : ''
  return (
    <div className="bi-modal-scrim">
      <div className="bi-modal">
        <header className="bi-modal-head"><span>Keep a copy inside the Draftline file?</span></header>
        <div className="bi-modal-body">
          <p>
            The wrap artwork is <em className="mono">{fileSizeLabel(added)}</em>. Keeping a copy grows{' '}
            <em>{projectName}</em>
            {projectBytes > 0 ? <> from <em className="mono">{fileSizeLabel(projectBytes)}</em> to about <em className="mono">{grown}</em></> : null}
            , and every save and backup carries it.
          </p>
          <p>
            Linking instead keeps the file small; an export then needs the original at{' '}
            <em className="mono">{wrap.source_path || 'its own location'}</em> to still be there.
          </p>
        </div>
        <footer className="bi-modal-foot">
          <button type="button" className="dialog-btn" onClick={onLink}>Link only</button>
          <button type="button" className="dialog-btn primary" onClick={onKeep}>
            Keep a copy (+{fileSizeLabel(added)})
          </button>
        </footer>
      </div>
    </div>
  )
}
