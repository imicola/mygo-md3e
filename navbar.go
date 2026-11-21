package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// NavigationDestination is one destination of a NavigationBar or
// NavigationRail.
type NavigationDestination struct {
	// Label is the visible label and the accessible name.
	Label string
	// Icon draws the destination's 24 DIP icon.
	Icon IconDraw
	// Badge adds the large count badge on the icon, as "3" or "999+"; empty
	// adds none.
	Badge string
	// SmallBadge shows the 6 DIP dot instead of a count.
	SmallBadge bool
	// Disabled disables this destination alone.
	Disabled bool
}

// NavigationLabelMode selects when a navigation destination shows its label.
type NavigationLabelMode int

const (
	// NavigationLabelsAlways shows every destination's label.
	NavigationLabelsAlways NavigationLabelMode = iota
	// NavigationLabelsSelectedOnly shows only the selected destination's label.
	NavigationLabelsSelectedOnly
	// NavigationLabelsNever hides every label, leaving the icons and their
	// accessible names.
	NavigationLabelsNever
)

// NavigationBarOptions configures a NavigationBar. The zero value is an
// enabled bar with always-visible labels that keeps no selection of its own.
type NavigationBarOptions struct {
	// Selected points to the selected destination, -1 for none. The bar writes
	// clicks and keyboard activation into it as it builds.
	Selected *int
	// Changed runs after a click or key chooses a destination, even when
	// Selected is nil.
	Changed func(i int)
	// Labels selects the label visibility policy.
	Labels NavigationLabelMode
	// Label names the bar for assistive technology; "Navigation" when empty.
	Label string
	// Disabled disables every destination.
	Disabled bool
}

// NavigationBar metrics in DIP.
const (
	navBarHeight        = 80
	navBarIndicatorWide = 64
	navBarIndicatorHigh = 32
	navBarIconSize      = 24
	// navBarIndicatorTop is the pill's top within the bar.
	navBarIndicatorTop = 12
	// navBarLabelTop is the label's top within the bar.
	navBarLabelTop = 48
)

type navBarIndicatorKey struct{}

type navBarItemKey struct{ i int }

// NavigationBar builds a Material Design 3 Expressive navigation bar: a row of
// destinations along the bottom of a window, each an icon and an optional
// label, with an elastic SecondaryContainer pill behind the selected icon. A
// click, or Enter or Space on a focused destination, writes Selected and calls
// Changed; the horizontal arrows move the focus among the destinations and
// mirror right to left. Count and dot badges anchor on the destination icons.
//
// The pill expands and fades with the Fast Spatial spring, and reduced motion
// settles it at once:
//
//	md3e.NavigationBar(c, []md3e.NavigationDestination{
//		{Label: "Home", Icon: mdicon.Home.Draw},
//		{Label: "Inbox", Icon: mdicon.Mail.Draw, Badge: "3"},
//	}, md3e.NavigationBarOptions{Selected: &app.tab})
func NavigationBar(c *ui.Context, destinations []NavigationDestination, opts ...NavigationBarOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: NavigationBar takes at most one NavigationBarOptions")
	}
	var o NavigationBarOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	cs := th.Colors
	label := o.Label
	if label == "" {
		label = "Navigation"
	}

	bar := ui.Row(c).
		DimDisabled(false).
		Height(navBarHeight).
		FillWidth().
		Background(SurfaceColorForElevation(th, tokens.Level2)).
		Role(ui.RoleNavigation).
		FocusGroup(ui.Horizontal).
		Label(label)
	bar.Disabled(o.Disabled)

	bar.Children(func() {
		for i := range destinations {
			navBarDestination(c, th, cs, &destinations[i], i, o)
		}
	})
	return bar
}

