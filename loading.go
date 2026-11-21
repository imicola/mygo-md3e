package md3e

import (
	"math"
	"slices"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// LoadingStyle selects the look of a LoadingIndicator.
type LoadingStyle int

const (
	// LoadingStandard draws no container: the morphing shape alone, in Primary.
	LoadingStandard LoadingStyle = iota
	// LoadingContained draws a PrimaryContainer circle with the shape inset in
	// OnPrimaryContainer, as the filled MD3E loading indicator.
	LoadingContained
)

// LoadingOptions configures LoadingIndicator. The zero value is a 48 DIP
// standard indicator running the default seven-shape sequence.
type LoadingOptions struct {
	// Style selects the standard or contained look.
	Style LoadingStyle
	// Size is the component's square side in DIP; 48 when zero or invalid.
	Size float32
	// Polygons overrides the shapes the indicator cycles through, in order.
	// Each adjacent pair, including the last back to the first, is morphed.
	// nil or an empty list uses DefaultLoadingPolygons.
	Polygons []*shape.RoundedPolygon
	// Paused freezes the indicator at the phase it shows, asking for no
	// animation frames.
	Paused bool
	// Label names the progress indicator for assistive technology; "Loading"
	// when empty.
	Label string
}

// LoadingIndicator rhythm matching MD3E specifications: a 650ms morph interval
// driven by the theme's Slow Spatial spring plus continuous global rotation
// (360° per 4666ms) and a 90° quarter-turn per step.
const (
	loadingDefaultSize    = 48
	loadingGraphicRatio   = 38.0 / 48.0
	loadingStepDuration   = 650 * time.Millisecond
	loadingGlobalDuration = 4666 * time.Millisecond
	loadingStepDegrees    = 90
)

// defaultLoadingShapes builds the MD3E sequence on first use without init-time
// closure state so binaries that omit LoadingIndicator link none of the 7
// catalog polygons or morph tables.
var (
	loadingDefaultOnce sync.Once
	loadingDefaultList []*shape.RoundedPolygon
)

func defaultLoadingShapes() []*shape.RoundedPolygon {
	loadingDefaultOnce.Do(func() {
		loadingDefaultList = []*shape.RoundedPolygon{
			shape.SoftBurst(),
			shape.Cookie9Sided(),
			shape.Pentagon(),
			shape.Pill(),
			shape.Sunny(),
			shape.Cookie4Sided(),
			shape.Oval(),
		}
	})
	return loadingDefaultList
}

// DefaultLoadingPolygons returns the seven-shape MD3E loading sequence:
// SoftBurst, Cookie9Sided, Pentagon, Pill, Sunny, Cookie4Sided and Oval.
func DefaultLoadingPolygons() []*shape.RoundedPolygon {
	return slices.Clone(defaultLoadingShapes())
}

// loadingStateKey is the Local holding an indicator's prebuilt morphs.
type loadingStateKey struct{}

// loadingClockKey is the Local holding an indicator's phase clock.
type loadingClockKey struct{}

// loadingState holds the polygon sequence and the morph between each adjacent
// pair, including the wrap from the last back to the first. It is rebuilt only
// when the caller swaps the sequence.
type loadingState struct {
	polys  []*shape.RoundedPolygon
	morphs []*shape.Morph
}

// rebuild prebuilds one morph per adjacent shape pair; a single shape morphs
// to itself, which keeps one reusable path for it.
func (s *loadingState) rebuild(polys []*shape.RoundedPolygon) {
	s.polys = slices.Clone(polys)
	if s.morphs == nil {
		s.morphs = make([]*shape.Morph, 0, len(polys))
	}
	s.morphs = s.morphs[:0]
	for i := range polys {
		s.morphs = append(s.morphs, shape.NewMorph(polys[i], polys[(i+1)%len(polys)]))
	}
}

// sameLoadingPolys reports whether two sequences hold geometrically identical
// polygons in the same order.
func sameLoadingPolys(a, b []*shape.RoundedPolygon) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] == b[i] {
			continue
		}
		if a[i] == nil || b[i] == nil {
			return false
		}
		if a[i].Center() != b[i].Center() || !slices.Equal(a[i].Cubics(), b[i].Cubics()) {
			return false
		}
	}
	return true
}

// loadingClock tracks the indicator's start and any pause, so resuming plays
// on from the shape it froze on rather than jumping ahead.
type loadingClock struct {
	start       time.Time
	paused      bool
	pausedAt    time.Time
	offset      time.Duration
	lastElapsed time.Duration
}

// elapsed returns how much animation time has passed at now.
func (c *loadingClock) elapsed(now time.Time) time.Duration {
	end := now
	if c.paused {
		end = c.pausedAt
	}
	d := end.Sub(c.start) - c.offset
	return max(d, 0)
}

