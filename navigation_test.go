package md3e

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
	"mygo-md3e/tokens"
)

// navigationCanvas builds a w x h white canvas holding build's content.
func navigationCanvas(th *Theme, w, h int, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(float32(w), float32(h)).Background(canvasRGB).Children(func() {
				build(c)
			})
		})
	}, w, h)
}

// navBar builds a bottom navigation bar over the canvas.
func navBar(selected *int, changed *int, labels NavigationLabelMode) func(c *ui.Context) {
	return func(c *ui.Context) {
		ui.Column(c).FillHeight().Justify(ui.End).Children(func() {
			NavigationBar(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
				{Label: "Saved", Icon: mdicon.Star.Draw, Badge: "3"},
			}, NavigationBarOptions{
				Selected: selected,
				Changed: func(i int) {
					if changed != nil {
						*changed = i
					}
				},
				Labels: labels,
				Label:  "Main",
			})
		})
	}
}

// tabInfo returns a tab's access info by name.
func tabInfo(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleTab, name)
	if !ok {
		t.Fatalf("%q tab not in the access tree", name)
	}
	return info
}

// TestNavigationBarSelectsWithPointerAndKeys checks clicking, arrow-key focus
// and Enter or Space choose destinations, badges render, and the pill settles.
func TestNavigationBarSelectsWithPointerAndKeys(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected, changed := 0, -1
	tt := navigationCanvas(th, 480, 320, navBar(&selected, &changed, NavigationLabelsAlways))
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Main"); !ok {
		t.Fatal("the bar is not a named navigation landmark")
	}
	if !tabInfo(t, tt, "Home").Selected {
		t.Fatal("the initial destination does not read as selected")
	}
	if !tt.HasText("3") {
		t.Fatal("the count badge does not render")
	}

	tt.Click("Search")
	tt.Advance(time.Second)
	if selected != 1 || changed != 1 {
		t.Fatalf("the click chose %d (callback %d), want 1", selected, changed)
	}
	if !tabInfo(t, tt, "Search").Selected {
		t.Fatal("the chosen destination does not read as selected")
	}
	if tt.FramePending() {
		t.Fatal("the bar keeps asking for frames after choosing")
	}

	// The horizontal arrows move the focus among the destinations, and Enter
	// or Space chooses the one focused.
	tt.Click("Home")
	tt.Advance(300 * time.Millisecond)
	tt.Key(0, ui.KeyRight)
	if !tt.Focused("Search") {
		t.Fatal("Right does not move the focus to the next destination")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Advance(300 * time.Millisecond)
	if selected != 1 {
		t.Fatalf("Enter chose %d, want 1", selected)
	}
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeySpace)
	tt.Advance(time.Second)
	if selected != 0 {
		t.Fatalf("Left and Space chose %d, want 0", selected)
	}
	if tt.FramePending() {
		t.Fatal("the bar keeps asking for frames after key navigation")
	}
}

// labelInk counts the pixels of a destination's label band that differ from
// the bar's background: labels paint their text there while hidden ones leave
// it empty. The band sits below both the 32 DIP indicator box and the pill of
// an icon-only destination in the 80 DIP bar.
func labelInk(tt *ui.Tester, b ui.Rect, bg ui.Color) int {
	count := 0
	for y := int(b.Y + 58); y < int(b.Y+68); y++ {
		for x := int(b.X) + 4; x < int(b.X+b.W)-4; x++ {
			if !colorMatches(tt.Image().RGBAAt(x, y), bg, 24) {
				count++
			}
		}
	}
	return count
}

