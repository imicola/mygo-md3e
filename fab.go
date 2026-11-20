package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// FABSize selects one of the three Material Design 3 Expressive floating
// action button sizes.
type FABSize int

const (
	// FABSizeSmall is the 40×40 button with a 24 DIP icon and 12 DIP corners.
	FABSizeSmall FABSize = iota
	// FABSizeMedium is the standard 56×56 button with a 24 DIP icon and 16
	// DIP corners.
	FABSizeMedium
	// FABSizeLarge is the 96×96 button with a 36 DIP icon and 28 DIP corners.
	FABSizeLarge
)

// FABColorScheme selects the container/content pairing of a floating action
// button.
type FABColorScheme int

const (
	// FABPrimary paints PrimaryContainer with OnPrimaryContainer content.
	FABPrimary FABColorScheme = iota
	// FABSecondary paints SecondaryContainer with OnSecondaryContainer
	// content.
	FABSecondary
	// FABTertiary paints TertiaryContainer with OnTertiaryContainer content.
	FABTertiary
	// FABSurface keeps the Surface container with Primary content, lifted by
	// its Level3 shadows.
	FABSurface
)

// FABOptions configures FAB. The zero value is a primary container, medium,
// enabled button.
type FABOptions struct {
	// Size selects the Small, Medium or Large geometry. A size outside the
	// three falls back to FABSizeMedium.
	Size FABSize
	// Color selects the container/content pairing.
	Color FABColorScheme
	// Disabled disables the button with MD3E's disabled tokens rather than
	// ui's automatic dimming.
	Disabled bool
	// Visible, when set, hides the button while false: it springs down to a
	// smaller scale and fades out without leaving the layout.
	Visible *bool
}

// fabSpec is the exact geometry of one FAB size, in DIP.
type fabSpec struct {
	Side          float32
	IconSize      float32
	Radius        float32
	PressedRadius float32
}

// fabGeometry returns the geometry of size, falling back to FABSizeMedium.
func fabGeometry(size FABSize) fabSpec {
	switch size {
	case FABSizeSmall:
		return fabSpec{Side: 40, IconSize: 24, Radius: 12, PressedRadius: 8}
	case FABSizeLarge:
		return fabSpec{Side: 96, IconSize: 36, Radius: 28, PressedRadius: 20}
	default:
		return fabSpec{Side: 56, IconSize: 24, Radius: 16, PressedRadius: 12}
	}
}

// fabPaletteOf resolves the container/content colors of a FAB scheme. Disabled
// buttons use OnSurface at 12% and 38% as MD3E's disabled tokens.
func fabPaletteOf(th *Theme, scheme FABColorScheme, disabled bool) buttonColors {
	cs := th.Colors
	if disabled {
		onSurface := cs.OnSurface
		return buttonColors{
			container: onSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
			content:   onSurface.Alpha(float32(tokens.DisabledContentOpacity)),
		}
	}
	switch scheme {
	case FABSecondary:
		return buttonColors{container: cs.SecondaryContainer, content: cs.OnSecondaryContainer}
	case FABTertiary:
		return buttonColors{container: cs.TertiaryContainer, content: cs.OnTertiaryContainer}
	case FABSurface:
		return buttonColors{container: cs.Surface, content: cs.Primary}
	default:
		return buttonColors{container: cs.PrimaryContainer, content: cs.OnPrimaryContainer}
	}
}

// fabCornerKey is the AnimateTo channel of a FAB's corner radius.
type fabCornerKey struct{}

// fabScaleKey is the AnimateTo channel of a FAB's hidden/visible scale.
type fabScaleKey struct{}

// fabFadeKey is the AnimateTo channel of a FAB's hidden/visible opacity.
type fabFadeKey struct{}

// FAB builds a Material Design 3 Expressive floating action button: a
// ui.ButtonBase of exact Small (40×40), Medium (56×56) or Large (96×96) size,
// with the scheme's container and content colors, Level3 Key/Ambient shadows
// that rise to Level4 while pressed, an MD3E StateLayer with Expressive
// Ripple, and a Fast Spatial corner morph toward its pressed radius.
//
// label is required and becomes the accessible name, since the icon alone
// names nothing to assistive technology; FAB panics for an empty one. The
// optional Visible pointer hides the button with a scale and fade when false:
//
//	md3e.FAB(c, "Compose", mdicon.Add.Draw, md3e.FABOptions{Color: md3e.FABTertiary})
//	if compose.Clicked() {
//		app.startCompose()
//	}
func FAB(c *ui.Context, label string, icon IconDraw, opts ...FABOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: FAB takes at most one FABOptions")
	}
	if label == "" {
		panic("md3e: FAB requires an accessibility label")
	}
	var o FABOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	spec := fabGeometry(o.Size)
	pal := fabPaletteOf(th, o.Color, o.Disabled)
	visible := o.Visible == nil || *o.Visible

	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Size(spec.Side, spec.Side).
		Label(label).
		Disabled(o.Disabled || !visible).
		TextColor(pal.content)
	if pal.container.A > 0 {
		el.Background(pal.container)
	}

	elevation := tokens.Level3
	if el.Pressed() {
		// MD3E lifts a pressed FAB one elevation step.
		elevation = tokens.Level4
	}
	if !o.Disabled && visible {
		ApplyElevationShadows(th, el, elevation)
	}

	radius := spec.Radius
	if el.Pressed() {
		radius = spec.PressedRadius
	}
	corner := el.AnimateTo(fabCornerKey{}, radius, th.Animation(motion.FastSpatial))
	el.Radius(corner)

	AttachRipple(c, el, StateLayerOptions{
		Color:    pal.content,
		Radius:   corner,
		Disabled: o.Disabled || !visible,
	})

	if o.Visible != nil {
		scaleTarget, fadeTarget := float32(1), float32(1)
		if !visible {
			scaleTarget, fadeTarget = 0.6, 0
		}
		scale := el.AnimateTo(fabScaleKey{}, scaleTarget, th.Animation(motion.DefaultSpatial))
		fade := el.AnimateTo(fabFadeKey{}, fadeTarget, th.Animation(motion.FastEffects))
		el.Scale(scale, scale).Opacity(fade)
	}

	el.Children(func() {
		if icon != nil {
			drawIconBox(c, icon, spec.IconSize, pal.content)
		}
	})
	return el
}

