package md3e

import (
	"math"
	"sync"
	"time"
	"weak"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// FocusRingStyle selects how AttachStateLayer and AttachRipple draw the
// keyboard focus outline when el.FocusVisible() is true.
type FocusRingStyle int

const (
	// FocusRingAuto draws a shape-conforming MD3E outer keyboard focus ring
	// matching the element's live corner radii or custom polygon contour.
	FocusRingAuto FocusRingStyle = iota
	// FocusRingInset draws the keyboard focus ring inside the element's clip
	// bounds (used by compact edge-aligned chrome such as WindowTitleBar).
	FocusRingInset
	// FocusRingNone suppresses the outline ring, retaining only the internal
	// Focus StateLayer tint.
	FocusRingNone
)

// StateLayerOptions configures MD3E StateLayer and ExpressiveRipple rendering.
type StateLayerOptions struct {
	// Color is the overlay tint color (typically the component's ContentColor or
	// OnSurface). Defaults to CurrentTheme(c).Colors.OnSurface when zero.
	Color ui.Color
	// Radii sets independent [TL, TR, BR, BL] corner radii in DIPs for clipping.
	Radii    [4]float32
	HasRadii bool
	// Radius sets a uniform corner radius when Radii is unset.
	Radius float32
	// Clip confines the state layer and ripple to a box and corner radii it
	// computes from the element's own painted window-space rectangle r, returning
	// (box, radii) in the same window coordinates (e.g. a navigation destination's
	// indicator pill inside its full-cell button). Nil paints over the whole box.
	Clip func(r ui.Rect) (box ui.Rect, radii [4]float32)
	// Shape optionally clips the state layer and ripple to a custom polygon.
	Shape *shape.RoundedPolygon
	// ClipPath optionally returns a dynamic *ui.Path (such as a live shape.Morph
	// outline) in window coordinates for clipping the state layer and ripple.
	// When set, ClipPath takes precedence over Shape and Radii.
	ClipPath func(r ui.Rect) *ui.Path
	// BorderInset shrinks the clip box inward by BorderInset DIPs (e.g., stroke width).
	BorderInset float32
	// FocusRing selects how the keyboard focus outline is rendered when
	// el.FocusVisible() is true (defaults to FocusRingAuto).
	FocusRing FocusRingStyle
	// FocusHaloColor optionally paints a backing contrast halo beneath the outer
	// focus ring (for example cs.Surface around photo-backed interactive surfaces).
	FocusHaloColor ui.Color
	// Dragged activates the MD3E Dragged state layer opacity (0.16).
	Dragged bool
	// DisableRipple disables expanding circular ripple waves while retaining StateLayer.
	DisableRipple bool
	// Disabled suppresses hover, focus, and ripple feedback.
	Disabled bool
}

type rippleWave struct {
	ox, oy     float32
	centered   bool
	fromKey    bool
	start      time.Time
	releasedAt time.Time
	released   bool
}

type evaluatedWave struct {
	ox, oy    float32
	curR      float32
	waveAlpha float32
}

type rippleLocalKey struct{}

// animChanKey provides shared per-element animation/local slot keys so
// components avoid emitting unique empty-struct type descriptors per channel
// (keeping Zero-Bloat binary size tight).
type animChanKey uint8

const (
	animChan0 animChanKey = iota
	animChan1
	animChan2
	animChan3
	animChan4
	animChan5
	animChan6
	animChan7 // reserved for attachHoverElevation
)

// localPtr stores a heap-allocated state pointer in ui.Local using a single
// pointer gcshape (*uint8) across all component state types, preventing
// per-type ui.Local/coreLocal/keepLocal generic code bloat while keeping
// steady-state lookups allocation-free.
func localPtr[P any](el ui.Element, key any, init func() P) P {
	p := ui.Local(el, key, init)
	if p == nil {
		var zero P
		return zero
	}
	return *p
}

func newHaloFadeState() *haloFadeState { return &haloFadeState{} }
func newRippleState() *rippleState     { return &rippleState{} }
func newFalse() *bool                  { return new(bool) }
func newZeroInt() *int                 { return new(int) }
func newNegOneInt() *int               { v := -1; return &v }
func newZeroFloat32() *float32         { return new(float32) }
func newScrollState() *ui.ScrollState  { return &ui.ScrollState{} }
func newUIHandle() *ui.Handle          { return &ui.Handle{} }

func localHaloFade(el ui.Element, key any) *haloFadeState {
	return localPtr(el, key, newHaloFadeState)
}

func localRippleState(el ui.Element) *rippleState {
	return localPtr(el, rippleLocalKey{}, newRippleState)
}

func localBool(el ui.Element, key any) *bool {
	return localPtr(el, key, newFalse)
}

func localInt(el ui.Element, key any) *int {
	return localPtr(el, key, newZeroInt)
}

func localNegInt(el ui.Element, key any) *int {
	return localPtr(el, key, newNegOneInt)
}

func localFloat32(el ui.Element, key any) *float32 {
	return localPtr(el, key, newZeroFloat32)
}

func localScrollState(el ui.Element, key any) *ui.ScrollState {
	return localPtr(el, key, newScrollState)
}

func localHandle(el ui.Element, key any) *ui.Handle {
	return localPtr(el, key, newUIHandle)
}

// haloFadeState tracks exponential StateLayer / Halo alpha transitions inside Draw
// so hover changes never trigger view rebuilds.
type haloFadeState struct {
	alpha     float32
	target    float32
	animating bool
	prevNow   time.Time
}

func (h *haloFadeState) reset() {
	h.alpha = 0
	h.target = 0
	h.animating = false
}

func (h *haloFadeState) step(target float32, now time.Time, reduceMotion bool) (alpha float32, needFrame bool) {
	if reduceMotion {
		h.alpha = target
		h.target = target
		h.animating = false
		return target, false
	}
	if h.alpha != target {
		if !h.animating || h.target != target {
			h.target = target
			h.animating = true
			h.prevNow = now
		} else {
			dt := float32(now.Sub(h.prevNow).Seconds())
			h.prevNow = now
			if dt > 0 {
				factor := 1 - float32(math.Exp(-28.0*float64(dt)))
				h.alpha += (target - h.alpha) * factor
				if absF32(target-h.alpha) < 0.0015 {
					h.alpha = target
				}
			}
		}
		if h.alpha != target {
			return h.alpha, true
		}
		h.animating = false
	} else {
		h.animating = false
	}
	return h.alpha, false
}

func resolveStateLayerTarget(disabled, dragged, pressed, withWavePressed, focused, hovered bool) float32 {
	if disabled {
		return 0
	}
	switch {
	case dragged:
		return float32(tokens.StateDraggedOpacity)
	case pressed && withWavePressed:
		return float32(tokens.StatePressedOpacity)
	case focused:
		return float32(tokens.StateFocusOpacity)
	case hovered:
		return float32(tokens.StateHoverOpacity)
	default:
		return 0
	}
}

func animateValue(el ui.Element, key any, target float32, anim ui.Animation, reduceMotion bool) float32 {
	if reduceMotion {
		return el.Jump(key, target)
	}
	return el.AnimateTo(key, target, anim)
}

func animateWithTheme(el ui.Element, th *Theme, key any, target float32, token motion.Token) float32 {
	if th == nil {
		th = DefaultTheme(nil)
	}
	anim := th.Animation(token)
	if th.ReduceMotion && !anim.NonMotion {
		return el.Jump(key, target)
	}
	return el.AnimateTo(key, target, anim)
}

// animateEffect drives an effects-channel transition (color/opacity/scrim) that
// snaps immediately with zero pending frames when ReduceMotion is active on
// either the Theme or desktop Preferences.
func animateEffect(c *ui.Context, el ui.Element, th *Theme, key any, target float32, token motion.Token) float32 {
	if th == nil {
		th = DefaultTheme(nil)
	}
	reduce := th.ReduceMotion || (c != nil && c.Preferences().ReduceMotion)
	if reduce {
		return el.Jump(key, target)
	}
	return el.AnimateTo(key, target, th.Animation(token))
}

type rippleState struct {
	waves          []rippleWave
	pointerHeld    bool
	sawPressedTrue bool
	allowWaves     bool
	hasDrawOver    bool
	hoverHook      func(p *ui.Painter, now time.Time, hovered bool)

	focusHandle *ui.Handle
	srv         ui.Services

	halo     haloFadeState
	clipPath ui.Path
	stops    [3]ui.GradientStop
}

type windowFocusSlot struct {
	srv    ui.Services
	handle weak.Pointer[ui.Handle]
}

var (
	focusTriggerMu    sync.Mutex
	focusTriggerSlots [16]windowFocusSlot
	focusTriggerNext  int
)

func recordFocusTrigger(srv ui.Services, h *ui.Handle) {
	if srv == (ui.Services{}) || h == nil {
		return
	}
	wp := weak.Make(h)
	focusTriggerMu.Lock()
	defer focusTriggerMu.Unlock()
	for i := range focusTriggerSlots {
		if focusTriggerSlots[i].srv == srv {
			focusTriggerSlots[i].handle = wp
			return
		}
	}
	for i := range focusTriggerSlots {
		if focusTriggerSlots[i].srv == (ui.Services{}) || focusTriggerSlots[i].handle.Value() == nil {
			focusTriggerSlots[i] = windowFocusSlot{srv: srv, handle: wp}
			return
		}
	}
	focusTriggerSlots[focusTriggerNext] = windowFocusSlot{srv: srv, handle: wp}
	focusTriggerNext = (focusTriggerNext + 1) % len(focusTriggerSlots)
}

func focusedTriggerHandle(c *ui.Context) *ui.Handle {
	if c == nil {
		return nil
	}
	srv := c.Services()
	if srv == (ui.Services{}) {
		return nil
	}
	focusTriggerMu.Lock()
	var wp weak.Pointer[ui.Handle]
	for i := range focusTriggerSlots {
		if focusTriggerSlots[i].srv == srv {
			wp = focusTriggerSlots[i].handle
			break
		}
	}
	focusTriggerMu.Unlock()
	h := wp.Value()
	if h == nil || !h.Focused(c) {
		return nil
	}
	return h
}

const (
	rippleExpandDuration  = 320 * time.Millisecond
	rippleFadeInDuration  = 70 * time.Millisecond
	rippleFadeOutDuration = 220 * time.Millisecond
	rippleMinHold         = 80 * time.Millisecond
	maxActiveRipples      = 4
)

func (s *rippleState) startWave(ox, oy float32, fromKey bool, now time.Time) {
	if now.IsZero() {
		now = time.Unix(0, 1)
	}
	w := rippleWave{
		ox:       ox,
		oy:       oy,
		centered: fromKey,
		fromKey:  fromKey,
		start:    now,
	}
	if !fromKey {
		s.pointerHeld = true
		s.sawPressedTrue = false
	}
	if len(s.waves) >= maxActiveRipples {
		copy(s.waves, s.waves[1:])
		s.waves[len(s.waves)-1] = w
		return
	}
	s.waves = append(s.waves, w)
}

func (s *rippleState) releaseWaves(fromKey bool, now time.Time) {
	if now.IsZero() {
		now = time.Unix(0, 1)
	}
	if !fromKey {
		s.pointerHeld = false
		s.sawPressedTrue = false
	}
	for i := range s.waves {
		if s.waves[i].fromKey == fromKey && !s.waves[i].released {
			s.waves[i].released = true
			s.waves[i].releasedAt = now
		}
	}
}

func (s *rippleState) clearTransient() {
	s.waves = s.waves[:0]
	s.pointerHeld = false
	s.sawPressedTrue = false
}

// AttachStateLayer attaches an MD3E StateLayer overlay (Hover 8%, Focus 10%,
// Pressed 10%, Dragged 16%) onto el without triggering view rebuilds on hover.
func AttachStateLayer(c *ui.Context, el ui.Element, opts ...StateLayerOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: AttachStateLayer takes at most one StateLayerOptions")
	}
	var opt StateLayerOptions
	if len(opts) == 1 {
		opt = opts[0]
	}
	return attachStateLayerWithTheme(CurrentTheme(c), el, opt)
}

