package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/media"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

var errSimulatedArtLoad = errors.New("Preview error: asset decode timed out")

func (a *appState) mediaView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	class := md3e.ContentSizeClassOf(c)

	showQueue := a.media.showQueue
	if class == md3e.WindowCompact {
		showQueue = a.media.compactQueue
	}

	md3e.SupportingPane(c.Key("media-supporting"), func() {
		a.mediaMainPane(c, th, cs, class)
	}, func() {
		a.mediaQueuePane(c, th, cs, class)
	}, md3e.SupportingPaneOptions{
		ShowSupporting:     &showQueue,
		DefaultMainPercent: 68,
		Label:              "Media studio and play queue",
	})

	media.ImageLightbox(c.Key("media-lightbox"), &a.media.lightboxOpen, a.media.lightboxItems, media.LightboxOptions{
		Index:    &a.media.lightboxIdx,
		MatchKey: a.media.activeLightboxMatchKey(),
		OnIndexChanged: func(idx int) {
			a.media.selectedArtIdx = idx
			a.media.customPaletteSeed = ui.Color{}
			a.dirty = true
		},
		Label: "Artwork viewer",
	})

	if a.media.videoFullscreen {
		a.videoTheaterOverlay(c, th)
	}
}

func (a *appState) mediaMainPane(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme, class md3e.WindowSizeClass) {
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Children(func() {
		ui.Scroll(c.Key("media-scroll")).TrackScroll(&a.mediaScroll).Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
				ui.Column(c).FillWidth().MaxWidth(1120).Padding(18, 20, 24, 20).Gap(16).Children(func() {
					a.mediaHeader(c, th, cs, class)
					a.mediaNowPlayingSection(c, th, cs)
					a.mediaVideoSection(c, cs)
					a.mediaGallerySection(c, cs)
				})
			})
		})

		ui.Box(c.Key("mini-player-dock")).
			FillWidth().
			Shrink(0).
			Padding(6, 16, 12, 16).
			Background(cs.Surface).
			Border(1, cs.OutlineVariant.Alpha(0.5)).
			BorderWidth(1, 0, 0, 0).
			Children(func() {
				media.MiniPlayer(c.Key("workbench-mini-player"), a.media.currentTrackInfo(), media.MiniPlayerOptions{
					Playing:         &a.media.audioPlaying,
					DynamicColor:    true,
					ColorVariant:    variantPresets[a.variantIdx].variant,
					HasColorVariant: true,
					OnPlayPause: func(bool) {
						a.dirty = true
					},
					OnPrevious: func() {
						a.media.prevTrack()
						a.dirty = true
					},
					OnNext: func() {
						a.media.nextTrack()
						a.dirty = true
					},
					OnExpand: func() {
						anim := th.Animation(motion.DefaultSpatial)
						if th.ReduceMotion || c.Preferences().ReduceMotion {
							anim = ui.Animation{}
						}
						a.mediaScroll.ScrollTo(0, 0, anim)
						a.dirty = true
					},
					Label: "Mini player",
				})
			})
	})
}

