package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// MenuItem is one row of a DropdownMenu, MenuButton or ContextMenu.
type MenuItem struct {
	// Label is the visible text and the accessible name.
	Label string
	// Icon draws an optional leading 20 DIP icon.
	Icon IconDraw
	// Shortcut is shown at the row's end, as "⌘C"; it is a hint only.
	Shortcut string
	// Separator draws a divider instead of a row.
	Separator bool
	// Checkable makes the row a check box item showing Checked.
	Checkable bool
	// Radio makes the row a radio item showing Checked.
	Radio bool
	// Checked marks a check box or radio item chosen.
	Checked bool
	// Disabled grays the row out and keeps the pointer and the keys off it.
	Disabled bool
	// Submenu turns the row into one opening another menu of these items.
	Submenu []MenuItem
	// OnClick runs when the row is chosen, before the menu closes.
	OnClick func()
}

// MenuOptions configures DropdownMenu, MenuButton and ContextMenu.
type MenuOptions struct {
	// Open lets the caller own the menu's open state; nil keeps it in the
	// control.
	Open *bool
	// Label names the menu for assistive technology.
	Label string
	// Disabled disables the trigger and keeps the menu closed.
	Disabled bool
	// Width fixes the menu panel's width in DIP; its content's when zero.
	Width float32
}

// Menu metrics, in DIP.
const (
	menuRowHeight     = 40
	menuMarkSize      = 18
	menuIconSize      = 20
	menuPanelPadding  = 8
	menuPanelMinWidth = 180
	menuTriggerHeight = 40
	menuTriggerRadius = 20
)

// Locals and keys of the menu controls.
type (
	menuOpenKey        struct{}
	contextMenuOpenKey struct{}
)

// DropdownMenu builds a Material Design 3 Expressive drop-down menu: a text
// trigger with a chevron that opens a ui.MenuBase panel below it, styled with
// large corners, a SurfaceContainerLow fill, Level2 shadows and
// SecondaryContainer highlighted rows. Items may carry icons, shortcuts,
// check and radio marks, dividers and submenus; Up, Down, Home, End and typing
// move the highlight, Right and Left open and close submenus (mirrored right to
// left), Enter and Space choose, and Escape, Tab or a press outside close the
// menu. A disabled trigger, or one under a disabled ancestor, keeps it closed.
//
//	md3e.DropdownMenu(c, "Sort by", []md3e.MenuItem{
//		{Label: "Name", Checkable: true, Checked: app.byName, OnClick: app.sortByName},
//		{Label: "Date", Checkable: true, Checked: !app.byName, OnClick: app.sortByDate},
//		{Separator: true},
//		{Label: "More", Submenu: []md3e.MenuItem{{Label: "Advanced", OnClick: app.advanced}}},
//	}, md3e.MenuOptions{Label: "Sort"})
func DropdownMenu(c *ui.Context, label string, items []MenuItem, opts ...MenuOptions) ui.Element {
	var o MenuOptions
	if len(opts) > 1 {
		panic("md3e: DropdownMenu takes at most one MenuOptions")
	}
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	buildTrigger := func(open *bool, grow bool) ui.Element {
		m := ui.MenuBase(c, open)
		trigger := m.Trigger.Height(menuTriggerHeight).PaddingX(16).Gap(8).
			Radius(menuTriggerRadius).Label(label).Disabled(o.Disabled).
			TextColor(th.Colors.OnSurface)
		if grow {
			trigger.Grow(1)
		}
		attachRippleWithTheme(th, trigger, StateLayerOptions{
			Color:    th.Colors.OnSurface,
			Radius:   menuTriggerRadius,
			Disabled: o.Disabled || trigger.IsDisabled(),
		})
		trigger.Children(func() {
			txt := ui.Text(c, label).NoWrap()
			applyTypeStyleWithTheme(th, txt, tokens.LabelLarge)
			drawIconBox(c, mdicon.ChevronDown.Draw, 18, th.Colors.OnSurfaceVariant)
		})
		m.Popup(func(panel ui.Element) {
			styleMenuPanel(th, panel, o)
			buildMenuItems(c, th, m, items, 0)
		})
		return trigger
	}
	if o.Open != nil {
		return buildTrigger(o.Open, false)
	}
	wrap := ui.Row(c).Shrink(0)
	open := ui.Local(wrap, menuOpenKey{}, func() bool { return false })
	wrap.Children(func() {
		trigger := buildTrigger(open, true)
		if wrap.Focused() {
			trigger.Focus()
		}
	})
	return wrap
}

