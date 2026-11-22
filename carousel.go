package md3e

import (
	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// CarouselLayout selects how a Carousel sizes the items around its focused
// one.
type CarouselLayout int

const (
	// CarouselHero shows the focused item large with its neighbours peeking at
	// the sides, which grow as they come into focus.
	CarouselHero CarouselLayout = iota
	// CarouselMultiBrowse shows the focused item large, the next one medium and
	// the rest small, so several items stay browsable at once.
	CarouselMultiBrowse
	// CarouselUncontained shows every item at the same width, without sizing
	// the focused one up.
	CarouselUncontained
)

// CarouselItem is one slide of a Carousel.
type CarouselItem struct {
	// Label is the accessible name of the slide and the fallback caption.
	Label string
	// Content builds what the slide shows; it is clipped to the slide's
	// rounded box. A nil builder paints the tonal surface and the label.
	Content func() ui.Element
}

// CarouselOptions configures a Carousel. The zero value is an enabled Hero
// carousel that keeps its own focused item.
type CarouselOptions struct {
	// Index points to the focused item; nil keeps it in the carousel's local
	// state. Clicks, dragging and the caller's writes move the focus.
	Index *int
	// Layout selects the Hero, MultiBrowse or Uncontained sizing.
	Layout CarouselLayout
	// Height is the item height in DIP; 220 when zero or negative.
	Height float32
	// Gap is the space between items in DIP; 8 when zero or negative.
	Gap float32
	// Label names the carousel for assistive technology; "Carousel" when
	// empty.
	Label string
	// Changed runs after a click, drag or the arrows choose another item.
	Changed func(i int)
	// Disabled disables the slides and their input.
	Disabled bool
}

// Carousel metrics, in DIP.
const (
	carouselDefaultHeight = 220
	carouselDefaultGap    = 8
	carouselMinItemWidth  = 120
	carouselDragThreshold = 8
	carouselAlignSlack    = 1.5
)

// carouselSizeState remembers the viewport width and whether the first real
// layout has placed the slides.
type carouselSizeState struct {
	init   bool
	probed bool
	viewW  float32
}

// carouselDragState is the pointer panning of the strip, and the bookkeeping
// that keeps the focused item in step with the scroll position without
// fighting the user's drag.
type carouselDragState struct {
	active     bool
	dragged    bool
	suppress   bool
	needStart  bool
	startX     float32
	startPan   float32
	lastScroll float32
	stable     int
	synced     bool
	aligning   bool
	lastCount  int
	// alignInit, lastAlignX, and lastAlignIndex remember the last alignment
	// asked for, so one that cannot move the strip (its end is too short for
	// the slide) is not asked for again frame after frame.
	alignInit      bool
	lastAlignX     float32
	lastAlignIndex int
}

// Carousel builds a Material Design 3 Expressive carousel: a horizontal strip
// of ui.RoleButton slides whose widths follow their distance from the focused
// one and the chosen layout, clip their content to rounded corners with
// ClipInset, and glide to their new sizes with the Default Spatial spring. A
// click, Enter or Space focuses a slide, a pointer drag pans the strip and
// snaps it to the nearest slide, the wheel pages through it, and Index may be
// set by the app. The focused slide stays aligned with the strip's start, also
// right to left, and reduced motion settles every width at once:
//
//	md3e.Carousel(c, []md3e.CarouselItem{
//		{Label: "Mountain", Content: func() ui.Element { return photo(c, "mountain") }},
//		{Label: "Lake", Content: func() ui.Element { return photo(c, "lake") }},
//	}, md3e.CarouselOptions{Index: &app.slide, Layout: md3e.CarouselHero})
func Carousel(c *ui.Context, items []CarouselItem, opts ...CarouselOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Carousel takes at most one CarouselOptions")
	}
	var o CarouselOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	th := CurrentTheme(c)
	cs := th.Colors
	rtl := c.Direction() == ui.RightToLeft
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	height := sanitizePositiveF32(o.Height, carouselDefaultHeight)
	gap := sanitizePositiveF32(o.Gap, carouselDefaultGap)
	label := o.Label
	if label == "" {
		label = "Carousel"
	}

	root := ui.Column(c).FillWidth().Height(height).Clip().DimDisabled(false).
		Role(ui.RoleGroup).Label(label)
	root.Disabled(o.Disabled)
	disabled := o.Disabled || root.IsDisabled()

	index := o.Index
	if index == nil {
		index = localInt(root, animChan0)
	}
	size := localPtr(root, animChan1, func() *carouselSizeState { return &carouselSizeState{} })
	drag := localPtr(root, animChan2, func() *carouselDragState { return &carouselDragState{} })
	// -1 rather than the initial index: a carousel the caller starts on a
	// later slide lines that slide up as it first lays out.
	lastIndex := localNegInt(root, animChan3)
	scrollState := localScrollState(root, animChan4)

	if len(items) == 0 {
		*drag = carouselDragState{}
		*lastIndex = -1
		size.probed = false
		return root
	}
	if drag.lastCount != len(items) {
		drag.active, drag.dragged, drag.suppress = false, false, false
		drag.needStart, drag.aligning, drag.synced, drag.alignInit = false, false, false, false
		drag.stable = 0
		drag.lastCount = len(items)
		*lastIndex = -1
	}
	*index = max(0, min(*index, len(items)-1))

	if disabled {
		drag.active, drag.dragged, drag.suppress = false, false, false
	}

	laidW := root.LayoutBounds().W
	viewW := laidW
	if viewW <= 0 {
		if w, _ := c.Size(); w > 0 {
			viewW = w
		}
	}
	if viewW > 0 {
		size.viewW = viewW
	}
	if size.viewW <= 0 {
		size.viewW = 480
	}
	jump := !size.init
	if laidW > 0 {
		size.init = true
		size.probed = true
	} else {
		size.init = false
		if !size.probed {
			// Ask for one post-layout pass when first unlaid, then rely on
			// DrawOver below once a non-zero width actually paints so a
			// permanently 0-wide container does not spin Invalidate.
			size.probed = true
			c.Invalidate()
		}
	}
	srv := c.Services()
	root.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !root.Valid() || r.W <= 0 || r.H <= 0 {
			return
		}
		if !size.init || absF32(r.W-size.viewW) > 0.5 {
			srv.Invalidate()
		}
	})

	radius := th.Shapes.Radius(tokens.ShapeExtraLarge)
	base := SurfaceColorForElevation(th, tokens.Level1)
	hairline := resolveHairlineColor(th, ui.Color{})
	ink := cs.OnSurface
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	scrimBase := cs.Scrim
	if scrimBase.A == 0 {
		scrimBase = ui.RGB(0, 0, 0)
	}
	peekVeil := scrimBase.Alpha(0.05)
	if th.Dark {
		peekVeil = scrimBase.Alpha(0.12)
	}

	var stackEls [12]ui.Element
	var itemEls []ui.Element
	if len(items) <= len(stackEls) {
		itemEls = stackEls[:len(items)]
	} else {
		itemEls = make([]ui.Element, len(items))
	}
	pressed := -1
	var viewport ui.Element

	root.Children(func() {
		// The carousel snaps itself: a rest in the scroll position hands the
		// nearest slide the focus, which is then lined up with the strip's
		// start, so the spring-animated widths and the snap place never argue.
		viewport = ui.ScrollHorizontal(c).
			DimDisabled(false).
			TrackScroll(scrollState).
			Scrollbars(false).
			FillWidth().
			FillHeight()
		viewport.Disabled(disabled)

		viewport.Children(func() {
			row := ui.Row(c).Gap(gap).AlignItems(ui.Stretch).FocusGroup(ui.Horizontal)
			row.Children(func() {
				for i := range items {
					item := &items[i]
					dist := i - *index
					if dist < 0 {
						dist = -dist
					}
					target := carouselItemWidth(o.Layout, dist, size.viewW)

					c.Key(i)
					el := ui.Box(c).
						DimDisabled(false).
						Focusable().
						FocusRing(false).
						Disabled(disabled).
						Shrink(0).
						Height(height).
						Radius(radius).
						ClipInset(0, 0, 0, 0, radius).
						Label(item.Label).
						Role(ui.RoleButton).
						AccessSelected(i == *index).
						Background(base).
						TextColor(ink)
					if !disabled {
						if hairline.A > 0 {
							el.Border(1, hairline)
						}
						if drag.active {
							el.Cursor(ui.CursorGrabbing)
						} else {
							el.Cursor(ui.CursorGrab)
						}
					}
					itemEls[i] = el
					idx := i
					el.HandleInput(func(ev ui.InputEvent) bool {
						if disabled {
							return false
						}
						switch ev.Kind {
						case ui.InputPointerDown:
							if ev.Button != 0 {
								return false
							}
							// Take the press: the slide keeps the moves, and the
							// release still clicks it unless it pans.
							drag.active, drag.dragged, drag.suppress = true, false, false
							drag.needStart = true
							drag.aligning = false
							return true
						case ui.InputPointerMove:
							// A frame measures the pointer in the strip's stable
							// axes and pans there, so the slides moving under it
							// never feed back into the drag.
							return drag.active
						case ui.InputPointerUp:
							if ev.Button != 0 {
								return false
							}
							if ev.Canceled {
								drag.active, drag.dragged, drag.suppress = false, false, false
								return false
							}
							if drag.dragged {
								drag.suppress = true
							}
							drag.active, drag.dragged = false, false
							return true
						case ui.InputKeyDown:
							if ev.Repeat {
								return false
							}
							if ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace {
								*index = idx
								drag.aligning = true
								if o.Changed != nil {
									o.Changed(idx)
								}
								return true
							}
						}
						return false
					})

					if el.PressCanceled() {
						drag.active, drag.dragged = false, false
					}

					widthTarget := target
					width := animateValue(el, animChan0, widthTarget, th.Animation(motion.DefaultSpatial), reduce || jump)
					el.Width(width)

					el.Children(func() {
						if item.Content != nil {
							if content := item.Content(); content.Valid() {
								content.Fill()
							}
						} else {
							caption := ui.Column(c).FillWidth().FillHeight().
								Justify(ui.Center).AlignItems(ui.Center).Padding(16)
							caption.Children(func() {
								txt := ui.Text(c, item.Label)
								applyTypeStyleWithTheme(th, txt, tokens.TitleMedium)
								txt.TextColor(ink)
							})
						}
						if !disabled && i != *index && o.Layout != CarouselUncontained {
							ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
								Background(peekVeil).PassThrough()
						}
					})
					borderInset := float32(0)
					if !disabled && hairline.A > 0 {
						borderInset = 1
					}
					attachRippleWithContext(c, th, el, StateLayerOptions{
						Color:          ink,
						Radius:         radius,
						BorderInset:    borderInset,
						FocusHaloColor: cs.Surface,
						Disabled:       disabled,
					})
					if el.Clicked() && !disabled {
						pressed = i
					}
				}
			})
		})
	})

	// Pan the strip from the pointer's place in the viewport, which does not
	// move with the content, and let go of the drag as a pan once it passes the
	// threshold.
	if drag.active && !disabled && viewport.Valid() {
		if x, _, _ := viewport.PointerPosition(); drag.needStart {
			drag.startX, drag.startPan = x, scrollState.X
			drag.needStart = false
		} else {
			dx := x - drag.startX
			if !drag.dragged && absF32(dx) >= carouselDragThreshold {
				drag.dragged = true
				drag.aligning = false
			}
			if drag.dragged {
				sign := float32(-1)
				if rtl {
					sign = 1
				}
				pan := drag.startPan + dx*sign
				scrollState.X = clampF32(pan, 0, scrollState.MaxX)
			}
		}
	}
	suppress := drag.suppress
	if suppress {
		drag.suppress = false
	}
	if pressed >= 0 && !suppress && !disabled {
		*index = pressed
		drag.aligning = true
		if o.Changed != nil {
			o.Changed(pressed)
		}
	}

	// Follow the caller's writes to Index, and the choice above, by lining the
	// chosen slide up with the strip's start; its snapping keeps it there as
	// the widths animate.
	if laidW > 0 {
		alignNow := false
		if *index != *lastIndex {
			*lastIndex = *index
			drag.aligning, drag.synced = true, false
			alignNow = true
		}
		alignSlide := func(i int) {
			if drag.alignInit && absF32(scrollState.X-drag.lastAlignX) < 0.5 && drag.lastAlignIndex == i {
				return
			}
			drag.alignInit = true
			drag.lastAlignX, drag.lastAlignIndex = scrollState.X, i
			var targetX float32
			for k := 0; k < i; k++ {
				targetX += carouselItemWidth(o.Layout, i-k, size.viewW) + gap
			}
			clamped := clampF32(targetX, 0, scrollState.MaxX)
			if absF32(scrollState.X-clamped) < 0.5 {
				return
			}
			if reduce {
				scrollState.X = clamped
				return
			}
			scrollState.ScrollTo(targetX, 0, th.Animation(motion.DefaultSpatial))
		}
		if alignNow && !disabled {
			alignSlide(*index)
		}

		// Watch the scroll position: once it rests, bring the focused slide in step
		// with whichever slide the strip settled on (a wheel notch or a touchpad
		// scroll), and line a dragged strip up on its nearest slide. While a
		// choice is being aligned this wait stays out of the way, so widths
		// reallocating cannot bounce the focus back.
		if viewport.Valid() && !disabled {
			if scrollState.X != drag.lastScroll {
				drag.lastScroll = scrollState.X
				drag.stable, drag.synced = 0, false
				// Keep a frame coming so the next one can tell the strip stopped.
				c.Invalidate()
			}
			if drag.stable < 2 {
				drag.stable++
				if drag.stable < 2 {
					c.Invalidate()
				}
			}
			if drag.stable >= 2 && !drag.active && !drag.synced {
				drag.synced = true
				if drag.aligning {
					drag.aligning = false
				} else {
					nearest, startDist := carouselNearest(itemEls, viewport.LayoutBounds(), *index, rtl)
					if nearest != *index {
						*index = nearest
						*lastIndex = nearest
						drag.aligning = true
						if o.Changed != nil {
							o.Changed(nearest)
						}
						alignSlide(nearest)
					} else if startDist > carouselAlignSlack {
						alignSlide(*index)
					}
				}
			}
		}
	}
	return root
}

