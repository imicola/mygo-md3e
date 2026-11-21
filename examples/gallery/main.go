// Command gallery is the interactive showcase of every mygo-md3e component,
// from the M1 design tokens to the M8 pickers: pages of a ui.Router reachable
// from a collapsible NavigationRail, with live seed, variant, contrast, light
// and dark, motion scheme, reduce motion and right-to-left controls in the
// header, all passed down by md3e.ThemeScope so pages, dialogs, sheets, menus,
// snackbars and modal pickers share the same theme.
//
// Run from the repository root:
//
//	CGO_ENABLED=0 go run ./examples/gallery
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// seedPreset is one seed color the header offers.
type seedPreset struct {
	name  string
	color ui.Color
}

var seedPresets = []seedPreset{
	{"Purple", ui.Hex("#6750A4")},
	{"Blue", ui.Hex("#0061A4")},
	{"Green", ui.Hex("#2E6B3E")},
	{"Coral", ui.Hex("#BA1A1A")},
	{"Amber", ui.Hex("#8C5000")},
	{"Teal", ui.Hex("#006A6A")},
}

// variantNames are the seven DynamicScheme variants, indexed as tokens.Variant.
var variantNames = []string{
	"TonalSpot", "Expressive", "Vibrant", "Neutral", "Monochrome", "Fidelity", "Content",
}

// galleryRoute is one page of the gallery: its router path, rail label, icon
// and builder.
type galleryRoute struct {
	path  string
	label string
	icon  md3e.IconDraw
	build func(a *app, c *ui.Context)
}

var galleryRoutes = []galleryRoute{
	{"/theme", "Theme & Type", mdicon.Star.Draw, (*app).themePageView},
	{"/shapes", "Shapes", mdicon.ChevronRight.Draw, (*app).shapesPageView},
	{"/motion", "Motion", mdicon.Clock.Draw, (*app).motionPageView},
	{"/actions", "Actions & Bars", mdicon.Add.Draw, (*app).actionsPageView},
	{"/inputs", "Inputs", mdicon.Check.Draw, (*app).inputsPageView},
	{"/feedback", "Feedback", mdicon.MoreHoriz.Draw, (*app).feedbackPageView},
	{"/navigation", "Navigation", mdicon.ArrowForward.Draw, (*app).navigationPageView},
	{"/overlays", "Overlays", mdicon.Menu.Draw, (*app).overlaysPageView},
}

// themeSignature identifies a built theme, so the HCT palettes are computed
// only when a control changes.
type themeSignature struct {
	seed     ui.Color
	variant  tokens.Variant
	contrast tokens.Contrast
	dark     bool
	scheme   motion.Scheme
	reduce   bool
}

// app is the gallery's state: the router, the global theme controls, the
// snackbar queue and each page's own controls.
type app struct {
	router       *ui.Router
	page         int
	railExpanded bool

	seedIdx    int
	variantIdx int
	darkIdx    int
	contrast   int
	motionIdx  int
	reduce     bool
	rtl        bool

	theme *md3e.Theme
	sig   themeSignature

	snack *md3e.SnackbarState

	// scrolls keeps where each page was scrolled, by its place in
	// galleryRoutes, so the rail brings a page back where it was left.
	scrolls      []ui.ScrollState
	dests        []md3e.NavigationDestination
	variantCache []md3e.MenuItem

	shapes     shapePage
	motionPage motionPage
	actions    actionsPage
	inputs     inputsPage
	feedback   feedbackPage
	navigation navigationPage
	overlays   overlaysPage
}

func newApp() *app {
	a := &app{
		router:       ui.NewRouter(galleryRoutes[0].path),
		snack:        &md3e.SnackbarState{},
		scrolls:      make([]ui.ScrollState, len(galleryRoutes)),
		dests:        make([]md3e.NavigationDestination, 0, len(galleryRoutes)),
		variantCache: make([]md3e.MenuItem, 0, len(variantNames)),
	}
	for _, route := range galleryRoutes {
		a.dests = append(a.dests, md3e.NavigationDestination{Label: route.label, Icon: route.icon})
	}
	for i, name := range variantNames {
		idx := i
		a.variantCache = append(a.variantCache, md3e.MenuItem{
			Label:   name,
			Radio:   true,
			OnClick: func() { a.variantIdx = idx },
		})
	}
	return a
}

