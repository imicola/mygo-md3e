package md3e

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// ImageOptions configures md3e.Image for displaying local bitmaps or SVGs with
// MD3E shape scales, custom RoundedPolygon masks, smooth load fade-in, and
// tonal loading/error placeholders.
type ImageOptions struct {
	// Fit selects how src fits inside the box (Contain by default).
	Fit ui.Fit
	// ObjectPosition places the image within its box as [x, y] fractions in 0..1
	// when HasObjectPosition is true (default center [0.5, 0.5]).
	ObjectPosition    [2]float32
	HasObjectPosition bool

	// Shape selection priority: Shape polygon > Radii [4] > Radius > ShapeScale.
	ShapeScale tokens.ShapeScale
	Radius     float32
	HasRadius  bool
	Radii      [4]float32
	HasRadii   bool
	Shape      *shape.RoundedPolygon

	// BorderWidth draws a crisp outline around the image container or polygon.
	BorderWidth float32
	// BorderColor overrides the outline color (defaults to OutlineVariant).
	BorderColor ui.Color
	// Background overrides the tonal placeholder/backdrop fill color.
	Background ui.Color

	// Grayscale renders rectangular/rounded-corner images in shades of gray.
	// Note: when Shape (custom polygon) is set, a subtle tonal veil is used
	// because ui.Painter clips polygons independently of ui.Image.Grayscale.
	Grayscale bool

	// Width, Height, and AspectRatio optionally size the host box in DIPs.
	// When Width and Height are zero and src is non-nil, the intrinsic size of
	// src is used unless overridden by chaining .Size(...) or .Fill().
	Width       float32
	Height      float32
	AspectRatio float32

	// FadeIn fades the image in on initial mount. Transitions from Loading/nil
	// to a loaded src fade in automatically unless NoFade or ReduceMotion is set.
	FadeIn bool
	// NoFade disables fade-in opacity transitions.
	NoFade bool

	// Loading renders an MD3E tonal pulsing Skeleton clipped to the image shape.
	Loading bool
	// Failed renders the MD3E error placeholder (implied automatically when
	// Error != nil).
	Failed bool
	// Error triggers the error placeholder when non-nil.
	Error error
	// ErrorText displays optional supporting copy below the error icon.
	ErrorText string
	// RetryLabel customizes the error retry button text (defaults to "Retry").
	RetryLabel string
	// OnRetry renders an interactive retry button in the error state.
	OnRetry func()
	// LoadingContent overrides the default Skeleton when Loading is true.
	LoadingContent func()
	// ErrorContent overrides the default error placeholder when Failed/Error != nil.
	ErrorContent func()

	// Label names the image (ui.RoleImage, or ui.RoleButton when OnClick != nil).
	Label string
	// MatchKey attaches a shared-element ui.ElementTransition (Fly: true) for
	// smooth hero transitions (for example into media.ImageLightbox).
	MatchKey any
	// OnClick makes the non-error image interactive with MD3E StateLayer/Ripple.
	OnClick func()
	// Disabled disables interaction and applies MD3E disabled visual treatment.
	Disabled bool
}

type (
	imageFadeKey       struct{}
	imageFadeAlphaKey  struct{}
	imagePolyClockKey  struct{}
	imageChildKey      struct{}
	imageSkeletonKey   struct{}
	imageRetryBtnKey   struct{}
	imageCustomSlotKey struct{}
	imagePolyBorderKey struct{}
)

type imageFadeState struct {
	initialized bool
	sawWaiting  bool
	fading      bool
}

type imagePolyClock struct {
	start     time.Time
	lastPulse float32
}

