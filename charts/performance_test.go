package charts

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

var measuredStackedLayout []stackedGeomSeg

func BenchmarkStackedLayout(b *testing.B) {
	fractions := []float32{0.32, 0.18, 0, 0.12, 0.08, 0.10}
	colors := make([]ui.Color, len(fractions))
	track := ui.RGB(255, 255, 255)
	geometry := computeStackedLayout(nil, 384, fractions, 0.2, colors, track, 3, 6, 3, true)
	b.ReportAllocs()
	for b.Loop() {
		geometry = computeStackedLayout(geometry, 384, fractions, 0.2, colors, track, 3, 6, 3, true)
	}
	measuredStackedLayout = geometry
}

func BenchmarkLineChartLimitedTicks(b *testing.B) {
	values := make([]float64, 10000)
	for i := range values {
		values[i] = float64(i % 100)
	}
	series := []LineSeries{{Name: "Samples", Values: values, Color: ui.RGB(0, 0, 0)}}
	animate := false
	opts := LineChartOptions{Animate: &animate, MaxXLabels: 6}
	var builds int
	tt := ui.NewTester(func(c *ui.Context) {
		builds++
		LineChart(c, series, opts)
	}, 480, 260)
	before := builds
	b.ReportAllocs()
	for b.Loop() {
		tt.Frame()
	}
	if builds == before {
		b.Fatal("benchmark did not rebuild the view")
	}
}

func BenchmarkDashboardChartsSteady(b *testing.B) { benchmarkDashboardCharts(b, false) }
func BenchmarkSparklineTick(b *testing.B)         { benchmarkDashboardCharts(b, true) }

func benchmarkDashboardCharts(b *testing.B, tick bool) {
	ring := NewRingBuffer(128)
	for i := range 128 {
		ring.Push(40 + 25*math.Sin(float64(i)*0.15))
	}
	spark := AreaSparklineOptions{Ring: ring, Height: 44, Curve: CurveSmooth}
	th := md3e.ThemeFromSeed(md3e.DefaultSeedColor, md3e.ThemeOptions{Density: tokens.DensityDesktop})
	tt := ui.NewTester(func(c *ui.Context) {
		md3e.ThemeScope(c, th, func() {
			ui.Column(c).Fill().Padding(8).Gap(10).Children(func() {
				MetricCard(c.Key("download"), MetricCardOptions{Title: "Download", Value: "128.4", Unit: "MB/s", Sparkline: &spark}).FillWidth()
				MetricCard(c.Key("capacity"), MetricCardOptions{Title: "Capacity", Value: "64", Unit: "%", Progress: &MetricProgress{Value: 0.64}}).FillWidth()
			})
		}).Fill()
	}, 360, 240)
	for range 10 {
		tt.Frame()
	}

	b.ReportAllocs()
	sample := 0
	for b.Loop() {
		if tick {
			ring.Push(float64(sample % 100))
			sample++
		}
		tt.Frame()
	}
}