// pageScroll is where a page path was scrolled.
func (a *app) pageScroll(path string) *ui.ScrollState {
	for i := range galleryRoutes {
		if galleryRoutes[i].path == path {
			return &a.scrolls[i]
		}
	}
	return &a.scrolls[0]
}

// scheme is the motion scheme the header chooses.
func (a *app) scheme() motion.Scheme {
	if a.motionIdx == 1 {
		return motion.Standard
	}
	return motion.Expressive
}

// currentTheme builds the MD3E theme the header controls describe, reusing
// the last one while they stay put.
func (a *app) currentTheme() *md3e.Theme {
	sig := themeSignature{
		seed:     seedPresets[a.seedIdx].color,
		variant:  tokens.Variant(a.variantIdx),
		contrast: tokens.Contrast(a.contrast),
		dark:     a.darkIdx == 1,
		scheme:   a.scheme(),
		reduce:   a.reduce,
	}
	if a.theme != nil && a.sig == sig {
		return a.theme
	}
	a.theme = md3e.ThemeFromSeed(sig.seed, md3e.ThemeOptions{
		Dark:         sig.dark,
		Variant:      sig.variant,
		Contrast:     sig.contrast,
		Motion:       sig.scheme,
		ReduceMotion: sig.reduce,
	})
	a.sig = sig
	return a.theme
}

// currentIndex is the page of the router's path.
func (a *app) currentIndex() int {
	path := a.router.Path()
	for i := range galleryRoutes {
		if galleryRoutes[i].path == path {
			return i
		}
	}
	return 0
}

// view builds the window: the header, the navigation rail and the page the
// router shows, all inside one ThemeScope.
func (a *app) view(c *ui.Context) {
	th := a.currentTheme()
	md3e.ThemeScope(c, th, func() {
		a.page = a.currentIndex()
		cs := th.Colors
		root := ui.Column(c).Fill().Background(cs.Surface)
		if a.rtl {
			root.Direction(ui.RightToLeft)
		}
		root.Children(func() {
			a.header(c, th)
			md3e.Divider(c)
			ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
				md3e.NavigationRail(c, a.destinations(), md3e.NavigationRailOptions{
					Selected: &a.page,
					Changed:  func(i int) { a.router.Push(galleryRoutes[i].path) },
					Expanded: &a.railExpanded,
					Header: func() ui.Element {
						return a.railHeader(c, th)
					},
					Label: "Gallery pages",
				})
				// Each route scrolls on its own, keeping the place it had
				// when the rail took the window to another page.
				ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
					a.router.View(c, func(r *ui.Route) {
						ui.Scroll(c.Key("page-scroll-" + r.Path())).TrackScroll(a.pageScroll(r.Path())).
							Grow(1).KeepScrollbars(true).Children(func() {
							a.route(c, r)
						})
					})
				})
			})
			md3e.SnackbarHost(c, a.snack)
		})
	}).Fill()
}

// route builds the page the router shows.
func (a *app) route(c *ui.Context, r *ui.Route) {
	for i := range galleryRoutes {
		if r.Match(galleryRoutes[i].path) {
			r.Title(galleryRoutes[i].label)
			galleryRoutes[i].build(a, c)
			return
		}
	}
	md3e.Text(c, "Page not found", tokens.HeadlineSmall)
}

// destinations are the rail's pages.
func (a *app) destinations() []md3e.NavigationDestination {
	return a.dests
}