// LoadingIndicator builds a Material Design 3 Expressive shape-morphing loading
// indicator: it cycles through a sequence of rounded polygons (the seven MD3E
// shapes by default), morphing each into the next with an eased step and
// turning 45° per step around its center. LoadingStandard paints the shape in
// Primary with no container; LoadingContained paints a PrimaryContainer circle
// with the shape inset in OnPrimaryContainer. Paused freezes it, and ReduceMotion
// (the theme's or the desktop's) shows the first shape still, asking for no
// frames.
//
// It reports assistive technology a progress indicator of unknown length,
// named by Label:
//
//	md3e.LoadingIndicator(c, md3e.LoadingOptions{Style: md3e.LoadingContained, Label: "Loading inbox"})
func LoadingIndicator(c *ui.Context, opts ...LoadingOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: LoadingIndicator takes at most one LoadingOptions")
	}
	var o LoadingOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	size := sanitizePositiveF32(o.Size, loadingDefaultSize)
	polys := o.Polygons
	if len(polys) == 0 {
		polys = defaultLoadingShapes()
	}
	polys = sanitizeLoadingPolys(polys)
	if len(polys) == 0 {
		polys = defaultLoadingShapes()
	}

	th := CurrentTheme(c)
	label := o.Label
	if label == "" {
		label = "Loading"
	}

	el := ui.Box(c).Size(size, size).Shrink(0).DimDisabled(false).Role(ui.RoleProgress).Label(label)
	// The indicator is of unknown length: assistive technology reads no value.
	el.Range(0, 1, -1)
	disabled := el.IsDisabled()

	st := ui.Local(el, loadingStateKey{}, func() loadingState {
		var s loadingState
		s.rebuild(polys)
		return s
	})
	if !sameLoadingPolys(st.polys, polys) {
		st.rebuild(polys)
	}

	clock := ui.Local(el, loadingClockKey{}, func() loadingClock {
		return loadingClock{start: c.Now()}
	})
	paused := o.Paused || disabled
	if paused != clock.paused {
		if paused {
			clock.pausedAt = c.Now()
		} else {
			clock.offset += c.Now().Sub(clock.pausedAt)
			clock.pausedAt = time.Time{}
		}
		clock.paused = paused
	}

	cs := th.Colors
	container := cs.PrimaryContainer
	ink := cs.Primary
	if o.Style == LoadingContained {
		ink = cs.OnPrimaryContainer
	}
	if container.A == 0 {
		container = cs.SurfaceContainerHighest
	}
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		container = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
	}

	morphSpring := motion.Spring(th.Motion, motion.SlowSpatial)

	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 || len(st.morphs) == 0 {
			return
		}
		live := el.Valid()
		reduceMotion := th.ReduceMotion || p.Preferences().ReduceMotion
		elapsed := clock.lastElapsed
		if live {
			elapsed = clock.elapsed(p.Now())
			if reduceMotion {
				elapsed = 0
			}
			clock.lastElapsed = elapsed
		} else if reduceMotion {
			elapsed = 0
		}

		side := min(r.W, r.H)
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		circle := ui.Rect{X: cx - side/2, Y: cy - side/2, W: side, H: side}

		var step int
		var progress, rotation float32
		if len(st.morphs) > 1 && !reduceMotion {
			step = int(elapsed / loadingStepDuration)
			phase := elapsed - time.Duration(step)*loadingStepDuration
			springT := float32(morphSpring.Progress(phase))
			progress = clampF32(springT, 0, 1)
			globalAngle := float32(elapsed) / float32(loadingGlobalDuration) * 360
			rotation = float32(math.Mod(float64(globalAngle+(float32(step)+springT)*loadingStepDegrees), 360))
		}

		gside := side * loadingGraphicRatio
		graphic := ui.Rect{X: cx - gside/2, Y: cy - gside/2, W: gside, H: gside}

		draw := func() {
			if o.Style == LoadingContained && container.A > 0 {
				p.Fill(circle, container, side/2)
			}
			path := st.morphs[step%len(st.morphs)].AsPath(progress, graphic)
			if path == nil {
				return
			}
			p.Transform(ui.Rotation(rotation), cx, cy, func() {
				p.FillPath(path, ink)
			})
		}
		if o.Style == LoadingContained {
			p.Clip(circle, side/2, draw)
		} else {
			p.Clip(r, 0, draw)
		}

		if live && !reduceMotion && !clock.paused && !disabled && len(st.morphs) > 1 {
			p.AnimationFrame()
		}
	})
	return el
}

// sanitizeLoadingPolys drops nil polygons from a caller's sequence, returning
// the slice unchanged when it holds none.
func sanitizeLoadingPolys(polys []*shape.RoundedPolygon) []*shape.RoundedPolygon {
	nilAt := -1
	for i, poly := range polys {
		if poly == nil {
			nilAt = i
			break
		}
	}
	if nilAt < 0 {
		return polys
	}
	kept := make([]*shape.RoundedPolygon, 0, len(polys)-1)
	kept = append(kept, polys[:nilAt]...)
	for _, poly := range polys[nilAt+1:] {
		if poly != nil {
			kept = append(kept, poly)
		}
	}
	return kept
}
