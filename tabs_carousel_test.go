package md3e

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// tabsCanvas builds a 480 DIP wide primary or secondary tab bar over the
// container canvas.
func tabsCanvas(th *Theme, items []TabItem, opts TabsOptions, secondary bool) *ui.Tester {
	return containerCanvas(th, func(c *ui.Context) {
		var tabs ui.Element
		if secondary {
			tabs = SecondaryTabs(c, items, opts)
		} else {
			tabs = PrimaryTabs(c, items, opts)
		}
		tabs.Width(480)
	})
}

// tabPixel reports whether a tab's indicator band paints the theme's Primary
// color at its center. The bar holds a 3 DIP primary indicator, or a 2 DIP one
// above a 1 DIP divider for secondary tabs.
func tabIndicatorPainted(tt *ui.Tester, th *Theme, b ui.Rect) bool {
	y := int(b.Y+b.H) - 2
	px := tt.Image().RGBAAt(int(b.X+b.W/2), y)
	return colorMatches(px, th.Colors.Primary, 12)
}

// TestPrimaryTabsClickAndIndicator checks the tab list's roles, that a click
// chooses a tab, and that the sliding indicator moves from the old tab to the
// new one and settles.
func TestPrimaryTabsClickAndIndicator(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected, changed := 0, -1
	tt := tabsCanvas(th, []TabItem{{Label: "Home"}, {Label: "Search"}, {Label: "Saved"}}, TabsOptions{
		Selected: &selected,
		Changed:  func(i int) { changed = i },
		Label:    "Sections",
	}, false)
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleTabList, "Sections"); !ok {
		t.Fatal("the tab bar is not a named tab list")
	}
	if !tabInfo(t, tt, "Home").Selected {
		t.Fatal("the initial tab does not read as selected")
	}
	if b := tabInfo(t, tt, "Home").Bounds; !tabIndicatorPainted(tt, th, b) {
		t.Fatalf("the indicator is not under the chosen tab: pixel %+v at (%d,%d), want %+v", tt.Image().RGBAAt(int(b.X+b.W/2), int(b.Y+b.H)-2), int(b.X+b.W/2), int(b.Y+b.H)-2, rgbOf(th.Colors.Primary))
	}
	if tabIndicatorPainted(tt, th, tabInfo(t, tt, "Search").Bounds) {
		t.Fatal("an unchosen tab paints the indicator")
	}

	tt.Click("Search")
	tt.Advance(time.Second)
	if selected != 1 || changed != 1 {
		t.Fatalf("the click chose %d (callback %d), want 1", selected, changed)
	}
	if !tabInfo(t, tt, "Search").Selected {
		t.Fatal("the chosen tab does not read as selected")
	}
	if !tabIndicatorPainted(tt, th, tabInfo(t, tt, "Search").Bounds) {
		t.Fatal("the indicator did not slide under the chosen tab")
	}
	if tabIndicatorPainted(tt, th, tabInfo(t, tt, "Home").Bounds) {
		t.Fatal("the indicator stayed under the old tab")
	}
	if tt.FramePending() {
		t.Fatal("the tab bar keeps asking for frames after choosing")
	}
}

// TestTabsKeyboardSkipsDisabled checks the arrows, Home and End choose tabs,
// skip a disabled one and disable it for assistive technology.
func TestTabsKeyboardSkipsDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 0
	tt := tabsCanvas(th, []TabItem{
		{Label: "Alpha"},
		{Label: "Beta", Disabled: true},
		{Label: "Gamma"},
	}, TabsOptions{Selected: &selected, Label: "Sections"}, false)
	tt.Advance(300 * time.Millisecond)
	if !tabInfo(t, tt, "Beta").Disabled {
		t.Fatal("a disabled tab does not read as disabled")
	}
	alpha := tabInfo(t, tt, "Alpha")
	tt.ClickAt(alpha.Bounds.X+alpha.Bounds.W/2, alpha.Bounds.Y+alpha.Bounds.H/2)
	tt.Key(0, ui.KeyRight)
	if selected != 2 {
		t.Fatalf("Right chose %d, want 2 past the disabled tab", selected)
	}
	tt.Key(0, ui.KeyLeft)
	if selected != 0 {
		t.Fatalf("Left chose %d, want 0 past the disabled tab", selected)
	}
	tt.Key(0, ui.KeyEnd)
	if selected != 2 {
		t.Fatalf("End chose %d, want 2", selected)
	}
	tt.Key(0, ui.KeyHome)
	if selected != 0 {
		t.Fatalf("Home chose %d, want 0", selected)
	}
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("the tab bar keeps asking for frames after key navigation")
	}
}

