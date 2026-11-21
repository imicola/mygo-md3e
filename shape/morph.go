package shape

import (
	"math"
	"slices"
	"sort"

	"github.com/egoist/mygo/ui"
)

// Morph morphs one closed polygon into another. Both outlines are cut where
// the other has a vertex, at matching angles around each shape's center, so
// the corners and features of the two shapes line up; each pair of cubics is
// then interpolated. At progress 0 the morph draws from exactly and at 1 to
// exactly.
//
// A Morph is built once and reused frame after frame: AsPath reuses its path
// rather than allocating one per frame. It is not safe for concurrent use.
type Morph struct {
	a, b []Cubic
	path ui.Path
}

// NewMorph returns the morph from one polygon to another. Both polygons are
// normalized into the unit square first, so shapes of different sizes and
// aspect ratios meet in a common space; the one that winds clockwise is
// reversed, and the second is turned to start near the first's start so that
// the morph does not twist.
func NewMorph(from, to *RoundedPolygon) *Morph {
	if from == nil || to == nil || len(from.cubics) < 2 || len(to.cubics) < 2 {
		panic("shape: a morph needs two polygons")
	}
	a := from.Normalized().cubics
	b := to.Normalized().cubics
	if chainArea(a) < 0 {
		a = reverseChain(a)
	}
	if chainArea(b) < 0 {
		b = reverseChain(b)
	}
	ca, cb := chainCenter(a), chainCenter(b)
	b = rotateChainToAngle(b, cb, angleAround(ca, a[0].P0))

	chainA := newMorphChain(a, ca)
	chainB := newMorphChain(b, cb)
	if chainA.total <= epsilon || chainB.total <= epsilon {
		panic("shape: a morph needs polygons with an outline around their center")
	}
	cuts := mergeFractions(chainA.starts, chainB.starts)
	chainA.splitAtFractions(cuts)
	chainB.splitAtFractions(cuts)
	// A chain whose angle repeats has a piece the other has no vertex for.
	// Cut the other one near that angle too, so that the pieces after it
	// stay paired instead of shifting by one.
	for len(chainA.cubics) != len(chainB.cubics) {
		if len(chainA.cubics) > len(chainB.cubics) {
			chainB.splitAt(missingFraction(chainA.starts, chainB.starts))
		} else {
			chainA.splitAt(missingFraction(chainB.starts, chainA.starts))
		}
	}
	return &Morph{a: chainA.cubics, b: chainB.cubics}
}

// morphChain is one polygon's outline as a morph cuts it: its cubics, the
// fraction of its total turn at which each starts, and the angles it measures.
type morphChain struct {
	cubics  []Cubic
	starts  []float64
	center  Point
	offsets []float64
	total   float64
}

// newMorphChain measures the angle at which each of the chain's cubics starts.
func newMorphChain(cubics []Cubic, center Point) *morphChain {
	c := &morphChain{cubics: cubics, center: center}
	c.measure()
	return c
}

// measure recomputes the chain's offsets and start fractions from its cubics.
func (c *morphChain) measure() {
	c.offsets, c.total = chainOffsets(c.cubics, c.center)
	c.starts = make([]float64, len(c.cubics))
	if c.total <= epsilon {
		return
	}
	for i := range c.cubics {
		c.starts[i] = c.offsets[i] / c.total
	}
}

// splitAtFractions cuts the chain at each fraction of its total turn, so that
// two chains cut at the fractions of both have each other's vertices.
func (c *morphChain) splitAtFractions(fractions []float64) {
	out := make([]Cubic, 0, len(c.cubics)+len(fractions))
	for i, cubic := range c.cubics {
		start, end := c.offsets[i], c.offsets[i+1]
		var times []float64
		for _, f := range fractions {
			target := f * c.total
			if target <= start+1e-9 || target >= end-1e-9 {
				continue
			}
			if t := splitTimeAtOffset(cubic, c.center, start, target); t > 1e-9 && t < 1-1e-9 {
				times = append(times, t)
			}
		}
		slices.Sort(times)
		rest, prev := cubic, 0.0
		for _, t := range times {
			u := (t - prev) / (1 - prev)
			if u <= 1e-9 || u >= 1-1e-9 {
				continue
			}
			left, right := rest.Split(u)
			out = append(out, left)
			rest, prev = right, t
		}
		out = append(out, rest)
	}
	c.cubics = out
	c.measure()
}

