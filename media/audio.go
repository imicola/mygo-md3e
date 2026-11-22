package media

import (
	"fmt"
	"image"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// TrackInfo describes the metadata, artwork, and playback progress rendered by
// AudioPlayerCard and MiniPlayer.
type TrackInfo struct {
	Title  string
	Artist string
	Album  string
	// Artwork supplies a pre-decoded bitmap or SVG cover image.
	Artwork ui.ImageSource
	// ArtworkImage supplies raw pixels used for memoized dynamic theme seed
	// extraction (when DynamicColor is true) and cached bitmap fallback.
	ArtworkImage image.Image
	// ArtworkURL fetches a remote album cover via NetImage when Artwork and
	// ArtworkImage are nil.
	ArtworkURL string
	Position   time.Duration
	Duration   time.Duration
	// Buffered specifies optional 0..1 buffered fraction shown behind the head.
	Buffered float64
}

// AudioPlayerOptions configures AudioPlayerCard controls, dynamic album cover
// theming, artwork shape clipping, elevation, and sizing.
type AudioPlayerOptions struct {
	Playing     *bool
	OnPlayPause func(playing bool)
	OnSeek      func(pos time.Duration)
	OnPrevious  func()
	OnNext      func()
	Shuffle     *bool
	OnShuffle   func(shuffle bool)
	Repeat      *bool
	OnRepeat    func(repeat bool)
	OnQueue     func()

	// DynamicColor opts in to memoized tokens.SeedFromImage color extraction
	// from TrackInfo.ArtworkImage wrapped in md3e.AnimatedThemeScope.
	DynamicColor    bool
	ColorVariant    tokens.Variant
	HasColorVariant bool
	// SeedColor optionally overrides dynamic color with a fixed seed hue.
	SeedColor ui.Color

	// Wavy toggles the expressive wavy timeline while playing (defaults to true).
	Wavy *bool
	// ArtworkShape optionally clips the album artwork with an MD3E polygon.
	ArtworkShape *shape.RoundedPolygon
	// Elevation overrides card elevation (defaults to tokens.Level2; set
	// HasElevation: true with tokens.Level0 for flat).
	Elevation    tokens.ElevationLevel
	HasElevation bool
	Width        float32
	Label        string
}

// MiniPlayerOptions configures MiniPlayer controls, dynamic album color
// extraction, edge progress bar styling, and density-scaled height.
type MiniPlayerOptions struct {
	Playing     *bool
	OnPlayPause func(playing bool)
	OnPrevious  func()
	OnNext      func()
	OnExpand    func()

	DynamicColor    bool
	ColorVariant    tokens.Variant
	HasColorVariant bool
	SeedColor       ui.Color

	// Wavy toggles LinearWavyProgress while actively playing (defaults to true).
	Wavy   *bool
	Height float32
	Label  string
}

type (
	audioCardHostKey    struct{}
	audioCoverMemoKey   struct{}
	audioLocalPlayKey   struct{}
	audioPlayMorphKey   struct{}
	audioScrubberKey    struct{}
	audioShuffleBtnKey  struct{}
	audioRepeatBtnKey   struct{}
	miniPlayerHostKey   struct{}
	miniLocalPlayKey    struct{}

	audioCoverMemo struct {
		lastImg       image.Image
		lastBitmap    *ui.Bitmap
		hasSeed       bool
		extractedSeed ui.Color
		lastSeed      ui.Color
		lastVariant   tokens.Variant
		lastDark      bool
		lastContrast  tokens.Contrast
		lastDensity   tokens.Density
		cachedTheme   *md3e.Theme
	}
)

// AudioPlayerCard renders an elevated MD3E Expressive audio card with cover
// artwork (optionally polygon-clipped), track metadata, a wavy/capsule scrubber
// with TabularNums timestamps, FocusGroup transport actions, and optional
// memoized dynamic album cover theming via AnimatedThemeScope.
func AudioPlayerCard(c *ui.Context, track TrackInfo, opts ...AudioPlayerOptions) ui.Element {
	if len(opts) > 1 {
		panic("media: AudioPlayerCard takes at most one AudioPlayerOptions")
	}
	var o AudioPlayerOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	host := ui.Box(c)
	coverSrc, dynTheme, useDynamic := resolveTrackCoverAndTheme(
		c, host, track, o.DynamicColor, o.SeedColor, o.ColorVariant, o.HasColorVariant,
	)

	renderCard := func() {
		th := md3e.CurrentTheme(c)
		cs := th.Colors

		localPlay := ui.Local(host, audioLocalPlayKey{}, func() bool { return false })
		playPtr := o.Playing
		if playPtr == nil {
			playPtr = localPlay
		}
		playing := *playPtr
		wavy := o.Wavy == nil || *o.Wavy

		elev := tokens.Level2
		if o.HasElevation {
			elev = o.Elevation
		}

		card := md3e.Surface(c, md3e.SurfaceOptions{
			Elevation:    elev,
			Color:        cs.SurfaceContainerHigh,
			ContentColor: cs.OnSurface,
			ShapeScale:   tokens.ShapeExtraLarge,
			Hairline:     true,
		}).Padding(18)
		card.Role(ui.RoleRegion).Label(firstNonEmpty(o.Label, track.Title, "Audio player"))
		if isPositiveFiniteF32(o.Width) {
			card.Width(o.Width)
		} else {
			card.FillWidth()
		}

		card.Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				// Artwork + Track Info header row.
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(14).Children(func() {
					artSize := float32(72)
					renderTrackArtwork(c, coverSrc, track.ArtworkURL, artSize, o.ArtworkShape, firstNonEmpty(track.Title, "Album artwork"))
					ui.Column(c).Grow(1).Gap(3).Children(func() {
						md3e.TextWith(c, firstNonEmpty(track.Title, "Untitled track"), md3e.TextOptions{
							Role:       tokens.TitleMedium,
							Emphasized: true,
							MaxLines:   1,
						})
						if track.Artist != "" || track.Album != "" {
							sub := track.Artist
							if track.Artist != "" && track.Album != "" {
								sub = track.Artist + " — " + track.Album
							} else if track.Album != "" {
								sub = track.Album
							}
							md3e.TextWith(c, sub, md3e.TextOptions{
								Role:     tokens.BodyMedium,
								Color:    cs.OnSurfaceVariant,
								MaxLines: 1,
							})
						}
					})
				})

				// Expressive wavy scrubber + tabular timestamps.
				ui.Column(c).FillWidth().Gap(2).Children(func() {
					mediaScrubber(c.Key(audioScrubberKey{}), scrubberOptions{
						position:   track.Position,
						duration:   track.Duration,
						buffered:   track.Buffered,
						playing:    playing,
						wavy:       wavy,
						active:     cs.Primary,
						buffer:     cs.Primary.Alpha(0.32),
						track:      cs.SecondaryContainer,
						thumb:      cs.Primary,
						bubbleBg:   cs.InverseSurface,
						bubbleFg:   cs.InverseOnSurface,
						height:     28,
						thumbH:     20,
						showBubble: true,
						label:      "Audio playback timeline",
						onSeek:     o.OnSeek,
					})
					ui.Row(c).FillWidth().Justify(ui.SpaceBetween).Children(func() {
						md3e.TextWith(c, formatMediaTime(track.Position), md3e.TextOptions{
							Role:        tokens.LabelSmall,
							Color:       cs.OnSurfaceVariant,
							TabularNums: true,
						})
						md3e.TextWith(c, formatMediaTime(track.Duration), md3e.TextOptions{
							Role:        tokens.LabelSmall,
							Color:       cs.OnSurfaceVariant,
							TabularNums: true,
						})
					})
				})

				// Transport bar inside horizontal FocusGroup.
				ui.Row(c).FillWidth().Justify(ui.SpaceBetween).AlignItems(ui.Center).
					FocusGroup(ui.Horizontal).Children(func() {
					if o.Shuffle != nil {
						if md3e.ToggleIconButton(c.Key(audioShuffleBtnKey{}), o.Shuffle, "Shuffle", symbols.Shuffle.Draw, md3e.ToggleIconButtonOptions{
							Variant: md3e.IconButtonStandard,
							Size:    tokens.SizeS,
						}).Changed() && o.OnShuffle != nil {
							o.OnShuffle(*o.Shuffle)
						}
					} else {
						ui.Box(c).Size(36, 1)
					}

					ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
						if md3e.IconButton(c, "Previous track", symbols.SkipPrevious.Draw, md3e.IconButtonOptions{
							Variant:  md3e.IconButtonStandard,
							Size:     tokens.SizeS,
							Disabled: o.OnPrevious == nil,
						}).Clicked() && o.OnPrevious != nil {
							o.OnPrevious()
						}

						playLabel := "Play"
						if *playPtr {
							playLabel = "Pause"
						}
						md3e.MorphIconButton(c.Key(audioPlayMorphKey{}), playPtr, playLabel, symbols.PlayArrow.Draw, md3e.MorphIconButtonOptions{
							Variant:       md3e.ShapeButtonFilled,
							Size:          tokens.SizeM,
							Shape:         defaultVideoPlayShape(),
							SelectedShape: defaultVideoPauseShape(),
							SelectedIcon:  symbols.Pause.Draw,
							OnChanged: func(next bool) {
								if o.OnPlayPause != nil {
									o.OnPlayPause(next)
								}
							},
						})

						if md3e.IconButton(c, "Next track", symbols.SkipNext.Draw, md3e.IconButtonOptions{
							Variant:  md3e.IconButtonStandard,
							Size:     tokens.SizeS,
							Disabled: o.OnNext == nil,
						}).Clicked() && o.OnNext != nil {
							o.OnNext()
						}
					})

					if o.Repeat != nil || o.OnQueue != nil {
						ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
							if o.Repeat != nil {
								if md3e.ToggleIconButton(c.Key(audioRepeatBtnKey{}), o.Repeat, "Repeat", symbols.Repeat.Draw, md3e.ToggleIconButtonOptions{
									Variant:      md3e.IconButtonStandard,
									Size:         tokens.SizeS,
									SelectedIcon: symbols.RepeatOne.Draw,
								}).Changed() && o.OnRepeat != nil {
									o.OnRepeat(*o.Repeat)
								}
							}
							if o.OnQueue != nil {
								if md3e.IconButton(c, "Queue", symbols.QueueMusic.Draw, md3e.IconButtonOptions{
									Variant: md3e.IconButtonStandard,
									Size:    tokens.SizeS,
								}).Clicked() {
									o.OnQueue()
								}
							}
						})
					} else {
						ui.Box(c).Size(36, 1)
					}
				})
			})
		})
	}

	host.Children(func() {
		if useDynamic && dynTheme != nil {
			md3e.AnimatedThemeScope(c, dynTheme, renderCard)
		} else {
			renderCard()
		}
	})
	return host
}

