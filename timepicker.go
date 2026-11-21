package md3e

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// TimePickerMode selects the body of a TimePicker: a clock dial, or hour and
// minute fields with steppers.
type TimePickerMode int

const (
	// TimePickerDial shows the Material clock dial, with the hours and minutes
	// above it and an AM/PM switch in a 12-hour cycle.
	TimePickerDial TimePickerMode = iota
	// TimePickerInput shows hour and minute text fields with steppers.
	TimePickerInput
)

// TimePickerOptions configures TimePicker. The zero value is an enabled clock
// dial in a 12-hour cycle.
type TimePickerOptions struct {
	// Mode selects the initial body; the user can switch it with the header's
	// mode button.
	Mode TimePickerMode
	// HourCycle is 12 for a 12-hour cycle with AM and PM, or 24 for a 24-hour
	// one; any other value means 12.
	HourCycle int
	// Label names the picker for assistive technology.
	Label string
	// Disabled disables the whole picker: it takes no pointer or keyboard
	// input and paints MD3E's disabled tokens.
	Disabled bool
	// OnChanged runs after the user picks a time.
	OnChanged func()
}

// Time picker metrics in DIP.
const (
	timePickerWidth = 328
	timePickerPad   = 16
	timeDialSize    = 256
	timeDialOuterR  = 102
	timeDialInnerR  = 64
	timeDialCircleR = 16
	timeDialBlindR  = 14
	timeFieldW      = 88
	timeStepW       = 26
	timeStepH       = 24
)

// timeSegment is the half of the time the dial edits.
type timeSegment int

const (
	timeSegmentHour timeSegment = iota
	timeSegmentMinute
)

// Locals, channels and state of a time picker.
type (
	timePickerStateKey struct{}
	timeDialAngleKey   struct{}
	timeDialRadiusKey  struct{}

	timePickerState struct {
		ready    bool
		mode     TimePickerMode
		segment  timeSegment
		handInit bool
		handPrev float32

		hourText, minuteText string
		hourSync, minuteSync int
		hour12Sync           bool
		hourErr, minuteErr   bool

		pendingChange bool

		// A press on the dial, tracked from its input so a simple click is a
		// press and release even when both land in one frame.
		held           bool
		pressX, pressY float32
		picked         bool
		releaseAdvance bool
	}
)

