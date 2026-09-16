package coverart

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/image/tiff"
)

// The three forms artwork actually arrives in.
//
// One assertion runs over all three, because the whole point of the pipeline
// is that what a designer sent stops mattering the moment it is decoded: a
// JPEG, a PNG with transparency and an uncompressed TIFF of the same painting
// all have to come out as the same sRGB JPEG inside the same box.
func TestEveryArtworkFormatProducesOneCover(t *testing.T) {
	dir := t.TempDir()
	art := paintCover(4000, 6400, coverPaint{grainAmount: 0.055})
	alphaArt := paintCover(4000, 6400, coverPaint{grainAmount: 0.055, alpha: true})

	cases := []struct {
		name      string
		path      string
		wantAlpha bool
	}{
		{"RGB JPEG", writeJPEG(t, dir, "art.jpg", art), false},
		{"PNG with alpha", writePNG(t, dir, "art.png", alphaArt), true},
		{"baseline TIFF", writeTIFF(t, dir, "art.tif", art, tiff.Uncompressed), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Prepare(tc.path, Options{})
			if err != nil {
				t.Fatalf("preparing %s: %v", tc.name, err)
			}

			if res.Cover.Encoding != "jpeg" {
				t.Errorf("cover came out as %s; photographic artwork should be a JPEG", res.Cover.Encoding)
			}
			if res.Cover.Width != 1600 || res.Cover.Height != 2560 {
				t.Errorf("cover is %dx%d, wanted it fitted to 1600x2560", res.Cover.Width, res.Cover.Height)
			}
			// The band a real cover lands in. Below it the cover is soft;
			// above it the author pays a delivery charge on every sale.
			if res.Cover.Bytes < 400_000 || res.Cover.Bytes > 1_200_000 {
				t.Errorf("cover is %d bytes; a cover of this artwork should land between 400 KB and 1.2 MB", res.Cover.Bytes)
			}

			if res.Thumb.Width > ThumbBox.W || res.Thumb.Height > ThumbBox.H {
				t.Errorf("thumbnail is %dx%d, outside the %dx%d box", res.Thumb.Width, res.Thumb.Height, ThumbBox.W, ThumbBox.H)
			}
			if res.Thumb.Bytes < 8_000 || res.Thumb.Bytes > 25_000 {
				t.Errorf("thumbnail is %d bytes; it should land between 8 KB and 25 KB", res.Thumb.Bytes)
			}

			// Whatever went in, what comes out is opaque and has the source's
			// proportions.
			cover := decodeDerivative(t, res.Cover)
			if !cover.(interface{ Opaque() bool }).Opaque() {
				t.Error("the cover is not opaque; a JPEG cover must carry no transparency at all")
			}
			if got, want := float64(res.Cover.Height)/float64(res.Cover.Width), 1.6; math.Abs(got-want) > 0.01 {
				t.Errorf("the cover is %.3f to 1; the artwork was 1.6 to 1 and nothing here crops", got)
			}

			if res.FlattenedAlpha != tc.wantAlpha {
				t.Errorf("FlattenedAlpha is %v, wanted %v", res.FlattenedAlpha, tc.wantAlpha)
			}
			if res.ConvertedFromCMYK {
				t.Error("nothing here was CMYK")
			}
		})
	}
}

// Resampling in linear light is the pipeline's one genuinely unusual claim, so
// it gets the assertion that can tell the difference.
//
// A one-pixel checkerboard of black and white is half black and half white.
// Half the LIGHT is 0.5 linear, which is sRGB 188. Averaging the stored
// numbers instead gives 128 - the mid grey almost every resizer produces, and
// about a third too dark. Sixty levels apart is not a rounding difference; it
// is the difference between a starfield and a grey rectangle.
func TestResamplingHappensInLinearLight(t *testing.T) {
	const size = 2048
	board := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			v := uint8(0)
			if (x+y)%2 == 0 {
				v = 255
			}
			board.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}

	linear, _ := toLinear(board)
	small := finish(resample(linear, Box{W: 64, H: 64}), false)

	mean := meanChannel(t, small)
	if math.Abs(mean-188) > 6 {
		t.Errorf("a half-black, half-white checkerboard averaged to sRGB %.1f. Resampled in linear light it is about 188; %.1f is what averaging the stored sRGB numbers gives, which is the mistake this pipeline exists to avoid", mean, mean)
	}
	if math.Abs(mean-128) < 20 {
		t.Errorf("the average came out at %.1f, which is the gamma-space answer", mean)
	}
}

