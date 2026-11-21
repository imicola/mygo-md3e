package md3e

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/imicola/mygo-md3e/internal/mdicon"
	"github.com/imicola/mygo-md3e/motion"
	"github.com/imicola/mygo-md3e/tokens"
)

// DatePickerMode selects how a DatePicker or DateRangePicker shows: docked
// inline in a card, or in a modal dialog over the window.
type DatePickerMode int

const (
	// DatePickerDocked draws the calendar inline: a SurfaceContainer card with
	// a month grid, a header and an optional text entry view.
	DatePickerDocked DatePickerMode = iota
	// DatePickerModal draws the calendar in a modal dialog over a scrim, with
	// Cancel and OK actions, spring scale and fade in and out, Escape and a
	// scrim press closing it, and the focus returning to the trigger.
	DatePickerModal
)

// DatePickerOptions configures DatePicker and DateRangePicker. The zero value
// is an enabled inline calendar whose weeks start on Sunday.
type DatePickerOptions struct {
	// Mode selects the docked or modal presentation.
	Mode DatePickerMode
	// Open owns the modal's open state and is required by DatePickerModal;
	// Escape, a scrim press, Cancel and OK clear it. Docked pickers ignore it.
	Open *bool
	// Label names the picker and its modal dialog for assistive technology;
	// "Choose date" or "Choose dates" by default.
	Label string
	// Disabled disables the whole picker: it takes no pointer or keyboard
	// input and paints MD3E's disabled tokens.
	Disabled bool
	// FirstWeekday is the first column of a week, Sunday by default and
	// Monday in many regions. Values outside Sunday..Saturday fall back to
	// Sunday.
	FirstWeekday time.Weekday
	// OnChanged runs after the user picks a date or finishes a valid edit.
	OnChanged func()
}

// Date picker metrics in DIP.
const (
	datePickerWidth  = 328
	datePickerPad    = 12
	dateCellWidth    = 40
	dateCellHeight   = 40
	dateWeekGap      = 2
	dateGridWidth    = dateCellWidth * 7
	dateWeekdayH     = 28
	dateHeaderH      = 40
	dateYearCellW    = 88
	dateYearCellH    = 40
	dateFieldGap     = 8
	dateYearColumns  = 3
	dateYearRows     = 4
	datePickerLabel  = "Choose date"
	dateRangeLabel   = "Choose dates"
	dateFormatLayout = "2006-01-02"
	dateFormatError  = "Use YYYY-MM-DD"
)

// dateView is the body a date picker shows.
type dateView int

const (
	dateViewCalendar dateView = iota
	dateViewYear
	dateViewInput
)

// Locals, channels and states of a date picker.
type (
	datePickerStateKey struct{}

	datePickerState struct {
		ready    bool
		cursor   time.Time
		lastDate time.Time
		lastFrom time.Time
		lastTo   time.Time

		view     dateView
		yearPage int

		// Single-date and range text entry, each buffer with the value it was
		// last synced to.
		input      string
		inputSync  time.Time
		inputErr   bool
		startInput string
		endInput   string
		startSync  time.Time
		endSync    time.Time
		startErr   bool
		endErr     bool

		// Transactional modal draft state, committed to caller pointers on OK
		// and discarded on Cancel, Escape, or scrim dismissal.
		wasOpen    bool
		opener     *ui.Handle
		draftValue time.Time
		draftStart time.Time
		draftEnd   time.Time

		// Cached accessible labels ("January 2, 2006") for the 42 day grid.
		cachedDayValid  bool
		cachedDayYear   int
		cachedDayMonth  time.Month
		cachedFirstDay  time.Weekday
		cachedDayLabels [42]string
	}
)

// datePickerModel is the selection a picker edits: one date, or a range whose
// zero endpoints mean "not chosen yet".
type datePickerModel struct {
	value *time.Time
	start *time.Time
	end   *time.Time
}

func (m datePickerModel) rangeMode() bool { return m.start != nil }

// DatePicker builds a Material Design 3 Expressive date picker over *value: a
// month calendar with six weeks, the adjacent months' days in gray, a ring on
// today, a Primary circle on the chosen day, month paging buttons, a title
// that opens a year grid, and a switch to a YYYY-MM-DD text field that
// validates as it changes and applies a valid date at once. Arrow keys move a
// cursor among the days, Page Up and Page Down page months, Home and End go to
// the month's first and last days, and Enter or Space chooses the cursor's day.
//
// With Mode DatePickerModal, Open is required: the calendar shows in a modal
// dialog with Cancel and OK actions, spring scale and fade, a scrim, Escape,
// focus locked inside, and the focus back on the trigger as it closes. A
// DatePicker keeps the time of day and the location of *value, as ui.Calendar
// does. Weeks start on FirstWeekday (Sunday by default), and the calendar
// mirrors right to left.
//
//	md3e.DatePicker(c, &app.day, md3e.DatePickerOptions{Label: "Due date"})
func DatePicker(c *ui.Context, value *time.Time, opts ...DatePickerOptions) ui.Element {
	if value == nil {
		panic("md3e: DatePicker requires a non-nil *time.Time")
	}
	return buildDatePicker(c, datePickerModel{value: value}, opts)
}

