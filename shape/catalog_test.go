package shape_test

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/shape"
)

// wantShapes is the catalog order docs/md3e-spec.md §3.3 gives.
var wantShapes = []string{
	"Circle", "Square", "Slanted", "Arch", "Fan", "Arrow", "SemiCircle", "Oval",
	"Pill", "Triangle", "Diamond", "ClamShell", "Pentagon", "Gem", "Sunny",
	"VerySunny", "Cookie4Sided", "Cookie6Sided", "Cookie7Sided", "Cookie9Sided",
	"Cookie12Sided", "Ghostish", "Clover4Leaf", "Clover8Leaf", "Burst",
	"SoftBurst", "Boom", "SoftBoom", "Flower", "Puffy", "PuffyDiamond",
	"PixelCircle", "PixelTriangle", "Bun", "Heart",
}

func TestCatalogNames(t *testing.T) {
	all := shape.All()
	if len(all) != 35 {
		t.Fatalf("%d shapes, want 35", len(all))
	}
	seen := make(map[string]bool, len(all))
	for i, s := range all {
		if s.Name != wantShapes[i] {
			t.Errorf("shape %d is %q, want %q", i, s.Name, wantShapes[i])
		}
		if seen[s.Name] {
			t.Errorf("duplicate shape name %q", s.Name)
		}
		seen[s.Name] = true
		if s.Polygon == nil {
			t.Fatalf("%s has no polygon builder", s.Name)
		}
		if byName, ok := shape.ByName(s.Name); !ok || byName.Name != s.Name {
			t.Errorf("ByName(%q) did not find the shape", s.Name)
		}
	}
	if _, ok := shape.ByName("NoSuchShape"); ok {
		t.Error("ByName found an unknown shape")
	}
}

func TestCatalogGeometry(t *testing.T) {
	for _, s := range shape.All() {
		t.Run(s.Name, func(t *testing.T) {
			poly := s.Polygon()
			if poly == nil {
				t.Fatal("no polygon")
			}
			cubics := poly.Cubics()
			if len(cubics) < 3 {
				t.Fatalf("%d cubics, want at least 3", len(cubics))
			}
			for i, c := range cubics {
				if !finiteCubic(c) {
					t.Fatalf("cubic %d is not finite: %+v", i, c)
				}
				// The anchors sit on the shape, inside the unit square; the
				// control points of a rounded corner may bulge a little past
				// the curve's bounds, as Bézier controls do, but stay near.
				for _, p := range []shape.Point{c.P0, c.P3} {
					if p.X < -1e-6 || p.X > 1+1e-6 || p.Y < -1e-6 || p.Y > 1+1e-6 {
						t.Fatalf("cubic %d has an anchor outside the unit square: %+v", i, p)
					}
				}
				for _, p := range []shape.Point{c.P1, c.P2} {
					if p.X < -0.1 || p.X > 1.1 || p.Y < -0.1 || p.Y > 1.1 {
						t.Fatalf("cubic %d has a control point far outside the unit square: %+v", i, p)
					}
				}
				if next := cubics[(i+1)%len(cubics)]; c.End().Distance(next.Start()) > 1e-6 {
					t.Fatalf("cubic %d does not meet cubic %d", i, (i+1)%len(cubics))
				}
			}
			b := poly.Bounds()
			if b.MinX < -1e-6 || b.MinY < -1e-6 || b.MaxX > 1+1e-6 || b.MaxY > 1+1e-6 {
				t.Errorf("bounds %+v are outside the unit square", b)
			}
			if w, h := b.Width(), b.Height(); math.Abs(math.Max(w, h)-1) > 1e-6 {
				t.Errorf("bounds %v×%v do not fill one side of the unit square", w, h)
			}
			path := poly.ToPath(ui.Rect{W: 1, H: 1})
			if !path.Contains(0.5, 0.5) {
				t.Error("the shape does not contain the center of its square")
			}
		})
	}
}

func TestCatalogRender(t *testing.T) {
	for _, s := range shape.All() {
		t.Run(s.Name, func(t *testing.T) {
			poly := s.Polygon()
			img := ui.Render(func(c *ui.Context) {
				ui.Box(c).Draw(func(p *ui.Painter, r ui.Rect) {
					p.FillPath(poly.ToPath(ui.Rect{X: 16, Y: 16, W: 64, H: 64}), ui.RGB(0, 0, 0))
				})
			}, 96, 96, 1)
			filled := 0
			for y := 0; y < 96; y++ {
				for x := 0; x < 96; x++ {
					if !isInk(img, x, y) {
						continue
					}
					filled++
					if x < 16 || x >= 80 || y < 16 || y >= 80 {
						t.Fatalf("pixel (%d,%d) painted outside the shape's 64×64 box", x, y)
					}
				}
			}
			if filled < 100 {
				t.Errorf("%d pixels painted, want a filled shape", filled)
			}
		})
	}
}

// isInk reports whether the pixel holds the black shapes are drawn in, over
// the renderer's white background.
func isInk(img *image.RGBA, x, y int) bool {
	c := img.RGBAAt(x, y)
	return c.R < 128 && c.G < 128 && c.B < 128 && c.A > 200
}
