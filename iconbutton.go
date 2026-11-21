package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// IconButtonVariant selects the MD3E icon button look of IconButton and
// ToggleIconButton.
type IconButtonVariant int

const (
	// IconButtonStandard draws no container in its resting state, with an
	// OnSurfaceVariant icon; a toggle's selected state keeps that plain
	// container and turns the icon Primary.
	IconButtonStandard IconButtonVariant = iota
	// IconButtonFilled paints Primary with an OnPrimary icon; a toggle's
	// resting state paints SurfaceContainerHighest instead.
	IconButtonFilled
	// IconButtonTonal paints SecondaryContainer with an OnSecondaryContainer
	// icon; a toggle's resting state paints SurfaceContainerHighest.
	IconButtonTonal
	// IconButtonOutlined draws an Outline border; a toggle's selected state
	// paints InverseSurface with an InverseOnSurface icon.
	IconButtonOutlined
)

// IconButtonWidth selects the container width of an icon button around its
// size tier's height.
type IconButtonWidth int

const (
	// IconButtonWidthDefault is as wide as the tier is high: a circle for the
	// round shape.
	IconButtonWidthDefault IconButtonWidth = iota
	// IconButtonWidthNarrow is 0.75 of the tier height.
	IconButtonWidthNarrow
	// IconButtonWidthWide is 1.25 of the tier height.
	IconButtonWidthWide
)

// IconButtonOptions configures IconButton and ToggleIconButton. The zero value
// is a standard, extra small, default width, round icon button.
type IconButtonOptions struct {
	// Variant selects the MD3E look.
	Variant IconButtonVariant
	// Size selects the XS..XL tier: height, icon size and pressed radius. An
	// out-of-range tier falls back to SizeXS.
	Size tokens.SizeTier
	// Width selects Narrow, Default or Wide.
	Width IconButtonWidth
	// Shape selects the resting corner shape. A round icon button of the
	// default width is a circle; narrow and wide ones are pills.
	Shape ButtonShape
	// Disabled disables the button and dims nothing: MD3E disabled colors are
	// applied directly.
	Disabled bool
}

// ToggleIconButtonOptions configures ToggleIconButton.
type ToggleIconButtonOptions struct {
	// Variant selects the MD3E look.
	Variant IconButtonVariant
	// Size selects the XS..XL tier.
	Size tokens.SizeTier
	// Width selects Narrow, Default or Wide.
	Width IconButtonWidth
	// SelectedIcon draws in place of the icon while *on is true, as a filled
	// star replacing an outline one.
	SelectedIcon IconDraw
	// Disabled disables the toggle and dims nothing.
	Disabled bool
}

// iconButtonCornerKey is the AnimateTo channel of an icon button's corner.
type iconButtonCornerKey struct{}

// toggleIconCornerKey is the AnimateTo channel of a toggle icon button's
// corner.
type toggleIconCornerKey struct{}

// IconButton builds a Material Design 3 Expressive icon button: a
// ui.ButtonBase of an exact tier height and Narrow/Default/Wide width with a
// single centered icon, MD3E variant colors, a StateLayer with Expressive
// Ripple, and a Fast Spatial corner morph toward the tier's pressed radius.
//
// label is required and becomes the accessible name, since the icon alone
// names nothing to assistive technology; IconButton panics for an empty one. A
// narrow or wide round button is a pill, its ends as round as its half width.
// The MD3E Focus state layer marks keyboard focus; ui's default focus ring is
// turned off so it cannot overdraw the morphing corner.
//
//	md3e.IconButton(c, "Search", mdicon.Search.Draw, md3e.IconButtonOptions{Size: tokens.SizeS})
//	if search.Clicked() {
//		app.openSearch()
//	}
func IconButton(c *ui.Context, label string, icon IconDraw, opts ...IconButtonOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: IconButton takes at most one IconButtonOptions")
	}
	if label == "" {
		panic("md3e: IconButton requires an accessibility label")
	}
	var o IconButtonOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	o.Size = normalizeSizeTier(o.Size)
	spec := tierSpec(o.Size)
	w := iconWidthOf(o.Size, o.Width)
	rest := actionRadius(w, spec, o.Shape, false)

	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Size(w, float32(spec.Height)).
		Label(label)
	// A disabled ancestor also disables the button: use MD3E's disabled palette.
	disabled := o.Disabled || el.IsDisabled()
	pal := iconButtonPaletteOf(th, o.Variant, disabled)
	el.Disabled(o.Disabled).TextColor(pal.content)
	if pal.container.A > 0 {
		el.Background(pal.container)
	}
	if pal.border.A > 0 {
		el.Border(1, pal.border)
	}

	target := rest
	if el.Pressed() {
		target = float32(spec.PressedRadius)
	}
	radius := animateWithTheme(el, th, iconButtonCornerKey{}, target, motion.FastSpatial)
	el.Radius(radius)

	borderInset := float32(0)
	if pal.border.A > 0 {
		borderInset = 1
	}
	attachRippleWithContext(c, th, el, StateLayerOptions{
		Color:       pal.content,
		Radius:      radius,
		BorderInset: borderInset,
		Disabled:    disabled,
	})

	el.Children(func() {
		drawIconBox(c, icon, float32(spec.IconSize), pal.content)
	})
	return el
}

