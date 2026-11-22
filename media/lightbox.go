package media

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// LightboxItem describes one local or network media item shown in ImageLightbox.
type LightboxItem struct {
	// Image renders a pre-decoded local bitmap or SVG when non-nil.
	Image ui.ImageSource
	// URL loads a remote image via NetImage when Image is nil.
	URL string
	// MaxSide overrides LightboxOptions.MaxSide when positive.
	MaxSide int
	// Title is shown in the top bar and caption header.
	Title string
	// Caption is shown in the bottom caption bar when non-empty.
	Caption string
	// Label overrides the accessible description of the active image.
	Label string
}

// LightboxOptions configures ImageLightbox presentation, zoom range, and
// callbacks. The zero value allows 1x..5x zoom, scrim dismissal, and internal
// index management starting at slide 0.
type LightboxOptions struct {
	// Index optionally binds the active slide index externally; when nil,
	// ImageLightbox tracks the slide index internally.
	Index *int
	// MaxSide constrains remote image decoding when loading item URLs.
	MaxSide int
	// MinZoom sets the minimum scale factor (defaults to 1.0).
	MinZoom float32
	// MaxZoom sets the maximum scale factor (defaults to 5.0).
	MaxZoom float32
	// DismissOnScrim controls whether clicking the dark backdrop outside the
	// viewer stage and controls closes the lightbox (defaults to true).
	DismissOnScrim *bool
	// MatchKey optionally links the initially opened slide with a thumbnail
	// sharing the same MatchKey via ui.ElementTransition (Fly: true).
	MatchKey any
	// OnIndexChanged runs when keyboard or button navigation selects a new slide.
	OnIndexChanged func(index int)
	// OnClose runs when Escape, Close, or scrim dismissal closes the viewer.
	OnClose func()
	// Label sets the modal dialog's accessible name ("Image viewer" by default).
	Label string
}

type (
	lightboxStateKey    struct{}
	lightboxShowKey     struct{}
	lightboxZoomAnimKey struct{}
	lightboxSlideKey    int

	lightboxState struct {
		shown     bool
		appeared  bool
		localIdx  int
		openIdx   int
		lastIdx   int
		zoom      float32
		minZoom   float32
		maxZoom   float32
		panX      float32
		panY      float32
		viewportW float32
		viewportH float32
		dragging  bool
		dragX     float32
		dragY     float32
		panStartX float32
		panStartY float32
		lastTapAt time.Time
		lastTapX  float32
		lastTapY  float32
	}
)

