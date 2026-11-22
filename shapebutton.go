package md3e

import (
	"slices"
	"sync"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// ShapeButtonVariant selects the Material Design 3 Expressive color treatment
// of ShapeButton and MorphIconButton. The zero value, ShapeButtonFilled, fills
// the polygon with Primary (or SurfaceContainerHighest when unselected on a
// MorphIconButton).
type ShapeButtonVariant int

const (
	// ShapeButtonFilled paints Primary with OnPrimary content (or
	// SurfaceContainerHighest with OnSurfaceVariant when unselected on a
	// MorphIconButton).
	ShapeButtonFilled ShapeButtonVariant = iota
	// ShapeButtonTonal paints SecondaryContainer with OnSecondaryContainer
	// content (or SurfaceContainerHighest when unselected on a MorphIconButton).
	ShapeButtonTonal
	// ShapeButtonOutlined draws an Outline stroke around the polygon; a
	// selected MorphIconButton switches to InverseSurface / InverseOnSurface.
	ShapeButtonOutlined
	// ShapeButtonStandard draws no resting fill, revealing the morphed polygon
	// through the MD3E StateLayer and Ripple; a selected MorphIconButton tints
	// its icon Primary.
	ShapeButtonStandard
)

// ShapeButtonOptions configures ShapeButton. The zero value is a filled,
// SizeXS, circular polygon button that morphs toward a rounded square while
// pressed.
type ShapeButtonOptions struct {
	// Variant selects the MD3E color role treatment (default ShapeButtonFilled).
	Variant ShapeButtonVariant
	// Size selects the XS..XL tier scaled by Theme.Density.ButtonSpec. An
	// out-of-range tier falls back to tokens.SizeXS.
	Size tokens.SizeTier
	// Width optionally overrides the container width in DIP. Defaults to the
	// density-scaled tier height (square bounding box).
	Width float32
	// Height optionally overrides the container height in DIP. Defaults to the
	// density-scaled tier height.
	Height float32
	// IconSize optionally overrides the centered icon size in DIP. Defaults to
	// the density-scaled tier icon size.
	IconSize float32
	// Shape is the resting polygon outline (for example shape.Cookie9Sided(),
	// shape.SoftBurst(), shape.Sunny(), or shape.Circle()). Defaults to
	// shape.Circle() when nil.
	Shape *shape.RoundedPolygon
	// PressedShape is the polygon morphed toward while pressed. When nil, a
	// custom Shape morphs toward shape.Circle() (a softer pressed silhouette),
	// and a nil/default circular Shape morphs toward shape.Square().
	PressedShape *shape.RoundedPolygon
	// Selected activates the controlled selected visual state, morphing from
	// Shape toward SelectedShape and cross-fading to SelectedIcon when set.
	Selected bool
	// SelectedShape is the polygon morphed toward when Selected is true.
	// Defaults to shape.Square() when nil.
	SelectedShape *shape.RoundedPolygon
	// SelectedIcon optionally cross-fades and scales in when Selected is true.
	SelectedIcon IconDraw
	// Spring selects the MD3E spring token driving shape morphs. Zero value
	// uses motion.FastSpatial.
	Spring motion.Token
	// ContainerColor optionally overrides the container fill color when enabled.
	ContainerColor ui.Color
	// ContentColor optionally overrides the icon ink color when enabled.
	ContentColor ui.Color
	// BorderColor optionally overrides the Outlined stroke color when enabled.
	BorderColor ui.Color
	// OnClick runs when the button is clicked or activated via Enter or Space.
	OnClick func()
	// Disabled disables interaction and applies MD3E disabled color tokens
	// without ui's automatic opacity dimming.
	Disabled bool
}

// MorphIconButtonOptions configures MorphIconButton, a *bool-bound expressive
// toggle button that continuously morphs between Shape (off) and SelectedShape
// (on) and cross-fades between icon and SelectedIcon (for example Play <-> Pause).
type MorphIconButtonOptions struct {
	// Variant selects the MD3E color role treatment (default ShapeButtonFilled).
	Variant ShapeButtonVariant
	// Size selects the XS..XL tier scaled by Theme.Density.ButtonSpec.
	Size tokens.SizeTier
	// Width optionally overrides the container width in DIP.
	Width float32
	// Height optionally overrides the container height in DIP.
	Height float32
	// IconSize optionally overrides the centered icon size in DIP.
	IconSize float32
	// Shape is the unselected polygon outline. Defaults to shape.Circle().
	Shape *shape.RoundedPolygon
	// SelectedShape is the selected polygon outline. Defaults to shape.Square().
	SelectedShape *shape.RoundedPolygon
	// PressedShape optionally overrides the polygon morphed toward while
	// actively pressed. When nil, pressing nudges along the Shape <->
	// SelectedShape morph so release flows seamlessly into the toggled shape.
	PressedShape *shape.RoundedPolygon
	// SelectedIcon optionally cross-fades and scales in while *on is true.
	SelectedIcon IconDraw
	// Spring selects the MD3E spring token driving shape morphs. Zero value
	// uses motion.FastSpatial.
	Spring motion.Token
	// ContainerColor optionally overrides the unselected container fill color.
	ContainerColor ui.Color
	// ContentColor optionally overrides the unselected icon ink color.
	ContentColor ui.Color
	// SelectedContainerColor optionally overrides the selected container fill.
	SelectedContainerColor ui.Color
	// SelectedContentColor optionally overrides the selected icon ink color.
	SelectedContentColor ui.Color
	// BorderColor optionally overrides the Outlined stroke color.
	BorderColor ui.Color
	// OnChanged runs immediately after a click or Enter/Space toggles *on.
	OnChanged func(on bool)
	// Disabled disables interaction and applies MD3E disabled color tokens.
	Disabled bool
}

// Lazy default polygons initialized via zero-value sync.Once so binaries that
// omit ShapeButton / MorphIconButton never link catalog polygons.
var (
	shapeBtnCircleOnce sync.Once
	shapeBtnCirclePoly *shape.RoundedPolygon
	shapeBtnSquareOnce sync.Once
	shapeBtnSquarePoly *shape.RoundedPolygon
)

func defaultShapeButtonCircle() *shape.RoundedPolygon {
	shapeBtnCircleOnce.Do(func() {
		shapeBtnCirclePoly = shape.Circle()
	})
	return shapeBtnCirclePoly
}

func defaultShapeButtonSquare() *shape.RoundedPolygon {
	shapeBtnSquareOnce.Do(func() {
		shapeBtnSquarePoly = shape.Square()
	})
	return shapeBtnSquarePoly
}

type (
	shapeBtnMorphCacheKey struct{}
	shapeBtnMorphTKey     struct{}
	shapeBtnPressTKey     struct{}
	shapeBtnIconTKey      struct{}
)

type shapeBtnMorphCache struct {
	rest          *shape.RoundedPolygon
	selected      *shape.RoundedPolygon
	pressed       *shape.RoundedPolygon
	selMorph      *shape.Morph
	pressMorph    *shape.Morph
	selPressMorph *shape.Morph
	fallbackPath  ui.Path
}

func samePolygon(a, b *shape.RoundedPolygon) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Center() == b.Center() && slices.Equal(a.Cubics(), b.Cubics())
}

