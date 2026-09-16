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
// left before PNG is worth trying. Photographs and painted artwork sit far
// below it even where they look smooth, because grain and gradients change the
// last bit constantly; flat vector work sits far above it.
const flatThreshold = 0.55

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
	var buf bytes.Buffer
	// BestCompression, and it is not a close call on this kind of picture:
	// flat artwork is long runs of identical pixels, which is what the deeper
	// match search is for. A two-colour cover comes out around five times
	// smaller at this level than at the default. It is only affordable because
	// the flatness gate above means this never runs on a photograph, where the
	// same level costs seconds and saves almost nothing.
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil || buf.Len() >= len(data) {
		return out, nil
	}
	alt := make([]byte, buf.Len())
	copy(alt, buf.Bytes())
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

// looksFlat estimates how much of the artwork is areas of one exact colour, by
// counting pixels identical to the one on their left across every fourth row.
//
// It is an estimate on purpose. Getting it wrong costs one PNG encode that is
// then discarded, or one PNG that was never tried; neither damages the cover,
// and the cheap answer keeps attaching a cover interactive.
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

func flatRuns(pix []byte, stride, w, h, depth int) bool {
	if w < 2 || h < 1 {
		return false
	}
	same, total := 0, 0
	for y := 0; y < h; y += 4 {
		row := pix[y*stride : y*stride+w*depth]
		for x := 1; x < w; x++ {
			i := x * depth
			if bytes.Equal(row[i:i+depth], row[i-depth:i]) {
				same++
			}
			total++
		}
	}
	if total == 0 {
		return false
	}
	return float64(same)/float64(total) >= flatThreshold
}
