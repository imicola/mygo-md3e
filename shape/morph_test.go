package shape

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// morphPairs are the cross-topology pairs the task requires, by catalog name,
// with deep concavities on at least one side.
var morphPairs = [][2]string{
	{"Circle", "SoftBurst"},
	{"Square", "Heart"},
	{"Pentagon", "Sunny"},
	{"Pill", "Cookie9Sided"},
	{"Arrow", "Heart"},
	{"PixelTriangle", "SoftBurst"},
}

func morphShapes(t *testing.T, from, to string) *Morph {
	t.Helper()
	a, ok := ByName(from)
	if !ok {
		t.Fatalf("unknown shape %q", from)
	}
	b, ok := ByName(to)
	if !ok {
		t.Fatalf("unknown shape %q", to)
	}
	return NewMorph(a.Polygon(), b.Polygon())
}

// TestMorphChains checks the two outlines are cut into the same number of
// cubics, each a closed, finite chain that keeps the outline's bounds.
func TestMorphChains(t *testing.T) {
	for _, pair := range morphPairs {
		t.Run(pair[0]+"->"+pair[1], func(t *testing.T) {
			from, _ := ByName(pair[0])
			to, _ := ByName(pair[1])
			m := NewMorph(from.Polygon(), to.Polygon())
			if len(m.a) != len(m.b) {
				t.Fatalf("%d and %d cubics, want the same count", len(m.a), len(m.b))
			}
			if len(m.a) < 3 {
				t.Fatalf("%d cubics, want at least 3", len(m.a))
			}
			checkChain(t, m.a)
			checkChain(t, m.b)
			for _, tc := range []struct {
				name string
				got  Box
				want Box
			}{
				{"from", chainBounds(m.a), from.Polygon().Bounds()},
				{"to", chainBounds(m.b), to.Polygon().Bounds()},
			} {
				if math.Abs(tc.got.MinX-tc.want.MinX) > 1e-6 || math.Abs(tc.got.MinY-tc.want.MinY) > 1e-6 ||
					math.Abs(tc.got.MaxX-tc.want.MaxX) > 1e-6 || math.Abs(tc.got.MaxY-tc.want.MaxY) > 1e-6 {
					t.Errorf("%s bounds %+v, want %+v", tc.name, tc.got, tc.want)
				}
			}
		})
	}
}

func checkChain(t *testing.T, cubics []Cubic) {
	t.Helper()
	for i, c := range cubics {
		for _, p := range []Point{c.P0, c.P1, c.P2, c.P3} {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
				t.Fatalf("cubic %d is not finite: %+v", i, c)
			}
		}
		next := cubics[(i+1)%len(cubics)]
		if d := c.End().Distance(next.Start()); d > 1e-6 {
			t.Fatalf("cubic %d ends %v from cubic %d's start", i, d, (i+1)%len(cubics))
		}
	}
}

func chainBounds(cubics []Cubic) Box {
	b := cubics[0].Bounds()
	for _, c := range cubics[1:] {
		b = b.Union(c.Bounds())
	}
	return b
}

// TestMorphEndpoints renders each pair's ends and compares them with the
// shapes they come from.
func TestMorphEndpoints(t *testing.T) {
	const size = 64
	dst := ui.Rect{W: size, H: size}
	for _, pair := range morphPairs {
		t.Run(pair[0]+"->"+pair[1], func(t *testing.T) {
			from, _ := ByName(pair[0])
			to, _ := ByName(pair[1])
			m := NewMorph(from.Polygon(), to.Polygon())
			start := renderMorph(m, 0, dst, size, size)
			end := renderMorph(m, 1, dst, size, size)
			// The cut chain flattens and anti-aliases a little differently
			// from the original chain, so compare the filled masks.
			if d := maskDiff(start, renderShape(from.Polygon(), dst, size, size)); d > size*size/100 {
				t.Errorf("progress 0 differs from %s in %d pixels", pair[0], d)
			}
			if d := maskDiff(end, renderShape(to.Polygon(), dst, size, size)); d > size*size/100 {
				t.Errorf("progress 1 differs from %s in %d pixels", pair[1], d)
			}
		})
	}
}

