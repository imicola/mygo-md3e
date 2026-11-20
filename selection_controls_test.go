package md3e

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/tokens"
)

// colorMatches reports whether a rendered pixel is within tol of col.
func colorMatches(px color.RGBA, col ui.Color, tol int) bool {
	return absColorDiff(px, rgbOf(col)) <= tol
}

// countRun counts the pixels of the row y between x0 and x1 (exclusive) that
// match col, as a thumb's width along its center line.
func countRun(img *image.RGBA, y, x0, x1 int, col ui.Color, tol int) int {
	n := 0
	for x := x0; x < x1; x++ {
		if colorMatches(img.RGBAAt(x, y), col, tol) {
			n++
		}
	}
	return n
}

// switchTrackRect returns the 52x32 track inside a switch row's access bounds.
func switchTrackRect(b ui.Rect, rtl bool) ui.Rect {
	x := b.X
	if rtl {
		x = b.X + b.W - switchTrackWidth
	}
	return ui.Rect{
		X: x,
		Y: b.Y + (b.H-switchTrackHeight)/2,
		W: switchTrackWidth,
		H: switchTrackHeight,
	}
}

// thumbWidthOnRow measures the widest run of col across the track center row,
// the thumb's diameter after its spring settled.
func thumbWidthOnRow(t *testing.T, tt *ui.Tester, track ui.Rect, col ui.Color) int {
	t.Helper()
	img := tt.Image()
	y := int(track.Y + track.H/2)
	best := 0
	for dy := -3; dy <= 3; dy++ {
		if n := countRun(img, y+dy, int(track.X), int(track.X+track.W), col, 30); n > best {
			best = n
		}
	}
	return best
}

// selectionCanvas builds a 320x200 white canvas holding build's controls.
func selectionCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(320, 200).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				build(c)
			})
		})
	}, 320, 200)
}

// TestSwitchClickSpaceAndThumbGeometry checks the switch's 16/24/28 DIP thumb
// growth, its click and Space toggling, and that the spring settles each way.
func TestSwitchClickSpaceAndThumbGeometry(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	on := false
	tt := selectionCanvas(th, func(c *ui.Context) {
		Switch(c, &on, SwitchOptions{Label: "Wi-Fi"})
	})

	info := buttonInfo(t, tt, ui.RoleSwitch, "Wi-Fi")
	if info.Checked {
		t.Fatal("switch starts on")
	}
	track := switchTrackRect(info.Bounds, false)
	offWidth := thumbWidthOnRow(t, tt, track, th.Colors.Outline)
	if offWidth < 12 || offWidth > 20 {
		t.Fatalf("off thumb width %d, want about 16", offWidth)
	}

	cx := track.X + track.W/2
	cy := track.Y + track.H/2
	tt.Move(cx, cy)
	tt.ClickAt(cx, cy)
	if !on {
		t.Fatal("click did not turn the switch on")
	}
	if !tt.FramePending() {
		t.Fatal("turning on should start the thumb spring")
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("thumb spring did not settle after turning on")
	}
	info, _ = tt.AccessByRole(ui.RoleSwitch, "Wi-Fi")
	if !info.Checked {
		t.Fatal("assistive technology does not see the switch on")
	}
	onWidth := thumbWidthOnRow(t, tt, track, th.Colors.OnPrimary)
	if onWidth < offWidth+4 {
		t.Fatalf("on thumb width %d barely exceeds the off %d", onWidth, offWidth)
	}

	// Pressed: the handle grows to 28 DIP and springs back on release.
	tt.Press(cx, cy)
	tt.Advance(500 * time.Millisecond)
	pressedWidth := thumbWidthOnRow(t, tt, track, th.Colors.OnPrimary)
	if pressedWidth < onWidth+2 || pressedWidth > 31 {
		t.Fatalf("pressed thumb width %d, want about 28 after the on %d", pressedWidth, onWidth)
	}
	tt.Release(cx, cy)
	if on {
		t.Fatal("releasing a press on the switch did not toggle it off")
	}
	tt.Advance(900 * time.Millisecond)
	if got := thumbWidthOnRow(t, tt, track, th.Colors.Outline); got < 12 || got > 20 {
		t.Fatalf("released thumb width %d, want back about 16", got)
	}

	// Click it on again, then Space toggles while it keeps the keyboard focus.
	tt.ClickAt(cx, cy)
	if !on {
		t.Fatal("second click did not turn the switch on")
	}
	tt.Advance(900 * time.Millisecond)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	if on {
		t.Fatal("Space did not turn the switch off")
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("thumb spring did not settle after turning off")
	}
	info, _ = tt.AccessByRole(ui.RoleSwitch, "Wi-Fi")
	if info.Checked {
		t.Fatal("assistive technology still sees the switch on")
	}
	if got := thumbWidthOnRow(t, tt, track, th.Colors.Outline); got < 12 || got > 20 {
		t.Fatalf("thumb did not shrink back off: width %d", got)
	}
}