// DateRangePicker builds a Material Design 3 Expressive date range picker over
// *start and *end: the same month calendar and header as DatePicker, where a
// first click chooses the start, a second the end (swapping them when the end
// comes first), and a third starts a new range. The endpoints paint as Primary
// circles with OnPrimary text, and the days between them join in a continuous
// PrimaryContainer band whose row ends are rounded, across weeks and months.
// A zero *end means the range is still open; a zero *start means none is
// chosen. The text entry view edits both endpoints as YYYY-MM-DD fields.
//
//	md3e.DateRangePicker(c, &app.from, &app.to, md3e.DatePickerOptions{Label: "Trip"})
func DateRangePicker(c *ui.Context, start, end *time.Time, opts ...DatePickerOptions) ui.Element {
	if start == nil || end == nil {
		panic("md3e: DateRangePicker requires non-nil *time.Time endpoints")
	}
	return buildDatePicker(c, datePickerModel{start: start, end: end}, opts)
}

func buildDatePicker(c *ui.Context, m datePickerModel, opts []DatePickerOptions) ui.Element {
	if len(opts) > 1 {
		panic("md3e: DatePicker takes at most one DatePickerOptions")
	}
	var o DatePickerOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	modal := o.Mode == DatePickerModal
	if modal && o.Open == nil {
		panic("md3e: DatePickerModal requires DatePickerOptions.Open")
	}
	if o.FirstWeekday < time.Sunday || o.FirstWeekday > time.Saturday {
		o.FirstWeekday = time.Sunday
	}

	th := CurrentTheme(c)
	reduce := th.ReduceMotion

	now := c.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if !modal {
		// A docked picker keeps its state on the panel itself, so it adds no
		// zero-width sibling that would take the caller's key or a layout gap.
		panel := newDatePickerPanel(c, m, o)
		st := ui.Local(panel, datePickerStateKey{}, func() datePickerState {
			return datePickerState{view: dateViewCalendar}
		})
		st.syncSelection(m, now)
		fillDatePickerPanel(c, th, panel, st, m, o, today, nil)
		return panel
	}

	// A modal keeps its state on a zero-width anchor in the caller's tree: the
	// dialog itself lives in the window's overlay.
	anchor := ui.Box(c).Size(0, 0)
	st := ui.Local(anchor, datePickerStateKey{}, func() datePickerState { return datePickerState{view: dateViewCalendar} })
	// An ancestor that disables the trigger disables the dialog too, though the
	// dialog is no child of it.
	if anchor.IsDisabled() {
		o.Disabled = true
	}

	if inExitingOverlay(c) {
		if *o.Open {
			*o.Open = false
		}
		st.wasOpen = false
		st.opener = nil
		return anchor
	}

	if !*o.Open {
		if st.wasOpen {
			st.wasOpen = false
			if st.opener != nil {
				if el := c.Resolve(*st.opener); !el.Valid() || !el.IsDisabled() {
					st.opener.Focus(c)
				}
				st.opener = nil
			}
		}
		return anchor
	}

	draftM := datePickerModel{
		value: &st.draftValue,
		start: &st.draftStart,
		end:   &st.draftEnd,
	}
	if m.rangeMode() {
		draftM.value = nil
	} else {
		draftM.start, draftM.end = nil, nil
	}
	if !st.wasOpen {
		// Every opening snapshots the committed selection into an isolated
		// draft on the calendar view, clearing any stale text from an earlier
		// canceled session, and records the focused trigger for nested modal
		// focus restoration.
		st.wasOpen = true
		st.opener = focusedTriggerHandle(c)
		if m.rangeMode() {
			st.draftStart, st.draftEnd = *m.start, *m.end
			st.draftValue = time.Time{}
			st.lastFrom, st.lastTo = st.draftStart, st.draftEnd
		} else {
			st.draftValue = *m.value
			st.draftStart, st.draftEnd = time.Time{}, time.Time{}
			st.lastDate = st.draftValue
		}
		st.ready = true
		st.view = dateViewCalendar
		st.cursor = st.selectionAnchor(draftM, now)
		st.syncInputsFromSelection(draftM)
	} else {
		st.syncSelection(draftM, now)
	}
	draftOpts := o
	draftOpts.OnChanged = nil

	cs := th.Colors
	scrim := cs.Scrim
	if scrim.A == 0 {
		scrim = ui.RGB(0, 0, 0)
	}
	backTr := ui.ElementTransition{
		Duration: -1,
		Enter:    &ui.Motion{Opacity: 0},
		Exit:     &ui.Motion{Opacity: 0},
	}
	panelTr := ui.ElementTransition{
		Duration: -1,
		Enter:    &ui.Motion{Opacity: 0, Scale: 0.9},
		Exit:     &ui.Motion{Opacity: 0, Scale: 0.9},
	}
	if !reduce {
		anim := th.Animation(motion.FastSpatial)
		backTr.Duration, backTr.Animation = 0, &anim
		panelTr.Duration, panelTr.Animation = 0, &anim
	}
	ui.DialogBase(c, o.Open, func(back, panel ui.Element) {
		back.Background(scrim.Alpha(0.32)).Transition(backTr)
		panel.Transition(panelTr)
		styleDatePickerPanel(th, panel, m, o)
		onConfirm := func() {
			if o.Disabled {
				return
			}
			if st.view == dateViewInput {
				if !m.rangeMode() && st.inputErr || m.rangeMode() && (st.startErr || st.endErr) {
					return
				}
			}
			if m.rangeMode() {
				if !(*m.start).Equal(st.draftStart) || !(*m.end).Equal(st.draftEnd) {
					*m.start = st.draftStart
					*m.end = st.draftEnd
					if o.OnChanged != nil {
						o.OnChanged()
					}
				}
			} else if !(*m.value).Equal(st.draftValue) {
				*m.value = st.draftValue
				if o.OnChanged != nil {
					o.OnChanged()
				}
			}
			if o.Open != nil {
				*o.Open = false
			}
		}
		fillDatePickerPanel(c, th, panel, st, draftM, draftOpts, today, onConfirm)
	})
	return anchor
}

