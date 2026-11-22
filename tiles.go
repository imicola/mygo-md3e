package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// SwitchCardOptions configures SwitchCard, a one-row card-style quick toggle
// combining an optional leading icon badge, title/subtitle, and trailing MD3E
// Switch with isolated key ownership.
type SwitchCardOptions struct {
	// Subtitle displays muted secondary copy beneath the title.
	Subtitle string
	// Leading renders a custom leading widget (such as Avatar or custom IconBadge).
	// Takes precedence over LeadingIcon when non-nil.
	Leading func()
	// LeadingIcon renders a leading IconBadge when Leading is nil.
	LeadingIcon IconDraw
	// IconBadge customizes the IconBadge generated for LeadingIcon.
	IconBadge IconBadgeOptions
	// OnIcon overrides the Switch's selected thumb icon.
	OnIcon IconDraw
	// OffIcon sets an optional Switch unselected thumb icon.
	OffIcon IconDraw
	// Variant selects the card container style (default CardFilled / SurfaceContainerLow).
	Variant CardVariant
	// HasVariant explicitly selects CardElevated (zero value) vs the SwitchCard
	// default tonal SurfaceContainerLow look.
	HasVariant bool
	// Hairline draws a subtle 1 DIP tonal border around the card.
	Hairline bool
	// ShapeScale selects an MD3E corner radius token (default ShapeLarge).
	ShapeScale tokens.ShapeScale
	// Radius overrides the corner radius in DIP when positive and finite.
	Radius float32
	// OnChanged runs after clicking the card row, clicking the switch, or
	// pressing Space while the switch has keyboard focus.
	OnChanged func(on bool)
	// Label overrides the switch's accessible name (defaults to title).
	Label string
	// Disabled disables both the card row and the embedded Switch.
	Disabled bool
}

type switchCardSwitchKey struct{}

