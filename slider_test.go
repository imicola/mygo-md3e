package md3e

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// sliderCanvas builds a 420x260 white canvas with the controls 100 DIP down,
// leaving room for value indicators above them.
func sliderCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(420, 260).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				ui.Box(c).Height(100)
				build(c)
			})
		})
	}, 420, 260)
}

// sliderX returns the x of fraction f of a slider's content box.
func sliderX(b ui.Rect, f float32) float32 {
	return b.X + sliderPad + (b.W-2*sliderPad)*f
}

// hasColorInRect reports whether any pixel of rect matches col.
func hasColorInRect(img *image.RGBA, rect ui.Rect, col ui.Color, tol int) bool {
	for y := int(rect.Y); y < int(rect.Y+rect.H); y++ {
		for x := int(rect.X); x < int(rect.X+rect.W); x++ {
			if colorMatches(img.RGBAAt(x, y), col, tol) {
				return true
			}
		}
	}
	return false
}

// TestSliderDragKeyboardAndStep checks pointer dragging, the keyboard keys, the
// step snapping and the access range and value.
func TestSliderDragKeyboardAndStep(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := 25.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume"})
	})
	info := buttonInfo(t, tt, ui.RoleSlider, "Volume")
	cx, cy := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
	target := sliderX(info.Bounds, 0.75)
	tt.Press(target, cy)
	tt.Release(target, cy)
	if math.Abs(value-75) > 1.5 {
		t.Fatalf("drag set value %v, want about 75", value)
	}
	tt.Advance(time.Millisecond)
	info, _ = tt.AccessByRole(ui.RoleSlider, "Volume")
	if math.Abs(info.Now-75) > 1.5 || info.Min != 0 || info.Max != 100 {
		t.Fatalf("access range %v..%v now %v, want 0..100 about 75", info.Min, info.Max, info.Now)
	}
	_ = cx

	// Step: dragging snaps to the step, and the arrows move by it.
	stepped := 0.0
	st := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &stepped, 0, 100, SliderOptions{Label: "Quality", Step: 10})
	})
	sinfo := buttonInfo(t, st, ui.RoleSlider, "Quality")
	x37 := sliderX(sinfo.Bounds, 0.37)
	y := sinfo.Bounds.Y + sinfo.Bounds.H/2
	st.ClickAt(x37, y)
	if stepped != 40 {
		t.Fatalf("step drag set %v, want 40", stepped)
	}
	st.Key(0, ui.KeyRight)
	if stepped != 50 {
		t.Fatalf("Right set %v, want 50", stepped)
	}
	st.Key(0, ui.KeyHome)
	if stepped != 0 {
		t.Fatalf("Home set %v, want 0", stepped)
	}
	st.Key(0, ui.KeyEnd)
	if stepped != 100 {
		t.Fatalf("End set %v, want 100", stepped)
	}
	st.Key(0, ui.KeyPageDown)
	if stepped != 90 {
		t.Fatalf("PageDown set %v, want 90", stepped)
	}
	st.Key(0, ui.KeyLeft)
	if stepped != 80 {
		t.Fatalf("Left set %v, want 80", stepped)
	}
	st.Advance(time.Millisecond)
	sinfo, _ = st.AccessByRole(ui.RoleSlider, "Quality")
	if sinfo.Now != 80 || sinfo.Step != 10 {
		t.Fatalf("access now %v step %v, want 80 and 10", sinfo.Now, sinfo.Step)
	}
}