func attachStateLayerWithTheme(th *Theme, el ui.Element, opt StateLayerOptions) ui.Element {
	return attachOverlayFeedbackWithTheme(nil, th, el, false, opt)
}

// AttachRipple attaches both MD3E StateLayer and ExpressiveRipple wave feedback
// onto el. All hover/press visual state reads happen strictly inside DrawOver to
// avoid rebuilding the UI tree on pointer hover, and animations respect
// c.Preferences().ReduceMotion and stop cleanly once settled.
func AttachRipple(c *ui.Context, el ui.Element, opts ...StateLayerOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: AttachRipple takes at most one StateLayerOptions")
	}
	var opt StateLayerOptions
	if len(opts) == 1 {
		opt = opts[0]
	}
	return attachRippleWithContext(c, CurrentTheme(c), el, opt)
}

func attachRippleWithTheme(th *Theme, el ui.Element, opt StateLayerOptions) ui.Element {
	return attachOverlayFeedbackWithTheme(nil, th, el, !opt.DisableRipple, opt)
}

func attachRippleWithContext(c *ui.Context, th *Theme, el ui.Element, opt StateLayerOptions) ui.Element {
	return attachOverlayFeedbackWithTheme(c, th, el, !opt.DisableRipple, opt)
}

func attachOverlayFeedbackWithTheme(c *ui.Context, th *Theme, el ui.Element, withRipple bool, opt StateLayerOptions) ui.Element {
	st := localRippleState(el)
	if st == nil {
		return el
	}
	if th == nil {
		th = DefaultTheme(nil)
	}
	thReduceMotion := th.ReduceMotion
	st.allowWaves = withRipple && !thReduceMotion
	st.hasDrawOver = true
	st.hoverHook = nil
	if c != nil {
		st.srv = c.Services()
	}

	ink := opt.Color
	if ink.A == 0 {
		ink = th.Colors.OnSurface
	}

	radii := opt.Radii
	if !opt.HasRadii && radii == ([4]float32{}) && opt.Radius > 0 {
		radii = [4]float32{opt.Radius, opt.Radius, opt.Radius, opt.Radius}
	}

	disabled, dragged := opt.Disabled, opt.Dragged
	clip, customPath, poly, inset := opt.Clip, opt.ClipPath, opt.Shape, opt.BorderInset
	focusStyle, focusHalo := opt.FocusRing, opt.FocusHaloColor
	focusRingCol := th.Colors.Secondary
	if focusRingCol.A == 0 {
		focusRingCol = th.Colors.Primary
	}
	focusRingW := float32(2)
	if th.Contrast >= tokens.ContrastHigh {
		focusRingW = 3
	}
	const focusRingOff = float32(2)
	if withRipple {
		if c != nil && !disabled {
			if st.focusHandle == nil {
				st.focusHandle = new(ui.Handle)
			}
			el.Bind(st.focusHandle)
		}
		el.ObserveInput(func(ev ui.InputEvent) {
			if disabled {
				return
			}
			switch ev.Kind {
			case ui.InputPointerDown:
				if ev.Button == 0 && st.srv != (ui.Services{}) && st.focusHandle != nil {
					recordFocusTrigger(st.srv, st.focusHandle)
				}
			case ui.InputKeyDown:
				if (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) && !ev.Repeat && st.srv != (ui.Services{}) && st.focusHandle != nil {
					recordFocusTrigger(st.srv, st.focusHandle)
				}
			}
			if !st.allowWaves {
				return
			}
			switch ev.Kind {
			case ui.InputPointerDown:
				if ev.Button == 0 {
					st.startWave(ev.X, ev.Y, false, ev.Time)
				}
			case ui.InputPointerUp:
				if ev.Button == 0 {
					st.releaseWaves(false, ev.Time)
				}
			case ui.InputKeyDown:
				if (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) && !ev.Repeat {
					st.startWave(0, 0, true, ev.Time)
				}
			case ui.InputKeyUp:
				if ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace {
					st.releaseWaves(true, ev.Time)
				}
			}
		})
	} else if !opt.Disabled {
		// StateLayer-only surfaces still need pointer-press capture without allocating
		// an ObserveInput closure or setting hoverBuilt.
		_, _ = el.PressedButton()
	}

	el.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !el.Valid() || r.W <= 0 || r.H <= 0 {
			return
		}
		if disabled || el.IsDisabled() {
			st.clearTransient()
			st.halo.reset()
			if st.hoverHook != nil {
				st.hoverHook(p, p.Now(), false)
			}
			return
		}

		reduceMotion := thReduceMotion || p.Preferences().ReduceMotion
		now := p.Now()
		hovered := el.Hovered()
		visible := rectIntersects(r, p.Visible())
		if st.hoverHook != nil {
			st.hoverHook(p, now, visible && hovered)
		}
		pressed := el.Pressed()
		focused := el.FocusVisible()

		if withRipple {
			if pressed {
				st.sawPressedTrue = true
			} else if st.pointerHeld && (st.sawPressedTrue || !hovered) {
				st.releaseWaves(false, now)
			}
			if reduceMotion {
				// Skip circular wave animation under ReduceMotion, retaining immediate static layer feedback.
				st.clearTransient()
			}
		}

		// The ripple itself is the pressed feedback; adding the 10% pressed layer on
		// top would double it. Keep the pressed layer for keyboard, StateLayer-only,
		// and ReduceMotion feedback where no wave spreads.
		targetAlpha := resolveStateLayerTarget(false, dragged, pressed || st.pointerHeld, !withRipple || reduceMotion, focused, hovered)
		layerAlpha, needFrame := st.halo.step(targetAlpha, now, reduceMotion)

		box, boxRadii := r, radii
		if clip != nil {
			box, boxRadii = clip(r)
		}
		clipRect := insetRect(box, inset)
		clipRadii := insetRadii(boxRadii, inset)
		if clipRect.W <= 0 || clipRect.H <= 0 || !rectIntersects(clipRect, p.Visible()) {
			return
		}

		var stackWaves [maxActiveRipples]evaluatedWave
		wavesToDraw := stackWaves[:0]
		if withRipple && !reduceMotion && len(st.waves) > 0 {
			kept := st.waves[:0]
			for i := range st.waves {
				w := st.waves[i]
				ox, oy := rippleOrigin(&w, r, clipRect)
				curR, enterAlpha, fadeAlpha, animating, alive := evalRippleWave(&w, ox, oy, clipRect, now)
				if alive && fadeAlpha > 0.001 {
					kept = append(kept, w)
					if animating {
						needFrame = true
					}
					if waveAlpha := float32(tokens.StatePressedOpacity) * enterAlpha * fadeAlpha; waveAlpha > 0.001 && curR > 0 {
						wavesToDraw = append(wavesToDraw, evaluatedWave{
							ox:        ox,
							oy:        oy,
							curR:      curR,
							waveAlpha: waveAlpha,
						})
					}
				}
			}
			st.waves = kept
		}

		if needFrame {
			p.After(16 * time.Millisecond)
		}
		drawFocusRing := focused && focusStyle != FocusRingNone && focusRingCol.A > 0
		if layerAlpha <= 0.001 && len(wavesToDraw) == 0 && !drawFocusRing {
			return
		}

		if layerAlpha > 0.001 || len(wavesToDraw) > 0 {
			paintBody := func(polyPath *ui.Path) {
				if layerAlpha > 0.001 {
					layerInk := ink.Alpha(layerAlpha)
					if polyPath != nil {
						p.FillPath(polyPath, layerInk)
					} else {
						p.FillRadii(clipRect, clipRadii, layerInk)
					}
				}

				st.paintWaves(p, clipRect, ink, wavesToDraw)
			}

			if customPath != nil {
				if path := customPath(clipRect); path != nil {
					p.ClipPath(path, func() {
						paintBody(path)
					})
				}
			} else if poly != nil {
				path := poly.WriteTo(&st.clipPath, clipRect)
				p.ClipPath(path, func() {
					paintBody(path)
				})
			} else {
				paintInClip(p, &st.clipPath, clipRect, clipRadii, func() {
					paintBody(nil)
				})
			}
		}

		if drawFocusRing && box.W > 0 && box.H > 0 {
			paintShapeFocusRing(p, &st.clipPath, box, boxRadii, customPath, poly, focusStyle, focusRingCol, focusHalo, focusRingOff, focusRingW)
		}
	})

	return el
}

