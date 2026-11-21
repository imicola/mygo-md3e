package md3e

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// pickerCanvas builds a 480x560 canvas holding build's content in a padded
// column at the top start corner, tall enough for a docked picker.
func pickerCanvas(th *Theme, build func(c *ui.Context)) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 560).Background(canvasRGB).Padding(16).AlignItems(ui.Start).Children(func() {
				ui.Column(c).AlignItems(ui.Start).Children(func() {
					build(c)
				})
			})
		})
	}, 480, 560)
}

// pickerDay returns a date at 9:30 in the local zone.
func pickerDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 9, 30, 0, 0, time.Local)
}

// pickerDayLabel is the accessible name a day cell carries.
func pickerDayLabel(t time.Time) string {
	return t.Format("January 2, 2006")
}

// cellInfo returns a grid cell's access info by name.
func cellInfo(t *testing.T, tt *ui.Tester, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleCell, name)
	if !ok {
		t.Fatalf("%q cell not in the access tree", name)
	}
	return info
}

// TestDatePickerClickAndToday checks the month grid's roles, that a click
// chooses a day, the Primary circle it paints, and today's ring.
func TestDatePickerClickAndToday(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	value := time.Time{}
	changed := 0
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &value, DatePickerOptions{
			Label:     "Due date",
			OnChanged: func() { changed++ },
		})
	})
	tt.Frame()
	now := tt.Now()
	first, second := 5, 7
	if now.Day() == first {
		first, second = 6, 8
	}
	value = time.Date(now.Year(), now.Month(), first, 9, 30, 0, 0, now.Location())
	tt.Frame()
	tt.Advance(time.Second)

	title := now.Format("January 2006")
	if _, ok := tt.AccessByRole(ui.RoleGrid, title); !ok {
		t.Fatalf("the calendar is not a grid named %q", title)
	}
	if _, ok := tt.AccessByRole(ui.RoleColumnHeader, "Sun"); !ok {
		t.Fatal("the weekday headers do not read as column headers")
	}
	today := cellInfo(t, tt, pickerDayLabel(now))
	if today.Current != ui.CurrentDate {
		t.Fatalf("today reads Current %v, want CurrentDate", today.Current)
	}
	if today.Selected {
		t.Fatal("today reads selected while another day is chosen")
	}
	ring := tt.Image().RGBAAt(int(today.Bounds.X+today.Bounds.W/2), int(today.Bounds.Y)+2)
	if !colorMatches(ring, cs.Primary, 60) {
		t.Fatalf("today's ring pixel %+v is not Primary %+v", ring, rgbOf(cs.Primary))
	}

	target := time.Date(now.Year(), now.Month(), second, 9, 30, 0, 0, now.Location())
	tt.Click(pickerDayLabel(target))
	tt.Advance(time.Second)
	if value.Day() != second || value.Month() != target.Month() || value.Year() != target.Year() {
		t.Fatalf("the click chose %v, want %v", value, target)
	}
	if changed == 0 {
		t.Fatal("choosing a day did not report OnChanged")
	}
	if !cellInfo(t, tt, pickerDayLabel(target)).Selected {
		t.Fatal("the chosen day does not read as selected")
	}
	info := cellInfo(t, tt, pickerDayLabel(target))
	px := tt.Image().RGBAAt(int(info.Bounds.X)+8, int(info.Bounds.Y+info.Bounds.H/2))
	if !colorMatches(px, cs.Primary, 24) {
		t.Fatalf("the chosen day's circle pixel %+v is not Primary %+v", px, rgbOf(cs.Primary))
	}
	if today.Selected {
		t.Fatal("the previous day stayed selected")
	}
	if tt.FramePending() {
		t.Fatal("the date picker keeps asking for frames after choosing")
	}
}