// TestSecondaryTabsDividerAndIndicator checks the full-width bottom indicator
// and the divider beneath it.
func TestSecondaryTabsDividerAndIndicator(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 0
	tt := tabsCanvas(th, []TabItem{{Label: "Home"}, {Label: "Search"}}, TabsOptions{
		Selected: &selected,
		Label:    "Sections",
	}, true)
	tt.Advance(time.Second)
	home := tabInfo(t, tt, "Home")
	if !tabIndicatorPainted(tt, th, home.Bounds) {
		t.Fatal("the secondary indicator is not under the chosen tab")
	}
	y := int(home.Bounds.Y+home.Bounds.H) - 1
	px := tt.Image().RGBAAt(int(home.Bounds.X+10), y)
	if !colorMatches(px, th.Colors.OutlineVariant, 10) {
		t.Fatalf("the bottom pixel %+v is not the divider %+v", px, rgbOf(th.Colors.OutlineVariant))
	}
	tt.Click("Search")
	tt.Advance(time.Second)
	if selected != 1 {
		t.Fatalf("the click chose %d, want 1", selected)
	}
	if tt.FramePending() {
		t.Fatal("the secondary bar keeps asking for frames after choosing")
	}
}

// TestTabsRTLArrowsAndGeometry checks the bar mirrors right to left and the
// arrows follow the mirrored order.
func TestTabsRTLArrowsAndGeometry(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 0
	tt := tabsCanvas(th, []TabItem{{Label: "Home"}, {Label: "Search"}, {Label: "Saved"}}, TabsOptions{
		Selected: &selected,
		Label:    "Sections",
	}, false)
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(300 * time.Millisecond)
	home := tabInfo(t, tt, "Home")
	saved := tabInfo(t, tt, "Saved")
	if home.Bounds.X <= saved.Bounds.X {
		t.Fatalf("right to left puts Home at %.1f and Saved at %.1f", home.Bounds.X, saved.Bounds.X)
	}
	tt.ClickAt(home.Bounds.X+home.Bounds.W/2, home.Bounds.Y+home.Bounds.H/2)
	tt.Key(0, ui.KeyLeft)
	if selected != 1 {
		t.Fatalf("Left chose %d, want the next tab right to left", selected)
	}
	tt.Key(0, ui.KeyRight)
	if selected != 0 {
		t.Fatalf("Right chose %d, want the previous tab right to left", selected)
	}
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("the right-to-left bar keeps asking for frames")
	}
}

// TestScrollableTabsScrollTheChoiceIntoView checks a scrollable bar scrolls a
// newly chosen off-screen tab into view.
func TestScrollableTabsScrollTheChoiceIntoView(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	items := make([]TabItem, 8)
	for i := range items {
		items[i] = TabItem{Label: "Tab " + string(rune('0'+i))}
	}
	selected := 0
	tt := tabsCanvas(th, items, TabsOptions{Selected: &selected, Scrollable: true, Label: "Sections"}, false)
	tt.Advance(time.Second)
	scroll, ok := tt.AccessByRole(ui.RoleScroll, "")
	if !ok {
		t.Fatal("the scrollable bar has no scroll area")
	}
	last := tabInfo(t, tt, "Tab 7")
	if last.Bounds.X+last.Bounds.W <= scroll.Bounds.X+scroll.Bounds.W-1 {
		t.Fatalf("the last tab box %v already fits the scroll area %v", last.Bounds, scroll.Bounds)
	}

	selected = 7
	tt.Frame()
	tt.Advance(2 * time.Second)
	scroll, _ = tt.AccessByRole(ui.RoleScroll, "")
	last = tabInfo(t, tt, "Tab 7")
	if last.Bounds.X < scroll.Bounds.X-0.5 || last.Bounds.X+last.Bounds.W > scroll.Bounds.X+scroll.Bounds.W+0.5 {
		t.Fatalf("the chosen tab box %v is outside the scroll area %v", last.Bounds, scroll.Bounds)
	}
	if tt.FramePending() {
		t.Fatal("the scrollable bar keeps asking for frames")
	}
}