func (a *appState) mediaHeader(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme, class md3e.WindowSizeClass) {
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	vp := a.media.activeVideoPlayer()
	videoPlaying := vp != nil && vp.Playing()

	statusText, statusTone, pulsing := "Studio Ready", md3e.ToneNeutral, false
	switch {
	case videoPlaying:
		statusText, statusTone, pulsing = "Video Playing", md3e.ToneTertiary, !reduce
	case a.media.audioPlaying:
		statusText, statusTone, pulsing = "Audio Playing", md3e.ToneSuccess, !reduce
	}

	ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Wrap().Children(func() {
		ui.Column(c).MinWidth(0).Gap(2).Children(func() {
			md3e.Text(c, "Media & Gallery Studio", tokens.HeadlineSmall, true)
			md3e.Text(c, "Dynamic cover color (SeedFromImage), expressive polygon gallery, lightbox, and playback", tokens.BodySmall).
				TextColor(cs.OnSurfaceVariant)
		})

		ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
			md3e.StatusPill(c, statusText, md3e.StatusPillOptions{
				Tone:    statusTone,
				Pulsing: pulsing,
			})
			md3e.FilterChip(c.Key("follow-cover-chip"), &a.media.followCoverTheme, "Follow cover", md3e.FilterChipOptions{
				LeadingIcon: symbols.Palette.Draw,
				OnChanged: func(bool) {
					a.dirty = true
				},
			})

			queueVisible := a.media.showQueue
			if class == md3e.WindowCompact {
				queueVisible = a.media.compactQueue
			}
			queueLabel := fmt.Sprintf("Queue (%d)", len(a.media.artworks))
			md3e.FilterChip(c.Key("toggle-queue-chip"), &queueVisible, queueLabel, md3e.FilterChipOptions{
				LeadingIcon: symbols.QueueMusic.Draw,
				OnChanged: func(next bool) {
					if class == md3e.WindowCompact {
						a.media.compactQueue = next
					} else {
						a.media.showQueue = next
					}
					a.dirty = true
				},
			})
		})
	})
}

