package md3e

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
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

// navInkPixels counts the pixels inside box that differ from bg.
func navInkPixels(img *image.RGBA, box ui.Rect, bg ui.Color) int {
	count := 0
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if !colorMatches(img.RGBAAt(x, y), bg, 10) {
				count++
			}
		}
	}
	return count
}

// TestNavigationRailCollapsedHoverStaysInPill checks that hovering a collapsed
// destination tints only its 56×32 pill, leaving the short label's band below
// it untouched.
func TestNavigationRailCollapsedHoverStaysInPill(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 0
	tt := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationRailOptions{Selected: &selected, Label: "Rail"})
		})
	})
	tt.Advance(400 * time.Millisecond)
	home := tabInfo(t, tt, "Home")
	before := tt.Image()

	// Hover over the selected pill itself, left of the centered icon.
	tt.Move(home.Bounds.X+4, home.Bounds.Y+20)
	tt.Advance(300 * time.Millisecond)
	after := tt.Image()

	insideBefore := before.RGBAAt(int(home.Bounds.X+4), int(home.Bounds.Y+20))
	insideAfter := after.RGBAAt(int(home.Bounds.X+4), int(home.Bounds.Y+20))
	if absColorDiff(insideBefore, insideAfter) < 4 {
		t.Fatalf("hovering the destination did not tint its pill: %v -> %v", insideBefore, insideAfter)
	}
	// The short label sits below the 32 DIP pill; the state layer must not
	// paint a 56 DIP square over it.
	for _, y := range []int{int(home.Bounds.Y + 40), int(home.Bounds.Y + 50)} {
		x := int(home.Bounds.X + 2)
		if a, b := before.RGBAAt(x, y), after.RGBAAt(x, y); a != b {
			t.Fatalf("hover leaked into the label band at (%d,%d): %v -> %v", x, y, a, b)
		}
	}
	if tt.FramePending() {
		t.Fatal("the hovered rail keeps asking for frames")
	}
}

// TestNavigationBarHoverStaysInPill checks that hovering a destination tints
// only its 64×32 pill rather than the whole 80 DIP cell, leaving the label's
// band untouched.
func TestNavigationBarHoverStaysInPill(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 0
	tt := navigationCanvas(th, 480, 320, navBar(&selected, nil, NavigationLabelsAlways))
	tt.Advance(400 * time.Millisecond)
	home := tabInfo(t, tt, "Home")
	before := tt.Image()

	tt.Move(home.Bounds.X+home.Bounds.W/2, home.Bounds.Y+28)
	tt.Advance(300 * time.Millisecond)
	after := tt.Image()

	probeX := int(home.Bounds.X+home.Bounds.W/2) - 28
	insideBefore := before.RGBAAt(probeX, int(home.Bounds.Y+24))
	insideAfter := after.RGBAAt(probeX, int(home.Bounds.Y+24))
	if absColorDiff(insideBefore, insideAfter) < 4 {
		t.Fatalf("hovering the destination did not tint its pill: %v -> %v", insideBefore, insideAfter)
	}
	x := int(home.Bounds.X + 2)
	for _, y := range []int{int(home.Bounds.Y + 58), int(home.Bounds.Y + 70)} {
		if a, b := before.RGBAAt(x, y), after.RGBAAt(x, y); a != b {
			t.Fatalf("hover leaked into the label band at (%d,%d): %v -> %v", x, y, a, b)
		}
	}
}