// syncSelection follows the edited value when the app changes it from outside:
// the cursor moves to the new date and its month, as ui.Calendar does, unless
// the user is only browsing another month. It also mirrors a range whose
// endpoints were swapped.
func (st *datePickerState) syncSelection(m datePickerModel, now time.Time) {
	if m.rangeMode() {
		if !st.ready {
			st.ready = true
			st.cursor = st.selectionAnchor(m, now)
			st.lastFrom, st.lastTo = *m.start, *m.end
			st.syncInputsFromSelection(m)
			return
		}
		if !sameCalendarDay(st.lastFrom, *m.start) || !sameCalendarDay(st.lastTo, *m.end) {
			st.lastFrom, st.lastTo = *m.start, *m.end
			if !m.start.IsZero() {
				st.cursor = *m.start
			}
			st.syncInputsFromSelection(m)
		}
		return
	}
	if !st.ready {
		st.ready = true
		st.cursor = st.selectionAnchor(m, now)
		st.lastDate = *m.value
		st.syncInputsFromSelection(m)
		return
	}
	if !sameCalendarDay(st.lastDate, *m.value) {
		st.lastDate = *m.value
		if !m.value.IsZero() {
			st.cursor = *m.value
		}
		st.syncInputsFromSelection(m)
	}
}

// selectionAnchor returns the day a picker shows when it opens: the chosen
// date, a range's start, or today when nothing is chosen.
func (st *datePickerState) selectionAnchor(m datePickerModel, now time.Time) time.Time {
	if m.rangeMode() {
		if !m.start.IsZero() {
			return *m.start
		}
		if !m.end.IsZero() {
			return *m.end
		}
		return now
	}
	if !m.value.IsZero() {
		return *m.value
	}
	return now
}

// syncInputsFromSelection resets every text buffer to the selection.
func (st *datePickerState) syncInputsFromSelection(m datePickerModel) {
	if m.rangeMode() {
		st.startInput = formatPickerDate(*m.start)
		st.endInput = formatPickerDate(*m.end)
		st.startSync, st.endSync = *m.start, *m.end
		st.startErr, st.endErr = false, false
		return
	}
	st.input = formatPickerDate(*m.value)
	st.inputSync = *m.value
	st.inputErr = false
}

// newDatePickerPanel builds the empty Surface card or dialog panel, before its
// state is bound or its content built.
func newDatePickerPanel(c *ui.Context, m datePickerModel, o DatePickerOptions) ui.Element {
	panel := Surface(c, SurfaceOptions{
		Elevation:  tokens.Level2,
		ShapeScale: tokens.ShapeExtraLarge,
	}).Role(ui.RoleGroup)
	styleDatePickerPanel(CurrentTheme(c), panel, m, o)
	return panel
}

func styleDatePickerPanel(th *Theme, panel ui.Element, m datePickerModel, o DatePickerOptions) {
	name := o.Label
	if name == "" {
		if m.rangeMode() {
			name = dateRangeLabel
		} else {
			name = datePickerLabel
		}
	}
	panel.Width(datePickerWidth).Padding(datePickerPad).Gap(dateFieldGap).
		Radius(th.Shapes.Radius(tokens.ShapeExtraLarge)).
		Background(SurfaceColorForElevation(th, tokens.Level2)).
		TextColor(th.Colors.OnSurface).
		DimDisabled(false).Disabled(o.Disabled).
		Label(name)
	ApplyElevationShadows(th, panel, tokens.Level2)
}

// fillDatePickerPanel adds the header, the calendar, year grid or text fields,
// and a modal's actions to the panel.
func fillDatePickerPanel(c *ui.Context, th *Theme, panel ui.Element, st *datePickerState, m datePickerModel, o DatePickerOptions, today time.Time, onConfirm func()) {
	rtl := c.Direction() == ui.RightToLeft
	disabled := o.Disabled || panel.IsDisabled()

	panel.Children(func() {
		buildDatePickerHeader(c, th, st, m, o, rtl, disabled)
		Divider(c)
		switch st.view {
		case dateViewYear:
			buildDateYearGrid(c, th, st, disabled)
		case dateViewInput:
			buildDateInputs(c, st, m, o, disabled)
		default:
			buildDateCalendar(c, th, st, m, o, today, rtl, disabled)
		}
		if o.Mode == DatePickerModal {
			buildDatePickerActions(c, o, onConfirm)
		}
	})
}

