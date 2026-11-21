package md3e

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/tokens"
)

// progressCanvas builds a 420x300 white canvas holding build's indicators in a
// 320 DIP wide column.
func progressCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(420, 300).Background(canvasRGB).Padding(24).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Width(320).Gap(24).Children(func() {
					build(c)
				})
			})
		})
	}, 420, 300)
}

// progressInfo returns an indicator's access info by name.
func progressInfo(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleProgress, name)
	if !ok {
		t.Fatalf("%q progress indicator not in the access tree", name)
	}
	return info
}

// progressClass classifies a pixel on a bar as active, track, or neither. The
// track's tolerance is tight: a light antialiased edge of the active color can
// land near SurfaceContainerHighest otherwise.
func progressClass(px color.RGBA, active, track ui.Color) byte {
	switch {
	case absColorDiff(px, rgbOf(active)) <= 30:
		return 'a'
	case absColorDiff(px, rgbOf(track)) <= 8:
		return 't'
	}
	return '.'
}

// progressRow returns the classes along one row of an image.
func progressRow(img *image.RGBA, y int, x0, x1 int, active, track ui.Color) []byte {
	row := make([]byte, 0, x1-x0)
	for x := x0; x < x1; x++ {
		row = append(row, progressClass(img.RGBAAt(x, y), active, track))
	}
	return row
}

// progressRuns returns the start and end indexes of each run of b in row.
func progressRuns(row []byte, b byte) [][2]int {
	var runs [][2]int
	start := -1
	for i, v := range row {
		if v == b && start < 0 {
			start = i
		}
		if v != b && start >= 0 {
			runs = append(runs, [2]int{start, i - 1})
			start = -1
		}
	}
	if start >= 0 {
		runs = append(runs, [2]int{start, len(row) - 1})
	}
	return runs
}

// progressRunsAtLeast returns the runs of b in row that are at least minLen
// bytes long, skipping the antialiased specks a stroke's edge leaves.
func progressRunsAtLeast(row []byte, b byte, minLen int) [][2]int {
	var runs [][2]int
	for _, r := range progressRuns(row, b) {
		if r[1]-r[0]+1 >= minLen {
			runs = append(runs, r)
		}
	}
	return runs
}

// progressInk returns the bounding box of pixels of class b in box.
func progressInk(img *image.RGBA, box ui.Rect, b byte, active, track ui.Color) (ui.Rect, int) {
	minX, minY := int(box.X+box.W), int(box.Y+box.H)
	maxX, maxY := int(box.X), int(box.Y)
	count := 0
	for y := int(box.Y); y < int(box.Y+box.H); y++ {
		for x := int(box.X); x < int(box.X+box.W); x++ {
			if progressClass(img.RGBAAt(x, y), active, track) == b {
				count++
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			}
		}
	}
	return ui.Rect{X: float32(minX), Y: float32(minY), W: float32(maxX - minX + 1), H: float32(maxY - minY + 1)}, count
}