func (a *appState) mediaNowPlayingSection(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme) {
	activeArt := a.media.activeArtwork()
	inspected := a.media.inspectedArtwork()
	activeSeed := a.media.activeCoverSeed()

	ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Stretch).Gap(14).Wrap().Children(func() {
		ui.Box(c.Key("audio-card-slot")).Grow(1).MinWidth(320).Children(func() {
			media.AudioPlayerCard(c.Key("workbench-audio-card"), a.media.currentTrackInfo(), media.AudioPlayerOptions{
				Playing:         &a.media.audioPlaying,
				Shuffle:         &a.media.shuffle,
				Repeat:          &a.media.repeat,
				DynamicColor:    true,
				ColorVariant:    variantPresets[a.variantIdx].variant,
				HasColorVariant: true,
				ArtworkShape:    activeArt.shape,
				OnPlayPause: func(bool) {
					a.dirty = true
				},
				OnSeek: func(pos time.Duration) {
					a.media.audioPos = pos
					a.dirty = true
				},
				OnPrevious: func() {
					a.media.prevTrack()
					a.dirty = true
				},
				OnNext: func() {
					a.media.nextTrack()
					a.dirty = true
				},
				OnShuffle: func(bool) {
					a.dirty = true
				},
				OnRepeat: func(bool) {
					a.dirty = true
				},
				OnQueue: func() {
					if md3e.ContentSizeClassOf(c) == md3e.WindowCompact {
						a.media.compactQueue = !a.media.compactQueue
					} else {
						a.media.showQueue = !a.media.showQueue
					}
					a.dirty = true
				},
				Label: "Audio player",
			})
		})

		ui.Box(c.Key("cover-palette-slot")).Grow(1).MinWidth(240).Children(func() {
			md3e.Surface(c, md3e.SurfaceOptions{
				Color:      cs.SurfaceContainerLow,
				ShapeScale: tokens.ShapeExtraLarge,
				Hairline:   true,
			}).Fill().Padding(16).Column().Gap(12).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(8).MinWidth(0).Children(func() {
						md3e.IconBadge(c, symbols.Palette.Draw, md3e.IconBadgeOptions{
							Variant: md3e.IconBadgePrimaryTonal,
							Size:    tokens.SizeXS,
							Shape:   shape.Cookie4Sided(),
							Label:   "Cover palette",
						})
						md3e.Text(c, "Dynamic Cover Seed", tokens.TitleSmall, true)
					})
					md3e.Tag(c, "SeedFromImage", md3e.TagOptions{
						Tone:    md3e.TonePrimary,
						Variant: md3e.TagOutlined,
					})
				})

				ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(12).Children(func() {
					md3e.Image(c.Key("inspected-cover-"+inspected.id), inspected.bitmap, md3e.ImageOptions{
						Width:  54,
						Height: 54,
						Fit:    ui.Cover,
						Shape:  inspected.shape,
						NoFade: true,
						Label:  inspected.title + " inspected artwork",
					})
					ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
						md3e.TextWith(c, mediaColorHex(activeSeed), md3e.TextOptions{
							Role:        tokens.TitleMedium,
							Emphasized:  true,
							Mono:        true,
							TabularNums: true,
							Color:       cs.OnSurface,
						})
						md3e.TextWith(c, "Extracted from "+inspected.title+" ("+inspected.shapeName+")", md3e.TextOptions{
							Role:     tokens.BodySmall,
							Color:    cs.OnSurfaceVariant,
							MaxLines: 1,
						})
					})
				})

				ui.Column(c).FillWidth().Gap(6).Children(func() {
					md3e.Text(c, "Scored palette (PaletteFromImage)", tokens.LabelMedium).TextColor(cs.OnSurfaceVariant)
					ui.Row(c).FillWidth().Gap(6).Wrap().Children(func() {
						for i, swCol := range inspected.palette {
							hex := mediaColorHex(swCol)
							selected := a.media.followCoverTheme && activeSeed == swCol
							bg := cs.SurfaceContainerHigh
							borderCol := cs.OutlineVariant
							borderW := float32(1)
							if selected {
								bg = cs.SecondaryContainer
								borderCol = cs.Primary
								borderW = 1.5
							}
							btn := ui.ButtonBase(c.Key("palette-swatch-"+strconv.Itoa(i))).
								Radius(th.Shapes.Radius(tokens.ShapeFull)).
								Background(bg).
								Border(borderW, borderCol).
								Padding(5, 10, 5, 6).
								Row().
								AlignItems(ui.Center).
								Gap(6).
								FocusRing(false).
								Label("Apply cover palette " + hex)
							md3e.AttachRipple(c, btn, md3e.StateLayerOptions{
								Color:       cs.OnSurface,
								Radius:      th.Shapes.Radius(tokens.ShapeFull),
								BorderInset: borderW,
							})
							btn.Children(func() {
								ui.Box(c).Size(14, 14).Radius(7).Background(swCol).Border(1, cs.Outline.Alpha(0.35))
								md3e.TextWith(c, hex, md3e.TextOptions{
									Role:        tokens.LabelSmall,
									Mono:        true,
									TabularNums: true,
									Color:       cs.OnSurface,
								})
							})
							if btn.Clicked() {
								a.media.customPaletteSeed = swCol
								a.media.followCoverTheme = true
								a.dirty = true
								a.snack.Show(md3e.SnackbarOptions{
									Message: "Workbench theme seeded from " + hex,
								})
							}
						}
					})
				})

				ui.Box(c).Grow(1)

				ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Wrap().Children(func() {
					if md3e.Switch(c.Key("sync-cover-theme"), &a.media.followCoverTheme, md3e.SwitchOptions{
						Label: "Sync workbench theme",
					}).Changed() {
						a.dirty = true
					}
					if a.media.customPaletteSeed.A > 0 || a.media.selectedArtIdx != a.media.activeTrack {
						if md3e.Button(c.Key("sync-to-track"), "Match playing", md3e.ButtonOptions{
							Variant: md3e.ButtonText,
							Size:    tokens.SizeXS,
						}).Clicked() {
							a.media.selectedArtIdx = a.media.activeTrack
							a.media.customPaletteSeed = ui.Color{}
							a.dirty = true
						}
					}
				})
			})
		})
	})
}

