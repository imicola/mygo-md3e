package md3e

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

func countNonBgPixelsInRect(img *image.RGBA, r ui.Rect, bg color.RGBA) int {
	b := img.Bounds()
	x0 := max(int(math.Floor(float64(r.X))), b.Min.X)
	y0 := max(int(math.Floor(float64(r.Y))), b.Min.Y)
	x1 := min(int(math.Ceil(float64(r.X+r.W))), b.Max.X)
	y1 := min(int(math.Ceil(float64(r.Y+r.H))), b.Max.Y)
	count := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if absColorDiff(img.RGBAAt(x, y), bg) > 8 {
				count++
			}
		}
	}
	return count
}

func countNonBgPixelsOutsideRect(img *image.RGBA, r ui.Rect, margin int, bg color.RGBA) int {
	b := img.Bounds()
	x0 := max(int(math.Floor(float64(r.X)))-margin, b.Min.X)
	y0 := max(int(math.Floor(float64(r.Y)))-margin, b.Min.Y)
	x1 := min(int(math.Ceil(float64(r.X+r.W)))+margin, b.Max.X)
	y1 := min(int(math.Ceil(float64(r.Y+r.H)))+margin, b.Max.Y)

	rx0 := int(math.Floor(float64(r.X))) - 2
	ry0 := int(math.Floor(float64(r.Y))) - 2
	rx1 := int(math.Ceil(float64(r.X+r.W))) + 2
	ry1 := int(math.Ceil(float64(r.Y+r.H))) + 2

	count := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x >= rx0 && x < rx1 && y >= ry0 && y < ry1 {
				continue
			}
			if absColorDiff(img.RGBAAt(x, y), bg) > 8 {
				count++
			}
		}
	}
	return count
}

func TestShapeButtonInteractionVariantsAndDensity(t *testing.T) {
	for _, density := range []tokens.Density{tokens.DensityComfortable, tokens.DensityDesktop, tokens.DensityCompact} {
		th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: density})
		clicks := 0
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(240, 160).Background(canvasRGB).Padding(20).Children(func() {
					ShapeButton(c, "Play", mdicon.Star.Draw, ShapeButtonOptions{
						Variant: ShapeButtonFilled,
						Size:    tokens.SizeM,
						Shape:   shape.Cookie9Sided(),
						OnClick: func() { clicks++ },
					})
				})
			})
		}, 240, 160)

		info, ok := tt.AccessByRole(ui.RoleButton, "Play")
		if !ok {
			t.Fatalf("density=%d: ShapeButton not found in access tree", density)
		}
		wantSide := float32(density.ButtonSpec(tokens.SizeM).Height)
		if math.Abs(float64(info.Bounds.W-wantSide)) > 0.5 || math.Abs(float64(info.Bounds.H-wantSide)) > 0.5 {
			t.Fatalf("density=%d: size=%vx%v want %vx%v", density, info.Bounds.W, info.Bounds.H, wantSide, wantSide)
		}

		cx, cy := center(info.Bounds)
		tt.ClickAt(cx, cy)
		tt.Key(0, ui.KeyTab)
		tt.Key(0, ui.KeyEnter)
		tt.Key(0, ui.KeySpace)
		tt.Advance(800 * time.Millisecond)

		if clicks != 3 {
			t.Fatalf("density=%d: clicks=%d want 3", density, clicks)
		}
		if tt.FramePending() {
			t.Fatalf("density=%d: expected frames to settle", density)
		}
	}
}

func TestShapeButtonDisabledAndPanics(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	clicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(200, 120).Background(canvasRGB).Padding(20).Children(func() {
				ShapeButton(c, "Blocked", mdicon.Add.Draw, ShapeButtonOptions{
					Variant:  ShapeButtonOutlined,
					Size:     tokens.SizeS,
					Shape:    shape.SoftBurst(),
					Disabled: true,
					OnClick:  func() { clicks++ },
				})
			})
		})
	}, 200, 120)

	info, ok := tt.AccessByRole(ui.RoleButton, "Blocked")
	if !ok || !info.Disabled {
		t.Fatalf("disabled ShapeButton access = %+v, ok=%v", info, ok)
	}
	cx, cy := center(info.Bounds)
	tt.ClickAt(cx, cy)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(400 * time.Millisecond)
	if clicks != 0 {
		t.Fatalf("disabled ShapeButton clicked %d times", clicks)
	}

	// Verify panic contracts.
	expectPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s did not panic", name)
			}
		}()
		fn()
	}
	ui.NewTester(func(c *ui.Context) {
		expectPanic("ShapeButton empty label", func() {
			ShapeButton(c, "", mdicon.Add.Draw)
		})
		expectPanic("ShapeButton multiple opts", func() {
			ShapeButton(c, "A", mdicon.Add.Draw, ShapeButtonOptions{}, ShapeButtonOptions{})
		})
		expectPanic("MorphIconButton nil bool", func() {
			MorphIconButton(c, nil, "A", mdicon.Add.Draw)
		})
		expectPanic("MorphIconButton empty label", func() {
			MorphIconButton(c, new(bool), "", mdicon.Add.Draw)
		})
		expectPanic("IconBadge multiple opts", func() {
			IconBadge(c, mdicon.Add.Draw, IconBadgeOptions{}, IconBadgeOptions{})
		})
	}, 100, 100)
}