// TestLinearWavyDeterminate checks the active length tracks the value, the wave
// anchors at its ends, the track gap is clear, and the stop dot sits at the far
// end.
func TestLinearWavyDeterminate(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}
	tt := progressCanvas(th, func(c *ui.Context) {
		LinearWavyProgress(c, 0, ProgressOptions{Label: "Zero"})
		LinearWavyProgress(c, 0.5, ProgressOptions{Label: "Half"})
		LinearWavyProgress(c, 1, ProgressOptions{Label: "Full"})
	})
	for _, name := range []string{"Zero", "Half", "Full"} {
		info := progressInfo(t, tt, name)
		if info.Min != 0 || info.Max != 1 {
			t.Fatalf("%s range %v..%v, want 0..1", name, info.Min, info.Max)
		}
		if n := indicatorOverflowPixels(tt.Image(), info.Bounds); n != 0 {
			t.Fatalf("%s paints %d pixels outside its box", name, n)
		}
	}
	img := tt.Image()
	zero := progressInfo(t, tt, "Zero")
	half := progressInfo(t, tt, "Half")
	full := progressInfo(t, tt, "Full")

	// Zero: no active ink, only the track and its end dot.
	row := progressRow(img, int(zero.Bounds.Y+zero.Bounds.H/2), int(zero.Bounds.X), int(zero.Bounds.X+zero.Bounds.W), cs.Primary, track)
	if runs := progressRuns(row, 'a'); len(runs) != 0 {
		t.Fatalf("value 0 paints active ink in %v", runs)
	}
	if runs := progressRuns(row, 't'); len(runs) < 2 {
		t.Fatalf("value 0 has %d track runs, want the track and the stop dot", len(runs))
	}

	// Half: the active segment spans half the bar, its wave bulges, and its
	// ends sit on the center line.
	ink, count := progressInk(img, half.Bounds, 'a', cs.Primary, track)
	if count == 0 {
		t.Fatal("half paints no active ink")
	}
	wantLen := half.Bounds.W / 2
	if math.Abs(float64(ink.W-wantLen)) > 8 {
		t.Fatalf("half's active ink is %.1f DIP wide, want about %.1f", ink.W, wantLen)
	}
	if ink.H < 8 || ink.H > half.Bounds.H {
		t.Fatalf("half's wave spans %.1f DIP high, want an anchored wave taller than the 4 DIP stroke", ink.H)
	}
	// The wave is pinned to the center line at the start.
	start := int(ink.X) + 1
	run := 0
	for y := int(half.Bounds.Y); y < int(half.Bounds.Y+half.Bounds.H); y++ {
		if progressClass(img.RGBAAt(start, y), cs.Primary, track) == 'a' {
			run++
		}
	}
	if run > 7 {
		t.Fatalf("the wave is %d DIP high at its start, want it anchored to the stroke", run)
	}

	// The track gap and the stop dot: two track runs on the center line, the
	// last a short dot at the far end.
	row = progressRow(img, int(half.Bounds.Y+half.Bounds.H/2), int(half.Bounds.X), int(half.Bounds.X+half.Bounds.W), cs.Primary, track)
	trackRuns := progressRunsAtLeast(row, 't', 3)
	if len(trackRuns) < 2 {
		t.Fatalf("half has %d track runs, want the track and the stop dot", len(trackRuns))
	}
	last := trackRuns[len(trackRuns)-1]
	if last[1]-last[0]+1 > 8 || last[1] < len(row)-3 {
		t.Fatalf("the stop dot spans %v, want a short dot at the far end", last)
	}
	if gap := trackRuns[0][0] - (int(ink.X+ink.W) - int(half.Bounds.X)) - 1; gap < 2 {
		t.Fatalf("the track gap is %d DIP, want a clear gap", gap)
	}

	// Full: the bar reaches the far end.
	fullInk, _ := progressInk(img, full.Bounds, 'a', cs.Primary, track)
	if fullInk.W < full.Bounds.W-3 {
		t.Fatalf("value 1's ink is %.1f DIP wide, want the whole %.1f", fullInk.W, full.Bounds.W)
	}
	if fullInk.X+fullInk.W < full.Bounds.X+full.Bounds.W-3 {
		t.Fatalf("value 1 stops %.1f DIP short of the end", full.Bounds.X+full.Bounds.W-fullInk.X-fullInk.W)
	}
}

// TestLinearWavyIndeterminateFlows checks an indeterminate bar paints a moving
// band, and that reduced motion stills it without extra frames.
func TestLinearWavyIndeterminateFlows(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := progressCanvas(th, func(c *ui.Context) {
		LinearWavyProgress(c, -1, ProgressOptions{Label: "Flow"})
	})
	info := progressInfo(t, tt, "Flow")
	if info.Now >= info.Min {
		t.Fatalf("indeterminate bar reports value %v", info.Now)
	}
	if n := indicatorOverflowPixels(tt.Image(), info.Bounds); n != 0 {
		t.Fatalf("indeterminate bar paints %d pixels outside its box", n)
	}
	first := tt.Image()
	tt.Advance(300 * time.Millisecond)
	if n := indicatorBoxDiff(first, tt.Image(), info.Bounds); n < 20 {
		t.Fatalf("the indeterminate band moved only %d pixels", n)
	}

	stillTh := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	still := progressCanvas(stillTh, func(c *ui.Context) {
		LinearWavyProgress(c, -1, ProgressOptions{Label: "Still"})
		CircularWavyProgress(c, -1, ProgressOptions{Label: "StillRing"})
	})
	if still.FramePending() {
		t.Fatal("reduced-motion indeterminate indicators ask for frames")
	}
	before := still.Image()
	still.Advance(time.Second)
	if still.FramePending() {
		t.Fatal("reduced-motion indeterminate indicators keep asking for frames")
	}
	if n := indicatorBoxDiff(before, still.Image(), ui.Rect{W: 420, H: 300}); n != 0 {
		t.Fatalf("reduced-motion indicators repainted %d pixels", n)
	}
}

// TestLinearProgressRTL checks both linear indicators fill from the right and
// put the stop dot at the left end.
func TestLinearProgressRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}
	tt := progressCanvas(th, func(c *ui.Context) {
		LinearProgress(c, 0.5, ProgressOptions{Label: "RightFill"})
		LinearWavyProgress(c, 0.5, ProgressOptions{Label: "RightWave"})
	})
	tt.SetDirection(ui.RightToLeft)
	img := tt.Image()
	for _, name := range []string{"RightFill", "RightWave"} {
		info := progressInfo(t, tt, name)
		ink, count := progressInk(img, info.Bounds, 'a', cs.Primary, track)
		if count == 0 {
			t.Fatalf("%s paints no active ink", name)
		}
		mid := (ink.X + ink.W/2) - info.Bounds.X
		if float64(mid) < float64(info.Bounds.W)*0.6 {
			t.Fatalf("%s's active ink centers at %.1f, want it on the right half", name, mid)
		}
		if ink.X+ink.W < info.Bounds.X+info.Bounds.W-3 {
			t.Fatalf("%s stops %.1f DIP before the right edge", name, info.Bounds.X+info.Bounds.W-ink.X-ink.W)
		}
		row := progressRow(img, int(info.Bounds.Y+info.Bounds.H/2), int(info.Bounds.X), int(info.Bounds.X+info.Bounds.W), cs.Primary, track)
		trackRuns := progressRunsAtLeast(row, 't', 3)
		if len(trackRuns) < 2 {
			t.Fatalf("%s has %d track runs, want the track and the left-end stop dot", name, len(trackRuns))
		}
		first := trackRuns[0]
		if first[0] > 3 || first[1]-first[0]+1 > 8 {
			t.Fatalf("%s's stop dot spans %v, want it at the left end", name, first)
		}
	}
}

