package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// ButtonVariant selects one of the five Material Design 3 Expressive button
// looks. The zero value, ButtonFilled, is the high-emphasis action.
type ButtonVariant int

const (
	// ButtonFilled paints the Primary container with OnPrimary content.
	ButtonFilled ButtonVariant = iota
	// ButtonTonal paints the SecondaryContainer with OnSecondaryContainer content.
	ButtonTonal
	// ButtonElevated paints the Level1 surface with Primary content and
	// MD3E Level1 Key/Ambient shadows, lifting to Level2 while pressed.
	ButtonElevated
	// ButtonOutlined draws an Outline border around Primary content.
	ButtonOutlined
	// ButtonText draws no container, only Primary content.
	ButtonText
)

// ButtonShape selects a button's resting corner shape. Both shapes shrink
// their corners toward the size tier's pressed radius with the Fast Spatial
// spring while the button is pressed.
type ButtonShape int

const (
	// ButtonShapeRound is the full pill shape: the tier height's half.
	ButtonShapeRound ButtonShape = iota
	// ButtonShapeSquare is the tier's square radius (12/12/16/28/28).
	ButtonShapeSquare
)

// IconDraw paints an icon inside r using ink. A built-in md3e/internal/mdicon
// icon's Draw method is one, as mdicon.Check.Draw, and so is any draw function
// of the same shape for custom artwork.
type IconDraw func(p *ui.Painter, r ui.Rect, ink ui.Color)

// ButtonOptions configures Button. The zero value is a filled, extra small,
// round button with no icons that is enabled.
type ButtonOptions struct {
	// Variant selects the button's MD3E look.
	Variant ButtonVariant
	// Size selects the XS..XL size tier, which fixes the height, horizontal
	// padding, icon size, icon-label gap and typography. An out-of-range tier
	// falls back to SizeXS.
	Size tokens.SizeTier
	// Shape selects the resting corner shape.
	Shape ButtonShape
	// LeadingIcon draws an icon before the label, at the tier's icon size.
	LeadingIcon IconDraw
	// TrailingIcon draws an icon after the label, at the tier's icon size.
	TrailingIcon IconDraw
	// Disabled disables the button: it takes no clicks, keys or focus, and
	// shows MD3E's disabled colors without ui's automatic 0.5 dimming.
	Disabled bool
	// Label overrides the accessible name. By default the visible label names
	// the button; give an empty label with an icon and a Label for an
	// icon-only button.
	Label string
}

// buttonCornerKey is the AnimateTo channel of a button's corner radius.
type buttonCornerKey struct{}

// Button builds a Material Design 3 Expressive button: a ui.ButtonBase with
// the variant's container, border, elevation and content colors, the size
// tier's exact height, padding, icon geometry and typography, an MD3E
// StateLayer with Expressive Ripple, and a Fast Spatial corner morph toward
// the tier's pressed radius that springs back on release. Pressing also lifts
// the Elevated variant from Level1 to Level2 shadows.
//
// Clicked reports pointer clicks and Enter or Space while the button has the
// focus, as ButtonBase's do; assistive technology sees a button named by the
// label:
//
//	md3e.Button(c, "Save", md3e.ButtonOptions{LeadingIcon: mdicon.Check.Draw})
//	if save.Clicked() {
//		app.save()
//	}
//
// The corner morph reads Pressed in the build pass, as the radius is an
// element attribute; hover and focus feedbacks stay in DrawOver inside
// AttachRipple, so hovering never rebuilds the view.
func Button(c *ui.Context, label string, opts ...ButtonOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Button takes at most one ButtonOptions")
	}
	var o ButtonOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	spec := tierSpec(o.Size)
	pal := buttonPaletteOf(th, o.Variant, o.Disabled)
	rest := restingRadius(spec, o.Shape)

	el := ui.ButtonBase(c).
		DimDisabled(false).
		Height(float32(spec.Height)).
		PaddingX(float32(spec.Padding)).
		Gap(iconLabelGap(o.Size)).
		Disabled(o.Disabled).
		TextColor(pal.content)
	if pal.container.A > 0 {
		el.Background(pal.container)
	}
	if pal.border.A > 0 {
		el.Border(1, pal.border)
	}

	pressed := el.Pressed()
	elevation := pal.elevation
	if pressed && elevation > tokens.Level0 && elevation < tokens.Level5 {
		// MD3E lifts the Elevated variant while it is pressed.
		elevation++
	}
	if elevation > tokens.Level0 {
		ApplyElevationShadows(th, el, elevation)
	}

	target := rest
	if pressed {
		target = float32(spec.PressedRadius)
	}
	radius := el.AnimateTo(buttonCornerKey{}, target, th.Animation(motion.FastSpatial))
	el.Radius(radius)

	borderInset := float32(0)
	if o.Variant == ButtonOutlined {
		borderInset = 1
	}
	AttachRipple(c, el, StateLayerOptions{
		Color:       pal.content,
		Radius:      radius,
		BorderInset: borderInset,
		Disabled:    o.Disabled,
	})

	name := o.Label
	if name == "" {
		name = label
	}
	if name != "" {
		el.Label(name)
	}

	role := buttonTypeRole(o.Size)
	el.Children(func() {
		if o.LeadingIcon != nil {
			drawIconBox(c, o.LeadingIcon, float32(spec.IconSize), pal.content)
		}
		if label != "" {
			ApplyTypeStyle(c, ui.Text(c, label).NoWrap(), role)
		}
		if o.TrailingIcon != nil {
			drawIconBox(c, o.TrailingIcon, float32(spec.IconSize), pal.content)
		}
	})
	return el
}

