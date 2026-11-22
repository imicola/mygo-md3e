package charts

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

// LineSeries defines one data series plotted by LineChart.
type LineSeries struct {
	// Name identifies the series in the legend, tooltip, and screen-reader output.
	Name string
	// Values are Y coordinates aligned by index with LineChartOptions.Labels.
	// NaN values break the line cleanly as missing data.
	Values []float64
	// Color overrides the series color; assigned from SeriesColor(th, index) when A == 0.
	Color ui.Color
	// FillArea paints a multi-stop vertical gradient fade beneath the curve.
	FillArea bool
	// FillAlpha sets the top opacity of FillArea (0..1); defaults to 0.26.
	FillAlpha float32
	// StrokeWidth sets the curve stroke width in DIP; defaults to 2.4.
	StrokeWidth float32
	// ShowPoints renders resting circular dots at every finite data point.
	ShowPoints bool
}

// LineChartOptions configures LineChart.
type LineChartOptions struct {
	// Labels are the X-axis tick labels aligned with series Values indices.
	// When empty, 1-based numeric indices are generated automatically.
	Labels []string
	// Curve selects the line interpolation style (default CurveSmooth).
	Curve CurveStyle
	// Height sets the plot area height in DIP (default 220).
	Height float32
	// YMin and YMax optionally fix the vertical Y domain when HasYDomain is true
	// or when both are finite and YMax > YMin.
	YMin, YMax float64
	HasYDomain bool
	// BaselineZero includes 0 in an auto-scaled Y domain.
	BaselineZero bool
	// YTicks is the target number of Y-axis ticks (default 4).
	YTicks int
	// MaxXLabels thins X-axis labels when len(Labels) exceeds this limit.
	MaxXLabels int
	// ShowGrid draws horizontal MD3E reference grid lines (nil defaults to true).
	ShowGrid *bool
	// Glow paints a subtle luminous under-stroke beneath curves (nil defaults to true).
	Glow *bool
	// ShowLegend renders an interactive legend that focuses individual series on
	// click (nil defaults to true when len(series) > 1).
	ShowLegend *bool
	// ShowTooltip enables the floating MD3E RichTooltip readout on hover/keyboard
	// selection (nil defaults to true).
	ShowTooltip *bool
	// Animate plays an MD3E DefaultSpatial spring entrance reveal, snapping
	// immediately under ReduceMotion (nil defaults to true).
	Animate *bool
	// Selected optionally binds the active X index (-1 = none). When nil,
	// LineChart keeps selection internally via ui.Local.
	Selected *int
	// OnSelect runs whenever the selected X index changes via pointer or keyboard.
	OnSelect func(index int)
	// ValueFormatter formats Y-axis ticks and tooltip values.
	ValueFormatter func(v float64) string
	// TooltipBuilder optionally overrides the floating tooltip title and rows.
	TooltipBuilder func(index int) (title string, rows []TooltipRow)
	// Label names the chart for assistive technology (default "Line chart").
	Label string
	// Description provides additional accessible context.
	Description string
}

type lineChartLocalKey struct{}

type lineChartState struct {
	selected      int
	focusedSeries int
	scratches     []sparklineScratch
	xFractions    []float32
	gridPath      ui.Path
}

