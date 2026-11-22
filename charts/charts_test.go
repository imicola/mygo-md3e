package charts

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestLineTicksResolveOnlyVisibleLabels(t *testing.T) {
	xs := make([]float32, 10000)
	for i := range xs {
		xs[i] = float32(i) / float32(len(xs)-1)
	}
	labels := make([]string, len(xs))
	labels[2000] = "Milestone"
	labels[2001] = "Between ticks"
	ticks := buildLineXTicks(xs, labels, 6)
	got := make([]string, 0, len(ticks))
	for _, tick := range ticks {
		got = append(got, tick.Label)
	}
	want := []string{"1", "Milestone", "4001", "6001", "8001", "10000"}
	if !slices.Equal(got, want) {
		t.Errorf("visible labels = %q, want %q", got, want)
	}
	if got := lineXLabel(labels, 2001); got != "Between ticks" {
		t.Errorf("intermediate point label = %q", got)
	}
}

func TestLineChartInteractionKeyboardSettlingAndEdges(t *testing.T) {
	sel := -1
	onSelectCount := 0
	series := []LineSeries{
		{Name: "Download", Values: []float64{12, 34, 28, 62, 54, 80}, FillArea: true, ShowPoints: true},
		{Name: "Upload", Values: []float64{6, 14, math.NaN(), 24, 19, 32}},
	}
	labels := []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00"}

	tt := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Dark: true})
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(16).Background(th.Colors.Surface).Children(func() {
				LineChart(c.Key("net-line"), series, LineChartOptions{
					Labels:   labels,
					Height:   180,
					Selected: &sel,
					OnSelect: func(int) { onSelectCount++ },
					Label:    "Network traffic",
				})
			})
		}).Fill()
	}, 420, 260)

	tt.Frame()
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("LineChart spring entrance did not settle")
	}

	// Verify rendered pixels differ from plain surface background.
	img := tt.Image()
	bg := img.RGBAAt(2, 2)
	nonBg := 0
	for y := 40; y < 220; y += 2 {
		for x := 40; x < 400; x += 2 {
			if img.RGBAAt(x, y) != bg {
				nonBg++
			}
		}
	}
	if nonBg < 120 {
		t.Fatalf("LineChart rendered too few curve/grid pixels: %d", nonBg)
	}

	// Pointer hover over plot selects a point and shows RichTooltip; leaving clears selection.
	tt.Move(220, 120)
	tt.Frame()
	if sel < 0 || sel >= len(labels) {
		t.Fatalf("expected hover selection in 0..%d, got %d", len(labels)-1, sel)
	}
	tt.Move(2, 2)
	tt.Frame()
	if sel != -1 {
		t.Fatalf("expected leaving plot to clear selection, got %d", sel)
	}

	// Legend click toggles series focus without panic.
	tt.Click("Download")
	tt.Frame()
	tt.Advance(400 * time.Millisecond)

	// Keyboard navigation: focus plot via Tab/Click and use Right/End/Escape.
	tt.ClickAt(220, 120)
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	tt.Key(0, ui.KeyEnd)
	tt.Frame()
	if sel != len(labels)-1 {
		t.Fatalf("KeyEnd expected index %d, got %d", len(labels)-1, sel)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if sel != -1 {
		t.Fatalf("KeyEscape expected -1, got %d", sel)
	}
	if onSelectCount == 0 {
		t.Fatal("expected OnSelect callback to fire during interaction")
	}

	// ReduceMotion settles immediately with zero pending frames.
	rmTT := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{ReduceMotion: true})
		md3e.ThemeScope(c, th, func() {
			LineChart(c.Key("line-rm"), series, LineChartOptions{Labels: labels})
		}).Fill()
	}, 360, 240)
	rmTT.Frame()
	if rmTT.FramePending() {
		t.Fatal("LineChart under ReduceMotion must not request animation frames")
	}

	// Edge cases: empty, single point, all NaN/Inf, ghost-draw exit transition.
	show := true
	exitTr := ui.ElementTransition{Duration: 120 * time.Millisecond, Exit: &ui.Motion{Opacity: 0}}
	edgeTT := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			LineChart(c.Key("empty"), nil)
			LineChart(c.Key("single"), []LineSeries{{Name: "One", Values: []float64{42}}})
			LineChart(c.Key("nan"), []LineSeries{{Name: "Bad", Values: []float64{math.NaN(), math.Inf(1)}}})
			if show {
				LineChart(c.Key("ghost"), series, LineChartOptions{Labels: labels, Height: 100}).Transition(exitTr)
			}
		}).Fill()
	}, 360, 320)
	edgeTT.Frame()
	edgeTT.Advance(800 * time.Millisecond)
	show = false
	edgeTT.Frame()
	edgeTT.Advance(300 * time.Millisecond)
	if edgeTT.FramePending() {
		t.Fatal("LineChart ghost-draw exit did not settle")
	}
}

