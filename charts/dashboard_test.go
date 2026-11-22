package charts

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestStackedProgressSegmentsRTLHoverAndSettling(t *testing.T) {
	red := ui.RGB(0xE5, 0x39, 0x35)
	blue := ui.RGB(0x1E, 0x88, 0xE5)

	segs := []StackedSegment{
		{Label: "Active", Value: 50, Color: red},
		{Label: "Cache", Value: 25, Color: blue},
	}

	// 1. LTR layout with remainder (Total = 100), legend, hover tooltip, and spring settling.
	ltrTT := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Dark: true})
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(16).Background(th.Colors.Surface).Children(func() {
				StackedProgress(c.Key("mem"), segs, StackedProgressOptions{
					Total:       100,
					Height:      14,
					ShowLegend:  true,
					ShowPercent: true,
					Label:       "Memory usage",
				})
			})
		}).Fill()
	}, 320, 120)
	ltrTT.Frame()
	ltrTT.Advance(800 * time.Millisecond)
	if ltrTT.FramePending() {
		t.Fatal("StackedProgress spring did not settle")
	}

	// Verify accessibility meter node and auto-generated value breakdown.
	info, ok := ltrTT.AccessByRole(ui.RoleMeter, "Memory usage")
	if !ok {
		t.Fatal("expected accessible RoleMeter for StackedProgress")
	}
	if !strings.Contains(info.Value, "Active: 50%") || !strings.Contains(info.Value, "Cache: 25%") {
		t.Fatalf("unexpected StackedProgress accessible value: %q", info.Value)
	}

	// Pixel check in LTR: first segment (red) must occupy left half of bar, second (blue) right of midpoint.
	img := ltrTT.Image()
	leftPx := img.RGBAAt(60, 23)
	midRightPx := img.RGBAAt(185, 23)
	if leftPx.R <= leftPx.B+40 {
		t.Fatalf("LTR expected red-dominant pixel on left segment, got %v", leftPx)
	}
	if midRightPx.B <= midRightPx.R+40 {
		t.Fatalf("LTR expected blue-dominant pixel on second segment, got %v", midRightPx)
	}

	// Hover first segment and verify floating tooltip renders cleanly without panic/pending frames.
	ltrTT.Move(60, 23)
	ltrTT.Frame()
	ltrTT.Advance(200 * time.Millisecond)
	if ltrTT.FramePending() {
		t.Fatal("StackedProgress hover left pending frames")
	}

	// 2. RTL mirroring: first segment (red) must appear on the RIGHT side of the bar!
	rtlTT := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Dark: true})
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Direction(ui.RightToLeft).Fill().Padding(16).Background(th.Colors.Surface).Children(func() {
				StackedProgress(c.Key("mem-rtl"), segs, StackedProgressOptions{
					Total:  100,
					Height: 14,
					Label:  "Memory RTL",
				})
			})
		}).Fill()
	}, 320, 80)
	rtlTT.Frame()
	rtlTT.Advance(800 * time.Millisecond)
	rtlImg := rtlTT.Image()
	rightPx := rtlImg.RGBAAt(260, 23)
	if rightPx.R <= rightPx.B+40 {
		t.Fatalf("RTL expected red-dominant first segment on right side, got %v", rightPx)
	}

	// 3. ReduceMotion settles on first frame with zero pending frames.
	rmTT := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{ReduceMotion: true})
		md3e.ThemeScope(c, th, func() {
			StackedProgress(c.Key("mem-rm"), segs, StackedProgressOptions{Total: 100})
		}).Fill()
	}, 260, 60)
	rmTT.Frame()
	if rmTT.FramePending() {
		t.Fatal("StackedProgress under ReduceMotion must not have pending frames")
	}

	// 4. Empty / NaN / negative values must render safely without panic.
	edgeTT := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			StackedProgress(c, []StackedSegment{
				{Label: "NaN", Value: math.NaN()},
				{Label: "Neg", Value: -10},
				{Label: "Inf", Value: math.Inf(1)},
			}, StackedProgressOptions{ShowLegend: true})
		}).Fill()
	}, 240, 80)
	edgeTT.Frame()
	edgeTT.Advance(400 * time.Millisecond)
	if edgeTT.FramePending() {
		t.Fatal("edge-case StackedProgress left frames pending")
	}
}

func TestMetricCardSparklineProgressInteractionAndDensity(t *testing.T) {
	rb := NewRingBuffer(12)
	rb.PushSlice(20, 35, 30, 52, 48, 70, 64, 88)

	dummyIcon := func(p *ui.Painter, r ui.Rect, c ui.Color) {
		p.Fill(r, c, 4)
	}

	clicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
			Dark:    true,
			Density: tokens.DensityDesktop,
		})
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(16).Gap(12).Background(th.Colors.Surface).Children(func() {
				MetricCard(c.Key("throughput"), MetricCardOptions{
					Title:     "Download",
					Subtitle:  "6 edge nodes",
					Icon:      dummyIcon,
					Badge:     "Live",
					BadgeTone: md3e.ToneSuccess,
					Value:     "128.4",
					Unit:      "MB/s",
					Secondary: "Peak 210 MB/s",
					Clickable: true,
					OnClick:   func() { clicks++ },
					Sparkline: &AreaSparklineOptions{
						Ring:       rb,
						ShowEndDot: true,
					},
				})
				MetricCard(c.Key("storage"), MetricCardOptions{
					Title:   "Storage",
					Value:   "72%",
					Variant: md3e.CardOutlined,
					Progress: &MetricProgress{
						Value:  0.72,
						Buffer: 0.85,
						Wavy:   false,
					},
				})
			})
		}).Fill()
	}, 360, 340)

	tt.Frame()
	tt.Advance(500 * time.Millisecond)

	// Verify accessible clickable MetricCard name and click callback.
	btnInfo, ok := tt.AccessByRole(ui.RoleButton, "Download: 128.4 MB/s")
	if !ok {
		t.Fatal("expected accessible button for clickable MetricCard")
	}
	tt.ClickAt(btnInfo.Bounds.X+btnInfo.Bounds.W/2, btnInfo.Bounds.Y+24)
	tt.Frame()
	if clicks != 1 {
		t.Fatalf("expected 1 MetricCard click, got %d", clicks)
	}
	tt.Advance(600 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("MetricCard pressed corner spring did not settle")
	}

	// Disabled MetricCard must block clicks.
	disabledClicks := 0
	dt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			MetricCard(c.Key("dis-card"), MetricCardOptions{
				Title:    "Offline",
				Value:    "0",
				Disabled: true,
				OnClick:  func() { disabledClicks++ },
			})
		}).Fill()
	}, 260, 120)
	dt.Frame()
	dt.ClickAt(100, 40)
	dt.Frame()
	if disabledClicks != 0 {
		t.Fatal("disabled MetricCard must not invoke OnClick")
	}

	// Wavy progress MetricCard under ReduceMotion settles with zero pending frames.
	wt := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{
			Density:      tokens.DensityCompact,
			ReduceMotion: true,
		})
		md3e.ThemeScope(c, th, func() {
			MetricCard(c.Key("wavy-card"), MetricCardOptions{
				Title: "Sync",
				Value: "64%",
				Progress: &MetricProgress{
					Value: 0.64,
					Wavy:  true,
				},
			})
		}).Fill()
	}, 260, 140)
	wt.Frame()
	if wt.FramePending() {
		t.Fatal("MetricCard with LinearWavyProgress under ReduceMotion must not request frames")
	}
}
