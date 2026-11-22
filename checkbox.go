package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// Checkbox metrics in DIP: an 18 DIP box with 2 DIP corners inside a 40 DIP
// touch target, whose circular state layer spans the target.
const (
	checkboxBoxSize      = 18
	checkboxBoxRadius    = 2
	checkboxTouchTarget  = 40
	checkboxHaloDiameter = 40
)

// CheckboxOptions configures Checkbox and CheckboxState. The zero value is an
// enabled, valid check box with a 40 DIP touch target and no visible label.
type CheckboxOptions struct {
	// Label is the visible text after the box, and names the check box for
	// assistive technology.
	Label string
	// Disabled disables the check box: it ignores the pointer, the keyboard
	// and focus, and uses MD3E's disabled colors without ui's dimming.
	Disabled bool
	// Error paints the box with the error palette: an Error outline while
	// off, and an Error container with an OnError check while on or mixed.
	Error bool
	// ErrorMessage tells assistive technology why the value is invalid. With
	// Error set, the check box reports itself invalid, and the message as its
	// description.
	ErrorMessage string
	// TouchTarget overrides the 40 DIP square the box sits in; values below
	// the box's 18 DIP are raised to it.
	TouchTarget float32
}

// Checkbox builds a Material Design 3 Expressive check box over *checked: a
// ui.CheckboxBase row whose key is the caller's c.Key, a centered 18 DIP box
// with 2 DIP corners, the optional visible label, and MD3E hover, focus and
// press layers. Checking springs the box to Primary with an OnPrimary check
// mark drawn along its path, from 0 to its full length; unchecking trims it
// back, both with the Fast Spatial spring. Error paints the error palette and
// marks the box invalid for assistive technology.
//
// A click, or Space while it has the keyboard focus, toggles *checked:
//
//	md3e.Checkbox(c, &app.sync, md3e.CheckboxOptions{Label: "Sync across devices"})
func Checkbox(c *ui.Context, checked *bool, opts ...CheckboxOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Checkbox takes at most one CheckboxOptions")
	}
	if checked == nil {
		panic("md3e: Checkbox requires a non-nil *bool")
	}
	var o CheckboxOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	el := ui.CheckboxBase(c, checked)
	return buildCheckbox(c, el, *checked, false, o)
}

// CheckboxState builds a check box that can be mixed, as the "select all" box
// of a table's header, over *state: ui.CheckboxStateBase with the same look and
// springs as Checkbox. Off draws an outline, On fills the box and draws a check
// mark, and Mixed fills it and draws a dash growing from its middle, all with
// the Fast Spatial spring.
//
// A click, or Space while it has the keyboard focus, sets CheckOn from Off or
// Mixed, and CheckOff from On; assistive technology sees the mixed state:
//
//	all := ui.CheckOff
//	if n := app.selected(); n == len(app.rows) {
//		all = ui.CheckOn
//	} else if n > 0 {
//		all = ui.CheckMixed
//	}
//	md3e.CheckboxState(c, &all, md3e.CheckboxOptions{Label: "Select all"})
func CheckboxState(c *ui.Context, state *ui.CheckState, opts ...CheckboxOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: CheckboxState takes at most one CheckboxOptions")
	}
	if state == nil {
		panic("md3e: CheckboxState requires a non-nil *ui.CheckState")
	}
	var o CheckboxOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	el := ui.CheckboxStateBase(c, state)
	on := *state == ui.CheckOn
	mixed := *state == ui.CheckMixed
	return buildCheckbox(c, el, on, mixed, o)
}

// checkboxPaint is one frame of a check box's box.
type checkboxPaint struct {
	fill    float32
	check   float32
	dash    float32
	halo    float32
	haloInk ui.Color
	bgOff   ui.Color
	bgOn    ui.Color
	border       ui.Color
	borderHover  ui.Color
	ink          ui.Color
	ripple       *radioRippleState
	focused      bool
	reduceMotion bool
}