// buildDatePickerHeader builds the month paging row with the title that opens
// the year grid and the calendar/text mode switch, or the year and input
// headers.
func buildDatePickerHeader(c *ui.Context, th *Theme, st *datePickerState, m datePickerModel, o DatePickerOptions, rtl, disabled bool) {
	cs := th.Colors
	row := ui.Row(c).FillWidth().Height(dateHeaderH).AlignItems(ui.Center).Gap(2)
	row.Children(func() {
		if st.view == dateViewYear {
			if IconButton(c, "Previous decade", chevronIcon(true, rtl), IconButtonOptions{
				Size: tokens.SizeXS, Disabled: disabled,
			}).Clicked() {
				st.yearPage -= 12
			}
			title := ui.Text(c, strconv.Itoa(st.yearPage)+" – "+strconv.Itoa(st.yearPage+11)).Grow(1).
				TextAlign(ui.Center).NoWrap().TextColor(cs.OnSurface)
			applyTypeStyleWithTheme(th, title, tokens.TitleSmall)
			if IconButton(c, "Next decade", chevronIcon(false, rtl), IconButtonOptions{
				Size: tokens.SizeXS, Disabled: disabled,
			}).Clicked() {
				st.yearPage += 12
			}
			return
		}
		if st.view == dateViewInput {
			title := ui.Text(c, dateInputTitle(m)).Grow(1).NoWrap().TextColor(cs.OnSurface)
			applyTypeStyleWithTheme(th, title, tokens.TitleSmall)
			if IconButton(c, "Show calendar", mdicon.Calendar.Draw, IconButtonOptions{
				Size: tokens.SizeXS, Disabled: disabled,
			}).Clicked() {
				st.view = dateViewCalendar
			}
			return
		}

		if IconButton(c, "Previous month", chevronIcon(true, rtl), IconButtonOptions{
			Size: tokens.SizeXS, Disabled: disabled,
		}).Clicked() {
			st.moveMonth(-1)
		}
		title := ui.ButtonBase(c).
			DimDisabled(false).
			FocusRing(false).
			Height(dateHeaderH - 4).
			Grow(1).
			Role(ui.RoleButton).
			Label(st.monthTitle()).
			Disabled(disabled)
		attachRippleWithTheme(th, title, StateLayerOptions{
			Color:    cs.OnSurface,
			Radius:   float32(th.Shapes.Radius(tokens.ShapeSmall)),
			Disabled: disabled,
		})
		title.Children(func() {
			txt := ui.Text(c, st.monthTitle()).NoWrap().TextColor(cs.OnSurface)
			applyTypeStyleWithTheme(th, txt, tokens.TitleSmall)
		})
		if title.Clicked() {
			st.yearPage = st.cursor.Year() / 12 * 12
			st.view = dateViewYear
		}
		if IconButton(c, "Next month", chevronIcon(false, rtl), IconButtonOptions{
			Size: tokens.SizeXS, Disabled: disabled,
		}).Clicked() {
			st.moveMonth(1)
		}
		if IconButton(c, "Type date", mdicon.Edit.Draw, IconButtonOptions{
			Size: tokens.SizeXS, Disabled: disabled,
		}).Clicked() {
			st.syncInputsFromSelection(m)
			st.view = dateViewInput
		}
	})
}

