package md3e

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

func TestWindowSizeClassBoundaries(t *testing.T) {
	cases := []struct {
		w    float32
		want WindowSizeClass
	}{
		{0, WindowCompact},
		{-100, WindowCompact},
		{float32(math.NaN()), WindowCompact},
		{float32(math.Inf(-1)), WindowCompact},
		{599.9, WindowCompact},
		{600, WindowMedium},
		{839.9, WindowMedium},
		{840, WindowExpanded},
		{1440, WindowExpanded},
		{float32(math.Inf(1)), WindowExpanded},
	}
	for _, tc := range cases {
		if got := WindowSizeClassForWidth(tc.w); got != tc.want {
			t.Errorf("WindowSizeClassForWidth(%v) = %v, want %v", tc.w, got, tc.want)
		}
	}
	if WindowSizeClassOf(nil) != WindowCompact {
		t.Fatal("WindowSizeClassOf(nil) should be WindowCompact")
	}
	if WindowCompact.String() != "Compact" || WindowMedium.String() != "Medium" || WindowExpanded.String() != "Expanded" {
		t.Fatalf("unexpected WindowSizeClass strings: %q %q %q", WindowCompact, WindowMedium, WindowExpanded)
	}
}

func TestAdaptiveScaffoldSwitchesNavAndPreservesContentState(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{
		Density:      tokens.DensityDesktop,
		ReduceMotion: true,
	})
	dests := []NavigationDestination{
		{Label: "Home", Icon: func(p *ui.Painter, r ui.Rect, ink ui.Color) { p.Fill(r, ink, 0) }},
		{Label: "Nodes", Icon: func(p *ui.Painter, r ui.Rect, ink ui.Color) { p.Fill(r, ink, 0) }},
	}
	selected := 0
	var localCounter *int
	type bodyCounterKey struct{}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			AdaptiveScaffold(c, dests, func() {
				box := ui.Column(c.Key("body-state")).Fill().Padding(16)
				localCounter = ui.Local(box, bodyCounterKey{}, func() int { return 0 })
				box.Children(func() {
					if Button(c, "Increment").Clicked() {
						*localCounter++
					}
					Text(c, "Body", tokens.BodyMedium)
				})
			}, AdaptiveScaffoldOptions{
				Selected:        &selected,
				NavigationLabel: "MainNav",
				Header: func() ui.Element {
					return WindowTitleBar(c, WindowTitleBarOptions{Title: "AdaptiveApp"})
				},
				FAB: func() ui.Element {
					return FAB(c, "Add", func(p *ui.Painter, r ui.Rect, ink ui.Color) { p.Fill(r, ink, 0) })
				},
			})
		}).Fill()
	}, 480, 640)

	// 1. Compact (480 DIP): bottom NavigationBar (full width 480, height 80 at bottom).
	nav, ok := tt.AccessByRole(ui.RoleNavigation, "MainNav")
	if !ok {
		t.Fatal("Compact scaffold missing MainNav")
	}
	if absF32(nav.Bounds.W-480) > 1 || absF32(nav.Bounds.H-navBarHeight) > 1 {
		t.Fatalf("Compact nav bounds = %+v, want bottom NavigationBar (480x80)", nav.Bounds)
	}

	// Mutate content-local state before resizing across breakpoints.
	tt.ClickAt(center(buttonInfo(t, tt, ui.RoleButton, "Increment").Bounds))
	tt.Frame()
	if localCounter == nil || *localCounter != 1 {
		t.Fatalf("localCounter = %v, want 1", localCounter)
	}

	// 2. Medium (720 DIP): switches to collapsed NavigationRail (80 DIP wide).
	tt.SetSize(720, 640)
	tt.Frame()
	nav, ok = tt.AccessByRole(ui.RoleNavigation, "MainNav")
	if !ok {
		t.Fatal("Medium scaffold missing MainNav")
	}
	if absF32(nav.Bounds.W-navRailCollapsedWidth) > 1 {
		t.Fatalf("Medium nav width = %.1f, want collapsed rail %v", nav.Bounds.W, navRailCollapsedWidth)
	}
	if *localCounter != 1 {
		t.Fatalf("content local state lost on Compact->Medium transition: got %d, want 1", *localCounter)
	}

	// 3. Expanded (1024 DIP): expands NavigationRail to Desktop default width (192 DIP).
	tt.SetSize(1024, 640)
	tt.Frame()
	nav, _ = tt.AccessByRole(ui.RoleNavigation, "MainNav")
	if absF32(nav.Bounds.W-float32(tokens.DensityDesktop.NavRailExpandedWidth())) > 1 {
		t.Fatalf("Expanded nav width = %.1f, want Desktop expanded rail 192", nav.Bounds.W)
	}
	if *localCounter != 1 {
		t.Fatalf("content local state lost on Medium->Expanded transition: got %d, want 1", *localCounter)
	}

	// Clicking a destination updates Selected and settles cleanly in RTL too.
	tt.SetDirection(ui.RightToLeft)
	tt.Frame()
	nodesTab := tabInfo(t, tt, "Nodes")
	tt.ClickAt(center(nodesTab.Bounds))
	tt.Frame()
	if selected != 1 {
		t.Fatalf("selected = %d after clicking Nodes, want 1", selected)
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion AdaptiveScaffold left pending frames")
	}
}