func TestBarChartGroupedStackedNegativeAndInteraction(t *testing.T) {
	sel := -1
	cats := []string{"Mon", "Tue", "Wed", "Thu"}
	series := []BarSeries{
		{Name: "API", Values: []float64{40, 65, 50, 80}},
		{Name: "CDN", Values: []float64{25, 30, 45, 35}},
	}

	for _, layout := range []BarLayout{BarGrouped, BarStacked} {
		sel = -1
		tt := ui.NewTester(func(c *ui.Context) {
			th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
				Dark:     layout == BarStacked,
				Contrast: tokens.ContrastStandard,
			})
			md3e.ThemeScope(c, th, func() {
				ui.Column(c).Fill().Padding(16).Background(th.Colors.Surface).Children(func() {
					BarChart(c.Key("bars"), series, BarChartOptions{
						Categories: cats,
						Layout:     layout,
						Height:     180,
						ShowValues: true,
						Selected:   &sel,
						Label:      "Weekly requests",
					})
				})
			}).Fill()
		}, 400, 260)

		tt.Frame()
		tt.Advance(900 * time.Millisecond)
		if tt.FramePending() {
			t.Fatalf("BarChart(layout=%v) spring entrance did not settle", layout)
		}

		// Hover plot and confirm category selection + RichTooltip.
		tt.Move(180, 120)
		tt.Frame()
		if sel < 0 || sel >= len(cats) {
			t.Fatalf("BarChart(layout=%v) expected hover selection in 0..%d, got %d", layout, len(cats)-1, sel)
		}
		tt.Move(2, 2)
		tt.Frame()
		if sel != -1 {
			t.Fatalf("BarChart(layout=%v) expected leave to clear selection, got %d", layout, sel)
		}
	}

	// Negative values & ReduceMotion zero-frame check.
	negTT := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{ReduceMotion: true})
		md3e.ThemeScope(c, th, func() {
			BarChart(c.Key("neg"), []BarSeries{
				{Name: "Delta", Values: []float64{-25, 15, -10, 40, math.NaN()}},
			}, BarChartOptions{
				Categories: []string{"Q1", "Q2", "Q3", "Q4", "Q5"},
			})
		}).Fill()
	}, 360, 240)
	negTT.Frame()
	if negTT.FramePending() {
		t.Fatal("BarChart under ReduceMotion must not request frames")
	}
}