// splitAt adds one piece to the chain, cutting the piece at fraction, or in
// its middle when the fraction falls on a vertex or past it, which keeps the
// outline exactly.
func (c *morphChain) splitAt(fraction float64) {
	n := len(c.cubics)
	if n == 0 {
		return
	}
	i := sort.Search(len(c.starts), func(k int) bool { return c.starts[k] > fraction }) - 1
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	for i+1 < n && c.starts[i+1] == c.starts[i] {
		i++ // a piece without angular width cannot be cut again
	}
	endFraction := 1.0
	if i+1 < len(c.starts) {
		endFraction = c.starts[i+1]
	}
	t := 0.5
	if fraction > c.starts[i]+1e-9 && fraction < endFraction-1e-9 {
		if tt := splitTimeAtOffset(c.cubics[i], c.center, c.offsets[i], fraction*c.total); tt > 1e-9 && tt < 1-1e-9 {
			t = tt
		}
	}
	left, right := c.cubics[i].Split(t)
	out := make([]Cubic, 0, n+1)
	out = append(out, c.cubics[:i]...)
	out = append(out, left, right)
	out = append(out, c.cubics[i+1:]...)
	c.cubics = out
	c.measure()
}

// missingFraction returns a fraction at which longer starts but shorter has no
// vertex: the angle of a piece the shorter chain has no counterpart for.
func missingFraction(longer, shorter []float64) float64 {
	const tolerance = 1e-9
	j := 0
	for _, f := range longer {
		for j < len(shorter) && shorter[j] < f-tolerance {
			j++
		}
		if j < len(shorter) && math.Abs(shorter[j]-f) <= tolerance {
			j++
			continue
		}
		return f
	}
	return 0.5
}

func sanitizeMorphProgress(progress float32) float64 {
	f := float64(progress)
	switch {
	case math.IsNaN(f), math.IsInf(f, -1), f <= 0:
		return 0
	case math.IsInf(f, 1), f >= 1:
		return 1
	default:
		return f
	}
}

// AsPath returns the morph at progress, from 0 (from) to 1 (to), in dst. The
// path is the Morph's own, reused and valid until the next call. Nil morphs,
// non-finite progress values, and empty/non-finite rects degrade safely.
func (m *Morph) AsPath(progress float32, dst ui.Rect) *ui.Path {
	if m == nil || len(m.a) == 0 || len(m.b) != len(m.a) {
		return &ui.Path{}
	}
	p := &m.path
	p.Reset()
	if !finiteRect(dst) || dst.W <= 0 || dst.H <= 0 {
		return p
	}
	t := sanitizeMorphProgress(progress)
	first := dstPoint(m.a[0].P0.Lerp(m.b[0].P0, t), dst)
	p.MoveTo(first.X, first.Y)
	for i := range m.a {
		a, b := m.a[i], m.b[i]
		c1 := dstPoint(a.P1.Lerp(b.P1, t), dst)
		c2 := dstPoint(a.P2.Lerp(b.P2, t), dst)
		end := dstPoint(a.P3.Lerp(b.P3, t), dst)
		p.CubeTo(c1.X, c1.Y, c2.X, c2.Y, end.X, end.Y)
	}
	return p.Close()
}

// CubicsAt returns the morph's whole cubic chain at progress, from 0 (from) to
// 1 (to), for callers that measure or draw the outline themselves; AsPath
// reuses a path instead. The slice is newly allocated.
func (m *Morph) CubicsAt(progress float32) []Cubic {
	if m == nil || len(m.a) == 0 || len(m.b) != len(m.a) {
		return nil
	}
	t := sanitizeMorphProgress(progress)
	out := make([]Cubic, len(m.a))
	for i := range m.a {
		out[i] = m.cubicAt(i, t)
	}
	return out
}

// cubicAt returns the i-th cubic of the morph at t, for tests and debug
// drawing.
func (m *Morph) cubicAt(i int, t float64) Cubic {
	a, b := m.a[i], m.b[i]
	return Cubic{
		P0: a.P0.Lerp(b.P0, t),
		P1: a.P1.Lerp(b.P1, t),
		P2: a.P2.Lerp(b.P2, t),
		P3: a.P3.Lerp(b.P3, t),
	}
}

