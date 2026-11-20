package md3e

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-md3e/internal/mdicon"
)

// alertCanvas builds a trigger and an alert over the container canvas.
func alertCanvas(th *Theme, open *bool, confirmed, dismissed *int, dismissOnScrim bool) *ui.Tester {
	return containerCanvas(th, func(c *ui.Context) {
		if Button(c, "Open alert").Clicked() {
			*open = true
		}
		AlertDialog(c, open, AlertDialogOptions{
			Icon:           mdicon.Edit.Draw,
			Headline:       "Delete note",
			SupportingText: "This cannot be undone.",
			Content: func() ui.Element {
				return ui.Text(c, "A long body that scrolls.")
			},
			DismissOnScrim: dismissOnScrim,
			Dismiss: &DialogAction{
				Label:   "Cancel",
				OnClick: func() { *dismissed++ },
			},
			Confirm: &DialogAction{
				Label:   "Delete",
				OnClick: func() { *confirmed++ },
			},
			Label: "Delete note",
		})
	})
}

// TestAlertDialogFocusScrimAndActions checks the alert takes the focus and
// makes the window behind inert, a scrim press does nothing by default,
// Escape cancels and gives the focus back, and Confirm runs its action.
func TestAlertDialogFocusScrimAndActions(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := false
	confirmed, dismissed := 0, 0
	tt := alertCanvas(th, &open, &confirmed, &dismissed, false)
	tt.Click("Open alert")
	tt.Advance(500 * time.Millisecond)
	info, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete note")
	if !ok {
		t.Fatal("the alert is not an accessible alert dialog")
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, "Open alert"); ok {
		t.Fatal("the window behind the alert is not inert")
	}
	if !tt.HasText("This cannot be undone.") {
		t.Fatal("the supporting text does not show")
	}

	// A scrim press outside the panel does nothing unless DismissOnScrim.
	tt.ClickAt(10, 10)
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete note"); !ok {
		t.Fatal("a scrim press closed a non-dismissible alert")
	}

	// Tab stays inside the alert.
	tt.Key(0, ui.KeyTab)
	tt.Key(0, ui.KeyTab)
	if tt.Focused("Open alert") {
		t.Fatal("Tab reached the window behind the alert")
	}

	tt.Key(0, ui.KeyEscape)
	tt.Advance(time.Second)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete note"); ok {
		t.Fatal("Escape does not close the alert")
	}
	if dismissed != 1 {
		t.Fatalf("Escape ran Dismiss %d times, want 1", dismissed)
	}
	if !tt.Focused("Open alert") {
		t.Fatal("closing does not give the focus back to the trigger")
	}
	if tt.FramePending() {
		t.Fatal("the closed alert keeps asking for frames")
	}

	// Reopen and confirm.
	tt.Click("Open alert")
	tt.Advance(500 * time.Millisecond)
	tt.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete note"); !ok {
		t.Fatal("the alert did not reopen")
	}
	confirm := buttonInfo(t, tt, ui.RoleButton, "Delete")
	tt.ClickAt(confirm.Bounds.X+confirm.Bounds.W/2, confirm.Bounds.Y+confirm.Bounds.H/2)
	tt.Advance(time.Second)
	if confirmed != 1 {
		t.Fatalf("Confirm ran %d times, want 1", confirmed)
	}
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete note"); ok {
		t.Fatal("Confirm does not close the alert")
	}
	if !tt.Focused("Open alert") {
		t.Fatal("Confirm does not give the focus back")
	}
	if tt.FramePending() {
		t.Fatal("the confirmed alert keeps asking for frames")
	}
}