// carouselItemWidth returns the width of a slide dist steps from the focused
// one for the layout: Hero and MultiBrowse shrink with distance, Uncontained
// keeps every slide the same.
func carouselItemWidth(layout CarouselLayout, dist int, viewW float32) float32 {
	var share float32
	switch layout {
	case CarouselHero:
		switch dist {
		case 0:
			share = 0.84
		case 1:
			share = 0.30
		default:
			share = 0.18
		}
	case CarouselMultiBrowse:
		switch dist {
		case 0:
			share = 0.62
		case 1:
			share = 0.38
		case 2:
			share = 0.26
		default:
			share = 0.18
		}
	default:
		share = 0.45
	}
	w := viewW * share
	if w < carouselMinItemWidth {
		w = carouselMinItemWidth
	}
	if w > viewW {
		w = viewW
	}
	return w
}

// carouselNearest returns the slide whose start is closest to the viewport's
// start, and how far the focused slide's start is from the viewport's: a
// settled strip has that distance at zero.
func carouselNearest(items []ui.Element, viewport ui.Rect, focused int, rtl bool) (int, float32) {
	best, bestDist := focused, float32(1e9)
	var startDist float32
	for i := range items {
		box := items[i].LayoutBounds()
		if box.W <= 0 {
			continue
		}
		d := box.X - viewport.X
		if rtl {
			d = (viewport.X + viewport.W) - (box.X + box.W)
		}
		d = absF32(d)
		if d < bestDist {
			best, bestDist = i, d
		}
		if i == focused {
			startDist = d
		}
	}
	return best, startDist
}
