// Command workbench is the Phase 2 flagship Material Design 3 Expressive
// multi-page application built over WindowTitleBar, AdaptiveScaffold,
// AnimatedThemeScope, PageTransition, charts, media, docview, and virtualized
// DataTable.
//
// M15.3.a delivers the adaptive shell and Page 1 ("Monitor"): a warm-orange
// dark-first Bento telemetry dashboard with a cookie MorphIconButton, live
// upload/download AreaSparklines, real-time process memory self-inspection via
// runtime.MemStats + StackedProgress, and a 10K/100K virtualized Connections
// & DNS Queries DataTable.
//
// M15.3.b delivers Page 2 ("Media"): cover-extracted Dynamic Color via
// tokens.SeedFromImage / PaletteFromImage, an expressive polygon-masked artwork
// gallery with shared-element ImageLightbox preview, Expressive AudioPlayerCard,
// MiniPlayer, and VideoPlayer stage, paired with a SupportingPane play queue.
//
// M15.3.c delivers Page 3 ("Docs"): a responsive three-pane Knowledge Workbench
// pairing a filterable multi-level TreeView explorer (ListDetailPane) with a
// Breadcrumb + docview.Markdown / MarkdownStream document reader (Callout,
// CodeBlock, interactive tasks, doc:// links) and a scroll-linked docview.Outline
// (SupportingPane).
//
// Run from the repository root:
//
//	CGO_ENABLED=0 go run ./examples/workbench
package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

type seedPreset struct {
	name  string
	color ui.Color
}

var seedPresets = []seedPreset{
	{name: "Amber", color: ui.Hex("#E06D1B")},
	{name: "Nebula", color: ui.Hex("#6750A4")},
	{name: "Harbor", color: ui.Hex("#0061A4")},
	{name: "Emerald", color: ui.Hex("#2E6B3E")},
}

type variantPreset struct {
	label   string
	variant tokens.Variant
}

var variantPresets = []variantPreset{
	{label: "Expressive", variant: tokens.Expressive},
	{label: "TonalSpot", variant: tokens.TonalSpot},
	{label: "Vibrant", variant: tokens.Vibrant},
	{label: "Mono", variant: tokens.Monochrome},
}

var densityNames = []string{"Comfort", "Desktop", "Compact"}

var surfaceStyleNames = []string{"Tonal", "Neutral"}

type themeSignature struct {
	seed            ui.Color
	variantIdx      int
	surfaceStyleIdx int
	dark            bool
	densityIdx      int
	reduce          bool
}

type appState struct {
	win           *mygo.Window
	hasSyncedWin  bool
	syncedWinDark bool
	syncedWinBg   ui.Color

	page            int
	seedIdx         int
	variantIdx      int
	surfaceStyleIdx int
	dark            bool
	densityIdx      int
	pinned          bool
	reduce          bool
	rtl             bool
	prefsOpen       bool

	theme *md3e.Theme
	sig   themeSignature
	dirty bool

	snack       *md3e.SnackbarState
	dests       []md3e.NavigationDestination
	mon         *monitorState
	media       *mediaState
	mediaScroll ui.ScrollState
	docs        *docsState
}

func newApp() *appState {
	return &appState{
		seedIdx:    0, // Warm orange default
		variantIdx: 0, // Expressive
		dark:       true,
		densityIdx: 1, // DensityDesktop
		snack:      &md3e.SnackbarState{},
		dests: []md3e.NavigationDestination{
			{Label: "Monitor", Icon: symbols.Dashboard.Draw},
			{Label: "Media", Icon: symbols.PlayCircle.Draw},
			{Label: "Docs", Icon: symbols.Article.Draw},
		},
		mon:   newMonitorState(),
		media: newMediaState(),
		docs:  newDocsState(),
	}
}

func (a *appState) density() tokens.Density {
	switch a.densityIdx {
	case 1:
		return tokens.DensityDesktop
	case 2:
		return tokens.DensityCompact
	default:
		return tokens.DensityComfortable
	}
}

func (a *appState) activeSeed() ui.Color {
	if a.page == 1 && a.media != nil && a.media.followCoverTheme {
		if s := a.media.activeCoverSeed(); s.A > 0 {
			return s
		}
	}
	return seedPresets[a.seedIdx].color
}

func (a *appState) surfaceStyle() tokens.SurfaceStyle {
	if a.surfaceStyleIdx == 1 {
		return tokens.SurfaceNeutral
	}
	return tokens.SurfaceTonal
}

