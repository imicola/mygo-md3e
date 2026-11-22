package md3e

import (
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/imicola/mygo-md3e/shape"
	"github.com/imicola/mygo-md3e/tokens"
)

var dashboardTabs = []TabItem{{Label: "Overview"}, {Label: "Activity"}, {Label: "Settings"}}
var dashboardActions = []ButtonGroupItem{{Label: "Hour"}, {Label: "Day"}, {Label: "Week"}}
var dashboardItems = []ListItem{{Headline: "API gateway"}, {Headline: "Worker pool"}, {Headline: "Storage"}}

func benchmarkDashboardView(th *Theme) func(*ui.Context) {
	rest, selected := shape.Cookie9Sided(), shape.Circle()
	var running bool
	var period int
	return func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(8).Gap(6).Children(func() {
				PrimaryTabs(c.Key("tabs"), dashboardTabs)
				FilledCard(c.Key("summary"), func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						Button(c.Key("refresh"), "Refresh")
						MorphIconButton(c.Key("run"), &running, "Run workers", func(p *ui.Painter, r ui.Rect, ink ui.Color) {
							p.Fill(ui.Rect{X: r.X + r.W/4, Y: r.Y + r.H/4, W: r.W / 2, H: r.H / 2}, ink, 2)
						}, MorphIconButtonOptions{Shape: rest, SelectedShape: selected})
						StatusPill(c, "Healthy", StatusPillOptions{Tone: ToneSuccess})
					})
					LinearProgress(c, 0.64).FillWidth()
				}).FillWidth()
				PillTabs(c.Key("sections"), dashboardTabs)
				ButtonGroup(c.Key("period"), &period, dashboardActions, ButtonGroupOptions{Kind: ButtonGroupConnected})
				List(c.Key("services"), dashboardItems, ListOptions{Segmented: true})
			})
		}).Fill()
	}
}

func warmPerformanceView(tt *ui.Tester) {
	for range 10 {
		tt.Frame()
	}
}

func BenchmarkDashboardSteady(b *testing.B) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
	tt := ui.NewTester(benchmarkDashboardView(th), 640, 420)
	warmPerformanceView(tt)
	b.ReportAllocs()
	for b.Loop() {
		tt.Frame()
	}
}

func BenchmarkDashboardControls(b *testing.B) {
	for _, tc := range []struct {
		name string
		view func(*ui.Context)
	}{
		{"Button", func(c *ui.Context) { Button(c, "Refresh") }},
		{"Tabs", func(c *ui.Context) { PrimaryTabs(c, dashboardTabs) }},
		{"PillTabs", func(c *ui.Context) { PillTabs(c, dashboardTabs) }},
		{"List", func(c *ui.Context) { List(c, dashboardItems, ListOptions{Segmented: true}) }},
		{"ButtonGroup", func(c *ui.Context) { ButtonGroupActions(c, dashboardActions, nil) }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			tt := ui.NewTester(tc.view, 360, 180)
			warmPerformanceView(tt)
			b.ReportAllocs()
			for b.Loop() {
				tt.Frame()
			}
		})
	}
}

func BenchmarkDataTable100K(b *testing.B) {
	columns := []DataTableColumn{
		{Title: "Service", Width: 150, Sortable: true},
		{Title: "Requests", Width: 120, Numeric: true, Sortable: true},
		{Title: "Endpoint", Width: 160, Mono: true},
		{Title: "Health", Width: 100},
	}
	services := [...]string{"Gateway", "Storage", "Workers", "Queue"}
	requests := [...]string{"1678", "128", "84", "9"}
	endpoints := [...]string{"/api/v1", "/assets", "/jobs", "/events"}
	for _, scroll := range []bool{false, true} {
		name := "Steady"
		if scroll {
			name = "Scroll"
		}
		b.Run(name, func(b *testing.B) {
			sortBy := []ui.SortOrder{{Column: "Service"}, {Column: "Requests"}}
			state := ui.ListState{SortBy: &sortBy}
			th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
			tt := ui.NewTester(func(c *ui.Context) {
				ThemeScope(c, th, func() {
					DataTable(c, columns, 100_000, func(row, col int) {
						i := row % len(services)
						switch col {
						case 0:
							DataTableCellText(c, services[i])
						case 1:
							DataTableCellText(c, requests[i])
						case 2:
							DataTableCellText(c, endpoints[i])
						case 3:
							DataTableCellText(c, "Healthy")
						}
					}, DataTableOptions{State: &state}).Fill()
				}).Fill()
			}, 580, 330)
			warmPerformanceView(tt)
			b.ReportAllocs()
			row := 0
			for b.Loop() {
				if scroll {
					row = (row + 3) % 99_900
					state.ScrollTo(row, ui.Start)
				}
				tt.Frame()
			}
		})
	}
}

