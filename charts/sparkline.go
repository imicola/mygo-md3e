package charts

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	md3e "github.com/imicola/mygo-md3e"
)

// CurveStyle selects how AreaSparkline and LineChart interpolate between points.
// The zero value (CurveSmooth) uses Steffen's monotone cubic Hermite spline so
// curves stay silky without overshooting local extrema.
type CurveStyle int

const (
	// CurveSmooth joins points with Steffen's monotone cubic spline (default).
	CurveSmooth CurveStyle = iota
	// CurveLinear joins points with straight line segments.
	CurveLinear
	// CurveStepAfter holds each value until the next x coordinate.
	CurveStepAfter
	// CurveStepBefore transitions to each value at the preceding x coordinate.
	CurveStepBefore
	// CurveStep transitions halfway between consecutive x coordinates.
	CurveStep
)

func (c CurveStyle) toChartCurve() chart.Curve {
	switch c {
	case CurveLinear:
		return chart.CurveLinear
	case CurveStepAfter:
		return chart.CurveStepAfter
	case CurveStepBefore:
		return chart.CurveStepBefore
	case CurveStep:
		return chart.CurveStep
	default:
		return chart.CurveMonotone
	}
}

// AreaSparklineOptions configures AreaSparkline. The zero value renders a
// smooth Primary-colored sparkline with a soft glow stroke and multi-stop
// vertical fade area fill at 48 DIP height.
type AreaSparklineOptions struct {
	// Ring is an optional live RingBuffer data source. When non-nil, it takes
	// precedence over Values and is read in-place without copying.
	Ring *RingBuffer
	// Values is an optional static/caller-managed slice of samples (NaN creates
	// a clean gap). Ignored when Ring is non-nil.
	Values []float64
	// Color overrides the stroke and area fill tint; defaults to Theme.Colors.Primary.
	Color ui.Color
	// FillTopAlpha sets the top opacity of the vertical area fade (0..1).
	// Defaults to 0.34 when <= 0 and !NoFill.
	FillTopAlpha float32
	// NoFill suppresses the vertical gradient area fill (stroke only).
	NoFill bool
	// Glow draws a soft low-opacity under-stroke behind the main curve using
	// the same reused ui.Path for luminous depth. Nil defaults to true.
	Glow *bool
	// StrokeWidth is the primary curve stroke width in DIP; defaults to 2.2.
	StrokeWidth float32
	// Curve selects the interpolation mode (default CurveSmooth).
	Curve CurveStyle
	// Min and Max optionally fix the vertical Y domain when HasDomain is true
	// or when both are finite and Max > Min.
	Min, Max  float64
	HasDomain bool
	// BaselineZero anchors the bottom of an auto-scaled domain at <= 0.
	BaselineZero bool
	// Height fixes the sparkline height in DIP; defaults to 48.
	Height float32
	// BottomRadius clips the bottom-left and bottom-right corners to r DIP so
	// the area gradient sits flush inside a rounded Card or MetricCard.
	BottomRadius float32
	// Radii explicitly specifies [tl, tr, br, bl] corner clip radii in DIP,
	// taking precedence over BottomRadius when any corner is positive.
	Radii [4]float32
	// ShowEndDot highlights the latest finite sample with a ringed luminous dot.
	ShowEndDot bool
	// Label names the sparkline for assistive technology (default "Sparkline").
	Label string
	// Description provides additional summary text for assistive technology.
	Description string
}

type sparklineLocalKey struct{}

// sparklineScratch holds reusable geometry buffers attached via ui.Local so
// repeated live ticks draw with 0 heap allocations once warmed.
type sparklineScratch struct {
	linePath ui.Path
	areaPath ui.Path
	clipPath ui.Path
	pts      [][2]float32
	slopes   []float32
	rawX     []float32
	rawY     []float32
	dsX      []float32
	dsY      []float32
	stops    [3]ui.GradientStop
}