func validShapePolygon(poly, fallback *shape.RoundedPolygon) *shape.RoundedPolygon {
	if poly != nil && len(poly.Cubics()) >= 2 {
		return poly
	}
	return fallback
}

func (c *shapeBtnMorphCache) ensure(rest, selected, pressed *shape.RoundedPolygon) {
	restChanged := !samePolygon(c.rest, rest)
	selChanged := !samePolygon(c.selected, selected)
	pressChanged := !samePolygon(c.pressed, pressed)

	if restChanged || selChanged || c.selMorph == nil {
		if selected != nil {
			c.selMorph = shape.NewMorph(rest, selected)
		} else {
			c.selMorph = nil
		}
	}
	if restChanged || pressChanged || c.pressMorph == nil {
		if pressed != nil {
			c.pressMorph = shape.NewMorph(rest, pressed)
		} else {
			c.pressMorph = nil
		}
	}
	if selChanged || pressChanged || (selected != nil && pressed != nil && c.selPressMorph == nil) {
		if selected != nil && pressed != nil {
			c.selPressMorph = shape.NewMorph(selected, pressed)
		} else {
			c.selPressMorph = nil
		}
	}
	c.rest = rest
	c.selected = selected
	c.pressed = pressed
}

func normalizeMorphSpring(tok motion.Token) motion.Token {
	if tok < motion.FastSpatial || tok > motion.SlowEffects {
		return motion.FastSpatial
	}
	return tok
}

