package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// ListItem describes one row of a List: its text, its leading and trailing
// slots, and what choosing it does.
type ListItem struct {
	// Key is the row's stable identity across rebuilds, as an item's ID. Rows
	// without one are keyed by their index.
	Key any
	// Overline is the small text above the headline.
	Overline string
	// Headline is the row's main text and its accessible name.
	Headline string
	// SupportingText is the secondary text below the headline.
	SupportingText string
	// Icon draws a 24 DIP leading icon when Leading is nil.
	Icon IconDraw
	// Leading builds the leading slot (an avatar, an icon or a control) in
	// place of Icon.
	Leading func() ui.Element
	// Trailing builds the trailing slot (metadata, an icon or a control).
	Trailing func() ui.Element
	// Meta is trailing text shown before any Trailing slot.
	Meta string
	// OnClick runs when the row is clicked or activated by Enter or Space. A
	// row with OnClick, or a list with Selected, is interactive.
	OnClick func()
	// Disabled disables this row alone.
	Disabled bool
}

// ListOptions configures a List. The zero value is a plain, enabled list of
// rows without selection.
type ListOptions struct {
	// Label names the list for assistive technology.
	Label string
	// Segmented draws the rows as attached card pieces: the first rounds its
	// top corners, the last its bottom corners, inner rows keep small corners
	// and a single row rounds all four. A chosen or pressed row rounds all four
	// with the Default Spatial spring.
	Segmented bool
	// Selected points to the chosen row, -1 for none. The list writes clicks
	// and Enter or Space into it as it builds, and their state reaches
	// assistive technology.
	Selected *int
	// OnSelect runs after a click chooses a row.
	OnSelect func(i int)
	// Disabled disables the whole list.
	Disabled bool
}

// list metrics in DIP.
const (
	listOneLineHeight   = 56
	listTwoLineHeight   = 72
	listThreeLineHeight = 88
	listSegmentGap      = 2
	listRowPadding      = 16
	listSlotGap         = 16
	listIconSize        = 24
)

type listRowKey struct{ i int }

type listTopRadiusKey struct{}

type listBottomRadiusKey struct{}

type listSelectKey struct{}

// listTrailingKey gives the trailing slot a stable container identity, so a
// Meta text appearing or disappearing never re-keys the control inside it.
type listTrailingKey struct{}

// List builds a Material Design 3 Expressive list: a ui.RoleList of
// ui.RoleListItem rows, each with an optional overline, headline, supporting
// text, leading slot (icon, avatar or control), trailing meta and control.
// Rows with OnClick, or every row of a list with Selected, are interactive
// ui.ButtonBase rows with a StateLayer and Expressive Ripple; the arrows move
// the focus among them and Enter or Space chooses.
//
// Segmented lists attach their rows as card pieces with md3e's expressive
// corner rhythm, animating a chosen or pressed row to full large corners while
// its container settles on SecondaryContainer. Right to left mirrors the row
// layout through ui's logical edges:
//
//	md3e.List(c, []md3e.ListItem{
//		{Headline: "Wi-Fi", SupportingText: "Home network", Icon: mdicon.Wifi.Draw},
//		{Headline: "Bluetooth", SupportingText: "2 devices"},
//	}, md3e.ListOptions{Label: "Settings", Segmented: true, Selected: &app.setting})
func List(c *ui.Context, items []ListItem, opts ...ListOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: List takes at most one ListOptions")
	}
	var o ListOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	selectable := o.Selected != nil
	interactive := selectable
	for i := range items {
		if items[i].OnClick != nil {
			interactive = true
		}
	}

	gap := float32(0)
	if o.Segmented {
		gap = listSegmentGap
	}
	root := ui.Column(c).Role(ui.RoleList).Gap(gap).DimDisabled(false)
	if o.Label != "" {
		root.Label(o.Label)
	}
	root.Disabled(o.Disabled)
	if interactive {
		root.FocusGroup(ui.Vertical)
	}

	n := len(items)
	root.Children(func() {
		for i := range items {
			listRow(c, th, &items[i], i, n, o)
		}
	})
	return root
}

