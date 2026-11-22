package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// NavigationRailIndicatorStyle selects the active destination indicator style
// rendered by NavigationRail.
type NavigationRailIndicatorStyle int

const (
	// NavigationRailIndicatorPill renders the default MD3E SecondaryContainer
	// capsule that slides vertically between destinations.
	NavigationRailIndicatorPill NavigationRailIndicatorStyle = iota
	// NavigationRailIndicatorPillWithBar pairs the sliding active capsule with
	// a desktop-style leading accent bar (on the left in LTR, right in RTL)
	// that tracks the active destination alongside the capsule.
	NavigationRailIndicatorPillWithBar
)

// NavigationRailOptions configures a NavigationRail. The zero value is an
// enabled, collapsed rail whose destinations align to the top.
type NavigationRailOptions struct {
	// Selected points to the selected destination, -1 for none. The rail
	// writes clicks and keyboard activation into it as it builds.
	Selected *int
	// Changed runs after a click or key chooses a destination.
	Changed func(i int)
	// Expanded lets the caller own the collapse state; nil keeps it in the
	// rail's own local state.
	Expanded *bool
	// ExpandedWidth overrides the expanded rail width in DIP. Values <= 0 or
	// non-finite fall back to the active theme Density default (256 Comfortable,
	// 192 Desktop, 180 Compact); positive values below the collapsed width (80)
	// are clamped to 80 so collapse/expand transitions stay aligned.
	ExpandedWidth float32
	// IndicatorStyle selects the active indicator appearance: the default MD3E
	// sliding capsule (NavigationRailIndicatorPill) or a desktop capsule paired
	// with a leading accent bar (NavigationRailIndicatorPillWithBar).
	IndicatorStyle NavigationRailIndicatorStyle
	// Header builds the optional block above the destinations, as a FAB.
	Header func() ui.Element
	// Align places the destinations at the Start, Center or End of the rail.
	Align ui.Align
	// Label names the rail for assistive technology; "Navigation" when empty.
	Label string
	// Disabled disables every destination.
	Disabled bool
}

// NavigationRail metrics in DIP.
const (
	navRailCollapsedWidth = 80
	navRailExpandedWidth  = 256
	navRailItemHeight     = 56
	navRailIndicatorHigh  = 32
	navRailIconSize       = 24
	navRailPaddingX       = 12
	navRailAccentBarWidth = 3
	navRailAccentBarInset = 4
	// navRailCollapsedItemW is a destination's width in the collapsed rail.
	navRailCollapsedItemW = navRailCollapsedWidth - 2*navRailPaddingX
	// navRailIconTop is the icon's leading slot offset in the collapsed rail;
	// the expanded pill centers it.
	navRailIconTop = 4
	// navRailLabelTop is the short label's top in the collapsed rail.
	navRailLabelTop = 38
)

type (
	navRailScrollKey   struct{}
	navRailListKey     struct{}
	navRailIndStateKey struct{}
	navRailItemKey     struct{ i int }
)

type navRailIndicatorState struct {
	set              bool
	hasSel           bool
	y, h             float32
	alpha            float32
	builtRelY        float32
	builtItemH       float32
	paintedListY     float32
	paintedListValid bool
}