// TestMorphIntermediates renders the steps between each pair and checks they
// are filled, stay inside their box, and never cross themselves.
func TestMorphIntermediates(t *testing.T) {
	const size = 96
	dst := ui.Rect{X: 16, Y: 16, W: 64, H: 64}
	for _, pair := range morphPairs {
		t.Run(pair[0]+"->"+pair[1], func(t *testing.T) {
			m := morphShapes(t, pair[0], pair[1])
			for _, progress := range []float32{0, 0.25, 0.5, 0.75, 1} {
				chain := make([]Cubic, len(m.a))
				for i := range m.a {
					chain[i] = m.cubicAt(i, float64(progress))
				}
				if a, b, c, d, ok := chainSelfIntersects(chain); ok {
					t.Errorf("progress %v crosses itself: %+v-%+v meets %+v-%+v", progress, a, b, c, d)
				}
				for i := range m.a {
					c := m.cubicAt(i, float64(progress))
					for _, p := range []Point{c.P0, c.P1, c.P2, c.P3} {
						if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
							t.Fatalf("progress %v cubic %d is not finite: %+v", progress, i, c)
						}
					}
				}
				img := renderMorph(m, progress, dst, size, size)
				ink := 0
				for y := 0; y < size; y++ {
					for x := 0; x < size; x++ {
						if !isDark(img, x, y) {
							continue
						}
						ink++
						if x < 16 || x >= 80 || y < 16 || y >= 80 {
							t.Fatalf("progress %v paints (%d,%d) outside its box", progress, x, y)
						}
					}
				}
				if ink < 200 {
					t.Errorf("progress %v painted %d pixels, want a filled shape", progress, ink)
				}
			}
		})
	}
}

// chainSelfIntersects samples the closed chain into a polyline and reports
// whether two of its non-adjacent segments cross, with where.
func chainSelfIntersects(cubics []Cubic) (Point, Point, Point, Point, bool) {
	const samples = 16
	points := make([]Point, 0, len(cubics)*samples)
	for _, c := range cubics {
		for i := range samples {
			points = append(points, c.Point(float64(i)/samples))
		}
	}
	n := len(points)
	for i := range n {
		a, b := points[i], points[(i+1)%n]
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue // adjacent through where the chain closes
			}
			c, d := points[j], points[(j+1)%n]
			if segmentsCross(a, b, c, d) {
				return a, b, c, d, true
			}
		}
	}
	return Point{}, Point{}, Point{}, Point{}, false
}

// segmentsCross reports whether the open segments ab and cd cross.
func segmentsCross(a, b, c, d Point) bool {
	ab, cd := b.Sub(a), d.Sub(c)
	d1, d2 := ab.Cross(c.Sub(a)), ab.Cross(d.Sub(a))
	d3, d4 := cd.Cross(a.Sub(c)), cd.Cross(b.Sub(c))
	return d1*d2 < 0 && d3*d4 < 0
}

// TestMorphAllShapes builds every catalog shape's morph with the circle and
// with its neighbour, checking the cut chains stay finite and aligned.
func TestMorphAllShapes(t *testing.T) {
	all := All()
	circle := Circle()
	for i, s := range all {
		next := all[(i+1)%len(all)]
		for _, pair := range [][2]*RoundedPolygon{
			{circle, s.Polygon()},
			{s.Polygon(), circle},
			{s.Polygon(), next.Polygon()},
		} {
			m := NewMorph(pair[0], pair[1])
			if len(m.a) != len(m.b) || len(m.a) < 3 {
				t.Fatalf("%s: chains of %d and %d cubics", s.Name, len(m.a), len(m.b))
			}
			for _, progress := range []float64{0, 0.25, 0.5, 0.75, 1} {
				for j := range m.a {
					c := m.cubicAt(j, progress)
					for _, p := range []Point{c.P0, c.P1, c.P2, c.P3} {
						if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) {
							t.Fatalf("%s at %v: cubic %d is not finite: %+v", s.Name, progress, j, c)
						}
					}
				}
			}
		}
	}
}

// TestClipShape checks the drawing helper clips a plain fill to the shape.
func TestClipShape(t *testing.T) {
	img := ui.Render(func(c *ui.Context) {
		ui.Box(c).Draw(func(p *ui.Painter, r ui.Rect) {
			ClipShape(p, Circle(), ui.Rect{W: 64, H: 64}, func() {
				p.Fill(ui.Rect{W: 64, H: 64}, ui.RGB(0, 0, 0), 0)
			})
		})
	}, 64, 64, 1)
	if !isDark(img, 32, 32) {
		t.Error("the clipped fill is empty at the center")
	}
	if isDark(img, 2, 2) {
		t.Error("the clipped fill escapes the circle's corner")
	}
}

