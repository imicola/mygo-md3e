// Package motion holds the Material Design 3 Expressive motion physics: the
// Expressive and Standard schemes' spatial and effects springs, and their
// conversion to the ui package's physical spring options.
//
// MD3E describes a spring by its damping ratio ζ and stiffness k. The ui
// package wants the viscous damping coefficient c, so every spring here is
// converted with c = 2ζ√(k·m) for a unit mass rather than handing ζ to
// ui.SpringOptions.Damping, which would leave the spring badly underdamped.
//
// The package depends only on the standard library and ui.Animation and
// ui.SpringOptions; it knows nothing of layout or state.
package motion

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Scheme selects the feel of the motion tokens.
type Scheme int

const (
	// Expressive is the default MD3E scheme: playful spatial motion with a
	// visible overshoot.
	Expressive Scheme = iota
	// Standard is the restrained scheme for tools: less bounce, and faster
	// settling.
	Standard
)

// String returns the scheme's name.
func (s Scheme) String() string {
	if s == Standard {
		return "Standard"
	}
	return "Expressive"
}

// Token names one of the six MD3E springs of a scheme.
type Token int

const (
	// FastSpatial moves small spatial changes: a pressed corner radius, a
	// switch thumb.
	FastSpatial Token = iota
	// DefaultSpatial moves most spatial changes: positions, sizes, shapes.
	DefaultSpatial
	// SlowSpatial moves large spatial changes: a sheet, a full-screen
	// transition.
	SlowSpatial
	// FastEffects moves fast non-spatial changes: a hover tint.
	FastEffects
	// DefaultEffects moves most non-spatial changes: colors and opacity.
	DefaultEffects
	// SlowEffects moves large non-spatial changes: a long fade.
	SlowEffects
)

// Kind says whether a token moves space or an effect.
type Kind int

const (
	// Spatial tokens move position, size, corner radius and shape; MD3E lets
	// them overshoot.
	Spatial Kind = iota
	// Effects tokens move color and opacity; MD3E keeps them critically
	// damped so they never overshoot.
	Effects
)

// Kind returns whether the token moves space or an effect.
func (t Token) Kind() Kind {
	if t >= FastEffects {
		return Effects
	}
	return Spatial
}

// String returns the token's name.
func (t Token) String() string {
	switch t {
	case FastSpatial:
		return "FastSpatial"
	case DefaultSpatial:
		return "DefaultSpatial"
	case SlowSpatial:
		return "SlowSpatial"
	case FastEffects:
		return "FastEffects"
	case DefaultEffects:
		return "DefaultEffects"
	case SlowEffects:
		return "SlowEffects"
	}
	return "Token(?)"
}

// Spec is a spring as MD3E describes it: the scheme and token it belongs to,
// its damping ratio ζ and its stiffness k.
type Spec struct {
	Scheme       Scheme
	Token        Token
	DampingRatio float64
	Stiffness    float64
}

// Damping returns the viscous damping coefficient c = 2ζ√(k·m) of the spec,
// for the unit mass ui springs use.
func (s Spec) Damping() float64 {
	return 2 * s.DampingRatio * math.Sqrt(s.Stiffness)
}

// SpringOptions converts the spec to the physical options of a ui spring:
// {Mass: 1, Stiffness: k, Damping: 2ζ√k}. Never pass DampingRatio straight
// to ui.SpringOptions.Damping.
func (s Spec) SpringOptions() ui.SpringOptions {
	return ui.SpringOptions{Mass: 1, Stiffness: s.Stiffness, Damping: s.Damping()}
}

// SettleTime returns how long the spec's spring takes to come to rest.
func (s Spec) SettleTime() time.Duration {
	return s.SpringOptions().SettleTime()
}

// spring is a row of MD3E's spring table.
type spring struct{ ratio, stiffness float64 }

var (
	// Expressive spatial springs: Fast, Default and Slow.
	expressiveSpatial = [3]spring{{0.6, 800}, {0.8, 380}, {0.8, 200}}
	// Standard spatial springs: Fast, Default and Slow.
	standardSpatial = [3]spring{{0.9, 1400}, {0.9, 700}, {0.9, 300}}
	// Effects springs, shared by both schemes: Fast, Default and Slow, all
	// critically damped.
	effectsSprings = [3]spring{{1, 3800}, {1, 1600}, {1, 800}}
)

// SpecOf returns the MD3E spec of token under scheme. It panics for a token
// outside the six MD3E tokens or an unknown scheme.
func SpecOf(scheme Scheme, token Token) Spec {
	if token < FastSpatial || token > SlowEffects {
		panic("motion: unknown token")
	}
	if scheme != Expressive && scheme != Standard {
		panic("motion: unknown scheme")
	}
	var s spring
	switch token.Kind() {
	case Spatial:
		table := expressiveSpatial
		if scheme == Standard {
			table = standardSpatial
		}
		s = table[token]
	default:
		s = effectsSprings[token-FastEffects]
	}
	return Spec{Scheme: scheme, Token: token, DampingRatio: s.ratio, Stiffness: s.stiffness}
}

// Spring returns the ui spring options of token under scheme, as SpecOf's
// spec converts them.
func Spring(scheme Scheme, token Token) ui.SpringOptions {
	return SpecOf(scheme, token).SpringOptions()
}

// Snap is the animation Resolve gives a spatial token when motion is to be
// reduced: a value reaches its target within the frame after the change.
var Snap = ui.Animation{Duration: time.Nanosecond, Ease: ui.Linear}

// Resolve returns the animation of token under scheme.
//
// Spatial springs overshoot as MD3E allows. Effects springs are critically
// damped, and set Oklab and NonMotion: colors mix in Oklab, and as they move
// nothing on screen they may still fade while the desktop asks for less
// motion. A spatial token with reduceMotion set resolves to Snap; callers
// that pass c.Preferences().ReduceMotion get the snap here and from
// ui.AnimateTo alike.
func Resolve(scheme Scheme, token Token, reduceMotion bool) ui.Animation {
	if token.Kind() == Spatial && reduceMotion {
		return Snap
	}
	a := ui.Animation{Spring: SpecOf(scheme, token).SpringOptions()}
	if token.Kind() == Effects {
		a.Oklab = true
		a.NonMotion = true
	}
	return a
}