// NavigationRail builds a Material Design 3 Expressive navigation rail: a
// vertical strip of destinations along the start edge of a window. Collapsed,
// it is 80 DIP wide and stacks each icon over its short label; expanded, it
// grows to the Density default (256/192/180 DIP) or ExpandedWidth with the icon
// and label in a row on a SecondaryContainer pill that glides smoothly along
// the vertical axis between destinations with the Fast Spatial spring. The rail
// width springs between collapsed and expanded with the Default Spatial spring,
// and reduced motion settles both at once with zero pending frames.
//
// An optional Header slot holds a FAB, and the arrows move the focus among the
// destinations, Enter or Space choosing:
//
//	md3e.NavigationRail(c, []md3e.NavigationDestination{
//		{Label: "Home", Icon: homeIcon},
//		{Label: "Mail", Icon: mailIcon, Badge: "3"},
//	}, md3e.NavigationRailOptions{Selected: &app.tab, Expanded: &app.railExpanded})
func NavigationRail(c *ui.Context, destinations []NavigationDestination, opts ...NavigationRailOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: NavigationRail takes at most one NavigationRailOptions")
	}
	var o NavigationRailOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	cs := th.Colors
	label := o.Label
	if label == "" {
		label = "Navigation"
	}

	root := ui.Column(c).
		DimDisabled(false).
		FillHeight().
		Clip().
		Background(SurfaceColorForElevation(th, tokens.Level1)).
		Role(ui.RoleNavigation).
		FocusGroup(ui.Vertical).
		Label(label)
	if hl := resolveHairlineColor(th, ui.Color{}); hl.A > 0 {
		if c.Direction() == ui.RightToLeft {
			root.Border(1, hl).BorderWidth(0, 0, 0, 1)
		} else {
			root.Border(1, hl).BorderWidth(0, 1, 0, 0)
		}
	}
	root.Disabled(o.Disabled)
	expanded := o.Expanded
	if expanded == nil {
		expanded = localBool(root, animChan1)
	}
	expandTarget := float32(0)
	if *expanded {
		expandTarget = 1
	}
	// One spring drives both the rail's width and every destination's pill,
	// icon and label, so the mid-transition frames stay aligned rather than
	// switching layouts on the boolean.
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	rtl := c.Direction() == ui.RightToLeft
	expandedW := resolveNavRailExpandedWidth(o.ExpandedWidth, th.Density)
	expand := animateValue(root, animChan0, expandTarget, th.Animation(motion.DefaultSpatial), reduce)
	railW := navRailCollapsedWidth + (expandedW-navRailCollapsedWidth)*expand
	root.Width(railW)
	itemW := max(0, railW-2*navRailPaddingX)
	itemGap := navRailItemGap(th.Density)

	var listCol ui.Element
	var stackItemEls [12]ui.Element
	var itemEls []ui.Element
	if len(destinations) <= len(stackItemEls) {
		itemEls = stackItemEls[:len(destinations)]
	} else {
		itemEls = make([]ui.Element, len(destinations))
	}

	root.Children(func() {
		if o.Header != nil {
			ui.Box(c).FillWidth().Padding(12).Children(func() {
				o.Header()
			})
		}
		// The destinations scroll when the rail is shorter than its list, as
		// in a short window or a phone's landscape rail; at full height the
		// scroll takes the list's own height and Align still places it.
		ui.Column(c).Grow(1).FillWidth().Justify(o.Align).Children(func() {
			ui.Scroll(c.Key(navRailScrollKey{})).FillWidth().Children(func() {
				listCol = ui.Column(c.Key(navRailListKey{})).FillWidth().PaddingX(navRailPaddingX).Gap(itemGap)
				listCol.Children(func() {
					for i := range destinations {
						itemEls[i] = navRailDestination(c, th, cs, &destinations[i], i, o, expand, itemW)
					}
				})
			})
		})
	})

	if !listCol.Valid() {
		return root
	}

	selIdx := -1
	if o.Selected != nil && *o.Selected >= 0 && *o.Selected < len(destinations) {
		selIdx = *o.Selected
	}
	ind := localPtr(listCol, navRailIndStateKey{}, func() *navRailIndicatorState {
		return &navRailIndicatorState{}
	})
	ind.paintedListValid = false

	srv := c.Services()
	if selIdx >= 0 {
		selEl := itemEls[selIdx]
		listBox := listCol.LayoutBounds()
		itemBox := selEl.LayoutBounds()
		targetY := float32(selIdx) * (navRailItemHeight + itemGap)
		targetH := float32(navRailItemHeight)
		if listBox.H > 0 && itemBox.H > 0 {
			targetY = itemBox.Y - listBox.Y
			targetH = itemBox.H
		}
		ind.builtRelY = targetY
		ind.builtItemH = targetH

		if !ind.set || !ind.hasSel {
			// First frame or appearing from deselected (-1): snap coordinates
			// immediately so the indicator never flies in from y = 0.
			ind.y = listCol.Jump(animChan0, targetY)
			ind.h = listCol.Jump(animChan1, targetH)
			if !ind.set {
				ind.alpha = listCol.Jump(animChan2, 1)
			} else {
				ind.alpha = animateValue(listCol, animChan2, 1, th.Animation(motion.FastEffects), reduce)
			}
			ind.set = true
			ind.hasSel = true
		} else {
			ind.y = animateValue(listCol, animChan0, targetY, th.Animation(motion.FastSpatial), reduce)
			ind.h = animateValue(listCol, animChan1, targetH, th.Animation(motion.FastSpatial), reduce)
			ind.alpha = animateValue(listCol, animChan2, 1, th.Animation(motion.FastEffects), reduce)
		}

		if selEl.Valid() {
			// Draw (not DrawOver) preserves attachRippleWithContext's DrawOver
			// state layer on selEl while verifying layout sync after listCol.Draw.
			selEl.Draw(func(_ *ui.Painter, tr ui.Rect) {
				if !selEl.Valid() || !ind.paintedListValid || tr.W <= 0 || tr.H <= 0 {
					return
				}
				if absF32((tr.Y-ind.paintedListY)-ind.builtRelY) > 0.5 ||
					absF32(tr.H-ind.builtItemH) > 0.5 {
					srv.Invalidate()
				}
			})
		}
	} else {
		if !ind.set {
			ind.alpha = listCol.Jump(animChan2, 0)
			ind.set = true
			ind.hasSel = false
		} else {
			ind.hasSel = false
			ind.alpha = animateValue(listCol, animChan2, 0, th.Animation(motion.FastEffects), reduce)
		}
	}

	indY := ind.y
	indH := ind.h
	indAlpha := clampF32(ind.alpha, 0, 1)
	indReady := ind.set && indAlpha > 0.001 && indH > 0
	expandT := clampF32(expand, 0, 1)

	selDisabled := o.Disabled || root.IsDisabled()
	if selIdx >= 0 {
		selDisabled = selDisabled || destinations[selIdx].Disabled || itemEls[selIdx].IsDisabled()
	}
	pillColor := cs.SecondaryContainer.Alpha(indAlpha)
	barColor := cs.Primary.Alpha(indAlpha)
	if selDisabled {
		pillColor = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity) * indAlpha)
		barColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity) * indAlpha)
	}
	showBar := o.IndicatorStyle == NavigationRailIndicatorPillWithBar

	// Paint the shared sliding pill (and optional leading accent bar) inside
	// listCol.Draw so it tracks scroll offsets automatically, clips to the
	// scroll viewport, paints beneath destination icons/labels/ripples, and
	// stays safe on ElementTransition exit ghosts (!listCol.Valid()).
	listCol.Draw(func(p *ui.Painter, r ui.Rect) {
		if !indReady || r.W <= 0 || r.H <= 0 {
			return
		}
		if listCol.Valid() {
			ind.paintedListY = r.Y
			ind.paintedListValid = true
		}
		cell := ui.Rect{
			X: r.X + navRailPaddingX,
			Y: r.Y + indY,
			W: max(0, r.W-2*navRailPaddingX),
			H: indH,
		}
		if cell.W <= 0 || cell.H <= 0 {
			return
		}
		box, radius := navRailPillRect(cell, expandT, 0, rtl)
		if box.W > 0 && box.H > 0 {
			p.FillRadii(box, [4]float32{radius, radius, radius, radius}, pillColor)
			if showBar && barColor.A > 0 {
				barW := float32(navRailAccentBarWidth)
				barH := clampF32(box.H*0.5, 14, 24)
				barY := box.Y + (box.H-barH)/2
				barX := r.X + navRailAccentBarInset
				if rtl {
					barX = r.X + r.W - navRailAccentBarInset - barW
				}
				p.Fill(ui.Rect{X: barX, Y: barY, W: barW, H: barH}, barColor, barW/2)
			}
		}
	})

	return root
}

