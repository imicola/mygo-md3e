package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// FABMenuItem is one action of a FABMenu.
type FABMenuItem struct {
	// Label is the visible text and the accessible name of the item.
	Label string
	// Icon draws the item's leading icon.
	Icon IconDraw
	// OnClick runs when the item is chosen.
	OnClick func()
	// Disabled disables this item alone.
	Disabled bool
}

// FABMenuOptions configures FABMenu. The zero value is a primary container
// trigger that keeps its own open state.
type FABMenuOptions struct {
	// Open lets the caller own the menu's expanded state, as when the app
	// closes it from elsewhere; nil keeps the state in the control.
	Open *bool
	// Color selects the trigger FAB's container/content pairing.
	Color FABColorScheme
	// Label names the trigger for assistive technology; "Menu" when empty.
	Label string
	// Icon is the trigger's icon while the menu is closed; mdicon.Add when
	// nil.
	Icon IconDraw
	// CloseLabel names the in-place close button; "Close menu" when empty.
	CloseLabel string
	// MenuLabel names the menu panel for assistive technology; "Menu actions"
	// when empty.
	MenuLabel string
	// Disabled disables the trigger and keeps the menu closed.
	Disabled bool
}

// FAB menu bounds and rhythm, in DIP.
const (
	fabMenuCloseSide = 56
	fabMenuItemSide  = 48
	fabMenuGap       = 8
	fabMenuTriggerTo = 12
)

// fabMenuOpenKey is the Local holding a FABMenu's expansion state when the
// caller does not own it.
type fabMenuOpenKey struct{}

// fabMenuHandleKey is the Local holding the trigger's handle, so that closing
// can give the keyboard focus back to it.
type fabMenuHandleKey struct{}

// fabMenuPoseKey marks an overlay element that has played its entry pose.
type fabMenuPoseKey struct{ i int }

// FABMenu builds a Material Design 3 Expressive FAB menu: a standard 56 DIP
// FAB that opens a scrim-covered menu of two to six large pill items, each with
// an icon and a label, in place of the old speed dial. While the menu shows,
// the trigger is covered by a high-contrast circular close button whose ×
// rotates in with the Fast Spatial spring, and the items spring up in sequence.
// Pressing the scrim, Escape, the close button, or an item closes the menu and
// gives the keyboard focus back to the trigger.
//
// The item count is part of the spec: FABMenu panics for fewer than two or
// more than six items.
//
//	md3e.FABMenu(c, []md3e.FABMenuItem{
//		{Label: "New file", Icon: addIcon, OnClick: app.newFile},
//		{Label: "New folder", Icon: editIcon, OnClick: app.newFolder},
//	}, md3e.FABMenuOptions{Label: "Create"})
func FABMenu(c *ui.Context, items []FABMenuItem, opts ...FABMenuOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: FABMenu takes at most one FABMenuOptions")
	}
	if len(items) < 2 || len(items) > fabMenuItemMax {
		panic("md3e: FABMenu takes 2 to 6 items")
	}
	var o FABMenuOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	label := o.Label
	if label == "" {
		label = "Menu"
	}
	icon := o.Icon
	if icon == nil {
		icon = mdicon.Add.Draw
	}
	closeLabel := o.CloseLabel
	if closeLabel == "" {
		closeLabel = "Close menu"
	}
	menuLabel := o.MenuLabel
	if menuLabel == "" {
		menuLabel = "Menu actions"
	}

	trigger := FAB(c, label, icon, FABOptions{Color: o.Color, Disabled: o.Disabled})
	open := o.Open
	if open == nil {
		open = ui.Local(trigger, fabMenuOpenKey{}, func() bool { return false })
	}
	handle := ui.Local(trigger, fabMenuHandleKey{}, func() ui.Handle { return ui.Handle{} })
	disabled := o.Disabled || trigger.IsDisabled()
	if disabled {
		*open = false
	}
	trigger.Bind(handle).HasPopup(ui.PopupMenu).Expanded(*open)

	if trigger.Clicked() && !disabled {
		*open = !*open
	}
	if inExitingOverlay(c) {
		if *open {
			*open = false
		}
		return trigger
	}
	if !*open || disabled {
		return trigger
	}
	// The circular close button covers the trigger while the menu shows. Hide
	// the trigger's own rounded square, icon and shadows so they do not show
	// around the circle, even while the close button is still scaling in;
	// Opacity keeps the trigger's layout, bounds and handle, so closing can
	// give the keyboard focus back to it. FABOptions.Visible would disable it
	// instead and lose that focus.
	trigger.Opacity(0)

	scrim := th.Colors.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}

	ui.Overlay(c, func() {
		back := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(scrim.Alpha(0.32)).
			Modal()
		back.Children(func() {
			// A plain anchor inside the modal keeps the panel and the close
			// button in the dialog's focus scope: an AttachTo(trigger) would
			// scope them to the trigger outside the modal, where Tab cannot
			// reach the items at all. It mirrors the trigger's last box.
			tb := trigger.Bounds()
			var anchor ui.Element
			if tb.W > 0 && tb.H > 0 {
				anchor = ui.Box(c).Absolute().Left(tb.X).Top(tb.Y).Size(tb.W, tb.H)
			} else {
				// A menu open on the first frame has no previous trigger box
				// to mirror yet.
				anchor = ui.Box(c).Size(1, 1).AttachTo(trigger, ui.AnchorCenter, ui.AnchorCenter)
			}
			closeBtn := fabMenuCloseButton(c, th, anchor, closeLabel)
			panel := ui.Column(c).
				AttachTo(anchor, ui.AnchorTopEnd, ui.AnchorBottomEnd).
				Margin(0, 0, fabMenuTriggerTo, 0).
				Gap(fabMenuGap).
				AlignItems(ui.End).
				Role(ui.RoleMenu).
				FocusGroup(ui.Vertical).
				Label(menuLabel)
			panel.Children(func() {
				// Build from the far end down, so that the first item sits
				// nearest the trigger and rises first.
				for i := len(items) - 1; i >= 0; i-- {
					if fabMenuItem(c, th, &items[i], i, th.ReduceMotion) {
						if items[i].OnClick != nil {
							items[i].OnClick()
						}
						*open = false
						handle.Focus(c)
					}
				}
			})

			// Absorb clicks on gaps within the menu column and dismiss via back.Clicked()
			// rather than PressedOutside() so concurrent overlays (e.g. SnackbarHost)
			// rendered above the FAB menu are unaffected.
			panel.Clicked()
			if closeBtn.Clicked() || back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape) {
				*open = false
				handle.Focus(c)
			}
		})
	})
	return trigger
}

