package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// TabItem is one tab of PrimaryTabs or SecondaryTabs.
type TabItem struct {
	// Label is the visible text and the accessible name.
	Label string
	// Icon draws an optional 24 DIP leading icon.
	Icon IconDraw
	// Disabled disables this tab alone: the pointer and the arrows, Home and
	// End pass over it.
	Disabled bool
}

// TabsOptions configures PrimaryTabs and SecondaryTabs. The zero value is a
// fixed, enabled tab bar that keeps its own selection.
type TabsOptions struct {
	// Selected points to the chosen tab; nil keeps the selection in the tab
	// bar's local state. The bar writes clicks and the keys into it as it
	// builds.
	Selected *int
	// Changed runs after a click or key chooses another tab.
	Changed func(i int)
	// Scrollable lays the tabs out at their natural widths in a horizontal
	// scroller, scrolling the chosen tab into view as the selection moves,
	// instead of sharing the width equally.
	Scrollable bool
	// Label names the tab list for assistive technology; "Tabs" when empty.
	Label string
	// Disabled disables every tab.
	Disabled bool
}

// Tab bar metrics and timing, in DIP.
const (
	tabsHeight                    = 48
	tabsPaddingX                  = 16
	tabsIconSize                  = 24
	tabsIconGap                   = 8
	tabsIndicatorHeight           = 3
	tabsSecondaryIndicatorHeight  = 2
	tabsIndicatorMinWidth         = 24
	tabsIndicatorMaxWidth         = 48
	tabsIndicatorWidthShare       = 0.4
	tabsSecondaryDividerThickness = 1
)

// Locals and animation channels of a tab bar.
type (
	tabsSelectedKey   struct{}
	tabsLastKey       struct{}
	tabsIndicatorKey  struct{}
	tabsIndicatorXKey struct{}
	tabsIndicatorWKey struct{}
	tabsScrollKey     struct{}
	tabsTabKey        struct{ i int }
)

// tabsIndicatorState is the resting geometry of the sliding indicator,
// relative to the tab bar's box.
type tabsIndicatorState struct {
	x, w float32
	set  bool
}

// PrimaryTabs builds a Material Design 3 Expressive primary tab bar: a
// ui.TabsBase with equal-width or scrollable tabs, an icon and a label each,
// and a rounded Primary bar under the chosen tab that slides along the track
// with the Fast Spatial spring as the choice moves. A click, or the arrows,
// Home or End while a tab has the focus, chooses a tab, past the disabled
// ones, and the horizontal keys mirror right to left. The first laid-out frame
// places the bar at once rather than sliding it in from zero.
//
//	md3e.PrimaryTabs(c, []md3e.TabItem{
//		{Label: "Overview"},
//		{Label: "Activity", Icon: mdicon.Star.Draw},
//	}, md3e.TabsOptions{Selected: &app.tab})
func PrimaryTabs(c *ui.Context, tabs []TabItem, opts ...TabsOptions) ui.Element {
	return buildTabs(c, tabs, opts, true)
}

// SecondaryTabs builds a Material Design 3 Expressive secondary tab bar: the
// same ui.TabsBase behaviour as PrimaryTabs, with a full-width Primary
// indicator sitting on an OutlineVariant divider along the bottom of the
// track. Icon and label tabs stay supported.
func SecondaryTabs(c *ui.Context, tabs []TabItem, opts ...TabsOptions) ui.Element {
	return buildTabs(c, tabs, opts, false)
}