// TestSliderTrackGapCenteredAndStops checks the 6 DIP gaps around the handle,
// the centered two-sided fill and the discrete stop dots.
func TestSliderTrackGapCenteredAndStops(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors

	value := 50.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume"})
	})
	b := buttonInfo(t, tt, ui.RoleSlider, "Volume").Bounds
	y := int(b.Y + b.H/2)
	tx := sliderX(b, 0.5)
	img := tt.Image()
	gapLeft := img.RGBAAt(int(tx-4), y)
	if absColorDiff(gapLeft, rgbOf(ui.RGB(255, 255, 255))) > 6 {
		t.Fatalf("left track gap %+v is not the canvas", gapLeft)
	}
	gapRight := img.RGBAAt(int(tx+4), y)
	if absColorDiff(gapRight, rgbOf(ui.RGB(255, 255, 255))) > 6 {
		t.Fatalf("right track gap %+v is not the canvas", gapRight)
	}
	if px := img.RGBAAt(int(tx-11), y); !colorMatches(px, cs.Primary, 20) {
		t.Fatalf("active track %+v is not Primary", px)
	}
	if px := img.RGBAAt(int(tx+11), y); !colorMatches(px, cs.SurfaceContainerHighest, 20) {
		t.Fatalf("inactive track %+v is not SurfaceContainerHighest", px)
	}
	if px := img.RGBAAt(int(tx), y); !colorMatches(px, cs.Primary, 20) {
		t.Fatalf("handle %+v is not the Primary capsule", px)
	}

	// Centered: the active track fills from the middle to the handle, and the
	// far side stays inactive.
	centered := 25.0
	ct := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &centered, 0, 100, SliderOptions{Label: "Balance", Centered: true})
	})
	cb := buttonInfo(t, ct, ui.RoleSlider, "Balance").Bounds
	cy := int(cb.Y + cb.H/2)
	if px := ct.Image().RGBAAt(int(sliderX(cb, 0.4)), cy); !colorMatches(px, cs.Primary, 20) {
		t.Fatalf("centered active half %+v is not Primary", px)
	}
	if px := ct.Image().RGBAAt(int(sliderX(cb, 0.6)), cy); !colorMatches(px, cs.SurfaceContainerHighest, 20) {
		t.Fatalf("centered far half %+v is not inactive", px)
	}

	// Stop dots: the active one is OnPrimary on the Primary track, the inactive
	// one OnSurfaceVariant on the inactive track.
	stop := 0.0
	st := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &stop, 0, 1, SliderOptions{Label: "Rate", Step: 0.25})
	})
	sb := buttonInfo(t, st, ui.RoleSlider, "Rate").Bounds
	sy := int(sb.Y + sb.H/2)
	// Click at 0.5 so one stop is active and one inactive.
	st.ClickAt(sliderX(sb, 0.5), float32(sy))
	sim := st.Image()
	activeDot := sim.RGBAAt(int(sliderX(sb, 0.25)), sy)
	if !colorMatches(activeDot, cs.OnPrimary, 40) {
		t.Fatalf("active stop dot %+v is not OnPrimary", activeDot)
	}
	inactiveDot := sim.RGBAAt(int(sliderX(sb, 0.75)), sy)
	if !colorMatches(inactiveDot, cs.OnSurfaceVariant, 40) {
		t.Fatalf("inactive stop dot %+v is not OnSurfaceVariant", inactiveDot)
	}
	if absColorDiff(activeDot, inactiveDot) < 40 {
		t.Fatalf("active and inactive stop dots are too alike: %+v and %+v", activeDot, inactiveDot)
	}

	// Pressing narrows the capsule handle from 4 to 2 DIP and it springs back.
	pressValue := 50.0
	pt := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &pressValue, 0, 100, SliderOptions{Label: "Volume"})
	})
	pb := buttonInfo(t, pt, ui.RoleSlider, "Volume").Bounds
	pty := int(pb.Y + pb.H/2)
	ptx := sliderX(pb, 0.5)
	pt.Press(ptx, float32(pty))
	pt.Advance(600 * time.Millisecond)
	pressInk := 0
	for x := int(ptx) - 4; x <= int(ptx)+4; x++ {
		if colorMatches(pt.Image().RGBAAt(x, pty), cs.Primary, 20) {
			pressInk++
		}
	}
	if pressInk > 3 {
		t.Fatalf("pressed handle is %d px wide, want the narrowed 2 DIP capsule", pressInk)
	}
	pt.Release(ptx, float32(pty))
	pt.Advance(900 * time.Millisecond)
	restInk := 0
	for x := int(ptx) - 4; x <= int(ptx)+4; x++ {
		if colorMatches(pt.Image().RGBAAt(x, pty), cs.Primary, 20) {
			restInk++
		}
	}
	if restInk < 3 {
		t.Fatalf("released handle is %d px wide, want the resting 4 DIP capsule", restInk)
	}
}

