// Package coverart turns the artwork an author points at into the cover files
// a book actually ships with.
//
// The whole package works from a PATH on disk and returns bytes. Nothing here
// ever sees an upload: a cover is a 40 MB TIFF as often as not, and pushing
// one through a webview and across a JSON bridge would cost more than the
// decode does. The caller opens a file dialog, or catches a drop, and hands
// this package the path.
//
// What comes out is deliberately small and deliberately plain:
//
//	cover.jpg        fitted inside 1600 x 2560, sRGB, no alpha
//	cover_thumb.jpg  fitted inside 252 x 378, for the rail
//	cover_large.jpg  fitted inside 2400 x 3840, only when asked for
//
// 1600 x 2560 is Amazon's stated ideal and clears Apple's 1400-pixel short
// edge. The larger derivative exists because Kobo asks for 2400, and it is not
// the default because Amazon deducts a delivery charge per megabyte from every
// sale: on a book that sells, cover size is a permanent royalty cost, not a
// storage one.
//
// Two things this package does that most resizers do not:
//
//   - It resamples in LINEAR LIGHT. sRGB is a gamma-encoded space; averaging
//     two sRGB numbers does not average the two amounts of light they stand
//     for, so an ordinary downscale darkens fine bright detail against dark
//     ground - exactly the texture a cover is made of. Every pixel goes out of
//     sRGB before the single resampling pass and back afterwards.
//   - It chooses JPEG quality by searching rather than by picking a number.
//     See quality.go.
//
// DPI is ignored throughout, and that is not an oversight: Go's encoders
// cannot write density metadata, no ebook reader consults it, and Amazon
// accepts 72 dpi submissions. Pixels are the whole specification.
//
// None of this is a print cover. The derivative is a front cover only - no
// spine, no back, no bleed - and at 300 dpi 1600 x 2560 is 5.33 by 8.53
// inches, smaller than Draftline's own 6 by 9 trim. The print-ready original
// stays where the author put it; see Fingerprint in source.go.
package coverart

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Input tolerance. An author's cover arrives as whatever their designer
	// sent, so the decoders are registered here even though only the first two
	// are common.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Limits applied before anything large is allocated.
const (
	// MaxSourceBytes is the largest file this will open. A cover bigger than
	// this is a layered working file, not artwork.
	MaxSourceBytes = 128 << 20

	// MaxSourcePixels is checked against the header, before decoding. A
	// 16-bit RGBA TIFF decodes to eight bytes a pixel, so 100 megapixels is
	// already 800 MB of image before this package adds its own linear-light
	// copy. Refusing at the header costs nothing; refusing after the
	// allocation may not be possible at all.
	MaxSourcePixels = 100_000_000

	// MinSourceWidth and MinSourceHeight are the retailer floor. Below this a
	// cover is rejected rather than upscaled, because upscaling invents detail
	// and a storefront will reject it anyway.
	MinSourceWidth  = 625
	MinSourceHeight = 1000
)

// Cover proportions. A trade cover is 1.6 times as tall as it is wide. A long
// way off that is not an error - it is someone's deliberate square - but it is
// worth saying, because the usual cause is artwork cropped for a different
// shop.
const (
	idealAspect     = 1.6
	aspectTolerance = 0.2
)

// Box is a bounding box a derivative is fitted inside. Fitting preserves the
// artwork's own proportions: the box is a maximum, never a target shape.
type Box struct{ W, H int }

// The three derivatives.
var (
	CoverBox = Box{W: 1600, H: 2560}
	ThumbBox = Box{W: 252, H: 378}
	LargeBox = Box{W: 2400, H: 3840}
)

// Derivative is one encoded image and the facts a screen needs to describe it.
type Derivative struct {
	Name     string `json:"name"`
	Mime     string `json:"mime"`
	Encoding string `json:"encoding"` // "jpeg" or "png"
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Bytes    int    `json:"bytes"`
	// Quality is the JPEG quality the search settled on, and 0 for a PNG.
	Quality int    `json:"quality"`
	Data    []byte `json:"-"`
}