// TestNavigationBarLabelsAndRTL checks the label visibility policies and that
// the destinations mirror right to left.
func TestNavigationBarLabelsAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	barBg := SurfaceColorForElevation(th, tokens.Level2)
	selected := 0
	only := navigationCanvas(th, 480, 320, navBar(&selected, nil, NavigationLabelsSelectedOnly))
	if labelInk(only, tabInfo(t, only, "Home").Bounds, barBg) < 4 {
		t.Fatal("the selected destination's label does not paint its ink")
	}
	if n := labelInk(only, tabInfo(t, only, "Search").Bounds, barBg); n != 0 {
		t.Fatalf("an unselected destination paints %d label pixels in SelectedOnly mode", n)
	}
	never := navigationCanvas(th, 480, 320, navBar(&selected, nil, NavigationLabelsNever))
	if n := labelInk(never, tabInfo(t, never, "Home").Bounds, barBg); n != 0 {
		t.Fatalf("a destination paints %d label pixels in Never mode", n)
	}

	rtl := navigationCanvas(th, 480, 320, navBar(&selected, nil, NavigationLabelsAlways))
	rtl.SetDirection(ui.RightToLeft)
	home := tabInfo(t, rtl, "Home")
	saved := tabInfo(t, rtl, "Saved")
	if home.Bounds.X <= saved.Bounds.X {
		t.Fatalf("right to left puts Home at %.1f and Saved at %.1f, want Home on the right", home.Bounds.X, saved.Bounds.X)
	}
}

// TestNavigationBarRTLArrowKeys checks the horizontal arrows mirror right to
// left: Left moves to the next destination and Right to the previous one.
func TestNavigationBarRTLArrowKeys(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 0
	tt := navigationCanvas(th, 480, 320, navBar(&selected, nil, NavigationLabelsAlways))
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(300 * time.Millisecond)

	home := tabInfo(t, tt, "Home")
	tt.ClickAt(home.Bounds.X+home.Bounds.W/2, home.Bounds.Y+home.Bounds.H/2)
	if !tt.Focused("Home") {
		t.Fatal("the right-to-left click does not focus the destination")
	}
	tt.Key(0, ui.KeyLeft)
	if !tt.Focused("Search") {
		t.Fatal("Left does not move to the next destination right to left")
	}
	tt.Key(0, ui.KeyLeft)
	if !tt.Focused("Saved") {
		t.Fatal("Left again does not move to the last destination")
	}
	tt.Key(0, ui.KeyRight)
	if !tt.Focused("Search") {
		t.Fatal("Right does not move back a destination right to left")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Advance(300 * time.Millisecond)
	if selected != 1 {
		t.Fatalf("Enter chose %d, want 1", selected)
	}
}

// TestModalNavigationDrawerDisabledKeepsOpenState checks a disabled modal
// drawer, by its option or by an ancestor, leaves the caller's open state
// alone so that re-enabling it brings the drawer back.
func TestModalNavigationDrawerDisabledKeepsOpenState(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cases := []struct {
		name  string
		build func(c *ui.Context, open, disabled *bool)
	}{
		{
			name: "option",
			build: func(c *ui.Context, open, disabled *bool) {
				NavigationDrawer(c, drawerItems(false), NavigationDrawerOptions{
					Modal: true, Open: open, Label: "Drawer", Disabled: *disabled,
				})
			},
		},
		{
			name: "ancestor",
			build: func(c *ui.Context, open, disabled *bool) {
				ui.Column(c).Disabled(*disabled).Children(func() {
					NavigationDrawer(c, drawerItems(false), NavigationDrawerOptions{
						Modal: true, Open: open, Label: "Drawer",
					})
				})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			open := true
			disabled := true
			tt := navigationCanvas(th, 600, 400, func(c *ui.Context) {
				tc.build(c, &open, &disabled)
			})
			tt.Frame()
			if _, ok := tt.AccessByRole(ui.RoleDialog, "Drawer"); ok {
				t.Fatal("a disabled drawer still shows")
			}
			if !open {
				t.Fatal("disabling the drawer rewrote the caller's open state")
			}

			disabled = false
			tt.Frame()
			tt.Advance(600 * time.Millisecond)
			if _, ok := tt.AccessByRole(ui.RoleDialog, "Drawer"); !ok {
				t.Fatal("re-enabling does not restore the open drawer")
			}
			if !open {
				t.Fatal("the restored drawer lost the caller's open state")
			}
		})
	}
}

// TestNavigationRailExpandAndCollapse checks the 80 and 256 DIP widths, the
// sprung transition settling, the header FAB slot, and selection.
func TestNavigationRailExpandAndCollapse(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	expanded := false
	selected := 0
	tt := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
				{Label: "Saved", Icon: mdicon.Star.Draw, SmallBadge: true},
			}, NavigationRailOptions{
				Selected: &selected,
				Expanded: &expanded,
				Header: func() ui.Element {
					return FAB(c, "Create", mdicon.Add.Draw)
				},
				Label: "Rail",
			})
		})
	})
	tt.Advance(time.Second)
	rail, ok := tt.AccessByRole(ui.RoleNavigation, "Rail")
	if !ok {
		t.Fatal("the rail is not a named navigation landmark")
	}
	if math.Abs(float64(rail.Bounds.W-navRailCollapsedWidth)) > 0.5 {
		t.Fatalf("collapsed rail width %.1f, want %d", rail.Bounds.W, navRailCollapsedWidth)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Create"); !ok {
		t.Fatal("the header FAB does not render")
	}
	tt.Click("Search")
	tt.Advance(300 * time.Millisecond)
	if selected != 1 {
		t.Fatalf("the rail click chose %d, want 1", selected)
	}

	expanded = true
	tt.Frame()
	tt.Advance(time.Second)
	rail, _ = tt.AccessByRole(ui.RoleNavigation, "Rail")
	if math.Abs(float64(rail.Bounds.W-navRailExpandedWidth)) > 0.5 {
		t.Fatalf("expanded rail width %.1f, want %d", rail.Bounds.W, navRailExpandedWidth)
	}
	if tt.FramePending() {
		t.Fatal("the expanded rail keeps asking for frames")
	}
	if !tabInfo(t, tt, "Search").Selected {
		t.Fatal("the chosen destination does not read as selected when expanded")
	}

	expanded = false
	tt.Frame()
	tt.Advance(time.Second)
	rail, _ = tt.AccessByRole(ui.RoleNavigation, "Rail")
	if math.Abs(float64(rail.Bounds.W-navRailCollapsedWidth)) > 0.5 {
		t.Fatalf("collapsed rail width %.1f, want %d", rail.Bounds.W, navRailCollapsedWidth)
	}
	if tt.FramePending() {
		t.Fatal("the collapsed rail keeps asking for frames")
	}
}