// TestCircularWavyProgress checks the determinate ring paints inside its box and
// that the indeterminate ring moves and waves over time.
func TestCircularWavyProgress(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := progressCanvas(th, func(c *ui.Context) {
		ui.Row(c).Gap(24).Children(func() {
			CircularWavyProgress(c, 0.5, ProgressOptions{Label: "HalfRing"})
			CircularProgress(c, 0.5, ProgressOptions{Label: "PlainRing"})
			CircularWavyProgress(c, -1, ProgressOptions{Label: "SpinRing"})
		})
	})
	cs := th.Colors
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}
	half := progressInfo(t, tt, "HalfRing")
	if n := indicatorOverflowPixels(tt.Image(), half.Bounds); n != 0 {
		t.Fatalf("circular indicator paints %d pixels outside its box", n)
	}
	if _, count := progressInk(tt.Image(), half.Bounds, 'a', cs.Primary, track); count < 60 {
		t.Fatalf("the determinate ring paints only %d active pixels", count)
	}
	plain := progressInfo(t, tt, "PlainRing")
	if n := indicatorOverflowPixels(tt.Image(), plain.Bounds); n != 0 {
		t.Fatalf("plain ring paints %d pixels outside its box", n)
	}
	spin := progressInfo(t, tt, "SpinRing")
	first := tt.Image()
	tt.Advance(300 * time.Millisecond)
	second := tt.Image()
	if n := indicatorBoxDiff(first, second, spin.Bounds); n < 30 {
		t.Fatalf("the spinning ring moved only %d pixels", n)
	}
	if n := indicatorOverflowPixels(second, spin.Bounds); n != 0 {
		t.Fatalf("the spinning ring paints %d pixels outside its box", n)
	}
}

// TestProgressDeterminateWavyFlows checks that a determinate wavy indicator
// keeps moving between its ends instead of freezing at phase 0.
func TestProgressDeterminateWavyFlows(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := progressCanvas(th, func(c *ui.Context) {
		LinearWavyProgress(c, 0.4, ProgressOptions{Label: "FlowBar"})
		CircularWavyProgress(c, 0.4, ProgressOptions{Label: "FlowRing", Size: 64})
	})
	if !tt.FramePending() {
		t.Fatal("a determinate wavy indicator asks for no frames")
	}
	bar := progressInfo(t, tt, "FlowBar")
	ring := progressInfo(t, tt, "FlowRing")
	before := tt.Image()
	tt.Advance(120 * time.Millisecond)
	after := tt.Image()
	if n := indicatorBoxDiff(before, after, bar.Bounds); n < 5 {
		t.Fatalf("the determinate bar's wave moved only %d pixels", n)
	}
	if n := indicatorBoxDiff(before, after, ring.Bounds); n < 5 {
		t.Fatalf("the determinate ring's wave moved only %d pixels", n)
	}
	if !tt.FramePending() {
		t.Fatal("the flowing determinate indicator stopped asking for frames")
	}
}

// TestProgressWavyTerminalStatesRest checks that 0, 1, disabled and reduced
// motion wavy indicators flatten and ask for no frames.
func TestProgressWavyTerminalStatesRest(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	rest := progressCanvas(th, func(c *ui.Context) {
		LinearWavyProgress(c, 0, ProgressOptions{Label: "ZeroBar"})
		CircularWavyProgress(c, 1, ProgressOptions{Label: "FullRing", Size: 64})
		LinearWavyProgress(c, 0.5, ProgressOptions{Label: "OffBar", Disabled: true})
		CircularWavyProgress(c, 0.5, ProgressOptions{Label: "OffRing", Size: 64, Disabled: true})
	})
	if rest.FramePending() {
		t.Fatal("terminal determinate wavy indicators ask for frames")
	}
	before := rest.Image()
	rest.Advance(500 * time.Millisecond)
	if rest.FramePending() {
		t.Fatal("terminal determinate wavy indicators keep asking for frames")
	}
	if n := indicatorBoxDiff(before, rest.Image(), ui.Rect{W: 420, H: 300}); n != 0 {
		t.Fatalf("terminal determinate wavy indicators repainted %d pixels", n)
	}

	stillTh := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	still := progressCanvas(stillTh, func(c *ui.Context) {
		LinearWavyProgress(c, 0.5, ProgressOptions{Label: "StillBar"})
		CircularWavyProgress(c, 0.5, ProgressOptions{Label: "StillRing", Size: 64})
	})
	if still.FramePending() {
		t.Fatal("reduced-motion determinate wavy indicators ask for frames")
	}
	stillBefore := still.Image()
	still.Advance(300 * time.Millisecond)
	if still.FramePending() {
		t.Fatal("reduced-motion determinate wavy indicators keep asking for frames")
	}
	if n := indicatorBoxDiff(stillBefore, still.Image(), ui.Rect{W: 420, H: 300}); n != 0 {
		t.Fatalf("reduced-motion determinate wavy indicators changed %d pixels", n)
	}
}

