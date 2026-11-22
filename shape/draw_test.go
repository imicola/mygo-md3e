package shape

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRoundedPolygonWriteTo(t *testing.T) {
	poly := Cookie9Sided()
	box := ui.Rect{X: 12, Y: 18, W: 60, H: 42}
	var dst ui.Path
	if got := poly.WriteTo(&dst, box); got != &dst {
		t.Fatal("WriteTo returned a path other than its destination")
	}
	if bounds := dst.Bounds(); bounds.W <= 0 || bounds.H <= 0 || !dst.Contains(42, 39) {
		t.Fatalf("expected a nonempty outline centered in %v, got %v", box, bounds)
	}
	if got := testing.AllocsPerRun(100, func() { poly.WriteTo(&dst, box) }); got != 0 {
		t.Fatalf("warmed WriteTo allocated %v times per call, want 0", got)
	}

	for _, tc := range []struct {
		poly *RoundedPolygon
		rect ui.Rect
	}{
		{nil, box},
		{&RoundedPolygon{}, box},
		{poly, ui.Rect{W: 0, H: 42}},
		{poly, ui.Rect{W: 60, H: -1}},
		{poly, ui.Rect{X: float32(math.NaN()), W: 60, H: 42}},
		{poly, ui.Rect{W: 60, H: float32(math.Inf(1))}},
	} {
		poly.WriteTo(&dst, box)
		if got := tc.poly.WriteTo(&dst, tc.rect); got != &dst || dst.Bounds() != (ui.Rect{}) {
			t.Fatalf("invalid geometry did not reset the destination for %v", tc.rect)
		}
	}
}

func TestMorphAsPathAllocations(t *testing.T) {
	morph := NewMorph(Cookie9Sided(), Circle())
	box := ui.Rect{X: 12, Y: 18, W: 60, H: 42}
	morph.AsPath(0.5, box)
	if got := testing.AllocsPerRun(100, func() { morph.AsPath(0.5, box) }); got != 0 {
		t.Fatalf("warmed AsPath allocated %v times per call, want 0", got)
	}
}