// buildDateCalendar builds the weekday header and six weeks of day cells.
func buildDateCalendar(c *ui.Context, th *Theme, st *datePickerState, m datePickerModel, o DatePickerOptions, today time.Time, rtl, disabled bool) {
	cs := th.Colors
	month := time.Date(st.cursor.Year(), st.cursor.Month(), 1, 0, 0, 0, 0, st.cursor.Location())
	first := int(o.FirstWeekday)
	start := month.AddDate(0, 0, -((int(month.Weekday()) - first + 7) % 7))

	centered := ui.Row(c).FillWidth().Justify(ui.Center)
	centered.Children(func() {
		grid := ui.Column(c).
			Width(dateGridWidth).
			Gap(dateWeekGap).
			Focusable().
			FocusRing(false).
			Role(ui.RoleGrid).
			Label(st.monthTitle()).
			RowCount(7).
			ColumnCount(7).
			Disabled(o.Disabled).
			DimDisabled(false)

		var cursorCell ui.Element
		var clickedDay time.Time
		hasClick := false

		back, forth := ui.KeyLeft, ui.KeyRight
		if rtl {
			back, forth = forth, back
		}
		switch {
		case grid.Shortcut(0, back):
			st.cursor = st.cursor.AddDate(0, 0, -1)
		case grid.Shortcut(0, forth):
			st.cursor = st.cursor.AddDate(0, 0, 1)
		case grid.Shortcut(0, ui.KeyUp):
			st.cursor = st.cursor.AddDate(0, 0, -7)
		case grid.Shortcut(0, ui.KeyDown):
			st.cursor = st.cursor.AddDate(0, 0, 7)
		case grid.Shortcut(0, ui.KeyPageUp):
			st.moveMonth(-1)
		case grid.Shortcut(0, ui.KeyPageDown):
			st.moveMonth(1)
		case grid.Shortcut(0, ui.KeyHome):
			st.cursor = time.Date(st.cursor.Year(), st.cursor.Month(), 1, 0, 0, 0, 0, st.cursor.Location())
		case grid.Shortcut(0, ui.KeyEnd):
			st.cursor = time.Date(st.cursor.Year(), st.cursor.Month()+1, 0, 0, 0, 0, 0, st.cursor.Location())
		case grid.Shortcut(0, ui.KeyEnter), grid.Shortcut(0, ui.KeySpace):
			if !disabled {
				st.pick(m, o, st.cursor)
			}
		}

		names := pickerWeekdays(first)
		dayLabels := st.monthDayLabels(month, start, o.FirstWeekday)
		grid.Children(func() {
			header := ui.Row(c).Height(dateWeekdayH).Role(ui.RoleRow)
			header.Children(func() {
				for i := range 7 {
					name := names[i]
					head := ui.Box(c).Size(dateCellWidth, dateWeekdayH).Shrink(0).
						Role(ui.RoleColumnHeader).Label(name)
					head.Draw(func(p *ui.Painter, r ui.Rect) {
						if r.W <= 0 || r.H <= 0 {
							return
						}
						ink := cs.OnSurfaceVariant
						if disabled {
							ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
						}
						paintCenteredSpan(p, r.X+r.W/2, r.Y+r.H/2,
							ui.Span{Text: name, Size: 12, Weight: 500, Color: ink})
					})
				}
			})
			for w := range 6 {
				row := ui.Row(c).Height(dateCellHeight).Role(ui.RoleRow)
				row.Children(func() {
					for d := range 7 {
						day := start.AddDate(0, 0, w*7+d)
						inMonth := day.Month() == month.Month() && day.Year() == month.Year()
						chosen, inRange := datePickerDayState(m, day)
						visual := dateCellPaint{
							band:       inRange && !chosen,
							selected:   chosen,
							inMonth:    inMonth,
							today:      sameCalendarDay(day, today) && !chosen,
							bandStart:  inRange && !chosen && d == 0,
							bandEnd:    inRange && !chosen && d == 6,
							hoverInk:   cs.OnSurface,
							selectedBg: cs.Primary,
							onSelected: cs.OnPrimary,
							bandBg:     cs.PrimaryContainer,
							onBand:     cs.OnPrimaryContainer,
							ink:        cs.OnSurface,
							inkOutside: cs.OnSurfaceVariant,
							todayInk:   cs.Primary,
							disabled:   disabled,
							rtl:        rtl,
						}
						if visual.bandBg.A == 0 {
							visual.bandBg = cs.SecondaryContainer
							visual.onBand = cs.OnSecondaryContainer
						}

						cellIdx := w*7 + d
						cell := ui.Box(c).Size(dateCellWidth, dateCellHeight).Shrink(0).
							Role(ui.RoleCell).
							Label(dayLabels[cellIdx]).
							AccessSelected(chosen || inRange)
						if sameCalendarDay(day, today) {
							cell.Current(ui.CurrentDate)
						}
						drawDateCell(cell, visual, day.Day())
						if sameCalendarDay(day, st.cursor) {
							cursorCell = cell
						}
						if cell.Clicked() {
							clickedDay, hasClick = day, true
						}
					}
				})
			}
		})
		grid.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if !grid.Valid() || !grid.FocusVisible() || disabled || r.W <= 0 || r.H <= 0 {
				return
			}
			days := calendarDayNumber(st.cursor) - calendarDayNumber(start)
			if days < 0 || days >= 42 {
				return
			}
			col := days % 7
			if rtl {
				// The rows mirror, so the week's first column sits at the right.
				col = 6 - col
			}
			rowIdx := days / 7
			x := r.X + float32(col*dateCellWidth)
			y := r.Y + dateWeekdayH + dateWeekGap + float32(rowIdx*(dateCellHeight+dateWeekGap))
			box := ui.Rect{X: x + 1, Y: y + 1, W: dateCellWidth - 2, H: dateCellHeight - 2}
			p.Stroke(box, cs.Primary, box.W/2, 2)
		})
		if cursorCell.Valid() {
			grid.ActiveDescendant(cursorCell)
		}
		if hasClick && !disabled {
			st.pick(m, o, clickedDay)
			grid.Focus()
		}
	})
}

// dateCellPaint is one day cell's resolved colors and band edges.
type dateCellPaint struct {
	band       bool
	bandStart  bool
	bandEnd    bool
	selected   bool
	inMonth    bool
	today      bool
	hoverInk   ui.Color
	selectedBg ui.Color
	onSelected ui.Color
	bandBg     ui.Color
	onBand     ui.Color
	ink        ui.Color
	inkOutside ui.Color
	todayInk   ui.Color
	disabled   bool
	rtl        bool
}

