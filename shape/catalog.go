package shape

import (
	"math"
)

// Shape is one of the 35 built-in Material 3 Expressive shapes, with the
// name docs/md3e-spec.md gives it.
type Shape struct {
	// Name is the shape's MD3E name, as "Cookie9Sided".
	Name string
	// Polygon builds the shape, normalized into the unit square
	// [0,1]×[0,1] with its aspect ratio kept. Every call returns a new
	// polygon.
	Polygon func() *RoundedPolygon
}

// All returns the 35 built-in shapes in the order docs/md3e-spec.md lists
// them, for catalogs and the gallery to walk.
func All() []Shape {
	return []Shape{
		{"Circle", Circle},
		{"Square", Square},
		{"Slanted", Slanted},
		{"Arch", Arch},
		{"Fan", Fan},
		{"Arrow", Arrow},
		{"SemiCircle", SemiCircle},
		{"Oval", Oval},
		{"Pill", Pill},
		{"Triangle", Triangle},
		{"Diamond", Diamond},
		{"ClamShell", ClamShell},
		{"Pentagon", Pentagon},
		{"Gem", Gem},
		{"Sunny", Sunny},
		{"VerySunny", VerySunny},
		{"Cookie4Sided", Cookie4Sided},
		{"Cookie6Sided", Cookie6Sided},
		{"Cookie7Sided", Cookie7Sided},
		{"Cookie9Sided", Cookie9Sided},
		{"Cookie12Sided", Cookie12Sided},
		{"Ghostish", Ghostish},
		{"Clover4Leaf", Clover4Leaf},
		{"Clover8Leaf", Clover8Leaf},
		{"Burst", Burst},
		{"SoftBurst", SoftBurst},
		{"Boom", Boom},
		{"SoftBoom", SoftBoom},
		{"Flower", Flower},
		{"Puffy", Puffy},
		{"PuffyDiamond", PuffyDiamond},
		{"PixelCircle", PixelCircle},
		{"PixelTriangle", PixelTriangle},
		{"Bun", Bun},
		{"Heart", Heart},
	}
}

// ByName returns the built-in shape called name.
func ByName(name string) (Shape, bool) {
	for _, s := range All() {
		if s.Name == name {
			return s, true
		}
	}
	return Shape{}, false
}

// Circle is a round circle.
func Circle() *RoundedPolygon { return NewCircle(1).Normalized() }

// Square is a square with softly rounded corners.
func Square() *RoundedPolygon {
	return NewRectangle(2, 2, CornerRounding{Radius: 0.34, Smoothing: 0.15}).Normalized()
}

// Slanted is a parallelogram leaning to the right.
func Slanted() *RoundedPolygon {
	return rounded([]Point{{-0.6, -1}, {1, -1}, {0.6, 1}, {-1, 1}}, CornerRounding{Radius: 0.1})
}

// Arch is a doorway: tall sides under a semicircular top, on a flat base.
func Arch() *RoundedPolygon {
	const r = 0.7
	k := 0.5522847498 * r
	return fromCubics(Point{0, 0.2},
		lineCubic(Point{-r, 1}, Point{-r, -0.1}),
		Cubic{Point{-r, -0.1}, Point{-r, -0.1 - k}, Point{-k, -0.1 - r}, Point{0, -0.1 - r}},
		Cubic{Point{0, -0.1 - r}, Point{k, -0.1 - r}, Point{r, -0.1 - k}, Point{r, -0.1}},
		lineCubic(Point{r, -0.1}, Point{r, 1}),
		lineCubic(Point{r, 1}, Point{-r, 1}),
	)
}

// Fan is a broad fan: a flat base under shoulders that rise to a point.
func Fan() *RoundedPolygon {
	return rounded([]Point{{-1, 1}, {-0.75, -0.35}, {0, -1}, {0.75, -0.35}, {1, 1}},
		CornerRounding{Radius: 0.18, Smoothing: 0.35})
}

// Arrow is an arrow pointing right, with a shaft and a barbed head.
func Arrow() *RoundedPolygon {
	return rounded([]Point{
		{-0.8, -0.35}, {0.1, -0.35}, {0.1, -0.8}, {0.95, 0},
		{0.1, 0.8}, {0.1, 0.35}, {-0.8, 0.35},
	}, CornerRounding{Radius: 0.06, Smoothing: 0.1})
}