// ShapeButton builds a Material Design 3 Expressive polygon-shaped button on
// top of ui.ButtonBase. It supports any *shape.RoundedPolygon (such as
// shape.Cookie9Sided(), shape.SoftBurst(), shape.Sunny(), or shape.Circle()),
// smoothly morphs toward PressedShape while pressed (and toward SelectedShape
// when Selected is true) via MD3E physics springs, clips its MD3E StateLayer
// and Expressive Ripple strictly to the live morphing outline, and adapts its
// SizeTier metrics to Theme.Density.
//
// label is required for accessibility and becomes the button's accessible name;
// ShapeButton panics when label is empty.
func ShapeButton(c *ui.Context, label string, icon IconDraw, opts ...ShapeButtonOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: ShapeButton takes at most one ShapeButtonOptions")
	}
	if label == "" {
		panic("md3e: ShapeButton requires an accessibility label")
	}
	var o ShapeButtonOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Role(ui.RoleButton).
		Label(label)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)
	if !disabled && el.Clicked() && o.OnClick != nil {
		o.OnClick()
	}

	th := CurrentTheme(c)
	restPoly := validShapePolygon(o.Shape, defaultShapeButtonCircle())
	var pressPoly *shape.RoundedPolygon
	if o.PressedShape != nil && len(o.PressedShape.Cubics()) >= 2 {
		pressPoly = o.PressedShape
	} else if o.Shape != nil && len(o.Shape.Cubics()) >= 2 && !samePolygon(restPoly, defaultShapeButtonCircle()) {
		pressPoly = defaultShapeButtonCircle()
	} else {
		pressPoly = defaultShapeButtonSquare()
	}
	var selPoly *shape.RoundedPolygon
	if o.Selected || o.SelectedShape != nil {
		selPoly = validShapePolygon(o.SelectedShape, defaultShapeButtonSquare())
	}

	offPal := shapeButtonPaletteOf(th, o.Variant, false, false, disabled, o.ContainerColor, o.ContentColor, o.BorderColor)
	onPal := offPal
	if o.Selected || o.SelectedShape != nil {
		onPal = shapeButtonPaletteOf(th, o.Variant, true, true, disabled, o.ContainerColor, o.ContentColor, o.BorderColor)
	}

	return renderShapeButton(c, th, el, shapeButtonRenderConfig{
		size:         o.Size,
		width:        o.Width,
		height:       o.Height,
		iconSize:     o.IconSize,
		restPoly:     restPoly,
		selPoly:      selPoly,
		pressPoly:    pressPoly,
		hasPressPoly: true,
		selected:     o.Selected,
		icon:         icon,
		selectedIcon: o.SelectedIcon,
		spring:       normalizeMorphSpring(o.Spring),
		offPal:       offPal,
		onPal:        onPal,
		disabled:     disabled,
	})
}

