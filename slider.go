package md3e

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// Slider metrics in DIP, as MD3E specifies the Expressive slider: a 48 DIP row
// whose track is 16 high by default, a 4x44 vertical capsule handle, 6 DIP
// Track Gaps between the handle and both track segments, and a 38 DIP value
// indicator above it. The horizontal padding is half the handle's 24 DIP hit
// width, so the handle and the track's capsule ends stay inside the track.
const (
	sliderHeight          = 48
	sliderHitWidth        = 24
	sliderPad             = sliderHitWidth / 2
	sliderThumbWidth      = 4
	sliderThumbPressed    = 2
	sliderThumbHeight     = 44
	sliderTrackGap        = 6
	sliderTrackHeight     = 16
	sliderHaloDiameter    = 40
	sliderIndicatorHeight = 38
	sliderIndicatorTail   = 6
	sliderDefaultWidth    = 200
	sliderMaxStops        = 200
)

// SliderOptions configures Slider. The zero value is an enabled, continuous,
// uncentered slider with a 16 DIP track and value indicator.
type SliderOptions struct {
	// Label names the slider for assistive technology.
	Label string
	// Disabled disables the slider: it ignores the pointer, the keys and
	// focus, and uses MD3E's disabled colors without ui's dimming.
	Disabled bool
	// Step snaps the value to lo and its multiples, draws a stop dot at each,
	// and moves it by one step per arrow key. Zero keeps the value continuous.
	Step float64
	// Centered fills the active track from the range's middle to the handle
	// rather than from its low end, as a gain or balance slider.
	Centered bool
	// TrackHeight overrides the 16 DIP track height; it is clamped to the
	// slider's 48 DIP row.
	TrackHeight float32
	// ThumbHeight overrides the 44 DIP capsule handle height.
	ThumbHeight float32
	// FormatLabel formats the value shown in the indicator above the handle.
	// The default writes the step's decimals, two for a continuous slider.
	FormatLabel func(float64) string
}

// AnimateTo channels and locals of a slider.
type (
	sliderThumbWidthKey struct{}
	sliderHaloKey       struct{}
	sliderIndicatorKey  struct{}

	sliderPaint struct {
		value       float64
		lo, hi      float64
		step        float64
		centered    bool
		rtl         bool
		trackH      float32
		thumbH      float32
		thumbW      float32
		halo        float32
		haloInk     ui.Color
		active      ui.Color
		inactive    ui.Color
		thumb       ui.Color
		activeDot   ui.Color
		inactiveDot ui.Color
		focusRing   bool
	}
)