func rippleOrigin(w *rippleWave, r, inner ui.Rect) (float32, float32) {
	if w.centered {
		return inner.W * 0.5, inner.H * 0.5
	}
	// w.ox is in r's coordinates while inner is already in window coordinates
	// (including any BorderInset from insetRect), so inner.X/Y already account
	// for both element origin and inset.
	ox := clampF32(r.X+w.ox-inner.X, 0, inner.W)
	oy := clampF32(r.Y+w.oy-inner.Y, 0, inner.H)
	return ox, oy
}

func evalRippleWave(w *rippleWave, ox, oy float32, box ui.Rect, now time.Time) (curR, enterAlpha, fadeAlpha float32, animating, alive bool) {
	elapsed := max(now.Sub(w.start), 0)
	expandT := clampF32(float32(elapsed)/float32(rippleExpandDuration), 0, 1)
	inv := 1 - expandT
	easedR := 1 - inv*inv*inv

	maxR := farthestCornerDist(ox, oy, box.W, box.H) + 2
	startR := max(min(10, maxR*0.5), maxR*0.20)
	curR = startR + (maxR-startR)*easedR

	enterT := clampF32(float32(elapsed)/float32(rippleFadeInDuration), 0, 1)
	enterAlpha = 0.45 + 0.55*enterT

	fadeAlpha = 1
	alive = true
	animating = expandT < 1 || enterT < 1

	if w.released {
		fadeStart := w.releasedAt
		if minAt := w.start.Add(rippleMinHold); minAt.After(fadeStart) {
			fadeStart = minAt
		}
		fadeElapsed := now.Sub(fadeStart)
		if fadeElapsed > 0 {
			fadeT := clampF32(float32(fadeElapsed)/float32(rippleFadeOutDuration), 0, 1)
			fadeAlpha = 1 - fadeT
			if fadeT >= 1 {
				alive = false
				fadeAlpha = 0
			}
		}
		animating = alive
	}

	return curR, enterAlpha, fadeAlpha, animating, alive
}

