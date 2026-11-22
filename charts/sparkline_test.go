package charts

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

func TestRingBufferOperationsAndZeroAlloc(t *testing.T) {
	// Zero/negative capacity is a safe no-op.
	empty := NewRingBuffer(0)
	empty.Push(42)
	if empty.Cap() != 0 || empty.Len() != 0 {
		t.Fatalf("zero-cap ring expected 0/0, got %d/%d", empty.Len(), empty.Cap())
	}
	if _, ok := empty.Latest(); ok {
		t.Fatal("empty ring Latest() should be false")
	}
	if !math.IsNaN(empty.At(0)) {
		t.Fatal("empty ring At(0) should return NaN")
	}

	rb := NewRingBuffer(4)
	rb.PushSlice(10, 20, 30)
	if rb.Len() != 3 || rb.Cap() != 4 {
		t.Fatalf("expected 3/4, got %d/%d", rb.Len(), rb.Cap())
	}
	rb.Push(40)
	rb.Push(50) // overwrites 10
	rb.Push(math.NaN())
	rb.Push(60) // buffer now: [40, 50, NaN, 60]

	want := []float64{40, 50, math.NaN(), 60}
	if rb.Len() != 4 {
		t.Fatalf("expected Len 4, got %d", rb.Len())
	}
	for i, w := range want {
		got := rb.At(i)
		if math.IsNaN(w) {
			if !math.IsNaN(got) {
				t.Fatalf("At(%d): want NaN, got %v", i, got)
			}
		} else if got != w {
			t.Fatalf("At(%d): want %v, got %v", i, w, got)
		}
	}
	if lo, hi, ok := rb.MinMax(); !ok || lo != 40 || hi != 60 {
		t.Fatalf("MinMax: want 40..60 true, got %v..%v %v", lo, hi, ok)
	}
	if latest, ok := rb.Latest(); !ok || latest != 60 {
		t.Fatalf("Latest: want 60 true, got %v %v", latest, ok)
	}
	seqBefore := rb.Sequence()
	rb.Clear()
	if rb.Len() != 0 || rb.Sequence() <= seqBefore {
		t.Fatalf("Clear failed: len=%d seq=%d", rb.Len(), rb.Sequence())
	}

	// Verify Push and CopyTo are zero-allocation once warmed.
	rb.Reset(64)
	for i := range 64 {
		rb.Push(float64(i))
	}
	dst := make([]float64, 0, 64)
	dst = rb.CopyTo(dst)
	if allocs := testing.AllocsPerRun(50, func() {
		rb.Push(12.5)
		dst = rb.CopyTo(dst)
	}); allocs != 0 {
		t.Fatalf("RingBuffer Push+CopyTo allocated %v allocs/op, want 0", allocs)
	}
}

func TestSeriesPaletteContrastAndThemes(t *testing.T) {
	seeds := []ui.Color{
		ui.RGB(0x67, 0x50, 0xA4),
		ui.RGB(0xE8, 0x6A, 0x17),
		ui.RGB(0x00, 0x6C, 0x4C),
	}
	for _, dark := range []bool{false, true} {
		for _, contrast := range []tokens.Contrast{tokens.ContrastStandard, tokens.ContrastHigh} {
			for _, seed := range seeds {
				th := md3e.ThemeFromSeed(seed, md3e.ThemeOptions{
					Dark:     dark,
					Contrast: contrast,
				})
				bg := md3e.SurfaceColorForElevation(th, tokens.Level1)
				minRatio := 3.0
				if contrast >= tokens.ContrastHigh {
					minRatio = 4.5
				}
				pal := SeriesPalette(th, 8)
				if len(pal) != 8 {
					t.Fatalf("SeriesPalette(8) returned %d", len(pal))
				}
				for i, col := range pal {
					ratio := tokens.ContrastRatio(col, bg)
					if ratio < minRatio-0.05 {
						t.Fatalf("dark=%v contrast=%v slot=%d ratio=%.2f < %.2f (col=%v bg=%v)",
							dark, contrast, i, ratio, minRatio, col, bg)
					}
				}
				// Verify first 4 colors are distinct.
				for i := range 4 {
					for j := i + 1; j < 4; j++ {
						if pal[i] == pal[j] {
							t.Fatalf("dark=%v contrast=%v slots %d and %d collided: %v", dark, contrast, i, j, pal[i])
						}
					}
				}
			}
		}
	}

	// Monochrome variant stays neutral gray (low chroma).
	monoTh := md3e.ThemeFromSeed(ui.RGB(0x67, 0x50, 0xA4), md3e.ThemeOptions{
		Variant: tokens.Monochrome,
		Dark:    true,
	})
	for i := range 5 {
		h := tokens.HCTFromColor(SeriesColor(monoTh, i))
		if h.Chroma > 4.0 {
			t.Fatalf("monochrome slot %d has unexpected chroma %.2f", i, h.Chroma)
		}
	}

	// SeriesColor itself must not allocate.
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor)
	if allocs := testing.AllocsPerRun(50, func() {
		_ = SeriesColor(th, 4)
	}); allocs != 0 {
		t.Fatalf("SeriesColor allocated %v allocs/op, want 0", allocs)
	}
}