// MorphIconButton builds a Material Design 3 Expressive *bool-bound toggle
// button on top of ui.ButtonBase. Clicking or pressing Enter/Space toggles *on,
// morphs smoothly between Shape (off) and SelectedShape (on) with
// motion.FastSpatial (or the configured Spring), cross-fades/scales between
// icon and SelectedIcon, and clips StateLayer and Expressive Ripple to the live
// morphed outline.
//
// label is required and becomes the accessible name with ui.RoleToggleButton
// and Checked(*on). Pass c.Key(...) before calling MorphIconButton when the
// button can reorder across frames.
func MorphIconButton(c *ui.Context, on *bool, label string, icon IconDraw, opts ...MorphIconButtonOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: MorphIconButton takes at most one MorphIconButtonOptions")
	}
	if on == nil {
		panic("md3e: MorphIconButton requires a non-nil *bool")
	}
	if label == "" {
		panic("md3e: MorphIconButton requires an accessibility label")
	}
	var o MorphIconButtonOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Role(ui.RoleToggleButton).
		Label(label)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)
	if !disabled && el.Clicked() {
		*on = !*on
		if o.OnChanged != nil {
			o.OnChanged(*on)
		}
	}
	selected := *on
	el.Checked(selected)

	th := CurrentTheme(c)
	restPoly := validShapePolygon(o.Shape, defaultShapeButtonCircle())
	selPoly := validShapePolygon(o.SelectedShape, defaultShapeButtonSquare())
	var pressPoly *shape.RoundedPolygon
	hasPressPoly := false
	if o.PressedShape != nil && len(o.PressedShape.Cubics()) >= 2 {
		pressPoly = o.PressedShape
		hasPressPoly = true
	}

	offPal := shapeButtonPaletteOf(th, o.Variant, true, false, disabled, o.ContainerColor, o.ContentColor, o.BorderColor)
	onPal := shapeButtonPaletteOf(th, o.Variant, true, true, disabled, o.SelectedContainerColor, o.SelectedContentColor, o.BorderColor)

	return renderShapeButton(c, th, el, shapeButtonRenderConfig{
		size:         o.Size,
		width:        o.Width,
		height:       o.Height,
		iconSize:     o.IconSize,
		restPoly:     restPoly,
		selPoly:      selPoly,
		pressPoly:    pressPoly,
		hasPressPoly: hasPressPoly,
		selected:     selected,
		icon:         icon,
		selectedIcon: o.SelectedIcon,
		spring:       normalizeMorphSpring(o.Spring),
		offPal:       offPal,
		onPal:        onPal,
		disabled:     disabled,
	})
}

type shapeButtonRenderConfig struct {
	size         tokens.SizeTier
	width        float32
	height       float32
	iconSize     float32
	restPoly     *shape.RoundedPolygon
	selPoly      *shape.RoundedPolygon
	pressPoly    *shape.RoundedPolygon
	hasPressPoly bool
	selected     bool
	icon         IconDraw
	selectedIcon IconDraw
	spring       motion.Token
	offPal       buttonColors
	onPal        buttonColors
	disabled     bool
}

