package shape_test

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/shape"
)

// polygons returns a spread of polygons the tests exercise.
func polygons() map[string]*shape.RoundedPolygon {
	return map[string]*shape.RoundedPolygon{
		"triangle":  shape.RegularPolygon(3, 100, shape.CornerRounding{Radius: 15}),
		"pentagon":  shape.RegularPolygon(5, 100, shape.CornerRounding{Radius: 20, Smoothing: 0.5}),
		"star":      shape.Star(8, 100, 45, shape.CornerRounding{Radius: 8}, shape.CornerRounding{Radius: 12, Smoothing: 1}),
		"rectangle": shape.NewRectangle(200, 120, shape.CornerRounding{Radius: 30}),
		"circle":    shape.NewCircle(100),
		"sharp":     shape.RegularPolygon(3, 100, shape.Unrounded),
	}
}

func TestClosedChain(t *testing.T) {
	for name, poly := range polygons() {
		t.Run(name, func(t *testing.T) {
			cubics := poly.Cubics()
			if len(cubics) < 3 {
				t.Fatalf("%d cubics, want at least 3", len(cubics))
			}
			for i, c := range cubics {
				if !finiteCubic(c) {
					t.Fatalf("cubic %d has a non-finite point: %+v", i, c)
				}
				next := cubics[(i+1)%len(cubics)]
				if d := c.End().Distance(next.Start()); d > 1e-6 {
					t.Errorf("cubic %d ends %v from cubic %d's start", i, d, (i+1)%len(cubics))
				}
			}
		})
	}
}

func TestNormalizedFitsUnitSquare(t *testing.T) {
	for name, poly := range polygons() {
		t.Run(name, func(t *testing.T) {
			n := poly.Normalized()
			b := n.Bounds()
			if b.MinX < -1e-6 || b.MinY < -1e-6 || b.MaxX > 1+1e-6 || b.MaxY > 1+1e-6 {
				t.Errorf("bounds %+v are outside the unit square", b)
			}
			if w, h := b.Width(), b.Height(); math.Abs(math.Max(w, h)-1) > 1e-6 {
				t.Errorf("bounds %v×%v do not fill one side of the unit square", w, h)
			}
			if c := b.Center(); math.Abs(c.X-0.5) > 1e-6 || math.Abs(c.Y-0.5) > 1e-6 {
				t.Errorf("bounds center %+v is not the square's center", c)
			}
		})
	}
}

func TestPathContains(t *testing.T) {
	for name, poly := range polygons() {
		t.Run(name, func(t *testing.T) {
			path := poly.Normalized().ToPath(ui.Rect{X: 10, Y: 20, W: 200, H: 200})
			if !path.Contains(110, 120) {
				t.Error("the path does not contain its center")
			}
			if path.Contains(12, 22) {
				t.Error("the path contains a corner of the box")
			}
		})
	}
}

func TestRoundedRectangleBounds(t *testing.T) {
	poly := shape.NewRectangle(2, 1, shape.CornerRounding{Radius: 0.4}).Normalized()
	path := poly.ToPath(ui.Rect{X: 10, Y: 20, W: 200, H: 100})
	b := path.Bounds()
	if math.Abs(float64(b.X)-10) > 1e-3 || math.Abs(float64(b.Y)-45) > 1e-3 ||
		math.Abs(float64(b.W)-200) > 1e-3 || math.Abs(float64(b.H)-50) > 1e-3 {
		t.Errorf("path bounds %+v, want {10 45 200 50}", b)
	}
	if !path.Contains(110, 70) {
		t.Error("the path does not contain the rectangle's center")
	}
	if path.Contains(12, 47) {
		t.Error("the path contains a corner outside the rounded rectangle")
	}
	if path.Contains(110, 30) {
		t.Error("the path reaches above the rectangle")
	}
}