// TestNavigationRailMidExpansionKeepsLabelsAligned checks that a mid-transition
// frame crossfades the short and long labels in place: both are visible, the
// long label stays inside its own destination, and the icon keeps its X.
func TestNavigationRailMidExpansionKeepsLabelsAligned(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	expanded := false
	selected := 1 // Search, so Home carries no pill in the way
	tt := navigationCanvas(th, 600, 400, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, []NavigationDestination{
				{Label: "Home", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationRailOptions{Selected: &selected, Expanded: &expanded, Label: "Rail"})
		})
	})
	tt.Advance(400 * time.Millisecond)

	expanded = true
	tt.Frame()
	tt.Advance(80 * time.Millisecond)
	rail, _ := tt.AccessByRole(ui.RoleNavigation, "Rail")
	if rail.Bounds.W <= navRailCollapsedWidth+2 || rail.Bounds.W >= navRailExpandedWidth-2 {
		t.Fatalf("the rail is at %.1f, want a mid-expansion frame between %d and %d",
			rail.Bounds.W, navRailCollapsedWidth, navRailExpandedWidth)
	}
	home := tabInfo(t, tt, "Home")
	bg := SurfaceColorForElevation(th, tokens.Level1)
	img := tt.Image()

	// Both label layers show: the short one under the icon, the long one
	// beside it, neither clipped away at the item's edge.
	short := ui.Rect{
		X: home.Bounds.X + 4,
		Y: home.Bounds.Y + navRailLabelTop,
		W: home.Bounds.W - 8,
		H: 16,
	}
	if n := navInkPixels(img, short, bg); n < 2 {
		t.Fatalf("the short label paints %d pixels mid-expansion", n)
	}
	longBand := ui.Rect{
		X: home.Bounds.X + 52,
		Y: home.Bounds.Y + 20,
		W: home.Bounds.W - 56,
		H: 16,
	}
	if n := navInkPixels(img, longBand, bg); n < 2 {
		t.Fatalf("the long label paints %d pixels mid-expansion", n)
	}
	// The icon's ink stays around x = 28 of the destination.
	iconBand := ui.Rect{X: home.Bounds.X + 4, Y: home.Bounds.Y + 6, W: 44, H: 24}
	left, right := -1, -1
	for x := int(iconBand.X); x < int(iconBand.X+iconBand.W); x++ {
		for y := int(iconBand.Y); y < int(iconBand.Y+iconBand.H); y++ {
			if !colorMatches(img.RGBAAt(x, y), bg, 20) {
				if left < 0 {
					left = x
				}
				right = x
				break
			}
		}
	}
	if left < 0 {
		t.Fatal("the icon paints nothing mid-expansion")
	}
	if got := float32(left+right)/2 - home.Bounds.X; math.Abs(float64(got-28)) > 3 {
		t.Fatalf("the icon centers at %.1f mid-expansion, want 28", got)
	}
	// And the long label never spills into the destination after Home.
	if n := navInkPixels(img, ui.Rect{
		X: home.Bounds.X + home.Bounds.W + 2,
		Y: home.Bounds.Y + 20,
		W: 10,
		H: 16,
	}, bg); n != 0 {
		t.Fatalf("the label spills %d pixels into the next destination", n)
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

// TestModalDrawerOverlayIsolationAndInlineClosingInert checks clicking an
// overlay above ModalNavigationDrawer does not dismiss it via PressedOutside
// and closing an inline NavigationDrawer immediately makes destinations inert.
func TestModalDrawerOverlayIsolationAndInlineClosingInert(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := true
	snack := &SnackbarState{}
	snack.Show(SnackbarOptions{
		Message:     "Saved",
		ActionLabel: "Undo",
		Duration:    5 * time.Second,
	})
	tt := navigationCanvas(th, 600, 320, func(c *ui.Context) {
		NavigationDrawer(c, []NavigationDrawerItem{
			{Label: "Inbox", Icon: mdicon.Star.Draw},
		}, NavigationDrawerOptions{Modal: true, Open: &open, Label: "Drawer"})
		SnackbarHost(c, snack)
	})
	tt.Advance(500 * time.Millisecond)
	if !open {
		t.Fatal("modal drawer should start open")
	}
	tt.Click("Undo")
	tt.Frame()
	if !open {
		t.Fatal("clicking Snackbar action above modal drawer dismissed drawer")
	}
	// Clicking the empty drawer scrim still dismisses cleanly.
	tt.ClickAt(500, 160)
	tt.Advance(500 * time.Millisecond)
	if open {
		t.Fatal("clicking modal drawer scrim did not close drawer")
	}
	if tt.FramePending() {
		t.Fatal("closed modal drawer keeps asking for frames")
	}

	// Inline closing drawer: destinations leave access tree immediately while closing.
	inlineOpen := true
	inlineSel := 0
	inlineChanged := 0
	inTT := navigationCanvas(th, 600, 320, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationDrawer(c, []NavigationDrawerItem{
				{Label: "Inbox", Icon: mdicon.Add.Draw},
				{Label: "Sent", Icon: mdicon.Star.Draw},
			}, NavigationDrawerOptions{
				Open:     &inlineOpen,
				Selected: &inlineSel,
				Changed:  func(int) { inlineChanged++ },
				Label:    "InlineDrawer",
			})
			if Button(c, "Close").Clicked() {
				inlineOpen = false
			}
		})
	})
	inTT.Advance(500 * time.Millisecond)
	sent := tabInfo(t, inTT, "Sent")
	sx, sy := center(sent.Bounds)
	inTT.Click("Close")
	inTT.Advance(50 * time.Millisecond) // mid-collapse while openWidth > 0.5
	if _, ok := inTT.AccessByRole(ui.RoleTab, "Sent"); ok {
		t.Fatal("closing inline drawer kept Sent destination in access tree")
	}
	inTT.ClickAt(sx, sy)
	inTT.Advance(500 * time.Millisecond)
	if inlineSel != 0 || inlineChanged != 0 {
		t.Fatalf("closing inline drawer still accepted selection (%d, changed=%d)", inlineSel, inlineChanged)
	}
}

