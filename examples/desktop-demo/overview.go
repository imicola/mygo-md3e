package main

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// overviewView is the Bento dashboard: a dismissible banner over a
// breakpoint-aware card grid.
func (a *appState) overviewView(c *ui.Context) {
	pageScroll(c, "overview-scroll", func() {
		pageHeading(c, "Pulse overview", "Desktop shell, density profiles and expressive micro components, live.")
		if a.bannerOpen {
			md3e.Banner(c, "PulseDesk 2.4 ships the adaptive shell, three density profiles and the 252-symbol pack.", md3e.BannerOptions{
				Tone:        md3e.TonePrimary,
				Icon:        symbols.Info.Draw,
				Title:       "Release channel · stable",
				Open:        &a.bannerOpen,
				Dismissible: true,
				Actions: []md3e.BannerAction{{
					Label: "Release notes",
					OnClick: func() {
						a.snack.Show(md3e.SnackbarOptions{Message: "Release notes opened in a scratch pane"})
					},
				}},
			})
		}
		md3e.BentoGrid(c, func() {
			a.heroCard(c)
			a.metricCard(c, "Latency", "14.2 ms", "Healthy", md3e.ToneSuccess,
				symbols.CheckCircle.Draw, 0.82, "p95 · edge fleet")
			a.metricCard(c, "Throughput", "128.4 MB/s", "Streaming", md3e.TonePrimary,
				symbols.Bolt.Draw, 0.64, "aggregate · 6 nodes")
			a.quickCard(c)
			a.actionCard(c)
		}, md3e.BentoGridOptions{Label: "PulseDesk overview grid"})
	})
}

