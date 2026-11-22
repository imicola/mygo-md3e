package md3e

import (
	"image"
	"math"
	"strconv"
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
	if value.Day() != 12 {
		t.Fatalf("draft click in modal mutated committed value to day %d before OK", value.Day())
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Due date"); !ok {
		t.Fatal("choosing a day closed the modal before OK")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	if open {
		t.Fatal("Escape did not close the modal")
	}
	if value.Day() != 12 {
		t.Fatalf("Escape committed draft day %d instead of discarding", value.Day())
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
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 20)))
	tt.Click("OK")
	tt.Advance(time.Second)
	if open {
		t.Fatal("OK did not close the modal")
	}
	if value.Day() != 20 {
		t.Fatalf("OK did not commit draft day 20, got %d", value.Day())
	}
	if !tt.Focused("Open date") {
		t.Fatal("closing via OK does not give focus back to the trigger")
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

// pickerRinged reports whether a 2-pixel Primary focus ring outlines b, at the
// middle of its four edges.
func pickerRinged(tt *ui.Tester, b ui.Rect, col ui.Color) bool {
	img := tt.Image()
	midY := int(b.Y + b.H/2)
	midX := int(b.X + b.W/2)
	return colorMatches(img.RGBAAt(int(b.X)+1, midY), col, 60) &&
		colorMatches(img.RGBAAt(int(b.X+b.W)-2, midY), col, 60) &&
		colorMatches(img.RGBAAt(midX, int(b.Y)+1), col, 60) &&
		colorMatches(img.RGBAAt(midX, int(b.Y+b.H)-2), col, 60)
}

// TestDatePickerRTLFocusRing checks the keyboard focus ring follows the cursor
// to the mirrored column, not the column at the same index from the left.
func TestDatePickerRTLFocusRing(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	cs := th.Colors
	value := pickerDay(2026, time.November, 15)
	tt := pickerCanvas(th, func(c *ui.Context) {
		DatePicker(c, &value, DatePickerOptions{Label: "Due date"})
	})
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(300 * time.Millisecond)

	// Tab past the header's four controls to the grid, which shows its keyboard
	// focus ring.
	for range 5 {
		tt.Key(0, ui.KeyTab)
	}
	// Right to left, Left moves on to the next day, one column left.
	tt.Key(0, ui.KeyLeft)
	tt.Advance(300 * time.Millisecond)
	next := cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.November, 16)))
	prev := cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.November, 15)))
	if next.Bounds.X >= prev.Bounds.X {
		t.Fatalf("the next day sits at %.1f, not left of %.1f", next.Bounds.X, prev.Bounds.X)
	}
	if !next.Focused {
		t.Fatal("the arrow keys did not move the cursor to the next day")
	}
	if !pickerRinged(tt, next.Bounds, cs.Primary) {
		t.Fatalf("no focus ring around the cursor at %v", next.Bounds)
	}
	// The column at the same index from the left belongs to another day: the
	// ring must not paint there.
	if wrong := cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.November, 20))); pickerRinged(tt, wrong.Bounds, cs.Primary) {
		t.Fatalf("the focus ring paints the unmirrored column at %v", wrong.Bounds)
	}
	if tt.FramePending() {
		t.Fatal("the focused right-to-left picker keeps asking for frames")
	}
}

