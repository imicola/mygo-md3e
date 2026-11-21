package md3e

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// ProgressOptions configures the linear and circular MD3E progress
// indicators. The zero value is an enabled 4 DIP thick indicator with a 4 DIP
// track gap that fills its column (linear) or is 48 DIP across (circular).
type ProgressOptions struct {
	// Label names the indicator for assistive technology.
	Label string
	// Disabled shows MD3E's disabled colors without ui's automatic dimming.
	Disabled bool
	// Thickness is the bar or ring thickness in DIP; 4 when zero or invalid.
	Thickness float32
	// TrackGap is the clear space between the active segment and the track, in
	// DIP; 4 when zero or invalid.
	TrackGap float32
	// Width fixes a linear indicator's width; zero fills its column.
	Width float32
	// Size is a circular indicator's square side in DIP; 48 when zero or
	// invalid.
	Size float32
	// Buffer is an optional secondary progress value in 0..1 (for buffered
	// media or streaming sync), drawn between the active segment and the
	// unfinished track with TrackGap clearance. Zero (or non-positive /
	// non-finite) keeps the default single-progress appearance.
	Buffer float64
}

// Linear progress metrics in DIP.
const (
	linearProgressRowHeight = 16
	progressDefaultThick    = 4
	progressDefaultGap      = 4
	progressDefaultCircular = 48

	// progressCycle is the indeterminate sweep's period.
	progressCycle = 1800 * time.Millisecond
	// progressWavePeriod is one full phase of a determinate wavy indicator.
	progressWavePeriod = 1500 * time.Millisecond
	// progressWaveLength is one sine wave of a wavy indicator, in DIP.
	progressWaveLength = 24
)

// progressValueKey is the AnimateTo channel of a determinate indicator's value.
type progressValueKey struct{}

// progressBufferKey is the AnimateTo channel of a determinate indicator's buffer.
type progressBufferKey struct{}

// progressStartKey is the Local holding an indeterminate indicator's start time.
type progressStartKey struct{}

// progressPathKey is the Local holding an indicator's reused paths.
type progressPathKey struct{}

// progressPaths are the paths an indicator rebuilds each frame, reusing their
// storage rather than allocating.
type progressPaths struct {
	active        ui.Path
	secondary     ui.Path
	track         ui.Path
	paused        bool
	pausedElapsed time.Duration
	lastElapsed   time.Duration
}

// progressIndicatorKind selects one of the four indicators.
type progressIndicatorKind int

const (
	progressLinear progressIndicatorKind = iota
	progressLinearWavy
	progressCircular
	progressCircularWavy
)

// normalizeProgressValue clamps a determinate value to 0..1 and normalizes
// non-finite values. A negative finite value selects indeterminate motion.
func normalizeProgressValue(v float64) (value float64, indeterminate bool) {
	switch {
	case math.IsNaN(v), math.IsInf(v, -1):
		return 0, false
	case math.IsInf(v, 1):
		return 1, false
	case v < 0:
		return 0, true
	case v > 1:
		return 1, false
	}
	return v, false
}

// normalizeBufferValue clamps an optional buffer value to 0..1, treating
// non-positive, NaN, and -Inf inputs as 0 (disabled).
func normalizeBufferValue(v float64) float64 {
	switch {
	case math.IsNaN(v), math.IsInf(v, -1), v <= 0:
		return 0
	case math.IsInf(v, 1), v >= 1:
		return 1
	}
	return v
}

// LinearProgress builds a Material Design 3 linear determinate progress
// indicator (value in 0..1) or an indeterminate one (value < 0): a rounded
// active segment, an optional buffered segment via ProgressOptions.Buffer, a
// clear track gap, an unfinished track and a stop indicator dot at the far end
// of the track. Right to left, it fills from the right and the dot sits at the
// left end. Determinate changes animate with the Default Spatial spring;
// indeterminate mode runs a continuous two-segment relay (Primary + Trailing
// bars) and pauses automatically when off-screen. ReduceMotion settles at once
// and shows a still centered segment without asking for frames.
func LinearProgress(c *ui.Context, value float64, opts ...ProgressOptions) ui.Element {
	return progressIndicator(c, progressLinear, value, opts...)
}