// TestCircularIndeterminateTrackLeavesGapAroundArc checks that the unfinished
// ring stops clear of the moving active arc instead of running underneath it.
func TestCircularIndeterminateTrackLeavesGapAroundArc(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}
	tt := progressCanvas(th, func(c *ui.Context) {
		CircularWavyProgress(c, -1, ProgressOptions{Label: "GapRing", Size: 64})
	})
	tt.Advance(500 * time.Millisecond)
	info := progressInfo(t, tt, "GapRing")
	cx, cy := center(info.Bounds)
	img := tt.Image()
	for deg := 0; deg < 360; deg += 2 {
		rad := float64(deg) * math.Pi / 180
		anyActive, trackRun, trackDepth := false, 0, 0
		for r := 20.0; r <= 32.0; r += 0.5 {
			x := int(cx + float32(r*math.Sin(rad)))
			y := int(cy - float32(r*math.Cos(rad)))
			px := img.RGBAAt(x, y)
			if colorMatches(px, cs.Primary, 40) {
				anyActive = true
			}
			// A solid track stroke spans its whole 4 DIP thickness; the
			// antialiased rim of the active stroke can flash the track color
			// for a pixel or two and must not count.
			if colorMatches(px, track, 6) {
				trackRun++
				trackDepth = max(trackDepth, trackRun)
			} else {
				trackRun = 0
			}
		}
		if anyActive && trackDepth >= 4 {
			t.Fatalf("the track runs under the active arc at %d°", deg)
		}
	}
}

// TestProgressAccessAndBounds checks the range exposed to assistive technology,
// NaN normalization, disabled state, and a determinate value settling.
func TestProgressAccessAndBounds(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := progressCanvas(th, func(c *ui.Context) {
		LinearProgress(c, 0.5, ProgressOptions{Label: "Upload"})
		LinearWavyProgress(c, math.NaN(), ProgressOptions{Label: "Broken"})
		CircularProgress(c, 0.5, ProgressOptions{Label: "Off", Disabled: true})
		LinearProgress(c, 0.25, ProgressOptions{Label: "Settle"})
	})
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("determinate indicators keep asking for frames")
	}
	upload := progressInfo(t, tt, "Upload")
	if upload.Min != 0 || upload.Max != 1 || math.Abs(upload.Now-0.5) > 1e-6 {
		t.Fatalf("Upload range %v..%v at %v, want 0..1 at 0.5", upload.Min, upload.Max, upload.Now)
	}
	broken := progressInfo(t, tt, "Broken")
	if broken.Now != 0 {
		t.Fatalf("NaN value reads %v, want 0", broken.Now)
	}
	off := progressInfo(t, tt, "Off")
	if !off.Disabled {
		t.Fatal("a disabled indicator does not read as disabled")
	}
	settle := progressInfo(t, tt, "Settle")
	if math.Abs(settle.Now-0.25) > 1e-6 {
		t.Fatalf("Settle reads %v, want 0.25", settle.Now)
	}
}

