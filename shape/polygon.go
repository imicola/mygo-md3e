// Package shape is the Material Design 3 Expressive shape system: a
// RoundedPolygon built from a closed chain of cubic Béziers, the 35 built-in
// MD3E shapes, and morphing between any two of them.
//
// Geometry is float64 in the polygon's own coordinates. The built-in shapes
// are normalized into the unit square [0,1]×[0,1] with their aspect ratio
// kept, and RoundedPolygon.ToPath maps that square onto any ui.Rect.
//
// The coordinates are screen-like, y growing downward, as ui.Rect's are.
// Winding and angles follow the math convention within them, so a positive
// signed area, counter-clockwise here, looks clockwise on screen; keep that
// in mind when comparing against Material's coordinate-free descriptions.
//
// The package depends only on the ui package's Path, Painter and Rect; it
// knows nothing of layout, state or animation.
package shape

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// epsilon is the distance below which points are the same point.
const epsilon = 1e-9

// Point is a position in a shape's coordinates.
type Point struct{ X, Y float64 }

// Add returns p moved by q.
func (p Point) Add(q Point) Point { return Point{p.X + q.X, p.Y + q.Y} }

// Sub returns p moved by -q.
func (p Point) Sub(q Point) Point { return Point{p.X - q.X, p.Y - q.Y} }

// Mul returns p scaled by k.
func (p Point) Mul(k float64) Point { return Point{p.X * k, p.Y * k} }

// Dot returns the dot product of p and q.
func (p Point) Dot(q Point) float64 { return p.X*q.X + p.Y*q.Y }

// Cross returns the 2D cross product of p and q, which is positive when q
// turns counter-clockwise from p.
func (p Point) Cross(q Point) float64 { return p.X*q.Y - p.Y*q.X }

// Len returns p's length.
func (p Point) Len() float64 { return math.Hypot(p.X, p.Y) }

// Distance returns the distance from p to q.
func (p Point) Distance(q Point) float64 { return p.Sub(q).Len() }

// Normalized returns p scaled to a length of one: the zero point for p at
// the origin.
func (p Point) Normalized() Point {
	l := p.Len()
	if l == 0 {
		return Point{}
	}
	return Point{p.X / l, p.Y / l}
}

// Lerp returns the point t of the way from p to q.
func (p Point) Lerp(q Point, t float64) Point {
	return Point{p.X + (q.X-p.X)*t, p.Y + (q.Y-p.Y)*t}
}

// Box is an axis-aligned rectangle in a shape's coordinates.
type Box struct{ MinX, MinY, MaxX, MaxY float64 }

// Width returns the box's width.
func (b Box) Width() float64 { return b.MaxX - b.MinX }

// Height returns the box's height.
func (b Box) Height() float64 { return b.MaxY - b.MinY }

// Center returns the box's center.
func (b Box) Center() Point {
	return Point{(b.MinX + b.MaxX) / 2, (b.MinY + b.MaxY) / 2}
}

// Union returns the smallest box holding b and o.
func (b Box) Union(o Box) Box {
	return Box{
		MinX: math.Min(b.MinX, o.MinX),
		MinY: math.Min(b.MinY, o.MinY),
		MaxX: math.Max(b.MaxX, o.MaxX),
		MaxY: math.Max(b.MaxY, o.MaxY),
	}
}

// CornerRounding rounds a polygon corner. Radius is the radius of the
// circular corner that a rounded sharp angle would take, in the polygon's
// units; zero leaves the corner sharp. Smoothing, from 0 to 1, blends the
// circular arc toward a fuller corner whose curve meets the edges at the
// vertex itself, as the superellipse-like corners of expressive shapes do.
type CornerRounding struct {
	Radius    float64
	Smoothing float64
}

// Unrounded is the sharp corner.
var Unrounded = CornerRounding{}

// Transform is a 2D affine transform, applied as
//
//	x' = A*x + C*y + E
//	y' = B*x + D*y + F
type Transform struct {
	A, B, C, D, E, F float64
}

// Translate returns the transform that moves points by (dx, dy).
func Translate(dx, dy float64) Transform {
	return Transform{A: 1, D: 1, E: dx, F: dy}
}

