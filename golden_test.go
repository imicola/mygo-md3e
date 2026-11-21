package md3e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/golden"

	"github.com/imicola/mygo-md3e/shape"
)

// goldenTolerance accepts a channel's one-step rounding difference in one pixel
// of a thousand, so the baselines hold across the platforms' CPUs while a real
// visual change still fails.
var goldenTolerance = golden.Tolerance{Channel: 1, PerMille: 1}

// matchGolden draws build on a fixed light theme with reduce motion so every
// spring lands at once, and compares the frame with testdata/name.png (or
// testdata/name_<GOOS>.png on non-Windows hosts when an OS baseline exists).
// Every baseline stays free of text, which the golden package notes differs
// from one platform to another; vector-path baselines can still differ across
// host rasterizers due to platform GammaRatios.
func matchGolden(t *testing.T, name string, width, height int, build func(c *ui.Context)) {
	t.Helper()
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Column(c).Fill().Background(th.Colors.Surface).Padding(16).Children(func() {
				build(c)
			})
		})
	}, width, height)
	tt.Advance(50 * time.Millisecond)
	golden.Match(t, tt, goldenPath(name), goldenTolerance)
}

func goldenPath(name string) string {
	path := filepath.Join("testdata", name+".png")
	if runtime.GOOS != "windows" {
		osPath := filepath.Join("testdata", name+"_"+runtime.GOOS+".png")
		if _, err := os.Stat(osPath); err == nil || os.Getenv("MYGO_UPDATE_GOLDEN") != "" {
			return osPath
		}
	}
	return path
}

// TestGoldenShapeMorphs covers the 35 built-in shapes as their morphs to a
// shape seven places on, halfway through, all in geometry alone.
func TestGoldenShapeMorphs(t *testing.T) {
	all := shape.All()
	morphs := make([]*shape.Morph, len(all))
	for i := range all {
		morphs[i] = shape.NewMorph(all[i].Polygon(), all[(i+7)%len(all)].Polygon())
	}
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	matchGolden(t, "shapes_morph", 7*96+6*8+32, 5*96+4*8+32, func(c *ui.Context) {
		cs := th.Colors
		ui.Column(c).Gap(8).Children(func() {
			for row := range 5 {
				ui.Row(c).Gap(8).Children(func() {
					for col := range 7 {
						cell := ui.Box(c).Size(96, 96).Radius(16).Background(cs.SurfaceContainerLowest)
						m := morphs[row*7+col]
						cell.Draw(func(p *ui.Painter, r ui.Rect) {
							if !cell.Valid() || r.W <= 0 || r.H <= 0 {
								return
							}
							shape.DrawMorph(p, m, 0.5, r, cs.Primary)
						})
					}
				})
			}
		})
	})
}

// TestGoldenLoadingIndicators covers the standard and contained loading
// indicators frozen by reduce motion at their first shape.
func TestGoldenLoadingIndicators(t *testing.T) {
	matchGolden(t, "loading_indicators", 4*72+3*24+32, 72+32, func(c *ui.Context) {
		ui.Row(c).Gap(24).AlignItems(ui.Center).Children(func() {
			LoadingIndicator(c, LoadingOptions{Style: LoadingStandard, Size: 48, Label: "Loading standard small"})
			LoadingIndicator(c, LoadingOptions{Style: LoadingStandard, Size: 72, Label: "Loading standard large"})
			LoadingIndicator(c, LoadingOptions{Style: LoadingContained, Size: 48, Label: "Loading contained small"})
			LoadingIndicator(c, LoadingOptions{Style: LoadingContained, Size: 72, Label: "Loading contained large"})
		})
	})
}

// TestGoldenProgress covers the linear, wavy linear, circular and wavy
// circular indicators at 40%, with their track gaps, in geometry alone.
func TestGoldenProgress(t *testing.T) {
	matchGolden(t, "progress", 360, 230, func(c *ui.Context) {
		ui.Column(c).Gap(18).Children(func() {
			LinearProgress(c, 0.4, ProgressOptions{Label: "Linear"})
			LinearWavyProgress(c, 0.4, ProgressOptions{Label: "Wavy linear"})
			ui.Row(c).Gap(24).Children(func() {
				CircularProgress(c, 0.4, ProgressOptions{Label: "Circular", Size: 96})
				CircularWavyProgress(c, 0.4, ProgressOptions{Label: "Wavy circular", Size: 96})
			})
		})
	})
}

// TestGoldenSelectionGeometry covers the switch tracks, the slider track and
// handle and the range slider at fixed values, with no labels drawn.
func TestGoldenSelectionGeometry(t *testing.T) {
	off, on := false, true
	value := 0.35
	low, high := 0.25, 0.7
	matchGolden(t, "selection_geometry", 320, 190, func(c *ui.Context) {
		ui.Column(c).Gap(16).Children(func() {
			ui.Row(c).Gap(16).Children(func() {
				Switch(c, &off)
				Switch(c, &on)
			})
			Slider(c, &value, 0, 1)
			RangeSlider(c, &low, &high, 0, 1)
		})
	})
}

// TestGoldenSegmentedList covers the expressive segmented list's corner
// rhythm, chosen row and gap with icon-only rows.
func TestGoldenSegmentedList(t *testing.T) {
	selected := 1
	matchGolden(t, "segmented_list", 360, 280, func(c *ui.Context) {
		cs := CurrentTheme(c).Colors
		slot := func() ui.Element {
			return ui.Box(c).Size(24, 24).Radius(6).Background(cs.OutlineVariant)
		}
		List(c, []ListItem{
			{Key: "one", Leading: slot},
			{Key: "two", Leading: slot},
			{Key: "three", Leading: slot},
			{Key: "four", Leading: slot},
		}, ListOptions{Segmented: true, Selected: &selected, Label: "Segments"})
	})
}