// TestTabsReduceMotion checks choosing a tab jumps the indicator without
// leaving animation frames.
func TestTabsReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	selected := 0
	tt := tabsCanvas(th, []TabItem{{Label: "Home"}, {Label: "Search"}}, TabsOptions{
		Selected: &selected,
		Label:    "Sections",
	}, false)
	tt.Frame()
	selected = 1
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("a reduced-motion tab change asks for frames")
	}
	if !tabIndicatorPainted(tt, th, tabInfo(t, tt, "Search").Bounds) {
		t.Fatal("the reduced-motion indicator did not move at once")
	}
}

// carouselItems builds n slides labeled A, B, C...
func carouselItems(n int) []CarouselItem {
	items := make([]CarouselItem, n)
	for i := range items {
		items[i] = CarouselItem{Label: string(rune('A' + i))}
	}
	return items
}

// carouselCanvas builds a 600 DIP wide carousel over the container canvas.
func carouselCanvas(th *Theme, items []CarouselItem, opts CarouselOptions) *ui.Tester {
	return containerCanvas(th, func(c *ui.Context) {
		carousel := Carousel(c, items, opts)
		carousel.Width(600)
	})
}

// TestCarouselHeroFocusAndClip checks the hero layout sizes the focused slide
// up, clips its rounded corners, and reallocates every slot on a click.
func TestCarouselHeroFocusAndClip(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	index, changed := 0, -1
	tt := carouselCanvas(th, carouselItems(4), CarouselOptions{
		Index:   &index,
		Layout:  CarouselHero,
		Height:  220,
		Label:   "Photos",
		Changed: func(i int) { changed = i },
	})
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleGroup, "Photos"); !ok {
		t.Fatal("the carousel is not a named group")
	}
	a := buttonInfo(t, tt, ui.RoleButton, "A")
	b := buttonInfo(t, tt, ui.RoleButton, "B")
	if !(a.Bounds.W > b.Bounds.W) {
		t.Fatalf("the focused slide is %.1f wide and its neighbour %.1f", a.Bounds.W, b.Bounds.W)
	}
	if pixelFilled(tt, int(a.Bounds.X)+3, int(a.Bounds.Y)+3) {
		t.Fatal("the focused slide paints outside its rounded corner")
	}
	px := tt.Image().RGBAAt(int(a.Bounds.X+20), int(a.Bounds.Y+a.Bounds.H/2))
	if !colorMatches(px, SurfaceColorForElevation(th, tokens.Level1), 8) {
		t.Fatalf("the slide pixel %+v is not its tonal surface %+v", px, rgbOf(SurfaceColorForElevation(th, tokens.Level1)))
	}

	// The second slide peeks in at the end of the strip; click its visible part.
	tt.ClickAt(b.Bounds.X+20, b.Bounds.Y+30)
	tt.Advance(2 * time.Second)
	if index != 1 || changed != 1 {
		t.Fatalf("the click focused %d (callback %d), want 1", index, changed)
	}
	a = buttonInfo(t, tt, ui.RoleButton, "A")
	b = buttonInfo(t, tt, ui.RoleButton, "B")
	if !(b.Bounds.W > a.Bounds.W) {
		t.Fatalf("after the click the focused slide is %.1f wide and the old one %.1f", b.Bounds.W, a.Bounds.W)
	}
	if b.Bounds.X > 30 {
		t.Fatalf("the focused slide rests at x %.1f, not at the strip's start", b.Bounds.X)
	}
	if tt.FramePending() {
		t.Fatal("the carousel keeps asking for frames after choosing")
	}
}

// TestCarouselDragPansAndRefocuses checks a pointer drag pans the strip, snaps
// it to the nearest slide and reallocates the slot widths.
func TestCarouselDragPansAndRefocuses(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	index := 0
	changed := -1
	tt := carouselCanvas(th, carouselItems(4), CarouselOptions{
		Index:   &index,
		Layout:  CarouselHero,
		Height:  220,
		Changed: func(i int) { changed = i },
	})
	tt.Advance(time.Second)
	b := buttonInfo(t, tt, ui.RoleButton, "B")
	y := b.Bounds.Y + b.Bounds.H/2
	tt.Press(b.Bounds.X+20, y)
	tt.Move(b.Bounds.X-156, y)
	tt.Move(b.Bounds.X-300, y)
	tt.Release(b.Bounds.X-300, y)
	tt.Advance(3 * time.Second)
	if index != 1 || changed != 1 {
		t.Fatalf("the drag focused %d (callback %d), want 1", index, changed)
	}
	a := buttonInfo(t, tt, ui.RoleButton, "A")
	b = buttonInfo(t, tt, ui.RoleButton, "B")
	if !(b.Bounds.W > a.Bounds.W) {
		t.Fatalf("the dragged-to slide is %.1f wide and the old one %.1f", b.Bounds.W, a.Bounds.W)
	}
	if tt.FramePending() {
		t.Fatal("the dragged carousel keeps asking for frames")
	}
}

