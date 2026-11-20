package md3e

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

var (
	buttonVariants = []struct {
		name string
		v    ButtonVariant
	}{
		{"Filled", ButtonFilled},
		{"Tonal", ButtonTonal},
		{"Elevated", ButtonElevated},
		{"Outlined", ButtonOutlined},
		{"Text", ButtonText},
	}
	buttonSizes = []tokens.SizeTier{tokens.SizeXS, tokens.SizeS, tokens.SizeM, tokens.SizeL, tokens.SizeXL}
)

var (
	white     = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	canvasRGB = ui.RGB(255, 255, 255)
)

// TestButtonVariantsAndSizesLayout checks that every variant and size tier
// renders at exactly the tier height, centers its label in the box, and keeps
// its whole box clickable as its minimum touch target.
func TestButtonVariantsAndSizesLayout(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	for _, variant := range buttonVariants {
		for _, size := range buttonSizes {
			name := "Save"
			accName := name + " button"
			tt := ui.NewTester(func(c *ui.Context) {
				ThemeScope(c, th, func() {
					ui.Box(c).Size(480, 200).Background(canvasRGB).Padding(16).Children(func() {
						Button(c, name, ButtonOptions{Variant: variant.v, Size: size, Label: accName})
					})
				})
			}, 480, 200)

			info, ok := tt.AccessByRole(ui.RoleButton, accName)
			if !ok {
				t.Fatalf("%s/%d: button not found in the access tree", variant.name, size)
			}
			spec := size.Spec()
			if info.Bounds.H != float32(spec.Height) {
				t.Fatalf("%s/%d: height = %v, want %v", variant.name, size, info.Bounds.H, spec.Height)
			}
			if info.Bounds.W < info.Bounds.H {
				t.Fatalf("%s/%d: width %v narrower than touch target height %v",
					variant.name, size, info.Bounds.W, info.Bounds.H)
			}
			if info.Disabled || !info.Focusable {
				t.Fatalf("%s/%d: enabled button reports disabled=%v focusable=%v",
					variant.name, size, info.Disabled, info.Focusable)
			}

			labelRect, ok := tt.Find(name)
			if !ok {
				t.Fatalf("%s/%d: label not laid out", variant.name, size)
			}
			// The typography follows the tier's role: the line box is at least
			// the role size and fits inside the button.
			wantSize := float32(th.TypeStyle(buttonTypeRole(size), false).Size)
			if labelRect.H < wantSize*0.9 || labelRect.H > info.Bounds.H {
				t.Fatalf("%s/%d: label height %v does not follow the %v type size",
					variant.name, size, labelRect.H, wantSize)
			}
			wantCX := info.Bounds.X + info.Bounds.W/2
			wantCY := info.Bounds.Y + info.Bounds.H/2
			if gotCX := labelRect.X + labelRect.W/2; math.Abs(float64(gotCX-wantCX)) > 1 {
				t.Fatalf("%s/%d: label center x = %v, want %v", variant.name, size, gotCX, wantCX)
			}
			if gotCY := labelRect.Y + labelRect.H/2; math.Abs(float64(gotCY-wantCY)) > 1 {
				t.Fatalf("%s/%d: label center y = %v, want %v", variant.name, size, gotCY, wantCY)
			}

			// The whole box is the click target: a click on its last row counts.
			clicks := 0
			tt2 := ui.NewTester(func(c *ui.Context) {
				ThemeScope(c, th, func() {
					ui.Box(c).Size(480, 200).Background(canvasRGB).Padding(16).Children(func() {
						b := Button(c, name, ButtonOptions{Variant: variant.v, Size: size})
						if b.Clicked() {
							clicks++
						}
					})
				})
			}, 480, 200)
			edgeX := info.Bounds.X + info.Bounds.W/2
			edgeY := info.Bounds.Y + info.Bounds.H - 1
			tt2.ClickAt(edgeX, edgeY)
			if clicks != 1 {
				t.Fatalf("%s/%d: click on the last row clicked %d times, want 1", variant.name, size, clicks)
			}
			if tt.FramePending() {
				t.Fatalf("%s/%d: idle button leaves frames pending", variant.name, size)
			}
		}
	}
}

