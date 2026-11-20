package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// Chip metrics in DIP, as MD3E specifies the four chip types: a 32 DIP tall
// container with 8 DIP corners, 16 DIP of padding without icons (8 with them),
// 18 DIP icons, a 24 DIP avatar slot and an 8 DIP gap between slots.
const (
	chipHeight     = 32
	chipRadius     = 8
	chipIconSize   = 18
	chipAvatarSize = 24
	chipIconGap    = 8
	chipPad        = 16
	chipIconPad    = 8
	chipAvatarPad  = 4
	chipCloseSize  = 24
	chipCloseIcon  = 18
)

// ChipOptions configures AssistChip and SuggestionChip. The zero value is an
// enabled, outlined chip with no icons.
type ChipOptions struct {
	// Elevated paints SurfaceContainerLow with MD3E Level1 shadows instead of
	// the default outlined look.
	Elevated bool
	// LeadingIcon draws an 18 DIP icon before the label.
	LeadingIcon IconDraw
	// TrailingIcon draws an 18 DIP icon after the label.
	TrailingIcon IconDraw
	// Disabled disables the chip: it takes no pointer, keys or focus, and
	// shows MD3E's disabled colors without ui's automatic dimming.
	Disabled bool
	// Label overrides the accessible name. By default the visible label names
	// the chip.
	Label string
	// OnClick runs when the chip is clicked, or with Enter or Space while it
	// has the keyboard focus.
	OnClick func()
}

// AssistChip builds a Material Design 3 Expressive assist chip that performs
// an action, as "Add to calendar": a ui.ButtonBase with the 32 DIP chip
// container, an optional leading icon in Primary, the label in LabelLarge, MD3E
// state layer and ripple, and either an OutlineVariant border or the elevated
// SurfaceContainerLow fill. The whole box is the hit target.
//
//	md3e.AssistChip(c, "Add", md3e.ChipOptions{
//		LeadingIcon: mdicon.Add.Draw,
//		OnClick:     app.add,
//	})
func AssistChip(c *ui.Context, label string, opts ...ChipOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: AssistChip takes at most one ChipOptions")
	}
	var o ChipOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildActionChip(c, label, o, "md3e: AssistChip requires a label")
}

// SuggestionChip builds a Material Design 3 Expressive suggestion chip that
// proposes a completion, as "Try a sample": the same geometry and looks as
// AssistChip, for a suggestion rather than an action.
func SuggestionChip(c *ui.Context, label string, opts ...ChipOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: SuggestionChip takes at most one ChipOptions")
	}
	var o ChipOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	return buildActionChip(c, label, o, "md3e: SuggestionChip requires a label")
}

// buildActionChip builds an AssistChip or SuggestionChip: a ui.ButtonBase with
// the chip's container, border, elevation, leading and trailing icons.
func buildActionChip(c *ui.Context, label string, o ChipOptions, panicMsg string) ui.Element {
	if label == "" && o.Label == "" {
		panic(panicMsg)
	}
	th := CurrentTheme(c)
	el := ui.ButtonBase(c).
		DimDisabled(false).
		FocusRing(false).
		Height(chipHeight).
		Gap(chipIconGap).
		Shrink(0)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)
	pal := chipPaletteOf(th, o.Elevated, false, disabled)
	if pal.container.A > 0 {
		el.Background(pal.container)
	}
	if pal.border.A > 0 {
		el.Border(1, pal.border)
	}
	el.Radius(chipRadius).TextColor(pal.content)
	if o.Elevated && !disabled {
		ApplyElevationShadows(th, el, tokens.Level1)
	}

	padStart, padEnd := float32(chipPad), float32(chipPad)
	if o.LeadingIcon != nil {
		padStart = chipIconPad
	}
	if o.TrailingIcon != nil {
		padEnd = chipIconPad
	}
	el.PaddingStart(padStart).PaddingEnd(padEnd)

	borderInset := float32(0)
	if pal.border.A > 0 {
		borderInset = 1
	}
	AttachRipple(c, el, StateLayerOptions{
		Color:       pal.content,
		Radius:      chipRadius,
		BorderInset: borderInset,
		Disabled:    disabled,
	})

	name := labelOr(o.Label, label)
	el.Label(name)
	clicked := el.Clicked()
	el.Children(func() {
		if o.LeadingIcon != nil {
			drawIconBox(c, o.LeadingIcon, chipIconSize, pal.icon)
		}
		ApplyTypeStyle(c, ui.Text(c, label).NoWrap().Shrink(0).TextColor(pal.content), tokens.LabelLarge)
		if o.TrailingIcon != nil {
			drawIconBox(c, o.TrailingIcon, chipIconSize, pal.icon)
		}
	})
	if clicked && o.OnClick != nil {
		o.OnClick()
	}
	return el
}

