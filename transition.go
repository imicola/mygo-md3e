package md3e

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
)

// TransitionMode selects the motion choreography used by PageTransition and
// AnimatedContent when switching between keyed pages.
type TransitionMode int

const (
	// TransitionFadeThrough is the default Material Design 3 Fade-Through
	// choreography: outgoing content fades out cleanly while incoming content
	// enters with a subtle scale-up (0.96 -> 1), gentle upward rise, and fade.
	TransitionFadeThrough TransitionMode = iota
	// TransitionSharedAxisY slides outgoing and incoming pages along the
	// vertical axis according to Direction (suited for vertical NavigationRail
	// page switching).
	TransitionSharedAxisY
	// TransitionSharedAxisX slides outgoing and incoming pages along the
	// horizontal axis according to Direction, mirroring automatically in
	// right-to-left layouts.
	TransitionSharedAxisX
	// TransitionNone swaps pages immediately with no transition animation.
	TransitionNone
)

// TransitionOptions configures PageTransition and AnimatedContent. The zero
// value uses TransitionFadeThrough with MD3E spring timing and forward
// direction (+1).
type TransitionOptions struct {
	// Mode selects the transition pattern (default TransitionFadeThrough).
	Mode TransitionMode
	// Direction controls the axis sign for SharedAxis modes: > 0 moves forward
	// (+1), < 0 moves backward (-1), and 0 auto-infers from ordered integer keys
	// (defaulting to +1 for other key types). TransitionSharedAxisX mirrors
	// automatically in right-to-left layouts.
	Direction int
	// Distance overrides the travel distance in DIP for SharedAxis modes
	// (default 30 DIP) and the subtle vertical rise for TransitionFadeThrough
	// (default 8 DIP). Non-positive or non-finite values use the mode default.
	Distance float32
	// Scale overrides the entering initial scale for TransitionFadeThrough
	// (default 0.96). Values <= 0, >= 1, or non-finite use the default.
	Scale float32
	// AfterExits holds the incoming page unseen until the outgoing page finishes
	// leaving, producing a sequential two-phase transition instead of an
	// overlapping cross-transition.
	AfterExits bool
}

const (
	defaultFadeThroughDistance float32 = 8
	defaultSharedAxisDistance  float32 = 30
	defaultFadeThroughScale    float32 = 0.96
)

type (
	pageTransitionStateKey struct{}
	pageTransitionChildKey struct{ key any }
)

type pageTransitionState struct {
	hasKey  bool
	lastKey any
	dir     float32
}