// TestButtonIconsGrowGeometry checks the tier icon sizes and the icon-label
// gap: a both-sided button widens by the icons and gaps, and the icons paint
// inside their slots.
func TestButtonIconsGrowGeometry(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	for _, size := range buttonSizes {
		spec := size.Spec()
		plain := 0.0
		withIcons := ui.Rect{}
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(640, 260).Background(canvasRGB).Padding(12).AlignItems(ui.Start).Children(func() {
					Button(c, "Go", ButtonOptions{Size: size, Label: "Plain"})
					Button(c, "Go", ButtonOptions{
						Size:         size,
						Label:        "Both",
						LeadingIcon:  mdicon.Check.Draw,
						TrailingIcon: mdicon.ChevronDown.Draw,
					})
				})
			})
		}, 640, 260)

		plainInfo, _ := tt.AccessByRole(ui.RoleButton, "Plain")
		bothInfo, _ := tt.AccessByRole(ui.RoleButton, "Both")
		plain = float64(plainInfo.Bounds.W)
		withIcons = bothInfo.Bounds
		wantGrowth := float64(2*spec.IconSize + 2*float64(iconLabelGap(size)))
		if got := float64(withIcons.W) - plain; math.Abs(got-wantGrowth) > 0.5 {
			t.Fatalf("%d: both-icon width grew %v, want %v", size, got, wantGrowth)
		}

		// The leading icon slot paints OnPrimary ink over the Primary
		// container: scan the whole slot, as a large check mark's middle is
		// not necessarily inked.
		container := rgbOf(th.Colors.Primary)
		left := int(withIcons.X + float32(spec.Padding))
		top := int(withIcons.Y + (withIcons.H-float32(spec.IconSize))/2)
		side := int(spec.IconSize)
		found := false
		for dy := 0; dy < side && !found; dy++ {
			for dx := 0; dx < side; dx++ {
				if absColorDiff(tt.Image().RGBAAt(left+dx, top+dy), container) > 30 {
					found = true
					break
				}
			}
		}
		if !found {
			t.Fatalf("%d: leading icon did not paint OnPrimary ink inside its %v slot", size, spec.IconSize)
		}
	}
}

// TestButtonAncestorDisabledPalette checks that a button inside a disabled
// container takes MD3E's disabled palette instead of looking enabled while it
// ignores input.
func TestButtonAncestorDisabledPalette(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	clicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(320, 120).Background(canvasRGB).Padding(20).
				Disabled(true).DimDisabled(false).Children(func() {
				if Button(c, "Blocked", ButtonOptions{Size: tokens.SizeS}).Clicked() {
					clicks++
				}
			})
		})
	}, 320, 120)

	info := buttonInfo(t, tt, ui.RoleButton, "Blocked")
	if !info.Disabled {
		t.Fatal("the button inside a disabled container does not report itself disabled")
	}
	on := th.Colors.OnSurface
	a := float32(on.A) / 255 * float32(tokens.DisabledContainerOpacity)
	want := color.RGBA{
		R: uint8(a*float32(on.R) + (1-a)*255 + 0.5),
		G: uint8(a*float32(on.G) + (1-a)*255 + 0.5),
		B: uint8(a*float32(on.B) + (1-a)*255 + 0.5),
		A: 255,
	}
	got := tt.Image().RGBAAt(int(info.Bounds.X+info.Bounds.W*0.15), int(info.Bounds.Y+info.Bounds.H/2))
	if absColorDiff(got, want) > 4 {
		t.Fatalf("ancestor-disabled container = %v, want the OnSurface 12%% %v", got, want)
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if clicks != 0 {
		t.Fatalf("the ancestor-disabled button handled %d clicks", clicks)
	}
}

// TestButtonClickKeyboardAndDisabled checks pointer clicks, Enter and Space
// activation, and that a disabled button ignores all of them and reports
// itself disabled to assistive technology.
func TestButtonClickKeyboardAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	for _, disabled := range []bool{false, true} {
		clicks := 0
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(320, 160).Background(canvasRGB).Padding(20).Children(func() {
					b := Button(c, "Run", ButtonOptions{Size: tokens.SizeS, Disabled: disabled})
					if b.Clicked() {
						clicks++
					}
				})
			})
		}, 320, 160)

		info, ok := tt.AccessByRole(ui.RoleButton, "Run")
		if !ok {
			t.Fatalf("disabled=%v: button not in access tree", disabled)
		}
		if info.Disabled != disabled {
			t.Fatalf("disabled=%v: access Disabled = %v", disabled, info.Disabled)
		}

		cx := info.Bounds.X + info.Bounds.W/2
		cy := info.Bounds.Y + info.Bounds.H/2
		tt.Move(cx, cy)
		tt.Press(cx, cy)
		tt.Release(cx, cy)
		tt.Advance(600 * time.Millisecond)

		tt.Key(0, ui.KeyTab)
		tt.Key(0, ui.KeyEnter)
		tt.Key(0, ui.KeySpace)
		tt.Advance(600 * time.Millisecond)

		want := 3
		if disabled {
			want = 0
		}
		if clicks != want {
			t.Fatalf("disabled=%v: clicked %d times, want %d", disabled, clicks, want)
		}
		if tt.FramePending() {
			t.Fatalf("disabled=%v: frames still pending after activation", disabled)
		}
	}
}

