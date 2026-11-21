package md3e

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestPillTabsClickKeyboardDisabledAndBadges(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	sel := 0
	var history []int

	items := []TabItem{
		{Label: "Overview", Icon: mdicon.Star.Draw},
		{Label: "Locked", Disabled: true},
		{Label: "Alerts", Badge: "12"},
		{Label: "Logs", BadgeDot: true},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(520, 140).Background(canvasRGB).Padding(20).Children(func() {
				PillTabs(c.Key("main-pills"), items, PillTabsOptions{
					Selected:  &sel,
					FullWidth: true,
					Label:     "Sections",
					Changed:   func(i int) { history = append(history, i) },
				})
			})
		})
	}, 520, 140)

	// Settle initial layout frame.
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("initial PillTabs frame should land immediately without sliding from 0")
	}

	listInfo, ok := tt.AccessByRole(ui.RoleTabList, "Sections")
	if !ok {
		t.Fatal("PillTabs missing RoleTabList")
	}
	if listInfo.Bounds.H < 43 || listInfo.Bounds.H > 45 {
		t.Fatalf("PillTabs comfortable height = %v, want 44", listInfo.Bounds.H)
	}

	overviewInfo, ok := tt.AccessByRole(ui.RoleTab, "Overview")
	if !ok || !overviewInfo.Selected {
		t.Fatalf("Overview tab access=%+v ok=%v", overviewInfo, ok)
	}
	alertsInfo, ok := tt.AccessByRole(ui.RoleTab, "Alerts")
	if !ok {
		t.Fatal("Alerts tab missing")
	}
	if alertsInfo.Description != "12" {
		t.Fatalf("Alerts tab badge Description = %q, want \"12\"", alertsInfo.Description)
	}

	// Click Alerts tab -> moves selection from 0 to 2.
	cx, cy := center(alertsInfo.Bounds)
	tt.ClickAt(cx, cy)
	if sel != 2 {
		t.Fatalf("after clicking Alerts, sel=%d want 2", sel)
	}
	tt.Move(2, 2)
	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("PillTabs sliding capsule did not settle")
	}

	// Verify active capsule painted PrimaryContainer inside Alerts bounds.
	img := tt.Image()
	px := img.RGBAAt(int(alertsInfo.Bounds.X)+6, int(alertsInfo.Bounds.Y+alertsInfo.Bounds.H/2))
	if !colorMatches(px, th.Colors.PrimaryContainer, 20) {
		t.Fatalf("Alerts active capsule color = %+v, want PrimaryContainer %+v", px, rgbOf(th.Colors.PrimaryContainer))
	}

	// Keyboard navigation: Right moves 2 -> 3, Home -> 0, Right skips disabled 1 -> 2, End -> 3.
	tt.Key(0, ui.KeyRight)
	if sel != 3 {
		t.Fatalf("KeyRight from 2 gave sel=%d want 3", sel)
	}
	tt.Key(0, ui.KeyHome)
	if sel != 0 {
		t.Fatalf("KeyHome gave sel=%d want 0", sel)
	}
	tt.Key(0, ui.KeyRight)
	if sel != 2 {
		t.Fatalf("KeyRight from 0 should skip disabled tab 1 -> sel=%d want 2", sel)
	}
	tt.Key(0, ui.KeyEnd)
	if sel != 3 {
		t.Fatalf("KeyEnd gave sel=%d want 3", sel)
	}

	tt.Advance(900 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("PillTabs did not settle after keyboard navigation")
	}
	if len(history) != 5 {
		t.Fatalf("Changed callbacks = %v, want 5 transitions", history)
	}
}

func TestPillTabsRTLMirrorAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	sel := 0
	items := []TabItem{
		{Label: "First"},
		{Label: "Second"},
		{Label: "Third"},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Direction(ui.RightToLeft).Size(440, 120).Background(canvasRGB).Padding(16).Children(func() {
				PillTabs(c.Key("rtl-pills"), items, PillTabsOptions{
					Selected:  &sel,
					FullWidth: true,
					Tone:      PillTabsToneSecondary,
				})
			})
		})
	}, 440, 120)
	tt.Frame()

	first, _ := tt.AccessByRole(ui.RoleTab, "First")
	third, _ := tt.AccessByRole(ui.RoleTab, "Third")
	if first.Bounds.X <= third.Bounds.X {
		t.Fatalf("RTL layout did not mirror tabs: First.X=%v Third.X=%v", first.Bounds.X, third.Bounds.X)
	}

	// Focus first tab and press Left (advances forward in RTL).
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyLeft)
	if sel != 1 {
		t.Fatalf("RTL KeyLeft from 0 gave sel=%d want 1", sel)
	}
	tt.Key(0, ui.KeyLeft)
	if sel != 2 {
		t.Fatalf("RTL KeyLeft from 1 gave sel=%d want 2", sel)
	}
	if tt.FramePending() {
		t.Fatal("ReduceMotion PillTabs left frames pending")
	}
}

func TestSegmentedTabsDensityAndInitialPlacement(t *testing.T) {
	wantH := map[tokens.Density]float32{
		tokens.DensityComfortable: 38,
		tokens.DensityDesktop:     32,
		tokens.DensityCompact:     28,
	}
	for d, want := range wantH {
		th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{Density: d})
		sel := 2
		items := []TabItem{
			{Label: "Day"},
			{Label: "Week"},
			{Label: "Month"},
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ThemeScope(c, th, func() {
				ui.Column(c).Size(360, 100).Background(canvasRGB).Padding(16).Children(func() {
					SegmentedTabs(c.Key("seg"), items, PillTabsOptions{
						Selected: &sel,
						Label:    "Range",
					})
				})
			})
		}, 360, 100)
		tt.Frame()

		// Must land directly on index 2 with no spring animation pending on first mount.
		if tt.FramePending() {
			t.Fatalf("density=%d: SegmentedTabs animated on initial mount instead of jumping", d)
		}
		info, ok := tt.AccessByRole(ui.RoleTabList, "Range")
		if !ok {
			t.Fatalf("density=%d: SegmentedTabs missing RoleTabList", d)
		}
		if math.Abs(float64(info.Bounds.H-want)) > 0.5 {
			t.Fatalf("density=%d: height=%v want %v", d, info.Bounds.H, want)
		}
		month, ok := tt.AccessByRole(ui.RoleTab, "Month")
		if !ok || !month.Selected {
			t.Fatalf("density=%d: Month should start selected", d)
		}
		px := tt.Image().RGBAAt(int(month.Bounds.X)+6, int(month.Bounds.Y+month.Bounds.H/2))
		if !colorMatches(px, th.Colors.SurfaceContainerLowest, 20) {
			t.Fatalf("density=%d: SegmentedTabs active capsule = %+v, want SurfaceContainerLowest %+v",
				d, px, rgbOf(th.Colors.SurfaceContainerLowest))
		}
	}
}

func TestPillTabsEmptyAndScrollableSafe(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Size(240, 160).Background(canvasRGB).Padding(12).Gap(12).Children(func() {
				PillTabs(c.Key("empty"), nil)
				PillTabs(c.Key("scroll"), []TabItem{
					{Label: "Alpha"},
					{Label: "Beta"},
					{Label: "Gamma"},
					{Label: "Delta"},
					{Label: "Epsilon"},
				}, PillTabsOptions{Scrollable: true})
			})
		})
	}, 240, 160)
	tt.Frame()
	tt.Advance(500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("empty/scrollable PillTabs left frames pending")
	}
}