// TestDatePickerDockedKeyAndLayout checks a docked picker adds no zero-width
// sibling: with two of them in a gapped column, the panels sit at the column's
// top and one gap apart, and their keys keep their states apart when they swap.
func TestDatePickerDockedKeyAndLayout(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	first := pickerDay(2026, time.November, 15)
	second := pickerDay(2027, time.January, 10)
	swap := false
	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(480, 900).Background(canvasRGB).Padding(16).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Gap(24).AlignItems(ui.Start).Children(func() {
					buildFirst := func() {
						DatePicker(c.Key("first"), &first, DatePickerOptions{Label: "First"})
					}
					buildSecond := func() {
						DatePicker(c.Key("second"), &second, DatePickerOptions{Label: "Second"})
					}
					if swap {
						buildSecond()
						buildFirst()
						return
					}
					buildFirst()
					buildSecond()
				})
			})
		})
	}, 480, 900)
	tt.Frame()

	firstInfo, ok := tt.AccessByRole(ui.RoleGroup, "First")
	if !ok {
		t.Fatal("the first picker is not a named group")
	}
	secondInfo, ok := tt.AccessByRole(ui.RoleGroup, "Second")
	if !ok {
		t.Fatal("the second picker is not a named group")
	}
	if firstInfo.Bounds.Y != 16 {
		t.Fatalf("the first picker starts at %.1f, not at the column top 16: a zero-width sibling takes a gap", firstInfo.Bounds.Y)
	}
	if got, want := secondInfo.Bounds.Y, firstInfo.Bounds.Y+firstInfo.Bounds.H+24; got != want {
		t.Fatalf("the second picker starts at %.1f, want %.1f: the gap does not match the column's", got, want)
	}
	if firstInfo.Bounds.X != secondInfo.Bounds.X {
		t.Fatalf("the pickers start at different x: %.1f and %.1f", firstInfo.Bounds.X, secondInfo.Bounds.X)
	}

	// Page the first picker's month on.
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleButton && a.Name == "Next month" && a.Bounds.Y < secondInfo.Bounds.Y {
			tt.ClickAt(center(a.Bounds))
			break
		}
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleGrid, "December 2026"); !ok {
		t.Fatal("paging the first picker did not show December 2026")
	}
	if _, ok := tt.AccessByRole(ui.RoleGrid, "January 2027"); !ok {
		t.Fatal("paging the first picker also changed the second")
	}

	// Switch the second picker to its text entry view.
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleButton && a.Name == "Type date" && a.Bounds.Y > secondInfo.Bounds.Y {
			tt.ClickAt(center(a.Bounds))
			break
		}
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleButton, "Show calendar"); !ok {
		t.Fatal("the second picker did not switch to its date field")
	}
	if _, ok := tt.AccessByRole(ui.RoleGrid, "December 2026"); !ok {
		t.Fatal("switching the second picker changed the first")
	}

	// Swapping the two keyed pickers keeps each state with its own key: the
	// first still shows its paged month, the second its text entry view.
	swap = true
	tt.Frame()
	firstInfo, _ = tt.AccessByRole(ui.RoleGroup, "First")
	secondInfo, _ = tt.AccessByRole(ui.RoleGroup, "Second")
	if field := buttonInfo(t, tt, ui.RoleTextField, "Date"); field.Bounds.Y < secondInfo.Bounds.Y ||
		field.Bounds.Y > secondInfo.Bounds.Y+secondInfo.Bounds.H {
		t.Fatalf("the date field at %.1f is outside the Second panel at %v: the keyed state did not follow its key",
			field.Bounds.Y, secondInfo.Bounds)
	}
	grid, ok := tt.AccessByRole(ui.RoleGrid, "December 2026")
	if !ok {
		t.Fatal("the first picker lost its paged month when the pickers swapped")
	}
	if grid.Bounds.Y < firstInfo.Bounds.Y || grid.Bounds.Y > firstInfo.Bounds.Y+firstInfo.Bounds.H {
		t.Fatalf("the December 2026 grid at %.1f is outside the First panel at %v", grid.Bounds.Y, firstInfo.Bounds)
	}
	if tt.FramePending() {
		t.Fatal("the docked pickers keep asking for frames")
	}
}

// TestDatePickerModalAncestorDisabled checks a disabled ancestor disables an
// open modal picker that was not given Disabled itself.
func TestDatePickerModalAncestorDisabled(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := true
	value := pickerDay(2026, time.November, 12)
	tt := pickerCanvas(th, func(c *ui.Context) {
		ui.Box(c).Disabled(true).Children(func() {
			DatePicker(c, &value, DatePickerOptions{Mode: DatePickerModal, Open: &open, Label: "Due date"})
		})
	})
	tt.Advance(600 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Due date"); !ok {
		t.Fatal("the disabled ancestor's modal is not a dialog named Due date")
	}
	cell := cellInfo(t, tt, pickerDayLabel(pickerDay(2026, time.November, 20)))
	if !cell.Disabled {
		t.Fatal("a day of the disabled ancestor's modal does not read as disabled")
	}
	tt.ClickAt(center(cell.Bounds))
	tt.Advance(300 * time.Millisecond)
	if value.Day() != 12 {
		t.Fatalf("the disabled ancestor's modal chose day %d", value.Day())
	}
	if tt.FramePending() {
		t.Fatal("the disabled modal keeps asking for frames")
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
	// Clicking the hour already chosen still commits and advances.
	if err := tt.Click("Hour"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(300 * time.Millisecond)
	x, y := dialPoint(cx, cy, 90, timeDialOuterR)
	tt.ClickAt(x, y)
	tt.Advance(time.Second)
	if value.Hour() != 3 {
		t.Fatalf("re-choosing the selected hour changed it to %d", value.Hour())
	}
	if info, ok := tt.AccessByRole(ui.RoleTab, "Minute"); !ok || !info.Selected {
		t.Fatal("clicking the selected hour does not switch the dial to minutes")
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

// TestTimePickerRTLLayout checks the hours stay left of the minutes, and the
// input fields with them, on a right-to-left page.
func TestTimePickerRTLLayout(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.SetDirection(ui.RightToLeft)
	tt.Advance(300 * time.Millisecond)

	hour, ok := tt.AccessByRole(ui.RoleTab, "Hour")
	if !ok {
		t.Fatal("the hours are not a named tab")
	}
	minute, _ := tt.AccessByRole(ui.RoleTab, "Minute")
	if hour.Bounds.X >= minute.Bounds.X {
		t.Fatalf("right to left puts the hours at %.1f and the minutes at %.1f", hour.Bounds.X, minute.Bounds.X)
	}

	tt.Click("Switch to input")
	tt.Advance(300 * time.Millisecond)
	hourField := buttonInfo(t, tt, ui.RoleTextField, "Hour")
	minuteField := buttonInfo(t, tt, ui.RoleTextField, "Minute")
	if hourField.Bounds.X >= minuteField.Bounds.X {
		t.Fatalf("right to left puts the hour field at %.1f and the minute field at %.1f",
			hourField.Bounds.X, minuteField.Bounds.X)
	}
	if tt.FramePending() {
		t.Fatal("the right-to-left picker keeps asking for frames")
	}
}

// TestTimePickerDisableMidDrag checks disabling a picker while the pointer
// holds the dial ends the drag, so enabling it again does not pick on hover.
func TestTimePickerDisableMidDrag(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	disabled := false
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12, Disabled: disabled})
	})
	tt.Advance(time.Second)
	_, cx, cy := dialInfo(t, tt)

	tt.Press(cx+timeDialOuterR, cy)
	if value.Hour() != 3 {
		t.Fatalf("the press chose hour %d, want 3", value.Hour())
	}
	disabled = true
	tt.Frame()
	disabled = false
	tt.Frame()

	held := value
	tt.Move(cx-timeDialOuterR, cy)
	tt.Advance(300 * time.Millisecond)
	tt.Release(cx-timeDialOuterR, cy)
	tt.Advance(300 * time.Millisecond)
	if value != held {
		t.Fatalf("re-enabling picked %v on hover, want %v", value, held)
	}

	// A fresh press still picks, and releasing an hour springs to the minutes.
	tt.Press(cx, cy+timeDialOuterR)
	tt.Release(cx, cy+timeDialOuterR)
	tt.Advance(time.Second)
	if value.Hour() != 6 {
		t.Fatalf("the press after re-enabling chose hour %d, want 6", value.Hour())
	}
	if info, ok := tt.AccessByRole(ui.RoleTab, "Minute"); !ok || !info.Selected {
		t.Fatal("releasing an hour after re-enabling does not switch to the minutes")
	}
	if tt.FramePending() {
		t.Fatal("the re-enabled picker keeps asking for frames")
	}
}

