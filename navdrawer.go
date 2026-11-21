package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
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
	navDrawerDragSlop       = 6
)

type navDrawerOpenKey struct{}

type navDrawerStateKey struct{}

type navDrawerShowKey struct{}

type navDrawerWidthKey struct{}

type navDrawerDragKey struct{}

type navDrawerOffsetKey struct{}

type navDrawerSelectKey struct{}

type navDrawerItemKey struct{ i int }

type navDrawerState struct {
	shown      bool
	appeared   bool
	exitOffset float32
}

type navDrawerDragState struct {
	dragging bool
	offset   float32
	travel   float32
	suppress bool
}

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
	width := sanitizePositiveF32(o.Width, navDrawerWidth)
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

// modalNavigationDrawer builds the overlay drawer with enter/exit slide
// motion and scrim fade, plus horizontal swipe-to-dismiss across the panel
// and its destinations.
func modalNavigationDrawer(c *ui.Context, th *Theme, items []NavigationDrawerItem, o NavigationDrawerOptions, width float32, label string) ui.Element {
	anchor := ui.Box(c).Size(0, 0)
	open := o.Open
	if open == nil {
		open = ui.Local(anchor, navDrawerOpenKey{}, func() bool { return false })
	}
	st := ui.Local(anchor, navDrawerStateKey{}, func() navDrawerState { return navDrawerState{} })
	drag := ui.Local(anchor, navDrawerDragKey{}, func() navDrawerDragState { return navDrawerDragState{} })

	// A disabled drawer leaves the caller's open state alone: re-enabling it
	// brings the drawer back as it was, rather than silently closing it.
	if o.Disabled || anchor.IsDisabled() {
		*st = navDrawerState{}
		*drag = navDrawerDragState{}
		return anchor
	}
	if !*open && !st.shown {
		*st = navDrawerState{}
		*drag = navDrawerDragState{}
		return anchor
	}
	if *open {
		if !st.shown {
			*drag = navDrawerDragState{}
			st.exitOffset = 0
		}
		st.shown = true
		st.exitOffset = 0
	}

	exiting := !*open
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	var show float32
	switch {
	case *open && !st.appeared:
		st.appeared = true
		if reduce {
			show = anchor.Jump(navDrawerShowKey{}, 1)
		} else {
			anchor.Jump(navDrawerShowKey{}, 0)
			show = 0
			c.Invalidate()
		}
	case exiting && reduce:
		st.shown, st.appeared = false, false
		st.exitOffset = 0
		*drag = navDrawerDragState{}
		c.Invalidate()
		return anchor
	case exiting:
		show = animateValue(anchor, navDrawerShowKey{}, 0, th.Animation(motion.FastSpatial), false)
		if anchor.Arrived(navDrawerShowKey{}) || (anchor.Settled(navDrawerShowKey{}) && show <= 0.001) {
			st.shown, st.appeared = false, false
			st.exitOffset = 0
			*drag = navDrawerDragState{}
			c.Invalidate()
			return anchor
		}
	default:
		show = animateValue(anchor, navDrawerShowKey{}, 1, th.Animation(motion.FastSpatial), false)
	}

	rtl := c.Direction() == ui.RightToLeft
	cs := th.Colors
	scrim := cs.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}
	sign := float32(-1)
	if rtl {
		sign = 1
	}

	ui.Overlay(c, func() {
		back := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(scrim.Alpha(0.32 * clampF32(show, 0, 1))).
			Justify(ui.Start)
		if !exiting {
			back.Modal()
		}
		back.IgnoreInput(exiting).AccessHidden(exiting)

		back.Children(func() {
			panel := ui.Column(c).
				Width(width).
				FillHeight().
				DimDisabled(false).
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

			drag.dragging = false
			if !exiting {
				if dx, _, ok := panel.Dragged(); ok {
					drag.dragging = true
					drag.offset += dx
					drag.travel += absF32(dx)
					if drag.travel >= navDrawerDragSlop {
						drag.suppress = true
					}
				}
			}

			panel.Children(func() {
				navDrawerContent(c, th, items, o, drag, exiting)
			})

			var liveOffset float32
			dismissTriggered := false
			if !exiting && drag.dragging {
				if drag.offset*sign < 0 {
					drag.offset = 0
				}
				liveOffset = panel.Jump(navDrawerOffsetKey{}, drag.offset)
				if absF32(drag.offset) >= navDrawerSwipeThreshold {
					st.exitOffset = liveOffset
					*open = false
					*drag = navDrawerDragState{}
					dismissTriggered = true
					c.Invalidate()
				}
			} else {
				drag.offset = 0
				drag.travel = 0
				drag.suppress = false
				if !exiting {
					liveOffset = animateValue(panel, navDrawerOffsetKey{}, 0, th.Animation(motion.FastSpatial), reduce)
				}
			}

			showClamped := clampF32(show, 0, 1)
			var tx float32
			if exiting || dismissTriggered {
				tx = st.exitOffset + sign*(1-showClamped)*max(width-absF32(st.exitOffset), 0)
			} else {
				tx = liveOffset + sign*(1-showClamped)*width
			}
			panel.Translate(tx, 0)

			// Absorb panel clicks so presses on the drawer surface or disabled
			// destinations do not reach the scrim backdrop, and use back.Clicked()
			// rather than PressedOutside() so independent overlays above the
			// drawer (such as a Snackbar) do not dismiss it.
			panel.Clicked()

			if !exiting && !dismissTriggered && (back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape)) {
				*open = false
			}
		})
	})
	return anchor
}