// TestNavigationRailScrollsShortViewport checks that a rail shorter than its
// list scrolls to its last destination, which then shows and chooses.
func TestNavigationRailScrollsShortViewport(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 0
	destinations := []NavigationDestination{
		{Label: "Home", Icon: mdicon.Add.Draw},
		{Label: "Search", Icon: mdicon.Search.Draw},
		{Label: "Saved", Icon: mdicon.Star.Draw},
		{Label: "Library", Icon: mdicon.Check.Draw},
		{Label: "Inbox", Icon: mdicon.MoreHoriz.Draw},
		{Label: "Help", Icon: mdicon.Edit.Draw},
	}
	tt := navigationCanvas(th, 320, 240, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, destinations, NavigationRailOptions{Selected: &selected, Label: "Rail"})
		})
	})
	tt.Advance(time.Second)
	rail, ok := tt.AccessByRole(ui.RoleNavigation, "Rail")
	if !ok {
		t.Fatal("the rail is not a named navigation landmark")
	}
	if info := tabInfo(t, tt, "Help"); !info.Offscreen && info.Bounds.H > 0 {
		t.Fatalf("the last destination shows before scrolling: %+v", info.Bounds)
	}
	if err := tt.Click("Help"); err == nil {
		t.Fatal("the clipped last destination takes a click before scrolling")
	}

	tt.Wheel(rail.Bounds.X+rail.Bounds.W/2, rail.Bounds.Y+rail.Bounds.H/2, 0, 300)
	tt.Advance(600 * time.Millisecond)
	info := tabInfo(t, tt, "Help")
	if info.Offscreen || info.Bounds.H <= 0 {
		t.Fatalf("scrolling the rail does not show its last destination: %+v", info)
	}
	if info.Bounds.Y < rail.Bounds.Y-0.5 || info.Bounds.Y+info.Bounds.H > rail.Bounds.Y+rail.Bounds.H+0.5 {
		t.Fatalf("the last destination at %v lies outside the rail %v", info.Bounds, rail.Bounds)
	}
	if err := tt.Click("Help"); err != nil {
		t.Fatalf("click the scrolled last destination: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if selected != len(destinations)-1 {
		t.Fatalf("the scrolled destination chose %d, want %d", selected, len(destinations)-1)
	}
}

