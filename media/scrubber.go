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
		waveAmp    float32
		haloAlpha  float32
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
		if !isPositiveFiniteF32(r.W) || !isPositiveFiniteF32(r.H) || !rectIntersects(r, p.Visible()) {
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
		red := snap.reduce || p.Preferences().ReduceMotion

		curFrac := st.lastFrac
		curBuf := st.lastBuf
		curPhase := st.lastPhase
		curThumbW := st.lastThumbW
		curAmp := st.waveAmp
		curHalo := st.haloAlpha
		if curThumbW <= 0 {
			curThumbW = scrubberThumbRestW
		}

		if live {
			curFrac = snap.frac
			curBuf = snap.buf
			st.lastFrac = curFrac
			st.lastBuf = curBuf

			now := p.Now()
			dt := float64(0)
			if !st.lastAt.IsZero() && now.After(st.lastAt) {
				dt = now.Sub(st.lastAt).Seconds()
			}
			st.lastAt = now

			pressed := !snap.disabled && sl.Pressed()
			hovered := !snap.disabled && sl.Hovered()
			targetW := scrubberThumbRestW
			if pressed {
				targetW = scrubberThumbDragW
			}
			targetHalo := float32(0)
			if pressed {
				targetHalo = 0.20
			} else if hovered {
				targetHalo = 0.12
			}
			targetAmp := float32(0)
			if snap.wavy && snap.playing && !snap.disabled && curFrac > 0 && !red {
				targetAmp = 1
			}

			if red {
				curThumbW = targetW
				curHalo = targetHalo
				curAmp = targetAmp
			} else {
				needFrame := false
				if dt > 0 {
					curThumbW += (targetW - curThumbW) * float32(1-math.Exp(-dt/0.04))
					curHalo += (targetHalo - curHalo) * float32(1-math.Exp(-dt/0.05))
					curAmp += (targetAmp - curAmp) * float32(1-math.Exp(-dt/0.06))
				}
				if math.Abs(float64(targetW-curThumbW)) <= 0.02 {
					curThumbW = targetW
				} else {
					needFrame = true
				}
				if math.Abs(float64(targetHalo-curHalo)) <= 0.005 {
					curHalo = targetHalo
				} else {
					needFrame = true
				}
				if math.Abs(float64(targetAmp-curAmp)) <= 0.015 {
					curAmp = targetAmp
				} else {
					needFrame = true
				}
				if targetAmp > 0 {
					if st.waveStart.IsZero() {
						st.waveStart = now
					}
					elapsed := max(now.Sub(st.waveStart), 0)
					curPhase = (float64(elapsed%scrubberWavePeriod) / float64(scrubberWavePeriod)) * 2 * math.Pi
					st.lastPhase = curPhase
					needFrame = true
				}
				if needFrame {
					p.AnimationFrame()
				}
			}
			st.lastThumbW = curThumbW
			st.haloAlpha = curHalo
			st.waveAmp = curAmp
		}

		visualFrac := curFrac
		if snap.rtl {
			visualFrac = 1 - curFrac
		}
		tx := x0 + trackLen*float32(visualFrac)
		gap := curThumbW*0.5 + scrubberTrackGap

		// 1. Inactive base track painted continuously first (eliminates hourglass pinch at buffer edge).
		var aheadLo, aheadHi float32
		if snap.rtl {
			aheadLo = x0
			aheadHi = min(x1, tx-gap)
		} else {
			aheadLo = max(x0, tx+gap)
			aheadHi = x1
		}
		if aheadHi > aheadLo {
			p.Fill(ui.Rect{X: aheadLo, Y: cy - scrubberTrackH*0.5, W: aheadHi - aheadLo, H: scrubberTrackH}, snap.track, scrubberTrackH*0.5)
			if curBuf > curFrac {
				if snap.rtl {
					bufEdge := x1 - trackLen*float32(curBuf)
					bufLo := max(aheadLo, bufEdge)
					if aheadHi > bufLo {
						p.Fill(ui.Rect{X: bufLo, Y: cy - scrubberTrackH*0.5, W: aheadHi - bufLo, H: scrubberTrackH}, snap.buffer, scrubberTrackH*0.5)
					}
				} else {
					bufEdge := x0 + trackLen*float32(curBuf)
					bufHi := min(aheadHi, bufEdge)
					if bufHi > aheadLo {
						p.Fill(ui.Rect{X: aheadLo, Y: cy - scrubberTrackH*0.5, W: bufHi - aheadLo, H: scrubberTrackH}, snap.buffer, scrubberTrackH*0.5)
					}
				}
			}
			// Subtle MD3E trailing stop dot at track end.
			if aheadHi-aheadLo > 8 {
				dotX := aheadHi - 2.5
				if snap.rtl {
					dotX = aheadLo + 2.5
				}
				p.Fill(ui.Rect{X: dotX - 1.5, Y: cy - 1.5, W: 3, H: 3}, snap.active.Alpha(0.75), 1.5)
			}
		}

		// 2. Active segment (wavy with smooth amplitude damping on pause, crisp capsule otherwise).
		var actLo, actHi float32
		if snap.rtl {
			actLo = min(x1, tx+gap)
			actHi = x1
		} else {
			actLo = x0
			actHi = max(x0, tx-gap)
		}
		if actHi > actLo {
			if curAmp > 0.02 && !snap.disabled && !red && (actHi-actLo) > 6 {
				if st.path == nil {
					st.path = &ui.Path{}
				}
				buildWavyScrubPath(st.path, actLo, actHi, cy, curPhase, scrubberWaveAmp*float64(curAmp))
				p.StrokePath(st.path, scrubberTrackH, snap.active)
			} else {
				p.Fill(ui.Rect{X: actLo, Y: cy - scrubberTrackH*0.5, W: actHi - actLo, H: scrubberTrackH}, snap.active, scrubberTrackH*0.5)
			}
		}

		// 3. Soft hover/press halo + keyboard focus ring.
		if curHalo > 0.005 {
			haloR := snap.thumbH*0.5 + 4
			p.Fill(ui.Rect{X: tx - haloR, Y: cy - haloR, W: haloR * 2, H: haloR * 2}, snap.active.Alpha(curHalo), haloR)
		}
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

func buildWavyScrubPath(path *ui.Path, x0, x1, cy float32, phase, amp float64) {
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
		py := cy + float32(amp*fade*math.Sin((float64(px)/scrubberWaveLength)*2*math.Pi-phase))
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
}

type timeBubblePathKey struct{}

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
	rad := h * 0.5
	tipX := clampF32(cx-left, rad, max(rad, w-rad))

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
		tri := ui.Local(box, timeBubblePathKey{}, func() ui.Path { return ui.Path{} })
		rimCol := fg.Alpha(0.14)

		box.Draw(func(p *ui.Painter, r ui.Rect) {
			pillR := ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H - tail}
			pr := pillR.H * 0.5
			// Soft ambient + key elevation shadow behind pill.
			p.Fill(ui.Rect{X: pillR.X - 1, Y: pillR.Y + 2, W: pillR.W + 2, H: pillR.H + 1}, ui.RGBA(0, 0, 0, 0.14), pr+1)
			p.Fill(ui.Rect{X: pillR.X, Y: pillR.Y + 1, W: pillR.W, H: pillR.H}, ui.RGBA(0, 0, 0, 0.18), pr)
			p.Fill(pillR, bg, pr)
			p.Stroke(pillR, rimCol, pr, 1)
			if tail > 0 {
				tri.Reset()
				tx := r.X + tipX
				by := r.Y + pillR.H
				tri.MoveTo(tx-tail, by-1)
				tri.LineTo(tx+tail, by-1)
				tri.LineTo(tx, by+tail)
				tri.Close()
				p.FillPath(tri, bg)
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
