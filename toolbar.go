package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// ToolbarOrientation selects the axis of a FloatingToolbar.
type ToolbarOrientation int

const (
	// ToolbarHorizontal lays the controls out in a 64 DIP high pill.
	ToolbarHorizontal ToolbarOrientation = iota
	// ToolbarVertical lays them out in a 64 DIP wide pill.
	ToolbarVertical
)

// FloatingToolbarTone selects the color family of a FloatingToolbar.
type FloatingToolbarTone int

const (
	// ToolbarStandard paints the raised Surface Container with OnSurface
	// content.
	ToolbarStandard FloatingToolbarTone = iota
	// ToolbarVibrant paints PrimaryContainer with OnPrimaryContainer content.
	ToolbarVibrant
)

// FloatingToolbarItem is one action of a FloatingToolbar.
type FloatingToolbarItem struct {
	// Label is the accessible name of the item.
	Label string
	// Icon draws the item's icon.
	Icon IconDraw
	// OnClick runs when the item is pressed.
	OnClick func()
	// Disabled disables this item alone.
	Disabled bool
}

// FloatingToolbarFab couples a standard FAB to a FloatingToolbar.
type FloatingToolbarFab struct {
	// Label is the accessible name of the FAB.
	Label string
	// Icon draws the FAB's icon; mdicon.Add when nil.
	Icon IconDraw
	// OnClick runs when the FAB is pressed.
	OnClick func()
	// Color selects the FAB's container/content pairing.
	Color FABColorScheme
}

// FloatingToolbarOptions configures FloatingToolbar. The zero value is a
// horizontal, standard tone, expanded and visible toolbar with no FAB.
type FloatingToolbarOptions struct {
	// Orientation selects the horizontal or vertical layout.
	Orientation ToolbarOrientation
	// Tone selects the standard or vibrant colors.
	Tone FloatingToolbarTone
	// Label names the toolbar for assistive technology; "Toolbar" when empty.
	Label string
	// Expanded, when set, collapses the toolbar to its first item while false,
	// springing its length down with the Default Spatial spring.
	Expanded *bool
	// Visible, when set, hides the whole toolbar and its FAB while false.
	Visible *bool
	// Fab, when set, couples a standard FAB beside the toolbar.
	Fab *FloatingToolbarFab
}

// Floating toolbar metrics, in DIP.
const (
	floatingToolbarSide    = 64
	floatingToolbarItem    = 48
	floatingToolbarPadding = 8
	floatingToolbarGap     = 4
	floatingToolbarFabGap  = 12
)

// FloatingToolbar builds a Material Design 3 Expressive floating toolbar: an
// elevated pill that gathers related actions, horizontal (64 DIP high) or
// vertical (64 DIP wide), in the standard raised surface or the vibrant
// PrimaryContainer tone. It holds the items as one RoleToolbar focus group that
// the arrow keys move through, carries Level3 Key/Ambient shadows, and can sit
// beside a standard FAB, sharing its container alignment and a 12 DIP gap; the
// row mirrors right to left.
//
// Expanded folds the toolbar down to its first item with the Default Spatial
// spring, and Visible hides the whole group with a scale and fade:
//
//	md3e.FloatingToolbar(c, []md3e.FloatingToolbarItem{
//		{Label: "Bold", Icon: editIcon, OnClick: app.bold},
//		{Label: "Add", Icon: addIcon, OnClick: app.add},
//	}, md3e.FloatingToolbarOptions{Label: "Format", Fab: &app.fab})
func FloatingToolbar(c *ui.Context, items []FloatingToolbarItem, opts ...FloatingToolbarOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: FloatingToolbar takes at most one FloatingToolbarOptions")
	}
	var o FloatingToolbarOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	var group ui.Element
	if o.Orientation == ToolbarVertical {
		group = ui.Column(c).Gap(floatingToolbarFabGap).AlignItems(ui.Center).Shrink(0)
	} else {
		group = ui.Row(c).Gap(floatingToolbarFabGap).AlignItems(ui.Center).Shrink(0)
	}
	group.Children(func() {
		bar := floatingToolbarBar(c, items, o)
		if o.Fab != nil {
			icon := o.Fab.Icon
			if icon == nil {
				icon = mdicon.Add.Draw
			}
			fab := FAB(c, o.Fab.Label, icon, FABOptions{Color: o.Fab.Color, Visible: o.Visible})
			if fab.Clicked() && o.Fab.OnClick != nil {
				o.Fab.OnClick()
			}
		}
		_ = bar
	})
	return group
}

