package coverart

// Choosing a JPEG quality, and choosing a format.
//
// A fixed quality number is a bet that every cover compresses the same way,
// and covers do not. A flat two-colour design is visually perfect at quality
// 70 and wastes three quarters of its bytes at 92. A grainy photographic wrap
// is visibly mushy at 70 and needs 90 to hold its texture. Amazon deducts a
// delivery charge per megabyte from every sale, so the bytes are a permanent
// cost on the author's royalty - and a soft cover is a permanent cost on the
// book. Neither is a number to guess at.
//
// So the encoder is asked several times and the answers are measured. The
// ladder is FIXED and SHORT - five probes - because this runs while somebody
// waits, and because a longer ladder buys less than the first two rungs do.
// The measurement is multi-scale SSIM on the LINEAR luma plane: linear
// because that is where a difference in light is a difference the eye weights
// evenly, luma because JPEG's damage is overwhelmingly there, multi-scale
// because a single-scale score cannot tell a softened texture from a blocked
// gradient and a cover suffers from the second far more than the first.
//
// The smallest probe that stays above the threshold wins. If the budget runs
// out before any probe passes, or if the metric returns something impossible,
// the fallback quality is used and the caller is not left guessing which
// happened - the chosen quality is reported on the Derivative.
//
// Choosing the FORMAT - JPEG or PNG - is a separate decision, made by
// looking at the artwork rather than by measuring it. It lives in encode.go.

import (
	"bytes"
	"image"
	"image/jpeg"
	"math"
	"time"
)

// The ladder, in ascending order. The lowest rung that measures well enough is
// the one that ships.
var qualityLadder = []int{78, 84, 88, 92, 96}

const (
	// qualityThreshold is the MS-SSIM score a probe must reach against the
	// image as it stood before encoding. 0.995 is where, on this metric, the
	// difference stops being a difference anyone finds by looking: below it
	// the flat areas of a cover start to show the block structure.
	qualityThreshold = 0.995

	// fallbackQuality is used when nothing on the ladder passes, or when the
	// clock runs out before the ladder finishes. It is the TOP of the ladder,
	// not a middling number: a cover nobody measured should err towards being
	// too large rather than towards being soft, and when the whole ladder did
	// run this costs no extra encode because the top rung is already in hand.
	fallbackQuality = 96

	// searchBudget bounds the whole ladder for one derivative. Attaching a
	// cover is an interactive act; a perfect answer that arrives four seconds
	// late is the wrong answer.
	searchBudget = 1200 * time.Millisecond
)

// encodeJPEGBySearch walks the ladder and returns the smallest acceptable
// encoding, with the quality it used.
func encodeJPEGBySearch(img image.Image) ([]byte, int, error) {
	deadline := time.Now().Add(searchBudget)
	reference := linearLuma(img)

	var lastData []byte
	var lastQuality int
	for _, quality := range qualityLadder {
		data, err := encodeJPEG(img, quality)
		if err != nil {
			return nil, 0, err
		}
		lastData, lastQuality = data, quality

		decoded, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			// A probe that will not decode is not a probe. Stop measuring and
			// take the fallback rather than trusting the rest of the ladder.
			break
		}
		if msSSIM(reference, linearLuma(decoded), img.Bounds().Dx(), img.Bounds().Dy()) >= qualityThreshold {
			return data, quality, nil
		}
		if time.Now().After(deadline) {
			break
		}
	}

	// Nothing measured well enough, or the clock ran out. If the ladder
	// happened to end on the fallback quality, that encoding is already in
	// hand; otherwise encode it once more.
	if lastQuality == fallbackQuality && lastData != nil {
		return lastData, lastQuality, nil
	}
	data, err := encodeJPEG(img, fallbackQuality)
	if err != nil {
		return nil, 0, err
	}
	return data, fallbackQuality, nil
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