// Image builds an MD3E image container displaying local bitmaps or SVGs
// (ui.ImageSource) with Fit, ObjectPosition, ShapeScale / independent Radii /
// *shape.RoundedPolygon clipping, smooth load fade-in, tonal Skeleton loading
// placeholder, error placeholder with optional retry, and optional click ripple.
func Image(c *ui.Context, src ui.ImageSource, opts ...ImageOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: Image takes at most one ImageOptions")
	}
	var o ImageOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	poly := validShapePolygon(o.Shape, nil)
	radii := resolveImageRadii(th, o)

	isError := o.Failed || o.Error != nil
	isLoading := !isError && o.Loading
	hasImage := !isError && !isLoading && src != nil

	host := ui.Box(c).DimDisabled(false).FocusRing(false)
	host.Disabled(o.Disabled)
	disabled := o.Disabled || host.IsDisabled()

	iw, ih := ui.ImageSize(src)
	hasIntrinsic := isPositiveFiniteF32(iw) && isPositiveFiniteF32(ih)
	aspect := sanitizeNonNegF32(o.AspectRatio)
	if aspect == 0 && hasIntrinsic {
		aspect = iw / ih
	}
	w := sanitizeNonNegF32(o.Width)
	h := sanitizeNonNegF32(o.Height)
	switch {
	case w > 0 && h > 0:
		host.Size(w, h)
	case w > 0:
		host.Width(w)
		if aspect > 0 {
			host.AspectRatio(aspect)
		}
	case h > 0:
		host.Height(h)
		if aspect > 0 {
			host.AspectRatio(aspect)
		}
	case aspect > 0 && hasIntrinsic:
		host.Size(iw, ih)
	case aspect > 0:
		host.AspectRatio(aspect)
	}

	if o.MatchKey != nil {
		anim := th.Animation(motion.FastSpatial)
		if reduce {
			anim = ui.Animation{Duration: -1}
		}
		host.Transition(ui.ElementTransition{
			Animation: &anim,
			Match:     o.MatchKey,
			Fly:       true,
		})
	}

	// Accessibility and interaction.
	interactive := o.OnClick != nil && !isError && !isLoading
	if interactive {
		host.Role(ui.RoleButton).Label(labelOr(o.Label, "Image"))
		if !disabled {
			host.Focusable()
		}
		if !disabled && host.Clicked() {
			o.OnClick()
		}
		attachRippleWithContext(c, th, host, StateLayerOptions{
			Color:          cs.OnSurface,
			Radii:          radii,
			HasRadii:       radii != [4]float32{},
			Shape:          poly,
			FocusHaloColor: cs.Surface,
			Disabled:       disabled,
		})
	} else if o.Label != "" {
		host.Role(ui.RoleImage).Label(o.Label)
	}

	// Fade-in state tracking on the stable host element.
	fadeSt := ui.Local(host, imageFadeKey{}, func() imageFadeState { return imageFadeState{} })
	alpha := float32(1)
	switch {
	case isLoading || (!hasImage && !isError):
		fadeSt.initialized = true
		fadeSt.sawWaiting = true
		fadeSt.fading = false
		host.Jump(imageFadeAlphaKey{}, 0)
		alpha = 0
	case hasImage:
		switch {
		case !fadeSt.initialized:
			fadeSt.initialized = true
			fadeSt.sawWaiting = false
			if !o.NoFade && o.FadeIn && !reduce {
				fadeSt.fading = true
				host.Jump(imageFadeAlphaKey{}, 0)
				alpha = animateValue(host, imageFadeAlphaKey{}, 1, th.Animation(motion.DefaultEffects), false)
			} else {
				fadeSt.fading = false
				alpha = host.Jump(imageFadeAlphaKey{}, 1)
			}
		case fadeSt.sawWaiting:
			fadeSt.sawWaiting = false
			if !o.NoFade && !reduce {
				fadeSt.fading = true
				host.Jump(imageFadeAlphaKey{}, 0)
				alpha = animateValue(host, imageFadeAlphaKey{}, 1, th.Animation(motion.DefaultEffects), false)
			} else {
				fadeSt.fading = false
				alpha = host.Jump(imageFadeAlphaKey{}, 1)
			}
		case fadeSt.fading && !o.NoFade && !reduce:
			alpha = animateValue(host, imageFadeAlphaKey{}, 1, th.Animation(motion.DefaultEffects), false)
			if alpha >= 1 {
				fadeSt.fading = false
			}
		default:
			fadeSt.fading = false
			alpha = host.Jump(imageFadeAlphaKey{}, 1)
		}
	default:
		fadeSt.initialized = true
		fadeSt.sawWaiting = false
		fadeSt.fading = false
		alpha = host.Jump(imageFadeAlphaKey{}, 1)
	}
	alpha = clampF32(alpha, 0, 1)

	bg := o.Background
	if bg.A == 0 {
		if isError {
			bg = cs.ErrorContainer
			if bg.A == 0 {
				bg = cs.SurfaceContainerHighest
			}
		} else if isLoading || !hasImage || alpha < 1 {
			bg = cs.SurfaceContainerHighest
			if bg.A == 0 {
				bg = cs.SurfaceVariant
			}
		}
	}

	borderW := sanitizeNonNegF32(o.BorderWidth)
	borderCol := o.BorderColor
	if borderW > 0 && borderCol.A == 0 {
		borderCol = cs.OutlineVariant
		if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
			borderCol = cs.Outline
		}
	}

	objPos := [2]float32{0.5, 0.5}
	if o.HasObjectPosition {
		objPos = [2]float32{
			sanitizeFiniteOr(o.ObjectPosition[0], 0.5),
			sanitizeFiniteOr(o.ObjectPosition[1], 0.5),
		}
	}
	fit := o.Fit

	if poly != nil {
		scratch := localDrawPath(host)
		polyClock := ui.Local(host, imagePolyClockKey{}, func() imagePolyClock { return imagePolyClock{} })
		emptyInk := cs.OnSurfaceVariant.Alpha(0.55)
		errInk := cs.OnErrorContainer
		if errInk.A == 0 {
			errInk = cs.OnSurfaceVariant
		}
		highlightCol := bg.Mix(cs.OnSurface, 0.08)

		host.Draw(func(p *ui.Painter, r ui.Rect) {
			if !rectIntersects(r, p.Visible()) {
				return
			}
			path := poly.WriteTo(scratch, r)
			if isLoading && o.LoadingContent == nil {
				live := host.Valid()
				red := reduce || p.Preferences().ReduceMotion
				pulse := float32(0)
				if !red {
					if live {
						now := p.Now()
						if polyClock.start.IsZero() {
							polyClock.start = now
						}
						elapsed := max(now.Sub(polyClock.start), 0)
						phase := float64(elapsed%skeletonPulsePeriod) / float64(skeletonPulsePeriod)
						pulse = float32(0.5 - 0.5*math.Cos(2*math.Pi*phase))
						polyClock.lastPulse = pulse
						p.AnimationFrame()
					} else {
						pulse = polyClock.lastPulse
					}
				}
				p.FillPath(path, bg.MixOklab(highlightCol, clampF32(pulse, 0, 1)))
				return
			}
			if bg.A > 0 {
				p.FillPath(path, bg)
			}
			if !hasImage && !isLoading && !isError {
				drawCenteredIconInRect(p, r, mdicon.Photo.Draw, emptyInk)
			}
			if isError && o.ErrorContent == nil && o.OnRetry == nil && o.ErrorText == "" {
				drawCenteredIconInRect(p, r, mdicon.ErrorOutline.Draw, errInk)
			}
		})

		paintBorderOverlay := func() {
			if borderW > 0 && borderCol.A > 0 {
				// Paint the polygon stroke in a PassThrough child overlay rather than
				// host.DrawOver so interactive polygon images keep their StateLayer/Ripple.
				ui.Box(c.Key(imagePolyBorderKey{})).Absolute().Fill().PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
					if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
						return
					}
					p.StrokePath(poly.WriteTo(scratch, insetRect(r, borderW*0.5)), borderW, borderCol)
				})
			}
		}

		if hasImage {
			host.Children(func() {
				layer := ui.Box(c.Key(imageChildKey{})).Absolute().Fill().PassThrough()
				if alpha < 1 {
					layer.Opacity(alpha)
				}
				layer.Draw(func(p *ui.Painter, r ui.Rect) {
					if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
						return
					}
					shape.ClipShapeTo(p, scratch, poly, r, func() {
						drawImageInRect(p, src, r, fit, objPos, o.HasObjectPosition)
						if disabled {
							p.Fill(r, cs.Surface.Alpha(0.38), 0)
						} else if o.Grayscale {
							p.Fill(r, cs.Surface.Alpha(0.28), 0)
						}
					})
				})
				paintBorderOverlay()
			})
		} else if isLoading && o.LoadingContent != nil {
			host.Clip().Children(func() {
				ui.Box(c.Key(imageCustomSlotKey{})).Absolute().Fill().Center().Children(o.LoadingContent)
				paintBorderOverlay()
			})
		} else if isError {
			buildImageErrorSlot(c, th, host, o, w, h, true)
			if borderW > 0 && borderCol.A > 0 {
				host.Children(func() {
					paintBorderOverlay()
				})
			}
		} else if borderW > 0 && borderCol.A > 0 {
			host.Children(func() {
				paintBorderOverlay()
			})
		}
		return host
	}

	// Rectangular / rounded-corner path.
	host.Radius(radii[0], radii[1], radii[2], radii[3]).Clip()
	if bg.A > 0 {
		host.Background(bg)
	}
	if borderW > 0 && borderCol.A > 0 {
		host.Border(borderW, borderCol)
	}

	switch {
	case isLoading:
		host.Children(func() {
			if o.LoadingContent != nil {
				ui.Box(c.Key(imageCustomSlotKey{})).Absolute().Fill().Center().Children(o.LoadingContent)
				return
			}
			Skeleton(c.Key(imageSkeletonKey{}), SkeletonOptions{
				Radius: maxRadius(radii),
			}).Absolute().Fill()
		})
	case isError:
		buildImageErrorSlot(c, th, host, o, w, h, false)
	case hasImage:
		host.Children(func() {
			imgEl := ui.Image(c.Key(imageChildKey{}), src).
				Absolute().
				Fill().
				Fit(fit).
				Radius(radii[0], radii[1], radii[2], radii[3])
			if o.HasObjectPosition {
				imgEl.ObjectPosition(objPos[0], objPos[1])
			}
			if o.Grayscale || disabled {
				imgEl.Grayscale()
			}
			if alpha < 1 {
				imgEl.Opacity(alpha)
			}
		})
	default:
		emptyInk := cs.OnSurfaceVariant.Alpha(0.55)
		host.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			drawCenteredIconInRect(p, r, mdicon.Photo.Draw, emptyInk)
		})
	}

	return host
}

