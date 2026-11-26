package md3e

import (
	"errors"
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

func makeTestBitmap(w, h int, c color.NRGBA) *ui.Bitmap {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return ui.NewBitmap(img)
}

func makeSplitTestBitmap(w, h int, top, bottom color.NRGBA) *ui.Bitmap {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		c := top
		if y >= h/2 {
			c = bottom
		}
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return ui.NewBitmap(img)
}

func TestImageBitmapSVGAndShapeClipping(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	bmp := makeTestBitmap(40, 20, color.NRGBA{R: 220, G: 60, B: 40, A: 255})
	svg := ui.MustParseSVG([]byte(`<svg width="40" height="40" viewBox="0 0 40 40"><rect width="40" height="40" fill="#2080e0"/></svg>`))
	canvasBG := ui.RGB(245, 245, 245)

	img := ui.Render(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Row(c).Size(320, 120).AlignItems(ui.Start).Background(canvasBG).Padding(16).Gap(16).Children(func() {
				// 1. Default intrinsic bitmap size (40x20).
				Image(c, bmp, ImageOptions{Label: "Red banner"})
				// 2. SVG with ShapeScaleLarge (16 DIP corners).
				Image(c, svg, ImageOptions{
					Width:      64,
					Height:     64,
					Fit:        ui.Cover,
					ShapeScale: tokens.ShapeLarge,
					Label:      "Blue square",
				})
				// 3. Polygon clipped bitmap (Diamond leaves corners as canvasBG).
				Image(c, bmp, ImageOptions{
					Width:  64,
					Height: 64,
					Fit:    ui.Cover,
					Shape:  shape.Diamond(),
					Label:  "Diamond photo",
				})
			})
		})
	}, 320, 120, 1)

	// Bitmap 1 at (16,16)..(56,36): center (36, 26) is red.
	px1 := img.RGBAAt(36, 26)
	if px1.R < 200 || px1.G > 80 {
		t.Fatalf("intrinsic bitmap center pixel = %+v, want red", px1)
	}
	// Just below intrinsic 40x20 at (36, 40) must remain canvas background.
	if pxBelow := img.RGBAAt(36, 40); pxBelow != (color.RGBA{245, 245, 245, 255}) {
		t.Fatalf("intrinsic bitmap leaked below height 20: %+v", pxBelow)
	}

	// SVG 2 starts at X = 16+40+16 = 72, Y = 16, size 64x64: center (104, 48) is blue.
	px2 := img.RGBAAt(104, 48)
	if px2.B < 200 || px2.R > 60 {
		t.Fatalf("SVG center pixel = %+v, want blue", px2)
	}
	// Outer corner (72, 16) clipped by ShapeLarge radius=16 -> canvasBG.
	if px2Corner := img.RGBAAt(72, 16); px2Corner != (color.RGBA{245, 245, 245, 255}) {
		t.Fatalf("SVG ShapeLarge corner (72,16) not clipped: %+v", px2Corner)
	}

	// Diamond 3 starts at X = 72+64+16 = 152, Y = 16, size 64x64: center (184, 48) is red.
	px3 := img.RGBAAt(184, 48)
	if px3.R < 200 || px3.G > 80 {
		t.Fatalf("diamond center pixel = %+v, want red", px3)
	}
	// Corner (154, 18) outside diamond polygon must remain canvasBG.
	if px3Corner := img.RGBAAt(154, 18); px3Corner != (color.RGBA{245, 245, 245, 255}) {
		t.Fatalf("diamond outer corner (154,18) not clipped: %+v", px3Corner)
	}
}

func TestImageFitObjectPositionAndPolygonClip(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	// Tall image (40x120): top half green, bottom half magenta, rendered inside 60x60 box.
	split := makeSplitTestBitmap(40, 120, color.NRGBA{0, 200, 70, 255}, color.NRGBA{210, 40, 200, 255})
	renderAtPos := func(py float32, poly *shape.RoundedPolygon) color.RGBA {
		img := ui.Render(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				Image(c, split, ImageOptions{
					Width:             60,
					Height:            60,
					Fit:               ui.Cover,
					ObjectPosition:    [2]float32{0.5, py},
					HasObjectPosition: true,
					Shape:             poly,
				})
			})
		}, 60, 60, 1)
		return img.RGBAAt(30, 30)
	}

	topPx := renderAtPos(0, nil)
	botPx := renderAtPos(1, nil)
	if topPx.G <= topPx.R+80 || botPx.R <= botPx.G+80 {
		t.Fatalf("ObjectPosition did not shift Cover crop: top=%+v bottom=%+v", topPx, botPx)
	}

	polyTop := renderAtPos(0, shape.Circle())
	polyBot := renderAtPos(1, shape.Circle())
	if polyTop.G <= polyTop.R+80 || polyBot.R <= polyBot.G+80 {
		t.Fatalf("Polygon ObjectPosition did not shift Cover crop: top=%+v bottom=%+v", polyTop, polyBot)
	}
}