func (a *appState) mediaVideoSection(c *ui.Context, cs tokens.ColorScheme) {
	if len(a.media.clips) == 0 {
		return
	}
	clipIdx := clampInt(a.media.activeClip, 0, len(a.media.clips)-1)
	clip := &a.media.clips[clipIdx]

	md3e.Surface(c, md3e.SurfaceOptions{
		Color:      cs.SurfaceContainerLow,
		ShapeScale: tokens.ShapeLargeIncreased,
		Hairline:   true,
	}).FillWidth().Padding(16).Column().Gap(12).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).Wrap().Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(10).MinWidth(0).Children(func() {
				md3e.IconBadge(c, symbols.Movie.Draw, md3e.IconBadgeOptions{
					Variant: md3e.IconBadgeTertiaryTonal,
					Size:    tokens.SizeXS,
					Shape:   shape.Cookie6Sided(),
					Label:   "Video stage",
				})
				ui.Column(c).MinWidth(0).Gap(1).Children(func() {
					md3e.Text(c, "Expressive Video Stage", tokens.TitleSmall, true)
					md3e.Text(c, clip.subtitle+" · Morph Play/Pause, wavy scrubber, speed menu & fullscreen", tokens.BodySmall).
						TextColor(cs.OnSurfaceVariant)
				})
			})

			clipTabs := make([]md3e.TabItem, len(a.media.clips))
			for i, cl := range a.media.clips {
				clipTabs[i] = md3e.TabItem{Label: cl.title}
			}
			selClip := clipIdx
			md3e.SegmentedTabs(c.Key("video-clip-tabs"), clipTabs, md3e.PillTabsOptions{
				Selected: &selClip,
				Label:    "Video clip selector",
				Changed: func(next int) {
					a.media.selectClip(next)
					a.dirty = true
				},
			})
		})

		if !a.media.videoFullscreen {
			ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
				ui.Box(c).FillWidth().MaxWidth(680).Children(func() {
					media.VideoPlayer(c.Key("workbench-video-player"), clip.player, media.VideoPlayerOptions{
						Title:         clip.title,
						Buffered:      0.78,
						AutoHideDelay: 3 * time.Second,
						Fullscreen:    &a.media.videoFullscreen,
						OnFullscreen:  func(bool) { a.dirty = true },
						Label:         "Video stage player",
					})
				})
			})
		} else {
			md3e.Surface(c, md3e.SurfaceOptions{
				Color:      cs.SurfaceContainerHigh,
				ShapeScale: tokens.ShapeLarge,
				Hairline:   true,
			}).FillWidth().Padding(18).Row().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Children(func() {
				md3e.Text(c, clip.title+" is playing in fullscreen theater overlay", tokens.BodyMedium).
					TextColor(cs.OnSurfaceVariant)
				if md3e.Button(c.Key("inline-exit-fs"), "Restore inline", md3e.ButtonOptions{
					Variant:     md3e.ButtonTonal,
					Size:        tokens.SizeXS,
					LeadingIcon: symbols.FullscreenExit.Draw,
				}).Clicked() {
					a.media.videoFullscreen = false
					a.dirty = true
				}
			})
		}
	})
}

func (a *appState) videoTheaterOverlay(c *ui.Context, th *md3e.Theme) {
	if len(a.media.clips) == 0 {
		return
	}
	clip := &a.media.clips[clampInt(a.media.activeClip, 0, len(a.media.clips)-1)]
	scrim := th.Colors.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}

	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Background(scrim.Alpha(0.92)).
			Modal().
			Center().
			Padding(24)
		if back.OverlayShortcut(0, ui.KeyEscape) {
			a.media.videoFullscreen = false
			a.dirty = true
		}
		back.Children(func() {
			panel := ui.Column(c).FillWidth().MaxWidth(960).Gap(10)
			panel.Clicked() // absorb clicks inside theater container
			panel.Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
					md3e.TextWith(c, "Theater — "+clip.title, md3e.TextOptions{
						Role:       tokens.TitleMedium,
						Emphasized: true,
						Color:      ui.RGB(255, 255, 255),
					})
					if md3e.Button(c.Key("theater-close"), "Close theater", md3e.ButtonOptions{
						Variant:     md3e.ButtonTonal,
						Size:        tokens.SizeXS,
						LeadingIcon: symbols.Close.Draw,
					}).Clicked() {
						a.media.videoFullscreen = false
						a.dirty = true
					}
				})
				media.VideoPlayer(c.Key("workbench-video-player-theater"), clip.player, media.VideoPlayerOptions{
					Title:         clip.title,
					Buffered:      0.78,
					AutoHideDelay: 3 * time.Second,
					Fullscreen:    &a.media.videoFullscreen,
					OnFullscreen:  func(bool) { a.dirty = true },
					Label:         "Video theater player",
				})
			})
		})
	})
}

