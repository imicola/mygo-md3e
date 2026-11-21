package md3e

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
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
	if desc != "" {
		anchor.Description(desc)
	}
	if strings.TrimSpace(text) == "" {
		return anchor
	}
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
	anchorHover  bool
	anchorFocus  bool
	panelHovered bool
	panelFocused bool
	hoverSince   time.Time
	leaveAt      time.Time
	delay        time.Duration
	srv          ui.Services
}

func (s *richTooltipState) stepOpenHover(p *ui.Painter, now time.Time) {
	if s.open {
		return
	}
	if !s.anchorHover && !s.anchorFocus && !s.panelHovered && !s.panelFocused {
		s.suppress = false
		s.hoverSince = time.Time{}
		return
	}
	if s.suppress || !s.anchorHover {
		s.hoverSince = time.Time{}
		return
	}
	if s.delay < 0 {
		s.open = true
		s.hoverSince = time.Time{}
		s.leaveAt = time.Time{}
		s.srv.Invalidate()
		return
	}
	if s.hoverSince.IsZero() {
		s.hoverSince = now
		p.After(s.delay)
		return
	}
	if elapsed := now.Sub(s.hoverSince); elapsed >= s.delay {
		s.open = true
		s.hoverSince = time.Time{}
		s.leaveAt = time.Time{}
		s.srv.Invalidate()
		return
	} else {
		p.After(s.delay - elapsed)
	}
}

func (s *richTooltipState) stepCloseHover(p *ui.Painter, now time.Time) {
	if !s.open {
		return
	}
	if s.anchorHover || s.anchorFocus || s.panelHovered || s.panelFocused {
		s.leaveAt = time.Time{}
		return
	}
	if s.leaveAt.IsZero() {
		s.leaveAt = now
		p.After(richTooltipLeaveGrace)
		return
	}
	elapsed := now.Sub(s.leaveAt)
	if elapsed >= richTooltipLeaveGrace {
		s.open = false
		s.leaveAt = time.Time{}
		s.srv.Invalidate()
		return
	}
	p.After(richTooltipLeaveGrace - elapsed)
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
	st.srv = c.Services()
	st.delay = delay
	st.anchorFocus = anchor.FocusVisible()

	// Check focus opening synchronously during build so keyboard Tab brings
	// the tooltip up immediately on the same frame without waiting a paint pass.
	if !st.open {
		st.panelHovered = false
		st.panelFocused = false
		if !st.anchorHover && !st.anchorFocus {
			st.suppress = false
			st.hoverSince = time.Time{}
		} else if !st.suppress && st.anchorFocus {
			st.open = true
			st.hoverSince = time.Time{}
			st.leaveAt = time.Time{}
		}
	}

	rs := ui.Local(anchor, rippleLocalKey{}, func() rippleState { return rippleState{} })
	if rs.hasDrawOver {
		rs.hoverHook = func(p *ui.Painter, now time.Time, hovered bool) {
			st.anchorHover = hovered
			st.stepOpenHover(p, now)
		}
	} else {
		anchor.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if !anchor.Valid() || r.W <= 0 || r.H <= 0 {
				return
			}
			st.anchorHover = !anchor.IsDisabled() && anchor.Hovered()
			st.stepOpenHover(p, p.Now())
		})
	}

	if st.open {
		panel := ui.PopoverBase(c, anchor, &st.open, func(panel ui.Element) {
			buildRichTooltipPanel(c, th, panel, o, func() {
				if o.Action != nil && o.Action.OnClick != nil {
					o.Action.OnClick()
				}
				st.open = false
				st.suppress = true
				st.hoverSince = time.Time{}
				st.leaveAt = time.Time{}
			})
		})
		if panel.Valid() {
			panel.AttachTo(anchor, ui.AnchorTopStart, ui.AnchorBottomStart).
				Margin(0, 0, 8, 0).AttachEdgeMargin(8)
			st.panelFocused = panel.FocusWithin()
			panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if !panel.Valid() || r.W <= 0 || r.H <= 0 {
					return
				}
				st.panelHovered = panel.Hovered()
				st.stepCloseHover(p, p.Now())
			})
			if panel.Dismissed() {
				st.open = false
				st.suppress = true
				st.panelHovered = false
				st.panelFocused = false
				st.hoverSince = time.Time{}
				st.leaveAt = time.Time{}
			} else if st.anchorHover || st.anchorFocus || st.panelHovered || st.panelFocused {
				st.leaveAt = time.Time{}
			}
		} else {
			st.panelHovered = false
			st.panelFocused = false
			st.leaveAt = time.Time{}
		}
	} else {
		st.panelHovered = false
		st.panelFocused = false
		st.leaveAt = time.Time{}
	}
	return anchor
}

// buildRichTooltipPanel styles a rich tooltip's popover panel and builds its
// text and action.
func buildRichTooltipPanel(c *ui.Context, th *Theme, panel ui.Element, o RichTooltipOptions, onAction func()) {
	width := sanitizePositiveF32(o.Width, richTooltipDefaultWidth)
	bg := th.Colors.SurfaceContainer
	if bg.A == 0 {
		bg = SurfaceColorForElevation(th, tokens.Level2)
	}
	panel.Width(width).MaxWidthPercent(92).MaxHeightPercent(85).
		Padding(16).Gap(6).Radius(th.Shapes.Radius(tokens.ShapeMedium)).
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
			title := ui.Text(c, o.Title)
			title.Shrink(0)
			applyTypeStyleWithTheme(th, title, tokens.TitleSmall)
		}
		if o.Body != "" {
			bodyScroll := ui.Scroll(c).Shrink(1).MinHeight(0).Scrollbars(false)
			bodyScroll.Children(func() {
				body := ui.Text(c, o.Body)
				body.Width(0).MinWidthPercent(100).TextColor(th.Colors.OnSurfaceVariant)
				applyTypeStyleWithTheme(th, body, tokens.BodyMedium)
			})
		}
		if o.Action == nil || o.Action.Label == "" {
			return
		}
		action := ui.Row(c).Justify(ui.End).Shrink(0)
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
