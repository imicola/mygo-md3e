package tokens

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func solidImageNRGBA(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

type countedImage struct {
	image.Image
	bounds image.Rectangle
	calls  int
}

func (img *countedImage) Bounds() image.Rectangle { return img.bounds }

func (img *countedImage) At(x, y int) color.Color {
	if !image.Pt(x, y).In(img.bounds) {
		panic("sample outside image bounds")
	}
	img.calls++
	return img.Image.At(x, y)
}

func TestImageSamplingRespectsBudget(t *testing.T) {
	const limit = 16384
	ink := color.NRGBA{R: 120, G: 40, B: 200, A: 255}
	for _, bounds := range []image.Rectangle{
		image.Rect(0, 0, 1, 1000000),
		image.Rect(-11, 7, 1000000-11, 10),
		image.Rect(11, 20, 11+1024, 20+1024),
		image.Rect(math.MaxInt-1000000, -32, math.MaxInt, -31),
	} {
		img := &countedImage{Image: image.NewUniform(ink), bounds: bounds}
		samples := sampleImagePixels(img)
		if len(samples) != 1 || samples[0].count != img.calls || img.calls == 0 || img.calls > limit {
			t.Errorf("%v: visited %d pixels and got %v, want 1..%d visits of the same color", bounds, img.calls, samples, limit)
		}
		if got, want := SeedFromImage(img), SeedFromImage(img); got != want {
			t.Errorf("%v: non-deterministic seed %v, %v", bounds, got, want)
		}
	}
	// The direct NRGBA path uses the same grid and respects nonzero origins.
	raw := image.NewNRGBA(image.Rect(-8, 3, -7, 100003))
	for y := raw.Rect.Min.Y; y < raw.Rect.Max.Y; y++ {
		raw.SetNRGBA(-8, y, ink)
	}
	if got := sampleImagePixels(raw); len(got) != 1 || got[0].count > limit {
		t.Errorf("NRGBA fast path exceeded its sampling budget: %v", got)
	}
}

func TestSeedFromImageNilEmptyTransparentMonochrome(t *testing.T) {
	if got := SeedFromImage(nil); got != defaultExtractSeed {
		t.Fatalf("nil image seed = %v, want %v", got, defaultExtractSeed)
	}
	empty := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	if got := SeedFromImage(empty); got != defaultExtractSeed {
		t.Fatalf("empty image seed = %v, want %v", got, defaultExtractSeed)
	}

	customFallback := ui.RGB(12, 120, 210)
	transparent := solidImageNRGBA(32, 32, color.NRGBA{R: 255, G: 0, B: 0, A: 0})
	if got := SeedFromImage(transparent, ExtractOptions{Fallback: customFallback}); got != customFallback {
		t.Fatalf("transparent image seed = %v, want custom fallback %v", got, customFallback)
	}

	gray := solidImageNRGBA(40, 40, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	if got := SeedFromImage(gray); got != defaultExtractSeed {
		t.Fatalf("grayscale image seed = %v, want fallback %v", got, defaultExtractSeed)
	}

	gotLow := SeedFromImage(gray, ExtractOptions{AllowLowChroma: true})
	if math.Abs(float64(gotLow.R)-128) > 2 || math.Abs(float64(gotLow.G)-128) > 2 || math.Abs(float64(gotLow.B)-128) > 2 {
		t.Fatalf("AllowLowChroma grayscale seed = %v, want ~gray(128)", gotLow)
	}
}

func TestSeedFromImageDominantAndVibrantExtraction(t *testing.T) {
	// Mostly deep teal background with a small coral accent: Material Score
	// balances proportion (0.7) and chroma above 48, returning a clean chromatic
	// hue and surfacing both hues in PaletteFromImage.
	img := image.NewNRGBA(image.Rect(10, 20, 110, 120)) // non-zero Min rect
	teal := color.NRGBA{R: 0, G: 128, B: 128, A: 255}
	coral := color.NRGBA{R: 235, G: 75, B: 65, A: 255}
	for y := 20; y < 120; y++ {
		for x := 10; x < 110; x++ {
			c := teal
			if x >= 80 && y >= 90 {
				c = coral
			}
			img.SetNRGBA(x, y, c)
		}
	}

	seed := SeedFromImage(img)
	hct := HCTFromColor(seed)
	if hct.Chroma < 20 {
		t.Fatalf("extracted seed %v has unexpectedly low chroma %.2f", seed, hct.Chroma)
	}

	pal := PaletteFromImage(img, ExtractOptions{Desired: 3})
	if len(pal) < 2 {
		t.Fatalf("PaletteFromImage returned %d colors, want >= 2 diverse colors", len(pal))
	}
	h0 := HCTFromColor(pal[0]).Hue
	h1 := HCTFromColor(pal[1]).Hue
	if diff := hueDifferenceDegrees(h0, h1); diff < 15 {
		t.Fatalf("top 2 palette colors lack hue diversity: h0=%.1f h1=%.1f (diff=%.1f)", h0, h1, diff)
	}
}

func TestSeedFromImageFormatsAndLargeDownsampling(t *testing.T) {
	// Premultiplied *image.RGBA at 50% alpha with bright blue (0, 80, 220) ->
	// must unpremultiply rather than extracting darkened color, and skip
	// alpha < 128 pixels.
	rgba := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			if x < 32 {
				// Alpha 200 (<255 but >=128) premultiplied color ~(0, 63, 173, 200).
				rgba.SetRGBA(x, y, color.RGBA{R: 0, G: 63, B: 173, A: 200})
			} else {
				// Nearly transparent red noise must be ignored.
				rgba.SetRGBA(x, y, color.RGBA{R: 25, G: 0, B: 0, A: 25})
			}
		}
	}
	seedRGBA := SeedFromImage(rgba)
	if seedRGBA.B <= seedRGBA.R+80 {
		t.Fatalf("RGBA extraction failed to pick blue over low-alpha red: %+v", seedRGBA)
	}

	// *image.Paletted fast path.
	palImg := image.NewPaletted(image.Rect(0, 0, 48, 48), color.Palette{
		color.NRGBA{R: 34, G: 170, B: 72, A: 255},
		color.NRGBA{R: 20, G: 20, B: 20, A: 255},
	})
	for y := range 48 {
		for x := range 48 {
			if y < 36 {
				palImg.SetColorIndex(x, y, 0)
			} else {
				palImg.SetColorIndex(x, y, 1)
			}
		}
	}
	seedPal := SeedFromImage(palImg)
	if seedPal.G <= seedPal.R+40 || seedPal.G <= seedPal.B+40 {
		t.Fatalf("Paletted extraction did not pick emerald green: %+v", seedPal)
	}

	// *image.YCbCr fast path (warm orange).
	ycc := image.NewYCbCr(image.Rect(0, 0, 64, 64), image.YCbCrSubsampleRatio420)
	vy, vcb, vcr := color.RGBToYCbCr(230, 120, 20)
	for i := range ycc.Y {
		ycc.Y[i] = vy
	}
	for i := range ycc.Cb {
		ycc.Cb[i] = vcb
		ycc.Cr[i] = vcr
	}
	seedYCC := SeedFromImage(ycc)
	if seedYCC.R <= seedYCC.B+100 {
		t.Fatalf("YCbCr extraction did not pick warm orange: %+v", seedYCC)
	}

	// Large image downsampling (1024x1024) must remain deterministic across runs.
	large := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	for y := range 1024 {
		for x := range 1024 {
			if x < 512 {
				large.SetNRGBA(x, y, color.NRGBA{R: 120, G: 40, B: 200, A: 255})
			} else {
				large.SetNRGBA(x, y, color.NRGBA{R: 240, G: 170, B: 30, A: 255})
			}
		}
	}
	s1 := SeedFromImage(large)
	s2 := SeedFromImage(large)
	if s1 != s2 {
		t.Fatalf("large image extraction non-deterministic: %v vs %v", s1, s2)
	}
}

func TestSeedFromImageInvalidOptionsPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when passing multiple ExtractOptions")
		}
	}()
	_ = SeedFromImage(nil, ExtractOptions{}, ExtractOptions{})
}
