package md3e_test

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

func testIconDraw(p *ui.Painter, r ui.Rect, ink ui.Color) {
	p.Fill(ui.Rect{X: r.X + 2, Y: r.Y + 2, W: max(0, r.W-4), H: max(0, r.H-4)}, ink, 2)
}

// TestLegacyConsumerCompatibilityFixture locks existing public API signatures
// and struct literal compatibility across upstream mygo upgrades (ADR-10 / M16.1).
func TestLegacyConsumerCompatibilityFixture(t *testing.T) {
	// Verify function type signatures explicitly so accidental signature drift fails compilation.
	var (
		_ func(*ui.Context, string, ...md3e.SplitButtonOptions) (ui.Element, ui.Element, ui.Element, *bool) = md3e.SplitButton
		_ func(*ui.Context, *md3e.Theme, func()) ui.Element                                                 = md3e.ThemeScope
		_ func(*ui.Context, *md3e.Theme, func(), ...md3e.ThemeTransitionOptions) ui.Element                 = md3e.AnimatedThemeScope
		_ func(*ui.Context, *md3e.SnackbarState) ui.Element                                                 = md3e.SnackbarHost
		_ func(*ui.Context, *string, ...md3e.TextFieldOptions) ui.Element                                   = md3e.TextField
		_ func(*ui.Context, *string, ...md3e.TextFieldOptions) ui.Element                                   = md3e.TextArea
		_ func(*ui.Context, *float64, float64, float64, ...md3e.SliderOptions) ui.Element                   = md3e.Slider
		_ func(*ui.Context, *float64, *float64, float64, float64, ...md3e.RangeSliderOptions) ui.Element    = md3e.RangeSlider
		_ func(*ui.Context, []md3e.TreeNode, ...md3e.TreeViewOptions) ui.Element                            = md3e.TreeView
		_ func(*ui.Context, []md3e.DataTableColumn, int, func(int, int), ...md3e.DataTableOptions) ui.Element = md3e.DataTable
		_ func(*ui.Context, int, func(ui.ListRow), ...md3e.VirtualListOptions) ui.Element                   = md3e.VirtualList
		_ func(*ui.Context, []md3e.NavigationDestination, func(), ...md3e.AdaptiveScaffoldOptions) ui.Element = md3e.AdaptiveScaffold
		_ func(*ui.Context, []md3e.NavigationDestination, ...md3e.NavigationRailOptions) ui.Element         = md3e.NavigationRail
		_ func(*ui.Context, *bool, ...md3e.AlertDialogOptions) ui.Element                                   = md3e.AlertDialog
		_ func(*ui.Context, *bool, ...md3e.FullScreenDialogOptions) ui.Element                              = md3e.FullScreenDialog
		_ func(*ui.Context, *bool, ...md3e.BottomSheetOptions) ui.Element                                   = md3e.BottomSheet
	)

	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
		Dark:         true,
		Variant:      tokens.Expressive,
		Density:      tokens.DensityDesktop,
		Motion:       motion.Expressive,
		ReduceMotion: true,
	})

	var (
		text       = "hello"
		sliderVal  = 50.0
		lowVal     = 20.0
		highVal    = 80.0
		tabIdx     = 0
		navIdx     = 0
		railExp    = false
		splitOpen  = false
		dialogOpen = false
		fsOpen     = false
		sheetOpen  = false
		treeSel    any = "root"
		snack      md3e.SnackbarState
	)
	dests := []md3e.NavigationDestination{
		{Label: "Home", Icon: testIconDraw},
		{Label: "Settings", Icon: testIconDraw},
	}
	tabs := []md3e.TabItem{{Label: "One"}, {Label: "Two"}}
	cols := []md3e.DataTableColumn{{Title: "Name", Width: 120, Sortable: true}}
	nodes := []md3e.TreeNode{{Key: "root", Label: "Root"}}

	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			md3e.AdaptiveScaffold(c.Key("scaffold"), dests, func() {
				md3e.PageTransition(c.Key("pt"), navIdx, func() {
					ui.Column(c).FillWidth().Gap(6).Children(func() {
						md3e.WindowTitleBar(c.Key("tb"), md3e.WindowTitleBarOptions{Title: "Compat"})
						md3e.Card(c.Key("card"), func() {
							md3e.Button(c.Key("btn"), "Primary", md3e.ButtonOptions{
								Variant:     md3e.ButtonFilled,
								LeadingIcon: testIconDraw,
							})
							group, action, arrow, openPtr := md3e.SplitButton(c.Key("split"), "Save", md3e.SplitButtonOptions{
								Open: &splitOpen,
								Menu: func(panel ui.Element) {
									panel.Children(func() {
										ui.ButtonBase(c).Label("Save As")
									})
								},
							})
							if !group.Valid() || !action.Valid() || !arrow.Valid() || openPtr != &splitOpen {
								t.Errorf("SplitButton returned unexpected handles/open pointer")
							}
						}, md3e.CardOptions{Variant: md3e.CardElevated})
						md3e.PillTabs(c.Key("pills"), tabs, md3e.PillTabsOptions{Selected: &tabIdx})
						md3e.SegmentedTabs(c.Key("segs"), tabs, md3e.PillTabsOptions{Selected: &tabIdx})
						md3e.TextField(c.Key("tf"), &text, md3e.TextFieldOptions{Label: "Name", Placeholder: "Enter"})
						md3e.Slider(c.Key("sl"), &sliderVal, 0, 100, md3e.SliderOptions{Label: "Volume", Step: 10})
						md3e.RangeSlider(c.Key("rsl"), &lowVal, &highVal, 0, 100, md3e.RangeSliderOptions{Label: "Range"})
						md3e.TreeView(c.Key("tv"), nodes, md3e.TreeViewOptions{Selected: &treeSel})
						md3e.DataTable(c.Key("dt"), cols, 1, func(row, col int) {
							md3e.DataTableCellText(c, "row0")
						}).FillWidth().Height(80)
						md3e.VirtualList(c.Key("vl"), 1, func(r ui.ListRow) {
							ui.Text(r.Context, "item0")
						}).FillWidth().Height(48)
					})
				}, md3e.TransitionOptions{Mode: md3e.TransitionFadeThrough})
			}, md3e.AdaptiveScaffoldOptions{
				Selected:     &navIdx,
				RailExpanded: &railExp,
			})
			md3e.AlertDialog(c.Key("ad"), &dialogOpen, md3e.AlertDialogOptions{Headline: "Alert"})
			md3e.FullScreenDialog(c.Key("fsd"), &fsOpen, md3e.FullScreenDialogOptions{Title: "Editor"})
			md3e.BottomSheet(c.Key("bs"), &sheetOpen, md3e.BottomSheetOptions{Modal: true, Label: "Sheet"})
			md3e.SnackbarHost(c, &snack)
		}).Fill()
	}, 900, 720)

	tt.Frame()
	if !tt.HasText("Primary") || !tt.HasText("Save") {
		t.Fatal("legacy consumer fixture failed to render core controls")
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion compat fixture should settle with zero pending frames")
	}
}