// LinearWavyProgress builds the wavy variant of LinearProgress: the active
// segment is a sine wave whose amplitude smooths to zero at its ends and near
// 0% and 100%. An indeterminate indicator runs a continuous two-segment wavy
// relay across the track and pauses when scrolled off-screen.
func LinearWavyProgress(c *ui.Context, value float64, opts ...ProgressOptions) ui.Element {
	return progressIndicator(c, progressLinearWavy, value, opts...)
}

// CircularProgress builds a Material Design 3 circular progress indicator:
// a rounded active arc, a track gap and the unfinished ring. Determinate values
// in 0..1 animate; a negative value spins an extending and shrinking arc.
// ReduceMotion freezes an indeterminate ring and settles a determinate one
// without extra frames.
func CircularProgress(c *ui.Context, value float64, opts ...ProgressOptions) ui.Element {
	return progressIndicator(c, progressCircular, value, opts...)
}

// CircularWavyProgress builds the wavy variant of CircularProgress: the ring's
// radius follows R + A·sin(kθ + φ), converging near the arc ends and at 0% and
// 100%. An indeterminate ring also turns continuously.
func CircularWavyProgress(c *ui.Context, value float64, opts ...ProgressOptions) ui.Element {
	return progressIndicator(c, progressCircularWavy, value, opts...)
}

// progressIndicator builds one of the four indicators.
func progressIndicator(c *ui.Context, kind progressIndicatorKind, value float64, opts ...ProgressOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: progress indicators take at most one ProgressOptions")
	}
	var o ProgressOptions
	if len(opts) == 1 {
		o = opts[0]
	}

	raw, indeterminate := normalizeProgressValue(value)
	bufRaw := normalizeBufferValue(o.Buffer)
	if indeterminate {
		bufRaw = 0
	}
	th := CurrentTheme(c)
	cs := th.Colors
	// A disabled ancestor also disables the indicator, which must show MD3E's
	// disabled tokens rather than the enabled palette.
	disabled := o.Disabled

	thickness := sanitizePositiveF32(o.Thickness, progressDefaultThick)
	gap := sanitizePositiveF32(o.TrackGap, progressDefaultGap)

	active := cs.Primary
	bufferColor := cs.Primary.Alpha(0.28)
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}

	var el ui.Element
	linear := kind == progressLinear || kind == progressLinearWavy
	if linear {
		el = ui.Box(c).Height(linearProgressRowHeight).Shrink(0).DimDisabled(false)
		if isPositiveFiniteF32(o.Width) {
			el.Width(o.Width)
		} else {
			el.FillWidth()
		}
		thickness = clampF32(thickness, 2, linearProgressRowHeight-6)
		gap = clampF32(gap, 1, 16)
	} else {
		size := sanitizePositiveF32(o.Size, progressDefaultCircular)
		el = ui.Box(c).Size(size, size).Shrink(0).DimDisabled(false)
		thickness = clampF32(thickness, 2, size/3)
		gap = clampF32(gap, 1, size/6)
	}
	el.Role(ui.RoleProgress).Disabled(o.Disabled)
	disabled = disabled || el.IsDisabled()
	if disabled {
		active = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		bufferColor = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity) * 0.6)
		track = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
	}
	if o.Label != "" {
		el.Label(o.Label)
	}
	if indeterminate {
		// Progress of unknown length: a negative value, as ui.Progress reports.
		el.Range(0, 1, -1)
	} else {
		el.Range(0, 1, raw)
	}

	reduceMotion := th.ReduceMotion || c.Preferences().ReduceMotion
	shown := float32(raw)
	bufShown := float32(0)
	if !indeterminate {
		shown = animateValue(el, progressValueKey{}, float32(raw), th.Animation(motion.DefaultSpatial), reduceMotion)
		if bufRaw > 0 {
			bufShown = animateValue(el, progressBufferKey{}, float32(bufRaw), th.Animation(motion.DefaultSpatial), reduceMotion)
		}
	}
	start := ui.Local(el, progressStartKey{}, func() time.Time { return c.Now() })
	paths := ui.Local(el, progressPathKey{}, func() progressPaths { return progressPaths{} })

	rtl := c.Direction() == ui.RightToLeft
	wavy := kind == progressLinearWavy || kind == progressCircularWavy

	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		live := el.Valid()
		reduce := reduceMotion || p.Preferences().ReduceMotion
		visible := rectIntersects(r, p.Visible())
		elapsed := paths.lastElapsed
		if live {
			now := p.Now()
			switch {
			case reduce:
				elapsed = 0
			case disabled || !visible:
				if !paths.paused {
					paths.paused = true
					paths.pausedElapsed = max(now.Sub(*start), 0)
				}
				elapsed = paths.pausedElapsed
			default:
				if paths.paused {
					paths.paused = false
					*start = now.Add(-paths.pausedElapsed)
				}
				elapsed = max(now.Sub(*start), 0)
			}
			paths.lastElapsed = elapsed
		} else if reduce {
			elapsed = 0
		}
		if visible {
			p.Clip(r, 0, func() {
				if linear {
					paintLinearProgress(p, r, paths, linearProgressPaint{
						thickness:     thickness,
						gap:           gap,
						value:         clampF32(shown, 0, 1),
						buffer:        clampF32(bufShown, 0, 1),
						indeterminate: indeterminate,
						wavy:          wavy && !disabled,
						reduce:        reduce,
						elapsed:       elapsed,
						rtl:           rtl,
						active:        active,
						bufferColor:   bufferColor,
						track:         track,
					})
				} else {
					paintCircularProgress(p, r, paths, circularProgressPaint{
						thickness:     thickness,
						gap:           gap,
						value:         clampF32(shown, 0, 1),
						buffer:        clampF32(bufShown, 0, 1),
						indeterminate: indeterminate,
						wavy:          wavy && !disabled,
						reduce:        reduce,
						elapsed:       elapsed,
						active:        active,
						bufferColor:   bufferColor,
						track:         track,
					})
				}
			})
		}
		// A determinate wavy indicator keeps flowing between its ends; at 0, 1,
		// disabled, off-screen or under reduced motion it rests flat and asks
		// for no frames, as a plain determinate indicator does.
		flowing := indeterminate || (wavy && shown > 0 && shown < 1)
		if live && visible && flowing && !reduce && !disabled {
			p.AnimationFrame()
		}
	})
	return el
}