func TestUnroundedPolygonIsSharp(t *testing.T) {
	tri := shape.RegularPolygon(3, 100, shape.Unrounded)
	if got := len(tri.Cubics()); got != 3 {
		t.Fatalf("%d cubics, want the triangle's 3 edges", got)
	}
	path := tri.Normalized().ToPath(ui.Rect{W: 100, H: 100})
	// An equilateral triangle over the unit square leaves its two bottom
	// corners empty and holds its centroid.
	if path.Contains(2, 98) || path.Contains(98, 98) {
		t.Error("the path contains a corner beside the triangle")
	}
	if !path.Contains(50, 60) {
		t.Error("the path does not contain the triangle's centroid")
	}
}

func TestSmoothingFillsTheCorner(t *testing.T) {
	corner := func(smoothing float64) *ui.Path {
		poly := shape.NewRectangle(2, 2, shape.CornerRounding{Radius: 0.5, Smoothing: smoothing}).Normalized()
		return poly.ToPath(ui.Rect{W: 1, H: 1}) // identity over the unit square
	}
	round := corner(0)
	smooth := corner(1)
	if !round.Contains(0.5, 0.5) || !smooth.Contains(0.5, 0.5) {
		t.Fatal("a rounded square does not contain its center")
	}
	if round.Contains(0.95, 0.95) {
		t.Error("the circular corner reaches the square's corner")
	}
	if !smooth.Contains(0.95, 0.95) {
		t.Error("the smoothed corner does not fill toward the square's corner")
	}
	if !smooth.Contains(0.5, 0.05) {
		t.Error("the smoothed square lost its straight edges")
	}
}

func TestTransformComposition(t *testing.T) {
	p := shape.Point{X: 2, Y: 3}
	turned := shape.Rotate(90).Apply(p)
	if math.Abs(turned.X-(-3)) > 1e-9 || math.Abs(turned.Y-2) > 1e-9 {
		t.Errorf("Rotate(90) sends (2,3) to %+v, want (-3,2)", turned)
	}
	moved := shape.Translate(10, -4).Apply(p)
	if moved.X != 12 || moved.Y != -1 {
		t.Errorf("Translate sends (2,3) to %+v, want (12,-1)", moved)
	}
	// Translate then scale: (2+10)*2, (3-4)*2.
	both := shape.Translate(10, -4).Then(shape.Scale(2, 2)).Apply(p)
	if both.X != 24 || both.Y != -2 {
		t.Errorf("Translate.Then(Scale) sends (2,3) to %+v, want (24,-2)", both)
	}
}

func TestRenderQuadrants(t *testing.T) {
	poly := shape.NewCircle(0.5).Normalized()
	img := ui.Render(func(c *ui.Context) {
		ui.Box(c).Size(64, 64).Draw(func(p *ui.Painter, r ui.Rect) {
			p.FillPath(poly.ToPath(r), ui.RGB(255, 0, 0))
		})
	}, 64, 64, 1)
	if !isRed(img, 32, 32) {
		t.Error("the center of the rendered circle is empty")
	}
	if isRed(img, 2, 2) {
		t.Error("the rendered circle reaches the corner of its box")
	}
	quadrants := [4][2]int{{16, 16}, {48, 16}, {16, 48}, {48, 48}}
	for i, q := range quadrants {
		if !isRed(img, q[0], q[1]) {
			t.Errorf("quadrant %d (%v) has no fill", i, q)
		}
	}
	// Nothing may paint outside the element's 64×64 box: its corners stay
	// empty, though the circle's own edge may touch the box's sides.
	for _, q := range [][2]int{{0, 0}, {63, 0}, {0, 63}, {63, 63}} {
		if isRed(img, q[0], q[1]) {
			t.Errorf("the fill spilled to the canvas corner %v", q)
		}
	}
}

func finiteCubic(c shape.Cubic) bool {
	for _, p := range []shape.Point{c.P0, c.P1, c.P2, c.P3} {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
			return false
		}
	}
	return true
}

// isRed reports whether the pixel is the opaque red fills are drawn in.
func isRed(img *image.RGBA, x, y int) bool {
	c := img.RGBAAt(x, y)
	return c.R > 200 && c.G < 100 && c.B < 100 && c.A > 200
}