// buttonColors are the resolved colors and elevation of a button variant in
// one state.
type buttonColors struct {
	container ui.Color
	content   ui.Color
	border    ui.Color
	elevation tokens.ElevationLevel
}

// buttonPaletteOf resolves the MD3E colors of variant, disabled or not, from
// th. Disabled containers and borders use OnSurface at 12% and disabled
// content uses OnSurface at 38%, as MD3E's disabled tokens, rather than ui's
// automatic dimming.
func buttonPaletteOf(th *Theme, v ButtonVariant, disabled bool) buttonColors {
	cs := th.Colors
	if disabled {
		onSurface := cs.OnSurface
		content := onSurface.Alpha(float32(tokens.DisabledContentOpacity))
		switch v {
		case ButtonOutlined:
			return buttonColors{
				content: content,
				border:  onSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
			}
		case ButtonText:
			return buttonColors{content: content}
		default:
			return buttonColors{
				container: onSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
				content:   content,
			}
		}
	}
	switch v {
	case ButtonFilled:
		return buttonColors{container: cs.Primary, content: cs.OnPrimary}
	case ButtonTonal:
		return buttonColors{container: cs.SecondaryContainer, content: cs.OnSecondaryContainer}
	case ButtonElevated:
		return buttonColors{
			container: SurfaceColorForElevation(th, tokens.Level1),
			content:   cs.Primary,
			elevation: tokens.Level1,
		}
	case ButtonOutlined:
		return buttonColors{content: cs.Primary, border: cs.Outline}
	default:
		return buttonColors{content: cs.Primary}
	}
}

// tierSpec returns the geometry of t, falling back to SizeXS for a tier
// outside SizeXS..SizeXL.
func tierSpec(t tokens.SizeTier) tokens.SizeSpec {
	if t < tokens.SizeXS || t > tokens.SizeXL {
		return tokens.SizeXS.Spec()
	}
	return t.Spec()
}

// restingRadius returns the idle corner radius of shape at spec. Round is a
// pill, half the tier height, so that the Fast Spatial morph to the pressed
// radius moves through visible intermediate corners rather than unwrapping
// the Full token's 9999.
func restingRadius(spec tokens.SizeSpec, shape ButtonShape) float32 {
	if shape == ButtonShapeSquare {
		return float32(spec.SquareRadius)
	}
	return float32(spec.Height / 2)
}

// buttonTypeRole returns the MD3E type role of the tier's label.
func buttonTypeRole(t tokens.SizeTier) tokens.TypeRole {
	switch t {
	case tokens.SizeXS, tokens.SizeS:
		return tokens.LabelLarge
	case tokens.SizeM:
		return tokens.TitleMedium
	case tokens.SizeL:
		return tokens.HeadlineSmall
	default:
		return tokens.HeadlineLarge
	}
}

// iconLabelGap returns the space between an icon and the label of a tier, in
// DIP: 8 up to the medium tier, 12 for large and 16 for extra large.
func iconLabelGap(t tokens.SizeTier) float32 {
	switch t {
	case tokens.SizeL:
		return 12
	case tokens.SizeXL:
		return 16
	default:
		return 8
	}
}

// drawIconBox adds a square icon slot of side size that paints draw in ink.
func drawIconBox(c *ui.Context, draw IconDraw, size float32, ink ui.Color) {
	ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		draw(p, r, ink)
	})
}