// MenuButton builds a Material Design 3 Expressive icon menu button: a round
// tonal trigger holding icon, named by label, which opens the same menu as
// DropdownMenu. It takes a width-sized icon button's look and the same keys.
func MenuButton(c *ui.Context, label string, icon IconDraw, items []MenuItem, opts ...MenuOptions) ui.Element {
	var o MenuOptions
	if len(opts) > 1 {
		panic("md3e: MenuButton takes at most one MenuOptions")
	}
	if len(opts) == 1 {
		o = opts[0]
	}
	if label == "" {
		panic("md3e: MenuButton requires an accessibility label")
	}
	th := CurrentTheme(c)
	buildTrigger := func(open *bool) ui.Element {
		m := ui.MenuBase(c, open)
		trigger := m.Trigger.Size(menuTriggerHeight, menuTriggerHeight).
			Radius(menuTriggerHeight / 2).
			Background(th.Colors.SurfaceContainerHighest).
			TextColor(th.Colors.OnSurfaceVariant).
			Label(label).Disabled(o.Disabled)
		attachRippleWithTheme(th, trigger, StateLayerOptions{
			Color:    th.Colors.OnSurfaceVariant,
			Radius:   menuTriggerHeight / 2,
			Disabled: o.Disabled || trigger.IsDisabled(),
		})
		trigger.Children(func() {
			drawIconBox(c, icon, 24, th.Colors.OnSurfaceVariant)
		})
		m.Popup(func(panel ui.Element) {
			styleMenuPanel(th, panel, o)
			buildMenuItems(c, th, m, items, 0)
		})
		return trigger
	}
	if o.Open != nil {
		return buildTrigger(o.Open)
	}
	wrap := ui.Row(c).Shrink(0)
	open := ui.Local(wrap, menuOpenKey{}, func() bool { return false })
	wrap.Children(func() {
		trigger := buildTrigger(open)
		if wrap.Focused() {
			trigger.Focus()
		}
	})
	return wrap
}

// ContextMenu gives target a Material Design 3 Expressive context menu, opened
// where target is right-clicked or below it for the menu key or Shift+F10, with
// the same panel and items as DropdownMenu. It returns target:
//
//	row := md3e.ContextMenu(c, row, []md3e.MenuItem{
//		{Label: "Rename", Shortcut: "F2", OnClick: app.rename},
//		{Label: "Delete", OnClick: app.delete, Disabled: note.locked},
//	}, md3e.MenuOptions{Label: "Row actions"})
func ContextMenu(c *ui.Context, target ui.Element, items []MenuItem, opts ...MenuOptions) ui.Element {
	var o MenuOptions
	if len(opts) > 1 {
		panic("md3e: ContextMenu takes at most one MenuOptions")
	}
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	open := o.Open
	if open == nil {
		open = ui.Local(target, contextMenuOpenKey{}, func() bool { return false })
	}
	m := ui.ContextMenuBase(c, target, open)
	m.Popup(func(panel ui.Element) {
		styleMenuPanel(th, panel, o)
		buildMenuItems(c, th, m, items, 0)
	})
	return target
}

// styleMenuPanel paints the layered, large-cornered menu surface.
func styleMenuPanel(th *Theme, panel ui.Element, o MenuOptions) {
	if o.Width > 0 {
		panel.Width(o.Width)
	} else {
		panel.MinWidth(menuPanelMinWidth)
	}
	panel.Padding(menuPanelPadding).
		Radius(th.Shapes.Radius(tokens.ShapeLarge)).
		Background(SurfaceColorForElevation(th, tokens.Level2)).
		TextColor(th.Colors.OnSurface)
	ApplyElevationShadows(th, panel, tokens.Level2)
	if o.Label != "" {
		panel.Label(o.Label)
	}
}