// SwitchCard builds a single-row card quick-toggle bound to *on: an optional
// leading IconBadge or custom widget, a title and optional subtitle, and a
// trailing MD3E Switch.
//
// Key isolation: any caller c.Key(...) lands on the outer card container while
// the embedded Switch uses an independent stable child key, preventing key
// theft across dynamic lists. Clicking either the card body or the Switch
// toggles *on exactly once without conflict, and keyboard focus lands cleanly
// on the single RoleSwitch control.
func SwitchCard(c *ui.Context, title string, on *bool, opts ...SwitchCardOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: SwitchCard takes at most one SwitchCardOptions")
	}
	if on == nil {
		panic("md3e: SwitchCard requires a non-nil *bool")
	}
	var o SwitchCardOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	m := tileMetricsForDensity(th.Density, o.Subtitle != "")
	radius := resolveTileRadius(th, o.ShapeScale, o.Radius)

	row := ui.Row(c).
		FillWidth().
		MinHeight(m.minH).
		Padding(m.padY, 0, m.padY, 0).
		PaddingStart(m.padX).
		PaddingEnd(m.padX).
		Gap(m.gap).
		AlignItems(ui.Center).
		Clip().
		DimDisabled(false)
	disabled := o.Disabled || row.IsDisabled()
	row.Disabled(o.Disabled)

	bg, border, elevation := tileContainerPalette(th, o.Variant, o.HasVariant, o.Hairline, disabled)
	onT := clampF32(animateEffect(c, row, th, animChan0, boolFloat(*on && !disabled), motion.FastEffects), 0, 1)
	if onT > 0.001 && bg.A > 0 && cs.SecondaryContainer.A > 0 {
		bg = bg.MixOklab(cs.SecondaryContainer, 0.28*onT)
	}
	rowPressed := !disabled && row.Pressed()
	cornerTarget := radius
	if rowPressed {
		cornerTarget = max(radius-4, 8)
	}
	curRadius := max(animateWithTheme(row, th, animChan1, cornerTarget, motion.FastSpatial), 0)
	row.Radius(curRadius)

	if bg.A > 0 {
		row.Background(bg)
	}
	if border.A > 0 {
		row.Border(1, border)
	}
	if elevation > tokens.Level0 && !disabled {
		ApplyElevationShadows(th, row, elevation)
		if !rowPressed {
			attachHoverElevation(row, th, elevation, min(elevation+1, tokens.Level5), curRadius, false)
		}
	}

	titleInk := cs.OnSurface
	subInk := cs.OnSurfaceVariant
	if disabled {
		titleInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		subInk = titleInk
	}
	row.TextColor(titleInk)

	borderInset := float32(0)
	if border.A > 0 {
		borderInset = 1
	}
	attachRippleWithContext(c, th, row, StateLayerOptions{
		Color:       titleInk,
		Radius:      curRadius,
		BorderInset: borderInset,
		Disabled:    disabled,
	})

	a11yLabel := o.Label
	if a11yLabel == "" {
		a11yLabel = title
	}

	var sw ui.Element
	row.Children(func() {
		if o.Leading != nil {
			o.Leading()
		} else if o.LeadingIcon != nil {
			bo := o.IconBadge
			if bo.ContainerSize <= 0 {
				bo.ContainerSize = m.badgeSide
			}
			if bo.IconSize <= 0 {
				bo.IconSize = m.iconSize
			}
			bo.Disabled = bo.Disabled || disabled
			IconBadge(c, o.LeadingIcon, bo)
		}

		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
			if title != "" {
				txt := applyTypeStyleWithTheme(th, ui.Text(c, title).MaxLines(1), m.titleRole)
				txt.TextColor(titleInk)
			}
			if o.Subtitle != "" {
				sub := applyTypeStyleWithTheme(th, ui.Text(c, o.Subtitle).MaxLines(2), m.subRole)
				sub.TextColor(subInk)
			}
		})

		sw = Switch(c.Key(switchCardSwitchKey{}), on, SwitchOptions{
			Disabled: disabled,
			OnIcon:   o.OnIcon,
			OffIcon:  o.OffIcon,
		})
		if a11yLabel != "" {
			sw.Label(a11yLabel)
		}
	})

	// Synchronize row click and embedded Switch without double-toggling: if the
	// Switch itself handled the interaction, sw.Changed() is true and *on is
	// already updated; otherwise a click anywhere else on the card toggles *on.
	swChanged := !disabled && sw.Valid() && sw.Changed()
	rowClicked := !disabled && row.Clicked()
	if swChanged {
		if o.OnChanged != nil {
			o.OnChanged(*on)
		}
	} else if rowClicked {
		*on = !*on
		if o.OnChanged != nil {
			o.OnChanged(*on)
		}
		c.Invalidate()
	}

	return row
}

// ActionTileOptions configures ActionTile, a one-row clickable navigation or
// action card with leading icon badge, title/subtitle, trailing metadata/badge,
// and an RTL-adaptive chevron.
type ActionTileOptions struct {
	// Subtitle displays muted secondary copy beneath the title.
	Subtitle string
	// Leading renders a custom leading widget (such as Avatar or custom IconBadge).
	// Takes precedence over LeadingIcon when non-nil.
	Leading func()
	// LeadingIcon renders a leading IconBadge when Leading is nil.
	LeadingIcon IconDraw
	// IconBadge customizes the IconBadge generated for LeadingIcon.
	IconBadge IconBadgeOptions
	// Meta displays optional right-aligned muted metadata (such as a version,
	// latency, or selected value) before the chevron.
	Meta string
	// Badge displays an optional compact Badge before the chevron.
	Badge string
	// Trailing renders custom trailing content before the chevron.
	Trailing func()
	// HideChevron suppresses the trailing RTL-adaptive navigation chevron.
	HideChevron bool
	// Variant selects the card container style.
	Variant CardVariant
	// HasVariant explicitly selects CardElevated (zero value) vs the default
	// tonal SurfaceContainerLow container.
	HasVariant bool
	// Hairline draws a subtle 1 DIP tonal border around the tile.
	Hairline bool
	// ShapeScale selects an MD3E corner radius token (default ShapeLarge).
	ShapeScale tokens.ShapeScale
	// Radius overrides the corner radius in DIP when positive and finite.
	Radius float32
	// OnClick runs when the tile is clicked or activated with Enter/Space.
	OnClick func()
	// Label overrides the accessible button name (defaults to title).
	Label string
	// Disabled disables interaction and applies MD3E disabled color tokens.
	Disabled bool
}

