package coverart

import "image"

// Chroma denoising, and the deliberate refusal to touch luma.
//
// Camera grain and scanner noise live in both the brightness of a picture and
// its colour, and the two behave completely differently once the image is a
// JPEG.
//
// Colour noise is pure cost. The eye cannot resolve it, JPEG stores colour at
// half resolution in each direction anyway, and speckled colour is expensive
// to encode because it fights the very subsampling the format applies to it.
// Smoothing it makes the file smaller AND the picture better, which is rare.
//
// Brightness grain is the opposite: it is doing work. A smooth sky, a fade
// behind a title, a dark gradient down a spine - these are exactly what a
// codec bands, and a little grain is what dithers those bands away. The
// problem is worse, not better, on an e-ink reader, where sixteen grey levels
// have to stand in for two hundred and fifty-six. Denoise the luma of a cover
// and the gradients go to steps. Grain is also, very often, the designer's
// choice rather than the sensor's.
//
// So the luma plane is left exactly as it is, and only Cb and Cr are smoothed,
// with the smallest kernel that does anything: a separable 1-2-1 blur, one
// pixel of support. It is a far lighter touch than the 4:2:0 subsampling the
// encoder applies immediately afterwards, which throws away three quarters of
// the colour samples outright.

// denoiseChroma smooths the colour of an 8-bit sRGB image in place and leaves
// its brightness alone. A greyscale image never reaches this: it has no
// chroma to smooth.
func denoiseChroma(img *image.RGBA) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	if w < 3 || h < 3 {
		return
	}

	// Y is kept as the exact 8-bit value the conversion produced, so that
	// recombining cannot shift brightness by a rounding step. Cb and Cr are
	// floats because they are about to be averaged.
	luma := make([]float32, w*h)
	cb := make([]float32, w*h)
	cr := make([]float32, w*h)
	for y := 0; y < h; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+w*4]
		for x := 0; x < w; x++ {
			i := x * 4
			r, g, b := float32(row[i]), float32(row[i+1]), float32(row[i+2])
			p := y*w + x
			luma[p] = 0.299*r + 0.587*g + 0.114*b
			cb[p] = -0.168736*r - 0.331264*g + 0.5*b
			cr[p] = 0.5*r - 0.418688*g - 0.081312*b
		}
	}

	blur121(cb, w, h)
	blur121(cr, w, h)

	for y := 0; y < h; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+w*4]
		for x := 0; x < w; x++ {
			p := y*w + x
			l, u, v := luma[p], cb[p], cr[p]
			i := x * 4
			row[i] = clamp8(l + 1.402*v)
			row[i+1] = clamp8(l - 0.344136*u - 0.714136*v)
			row[i+2] = clamp8(l + 1.772*u)
		}
	}
}

// blur121 applies a separable 1-2-1 kernel in place, clamping at the edges so
// the border does not darken towards nothing.
func blur121(plane []float32, w, h int) {
	scratch := make([]float32, len(plane))

	for y := 0; y < h; y++ {
		row := plane[y*w : y*w+w]
		out := scratch[y*w : y*w+w]
		for x := 0; x < w; x++ {
			left := row[max(x-1, 0)]
			right := row[min(x+1, w-1)]
			out[x] = 0.25*left + 0.5*row[x] + 0.25*right
		}
	}
	for y := 0; y < h; y++ {
		up := scratch[max(y-1, 0)*w:]
		mid := scratch[y*w:]
		down := scratch[min(y+1, h-1)*w:]
		out := plane[y*w : y*w+w]
		for x := 0; x < w; x++ {
			out[x] = 0.25*up[x] + 0.5*mid[x] + 0.25*down[x]
		}
	}
}

func clamp8(v float32) byte {
	n := int(v + 0.5)
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return byte(n)
}
