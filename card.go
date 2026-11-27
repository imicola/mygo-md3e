package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// CardVariant selects one of the three Material Design 3 card looks.
type CardVariant int

const (
	// CardElevated paints SurfaceContainerLow with Level1 shadows, lifting to
	// Level2 while hovered or pressed.
	CardElevated CardVariant = iota
	// CardFilled paints SurfaceContainerLow without shadows.
	CardFilled
	// CardOutlined paints the Surface fill with an OutlineVariant border (or
	// 1.5 DIP Outline in HighContrast).
	CardOutlined
)

// CardOptions configures Card. The zero value is an elevated, padding 16 card
// that is not clickable.
type CardOptions struct {
	// Variant selects the card's container treatment.
	Variant CardVariant
	// Clickable makes the card an interactive ui.ButtonBase with ripple
	// feedback, a focus stop and a pressed corner morph. OnClick alone makes it
	// clickable too.
	Clickable bool
	// OnClick runs when a clickable card is clicked or activated by Enter or
	// Space. It makes the card clickable even without Clickable.
	OnClick func()
	// Disabled disables a clickable card: it takes no clicks or focus and shows
	// MD3E's disabled container and content tokens.
	Disabled bool
	// Label names the card for assistive technology.
	Label string
	// Padding is the content inset in DIP; 16 when zero or negative.
	Padding float32
	// ShapeScale selects an MD3E corner radius token for the card (for example
	// tokens.ShapeLargeIncreased or tokens.ShapeExtraLarge for Bento layouts).
	// The zero value (ShapeNone) defaults to tokens.ShapeLargeIncreased (20 DIP)
	// for Elevated/Filled cards and tokens.ShapeLarge (16 DIP) for Outlined
	// cards. Radius > 0 overrides ShapeScale when set.
	ShapeScale tokens.ShapeScale
	// Radius overrides the resting corner radius in DIP; takes precedence over
	// ShapeScale when positive and finite.
	Radius float32
	// Hairline draws a subtle 1 DIP tonal highlight border on elevated or filled
	// cards (ignored when disabled). On outlined cards, HairlineColor can still
	// customize the border color.
	Hairline bool
	// HairlineColor overrides the default Hairline stroke color when A > 0.
	HairlineColor ui.Color
	// Optional gradient fills applied when the card is enabled. When both are
	// set, LinearGradient paints first and RadialGradient overlays it.
	LinearGradient *ui.LinearGradient
	RadialGradient *ui.RadialGradient
	// Backdrop optionally paints clipped decorative graphics inside the card
	// bounds after the fill/gradient and before children. The callback must be
	// pure and ghost-draw safe (no animation requests or interaction queries).
	Backdrop func(p *ui.Painter, r ui.Rect)
}

// cardPressedRadiusDelta is how much a pressed card's corners grow, in DIP.
const cardPressedRadiusDelta = 4

// ElevatedCard builds a Card with the elevated container. Options other than
// the variant apply as Card's do.
func ElevatedCard(c *ui.Context, content func(), opts ...CardOptions) ui.Element {
	o := cardOptions(opts)
	o.Variant = CardElevated
	return Card(c, content, o)
}

// FilledCard builds a Card with the filled container.
func FilledCard(c *ui.Context, content func(), opts ...CardOptions) ui.Element {
	o := cardOptions(opts)
	o.Variant = CardFilled
	return Card(c, content, o)
}

// OutlinedCard builds a Card with the outlined container.
func OutlinedCard(c *ui.Context, content func(), opts ...CardOptions) ui.Element {
	o := cardOptions(opts)
	o.Variant = CardOutlined
	return Card(c, content, o)
}

// cardOptions validates and returns the single CardOptions of opts.
func cardOptions(opts []CardOptions) CardOptions {
	if len(opts) > 1 {
		panic("md3e: Card takes at most one CardOptions")
	}
	if len(opts) == 1 {
		return opts[0]
	}
	return CardOptions{}
}