func (a *appState) currentTheme() *md3e.Theme {
	seed := a.activeSeed()
	sig := themeSignature{
		seed:            seed,
		variantIdx:      a.variantIdx,
		surfaceStyleIdx: a.surfaceStyleIdx,
		dark:            a.dark,
		densityIdx:      a.densityIdx,
		reduce:          a.reduce,
	}
	if a.theme != nil && a.sig == sig {
		return a.theme
	}
	a.theme = md3e.ThemeFromSeed(seed, md3e.ThemeOptions{
		Dark:         a.dark,
		Variant:      variantPresets[a.variantIdx].variant,
		SurfaceStyle: a.surfaceStyle(),
		Density:      a.density(),
		Motion:       motion.Expressive,
		ReduceMotion: a.reduce,
	})
	a.sig = sig
	return a.theme
}

func (a *appState) view(c *ui.Context) {
	if a.dirty {
		a.dirty = false
		c.Invalidate()
	}
	now := c.Now()
	if a.page == 0 {
		a.mon.advanceIfDue(now)
	} else {
		a.mon.lastTick = time.Time{}
	}
	if a.page == 1 && a.media != nil {
		a.media.advanceIfDue(now)
	} else if a.media != nil {
		a.media.pauseVideo()
		a.media.lightboxOpen = false
		a.media.videoFullscreen = false
		a.media.lastAudioTick = time.Time{}
	}
	if a.page == 2 && a.docs != nil {
		a.docs.advanceIfDue(now)
	} else if a.docs != nil {
		a.docs.pauseStream()
	}

	th := a.currentTheme()
	a.syncWindowTheme(th)

	dir := ui.LeftToRight
	if a.rtl {
		dir = ui.RightToLeft
	}
	ui.Box(c).Fill().Direction(dir).Children(func() {
		md3e.AnimatedThemeScope(c, th, func() {
			md3e.AdaptiveScaffold(c, a.dests, func() {
				md3e.PageTransition(c.Key("page-transition"), a.page, func() {
					a.pageView(c)
				}, md3e.TransitionOptions{
					Mode: md3e.TransitionSharedAxisY,
				}).Fill()
			}, md3e.AdaptiveScaffoldOptions{
				Selected:        &a.page,
				Header:          func() ui.Element { return a.titleBar(c) },
				NavigationLabel: "Workbench sections",
			})
			a.preferencesDialog(c)
			md3e.SnackbarHost(c, a.snack)
		}).Fill()
	})

	if a.page == 0 {
		a.mon.scheduleNext(c, c.Now())
	} else if a.page == 1 && a.media != nil {
		a.media.scheduleNext(c, c.Now())
	} else if a.page == 2 && a.docs != nil {
		a.docs.scheduleNext(c, c.Now())
	}
}

func (a *appState) syncWindowTheme(th *md3e.Theme) {
	if a.win == nil || a.win.IsDestroyed() || th == nil {
		return
	}
	bg := th.Colors.Surface
	if !a.hasSyncedWin || a.syncedWinBg != bg {
		_ = a.win.SetBackgroundColor(formatHexRGB(bg))
		a.syncedWinBg = bg
	}
	if !a.hasSyncedWin || a.syncedWinDark != th.Dark {
		src := mygo.ThemeLight
		if th.Dark {
			src = mygo.ThemeDark
		}
		mygo.Theme.SetSource(src)
		a.syncedWinDark = th.Dark
	}
	a.hasSyncedWin = true
}

func formatHexRGB(c ui.Color) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func (a *appState) titleBar(c *ui.Context) ui.Element {
	th := md3e.CurrentTheme(c)
	var win md3e.WindowController
	if a.win != nil {
		win = a.win
	}
	showControls := win != nil && (runtime.GOOS != "darwin" || c.TitleBar().Height <= 0)

	pageTitle := "MyGo MD3E Workbench — Monitor"
	switch a.page {
	case 1:
		pageTitle = "MyGo MD3E Workbench — Media"
	case 2:
		pageTitle = "MyGo MD3E Workbench — Docs"
	}

	return md3e.WindowTitleBar(c, md3e.WindowTitleBarOptions{
		Title:              pageTitle,
		Divider:            true,
		Window:             win,
		ShowWindowControls: showControls,
		Pinned:             &a.pinned,
		OnPin:              func(bool) { a.dirty = true },
		Leading: func() ui.Element {
			return ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				md3e.IconBadge(c, symbols.Insights.Draw, md3e.IconBadgeOptions{
					Variant: md3e.IconBadgeFilled,
					Size:    tokens.SizeXS,
					Shape:   shape.Cookie9Sided(),
					Label:   "Workbench",
				})
				md3e.Text(c, "Workbench", tokens.TitleSmall, true).TextColor(th.Colors.OnSurface)
			})
		},
		Trailing: func() ui.Element { return a.chrome(c) },
	})
}