// LineChart builds an MD3E Expressive multi-series line/area chart over
// github.com/egoist/mygo/ui/chart with HCT-harmonized colors, subtle curve
// under-glow, multi-stop gradient area fills, spring entrance reveal,
// interactive legend series focus, and hover/keyboard RichTooltip readouts.
func LineChart(c *ui.Context, series []LineSeries, opts ...LineChartOptions) ui.Element {
	if len(opts) > 1 {
		panic("charts: LineChart takes at most one LineChartOptions")
	}
	var o LineChartOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	animate := boolDefault(o.Animate, true)
	showGrid := boolDefault(o.ShowGrid, true)
	glow := boolDefault(o.Glow, true)
	showLegend := boolDefault(o.ShowLegend, len(series) > 1)
	showTooltip := boolDefault(o.ShowTooltip, true)
	height := sanitizePositiveF32(o.Height, 220)
	yTickCount := o.YTicks
	if yTickCount <= 0 {
		yTickCount = 4
	}
	fmtVal := o.ValueFormatter
	if fmtVal == nil {
		fmtVal = defaultFormatNumber
	}

	// Determine point count and resolve series colors.
	maxN := len(o.Labels)
	colors := make([]ui.Color, len(series))
	for i, s := range series {
		if len(s.Values) > maxN {
			maxN = len(s.Values)
		}
		col := s.Color
		if col.A == 0 {
			col = SeriesColor(th, i)
		}
		colors[i] = col
	}

	// Compute Y scale domain safely across all finite values.
	yScale := resolveLineYScale(series, o.YMin, o.YMax, o.HasYDomain, o.BaselineZero, yTickCount, fmtVal)
	yTicks := yScale.Ticks(yTickCount)

	label := o.Label
	if label == "" {
		label = "Line chart"
	}

	root := ui.Column(c).FillWidth().MinWidth(0).Gap(10)
	st := ui.Local(root, lineChartLocalKey{}, func() lineChartState {
		return lineChartState{selected: -1, focusedSeries: -1}
	})
	if st.focusedSeries >= len(series) {
		st.focusedSeries = -1
	}
	if len(st.scratches) != len(series) {
		grown := make([]sparklineScratch, len(series))
		copy(grown, st.scratches)
		st.scratches = grown
	}

	// Positions depend only on the point count. Replace the slice on a
	// resize so old draw callbacks still see the positions they captured.
	if len(st.xFractions) != maxN {
		st.xFractions = make([]float32, maxN)
		switch {
		case maxN == 1:
			st.xFractions[0] = 0.5
		case maxN > 1:
			inv := float32(1) / float32(maxN-1)
			for i := range st.xFractions {
				st.xFractions[i] = float32(i) * inv
			}
		}
	}
	xs := st.xFractions
	xTicks := buildLineXTicks(xs, o.Labels, o.MaxXLabels)

	selPtr := o.Selected
	if selPtr == nil {
		selPtr = &st.selected
	}
	if *selPtr >= maxN {
		*selPtr = -1
	}

	enter := animateEntrance(root, th, animate, reduce)
	revealClip := clampF32(enter, 0, 1)
	riseScale := clampF32(enter, 0, 1.06)

	surfaceBg := md3e.SurfaceColorForElevation(th, tokens.Level1)
	if surfaceBg.A == 0 {
		surfaceBg = cs.Surface
	}
	gridCol := gridLineColor(th)
	crosshairCol := cs.OutlineVariant
	if th.Contrast >= tokens.ContrastHigh && cs.Outline.A > 0 {
		crosshairCol = cs.Outline
	}
	dark := th.Dark

	root.Children(func() {
		if showLegend && len(series) > 0 {
			items := make([]chartLegendItem, len(series))
			for i, s := range series {
				name := s.Name
				if name == "" {
					name = fmt.Sprintf("Series %d", i+1)
				}
				items[i] = chartLegendItem{Name: name, Color: colors[i]}
			}
			renderChartLegend(c, th, items, st.focusedSeries, func(idx int) {
				if st.focusedSeries == idx {
					st.focusedSeries = -1
				} else {
					st.focusedSeries = idx
				}
			})
		}

		axes := chart.Axes{
			Left:   yTicks,
			Bottom: xTicks,
			Label:  md3eAxisLabel(c, th),
		}

		plot := chart.Plot(c.Key("plot"), axes, func(area ui.Element) {
			area.Label(label)
			if o.Description != "" {
				area.Description(o.Description)
			}

			describe := func(idx int) string {
				if idx < 0 || idx >= maxN {
					return ""
				}
				var parts []string
				for si, s := range series {
					if idx >= len(s.Values) || !isFiniteF64(s.Values[idx]) {
						continue
					}
					name := s.Name
					if name == "" {
						name = fmt.Sprintf("Series %d", si+1)
					}
					parts = append(parts, fmt.Sprintf("%s %s", name, fmtVal(s.Values[idx])))
				}
				if len(parts) == 0 {
					return lineXLabel(o.Labels, idx)
				}
				return lineXLabel(o.Labels, idx) + ": " + strings.Join(parts, ", ")
			}

			if maxN > 0 {
				if chart.Select(c, area, selPtr, xs, describe) && o.OnSelect != nil {
					o.OnSelect(*selPtr)
				}
			}

			// Copy the caller's series headers for ghost draws. Colors and ticks
			// are already owned by this build.
			seriesSnap := make([]LineSeries, len(series))
			copy(seriesSnap, series)
			focusedSnap := st.focusedSeries
			scratches := st.scratches
			yLo, yHi := yScale.Lo, yScale.Hi
			baseFrac := clampF32(yScale.At(max(yLo, min(yHi, 0))), 0, 1)
			curve := o.Curve

			gridPath := &st.gridPath
			baselineInk := cs.OutlineVariant
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
					return
				}
				if !rectIntersects(r, p.Visible()) {
					return
				}

				// Dashed horizontal reference gridlines & crisp 0-baseline when in domain.
				if showGrid {
					drawDashedGrid(p, gridPath, r, yTicks, gridCol)
				}
				if yLo <= 0 && yHi >= 0 {
					zy := chart.Y(r, baseFrac)
					p.Line(r.X, zy, r.X+r.W, zy, 1.2, baselineInk)
				}
				if maxN == 0 || len(seriesSnap) == 0 || !(yHi > yLo) {
					return
				}

				paintSeries := func() {
					pixelW := int(math.Ceil(float64(r.W * max(1, p.Scale()))))
					for si, s := range seriesSnap {
						if len(s.Values) == 0 {
							continue
						}
						col := colors[si]
						dimmed := focusedSnap >= 0 && focusedSnap != si
						strokeW := sanitizePositiveF32(s.StrokeWidth, 2.4)
						if focusedSnap == si {
							strokeW += 0.6
						}
						if dimmed {
							col = col.Alpha(0.26)
						}

						// Build path into per-series scratch with spring rise around baseline.
						sc := &scratches[si]
						sc.linePath.Reset()
						sc.areaPath.Reset()
						hasLine := buildLineSeriesPaths(sc, r, xs, s.Values, yLo, yHi, baseFrac, riseScale, curve, pixelW)
						if !hasLine {
							continue
						}

						if s.FillArea {
							topA := s.FillAlpha
							if !isPositiveFiniteF32(topA) {
								topA = 0.26
							}
							topA = clampF32(topA, 0, 1)
							if dimmed {
								topA *= 0.25
							}
							midA := topA * 0.35
							if dark {
								midA = topA * 0.42
							}
							sc.stops[0] = ui.GradientStop{Color: col.Alpha(topA), At: 0}
							sc.stops[1] = ui.GradientStop{Color: col.Alpha(midA), At: 0.58}
							sc.stops[2] = ui.GradientStop{Color: col.Alpha(0), At: 1}
							p.FillPathGradient(&sc.areaPath, ui.LinearGradient{
								Angle: 180,
								Stops: sc.stops[:],
							})
						}

						if glow && !dimmed {
							ga := float32(0.16)
							if dark {
								ga = 0.24
							}
							if focusedSnap == si {
								ga += 0.10
							}
							p.StrokePath(&sc.linePath, strokeW*2.6, col.Alpha(ga))
						}
						p.StrokePath(&sc.linePath, strokeW, col)

						if s.ShowPoints && !dimmed {
							invSpan := 1.0 / (yHi - yLo)
							for i, v := range s.Values {
								if i >= len(xs) || !isFiniteF64(v) {
									continue
								}
								yf := clampF32(float32((v-yLo)*invSpan), 0, 1)
								yf = baseFrac + (yf-baseFrac)*riseScale
								px, py := chart.X(r, xs[i]), chart.Y(r, clampF32(yf, 0, 1))
								p.Fill(ui.Rect{X: px - 4, Y: py - 4, W: 8, H: 8}, surfaceBg, 4)
								p.Fill(ui.Rect{X: px - 2.6, Y: py - 2.6, W: 5.2, H: 5.2}, col, 2.6)
							}
						}
					}
				}

				if revealClip < 0.999 {
					clipRect := ui.Rect{X: r.X - 6, Y: r.Y - 12, W: r.W*revealClip + 12, H: r.H + 24}
					p.Clip(clipRect, 0, paintSeries)
				} else {
					paintSeries()
				}
			})

			// Active selection crosshair, ringed point markers, and MD3E RichTooltip card.
			if idx := *selPtr; idx >= 0 && idx < maxN {
				selX := xs[idx]
				chart.RuleX(c, selX, 1).BorderWidth(0, 0, 0, 1).BorderColor(crosshairCol).BorderStyle(ui.BorderDashed).PassThrough()

				var tipRows []TooltipRow
				topY := float32(0.5)
				foundPoint := false
				for si, s := range series {
					if idx >= len(s.Values) || !isFiniteF64(s.Values[idx]) {
						continue
					}
					if st.focusedSeries >= 0 && st.focusedSeries != si {
						continue
					}
					yf := clampF32(yScale.At(s.Values[idx]), 0, 1)
					yf = clampF32(baseFrac+(yf-baseFrac)*riseScale, 0, 1)
					if !foundPoint || yf > topY {
						topY = yf
						foundPoint = true
					}
					col := colors[si]
					// Outer translucent halo + surface ring + crisp dot.
					chart.Point(c, selX, yf, 18, 18).Radius(9).Background(col.Alpha(0.22)).PassThrough()
					chart.Point(c, selX, yf, 10, 10).Radius(5).Background(col).Border(2, surfaceBg).PassThrough()

					name := s.Name
					if name == "" {
						name = fmt.Sprintf("Series %d", si+1)
					}
					tipRows = append(tipRows, TooltipRow{
						Name:  name,
						Value: fmtVal(s.Values[idx]),
						Color: col,
					})
				}

				if showTooltip {
					tipTitle := lineXLabel(o.Labels, idx)
					if o.TooltipBuilder != nil {
						tipTitle, tipRows = o.TooltipBuilder(idx)
					}
					if tipTitle != "" || len(tipRows) > 0 {
						hAlign := ui.Start
						marginL, marginR := float32(10), float32(0)
						if selX > 0.55 {
							hAlign = ui.End
							marginL, marginR = 0, 10
						}
						anchorY := clampF32(topY, 0.25, 0.92)
						vAlign := ui.Start
						if anchorY < 0.45 {
							vAlign = ui.End
						}
						chart.Anchor(c, selX, anchorY, hAlign, vAlign, func() {
							ui.Box(c).Margin(0, marginR, 0, marginL).PassThrough().Children(func() {
								renderChartTooltipCard(c, th, tipTitle, tipRows, "", "")
							})
						}).PassThrough()
					}
				}
			}
		}).Height(height).Gap(8).Label(label)
		if o.Description != "" {
			plot.Description(o.Description)
		}
	})

	return root
}