func farthestCornerDist(ox, oy, w, h float32) float32 {
	dx := max(ox, w-ox)
	dy := max(oy, h-oy)
	return float32(math.Hypot(float64(dx), float64(dy)))
}

func (st *rippleState) paintWaves(p *ui.Painter, box ui.Rect, ink ui.Color, waves []evaluatedWave) {
	for _, wave := range waves {
		waveInk := ink.Alpha(wave.waveAlpha)
		cx, cy := wave.ox/box.W, wave.oy/box.H
		if cx == 0 && cy == 0 {
			// The zero Point means the center of the box to RadialGradient.
			cx = 1e-4
		}
		st.stops = [3]ui.GradientStop{
			{Color: waveInk, At: 0},
			{Color: waveInk, At: 0.80},
			{Color: ui.Transparent, At: 1},
		}
		p.FillRadialGradient(box, ui.RadialGradient{
			Center: ui.Point{X: cx, Y: cy},
			Radius: wave.curR,
			Stops:  st.stops[:],
		}, 0)
	}
}

func paintInClip(p *ui.Painter, scratch *ui.Path, box ui.Rect, radii [4]float32, fn func()) {
	if equalRadii(radii) {
		p.Clip(box, radii[0], fn)
		return
	}
	if path := writeRoundedRectPath(scratch, box, radii); path != nil {
		p.ClipPath(path, fn)
	}
}

