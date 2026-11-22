package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// Radio metrics in DIP: a 20 DIP ring with a 10 DIP inner dot, inside a 40 DIP
// touch target whose circular state layer spans the target.
const (
	radioRingSize     = 20
	radioRingWidth    = 2
	radioDotSize      = 10
	radioTouchTarget  = 40
	radioHaloDiameter = 40
	radioRippleTime   = 320 * time.Millisecond
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

// radioRippleState is the press wave of a radio's heat area: where it began
// in the touch target's DIPs, when, whether it still runs, and the row's
// size as the last build laid it out, which tells where the target sits.
type radioRippleState struct {
	active bool
	ox, oy float32
	start  time.Time
	rowW   float32
	rowH   float32
}

func newRadioRippleState() *radioRippleState { return &radioRippleState{} }

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
	rtl := c.Direction() == ui.RightToLeft

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
	ringHover := cs.OnSurface
	ringOn := cs.Primary
	dotInk := cs.Primary
	if disabled {
		ringOff = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		ringHover = ringOff
		ringOn = ringOff
		dotInk = ringOff
	}

	thReduceMotion := th.ReduceMotion || c.Preferences().ReduceMotion
	spatial := th.Animation(motion.FastSpatial)
	sel := animateValue(el, animChan0, boolFloat(on), spatial, thReduceMotion)
	selColor := clampF32(sel, 0, 1)
	dotScale := clampF32(sel, 0, 1.25)

	haloSt := localHaloFade(el, animChan1)
	haloInk := cs.OnSurface.MixOklab(cs.Primary, selColor)

	rp := localPtr(el, animChan2, newRadioRippleState)
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

	paintSnap := radioPaint{
		sel:          selColor,
		dotScale:     dotScale,
		haloInk:      haloInk,
		ringOff:      ringOff,
		ringHover:    ringHover,
		ringOn:       ringOn,
		dot:          dotInk,
		ripple:       rp,
		reduceMotion: thReduceMotion,
	}

	el.Children(func() {
		ui.Box(c).Size(target, target).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			if !el.Valid() {
				ghost := paintSnap
				ghost.ripple = nil
				drawRadio(p, r, ghost)
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
			drawRadio(p, r, paintCfg)
		})
		if label != "" {
			applyTypeStyleWithTheme(th, ui.Text(c, label).NoWrap().TextColor(ink), tokens.LabelLarge)
		}
	})
	return el
}

// radioPaint is one frame of a radio button's ring.
type radioPaint struct {
	sel          float32
	dotScale     float32
	halo         float32
	haloInk      ui.Color
	ringOff      ui.Color
	ringHover    ui.Color
	ringOn       ui.Color
	dot          ui.Color
	ripple       *radioRippleState
	focused      bool
	reduceMotion bool
}

// drawRadio paints a radio button's state layer, press wave, ring and dot in
// its touch-target square r.
func drawRadio(p *ui.Painter, r ui.Rect, s radioPaint) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	radius := min(float32(radioHaloDiameter)/2, min(r.W, r.H)/2)
	paintHalo(p, cx, cy, radius, s.haloInk, s.halo)

	if rp := s.ripple; rp != nil && rp.active && !s.reduceMotion {
		elapsed := p.Now().Sub(rp.start)
		if elapsed < 0 {
			elapsed = 0
		}
		if elapsed >= radioRippleTime {
			rp.active = false
		} else {
			t := float32(elapsed) / float32(radioRippleTime)
			inv := 1 - t
			grow := 1 - inv*inv*inv
			waveR := 6 + (radius-6)*grow
			alpha := float32(tokens.StatePressedOpacity) * 1.4 * inv
			ox, oy := r.X+rp.ox, r.Y+rp.oy
			clip := ui.Rect{X: cx - radius, Y: cy - radius, W: 2 * radius, H: 2 * radius}
			p.Clip(clip, radius, func() {
				p.Fill(ui.Rect{X: ox - waveR, Y: oy - waveR, W: 2 * waveR, H: 2 * waveR},
					s.haloInk.Alpha(alpha), waveR)
			})
			if rectIntersects(r, p.Visible()) {
				p.After(16 * time.Millisecond)
			}
		}
	}

	ring := ui.Rect{X: cx - radioRingSize/2, Y: cy - radioRingSize/2, W: radioRingSize, H: radioRingSize}
	if s.focused {
		p.FocusRing(ring, [4]float32{radioRingSize / 2, radioRingSize / 2, radioRingSize / 2, radioRingSize / 2})
	}
	hoverMix := clampF32(s.halo/float32(tokens.StateHoverOpacity), 0, 1)
	offCol := s.ringOff.MixOklab(s.ringHover, hoverMix)
	p.Stroke(ring, offCol.MixOklab(s.ringOn, s.sel), radioRingSize/2, radioRingWidth)

	dot := radioDotSize * s.dotScale
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

	el := ui.RadioGroup(c, func() {}).DimDisabled(false)
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
	if fn != nil {
		el.Children(fn)
	}
	return el
}