func TestImageLoadingFadeInErrorRetryAndClick(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	bmp := makeTestBitmap(48, 48, color.NRGBA{R: 40, G: 180, B: 90, A: 255})
	loading := true
	failed := false
	retried := 0
	clicked := 0

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(200, 200).Background(ui.RGB(255, 255, 255)).Padding(16).Children(func() {
				var src ui.ImageSource
				if !loading && !failed {
					src = bmp
				}
				Image(c.Key("photo"), src, ImageOptions{
					Width:   120,
					Height:  120,
					Loading: loading,
					Failed:  failed,
					OnRetry: func() {
						retried++
						failed = false
						loading = false
					},
					OnClick: func() { clicked++ },
					Label:   "Cover art",
				})
			})
		})
	}, 200, 200)

	// Loading state animates skeleton pulse while visible.
	tt.Advance(200 * time.Millisecond)
	if !tt.FramePending() {
		t.Fatal("expected skeleton pulse to request animation frames while loading")
	}

	// Transition Loading -> Loaded triggers smooth fade-in then settles.
	loading = false
	tt.Frame()
	if !tt.FramePending() {
		t.Fatal("expected smooth fade-in transition after loading completes")
	}
	tt.Advance(500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("fade-in did not settle")
	}

	px := tt.Image().RGBAAt(76, 76)
	if px.G < 160 || px.R > 80 {
		t.Fatalf("loaded image center pixel = %+v, want green", px)
	}

	// Click & keyboard activation on loaded image.
	btnInfo, ok := tt.AccessByRole(ui.RoleButton, "Cover art")
	if !ok {
		t.Fatal("clickable image missing RoleButton accessibility node")
	}
	cx, cy := btnInfo.Bounds.X+btnInfo.Bounds.W/2, btnInfo.Bounds.Y+btnInfo.Bounds.H/2
	tt.ClickAt(cx, cy)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(400 * time.Millisecond)
	if clicked != 2 {
		t.Fatalf("clicked = %d, want 2 (click + Enter)", clicked)
	}

	// Error state shows Retry button and invokes OnRetry.
	failed = true
	tt.Frame()
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("click Retry: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if retried != 1 {
		t.Fatalf("retried = %d, want 1", retried)
	}
	if tt.FramePending() {
		t.Fatal("image did not settle after retry recovery")
	}
}

func TestImageReduceMotionAndNaNSafety(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	bmp := makeTestBitmap(40, 40, color.NRGBA{R: 100, G: 50, B: 200, A: 255})
	nan := float32(math.NaN())

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(200, 200).Gap(8).Children(func() {
				Image(c.Key("fade"), bmp, ImageOptions{
					Width:             64,
					Height:            64,
					FadeIn:            true,
					Radius:            nan,
					HasRadius:         true,
					ObjectPosition:    [2]float32{nan, nan},
					HasObjectPosition: true,
					AspectRatio:       nan,
					BorderWidth:       nan,
					Label:             "Reduced image",
				})
				Image(c.Key("skel-poly"), nil, ImageOptions{
					Width:   64,
					Height:  64,
					Loading: true,
					Shape:   shape.Cookie9Sided(),
				})
				Image(c.Key("err"), nil, ImageOptions{
					Width: 64, Height: 64, Error: errors.New("boom"),
				})
			})
		})
	}, 200, 200)

	if tt.FramePending() {
		t.Fatal("ReduceMotion image must settle immediately with zero pending frames")
	}
	if _, ok := tt.AccessByRole(ui.RoleImage, "Reduced image"); !ok {
		t.Fatal("non-interactive image missing RoleImage accessibility node")
	}
}

func TestImageExplicitAspectRatioWithFillWidth(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	bmp := makeTestBitmap(256, 256, color.NRGBA{R: 80, G: 140, B: 220, A: 255})

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(200, 320).Padding(16).Children(func() {
				Image(c.Key("square"), bmp, ImageOptions{
					Fit:         ui.Cover,
					AspectRatio: 1,
					Label:       "Responsive square",
				}).FillWidth()
			})
		})
	}, 200, 320)

	tt.Frame()
	info, ok := tt.AccessByRole(ui.RoleImage, "Responsive square")
	if !ok {
		t.Fatal("Responsive square image missing")
	}
	// Available width inside 200px column with 16px padding on both sides is 168px.
	if math.Abs(float64(info.Bounds.W-168)) > 1 || math.Abs(float64(info.Bounds.H-168)) > 1 {
		t.Fatalf("explicit AspectRatio=1 image bounds = %+v, want ~168x168 (not 256px intrinsic height)", info.Bounds)
	}
}

func TestImagePanicsOnMultipleOptions(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on multiple ImageOptions")
		}
	}()
	_ = ui.NewTester(func(c *ui.Context) {
		Image(c, nil, ImageOptions{}, ImageOptions{})
	}, 100, 100)
}