// TestAlertDialogDismissOnScrimAndReduceMotion checks a dismissible scrim and
// the reduced-motion exit, which leaves no frames behind.
func TestAlertDialogDismissOnScrimAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	open := false
	confirmed, dismissed := 0, 0
	tt := alertCanvas(th, &open, &confirmed, &dismissed, true)
	tt.Click("Open alert")
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete note"); !ok {
		t.Fatal("the reduced-motion alert does not open")
	}
	tt.ClickAt(10, 10)
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleAlertDialog, "Delete note"); ok {
		t.Fatal("a dismissible scrim press does not close the alert")
	}
	if dismissed != 1 {
		t.Fatalf("the scrim ran Dismiss %d times, want 1", dismissed)
	}
	if !tt.Focused("Open alert") {
		t.Fatal("the reduced-motion alert does not restore the focus")
	}
	if tt.FramePending() {
		t.Fatal("the reduced-motion alert keeps asking for frames")
	}
}

// sheetContent builds 20 text rows so the sheet's body can scroll.
func sheetContent(c *ui.Context) ui.Element {
	col := ui.Column(c).Gap(4)
	col.Children(func() {
		for i := 0; i < 20; i++ {
			ui.Text(c, "Sheet row "+string(rune('0'+i/10))+string(rune('0'+i%10)))
		}
	})
	return col
}

// sheetCanvas builds a trigger and a modal sheet over the container canvas.
func sheetCanvas(th *Theme, open *bool, detent *int) *ui.Tester {
	return containerCanvas(th, func(c *ui.Context) {
		if Button(c, "Share").Clicked() {
			*open = true
		}
		BottomSheet(c, open, BottomSheetOptions{
			Modal:   true,
			Detents: []float32{0.5, 1},
			Detent:  detent,
			Label:   "Share sheet",
			Content: func() ui.Element { return sheetContent(c) },
		})
	})
}

// TestModalBottomSheetScrollDetentsAndDrag checks the sheet's dialog role, that
// its body scrolls without the pull closing it, that a detent change grows it,
// a short pull springs back, and a long pull closes it and restores the focus.
func TestModalBottomSheetScrollDetentsAndDrag(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := false
	detent := 0
	tt := sheetCanvas(th, &open, &detent)
	tt.Click("Share")
	tt.Advance(500 * time.Millisecond)
	info, ok := tt.AccessByRole(ui.RoleDialog, "Share sheet")
	if !ok {
		t.Fatal("the modal sheet is not an accessible dialog")
	}

	// The body scrolls under the wheel, and scrolling never closes the sheet.
	first, ok := tt.Find("Sheet row 00")
	if !ok {
		t.Fatal("the first row does not show")
	}
	tt.Wheel(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2, 0, 160)
	tt.Advance(500 * time.Millisecond)
	after, ok := tt.Find("Sheet row 00")
	if !ok {
		t.Fatal("the first row vanished from the scroll area")
	}
	if after.Y >= first.Y {
		t.Fatalf("the wheel did not scroll the body: row at %.1f then %.1f", first.Y, after.Y)
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Share sheet"); !ok {
		t.Fatal("scrolling the body closed the sheet")
	}

	// A taller detent grows the sheet upward.
	detent = 1
	tt.Frame()
	tt.Advance(time.Second)
	grown, _ := tt.AccessByRole(ui.RoleDialog, "Share sheet")
	if grown.Bounds.Y >= info.Bounds.Y-50 || grown.Bounds.H <= info.Bounds.H+50 {
		t.Fatalf("the full detent did not grow the sheet: %v then %v", info.Bounds, grown.Bounds)
	}

	handle := func(pull float32) {
		t.Helper()
		box, _ := tt.AccessByRole(ui.RoleDialog, "Share sheet")
		x, y := box.Bounds.X+box.Bounds.W/2, box.Bounds.Y+16
		tt.Press(x, y)
		tt.Move(x, y+pull)
		tt.Release(x, y+pull)
		tt.Advance(time.Second)
	}
	handle(40)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Share sheet"); !ok {
		t.Fatal("a short pull closed the sheet")
	}
	handle(180)
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Share sheet"); ok {
		t.Fatal("a long pull does not close the sheet")
	}
	if !tt.Focused("Share") {
		t.Fatal("closing the sheet does not give the focus back")
	}
	if tt.FramePending() {
		t.Fatal("the closed sheet keeps asking for frames")
	}
}