func shapeMorphPerformanceStep(animated bool) func() {
	poly, end := shape.Cookie9Sided(), shape.Circle()
	morph := shape.NewMorph(poly, end)
	progress := float32(0.5)
	var scratch ui.Path
	draw := func(p *ui.Painter, r ui.Rect) {
		r.W /= 2
		p.FillPath(poly.WriteTo(&scratch, r), ui.RGB(32, 88, 184))
		r.X += r.W
		p.FillPath(morph.AsPath(progress, r), ui.RGB(32, 88, 184))
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Size(72, 36).Draw(draw)
	}, 72, 36)
	if !animated {
		warmPerformanceView(tt)
		return tt.Frame
	}
	step := func() {
		progress += 0.03125
		if progress > 1 {
			progress = 0
		}
		tt.Frame()
	}
	for range 33 {
		step()
	}
	return step
}

func BenchmarkShapeMorphDraw(b *testing.B) {
	for _, tc := range []struct {
		name     string
		animated bool
	}{
		{"Steady", false},
		{"Animated", true},
	} {
		b.Run(tc.name, func(b *testing.B) {
			step := shapeMorphPerformanceStep(tc.animated)
			b.ReportAllocs()
			for b.Loop() {
				step()
			}
		})
	}
}

func TestShapeMorphDrawAllocations(t *testing.T) {
	step := shapeMorphPerformanceStep(false)
	if got := testing.AllocsPerRun(99, step); got != 0 {
		t.Fatalf("warmed shape and morph frame allocated %v times, want 0", got)
	}
}

func TestRetainedHeap(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: tokens.DensityDesktop})
	view := benchmarkDashboardView(th)
	warmPerformanceView(ui.NewTester(view, 640, 420))
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	tt := ui.NewTester(view, 640, 420)
	warmPerformanceView(tt)
	runtime.GC()
	runtime.ReadMemStats(&after)
	retained := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	t.Logf("retained Go heap for warmed dashboard: %d bytes", retained)
	runtime.KeepAlive(tt)
	if retained >= 5_000_000 {
		t.Fatalf("dashboard retained %d bytes, want < 5,000,000", retained)
	}

	t.Run("ThemeChurn", checkThemeCacheHeap)
}

func exerciseThemeCacheChurn(offset int) {
	for round := range 4 {
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, DefaultTheme(c), func() {
				DataTable(c, nil, 0, nil).Fill()
			}).Fill()
		}, 64, 48)
		for step := range 24 {
			n := offset + round*24 + step
			tt.SetDark(n%2 != 0)
			tt.SetPreferences(ui.Preferences{
				Accent:       ui.RGB(uint8(n), 95, uint8(n*3)),
				HighContrast: n%3 == 0,
				TextScale:    1 + float32(n%5)*0.1,
				ReduceMotion: true,
			})
		}
	}
}

func checkThemeCacheHeap(t *testing.T) {
	exerciseThemeCacheChurn(0)
	for range 4 {
		runtime.GC()
		runtime.Gosched()
	}
	uiBefore, specBefore := themeRegistryCountForTest(), themeSpecRegistryCountForTest()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	exerciseThemeCacheChurn(96)
	for range 20 {
		runtime.GC()
		runtime.Gosched()
		if themeRegistryCountForTest() <= uiBefore && themeSpecRegistryCountForTest() <= specBefore {
			break
		}
	}
	uiAfter, specAfter := themeRegistryCountForTest(), themeSpecRegistryCountForTest()
	if uiAfter > uiBefore || specAfter > specBefore {
		t.Errorf("weak theme registries grew after GC: UI %d to %d, spec %d to %d", uiBefore, uiAfter, specBefore, specAfter)
	}

	schemeCacheMu.RLock()
	schemes := len(schemeCache)
	schemeCacheMu.RUnlock()
	defaultThemeMu.RLock()
	defaults := len(defaultThemes)
	defaultThemeMu.RUnlock()
	if schemes > 64 || defaults > 32 {
		t.Errorf("theme caches exceed 64 schemes and 32 defaults: %d schemes, %d defaults", schemes, defaults)
	}
	runtime.ReadMemStats(&after)
	growth := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	t.Logf("theme churn retained growth: %d bytes; schemes %d, defaults %d, weak UI %d→%d, weak specs %d→%d",
		growth, schemes, defaults, uiBefore, uiAfter, specBefore, specAfter)
	if growth > 1_000_000 {
		t.Errorf("theme churn retained %d more bytes after warming, want <= 1,000,000", growth)
	}
}