// Card builds a Material Design 3 Expressive card: an elevated, filled or
// outlined container around content. A clickable card (Clickable or OnClick)
// is a ui.ButtonBase with a StateLayer and Expressive Ripple, and its corners
// and elevation change with the Fast Spatial spring while it is pressed. A
// plain card is a non-focusable container with no feedback. Both support
// Disabled, and a disabled ancestor disables a clickable card too.
//
// Hover feedback stays in the ripple's DrawOver pass, so hovering never
// rebuilds the view; only the press, which changes element attributes, is read
// in the build pass.
//
//	md3e.Card(c, func() {
//		md3e.Text(c, "Photos", tokens.TitleMedium)
//	}, md3e.CardOptions{Variant: md3e.CardFilled, OnClick: app.openPhotos})
func Card(c *ui.Context, content func(), opts ...CardOptions) ui.Element {
	o := cardOptions(opts)
	th := CurrentTheme(c)
	cs := th.Colors

	interactive := o.Clickable || o.OnClick != nil
	baseScale := tokens.ShapeLargeIncreased
	if o.Variant == CardOutlined {
		baseScale = tokens.ShapeLarge
	}
	if o.ShapeScale > tokens.ShapeNone && o.ShapeScale <= tokens.ShapeFull {
		baseScale = o.ShapeScale
	}
	rest := sanitizePositiveF32(o.Radius, th.Shapes.Radius(baseScale))
	padding := sanitizePositiveF32(o.Padding, 16)

	var el ui.Element
	if interactive {
		el = ui.ButtonBase(c).Column().Justify(ui.Start).AlignItems(ui.Stretch).
			MinWidth(0).Clip().DimDisabled(false).FocusRing(false)
	} else {
		// Match the interactive card: stretch the content to the card's inner
		// width, allow the card to shrink below its content's min-content width
		// in narrow grid tracks, and clip rows that still overflow so they
		// cannot paint past the border. Clip only wraps the children; the
		// card's own shadow, background and ripple stay unclipped.
		el = ui.Column(c).AlignItems(ui.Stretch).MinWidth(0).Clip().DimDisabled(false)
	}
	el.Disabled(o.Disabled)
	disabled := o.Disabled || el.IsDisabled()

	bg, border, elevation := cardPalette(th, o.Variant, disabled)
	borderW := float32(0)
	if !disabled {
		if o.Variant == CardOutlined {
			borderW = outlineBorderWidth(th)
		}
		if o.Hairline {
			if border.A == 0 || o.HairlineColor.A > 0 {
				border = resolveHairlineColor(th, o.HairlineColor)
			}
			borderW = outlineBorderWidth(th)
		} else if border.A == 0 {
			if hcW, hcCol := highContrastBorder(th); hcW > 0 {
				borderW = hcW
				border = hcCol
			}
		}
	}
	if disabled {
		if o.Variant == CardOutlined {
			bg = cs.Surface
			border = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
			borderW = 1
		} else {
			bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
			border = ui.Transparent
			borderW = 0
		}
		elevation = tokens.Level0
	}

	pressed := false
	radius := rest
	if interactive {
		target := rest
		pressed = !disabled && el.Pressed()
		if pressed {
			// MD3E lifts a pressed card and rounds its corners slightly.
			target = rest + cardPressedRadiusDelta
			if elevation > tokens.Level0 && elevation < tokens.Level5 {
				elevation++
				bg = SurfaceColorForElevation(th, elevation)
			}
		}
		radius = max(animateWithTheme(el, th, animChan0, target, motion.FastSpatial), 0)
	}

	el.Padding(padding).Radius(radius)
	if bg.A > 0 {
		el.Background(bg)
	}
	var linGrad *ui.LinearGradient
	var radGrad *ui.RadialGradient
	if !disabled {
		if o.LinearGradient != nil {
			g := *o.LinearGradient
			linGrad = &g
		}
		if o.RadialGradient != nil {
			g := *o.RadialGradient
			radGrad = &g
		}
		if linGrad != nil {
			el.LinearGradient(*linGrad)
		} else if radGrad != nil {
			el.RadialGradient(*radGrad)
		}
	}
	hasCustomDraw := (linGrad != nil && radGrad != nil) || o.Backdrop != nil
	wantHoverElev := interactive && !disabled && !pressed && o.Variant == CardElevated
	wantSpecular := !disabled && hasTopSpecularReflection(th, elevation)
	if hasCustomDraw {
		drawRadius := radius
		backdrop := o.Backdrop
		overlayRadial := linGrad != nil && radGrad != nil
		var hoverSt *haloFadeState
		var baseSpec, hoverSpec tokens.ElevationSpec
		var shadowCol ui.Color
		thReduce := th.ReduceMotion
		if wantHoverElev {
			hoverSt = localHaloFade(el, animChan7)
			baseSpec = tokens.Level1.Spec()
			hoverSpec = tokens.Level2.Spec()
			shadowCol = cs.Shadow
			if shadowCol.A == 0 {
				shadowCol = ui.RGB(0, 0, 0)
			}
		}
		el.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			p.Clip(r, drawRadius, func() {
				if overlayRadial {
					p.FillRadialGradient(r, *radGrad, 0)
				}
				if backdrop != nil {
					backdrop(p, r)
				}
				if wantSpecular {
					paintTopSpecularStrip(p, r)
				}
			})
			if hoverSt != nil && el.Valid() && !el.IsDisabled() && rectIntersects(r, p.Visible()) {
				target := float32(0)
				if el.Hovered() && !el.Pressed() {
					target = 1
				}
				t, needFrame := hoverSt.step(target, p.Now(), thReduce || p.Preferences().ReduceMotion)
				if needFrame {
					p.After(16 * time.Millisecond)
				}
				if t > 0.005 {
					paintHoverElevationDelta(p, r, drawRadius, shadowCol, baseSpec, hoverSpec, t)
				}
			}
		})
	} else if wantHoverElev || wantSpecular {
		baseL, hoverL := tokens.Level0, tokens.Level0
		if wantHoverElev {
			baseL, hoverL = tokens.Level1, tokens.Level2
		}
		specL := tokens.Level0
		if wantSpecular {
			specL = elevation
		}
		attachElevatedSurfaceEffects(el, th, baseL, hoverL, radius, disabled, specL)
	}
	if borderW > 0 && border.A > 0 {
		el.Border(borderW, border)
	}
	if elevation > tokens.Level0 {
		ApplyElevationShadows(th, el, elevation)
	}

	ink := cs.OnSurface
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	el.TextColor(ink)
	if interactive {
		borderInset := float32(0)
		if borderW > 0 && border.A > 0 {
			borderInset = borderW
		}
		attachRippleWithContext(c, th, el, StateLayerOptions{
			Color:       ink,
			Radius:      radius,
			BorderInset: borderInset,
			Disabled:    disabled,
		})
	}
	if o.Label != "" {
		el.Label(o.Label)
	}

	clicked := interactive && el.Clicked()
	if content != nil {
		el.Children(content)
	}
	if clicked && o.OnClick != nil && !disabled {
		o.OnClick()
	}
	return el
}

// cardPalette resolves a card variant's container, border and shadow level.
func cardPalette(th *Theme, v CardVariant, disabled bool) (bg, border ui.Color, elevation tokens.ElevationLevel) {
	cs := th.Colors
	if disabled {
		return cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)), ui.Transparent, tokens.Level0
	}
	switch v {
	case CardFilled:
		bg = cs.SurfaceContainerLow
		if bg.A == 0 {
			bg = SurfaceColorForElevation(th, tokens.Level1)
		}
	case CardOutlined:
		bg = cs.Surface
		border = cs.OutlineVariant
		if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
			border = cs.Outline
		}
	default:
		bg = SurfaceColorForElevation(th, tokens.Level1)
		elevation = tokens.Level1
	}
	return bg, border, elevation
}