// navRailDestination builds one destination of a NavigationRail. expand is the
// spring's 0 (collapsed) to 1 (expanded) progress, shared by every destination
// and the rail's width.
func navRailDestination(c *ui.Context, th *Theme, cs tokens.ColorScheme, dest *NavigationDestination, i int, o NavigationRailOptions, expand, itemW float32) ui.Element {
	c.Key(navRailItemKey{i: i})
	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		FillWidth().
		Height(navRailItemHeight).
		Label(dest.Label).
		Role(ui.RoleTab)
	disabled := o.Disabled || dest.Disabled || el.IsDisabled()
	selected := o.Selected != nil && *o.Selected == i

	selT := clampF32(animateEffect(c, el, th, animChan0, boolFloat(selected), motion.FastEffects), 0, 1)
	iconInk := cs.OnSurfaceVariant.MixOklab(cs.OnSecondaryContainer, selT)
	// Collapsed short labels rest below the pill on the rail surface (OnSurface);
	// expanded long labels sit inside the SecondaryContainer pill (OnSecondaryContainer).
	shortInk := cs.OnSurfaceVariant.MixOklab(cs.OnSurface, selT)
	longInk := cs.OnSurfaceVariant.MixOklab(cs.OnSecondaryContainer, selT)
	if disabled {
		iconInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		shortInk = iconInk
		longInk = iconInk
	}
	el.Disabled(dest.Disabled).AccessSelected(selected)

	t := clampF32(expand, 0, 1)
	// Smooth crossfade that fades the short label cleanly after mid-expansion
	// while bringing the long label in so endpoints never ghost each other.
	shortAlpha := clampF32(1-(t-0.12)/0.56, 0, 1)
	longAlpha := clampF32((t-0.30)/0.55, 0, 1)
	rtl := c.Direction() == ui.RightToLeft
	el.Row().AlignItems(ui.Center).Radius(navRailItemHeight / 2)

	el.Children(func() {
		// The icon rides a fixed 56 DIP leading slot, so its horizontal center
		// stays put while the item grows; it only glides down to the middle of
		// the expanded pill.
		iconTop := navRailIconTop + (float32(navRailItemHeight-navRailIndicatorHigh)/2-navRailIconTop)*t
		ui.Box(c).Size(navRailCollapsedItemW, navRailIndicatorHigh).Center().
			Attach(ui.AnchorTopStart, ui.AnchorTopStart).Start(0).Top(iconTop).
			Children(func() {
				badgedIcon(c, dest, navRailIconSize, iconInk)
			})
		// Keep the short label centered directly inside the 56 DIP leading
		// slot so it stays beneath the icon as the item grows.
		shortWrap := ui.Row(c).Width(navRailCollapsedItemW).Justify(ui.Center).
			Attach(ui.AnchorTopStart, ui.AnchorTopStart).Start(0).Top(navRailLabelTop).
			Clip().Opacity(shortAlpha)
		shortWrap.Children(func() {
			short := ui.Text(c, dest.Label).SingleLine().MaxWidthPercent(100)
			applyTypeStyleWithTheme(th, short, tokens.LabelSmall, selected && !disabled)
			short.TextColor(shortInk)
		})
		// Constrain the expanded label within [Start(52), itemW-12] so long
		// titles ellipsize cleanly inside their cell.
		longW := max(0, itemW-52-12)
		longWrap := ui.Row(c).Width(longW).Clip().
			Attach(ui.AnchorStart, ui.AnchorStart).Start(52).
			Opacity(longAlpha)
		longWrap.Children(func() {
			long := ui.Text(c, dest.Label).SingleLine().MaxWidthPercent(100)
			applyTypeStyleWithTheme(th, long, tokens.LabelLarge, selected && !disabled)
			long.TextColor(longInk)
		})
	})
	if el.Clicked() && !disabled {
		if o.Selected != nil {
			*o.Selected = i
		}
		if o.Changed != nil {
			o.Changed(i)
		}
	}
	// The button keeps the whole cell's hit area, focus and accessibility;
	// hover, focus and ripple feedback paint only inside the pill.
	attachRippleWithContext(c, th, el, StateLayerOptions{
		Color:    iconInk,
		Disabled: disabled,
		Clip: func(r ui.Rect) (ui.Rect, [4]float32) {
			box, radius := navRailPillRect(r, t, 0, rtl)
			return box, [4]float32{radius, radius, radius, radius}
		},
	})
	return el
}