// Slider builds a Material Design 3 Expressive slider over *value between lo
// and hi: a ui.SliderBase whose key is the caller's c.Key, drawing a 16 DIP
// rounded track, a 4x44 vertical capsule handle with 6 DIP Track Gaps, stop
// dots for a discrete Step, and a pill value indicator above the handle while
// it is dragged or keyboard-focused. The handle narrows from 4 to 2 DIP as it
// is pressed or focused, with the Fast Spatial spring, and the indicator
// springs in and out around it in an overlay that inherits the active
// ThemeScope.
//
// Dragging the track, pressing it, or the arrows, Page Up, Page Down, Home and
// End while it has the focus set *value; Centered fills from the middle.
// Right to left, the low value sits at the right and the arrows mirror:
//
//	md3e.Slider(c, &app.volume, 0, 100, md3e.SliderOptions{
//		Label: "Volume", Step: 5,
//	})
func Slider(c *ui.Context, value *float64, lo, hi float64, opts ...SliderOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Slider takes at most one SliderOptions")
	}
	if value == nil {
		panic("md3e: Slider requires a non-nil *float64")
	}
	if math.IsNaN(lo) || math.IsInf(lo, 0) || math.IsNaN(hi) || math.IsInf(hi, 0) || !(hi > lo) {
		panic("md3e: Slider requires finite hi > lo")
	}
	var o SliderOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	step := o.Step
	if math.IsNaN(step) || math.IsInf(step, 0) || step <= 0 {
		step = 0
	}
	*value = clampRangeValue(*value, lo, hi)

	th := CurrentTheme(c)
	cs := th.Colors
	rtl := c.Direction() == ui.RightToLeft

	el := ui.SliderBase(c, value, lo, hi).
		DimDisabled(false).
		FocusRing(false).
		Height(sliderHeight).
		MinWidth(sliderDefaultWidth).
		PaddingX(sliderPad)
	if step > 0 {
		el.Step(step)
	}
	if o.Label != "" {
		el.Label(o.Label)
	}
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	btn, ok := el.PressedButton()
	_, _, dragging := el.Dragged()
	held := !disabled && ((ok && btn == 0) || dragging)

	thReduceMotion := th.ReduceMotion
	spatial := th.Animation(motion.FastSpatial)

	activeColor := cs.Primary
	inactiveColor := cs.SurfaceContainerHighest
	thumbColor := cs.Primary
	activeDot := cs.OnPrimary
	inactiveDot := cs.OnSurfaceVariant
	haloInk := cs.OnSurfaceVariant
	if disabled {
		activeColor = cs.OnSurface.Alpha(0.38)
		inactiveColor = cs.OnSurface.Alpha(0.12)
		thumbColor = cs.OnSurface.Alpha(0.38)
		activeDot = cs.OnSurface.Alpha(0.12)
		inactiveDot = cs.OnSurface.Alpha(0.38)
	}

	trackH := o.TrackHeight
	if math.IsNaN(float64(trackH)) || math.IsInf(float64(trackH), 0) || !(trackH > 0) {
		trackH = sliderTrackHeight
	}
	trackH = clampF32(trackH, 4, sliderHeight-8)
	thumbH := o.ThumbHeight
	if math.IsNaN(float64(thumbH)) || math.IsInf(float64(thumbH), 0) || !(thumbH > 0) {
		thumbH = sliderThumbHeight
	}
	thumbH = clampF32(thumbH, 8, sliderHeight)

	focused := el.FocusVisible()
	engaged := !disabled && (held || focused)
	thumbTarget := float32(sliderThumbWidth)
	if engaged {
		thumbTarget = sliderThumbPressed
	}
	thumbW := animateValue(el, sliderThumbWidthKey{}, thumbTarget, spatial, thReduceMotion)
	indicator := animateValue(el, sliderIndicatorKey{}, boolFloat(engaged), spatial, thReduceMotion)

	haloSt := ui.Local(el, sliderHaloKey{}, func() haloFadeState { return haloFadeState{} })
	haloBase := resolveStateLayerTarget(disabled, held, held, true, focused, false)

	paintSnap := sliderPaint{
		value:       *value,
		lo:          lo,
		hi:          hi,
		step:        step,
		centered:    o.Centered,
		rtl:         rtl,
		trackH:      trackH,
		thumbH:      thumbH,
		thumbW:      clampF32(thumbW, 1, thumbH),
		haloInk:     haloInk,
		active:      activeColor,
		inactive:    inactiveColor,
		thumb:       thumbColor,
		activeDot:   activeDot,
		inactiveDot: inactiveDot,
		focusRing:   focused && !disabled,
	}
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !el.Valid() {
			return
		}
		targetHalo := haloBase
		hovered := el.Hovered()
		if !disabled && targetHalo < float32(tokens.StatePressedOpacity) {
			if el.Pressed() {
				targetHalo = float32(tokens.StatePressedOpacity)
			} else if targetHalo < float32(tokens.StateHoverOpacity) && hovered {
				targetHalo = float32(tokens.StateHoverOpacity)
			}
		}
		if disabled {
			haloSt.reset()
		}
		alpha, needFrame := haloSt.step(targetHalo, p.Now(), thReduceMotion || p.Preferences().ReduceMotion)
		if needFrame {
			p.After(16 * time.Millisecond)
		}
		paintCfg := paintSnap
		paintCfg.halo = clampF32(alpha, 0, 1)
		drawSlider(p, r, paintCfg)
	})

	if indicator > 0.02 {
		if b := el.Bounds(); b.W > 2*sliderPad {
			f := sliderFraction(*value, lo, hi)
			if rtl {
				f = 1 - f
			}
			tx := b.X + sliderPad + (b.W-2*sliderPad)*f
			text := sliderValueText(*value, step, o.FormatLabel)
			showValueIndicator(c, th, text, tx, b.Y, indicator)
		}
	}
	return el
}

// sliderFraction returns (v-lo)/(hi-lo) clamped to 0..1.
func sliderFraction(v, lo, hi float64) float32 {
	if math.IsNaN(v) || math.IsNaN(lo) || math.IsNaN(hi) || math.IsInf(lo, 0) || math.IsInf(hi, 0) || !(hi > lo) {
		return 0
	}
	if v <= lo {
		return 0
	}
	if v >= hi {
		return 1
	}
	return clampF32(float32((v-lo)/(hi-lo)), 0, 1)
}