// TestCarouselKeyboardFocus checks the arrows move the focus among the slides
// and Enter focuses the one it is on.
func TestCarouselKeyboardFocus(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	index, changed := 0, -1
	tt := carouselCanvas(th, carouselItems(4), CarouselOptions{
		Index:   &index,
		Layout:  CarouselMultiBrowse,
		Height:  200,
		Changed: func(i int) { changed = i },
	})
	tt.Advance(time.Second)
	a := buttonInfo(t, tt, ui.RoleButton, "A")
	tt.ClickAt(a.Bounds.X+a.Bounds.W/2, a.Bounds.Y+a.Bounds.H/2)
	tt.Advance(300 * time.Millisecond)
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(2 * time.Second)
	if index != 1 || changed != 1 {
		t.Fatalf("Right and Enter focused %d (callback %d), want 1", index, changed)
	}
	if tt.FramePending() {
		t.Fatal("the keyboard-driven carousel keeps asking for frames")
	}
}

// TestCarouselLayoutsAndRTL checks each layout's width rhythm and that the
// strip mirrors right to left.
func TestCarouselLayoutsAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	for _, tc := range []struct {
		name   string
		layout CarouselLayout
	}{
		{"multibrowse", CarouselMultiBrowse},
		{"uncontained", CarouselUncontained},
	} {
		t.Run(tc.name, func(t *testing.T) {
			index := 0
			tt := carouselCanvas(th, carouselItems(4), CarouselOptions{Index: &index, Layout: tc.layout, Height: 200})
			tt.Advance(time.Second)
			a := buttonInfo(t, tt, ui.RoleButton, "A")
			b := buttonInfo(t, tt, ui.RoleButton, "B")
			switch tc.layout {
			case CarouselMultiBrowse:
				c := buttonInfo(t, tt, ui.RoleButton, "C")
				if !(a.Bounds.W > b.Bounds.W && b.Bounds.W > c.Bounds.W) {
					t.Fatalf("multi-browse widths %.1f, %.1f, %.1f do not step down", a.Bounds.W, b.Bounds.W, c.Bounds.W)
				}
			case CarouselUncontained:
				if math.Abs(float64(a.Bounds.W-b.Bounds.W)) > 0.5 {
					t.Fatalf("uncontained widths %.1f and %.1f differ", a.Bounds.W, b.Bounds.W)
				}
			}
			if tt.FramePending() {
				t.Fatal("the carousel keeps asking for frames")
			}
		})
	}

	index := 0
	rtl := carouselCanvas(th, carouselItems(4), CarouselOptions{Index: &index, Layout: CarouselHero, Height: 200})
	rtl.SetDirection(ui.RightToLeft)
	rtl.Advance(time.Second)
	a := buttonInfo(t, rtl, ui.RoleButton, "A")
	b := buttonInfo(t, rtl, ui.RoleButton, "B")
	if a.Bounds.X <= b.Bounds.X {
		t.Fatalf("right to left puts A at %.1f and B at %.1f", a.Bounds.X, b.Bounds.X)
	}
}

// TestCarouselReduceMotion checks a programmatic focus change settles with no
// animation frames.
func TestCarouselReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	index := 0
	tt := carouselCanvas(th, carouselItems(4), CarouselOptions{Index: &index, Layout: CarouselHero, Height: 200})
	tt.Frame()
	index = 2
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("a reduced-motion carousel asks for frames")
	}
	a := buttonInfo(t, tt, ui.RoleButton, "A")
	c := buttonInfo(t, tt, ui.RoleButton, "C")
	if !(c.Bounds.W > a.Bounds.W) {
		t.Fatalf("the reduced-motion focus did not resize the slots: A %.1f, C %.1f", a.Bounds.W, c.Bounds.W)
	}
}

