package media

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/video"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// VideoPlayerOptions configures VideoPlayer chrome, scrubber styling,
// auto-hiding controls, rate menu, fullscreen hook, and layout geometry.
type VideoPlayerOptions struct {
	// Title optionally renders in the top overlay scrim when controls show.
	Title string
	// Fit chooses how video frames scale inside the player viewport (defaults
	// to ui.Contain).
	Fit ui.Fit
	// ShapeScale selects the rounded container radius token (defaults to
	// tokens.ShapeLarge).
	ShapeScale tokens.ShapeScale
	// Radius explicitly sets the container corner radius in DIP when HasRadius
	// is true (or positive when ShapeScale == 0).
	Radius    float32
	HasRadius bool

	// Buffered optionally supplies buffered fraction in 0..1 for the scrubber.
	Buffered float64
	// Wavy enables the expressive wavy active track while playing (defaults to
	// true when nil).
	Wavy *bool

	// AutoHideDelay sets how long controls stay visible after interaction
	// while actively playing (defaults to 3s).
	AutoHideDelay time.Duration
	// AlwaysShowControls keeps the control overlay permanently visible.
	AlwaysShowControls bool

	// Rates customizes playback speed options in the speed menu (defaults to
	// [0.5, 0.75, 1.0, 1.25, 1.5, 2.0]).
	Rates []float64

	// Fullscreen optionally binds external fullscreen state; when nil and
	// OnFullscreen != nil, VideoPlayer tracks fullscreen state internally.
	Fullscreen   *bool
	OnFullscreen func(fullscreen bool)
	// OnRetry supplies a retry action shown in error states.
	OnRetry func()

	// Sizing overrides. When Width and Height are omitted, FillWidth() +
	// AspectRatio (intrinsic video aspect or 16:9 fallback) is used.
	Width       float32
	Height      float32
	AspectRatio float32

	// Label overrides the region's accessible name ("Video player" fallback).
	Label string
}

type (
	videoStateKey    struct{}
	videoCtrlFadeKey struct{}
	videoPlayBtnKey  struct{}
	videoScrubKey    struct{}
	videoVolKey      struct{}
	videoRateKey     struct{}

	videoPlayerState struct {
		appeared     bool
		lastInteract time.Time
		controlsVis  bool
		rateMenuOpen bool
		localFS      bool
	}
)

var defaultPlaybackRates = []float64{0.5, 0.75, 1.0, 1.25, 1.5, 2.0}

// OpenVideo opens a local video file using github.com/egoist/mygo/ui/video.
func OpenVideo(path string) (*ui.VideoPlayer, error) {
	return video.Open(path)
}

// OpenVideoURL opens a remote HTTP/HTTPS video stream using
// github.com/egoist/mygo/ui/video.
func OpenVideoURL(url string) (*ui.VideoPlayer, error) {
	return video.OpenURL(url)
}

// OpenVideoData decodes an in-memory video container (or animated GIF) using
// github.com/egoist/mygo/ui/video.
func OpenVideoData(data []byte) (*ui.VideoPlayer, error) {
	return video.OpenData(data)
}