// TestNavigationRailReduceMotion checks the width jumps between states with no
// animation frames under reduced motion.
func TestNavigationRailReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	expanded := false
	selected := 0
	tt := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationRailOptions{Selected: &selected, Expanded: &expanded, Label: "Rail"})
		})
	})
	expanded = true
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("a reduced-motion expansion asks for frames")
	}
	rail, _ := tt.AccessByRole(ui.RoleNavigation, "Rail")
	if math.Abs(float64(rail.Bounds.W-navRailExpandedWidth)) > 0.5 {
		t.Fatalf("reduced-motion expanded width %.1f, want %d", rail.Bounds.W, navRailExpandedWidth)
	}
	expanded = false
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("a reduced-motion collapse asks for frames")
	}
	rail, _ = tt.AccessByRole(ui.RoleNavigation, "Rail")
	if math.Abs(float64(rail.Bounds.W-navRailCollapsedWidth)) > 0.5 {
		t.Fatalf("reduced-motion collapsed width %.1f, want %d", rail.Bounds.W, navRailCollapsedWidth)
	}
}

// drawerItems are the destinations of the drawer tests.
func drawerItems(locked bool) []NavigationDrawerItem {
	return []NavigationDrawerItem{
		{Label: "Inbox", Icon: mdicon.Search.Draw, Badge: "12"},
		{Label: "Settings", Icon: mdicon.Edit.Draw, Section: "Account", Disabled: locked},
	}
}

// TestModalNavigationDrawerClosesOnScrimEscapeAndDrag checks the modal
// drawer's scrim, Escape and drag closing, its focus restoration, and that a
// short drag springs back.
func TestModalNavigationDrawerClosesOnScrimEscapeAndDrag(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := false
	selected := -1
	tt := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		NavigationDrawer(c, drawerItems(false), NavigationDrawerOptions{
			Modal:    true,
			Open:     &open,
			Selected: &selected,
			Label:    "Drawer",
		})
		ui.Box(c).Size(600, 400).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			if Button(c, "Menu").Clicked() {
				open = true
			}
		})
	})
	openDrawer := func() ui.AccessInfo {
		t.Helper()
		if err := tt.Click("Menu"); err != nil {
			t.Fatalf("clicking Menu: %v", err)
		}
		tt.Advance(600 * time.Millisecond)
		info, ok := tt.AccessByRole(ui.RoleDialog, "Drawer")
		if !ok {
			t.Fatal("the modal drawer is not a dialog in the access tree")
		}
		return info
	}
	closeDrawer := func(what string) {
		t.Helper()
		tt.Advance(600 * time.Millisecond)
		if _, ok := tt.AccessByRole(ui.RoleDialog, "Drawer"); ok {
			t.Fatalf("%s does not close the drawer", what)
		}
		if !tt.Focused("Menu") {
			t.Fatalf("%s does not give the focus back to the trigger", what)
		}
		if tt.FramePending() {
			t.Fatal("the closed drawer keeps asking for frames")
		}
	}

	dialog := openDrawer()
	if !tt.HasText("12") {
		t.Fatal("the drawer badge text does not show")
	}
	if !tt.HasText("Account") {
		t.Fatal("the drawer section title does not show")
	}
	tt.ClickAt(580, 200)
	closeDrawer("a scrim press")

	openDrawer()
	tt.Key(0, ui.KeyEscape)
	closeDrawer("Escape")

	// A drag past the threshold toward the drawer's edge closes it.
	dialog = openDrawer()
	x, y := dialog.Bounds.X+20, dialog.Bounds.Y+6
	tt.Press(x, y)
	tt.Move(x-120, y)
	tt.Release(x-120, y)
	closeDrawer("a leftward drag")

	// A short drag springs the panel back and leaves the drawer open.
	dialog = openDrawer()
	x, y = dialog.Bounds.X+20, dialog.Bounds.Y+6
	tt.Press(x, y)
	tt.Move(x-40, y)
	tt.Release(x-40, y)
	tt.Move(590, 390)
	tt.Advance(time.Second)
	after, ok := tt.AccessByRole(ui.RoleDialog, "Drawer")
	if !ok {
		t.Fatal("a short drag closes the drawer")
	}
	if math.Abs(float64(after.Bounds.X-dialog.Bounds.X)) > 1 {
		t.Fatalf("the drawer rests at x %.1f, want its original %.1f", after.Bounds.X, dialog.Bounds.X)
	}
	if tt.FramePending() {
		t.Fatal("the sprung-back drawer keeps asking for frames")
	}
}