func TestMorphIconButtonToggleMorphGeometryAndCrossfade(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	playing := false
	changes := 0

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 200).Background(canvasRGB).Padding(40).Children(func() {
				MorphIconButton(c.Key("play"), &playing, "Play or pause", mdicon.Star.Draw, MorphIconButtonOptions{
					Variant:       ShapeButtonFilled,
					Size:          tokens.SizeL,
					Shape:         shape.SoftBurst(),
					SelectedShape: shape.Square(),
					SelectedIcon:  mdicon.Check.Draw,
					OnChanged:     func(on bool) { changes++ },
				})
			})
		})
	}, 240, 200)

	info, ok := tt.AccessByRole(ui.RoleToggleButton, "Play or pause")
	if !ok {
		t.Fatal("MorphIconButton missing RoleToggleButton")
	}
	if info.Checked {
		t.Fatal("MorphIconButton should start unchecked")
	}

	// Stage 0: resting SoftBurst shape.
	img0 := tt.Image()
	inside0 := countNonBgPixelsInRect(img0, info.Bounds, white)
	outside0 := countNonBgPixelsOutsideRect(img0, info.Bounds, 12, white)
	if inside0 < 2000 {
		t.Fatalf("stage 0 painted too few pixels: %d", inside0)
	}
	if outside0 != 0 {
		t.Fatalf("stage 0 bled %d pixels outside bounds", outside0)
	}
	corner0 := cornerFill(img0, info.Bounds, 16)

	// Toggle via pointer click.
	cx, cy := center(info.Bounds)
	tt.ClickAt(cx, cy)
	if !playing || changes != 1 {
		t.Fatalf("click failed: playing=%v changes=%d", playing, changes)
	}
	tt.Move(4, 4)

	// Stage ~0.5: mid-morph frame.
	tt.Advance(60 * time.Millisecond)
	imgMid := tt.Image()
	insideMid := countNonBgPixelsInRect(imgMid, info.Bounds, white)
	outsideMid := countNonBgPixelsOutsideRect(imgMid, info.Bounds, 12, white)
	if insideMid < 2000 {
		t.Fatalf("mid-morph painted too few pixels: %d", insideMid)
	}
	if outsideMid != 0 {
		t.Fatalf("mid-morph bled %d pixels outside bounds", outsideMid)
	}

	// Stage 1: settled Square shape.
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("MorphIconButton did not settle after 960ms")
	}
	info, _ = tt.AccessByRole(ui.RoleToggleButton, "Play or pause")
	if !info.Checked {
		t.Fatal("MorphIconButton should report Checked=true after toggle")
	}
	img1 := tt.Image()
	inside1 := countNonBgPixelsInRect(img1, info.Bounds, white)
	outside1 := countNonBgPixelsOutsideRect(img1, info.Bounds, 12, white)
	corner1 := cornerFill(img1, info.Bounds, 16)
	if inside1 < 2000 || outside1 != 0 {
		t.Fatalf("stage 1 inside=%d outside=%d", inside1, outside1)
	}
	if corner1 <= corner0 {
		t.Fatalf("morph from SoftBurst to Square did not fill corners: corner0=%d corner1=%d", corner0, corner1)
	}

	// Keyboard Enter and Space toggle.
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyEnter)
	if playing {
		t.Fatal("Enter did not toggle MorphIconButton off")
	}
	tt.Key(0, ui.KeySpace)
	if !playing {
		t.Fatal("Space did not toggle MorphIconButton back on")
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("MorphIconButton still pending after keyboard settle")
	}
}

func TestMorphIconButtonReduceMotionAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	on := false
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Row(c).Direction(ui.RightToLeft).Size(240, 140).Background(canvasRGB).Padding(20).Children(func() {
				MorphIconButton(c.Key("rtl-morph"), &on, "Mirror", mdicon.Star.Draw, MorphIconButtonOptions{
					Variant:       ShapeButtonTonal,
					Size:          tokens.SizeM,
					Shape:         shape.Cookie9Sided(),
					SelectedShape: shape.Sunny(),
					SelectedIcon:  mdicon.Check.Draw,
				})
			})
		})
	}, 240, 140)

	info, ok := tt.AccessByRole(ui.RoleToggleButton, "Mirror")
	if !ok {
		t.Fatal("RTL MorphIconButton not found")
	}
	cx, cy := center(info.Bounds)
	tt.ClickAt(cx, cy)
	if !on {
		t.Fatal("RTL click did not toggle")
	}
	tt.Move(2, 2)
	if tt.FramePending() {
		t.Fatal("ReduceMotion MorphIconButton should settle with zero pending frames")
	}
}

func TestIconBadgeGeometryShapesAndSanitization(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Row(c).Size(320, 140).Background(canvasRGB).Padding(20).Gap(16).Children(func() {
				IconBadge(c, mdicon.Star.Draw, IconBadgeOptions{
					Variant:       IconBadgePrimaryTonal,
					ContainerSize: 48,
					IconSize:      24,
					Shape:         shape.Cookie9Sided(),
					Label:         "Cookie Badge",
				})
				IconBadge(c, mdicon.Check.Draw, IconBadgeOptions{
					Variant:       IconBadgeOutlined,
					ContainerSize: nan,
					IconSize:      inf,
					ShapeScale:    tokens.ShapeFull,
					Label:         "Sanitized Badge",
				})
			})
		})
	}, 320, 140)

	cookieInfo, ok := tt.AccessByRole(ui.RoleImage, "Cookie Badge")
	if !ok {
		t.Fatal("Cookie Badge missing RoleImage")
	}
	img := tt.Image()
	if got := countNonBgPixelsInRect(img, cookieInfo.Bounds, white); got < 800 {
		t.Fatalf("Cookie IconBadge painted too few pixels: %d", got)
	}
	if out := countNonBgPixelsOutsideRect(img, cookieInfo.Bounds, 6, white); out != 0 {
		t.Fatalf("Cookie IconBadge bled %d pixels outside bounds", out)
	}

	sanitizedInfo, ok := tt.AccessByRole(ui.RoleImage, "Sanitized Badge")
	if !ok {
		t.Fatal("Sanitized Badge missing RoleImage")
	}
	wantSide := float32(tokens.SizeXS.Spec().Height)
	if math.Abs(float64(sanitizedInfo.Bounds.W-wantSide)) > 0.5 || math.Abs(float64(sanitizedInfo.Bounds.H-wantSide)) > 0.5 {
		t.Fatalf("Sanitized Badge size=%vx%v want %vx%v", sanitizedInfo.Bounds.W, sanitizedInfo.Bounds.H, wantSide, wantSide)
	}
}

func TestShapeButtonGhostDrawExitSafe(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	exitTr := ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}
	show := true
	on := false
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 160).Background(canvasRGB).Padding(20).Children(func() {
				if show {
					MorphIconButton(c.Key("exit-morph"), &on, "Exit Test", mdicon.Star.Draw, MorphIconButtonOptions{
						Size:          tokens.SizeM,
						Shape:         shape.Cookie9Sided(),
						SelectedShape: shape.Circle(),
						SelectedIcon:  mdicon.Check.Draw,
					}).Transition(exitTr)
				}
			})
		})
	}, 240, 160)

	info, ok := tt.AccessByRole(ui.RoleToggleButton, "Exit Test")
	if !ok {
		t.Fatal("Exit Test button missing")
	}
	cx, cy := center(info.Bounds)
	tt.ClickAt(cx, cy)
	show = false
	tt.Frame()
	tt.Advance(50 * time.Millisecond)
	midExit := tt.Image()
	tt.Advance(400 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("exited MorphIconButton left frames pending")
	}
	settled := tt.Image()
	if diff := countNonBgPixelsInRect(midExit, info.Bounds, white) - countNonBgPixelsInRect(settled, info.Bounds, white); diff < 100 {
		t.Fatalf("exiting MorphIconButton ghost did not stay visible during exit (diff=%d)", diff)
	}
}