// linearProgressPaint is the resolved snapshot a linear indicator draws.
type linearProgressPaint struct {
	thickness     float32
	gap           float32
	value         float32
	buffer        float32
	indeterminate bool
	wavy          bool
	reduce        bool
	elapsed       time.Duration
	rtl           bool
	active        ui.Color
	bufferColor   ui.Color
	track         ui.Color
}

// progressMirrorX maps a logical horizontal offset inside r to screen X,
// mirroring when RTL.
func progressMirrorX(rtl bool, r ui.Rect, logical float32) float32 {
	if rtl {
		return r.X + r.W - logical
	}
	return r.X + logical
}

// progressSpan maps logical [a, b] into ordered screen coordinates [minX, maxX].
func progressSpan(rtl bool, r ui.Rect, a, b float32) (float32, float32) {
	xa, xb := progressMirrorX(rtl, r, a), progressMirrorX(rtl, r, b)
	return min(xa, xb), max(xa, xb)
}

// paintLinearProgress paints one linear progress frame into r.
func paintLinearProgress(p *ui.Painter, r ui.Rect, paths *progressPaths, s linearProgressPaint) {
	if paths == nil {
		return
	}
	cy := r.Y + r.H/2
	dotD := min(s.thickness, r.H)
	amplitude := min(s.thickness*0.75, max((r.H-s.thickness)/2, 0))
	if !s.wavy || s.reduce {
		amplitude = 0
	}
	if s.indeterminate {
		paintIndeterminateLinearProgress(p, r, paths, s, cy, dotD, r.W, amplitude)
		return
	}

	// Determinate: active segment, optional buffer segment, unfinished track,
	// and stop indicator dot, keeping TrackGap between adjacent segments.
	activeUsed := r.W * s.value
	bufferUsed := r.W * s.buffer
	trackEnd := r.W - dotD - s.gap
	if bufferUsed > activeUsed {
		bufStart := float32(0)
		if activeUsed > 0.5 {
			bufStart = activeUsed + s.gap
		}
		bufEnd := bufferUsed
		hasBuf := bufEnd > bufStart+0.5
		if hasBuf {
			ba, bb := progressSpan(s.rtl, r, bufStart, bufEnd)
			p.Fill(ui.Rect{X: ba, Y: cy - s.thickness/2, W: bb - ba, H: s.thickness}, s.bufferColor, min(s.thickness/2, (bb-ba)/2))
			trackStart := bufEnd + s.gap
			if trackEnd > trackStart {
				ta, tb := progressSpan(s.rtl, r, trackStart, trackEnd)
				p.Fill(ui.Rect{X: ta, Y: cy - s.thickness/2, W: tb - ta, H: s.thickness}, s.track, s.thickness/2)
			}
			// Skip the opaque stop dot when a translucent buffer covers it.
			if bufEnd <= trackEnd {
				da, db := progressSpan(s.rtl, r, r.W-dotD, r.W)
				p.Fill(ui.Rect{X: da, Y: cy - dotD/2, W: db - da, H: dotD}, s.track, dotD/2)
			}
		} else {
			trackStart := activeUsed + s.gap
			if trackEnd > trackStart {
				ta, tb := progressSpan(s.rtl, r, trackStart, trackEnd)
				p.Fill(ui.Rect{X: ta, Y: cy - s.thickness/2, W: tb - ta, H: s.thickness}, s.track, s.thickness/2)
			}
			da, db := progressSpan(s.rtl, r, r.W-dotD, r.W)
			p.Fill(ui.Rect{X: da, Y: cy - dotD/2, W: db - da, H: dotD}, s.track, dotD/2)
		}
	} else {
		trackStart := activeUsed + s.gap
		if trackEnd > trackStart {
			ta, tb := progressSpan(s.rtl, r, trackStart, trackEnd)
			p.Fill(ui.Rect{X: ta, Y: cy - s.thickness/2, W: tb - ta, H: s.thickness}, s.track, s.thickness/2)
		}
		da, db := progressSpan(s.rtl, r, r.W-dotD, r.W)
		p.Fill(ui.Rect{X: da, Y: cy - dotD/2, W: db - da, H: dotD}, s.track, dotD/2)
	}

	// The amplitude converges to zero as the value nears either end, so a
	// short bar never pokes out of the track.
	amplitude *= progressEdgeScale(s.value)
	if !s.wavy || amplitude < 0.15 || activeUsed <= s.thickness+0.5 {
		if activeUsed > 0.5 {
			aa, ab := progressSpan(s.rtl, r, 0, activeUsed)
			p.Fill(ui.Rect{X: aa, Y: cy - s.thickness/2, W: ab - aa, H: s.thickness}, s.active, min(s.thickness/2, (ab-aa)/2))
		}
		return
	}
	capPad := min(s.thickness/2, activeUsed/2)
	path := progressWavePath(&paths.active, capPad, activeUsed-capPad, cy, amplitude, progressWavePhase(s.elapsed), s.rtl, r)
	path.Cap(ui.RoundCap)
	p.StrokePath(path, s.thickness, s.active)
}