// drawDateCell paints a day cell's range band, selection circle, state layer,
// today ring and day number.
func drawDateCell(cell ui.Element, v dateCellPaint, dayNum int) {
	cell.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W <= 0 || r.H <= 0 {
			return
		}
		live := cell.Valid()
		circle := ui.Rect{X: r.X + 4, Y: r.Y + 4, W: r.W - 8, H: r.H - 8}
		disabledInk := v.ink.Alpha(float32(tokens.DisabledContentOpacity))
		disabledContainer := v.ink.Alpha(float32(tokens.DisabledContainerOpacity))
		ink := v.ink
		if !v.inMonth {
			ink = v.inkOutside
		}
		if v.disabled {
			ink = disabledInk
		}
		switch {
		case v.selected && v.disabled:
			p.Fill(circle, disabledContainer, circle.W/2)
			ink = disabledInk
		case v.selected:
			p.Fill(circle, v.selectedBg, circle.W/2)
			ink = v.onSelected
		case v.band && v.disabled:
			p.FillRadii(r, dateBandRadii(v), disabledContainer)
			ink = disabledInk
		case v.band:
			p.FillRadii(r, dateBandRadii(v), v.bandBg)
			if v.inMonth {
				ink = v.onBand
			}
		}
		if live && !v.disabled && !v.selected && !v.band {
			if cell.Pressed() {
				p.Fill(circle, v.hoverInk.Alpha(float32(tokens.StatePressedOpacity)), circle.W/2)
			} else if cell.Hovered() {
				p.Fill(circle, v.hoverInk.Alpha(float32(tokens.StateHoverOpacity)), circle.W/2)
			}
		}
		if v.today {
			ring := ui.Rect{X: r.X + 2, Y: r.Y + 2, W: r.W - 4, H: r.H - 4}
			color := v.todayInk
			if v.disabled {
				color = disabledInk
			}
			p.Stroke(ring, color, ring.W/2, 1)
		}
		paintCenteredSpan(p, r.X+r.W/2, r.Y+r.H/2,
			ui.Span{Text: dateDayText(dayNum), Size: 13, Weight: 500, Color: ink})
	})
}

var dateDayTexts = [32]string{
	"", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
	"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
	"21", "22", "23", "24", "25", "26", "27", "28", "29", "30", "31",
}

func dateDayText(dayNum int) string {
	if dayNum >= 1 && dayNum < len(dateDayTexts) {
		return dateDayTexts[dayNum]
	}
	return strconv.Itoa(dayNum)
}

func (st *datePickerState) monthDayLabels(month, start time.Time, first time.Weekday) *[42]string {
	if st.cachedDayValid &&
		st.cachedDayYear == month.Year() &&
		st.cachedDayMonth == month.Month() &&
		st.cachedFirstDay == first {
		return &st.cachedDayLabels
	}
	st.cachedDayValid = true
	st.cachedDayYear = month.Year()
	st.cachedDayMonth = month.Month()
	st.cachedFirstDay = first
	for i := range st.cachedDayLabels {
		st.cachedDayLabels[i] = start.AddDate(0, 0, i).Format("January 2, 2006")
	}
	return &st.cachedDayLabels
}

// dateBandRadii rounds a range band's outer edge at the row's ends, on the
// visual start or end side, mirrored right to left.
func dateBandRadii(v dateCellPaint) [4]float32 {
	r := float32(dateCellHeight) / 2
	var radii [4]float32
	if v.bandStart {
		if v.rtl {
			radii[1], radii[2] = r, r
		} else {
			radii[0], radii[3] = r, r
		}
	}
	if v.bandEnd {
		if v.rtl {
			radii[0], radii[3] = r, r
		} else {
			radii[1], radii[2] = r, r
		}
	}
	return radii
}

// buildDateYearGrid builds the 12-year page that the title opens.
func buildDateYearGrid(c *ui.Context, th *Theme, st *datePickerState, disabled bool) {
	cs := th.Colors
	centered := ui.Row(c).FillWidth().Justify(ui.Center)
	centered.Children(func() {
		grid := ui.Column(c).Gap(4).Role(ui.RoleGrid).Label("Choose year").
			RowCount(dateYearRows).ColumnCount(dateYearColumns)
		grid.Children(func() {
			for rowIdx := range dateYearRows {
				row := ui.Row(c).Height(dateYearCellH).Gap(4).Role(ui.RoleRow)
				row.Children(func() {
					for col := range dateYearColumns {
						year := st.yearPage + rowIdx*dateYearColumns + col
						selected := year == st.cursor.Year()
						cell := ui.Box(c).Size(dateYearCellW, dateYearCellH).Shrink(0).
							Role(ui.RoleCell).
							Label(strconv.Itoa(year)).
							AccessSelected(selected).
							Disabled(disabled)
						cell.Draw(func(p *ui.Painter, r ui.Rect) {
							if r.W <= 0 || r.H <= 0 {
								return
							}
							live := cell.Valid()
							ink := cs.OnSurface
							switch {
							case selected && disabled:
								p.Fill(r, cs.OnSurface.Alpha(float32(tokens.DisabledContainerOpacity)),
									float32(th.Shapes.Radius(tokens.ShapeSmall)))
								ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
							case selected:
								p.Fill(r, cs.Primary, float32(th.Shapes.Radius(tokens.ShapeSmall)))
								ink = cs.OnPrimary
							case disabled:
								ink = cs.OnSurface.Alpha(float32(tokens.DisabledContentOpacity))
							case live && cell.Hovered():
								p.Fill(r, cs.OnSurface.Alpha(float32(tokens.StateHoverOpacity)),
									float32(th.Shapes.Radius(tokens.ShapeSmall)))
							}
							paintCenteredSpan(p, r.X+r.W/2, r.Y+r.H/2,
								ui.Span{Text: strconv.Itoa(year), Size: 14, Weight: 500, Color: ink})
						})
						if cell.Clicked() && !disabled {
							st.cursor = time.Date(year, st.cursor.Month(), 1,
								st.cursor.Hour(), st.cursor.Minute(), st.cursor.Second(), 0, st.cursor.Location())
							st.view = dateViewCalendar
						}
					}
				})
			}
		})
	})
}

