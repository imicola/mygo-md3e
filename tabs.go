package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// TabItem is one tab of PrimaryTabs, SecondaryTabs, PillTabs, or SegmentedTabs.
type TabItem struct {
	// Key optionally identifies the tab across dynamic reordering or edits.
	Key any
	// Label is the visible text and the accessible name.
	Label string
	// Icon draws an optional leading icon.
	Icon IconDraw
	// Badge optionally displays a compact count or short text badge beside the
	// label (rendered by PillTabs and SegmentedTabs).
	Badge string
	// BadgeDot optionally shows a 6 DIP indicator dot when Badge is empty
	// (rendered by PillTabs and SegmentedTabs).
	BadgeDot bool
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
	x, w       float32
	set        bool
	builtRelX  float32
	builtTabW  float32
	builtListW float32
	paintedTab ui.Rect
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
//		{Label: "Activity", Icon: starIcon},
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
	activeInk, inactiveInk := th.Colors.Primary, th.Colors.OnSurfaceVariant
	disabledTabInk := th.Colors.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	dividerInk := th.Colors.OutlineVariant
	label := o.Label
	if label == "" {
		label = "Tabs"
	}
	tabH, padX, iconSize, iconGap := tabsMetricsForDensity(th.Density)

	root := ui.Column(c).FillWidth().Height(tabH).DimDisabled(false)
	root.Disabled(o.Disabled)
	selected := o.Selected
	if selected == nil {
		selected = ui.Local(root, tabsSelectedKey{}, func() int { return 0 })
	}
	// -1 so an initial selection starting off-screen scrolls into view once
	// the scrollable tab strip lays out.
	lastSelected := ui.Local(root, tabsLastKey{}, func() int { return -1 })
	scrollState := ui.Local(root, tabsScrollKey{}, func() ui.ScrollState { return ui.ScrollState{} })
	if len(items) == 0 {
		*lastSelected = -1
		return root
	}
	*selected = max(0, min(*selected, len(items)-1))

	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	rtl := c.Direction() == ui.RightToLeft

	var tabs ui.TabsParts
	var stackTabEls [12]ui.Element
	var tabEls []ui.Element
	if len(items) <= len(stackTabEls) {
		tabEls = stackTabEls[:len(items)]
	} else {
		tabEls = make([]ui.Element, len(items))
	}

	buildList := func() {
		tabs = ui.TabsBase(c, selected, len(items))
		tabs.List.DimDisabled(false).Label(label).Height(tabH)
		if !o.Scrollable {
			tabs.List.FillWidth()
		}
		tabs.List.Children(func() {
			for i := range items {
				item := &items[i]
				var key any = tabsTabKey{i: i}
				if item.Key != nil {
					key = item.Key
				}
				c.Key(key)
				tab := tabs.Tab(i).
					DimDisabled(false).
					Height(tabH).
					PaddingX(padX).
					Gap(iconGap).
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
				ink := inactiveInk
				if on {
					ink = activeInk
				}
				if disabled {
					ink = disabledTabInk
				}
				tab.Disabled(o.Disabled || item.Disabled).
					AccessSelected(on).
					TextColor(ink)
				tab.Children(func() {
					if item.Icon != nil {
						drawIconBox(c, item.Icon, iconSize, ink)
					}
					txt := ui.Text(c, item.Label).NoWrap()
					applyTypeStyleWithTheme(th, txt, tokens.LabelLarge, on)
					txt.TextColor(ink)
				})
				radius := th.Shapes.Radius(tokens.ShapeSmall)
				attachRippleWithContext(c, th, tab, StateLayerOptions{
					Color:    ink,
					Radius:   radius,
					Disabled: disabled,
				})
			}
		})
	}

	var scrollEl ui.Element
	if o.Scrollable {
		root.Children(func() {
			scrollEl = ui.ScrollHorizontal(c).
				TrackScroll(scrollState).
				DimDisabled(false).
				Height(tabH).
				FillWidth().
				Scrollbars(false)
			scrollEl.Disabled(o.Disabled)
			scrollEl.Children(func() {
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

	// Slide the indicator to the chosen tab's last laid-out box relative to
	// tabs.List, so scrolling moves and clips the indicator together with the
	// tabs inside ScrollHorizontal. Before the first layout there is no box
	// yet, so the bar appears where it belongs instead of flying in from zero.
	indicator := ui.Local(tabs.List, tabsIndicatorKey{}, func() tabsIndicatorState { return tabsIndicatorState{} })
	indicator.paintedTab = ui.Rect{}
	listBox := tabs.List.LayoutBounds()
	selTab := tabEls[*selected]
	tabBox := selTab.LayoutBounds()
	if listBox.W > 0 && tabBox.W > 0 {
		relX := tabBox.X - listBox.X
		inset, width := tabsIndicatorBand(primary, tabBox.W)
		targetX := relX + inset
		indicator.builtRelX, indicator.builtTabW, indicator.builtListW = relX, tabBox.W, listBox.W
		if !indicator.set {
			indicator.set = true
			indicator.x = tabs.List.Jump(tabsIndicatorXKey{}, targetX)
			indicator.w = tabs.List.Jump(tabsIndicatorWKey{}, width)
		} else {
			indicator.x = animateValue(tabs.List, tabsIndicatorXKey{}, targetX, th.Animation(motion.FastSpatial), reduce)
			indicator.w = animateValue(tabs.List, tabsIndicatorWKey{}, width, th.Animation(motion.FastSpatial), reduce)
		}

		// A scrollable bar brings the chosen tab into horizontal view using its
		// own ScrollHorizontal state so enclosing vertical pages never scroll.
		if o.Scrollable && *selected != *lastSelected {
			viewW := scrollEl.LayoutBounds().W
			if viewW > 0 {
				firstAlign := *lastSelected < 0
				*lastSelected = *selected
				relStart := relX
				if rtl {
					relStart = (listBox.X + listBox.W) - (tabBox.X + tabBox.W)
				}
				relEnd := relStart + tabBox.W
				scrollTarget := scrollState.X
				needScroll := false
				if relStart < scrollState.X {
					scrollTarget = relStart
					needScroll = true
				} else if relEnd > scrollState.X+viewW {
					scrollTarget = relEnd - viewW
					needScroll = true
				}
				if needScroll {
					clamped := clampF32(scrollTarget, 0, scrollState.MaxX)
					if absF32(scrollState.X-clamped) > 0.5 {
						if firstAlign || reduce {
							scrollState.X = clamped
						} else {
							scrollState.ScrollTo(clamped, 0, th.Animation(motion.DefaultSpatial))
						}
						c.Invalidate()
					}
				}
			}
		}
	} else if !indicator.set {
		// The first frame has not laid the tabs out yet; build once more so
		// the indicator can land in place without flying in from zero.
		c.Invalidate()
	}

	tabList := tabs.List
	indicatorInk := activeInk
	if o.Disabled || tabList.IsDisabled() || selTab.IsDisabled() {
		indicatorInk = disabledTabInk
	}
	if !primary && dividerInk.A > 0 {
		root.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if !root.Valid() || r.W <= 0 || r.H <= 0 {
				return
			}
			p.Fill(ui.Rect{
				X: r.X,
				Y: r.Y + r.H - tabsSecondaryDividerThickness,
				W: r.W,
				H: tabsSecondaryDividerThickness,
			}, dividerInk, 0)
		})
	}
	srv := c.Services()
	if selTab.Valid() {
		selTab.DrawOver(func(p *ui.Painter, tr ui.Rect) {
			if !selTab.Valid() || tr.W <= 0 || tr.H <= 0 {
				return
			}
			indicator.paintedTab = tr
		})
	}
	tabList.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !tabList.Valid() || !indicator.set || r.W <= 0 || r.H <= 0 {
			return
		}
		if pt := indicator.paintedTab; pt.W > 0 {
			if absF32(pt.W-indicator.builtTabW) > 0.5 ||
				absF32((pt.X-r.X)-indicator.builtRelX) > 0.5 ||
				absF32(r.W-indicator.builtListW) > 0.5 {
				srv.Invalidate()
			}
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
		}, [4]float32{half, half, half, half}, indicatorInk)
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

func tabsMetricsForDensity(d tokens.Density) (height, padX, iconSize, iconGap float32) {
	height = float32(d.TabHeight())
	switch d.Normalize() {
	case tokens.DensityDesktop:
		return height, 12, 20, 6
	case tokens.DensityCompact:
		return height, 10, 18, 6
	default:
		return height, tabsPaddingX, tabsIconSize, tabsIconGap
	}
}
