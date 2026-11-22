package media

import (
	"math"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

type cinemaHUDMemoKey struct{}

type cinemaHUDMemo struct {
	primary      ui.Color
	contrast     tokens.Contrast
	density      tokens.Density
	reduceMotion bool
	theme        *md3e.Theme
}

// cinemaHUDTheme returns a memoized dark-mode MD3E sub-theme derived from base's
// primary hue so video and lightbox chrome stays legible dark glass over media
// even when the host app is in Light mode.
func cinemaHUDTheme(host ui.Element, base *md3e.Theme) *md3e.Theme {
	if base == nil || base.Dark {
		return base
	}
	memo := ui.Local(host, cinemaHUDMemoKey{}, func() cinemaHUDMemo { return cinemaHUDMemo{} })
	if memo.theme == nil ||
		memo.primary != base.Colors.Primary ||
		memo.contrast != base.Contrast ||
		memo.density != base.Density ||
		memo.reduceMotion != base.ReduceMotion ||
		memo.theme.Shapes != base.Shapes {
		seed := base.Colors.Primary
		if seed.A == 0 {
			seed = md3e.DefaultSeedColor
		}
		th := md3e.ThemeFromSeed(seed, md3e.ThemeOptions{
			Dark:     true,
			Contrast: base.Contrast,
			Density:  base.Density,
		})
		th.ReduceMotion = base.ReduceMotion
		th.Shapes = base.Shapes
		th.Motion = base.Motion
		th.Font = base.Font
		th.MonoFont = base.MonoFont
		th.TextScale = base.TextScale
		memo.primary = base.Colors.Primary
		memo.contrast = base.Contrast
		memo.density = base.Density
		memo.reduceMotion = base.ReduceMotion
		memo.theme = th
	}
	return memo.theme
}

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
