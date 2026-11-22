package md3e

import (
	"image/color"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// colorMatches reports whether a rendered pixel is within tol of col.
func colorMatches(px color.RGBA, col ui.Color, tol int) bool {
	return absColorDiff(px, rgbOf(col)) <= tol
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

// thumbExtent finds the leftmost and rightmost pixels of col inside the track,
// leaving out its thin border, across a band around its center row.
func thumbExtent(tt *ui.Tester, track ui.Rect, col ui.Color) (int, int) {
	img := tt.Image()
	x0, x1 := int(track.X)+3, int(track.X+track.W)-3
	y0 := int(track.Y+track.H/2) - 5
	y1 := int(track.Y+track.H/2) + 5
	minX, maxX := -1, -1
	for y := y0; y <= y1; y++ {
		for x := x0; x < x1; x++ {
			if colorMatches(img.RGBAAt(x, y), col, 30) {
				if minX < 0 || x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
			}
		}
	}
	return minX, maxX
}

// thumbWidthOnRow measures the handle of col in the track, its diameter after
// its spring settled. Its icon may cut the handle's color in two, so the
// leftmost and rightmost pixels of a band around the center row tell its size.
func thumbWidthOnRow(t *testing.T, tt *ui.Tester, track ui.Rect, col ui.Color) int {
	t.Helper()
	minX, maxX := thumbExtent(tt, track, col)
	if minX < 0 {
		return 0
	}
	return maxX - minX + 1
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
	offCenter := thumbCenterOnRow(t, tt, track, th.Colors.Outline)
	if math.Abs(float64(offCenter-(track.X+16))) > 3 {
		t.Fatalf("off thumb center %v, want the track's left start %v", offCenter, track.X+16)
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
	onCenter := thumbCenterOnRow(t, tt, track, th.Colors.OnPrimary)
	if math.Abs(float64(onCenter-(track.X+36))) > 4 {
		t.Fatalf("on thumb center %v, want the track's right end %v", onCenter, track.X+36)
	}
	// The check mark is the track color cut out of the OnPrimary handle. Its
	// glyph is thin and antialiased, so any pixel of the inner handle window
	// clearly darker than OnPrimary is its ink.
	img := tt.Image()
	checkInk := 0
	for y := int(track.Y + 10); y < int(track.Y+22); y++ {
		for x := int(track.X + 30); x < int(track.X+42); x++ {
			px := img.RGBAAt(x, y)
			if int(px.R)+int(px.G)+int(px.B) < 650 {
				checkInk++
			}
		}
	}
	if checkInk < 6 {
		t.Fatalf("selected handle shows %d check pixels, want a visible check", checkInk)
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
	offCenter := thumbCenterOnRow(t, tt, track, th.Colors.Outline)
	if offCenter < track.X+track.W/2 {
		t.Fatalf("RTL off thumb center %v is not at the track's right end", offCenter)
	}

	cx := track.X + track.W/2
	tt.ClickAt(cx, track.Y+track.H/2)
	if !on {
		t.Fatal("RTL switch click did not turn on")
	}
	tt.Advance(900 * time.Millisecond)
	onCenter := thumbCenterOnRow(t, tt, track, th.Colors.OnPrimary)
	if onCenter > track.X+track.W/2 {
		t.Fatalf("RTL on thumb center %v is not at the track's left end", onCenter)
	}
}

// thumbCenterOnRow returns the middle of the handle of col in the track.
func thumbCenterOnRow(t *testing.T, tt *ui.Tester, track ui.Rect, col ui.Color) float32 {
	t.Helper()
	minX, maxX := thumbExtent(tt, track, col)
	if minX < 0 {
		return track.X + track.W/2
	}
	return float32(minX+maxX) / 2
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
	want := cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity)).Over(ui.RGB(255, 255, 255))
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

// TestRadioPressRipple checks the expanding press wave of the radio's 40 DIP
// heat area: right after the press the wave darkens pixels at its origin while
// pixels away from it keep only the hover layer, and it settles without frames.
func TestRadioPressRipple(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	choice := "A"
	tt := selectionCanvas(th, func(c *ui.Context) {
		Radio(c, &choice, "A", "Alpha")
		Radio(c, &choice, "B", "Beta")
	})
	info := buttonInfo(t, tt, ui.RoleRadio, "Beta")
	cx, cy := checkboxSlotCenter(info.Bounds, false, radioTouchTarget)

	// Hover where the press lands so the state layer is already at its hover
	// opacity, then press: the wave alone darkens the pixels around its origin.
	tt.Move(cx+16, cy)
	tt.Advance(600 * time.Millisecond)
	tt.Press(cx+16, cy)
	near := tt.Image().RGBAAt(int(cx+18), int(cy))
	far := tt.Image().RGBAAt(int(cx), int(cy))
	nearSum := int(near.R) + int(near.G) + int(near.B)
	farSum := int(far.R) + int(far.G) + int(far.B)
	if farSum-nearSum < 10 {
		t.Fatalf("press wave did not darken its origin: near %+v far %+v", near, far)
	}
	tt.Release(cx+16, cy)
	tt.Advance(700 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("radio press wave did not settle")
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
	tt.Key(0, ui.KeyDown)
	if choice != "Medium" {
		t.Fatalf("Down chose %q, want Medium", choice)
	}
	tt.Key(0, ui.KeyUp)
	if choice != "Small" {
		t.Fatalf("Up chose %q, want Small", choice)
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

// TestSelectionContextKeys checks that Context.Key given to a control lands on
// its stateful base, so the caller's c.Key usage works as documented.
func TestSelectionContextKeys(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	on := false
	checked := false
	choice := ""
	tt := selectionCanvas(th, func(c *ui.Context) {
		Switch(c.Key("wifi"), &on, SwitchOptions{Label: "Wi-Fi"})
		Checkbox(c.Key("sync"), &checked, CheckboxOptions{Label: "Sync"})
		RadioGroup(c, func() {
			Radio(c.Key("radio-a"), &choice, "A", "Alpha")
		}, RadioGroupOptions{Label: "Choice"})
	})
	sw := buttonInfo(t, tt, ui.RoleSwitch, "Wi-Fi")
	tt.ClickAt(sw.Bounds.X+sw.Bounds.W/2, sw.Bounds.Y+sw.Bounds.H/2)
	if !on {
		t.Fatal("keyed switch did not toggle")
	}
	cb := buttonInfo(t, tt, ui.RoleCheckBox, "Sync")
	tt.ClickAt(cb.Bounds.X+cb.Bounds.W/2, cb.Bounds.Y+cb.Bounds.H/2)
	if !checked {
		t.Fatal("keyed check box did not toggle")
	}
	rd := buttonInfo(t, tt, ui.RoleRadio, "Alpha")
	tt.ClickAt(rd.Bounds.X+rd.Bounds.W/2, rd.Bounds.Y+rd.Bounds.H/2)
	if choice != "A" {
		t.Fatalf("keyed radio chose %q", choice)
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

func TestRadioGroupDisabledPropagationAndColors(t *testing.T) {
	th := ThemeFromSeed(ui.Hex("#6750A4"))
	th.ReduceMotion = true

	// Compare standalone disabled selected Radio vs Radio inside disabled RadioGroup vs ancestor disabled.
	vStandalone := "a"
	ttStandalone := selectionCanvas(th, func(c *ui.Context) {
		Radio(c, &vStandalone, "a", "OptA", RadioOptions{Disabled: true})
	})
	saInfo := buttonInfo(t, ttStandalone, ui.RoleRadio, "OptA")
	wantCol := ttStandalone.Image().RGBAAt(int(saInfo.Bounds.X+20), int(saInfo.Bounds.Y+20))

	checkGroup := func(name string, useAncestor bool) {
		t.Run(name, func(t *testing.T) {
			choice := "a"
			tt := selectionCanvas(th, func(c *ui.Context) {
				wrap := ui.Column(c)
				if useAncestor {
					wrap.Disabled(true).DimDisabled(false)
				}
				wrap.Children(func() {
					RadioGroup(c, func() {
						Radio(c, &choice, "a", "OptA")
						Radio(c, &choice, "b", "OptB")
					}, RadioGroupOptions{Label: "Modes", Disabled: !useAncestor})
				})
			})
			aInfo := buttonInfo(t, tt, ui.RoleRadio, "OptA")
			bInfo := buttonInfo(t, tt, ui.RoleRadio, "OptB")
			if !aInfo.Disabled || !bInfo.Disabled {
				t.Fatalf("expected radios disabled: a=%v b=%v", aInfo.Disabled, bInfo.Disabled)
			}
			tt.ClickAt(bInfo.Bounds.X+bInfo.Bounds.W/2, bInfo.Bounds.Y+bInfo.Bounds.H/2)
			if choice != "a" {
				t.Fatalf("clicking disabled group radio changed choice to %q", choice)
			}
			gotCol := tt.Image().RGBAAt(int(aInfo.Bounds.X+20), int(aInfo.Bounds.Y+20))
			if absColorDiff(gotCol, wantCol) > 12 {
				t.Fatalf("group-disabled radio pixel %+v != standalone disabled radio %+v", gotCol, wantCol)
			}
		})
	}

	checkGroup("group-option", false)
	checkGroup("ancestor-disabled", true)
}

func TestSelectionControlsHoverZeroRebuildsAndSettles(t *testing.T) {
	check := func(name string, buildFn func(c *ui.Context, on *bool, choice *string), centerFn func(t *testing.T, tt *ui.Tester) (float32, float32)) {
		t.Run(name, func(t *testing.T) {
			builds := 0
			on := false
			choice := "other"
			tt := ui.NewTester(func(c *ui.Context) {
				builds++
				ThemeScope(c, nil, func() {
					ui.Box(c).Size(240, 120).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
						buildFn(c, &on, &choice)
					})
				})
			}, 240, 120)

			cx, cy := centerFn(t, tt)
			b0 := builds
			tt.Move(cx, cy)
			if builds > b0+1 {
				t.Fatalf("%s hover enter caused %d builds (expected at most 1 Tester settle build)", name, builds-b0)
			}
			afterEnter := builds
			tt.Advance(300 * time.Millisecond)
			if builds != afterEnter {
				t.Fatalf("%s hover fade triggered %d rebuilds during Advance, want 0", name, builds-afterEnter)
			}
			if tt.FramePending() {
				t.Fatalf("%s hover fade did not settle", name)
			}

			tt.Move(2, 2)
			if builds > afterEnter+1 {
				t.Fatalf("%s hover leave caused %d builds (expected at most 1 Tester settle build)", name, builds-afterEnter)
			}
			afterLeave := builds
			tt.Advance(300 * time.Millisecond)
			if builds != afterLeave {
				t.Fatalf("%s hover leave fade triggered %d rebuilds during Advance, want 0", name, builds-afterLeave)
			}
			if tt.FramePending() {
				t.Fatalf("%s hover leave fade did not settle", name)
			}

			// ReduceMotion must snap hover halo with zero pending frames.
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			bRM := builds
			tt.Move(cx, cy)
			if tt.FramePending() {
				t.Fatalf("%s ReduceMotion hover enter left frames pending", name)
			}
			if builds > bRM+1 {
				t.Fatalf("%s ReduceMotion hover enter caused %d builds", name, builds-bRM)
			}
			bRM2 := builds
			tt.Move(2, 2)
			if tt.FramePending() {
				t.Fatalf("%s ReduceMotion hover leave left frames pending", name)
			}
			if builds > bRM2+1 {
				t.Fatalf("%s ReduceMotion hover leave caused %d builds", name, builds-bRM2)
			}
		})
	}

	check("switch", func(c *ui.Context, on *bool, _ *string) {
		Switch(c, on, SwitchOptions{Label: "Wi-Fi"})
	}, func(t *testing.T, tt *ui.Tester) (float32, float32) {
		info := buttonInfo(t, tt, ui.RoleSwitch, "Wi-Fi")
		return info.Bounds.X + 20, info.Bounds.Y + info.Bounds.H/2
	})

	check("checkbox", func(c *ui.Context, on *bool, _ *string) {
		Checkbox(c, on, CheckboxOptions{Label: "Sync"})
	}, func(t *testing.T, tt *ui.Tester) (float32, float32) {
		info := buttonInfo(t, tt, ui.RoleCheckBox, "Sync")
		return info.Bounds.X + 20, info.Bounds.Y + info.Bounds.H/2
	})

	check("radio", func(c *ui.Context, _ *bool, choice *string) {
		Radio(c, choice, "a", "ChoiceA")
	}, func(t *testing.T, tt *ui.Tester) (float32, float32) {
		info := buttonInfo(t, tt, ui.RoleRadio, "ChoiceA")
		return info.Bounds.X + 20, info.Bounds.Y + info.Bounds.H/2
	})
}