// drawSlider paints a slider's track, stop dots, state layer, focus cue and
// capsule handle in the row r.
func drawSlider(p *ui.Painter, r ui.Rect, s sliderPaint) {
	w := r.W - 2*sliderPad
	if w <= 0 || r.H <= 0 {
		return
	}
	x0 := r.X + sliderPad
	x1 := x0 + w
	cy := r.Y + r.H/2
	f := sliderFraction(s.value, s.lo, s.hi)
	if s.rtl {
		f = 1 - f
	}
	tx := x0 + w*f
	half := s.thumbW / 2
	leftEnd := clampF32(tx-half-sliderTrackGap, x0, x1)
	rightStart := clampF32(tx+half+sliderTrackGap, x0, x1)

	// The track keeps a clear gap on both sides of the handle. Its pieces do
	// not overlap, so a disabled active track at 38% does not darken over the
	// 12% inactive one.
	var a0, a1 float32
	switch {
	case s.centered:
		mid := x0 + w/2
		if tx >= mid {
			fillTrackPiece(p, x0, mid, cy, s.trackH, s.inactive)
			fillTrackPiece(p, rightStart, x1, cy, s.trackH, s.inactive)
			a0, a1 = mid, leftEnd
		} else {
			fillTrackPiece(p, x0, leftEnd, cy, s.trackH, s.inactive)
			fillTrackPiece(p, mid, x1, cy, s.trackH, s.inactive)
			a0, a1 = rightStart, mid
		}
	case s.rtl:
		// Right to left, the low end is at the right: the active track runs
		// from there to the handle.
		fillTrackPiece(p, x0, leftEnd, cy, s.trackH, s.inactive)
		a0, a1 = rightStart, x1
	default:
		fillTrackPiece(p, rightStart, x1, cy, s.trackH, s.inactive)
		a0, a1 = x0, leftEnd
	}
	fillTrackPiece(p, a0, a1, cy, s.trackH, s.active)

	drawSliderStops(p, s, x0, w, cy, tx, half, a0, a1)

	if s.halo > 0.001 {
		paintHalo(p, tx, cy, sliderHaloDiameter/2, s.haloInk, s.halo)
	}
	thumb := ui.Rect{X: tx - half, Y: cy - s.thumbH/2, W: s.thumbW, H: s.thumbH}
	if s.focusRing {
		ring := ui.Rect{
			X: thumb.X - 4,
			Y: thumb.Y - 4,
			W: thumb.W + 8,
			H: thumb.H + 8,
		}
		p.Stroke(ring, s.active.Alpha(0.6), ring.W/2, 1.5)
	}
	p.Fill(thumb, s.thumb, half)
}

// fillTrackPiece paints one rounded piece of a track centered on cy.
func fillTrackPiece(p *ui.Painter, xa, xb, cy, h float32, col ui.Color) {
	if xb <= xa || col.A == 0 {
		return
	}
	w := xb - xa
	radius := min(h/2, w/2)
	p.Fill(ui.Rect{X: xa, Y: cy - h/2, W: w, H: h}, col, radius)
}

// drawSliderStops paints a dot at each step of a discrete slider, in the
// active or inactive dot color as the active track covers it.
func drawSliderStops(p *ui.Painter, s sliderPaint, x0, w, cy, tx, half, a0, a1 float32) {
	if s.step <= 0 {
		return
	}
	span := s.hi - s.lo
	n := int(span/s.step + 1e-9)
	if n <= 0 || n > sliderMaxStops {
		return
	}
	dotR := float32(1.5)
	gapL := tx - half - sliderTrackGap
	gapR := tx + half + sliderTrackGap
	lo, hi := min(a0, a1), max(a0, a1)
	for i := 1; i < n; i++ {
		f := float32((float64(i) * s.step) / span)
		if s.rtl {
			f = 1 - f
		}
		x := x0 + w*f
		if x >= gapL && x <= gapR {
			continue
		}
		col := s.inactiveDot
		if x >= lo && x <= hi {
			col = s.activeDot
		}
		if col.A > 0 {
			p.Fill(ui.Rect{X: x - dotR, Y: cy - dotR, W: 2 * dotR, H: 2 * dotR}, col, dotR)
		}
	}
}

// sliderValueText formats v for the value indicator: the step's decimals, or
// two trimmed decimals for a continuous slider.
func sliderValueText(v, step float64, format func(float64) string) string {
	if format != nil {
		return format(v)
	}
	decimals := 2
	if step > 0 {
		decimals = 0
		s := step
		for decimals < 4 && math.Abs(s-math.Round(s)) > 1e-9 {
			s *= 10
			decimals++
		}
	}
	text := strconv.FormatFloat(v, 'f', decimals, 64)
	if step <= 0 {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
		if text == "" || text == "-0" {
			text = "0"
		}
	}
	return text
}