func renderShapeButton(c *ui.Context, th *Theme, el ui.Element, cfg shapeButtonRenderConfig) ui.Element {
	spec := tierSpecForDensity(th.Density, cfg.size)
	w := sanitizePositiveF32(cfg.width, float32(spec.Height))
	h := sanitizePositiveF32(cfg.height, float32(spec.Height))
	iconSize := sanitizePositiveF32(cfg.iconSize, float32(spec.IconSize))
	el.Size(w, h).Shrink(0)

	cache := ui.Local(el, shapeBtnMorphCacheKey{}, func() shapeBtnMorphCache {
		return shapeBtnMorphCache{}
	})
	cache.ensure(cfg.restPoly, cfg.selPoly, cfg.pressPoly)

	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	spatialAnim := th.Animation(cfg.spring)
	effectsAnim := th.Animation(motion.FastEffects)
	pressed := !cfg.disabled && el.Pressed()

	selTarget := boolFloat(cfg.selected)
	if !cfg.hasPressPoly && pressed && !reduce && cfg.selPoly != nil {
		// Subtle spring anticipation along the Shape <-> SelectedShape axis
		// when no distinct PressedShape was requested on a MorphIconButton.
		if cfg.selected {
			selTarget = 0.78
		} else {
			selTarget = 0.22
		}
	}
	selT := animateValue(el, shapeBtnMorphTKey{}, selTarget, spatialAnim, reduce)
	pressTarget := float32(0)
	if cfg.hasPressPoly && pressed {
		pressTarget = 1
	}
	pressT := animateValue(el, shapeBtnPressTKey{}, pressTarget, spatialAnim, reduce)
	iconT := animateValue(el, shapeBtnIconTKey{}, boolFloat(cfg.selected), effectsAnim, reduce)

	clampedIconT := clampF32(iconT, 0, 1)
	containerCol := cfg.offPal.container.MixOklab(cfg.onPal.container, clampedIconT)
	contentCol := cfg.offPal.content.MixOklab(cfg.onPal.content, clampedIconT)
	borderCol := cfg.offPal.border.MixOklab(cfg.onPal.border, clampedIconT)
	el.TextColor(contentCol)

	// Pick the active morph & progress for this frame (captured by value so
	// exit ghost-draws remain completely pure and safe).
	var activeMorph *shape.Morph
	var activeProgress float32
	switch {
	case cfg.hasPressPoly && pressT > 0.001 && (selT <= 0.01 || cfg.selPoly == nil) && cache.pressMorph != nil:
		activeMorph = cache.pressMorph
		activeProgress = clampF32(pressT, 0, 1)
	case cfg.hasPressPoly && pressT > 0.001 && selT >= 0.99 && cache.selPressMorph != nil:
		activeMorph = cache.selPressMorph
		activeProgress = clampF32(pressT, 0, 1)
	case cache.selMorph != nil:
		activeMorph = cache.selMorph
		activeProgress = clampF32(selT, 0, 1)
	case cache.pressMorph != nil:
		activeMorph = cache.pressMorph
		activeProgress = clampF32(pressT, 0, 1)
	}
	fallbackPoly := cfg.restPoly
	if cfg.selected && cfg.selPoly != nil {
		fallbackPoly = cfg.selPoly
	}

	pathForRect := func(rect ui.Rect) *ui.Path {
		if activeMorph != nil {
			return activeMorph.AsPath(activeProgress, rect)
		}
		if fallbackPoly != nil {
			return fallbackPoly.WriteTo(&cache.fallbackPath, rect)
		}
		return nil
	}

	// Draw and DrawOver finish flattening the outline before the other
	// callback can reuse its storage, even when the ripple uses an inset.
	borderInset := float32(0)
	if borderCol.A > 0 {
		borderInset = 0.5
	}
	attachRippleWithContext(c, th, el, StateLayerOptions{
		Color:       contentCol,
		ClipPath:    pathForRect,
		BorderInset: borderInset,
		Disabled:    cfg.disabled,
	})

	icon := cfg.icon
	selectedIcon := cfg.selectedIcon
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
			return
		}
		path := pathForRect(r)
		if path != nil {
			if containerCol.A > 0 {
				p.FillPath(path, containerCol)
			}
			if borderCol.A > 0 {
				p.StrokePath(path, 1, borderCol)
			}
		}
		drawMorphCrossfadeIcon(p, r, iconSize, icon, selectedIcon, contentCol, clampedIconT)
	})

	return el
}

// drawMorphCrossfadeIcon paints icon (and optionally cross-fades/scales to
// selectedIcon at progress t in [0, 1]) centered inside r.
func drawMorphCrossfadeIcon(p *ui.Painter, r ui.Rect, size float32, icon, selectedIcon IconDraw, ink ui.Color, t float32) {
	if p == nil || ink.A == 0 || size <= 0 {
		return
	}
	side := min(size, min(r.W, r.H))
	if side <= 0 {
		return
	}
	cx := r.X + r.W/2
	cy := r.Y + r.H/2
	if selectedIcon == nil {
		if icon == nil {
			return
		}
		dst := ui.Rect{X: cx - side/2, Y: cy - side/2, W: side, H: side}
		icon(p, dst, ink)
		return
	}

	if offA := 1 - t; offA > 0.01 && icon != nil {
		s := side * (0.75 + 0.25*offA)
		dst := ui.Rect{X: cx - s/2, Y: cy - s/2, W: s, H: s}
		icon(p, dst, ink.Alpha(offA))
	}
	if t > 0.01 {
		s := side * (0.75 + 0.25*t)
		dst := ui.Rect{X: cx - s/2, Y: cy - s/2, W: s, H: s}
		selectedIcon(p, dst, ink.Alpha(t))
	}
}

