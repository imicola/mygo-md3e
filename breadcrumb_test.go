package md3e

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestBreadcrumbClicksCurrentPageAndAccessibility(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	var clicked []string

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(640, 160).Background(canvasRGB).Padding(16).Children(func() {
				Breadcrumb(c.Key("trail"), []BreadcrumbItem{
					{Label: "Home", Icon: mdicon.Add.Draw, OnClick: func() { clicked = append(clicked, "Home") }},
					{Label: "Docs", OnClick: func() { clicked = append(clicked, "Docs") }},
					{Label: "Locked", Disabled: true, OnClick: func() { clicked = append(clicked, "Locked") }},
					{Label: "Pagination", OnClick: func() { clicked = append(clicked, "Pagination") }},
				}, BreadcrumbOptions{
					Label: "Document path",
				})
			})
		})
	}, 640, 160)

	if tt.FramePending() {
		t.Fatal("settled Breadcrumb left FramePending true")
	}

	nav, ok := tt.AccessByRole(ui.RoleNavigation, "Document path")
	if !ok {
		t.Fatal("Breadcrumb missing RoleNavigation landmark")
	}

	home := buttonInfo(t, tt, ui.RoleButton, "Home")
	docs := buttonInfo(t, tt, ui.RoleButton, "Docs")
	locked := buttonInfo(t, tt, ui.RoleButton, "Locked")
	if !locked.Disabled {
		t.Fatal("disabled BreadcrumbItem does not report Disabled=true")
	}
	if !(home.Bounds.X < docs.Bounds.X && docs.Bounds.X < locked.Bounds.X) {
		t.Fatalf("LTR order invalid: home=%v docs=%v locked=%v", home.Bounds, docs.Bounds, locked.Bounds)
	}

	cur, ok := tt.AccessByRole(ui.RoleText, "Pagination")
	if !ok {
		t.Fatal("current page RoleText missing")
	}
	if cur.Current != ui.CurrentPage {
		t.Fatalf("final item Current=%v, want ui.CurrentPage", cur.Current)
	}
	if math.Abs(float64(nav.Bounds.H-32)) > 0.5 {
		t.Fatalf("Breadcrumb wrapped or unexpected height=%v, want 32", nav.Bounds.H)
	}

	// Current page has SecondaryContainer static highlight.
	curPx := tt.Image().RGBAAt(int(cur.Bounds.X+4), int(cur.Bounds.Y+cur.Bounds.H/2))
	if !colorMatches(curPx, th.Colors.SecondaryContainer, 8) {
		t.Fatalf("current page pixel=%+v, want SecondaryContainer %+v", curPx, rgbOf(th.Colors.SecondaryContainer))
	}

	// Clicking ancestor fires OnClick; clicking disabled or current page does nothing.
	tt.ClickAt(center(home.Bounds))
	tt.ClickAt(center(docs.Bounds))
	tt.ClickAt(center(locked.Bounds))
	tt.ClickAt(center(cur.Bounds))
	tt.Move(2, 2)
	tt.Advance(400 * time.Millisecond)

	if !slices.Equal(clicked, []string{"Home", "Docs"}) {
		t.Fatalf("clicked=%v, want [Home Docs]", clicked)
	}
	if tt.FramePending() {
		t.Fatal("Breadcrumb left FramePending true after clicks")
	}
}