func resolveLineYScale(series []LineSeries, yMin, yMax float64, hasDomain, baselineZero bool, tickCount int, fmtVal func(float64) string) chart.Linear {
	if (hasDomain || (isFiniteF64(yMin) && isFiniteF64(yMax) && yMax > yMin)) &&
		isFiniteF64(yMin) && isFiniteF64(yMax) && yMax > yMin {
		sc := chart.NewLinear(yMin, yMax)
		sc.Format = fmtVal
		return sc
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	found := false
	for _, s := range series {
		for _, v := range s.Values {
			if !isFiniteF64(v) {
				continue
			}
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
			found = true
		}
	}
	if !found {
		sc := chart.NewLinear(0, 100)
		sc.Format = fmtVal
		return sc
	}
	if baselineZero {
		if lo > 0 {
			lo = 0
		}
		if hi < 0 {
			hi = 0
		}
	}
	if hi == lo {
		pad := math.Abs(lo) * 0.2
		if pad == 0 {
			pad = 1
		}
		lo -= pad
		hi += pad
	}
	sc := chart.NewLinear(lo, hi).Nice(tickCount)
	sc.Format = fmtVal
	return sc
}

func lineXLabel(labels []string, index int) string {
	if index < len(labels) && labels[index] != "" {
		return labels[index]
	}
	return strconv.Itoa(index + 1)
}

func buildLineXTicks(xs []float32, labels []string, maxLabels int) []chart.Tick {
	n := len(xs)
	if n == 0 {
		return nil
	}
	step := 1
	if maxLabels > 0 && n > maxLabels {
		step = int(math.Ceil(float64(n-1) / float64(max(1, maxLabels-1))))
	}
	ticks := make([]chart.Tick, 0, min(n, 2+(n-2)/step))
	for i := 0; i < n; i += step {
		ticks = append(ticks, chart.Tick{At: xs[i], Value: float64(i), Label: lineXLabel(labels, i)})
	}
	if last := n - 1; last > 0 && ticks[len(ticks)-1].Value != float64(last) {
		ticks = append(ticks, chart.Tick{At: xs[last], Value: float64(last), Label: lineXLabel(labels, last)})
	}
	return ticks
}

func buildLineSeriesPaths(
	s *sparklineScratch,
	r ui.Rect,
	xs []float32,
	values []float64,
	yLo, yHi float64,
	baseFrac, riseScale float32,
	curve CurveStyle,
	pixelW int,
) bool {
	n := min(len(xs), len(values))
	if n == 0 || !(yHi > yLo) {
		return false
	}
	invSpan := 1.0 / (yHi - yLo)
	bottomY := r.Y + r.H

	if pixelW > 0 && n > pixelW*2 {
		s.rawY = ensureF32Cap(s.rawY, n)
		for i := range n {
			v := values[i]
			if !isFiniteF64(v) {
				s.rawY[i] = float32(math.NaN())
			} else {
				yf := clampF32(float32((v-yLo)*invSpan), 0, 1)
				s.rawY[i] = clampF32(baseFrac+(yf-baseFrac)*riseScale, 0, 1)
			}
		}
		s.dsX, s.dsY = chart.MinMax(s.dsX, s.dsY, xs[:n], s.rawY, pixelW)
		ok, _, _ := buildSparklineRunsFromFractions(s, r, bottomY, s.dsX, s.dsY, curve)
		return ok
	}

	hasLine := false
	runStart := -1
	for i := 0; i <= n; i++ {
		finite := i < n && isFiniteF64(values[i])
		if finite {
			if runStart < 0 {
				runStart = i
				s.pts = s.pts[:0]
			}
			yf := clampF32(float32((values[i]-yLo)*invSpan), 0, 1)
			yf = clampF32(baseFrac+(yf-baseFrac)*riseScale, 0, 1)
			s.pts = append(s.pts, [2]float32{chart.X(r, xs[i]), chart.Y(r, yf)})
			continue
		}
		if runStart >= 0 {
			if appendSparklineRun(s, bottomY, curve) {
				hasLine = true
			}
			runStart = -1
		}
	}
	return hasLine
}
