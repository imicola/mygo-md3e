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
// to itself, which keeps one reusable path for it. Each polygon is centered on
// its true centroid and inscribed within r <= 0.5 around (0.5, 0.5) so odd-sided
// shapes such as Pentagon rotate without wobble and custom square shapes never
// clip against a LoadingContained circle at 45°.
func (s *loadingState) rebuild(polys []*shape.RoundedPolygon) {
	s.polys = slices.Clone(polys)
	fitted := make([]*shape.RoundedPolygon, len(polys))
	for i, poly := range polys {
		fitted[i] = centerLoadingPolygon(poly)
	}
	if s.morphs == nil {
		s.morphs = make([]*shape.Morph, 0, len(fitted))
	}
	s.morphs = s.morphs[:0]
	for i := range fitted {
		s.morphs = append(s.morphs, shape.NewMorph(fitted[i], fitted[(i+1)%len(fitted)]))
	}
}

// centerLoadingPolygon moves poly's true centroid to (0.5, 0.5) and uniformly
// scales it down only when any outline point exceeds radius 0.5 from the
// centroid, eliminating rotational wobble and 45° corner clipping while leaving
// already-inscribed symmetric shapes unchanged.
func centerLoadingPolygon(poly *shape.RoundedPolygon) *shape.RoundedPolygon {
	if poly == nil {
		return nil
	}
	cubics := poly.Cubics()
	if len(cubics) == 0 {
		return poly
	}
	c := poly.Center()
	if math.IsNaN(c.X) || math.IsInf(c.X, 0) || math.IsNaN(c.Y) || math.IsInf(c.Y, 0) {
		return poly
	}
	var maxR float64
	for _, cb := range cubics {
		for k := 0; k <= 4; k++ {
			pt := cb.Point(float64(k) * 0.25)
			if d := pt.Distance(c); d > maxR {
				maxR = d
			}
		}
	}
	if !(maxR > 1e-6) {
		return poly
	}
	scale := 1.0
	if maxR > 0.5 {
		scale = 0.5 / maxR
	}
	dx := 0.5 - c.X*scale
	dy := 0.5 - c.Y*scale
	if math.Abs(scale-1) < 1e-4 && math.Abs(dx) < 1e-4 && math.Abs(dy) < 1e-4 {
		return poly
	}
	return poly.Transformed(shape.Scale(scale, scale).Then(shape.Translate(dx, dy)))
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

// loadingClock advances from the last painted phase. After a pause, the
// first visible frame uses that phase again.
type loadingClock struct {
	lastAt      time.Time
	lastElapsed time.Duration
}

func (c *loadingClock) tick(now time.Time, running bool) time.Duration {
	if !running {
		c.lastAt = time.Time{}
		return c.lastElapsed
	}
	if !c.lastAt.IsZero() {
		// MyGo can cull the draw callback completely. Treat a gap of more
		// than 100 ms between requested frames as an interruption.
		if dt := now.Sub(c.lastAt); dt > 0 && dt <= 100*time.Millisecond {
			c.lastElapsed += dt
		}
	}
	c.lastAt = now
	return c.lastElapsed
}

// LoadingIndicator builds a Material Design 3 Expressive shape-morphing loading
// indicator: it cycles through a sequence of rounded polygons (the seven MD3E
// shapes by default), morphing each into the next with an eased step and
// turning 45° per step around its center. LoadingStandard paints the shape in
// Primary with no container; LoadingContained paints a PrimaryContainer circle
// with the shape inset in OnPrimaryContainer. Paused freezes it, and ReduceMotion
// (the theme's or the desktop's) shows the first shape still. The indicator
// pauses while outside the viewport.
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

	st := localPtr(el, loadingStateKey{}, func() *loadingState {
		s := &loadingState{}
		s.rebuild(polys)
		return s
	})
	if !sameLoadingPolys(st.polys, polys) {
		st.rebuild(polys)
	}

	clock := localPtr(el, loadingClockKey{}, func() *loadingClock { return &loadingClock{} })
	paused := o.Paused || disabled
	thReduce := th.ReduceMotion
	contained := o.Style == LoadingContained

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
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) || len(st.morphs) == 0 {
			return
		}
		live := el.Valid()
		visible := rectIntersects(r, p.Visible())
		reduceMotion := thReduce || p.Preferences().ReduceMotion
		running := visible && !paused && !reduceMotion && len(st.morphs) > 1
		elapsed := clock.lastElapsed
		if live {
			elapsed = clock.tick(p.Now(), running)
		}
		if !visible {
			return
		}
		if reduceMotion {
			elapsed = 0
		}

		side := min(r.W, r.H)
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		circle := ui.Rect{X: cx - side/2, Y: cy - side/2, W: side, H: side}

		var step int
		var progress, rotation, breathe float32
		breathe = 1
		if len(st.morphs) > 1 && !reduceMotion {
			step = int(elapsed / loadingStepDuration)
			phase := elapsed - time.Duration(step)*loadingStepDuration
			springT := float32(morphSpring.Progress(phase))
			progress = clampF32(springT, 0, 1)
			breathe = 1 + 0.045*(springT-progress)
			globalAngle := float32(elapsed) / float32(loadingGlobalDuration) * 360
			rotation = float32(math.Mod(float64(globalAngle+(float32(step)+springT)*loadingStepDegrees), 360))
		}

		gside := side * loadingGraphicRatio
		graphic := ui.Rect{X: cx - gside/2, Y: cy - gside/2, W: gside, H: gside}

		draw := func() {
			if contained && container.A > 0 {
				p.Fill(circle, container, side/2)
			}
			path := st.morphs[step%len(st.morphs)].AsPath(progress, graphic)
			if path == nil {
				return
			}
			xform := ui.Rotation(rotation)
			if breathe != 1 {
				xform = ui.Scaling(breathe, breathe).Then(xform)
			}
			p.Transform(xform, cx, cy, func() {
				p.FillPath(path, ink)
			})
		}
		if contained {
			p.Clip(circle, side/2, draw)
		} else {
			p.Clip(r, 0, draw)
		}

		if live && running {
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