// AreaSparkline builds a compact, high-frame-rate MD3E sparkline with a smooth
// curve, multi-stop vertical gradient area fade, subtle curve under-glow, and
// optional rounded-bottom clipping for seamless embedding inside cards.
//
// Pass a live *RingBuffer in opts.Ring to reuse geometry buffers and ui.Path
// storage across ticks without per-frame heap allocations. AreaSparkline never
// schedules idle animation frames and is ghost-draw safe during exit transitions.
func AreaSparkline(c *ui.Context, opts ...AreaSparklineOptions) ui.Element {
	if len(opts) > 1 {
		panic("charts: AreaSparkline takes at most one AreaSparklineOptions")
	}
	var o AreaSparklineOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	th := md3e.CurrentTheme(c)
	cs := th.Colors

	col := o.Color
	if col.A == 0 {
		col = SeriesColor(th, 0)
	}
	strokeW := sanitizePositiveF32(o.StrokeWidth, 2.2)
	height := sanitizePositiveF32(o.Height, 48)
	fillAlpha := o.FillTopAlpha
	if !isPositiveFiniteF32(fillAlpha) {
		fillAlpha = 0.34
	}
	fillAlpha = clampF32(fillAlpha, 0, 1)
	glow := boolDefault(o.Glow, true)
	noFill := o.NoFill
	curve := o.Curve
	showEndDot := o.ShowEndDot
	ring := o.Ring
	values := o.Values
	surfaceBg := md3e.SurfaceColorForElevation(th, 1)
	if surfaceBg.A == 0 {
		surfaceBg = cs.Surface
	}
	dark := th.Dark

	radii := o.Radii
	hasClip := radii[0] > 0 || radii[1] > 0 || radii[2] > 0 || radii[3] > 0
	if !hasClip && isPositiveFiniteF32(o.BottomRadius) {
		radii = [4]float32{0, 0, o.BottomRadius, o.BottomRadius}
		hasClip = true
	}

	// Resolve Y domain safely.
	domainLo, domainHi, hasDomain := resolveSparklineDomain(ring, values, o.Min, o.Max, o.HasDomain, o.BaselineZero)

	label := o.Label
	if label == "" {
		label = "Sparkline"
	}

	el := ui.Box(c).FillWidth().Height(height).Role(ui.RoleImage).Label(label)
	if o.Description != "" {
		el.Description(o.Description)
	}
	scratch := ui.Local(el, sparklineLocalKey{}, func() sparklineScratch { return sparklineScratch{} })

	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) || !hasDomain {
			return
		}
		if !rectIntersects(r, p.Visible()) {
			return
		}

		// Reserve slight vertical and trailing inset so stroke caps & endpoint aura never clip.
		topPad := strokeW*0.9 + 1.5
		endDotPad := max(strokeW*2.4+2.5, float32(8))
		if showEndDot {
			topPad = max(topPad, endDotPad)
		}
		if topPad*2 >= r.H {
			topPad = r.H * 0.14
		}
		plotRect := ui.Rect{
			X: r.X,
			Y: r.Y + topPad,
			W: r.W,
			H: max(1, r.H-topPad),
		}
		if showEndDot && plotRect.W > endDotPad*2 {
			plotRect.W -= endDotPad
		}

		pixelW := int(math.Ceil(float64(plotRect.W * max(1, p.Scale()))))
		hasLine, lastPt, hasLast := buildSparklinePaths(scratch, plotRect, r.Y+r.H, ring, values, domainLo, domainHi, curve, pixelW)
		if !hasLine {
			return
		}

		paint := func() {
			if !noFill {
				midAlpha := fillAlpha * 0.34
				if dark {
					midAlpha = fillAlpha * 0.40
				}
				scratch.stops[0] = ui.GradientStop{Color: col.Alpha(fillAlpha), At: 0}
				scratch.stops[1] = ui.GradientStop{Color: col.Alpha(midAlpha), At: 0.56}
				scratch.stops[2] = ui.GradientStop{Color: col.Alpha(0), At: 1}
				p.FillPathGradient(&scratch.areaPath, ui.LinearGradient{
					Angle: 180,
					Stops: scratch.stops[:],
				})
			}
			if glow {
				glowAlpha := float32(0.18)
				if dark {
					glowAlpha = 0.26
				}
				p.StrokePath(&scratch.linePath, strokeW*2.6, col.Alpha(glowAlpha))
			}
			p.StrokePath(&scratch.linePath, strokeW, col)
			if showEndDot && hasLast {
				// Soft translucent aura + surface collar + crisp core dot.
				auraR := strokeW*2.4 + 2.5
				p.Fill(ui.Rect{X: lastPt[0] - auraR, Y: lastPt[1] - auraR, W: auraR * 2, H: auraR * 2}, col.Alpha(0.22), auraR)
				collarR := strokeW + 2.0
				p.Fill(ui.Rect{X: lastPt[0] - collarR, Y: lastPt[1] - collarR, W: collarR * 2, H: collarR * 2}, surfaceBg, collarR)
				coreR := strokeW + 0.6
				p.Fill(ui.Rect{X: lastPt[0] - coreR, Y: lastPt[1] - coreR, W: coreR * 2, H: coreR * 2}, col, coreR)
			}
		}

		if hasClip {
			buildRoundedRectPath(&scratch.clipPath, r, radii)
			p.ClipPath(&scratch.clipPath, paint)
		} else {
			paint()
		}
	})
	return el
}