// TimePicker builds a Material Design 3 Expressive time picker over *value:
// a clock dial with a Primary hand and a circle at the chosen tick, whose
// hours and minutes sit above it as tabs, with an AM/PM switch in a 12-hour
// cycle. Pressing or dragging anywhere on the dial snaps to the angle's hour
// or minute, the 24-hour dial telling its outer 00, 13–23 ring from the inner
// 1–12 one by radius, and releasing after choosing an hour springs the hand
// over to the minute dial. The header's mode button swaps the dial for hour
// and minute fields with steppers, each validating its range and syncing with
// the dial; arrow keys step the dial's active segment, Page Up and Page Down
// step five, and Enter or Space moves from the hours to the minutes. Value
// keeps its date and location.
//
//	md3e.TimePicker(c, &app.alarm, md3e.TimePickerOptions{Label: "Alarm"})
func TimePicker(c *ui.Context, value *time.Time, opts ...TimePickerOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: TimePicker takes at most one TimePickerOptions")
	}
	if value == nil {
		panic("md3e: TimePicker requires a non-nil *time.Time")
	}
	var o TimePickerOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	o.Mode = sanitizeTimeMode(o.Mode)
	hour12 := o.HourCycle != 24

	th := CurrentTheme(c)
	cs := th.Colors
	reduce := th.ReduceMotion
	spatial := th.Animation(motion.FastSpatial)
	now := c.Now()

	_, pickerW := timeFieldMetrics(th)
	outer := ui.Column(c).Width(pickerW).Padding(timePickerPad).Gap(12).
		DimDisabled(false).Disabled(o.Disabled)
	st := ui.Local(outer, timePickerStateKey{}, func() timePickerState {
		return timePickerState{mode: o.Mode, segment: timeSegmentHour}
	})
	disabled := o.Disabled || outer.IsDisabled()
	if disabled {
		// A disable in the middle of a drag must not leave the dial stuck as if
		// the pointer were still held down.
		st.held, st.picked, st.releaseAdvance = false, false, false
	}

	h, m := value.Hour(), value.Minute()
	if value.IsZero() {
		h, m = now.Hour(), now.Minute()
	}
	if !st.ready {
		st.ready = true
		st.mode = o.Mode
		st.segment = timeSegmentHour
		st.syncTexts(h, m, hour12)
	}
	// A change from outside, the dial, the hour cycle, or the fields resets
	// buffers that did not produce it; a buffer the user is editing keeps its text.
	if h != st.hourSync || m != st.minuteSync || hour12 != st.hour12Sync {
		st.syncTexts(h, m, hour12)
	}
	if st.releaseAdvance {
		st.releaseAdvance = false
		if st.segment == timeSegmentHour {
			st.segment = timeSegmentMinute
		}
	}

	setClock := func(hh, mm int) {
		if setClockTime(value, hh, mm, now) {
			st.pendingChange = true
		}
	}
	setPeriod := func(pm bool) {
		hh := h % 12
		if pm {
			hh += 12
		}
		setClock(hh, m)
	}

	outer.Children(func() {
		// Header: the optional name and the mode switch.
		head := ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(4)
		head.Children(func() {
			if o.Label != "" {
				name := ui.Text(c, o.Label).NoWrap().Grow(1).TextColor(cs.OnSurface)
				applyTypeStyleWithTheme(th, name, tokens.TitleSmall)
			} else {
				ui.Box(c).Grow(1)
			}
			if st.mode == TimePickerDial {
				if IconButton(c, "Switch to input", mdicon.Edit.Draw, IconButtonOptions{
					Size: tokens.SizeXS, Disabled: disabled,
				}).Clicked() {
					st.syncTexts(h, m, hour12)
					st.mode = TimePickerInput
				}
			} else if IconButton(c, "Switch to dial", mdicon.Schedule.Draw, IconButtonOptions{
				Size: tokens.SizeXS, Disabled: disabled,
			}).Clicked() {
				st.mode = TimePickerDial
			}
		})

		// The hours and minutes shown as tabs, with the half of the day.
		display := ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(6)
		display.Children(func() {
			// The hours and minutes read left to right as Hour : Minute even on a
			// right-to-left page, as ui.TimeInput does.
			tabs := ui.Row(c).Role(ui.RoleTabList).Direction(ui.LeftToRight).AlignItems(ui.Center).Gap(0)
			tabs.Children(func() {
				timeTab(c, th, formatHour(h, hour12), "Hour", st.segment == timeSegmentHour, disabled, func() {
					st.segment = timeSegmentHour
				})
				colon := ui.Text(c, ":").NoWrap().TextColor(cs.OnSurfaceVariant)
				applyTypeStyleWithTheme(th, colon, tokens.DisplayMedium)
				timeTab(c, th, fmt.Sprintf("%02d", m), "Minute", st.segment == timeSegmentMinute, disabled, func() {
					st.segment = timeSegmentMinute
				})
			})
			if hour12 {
				period := 0
				if h >= 12 {
					period = 1
				}
				seg := ui.SegmentedBase(c, &period, 2)
				seg.Track.Padding(2).Radius(18).Background(cs.SurfaceContainerHighest).Disabled(disabled)
				labels := [2]string{"AM", "PM"}
				for i := range 2 {
					item := seg.Segment(i).Padding(4, 10).Radius(16).
						Label(labels[i]).Disabled(disabled)
					ink := cs.OnSurfaceVariant
					if i == period {
						item.Background(cs.Primary)
						ink = cs.OnPrimary
					}
					item.TextColor(ink)
					idx := i
					item.Children(func() {
						txt := ui.Text(c, labels[idx]).NoWrap().TextColor(ink)
						applyTypeStyleWithTheme(th, txt, tokens.LabelMedium)
					})
				}
				if seg.Track.Changed() {
					setPeriod(period == 1)
				}
			}
		})

		if st.mode == TimePickerInput {
			buildTimeInputs(c, th, st, value, disabled, hour12, h, m, setClock)
		} else {
			buildTimeDial(c, th, st, value, disabled, hour12, reduce, spatial, h, m, setClock)
		}
	})

	if st.pendingChange {
		st.pendingChange = false
		if o.OnChanged != nil {
			o.OnChanged()
		}
	}
	return outer
}

// syncTexts resets the input buffers to h and m.
func (st *timePickerState) syncTexts(h, m int, hour12 bool) {
	st.hourText = formatHour(h, hour12)
	st.minuteText = fmt.Sprintf("%02d", m)
	st.hourSync, st.minuteSync, st.hour12Sync = h, m, hour12
	st.hourErr, st.minuteErr = false, false
}