// TestModalBottomSheetEscapeAndReduceMotion checks Escape closes the reduced
// motion sheet at once and leaves no frames behind.
func TestModalBottomSheetEscapeAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	open := false
	detent := 0
	tt := sheetCanvas(th, &open, &detent)
	tt.Click("Share")
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Share sheet"); !ok {
		t.Fatal("the reduced-motion sheet does not open")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleDialog, "Share sheet"); ok {
		t.Fatal("Escape does not close the reduced-motion sheet")
	}
	if !tt.Focused("Share") {
		t.Fatal("the reduced-motion sheet does not restore the focus")
	}
	if tt.FramePending() {
		t.Fatal("the reduced-motion sheet keeps asking for frames")
	}
}

// TestStandardBottomSheetInline checks a standard sheet lays out at its
// parent's bottom and leaves with the open state.
func TestStandardBottomSheetInline(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := true
	tt := containerCanvas(th, func(c *ui.Context) {
		ui.Column(c).Width(320).Height(352).Justify(ui.End).Children(func() {
			BottomSheet(c, &open, BottomSheetOptions{
				Height:  200,
				Label:   "Inline sheet",
				Content: func() ui.Element { return sheetContent(c) },
			})
		})
	})
	tt.Frame()
	info, ok := tt.AccessByRole(ui.RoleRegion, "Inline sheet")
	if !ok {
		t.Fatal("the standard sheet is not an accessible region")
	}
	if absF32(info.Bounds.H-200) > 0.5 {
		t.Fatalf("the standard sheet is %.1f high, want 200", info.Bounds.H)
	}
	if info.Bounds.Y+info.Bounds.H < 375 || info.Bounds.Y+info.Bounds.H > 377 {
		t.Fatalf("the standard sheet rests at %v, not at the bottom of the canvas", info.Bounds)
	}
	open = false
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleRegion, "Inline sheet"); ok {
		t.Fatal("a closed standard sheet keeps its region")
	}
}

// menuItems builds the items of the menu tests around the mutable app state.
func menuItems(rename, pin, radio, advanced *int, pinned *bool, sortBy *string) []MenuItem {
	return []MenuItem{
		{Label: "Rename", Shortcut: "F2", OnClick: func() { *rename++ }},
		{Label: "Pin", Checkable: true, Checked: *pinned, OnClick: func() {
			*pinned = !*pinned
			*pin++
		}},
		{Label: "Name", Radio: true, Checked: *sortBy == "name", OnClick: func() {
			*sortBy = "name"
			*radio++
		}},
		{Separator: true},
		{Label: "More", Submenu: []MenuItem{
			{Label: "Advanced", OnClick: func() { *advanced++ }},
		}},
	}
}