func resolveSparklineDomain(ring *RingBuffer, values []float64, optMin, optMax float64, hasExplicit, baselineZero bool) (lo, hi float64, ok bool) {
	if (hasExplicit || (isFiniteF64(optMin) && isFiniteF64(optMax) && optMax > optMin)) &&
		isFiniteF64(optMin) && isFiniteF64(optMax) && optMax > optMin {
		return optMin, optMax, true
	}

	if ring != nil {
		lo, hi, ok = ring.MinMax()
	} else {
		lo, hi = math.Inf(1), math.Inf(-1)
		for _, v := range values {
			if !isFiniteF64(v) {
				continue
			}
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
			ok = true
		}
	}
	if !ok {
		return 0, 0, false
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
		if baselineZero && hi > 0 {
			return 0, hi * 1.25, true
		}
		pad := math.Abs(lo) * 0.2
		if pad == 0 {
			pad = 1
		}
		return lo - pad, hi + pad, true
	}
	// Add gentle breathing headroom so peaks look airy and never kiss the top.
	span := hi - lo
	if !baselineZero {
		lo -= span * 0.06
	}
	hi += span * 0.08
	return lo, hi, true
}

func sparklineSampleCount(ring *RingBuffer, values []float64) int {
	if ring != nil {
		return ring.Len()
	}
	return len(values)
}

func sparklineSampleAt(ring *RingBuffer, values []float64, i int) float64 {
	if ring != nil {
		return ring.At(i)
	}
	return values[i]
}