// dialInkNear reports whether any pixel within rad of (x, y) paints the dial's
// Primary hand.
func dialInkNear(img *image.RGBA, th *Theme, x, y, rad int) bool {
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			if colorMatches(img.RGBAAt(x+dx, y+dy), th.Colors.Primary, 60) {
				return true
			}
		}
	}
	return false
}

// dialHandAngle returns the hand's angle clockwise from the top, measured from
// the Primary ink between radius 40 and 70, or -1 when no hand shows.
func dialHandAngle(th *Theme, tt *ui.Tester, b ui.Rect) float32 {
	cx, cy := center(b)
	img := tt.Image()
	var sx, sy float64
	for y := int(cy) - 72; y <= int(cy)+72; y++ {
		for x := int(cx) - 72; x <= int(cx)+72; x++ {
			dx, dy := float64(x)-float64(cx), float64(y)-float64(cy)
			r := math.Hypot(dx, dy)
			if r < 40 || r > 70 {
				continue
			}
			if !colorMatches(img.RGBAAt(x, y), th.Colors.Primary, 60) {
				continue
			}
			ang := math.Atan2(dx, -dy)
			sx += math.Cos(ang)
			sy += math.Sin(ang)
		}
	}
	if sx == 0 && sy == 0 {
		return -1
	}
	a := math.Atan2(sy, sx) * 180 / math.Pi
	if a < 0 {
		a += 360
	}
	return float32(a)
}

// angleDistance returns how far apart two angles are, around the circle.
func angleDistance(a, b float32) float32 {
	d := absF32(normalizeDialAngle(a) - normalizeDialAngle(b))
	return min(d, 360-d)
}

// TestTimePickerDialHandFollowsAngle checks the hand paints a true diagonal
// from the center to the chosen minute at 36°, 42°, 330° and 60°, and never a
// stray vertical line up the 12 o'clock axis.
func TestTimePickerDialHandFollowsAngle(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	for _, tc := range []struct {
		minute int
		deg    float32
	}{
		{6, 36}, {7, 42}, {55, 330}, {10, 60},
	} {
		value := time.Date(2026, time.May, 1, 10, tc.minute, 0, 0, time.Local)
		tt := pickerCanvas(th, func(c *ui.Context) {
			TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
		})
		tt.Advance(300 * time.Millisecond)
		if err := tt.Click("Minute"); err != nil {
			t.Fatal(err)
		}
		tt.Advance(600 * time.Millisecond)
		_, cx, cy := dialInfo(t, tt)
		img := tt.Image()
		for _, f := range []float32{0.30, 0.45, 0.60, 0.72} {
			x, y := dialPoint(cx, cy, tc.deg, timeDialOuterR*f)
			if !dialInkNear(img, th, int(x), int(y), 2) {
				t.Fatalf("%02d (%.0f°): the hand does not reach r=%.0f at (%.1f,%.1f)",
					tc.minute, tc.deg, timeDialOuterR*f, x, y)
			}
		}
		for _, f := range []float32{0.3, 0.45, 0.6, 0.72} {
			x, y := dialPoint(cx, cy, 0, timeDialOuterR*f)
			if dialInkNear(img, th, int(x), int(y), 1) {
				t.Fatalf("%02d (%.0f°): a stray vertical hand paints 12 o'clock at r=%.0f",
					tc.minute, tc.deg, timeDialOuterR*f)
			}
		}
	}
}