// Transparency becomes white, not a dark fringe.
//
// The fringe is what a naive flatten produces: compositing the stored numbers
// rather than the light they stand for leaves a half-transparent white pixel
// at sRGB 128 instead of 188, and a transparent PNG dropped into a JPEG picks
// up a grey halo around everything.
func TestTransparencyIsFlattenedOntoWhiteWithNoFringe(t *testing.T) {
	const size = 1024
	art := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			switch {
			case x < size/3:
				art.SetNRGBA(x, y, color.NRGBA{A: 0}) // wholly transparent
			case x < 2*size/3:
				art.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 128}) // half-transparent white
			default:
				art.SetNRGBA(x, y, color.NRGBA{R: 20, G: 30, B: 40, A: 255})
			}
		}
	}

	linear, flattened := toLinear(art)
	if !flattened {
		t.Fatal("flattening was not reported for artwork that is a third transparent")
	}
	flat := finish(linear, false).(*image.RGBA)

	clear := flat.RGBAAt(size/6, size/2)
	if clear.R != 255 || clear.G != 255 || clear.B != 255 || clear.A != 255 {
		t.Errorf("a wholly transparent pixel came out %v; it should be opaque white", clear)
	}

	half := flat.RGBAAt(size/2, size/2)
	if half.R < 250 {
		t.Errorf("half-transparent white over white came out at %d; white over white is white whatever the alpha", half.R)
	}

	opaque := flat.RGBAAt(5*size/6, size/2)
	if opaque.A != 255 {
		t.Errorf("an opaque pixel lost its alpha: %v", opaque)
	}
}

// Half-transparent BLACK over white is where the two flattening methods differ
// most, and where the halo comes from.
func TestHalfTransparentBlackOverWhiteIsNotGammaGrey(t *testing.T) {
	art := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			art.SetNRGBA(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: 128})
		}
	}
	linear, _ := toLinear(art)
	flat := finish(linear, false).(*image.RGBA)
	got := int(flat.RGBAAt(4, 4).R)
	// Half the light of white is sRGB 188. Half the stored number is 128.
	if got < 182 || got > 194 {
		t.Errorf("black at 50%% alpha over white came out at sRGB %d; composited in linear light it is about 188 (128 is the gamma-space answer)", got)
	}
}

// The two TIFFs a print designer sends that x/image cannot read.
//
// Neither is broken and neither is the author's mistake; both are ordinary
// print files. What matters is that the message says what to do, once, in the
// same words for both, rather than repeating the decoder's "color model" or
// "compression value 7".
func TestPrintTIFFsAskForAnRGBExport(t *testing.T) {
	dir := t.TempDir()
	art := paintCover(1600, 2560, coverPaint{grainAmount: 0.03})

	cases := []struct {
		name string
		path string
	}{
		{"CMYK TIFF", writePatchedTIFF(t, dir, "cmyk.tif", art, tiff.Uncompressed, tagPhotometric, photometricCMYK)},
		{"JPEG-compressed TIFF", writePatchedTIFF(t, dir, "jpegtiff.tif", art, tiff.Deflate, tagCompression, compressionJPEG)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Prepare(tc.path, Options{})
			if err == nil {
				t.Fatal("this TIFF was accepted; x/image cannot decode it")
			}
			msg := err.Error()
			for _, want := range []string{"Re-export the cover as an RGB TIFF", "not CMYK TIFFs and not JPEG-compressed ones"} {
				if !strings.Contains(msg, want) {
					t.Errorf("the message does not say %q.\ngot: %s", want, msg)
				}
			}
			if strings.Contains(msg, "unsupported feature") {
				t.Errorf("the decoder's own wording reached the author: %s", msg)
			}
		})
	}
}