// VideoPlayer renders an Expressive MD3E video stage wrapping *ui.VideoPlayer
// with auto-hiding translucent controls, wavy/capsule scrubber with buffered
// progress and teardrop timestamp bubble, morphing Play/Pause button,
// TabularNums timestamps, volume slider, playback rate menu, fullscreen hook,
// buffering/error overlays, and keyboard shortcuts (Space/K play-pause,
// Left/Right ±5s RTL-aware seek, M mute, F fullscreen).
func VideoPlayer(c *ui.Context, player *ui.VideoPlayer, opts ...VideoPlayerOptions) ui.Element {
	if len(opts) > 1 {
		panic("media: VideoPlayer takes at most one VideoPlayerOptions")
	}
	var o VideoPlayerOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	rtl := c.Direction() == ui.RightToLeft

	radius := th.Shapes.Radius(tokens.ShapeLarge)
	if o.HasRadius {
		radius = max(0, o.Radius)
	} else if o.ShapeScale > 0 {
		radius = th.Shapes.Radius(o.ShapeScale)
	} else if isPositiveFiniteF32(o.Radius) {
		radius = o.Radius
	}

	aspect := float32(16.0 / 9.0)
	if isPositiveFiniteF32(o.AspectRatio) {
		aspect = o.AspectRatio
	} else if player != nil {
		if vw, vh := player.Size(); vw > 0 && vh > 0 {
			aspect = float32(vw) / float32(vh)
		}
	}

	host := ui.Box(c).
		Radius(radius).
		Clip().
		Background(ui.RGB(14, 14, 18)).
		Role(ui.RoleRegion).
		Label(firstNonEmpty(o.Label, o.Title, "Video player")).
		FocusRing(false)

	w := o.Width
	h := o.Height
	switch {
	case isPositiveFiniteF32(w) && isPositiveFiniteF32(h):
		host.Size(w, h)
	case isPositiveFiniteF32(w):
		host.Width(w).AspectRatio(aspect)
	case isPositiveFiniteF32(h):
		host.Height(h).AspectRatio(aspect)
	default:
		host.FillWidth().AspectRatio(aspect)
	}

	st := ui.Local(host, videoStateKey{}, func() videoPlayerState {
		return videoPlayerState{controlsVis: true}
	})
	now := c.Now()
	if st.lastInteract.IsZero() {
		st.lastInteract = now
	}
	srv := c.Services()

	var (
		ready     bool
		playing   bool
		ended     bool
		buffering bool
		playErr   error
		dur       time.Duration
		pos       time.Duration
		muted     bool
		vol       = 1.0
		rate      = 1.0
	)
	if player != nil {
		ready = player.Ready()
		playing = player.Playing()
		ended = player.Ended()
		buffering = player.Buffering()
		playErr = player.Err()
		dur = player.Duration()
		pos = player.Position()
		muted = player.Muted()
		vol = clampF64(player.Volume(), 0, 1)
		if r := player.Rate(); !math.IsNaN(r) && !math.IsInf(r, 0) && r > 0 {
			rate = r
		}
	}

	wavy := true
	if o.Wavy != nil {
		wavy = *o.Wavy
	}
	delay := o.AutoHideDelay
	if delay <= 0 {
		delay = 3 * time.Second
	}

	touchControls := func() {
		st.lastInteract = now
		st.controlsVis = true
	}

	togglePlay := func() {
		if player == nil || playErr != nil {
			return
		}
		if playing {
			player.Pause()
		} else {
			if ended {
				player.Seek(0)
			}
			player.Play()
		}
		playing = player.Playing()
		touchControls()
		srv.Invalidate()
	}

	seekBy := func(delta time.Duration) {
		if player == nil || dur <= 0 {
			return
		}
		next := pos + delta
		if next < 0 {
			next = 0
		} else if next > dur {
			next = dur
		}
		player.Seek(next)
		pos = next
		touchControls()
		srv.Invalidate()
	}

	toggleMute := func() {
		if player == nil {
			return
		}
		player.SetMuted(!muted)
		muted = player.Muted()
		touchControls()
		srv.Invalidate()
	}

	hasFS := o.Fullscreen != nil || o.OnFullscreen != nil
	fsState := st.localFS
	if o.Fullscreen != nil {
		fsState = *o.Fullscreen
	}
	toggleFullscreen := func() {
		if !hasFS {
			return
		}
		next := !fsState
		if o.Fullscreen != nil {
			*o.Fullscreen = next
		} else {
			st.localFS = next
		}
		fsState = next
		if o.OnFullscreen != nil {
			o.OnFullscreen(next)
		}
		touchControls()
		srv.Invalidate()
	}

	// Keyboard shortcuts active when focus is anywhere within the VideoPlayer.
	if player != nil && playErr == nil {
		switch {
		case host.Shortcut(0, ui.KeyK), host.Shortcut(0, ui.KeySpace):
			togglePlay()
		case host.Shortcut(0, ui.KeyM):
			toggleMute()
		case host.Shortcut(0, ui.KeyF):
			toggleFullscreen()
		case host.Shortcut(0, ui.KeyLeft):
			if rtl {
				seekBy(5 * time.Second)
			} else {
				seekBy(-5 * time.Second)
			}
		case host.Shortcut(0, ui.KeyRight):
			if rtl {
				seekBy(-5 * time.Second)
			} else {
				seekBy(5 * time.Second)
			}
		}
	}

	// Keep st.lastInteract fresh on any focus/press/key activity within the player.
	host.ObserveInput(func(ev ui.InputEvent) {
		if !ev.Time.IsZero() {
			st.lastInteract = ev.Time
		} else {
			st.lastInteract = now
		}
		st.controlsVis = true
	})

	// Decide whether controls remain active or auto-hide after AutoHideDelay.
	stayShown := o.AlwaysShowControls || !playing || ended || playErr != nil || player == nil || st.rateMenuOpen
	controlsActive := true
	if !stayShown {
		elapsed := now.Sub(st.lastInteract)
		if elapsed < 0 {
			elapsed = 0
		}
		controlsActive = elapsed < delay
		if controlsActive {
			host.DrawOver(func(p *ui.Painter, r ui.Rect) {
				if !host.Valid() || !rectIntersects(r, p.Visible()) {
					return
				}
				remaining := delay - max(p.Now().Sub(st.lastInteract), 0)
				if remaining > 0 {
					p.After(remaining)
				} else {
					srv.Invalidate()
				}
			})
		}
	}
	st.controlsVis = controlsActive

	fadeTarget := float32(0)
	if controlsActive {
		fadeTarget = 1
	}
	var ctrlAlpha float32
	if !st.appeared {
		st.appeared = true
		ctrlAlpha = host.Jump(videoCtrlFadeKey{}, fadeTarget)
	} else {
		ctrlAlpha = clampF32(animateF32(host, videoCtrlFadeKey{}, fadeTarget, th.Animation(motion.DefaultEffects), reduce), 0, 1)
	}

	fit := o.Fit
	if fit == 0 {
		fit = ui.Contain
	}

	host.Children(func() {
		// 1. Video frame layer.
		if player != nil {
			ui.Video(c, player).Fit(fit).Absolute().Fill()
		}

		// 2. Clickable / hover-revealing stage hit surface under the control bar.
		stage := ui.Box(c).Absolute().Fill().FocusRing(false)
		if player != nil && playErr == nil {
			stageLabel := "Play video"
			if playing {
				stageLabel = "Pause video"
			} else if ended {
				stageLabel = "Replay video"
			}
			stage.Role(ui.RoleButton).Label(stageLabel).Focusable()
			if stage.Clicked() || stage.Shortcut(0, ui.KeyEnter) {
				togglePlay()
			}
		}
		stage.HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputPointerMove {
				wasHidden := !st.controlsVis
				if !ev.Time.IsZero() {
					st.lastInteract = ev.Time
				} else {
					st.lastInteract = now
				}
				if wasHidden {
					st.controlsVis = true
					return true
				}
			}
			return false
		})

		// 3. Buffering spinner overlay when decoder isn't ready or is starving.
		if player != nil && (!ready || buffering) && playErr == nil {
			ui.Column(c).Absolute().Fill().Center().PassThrough().Children(func() {
				md3e.LoadingIndicator(c, md3e.LoadingOptions{
					Style: md3e.LoadingContained,
					Size:  48,
					Label: "Buffering video",
				})
			})
		}

		// 4. Error / missing player overlay.
		if player == nil || playErr != nil {
			errMsg := "Video source unavailable"
			if playErr != nil {
				errMsg = playErr.Error()
			}
			ui.Column(c).Absolute().Fill().Center().Background(ui.RGBA(0, 0, 0, 0.72)).Padding(16).Gap(10).Children(func() {
				errIcon := ui.Box(c).Size(36, 36)
				ink := cs.Error
				if ink.A == 0 {
					ink = ui.RGB(242, 184, 181)
				}
				errIcon.Draw(func(p *ui.Painter, r ui.Rect) {
					symbols.ErrorOutline.Draw(p, r, ink)
				})
				md3e.TextWith(c, errMsg, md3e.TextOptions{
					Role:       tokens.TitleSmall,
					Color:      ui.RGB(255, 255, 255),
					MaxLines:   2,
					Emphasized: true,
				})
				if o.OnRetry != nil {
					if md3e.Button(c, "Retry", md3e.ButtonOptions{
						Variant:     md3e.ButtonTonal,
						Size:        tokens.SizeS,
						LeadingIcon: symbols.Refresh.Draw,
					}).Clicked() {
						o.OnRetry()
					}
				}
			})
		}

		// 5. Title top gradient scrim (fades alongside bottom controls).
		if player != nil && playErr == nil && o.Title != "" && ctrlAlpha > 0.01 {
			ui.Row(c).Absolute().Top(0).Left(0).Right(0).
				PaddingX(16).PaddingY(12).
				LinearGradient(ui.LinearGradient{
					Angle: 180,
					Stops: []ui.GradientStop{
						{At: 0, Color: ui.RGBA(0, 0, 0, 0.65)},
						{At: 1, Color: ui.Transparent},
					},
				}).
				Opacity(ctrlAlpha).
				PassThrough().
				Children(func() {
					md3e.TextWith(c, o.Title, md3e.TextOptions{
						Role:       tokens.TitleMedium,
						Emphasized: true,
						MaxLines:   1,
						Color:      ui.RGB(255, 255, 255),
					})
				})
		}

		// 6. Floating translucent MD3E bottom controls overlay.
		if player != nil && playErr == nil && ctrlAlpha > 0.01 {
			lightText := ui.RGB(255, 255, 255)
			lightMuted := ui.RGBA(255, 255, 255, 0.72)

			ctrlWrap := ui.Column(c).
				Absolute().Left(0).Right(0).Bottom(0).
				Padding(20, 12, 10, 12).
				Gap(6).
				LinearGradient(ui.LinearGradient{
					Angle: 0,
					Stops: []ui.GradientStop{
						{At: 0, Color: ui.RGBA(0, 0, 0, 0.82)},
						{At: 1, Color: ui.Transparent},
					},
				}).
				Opacity(ctrlAlpha).
				Translate(0, (1-ctrlAlpha)*10).
				IgnoreInput(!controlsActive).
				AccessHidden(!controlsActive)

			ctrlWrap.Children(func() {
				// Expressive wavy / capsule scrubber row.
				mediaScrubber(c.Key(videoScrubKey{}), scrubberOptions{
					position:   pos,
					duration:   dur,
					buffered:   o.Buffered,
					playing:    playing,
					wavy:       wavy,
					disabled:   player == nil || playErr != nil,
					active:     cs.Primary,
					buffer:     ui.RGBA(255, 255, 255, 0.42),
					track:      ui.RGBA(255, 255, 255, 0.24),
					thumb:      lightText,
					bubbleBg:   cs.SurfaceContainerHighest,
					bubbleFg:   cs.OnSurface,
					height:     28,
					thumbH:     20,
					showBubble: true,
					label:      "Video timeline",
					onSeek: func(next time.Duration) {
						if player != nil {
							player.Seek(next)
							pos = next
							touchControls()
						}
					},
					onInteract: touchControls,
				})

				// Control bar pill container.
				barCard := ui.Box(c).FillWidth().
					Radius(th.Shapes.Radius(tokens.ShapeLarge)).
					Background(cs.SurfaceContainerHigh.Alpha(0.86)).
					Border(1, cs.OutlineVariant.Alpha(0.28)).
					PaddingX(10).PaddingY(6)
				// Consume clicks on the bar itself so clicking between buttons doesn't toggle stage pause.
				barCard.Clicked()
				barCard.Children(func() {
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).Gap(8).Children(func() {
						// Start cluster: Morph Play/Pause + timestamp.
						ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
							isPlaying := playing
							playLabel := "Play"
							playIcon := symbols.PlayArrow.Draw
							if playing {
								playLabel = "Pause"
							} else if ended {
								playLabel = "Replay"
								playIcon = symbols.Replay.Draw
							}
							md3e.MorphIconButton(c.Key(videoPlayBtnKey{}), &isPlaying, playLabel, playIcon, md3e.MorphIconButtonOptions{
								Variant:       md3e.ShapeButtonFilled,
								Size:          tokens.SizeS,
								Shape:         defaultVideoPlayShape(),
								SelectedShape: defaultVideoPauseShape(),
								SelectedIcon:  symbols.Pause.Draw,
								Disabled:      player == nil || playErr != nil,
								OnChanged: func(nextPlay bool) {
									if player == nil {
										return
									}
									if nextPlay {
										if player.Ended() {
											player.Seek(0)
										}
										player.Play()
									} else {
										player.Pause()
									}
									touchControls()
									srv.Invalidate()
								},
							})

							timeStr := fmt.Sprintf("%s / %s", formatMediaTime(pos), formatMediaTime(dur))
							md3e.TextWith(c, timeStr, md3e.TextOptions{
								Role:        tokens.LabelLarge,
								TabularNums: true,
								Color:       cs.OnSurface,
							})
						})

						// End cluster: Mute + volume slider + playback speed menu + fullscreen.
						ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
							volIcon := symbols.VolumeUp.Draw
							muteLabel := "Mute"
							if muted || vol <= 0.001 {
								volIcon = symbols.VolumeOff.Draw
								muteLabel = "Unmute"
							} else if vol < 0.5 {
								volIcon = symbols.VolumeDown.Draw
							}
							if md3e.IconButton(c, muteLabel, volIcon, md3e.IconButtonOptions{
								Variant:  md3e.IconButtonStandard,
								Size:     tokens.SizeXS,
								Disabled: player == nil,
							}).Clicked() {
								toggleMute()
							}

							volVal := vol
							if muted {
								volVal = 0
							}
							volSlider := ui.SliderBase(c.Key(videoVolKey{}), &volVal, 0, 1).
								Disabled(player == nil).
								Label("Volume").
								FocusRing(false).
								Width(64).
								Height(24).
								PaddingX(6)
							if volSlider.Changed() && player != nil {
								player.SetVolume(volVal)
								if volVal > 0 && muted {
									player.SetMuted(false)
								} else if volVal <= 0.001 {
									player.SetMuted(true)
								}
								touchControls()
							}
							drawVolumeTrack(volSlider, cs.Primary, cs.SecondaryContainer, cs.OnSurface, volVal, player == nil, rtl)

							rates := o.Rates
							if len(rates) == 0 {
								rates = defaultPlaybackRates
							}
							rateItems := make([]md3e.MenuItem, 0, len(rates))
							for _, r := range rates {
								rVal := r
								if rVal <= 0 || math.IsNaN(rVal) || math.IsInf(rVal, 0) {
									continue
								}
								rateItems = append(rateItems, md3e.MenuItem{
									Label:   fmt.Sprintf("%gx", rVal),
									Radio:   true,
									Checked: math.Abs(rate-rVal) < 0.01,
									OnClick: func() {
										if player != nil {
											player.SetRate(rVal)
										}
										touchControls()
									},
								})
							}
							if len(rateItems) > 0 {
								md3e.MenuButton(c.Key(videoRateKey{}), fmt.Sprintf("%gx", rate), symbols.Speed.Draw, rateItems, md3e.MenuOptions{
									Open:     &st.rateMenuOpen,
									Label:    fmt.Sprintf("Playback speed %gx", rate),
									Disabled: player == nil,
								})
							}

							if hasFS {
								fsIcon := symbols.Fullscreen.Draw
								fsLabel := "Fullscreen"
								if fsState {
									fsIcon = symbols.FullscreenExit.Draw
									fsLabel = "Exit fullscreen"
								}
								if md3e.IconButton(c, fsLabel, fsIcon, md3e.IconButtonOptions{
									Variant: md3e.IconButtonStandard,
									Size:    tokens.SizeXS,
								}).Clicked() {
									toggleFullscreen()
								}
							}
						})
					})
				})
				_ = lightMuted
			})
		}
	})

	return host
}