// TestTimePickerSelectionCircleKeepsTickDigits checks that the selection circle
// at 06 and 07 minutes carries the OnPrimary center dot and repaints the tick
// digits it covers in OnPrimary rather than swallowing them.
func TestTimePickerSelectionCircleKeepsTickDigits(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	onSelected := rgbOf(th.Colors.OnPrimary)
	for _, minute := range []int{6, 7} {
		value := time.Date(2026, time.May, 1, 10, minute, 0, 0, time.Local)
		tt := pickerCanvas(th, func(c *ui.Context) {
			TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
		})
		tt.Advance(300 * time.Millisecond)
		if err := tt.Click("Minute"); err != nil {
			t.Fatal(err)
		}
		tt.Advance(600 * time.Millisecond)
		_, cx, cy := dialInfo(t, tt)
		img := tt.Image()
		tipX, tipY := dialPoint(cx, cy, float32(minute)*6, timeDialOuterR)
		if got := img.RGBAAt(int(tipX), int(tipY)); absColorDiff(got, onSelected) > 40 {
			t.Fatalf("%02d: the selection circle center = %v, want the OnPrimary dot %v", minute, got, onSelected)
		}
		// "05" sits at 30°, within or beside the circle; every digit pixel the
		// circle covers must be OnPrimary.
		lx, ly := dialPoint(cx, cy, 30, timeDialOuterR)
		white, ink, darkInside := 0, 0, 0
		for py := int(ly) - 11; py <= int(ly)+11; py++ {
			for px := int(lx) - 12; px <= int(lx)+12; px++ {
				got := img.RGBAAt(px, py)
				d := math.Hypot(float64(px)-float64(tipX), float64(py)-float64(tipY))
				if d <= float64(timeDialCircleR)-1 {
					switch {
					case absColorDiff(got, onSelected) <= 60:
						white++
					case absColorDiff(got, rgbOf(th.Colors.OnSurfaceVariant)) <= 80:
						darkInside++
					}
				} else if absColorDiff(got, rgbOf(th.Colors.OnSurfaceVariant)) <= 80 {
					ink++
				}
			}
		}
		if darkInside != 0 {
			t.Fatalf("%02d: %d pixels of a covered digit stay dark inside the circle", minute, darkInside)
		}
		if white+ink < 4 {
			t.Fatalf("%02d: \"05\" nearly disappeared (%d white, %d ink pixels)", minute, white, ink)
		}
		if minute == 6 && white < 4 {
			t.Fatalf("06: only %d pixels of the covered \"05\" are OnPrimary", white)
		}
	}
}

// TestTimePickerDragReleasedOffDialClearsHeld checks that releasing a drag off
// the dial ends it, advances to the minutes, and leaves no stuck hand that
// picks on hover.
func TestTimePickerDragReleasedOffDialClearsHeld(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(300 * time.Millisecond)
	_, cx, cy := dialInfo(t, tt)

	// Press three o'clock and drag well off the dial before releasing there.
	tt.Press(cx+timeDialOuterR, cy)
	tt.Move(cx+timeDialOuterR+140, cy+90)
	tt.Release(cx+timeDialOuterR+140, cy+90)
	tt.Advance(600 * time.Millisecond)
	if info, ok := tt.AccessByRole(ui.RoleTab, "Minute"); !ok || !info.Selected {
		t.Fatal("releasing off the dial does not advance to the minutes")
	}
	picked := value
	// Hovering the dial afterwards must not pick anything.
	tt.Move(cx, cy+timeDialOuterR)
	tt.Advance(300 * time.Millisecond)
	if value != picked {
		t.Fatalf("hovering after an off-dial release picked %v, want %v", value, picked)
	}
	if tt.FramePending() {
		t.Fatal("the released dial keeps asking for frames")
	}
	// A fresh gesture works: nine o'clock is minute 45.
	tt.Press(cx-timeDialOuterR, cy)
	tt.Release(cx-timeDialOuterR, cy)
	tt.Advance(600 * time.Millisecond)
	if value.Minute() != 45 {
		t.Fatalf("the fresh minute press chose %d, want 45", value.Minute())
	}
	if value.Hour() != picked.Hour() {
		t.Fatalf("the fresh minute press changed the hour to %d", value.Hour())
	}
}