// navBarDestination builds one destination of a NavigationBar.
func navBarDestination(c *ui.Context, th *Theme, cs tokens.ColorScheme, dest *NavigationDestination, i int, o NavigationBarOptions) {
	c.Key(navBarItemKey{i: i})
	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Grow(1).
		Height(navBarHeight).
		Label(dest.Label).
		Role(ui.RoleTab)
	disabled := o.Disabled || dest.Disabled || el.IsDisabled()
	selected := o.Selected != nil && *o.Selected == i

	target := float32(0)
	if selected {
		target = 1
	}
	indicator := animateWithTheme(el, th, navBarIndicatorKey{}, target, motion.FastSpatial)
	iconInk := cs.OnSurfaceVariant
	labelInk := cs.OnSurfaceVariant
	if selected {
		iconInk = cs.OnSecondaryContainer
		labelInk = cs.OnSurface
	}
	if disabled {
		iconInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		labelInk = iconInk
	}
	el.Disabled(dest.Disabled).AccessSelected(selected)

	showLabel := o.Labels == NavigationLabelsAlways ||
		(o.Labels == NavigationLabelsSelectedOnly && selected)
	el.Children(func() {
		// The pill and the label are placed by their exact boxes, so their
		// geometry does not depend on whether a label is shown.
		ui.Box(c).Size(navBarIndicatorWide, navBarIndicatorHigh).Center().
			Attach(ui.AnchorTop, ui.AnchorTop).Top(navBarIndicatorTop).
			Draw(func(p *ui.Painter, r ui.Rect) {
				if indicator <= 0.001 || r.W <= 0 || r.H <= 0 {
					return
				}
				alpha := clampF32(indicator, 0, 1)
				scale := min(indicator, 1.1)
				w := r.W * scale
				box := ui.Rect{X: r.X + (r.W-w)/2, Y: r.Y, W: w, H: r.H}
				pill := cs.SecondaryContainer.Alpha(alpha)
				if disabled {
					pill = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity) * alpha)
				}
				p.FillRadii(box, [4]float32{r.H / 2, r.H / 2, r.H / 2, r.H / 2}, pill)
			}).
			Children(func() {
				badgedIcon(c, dest, navBarIconSize, iconInk)
			})
		if showLabel {
			txt := ui.Text(c, dest.Label).NoWrap()
			applyTypeStyleWithTheme(th, txt, tokens.LabelMedium)
			txt.TextColor(labelInk)
			txt.Attach(ui.AnchorTop, ui.AnchorTop).Top(navBarLabelTop)
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
	// The whole cell stays the hot zone, focus target and accessible tab; the
	// hover, focus and ripple feedback paint only in the pill.
	attachRippleWithTheme(th, el, StateLayerOptions{
		Color:    iconInk,
		Disabled: disabled,
		Clip: func(r ui.Rect) (ui.Rect, [4]float32) {
			box := ui.Rect{
				X: r.X + (r.W-navBarIndicatorWide)/2,
				Y: r.Y + navBarIndicatorTop,
				W: navBarIndicatorWide,
				H: navBarIndicatorHigh,
			}
			radius := float32(navBarIndicatorHigh / 2)
			return box, [4]float32{radius, radius, radius, radius}
		},
	})
}

// badgedIcon draws a destination's icon in ink, with its count or dot badge
// anchored on the icon's top-end corner.
func badgedIcon(c *ui.Context, dest *NavigationDestination, size float32, ink ui.Color) {
	draw := func() { drawIconBox(c, dest.Icon, size, ink) }
	switch {
	case dest.Badge != "":
		BadgedBox(c, draw, func() ui.Element {
			return Badge(c, dest.Badge, BadgeOptions{Label: dest.Label + " " + dest.Badge})
		}, BadgedBoxOptions{Label: dest.Label})
	case dest.SmallBadge:
		BadgedBox(c, draw, func() ui.Element {
			return SmallBadge(c, BadgeOptions{Label: dest.Label + " new"})
		}, BadgedBoxOptions{Label: dest.Label})
	default:
		draw()
	}
}
