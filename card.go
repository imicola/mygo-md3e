package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// CardVariant selects one of the three Material Design 3 card looks.
type CardVariant int

const (
	// CardElevated paints the Level1 surface with Level1 shadows, lifting to
	// Level2 while pressed.
	CardElevated CardVariant = iota
	// CardFilled paints the SurfaceContainerHighest fill without shadows.
	CardFilled
	// CardOutlined paints the Surface fill with an OutlineVariant border.
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
	// Radius overrides the resting corner radius in DIP; the Medium shape when
	// zero or negative.
	Radius float32
}

// cardCornerKey is the AnimateTo channel of a card's corner radius.
type cardCornerKey struct{}

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
	rest := sanitizePositiveF32(o.Radius, th.Shapes.Radius(tokens.ShapeMedium))
	padding := sanitizePositiveF32(o.Padding, 16)

	var el ui.Element
	if interactive {
		el = ui.ButtonBase(c).Column().Justify(ui.Start).AlignItems(ui.Stretch).
			DimDisabled(false).FocusRing(false)
	} else {
		el = ui.Column(c).DimDisabled(false)
	}
	el.Disabled(o.Disabled)
	disabled := o.Disabled || el.IsDisabled()

	bg, border, elevation := cardPalette(th, o.Variant, disabled)
	if disabled {
		bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		border = ui.Transparent
		elevation = tokens.Level0
	}

	radius := rest
	if interactive {
		target := rest
		if !disabled && el.Pressed() {
			// MD3E lifts a pressed card and rounds its corners slightly.
			target = rest + cardPressedRadiusDelta
			if elevation > tokens.Level0 && elevation < tokens.Level5 {
				elevation++
				bg = SurfaceColorForElevation(th, elevation)
			}
		}
		radius = animateWithTheme(el, th, cardCornerKey{}, target, motion.FastSpatial)
	}

	el.Padding(padding).Radius(radius)
	if bg.A > 0 {
		el.Background(bg)
	}
	if border.A > 0 {
		el.Border(1, border)
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
		if border.A > 0 {
			borderInset = 1
		}
		attachRippleWithTheme(th, el, StateLayerOptions{
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
		bg = cs.SurfaceContainerHighest
		if bg.A == 0 {
			bg = SurfaceColorForElevation(th, tokens.Level2)
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