// buildDateInputs builds the YYYY-MM-DD text entry, one field for a date and
// two for a range. A buffer that differs from the selection is parsed as it
// is built, so a valid date applies at once and an invalid one shows its
// error on the next frame.
func buildDateInputs(c *ui.Context, st *datePickerState, m datePickerModel, o DatePickerOptions, disabled bool) {
	if !m.rangeMode() {
		TextField(c, &st.input, TextFieldOptions{
			Variant:      TextFieldOutlined,
			Label:        "Date",
			Placeholder:  "YYYY-MM-DD",
			Error:        st.inputErr,
			ErrorMessage: dateFormatError,
			Disabled:     disabled,
		})
		want := formatPickerDate(*m.value)
		if st.input == want {
			if st.inputErr {
				st.inputErr = false
				c.Invalidate()
			}
			return
		}
		before := st.inputErr
		if day, ok := parsePickerDate(st.input); ok {
			st.setSingle(m, day, o)
			st.inputSync = *m.value
			st.inputErr = false
		} else {
			st.inputErr = true
		}
		if st.inputErr != before {
			c.Invalidate()
		}
		return
	}
	TextField(c, &st.startInput, TextFieldOptions{
		Variant:      TextFieldOutlined,
		Label:        "Start date",
		Placeholder:  "YYYY-MM-DD",
		Error:        st.startErr,
		ErrorMessage: dateFormatError,
		Disabled:     disabled,
	})
	TextField(c, &st.endInput, TextFieldOptions{
		Variant:      TextFieldOutlined,
		Label:        "End date",
		Placeholder:  "YYYY-MM-DD",
		Error:        st.endErr,
		ErrorMessage: dateFormatError,
		Disabled:     disabled,
	})
	wantStart, wantEnd := formatPickerDate(*m.start), formatPickerDate(*m.end)
	if st.startInput == wantStart && st.endInput == wantEnd {
		if st.startErr || st.endErr {
			st.startErr, st.endErr = false, false
			c.Invalidate()
		}
		return
	}
	errChanged, valueChanged := st.applyRangeInputs(m, o)
	if errChanged || valueChanged {
		c.Invalidate()
	}
}

// buildDatePickerActions builds a modal's Cancel and OK buttons.
func buildDatePickerActions(c *ui.Context, o DatePickerOptions, onConfirm func()) {
	row := ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Margin(4, 0, 0, 0)
	row.Children(func() {
		if Button(c, "Cancel", ButtonOptions{Variant: ButtonText, Size: tokens.SizeXS}).Clicked() {
			if o.Open != nil {
				*o.Open = false
			}
		}
		if Button(c, "OK", ButtonOptions{Variant: ButtonFilled, Size: tokens.SizeXS}).Clicked() && onConfirm != nil {
			onConfirm()
		}
	})
}

// applyRangeInputs parses both range fields and applies the valid pair. It
// reports whether an error state changed and whether the selection did.
func (st *datePickerState) applyRangeInputs(m datePickerModel, o DatePickerOptions) (errChanged, valueChanged bool) {
	startText := strings.TrimSpace(st.startInput)
	endText := strings.TrimSpace(st.endInput)
	s, sok := parsePickerDate(startText)
	e, eok := parsePickerDate(endText)
	beforeStart, beforeEnd := st.startErr, st.endErr
	st.startErr = startText != "" && !sok || startText == "" && endText != ""
	st.endErr = endText != "" && !eok
	if st.startErr || st.endErr {
		return st.startErr != beforeStart || st.endErr != beforeEnd, false
	}
	from := time.Time{}
	if sok {
		from = pickerKeepClock(*m.start, s)
	}
	to := time.Time{}
	if eok {
		to = pickerKeepClock(*m.end, e)
	}
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		from, to = to, from
		st.startInput, st.endInput = formatPickerDate(from), formatPickerDate(to)
	}
	if !sameCalendarDay(*m.start, from) || !sameCalendarDay(*m.end, to) {
		*m.start, *m.end = from, to
		valueChanged = true
		if o.OnChanged != nil {
			o.OnChanged()
		}
	}
	st.startSync, st.endSync = from, to
	return st.startErr != beforeStart || st.endErr != beforeEnd, valueChanged
}

// pick applies a day clicked or chosen with the keyboard.
func (st *datePickerState) pick(m datePickerModel, o DatePickerOptions, day time.Time) {
	if m.rangeMode() {
		st.pickRange(m, o, day)
		return
	}
	st.setSingle(m, day, o)
	st.cursor = day
}