// showValueIndicator builds the overlay above a slider's handle: a pill with a
// teardrop tail, in InverseSurface with InverseOnSurface text, that springs
// from scale 0.6 to its size with progress. The overlay passes the pointer
// through and inherits the active ThemeScope.
func showValueIndicator(c *ui.Context, th *Theme, text string, cx, anchorTop, progress float32) {
	tail := float32(sliderIndicatorTail)
	w := clampF32(24+float32(utf8.RuneCountInString(text))*8, 40, 180)
	h := float32(sliderIndicatorHeight)
	top := anchorTop - h - 8
	if top < 0 {
		top = 0
	}
	scale := 0.6 + 0.4*clampF32(progress, 0, 1.15)
	opacity := clampF32(progress, 0, 1)
	bg := th.Colors.InverseSurface
	fg := th.Colors.InverseOnSurface
	shadow := th.Colors.Shadow
	if shadow.A == 0 {
		shadow = ui.RGB(0, 0, 0)
	}

	ui.Overlay(c, func() {
		box := ui.Box(c).
			Absolute().
			Left(cx-w/2).
			Top(top).
			Size(w, h).
			PassThrough().
			Scale(scale, scale).
			Opacity(opacity).
			TransformOrigin(0.5, 1)
		tip := localDrawPath(box)
		box.Draw(func(p *ui.Painter, r ui.Rect) {
			if r.W <= 0 || r.H <= 0 {
				return
			}
			pill := ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H - tail}
			radius := pill.H / 2
			p.Shadow(pill, radius, 0, 2, 6, 0, shadow.Alpha(0.18))
			p.Fill(pill, bg, radius)
			tip.Reset()
			tip.MoveTo(r.X+r.W/2-5, pill.Y+pill.H-0.5)
			tip.LineTo(r.X+r.W/2+5, pill.Y+pill.H-0.5)
			tip.LineTo(r.X+r.W/2, r.Y+r.H)
			tip.Close()
			p.FillPath(tip, bg)

			spans := []ui.Span{{Text: text, Size: 12, Weight: 500, Color: fg}}
			tw, th := p.MeasureText(0, spans...)
			p.RichText(r.X+(r.W-tw)/2, r.Y+(pill.H-th)/2, 0, spans...)
		})
	})
}

// RangeSliderOptions configures RangeSlider. The zero value is an enabled,
// continuous range slider named "Min" and "Max" with a 16 DIP track.
type RangeSliderOptions struct {
	// Label names the group of the two handles for assistive technology.
	Label string
	// MinLabel and MaxLabel name the low and high handles; "Min" and "Max" by
	// default.
	MinLabel, MaxLabel string
	// Disabled disables the range slider and both handles.
	Disabled bool
	// Step snaps both values to lo and its multiples, draws a stop dot at
	// each, and moves the focused handle by one step per key. Zero keeps the
	// values continuous.
	Step float64
	// TrackHeight overrides the 16 DIP track height.
	TrackHeight float32
	// ThumbHeight overrides the 44 DIP capsule handle height.
	ThumbHeight float32
	// FormatLabel formats the values shown in the indicators.
	FormatLabel func(float64) string
}

// Range drag channels and locals.
type (
	rangeDragKey      struct{}
	rangeLowWidthKey  struct{}
	rangeHighWidthKey struct{}
	rangeHalosKey     struct{}
	rangeIndicatorKey struct{}

	rangeDragState  struct{ knob int }
	rangeHalosState struct{ low, high haloFadeState }

	rangePaint struct {
		low, high   float64
		lo, hi      float64
		step        float64
		rtl         bool
		trackH      float32
		thumbH      float32
		lowW        float32
		highW       float32
		lowHalo     float32
		highHalo    float32
		haloInk     ui.Color
		active      ui.Color
		inactive    ui.Color
		thumb       ui.Color
		activeDot   ui.Color
		inactiveDot ui.Color
		lowFocus    bool
		highFocus   bool
	}
)