// progressWavePhase returns the phase of a determinate wave at elapsed, a full
// turn every progressWavePeriod so that it never jumps as the clock loops.
func progressWavePhase(elapsed time.Duration) float32 {
	turns := float64(max(elapsed, 0)) / float64(progressWavePeriod)
	return float32(2 * math.Pi * math.Mod(turns, 1))
}

// progressLinearSeg is a logical [lo, hi] active bar span in 0..1.
type progressLinearSeg struct {
	lo, hi float64
}

// progressCubicBezier evaluates cubic-bezier(x1, y1, x2, y2) at input x in 0..1.
func progressCubicBezier(x1, y1, x2, y2, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	sampleX := func(t float64) float64 {
		u := 1 - t
		return 3*u*u*t*x1 + 3*u*t*t*x2 + t*t*t
	}
	sampleDX := func(t float64) float64 {
		u := 1 - t
		return 3*u*u*x1 + 6*u*t*(x2-x1) + 3*t*t*(1-x2)
	}
	t := x
	lo, hi := 0.0, 1.0
	for range 6 {
		err := sampleX(t) - x
		if math.Abs(err) < 1e-6 {
			break
		}
		if err < 0 {
			lo = t
		} else {
			hi = t
		}
		dx := sampleDX(t)
		if math.Abs(dx) > 1e-6 {
			next := t - err/dx
			if next > lo && next < hi {
				t = next
				continue
			}
		}
		t = 0.5 * (lo + hi)
	}
	u := 1 - t
	return 3*u*u*t*y1 + 3*u*t*t*y2 + t*t*t
}

// progressIntervalEase maps u over [start, end] through cubic-bezier(x1, y1, x2, y2).
func progressIntervalEase(u, start, end, x1, y1, x2, y2 float64) float64 {
	if u <= start {
		return 0
	}
	if u >= end {
		return 1
	}
	return progressCubicBezier(x1, y1, x2, y2, (u-start)/(end-start))
}

