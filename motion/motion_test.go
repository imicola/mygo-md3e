package motion_test

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
)

var (
	schemes = []motion.Scheme{motion.Expressive, motion.Standard}
	tokens  = []motion.Token{
		motion.FastSpatial, motion.DefaultSpatial, motion.SlowSpatial,
		motion.FastEffects, motion.DefaultEffects, motion.SlowEffects,
	}
	spatialTokens = []motion.Token{motion.FastSpatial, motion.DefaultSpatial, motion.SlowSpatial}
	effectTokens  = []motion.Token{motion.FastEffects, motion.DefaultEffects, motion.SlowEffects}
)

// TestSpecTable checks all twelve presets against docs/md3e-spec.md §2.1,
// including the damping conversion c = 2ζ√k of a unit mass.
func TestSpecTable(t *testing.T) {
	tests := []struct {
		scheme    motion.Scheme
		token     motion.Token
		ratio     float64
		stiffness float64
		damping   float64
	}{
		{motion.Expressive, motion.FastSpatial, 0.6, 800, 33.9411},
		{motion.Expressive, motion.DefaultSpatial, 0.8, 380, 31.1897},
		{motion.Expressive, motion.SlowSpatial, 0.8, 200, 22.6274},
		{motion.Standard, motion.FastSpatial, 0.9, 1400, 67.3498},
		{motion.Standard, motion.DefaultSpatial, 0.9, 700, 47.6235},
		{motion.Standard, motion.SlowSpatial, 0.9, 300, 31.1769},
		{motion.Expressive, motion.FastEffects, 1, 3800, 123.2883},
		{motion.Expressive, motion.DefaultEffects, 1, 1600, 80},
		{motion.Expressive, motion.SlowEffects, 1, 800, 56.5685},
		{motion.Standard, motion.FastEffects, 1, 3800, 123.2883},
		{motion.Standard, motion.DefaultEffects, 1, 1600, 80},
		{motion.Standard, motion.SlowEffects, 1, 800, 56.5685},
	}
	for _, tt := range tests {
		t.Run(tt.scheme.String()+"/"+tt.token.String(), func(t *testing.T) {
			s := motion.SpecOf(tt.scheme, tt.token)
			if s.DampingRatio != tt.ratio || s.Stiffness != tt.stiffness {
				t.Fatalf("spec = ζ%v k%v, want ζ%v k%v", s.DampingRatio, s.Stiffness, tt.ratio, tt.stiffness)
			}
			if got, want := s.Damping(), 2*tt.ratio*math.Sqrt(tt.stiffness); math.Abs(got-want) > 1e-12 {
				t.Errorf("damping = %v, want %v", got, want)
			}
			o := s.SpringOptions()
			if o.Mass != 1 || o.Stiffness != tt.stiffness {
				t.Errorf("spring = %+v, want mass 1 and stiffness %v", o, tt.stiffness)
			}
			if math.Abs(o.Damping-tt.damping) > 1e-3 {
				t.Errorf("spring damping = %v, want about %v", o.Damping, tt.damping)
			}
			// A damping ratio, not the damping coefficient, must never reach
			// the physical options.
			if o.Damping < 20 {
				t.Errorf("spring damping = %v, too low for a critically damped or near spring", o.Damping)
			}
		})
	}
}

// TestProgressOvershoot samples the spring trajectories: the Expressive
// spatial fast spring overshoots visibly, and the effect springs, critically
// damped, never do.
func TestProgressOvershoot(t *testing.T) {
	for _, scheme := range schemes {
		for _, token := range tokens {
			spec := motion.SpecOf(scheme, token)
			o := spec.SpringOptions()
			peak, low := 0.0, math.Inf(1)
			for ms := 1; ms <= 1500; ms++ {
				v := o.Progress(time.Duration(ms) * time.Millisecond)
				peak = math.Max(peak, v)
				low = math.Min(low, v)
			}
			name := scheme.String() + "/" + token.String()
			if token.Kind() == motion.Effects {
				if peak > 1.001 {
					t.Errorf("%s peaks at %v, want no overshoot (≤ 1.001)", name, peak)
				}
				if low < 0 {
					t.Errorf("%s undershoots to %v", name, low)
				}
				continue
			}
			if token == motion.FastSpatial && scheme == motion.Expressive && peak <= 1.04 {
				t.Errorf("%s peaks at %v, want a visible overshoot (> 1.04)", name, peak)
			}
			t.Logf("%s peak %.4f", name, peak)
		}
	}
}

// TestSettleTime checks the settle times of the table are within the
// documented magnitude: fast springs under half a second, slow spatial ones
// under a second, effects under half a second.
func TestSettleTime(t *testing.T) {
	for _, scheme := range schemes {
		for _, token := range tokens {
			spec := motion.SpecOf(scheme, token)
			settle := spec.SettleTime()
			name := scheme.String() + "/" + token.String()
			if settle <= 0 || settle >= time.Second {
				t.Errorf("%s settles in %v, want under a second", name, settle)
			}
			t.Logf("%s settles in %v", name, settle)
		}
	}
}