// RangeSlider builds a Material Design 3 Expressive range slider over *low and
// *high between lo and hi, as a price range: two independently focusable
// capsule handles, each a RoleSlider named by MinLabel and MaxLabel, that keep
// *low <= *high. Pressing the track drags the handle nearest the pointer (and
// on equal values the one the drag moves toward, so they never lock); the
// arrows, Page Up, Page Down, Home and End move the focused handle. The active
// track spans the handles with 6 DIP Track Gaps, a discrete Step draws stop
// dots, and the dragged or focused handle shows its value in a teardrop pill.
// Track: the row mirrors right to left, the low value at the right.
//
//	md3e.RangeSlider(c, &app.min, &app.max, 0, 1000, md3e.RangeSliderOptions{
//		Label: "Price", Step: 10,
//	})
func RangeSlider(c *ui.Context, low, high *float64, lo, hi float64, opts ...RangeSliderOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: RangeSlider takes at most one RangeSliderOptions")
	}
	if low == nil || high == nil {
		panic("md3e: RangeSlider requires non-nil *float64 handles")
	}
	if math.IsNaN(lo) || math.IsInf(lo, 0) || math.IsNaN(hi) || math.IsInf(hi, 0) || !(hi > lo) {
		panic("md3e: RangeSlider requires finite hi > lo")
	}
	var o RangeSliderOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	step := o.Step
	if math.IsNaN(step) || math.IsInf(step, 0) || step <= 0 {
		step = 0
	}

	l := clampRangeValue(*low, lo, hi)
	h := clampRangeValue(*high, lo, hi)
	if l > h {
		l = h
	}
	*low, *high = l, h

	th := CurrentTheme(c)
	cs := th.Colors
	rtl := c.Direction() == ui.RightToLeft

	el := ui.Box(c).
		Height(sliderHeight).
		MinWidth(sliderDefaultWidth).
		PaddingX(sliderPad).
		DimDisabled(false).
		Role(ui.RoleGroup)
	if o.Label != "" {
		el.Label(o.Label)
	}
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	minName := labelOr(o.MinLabel, "Min")
	maxName := labelOr(o.MaxLabel, "Max")
	keyStep := step
	if keyStep <= 0 {
		keyStep = (hi - lo) / 100
	}
	pageStep := math.Max(keyStep, (hi-lo)/10)

	var lowEl, highEl ui.Element
	el.Children(func() {
		lowEl = rangeThumb(c, minName, low, lo, hi, lo, *high, keyStep, pageStep, rtl)
		highEl = rangeThumb(c, maxName, high, lo, hi, *low, hi, keyStep, pageStep, rtl)
	})

	drags := ui.Local(el, rangeDragKey{}, func() rangeDragState { return rangeDragState{knob: -1} })
	if !disabled {
		var ldx, hdx, cdx float32
		var lowDrag, highDrag, containerDrag bool
		if lowEl.Valid() {
			ldx, _, lowDrag = lowEl.Dragged()
		}
		if highEl.Valid() {
			hdx, _, highDrag = highEl.Dragged()
		}
		cdx, _, containerDrag = el.Dragged()
		anyDrag := lowDrag || highDrag || containerDrag
		dragDX := cdx
		if lowDrag {
			dragDX = ldx
		} else if highDrag {
			dragDX = hdx
		}
		if !anyDrag {
			drags.knob = -1
		} else if b := el.Bounds(); b.W > 2*sliderPad {
			px, _, _ := el.PointerPosition()
			contentW := b.W - 2*sliderPad
			f := clampF32((px-sliderPad)/contentW, 0, 1)
			if rtl {
				f = 1 - f
			}
			pv := lo + float64(f)*(hi-lo)
			if drags.knob < 0 {
				if *low == *high {
					// When both handles coincide, highEl overlays lowEl in z-order. Do not
					// let highEl blindly preempt the drag; decide by drag direction first.
					delta := dragDX
					if rtl {
						delta = -delta
					}
					switch {
					case delta < 0:
						if *low > lo {
							drags.knob = 0
						}
					case delta > 0:
						if *high < hi {
							drags.knob = 1
						}
					case !lowDrag && !highDrag && containerDrag:
						if pv < *low {
							drags.knob = 0
						} else if pv > *high {
							drags.knob = 1
						}
					}
				} else {
					switch {
					case lowDrag:
						drags.knob = 0
					case highDrag:
						drags.knob = 1
					case math.Abs(pv-*low) <= math.Abs(pv-*high):
						drags.knob = 0
					default:
						drags.knob = 1
					}
				}
				if drags.knob == 0 && lowEl.Valid() {
					lowEl.Focus(ui.FocusOptions{PreventScroll: true})
				} else if drags.knob == 1 && highEl.Valid() {
					highEl.Focus(ui.FocusOptions{PreventScroll: true})
				}
			}
			switch drags.knob {
			case 0:
				*low = clampRangeValue(snapRangeValue(pv, lo, hi, step), lo, *high)
			case 1:
				*high = clampRangeValue(snapRangeValue(pv, lo, hi, step), *low, hi)
			}
		}
	} else {
		drags.knob = -1
	}

	if lowEl.Valid() {
		lf := sliderFraction(*low, lo, hi)
		lowEl.StartPercent(lf*100).
			MarginStart(-sliderHitWidth*lf).
			Range(lo, *high, *low)
	}
	if highEl.Valid() {
		hf := sliderFraction(*high, lo, hi)
		highEl.StartPercent(hf*100).
			MarginStart(-sliderHitWidth*hf).
			Range(*low, hi, *high)
	}

	thReduceMotion := th.ReduceMotion
	spatial := th.Animation(motion.FastSpatial)

	var lowFocused, highFocused bool
	if lowEl.Valid() {
		lowFocused = lowEl.FocusVisible()
	}
	if highEl.Valid() {
		highFocused = highEl.FocusVisible()
	}
	lowW := animateValue(el, rangeLowWidthKey{}, rangeThumbTarget(lowFocused, drags.knob == 0), spatial, thReduceMotion)
	highW := animateValue(el, rangeHighWidthKey{}, rangeThumbTarget(highFocused, drags.knob == 1), spatial, thReduceMotion)

	activeKnob := -1
	switch {
	case drags.knob >= 0:
		activeKnob = drags.knob
	case lowFocused:
		activeKnob = 0
	case highFocused:
		activeKnob = 1
	}
	engaged := !disabled && activeKnob >= 0
	indicator := animateValue(el, rangeIndicatorKey{}, boolFloat(engaged), spatial, thReduceMotion)

	halosSt := ui.Local(el, rangeHalosKey{}, func() rangeHalosState { return rangeHalosState{} })
	lowHaloBase := rangeHaloTarget(disabled, false, lowFocused, drags.knob == 0)
	highHaloBase := rangeHaloTarget(disabled, false, highFocused, drags.knob == 1)

	activeColor := cs.Primary
	inactiveColor := cs.SurfaceContainerHighest
	thumbColor := cs.Primary
	activeDot := cs.OnPrimary
	inactiveDot := cs.OnSurfaceVariant
	haloInk := cs.OnSurfaceVariant
	if disabled {
		activeColor = cs.OnSurface.Alpha(0.38)
		inactiveColor = cs.OnSurface.Alpha(0.12)
		thumbColor = cs.OnSurface.Alpha(0.38)
		activeDot = cs.OnSurface.Alpha(0.12)
		inactiveDot = cs.OnSurface.Alpha(0.38)
	}

	trackH := o.TrackHeight
	if math.IsNaN(float64(trackH)) || math.IsInf(float64(trackH), 0) || !(trackH > 0) {
		trackH = sliderTrackHeight
	}
	trackH = clampF32(trackH, 4, sliderHeight-8)
	thumbH := o.ThumbHeight
	if math.IsNaN(float64(thumbH)) || math.IsInf(float64(thumbH), 0) || !(thumbH > 0) {
		thumbH = sliderThumbHeight
	}
	thumbH = clampF32(thumbH, 8, sliderHeight)

	paintSnap := rangePaint{
		low:         *low,
		high:        *high,
		lo:          lo,
		hi:          hi,
		step:        step,
		rtl:         rtl,
		trackH:      trackH,
		thumbH:      thumbH,
		lowW:        clampF32(lowW, 1, thumbH),
		highW:       clampF32(highW, 1, thumbH),
		haloInk:     haloInk,
		active:      activeColor,
		inactive:    inactiveColor,
		thumb:       thumbColor,
		activeDot:   activeDot,
		inactiveDot: inactiveDot,
		lowFocus:    lowFocused && !disabled,
		highFocus:   highFocused && !disabled,
	}
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !el.Valid() {
			return
		}
		drawRangeTrack(p, r, paintSnap)
	})
	// Halos and thumbs paint in DrawOver so reading child Hovered()/Pressed()
	// happens after p.element(lowEl)/p.element(highEl) clears per-element hoverLook.
	el.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !el.Valid() {
			return
		}
		lowTarget := lowHaloBase
		if lowEl.Valid() {
			lowHover := lowEl.Hovered()
			if !disabled && lowTarget < float32(tokens.StatePressedOpacity) {
				if lowEl.Pressed() {
					lowTarget = float32(tokens.StatePressedOpacity)
				} else if lowTarget < float32(tokens.StateHoverOpacity) && lowHover {
					lowTarget = float32(tokens.StateHoverOpacity)
				}
			}
		}
		highTarget := highHaloBase
		if highEl.Valid() {
			highHover := highEl.Hovered()
			if !disabled && highTarget < float32(tokens.StatePressedOpacity) {
				if highEl.Pressed() {
					highTarget = float32(tokens.StatePressedOpacity)
				} else if highTarget < float32(tokens.StateHoverOpacity) && highHover {
					highTarget = float32(tokens.StateHoverOpacity)
				}
			}
		}
		if disabled {
			halosSt.low.reset()
			halosSt.high.reset()
		}
		rm := thReduceMotion || p.Preferences().ReduceMotion
		now := p.Now()
		lowAlpha, lowNeed := halosSt.low.step(lowTarget, now, rm)
		highAlpha, highNeed := halosSt.high.step(highTarget, now, rm)
		if lowNeed || highNeed {
			p.After(16 * time.Millisecond)
		}
		paintCfg := paintSnap
		paintCfg.lowHalo = clampF32(lowAlpha, 0, 1)
		paintCfg.highHalo = clampF32(highAlpha, 0, 1)
		drawRangeThumbs(p, r, paintCfg)
	})

	if indicator > 0.02 && activeKnob >= 0 {
		if b := el.Bounds(); b.W > 2*sliderPad {
			v := *high
			if activeKnob == 0 {
				v = *low
			}
			f := sliderFraction(v, lo, hi)
			if rtl {
				f = 1 - f
			}
			tx := b.X + sliderPad + (b.W-2*sliderPad)*f
			text := sliderValueText(v, step, o.FormatLabel)
			showValueIndicator(c, th, text, tx, b.Y, indicator)
		}
	}
	return el
}