// Scale returns the transform that scales x by sx and y by sy.
func Scale(sx, sy float64) Transform {
	return Transform{A: sx, D: sy}
}

// Rotate returns the transform that turns points counter-clockwise by
// degrees in the polygon's coordinates, as screen coordinates turn them
// clockwise.
func Rotate(degrees float64) Transform {
	radians := degrees * math.Pi / 180
	sin, cos := math.Sincos(radians)
	return Transform{A: cos, B: sin, C: -sin, D: cos}
}

// Apply returns the point t moves p to.
func (t Transform) Apply(p Point) Point {
	return Point{t.A*p.X + t.C*p.Y + t.E, t.B*p.X + t.D*p.Y + t.F}
}

// Then returns the transform that applies t and then u.
func (t Transform) Then(u Transform) Transform {
	return Transform{
		A: u.A*t.A + u.C*t.B,
		B: u.B*t.A + u.D*t.B,
		C: u.A*t.C + u.C*t.D,
		D: u.B*t.C + u.D*t.D,
		E: u.A*t.E + u.C*t.F + u.E,
		F: u.B*t.E + u.D*t.F + u.F,
	}
}

// Cubic is one cubic Bézier segment of a polygon's outline, from P0 to P3
// with control points P1 and P2.
type Cubic struct{ P0, P1, P2, P3 Point }

// Start returns the cubic's first point.
func (c Cubic) Start() Point { return c.P0 }

// End returns the cubic's last point.
func (c Cubic) End() Point { return c.P3 }

// Point returns the point of the cubic at t, from 0 to 1.
func (c Cubic) Point(t float64) Point {
	u := 1 - t
	a, b, cc, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
	return Point{
		a*c.P0.X + b*c.P1.X + cc*c.P2.X + d*c.P3.X,
		a*c.P0.Y + b*c.P1.Y + cc*c.P2.Y + d*c.P3.Y,
	}
}

// Derivative returns the cubic's tangent at t, not normalized.
func (c Cubic) Derivative(t float64) Point {
	u := 1 - t
	a, b, cc := 3*u*u, 6*u*t, 3*t*t
	return Point{
		a*(c.P1.X-c.P0.X) + b*(c.P2.X-c.P1.X) + cc*(c.P3.X-c.P2.X),
		a*(c.P1.Y-c.P0.Y) + b*(c.P2.Y-c.P1.Y) + cc*(c.P3.Y-c.P2.Y),
	}
}

// Reversed returns the cubic drawn the other way round.
func (c Cubic) Reversed() Cubic {
	return Cubic{P0: c.P3, P1: c.P2, P2: c.P1, P3: c.P0}
}

// Split returns the two cubics that together draw c, cut at t.
func (c Cubic) Split(t float64) (Cubic, Cubic) {
	a := c.P0.Lerp(c.P1, t)
	b := c.P1.Lerp(c.P2, t)
	d := c.P2.Lerp(c.P3, t)
	ab := a.Lerp(b, t)
	bd := b.Lerp(d, t)
	m := ab.Lerp(bd, t)
	return Cubic{c.P0, a, ab, m}, Cubic{m, bd, d, c.P3}
}

// Bounds returns the smallest box holding the cubic's curve, its extrema
// included.
func (c Cubic) Bounds() Box {
	b := Box{
		MinX: math.Min(c.P0.X, c.P3.X), MinY: math.Min(c.P0.Y, c.P3.Y),
		MaxX: math.Max(c.P0.X, c.P3.X), MaxY: math.Max(c.P0.Y, c.P3.Y),
	}
	for _, t := range cubicExtrema(c.P0.X, c.P1.X, c.P2.X, c.P3.X) {
		p := c.Point(t)
		b.MinX, b.MaxX = math.Min(b.MinX, p.X), math.Max(b.MaxX, p.X)
	}
	for _, t := range cubicExtrema(c.P0.Y, c.P1.Y, c.P2.Y, c.P3.Y) {
		p := c.Point(t)
		b.MinY, b.MaxY = math.Min(b.MinY, p.Y), math.Max(b.MaxY, p.Y)
	}
	return b
}