// MiniPlayer renders a compact density-scaled transport bar with album artwork
// thumbnail, track summary, expandable click area, transport buttons, edge-to-
// edge progress bar (wavy while playing, crisp linear when paused), and
// optional memoized dynamic cover theming.
func MiniPlayer(c *ui.Context, track TrackInfo, opts ...MiniPlayerOptions) ui.Element {
	if len(opts) > 1 {
		panic("media: MiniPlayer takes at most one MiniPlayerOptions")
	}
	var o MiniPlayerOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	host := ui.Box(c).FillWidth()
	coverSrc, dynTheme, useDynamic := resolveTrackCoverAndTheme(
		c, host, track, o.DynamicColor, o.SeedColor, o.ColorVariant, o.HasColorVariant,
	)

	renderBar := func() {
		th := md3e.CurrentTheme(c)
		cs := th.Colors
		reduce := th.ReduceMotion || c.Preferences().ReduceMotion

		localPlay := ui.Local(host, miniLocalPlayKey{}, func() bool { return false })
		playPtr := o.Playing
		if playPtr == nil {
			playPtr = localPlay
		}
		playing := *playPtr
		wavy := o.Wavy == nil || *o.Wavy

		h := float32(64)
		switch th.Density.Normalize() {
		case tokens.DensityDesktop:
			h = 56
		case tokens.DensityCompact:
			h = 52
		}
		if isPositiveFiniteF32(o.Height) {
			h = o.Height
		}

		bar := md3e.Surface(c, md3e.SurfaceOptions{
			Elevation:    tokens.Level2,
			Color:        cs.SurfaceContainerHigh,
			ContentColor: cs.OnSurface,
			ShapeScale:   tokens.ShapeLarge,
			Hairline:     true,
			Clip:         true,
		}).FillWidth().Height(h)
		bar.Role(ui.RoleRegion).Label(firstNonEmpty(o.Label, track.Title, "Mini player"))

		bar.Children(func() {
			ui.Column(c).Fill().Justify(ui.SpaceBetween).Children(func() {
				ui.Row(c).FillWidth().Grow(1).PaddingX(12).AlignItems(ui.Center).Gap(10).Children(func() {
					// Clickable track info area when OnExpand is supplied.
					infoBox := ui.Row(c).Grow(1).AlignItems(ui.Center).Gap(10)
					if o.OnExpand != nil {
						infoLabel := "Expand player"
						if track.Title != "" {
							infoLabel = fmt.Sprintf("Expand %s", track.Title)
						}
						infoBox.Role(ui.RoleButton).Label(infoLabel).Focusable()
						if infoBox.Clicked() {
							o.OnExpand()
						}
					}
					infoBox.Children(func() {
						artSize := max(32, h-20)
						renderTrackArtwork(c, coverSrc, track.ArtworkURL, artSize, nil, firstNonEmpty(track.Title, "Album artwork"))
						ui.Column(c).Grow(1).Gap(1).Children(func() {
							md3e.TextWith(c, firstNonEmpty(track.Title, "Untitled track"), md3e.TextOptions{
								Role:       tokens.TitleSmall,
								Emphasized: true,
								MaxLines:   1,
							})
							if track.Artist != "" {
								md3e.TextWith(c, track.Artist, md3e.TextOptions{
									Role:     tokens.BodySmall,
									Color:    cs.OnSurfaceVariant,
									MaxLines: 1,
								})
							}
						})
					})

					// Compact transport controls.
					ui.Row(c).AlignItems(ui.Center).Gap(2).FocusGroup(ui.Horizontal).Children(func() {
						if o.OnPrevious != nil {
							if md3e.IconButton(c, "Previous track", symbols.SkipPrevious.Draw, md3e.IconButtonOptions{
								Variant: md3e.IconButtonStandard,
								Size:    tokens.SizeXS,
							}).Clicked() {
								o.OnPrevious()
							}
						}
						playIcon := symbols.PlayArrow.Draw
						playLabel := "Play"
						if playing {
							playIcon = symbols.Pause.Draw
							playLabel = "Pause"
						}
						if md3e.IconButton(c, playLabel, playIcon, md3e.IconButtonOptions{
							Variant: md3e.IconButtonFilled,
							Size:    tokens.SizeS,
						}).Clicked() {
							next := !playing
							*playPtr = next
							if o.OnPlayPause != nil {
								o.OnPlayPause(next)
							}
						}
						if o.OnNext != nil {
							if md3e.IconButton(c, "Next track", symbols.SkipNext.Draw, md3e.IconButtonOptions{
								Variant: md3e.IconButtonStandard,
								Size:    tokens.SizeXS,
							}).Clicked() {
								o.OnNext()
							}
						}
					})
				})

				// Edge-to-edge bottom playback progress bar (wavy only when playing & !ReduceMotion).
				progress := 0.0
				if track.Duration > 0 {
					progress = clampF64(track.Position.Seconds()/track.Duration.Seconds(), 0, 1)
				}
				progOpts := md3e.ProgressOptions{
					Thickness: 3,
					Buffer:    clampF64(track.Buffered, 0, 1),
					Label:     "Track progress",
				}
				if playing && wavy && !reduce {
					md3e.LinearWavyProgress(c, progress, progOpts).FillWidth()
				} else {
					md3e.LinearProgress(c, progress, progOpts).FillWidth()
				}
			})
		})
	}

	host.Children(func() {
		if useDynamic && dynTheme != nil {
			md3e.AnimatedThemeScope(c, dynTheme, renderBar)
		} else {
			renderBar()
		}
	})
	return host
}

