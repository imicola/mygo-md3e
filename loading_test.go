package md3e

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/shape"
)

// loadingCanvas builds a 320x200 white canvas holding build's indicators.
func loadingCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(320, 200).Background(canvasRGB).Padding(20).AlignItems(ui.Start).Children(func() {
				build(c)
			})
		})
	}, 320, 200)
}

// loadingInfo returns a progress indicator's access info by name.
func loadingInfo(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleProgress, name)
	if !ok {
		t.Fatalf("%q progress indicator not in the access tree", name)
	}
	return info
}

// TestLoadingMorphPairsAreSimple verifies that each adjacent pair of the default
// seven loading shapes keeps a finite, non-self-intersecting outline through the
// whole morph.
func TestLoadingMorphPairsAreSimple(t *testing.T) {
	polys := DefaultLoadingPolygons()
	if len(polys) != 7 {
		t.Fatalf("default sequence has %d shapes, want 7", len(polys))
	}
	for i := range polys {
		from, to := polys[i], polys[(i+1)%len(polys)]
		m := shape.NewMorph(from, to)
		for step := 0; step <= 20; step++ {
			progress := float32(step) / 20
			chain := m.CubicsAt(progress)
			if len(chain) < 3 {
				t.Fatalf("pair %d at %v has %d cubics", i, progress, len(chain))
			}
			for j, cubic := range chain {
				for _, pt := range []shape.Point{cubic.P0, cubic.P1, cubic.P2, cubic.P3} {
					if math.IsNaN(pt.X) || math.IsNaN(pt.Y) || math.IsInf(pt.X, 0) || math.IsInf(pt.Y, 0) {
						t.Fatalf("pair %d at %v cubic %d is not finite: %+v", i, progress, j, cubic)
					}
				}
			}
			if a, b, c, d, crossed := loadingChainCrosses(chain); crossed {
				t.Fatalf("pair %d at %v crosses itself: %+v-%+v meets %+v-%+v", i, progress, a, b, c, d)
			}
		}
	}
}

// loadingChainCrosses samples a closed cubic chain into a polyline and reports
// whether two non-adjacent segments cross.
func loadingChainCrosses(cubics []shape.Cubic) (shape.Point, shape.Point, shape.Point, shape.Point, bool) {
	const samples = 12
	points := make([]shape.Point, 0, len(cubics)*samples)
	for _, cubic := range cubics {
		for i := range samples {
			points = append(points, cubic.Point(float64(i)/samples))
		}
	}
	n := len(points)
	for i := range n {
		a, b := points[i], points[(i+1)%n]
		if a == b {
			continue
		}
		for j := i + 2; j < n; j++ {
			if i == 0 && j == n-1 {
				continue
			}
			c, d := points[j], points[(j+1)%n]
			if c == d {
				continue
			}
			ab, cd := b.Sub(a), d.Sub(c)
			d1, d2 := ab.Cross(c.Sub(a)), ab.Cross(d.Sub(a))
			d3, d4 := cd.Cross(a.Sub(c)), cd.Cross(b.Sub(c))
			if d1*d2 < 0 && d3*d4 < 0 {
				return a, b, c, d, true
			}
		}
	}
	return shape.Point{}, shape.Point{}, shape.Point{}, shape.Point{}, false
}

// indicatorBoxDiff counts the pixels of two frames that differ inside box.
func indicatorBoxDiff(a, b *image.RGBA, box ui.Rect) int {
	different := 0
	for y := int(box.Y) + 1; y < int(box.Y+box.H)-1; y++ {
		for x := int(box.X) + 1; x < int(box.X+box.W)-1; x++ {
			pa, pb := a.RGBAAt(x, y), b.RGBAAt(x, y)
			if absColorDiff(pa, pb) > 8 {
				different++
			}
		}
	}
	return different
}

// indicatorOverflowPixels reports how many pixels around box paint outside it.
func indicatorOverflowPixels(img *image.RGBA, box ui.Rect) int {
	count := 0
	for y := int(box.Y) - 2; y < int(box.Y+box.H)+2; y++ {
		for x := int(box.X) - 2; x < int(box.X+box.W)+2; x++ {
			if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
				continue
			}
			if x >= int(box.X) && x < int(box.X+box.W) && y >= int(box.Y) && y < int(box.Y+box.H) {
				continue
			}
			if !colorMatches(img.RGBAAt(x, y), canvasRGB, 6) {
				count++
			}
		}
	}
	return count
}

// TestLoadingIndicatorAnimates checks that the shape moves and turns between
// sampled times and that no stage paints outside the component's box.
func TestLoadingIndicatorAnimates(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := loadingCanvas(th, func(c *ui.Context) {
		LoadingIndicator(c, LoadingOptions{Size: 64, Label: "Busy"})
	})
	info := loadingInfo(t, tt, "Busy")
	if math.Abs(float64(info.Bounds.W)-64) > 0.5 || math.Abs(float64(info.Bounds.H)-64) > 0.5 {
		t.Fatalf("indicator box %v, want 64x64", info.Bounds)
	}
	if info.Now >= info.Min {
		t.Fatalf("indeterminate indicator reports value %v", info.Now)
	}
	first := tt.Image()
	if n := indicatorOverflowPixels(first, info.Bounds); n != 0 {
		t.Fatalf("first frame paints %d pixels outside its box", n)
	}

	tt.Advance(300 * time.Millisecond)
	mid := tt.Image()
	if n := indicatorOverflowPixels(mid, info.Bounds); n != 0 {
		t.Fatalf("300ms frame paints %d pixels outside its box", n)
	}
	tt.Advance(350 * time.Millisecond)
	late := tt.Image()
	if n := indicatorOverflowPixels(late, info.Bounds); n != 0 {
		t.Fatalf("650ms frame paints %d pixels outside its box", n)
	}

	area := int(info.Bounds.W) * int(info.Bounds.H)
	if n := indicatorBoxDiff(first, mid, info.Bounds); n < area/20 {
		t.Fatalf("0ms to 300ms changed %d pixels, want a visible morph", n)
	}
	if n := indicatorBoxDiff(mid, late, info.Bounds); n < area/20 {
		t.Fatalf("300ms to 650ms changed %d pixels, want a visible morph", n)
	}
}

