package media

import (
	"net/http"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/netimage"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

// NetImageOptions configures network image decoding, caching, shape clipping,
// placeholders, and user interaction for NetImage.
type NetImageOptions struct {
	// MaxSide constrains the maximum decoded pixel edge in ui/netimage
	// (0 preserves full image resolution).
	MaxSide int

	// Layout & visual styling mapped to md3e.ImageOptions:
	Fit               ui.Fit
	ObjectPosition    [2]float32
	HasObjectPosition bool
	ShapeScale        tokens.ShapeScale
	Radius            float32
	HasRadius         bool
	Radii             [4]float32
	HasRadii          bool
	Shape             *shape.RoundedPolygon
	BorderWidth       float32
	BorderColor       ui.Color
	Background        ui.Color
	Grayscale         bool
	Width             float32
	Height            float32
	AspectRatio       float32

	// State presentation & transitions:
	FadeIn         bool
	NoFade         bool
	ErrorText      string
	RetryLabel     string
	LoadingContent func()
	ErrorContent   func()
	// Fallback is displayed when url is empty.
	Fallback ui.ImageSource

	// Interaction & accessibility:
	Label    string
	MatchKey any
	OnClick  func()
	OnRetry  func()
	Disabled bool
}

// NetImage fetches, decodes, caches, and renders a remote HTTP/HTTPS image
// using github.com/egoist/mygo/ui/netimage, presenting MD3E tonal Skeleton
// placeholders while loading, smooth fade-in on arrival, optional shape
// clipping, and retryable error states.
func NetImage(c *ui.Context, url string, opts ...NetImageOptions) ui.Element {
	if len(opts) > 1 {
		panic("media: NetImage takes at most one NetImageOptions")
	}
	var o NetImageOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	imgOpts := md3e.ImageOptions{
		Fit:               o.Fit,
		ObjectPosition:    o.ObjectPosition,
		HasObjectPosition: o.HasObjectPosition,
		ShapeScale:        o.ShapeScale,
		Radius:            o.Radius,
		HasRadius:         o.HasRadius,
		Radii:             o.Radii,
		HasRadii:          o.HasRadii,
		Shape:             o.Shape,
		BorderWidth:       o.BorderWidth,
		BorderColor:       o.BorderColor,
		Background:        o.Background,
		Grayscale:         o.Grayscale,
		Width:             o.Width,
		Height:            o.Height,
		AspectRatio:       o.AspectRatio,
		FadeIn:            o.FadeIn,
		NoFade:            o.NoFade,
		ErrorText:         o.ErrorText,
		RetryLabel:        o.RetryLabel,
		LoadingContent:    o.LoadingContent,
		ErrorContent:      o.ErrorContent,
		Label:             o.Label,
		MatchKey:          o.MatchKey,
		OnClick:           o.OnClick,
		Disabled:          o.Disabled,
	}

	if url == "" {
		return md3e.Image(c, o.Fallback, imgOpts)
	}

	src, err := netimage.Load(c, url, netimage.Options{MaxSide: o.MaxSide})
	switch {
	case src != nil:
		return md3e.Image(c, src, imgOpts)
	case err != nil:
		imgOpts.Error = err
		imgOpts.Failed = true
		srv := c.Services()
		imgOpts.OnRetry = func() {
			netimage.Forget(url)
			srv.Invalidate()
			if o.OnRetry != nil {
				o.OnRetry()
			}
		}
		return md3e.Image(c, nil, imgOpts)
	default:
		imgOpts.Loading = true
		return md3e.Image(c, nil, imgOpts)
	}
}

// PreloadImages starts warming the ui/netimage memory and disk cache for urls.
func PreloadImages(urls ...string) {
	netimage.Preload(urls...)
}

// ForgetImage removes a cached URL (including cached failure backoff) so the
// next NetImage build fetches it anew.
func ForgetImage(url string) {
	netimage.Forget(url)
}

// ClearImageCache clears all cached network images in memory and on disk.
func ClearImageCache() {
	netimage.Clear()
}

// SetImageHTTPClient configures the HTTP client used by NetImage and
// PreloadImages (passing nil resets to ui/netimage defaults).
func SetImageHTTPClient(client *http.Client) {
	netimage.SetClient(client)
}
