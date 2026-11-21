package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// BottomSheetOptions configures BottomSheet. The zero value is an inline
// standard sheet 320 DIP high that keeps no detent of its own.
type BottomSheetOptions struct {
	// Modal shows the sheet over a fading scrim in a ui.Overlay, taking the
	// focus, closing with Escape or a scrim press and giving the focus back;
	// without it the sheet is laid out inline at the bottom of its parent.
	Modal bool
	// Detents are the modal sheet's resting heights as fractions of the
	// window height, smallest first; {0.5, 1} when empty. A drag settles on
	// the nearest, and one pulled further than the dismiss threshold closes
	// the sheet.
	Detents []float32
	// Detent points to the resting detent; nil keeps it in the sheet's local
	// state. Dragging writes the nearest detent into it.
	Detent *int
	// Height is the standard sheet's height in DIP; 320 when zero or negative.
	Height float32
	// Content builds the sheet's scrolling body.
	Content func() ui.Element
	// Label names the sheet for assistive technology; "Bottom sheet" when
	// empty.
	Label string
	// Changed runs after a drag settles on another detent.
	Changed func(i int)
}

// Bottom sheet metrics and timing, in DIP.
const (
	bottomSheetDefaultHeight = 320
	bottomSheetHandleHeight  = 32
	bottomSheetHandleWidth   = 32
	bottomSheetHandleTall    = 4
	bottomSheetDismissMin    = 96
	bottomSheetDismissShare  = 0.25
)

// Locals and channels of a bottom sheet.
type (
	bottomSheetStateKey      struct{}
	bottomSheetShowKey       struct{}
	bottomSheetHeightKey     struct{}
	bottomSheetDetentKey     struct{}
	bottomSheetDragKey       struct{}
	bottomSheetDragOffsetKey struct{}
)

// bottomSheetState remembers a modal sheet through its exit animation.
type bottomSheetState struct {
	shown      bool
	appeared   bool
	exitHeight float32
	exitOffset float32
}

// bottomSheetDragState follows a pull of the sheet's handle in the scrim's
// stable axes, so the sheet moving with the pointer never cancels the pull.
type bottomSheetDragState struct {
	active    bool
	needStart bool
	release   bool
	startY    float32
	startH    float32
	offset    float32
}