// linearLuma is the plane the metric runs on: brightness, in light rather than
// in sRGB code, normalised to 0..1.
func linearLuma(img image.Image) []float64 {
	transferTables()
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]float64, w*h)

	if grey, ok := img.(*image.Gray); ok {
		for y := 0; y < h; y++ {
			row := grey.Pix[y*grey.Stride : y*grey.Stride+w]
			for x := 0; x < w; x++ {
				out[y*w+x] = float64(srgbToLinear16[uint16(row[x])<<8|uint16(row[x])])
			}
		}
		return out
	}

	accessor, direct := img.(image.RGBA64Image)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, bl uint16
			if direct {
				c := accessor.RGBA64At(b.Min.X+x, b.Min.Y+y)
				r, g, bl = c.R, c.G, c.B
			} else {
				cr, cg, cb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
				r, g, bl = uint16(cr), uint16(cg), uint16(cb)
			}
			out[y*w+x] = 0.2126*float64(srgbToLinear16[r]) +
				0.7152*float64(srgbToLinear16[g]) +
				0.0722*float64(srgbToLinear16[bl])
		}
	}
	return out
}

// Multi-scale SSIM.
//
// Three scales, each a 2x2 box-averaged halving of the one before it, scored
// with ordinary SSIM over 8x8 windows at a stride of 4. The scales are
// combined as a weighted product, weighted towards the coarse ones: a cover is
// looked at whole far more often than it is looked at closely, and a fault in
// a broad gradient matters more than a fault in one leaf of a tree.
var msScaleWeights = [3]float64{0.25, 0.35, 0.40}

func msSSIM(a, b []float64, w, h int) float64 {
	if len(a) != len(b) || len(a) != w*h {
		return 0
	}
	score := 1.0
	for scale := 0; scale < len(msScaleWeights); scale++ {
		if w < 8 || h < 8 {
			// Too small to window. The scales that did run carry the answer;
			// renormalising here would let one coarse scale claim a perfect
			// score on a four-pixel image.
			break
		}
		s := ssim(a, b, w, h)
		if s <= 0 {
			return 0
		}
		score *= math.Pow(s, msScaleWeights[scale])
		if scale == len(msScaleWeights)-1 {
			break
		}
		a, b, w, h = halve(a, w, h), halve(b, w, h), w/2, h/2
	}
	return score
}

// ssim is mean structural similarity over 8x8 windows.
//
// C1 and C2 are the standard stabilisers for a 0..1 signal: they stop a window
// that is flat in both images - a plain black margin, the white of a page -
// from producing a zero-over-zero score.
func ssim(a, b []float64, w, h int) float64 {
	const (
		window = 8
		stride = 4
		c1     = 0.01 * 0.01
		c2     = 0.03 * 0.03
	)
	total, count := 0.0, 0
	for y := 0; y+window <= h; y += stride {
		for x := 0; x+window <= w; x += stride {
			var sa, sb, saa, sbb, sab float64
			for j := 0; j < window; j++ {
				rowA := a[(y+j)*w+x:]
				rowB := b[(y+j)*w+x:]
				for i := 0; i < window; i++ {
					va, vb := rowA[i], rowB[i]
					sa += va
					sb += vb
					saa += va * va
					sbb += vb * vb
					sab += va * vb
				}
			}
			const n = window * window
			ma, mb := sa/n, sb/n
			va := saa/n - ma*ma
			vb := sbb/n - mb*mb
			cov := sab/n - ma*mb
			total += ((2*ma*mb + c1) * (2*cov + c2)) / ((ma*ma + mb*mb + c1) * (va + vb + c2))
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return total / float64(count)
}

// halve box-averages a plane down by two in each direction.
func halve(plane []float64, w, h int) []float64 {
	hw, hh := w/2, h/2
	out := make([]float64, hw*hh)
	for y := 0; y < hh; y++ {
		top := plane[2*y*w:]
		bottom := plane[(2*y+1)*w:]
		row := out[y*hw : y*hw+hw]
		for x := 0; x < hw; x++ {
			row[x] = (top[2*x] + top[2*x+1] + bottom[2*x] + bottom[2*x+1]) / 4
		}
	}
	return out
}