// FilterChipOptions configures FilterChip. The zero value is an enabled,
// outlined filter chip.
type FilterChipOptions struct {
	// Elevated paints SurfaceContainerLow with MD3E Level1 shadows instead of
	// the default outlined look.
	Elevated bool
	// LeadingIcon draws an 18 DIP icon before the label while the chip is
	// unselected; selecting it replaces the icon with the check mark.
	LeadingIcon IconDraw
	// TrailingIcon draws an 18 DIP icon after the label.
	TrailingIcon IconDraw
	// Disabled disables the chip: it takes no pointer, keys or focus, and
	// still reports its selected state to assistive technology.
	Disabled bool
	// Label overrides the accessible name. By default the visible label names
	// the chip.
	Label string
	// OnChanged runs with the new selected state after a click, Space, or
	// Enter toggles it.
	OnChanged func(selected bool)
}

// FilterChip animation channels.
type (
	filterSelectKey struct{}
	filterCheckKey  struct{}
)

// FilterChip builds a Material Design 3 Expressive filter chip over *selected,
// as a tag filter: a ui.ToggleBase with the chip container, which a click, or
// Space while it has the keyboard focus, toggles and Changed reports. Selecting
// it springs the container to SecondaryContainer without a border, switches the
// label and icons to OnSecondaryContainer with the Fast Effects spring, and
// scales mdicon.Check into the leading slot with the Fast Spatial spring;
// deselecting springs it back. Assistive technology sees a check box whose
// checked state follows *selected.
//
//	md3e.FilterChip(c, &app.onlyUnread, "Unread", md3e.FilterChipOptions{
//		LeadingIcon: mdicon.Star.Draw,
//	})
func FilterChip(c *ui.Context, selected *bool, label string, opts ...FilterChipOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: FilterChip takes at most one FilterChipOptions")
	}
	if selected == nil {
		panic("md3e: FilterChip requires a non-nil *bool")
	}
	if label == "" {
		panic("md3e: FilterChip requires a label")
	}
	var o FilterChipOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	el := ui.ToggleBase(c, selected).
		Role(ui.RoleCheckBox).
		Checked(*selected).
		DimDisabled(false).
		FocusRing(false).
		Height(chipHeight).
		Gap(chipIconGap).
		Shrink(0)
	disabled := o.Disabled || el.IsDisabled()
	el.Disabled(o.Disabled)

	pal := chipPaletteOf(th, o.Elevated, true, disabled)
	el.Radius(chipRadius)

	padStart, padEnd := float32(chipPad), float32(chipPad)
	if o.LeadingIcon != nil {
		padStart = chipIconPad
	}
	if o.TrailingIcon != nil {
		padEnd = chipIconPad
	}
	el.PaddingStart(padStart).PaddingEnd(padEnd)

	spatial := th.Animation(motion.FastSpatial)
	effects := th.Animation(motion.FastEffects)
	if th.ReduceMotion {
		spatial, effects = motion.Snap, motion.Snap
	}
	selectT := clampF32(el.AnimateTo(filterSelectKey{}, boolFloat(*selected), effects), 0, 1)
	checkT := clampF32(el.AnimateTo(filterCheckKey{}, boolFloat(*selected), spatial), 0, 1.1)

	if o.Elevated && !disabled {
		ApplyElevationShadows(th, el, tokens.Level1)
	}

	el.Draw(func(p *ui.Painter, r ui.Rect) {
		drawFilterChip(p, r, selectT, pal)
	})
	AttachRipple(c, el, StateLayerOptions{
		Color:    pal.content.Mix(pal.selectedContent, selectT),
		Radius:   chipRadius,
		Disabled: disabled,
	})

	name := labelOr(o.Label, label)
	el.Label(name)
	changed := el.Changed()
	content := pal.content.Mix(pal.selectedContent, selectT)
	iconColor := pal.icon.Mix(pal.selectedContent, selectT)
	showIcon := o.LeadingIcon != nil || checkT > 0.001
	el.Children(func() {
		if showIcon {
			ui.Box(c).Size(chipIconSize, chipIconSize).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
				if o.LeadingIcon != nil {
					if alpha := 1 - checkT; alpha > 0.001 {
						o.LeadingIcon(p, r, iconColor.Alpha(alpha))
					}
				}
				if checkT > 0.001 {
					cx, cy := r.X+r.W/2, r.Y+r.H/2
					s := 0.4 + 0.6*checkT
					p.Transform(ui.Scaling(s, s), cx, cy, func() {
						mdicon.Check.Draw(p, r, content)
					})
				}
			})
		}
		ApplyTypeStyle(c, ui.Text(c, label).NoWrap().Shrink(0).TextColor(content), tokens.LabelLarge)
		if o.TrailingIcon != nil {
			drawIconBox(c, o.TrailingIcon, chipIconSize, iconColor)
		}
	})
	if changed && o.OnChanged != nil {
		o.OnChanged(*selected)
	}
	return el
}

