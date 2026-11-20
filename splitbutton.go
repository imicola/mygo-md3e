package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// SplitButtonOptions configures SplitButton. The zero value is a filled, extra
// small split button that is enabled.
type SplitButtonOptions struct {
	// Variant selects the look: Filled, Tonal, Elevated or Outlined. The text
	// variant has no container to split, and falls back to Outlined.
	Variant ButtonVariant
	// Size selects the XS..XL tier.
	Size tokens.SizeTier
	// Disabled disables the whole control, the arrow included.
	Disabled bool
	// ActionDisabled disables the leading action alone, leaving the menu.
	ActionDisabled bool
	// LeadingIcon draws an icon before the action's label, at the tier's
	// icon size.
	LeadingIcon IconDraw
	// ArrowLabel names the menu trigger for assistive technology; "More
	// options" when empty.
	ArrowLabel string
	// Open lets the caller own the menu's expanded state, as when the app opens
	// it from elsewhere; nil keeps the state in the control.
	Open *bool
	// Menu builds the menu's content in panel whenever the menu shows. Without
	// one the arrow has nothing to open.
	Menu func(panel ui.Element)
}

// splitOpenKey is the local holding a split button's expansion state when the
// caller does not own it.
type splitOpenKey struct{}

// splitChevronKey animates the arrow's chevron rotation in degrees.
type splitChevronKey struct{}

// splitCornerKey animates one corner of a split button segment.
type splitCornerKey struct{ i int }

