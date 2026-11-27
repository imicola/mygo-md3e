package md3e

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
)

// Canonical MD3E Phase 3 choreography curves (docs/md3e-spec.md §12.3–12.4).
var (
	keyframeQuarterFracs = []float64{0, 0.25, 0.5, 0.75, 1.0}
	keyframeCurveO       = ui.CubicBezier(0.20, 0, 0, 1)
	keyframeCurveI       = ui.CubicBezier(0.40, 0, 1, 1)
	keyframeCurveS       = ui.CubicBezier(0.20, 0, 0.20, 1)
)

// sampleKeyframes steps tt across absolute timestamps frac*duration by
// advancing only the delta between consecutive sample points, avoiding the
// classic bug of advancing by cumulative duration on every step.
func sampleKeyframes(tt *ui.Tester, duration time.Duration, fracs []float64, fn func(absT time.Duration, frac float64, img *image.RGBA)) {
	var prev time.Duration
	for _, f := range fracs {
		absT := time.Duration(math.Round(float64(duration) * f))
		if delta := absT - prev; delta > 0 {
			tt.Advance(delta)
			prev = absT
		}
		fn(absT, f, tt.Image())
	}
}

// invertCompositedAlpha solves got = alpha*fg + (1-alpha)*bg using the RGB
// channel with the widest contrast delta between fg and bg.
func invertCompositedAlpha(got color.RGBA, fg, bg ui.Color) float64 {
	deltas := [3]float64{
		float64(fg.R) - float64(bg.R),
		float64(fg.G) - float64(bg.G),
		float64(fg.B) - float64(bg.B),
	}
	vals := [3]float64{
		float64(got.R) - float64(bg.R),
		float64(got.G) - float64(bg.G),
		float64(got.B) - float64(bg.B),
	}
	best := 0
	for i := 1; i < 3; i++ {
		if math.Abs(deltas[i]) > math.Abs(deltas[best]) {
			best = i
		}
	}
	if math.Abs(deltas[best]) < 1 {
		return 0
	}
	return vals[best] / deltas[best]
}

// assertCompositedAlphaNear verifies that pixel got matches alpha*fg + (1-alpha)*bg
// within tol8Bit (typically 2 8-bit code values per docs/md3e-spec.md §12.4).
func assertCompositedAlphaNear(t *testing.T, label string, got color.RGBA, fg, bg ui.Color, wantAlpha float64, tol8Bit int) {
	t.Helper()
	wantR := float64(bg.R) + wantAlpha*(float64(fg.R)-float64(bg.R))
	wantG := float64(bg.G) + wantAlpha*(float64(fg.G)-float64(bg.G))
	wantB := float64(bg.B) + wantAlpha*(float64(fg.B)-float64(bg.B))
	dr := math.Abs(float64(got.R) - wantR)
	dg := math.Abs(float64(got.G) - wantG)
	db := math.Abs(float64(got.B) - wantB)
	if dr > float64(tol8Bit) || dg > float64(tol8Bit) || db > float64(tol8Bit) {
		gotAlpha := invertCompositedAlpha(got, fg, bg)
		t.Fatalf("%s: got RGBA(%d,%d,%d) (alpha≈%.3f), want alpha=%.3f RGBA(%.0f,%.0f,%.0f) ±%d",
			label, got.R, got.G, got.B, gotAlpha, wantAlpha, wantR, wantG, wantB, tol8Bit)
	}
}

func keyframeAbsInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// scanPanelBounds returns the tight pixel bounding box of all pixels in img
// differing from bg by more than tol in any RGB channel.
func scanPanelBounds(img *image.RGBA, bg ui.Color, tol int) (ui.Rect, bool) {
	b := img.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px := img.RGBAAt(x, y)
			if keyframeAbsInt(int(px.R)-int(bg.R)) > tol ||
				keyframeAbsInt(int(px.G)-int(bg.G)) > tol ||
				keyframeAbsInt(int(px.B)-int(bg.B)) > tol {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if maxX < minX || maxY < minY {
		return ui.Rect{}, false
	}
	return ui.Rect{
		X: float32(minX),
		Y: float32(minY),
		W: float32(maxX - minX + 1),
		H: float32(maxY - minY + 1),
	}, true
}

func assertBoundsNear(t *testing.T, label string, got, want ui.Rect, tolPx float32) {
	t.Helper()
	if absF32(got.X-want.X) > tolPx || absF32(got.Y-want.Y) > tolPx ||
		absF32(got.W-want.W) > tolPx || absF32(got.H-want.H) > tolPx {
		t.Fatalf("%s: bounds %+v differ from want %+v by more than %.1fpx", label, got, want, tolPx)
	}
}

// cubicBezierY solves B_x(u) = t on [0,1] via Newton/bisection and returns B_y(u).
func cubicBezierY(x1, y1, x2, y2, t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	eval := func(a, b, u float64) float64 {
		inv := 1 - u
		return 3*inv*inv*u*a + 3*inv*u*u*b + u*u*u
	}
	lo, hi := 0.0, 1.0
	u := t
	for range 40 {
		x := eval(x1, x2, u)
		if math.Abs(x-t) < 1e-9 {
			break
		}
		if x < t {
			lo = u
		} else {
			hi = u
		}
		u = (lo + hi) * 0.5
	}
	return eval(y1, y2, u)
}

// assertPureHoverZeroRebuilds verifies that hovering onto target and leaving to
// leave produces zero tree rebuilds during the hover/leave paint animations and
// leaves zero pending frames once settled.
func assertPureHoverZeroRebuilds(t *testing.T, tt *ui.Tester, builds *int, target, leave ui.Point, settle time.Duration) {
	t.Helper()
	tt.Move(leave.X, leave.Y)
	tt.Advance(settle)
	beforeEnter := *builds

	tt.Move(target.X, target.Y)
	afterEnterMove := *builds
	if afterEnterMove > beforeEnter+1 {
		t.Fatalf("hover enter caused %d builds on pointer move (want <= 1)", afterEnterMove-beforeEnter)
	}
	tt.Advance(settle)
	if *builds != afterEnterMove {
		t.Fatalf("hover enter paint ramp caused %d tree rebuilds during Advance, want 0", *builds-afterEnterMove)
	}
	if tt.FramePending() {
		t.Fatal("hover enter did not settle to FramePending() == false")
	}

	tt.Move(leave.X, leave.Y)
	afterLeaveMove := *builds
	if afterLeaveMove > afterEnterMove+1 {
		t.Fatalf("hover leave caused %d builds on pointer move (want <= 1)", afterLeaveMove-afterEnterMove)
	}
	tt.Advance(settle)
	if *builds != afterLeaveMove {
		t.Fatalf("hover leave paint ramp caused %d tree rebuilds during Advance, want 0", *builds-afterLeaveMove)
	}
	if tt.FramePending() {
		t.Fatal("hover leave did not settle to FramePending() == false")
	}
}

// assertReduceMotionZeroFrames asserts immediate zero pending frames and checks
// an extra advance window to rule out lingering timers.
func assertReduceMotionZeroFrames(t *testing.T, tt *ui.Tester, extra time.Duration) {
	t.Helper()
	if tt.FramePending() {
		t.Fatal("ReduceMotion left FramePending() == true immediately")
	}
	if extra > 0 {
		tt.Advance(extra)
		if tt.FramePending() {
			t.Fatalf("ReduceMotion resurrected pending frames within %v", extra)
		}
	}
}

func TestKeyframeBezierQuarterPointsAndSolidPanelTransition(t *testing.T) {
	// 1. Verify exact mathematical quarter points from docs/md3e-spec.md §12.4.
	curves := []struct {
		name           string
		x1, y1, x2, y2 float64
		easing         ui.Easing
		wantQuarters   [3]float64
	}{
		{"O", 0.20, 0, 0, 1, keyframeCurveO, [3]float64{0.607, 0.878, 0.975}},
		{"I", 0.40, 0, 1, 1, keyframeCurveI, [3]float64{0.099, 0.325, 0.630}},
		{"S", 0.20, 0, 0.20, 1, keyframeCurveS, [3]float64{0.448, 0.814, 0.962}},
	}

	bg := ui.RGB(240, 240, 240)
	fg := ui.RGB(20, 60, 200)
	const dur = 200 * time.Millisecond

	for _, tc := range curves {
		t.Run(tc.name, func(t *testing.T) {
			for qi, frac := range []float64{0.25, 0.50, 0.75} {
				gotY := cubicBezierY(tc.x1, tc.y1, tc.x2, tc.y2, frac)
				if math.Abs(gotY-tc.wantQuarters[qi]) > 0.003 {
					t.Fatalf("curve %s at %.2f = %.4f, want %.3f", tc.name, frac, gotY, tc.wantQuarters[qi])
				}
			}

			// 2. Drive a text-free solid panel with ui.ElementTransition and sample
			// at t = 0, D/4, D/2, 3D/4, D verifying composited alpha and bounds.
			shown := false
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Box(c).Size(200, 160).Background(bg).Children(func() {
					if shown {
						ui.Box(c.Key("solid-panel")).
							Absolute().Left(40).Top(40).Size(120, 80).
							Background(fg).
							Transition(ui.ElementTransition{
								Duration: dur,
								Ease:     tc.easing,
								Enter:    &ui.Motion{Opacity: 0},
								Exit:     &ui.Motion{Opacity: 0},
							})
					}
				})
			}, 200, 160)
			tt.Frame()

			shown = true
			tt.Frame()
			sampleKeyframes(tt, dur, keyframeQuarterFracs, func(absT time.Duration, frac float64, img *image.RGBA) {
				wantA := cubicBezierY(tc.x1, tc.y1, tc.x2, tc.y2, frac)
				centerPx := img.RGBAAt(100, 80)
				assertCompositedAlphaNear(t, tc.name+" enter "+absT.String(), centerPx, fg, bg, wantA, 2)
				if frac >= 0.25 {
					bounds, ok := scanPanelBounds(img, bg, 3)
					if !ok {
						t.Fatalf("%s enter %v: solid panel bounds not detected", tc.name, absT)
					}
					assertBoundsNear(t, tc.name+" enter bounds", bounds, ui.Rect{X: 40, Y: 40, W: 120, H: 80}, 1)
				}
			})
			tt.Advance(20 * time.Millisecond)
			if tt.FramePending() {
				t.Fatalf("%s enter did not settle after t=D", tc.name)
			}

			// Exit transition: alpha decays as 1 - curve(frac).
			shown = false
			tt.Frame()
			sampleKeyframes(tt, dur, keyframeQuarterFracs, func(absT time.Duration, frac float64, img *image.RGBA) {
				wantA := 1 - cubicBezierY(tc.x1, tc.y1, tc.x2, tc.y2, frac)
				centerPx := img.RGBAAt(100, 80)
				assertCompositedAlphaNear(t, tc.name+" exit "+absT.String(), centerPx, fg, bg, wantA, 2)
			})
			tt.Advance(50 * time.Millisecond)
			if tt.FramePending() {
				t.Fatalf("%s exit left pending frames after settle", tc.name)
			}
		})
	}
}