// TestSwitchRTLMirror checks that the off handle sits at the track's right end
// and the on handle at its left end when the window reads right to left.
func TestSwitchRTLMirror(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	on := false
	tt := selectionCanvas(th, func(c *ui.Context) {
		Switch(c, &on, SwitchOptions{Label: "Wi-Fi"})
	})
	tt.SetDirection(ui.RightToLeft)

	info := buttonInfo(t, tt, ui.RoleSwitch, "Wi-Fi")
	track := switchTrackRect(info.Bounds, true)
	img := tt.Image()
	y := int(track.Y + track.H/2)
	offCenter := thumbCenterOnRow(img, y, track, th.Colors.Outline)
	if offCenter < track.X+track.W/2 {
		t.Fatalf("RTL off thumb center %v is not at the track's right end", offCenter)
	}

	cx := track.X + track.W/2
	tt.ClickAt(cx, track.Y+track.H/2)
	if !on {
		t.Fatal("RTL switch click did not turn on")
	}
	tt.Advance(900 * time.Millisecond)
	onCenter := thumbCenterOnRow(tt.Image(), y, track, th.Colors.OnPrimary)
	if onCenter > track.X+track.W/2 {
		t.Fatalf("RTL on thumb center %v is not at the track's left end", onCenter)
	}
}

// thumbCenterOnRow returns the middle of the widest run of col along the row y
// inside track, the handle rather than the track's thin border, or the track's
// middle when no pixel matches.
func thumbCenterOnRow(img *image.RGBA, y int, track ui.Rect, col ui.Color) float32 {
	bestFirst, bestLast := -1, -1
	first := -1
	for x := int(track.X); x < int(track.X+track.W); x++ {
		if colorMatches(img.RGBAAt(x, y), col, 30) {
			if first < 0 {
				first = x
			}
			if x-first > bestLast-bestFirst {
				bestFirst, bestLast = first, x
			}
		} else {
			first = -1
		}
	}
	if bestFirst < 0 {
		return track.X + track.W/2
	}
	return float32(bestFirst+bestLast) / 2
}

// TestSwitchDisabledAndReduceMotion checks that a disabled switch ignores the
// pointer, Space and focus with MD3E's disabled colors, and that ReduceMotion
// snaps the thumb without leaving frames pending.
func TestSwitchDisabledAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)

	on := false
	disabled := selectionCanvas(th, func(c *ui.Context) {
		Switch(c, &on, SwitchOptions{Label: "Wi-Fi", Disabled: true})
	})
	info := buttonInfo(t, disabled, ui.RoleSwitch, "Wi-Fi")
	if !info.Disabled {
		t.Fatal("disabled switch does not report itself disabled")
	}
	track := switchTrackRect(info.Bounds, false)
	// The track fill is OnSurface at 12%, not a 0.5 dim of an enabled color.
	want := th.Colors.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(ui.RGB(255, 255, 255))
	px := disabled.Image().RGBAAt(int(track.X+track.W-8), int(track.Y+track.H/2))
	if !colorMatches(px, want, 12) {
		t.Fatalf("disabled track pixel %+v, want MD3E disabled container %+v", px, rgbOf(want))
	}
	disabled.ClickAt(track.X+track.W/2, track.Y+track.H/2)
	disabled.Key(0, ui.KeyTab)
	disabled.Key(0, ui.KeySpace)
	disabled.Advance(600 * time.Millisecond)
	if on {
		t.Fatal("disabled switch changed its value")
	}
	if disabled.FramePending() {
		t.Fatal("disabled switch left frames pending")
	}

	on = false
	rm := selectionCanvas(th, func(c *ui.Context) {
		Switch(c, &on, SwitchOptions{Label: "Wi-Fi"})
	})
	rm.SetPreferences(ui.Preferences{ReduceMotion: true})
	info = buttonInfo(t, rm, ui.RoleSwitch, "Wi-Fi")
	track = switchTrackRect(info.Bounds, false)
	rm.ClickAt(track.X+track.W/2, track.Y+track.H/2)
	if !on {
		t.Fatal("ReduceMotion click did not turn the switch on")
	}
	if rm.FramePending() {
		t.Fatal("ReduceMotion switch should not ask for frames")
	}
	if got := thumbWidthOnRow(t, rm, track, th.Colors.OnPrimary); got < 21 {
		t.Fatalf("ReduceMotion thumb width %d, want the settled 24", got)
	}
}