// indeterminateLinearSegments computes the MD3 dual-segment relay (Primary +
// Trailing bars) for cycle phase u in [0, 1), returning up to two ordered
// non-overlapping spans so every frame has continuous active progress.
func indeterminateLinearSegments(u float64, reduce bool) ([2]progressLinearSeg, int) {
	var segs [2]progressLinearSeg
	if reduce {
		segs[0] = progressLinearSeg{lo: 0.325, hi: 0.675}
		return segs, 1
	}
	count := 0
	push := func(lo, hi float64) {
		lo = max(lo, 0)
		hi = min(hi, 1)
		if hi-lo > 1e-4 && count < len(segs) {
			segs[count] = progressLinearSeg{lo: lo, hi: hi}
			count++
		}
	}

	// Primary bar sweeps u in [0.00, 0.68].
	head1 := progressIntervalEase(u, 0.00, 0.42, 0.2, 0.0, 0.8, 1.0)
	tail1 := progressIntervalEase(u, 0.18, 0.68, 0.4, 0.0, 1.0, 1.0)
	push(tail1, head1)

	// Trailing bar spans [0.48, 1.24], wrapping seamlessly across u = 0 / 1 so
	// the track is never empty at cycle boundaries.
	for _, u2 := range [2]float64{u, u + 1} {
		head2 := progressIntervalEase(u2, 0.48, 0.88, 0.15, 0.0, 0.65, 1.0)
		tail2 := progressIntervalEase(u2, 0.66, 1.24, 0.35, 0.0, 0.85, 1.0)
		push(tail2, head2)
	}

	if count == 2 && segs[1].lo < segs[0].lo {
		segs[0], segs[1] = segs[1], segs[0]
	}
	return segs, count
}

// paintIndeterminateLinearProgress paints the gapped track and the MD3
// two-segment indeterminate linear indicator (flat or wavy).
func paintIndeterminateLinearProgress(p *ui.Painter, r ui.Rect, paths *progressPaths, s linearProgressPaint, cy, dotD, width, amplitude float32) {
	cycle := float64(progressCycle)
	u := 0.0
	if !s.reduce {
		u = math.Mod(math.Max(float64(s.elapsed), 0), cycle) / cycle
	}
	segs, count := indeterminateLinearSegments(u, s.reduce)

	// Paint track segments with TrackGap clearance around every active bar.
	trackEnd := width - dotD - s.gap
	cur := float32(0)
	maxEnd := float32(0)
	var pxSegs [2]struct{ start, end float32 }
	pxCount := 0
	for i := range count {
		start, end := float32(segs[i].lo)*width, float32(segs[i].hi)*width
		if end-start <= 0.5 {
			continue
		}
		pxSegs[pxCount] = struct{ start, end float32 }{start: start, end: end}
		pxCount++
		tEnd := min(start-s.gap, trackEnd)
		if tEnd > cur+0.5 {
			ta, tb := progressSpan(s.rtl, r, cur, tEnd)
			p.Fill(ui.Rect{X: ta, Y: cy - s.thickness/2, W: tb - ta, H: s.thickness}, s.track, s.thickness/2)
		}
		cur = max(cur, end+s.gap)
		maxEnd = max(maxEnd, end)
	}
	if trackEnd > cur+0.5 {
		ta, tb := progressSpan(s.rtl, r, cur, trackEnd)
		p.Fill(ui.Rect{X: ta, Y: cy - s.thickness/2, W: tb - ta, H: s.thickness}, s.track, s.thickness/2)
	}
	if maxEnd <= trackEnd {
		da, db := progressSpan(s.rtl, r, width-dotD, width)
		p.Fill(ui.Rect{X: da, Y: cy - dotD/2, W: db - da, H: dotD}, s.track, dotD/2)
	}

	phase := float32(u * 6 * math.Pi)
	segPaths := [2]*ui.Path{&paths.active, &paths.secondary}
	for i := range pxCount {
		start, end := pxSegs[i].start, pxSegs[i].end
		length := end - start
		segAmp := amplitude * clampF32((length-s.thickness)/(progressWaveLength*0.75), 0, 1)
		if !s.wavy || segAmp < 0.15 || length <= s.thickness+0.5 {
			aa, ab := progressSpan(s.rtl, r, start, end)
			p.Fill(ui.Rect{X: aa, Y: cy - s.thickness/2, W: ab - aa, H: s.thickness}, s.active, min(s.thickness/2, (ab-aa)/2))
			continue
		}
		capPad := min(s.thickness/2, length/2)
		path := progressWavePath(segPaths[i], start+capPad, end-capPad, cy, segAmp, phase, s.rtl, r)
		path.Cap(ui.RoundCap)
		p.StrokePath(path, s.thickness, s.active)
	}
}