// header holds the live theme controls.
func (a *app) header(c *ui.Context, th *md3e.Theme) {
	cs := th.Colors
	ui.Column(c).FillWidth().Shrink(0).Padding(16, 10).Gap(8).Background(cs.SurfaceContainerLow).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Wrap().Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				md3e.Text(c, "MyGo MD3E Gallery", tokens.TitleLarge, true)
				md3e.Text(c, "M1–M8 native components · pure Go, zero cgo, zero HTML", tokens.BodySmall)
			})
			ui.Box(c).Grow(1)
			md3e.ToggleIconButton(c.Key("rail-expand"), &a.railExpanded, "Expand navigation",
				mdicon.Menu.Draw, md3e.ToggleIconButtonOptions{Variant: md3e.IconButtonStandard, Size: tokens.SizeS})
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Wrap().Children(func() {
			// Seed color swatches.
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				for i, p := range seedPresets {
					border := cs.OutlineVariant
					width := float32(1)
					if i == a.seedIdx {
						border, width = cs.OnSurface, 2
					}
					btn := ui.ButtonBase(c.Key("seed-"+p.name)).Size(26, 26).
						Radius(13).Background(p.color).Border(width, border).
						Label("Seed " + p.name)
					if btn.Clicked() {
						a.seedIdx = i
					}
				}
			})
			md3e.ButtonGroup(c.Key("dark"), &a.darkIdx, []md3e.ButtonGroupItem{
				{Label: "Light"}, {Label: "Dark"},
			}, md3e.ButtonGroupOptions{
				Kind: md3e.ButtonGroupConnected, Variant: md3e.ButtonTonal, Size: tokens.SizeXS, Label: "Appearance",
			})
			md3e.ButtonGroup(c.Key("contrast"), &a.contrast, []md3e.ButtonGroupItem{
				{Label: "Std"}, {Label: "Medium"}, {Label: "High"},
			}, md3e.ButtonGroupOptions{
				Kind: md3e.ButtonGroupConnected, Variant: md3e.ButtonTonal, Size: tokens.SizeXS, Label: "Contrast",
			})
			md3e.ButtonGroup(c.Key("motion-scheme"), &a.motionIdx, []md3e.ButtonGroupItem{
				{Label: "Expressive"}, {Label: "Standard"},
			}, md3e.ButtonGroupOptions{
				Kind: md3e.ButtonGroupConnected, Variant: md3e.ButtonTonal, Size: tokens.SizeXS, Label: "Motion scheme",
			})
			md3e.DropdownMenu(c.Key("variant"), "Variant: "+variantNames[a.variantIdx], a.variantItems(),
				md3e.MenuOptions{Label: "Scheme variant"})
			md3e.Checkbox(c.Key("reduce"), &a.reduce, md3e.CheckboxOptions{Label: "Reduce motion"})
			md3e.Switch(c.Key("rtl"), &a.rtl, md3e.SwitchOptions{Label: "RTL"})
		})
	})
}

// variantItems are the menu rows that pick a DynamicScheme variant.
func (a *app) variantItems() []md3e.MenuItem {
	for i := range a.variantCache {
		a.variantCache[i].Checked = i == a.variantIdx
	}
	return a.variantCache
}

// railHeader shows the palette preview above the rail's destinations.
func (a *app) railHeader(c *ui.Context, th *md3e.Theme) ui.Element {
	cs := th.Colors
	return ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
		ui.Box(c).Size(52, 52).Radius(16).Background(cs.Primary).Center().Children(func() {
			md3e.Text(c, "M3", tokens.TitleMedium, true).TextColor(cs.OnPrimary)
		})
	})
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "MyGo MD3E Gallery",
			Width:   1360,
			Height:  900,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// page lays out a page: its title, a description and its sections, in a
// centered reading column that keeps its lines comfortable on wide windows.
func page(c *ui.Context, title, subtitle string, body func()) {
	cs := md3e.CurrentTheme(c).Colors
	ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(1120).Padding(24, 24, 48).Gap(18).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				md3e.Text(c, title, tokens.HeadlineMedium, true)
				md3e.Text(c, subtitle, tokens.BodyMedium).TextColor(cs.OnSurfaceVariant)
			})
			body()
		})
	})
}

// section is a titled card of a page.
func section(c *ui.Context, title string, body func()) {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	ui.Column(c).FillWidth().Padding(16).Gap(12).
		Radius(float32(th.Shapes.Large)).Background(cs.SurfaceContainerLow).
		Border(1, cs.OutlineVariant).Children(func() {
		md3e.Text(c, title, tokens.TitleMedium, true)
		body()
	})
}

// caption is a supporting line under a control.
func caption(c *ui.Context, text string) {
	cs := md3e.CurrentTheme(c).Colors
	md3e.Text(c, text, tokens.BodySmall).TextColor(cs.OnSurfaceVariant)
}

// statusLine is a caption that keeps its place before the first action, so a
// page does not jump when a control reports one.
func statusLine(c *ui.Context, prefix, value string) {
	if value == "" {
		value = "None yet"
	}
	caption(c, prefix+value)
}
