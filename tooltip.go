package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/tokens"
)

// PlainTooltipOptions configures PlainTooltip.
type PlainTooltipOptions struct {
	// Description is the help text assistive technology reads after the
	// anchor's name; text when empty.
	Description string
}

// PlainTooltip shows text in a Material Design 3 plain tooltip beside anchor
// once the pointer rests on it or the keyboard focus comes to it, and sets the
// anchor's description so screen readers read the same help. The tip uses
// InverseSurface/InverseOnSurface, goes with Escape or a press, and the pointer
// goes through it, as Base UI's tooltip: it is not interactive.
//
// It returns anchor for chaining:
//
//	btn := md3e.PlainTooltip(c, md3e.IconButton(c, mdicon.Share.Draw), "Share with others")
func PlainTooltip(c *ui.Context, anchor ui.Element, text string, opts ...PlainTooltipOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: PlainTooltip takes at most one PlainTooltipOptions")
	}
	var o PlainTooltipOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	cs := th.Colors
	bg := cs.InverseSurface
	ink := cs.InverseOnSurface
	if bg.A == 0 {
		if th.Dark {
			bg, ink = ui.RGB(0xE6, 0xE0, 0xE9), ui.RGB(0x31, 0x30, 0x36)
		} else {
			bg, ink = ui.RGB(0x31, 0x30, 0x36), ui.RGB(0xF5, 0xEF, 0xF7)
		}
	}

	desc := o.Description
	if desc == "" {
		desc = text
	}
	anchor.Description(desc)
	ui.TooltipBase(c, anchor, func(tip ui.Element) {
		tip.Padding(4, 8).Radius(4).MaxWidth(240).
			Background(bg).TextColor(ink)
		applyTypeStyleWithTheme(th, tip, tokens.LabelMedium)
		tip.Children(func() {
			ui.Text(c, text)
		})
	})
	return anchor
}

// RichTooltipAction is the optional action button of a RichTooltip.
type RichTooltipAction struct {
	// Label is the button's visible text and accessible name.
	Label string
	// OnClick runs when the action is chosen.
	OnClick func()
}

// RichTooltipOptions configures RichTooltip.
type RichTooltipOptions struct {
	// Title is the tooltip's heading; optional.
	Title string
	// Body is the tooltip's supporting text.
	Body string
	// Action adds an interactive button at the tooltip's end, whose reach the
	// pointer must keep, so RichTooltip uses a popover rather than the
	// pointer-through tooltip base.
	Action *RichTooltipAction
	// Label names the tooltip for assistive technology; the title when empty.
	Label string
	// Width fixes the panel's width in DIP; 320 when zero or invalid.
	Width float32
	// Delay is how long the pointer rests on the anchor before the tooltip
	// shows; 600ms when zero, and a negative delay shows it at once.
	Delay time.Duration
	// Description, when set, describes the anchor to assistive technology.
	Description string
}

// RichTooltip rhythm, in DIP.
const (
	richTooltipDefaultWidth = 320
	richTooltipLeaveGrace   = 120 * time.Millisecond
	richTooltipDefaultDelay = 600 * time.Millisecond
)

// richTooltipKey is the Local holding an anchor's rich tooltip state.
type richTooltipKey struct{}

// richTooltipState tracks the hover and focus that keep an interactive tooltip
// open, including a short grace while the pointer crosses to the panel.
type richTooltipState struct {
	open         bool
	suppress     bool
	hoverSince   time.Time
	leaveAt      time.Time
	panelHovered bool
	panelFocused bool
}

