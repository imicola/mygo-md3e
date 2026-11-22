package charts

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/tokens"
)

// BarSeries defines one data series plotted by BarChart.
type BarSeries struct {
	// Name identifies this series in the legend, tooltip, and screen-reader output.
	Name string
	// Values are numeric magnitudes aligned with BarChartOptions.Categories.
	// NaN/non-finite entries are safely treated as 0.
	Values []float64
	// Color overrides the bar fill; assigned from SeriesColor(th, index) when A == 0.
	Color ui.Color
}

// BarLayout selects grouped side-by-side bars or stacked bars.
type BarLayout int

const (
	// BarGrouped places series bars side-by-side inside each category band (default).
	BarGrouped BarLayout = iota
	// BarStacked piles series values vertically via chart.Stack(StackStandard).
	BarStacked
)

// BarChartOptions configures BarChart.
type BarChartOptions struct {
	// Categories are the X-axis band labels. Auto-generated ("1", "2", ...) when empty.
	Categories []string
	// Layout selects BarGrouped (default) or BarStacked.
	Layout BarLayout
	// Height sets the plot height in DIP (default 220).
	Height float32
	// YMin and YMax optionally fix the vertical domain when HasYDomain is true
	// or when both are finite and YMax > YMin.
	YMin, YMax float64
	HasYDomain bool
	// YTicks is the target Y-axis tick count (default 4).
	YTicks int
	// BarRadius is the outer cap corner radius in DIP (default 6).
	BarRadius float32
	// GroupPadding is the inner spacing fraction between category bands (default 0.28).
	GroupPadding float32
	// BarGap is the gap fraction between grouped bars inside a category (default 0.14).
	BarGap float32
	// ShowGrid draws horizontal MD3E grid lines (nil defaults to true).
	ShowGrid *bool
	// ShowLegend renders an interactive legend (nil defaults to true when len(series) > 1).
	ShowLegend *bool
	// ShowTooltip enables the floating MD3E RichTooltip readout on hover/keyboard
	// selection (nil defaults to true).
	ShowTooltip *bool
	// ShowValues optionally draws compact value labels above bars.
	ShowValues bool
	// Animate springs bars upward from the 0 baseline via MD3E DefaultSpatial,
	// snapping immediately under ReduceMotion (nil defaults to true).
	Animate *bool
	// Selected optionally binds the active category index (-1 = none).
	Selected *int
	// OnSelect runs whenever the selected category changes via pointer, click, or keyboard.
	OnSelect func(index int)
	// ValueFormatter formats Y-axis ticks, data labels, and tooltip values.
	ValueFormatter func(v float64) string
	// Label names the chart for assistive technology (default "Bar chart").
	Label string
	// Description provides additional accessible context.
	Description string
}

type barChartLocalKey struct{}

type barChartState struct {
	selected      int
	focusedSeries int
}