func TestBreadcrumbMaxVisibleAndOverflowMenu(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	var clicked []string

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(640, 260).Background(canvasRGB).Padding(16).Children(func() {
				Breadcrumb(c, []BreadcrumbItem{
					{Label: "Root", OnClick: func() { clicked = append(clicked, "Root") }},
					{Label: "Workspace", Icon: mdicon.Star.Draw, OnClick: func() { clicked = append(clicked, "Workspace") }},
					{Label: "Design", Disabled: true, OnClick: func() { clicked = append(clicked, "Design") }},
					{Label: "Components", OnClick: func() { clicked = append(clicked, "Components") }},
					{Label: "Breadcrumb"},
				}, BreadcrumbOptions{
					MaxVisible: 3,
				})
			})
		})
	}, 640, 260)

	// Default label "Breadcrumb" for RoleNavigation.
	if _, ok := tt.AccessByRole(ui.RoleNavigation, "Breadcrumb"); !ok {
		t.Fatal("default RoleNavigation label \"Breadcrumb\" missing")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Root"); !ok {
		t.Fatal("first crumb Root should stay visible")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Workspace"); ok {
		t.Fatal("collapsed middle crumb Workspace should not be rendered inline")
	}
	if _, ok := tt.AccessByRole(ui.RoleText, "Breadcrumb"); !ok {
		t.Fatal("final crumb Breadcrumb should stay visible")
	}

	more, ok := tt.AccessByRole(ui.RoleMenuButton, "More path items")
	if !ok {
		t.Fatal("overflow RoleMenuButton missing")
	}
	if more.Expanded {
		t.Fatal("overflow menu button should start collapsed")
	}

	// Open overflow menu and click a collapsed item.
	tt.ClickAt(center(more.Bounds))
	tt.Advance(200 * time.Millisecond)

	more, _ = tt.AccessByRole(ui.RoleMenuButton, "More path items")
	if !more.Expanded {
		t.Fatal("overflow menu button did not report Expanded=true while open")
	}
	if _, ok := tt.AccessByRole(ui.RoleMenu, "More path items"); !ok {
		t.Fatal("overflow RoleMenu missing when opened")
	}

	// Disabled item inside menu cannot be activated; enabled item runs OnClick and closes menu.
	designItem, ok := tt.AccessByRole(ui.RoleMenuItem, "Design")
	if !ok || !designItem.Disabled {
		t.Fatalf("collapsed disabled menu item = %+v ok=%v", designItem, ok)
	}
	tt.ClickAt(center(designItem.Bounds))
	if len(clicked) != 0 {
		t.Fatalf("disabled overflow item fired OnClick: %v", clicked)
	}

	compItem, ok := tt.AccessByRole(ui.RoleMenuItem, "Components")
	if !ok {
		t.Fatal("collapsed menu item Components missing")
	}
	tt.ClickAt(center(compItem.Bounds))
	tt.Advance(300 * time.Millisecond)
	if !slices.Equal(clicked, []string{"Components"}) {
		t.Fatalf("overflow item click gave clicked=%v, want [Components]", clicked)
	}
	if _, ok := tt.AccessByRole(ui.RoleMenu, "More path items"); ok {
		t.Fatal("overflow menu stayed open after selecting item")
	}
	if tt.FramePending() {
		t.Fatal("Breadcrumb overflow menu left FramePending true")
	}
}

func TestBreadcrumbResponsiveNarrowCollapseAndExpand(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	boxW := float32(720)
	var clicked []string

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(boxW, 200).Background(canvasRGB).Padding(8).Children(func() {
				Breadcrumb(c, []BreadcrumbItem{
					{Label: "Home", OnClick: func() { clicked = append(clicked, "Home") }},
					{Label: "Engineering", OnClick: func() { clicked = append(clicked, "Engineering") }},
					{Label: "Architecture", OnClick: func() { clicked = append(clicked, "Architecture") }},
					{Label: "Navigation", OnClick: func() { clicked = append(clicked, "Navigation") }},
					{Label: "Overview"},
				})
			})
		})
	}, 720, 200)

	if _, ok := tt.AccessByRole(ui.RoleMenuButton, "More path items"); ok {
		t.Fatal("wide 720 DIP Breadcrumb unexpectedly collapsed items")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Architecture"); !ok {
		t.Fatal("wide 720 DIP Breadcrumb did not show Architecture")
	}

	// Shrink container width: middle items collapse automatically without wrapping.
	boxW = 240
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleMenuButton, "More path items"); !ok {
		t.Fatal("narrow 240 DIP Breadcrumb did not collapse into overflow menu")
	}
	nav, _ := tt.AccessByRole(ui.RoleNavigation, "Breadcrumb")
	if nav.Bounds.H > 34 {
		t.Fatalf("narrow Breadcrumb wrapped to height=%v", nav.Bounds.H)
	}
	cur, ok := tt.AccessByRole(ui.RoleText, "Overview")
	if !ok {
		t.Fatal("current page Overview missing after narrow collapse")
	}
	if cur.Bounds.X+cur.Bounds.W > nav.Bounds.X+nav.Bounds.W+1 {
		t.Fatalf("Overview overflowed container bounds: cur=%+v nav=%+v", cur.Bounds, nav.Bounds)
	}
	if tt.FramePending() {
		t.Fatal("narrow Breadcrumb left FramePending true")
	}

	// Widen back to 720 DIP: collapsed items expand inline again.
	boxW = 720
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleMenuButton, "More path items"); ok {
		t.Fatal("widened 720 DIP Breadcrumb kept overflow button")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Architecture"); !ok {
		t.Fatal("widened 720 DIP Breadcrumb did not restore Architecture")
	}
	if tt.FramePending() {
		t.Fatal("re-widened Breadcrumb left FramePending true")
	}

	// Verify 2-item path in narrow width never creates an empty overflow menu and truncates cleanly.
	tt2 := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(150, 120).Background(canvasRGB).Padding(4).Children(func() {
				Breadcrumb(c, []BreadcrumbItem{
					{Label: "Workspace Root"},
					{Label: "Very Long Settings Page"},
				})
			})
		})
	}, 150, 120)
	if _, ok := tt2.AccessByRole(ui.RoleMenuButton, "More path items"); ok {
		t.Fatal("2-item Breadcrumb unexpectedly created overflow menu")
	}
	if tt2.FramePending() {
		t.Fatal("narrow 2-item Breadcrumb left FramePending true")
	}
}

