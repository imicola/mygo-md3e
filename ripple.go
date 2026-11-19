package md3e

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/shape"
	"mygo-md3e/tokens"
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

type rippleLocalKey struct{}

type rippleState struct {
	waves          []rippleWave
	pointerHeld    bool
	sawPressedTrue bool

	layerAlpha     float32
	layerTarget    float32
	layerAnimating bool
	layerPrevNow   time.Time
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
	return attachOverlayFeedback(c, el, false, opt)
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
	return attachOverlayFeedback(c, el, !opt.DisableRipple, opt)
}

func attachOverlayFeedback(c *ui.Context, el ui.Element, withRipple bool, opt StateLayerOptions) ui.Element {
	st := ui.Local(el, rippleLocalKey{}, func() rippleState { return rippleState{} })
	if st == nil {
		return el
	}

	th := CurrentTheme(c)
	ink := opt.Color
	if ink.A == 0 {
		ink = th.Colors.OnSurface
	}

	radii := opt.Radii
	if !opt.HasRadii && radii == ([4]float32{}) && opt.Radius > 0 {
		radii = [4]float32{opt.Radius, opt.Radius, opt.Radius, opt.Radius}
	}

	el.ObserveInput(func(ev ui.InputEvent) {
		if opt.Disabled {
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

	el.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		if opt.Disabled || el.IsDisabled() {
			st.clearTransient()
			st.layerAlpha = 0
			st.layerAnimating = false
			return
		}

		reduceMotion := th.ReduceMotion || p.Preferences().ReduceMotion
		now := p.Now()
		hovered := el.Hovered()
		pressed := el.Pressed()
		focused := el.FocusVisible()

		if pressed {
			st.sawPressedTrue = true
		} else if st.pointerHeld && (st.sawPressedTrue || !hovered) {
			st.releaseWaves(false, now)
		}

		if reduceMotion {
			// Skip circular wave animation under ReduceMotion, retaining immediate static layer feedback.
			st.waves = st.waves[:0]
		}

		var targetAlpha float32
		switch {
		case opt.Dragged:
			targetAlpha = float32(tokens.StateDraggedOpacity)
		case pressed || (st.pointerHeld && (!withRipple || reduceMotion)):
			targetAlpha = float32(tokens.StatePressedOpacity)
		case focused:
			targetAlpha = float32(tokens.StateFocusOpacity)
		case hovered:
			targetAlpha = float32(tokens.StateHoverOpacity)
		}

		needFrame := false
		if reduceMotion {
			st.layerAlpha = targetAlpha
			st.layerAnimating = false
		} else if st.layerAlpha != targetAlpha {
			if !st.layerAnimating || st.layerTarget != targetAlpha {
				st.layerTarget = targetAlpha
				st.layerAnimating = true
				st.layerPrevNow = now
			} else {
				dt := float32(now.Sub(st.layerPrevNow).Seconds())
				st.layerPrevNow = now
				if dt > 0 {
					factor := 1 - float32(math.Exp(-28.0*float64(dt)))
					st.layerAlpha += (targetAlpha - st.layerAlpha) * factor
					if absF32(targetAlpha-st.layerAlpha) < 0.0015 {
						st.layerAlpha = targetAlpha
					}
				}
			}
			if st.layerAlpha != targetAlpha {
				needFrame = true
			} else {
				st.layerAnimating = false
			}
		} else {
			st.layerAnimating = false
		}

		clipRect := insetRect(r, opt.BorderInset)
		clipRadii := insetRadii(radii, opt.BorderInset)
		if clipRect.W <= 0 || clipRect.H <= 0 {
			return
		}

		// Update active ripple waves before painting so settled waves are pruned promptly.
		if withRipple && !reduceMotion && len(st.waves) > 0 {
			kept := st.waves[:0]
			for i := range st.waves {
				w := st.waves[i]
				ox, oy := rippleOrigin(&w, opt.BorderInset, clipRect)
				_, _, fadeAlpha, animating, alive := evalRippleWave(&w, ox, oy, clipRect, now)
				if alive && fadeAlpha > 0.001 {
					kept = append(kept, w)
					if animating {
						needFrame = true
					}
				}
			}
			st.waves = kept
		}

		paintInClip(p, clipRect, clipRadii, opt.Shape, func() {
			if st.layerAlpha > 0.001 {
				layerInk := ink.Alpha(st.layerAlpha)
				if opt.Shape != nil {
					if path := opt.Shape.ToPath(clipRect); path != nil {
						p.FillPath(path, layerInk)
					}
				} else {
					p.FillRadii(clipRect, clipRadii, layerInk)
				}
			}

			if withRipple && !reduceMotion {
				for i := range st.waves {
					w := &st.waves[i]
					ox, oy := rippleOrigin(w, opt.BorderInset, clipRect)
					curR, enterAlpha, fadeAlpha, _, _ := evalRippleWave(w, ox, oy, clipRect, now)
					waveAlpha := float32(tokens.StatePressedOpacity) * enterAlpha * fadeAlpha
					if waveAlpha <= 0.001 || curR <= 0 {
						continue
					}
					waveInk := ink.Alpha(waveAlpha)
					p.FillRadialGradient(clipRect, ui.RadialGradient{
						Center: ui.Point{X: ox / clipRect.W, Y: oy / clipRect.H},
						Radius: curR,
						Stops: []ui.GradientStop{
							{Color: waveInk, At: 0},
							{Color: waveInk, At: 0.80},
							{Color: ui.Transparent, At: 1},
						},
					}, 0)
				}
			}
		})

		if needFrame {
			p.After(16 * time.Millisecond)
		}
	})

	return el
}

func rippleOrigin(w *rippleWave, inset float32, inner ui.Rect) (float32, float32) {
	if w.centered {
		return inner.W * 0.5, inner.H * 0.5
	}
	ox := clampF32(w.ox-inset, 0, inner.W)
	oy := clampF32(w.oy-inset, 0, inner.H)
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
	if v < lo {
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