// BottomSheet builds a Material Design 3 Expressive bottom sheet. A modal one
// sits in a ui.Overlay over a scrim that fades with the sheet, takes the focus,
// closes with Escape, a scrim press or a downward pull of its handle past the
// threshold, and gives the focus back when the exit animation ends. The handle
// springs the sheet to its nearest detent on release, and the body scrolls its
// own content without the pull ever fighting it. A standard sheet lays out
// inline at its parent's bottom.
//
//	md3e.BottomSheet(c, &app.share, md3e.BottomSheetOptions{
//		Modal: true, Detents: []float32{0.5, 1}, Detent: &app.shareDetent,
//		Label: "Share", Content: app.shareOptions,
//	})
func BottomSheet(c *ui.Context, open *bool, opts ...BottomSheetOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: BottomSheet takes at most one BottomSheetOptions")
	}
	var o BottomSheetOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if open == nil {
		return ui.Box(c).Size(0, 0)
	}
	if inExitingOverlay(c) && *open {
		*open = false
	}
	th := CurrentTheme(c)
	label := o.Label
	if label == "" {
		label = "Bottom sheet"
	}
	if !o.Modal {
		if !*open {
			return ui.Box(c).Size(0, 0)
		}
		height := sanitizePositiveF32(o.Height, bottomSheetDefaultHeight)
		return bottomSheetPanel(c, th, height, label, o, false, nil)
	}

	anchor := ui.Box(c).Size(0, 0)
	st := ui.Local(anchor, bottomSheetStateKey{}, func() bottomSheetState { return bottomSheetState{} })
	drag := ui.Local(anchor, bottomSheetDragKey{}, func() bottomSheetDragState { return bottomSheetDragState{} })
	detent := o.Detent
	if detent == nil {
		detent = ui.Local(anchor, bottomSheetDetentKey{}, func() int { return 0 })
	}
	detents := normalizeDetents(o.Detents)
	*detent = max(0, min(*detent, len(detents)-1))

	_, windowH := c.Size()
	if !(windowH > 0) {
		windowH = 600
	}
	minH := windowH * detents[0]
	maxH := windowH * detents[len(detents)-1]
	baseH := windowH * detents[*detent]
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	if !*open && !st.shown {
		*st = bottomSheetState{}
		*drag = bottomSheetDragState{}
		return anchor
	}
	if *open {
		if !st.shown {
			st.exitHeight, st.exitOffset = 0, 0
			*drag = bottomSheetDragState{}
		}
		st.shown = true
		st.exitHeight, st.exitOffset = 0, 0
	}
	exiting := !*open
	var show float32
	switch {
	case *open && !st.appeared:
		st.appeared = true
		if reduce {
			show = anchor.Jump(bottomSheetShowKey{}, 1)
		} else {
			anchor.Jump(bottomSheetShowKey{}, 0)
			show = 0
			c.Invalidate()
		}
	case exiting && reduce:
		st.shown, st.appeared = false, false
		st.exitHeight, st.exitOffset = 0, 0
		*drag = bottomSheetDragState{}
		c.Invalidate()
		return anchor
	case exiting:
		show = animateValue(anchor, bottomSheetShowKey{}, 0, th.Animation(motion.FastSpatial), false)
		if anchor.Arrived(bottomSheetShowKey{}) || (anchor.Settled(bottomSheetShowKey{}) && show <= 0.001) {
			st.shown, st.appeared = false, false
			st.exitHeight, st.exitOffset = 0, 0
			*drag = bottomSheetDragState{}
			c.Invalidate()
			return anchor
		}
	default:
		show = animateValue(anchor, bottomSheetShowKey{}, 1, th.Animation(motion.FastSpatial), false)
	}

	scrim := th.Colors.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}
	handleInput := func(ev ui.InputEvent) bool {
		if exiting {
			return false
		}
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			drag.active, drag.needStart, drag.release = true, true, false
			drag.offset = 0
			drag.startH = baseH
			return true
		case ui.InputPointerMove:
			return drag.active
		case ui.InputPointerUp:
			if ev.Button != 0 {
				return false
			}
			if ev.Canceled {
				drag.active, drag.needStart, drag.release = false, false, false
				drag.offset, drag.startH = 0, 0
				return false
			}
			if drag.active {
				drag.release = true
			}
			drag.active = false
			return true
		}
		return false
	}

	var back ui.Element
	ui.Overlay(c, func() {
		back = ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			AlignItems(ui.End).
			Background(scrim.Alpha(0.32 * clampF32(show, 0, 1)))
		if !exiting {
			back.Modal()
		}
		back.IgnoreInput(exiting).AccessHidden(exiting)

		// Read the pointer in the scrim's stable axes before building the
		// panel, so the sheet moving under the pointer never feeds back.
		if !exiting && (drag.active || drag.release) {
			if drag.startH <= 0 {
				drag.startH = baseH
			}
			if _, y, _ := back.PointerPosition(); drag.needStart {
				drag.startY, drag.offset = y, 0
				drag.needStart = false
			} else {
				drag.offset = y - drag.startY
			}
		} else if exiting {
			drag.active, drag.needStart, drag.release = false, false, false
			drag.offset, drag.startH = 0, 0
		}

		startH := baseH
		if drag.startH > 0 {
			startH = drag.startH
		}
		dismissTriggered := false
		if !exiting && drag.release {
			drag.release = false
			dy := drag.offset
			drag.offset = 0
			drag.startH = 0
			rawH := startH - dy
			dismissAt := max(float32(bottomSheetDismissMin), minH*bottomSheetDismissShare)
			if minH-rawH >= dismissAt {
				st.exitHeight = minH
				st.exitOffset = max(float32(0), minH-rawH)
				*open = false
				dismissTriggered = true
				c.Invalidate()
			} else {
				best := *detent
				bestDist := absF32(rawH - startH)
				for i := range detents {
					if d := absF32(rawH - windowH*detents[i]); d < bestDist {
						best, bestDist = i, d
					}
				}
				if best != *detent {
					*detent = best
					baseH = windowH * detents[best]
					if o.Changed != nil {
						o.Changed(best)
					}
				}
			}
		}

		withExitingOverlay(c, exiting, func() {
			back.Children(func() {
				rawH := startH - drag.offset
				panelBaseH := baseH
				if exiting && st.exitHeight > 0 {
					panelBaseH = st.exitHeight
				} else if dismissTriggered {
					panelBaseH = st.exitHeight
				}
				panel := bottomSheetPanel(c, th, panelBaseH, label, o, true, handleInput)
				panel.Opacity(clampF32(show, 0, 1))

				var height, slideY float32
				switch {
				case exiting || dismissTriggered:
					height = panel.Jump(bottomSheetHeightKey{}, panelBaseH)
					slideY = st.exitOffset + (1-clampF32(show, 0, 1))*max(height-st.exitOffset, 0)
				case drag.active:
					targetH := clampF32(rawH, minH, maxH)
					height = panel.Jump(bottomSheetHeightKey{}, targetH)
					dragTranslate := panel.Jump(bottomSheetDragOffsetKey{}, max(float32(0), minH-rawH))
					slideY = (1-clampF32(show, 0, 1))*height + dragTranslate
				default:
					height = animateValue(panel, bottomSheetHeightKey{}, baseH, th.Animation(motion.DefaultSpatial), reduce)
					dragTranslate := animateValue(panel, bottomSheetDragOffsetKey{}, 0, th.Animation(motion.FastSpatial), reduce)
					slideY = (1-clampF32(show, 0, 1))*height + dragTranslate
				}
				panel.Height(height)
				panel.Translate(0, slideY)
				if !exiting && !dismissTriggered {
					// The panel's own surface takes presses, so the scrim below never
					// sees them.
					panel.Clicked()
					if back.Clicked() {
						*open = false
					}
					if back.OverlayShortcut(0, ui.KeyEscape) {
						*open = false
					}
				}
			})
		})
	})
	return anchor
}

