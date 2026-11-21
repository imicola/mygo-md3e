package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// SurfaceColorForElevation maps level (Level0..Level5) to MD3E's tonal Surface
// Container step scale (Surface -> SurfaceContainerLow -> SurfaceContainer ->
// SurfaceContainerHigh -> SurfaceContainerHighest), falling back to tonal tint
// mixing if container roles are unset in a partial custom Theme.
func SurfaceColorForElevation(t *Theme, level tokens.ElevationLevel) ui.Color {
	if t == nil {
		t = DefaultTheme(nil)
	}
	cs := t.Colors
	base := cs.Surface
	var chosen ui.Color
	switch {
	case level <= tokens.Level0:
		chosen = base
	case level == tokens.Level1:
		chosen = cs.SurfaceContainerLow
	case level == tokens.Level2:
		chosen = cs.SurfaceContainer
	case level == tokens.Level3:
		chosen = cs.SurfaceContainerHigh
	case level == tokens.Level4:
		if cs.SurfaceContainerHigh.A > 0 && cs.SurfaceContainerHighest.A > 0 {
			chosen = cs.SurfaceContainerHigh.Mix(cs.SurfaceContainerHighest, 0.5)
		} else {
			chosen = cs.SurfaceContainerHigh
		}
	default:
		chosen = cs.SurfaceContainerHighest
	}
	if chosen.A == 0 {
		return ApplyTonalElevation(t, base, level)
	}
	return chosen
}

// ApplyTonalElevation mixes t.Colors.SurfaceTint onto base using level's
// MD3E TintRatio (0% at Level0 up to 14% at Level5).
func ApplyTonalElevation(t *Theme, base ui.Color, level tokens.ElevationLevel) ui.Color {
	if level <= tokens.Level0 || base.A == 0 {
		return base
	}
	if t == nil {
		t = DefaultTheme(nil)
	}
	tint := t.Colors.SurfaceTint
	if tint.A == 0 {
		tint = t.Colors.Primary
	}
	ratio := float32(level.TintRatio())
	if ratio <= 0 || tint.A == 0 {
		return base
	}
	return base.Mix(tint, ratio)
}

// ApplyElevationShadows appends MD3E Key and Ambient shadows for level to el
// when level > Level0.
func ApplyElevationShadows(t *Theme, el ui.Element, level tokens.ElevationLevel) ui.Element {
	if level <= tokens.Level0 {
		return el
	}
	if t == nil {
		t = DefaultTheme(nil)
	}
	spec := level.Spec()
	shadowCol := t.Colors.Shadow
	if shadowCol.A == 0 {
		shadowCol = ui.RGB(0, 0, 0)
	}
	if spec.Key.Opacity > 0 {
		el.Shadow(0, float32(spec.Key.OffsetY), float32(spec.Key.Blur), float32(spec.Key.Spread),
			shadowCol.Alpha(float32(spec.Key.Opacity)))
	}
	if spec.Ambient.Opacity > 0 {
		el.Shadow(0, float32(spec.Ambient.OffsetY), float32(spec.Ambient.Blur), float32(spec.Ambient.Spread),
			shadowCol.Alpha(float32(spec.Ambient.Opacity)))
	}
	return el
}

