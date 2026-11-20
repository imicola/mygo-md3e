package md3e

import (
	"fmt"
	"image"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

var iconButtonWidths = []struct {
	name  string
	width IconButtonWidth
}{
	{"Narrow", IconButtonWidthNarrow},
	{"Default", IconButtonWidthDefault},
	{"Wide", IconButtonWidthWide},
}

// cornerFill counts the pixels in the span×span box at b's top-left corner
// that differ from bg: a circle leaves the corner mostly empty where the
// tier's square radius fills it.
func cornerFill(img *image.RGBA, b ui.Rect, span int) int {
	n := 0
	for y := 0; y < span; y++ {
		for x := 0; x < span; x++ {
			if absColorDiff(img.RGBAAt(int(b.X)+x, int(b.Y)+y), white) > 12 {
				n++
			}
		}
	}
	return n
}

// TestIconButtonWidthsAndSizes checks the exact container geometry of every
// width mode at every size tier, and that the icon paints inside it.
func TestIconButtonWidthsAndSizes(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	for _, size := range buttonSizes {
		for _, mode := range iconButtonWidths {
			label := fmt.Sprintf("%s %d", mode.name, size)
			tt := ui.NewTester(func(c *ui.Context) {
				ThemeScope(c, th, func() {
					ui.Box(c).Size(480, 240).Background(canvasRGB).Padding(16).Children(func() {
						IconButton(c, label, mdicon.Add.Draw, IconButtonOptions{
							Variant: IconButtonStandard,
							Size:    size,
							Width:   mode.width,
						})
					})
				})
			}, 480, 240)

			info, ok := tt.AccessByRole(ui.RoleButton, label)
			if !ok {
				t.Fatalf("%s/%d: icon button not in the access tree", mode.name, size)
			}
			spec := size.Spec()
			wantH := float32(spec.Height)
			var wantW float32
			switch mode.width {
			case IconButtonWidthNarrow:
				wantW = wantH * 0.75
			case IconButtonWidthWide:
				wantW = wantH * 1.25
			default:
				wantW = wantH
			}
			if math.Abs(float64(info.Bounds.W-wantW)) > 0.01 || math.Abs(float64(info.Bounds.H-wantH)) > 0.01 {
				t.Fatalf("%s/%d: size = %vx%v, want %vx%v",
					mode.name, size, info.Bounds.W, info.Bounds.H, wantW, wantH)
			}
			if info.Disabled || !info.Focusable {
				t.Fatalf("%s/%d: enabled icon button reports disabled=%v focusable=%v",
					mode.name, size, info.Disabled, info.Focusable)
			}

			// The icon paints around the center of the standard (transparent)
			// button.
			cx := int(info.Bounds.X + info.Bounds.W/2)
			cy := int(info.Bounds.Y + info.Bounds.H/2)
			painted := false
			for dy := -2; dy <= 2 && !painted; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if tt.Image().RGBAAt(cx+dx, cy+dy) != white {
						painted = true
						break
					}
				}
			}
			if !painted {
				t.Fatalf("%s/%d: icon did not paint at the center", mode.name, size)
			}
		}
	}
}

// TestIconButtonClickKeyboardAndDisabled checks pointer and keyboard
// activation and the required accessible label, enabled and disabled.
func TestIconButtonClickKeyboardAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	for _, disabled := range []bool{false, true} {
		clicks := 0
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Box(c).Size(240, 160).Background(canvasRGB).Padding(20).Children(func() {
					b := IconButton(c, "Search", mdicon.Search.Draw, IconButtonOptions{
						Size:     tokens.SizeS,
						Disabled: disabled,
					})
					if b.Clicked() {
						clicks++
					}
				})
			})
		}, 240, 160)

		info, ok := tt.AccessByRole(ui.RoleButton, "Search")
		if !ok {
			t.Fatalf("disabled=%v: icon button not in the access tree", disabled)
		}
		if info.Disabled != disabled {
			t.Fatalf("disabled=%v: access Disabled = %v", disabled, info.Disabled)
		}
		if info.Name != "Search" {
			t.Fatalf("disabled=%v: accessible name = %q, want Search", disabled, info.Name)
		}

		cx := info.Bounds.X + info.Bounds.W/2
		cy := info.Bounds.Y + info.Bounds.H/2
		tt.Move(cx, cy)
		tt.ClickAt(cx, cy)
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
			t.Fatalf("disabled=%v: frames still pending", disabled)
		}
	}
}