func timeFieldMetrics(th *Theme) (fieldW, pickerW float32) {
	scale := max(float32(1), th.EffectiveTextScale())
	fieldW = float32(timeFieldW) * scale
	colonStyle := typeStyleOfWithTheme(th, tokens.TitleLarge)
	minRowW := 2*fieldW + 2*timeStepW + 8*4 + colonStyle.Size + 12
	pickerW = max(float32(timePickerWidth), 2*timePickerPad+minRowW)
	return fieldW, pickerW
}

// sanitizeTimeMode maps an unknown mode to the dial.
func sanitizeTimeMode(m TimePickerMode) TimePickerMode {
	if m == TimePickerInput {
		return TimePickerInput
	}
	return TimePickerDial
}

// timeTab builds one of the hours and minutes above the dial: a button that
// chooses the segment, filled in Primary with OnPrimary text while active, as
// MD3E's time picker header has it.
func timeTab(c *ui.Context, th *Theme, text, name string, active, disabled bool, choose func()) ui.Element {
	cs := th.Colors
	radius := float32(th.Shapes.Radius(tokens.ShapeSmall))
	ink := cs.OnSurfaceVariant
	container := ui.Color{}
	if active {
		ink = cs.OnPrimary
		container = cs.Primary
	}
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		if active {
			container = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		}
	}
	tab := ui.ButtonBase(c).DimDisabled(false).FocusRing(false).
		Padding(2, 8).Radius(radius).
		Role(ui.RoleTab).Label(name).AccessSelected(active).Disabled(disabled).
		TextColor(ink)
	if container.A > 0 {
		tab.Background(container)
	}
	attachRippleWithTheme(th, tab, StateLayerOptions{
		Color:    ink,
		Radius:   radius,
		Disabled: disabled,
	})
	tab.Children(func() {
		txt := ui.Text(c, text).NoWrap().TextColor(ink)
		applyTypeStyleWithTheme(th, txt, tokens.DisplayMedium)
	})
	if tab.Clicked() {
		choose()
	}
	return tab
}