// TestDatePickerMonthYearAndKeyboard checks month paging, the year grid the
// title opens, and the grid's arrow, Page Down, Enter and Home keys.
func TestDatePickerMonthYearAndKeyboard(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := pickerDay(2026, time.November, 15)
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &value, DatePickerOptions{Label: "Due date"})
	})
	tt.Advance(300 * time.Millisecond)

	tt.Click("Next month")
	if _, ok := tt.AccessByRole(ui.RoleGrid, "December 2026"); !ok {
		t.Fatal("Next month did not show December 2026")
	}
	tt.Click("Previous month")
	tt.Click("Previous month")
	if _, ok := tt.AccessByRole(ui.RoleGrid, "October 2026"); !ok {
		t.Fatal("Previous month did not show October 2026")
	}

	tt.Click("October 2026")
	if _, ok := tt.AccessByRole(ui.RoleGrid, "Choose year"); !ok {
		t.Fatal("the title did not open the year grid")
	}
	tt.Click("2024")
	if _, ok := tt.AccessByRole(ui.RoleGrid, "October 2024"); !ok {
		t.Fatal("choosing a year did not return to October 2024")
	}

	tt.Click(pickerDayLabel(pickerDay(2024, time.October, 20)))
	tt.Advance(300 * time.Millisecond)
	tt.Key(0, ui.KeyPageDown)
	if _, ok := tt.AccessByRole(ui.RoleGrid, "November 2024"); !ok {
		t.Fatal("Page Down did not page to November 2024")
	}
	tt.Key(0, ui.KeyEnter)
	if value.Year() != 2024 || value.Month() != time.November || value.Day() != 20 {
		t.Fatalf("Enter chose %v, want 2024-11-20", value)
	}
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyEnter)
	if value.Month() != time.November || value.Day() != 19 {
		t.Fatalf("Left and Enter chose %v, want 2024-11-19", value)
	}
	tt.Key(0, ui.KeyHome)
	tt.Key(0, ui.KeyEnter)
	if value.Day() != 1 {
		t.Fatalf("Home and Enter chose day %d, want 1", value.Day())
	}
	if tt.FramePending() {
		t.Fatal("the date picker keeps asking for frames after key navigation")
	}
}

// TestDateRangeHighlight checks the endpoints' Primary circles, the
// PrimaryContainer band between them, its rounded row start and a new range.
func TestDateRangeHighlight(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	start := pickerDay(2026, time.November, 28)
	end := pickerDay(2026, time.December, 5)
	tt := pickerCanvas(th, func(c *ui.Context) {
		DateRangePicker(c, &start, &end, DatePickerOptions{Label: "Trip"})
	})
	tt.Advance(time.Second)

	if _, ok := tt.AccessByRole(ui.RoleGroup, "Trip"); !ok {
		t.Fatal("the range picker is not a named group")
	}

	for _, day := range []time.Time{
		pickerDay(2026, time.November, 28),
		pickerDay(2026, time.December, 5),
	} {
		cell := cellInfo(t, tt, pickerDayLabel(day))
		if !cell.Selected {
			t.Fatalf("endpoint %v does not read as selected", day)
		}
		px := tt.Image().RGBAAt(int(cell.Bounds.X)+8, int(cell.Bounds.Y+cell.Bounds.H/2))
		if !colorMatches(px, cs.Primary, 24) {
			t.Fatalf("endpoint %v pixel %+v is not Primary", day, px)
		}
	}
	mid := cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.December, 2)))
	px := tt.Image().RGBAAt(int(mid.Bounds.X)+2, int(mid.Bounds.Y+mid.Bounds.H/2))
	if !colorMatches(px, cs.PrimaryContainer, 16) {
		t.Fatalf("the band pixel %+v is not PrimaryContainer %+v", px, rgbOf(cs.PrimaryContainer))
	}
	// A band that begins a week rounds its outer corner: the row start is
	// filled at its middle and empty in the corner.
	rowStart := cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.November, 29)))
	midPx := tt.Image().RGBAAt(int(rowStart.Bounds.X)+1, int(rowStart.Bounds.Y+rowStart.Bounds.H/2))
	if !colorMatches(midPx, cs.PrimaryContainer, 16) {
		t.Fatalf("the row-start band pixel %+v is not PrimaryContainer", midPx)
	}
	cornerPx := tt.Image().RGBAAt(int(rowStart.Bounds.X)+1, int(rowStart.Bounds.Y)+1)
	if colorMatches(cornerPx, cs.PrimaryContainer, 16) {
		t.Fatalf("the rounded row start paints the band into its corner: %+v", cornerPx)
	}
	if !cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.December, 2))).Selected {
		t.Fatal("a day inside the range does not read as selected")
	}

	// A click after a complete range starts a new one.
	tt.Click(pickerDayLabel(pickerDay(2026, time.December, 10)))
	tt.Advance(300 * time.Millisecond)
	if !sameCalendarDay(start, pickerDay(2026, time.December, 10)) || !end.IsZero() {
		t.Fatalf("a new range starts at %v..%v", start, end)
	}
	// A second click before the start swaps them.
	tt.Click(pickerDayLabel(pickerDay(2026, time.December, 3)))
	tt.Advance(300 * time.Millisecond)
	if !sameCalendarDay(start, pickerDay(2026, time.December, 3)) ||
		!sameCalendarDay(end, pickerDay(2026, time.December, 10)) {
		t.Fatalf("a reversed pair chose %v..%v", start, end)
	}
	if tt.FramePending() {
		t.Fatal("the range picker keeps asking for frames")
	}
}