// TestNavigationRailRTLPillAndNarrowNavBarClamp checks RTL mid-expansion pill
// alignment and label bounds in NavigationRail and pill/label clamping in a
// narrow 5-destination NavigationBar.
func TestNavigationRailRTLPillAndNarrowNavBarClamp(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	expanded := false
	selected := 0
	rtlTT := navigationCanvas(th, 400, 320, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, []NavigationDestination{
				{Label: "Very Long Destination Title", Icon: mdicon.Add.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
			}, NavigationRailOptions{Selected: &selected, Expanded: &expanded, Label: "RTLRail"})
		})
	})
	rtlTT.SetDirection(ui.RightToLeft)
	rtlTT.Advance(400 * time.Millisecond)
	expanded = true
	rtlTT.Frame()
	rtlTT.Advance(80 * time.Millisecond)
	home := tabInfo(t, rtlTT, "Very Long Destination Title")
	bg := SurfaceColorForElevation(th, tokens.Level1)
	img := rtlTT.Image()
	// Pill under selected RTL icon on the right-hand Start(0) slot must paint
	// SecondaryContainer near the right edge of home.Bounds.
	rightSlotPx := img.RGBAAt(int(home.Bounds.X+home.Bounds.W-10), int(home.Bounds.Y+16))
	if !colorMatches(rightSlotPx, cs.SecondaryContainer, 18) {
		t.Fatalf("RTL rail pill missed right Start(0) slot mid-expansion: %+v", rightSlotPx)
	}
	// Long label in expanded state must not spill outside home.Bounds.
	rtlTT.Advance(600 * time.Millisecond)
	home = tabInfo(t, rtlTT, "Very Long Destination Title")
	if home.Bounds.X < 0 || home.Bounds.W > navRailExpandedWidth {
		t.Fatalf("RTL rail destination spilled bounds: %+v", home.Bounds)
	}
	if rtlTT.FramePending() {
		t.Fatal("RTL rail kept requesting frames after expansion")
	}

	// Narrow 240-wide bar with 5 items -> each cell is 48 DIP (< 64 DIP).
	barSel := 2
	narrowTT := navigationCanvas(th, 240, 160, func(c *ui.Context) {
		ui.Column(c).FillHeight().Justify(ui.End).Children(func() {
			NavigationBar(c, []NavigationDestination{
				{Label: "Dashboard", Icon: mdicon.Add.Draw},
				{Label: "Messages", Icon: mdicon.Check.Draw},
				{Label: "Favorites", Icon: mdicon.Star.Draw},
				{Label: "Search", Icon: mdicon.Search.Draw},
				{Label: "Settings", Icon: mdicon.Add.Draw},
			}, NavigationBarOptions{Selected: &barSel, Label: "NarrowBar"})
		})
	})
	narrowTT.Advance(400 * time.Millisecond)
	fav := tabInfo(t, narrowTT, "Favorites")
	msg := tabInfo(t, narrowTT, "Messages")
	// Check pill color does not spill past Favorites left/right cell boundaries into Messages.
	if px := narrowTT.Image().RGBAAt(int(msg.Bounds.X+msg.Bounds.W/2), int(fav.Bounds.Y+navBarIndicatorTop+12)); colorMatches(px, cs.SecondaryContainer, 14) {
		t.Fatalf("narrow bar indicator pill spilled from Favorites (%+v) into Messages (%+v): %+v", fav.Bounds, msg.Bounds, px)
	}
	_ = bg
}