// Options are the caller's choices. There is only one, and its default is the
// cheaper answer.
type Options struct {
	// Large also produces cover_large.jpg at 2400 x 3840. Off by default.
	Large bool
}

// Result is everything one attach produced.
type Result struct {
	Cover Derivative
	Thumb Derivative
	Large *Derivative

	SourceWidth  int
	SourceHeight int
	SourceFormat string

	// Greyscale is set when the artwork carried no colour at all and was
	// encoded as greyscale, which roughly halves the file.
	Greyscale bool
	// ConvertedFromCMYK is set when the artwork arrived in CMYK.
	ConvertedFromCMYK bool
	// FlattenedAlpha is set when transparency was composited onto white.
	FlattenedAlpha bool

	// Notes are sentences for the author, in their own terms, about what was
	// done to their artwork. They are not warnings in the logging sense and
	// they are not errors: a conversion that changed the picture has to be
	// visible on screen, because a silently dulled cover is worse than a
	// stated one.
	Notes []string

	// Elapsed is how long the whole pipeline took, so that a test can hold
	// attaching a cover to being an interactive act rather than a wait.
	Elapsed time.Duration
}

// knownExtensions are the file types worth opening. An extension that is not
// here is refused by name rather than guessed at, because the alternative is
// reading 128 MB of somebody's layered working file to find out it is not an
// image.
var knownExtensions = map[string]string{
	".jpg":  "JPEG",
	".jpeg": "JPEG",
	".png":  "PNG",
	".tif":  "TIFF",
	".tiff": "TIFF",
	".webp": "WebP",
	".bmp":  "BMP",
	".gif":  "GIF",
}

// Probe is what the file header says, before any pixel is decoded.
type Probe struct {
	Path   string
	Format string
	Width  int
	Height int
	Bytes  int64
	// Aspect is height divided by width.
	Aspect float64
}

// Inspect reads the header of the file at path and applies every limit that
// can be applied without allocating an image.
//
// The order is the order of cost: stat first, then the extension, then the
// image header. A 12000 x 12000 scan is refused here, having allocated a few
// hundred bytes of header and no pixels at all.
func Inspect(path string) (Probe, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Probe{}, fmt.Errorf("there is no file at %s any more", path)
		}
		return Probe{}, fmt.Errorf("%s could not be read: %w", filepath.Base(path), err)
	}
	if info.IsDir() {
		return Probe{}, fmt.Errorf("%s is a folder, not a cover image", filepath.Base(path))
	}
	if info.Size() > MaxSourceBytes {
		return Probe{}, fmt.Errorf(
			"%s is %s, and Draftline opens cover artwork up to %s. Flatten the working file and export a single image",
			filepath.Base(path), humanBytes(info.Size()), humanBytes(MaxSourceBytes))
	}
	if info.Size() == 0 {
		return Probe{}, fmt.Errorf("%s is empty", filepath.Base(path))
	}

	ext := strings.ToLower(filepath.Ext(path))
	label, known := knownExtensions[ext]
	if !known {
		return Probe{}, fmt.Errorf(
			"Draftline reads cover artwork as JPEG, PNG, TIFF, WebP or BMP, and %s is none of those",
			filepath.Base(path))
	}

	f, err := os.Open(path)
	if err != nil {
		return Probe{}, fmt.Errorf("%s could not be opened: %w", filepath.Base(path), err)
	}
	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return Probe{}, decodeFault(path, label, err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return Probe{}, fmt.Errorf("%s reports no size at all and cannot be used", filepath.Base(path))
	}
	// int64 on purpose: a corrupt header claiming a million on each side would
	// overflow a 32-bit multiply and pass a check done in int.
	if int64(cfg.Width)*int64(cfg.Height) > MaxSourcePixels {
		return Probe{}, fmt.Errorf(
			"%s is %d x %d, which is %.0f megapixels. Draftline works with cover artwork up to %d megapixels - above that a single image can need a gigabyte of memory just to open",
			filepath.Base(path), cfg.Width, cfg.Height,
			float64(cfg.Width)*float64(cfg.Height)/1e6, MaxSourcePixels/1_000_000)
	}
	if cfg.Width < MinSourceWidth || cfg.Height < MinSourceHeight {
		return Probe{}, fmt.Errorf(
			"%s is %d x %d. A cover needs to be at least %d x %d. Draftline will not enlarge artwork, because enlarging invents detail that was never there and a storefront refuses it anyway",
			filepath.Base(path), cfg.Width, cfg.Height, MinSourceWidth, MinSourceHeight)
	}

	return Probe{
		Path:   path,
		Format: format,
		Width:  cfg.Width,
		Height: cfg.Height,
		Bytes:  info.Size(),
		Aspect: float64(cfg.Height) / float64(cfg.Width),
	}, nil
}