// TestProgressAncestorDisabledUsesDisabledTokens checks that an indicator inside
// a disabled container reads as disabled and paints the MD3E disabled tokens,
// not Primary, even though its own Disabled option is false.
func TestProgressAncestorDisabledUsesDisabledTokens(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	wantActive := cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity)).Over(canvasRGB)
	wantTrack := cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)).Over(canvasRGB)
	tt := progressCanvas(th, func(c *ui.Context) {
		// The container disables the indicators without ui's 0.5 dimming, so
		// that the MD3E disabled tokens show at their own opacity.
		ui.Column(c).Disabled(true).DimDisabled(false).FillWidth().Gap(24).Children(func() {
			LinearProgress(c, 1, ProgressOptions{Label: "Off bar"})
			CircularWavyProgress(c, 1, ProgressOptions{Label: "Off ring"})
		})
	})
	bar := progressInfo(t, tt, "Off bar")
	ring := progressInfo(t, tt, "Off ring")
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("determinate disabled indicators keep asking for frames")
	}
	for _, info := range []ui.AccessInfo{bar, ring} {
		if !info.Disabled {
			t.Fatalf("%q under a disabled container does not read as disabled", info.Name)
		}
	}
	img := tt.Image()
	px := img.RGBAAt(int(bar.Bounds.X+bar.Bounds.W/2), int(bar.Bounds.Y+bar.Bounds.H/2))
	if !colorMatches(px, wantActive, 18) {
		t.Fatalf("disabled bar %+v is not the disabled content ink %+v", px, rgbOf(wantActive))
	}
	if colorMatches(px, cs.Primary, 18) {
		t.Fatalf("disabled bar still paints Primary %+v", rgbOf(cs.Primary))
	}
	// The ring must be entirely in the disabled palette: its active arc is the
	// disabled content ink and its track the disabled container ink.
	active, track := 0, 0
	for y := int(ring.Bounds.Y); y < int(ring.Bounds.Y+ring.Bounds.H); y++ {
		for x := int(ring.Bounds.X); x < int(ring.Bounds.X+ring.Bounds.W); x++ {
			got := img.RGBAAt(x, y)
			switch {
			case colorMatches(got, cs.Primary, 18):
				t.Fatalf("disabled ring paints Primary %+v at (%d,%d)", rgbOf(cs.Primary), x, y)
			case colorMatches(got, wantActive, 14):
				active++
			case colorMatches(got, wantTrack, 14):
				track++
			}
		}
	}
	if active < 40 {
		t.Fatalf("disabled ring paints only %d disabled-content pixels", active)
	}
	if track > 0 && active+track < 80 {
		t.Fatalf("disabled ring paints %d content and %d container pixels", active, track)
	}
}

// TestProgressGeometrySmoothTrackSmallValueAndWrapContinuity checks smooth
// wavy tracks, circular small values, ReduceMotion linear band visibility, and
// wrap continuity across the 1800ms cycle boundary.
func TestProgressGeometrySmoothTrackSmallValueAndWrapContinuity(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}

	// Circular wavy track uses zero amplitude (same smooth track as non-wavy)
	// even at determinate values near zero; 0.01 paints a minimal active dot.
	tt := progressCanvas(th, func(c *ui.Context) {
		ui.Row(c).Gap(24).Children(func() {
			CircularWavyProgress(c, 0, ProgressOptions{Label: "ZeroWave", Size: 64})
			CircularWavyProgress(c, 0.01, ProgressOptions{Label: "TinyWave", Size: 64})
		})
	})
	zeroInfo := progressInfo(t, tt, "ZeroWave")
	tinyInfo := progressInfo(t, tt, "TinyWave")
	img := tt.Image()
	if _, activeZero := progressInk(img, zeroInfo.Bounds, 'a', cs.Primary, track); activeZero != 0 {
		t.Fatalf("value=0 circular wavy painted %d active pixels, want 0", activeZero)
	}
	if _, activeTiny := progressInk(img, tinyInfo.Bounds, 'a', cs.Primary, track); activeTiny < 4 {
		t.Fatalf("value=0.01 circular wavy painted %d active pixels, want visible start dot", activeTiny)
	}

	// Indeterminate circular ring rotation across progressCycle (1800ms) must
	// change continuously instead of jumping 180 degrees.
	ringTT := progressCanvas(th, func(c *ui.Context) {
		CircularWavyProgress(c, -1, ProgressOptions{Label: "ContinuousRing", Size: 64})
	})
	ringInfo := progressInfo(t, ringTT, "ContinuousRing")
	ringTT.Advance(900 * time.Millisecond)
	midA := ringTT.Image()
	ringTT.Advance(20 * time.Millisecond)
	midB := ringTT.Image()
	midDiff := indicatorBoxDiff(midA, midB, ringInfo.Bounds)

	ringTT.Advance(progressCycle - 920*time.Millisecond - 10*time.Millisecond)
	beforeWrap := ringTT.Image()
	ringTT.Advance(20 * time.Millisecond)
	afterWrap := ringTT.Image()
	wrapDiff := indicatorBoxDiff(beforeWrap, afterWrap, ringInfo.Bounds)
	if wrapDiff > max(120, midDiff*2) {
		t.Fatalf("circular indeterminate jumped across 1800ms cycle wrap (%d pixels vs mid-cycle %d)", wrapDiff, midDiff)
	}

	// ReduceMotion indeterminate linear bar still renders a centered active band.
	rmTh := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	rmTT := progressCanvas(rmTh, func(c *ui.Context) {
		LinearWavyProgress(c, -1, ProgressOptions{Label: "RMBar"})
	})
	rmInfo := progressInfo(t, rmTT, "RMBar")
	rmInk, rmCount := progressInk(rmTT.Image(), rmInfo.Bounds, 'a', rmTh.Colors.Primary, track)
	if rmCount < 20 {
		t.Fatalf("ReduceMotion indeterminate linear painted only %d active pixels", rmCount)
	}
	mid := (rmInk.X + rmInk.W/2) - rmInfo.Bounds.X
	if absF32(mid-rmInfo.Bounds.W/2) > rmInfo.Bounds.W*0.15 {
		t.Fatalf("ReduceMotion indeterminate linear band centered at %.1f of %.1f, want centered", mid, rmInfo.Bounds.W)
	}
}