// cubicExtrema returns the times within (0, 1) at which the cubic coordinate
// with the four values is at a minimum or maximum.
func cubicExtrema(p0, p1, p2, p3 float64) []float64 {
	a := -p0 + 3*p1 - 3*p2 + p3
	b := 2 * (p0 - 2*p1 + p2)
	c := p1 - p0
	var times []float64
	if math.Abs(a) < 1e-12 {
		if math.Abs(b) > 1e-12 {
			times = append(times, -c/b)
		}
	} else {
		disc := b*b - 4*a*c
		if disc >= 0 {
			root := math.Sqrt(disc)
			times = append(times, (-b+root)/(2*a), (-b-root)/(2*a))
		}
	}
	kept := times[:0]
	for _, t := range times {
		if t > 0 && t < 1 {
			kept = append(kept, t)
		}
	}
	return kept
}

// RoundedPolygon is a closed outline of cubic Béziers, each starting where
// the one before it ended, with the last returning to the first. Polygons
// are immutable once built.
type RoundedPolygon struct {
	cubics []Cubic
	center Point
}

// Cubics returns the polygon's closed chain of cubics. The slice is the
// polygon's own and must not be modified.
func (p *RoundedPolygon) Cubics() []Cubic { return p.cubics }

// Center returns the point the polygon was built around: the centroid of its
// vertices, which shape morphing rotates and aligns features around.
func (p *RoundedPolygon) Center() Point { return p.center }

// Bounds returns the smallest box holding the polygon.
func (p *RoundedPolygon) Bounds() Box {
	b := p.cubics[0].Bounds()
	for _, c := range p.cubics[1:] {
		b = b.Union(c.Bounds())
	}
	return b
}

// Transformed returns the polygon moved by t.
func (p *RoundedPolygon) Transformed(t Transform) *RoundedPolygon {
	cubics := make([]Cubic, len(p.cubics))
	for i, c := range p.cubics {
		cubics[i] = Cubic{t.Apply(c.P0), t.Apply(c.P1), t.Apply(c.P2), t.Apply(c.P3)}
	}
	return &RoundedPolygon{cubics: cubics, center: t.Apply(p.center)}
}

// Normalized returns the polygon scaled uniformly and moved so that its
// bounds fit exactly across the unit square [0,1]×[0,1], centered, with its
// aspect ratio kept: a wide shape keeps its shape and leaves the square's
// height partly empty.
func (p *RoundedPolygon) Normalized() *RoundedPolygon {
	b := p.Bounds()
	size := math.Max(b.Width(), b.Height())
	if size <= epsilon {
		panic("shape: a polygon without extent cannot be normalized")
	}
	scale := 1 / size
	c := b.Center()
	t := Scale(scale, scale).Then(Translate(0.5-c.X*scale, 0.5-c.Y*scale))
	return p.Transformed(t)
}

// ToPath converts the polygon to a ui.Path, mapping the unit square [0,1]×
// [0,1] that Normalized produces onto r. It suits normalized polygons; on
// others, call Normalized first.
func (p *RoundedPolygon) ToPath(r ui.Rect) *ui.Path {
	path := &ui.Path{}
	first := p.cubics[0].P0
	path.MoveTo(r.X+float32(first.X)*r.W, r.Y+float32(first.Y)*r.H)
	for _, c := range p.cubics {
		path.CubeTo(
			r.X+float32(c.P1.X)*r.W, r.Y+float32(c.P1.Y)*r.H,
			r.X+float32(c.P2.X)*r.W, r.Y+float32(c.P2.Y)*r.H,
			r.X+float32(c.P3.X)*r.W, r.Y+float32(c.P3.Y)*r.H,
		)
	}
	return path.Close()
}