// listRow builds one row of a List.
func listRow(c *ui.Context, th *Theme, item *ListItem, i, n int, o ListOptions) {
	selectable := o.Selected != nil
	interactive := item.OnClick != nil || selectable
	if item.Key != nil {
		c.Key(item.Key)
	} else {
		c.Key(listRowKey{i: i})
	}

	var row ui.Element
	if interactive {
		row = ui.ButtonBase(c).DimDisabled(false).FocusRing(false).Role(ui.RoleListItem)
	} else {
		row = ui.Row(c).DimDisabled(false).Role(ui.RoleListItem)
	}
	disabled := o.Disabled || item.Disabled || row.IsDisabled()
	selected := selectable && *o.Selected == i

	lines := 1
	if item.Overline != "" {
		lines++
	}
	if item.SupportingText != "" {
		lines++
	}
	height := float32(listOneLineHeight)
	switch lines {
	case 2:
		height = listTwoLineHeight
	case 3:
		height = listThreeLineHeight
	}
	row.Height(height).FillWidth().PaddingX(listRowPadding).Gap(listSlotGap).AlignItems(ui.Center).
		Disabled(item.Disabled)
	if item.Headline != "" {
		row.Label(item.Headline)
	}
	if interactive {
		row.AccessSelected(selected)
	}

	var radii [4]float32
	if o.Segmented || interactive {
		large := th.Shapes.Radius(tokens.ShapeLarge)
		small := th.Shapes.Radius(tokens.ShapeSmall)
		topTarget := float32(0)
		bottomTarget := float32(0)
		if o.Segmented {
			target := segmentedRowRadii(i, n, large, small)
			topTarget, bottomTarget = target[0], target[2]
		}
		pressed := interactive && !disabled && row.Pressed()
		if selected || pressed {
			topTarget, bottomTarget = large, large
		}
		topR := animateWithTheme(row, th, listTopRadiusKey{}, topTarget, motion.DefaultSpatial)
		bottomR := animateWithTheme(row, th, listBottomRadiusKey{}, bottomTarget, motion.DefaultSpatial)
		radii = [4]float32{topR, topR, bottomR, bottomR}
		row.Radius(topR, topR, bottomR, bottomR)
	}

	cs := th.Colors
	base := cs.Surface
	if o.Segmented {
		base = SurfaceColorForElevation(th, tokens.Level1)
	}
	selectedT := float32(0)
	if selectable {
		if selected {
			selectedT = 1
		}
		selectedT = clampF32(animateWithTheme(row, th, listSelectKey{}, selectedT, motion.DefaultEffects), 0, 1)
	}
	selectedBg := cs.SecondaryContainer
	if selectedBg.A == 0 {
		selectedBg = cs.PrimaryContainer
	}
	bg := base
	if selectedBg.A > 0 && selectedT > 0.001 {
		bg = base.Mix(selectedBg, selectedT)
	}
	content := cs.OnSurface
	supporting := cs.OnSurfaceVariant
	if selected {
		content = cs.OnSecondaryContainer
		supporting = cs.OnSecondaryContainer
	}
	if disabled {
		bg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		content = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		supporting = content
	}
	row.Background(bg).TextColor(content)
	if interactive {
		attachRippleWithTheme(th, row, StateLayerOptions{
			Color:    content,
			Radii:    radii,
			HasRadii: true,
			Disabled: disabled,
		})
	}

	clicked := interactive && row.Clicked()
	row.Children(func() {
		if item.Leading != nil {
			if el := item.Leading(); el.Valid() {
				el.Shrink(0)
			}
		} else if item.Icon != nil {
			drawIconBox(c, item.Icon, listIconSize, content)
		}
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Clip().AlignItems(ui.Start).Gap(2).Children(func() {
			if item.Overline != "" {
				over := ui.Text(c, item.Overline).SingleLine()
				applyTypeStyleWithTheme(th, over, tokens.LabelSmall)
				over.TextColor(supporting)
			}
			if item.Headline != "" {
				h := ui.Text(c, item.Headline).SingleLine()
				applyTypeStyleWithTheme(th, h, tokens.BodyLarge)
				h.TextColor(content)
			}
			if item.SupportingText != "" {
				s := ui.Text(c, item.SupportingText).SingleLine()
				applyTypeStyleWithTheme(th, s, tokens.BodyMedium)
				s.TextColor(supporting)
			}
		})
		if item.Meta != "" {
			m := ui.Text(c, item.Meta).NoWrap().Shrink(0)
			applyTypeStyleWithTheme(th, m, tokens.LabelMedium)
			m.TextColor(supporting)
		}
		if item.Trailing != nil {
			slot := ui.Row(c).Key(listTrailingKey{}).Shrink(0).AlignItems(ui.Center)
			slot.Children(func() {
				if el := item.Trailing(); el.Valid() {
					el.Shrink(0)
				}
			})
		}
	})
	if clicked && !disabled {
		if item.OnClick != nil {
			item.OnClick()
		}
		if selectable {
			*o.Selected = i
			if o.OnSelect != nil {
				o.OnSelect(i)
			}
		}
	}
}

// segmentedRowRadii returns the four corners of row i of n in a segmented
// list: large at the group's top for the first row, large at the bottom for the
// last, small between, and large all round for a single row.
func segmentedRowRadii(i, n int, large, small float32) [4]float32 {
	smallAll := [4]float32{small, small, small, small}
	if n <= 1 {
		return [4]float32{large, large, large, large}
	}
	first, last := i == 0, i == n-1
	switch {
	case first && last:
		return [4]float32{large, large, large, large}
	case first:
		return [4]float32{large, large, small, small}
	case last:
		return [4]float32{small, small, large, large}
	default:
		return smallAll
	}
}