func TestListDetailPaneAndSupportingPaneAdaptiveAndDrag(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	splitW := float32(300)
	showDetail := false
	var listR, detailR ui.Rect

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ListDetailPane(c,
				func() {
					ui.Box(c).Fill().DrawOver(func(_ *ui.Painter, r ui.Rect) { listR = r }).Children(func() {
						Text(c, "ListPane", tokens.BodyMedium)
					})
				},
				func() {
					ui.Box(c).Fill().DrawOver(func(_ *ui.Painter, r ui.Rect) { detailR = r }).Children(func() {
						Text(c, "DetailPane", tokens.BodyMedium)
					})
				},
				ListDetailPaneOptions{
					SplitWidth: &splitW,
					ShowDetail: &showDetail,
					Label:      "InboxSplit",
				},
			)
		}).Fill()
	}, 960, 480)

	// Expanded: both panes render side by side.
	if !tt.HasText("ListPane") || !tt.HasText("DetailPane") {
		t.Fatal("Expanded ListDetailPane should render both panes")
	}
	if absF32(listR.W-300) > 4 || detailR.W <= 0 || listR.X >= detailR.X {
		t.Fatalf("Expanded ListDetailPane rects unexpected: list=%+v detail=%+v", listR, detailR)
	}

	// Drag split divider rightward to widen the list pane.
	seamX := listR.X + listR.W + 2
	midY := listR.Y + listR.H/2
	tt.Press(seamX, midY)
	tt.Move(seamX+80, midY)
	tt.Release(seamX+80, midY)
	tt.Frame()
	if splitW <= 340 {
		t.Fatalf("dragging ListDetailPane divider did not grow splitW: got %.1f, want > 340", splitW)
	}

	// RTL places list pane on the right of detail pane.
	tt.SetDirection(ui.RightToLeft)
	tt.Frame()
	if listR.X <= detailR.X {
		t.Fatalf("RTL ListDetailPane did not mirror panes: list=%+v detail=%+v", listR, detailR)
	}

	// Compact: collapses to single pane controlled by ShowDetail.
	tt.SetDirection(ui.LeftToRight)
	tt.SetSize(480, 480)
	tt.Frame()
	if !tt.HasText("ListPane") || tt.HasText("DetailPane") {
		t.Fatal("Compact ListDetailPane with ShowDetail=false should show only ListPane")
	}
	showDetail = true
	tt.Frame()
	if tt.HasText("ListPane") || !tt.HasText("DetailPane") {
		t.Fatal("Compact ListDetailPane with ShowDetail=true should show only DetailPane")
	}

	// SupportingPane defaults to visible on Expanded and hidden on Medium/Compact.
	var showSup *bool
	supTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			SupportingPane(c,
				func() { Text(c, "MainDoc", tokens.BodyMedium) },
				func() { Text(c, "Outline", tokens.BodyMedium) },
				SupportingPaneOptions{
					ShowSupporting: showSup,
					Label:          "DocWorkspace",
				},
			)
		}).Fill()
	}, 1000, 480)
	if !supTT.HasText("MainDoc") || !supTT.HasText("Outline") {
		t.Fatal("Expanded SupportingPane should display both MainDoc and Outline by default")
	}
	supTT.SetSize(720, 480)
	supTT.Frame()
	if !supTT.HasText("MainDoc") || supTT.HasText("Outline") {
		t.Fatal("Medium SupportingPane should hide Outline by default")
	}
	tVal := true
	showSup = &tVal
	supTT.SetSize(480, 480)
	supTT.Frame()
	if supTT.HasText("MainDoc") || !supTT.HasText("Outline") {
		t.Fatal("Compact SupportingPane with ShowSupporting=true should show Outline")
	}
	supTT.Advance(100 * time.Millisecond)
	if supTT.FramePending() {
		t.Fatal("SupportingPane left pending frames")
	}
}

