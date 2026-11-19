package shape

import (
	"math"
	"slices"

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
	a := append([]Cubic(nil), from.Normalized().cubics...)
	b := append([]Cubic(nil), to.Normalized().cubics...)
	if chainArea(a) < 0 {
		a = reverseChain(a)
	}
	if chainArea(b) < 0 {
		b = reverseChain(b)
	}
	ca, cb := chainCenter(a), chainCenter(b)
	b = rotateChainToAngle(b, cb, angleAround(ca, a[0].P0))

	aOffsets, aTotal := chainOffsets(a, ca)
	bOffsets, bTotal := chainOffsets(b, cb)
	if aTotal <= epsilon || bTotal <= epsilon {
		panic("shape: a morph needs polygons with an outline around their center")
	}
	cuts := mergeFractions(offsetFractions(aOffsets, aTotal), offsetFractions(bOffsets, bTotal))
	a = splitChainAtFractions(a, ca, aOffsets, aTotal, cuts)
	b = splitChainAtFractions(b, cb, bOffsets, bTotal, cuts)
	for len(a) < len(b) {
		a = subdivideLongest(a)
	}
	for len(b) < len(a) {
		b = subdivideLongest(b)
	}
	return &Morph{a: a, b: b}
}

// AsPath returns the morph at progress, from 0 (from) to 1 (to), in dst. The
// path is the Morph's own, reused and valid until the next call.
func (m *Morph) AsPath(progress float32, dst ui.Rect) *ui.Path {
	t := float64(max(0, min(1, progress)))
	p := &m.path
	p.Reset()
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

// wrapPositive returns a within [0, 2π).
func wrapPositive(a float64) float64 {
	a = math.Mod(a, 2*math.Pi)
	if a < 0 {
		a += 2 * math.Pi
	}
	return a
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

// offsetFractions returns the chain's offsets as fractions of total.
func offsetFractions(offsets []float64, total float64) []float64 {
	out := make([]float64, len(offsets))
	for i, o := range offsets {
		out[i] = o / total
	}
	return out
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

// splitChainAtFractions cuts the chain at each fraction of its total angle,
// so that a chain split at the fractions of two shapes has both their
// vertices.
func splitChainAtFractions(cubics []Cubic, center Point, offsets []float64, total float64, fractions []float64) []Cubic {
	out := make([]Cubic, 0, len(cubics)+len(fractions))
	for i, c := range cubics {
		start, end := offsets[i], offsets[i+1]
		var times []float64
		for _, f := range fractions {
			target := f * total
			if target <= start+1e-9 || target >= end-1e-9 {
				continue
			}
			if t := splitTimeAtOffset(c, center, start, target); t > 1e-9 && t < 1-1e-9 {
				times = append(times, t)
			}
		}
		rest, prev := c, 0.0
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
	return out
}

// splitTimeAtOffset returns the t at which the cubic reaches the given angle
// offset from the chain's start, found by bisection on the curve's angle
// around center.
func splitTimeAtOffset(c Cubic, center Point, startOffset, target float64) float64 {
	startAngle := angleAround(center, c.P0)
	lo, hi := 0.0, 1.0
	for range 50 {
		mid := (lo + hi) / 2
		offset := startOffset + wrapPositive(angleAround(center, c.Point(mid))-startAngle)
		if offset < target {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// subdivideLongest splits the chain's longest cubic in half. It is the
// safety net that keeps two chains the same length if rounding left one
// segment more in a chain.
func subdivideLongest(cubics []Cubic) []Cubic {
	best, bestLength := 0, -1.0
	for i, c := range cubics {
		if l := c.P0.Distance(c.P3); l > bestLength {
			best, bestLength = i, l
		}
	}
	left, right := cubics[best].Split(0.5)
	out := make([]Cubic, 0, len(cubics)+1)
	out = append(out, cubics[:best]...)
	out = append(out, left, right)
	return append(out, cubics[best+1:]...)
}