// CMYK that Go CAN read.
//
// image/jpeg decodes CMYK and YCCK through the Adobe APP14 marker and keeps
// working, so a CMYK JPEG is not a failure - it is a conversion, done without
// a colour profile because pure-Go colour management does not exist. The
// author is told, because the picture changed.
func TestCMYKArtworkConvertsAndSaysSo(t *testing.T) {
	const w, h = 1200, 1920
	art := image.NewCMYK(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := art.PixOffset(x, y)
			// A flat swatch of one ink in the corner, so the conversion can be
			// checked against a pixel a codec has no reason to move, and
			// textured ink everywhere else so the artwork behaves like
			// artwork rather than like a gradient.
			if x < 32 && y < 32 {
				art.Pix[i], art.Pix[i+1], art.Pix[i+2], art.Pix[i+3] = 180, 90, 40, 20
				continue
			}
			n := grain(x, y, 3) * 40
			art.Pix[i] = clampInk(float64(200*x/w) + n)
			art.Pix[i+1] = clampInk(float64(120*y/h) - n)
			art.Pix[i+2] = clampInk(40 + n*0.5)
			art.Pix[i+3] = clampInk(float64(60*y/h) + n*0.3)
		}
	}

	res, err := prepareDecoded(art, Probe{Width: w, Height: h, Format: "jpeg"}, Options{})
	if err != nil {
		t.Fatalf("preparing CMYK artwork: %v", err)
	}
	if !res.ConvertedFromCMYK {
		t.Fatal("the conversion out of CMYK was not reported")
	}
	if !containsNote(res.Notes, CMYKNote) {
		t.Errorf("the CMYK note is not among the notes shown to the author: %v", res.Notes)
	}
	if !strings.Contains(CMYKNote, "no colour profile") {
		t.Error("the note has to say the conversion was made without a profile; that is the whole point of showing it")
	}
	if res.Cover.Bytes == 0 || res.Cover.Encoding != "jpeg" {
		t.Errorf("CMYK artwork should still produce a cover; got %d bytes as %s", res.Cover.Bytes, res.Cover.Encoding)
	}

	// The conversion is the naive one in image/color, and the result has to
	// match it exactly: anything else would mean this pipeline invented a
	// colour transform of its own.
	cover := decodeDerivative(t, res.Cover)
	ink := art.CMYKAt(2, 2)
	wantR, wantG, wantB := color.CMYKToRGB(ink.C, ink.M, ink.Y, ink.K)
	gotR, gotG, gotB, _ := cover.At(2, 2).RGBA()
	if absDiff(int(gotR>>8), int(wantR)) > 12 || absDiff(int(gotG>>8), int(wantG)) > 12 || absDiff(int(gotB>>8), int(wantB)) > 12 {
		t.Errorf("the top-left corner came out (%d,%d,%d); the standard CMYK conversion of that ink is (%d,%d,%d)",
			gotR>>8, gotG>>8, gotB>>8, wantR, wantG, wantB)
	}
}

// A huge image is refused by its header, before anything is allocated.
//
// The file here is a valid PNG header with no image data behind it, which is
// exactly what the guard is supposed to be able to answer: a 12000 x 12000
// scan is 144 megapixels, and a 16-bit one decodes to well over a gigabyte.
// The memory assertion is the real one - refusing after the allocation is not
// refusing at all.
func TestAnEnormousImageIsRefusedAtTheHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scan.png")
	writeFile(t, path, pngHeaderOnly(12000, 12000))

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	_, err := Prepare(path, Options{})

	runtime.ReadMemStats(&after)

	if err == nil {
		t.Fatal("a 12000 x 12000 image was accepted")
	}
	if !strings.Contains(err.Error(), "megapixels") {
		t.Errorf("the refusal does not explain the size: %v", err)
	}
	allocated := after.TotalAlloc - before.TotalAlloc
	// A whole 12000 x 12000 image is at least 576 MB. Anything under a few
	// megabytes means nothing of the sort was allocated.
	if allocated > 8<<20 {
		t.Errorf("refusing the image allocated %d bytes; the check is supposed to happen before the pixels are read", allocated)
	}
}

// Artwork below the retailer floor is refused rather than enlarged.
func TestArtworkBelowTheFloorIsRefusedNotEnlarged(t *testing.T) {
	dir := t.TempDir()
	path := writePNG(t, dir, "small.png", paintCover(400, 640, coverPaint{}))
	_, err := Prepare(path, Options{})
	if err == nil {
		t.Fatal("a 400 x 640 cover was accepted")
	}
	for _, want := range []string{"at least 625 x 1000", "will not enlarge"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %q: %v", want, err)
		}
	}
}

// Artwork that is already inside the box is not resampled at all, because
// resampling 1:1 with any kernel is a blur and buys nothing.
func TestArtworkAlreadyInsideTheBoxIsNotResampled(t *testing.T) {
	art := paintCover(800, 1280, coverPaint{grainAmount: 0.04})
	linear, _ := toLinear(art)
	out := resample(linear, CoverBox)
	if out != linear {
		t.Error("artwork smaller than the box was put through the resampler")
	}
	if w, h := Fit(800, 1280, CoverBox); w != 800 || h != 1280 {
		t.Errorf("Fit enlarged 800x1280 to %dx%d", w, h)
	}
}