func (a *appState) chrome(c *ui.Context) ui.Element {
	class := md3e.WindowSizeClassOf(c)
	return ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
		seedLabel := "Seed color: " + seedPresets[a.seedIdx].name
		if a.page == 1 && a.media != nil && a.media.followCoverTheme {
			seedLabel = "Seed color: Cover " + mediaColorHex(a.activeSeed())
		}
		md3e.ShapeButton(c.Key("seed-cycle"), seedLabel, symbols.Palette.Draw, md3e.ShapeButtonOptions{
			Variant: md3e.ShapeButtonTonal,
			Size:    tokens.SizeXS,
			Shape:   shape.Cookie4Sided(),
			OnClick: func() {
				if a.page == 1 && a.media != nil && a.media.followCoverTheme {
					a.media.followCoverTheme = false
				}
				a.seedIdx = (a.seedIdx + 1) % len(seedPresets)
				a.dirty = true
			},
		})
		md3e.MorphIconButton(c.Key("dark-mode"), &a.dark, "Dark mode", symbols.DarkMode.Draw, md3e.MorphIconButtonOptions{
			Variant:       md3e.ShapeButtonTonal,
			Size:          tokens.SizeXS,
			Shape:         shape.Circle(),
			SelectedShape: shape.Sunny(),
			SelectedIcon:  symbols.LightMode.Draw,
			OnChanged:     func(bool) { a.dirty = true },
		})
		if class == md3e.WindowCompact {
			md3e.ShapeButton(c.Key("density-cycle"), "Density: "+densityNames[a.densityIdx], symbols.ViewList.Draw, md3e.ShapeButtonOptions{
				Variant: md3e.ShapeButtonStandard,
				Size:    tokens.SizeXS,
				Shape:   shape.Square(),
				OnClick: func() {
					a.densityIdx = (a.densityIdx + 1) % len(densityNames)
					a.dirty = true
				},
			})
		} else {
			md3e.SegmentedTabs(c.Key("density-tabs"), []md3e.TabItem{
				{Label: densityNames[0]},
				{Label: densityNames[1]},
				{Label: densityNames[2]},
			}, md3e.PillTabsOptions{
				Selected: &a.densityIdx,
				Changed:  func(int) { a.dirty = true },
				Label:    "Density",
			})
		}
		if class == md3e.WindowExpanded {
			md3e.MorphIconButton(c.Key("title-reduce"), &a.reduce, "Reduce motion", symbols.Speed.Draw, md3e.MorphIconButtonOptions{
				Variant:       md3e.ShapeButtonTonal,
				Size:          tokens.SizeXS,
				Shape:         shape.Circle(),
				SelectedShape: shape.SoftBurst(),
				SelectedIcon:  symbols.Speed.Draw,
				OnChanged:     func(bool) { a.dirty = true },
			})
			md3e.MorphIconButton(c.Key("title-rtl"), &a.rtl, "RTL", symbols.SwapHoriz.Draw, md3e.MorphIconButtonOptions{
				Variant:       md3e.ShapeButtonTonal,
				Size:          tokens.SizeXS,
				Shape:         shape.Circle(),
				SelectedShape: shape.Square(),
				SelectedIcon:  symbols.SwapHoriz.Draw,
				OnChanged:     func(bool) { a.dirty = true },
			})
		}
		md3e.ShapeButton(c.Key("open-prefs"), "Preferences", symbols.Tune.Draw, md3e.ShapeButtonOptions{
			Variant:       md3e.ShapeButtonStandard,
			Size:          tokens.SizeXS,
			Shape:         shape.Circle(),
			Selected:      a.prefsOpen,
			SelectedShape: shape.Square(),
			OnClick: func() {
				a.prefsOpen = true
				a.dirty = true
			},
		})
	})
}

