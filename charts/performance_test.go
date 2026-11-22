package charts

import (
	"testing"

	"github.com/egoist/mygo/ui"
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