// buildSparklinePaths populates s.linePath and s.areaPath in-place with zero
// heap allocations once s's backing slices are warmed.
func buildSparklinePaths(
	s *sparklineScratch,
	plotRect ui.Rect,
	bottomY float32,
	ring *RingBuffer,
	values []float64,
	domainLo, domainHi float64,
	curve CurveStyle,
	pixelWidth int,
) (hasLine bool, lastPt [2]float32, hasLast bool) {
	s.linePath.Reset()
	s.areaPath.Reset()

	n := sparklineSampleCount(ring, values)
	if n == 0 || !isFiniteF64(domainLo) || !isFiniteF64(domainHi) || !(domainHi > domainLo) {
		return false, [2]float32{}, false
	}
	invSpan := 1.0 / (domainHi - domainLo)

	// Single finite sample renders as a horizontal line + gradient fill so
	// brand-new telemetry streams look intentional immediately.
	if n == 1 {
		v := sparklineSampleAt(ring, values, 0)
		if !isFiniteF64(v) {
			return false, [2]float32{}, false
		}
		yf := clampF32(float32((v-domainLo)*invSpan), 0, 1)
		py := chart.Y(plotRect, yf)
		x0, x1 := plotRect.X, plotRect.X+plotRect.W
		s.linePath.MoveTo(x0, py)
		s.linePath.LineTo(x1, py)
		s.areaPath.MoveTo(x0, py)
		s.areaPath.LineTo(x1, py)
		s.areaPath.LineTo(x1, bottomY)
		s.areaPath.LineTo(x0, bottomY)
		s.areaPath.Close()
		return true, [2]float32{x1, py}, true
	}

	// For dense histories (> 2 samples per pixel column), reuse chart.MinMax
	// with s.rawX/s.rawY/s.dsX/s.dsY so visual spikes survive and path size stays bounded.
	if pixelWidth > 0 && n > pixelWidth*2 {
		s.rawX = ensureF32Cap(s.rawX, n)
		s.rawY = ensureF32Cap(s.rawY, n)
		invN := float32(1) / float32(n-1)
		for i := range n {
			s.rawX[i] = float32(i) * invN
			v := sparklineSampleAt(ring, values, i)
			if !isFiniteF64(v) {
				s.rawY[i] = float32(math.NaN())
			} else {
				s.rawY[i] = clampF32(float32((v-domainLo)*invSpan), 0, 1)
			}
		}
		s.dsX, s.dsY = chart.MinMax(s.dsX, s.dsY, s.rawX, s.rawY, pixelWidth)
		return buildSparklineRunsFromFractions(s, plotRect, bottomY, s.dsX, s.dsY, curve)
	}

	// Direct finite-run walk without intermediate normalization slices.
	invN := float32(1) / float32(n-1)
	runStart := -1
	for i := 0; i <= n; i++ {
		finite := false
		var v float64
		if i < n {
			v = sparklineSampleAt(ring, values, i)
			finite = isFiniteF64(v)
		}
		if finite {
			if runStart < 0 {
				runStart = i
				s.pts = s.pts[:0]
			}
			xf := float32(i) * invN
			yf := clampF32(float32((v-domainLo)*invSpan), 0, 1)
			pt := [2]float32{chart.X(plotRect, xf), chart.Y(plotRect, yf)}
			s.pts = append(s.pts, pt)
			continue
		}
		if runStart >= 0 {
			if appendSparklineRun(s, bottomY, curve) {
				hasLine = true
				lastPt = s.pts[len(s.pts)-1]
				hasLast = true
			}
			runStart = -1
		}
	}
	return hasLine, lastPt, hasLast
}

func buildSparklineRunsFromFractions(
	s *sparklineScratch,
	plotRect ui.Rect,
	bottomY float32,
	xs, ys []float32,
	curve CurveStyle,
) (hasLine bool, lastPt [2]float32, hasLast bool) {
	n := min(len(xs), len(ys))
	runStart := -1
	for i := 0; i <= n; i++ {
		finite := i < n && isFiniteF32(xs[i]) && isFiniteF32(ys[i])
		if finite {
			if runStart < 0 {
				runStart = i
				s.pts = s.pts[:0]
			}
			s.pts = append(s.pts, [2]float32{chart.X(plotRect, xs[i]), chart.Y(plotRect, ys[i])})
			continue
		}
		if runStart >= 0 {
			if appendSparklineRun(s, bottomY, curve) {
				hasLine = true
				lastPt = s.pts[len(s.pts)-1]
				hasLast = true
			}
			runStart = -1
		}
	}
	return hasLine, lastPt, hasLast
}

func ensureF32Cap(buf []float32, n int) []float32 {
	if cap(buf) < n {
		return make([]float32, n)
	}
	return buf[:n]
}