// rangeThumb builds one focusable handle of a RangeSlider: a transparent
// 24x44 hit box placed over the value's fraction, named name, with the slider
// range for assistive technology and the keys that move it between from and to.
func rangeThumb(c *ui.Context, name string, value *float64, lo, hi, from, to, step, page float64, rtl bool) ui.Element {
	// StartPercent counts from the start side, which mirrors right to left on
	// its own, so the value's own fraction places the handle either way.
	fd := sliderFraction(*value, lo, hi)
	k := ui.Box(c).
		Absolute().
		Top((sliderHeight-sliderThumbHeight)/2).
		Height(sliderThumbHeight).
		Width(sliderHitWidth).
		StartPercent(fd*100).
		MarginStart(-sliderHitWidth*fd).
		Focusable().
		FocusRing(false).
		Role(ui.RoleSlider).
		Label(name).
		Range(from, to, *value).
		Step(step)

	back, forth := ui.KeyLeft, ui.KeyRight
	if rtl {
		back, forth = forth, back
	}
	next := *value
	changed := true
	switch {
	case k.Shortcut(0, back), k.Shortcut(0, ui.KeyDown):
		next -= step
	case k.Shortcut(0, forth), k.Shortcut(0, ui.KeyUp):
		next += step
	case k.Shortcut(0, ui.KeyPageDown):
		next -= page
	case k.Shortcut(0, ui.KeyPageUp):
		next += page
	case k.Shortcut(0, ui.KeyHome):
		next = from
	case k.Shortcut(0, ui.KeyEnd):
		next = to
	default:
		changed = false
	}
	if changed {
		next = clampRangeValue(snapRangeValue(next, lo, hi, step), from, to)
		if next != *value {
			*value = next
		}
	}
	return k
}