func resolveTrackCoverAndTheme(
	c *ui.Context,
	host ui.Element,
	track TrackInfo,
	dynamicColor bool,
	seedOverride ui.Color,
	colorVariant tokens.Variant,
	hasVariant bool,
) (cover ui.ImageSource, dynTheme *md3e.Theme, useDynamic bool) {
	memo := ui.Local(host, audioCoverMemoKey{}, func() audioCoverMemo { return audioCoverMemo{} })

	if track.Artwork != nil {
		cover = track.Artwork
	} else if track.ArtworkImage != nil {
		if memo.lastImg != track.ArtworkImage || memo.lastBitmap == nil {
			memo.lastBitmap = ui.NewBitmap(track.ArtworkImage)
		}
		cover = memo.lastBitmap
	}

	if seedOverride.A == 0 && !(dynamicColor && track.ArtworkImage != nil) {
		memo.lastImg = track.ArtworkImage
		return cover, nil, false
	}

	base := md3e.CurrentTheme(c)
	variant := tokens.Expressive
	if hasVariant || colorVariant != 0 {
		variant = colorVariant
	}

	var seed ui.Color
	switch {
	case seedOverride.A > 0:
		seed = seedOverride
	case memo.lastImg == track.ArtworkImage && memo.hasSeed:
		seed = memo.extractedSeed
	default:
		seed = tokens.SeedFromImage(track.ArtworkImage)
		memo.lastImg = track.ArtworkImage
		memo.extractedSeed = seed
		memo.hasSeed = true
	}

	if memo.cachedTheme == nil ||
		memo.lastSeed != seed ||
		memo.lastVariant != variant ||
		memo.lastDark != base.Dark ||
		memo.lastContrast != base.Contrast ||
		memo.lastDensity != base.Density {
		memo.cachedTheme = md3e.ThemeFromSeed(seed, md3e.ThemeOptions{
			Dark:     base.Dark,
			Variant:  variant,
			Contrast: base.Contrast,
			Density:  base.Density,
		})
		memo.cachedTheme.ReduceMotion = base.ReduceMotion
		memo.cachedTheme.Shapes = base.Shapes
		memo.lastSeed = seed
		memo.lastVariant = variant
		memo.lastDark = base.Dark
		memo.lastContrast = base.Contrast
		memo.lastDensity = base.Density
	}

	return cover, memo.cachedTheme, true
}

func renderTrackArtwork(c *ui.Context, src ui.ImageSource, url string, size float32, poly *shape.RoundedPolygon, label string) ui.Element {
	if src == nil && url != "" {
		return NetImage(c, url, NetImageOptions{
			Width:      size,
			Height:     size,
			Fit:        ui.Cover,
			ShapeScale: tokens.ShapeMedium,
			Shape:      poly,
			Label:      label,
		})
	}
	return md3e.Image(c, src, md3e.ImageOptions{
		Width:      size,
		Height:     size,
		Fit:        ui.Cover,
		ShapeScale: tokens.ShapeMedium,
		Shape:      poly,
		NoFade:     true,
		Label:      label,
	})
}