// TestButtonDisabledColorsNotDimmed checks that the disabled container is
// exactly OnSurface at 12% over the surface, not ui's extra 0.5 dimming.
func TestButtonDisabledColorsNotDimmed(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(320, 120).Background(canvasRGB).Padding(20).Children(func() {
				Button(c, "Off", ButtonOptions{Variant: ButtonFilled, Size: tokens.SizeS, Disabled: true})
			})
		})
	}, 320, 120)

	info, ok := tt.AccessByRole(ui.RoleButton, "Off")
	if !ok {
		t.Fatal("disabled button not in access tree")
	}
	// Sample inside the pill, left of the label.
	x := int(info.Bounds.X + info.Bounds.W*0.15)
	y := int(info.Bounds.Y + info.Bounds.H/2)
	got := tt.Image().RGBAAt(x, y)

	on := th.Colors.OnSurface
	a := float32(on.A) / 255 * float32(tokens.DisabledContainerOpacity)
	want := color.RGBA{
		R: uint8(a*float32(on.R) + (1-a)*255 + 0.5),
		G: uint8(a*float32(on.G) + (1-a)*255 + 0.5),
		B: uint8(a*float32(on.B) + (1-a)*255 + 0.5),
		A: 255,
	}
	if absColorDiff(got, want) > 3 {
		t.Fatalf("disabled container pixel = %+v, want %+v (OnSurface at 12%% over white)", got, want)
	}
	// The half-opacity dimming ui applies by default would be much lighter.
	dimmed := color.RGBA{
		R: uint8(a/2*float32(on.R) + (1-a/2)*255 + 0.5),
		G: uint8(a/2*float32(on.G) + (1-a/2)*255 + 0.5),
		B: uint8(a/2*float32(on.B) + (1-a/2)*255 + 0.5),
		A: 255,
	}
	if absColorDiff(got, dimmed) <= 3 {
		t.Fatalf("disabled container pixel = %+v matches double-dimmed %+v", got, dimmed)
	}
}

// TestButtonPressedCornerMorphAndReduceMotion checks the Fast Spatial corner
// morph: pressing shrinks the pill's corner toward the tier's pressed radius,
// releasing springs back, both settle with no pending frames, and ReduceMotion
// snaps without asking for animation frames.
func TestButtonPressedCornerMorphAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(320, 160).Background(canvasRGB).Padding(20).Children(func() {
				Button(c, "Morph", ButtonOptions{Variant: ButtonFilled, Size: tokens.SizeXS})
			})
		})
	}, 320, 160)

	info, ok := tt.AccessByRole(ui.RoleButton, "Morph")
	if !ok {
		t.Fatal("button not in access tree")
	}
	probeX := int(info.Bounds.X + 3)
	probeY := int(info.Bounds.Y + 3)
	corner := func() color.RGBA { return tt.Image().RGBAAt(probeX, probeY) }

	if got := corner(); got != white {
		t.Fatalf("idle pill corner pixel = %+v, want white outside the full radius", got)
	}

	cx := info.Bounds.X + info.Bounds.W/2
	cy := info.Bounds.Y + info.Bounds.H/2
	tt.Move(cx, cy)
	tt.Press(cx, cy)
	if !tt.FramePending() {
		t.Fatal("pressing the button should start the corner spring")
	}
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("pressed corner spring did not settle")
	}
	if got := corner(); got == white {
		t.Fatalf("pressed corner pixel = %+v, want the shrunken corner to fill it", got)
	}

	tt.Release(cx, cy)
	tt.Move(5, 5)
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("released corner spring did not settle")
	}
	if got := corner(); got != white {
		t.Fatalf("released corner pixel = %+v, want the pill restored", got)
	}

	// ReduceMotion: the corner snaps, with no animation frames at all.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Move(cx, cy)
	tt.Press(cx, cy)
	if tt.FramePending() {
		t.Fatal("ReduceMotion press should not ask for animation frames")
	}
	if got := corner(); got == white {
		t.Fatalf("ReduceMotion press corner pixel = %+v, want the pressed radius at once", got)
	}
	tt.Release(cx, cy)
	tt.Move(5, 5)
	if tt.FramePending() {
		t.Fatal("ReduceMotion release should not ask for animation frames")
	}
	if got := corner(); got != white {
		t.Fatalf("ReduceMotion release corner pixel = %+v, want the pill at once", got)
	}
}

