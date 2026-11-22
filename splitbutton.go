package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
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
	spec := tierSpecForDensity(th.Density, o.Size)
	rtl := c.Direction() == ui.RightToLeft
	outer := restingRadius(spec, ButtonShapeRound)
	inner := float32(connectedInnerRadii)
	pad := float32(spec.Padding)
	iconGap := iconLabelGapForDensity(th.Density, o.Size)

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
		open = localBool(group, animChan0)
	}
	if groupDisabled {
		*open = false
	}

	arrowPal := pal
	if *open && o.Variant == ButtonOutlined && !groupDisabled {
		arrowPal.container = th.Colors.SecondaryContainer
		arrowPal.content = th.Colors.OnSecondaryContainer
	}

	actionLabel := label
	arrowLabel := o.ArrowLabel
	if arrowLabel == "" {
		arrowLabel = "More options"
	}

	// Optical balance on the D-shaped closed arrow: give slightly more padding
	// on the curved outer side so the chevron sits centered in the visible mass.
	arrowPadStart := pad * 0.68
	arrowPadEnd := pad * 0.82
	if *open {
		arrowPadStart = pad * 0.75
		arrowPadEnd = pad * 0.75
	}

	group.Children(func() {
		action = ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			PaddingX(pad).
			Gap(iconGap).
			Disabled(o.Disabled || o.ActionDisabled).
			Label(actionLabel).
			TextColor(actionPal.content)
		arrow = ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			PaddingStart(arrowPadStart).
			PaddingEnd(arrowPadEnd).
			Disabled(o.Disabled).
			Label(arrowLabel).
			TextColor(arrowPal.content).
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
	actionPressed := action.Pressed()
	arrowPressed := arrow.Pressed()
	if actionPressed {
		actionTarget = press(actionTarget)
	}
	if arrowPressed && !*open {
		arrowTarget = press(arrowTarget)
	}

	// The Elevated variant lifts each segment on its own, so that the shadow
	// follows the arrow's pill and the seam's 2 DIP gap rather than the group's
	// joint box; the pressed segment rises from Level1 to Level2.
	actionElevation := actionPal.elevation
	if actionPressed && actionElevation > tokens.Level0 && actionElevation < tokens.Level5 {
		actionElevation++
	}
	arrowElevation := arrowPal.elevation
	if arrowPressed && arrowElevation > tokens.Level0 && arrowElevation < tokens.Level5 {
		arrowElevation++
	}

	arrowSpring := motion.DefaultSpatial
	if arrowPressed && !*open {
		arrowSpring = motion.FastSpatial
	}
	styleSplitSegment(c, th, action, o.Variant, actionPal, actionTarget, actionElevation, actionPressed, motion.FastSpatial)
	styleSplitSegment(c, th, arrow, o.Variant, arrowPal, arrowTarget, arrowElevation, arrowPressed, arrowSpring)

	// The chevron turns 180 degrees as the menu opens and back as it closes.
	degrees := animateWithTheme(arrow, th, animChan4, splitChevronDegrees(*open), motion.DefaultSpatial)
	action.Children(func() {
		if o.LeadingIcon != nil {
			drawIconBox(c, o.LeadingIcon, float32(spec.IconSize), actionPal.content)
		}
		buildButtonLabel(c, th, label, o.Size, spec)
	})
	arrow.Children(func() {
		ui.Box(c).Size(float32(spec.IconSize), float32(spec.IconSize)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			if degrees == 0 {
				mdicon.ChevronDown.Draw(p, r, arrowPal.content)
				return
			}
			p.Transform(ui.Rotation(degrees), r.X+r.W/2, r.Y+r.H/2, func() {
				mdicon.ChevronDown.Draw(p, r, arrowPal.content)
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
		if inExitingOverlay(c) && *open {
			*open = false
		}
		reduce := th.ReduceMotion || c.Preferences().ReduceMotion
		panel := ui.PopoverBase(c, arrow, open, func(panel ui.Element) {
			panel.MinWidth(112).
				Background(SurfaceColorForElevation(th, tokens.Level2)).
				TextColor(th.Colors.OnSurface).
				Radius(th.Shapes.Radius(tokens.ShapeLarge)).
				Padding(6)
			if hl := resolveHairlineColor(th, ui.Color{}); hl.A > 0 {
				panel.Border(1, hl)
			}
			ApplyElevationShadows(th, panel, tokens.Level2)
			o.Menu(panel)
		})
		if panel.Valid() {
			panel.AttachTo(arrow, ui.AnchorBottomEnd, ui.AnchorTopEnd).
				AttachedOrigin().
				Margin(4, 0, 4, 0).
				AttachEdgeMargin(8)
			if !reduce {
				panel.Transition(ui.ElementTransition{
					Duration: 140 * time.Millisecond,
					Enter:    &ui.Motion{Opacity: 0, Scale: 0.94, Y: -4, Flip: true},
				})
			}
		}
	}
	return group, action, arrow, open
}

// styleSplitSegment gives one segment its container, border, elevation, corner
// morph and state layer, and stretches it to the group's height.
func styleSplitSegment(c *ui.Context, th *Theme, el ui.Element, variant ButtonVariant, pal buttonColors, target [4]float32, elevation tokens.ElevationLevel, pressed bool, cornerRole motion.Token) {
	disabled := el.IsDisabled()
	if pal.container.A > 0 {
		el.Background(pal.container)
	}
	border := resolveButtonBorder(th, variant, pal.border, disabled)
	if border.A > 0 {
		el.Border(1, border)
	}
	if elevation > tokens.Level0 {
		ApplyElevationShadows(th, el, elevation)
	}
	var radii [4]float32
	for i := 0; i < 4; i++ {
		radii[i] = max(animateWithTheme(el, th, animChan0+animChanKey(i), target[i], cornerRole), 0)
	}
	el.Radius(radii[0], radii[1], radii[2], radii[3])
	avgR := (radii[0] + radii[1] + radii[2] + radii[3]) * 0.25
	applyButtonHoverElevation(el, th, variant, avgR, disabled || pressed)
	borderInset := float32(0)
	if border.A > 0 {
		borderInset = 1
	}
	attachRippleWithContext(c, th, el, StateLayerOptions{
		Color:       pal.content,
		Radii:       radii,
		HasRadii:    true,
		BorderInset: borderInset,
		Disabled:    disabled,
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