// ImageLightbox shows a modal full-screen media viewer over the window while
// *open is true, supporting zoom (1x..5x via buttons, scroll wheel, pinch,
// double-click, and =/-/0 keys), constrained drag/touchpad panning, RTL-aware
// previous/next navigation, caption display, shared-element MatchKey entry, and
// Escape or scrim dismissal.
func ImageLightbox(c *ui.Context, open *bool, items []LightboxItem, opts ...LightboxOptions) ui.Element {
	if len(opts) > 1 {
		panic("media: ImageLightbox takes at most one LightboxOptions")
	}
	var o LightboxOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	anchor := ui.Box(c).Size(0, 0)
	if open == nil || len(items) == 0 {
		return anchor
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	st := ui.Local(anchor, lightboxStateKey{}, func() lightboxState {
		return lightboxState{zoom: 1, minZoom: 1, maxZoom: 5}
	})

	minZ := float32(1.0)
	if isPositiveFiniteF32(o.MinZoom) {
		minZ = o.MinZoom
	}
	maxZ := float32(5.0)
	if isPositiveFiniteF32(o.MaxZoom) && o.MaxZoom > minZ {
		maxZ = o.MaxZoom
	} else if maxZ <= minZ {
		maxZ = minZ + 4
	}
	st.minZoom = minZ
	st.maxZoom = maxZ

	idx := st.localIdx
	if o.Index != nil {
		idx = *o.Index
	}
	if idx < 0 {
		idx = 0
	} else if idx >= len(items) {
		idx = len(items) - 1
	}
	if o.Index != nil {
		*o.Index = idx
	} else {
		st.localIdx = idx
	}

	if !*open && !st.shown {
		return anchor
	}
	if *open && !st.shown {
		st.shown = true
		st.openIdx = idx
		st.lastIdx = idx
		st.zoom = minZ
		st.panX, st.panY = 0, 0
		st.dragging = false
		st.lastTapAt = time.Time{}
	} else if st.lastIdx != idx {
		st.lastIdx = idx
		st.zoom = minZ
		st.panX, st.panY = 0, 0
		st.dragging = false
		st.lastTapAt = time.Time{}
	}

	var show float32
	switch {
	case *open && !st.appeared:
		st.appeared = true
		if reduce {
			show = anchor.Jump(lightboxShowKey{}, 1)
		} else {
			anchor.Jump(lightboxShowKey{}, 0)
			show = 0
			c.Invalidate()
		}
	case !*open && reduce:
		st.shown, st.appeared = false, false
		st.zoom, st.panX, st.panY = minZ, 0, 0
		st.dragging = false
		c.Invalidate()
		return anchor
	case !*open:
		show = animateF32(anchor, lightboxShowKey{}, 0, th.Animation(motion.FastSpatial), false)
		if anchor.Arrived(lightboxShowKey{}) || (anchor.Settled(lightboxShowKey{}) && show <= 0.001) {
			st.shown, st.appeared = false, false
			st.zoom, st.panX, st.panY = minZ, 0, 0
			st.dragging = false
			c.Invalidate()
			return anchor
		}
	default:
		show = animateF32(anchor, lightboxShowKey{}, 1, th.Animation(motion.DefaultEffects), false)
	}

	st.zoom = clampF32(st.zoom, minZ, maxZ)
	lightboxClampPan(st)

	exiting := !*open
	dismissOnScrim := true
	if o.DismissOnScrim != nil {
		dismissOnScrim = *o.DismissOnScrim
	}
	closeLightbox := func() {
		if !*open {
			return
		}
		*open = false
		st.dragging = false
		if o.OnClose != nil {
			o.OnClose()
		}
	}
	setIndex := func(next int) {
		if next < 0 || next >= len(items) || next == idx {
			return
		}
		if o.Index != nil {
			*o.Index = next
		} else {
			st.localIdx = next
		}
		idx = next
		st.lastIdx = next
		st.zoom = minZ
		st.panX, st.panY = 0, 0
		st.dragging = false
		st.lastTapAt = time.Time{}
		if o.OnIndexChanged != nil {
			o.OnIndexChanged(next)
		}
	}

	rtl := c.Direction() == ui.RightToLeft
	scrim := cs.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}
	showAlpha := clampF32(show, 0, 1)

	hudTh := cinemaHUDTheme(anchor, th)
	hudCs := hudTh.Colors

	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(scrim.Alpha(0.90 * showAlpha))
		if !exiting {
			back.Modal()
		}
		back.IgnoreInput(exiting).AccessHidden(exiting)

		if !exiting {
			if back.OverlayShortcut(0, ui.KeyEscape) {
				closeLightbox()
			}
			if dismissOnScrim && back.Clicked() {
				closeLightbox()
			}
			if len(items) > 1 {
				switch {
				case back.OverlayShortcut(0, ui.KeyLeft):
					if rtl {
						setIndex(idx + 1)
					} else {
						setIndex(idx - 1)
					}
				case back.OverlayShortcut(0, ui.KeyRight):
					if rtl {
						setIndex(idx - 1)
					} else {
						setIndex(idx + 1)
					}
				}
			}
			switch {
			case back.OverlayShortcut(0, ui.KeyEqual), back.OverlayShortcut(ui.Shift, ui.KeyEqual):
				st.zoom = clampF32(st.zoom+0.5, minZ, maxZ)
				lightboxClampPan(st)
			case back.OverlayShortcut(0, ui.KeyMinus):
				st.zoom = clampF32(st.zoom-0.5, minZ, maxZ)
				lightboxClampPan(st)
			case back.OverlayShortcut(0, ui.Key0):
				st.zoom = minZ
				st.panX, st.panY = 0, 0
			}
		}

		back.Children(func() {
			md3e.ThemeScope(c, hudTh, func() {
				dialog := ui.Box(c).Fill().
					Role(ui.RoleDialog).
					Label(firstNonEmpty(o.Label, "Image viewer")).
					Opacity(showAlpha).
					Scale(0.95+0.05*showAlpha, 0.95+0.05*showAlpha)
				dialog.Children(func() {
					// Main zoomable/pannable stage inset slightly so scrim border remains reachable.
					stage := ui.Box(c).Absolute().Left(48).Top(64).Right(48).Bottom(72).
						Center().Clip()
					stage.Draw(func(_ *ui.Painter, r ui.Rect) {
						st.viewportW, st.viewportH = r.W, r.H
					})

					zoomInstant := reduce || st.dragging
					visZoom := animateF32(stage, lightboxZoomAnimKey{}, st.zoom, hudTh.Animation(motion.FastSpatial), zoomInstant)

					stage.HandleInput(func(ev ui.InputEvent) bool {
						if exiting {
							return false
						}
						switch ev.Kind {
						case ui.InputPointerDown:
							if ev.Button != 0 {
								return false
							}
							isDouble := ev.Clicks >= 2 || (!st.lastTapAt.IsZero() &&
								ev.Time.Sub(st.lastTapAt) <= 400*time.Millisecond &&
								math.Hypot(float64(ev.X-st.lastTapX), float64(ev.Y-st.lastTapY)) <= 8)
							st.lastTapAt = ev.Time
							st.lastTapX = ev.X
							st.lastTapY = ev.Y
							if isDouble {
								if st.zoom > st.minZoom+0.05 {
									st.zoom = st.minZoom
									st.panX, st.panY = 0, 0
								} else {
									target := min(st.maxZoom, max(st.minZoom, float32(2.0)))
									st.zoom = target
									lightboxClampPan(st)
								}
								st.dragging = false
								return true
							}
							if st.zoom > st.minZoom+0.001 {
								st.dragging = true
								st.dragX = ev.X
								st.dragY = ev.Y
								st.panStartX = st.panX
								st.panStartY = st.panY
							}
							return true
						case ui.InputPointerMove:
							if !st.dragging {
								return false
							}
							st.panX = st.panStartX + (ev.X - st.dragX)
							st.panY = st.panStartY + (ev.Y - st.dragY)
							lightboxClampPan(st)
							return true
						case ui.InputPointerUp:
							if !st.dragging {
								return false
							}
							if ev.Canceled {
								st.panX = st.panStartX
								st.panY = st.panStartY
								lightboxClampPan(st)
							}
							st.dragging = false
							return true
						case ui.InputZoom:
							if isPositiveFiniteF32(ev.Scale) {
								st.zoom = clampF32(st.zoom*ev.Scale, st.minZoom, st.maxZoom)
								lightboxClampPan(st)
							}
							return true
						case ui.InputScroll:
							if !ev.Precise || (ev.Mods&(ui.Cmd|ui.Ctrl)) != 0 {
								delta := ev.DY
								if delta == 0 {
									delta = ev.DX
								}
								if delta != 0 && isFiniteF32(delta) {
									steps := -float64(delta)
									if math.Abs(steps) >= 15 {
										steps /= 40.0
									} else if steps > 0 {
										steps = 1
									} else {
										steps = -1
									}
									factor := float32(clampF64(math.Pow(1.25, steps), 0.65, 1.55))
									st.zoom = clampF32(st.zoom*factor, st.minZoom, st.maxZoom)
									lightboxClampPan(st)
								}
								return true
							}
							if st.zoom > st.minZoom+0.001 {
								if isFiniteF32(ev.DX) {
									st.panX -= ev.DX
								}
								if isFiniteF32(ev.DY) {
									st.panY -= ev.DY
								}
								lightboxClampPan(st)
								return true
							}
						}
						return false
					})

					item := items[idx]
					var matchKey any
					if o.MatchKey != nil && idx == st.openIdx {
						matchKey = o.MatchKey
					}
					slideLabel := firstNonEmpty(item.Label, item.Title, fmt.Sprintf("Image %d of %d", idx+1, len(items)))

					stage.Children(func() {
						content := ui.Box(c).Fill().Center().
							Scale(visZoom, visZoom).
							Translate(st.panX, st.panY)
						content.Children(func() {
							if item.Image != nil {
								md3e.Image(c.Key(lightboxSlideKey(idx)), item.Image, md3e.ImageOptions{
									Fit:      ui.Contain,
									NoFade:   true,
									MatchKey: matchKey,
									Label:    slideLabel,
								}).Fill()
							} else {
								maxSide := o.MaxSide
								if item.MaxSide > 0 {
									maxSide = item.MaxSide
								}
								NetImage(c.Key(lightboxSlideKey(idx)), item.URL, NetImageOptions{
									MaxSide:  maxSide,
									Fit:      ui.Contain,
									MatchKey: matchKey,
									Label:    slideLabel,
								}).Fill()
							}
						})
					})

					// Previous / Next floating buttons at Start / End edges (automatically mirrored in RTL).
					if len(items) > 1 {
						prevIcon := symbols.ChevronLeft.Draw
						nextIcon := symbols.ChevronRight.Draw
						if rtl {
							prevIcon = symbols.ChevronRight.Draw
							nextIcon = symbols.ChevronLeft.Draw
						}
						ui.Column(c).Absolute().Start(16).Top(0).Bottom(0).Justify(ui.Center).Children(func() {
							if md3e.IconButton(c, "Previous image", prevIcon, md3e.IconButtonOptions{
								Variant:  md3e.IconButtonTonal,
								Size:     tokens.SizeM,
								Disabled: idx <= 0,
							}).Clicked() {
								setIndex(idx - 1)
							}
						})
						ui.Column(c).Absolute().End(16).Top(0).Bottom(0).Justify(ui.Center).Children(func() {
							if md3e.IconButton(c, "Next image", nextIcon, md3e.IconButtonOptions{
								Variant:  md3e.IconButtonTonal,
								Size:     tokens.SizeM,
								Disabled: idx >= len(items)-1,
							}).Clicked() {
								setIndex(idx + 1)
							}
						})
					}

					// Smoked dark-glass toolbar pills over cinema scrim.
					pillBg := ui.RGBA(22, 22, 28, 0.84)
					pillBorder := ui.RGBA(255, 255, 255, 0.14)
					pillInset := ui.RGBA(255, 255, 255, 0.08)

					ui.Row(c).Absolute().Top(12).Left(16).Right(16).
						Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(12).Children(func() {
						if len(items) > 1 || item.Title != "" {
							infoPill := md3e.Surface(c, md3e.SurfaceOptions{
								Elevation:    tokens.Level3,
								Color:        pillBg,
								ContentColor: hudCs.OnSurface,
								ShapeScale:   tokens.ShapeFull,
								BorderWidth:  1,
								BorderColor:  pillBorder,
							}).Shrink(1).MinWidth(0).PaddingX(14).PaddingY(8).InsetShadow(0, 1, 0, 0, pillInset)
							infoPill.Clicked()
							infoPill.Children(func() {
								ui.Row(c).AlignItems(ui.Center).Gap(10).MinWidth(0).Children(func() {
									if len(items) > 1 {
										md3e.TextWith(c, fmt.Sprintf("%d / %d", idx+1, len(items)), md3e.TextOptions{
											Role:        tokens.LabelLarge,
											Emphasized:  true,
											TabularNums: true,
											Color:       hudCs.OnSurface,
										}).NoWrap()
									}
									if item.Title != "" {
										md3e.TextWith(c, item.Title, md3e.TextOptions{
											Role:     tokens.TitleSmall,
											MaxLines: 1,
											Color:    hudCs.OnSurface,
										}).Shrink(1).MinWidth(0)
									}
								})
							})
						} else {
							ui.Box(c).Size(1, 1)
						}

						ctrlPill := md3e.Surface(c, md3e.SurfaceOptions{
							Elevation:    tokens.Level3,
							Color:        pillBg,
							ContentColor: hudCs.OnSurface,
							ShapeScale:   tokens.ShapeFull,
							BorderWidth:  1,
							BorderColor:  pillBorder,
						}).Shrink(0).PaddingX(8).PaddingY(4).InsetShadow(0, 1, 0, 0, pillInset)
						ctrlPill.Clicked()
						ctrlPill.Children(func() {
							zoomPct := int(math.Round(float64(st.zoom * 100)))
							ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
								if md3e.IconButton(c, "Zoom out", symbols.ZoomOut.Draw, md3e.IconButtonOptions{
									Variant:  md3e.IconButtonStandard,
									Size:     tokens.SizeS,
									Disabled: st.zoom <= minZ+0.001,
								}).Clicked() {
									st.zoom = clampF32(st.zoom-0.5, minZ, maxZ)
									lightboxClampPan(st)
								}
								if md3e.TabularNums(md3e.Button(c, fmt.Sprintf("%d%%", zoomPct), md3e.ButtonOptions{
									Variant: md3e.ButtonText,
									Size:    tokens.SizeXS,
									Label:   fmt.Sprintf("Reset zoom %d%%", zoomPct),
								})).Clicked() {
									st.zoom = minZ
									st.panX, st.panY = 0, 0
								}
								if md3e.IconButton(c, "Zoom in", symbols.ZoomIn.Draw, md3e.IconButtonOptions{
									Variant:  md3e.IconButtonStandard,
									Size:     tokens.SizeS,
									Disabled: st.zoom >= maxZ-0.001,
								}).Clicked() {
									st.zoom = clampF32(st.zoom+0.5, minZ, maxZ)
									lightboxClampPan(st)
								}
								ui.Box(c).Size(1, 18).MarginX(2).Background(ui.RGBA(255, 255, 255, 0.18))
								if md3e.IconButton(c, "Close image viewer", symbols.Close.Draw, md3e.IconButtonOptions{
									Variant: md3e.IconButtonStandard,
									Size:    tokens.SizeS,
								}).Clicked() {
									closeLightbox()
								}
							})
						})
					})

					// Bottom caption pill (only when Caption is provided; avoids duplicating single-item Title).
					if item.Caption != "" {
						ui.Row(c).Absolute().Bottom(14).Left(24).Right(24).Justify(ui.Center).Children(func() {
							capCard := md3e.Surface(c, md3e.SurfaceOptions{
								Elevation:    tokens.Level3,
								Color:        pillBg,
								ContentColor: hudCs.OnSurface,
								ShapeScale:   tokens.ShapeLarge,
								BorderWidth:  1,
								BorderColor:  pillBorder,
							}).MaxWidth(560).PaddingX(18).PaddingY(10).InsetShadow(0, 1, 0, 0, pillInset)
							capCard.Clicked()
							capCard.Children(func() {
								md3e.TextWith(c, item.Caption, md3e.TextOptions{
									Role:     tokens.BodyMedium,
									Color:    hudCs.OnSurfaceVariant,
									MaxLines: 2,
								})
							})
						})
					}
				})
			}).Fill()
		})
	})
	return anchor
}

func lightboxClampPan(st *lightboxState) {
	if st.zoom <= st.minZoom+0.001 {
		st.panX, st.panY = 0, 0
		return
	}
	if st.viewportW > 0 {
		maxPanX := st.viewportW * (st.zoom - 1) * 0.5
		if maxPanX < 0 {
			maxPanX = 0
		}
		st.panX = clampF32(st.panX, -maxPanX, maxPanX)
	}
	if st.viewportH > 0 {
		maxPanY := st.viewportH * (st.zoom - 1) * 0.5
		if maxPanY < 0 {
			maxPanY = 0
		}
		st.panY = clampF32(st.panY, -maxPanY, maxPanY)
	}
}