// TestSliderValueIndicator checks that the teardrop indicator springs in above
// the dragged handle and leaves again on release, and that ReduceMotion shows
// and hides it without pending frames.
func TestSliderValueIndicator(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := 50.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume"})
	})
	info := buttonInfo(t, tt, ui.RoleSlider, "Volume")
	cx, cy := sliderX(info.Bounds, 0.5), info.Bounds.Y+info.Bounds.H/2
	above := ui.Rect{X: cx - 70, Y: info.Bounds.Y - sliderIndicatorHeight - 8, W: 140, H: sliderIndicatorHeight}
	tt.Press(cx, cy)
	tt.Advance(700 * time.Millisecond)
	if !hasColorInRect(tt.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("value indicator did not show above the pressed handle")
	}
	tt.Release(cx, cy)
	tt.Advance(1200 * time.Millisecond)
	if hasColorInRect(tt.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("value indicator did not hide after release")
	}
	if tt.FramePending() {
		t.Fatal("indicator spring did not settle")
	}

	// ReduceMotion: it appears at once and leaves at once.
	rm := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume"})
	})
	rm.SetPreferences(ui.Preferences{ReduceMotion: true})
	info = buttonInfo(t, rm, ui.RoleSlider, "Volume")
	cx, cy = sliderX(info.Bounds, 0.5), info.Bounds.Y+info.Bounds.H/2
	above = ui.Rect{X: cx - 70, Y: info.Bounds.Y - sliderIndicatorHeight - 8, W: 140, H: sliderIndicatorHeight}
	rm.Press(cx, cy)
	if !hasColorInRect(rm.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("ReduceMotion value indicator did not show at once")
	}
	if rm.FramePending() {
		t.Fatal("ReduceMotion indicator should not ask for frames")
	}
	rm.Release(cx, cy)
	if hasColorInRect(rm.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("ReduceMotion value indicator did not hide at once")
	}
	if rm.FramePending() {
		t.Fatal("ReduceMotion indicator release should not ask for frames")
	}

	// Keyboard focus alone shows the indicator, and losing focus hides it.
	kv := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume"})
	})
	kv.Key(0, ui.KeyTab)
	kinfo := buttonInfo(t, kv, ui.RoleSlider, "Volume")
	above = ui.Rect{
		X: sliderX(kinfo.Bounds, 0.5) - 70,
		Y: kinfo.Bounds.Y - sliderIndicatorHeight - 8,
		W: 140,
		H: sliderIndicatorHeight,
	}
	kv.Advance(700 * time.Millisecond)
	if !hasColorInRect(kv.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("keyboard focus did not show the value indicator")
	}
	kv.ClickAt(400, 20)
	kv.Advance(1200 * time.Millisecond)
	if hasColorInRect(kv.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("losing focus did not hide the value indicator")
	}
	if kv.FramePending() {
		t.Fatal("focus indicator spring did not settle")
	}
}

// TestSliderRTLMirror checks that the low value sits at the right, that a drag
// mirrors the value and that Right decreases it.
func TestSliderRTLMirror(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := 25.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume"})
	})
	tt.SetDirection(ui.RightToLeft)
	info := buttonInfo(t, tt, ui.RoleSlider, "Volume")
	b := info.Bounds
	y := int(b.Y + b.H/2)
	if px := tt.Image().RGBAAt(int(sliderX(b, 0.75)), y); !colorMatches(px, th.Colors.Primary, 20) {
		t.Fatalf("RTL handle at the 75%% display position is %+v, not Primary", px)
	}
	if px := tt.Image().RGBAAt(int(sliderX(b, 0.25)), y); !colorMatches(px, th.Colors.SurfaceContainerHighest, 20) {
		t.Fatalf("RTL 25%% position %+v is not inactive track", px)
	}

	target := sliderX(b, 0.25)
	tt.Press(target, b.Y+b.H/2)
	tt.Release(target, b.Y+b.H/2)
	if math.Abs(value-75) > 1.5 {
		t.Fatalf("RTL drag set %v, want about 75", value)
	}
	tt.Key(0, ui.KeyRight)
	if value >= 75 {
		t.Fatalf("RTL Right set %v, want a lower value", value)
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("RTL slider springs did not settle")
	}
}