func TestBreadcrumbRTLCustomSeparatorDensityDisabledAndHover(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	sepCalls := 0
	builds := 0

	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		ThemeScope(c, th, func() {
			ui.Column(c).Size(460, 180).Background(canvasRGB).Padding(12).Children(func() {
				Breadcrumb(c, []BreadcrumbItem{
					{Label: "First", OnClick: func() {}},
					{Label: "Second", OnClick: func() {}},
					{Label: "Current"},
				}, BreadcrumbOptions{
					Label: "RTL trail",
				})
			})
		})
	}, 460, 180)

	// Measure separator orientation in LTR vs RTL: ChevronRight ink sits further right in LTR,
	// ChevronLeft ink sits further left in RTL.
	firstLTR := buttonInfo(t, tt, ui.RoleButton, "First")
	secondLTR := buttonInfo(t, tt, ui.RoleButton, "Second")
	sepBandLTR := ui.Rect{
		X: firstLTR.Bounds.X + firstLTR.Bounds.W,
		Y: firstLTR.Bounds.Y,
		W: secondLTR.Bounds.X - (firstLTR.Bounds.X + firstLTR.Bounds.W),
		H: firstLTR.Bounds.H,
	}
	ltrCenter := inkCenterX(tt, sepBandLTR, canvasRGB)

	// Hovering a crumb triggers 0 view rebuilds.
	tt.Move(center(firstLTR.Bounds))
	hoverStart := builds
	tt.Advance(250 * time.Millisecond)
	if builds != hoverStart {
		t.Fatalf("hovering Breadcrumb item triggered %d view rebuilds", builds-hoverStart)
	}
	if tt.FramePending() {
		t.Fatal("hovered Breadcrumb did not settle")
	}

	tt.SetDirection(ui.RightToLeft)
	tt.Advance(200 * time.Millisecond)

	firstRTL := buttonInfo(t, tt, ui.RoleButton, "First")
	secondRTL := buttonInfo(t, tt, ui.RoleButton, "Second")
	curRTL, _ := tt.AccessByRole(ui.RoleText, "Current")
	if !(firstRTL.Bounds.X > secondRTL.Bounds.X && secondRTL.Bounds.X > curRTL.Bounds.X) {
		t.Fatalf("RTL order not mirrored: first=%v second=%v cur=%v", firstRTL.Bounds, secondRTL.Bounds, curRTL.Bounds)
	}
	sepBandRTL := ui.Rect{
		X: secondRTL.Bounds.X + secondRTL.Bounds.W,
		Y: firstRTL.Bounds.Y,
		W: firstRTL.Bounds.X - (secondRTL.Bounds.X + secondRTL.Bounds.W),
		H: firstRTL.Bounds.H,
	}
	rtlCenter := inkCenterX(tt, sepBandRTL, canvasRGB)
	ltrRel := ltrCenter - (sepBandLTR.X + sepBandLTR.W/2)
	rtlRel := rtlCenter - (sepBandRTL.X + sepBandRTL.W/2)
	if !(ltrRel > 0 && rtlRel < 0) {
		t.Fatalf("separator did not mirror direction: ltrOffset=%v rtlOffset=%v", ltrRel, rtlRel)
	}

	// Custom Separator, Density heights, and Disabled.
	disClicks := 0
	for _, tc := range []struct {
		d     tokens.Density
		wantH float32
	}{
		{tokens.DensityComfortable, 32},
		{tokens.DensityDesktop, 28},
		{tokens.DensityCompact, 24},
	} {
		dth := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tc.d})
		dtt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, dth, func() {
				ui.Column(c).Size(360, 120).Background(canvasRGB).Children(func() {
					Breadcrumb(c, []BreadcrumbItem{
						{Label: "Home", OnClick: func() { disClicks++ }},
						{Label: "End"},
					}, BreadcrumbOptions{
						Disabled: true,
						Separator: func(p *ui.Painter, r ui.Rect, ink ui.Color) {
							sepCalls++
							p.Fill(r, ink, 1)
						},
					})
				})
			})
		}, 360, 120)
		home := buttonInfo(t, dtt, ui.RoleButton, "Home")
		if math.Abs(float64(home.Bounds.H-tc.wantH)) > 0.5 || !home.Disabled {
			t.Fatalf("density %v crumb height=%v (want %v) disabled=%v", tc.d, home.Bounds.H, tc.wantH, home.Disabled)
		}
		dtt.ClickAt(center(home.Bounds))
	}
	if sepCalls == 0 || disClicks != 0 {
		t.Fatalf("custom separator calls=%d (want >0), disabled clicks=%d (want 0)", sepCalls, disClicks)
	}
}