// BarChart builds an MD3E Expressive grouped or stacked bar chart over
// github.com/egoist/mygo/ui/chart with HCT-harmonized vertical tonal gradients,
// rounded outer caps, spring entrance growth, active category column spotlight,
// interactive legend focus, and hover/keyboard RichTooltip readouts.
func BarChart(c *ui.Context, series []BarSeries, opts ...BarChartOptions) ui.Element {
	if len(opts) > 1 {
		panic("charts: BarChart takes at most one BarChartOptions")
	}
	var o BarChartOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	animate := boolDefault(o.Animate, true)
	showGrid := boolDefault(o.ShowGrid, true)
	showLegend := boolDefault(o.ShowLegend, len(series) > 1)
	showTooltip := boolDefault(o.ShowTooltip, true)
	height := sanitizePositiveF32(o.Height, 220)
	yTickCount := o.YTicks
	if yTickCount <= 0 {
		yTickCount = 4
	}
	barRadius := sanitizeNonNegativeF32(o.BarRadius, 6)
	groupPad := clampF32(sanitizePositiveF32(o.GroupPadding, 0.28), 0.05, 0.75)
	barGap := clampF32(sanitizeNonNegativeF32(o.BarGap, 0.14), 0, 0.5)
	fmtVal := o.ValueFormatter
	if fmtVal == nil {
		fmtVal = defaultFormatNumber
	}

	numCats := len(o.Categories)
	colors := make([]ui.Color, len(series))
	for i, s := range series {
		if len(s.Values) > numCats {
			numCats = len(s.Values)
		}
		col := s.Color
		if col.A == 0 {
			col = SeriesColor(th, i)
		}
		colors[i] = col
	}
	cats := make([]string, numCats)
	for i := range numCats {
		if i < len(o.Categories) && o.Categories[i] != "" {
			cats[i] = o.Categories[i]
		} else {
			cats[i] = fmt.Sprintf("%d", i+1)
		}
	}

	// Sanitize matrix for stacking & domain computation.
	matrix := make([][]float64, len(series))
	for si, s := range series {
		row := make([]float64, numCats)
		for ci := range numCats {
			if ci < len(s.Values) && isFiniteF64(s.Values[ci]) {
				row[ci] = s.Values[ci]
			}
		}
		matrix[si] = row
	}

	var stacked [][]chart.Segment
	if o.Layout == BarStacked && len(series) > 0 && numCats > 0 {
		stacked = chart.Stack(matrix, chart.StackStandard)
	}

	yScale := resolveBarYScale(matrix, stacked, o.Layout, o.YMin, o.YMax, o.HasYDomain, yTickCount, fmtVal)
	yTicks := yScale.Ticks(yTickCount)
	xBand := chart.NewBand(cats...).Padding(groupPad, 0.12)
	bandCenters := make([]float32, numCats)
	for i := range numCats {
		bandCenters[i] = xBand.At(i)
	}

	label := o.Label
	if label == "" {
		label = "Bar chart"
	}

	root := ui.Column(c).FillWidth().MinWidth(0).Gap(10)
	st := ui.Local(root, barChartLocalKey{}, func() barChartState {
		return barChartState{selected: -1, focusedSeries: -1}
	})
	if st.focusedSeries >= len(series) {
		st.focusedSeries = -1
	}

	selPtr := o.Selected
	if selPtr == nil {
		selPtr = &st.selected
	}
	if *selPtr >= numCats {
		*selPtr = -1
	}

	enter := animateEntrance(root, th, animate, reduce)
	growScale := clampF32(enter, 0, 1.08)

	gridCol := gridLineColor(th)
	spotlightCol := cs.OnSurface.Alpha(0.055)
	if th.Dark {
		spotlightCol = cs.OnSurface.Alpha(0.075)
	}
	zeroAt := clampF32(yScale.At(0), 0, 1)

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
			Bottom: xBand.Ticks(),
			Label:  md3eAxisLabel(c, th),
		}

		plot := chart.Plot(c.Key("plot"), axes, func(area ui.Element) {
			area.Label(label)
			if o.Description != "" {
				area.Description(o.Description)
			}

			describe := func(ci int) string {
				if ci < 0 || ci >= numCats {
					return ""
				}
				var parts []string
				for si, s := range series {
					name := s.Name
					if name == "" {
						name = fmt.Sprintf("Series %d", si+1)
					}
					parts = append(parts, fmt.Sprintf("%s %s", name, fmtVal(matrix[si][ci])))
				}
				if len(parts) == 0 {
					return cats[ci]
				}
				return cats[ci] + ": " + strings.Join(parts, ", ")
			}

			if numCats > 0 {
				if chart.Select(c, area, selPtr, bandCenters, describe) && o.OnSelect != nil {
					o.OnSelect(*selPtr)
				}
			}

			yTicksSnap := append([]chart.Tick(nil), yTicks...)
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
					return
				}
				if vis := p.Visible(); vis.W > 0 && vis.H > 0 && !rectIntersects(r, vis) {
					return
				}
				if showGrid {
					for _, tick := range yTicksSnap {
						ty := chart.Y(r, tick.At)
						p.Line(r.X, ty, r.X+r.W, ty, 1, gridCol)
					}
				}
				// Crisp baseline at 0 when inside domain.
				if yScale.Lo <= 0 && yScale.Hi >= 0 {
					zy := chart.Y(r, zeroAt)
					p.Line(r.X, zy, r.X+r.W, zy, 1.2, cs.OutlineVariant)
				}
			})

			activeCat := *selPtr

			// Soft tonal column spotlight behind the active category band.
			if activeCat >= 0 && activeCat < numCats {
				band := xBand.Band(activeCat)
				padSpan := (band.Hi - band.Lo) * 0.14
				spot := chart.Span{Lo: max(0, band.Lo-padSpan), Hi: min(1, band.Hi+padSpan)}
				chart.Rect(c, spot, chart.Span{Lo: 0, Hi: 1}).
					Radius(th.Shapes.Radius(tokens.ShapeSmall)).
					Background(spotlightCol).PassThrough()
			}

			// Render grouped or stacked bars with vertical tonal gradients.
			if numCats > 0 && len(series) > 0 {
				switch o.Layout {
				case BarStacked:
					renderStackedBars(c, th, series, matrix, stacked, colors, xBand, yScale, barRadius, growScale, activeCat, st.focusedSeries, o.ShowValues, fmtVal, func(ci int) {
						if *selPtr != ci {
							*selPtr = ci
							if o.OnSelect != nil {
								o.OnSelect(ci)
							}
						}
					})
				default:
					renderGroupedBars(c, th, series, matrix, colors, xBand, yScale, barRadius, barGap, growScale, activeCat, st.focusedSeries, o.ShowValues, fmtVal, func(ci int) {
						if *selPtr != ci {
							*selPtr = ci
							if o.OnSelect != nil {
								o.OnSelect(ci)
							}
						}
					})
				}
			}

			// Floating MD3E RichTooltip card for active category.
			if showTooltip && activeCat >= 0 && activeCat < numCats && len(series) > 0 {
				var rows []TooltipRow
				total := 0.0
				maxFrac := float32(0.4)
				for si, s := range series {
					v := matrix[si][activeCat]
					total += v
					if st.focusedSeries >= 0 && st.focusedSeries != si {
						continue
					}
					name := s.Name
					if name == "" {
						name = fmt.Sprintf("Series %d", si+1)
					}
					rows = append(rows, TooltipRow{
						Name:  name,
						Value: fmtVal(v),
						Color: colors[si],
					})
					var hiVal float64
					if o.Layout == BarStacked && len(stacked) > si {
						hiVal = stacked[si][activeCat].Hi
					} else {
						hiVal = v
					}
					if f := clampF32(yScale.At(hiVal), 0, 1); f > maxFrac {
						maxFrac = f
					}
				}
				footerLbl, footerVal := "", ""
				if o.Layout == BarStacked && len(series) > 1 && st.focusedSeries < 0 {
					footerLbl = "Total"
					footerVal = fmtVal(total)
				}
				cx := xBand.At(activeCat)
				hAlign := ui.Start
				mL, mR := float32(10), float32(0)
				if cx > 0.55 {
					hAlign = ui.End
					mL, mR = 0, 10
				}
				anchorY := clampF32(maxFrac, 0.25, 0.94)
				vAlign := ui.Start
				if anchorY < 0.45 {
					vAlign = ui.End
				}
				chart.Anchor(c, cx, anchorY, hAlign, vAlign, func() {
					ui.Box(c).Margin(0, mR, 0, mL).PassThrough().Children(func() {
						renderChartTooltipCard(c, th, cats[activeCat], rows, footerLbl, footerVal)
					})
				}).PassThrough()
			}
		}).Height(height).Gap(8).Label(label)
		if o.Description != "" {
			plot.Description(o.Description)
		}
	})

	return root
}