// buildTimeDial builds the clock dial and its input, animation and keys.
func buildTimeDial(c *ui.Context, th *Theme, st *timePickerState, value *time.Time, disabled, hour12, reduce bool, spatial ui.Animation, h, m int, setClock func(h, m int)) {
	cs := th.Colors

	var dial ui.Element
	center := ui.Row(c).FillWidth().Justify(ui.Center)
	center.Children(func() {
		dial = ui.Box(c).Size(timeDialSize, timeDialSize).Shrink(0).
			Focusable().FocusRing(false).
			Role(ui.RoleGroup).Label("Clock dial").
			Value(fmt.Sprintf("%02d:%02d", h, m)).
			Disabled(disabled).DimDisabled(false)
	})

	pickAt := func(x, y float32) {
		v, ok := dialPick(x, y, st.segment, hour12)
		if !ok {
			return
		}
		switch {
		case st.segment == timeSegmentMinute:
			// Keep the displayed hour, which is now's for a zero value, so a
			// minute pick does not drop the half of the day.
			setClock(h, v)
		case hour12:
			// Keep the half of the day the shown hour is in: 12 on the dial is
			// 12 AM at midnight and 12 PM at noon.
			setClock(h/12*12+v%12, m)
		default:
			setClock(v, m)
		}
		st.picked = true
	}

	// HandleInput, not ObserveInput, so the dial keeps the moves and the
	// release wherever the pointer goes: a drag released off the rim still
	// ends, clears held, and advances to the minutes.
	dial.HandleInput(func(ev ui.InputEvent) bool {
		if disabled {
			return false
		}
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			st.held = true
			st.picked = false
			st.pressX, st.pressY = ev.X, ev.Y
			pickAt(ev.X, ev.Y)
			return true
		case ui.InputPointerMove:
			if !st.held || ev.Button != 0 {
				return false
			}
			st.pressX, st.pressY = ev.X, ev.Y
			pickAt(ev.X, ev.Y)
			return true
		case ui.InputPointerUp:
			if !st.held {
				return false
			}
			st.held = false
			if ev.Canceled {
				return true
			}
			st.pressX, st.pressY = ev.X, ev.Y
			pickAt(ev.X, ev.Y)
			if st.picked && st.segment == timeSegmentHour {
				st.releaseAdvance = true
			}
			return true
		}
		return false
	})

	step := 0
	switch {
	case dial.Shortcut(0, ui.KeyUp), dial.Shortcut(0, ui.KeyRight):
		step = 1
	case dial.Shortcut(0, ui.KeyDown), dial.Shortcut(0, ui.KeyLeft):
		step = -1
	case dial.Shortcut(0, ui.KeyPageUp):
		step = 5
	case dial.Shortcut(0, ui.KeyPageDown):
		step = -5
	case dial.Shortcut(0, ui.KeyEnter), dial.Shortcut(0, ui.KeySpace):
		if !disabled {
			if st.segment == timeSegmentHour {
				st.segment = timeSegmentMinute
			} else {
				st.segment = timeSegmentHour
			}
		}
	}
	if step != 0 && !disabled {
		if st.segment == timeSegmentHour {
			if hour12 {
				half := h / 12 * 12
				setClock(half+(h%12+step+12)%12, m)
			} else {
				setClock((h+step+24)%24, m)
			}
		} else {
			setClock(h, (m+step+60)%60)
		}
	}

	targetAngle, targetRadius := dialTarget(h, m, st.segment, hour12)
	var angle, radius float32
	switch {
	case !st.handInit:
		st.handInit = true
		angle = dial.Jump(timeDialAngleKey{}, targetAngle)
		radius = dial.Jump(timeDialRadiusKey{}, targetRadius)
		st.handPrev = targetAngle
	case st.held:
		if _, ok := dialPick(st.pressX, st.pressY, st.segment, hour12); ok {
			// The hand follows the pointer's angle but rides the ring's real
			// radius, so pulling off the rim never drags it toward the center,
			// and handPrev moves with it so the release takes the short way.
			angle = dial.Jump(timeDialAngleKey{}, dialAngleOf(st.pressX, st.pressY))
			radius = dial.Jump(timeDialRadiusKey{}, dialSnapRadius(st.pressX, st.pressY, st.segment, hour12))
			st.handPrev = angle
		} else {
			// Inside the blind center, hold the chosen value instead of
			// spinning the hand around a point with no angle.
			angle = dial.Jump(timeDialAngleKey{}, targetAngle)
			radius = dial.Jump(timeDialRadiusKey{}, targetRadius)
			st.handPrev = targetAngle
		}
	case reduce:
		angle = dial.Jump(timeDialAngleKey{}, targetAngle)
		radius = dial.Jump(timeDialRadiusKey{}, targetRadius)
		st.handPrev = targetAngle
	default:
		unwrapped := unwrapDialAngle(targetAngle, st.handPrev)
		angle = dial.AnimateTo(timeDialAngleKey{}, unwrapped, spatial)
		radius = dial.AnimateTo(timeDialRadiusKey{}, targetRadius, spatial)
		st.handPrev = unwrapped
	}

	paint := dialPaint{
		hour:       h,
		minute:     m,
		segment:    st.segment,
		hour12:     hour12,
		angle:      angle,
		radius:     radius,
		background: cs.SurfaceContainerHighest,
		ink:        cs.OnSurfaceVariant,
		selectedBg: cs.Primary,
		onSelected: cs.OnPrimary,
		handInk:    cs.Primary,
		disabled:   disabled,
	}
	if paint.background.A == 0 {
		paint.background = SurfaceColorForElevation(th, tokens.Level2)
	}
	if disabled {
		paint.background = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		paint.ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		paint.selectedBg = cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity))
		paint.onSelected = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
		paint.handInk = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	dial.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		drawTimeDial(p, r, paint)
	})
	dial.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if !dial.Valid() || !dial.FocusVisible() || disabled || r.W <= 0 || r.H <= 0 {
			return
		}
		box := ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
		p.Stroke(box, cs.Primary, min(box.W, box.H)/2, 2)
	})
}