func buildImageErrorSlot(c *ui.Context, th *Theme, host ui.Element, o ImageOptions, w, h float32, polyMode bool) {
	if o.ErrorContent != nil {
		host.Children(func() {
			ui.Box(c.Key(imageCustomSlotKey{})).Absolute().Fill().Center().Children(o.ErrorContent)
		})
		return
	}
	if polyMode && o.OnRetry == nil && o.ErrorText == "" {
		return
	}
	cs := th.Colors
	errInk := cs.OnErrorContainer
	if errInk.A == 0 {
		errInk = cs.OnSurfaceVariant
	}
	compact := (w > 0 && w < 110) || (h > 0 && h < 84)
	host.Children(func() {
		col := ui.Column(c.Key(imageCustomSlotKey{})).
			Absolute().
			Fill().
			Padding(8).
			Gap(6).
			AlignItems(ui.Center).
			Justify(ui.Center)
		col.Children(func() {
			iconSide := float32(24)
			if compact {
				iconSide = 20
			}
			ui.Box(c).Size(iconSide, iconSide).Draw(func(p *ui.Painter, r ui.Rect) {
				mdicon.ErrorOutline.Draw(p, r, errInk)
			})
			if o.ErrorText != "" && !compact {
				TextWith(c, o.ErrorText, TextOptions{
					Role:     tokens.BodySmall,
					MaxLines: 2,
					Color:    errInk,
				}).TextAlign(ui.Center)
			}
			if o.OnRetry != nil {
				retryLbl := labelOr(o.RetryLabel, "Retry")
				var btn ui.Element
				if compact && o.RetryLabel == "" {
					btn = IconButton(c.Key(imageRetryBtnKey{}), retryLbl, mdicon.Refresh.Draw, IconButtonOptions{
						Variant: IconButtonTonal,
						Size:    tokens.SizeXS,
					})
				} else {
					btn = Button(c.Key(imageRetryBtnKey{}), retryLbl, ButtonOptions{
						Variant:     ButtonTonal,
						Size:        tokens.SizeXS,
						LeadingIcon: mdicon.Refresh.Draw,
					})
				}
				if btn.Clicked() {
					o.OnRetry()
				}
			}
		})
	})
}