func TestKeyframeSpringOvershootBounds(t *testing.T) {
	cases := []struct {
		name         string
		tok          motion.Token
		maxOvershoot float64 // fraction above 1.0 (e.g. 0.0152 for zeta=0.8, 0.0948 for zeta=0.6)
		minOvershoot float64 // underdamped spatial springs visibly overshoot slightly
	}{
		{"ExpressiveFastSpatial_z0.6", motion.FastSpatial, 0.11, 0.05},
		{"ExpressiveDefaultSpatial_z0.8", motion.DefaultSpatial, 0.025, 0.005},
		{"ExpressiveFastEffects_z1.0", motion.FastEffects, 0.001, 0},
	}

	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Motion: motion.Expressive})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := motion.SpecOf(motion.Expressive, tc.tok)
			anim := th.Animation(tc.tok)
			target := float32(0)
			curVal := float32(0)

			tt := ui.NewTester(func(c *ui.Context) {
				el := ui.Box(c.Key("spring-box")).Size(20, 20)
				curVal = animateValue(el, animChan0, target, anim, false)
			}, 40, 40)
			tt.Frame()

			target = 1
			tt.Frame()
			peak := float64(curVal)
			window := spec.SettleTime() + 100*time.Millisecond
			for elapsed := time.Duration(0); elapsed < window; elapsed += 8 * time.Millisecond {
				tt.Advance(8 * time.Millisecond)
				if float64(curVal) > peak {
					peak = float64(curVal)
				}
			}
			overshoot := peak - 1.0
			if overshoot > tc.maxOvershoot {
				t.Fatalf("%s peak=%.4f overshoot=%.4f exceeds upper bound %.4f", tc.name, peak, overshoot, tc.maxOvershoot)
			}
			if overshoot < tc.minOvershoot {
				t.Fatalf("%s peak=%.4f overshoot=%.4f below expected spatial bounce %.4f", tc.name, peak, overshoot, tc.minOvershoot)
			}
			if absF32(curVal-1) > 0.01 {
				t.Fatalf("%s did not converge to 1.0 (got %.4f)", tc.name, curVal)
			}
			if tt.FramePending() {
				t.Fatalf("%s left FramePending() == true after settle window %v", tc.name, window)
			}
		})
	}
}