// buildTimeInputs builds the hour and minute fields with their steppers. A
// buffer that differs from the value is parsed as it is built, so a valid
// time applies at once and an invalid one shows its error on the next frame.
func buildTimeInputs(c *ui.Context, th *Theme, st *timePickerState, value *time.Time, disabled, hour12 bool, h, m int, setClock func(h, m int)) {
	cs := th.Colors
	// Hour : Minute keeps its left-to-right order on a right-to-left page. The
	// fields carry a 10 DIP top band and room for error text below, so the row
	// starts its children at the top and the colon and steppers take explicit
	// offsets onto the fields' 56 DIP boxes instead of centering on that
	// composite height.
	scale := max(float32(1), th.EffectiveTextScale())
	fieldW, _ := timeFieldMetrics(th)
	fieldH := float32(textFieldHeight) * scale
	colonStyle := typeStyleOfWithTheme(th, tokens.TitleLarge)
	colonTop := textFieldOuterTop*scale + (fieldH-colonStyle.LineHeight)/2 - colonStyle.Size*0.18
	stepTop := textFieldOuterTop*scale + (fieldH-float32(2*timeStepH+2))/2
	row := ui.Row(c).FillWidth().Direction(ui.LeftToRight).AlignItems(ui.Start).Gap(8)
	var hourField, minuteField ui.Element
	row.Children(func() {
		hourRange := "0–23"
		if hour12 {
			hourRange = "1–12"
		}
		hourField = TextField(c, &st.hourText, TextFieldOptions{
			Variant:      TextFieldOutlined,
			Label:        "Hour",
			Placeholder:  hourRange,
			Error:        st.hourErr,
			ErrorMessage: "Enter " + hourRange,
			Disabled:     disabled,
			MaxLength:    2,
		}).Width(fieldW)
		timeStepColumn(c, th, stepTop, disabled, "hour", func(delta int) {
			if hour12 {
				half := h / 12 * 12
				setClock(half+(h%12+delta+12)%12, m)
			} else {
				setClock((h+delta+24)%24, m)
			}
		})
		colon := ui.Text(c, ":").NoWrap().TextColor(cs.OnSurfaceVariant)
		applyTypeStyleWithTheme(th, colon, tokens.TitleLarge)
		// A colon's ink sits in the lower half of its line box, so its line box
		// starts that much higher to center the ink on the fields' text.
		colon.Margin(colonTop, 0, 0, 0)
		minuteField = TextField(c, &st.minuteText, TextFieldOptions{
			Variant:      TextFieldOutlined,
			Label:        "Minute",
			Placeholder:  "0–59",
			Error:        st.minuteErr,
			ErrorMessage: "Enter 0–59",
			Disabled:     disabled,
			MaxLength:    2,
		}).Width(fieldW)
		timeStepColumn(c, th, stepTop, disabled, "minute", func(delta int) {
			setClock(h, (m+delta+60)%60)
		})
	})

	wantHour, wantMinute := formatHour(h, hour12), fmt.Sprintf("%02d", m)
	if st.hourText == wantHour && st.minuteText == wantMinute {
		if st.hourErr || st.minuteErr {
			st.hourErr, st.minuteErr = false, false
			c.Invalidate()
		}
		return
	}
	beforeHour, beforeMinute := value.Hour(), value.Minute()
	applyInput := func(hh, mm int) {
		if hour12 {
			// The field's 1–12 belong to the displayed half of the day (h, not
			// a zero value's Hour()==0), so 12 AM is midnight and 12 PM noon.
			setClock(h/12*12+hh%12, mm)
			return
		}
		setClock(hh, mm)
	}
	// FocusWithin reaches the text input inside each field; the field's outer
	// box does not carry the focus itself.
	errChanged := st.applyTimeInputs(hour12, hourField.FocusWithin(), minuteField.FocusWithin(), applyInput)
	if errChanged || value.Hour() != beforeHour || value.Minute() != beforeMinute {
		c.Invalidate()
	}
}

// timeStepColumn builds the two small stepper buttons beside a field, top DIPs
// down the row onto the field's box.
func timeStepColumn(c *ui.Context, th *Theme, top float32, disabled bool, unit string, step func(delta int)) {
	cs := th.Colors
	ink := cs.OnSurfaceVariant
	if disabled {
		ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
	}
	col := ui.Column(c).Width(timeStepW).Gap(2).Margin(top, 0, 0, 0)
	col.Children(func() {
		up := timeStepButton(c, th, "Increase "+unit, mdicon.ChevronUp.Draw, ink, disabled)
		if up.Clicked() {
			step(1)
		}
		down := timeStepButton(c, th, "Decrease "+unit, mdicon.ChevronDown.Draw, ink, disabled)
		if down.Clicked() {
			step(-1)
		}
	})
}

// timeStepButton builds one stepper button.
func timeStepButton(c *ui.Context, th *Theme, name string, icon IconDraw, ink ui.Color, disabled bool) ui.Element {
	btn := ui.ButtonBase(c).DimDisabled(false).FocusRing(false).
		Size(timeStepW, timeStepH).Shrink(0).
		Role(ui.RoleButton).Label(name).Disabled(disabled).TextColor(ink)
	attachRippleWithTheme(th, btn, StateLayerOptions{
		Color:    ink,
		Radius:   float32(th.Shapes.Radius(tokens.ShapeSmall)),
		Disabled: disabled,
	})
	btn.Children(func() {
		drawIconBox(c, icon, 14, ink)
	})
	return btn
}