// PageTransition wraps a keyed view in a clipped stage container and animates
// content swaps whenever key changes. Only key changes trigger transitions;
// repeated builds with the same key produce zero extra animation frames, even
// if the surrounding pane resizes.
//
// Under ReduceMotion (c.Preferences().ReduceMotion or Theme.ReduceMotion) or
// TransitionNone, pages swap immediately with zero pending animation frames.
// Outgoing pages render as inert exit ghosts (non-interactive, hidden from
// assistive technology, and with Element.Valid() == false in draw callbacks).
//
// Like ThemeScope, PageTransition sizes to its content by default; chain
// .Fill() or .Grow(1) when placing it inside a full-height layout pane.
// Keys must be comparable.
func PageTransition(c *ui.Context, key any, build func(), opts ...TransitionOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: PageTransition takes at most one TransitionOptions")
	}
	var o TransitionOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if c == nil {
		return ui.Element{}
	}

	th := CurrentTheme(c)
	reduce := c.Preferences().ReduceMotion || th.ReduceMotion
	rtl := c.Direction() == ui.RightToLeft

	mode := o.Mode
	if mode < TransitionFadeThrough || mode > TransitionNone {
		mode = TransitionFadeThrough
	}

	// Consume any caller-supplied c.Key(...) on the persistent stage container
	// before creating internal state or keyed page children.
	stage := ui.Column(c).Clip()
	st := ui.Local(stage, pageTransitionStateKey{}, func() pageTransitionState {
		return pageTransitionState{dir: 1}
	})

	if !st.hasKey {
		st.hasKey = true
		st.lastKey = key
		st.dir = explicitTransitionDir(o.Direction, 1)
	} else if st.lastKey != key {
		st.dir = resolveTransitionDir(o.Direction, st.lastKey, key)
		st.lastKey = key
	} else if o.Direction != 0 {
		st.dir = explicitTransitionDir(o.Direction, st.dir)
	}

	if reduce || mode == TransitionNone {
		// Suppress any fade-only exit ghost if ReduceMotion toggled on between frames.
		stage.ChildExit(ui.Motion{Opacity: 1})
		stage.Children(func() {
			page := ui.Column(c.Key(pageTransitionChildKey{key: key})).Grow(1)
			if build != nil {
				page.Children(build)
			}
		})
		return stage
	}

	scheme := th.Motion
	if scheme != motion.Expressive && scheme != motion.Standard {
		scheme = motion.Expressive
	}

	var (
		enter    *ui.Motion
		exit     ui.Motion
		anim     ui.Animation
		exitDur  = motion.SpecOf(scheme, motion.FastEffects).SettleTime()
	)

	switch mode {
	case TransitionSharedAxisY:
		dist := sanitizePositiveF32(o.Distance, defaultSharedAxisDistance)
		enter = &ui.Motion{Opacity: 0, Y: dist * st.dir}
		exit = ui.Motion{Opacity: 0, Y: -dist * st.dir}
		anim = motion.Resolve(scheme, motion.DefaultSpatial, false)
		exitDur = min(exitDur, 150*time.Millisecond)
	case TransitionSharedAxisX:
		dist := sanitizePositiveF32(o.Distance, defaultSharedAxisDistance)
		axisDir := st.dir
		if rtl {
			axisDir = -axisDir
		}
		enter = &ui.Motion{Opacity: 0, X: dist * axisDir}
		exit = ui.Motion{Opacity: 0, X: -dist * axisDir}
		anim = motion.Resolve(scheme, motion.DefaultSpatial, false)
		exitDur = min(exitDur, 150*time.Millisecond)
	default: // TransitionFadeThrough
		dist := sanitizePositiveF32(o.Distance, defaultFadeThroughDistance)
		scale := sanitizeTransitionScale(o.Scale, defaultFadeThroughScale)
		enter = &ui.Motion{Opacity: 0, Scale: scale, Y: dist}
		exit = ui.Motion{Opacity: 0}
		anim = motion.Resolve(scheme, motion.DefaultEffects, false)
		exitDur = min(exitDur, 120*time.Millisecond)
	}

	// ChildExit ensures an outgoing page whose direction flipped on this exact
	// frame exits along the updated direction rather than its stale entry direction.
	stage.ChildExit(exit)

	exitCopy := exit
	tr := ui.ElementTransition{
		Animation:    &anim,
		ExitDuration: exitDur,
		ExitEase:     ui.EaseOut,
		// Restrict property tracking to Colors on this unstyled transparent page
		// wrapper so layout/size changes never trigger transitions while key is
		// unchanged; Enter and Exit still animate position, scale, and opacity.
		Colors:     true,
		Enter:      enter,
		Exit:       &exitCopy,
		AfterExits: o.AfterExits,
	}

	stage.Children(func() {
		page := ui.Column(c.Key(pageTransitionChildKey{key: key})).Grow(1).Transition(tr)
		if build != nil {
			page.Children(build)
		}
	})
	return stage
}

// AnimatedContent is a type-safe generic wrapper around PageTransition for
// comparable keys.
func AnimatedContent[K comparable](c *ui.Context, key K, build func(), opts ...TransitionOptions) ui.Element {
	return PageTransition(c, key, build, opts...)
}

func sanitizeTransitionScale(v, fallback float32) float32 {
	if !isPositiveFiniteF32(v) || v >= 1 {
		return fallback
	}
	return v
}

func explicitTransitionDir(dir int, fallback float32) float32 {
	if dir < 0 {
		return -1
	}
	if dir > 0 {
		return 1
	}
	return fallback
}

func resolveTransitionDir(explicit int, prev, next any) float32 {
	if explicit < 0 {
		return -1
	}
	if explicit > 0 {
		return 1
	}
	if a, okA := orderedIntKey(prev); okA {
		if b, okB := orderedIntKey(next); okB && b < a {
			return -1
		}
	}
	return 1
}

func orderedIntKey(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		if uint64(n) <= math.MaxInt64 {
			return int64(n), true
		}
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		if n <= math.MaxInt64 {
			return int64(n), true
		}
	}
	return 0, false
}