// SemiCircle is a half disc resting on its flat side.
func SemiCircle() *RoundedPolygon {
	const r = 1.0
	const base = 0.7
	k := 0.5522847498 * r
	return fromCubics(Point{0, 0.2},
		Cubic{Point{-r, base}, Point{-r, base - k}, Point{-k, base - r}, Point{0, base - r}},
		Cubic{Point{0, base - r}, Point{k, base - r}, Point{r, base - k}, Point{r, base}},
		lineCubic(Point{r, base}, Point{-r, base}),
	)
}

// Oval is a wide ellipse.
func Oval() *RoundedPolygon { return NewCircle(1).Transformed(Scale(1, 0.55)).Normalized() }

// Pill is a wide capsule: a rectangle with fully rounded ends.
func Pill() *RoundedPolygon {
	return NewRectangle(2, 0.8, CornerRounding{Radius: 0.4}).Normalized()
}

// Triangle is an equilateral triangle with rounded corners.
func Triangle() *RoundedPolygon {
	return RegularPolygon(3, 1, CornerRounding{Radius: 0.24, Smoothing: 0.2}).Normalized()
}

// Diamond is a square standing on a point, with rounded corners.
func Diamond() *RoundedPolygon {
	return RegularPolygon(4, 1, CornerRounding{Radius: 0.2, Smoothing: 0.2}).Normalized()
}

// ClamShell is a shell: a rounded dome over a scalloped base.
func ClamShell() *RoundedPolygon {
	return rounded([]Point{
		{-1, 0.25}, {-0.9, -0.35}, {-0.5, -0.8}, {0, -0.95},
		{0.5, -0.8}, {0.9, -0.35}, {1, 0.25},
		{0.75, 0.55}, {0.4, 0.45}, {0, 0.75}, {-0.4, 0.45}, {-0.75, 0.55},
	}, CornerRounding{Radius: 0.12, Smoothing: 0.6})
}

// Pentagon is a regular pentagon with rounded corners.
func Pentagon() *RoundedPolygon {
	return RegularPolygon(5, 1, CornerRounding{Radius: 0.22, Smoothing: 0.2}).Normalized()
}

// Gem is a cut gem: a faceted crown over a pointed base.
func Gem() *RoundedPolygon {
	return rounded([]Point{
		{-0.8, -0.5}, {-0.35, -0.9}, {0.35, -0.9}, {0.8, -0.5},
		{0.55, 0.35}, {0, 1}, {-0.55, 0.35},
	}, CornerRounding{Radius: 0.08, Smoothing: 0.2})
}

// Sunny is a sun of eight softly rounded rays.
func Sunny() *RoundedPolygon {
	return Star(8, 1, 0.7,
		CornerRounding{Radius: 0.5, Smoothing: 0.05}, CornerRounding{Radius: 0.22, Smoothing: 1}).Normalized()
}

// VerySunny is a sun of twelve softly rounded rays.
func VerySunny() *RoundedPolygon {
	return Star(12, 1, 0.76,
		CornerRounding{Radius: 0.35, Smoothing: 0.05}, CornerRounding{Radius: 0.18, Smoothing: 1}).Normalized()
}

// Cookie4Sided is a four-sided cookie with softly scalloped edges.
func Cookie4Sided() *RoundedPolygon {
	return Star(4, 1, 0.72, CornerRounding{Radius: 0.4, Smoothing: 0.05}, CornerRounding{Radius: 0.3, Smoothing: 1}).Normalized()
}

// Cookie6Sided is a six-sided cookie with softly scalloped edges.
func Cookie6Sided() *RoundedPolygon {
	return Star(6, 1, 0.76, CornerRounding{Radius: 0.32, Smoothing: 0.05}, CornerRounding{Radius: 0.25, Smoothing: 1}).Normalized()
}

// Cookie7Sided is a seven-sided cookie with softly scalloped edges.
func Cookie7Sided() *RoundedPolygon {
	return Star(7, 1, 0.78, CornerRounding{Radius: 0.3, Smoothing: 0.05}, CornerRounding{Radius: 0.23, Smoothing: 1}).Normalized()
}