// animationView returns a view that animates one value with the token's
// animation, recording the value it shows this frame.
func animationView(scheme motion.Scheme, token motion.Token, target, shown *float32) func(*ui.Context) {
	return func(c *ui.Context) {
		e := ui.Box(c).Size(10, 10)
		*shown = e.AnimateTo("v", *target, motion.Resolve(scheme, token, c.Preferences().ReduceMotion))
	}
}

// TestAnimationSettles drives the real animation of each token in a tester
// and checks it comes to rest, asking for no further frames.
func TestAnimationSettles(t *testing.T) {
	for _, scheme := range schemes {
		for _, token := range tokens {
			t.Run(scheme.String()+"/"+token.String(), func(t *testing.T) {
				var target, shown float32
				tt := ui.NewTester(animationView(scheme, token, &target, &shown), 100, 100)

				target = 1
				tt.Frame()
				if !tt.FramePending() {
					t.Fatal("the animation asked for no frame while starting")
				}
				tt.Advance(motion.SpecOf(scheme, token).SettleTime() + 200*time.Millisecond)
				if shown != 1 {
					t.Errorf("value = %v after settling, want 1", shown)
				}
				if tt.FramePending() {
					t.Error("the view keeps asking for frames after the animation settles")
				}
			})
		}
	}
}

// TestReduceMotionSnapsSpatial checks that a spatial token with motion to be
// reduced reaches its target in the frame of the change and asks for no
// frames at all.
func TestReduceMotionSnapsSpatial(t *testing.T) {
	for _, scheme := range schemes {
		for _, token := range spatialTokens {
			t.Run(scheme.String()+"/"+token.String(), func(t *testing.T) {
				var target, shown float32
				view := animationView(scheme, token, &target, &shown)
				tt := ui.NewTesterWith(view, 100, 100,
					ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})

				target = 1
				tt.Frame()
				if shown != 1 {
					t.Errorf("value = %v in the frame of the change, want 1", shown)
				}
				if tt.FramePending() {
					t.Error("the view asks for frames with motion to be reduced")
				}
			})
		}
	}
}

// TestEffectsAnimateWithReducedMotion checks that effects still fade while
// the desktop asks for less motion, as their NonMotion says they may, and
// then rest without asking for frames.
func TestEffectsAnimateWithReducedMotion(t *testing.T) {
	for _, scheme := range schemes {
		for _, token := range effectTokens {
			t.Run(scheme.String()+"/"+token.String(), func(t *testing.T) {
				var target, shown float32
				view := animationView(scheme, token, &target, &shown)
				tt := ui.NewTesterWith(view, 100, 100,
					ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})

				target = 1
				tt.Frame()
				if shown != 0 {
					t.Errorf("value = %v in the frame of the change, want 0 (effects still animate)", shown)
				}
				tt.Advance(motion.SpecOf(scheme, token).SettleTime() + 200*time.Millisecond)
				if shown != 1 {
					t.Errorf("value = %v after settling, want 1", shown)
				}
				if tt.FramePending() {
					t.Error("the view keeps asking for frames after the effect settles")
				}
			})
		}
	}
}

// TestResolveFlags checks the animation flags each kind of token carries.
func TestResolveFlags(t *testing.T) {
	for _, scheme := range schemes {
		for _, token := range effectTokens {
			a := motion.Resolve(scheme, token, false)
			if !a.Oklab || !a.NonMotion {
				t.Errorf("%v effects animation lacks Oklab or NonMotion: %+v", token, a)
			}
			if a.Duration != 0 || a.Spring.Stiffness == 0 {
				t.Errorf("%v effects animation is not a spring: %+v", token, a)
			}
			if a.Spring.Damping != 2*math.Sqrt(a.Spring.Stiffness) {
				t.Errorf("%v effects spring is not critically damped: c=%v", token, a.Spring.Damping)
			}
			if reduced := motion.Resolve(scheme, token, true); reduced.Duration != 0 || !reduced.NonMotion {
				t.Errorf("%v effects animation snaps with reduced motion: %+v", token, reduced)
			}
		}
		for _, token := range spatialTokens {
			a := motion.Resolve(scheme, token, false)
			if a.Duration != 0 || a.Oklab || a.NonMotion {
				t.Errorf("%v spatial animation = %+v, want a plain spring", token, a)
			}
			if reduced := motion.Resolve(scheme, token, true); reduced.Duration == 0 {
				t.Errorf("%v spatial animation does not snap with reduced motion: %+v", token, reduced)
			}
		}
	}
}