func renderGroupedBars(
	c *ui.Context,
	th *md3e.Theme,
	series []BarSeries,
	matrix [][]float64,
	colors []ui.Color,
	xBand chart.Band,
	yScale chart.Linear,
	barRadius, barGap, growScale float32,
	activeCat, focusedSeries int,
	showValues bool,
	fmtVal func(float64) string,
	onClick func(int),
) {
	numSeries := len(series)
	numCats := len(xBand.Labels)
	for ci := range numCats {
		band := xBand.Band(ci)
		bandW := band.Hi - band.Lo
		subStep := bandW / float32(numSeries)
		gapW := subStep * barGap
		if numSeries == 1 {
			gapW = 0
		}
		subW := max(0, subStep-gapW)

		for si := range numSeries {
			val := matrix[si][ci]
			scaledVal := val * float64(growScale)
			spanY := yScale.Span(0, scaledVal)
			x0 := band.Lo + float32(si)*subStep + gapW*0.5
			spanX := chart.Span{Lo: x0, Hi: x0 + subW}

			col := colors[si]
			dimmed := (activeCat >= 0 && activeCat != ci) || (focusedSeries >= 0 && focusedSeries != si)
			if dimmed {
				col = col.Alpha(0.46)
			}

			grad := barVerticalGradient(col, th.Dark, activeCat == ci && !dimmed)
			bar := chart.Rect(c, spanX, spanY).LinearGradient(grad)
			if val >= 0 {
				bar.Radius(barRadius, barRadius, 0, 0)
			} else {
				bar.Radius(0, 0, barRadius, barRadius)
			}
			if bar.Clicked() && onClick != nil {
				onClick(ci)
			}

			if showValues && growScale > 0.85 && !dimmed && val != 0 {
				vAlign := ui.End
				if val < 0 {
					vAlign = ui.Start
				}
				chart.Anchor(c, spanX.Mid(), clampF32(yScale.At(val), 0, 1), ui.Center, vAlign, func() {
					md3e.TextWith(c, fmtVal(val), md3e.TextOptions{
						Role:        tokens.LabelSmall,
						TabularNums: true,
						Color:       th.Colors.OnSurfaceVariant,
					}).NoWrap().Margin(2, 0)
				}).PassThrough()
			}
		}
	}
}

