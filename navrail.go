package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
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
)

type navRailWidthKey struct{}

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
	target := float32(navRailCollapsedWidth)
	if *expanded {
		target = navRailExpandedWidth
	}
	root.Width(animateWithTheme(root, th, navRailWidthKey{}, target, motion.DefaultSpatial))

	open := *expanded
	root.Children(func() {
		if o.Header != nil {
			ui.Box(c).FillWidth().Padding(12).Children(func() {
				o.Header()
			})
		}
		items := ui.Column(c).Grow(1).PaddingX(12).Gap(8).Justify(o.Align)
		items.Children(func() {
			for i := range destinations {
				navRailDestination(c, th, cs, &destinations[i], i, o, open)
			}
		})
	})
	return root
}

type navRailExpandedKey struct{}

// navRailDestination builds one destination of a NavigationRail.
func navRailDestination(c *ui.Context, th *Theme, cs tokens.ColorScheme, dest *NavigationDestination, i int, o NavigationRailOptions, expanded bool) {
	c.Key(navRailItemKey{i: i})
	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		FillWidth().
		Height(navRailItemHeight).
		Gap(2).
		Label(dest.Label).
		Role(ui.RoleTab)
	disabled := o.Disabled || dest.Disabled || el.IsDisabled()
	selected := o.Selected != nil && *o.Selected == i
	target := float32(0)
	if selected {
		target = 1
	}
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

	if expanded {
		el.Row().AlignItems(ui.Center).PaddingStart(16).PaddingEnd(12).Gap(12).Radius(navRailItemHeight / 2)
	} else {
		el.Column().Justify(ui.Center).AlignItems(ui.Center)
	}

	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if indicator <= 0.001 || r.W <= 0 || r.H <= 0 {
			return
		}
		alpha := clampF32(indicator, 0, 1)
		box := r
		if expanded {
			box = ui.Rect{X: r.X, Y: r.Y + (r.H-navRailItemHeight)/2, W: r.W, H: navRailItemHeight}
		} else {
			w := max(r.W, navRailItemHeight)
			box = ui.Rect{
				X: r.X + (r.W-w)/2,
				Y: r.Y + (r.H-navRailIndicatorHigh)/2,
				W: w,
				H: navRailIndicatorHigh,
			}
		}
		radius := box.H / 2
		p.FillRadii(box, [4]float32{radius, radius, radius, radius}, cs.SecondaryContainer.Alpha(alpha))
	})

	el.Children(func() {
		badgedIcon(c, dest, navRailIconSize, iconInk)
		txt := ui.Text(c, dest.Label).NoWrap()
		if expanded {
			applyTypeStyleWithTheme(th, txt, tokens.LabelLarge)
			txt.Grow(1)
		} else {
			applyTypeStyleWithTheme(th, txt, tokens.LabelSmall)
		}
		txt.TextColor(labelInk)
	})
	if el.Clicked() && !disabled {
		if o.Selected != nil {
			*o.Selected = i
		}
		if o.Changed != nil {
			o.Changed(i)
		}
	}
	attachRippleWithTheme(th, el, StateLayerOptions{
		Color:    iconInk,
		Radius:   navRailItemHeight / 2,
		Disabled: disabled,
	})
}