// ExtendedFABOptions configures ExtendedFAB. The zero value is a primary
// container, medium, enabled, expanded button.
type ExtendedFABOptions struct {
	// Size selects the Medium (56 DIP) or Large (96 DIP) height; other sizes
	// fall back to Medium.
	Size FABSize
	// Color selects the container/content pairing.
	Color FABColorScheme
	// Disabled disables the button with MD3E's disabled tokens.
	Disabled bool
	// Expanded, when set, collapses the label while false: the button springs
	// to its square, icon-only width and fades the text out. Nil keeps it
	// expanded.
	Expanded *bool
	// Label overrides the accessible name; the visible label names it by
	// default.
	Label string
}

// extFabOpenKey is the AnimateTo channel of an extended FAB's expansion.
type extFabOpenKey struct{}

// extFabFadeKey is the AnimateTo channel of an extended FAB label's opacity.
type extFabFadeKey struct{}

// ExtendedFAB builds a Material Design 3 Expressive extended floating action
// button: a ui.ButtonBase 56 DIP high (or 96 for the large size) carrying an
// icon and a label, with the scheme's container and content colors, Level3
// shadows lifting to Level4 while pressed, StateLayer and Ripple, and a Full
// corner radius.
//
// A non-nil Expanded pointer folds the label: the button springs from its
// square, icon-only width to its full width and the text fades with it, as an
// app makes room around a scrolling list:
//
//	md3e.ExtendedFAB(c, "New message", mdicon.Edit.Draw,
//		md3e.ExtendedFABOptions{Expanded: &app.composeExpanded})
func ExtendedFAB(c *ui.Context, label string, icon IconDraw, opts ...ExtendedFABOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: ExtendedFAB takes at most one ExtendedFABOptions")
	}
	if label == "" {
		panic("md3e: ExtendedFAB requires a label")
	}
	var o ExtendedFABOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	spec := fabGeometry(o.Size)
	if o.Size != FABSizeLarge {
		spec = fabGeometry(FABSizeMedium)
	}
	pal := fabPaletteOf(th, o.Color, o.Disabled)
	expanded := o.Expanded == nil || *o.Expanded

	role := tokens.TitleMedium
	if o.Size == FABSizeLarge {
		role = tokens.HeadlineSmall
	}
	style := th.TypeStyle(role, false)
	scale := th.EffectiveTextScale()
	pad := (spec.Side - spec.IconSize) / 2
	gap := float32(12)
	if o.Size == FABSizeLarge {
		gap = 16
	}
	textW, _ := c.MeasureText(0, ui.Span{
		Text:   label,
		Font:   th.Font,
		Size:   float32(style.Size) * scale,
		Weight: style.Weight,
	})

	progress := float32(0)
	if expanded {
		progress = 1
	}
	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Height(spec.Side).
		Clip().
		Label(labelOr(o.Label, label)).
		Disabled(o.Disabled).
		TextColor(pal.content)
	if pal.container.A > 0 {
		el.Background(pal.container)
	}
	if !o.Disabled {
		elevation := tokens.Level3
		if el.Pressed() {
			elevation = tokens.Level4
		}
		ApplyElevationShadows(th, el, elevation)
	}

	p := el.AnimateTo(extFabOpenKey{}, progress, th.Animation(motion.DefaultSpatial))
	radius := el.AnimateTo(extFabRadiusKey{}, spec.Side/2, th.Animation(motion.FastSpatial))
	el.Radius(radius)

	// The width springs between the square icon-only button and the full
	// icon-label pill; the row clips the label while it collapses.
	width := 2*pad + spec.IconSize + p*(gap+textW)
	el.Width(width).PaddingX(pad)
	labelFade := float32(1)
	if o.Expanded != nil {
		labelFade = el.AnimateTo(extFabFadeKey{}, progress, th.Animation(motion.FastEffects))
	}

	AttachRipple(c, el, StateLayerOptions{
		Color:    pal.content,
		Radius:   radius,
		Disabled: o.Disabled,
	})

	el.Children(func() {
		if icon != nil {
			drawIconBox(c, icon, spec.IconSize, pal.content)
		}
		name := ApplyTypeStyle(c, ui.Text(c, label).NoWrap().Shrink(0), role)
		if o.Expanded != nil {
			name.Opacity(labelFade)
		}
	})
	return el
}

// extFabRadiusKey is the AnimateTo channel of an extended FAB's corner radius.
type extFabRadiusKey struct{}

// labelOr returns s when set, fallback otherwise.
func labelOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
