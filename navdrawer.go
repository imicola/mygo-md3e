package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// NavigationDrawerItem is one destination of a NavigationDrawer, optionally
// opening a new section.
type NavigationDrawerItem struct {
	// Label is the visible label and the accessible name.
	Label string
	// Icon draws the destination's 24 DIP icon.
	Icon IconDraw
	// Badge is trailing count text, as "12"; empty adds none.
	Badge string
	// Section starts a titled section above this item when it differs from the
	// item before it, adding a divider between sections.
	Section string
	// Disabled disables this destination alone.
	Disabled bool
}

// NavigationDrawerOptions configures a NavigationDrawer. The zero value is an
// inline, enabled drawer that keeps its own open state.
type NavigationDrawerOptions struct {
	// Open lets the caller own the drawer's open state; nil keeps it in the
	// drawer's local state.
	Open *bool
	// Modal makes the drawer a modal overlay with a scrim: it takes the focus,
	// closes on Escape, a scrim press or a drag past the threshold, and gives
	// the focus back to where it was. Without it the drawer is inline and
	// dismissible, animating its width in and out of the layout.
	Modal bool
	// Selected points to the selected destination, -1 for none.
	Selected *int
	// Changed runs after a click or key chooses a destination.
	Changed func(i int)
	// Label names the drawer for assistive technology; "Navigation" when empty.
	Label string
	// Disabled disables every destination.
	Disabled bool
	// Width is the open drawer width in DIP; 320 when zero or negative.
	Width float32
}

// navigationDrawer metrics and timing.
const (
	navDrawerWidth          = 320
	navDrawerItemHeight     = 56
	navDrawerIconSize       = 24
	navDrawerSwipeThreshold = 72
)

type navDrawerOpenKey struct{}

type navDrawerWidthKey struct{}

type navDrawerDragKey struct{}

type navDrawerOffsetKey struct{}

type navDrawerSelectKey struct{}

type navDrawerItemKey struct{ i int }

// NavigationDrawer builds a Material Design 3 Expressive navigation drawer: a
// vertical panel of destinations with optional section titles, dividers, count
// badges and a long SecondaryContainer pill behind the selected destination.
//
// As a modal drawer it shows in a ui.Overlay over a scrim that takes the focus,
// inheriting the active ThemeScope. A scrim press, Escape, or a horizontal drag
// past 72 DIP toward the drawer's edge closes it and gives the focus back to
// where it was; a shorter drag springs back. Right to left anchors the panel on
// the right and mirrors the closing direction. The inline dismissible drawer
// animates its width in and out of the layout, keeps its contents out of the
// focus order while closed, and names itself a region:
//
//	md3e.NavigationDrawer(c, []md3e.NavigationDrawerItem{
//		{Label: "Inbox", Icon: mdicon.Mail.Draw, Badge: "3"},
//		{Label: "Settings", Icon: mdicon.Settings.Draw, Section: "Account"},
//	}, md3e.NavigationDrawerOptions{Modal: true, Open: &app.drawer, Selected: &app.page})
func NavigationDrawer(c *ui.Context, items []NavigationDrawerItem, opts ...NavigationDrawerOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: NavigationDrawer takes at most one NavigationDrawerOptions")
	}
	var o NavigationDrawerOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	width := o.Width
	if !(width > 0) {
		width = navDrawerWidth
	}
	th := CurrentTheme(c)
	label := o.Label
	if label == "" {
		label = "Navigation"
	}
	if o.Modal {
		return modalNavigationDrawer(c, th, items, o, width, label)
	}
	return inlineNavigationDrawer(c, th, items, o, width, label)
}

// modalNavigationDrawer builds the overlay drawer.
func modalNavigationDrawer(c *ui.Context, th *Theme, items []NavigationDrawerItem, o NavigationDrawerOptions, width float32, label string) ui.Element {
	anchor := ui.Box(c).Size(0, 0)
	open := o.Open
	if open == nil {
		open = ui.Local(anchor, navDrawerOpenKey{}, func() bool { return false })
	}
	// A disabled drawer leaves the caller's open state alone: re-enabling it
	// brings the drawer back as it was, rather than silently closing it.
	if o.Disabled || anchor.IsDisabled() || !*open {
		return anchor
	}
	rtl := c.Direction() == ui.RightToLeft
	cs := th.Colors
	scrim := cs.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}

	ui.Overlay(c, func() {
		back := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(scrim.Alpha(0.32)).
			Justify(ui.Start).
			Modal()
		back.Children(func() {
			panel := ui.Column(c).
				Width(width).
				FillHeight().
				Background(SurfaceColorForElevation(th, tokens.Level1)).
				Role(ui.RoleDialog).
				Label(label)
			edge := float32(th.Shapes.Radius(tokens.ShapeLarge))
			if rtl {
				panel.Radius(edge, 0, 0, edge)
			} else {
				panel.Radius(0, edge, edge, 0)
			}
			panel.Disabled(o.Disabled)

			sign := float32(-1)
			if rtl {
				sign = 1
			}
			drag := ui.Local(panel, navDrawerDragKey{}, func() float32 { return 0 })
			var offset float32
			reduce := th.ReduceMotion || c.Preferences().ReduceMotion
			if dx, _, ok := panel.Dragged(); ok {
				// Follow the pointer both ways, dragging out and back, but
				// clamp movement past the open baseline so the panel never
				// drifts away from the drawer's edge.
				*drag += dx
				if *drag*sign < 0 {
					*drag = 0
				}
				offset = panel.Jump(navDrawerOffsetKey{}, *drag)
			} else {
				offset = animateValue(panel, navDrawerOffsetKey{}, 0, th.Animation(motion.FastSpatial), reduce)
				*drag = 0
			}
			if absF32(*drag) >= navDrawerSwipeThreshold {
				*open = false
			}
			panel.Translate(offset, 0)

			navDrawerContent(c, th, items, o)

			if panel.PressedOutside() || back.OverlayShortcut(0, ui.KeyEscape) {
				*open = false
			}
		})
	})
	return anchor
}

