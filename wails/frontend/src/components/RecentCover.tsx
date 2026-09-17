// A recent project's own cover on the start screen.
//
// Shows the artwork of the newest edition that has any, and the book's colour
// when it has none. Most books have none for most of their life, so the colour
// is not a fallback state to be apologised for — it is the ordinary one, and it
// stays underneath the art either way so the spine shadow and the glow read the
// same whichever is showing.
//
// The image arrives over an ordinary same-origin URL served by this process
// (see recentcover.go), addressed by a key that process gave us. Nothing is
// base64-encoded and no path crosses the bridge.
//
// The title lettering is hidden once art loads, because a cover already has
// its title on it and printing ours over the top looks like a mistake.

import { useState } from 'react'

interface Props {
  /** The key the backend gave this project. Not a path, and stable while
   *  the path is -- addressing by list position meant a book wore the art of
   *  whichever one took its place when the list reordered. */
  coverKey: string
  name: string
  /** The book's colour: the ground under the art, and its stand-in without. */
  accent: string
  /** 'hero' is the large book on the left; 'mini' is a row in the list. */
  size: 'hero' | 'mini'
}

export default function RecentCover({ coverKey, name, accent, size }: Props) {
  // 'idle' until the image resolves, so nothing flickers on a book with no art:
  // the letter or title simply stays put and the img never becomes visible.
  const [art, setArt] = useState<'idle' | 'shown' | 'none'>('idle')

  const className = size === 'hero' ? 'sgw-cover' : 'sgw-mini'
  const label = size === 'hero' ? name : name.charAt(0).toUpperCase()

  return (
    <span className={`${className} rc-wrap`} style={{ background: accent }}>
      {art === 'shown' ? null : <span>{label}</span>}
      {/* No key, nothing to ask for. A recents file written before covers
          were shown has none, and /recent-cover/ would only 404. */}
      {coverKey ? (
        <img
          className="rc-art"
          src={`/recent-cover/${coverKey}`}
          alt=""
          aria-hidden="true"
          hidden={art !== 'shown'}
          onLoad={() => setArt('shown')}
          onError={() => setArt('none')}
        />
      ) : null}
    </span>
  )
}
