package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/motion"
	"mygo-md3e/tokens"
)

// DialogAction is one button of an AlertDialog.
type DialogAction struct {
	// Label is the visible text and the accessible name.
	Label string
	// OnClick runs when the button is pressed, before the dialog closes.
	OnClick func()
}

// AlertDialogOptions configures AlertDialog. The zero value is an enabled,
// 360 DIP wide alert that closes with Escape but not with a click on its scrim.
type AlertDialogOptions struct {
	// Icon draws an optional centered icon above the headline.
	Icon IconDraw
	// Headline is the dialog's title and accessible name.
	Headline string
	// SupportingText is the body below the headline.
	SupportingText string
	// Content builds an optional scrollable area between the text and the
	// actions, for a longer message or a form.
	Content func() ui.Element
	// Confirm is the high-emphasis action, drawn last.
	Confirm *DialogAction
	// Dismiss is the low-emphasis action, drawn before Confirm, and run when
	// Escape closes the dialog.
	Dismiss *DialogAction
	// DismissOnScrim closes the dialog when its scrim is pressed; a click on
	// the scrim does nothing unless it is set.
	DismissOnScrim bool
	// Label overrides the accessible name; the headline by default.
	Label string
	// Width is the panel width in DIP; 360 when zero or negative.
	Width float32
}

// Alert dialog metrics, in DIP.
const (
	alertDialogWidth        = 360
	alertDialogPadding      = 24
	alertDialogIconSize     = 24
	alertDialogContentLimit = 260
)

// Locals and channels of an alert or full-screen dialog.
type (
	alertDialogStateKey struct{}
	alertDialogShowKey  struct{}
)

// alertDialogState remembers a dialog through its exit animation, so the
// overlay can fade out before it goes and gives the focus back.
type alertDialogState struct {
	shown    bool
	appeared bool
}

// AlertDialog builds a Material Design 3 Expressive alert over the window while
// *open is true: an optional centered icon, a headline, supporting text, an
// optional scrollable middle area and Confirm and Dismiss buttons, on a large
// cornered SurfaceContainerHigh panel with a Level3 shadow. It scales and
// fades in with the Fast Spatial spring and fades out the same way, keeping its
// closing overlay until the animation ends so the focus returns to the element
// that opened it. Focus is locked inside and the window behind is inert; Escape
// runs Dismiss and closes, and a scrim press closes only with DismissOnScrim.
//
//	alert := false
//	md3e.AlertDialog(c, &alert, md3e.AlertDialogOptions{
//		Headline: "Delete “Notes”?", SupportingText: "You can't undo this.",
//		Dismiss: &md3e.DialogAction{Label: "Cancel"},
//		Confirm: &md3e.DialogAction{Label: "Delete", OnClick: app.delete},
//	})
func AlertDialog(c *ui.Context, open *bool, opts ...AlertDialogOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: AlertDialog takes at most one AlertDialogOptions")
	}
	var o AlertDialogOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	anchor := ui.Box(c).Size(0, 0)
	if open == nil {
		return anchor
	}
	th := CurrentTheme(c)
	cs := th.Colors
	st := ui.Local(anchor, alertDialogStateKey{}, func() alertDialogState { return alertDialogState{} })
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	if !*open && !st.shown {
		return anchor
	}
	if *open {
		st.shown = true
	}
	target := float32(1)
	if !*open {
		target = 0
	}
	var show float32
	switch {
	case *open && !st.appeared:
		st.appeared = true
		if reduce {
			show = anchor.Jump(alertDialogShowKey{}, 1)
		} else {
			anchor.Jump(alertDialogShowKey{}, 0)
			show = 0
			c.Invalidate()
		}
	case !*open && reduce:
		st.shown, st.appeared = false, false
		c.Invalidate()
		return anchor
	case !*open:
		show = animateValue(anchor, alertDialogShowKey{}, 0, th.Animation(motion.FastSpatial), false)
		if anchor.Arrived(alertDialogShowKey{}) {
			st.shown, st.appeared = false, false
			c.Invalidate()
			return anchor
		}
	default:
		show = animateValue(anchor, alertDialogShowKey{}, target, th.Animation(motion.FastSpatial), false)
	}

	scrim := cs.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}
	closeDialog := func() {
		*open = false
		if o.Dismiss != nil && o.Dismiss.OnClick != nil {
			o.Dismiss.OnClick()
		}
	}

	ui.Overlay(c, func() {
		back := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Justify(ui.Center).AlignItems(ui.Center).
			Background(scrim.Alpha(0.32 * clampF32(show, 0, 1))).
			Modal()
		back.Children(func() {
			panel := ui.Column(c).
				Width(alertDialogWidth).
				MaxWidthPercent(92).
				Padding(alertDialogPadding).
				Gap(16).
				Radius(th.Shapes.Radius(tokens.ShapeExtraLarge)).
				Background(SurfaceColorForElevation(th, tokens.Level3)).
				TextColor(cs.OnSurface).
				Role(ui.RoleAlertDialog).
				Label(labelOr(o.Label, o.Headline)).
				Opacity(clampF32(show, 0, 1)).
				Scale(0.9+0.1*clampF32(show, 0, 1), 0.9+0.1*clampF32(show, 0, 1))
			ApplyElevationShadows(th, panel, tokens.Level3)
			// A press on the panel's own surface must not reach the scrim.
			panel.Clicked()
			if o.Width > 0 {
				panel.Width(o.Width)
			}
			panel.Children(func() {
				if o.Icon != nil {
					ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
						drawIconBox(c, o.Icon, alertDialogIconSize, cs.Primary)
					})
				}
				if o.Headline != "" {
					title := ui.Text(c, o.Headline)
					title.TextColor(cs.OnSurface)
					applyTypeStyleWithTheme(th, title, tokens.HeadlineSmall)
				}
				if o.SupportingText != "" {
					body := ui.Text(c, o.SupportingText)
					body.TextColor(cs.OnSurfaceVariant)
					applyTypeStyleWithTheme(th, body, tokens.BodyMedium)
				}
				if o.Content != nil {
					scroll := ui.Scroll(c).MaxHeight(alertDialogContentLimit).Scrollbars(false)
					scroll.Children(func() {
						if content := o.Content(); content.Valid() {
							content.FillWidth()
						}
					})
				}
				if o.Dismiss != nil || o.Confirm != nil {
					actions := ui.Row(c).FillWidth().Justify(ui.End).Gap(8)
					actions.Children(func() {
						if o.Dismiss != nil && o.Dismiss.Label != "" {
							dismiss := Button(c, o.Dismiss.Label, ButtonOptions{Variant: ButtonText})
							if dismiss.Clicked() {
								closeDialog()
							}
						}
						if o.Confirm != nil && o.Confirm.Label != "" {
							confirm := Button(c, o.Confirm.Label, ButtonOptions{Variant: ButtonFilled})
							if confirm.Clicked() {
								if o.Confirm.OnClick != nil {
									o.Confirm.OnClick()
								}
								*open = false
							}
						}
					})
				}
			})
		})
		if back.Clicked() && o.DismissOnScrim {
			closeDialog()
		}
		if back.OverlayShortcut(0, ui.KeyEscape) {
			closeDialog()
		}
	})
	return anchor
}