// buildCheckbox styles the base el, whose input ran as it was built, from the
// value and options.
func buildCheckbox(c *ui.Context, el ui.Element, on, mixed bool, o CheckboxOptions) ui.Element {
	th := CurrentTheme(c)
	cs := th.Colors
	rtl := c.Direction() == ui.RightToLeft

	target := o.TouchTarget
	if target < checkboxBoxSize {
		target = checkboxTouchTarget
	}
	el.DimDisabled(false).
		FocusRing(false).
		Height(target).
		Gap(8)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	ink := cs.OnSurface
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	el.TextColor(ink)
	name := o.Label
	if name != "" {
		el.Label(name)
	}
	if o.Error {
		if o.ErrorMessage != "" {
			el.Error(o.ErrorMessage)
		} else {
			el.Invalid(true)
		}
	}

	bgOff := ui.Transparent
	bgOn := cs.Primary
	content := cs.OnPrimary
	border := cs.OnSurfaceVariant
	borderHover := cs.OnSurface
	if o.Error {
		bgOn = cs.Error
		content = cs.OnError
		border = cs.Error
		borderHover = cs.Error
	}
	if disabled {
		// MD3 spec: disabled selected checkbox fills with OnSurface @ 38% and
		// cuts the checkmark out in Surface for crisp legibility.
		bgOn = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		content = cs.Surface
		border = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		borderHover = border
	}

	thReduceMotion := th.ReduceMotion || c.Preferences().ReduceMotion
	effects := th.Animation(motion.FastEffects)
	spatial := th.Animation(motion.FastSpatial)
	fill := animateValue(el, animChan0, boolFloat(on || mixed), effects, thReduceMotion)
	check := animateValue(el, animChan1, boolFloat(on), spatial, thReduceMotion)
	dash := animateValue(el, animChan2, boolFloat(mixed), spatial, thReduceMotion)

	haloSt := localHaloFade(el, animChan3)
	rp := localPtr(el, animChan4, newRadioRippleState)
	if b := el.Bounds(); b.W > 0 {
		rp.rowW, rp.rowH = b.W, b.H
	}
	if disabled {
		rp.active = false
	}
	el.ObserveInput(func(ev ui.InputEvent) {
		if disabled {
			return
		}
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return
			}
			ox, oy := target/2, target/2
			if rp.rowW > 0 {
				slotX := float32(0)
				if rtl {
					slotX = rp.rowW - target
				}
				slotY := (rp.rowH - target) / 2
				if ev.X >= slotX && ev.X <= slotX+target && ev.Y >= slotY && ev.Y <= slotY+target {
					ox = clampF32(ev.X-slotX, 0, target)
					oy = clampF32(ev.Y-slotY, 0, target)
				}
			}
			rp.active = true
			rp.ox, rp.oy, rp.start = ox, oy, ev.Time
		case ui.InputKeyDown:
			if ev.Key == ui.KeySpace && !ev.Repeat {
				rp.active = true
				rp.ox, rp.oy, rp.start = target/2, target/2, ev.Time
			}
		}
	})

	mark := localDrawPath(el)
	fillClamped := clampF32(fill, 0, 1)
	haloInk := cs.OnSurface.MixOklab(cs.Primary, fillClamped)
	if o.Error {
		haloInk = cs.Error
	}
	paintSnap := checkboxPaint{
		fill:         fillClamped,
		check:        clampF32(check, 0, 1),
		dash:         clampF32(dash, 0, 1),
		haloInk:      haloInk,
		bgOff:        bgOff,
		bgOn:         bgOn,
		border:       border,
		borderHover:  borderHover,
		ink:          content,
		ripple:       rp,
		reduceMotion: thReduceMotion,
	}

	el.Children(func() {
		ui.Box(c).Size(target, target).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			if !el.Valid() {
				ghost := paintSnap
				ghost.ripple = nil
				drawCheckbox(p, r, ghost, mark)
				return
			}
			hovered := el.Hovered() && !disabled
			pressed := !disabled && el.Pressed()
			focused := !disabled && el.FocusVisible()
			targetHalo := resolveStateLayerTarget(disabled, false, pressed, true, focused, hovered)
			if disabled {
				haloSt.reset()
			}
			rm := thReduceMotion || p.Preferences().ReduceMotion
			halo, needFrame := haloSt.step(targetHalo, p.Now(), rm)
			if needFrame && rectIntersects(r, p.Visible()) {
				p.After(16 * time.Millisecond)
			}
			paintCfg := paintSnap
			paintCfg.halo = clampF32(halo, 0, 1)
			paintCfg.focused = focused
			paintCfg.reduceMotion = rm
			drawCheckbox(p, r, paintCfg, mark)
		})
		if o.Label != "" {
			applyTypeStyleWithTheme(th, ui.Text(c, o.Label).NoWrap().TextColor(ink), tokens.LabelLarge)
		}
	})
	return el
}