// TestDatePickerTextMode checks the calendar/text switch, invalid input
// rejection and a valid date sync.
func TestDatePickerTextMode(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Time{}
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &value, DatePickerOptions{Label: "Due date"})
	})
	tt.Advance(300 * time.Millisecond)
	tt.Click("Type date")
	tt.Advance(300 * time.Millisecond)

	field := buttonInfo(t, tt, ui.RoleTextField, "Date")
	tt.ClickAt(center(field.Bounds))
	tt.Type("2026-13-40")
	tt.Advance(300 * time.Millisecond)
	if !value.IsZero() {
		t.Fatalf("invalid text changed the date to %v", value)
	}
	if info := buttonInfo(t, tt, ui.RoleTextField, "Date"); !info.Invalid {
		t.Fatal("invalid text does not read as invalid")
	}

	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("2026-12-24")
	tt.Frame()
	if value.Year() != 2026 || value.Month() != time.December || value.Day() != 24 {
		t.Fatalf("valid text set %v, want 2026-12-24", value)
	}
	if info := buttonInfo(t, tt, ui.RoleTextField, "Date"); info.Invalid {
		t.Fatal("a valid date still reads as invalid")
	}
	tt.Click("Show calendar")
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleGrid, "December 2026"); !ok {
		t.Fatal("returning to the calendar did not show December 2026")
	}
	if tt.FramePending() {
		t.Fatal("the text mode keeps asking for frames")
	}
}

// TestDatePickerModalFocusAndEscape checks the modal's dialog role, focus
// lock, Escape closing and the focus returning to the trigger.
func TestDatePickerModalFocusAndEscape(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := false
	value := pickerDay(2026, time.November, 12)
	tt := pickerCanvas(th, func(c *ui.Context) {
		if Button(c, "Open date").Clicked() {
			open = true
		}
		DatePicker(c, &value, DatePickerOptions{
			Mode:  DatePickerModal,
			Open:  &open,
			Label: "Due date",
		})
	})
	tt.Click("Open date")
	tt.Advance(600 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Due date"); !ok {
		t.Fatal("the modal is not a dialog named Due date")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Open date"); ok {
		t.Fatal("the window behind the modal is not inert")
	}
	if tt.Focused("Open date") {
		t.Fatal("the modal did not take the focus")
	}

	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 20)))
	tt.Advance(300 * time.Millisecond)
	if value.Day() != 20 {
		t.Fatalf("a click in the modal chose day %d, want 20", value.Day())
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Due date"); !ok {
		t.Fatal("choosing a day closed the modal before OK")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	if open {
		t.Fatal("Escape did not close the modal")
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Due date"); ok {
		t.Fatal("the closed modal still shows")
	}
	if !tt.Focused("Open date") {
		t.Fatal("closing does not give the focus back to the trigger")
	}
	if tt.FramePending() {
		t.Fatal("the closed modal keeps asking for frames")
	}

	tt.Click("Open date")
	tt.Advance(500 * time.Millisecond)
	tt.Click("OK")
	tt.Advance(time.Second)
	if open {
		t.Fatal("OK did not close the modal")
	}
}