// TestToggleIconButtonMorphAndState checks that the toggle flips *on with the
// pointer and with Space, that its round resting shape morphs into the square
// selected one and back, settling both ways, and that assistive technology
// reads its checked state.
func TestToggleIconButtonMorphAndState(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	on := false
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 160).Background(canvasRGB).Padding(20).Children(func() {
				ToggleIconButton(c, &on, "Favorite", mdicon.Star.Draw, ToggleIconButtonOptions{
					Variant:      IconButtonFilled,
					Size:         tokens.SizeXS,
					SelectedIcon: mdicon.Check.Draw,
				})
			})
		})
	}, 240, 160)

	info, ok := tt.AccessByRole(ui.RoleToggleButton, "Favorite")
	if !ok {
		t.Fatal("toggle icon button not in the access tree")
	}
	if info.Checked {
		t.Fatal("toggle starts checked")
	}
	span := int(tokens.SizeXS.Spec().SquareRadius)
	roundFill := cornerFill(tt.Image(), info.Bounds, span)
	if roundFill == 0 {
		t.Fatal("unselected filled toggle left its whole corner empty")
	}

	cx := info.Bounds.X + info.Bounds.W/2
	cy := info.Bounds.Y + info.Bounds.H/2
	tt.Move(cx, cy)
	tt.ClickAt(cx, cy)
	if !on {
		t.Fatal("click did not flip *on")
	}
	if !tt.FramePending() {
		t.Fatal("selecting should start the corner spring")
	}
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("selected corner spring did not settle")
	}
	info, _ = tt.AccessByRole(ui.RoleToggleButton, "Favorite")
	if !info.Checked {
		t.Fatal("assistive technology does not see the toggle checked")
	}
	squareFill := cornerFill(tt.Image(), info.Bounds, span)
	if squareFill-roundFill < span*span/10 {
		t.Fatalf("selected corner fill %d barely exceeds the round %d: no circle to square morph",
			squareFill, roundFill)
	}

	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	if on {
		t.Fatal("Space did not flip *on back off")
	}
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("deselected corner spring did not settle")
	}
	info, _ = tt.AccessByRole(ui.RoleToggleButton, "Favorite")
	if info.Checked {
		t.Fatal("assistive technology still reads the toggle checked")
	}
	if got := cornerFill(tt.Image(), info.Bounds, span); got >= squareFill-span*span/10 {
		t.Fatalf("deselected corner fill %d did not return toward the round %d", got, roundFill)
	}
}

// TestToggleIconButtonDisabledAndReduceMotion checks that a disabled toggle
// ignores the pointer and stays round, and that ReduceMotion snaps the shape
// without asking for animation frames.
func TestToggleIconButtonDisabledAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	// Disabled.
	on := false
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 160).Background(canvasRGB).Padding(20).Children(func() {
				ToggleIconButton(c, &on, "Lock", mdicon.Check.Draw, ToggleIconButtonOptions{
					Variant:  IconButtonTonal,
					Size:     tokens.SizeS,
					Disabled: true,
				})
			})
		})
	}, 240, 160)
	info, ok := tt.AccessByRole(ui.RoleToggleButton, "Lock")
	if !ok {
		t.Fatal("disabled toggle not in the access tree")
	}
	if !info.Disabled {
		t.Fatal("disabled toggle does not report itself disabled")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	tt.Advance(600 * time.Millisecond)
	if on {
		t.Fatal("disabled toggle flipped *on")
	}
	if tt.FramePending() {
		t.Fatal("disabled toggle left frames pending")
	}

	// ReduceMotion.
	on = false
	rm := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(240, 160).Background(canvasRGB).Padding(20).Children(func() {
				ToggleIconButton(c, &on, "Mute", mdicon.Minus.Draw, ToggleIconButtonOptions{
					Variant: IconButtonOutlined,
					Size:    tokens.SizeXS,
				})
			})
		})
	}, 240, 160)
	rm.SetPreferences(ui.Preferences{ReduceMotion: true})
	info, ok = rm.AccessByRole(ui.RoleToggleButton, "Mute")
	if !ok {
		t.Fatal("ReduceMotion toggle not in the access tree")
	}
	probeSpan := int(tokens.SizeXS.Spec().SquareRadius)
	idleFill := cornerFill(rm.Image(), info.Bounds, probeSpan)
	rm.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if !on {
		t.Fatal("ReduceMotion click did not flip *on")
	}
	if rm.FramePending() {
		t.Fatal("ReduceMotion toggle should not ask for animation frames")
	}
	onFill := cornerFill(rm.Image(), info.Bounds, probeSpan)
	if onFill-idleFill < probeSpan*probeSpan/10 {
		t.Fatalf("ReduceMotion selected corner fill %d barely exceeds the unselected %d", onFill, idleFill)
	}
	rm.Key(0, ui.KeyTab)
	rm.Key(0, ui.KeySpace)
	if on {
		t.Fatal("ReduceMotion Space did not flip *on back off")
	}
	if rm.FramePending() {
		t.Fatal("ReduceMotion toggle release should not ask for animation frames")
	}
	if got := cornerFill(rm.Image(), info.Bounds, probeSpan); got >= onFill-probeSpan*probeSpan/10 {
		t.Fatalf("ReduceMotion deselected corner fill %d did not return toward the round %d", got, idleFill)
	}
}