// inlineNavigationDrawer builds the dismissible drawer in the layout.
func inlineNavigationDrawer(c *ui.Context, th *Theme, items []NavigationDrawerItem, o NavigationDrawerOptions, width float32, label string) ui.Element {
	root := ui.Column(c).
		DimDisabled(false).
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
	if openWidth < 0 {
		openWidth = 0
	}
	root.Width(openWidth)
	root.Disabled(o.Disabled)
	exiting := !*open
	root.IgnoreInput(exiting).AccessHidden(exiting)

	// Keep the content clipped inside the shrinking panel while it animates
	// closed, so no empty strip shows; make closing content inert and hidden
	// immediately so it leaves focus order and the access tree as it shuts.
	if *open || openWidth > 0.5 {
		root.Children(func() {
			navDrawerContent(c, th, items, o, nil, exiting)
		})
	}
	return root
}

// navDrawerContent builds the drawer's scrollable section titles, dividers and
// destinations into the current parent.
func navDrawerContent(c *ui.Context, th *Theme, items []NavigationDrawerItem, o NavigationDrawerOptions, drag *navDrawerDragState, exiting bool) {
	cs := th.Colors
	scroll := ui.Scroll(c).Grow(1).FillWidth().Scrollbars(false)
	scroll.Children(func() {
		list := ui.Column(c).FillWidth().Padding(12).Gap(4).FocusGroup(ui.Vertical)
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
				navDrawerDestination(c, th, item, i, o, drag, exiting)
			}
		})
	})
}

// navDrawerDestination builds one destination of a drawer.
func navDrawerDestination(c *ui.Context, th *Theme, item *NavigationDrawerItem, i int, o NavigationDrawerOptions, drag *navDrawerDragState, exiting bool) {
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
	pill := animateWithTheme(el, th, navDrawerSelectKey{}, target, motion.DefaultEffects)

	if !exiting && drag != nil {
		if dx, _, ok := el.Dragged(); ok {
			drag.dragging = true
			drag.offset += dx
			drag.travel += absF32(dx)
			if drag.travel >= navDrawerDragSlop {
				drag.suppress = true
			}
		}
	}

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
	if p := clampF32(pill, 0, 1); p > 0.001 {
		if disabled {
			bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity) * p)
		} else if selectedBg.A > 0 {
			bg = base.Mix(selectedBg, p)
		}
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
	suppressClick := drag != nil && drag.suppress
	if el.Clicked() && !disabled && !suppressClick && !exiting {
		if o.Selected != nil {
			*o.Selected = i
		}
		if o.Changed != nil {
			o.Changed(i)
		}
	}
	attachRippleWithContext(c, th, el, StateLayerOptions{
		Color:    iconInk,
		Radius:   navDrawerItemHeight / 2,
		Disabled: disabled,
	})
}
