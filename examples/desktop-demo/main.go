// Command desktop-demo is PulseDesk, a compact Material Design 3 Expressive
// desktop workbench. It shows the M9 desktop foundation (WindowTitleBar,
// density profiles, Surface texture, AdaptiveScaffold, ListDetailPane,
// SupportingPane and BentoGrid) working together with the M10 expressive layer
// (ShapeButton, MorphIconButton, IconBadge, PillTabs, SegmentedTabs, Tag,
// StatusPill, SwitchCard, ActionTile, Avatar, Skeleton, EmptyState, Banner and
// the zero-bloat symbols pack).
//
// The window starts dark with the Desktop density profile; the title bar
// switches seed color, light/dark, density and always-on-top, while the three
// pages reflow across the Compact, Medium and Expanded breakpoints.
//
// Run from the repository root:
//
//	CGO_ENABLED=0 go run ./examples/desktop-demo
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

// seedPreset is one seed color the title bar cycles through.
type seedPreset struct {
	name  string
	color ui.Color
}

// seedPresets are warm and cool starting points for the HCT palette.
var seedPresets = []seedPreset{
	{"Nebula", ui.Hex("#6750A4")},
	{"Harbor", ui.Hex("#0061A4")},
	{"Ember", ui.Hex("#8C5000")},
	{"Moss", ui.Hex("#2E6B3E")},
}

// densityNames labels the three tokens.Density profiles in order.
var densityNames = []string{"Comfort", "Desktop", "Compact"}

// themeSignature identifies a built theme so the HCT palettes are recomputed
// only when a title bar control actually changes.
type themeSignature struct {
	seedIdx    int
	dark       bool
	densityIdx int
	reduce     bool
}

// appState is PulseDesk's whole state. It outlives frames, while the views
// below rebuild every pass and never keep ui.Element or *ui.Context handles.
type appState struct {
	// win is nil under ui.NewTester; the title bar only binds it when set.
	win           *mygo.Window
	hasSyncedWin  bool
	syncedWinDark bool
	syncedWinBg   ui.Color

	// Shell controls.
	page       int
	seedIdx    int
	dark       bool
	densityIdx int
	pinned     bool
	reduce     bool

	theme *md3e.Theme
	sig   themeSignature

	// dirty asks for one more frame after a control changed state that its
	// siblings (the navigation rail, the title bar) already built this pass.
	dirty bool

	// Overview page.
	bannerOpen    bool
	running       bool
	progressMode  int // 0: Stream (two-segment relay), 1: 0->100% + Buffer
	progressStyle int // 0: LinearWavyProgress, 1: LinearProgress
	syncPhase     float64
	lastTick      time.Time
	focusMode     bool
	liveSync      bool

	// Workspace page.
	tab            int
	query          string
	skeleton       bool
	selectedID     string
	showDetail     bool
	split          float32
	intensity      float64
	showSupporting bool

	// Components page.
	tags []string

	snack *md3e.SnackbarState
	dests []md3e.NavigationDestination
}

// newApp builds the demo with its dark, Desktop-density defaults.
func newApp() *appState {
	return &appState{
		seedIdx:        0,
		dark:           true,
		densityIdx:     1,
		bannerOpen:     true,
		running:        true,
		syncPhase:      0.68,
		liveSync:       true,
		selectedID:     nodes[0].id,
		split:          280,
		intensity:      62,
		showSupporting: true,
		tags:           []string{"Shell", "Density", "Symbols"},
		snack:          &md3e.SnackbarState{},
		dests: []md3e.NavigationDestination{
			{Label: "Overview", Icon: symbols.Dashboard.Draw},
			{Label: "Workspace", Icon: symbols.ViewSidebar.Draw},
			{Label: "Components", Icon: symbols.Palette.Draw},
		},
	}
}

// density is the selected tokens.Density profile.
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

// currentTheme builds the MD3E theme the title bar describes, reusing the last
// one while its controls stay put.
func (a *appState) currentTheme() *md3e.Theme {
	sig := themeSignature{a.seedIdx, a.dark, a.densityIdx, a.reduce}
	if a.theme != nil && a.sig == sig {
		return a.theme
	}
	a.theme = md3e.ThemeFromSeed(seedPresets[a.seedIdx].color, md3e.ThemeOptions{
		Dark:         a.dark,
		Density:      a.density(),
		Motion:       motion.Expressive,
		ReduceMotion: a.reduce,
	})
	a.sig = sig
	return a.theme
}

// view builds the whole window: the title bar, the adaptive navigation shell
// and the snackbar overlay, all inside one AnimatedThemeScope so theme/seed
// switches transition smoothly and popups inherit the active theme.
func (a *appState) view(c *ui.Context) {
	if a.dirty {
		a.dirty = false
		c.Invalidate()
	}
	th := a.currentTheme()
	a.syncWindowTheme(th)
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
			NavigationLabel: "PulseDesk sections",
		})
		// Anchored after the content, so a click that shows a snackbar during
		// this pass is already visible on this frame.
		md3e.SnackbarHost(c, a.snack)
	}).Fill()
}

// syncWindowTheme keeps the native OS window background color and light/dark
// frame appearance synchronized with the active MD3E theme.
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