// SurfaceOptions configures a Material Design 3 Expressive Surface container.
type SurfaceOptions struct {
	// Elevation selects the MD3E tonal surface tier and Key/Ambient drop shadows.
	Elevation tokens.ElevationLevel
	// TonalElevation optionally overrides the tonal elevation step used when
	// Color is unset (or mixed via SurfaceTint when TonalTint is true). A
	// non-zero value may raise or lower the tone independently of Elevation's
	// shadows.
	TonalElevation tokens.ElevationLevel
	// TonalTint mixes SurfaceTint onto an explicit Color using the active elevation's TintRatio.
	TonalTint bool
	// Color overrides the background fill. When zero, the tonal surface color for
	// TonalElevation (or Elevation) is resolved from CurrentTheme(c).
	Color ui.Color
	// ContentColor overrides the foreground TextColor inherited by child elements.
	// When zero, Theme.ContentColorFor(bg) picks the accessible On* role.
	ContentColor ui.Color
	// BorderWidth draws an outline border in DIPs around the surface.
	BorderWidth float32
	// BorderColor overrides the outline color (defaults to Theme.Colors.OutlineVariant).
	BorderColor ui.Color
	// Hairline draws a subtle 1 DIP tonal highlight border when BorderWidth <= 0,
	// adapting to light, dark, and high-contrast themes for crisp surface separation.
	Hairline bool
	// HairlineColor overrides the default Hairline stroke color when A > 0.
	HairlineColor ui.Color

	// Optional gradient fills applied over the surface container. When both are
	// supplied, LinearGradient paints first and RadialGradient overlays it.
	LinearGradient *ui.LinearGradient
	RadialGradient *ui.RadialGradient

	// Backdrop optionally paints custom decorative graphics (such as Hero card
	// organic waves or clipped geometric patterns) inside the surface bounds
	// after the fill/gradient and before children. The callback must be pure
	// and must not request animation frames or query element state so exit
	// transitions remain ghost-draw safe.
	Backdrop func(p *ui.Painter, r ui.Rect)

	// Shape selection priority: Shape polygon > Radii [4] > Radius > ShapeScale.
	ShapeScale tokens.ShapeScale
	Radius     float32
	HasRadius  bool
	Radii      [4]float32
	HasRadii   bool
	Shape      *shape.RoundedPolygon

	// Typography inheritance (optional): applies TypeRole onto the surface box.
	TypeRole       tokens.TypeRole
	HasTypeRole    bool
	TypeEmphasized bool

	// RTL-aware logical padding and margins in DIPs.
	PaddingStart  float32
	PaddingEnd    float32
	PaddingTop    float32
	PaddingBottom float32
	MarginStart   float32
	MarginEnd     float32
	MarginTop     float32
	MarginBottom  float32

	// Clip enforces child clipping inside the surface corners.
	Clip bool
	// DisableShadow suppresses Key/Ambient drop shadows while keeping tonal fill.
	DisableShadow bool
}

type resolvedSurfaceShape struct {
	poly    *shape.RoundedPolygon
	radii   [4]float32
	hasMask bool
}

func resolveSurfaceShape(th *Theme, opt SurfaceOptions) resolvedSurfaceShape {
	if opt.Shape != nil {
		return resolvedSurfaceShape{poly: opt.Shape}
	}
	switch {
	case opt.HasRadii || opt.Radii != [4]float32{}:
		radii := [4]float32{
			sanitizeNonNegF32(opt.Radii[0]),
			sanitizeNonNegF32(opt.Radii[1]),
			sanitizeNonNegF32(opt.Radii[2]),
			sanitizeNonNegF32(opt.Radii[3]),
		}
		return resolvedSurfaceShape{radii: radii, hasMask: true}
	case opt.HasRadius || isPositiveFiniteF32(opt.Radius):
		r := sanitizeNonNegF32(opt.Radius)
		return resolvedSurfaceShape{radii: [4]float32{r, r, r, r}, hasMask: r > 0}
	case opt.ShapeScale != tokens.ShapeNone:
		r := th.Shapes.Radius(opt.ShapeScale)
		return resolvedSurfaceShape{radii: [4]float32{r, r, r, r}, hasMask: r > 0}
	default:
		return resolvedSurfaceShape{}
	}
}

func resolveHairlineColor(th *Theme, override ui.Color) ui.Color {
	if override.A > 0 {
		return override
	}
	cs := th.Colors
	if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
		return cs.Outline
	}
	base := cs.OutlineVariant
	if base.A == 0 {
		base = cs.Outline
	}
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

func clipSurfaceRadii(p *ui.Painter, r ui.Rect, radii [4]float32, fn func()) {
	if radii[0] == radii[1] && radii[1] == radii[2] && radii[2] == radii[3] {
		p.Clip(r, radii[0], fn)
		return
	}
	if path := roundedRectPath(r, radii); path != nil {
		p.ClipPath(path, fn)
		return
	}
	fn()
}