// InputChipOptions configures InputChip. The zero value is an enabled,
// outlined chip with an accessible close button.
type InputChipOptions struct {
	// Elevated paints SurfaceContainerLow with MD3E Level1 shadows instead of
	// the default outlined look.
	Elevated bool
	// LeadingIcon draws a 24 DIP avatar when Avatar is set, or an 18 DIP icon
	// otherwise, before the label.
	LeadingIcon IconDraw
	// Avatar clips LeadingIcon to a 24 DIP circle and tightens the start
	// padding, as an input chip shows a person or entity.
	Avatar bool
	// TrailingIcon draws an 18 DIP icon after the label, before the close
	// button.
	TrailingIcon IconDraw
	// Disabled disables the whole chip, its close button too.
	Disabled bool
	// Label overrides the accessible name of the chip's body. By default the
	// visible label names it.
	Label string
	// RemoveLabel overrides the accessible name of the close button; "Remove
	// <label>" by default.
	RemoveLabel string
	// OnClick runs when the body of the chip is clicked, or with Enter or
	// Space while it has the keyboard focus.
	OnClick func()
	// OnRemove runs when the close button is clicked, or pressed with Enter or
	// Space while it has the keyboard focus.
	OnRemove func()
}

// InputChip builds a Material Design 3 Expressive input chip, as a contact or
// filter token the user entered: a 32 DIP container holding a body
// ui.ButtonBase (an optional avatar or leading icon, the label, and an optional
// trailing icon) and a separate 24 DIP close button drawing mdicon.Close. The
// body and the close button are siblings with independent hit regions, MD3E
// ripples and accessible names, so clicking one never reports the other; the
// close button defaults to the name "Remove <label>".
//
//	md3e.InputChip(c, person.Name, md3e.InputChipOptions{
//		LeadingIcon: person.Avatar,
//		Avatar:      true,
//		OnRemove:    func() { app.remove(person) },
//	})
func InputChip(c *ui.Context, label string, opts ...InputChipOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: InputChip takes at most one InputChipOptions")
	}
	if label == "" && (len(opts) == 0 || opts[0].Label == "") {
		panic("md3e: InputChip requires a label")
	}
	var o InputChipOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	rtl := c.Direction() == ui.RightToLeft
	outer := ui.Row(c).Height(chipHeight).Shrink(0).DimDisabled(false)
	disabled := o.Disabled || outer.IsDisabled()

	pal := chipPaletteOf(th, o.Elevated, false, disabled)
	if pal.container.A > 0 {
		outer.Background(pal.container)
	}
	if pal.border.A > 0 {
		outer.Border(1, pal.border)
	}
	outer.Radius(chipRadius).TextColor(pal.content)
	if o.Elevated && !disabled {
		ApplyElevationShadows(th, outer, tokens.Level1)
	}

	bodyRadii := [4]float32{chipRadius, 0, 0, chipRadius}
	closeRadii := [4]float32{0, chipRadius, chipRadius, 0}
	if rtl {
		bodyRadii = [4]float32{0, chipRadius, chipRadius, 0}
		closeRadii = [4]float32{chipRadius, 0, 0, chipRadius}
	}

	bodyInk := pal.content
	iconInk := pal.icon
	closeInk := cs.OnSurfaceVariant
	if disabled {
		bodyInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		iconInk = bodyInk
		closeInk = bodyInk
	}
	removeLabel := o.RemoveLabel
	if removeLabel == "" {
		removeLabel = "Remove " + label
	}
	padStart := float32(chipPad)
	switch {
	case o.Avatar && o.LeadingIcon != nil:
		padStart = chipAvatarPad
	case o.LeadingIcon != nil:
		padStart = chipIconPad
	}

	var body, closeEl ui.Element
	outer.Children(func() {
		body = ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			Height(chipHeight).
			Gap(chipIconGap).
			Shrink(0).
			Disabled(disabled).
			PaddingStart(padStart).
			PaddingEnd(4).
			Label(labelOr(o.Label, label))
		body.TextColor(bodyInk)
		AttachRipple(c, body, StateLayerOptions{
			Color:    bodyInk,
			Radii:    bodyRadii,
			HasRadii: true,
			Disabled: disabled,
		})
		body.Children(func() {
			if o.LeadingIcon != nil {
				size := float32(chipIconSize)
				if o.Avatar {
					size = chipAvatarSize
				}
				ui.Box(c).Size(size, size).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
					if o.Avatar {
						p.Clip(r, size/2, func() {
							o.LeadingIcon(p, r, iconInk)
						})
						return
					}
					o.LeadingIcon(p, r, iconInk)
				})
			}
			ApplyTypeStyle(c, ui.Text(c, label).NoWrap().Shrink(0).TextColor(bodyInk), tokens.LabelLarge)
			if o.TrailingIcon != nil {
				drawIconBox(c, o.TrailingIcon, chipIconSize, iconInk)
			}
		})

		closeEl = ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			Size(chipCloseSize, chipCloseSize).
			Shrink(0).
			Disabled(disabled).
			Label(removeLabel).
			TextColor(closeInk)
		AttachRipple(c, closeEl, StateLayerOptions{
			Color:    closeInk,
			Radii:    closeRadii,
			HasRadii: true,
			Disabled: disabled,
		})
		closeEl.Children(func() {
			drawIconBox(c, mdicon.Close.Draw, chipCloseIcon, closeInk)
		})
	})

	// The body and the close button are siblings, so each has its own hit
	// region: a click on one never commits the other.
	bodyClicked := body.Clicked()
	closeClicked := closeEl.Clicked()
	if bodyClicked && o.OnClick != nil {
		o.OnClick()
	}
	if closeClicked && o.OnRemove != nil {
		o.OnRemove()
	}
	return outer
}

