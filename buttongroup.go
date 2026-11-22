package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// ButtonGroupKind selects how a button group's buttons meet.
type ButtonGroupKind int

const (
	// ButtonGroupStandard keeps its buttons apart, each a full shape of its
	// own, and lets the pressed one widen as its neighbors give way.
	ButtonGroupStandard ButtonGroupKind = iota
	// ButtonGroupConnected joins its buttons with a tight gap and small inner
	// corners, the group's outer ends keeping the large shape radius; the
	// chosen button becomes a full pill.
	ButtonGroupConnected
)

// ButtonGroupOptions configures ButtonGroup, ButtonGroupMulti and
// ButtonGroupActions. The zero value is a standard, filled, extra small, round
// group that is enabled.
type ButtonGroupOptions struct {
	// Kind selects the Standard or Connected layout.
	Kind ButtonGroupKind
	// Variant selects the look of the group's buttons; the chosen buttons of a
	// selectable group use it, the others show an outline.
	Variant ButtonVariant
	// Size selects the XS..XL tier of every button.
	Size tokens.SizeTier
	// Shape selects the resting shape. Connected groups keep this radius on
	// their outer ends.
	Shape ButtonShape
	// Disabled disables the whole group.
	Disabled bool
	// Label names the group for assistive technology.
	Label string
}

// ButtonGroupItem is one button of a button group.
type ButtonGroupItem struct {
	// Label is the visible label and the accessible name.
	Label string
	// LeadingIcon draws an icon before the label.
	LeadingIcon IconDraw
	// TrailingIcon draws an icon after the label.
	TrailingIcon IconDraw
	// Disabled disables this button alone.
	Disabled bool
}

// Standard and Connected group geometry, in DIP.
const (
	standardGroupGap    = 8
	connectedGroupGap   = 2
	connectedInnerRadii = 8
	groupExpandDelta    = 6
)

// ButtonGroup builds a single-select Material Design 3 Expressive button group:
// a row of buttons, one of which *selected indexes, with the arrow keys moving
// the focus among them and Enter or Space choosing. A value outside the items
// chooses none.
//
// A Standard group keeps its buttons apart and widens the pressed one with the
// Fast Spatial spring while its neighbors give way; a Connected group joins
// them with small inner corners, its outer ends keeping the large radius, and
// morphs the chosen button into a full pill. Round groups mirror right to left:
//
//	md3e.ButtonGroup(c, &app.view, []md3e.ButtonGroupItem{
//		{Label: "Day"}, {Label: "Week"}, {Label: "Month"},
//	}, md3e.ButtonGroupOptions{Kind: md3e.ButtonGroupConnected})
func ButtonGroup(c *ui.Context, selected *int, items []ButtonGroupItem, opts ...ButtonGroupOptions) ui.Element {
	o := buttonGroupOptions(opts)
	return buildButtonGroup(c, items, o, func(i int) bool {
		return selected != nil && *selected == i
	}, func(i int) {
		if selected != nil {
			*selected = i
		}
	}, nil)
}

// ButtonGroupMulti builds a multi-select button group: each button toggles its
// slot of *selected, which grows to the number of items.
func ButtonGroupMulti(c *ui.Context, selected *[]bool, items []ButtonGroupItem, opts ...ButtonGroupOptions) ui.Element {
	o := buttonGroupOptions(opts)
	if selected != nil && len(*selected) < len(items) {
		*selected = append(*selected, make([]bool, len(items)-len(*selected))...)
	}
	return buildButtonGroup(c, items, o, func(i int) bool {
		return selected != nil && i < len(*selected) && (*selected)[i]
	}, func(i int) {
		if selected != nil && i < len(*selected) {
			(*selected)[i] = !(*selected)[i]
		}
	}, nil)
}

// ButtonGroupActions builds an action group: its buttons are commands rather
// than choices, and a click calls onAction with its index.
func ButtonGroupActions(c *ui.Context, items []ButtonGroupItem, onAction func(i int), opts ...ButtonGroupOptions) ui.Element {
	return buildButtonGroup(c, items, buttonGroupOptions(opts), func(int) bool { return true }, nil, onAction)
}

func buttonGroupOptions(opts []ButtonGroupOptions) ButtonGroupOptions {
	if len(opts) > 1 {
		panic("md3e: ButtonGroup takes at most one ButtonGroupOptions")
	}
	if len(opts) == 1 {
		return opts[0]
	}
	return ButtonGroupOptions{}
}