// drawCheckbox paints the state layer, press wave, box and mark of a check box
// in its touch-target square r.
func drawCheckbox(p *ui.Painter, r ui.Rect, s checkboxPaint, mark *ui.Path) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	side := float32(checkboxBoxSize)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	haloR := min(float32(checkboxHaloDiameter)/2, min(r.W, r.H)/2)
	paintHalo(p, cx, cy, haloR, s.haloInk, s.halo)

	if rp := s.ripple; rp != nil && rp.active && !s.reduceMotion {
		elapsed := max(p.Now().Sub(rp.start), 0)
		if elapsed >= radioRippleTime {
			rp.active = false
		} else {
			t := float32(elapsed) / float32(radioRippleTime)
			inv := 1 - t
			grow := 1 - inv*inv*inv
			waveR := 5 + (haloR-5)*grow
			alpha := float32(tokens.StatePressedOpacity) * 1.4 * inv
			ox, oy := r.X+rp.ox, r.Y+rp.oy
			clip := ui.Rect{X: cx - haloR, Y: cy - haloR, W: 2 * haloR, H: 2 * haloR}
			p.Clip(clip, haloR, func() {
				p.Fill(ui.Rect{X: ox - waveR, Y: oy - waveR, W: 2 * waveR, H: 2 * waveR},
					s.haloInk.Alpha(alpha), waveR)
			})
			if rectIntersects(r, p.Visible()) {
				p.After(16 * time.Millisecond)
			}
		}
	}

	box := ui.Rect{X: cx - side/2, Y: cy - side/2, W: side, H: side}
	if s.focused {
		p.FocusRing(box, [4]float32{checkboxBoxRadius, checkboxBoxRadius, checkboxBoxRadius, checkboxBoxRadius})
	}
	bg := s.bgOff.MixOklab(s.bgOn, s.fill)
	if bg.A > 0 {
		p.Fill(box, bg, checkboxBoxRadius)
	}
	if a := clampF32(1-s.fill*1.8, 0, 1); a > 0.01 && s.border.A > 0 {
		hoverMix := clampF32(s.halo/float32(tokens.StateHoverOpacity), 0, 1)
		bCol := s.border.MixOklab(s.borderHover, hoverMix)
		p.Stroke(box, bCol.Alpha(a), checkboxBoxRadius, 2)
	}

	markAlpha := clampF32(s.fill*1.3, 0, 1)
	if markAlpha <= 0.01 {
		return
	}
	markInk := s.ink.Alpha(markAlpha)
	switch {
	case s.check > 0.001 && s.dash > 0.001:
		// Direct 3-point vector morph between indeterminate dash and checkmark.
		t := s.check / (s.check + s.dash)
		p0y := 0.50 + (0.53-0.50)*t
		p1y := 0.50 + (0.71-0.50)*t
		p2y := 0.50 + (0.31-0.50)*t
		mark.Reset()
		mark.MoveTo(box.X+box.W*0.24, box.Y+box.H*p0y).
			LineTo(box.X+box.W*0.42, box.Y+box.H*p1y).
			LineTo(box.X+box.W*0.76, box.Y+box.H*p2y)
		p.StrokePath(mark, 2, markInk)
	case s.check > 0.001:
		path := checkboxCheckPath(mark, box)
		path.Trim(0, s.check)
		p.StrokePath(path, 2, markInk)
	case s.dash > 0.001:
		d := s.dash
		path := checkboxDashPath(mark, box)
		path.Trim((1-d)/2, (1+d)/2)
		p.StrokePath(path, 2, markInk)
	}
}

// checkboxCheckPath returns the check mark of a box, in DIPs of box.
func checkboxCheckPath(p *ui.Path, box ui.Rect) *ui.Path {
	p.Reset()
	p.MoveTo(box.X+box.W*0.24, box.Y+box.H*0.53)
	p.LineTo(box.X+box.W*0.42, box.Y+box.H*0.71)
	p.LineTo(box.X+box.W*0.76, box.Y+box.H*0.31)
	return p
}

// checkboxDashPath returns the mixed dash of a box, in DIPs of box.
func checkboxDashPath(p *ui.Path, box ui.Rect) *ui.Path {
	p.Reset()
	y := box.Y + box.H/2
	p.MoveTo(box.X+box.W*0.24, y)
	p.LineTo(box.X+box.W*0.76, y)
	return p
}
