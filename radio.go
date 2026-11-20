package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// Radio metrics in DIP: a 20 DIP ring with a 10 DIP inner dot, inside a 40 DIP
// touch target whose circular state layer spans the target.
const (
	radioRingSize     = 20
	radioRingWidth    = 2
	radioDotSize      = 10
	radioTouchTarget  = 40
	radioHaloDiameter = 40
)

// RadioOptions configures Radio. The zero value is an enabled radio button
// with a 40 DIP touch target.
type RadioOptions struct {
	// Disabled disables the radio button: it ignores the pointer, the keyboard
	// and focus, and uses MD3E's disabled colors without ui's dimming.
	Disabled bool
	// Label overrides the accessible name. The visible label names the button
	// by default.
	Label string
	// TouchTarget overrides the 40 DIP square the ring sits in; values below
	// the ring's 20 DIP are raised to it.
	TouchTarget float32
}

// AnimateTo channels of a radio button.
type (
	radioSelectKey struct{}
	radioHaloKey   struct{}
)

// Radio builds a Material Design 3 Expressive radio button over *selected,
// choosing value: a ui.RadioBase[T] row whose key is the caller's c.Key, a
// 20 DIP ring with a 10 DIP dot that springs into its middle when chosen, the
// visible label, and MD3E hover, focus and press layers. The ring turns from
// OnSurfaceVariant to Primary as the dot arrives with the Fast Spatial spring.
//
// A click, or Space while it has the keyboard focus, selects value; assistive
// technology sees a radio button, on while *selected is value. Put radios in a
// RadioGroup for one Tab stop and arrow-key navigation:
//
//	md3e.Radio(c, &app.size, "Large", "Large")
func Radio[T comparable](c *ui.Context, selected *T, value T, label string, opts ...RadioOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Radio takes at most one RadioOptions")
	}
	if selected == nil {
		panic("md3e: Radio requires a non-nil *T")
	}
	var o RadioOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors

	el := ui.RadioBase(c, selected, value).
		DimDisabled(false).
		FocusRing(false)
	target := o.TouchTarget
	if target < radioRingSize {
		target = radioTouchTarget
	}
	el.Height(target).Gap(8)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	ink := cs.OnSurface
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	el.TextColor(ink)
	name := label
	if o.Label != "" {
		name = o.Label
	}
	if name != "" {
		el.Label(name)
	}

	on := *selected == value
	ringOff := cs.OnSurfaceVariant
	ringOn := cs.Primary
	dotInk := cs.Primary
	if disabled {
		ringOff = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		ringOn = ringOff
		dotInk = ringOff
	}

	spatial := th.Animation(motion.FastSpatial)
	effects := th.Animation(motion.FastEffects)
	if th.ReduceMotion {
		spatial, effects = motion.Snap, motion.Snap
	}
	sel := el.AnimateTo(radioSelectKey{}, boolFloat(on), spatial)

	haloTarget := float32(0)
	switch {
	case disabled:
	case el.Pressed():
		haloTarget = float32(tokens.StatePressedOpacity)
	case el.FocusVisible():
		haloTarget = float32(tokens.StateFocusOpacity)
	case el.Hovered():
		haloTarget = float32(tokens.StateHoverOpacity)
	}
	halo := el.AnimateTo(radioHaloKey{}, haloTarget, effects)
	haloInk := cs.OnSurfaceVariant

	el.Children(func() {
		ui.Box(c).Size(target, target).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			drawRadio(p, r, radioPaint{
				sel:     clampF32(sel, 0, 1),
				halo:    clampF32(halo, 0, 1),
				haloInk: haloInk,
				ringOff: ringOff,
				ringOn:  ringOn,
				dot:     dotInk,
			})
		})
		if label != "" {
			ApplyTypeStyle(c, ui.Text(c, label).NoWrap().TextColor(ink), tokens.LabelLarge)
		}
	})
	return el
}

// radioPaint is one frame of a radio button's ring.
type radioPaint struct {
	sel     float32
	halo    float32
	haloInk ui.Color
	ringOff ui.Color
	ringOn  ui.Color
	dot     ui.Color
}

// drawRadio paints a radio button's state layer, ring and dot in its
// touch-target square r.
func drawRadio(p *ui.Painter, r ui.Rect, s radioPaint) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	paintHalo(p, cx, cy, radioHaloDiameter/2, s.haloInk, s.halo)

	ring := ui.Rect{X: cx - radioRingSize/2, Y: cy - radioRingSize/2, W: radioRingSize, H: radioRingSize}
	p.Stroke(ring, s.ringOff.Mix(s.ringOn, s.sel), radioRingSize/2, radioRingWidth)

	dot := radioDotSize * s.sel
	if dot > 0.4 {
		p.Fill(ui.Rect{X: cx - dot/2, Y: cy - dot/2, W: dot, H: dot}, s.dot, dot/2)
	}
}

// RadioGroupOptions configures RadioGroup. The zero value is a vertical group
// with an 8 DIP gap and no accessible name.
type RadioGroupOptions struct {
	// Label names the group for assistive technology.
	Label string
	// Horizontal lays the radios in a row with a 16 DIP gap instead of the
	// default column.
	Horizontal bool
	// Disabled disables the whole group and its radios.
	Disabled bool
}

// RadioGroup builds a Material Design 3 Expressive radio group: a
// ui.RadioGroup column (or row with Horizontal) of the radios fn builds, which
// is one Tab stop and takes the arrows, Home and End to move and choose among
// them. Assistive technology sees RoleRadioGroup named by Label:
//
//	md3e.RadioGroup(c, func() {
//		md3e.Radio(c, &app.size, "Small", "Small")
//		md3e.Radio(c, &app.size, "Large", "Large")
//	}, md3e.RadioGroupOptions{Label: "Size"})
func RadioGroup(c *ui.Context, fn func(), opts ...RadioGroupOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: RadioGroup takes at most one RadioGroupOptions")
	}
	if fn == nil {
		panic("md3e: RadioGroup requires a build function")
	}
	var o RadioGroupOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	el := ui.RadioGroup(c, fn)
	if o.Label != "" {
		el.Label(o.Label)
	}
	if o.Horizontal {
		el.Row().Gap(16)
	} else {
		el.Gap(8)
	}
	if o.Disabled {
		el.Disabled(true)
	}
	return el
}