func renderStackedBars(
	c *ui.Context,
	th *md3e.Theme,
	series []BarSeries,
	matrix [][]float64,
	stacked [][]chart.Segment,
	colors []ui.Color,
	xBand chart.Band,
	yScale chart.Linear,
	barRadius, growScale float32,
	activeCat, focusedSeries int,
	showValues bool,
	fmtVal func(float64) string,
	onClick func(int),
) {
	numSeries := len(series)
	numCats := len(xBand.Labels)
	for ci := range numCats {
		band := xBand.Band(ci)
		// Find outermost positive and negative non-zero series indices for crisp cap rounding.
		topPos, botNeg := -1, -1
		totalPos := 0.0
		for si := range numSeries {
			v := matrix[si][ci]
			if v > 0 {
				topPos = si
				totalPos += v
			} else if v < 0 {
				botNeg = si
			}
		}

		for si := range numSeries {
			v := matrix[si][ci]
			if v == 0 {
				continue
			}
			seg := stacked[si][ci]
			lo := seg.Lo * float64(growScale)
			hi := seg.Hi * float64(growScale)
			spanY := yScale.Span(lo, hi)

			col := colors[si]
			dimmed := (activeCat >= 0 && activeCat != ci) || (focusedSeries >= 0 && focusedSeries != si)
			if dimmed {
				col = col.Alpha(0.46)
			}
			grad := barVerticalGradient(col, th.Dark, activeCat == ci && !dimmed)
			bar := chart.Rect(c, band, spanY).LinearGradient(grad)
			switch {
			case si == topPos:
				bar.Radius(barRadius, barRadius, 1.5, 1.5)
			case si == botNeg:
				bar.Radius(1.5, 1.5, barRadius, barRadius)
			default:
				bar.Radius(1.5)
			}
			if bar.Clicked() && onClick != nil {
				onClick(ci)
			}
		}

		if showValues && growScale > 0.85 && totalPos > 0 {
			chart.Anchor(c, band.Mid(), clampF32(yScale.At(totalPos), 0, 1), ui.Center, ui.End, func() {
				md3e.TextWith(c, fmtVal(totalPos), md3e.TextOptions{
					Role:        tokens.LabelSmall,
					TabularNums: true,
					Color:       th.Colors.OnSurfaceVariant,
				}).NoWrap().Margin(0, 0, 3, 0)
			}).PassThrough()
		}
	}
}

func barVerticalGradient(base ui.Color, dark, active bool) ui.LinearGradient {
	lift := float32(0.14)
	if dark {
		lift = 0.16
	}
	if active {
		lift += 0.10
	}
	topCol := base.Mix(ui.RGB(255, 255, 255), lift)
	botCol := base.Mix(ui.RGB(0, 0, 0), 0.08)
	return ui.LinearGradient{
		From:  topCol,
		To:    botCol,
		Angle: 180,
		Oklab: true,
	}
}

func resolveBarYScale(matrix [][]float64, stacked [][]chart.Segment, layout BarLayout, yMin, yMax float64, hasDomain bool, tickCount int, fmtVal func(float64) string) chart.Linear {
	if (hasDomain || (isFiniteF64(yMin) && isFiniteF64(yMax) && yMax > yMin)) &&
		isFiniteF64(yMin) && isFiniteF64(yMax) && yMax > yMin {
		sc := chart.NewLinear(yMin, yMax)
		sc.Format = fmtVal
		return sc
	}
	lo, hi := 0.0, 0.0
	if layout == BarStacked && len(stacked) > 0 {
		for _, row := range stacked {
			for _, seg := range row {
				if seg.Lo < lo {
					lo = seg.Lo
				}
				if seg.Hi > hi {
					hi = seg.Hi
				}
			}
		}
	} else {
		for _, row := range matrix {
			for _, v := range row {
				if v < lo {
					lo = v
				}
				if v > hi {
					hi = v
				}
			}
		}
	}
	if hi == lo {
		hi = 10
	}
	sc := chart.NewLinear(lo, hi).Nice(tickCount)
	sc.Format = fmtVal
	return sc
}