func shapeButtonPaletteOf(th *Theme, v ShapeButtonVariant, isToggle, on, disabled bool, containerOv, contentOv, borderOv ui.Color) buttonColors {
	cs := th.Colors
	if disabled {
		onSurface := cs.OnSurface
		content := onSurface.Alpha(float32(tokens.DisabledContentOpacity))
		switch v {
		case ShapeButtonOutlined:
			return buttonColors{content: content, border: onSurface.Alpha(float32(tokens.DisabledContainerOpacity))}
		case ShapeButtonStandard:
			return buttonColors{content: content}
		default:
			return buttonColors{
				container: onSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
				content:   content,
			}
		}
	}

	var pal buttonColors
	if isToggle && !on {
		switch v {
		case ShapeButtonFilled, ShapeButtonTonal:
			pal = buttonColors{
				container: cs.SurfaceContainerHighest,
				content:   cs.OnSurfaceVariant,
			}
		case ShapeButtonOutlined:
			pal = buttonColors{content: cs.OnSurfaceVariant, border: cs.Outline}
		default:
			pal = buttonColors{content: cs.OnSurfaceVariant}
		}
	} else {
		switch v {
		case ShapeButtonFilled:
			pal = buttonColors{container: cs.Primary, content: cs.OnPrimary}
		case ShapeButtonTonal:
			pal = buttonColors{container: cs.SecondaryContainer, content: cs.OnSecondaryContainer}
		case ShapeButtonOutlined:
			if isToggle && on {
				pal = buttonColors{container: cs.InverseSurface, content: cs.InverseOnSurface}
			} else {
				pal = buttonColors{content: cs.OnSurfaceVariant, border: cs.Outline}
			}
		default:
			if isToggle && on {
				pal = buttonColors{content: cs.Primary}
			} else {
				pal = buttonColors{content: cs.OnSurfaceVariant}
			}
		}
	}

	if containerOv.A > 0 {
		pal.container = containerOv
	}
	if contentOv.A > 0 {
		pal.content = contentOv
	}
	if borderOv.A > 0 {
		pal.border = borderOv
	}
	return pal
}

// IconBadgeVariant selects the MD3E color treatment of IconBadge.
type IconBadgeVariant int

const (
	// IconBadgeTonal paints SecondaryContainer with OnSecondaryContainer ink (default).
	IconBadgeTonal IconBadgeVariant = iota
	// IconBadgeFilled paints Primary with OnPrimary ink.
	IconBadgeFilled
	// IconBadgeOutlined paints a transparent container with OutlineVariant border
	// and OnSurfaceVariant ink.
	IconBadgeOutlined
	// IconBadgePrimaryTonal paints PrimaryContainer with OnPrimaryContainer ink.
	IconBadgePrimaryTonal
	// IconBadgeTertiaryTonal paints TertiaryContainer with OnTertiaryContainer ink.
	IconBadgeTertiaryTonal
	// IconBadgeErrorTonal paints ErrorContainer with OnErrorContainer ink.
	IconBadgeErrorTonal
)

// IconBadgeOptions configures IconBadge. The zero value is a SizeXS tonal
// rounded-square badge scaled by Theme.Density.
type IconBadgeOptions struct {
	// Variant selects the MD3E container and icon color roles (default IconBadgeTonal).
	Variant IconBadgeVariant
	// Size selects the XS..XL tier scaled by Theme.Density.ButtonSpec.
	Size tokens.SizeTier
	// ContainerSize optionally overrides the badge's square side in DIP.
	ContainerSize float32
	// IconSize optionally overrides the inner icon size in DIP.
	IconSize float32
	// Shape optionally renders the badge container using a custom
	// *shape.RoundedPolygon (for example shape.Cookie9Sided() or shape.Sunny()).
	// Takes precedence over Radius and ShapeScale.
	Shape *shape.RoundedPolygon
	// ShapeScale selects an MD3E corner radius token when Shape is nil. Zero
	// defaults to tokens.ShapeMedium.
	ShapeScale tokens.ShapeScale
	// Radius overrides the corner radius in DIP when Shape is nil and positive.
	Radius float32
	// ContainerColor optionally overrides the container fill color.
	ContainerColor ui.Color
	// ContentColor optionally overrides the icon ink color.
	ContentColor ui.Color
	// BorderColor optionally overrides the Outlined border color.
	BorderColor ui.Color
	// Label optionally sets an accessible name (RoleImage). When empty, the
	// badge is decorative.
	Label string
	// Disabled applies MD3E disabled container and content tokens.
	Disabled bool
}