// ToggleIconButton builds a Material Design 3 Expressive toggle icon button: a
// ui.ToggleBase that flips *on when clicked, or with Space while it has the
// focus, which Changed reports. Its unselected shape is the round one and its
// selected shape is the tier's square radius; the corner morphs between them
// with the Fast Spatial spring, while pressed it shrinks toward the tier's
// pressed radius, and its container and icon colors follow the variant and the
// state. The optional SelectedIcon replaces the icon while it is on.
//
// label is required and becomes the accessible name; assistive technology sees
// a toggle button pressed while *on. As with ui.ToggleBase, a toggle whose
// identity changes with the data needs its persistent key before it is built,
// so that its state and focus follow it:
//
//	md3e.ToggleIconButton(c.Key("favorite"), &app.favorite, "Favorite", mdicon.Star.Draw,
//		md3e.ToggleIconButtonOptions{SelectedIcon: mdicon.Star.Draw})
func ToggleIconButton(c *ui.Context, on *bool, label string, icon IconDraw, opts ...ToggleIconButtonOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: ToggleIconButton takes at most one ToggleIconButtonOptions")
	}
	if on == nil {
		panic("md3e: ToggleIconButton requires a non-nil *bool")
	}
	if label == "" {
		panic("md3e: ToggleIconButton requires an accessibility label")
	}
	var o ToggleIconButtonOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	o.Size = normalizeSizeTier(o.Size)
	spec := tierSpec(o.Size)
	w := iconWidthOf(o.Size, o.Width)
	// The base is the element that takes the key: build it first.
	el := ui.ToggleBase(c, on).
		DimDisabled(false).
		FocusRing(false).
		Size(w, float32(spec.Height)).
		Label(label)
	disabled := o.Disabled || el.IsDisabled()
	pal := toggleIconButtonPaletteOf(th, o.Variant, *on, disabled)
	el.Disabled(o.Disabled).TextColor(pal.content)
	if pal.container.A > 0 {
		el.Background(pal.container)
	}
	if pal.border.A > 0 {
		el.Border(1, pal.border)
	}

	rest := actionRadius(w, spec, ButtonShapeRound, *on)
	target := rest
	if el.Pressed() {
		target = float32(spec.PressedRadius)
	}
	radius := animateWithTheme(el, th, toggleIconCornerKey{}, target, motion.FastSpatial)
	el.Radius(radius)

	borderInset := float32(0)
	if pal.border.A > 0 {
		borderInset = 1
	}
	attachRippleWithContext(c, th, el, StateLayerOptions{
		Color:       pal.content,
		Radius:      radius,
		BorderInset: borderInset,
		Disabled:    disabled,
	})

	draw := icon
	if *on && o.SelectedIcon != nil {
		draw = o.SelectedIcon
	}
	el.Children(func() {
		drawIconBox(c, draw, float32(spec.IconSize), pal.content)
	})
	return el
}

// iconWidthOf returns the container width of an icon button: the tier height
// scaled by 0.75, 1 or 1.25 for the narrow, default and wide modes.
func iconWidthOf(t tokens.SizeTier, w IconButtonWidth) float32 {
	h := float32(tierSpec(t).Height)
	switch w {
	case IconButtonWidthNarrow:
		return h * 0.75
	case IconButtonWidthWide:
		return h * 1.25
	default:
		return h
	}
}

// actionRadius returns the resting corner radius of an action container w by
// spec: a pill of half its shorter side for the round shape, the tier's square
// radius for the square one, and the selected square radius for a toggle that
// is on.
func actionRadius(w float32, spec tokens.SizeSpec, shape ButtonShape, on bool) float32 {
	if shape == ButtonShapeSquare || on {
		return float32(spec.SquareRadius)
	}
	return min(w, float32(spec.Height)) / 2
}

// iconButtonPaletteOf resolves the colors of a plain icon button.
func iconButtonPaletteOf(th *Theme, v IconButtonVariant, disabled bool) buttonColors {
	cs := th.Colors
	if disabled {
		onSurface := cs.OnSurface
		content := onSurface.Alpha(float32(tokens.DisabledContentOpacity))
		switch v {
		case IconButtonOutlined:
			return buttonColors{content: content, border: onSurface.Alpha(float32(tokens.DisabledContainerOpacity))}
		case IconButtonStandard:
			return buttonColors{content: content}
		default:
			return buttonColors{
				container: onSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
				content:   content,
			}
		}
	}
	switch v {
	case IconButtonFilled:
		return buttonColors{container: cs.Primary, content: cs.OnPrimary}
	case IconButtonTonal:
		return buttonColors{container: cs.SecondaryContainer, content: cs.OnSecondaryContainer}
	case IconButtonOutlined:
		return buttonColors{content: cs.OnSurfaceVariant, border: cs.Outline}
	default:
		return buttonColors{content: cs.OnSurfaceVariant}
	}
}

// toggleIconButtonPaletteOf resolves the colors of a toggle icon button in its
// current state.
func toggleIconButtonPaletteOf(th *Theme, v IconButtonVariant, on, disabled bool) buttonColors {
	if disabled {
		return iconButtonPaletteOf(th, v, true)
	}
	cs := th.Colors
	if on {
		switch v {
		case IconButtonFilled:
			return buttonColors{container: cs.Primary, content: cs.OnPrimary}
		case IconButtonTonal:
			return buttonColors{container: cs.SecondaryContainer, content: cs.OnSecondaryContainer}
		case IconButtonOutlined:
			return buttonColors{container: cs.InverseSurface, content: cs.InverseOnSurface}
		default:
			// Standard keeps its plain container and only turns the icon
			// Primary, as MD3E specifies; the Tonal selected fill belongs to
			// IconButtonTonal alone.
			return buttonColors{content: cs.Primary}
		}
	}
	switch v {
	case IconButtonFilled, IconButtonTonal:
		return buttonColors{
			container: cs.SurfaceContainerHighest,
			content:   cs.OnSurfaceVariant,
		}
	case IconButtonOutlined:
		return buttonColors{content: cs.OnSurfaceVariant, border: cs.Outline}
	default:
		return buttonColors{content: cs.OnSurfaceVariant}
	}
}