// TestProgressDisabledIndeterminateStopsFrames checks disabled indeterminate
// progress indicators emit no continuous animation frames and paint disabled tokens.
func TestProgressDisabledIndeterminateStopsFrames(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	tt := progressCanvas(th, func(c *ui.Context) {
		ui.Column(c).Disabled(true).DimDisabled(false).Gap(16).FillWidth().Children(func() {
			LinearWavyProgress(c, -1, ProgressOptions{Label: "DisIndBar"})
			CircularWavyProgress(c, -1, ProgressOptions{Label: "DisIndRing"})
		})
	})
	if tt.FramePending() {
		t.Fatal("disabled indeterminate indicators requested animation frames")
	}
	before := tt.Image()
	tt.Advance(500 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("disabled indeterminate indicators kept requesting frames")
	}
	if diff := indicatorBoxDiff(before, tt.Image(), ui.Rect{W: 420, H: 300}); diff != 0 {
		t.Fatalf("disabled indeterminate indicators changed %d pixels", diff)
	}
}

// TestProgressExitTransitionGhostStaysVisibleAndSettles verifies progress
// indicators keep painting frozen waves on ElementTransition exit ghosts and
// stop requesting frames once exited.
func TestProgressExitTransitionGhostStaysVisibleAndSettles(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	exitTr := ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}
	show := true
	tt := progressCanvas(th, func(c *ui.Context) {
		if show {
			LinearWavyProgress(c.Key("exit-wave"), 0.5, ProgressOptions{Label: "ExitWave"}).Transition(exitTr)
			CircularWavyProgress(c.Key("exit-ring"), -1, ProgressOptions{Label: "ExitRing", Size: 64}).Transition(exitTr)
		}
	})
	waveBounds := progressInfo(t, tt, "ExitWave").Bounds
	ringBounds := progressInfo(t, tt, "ExitRing").Bounds
	tt.Advance(100 * time.Millisecond)

	show = false
	tt.Frame()
	tt.Advance(40 * time.Millisecond)
	midExit := tt.Image()

	tt.Advance(400 * time.Millisecond)
	if tt.FramePending() {
		t.Fatal("exited progress ghost left pending animation frames")
	}
	settled := tt.Image()
	if diff := indicatorBoxDiff(midExit, settled, waveBounds); diff < 40 {
		t.Fatalf("exiting linear wavy ghost rendered only %d differing pixels vs settled canvas", diff)
	}
	if diff := indicatorBoxDiff(midExit, settled, ringBounds); diff < 60 {
		t.Fatalf("exiting circular wavy ghost rendered only %d differing pixels vs settled canvas", diff)
	}
}

// TestLinearIndeterminateTwoSegmentContinuous verifies that both LinearProgress
// and LinearWavyProgress render continuous non-empty active segments across all
// phases of the 1800ms cycle (0ms, 300ms, 900ms, 1500ms and wrap) without blank
// frames, and settle to a centered bar with zero pending frames under ReduceMotion.
func TestLinearIndeterminateTwoSegmentContinuous(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}
	tt := progressCanvas(th, func(c *ui.Context) {
		LinearProgress(c, -1, ProgressOptions{Label: "IndLinear"})
		LinearWavyProgress(c, -1, ProgressOptions{Label: "IndWavy"})
	})
	linInfo := progressInfo(t, tt, "IndLinear")
	wavInfo := progressInfo(t, tt, "IndWavy")

	// Sample at 0ms, 300ms, 900ms, 1500ms, and 1800ms (wrap).
	targets := []time.Duration{
		0,
		300 * time.Millisecond,
		900 * time.Millisecond,
		1500 * time.Millisecond,
		1800 * time.Millisecond,
	}
	var atZero, atWrap *image.RGBA
	var elapsed time.Duration
	for _, target := range targets {
		if delta := target - elapsed; delta > 0 {
			tt.Advance(delta)
			elapsed = target
		}
		img := tt.Image()
		if target == 0 {
			atZero = img
		}
		if target == 1800*time.Millisecond {
			atWrap = img
		}
		_, linCount := progressInk(img, linInfo.Bounds, 'a', cs.Primary, track)
		_, wavCount := progressInk(img, wavInfo.Bounds, 'a', cs.Primary, track)
		if linCount < 200 {
			t.Fatalf("at %v, LinearProgress(-1) painted only %d active pixels (want continuous full bar)", target, linCount)
		}
		if wavCount < 200 {
			t.Fatalf("at %v, LinearWavyProgress(-1) painted only %d active pixels (want continuous full wave)", target, wavCount)
		}
	}
	// Wrap continuity: 0ms and 1800ms should match within tiny raster rounding.
	if diff := indicatorBoxDiff(atZero, atWrap, linInfo.Bounds); diff > 20 {
		t.Fatalf("LinearProgress(-1) jumped at 1800ms wrap (%d differing pixels)", diff)
	}
	if diff := indicatorBoxDiff(atZero, atWrap, wavInfo.Bounds); diff > 20 {
		t.Fatalf("LinearWavyProgress(-1) jumped at 1800ms wrap (%d differing pixels)", diff)
	}

	// Verify mathematical segment invariant over every 1% of the cycle: never empty, never overlapping.
	for step := range 100 {
		u := float64(step) / 100.0
		segs, count := indeterminateLinearSegments(u, false)
		if count == 0 {
			t.Fatalf("u=%.2f produced 0 segments", u)
		}
		total := 0.0
		for i := range count {
			if segs[i].hi <= segs[i].lo {
				t.Fatalf("u=%.2f invalid segment %+v", u, segs[i])
			}
			if i > 0 && segs[i].lo < segs[i-1].hi {
				t.Fatalf("u=%.2f overlapping segments %+v and %+v", u, segs[i-1], segs[i])
			}
			total += segs[i].hi - segs[i].lo
		}
		if total < 0.30 {
			t.Fatalf("u=%.2f active coverage %.3f < 0.30", u, total)
		}
	}

	// ReduceMotion: static centered bar, zero frames pending.
	rmTh := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	rmTT := progressCanvas(rmTh, func(c *ui.Context) {
		LinearProgress(c, -1, ProgressOptions{Label: "RMLin"})
		LinearWavyProgress(c, -1, ProgressOptions{Label: "RMWav"})
	})
	if rmTT.FramePending() {
		t.Fatal("ReduceMotion indeterminate linear indicators requested frames")
	}
	for _, name := range []string{"RMLin", "RMWav"} {
		info := progressInfo(t, rmTT, name)
		ink, count := progressInk(rmTT.Image(), info.Bounds, 'a', rmTh.Colors.Primary, track)
		if count < 100 {
			t.Fatalf("%s ReduceMotion painted only %d active pixels", name, count)
		}
		mid := (ink.X + ink.W/2) - info.Bounds.X
		if absF32(mid-info.Bounds.W/2) > info.Bounds.W*0.15 {
			t.Fatalf("%s ReduceMotion bar centered at %.1f of %.1f, want centered", name, mid, info.Bounds.W)
		}
	}
}