func resolveSurfaceBackground(th *Theme, opt SurfaceOptions) ui.Color {
	tonalLevel := opt.Elevation
	// A non-zero TonalElevation overrides the shadow elevation in either
	// direction: callers may want Level3 shadows over a Level1 tone.
	if opt.TonalElevation > tokens.Level0 {
		tonalLevel = opt.TonalElevation
	}
	if opt.Color.A == 0 {
		return SurfaceColorForElevation(th, tonalLevel)
	}
	if opt.TonalTint && tonalLevel > tokens.Level0 {
		return ApplyTonalElevation(th, opt.Color, tonalLevel)
	}
	return opt.Color
}

// Surface builds an MD3E layout container (ui.Column) with tonal surface and
// Key/Ambient elevation, uniform/independent-4-corner or RoundedPolygon shape
// rendering, optional border, accessible content color & typography inheritance,
// and RTL-aware logical spacing.
func Surface(c *ui.Context, opts ...SurfaceOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Surface takes at most one SurfaceOptions")
	}
	var opt SurfaceOptions
	if len(opts) == 1 {
		opt = opts[0]
	}

	th := CurrentTheme(c)
	bg := resolveSurfaceBackground(th, opt)
	geom := resolveSurfaceShape(th, opt)

	fg := opt.ContentColor
	if fg.A == 0 {
		fg = th.ContentColorFor(bg)
	}

	borderW := sanitizeNonNegF32(opt.BorderWidth)
	borderCol := opt.BorderColor
	if borderW > 0 && borderCol.A == 0 {
		borderCol = th.Colors.OutlineVariant
		if th.Contrast >= tokens.ContrastHigh && th.Colors.Outline.A > 0 {
			borderCol = th.Colors.Outline
		}
	} else if borderW == 0 && opt.Hairline {
		borderW = 1
		borderCol = resolveHairlineColor(th, opt.HairlineColor)
	}

	el := ui.Column(c)
	if fg.A > 0 {
		el.TextColor(fg)
	}
	if opt.HasTypeRole {
		applyTypeStyleWithTheme(th, el, opt.TypeRole, opt.TypeEmphasized)
	}

	var linGrad *ui.LinearGradient
	if opt.LinearGradient != nil {
		g := *opt.LinearGradient
		linGrad = &g
	}
	var radGrad *ui.RadialGradient
	if opt.RadialGradient != nil {
		g := *opt.RadialGradient
		radGrad = &g
	}
	backdrop := opt.Backdrop

	if geom.poly != nil {
		poly := geom.poly
		bw := borderW
		bCol := borderCol
		if opt.Clip {
			// ui clips children to boxes only; the polygon's own corners are
			// painted above, and this keeps content inside its bounds.
			el.Clip()
		}
		el.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			path := poly.ToPath(r)
			if path == nil {
				return
			}
			if !opt.DisableShadow && opt.Elevation > tokens.Level0 {
				spec := opt.Elevation.Spec()
				shadowCol := th.Colors.Shadow
				if shadowCol.A == 0 {
					shadowCol = ui.RGB(0, 0, 0)
				}
				if spec.Ambient.Opacity > 0 {
					p.Transform(ui.Translation(0, float32(spec.Ambient.OffsetY)), 0, 0, func() {
						p.FillPath(path, shadowCol.Alpha(float32(spec.Ambient.Opacity*0.45)))
					})
				}
			}
			if bg.A > 0 {
				p.FillPath(path, bg)
			}
			if linGrad != nil {
				p.FillPathGradient(path, *linGrad)
			}
			if radGrad != nil || backdrop != nil {
				p.ClipPath(path, func() {
					if radGrad != nil {
						p.FillRadialGradient(r, *radGrad, 0)
					}
					if backdrop != nil {
						backdrop(p, r)
					}
				})
			}
			if bw > 0 && bCol.A > 0 {
				p.StrokePath(path, bw, bCol)
			}
		})
	} else {
		if bg.A > 0 {
			el.Background(bg)
		}
		el.Radius(geom.radii[0], geom.radii[1], geom.radii[2], geom.radii[3])
		if linGrad != nil {
			el.LinearGradient(*linGrad)
		} else if radGrad != nil {
			el.RadialGradient(*radGrad)
		}
		if (linGrad != nil && radGrad != nil) || backdrop != nil {
			radii := geom.radii
			overlayRadial := linGrad != nil && radGrad != nil
			el.Draw(func(p *ui.Painter, r ui.Rect) {
				if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
					return
				}
				clipSurfaceRadii(p, r, radii, func() {
					if overlayRadial {
						p.FillRadialGradient(r, *radGrad, 0)
					}
					if backdrop != nil {
						backdrop(p, r)
					}
				})
			})
		}
		if borderW > 0 && borderCol.A > 0 {
			el.Border(borderW, borderCol)
		}
		if !opt.DisableShadow && opt.Elevation > tokens.Level0 {
			ApplyElevationShadows(th, el, opt.Elevation)
		}
		if opt.Clip {
			el.Clip()
		}
	}

	if opt.PaddingTop != 0 || opt.PaddingBottom != 0 {
		el.Padding(opt.PaddingTop, 0, opt.PaddingBottom, 0)
	}
	if opt.PaddingStart != 0 {
		el.PaddingStart(opt.PaddingStart)
	}
	if opt.PaddingEnd != 0 {
		el.PaddingEnd(opt.PaddingEnd)
	}
	if opt.MarginTop != 0 || opt.MarginBottom != 0 {
		el.Margin(opt.MarginTop, 0, opt.MarginBottom, 0)
	}
	if opt.MarginStart != 0 {
		el.MarginStart(opt.MarginStart)
	}
	if opt.MarginEnd != 0 {
		el.MarginEnd(opt.MarginEnd)
	}

	return el
}