func TestKeyframePureHoverZeroRebuildsAndReduceMotionContract(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			ui.Column(c).Size(320, 140).Background(canvasRGB).Padding(24).Gap(12).Children(func() {
				Button(c.Key("hover-btn"), "Hover Me")
			})
		})
	}, 320, 140)
	tt.Advance(300 * time.Millisecond)

	btn := buttonInfo(t, tt, ui.RoleButton, "Hover Me")
	bx, by := center(btn.Bounds)
	assertPureHoverZeroRebuilds(t, tt, &builds, ui.Point{X: bx, Y: by}, ui.Point{X: 2, Y: 2}, 400*time.Millisecond)

	// System-level ReduceMotion: hover enter/leave converges with zero pending frames.
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Move(bx, by)
	assertReduceMotionZeroFrames(t, tt, 200*time.Millisecond)
	tt.Move(2, 2)
	assertReduceMotionZeroFrames(t, tt, 200*time.Millisecond)
}

// TestKeyframeRouterMidTransitionCancelBoundary documents and verifies upstream
// ui.Router behavior when switching Transition to TransitionNone and toggling
// DisableAnimations during a route change (phase3-architecture-review.md §4.1).
func TestKeyframeRouterMidTransitionCancelBoundary(t *testing.T) {
	rtr := ui.NewRouter("/a")
	rtr.Transition = ui.TransitionFade
	instant := false

	tt := ui.NewTester(func(c *ui.Context) {
		if instant {
			c.DisableAnimations()
		}
		rtr.View(c, func(r *ui.Route) {
			switch {
			case r.Match("/a"):
				ui.Box(c).Fill().Background(ui.RGB(200, 40, 40)).Children(func() { ui.Text(c, "Page A") })
			case r.Match("/b"):
				ui.Box(c).Fill().Background(ui.RGB(40, 200, 40)).Children(func() { ui.Text(c, "Page B") })
			default:
				ui.Box(c).Fill().Background(ui.RGB(40, 40, 200)).Children(func() { ui.Text(c, "Page C") })
			}
		}).Fill()
	}, 240, 160)
	tt.Frame()

	// Normal TransitionFade starts animating on Push("/b").
	rtr.Push("/b")
	tt.Frame()
	tt.Advance(40 * time.Millisecond)

	// Setting TransitionNone prevents new transitions from starting animations.
	rtr.Transition = ui.TransitionNone
	instant = true
	rtr.Push("/c")
	tt.Frame()
	if !tt.HasText("Page C") {
		t.Fatal("Router did not reach Page C under TransitionNone + DisableAnimations")
	}
	// Advance past any prior fade window and verify zero lingering frames.
	tt.Advance(300 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("Router left pending frames after TransitionNone settle")
	}
}