// rangeThumbTarget returns the capsule width of a handle that is focused or
// dragged, 2 DIP, or its resting 4 DIP.
func rangeThumbTarget(focused, dragged bool) float32 {
	if focused || dragged {
		return sliderThumbPressed
	}
	return sliderThumbWidth
}

// rangeHaloTarget returns the state layer opacity of a handle.
func rangeHaloTarget(disabled, hovered, focused, dragged bool) float32 {
	if disabled {
		return 0
	}
	switch {
	case dragged, focused:
		return float32(tokens.StateFocusOpacity)
	case hovered:
		return float32(tokens.StateHoverOpacity)
	}
	return 0
}

func rangeThumbPositions(r ui.Rect, s rangePaint) (w, x0, cy, lx, hx float32, ok bool) {
	w = r.W - 2*sliderPad
	if w <= 0 || r.H <= 0 {
		return 0, 0, 0, 0, 0, false
	}
	x0 = r.X + sliderPad
	cy = r.Y + r.H/2
	fl := sliderFraction(s.low, s.lo, s.hi)
	fh := sliderFraction(s.high, s.lo, s.hi)
	if s.rtl {
		fl, fh = 1-fl, 1-fh
	}
	lx = x0 + w*fl
	hx = x0 + w*fh
	return w, x0, cy, lx, hx, true
}