// TestLoadingStylesColors checks the contained circle and its inset
// OnPrimaryContainer shape against the standard indicator's bare Primary shape.
func TestLoadingStylesColors(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	tt := loadingCanvas(th, func(c *ui.Context) {
		ui.Column(c).Gap(20).Children(func() {
			LoadingIndicator(c, LoadingOptions{Label: "Standard"})
			LoadingIndicator(c, LoadingOptions{Style: LoadingContained, Label: "Contained"})
		})
	})
	std := loadingInfo(t, tt, "Standard")
	con := loadingInfo(t, tt, "Contained")
	img := tt.Image()

	// The standard indicator has no container: its left edge is the canvas.
	if px := img.RGBAAt(int(std.Bounds.X)+2, int(std.Bounds.Y+std.Bounds.H/2)); !colorMatches(px, canvasRGB, 6) {
		t.Fatalf("standard indicator paints a container: %+v", px)
	}
	if px := img.RGBAAt(int(std.Bounds.X+std.Bounds.W/2), int(std.Bounds.Y+std.Bounds.H/2)); !colorMatches(px, cs.Primary, 40) {
		t.Fatalf("standard shape center %+v is not Primary %+v", px, rgbOf(cs.Primary))
	}

	// The contained indicator fills a PrimaryContainer circle, its shape inset
	// in OnPrimaryContainer.
	if px := img.RGBAAt(int(con.Bounds.X)+2, int(con.Bounds.Y+con.Bounds.H/2)); !colorMatches(px, cs.PrimaryContainer, 20) {
		t.Fatalf("contained circle edge %+v is not PrimaryContainer %+v", px, rgbOf(cs.PrimaryContainer))
	}
	if px := img.RGBAAt(int(con.Bounds.X+con.Bounds.W/2), int(con.Bounds.Y+con.Bounds.H/2)); !colorMatches(px, cs.OnPrimaryContainer, 40) {
		t.Fatalf("contained shape center %+v is not OnPrimaryContainer %+v", px, rgbOf(cs.OnPrimaryContainer))
	}
}

// TestLoadingPausedAndReduceMotionAreStill checks both a paused indicator and a
// reduced-motion one ask for no frames and paint the same pixels as time passes.
func TestLoadingPausedAndReduceMotionAreStill(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := loadingCanvas(th, func(c *ui.Context) {
		LoadingIndicator(c, LoadingOptions{Label: "Paused", Paused: true})
	})
	if tt.FramePending() {
		t.Fatal("a paused indicator asks for frames")
	}
	before := tt.Image()
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("a paused indicator asks for frames after time passes")
	}
	if n := indicatorBoxDiff(before, tt.Image(), ui.Rect{X: 0, Y: 0, W: 320, H: 200}); n != 0 {
		t.Fatalf("a paused indicator repainted %d pixels", n)
	}
	loadingInfo(t, tt, "Paused")

	stillTh := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	still := loadingCanvas(stillTh, func(c *ui.Context) {
		LoadingIndicator(c, LoadingOptions{Label: "Still"})
	})
	if still.FramePending() {
		t.Fatal("a reduced-motion indicator asks for frames")
	}
	first := still.Image()
	still.Advance(time.Second)
	if still.FramePending() {
		t.Fatal("a reduced-motion indicator asks for frames after time passes")
	}
	if n := indicatorBoxDiff(first, still.Image(), ui.Rect{X: 0, Y: 0, W: 320, H: 200}); n != 0 {
		t.Fatalf("a reduced-motion indicator repainted %d pixels", n)
	}
}

// TestLoadingDisabledAncestorAndCustomSequence checks a disabled ancestor reaches
// assistive technology and a custom, single-shape sequence stays quiet.
func TestLoadingDisabledAncestorAndCustomSequence(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	pill := shape.Pill()
	tt := loadingCanvas(th, func(c *ui.Context) {
		ui.Box(c).Disabled(true).Children(func() {
			LoadingIndicator(c, LoadingOptions{Label: "Disabled", Size: 40, Paused: true})
		})
		LoadingIndicator(c, LoadingOptions{
			Label:    "Custom",
			Polygons: []*shape.RoundedPolygon{pill},
			Paused:   true,
			Size:     32,
		})
	})
	disabled := loadingInfo(t, tt, "Disabled")
	if !disabled.Disabled {
		t.Fatal("a disabled ancestor does not reach the progress indicator")
	}
	if math.Abs(float64(disabled.Bounds.W)-40) > 0.5 {
		t.Fatalf("custom size box %v, want 40", disabled.Bounds)
	}
	custom := loadingInfo(t, tt, "Custom")
	if math.Abs(float64(custom.Bounds.W)-32) > 0.5 {
		t.Fatalf("custom sequence box %v, want 32", custom.Bounds)
	}
	if tt.FramePending() {
		t.Fatal("a paused custom sequence asks for frames")
	}
}