// TestUpstreamSplitSecondLimitsGap verifies the 611bbf74402f SplitSecond,
// SplitLimits, and SplitGap capabilities required by M21 SideSheet.
func TestUpstreamSplitSecondLimitsGap(t *testing.T) {
	secondW := float32(300)
	var firstRect, secondRect ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Split(c.Key("sp"), &secondW, func() {
			ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) {
				firstRect = r
			})
		}, func() {
			ui.Box(c).Fill().Draw(func(_ *ui.Painter, r ui.Rect) {
				secondRect = r
			})
		}).
			SplitSecond().
			SplitGap(8).
			SplitLimits(ui.SplitLimits{FirstMin: 360, SecondMin: 240, SecondMax: 420}).
			Fill()
	}, 900, 400)

	tt.Frame()
	if secondRect.W < 299 || secondRect.W > 301 {
		t.Fatalf("second pane width = %.1f, want 300", secondRect.W)
	}
	wantFirst := float32(900 - 300 - 8)
	if firstRect.W < wantFirst-1 || firstRect.W > wantFirst+1 {
		t.Fatalf("first pane width = %.1f, want %.1f (SplitGap=8)", firstRect.W, wantFirst)
	}

	// Oversized request is clamped to SecondMax=420 without losing *secondW until dragged.
	secondW = 600
	tt.Frame()
	if secondRect.W < 419 || secondRect.W > 421 {
		t.Fatalf("clamped second pane width = %.1f, want 420 (SecondMax)", secondRect.W)
	}

	// Undersized request is clamped to SecondMin=240.
	secondW = 100
	tt.Frame()
	if secondRect.W < 239 || secondRect.W > 241 {
		t.Fatalf("clamped second pane width = %.1f, want 240 (SecondMin)", secondRect.W)
	}
}

