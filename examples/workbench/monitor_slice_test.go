package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/charts"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/symbols"
	"github.com/imicola/mygo-md3e/tokens"
)

// monitorSliceOptions configures the reusable Phase 3 Monitor slice fixture
// (title bar, two MetricCards, ledger filter field, and virtualized DataTable)
// shared across M16 baseline benchmarks, M16.3 visual state boards, and M17–M22
// before/after comparisons.
type monitorSliceOptions struct {
	Seed         ui.Color
	Variant      tokens.Variant
	Dark         bool
	Contrast     tokens.Contrast
	Density      tokens.Density
	ReduceMotion bool
	RTL          bool
	Query        string
}

func defaultMonitorSliceOptions() monitorSliceOptions {
	return monitorSliceOptions{
		Seed:         seedPresets[0].color,
		Variant:      tokens.Expressive,
		Dark:         true,
		Contrast:     tokens.ContrastStandard,
		Density:      tokens.DensityDesktop,
		ReduceMotion: true,
	}
}

type monitorSliceFixture struct {
	app   *appState
	theme *md3e.Theme
	rtl   bool
}

func newMonitorSliceFixture(o monitorSliceOptions) *monitorSliceFixture {
	a := newApp()
	// Freeze live telemetry so virtual-clock frames are 100% deterministic.
	a.mon.running = false
	a.mon.query = o.Query
	a.mon.markFilterDirty()
	a.mon.ensureFiltered()

	seed := o.Seed
	if seed.A == 0 {
		seed = seedPresets[0].color
	}
	th := md3e.ThemeFromSeed(seed, md3e.ThemeOptions{
		Dark:         o.Dark,
		Variant:      o.Variant,
		Contrast:     o.Contrast,
		Density:      o.Density,
		Motion:       motion.Expressive,
		ReduceMotion: o.ReduceMotion,
	})
	return &monitorSliceFixture{app: a, theme: th, rtl: o.RTL}
}