func buildButtonGroup(c *ui.Context, items []ButtonGroupItem, o ButtonGroupOptions, isActive func(i int) bool, choose func(i int), act func(i int)) ui.Element {
	th := CurrentTheme(c)
	o.Size = normalizeSizeTier(o.Size)
	spec := tierSpecForDensity(th.Density, o.Size)
	rtl := c.Direction() == ui.RightToLeft
	selectable := choose != nil
	outer := restingRadius(spec, o.Shape)
	squareR := float32(spec.SquareRadius)
	full := float32(spec.Height) / 2
	pad := float32(spec.Padding)
	iconGap := iconLabelGapForDensity(th.Density, o.Size)
	expandDelta := groupExpandDeltaForSize(o.Size)

	row := ui.Row(c).
		Shrink(0).
		Role(ui.RoleGroup).
		FocusGroup(ui.Horizontal).
		Gap(groupGapOf(o.Kind))
	if o.Label != "" {
		row.Label(o.Label)
	}

	n := len(items)
	row.Children(func() {
		// Read every segment's press before styling any: the neighbors of the
		// pressed button give way as it widens.
		var stackEls [12]ui.Element
		var stackPressed [12]bool
		var els []ui.Element
		var pressed []bool
		if n <= len(stackEls) {
			els, pressed = stackEls[:n], stackPressed[:n]
		} else {
			els, pressed = make([]ui.Element, n), make([]bool, n)
		}
		for i := range items {
			els[i] = ui.ButtonBase(c).
				DimDisabled(false).
				FocusRing(false).
				Height(float32(spec.Height)).
				PaddingX(pad).
				Gap(iconGap).
				Label(items[i].Label)
			pressed[i] = els[i].Pressed()
		}
		anyPressed := false
		for _, p := range pressed {
			anyPressed = anyPressed || p
		}

		for i := range items {
			item := &items[i]
			el := els[i]
			// A disabled ancestor also disables every segment.
			disabled := o.Disabled || item.Disabled || el.IsDisabled()
			active := isActive(i)

			var container, content, border ui.Color
			var elevation tokens.ElevationLevel
			if selectable && !disabled {
				selT := clampF32(animateEffect(c, el, th, animChan0, boolFloat(active), motion.FastEffects), 0, 1)
				offPal := groupSegmentPalette(th, o.Variant, true, false, false)
				onPal := groupSegmentPalette(th, o.Variant, true, true, false)
				container = mixOptionalColor(offPal.container, onPal.container, selT)
				content = offPal.content.MixOklab(onPal.content, selT)
				border = mixOptionalColor(offPal.border, onPal.border, selT)
				elevation = offPal.elevation
				if active {
					elevation = onPal.elevation
				}
			} else {
				pal := groupSegmentPalette(th, o.Variant, selectable, active, disabled)
				container, content, border, elevation = pal.container, pal.content, pal.border, pal.elevation
			}
			border = resolveButtonBorder(th, o.Variant, border, disabled)
			el.Disabled(o.Disabled || item.Disabled).TextColor(content)
			if container.A > 0 {
				el.Background(container)
			}
			if border.A > 0 {
				el.Border(1, border)
			}
			if pressed[i] && elevation > tokens.Level0 && elevation < tokens.Level5 {
				// MD3E lifts a pressed Elevated segment one step.
				elevation++
			}
			if elevation > tokens.Level0 {
				ApplyElevationShadows(th, el, elevation)
			}
			if selectable {
				el.Checked(active)
			}

			// The Standard group widens the pressed button with the Fast
			// Spatial spring, its neighbors giving way by the same total.
			if o.Kind == ButtonGroupStandard {
				expand := animateWithTheme(el, th, animChan5, groupExpandTargetFor(pressed[i], anyPressed, n, expandDelta), motion.FastSpatial)
				el.PaddingX(max(pad+expand, 0))
			}

			// Corners: Standard keeps the resting shape (and morphs Round <->
			// Square on selected toggles) while shrinking on press; Connected
			// joins seams with small radii, morphs the chosen button into a
			// full pill and shrinks when pressed.
			var target [4]float32
			switch o.Kind {
			case ButtonGroupConnected:
				target = connectedRadii(i, n, outer, connectedInnerRadii, rtl)
				if active && selectable {
					target = [4]float32{full, full, full, full}
				}
			default:
				r := outer
				if active && selectable {
					if o.Shape == ButtonShapeSquare {
						r = full
					} else {
						r = squareR
					}
				}
				target = [4]float32{r, r, r, r}
			}
			if pressed[i] {
				r := float32(spec.PressedRadius)
				for k := range target {
					target[k] = min(target[k], r)
				}
			}
			var radii [4]float32
			for k := 0; k < 4; k++ {
				radii[k] = max(animateWithTheme(el, th, animChan1+animChanKey(k), target[k], motion.FastSpatial), 0)
			}
			el.Radius(radii[0], radii[1], radii[2], radii[3])

			avgR := (radii[0] + radii[1] + radii[2] + radii[3]) * 0.25
			applyButtonHoverElevation(el, th, o.Variant, avgR, disabled || pressed[i])

			borderInset := float32(0)
			if border.A > 0 {
				borderInset = 1
			}
			attachRippleWithContext(c, th, el, StateLayerOptions{
				Color:       content,
				Radii:       radii,
				HasRadii:    true,
				BorderInset: borderInset,
				Disabled:    disabled,
			})

			clicked := el.Clicked()
			el.Children(func() {
				if item.LeadingIcon != nil {
					drawIconBox(c, item.LeadingIcon, float32(spec.IconSize), content)
				}
				buildButtonLabel(c, th, item.Label, o.Size, spec)
				if item.TrailingIcon != nil {
					drawIconBox(c, item.TrailingIcon, float32(spec.IconSize), content)
				}
			})
			if clicked {
				switch {
				case choose != nil:
					choose(i)
				case act != nil:
					act(i)
				}
			}
		}
	})
	return row
}