// drawRangeTrack paints a range slider's active/inactive track segments and stop dots.
func drawRangeTrack(p *ui.Painter, r ui.Rect, s rangePaint) {
	w, x0, cy, lx, hx, ok := rangeThumbPositions(r, s)
	if !ok {
		return
	}
	x1 := x0 + w
	leftX, rightX := min(lx, hx), max(lx, hx)
	var leftW, rightW float32
	if lx <= hx {
		leftW, rightW = s.lowW, s.highW
	} else {
		leftW, rightW = s.highW, s.lowW
	}
	if lx == hx {
		leftW = max(s.lowW, s.highW)
		rightW = leftW
	}
	inactiveL := clampF32(leftX-leftW/2-sliderTrackGap, x0, x1)
	inactiveR := clampF32(rightX+rightW/2+sliderTrackGap, x0, x1)
	fillTrackPiece(p, x0, inactiveL, cy, s.trackH, s.inactive)
	fillTrackPiece(p, inactiveR, x1, cy, s.trackH, s.inactive)
	a0 := clampF32(leftX+leftW/2+sliderTrackGap, x0, x1)
	a1 := clampF32(rightX-rightW/2-sliderTrackGap, x0, x1)
	fillTrackPiece(p, a0, a1, cy, s.trackH, s.active)

	if s.step > 0 {
		span := s.hi - s.lo
		n := int(span/s.step + 1e-9)
		if n > 0 && n <= sliderMaxStops {
			dotR := float32(1.5)
			lo, hi := min(a0, a1), max(a0, a1)
			for i := 1; i < n; i++ {
				f := float32((float64(i) * s.step) / span)
				if s.rtl {
					f = 1 - f
				}
				x := x0 + w*f
				if x >= lx-s.lowW/2-sliderTrackGap && x <= lx+s.lowW/2+sliderTrackGap {
					continue
				}
				if x >= hx-s.highW/2-sliderTrackGap && x <= hx+s.highW/2+sliderTrackGap {
					continue
				}
				col := s.inactiveDot
				if x >= lo && x <= hi {
					col = s.activeDot
				}
				if col.A > 0 {
					p.Fill(ui.Rect{X: x - dotR, Y: cy - dotR, W: 2 * dotR, H: 2 * dotR}, col, dotR)
				}
			}
		}
	}
}

// drawRangeThumbs paints the two capsule handles, focus rings, and state layer halos.
func drawRangeThumbs(p *ui.Painter, r ui.Rect, s rangePaint) {
	_, _, cy, lx, hx, ok := rangeThumbPositions(r, s)
	if !ok {
		return
	}
	if s.lowHalo > 0.001 {
		paintHalo(p, lx, cy, sliderHaloDiameter/2, s.haloInk, s.lowHalo)
	}
	if s.highHalo > 0.001 {
		paintHalo(p, hx, cy, sliderHaloDiameter/2, s.haloInk, s.highHalo)
	}
	if s.lowFocus {
		drawThumbFocus(p, lx, cy, s.lowW, s.thumbH, s.active)
	}
	if s.highFocus {
		drawThumbFocus(p, hx, cy, s.highW, s.thumbH, s.active)
	}
	p.Fill(ui.Rect{X: lx - s.lowW/2, Y: cy - s.thumbH/2, W: s.lowW, H: s.thumbH}, s.thumb, s.lowW/2)
	p.Fill(ui.Rect{X: hx - s.highW/2, Y: cy - s.thumbH/2, W: s.highW, H: s.thumbH}, s.thumb, s.highW/2)
}

// drawThumbFocus paints the keyboard-focus cue around a handle.
func drawThumbFocus(p *ui.Painter, cx, cy, thumbW, thumbH float32, ink ui.Color) {
	ring := ui.Rect{X: cx - thumbW/2 - 4, Y: cy - thumbH/2 - 4, W: thumbW + 8, H: thumbH + 8}
	p.Stroke(ring, ink.Alpha(0.6), ring.W/2, 1.5)
}

// snapRangeValue snaps v to lo and its multiples of step, as ui's sliders do,
// leaving v alone for a step of zero.
func snapRangeValue(v, lo, hi, step float64) float64 {
	if math.IsNaN(v) {
		return lo
	}
	if math.IsNaN(step) || math.IsInf(step, 0) || step <= 0 {
		return clampRangeValue(v, lo, hi)
	}
	v = lo + math.Round((v-lo)/step)*step
	if v > hi {
		v -= step
	}
	return clampRangeValue(v, lo, hi)
}

// clampRangeValue limits v to the closed interval lo..hi.
func clampRangeValue(v, lo, hi float64) float64 {
	if math.IsNaN(v) || v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