// fabMenuItemMax is the most items a FABMenu takes.
const fabMenuItemMax = 6

// fabMenuCloseButton builds the high-contrast circular close button that covers
// the trigger while the menu shows, rotating its × in with the Fast Spatial
// spring.
func fabMenuCloseButton(c *ui.Context, th *Theme, trigger ui.Element, label string) ui.Element {
	btn := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Size(fabMenuCloseSide, fabMenuCloseSide).
		Label(label).
		Background(th.Colors.Primary).
		TextColor(th.Colors.OnPrimary).
		Radius(fabMenuCloseSide / 2)
	ApplyElevationShadows(th, btn, tokens.Level3)
	attachRippleWithContext(c, th, btn, StateLayerOptions{Color: th.Colors.OnPrimary, Radius: fabMenuCloseSide / 2})
	btn.AttachTo(trigger, ui.AnchorCenter, ui.AnchorCenter)

	entered := ui.Local(btn, fabMenuPoseKey{}, func() bool { return false })
	pose := ui.Pose{}
	if !*entered {
		*entered = true
		// ReduceMotion removes the entry pose entirely: posing Fade still
		// animates under reduced motion and would ask for a needless frame.
		if !th.ReduceMotion {
			pose = ui.Pose{Rotate: -90, Scale: 0.6, Fade: 1}
			c.Invalidate()
		}
	}
	btn.AnimatePose(pose, fabMenuEntryAnimation(th, 0))

	btn.Children(func() {
		drawIconBox(c, mdicon.Close.Draw, 24, th.Colors.OnPrimary)
	})
	return btn
}

// fabMenuItem builds one pill item of a FAB menu and reports whether it was
// chosen.
func fabMenuItem(c *ui.Context, th *Theme, item *FABMenuItem, i int, reduceMotion bool) bool {
	cs := th.Colors

	btn := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Height(fabMenuItemSide).
		PaddingX(16).
		Gap(12).
		Label(item.Label).
		Role(ui.RoleMenuItem)
	disabled := item.Disabled || btn.IsDisabled()
	pal := buttonColors{container: cs.SecondaryContainer, content: cs.OnSecondaryContainer}
	if disabled {
		pal = buttonColors{
			container: cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
			content:   cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity)),
		}
	}
	btn.Disabled(item.Disabled).
		TextColor(pal.content).
		Background(pal.container).
		Radius(fabMenuItemSide / 2)
	ApplyElevationShadows(th, btn, tokens.Level2)

	entered := ui.Local(btn, fabMenuPoseKey{i: i + 1}, func() bool { return false })
	pose := ui.Pose{}
	if !*entered {
		*entered = true
		if !reduceMotion {
			pose = ui.Pose{Y: 20, Scale: 0.85, Fade: 1}
			c.Invalidate()
		}
	}
	delay := 0
	if !reduceMotion {
		delay = i
	}
	btn.AnimatePose(pose, fabMenuEntryAnimation(th, delay))

	attachRippleWithContext(c, th, btn, StateLayerOptions{
		Color:    pal.content,
		Radius:   fabMenuItemSide / 2,
		Disabled: disabled,
	})

	clicked := btn.Clicked()
	btn.Children(func() {
		if item.Icon != nil {
			drawIconBox(c, item.Icon, 24, pal.content)
		}
		applyTypeStyleWithTheme(th, ui.Text(c, item.Label).NoWrap(), tokens.LabelLarge)
	})
	return clicked
}

// fabMenuEntryAnimation returns the spring that brings an overlay element in,
// with delay steps of i for the staggered items.
func fabMenuEntryAnimation(th *Theme, i int) ui.Animation {
	a := th.Animation(motion.FastSpatial)
	if i > 0 {
		a.Delay = time.Duration(i) * 40 * time.Millisecond
	}
	return a
}