// TestRangeSliderDragCrossingAndKeyboard checks that the low handle cannot pass
// the high one, that the track presses drag the nearest handle, and that each
// handle takes the keys independently.
func TestRangeSliderDragCrossingAndKeyboard(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	low, high := 20.0, 80.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	minInfo := buttonInfo(t, tt, ui.RoleSlider, "Min")
	maxInfo := buttonInfo(t, tt, ui.RoleSlider, "Max")
	if minInfo.Bounds.X >= maxInfo.Bounds.X {
		t.Fatalf("LTR Min at %v is not left of Max at %v", minInfo.Bounds.X, maxInfo.Bounds.X)
	}
	minCx, minCy := minInfo.Bounds.X+minInfo.Bounds.W/2, minInfo.Bounds.Y+minInfo.Bounds.H/2
	tt.Press(minCx, minCy)
	tt.Move(360, minCy)
	tt.Release(360, minCy)
	if low > high {
		t.Fatalf("low %v passed high %v", low, high)
	}
	if math.Abs(low-80) > 1.5 {
		t.Fatalf("crossing drag left low at %v, want clamped to 80", low)
	}
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("range thumb springs did not settle")
	}

	// Keyboard: each handle moves on its own, within the other's value.
	low, high = 20, 80
	kb := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	kb.Key(0, ui.KeyTab)
	if !kb.Focused("Min") {
		t.Fatal("Tab did not focus the Min handle")
	}
	kb.Key(0, ui.KeyRight)
	if low != 21 {
		t.Fatalf("Right set Min %v, want 21", low)
	}
	kb.Key(0, ui.KeyLeft)
	if low != 20 {
		t.Fatalf("Left set Min %v, want 20", low)
	}
	kb.Key(0, ui.KeyRight)
	kb.Key(0, ui.KeyTab)
	if !kb.Focused("Max") {
		t.Fatal("second Tab did not focus the Max handle")
	}
	kb.Key(0, ui.KeyHome)
	if high != low {
		t.Fatalf("Home set Max %v, want the Min value %v", high, low)
	}
	kb.Key(0, ui.KeyEnd)
	if high != 100 {
		t.Fatalf("End set Max %v, want 100", high)
	}
	kb.Key(0, ui.KeyPageDown)
	if high != 90 {
		t.Fatalf("PageDown set Max %v, want 90", high)
	}
	kb.Key(0, ui.KeyHome)
	kb.Advance(time.Millisecond)
	maxInfo = buttonInfo(t, kb, ui.RoleSlider, "Max")
	minInfo = buttonInfo(t, kb, ui.RoleSlider, "Min")
	if maxInfo.Now != low || maxInfo.Min != low || maxInfo.Max != 100 {
		t.Fatalf("Max access range Min %v Max %v Now %v, want %v 100 %v", maxInfo.Min, maxInfo.Max, maxInfo.Now, low, low)
	}
	if minInfo.Min != 0 || minInfo.Max != high {
		t.Fatalf("Min access range %v..%v, want 0..%v", minInfo.Min, minInfo.Max, high)
	}

	// A press on the track beside the handles drags the nearer one.
	low, high = 20, 80
	nearest := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	gb := buttonInfo(t, nearest, ui.RoleGroup, "Price").Bounds
	tx := sliderX(gb, 0.65)
	ty := gb.Y + gb.H/2
	nearest.Press(tx, ty)
	nearest.Release(tx, ty)
	if math.Abs(high-65) > 2 || low != 20 {
		t.Fatalf("track press set low=%v high=%v, want the nearer Max at about 65", low, high)
	}
	nearest.Advance(900 * time.Millisecond)
	if nearest.FramePending() {
		t.Fatal("track press left halo springs pending")
	}
}

// TestRangeSliderEqualValuesAndRTL checks that equal handles can be pulled
// apart, that the handles mirror right to left, and that a drag moves the low
// value toward the left there.
func TestRangeSliderEqualValuesAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	low, high := 50.0, 50.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	b := buttonInfo(t, tt, ui.RoleGroup, "Price").Bounds
	y := b.Y + b.H/2
	x := sliderX(b, 0.4)
	tt.Press(x, y)
	tt.Move(sliderX(b, 0.1), y)
	tt.Release(sliderX(b, 0.1), y)
	if low >= 50 || math.Abs(high-50) > 1.5 {
		t.Fatalf("left drag on equal handles set low=%v high=%v, want low below 50 and high at 50", low, high)
	}

	// Rightward drag from the same equal pair moves the other handle.
	low, high = 50, 50
	rt := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	rb := buttonInfo(t, rt, ui.RoleGroup, "Price").Bounds
	ry := rb.Y + rb.H/2
	rx := sliderX(rb, 0.6)
	rt.Press(rx, ry)
	rt.Move(sliderX(rb, 0.9), ry)
	rt.Release(sliderX(rb, 0.9), ry)
	if high <= 50 || math.Abs(low-50) > 1.5 {
		t.Fatalf("right drag on equal handles set low=%v high=%v, want high above 50 and low at 50", low, high)
	}

	// Right to left: Min is at the right, and dragging it further left raises it.
	low, high = 20, 80
	rtl := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	rtl.SetDirection(ui.RightToLeft)
	minInfo := buttonInfo(t, rtl, ui.RoleSlider, "Min")
	maxInfo := buttonInfo(t, rtl, ui.RoleSlider, "Max")
	if minInfo.Bounds.X <= maxInfo.Bounds.X {
		t.Fatalf("RTL Min at %v is not right of Max at %v", minInfo.Bounds.X, maxInfo.Bounds.X)
	}
	mcx, mcy := minInfo.Bounds.X+minInfo.Bounds.W/2, minInfo.Bounds.Y+minInfo.Bounds.H/2
	rtl.Press(mcx, mcy)
	rtl.Move(mcx-60, mcy)
	rtl.Release(mcx-60, mcy)
	if low <= 20 || low > high {
		t.Fatalf("RTL leftward Min drag set low=%v high=%v, want low raised within Max", low, high)
	}
	before := low
	rtl.Key(0, ui.KeyLeft)
	if low <= before {
		t.Fatalf("RTL Left set Min %v, want a higher value than %v", low, before)
	}
	rtl.Advance(900 * time.Millisecond)
	if rtl.FramePending() {
		t.Fatal("RTL range springs did not settle")
	}
}