// Cookie9Sided is a nine-sided cookie with softly scalloped edges.
func Cookie9Sided() *RoundedPolygon {
	return Star(9, 1, 0.81, CornerRounding{Radius: 0.26, Smoothing: 0.05}, CornerRounding{Radius: 0.2, Smoothing: 1}).Normalized()
}

// Cookie12Sided is a twelve-sided cookie with softly scalloped edges.
func Cookie12Sided() *RoundedPolygon {
	return Star(12, 1, 0.84, CornerRounding{Radius: 0.22, Smoothing: 0.05}, CornerRounding{Radius: 0.17, Smoothing: 1}).Normalized()
}

// Ghostish is a ghost: a rounded hood over a wavy hem.
func Ghostish() *RoundedPolygon {
	return rounded([]Point{
		{-0.8, 0.5}, {-0.85, 0}, {-0.55, -0.7}, {0, -0.9},
		{0.55, -0.7}, {0.85, 0}, {0.8, 0.5},
		{0.45, 0.3}, {0, 0.6}, {-0.45, 0.3},
	}, CornerRounding{Radius: 0.12, Smoothing: 0.7})
}

// Clover4Leaf is a clover of four round leaves.
func Clover4Leaf() *RoundedPolygon {
	return Star(4, 1, 0.5,
		CornerRounding{Radius: 0.6, Smoothing: 0.1}, CornerRounding{Radius: 0.08, Smoothing: 1}).Normalized()
}

// Clover8Leaf is a clover of eight round leaves.
func Clover8Leaf() *RoundedPolygon {
	return Star(8, 1, 0.65,
		CornerRounding{Radius: 0.4, Smoothing: 0.1}, CornerRounding{Radius: 0.06, Smoothing: 1}).Normalized()
}

// Burst is a sharp twelve-pointed burst.
func Burst() *RoundedPolygon {
	return Star(12, 1, 0.55, Unrounded, Unrounded).Normalized()
}

// SoftBurst is a twelve-pointed burst with softened points.
func SoftBurst() *RoundedPolygon {
	return Star(12, 1, 0.62,
		CornerRounding{Radius: 0.15}, CornerRounding{Radius: 0.1, Smoothing: 1}).Normalized()
}

// Boom is a sharp sixteen-pointed burst.
func Boom() *RoundedPolygon {
	return Star(16, 1, 0.7, Unrounded, Unrounded).Normalized()
}

// SoftBoom is a sixteen-pointed burst with softened points.
func SoftBoom() *RoundedPolygon {
	return Star(16, 1, 0.78,
		CornerRounding{Radius: 0.1}, CornerRounding{Radius: 0.08, Smoothing: 1}).Normalized()
}

// Flower is a flower of eight petals.
func Flower() *RoundedPolygon {
	return Star(8, 1, 0.58,
		CornerRounding{Radius: 0.5, Smoothing: 0.05}, CornerRounding{Radius: 0.15, Smoothing: 1}).Normalized()
}

// Puffy is a soft, slightly irregular blob.
func Puffy() *RoundedPolygon {
	radii := []float64{1, 0.94, 1, 0.9, 0.98, 0.93, 1, 0.95, 0.97}
	vertices := make([]Point, len(radii))
	for i, r := range radii {
		angle := -math.Pi/2 + 2*math.Pi*float64(i)/float64(len(radii))
		vertices[i] = Point{r * math.Cos(angle), r * math.Sin(angle)}
	}
	return rounded(vertices, CornerRounding{Radius: 0.4, Smoothing: 1})
}

// PuffyDiamond is a diamond rounded until it is almost a blob.
func PuffyDiamond() *RoundedPolygon {
	return RegularPolygon(4, 1, CornerRounding{Radius: 0.85, Smoothing: 1}).Normalized()
}