// buildMenuItems adds items to the menu being built, recursing into submenus.
func buildMenuItems(c *ui.Context, th *Theme, m ui.MenuParts, items []MenuItem, level int) {
	for i := range items {
		item := &items[i]
		if item.Separator {
			Divider(c, DividerOptions{Inset: menuPanelPadding})
			continue
		}
		if len(item.Submenu) > 0 {
			row := m.SubmenuItem()
			styleMenuRow(c, th, row, item, true, false)
			m.Submenu(row, func(panel ui.Element) {
				styleMenuPanel(th, panel, MenuOptions{})
				buildMenuItems(c, th, m, item.Submenu, level+1)
			})
			continue
		}
		var row ui.Element
		if item.Radio {
			row = m.RadioItem(item.Checked)
		} else if item.Checkable {
			row = m.CheckItem(item.Checked)
		} else {
			row = m.Item()
		}
		styleMenuRow(c, th, row, item, false, item.Radio)
		if row.Clicked() && !item.Disabled && item.OnClick != nil {
			item.OnClick()
		}
	}
}

// styleMenuRow gives a menu row its MD3E padding, corner, highlight, marks,
// icon, label and shortcut. marked draws the check or radio indicator in the
// leading slot, and chevron its submenu arrow at the end.
func styleMenuRow(c *ui.Context, th *Theme, row ui.Element, item *MenuItem, chevron, radio bool) {
	cs := th.Colors
	disabled := item.Disabled || row.IsDisabled()
	highlighted := row.Highlighted()

	bg := ui.Transparent
	ink := cs.OnSurface
	subInk := cs.OnSurfaceVariant
	if highlighted {
		bg = cs.SecondaryContainer
		if bg.A == 0 {
			bg = cs.PrimaryContainer
		}
		ink = cs.OnSecondaryContainer
		subInk = ink.Alpha(0.8)
	}
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		subInk = ink
	}

	row.Height(menuRowHeight).
		PaddingY(8).PaddingX(12).
		Radius(th.Shapes.Radius(tokens.ShapeSmall)).
		Gap(12).
		AlignItems(ui.Center).
		Disabled(item.Disabled).
		Background(bg).
		TextColor(ink)
	marked := radio || item.Checkable
	row.Children(func() {
		if marked {
			drawMenuMark(c, item.Checked, radio, menuMarkSize, ink)
		}
		if item.Icon != nil {
			drawIconBox(c, item.Icon, menuIconSize, ink)
		}
		txt := ui.Text(c, item.Label).NoWrap().Grow(1)
		applyTypeStyleWithTheme(th, txt, tokens.LabelLarge)
		txt.TextColor(ink)
		if item.Shortcut != "" {
			short := ui.Text(c, item.Shortcut).NoWrap().Shrink(0)
			applyTypeStyleWithTheme(th, short, tokens.LabelSmall)
			short.TextColor(subInk)
		}
		if chevron {
			icon := mdicon.ChevronRight.Draw
			if c.Direction() == ui.RightToLeft {
				icon = mdicon.ChevronLeft.Draw
			}
			drawIconBox(c, icon, 18, subInk)
		}
	})
}

// drawMenuMark paints a checked or unchecked check box or radio indicator in
// the leading slot of an item, keeping the slot the same width either way.
func drawMenuMark(c *ui.Context, checked, radio bool, size float32, ink ui.Color) {
	ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		if radio {
			center := ui.Point{X: r.X + r.W/2, Y: r.Y + r.H/2}
			if checked {
				p.Fill(ui.Rect{X: center.X - 5, Y: center.Y - 5, W: 10, H: 10}, ink, 5)
				return
			}
			p.Stroke(ui.Rect{X: center.X - 7, Y: center.Y - 7, W: 14, H: 14}, ink, 7, 1.5)
			return
		}
		if checked {
			mdicon.Check.Draw(p, r, ink)
		}
	})
}
