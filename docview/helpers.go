package docview

import (
	"math"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

func boolDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func sanitizePositiveF32(v, def float32) float32 {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) || v <= 0 {
		return def
	}
	return v
}

func sanitizeNonNegF32(v float32) float32 {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) || v < 0 {
		return 0
	}
	return v
}

func pickColor(candidates ...ui.Color) ui.Color {
	for _, c := range candidates {
		if c.A > 0 {
			return c
		}
	}
	return ui.Color{}
}

func resolveHairlineColor(th *md3e.Theme) ui.Color {
	if th == nil {
		th = md3e.DefaultTheme(nil)
	}
	cs := th.Colors
	if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
		return cs.Outline
	}
	base := pickColor(cs.OutlineVariant, cs.Outline)
	if base.A == 0 {
		if th.Dark {
			return ui.RGBA(255, 255, 255, 0.14)
		}
		return ui.RGBA(0, 0, 0, 0.12)
	}
	if th.Dark {
		if cs.OnSurface.A > 0 {
			return base.Mix(cs.OnSurface, 0.15).Alpha(0.42)
		}
		return base.Alpha(0.42)
	}
	return base.Alpha(0.45)
}

func resolveCodeRadius(th *md3e.Theme, scale tokens.ShapeScale, explicit float32) float32 {
	if isPos := sanitizePositiveF32(explicit, 0); isPos > 0 {
		return isPos
	}
	if scale > tokens.ShapeNone && scale <= tokens.ShapeFull {
		return th.Shapes.Radius(scale)
	}
	r := th.Shapes.Radius(tokens.ShapeMedium)
	if r <= 0 {
		return 12
	}
	return r
}

type docCodeMetrics struct {
	fontSize    float32
	lineHeight  float32
	headerPadY  float32
	headerPadX  float32
	bodyPad     float32
	copyHeight  float32
	lineNumGap  float32
	lineNumSize float32
}

func docCodeMetricsFor(th *md3e.Theme) docCodeMetrics {
	scale := th.EffectiveTextScale()
	switch th.Density.Normalize() {
	case tokens.DensityCompact:
		return docCodeMetrics{
			fontSize:    12 * scale,
			lineHeight:  1.40,
			headerPadY:  5,
			headerPadX:  10,
			bodyPad:     10,
			copyHeight:  24,
			lineNumGap:  10,
			lineNumSize: 11 * scale,
		}
	case tokens.DensityDesktop:
		return docCodeMetrics{
			fontSize:    13 * scale,
			lineHeight:  1.45,
			headerPadY:  6,
			headerPadX:  12,
			bodyPad:     12,
			copyHeight:  26,
			lineNumGap:  12,
			lineNumSize: 12 * scale,
		}
	default:
		return docCodeMetrics{
			fontSize:    14 * scale,
			lineHeight:  1.48,
			headerPadY:  8,
			headerPadX:  14,
			bodyPad:     14,
			copyHeight:  28,
			lineNumGap:  14,
			lineNumSize: 12.5 * scale,
		}
	}
}

func docIconBox(c *ui.Context, draw md3e.IconDraw, size float32, ink ui.Color) ui.Element {
	return ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		if draw == nil || r.W <= 0 || r.H <= 0 || ink.A == 0 {
			return
		}
		draw(p, r, ink)
	})
}