func (f *monitorSliceFixture) view(c *ui.Context) {
	a := f.app
	th := f.theme
	cs := th.Colors
	a.mon.ensureFiltered()

	dir := ui.LeftToRight
	if f.rtl {
		dir = ui.RightToLeft
	}
	ui.Box(c).Fill().Direction(dir).Children(func() {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Background(cs.Surface).TextColor(cs.OnSurface).Children(func() {
				md3e.WindowTitleBar(c.Key("slice-titlebar"), md3e.WindowTitleBarOptions{
					Title: "Workbench Monitor Slice",
					Leading: func() ui.Element {
						return md3e.Text(c, "Workbench", tokens.TitleSmall, true)
					},
				})
				ui.Scroll(c.Key("slice-scroll")).Grow(1).FillWidth().Children(func() {
					ui.Column(c).FillWidth().Padding(16, 20, 24, 20).Gap(14).Children(func() {
						md3e.BentoGrid(c.Key("slice-metrics"), func() {
							downCol := charts.SeriesColor(th, 0)
							charts.MetricCard(c.Key("slice-download"), charts.MetricCardOptions{
								Title:      "Download",
								Subtitle:   "Aggregated ingress",
								Icon:       symbols.ArrowDownward.Draw,
								IconTone:   md3e.TonePrimary,
								Badge:      "Paused",
								BadgeTone:  md3e.TonePrimary,
								Value:      a.mon.downSpeedText,
								Unit:       "MB/s",
								Secondary:  a.mon.downPeakText,
								ValueColor: downCol,
								Sparkline: &charts.AreaSparklineOptions{
									Ring:         a.mon.downRing,
									Color:        downCol,
									ShowEndDot:   true,
									BaselineZero: true,
									Curve:        charts.CurveSmooth,
									Label:        "Download throughput trend",
								},
							})
							upCol := charts.SeriesColor(th, 1)
							charts.MetricCard(c.Key("slice-upload"), charts.MetricCardOptions{
								Title:      "Upload",
								Subtitle:   "Aggregated egress",
								Icon:       symbols.ArrowUpward.Draw,
								IconTone:   md3e.ToneTertiary,
								Badge:      "Uplink",
								BadgeTone:  md3e.ToneTertiary,
								Value:      a.mon.upSpeedText,
								Unit:       "MB/s",
								Secondary:  a.mon.upPeakText,
								ValueColor: upCol,
								Sparkline: &charts.AreaSparklineOptions{
									Ring:         a.mon.upRing,
									Color:        upCol,
									ShowEndDot:   true,
									BaselineZero: true,
									Curve:        charts.CurveSmooth,
									Label:        "Upload throughput trend",
								},
							})
						}, md3e.BentoGridOptions{
							CompactColumns:  1,
							MediumColumns:   2,
							ExpandedColumns: 2,
							Label:           "Monitor slice metrics",
						})

						md3e.Surface(c.Key("slice-ledger"), md3e.SurfaceOptions{
							Color:      cs.SurfaceContainerLow,
							ShapeScale: tokens.ShapeLargeIncreased,
							Hairline:   true,
						}).FillWidth().Padding(14).Column().Gap(10).Children(func() {
							md3e.TextField(c.Key("slice-filter"), &a.mon.query, md3e.TextFieldOptions{
								Variant:       md3e.TextFieldOutlined,
								Label:         "Filter host, process, or rule",
								Placeholder:   "github, go, Blocked...",
								LeadingIcon:   symbols.Search.Draw,
								TrailingIcon:  symbols.Close.Draw,
								TrailingLabel: "Clear slice filter",
								OnTrailing: func() {
									a.mon.query = ""
									a.mon.markFilterDirty()
								},
								OnChanged: func(string) {
									a.mon.markFilterDirty()
								},
							}).FillWidth()

							a.mon.ensureFiltered()
							sortPtr, sortByPtr := a.mon.activeSort()
							md3e.DataTable(c.Key("slice-table"), connColumns, len(a.mon.filtered), func(row, col int) {
								a.renderLedgerCell(c, cs, row, col)
							}, md3e.DataTableOptions{
								State:     a.mon.activeListState(),
								Selected:  &a.mon.selectedRow,
								Sort:      sortPtr,
								SortBy:    sortByPtr,
								RoundRows: true,
								Hairline:  true,
								Label:     "Connections slice table",
								ItemKey:   func(row int) any { return a.mon.recordIDAt(row) },
								RowLabel:  func(row int) string { return a.mon.hostAt(row) },
							}).FillWidth().Height(300)
						})
					})
				})
			})
		}).Fill()
	})
}

func TestMonitorSliceFixture(t *testing.T) {
	f := newMonitorSliceFixture(defaultMonitorSliceOptions())
	t.Cleanup(f.app.media.close)

	tt := ui.NewTester(f.view, 1280, 760)
	tt.Frame()
	if !tt.HasText("Workbench Monitor Slice") || !tt.HasText("Download") || !tt.HasText("Upload") {
		t.Fatal("Monitor slice fixture missing expected title bar or metric cards")
	}
	if _, ok := tt.AccessByRole(ui.RoleTable, "Connections slice table"); !ok {
		t.Fatal("Monitor slice fixture missing DataTable")
	}
	checkNamedControls(t, tt, "monitor slice")

	// Filter via TextField and verify deterministic narrowing and zero pending frames.
	field, ok := tt.AccessByRole(ui.RoleTextField, "Filter host, process, or rule")
	if !ok {
		t.Fatal("Monitor slice filter field missing")
	}
	tt.ClickAt(centerOf(field.Bounds))
	tt.Type("github")
	tt.Advance(200 * time.Millisecond)
	if len(f.app.mon.filtered) == 0 || len(f.app.mon.filtered) >= scale10K {
		t.Fatalf("filtered count = %d, want 0 < n < %d", len(f.app.mon.filtered), scale10K)
	}
	if tt.FramePending() {
		t.Fatal("settled ReduceMotion Monitor slice fixture left pending frames")
	}
}

func BenchmarkMonitorSliceSteady(b *testing.B) {
	f := newMonitorSliceFixture(defaultMonitorSliceOptions())
	b.Cleanup(f.app.media.close)
	tt := ui.NewTester(f.view, 1280, 760)
	for range 10 {
		tt.Frame()
	}
	b.ReportAllocs()
	for b.Loop() {
		tt.Frame()
	}
}