// ActionTile builds a single-row clickable card tile on top of ui.ButtonBase
// with optional leading icon badge, title/subtitle, trailing metadata/badge,
// RTL-adaptive chevron (ChevronRight in LTR, ChevronLeft in RTL), full keyboard
// activation (Enter/Space), and MD3E StateLayer/Ripple feedback.
func ActionTile(c *ui.Context, title string, opts ...ActionTileOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: ActionTile takes at most one ActionTileOptions")
	}
	var o ActionTileOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	m := tileMetricsForDensity(th.Density, o.Subtitle != "")
	radius := resolveTileRadius(th, o.ShapeScale, o.Radius)
	rtl := c.Direction() == ui.RightToLeft

	el := ui.ButtonBase(c).
		FillWidth().
		MinHeight(m.minH).
		Padding(m.padY, 0, m.padY, 0).
		PaddingStart(m.padX).
		PaddingEnd(m.padX).
		Gap(m.gap).
		Justify(ui.Start).
		AlignItems(ui.Center).
		Clip().
		DimDisabled(false).
		FocusRing(false)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	pressed := !disabled && el.Pressed()
	cornerTarget := radius
	if pressed {
		cornerTarget = max(radius-4, 8)
	}
	curRadius := max(animateWithTheme(el, th, animChan0, cornerTarget, motion.FastSpatial), 0)
	el.Radius(curRadius)

	bg, border, elevation := tileContainerPalette(th, o.Variant, o.HasVariant, o.Hairline, disabled)
	if bg.A > 0 {
		el.Background(bg)
	}
	if border.A > 0 {
		el.Border(1, border)
	}
	if elevation > tokens.Level0 && !disabled {
		ApplyElevationShadows(th, el, elevation)
		if !pressed {
			attachHoverElevation(el, th, elevation, min(elevation+1, tokens.Level5), curRadius, false)
		}
	}

	titleInk := cs.OnSurface
	subInk := cs.OnSurfaceVariant
	if disabled {
		titleInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		subInk = titleInk
	}
	el.TextColor(titleInk)

	name := o.Label
	if name == "" {
		name = title
	}
	if name != "" {
		el.Label(name)
	}
	if o.Meta != "" {
		el.Description(o.Meta)
	}

	if !disabled && el.Clicked() && o.OnClick != nil {
		o.OnClick()
	}

	borderInset := float32(0)
	if border.A > 0 {
		borderInset = 1
	}
	attachRippleWithContext(c, th, el, StateLayerOptions{
		Color:       titleInk,
		Radius:      curRadius,
		BorderInset: borderInset,
		Disabled:    disabled,
	})

	thReduce := th.ReduceMotion
	el.Children(func() {
		if o.Leading != nil {
			o.Leading()
		} else if o.LeadingIcon != nil {
			bo := o.IconBadge
			if bo.ContainerSize <= 0 {
				bo.ContainerSize = m.badgeSide
			}
			if bo.IconSize <= 0 {
				bo.IconSize = m.iconSize
			}
			bo.Disabled = bo.Disabled || disabled
			IconBadge(c, o.LeadingIcon, bo)
		}

		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
			if title != "" {
				txt := applyTypeStyleWithTheme(th, ui.Text(c, title).MaxLines(1), m.titleRole)
				txt.TextColor(titleInk)
			}
			if o.Subtitle != "" {
				sub := applyTypeStyleWithTheme(th, ui.Text(c, o.Subtitle).MaxLines(2), m.subRole)
				sub.TextColor(subInk)
			}
		})

		if o.Meta != "" {
			meta := applyTypeStyleWithTheme(th, ui.Text(c, o.Meta).NoWrap().Shrink(0), tokens.LabelMedium)
			meta.TextColor(subInk)
		}
		if o.Badge != "" {
			bdgOpts := BadgeOptions{ShowZero: true}
			if disabled {
				bdgOpts.ContainerColor = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
				bdgOpts.ContentColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
			}
			Badge(c, o.Badge, bdgOpts)
		}
		if o.Trailing != nil {
			o.Trailing()
		}
		if !o.HideChevron {
			chev := mdicon.ChevronRight.Draw
			nudgeSign := float32(1)
			if rtl {
				chev = mdicon.ChevronLeft.Draw
				nudgeSign = -1
			}
			chevEl := ui.Box(c).Size(18, 18).Shrink(0)
			chevHover := localHaloFade(chevEl, animChan0)
			chevEl.Draw(func(p *ui.Painter, r ui.Rect) {
				if r.W <= 0 || r.H <= 0 {
					return
				}
				live := el.Valid() && chevEl.Valid()
				reduce := thReduce || p.Preferences().ReduceMotion
				var t float32
				var more bool
				if live && !disabled && rectIntersects(r, p.Visible()) {
					target := float32(0)
					if el.Hovered() {
						target = 1
					}
					t, more = chevHover.step(target, p.Now(), reduce)
				} else {
					t = chevHover.alpha
				}
				dx := nudgeSign * 2 * t
				ink := subInk
				if t > 0.001 && !disabled {
					ink = subInk.MixOklab(titleInk, 0.5*t)
				}
				chev(p, ui.Rect{X: r.X + dx, Y: r.Y, W: r.W, H: r.H}, ink)
				if live && more {
					p.After(16 * time.Millisecond)
				}
			})
		}
	})

	return el
}

