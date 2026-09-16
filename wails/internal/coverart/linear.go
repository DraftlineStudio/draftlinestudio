package coverart

// Linear light.
//
// sRGB is not a measure of light. The number 128 in an sRGB image does not
// mean half as much light as 255; it means about 21% as much. Every averaging
// operation - and a downscale is nothing but averaging - is therefore wrong if
// it is done on the stored numbers. The error is not subtle on the kind of
// picture a cover is: fine bright detail on a dark ground loses brightness,
// smooth skies pick up a dirty cast, and a starfield turns grey.
//
// So the artwork is converted out of sRGB into linear light before it is
// resampled, and back into sRGB afterwards. This is the single biggest
// quality win available to a resizer and almost nothing does it.
//
// The linear buffer is *image.RGBA64: sixteen bits a channel, because linear
// light in eight bits bands visibly in the shadows, and RGBA64 specifically
// because x/image/draw takes a no-allocation path for any source implementing
// image.RGBA64Image and any destination implementing draw.RGBA64Image. A
// float buffer would be tidier and would run through the interface-boxing
// path instead, one heap allocation per kernel tap.

import (
	"image"
	"image/color"
	"math"
	"sync"

	"golang.org/x/image/draw"
)

// Two lookup tables stand in for the transfer function in both directions.
//
// Forward: 16-bit sRGB code to linear light, as float32 in 0..1.
// Reverse: linear light quantised to 16 bits, to an 8-bit sRGB code.
//
// The reverse table is indexed by the linear value, not by the sRGB one, which
// is why it can be a table at all. Sixteen bits of linear resolves about
// twenty steps inside the darkest sRGB code, so the quantisation costs
// nothing that eight-bit output could show.
var (
	transferOnce   sync.Once
	srgbToLinear16 [65536]float32
	linearToSRGB8  [65536]uint8
)

func transferTables() {
	transferOnce.Do(func() {
		for i := range srgbToLinear16 {
			srgbToLinear16[i] = float32(srgbDecode(float64(i) / 65535))
		}
		for i := range linearToSRGB8 {
			v := srgbEncode(float64(i) / 65535)
			linearToSRGB8[i] = uint8(math.Round(v * 255))
		}
	})
}