// Prepare runs the whole pipeline: inspect, decode, convert, flatten,
// resample in linear light, denoise the chroma, and encode.
func Prepare(path string, opts Options) (Result, error) {
	started := time.Now()

	probe, err := Inspect(path)
	if err != nil {
		return Result{}, err
	}

	src, err := decodeImage(path, probe.Format)
	if err != nil {
		return Result{}, err
	}

	res, err := prepareDecoded(src, probe, opts)
	res.Elapsed = time.Since(started)
	return res, err
}

// prepareDecoded is everything after the file has been read.
//
// It is split out from Prepare so the pipeline can be driven with artwork that
// no Go encoder can put on disk. CMYK is the case that matters: image/jpeg
// DECODES CMYK and YCCK through the Adobe APP14 marker, and neither the
// standard library nor x/image can encode either, so the only way to test what
// Draftline does with a print designer's CMYK artwork is to hand it the
// decoded image.
func prepareDecoded(src image.Image, probe Probe, opts Options) (Result, error) {
	res := Result{
		SourceWidth:  probe.Width,
		SourceHeight: probe.Height,
		SourceFormat: probe.Format,
	}
	if aspect := probe.Aspect; aspect < idealAspect-aspectTolerance || aspect > idealAspect+aspectTolerance {
		res.Notes = append(res.Notes, fmt.Sprintf(
			"This artwork is %d x %d, which is %.2f to 1 rather than the 1.6 to 1 a cover usually is. Draftline keeps the shape it was given and never crops, so the cover will sit in its own proportions wherever it is shown.",
			probe.Width, probe.Height, aspect))
	}

	// CMYK arrives from a print designer. Converting it without the source's
	// ICC profile shifts colour - usually towards dull - and pure-Go colour
	// management does not exist, so the conversion is the naive formula in
	// image/color and the author is told so rather than left to notice.
	if _, isCMYK := src.(*image.CMYK); isCMYK {
		res.ConvertedFromCMYK = true
		res.Notes = append(res.Notes, CMYKNote)
	}

	linear, flattened := toLinear(src)
	src = nil
	if flattened {
		res.FlattenedAlpha = true
		res.Notes = append(res.Notes,
			"This artwork had transparent areas. A JPEG cannot hold transparency, so Draftline laid the artwork on opaque white before encoding it. Anything that was see-through is now white.")
	}

	grey := isMonochrome(linear)
	res.Greyscale = grey

	cover := resample(linear, CoverBox)
	coverData, err := encodeBest(finish(cover, grey), "cover", true)
	if err != nil {
		return Result{}, err
	}
	res.Cover = coverData

	// The thumbnail is resampled from the cover rather than from the source.
	// It is the same picture in the same linear light, one further shrink of
	// an image that is still six times wider than the thumbnail box, and doing
	// it this way keeps a second full-resolution pass out of the wait an
	// author sits through when they attach a cover.
	thumbData, err := encodeBest(finish(resample(cover, ThumbBox), grey), "cover_thumb", false)
	if err != nil {
		return Result{}, err
	}
	res.Thumb = thumbData

	if opts.Large {
		largeData, err := encodeBest(finish(resample(linear, LargeBox), grey), "cover_large", true)
		if err != nil {
			return Result{}, err
		}
		res.Large = &largeData
	}

	if grey {
		res.Notes = append(res.Notes,
			"This artwork carries no colour at all, so it was encoded as greyscale. That roughly halves the file, and an e-ink reader shows it in grey either way.")
	}

	return res, nil
}