type tileMetrics struct {
	minH      float32
	padX      float32
	padY      float32
	gap       float32
	badgeSide float32
	iconSize  float32
	titleRole tokens.TypeRole
	subRole   tokens.TypeRole
}

func tileMetricsForDensity(d tokens.Density, twoLine bool) tileMetrics {
	one, two, _ := d.ListHeights()
	minH := float32(one)
	if twoLine {
		minH = float32(two)
	}
	switch d.Normalize() {
	case tokens.DensityCompact:
		return tileMetrics{
			minH:      minH,
			padX:      10,
			padY:      6,
			gap:       10,
			badgeSide: 28,
			iconSize:  16,
			titleRole: tokens.TitleSmall,
			subRole:   tokens.BodySmall,
		}
	case tokens.DensityDesktop:
		return tileMetrics{
			minH:      minH,
			padX:      14,
			padY:      8,
			gap:       12,
			badgeSide: 34,
			iconSize:  18,
			titleRole: tokens.TitleSmall,
			subRole:   tokens.BodySmall,
		}
	default:
		return tileMetrics{
			minH:      minH,
			padX:      16,
			padY:      10,
			gap:       14,
			badgeSide: 40,
			iconSize:  20,
			titleRole: tokens.TitleMedium,
			subRole:   tokens.BodyMedium,
		}
	}
}

func resolveTileRadius(th *Theme, scale tokens.ShapeScale, override float32) float32 {
	base := tokens.ShapeLarge
	if scale > tokens.ShapeNone && scale <= tokens.ShapeFull {
		base = scale
	}
	return sanitizePositiveF32(override, th.Shapes.Radius(base))
}

func tileContainerPalette(th *Theme, variant CardVariant, hasVariant, hairline, disabled bool) (bg, border ui.Color, elev tokens.ElevationLevel) {
	cs := th.Colors
	if disabled {
		return cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)), ui.Transparent, tokens.Level0
	}
	if !hasVariant && variant == CardElevated {
		// Default tile surface: clean tonal SurfaceContainerLow without heavy shadow.
		bg = cs.SurfaceContainerLow
		if bg.A == 0 {
			bg = SurfaceColorForElevation(th, tokens.Level1)
		}
		if hairline {
			border = resolveHairlineColor(th, ui.Color{})
		}
		return bg, border, tokens.Level0
	}
	bg, border, elev = cardPalette(th, variant, false)
	if hairline && border.A == 0 {
		border = resolveHairlineColor(th, ui.Color{})
	}
	return bg, border, elev
}