// TestTimePickerDragAcrossTwelveTakesShortWay checks that dragging clockwise
// across 12 and releasing springs back over the short arc instead of spinning
// the other way around.
func TestTimePickerDragAcrossTwelveTakesShortWay(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 0, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(300 * time.Millisecond)
	if err := tt.Click("Minute"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(600 * time.Millisecond)
	info, cx, cy := dialInfo(t, tt)

	tt.Press(dialPoint(cx, cy, 350, timeDialOuterR))
	tt.Move(dialPoint(cx, cy, 355, timeDialOuterR))
	tt.Move(dialPoint(cx, cy, 358, timeDialOuterR))
	tt.Release(dialPoint(cx, cy, 358, timeDialOuterR))
	tt.Advance(120 * time.Millisecond)
	if value.Minute() != 0 {
		t.Fatalf("the drag chose minute %d, want 0", value.Minute())
	}
	if got := dialHandAngle(th, tt, info.Bounds); angleDistance(got, 0) > 12 {
		t.Fatalf("the hand sits at %.0f° right after the drag, want within 12° of 12 o'clock", got)
	}
}

// TestTimePickerZeroValueKeepsPeriod checks that a zero value picks with the
// hour now shows, keeping AM or PM.
func TestTimePickerZeroValueKeepsPeriod(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	var value time.Time
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(300 * time.Millisecond)
	_, cx, cy := dialInfo(t, tt)
	pm := tt.Now().Hour() >= 12
	tt.ClickAt(cx+timeDialOuterR, cy) // three o'clock
	tt.Advance(600 * time.Millisecond)
	want := 3
	if pm {
		want = 15
	}
	if value.Hour() != want {
		t.Fatalf("a zero value chose %02d:00, want %02d:00 (%s)", value.Hour(), want, map[bool]string{true: "PM", false: "AM"}[pm])
	}
	if value.IsZero() {
		t.Fatal("choosing an hour left the value zero")
	}
}

// TestTimePickerInputLayoutAndClearing checks the colon and steppers center on
// the fields, and that clearing a field with Backspace does not flash an error.
func TestTimePickerInputLayoutAndClearing(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	value := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	tt := pickerCanvas(th, func(c *ui.Context) {
		TimePicker(c, &value, TimePickerOptions{HourCycle: 12})
	})
	tt.Advance(300 * time.Millisecond)
	if err := tt.Click("Switch to input"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(300 * time.Millisecond)

	hour := buttonInfo(t, tt, ui.RoleTextField, "Hour")
	minute := buttonInfo(t, tt, ui.RoleTextField, "Minute")
	_, hourY := center(hour.Bounds)
	// The colon sits between the hour stepper and the minute field; its ink
	// must center on the fields rather than hang at the row's top.
	img := tt.Image()
	// The access bounds are the field's input text, 16 DIP inside its outlined
	// container; the colon fills the gap just before the minute container.
	containerLeft := float32(minute.Bounds.X - timePickerPad)
	x0, x1 := int(containerLeft)-16, int(containerLeft)
	top := int(hour.Bounds.Y) - (textFieldHeight-int(hour.Bounds.H))/2
	bottom := top + textFieldHeight
	inkTop, inkBottom, inkCount := 1<<30, -1, 0
	for y := top; y < bottom; y++ {
		for x := x0; x < x1; x++ {
			if absColorDiff(img.RGBAAt(x, y), rgbOf(canvasRGB)) > 40 {
				inkCount++
				inkTop = min(inkTop, y)
				inkBottom = max(inkBottom, y)
			}
		}
	}
	if inkCount == 0 {
		t.Fatalf("the input mode paints no colon in [%d,%d)×[%d,%d)", x0, x1, top, bottom)
	}
	colonY := float32(inkTop+inkBottom) / 2
	if math.Abs(float64(colonY-hourY)) > 3 {
		t.Fatalf("the colon centers at %.1f, want the fields' %.1f", colonY, hourY)
	}
	up := buttonInfo(t, tt, ui.RoleButton, "Increase hour")
	down := buttonInfo(t, tt, ui.RoleButton, "Decrease hour")
	stepperY := (up.Bounds.Y + down.Bounds.Y + down.Bounds.H) / 2
	if math.Abs(float64(stepperY-hourY)) > 3 {
		t.Fatalf("the steppers center at %.1f, want the fields' %.1f", stepperY, hourY)
	}

	tt.ClickAt(center(hour.Bounds))
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Key(0, ui.KeyBackspace)
	tt.Advance(100 * time.Millisecond)
	if info := buttonInfo(t, tt, ui.RoleTextField, "Hour"); info.Invalid {
		t.Fatal("clearing the hour with Backspace flashed an error")
	}
	if value.Hour() != 10 {
		t.Fatalf("clearing the field changed the hour to %d", value.Hour())
	}
	// Leaving the empty field shows the error as a committed edit.
	tt.ClickAt(center(minute.Bounds))
	tt.Advance(100 * time.Millisecond)
	if info := buttonInfo(t, tt, ui.RoleTextField, "Hour"); !info.Invalid {
		t.Fatal("leaving an empty hour does not show its error")
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

// TestTimePickerZeroValueInputKeepsPeriod checks that typing an hour in 12h
// input mode with a zero initial time.Time preserves the displayed AM/PM half
// (even in PM when value.Hour() is 0 while c.Now() is afternoon).
func TestTimePickerZeroValueInputKeepsPeriod(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	var zeroAM, zeroPM time.Time
	usePM := false
	tt := pickerCanvas(th, func(c *ui.Context) {
		ui.Column(c).Children(func() {
			if Button(c, "Use PM").Clicked() {
				usePM = true
			}
			if !usePM {
				TimePicker(c.Key("first"), &zeroAM, TimePickerOptions{
					Mode:      TimePickerInput,
					HourCycle: 12,
				})
			} else {
				TimePicker(c.Key("second"), &zeroPM, TimePickerOptions{
					Mode:      TimePickerInput,
					HourCycle: 12,
				})
			}
		})
	})

	now0 := tt.Now()
	pm0 := now0.Hour() >= 12
	pick0 := 3
	if now0.Hour()%12 == pick0 {
		pick0 = 5
	}
	hour := buttonInfo(t, tt, ui.RoleTextField, "Hour")
	tt.ClickAt(center(hour.Bounds))
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(strconv.Itoa(pick0))
	tt.Frame()
	want0 := pick0
	if pm0 {
		want0 = pick0 + 12
	}
	if zeroAM.Hour() != want0 {
		t.Fatalf("zero-value 12h input set %02d:00, want %02d:00 (pm=%v)", zeroAM.Hour(), want0, pm0)
	}

	// Force the simulated clock into the opposite 12h half so both AM and PM
	// zero-value input behavior are covered regardless of wall-clock time.
	tt.Advance(12 * time.Hour)
	tt.Click("Use PM")
	tt.Frame()
	now1 := tt.Now()
	pm1 := now1.Hour() >= 12
	pick1 := 4
	if now1.Hour()%12 == pick1 {
		pick1 = 6
	}
	hour2 := buttonInfo(t, tt, ui.RoleTextField, "Hour")
	tt.ClickAt(center(hour2.Bounds))
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(strconv.Itoa(pick1))
	tt.Frame()
	want1 := pick1
	if pm1 {
		want1 = pick1 + 12
	}
	if zeroPM.Hour() != want1 {
		t.Fatalf("shifted zero-value 12h input set %02d:00, want %02d:00 (pm=%v)", zeroPM.Hour(), want1, pm1)
	}
}

// TestTimePickerHourCycleToggleSync checks switching between 24h and 12h updates
// the Hour input buffer instead of validating stale "14" under 12h mode.
func TestTimePickerHourCycleToggleSync(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	value := time.Date(2026, time.May, 1, 14, 30, 0, 0, time.Local)
	cycle := 24
	tt := pickerCanvas(th, func(c *ui.Context) {
		ui.Column(c).Gap(8).Children(func() {
			if Button(c, "Toggle Cycle").Clicked() {
				if cycle == 24 {
					cycle = 12
				} else {
					cycle = 24
				}
			}
			TimePicker(c, &value, TimePickerOptions{
				Mode:      TimePickerInput,
				HourCycle: cycle,
			})
		})
	})

	hour := buttonInfo(t, tt, ui.RoleTextField, "Hour")
	if hour.Value != "14" || hour.Invalid {
		t.Fatalf("24h hour = %q invalid=%v, want \"14\" valid", hour.Value, hour.Invalid)
	}
	tt.Click("Toggle Cycle")
	tt.Frame()
	hour = buttonInfo(t, tt, ui.RoleTextField, "Hour")
	if hour.Value != "2" || hour.Invalid {
		t.Fatalf("after switching 24->12, hour = %q invalid=%v, want \"2\" valid", hour.Value, hour.Invalid)
	}
	tt.Click("Toggle Cycle")
	tt.Frame()
	hour = buttonInfo(t, tt, ui.RoleTextField, "Hour")
	if hour.Value != "14" || hour.Invalid {
		t.Fatalf("after switching 12->24, hour = %q invalid=%v, want \"14\" valid", hour.Value, hour.Invalid)
	}
}

// TestTimePickerTextScaleInputWidth checks large TextScale widens the hour and
// minute fields and container instead of clipping inside 88 DIP.
func TestTimePickerTextScaleInputWidth(t *testing.T) {
	thBase := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	thBig := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true, TextScale: 1.5})
	v1 := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
	v2 := v1

	tt1 := pickerCanvas(thBase, func(c *ui.Context) {
		TimePicker(c, &v1, TimePickerOptions{Mode: TimePickerInput, HourCycle: 12})
	})
	tt2 := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, thBig, func() {
			ui.Box(c).Size(640, 560).Background(canvasRGB).Padding(20).Children(func() {
				TimePicker(c, &v2, TimePickerOptions{Mode: TimePickerInput, HourCycle: 12})
			})
		})
	}, 640, 560)

	h1 := buttonInfo(t, tt1, ui.RoleTextField, "Hour")
	h2 := buttonInfo(t, tt2, ui.RoleTextField, "Hour")
	if h2.Bounds.W <= h1.Bounds.W {
		t.Fatalf("scaled hour field width %.1f did not grow past default %.1f", h2.Bounds.W, h1.Bounds.W)
	}
	down := buttonInfo(t, tt2, ui.RoleButton, "Decrease minute")
	if down.Bounds.X+down.Bounds.W > 640 {
		t.Fatalf("scaled minute stepper overflowed wide canvas: %+v", down.Bounds)
	}
	if allocs := testing.AllocsPerRun(20, func() {
		_ = dialLabels(timeSegmentHour, false)
		_ = dialLabels(timeSegmentHour, true)
		_ = dialLabels(timeSegmentMinute, false)
	}); allocs != 0 {
		t.Fatalf("dialLabels allocated %.0f times per run, want 0", allocs)
	}
}