// CMYKNote is what the screen says when artwork arrives in CMYK. It is a
// constant because the panel shows it and a test asserts it: the conversion is
// lossy in a way the author can see, and the one thing worse than a dulled
// cover is a dulled cover nobody mentioned.
const CMYKNote = "This artwork was in CMYK, the colour space a printing press uses. Draftline converted it to sRGB with the standard formula and no colour profile, which is the only conversion a program without colour management can do: expect the screen colours to sit a little flatter than the printed ones. If that shift matters, export an sRGB copy from the program the cover was made in and attach that instead."

// decodeImage reads the whole image.
func decodeImage(path, format string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%s could not be opened: %w", filepath.Base(path), err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, decodeFault(path, strings.ToUpper(format), err)
	}
	return img, nil
}

// decodeFault turns a decoder's own words into the author's.
//
// x/image/tiff cannot read a CMYK TIFF and cannot read a JPEG-compressed
// TIFF. Both are ordinary things for a print designer to hand over, both come
// back as tiff.UnsupportedError, and neither of the decoder's own messages -
// "color model", "compression value 7" - tells anybody what to do about it.
// They collapse into one instruction here, because the fix is the same for
// both: re-export as RGB.
//
// Go's own JPEG decoder, by contrast, reads CMYK and YCCK JPEGs through the
// Adobe APP14 marker and keeps working, so a CMYK JPEG never reaches this.
func decodeFault(path, label string, err error) error {
	var unsupported tiff.UnsupportedError
	if errors.As(err, &unsupported) {
		return fmt.Errorf(
			"%s is a TIFF Draftline cannot read: it uses %s. Draftline reads RGB and greyscale TIFFs that are uncompressed or use LZW, Deflate or PackBits - not CMYK TIFFs and not JPEG-compressed ones. Re-export the cover as an RGB TIFF, or as a PNG or a JPEG, and attach it again.",
			filepath.Base(path), tiffDetail(string(unsupported)))
	}
	return fmt.Errorf("%s could not be read as a %s image: %v", filepath.Base(path), label, err)
}

// tiffDetail rewrites the decoder's shorthand as something readable, and
// passes anything unrecognised through rather than pretending to know it.
func tiffDetail(detail string) string {
	switch {
	case detail == "color model":
		return "a colour model Draftline has no decoder for, which is almost always CMYK"
	case strings.HasPrefix(detail, "compression value"):
		return "a compression Draftline has no decoder for (" + detail + "), which is almost always JPEG inside TIFF"
	default:
		return detail
	}
}

// Fit returns the size artwork takes inside a box, preserving its shape.
//
// It never enlarges. Artwork smaller than the box comes out at its own size,
// because an upscale is detail that was never drawn, and the honest answer to
// "the cover is too small" is the message Inspect already gives.
func Fit(w, h int, box Box) (int, int) {
	if w <= 0 || h <= 0 {
		return 0, 0
	}
	scale := min(float64(box.W)/float64(w), float64(box.H)/float64(h))
	if scale >= 1 {
		return w, h
	}
	outW := max(1, int(float64(w)*scale+0.5))
	outH := max(1, int(float64(h)*scale+0.5))
	// Rounding can push one side a pixel past the box; the box is a maximum.
	return min(outW, box.W), min(outH, box.H)
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d bytes", n)
	}
}