// TestUpstreamTruncated verifies the 611bbf74402f Element.Truncated() API.
func TestUpstreamTruncated(t *testing.T) {
	var longTrunc, shortTrunc bool
	var frame1Long, frame1Short bool
	frame := 0
	tt := ui.NewTester(func(c *ui.Context) {
		frame++
		ui.Column(c).Width(80).Children(func() {
			longEl := ui.Text(c.Key("long"), "A very long label that cannot possibly fit in eighty DIPs").SingleLine().MaxWidth(80)
			longTrunc = longEl.Truncated()
			shortEl := ui.Text(c.Key("short"), "OK").SingleLine().MaxWidth(80)
			shortTrunc = shortEl.Truncated()
		})
		if frame == 1 {
			frame1Long, frame1Short = longTrunc, shortTrunc
		}
	}, 200, 120)

	// Frame 1 (built during NewTester) has no prior frame, so Truncated() is false.
	if frame1Long || frame1Short {
		t.Fatalf("frame 1 Truncated = (%v, %v), want (false, false)", frame1Long, frame1Short)
	}

	// Next frame reads frame 1's truncation result.
	tt.Frame()
	if !longTrunc {
		t.Fatal("subsequent frame expected long SingleLine text to report Truncated() == true")
	}
	if shortTrunc {
		t.Fatal("subsequent frame expected short SingleLine text to report Truncated() == false")
	}
}

// TestUpstreamAccordionDisabledHeaderSkip verifies 611bbf74402f AccordionBase
// arrow navigation skips disabled section headers.
func TestUpstreamAccordionDisabledHeaderSkip(t *testing.T) {
	var o1, o2, o3 bool
	var f1, f2, f3 bool
	tt := ui.NewTester(func(c *ui.Context) {
		acc := ui.AccordionBase(c.Key("acc"))
		acc.Root.FillWidth().Children(func() {
			it1 := acc.Item(&o1)
			it1.Trigger.Label("First").Height(32).Children(func() { ui.Text(c, "First") })
			f1 = it1.Trigger.Focused()
			it1.Panel(func() { ui.Text(c, "Body 1") })

			it2 := acc.Item(&o2)
			it2.Trigger.Label("Disabled Middle").Disabled(true).Height(32).Children(func() { ui.Text(c, "Disabled Middle") })
			f2 = it2.Trigger.Focused()
			it2.Panel(func() { ui.Text(c, "Body 2") })

			it3 := acc.Item(&o3)
			it3.Trigger.Label("Third").Height(32).Children(func() { ui.Text(c, "Third") })
			f3 = it3.Trigger.Focused()
			it3.Panel(func() { ui.Text(c, "Body 3") })
		})
	}, 320, 240)

	tt.Key(0, ui.KeyTab)
	tt.Frame()
	if !f1 || f2 || f3 {
		t.Fatalf("after Tab focus = (%v, %v, %v), want (true, false, false)", f1, f2, f3)
	}

	tt.Key(0, ui.KeyDown)
	tt.Frame()
	if f1 || f2 || !f3 {
		t.Fatalf("after KeyDown across disabled header focus = (%v, %v, %v), want (false, false, true)", f1, f2, f3)
	}
}

// TestUpstreamRouterEncodedSlashParam verifies 611bbf74402f Router preserves
// %2F inside a path segment so /docs/a%2Fb matches /docs/{id} with Param("id") == "a/b".
func TestUpstreamRouterEncodedSlashParam(t *testing.T) {
	rtr := ui.NewRouter("/docs/a%2Fb")
	rtr.Transition = ui.TransitionNone
	var matched bool
	var gotID string

	tt := ui.NewTester(func(c *ui.Context) {
		rtr.View(c, func(r *ui.Route) {
			if r.Match("/docs/{id}") {
				matched = true
				gotID = r.Param("id")
				ui.Text(c, "Doc: "+gotID)
			} else {
				matched = false
				gotID = ""
				ui.Text(c, "Not found")
			}
		}).Fill()
	}, 320, 200)

	tt.Frame()
	tt.Advance(50 * time.Millisecond)
	if !matched {
		t.Fatal("expected /docs/a%2Fb to match /docs/{id}")
	}
	if gotID != "a/b" {
		t.Fatalf("Route.Param(\"id\") = %q, want %q", gotID, "a/b")
	}
}
