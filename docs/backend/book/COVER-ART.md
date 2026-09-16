# Cover art

The first binary Draftline owns. Everything here exists to keep image bytes out
of two places they must never be: `types.BookData`, and the Wails bridge.

Code: `wails/internal/coverart/` (the pipeline), `wails/cover.go` (entry,
cache, display route), `wails/internal/types/cover.go` (the record),
`wails/frontend/src/components/dialogs/CoverCard.tsx` and `coverModel.ts` (the
panel).

## Where the bytes are

| Where | What |
|---|---|
| `editions/<edition id>/cover.jpg` | the distributable cover, fitted inside 1600 × 2560 |
| `editions/<edition id>/cover_thumb.jpg` | the rail and panel thumbnail, fitted inside 252 × 378 |
| `editions/<edition id>/cover_large.jpg` | opt-in, fitted inside 2400 × 3840 |
| `editions/index.json` → `editions[].cover` | the record: sizes, encoding, conversion notes, the original's path and checksum |

The extension is `.png` instead of `.jpg` when PNG came out smaller, which
happens on flat, typographic artwork.

Three rules hold this together:

1. **A cover enters as a PATH.** `runtime.OpenFileDialog` returns one, and
   `EnableFileDrop` in `main.go` makes a drag-and-drop return one too. There is
   no `<input type="file">`: a forty-megabyte TIFF would go through the webview
   and then base64 across the JSON bridge to reach the same place.
2. **A cover is never on `types.BookData`.** That struct is serialised on every
   save, and autosave runs five seconds after a keystroke. Bytes live in the
   `coverCache` in `cover.go`, keyed by edition, and reach `book.WriteArchive`
   as a sibling `book.Assets` parameter.
3. **A cover is displayed from `/editions/<edition id>/<file>`**, served by this
   process through the middleware in `plugins.go` beside `/plugins/`. The URL
   carries the cover's identifier as `?v=`, which changes whenever the artwork
   does, so replacing a cover actually changes the picture on screen.

## Surviving a save

The archive is rebuilt from scratch on every save; only the prefixes in
`preservedArchivePrefixes` (`history/`, `editions/`) are carried across. Cover
members are written stored (`zip.Store`), not deflated — an already-compressed
JPEG gains nothing from a second pass and costs CPU on every autosave.

An unchanged cover is never re-encoded: it survives by passthrough. Only what
has changed since the last save is in `Assets.Files`. `Assets.Superseded` names
the prefixes this save replaces, which is what lets a `cover.jpg` be replaced by
a `cover.png` without the old one living for ever beside it.

The edition identifier becomes a folder name and a URL segment, so
`prepareEditionsData` refuses one that could not be either.

## The pipeline

In order, in `internal/coverart`:

1. **Stat.** Reject a folder, an empty file, anything over 128 MB, and any
   extension that is not JPEG, PNG, TIFF, WebP, BMP or GIF.
2. **Probe** with `image.DecodeConfig`. Reject over 100 megapixels *before*
   allocating — a 16-bit TIFF decodes to eight bytes a pixel — and below the
   625 × 1000 retailer floor. Note, but accept, artwork far from 1.6:1.
3. **Decode.** Go's JPEG decoder reads CMYK and YCCK through the Adobe APP14
   marker and keeps working. `x/image/tiff` reads neither CMYK TIFF nor
   JPEG-compressed TIFF; both come back as `tiff.UnsupportedError` and both map
   to one instruction — re-export as RGB.
4. **CMYK → sRGB** with the naive formula in `image/color`, because pure-Go
   colour management does not exist. Said on screen, every time: see
   `coverart.CMYKNote`.
5. **Flatten alpha onto white**, in linear light, so a transparent PNG gets no
   dark fringe from the JPEG encoder.
6. **Resample in linear light.** Out of sRGB into a 16-bit linear buffer, one
   `draw.CatmullRom.Scale` pass, back to sRGB. One pass is correct: x/image/draw
   widens the kernel's support by the scale factor when shrinking, so it
   area-averages and there is no aliasing left for a prefilter to remove.
   `internal/coverart/linear.go` has the argument in full.
7. **Denoise chroma, leave luma alone.** Colour noise is invisible and expensive
   under 4:2:0. Brightness grain dithers gradients that would otherwise band,
   and e-ink's sixteen greys make banding worse.
8. **Choose a quality by search.** A fixed ladder of five JPEG probes, scored by
   multi-scale SSIM on the linear luma plane; the smallest that clears the
   threshold ships. The ladder and the budget are in `quality.go`; the
   format decision — JPEG or PNG — is in `encode.go` and is made by measuring
   how flat the artwork is, so the PNG encode never runs on a photograph.

Always sRGB. DPI is ignored throughout: Go's encoders cannot write density
metadata, no ebook reader consults it, and 72 dpi submissions are accepted.

## What it is not

Not a print cover, and the panel says so. Front only — no spine, no back, no
bleed — and at 300 dpi 1600 × 2560 is 5.33 × 8.53 inches, smaller than the 6 × 9
trim the export wizard prints at.

The print-ready original is recorded rather than copied: path, size and a
SHA-256 checksum. `coverart.CheckSource` answers `present`, `moved` or
`changed`, and the panel shows the answer. A print export must read the original
at export time and ask when it has moved, rather than silently enlarging the
1600-pixel derivative to trim size, which would print at about 190 dpi.

## Dependency

One: `golang.org/x/image` (tiff, draw; bmp and webp for input tolerance). Its
only requirement is `x/text`, already direct at a higher version, so the module
graph grew by exactly one line. `CGO_ENABLED=0` is unchanged and unchallenged —
every cgo imaging library is disqualified by it, `mozjpeg` included.
