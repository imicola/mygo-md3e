package charts

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	md3e "github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// DonutSlice describes one wedge/sector of DonutChart.
type DonutSlice struct {
	// Label names this slice in the legend, tooltip, center readout, and screen reader.
	Label string
	// Value is the slice magnitude. Non-finite or <= 0 values are clamped to 0.
	Value float64
	// Color overrides the slice fill; assigned from SeriesColor(th, index) when A == 0.
	Color ui.Color
	// ValueLabel optionally overrides the formatted numeric label.
	ValueLabel string
}

// DonutChartOptions configures DonutChart.
type DonutChartOptions struct {
	// Size is the outer ring diameter in DIP (default 184).
	Size float32
	// Thickness overrides the ring thickness in DIP when > 0.
	Thickness float32
	// InnerRatio sets innerRadius/outerRadius in 0..0.88 when Thickness <= 0
	// (default 0.62; pass negative or use a small value for a full pie).
	InnerRatio float32
	// PieMode renders a solid pie chart (inner radius = 0) instead of a donut.
	PieMode bool
	// PadAngle is the radian separation between adjacent slices (default 0.035).
	PadAngle float64
	// CenterTitle is the primary readout shown inside the donut hole at rest.
	// When empty, displays the formatted sum of all slices.
	CenterTitle string
	// CenterSubtitle is the secondary caption shown below CenterTitle at rest
	// (default "Total" when CenterTitle is empty).
	CenterSubtitle string
	// ShowLegend displays an interactive legend beside (or below) the ring
	// (nil defaults to true).
	ShowLegend *bool
	// LegendBelow places the legend underneath the ring instead of beside it.
	LegendBelow bool
	// ShowTooltip displays a floating MD3E RichTooltip anchored near the active
	// slice midpoint (nil defaults to true).
	ShowTooltip *bool
	// Animate drives a clockwise MD3E DefaultSpatial entrance sweep and
	// FastSpatial active-slice pop-out, snapping immediately under ReduceMotion
	// (nil defaults to true).
	Animate *bool
	// Selected optionally binds the active slice index (-1 = none).
	Selected *int
	// OnSelect runs whenever the active slice changes via hover, click, legend, or keyboard.
	OnSelect func(index int)
	// ValueFormatter customizes slice value formatting when ValueLabel is empty.
	ValueFormatter func(v, pct float64) string
	// Label names the chart for assistive technology (default "Donut chart").
	Label string
	// Description provides additional accessible context.
	Description string
}

type donutLocalKey struct{}
type donutSlicePopKey int

type donutState struct {
	selected   int
	hovering   bool
	lastX      float32
	lastY      float32
	lastAnn    int
	sectorPath ui.Path
}