// buildTabs implements both tab bars.
func buildTabs(c *ui.Context, items []TabItem, opts []TabsOptions, primary bool) ui.Element {
	if len(opts) > 1 {
		panic("md3e: PrimaryTabs and SecondaryTabs take at most one TabsOptions")
	}
	var o TabsOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	cs := th.Colors
	label := o.Label
	if label == "" {
		label = "Tabs"
	}

	root := ui.Column(c).FillWidth().Height(tabsHeight)
	selected := o.Selected
	if selected == nil {
		selected = ui.Local(root, tabsSelectedKey{}, func() int { return 0 })
	}
	if len(items) == 0 {
		return root
	}

	lastSelected := ui.Local(root, tabsLastKey{}, func() int { return *selected })
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	var tabs ui.TabsParts
	tabEls := make([]ui.Element, len(items))

	buildList := func() {
		tabs = ui.TabsBase(c, selected, len(items))
		tabs.List.Label(label).Height(tabsHeight)
		if !o.Scrollable {
			tabs.List.FillWidth()
		}
		tabs.List.Children(func() {
			for i := range items {
				item := &items[i]
				c.Key(tabsTabKey{i: i})
				tab := tabs.Tab(i).
					Height(tabsHeight).
					PaddingX(tabsPaddingX).
					Gap(tabsIconGap).
					AlignItems(ui.Center).
					Justify(ui.Center)
				if o.Scrollable {
					tab.Shrink(0)
				} else {
					tab.Grow(1)
				}
				tabEls[i] = tab

				on := *selected == i
				disabled := o.Disabled || item.Disabled || tab.IsDisabled()
				ink := cs.OnSurfaceVariant
				if on {
					ink = cs.Primary
				}
				if disabled {
					ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
				}
				tab.Disabled(o.Disabled || item.Disabled).
					AccessSelected(on).
					TextColor(ink)
				tab.Children(func() {
					if item.Icon != nil {
						drawIconBox(c, item.Icon, tabsIconSize, ink)
					}
					txt := ui.Text(c, item.Label).NoWrap()
					applyTypeStyleWithTheme(th, txt, tokens.LabelLarge, on)
					txt.TextColor(ink)
				})
				radius := th.Shapes.Radius(tokens.ShapeSmall)
				attachRippleWithTheme(th, tab, StateLayerOptions{
					Color:    ink,
					Radius:   radius,
					Disabled: disabled,
				})
			}
		})
	}

	if o.Scrollable {
		root.Children(func() {
			scroll := ui.ScrollHorizontal(c).
				Height(tabsHeight).
				FillWidth().
				Scrollbars(false)
			scroll.Disabled(o.Disabled)
			scroll.Children(func() {
				buildList()
			})
		})
	} else {
		root.Children(func() {
			buildList()
		})
	}

	changed := tabs.List.Changed()
	if changed && o.Changed != nil {
		o.Changed(*selected)
	}

	// A scrollable bar brings the chosen tab into view whenever the choice
	// changes, by click, key or the caller.
	if o.Scrollable && *selected != *lastSelected {
		*lastSelected = *selected
		if el := tabEls[*selected]; el.Valid() {
			opts := ui.ScrollIntoViewOptions{Inline: ui.ScrollNearest}
			if !th.ReduceMotion {
				anim := th.Animation(motion.DefaultSpatial)
				opts.Animation = &anim
			}
			el.ScrollIntoView(opts)
		}
	}

	// Slide the indicator to the chosen tab's last laid-out box relative to
	// tabs.List, so scrolling moves and clips the indicator together with the
	// tabs inside ScrollHorizontal. Before the first layout there is no box
	// yet, so the bar appears where it belongs instead of flying in from zero.
	indicator := ui.Local(tabs.List, tabsIndicatorKey{}, func() tabsIndicatorState { return tabsIndicatorState{} })
	listBox := tabs.List.LayoutBounds()
	tabBox := tabEls[*selected].LayoutBounds()
	if listBox.W > 0 && tabBox.W > 0 {
		inset, width := tabsIndicatorBand(primary, tabBox.W)
		targetX := tabBox.X - listBox.X + inset
		if !indicator.set {
			indicator.set = true
			indicator.x = tabs.List.Jump(tabsIndicatorXKey{}, targetX)
			indicator.w = tabs.List.Jump(tabsIndicatorWKey{}, width)
		} else {
			indicator.x = animateValue(tabs.List, tabsIndicatorXKey{}, targetX, th.Animation(motion.FastSpatial), reduce)
			indicator.w = animateValue(tabs.List, tabsIndicatorWKey{}, width, th.Animation(motion.FastSpatial), reduce)
		}
	} else if !indicator.set {
		// The first frame has not laid the tabs out yet; build once more so
		// the indicator can land in place without flying in from zero.
		c.Invalidate()
	}

	disabledInk := cs.Primary
	if o.Disabled {
		disabledInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	if !primary && cs.OutlineVariant.A > 0 {
		root.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if r.W <= 0 || r.H <= 0 {
				return
			}
			p.Fill(ui.Rect{
				X: r.X,
				Y: r.Y + r.H - tabsSecondaryDividerThickness,
				W: r.W,
				H: tabsSecondaryDividerThickness,
			}, cs.OutlineVariant, 0)
		})
	}
	tabs.List.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !indicator.set || r.W <= 0 || r.H <= 0 {
			return
		}
		height := float32(tabsIndicatorHeight)
		if !primary {
			height = tabsSecondaryIndicatorHeight
		}
		y := r.Y + r.H - height
		if !primary {
			y -= tabsSecondaryDividerThickness
		}
		half := height / 2
		p.FillRadii(ui.Rect{
			X: r.X + indicator.x,
			Y: y,
			W: min(indicator.w, max(r.W-indicator.x, 0)),
			H: height,
		}, [4]float32{half, half, half, half}, disabledInk)
	})
	return root
}

// tabsIndicatorBand returns the inset and width of the primary bar inside a
// tab of width tabW: a rounded piece of 40% of the tab, between 24 and 48 DIP,
// centered under its label. The secondary bar fills the whole tab.
func tabsIndicatorBand(primary bool, tabW float32) (inset, width float32) {
	if !primary {
		return 0, tabW
	}
	w := tabW * tabsIndicatorWidthShare
	if w < tabsIndicatorMinWidth {
		w = tabsIndicatorMinWidth
	}
	if w > tabsIndicatorMaxWidth {
		w = tabsIndicatorMaxWidth
	}
	if w > tabW {
		w = tabW
	}
	return (tabW - w) / 2, w
}