// applyTimeInputs parses the two fields and applies a valid pair, reporting
// whether an error state changed. An empty field being edited is not an error
// yet: clearing it with Backspace must not flash red, while an empty field the
// user left, or a non-empty invalid one, shows its error at once.
func (st *timePickerState) applyTimeInputs(hour12, hourFocused, minuteFocused bool, setClock func(h, m int)) (errChanged bool) {
	hs := strings.TrimSpace(st.hourText)
	ms := strings.TrimSpace(st.minuteText)
	hi, herr := parseClockNumber(hs)
	mi, merr := parseClockNumber(ms)
	lo, hiTop := 0, 23
	if hour12 {
		lo, hiTop = 1, 12
	}
	hourEmpty, minuteEmpty := hs == "", ms == ""
	beforeHourErr, beforeMinuteErr := st.hourErr, st.minuteErr
	st.hourErr = (!hourEmpty || !hourFocused) && (herr != nil || hi < lo || hi > hiTop)
	st.minuteErr = (!minuteEmpty || !minuteFocused) && (merr != nil || mi < 0 || mi > 59)
	errChanged = st.hourErr != beforeHourErr || st.minuteErr != beforeMinuteErr
	if st.hourErr || st.minuteErr || hourEmpty || minuteEmpty {
		return errChanged
	}
	setClock(hi, mi)
	return errChanged
}

// parseClockNumber parses one or two digits.
func parseClockNumber(s string) (int, error) {
	if s == "" || len(s) > 2 {
		return 0, fmt.Errorf("md3e: not a clock number")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("md3e: not a clock number")
		}
	}
	return strconv.Atoi(s)
}

// formatHour writes an hour for the display: 1–12 in a 12-hour cycle, 0–23
// otherwise.
func formatHour(h int, hour12 bool) string {
	if hour12 {
		h %= 12
		if h == 0 {
			return "12"
		}
		return strconv.Itoa(h)
	}
	return strconv.Itoa(h)
}

// dialPaint is one frame of the clock dial.
type dialPaint struct {
	hour       int
	minute     int
	segment    timeSegment
	hour12     bool
	angle      float32
	radius     float32
	background ui.Color
	ink        ui.Color
	selectedBg ui.Color
	onSelected ui.Color
	handInk    ui.Color
	disabled   bool
}

// drawTimeDial paints the dial's plate, hand, tick labels and selected circle
// in three passes, so the circle covers no digit and every digit it does cover
// is drawn again in its OnPrimary ink.
func drawTimeDial(p *ui.Painter, r ui.Rect, d dialPaint) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	cx := r.X + r.W/2
	cy := r.Y + r.H/2
	plateR := min(r.W, r.H)/2 - 2
	p.Fill(ui.Rect{X: cx - plateR, Y: cy - plateR, W: 2 * plateR, H: 2 * plateR}, d.background, plateR)

	labels := dialLabels(d.segment, d.hour12)
	selHour := d.hour
	if d.hour12 {
		selHour = d.hour % 12
		if selHour == 0 {
			selHour = 12
		}
	}
	selectedLabel := -1
	for _, label := range labels {
		if d.segment == timeSegmentHour && label.value == selHour ||
			d.segment == timeSegmentMinute && label.value == d.minute {
			selectedLabel = label.value
			break
		}
	}

	tipX, tipY := dialPoint(cx, cy, d.angle, d.radius)

	// Pass 1: the plate, every tick in ink, the hand from the center to the
	// tick, and the center dot. A ui.Path reaches any angle; Painter.Line
	// would only draw the horizontal and vertical cases.
	for _, label := range labels {
		x, y := dialPoint(cx, cy, label.angle, label.radius)
		paintCenteredSpan(p, x, y, ui.Span{Text: label.text, Size: 13, Weight: 500, Color: d.ink})
	}
	var hand ui.Path
	hand.MoveTo(cx, cy)
	hand.LineTo(tipX, tipY)
	hand.Cap(ui.RoundCap)
	p.StrokePath(&hand, 2, d.handInk)
	p.Fill(ui.Rect{X: cx - 4, Y: cy - 4, W: 8, H: 8}, d.handInk, 4)

	// Pass 2: the selection circle over the ticks it lands on.
	circle := float32(timeDialCircleR)
	circleBox := ui.Rect{X: tipX - circle, Y: tipY - circle, W: 2 * circle, H: 2 * circle}
	p.Fill(circleBox, d.selectedBg, circle)

	// Pass 3: inside the circle, redraw every covered tick in OnPrimary so a
	// "05" under the circle of a "06" stays readable, and mark a value that
	// falls between ticks with a 4 DIP dot.
	p.Clip(circleBox, circle, func() {
		if selectedLabel < 0 {
			p.Fill(ui.Rect{X: tipX - 2, Y: tipY - 2, W: 4, H: 4}, d.onSelected, 2)
		}
		for _, label := range labels {
			x, y := dialPoint(cx, cy, label.angle, label.radius)
			paintCenteredSpan(p, x, y, ui.Span{Text: label.text, Size: 13, Weight: 500, Color: d.onSelected})
		}
	})
}