// RichTooltip shows a Material Design 3 rich tooltip beside anchor: a title, a
// body and an optional interactive action button, on a SurfaceContainer panel
// with a Level2 shadow. Unlike PlainTooltip it is a real popover, so the
// pointer can move onto the panel and press its action; the panel goes with
// Escape, a press outside, or the pointer leaving both the anchor and the panel.
// It shows after Delay on hover, at once on keyboard focus, keeps its local
// ThemeScope, and returns anchor:
//
//	btn := md3e.RichTooltip(c, md3e.Button(c, "Details"), md3e.RichTooltipOptions{
//		Title: "Details", Body: "Shows the full record.",
//		Action: &md3e.RichTooltipAction{Label: "Open", OnClick: app.open},
//	})
func RichTooltip(c *ui.Context, anchor ui.Element, opts ...RichTooltipOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: RichTooltip takes at most one RichTooltipOptions")
	}
	var o RichTooltipOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	delay := o.Delay
	if delay == 0 {
		delay = richTooltipDefaultDelay
	}
	if o.Description != "" {
		anchor.Description(o.Description)
	}

	st := ui.Local(anchor, richTooltipKey{}, func() richTooltipState { return richTooltipState{} })
	now := c.Now()
	hovering := anchor.Hovered()
	focused := anchor.Focused() || anchor.FocusWithin()

	// A dismissed tooltip stays dismissed until the pointer and the focus have
	// both left it, as Base UI's and MyGo's tooltips do.
	if !hovering && !focused && !st.panelHovered && !st.panelFocused {
		st.suppress = false
		st.hoverSince = time.Time{}
	}
	if !st.open && !st.suppress {
		switch {
		case focused:
			st.open = true
			st.hoverSince = time.Time{}
		case hovering:
			if delay < 0 {
				st.open = true
			} else if st.hoverSince.IsZero() {
				st.hoverSince = now
				c.After(delay)
			} else if now.Sub(st.hoverSince) >= delay {
				st.open = true
			}
		default:
			st.hoverSince = time.Time{}
		}
	}
	if st.open {
		if hovering || focused || st.panelHovered || st.panelFocused {
			st.leaveAt = time.Time{}
		} else if st.leaveAt.IsZero() {
			// Give the pointer a moment to cross the gap onto the panel.
			st.leaveAt = now
			c.After(richTooltipLeaveGrace)
		} else if now.Sub(st.leaveAt) >= richTooltipLeaveGrace {
			st.open = false
			st.leaveAt = time.Time{}
		}
	}

	if st.open {
		panel := ui.PopoverBase(c, anchor, &st.open, func(panel ui.Element) {
			buildRichTooltipPanel(c, th, panel, o, func() {
				if o.Action != nil && o.Action.OnClick != nil {
					o.Action.OnClick()
				}
				st.open = false
				st.suppress = true
			})
		})
		if panel.Valid() {
			panel.AttachTo(anchor, ui.AnchorTopStart, ui.AnchorBottomStart).
				Margin(0, 0, 8, 0).AttachEdgeMargin(8)
			st.panelHovered = panel.Hovered()
			st.panelFocused = panel.FocusWithin()
			if panel.Dismissed() {
				st.open = false
				st.suppress = true
			}
		} else {
			st.panelHovered = false
			st.panelFocused = false
		}
	} else {
		st.panelHovered = false
		st.panelFocused = false
	}
	return anchor
}

// buildRichTooltipPanel styles a rich tooltip's popover panel and builds its
// text and action.
func buildRichTooltipPanel(c *ui.Context, th *Theme, panel ui.Element, o RichTooltipOptions, onAction func()) {
	width := o.Width
	if !(width > 0) {
		width = richTooltipDefaultWidth
	}
	bg := th.Colors.SurfaceContainer
	if bg.A == 0 {
		bg = SurfaceColorForElevation(th, tokens.Level2)
	}
	panel.Width(width).Padding(16).Radius(th.Shapes.Radius(tokens.ShapeMedium)).
		Background(bg).TextColor(th.Colors.OnSurface).
		Role(ui.RoleGroup)
	label := o.Label
	if label == "" {
		label = o.Title
	}
	if label != "" {
		panel.Label(label)
	}
	ApplyElevationShadows(th, panel, tokens.Level2)
	panel.Children(func() {
		if o.Title != "" {
			applyTypeStyleWithTheme(th, ui.Text(c, o.Title), tokens.TitleSmall)
		}
		if o.Body != "" {
			body := ui.Text(c, o.Body)
			body.TextColor(th.Colors.OnSurfaceVariant)
			applyTypeStyleWithTheme(th, body, tokens.BodyMedium)
		}
		if o.Action == nil || o.Action.Label == "" {
			return
		}
		action := ui.Row(c).Justify(ui.End)
		action.Children(func() {
			btn := Button(c, o.Action.Label, ButtonOptions{Variant: ButtonText, Size: tokens.SizeXS})
			btn.Shrink(0)
			if btn.Clicked() {
				onAction()
			}
		})
	})
	// The popover's own panel returns a Box; the content above was built inside it.
}