// progressEdgeScale smooths a wave's amplitude to zero at 0 and 1.
func progressEdgeScale(v float32) float32 {
	edge := min(v, 1-v)
	t := clampF32(edge/0.12, 0, 1)
	return t * t * (3 - 2*t)
}

// progressWavePath builds a cubic Hermite-fitted sine wave between logical
// from and to, centered on centerY, with its amplitude anchored at zero at both ends.
func progressWavePath(path *ui.Path, from, to, centerY, amplitude, phase float32, rtl bool, r ui.Rect) *ui.Path {
	path.Reset()
	length := to - from
	if length <= 0 {
		return path
	}
	n := max(6, int(length/6))
	waves := float64(length / progressWaveLength)
	w := waves * 2 * math.Pi
	df := 1.0 / float64(n)
	eval := func(f float64) (x, y, dydf float32) {
		x = progressMirrorX(rtl, r, from+length*float32(f))
		var env, dEnv float64
		switch {
		case f < 0.08:
			t := f / 0.08
			env = t * t * (3 - 2*t)
			dEnv = 6 * t * (1 - t) / 0.08
		case f > 0.92:
			t := (1 - f) / 0.08
			env = t * t * (3 - 2*t)
			dEnv = -6 * t * (1 - t) / 0.08
		default:
			env = 1
		}
		ang := float64(phase) + w*f
		s, c := math.Sin(ang), math.Cos(ang)
		y = centerY + amplitude*float32(env*s)
		dydf = amplitude * float32(dEnv*s+env*w*c)
		return x, y, dydf
	}
	x0, y0, m0 := eval(0)
	path.MoveTo(x0, y0)
	df3 := float32(df / 3)
	for i := 1; i <= n; i++ {
		x1, y1, m1 := eval(float64(i) * df)
		dx3 := (x1 - x0) / 3
		path.CubeTo(x0+dx3, y0+m0*df3, x1-dx3, y1-m1*df3, x1, y1)
		x0, y0, m0 = x1, y1, m1
	}
	return path
}

// circularProgressPaint is the resolved snapshot a circular indicator draws.
type circularProgressPaint struct {
	thickness     float32
	gap           float32
	value         float32
	buffer        float32
	indeterminate bool
	wavy          bool
	reduce        bool
	elapsed       time.Duration
	active        ui.Color
	bufferColor   ui.Color
	track         ui.Color
}

