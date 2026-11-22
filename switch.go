package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// Switch metrics in DIP, as MD3E specifies the switch: a 52x32 track, a
// 16 DIP handle when off (24 with an icon), 24 DIP when on, 28 DIP pressed,
// and a 48 DIP row that gives the control its touch target.
const (
	switchTrackWidth   = 52
	switchTrackHeight  = 32
	switchRowHeight    = 48
	switchThumbOff     = 16
	switchThumbOn      = 24
	switchThumbPressed = 28
	switchThumbTravel  = 20
	switchHaloDiameter = 40
)

// SwitchOptions configures Switch. The zero value is an enabled switch with an
// mdicon.Check in its selected handle and no visible label.
type SwitchOptions struct {
	// Label is the visible text after the track, and names the switch for
	// assistive technology. An empty label draws the track alone.
	Label string
	// Disabled disables the switch: it ignores the pointer, the keyboard and
	// focus, and uses MD3E's disabled colors without ui's automatic dimming.
	Disabled bool
	// OffIcon draws in the off handle, beside the track's outline, as a close
	// mark. A nil OffIcon keeps the bare 16 DIP handle; an icon makes the off
	// handle 24 DIP, as MD3E has it.
	OffIcon IconDraw
	// OnIcon replaces the mdicon.Check of the selected handle.
	OnIcon IconDraw
}

// Switch builds a Material Design 3 Expressive switch over *on: a
// ui.SwitchBase row whose key is the caller's c.Key, carrying the 52x32 track
// and its handle, and optionally the visible label that toggles it. The handle
// moves with the Fast Spatial spring, grows from 16 to 24 DIP as the switch
// turns on (28 while pressed) and shows an mdicon.Check, while the track fades
// from SurfaceContainerHighest with an Outline border to Primary. MD3E state
// layers mark hover, focus and press.
//
// A click, or Space while the switch has the keyboard focus, turns *on over;
// assistive technology sees a switch named by Label:
//
//	md3e.Switch(c.Key("wifi"), &app.wifi, md3e.SwitchOptions{Label: "Wi-Fi"})
//
// The key lands on the stateful base itself, never on a wrapper row, so
// c.Key("wifi") keeps the switch's identity. Right to left, the track sits at
// the row's end and the handle mirrors with it.
func Switch(c *ui.Context, on *bool, opts ...SwitchOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Switch takes at most one SwitchOptions")
	}
	if on == nil {
		panic("md3e: Switch requires a non-nil *bool")
	}
	var o SwitchOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors

	el := ui.SwitchBase(c, on).
		DimDisabled(false).
		FocusRing(false).
		Height(switchRowHeight).
		Gap(12)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	ink := cs.OnSurface
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	el.TextColor(ink)
	if o.Label != "" {
		el.Label(o.Label)
	}

	onTrack := cs.Primary
	onThumb := cs.OnPrimary
	offTrack := cs.SurfaceContainerHighest
	offThumb := cs.Outline
	offBorder := cs.Outline
	// The handle's icon is the track color cut out of it, as MD3 draws it.
	onIconInk := onTrack
	offIconInk := offTrack
	if disabled {
		onTrack = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		offTrack = onTrack
		onThumb = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		offThumb = onThumb
		offBorder = onTrack
		onIconInk = cs.Surface
		offIconInk = cs.Surface
	}

	onNow := *on
	thReduceMotion := th.ReduceMotion
	effects := th.Animation(motion.FastEffects)
	spatial := th.Animation(motion.FastSpatial)

	onP := animateValue(el, animChan0, boolFloat(onNow), effects, thReduceMotion)
	pos := animateValue(el, animChan1, boolFloat(onNow), spatial, thReduceMotion)
	offSize := float32(switchThumbOff)
	if o.OffIcon != nil {
		offSize = switchThumbOn
	}
	thumbTarget := offSize
	if onNow {
		thumbTarget = switchThumbOn
	}
	pressed := !disabled && el.Pressed()
	if pressed {
		thumbTarget = switchThumbPressed
	}
	thumbSize := animateValue(el, animChan2, thumbTarget, spatial, thReduceMotion)

	haloSt := localHaloFade(el, animChan3)

	onIcon := o.OnIcon
	if onIcon == nil {
		onIcon = mdicon.Check.Draw
	}
	offIcon := o.OffIcon
	rtl := c.Direction() == ui.RightToLeft
	onPClamped := clampF32(onP, 0, 1)
	if !disabled && cs.OnPrimaryContainer.A > 0 {
		onIconInk = cs.OnPrimaryContainer
	}
	haloInk := cs.OnSurface.MixOklab(cs.Primary, onPClamped)
	offThumbHover := cs.OnSurfaceVariant
	if disabled || offThumbHover.A == 0 {
		offThumbHover = offThumb
	}
	paintSnap := switchPaint{
		onP:           onPClamped,
		pos:           clampF32(pos, -0.14, 1.14),
		thumb:         clampF32(thumbSize, 0, switchTrackHeight),
		haloInk:       haloInk,
		onTrack:       onTrack,
		offTrack:      offTrack,
		onThumb:       onThumb,
		offThumb:      offThumb,
		offThumbHover: offThumbHover,
		offBorder:     offBorder,
		onIcon:        onIcon,
		offIcon:       offIcon,
		onIconInk:     onIconInk,
		offIconInk:    offIconInk,
		rtl:           rtl,
	}

	el.Children(func() {
		ui.Box(c).Size(switchTrackWidth, switchTrackHeight).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			if !el.Valid() {
				// Exit ghost draw: paint frozen snapshot without reading live state or scheduling frames.
				drawSwitch(p, r, paintSnap)
				return
			}
			hovered := el.Hovered() && !disabled
			pressedDraw := !disabled && (pressed || el.Pressed())
			focusedDraw := !disabled && el.FocusVisible()
			targetHalo := resolveStateLayerTarget(disabled, false, pressedDraw, true, focusedDraw, hovered)
			if disabled {
				haloSt.reset()
			}
			halo, needFrame := haloSt.step(targetHalo, p.Now(), thReduceMotion || p.Preferences().ReduceMotion)
			if needFrame && rectIntersects(r, p.Visible()) {
				p.After(16 * time.Millisecond)
			}
			paintCfg := paintSnap
			paintCfg.halo = clampF32(halo, 0, 1)
			paintCfg.focused = focusedDraw
			drawSwitch(p, r, paintCfg)
		})
		if o.Label != "" {
			applyTypeStyleWithTheme(th, ui.Text(c, o.Label).NoWrap().TextColor(ink), tokens.LabelLarge)
		}
	})
	return el
}