// FullScreenDialogOptions configures FullScreenDialog.
type FullScreenDialogOptions struct {
	// Title is the top bar's title and the dialog's accessible name.
	Title string
	// Content builds the scrolling body.
	Content func() ui.Element
	// SaveLabel adds a text action at the top bar's end; none when empty.
	SaveLabel string
	// OnSave runs when the save action is chosen, before the dialog closes.
	OnSave func()
	// CloseLabel names the top bar's close button; "Close" when empty.
	CloseLabel string
	// OnClose runs when the dialog closes by its button, Escape or a backdrop
	// press.
	OnClose func()
	// Label overrides the accessible name; the title by default.
	Label string
}

// FullScreenDialog build metrics, in DIP.
const (
	fullScreenBarHeight = 64
)

// FullScreenDialog builds a Material Design 3 Expressive full-screen dialog: a
// ui.DialogBase whose panel fills the window, with a top bar holding a close
// button, the title and an optional save action, and a scrolling body below it.
// It keeps the keyboard inside, closes with Escape or its close button, and
// gives the focus back to the element that opened it.
//
//	md3e.FullScreenDialog(c, &app.editing, md3e.FullScreenDialogOptions{
//		Title: "New note", SaveLabel: "Save", OnSave: app.save, Content: app.editor,
//	})
func FullScreenDialog(c *ui.Context, open *bool, opts ...FullScreenDialogOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: FullScreenDialog takes at most one FullScreenDialogOptions")
	}
	var o FullScreenDialogOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	anchor := ui.Box(c).Size(0, 0)
	if open == nil || !*open {
		return anchor
	}
	th := CurrentTheme(c)
	cs := th.Colors
	closeLabel := o.CloseLabel
	if closeLabel == "" {
		closeLabel = "Close"
	}
	dismissed := false

	panel := ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(cs.Scrim.Alpha(0.32))
		panel.FillWidth().FillHeight().Column().
			Background(cs.Surface).TextColor(cs.OnSurface).
			Role(ui.RoleDialog).
			Label(labelOr(o.Label, o.Title))
		panel.Children(func() {
			bar := ui.Row(c).Height(fullScreenBarHeight).FillWidth().
				PaddingX(4).AlignItems(ui.Center).Gap(4)
			bar.Children(func() {
				close := IconButton(c, closeLabel, mdicon.Close.Draw, IconButtonOptions{Size: tokens.SizeS})
				if close.Clicked() {
					dismissed = true
					if o.OnClose != nil {
						o.OnClose()
					}
					*open = false
				}
				if o.Title != "" {
					title := ui.Text(c, o.Title).Grow(1)
					title.TextColor(cs.OnSurface)
					applyTypeStyleWithTheme(th, title, tokens.TitleLarge)
				} else {
					ui.Box(c).Grow(1)
				}
				if o.SaveLabel != "" {
					save := Button(c, o.SaveLabel, ButtonOptions{Variant: ButtonText})
					if save.Clicked() {
						if o.OnSave != nil {
							o.OnSave()
						}
						*open = false
					}
				}
			})
			Divider(c)
			scroll := ui.Scroll(c).Grow(1).Scrollbars(false)
			scroll.Children(func() {
				if o.Content != nil {
					if content := o.Content(); content.Valid() {
						content.FillWidth()
					}
				}
			})
		})
	})
	if panel.Valid() && panel.Dismissed() && !dismissed && o.OnClose != nil {
		o.OnClose()
	}
	return anchor
}
