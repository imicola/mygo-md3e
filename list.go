package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
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

// listTrailingKey gives the trailing slot a stable container identity, so a
// Meta text appearing or disappearing never re-keys the control inside it.
type listTrailingKey struct{}

// List builds a Material Design 3 Expressive list: a ui.RoleList of
// ui.RoleListItem rows, each with an optional overline, headline, supporting
// text, leading slot (icon, avatar or control), trailing meta and control.
// Rows with OnClick, or every row of a list with Selected, are interactive
// ui.ButtonBase rows with a StateLayer and Expressive Ripple; the arrows move
// the focus among them and Enter or Space chooses. When an interactive row is
// chosen, *Selected is updated and OnSelect fires before item.OnClick so click
// handlers that delete or reorder rows never have *Selected overwritten.
//
// List materializes all rows in a ui.Column so it measures naturally inside any
// layout or outer ui.Scroll container; for virtualized feeds of thousands of
// rows, compose ui.List directly under ThemeScope.
//
// Segmented lists attach their rows as card pieces with md3e's expressive
// corner rhythm, animating a chosen or pressed row to full large corners while
// its container settles on SecondaryContainer. Right to left mirrors the row
// layout through ui's logical edges:
//
//	md3e.List(c, []md3e.ListItem{
//		{Headline: "Wi-Fi", SupportingText: "Home network", Icon: wifiIcon},
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

	style := newListStyle(th, o.Segmented)
	n := len(items)
	root.Children(func() {
		for i := range items {
			listRow(c, th, style, &items[i], i, n, o)
		}
	})
	return root
}

type listStyle struct {
	large           float32
	small           float32
	oneLineH        float32
	twoLineH        float32
	threeLineH      float32
	padX            float32
	slotGap         float32
	textGap         float32
	iconSize        float32
	headlineRole    tokens.TypeRole
	supportingRole  tokens.TypeRole
	base            ui.Color
	selectedBg      ui.Color
	content         ui.Color
	supporting      ui.Color
	selectedContent ui.Color
	disabledBg      ui.Color
	disabledContent ui.Color
}

func newListStyle(th *Theme, segmented bool) listStyle {
	cs := th.Colors
	base := cs.Surface
	if segmented {
		base = SurfaceColorForElevation(th, tokens.Level1)
	}
	selectedBg := cs.SecondaryContainer
	if selectedBg.A == 0 {
		selectedBg = cs.PrimaryContainer
	}
	oneH, twoH, threeH := th.Density.ListHeights()
	padX := float32(listRowPadding)
	slotGap := float32(listSlotGap)
	textGap := float32(2)
	iconSize := float32(listIconSize)
	headlineRole := tokens.BodyLarge
	supportingRole := tokens.BodyMedium
	switch th.Density.Normalize() {
	case tokens.DensityDesktop:
		padX = 12
		slotGap = 12
		textGap = 1
		iconSize = 20
		headlineRole = tokens.BodyMedium
		supportingRole = tokens.BodySmall
	case tokens.DensityCompact:
		padX = 10
		slotGap = 10
		textGap = 0
		iconSize = 18
		headlineRole = tokens.BodyMedium
		supportingRole = tokens.BodySmall
	}
	return listStyle{
		large:           th.Shapes.Radius(tokens.ShapeLarge),
		small:           th.Shapes.Radius(tokens.ShapeSmall),
		oneLineH:        float32(oneH),
		twoLineH:        float32(twoH),
		threeLineH:      float32(threeH),
		padX:            padX,
		slotGap:         slotGap,
		textGap:         textGap,
		iconSize:        iconSize,
		headlineRole:    headlineRole,
		supportingRole:  supportingRole,
		base:            base,
		selectedBg:      selectedBg,
		content:         cs.OnSurface,
		supporting:      cs.OnSurfaceVariant,
		selectedContent: cs.OnSecondaryContainer,
		disabledBg:      cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
		disabledContent: cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity)),
	}
}

