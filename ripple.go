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
	// BorderInset shrinks the clip box inward by BorderInset DIPs (e.g., stroke width).
	BorderInset float32
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

type rippleState struct {
	waves          []rippleWave
	pointerHeld    bool
	sawPressedTrue bool
	allowWaves     bool
	hasDrawOver    bool
	hoverHook      func(p *ui.Painter, now time.Time, hovered bool)

	focusHandle *ui.Handle
	srv         ui.Services

	halo haloFadeState
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
	st := ui.Local(el, rippleLocalKey{}, func() rippleState { return rippleState{} })
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

	if withRipple {
		if c != nil && !opt.Disabled {
			if st.focusHandle == nil {
				st.focusHandle = new(ui.Handle)
			}
			el.Bind(st.focusHandle)
		}
		el.ObserveInput(func(ev ui.InputEvent) {
			if opt.Disabled {
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
		if opt.Disabled || el.IsDisabled() {
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
		if st.hoverHook != nil {
			st.hoverHook(p, now, hovered)
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
		targetAlpha := resolveStateLayerTarget(false, opt.Dragged, pressed || st.pointerHeld, !withRipple || reduceMotion, focused, hovered)
		layerAlpha, needFrame := st.halo.step(targetAlpha, now, reduceMotion)

		box, boxRadii := r, radii
		if opt.Clip != nil {
			box, boxRadii = opt.Clip(r)
		}
		clipRect := insetRect(box, opt.BorderInset)
		clipRadii := insetRadii(boxRadii, opt.BorderInset)
		if clipRect.W <= 0 || clipRect.H <= 0 {
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
		if layerAlpha <= 0.001 && len(wavesToDraw) == 0 {
			return
		}

		paintBody := func(polyPath *ui.Path) {
			if layerAlpha > 0.001 {
				layerInk := ink.Alpha(layerAlpha)
				if polyPath != nil {
					p.FillPath(polyPath, layerInk)
				} else {
					p.FillRadii(clipRect, clipRadii, layerInk)
				}
			}

			for i := range wavesToDraw {
				w := &wavesToDraw[i]
				waveInk := ink.Alpha(w.waveAlpha)
				cx, cy := w.ox/clipRect.W, w.oy/clipRect.H
				if cx == 0 && cy == 0 {
					// ui's RadialGradient reads the zero Point as the box
					// center; nudge a top-left origin off it.
					cx = 1e-4
				}
				p.FillRadialGradient(clipRect, ui.RadialGradient{
					Center: ui.Point{X: cx, Y: cy},
					Radius: w.curR,
					Stops: []ui.GradientStop{
						{Color: waveInk, At: 0},
						{Color: waveInk, At: 0.80},
						{Color: ui.Transparent, At: 1},
					},
				}, 0)
			}
		}

		if opt.Shape != nil {
			if path := opt.Shape.ToPath(clipRect); path != nil {
				p.ClipPath(path, func() {
					paintBody(path)
				})
			}
			return
		}
		paintInClip(p, clipRect, clipRadii, nil, func() {
			paintBody(nil)
		})
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

func paintInClip(p *ui.Painter, box ui.Rect, radii [4]float32, poly *shape.RoundedPolygon, fn func()) {
	if poly != nil {
		shape.ClipShape(p, poly, box, fn)
		return
	}
	if radii[0] == radii[1] && radii[1] == radii[2] && radii[2] == radii[3] {
		p.Clip(box, radii[0], fn)
		return
	}
	if path := roundedRectPath(box, radii); path != nil {
		p.ClipPath(path, fn)
	}
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