func (a *appState) mediaGallerySection(c *ui.Context, cs tokens.ColorScheme) {
	visible := a.media.filteredGalleryIndices()

	md3e.Surface(c, md3e.SurfaceOptions{
		Color:      cs.SurfaceContainerLow,
		ShapeScale: tokens.ShapeLargeIncreased,
		Hairline:   true,
	}).FillWidth().Padding(16).Column().Gap(14).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).Wrap().Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(10).MinWidth(0).Children(func() {
				md3e.IconBadge(c, symbols.Photo.Draw, md3e.IconBadgeOptions{
					Variant: md3e.IconBadgeTonal,
					Size:    tokens.SizeXS,
					Shape:   shape.Sunny(),
					Label:   "Artwork gallery",
				})
				ui.Column(c).MinWidth(0).Gap(1).Children(func() {
					md3e.Text(c, "Expressive Artwork Gallery", tokens.TitleSmall, true)
					md3e.Text(c, "Polygon-masked md3e.Image tiles · click to open ImageLightbox with MatchKey", tokens.BodySmall).
						TextColor(cs.OnSurfaceVariant)
				})
			})

			ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
				md3e.Tag(c, fmt.Sprintf("%d items", len(visible)), md3e.TagOptions{
					Tone:    md3e.ToneSecondary,
					Variant: md3e.TagOutlined,
				})
				md3e.SegmentedTabs(c.Key("gallery-mask-mode"), []md3e.TabItem{
					{Label: "Expressive"},
					{Label: "Rounded"},
					{Label: "Mono"},
				}, md3e.PillTabsOptions{
					Selected: &a.media.galleryMaskMode,
					Label:    "Gallery mask style",
				})
				md3e.SegmentedTabs(c.Key("gallery-tone-filter"), []md3e.TabItem{
					{Label: "All"},
					{Label: "Warm"},
					{Label: "Cool"},
				}, md3e.PillTabsOptions{
					Selected: &a.media.galleryToneFilter,
					Label:    "Gallery tone filter",
				})
				md3e.FilterChip(c.Key("gallery-placeholders"), &a.media.previewPlaceholders, "States", md3e.FilterChipOptions{
					LeadingIcon: symbols.HourglassEmpty.Draw,
					OnChanged: func(bool) {
						a.media.placeholderErrFixed = false
						a.dirty = true
					},
				})
			})
		})

		md3e.BentoGrid(c, func() {
			for pos, idx := range visible {
				art := &a.media.artworks[idx]
				isInspected := idx == a.media.selectedArtIdx
				tileBg := cs.SurfaceContainer
				if isInspected {
					tileBg = cs.SurfaceContainerHigh
				}

				var maskPoly *shape.RoundedPolygon
				var shapeScale tokens.ShapeScale
				grayscale := false
				switch a.media.galleryMaskMode {
				case maskModeRounded:
					shapeScale = tokens.ShapeExtraLarge
				case maskModeMono:
					shapeScale = tokens.ShapeLarge
					grayscale = true
				default:
					maskPoly = art.shape
				}

				var imgSrc ui.ImageSource = art.bitmap
				simLoading := false
				var simErr error
				var onRetry func()
				if a.media.previewPlaceholders {
					switch pos {
					case 0:
						imgSrc = nil
						simLoading = true
					case 1:
						if !a.media.placeholderErrFixed {
							imgSrc = nil
							simErr = errSimulatedArtLoad
							onRetry = func() {
								a.media.placeholderErrFixed = true
								a.dirty = true
								a.snack.Show(md3e.SnackbarOptions{Message: "Recovered " + art.title})
							}
						}
					}
				}

				md3e.Surface(c.Key("gallery-card-"+art.id), md3e.SurfaceOptions{
					Color:      tileBg,
					ShapeScale: tokens.ShapeLarge,
					Hairline:   true,
				}).Padding(10).Column().Gap(10).Children(func() {
					artIdx := idx
					md3e.Image(c.Key("gallery-img-"+art.id), imgSrc, md3e.ImageOptions{
						AspectRatio: 1,
						Fit:         ui.Cover,
						Shape:       maskPoly,
						ShapeScale:  shapeScale,
						Grayscale:   grayscale,
						FadeIn:      true,
						Loading:     simLoading,
						Error:       simErr,
						OnRetry:     onRetry,
						MatchKey:    art.matchKey,
						BorderWidth: 1.5,
						OnClick: func() {
							a.media.openLightbox(artIdx)
							a.dirty = true
						},
						Label: "Open " + art.title,
					}).FillWidth()

					ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(8).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							md3e.TextWith(c, art.title, md3e.TextOptions{
								Role:       tokens.LabelLarge,
								Emphasized: true,
								MaxLines:   1,
								Color:      cs.OnSurface,
							})
							ui.Row(c).AlignItems(ui.Center).Gap(5).MinWidth(0).Children(func() {
								ui.Box(c).Size(9, 9).Radius(4.5).Background(art.seed).Border(1, cs.Outline.Alpha(0.4))
								md3e.TextWith(c, art.shapeName+" · "+art.seedHex, md3e.TextOptions{
									Role:        tokens.BodySmall,
									Mono:        true,
									TabularNums: true,
									MaxLines:    1,
									Color:       cs.OnSurfaceVariant,
								})
							})
						})

						md3e.ShapeButton(c.Key("gallery-seed-"+art.id), "Use seed "+art.title, symbols.Palette.Draw, md3e.ShapeButtonOptions{
							Variant: md3e.ShapeButtonTonal,
							Size:    tokens.SizeXS,
							Shape:   art.shape,
							OnClick: func() {
								a.media.selectedArtIdx = artIdx
								a.media.customPaletteSeed = ui.Color{}
								a.media.followCoverTheme = true
								a.dirty = true
								a.snack.Show(md3e.SnackbarOptions{
									Message: fmt.Sprintf("Theme seeded from %s (%s)", art.title, art.seedHex),
								})
							},
						})
					})
				})
			}
		}, md3e.BentoGridOptions{
			CompactColumns:  2,
			MediumColumns:   3,
			ExpandedColumns: 3,
			Label:           "Expressive artwork grid",
		})
	})
}