// srgbDecode is the sRGB electro-optical transfer function: stored value in,
// light out. The linear toe below 0.04045 is part of the standard, not an
// approximation of it.
func srgbDecode(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// srgbEncode is the inverse.
func srgbEncode(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// toLinear converts a decoded image into a linear-light RGBA64 buffer with
// transparency already composited onto opaque white.
//
// Flattening happens here, in linear light, rather than later in sRGB, for the
// same reason the resampling does: a half-transparent pixel over white is a
// mixture of two amounts of light, and mixing the stored numbers instead gives
// a fringe that is too dark. The fringe is what a transparent PNG otherwise
// picks up from the JPEG encoder, since JPEG has no alpha and something has to
// be behind the artwork.
//
// The second return value reports whether anything was actually transparent,
// so the screen only mentions flattening when flattening happened.
func toLinear(src image.Image) (*image.RGBA64, bool) {
	transferTables()

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA64(image.Rect(0, 0, w, h))

	// Every image type in the standard library, and every one x/image
	// produces, implements RGBA64At. The interface path is the fallback for
	// anything exotic; it boxes a color.Color per pixel, which is why it is
	// not the main path.
	accessor, direct := src.(image.RGBA64Image)

	flattened := false
	for y := 0; y < h; y++ {
		row := dst.Pix[y*dst.Stride : y*dst.Stride+w*8]
		for x := 0; x < w; x++ {
			var c color.RGBA64
			if direct {
				c = accessor.RGBA64At(b.Min.X+x, b.Min.Y+y)
			} else {
				r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
				c = color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(bl), A: uint16(a)}
			}

			var lr, lg, lb float32
			switch c.A {
			case 0xffff:
				lr = srgbToLinear16[c.R]
				lg = srgbToLinear16[c.G]
				lb = srgbToLinear16[c.B]
			case 0:
				// Wholly transparent: the white behind it, entire.
				flattened = true
				lr, lg, lb = 1, 1, 1
			default:
				flattened = true
				// RGBA64At is premultiplied. Undo that to get the colour the
				// artist chose, linearise it, then mix it with white by alpha.
				alpha := float32(c.A) / 65535
				lr = mixOverWhite(unpremultiply(c.R, c.A), alpha)
				lg = mixOverWhite(unpremultiply(c.G, c.A), alpha)
				lb = mixOverWhite(unpremultiply(c.B, c.A), alpha)
			}

			i := x * 8
			putChannel(row[i:i+2], lr)
			putChannel(row[i+2:i+4], lg)
			putChannel(row[i+4:i+6], lb)
			row[i+6], row[i+7] = 0xff, 0xff
		}
	}
	return dst, flattened
}

// unpremultiply recovers the stored sRGB code of a channel and returns the
// light it stands for.
func unpremultiply(c, a uint16) float32 {
	v := uint32(c) * 65535 / uint32(a)
	if v > 65535 {
		v = 65535
	}
	return srgbToLinear16[v]
}

// mixOverWhite composites one linear channel over opaque white.
func mixOverWhite(linear, alpha float32) float32 {
	return linear*alpha + (1 - alpha)
}

func putChannel(dst []byte, v float32) {
	n := int32(v*65535 + 0.5)
	if n < 0 {
		n = 0
	} else if n > 65535 {
		n = 65535
	}
	dst[0] = byte(uint16(n) >> 8)
	dst[1] = byte(uint16(n))
}

// resample fits a linear-light image inside a box in ONE pass.
//
// One pass is correct, not a shortcut. x/image/draw widens a kernel's support
// by the scale factor when it shrinks, so a Catmull-Rom shrink to a third of
// the size reads a neighbourhood three times as wide and area-averages it.
// There is no aliasing left for a prefilter to remove, and a prefilter would
// only blur what the kernel already handled.
//
// An image already inside the box is returned untouched: resampling 1:1 with
// any kernel is a blur, and there is nothing to gain from it.
func resample(src *image.RGBA64, box Box) *image.RGBA64 {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	w, h := Fit(sw, sh, box)
	if w == sw && h == sh {
		return src
	}
	dst := image.NewRGBA64(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst
}

// finish converts a linear-light buffer back to 8-bit sRGB and hands it to the
// chroma denoiser.
//
// The result is *image.RGBA with every alpha at 255, which is the shape both
// encoders like: image/jpeg has a fast path for it, and image/png notices it
// is opaque and writes three channels instead of four.
//
// grey collapses the picture to a single channel. It is only ever set when the
// artwork carried no colour to begin with, so nothing is being discarded.
func finish(linear *image.RGBA64, grey bool) image.Image {
	transferTables()

	w, h := linear.Bounds().Dx(), linear.Bounds().Dy()
	if grey {
		out := image.NewGray(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			src := linear.Pix[y*linear.Stride:]
			dstRow := out.Pix[y*out.Stride : y*out.Stride+w]
			for x := 0; x < w; x++ {
				i := x * 8
				// Luma in linear light, then encoded once. Taking the green
				// channel alone would be near enough on a monochrome image,
				// but the weighted sum costs nothing and is right even when
				// "monochrome" means a very slightly tinted scan.
				y709 := 0.2126*channel(src[i:]) + 0.7152*channel(src[i+2:]) + 0.0722*channel(src[i+4:])
				dstRow[x] = linearToSRGB8[quantise(y709)]
			}
		}
		return out
	}

	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		src := linear.Pix[y*linear.Stride:]
		dstRow := out.Pix[y*out.Stride : y*out.Stride+w*4]
		for x := 0; x < w; x++ {
			i, j := x*8, x*4
			dstRow[j] = linearToSRGB8[uint16(src[i])<<8|uint16(src[i+1])]
			dstRow[j+1] = linearToSRGB8[uint16(src[i+2])<<8|uint16(src[i+3])]
			dstRow[j+2] = linearToSRGB8[uint16(src[i+4])<<8|uint16(src[i+5])]
			dstRow[j+3] = 0xff
		}
	}
	// Flat artwork is not denoised. Chroma smoothing is there to take grain
	// out of a photograph; a two-colour poster has no grain, and blurring its
	// colour softens exactly the hard, saturated edges that made it worth
	// keeping as a PNG in the first place. Left alone, it stays flat, and the
	// encoder finds it flat too.
	if !looksFlat(out) {
		denoiseChroma(out)
	}
	return out
}

func channel(p []byte) float64 {
	return float64(uint16(p[0])<<8|uint16(p[1])) / 65535
}

func quantise(v float64) uint16 {
	n := int(v*65535 + 0.5)
	if n < 0 {
		return 0
	}
	if n > 65535 {
		return 65535
	}
	return uint16(n)
}

// isMonochrome reports whether the artwork carries no colour worth keeping.
//
// The test is done on the linear buffer, on every pixel, because a single
// coloured logo on an otherwise grey cover has to keep its colour and a
// sampled test can miss one. The tolerance is one part in 255 of light: a
// scan of a grey photograph drifts by that much and is still grey.
func isMonochrome(linear *image.RGBA64) bool {
	const tolerance = 1.0 / 255
	w, h := linear.Bounds().Dx(), linear.Bounds().Dy()
	for y := 0; y < h; y++ {
		row := linear.Pix[y*linear.Stride:]
		for x := 0; x < w; x++ {
			i := x * 8
			r, g, b := channel(row[i:]), channel(row[i+2:]), channel(row[i+4:])
			if math.Abs(r-g) > tolerance || math.Abs(g-b) > tolerance || math.Abs(r-b) > tolerance {
				return false
			}
		}
	}
	return true
}