// TestDatePickerModalReduceMotion checks a reduced-motion modal opens and
// closes without leaving animation frames.
func TestDatePickerModalReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	open := false
	value := pickerDay(2026, time.November, 12)
	tt := pickerCanvas(th, func(c *ui.Context) {
		if Button(c, "Open date").Clicked() {
			open = true
		}
		DatePicker(c, &value, DatePickerOptions{Mode: DatePickerModal, Open: &open, Label: "Due date"})
	})
	tt.Click("Open date")
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Due date"); !ok {
		t.Fatal("the reduced-motion modal does not open")
	}
	if tt.FramePending() {
		t.Fatal("the reduced-motion modal asks for frames while open")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if open {
		t.Fatal("Escape did not close the reduced-motion modal")
	}
	if !tt.Focused("Open date") {
		t.Fatal("the reduced-motion modal does not restore the focus")
	}
	if tt.FramePending() {
		t.Fatal("the closed reduced-motion modal asks for frames")
	}
}

// TestDatePickerRTLLayout checks the mirrored weekday order and arrow keys.
func TestDatePickerRTLLayout(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := pickerDay(2026, time.November, 15)
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &value, DatePickerOptions{Label: "Due date"})
	})
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(300 * time.Millisecond)

	sun, ok := tt.AccessByRole(ui.RoleColumnHeader, "Sun")
	if !ok {
		t.Fatal("no Sunday header")
	}
	sat, _ := tt.AccessByRole(ui.RoleColumnHeader, "Sat")
	if sun.Bounds.X <= sat.Bounds.X {
		t.Fatalf("right to left puts Sun at %.1f and Sat at %.1f", sun.Bounds.X, sat.Bounds.X)
	}
	prev := buttonInfo(t, tt, ui.RoleButton, "Previous month")
	next := buttonInfo(t, tt, ui.RoleButton, "Next month")
	if prev.Bounds.X <= next.Bounds.X {
		t.Fatalf("right to left puts Previous at %.1f and Next at %.1f", prev.Bounds.X, next.Bounds.X)
	}

	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 15)))
	tt.Advance(300 * time.Millisecond)
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyEnter)
	if value.Day() != 16 {
		t.Fatalf("right-to-left Left chose day %d, want 16", value.Day())
	}
	tt.Key(0, ui.KeyRight)
	tt.Key(0, ui.KeyEnter)
	if value.Day() != 15 {
		t.Fatalf("right-to-left Right chose day %d, want 15", value.Day())
	}
	if tt.FramePending() {
		t.Fatal("the right-to-left picker keeps asking for frames")
	}
}

// TestDatePickerReduceMotion checks a reduced-motion picker settles without
// animation frames and still chooses days.
func TestDatePickerReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	value := pickerDay(2026, time.November, 15)
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &value, DatePickerOptions{Label: "Due date"})
	})
	tt.Frame()
	tt.Click("Next month")
	tt.Frame()
	if tt.FramePending() {
		t.Fatal("a reduced-motion month page asks for frames")
	}
	tt.Click(pickerDayLabel(pickerDay(2026, time.December, 3)))
	tt.Frame()
	if value.Month() != time.December || value.Day() != 3 {
		t.Fatalf("the reduced-motion picker chose %v", value)
	}
	if tt.FramePending() {
		t.Fatal("a reduced-motion choice asks for frames")
	}
}

// dialInfo returns the clock dial's access info and center.
func dialInfo(t *testing.T, tt *ui.Tester) (ui.AccessInfo, float32, float32) {
	t.Helper()
	info, ok := tt.AccessByRole(ui.RoleGroup, "Clock dial")
	if !ok {
		t.Fatal("the clock dial is not a named group")
	}
	cx, cy := center(info.Bounds)
	return info, cx, cy
}