func TestNavigationRailDensityAndExpandedWidth(t *testing.T) {
	cases := []struct {
		name     string
		density  tokens.Density
		override float32
		wantW    float32
	}{
		{"ComfortableDefault", tokens.DensityComfortable, 0, 256},
		{"DesktopDefault", tokens.DensityDesktop, 0, 192},
		{"CompactDefault", tokens.DensityCompact, 0, 180},
		{"ExplicitOverride", tokens.DensityDesktop, 220, 220},
		{"NaNFallback", tokens.DensityDesktop, float32(math.NaN()), 192},
		{"InfFallback", tokens.DensityCompact, float32(math.Inf(1)), 180},
		{"TooSmallClamped", tokens.DensityDesktop, 40, navRailCollapsedWidth},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{
				Density:      tc.density,
				ReduceMotion: true,
			})
			expanded := true
			selected := 0
			tt := navigationCanvas(th, 400, 300, func(c *ui.Context) {
				ui.Row(c).FillHeight().Children(func() {
					NavigationRail(c, []NavigationDestination{
						{Label: "Home", Icon: mdicon.Add.Draw},
						{Label: "Search", Icon: mdicon.Search.Draw},
					}, NavigationRailOptions{
						Selected:      &selected,
						Expanded:      &expanded,
						ExpandedWidth: tc.override,
						Label:         "Rail",
					})
				})
			})
			if tt.FramePending() {
				t.Fatal("ReduceMotion rail should settle immediately without pending frames")
			}
			rail, ok := tt.AccessByRole(ui.RoleNavigation, "Rail")
			if !ok {
				t.Fatal("NavigationRail missing from access tree")
			}
			if absF32(rail.Bounds.W-tc.wantW) > 0.5 {
				t.Fatalf("expanded rail width = %.1f, want %.1f", rail.Bounds.W, tc.wantW)
			}

			// Verify RTL layout and collapse transition settle cleanly.
			tt.SetDirection(ui.RightToLeft)
			expanded = false
			tt.Frame()
			if tt.FramePending() {
				t.Fatal("ReduceMotion RTL collapse left pending frames")
			}
			rail, _ = tt.AccessByRole(ui.RoleNavigation, "Rail")
			if absF32(rail.Bounds.W-navRailCollapsedWidth) > 0.5 {
				t.Fatalf("collapsed rail width = %.1f, want %v", rail.Bounds.W, navRailCollapsedWidth)
			}
		})
	}
}

// verticalColorSpan returns the vertical [minY, maxY] extent of pixels matching
// want within tolerance tol inside the column strip [x0, x1) x [y0, y1).
func verticalColorSpan(img *image.RGBA, x0, x1, y0, y1 int, want ui.Color, tol int) (minY, maxY int, count int) {
	minY, maxY = -1, -1
	b := img.Bounds()
	x0, x1 = max(x0, b.Min.X), min(x1, b.Max.X)
	y0, y1 = max(y0, b.Min.Y), min(y1, b.Max.Y)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if colorMatches(img.RGBAAt(x, y), want, tol) {
				if minY < 0 {
					minY = y
				}
				maxY = y
				count++
			}
		}
	}
	return minY, maxY, count
}