// checkboxSlotCenter returns the center of the 40 DIP touch target where a
// check box or radio button draws its box.
func checkboxSlotCenter(b ui.Rect, rtl bool, target float32) (float32, float32) {
	x := b.X + target/2
	if rtl {
		x = b.X + b.W - target/2
	}
	return x, b.Y + b.H/2
}

// TestCheckboxStatesMixedAndError checks the off, on and mixed boxes, the
// check mark's ink, the mixed dash, the error palette and the invalid state.
func TestCheckboxStatesMixedAndError(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	checked := false
	selectAll := ui.CheckMixed
	terms := false
	tt := selectionCanvas(th, func(c *ui.Context) {
		Checkbox(c, &checked, CheckboxOptions{Label: "Sync"})
		CheckboxState(c, &selectAll, CheckboxOptions{Label: "Select all"})
		Checkbox(c, &terms, CheckboxOptions{Label: "Terms", Error: true, ErrorMessage: "Accept the terms"})
	})

	sync := buttonInfo(t, tt, ui.RoleCheckBox, "Sync")
	if sync.Checked || sync.Mixed {
		t.Fatal("Sync starts checked or mixed")
	}
	all := buttonInfo(t, tt, ui.RoleCheckBox, "Select all")
	if !all.Mixed {
		t.Fatal("Select all does not report itself mixed")
	}
	errInfo := buttonInfo(t, tt, ui.RoleCheckBox, "Terms")
	if !errInfo.Invalid || !strings.Contains(errInfo.Description, "Accept the terms") {
		t.Fatalf("error check box invalid=%v description=%q", errInfo.Invalid, errInfo.Description)
	}

	// Off: no fill, an OnSurfaceVariant border at the box edge.
	sx, sy := checkboxSlotCenter(sync.Bounds, false, checkboxTouchTarget)
	img := tt.Image()
	if px := img.RGBAAt(int(sx), int(sy)); !colorMatches(px, ui.RGB(255, 255, 255), 6) {
		t.Fatalf("unchecked box center %+v is not the canvas", px)
	}
	if px := img.RGBAAt(int(sx-checkboxBoxSize/2), int(sy)); !colorMatches(px, cs.OnSurfaceVariant, 40) {
		t.Fatalf("unchecked box border %+v is not OnSurfaceVariant", px)
	}

	// On: Primary fill and an OnPrimary check mark.
	tt.ClickAt(sx, sy)
	if !checked {
		t.Fatal("click did not check Sync")
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("check spring did not settle")
	}
	sync, _ = tt.AccessByRole(ui.RoleCheckBox, "Sync")
	if !sync.Checked {
		t.Fatal("assistive technology does not see Sync checked")
	}
	sx, sy = checkboxSlotCenter(sync.Bounds, false, checkboxTouchTarget)
	img = tt.Image()
	box := ui.Rect{X: sx - checkboxBoxSize/2, Y: sy - checkboxBoxSize/2, W: checkboxBoxSize, H: checkboxBoxSize}
	if px := img.RGBAAt(int(box.X+4), int(box.Y+4)); !colorMatches(px, cs.Primary, 25) {
		t.Fatalf("checked box fill %+v is not Primary", px)
	}
	ink := 0
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if colorMatches(img.RGBAAt(x, y), cs.OnPrimary, 24) {
				ink++
			}
		}
	}
	if ink < 6 {
		t.Fatalf("check mark inked %d pixels, want a visible trim stroke", ink)
	}

	// Mixed: a dash through the middle of a Primary box.
	ax, ay := checkboxSlotCenter(all.Bounds, false, checkboxTouchTarget)
	if px := tt.Image().RGBAAt(int(ax), int(ay)); !colorMatches(px, cs.OnPrimary, 24) {
		t.Fatalf("mixed dash center %+v is not OnPrimary", px)
	}
	if px := tt.Image().RGBAAt(int(ax-checkboxBoxSize/2+4), int(ay-5)); !colorMatches(px, cs.Primary, 25) {
		t.Fatalf("mixed box fill %+v is not Primary", px)
	}

	// Clicking the mixed box checks it, replacing the dash with the check.
	tt.ClickAt(ax, ay)
	if selectAll != ui.CheckOn {
		t.Fatalf("mixed box click set %v, want CheckOn", selectAll)
	}
	tt.Advance(900 * time.Millisecond)

	// Error: the unchecked box outlines in Error.
	ex, ey := checkboxSlotCenter(errInfo.Bounds, false, checkboxTouchTarget)
	if px := tt.Image().RGBAAt(int(ex-checkboxBoxSize/2), int(ey)); !colorMatches(px, cs.Error, 40) {
		t.Fatalf("error box border %+v is not Error", px)
	}
}