// floatingToolbarBar builds the toolbar pill itself, inside the group that
// positions it beside the FAB.
func floatingToolbarBar(c *ui.Context, items []FloatingToolbarItem, o FloatingToolbarOptions) ui.Element {
	th := CurrentTheme(c)
	label := labelOr(o.Label, "Toolbar")
	expanded := o.Expanded == nil || *o.Expanded
	visible := o.Visible == nil || *o.Visible

	container := th.Colors.PrimaryContainer
	ink := th.Colors.OnPrimaryContainer
	if o.Tone == ToolbarStandard {
		container = th.Colors.SurfaceContainer
		if container.A == 0 {
			container = SurfaceColorForElevation(th, tokens.Level2)
		}
		ink = th.Colors.OnSurface
	}

	var bar ui.Element
	if o.Orientation == ToolbarVertical {
		bar = ui.Column(c).
			Width(floatingToolbarSide).
			Padding(floatingToolbarPadding).
			Gap(floatingToolbarGap).
			AlignItems(ui.Center)
	} else {
		bar = ui.Row(c).
			Height(floatingToolbarSide).
			PaddingX(floatingToolbarPadding).
			Gap(floatingToolbarGap).
			AlignItems(ui.Center)
	}
	bar.Clip().
		Background(container).
		TextColor(ink).
		Radius(floatingToolbarSide / 2).
		DimDisabled(false).
		Role(ui.RoleToolbar).
		Orientation(toolbarOrientation(o.Orientation)).
		FocusGroup(toolbarOrientation(o.Orientation)).
		Label(label)
	if o.Tone == ToolbarStandard && th.Dark {
		bar.Border(1, resolveHairlineColor(th, ui.Color{}))
	}
	ApplyElevationShadows(th, bar, tokens.Level3)

	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	shown := len(items)
	if !expanded {
		shown = min(shown, 1)
	}
	length := float32(floatingToolbarPadding*2 + shown*floatingToolbarItem + max(shown-1, 0)*floatingToolbarGap)
	animated := animateWithTheme(bar, th, animChan0, length, motion.DefaultSpatial)
	pillLen := max(animated, float32(floatingToolbarSide))
	if o.Orientation == ToolbarVertical {
		bar.Height(pillLen)
	} else {
		bar.Width(pillLen)
	}

	if o.Visible != nil {
		scaleTarget, fadeTarget := float32(1), float32(1)
		if !visible {
			scaleTarget, fadeTarget = 0.6, 0
		}
		scale := animateWithTheme(bar, th, animChan1, scaleTarget, motion.FastSpatial)
		fade := animateWithTheme(bar, th, animChan2, fadeTarget, motion.FastEffects)
		bar.Scale(scale, scale).Opacity(clampF32(fade, 0, 1))
		if !visible {
			bar.Disabled(true)
		}
	}

	bar.Children(func() {
		for i := range items {
			// Always keep item 0 so collapse spring undershoot never blinks out
			// the anchor action; only drop trailing items past the animated pill.
			slotEnd := float32(floatingToolbarPadding*2 + (i+1)*floatingToolbarItem + i*floatingToolbarGap)
			if i > 0 && slotEnd > animated+0.5 {
				continue
			}
			item := &items[i]
			btn := ui.ButtonBase(c).
				DimDisabled(false).
				FocusRing(false).
				Size(floatingToolbarItem, floatingToolbarItem).
				Shrink(0).
				Label(item.Label)
			disabled := item.Disabled || (visible && btn.IsDisabled())
			itemInk := ink
			if disabled {
				itemInk = ink.Alpha(float32(tokens.DisabledContentOpacity))
			}
			radTarget := float32(floatingToolbarItem / 2)
			if !disabled && btn.Pressed() {
				radTarget = 12
			}
			rad := max(animateWithTheme(btn, th, animChan0, radTarget, motion.FastSpatial), 0)
			btn.Disabled(item.Disabled).TextColor(itemInk).Radius(rad)
			if i > 0 && !reduce {
				prevEnd := float32(floatingToolbarPadding*2 + i*floatingToolbarItem + (i-1)*floatingToolbarGap)
				reveal := clampF32((animated-prevEnd)/float32(floatingToolbarItem), 0, 1)
				if reveal < 0.999 {
					s := 0.85 + 0.15*reveal
					btn.Opacity(reveal).Scale(s, s)
				}
			}
			attachRippleWithContext(c, th, btn, StateLayerOptions{
				Color:    itemInk,
				Radius:   rad,
				Disabled: disabled,
			})
			clicked := btn.Clicked()
			btn.Children(func() {
				if item.Icon != nil {
					drawIconBox(c, item.Icon, 24, itemInk)
				}
			})
			if clicked && item.OnClick != nil {
				item.OnClick()
			}
		}
	})
	return bar
}

// toolbarOrientation maps the toolbar axis to a focus group and accessibility
// orientation.
func toolbarOrientation(o ToolbarOrientation) ui.Orientation {
	if o == ToolbarVertical {
		return ui.Vertical
	}
	return ui.Horizontal
}