// DonutChart builds an MD3E Expressive donut/pie chart over
// github.com/egoist/mygo/ui/chart with HCT-harmonized slices, clockwise spring
// sweep entrance, active-slice radial spring pop-out with soft outer glow,
// dynamic center metric readout, keyboard & RTL navigation, interactive legend,
// and floating RichTooltip card.
func DonutChart(c *ui.Context, slices []DonutSlice, opts ...DonutChartOptions) ui.Element {
	if len(opts) > 1 {
		panic("charts: DonutChart takes at most one DonutChartOptions")
	}
	var o DonutChartOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	animate := boolDefault(o.Animate, true)
	showLegend := boolDefault(o.ShowLegend, true)
	showTooltip := boolDefault(o.ShowTooltip, true)
	rtl := c.Direction() == ui.RightToLeft

	size := sanitizePositiveF32(o.Size, 184)
	padAngle := 0.035
	if isFiniteF64(o.PadAngle) && o.PadAngle >= 0 {
		if o.PadAngle > 0 {
			padAngle = min(o.PadAngle, 0.25)
		}
	}

	// Reserve outer margin for the active slice's +6 DIP spring pop-out & aura.
	popExpand := float32(6)
	baseOuter := max(12, size*0.5-popExpand-2)
	var inner float32
	if !o.PieMode {
		if isPositiveFiniteF32(o.Thickness) {
			inner = max(0, baseOuter-o.Thickness)
		} else {
			ratio := float32(0.62)
			if isPositiveFiniteF32(o.InnerRatio) {
				ratio = clampF32(o.InnerRatio, 0.15, 0.88)
			}
			inner = baseOuter * ratio
		}
	}

	n := len(slices)
	vals := make([]float64, n)
	colors := make([]ui.Color, n)
	total := 0.0
	positiveCount := 0
	for i, sl := range slices {
		v := sl.Value
		if !isFiniteF64(v) || v < 0 {
			v = 0
		}
		vals[i] = v
		total += v
		if v > 0 {
			positiveCount++
		}
		col := sl.Color
		if col.A == 0 {
			col = SeriesColor(th, i)
		}
		colors[i] = col
	}

	effPad := padAngle
	if positiveCount <= 1 {
		effPad = 0
	}
	pieSlices := chart.Pie(vals, effPad)

	label := o.Label
	if label == "" {
		label = "Donut chart"
	}

	var root ui.Element
	if o.LegendBelow || !showLegend {
		root = ui.Column(c).AlignItems(ui.Center).Gap(14)
	} else {
		root = ui.Row(c).AlignItems(ui.Center).Wrap().Gap(20)
	}

	st := ui.Local(root, donutLocalKey{}, func() donutState {
		return donutState{selected: -1, lastAnn: -1}
	})
	selPtr := o.Selected
	if selPtr == nil {
		selPtr = &st.selected
	}
	if *selPtr >= n {
		*selPtr = -1
	}

	enter := animateEntrance(root, th, animate, reduce)
	sweep := clampF32(enter, 0, 1)

	formatSliceVal := func(i int) (valStr, pctStr string) {
		pct := 0.0
		if total > 0 {
			pct = (vals[i] / total) * 100
		}
		pctStr = defaultFormatPercent(pct)
		if slices[i].ValueLabel != "" {
			return slices[i].ValueLabel, pctStr
		}
		if o.ValueFormatter != nil {
			return o.ValueFormatter(vals[i], pct), pctStr
		}
		return defaultFormatNumber(vals[i]), pctStr
	}

	emptyTrackCol := cs.SurfaceContainerHighest
	if emptyTrackCol.A == 0 {
		emptyTrackCol = cs.OutlineVariant
	}
	bgTrackCol := emptyTrackCol.Alpha(0.45)
	if th.Dark {
		bgTrackCol = emptyTrackCol.Alpha(0.38)
	}
	dark := th.Dark

	root.Children(func() {
		ring := ui.Box(c.Key("ring")).Size(size, size).Focusable().
			Role(ui.RoleGroup).Label(label)
		if o.Description != "" {
			ring.Description(o.Description)
		}

		// Keyboard navigation (RTL-aware Left/Right + Home/End/Escape).
		oldSel := *selPtr
		announce := false
		if n > 0 && total > 0 {
			nextKey, prevKey := ui.KeyRight, ui.KeyLeft
			if rtl {
				nextKey, prevKey = ui.KeyLeft, ui.KeyRight
			}
			moveTo := func(idx int) {
				*selPtr = max(0, min(n-1, idx))
				announce = true
			}
			switch {
			case ring.Shortcut(0, nextKey):
				if *selPtr < 0 {
					moveTo(0)
				} else {
					moveTo((*selPtr + 1) % n)
				}
			case ring.Shortcut(0, prevKey):
				if *selPtr < 0 {
					moveTo(n - 1)
				} else {
					moveTo((*selPtr - 1 + n) % n)
				}
			case ring.Shortcut(0, ui.KeyHome):
				moveTo(0)
			case ring.Shortcut(0, ui.KeyEnd):
				moveTo(n - 1)
			case *selPtr >= 0 && ring.Shortcut(0, ui.KeyEscape):
				*selPtr = -1
			}
		}

		// Pointer hover / click slice hit testing via chart.SliceAt.
		if px, py, over := ring.PointerPosition(); over {
			if !announce && (!st.hovering || px != st.lastX || py != st.lastY) && total > 0 {
				cx, cy := size*0.5, size*0.5
				if hit, ok := chart.SliceAt(pieSlices, px-cx, py-cy, max(0, inner-2), baseOuter+popExpand); ok {
					*selPtr = hit
				} else if st.hovering {
					*selPtr = -1
				}
			}
			st.hovering, st.lastX, st.lastY = true, px, py
		} else if st.hovering {
			st.hovering = false
			*selPtr = -1
		}

		if *selPtr != oldSel && o.OnSelect != nil {
			o.OnSelect(*selPtr)
		}

		// Update accessible Value and announce keyboard changes.
		if idx := *selPtr; idx >= 0 && idx < n {
			vs, ps := formatSliceVal(idx)
			name := slices[idx].Label
			if name == "" {
				name = fmt.Sprintf("Slice %d", idx+1)
			}
			desc := fmt.Sprintf("%s: %s (%s)", name, vs, ps)
			ring.Value(desc)
			if announce && idx != st.lastAnn {
				c.Announce(desc)
			}
			st.lastAnn = idx
		} else {
			if total > 0 {
				ring.Value(fmt.Sprintf("Total: %s", defaultFormatNumber(total)))
			} else {
				ring.Value("Empty")
			}
			st.lastAnn = -1
		}

		// Per-slice spring radial pop-out offsets (FastSpatial, normalized 0..1).
		popAnim := motion.Resolve(th.Motion, motion.FastSpatial, reduce)
		pops := make([]float32, n)
		for i := range n {
			target := float32(0)
			if *selPtr == i {
				target = 1
			}
			pops[i] = max(0, animateF32(ring, donutSlicePopKey(i), target, popAnim, reduce || !animate)) * popExpand
		}

		// The slices are owned by this build and remain read-only while
		// a departing element draws its captured frame.
		drawActive := *selPtr
		drawTotal := total
		drawSweep := float64(sweep)
		pieMode := o.PieMode
		sectorPath := &st.sectorPath
		sliceCorner := float32(0)
		if !pieMode && positiveCount > 1 && baseOuter-inner > 8 {
			sliceCorner = min(float32(4), (baseOuter-inner)*0.24)
		}

		ring.Draw(func(p *ui.Painter, r ui.Rect) {
			if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
				return
			}
			if !rectIntersects(r, p.Visible()) {
				return
			}
			cx, cy := r.X+r.W*0.5, r.Y+r.H*0.5
			full := chart.Slice{Start: 0, End: 2 * math.Pi}
			if drawTotal <= 0 || len(pieSlices) == 0 {
				p.FillPath(buildRoundedSectorPath(sectorPath, cx, cy, inner, baseOuter, full, 0), emptyTrackCol)
				return
			}

			// Subtle full-ring background bed behind donut slices (replaces gap-bridging inner stroke).
			if !pieMode && inner > 4 {
				p.FillPath(buildRoundedSectorPath(sectorPath, cx, cy, inner, baseOuter, full, 0), bgTrackCol)
			}

			for i, sl := range pieSlices {
				if vals[i] <= 0 || sl.End <= sl.Start {
					continue
				}
				swept := chart.Slice{
					Start: sl.Start * drawSweep,
					End:   sl.End * drawSweep,
					Pad:   sl.Pad * drawSweep,
				}
				if swept.End <= swept.Start {
					continue
				}
				col := colors[i]
				outer := baseOuter + pops[i]
				popFrac := clampF32(pops[i]/popExpand, 0, 1)

				if drawActive >= 0 && drawActive != i {
					col = col.Alpha(0.56)
				}

				// Smooth continuous 2-layer colored aura driven by spring popFrac.
				if popFrac > 0.01 {
					outerA, innerA := float32(0.10), float32(0.16)
					if dark {
						outerA, innerA = 0.14, 0.22
					}
					p.FillPath(buildRoundedSectorPath(sectorPath, cx, cy, max(0, inner-1), outer+4.5*popFrac, swept, sliceCorner+1), col.Alpha(outerA*popFrac))
					p.FillPath(buildRoundedSectorPath(sectorPath, cx, cy, max(0, inner-0.5), outer+2.2*popFrac, swept, sliceCorner+0.5), col.Alpha(innerA*popFrac))
				}

				p.FillPath(buildRoundedSectorPath(sectorPath, cx, cy, inner, outer, swept, sliceCorner), col)
			}
		})

		// Dynamic center metric readout inside the donut hole + floating RichTooltip.
		ring.Children(func() {
			if !o.PieMode && inner >= 26 {
				cTitle := o.CenterTitle
				cSub := o.CenterSubtitle
				var activeSwatch ui.Color
				if idx := *selPtr; idx >= 0 && idx < n && total > 0 {
					vs, ps := formatSliceVal(idx)
					cTitle = vs
					activeSwatch = colors[idx]
					name := slices[idx].Label
					if name != "" {
						cSub = fmt.Sprintf("%s · %s", name, ps)
					} else {
						cSub = ps
					}
				} else if cTitle == "" {
					if total > 0 {
						cTitle = defaultFormatNumber(total)
					} else {
						cTitle = "0"
					}
					if cSub == "" {
						cSub = "Total"
					}
				}
				centerW := max(40, inner*1.72)
				ui.Column(c.Key("center")).Absolute().Fill().
					Justify(ui.Center).AlignItems(ui.Center).PassThrough().
					Children(func() {
						ui.Column(c).MaxWidth(centerW).AlignItems(ui.Center).Gap(2).Children(func() {
							md3e.TextWith(c, cTitle, md3e.TextOptions{
								Role:        tokens.TitleMedium,
								Emphasized:  true,
								TabularNums: true,
								MaxLines:    1,
								Color:       cs.OnSurface,
							})
							if cSub != "" {
								ui.Row(c).AlignItems(ui.Center).Gap(5).MaxWidth(centerW).Children(func() {
									if activeSwatch.A > 0 {
										ui.Box(c).Size(7, 7).Radius(3.5).Background(activeSwatch).Shrink(0)
									}
									md3e.TextWith(c, cSub, md3e.TextOptions{
										Role:     tokens.LabelSmall,
										MaxLines: 1,
										Color:    cs.OnSurfaceVariant,
									})
								})
							}
						})
					})
			}

			// Floating RichTooltip anchored near active slice midpoint.
			if showTooltip && total > 0 {
				if idx := *selPtr; idx >= 0 && idx < n && vals[idx] > 0 {
					sl := pieSlices[idx]
					ax, ay := chart.Polar(size*0.5, size*0.5, baseOuter+popExpand+4, sl.Mid())
					ax = clampF32(ax, 12, size-12)
					ay = clampF32(ay, 12, size-12)
					hAlign := ui.Start
					if ax > size*0.52 {
						hAlign = ui.End
					}
					vAlign := ui.End
					if ay < size*0.42 {
						vAlign = ui.Start
					}
					vs, ps := formatSliceVal(idx)
					name := slices[idx].Label
					if name == "" {
						name = fmt.Sprintf("Slice %d", idx+1)
					}
					ui.Row(c.Key("tip")).Absolute().Left(ax).Top(ay).Size(0, 0).
						AlignItems(vAlign).PassThrough().Children(func() {
						ui.Column(c).Width(0).AlignItems(hAlign).Children(func() {
							renderChartTooltipCard(c, th, name, []TooltipRow{
								{Name: ps, Value: vs, Color: colors[idx]},
							}, "", "")
						})
					})
				}
			}
		})

		if showLegend && n > 0 {
			items := make([]chartLegendItem, n)
			for i, sl := range slices {
				name := sl.Label
				if name == "" {
					name = fmt.Sprintf("Slice %d", i+1)
				}
				vs, ps := formatSliceVal(i)
				items[i] = chartLegendItem{
					Name:  name,
					Value: fmt.Sprintf("%s (%s)", vs, ps),
					Color: colors[i],
				}
			}
			renderChartLegend(c, th, items, *selPtr, func(idx int) {
				if *selPtr == idx {
					*selPtr = -1
				} else {
					*selPtr = idx
				}
				if o.OnSelect != nil {
					o.OnSelect(*selPtr)
				}
			})
		}
	})

	return root
}

