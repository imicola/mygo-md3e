package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	md3e "mygo-md3e"
	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// navigationPage holds the state of the navigation and media demos.
type navigationPage struct {
	ready        bool
	bar          int
	rail         int
	railExpanded bool
	drawer       int
	drawerOpen   bool
	modalDrawer  bool
	primary      int
	secondary    int
	hero         int
	browse       int
	uncontained  int
}

func (p *navigationPage) ensure() {
	if !p.ready {
		p.ready = true
		p.bar, p.rail, p.drawer = 0, 1, 2
		p.railExpanded = true
		p.drawerOpen = true
	}
}

// navDestinations are the demo destinations of the navigation controls.
var navDestinations = []md3e.NavigationDestination{
	{Label: "Home", Icon: mdicon.Star.Draw},
	{Label: "Inbox", Icon: mdicon.MoreHoriz.Draw, Badge: "3"},
	{Label: "Explore", Icon: mdicon.Search.Draw},
	{Label: "Library", Icon: mdicon.Check.Draw, SmallBadge: true},
}

// railDemoDestinations are the labels of the inline rail demo, kept apart
// from the bar's so both can be told apart by screen readers and tests.
var railDemoDestinations = []md3e.NavigationDestination{
	{Label: "Rail overview", Icon: mdicon.Star.Draw},
	{Label: "Rail messages", Icon: mdicon.MoreHoriz.Draw, Badge: "3"},
	{Label: "Rail search", Icon: mdicon.Search.Draw},
	{Label: "Rail saved", Icon: mdicon.Check.Draw, SmallBadge: true},
}

// drawerItems are the demo rows of the drawers.
var drawerItems = []md3e.NavigationDrawerItem{
	{Label: "Home", Icon: mdicon.Star.Draw, Section: "Mail"},
	{Label: "Inbox", Icon: mdicon.MoreHoriz.Draw, Badge: "12"},
	{Label: "Starred", Icon: mdicon.Star.Draw},
	{Label: "Settings", Icon: mdicon.Edit.Draw, Section: "App"},
}