// titleBar is the M9 window chrome: an expressive app mark, the window title,
// and the live theme controls on the trailing side.
func (a *appState) titleBar(c *ui.Context) ui.Element {
	th := md3e.CurrentTheme(c)
	var win md3e.WindowController
	if a.win != nil {
		win = a.win
	}
	// Non-macOS windows hide native caption buttons (HideWindowButtons: true)
	// so WindowTitleBar renders MD3E rounded controls with Snap Layout support.
	// macOS keeps native traffic lights and obeys c.TitleBar() (showing custom
	// controls only when fullscreen hides the traffic lights).
	showControls := win != nil && (runtime.GOOS != "darwin" || c.TitleBar().Height <= 0)

	return md3e.WindowTitleBar(c, md3e.WindowTitleBarOptions{
		Title:              "PulseDesk — MD3E Demo",
		Divider:            true,
		Window:             win,
		ShowWindowControls: showControls,
		Pinned:             &a.pinned,
		OnPin:              func(bool) { a.dirty = true },
		Leading: func() ui.Element {
			return ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				md3e.IconBadge(c, symbols.Bolt.Draw, md3e.IconBadgeOptions{
					Variant: md3e.IconBadgeFilled,
					Size:    tokens.SizeXS,
					Shape:   shape.Cookie9Sided(),
					Label:   "PulseDesk",
				})
				md3e.Text(c, "PulseDesk", tokens.TitleSmall, true).TextColor(th.Colors.OnSurface)
			})
		},
		Trailing: func() ui.Element { return a.chrome(c) },
	})
}

// chrome holds the compact theme controls of the title bar.
func (a *appState) chrome(c *ui.Context) ui.Element {
	return ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
		preset := seedPresets[a.seedIdx]
		md3e.ShapeButton(c.Key("seed-cycle"), "Seed color: "+preset.name, symbols.Palette.Draw, md3e.ShapeButtonOptions{
			Variant: md3e.ShapeButtonTonal,
			Size:    tokens.SizeXS,
			Shape:   shape.Cookie4Sided(),
			OnClick: func() {
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
		// Below the medium breakpoint the segmented control would crowd the
		// title out, so it becomes one cycling shape button.
		if md3e.WindowSizeClassOf(c) == md3e.WindowCompact {
			md3e.ShapeButton(c.Key("density-cycle"), "Density: "+densityNames[a.densityIdx], symbols.ViewList.Draw, md3e.ShapeButtonOptions{
				Variant: md3e.ShapeButtonStandard,
				Size:    tokens.SizeXS,
				Shape:   shape.Square(),
				OnClick: func() {
					a.densityIdx = (a.densityIdx + 1) % len(densityNames)
					a.dirty = true
				},
			})
			return
		}
		md3e.SegmentedTabs(c.Key("density-tabs"), []md3e.TabItem{
			{Label: densityNames[0]},
			{Label: densityNames[1]},
			{Label: densityNames[2]},
		}, md3e.PillTabsOptions{
			Selected: &a.densityIdx,
			Changed:  func(int) { a.dirty = true },
			Label:    "Density",
		})
	})
}

// pageView builds the page the adaptive shell shows.
func (a *appState) pageView(c *ui.Context) {
	switch a.page {
	case 1:
		a.workspaceView(c)
	case 2:
		a.componentsView(c)
	default:
		a.overviewView(c)
	}
}

// pageScroll lays a scrolling page in a centered reading column, as the
// gallery does, so lines stay comfortable on wide windows.
func pageScroll(c *ui.Context, key string, body func()) {
	ui.Scroll(c.Key(key)).Fill().Children(func() {
		ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(1080).Padding(20, 20, 48, 20).Gap(16).Children(body)
		})
	})
}

// pageHeading is the title block every page starts with.
func pageHeading(c *ui.Context, title, subtitle string) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	ui.Column(c).Gap(2).Children(func() {
		md3e.Text(c, title, tokens.HeadlineSmall, true)
		md3e.Text(c, subtitle, tokens.BodySmall).TextColor(cs.OnSurfaceVariant)
	})
}

// section is a tonal panel with a title and optional supporting line.
func section(c *ui.Context, title, subtitle string, body func()) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	md3e.Surface(c, md3e.SurfaceOptions{
		Color:      cs.SurfaceContainerLow,
		ShapeScale: tokens.ShapeLargeIncreased,
		Hairline:   true,
	}).FillWidth().Padding(16).Column().Gap(12).
		Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				md3e.Text(c, title, tokens.TitleSmall, true)
				if subtitle != "" {
					md3e.Text(c, subtitle, tokens.BodySmall).TextColor(cs.OnSurfaceVariant)
				}
			})
			body()
		})
}

// caption is a muted supporting line.
func caption(c *ui.Context, text string) {
	md3e.Text(c, text, tokens.BodySmall).TextColor(md3e.CurrentTheme(c).Colors.OnSurfaceVariant)
}

func main() {
	a := newApp()
	initTheme := a.currentTheme()
	mygo.Theme.SetSource(mygo.ThemeDark)
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:             "PulseDesk — MD3E Demo",
			Width:             1120,
			Height:            740,
			MinWidth:          420,
			MinHeight:         520,
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