// dialLabel is one number around the dial.
type dialLabel struct {
	text   string
	value  int
	angle  float32
	radius float32
}

var (
	dialLabelsMinute = []dialLabel{
		{text: "00", value: 0, angle: 0, radius: timeDialOuterR},
		{text: "05", value: 5, angle: 30, radius: timeDialOuterR},
		{text: "10", value: 10, angle: 60, radius: timeDialOuterR},
		{text: "15", value: 15, angle: 90, radius: timeDialOuterR},
		{text: "20", value: 20, angle: 120, radius: timeDialOuterR},
		{text: "25", value: 25, angle: 150, radius: timeDialOuterR},
		{text: "30", value: 30, angle: 180, radius: timeDialOuterR},
		{text: "35", value: 35, angle: 210, radius: timeDialOuterR},
		{text: "40", value: 40, angle: 240, radius: timeDialOuterR},
		{text: "45", value: 45, angle: 270, radius: timeDialOuterR},
		{text: "50", value: 50, angle: 300, radius: timeDialOuterR},
		{text: "55", value: 55, angle: 330, radius: timeDialOuterR},
	}
	dialLabels12Hour = []dialLabel{
		{text: "12", value: 12, angle: 0, radius: timeDialOuterR},
		{text: "1", value: 1, angle: 30, radius: timeDialOuterR},
		{text: "2", value: 2, angle: 60, radius: timeDialOuterR},
		{text: "3", value: 3, angle: 90, radius: timeDialOuterR},
		{text: "4", value: 4, angle: 120, radius: timeDialOuterR},
		{text: "5", value: 5, angle: 150, radius: timeDialOuterR},
		{text: "6", value: 6, angle: 180, radius: timeDialOuterR},
		{text: "7", value: 7, angle: 210, radius: timeDialOuterR},
		{text: "8", value: 8, angle: 240, radius: timeDialOuterR},
		{text: "9", value: 9, angle: 270, radius: timeDialOuterR},
		{text: "10", value: 10, angle: 300, radius: timeDialOuterR},
		{text: "11", value: 11, angle: 330, radius: timeDialOuterR},
	}
	dialLabels24Hour = []dialLabel{
		{text: "00", value: 0, angle: 0, radius: timeDialOuterR},
		{text: "13", value: 13, angle: 30, radius: timeDialOuterR},
		{text: "14", value: 14, angle: 60, radius: timeDialOuterR},
		{text: "15", value: 15, angle: 90, radius: timeDialOuterR},
		{text: "16", value: 16, angle: 120, radius: timeDialOuterR},
		{text: "17", value: 17, angle: 150, radius: timeDialOuterR},
		{text: "18", value: 18, angle: 180, radius: timeDialOuterR},
		{text: "19", value: 19, angle: 210, radius: timeDialOuterR},
		{text: "20", value: 20, angle: 240, radius: timeDialOuterR},
		{text: "21", value: 21, angle: 270, radius: timeDialOuterR},
		{text: "22", value: 22, angle: 300, radius: timeDialOuterR},
		{text: "23", value: 23, angle: 330, radius: timeDialOuterR},
		{text: "12", value: 12, angle: 0, radius: timeDialInnerR},
		{text: "1", value: 1, angle: 30, radius: timeDialInnerR},
		{text: "2", value: 2, angle: 60, radius: timeDialInnerR},
		{text: "3", value: 3, angle: 90, radius: timeDialInnerR},
		{text: "4", value: 4, angle: 120, radius: timeDialInnerR},
		{text: "5", value: 5, angle: 150, radius: timeDialInnerR},
		{text: "6", value: 6, angle: 180, radius: timeDialInnerR},
		{text: "7", value: 7, angle: 210, radius: timeDialInnerR},
		{text: "8", value: 8, angle: 240, radius: timeDialInnerR},
		{text: "9", value: 9, angle: 270, radius: timeDialInnerR},
		{text: "10", value: 10, angle: 300, radius: timeDialInnerR},
		{text: "11", value: 11, angle: 330, radius: timeDialInnerR},
	}
)

// dialLabels returns the numbers of the segment: hours 1–12, hours 00 and 13–23
// outside 1–12, or minutes every five.
func dialLabels(segment timeSegment, hour12 bool) []dialLabel {
	if segment == timeSegmentMinute {
		return dialLabelsMinute
	}
	if hour12 {
		return dialLabels12Hour
	}
	return dialLabels24Hour
}