// ChipGroupOptions configures ChipGroup. The zero value spaces chips 8 DIP
// apart in both directions.
type ChipGroupOptions struct {
	// Label names the group for assistive technology.
	Label string
	// RowGap is the vertical space between wrapped lines, 8 DIP by default.
	RowGap float32
	// ColumnGap is the horizontal space between chips, 8 DIP by default.
	ColumnGap float32
}

// ChipGroup builds a wrapping flow container for chips, as a tag editor or a
// filter bar: a ui.Row with Wrap, RoleGroup, the configured column and row
// gaps, and the chips that fn builds. Chips that do not fit on a line flow
// onto the next, right to left when the window reads so.
//
//	md3e.ChipGroup(c, func() {
//		for _, tag := range app.tags {
//			md3e.InputChip(c, tag, md3e.InputChipOptions{OnRemove: ...})
//		}
//	}, md3e.ChipGroupOptions{Label: "Tags"})
func ChipGroup(c *ui.Context, fn func(), opts ...ChipGroupOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: ChipGroup takes at most one ChipGroupOptions")
	}
	var o ChipGroupOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	rowGap := o.RowGap
	if rowGap <= 0 {
		rowGap = 8
	}
	colGap := o.ColumnGap
	if colGap <= 0 {
		colGap = 8
	}
	g := ui.Row(c).
		Wrap().
		AlignItems(ui.Center).
		Role(ui.RoleGroup).
		GapY(rowGap).
		GapX(colGap)
	if o.Label != "" {
		g.Label(o.Label)
	}
	if fn != nil {
		g.Children(fn)
	}
	return g
}