// TestCheckboxDisabled checks that a disabled check box ignores input and uses
// the 12% container and 38% content of MD3E's disabled tokens.
func TestCheckboxDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	checked := true
	tt := selectionCanvas(th, func(c *ui.Context) {
		Checkbox(c, &checked, CheckboxOptions{Label: "Sync", Disabled: true})
	})
	info := buttonInfo(t, tt, ui.RoleCheckBox, "Sync")
	if !info.Disabled || !info.Checked {
		t.Fatalf("disabled checked box reports disabled=%v checked=%v", info.Disabled, info.Checked)
	}
	cx, cy := checkboxSlotCenter(info.Bounds, false, checkboxTouchTarget)
	box := ui.Rect{X: cx - checkboxBoxSize/2, Y: cy - checkboxBoxSize/2, W: checkboxBoxSize, H: checkboxBoxSize}
	want := cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(ui.RGB(255, 255, 255))
	if px := tt.Image().RGBAAt(int(box.X+4), int(box.Y+4)); !colorMatches(px, want, 12) {
		t.Fatalf("disabled checked fill %+v, want %+v", px, rgbOf(want))
	}
	tt.ClickAt(cx, cy)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeySpace)
	tt.Advance(600 * time.Millisecond)
	if !checked {
		t.Fatal("disabled check box changed its value")
	}
	if tt.FramePending() {
		t.Fatal("disabled check box left frames pending")
	}
}

// TestRadioSelectedDotAndDisabledPalette checks the ring and dot pixels and the
// disabled palette.
func TestRadioSelectedDotAndDisabledPalette(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	choice := "B"
	tt := selectionCanvas(th, func(c *ui.Context) {
		Radio(c, &choice, "A", "Alpha")
		Radio(c, &choice, "B", "Beta")
	})
	a := buttonInfo(t, tt, ui.RoleRadio, "Alpha")
	b := buttonInfo(t, tt, ui.RoleRadio, "Beta")
	ax, ay := checkboxSlotCenter(a.Bounds, false, radioTouchTarget)
	bx, by := checkboxSlotCenter(b.Bounds, false, radioTouchTarget)
	if px := tt.Image().RGBAAt(int(ax), int(ay)); !colorMatches(px, ui.RGB(255, 255, 255), 6) {
		t.Fatalf("unselected radio center %+v is not empty", px)
	}
	if px := tt.Image().RGBAAt(int(bx), int(by)); !colorMatches(px, cs.Primary, 30) {
		t.Fatalf("selected radio dot %+v is not Primary", px)
	}
	if px := tt.Image().RGBAAt(int(ax), int(ay-radioRingSize/2)); !colorMatches(px, cs.OnSurfaceVariant, 40) {
		t.Fatalf("unselected radio ring %+v is not OnSurfaceVariant", px)
	}

	choice = "A"
	disabled := selectionCanvas(th, func(c *ui.Context) {
		Radio(c, &choice, "A", "Alpha", RadioOptions{Disabled: true})
	})
	info := buttonInfo(t, disabled, ui.RoleRadio, "Alpha")
	if !info.Disabled {
		t.Fatal("disabled radio does not report itself disabled")
	}
	want := cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity)).Over(ui.RGB(255, 255, 255))
	cx, cy := checkboxSlotCenter(info.Bounds, false, radioTouchTarget)
	if px := disabled.Image().RGBAAt(int(cx), int(cy)); !colorMatches(px, want, 12) {
		t.Fatalf("disabled radio dot %+v, want %+v", px, rgbOf(want))
	}
}