// TestModalNavigationDrawerRTLDrag checks the panel anchors on the right and
// closes with a rightward drag right to left.
func TestModalNavigationDrawerRTLDrag(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := false
	selected := -1
	tt := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		NavigationDrawer(c, drawerItems(false), NavigationDrawerOptions{
			Modal:    true,
			Open:     &open,
			Selected: &selected,
			Label:    "Drawer",
		})
		ui.Box(c).Size(600, 400).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			if Button(c, "Menu").Clicked() {
				open = true
			}
		})
	})
	tt.SetDirection(ui.RightToLeft)
	if err := tt.Click("Menu"); err != nil {
		t.Fatalf("clicking Menu: %v", err)
	}
	tt.Advance(600 * time.Millisecond)
	info, ok := tt.AccessByRole(ui.RoleDialog, "Drawer")
	if !ok {
		t.Fatal("the modal drawer is not a dialog in the access tree")
	}
	if info.Bounds.X < 600-navDrawerWidth-1 {
		t.Fatalf("right-to-left drawer x %.1f, want it anchored on the right", info.Bounds.X)
	}
	x, y := info.Bounds.X+info.Bounds.W-20, info.Bounds.Y+6
	tt.Press(x, y)
	tt.Move(x+120, y)
	tt.Release(x+120, y)
	tt.Advance(600 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Drawer"); ok {
		t.Fatal("a rightward drag does not close the right-to-left drawer")
	}
}

// TestInlineNavigationDrawerWidthAndSelection checks the dismissible drawer's
// width animation, selection, badge text, disabled row and closed focus state.
func TestInlineNavigationDrawerWidthAndSelection(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := true
	selected := 0
	changed := -1
	tt := navigationCanvas(th, 620, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().AlignItems(ui.Stretch).Children(func() {
			NavigationDrawer(c, drawerItems(false), NavigationDrawerOptions{
				Open:     &open,
				Selected: &selected,
				Changed:  func(i int) { changed = i },
				Label:    "Drawer",
				Width:    300,
			})
			ui.Box(c).Size(320, 400)
		})
	})
	tt.Frame()
	tt.Advance(time.Second)
	drawer, ok := tt.AccessByRole(ui.RoleRegion, "Drawer")
	if !ok {
		t.Fatal("the inline drawer is not a named region")
	}
	if math.Abs(float64(drawer.Bounds.W)-300) > 0.5 {
		t.Fatalf("open drawer width %.1f, want 300", drawer.Bounds.W)
	}
	if !tt.HasText("12") {
		t.Fatal("the drawer badge text does not show")
	}
	if !tt.HasText("Account") {
		t.Fatal("the drawer section title does not show")
	}
	tt.Click("Settings")
	tt.Advance(time.Second)
	if selected != 1 || changed != 1 {
		t.Fatalf("the drawer click chose %d (callback %d), want 1", selected, changed)
	}
	if tt.FramePending() {
		t.Fatal("the drawer keeps asking for frames after choosing")
	}

	open = false
	tt.Frame()
	tt.Advance(time.Second)
	drawer, _ = tt.AccessByRole(ui.RoleRegion, "Drawer")
	if drawer.Bounds.W > 0.5 {
		t.Fatalf("closed drawer width %.1f, want 0", drawer.Bounds.W)
	}
	if _, ok := tt.AccessByRole(ui.RoleTab, "Inbox"); ok {
		t.Fatal("a closed drawer keeps its destinations in the access tree")
	}
	if tt.FramePending() {
		t.Fatal("the closed drawer keeps asking for frames")
	}
}