// TestProgressBufferTrackAndRTL checks ProgressOptions.Buffer across linear,
// circular, zero-value, non-finite, and RTL layouts, ensuring TrackGap
// separation between active, buffer, and track segments.
func TestProgressBufferTrackAndRTL(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}
	wantBuf := cs.Primary.Alpha(0.28).Over(canvasRGB)

	countBuffer := func(img *image.RGBA, box ui.Rect) int {
		n := 0
		for y := int(box.Y); y < int(box.Y+box.H); y++ {
			for x := int(box.X); x < int(box.X+box.W); x++ {
				px := img.RGBAAt(x, y)
				if colorMatches(px, wantBuf, 14) && !colorMatches(px, track, 8) && !colorMatches(px, cs.Primary, 30) {
					n++
				}
			}
		}
		return n
	}

	bufferCenterRun := func(img *image.RGBA, box ui.Rect) (startX, endX float32, ok bool) {
		y := int(box.Y + box.H/2)
		row := make([]byte, int(box.W))
		for i := range row {
			px := img.RGBAAt(int(box.X)+i, y)
			if colorMatches(px, wantBuf, 14) && !colorMatches(px, track, 8) && !colorMatches(px, cs.Primary, 30) {
				row[i] = 'b'
			}
		}
		runs := progressRunsAtLeast(row, 'b', 8)
		if len(runs) == 0 {
			return 0, 0, false
		}
		best := runs[0]
		for _, r := range runs[1:] {
			if r[1]-r[0] > best[1]-best[0] {
				best = r
			}
		}
		return box.X + float32(best[0]), box.X + float32(best[1]+1), true
	}

	tt := progressCanvas(th, func(c *ui.Context) {
		LinearProgress(c, 0.25, ProgressOptions{Label: "BufLinear", Buffer: 0.65})
		LinearWavyProgress(c, 0.25, ProgressOptions{Label: "BufWavy", Buffer: 0.65})
		LinearProgress(c, 0, ProgressOptions{Label: "BufZeroVal", Buffer: 0.50})
		LinearProgress(c, 0.40, ProgressOptions{Label: "BufNaN", Buffer: math.NaN()})
		ui.Row(c).Gap(16).Children(func() {
			CircularProgress(c, 0.25, ProgressOptions{Label: "BufRing", Size: 64, Buffer: 0.65})
			CircularWavyProgress(c, 0.25, ProgressOptions{Label: "BufWavyRing", Size: 64, Buffer: 0.65})
		})
	})
	img := tt.Image()

	for _, name := range []string{"BufLinear", "BufWavy"} {
		info := progressInfo(t, tt, name)
		actInk, actN := progressInk(img, info.Bounds, 'a', cs.Primary, track)
		bufStart, bufEnd, ok := bufferCenterRun(img, info.Bounds)
		if actN < 50 || !ok {
			t.Fatalf("%s active=%d bufferRun=%v, want visible active and buffer segments", name, actN, ok)
		}
		// Buffer must start to the right of active with a clear TrackGap (>= 2 DIP).
		gap := bufStart - (actInk.X + actInk.W)
		if gap < 2 {
			t.Fatalf("%s gap between active and buffer is %.1f DIP, want >= 2", name, gap)
		}
		// Track must resume to the right of buffer with a clear TrackGap.
		row := progressRow(img, int(info.Bounds.Y+info.Bounds.H/2), int(info.Bounds.X), int(info.Bounds.X+info.Bounds.W), cs.Primary, track)
		trackRuns := progressRunsAtLeast(row, 't', 3)
		if len(trackRuns) < 2 {
			t.Fatalf("%s has %d track runs, want unfinished track and stop dot", name, len(trackRuns))
		}
		trackStartX := info.Bounds.X + float32(trackRuns[0][0])
		bufTrackGap := trackStartX - bufEnd
		if bufTrackGap < 2 {
			t.Fatalf("%s gap between buffer and track is %.1f DIP, want >= 2", name, bufTrackGap)
		}
	}

	zeroVal := progressInfo(t, tt, "BufZeroVal")
	if _, actN := progressInk(img, zeroVal.Bounds, 'a', cs.Primary, track); actN != 0 {
		t.Fatalf("BufZeroVal painted %d active pixels, want 0", actN)
	}
	if bufN := countBuffer(img, zeroVal.Bounds); bufN < 100 {
		t.Fatalf("BufZeroVal painted %d buffer pixels, want visible buffer segment", bufN)
	}

	nanInfo := progressInfo(t, tt, "BufNaN")
	if bufN := countBuffer(img, nanInfo.Bounds); bufN != 0 {
		t.Fatalf("BufNaN painted %d buffer pixels, want 0", bufN)
	}

	for _, name := range []string{"BufRing", "BufWavyRing"} {
		info := progressInfo(t, tt, name)
		if bufN := countBuffer(img, info.Bounds); bufN < 30 {
			t.Fatalf("%s painted only %d buffer pixels, want visible buffer arc", name, bufN)
		}
	}

	// RTL layout mirrors active -> buffer -> track from right to left.
	rtlTT := progressCanvas(th, func(c *ui.Context) {
		LinearProgress(c, 0.25, ProgressOptions{Label: "RTLBuf", Buffer: 0.65})
	})
	rtlTT.SetDirection(ui.RightToLeft)
	rtlImg := rtlTT.Image()
	rtlInfo := progressInfo(t, rtlTT, "RTLBuf")
	actInk, actN := progressInk(rtlImg, rtlInfo.Bounds, 'a', cs.Primary, track)
	_, bufEnd, ok := bufferCenterRun(rtlImg, rtlInfo.Bounds)
	if actN < 50 || !ok {
		t.Fatalf("RTLBuf active=%d bufferRun=%v, want visible segments", actN, ok)
	}
	if gap := actInk.X - bufEnd; gap < 2 {
		t.Fatalf("RTLBuf expected buffer left of active with gap >= 2, got gap %.1f (bufEnd=%.1f act=%+v)", gap, bufEnd, actInk)
	}
}

