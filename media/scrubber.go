package media

import (
	"fmt"
	"math"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

type scrubberOptions struct {
	position   time.Duration
	duration   time.Duration
	buffered   float64
	playing    bool
	wavy       bool
	disabled   bool
	active     ui.Color
	buffer     ui.Color
	track      ui.Color
	thumb      ui.Color
	bubbleBg   ui.Color
	bubbleFg   ui.Color
	height     float32
	thumbH     float32
	showBubble bool
	label      string
	onSeek     func(pos time.Duration)
	onInteract func()
}

type (
	scrubberAnimKey      struct{}
	scrubberIndicatorKey struct{}

	scrubberState struct {
		waveStart  time.Time
		lastAt     time.Time
		lastPhase  float64
		lastFrac   float64
		lastBuf    float64
		lastThumbW float32
		path       *ui.Path
	}

	scrubberSnap struct {
		frac     float64
		buf      float64
		rtl      bool
		wavy     bool
		playing  bool
		disabled bool
		reduce   bool
		thumbH   float32
		active   ui.Color
		buffer   ui.Color
		track    ui.Color
		thumb    ui.Color
	}
)

const (
	scrubberPadX        = float32(8)
	scrubberTrackH      = float32(4)
	scrubberTrackGap    = float32(4)
	scrubberWaveLength  = float64(24)
	scrubberWavePeriod  = 1500 * time.Millisecond
	scrubberWaveAmp     = float64(2.8)
	scrubberDefaultH    = float32(32)
	scrubberThumbRestW  = float32(4)
	scrubberThumbDragW  = float32(2)
	scrubberDefaultThmH = float32(22)
)

func mediaScrubber(c *ui.Context, o scrubberOptions) ui.Element {
	th := md3e.CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion || c.Preferences().ReduceMotion
	rtl := c.Direction() == ui.RightToLeft

	durSec := o.duration.Seconds()
	hasDuration := !math.IsNaN(durSec) && !math.IsInf(durSec, 0) && durSec > 0
	maxSec := 1.0
	posSec := 0.0
	step := 0.1
	disabled := o.disabled || !hasDuration
	if hasDuration {
		maxSec = durSec
		posSec = clampF64(o.position.Seconds(), 0, maxSec)
		step = min(5.0, max(0.05, maxSec/20.0))
	}

	h := scrubberDefaultH
	if isPositiveFiniteF32(o.height) {
		h = o.height
	}
	thumbH := scrubberDefaultThmH
	if isPositiveFiniteF32(o.thumbH) {
		thumbH = min(o.thumbH, h)
	}

	activeCol := o.active
	if activeCol.A == 0 {
		activeCol = cs.Primary
	}
	bufCol := o.buffer
	if bufCol.A == 0 {
		bufCol = activeCol.Alpha(0.35)
	}
	trackCol := o.track
	if trackCol.A == 0 {
		trackCol = cs.SecondaryContainer
	}
	thumbCol := o.thumb
	if thumbCol.A == 0 {
		thumbCol = activeCol
	}
	if disabled {
		activeCol = cs.OnSurface.Alpha(0.38)
		bufCol = cs.OnSurface.Alpha(0.18)
		trackCol = cs.OnSurface.Alpha(0.12)
		thumbCol = cs.OnSurface.Alpha(0.38)
	}

	sl := ui.SliderBase(c, &posSec, 0, maxSec).
		Step(step).
		Disabled(disabled).
		Label(firstNonEmpty(o.label, "Playback timeline")).
		FocusRing(false).
		FillWidth().
		Height(h).
		PaddingX(scrubberPadX)

	if sl.Changed() {
		posSec = clampF64(posSec, 0, maxSec)
		if hasDuration && o.onSeek != nil {
			o.onSeek(time.Duration(math.Round(posSec * float64(time.Second))))
		}
		if o.onInteract != nil {
			o.onInteract()
		}
	}

	var engaged bool
	if !disabled {
		engaged = sl.Pressed() || sl.Focused()
	}

	frac := 0.0
	if hasDuration {
		frac = clampF64(posSec/maxSec, 0, 1)
	}
	bufFrac := clampF64(o.buffered, 0, 1)

	st := ui.Local(sl, scrubberAnimKey{}, func() scrubberState {
		return scrubberState{lastThumbW: scrubberThumbRestW}
	})
	snap := scrubberSnap{
		frac:     frac,
		buf:      bufFrac,
		rtl:      rtl,
		wavy:     o.wavy,
		playing:  o.playing,
		disabled: disabled,
		reduce:   reduce,
		thumbH:   thumbH,
		active:   activeCol,
		buffer:   bufCol,
		track:    trackCol,
		thumb:    thumbCol,
	}

	sl.Draw(func(p *ui.Painter, r ui.Rect) {
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) {
			return
		}
		trackLen := r.W - 2*scrubberPadX
		if trackLen <= 0 {
			return
		}
		x0 := r.X + scrubberPadX
		x1 := r.X + r.W - scrubberPadX
		cy := r.Y + r.H*0.5

		live := sl.Valid()
		vis := rectIntersects(r, p.Visible())
		red := snap.reduce || p.Preferences().ReduceMotion

		curFrac := st.lastFrac
		curBuf := st.lastBuf
		curPhase := st.lastPhase
		curThumbW := st.lastThumbW
		if curThumbW <= 0 {
			curThumbW = scrubberThumbRestW
		}

		if live {
			curFrac = snap.frac
			curBuf = snap.buf
			st.lastFrac = curFrac
			st.lastBuf = curBuf

			now := p.Now()
			targetW := scrubberThumbRestW
			if !snap.disabled && sl.Pressed() {
				targetW = scrubberThumbDragW
			}
			if red {
				curThumbW = targetW
				st.lastAt = now
			} else {
				dt := float64(0)
				if !st.lastAt.IsZero() && now.After(st.lastAt) {
					dt = now.Sub(st.lastAt).Seconds()
				}
				st.lastAt = now
				if dt > 0 {
					curThumbW += (targetW - curThumbW) * float32(1-math.Exp(-dt/0.04))
				}
				if math.Abs(float64(targetW-curThumbW)) <= 0.02 {
					curThumbW = targetW
				} else if vis {
					p.AnimationFrame()
				}
			}
			st.lastThumbW = curThumbW

			if snap.wavy && snap.playing && !snap.disabled && curFrac > 0 && !red {
				if st.waveStart.IsZero() {
					st.waveStart = now
				}
				elapsed := max(now.Sub(st.waveStart), 0)
				curPhase = (float64(elapsed%scrubberWavePeriod) / float64(scrubberWavePeriod)) * 2 * math.Pi
				st.lastPhase = curPhase
				if vis {
					p.AnimationFrame()
				}
			}
		}

		visualFrac := curFrac
		if snap.rtl {
			visualFrac = 1 - curFrac
		}
		tx := x0 + trackLen*float32(visualFrac)
		gap := curThumbW*0.5 + scrubberTrackGap

		// 1. Inactive base track ahead of thumb (and behind buffered portion if present).
		var aheadLo, aheadHi float32
		if snap.rtl {
			aheadLo = x0
			aheadHi = min(x1, tx-gap)
		} else {
			aheadLo = max(x0, tx+gap)
			aheadHi = x1
		}
		if aheadHi > aheadLo {
			if snap.buf > curFrac {
				bufEdge := x0 + trackLen*float32(snap.buf)
				if snap.rtl {
					bufEdge = x1 - trackLen*float32(snap.buf)
					bufLo := max(aheadLo, bufEdge)
					if aheadHi > bufLo {
						p.Fill(ui.Rect{X: bufLo, Y: cy - scrubberTrackH*0.5, W: aheadHi - bufLo, H: scrubberTrackH}, snap.buffer, scrubberTrackH*0.5)
					}
					if bufLo > aheadLo {
						p.Fill(ui.Rect{X: aheadLo, Y: cy - scrubberTrackH*0.5, W: bufLo - aheadLo, H: scrubberTrackH}, snap.track, scrubberTrackH*0.5)
					}
				} else {
					bufHi := min(aheadHi, bufEdge)
					if bufHi > aheadLo {
						p.Fill(ui.Rect{X: aheadLo, Y: cy - scrubberTrackH*0.5, W: bufHi - aheadLo, H: scrubberTrackH}, snap.buffer, scrubberTrackH*0.5)
					}
					if aheadHi > bufHi {
						p.Fill(ui.Rect{X: bufHi, Y: cy - scrubberTrackH*0.5, W: aheadHi - bufHi, H: scrubberTrackH}, snap.track, scrubberTrackH*0.5)
					}
				}
			} else {
				p.Fill(ui.Rect{X: aheadLo, Y: cy - scrubberTrackH*0.5, W: aheadHi - aheadLo, H: scrubberTrackH}, snap.track, scrubberTrackH*0.5)
			}
		}

		// 2. Active segment (wavy when playing, crisp capsule when paused/reduced motion).
		var actLo, actHi float32
		if snap.rtl {
			actLo = min(x1, tx+gap)
			actHi = x1
		} else {
			actLo = x0
			actHi = max(x0, tx-gap)
		}
		if actHi > actLo {
			drawWavy := snap.wavy && snap.playing && !snap.disabled && !red && (actHi-actLo) > 6
			if drawWavy {
				if st.path == nil {
					st.path = &ui.Path{}
				}
				buildWavyScrubPath(st.path, actLo, actHi, cy, curPhase)
				p.StrokePath(st.path, scrubberTrackH, snap.active)
			} else {
				p.Fill(ui.Rect{X: actLo, Y: cy - scrubberTrackH*0.5, W: actHi - actLo, H: scrubberTrackH}, snap.active, scrubberTrackH*0.5)
			}
		}

		// 3. Focus ring when keyboard focused (checked only while live).
		if live && sl.Focused() {
			ring := ui.Rect{
				X: tx - curThumbW*0.5 - 3,
				Y: cy - snap.thumbH*0.5 - 3,
				W: curThumbW + 6,
				H: snap.thumbH + 6,
			}
			p.Stroke(ring, snap.active.Alpha(0.6), ring.W*0.5, 1.5)
		}

		// 4. Vertical capsule handle.
		p.Fill(ui.Rect{
			X: tx - curThumbW*0.5,
			Y: cy - snap.thumbH*0.5,
			W: curThumbW,
			H: snap.thumbH,
		}, snap.thumb, curThumbW*0.5)
	})

	if o.showBubble && hasDuration {
		showVal := float32(0)
		if engaged {
			showVal = 1
		}
		indicator := animateF32(sl, scrubberIndicatorKey{}, showVal, th.Animation(motion.FastSpatial), reduce)
		if indicator > 0.02 {
			if b := sl.Bounds(); b.W > 2*scrubberPadX {
				f := float32(frac)
				if rtl {
					f = 1 - f
				}
				tx := b.X + scrubberPadX + (b.W-2*scrubberPadX)*f
				bg := o.bubbleBg
				if bg.A == 0 {
					bg = cs.InverseSurface
				}
				fg := o.bubbleFg
				if fg.A == 0 {
					fg = cs.InverseOnSurface
				}
				showTimeBubble(c, th, formatMediaTime(time.Duration(math.Round(posSec*float64(time.Second)))), tx, b.Y, indicator, bg, fg)
			}
		}
	}

	return sl
}