// TestNavigationBarAndRailDisabled checks a disabled bar and a disabled rail
// destination read as disabled and take no input.
func TestNavigationBarAndRailDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 2
	tt := navigationCanvas(th, 480, 320, func(c *ui.Context) {
		ui.Column(c).FillHeight().Justify(ui.End).Children(func() {
			NavigationBar(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationBarOptions{Selected: &selected, Label: "Main", Disabled: true})
		})
	})
	tt.Advance(300 * time.Millisecond)
	home := tabInfo(t, tt, "Home")
	if !home.Disabled {
		t.Fatal("a destination under a disabled bar does not read as disabled")
	}
	tt.ClickAt(home.Bounds.X+home.Bounds.W/2, home.Bounds.Y+home.Bounds.H/2)
	tt.Advance(300 * time.Millisecond)
	if selected != 2 {
		t.Fatalf("the disabled bar chose %d", selected)
	}

	railSelected := -1
	rail := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw, Disabled: true},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationRailOptions{Selected: &railSelected, Label: "Rail"})
		})
	})
	rail.Advance(300 * time.Millisecond)
	info := tabInfo(t, rail, "Home")
	if !info.Disabled {
		t.Fatal("a disabled rail destination does not read as disabled")
	}
	rail.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	rail.Advance(300 * time.Millisecond)
	if railSelected != -1 {
		t.Fatalf("the disabled rail destination chose %d", railSelected)
	}
}

// TestNavigationDrawerDisabled checks a disabled drawer disables its
// destinations and takes no input.
func TestNavigationDrawerDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := true
	selected := 3
	tt := navigationCanvas(th, 400, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().AlignItems(ui.Stretch).Children(func() {
			NavigationDrawer(c, drawerItems(false), NavigationDrawerOptions{
				Open:     &open,
				Selected: &selected,
				Label:    "Drawer",
				Disabled: true,
			})
		})
	})
	tt.Advance(300 * time.Millisecond)
	info := tabInfo(t, tt, "Inbox")
	if !info.Disabled {
		t.Fatal("a destination under a disabled drawer does not read as disabled")
	}
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tt.Advance(300 * time.Millisecond)
	if selected != 3 {
		t.Fatalf("the disabled drawer chose %d", selected)
	}
}

// TestNavDisabledSelectedPillAndCollapsedRailGeometry checks disabled selected
// destinations paint DisabledContainerOpacity instead of SecondaryContainer and
// that the collapsed rail indicator wraps only its icon slot without clipping
// the label or exceeding item width.
func TestNavDisabledSelectedPillAndCollapsedRailGeometry(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors

	barSel := 0
	bar := navigationCanvas(th, 480, 320, func(c *ui.Context) {
		ui.Column(c).FillHeight().Justify(ui.End).Children(func() {
			NavigationBar(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationBarOptions{Selected: &barSel, Label: "Bar", Disabled: true})
		})
	})
	bar.Advance(400 * time.Millisecond)
	home := tabInfo(t, bar, "Home")
	barBg := SurfaceColorForElevation(th, tokens.Level2)
	wantBarPill := cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(barBg)
	barImg := bar.Image()
	barPx := barImg.RGBAAt(int(home.Bounds.X+home.Bounds.W/2-22), int(home.Bounds.Y+28))
	if colorMatches(barPx, cs.SecondaryContainer, 14) {
		t.Fatalf("disabled selected bar pill still painted bright SecondaryContainer %+v", barPx)
	}
	if !colorMatches(barPx, wantBarPill, 14) {
		t.Fatalf("disabled selected bar pill = %+v, want %+v", barPx, rgbOf(wantBarPill))
	}

	// Collapsed rail pill sits over the top icon slot (width <= 56, height 32)
	// and never paints SecondaryContainer into the label band below it.
	railSel := 0
	rail := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationRailOptions{Selected: &railSel, Label: "Rail"})
		})
	})
	rail.Advance(400 * time.Millisecond)
	railHome := tabInfo(t, rail, "Home")
	railImg := rail.Image()
	iconY := int(railHome.Bounds.Y + 14)
	labelY := int(railHome.Bounds.Y + railHome.Bounds.H - 6)
	railPx := railImg.RGBAAt(int(railHome.Bounds.X+8), iconY)
	if !colorMatches(railPx, cs.SecondaryContainer, 14) {
		t.Fatalf("collapsed rail icon pill pixel = %+v, want SecondaryContainer %+v", railPx, rgbOf(cs.SecondaryContainer))
	}
	labelBandPx := railImg.RGBAAt(int(railHome.Bounds.X+6), labelY)
	if colorMatches(labelBandPx, cs.SecondaryContainer, 14) {
		t.Fatalf("collapsed rail pill leaked into label band at y=%d", labelY)
	}
}