// setSingle sets *m.value's day to day, keeping its clock and location, and
// reports whether it changed.
func (st *datePickerState) setSingle(m datePickerModel, day time.Time, o DatePickerOptions) bool {
	next := pickerKeepClock(*m.value, day)
	if next.Equal(*m.value) {
		st.cursor = day
		return false
	}
	*m.value = next
	st.lastDate = next
	st.cursor = day
	if o.OnChanged != nil {
		o.OnChanged()
	}
	return true
}

// pickRange advances a range selection: the first click sets the start, the
// second the end (swapping a reversed pair), and a click after a complete
// range starts a new one.
func (st *datePickerState) pickRange(m datePickerModel, o DatePickerOptions, day time.Time) {
	switch {
	case m.start.IsZero() || !m.end.IsZero():
		*m.start = pickerKeepClock(*m.start, day)
		*m.end = time.Time{}
	default:
		to := pickerKeepClock(*m.end, day)
		if to.Before(*m.start) {
			*m.start, *m.end = to, *m.start
		} else {
			*m.end = to
		}
	}
	st.cursor = day
	st.lastFrom, st.lastTo = *m.start, *m.end
	st.startInput = formatPickerDate(*m.start)
	st.endInput = formatPickerDate(*m.end)
	st.startSync, st.endSync = *m.start, *m.end
	st.startErr, st.endErr = false, false
	if o.OnChanged != nil {
		o.OnChanged()
	}
}

// datePickerDayState reports whether day is an endpoint or lies inside the
// selection, comparing calendar days.
func datePickerDayState(m datePickerModel, day time.Time) (chosen, inRange bool) {
	if !m.rangeMode() {
		return !m.value.IsZero() && sameCalendarDay(*m.value, day), false
	}
	from, to := *m.start, *m.end
	if from.IsZero() {
		return false, false
	}
	if !to.IsZero() && to.Before(from) {
		from, to = to, from
	}
	dayNum := calendarDayNumber(day)
	fromNum := calendarDayNumber(from)
	if dayNum == fromNum {
		return true, true
	}
	if to.IsZero() {
		return false, false
	}
	toNum := calendarDayNumber(to)
	if dayNum == toNum {
		return true, true
	}
	return false, dayNum > fromNum && dayNum < toNum
}

// moveMonth moves the cursor by delta months, keeping its day when the target
// month has it.
func (st *datePickerState) moveMonth(delta int) {
	cur := st.cursor
	last := time.Date(cur.Year(), cur.Month()+time.Month(delta)+1, 0, 0, 0, 0, 0, cur.Location())
	day := min(cur.Day(), last.Day())
	st.cursor = time.Date(last.Year(), last.Month(), day, cur.Hour(), cur.Minute(), cur.Second(), cur.Nanosecond(), cur.Location())
}

// monthTitle formats the cursor's month and year.
func (st *datePickerState) monthTitle() string {
	return st.cursor.Format("January 2006")
}

// dateInputTitle labels the text entry view.
func dateInputTitle(m datePickerModel) string {
	if m.rangeMode() {
		return "Enter date range"
	}
	return "Enter date"
}

// pickerWeekdays returns the seven short weekday names from first.
func pickerWeekdays(first int) [7]string {
	names := [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	var out [7]string
	for i := range 7 {
		out[i] = names[(first+i)%7]
	}
	return out
}

// chevronIcon returns the icon of a previous or next control, mirrored right
// to left.
func chevronIcon(back, rtl bool) IconDraw {
	if back != rtl {
		return mdicon.ChevronLeft.Draw
	}
	return mdicon.ChevronRight.Draw
}

// paintCenteredSpan paints one text span centered on (cx, cy).
func paintCenteredSpan(p *ui.Painter, cx, cy float32, span ui.Span) {
	if span.Text == "" {
		return
	}
	w, h := p.MeasureText(0, span)
	p.RichText(cx-w/2, cy-h/2, 0, span)
}

// formatPickerDate formats a date as YYYY-MM-DD, or "" for a zero one.
func formatPickerDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateFormatLayout)
}

// parsePickerDate parses YYYY-MM-DD, allowing one-digit months and days, and
// reports whether it is a real calendar date.
func parsePickerDate(text string) (time.Time, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return time.Time{}, false
	}
	parts := strings.Split(text, "-")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	year, err1 := strconv.Atoi(parts[0])
	month, err2 := strconv.Atoi(parts[1])
	day, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil || year <= 0 || month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, false
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, false
	}
	return t, true
}

// pickerKeepClock returns day with base's clock, location and nanoseconds, or
// midnight in the local zone for a zero base.
func pickerKeepClock(base, day time.Time) time.Time {
	y, mo, d := day.Date()
	if base.IsZero() {
		return time.Date(y, mo, d, 0, 0, 0, 0, time.Local)
	}
	h, mi, s := base.Clock()
	return time.Date(y, mo, d, h, mi, s, base.Nanosecond(), base.Location())
}

// sameCalendarDay reports whether a and b fall on the same calendar day, with
// two zero times equal.
func sameCalendarDay(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return a.IsZero() && b.IsZero()
	}
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// calendarDayNumber numbers a calendar day so days in different locations
// compare by their date alone.
func calendarDayNumber(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}
