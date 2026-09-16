package coverart

// Encoding one derivative: which format, and at what quality.
//
// The two decisions are separate and are made differently.
//
// FORMAT is decided by looking at the picture. JPEG is right for almost every
// cover, and PNG is right for the minority that are flat - a two-colour
// design, a typographic cover, big fields of one ink - where JPEG's 4:2:0
// chroma subsampling smears saturated title type and its ringing shows around
// hard edges. Rather than encode a PNG of every cover to find out (which costs
// seconds on a four-megapixel image and is thrown away nearly every time),
// flatness is measured first, cheaply, and the PNG is only attempted when the
// artwork actually looks like that kind of cover.
//
// QUALITY is decided by search, and only for the cover itself: see quality.go.

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
)

// thumbQuality is fixed, and deliberately generous.
//
// The search exists because the distributable cover's bytes are charged to the
// author on every single sale. A thumbnail never leaves the project file: it
// is twenty kilobytes in an archive that holds a novel, and the only thing it
// has to do is look right in a list. There is nothing to optimise, so it is
// encoded well and the five probes are not spent.
const thumbQuality = 92

// flatThreshold is the share of pixels that have to equal the pixel to their
// left before PNG is worth trying.
const flatThreshold = 0.55

// flatColourCeiling is the second half of the same question, and on its own it
// is the half that matters.
//
// Counting runs of identical pixels was not enough, and the way it failed is
// worth recording. A smooth vertical gradient across a 1600-pixel-wide cover
// repeats every tone about six times along each row, so more than half of its
// pixels equal the pixel to their left and it read as flat - while being the
// single worst thing PNG can be handed. Measured on invented artwork: the
// gradient cover produced an 817 KB PNG in 5.2 seconds against a 110 KB JPEG
// in 0.17, and the PNG was discarded every time. That was most of the nine and
// a half seconds an author waited to attach a perfectly ordinary cover.
//
// Distinct colours separate the two cases outright. On the same measurements a
// genuinely flat poster-style cover holds 3 of them, the gradient 29,510 and a
// grainy photographic wrap 67,559. The ceiling sits far above the first and far
// below the other two on purpose: antialiased title type over a flat ground
// runs to a few hundred blends, and a duotone with a little texture to a few
// thousand, and both of those should still get their PNG.
const flatColourCeiling = 4096

// errPNGOverBudget stops a PNG encode that has already lost.
var errPNGOverBudget = errors.New("png exceeded the jpeg it has to beat")

// encodeBest encodes one derivative.
//
// search distinguishes the cover, whose size is a permanent cost, from the
// thumbnail, whose size is not.
func encodeBest(img image.Image, name string, search bool) (Derivative, error) {
	var (
		data    []byte
		quality int
		err     error
	)
	if search {
		data, quality, err = encodeJPEGBySearch(img)
	} else {
		quality = thumbQuality
		data, err = encodeJPEG(img, quality)
	}
	if err != nil {
		return Derivative{}, fmt.Errorf("the cover could not be encoded: %w", err)
	}

	out := Derivative{
		Name:     name + ".jpg",
		Mime:     "image/jpeg",
		Encoding: "jpeg",
		Width:    img.Bounds().Dx(),
		Height:   img.Bounds().Dy(),
		Bytes:    len(data),
		Quality:  quality,
		Data:     data,
	}

	if !looksFlat(img) {
		return out, nil
	}
	// BestCompression, and it is not a close call on this kind of picture:
	// flat artwork is long runs of identical pixels, which is what the deeper
	// match search is for. A two-colour cover comes out around five times
	// smaller at this level than at the default, and encodes in under a fifth
	// of a second because there is so little entropy to chew on.
	//
	// The budget below is the guard rail rather than the plan. A PNG that has
	// already written more bytes than the JPEG it is competing against cannot
	// win however it finishes, so the encode is abandoned the moment it
	// crosses that line. If the gate above is ever wrong about a cover, the
	// author waits for a fraction of the losing encode instead of all of it.
	budget := &budgetWriter{limit: len(data)}
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(budget, img); err != nil {
		return out, nil
	}
	alt := make([]byte, budget.buf.Len())
	copy(alt, budget.buf.Bytes())
	return Derivative{
		Name:     name + ".png",
		Mime:     "image/png",
		Encoding: "png",
		Width:    out.Width,
		Height:   out.Height,
		Bytes:    len(alt),
		Data:     alt,
	}, nil
}

// budgetWriter collects output until it would pass a limit, then refuses. The
// encoder writing into it stops with that error, which is the point.
type budgetWriter struct {
	buf   bytes.Buffer
	limit int
}

func (b *budgetWriter) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.limit {
		return 0, errPNGOverBudget
	}
	return b.buf.Write(p)
}

// looksFlat estimates whether the artwork is the kind PNG is for: areas of one
// exact colour, drawn from few colours.
//
// It is an estimate on purpose. Getting it wrong costs one PNG encode that is
// then abandoned at the budget above, or one PNG that was never tried; neither
// damages the cover, and the cheap answer keeps attaching a cover interactive.
func looksFlat(img image.Image) bool {
	switch src := img.(type) {
	case *image.RGBA:
		return flatRuns(src.Pix, src.Stride, src.Rect.Dx(), src.Rect.Dy(), 4)
	case *image.Gray:
		return flatRuns(src.Pix, src.Stride, src.Rect.Dx(), src.Rect.Dy(), 1)
	default:
		return false
	}
}

// flatRuns walks every fourth row, counting pixels identical to the one on
// their left and, every fourth column, the distinct colours it has seen. It
// gives up the moment the colour count passes the ceiling, which is why it
// costs almost nothing on the photographs it rejects.
func flatRuns(pix []byte, stride, w, h, depth int) bool {
	if w < 2 || h < 1 {
		return false
	}
	same, total := 0, 0
	seen := make(map[uint32]struct{}, flatColourCeiling)
	for y := 0; y < h; y += 4 {
		row := pix[y*stride : y*stride+w*depth]
		for x := 1; x < w; x++ {
			i := x * depth
			if bytes.Equal(row[i:i+depth], row[i-depth:i]) {
				same++
			}
			total++
			if x%4 != 0 {
				continue
			}
			seen[colourKey(row[i:i+depth])] = struct{}{}
			if len(seen) > flatColourCeiling {
				return false
			}
		}
	}
	if total == 0 {
		return false
	}
	return float64(same)/float64(total) >= flatThreshold
}

// colourKey packs one sampled pixel into a comparable value. Alpha is not part
// of it: by this point the artwork has been flattened onto opaque white, so
// every pixel carries the same alpha and including it would only cost a shift.
func colourKey(px []byte) uint32 {
	if len(px) == 1 {
		return uint32(px[0])
	}
	return uint32(px[0])<<16 | uint32(px[1])<<8 | uint32(px[2])
}