func TestDonutChartHoverKeyboardRTLAndSettling(t *testing.T) {
	sel := -1
	slices := []DonutSlice{
		{Label: "Direct", Value: 50},
		{Label: "Proxy", Value: 30},
		{Label: "DNS", Value: 20},
	}

	// 1. LTR: entrance settle, hover hit test, legend toggle, keyboard navigation & accessibility.
	tt := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Dark: true})
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(16).Background(th.Colors.Surface).Children(func() {
				DonutChart(c.Key("donut"), slices, DonutChartOptions{
					Size:     180,
					Selected: &sel,
					Label:    "Traffic split",
				})
			})
		}).Fill()
	}, 420, 240)

	tt.Frame()
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("DonutChart entrance did not settle")
	}

	ringInfo, ok := tt.AccessByRole(ui.RoleGroup, "Traffic split")
	if !ok {
		t.Fatal("expected accessible RoleGroup for DonutChart")
	}
	if !strings.Contains(ringInfo.Value, "100") {
		t.Fatalf("expected resting DonutChart value to mention total 100, got %q", ringInfo.Value)
	}

	// Hover right-side ring (at ~3 o'clock, inside first slice 0..180 deg) -> selects slice 0.
	cx := ringInfo.Bounds.X + ringInfo.Bounds.W*0.5
	cy := ringInfo.Bounds.Y + ringInfo.Bounds.H*0.5
	tt.Move(cx+68, cy)
	tt.Frame()
	tt.Advance(700 * time.Millisecond)
	if sel != 0 {
		t.Fatalf("expected hovering 3 o'clock to select slice 0, got %d", sel)
	}
	if tt.FramePending() {
		t.Fatal("DonutChart active slice pop-out spring did not settle")
	}

	// Leave ring -> clears selection.
	tt.Move(2, 2)
	tt.Frame()
	tt.Advance(700 * time.Millisecond)
	if sel != -1 {
		t.Fatalf("expected leaving DonutChart to clear selection, got %d", sel)
	}

	// Legend click selects slice 1 ("Proxy").
	tt.Click("Proxy")
	tt.Frame()
	if sel != 1 {
		t.Fatalf("expected clicking Proxy legend to select slice 1, got %d", sel)
	}

	// Keyboard focus on ring: Home -> 0, Right -> 1, Escape -> -1.
	tt.ClickAt(cx, cy)
	tt.Frame()
	tt.Key(0, ui.KeyHome)
	tt.Frame()
	if sel != 0 {
		t.Fatalf("KeyHome expected 0, got %d", sel)
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if sel != 1 {
		t.Fatalf("LTR KeyRight expected 1, got %d", sel)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if sel != -1 {
		t.Fatalf("KeyEscape expected -1, got %d", sel)
	}

	// 2. RTL keyboard mirroring: KeyLeft advances forward from 0 -> 1.
	rtlSel := 0
	rtlTT := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{ReduceMotion: true})
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Direction(ui.RightToLeft).Fill().Padding(16).Children(func() {
				DonutChart(c.Key("donut-rtl"), slices, DonutChartOptions{
					Size:     160,
					Selected: &rtlSel,
					Label:    "RTL Donut",
				})
			})
		}).Fill()
	}, 380, 220)
	rtlTT.Frame()
	if rtlTT.FramePending() {
		t.Fatal("DonutChart under ReduceMotion must not request frames")
	}
	rtlInfo, _ := rtlTT.AccessByRole(ui.RoleGroup, "RTL Donut")
	rtlTT.ClickAt(rtlInfo.Bounds.X+rtlInfo.Bounds.W*0.5, rtlInfo.Bounds.Y+rtlInfo.Bounds.H*0.5)
	rtlTT.Frame()
	rtlTT.Key(0, ui.KeyLeft)
	rtlTT.Frame()
	if rtlSel != 1 {
		t.Fatalf("RTL KeyLeft expected advance 0 -> 1, got %d", rtlSel)
	}

	// 3. Empty / zero / NaN data renders empty ring without panic.
	emptyTT := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			DonutChart(c, []DonutSlice{
				{Label: "Zero", Value: 0},
				{Label: "NaN", Value: math.NaN()},
			}, DonutChartOptions{PieMode: true})
		}).Fill()
	}, 260, 220)
	emptyTT.Frame()
	emptyTT.Advance(600 * time.Millisecond)
	if emptyTT.FramePending() {
		t.Fatal("empty DonutChart left frames pending")
	}
}