// TestNavigationRailSlidingIndicatorAndAccentBar checks that switching
// NavigationRail selection smoothly slides the shared indicator vertically
// between destinations (never snapping/fading in place), tracks both collapsed
// and expanded states, paints and mirrors the optional leading accent bar in
// LTR/RTL, follows scroll offsets, and snaps immediately under ReduceMotion.
func TestNavigationRailSlidingIndicatorAndAccentBar(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	destinations := []NavigationDestination{
		{Label: "Home", Icon: mdicon.Add.Draw},
		{Label: "Search", Icon: mdicon.Search.Draw},
		{Label: "Saved", Icon: mdicon.Star.Draw},
		{Label: "Settings", Icon: mdicon.Edit.Draw},
	}

	// 1. Collapsed rail: first frame lands on item 0 without flying from 0;
	// switching 0 -> 3 produces intermediate Y coordinates strictly between
	// item 0 and item 3 before settling with FramePending() == false.
	sel := 0
	tt := navigationCanvas(th, 320, 360, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, destinations, NavigationRailOptions{
				Selected: &sel,
				Label:    "SlidingRail",
			})
		})
	})
	if tt.FramePending() {
		t.Fatal("initial NavigationRail frame should land immediately without pending frames")
	}
	home := tabInfo(t, tt, "Home")
	settings := tabInfo(t, tt, "Settings")
	probeX0, probeX1 := int(home.Bounds.X+4), int(home.Bounds.X+12)
	y0Min, y0Max, c0 := verticalColorSpan(tt.Image(), probeX0, probeX1, 0, 360, cs.SecondaryContainer, 14)
	if c0 == 0 {
		t.Fatal("initial collapsed rail pill did not paint on item 0")
	}
	startCenterY := float32(y0Min+y0Max) / 2

	sel = 3
	tt.Frame()
	if !tt.FramePending() {
		t.Fatal("switching NavigationRail destination did not schedule sliding animation frames")
	}
	tt.Advance(48 * time.Millisecond)
	midMin, midMax, midCount := verticalColorSpan(tt.Image(), probeX0, probeX1, 0, 360, cs.SecondaryContainer, 14)
	if midCount == 0 {
		t.Fatal("mid-flight collapsed rail sliding pill disappeared")
	}
	midCenterY := float32(midMin+midMax) / 2
	targetPillCenterY := settings.Bounds.Y + navRailIconTop + float32(navRailIndicatorHigh)/2
	if !(midCenterY > startCenterY+10 && midCenterY < targetPillCenterY-10) {
		t.Fatalf("collapsed pill mid-flight center Y = %.1f, want strictly between start %.1f and target %.1f",
			midCenterY, startCenterY, targetPillCenterY)
	}

	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("collapsed NavigationRail kept requesting frames after sliding settled")
	}
	endMin, endMax, endCount := verticalColorSpan(tt.Image(), probeX0, probeX1, 0, 360, cs.SecondaryContainer, 14)
	if endCount == 0 {
		t.Fatal("settled collapsed rail pill missing on item 3")
	}
	endCenterY := float32(endMin+endMax) / 2
	if absF32(endCenterY-targetPillCenterY) > 2.5 {
		t.Fatalf("settled collapsed pill center Y = %.1f, want %.1f", endCenterY, targetPillCenterY)
	}
	if _, _, leftover := verticalColorSpan(tt.Image(), probeX0, probeX1, int(home.Bounds.Y), int(home.Bounds.Y+home.Bounds.H), cs.SecondaryContainer, 14); leftover != 0 {
		t.Fatalf("old item 0 still painted %d pill pixels after settling on item 3", leftover)
	}

	// 2. Expanded rail with NavigationRailIndicatorPillWithBar: both pill and
	// leading Primary accent bar glide together, and the bar mirrors to the
	// right edge in RTL.
	expSel := 0
	expanded := true
	barTT := navigationCanvas(th, 360, 360, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, destinations, NavigationRailOptions{
				Selected:       &expSel,
				Expanded:       &expanded,
				IndicatorStyle: NavigationRailIndicatorPillWithBar,
				Label:          "BarRail",
			})
		})
	})
	if barTT.FramePending() {
		t.Fatal("initial expanded NavigationRail should start settled")
	}
	railInfo, _ := barTT.AccessByRole(ui.RoleNavigation, "BarRail")
	leftStripX0, leftStripX1 := int(railInfo.Bounds.X+3), int(railInfo.Bounds.X+9)
	rightStripX0, rightStripX1 := int(railInfo.Bounds.X+railInfo.Bounds.W-9), int(railInfo.Bounds.X+railInfo.Bounds.W-3)

	b0Min, b0Max, b0Count := verticalColorSpan(barTT.Image(), leftStripX0, leftStripX1, 0, 360, cs.Primary, 16)
	if b0Count == 0 {
		t.Fatal("LTR NavigationRailIndicatorPillWithBar did not paint leading Primary accent bar")
	}
	if _, _, rightCount := verticalColorSpan(barTT.Image(), rightStripX0, rightStripX1, 0, 360, cs.Primary, 16); rightCount != 0 {
		t.Fatalf("LTR accent bar leaked %d pixels onto right edge", rightCount)
	}
	barStartY := float32(b0Min+b0Max) / 2
	expSettings := tabInfo(t, barTT, "Settings")
	barTargetY := expSettings.Bounds.Y + expSettings.Bounds.H/2

	expSel = 3
	barTT.Frame()
	barTT.Advance(48 * time.Millisecond)
	bMidMin, bMidMax, bMidCount := verticalColorSpan(barTT.Image(), leftStripX0, leftStripX1, 0, 360, cs.Primary, 16)
	if bMidCount == 0 {
		t.Fatal("mid-flight leading accent bar missing during vertical slide")
	}
	barMidY := float32(bMidMin+bMidMax) / 2
	if !(barMidY > barStartY+10 && barMidY < barTargetY-10) {
		t.Fatalf("accent bar mid-flight center Y = %.1f, want strictly between %.1f and %.1f", barMidY, barStartY, barTargetY)
	}
	barTT.Advance(time.Second)
	if barTT.FramePending() {
		t.Fatal("expanded NavigationRail with accent bar kept requesting frames after settle")
	}
	bEndMin, bEndMax, bEndCount := verticalColorSpan(barTT.Image(), leftStripX0, leftStripX1, 0, 360, cs.Primary, 16)
	if bEndCount == 0 || absF32(float32(bEndMin+bEndMax)/2-barTargetY) > 2.5 {
		t.Fatalf("settled LTR accent bar center Y = %.1f (count=%d), want %.1f", float32(bEndMin+bEndMax)/2, bEndCount, barTargetY)
	}

	// Switch to RTL: accent bar must mirror to the right edge and slide back cleanly.
	barTT.SetDirection(ui.RightToLeft)
	barTT.Advance(100 * time.Millisecond)
	rtlRail, _ := barTT.AccessByRole(ui.RoleNavigation, "BarRail")
	rtlLeftX0, rtlLeftX1 := int(rtlRail.Bounds.X+3), int(rtlRail.Bounds.X+9)
	rtlRightX0, rtlRightX1 := int(rtlRail.Bounds.X+rtlRail.Bounds.W-9), int(rtlRail.Bounds.X+rtlRail.Bounds.W-3)
	if _, _, rc := verticalColorSpan(barTT.Image(), rtlRightX0, rtlRightX1, int(expSettings.Bounds.Y), int(expSettings.Bounds.Y+expSettings.Bounds.H), cs.Primary, 16); rc == 0 {
		t.Fatal("RTL NavigationRailIndicatorPillWithBar did not mirror accent bar to the right edge")
	}
	if _, _, lc := verticalColorSpan(barTT.Image(), rtlLeftX0, rtlLeftX1, 0, 360, cs.Primary, 16); lc != 0 {
		t.Fatalf("RTL accent bar still painted %d pixels on the left edge", lc)
	}
	expSel = 0
	barTT.Frame()
	barTT.Advance(48 * time.Millisecond)
	rbMin, rbMax, rbCount := verticalColorSpan(barTT.Image(), rtlRightX0, rtlRightX1, 0, 360, cs.Primary, 16)
	if rbCount == 0 || !(float32(rbMin+rbMax)/2 > barStartY+10 && float32(rbMin+rbMax)/2 < barTargetY-10) {
		t.Fatalf("RTL accent bar did not slide smoothly back toward item 0: span=[%d,%d] count=%d", rbMin, rbMax, rbCount)
	}
	barTT.Advance(time.Second)
	if barTT.FramePending() {
		t.Fatal("RTL NavigationRail kept requesting frames after sliding back")
	}

	// 3. ReduceMotion: switching selection, deselecting (-1), and reselecting
	// snap immediately with zero pending frames.
	rmTheme := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	rmSel := 0
	rmExp := true
	rmTT := navigationCanvas(rmTheme, 360, 360, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, destinations, NavigationRailOptions{
				Selected:       &rmSel,
				Expanded:       &rmExp,
				IndicatorStyle: NavigationRailIndicatorPillWithBar,
				Label:          "RMRail",
			})
		})
	})
	if rmTT.FramePending() {
		t.Fatal("ReduceMotion rail requested frames on initial mount")
	}
	rmSettings := tabInfo(t, rmTT, "Settings")
	rmSel = 3
	rmTT.Frame()
	if rmTT.FramePending() {
		t.Fatal("ReduceMotion rail selection switch left frames pending")
	}
	if _, _, cnt := verticalColorSpan(rmTT.Image(), 3, 9, int(rmSettings.Bounds.Y), int(rmSettings.Bounds.Y+rmSettings.Bounds.H), rmTheme.Colors.Primary, 16); cnt == 0 {
		t.Fatal("ReduceMotion rail accent bar did not snap immediately to item 3")
	}
	rmSel = -1
	rmTT.Frame()
	if rmTT.FramePending() {
		t.Fatal("ReduceMotion rail deselection left frames pending")
	}
	if _, _, cnt := verticalColorSpan(rmTT.Image(), 3, 9, 0, 360, rmTheme.Colors.Primary, 16); cnt != 0 {
		t.Fatalf("deselected rail still painted %d accent bar pixels", cnt)
	}
	rmSel = 2
	rmTT.Frame()
	if rmTT.FramePending() {
		t.Fatal("ReduceMotion rail reselection left frames pending")
	}
	rmSaved := tabInfo(t, rmTT, "Saved")
	if _, _, cnt := verticalColorSpan(rmTT.Image(), 3, 9, int(rmSaved.Bounds.Y), int(rmSaved.Bounds.Y+rmSaved.Bounds.H), rmTheme.Colors.Primary, 16); cnt == 0 {
		t.Fatal("ReduceMotion rail accent bar did not snap immediately on reselect")
	}

	// 4. Scrolled viewport: indicator and accent bar follow scroll offsets cleanly.
	scrollDests := []NavigationDestination{
		{Label: "Home", Icon: mdicon.Add.Draw},
		{Label: "Search", Icon: mdicon.Search.Draw},
		{Label: "Saved", Icon: mdicon.Star.Draw},
		{Label: "Library", Icon: mdicon.Check.Draw},
		{Label: "Inbox", Icon: mdicon.MoreHoriz.Draw},
		{Label: "Help", Icon: mdicon.Edit.Draw},
	}
	scrollSel := 0
	scrollExp := true
	scrollTT := navigationCanvas(th, 320, 220, func(c *ui.Context) {
		ui.Row(c).FillHeight().Children(func() {
			NavigationRail(c, scrollDests, NavigationRailOptions{
				Selected:       &scrollSel,
				Expanded:       &scrollExp,
				IndicatorStyle: NavigationRailIndicatorPillWithBar,
				Label:          "ScrollRail",
			})
		})
	})
	scrollRail, _ := scrollTT.AccessByRole(ui.RoleNavigation, "ScrollRail")
	scrollTT.Wheel(scrollRail.Bounds.X+scrollRail.Bounds.W/2, scrollRail.Bounds.Y+scrollRail.Bounds.H/2, 0, 300)
	scrollTT.Advance(400 * time.Millisecond)
	if err := scrollTT.Click("Help"); err != nil {
		t.Fatalf("clicking scrolled Help destination: %v", err)
	}
	scrollTT.Advance(time.Second)
	if scrollSel != len(scrollDests)-1 {
		t.Fatalf("scrolled rail selected %d, want %d", scrollSel, len(scrollDests)-1)
	}
	if scrollTT.FramePending() {
		t.Fatal("scrolled rail kept requesting frames after settle")
	}
	helpInfo := tabInfo(t, scrollTT, "Help")
	if _, _, cnt := verticalColorSpan(scrollTT.Image(), int(scrollRail.Bounds.X+3), int(scrollRail.Bounds.X+9), int(helpInfo.Bounds.Y), int(helpInfo.Bounds.Y+helpInfo.Bounds.H), cs.Primary, 16); cnt == 0 {
		t.Fatalf("scrolled rail accent bar missed Help bounds %+v", helpInfo.Bounds)
	}

	// 5. Exit ghost safety: mounting NavigationRail inside an ElementTransition
	// that exits must keep drawing its frozen snapshot cleanly and settle.
	showRail := true
	ghostSel := 1
	ghostTT := navigationCanvas(th, 320, 260, func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			if showRail {
				NavigationRail(c.Key("ghost-rail"), destinations, NavigationRailOptions{
					Selected:       &ghostSel,
					IndicatorStyle: NavigationRailIndicatorPillWithBar,
					Label:          "GhostRail",
				}).Transition(ui.ElementTransition{
					Duration: 150 * time.Millisecond,
					Exit:     &ui.Motion{Opacity: 0},
				})
			}
		})
	})
	showRail = false
	ghostTT.Frame()
	for range 5 {
		ghostTT.Advance(30 * time.Millisecond)
	}
	ghostTT.Advance(300 * time.Millisecond)
	if ghostTT.FramePending() {
		t.Fatal("exiting NavigationRail ghost kept requesting frames after settle")
	}
}
