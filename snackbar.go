package md3e

import (
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// SnackbarOptions is one message of a SnackbarState queue.
type SnackbarOptions struct {
	// Message is the snackbar's text, which also names it for assistive
	// technology.
	Message string
	// ActionLabel adds a high-contrast text button at the end; OnAction runs
	// when it is chosen, and the snackbar closes.
	ActionLabel string
	// OnAction runs when the action button is pressed.
	OnAction func()
	// ShowClose adds a close button beside the message.
	ShowClose bool
	// DoubleLine puts the message above the actions instead of beside them,
	// for a longer message.
	DoubleLine bool
	// Duration is how long the snackbar stays before it goes by itself; 4
	// seconds when zero or negative. Hovering or focusing it pauses the
	// countdown.
	Duration time.Duration
}

// snackbar metrics and timing.
const (
	snackbarDefaultDuration = 4 * time.Second
	snackbarMaxWidth        = 344
	snackbarBottom          = 24
	snackbarSwipeThreshold  = 72
	snackbarEnterOffset     = 20
)

// snackbarItem is the snackbar showing now, with its countdown.
type snackbarItem struct {
	opts     SnackbarOptions
	seq      uint64
	deadline time.Time
	paused   bool
	pausedAt time.Time
	exiting  bool
}

// SnackbarState is a single-queue Material Design 3 snackbar controller. Show
// enqueues messages; the one showing stays until its action, its close button,
// its timeout, or a swipe to either side, then the next follows in order. The
// SnackbarHost that draws it reads this state, and its countdown follows the
// frame clock, which ui.NewTester.Advance drives.
type SnackbarState struct {
	queue   []SnackbarOptions
	current *snackbarItem
	seq     uint64
	version uint64
}

// Show enqueues a snackbar. With none showing it starts at once; otherwise it
// waits its turn behind the ones already queued:
//
//	app.snack.Show(md3e.SnackbarOptions{Message: "Note deleted", ActionLabel: "Undo", OnAction: app.undo})
func (s *SnackbarState) Show(o SnackbarOptions) {
	if s == nil {
		return
	}
	if o.Duration <= 0 {
		o.Duration = snackbarDefaultDuration
	}
	if s.current == nil {
		s.seq++
		s.current = &snackbarItem{opts: o, seq: s.seq}
	} else {
		s.queue = append(s.queue, o)
	}
	s.version++
}

// dismiss starts the showing snackbar's exit.
func (s *SnackbarState) dismiss() {
	if s == nil || s.current == nil || s.current.exiting {
		return
	}
	s.current.exiting = true
	s.version++
}

// finish drops the showing snackbar and promotes the next queued one.
func (s *SnackbarState) finish() {
	if s == nil {
		return
	}
	s.current = nil
	if len(s.queue) > 0 {
		next := s.queue[0]
		s.queue[0] = SnackbarOptions{}
		s.queue = s.queue[1:]
		if len(s.queue) == 0 {
			s.queue = nil
		}
		s.seq++
		s.current = &snackbarItem{opts: next, seq: s.seq}
	}
	s.version++
}

// snackbarHostKey is the Local holding a host's seen version and entry state.
type snackbarHostKey struct{}

// snackbarHostState tracks the currently displayed snackbar's entry state.
type snackbarHostState struct {
	seq      uint64
	appeared bool
}

// snackbarShowKey is the AnimateTo channel of a snackbar's show progress.
type snackbarShowKey struct{}

// snackbarSwipeKey is the Local holding a snackbar's horizontal drag offset.
type snackbarSwipeKey struct{}

// snackbarSwipeOffsetKey is the AnimateTo channel of a snackbar's horizontal
// offset, which springs back to zero when a short drag is let go.
type snackbarSwipeOffsetKey struct{}

// SnackbarHost draws state's snackbars in a ui.Overlay, inheriting the active
// ThemeScope, and returns a zero-size element that takes no room in its parent.
// Place it anywhere in the view; one state has one host:
//
//	app.snack = &md3e.SnackbarState{}
//	// ... in the view:
//	md3e.SnackbarHost(c, app.snack)
func SnackbarHost(c *ui.Context, state *SnackbarState) ui.Element {
	host := ui.Box(c).Size(0, 0)
	if state == nil {
		return host
	}
	hs := ui.Local(host, snackbarHostKey{}, func() snackbarHostState { return snackbarHostState{} })
	item := state.current
	if item == nil {
		hs.seq = 0
		hs.appeared = false
		return host
	}
	if hs.seq != item.seq {
		hs.seq = item.seq
		hs.appeared = false
	}

	th := CurrentTheme(c)
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	now := c.Now()
	if item.deadline.IsZero() {
		item.deadline = now.Add(item.opts.Duration)
	}

	ui.Overlay(c, func() {
		row := ui.Row(c).Absolute().Left(0).Right(0).Bottom(snackbarBottom).
			Justify(ui.Center).PaddingX(16).PassThrough()
		row.Children(func() {
			// Key each snackbar item by sequence number so a freshly promoted
			// item never inherits the previous item's show or swipe channels.
			c.Key(item.seq)
			// The overlay is inside the caller's ThemeScope, so this reads the
			// same theme as the content around it.
			sb := snackbarSurface(c, th, item.opts,
				func() {
					if item.exiting {
						return
					}
					if item.opts.OnAction != nil {
						item.opts.OnAction()
					}
					state.dismiss()
				},
				func() {
					if item.exiting {
						return
					}
					state.dismiss()
				},
			)
			sb.IgnoreInput(item.exiting).AccessHidden(item.exiting)

			// The countdown pauses while the pointer rests on the snackbar or
			// the focus is in it, and picks up the remaining time on leave.
			hovered := !item.exiting && (sb.Hovered() || sb.FocusWithin())
			if item.exiting {
				item.paused = false
			} else if hovered && !item.paused {
				item.paused = true
				item.pausedAt = now
			} else if !hovered && item.paused {
				item.deadline = item.deadline.Add(now.Sub(item.pausedAt))
				item.paused = false
			}
			if !item.exiting && !hovered && !now.Before(item.deadline) {
				item.exiting = true
			}

			// A horizontal drag past the threshold dismisses it; a shorter one
			// springs back as it is let go.
			swipe := ui.Local(sb, snackbarSwipeKey{}, func() float32 { return 0 })
			var offset float32
			if !item.exiting {
				if dx, _, ok := sb.Dragged(); ok {
					*swipe += dx
					// Track the pointer at once while dragging; the spring takes
					// over from the last offset once it is released.
					offset = sb.Jump(snackbarSwipeOffsetKey{}, *swipe)
				} else {
					offset = animateValue(sb, snackbarSwipeOffsetKey{}, 0, th.Animation(motion.FastSpatial), reduce)
					*swipe = 0
				}
				if absF32(*swipe) >= snackbarSwipeThreshold {
					state.dismiss()
				}
			} else {
				offset = animateValue(sb, snackbarSwipeOffsetKey{}, *swipe, th.Animation(motion.FastSpatial), reduce)
			}

			target := float32(1)
			if !hs.appeared && !reduce {
				target = 0
				hs.appeared = true
				c.Invalidate()
			}
			hs.appeared = true
			if item.exiting {
				target = 0
			}
			show := animateValue(sb, snackbarShowKey{}, target, th.Animation(motion.FastSpatial), reduce)
			if item.exiting && (reduce && show <= 0.001 || sb.Arrived(snackbarShowKey{}) || (sb.Settled(snackbarShowKey{}) && show <= 0.001)) {
				state.finish()
				c.Invalidate()
			}
			sb.Opacity(show).Translate(offset, (1-show)*snackbarEnterOffset)
		})
	})

	if !item.exiting && !item.paused {
		if d := item.deadline.Sub(now); d > 0 {
			c.After(d)
		}
	}
	return host
}

// snackbarSurface builds the inverse-surface snackbar panel with its message and
// optional actions.
func snackbarSurface(c *ui.Context, th *Theme, o SnackbarOptions, onAction, onClose func()) ui.Element {
	cs := th.Colors
	bg := cs.InverseSurface
	ink := cs.InverseOnSurface
	actionInk := cs.InversePrimary
	if bg.A == 0 {
		if th.Dark {
			bg, ink = ui.RGB(0xE6, 0xE0, 0xE9), ui.RGB(0x31, 0x30, 0x36)
		} else {
			bg, ink = ui.RGB(0x31, 0x30, 0x36), ui.RGB(0xF5, 0xEF, 0xF7)
		}
	}
	if ink.A == 0 {
		ink = th.ContentColorFor(bg)
	}
	if actionInk.A == 0 {
		actionInk = th.Colors.Primary
	}
	endPad := float32(8)
	if o.ActionLabel == "" && !o.ShowClose {
		endPad = 16
	}

	var sb ui.Element
	if o.DoubleLine {
		sb = ui.Column(c).MinHeight(48).Padding(12, 16).Gap(4).AlignItems(ui.Start)
	} else {
		sb = ui.Row(c).MinHeight(48).PaddingY(8).PaddingStart(16).
			PaddingEnd(endPad).Gap(8).AlignItems(ui.Center)
	}
	sb.WidthPercent(100).MaxWidth(snackbarMaxWidth).Radius(4).
		Background(bg).TextColor(ink).Role(ui.RoleStatus).Label(o.Message)
	ApplyElevationShadows(th, sb, tokens.Level3)

	sb.Children(func() {
		if o.DoubleLine {
			applyTypeStyleWithTheme(th, ui.Text(c, o.Message), tokens.BodyMedium)
			actions := ui.Row(c).FillWidth().Justify(ui.End).Gap(4)
			actions.Children(func() {
				snackbarActions(c, th, o, ink, actionInk, onAction, onClose)
			})
			return
		}
		msg := ui.Text(c, o.Message).Grow(1)
		applyTypeStyleWithTheme(th, msg, tokens.BodyMedium)
		snackbarActions(c, th, o, ink, actionInk, onAction, onClose)
	})
	return sb
}

// snackbarActions builds the snackbar's optional action and close buttons.
// ink is the message's resolved contrast color, which the close button shares
// so that a theme without the inverse roles still paints legible chrome.
func snackbarActions(c *ui.Context, th *Theme, o SnackbarOptions, ink, actionInk ui.Color, onAction, onClose func()) {
	if o.ActionLabel != "" {
		act := ui.ButtonBase(c).Height(32).PaddingX(12).Radius(16).
			FocusRing(false).DimDisabled(false).Label(o.ActionLabel)
		act.TextColor(actionInk)
		applyTypeStyleWithTheme(th, act, tokens.LabelLarge)
		attachRippleWithTheme(th, act, StateLayerOptions{Color: actionInk, Radius: 16})
		clicked := act.Clicked()
		act.Children(func() {
			ui.Text(c, o.ActionLabel).NoWrap()
		})
		if clicked {
			onAction()
		}
	}
	if o.ShowClose {
		close := ui.ButtonBase(c).Size(32, 32).Radius(16).
			FocusRing(false).DimDisabled(false).Label("Dismiss")
		close.TextColor(ink)
		attachRippleWithTheme(th, close, StateLayerOptions{Color: ink, Radius: 16})
		clicked := close.Clicked()
		close.Children(func() {
			drawIconBox(c, mdicon.Close.Draw, 18, ink)
		})
		if clicked {
			onClose()
		}
	}
}