// NewPolygon builds a rounded polygon from vertices in order and the
// rounding of each vertex, as roundings[i] rounds vertices[i]. A nil
// roundings rounds none. A clockwise list is reversed to the
// counter-clockwise winding every MD3E shape uses. It panics for fewer than
// three vertices or roundings of another length.
func NewPolygon(vertices []Point, roundings []CornerRounding) *RoundedPolygon {
	if len(vertices) < 3 {
		panic("shape: a polygon needs at least three vertices")
	}
	if roundings == nil {
		roundings = make([]CornerRounding, len(vertices))
	}
	if len(roundings) != len(vertices) {
		panic("shape: one rounding per vertex is required")
	}
	if signedArea(vertices) < 0 {
		vertices = reversedPoints(vertices)
		roundings = reversedRoundings(roundings)
	}
	n := len(vertices)
	corners := make([]corner, n)
	for i := range vertices {
		prev := vertices[(i+n-1)%n]
		next := vertices[(i+1)%n]
		corners[i] = roundCorner(prev, vertices[i], next, roundings[i])
	}
	var cubics []Cubic
	for i := range corners {
		c := corners[i]
		if !c.sharp {
			cubics = append(cubics, Cubic{c.start, c.c1, c.c2, c.end})
		}
		nextStart := corners[(i+1)%n].start
		if c.end.Distance(nextStart) > epsilon {
			cubics = append(cubics, lineCubic(c.end, nextStart))
		}
	}
	if len(cubics) == 0 {
		panic("shape: the polygon has no outline")
	}
	return &RoundedPolygon{cubics: cubics, center: polygonCentroid(vertices)}
}

// RegularPolygon returns a regular polygon of vertices sides around the
// origin, with a vertex at the top and the given rounding on every corner.
func RegularPolygon(vertices int, radius float64, rounding CornerRounding) *RoundedPolygon {
	if vertices < 3 {
		panic("shape: a regular polygon needs at least three vertices")
	}
	if radius <= 0 {
		panic("shape: a polygon's radius must be positive")
	}
	points := make([]Point, vertices)
	roundings := make([]CornerRounding, vertices)
	for i := range points {
		angle := -math.Pi/2 + 2*math.Pi*float64(i)/float64(vertices)
		points[i] = Point{radius * math.Cos(angle), radius * math.Sin(angle)}
		roundings[i] = rounding
	}
	return NewPolygon(points, roundings)
}

// Star returns a star of points outer points around the origin, with outer
// vertices at radius rounded by rounding and the valleys between them at
// innerRadius rounded by innerRounding.
func Star(points int, radius, innerRadius float64, rounding, innerRounding CornerRounding) *RoundedPolygon {
	if points < 2 {
		panic("shape: a star needs at least two points")
	}
	if radius <= 0 || innerRadius <= 0 || innerRadius >= radius {
		panic("shape: a star's inner radius must be within its outer radius")
	}
	vertices := make([]Point, 2*points)
	roundings := make([]CornerRounding, 2*points)
	for i := range vertices {
		angle := -math.Pi/2 + math.Pi*float64(i)/float64(points)
		r := radius
		roundings[i] = rounding
		if i%2 == 1 {
			r = innerRadius
			roundings[i] = innerRounding
		}
		vertices[i] = Point{r * math.Cos(angle), r * math.Sin(angle)}
	}
	return NewPolygon(vertices, roundings)
}

// NewRectangle returns a rectangle of the given size centered at the
// origin, its corners rounded by rounding.
func NewRectangle(width, height float64, rounding CornerRounding) *RoundedPolygon {
	if width <= 0 || height <= 0 {
		panic("shape: a rectangle's size must be positive")
	}
	hw, hh := width/2, height/2
	vertices := []Point{{-hw, -hh}, {hw, -hh}, {hw, hh}, {-hw, hh}}
	roundings := []CornerRounding{rounding, rounding, rounding, rounding}
	return NewPolygon(vertices, roundings)
}

// NewCircle returns a circle of radius centered at the origin, in four
// cubics as ui.Path.Circle draws it.
func NewCircle(radius float64) *RoundedPolygon {
	if radius <= 0 {
		panic("shape: a circle's radius must be positive")
	}
	const kappa = 0.5522847498
	r, k := radius, radius*kappa
	return &RoundedPolygon{
		cubics: []Cubic{
			{Point{r, 0}, Point{r, k}, Point{k, r}, Point{0, r}},
			{Point{0, r}, Point{-k, r}, Point{-r, k}, Point{-r, 0}},
			{Point{-r, 0}, Point{-r, -k}, Point{-k, -r}, Point{0, -r}},
			{Point{0, -r}, Point{k, -r}, Point{r, -k}, Point{r, 0}},
		},
		center: Point{},
	}
}