func outsetRect(r ui.Rect, outset float32) ui.Rect {
	if outset <= 0 {
		return r
	}
	return ui.Rect{
		X: r.X - outset,
		Y: r.Y - outset,
		W: r.W + 2*outset,
		H: r.H + 2*outset,
	}
}

func outsetRadii(radii [4]float32, outset float32) [4]float32 {
	if outset <= 0 {
		return radii
	}
	return [4]float32{
		radii[0] + outset,
		radii[1] + outset,
		radii[2] + outset,
		radii[3] + outset,
	}
}

func paintShapeFocusRing(
	p *ui.Painter,
	scratch *ui.Path,
	box ui.Rect,
	radii [4]float32,
	customPath func(ui.Rect) *ui.Path,
	poly *shape.RoundedPolygon,
	style FocusRingStyle,
	ringCol, haloCol ui.Color,
	off, width float32,
) {
	if style == FocusRingInset {
		inBox := insetRect(box, width*0.5+0.5)
		inRad := insetRadii(radii, width*0.5+0.5)
		if inBox.W <= 0 || inBox.H <= 0 {
			return
		}
		if customPath != nil {
			if path := customPath(inBox); path != nil {
				p.StrokePath(path, width, ringCol)
			}
			return
		}
		if poly != nil {
			p.StrokePath(poly.WriteTo(scratch, inBox), width, ringCol)
			return
		}
		if equalRadii(inRad) {
			p.Stroke(inBox, ringCol, inRad[0], width)
			return
		}
		if path := writeRoundedRectPath(scratch, inBox, inRad); path != nil {
			p.StrokePath(path, width, ringCol)
		}
		return
	}

	if customPath != nil {
		if haloCol.A > 0 {
			if hPath := customPath(outsetRect(box, off*0.5+width*0.5)); hPath != nil {
				p.StrokePath(hPath, width+2, haloCol)
			}
		}
		if path := customPath(outsetRect(box, off+width*0.5)); path != nil {
			p.StrokePath(path, width, ringCol)
		}
		return
	}
	if poly != nil {
		if haloCol.A > 0 {
			p.StrokePath(poly.WriteTo(scratch, outsetRect(box, off*0.5+width*0.5)), width+2, haloCol)
		}
		p.StrokePath(poly.WriteTo(scratch, outsetRect(box, off+width*0.5)), width, ringCol)
		return
	}
	if haloCol.A > 0 {
		hBox := outsetRect(box, off*0.5+width*0.5)
		hRad := outsetRadii(radii, off*0.5+width*0.5)
		if equalRadii(hRad) {
			p.Stroke(hBox, haloCol, hRad[0], width+2)
		} else if path := writeRoundedRectPath(scratch, hBox, hRad); path != nil {
			p.StrokePath(path, width+2, haloCol)
		}
	}
	p.FocusRing(box, radii)
}

func insetRect(r ui.Rect, inset float32) ui.Rect {
	if inset <= 0 {
		return r
	}
	return ui.Rect{
		X: r.X + inset,
		Y: r.Y + inset,
		W: max(r.W-2*inset, 0),
		H: max(r.H-2*inset, 0),
	}
}

func insetRadii(radii [4]float32, inset float32) [4]float32 {
	if inset <= 0 {
		return radii
	}
	return [4]float32{
		max(radii[0]-inset, 0),
		max(radii[1]-inset, 0),
		max(radii[2]-inset, 0),
		max(radii[3]-inset, 0),
	}
}

func clampF32(v, lo, hi float32) float32 {
	if math.IsNaN(float64(v)) || v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absF32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