// buildRoundedSectorPath writes a pie or donut sector into dst with optional
// rounded arc corners, reusing dst's backing storage with zero allocations.
func buildRoundedSectorPath(dst *ui.Path, cx, cy, inner, outer float32, s chart.Slice, cornerR float32) *ui.Path {
	dst.Reset()
	if outer <= 0 || s.End <= s.Start {
		return dst
	}
	inner = max(0, min(inner, outer))
	gap := s.Pad * math.Sqrt(float64(inner*inner+outer*outer))
	angles := func(r float32) (a0, a1 float64) {
		a0, a1 = s.Start, s.End
		if gap > 0 && r > 0 {
			d := math.Asin(min(1, gap*0.5/float64(r)))
			if a1-a0 > 2*d {
				return a0 + d, a1 - d
			}
			m := (a0 + a1) * 0.5
			return m, m
		}
		return a0, a1
	}

	a0, a1 := angles(outer)
	if a1 <= a0 {
		return dst
	}
	// Full ring / full circle or unrounded pie slice.
	isFull := (s.End - s.Start) >= 2*math.Pi-1e-4 && gap <= 1e-5
	if isFull || inner <= 0 || cornerR <= 0.25 {
		x, y := chart.Polar(cx, cy, outer, a0)
		dst.MoveTo(x, y)
		appendSectorArc(dst, cx, cy, outer, a0, a1)
		if inner > 0 {
			b0, b1 := angles(inner)
			if isFull {
				dst.Close()
				ix, iy := chart.Polar(cx, cy, inner, b1)
				dst.MoveTo(ix, iy)
				appendSectorArc(dst, cx, cy, inner, b1, b0)
			} else {
				ix, iy := chart.Polar(cx, cy, inner, b1)
				dst.LineTo(ix, iy)
				appendSectorArc(dst, cx, cy, inner, b1, b0)
			}
		} else if !isFull {
			dst.LineTo(cx, cy)
		}
		dst.Close()
		return dst
	}

	b0, b1 := angles(inner)
	if b1 <= b0 {
		b0, b1 = a0, a1
	}

	thick := outer - inner
	o0x, o0y := chart.Polar(cx, cy, outer, a0)
	o1x, o1y := chart.Polar(cx, cy, outer, a1)
	i0x, i0y := chart.Polar(cx, cy, inner, b0)
	i1x, i1y := chart.Polar(cx, cy, inner, b1)

	edgeStartLen := float32(math.Hypot(float64(o0x-i0x), float64(o0y-i0y)))
	edgeEndLen := float32(math.Hypot(float64(o1x-i1x), float64(o1y-i1y)))
	minEdge := min(thick, min(edgeStartLen, edgeEndLen))
	rOut := min(cornerR, minEdge*0.42, outer*float32(a1-a0)*0.36)
	rIn := min(cornerR, minEdge*0.42, inner*float32(b1-b0)*0.36)
	daOut := min(float64(rOut)/float64(outer), (a1-a0)*0.45)
	daIn := min(float64(rIn)/float64(inner), (b1-b0)*0.45)

	const k = float32(0.55228475)
	lerpEdge := func(ax, ay, bx, by, dist, total float32) (float32, float32) {
		if total <= 1e-4 {
			return ax, ay
		}
		t := clampF32(dist/total, 0, 0.45)
		return ax + (bx-ax)*t, ay + (by-ay)*t
	}
	corner := func(ax, ay, vx, vy, bx, by float32) {
		dst.CubeTo(ax+k*(vx-ax), ay+k*(vy-ay), bx+k*(vx-bx), by+k*(vy-by), bx, by)
	}

	// Start edge -> outer-start corner -> outer arc -> outer-end corner -> end edge.
	sOutX, sOutY := lerpEdge(o0x, o0y, i0x, i0y, rOut, edgeStartLen)
	oa0x, oa0y := chart.Polar(cx, cy, outer, a0+daOut)
	oa1x, oa1y := chart.Polar(cx, cy, outer, a1-daOut)
	eOutX, eOutY := lerpEdge(o1x, o1y, i1x, i1y, rOut, edgeEndLen)

	eInX, eInY := lerpEdge(i1x, i1y, o1x, o1y, rIn, edgeEndLen)
	ib1x, ib1y := chart.Polar(cx, cy, inner, b1-daIn)
	ib0x, ib0y := chart.Polar(cx, cy, inner, b0+daIn)
	sInX, sInY := lerpEdge(i0x, i0y, o0x, o0y, rIn, edgeStartLen)

	dst.MoveTo(sOutX, sOutY)
	corner(sOutX, sOutY, o0x, o0y, oa0x, oa0y)
	appendSectorArc(dst, cx, cy, outer, a0+daOut, a1-daOut)
	corner(oa1x, oa1y, o1x, o1y, eOutX, eOutY)
	dst.LineTo(eInX, eInY)
	corner(eInX, eInY, i1x, i1y, ib1x, ib1y)
	appendSectorArc(dst, cx, cy, inner, b1-daIn, b0+daIn)
	corner(ib0x, ib0y, i0x, i0y, sInX, sInY)
	dst.Close()
	return dst
}

func appendSectorArc(dst *ui.Path, cx, cy, r float32, a0, a1 float64) {
	n := int(math.Ceil(math.Abs(a1-a0) / (math.Pi * 0.5)))
	if n == 0 {
		return
	}
	step := (a1 - a0) / float64(n)
	k := float32(4.0 / 3.0 * math.Tan(step*0.25))
	for i := range n {
		b0, b1 := a0+float64(i)*step, a0+float64(i+1)*step
		x0, y0 := chart.Polar(cx, cy, r, b0)
		x1, y1 := chart.Polar(cx, cy, r, b1)
		t0x, t0y := float32(math.Cos(b0)), float32(math.Sin(b0))
		t1x, t1y := float32(math.Cos(b1)), float32(math.Sin(b1))
		dst.CubeTo(x0+k*r*t0x, y0+k*r*t0y, x1-k*r*t1x, y1-k*r*t1y, x1, y1)
	}
}