// appendSparklineRun writes the current contiguous finite run in s.pts into
// both s.linePath and s.areaPath with zero heap allocations.
func appendSparklineRun(s *sparklineScratch, bottomY float32, curve CurveStyle) bool {
	pts := s.pts
	n := len(pts)
	if n == 0 {
		return false
	}
	if n == 1 {
		// Isolated single point between gaps: draw a tiny crisp 1 DIP dash.
		p0 := pts[0]
		s.linePath.MoveTo(p0[0]-0.5, p0[1])
		s.linePath.LineTo(p0[0]+0.5, p0[1])
		return true
	}

	s.linePath.MoveTo(pts[0][0], pts[0][1])
	s.areaPath.MoveTo(pts[0][0], pts[0][1])

	switch curve {
	case CurveSmooth:
		if n < 3 {
			appendLinearSegments(&s.linePath, &s.areaPath, pts)
		} else {
			appendMonotoneSegments(s, pts)
		}
	case CurveStepAfter:
		for i := 1; i < n; i++ {
			p0, p1 := pts[i-1], pts[i]
			s.linePath.LineTo(p1[0], p0[1])
			s.linePath.LineTo(p1[0], p1[1])
			s.areaPath.LineTo(p1[0], p0[1])
			s.areaPath.LineTo(p1[0], p1[1])
		}
	case CurveStepBefore:
		for i := 1; i < n; i++ {
			p0, p1 := pts[i-1], pts[i]
			s.linePath.LineTo(p0[0], p1[1])
			s.linePath.LineTo(p1[0], p1[1])
			s.areaPath.LineTo(p0[0], p1[1])
			s.areaPath.LineTo(p1[0], p1[1])
		}
	case CurveStep:
		for i := 1; i < n; i++ {
			p0, p1 := pts[i-1], pts[i]
			mid := (p0[0] + p1[0]) * 0.5
			s.linePath.LineTo(mid, p0[1])
			s.linePath.LineTo(mid, p1[1])
			s.linePath.LineTo(p1[0], p1[1])
			s.areaPath.LineTo(mid, p0[1])
			s.areaPath.LineTo(mid, p1[1])
			s.areaPath.LineTo(p1[0], p1[1])
		}
	default:
		appendLinearSegments(&s.linePath, &s.areaPath, pts)
	}

	last := pts[n-1]
	first := pts[0]
	s.areaPath.LineTo(last[0], bottomY)
	s.areaPath.LineTo(first[0], bottomY)
	s.areaPath.Close()
	return true
}

func appendLinearSegments(linePath, areaPath *ui.Path, pts [][2]float32) {
	for i := 1; i < len(pts); i++ {
		p := pts[i]
		linePath.LineTo(p[0], p[1])
		areaPath.LineTo(p[0], p[1])
	}
}

// appendMonotoneSegments computes Steffen's monotone cubic Hermite spline
// matching github.com/egoist/mygo/ui/chart.CurveMonotone in-place without allocating.
func appendMonotoneSegments(s *sparklineScratch, pts [][2]float32) {
	n := len(pts)
	s.slopes = ensureF32Cap(s.slopes, n)
	m := s.slopes
	for i := 1; i < n-1; i++ {
		m[i] = steffenSlope(pts[i-1], pts[i], pts[i+1])
	}
	m[0] = steffenEndSlope(pts[0], pts[1], m[1])
	m[n-1] = steffenEndSlope(pts[n-2], pts[n-1], m[n-2])

	for i := 1; i < n; i++ {
		p0, p1 := pts[i-1], pts[i]
		dx := (p1[0] - p0[0]) / 3
		c1x, c1y := p0[0]+dx, p0[1]+dx*m[i-1]
		c2x, c2y := p1[0]-dx, p1[1]-dx*m[i]
		s.linePath.CubeTo(c1x, c1y, c2x, c2y, p1[0], p1[1])
		s.areaPath.CubeTo(c1x, c1y, c2x, c2y, p1[0], p1[1])
	}
}

func steffenSlope(a, b, c [2]float32) float32 {
	h0, h1 := b[0]-a[0], c[0]-b[0]
	if h0 == 0 || h1 == 0 {
		return 0
	}
	s0, s1 := (b[1]-a[1])/h0, (c[1]-b[1])/h1
	if math.Signbit(float64(s0)) != math.Signbit(float64(s1)) || s0 == 0 || s1 == 0 {
		return 0
	}
	p := (s0*h1 + s1*h0) / (h0 + h1)
	m := min(absF32(s0), absF32(s1), 0.5*absF32(p))
	if s0 < 0 {
		return -2 * m
	}
	return 2 * m
}

func steffenEndSlope(a, b [2]float32, t float32) float32 {
	h := b[0] - a[0]
	if h == 0 {
		return t
	}
	return (3*(b[1]-a[1])/h - t) / 2
}

func absF32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