// TestModalDrawerItemSwipeAndScroll checks horizontal swipe-to-dismiss initiated
// over a destination item suppresses click selection and closes, and long
// drawer lists scroll cleanly.
func TestModalDrawerItemSwipeAndScroll(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := false
	selected := 0
	changed := 0
	var items []NavigationDrawerItem
	for i := 0; i < 16; i++ {
		items = append(items, NavigationDrawerItem{
			Label: "Item " + string(rune('A'+i)),
			Icon:  mdicon.Star.Draw,
		})
	}
	tt := navigationCanvas(th, 600, 320, func(c *ui.Context) {
		NavigationDrawer(c, items, NavigationDrawerOptions{
			Modal:    true,
			Open:     &open,
			Selected: &selected,
			Changed:  func(int) { changed++ },
			Label:    "Drawer",
		})
		ui.Box(c).Size(600, 320).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			if Button(c, "Menu").Clicked() {
				open = true
			}
		})
	})
	tt.Click("Menu")
	tt.Advance(600 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Drawer"); !ok {
		t.Fatal("drawer did not open")
	}
	itemB := tabInfo(t, tt, "Item B")
	// Dragging leftward starting directly on "Item B" closes the drawer without
	// clicking/selecting Item B.
	x, y := itemB.Bounds.X+itemB.Bounds.W/2, itemB.Bounds.Y+itemB.Bounds.H/2
	tt.Press(x, y)
	tt.Move(x-100, y)
	tt.Release(x-100, y)
	tt.Advance(600 * time.Millisecond)
	if open {
		t.Fatal("horizontal swipe starting on destination item did not close modal drawer")
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Drawer"); ok {
		t.Fatal("swiped modal drawer still present in access tree")
	}
	if selected != 0 || changed != 0 {
		t.Fatalf("item swipe also triggered selection (%d, changed=%d)", selected, changed)
	}
	if tt.FramePending() {
		t.Fatal("swiped modal drawer keeps requesting frames")
	}

	// Reopen and verify vertical scrolling moves off-screen items into view.
	tt.Click("Menu")
	tt.Advance(600 * time.Millisecond)
	firstBefore := tabInfo(t, tt, "Item A").Bounds
	dlg, _ := tt.AccessByRole(ui.RoleDialog, "Drawer")
	if firstBefore.X < dlg.Bounds.X || firstBefore.X >= dlg.Bounds.X+dlg.Bounds.W || firstBefore.W <= 0 || firstBefore.H <= 0 {
		t.Fatalf("drawer item not laid out inside dialog panel: item=%+v dlg=%+v", firstBefore, dlg.Bounds)
	}
	tt.Wheel(dlg.Bounds.X+100, dlg.Bounds.Y+160, 0, -240)
	tt.Advance(400 * time.Millisecond)
	firstAfter := tabInfo(t, tt, "Item A").Bounds
	if firstAfter.Y >= firstBefore.Y {
		tt.Wheel(dlg.Bounds.X+100, dlg.Bounds.Y+160, 0, 240)
		tt.Advance(400 * time.Millisecond)
		firstAfter = tabInfo(t, tt, "Item A").Bounds
	}
	if absF32(firstAfter.Y-firstBefore.Y) < 1 {
		t.Fatalf("drawer list did not scroll vertically (y stayed at %.1f, before=%.1f)", firstAfter.Y, firstBefore.Y)
	}
}