// corner is one vertex's rounded corner: the cubic from start to end, with
// its control points, or a sharp corner where start and end are the vertex.
type corner struct {
	start, end Point
	c1, c2     Point
	sharp      bool
}

// roundCorner cuts the corner at curr, between prev and next, as rounding
// says: a circular fillet of the corner's radius, fuller toward the vertex
// as smoothing rises.
func roundCorner(prev, curr, next Point, rounding CornerRounding) corner {
	sharp := corner{start: curr, end: curr, c1: curr, c2: curr, sharp: true}
	prevLen, nextLen := prev.Distance(curr), next.Distance(curr)
	if rounding.Radius <= 0 || prevLen <= epsilon || nextLen <= epsilon {
		return sharp
	}
	u0 := prev.Sub(curr).Normalized() // toward the previous vertex
	u1 := next.Sub(curr).Normalized() // toward the next vertex
	beta := math.Acos(math.Max(-1, math.Min(1, u0.Dot(u1))))
	if beta < 1e-6 || math.Pi-beta < 1e-6 {
		return sharp
	}
	// The fillet touches each edge this far from the vertex, at most half of
	// the shorter edge so that neighbouring corners never overlap.
	t := rounding.Radius / math.Tan(beta/2)
	t = math.Min(t, math.Min(prevLen, nextLen)/2)
	if t <= epsilon {
		return sharp
	}
	start := curr.Add(u0.Mul(t))
	end := curr.Add(u1.Mul(t))
	// The arc between the tangent points sweeps π−β; its radius follows the
	// cut, which may have been clamped.
	radius := t * math.Tan(beta/2)
	k := 4.0 / 3.0 * math.Tan((math.Pi-beta)/4) * radius
	// Smoothing moves the controls toward the vertex itself, which fills the
	// corner out while keeping the tangents along the edges.
	smoothing := math.Max(0, math.Min(1, rounding.Smoothing))
	k = k*(1-smoothing) + t*smoothing
	// The outline runs prev → start → end → next, so it leaves start along
	// -u0 and reaches end along u1 whichever way the corner bends: both
	// controls sit between their tangent point and the vertex, toward it.
	// A convex corner bows toward the vertex and a reflex one wraps the
	// valley around it, with the tangents continuous at both joins.
	return corner{
		start: start,
		end:   end,
		c1:    start.Sub(u0.Mul(k)),
		c2:    end.Sub(u1.Mul(k)),
	}
}

// lineCubic returns the cubic that draws the straight line from a to b.
func lineCubic(a, b Point) Cubic {
	return Cubic{a, a.Lerp(b, 1.0/3), a.Lerp(b, 2.0/3), b}
}

// signedArea returns twice the signed area of the polygon through vertices,
// positive when they wind counter-clockwise.
func signedArea(vertices []Point) float64 {
	var sum float64
	for i, a := range vertices {
		sum += a.Cross(vertices[(i+1)%len(vertices)])
	}
	return sum
}

// polygonCentroid returns the area centroid of the polygon through
// vertices, or their average for a polygon without area.
func polygonCentroid(vertices []Point) Point {
	var area, x, y float64
	for i, a := range vertices {
		b := vertices[(i+1)%len(vertices)]
		cross := a.Cross(b)
		area += cross
		x += (a.X + b.X) * cross
		y += (a.Y + b.Y) * cross
	}
	if math.Abs(area) <= epsilon {
		var sum Point
		for _, v := range vertices {
			sum = sum.Add(v)
		}
		return sum.Mul(1 / float64(len(vertices)))
	}
	f := 1 / (3 * area)
	return Point{x * f, y * f}
}

// reversedPoints returns vertices in the other order.
func reversedPoints(vertices []Point) []Point {
	out := make([]Point, len(vertices))
	for i, v := range vertices {
		out[len(vertices)-1-i] = v
	}
	return out
}

// reversedRoundings returns roundings in the other order.
func reversedRoundings(roundings []CornerRounding) []CornerRounding {
	out := make([]CornerRounding, len(roundings))
	for i, r := range roundings {
		out[len(roundings)-1-i] = r
	}
	return out
}