// TestRangeSliderIndicator checks that the dragged handle shows its own value
// above it, and that the indicator hides on release.
func TestRangeSliderIndicator(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	low, high := 20.0, 80.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	minInfo := buttonInfo(t, tt, ui.RoleSlider, "Min")
	maxInfo := buttonInfo(t, tt, ui.RoleSlider, "Max")
	if minInfo.Bounds.Y != maxInfo.Bounds.Y {
		t.Fatalf("range handles differ in y: %v and %v", minInfo.Bounds.Y, maxInfo.Bounds.Y)
	}
	cx := minInfo.Bounds.X + minInfo.Bounds.W/2
	cy := minInfo.Bounds.Y + minInfo.Bounds.H/2
	above := ui.Rect{
		X: cx - 70,
		Y: minInfo.Bounds.Y - sliderIndicatorHeight - 8,
		W: 140,
		H: sliderIndicatorHeight,
	}
	tt.Press(cx, cy)
	tt.Advance(700 * time.Millisecond)
	if !hasColorInRect(tt.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("range slider did not show the dragged handle's value")
	}
	tt.Release(cx, cy)
	tt.Advance(1200 * time.Millisecond)
	if hasColorInRect(tt.Image(), above, th.Colors.InverseSurface, 16) {
		t.Fatal("range slider value indicator did not hide")
	}
	if tt.FramePending() {
		t.Fatal("range slider springs did not settle")
	}
}

// TestSliderDisabledAndReduceMotion checks that disabled sliders ignore the
// pointer and the keys, and that ReduceMotion leaves no frames pending.
func TestSliderDisabledAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := 30.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume", Disabled: true})
	})
	info := buttonInfo(t, tt, ui.RoleSlider, "Volume")
	if !info.Disabled {
		t.Fatal("disabled slider does not report itself disabled")
	}
	cs := th.Colors
	white := ui.RGB(255, 255, 255)
	b := info.Bounds
	y := int(b.Y + b.H/2)
	wantInactive := cs.OnSurface.Alpha(0.12).Over(white)
	if px := tt.Image().RGBAAt(int(sliderX(b, 0.9)), y); !colorMatches(px, wantInactive, 12) {
		t.Fatalf("disabled inactive track %+v, want OnSurface at 12%%", px)
	}
	wantActive := cs.OnSurface.Alpha(0.38).Over(white)
	if px := tt.Image().RGBAAt(int(sliderX(b, 0.15)), y); !colorMatches(px, wantActive, 12) {
		t.Fatalf("disabled active track %+v, want OnSurface at 38%%", px)
	}
	tt.ClickAt(sliderX(info.Bounds, 0.8), info.Bounds.Y+info.Bounds.H/2)
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyRight)
	tt.Advance(600 * time.Millisecond)
	if value != 30 {
		t.Fatalf("disabled slider changed its value to %v", value)
	}
	if tt.FramePending() {
		t.Fatal("disabled slider left frames pending")
	}

	low, high := 20.0, 80.0
	rd := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price", Disabled: true})
	})
	minInfo := buttonInfo(t, rd, ui.RoleSlider, "Min")
	if !minInfo.Disabled {
		t.Fatal("disabled range handle does not report itself disabled")
	}
	rd.ClickAt(sliderX(minInfo.Bounds, 0.9), minInfo.Bounds.Y+minInfo.Bounds.H/2)
	rd.Key(0, ui.KeyTab)
	rd.Key(0, ui.KeyRight)
	rd.Advance(600 * time.Millisecond)
	if low != 20 || high != 80 {
		t.Fatalf("disabled range slider changed to %v..%v", low, high)
	}
	if rd.FramePending() {
		t.Fatal("disabled range slider left frames pending")
	}

	// ReduceMotion: dragging lands at once without animation frames.
	value = 30
	rm := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &value, 0, 100, SliderOptions{Label: "Volume"})
	})
	rm.SetPreferences(ui.Preferences{ReduceMotion: true})
	info = buttonInfo(t, rm, ui.RoleSlider, "Volume")
	rm.ClickAt(sliderX(info.Bounds, 0.8), info.Bounds.Y+info.Bounds.H/2)
	if math.Abs(value-80) > 1.5 {
		t.Fatalf("ReduceMotion drag set %v, want about 80", value)
	}
	if rm.FramePending() {
		t.Fatal("ReduceMotion slider should not ask for frames")
	}

	// A ReduceMotion range drag lands at once and leaves no animations behind.
	low, high = 20, 80
	rrm := sliderCanvas(th, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price"})
	})
	rrm.SetPreferences(ui.Preferences{ReduceMotion: true})
	rmin := buttonInfo(t, rrm, ui.RoleSlider, "Min")
	rmx, rmy := rmin.Bounds.X+rmin.Bounds.W/2, rmin.Bounds.Y+rmin.Bounds.H/2
	rrm.Press(rmx, rmy)
	rrm.Move(rmx+60, rmy)
	rrm.Release(rmx+60, rmy)
	if low <= 20 || low > high {
		t.Fatalf("ReduceMotion range drag set low=%v high=%v", low, high)
	}
	if rrm.FramePending() {
		t.Fatal("ReduceMotion range slider should not ask for frames")
	}
}