// dstPoint maps a point of the unit square into dst.
func dstPoint(p Point, dst ui.Rect) ui.Point {
	return ui.Point{X: dst.X + float32(p.X)*dst.W, Y: dst.Y + float32(p.Y)*dst.H}
}

// angleAround returns the angle of p around center, in radians.
func angleAround(center, p Point) float64 {
	return math.Atan2(p.Y-center.Y, p.X-center.X)
}

// wrapSigned returns a within (-π, π].
func wrapSigned(a float64) float64 {
	a = math.Mod(a+math.Pi, 2*math.Pi)
	if a < 0 {
		a += 2 * math.Pi
	}
	return a - math.Pi
}

// chainArea returns twice the signed area of the anchors of the cubic chain,
// positive when it winds counter-clockwise.
func chainArea(cubics []Cubic) float64 {
	var sum float64
	for i, c := range cubics {
		sum += c.P0.Cross(cubics[(i+1)%len(cubics)].P0)
	}
	return sum
}

// chainCenter returns the area centroid of the cubic chain's anchors.
func chainCenter(cubics []Cubic) Point {
	anchors := make([]Point, len(cubics))
	for i, c := range cubics {
		anchors[i] = c.P0
	}
	return polygonCentroid(anchors)
}

// reverseChain returns the chain drawn the other way round.
func reverseChain(cubics []Cubic) []Cubic {
	out := make([]Cubic, len(cubics))
	for i, c := range cubics {
		out[len(cubics)-1-i] = c.Reversed()
	}
	return out
}

// rotateChainToAngle returns the chain turned to start at its vertex whose
// angle around center is nearest target, so two chains start their traversal
// at the same place.
func rotateChainToAngle(cubics []Cubic, center Point, target float64) []Cubic {
	best, bestDistance := 0, math.Inf(1)
	for i, c := range cubics {
		if d := math.Abs(wrapSigned(angleAround(center, c.P0) - target)); d < bestDistance {
			best, bestDistance = i, d
		}
	}
	if best == 0 {
		return cubics
	}
	out := make([]Cubic, 0, len(cubics))
	out = append(out, cubics[best:]...)
	return append(out, cubics[:best]...)
}

// chainOffsets returns the unwrapped angle at which each cubic of the chain
// starts around center, measured from the chain's first point, and the total
// angle the chain turns through. Offsets grow monotonically for a chain that
// winds counter-clockwise around center.
func chainOffsets(cubics []Cubic, center Point) ([]float64, float64) {
	offsets := make([]float64, len(cubics)+1)
	prev := angleAround(center, cubics[0].P0)
	for i, c := range cubics {
		end := angleAround(center, c.P3)
		step := wrapSigned(end - prev)
		if step < 0 {
			// A back step, or a segment through the center: keep the chain's
			// angles monotonic.
			step = 0
		}
		offsets[i+1] = offsets[i] + step
		prev = end
	}
	return offsets, offsets[len(cubics)]
}

// mergeFractions returns the interior fractions of both lists, sorted and
// deduplicated: the angles at which both chains are cut.
func mergeFractions(lists ...[]float64) []float64 {
	var all []float64
	for _, list := range lists {
		for _, f := range list {
			if f > 1e-9 && f < 1-1e-9 {
				all = append(all, f)
			}
		}
	}
	slices.Sort(all)
	kept := all[:0]
	for _, f := range all {
		if len(kept) == 0 || f-kept[len(kept)-1] > 1e-9 {
			kept = append(kept, f)
		}
	}
	return kept
}

// splitTimeAtOffset returns the t at which the cubic reaches the given angle
// offset from the chain's start, found by bisection on the curve's angle
// around center. The angle is unwrapped signed, so a small backward wobble in
// a concave segment cannot jump by a whole turn and send the bisection the
// wrong way; as chainOffsets does, the cut only moves forward.
func splitTimeAtOffset(c Cubic, center Point, startOffset, target float64) float64 {
	startAngle := angleAround(center, c.P0)
	offsetAt := func(t float64) float64 {
		return startOffset + math.Max(0, wrapSigned(angleAround(center, c.Point(t))-startAngle))
	}
	lo, hi := 0.0, 1.0
	for range 50 {
		mid := (lo + hi) / 2
		if offsetAt(mid) < target {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}