// TestButtonElevatedShadows checks that the Elevated variant casts Level1
// shadows below its box, lifting to Level2 while pressed, while the Filled
// variant casts none.
func TestButtonElevatedShadows(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(320, 200).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ui.Row(c).Gap(40).Children(func() {
					Button(c, "Flat", ButtonOptions{Variant: ButtonFilled, Size: tokens.SizeS})
					Button(c, "Lift", ButtonOptions{Variant: ButtonElevated, Size: tokens.SizeS})
				})
			})
		})
	}, 320, 200)

	flat, _ := tt.AccessByRole(ui.RoleButton, "Flat")
	lift, _ := tt.AccessByRole(ui.RoleButton, "Lift")
	// Shadow darkness below the box, over the four rows under its bottom
	// edge: the Level1 shadows fade there, and the pressed Level2 shadows
	// spread further out.
	below := func(b ui.Rect) (px color.RGBA, dark int) {
		x := int(b.X + b.W/2)
		for dy := 1; dy <= 4; dy++ {
			p := tt.Image().RGBAAt(x, int(b.Y+b.H)+dy)
			if dy == 1 {
				px = p
			}
			dark += 255 - int(p.R)
		}
		return px, dark
	}
	if px, dark := below(flat.Bounds); dark != 0 {
		t.Fatalf("filled button cast a shadow: pixel below = %+v, darkness %d", px, dark)
	}
	idlePx, idleDark := below(lift.Bounds)
	if idleDark == 0 {
		t.Fatal("elevated button cast no Level1 shadow below its box")
	}

	cx := lift.Bounds.X + lift.Bounds.W/2
	cy := lift.Bounds.Y + lift.Bounds.H/2
	tt.Move(cx, cy)
	tt.Press(cx, cy)
	tt.Advance(700 * time.Millisecond)
	pressedPx, pressedDark := below(lift.Bounds)
	if pressedDark == 0 {
		t.Fatalf("elevated button cast no shadow while pressed: %+v", pressedPx)
	}
	if pressedDark <= idleDark {
		t.Fatalf("pressed shadow darkness %d not stronger than the idle %d (%+v vs %+v)",
			pressedDark, idleDark, pressedPx, idlePx)
	}
	tt.Release(cx, cy)
	tt.Move(5, 5)
	tt.Advance(800 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("elevated button did not settle after release")
	}
}

func TestButtonOutOfRangeSizeTierNormalizesToXS(t *testing.T) {
	if role := buttonTypeRole(tokens.SizeTier(99)); role != tokens.LabelLarge {
		t.Fatalf("buttonTypeRole(99) = %v, want LabelLarge", role)
	}
	if role := buttonTypeRole(tokens.SizeTier(-1)); role != tokens.LabelLarge {
		t.Fatalf("buttonTypeRole(-1) = %v, want LabelLarge", role)
	}
	if gap := iconLabelGap(tokens.SizeTier(99)); gap != 8 {
		t.Fatalf("iconLabelGap(99) = %v, want 8", gap)
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, nil, func() {
			ui.Column(c).Size(240, 160).Padding(16).Gap(8).Children(func() {
				Button(c, "Same", ButtonOptions{Size: tokens.SizeXS, Label: "XSRef"})
				Button(c, "Same", ButtonOptions{Size: tokens.SizeTier(99), Label: "InvalidBig"})
				Button(c, "Same", ButtonOptions{Size: tokens.SizeTier(-1), Label: "InvalidNeg"})
			})
		})
	}, 240, 160)

	xs := buttonInfo(t, tt, ui.RoleButton, "XSRef")
	big := buttonInfo(t, tt, ui.RoleButton, "InvalidBig")
	neg := buttonInfo(t, tt, ui.RoleButton, "InvalidNeg")
	if big.Bounds.H != xs.Bounds.H || neg.Bounds.H != xs.Bounds.H {
		t.Fatalf("invalid tier heights (%v, %v) != SizeXS (%v)", big.Bounds.H, neg.Bounds.H, xs.Bounds.H)
	}
	if big.Bounds.W != xs.Bounds.W || neg.Bounds.W != xs.Bounds.W {
		t.Fatalf("invalid tier widths (%v, %v) != SizeXS (%v) — label typography mismatch", big.Bounds.W, neg.Bounds.W, xs.Bounds.W)
	}
}
