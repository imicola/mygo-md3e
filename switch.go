package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/motion"
	"mygo-md3e/tokens"
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

// AnimateTo channels of a switch.
type (
	switchOnKey    struct{}
	switchPosKey   struct{}
	switchThumbKey struct{}
	switchHaloKey  struct{}
)

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
	effects := th.Animation(motion.FastEffects)
	spatial := th.Animation(motion.FastSpatial)
	if th.ReduceMotion {
		effects, spatial = motion.Snap, motion.Snap
	}

	onP := el.AnimateTo(switchOnKey{}, boolFloat(onNow), effects)
	pos := el.AnimateTo(switchPosKey{}, boolFloat(onNow), spatial)
	offSize := float32(switchThumbOff)
	if o.OffIcon != nil {
		offSize = switchThumbOn
	}
	thumbTarget := offSize
	if onNow {
		thumbTarget = switchThumbOn
	}
	pressed := el.Pressed()
	if pressed {
		thumbTarget = switchThumbPressed
	}
	thumbSize := el.AnimateTo(switchThumbKey{}, thumbTarget, spatial)

	haloTarget := float32(0)
	switch {
	case disabled:
	case pressed:
		haloTarget = float32(tokens.StatePressedOpacity)
	case el.FocusVisible():
		haloTarget = float32(tokens.StateFocusOpacity)
	case el.Hovered():
		haloTarget = float32(tokens.StateHoverOpacity)
	}
	halo := el.AnimateTo(switchHaloKey{}, haloTarget, effects)

	onIcon := o.OnIcon
	if onIcon == nil {
		onIcon = mdicon.Check.Draw
	}
	offIcon := o.OffIcon
	rtl := c.Direction() == ui.RightToLeft
	haloInk := cs.OnSurfaceVariant

	el.Children(func() {
		ui.Box(c).Size(switchTrackWidth, switchTrackHeight).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			drawSwitch(p, r, switchPaint{
				onP:        clampF32(onP, 0, 1),
				pos:        clampF32(pos, 0, 1),
				thumb:      clampF32(thumbSize, 0, switchTrackHeight),
				halo:       clampF32(halo, 0, 1),
				haloInk:    haloInk,
				onTrack:    onTrack,
				offTrack:   offTrack,
				onThumb:    onThumb,
				offThumb:   offThumb,
				offBorder:  offBorder,
				onIcon:     onIcon,
				offIcon:    offIcon,
				onIconInk:  onIconInk,
				offIconInk: offIconInk,
				rtl:        rtl,
			})
		})
		if o.Label != "" {
			ApplyTypeStyle(c, ui.Text(c, o.Label).NoWrap().TextColor(ink), tokens.LabelLarge)
		}
	})
	return el
}

// switchPaint is one frame of a switch's track.
type switchPaint struct {
	onP        float32
	pos        float32
	thumb      float32
	halo       float32
	haloInk    ui.Color
	onTrack    ui.Color
	offTrack   ui.Color
	onThumb    ui.Color
	offThumb   ui.Color
	offBorder  ui.Color
	onIcon     IconDraw
	offIcon    IconDraw
	onIconInk  ui.Color
	offIconInk ui.Color
	rtl        bool
}

// drawSwitch paints a switch's track, state layer, handle and handle icon in
// the 52x32 box r.
func drawSwitch(p *ui.Painter, r ui.Rect, s switchPaint) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	radius := float32(switchTrackHeight) / 2
	cy := r.Y + r.H/2
	track := s.offTrack.Mix(s.onTrack, s.onP)
	p.Fill(r, track, radius)
	if borderA := 1 - s.onP; borderA > 0.01 && s.offBorder.A > 0 {
		p.Stroke(r, s.offBorder, radius, 2)
	}

	// Off, the 16 DIP handle sits 8 DIP in from the track's end, so its center
	// is 16 DIP from there; on, the 24 DIP handle travels 20 DIP to lie 4 DIP
	// in, its center 36 from the start.
	cx := r.X + 16 + switchThumbTravel*s.pos
	if s.rtl {
		cx = r.X + r.W - 16 - switchThumbTravel*s.pos
	}
	if s.halo > 0.001 && s.haloInk.A > 0 {
		h := float32(switchHaloDiameter) / 2
		p.Fill(ui.Rect{X: cx - h, Y: cy - h, W: 2 * h, H: 2 * h}, s.haloInk.Alpha(s.halo), h)
	}

	ts := s.thumb
	thumb := ui.Rect{X: cx - ts/2, Y: cy - ts/2, W: ts, H: ts}
	thumbInk := s.offThumb.Mix(s.onThumb, s.onP)
	p.Fill(thumb, thumbInk, ts/2)

	iconSide := float32(16)
	iconRect := ui.Rect{X: cx - iconSide/2, Y: cy - iconSide/2, W: iconSide, H: iconSide}
	if s.onIcon != nil && s.onP > 0.02 {
		s.onIcon(p, iconRect, s.onIconInk.Alpha(clampF32(s.onP, 0, 1)))
	}
	if s.offIcon != nil && s.onP < 0.98 {
		s.offIcon(p, iconRect, s.offIconInk.Alpha(clampF32(1-s.onP, 0, 1)))
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