// roundedRectPath constructs a closed *ui.Path for rectangle r with independent
// [TL, TR, BR, BL] corner radii clamped according to CSS border-radius proportions.
func roundedRectPath(r ui.Rect, radii [4]float32) *ui.Path {
	if r.W <= 0 || r.H <= 0 {
		return nil
	}
	tl := max(radii[0], 0)
	tr := max(radii[1], 0)
	br := max(radii[2], 0)
	bl := max(radii[3], 0)

	scale := float32(1)
	clampEdge := func(sum, limit float32) {
		if sum > 0 && limit < sum {
			if s := limit / sum; s < scale {
				scale = s
			}
		}
	}
	clampEdge(tl+tr, r.W)
	clampEdge(bl+br, r.W)
	clampEdge(tl+bl, r.H)
	clampEdge(tr+br, r.H)
	tl, tr, br, bl = tl*scale, tr*scale, br*scale, bl*scale

	// Standard cubic Bézier circular corner approximation: 1 - 4*(sqrt(2)-1)/3.
	const c = float32(1.0 - 0.5522847498)

	var p ui.Path
	p.MoveTo(r.X+tl, r.Y)
	p.LineTo(r.X+r.W-tr, r.Y)
	if tr > 0 {
		p.CubeTo(r.X+r.W-tr*c, r.Y, r.X+r.W, r.Y+tr*c, r.X+r.W, r.Y+tr)
	}
	p.LineTo(r.X+r.W, r.Y+r.H-br)
	if br > 0 {
		p.CubeTo(r.X+r.W, r.Y+r.H-br*c, r.X+r.W-br*c, r.Y+r.H, r.X+r.W-br, r.Y+r.H)
	}
	p.LineTo(r.X+bl, r.Y+r.H)
	if bl > 0 {
		p.CubeTo(r.X+bl*c, r.Y+r.H, r.X, r.Y+r.H-bl*c, r.X, r.Y+r.H-bl)
	}
	p.LineTo(r.X, r.Y+tl)
	if tl > 0 {
		p.CubeTo(r.X, r.Y+tl*c, r.X+tl*c, r.Y, r.X+tl, r.Y)
	}
	p.Close()
	return &p
}