func drawVolumeTrack(sl ui.Element, active, inactive, disabledInk ui.Color, val float64, disabled, rtl bool) {
	if disabled {
		active = disabledInk.Alpha(0.38)
		inactive = disabledInk.Alpha(0.14)
	}
	f := float32(clampF64(val, 0, 1))
	sl.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 12 || r.H <= 0 {
			return
		}
		pad := float32(6)
		x0 := r.X + pad
		trackW := r.W - 2*pad
		cy := r.Y + r.H*0.5
		p.Fill(ui.Rect{X: x0, Y: cy - 2, W: trackW, H: 4}, inactive, 2)
		actW := trackW * f
		actX := x0
		if rtl {
			actX = x0 + trackW - actW
		}
		if actW > 0 {
			p.Fill(ui.Rect{X: actX, Y: cy - 2, W: actW, H: 4}, active, 2)
		}
		tx := x0 + actW
		if rtl {
			tx = actX
		}
		p.Fill(ui.Rect{X: tx - 5, Y: cy - 5, W: 10, H: 10}, active, 5)
	})
}

var (
	videoPlayOnce  sync.Once
	videoPlayPoly  *shape.RoundedPolygon
	videoPauseOnce sync.Once
	videoPausePoly *shape.RoundedPolygon
)

func defaultVideoPlayShape() *shape.RoundedPolygon {
	videoPlayOnce.Do(func() {
		videoPlayPoly = shape.Circle()
	})
	return videoPlayPoly
}

func defaultVideoPauseShape() *shape.RoundedPolygon {
	videoPauseOnce.Do(func() {
		videoPausePoly = shape.Square()
	})
	return videoPausePoly
}
