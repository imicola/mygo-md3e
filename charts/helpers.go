package charts

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
)

func isFiniteF64(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func isFiniteF32(v float32) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
}

func isPositiveFiniteF32(v float32) bool {
	return isFiniteF32(v) && v > 0
}

func sanitizePositiveF32(v, fallback float32) float32 {
	if isPositiveFiniteF32(v) {
		return v
	}
	return fallback
}

func sanitizeNonNegativeF32(v, fallback float32) float32 {
	if isFiniteF32(v) && v >= 0 {
		return v
	}
	return fallback
}

func clampF32(v, lo, hi float32) float32 {
	if math.IsNaN(float64(v)) {
		return lo
	}
	if math.IsInf(float64(v), -1) || v < lo {
		return lo
	}
	if math.IsInf(float64(v), 1) || v > hi {
		return hi
	}
	return v
}

func clampF64(v, lo, hi float64) float64 {
	if math.IsNaN(v) {
		return lo
	}
	if math.IsInf(v, -1) || v < lo {
		return lo
	}
	if math.IsInf(v, 1) || v > hi {
		return hi
	}
	return v
}

func boolDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func rectIntersects(a, b ui.Rect) bool {
	return a.W > 0 && a.H > 0 && b.W > 0 && b.H > 0 &&
		a.X < b.X+b.W && a.X+a.W > b.X &&
		a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

type entranceLocalKey struct{}
type entranceAnimKey struct{}

type entranceState struct {
	started bool
}

// animateEntrance drives a 0 -> 1 spring entrance on first mount using MD3E's
// DefaultSpatial spring, snapping immediately to 1 when animate is false or
// ReduceMotion is active (zero pending frames).
func animateEntrance(el ui.Element, th *md3e.Theme, animate, reduce bool) float32 {
	st := ui.Local(el, entranceLocalKey{}, func() entranceState { return entranceState{} })
	if reduce || !animate {
		st.started = true
		return el.Jump(entranceAnimKey{}, 1)
	}
	anim := motion.Resolve(th.Motion, motion.DefaultSpatial, false)
	if !st.started {
		st.started = true
		el.Jump(entranceAnimKey{}, 0)
	}
	return el.AnimateTo(entranceAnimKey{}, 1, anim)
}

func animateF32(el ui.Element, key any, target float32, anim ui.Animation, reduce bool) float32 {
	if !isFiniteF32(target) {
		target = 0
	}
	if reduce {
		return el.Jump(key, target)
	}
	return el.AnimateTo(key, target, anim)
}

// buildRoundedRectPath writes a clockwise rounded rectangle with independent
// [tl, tr, br, bl] corner radii into dst without allocating.
func buildRoundedRectPath(dst *ui.Path, r ui.Rect, radii [4]float32) *ui.Path {
	dst.Reset()
	if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
		return dst
	}
	tl := sanitizeNonNegativeF32(radii[0], 0)
	tr := sanitizeNonNegativeF32(radii[1], 0)
	br := sanitizeNonNegativeF32(radii[2], 0)
	bl := sanitizeNonNegativeF32(radii[3], 0)

	// Clamp overlapping corner radii proportionally (CSS border-radius rule).
	scale := float32(1)
	if s := tl + tr; s > r.W && s > 0 {
		scale = min(scale, r.W/s)
	}
	if s := bl + br; s > r.W && s > 0 {
		scale = min(scale, r.W/s)
	}
	if s := tl + bl; s > r.H && s > 0 {
		scale = min(scale, r.H/s)
	}
	if s := tr + br; s > r.H && s > 0 {
		scale = min(scale, r.H/s)
	}
	tl *= scale
	tr *= scale
	br *= scale
	bl *= scale

	// Standard cubic Bézier kappa for a quarter-circle arc: 4/3 * (sqrt(2)-1).
	const k = float32(0.55228475)
	x0, y0 := r.X, r.Y
	x1, y1 := r.X+r.W, r.Y+r.H

	dst.MoveTo(x0+tl, y0)
	dst.LineTo(x1-tr, y0)
	if tr > 0 {
		dst.CubeTo(x1-tr*(1-k), y0, x1, y0+tr*(1-k), x1, y0+tr)
	}
	dst.LineTo(x1, y1-br)
	if br > 0 {
		dst.CubeTo(x1, y1-br*(1-k), x1-br*(1-k), y1, x1-br, y1)
	}
	dst.LineTo(x0+bl, y1)
	if bl > 0 {
		dst.CubeTo(x0+bl*(1-k), y1, x0, y1-bl*(1-k), x0, y1-bl)
	}
	dst.LineTo(x0, y0+tl)
	if tl > 0 {
		dst.CubeTo(x0, y0+tl*(1-k), x0+tl*(1-k), y0, x0+tl, y0)
	}
	dst.Close()
	return dst
}

// defaultFormatNumber formats numeric values concisely for tooltips and labels.
func defaultFormatNumber(v float64) string {
	if !isFiniteF64(v) {
		return "—"
	}
	if v == 0 {
		return "0"
	}
	abs := math.Abs(v)
	switch {
	case abs >= 1e9:
		return fmt.Sprintf("%.2fB", v/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("%.2fM", v/1e6)
	case abs >= 1e4:
		return fmt.Sprintf("%.1fK", v/1e3)
	case math.Abs(v-math.Round(v)) < 1e-6:
		return fmt.Sprintf("%.0f", v)
	case abs >= 100:
		return fmt.Sprintf("%.1f", v)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

func defaultFormatPercent(pct float64) string {
	if !isFiniteF64(pct) {
		return "0%"
	}
	pct = clampF64(pct, 0, 100)
	if pct > 0 && pct < 0.1 {
		return "<0.1%"
	}
	if math.Abs(pct-math.Round(pct)) < 0.05 {
		return fmt.Sprintf("%.0f%%", math.Round(pct))
	}
	return fmt.Sprintf("%.1f%%", pct)
}