func TestAreaSparklineRenderEdgesAndGhostSafety(t *testing.T) {
	rb := NewRingBuffer(16)
	rb.PushSlice(12, 24, 18, 45, 38, 62, 55, 74, 48, 80)

	// 1. Render smooth sparkline with bottom radius clip and end dot; verify non-blank pixels and zero pending frames.
	tt := ui.NewTester(func(c *ui.Context) {
		th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Dark: true})
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Background(th.Colors.SurfaceContainerLow).Children(func() {
				AreaSparkline(c.Key("sp"), AreaSparklineOptions{
					Ring:         rb,
					Height:       64,
					BottomRadius: 16,
					ShowEndDot:   true,
					Label:        "Throughput history",
				})
			})
		}).Fill()
	}, 240, 80)
	tt.Frame()
	img := tt.Image()
	if tt.FramePending() {
		t.Fatal("AreaSparkline must not request idle animation frames")
	}

	// Confirm pixels inside the curve region differ from the plain background.
	bgPix := img.RGBAAt(2, 76) // below sparkline
	nonBg := 0
	for y := 4; y < 62; y++ {
		for x := 4; x < 236; x++ {
			if img.RGBAAt(x, y) != bgPix {
				nonBg++
			}
		}
	}
	if nonBg < 150 {
		t.Fatalf("AreaSparkline rendered too few curve/fill pixels: %d", nonBg)
	}

	// Verify accessibility role & label.
	if _, ok := tt.AccessByRole(ui.RoleImage, "Throughput history"); !ok {
		t.Fatal("expected accessible image node for AreaSparkline")
	}

	// 2. Edge cases: empty, single sample, all NaN/Inf, NaN gaps, all curve modes.
	edgeCases := [][]float64{
		nil,
		{42},
		{math.NaN(), math.Inf(1), math.Inf(-1)},
		{10, 20, math.NaN(), 35, 25, 50},
		{5, 5, 5, 5},
	}
	for _, curve := range []CurveStyle{CurveSmooth, CurveLinear, CurveStepAfter, CurveStepBefore, CurveStep} {
		for _, vals := range edgeCases {
			et := ui.NewTester(func(c *ui.Context) {
				md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
					AreaSparkline(c, AreaSparklineOptions{
						Values:       vals,
						Curve:        curve,
						BaselineZero: true,
						Height:       40,
					})
				}).Fill()
			}, 160, 48)
			et.Frame()
			if et.FramePending() {
				t.Fatalf("edge case curve=%v left frame pending", curve)
			}
		}
	}

	// 3. Ghost-draw exit transition safety.
	show := true
	exitTr := ui.ElementTransition{
		Duration: 120 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}
	gt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, md3e.DefaultTheme(c), func() {
			if show {
				AreaSparkline(c.Key("ghost-sp"), AreaSparklineOptions{
					Ring:   rb,
					Height: 48,
				}).Transition(exitTr)
			}
		}).Fill()
	}, 200, 60)
	gt.Frame()
	show = false
	gt.Frame()
	gt.Advance(50 * time.Millisecond)
	gt.Advance(200 * time.Millisecond)
	if gt.FramePending() {
		t.Fatal("ghost-exiting AreaSparkline did not settle to zero pending frames")
	}
}

func TestAreaSparklineZeroAllocs(t *testing.T) {
	rb := NewRingBuffer(128)
	for i := range 128 {
		rb.Push(40 + 25*math.Sin(float64(i)*0.15))
	}
	var scratch sparklineScratch
	plotRect := ui.Rect{X: 0, Y: 4, W: 240, H: 44}

	// Warm up backing buffers once.
	for _, curve := range []CurveStyle{CurveSmooth, CurveLinear, CurveStep} {
		ok, _, _ := buildSparklinePaths(&scratch, plotRect, 48, rb, nil, 0, 100, curve, 480)
		if !ok {
			t.Fatalf("warmup failed for curve %v", curve)
		}
		allocs := testing.AllocsPerRun(50, func() {
			rb.Push(50)
			buildSparklinePaths(&scratch, plotRect, 48, rb, nil, 0, 100, curve, 480)
		})
		if allocs != 0 {
			t.Fatalf("buildSparklinePaths(curve=%v) allocated %v allocs/op, want 0", curve, allocs)
		}
	}

	// Also verify dense downsampling path (1024 samples into 120px) is 0 allocs once warmed.
	dense := NewRingBuffer(1024)
	for i := range 1024 {
		dense.Push(50 + 30*math.Cos(float64(i)*0.07))
	}
	buildSparklinePaths(&scratch, plotRect, 48, dense, nil, 0, 100, CurveSmooth, 120)
	if allocs := testing.AllocsPerRun(30, func() {
		dense.Push(55)
		buildSparklinePaths(&scratch, plotRect, 48, dense, nil, 0, 100, CurveSmooth, 120)
	}); allocs != 0 {
		t.Fatalf("dense buildSparklinePaths allocated %v allocs/op, want 0", allocs)
	}

	// Verify rounded-rect clip path builder is 0 allocs once warmed.
	buildRoundedRectPath(&scratch.clipPath, ui.Rect{W: 240, H: 48}, [4]float32{0, 0, 16, 16})
	if allocs := testing.AllocsPerRun(50, func() {
		buildRoundedRectPath(&scratch.clipPath, ui.Rect{W: 240, H: 48}, [4]float32{0, 0, 16, 16})
	}); allocs != 0 {
		t.Fatalf("buildRoundedRectPath allocated %v allocs/op, want 0", allocs)
	}
}

func BenchmarkAreaSparklineTick(b *testing.B) {
	rb := NewRingBuffer(128)
	for i := range 128 {
		rb.Push(40 + 25*math.Sin(float64(i)*0.15))
	}
	var scratch sparklineScratch
	plotRect := ui.Rect{X: 0, Y: 4, W: 240, H: 44}
	buildSparklinePaths(&scratch, plotRect, 48, rb, nil, 0, 100, CurveSmooth, 480)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		rb.Push(float64(i % 100))
		buildSparklinePaths(&scratch, plotRect, 48, rb, nil, 0, 100, CurveSmooth, 480)
	}
}