func TestFitKeepsTheShapeAndNeverLeavesTheBox(t *testing.T) {
	cases := []struct{ w, h, wantW, wantH int }{
		{4000, 6400, 1600, 2560}, // exactly the box's shape
		{4000, 4000, 1600, 1600}, // square: width-limited
		{3000, 6000, 1280, 2560}, // tall: height-limited
		{6000, 3000, 1600, 800},  // wide: width-limited
		{1200, 1920, 1200, 1920}, // already inside
	}
	for _, tc := range cases {
		w, h := Fit(tc.w, tc.h, CoverBox)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("Fit(%d, %d) = %dx%d, wanted %dx%d", tc.w, tc.h, w, h, tc.wantW, tc.wantH)
		}
		if w > CoverBox.W || h > CoverBox.H {
			t.Errorf("Fit(%d, %d) left the box at %dx%d", tc.w, tc.h, w, h)
		}
	}
}

// A folder, an empty file and a file that is not an image at all.
func TestWhatIsNotArtworkIsRefusedByName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "empty.png"), nil)
	writeFile(t, filepath.Join(dir, "notes.txt"), []byte("chapter one"))
	if err := os.Mkdir(filepath.Join(dir, "covers"), 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct{ path, want string }{
		{filepath.Join(dir, "covers"), "is a folder"},
		{filepath.Join(dir, "empty.png"), "is empty"},
		{filepath.Join(dir, "notes.txt"), "JPEG, PNG, TIFF, WebP or BMP"},
		{filepath.Join(dir, "gone.png"), "no file at"},
	}
	for _, tc := range cases {
		_, err := Inspect(tc.path)
		if err == nil {
			t.Errorf("%s was accepted", tc.path)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: wanted a message saying %q, got %v", filepath.Base(tc.path), tc.want, err)
		}
	}
}