// TestDropdownMenuKeysChecksAndSubmenu checks the trigger opens the menu, the
// keys choose, check and radio items report their state, a submenu opens with
// Right and closes on a choice, and the focus returns to the trigger.
func TestDropdownMenuKeysChecksAndSubmenu(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	rename, pin, radio, advanced := 0, 0, 0, 0
	pinned := false
	sortBy := "date"
	tt := containerCanvas(th, func(c *ui.Context) {
		DropdownMenu(c, "Actions", menuItems(&rename, &pin, &radio, &advanced, &pinned, &sortBy),
			MenuOptions{Label: "Actions menu"})
	})
	openMenu := func() {
		t.Helper()
		tt.Click("Actions")
		tt.Advance(300 * time.Millisecond)
		if _, ok := tt.AccessByRole(ui.RoleMenu, "Actions menu"); !ok {
			t.Fatal("the drop-down menu does not open")
		}
	}
	openMenu()
	if !tt.HasText("Rename") || !tt.HasText("More") {
		t.Fatal("the menu items do not show")
	}

	// Down highlights the first item, Enter chooses it.
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	if rename != 1 {
		t.Fatalf("Enter ran the first item %d times, want 1", rename)
	}
	if _, ok := tt.AccessByRole(ui.RoleMenu, "Actions menu"); ok {
		t.Fatal("choosing an item does not close the menu")
	}
	if !tt.Focused("Actions") {
		t.Fatal("closing the menu does not give the focus back to the trigger")
	}

	// A check item toggles its state and closes.
	openMenu()
	tt.Click("Pin")
	tt.Advance(300 * time.Millisecond)
	if !pinned || pin != 1 {
		t.Fatalf("the check item toggled to %v (%d), want true once", pinned, pin)
	}
	openMenu()
	if info, ok := tt.AccessByRole(ui.RoleMenuItemCheckBox, "Pin"); !ok || !info.Checked {
		t.Fatal("the chosen check item does not read as checked")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Advance(300 * time.Millisecond)

	// The radio item is chosen with the keys.
	openMenu()
	if _, ok := tt.AccessByRole(ui.RoleMenuItemRadio, "Name"); !ok {
		t.Fatal("the radio item does not read as one")
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	if sortBy != "name" || radio != 1 {
		t.Fatalf("the radio item chose %q (%d), want name once", sortBy, radio)
	}

	// Right opens the submenu of the fourth row, and its item is chosen.
	openMenu()
	for range 4 {
		tt.Key(0, ui.KeyDown)
	}
	tt.Key(0, ui.KeyRight)
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("Advanced") {
		t.Fatal("Right does not open the submenu")
	}
	tt.Key(0, ui.KeyEnter)
	tt.Advance(500 * time.Millisecond)
	if advanced != 1 {
		t.Fatalf("the submenu item ran %d times, want 1", advanced)
	}
	if tt.FramePending() {
		t.Fatal("the closed menu keeps asking for frames")
	}
}

// TestDropdownMenuDisabledAndReduceMotion checks a disabled trigger, alone or
// under a disabled ancestor, opens nothing, and a reduced-motion menu closes
// with no frames left.
func TestDropdownMenuDisabledAndReduceMotion(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor, ThemeOptions{ReduceMotion: true})
	rename, pin, radio, advanced := 0, 0, 0, 0
	pinned := false
	sortBy := ""
	build := func(disabled bool, ancestor bool) func(c *ui.Context) {
		return func(c *ui.Context) {
			items := menuItems(&rename, &pin, &radio, &advanced, &pinned, &sortBy)
			if ancestor {
				ui.Box(c).Disabled(true).Children(func() {
					DropdownMenu(c, "Actions", items, MenuOptions{Label: "Actions menu"})
				})
				return
			}
			DropdownMenu(c, "Actions", items, MenuOptions{Label: "Actions menu", Disabled: disabled})
		}
	}
	for _, tc := range []struct {
		name     string
		disabled bool
		ancestor bool
	}{{"option", true, false}, {"ancestor", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			tt := containerCanvas(th, build(tc.disabled, tc.ancestor))
			tt.Click("Actions")
			tt.Frame()
			if _, ok := tt.AccessByRole(ui.RoleMenu, "Actions menu"); ok {
				t.Fatal("a disabled trigger opened its menu")
			}
			if tt.FramePending() {
				t.Fatal("the disabled menu keeps asking for frames")
			}
		})
	}

	tt := containerCanvas(th, build(false, false))
	tt.Click("Actions")
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleMenu, "Actions menu"); !ok {
		t.Fatal("the reduced-motion menu does not open")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if _, ok := tt.AccessByRole(ui.RoleMenu, "Actions menu"); ok {
		t.Fatal("Escape does not close the reduced-motion menu")
	}
	if tt.FramePending() {
		t.Fatal("the reduced-motion menu keeps asking for frames")
	}
}