func buildWavyScrubPath(path *ui.Path, x0, x1, cy float32, phase float64) {
	path.Reset()
	span := float64(x1 - x0)
	if span <= 0 {
		return
	}
	steps := max(12, int(math.Ceil(span/3)))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		px := x0 + float32(span*t)
		edgeDist := min(t*span, (1-t)*span)
		fade := clampF64(edgeDist/10.0, 0, 1)
		fade = fade * fade * (3 - 2*fade)
		py := cy + float32(scrubberWaveAmp*fade*math.Sin((float64(px)/scrubberWaveLength)*2*math.Pi-phase))
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
}

func showTimeBubble(c *ui.Context, th *md3e.Theme, text string, cx, anchorTop, progress float32, bg, fg ui.Color) {
	w := clampF32(20+float32(utf8.RuneCountInString(text))*7.5, 48, 120)
	h := float32(30)
	tail := float32(5)
	top := anchorTop - h - tail - 6
	if top < 4 {
		top = 4
	}
	left := cx - w*0.5
	winW, _ := c.Size()
	if winW > 0 {
		left = clampF32(left, 4, max(4, winW-w-4))
	}
	scale := 0.65 + 0.35*clampF32(progress, 0, 1.1)
	opacity := clampF32(progress, 0, 1)
	tipX := clampF32(cx-left, 8, w-8)

	ui.Overlay(c, func() {
		box := ui.Box(c).
			Absolute().
			Left(left).
			Top(top).
			Size(w, h+tail).
			Padding(0, 0, tail, 0).
			Center().
			PassThrough().
			Scale(scale, scale).
			Opacity(opacity)

		box.Draw(func(p *ui.Painter, r ui.Rect) {
			pillR := ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H - tail}
			rad := pillR.H * 0.5
			p.Fill(pillR, bg, rad)
			if tail > 0 {
				var tri ui.Path
				tx := r.X + tipX
				by := r.Y + pillR.H
				tri.MoveTo(tx-tail, by-1)
				tri.LineTo(tx+tail, by-1)
				tri.LineTo(tx, by+tail)
				tri.Close()
				p.FillPath(&tri, bg)
			}
		})

		box.Children(func() {
			md3e.TextWith(c, text, md3e.TextOptions{
				Role:        tokens.LabelMedium,
				Emphasized:  true,
				TabularNums: true,
				Color:       fg,
			})
		})
	})
}

func formatMediaTime(d time.Duration) string {
	if d <= 0 {
		return "00:00"
	}
	sec := int(d.Seconds())
	if sec < 0 {
		sec = 0
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func rectIntersects(a, b ui.Rect) bool {
	if b.W <= 0 || b.H <= 0 {
		return true
	}
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}