// Artwork with no colour in it is encoded as greyscale, which roughly halves
// the file for a picture that has lost nothing.
func TestMonochromeArtworkIsEncodedGrey(t *testing.T) {
	dir := t.TempDir()
	grey := paintCover(2000, 3200, coverPaint{grainAmount: 0.05, monochrome: true})
	colour := paintCover(2000, 3200, coverPaint{grainAmount: 0.05})

	greyRes, err := Prepare(writePNG(t, dir, "grey.png", grey), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !greyRes.Greyscale {
		t.Fatal("artwork with no colour was not recognised as monochrome")
	}
	if !containsSubstring(greyRes.Notes, "no colour at all") {
		t.Errorf("the author is not told the cover was encoded in grey: %v", greyRes.Notes)
	}
	if got := decodeDerivative(t, greyRes.Cover); got.ColorModel() != color.GrayModel && got.ColorModel() != color.YCbCrModel {
		t.Errorf("the encoded cover is %v, not a greyscale image", got.ColorModel())
	}

	colourRes, err := Prepare(writePNG(t, dir, "colour.png", colour), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if colourRes.Greyscale {
		t.Error("coloured artwork was flattened to grey")
	}
}

// One coloured element on an otherwise grey cover keeps the cover in colour.
// The monochrome test looks at every pixel for exactly this reason.
func TestOneColouredMarkKeepsTheCoverInColour(t *testing.T) {
	art := paintCover(700, 1120, coverPaint{monochrome: true})
	art.SetNRGBA(400, 900, color.NRGBA{R: 220, G: 30, B: 30, A: 255})
	linear, _ := toLinear(art)
	if isMonochrome(linear) {
		t.Error("a cover with a red mark on it was called monochrome")
	}
}

// Flat artwork keeps a PNG when the PNG is smaller, which is the case JPEG is
// worst at: hard edges and saturated type under 4:2:0 subsampling.
func TestFlatArtworkKeepsAPNGWhenItIsSmaller(t *testing.T) {
	dir := t.TempDir()
	poster := paintPoster(1600, 2560)
	res, err := Prepare(writePNG(t, dir, "poster.png", poster), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Cover.Encoding != "png" {
		t.Fatalf("a flat two-colour cover was kept as %s at %d bytes; PNG should have won", res.Cover.Encoding, res.Cover.Bytes)
	}
	if !strings.HasSuffix(res.Cover.Name, ".png") {
		t.Errorf("the derivative is a PNG but is named %q", res.Cover.Name)
	}
	if res.Cover.Quality != 0 {
		t.Errorf("a PNG reported JPEG quality %d", res.Cover.Quality)
	}
	// The flatness gate is what keeps the PNG encode off every other cover, so
	// it is asserted in both directions on the images the encoder actually
	// sees: 8-bit sRGB, after the resample.
	if !looksFlat(finishFor(poster)) {
		t.Error("the flatness test did not recognise flat artwork")
	}
	if looksFlat(finishFor(paintCover(700, 1120, coverPaint{grainAmount: 0.05}))) {
		t.Error("the flatness test called grainy artwork flat, which would cost a wasted PNG encode on every cover")
	}
}

// The search is the point: harder artwork gets more bytes, easier artwork gets
// fewer, and neither is a number anybody typed in.
func TestQualityIsSearchedNotFixed(t *testing.T) {
	dir := t.TempDir()
	hard, err := Prepare(writePNG(t, dir, "hard.png", paintCover(2000, 3200, coverPaint{grainAmount: 0.07})), Options{})
	if err != nil {
		t.Fatal(err)
	}
	easy, err := Prepare(writePNG(t, dir, "easy.png", paintCover(2000, 3200, coverPaint{grainAmount: 0.004})), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if hard.Cover.Quality <= easy.Cover.Quality {
		t.Errorf("grainy artwork settled at quality %d and smooth artwork at %d; the search is not distinguishing them",
			hard.Cover.Quality, easy.Cover.Quality)
	}
	if hard.Cover.Bytes <= easy.Cover.Bytes {
		t.Errorf("grainy artwork came out at %d bytes and smooth artwork at %d", hard.Cover.Bytes, easy.Cover.Bytes)
	}
	if !onLadder(hard.Cover.Quality) || !onLadder(easy.Cover.Quality) {
		t.Errorf("the search returned qualities %d and %d, which are not on the ladder %v",
			hard.Cover.Quality, easy.Cover.Quality, qualityLadder)
	}
}

// The larger derivative is opt-in, and it is genuinely larger.
func TestTheKoboDerivativeIsOptIn(t *testing.T) {
	dir := t.TempDir()
	path := writePNG(t, dir, "art.png", paintCover(3000, 4800, coverPaint{grainAmount: 0.05}))

	plain, err := Prepare(path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if plain.Large != nil {
		t.Fatal("the 2400-pixel derivative was produced without being asked for; it costs the author a delivery charge on every sale")
	}

	big, err := Prepare(path, Options{Large: true})
	if err != nil {
		t.Fatal(err)
	}
	if big.Large == nil {
		t.Fatal("the larger derivative was asked for and not produced")
	}
	if big.Large.Width != 2400 || big.Large.Height != 3840 {
		t.Errorf("the larger derivative is %dx%d, wanted 2400x3840", big.Large.Width, big.Large.Height)
	}
	if big.Large.Bytes <= big.Cover.Bytes {
		t.Errorf("the larger derivative is %d bytes against the cover's %d", big.Large.Bytes, big.Cover.Bytes)
	}
}

// Artwork a long way off 1.6:1 is accepted and mentioned, never cropped.
func TestOddlyShapedArtworkIsMentionedAndNotCropped(t *testing.T) {
	dir := t.TempDir()
	res, err := Prepare(writePNG(t, dir, "square.png", paintCover(2000, 2000, coverPaint{grainAmount: 0.04})), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsSubstring(res.Notes, "1.6 to 1") {
		t.Errorf("square artwork produced no note about its shape: %v", res.Notes)
	}
	if res.Cover.Width != res.Cover.Height {
		t.Errorf("square artwork came out %dx%d; nothing here crops", res.Cover.Width, res.Cover.Height)
	}
}

// ── The print-ready original ───────────────────────────────────────────────

func TestTheOriginalIsFoundMovedOrChanged(t *testing.T) {
	dir := t.TempDir()
	path := writePNG(t, dir, "wrap.png", paintCover(700, 1120, coverPaint{}))

	print, err := TakeFingerprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if print.Checksum == "" || print.Bytes == 0 {
		t.Fatalf("the fingerprint recorded nothing useful: %+v", print)
	}

	if got := CheckSource(print); got.Status != SourcePresent {
		t.Errorf("an untouched original reported %q: %s", got.Status, got.Message)
	}

	// Changed in place: same name, different bytes.
	writeFile(t, path, append(mustRead(t, path), 0))
	changed := CheckSource(print)
	if changed.Status != SourceChanged {
		t.Errorf("an edited original reported %q", changed.Status)
	}
	if !strings.Contains(changed.Message, "attach it again") {
		t.Errorf("the changed message does not say what to do: %s", changed.Message)
	}

	// Moved: gone from where it was.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	moved := CheckSource(print)
	if moved.Status != SourceMoved {
		t.Errorf("a missing original reported %q", moved.Status)
	}
	for _, want := range []string{"no longer at", "190 dots per inch"} {
		if !strings.Contains(moved.Message, want) {
			t.Errorf("the moved message does not say %q: %s", want, moved.Message)
		}
	}

	if got := CheckSource(Fingerprint{}); got.Status != SourceMoved {
		t.Errorf("a cover with no recorded original reported %q", got.Status)
	}
}

// ── The metric itself ──────────────────────────────────────────────────────

func TestTheMetricScoresAnIdenticalImagePerfectly(t *testing.T) {
	art := finish(func() *image.RGBA64 { l, _ := toLinear(paintCover(320, 512, coverPaint{grainAmount: 0.05})); return l }(), false)
	plane := linearLuma(art)
	if got := msSSIM(plane, plane, 320, 512); math.Abs(got-1) > 1e-9 {
		t.Errorf("an image scored %.9f against itself", got)
	}

	// A visibly damaged copy has to score below the threshold, or the search
	// would accept anything.
	damaged, err := encodeJPEG(art, 12)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(damaged))
	if err != nil {
		t.Fatal(err)
	}
	if got := msSSIM(plane, linearLuma(decoded), 320, 512); got >= qualityThreshold {
		t.Errorf("a quality-12 JPEG scored %.6f, at or above the %.3f threshold", got, qualityThreshold)
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

// paintPoster is flat artwork: a handful of exact colours, no gradient, no
// grain. This is the cover PNG exists for.
func paintPoster(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	ground := color.NRGBA{R: 18, G: 24, B: 38, A: 255}
	band := color.NRGBA{R: 226, G: 84, B: 46, A: 255}
	type_ := color.NRGBA{R: 244, G: 241, B: 232, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := ground
			if y > h/3 && y < h/3+h/9 {
				c = band
			}
			if y > h/2 && y < h/2+h/26 && x > w/8 && x < w*7/8 {
				c = type_
			}
			if y > h*4/5 && y < h*4/5+h/50 && x > w/4 && x < w*3/4 {
				c = type_
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func decodeDerivative(t *testing.T, d Derivative) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(d.Data))
	if err != nil {
		t.Fatalf("the derivative %s does not decode: %v", d.Name, err)
	}
	return img
}

func meanChannel(t *testing.T, img image.Image) float64 {
	t.Helper()
	b := img.Bounds()
	total := 0.0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			total += float64(r >> 8)
		}
	}
	return total / float64(b.Dx()*b.Dy())
}

func containsNote(notes []string, want string) bool {
	for _, note := range notes {
		if note == want {
			return true
		}
	}
	return false
}

func containsSubstring(notes []string, want string) bool {
	for _, note := range notes {
		if strings.Contains(note, want) {
			return true
		}
	}
	return false
}

func onLadder(q int) bool {
	for _, rung := range qualityLadder {
		if rung == q {
			return true
		}
	}
	return q == fallbackQuality
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// pngHeaderOnly builds a PNG that declares a size and carries no pixels.
// image.DecodeConfig reads only the header, which is exactly the property the
// megapixel guard depends on.
func pngHeaderOnly(w, h uint32) []byte {
	var out bytes.Buffer
	out.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})

	var ihdr bytes.Buffer
	ihdr.WriteString("IHDR")
	binary.Write(&ihdr, binary.BigEndian, w)
	binary.Write(&ihdr, binary.BigEndian, h)
	ihdr.Write([]byte{8, 2, 0, 0, 0}) // 8-bit, truecolour, no interlace

	binary.Write(&out, binary.BigEndian, uint32(ihdr.Len()-4))
	out.Write(ihdr.Bytes())
	binary.Write(&out, binary.BigEndian, crc32.ChecksumIEEE(ihdr.Bytes()))
	return out.Bytes()
}

// Keep the png import honest: the header builder above writes PNG by hand, and
// this is the one place the package's own encoder is named in a test that does
// not go through writePNG.
var _ = png.Encode

// finishFor runs artwork through the conversion the encoder sees, without
// resampling it, so a test can ask what the encoder would be handed.
func finishFor(art image.Image) image.Image {
	linear, _ := toLinear(art)
	return finish(linear, false)
}

func clampInk(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
