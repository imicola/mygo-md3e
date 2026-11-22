package media

import (
	"math"

	"github.com/egoist/mygo/ui"
)

func animateF32(el ui.Element, key any, target float32, anim ui.Animation, reduce bool) float32 {
	if math.IsNaN(float64(target)) || math.IsInf(float64(target), 0) {
		target = 0
	}
	if reduce {
		return el.Jump(key, target)
	}
	return el.AnimateTo(key, target, anim)
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

func isFiniteF32(v float32) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
}

func isPositiveFiniteF32(v float32) bool {
	return isFiniteF32(v) && v > 0
}

func firstNonEmpty(vals ...string) string {
	for _, s := range vals {
		if s != "" {
			return s
		}
	}
	return ""
}