// navRailPillRect returns the destination's pill within its full-cell box at
// expand progress t, growing elastically with a selected indicator above zero.
func navRailPillRect(r ui.Rect, t, indicator float32, rtl bool) (ui.Rect, float32) {
	baseW := min(r.W, float32(navRailCollapsedItemW))
	slotW := min(r.W, baseW+(r.W-baseW)*t)
	w := slotW
	h := float32(navRailIndicatorHigh) + float32(navRailItemHeight-navRailIndicatorHigh)*t
	if indicator > 0 {
		grow := 0.9 + 0.1*min(indicator, 1.1)
		w, h = w*grow, h*grow
	}
	x := r.X + (slotW-w)/2
	if rtl {
		x = r.X + r.W - slotW + (slotW-w)/2
	}
	y := navRailIconTop*(1-t) + (r.H-h)/2*t
	return ui.Rect{X: x, Y: r.Y + y, W: w, H: h}, h / 2
}

func resolveNavRailExpandedWidth(w float32, d tokens.Density) float32 {
	if !isPositiveFiniteF32(w) {
		return float32(d.NavRailExpandedWidth())
	}
	if w < navRailCollapsedWidth {
		return navRailCollapsedWidth
	}
	return w
}

func navRailItemGap(d tokens.Density) float32 {
	switch d.Normalize() {
	case tokens.DensityDesktop:
		return 4
	case tokens.DensityCompact:
		return 2
	default:
		return 8
	}
}