// TestAsPathReuse checks the morph's path is reused between calls.
func TestAsPathReuse(t *testing.T) {
	m := morphShapes(t, "Circle", "Heart")
	first := m.AsPath(0.25, ui.Rect{W: 10, H: 10})
	second := m.AsPath(0.75, ui.Rect{W: 10, H: 10})
	if first != second {
		t.Error("AsPath allocated a new path instead of reusing the morph's")
	}
}

// renderShape draws poly filled black into dst on a width×height canvas.
func renderShape(poly *RoundedPolygon, dst ui.Rect, width, height int) *image.RGBA {
	path := poly.ToPath(dst)
	return ui.Render(func(c *ui.Context) {
		ui.Box(c).Draw(func(p *ui.Painter, r ui.Rect) {
			p.FillPath(path, ui.RGB(0, 0, 0))
		})
	}, width, height, 1)
}

// renderMorph draws m at progress filled black into dst on a width×height
// canvas.
func renderMorph(m *Morph, progress float32, dst ui.Rect, width, height int) *image.RGBA {
	return ui.Render(func(c *ui.Context) {
		ui.Box(c).Draw(func(p *ui.Painter, r ui.Rect) {
			DrawMorph(p, m, progress, dst, ui.RGB(0, 0, 0))
		})
	}, width, height, 1)
}

// maskDiff returns how many pixels of the two canvases fall on opposite
// sides of the fill line.
func maskDiff(a, b *image.RGBA) int {
	bounds := a.Bounds()
	different := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if isDark(a, x, y) != isDark(b, x, y) {
				different++
			}
		}
	}
	return different
}

// isDark reports whether the pixel holds the black shapes are drawn in, over
// the renderer's white background.
func isDark(img *image.RGBA, x, y int) bool {
	c := img.RGBAAt(x, y)
	return c.R < 128 && c.G < 128 && c.B < 128 && c.A > 200
}

// TestMorphNilProgressAndEmptyRectSafe verifies nil Morph receivers, non-finite
// progress values, and empty/non-finite rects stay panic-free and finite.
func TestMorphNilProgressAndEmptyRectSafe(t *testing.T) {
	var nilMorph *Morph
	if p := nilMorph.AsPath(0.5, ui.Rect{W: 20, H: 20}); p == nil {
		t.Fatal("nil Morph.AsPath returned nil")
	}
	if cubics := nilMorph.CubicsAt(0.5); cubics != nil {
		t.Fatalf("nil Morph.CubicsAt = %v, want nil", cubics)
	}

	m := morphShapes(t, "Circle", "Heart")
	for _, prog := range []float32{
		float32(math.NaN()),
		float32(math.Inf(-1)),
		float32(math.Inf(1)),
		-0.5,
		1.5,
	} {
		cubics := m.CubicsAt(prog)
		if len(cubics) == 0 {
			t.Fatalf("CubicsAt(%v) empty", prog)
		}
		for j, c := range cubics {
			for _, pt := range []Point{c.P0, c.P1, c.P2, c.P3} {
				if math.IsNaN(pt.X) || math.IsNaN(pt.Y) || math.IsInf(pt.X, 0) || math.IsInf(pt.Y, 0) {
					t.Fatalf("CubicsAt(%v)[%d] non-finite: %+v", prog, j, c)
				}
			}
		}
		path := m.AsPath(prog, ui.Rect{X: 2, Y: 2, W: 28, H: 28})
		if b := path.Bounds(); b.W <= 0 || b.H <= 0 {
			t.Fatalf("AsPath(%v) produced empty bounds %+v", prog, b)
		}
	}

	ui.Render(func(c *ui.Context) {
		ui.Box(c).Draw(func(p *ui.Painter, r ui.Rect) {
			DrawMorph(p, nilMorph, 0.5, ui.Rect{W: 32, H: 32}, ui.RGB(0, 0, 0))
			DrawMorph(p, m, float32(math.NaN()), ui.Rect{W: 0, H: 32}, ui.RGB(0, 0, 0))
			DrawMorph(p, m, 0.5, ui.Rect{W: float32(math.NaN()), H: 32}, ui.RGB(0, 0, 0))
			DrawMorph(p, m, float32(math.NaN()), ui.Rect{W: 32, H: 32}, ui.RGB(0, 0, 0))
		})
	}, 32, 32, 1)
}