// TestTimePickerDialClickAndAutoAdvance checks pressing the dial picks the
// hour, releasing springs to the minutes, and the next press picks one.
func TestTimePickerDialClickAndAutoAdvance(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := pickerDay(2026, time.May, 1)
	value = time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{Label: "Alarm"})
	})
	tt.Advance(time.Second)
	info, cx, cy := dialInfo(t, tt)
	if info.Value != "10:30" {
		t.Fatalf("the dial reads %q, want 10:30", info.Value)
	}

	// Three o'clock on the hour dial.
	tt.ClickAt(cx+timeDialOuterR, cy)
	tt.Advance(time.Second)
	if value.Hour() != 3 {
		t.Fatalf("the dial chose hour %d, want 3", value.Hour())
	}
	if info, ok := tt.AccessByRole(ui.RoleTab, "Minute"); !ok || !info.Selected {
		t.Fatal("releasing an hour does not switch the dial to minutes")
	}
	// Six o'clock on the minute dial.
	tt.ClickAt(cx, cy+timeDialOuterR)
	tt.Advance(time.Second)
	if value.Minute() != 30 {
		t.Fatalf("the minute dial chose %d, want 30", value.Minute())
	}
	if tt.FramePending() {
		t.Fatal("the dial keeps asking for frames after choosing")
	}
	info, _, _ = dialInfo(t, tt)
	if info.Value != "03:30" {
		t.Fatalf("the dial reads %q, want 03:30", info.Value)
	}
}

// TestTimePickerDialDrag checks dragging the hand around the dial.
func TestTimePickerDialDrag(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(time.Second)
	_, cx, cy := dialInfo(t, tt)

	// From three o'clock to nine o'clock on the hour dial.
	tt.Press(cx+timeDialOuterR, cy)
	tt.Move(cx, cy+timeDialOuterR)
	tt.Move(cx-timeDialOuterR, cy)
	tt.Release(cx-timeDialOuterR, cy)
	tt.Advance(time.Second)
	if value.Hour() != 9 {
		t.Fatalf("the drag chose hour %d, want 9", value.Hour())
	}
	if info, ok := tt.AccessByRole(ui.RoleTab, "Minute"); !ok || !info.Selected {
		t.Fatal("releasing a dragged hour does not switch to minutes")
	}
	if tt.FramePending() {
		t.Fatal("the dragged dial keeps asking for frames")
	}
}

// TestTimePickerTwelveHourPeriod checks the AM/PM switch and that the dial
// keeps the half of the day.
func TestTimePickerTwelveHourPeriod(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12, Label: "Alarm"})
	})
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleRadioGroup, ""); !ok {
		t.Fatal("the AM/PM switch is not a radio group")
	}
	tt.Click("PM")
	tt.Advance(300 * time.Millisecond)
	if value.Hour() != 22 {
		t.Fatalf("PM chose hour %d, want 22", value.Hour())
	}
	_, cx, cy := dialInfo(t, tt)
	tt.ClickAt(cx+timeDialOuterR, cy)
	tt.Advance(300 * time.Millisecond)
	if value.Hour() != 15 {
		t.Fatalf("the dial chose hour %d in PM, want 15", value.Hour())
	}
	tt.Click("AM")
	tt.Advance(time.Second)
	if value.Hour() != 3 {
		t.Fatalf("AM chose hour %d, want 3", value.Hour())
	}
	if tt.FramePending() {
		t.Fatal("the half-day switch keeps asking for frames")
	}
}