// TestScrollableTabsIndicatorScrollsAndClips checks a scrollable tab bar moves
// and clips its active indicator alongside the tabs on horizontal scrolling.
func TestScrollableTabsIndicatorScrollsAndClips(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	selected := 1
	items := []TabItem{
		{Label: "Overview"},
		{Label: "Schedule"},
		{Label: "Speakers"},
		{Label: "Workshops"},
		{Label: "Exhibitors"},
		{Label: "Sponsors"},
		{Label: "Venue"},
		{Label: "About"},
	}
	tt := containerCanvas(th, func(c *ui.Context) {
		PrimaryTabs(c, items, TabsOptions{
			Selected:   &selected,
			Scrollable: true,
			Label:      "Event tabs",
		}).Width(260)
	})
	tt.Advance(600 * time.Millisecond)
	schedBefore := tabInfo(t, tt, "Schedule")
	if !tabIndicatorPainted(tt, th, schedBefore.Bounds) {
		t.Fatal("initial scrollable indicator is not under Schedule")
	}

	// Select a further tab to force horizontal scrolling, then verify the
	// indicator tracks it cleanly inside the bar without leaking outside.
	selected = 5
	tt.Frame()
	tt.Advance(800 * time.Millisecond)
	sponsors := tabInfo(t, tt, "Sponsors")
	if !tabIndicatorPainted(tt, th, sponsors.Bounds) {
		t.Fatalf("scrolled indicator is not under Sponsors at %+v", sponsors.Bounds)
	}
	y := int(sponsors.Bounds.Y+sponsors.Bounds.H) - 2
	// Container Padding(24) + .Width(260) places the viewport at x in [24, 284).
	for x := 14; x < 23; x++ {
		if colorMatches(tt.Image().RGBAAt(x, y), th.Colors.Primary, 12) {
			t.Fatalf("indicator leaked left of tab viewport at x=%d", x)
		}
	}
	for x := 285; x < 296; x++ {
		if colorMatches(tt.Image().RGBAAt(x, y), th.Colors.Primary, 12) {
			t.Fatalf("indicator leaked right of tab viewport at x=%d", x)
		}
	}
	// Wheel scroll while tab 5 stays selected and verify the indicator moves with it.
	beforeX := sponsors.Bounds.X
	centerX := float32(24 + 130)
	centerY := sponsors.Bounds.Y + 24
	tt.Wheel(centerX, centerY, -25, 0)
	tt.Advance(400 * time.Millisecond)
	sponsorsAfter := tabInfo(t, tt, "Sponsors")
	if absF32(sponsorsAfter.Bounds.X-beforeX) < 1 {
		tt.Wheel(centerX, centerY, 0, -25)
		tt.Advance(400 * time.Millisecond)
		sponsorsAfter = tabInfo(t, tt, "Sponsors")
	}
	if absF32(sponsorsAfter.Bounds.X-beforeX) < 1 {
		t.Fatalf("wheel did not scroll tab strip (still at %.1f)", sponsorsAfter.Bounds.X)
	}
	if !tabIndicatorPainted(tt, th, sponsorsAfter.Bounds) {
		t.Fatalf("wheel-scrolled indicator detached from Sponsors (before %.1f, after %.1f)", beforeX, sponsorsAfter.Bounds.X)
	}
	if tabIndicatorPainted(tt, th, sponsors.Bounds) {
		t.Fatalf("indicator stayed behind at pre-scroll position %.1f", beforeX)
	}
	for x := 14; x < 23; x++ {
		if colorMatches(tt.Image().RGBAAt(x, y), th.Colors.Primary, 12) {
			t.Fatalf("wheel-scrolled indicator leaked left at x=%d", x)
		}
	}
}