func (a *appState) mediaQueuePane(c *ui.Context, th *md3e.Theme, cs tokens.ColorScheme, class md3e.WindowSizeClass) {
	filtered := a.media.filteredTrackIndices()
	var totalDur time.Duration
	for i := range a.media.artworks {
		totalDur += a.media.artworks[i].duration
	}

	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(cs.SurfaceContainerLow).Padding(14).Gap(10).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).MinWidth(0).Children(func() {
				if class == md3e.WindowCompact {
					if md3e.IconButton(c.Key("queue-back"), "Back to studio", symbols.ArrowBack.Draw, md3e.IconButtonOptions{
						Variant: md3e.IconButtonStandard,
						Size:    tokens.SizeXS,
					}).Clicked() {
						a.media.compactQueue = false
						a.dirty = true
					}
				}
				md3e.IconBadge(c, symbols.QueueMusic.Draw, md3e.IconBadgeOptions{
					Variant: md3e.IconBadgePrimaryTonal,
					Size:    tokens.SizeXS,
					Shape:   shape.Cookie6Sided(),
					Label:   "Play queue",
				})
				md3e.Text(c, "Play Queue", tokens.TitleSmall, true)
			})

			ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
				if md3e.IconButton(c.Key("queue-move-up"), "Move track up", symbols.ArrowUpward.Draw, md3e.IconButtonOptions{
					Variant:  md3e.IconButtonStandard,
					Size:     tokens.SizeXS,
					Disabled: a.media.activeTrack <= 0,
				}).Clicked() {
					a.media.moveTrackInQueue(a.media.activeTrack, -1)
					a.dirty = true
				}
				if md3e.IconButton(c.Key("queue-move-down"), "Move track down", symbols.ArrowDownward.Draw, md3e.IconButtonOptions{
					Variant:  md3e.IconButtonStandard,
					Size:     tokens.SizeXS,
					Disabled: a.media.activeTrack >= len(a.media.artworks)-1,
				}).Clicked() {
					a.media.moveTrackInQueue(a.media.activeTrack, 1)
					a.dirty = true
				}
			})
		})

		md3e.TextField(c.Key("queue-search"), &a.media.queueFilter, md3e.TextFieldOptions{
			Variant:       md3e.TextFieldOutlined,
			Label:         "Filter queue",
			Placeholder:   "Track, artist, album...",
			LeadingIcon:   symbols.Search.Draw,
			TrailingIcon:  symbols.Close.Draw,
			TrailingLabel: "Clear queue filter",
			OnTrailing: func() {
				a.media.queueFilter = ""
				a.dirty = true
			},
			OnChanged: func(string) {
				a.dirty = true
			},
		})

		ui.Scroll(c.Key("queue-scroll")).Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().PaddingEnd(12).MinWidth(0).Children(func() {
				if len(filtered) == 0 {
					md3e.EmptyState(c, "No matching tracks", md3e.EmptyStateOptions{
						Icon:               symbols.MusicNote.Draw,
						IconShape:          shape.Cookie7Sided(),
						Description:        fmt.Sprintf("No tracks match %q.", a.media.queueFilter),
						PrimaryActionLabel: "Clear filter",
						OnPrimaryAction: func() {
							a.media.queueFilter = ""
							a.dirty = true
						},
						Label: "No matching tracks",
					})
					return
				}

				activeVis := -1
				items := make([]md3e.ListItem, len(filtered))
				for visIdx, realIdx := range filtered {
					art := &a.media.artworks[realIdx]
					if realIdx == a.media.activeTrack {
						activeVis = visIdx
					}
					isCurrent := realIdx == a.media.activeTrack
					items[visIdx] = md3e.ListItem{
						Key:            art.id,
						Headline:       art.title,
						SupportingText: art.artist + " · " + art.album,
						Meta:           formatMediaDuration(art.duration),
						Leading: func() ui.Element {
							return md3e.Image(c.Key("queue-thumb-"+art.id), art.bitmap, md3e.ImageOptions{
								Width:  36,
								Height: 36,
								Fit:    ui.Cover,
								Shape:  art.shape,
								NoFade: true,
							})
						},
						Trailing: func() ui.Element {
							return ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
								if isCurrent {
									eqCol := cs.Primary
									ui.Box(c).Size(16, 16).Draw(func(p *ui.Painter, r ui.Rect) {
										symbols.Equalizer.Draw(p, r, eqCol)
									})
								}
								ui.Box(c).Size(10, 10).Radius(5).Background(art.seed).Border(1, cs.Outline.Alpha(0.4))
							})
						},
					}
				}

				md3e.List(c.Key("queue-list"), items, md3e.ListOptions{
					Label:     "Play queue",
					Segmented: true,
					Selected:  &activeVis,
					OnSelect: func(visIdx int) {
						if visIdx >= 0 && visIdx < len(filtered) {
							a.media.selectTrack(filtered[visIdx])
							a.dirty = true
						}
					},
				})
			})
		})

		md3e.Divider(c)
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
			md3e.TextWith(c, fmt.Sprintf("%d/%d tracks · %s", len(filtered), len(a.media.artworks), formatMediaDuration(totalDur)), md3e.TextOptions{
				Role:        tokens.LabelMedium,
				Mono:        true,
				TabularNums: true,
				Color:       cs.OnSurfaceVariant,
			})
			md3e.Tag(c, densityNames[a.densityIdx], md3e.TagOptions{
				Tone:    md3e.ToneSecondary,
				Variant: md3e.TagOutlined,
			})
		})
	})
}