// switchPaint is one frame of a switch's track.
type switchPaint struct {
	onP           float32
	pos           float32
	thumb         float32
	halo          float32
	haloInk       ui.Color
	onTrack       ui.Color
	offTrack      ui.Color
	onThumb       ui.Color
	offThumb      ui.Color
	offThumbHover ui.Color
	offBorder     ui.Color
	onIcon        IconDraw
	offIcon       IconDraw
	onIconInk     ui.Color
	offIconInk    ui.Color
	focused       bool
	rtl           bool
}

// drawSwitch paints a switch's track, state layer, handle and handle icon in
// the 52x32 box r.
func drawSwitch(p *ui.Painter, r ui.Rect, s switchPaint) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	radius := float32(switchTrackHeight) / 2
	cy := r.Y + r.H/2
	track := s.offTrack.MixOklab(s.onTrack, s.onP)
	p.Fill(r, track, radius)
	if borderA := 1 - s.onP; borderA > 0.01 && s.offBorder.A > 0 {
		p.Stroke(r, s.offBorder.Alpha(borderA), radius, 2)
	}
	if s.focused {
		p.FocusRing(r, [4]float32{radius, radius, radius, radius})
	}

	// Off, the 16 DIP handle sits 8 DIP in from the track's end, so its center
	// is 16 DIP from there; on, the 24 DIP handle travels 20 DIP to lie 4 DIP
	// in, its center 36 from the start. Allow subtle FastSpatial spring overshoot
	// while keeping the handle cleanly inside the 2 DIP track wall.
	ts := s.thumb
	cx := r.X + 16 + switchThumbTravel*s.pos
	if s.rtl {
		cx = r.X + r.W - 16 - switchThumbTravel*s.pos
	}
	minCX := r.X + ts*0.5 + 2
	maxCX := r.X + r.W - ts*0.5 - 2
	if maxCX >= minCX {
		cx = clampF32(cx, minCX, maxCX)
	}
	if s.halo > 0.001 && s.haloInk.A > 0 {
		h := float32(switchHaloDiameter) / 2
		p.Fill(ui.Rect{X: cx - h, Y: cy - h, W: 2 * h, H: 2 * h}, s.haloInk.Alpha(s.halo), h)
	}

	thumb := ui.Rect{X: cx - ts/2, Y: cy - ts/2, W: ts, H: ts}
	hoverMix := clampF32(s.halo/float32(tokens.StateHoverOpacity), 0, 1)
	offThumbCol := s.offThumb.MixOklab(s.offThumbHover, hoverMix)
	thumbInk := offThumbCol.MixOklab(s.onThumb, s.onP)
	p.Fill(thumb, thumbInk, ts/2)

	iconSide := float32(16)
	iconRect := ui.Rect{X: cx - iconSide/2, Y: cy - iconSide/2, W: iconSide, H: iconSide}
	if s.onIcon != nil && s.onP > 0.02 {
		sc := 0.55 + 0.45*s.onP
		p.Transform(ui.Scaling(sc, sc), cx, cy, func() {
			s.onIcon(p, iconRect, s.onIconInk.Alpha(clampF32(s.onP, 0, 1)))
		})
	}
	if s.offIcon != nil && s.onP < 0.98 {
		offP := clampF32(1-s.onP, 0, 1)
		sc := 0.55 + 0.45*offP
		p.Transform(ui.Scaling(sc, sc), cx, cy, func() {
			s.offIcon(p, iconRect, s.offIconInk.Alpha(offP))
		})
	}
}

// paintHalo paints an MD3E state layer disc of radius around (cx, cy).
func paintHalo(p *ui.Painter, cx, cy, radius float32, ink ui.Color, alpha float32) {
	if alpha <= 0.001 || ink.A == 0 || radius <= 0 {
		return
	}
	p.Fill(ui.Rect{X: cx - radius, Y: cy - radius, W: 2 * radius, H: 2 * radius}, ink.Alpha(alpha), radius)
}

// boolFloat returns 1 for true and 0 for false, the target of an AnimateTo
// channel driven by a flag.
func boolFloat(b bool) float32 {
	if b {
		return 1
	}
	return 0
}