// TestRadioGroupKeyboardAndDisabled checks the group's single Tab stop, arrow,
// Home and End selection, the update of *selected and the access states.
func TestRadioGroupKeyboardAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	choice := "Small"
	tt := selectionCanvas(th, func(c *ui.Context) {
		RadioGroup(c, func() {
			Radio(c, &choice, "Small", "Small")
			Radio(c, &choice, "Medium", "Medium")
			Radio(c, &choice, "Large", "Large")
		}, RadioGroupOptions{Label: "Size"})
	})

	if _, ok := tt.AccessByRole(ui.RoleRadioGroup, "Size"); !ok {
		t.Fatal("radio group not in the access tree")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Small") {
		t.Fatal("Tab did not focus the chosen radio")
	}
	tt.Key(0, ui.KeyRight)
	if choice != "Medium" {
		t.Fatalf("Right chose %q, want Medium", choice)
	}
	medium := buttonInfo(t, tt, ui.RoleRadio, "Medium")
	if !medium.Checked {
		t.Fatal("assistive technology does not see Medium chosen")
	}
	tt.Key(0, ui.KeyHome)
	if choice != "Small" {
		t.Fatalf("Home chose %q, want Small", choice)
	}
	tt.Key(0, ui.KeyEnd)
	if choice != "Large" {
		t.Fatalf("End chose %q, want Large", choice)
	}
	info := buttonInfo(t, tt, ui.RoleRadio, "Medium")
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if choice != "Medium" {
		t.Fatalf("click chose %q, want Medium", choice)
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("radio dot spring did not settle")
	}

	choice = "Small"
	disabled := selectionCanvas(th, func(c *ui.Context) {
		RadioGroup(c, func() {
			Radio(c, &choice, "Small", "Small")
			Radio(c, &choice, "Medium", "Medium")
		}, RadioGroupOptions{Label: "Size", Disabled: true})
	})
	if info := buttonInfo(t, disabled, ui.RoleRadio, "Small"); !info.Disabled {
		t.Fatal("radio in a disabled group is not disabled")
	}
	disabled.Key(0, ui.KeyTab)
	disabled.Key(0, ui.KeyRight)
	disabled.Advance(600 * time.Millisecond)
	if choice != "Small" {
		t.Fatalf("disabled group chose %q", choice)
	}
	if disabled.FramePending() {
		t.Fatal("disabled group left frames pending")
	}
}

// TestSelectionReduceMotion checks that a ReduceMotion check box and radio mark
// their new state at once, without asking for animation frames.
func TestSelectionReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	checked := false
	choice := "A"
	tt := selectionCanvas(th, func(c *ui.Context) {
		Checkbox(c, &checked, CheckboxOptions{Label: "Sync"})
		Radio(c, &choice, "A", "A")
		Radio(c, &choice, "B", "B")
	})
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	info := buttonInfo(t, tt, ui.RoleCheckBox, "Sync")
	cx, cy := checkboxSlotCenter(info.Bounds, false, checkboxTouchTarget)
	tt.ClickAt(cx, cy)
	if !checked {
		t.Fatal("ReduceMotion click did not check the box")
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion check box should not ask for frames")
	}
	img := tt.Image()
	ink := 0
	box := ui.Rect{X: cx - checkboxBoxSize/2, Y: cy - checkboxBoxSize/2, W: checkboxBoxSize, H: checkboxBoxSize}
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if colorMatches(img.RGBAAt(x, y), th.Colors.OnPrimary, 24) {
				ink++
			}
		}
	}
	if ink < 6 {
		t.Fatalf("ReduceMotion check box inked only %d pixels", ink)
	}

	b := buttonInfo(t, tt, ui.RoleRadio, "B")
	bcx, bcy := checkboxSlotCenter(b.Bounds, false, radioTouchTarget)
	tt.ClickAt(bcx, bcy)
	if choice != "B" {
		t.Fatalf("ReduceMotion radio chose %q", choice)
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion radio should not ask for frames")
	}
	if px := tt.Image().RGBAAt(int(bcx), int(bcy)); !colorMatches(px, th.Colors.Primary, 30) {
		t.Fatalf("ReduceMotion radio dot %+v is not Primary", px)
	}
}