func drawCenteredIconInRect(p *ui.Painter, r ui.Rect, draw IconDraw, ink ui.Color) {
	if draw == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	side := clampF32(min(r.W, r.H)*0.35, 16, 36)
	dst := ui.Rect{
		X: r.X + (r.W-side)/2,
		Y: r.Y + (r.H-side)/2,
		W: side,
		H: side,
	}
	draw(p, dst, ink)
}

func drawImageInRect(p *ui.Painter, src ui.ImageSource, r ui.Rect, fit ui.Fit, objPos [2]float32, hasObjPos bool) {
	if src == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	if !hasObjPos || (objPos[0] == 0.5 && objPos[1] == 0.5) {
		p.Image(src, r, fit)
		return
	}
	iw, ih := ui.ImageSize(src)
	if iw <= 0 || ih <= 0 {
		p.Image(src, r, fit)
		return
	}
	if fit == ui.FillBox {
		p.Image(src, r, ui.FillBox)
		return
	}
	scale := float32(1)
	switch fit {
	case ui.Contain:
		scale = min(r.W/iw, r.H/ih)
	case ui.Cover:
		scale = max(r.W/iw, r.H/ih)
	case ui.ScaleDown:
		scale = min(r.W/iw, r.H/ih, 1)
	}
	dw, dh := iw*scale, ih*scale
	dst := ui.Rect{
		X: r.X + (r.W-dw)*objPos[0],
		Y: r.Y + (r.H-dh)*objPos[1],
		W: dw,
		H: dh,
	}
	p.Clip(r, 0, func() {
		p.Image(src, dst, ui.FillBox)
	})
}

func resolveImageRadii(th *Theme, o ImageOptions) [4]float32 {
	if o.HasRadii || o.Radii != [4]float32{} {
		return [4]float32{
			sanitizeNonNegF32(o.Radii[0]),
			sanitizeNonNegF32(o.Radii[1]),
			sanitizeNonNegF32(o.Radii[2]),
			sanitizeNonNegF32(o.Radii[3]),
		}
	}
	if o.HasRadius || isPositiveFiniteF32(o.Radius) {
		r := sanitizeNonNegF32(o.Radius)
		return [4]float32{r, r, r, r}
	}
	if o.ShapeScale > tokens.ShapeNone && o.ShapeScale <= tokens.ShapeFull {
		r := th.Shapes.Radius(o.ShapeScale)
		return [4]float32{r, r, r, r}
	}
	return [4]float32{}
}

func maxRadius(r [4]float32) float32 {
	return max(max(r[0], r[1]), max(r[2], r[3]))
}

func sanitizeFiniteOr(v, fallback float32) float32 {
	f := float64(v)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fallback
	}
	return v
}