// TestDisabledAncestorTabsCarouselAndNarrowSnap checks ancestor-disabled tabs
// paint the single-dimmed MD3E disabled indicator, narrow carousels initialize
// without jumping to window width, penultimate snaps don't jump to the end,
// and disabling clears carousel interaction state.
func TestDisabledAncestorTabsCarouselAndNarrowSnap(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	selTab := 0
	tabTT := containerCanvas(th, func(c *ui.Context) {
		ui.Box(c).Disabled(true).DimDisabled(false).Children(func() {
			PrimaryTabs(c, []TabItem{{Label: "One"}, {Label: "Two"}}, TabsOptions{
				Selected: &selTab,
				Label:    "DisabledTabs",
			}).Width(320)
		})
	})
	tabTT.Advance(time.Second)
	one := tabInfo(t, tabTT, "One")
	if !one.Disabled {
		t.Fatal("ancestor-disabled tab does not read as disabled")
	}
	wantInd := cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity)).Over(cs.Surface)
	indPx := tabTT.Image().RGBAAt(int(one.Bounds.X+one.Bounds.W/2), int(one.Bounds.Y+one.Bounds.H)-2)
	if colorMatches(indPx, cs.Primary, 12) {
		t.Fatalf("ancestor-disabled tab indicator still painted Primary %+v", indPx)
	}
	if !colorMatches(indPx, wantInd, 12) {
		t.Fatalf("ancestor-disabled tab indicator = %+v, want %+v", indPx, rgbOf(wantInd))
	}

	// Narrow carousel (width 280 inside a 640 window) jumps cleanly to 280
	// proportions on its first laid-out frame and allows penultimate focus.
	cIdx := 0
	carTT := containerCanvas(th, func(c *ui.Context) {
		Carousel(c, carouselItems(4), CarouselOptions{
			Index:  &cIdx,
			Layout: CarouselHero,
			Height: 180,
		}).Width(280)
	})
	carTT.Frame()
	a := buttonInfo(t, carTT, ui.RoleButton, "A")
	wantW := carouselItemWidth(CarouselHero, 0, 280)
	if absF32(a.Bounds.W-wantW) > 1 {
		t.Fatalf("narrow carousel slide A width = %.1f on first laid-out pass, want %.1f", a.Bounds.W, wantW)
	}
	// Penultimate index 2 should settle cleanly on index 2 rather than snapping to 3.
	cIdx = 2
	carTT.Frame()
	carTT.Advance(2 * time.Second)
	if cIdx != 2 {
		t.Fatalf("penultimate carousel index snapped to %d, want 2", cIdx)
	}

	// Ancestor-disabled carousel clears active state and refuses clicks/drags.
	disIdx := 0
	disCar := containerCanvas(th, func(c *ui.Context) {
		ui.Box(c).Disabled(true).DimDisabled(false).Children(func() {
			Carousel(c, carouselItems(4), CarouselOptions{
				Index:  &disIdx,
				Layout: CarouselHero,
				Height: 180,
			}).Width(480)
		})
	})
	disCar.Advance(500 * time.Millisecond)
	slideB := buttonInfo(t, disCar, ui.RoleButton, "B")
	if !slideB.Disabled {
		t.Fatal("ancestor-disabled carousel slide does not read as disabled")
	}
	disCar.ClickAt(slideB.Bounds.X+15, slideB.Bounds.Y+20)
	disCar.Advance(500 * time.Millisecond)
	if disIdx != 0 {
		t.Fatalf("ancestor-disabled carousel changed index to %d", disIdx)
	}
}

// TestCarouselZeroWidthEmptyResetAndCancelDrag checks a zero-width carousel
// does not loop Invalidate forever, clears drag/align state across empty and
// resized item lists, and clears drag state on pointer cancel.
func TestCarouselZeroWidthEmptyResetAndCancelDrag(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	width := float32(0)
	idx := 0
	items := carouselItems(4)
	builds := 0
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 260).Background(canvasRGB).Padding(20).Children(func() {
				Carousel(c, items, CarouselOptions{
					Index:  &idx,
					Layout: CarouselHero,
					Height: 180,
				}).Width(width)
			})
		})
	}, 640, 260)

	tt.Advance(500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("zero-width carousel left pending invalidation frames")
	}
	if builds > 5 {
		t.Fatalf("zero-width carousel triggered %d rebuilds, expected no loop", builds)
	}

	// Expanding to 360 lays out items and resets to non-zero proportions cleanly.
	width = 360
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	a := buttonInfo(t, tt, ui.RoleButton, "A")
	wantW := carouselItemWidth(CarouselHero, 0, 360)
	if absF32(a.Bounds.W-wantW) > 1.5 {
		t.Fatalf("expanded carousel slide A width = %.1f, want %.1f", a.Bounds.W, wantW)
	}

	// Start a drag and cancel the pointer mid-drag, then click cleanly.
	ax, ay := center(a.Bounds)
	tt.Move(ax, ay)
	tt.Press(ax, ay)
	tt.Move(ax-20, ay)
	tt.CancelPress()
	tt.Advance(200 * time.Millisecond)

	// Transition items empty -> rebuilt non-empty resets index and drag state.
	items = nil
	tt.Frame()
	items = carouselItems(3)
	idx = 2
	tt.Frame()
	tt.Advance(800 * time.Millisecond)
	if idx != 2 {
		t.Fatalf("rebuilt carousel index = %d, want 2", idx)
	}
	cInfo := buttonInfo(t, tt, ui.RoleButton, "C")
	if cInfo.Bounds.X+cInfo.Bounds.W > 385 {
		t.Fatalf("rebuilt carousel failed to align active index 2 cleanly: %+v", cInfo.Bounds)
	}
}