func TestBentoGridResponsiveSpansAndZeroSizeSafety(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
	var heroR, cell1R, cell2R ui.Rect

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(16).Children(func() {
				BentoGrid(c, func() {
					BentoSpan(
						Card(c, func() { ui.Box(c).Height(120) }, CardOptions{Label: "Hero"}).
							DrawOver(func(_ *ui.Painter, r ui.Rect) { heroR = r }),
						2, 2,
					)
					Card(c, func() { ui.Box(c).Height(50) }, CardOptions{Label: "Cell1"}).
						DrawOver(func(_ *ui.Painter, r ui.Rect) { cell1R = r })
					Card(c, func() { ui.Box(c).Height(50) }, CardOptions{Label: "Cell2"}).
						DrawOver(func(_ *ui.Painter, r ui.Rect) { cell2R = r })
				}, BentoGridOptions{Label: "DashboardGrid"})
			})
		}).Fill()
	}, 1000, 480)

	// Expanded (1000 DIP) -> 4 columns; Hero spans 2 columns so heroR.W > cell1R.W*1.8.
	if heroR.W <= cell1R.W*1.8 {
		t.Fatalf("BentoSpan(2,2) hero width %.1f not ~2x single cell %.1f", heroR.W, cell1R.W)
	}
	if absF32(heroR.Y-cell1R.Y) > 1 || absF32(cell1R.Y-cell2R.Y) > 1 {
		t.Fatalf("Expanded 4-col BentoGrid should place Hero, Cell1, Cell2 on row 1: hero=%+v c1=%+v c2=%+v", heroR, cell1R, cell2R)
	}

	// Compact (420 DIP) -> 1 column, stacked vertically.
	tt.SetSize(420, 480)
	tt.Frame()
	if cell1R.Y <= heroR.Y || cell2R.Y <= cell1R.Y {
		t.Fatalf("Compact 1-col BentoGrid did not stack cells vertically: hero=%+v c1=%+v c2=%+v", heroR, cell1R, cell2R)
	}

	// Zero-sized BentoGrid must not panic or loop Invalidate().
	builds := 0
	zeroTT := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			BentoGrid(c, func() {
				BentoSpan(ui.Box(c).Height(20), 2, 1)
			}).Size(0, 0)
		})
	}, 200, 200)
	zeroTT.Advance(200 * time.Millisecond)
	if zeroTT.FramePending() {
		t.Fatal("zero-sized BentoGrid left pending frames")
	}
	if builds > 3 {
		t.Fatalf("zero-sized BentoGrid triggered %d rebuilds", builds)
	}
}