// TestMenuButtonOpensMenu checks the icon menu button is named, opens its menu
// and runs the chosen action.
func TestMenuButtonOpensMenu(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	rename := 0
	tt := containerCanvas(th, func(c *ui.Context) {
		MenuButton(c, "More actions", mdicon.MoreVert.Draw, []MenuItem{
			{Label: "Rename", OnClick: func() { rename++ }},
		}, MenuOptions{Label: "More menu"})
	})
	tt.Click("More actions")
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleMenu, "More menu"); !ok {
		t.Fatal("the icon menu button does not open its menu")
	}
	tt.Click("Rename")
	tt.Advance(300 * time.Millisecond)
	if rename != 1 {
		t.Fatalf("the menu button ran Rename %d times, want 1", rename)
	}
	if !tt.Focused("More actions") {
		t.Fatal("the menu button does not take the focus back")
	}
	if tt.FramePending() {
		t.Fatal("the closed menu keeps asking for frames")
	}
}

// TestContextMenuRightClick checks a right click opens the context menu and a
// choice runs its action.
func TestContextMenuRightClick(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	rename, pin, radio, advanced := 0, 0, 0, 0
	pinned := false
	sortBy := ""
	tt := containerCanvas(th, func(c *ui.Context) {
		row := Button(c, "Note", ButtonOptions{Variant: ButtonOutlined})
		ContextMenu(c, row, menuItems(&rename, &pin, &radio, &advanced, &pinned, &sortBy),
			MenuOptions{Label: "Note menu"})
	})
	tt.Advance(300 * time.Millisecond)
	if err := tt.RightClick("Note"); err != nil {
		t.Fatalf("right-clicking the note: %v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if _, ok := tt.AccessByRole(ui.RoleMenu, "Note menu"); !ok {
		t.Fatal("the right click does not open the context menu")
	}
	tt.Click("Rename")
	tt.Advance(300 * time.Millisecond)
	if rename != 1 {
		t.Fatalf("the context menu ran Rename %d times, want 1", rename)
	}
	if _, ok := tt.AccessByRole(ui.RoleMenu, "Note menu"); ok {
		t.Fatal("choosing a context menu item does not close it")
	}
	if tt.FramePending() {
		t.Fatal("the closed context menu keeps asking for frames")
	}
}

// TestFullScreenDialogSaveCloseAndEscape checks the full-screen dialog's top
// bar actions, Escape, its dialog role and focus restoration.
func TestFullScreenDialogSaveCloseAndEscape(t *testing.T) {
	th := ThemeFromSeed(DefaultSeedColor)
	open := false
	saved, closed := 0, 0
	tt := containerCanvas(th, func(c *ui.Context) {
		if Button(c, "Compose").Clicked() {
			open = true
		}
		FullScreenDialog(c, &open, FullScreenDialogOptions{
			Title:     "New note",
			SaveLabel: "Save",
			OnSave:    func() { saved++ },
			OnClose:   func() { closed++ },
			Content: func() ui.Element {
				return ui.Text(c, "Write something.")
			},
		})
	})
	openFull := func() {
		t.Helper()
		tt.Click("Compose")
		tt.Advance(300 * time.Millisecond)
		if _, ok := tt.AccessByRole(ui.RoleDialog, "New note"); !ok {
			t.Fatal("the full-screen dialog is not an accessible dialog")
		}
	}
	openFull()
	tt.Click("Save")
	tt.Advance(300 * time.Millisecond)
	if saved != 1 {
		t.Fatalf("Save ran %d times, want 1", saved)
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "New note"); ok {
		t.Fatal("Save does not close the full-screen dialog")
	}
	if !tt.Focused("Compose") {
		t.Fatal("Save does not give the focus back")
	}

	openFull()
	tt.Key(0, ui.KeyEscape)
	tt.Advance(300 * time.Millisecond)
	if closed != 1 {
		t.Fatalf("Escape ran OnClose %d times, want 1", closed)
	}
	if _, ok := tt.AccessByRole(ui.RoleDialog, "New note"); ok {
		t.Fatal("Escape does not close the full-screen dialog")
	}

	openFull()
	tt.Click("Close")
	tt.Advance(300 * time.Millisecond)
	if closed != 2 {
		t.Fatalf("the close button ran OnClose %d times, want 2", closed)
	}
	if tt.FramePending() {
		t.Fatal("the closed full-screen dialog keeps asking for frames")
	}
}