func (a *appState) preferencesDialog(c *ui.Context) {
	md3e.AlertDialog(c.Key("prefs-dialog"), &a.prefsOpen, md3e.AlertDialogOptions{
		Icon:           symbols.Tune.Draw,
		Headline:       "Workbench preferences",
		SupportingText: "Adjust MD3E theme tokens, density, motion, and layout direction.",
		DismissOnScrim: true,
		Width:          440,
		Content: func() ui.Element {
			th := md3e.CurrentTheme(c)
			cs := th.Colors
			return ui.Column(c).FillWidth().Gap(10).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Wrap().Children(func() {
					md3e.Text(c, "Seed color", tokens.LabelLarge).TextColor(cs.OnSurface)
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						for i, p := range seedPresets {
							border := cs.OutlineVariant
							bw := float32(1)
							if i == a.seedIdx {
								border, bw = cs.OnSurface, 2
							}
							btn := ui.ButtonBase(c.Key("pref-seed-"+p.name)).
								Size(28, 28).Radius(14).Background(p.color).Border(bw, border).
								Label("Seed " + p.name)
							if btn.Clicked() {
								a.seedIdx = i
								a.dirty = true
							}
						}
					})
				})

				ui.Column(c).FillWidth().Gap(4).Children(func() {
					md3e.Text(c, "Scheme variant", tokens.LabelLarge).TextColor(cs.OnSurface)
					variantItems := make([]md3e.TabItem, len(variantPresets))
					for i, v := range variantPresets {
						variantItems[i] = md3e.TabItem{Label: v.label}
					}
					md3e.SegmentedTabs(c.Key("pref-variant"), variantItems, md3e.PillTabsOptions{
						Selected:  &a.variantIdx,
						FullWidth: true,
						Changed:   func(int) { a.dirty = true },
						Label:     "Scheme variant",
					})
				})

				ui.Column(c).FillWidth().Gap(4).Children(func() {
					md3e.Text(c, "Surface style", tokens.LabelLarge).TextColor(cs.OnSurface)
					md3e.SegmentedTabs(c.Key("pref-surface-style"), []md3e.TabItem{
						{Label: surfaceStyleNames[0]},
						{Label: surfaceStyleNames[1]},
					}, md3e.PillTabsOptions{
						Selected:  &a.surfaceStyleIdx,
						FullWidth: true,
						Changed:   func(int) { a.dirty = true },
						Label:     "Surface style",
					})
				})

				ui.Column(c).FillWidth().Gap(4).Children(func() {
					md3e.Text(c, "Density", tokens.LabelLarge).TextColor(cs.OnSurface)
					md3e.SegmentedTabs(c.Key("pref-density"), []md3e.TabItem{
						{Label: densityNames[0]},
						{Label: densityNames[1]},
						{Label: densityNames[2]},
					}, md3e.PillTabsOptions{
						Selected:  &a.densityIdx,
						FullWidth: true,
						Changed:   func(int) { a.dirty = true },
						Label:     "Density preset",
					})
				})

				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).Wrap().Children(func() {
					if md3e.Switch(c.Key("pref-reduce"), &a.reduce, md3e.SwitchOptions{
						Label: "Reduce motion",
					}).Changed() {
						a.dirty = true
					}
					if md3e.Switch(c.Key("pref-rtl"), &a.rtl, md3e.SwitchOptions{
						Label: "Right-to-left (RTL)",
					}).Changed() {
						a.dirty = true
					}
				})
			})
		},
		Confirm: &md3e.DialogAction{
			Label: "Done",
		},
	})
}

func (a *appState) pageView(c *ui.Context) {
	switch a.page {
	case 1:
		a.mediaView(c)
	case 2:
		a.docsView(c)
	default:
		a.monitorView(c)
	}
}

func main() {
	a := newApp()
	defer a.media.close()
	initTheme := a.currentTheme()
	mygo.Theme.SetSource(mygo.ThemeDark)
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:             "MyGo MD3E Workbench",
			Width:             1200,
			Height:            780,
			MinWidth:          420,
			MinHeight:         540,
			TitleBarStyle:     mygo.TitleBarHidden,
			HideWindowButtons: runtime.GOOS != "darwin",
			BackgroundColor:   formatHexRGB(initTheme.Colors.Surface),
			Content:           ui.View(a.view),
		})
		a.hasSyncedWin = true
		a.syncedWinDark = initTheme.Dark
		a.syncedWinBg = initTheme.Colors.Surface
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