// TestTimePickerTwentyFourHourRings checks the outer 00, 13–23 ring and the
// inner 1–12 ring.
func TestTimePickerTwentyFourHourRings(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 9, 15, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 24})
	})
	tt.Advance(time.Second)
	_, cx, cy := dialInfo(t, tt)

	// Outer ring, one tick clockwise of the top: 13.
	x, y := dialPoint(cx, cy, 30, timeDialOuterR)
	tt.ClickAt(x, y)
	tt.Advance(300 * time.Millisecond)
	if value.Hour() != 13 {
		t.Fatalf("the outer ring chose hour %d, want 13", value.Hour())
	}
	tt.Click("Hour")
	tt.Advance(300 * time.Millisecond)
	x, y = dialPoint(cx, cy, 30, timeDialInnerR)
	tt.ClickAt(x, y)
	tt.Advance(300 * time.Millisecond)
	if value.Hour() != 1 {
		t.Fatalf("the inner ring chose hour %d, want 1", value.Hour())
	}
	tt.Click("Hour")
	tt.Advance(300 * time.Millisecond)
	x, y = dialPoint(cx, cy, 0, timeDialOuterR)
	tt.ClickAt(x, y)
	tt.Advance(time.Second)
	if value.Hour() != 0 {
		t.Fatalf("the outer top chose hour %d, want 0", value.Hour())
	}
	if tt.FramePending() {
		t.Fatal("the 24-hour dial keeps asking for frames")
	}
}

// TestTimePickerInputModeSync checks the dial/input switch, range validation,
// steppers and the sync back to the dial.
func TestTimePickerInputModeSync(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(300 * time.Millisecond)
	tt.Click("Switch to input")
	tt.Advance(300 * time.Millisecond)

	hour := buttonInfo(t, tt, ui.RoleTextField, "Hour")
	minute := buttonInfo(t, tt, ui.RoleTextField, "Minute")
	if hour.Value != "10" || minute.Value != "30" {
		t.Fatalf("the fields read %q and %q, want 10 and 30", hour.Value, minute.Value)
	}
	tt.ClickAt(center(hour.Bounds))
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("99")
	tt.Advance(300 * time.Millisecond)
	if value.Hour() != 10 {
		t.Fatalf("invalid hour input changed the hour to %d", value.Hour())
	}
	if info := buttonInfo(t, tt, ui.RoleTextField, "Hour"); !info.Invalid {
		t.Fatal("an out-of-range hour does not read as invalid")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("07")
	tt.Frame()
	if value.Hour() != 7 {
		t.Fatalf("the hour field set hour %d, want 7", value.Hour())
	}
	if info := buttonInfo(t, tt, ui.RoleTextField, "Hour"); info.Invalid {
		t.Fatal("a valid hour still reads as invalid")
	}

	tt.Click("Increase minute")
	tt.Advance(300 * time.Millisecond)
	if value.Minute() != 31 {
		t.Fatalf("the minute stepper set %d, want 31", value.Minute())
	}
	tt.Click("Decrease minute")
	tt.Advance(300 * time.Millisecond)
	if value.Minute() != 30 {
		t.Fatalf("the minute stepper set %d, want 30", value.Minute())
	}

	tt.Click("Switch to dial")
	tt.Advance(time.Second)
	info, _, _ := dialInfo(t, tt)
	if info.Value != "07:30" {
		t.Fatalf("the dial reads %q after input, want 07:30", info.Value)
	}
	if tt.FramePending() {
		t.Fatal("the input mode keeps asking for frames")
	}
}

// TestTimePickerKeyboard checks focusing the dial and stepping its active
// segment, and Enter moving to the minutes.
func TestTimePickerKeyboard(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(time.Second)
	_, cx, cy := dialInfo(t, tt)
	tt.ClickAt(cx, cy) // the center picks nothing but focuses the dial
	tt.Advance(300 * time.Millisecond)
	if !tt.Focused("Clock dial") {
		t.Fatal("clicking the dial center did not focus it")
	}
	tt.Key(0, ui.KeyRight)
	if value.Hour() != 11 {
		t.Fatalf("Right stepped the hour to %d, want 11", value.Hour())
	}
	tt.Key(0, ui.KeyEnter)
	tt.Key(0, ui.KeyRight)
	if value.Minute() != 31 {
		t.Fatalf("Right after Enter stepped the minute to %d, want 31", value.Minute())
	}
	tt.Key(0, ui.KeyPageUp)
	if value.Minute() != 36 {
		t.Fatalf("Page Up stepped the minute to %d, want 36", value.Minute())
	}
	tt.Key(0, ui.KeyDown)
	if value.Minute() != 35 {
		t.Fatalf("Down stepped the minute to %d, want 35", value.Minute())
	}
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("the keyboard-driven dial keeps asking for frames")
	}
}