// dialTarget returns the hand angle and radius of the chosen value.
func dialTarget(h, m int, segment timeSegment, hour12 bool) (angle, radius float32) {
	if segment == timeSegmentMinute {
		return normalizeDialAngle(float32(m) * 6), timeDialOuterR
	}
	if !hour12 {
		switch {
		case h == 0:
			return 0, timeDialOuterR
		case h >= 13:
			return normalizeDialAngle(float32((h - 12) * 30)), timeDialOuterR
		case h == 12:
			return 0, timeDialInnerR
		default:
			return normalizeDialAngle(float32(h) * 30), timeDialInnerR
		}
	}
	h12 := h % 12
	return normalizeDialAngle(float32(h12) * 30), timeDialOuterR
}

// dialPick snaps a point on the dial to the active segment's value: an hour
// from 1 to 12, an hour from 0 to 23, or a minute from 0 to 59.
func dialPick(x, y float32, segment timeSegment, hour12 bool) (int, bool) {
	angle := dialAngleOf(x, y)
	dist := dialDistanceOf(x, y)
	if dist < timeDialBlindR {
		return 0, false
	}
	idx := 0
	if segment == timeSegmentMinute {
		idx = int(math.Round(float64(angle)/6)) % 60
		return idx, true
	}
	idx = int(math.Round(float64(angle)/30)) % 12
	if hour12 {
		if idx == 0 {
			return 12, true
		}
		return idx, true
	}
	outer := dist >= (timeDialOuterR+timeDialInnerR)/2
	if outer {
		if idx == 0 {
			return 0, true
		}
		return 12 + idx, true
	}
	if idx == 0 {
		return 12, true
	}
	return idx, true
}

// dialSnapRadius returns the radius the hand rides at a point: the outer ring
// for the minutes and the 12-hour hours, and the picked inner or outer ring
// for the 24-hour hours.
func dialSnapRadius(x, y float32, segment timeSegment, hour12 bool) float32 {
	if segment == timeSegmentMinute || hour12 {
		return timeDialOuterR
	}
	if dialDistanceOf(x, y) >= (timeDialOuterR+timeDialInnerR)/2 {
		return timeDialOuterR
	}
	return timeDialInnerR
}

// dialAngleOf returns the angle of a point clockwise from the top.
func dialAngleOf(x, y float32) float32 {
	cx := float32(timeDialSize) / 2
	cy := float32(timeDialSize) / 2
	deg := math.Atan2(float64(x-cx), float64(cy-y)) * 180 / math.Pi
	return normalizeDialAngle(float32(deg))
}

// dialDistanceOf returns how far a point sits from the dial's center.
func dialDistanceOf(x, y float32) float32 {
	cx := float32(timeDialSize) / 2
	cy := float32(timeDialSize) / 2
	return clampF32(float32(math.Hypot(float64(x-cx), float64(y-cy))), 0, timeDialOuterR)
}

// dialPoint places a point at angle degrees clockwise from the top.
func dialPoint(cx, cy, angle, radius float32) (float32, float32) {
	rad := float64(angle) * math.Pi / 180
	return cx + radius*float32(math.Sin(rad)), cy - radius*float32(math.Cos(rad))
}

// normalizeDialAngle wraps deg into 0..360.
func normalizeDialAngle(deg float32) float32 {
	deg = float32(math.Mod(float64(deg), 360))
	if deg < 0 {
		deg += 360
	}
	return deg
}

// unwrapDialAngle returns target as close to prev as its equivalent angle
// allows, so the hand takes the short way round.
func unwrapDialAngle(target, prev float32) float32 {
	for target-prev > 180 {
		target -= 360
	}
	for target-prev < -180 {
		target += 360
	}
	return target
}

// setClockTime sets tm's time of day to h:m, keeping its date and location (or
// taking now's date for a zero tm), and reports whether it changed.
func setClockTime(tm *time.Time, h, m int, now time.Time) bool {
	if tm == nil {
		return false
	}
	h = clampInt(h, 0, 23)
	m = clampInt(m, 0, 59)
	y, mo, d := tm.Date()
	loc := tm.Location()
	if tm.IsZero() {
		y, mo, d = now.Date()
		loc = now.Location()
	}
	next := time.Date(y, mo, d, h, m, tm.Second(), 0, loc)
	if next.Equal(*tm) {
		return false
	}
	*tm = next
	return true
}

// clampInt limits v to lo..hi.
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