// listRow builds one row of a List.
func listRow(c *ui.Context, th *Theme, st listStyle, item *ListItem, i, n int, o ListOptions) {
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
	height := st.oneLineH
	switch lines {
	case 2:
		height = st.twoLineH
	case 3:
		height = st.threeLineH
	}
	row.Height(height).FillWidth().PaddingX(st.padX).Gap(st.slotGap).AlignItems(ui.Center).
		Disabled(item.Disabled)
	if item.Headline != "" {
		row.Label(item.Headline)
	}
	if interactive {
		row.AccessSelected(selected)
	}

	var radii [4]float32
	if o.Segmented || interactive {
		topTarget := float32(0)
		bottomTarget := float32(0)
		if o.Segmented {
			target := segmentedRowRadii(i, n, st.large, st.small)
			topTarget, bottomTarget = target[0], target[2]
		}
		pressed := interactive && !disabled && row.Pressed()
		cornerSpec := motion.DefaultSpatial
		switch {
		case selected && pressed:
			// Tactile micro-tighten when re-pressing an already-selected row.
			r := max(st.large-4, st.small)
			topTarget, bottomTarget = r, r
			cornerSpec = motion.FastSpatial
		case pressed:
			topTarget, bottomTarget = st.large, st.large
			cornerSpec = motion.FastSpatial
		case selected:
			topTarget, bottomTarget = st.large, st.large
		}
		topR := max(animateWithTheme(row, th, animChan0, topTarget, cornerSpec), 0)
		bottomR := max(animateWithTheme(row, th, animChan1, bottomTarget, cornerSpec), 0)
		radii = [4]float32{topR, topR, bottomR, bottomR}
		row.Radius(topR, topR, bottomR, bottomR)
	}

	selectedT := float32(0)
	if selectable {
		if selected {
			selectedT = 1
		}
		selectedT = clampF32(animateWithTheme(row, th, animChan2, selectedT, motion.DefaultEffects), 0, 1)
	}
	bg := st.base
	if st.selectedBg.A > 0 && selectedT > 0.001 {
		bg = st.base.MixOklab(st.selectedBg, selectedT)
	}
	content := st.content
	supporting := st.supporting
	if selectedT > 0.001 {
		content = st.content.MixOklab(st.selectedContent, selectedT)
		supporting = st.supporting.MixOklab(st.selectedContent.Alpha(0.78), selectedT)
	}
	if disabled {
		bg = st.disabledBg
		content = st.disabledContent
		supporting = st.disabledContent
	}
	row.Background(bg).TextColor(content)
	borderInset := float32(0)
	if o.Segmented && !disabled && !selected && th.Dark {
		if hl := resolveHairlineColor(th, ui.Color{}); hl.A > 0 {
			row.Border(1, hl)
			borderInset = 1
		}
	}
	if interactive {
		attachRippleWithContext(c, th, row, StateLayerOptions{
			Color:       content,
			Radii:       radii,
			HasRadii:    true,
			BorderInset: borderInset,
			Disabled:    disabled,
		})
	}

	clicked := interactive && row.Clicked()
	row.Children(func() {
		if item.Leading != nil {
			if el := item.Leading(); el.Valid() {
				el.Shrink(0)
			}
		} else if item.Icon != nil {
			drawIconBox(c, item.Icon, st.iconSize, content)
		}
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Clip().AlignItems(ui.Start).Gap(st.textGap).Children(func() {
			if item.Overline != "" {
				over := ui.Text(c, item.Overline).SingleLine()
				applyTypeStyleWithTheme(th, over, tokens.LabelSmall)
				over.TextColor(supporting)
			}
			if item.Headline != "" {
				h := ui.Text(c, item.Headline).SingleLine()
				applyTypeStyleWithTheme(th, h, st.headlineRole, selected && !disabled)
				h.TextColor(content)
			}
			if item.SupportingText != "" {
				s := ui.Text(c, item.SupportingText).SingleLine()
				applyTypeStyleWithTheme(th, s, st.supportingRole)
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
		if selectable {
			*o.Selected = i
			if o.OnSelect != nil {
				o.OnSelect(i)
			}
		}
		if item.OnClick != nil {
			item.OnClick()
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