// heroCard is the wide status card: a morphing playback control, a pulsing
// status pill, two-segment continuous stream / buffered 0->100% progress
// showcase with Wavy/Linear toggle, and an organic backdrop.
func (a *appState) heroCard(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion

	status, tone := "Paused", md3e.ToneNeutral
	if a.running {
		status, tone = "Running", md3e.ToneSuccess
	}

	var progress, buffer float64
	if a.progressMode == 1 {
		if a.running && !reduce {
			now := c.Now()
			if !a.lastTick.IsZero() {
				dt := now.Sub(a.lastTick)
				if dt > 0 {
					if dt > 100*time.Millisecond {
						dt = 100 * time.Millisecond
					}
					a.syncPhase += dt.Seconds() * 0.20
					if a.syncPhase >= 1.15 {
						a.syncPhase = 0
					}
				}
			}
			a.lastTick = now
			c.AnimationFrame()
		} else {
			a.lastTick = time.Time{}
		}
		progress = min(max(a.syncPhase, 0), 1)
		buffer = min(1.0, progress+0.18)
	} else {
		a.lastTick = time.Time{}
		if a.running {
			// Continuous two-segment indeterminate relay (no blank gap).
			progress = -1
			buffer = 0
		} else {
			progress = min(max(a.syncPhase, 0), 1)
			buffer = min(1.0, progress+0.18)
		}
	}

	fromCol := cs.PrimaryContainer.MixOklab(cs.SurfaceContainerHigh, 0.16)
	if th.Dark {
		fromCol = cs.PrimaryContainer.MixOklab(cs.SurfaceContainerHigh, 0.34)
	}
	gradient := ui.LinearGradient{
		From:  fromCol,
		To:    cs.SurfaceContainerHigh,
		Angle: 128,
		Oklab: true,
	}
	// The backdrop stays pure: soft ambient radial orbs captured from theme colors.
	orbPrimaryA := float32(0.16)
	orbTertiaryA := float32(0.12)
	if th.Dark {
		orbPrimaryA = 0.20
		orbTertiaryA = 0.15
	}
	backdrop := func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		stops1 := [3]ui.GradientStop{
			{Color: cs.Primary.Alpha(orbPrimaryA), At: 0},
			{Color: cs.Primary.Alpha(orbPrimaryA * 0.35), At: 0.48},
			{Color: cs.Primary.Alpha(0), At: 1},
		}
		p.FillRadialGradient(r, ui.RadialGradient{
			Stops:  stops1[:],
			Center: ui.Point{X: 0.84, Y: 0.14},
			Oklab:  true,
		}, 0)
		stops2 := [3]ui.GradientStop{
			{Color: cs.Tertiary.Alpha(orbTertiaryA), At: 0},
			{Color: cs.Tertiary.Alpha(orbTertiaryA * 0.35), At: 0.45},
			{Color: cs.Tertiary.Alpha(0), At: 1},
		}
		p.FillRadialGradient(r, ui.RadialGradient{
			Stops:  stops2[:],
			Center: ui.Point{X: 0.94, Y: 0.82},
			Oklab:  true,
		}, 0)
	}

	card := md3e.Card(c, func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(12).Children(func() {
				md3e.IconBadge(c, symbols.Bolt.Draw, md3e.IconBadgeOptions{
					Variant:       md3e.IconBadgeFilled,
					ContainerSize: 44,
					IconSize:      24,
					Shape:         shape.Cookie9Sided(),
					Label:         "PulseDesk engine",
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					md3e.Text(c, "PulseDesk Engine", tokens.TitleMedium, true)
					md3e.Text(c, "Adaptive workspace supervisor", tokens.BodySmall).
						TextColor(cs.OnSurfaceVariant)
				})
				md3e.MorphIconButton(c.Key("engine-playback"), &a.running, "Engine playback",
					symbols.PlayArrow.Draw, md3e.MorphIconButtonOptions{
						Variant:       md3e.ShapeButtonFilled,
						Size:          tokens.SizeS,
						Shape:         shape.Cookie9Sided(),
						SelectedShape: shape.Circle(),
						SelectedIcon:  symbols.Pause.Draw,
						OnChanged: func(bool) {
							a.lastTick = time.Time{}
							a.dirty = true
						},
					})
			})
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
				md3e.StatusPill(c, status, md3e.StatusPillOptions{Tone: tone, Pulsing: a.running})
				md3e.Tag(c, "M9 shell", md3e.TagOptions{Tone: md3e.TonePrimary, Variant: md3e.TagOutlined})
				md3e.Tag(c, "M10 morph", md3e.TagOptions{Tone: md3e.ToneTertiary, Variant: md3e.TagOutlined})
			})
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
				md3e.SegmentedTabs(c.Key("progress-mode"), []md3e.TabItem{
					{Label: "Stream"},
					{Label: "0→100%"},
				}, md3e.PillTabsOptions{
					Selected: &a.progressMode,
					Label:    "Progress mode",
					Changed: func(i int) {
						a.lastTick = time.Time{}
						if i == 1 && a.running {
							a.syncPhase = 0.08
						}
						a.dirty = true
					},
				})
				md3e.SegmentedTabs(c.Key("progress-style"), []md3e.TabItem{
					{Label: "Wavy"},
					{Label: "Linear"},
				}, md3e.PillTabsOptions{
					Selected: &a.progressStyle,
					Label:    "Progress style",
					Changed:  func(int) { a.dirty = true },
				})
			})
			progOpts := md3e.ProgressOptions{
				Label:     "Workspace sync",
				Thickness: 6,
				Buffer:    buffer,
			}
			if a.progressStyle == 1 {
				md3e.LinearProgress(c.Key("hero-progress-linear"), progress, progOpts)
			} else {
				md3e.LinearWavyProgress(c.Key("hero-progress-wavy"), progress, progOpts)
			}
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					switch {
					case progress < 0:
						caption(c, "Two-segment continuous relay · zero blank gap between cycles")
					case a.running:
						caption(c, "Continuous 0%→100% sweep with live Buffer ahead of active track")
					default:
						caption(c, "Paused determinate state · keeps buffer segment, track gap & stop dot")
					}
				})
				if progress >= 0 {
					md3e.TextWith(c, fmt.Sprintf("%.0f%% · buf %.0f%%", progress*100, buffer*100), md3e.TextOptions{
						Role:        tokens.LabelSmall,
						Mono:        true,
						TabularNums: true,
						Color:       cs.OnSurfaceVariant,
					})
				}
			})
		})
	}, md3e.CardOptions{
		Variant:        md3e.CardFilled,
		Padding:        20,
		ShapeScale:     tokens.ShapeExtraLarge,
		Hairline:       true,
		LinearGradient: &gradient,
		Backdrop:       backdrop,
	})
	md3e.BentoSpan(card, 2, 1)
}

// metricCard is one aligned numeric tile with a wavy ring gauge. The flexible
// spacer keeps the gauge and the note on the card's bottom edge when the Bento
// row stretches.
func (a *appState) metricCard(c *ui.Context, title, value, tag string, tone md3e.SemanticTone,
	icon md3e.IconDraw, progress float64, note string,
) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	md3e.Card(c, func() {
		ui.Column(c).FillWidth().Grow(1).MinWidth(0).Gap(8).Children(func() {
			md3e.Text(c, title, tokens.LabelMedium).TextColor(cs.OnSurfaceVariant)
			md3e.TextWith(c, value, md3e.TextOptions{
				Role:        tokens.TitleLarge,
				Emphasized:  true,
				Mono:        true,
				TabularNums: true,
			})
			ui.Box(c).Grow(1)
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(8).Children(func() {
				md3e.Tag(c, tag, md3e.TagOptions{Tone: tone, LeadingIcon: icon})
				ui.Box(c).Grow(1).MinWidth(0)
				md3e.CircularWavyProgress(c, progress, md3e.ProgressOptions{
					Size:      44,
					Thickness: 4,
					Label:     title + " gauge",
				})
			})
			md3e.Text(c, note, tokens.BodySmall).TextColor(cs.OnSurfaceVariant)
		})
	}, md3e.CardOptions{
		Variant:    md3e.CardOutlined,
		Padding:    16,
		ShapeScale: tokens.ShapeLargeIncreased,
	})
}