// paintCircularProgress paints one circular progress frame into r.
func paintCircularProgress(p *ui.Painter, r ui.Rect, paths *progressPaths, s circularProgressPaint) {
	if paths == nil {
		return
	}
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	radius := min(r.W, r.H)/2 - s.thickness/2
	amplitude := float32(0)
	if s.wavy && !s.reduce {
		amplitude = min(min(r.W, r.H)*0.055, max(radius*0.4, 0))
		radius -= amplitude
	}
	if radius <= s.thickness*0.5 || radius <= 0 {
		return
	}
	// The gap is the angle a track gap plus the round cap's radius spans; the
	// wave's outward bulge widens it so the track never touches the crest.
	gapDeg := (s.gap + s.thickness/2 + amplitude) / radius * (180 / math.Pi)

	if s.indeterminate {
		var rotation, sweep, phase float32
		if s.reduce {
			sweep = 90
		} else {
			cycle := float64(progressCycle)
			totalTurns := math.Max(float64(s.elapsed), 0) / cycle
			u := math.Mod(totalTurns, 1)
			rotation = float32(math.Mod(totalTurns*540, 360))
			sweep = 90 + 110*(0.5+0.5*float32(math.Sin(2*math.Pi*u)))
			if s.wavy {
				phase = float32(u * 4 * math.Pi)
			}
		}
		// The unfinished track leaves a clear gap around the moving arc rather
		// than running underneath it.
		trackSweep := 360 - sweep - 2*gapDeg
		if trackSweep > 0.5 {
			progressArcPath(&paths.track, cx, cy, radius, rotation+sweep+gapDeg, trackSweep, 0, 0)
			paths.track.Cap(ui.RoundCap)
			p.StrokePath(&paths.track, s.thickness, s.track)
		}
		progressArcPath(&paths.active, cx, cy, radius, rotation, sweep, amplitude, phase)
		paths.active.Cap(ui.RoundCap)
		p.StrokePath(&paths.active, s.thickness, s.active)
		return
	}

	// Determinate: the active arc first, optional buffer arc, then the rest of
	// the ring, each separated by a clear gap at the joins.
	activeSweep := s.value * 360
	bufferSweep := s.buffer * 360
	phase := progressWavePhase(s.elapsed)
	if bufferSweep > activeSweep && activeSweep < 359.5 {
		paintCircularBufferedDeterminate(p, paths, s, cx, cy, radius, gapDeg, amplitude, phase, activeSweep, bufferSweep)
		return
	}
	if activeSweep <= 0.01 {
		progressArcPath(&paths.track, cx, cy, radius, 0, 360, 0, 0)
		paths.track.Cap(ui.RoundCap)
		p.StrokePath(&paths.track, s.thickness, s.track)
		return
	}
	if activeSweep >= 359.5 {
		activeAmp := amplitude * progressEdgeScale(s.value)
		progressArcPath(&paths.active, cx, cy, radius, 0, 360, activeAmp, phase)
		paths.active.Cap(ui.RoundCap)
		p.StrokePath(&paths.active, s.thickness, s.active)
		return
	}
	a0 := gapDeg / 2
	a1 := activeSweep - gapDeg/2
	if a1 > a0 {
		activeAmp := amplitude * progressEdgeScale(s.value)
		progressArcPath(&paths.active, cx, cy, radius, a0, a1-a0, activeAmp, phase)
		paths.active.Cap(ui.RoundCap)
		p.StrokePath(&paths.active, s.thickness, s.active)
		rest := 360 - activeSweep - gapDeg
		if rest > 0.5 {
			progressArcPath(&paths.track, cx, cy, radius, a1+gapDeg, rest, 0, 0)
			paths.track.Cap(ui.RoundCap)
			p.StrokePath(&paths.track, s.thickness, s.track)
		}
		return
	}
	// Very small non-zero values draw a minimal round active dot at 12 o'clock
	// so the gap never leaves an unindicated hole.
	p.Fill(ui.Rect{
		X: cx - s.thickness/2,
		Y: cy - radius - s.thickness/2,
		W: s.thickness,
		H: s.thickness,
	}, s.active, s.thickness/2)
	trackSweep := float32(360) - 2*gapDeg
	if trackSweep > 0.5 {
		progressArcPath(&paths.track, cx, cy, radius, gapDeg, trackSweep, 0, 0)
		paths.track.Cap(ui.RoundCap)
		p.StrokePath(&paths.track, s.thickness, s.track)
	}
}