// PixelCircle is a circle drawn in stair steps.
func PixelCircle() *RoundedPolygon {
	return NewPolygon([]Point{
		{0, -1}, {0.4, -1}, {0.4, -0.8}, {0.6, -0.8}, {0.6, -0.6}, {0.8, -0.6}, {0.8, -0.4}, {1, -0.4},
		{1, 0}, {1, 0.4}, {0.8, 0.4}, {0.8, 0.6}, {0.6, 0.6}, {0.6, 0.8}, {0.4, 0.8}, {0.4, 1},
		{0, 1}, {-0.4, 1}, {-0.4, 0.8}, {-0.6, 0.8}, {-0.6, 0.6}, {-0.8, 0.6}, {-0.8, 0.4}, {-1, 0.4},
		{-1, 0}, {-1, -0.4}, {-0.8, -0.4}, {-0.8, -0.6}, {-0.6, -0.6}, {-0.6, -0.8}, {-0.4, -0.8}, {-0.4, -1},
	}, nil).Normalized()
}

// PixelTriangle is a triangle drawn in stair steps.
func PixelTriangle() *RoundedPolygon {
	return NewPolygon(pixelate([]Point{{0, -1}, {0.9, 0.7}, {-0.9, 0.7}}, 4), nil).Normalized()
}

// Bun is a bread bun: a domed top over a softly rounded base.
func Bun() *RoundedPolygon {
	return rounded([]Point{
		{-1, 0.3}, {-0.9, -0.4}, {-0.5, -0.85}, {0, -1}, {0.5, -0.85}, {0.9, -0.4}, {1, 0.3},
		{0.7, 0.7}, {0, 0.85}, {-0.7, 0.7},
	}, CornerRounding{Radius: 0.16, Smoothing: 0.5})
}

// Heart is a heart, its lobes over a pointed base.
func Heart() *RoundedPolygon {
	vertices := []Point{
		{0, 1}, {0.45, 0.6}, {0.9, 0.15}, {0.95, -0.3}, {0.65, -0.75}, {0.25, -0.85},
		{0, -0.55},
		{-0.25, -0.85}, {-0.65, -0.75}, {-0.95, -0.3}, {-0.9, 0.15}, {-0.45, 0.6},
	}
	roundings := []CornerRounding{
		{Radius: 0.08, Smoothing: 0.4},
		{Radius: 0.2, Smoothing: 0.9}, {Radius: 0.25, Smoothing: 0.9}, {Radius: 0.2, Smoothing: 0.9},
		{Radius: 0.2, Smoothing: 0.9}, {Radius: 0.15, Smoothing: 0.9},
		{Radius: 0.18, Smoothing: 1},
		{Radius: 0.15, Smoothing: 0.9}, {Radius: 0.2, Smoothing: 0.9}, {Radius: 0.2, Smoothing: 0.9},
		{Radius: 0.25, Smoothing: 0.9}, {Radius: 0.2, Smoothing: 0.9},
	}
	return NewPolygon(vertices, roundings).Normalized()
}

// rounded builds the polygon of vertices with the same rounding on each and
// normalizes it.
func rounded(vertices []Point, rounding CornerRounding) *RoundedPolygon {
	roundings := make([]CornerRounding, len(vertices))
	for i := range roundings {
		roundings[i] = rounding
	}
	return NewPolygon(vertices, roundings).Normalized()
}

// fromCubics builds a polygon from a closed chain of cubics and normalizes
// it.
func fromCubics(center Point, cubics ...Cubic) *RoundedPolygon {
	if len(cubics) < 2 {
		panic("shape: a cubic chain needs at least two cubics")
	}
	for i := range cubics {
		if cubics[i].End().Distance(cubics[(i+1)%len(cubics)].Start()) > 1e-6 {
			panic("shape: the cubic chain is not closed")
		}
	}
	return (&RoundedPolygon{cubics: cubics, center: center}).Normalized()
}

// pixelate returns the vertices of a stair-stepped outline along the edges
// of vertices, steps steps an edge.
func pixelate(vertices []Point, steps int) []Point {
	n := len(vertices)
	var out []Point
	for i := range vertices {
		a, b := vertices[i], vertices[(i+1)%n]
		cur := a
		for s := 1; s <= steps; s++ {
			p := a.Lerp(b, float64(s)/float64(steps))
			if p == cur {
				continue
			}
			out = append(out, Point{X: p.X, Y: cur.Y}, p)
			cur = p
		}
	}
	kept := out[:0]
	for _, p := range out {
		if len(kept) == 0 || kept[len(kept)-1].Distance(p) > epsilon {
			kept = append(kept, p)
		}
	}
	return kept
}