// quickCard groups two SwitchCards and shows density-aware tile metrics.
func (a *appState) quickCard(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	card := md3e.Card(c, func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				md3e.Text(c, "Quick controls", tokens.TitleSmall, true)
				ui.Box(c).Grow(1)
				md3e.Text(c, densityNames[a.densityIdx]+" density", tokens.LabelSmall).
					TextColor(cs.OnSurfaceVariant)
			})
			md3e.SwitchCard(c.Key("focus-mode"), "Focus mode", &a.focusMode, md3e.SwitchCardOptions{
				Subtitle:    "Silence non-critical alerts",
				LeadingIcon: symbols.NotificationsOff.Draw,
				IconBadge: md3e.IconBadgeOptions{
					Variant: md3e.IconBadgePrimaryTonal,
					Shape:   shape.SoftBurst(),
				},
				Hairline: true,
				OnChanged: func(on bool) {
					a.snack.Show(md3e.SnackbarOptions{Message: "Focus mode " + onOff(on)})
				},
			})
			md3e.SwitchCard(c.Key("live-sync"), "Live sync", &a.liveSync, md3e.SwitchCardOptions{
				Subtitle:    "Stream workspace deltas",
				LeadingIcon: symbols.CloudDone.Draw,
				IconBadge: md3e.IconBadgeOptions{
					Variant: md3e.IconBadgeTertiaryTonal,
					Shape:   shape.Sunny(),
				},
				Hairline: true,
				OnChanged: func(on bool) {
					a.snack.Show(md3e.SnackbarOptions{Message: "Live sync " + onOff(on)})
				},
			})
		})
	}, md3e.CardOptions{
		Variant:    md3e.CardFilled,
		Padding:    16,
		ShapeScale: tokens.ShapeExtraLarge,
		Hairline:   true,
	})
	md3e.BentoSpan(card, 2, 1)
}

// actionCard is the jump-back-in tile stack.
func (a *appState) actionCard(c *ui.Context) {
	card := md3e.Card(c, func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(6).Children(func() {
			md3e.Text(c, "Jump back in", tokens.TitleSmall, true)
			md3e.ActionTile(c.Key("jump-log"), "Activity log", md3e.ActionTileOptions{
				Subtitle:    "12 events since launch",
				LeadingIcon: symbols.History.Draw,
				IconBadge:   md3e.IconBadgeOptions{Variant: md3e.IconBadgePrimaryTonal},
				Meta:        "12",
				Variant:     md3e.CardElevated,
				HasVariant:  true,
				Hairline:    true,
				OnClick: func() {
					a.snack.Show(md3e.SnackbarOptions{Message: "Activity log opened", ActionLabel: "Undo"})
				},
			})
			md3e.ActionTile(c.Key("jump-workspace"), "Review workspace", md3e.ActionTileOptions{
				Subtitle:    "6 managed nodes",
				LeadingIcon: symbols.Dashboard.Draw,
				IconBadge:   md3e.IconBadgeOptions{Variant: md3e.IconBadgeTonal},
				Meta:        "6",
				Variant:     md3e.CardElevated,
				HasVariant:  true,
				Hairline:    true,
				OnClick:     func() { a.page = 1; a.dirty = true },
			})
			md3e.ActionTile(c.Key("jump-components"), "Component lab", md3e.ActionTileOptions{
				Subtitle:    "M10 micro components",
				LeadingIcon: symbols.Palette.Draw,
				IconBadge:   md3e.IconBadgeOptions{Variant: md3e.IconBadgeTertiaryTonal},
				Meta:        "M10",
				Variant:     md3e.CardElevated,
				HasVariant:  true,
				Hairline:    true,
				OnClick:     func() { a.page = 2; a.dirty = true },
			})
		})
	}, md3e.CardOptions{
		Variant:    md3e.CardFilled,
		Padding:    16,
		ShapeScale: tokens.ShapeExtraLarge,
		Hairline:   true,
	})
	md3e.BentoSpan(card, 2, 1)
}

// onOff renders a toggle state for snackbar copy.
func onOff(on bool) string {
	if on {
		return "enabled"
	}
	return "paused"
}