// navigationPageView shows the navigation bar, rail, drawers, tabs and the
// three carousel layouts.
func (a *app) navigationPageView(c *ui.Context) {
	th := md3e.CurrentTheme(c)
	p := &a.navigation
	p.ensure()

	page(c, "Navigation & Media",
		"Navigation bars, rails and drawers, the primary and secondary tabs, and the hero, multi-browse and uncontained carousels.",
		func() {
			section(c, "Navigation bar", func() {
				ui.Box(c).FillWidth().MaxWidth(560).Children(func() {
					md3e.NavigationBar(c, navDestinations, md3e.NavigationBarOptions{
						Selected: &p.bar, Labels: md3e.NavigationLabelsAlways, Label: "Demo navigation bar",
					})
				})
				caption(c, "Always, selected-only and never label modes are available; the pill springs between destinations.")
			})

			section(c, "Navigation rail", func() {
				ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Start).Children(func() {
					ui.Box(c).Height(280).Children(func() {
						md3e.NavigationRail(c, railDemoDestinations, md3e.NavigationRailOptions{
							Selected: &p.rail, Expanded: &p.railExpanded, Label: "Demo navigation rail",
						})
					})
					ui.Column(c).Gap(6).Grow(1).Children(func() {
						caption(c, "Collapsed it is 80 DIP wide; expanded it grows to 256 DIP with long pills.")
						md3e.Switch(c.Key("rail-expand-demo"), &p.railExpanded, md3e.SwitchOptions{
							Label: "Expand rail",
						})
					})
				})
			})

			section(c, "Navigation drawers", func() {
				ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Start).Wrap().Children(func() {
					md3e.NavigationDrawer(c.Key("drawer-inline"), drawerItems, md3e.NavigationDrawerOptions{
						Open: &p.drawerOpen, Selected: &p.drawer, Width: 260, Label: "Inline drawer",
					})
					ui.Column(c).Gap(10).Grow(1).Children(func() {
						md3e.Switch(c.Key("drawer-open"), &p.drawerOpen, md3e.SwitchOptions{Label: "Inline drawer open"})
						if md3e.Button(c.Key("drawer-modal"), "Open modal drawer", md3e.ButtonOptions{
							Variant: md3e.ButtonTonal, Size: tokens.SizeS,
						}).Clicked() {
							p.modalDrawer = true
						}
						caption(c, "The modal drawer takes the focus, closes with Escape or a scrim press and gives the focus back.")
					})
					md3e.NavigationDrawer(c.Key("drawer-modal-el"), drawerItems, md3e.NavigationDrawerOptions{
						Open: &p.modalDrawer, Modal: true, Selected: &p.drawer, Width: 280, Label: "Modal drawer",
					})
				})
			})

			section(c, "Tabs", func() {
				tabs := []md3e.TabItem{
					{Label: "Overview", Icon: mdicon.Star.Draw},
					{Label: "Specs", Icon: mdicon.Check.Draw},
					{Label: "Reviews", Icon: mdicon.MoreHoriz.Draw},
				}
				ui.Column(c).FillWidth().Gap(12).Children(func() {
					md3e.PrimaryTabs(c.Key("primary-tabs"), tabs, md3e.TabsOptions{
						Selected: &p.primary, Label: "Primary tabs",
					})
					md3e.SecondaryTabs(c.Key("secondary-tabs"), []md3e.TabItem{
						{Label: "Activity", Icon: mdicon.Clock.Draw},
						{Label: "Files", Icon: mdicon.Edit.Draw},
						{Label: "Notes", Icon: mdicon.Check.Draw},
					}, md3e.TabsOptions{
						Selected: &p.secondary, Label: "Secondary tabs",
					})
					md3e.PrimaryTabs(c.Key("scrollable-tabs"), []md3e.TabItem{
						{Label: "One"}, {Label: "Two"}, {Label: "Three"}, {Label: "Four"},
						{Label: "Five"}, {Label: "Six"}, {Label: "Seven"},
					}, md3e.TabsOptions{Scrollable: true, Label: "Scrollable tabs"})
				})
			})

			section(c, "Carousels", func() {
				ui.Column(c).FillWidth().Gap(16).Children(func() {
					md3e.Text(c, "Hero", tokens.LabelMedium, true)
					md3e.Carousel(c.Key("carousel-hero"), carouselItems(c, th), md3e.CarouselOptions{
						Index: &p.hero, Layout: md3e.CarouselHero, Height: 190, Label: "Hero carousel",
					})
					md3e.Text(c, "Multi-browse", tokens.LabelMedium, true)
					md3e.Carousel(c.Key("carousel-browse"), carouselItems(c, th), md3e.CarouselOptions{
						Index: &p.browse, Layout: md3e.CarouselMultiBrowse, Height: 170, Label: "Multi-browse carousel",
					})
					md3e.Text(c, "Uncontained", tokens.LabelMedium, true)
					md3e.Carousel(c.Key("carousel-uncontained"), carouselItems(c, th), md3e.CarouselOptions{
						Index: &p.uncontained, Layout: md3e.CarouselUncontained, Height: 150, Label: "Uncontained carousel",
					})
				})
			})
		})
}

// carouselItems builds the slides of the demo carousels.
func carouselItems(c *ui.Context, th *md3e.Theme) []md3e.CarouselItem {
	cs := th.Colors
	paints := []struct {
		label string
		bg    ui.Color
		fg    ui.Color
	}{
		{"Mountains", cs.PrimaryContainer, cs.OnPrimaryContainer},
		{"Lakes", cs.SecondaryContainer, cs.OnSecondaryContainer},
		{"Forests", cs.TertiaryContainer, cs.OnTertiaryContainer},
		{"Deserts", cs.SurfaceContainerHighest, cs.OnSurface},
		{"Coasts", cs.Primary, cs.OnPrimary},
	}
	items := make([]md3e.CarouselItem, 0, len(paints))
	for i := range paints {
		paint := paints[i]
		items = append(items, md3e.CarouselItem{
			Label: paint.label,
			Content: func() ui.Element {
				return ui.Box(c).Fill().Background(paint.bg).Center().Children(func() {
					ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
						md3e.Text(c, paint.label, tokens.HeadlineSmall, true).TextColor(paint.fg)
						md3e.Text(c, fmt.Sprintf("Slide %d", i+1), tokens.LabelSmall).TextColor(paint.fg.Alpha(0.8))
					})
				})
			},
		})
	}
	return items
}