// chipColors are the resolved colors of one chip look in one state.
type chipColors struct {
	container         ui.Color
	selectedContainer ui.Color
	content           ui.Color
	selectedContent   ui.Color
	icon              ui.Color
	border            ui.Color
}

// chipPaletteOf resolves the MD3E colors of a chip: outlined chips keep a
// transparent container with an OutlineVariant border, elevated ones fill
// SurfaceContainerLow, and selected filter chips fill SecondaryContainer with
// OnSecondaryContainer content and no border. Disabled chips use OnSurface at
// MD3E's 12% container and 38% content instead of ui's automatic dimming.
func chipPaletteOf(th *Theme, elevated, filter, disabled bool) chipColors {
	cs := th.Colors
	if disabled {
		onSurface := cs.OnSurface
		content := onSurface.Alpha(float32(tokens.DisabledContentOpacity))
		fill := onSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		out := chipColors{
			content:         content,
			selectedContent: content,
			icon:            content,
		}
		if elevated {
			out.container = fill
		}
		out.border = fill
		if filter {
			out.selectedContainer = fill
			if !elevated {
				out.border = fill
			}
		}
		return out
	}
	container := ui.Transparent
	if elevated {
		container = cs.SurfaceContainerLow
		if container.A == 0 {
			container = SurfaceColorForElevation(th, tokens.Level1)
		}
	}
	out := chipColors{
		container:       container,
		content:         cs.OnSurface,
		selectedContent: cs.OnSecondaryContainer,
		icon:            cs.Primary,
		border:          cs.OutlineVariant,
	}
	if elevated {
		// MD3E elevated chips carry their container instead of a border.
		out.border = ui.Transparent
	}
	if filter {
		out.selectedContainer = cs.SecondaryContainer
	}
	return out
}

// drawFilterChip paints a filter chip's container: the unselected outlined or
// elevated fill and border cross-fading into the selected SecondaryContainer
// with no border, by t from 0 to 1.
func drawFilterChip(p *ui.Painter, r ui.Rect, t float32, pal chipColors) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	if pal.container.A > 0 {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, pal.container, chipRadius)
	}
	if t > 0.001 && pal.selectedContainer.A > 0 {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}, pal.selectedContainer.Alpha(t), chipRadius)
	}
	if a := 1 - t; a > 0.01 && pal.border.A > 0 {
		p.Stroke(r, pal.border.Alpha(a), chipRadius, 1)
	}
}
