package md3e

import (
	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
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

// Locals and animation channels of a carousel.
type (
	carouselIndexKey  struct{}
	carouselLastKey   struct{}
	carouselScrollKey struct{}
	carouselSizeKey   struct{}
	carouselDragKey   struct{}
	carouselWidthKey  struct{ i int }
	carouselItemKey   struct{ i int }
)

// carouselSizeState remembers the viewport width and whether the first real
// layout has placed the slides.
type carouselSizeState struct {
	init  bool
	viewW float32
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
	// lastAlignX and lastAlignIndex remember the last alignment asked for, so
	// one that cannot move the strip (its end is too short for the slide) is
	// not asked for again frame after frame.
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

	height := o.Height
	if !(height > 0) {
		height = carouselDefaultHeight
	}
	gap := o.Gap
	if !(gap > 0) {
		gap = carouselDefaultGap
	}
	label := o.Label
	if label == "" {
		label = "Carousel"
	}

	root := ui.Column(c).FillWidth().Height(height).Clip().
		Role(ui.RoleGroup).Label(label)
	root.Disabled(o.Disabled)

	index := o.Index
	if index == nil {
		index = ui.Local(root, carouselIndexKey{}, func() int { return 0 })
	}
	if len(items) == 0 {
		return root
	}
	*index = max(0, min(*index, len(items)-1))

	size := ui.Local(root, carouselSizeKey{}, func() carouselSizeState { return carouselSizeState{} })
	drag := ui.Local(root, carouselDragKey{}, func() carouselDragState { return carouselDragState{} })
	// -1 rather than the initial index: a carousel the caller starts on a
	// later slide lines that slide up as it first lays out.
	lastIndex := ui.Local(root, carouselLastKey{}, func() int { return -1 })
	scrollState := ui.Local(root, carouselScrollKey{}, func() ui.ScrollState { return ui.ScrollState{} })

	viewW := root.LayoutBounds().W
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
	jump := false
	if viewW > 0 && !size.init {
		size.init = true
		jump = true
	}

	radius := th.Shapes.Radius(tokens.ShapeLarge)
	base := SurfaceColorForElevation(th, tokens.Level1)
	ink := cs.OnSurface
	if o.Disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}

	itemEls := make([]ui.Element, len(items))
	pressed := -1
	var viewport ui.Element

	root.Children(func() {
		// The carousel snaps itself: a rest in the scroll position hands the
		// nearest slide the focus, which is then lined up with the strip's
		// start, so the spring-animated widths and the snap place never argue.
		viewport = ui.ScrollHorizontal(c).
			TrackScroll(scrollState).
			Scrollbars(false).
			FillWidth().
			FillHeight()
		viewport.Disabled(o.Disabled)

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

					c.Key(carouselItemKey{i: i})
					el := ui.Box(c).
						Focusable().
						Shrink(0).
						Height(height).
						ClipInset(0, 0, 0, 0, radius).
						Label(item.Label).
						Role(ui.RoleButton).
						AccessSelected(i == *index).
						Background(base).
						TextColor(ink)
					itemEls[i] = el
					idx := i
					el.HandleInput(func(ev ui.InputEvent) bool {
						if o.Disabled {
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

					widthTarget := target
					var width float32
					if jump {
						width = el.Jump(carouselWidthKey{i: i}, widthTarget)
					} else {
						width = animateWithTheme(el, th, carouselWidthKey{i: i}, widthTarget, motion.DefaultSpatial)
					}
					el.Width(width)

					el.Children(func() {
						if item.Content != nil {
							if content := item.Content(); content.Valid() {
								content.Fill()
								return
							}
						}
						caption := ui.Column(c).FillWidth().FillHeight().
							Justify(ui.Center).AlignItems(ui.Center).Padding(16)
						caption.Children(func() {
							txt := ui.Text(c, item.Label)
							applyTypeStyleWithTheme(th, txt, tokens.TitleMedium)
							txt.TextColor(ink)
						})
					})
					attachRippleWithTheme(th, el, StateLayerOptions{Color: ink, Radius: radius, Disabled: o.Disabled})
					if el.Clicked() && !o.Disabled {
						pressed = i
					}
				}
			})
		})
	})

	// Pan the strip from the pointer's place in the viewport, which does not
	// move with the content, and let go of the drag as a pan once it passes the
	// threshold.
	if drag.active && !o.Disabled && viewport.Valid() {
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
	if pressed >= 0 && !suppress {
		*index = pressed
		drag.aligning = true
		if o.Changed != nil {
			o.Changed(pressed)
		}
	}

	// Follow the caller's writes to Index, and the choice above, by lining the
	// chosen slide up with the strip's start; its snapping keeps it there as
	// the widths animate.
	alignNow := false
	if *index != *lastIndex {
		*lastIndex = *index
		drag.aligning, drag.synced = true, false
		alignNow = true
	}
	alignSlide := func(i int) {
		if absF32(scrollState.X-drag.lastAlignX) < 0.5 && drag.lastAlignIndex == i {
			return
		}
		drag.lastAlignX, drag.lastAlignIndex = scrollState.X, i
		opts := ui.ScrollIntoViewOptions{Inline: ui.ScrollStart}
		if !th.ReduceMotion {
			anim := th.Animation(motion.DefaultSpatial)
			opts.Animation = &anim
		}
		itemEls[i].ScrollIntoView(opts)
	}
	if alignNow && !o.Disabled {
		alignSlide(*index)
	}

	// Watch the scroll position: once it rests, bring the focused slide in step
	// with whichever slide the strip settled on (a wheel notch or a touchpad
	// scroll), and line a dragged strip up on its nearest slide. While a
	// choice is being aligned this wait stays out of the way, so widths
	// reallocating cannot bounce the focus back.
	if viewport.Valid() {
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
				atEnd := scrollState.MaxX > 0 && scrollState.X >= scrollState.MaxX-1
				nearest, startDist := carouselNearest(itemEls, viewport.LayoutBounds(), *index, rtl, atEnd)
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
// start, or the last slide at the strip's end where it cannot line up any
// further, and how far the focused slide's start is from the viewport's: a
// settled strip has that distance at zero.
func carouselNearest(items []ui.Element, viewport ui.Rect, focused int, rtl, atEnd bool) (int, float32) {
	if atEnd {
		focused = len(items) - 1
	}
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
	if atEnd {
		best = len(items) - 1
	}
	return best, startDist
}