// TestDatePickerModalInsideAlertDialogRestoresFocusAndIsTransactional checks
// that a DatePickerModal opened from inside an AlertDialog restores focus to
// its exact trigger (not an earlier sibling in the AlertDialog) on Escape,
// Cancel, and OK, and commits edits only on OK.
func TestDatePickerModalInsideAlertDialogRestoresFocusAndIsTransactional(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	openDialog := false
	openPicker := false
	value := pickerDay(2026, time.November, 12)
	changes := 0

	tt := ui.NewTester(func(c *ui.Context) {
		ThemeScope(c, th, func() {
			ui.Box(c).Size(640, 620).Background(canvasRGB).Padding(20).Children(func() {
				if Button(c, "Open dialog").Clicked() {
					openDialog = true
				}
				AlertDialog(c, &openDialog, AlertDialogOptions{
					Headline: "Schedule",
					Content: func() ui.Element {
						return ui.Column(c).Gap(8).Children(func() {
							Button(c, "Earlier sibling")
							if Button(c, "Pick date").Clicked() {
								openPicker = true
							}
							DatePicker(c, &value, DatePickerOptions{
								Mode:      DatePickerModal,
								Open:      &openPicker,
								Label:     "Due date",
								OnChanged: func() { changes++ },
							})
						})
					},
					Dismiss: &DialogAction{Label: "Close"},
				})
			})
		})
	}, 640, 620)

	tt.Click("Open dialog")
	tt.Advance(400 * time.Millisecond)
	tt.Click("Pick date")
	tt.Advance(500 * time.Millisecond)

	if _, ok := tt.AccessByRole(ui.RoleDialog, "Due date"); !ok {
		t.Fatal("nested DatePickerModal did not open inside AlertDialog")
	}
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 20)))
	tt.Advance(300 * time.Millisecond)
	if value.Day() != 12 || changes != 0 {
		t.Fatalf("draft edit leaked before OK: day=%d changes=%d", value.Day(), changes)
	}

	// Escape must close the nested picker, discard the draft, keep AlertDialog
	// open, and immediately restore focus to "Pick date" (not "Earlier sibling").
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if openPicker {
		t.Fatal("Escape did not close nested DatePickerModal")
	}
	if !openDialog {
		t.Fatal("Escape closed outer AlertDialog alongside nested DatePickerModal")
	}
	if value.Day() != 12 || changes != 0 {
		t.Fatalf("Escape did not discard draft: day=%d changes=%d", value.Day(), changes)
	}
	if !tt.Focused("Pick date") {
		var focusNames []string
		for _, a := range tt.AccessAll() {
			if a.Focused {
				focusNames = append(focusNames, a.Name)
			}
		}
		t.Fatalf("nested modal Escape did not restore focus to trigger inside AlertDialog (focused=%v, after=%v)", focusNames, tt.FramePending())
	}
	tt.Advance(time.Second)
	if !tt.Focused("Pick date") {
		t.Fatal("focus drifted from trigger after nested modal exit animation settled")
	}
	if tt.FramePending() {
		t.Fatal("nested modal exit left pending frames")
	}

	// Cancel also discards the draft and restores trigger focus.
	tt.Click("Pick date")
	tt.Advance(500 * time.Millisecond)
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 22)))
	tt.Click("Cancel")
	tt.Advance(time.Second)
	if value.Day() != 12 || changes != 0 {
		t.Fatalf("Cancel did not discard draft: day=%d changes=%d", value.Day(), changes)
	}
	if !tt.Focused("Pick date") {
		t.Fatal("Cancel did not restore focus to trigger inside AlertDialog")
	}

	// OK commits the draft once and fires OnChanged.
	tt.Click("Pick date")
	tt.Advance(500 * time.Millisecond)
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 25)))
	tt.Click("OK")
	tt.Advance(time.Second)
	if value.Day() != 25 || changes != 1 {
		t.Fatalf("OK commit got day=%d changes=%d, want day=25 changes=1", value.Day(), changes)
	}
	if !tt.Focused("Pick date") {
		t.Fatal("OK did not restore focus to trigger inside AlertDialog")
	}
}