func TestSliderExitTransitionSnapshotAndValidChildHandles(t *testing.T) {
	show := true
	val := 60.0
	low, high := 20.0, 80.0
	tr := ui.ElementTransition{
		Duration: 180 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}

	tt := sliderCanvas(nil, func(c *ui.Context) {
		ui.Column(c).Width(280).Gap(12).Children(func() {
			if show {
				ui.Column(c.Key("sliders-wrap")).Width(260).Gap(12).Transition(tr).Children(func() {
					Slider(c, &val, 0, 100, SliderOptions{Label: "Volume", Step: 10})
					RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Price", Step: 10})
				})
			}
		})
	})

	// Exit while simultaneously mutating caller-owned pointers: exit ghosts must
	// draw their build-time snapshots cleanly with no invalid handle panics.
	show = false
	tt.Frame()
	val = 0
	low = 0
	high = 0
	tt.Advance(40 * time.Millisecond)

	// Interrupt mid-exit by re-showing, then settle.
	show = true
	tt.Frame()
	tt.Advance(300 * time.Millisecond)

	// Full exit and settle.
	show = false
	tt.Frame()
	tt.Advance(300 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("slider exit transition should converge with zero pending frames")
	}
}

func TestSliderDragOutsideReleaseUnlocksThumbAndIndicator(t *testing.T) {
	th := ThemeFromSeed(ui.Hex("#6750A4"))
	val := 50.0
	tt := sliderCanvas(th, func(c *ui.Context) {
		Slider(c, &val, 0, 100, SliderOptions{Label: "Volume"})
	})
	b := buttonInfo(t, tt, ui.RoleSlider, "Volume").Bounds
	cx, cy := sliderX(b, 0.5), b.Y+b.H/2

	tt.Press(cx, cy)
	tt.Move(cx+30, cy+120)
	// Release outside the slider hit box; must still clear held/engaged state.
	tt.Release(cx+30, cy+120)
	tt.Move(2, 2)
	tt.Advance(400 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("slider did not settle after outside release")
	}
	if _, ok := tt.Find(sliderValueText(val, 0, nil)); ok {
		t.Fatal("value indicator remained open after pointer released outside slider")
	}
	// Verify thumb returned to resting width (~4 DIP, not narrow 2 DIP engaged).
	tx := sliderX(b, sliderFraction(val, 0, 100))
	if w := thumbWidthOnRow(t, tt, ui.Rect{X: tx - 10, Y: cy - 4, W: 20, H: 8}, th.Colors.Primary); w < 3 {
		t.Fatalf("thumb width = %d px after outside release, want resting ~4 DIP", w)
	}
}

func TestSliderNaNBoundsAndNarrowBoxSafety(t *testing.T) {
	nan64 := math.NaN()
	nan32 := float32(math.NaN())
	inf32 := float32(math.Inf(1))
	val := nan64
	low, high := math.Inf(1), math.NaN()

	tt := sliderCanvas(nil, func(c *ui.Context) {
		ui.Column(c).Width(16).Gap(8).Children(func() {
			Slider(c, &val, 0, 100, SliderOptions{
				Label:       "NaN",
				Step:        nan64,
				TrackHeight: nan32,
				ThumbHeight: inf32,
			})
			RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{
				Label:       "NaNRange",
				Step:        math.Inf(1),
				TrackHeight: nan32,
				ThumbHeight: nan32,
			})
		})
	})
	if math.IsNaN(val) || val != 0 {
		t.Fatalf("Slider did not sanitize NaN value: %v", val)
	}
	if math.IsNaN(low) || math.IsNaN(high) || low > high || low < 0 || high > 100 {
		t.Fatalf("RangeSlider produced invalid sanitized values: low=%v high=%v", low, high)
	}
	// Focus inside a narrow box (W <= 2*sliderPad) so value indicator checks guard cleanly.
	tt.Key(0, ui.KeyTab)
	tt.Advance(300 * time.Millisecond)
}

