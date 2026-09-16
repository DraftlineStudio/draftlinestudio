package coverart

// Invented artwork.
//
// Every image these tests use is generated here from arithmetic. Nothing is
// copied from a real book, a real cover or a real photograph, and nothing
// needs to be: the pipeline cares about gradients, grain, hard edges, alpha
// and colour space, all of which can be drawn.
//
// syntheticCover is built to behave like a real cover under a codec rather
// than to look like one: a vertical gradient that will band if anything
// mistreats it, a plate of flat colour where blocking shows, hard-edged
// rectangles standing in for title type, and a layer of fine deterministic
// grain over the whole thing so the file compresses the way a photographic
// wrap does instead of collapsing to nothing.

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/tiff"
)

// grain is a deterministic value noise: the same picture every run, on every
// machine, so a size assertion means something.
func grain(x, y, salt int) float64 {
	n := uint32(x)*374761393 + uint32(y)*668265263 + uint32(salt)*2246822519
	n = (n ^ (n >> 13)) * 1274126177
	n ^= n >> 16
	return float64(n%2048)/2048 - 0.5
}

type coverPaint struct {
	grainAmount float64 // 0 for a flat vector-style cover
	monochrome  bool
	alpha       bool
}

// paintCover draws artwork w by h.
func paintCover(w, h int, paint coverPaint) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		v := float64(y) / float64(h-1)
		// A long tonal sweep from near black to a pale sky. This is the part
		// that bands when a resizer works in gamma space.
		base := 0.04 + 0.80*math.Pow(v, 1.35)
		for x := 0; x < w; x++ {
			u := float64(x) / float64(w-1)
			r := base * (0.82 + 0.30*u)
			g := base * (0.90 + 0.10*math.Sin(u*3.1))
			b := base * (1.15 - 0.20*u)

			// A flat plate across the upper third: where blocking shows.
			if y > h/8 && y < h/3 && x > w/10 && x < w*9/10 {
				r, g, b = 0.09, 0.13, 0.19
			}
			// Hard-edged bars standing in for title and author type.
			if (y > h*2/5 && y < h*2/5+h/22 && x > w/8 && x < w*7/8) ||
				(y > h*3/4 && y < h*3/4+h/40 && x > w/4 && x < w*3/4) {
				r, g, b = 0.94, 0.91, 0.84
			}
			// A low-frequency mottle: broad texture that survives being shrunk to
			// thumbnail size, the way a painted or photographed cover does and a
			// flat colour field does not.
			mottle := 0.045 * (math.Sin(u*17.3+v*9.1) * math.Cos(v*23.7-u*5.2))
			mottle += 0.028 * math.Sin((u+v)*41.9)
			r, g, b = r+mottle, g+mottle*0.85, b+mottle*1.2

			if paint.grainAmount > 0 {
				n := grain(x, y, 1) * paint.grainAmount
				r, g, b = r+n, g+n*0.9, b+n*1.1
				r += grain(x, y, 7) * paint.grainAmount * 0.4
			}
			if paint.monochrome {
				y709 := 0.2126*r + 0.7152*g + 0.0722*b
				r, g, b = y709, y709, y709
			}
			a := uint8(255)
			if paint.alpha {
				// A transparent corner and a soft edge across it, so that
				// flattening has both a hard and a graded case to handle.
				d := float64(x)/float64(w) + float64(y)/float64(h)
				switch {
				case d < 0.18:
					a = 0
				case d < 0.34:
					a = uint8(255 * (d - 0.18) / 0.16)
				}
			}
			img.SetNRGBA(x, y, color.NRGBA{R: clampF(r), G: clampF(g), B: clampF(b), A: a})
		}
	}
	return img
}

func clampF(v float64) uint8 {
	n := int(v*255 + 0.5)
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}

// writeJPEG, writePNG and writeTIFF put artwork on disk, which is the only
// form this package accepts.
func writeJPEG(t *testing.T, dir, name string, img image.Image) string {
	t.Helper()
	path := filepath.Join(dir, name)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encoding the test JPEG: %v", err)
	}
	writeFile(t, path, buf.Bytes())
	return path
}

func writePNG(t *testing.T, dir, name string, img image.Image) string {
	t.Helper()
	path := filepath.Join(dir, name)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding the test PNG: %v", err)
	}
	writeFile(t, path, buf.Bytes())
	return path
}

func writeTIFF(t *testing.T, dir, name string, img image.Image, compression tiff.CompressionType) string {
	t.Helper()
	path := filepath.Join(dir, name)
	var buf bytes.Buffer
	if err := tiff.Encode(&buf, img, &tiff.Options{Compression: compression}); err != nil {
		t.Fatalf("encoding the test TIFF: %v", err)
	}
	writeFile(t, path, buf.Bytes())
	return path
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// ── TIFFs x/image cannot read ──────────────────────────────────────────────
//
// x/image/tiff can only write RGB, greyscale and paletted TIFFs, so the two
// files that matter most to this milestone - a CMYK TIFF and a
// JPEG-compressed TIFF - cannot be produced by encoding them. They are made by
// encoding an ordinary TIFF and then rewriting one tag in its directory, which
// is exactly the byte a print program would have written differently. The
// decoder reaches its refusal on the same tag either way.

const (
	tagPhotometric  = 262
	tagCompression  = 259
	photometricCMYK = 5
	compressionJPEG = 7
)

// patchTIFFTag rewrites the value of a SHORT tag in the first image file
// directory of a little-endian TIFF.
func patchTIFFTag(t *testing.T, data []byte, tag, value uint16) []byte {
	t.Helper()
	out := append([]byte(nil), data...)
	if len(out) < 8 || out[0] != 'I' || out[1] != 'I' {
		t.Fatalf("test TIFF is not little-endian; this helper only reads the form x/image writes")
	}
	ifd := int(binary.LittleEndian.Uint32(out[4:8]))
	if ifd+2 > len(out) {
		t.Fatalf("test TIFF directory offset %d is past the end", ifd)
	}
	count := int(binary.LittleEndian.Uint16(out[ifd : ifd+2]))
	for i := 0; i < count; i++ {
		entry := ifd + 2 + i*12
		if entry+12 > len(out) {
			break
		}
		if binary.LittleEndian.Uint16(out[entry:entry+2]) == tag {
			binary.LittleEndian.PutUint16(out[entry+8:entry+10], value)
			return out
		}
	}
	t.Fatalf("test TIFF has no tag %d to patch", tag)
	return nil
}

func writePatchedTIFF(t *testing.T, dir, name string, img image.Image, compression tiff.CompressionType, tag, value uint16) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tiff.Encode(&buf, img, &tiff.Options{Compression: compression}); err != nil {
		t.Fatalf("encoding the test TIFF: %v", err)
	}
	path := filepath.Join(dir, name)
	writeFile(t, path, patchTIFFTag(t, buf.Bytes(), tag, value))
	return path
}