func TestBreadcrumbEdgeCasesAndReduceMotion(t *testing.T) {
	thRM := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	var clicked string

	// Empty and single-item trails settle without pending frames.
	ttEmpty := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, thRM, func() {
			ui.Column(c).Size(0, 100).Children(func() {
				Breadcrumb(c, nil)
			})
			ui.Column(c).Size(240, 100).Children(func() {
				Breadcrumb(c, []BreadcrumbItem{{Label: "OnlyPage"}})
			})
		})
	}, 240, 100)
	if ttEmpty.FramePending() {
		t.Fatal("empty/single-item Breadcrumb left FramePending true")
	}
	onlyCur, ok := ttEmpty.AccessByRole(ui.RoleText, "OnlyPage")
	if !ok || onlyCur.Current != ui.CurrentPage {
		t.Fatalf("single-item Breadcrumb access=%+v ok=%v", onlyCur, ok)
	}

	// MaxVisible == 1 shows ellipsis + current page and triggers collapsed callback under ReduceMotion.
	ttOne := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, thRM, func() {
			ui.Column(c).Size(320, 200).Background(canvasRGB).Padding(8).Children(func() {
				Breadcrumb(c, []BreadcrumbItem{
					{Label: "Root", OnClick: func() { clicked = "Root" }},
					{Label: "Folder", OnClick: func() { clicked = "Folder" }},
					{Label: "Leaf"},
				}, BreadcrumbOptions{MaxVisible: 1})
			})
		})
	}, 320, 200)
	if _, ok := ttOne.AccessByRole(ui.RoleButton, "Root"); ok {
		t.Fatal("MaxVisible=1 should collapse Root into overflow menu")
	}
	more, ok := ttOne.AccessByRole(ui.RoleMenuButton, "More path items")
	if !ok {
		t.Fatal("MaxVisible=1 missing overflow menu trigger")
	}
	ttOne.Move(center(more.Bounds))
	if ttOne.FramePending() {
		t.Fatal("ReduceMotion Breadcrumb hover left FramePending true")
	}
	ttOne.ClickAt(center(more.Bounds))
	rootMenuItem, ok := ttOne.AccessByRole(ui.RoleMenuItem, "Root")
	if !ok {
		t.Fatal("MaxVisible=1 overflow menu missing Root item")
	}
	ttOne.ClickAt(center(rootMenuItem.Bounds))
	if clicked != "Root" || ttOne.FramePending() {
		t.Fatalf("MaxVisible=1 overflow click=%q FramePending=%v", clicked, ttOne.FramePending())
	}
}

func inkCenterX(tt *ui.Tester, band ui.Rect, bg ui.Color) float32 {
	img := tt.Image()
	var sumX, count float32
	for y := int(band.Y); y < int(band.Y+band.H); y++ {
		for x := int(band.X); x < int(band.X+band.W); x++ {
			if !colorMatches(img.RGBAAt(x, y), bg, 12) {
				sumX += float32(x) + 0.5
				count++
			}
		}
	}
	if count == 0 {
		return band.X + band.W/2
	}
	return sumX / count
}