func TestRangeSliderClampOrderAndCoincidentCenterDragLTRAndRTL(t *testing.T) {
	// 1. Out-of-bounds clamp must clamp to [lo, hi] BEFORE enforcing low <= high.
	low, high := 150.0, 120.0
	ttClamp := sliderCanvas(nil, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Clamp"})
	})
	if low != 100 || high != 100 {
		t.Fatalf("out-of-bounds clamp gave low=%v high=%v, want 100 100", low, high)
	}
	_ = ttClamp

	// 2. Coincident center drag in LTR: dragging left lowers Min; dragging right raises Max.
	low, high = 50, 50
	ltr := sliderCanvas(nil, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "CoincidentLTR"})
	})
	gb := buttonInfo(t, ltr, ui.RoleGroup, "CoincidentLTR").Bounds
	cx, cy := sliderX(gb, 0.5), gb.Y+gb.H/2
	ltr.Press(cx, cy)
	ltr.Move(cx-50, cy)
	ltr.Release(cx-50, cy)
	if low >= 50 || high != 50 {
		t.Fatalf("LTR left drag from coincident center gave low=%v high=%v, want low<50 high=50", low, high)
	}

	low, high = 50, 50
	ltr.Frame()
	ltr.Press(cx, cy)
	ltr.Move(cx+50, cy)
	ltr.Release(cx+50, cy)
	if low != 50 || high <= 50 {
		t.Fatalf("LTR right drag from coincident center gave low=%v high=%v, want low=50 high>50", low, high)
	}

	// 3. Coincident center drag in RTL: dragging left raises Max; dragging right lowers Min.
	low, high = 50, 50
	rtl := sliderCanvas(nil, func(c *ui.Context) {
		RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "CoincidentRTL"})
	})
	rtl.SetDirection(ui.RightToLeft)
	rgb := buttonInfo(t, rtl, ui.RoleGroup, "CoincidentRTL").Bounds
	rcx, rcy := sliderX(rgb, 0.5), rgb.Y+rgb.H/2
	rtl.Press(rcx, rcy)
	rtl.Move(rcx-50, rcy)
	rtl.Release(rcx-50, rcy)
	if low != 50 || high <= 50 {
		t.Fatalf("RTL left drag from coincident center gave low=%v high=%v, want low=50 high>50", low, high)
	}

	low, high = 50, 50
	rtl.Frame()
	rtl.Press(rcx, rcy)
	rtl.Move(rcx+50, rcy)
	rtl.Release(rcx+50, rcy)
	if low >= 50 || high != 50 {
		t.Fatalf("RTL right drag from coincident center gave low=%v high=%v, want low<50 high=50", low, high)
	}
}