// inlineNavigationDrawer builds the dismissible drawer in the layout.
func inlineNavigationDrawer(c *ui.Context, th *Theme, items []NavigationDrawerItem, o NavigationDrawerOptions, width float32, label string) ui.Element {
	root := ui.Column(c).
		FillHeight().
		Clip().
		Background(SurfaceColorForElevation(th, tokens.Level1)).
		Role(ui.RoleRegion).
		Label(label)
	open := o.Open
	if open == nil {
		open = ui.Local(root, navDrawerOpenKey{}, func() bool { return false })
	}
	edge := float32(th.Shapes.Radius(tokens.ShapeLarge))
	if c.Direction() == ui.RightToLeft {
		root.Radius(edge, 0, 0, edge)
	} else {
		root.Radius(0, edge, edge, 0)
	}
	target := float32(0)
	if *open {
		target = width
	}
	openWidth := animateWithTheme(root, th, navDrawerWidthKey{}, target, motion.DefaultSpatial)
	root.Width(openWidth)
	root.Disabled(o.Disabled)

	// Keep the content clipped inside the shrinking panel while it animates
	// closed, so no empty strip shows; once it has settled at zero width the
	// contents leave the focus order and the access tree entirely.
	if *open || openWidth > 0.5 {
		root.Children(func() {
			navDrawerContent(c, th, items, o)
		})
	}
	return root
}

// navDrawerContent builds the drawer's section titles, dividers and
// destinations into the current parent.
func navDrawerContent(c *ui.Context, th *Theme, items []NavigationDrawerItem, o NavigationDrawerOptions) {
	cs := th.Colors
	list := ui.Column(c).FillWidth().Padding(12).Gap(4)
	list.Children(func() {
		lastSection := ""
		for i := range items {
			item := &items[i]
			if item.Section != "" && item.Section != lastSection {
				if lastSection != "" {
					Divider(c, DividerOptions{Inset: 16})
				}
				label := ui.Text(c, item.Section).NoWrap()
				applyTypeStyleWithTheme(th, label, tokens.LabelSmall)
				label.TextColor(cs.OnSurfaceVariant).Padding(16, 8, 4, 8)
				lastSection = item.Section
			}
			navDrawerDestination(c, th, item, i, o)
		}
	})
}

// navDrawerDestination builds one destination of a drawer.
func navDrawerDestination(c *ui.Context, th *Theme, item *NavigationDrawerItem, i int, o NavigationDrawerOptions) {
	cs := th.Colors
	c.Key(navDrawerItemKey{i: i})
	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		FillWidth().
		Height(navDrawerItemHeight).
		Row().
		AlignItems(ui.Center).
		PaddingX(16).
		Gap(12).
		Radius(navDrawerItemHeight / 2).
		Label(item.Label).
		Role(ui.RoleTab)
	disabled := o.Disabled || item.Disabled || el.IsDisabled()
	selected := o.Selected != nil && *o.Selected == i
	target := float32(0)
	if selected {
		target = 1
	}
	pill := animateWithTheme(el, th, navDrawerSelectKey{}, target, motion.DefaultSpatial)

	base := SurfaceColorForElevation(th, tokens.Level1)
	selectedBg := cs.SecondaryContainer
	if selectedBg.A == 0 {
		selectedBg = cs.PrimaryContainer
	}
	iconInk := cs.OnSurfaceVariant
	labelInk := cs.OnSurface
	if selected {
		iconInk = cs.OnSecondaryContainer
		labelInk = cs.OnSecondaryContainer
	}
	if disabled {
		iconInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		labelInk = iconInk
	}
	bg := base
	if selectedBg.A > 0 && pill > 0.001 {
		bg = base.Mix(selectedBg, clampF32(pill, 0, 1))
	}
	el.Disabled(item.Disabled).AccessSelected(selected).Background(bg).TextColor(labelInk)

	el.Children(func() {
		drawIconBox(c, item.Icon, navDrawerIconSize, iconInk)
		txt := ui.Text(c, item.Label).NoWrap().Grow(1)
		applyTypeStyleWithTheme(th, txt, tokens.LabelLarge)
		txt.TextColor(labelInk)
		if item.Badge != "" {
			badge := ui.Text(c, item.Badge).NoWrap().Shrink(0)
			applyTypeStyleWithTheme(th, badge, tokens.LabelMedium)
			badge.TextColor(labelInk)
		}
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
		Radius:   navDrawerItemHeight / 2,
		Disabled: disabled,
	})
}