func groupExpandDeltaForSize(t tokens.SizeTier) float32 {
	switch normalizeSizeTier(t) {
	case tokens.SizeXS:
		return 4
	case tokens.SizeL:
		return 8
	case tokens.SizeXL:
		return 10
	default:
		return groupExpandDelta
	}
}

// groupGapOf returns the space between the buttons of a group kind.
func groupGapOf(k ButtonGroupKind) float32 {
	if k == ButtonGroupConnected {
		return connectedGroupGap
	}
	return standardGroupGap
}

// groupExpandTarget returns the padding expansion of one button of an n-button
// Standard group: the pressed button widens by groupExpandDelta, the others
// give way by the same total among them.
func groupExpandTarget(pressed, anyPressed bool, n int) float32 {
	return groupExpandTargetFor(pressed, anyPressed, n, groupExpandDelta)
}

func groupExpandTargetFor(pressed, anyPressed bool, n int, delta float32) float32 {
	if pressed {
		return delta
	}
	if !anyPressed || n <= 1 {
		return 0
	}
	return -delta / float32(n-1)
}

// connectedRadii returns the four corners of connected segment i of n: the
// group's outer ends keep outer, the seams use inner, mirrored right to left.
func connectedRadii(i, n int, outer, inner float32, rtl bool) [4]float32 {
	innerAll := [4]float32{inner, inner, inner, inner}
	if n <= 1 {
		return [4]float32{outer, outer, outer, outer}
	}
	first := i == 0
	last := i == n-1
	switch {
	case first && last:
		return [4]float32{outer, outer, outer, outer}
	case first && !rtl, last && rtl:
		// The start of the row: rounded on the left.
		return [4]float32{outer, innerAll[1], innerAll[2], outer}
	case last && !rtl, first && rtl:
		// The end of the row: rounded on the right.
		return [4]float32{innerAll[0], outer, outer, innerAll[3]}
	default:
		return innerAll
	}
}

// groupSegmentPalette resolves the variant-aware colors of one group button.
func groupSegmentPalette(th *Theme, variant ButtonVariant, selectable, active, disabled bool) buttonColors {
	cs := th.Colors
	if disabled {
		onSurface := cs.OnSurface
		content := onSurface.Alpha(float32(tokens.DisabledContentOpacity))
		fill := onSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		switch variant {
		case ButtonOutlined:
			if selectable && active {
				return buttonColors{container: fill, content: content, border: fill}
			}
			return buttonColors{content: content, border: fill}
		case ButtonText:
			return buttonColors{content: content}
		default:
			return buttonColors{container: fill, content: content}
		}
	}
	if !selectable {
		return buttonPaletteOf(th, variant, false)
	}
	if active {
		switch variant {
		case ButtonOutlined:
			return buttonColors{
				container: cs.SecondaryContainer,
				content:   cs.OnSecondaryContainer,
				border:    cs.SecondaryContainer,
			}
		case ButtonText:
			return buttonColors{
				container: cs.SecondaryContainer,
				content:   cs.OnSecondaryContainer,
			}
		case ButtonElevated:
			return buttonColors{
				container: cs.Primary,
				content:   cs.OnPrimary,
				elevation: tokens.Level1,
			}
		default:
			return buttonPaletteOf(th, variant, false)
		}
	}
	// Unselected segment in a selectable group: stay harmonious with variant.
	switch variant {
	case ButtonOutlined:
		border := cs.OutlineVariant
		if th.Contrast >= tokens.ContrastHigh || border.A == 0 {
			border = cs.Outline
		}
		return buttonColors{content: cs.OnSurfaceVariant, border: border}
	case ButtonText:
		return buttonColors{content: cs.OnSurfaceVariant}
	case ButtonElevated:
		return buttonColors{
			container: SurfaceColorForElevation(th, tokens.Level1),
			content:   cs.OnSurfaceVariant,
			elevation: tokens.Level1,
		}
	default:
		unselectedBg := cs.SurfaceContainerHigh
		if unselectedBg.A == 0 {
			unselectedBg = SurfaceColorForElevation(th, tokens.Level2)
		}
		return buttonColors{
			container: unselectedBg,
			content:   cs.OnSurfaceVariant,
		}
	}
}