// TestTimePickerReduceMotionAndDisabled checks the reduced-motion hand and
// that a disabled picker takes no input.
func TestTimePickerReduceMotionAndDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Frame()
	_, cx, cy := dialInfo(t, tt)
	tt.ClickAt(cx+timeDialOuterR, cy)
	tt.Frame()
	if value.Hour() != 3 {
		t.Fatalf("the reduced-motion dial chose hour %d, want 3", value.Hour())
	}
	tt.ClickAt(cx, cy+timeDialOuterR)
	tt.Frame()
	if value.Minute() != 30 {
		t.Fatalf("the reduced-motion minute dial chose %d, want 30", value.Minute())
	}
	if tt.FramePending() {
		t.Fatal("the reduced-motion dial asks for frames")
	}

	locked := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	dis := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &locked, TimePickerOptions{HourCycle: 12, Disabled: true})
	})
	dis.Frame()
	info, ok := dis.AccessByRole(ui.RoleGroup, "Clock dial")
	if !ok || !info.Disabled {
		t.Fatal("the disabled dial does not read as disabled")
	}
	dcx, dcy := center(info.Bounds)
	dis.ClickAt(dcx+timeDialOuterR, dcy)
	dis.Advance(300 * time.Millisecond)
	if locked.Hour() != 10 {
		t.Fatalf("the disabled dial changed the hour to %d", locked.Hour())
	}
	if dis.FramePending() {
		t.Fatal("the disabled picker keeps asking for frames")
	}
}

// TestPickerNamedControls checks that both pickers leave no unnamed
// interactive node.
func TestPickerNamedControls(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	day := pickerDay(2026, time.November, 12)
	clock := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &day, DatePickerOptions{Label: "Due date"})
		TimePicker(c, &clock, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(time.Second)
	tt.Click("Switch to input")
	tt.Advance(300 * time.Millisecond)
	for _, a := range tt.AccessAll() {
		switch a.Role {
		case ui.RoleButton, ui.RoleTab, ui.RoleRadio, ui.RoleToggleButton,
			ui.RolePopUpButton, ui.RoleLink, ui.RoleCheckBox, ui.RoleSwitch, ui.RoleCell:
			if a.Name == "" {
				t.Fatalf("a %v at %v has no name", a.Role, a.Bounds)
			}
		}
	}
	if _, ok := tt.AccessByRole(ui.RoleTab, "Hour"); !ok {
		t.Fatal("the hours are not a named tab")
	}
	if _, ok := tt.AccessByRole(ui.RoleTab, "Minute"); !ok {
		t.Fatal("the minutes are not a named tab")
	}
	if _, ok := tt.AccessByRole(ui.RoleRadio, "AM"); !ok {
		t.Fatal("AM is not a named radio button")
	}
	if _, ok := tt.AccessByRole(ui.RoleRadio, "PM"); !ok {
		t.Fatal("PM is not a named radio button")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Increase hour"); !ok {
		t.Fatal("the hour stepper is not named")
	}
	if tt.FramePending() {
		t.Fatal("the pickers keep asking for frames")
	}
}

// TestDatePickerDisabled checks a disabled picker takes no input and paints
// the disabled tokens.
func TestDatePickerDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := pickerDay(2026, time.November, 12)
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &value, DatePickerOptions{Label: "Due date", Disabled: true})
	})
	tt.Advance(300 * time.Millisecond)
	cell := cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.November, 20)))
	if !cell.Disabled {
		t.Fatal("a disabled picker's day does not read as disabled")
	}
	tt.ClickAt(center(cell.Bounds))
	tt.Advance(300 * time.Millisecond)
	if value.Day() != 12 {
		t.Fatalf("the disabled picker chose day %d", value.Day())
	}
	if tt.FramePending() {
		t.Fatal("the disabled picker keeps asking for frames")
	}
}