// paintCircularBufferedDeterminate paints determinate circular progress when
// bufferSweep > activeSweep, keeping TrackGap clearance between active, buffer,
// and unfinished track arcs.
func paintCircularBufferedDeterminate(p *ui.Painter, paths *progressPaths, s circularProgressPaint, cx, cy, radius, gapDeg, amplitude, phase, activeSweep, bufferSweep float32) {
	a0 := gapDeg / 2
	if activeSweep <= 0.01 {
		if bufferSweep >= 359.5 {
			progressArcPath(&paths.secondary, cx, cy, radius, 0, 360, 0, 0)
			paths.secondary.Cap(ui.RoundCap)
			p.StrokePath(&paths.secondary, s.thickness, s.bufferColor)
			return
		}
		b1 := bufferSweep - gapDeg/2
		if b1 > a0 {
			progressArcPath(&paths.secondary, cx, cy, radius, a0, b1-a0, 0, 0)
			paths.secondary.Cap(ui.RoundCap)
			p.StrokePath(&paths.secondary, s.thickness, s.bufferColor)
			rest := 360 - bufferSweep - gapDeg
			if rest > 0.5 {
				progressArcPath(&paths.track, cx, cy, radius, b1+gapDeg, rest, 0, 0)
				paths.track.Cap(ui.RoundCap)
				p.StrokePath(&paths.track, s.thickness, s.track)
			}
			return
		}
		p.Fill(ui.Rect{
			X: cx - s.thickness/2,
			Y: cy - radius - s.thickness/2,
			W: s.thickness,
			H: s.thickness,
		}, s.bufferColor, s.thickness/2)
		trackSweep := float32(360) - 2*gapDeg
		if trackSweep > 0.5 {
			progressArcPath(&paths.track, cx, cy, radius, gapDeg, trackSweep, 0, 0)
			paths.track.Cap(ui.RoundCap)
			p.StrokePath(&paths.track, s.thickness, s.track)
		}
		return
	}

	a1 := activeSweep - gapDeg/2
	var b0 float32
	if a1 > a0 {
		activeAmp := amplitude * progressEdgeScale(s.value)
		progressArcPath(&paths.active, cx, cy, radius, a0, a1-a0, activeAmp, phase)
		paths.active.Cap(ui.RoundCap)
		p.StrokePath(&paths.active, s.thickness, s.active)
		b0 = a1 + gapDeg
	} else {
		p.Fill(ui.Rect{
			X: cx - s.thickness/2,
			Y: cy - radius - s.thickness/2,
			W: s.thickness,
			H: s.thickness,
		}, s.active, s.thickness/2)
		b0 = gapDeg
	}

	maxEnd := float32(360) - gapDeg/2
	b1 := min(bufferSweep-gapDeg/2, maxEnd)
	if b1-b0 > 0.5 {
		progressArcPath(&paths.secondary, cx, cy, radius, b0, b1-b0, 0, 0)
		paths.secondary.Cap(ui.RoundCap)
		p.StrokePath(&paths.secondary, s.thickness, s.bufferColor)
		t0 := b1 + gapDeg
		rest := maxEnd - t0
		if rest > 0.5 {
			progressArcPath(&paths.track, cx, cy, radius, t0, rest, 0, 0)
			paths.track.Cap(ui.RoundCap)
			p.StrokePath(&paths.track, s.thickness, s.track)
		}
		return
	}
	rest := maxEnd - b0
	if rest > 0.5 {
		progressArcPath(&paths.track, cx, cy, radius, b0, rest, 0, 0)
		paths.track.Cap(ui.RoundCap)
		p.StrokePath(&paths.track, s.thickness, s.track)
	}
}

// progressArcPath builds a smooth arc from startDeg, clockwise from 12 o'clock,
// into path. A positive amplitude waves the radius as R + A·sin(6θ + phase),
// with the wave anchored at the arc's ends; cubic segments follow the wave's
// tangents so a small ring has no visible facets.
func progressArcPath(path *ui.Path, cx, cy, radius, startDeg, sweepDeg, amplitude, phase float32) *ui.Path {
	path.Reset()
	if sweepDeg <= 0 {
		return path
	}
	steps := max(12, int(sweepDeg/6))
	sweepRad := float64(sweepDeg) * math.Pi / 180
	eval := func(f float64) (x, y, dx, dy float32) {
		theta := float64(startDeg)*math.Pi/180 + sweepRad*f
		rad := float64(radius)
		drdf := 0.0
		if amplitude > 0 {
			env, dEnv := 1.0, 0.0
			switch {
			case f < 0.08:
				u := f / 0.08
				env, dEnv = u*u*(3-2*u), 6*u*(1-u)/0.08
			case f > 0.92:
				u := (1 - f) / 0.08
				env, dEnv = u*u*(3-2*u), -6*u*(1-u)/0.08
			}
			ang := 6*theta + float64(phase)
			sn, cs := math.Sincos(ang)
			rad += float64(amplitude) * env * sn
			drdf = float64(amplitude) * (dEnv*sn + env*cs*6*sweepRad)
		}
		sn, cs := math.Sincos(theta)
		x = cx + float32(rad*sn)
		y = cy - float32(rad*cs)
		dx = float32(drdf*sn + rad*cs*sweepRad)
		dy = float32(-drdf*cs + rad*sn*sweepRad)
		return x, y, dx, dy
	}
	x0, y0, dx0, dy0 := eval(0)
	path.MoveTo(x0, y0)
	df := 1 / float64(steps)
	step3 := float32(df / 3)
	for i := 1; i <= steps; i++ {
		x1, y1, dx1, dy1 := eval(float64(i) * df)
		path.CubeTo(x0+dx0*step3, y0+dy0*step3, x1-dx1*step3, y1-dy1*step3, x1, y1)
		x0, y0, dx0, dy0 = x1, y1, dx1, dy1
	}
	return path
}