// SplitButton builds a Material Design 3 Expressive split button: a leading
// ui.ButtonBase doing the action joined to a trailing ui.ButtonBase whose
// chevron opens a menu panel through ui.PopoverBase. Its two segments share the
// variant's container and border, with the outer ends as round as the shape and
// the seam corners small, mirrored right to left. Opening the menu morphs the
// arrow segment into a symmetric pill with the Default Spatial spring and turns
// its chevron 180 degrees; closing restores both.
//
// Clicking the arrow, Enter, Space or the up and down arrows on it, and
// Alt+Down on either segment toggle the menu; Escape and a press outside close
// it. Assistive technology sees a group of the action button and a menu button,
// expanded while the menu shows:
//
//	group, action, _, _ := md3e.SplitButton(c, "Save", md3e.SplitButtonOptions{
//		Menu: func(panel ui.Element) {
//			panel.Children(func() {
//				if ui.ButtonBase(c).Label("Save as").Clicked() {
//					app.saveAs()
//				}
//			})
//		},
//	})
//	if action.Clicked() {
//		app.save()
//	}
//
// group is the RoleGroup already placed among its siblings, which a caller may
// style further; action and arrow are its segments, read in the pass that built
// them; open reports and sets whether the menu shows, owned by the caller
// through SplitButtonOptions.Open or kept in a local of group otherwise. They
// are separate values because go tool mygo vet forbids ui.Element fields in
// structs: an element expires before the next build pass.
func SplitButton(c *ui.Context, label string, opts ...SplitButtonOptions) (group, action, arrow ui.Element, open *bool) {
	if len(opts) > 1 {
		panic("md3e: SplitButton takes at most one SplitButtonOptions")
	}
	var o SplitButtonOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Variant == ButtonText {
		o.Variant = ButtonOutlined
	}

	th := CurrentTheme(c)
	o.Size = normalizeSizeTier(o.Size)
	spec := tierSpec(o.Size)
	rtl := c.Direction() == ui.RightToLeft
	outer := restingRadius(spec, ButtonShapeRound)
	inner := float32(connectedInnerRadii)
	pad := float32(spec.Padding)

	group = ui.Row(c).
		AlignItems(ui.Stretch).
		Shrink(0).
		Height(float32(spec.Height)).
		Gap(connectedGroupGap).
		DimDisabled(false).
		Role(ui.RoleGroup).
		FocusGroup(ui.Horizontal).
		Label(label)
	// A disabled ancestor also disables the control and closes its menu.
	groupDisabled := o.Disabled || group.IsDisabled()
	if groupDisabled {
		group.Disabled(true)
	}
	pal := buttonPaletteOf(th, o.Variant, groupDisabled)
	// The action resolves its own palette: disabling it alone must show the
	// MD3E disabled container and content while the arrow stays enabled.
	actionPal := buttonPaletteOf(th, o.Variant, groupDisabled || o.ActionDisabled)

	open = o.Open
	if open == nil {
		open = ui.Local(group, splitOpenKey{}, func() bool { return false })
	}
	if groupDisabled {
		*open = false
	}

	actionLabel := label
	arrowLabel := o.ArrowLabel
	if arrowLabel == "" {
		arrowLabel = "More options"
	}

	group.Children(func() {
		action = ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			PaddingX(pad).
			Gap(iconLabelGap(o.Size)).
			Disabled(o.Disabled || o.ActionDisabled).
			Label(actionLabel).
			TextColor(actionPal.content)
		arrow = ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			PaddingX(pad * 0.75).
			Disabled(o.Disabled).
			Label(arrowLabel).
			TextColor(pal.content).
			HasPopup(ui.PopupMenu).
			Expanded(*open)
	})

	// The seam corners are small; the outer end of each segment keeps the
	// shape radius, mirrored right to left.
	actionTarget := [4]float32{outer, inner, inner, outer}
	arrowTarget := [4]float32{inner, outer, outer, inner}
	if rtl {
		actionTarget, arrowTarget = arrowTarget, actionTarget
	}
	if *open {
		// The arrow opens into a symmetric pill.
		arrowTarget = [4]float32{outer, outer, outer, outer}
	}
	pressedRadius := float32(spec.PressedRadius)
	// Pressing shrinks the segment's corners toward the tier's pressed radius;
	// a seam corner keeps its smaller radius instead of inflating to it.
	press := func(base [4]float32) [4]float32 {
		for i := range base {
			base[i] = min(base[i], pressedRadius)
		}
		return base
	}
	if action.Pressed() {
		actionTarget = press(actionTarget)
	}
	if arrow.Pressed() && !*open {
		arrowTarget = press(arrowTarget)
	}

	// The Elevated variant lifts each segment on its own, so that the shadow
	// follows the arrow's pill and the seam's 2 DIP gap rather than the group's
	// joint box; the pressed segment rises from Level1 to Level2.
	actionElevation := actionPal.elevation
	if action.Pressed() && actionElevation > tokens.Level0 && actionElevation < tokens.Level5 {
		actionElevation++
	}
	arrowElevation := pal.elevation
	if arrow.Pressed() && arrowElevation > tokens.Level0 && arrowElevation < tokens.Level5 {
		arrowElevation++
	}

	styleSplitSegment(c, th, action, actionPal, actionTarget, actionElevation)
	styleSplitSegment(c, th, arrow, pal, arrowTarget, arrowElevation)

	// The chevron turns 180 degrees as the menu opens and back as it closes.
	degrees := arrow.AnimateTo(splitChevronKey{}, splitChevronDegrees(*open), th.Animation(motion.DefaultSpatial))
	action.Children(func() {
		if o.LeadingIcon != nil {
			drawIconBox(c, o.LeadingIcon, float32(spec.IconSize), actionPal.content)
		}
		if label != "" {
			ApplyTypeStyle(c, ui.Text(c, label).NoWrap(), buttonTypeRole(o.Size))
		}
	})
	arrow.Children(func() {
		ui.Box(c).Size(float32(spec.IconSize), float32(spec.IconSize)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			if degrees == 0 {
				mdicon.ChevronDown.Draw(p, r, pal.content)
				return
			}
			p.Transform(ui.Rotation(degrees), r.X+r.W/2, r.Y+r.H/2, func() {
				mdicon.ChevronDown.Draw(p, r, pal.content)
			})
		})
	})

	// Toggling: a click on the arrow and Alt+Down on either segment. Up and
	// Down on the arrow open the menu when it is closed, as a menu button
	// does, and leave an open menu open.
	toggle := arrow.Clicked() ||
		action.Shortcut(ui.Alt, ui.KeyDown) || arrow.Shortcut(ui.Alt, ui.KeyDown)
	openOnArrow := arrow.Shortcut(0, ui.KeyUp) || arrow.Shortcut(0, ui.KeyDown)
	if o.Menu != nil {
		if toggle {
			*open = !*open
		}
		if openOnArrow {
			*open = true
		}
	}

	if o.Menu != nil {
		ui.PopoverBase(c, arrow, open, func(panel ui.Element) {
			panel.Background(SurfaceColorForElevation(th, tokens.Level2)).
				Radius(th.Shapes.Radius(tokens.ShapeExtraSmall)).
				Padding(8)
			ApplyElevationShadows(th, panel, tokens.Level2)
			o.Menu(panel)
		})
	}
	return group, action, arrow, open
}

// styleSplitSegment gives one segment its container, border, elevation, corner
// morph and state layer, and stretches it to the group's height.
func styleSplitSegment(c *ui.Context, th *Theme, el ui.Element, pal buttonColors, target [4]float32, elevation tokens.ElevationLevel) {
	if pal.container.A > 0 {
		el.Background(pal.container)
	}
	if pal.border.A > 0 {
		el.Border(1, pal.border)
	}
	if elevation > tokens.Level0 {
		ApplyElevationShadows(th, el, elevation)
	}
	var radii [4]float32
	for i := 0; i < 4; i++ {
		radii[i] = el.AnimateTo(splitCornerKey{i: i}, target[i], th.Animation(motion.FastSpatial))
	}
	el.Radius(radii[0], radii[1], radii[2], radii[3])
	borderInset := float32(0)
	if pal.border.A > 0 {
		borderInset = 1
	}
	AttachRipple(c, el, StateLayerOptions{
		Color:       pal.content,
		Radii:       radii,
		HasRadii:    true,
		BorderInset: borderInset,
		Disabled:    el.IsDisabled(),
	})
}

// splitChevronDegrees returns the arrow's rotation: 180 degrees while the menu
// shows, 0 while it is closed.
func splitChevronDegrees(open bool) float32 {
	if open {
		return 180
	}
	return 0
}