// TestDateRangePickerModalTransactional checks DateRangePicker in modal mode
// isolates edits in a draft until OK and discards them on Cancel.
func TestDateRangePickerModalTransactional(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	open := false
	start := pickerDay(2026, time.November, 10)
	end := pickerDay(2026, time.November, 14)
	changes := 0
	tt := pickerCanvas(th, func(c *ui.Context) {
		ui.Column(c).Children(func() {
			if Button(c, "Pick range").Clicked() {
				open = true
			}
			DateRangePicker(c, &start, &end, DatePickerOptions{
				Mode:      DatePickerModal,
				Open:      &open,
				Label:     "Trip dates",
				OnChanged: func() { changes++ },
			})
		})
	})

	tt.Click("Pick range")
	tt.Frame()
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 18)))
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 21)))
	tt.Frame()
	if start.Day() != 10 || end.Day() != 14 || changes != 0 {
		t.Fatalf("range draft leaked before OK: %d..%d changes=%d", start.Day(), end.Day(), changes)
	}
	tt.Click("Cancel")
	tt.Frame()
	if start.Day() != 10 || end.Day() != 14 || changes != 0 {
		t.Fatalf("range Cancel did not discard draft: %d..%d changes=%d", start.Day(), end.Day(), changes)
	}

	tt.Click("Pick range")
	tt.Frame()
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 18)))
	tt.Click(pickerDayLabel(pickerDay(2026, time.November, 21)))
	tt.Click("OK")
	tt.Frame()
	if start.Day() != 18 || end.Day() != 21 || changes != 1 {
		t.Fatalf("range OK got %d..%d changes=%d, want 18..21 changes=1", start.Day(), end.Day(), changes)
	}
}

