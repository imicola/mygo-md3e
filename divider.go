package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/tokens"
)

// DividerOptions configures a Divider. The zero value is a 1 DIP horizontal
// line in OutlineVariant with no inset.
type DividerOptions struct {
	// Vertical draws a line down its container instead of across it, as a
	// separator between columns.
	Vertical bool
	// Thickness is the line's width in DIP; 1 when zero or negative.
	Thickness float32
	// Color overrides the line color; OutlineVariant when zero.
	Color ui.Color
	// Inset insets both logical ends of a horizontal line, or both vertical
	// ends of a vertical one, in DIP.
	Inset float32
	// StartInset insets the line's logical start (the left in a left-to-right
	// layout, the right right to left), in DIP.
	StartInset float32
	// EndInset insets the line's logical end, in DIP.
	EndInset float32
	// Label names the separator for assistive technology.
	Label string
}

// Divider builds a Material Design 3 divider: a 1 DIP OutlineVariant line with
// ui.RoleSeparator, across its parent or down it, inset from the logical start
// and end. The horizontal insets use ui's logical padding, so a line inset from
// the left in a left-to-right layout is inset from the right right to left:
//
//	md3e.Divider(c)
//	md3e.Divider(c, md3e.DividerOptions{Inset: 16})
func Divider(c *ui.Context, opts ...DividerOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Divider takes at most one DividerOptions")
	}
	var o DividerOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	thickness := o.Thickness
	if !(thickness > 0) {
		thickness = 1
	}
	col := o.Color
	if col.A == 0 {
		col = th.Colors.OutlineVariant
		if th.Contrast >= tokens.ContrastHigh && th.Colors.Outline.A > 0 {
			col = th.Colors.Outline
		}
	}
	if col.A == 0 {
		col = th.Colors.Outline
	}
	if col.A == 0 {
		scrim := th.Colors.Scrim
		if scrim.A == 0 {
			scrim = ui.RGB(0, 0, 0)
		}
		col = scrim.Alpha(0.12)
	}

	if o.Vertical {
		if o.Inset > 0 {
			// Padding the box the line fills insets both ends; a margin on a
			// stretched line would not shorten it.
			el := ui.Box(c).Width(thickness).Shrink(0).FillHeight().PaddingY(o.Inset)
			el.Children(func() {
				inner := ui.Box(c).Width(thickness).FillHeight().Background(col).Role(ui.RoleSeparator)
				if o.Label != "" {
					inner.Label(o.Label)
				}
			})
			return el
		}
		el := ui.Box(c).Width(thickness).Shrink(0).FillHeight().Background(col).Role(ui.RoleSeparator)
		if o.Label != "" {
			el.Label(o.Label)
		}
		return el
	}

	// A line's logical insets pad the box it fills, which a bare fill would
	// ignore; the box itself carries the separator's role and paint.
	start := o.StartInset + o.Inset
	end := o.EndInset + o.Inset
	if start != 0 || end != 0 {
		el := ui.Box(c).Height(thickness).FillWidth().PaddingStart(start).PaddingEnd(end)
		el.Children(func() {
			inner := ui.Box(c).Height(thickness).FillWidth().Background(col).Role(ui.RoleSeparator)
			if o.Label != "" {
				inner.Label(o.Label)
			}
		})
		return el
	}
	el := ui.Box(c).Height(thickness).FillWidth().Background(col).Role(ui.RoleSeparator)
	if o.Label != "" {
		el.Label(o.Label)
	}
	return el
}
