package md3e

import (
	"math"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/motion"
	"mygo-md3e/tokens"
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
}

// Linear progress metrics in DIP.
const (
	linearProgressRowHeight = 16
	progressDefaultThick    = 4
	progressDefaultGap      = 4
	progressDefaultCircular = 48

	// progressCycle is the indeterminate sweep's period.
	progressCycle = 1800 * time.Millisecond
	// progressWaveLength is one sine wave of a wavy indicator, in DIP.
	progressWaveLength = 24
)

// progressValueKey is the AnimateTo channel of a determinate indicator's value.
type progressValueKey struct{}

// progressStartKey is the Local holding an indeterminate indicator's start time.
type progressStartKey struct{}

// progressPathKey is the Local holding an indicator's reused paths.
type progressPathKey struct{}

// progressPaths are the paths an indicator rebuilds each frame, reusing their
// storage rather than allocating.
type progressPaths struct {
	active ui.Path
	track  ui.Path
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

// LinearProgress builds a Material Design 3 linear determinate progress
// indicator (value in 0..1) or an indeterminate one (value < 0): a rounded
// active segment, a clear track gap, an unfinished track and a stop indicator
// dot at the far end of the track. Right to left, it fills from the right and
// the dot sits at the left end. Determinate changes animate with the Default
// Spatial spring; ReduceMotion settles at once and an indeterminate indicator
// shows a still segment without asking for frames.
func LinearProgress(c *ui.Context, value float64, opts ...ProgressOptions) ui.Element {
	return progressIndicator(c, progressLinear, value, opts...)
}

// LinearWavyProgress builds the wavy variant of LinearProgress: the active
// segment is a sine wave whose amplitude smooths to zero at its ends and near
// 0% and 100%. An indeterminate indicator drifts its band and phase across the
// track.
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
	th := CurrentTheme(c)
	cs := th.Colors

	thickness := o.Thickness
	if !(thickness > 0) || math.IsNaN(float64(thickness)) || math.IsInf(float64(thickness), 0) {
		thickness = progressDefaultThick
	}
	gap := o.TrackGap
	if !(gap > 0) || math.IsNaN(float64(gap)) || math.IsInf(float64(gap), 0) {
		gap = progressDefaultGap
	}

	active := cs.Primary
	track := cs.SurfaceContainerHighest
	if track.A == 0 {
		track = cs.SurfaceVariant
	}
	if o.Disabled {
		active = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		track = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
	}

	var el ui.Element
	linear := kind == progressLinear || kind == progressLinearWavy
	if linear {
		el = ui.Box(c).Height(linearProgressRowHeight).Shrink(0).DimDisabled(false)
		if o.Width > 0 && !math.IsNaN(float64(o.Width)) && !math.IsInf(float64(o.Width), 0) {
			el.Width(o.Width)
		} else {
			el.FillWidth()
		}
		thickness = clampF32(thickness, 2, linearProgressRowHeight-6)
		gap = clampF32(gap, 1, 16)
	} else {
		size := o.Size
		if !(size > 0) || math.IsNaN(float64(size)) || math.IsInf(float64(size), 0) {
			size = progressDefaultCircular
		}
		el = ui.Box(c).Size(size, size).Shrink(0).DimDisabled(false)
		thickness = clampF32(thickness, 2, size/3)
		gap = clampF32(gap, 1, size/6)
	}
	el.Role(ui.RoleProgress).Disabled(o.Disabled)
	if o.Label != "" {
		el.Label(o.Label)
	}
	if indeterminate {
		// Progress of unknown length: a negative value, as ui.Progress reports.
		el.Range(0, 1, -1)
	} else {
		el.Range(0, 1, raw)
	}

	shown := float32(raw)
	if !indeterminate {
		shown = animateValue(el, progressValueKey{}, float32(raw), th.Animation(motion.DefaultSpatial), th.ReduceMotion)
	}
	start := ui.Local(el, progressStartKey{}, func() time.Time { return c.Now() })
	paths := ui.Local(el, progressPathKey{}, func() progressPaths { return progressPaths{} })

	rtl := c.Direction() == ui.RightToLeft
	reduceMotion := th.ReduceMotion
	wavy := kind == progressLinearWavy || kind == progressCircularWavy

	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if !el.Valid() || r.W <= 0 || r.H <= 0 {
			return
		}
		reduce := reduceMotion || p.Preferences().ReduceMotion
		elapsed := p.Now().Sub(*start)
		if reduce || indeterminate {
			// Keep an indeterminate band still under reduced motion; a
			// determinate value has already jumped to its target.
			if reduce {
				elapsed = 0
			}
		}
		p.Clip(r, 0, func() {
			if linear {
				paintLinearProgress(p, r, paths, linearProgressPaint{
					thickness:     thickness,
					gap:           gap,
					value:         clampF32(shown, 0, 1),
					indeterminate: indeterminate,
					wavy:          wavy,
					elapsed:       elapsed,
					rtl:           rtl,
					active:        active,
					track:         track,
				})
			} else {
				paintCircularProgress(p, r, paths, circularProgressPaint{
					thickness:     thickness,
					gap:           gap,
					value:         clampF32(shown, 0, 1),
					indeterminate: indeterminate,
					wavy:          wavy,
					elapsed:       elapsed,
					active:        active,
					track:         track,
				})
			}
		})
		if indeterminate && !reduce {
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
	indeterminate bool
	wavy          bool
	elapsed       time.Duration
	rtl           bool
	active        ui.Color
	track         ui.Color
}

// paintLinearProgress paints one linear progress frame into r.
func paintLinearProgress(p *ui.Painter, r ui.Rect, paths *progressPaths, s linearProgressPaint) {
	if paths == nil {
		return
	}
	cy := r.Y + r.H/2
	dotD := min(s.thickness, r.H)
	x0, x1 := r.X, r.X+r.W
	// mirror maps a logical offset from the start (0 at the low end) to a
	// screen x, filling from the right under RTL.
	mirror := func(logical float32) float32 {
		if s.rtl {
			return x1 - logical
		}
		return x0 + logical
	}
	span := func(a, b float32) (float32, float32) {
		xa, xb := mirror(a), mirror(b)
		return min(xa, xb), max(xa, xb)
	}

	// The unfinished track, clear of both the active segment and the dot.
	activeUsed := r.W * s.value
	trackStart := activeUsed + s.gap
	trackEnd := r.W - dotD - s.gap
	if s.indeterminate {
		trackStart = s.gap
	}
	if trackEnd > trackStart {
		ta, tb := span(trackStart, trackEnd)
		p.Fill(ui.Rect{X: ta, Y: cy - s.thickness/2, W: tb - ta, H: s.thickness}, s.track, s.thickness/2)
	}
	// The stop indicator dot at the far end.
	da, db := span(r.W-dotD, r.W)
	p.Fill(ui.Rect{X: da, Y: cy - dotD/2, W: db - da, H: dotD}, s.track, dotD/2)

	amplitude := min(s.thickness*0.75, max((r.H-s.thickness)/2, 0))
	if !s.wavy {
		amplitude = 0
	}
	if s.indeterminate {
		paintIndeterminateLinearWave(p, paths, s, cy, r.W, amplitude, mirror)
		return
	}
	// The amplitude converges to zero as the value nears either end, so a
	// short bar never pokes out of the track.
	amplitude *= progressEdgeScale(s.value)
	if !s.wavy || amplitude < 0.15 || activeUsed <= 0.5 {
		if activeUsed > 0.5 {
			aa, ab := span(0, activeUsed)
			p.Fill(ui.Rect{X: aa, Y: cy - s.thickness/2, W: ab - aa, H: s.thickness}, s.active, s.thickness/2)
		}
		return
	}
	path := progressWavePath(&paths.active, 0, activeUsed, cy, amplitude, 0, mirror)
	path.Cap(ui.ButtCap)
	p.StrokePath(path, s.thickness, s.active)
}

// paintIndeterminateLinearWave paints the drifting band of an indeterminate
// linear indicator.
func paintIndeterminateLinearWave(p *ui.Painter, paths *progressPaths, s linearProgressPaint, cy float32, width, amplitude float32, mirror func(float32) float32) {
	const band = 0.35
	cycle := float64(progressCycle)
	u := math.Mod(math.Max(float64(s.elapsed), 0), cycle) / cycle
	from := u*(1+band) - band*0.5
	to := from + band
	lo, hi := max(from, 0), min(to, 1)
	if hi <= lo {
		return
	}
	start, end := float32(lo)*width, float32(hi)*width
	if !s.wavy || amplitude < 0.15 {
		aa, ab := orderedSpan(mirror, start, end)
		p.Fill(ui.Rect{X: aa, Y: cy - s.thickness/2, W: ab - aa, H: s.thickness}, s.active, min(s.thickness/2, (ab-aa)/2))
		return
	}
	phase := float32(u * 4 * math.Pi)
	path := progressWavePath(&paths.active, start, end, cy, amplitude, phase, mirror)
	path.Cap(ui.ButtCap)
	p.StrokePath(path, s.thickness, s.active)
}

// orderedSpan maps logical a and b through mirror and returns the ordered
// screen span.
func orderedSpan(mirror func(float32) float32, a, b float32) (float32, float32) {
	xa, xb := mirror(a), mirror(b)
	return min(xa, xb), max(xa, xb)
}

// progressEdgeScale smooths a wave's amplitude to zero at 0 and 1.
func progressEdgeScale(v float32) float32 {
	edge := min(v, 1-v)
	t := clampF32(edge/0.12, 0, 1)
	return t * t * (3 - 2*t)
}

// progressWavePath builds a sine wave between logical from and to, centered on
// centerY, with its amplitude anchored at zero at both ends.
func progressWavePath(path *ui.Path, from, to, centerY, amplitude, phase float32, mirror func(float32) float32) *ui.Path {
	path.Reset()
	length := to - from
	if length <= 0 {
		return path
	}
	n := max(8, int(length/1.5))
	waves := length / progressWaveLength
	for i := 0; i <= n; i++ {
		f := float32(i) / float32(n)
		x := mirror(from + length*f)
		env := min(1, min(f, 1-f)/0.08)
		env = env * env * (3 - 2*env)
		y := centerY + amplitude*env*float32(math.Sin(float64(phase+waves*2*math.Pi*f)))
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	return path
}

// circularProgressPaint is the resolved snapshot a circular indicator draws.
type circularProgressPaint struct {
	thickness     float32
	gap           float32
	value         float32
	indeterminate bool
	wavy          bool
	elapsed       time.Duration
	active        ui.Color
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
	if s.wavy {
		amplitude = min(r.W*0.055, max(radius*0.4, 0))
		radius -= amplitude
	}
	if radius <= s.thickness*0.5 || radius <= 0 {
		return
	}
	// The gap is the angle a track gap plus the round cap's radius spans.
	gapDeg := (s.gap + s.thickness/2) / radius * (180 / math.Pi)

	if s.indeterminate {
		trackSweep := float32(360) - 2*gapDeg
		if trackSweep > 0 {
			progressArcPath(&paths.track, cx, cy, radius, gapDeg, trackSweep, amplitude, 0)
			paths.track.Cap(ui.RoundCap)
			p.StrokePath(&paths.track, s.thickness, s.track)
		}
		cycle := float64(progressCycle)
		u := math.Mod(math.Max(float64(s.elapsed), 0), cycle) / cycle
		rotation := float32(u * 540)
		sweep := 90 + 110*(0.5+0.5*float32(math.Sin(2*math.Pi*u)))
		phase := float32(0)
		if s.wavy {
			phase = float32(u * 4 * math.Pi)
		}
		progressArcPath(&paths.active, cx, cy, radius, rotation, sweep, amplitude, phase)
		paths.active.Cap(ui.RoundCap)
		p.StrokePath(&paths.active, s.thickness, s.active)
		return
	}

	// Determinate: the active arc first, then the rest of the ring, both with
	// a clear gap at the join.
	activeSweep := s.value * 360
	if activeSweep <= 0.01 {
		progressArcPath(&paths.track, cx, cy, radius, gapDeg, 360-2*gapDeg, amplitude, 0)
		paths.track.Cap(ui.RoundCap)
		p.StrokePath(&paths.track, s.thickness, s.track)
		return
	}
	if activeSweep >= 359.5 {
		progressArcPath(&paths.active, cx, cy, radius, 0, 360, amplitude, 0)
		paths.active.Cap(ui.RoundCap)
		p.StrokePath(&paths.active, s.thickness, s.active)
		return
	}
	a0 := gapDeg / 2
	a1 := activeSweep - gapDeg/2
	if a1 > a0 {
		progressArcPath(&paths.active, cx, cy, radius, a0, a1-a0, amplitude, 0)
		paths.active.Cap(ui.RoundCap)
		p.StrokePath(&paths.active, s.thickness, s.active)
	}
	rest := 360 - activeSweep - gapDeg
	if rest > 0.5 {
		progressArcPath(&paths.track, cx, cy, radius, a1+gapDeg, rest, amplitude, 0)
		paths.track.Cap(ui.RoundCap)
		p.StrokePath(&paths.track, s.thickness, s.track)
	}
}

// progressArcPath samples an arc from startDeg, clockwise from 12 o'clock, into
// path. A positive amplitude waves the radius as R + A·sin(6θ + phase), with
// the wave anchored at the arc's ends.
func progressArcPath(path *ui.Path, cx, cy, radius, startDeg, sweepDeg, amplitude, phase float32) *ui.Path {
	path.Reset()
	if sweepDeg <= 0 {
		return path
	}
	steps := max(8, int(sweepDeg/2))
	for i := 0; i <= steps; i++ {
		f := float32(i) / float32(steps)
		deg := startDeg + sweepDeg*f
		rad := radius
		if amplitude > 0 {
			env := min(1, min(f, 1-f)/0.08)
			env = env * env * (3 - 2*env)
			// Six waves around the full ring; θ spans the whole circle in the
			// formula so the wave keeps its shape as the arc grows.
			theta := float64(deg) * math.Pi / 180
			rad += amplitude * env * float32(math.Sin(6*theta+float64(phase)))
		}
		angle := float64(deg) * math.Pi / 180
		x := cx + rad*float32(math.Sin(angle))
		y := cy - rad*float32(math.Cos(angle))
		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}
	return path
}