// TestProgressOffScreenZeroFrames checks that all four indicator types stop
// requesting animation frames when clipped outside p.Visible() in a scroll view.
func TestProgressOffScreenZeroFrames(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	offscreenTT := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Scroll(c).Size(240, 100).Background(canvasRGB).Children(func() {
				ui.Column(c).Width(220).Gap(12).Children(func() {
					ui.Box(c).Size(220, 260) // push indicators completely below the 100 DIP viewport
					LinearProgress(c, -1, ProgressOptions{Label: "OffLin"})
					LinearWavyProgress(c, -1, ProgressOptions{Label: "OffLinWav"})
					LinearWavyProgress(c, 0.4, ProgressOptions{Label: "OffLinDetWav"})
					CircularProgress(c, -1, ProgressOptions{Label: "OffCirc", Size: 48})
					CircularWavyProgress(c, -1, ProgressOptions{Label: "OffCircWav", Size: 48})
					CircularWavyProgress(c, 0.4, ProgressOptions{Label: "OffCircDetWav", Size: 48})
				})
			})
		})
	}, 240, 100)
	offscreenTT.Frame()
	if offscreenTT.FramePending() {
		t.Fatal("off-screen progress indicators must not request frames when outside p.Visible()")
	}
	before := offscreenTT.Image()
	offscreenTT.Advance(300 * time.Millisecond)
	if offscreenTT.FramePending() {
		t.Fatal("off-screen progress indicators kept requesting frames after Advance")
	}
	if diff := indicatorBoxDiff(before, offscreenTT.Image(), ui.Rect{W: 240, H: 100}); diff != 0 {
		t.Fatalf("off-screen progress indicators repainted %d pixels", diff)
	}
}