// TestPickerExitTransitionGhostStaysVisible verifies TimePicker dials and
// DatePicker headers/day cells continue painting on ElementTransition exit ghosts.
func TestPickerExitTransitionGhostStaysVisible(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	exitTr := ui.ElementTransition{
		Duration: 200 * time.Millisecond,
		Exit:     &ui.Motion{Opacity: 0},
	}

	t.Run("timepicker-exit-ghost", func(t *testing.T) {
		show := true
		val := time.Date(2026, time.May, 1, 10, 30, 0, 0, time.Local)
		tt := pickerCanvas(th, func(c *ui.Context) {
			if show {
				ui.Column(c.Key("tp-exit")).Transition(exitTr).Children(func() {
					TimePicker(c, &val, TimePickerOptions{HourCycle: 12})
				})
			}
		})
		dial, _, _ := dialInfo(t, tt)
		tt.Advance(100 * time.Millisecond)

		show = false
		tt.Frame()
		tt.Advance(40 * time.Millisecond)
		midExit := tt.Image()

		tt.Advance(400 * time.Millisecond)
		if tt.FramePending() {
			t.Fatal("exited TimePicker ghost left pending frames")
		}
		settled := tt.Image()
		if diff := indicatorBoxDiff(midExit, settled, dial.Bounds); diff < 100 {
			t.Fatalf("exiting TimePicker dial ghost painted only %d differing pixels", diff)
		}
	})

	t.Run("datepicker-modal-exit-ghost", func(t *testing.T) {
		open := false
		val := pickerDay(2026, time.November, 12)
		tt := pickerCanvas(th, func(c *ui.Context) {
			if Button(c, "Pick ghost").Clicked() {
				open = true
			}
			DatePicker(c, &val, DatePickerOptions{
				Mode: DatePickerModal,
				Open: &open,
			})
		})
		tt.Click("Pick ghost")
		tt.Advance(400 * time.Millisecond)

		wedInfo, ok := tt.AccessByRole(ui.RoleColumnHeader, "Wed")
		if !ok {
			t.Fatal("weekday column header not found")
		}
		dayInfo, ok := tt.AccessByRole(ui.RoleCell, pickerDayLabel(pickerDay(2026, time.November, 15)))
		if !ok {
			t.Fatal("November 15 day cell not found")
		}

		tt.Click("Cancel")
		tt.Advance(30 * time.Millisecond)
		midExit := tt.Image()

		tt.Advance(500 * time.Millisecond)
		if tt.FramePending() {
			t.Fatal("DatePickerModal exit left pending frames")
		}
		settled := tt.Image()
		if diff := indicatorBoxDiff(midExit, settled, dayInfo.Bounds); diff < 10 {
			t.Fatalf("exiting DatePicker day cell ghost painted only %d differing pixels", diff)
		}
		if diff := indicatorBoxDiff(midExit, settled, wedInfo.Bounds); diff < 4 {
			t.Fatalf("exiting DatePicker weekday header ghost painted only %d differing pixels", diff)
		}
	})
}