func TestSlidersHoverZeroRebuildsAndSettles(t *testing.T) {
	t.Run("slider", func(t *testing.T) {
		builds := 0
		val := 50.0
		tt := ui.NewTester(func(c *ui.Context) {
			builds++
			ThemeScope(c, nil, func() {
				ui.Box(c).Size(340, 120).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
					Slider(c, &val, 0, 100, SliderOptions{Label: "Vol"})
				})
			})
		}, 340, 120)

		b := buttonInfo(t, tt, ui.RoleSlider, "Vol").Bounds
		cx, cy := sliderX(b, 0.5), b.Y+b.H/2
		b0 := builds

		tt.Move(cx, cy)
		if builds > b0+1 {
			t.Fatalf("Slider hover enter caused %d builds (expected at most 1 Tester settle build)", builds-b0)
		}
		afterEnter := builds
		tt.Advance(300 * time.Millisecond)
		if builds != afterEnter {
			t.Fatalf("Slider hover settle caused %d rebuilds during Advance, want 0", builds-afterEnter)
		}
		if tt.FramePending() {
			t.Fatal("Slider hover halo did not settle")
		}
		// Halo ring around thumb must visibly tint white canvas above track.
		if px := tt.Image().RGBAAt(int(cx+8), int(cy-12)); px.R >= 253 {
			t.Fatalf("expected Slider hover halo above track, got %+v", px)
		}

		tt.Move(2, 2)
		if builds > afterEnter+1 {
			t.Fatalf("Slider hover leave caused %d builds (expected at most 1 Tester settle build)", builds-afterEnter)
		}
		afterLeave := builds
		tt.Advance(300 * time.Millisecond)
		if builds != afterLeave {
			t.Fatalf("Slider hover leave settle caused %d rebuilds during Advance, want 0", builds-afterLeave)
		}
		if tt.FramePending() {
			t.Fatal("Slider hover leave did not settle")
		}

		tt.SetPreferences(ui.Preferences{ReduceMotion: true})
		bRM := builds
		tt.Move(cx, cy)
		if tt.FramePending() {
			t.Fatal("ReduceMotion Slider hover enter left frames pending")
		}
		if builds > bRM+1 {
			t.Fatalf("ReduceMotion Slider hover enter caused %d builds", builds-bRM)
		}
		bRM2 := builds
		tt.Move(2, 2)
		if tt.FramePending() {
			t.Fatal("ReduceMotion Slider hover leave left frames pending")
		}
		if builds > bRM2+1 {
			t.Fatalf("ReduceMotion Slider hover leave caused %d builds", builds-bRM2)
		}
	})

	t.Run("range-slider", func(t *testing.T) {
		builds := 0
		low, high := 25.0, 75.0
		tt := ui.NewTester(func(c *ui.Context) {
			builds++
			ThemeScope(c, nil, func() {
				ui.Box(c).Size(340, 120).Background(ui.RGB(255, 255, 255)).Padding(20).Children(func() {
					RangeSlider(c, &low, &high, 0, 100, RangeSliderOptions{Label: "Range"})
				})
			})
		}, 340, 120)

		gb := buttonInfo(t, tt, ui.RoleGroup, "Range").Bounds
		lx, hx, cy := sliderX(gb, 0.25), sliderX(gb, 0.75), gb.Y+gb.H/2
		b0 := builds

		tt.Move(lx, cy)
		if builds > b0+1 {
			t.Fatalf("RangeSlider Min hover enter caused %d builds (expected at most 1 Tester settle build)", builds-b0)
		}
		afterMin := builds
		tt.Advance(300 * time.Millisecond)
		if builds != afterMin {
			t.Fatalf("RangeSlider Min handle hover caused %d rebuilds during Advance, want 0", builds-afterMin)
		}
		if tt.FramePending() {
			t.Fatal("RangeSlider Min hover did not settle")
		}
		if px := tt.Image().RGBAAt(int(lx+8), int(cy-12)); px.R >= 253 {
			t.Fatalf("expected RangeSlider Min hover halo above track, got %+v", px)
		}

		tt.Move(hx, cy)
		if builds > afterMin+1 {
			t.Fatalf("RangeSlider Max hover enter caused %d builds (expected at most 1 Tester settle build)", builds-afterMin)
		}
		afterMax := builds
		tt.Advance(300 * time.Millisecond)
		if builds != afterMax {
			t.Fatalf("RangeSlider Max handle hover caused %d rebuilds during Advance, want 0", builds-afterMax)
		}
		if tt.FramePending() {
			t.Fatal("RangeSlider Max hover did not settle")
		}
		if px := tt.Image().RGBAAt(int(hx+8), int(cy-12)); px.R >= 253 {
			t.Fatalf("expected RangeSlider Max hover halo above track, got %+v", px)
		}

		tt.Move(2, 2)
		if builds > afterMax+1 {
			t.Fatalf("RangeSlider hover leave caused %d builds (expected at most 1 Tester settle build)", builds-afterMax)
		}
		afterLeave := builds
		tt.Advance(300 * time.Millisecond)
		if builds != afterLeave {
			t.Fatalf("RangeSlider hover leave caused %d rebuilds during Advance, want 0", builds-afterLeave)
		}
		if tt.FramePending() {
			t.Fatal("RangeSlider hover leave did not settle")
		}
		if px := tt.Image().RGBAAt(int(hx+8), int(cy-12)); px.R < 254 {
			t.Fatalf("RangeSlider Max halo lingered after hover leave: %+v", px)
		}

		tt.SetPreferences(ui.Preferences{ReduceMotion: true})
		bRM := builds
		tt.Move(lx, cy)
		if tt.FramePending() {
			t.Fatal("ReduceMotion RangeSlider hover enter left frames pending")
		}
		if builds > bRM+1 {
			t.Fatalf("ReduceMotion RangeSlider hover enter caused %d builds", builds-bRM)
		}
		bRM2 := builds
		tt.Move(2, 2)
		if tt.FramePending() {
			t.Fatal("ReduceMotion RangeSlider hover leave left frames pending")
		}
		if builds > bRM2+1 {
			t.Fatalf("ReduceMotion RangeSlider hover leave caused %d builds", builds-bRM2)
		}
	})
}