// bottomSheetPanel builds the sheet's handle and scrolling body into one panel
// column, rounding its top corners and clipping its content. A modal panel is
// a dialog whose handle takes the pull through handle; a standard panel is a
// region laid out inline and takes none.
func bottomSheetPanel(c *ui.Context, th *Theme, height float32, label string, o BottomSheetOptions, modal bool, handle func(ui.InputEvent) bool) ui.Element {
	cs := th.Colors
	radius := th.Shapes.Radius(tokens.ShapeExtraLarge)
	panel := ui.Column(c).
		FillWidth().
		Height(height).
		Clip().
		Radius(radius, radius, 0, 0).
		Background(SurfaceColorForElevation(th, tokens.Level2)).
		TextColor(cs.OnSurface)
	if modal {
		panel.Role(ui.RoleDialog).Label(label)
		ApplyElevationShadows(th, panel, tokens.Level3)
	} else {
		panel.Role(ui.RoleRegion).Label(label)
	}
	panel.Children(func() {
		handleRow := ui.Row(c).FillWidth().Height(bottomSheetHandleHeight).
			Justify(ui.Center).AlignItems(ui.Center)
		if handle != nil {
			handleRow.HandleInput(handle)
		}
		handleRow.Children(func() {
			ui.Box(c).Size(bottomSheetHandleWidth, bottomSheetHandleTall).
				Radius(bottomSheetHandleTall / 2).
				Background(cs.OnSurfaceVariant.Alpha(0.4))
		})
		body := ui.Scroll(c).Grow(1).FillWidth().Scrollbars(false)
		body.Children(func() {
			content := ui.Column(c).FillWidth().Padding(24)
			content.Children(func() {
				if o.Content != nil {
					if built := o.Content(); built.Valid() {
						built.FillWidth()
					}
				}
			})
		})
	})
	return panel
}

// normalizeDetents returns the sheet's detents clamped to (0, 1] and sorted
// ascending, or the {0.5, 1} default.
func normalizeDetents(detents []float32) []float32 {
	out := make([]float32, 0, len(detents))
	for _, d := range detents {
		if d > 0 && d <= 1 {
			out = append(out, d)
		}
	}
	if len(out) == 0 {
		return []float32{0.5, 1}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