// IconBadge builds an MD3E icon container badge with tonal, filled, outlined,
// or custom polygon styling, suited for quick-toggle cards, list prefixes, and
// expressive status accents.
func IconBadge(c *ui.Context, icon IconDraw, opts ...IconBadgeOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: IconBadge takes at most one IconBadgeOptions")
	}
	var o IconBadgeOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	spec := tierSpecForDensity(th.Density, o.Size)
	side := sanitizePositiveF32(o.ContainerSize, float32(spec.Height))
	iconSize := sanitizePositiveF32(o.IconSize, float32(spec.IconSize))
	if iconSize > side {
		iconSize = side * 0.65
	}

	el := ui.Box(c).Size(side, side).Shrink(0).DimDisabled(false)
	if o.Label != "" {
		el.Role(ui.RoleImage).Label(o.Label)
	}
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	bg, fg, border := iconBadgePaletteOf(th, o.Variant, disabled, o.ContainerColor, o.ContentColor, o.BorderColor)
	el.TextColor(fg)

	poly := validShapePolygon(o.Shape, nil)
	if poly != nil {
		scratch := localDrawPath(el)
		el.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			path := poly.WriteTo(scratch, r)
			if bg.A > 0 {
				p.FillPath(path, bg)
			}
			if border.A > 0 {
				p.StrokePath(path, 1, border)
			}
			drawMorphCrossfadeIcon(p, r, iconSize, icon, nil, fg, 0)
		})
		return el
	}

	scale := tokens.ShapeMedium
	if o.ShapeScale > tokens.ShapeNone && o.ShapeScale <= tokens.ShapeFull {
		scale = o.ShapeScale
	}
	radius := sanitizePositiveF32(o.Radius, th.Shapes.Radius(scale))
	if radius > side/2 {
		radius = side / 2
	}
	if bg.A > 0 {
		el.Background(bg)
	}
	if border.A > 0 {
		el.Border(1, border)
	}
	el.Radius(radius)
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
			return
		}
		drawMorphCrossfadeIcon(p, r, iconSize, icon, nil, fg, 0)
	})
	return el
}

func iconBadgePaletteOf(th *Theme, v IconBadgeVariant, disabled bool, containerOv, contentOv, borderOv ui.Color) (bg, fg, border ui.Color) {
	cs := th.Colors
	if disabled {
		onSurface := cs.OnSurface
		fg = onSurface.Alpha(float32(tokens.DisabledContentOpacity))
		if v == IconBadgeOutlined {
			return ui.Transparent, fg, onSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		}
		return onSurface.Alpha(float32(tokens.DisabledContainerOpacity)), fg, ui.Transparent
	}
	switch v {
	case IconBadgeFilled:
		bg, fg = cs.Primary, cs.OnPrimary
	case IconBadgeOutlined:
		bg, fg, border = ui.Transparent, cs.OnSurfaceVariant, cs.OutlineVariant
		if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
			border = cs.Outline
		}
	case IconBadgePrimaryTonal:
		bg, fg = cs.PrimaryContainer, cs.OnPrimaryContainer
	case IconBadgeTertiaryTonal:
		bg, fg = cs.TertiaryContainer, cs.OnTertiaryContainer
	case IconBadgeErrorTonal:
		bg, fg = cs.ErrorContainer, cs.OnErrorContainer
	default:
		bg, fg = cs.SecondaryContainer, cs.OnSecondaryContainer
	}
	if containerOv.A > 0 {
		bg = containerOv
	}
	if contentOv.A > 0 {
		fg = contentOv
	}
	if borderOv.A > 0 {
		border = borderOv
	}
	return bg, fg, border
}
