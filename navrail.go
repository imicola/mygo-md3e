package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
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
	// navRailCollapsedItemW is a destination's width in the collapsed rail.
	navRailCollapsedItemW = navRailCollapsedWidth - 2*navRailPaddingX
	// navRailIconTop is the icon's leading slot offset in the collapsed rail;
	// the expanded pill centers it.
	navRailIconTop = 4
	// navRailLabelTop is the short label's top in the collapsed rail.
	navRailLabelTop = 38
)

type navRailExpandKey struct{}

type navRailIndicatorKey struct{}

type navRailItemKey struct{ i int }

// NavigationRail builds a Material Design 3 Expressive navigation rail: a
// vertical strip of destinations along the start edge of a window. Collapsed,
// it is 80 DIP wide and stacks each icon over its short label; expanded, it
// grows to 256 DIP with the icon and label in a row on a long
// SecondaryContainer pill behind the selected destination. The width springs
// between the two with the Default Spatial spring, clipped while it moves, and
// reduced motion changes it at once.
//
// An optional Header slot holds a FAB, and the arrows move the focus among the
// destinations, Enter or Space choosing:
//
//	md3e.NavigationRail(c, []md3e.NavigationDestination{
//		{Label: "Home", Icon: mdicon.Home.Draw},
//		{Label: "Mail", Icon: mdicon.Mail.Draw, Badge: "3"},
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
	root.Disabled(o.Disabled)
	expanded := o.Expanded
	if expanded == nil {
		expanded = ui.Local(root, navRailExpandedKey{}, func() bool { return false })
	}
	expandTarget := float32(0)
	if *expanded {
		expandTarget = 1
	}
	// One spring drives both the rail's width and every destination's pill,
	// icon and label, so the mid-transition frames stay aligned rather than
	// switching layouts on the boolean.
	expand := animateWithTheme(root, th, navRailExpandKey{}, expandTarget, motion.DefaultSpatial)
	root.Width(navRailCollapsedWidth + (navRailExpandedWidth-navRailCollapsedWidth)*expand)

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
				ui.Column(c).FillWidth().PaddingX(navRailPaddingX).Gap(8).Children(func() {
					for i := range destinations {
						navRailDestination(c, th, cs, &destinations[i], i, o, expand)
					}
				})
			})
		})
	})
	return root
}

type navRailExpandedKey struct{}

type navRailScrollKey struct{}

// navRailDestination builds one destination of a NavigationRail. expand is the
// spring's 0 (collapsed) to 1 (expanded) progress, shared by every destination
// and the rail's width.
func navRailDestination(c *ui.Context, th *Theme, cs tokens.ColorScheme, dest *NavigationDestination, i int, o NavigationRailOptions, expand float32) {
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
	target := float32(0)
	if selected {
		target = 1
	}
	// The selection pill grows and fades with the Fast Spatial spring, so it
	// pops elastically instead of only cross-fading.
	indicator := animateWithTheme(el, th, navRailIndicatorKey{}, target, motion.FastSpatial)

	iconInk := cs.OnSurfaceVariant
	labelInk := cs.OnSurfaceVariant
	if selected {
		iconInk = cs.OnSecondaryContainer
		labelInk = cs.OnSecondaryContainer
	}
	if disabled {
		iconInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		labelInk = iconInk
	}
	el.Disabled(dest.Disabled).AccessSelected(selected)

	pillAlpha := clampF32(indicator, 0, 1)
	pillColor := cs.SecondaryContainer.Alpha(pillAlpha)
	if disabled {
		pillColor = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity) * pillAlpha)
	}

	t := clampF32(expand, 0, 1)
	el.Row().AlignItems(ui.Center).Radius(navRailItemHeight / 2)
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if indicator <= 0.001 || r.W <= 0 || r.H <= 0 {
			return
		}
		box, radius := navRailPillRect(r, t, indicator)
		p.FillRadii(box, [4]float32{radius, radius, radius, radius}, pillColor)
	})

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
		// The short label fades out under the icon while the long one fades in
		// beside it; both are laid out as the pill grows, so neither is clipped
		// or jumps to the next destination mid-transition.
		short := ui.Text(c, dest.Label).NoWrap()
		applyTypeStyleWithTheme(th, short, tokens.LabelSmall)
		short.TextColor(labelInk)
		short.Attach(ui.AnchorTop, ui.AnchorTop).Top(navRailLabelTop)
		short.Opacity(1 - t)
		long := ui.Text(c, dest.Label).NoWrap()
		applyTypeStyleWithTheme(th, long, tokens.LabelLarge)
		long.TextColor(labelInk)
		long.Attach(ui.AnchorStart, ui.AnchorStart).Start(52)
		long.Opacity(t)
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
	attachRippleWithTheme(th, el, StateLayerOptions{
		Color:    iconInk,
		Disabled: disabled,
		Clip: func(r ui.Rect) (ui.Rect, [4]float32) {
			box, radius := navRailPillRect(r, t, 0)
			return box, [4]float32{radius, radius, radius, radius}
		},
	})
}

// navRailPillRect returns the destination's pill within its full-cell box at
// expand progress t, growing elastically with a selected indicator above zero.
func navRailPillRect(r ui.Rect, t, indicator float32) (ui.Rect, float32) {
	w := navRailCollapsedItemW + (r.W-navRailCollapsedItemW)*t
	h := float32(navRailIndicatorHigh) + float32(navRailItemHeight-navRailIndicatorHigh)*t
	if indicator > 0 {
		grow := 0.9 + 0.1*min(indicator, 1.1)
		w, h = w*grow, h*grow
	}
	y := navRailIconTop*(1-t) + (r.H-h)/2*t
	return ui.Rect{X: r.X + (r.W-w)/2, Y: r.Y + y, W: w, H: h}, h / 2
}
